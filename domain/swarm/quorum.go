package swarm

import (
	"math"
	"swarmsim/logger"
)

// QuorumState manages the quorum sensing system.
// Bots measure local neighbor density and shared "vote" signals.
// When enough nearby bots agree (quorum threshold), collective
// decisions trigger — e.g., cluster migration, role switching,
// coordinated pickup, or alarm propagation.
type QuorumState struct {
	Threshold    float64 // fraction of neighbors that must agree (default 0.6)
	VoteRange    float64 // sensing range for votes (default 80)
	DecayRate    float64 // vote confidence decay per tick (default 0.02)
	NumProposals int     // number of distinct proposals (default 4)

	// Per-bot voting state
	Votes      []BotVote
	Decisions  []QuorumDecision // active collective decisions this tick
	TotalVotes int              // total votes cast this tick

	// Stats
	DecisionCount int     // total decisions triggered
	AvgQuorum     float64 // rolling average quorum size
}

// BotVote is a bot's current vote/proposal state.
type BotVote struct {
	Proposal   int     // which proposal this bot supports (0..NumProposals-1)
	Confidence float64 // 0.0-1.0 how strongly the bot supports its proposal
	QuorumMet  bool    // whether quorum was reached around this bot
	LocalCount int     // number of nearby bots
	LocalAgree int     // number of nearby bots with same proposal
}

// QuorumDecision is a triggered collective decision.
type QuorumDecision struct {
	Proposal    int
	CenterX     float64
	CenterY     float64
	Participants int
	Strength    float64 // average confidence of participants
}

// QuorumProposal constants
const (
	ProposalMigrate  = 0 // group should move together
	ProposalCluster  = 1 // form tight cluster
	ProposalDisperse = 2 // spread out
	ProposalAlarm    = 3 // danger signal
)

// ProposalName returns the display name of a proposal.
func ProposalName(p int) string {
	switch p {
	case ProposalMigrate:
		return "Migration"
	case ProposalCluster:
		return "Cluster"
	case ProposalDisperse:
		return "Ausbreitung"
	case ProposalAlarm:
		return "Alarm"
	default:
		return "?"
	}
}

// InitQuorum sets up the quorum sensing system.
func InitQuorum(ss *SwarmState) {
	n := len(ss.Bots)
	qs := &QuorumState{
		Threshold:    0.6,
		VoteRange:    80,
		DecayRate:    0.02,
		NumProposals: 4,
		Votes:        make([]BotVote, n),
	}

	// Initial random votes based on bot state
	for i := range ss.Bots {
		qs.Votes[i].Proposal = proposalFromState(&ss.Bots[i])
		qs.Votes[i].Confidence = 0.5
	}

	ss.Quorum = qs
	logger.Info("QUORUM", "Initialisiert: %d Bots, Schwelle=%.0f%%, Reichweite=%.0f",
		n, qs.Threshold*100, qs.VoteRange)
}

// ClearQuorum disables the quorum sensing system.
func ClearQuorum(ss *SwarmState) {
	ss.Quorum = nil
	ss.QuorumOn = false
}

// proposalFromState derives a proposal from a bot's current situation.
func proposalFromState(bot *SwarmBot) int {
	if bot.CarryingPkg >= 0 {
		return ProposalMigrate // carrying → wants group movement toward dropoff
	}
	if bot.NearestPickupDist < 60 {
		return ProposalCluster // near pickup → cluster to help
	}
	if bot.NeighborCount > 8 {
		return ProposalDisperse // too crowded → spread out
	}
	return ProposalMigrate // default: explore together
}

