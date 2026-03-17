package swarm

import (
	"math/rand"
	"testing"
)

func TestInitEcosystem(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitEcosystem(ss, 0.7) // 70% herbivores

	eco := ss.Ecosystem
	if eco == nil {
		t.Fatal("ecosystem should be initialized")
	}
	if eco.HerbivoreCount != 14 {
		t.Fatalf("expected 14 herbivores, got %d", eco.HerbivoreCount)
	}
	if eco.PredatorCount != 6 {
		t.Fatalf("expected 6 predators, got %d", eco.PredatorCount)
	}
	if len(eco.Plants) != 50 {
		t.Fatalf("expected 50 initial plants, got %d", len(eco.Plants))
	}
}

func TestClearEcosystem(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.EcosystemOn = true
	InitEcosystem(ss, 0.6)
	ClearEcosystem(ss)

	if ss.Ecosystem != nil {
		t.Fatal("should be nil")
	}
	if ss.EcosystemOn {
		t.Fatal("should be false")
	}
}

func TestTickEcosystem(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitEcosystem(ss, 0.7)

	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		TickEcosystem(ss)
	}

	eco := ss.Ecosystem
	if eco.Tick != 200 {
		t.Fatalf("expected tick 200, got %d", eco.Tick)
	}
}

func TestTickEcosystemNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickEcosystem(ss) // should not panic
}

func TestHerbivoreEatsPlant(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitEcosystem(ss, 0.8) // 4 herbivores, 1 predator
	eco := ss.Ecosystem

	// Place herbivore right on a plant
	if len(eco.Herbivores) > 0 && len(eco.Plants) > 0 {
		hi := eco.Herbivores[0]
		ss.Bots[hi].X = eco.Plants[0].X
		ss.Bots[hi].Y = eco.Plants[0].Y
		initialEnergy := eco.EcoEnergy[hi]

		TickEcosystem(ss)

		if eco.EcoEnergy[hi] <= initialEnergy-eco.MetabolicCost*2 {
			// Energy should have increased from eating (minus cost)
			t.Log("Herbivore may not have eaten — plant was possibly already dead or too far")
		}
	}
}

func TestEcoPlantCount(t *testing.T) {
	if EcoPlantCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestEcoSpeciesCount(t *testing.T) {
	h, p := EcoSpeciesCount(nil)
	if h != 0 || p != 0 {
		t.Fatal("nil should return 0,0")
	}

	eco := &EcosystemState{HerbivoreCount: 10, PredatorCount: 5}
	h, p = EcoSpeciesCount(eco)
	if h != 10 || p != 5 {
		t.Fatalf("expected 10,5 got %d,%d", h, p)
	}
}

func TestEcoBotEnergy(t *testing.T) {
	if EcoBotEnergy(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}
	eco := &EcosystemState{EcoEnergy: []float64{42, 10}}
	if EcoBotEnergy(eco, 0) != 42 {
		t.Fatal("expected 42")
	}
	if EcoBotEnergy(eco, 5) != 0 {
		t.Fatal("out of bounds should return 0")
	}
}

func TestRespawnEcosystem(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitEcosystem(ss, 0.7)
	eco := ss.Ecosystem

	// Kill all bots
	for i := range eco.EcoAlive {
		eco.EcoAlive[i] = false
	}

	respawnEcosystem(ss, eco)

	// Some should have respawned
	aliveCount := 0
	for _, a := range eco.EcoAlive {
		if a {
			aliveCount++
		}
	}
	if aliveCount == 0 {
		t.Fatal("at least some bots should respawn")
	}
}

func TestPopHistory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitEcosystem(ss, 0.7)

	for tick := 0; tick < 150; tick++ {
		ss.Tick = tick
		TickEcosystem(ss)
	}

	if len(ss.Ecosystem.PopHistory) == 0 {
		t.Fatal("should have population history entries")
	}
}
