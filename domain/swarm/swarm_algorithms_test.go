package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestSwarmAlgorithmName(t *testing.T) {
	if SwarmAlgorithmName(AlgoBoids) != "Boids (Reynolds)" {
		t.Fatal("wrong name for Boids")
	}
	if SwarmAlgorithmName(AlgoNone) != "Keiner" {
		t.Fatal("wrong name for None")
	}
}

func TestInitSwarmAlgorithmBoids(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSwarmAlgorithm(ss, AlgoBoids)

	if ss.SwarmAlgo == nil {
		t.Fatal("swarm algo should be initialized")
	}
	if ss.SwarmAlgo.ActiveAlgo != AlgoBoids {
		t.Fatal("should be Boids")
	}
}

func TestTickBoids(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSwarmAlgorithm(ss, AlgoBoids)

	// Run a few ticks
	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	// Bots should be moving
	moving := false
	for _, bot := range ss.Bots {
		if bot.Speed > 0 {
			moving = true
			break
		}
	}
	if !moving {
		t.Fatal("bots should be moving after Boids ticks")
	}
}

func TestInitSwarmAlgorithmPSO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSwarmAlgorithm(ss, AlgoPSO)

	if ss.SwarmAlgo.PSOPersonalBest == nil {
		t.Fatal("PSO personal best should be initialized")
	}
	if len(ss.SwarmAlgo.PSOPersonalBest) != 20 {
		t.Fatalf("expected 20 particles, got %d", len(ss.SwarmAlgo.PSOPersonalBest))
	}
}

func TestTickPSO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400
	InitSwarmAlgorithm(ss, AlgoPSO)

	for i := 0; i < 50; i++ {
		TickSwarmAlgorithm(ss)
	}
	// Global best should be updated
	if ss.SwarmAlgo.PSOGlobalBestF == -1e9 {
		t.Fatal("global best should be updated")
	}
}

func TestInitSwarmAlgorithmACO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSwarmAlgorithm(ss, AlgoACO)

	if ss.SwarmAlgo.ACOGrid == nil {
		t.Fatal("ACO grid should be initialized")
	}
	if ss.SwarmAlgo.ACOGridCols <= 0 {
		t.Fatal("ACO grid should have columns")
	}
}

func TestTickACO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSwarmAlgorithm(ss, AlgoACO)

	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	// Some pheromone should have been deposited
	hasPher := false
	for _, v := range ss.SwarmAlgo.ACOGrid {
		if v > 0 {
			hasPher = true
			break
		}
	}
	if !hasPher {
		t.Fatal("should have pheromone deposited")
	}
}

func TestTickFirefly(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400
	InitSwarmAlgorithm(ss, AlgoFirefly)

	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	// All bots should have some brightness
	for _, b := range ss.SwarmAlgo.FireflyBrightness {
		if math.IsNaN(b) {
			t.Fatal("brightness should not be NaN")
		}
	}
}

func TestHsvToRGB(t *testing.T) {
	r, g, b := hsvToRGB(0, 1, 1) // pure red
	if r != 255 || g != 0 || b != 0 {
		t.Fatalf("expected (255,0,0) got (%d,%d,%d)", r, g, b)
	}

	r, g, b = hsvToRGB(0.333, 1, 1) // green-ish
	if g < 200 {
		t.Fatalf("expected green > 200, got %d", g)
	}
}

func TestAlgorithmNames(t *testing.T) {
	names := AlgorithmNames()
	if len(names) != 5 {
		t.Fatalf("expected 5 names, got %d", len(names))
	}
}
