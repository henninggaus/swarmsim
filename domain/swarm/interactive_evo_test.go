package swarm

import (
	"math/rand"
	"testing"
)

func TestInitInteractiveEvo(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNeuro(ss)
	InitInteractiveEvo(ss, 6)

	if ss.InteractiveEvo == nil {
		t.Fatal("interactive evo should be initialized")
	}
	ie := ss.InteractiveEvo
	if len(ie.Candidates) != 6 {
		t.Fatalf("expected 6 candidates, got %d", len(ie.Candidates))
	}
	if len(ie.Selected) != 6 {
		t.Fatal("selected array should match candidates")
	}
}

func TestLoadCandidate(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNeuro(ss)
	InitInteractiveEvo(ss, 4)

	LoadCandidate(ss, 2)
	if ss.InteractiveEvo.ActiveCandidate != 2 {
		t.Fatal("active candidate should be 2")
	}
	// Weights should be loaded from candidate 2
	for w := 0; w < NeuroWeights; w++ {
		if ss.Bots[0].Brain.Weights[w] != ss.InteractiveEvo.Candidates[2].Weights[0][w] {
			t.Fatal("weights should match candidate")
		}
	}
}

func TestSelectCandidate(t *testing.T) {
	ie := &InteractiveEvoState{
		Selected: make([]bool, 4),
	}
	SelectCandidate(ie, 1)
	if !ie.Selected[1] {
		t.Fatal("candidate 1 should be selected")
	}
	SelectCandidate(ie, 1) // toggle off
	if ie.Selected[1] {
		t.Fatal("candidate 1 should be deselected")
	}
}

func TestSetUserScore(t *testing.T) {
	ie := &InteractiveEvoState{
		Candidates: make([]InteractiveCandidate, 4),
	}
	SetUserScore(ie, 2, 4)
	if ie.Candidates[2].UserScore != 4 {
		t.Fatal("score should be 4")
	}
	SetUserScore(ie, 2, 10) // clamp to 5
	if ie.Candidates[2].UserScore != 5 {
		t.Fatal("score should be clamped to 5")
	}
}

func TestEvolveInteractive(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNeuro(ss)
	InitInteractiveEvo(ss, 4)

	// Select candidates 0 and 2
	ie := ss.InteractiveEvo
	ie.Selected[0] = true
	ie.Selected[2] = true

	EvolveInteractive(ss)

	if ie.Generation != 1 {
		t.Fatalf("expected generation 1, got %d", ie.Generation)
	}
	if len(ie.Candidates) != 4 {
		t.Fatalf("should still have 4 candidates, got %d", len(ie.Candidates))
	}
	// Selection should be reset
	for _, s := range ie.Selected {
		if s {
			t.Fatal("selection should be reset after evolution")
		}
	}
}

func TestInteractiveSelectedCount(t *testing.T) {
	ie := &InteractiveEvoState{
		Selected: []bool{true, false, true, false, true},
	}
	if InteractiveSelectedCount(ie) != 3 {
		t.Fatal("expected 3 selected")
	}
}

func TestTopByUserScore(t *testing.T) {
	ie := &InteractiveEvoState{
		Candidates: []InteractiveCandidate{
			{UserScore: 1},
			{UserScore: 5},
			{UserScore: 3},
			{UserScore: 4},
		},
	}
	top := topByUserScore(ie, 2)
	if len(top) != 2 {
		t.Fatalf("expected 2, got %d", len(top))
	}
	if ie.Candidates[top[0]].UserScore < ie.Candidates[top[1]].UserScore {
		t.Fatal("top should be sorted by score descending")
	}
}
