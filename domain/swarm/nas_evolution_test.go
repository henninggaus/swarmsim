package swarm

import (
	"math/rand"
	"testing"
)

func TestInitNASEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNASEvolution(ss)

	evo := ss.NASEvo
	if evo == nil {
		t.Fatal("NAS evolution should be initialized")
	}
	if len(evo.Genomes) != 10 {
		t.Fatalf("expected 10 genomes, got %d", len(evo.Genomes))
	}
	for i, g := range evo.Genomes {
		if g == nil {
			t.Fatalf("genome %d should not be nil", i)
		}
		if len(g.Nodes) < 10 {
			t.Fatalf("genome %d: expected at least 10 nodes (6 in + 4 out), got %d", i, len(g.Nodes))
		}
	}
}

func TestClearNASEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.NASEvoOn = true
	InitNASEvolution(ss)
	ClearNASEvolution(ss)

	if ss.NASEvo != nil {
		t.Fatal("should be nil")
	}
	if ss.NASEvoOn {
		t.Fatal("should be false")
	}
}

func TestTickNASEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNASEvolution(ss)

	for tick := 0; tick < 50; tick++ {
		TickNASEvolution(ss)
	}
	// Should not panic
}

func TestTickNASEvolutionNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickNASEvolution(ss) // should not panic
}

func TestEvolveNASEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNASEvolution(ss)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveNASEvolution(ss, sorted)

	if ss.NASEvo.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.NASEvo.Generation)
	}
}

func TestNASEvoAvgComplexity(t *testing.T) {
	if NASEvoAvgComplexity(nil) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitNASEvolution(ss)
	updateNASEvoStats(ss.NASEvo)

	c := NASEvoAvgComplexity(ss.NASEvo)
	if c <= 0 {
		t.Fatal("complexity should be > 0")
	}
}

func TestNASEvoBotNodeCount(t *testing.T) {
	if NASEvoBotNodeCount(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitNASEvolution(ss)

	count := NASEvoBotNodeCount(ss.NASEvo, 0)
	if count < 10 {
		t.Fatalf("expected at least 10 nodes, got %d", count)
	}
}

func TestNASEvoMultipleGenerations(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitNASEvolution(ss)

	sorted := make([]int, 15)
	for i := range sorted {
		sorted[i] = i
	}

	for gen := 0; gen < 5; gen++ {
		for tick := 0; tick < 20; tick++ {
			TickNASEvolution(ss)
		}
		EvolveNASEvolution(ss, sorted)
	}

	if ss.NASEvo.Generation != 5 {
		t.Fatalf("expected gen 5, got %d", ss.NASEvo.Generation)
	}
}
