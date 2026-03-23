package swarm

import (
	"math"
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeAOState(n int) *SwarmState {
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

func TestInitAO(t *testing.T) {
	ss := makeAOState(20)
	InitAO(ss)
	if ss.AO == nil {
		t.Fatal("AO state should not be nil after init")
	}
	if !ss.AOOn {
		t.Fatal("AOOn should be true after init")
	}
	if len(ss.AO.Fitness) != 20 {
		t.Fatalf("expected 20 fitness values, got %d", len(ss.AO.Fitness))
	}
	if len(ss.AO.Phase) != 20 {
		t.Fatalf("expected 20 phase values, got %d", len(ss.AO.Phase))
	}
}

func TestClearAO(t *testing.T) {
	ss := makeAOState(10)
	InitAO(ss)
	ClearAO(ss)
	if ss.AO != nil {
		t.Fatal("AO should be nil after clear")
	}
	if ss.AOOn {
		t.Fatal("AOOn should be false after clear")
	}
}

func TestTickAO(t *testing.T) {
	ss := makeAOState(20)
	InitAO(ss)
	for tick := 0; tick < 50; tick++ {
		TickAO(ss)
	}
	st := ss.AO
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].AOPhase < 0 || ss.Bots[i].AOPhase > 3 {
			t.Fatalf("bot %d: AOPhase out of range: %d", i, ss.Bots[i].AOPhase)
		}
	}
}

func TestTickAONil(t *testing.T) {
	ss := makeAOState(10)
	// Should not panic when AO is nil
	TickAO(ss)
}

func TestApplyAO(t *testing.T) {
	ss := makeAOState(20)
	InitAO(ss)
	for tick := 0; tick < 10; tick++ {
		TickAO(ss)
	}
	// Apply to a non-best eagle
	for i := range ss.Bots {
		if i != ss.AO.BestIdx {
			ApplyAO(&ss.Bots[i], ss, i)
			if ss.Bots[i].Speed <= 0 {
				t.Fatal("eagle speed should be positive")
			}
			break
		}
	}
}

func TestApplyAOBestEagle(t *testing.T) {
	ss := makeAOState(20)
	InitAO(ss)
	for tick := 0; tick < 10; tick++ {
		TickAO(ss)
	}
	bestIdx := ss.AO.BestIdx
	if bestIdx >= 0 {
		ApplyAO(&ss.Bots[bestIdx], ss, bestIdx)
		// Best eagle (prey) should get gold LED
		if ss.Bots[bestIdx].LEDColor != [3]uint8{255, 215, 0} {
			t.Fatal("best eagle should have gold LED")
		}
	}
}

func TestAOGrowSlices(t *testing.T) {
	ss := makeAOState(5)
	InitAO(ss)
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	TickAO(ss)
	if len(ss.AO.Fitness) != 10 {
		t.Fatalf("expected 10 fitness values after grow, got %d", len(ss.AO.Fitness))
	}
}

func TestAOAllPhases(t *testing.T) {
	ss := makeAOState(50)
	InitAO(ss)
	// Run many ticks to hit all phases (exploration early, exploitation late)
	phasesSeen := map[int]bool{}
	for tick := 0; tick < 500; tick++ {
		TickAO(ss)
		for i := range ss.Bots {
			phasesSeen[ss.AO.Phase[i]] = true
			if i != ss.AO.BestIdx {
				ApplyAO(&ss.Bots[i], ss, i)
			}
		}
	}
	// Should see all 4 phases
	for p := 0; p <= 3; p++ {
		if !phasesSeen[p] {
			t.Errorf("phase %d never seen in 500 ticks", p)
		}
	}
}

func TestAOLevyStep(t *testing.T) {
	ss := makeAOState(1)
	for i := 0; i < 100; i++ {
		step := aoLevyStep(ss)
		if math.IsNaN(step) || math.IsInf(step, 0) {
			t.Fatalf("aoLevyStep returned %v", step)
		}
	}
}

func TestAOMeanComputation(t *testing.T) {
	ss := makeAOState(10)
	InitAO(ss)
	TickAO(ss)
	// Mean should be within arena bounds
	if ss.AO.MeanX < 0 || ss.AO.MeanX > 800 {
		t.Fatalf("MeanX out of bounds: %f", ss.AO.MeanX)
	}
	if ss.AO.MeanY < 0 || ss.AO.MeanY > 800 {
		t.Fatalf("MeanY out of bounds: %f", ss.AO.MeanY)
	}
}

func TestAOCycleReset(t *testing.T) {
	ss := makeAOState(10)
	InitAO(ss)
	// Run past the max ticks to trigger a cycle reset
	for tick := 0; tick < aoMaxTicks+10; tick++ {
		TickAO(ss)
	}
	if ss.AO.HuntTick > aoMaxTicks {
		t.Fatalf("HuntTick should have reset, got %d", ss.AO.HuntTick)
	}
}
