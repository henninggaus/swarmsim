package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeGSAState(n int) *SwarmState {
	ss := &SwarmState{
		Bots:   make([]SwarmBot, n),
		ArenaW: 800,
		ArenaH: 800,
		Rng:    rand.New(rand.NewSource(42)),
		Hash:   physics.NewSpatialHash(800, 800, 30),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = ss.Rng.Float64() * 800
		ss.Bots[i].Y = ss.Rng.Float64() * 800
		ss.Bots[i].Angle = ss.Rng.Float64() * 6.28
		ss.Bots[i].Energy = 80
		ss.Bots[i].CarryingPkg = -1
	}
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}
	return ss
}

func TestInitGSA(t *testing.T) {
	ss := makeGSAState(20)
	InitGSA(ss)
	if ss.GSA == nil {
		t.Fatal("GSA state should not be nil after init")
	}
	if !ss.GSAOn {
		t.Fatal("GSAOn should be true after init")
	}
	if len(ss.GSA.Mass) != 20 {
		t.Fatalf("expected 20 masses, got %d", len(ss.GSA.Mass))
	}
}

func TestTickGSA(t *testing.T) {
	ss := makeGSAState(20)
	InitGSA(ss)
	for tick := 0; tick < 50; tick++ {
		TickGSA(ss)
	}
	st := ss.GSA
	// Best index should be valid
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	// G should have decayed from initial value
	if st.G >= gsaG0 {
		t.Fatalf("G should have decayed, got %f", st.G)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].GSAMass < 0 {
			t.Fatalf("bot %d: GSAMass should be non-negative, got %d", i, ss.Bots[i].GSAMass)
		}
		if ss.Bots[i].GSAForce < 0 {
			t.Fatalf("bot %d: GSAForce should be non-negative, got %d", i, ss.Bots[i].GSAForce)
		}
	}
}

func TestTickGSANil(t *testing.T) {
	ss := makeGSAState(10)
	// Should not panic when GSA is nil
	TickGSA(ss)
}

func TestClearGSA(t *testing.T) {
	ss := makeGSAState(10)
	InitGSA(ss)
	ClearGSA(ss)
	if ss.GSA != nil {
		t.Fatal("GSA should be nil after clear")
	}
	if ss.GSAOn {
		t.Fatal("GSAOn should be false after clear")
	}
}

func TestApplyGSA(t *testing.T) {
	ss := makeGSAState(20)
	InitGSA(ss)
	for tick := 0; tick < 10; tick++ {
		TickGSA(ss)
	}
	// Apply to a non-best bot
	for i := range ss.Bots {
		if i != ss.GSA.BestIdx {
			ApplyGSA(&ss.Bots[i], ss, i)
			if ss.Bots[i].Speed <= 0 {
				t.Fatal("bot speed should be positive after ApplyGSA")
			}
			break
		}
	}
}

func TestApplyGSABestGetsGold(t *testing.T) {
	ss := makeGSAState(20)
	InitGSA(ss)
	for tick := 0; tick < 10; tick++ {
		TickGSA(ss)
	}
	bestIdx := ss.GSA.BestIdx
	ApplyGSA(&ss.Bots[bestIdx], ss, bestIdx)
	if ss.Bots[bestIdx].LEDColor != [3]uint8{255, 215, 0} {
		t.Fatalf("best bot should have gold LED, got %v", ss.Bots[bestIdx].LEDColor)
	}
}

func TestGSAGrowSlices(t *testing.T) {
	ss := makeGSAState(5)
	InitGSA(ss)
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	// Should not panic
	TickGSA(ss)
	if len(ss.GSA.Mass) != 10 {
		t.Fatalf("expected 10 masses after grow, got %d", len(ss.GSA.Mass))
	}
}

func TestGSAMassNormalization(t *testing.T) {
	ss := makeGSAState(10)
	// Place bots at known positions for predictable fitness
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i) * 80
		ss.Bots[i].Y = 400
	}
	InitGSA(ss)
	TickGSA(ss)

	// Masses should sum to approximately 1.0
	totalMass := 0.0
	for _, m := range ss.GSA.Mass {
		totalMass += m
	}
	if totalMass < 0.99 || totalMass > 1.01 {
		t.Fatalf("masses should sum to ~1.0, got %f", totalMass)
	}
}
