package swarm

import (
	"math/rand"
	"testing"
)

func TestInitDemocracy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitDemocracy(ss)

	dm := ss.Democracy
	if dm == nil {
		t.Fatal("democracy should be initialized")
	}
	if len(dm.Ballots) != 15 {
		t.Fatalf("expected 15 ballots, got %d", len(dm.Ballots))
	}
	if len(dm.Proposals) != 5 {
		t.Fatalf("expected 5 proposals, got %d", len(dm.Proposals))
	}
}

func TestClearDemocracy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.DemocracyOn = true
	InitDemocracy(ss)
	ClearDemocracy(ss)

	if ss.Democracy != nil {
		t.Fatal("should be nil")
	}
	if ss.DemocracyOn {
		t.Fatal("should be false")
	}
}

func TestTickDemocracy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitDemocracy(ss)

	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		TickDemocracy(ss)
	}

	dm := ss.Democracy
	if dm.TotalElections == 0 {
		t.Fatal("should have held at least one election")
	}
	if dm.WinnerName == "" {
		t.Fatal("should have a winner")
	}
}

func TestTickDemocracyNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickDemocracy(ss) // should not panic
}

func TestRunElection(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 30)
	InitDemocracy(ss)

	// Bias most bots toward proposal 2
	for i := 0; i < 20; i++ {
		ss.Democracy.Ballots[i].Rankings = [5]int{2, 0, 1, 3, 4}
	}

	runElection(ss, ss.Democracy)

	if ss.Democracy.WinningStrategy != 2 {
		t.Fatalf("expected proposal 2 to win, got %d", ss.Democracy.WinningStrategy)
	}
	if ss.Democracy.WinnerName != "Schwarm" {
		t.Fatalf("expected Schwarm to win, got %s", ss.Democracy.WinnerName)
	}
}

func TestBoostProposal(t *testing.T) {
	ballot := &RankedBallot{
		Rankings: [5]int{0, 1, 2, 3, 4},
	}

	boostProposal(ballot, 3) // should move from pos 3 to pos 2
	if ballot.Rankings[2] != 3 {
		t.Fatalf("expected 3 at position 2, got %d", ballot.Rankings[2])
	}

	boostProposal(ballot, 3) // should move from pos 2 to pos 1
	if ballot.Rankings[1] != 3 {
		t.Fatalf("expected 3 at position 1, got %d", ballot.Rankings[1])
	}
}

func TestDemocracyWinner(t *testing.T) {
	if DemocracyWinner(nil) != "" {
		t.Fatal("nil should return empty string")
	}
}

func TestDemocracyConsensus(t *testing.T) {
	if DemocracyConsensus(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestDemocracyElections(t *testing.T) {
	if DemocracyElections(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestInstantRunoff(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitDemocracy(ss)

	// Set up a scenario where runoff matters:
	// 4 prefer A, 3 prefer B, 3 prefer C with B as second choice
	dm := ss.Democracy
	for i := 0; i < 4; i++ {
		dm.Ballots[i].Rankings = [5]int{0, 1, 2, 3, 4}
	}
	for i := 4; i < 7; i++ {
		dm.Ballots[i].Rankings = [5]int{1, 0, 2, 3, 4}
	}
	for i := 7; i < 10; i++ {
		dm.Ballots[i].Rankings = [5]int{2, 1, 0, 3, 4} // C first, B second
	}

	runElection(ss, dm)

	// No one has majority (5+). After elimination, B should accumulate.
	// The IRV should produce a winner eventually
	if dm.TotalElections == 0 {
		t.Fatal("election should have completed")
	}
	if dm.RoundsLastElec < 1 {
		t.Fatal("should have at least 1 round")
	}
}
