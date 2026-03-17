package swarm

import (
	"math/rand"
	"testing"
)

func TestInitMetaEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMetaEvolution(ss)

	me := ss.MetaEvo
	if me == nil {
		t.Fatal("meta evo should be initialized")
	}
	if len(me.Params) != 20 {
		t.Fatalf("expected 20 params, got %d", len(me.Params))
	}
	// Check reasonable ranges
	for i, p := range me.Params {
		if p.MutationRate < 0.01 || p.MutationRate > 0.5 {
			t.Fatalf("bot %d: mutation rate %.3f out of range", i, p.MutationRate)
		}
		if p.CrossoverRate < 0.1 || p.CrossoverRate > 0.9 {
			t.Fatalf("bot %d: crossover rate %.3f out of range", i, p.CrossoverRate)
		}
	}
}

func TestClearMetaEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.MetaEvoOn = true
	InitMetaEvolution(ss)
	ClearMetaEvolution(ss)

	if ss.MetaEvo != nil {
		t.Fatal("should be nil")
	}
	if ss.MetaEvoOn {
		t.Fatal("should be false")
	}
}

func TestGetMetaMutationRate(t *testing.T) {
	if GetMetaMutationRate(nil, 0) != 0.1 {
		t.Fatal("nil should return default 0.1")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitMetaEvolution(ss)

	rate := GetMetaMutationRate(ss.MetaEvo, 0)
	if rate < 0.01 || rate > 0.5 {
		t.Fatalf("rate %.3f out of range", rate)
	}
}

func TestGetMetaCrossoverRate(t *testing.T) {
	if GetMetaCrossoverRate(nil, 0) != 0.5 {
		t.Fatal("nil should return default 0.5")
	}
}

func TestRecordFitness(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitMetaEvolution(ss)

	RecordFitness(ss.MetaEvo, 0, 10)
	RecordFitness(ss.MetaEvo, 0, 20)
	RecordFitness(ss.MetaEvo, 0, 30)

	p := ss.MetaEvo.Params[0]
	if len(p.FitnessHistory) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(p.FitnessHistory))
	}
	if p.Improvement <= 0 {
		t.Fatal("improvement should be positive with increasing fitness")
	}

	// Test nil safety
	RecordFitness(nil, 0, 10) // should not panic
}

func TestEvolveMetaParams(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMetaEvolution(ss)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveMetaParams(ss, sorted)

	if ss.MetaEvo.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.MetaEvo.Generation)
	}
	if ss.MetaEvo.AvgMutationRate <= 0 {
		t.Fatal("avg mutation rate should be > 0")
	}
}

func TestUpdateMetaStats(t *testing.T) {
	me := &MetaEvoState{
		Params: []MetaParams{
			{MutationRate: 0.1, CrossoverRate: 0.5, SelectionPress: 2.0, ExplorationRate: 0.3},
			{MutationRate: 0.2, CrossoverRate: 0.6, SelectionPress: 3.0, ExplorationRate: 0.4},
		},
	}
	updateMetaStats(me)

	if me.AvgMutationRate < 0.14 || me.AvgMutationRate > 0.16 {
		t.Fatalf("expected ~0.15, got %.3f", me.AvgMutationRate)
	}
	if me.Diversity <= 0 {
		t.Fatal("diversity should be > 0 for different params")
	}
}

func TestMetaAvgMutationRate(t *testing.T) {
	if MetaAvgMutationRate(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestMetaDiversity(t *testing.T) {
	if MetaDiversity(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}
