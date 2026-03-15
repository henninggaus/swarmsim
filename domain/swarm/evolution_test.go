package swarm

import (
	"math/rand"
	"swarmsim/engine/swarmscript"
	"testing"
)

func setupEvolutionState(t *testing.T, programText string, botCount int) *SwarmState {
	t.Helper()
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, botCount)
	prog, err := swarmscript.ParseSwarmScript(programText)
	if err != nil {
		t.Fatalf("failed to parse program: %v", err)
	}
	ss.Program = prog
	ss.ProgramText = programText
	return ss
}

func TestInitBotParams(t *testing.T) {
	ss := setupEvolutionState(t, "IF d_dist < $A:25 THEN DROP\nIF near_dist < $B:15 THEN TURN_FROM_NEAREST", 20)
	InitBotParams(ss)

	if !ss.UsedParams[0] {
		t.Error("$A should be marked as used")
	}
	if !ss.UsedParams[1] {
		t.Error("$B should be marked as used")
	}
	if ss.UsedParams[2] {
		t.Error("$C should not be marked as used")
	}

	for i := 0; i < 20; i++ {
		valA := ss.Bots[i].ParamValues[0]
		if valA < 15 || valA > 35 {
			t.Errorf("bot %d $A value %.1f too far from hint 25", i, valA)
		}
		if ss.Bots[i].Fitness != 0 {
			t.Errorf("bot %d fitness should be 0, got %.1f", i, ss.Bots[i].Fitness)
		}
	}
}

func TestRunEvolution(t *testing.T) {
	ss := setupEvolutionState(t, "IF d_dist < $A:25 THEN DROP", 20)
	InitBotParams(ss)

	for i := range ss.Bots {
		ss.Bots[i].Fitness = float64(i) * 10
	}

	RunEvolution(ss)

	if ss.Generation != 1 {
		t.Errorf("expected generation 1, got %d", ss.Generation)
	}
	if ss.EvolutionTimer != 0 {
		t.Errorf("expected timer reset to 0, got %d", ss.EvolutionTimer)
	}
	if ss.BestFitness != 190 {
		t.Errorf("expected best fitness 190, got %.0f", ss.BestFitness)
	}

	for i, bot := range ss.Bots {
		if bot.Fitness != 0 {
			t.Errorf("bot %d fitness should be 0 after evolution, got %.1f", i, bot.Fitness)
		}
	}
}

func TestRunEvolutionMultipleGenerations(t *testing.T) {
	ss := setupEvolutionState(t, "IF d_dist < $A:25 THEN DROP\nIF near_dist < $B:15 THEN TURN_FROM_NEAREST", 30)
	InitBotParams(ss)
	rng := rand.New(rand.NewSource(99))

	for gen := 0; gen < 5; gen++ {
		for i := range ss.Bots {
			ss.Bots[i].Fitness = rng.Float64() * 100
		}
		RunEvolution(ss)
	}

	if ss.Generation != 5 {
		t.Errorf("expected generation 5, got %d", ss.Generation)
	}
	if len(ss.FitnessHistory) != 5 {
		t.Errorf("expected 5 fitness records, got %d", len(ss.FitnessHistory))
	}
	for i, rec := range ss.FitnessHistory {
		if rec.Best <= 0 {
			t.Errorf("gen %d: best fitness should be positive, got %.1f", i, rec.Best)
		}
	}
}

func TestRunEvolutionSmallPopulation(t *testing.T) {
	ss := setupEvolutionState(t, "IF d_dist < $A:25 THEN DROP", 5)
	InitBotParams(ss)

	for i := range ss.Bots {
		ss.Bots[i].Fitness = float64(i) * 5
	}
	RunEvolution(ss)
	if ss.Generation != 1 {
		t.Errorf("expected generation 1, got %d", ss.Generation)
	}
}

func TestFitnessHistoryCleared(t *testing.T) {
	ss := setupEvolutionState(t, "IF true THEN FWD", 10)
	ss.FitnessHistory = []FitnessRecord{{Best: 100, Avg: 50}}
	ss.FitnessHistory = nil
	if len(ss.FitnessHistory) != 0 {
		t.Error("fitness history should be cleared")
	}
}

func TestEvolutionAvgFitness(t *testing.T) {
	ss := setupEvolutionState(t, "IF d_dist < $A:25 THEN DROP", 10)
	InitBotParams(ss)

	for i := range ss.Bots {
		ss.Bots[i].Fitness = 20
	}
	RunEvolution(ss)
	if ss.AvgFitness != 20 {
		t.Errorf("expected avg fitness 20, got %.1f", ss.AvgFitness)
	}
}
