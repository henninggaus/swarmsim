package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeSSAState(n int) *SwarmState {
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

func TestInitSSA(t *testing.T) {
	ss := makeSSAState(20)
	InitSSA(ss)
	if ss.SSA == nil {
		t.Fatal("SSA state should not be nil after init")
	}
	if !ss.SSAOn {
		t.Fatal("SSAOn should be true after init")
	}
	if len(ss.SSA.Role) != 20 {
		t.Fatalf("expected 20 roles, got %d", len(ss.SSA.Role))
	}
	// First half should be leaders (0), second half followers (1)
	for i := 0; i < 10; i++ {
		if ss.SSA.Role[i] != 0 {
			t.Fatalf("bot %d: expected role 0 (leader), got %d", i, ss.SSA.Role[i])
		}
	}
	for i := 10; i < 20; i++ {
		if ss.SSA.Role[i] != 1 {
			t.Fatalf("bot %d: expected role 1 (follower), got %d", i, ss.SSA.Role[i])
		}
	}
}

func TestTickSSA(t *testing.T) {
	ss := makeSSAState(20)
	InitSSA(ss)
	for tick := 0; tick < 50; tick++ {
		TickSSA(ss)
	}
	st := ss.SSA
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].SSARole < 0 || ss.Bots[i].SSARole > 1 {
			t.Fatalf("bot %d: SSARole out of range: %d", i, ss.Bots[i].SSARole)
		}
	}
}

func TestTickSSANil(t *testing.T) {
	ss := makeSSAState(10)
	// Should not panic when SSA is nil
	TickSSA(ss)
}

func TestClearSSA(t *testing.T) {
	ss := makeSSAState(10)
	InitSSA(ss)
	ClearSSA(ss)
	if ss.SSA != nil {
		t.Fatal("SSA should be nil after clear")
	}
	if ss.SSAOn {
		t.Fatal("SSAOn should be false after clear")
	}
}

func TestApplySSA(t *testing.T) {
	ss := makeSSAState(20)
	InitSSA(ss)
	for tick := 0; tick < 10; tick++ {
		TickSSA(ss)
	}
	// Apply to a leader — should move via position update, Speed=0
	for i := range ss.Bots {
		if ss.SSA.Role[i] == 0 {
			oldX, oldY := ss.Bots[i].X, ss.Bots[i].Y
			ApplySSA(&ss.Bots[i], ss, i)
			if ss.Bots[i].Speed != 0 {
				t.Fatal("leader speed should be 0 (direct movement)")
			}
			if ss.Bots[i].X == oldX && ss.Bots[i].Y == oldY {
				t.Fatal("leader should have moved")
			}
			break
		}
	}
	// Apply to a follower — should move via position update, Speed=0
	for i := range ss.Bots {
		if ss.SSA.Role[i] == 1 {
			oldX, oldY := ss.Bots[i].X, ss.Bots[i].Y
			ApplySSA(&ss.Bots[i], ss, i)
			if ss.Bots[i].Speed != 0 {
				t.Fatal("follower speed should be 0 (direct movement)")
			}
			dx := ss.Bots[i].X - oldX
			dy := ss.Bots[i].Y - oldY
			if dx == 0 && dy == 0 {
				t.Fatal("follower should have moved")
			}
			break
		}
	}
}

func TestSSAGlobalBest(t *testing.T) {
	ss := makeSSAState(20)
	InitSSA(ss)
	for tick := 0; tick < 100; tick++ {
		TickSSA(ss)
	}
	st := ss.SSA
	if st.FoodFit <= -1e9 {
		t.Fatal("FoodFit should have been updated from initial -1e9")
	}
	if st.FoodX == 0 && st.FoodY == 0 {
		t.Fatal("FoodX/FoodY should have been set")
	}
}

func TestSSAGrowSlices(t *testing.T) {
	ss := makeSSAState(5)
	InitSSA(ss)
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	TickSSA(ss)
	if len(ss.SSA.Fitness) != 10 {
		t.Fatalf("expected 10 fitness entries after grow, got %d", len(ss.SSA.Fitness))
	}
}
