package swarm

import (
	"math"
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeHHOState(n int) *SwarmState {
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

func TestInitHHO(t *testing.T) {
	ss := makeHHOState(20)
	InitHHO(ss)
	if ss.HHO == nil {
		t.Fatal("HHO state should not be nil after init")
	}
	if !ss.HHOOn {
		t.Fatal("HHOOn should be true after init")
	}
	if len(ss.HHO.Fitness) != 20 {
		t.Fatalf("expected 20 fitness values, got %d", len(ss.HHO.Fitness))
	}
	if len(ss.HHO.Phase) != 20 {
		t.Fatalf("expected 20 phase values, got %d", len(ss.HHO.Phase))
	}
}

func TestTickHHO(t *testing.T) {
	ss := makeHHOState(20)
	InitHHO(ss)
	for tick := 0; tick < 50; tick++ {
		TickHHO(ss)
	}
	st := ss.HHO
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].HHOPhase < 0 || ss.Bots[i].HHOPhase > 3 {
			t.Fatalf("bot %d: HHOPhase out of range: %d", i, ss.Bots[i].HHOPhase)
		}
	}
}

func TestTickHHONil(t *testing.T) {
	ss := makeHHOState(10)
	// Should not panic when HHO is nil
	TickHHO(ss)
}

func TestClearHHO(t *testing.T) {
	ss := makeHHOState(10)
	InitHHO(ss)
	ClearHHO(ss)
	if ss.HHO != nil {
		t.Fatal("HHO should be nil after clear")
	}
	if ss.HHOOn {
		t.Fatal("HHOOn should be false after clear")
	}
}

func TestApplyHHO(t *testing.T) {
	ss := makeHHOState(20)
	InitHHO(ss)
	for tick := 0; tick < 10; tick++ {
		TickHHO(ss)
	}
	// Apply to a non-best hawk
	for i := range ss.Bots {
		if i != ss.HHO.BestIdx {
			ApplyHHO(&ss.Bots[i], ss, i)
			if ss.Bots[i].Speed <= 0 {
				t.Fatal("hawk speed should be positive")
			}
			break
		}
	}
}

func TestApplyHHOBestHawk(t *testing.T) {
	ss := makeHHOState(20)
	InitHHO(ss)
	for tick := 0; tick < 10; tick++ {
		TickHHO(ss)
	}
	bestIdx := ss.HHO.BestIdx
	if bestIdx >= 0 {
		ApplyHHO(&ss.Bots[bestIdx], ss, bestIdx)
		// Best hawk (rabbit) should get gold LED
		if ss.Bots[bestIdx].LEDColor != [3]uint8{255, 215, 0} {
			t.Fatal("best hawk should have gold LED")
		}
	}
}

func TestHHOGrowSlices(t *testing.T) {
	ss := makeHHOState(5)
	InitHHO(ss)
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	TickHHO(ss)
	if len(ss.HHO.Fitness) != 10 {
		t.Fatalf("expected 10 fitness values after grow, got %d", len(ss.HHO.Fitness))
	}
}

func TestHHOAllPhases(t *testing.T) {
	ss := makeHHOState(50)
	InitHHO(ss)
	// Run many ticks to hit all phases
	phasesSeen := map[int]bool{}
	for tick := 0; tick < 200; tick++ {
		TickHHO(ss)
		for i := range ss.Bots {
			phasesSeen[ss.HHO.Phase[i]] = true
			if i != ss.HHO.BestIdx {
				ApplyHHO(&ss.Bots[i], ss, i)
			}
		}
	}
	// Should see all 4 phases
	for p := 0; p <= 3; p++ {
		if !phasesSeen[p] {
			t.Errorf("phase %d never seen in 200 ticks", p)
		}
	}
}

func TestLevyStep(t *testing.T) {
	ss := makeHHOState(1)
	// Just verify it doesn't panic or return NaN
	for i := 0; i < 100; i++ {
		step := levyStep(ss)
		if math.IsNaN(step) || math.IsInf(step, 0) {
			t.Fatalf("levyStep returned %v", step)
		}
	}
}
