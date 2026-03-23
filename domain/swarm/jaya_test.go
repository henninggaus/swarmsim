package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeJayaState(n int) *SwarmState {
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

func TestInitJaya(t *testing.T) {
	ss := makeJayaState(20)
	InitJaya(ss)
	if ss.Jaya == nil {
		t.Fatal("Jaya state should not be nil after init")
	}
	if !ss.JayaOn {
		t.Fatal("JayaOn should be true after init")
	}
	if len(ss.Jaya.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.Jaya.Fitness))
	}
	if len(ss.Jaya.PersonalBestX) != 20 {
		t.Fatalf("expected 20 personal best X entries, got %d", len(ss.Jaya.PersonalBestX))
	}
	if len(ss.Jaya.PersonalBestY) != 20 {
		t.Fatalf("expected 20 personal best Y entries, got %d", len(ss.Jaya.PersonalBestY))
	}
	if len(ss.Jaya.PersonalBestF) != 20 {
		t.Fatalf("expected 20 personal best F entries, got %d", len(ss.Jaya.PersonalBestF))
	}
}

func TestClearJaya(t *testing.T) {
	ss := makeJayaState(10)
	InitJaya(ss)
	ClearJaya(ss)
	if ss.Jaya != nil {
		t.Fatal("Jaya should be nil after clear")
	}
	if ss.JayaOn {
		t.Fatal("JayaOn should be false after clear")
	}
}

func TestTickJayaNil(t *testing.T) {
	ss := makeJayaState(10)
	// Should not panic when Jaya is nil
	TickJaya(ss)
}

func TestTickJaya(t *testing.T) {
	ss := makeJayaState(20)
	InitJaya(ss)
	for tick := 0; tick < 50; tick++ {
		TickJaya(ss)
	}
	st := ss.Jaya
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	if st.WorstIdx < 0 || st.WorstIdx >= 20 {
		t.Fatalf("worst index out of range: %d", st.WorstIdx)
	}
	if st.BestF < st.WorstF {
		t.Fatal("best fitness should be >= worst fitness")
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].JayaFitness < 0 || ss.Bots[i].JayaFitness > 100 {
			t.Fatalf("bot %d: JayaFitness out of range: %d", i, ss.Bots[i].JayaFitness)
		}
	}
}

func TestApplyJaya(t *testing.T) {
	ss := makeJayaState(20)
	InitJaya(ss)
	for tick := 0; tick < 10; tick++ {
		TickJaya(ss)
	}
	// Apply to a non-best bot
	for i := range ss.Bots {
		if i != ss.Jaya.BestIdx {
			ApplyJaya(&ss.Bots[i], ss, i)
			if ss.Bots[i].Speed <= 0 {
				t.Fatal("bot speed should be positive after apply")
			}
			break
		}
	}
}

func TestApplyJayaBestBot(t *testing.T) {
	ss := makeJayaState(20)
	InitJaya(ss)
	for tick := 0; tick < 10; tick++ {
		TickJaya(ss)
	}
	bestIdx := ss.Jaya.BestIdx
	if bestIdx >= 0 {
		ApplyJaya(&ss.Bots[bestIdx], ss, bestIdx)
		// Best bot should get gold LED
		if ss.Bots[bestIdx].LEDColor != [3]uint8{255, 215, 0} {
			t.Fatal("best bot should have gold LED")
		}
	}
}

func TestJayaGrowSlices(t *testing.T) {
	ss := makeJayaState(5)
	InitJaya(ss)
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	TickJaya(ss)
	if len(ss.Jaya.Fitness) != 10 {
		t.Fatalf("expected 10 fitness after grow, got %d", len(ss.Jaya.Fitness))
	}
	if len(ss.Jaya.PersonalBestX) != 10 {
		t.Fatalf("expected 10 personal best X after grow, got %d", len(ss.Jaya.PersonalBestX))
	}
}

func TestJayaCycleReset(t *testing.T) {
	ss := makeJayaState(10)
	InitJaya(ss)
	for tick := 0; tick < jayaMaxTicks+10; tick++ {
		TickJaya(ss)
	}
	if ss.Jaya.Tick > jayaMaxTicks {
		t.Fatalf("tick should wrap around, got %d", ss.Jaya.Tick)
	}
}

func TestApplyJayaNil(t *testing.T) {
	ss := makeJayaState(10)
	bot := &ss.Bots[0]
	// Should not panic with nil Jaya
	ApplyJaya(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("speed should be default when Jaya is nil")
	}
}

func TestJayaPersonalBestUpdates(t *testing.T) {
	ss := makeJayaState(10)
	InitJaya(ss)
	// Run a few ticks to build up personal bests
	for tick := 0; tick < 20; tick++ {
		TickJaya(ss)
	}
	// All personal best fitnesses should be >= 0 (updated from -1e18)
	for i := 0; i < len(ss.Bots); i++ {
		if ss.Jaya.PersonalBestF[i] < 0 {
			t.Fatalf("bot %d: personal best fitness should be >= 0, got %f", i, ss.Jaya.PersonalBestF[i])
		}
	}
}

func TestJayaBestWorstDifferent(t *testing.T) {
	ss := makeJayaState(20)
	InitJaya(ss)
	for tick := 0; tick < 10; tick++ {
		TickJaya(ss)
	}
	// With 20 bots at random positions, best and worst should differ
	if ss.Jaya.BestIdx == ss.Jaya.WorstIdx && ss.Jaya.BestF != ss.Jaya.WorstF {
		t.Fatal("best and worst should not be same bot with different fitness")
	}
}

func TestJayaLEDColor(t *testing.T) {
	ss := makeJayaState(20)
	InitJaya(ss)
	for tick := 0; tick < 10; tick++ {
		TickJaya(ss)
	}
	// Apply to all non-best bots and check LED is set
	for i := range ss.Bots {
		if i != ss.Jaya.BestIdx {
			ApplyJaya(&ss.Bots[i], ss, i)
			led := ss.Bots[i].LEDColor
			// LED should not be all zeros (black)
			if led[0] == 0 && led[1] == 0 && led[2] == 0 {
				t.Fatalf("bot %d: LED should not be black after apply", i)
			}
		}
	}
}
