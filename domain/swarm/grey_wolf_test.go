package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeGWOState(n int) *SwarmState {
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
	// Rebuild spatial hash
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}
	return ss
}

func TestInitGWO(t *testing.T) {
	ss := makeGWOState(20)
	InitGWO(ss)
	if ss.GWO == nil {
		t.Fatal("GWO state should not be nil after init")
	}
	if !ss.GWOOn {
		t.Fatal("GWOOn should be true after init")
	}
	if len(ss.GWO.Rank) != 20 {
		t.Fatalf("expected 20 ranks, got %d", len(ss.GWO.Rank))
	}
	// All should start as omega (rank 3)
	for i, r := range ss.GWO.Rank {
		if r != 3 {
			t.Fatalf("bot %d: expected rank 3 (omega), got %d", i, r)
		}
	}
}

func TestTickGWO(t *testing.T) {
	ss := makeGWOState(20)
	InitGWO(ss)
	// Run several ticks
	for tick := 0; tick < 50; tick++ {
		TickGWO(ss)
	}
	// After ticking, alpha/beta/delta should be assigned
	st := ss.GWO
	if st.AlphaIdx < 0 || st.AlphaIdx >= 20 {
		t.Fatalf("alpha index out of range: %d", st.AlphaIdx)
	}
	if st.Rank[st.AlphaIdx] != 0 {
		t.Fatalf("alpha bot should have rank 0, got %d", st.Rank[st.AlphaIdx])
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].GWORank < 0 || ss.Bots[i].GWORank > 3 {
			t.Fatalf("bot %d: GWORank out of range: %d", i, ss.Bots[i].GWORank)
		}
	}
}

func TestTickGWONil(t *testing.T) {
	ss := makeGWOState(10)
	// Should not panic when GWO is nil
	TickGWO(ss)
}

func TestClearGWO(t *testing.T) {
	ss := makeGWOState(10)
	InitGWO(ss)
	ClearGWO(ss)
	if ss.GWO != nil {
		t.Fatal("GWO should be nil after clear")
	}
	if ss.GWOOn {
		t.Fatal("GWOOn should be false after clear")
	}
}

func TestApplyGWO(t *testing.T) {
	ss := makeGWOState(20)
	InitGWO(ss)
	for tick := 0; tick < 10; tick++ {
		TickGWO(ss)
	}
	// Apply to omega wolf — with direct movement, speed is set to 0
	// after position update, so check that position changed instead.
	for i := range ss.Bots {
		if ss.GWO.Rank[i] == 3 { // omega
			oldX := ss.Bots[i].X
			oldY := ss.Bots[i].Y
			ApplyGWO(&ss.Bots[i], ss, i)
			// Position should have changed (direct movement)
			if ss.Bots[i].X == oldX && ss.Bots[i].Y == oldY {
				t.Fatal("omega wolf should have moved after ApplyGWO")
			}
			// Speed should be 0 (prevents double movement in GUI)
			if ss.Bots[i].Speed != 0 {
				t.Fatalf("omega wolf speed should be 0 after direct move, got %f", ss.Bots[i].Speed)
			}
			break
		}
	}
}

func TestGWOGrowSlices(t *testing.T) {
	ss := makeGWOState(5)
	InitGWO(ss)
	// Add bots
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	// Should not panic
	TickGWO(ss)
	if len(ss.GWO.Rank) != 10 {
		t.Fatalf("expected 10 ranks after grow, got %d", len(ss.GWO.Rank))
	}
}
