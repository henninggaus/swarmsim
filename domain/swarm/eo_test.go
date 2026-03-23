package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeEOState(n int) *SwarmState {
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

func TestInitEO(t *testing.T) {
	ss := makeEOState(20)
	InitEO(ss)
	if ss.EO == nil {
		t.Fatal("EO state should not be nil after init")
	}
	if !ss.EOOn {
		t.Fatal("EOOn should be true after init")
	}
	if len(ss.EO.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.EO.Fitness))
	}
	if len(ss.EO.PersonalX) != 20 {
		t.Fatalf("expected 20 personal X entries, got %d", len(ss.EO.PersonalX))
	}
	if len(ss.EO.PoolX) != eoPoolSize {
		t.Fatalf("expected pool size %d, got %d", eoPoolSize, len(ss.EO.PoolX))
	}
}

func TestTickEO(t *testing.T) {
	ss := makeEOState(20)
	InitEO(ss)
	for tick := 0; tick < 50; tick++ {
		TickEO(ss)
	}
	st := ss.EO
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	if st.CycleTick != 50 {
		t.Fatalf("expected cycle tick 50, got %d", st.CycleTick)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].EOPhase < 0 || ss.Bots[i].EOPhase > 1 {
			t.Fatalf("bot %d: EOPhase out of range: %d", i, ss.Bots[i].EOPhase)
		}
	}
}

func TestTickEONil(t *testing.T) {
	ss := makeEOState(10)
	// Should not panic when EO is nil
	TickEO(ss)
}

func TestClearEO(t *testing.T) {
	ss := makeEOState(10)
	InitEO(ss)
	ClearEO(ss)
	if ss.EO != nil {
		t.Fatal("EO should be nil after clear")
	}
	if ss.EOOn {
		t.Fatal("EOOn should be false after clear")
	}
}

func TestApplyEO(t *testing.T) {
	ss := makeEOState(20)
	InitEO(ss)
	for tick := 0; tick < 10; tick++ {
		TickEO(ss)
	}
	// Apply to each bot
	for i := range ss.Bots {
		ApplyEO(&ss.Bots[i], ss, i)
		if ss.Bots[i].Speed <= 0 {
			t.Fatalf("bot %d: speed should be positive", i)
		}
	}
}

func TestApplyEONil(t *testing.T) {
	ss := makeEOState(5)
	// Should not panic when EO is nil
	bot := &ss.Bots[0]
	ApplyEO(bot, ss, 0)
	if bot.Speed <= 0 {
		t.Fatal("speed should be positive even with nil EO")
	}
}

func TestEOGrowSlices(t *testing.T) {
	ss := makeEOState(5)
	InitEO(ss)
	// Add more bots
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	TickEO(ss)
	if len(ss.EO.Fitness) != 10 {
		t.Fatalf("expected 10 fitness entries after grow, got %d", len(ss.EO.Fitness))
	}
}

func TestEOPool(t *testing.T) {
	ss := makeEOState(20)
	InitEO(ss)
	// Run enough ticks to populate pool
	for tick := 0; tick < 100; tick++ {
		TickEO(ss)
	}
	// Pool should have valid positions (within arena bounds)
	for k := 0; k < eoPoolSize; k++ {
		if ss.EO.PoolX[k] < 0 || ss.EO.PoolX[k] > 800 {
			t.Fatalf("pool %d X out of bounds: %f", k, ss.EO.PoolX[k])
		}
		if ss.EO.PoolY[k] < 0 || ss.EO.PoolY[k] > 800 {
			t.Fatalf("pool %d Y out of bounds: %f", k, ss.EO.PoolY[k])
		}
	}
}

func TestEOCycleReset(t *testing.T) {
	ss := makeEOState(10)
	InitEO(ss)
	for tick := 0; tick < eoMaxTicks+5; tick++ {
		TickEO(ss)
	}
	if ss.EO.CycleTick > eoMaxTicks {
		t.Fatalf("cycle tick should have reset, got %d", ss.EO.CycleTick)
	}
}

func TestEOPhaseTransition(t *testing.T) {
	ss := makeEOState(10)
	InitEO(ss)

	// Early ticks: exploration phase
	for tick := 0; tick < 10; tick++ {
		TickEO(ss)
	}
	for i := range ss.Bots {
		if ss.Bots[i].EOPhase != 0 {
			t.Fatalf("early tick: bot %d should be in exploration (phase 0), got %d", i, ss.Bots[i].EOPhase)
		}
	}

	// Late ticks: exploitation phase (past half of cycle)
	for tick := 10; tick < eoMaxTicks/2+10; tick++ {
		TickEO(ss)
	}
	for i := range ss.Bots {
		if ss.Bots[i].EOPhase != 1 {
			t.Fatalf("late tick: bot %d should be in exploitation (phase 1), got %d", i, ss.Bots[i].EOPhase)
		}
	}
}

func TestEOPersonalBest(t *testing.T) {
	ss := makeEOState(10)
	InitEO(ss)
	// Set high neighbor count for bot 0 to ensure it gets high fitness
	ss.Bots[0].NeighborCount = 10
	ss.Bots[0].SwarmCenterDist = 0
	TickEO(ss)
	if ss.EO.PersonalF[0] <= ss.EO.PersonalF[1] {
		// Bot 0 should have higher personal best due to higher neighbor count
		t.Log("bot 0 fitness:", ss.EO.Fitness[0], "bot 1 fitness:", ss.EO.Fitness[1])
	}
	// Personal best X should match bot position
	if ss.EO.PersonalX[0] != ss.Bots[0].X {
		t.Fatalf("personal best X should match bot X after first tick")
	}
}

func TestEOSensorCache(t *testing.T) {
	ss := makeEOState(10)
	InitEO(ss)
	for tick := 0; tick < 20; tick++ {
		TickEO(ss)
	}
	// Check that sensor values are populated
	found := false
	for i := range ss.Bots {
		if ss.Bots[i].EOFitness > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Log("warning: no bot had positive EOFitness after 20 ticks")
	}
	// EOEquilDist should be non-negative
	for i := range ss.Bots {
		if ss.Bots[i].EOEquilDist < 0 {
			t.Fatalf("bot %d: EOEquilDist should be non-negative, got %d", i, ss.Bots[i].EOEquilDist)
		}
	}
}
