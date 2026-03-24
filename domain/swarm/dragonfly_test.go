package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeDAState(n int) *SwarmState {
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

func TestInitDA(t *testing.T) {
	ss := makeDAState(20)
	InitDA(ss)
	if ss.DA == nil {
		t.Fatal("DA state should not be nil after init")
	}
	if !ss.DAOn {
		t.Fatal("DAOn should be true after init")
	}
	if len(ss.DA.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.DA.Fitness))
	}
	if len(ss.DA.StepX) != 20 {
		t.Fatalf("expected 20 step entries, got %d", len(ss.DA.StepX))
	}
	if len(ss.DA.Role) != 20 {
		t.Fatalf("expected 20 role entries, got %d", len(ss.DA.Role))
	}
}

func TestClearDA(t *testing.T) {
	ss := makeDAState(10)
	InitDA(ss)
	ClearDA(ss)
	if ss.DA != nil {
		t.Fatal("DA should be nil after clear")
	}
	if ss.DAOn {
		t.Fatal("DAOn should be false after clear")
	}
}

func TestTickDANil(t *testing.T) {
	ss := makeDAState(10)
	// Should not panic when DA is nil
	TickDA(ss)
}

func TestTickDA(t *testing.T) {
	ss := makeDAState(20)
	InitDA(ss)
	for tick := 0; tick < 50; tick++ {
		TickDA(ss)
	}
	st := ss.DA
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	if st.WorstIdx < 0 || st.WorstIdx >= 20 {
		t.Fatalf("worst index out of range: %d", st.WorstIdx)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].DAFitness < 0 || ss.Bots[i].DAFitness > 100 {
			t.Fatalf("bot %d: DAFitness out of range: %d", i, ss.Bots[i].DAFitness)
		}
		if ss.Bots[i].DARole < 0 || ss.Bots[i].DARole > 2 {
			t.Fatalf("bot %d: DARole out of range: %d", i, ss.Bots[i].DARole)
		}
	}
}

func TestApplyDA(t *testing.T) {
	ss := makeDAState(20)
	InitDA(ss)
	for tick := 0; tick < 10; tick++ {
		TickDA(ss)
	}
	// Apply to a non-best bot — position should change (Eigenbewegung)
	for i := range ss.Bots {
		if i != ss.DA.CurBestIdx {
			oldX := ss.Bots[i].X
			oldY := ss.Bots[i].Y
			ApplyDA(&ss.Bots[i], ss, i)
			if ss.Bots[i].X == oldX && ss.Bots[i].Y == oldY {
				t.Fatal("bot position should change after apply (Eigenbewegung)")
			}
			break
		}
	}
}

func TestApplyDABestBot(t *testing.T) {
	ss := makeDAState(20)
	InitDA(ss)
	for tick := 0; tick < 10; tick++ {
		TickDA(ss)
	}
	bestIdx := ss.DA.CurBestIdx
	if bestIdx >= 0 {
		ApplyDA(&ss.Bots[bestIdx], ss, bestIdx)
		// Best bot should get gold LED
		if ss.Bots[bestIdx].LEDColor != [3]uint8{255, 215, 0} {
			t.Fatal("best bot should have gold LED")
		}
	}
}

func TestDAGrowSlices(t *testing.T) {
	ss := makeDAState(5)
	InitDA(ss)
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	TickDA(ss)
	if len(ss.DA.Fitness) != 10 {
		t.Fatalf("expected 10 fitness after grow, got %d", len(ss.DA.Fitness))
	}
	if len(ss.DA.StepX) != 10 {
		t.Fatalf("expected 10 steps after grow, got %d", len(ss.DA.StepX))
	}
}

func TestDACycleReset(t *testing.T) {
	ss := makeDAState(10)
	InitDA(ss)
	for tick := 0; tick < daMaxTicks+10; tick++ {
		TickDA(ss)
	}
	if ss.DA.Tick > daMaxTicks {
		t.Fatalf("tick should wrap around, got %d", ss.DA.Tick)
	}
}

func TestApplyDANil(t *testing.T) {
	ss := makeDAState(10)
	bot := &ss.Bots[0]
	// Should not panic with nil DA
	ApplyDA(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("speed should be default when DA is nil")
	}
}

func TestDARoleDistribution(t *testing.T) {
	ss := makeDAState(50)
	InitDA(ss)
	// Run enough ticks to get past midpoint (roles should transition)
	for tick := 0; tick < 350; tick++ {
		TickDA(ss)
	}
	// Apply to all non-best bots and check roles are assigned
	roles := map[int]int{}
	for i := range ss.Bots {
		if i != ss.DA.CurBestIdx {
			ApplyDA(&ss.Bots[i], ss, i)
			roles[ss.DA.Role[i]]++
		}
	}
	// After midpoint, static (0) or lévy (2) roles should dominate
	if len(roles) == 0 {
		t.Fatal("expected some role assignments")
	}
}

func TestDALevyFlight(t *testing.T) {
	// Create a single isolated bot far from others to trigger Lévy flight
	ss := makeDAState(2)
	ss.Bots[0].X = 50
	ss.Bots[0].Y = 50
	ss.Bots[1].X = 750 // far away, outside neighbour radius
	ss.Bots[1].Y = 750
	ss.Hash = physics.NewSpatialHash(800, 800, 30)
	ss.Hash.Insert(0, ss.Bots[0].X, ss.Bots[0].Y)
	ss.Hash.Insert(1, ss.Bots[1].X, ss.Bots[1].Y)

	InitDA(ss)
	for tick := 0; tick < 10; tick++ {
		TickDA(ss)
	}

	// Apply to bot 0 — it should have no neighbours and use Lévy flight
	ApplyDA(&ss.Bots[0], ss, 0)
	// Lévy flight should set role to 2
	if ss.DA.Role[0] != 2 && ss.DA.CurBestIdx != 0 {
		// Only check if bot 0 is not the best (best keeps role 0)
		t.Fatalf("isolated bot should be in lévy flight mode, got role %d", ss.DA.Role[0])
	}
}

func TestDAWorstTracking(t *testing.T) {
	ss := makeDAState(20)
	InitDA(ss)
	for tick := 0; tick < 30; tick++ {
		TickDA(ss)
	}
	st := ss.DA
	// Worst should track the lowest-fitness bot
	if st.WorstIdx < 0 || st.WorstIdx >= 20 {
		t.Fatalf("worst index out of range: %d", st.WorstIdx)
	}
	for i := range ss.Bots {
		if st.Fitness[i] < st.Fitness[st.WorstIdx] {
			t.Fatalf("bot %d has lower fitness than worst bot %d", i, st.WorstIdx)
		}
	}
}
