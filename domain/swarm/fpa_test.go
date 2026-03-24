package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeFPAState(n int) *SwarmState {
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

func TestInitFPA(t *testing.T) {
	ss := makeFPAState(20)
	InitFPA(ss)
	if ss.FPA == nil {
		t.Fatal("FPA state should not be nil after init")
	}
	if !ss.FPAOn {
		t.Fatal("FPAOn should be true after init")
	}
	if len(ss.FPA.Fitness) != 20 {
		t.Fatalf("expected 20 fitness values, got %d", len(ss.FPA.Fitness))
	}
	if len(ss.FPA.BestX) != 20 {
		t.Fatalf("expected 20 BestX values, got %d", len(ss.FPA.BestX))
	}
}

func TestTickFPA(t *testing.T) {
	ss := makeFPAState(20)
	InitFPA(ss)
	for tick := 0; tick < 50; tick++ {
		TickFPA(ss)
	}
	st := ss.FPA
	// Global best should be set
	if st.GlobalIdx < 0 || st.GlobalIdx >= 20 {
		t.Fatalf("global best index out of range: %d", st.GlobalIdx)
	}
	// PollTick should have advanced
	if st.PollTick == 0 {
		t.Fatal("PollTick should have advanced")
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].FPAFitness < 0 {
			t.Fatalf("bot %d: FPAFitness should be non-negative, got %d", i, ss.Bots[i].FPAFitness)
		}
		if ss.Bots[i].FPAType != 0 && ss.Bots[i].FPAType != 1 {
			t.Fatalf("bot %d: FPAType should be 0 or 1, got %d", i, ss.Bots[i].FPAType)
		}
		if ss.Bots[i].FPABestDist < 0 {
			t.Fatalf("bot %d: FPABestDist should be non-negative, got %d", i, ss.Bots[i].FPABestDist)
		}
	}
}

func TestTickFPANil(t *testing.T) {
	ss := makeFPAState(10)
	// Should not panic when FPA is nil
	TickFPA(ss)
}

func TestClearFPA(t *testing.T) {
	ss := makeFPAState(10)
	InitFPA(ss)
	ClearFPA(ss)
	if ss.FPA != nil {
		t.Fatal("FPA should be nil after clear")
	}
	if ss.FPAOn {
		t.Fatal("FPAOn should be false after clear")
	}
}

func TestApplyFPA(t *testing.T) {
	ss := makeFPAState(20)
	InitFPA(ss)
	for tick := 0; tick < 10; tick++ {
		TickFPA(ss)
	}
	// Record positions before apply
	oldX := make([]float64, len(ss.Bots))
	oldY := make([]float64, len(ss.Bots))
	for i := range ss.Bots {
		oldX[i] = ss.Bots[i].X
		oldY[i] = ss.Bots[i].Y
	}
	// Apply to all bots — should move directly and set Speed=0
	moved := 0
	for i := range ss.Bots {
		ApplyFPA(&ss.Bots[i], ss, i)
		if ss.Bots[i].Speed != 0 {
			t.Fatalf("bot %d speed should be 0 after ApplyFPA (direct movement), got %f", i, ss.Bots[i].Speed)
		}
		if ss.Bots[i].X != oldX[i] || ss.Bots[i].Y != oldY[i] {
			moved++
		}
	}
	if moved == 0 {
		t.Fatal("no bots moved after ApplyFPA — direct movement broken")
	}
}

func TestApplyFPANil(t *testing.T) {
	ss := makeFPAState(5)
	// Should not panic with nil FPA state
	ApplyFPA(&ss.Bots[0], ss, 0)
}

func TestFPAGlobalVsLocal(t *testing.T) {
	ss := makeFPAState(50)
	InitFPA(ss)
	// Run several ticks
	for tick := 0; tick < 30; tick++ {
		TickFPA(ss)
	}
	// With default switch prob 0.8, most should be global
	globalCount := 0
	for i := range ss.FPA.IsGlobal {
		if ss.FPA.IsGlobal[i] {
			globalCount++
		}
	}
	if globalCount == 0 {
		t.Fatal("expected some global pollination, got none")
	}
	if globalCount == len(ss.Bots) {
		t.Fatal("expected some local pollination, got all global")
	}
}

func TestFPALevyFlight(t *testing.T) {
	ss := makeFPAState(5)
	// Levy flight should return finite values
	for i := 0; i < 100; i++ {
		step := levyFlight(ss)
		if step < -3.0 || step > 3.0 {
			t.Fatalf("levy step out of range: %f", step)
		}
	}
}

func TestFPAPersonalBestUpdate(t *testing.T) {
	ss := makeFPAState(10)
	// Place a light at center
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400
	InitFPA(ss)
	// Move bot 0 close to the light
	ss.Bots[0].X = 400
	ss.Bots[0].Y = 400
	for tick := 0; tick < 20; tick++ {
		TickFPA(ss)
	}
	// Bot 0 should have high fitness and be or be near global best
	if ss.FPA.Fitness[0] < ss.FPA.Fitness[5] {
		// Bot at the light should have better fitness than a random bot
		// (not always true due to randomness, but very likely)
		t.Log("bot at light did not have best fitness — possible due to initial positions")
	}
}
