package swarm

import (
	"swarmsim/logger"
)

// DemocracyState manages ranked-choice voting on swarm strategies.
// Each bot ranks 3 strategy proposals. Votes are tallied using instant-
// runoff: the weakest option is eliminated and its votes redistributed
// until a winner emerges. Emergent "parties" form around strategies.
type DemocracyState struct {
	Ballots   []RankedBallot // per-bot ranked preferences
	Proposals []StrategyProposal

	VoteInterval int // ticks between elections (default 100)
	LastElection int // tick of last election

	// Current elected strategy
	WinningStrategy int    // index of winning proposal
	WinnerName      string // name of winning strategy
	WinnerVotes     int    // final vote count

	// Party affiliations
	PartyAffil []int // per-bot: which proposal they most prefer

	// Stats
	TotalElections int
	RoundsLastElec int // rounds of elimination in last election
	Turnout        float64
	ConsensusLevel float64 // how strongly bots agree (0-1)
}

// RankedBallot holds one bot's ranked preferences.
type RankedBallot struct {
	Rankings [5]int // ranked list of proposal indices (most preferred first)
	Confidence float64 // how strongly they feel about their #1
}

// StrategyProposal is a swarm-wide strategy option.
type StrategyProposal struct {
	Name       string
	SpeedMod   float64 // speed multiplier for adopters
	SpreadMod  float64 // how spread out bots should be
	FocusMod   float64 // focus on task vs exploration
	Wins       int     // historical win count
}

var defaultProposals = []StrategyProposal{
	{Name: "Blitz", SpeedMod: 1.4, SpreadMod: 1.2, FocusMod: 0.6},
	{Name: "Schildkroete", SpeedMod: 0.7, SpreadMod: 0.6, FocusMod: 1.3},
	{Name: "Schwarm", SpeedMod: 1.0, SpreadMod: 0.4, FocusMod: 1.0},
	{Name: "Kundschafter", SpeedMod: 1.2, SpreadMod: 1.5, FocusMod: 0.4},
	{Name: "Festung", SpeedMod: 0.5, SpreadMod: 0.3, FocusMod: 1.5},
}

// InitDemocracy sets up the ranked-choice voting system.
func InitDemocracy(ss *SwarmState) {
	n := len(ss.Bots)
	dm := &DemocracyState{
		Ballots:      make([]RankedBallot, n),
		Proposals:    make([]StrategyProposal, len(defaultProposals)),
		VoteInterval: 100,
		PartyAffil:   make([]int, n),
	}

	copy(dm.Proposals, defaultProposals)

	// Initial random preferences
	for i := 0; i < n; i++ {
		dm.Ballots[i] = generateBallot(ss, n)
		dm.PartyAffil[i] = dm.Ballots[i].Rankings[0]
	}

	ss.Democracy = dm
	logger.Info("DEMO", "Initialisiert: %d Waehler, %d Vorschlaege", n, len(dm.Proposals))
}

func generateBallot(ss *SwarmState, _ int) RankedBallot {
	b := RankedBallot{
		Confidence: 0.3 + ss.Rng.Float64()*0.7,
	}

	// Fisher-Yates shuffle for ranking
	perm := [5]int{0, 1, 2, 3, 4}
	for i := 4; i > 0; i-- {
		j := ss.Rng.Intn(i + 1)
		perm[i], perm[j] = perm[j], perm[i]
	}
	b.Rankings = perm
	return b
}

// ClearDemocracy disables the democracy system.
func ClearDemocracy(ss *SwarmState) {
	ss.Democracy = nil
	ss.DemocracyOn = false
}

// TickDemocracy runs one tick of the democratic process.
func TickDemocracy(ss *SwarmState) {
	dm := ss.Democracy
	if dm == nil {
		return
	}

	n := len(ss.Bots)
	if len(dm.Ballots) != n {
		return
	}

	// Update preferences based on experience
	if ss.Tick%20 == 0 {
		updatePreferences(ss, dm)
	}

	// Run election at interval
	if ss.Tick-dm.LastElection >= dm.VoteInterval {
		runElection(ss, dm)
		dm.LastElection = ss.Tick
	}

	// Apply winning strategy
	applyWinningStrategy(ss, dm)
}

// updatePreferences adjusts bot preferences based on current conditions.
func updatePreferences(ss *SwarmState, dm *DemocracyState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		ballot := &dm.Ballots[i]

		// Bots near resources prefer focused strategies
		if bot.NearestPickupDist < 60 {
			boostProposal(ballot, 1) // Schildkroete
			boostProposal(ballot, 4) // Festung
		}

		// Fast bots prefer speed strategies
		if bot.Speed > SwarmBotSpeed*1.1 {
			boostProposal(ballot, 0) // Blitz
			boostProposal(ballot, 3) // Kundschafter
		}

		// Bots with many neighbors prefer swarm strategy
		if bot.NeighborCount > 5 {
			boostProposal(ballot, 2) // Schwarm
		}

		// Social influence: adopt neighbor preferences slightly
		if bot.NeighborCount > 0 && ss.Rng.Float64() < 0.1 {
			neighbor := ss.Rng.Intn(len(ss.Bots))
			if neighbor != i {
				neighborPref := dm.Ballots[neighbor].Rankings[0]
				boostProposal(ballot, neighborPref)
			}
		}

		dm.PartyAffil[i] = ballot.Rankings[0]
	}
}

