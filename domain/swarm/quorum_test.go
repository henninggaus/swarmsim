package swarm

import (
	"math/rand"
	"testing"
)

func TestInitQuorum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitQuorum(ss)

	if ss.Quorum == nil {
		t.Fatal("quorum should be initialized")
	}
	if len(ss.Quorum.Votes) != 20 {
		t.Fatalf("expected 20 votes, got %d", len(ss.Quorum.Votes))
	}
	if ss.Quorum.Threshold != 0.6 {
		t.Fatal("default threshold should be 0.6")
	}
}

func TestClearQuorum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ss.QuorumOn = true
	InitQuorum(ss)
	ClearQuorum(ss)

	if ss.Quorum != nil {
		t.Fatal("quorum should be nil after clear")
	}
	if ss.QuorumOn {
		t.Fatal("QuorumOn should be false")
	}
}

func TestTickQuorum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 30)
	InitQuorum(ss)

	// Place bots close together so they can sense each other
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i%6)*10
		ss.Bots[i].Y = 400 + float64(i/6)*10
	}

	for tick := 0; tick < 50; tick++ {
		ss.Tick = tick
		TickQuorum(ss)
	}

	if ss.Quorum.TotalVotes != 30 {
		t.Fatalf("expected 30 total votes, got %d", ss.Quorum.TotalVotes)
	}
}

func TestTickQuorumNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickQuorum(ss) // should not panic
}

func TestProposalFromState(t *testing.T) {
	bot := &SwarmBot{CarryingPkg: 0}
	if proposalFromState(bot) != ProposalMigrate {
		t.Fatal("carrying bot should propose migration")
	}

	bot = &SwarmBot{CarryingPkg: -1, NearestPickupDist: 30}
	if proposalFromState(bot) != ProposalCluster {
		t.Fatal("near-pickup bot should propose cluster")
	}

	bot = &SwarmBot{CarryingPkg: -1, NearestPickupDist: 200, NeighborCount: 12}
	if proposalFromState(bot) != ProposalDisperse {
		t.Fatal("crowded bot should propose disperse")
	}
}

func TestProposalName(t *testing.T) {
	if ProposalName(ProposalMigrate) != "Migration" {
		t.Fatal("expected Migration")
	}
	if ProposalName(ProposalAlarm) != "Alarm" {
		t.Fatal("expected Alarm")
	}
	if ProposalName(99) != "?" {
		t.Fatal("expected ? for unknown")
	}
}

func TestQuorumDecisionCount(t *testing.T) {
	if QuorumDecisionCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	qs := &QuorumState{
		Decisions: []QuorumDecision{{}, {}},
	}
	if QuorumDecisionCount(qs) != 2 {
		t.Fatal("expected 2")
	}
}

func TestQuorumAvgAgreement(t *testing.T) {
	if QuorumAvgAgreement(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	qs := &QuorumState{
		Votes: []BotVote{
			{LocalCount: 10, LocalAgree: 8},
			{LocalCount: 10, LocalAgree: 6},
		},
	}
	avg := QuorumAvgAgreement(qs)
	if avg < 0.69 || avg > 0.71 {
		t.Fatalf("expected ~0.7, got %.3f", avg)
	}
}

func TestQuorumDecisionsTrigger(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitQuorum(ss)

	// Force all bots to same spot with same proposal
	for i := range ss.Bots {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
		ss.Quorum.Votes[i].Proposal = ProposalCluster
		ss.Quorum.Votes[i].Confidence = 0.9
	}

	ss.Tick = 1
	TickQuorum(ss)

	if len(ss.Quorum.Decisions) == 0 {
		t.Fatal("should trigger at least one decision when all agree")
	}
}
