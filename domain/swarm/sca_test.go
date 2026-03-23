package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeSCAState(n int) *SwarmState {
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

func TestInitSCA(t *testing.T) {
	ss := makeSCAState(20)
	InitSCA(ss)
	if ss.SCA == nil {
		t.Fatal("SCA state should not be nil after init")
	}
	if !ss.SCAOn {
		t.Fatal("SCAOn should be true after init")
	}
	if len(ss.SCA.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.SCA.Fitness))
	}
	if len(ss.SCA.Phase) != 20 {
		t.Fatalf("expected 20 phase entries, got %d", len(ss.SCA.Phase))
	}
}

func TestClearSCA(t *testing.T) {
	ss := makeSCAState(10)
	InitSCA(ss)
	ClearSCA(ss)
	if ss.SCA != nil {
		t.Fatal("SCA should be nil after clear")
	}
	if ss.SCAOn {
		t.Fatal("SCAOn should be false after clear")
	}
}

func TestTickSCANil(t *testing.T) {
	ss := makeSCAState(10)
	// Should not panic when SCA is nil
	TickSCA(ss)
}

func TestTickSCA(t *testing.T) {
	ss := makeSCAState(20)
	InitSCA(ss)
	for tick := 0; tick < 50; tick++ {
		TickSCA(ss)
	}
	st := ss.SCA
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].SCAFitness < 0 || ss.Bots[i].SCAFitness > 100 {
			t.Fatalf("bot %d: SCAFitness out of range: %d", i, ss.Bots[i].SCAFitness)
		}
		if ss.Bots[i].SCAPhase < 0 || ss.Bots[i].SCAPhase > 1 {
			t.Fatalf("bot %d: SCAPhase out of range: %d", i, ss.Bots[i].SCAPhase)
		}
	}
}

func TestApplySCA(t *testing.T) {
	ss := makeSCAState(20)
	InitSCA(ss)
	for tick := 0; tick < 10; tick++ {
		TickSCA(ss)
	}
	// Apply to a non-best bot
	for i := range ss.Bots {
		if i != ss.SCA.BestIdx {
			ApplySCA(&ss.Bots[i], ss, i)
			if ss.Bots[i].Speed <= 0 {
				t.Fatal("bot speed should be positive after apply")
			}
			break
		}
	}
}

func TestApplySCABestBot(t *testing.T) {
	ss := makeSCAState(20)
	InitSCA(ss)
	for tick := 0; tick < 10; tick++ {
		TickSCA(ss)
	}
	bestIdx := ss.SCA.BestIdx
	if bestIdx >= 0 {
		ApplySCA(&ss.Bots[bestIdx], ss, bestIdx)
		// Best bot should get gold LED
		if ss.Bots[bestIdx].LEDColor != [3]uint8{255, 215, 0} {
			t.Fatal("best bot should have gold LED")
		}
	}
}

func TestSCAGrowSlices(t *testing.T) {
	ss := makeSCAState(5)
	InitSCA(ss)
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	TickSCA(ss)
	if len(ss.SCA.Fitness) != 10 {
		t.Fatalf("expected 10 fitness after grow, got %d", len(ss.SCA.Fitness))
	}
	if len(ss.SCA.Phase) != 10 {
		t.Fatalf("expected 10 phases after grow, got %d", len(ss.SCA.Phase))
	}
}

func TestSCACycleReset(t *testing.T) {
	ss := makeSCAState(10)
	InitSCA(ss)
	// Run past cycle length
	for tick := 0; tick < scaMaxTicks+10; tick++ {
		TickSCA(ss)
	}
	if ss.SCA.Tick > scaMaxTicks {
		t.Fatalf("tick should wrap around, got %d", ss.SCA.Tick)
	}
}

func TestApplySCANil(t *testing.T) {
	ss := makeSCAState(10)
	bot := &ss.Bots[0]
	// Should not panic with nil SCA
	ApplySCA(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("speed should be default when SCA is nil")
	}
}

func TestSCAPhaseDistribution(t *testing.T) {
	ss := makeSCAState(100)
	InitSCA(ss)
	for tick := 0; tick < 50; tick++ {
		TickSCA(ss)
	}
	// Apply to all non-best bots and count sine vs cosine phases
	sine, cosine := 0, 0
	for i := range ss.Bots {
		if i != ss.SCA.BestIdx {
			ApplySCA(&ss.Bots[i], ss, i)
			if ss.SCA.Phase[i] == 0 {
				sine++
			} else {
				cosine++
			}
		}
	}
	// With 50/50 probability, both should have some representation
	if sine == 0 {
		t.Fatal("expected some bots in sine phase")
	}
	if cosine == 0 {
		t.Fatal("expected some bots in cosine phase")
	}
}
