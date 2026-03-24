package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeCuckooState(n int) *SwarmState {
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

func TestInitCuckoo(t *testing.T) {
	ss := makeCuckooState(20)
	InitCuckoo(ss)
	if ss.Cuckoo == nil {
		t.Fatal("Cuckoo state should not be nil after init")
	}
	if !ss.CuckooOn {
		t.Fatal("CuckooOn should be true after init")
	}
	if len(ss.Cuckoo.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.Cuckoo.Fitness))
	}
	if len(ss.Cuckoo.NestAge) != 20 {
		t.Fatalf("expected 20 nest ages, got %d", len(ss.Cuckoo.NestAge))
	}
}

func TestTickCuckoo(t *testing.T) {
	ss := makeCuckooState(20)
	InitCuckoo(ss)
	for tick := 0; tick < 50; tick++ {
		TickCuckoo(ss)
	}
	st := ss.Cuckoo
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].CuckooFitness < 0 || ss.Bots[i].CuckooFitness > 100 {
			t.Fatalf("bot %d: CuckooFitness out of range: %d", i, ss.Bots[i].CuckooFitness)
		}
		if ss.Bots[i].CuckooNestAge < 0 || ss.Bots[i].CuckooNestAge > 100 {
			t.Fatalf("bot %d: CuckooNestAge out of range: %d", i, ss.Bots[i].CuckooNestAge)
		}
		if ss.Bots[i].CuckooBest != 0 && ss.Bots[i].CuckooBest != 1 {
			t.Fatalf("bot %d: CuckooBest should be 0 or 1, got %d", i, ss.Bots[i].CuckooBest)
		}
	}
	// At least some bots should be marked as best
	bestCount := 0
	for i := range ss.Bots {
		if ss.Bots[i].CuckooBest == 1 {
			bestCount++
		}
	}
	if bestCount == 0 {
		t.Fatal("expected some bots to be marked as best")
	}
}

func TestTickCuckooNil(t *testing.T) {
	ss := makeCuckooState(10)
	TickCuckoo(ss) // should not panic with nil state
}

func TestClearCuckoo(t *testing.T) {
	ss := makeCuckooState(10)
	InitCuckoo(ss)
	ClearCuckoo(ss)
	if ss.Cuckoo != nil {
		t.Fatal("Cuckoo should be nil after clear")
	}
	if ss.CuckooOn {
		t.Fatal("CuckooOn should be false after clear")
	}
}

func TestApplyCuckoo(t *testing.T) {
	ss := makeCuckooState(20)
	InitCuckoo(ss)
	for tick := 0; tick < 10; tick++ {
		TickCuckoo(ss)
	}
	// Record initial positions
	initX := make([]float64, len(ss.Bots))
	for i := range ss.Bots {
		initX[i] = ss.Bots[i].X
	}
	for i := range ss.Bots {
		ApplyCuckoo(&ss.Bots[i], ss, i)
		// Speed should be 0 after eigenbewegung
		if ss.Bots[i].Speed != 0 {
			t.Fatalf("bot %d: speed should be 0 after eigenbewegung", i)
		}
	}
	// At least some bots should have moved
	moved := 0
	for i := range ss.Bots {
		if ss.Bots[i].X != initX[i] {
			moved++
		}
	}
	if moved == 0 {
		t.Fatal("expected some bots to move via eigenbewegung")
	}
}

func TestApplyCuckooNil(t *testing.T) {
	ss := makeCuckooState(5)
	bot := &ss.Bots[0]
	ApplyCuckoo(bot, ss, 0) // should not panic with nil state
	if bot.Speed != 0 {
		t.Fatalf("expected Speed=0 with nil cuckoo, got %f", bot.Speed)
	}
}

func TestCuckooGrowSlices(t *testing.T) {
	ss := makeCuckooState(5)
	InitCuckoo(ss)
	ss.Bots = append(ss.Bots, SwarmBot{X: 100, Y: 100, Energy: 50, CarryingPkg: -1})
	TickCuckoo(ss) // should not panic
	if len(ss.Cuckoo.Fitness) != 6 {
		t.Fatalf("expected 6 fitness entries, got %d", len(ss.Cuckoo.Fitness))
	}
}

func TestCuckooNestAbandonment(t *testing.T) {
	ss := makeCuckooState(20)
	InitCuckoo(ss)
	// Record initial positions
	initialX := make([]float64, len(ss.Bots))
	for i := range ss.Bots {
		initialX[i] = ss.Bots[i].X
	}
	// Run enough ticks for abandonment to happen
	for tick := 0; tick < 100; tick++ {
		TickCuckoo(ss)
	}
	// Some bots should have moved (abandoned nests reset to random positions)
	movedCount := 0
	for i := range ss.Bots {
		if ss.Bots[i].X != initialX[i] {
			movedCount++
		}
	}
	if movedCount == 0 {
		t.Fatal("expected at least some nests to be abandoned and moved")
	}
}

func TestCuckooBestLED(t *testing.T) {
	ss := makeCuckooState(10)
	InitCuckoo(ss)
	for tick := 0; tick < 10; tick++ {
		TickCuckoo(ss)
	}
	bestIdx := ss.Cuckoo.BestIdx
	ApplyCuckoo(&ss.Bots[bestIdx], ss, bestIdx)
	// Best nest should have gold LED
	led := ss.Bots[bestIdx].LEDColor
	if led[0] != 255 || led[1] != 215 || led[2] != 0 {
		t.Fatalf("best nest should have gold LED, got (%d,%d,%d)", led[0], led[1], led[2])
	}
}
