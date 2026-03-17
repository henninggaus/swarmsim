package swarm

import (
	"math/rand"
	"testing"
)

func TestInitPredatorPrey(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitPredatorPrey(ss, 0.2)

	if ss.PredatorPrey == nil {
		t.Fatal("predator-prey should be initialized")
	}
	pp := ss.PredatorPrey
	if pp.PredatorCount != 4 {
		t.Fatalf("expected 4 predators, got %d", pp.PredatorCount)
	}
	if pp.PreyCount != 16 {
		t.Fatalf("expected 16 prey, got %d", pp.PreyCount)
	}

	// Check roles assigned
	predCount := 0
	for _, r := range pp.Roles {
		if r == RolePredator {
			predCount++
		}
	}
	if predCount != 4 {
		t.Fatalf("expected 4 predator roles, got %d", predCount)
	}
}

func TestPredatorPreyFitness(t *testing.T) {
	pp := &PredatorPreyState{
		Roles:       []PredatorPreyRole{RolePredator, RolePrey},
		CatchCount:  []int{5, 0},
		EscapeCount: []int{0, 1000},
	}

	predFit := PredatorPreyFitness(pp, 0)
	if predFit != 500 {
		t.Fatalf("expected predator fitness 500, got %f", predFit)
	}

	preyFit := PredatorPreyFitness(pp, 1)
	if preyFit != 100 {
		t.Fatalf("expected prey fitness 100, got %f", preyFit)
	}
}

func TestIsPredatorPrey(t *testing.T) {
	pp := &PredatorPreyState{
		Roles: []PredatorPreyRole{RolePredator, RolePrey, RolePrey},
	}
	if !IsPredator(pp, 0) {
		t.Fatal("bot 0 should be predator")
	}
	if !IsPrey(pp, 1) {
		t.Fatal("bot 1 should be prey")
	}
	if IsPredator(pp, 1) {
		t.Fatal("bot 1 should not be predator")
	}
}

func TestTickPredatorPrey(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitPredatorPrey(ss, 0.2)

	// Place predator right on top of prey
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100
	ss.Bots[4].X = 105 // prey very close
	ss.Bots[4].Y = 100

	TickPredatorPrey(ss)

	if ss.PredatorPrey.TotalCatches < 1 {
		t.Fatal("should have caught at least one prey")
	}
}

func TestPredatorPreyBotCount(t *testing.T) {
	pp := &PredatorPreyState{
		PredatorCount: 5,
		PreyCount:     15,
	}
	pred, prey := PredatorPreyBotCount(pp)
	if pred != 5 || prey != 15 {
		t.Fatalf("expected 5/15, got %d/%d", pred, prey)
	}
}

func TestBuildPredatorPreyInputs(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitPredatorPrey(ss, 0.3)

	inp := BuildPredatorPreyInputs(&ss.Bots[0], ss, 0)
	// Bias should always be 1.0
	if inp[11] != 1.0 {
		t.Fatalf("bias should be 1.0, got %f", inp[11])
	}
}
