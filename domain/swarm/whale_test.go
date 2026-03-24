package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeWOAState(n int) *SwarmState {
	ss := &SwarmState{
		Bots:   make([]SwarmBot, n),
		ArenaW: 800,
		ArenaH: 800,
		Rng:    rand.New(rand.NewSource(99)),
		Hash:   physics.NewSpatialHash(800, 800, 30),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = ss.Rng.Float64() * 800
		ss.Bots[i].Y = ss.Rng.Float64() * 800
		ss.Bots[i].Angle = ss.Rng.Float64() * 6.28
		ss.Bots[i].Energy = 70
		ss.Bots[i].CarryingPkg = -1
	}
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}
	return ss
}

func TestInitWOA(t *testing.T) {
	ss := makeWOAState(15)
	InitWOA(ss)
	if ss.WOA == nil {
		t.Fatal("WOA state should not be nil after init")
	}
	if !ss.WOAOn {
		t.Fatal("WOAOn should be true after init")
	}
	if len(ss.WOA.Fitness) != 15 {
		t.Fatalf("expected 15 fitness entries, got %d", len(ss.WOA.Fitness))
	}
}

func TestTickWOA(t *testing.T) {
	ss := makeWOAState(15)
	InitWOA(ss)
	for tick := 0; tick < 30; tick++ {
		TickWOA(ss)
	}
	st := ss.WOA
	if st.BestIdx < 0 || st.BestIdx >= 15 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	// Phases should be assigned
	for i := range ss.Bots {
		p := ss.Bots[i].WOAPhase
		if p < 0 || p > 2 {
			t.Fatalf("bot %d: WOAPhase out of range: %d", i, p)
		}
	}
}

func TestTickWOANil(t *testing.T) {
	ss := makeWOAState(10)
	TickWOA(ss) // should not panic
}

func TestClearWOA(t *testing.T) {
	ss := makeWOAState(10)
	InitWOA(ss)
	ClearWOA(ss)
	if ss.WOA != nil {
		t.Fatal("WOA should be nil after clear")
	}
	if ss.WOAOn {
		t.Fatal("WOAOn should be false after clear")
	}
}

func TestApplyWOA(t *testing.T) {
	ss := makeWOAState(15)
	InitWOA(ss)
	for tick := 0; tick < 10; tick++ {
		TickWOA(ss)
	}
	initX := make([]float64, len(ss.Bots))
	for i := range ss.Bots {
		initX[i] = ss.Bots[i].X
	}
	for i := range ss.Bots {
		ApplyWOA(&ss.Bots[i], ss, i)
		if ss.Bots[i].Speed != 0 {
			t.Fatalf("bot %d: speed should be 0 after eigenbewegung", i)
		}
	}
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

func TestWOAGlobalBest(t *testing.T) {
	ss := makeWOAState(15)
	InitWOA(ss)
	for tick := 0; tick < 50; tick++ {
		TickWOA(ss)
		for i := range ss.Bots {
			ApplyWOA(&ss.Bots[i], ss, i)
		}
	}
	if ss.WOA.GlobalBestF <= -1e18 {
		t.Fatal("GlobalBestF should be updated after 50 ticks")
	}
}

func TestWOAGrowSlices(t *testing.T) {
	ss := makeWOAState(5)
	InitWOA(ss)
	ss.Bots = append(ss.Bots, SwarmBot{X: 100, Y: 100, Energy: 50, CarryingPkg: -1})
	TickWOA(ss) // should not panic
	if len(ss.WOA.Phase) != 6 {
		t.Fatalf("expected 6 phases, got %d", len(ss.WOA.Phase))
	}
}