// boostProposal moves a proposal closer to #1 in the ranking.
func boostProposal(ballot *RankedBallot, proposalIdx int) {
	for pos := 1; pos < 5; pos++ {
		if ballot.Rankings[pos] == proposalIdx {
			// Swap with position above
			ballot.Rankings[pos], ballot.Rankings[pos-1] = ballot.Rankings[pos-1], ballot.Rankings[pos]
			return
		}
	}
}

// runElection performs instant-runoff voting.
func runElection(ss *SwarmState, dm *DemocracyState) {
	n := len(dm.Ballots)
	numProposals := len(dm.Proposals)
	eliminated := make([]bool, numProposals)
	rounds := 0

	for {
		rounds++

		// Count first-choice votes
		votes := make([]int, numProposals)
		for _, ballot := range dm.Ballots {
			// Find first non-eliminated choice
			for _, choice := range ballot.Rankings {
				if choice >= 0 && choice < numProposals && !eliminated[choice] {
					votes[choice]++
					break
				}
			}
		}

		// Check for majority
		for p, v := range votes {
			if v > n/2 {
				dm.WinningStrategy = p
				dm.WinnerName = dm.Proposals[p].Name
				dm.WinnerVotes = v
				dm.RoundsLastElec = rounds
				dm.TotalElections++
				dm.Proposals[p].Wins++

				// Consensus: winner votes / total
				dm.ConsensusLevel = float64(v) / float64(n)
				dm.Turnout = 1.0

				logger.Info("DEMO", "Wahl %d: '%s' gewinnt mit %d/%d Stimmen in %d Runden",
					dm.TotalElections, dm.WinnerName, v, n, rounds)
				return
			}
		}

		// Eliminate weakest non-eliminated proposal
		minVotes := n + 1
		minIdx := -1
		for p, v := range votes {
			if !eliminated[p] && v < minVotes {
				minVotes = v
				minIdx = p
			}
		}

		if minIdx < 0 || rounds > numProposals {
			// Fallback: pick whoever has most votes
			bestIdx := 0
			for p, v := range votes {
				if v > votes[bestIdx] {
					bestIdx = p
				}
			}
			dm.WinningStrategy = bestIdx
			dm.WinnerName = dm.Proposals[bestIdx].Name
			dm.WinnerVotes = votes[bestIdx]
			dm.RoundsLastElec = rounds
			dm.TotalElections++
			dm.ConsensusLevel = float64(votes[bestIdx]) / float64(n)
			return
		}

		eliminated[minIdx] = true
	}
}

// applyWinningStrategy applies the elected strategy.
func applyWinningStrategy(ss *SwarmState, dm *DemocracyState) {
	if dm.WinningStrategy < 0 || dm.WinningStrategy >= len(dm.Proposals) {
		return
	}

	prop := &dm.Proposals[dm.WinningStrategy]
	strength := 0.1 // gentle application

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Bots who voted for the winner comply more
		compliance := strength
		if dm.PartyAffil[i] == dm.WinningStrategy {
			compliance *= 2.0
		}

		bot.Speed *= 1.0 + (prop.SpeedMod-1.0)*compliance

		// Color by party
		switch dm.PartyAffil[i] {
		case 0: // Blitz: yellow
			bot.LEDColor = [3]uint8{200, 200, 50}
		case 1: // Schildkroete: green
			bot.LEDColor = [3]uint8{50, 200, 50}
		case 2: // Schwarm: cyan
			bot.LEDColor = [3]uint8{50, 200, 200}
		case 3: // Kundschafter: orange
			bot.LEDColor = [3]uint8{220, 150, 30}
		case 4: // Festung: purple
			bot.LEDColor = [3]uint8{180, 50, 180}
		}
	}
}

// DemocracyWinner returns the name of the winning strategy.
func DemocracyWinner(dm *DemocracyState) string {
	if dm == nil {
		return ""
	}
	return dm.WinnerName
}

// DemocracyConsensus returns consensus level.
func DemocracyConsensus(dm *DemocracyState) float64 {
	if dm == nil {
		return 0
	}
	return dm.ConsensusLevel
}

// DemocracyElections returns total elections held.
func DemocracyElections(dm *DemocracyState) int {
	if dm == nil {
		return 0
	}
	return dm.TotalElections
}