// TickQuorum runs one tick of the quorum sensing system.
func TickQuorum(ss *SwarmState) {
	qs := ss.Quorum
	if qs == nil {
		return
	}

	n := len(ss.Bots)
	if len(qs.Votes) != n {
		qs.Votes = make([]BotVote, n)
	}

	// Phase 1: Each bot updates its proposal based on state
	for i := range ss.Bots {
		// Slowly shift proposal based on current state
		stateProposal := proposalFromState(&ss.Bots[i])
		if stateProposal != qs.Votes[i].Proposal {
			qs.Votes[i].Confidence -= 0.05
			if qs.Votes[i].Confidence < 0.1 {
				qs.Votes[i].Proposal = stateProposal
				qs.Votes[i].Confidence = 0.3
			}
		} else {
			qs.Votes[i].Confidence += 0.01
			if qs.Votes[i].Confidence > 1.0 {
				qs.Votes[i].Confidence = 1.0
			}
		}
	}

	// Phase 2: Count local votes and check quorum
	rangeSq := qs.VoteRange * qs.VoteRange
	qs.Decisions = nil
	qs.TotalVotes = 0
	totalQuorum := 0.0

	// Use spatial hash if available. Ensure hash is populated with current
	// bot positions (main sim loop rebuilds it each tick, but this is
	// cheap O(n) insurance for standalone/test invocations).
	useSpatial := ss.Hash != nil
	if useSpatial {
		ss.Hash.Clear()
		for i := range ss.Bots {
			ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
		}
	}

	for i := range ss.Bots {
		localCount := 0
		localAgree := 0
		totalConf := 0.0

		if useSpatial {
			// O(n·k): query only nearby bots via spatial hash
			nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, qs.VoteRange)
			for _, j := range nearIDs {
				if j == i || j >= n {
					continue
				}
				dx := ss.Bots[i].X - ss.Bots[j].X
				dy := ss.Bots[i].Y - ss.Bots[j].Y
				if dx*dx+dy*dy > rangeSq {
					continue
				}
				localCount++
				if qs.Votes[j].Proposal == qs.Votes[i].Proposal {
					localAgree++
					totalConf += qs.Votes[j].Confidence
				}
			}
		} else {
			// Fallback: brute-force O(n²)
			for j := range ss.Bots {
				if i == j {
					continue
				}
				dx := ss.Bots[i].X - ss.Bots[j].X
				dy := ss.Bots[i].Y - ss.Bots[j].Y
				if dx*dx+dy*dy > rangeSq {
					continue
				}
				localCount++
				if qs.Votes[j].Proposal == qs.Votes[i].Proposal {
					localAgree++
					totalConf += qs.Votes[j].Confidence
				}
			}
		}

		qs.Votes[i].LocalCount = localCount
		qs.Votes[i].LocalAgree = localAgree

		// Check quorum
		quorumMet := false
		if localCount >= 3 {
			ratio := float64(localAgree) / float64(localCount)
			if ratio >= qs.Threshold {
				quorumMet = true
				avgConf := totalConf / float64(localAgree)
				totalQuorum += float64(localAgree)

				// Only the bot with highest confidence in cluster triggers decision
				isLeader := true
				if useSpatial {
					nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, qs.VoteRange)
					for _, j := range nearIDs {
						if j == i || j >= n {
							continue
						}
						dx := ss.Bots[i].X - ss.Bots[j].X
						dy := ss.Bots[i].Y - ss.Bots[j].Y
						if dx*dx+dy*dy > rangeSq {
							continue
						}
						if qs.Votes[j].Proposal == qs.Votes[i].Proposal &&
							qs.Votes[j].Confidence > qs.Votes[i].Confidence {
							isLeader = false
							break
						}
					}
				} else {
					for j := range ss.Bots {
						if j == i {
							continue
						}
						dx := ss.Bots[i].X - ss.Bots[j].X
						dy := ss.Bots[i].Y - ss.Bots[j].Y
						if dx*dx+dy*dy > rangeSq {
							continue
						}
						if qs.Votes[j].Proposal == qs.Votes[i].Proposal &&
							qs.Votes[j].Confidence > qs.Votes[i].Confidence {
							isLeader = false
							break
						}
					}
				}

				if isLeader {
					qs.Decisions = append(qs.Decisions, QuorumDecision{
						Proposal:     qs.Votes[i].Proposal,
						CenterX:      ss.Bots[i].X,
						CenterY:      ss.Bots[i].Y,
						Participants: localAgree + 1,
						Strength:     avgConf,
					})
				}
			}
		}
		qs.Votes[i].QuorumMet = quorumMet
		qs.TotalVotes++
	}

	// Phase 3: Apply decisions — influence nearby bots
	for _, dec := range qs.Decisions {
		qs.DecisionCount++
		applyQuorumDecision(ss, qs, &dec)
	}

	if n > 0 {
		qs.AvgQuorum = totalQuorum / float64(n)
	}

	// Phase 4: Social influence — bots adopt neighbors' proposals
	if ss.Tick%5 == 0 {
		socialInfluence(ss, qs)
	}
}

