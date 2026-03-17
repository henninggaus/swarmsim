package swarm

import (
	"math/rand"
	"testing"
)

func TestInitBodyEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitBodyEvolution(ss)

	be := ss.BodyEvo
	if be == nil {
		t.Fatal("body evolution should be initialized")
	}
	if len(be.Bodies) != 15 {
		t.Fatalf("expected 15 bodies, got %d", len(be.Bodies))
	}
	for i, b := range be.Bodies {
		if b.Size < 0.5 || b.Size > 2.0 {
			t.Fatalf("bot %d: size %.2f out of range", i, b.Size)
		}
		if b.MaxSpeed <= 0 {
			t.Fatalf("bot %d: speed should be positive", i)
		}
	}
}

func TestClearBodyEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.BodyEvoOn = true
	InitBodyEvolution(ss)
	ClearBodyEvolution(ss)

	if ss.BodyEvo != nil {
		t.Fatal("should be nil")
	}
	if ss.BodyEvoOn {
		t.Fatal("should be false")
	}
}

func TestTickBodyEvolution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitBodyEvolution(ss)

	// Set some bots faster than their max
	for i := range ss.Bots {
		ss.Bots[i].Speed = SwarmBotSpeed * 3
	}

	TickBodyEvolution(ss)

	// Speeds should be capped
	for i := range ss.Bots {
		if ss.Bots[i].Speed > ss.BodyEvo.Bodies[i].MaxSpeed+0.001 {
			t.Fatalf("bot %d: speed %.2f exceeds max %.2f", i,
				ss.Bots[i].Speed, ss.BodyEvo.Bodies[i].MaxSpeed)
		}
	}
}

func TestTickBodyEvolutionNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickBodyEvolution(ss) // should not panic
}

func TestEvolveBodyPlans(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitBodyEvolution(ss)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveBodyPlans(ss, sorted)
	if ss.BodyEvo.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.BodyEvo.Generation)
	}
}

func TestExpressBody(t *testing.T) {
	// Small bot
	small := BotBody{Genes: [6]float64{0.0, 0.5, 0.5, 0.0, 0, 0}}
	expressBody(&small)

	// Large bot
	large := BotBody{Genes: [6]float64{1.0, 0.95, 0.5, 0.0, 0, 0}}
	expressBody(&large)

	if small.Size >= large.Size {
		t.Fatal("small should be smaller than large")
	}
	if small.MaxSpeed <= large.MaxSpeed {
		t.Fatal("small should be faster than large")
	}
	if large.CarryCapacity <= small.CarryCapacity {
		t.Fatal("large should carry more")
	}
}

func TestBodySize(t *testing.T) {
	if BodySize(nil, 0) != 1.0 {
		t.Fatal("nil should return 1.0")
	}
}

func TestBodyMaxSpeed(t *testing.T) {
	if BodyMaxSpeed(nil, 0) != SwarmBotSpeed {
		t.Fatal("nil should return SwarmBotSpeed")
	}
}

func TestAvgBodySize(t *testing.T) {
	if AvgBodySize(nil) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitBodyEvolution(ss)
	updateBodyStats(ss.BodyEvo)

	avg := AvgBodySize(ss.BodyEvo)
	if avg < 0.5 || avg > 2.0 {
		t.Fatalf("average size %.2f out of expected range", avg)
	}
}

func TestBodySizeDiversity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitBodyEvolution(ss)
	updateBodyStats(ss.BodyEvo)

	if ss.BodyEvo.SizeDiversity <= 0 {
		t.Fatal("diversity should be > 0 with random init")
	}
}