// applyQuorumDecision affects nearby bots based on the collective decision.
func applyQuorumDecision(ss *SwarmState, qs *QuorumState, dec *QuorumDecision) {
	rangeSq := qs.VoteRange * qs.VoteRange

	for i := range ss.Bots {
		dx := ss.Bots[i].X - dec.CenterX
		dy := ss.Bots[i].Y - dec.CenterY
		distSq := dx*dx + dy*dy
		if distSq > rangeSq {
			continue
		}

		influence := dec.Strength * 0.3
		switch dec.Proposal {
		case ProposalMigrate:
			// Boost speed and align toward center of decision
			ss.Bots[i].Speed = math.Min(ss.Bots[i].Speed+influence*SwarmBotSpeed*0.2, SwarmBotSpeed)

		case ProposalCluster:
			// Move toward decision center
			dist := math.Sqrt(distSq)
			if dist > 5 {
				angle := math.Atan2(-dy, -dx)
				ss.Bots[i].Angle += (angle - ss.Bots[i].Angle) * influence * 0.1
			}

		case ProposalDisperse:
			// Move away from center
			dist := math.Sqrt(distSq)
			if dist > 1 {
				angle := math.Atan2(dy, dx)
				ss.Bots[i].Angle += (angle - ss.Bots[i].Angle) * influence * 0.1
			}

		case ProposalAlarm:
			// Flash red LED
			ss.Bots[i].LEDColor = [3]uint8{255, uint8(50 * (1 - influence)), 0}
		}
	}
}

// socialInfluence makes bots adopt nearby majority proposals.
func socialInfluence(ss *SwarmState, qs *QuorumState) {
	rangeSq := qs.VoteRange * qs.VoteRange
	n := len(ss.Bots)
	newProposals := make([]int, n)
	useSpatial := ss.Hash != nil

	for i := range ss.Bots {
		counts := make([]int, qs.NumProposals)

		if useSpatial {
			// O(n·k): query only nearby bots via spatial hash
			nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, qs.VoteRange)
			for _, j := range nearIDs {
				if j == i || j >= n {
					continue
				}
				dx := ss.Bots[i].X - ss.Bots[j].X
				dy := ss.Bots[i].Y - ss.Bots[j].Y
				if dx*dx+dy*dy > rangeSq {
					continue
				}
				if qs.Votes[j].Proposal < qs.NumProposals {
					counts[qs.Votes[j].Proposal]++
				}
			}
		} else {
			// Fallback: brute-force O(n²)
			for j := range ss.Bots {
				if i == j {
					continue
				}
				dx := ss.Bots[i].X - ss.Bots[j].X
				dy := ss.Bots[i].Y - ss.Bots[j].Y
				if dx*dx+dy*dy > rangeSq {
					continue
				}
				if qs.Votes[j].Proposal < qs.NumProposals {
					counts[qs.Votes[j].Proposal]++
				}
			}
		}

		// Find majority
		bestP := qs.Votes[i].Proposal
		bestC := 0
		for p, c := range counts {
			if c > bestC {
				bestC = c
				bestP = p
			}
		}
		newProposals[i] = bestP
	}

	// Apply with some inertia
	for i := range qs.Votes {
		if newProposals[i] != qs.Votes[i].Proposal {
			if ss.Rng.Float64() < 0.3 { // 30% chance to adopt neighbor majority
				qs.Votes[i].Proposal = newProposals[i]
				qs.Votes[i].Confidence = 0.4
			}
		}
	}
}

// QuorumDecisionCount returns number of active decisions.
func QuorumDecisionCount(qs *QuorumState) int {
	if qs == nil {
		return 0
	}
	return len(qs.Decisions)
}

// QuorumAvgAgreement returns the average local agreement ratio.
func QuorumAvgAgreement(qs *QuorumState) float64 {
	if qs == nil || len(qs.Votes) == 0 {
		return 0
	}
	total := 0.0
	count := 0
	for _, v := range qs.Votes {
		if v.LocalCount > 0 {
			total += float64(v.LocalAgree) / float64(v.LocalCount)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}
