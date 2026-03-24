package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeTLBOState(n int) *SwarmState {
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

func TestInitTLBO(t *testing.T) {
	ss := makeTLBOState(20)
	InitTLBO(ss)
	if ss.TLBO == nil {
		t.Fatal("TLBO state should not be nil after init")
	}
	if !ss.TLBOOn {
		t.Fatal("TLBOOn should be true after init")
	}
	if len(ss.TLBO.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.TLBO.Fitness))
	}
	if len(ss.TLBO.Phase) != 20 {
		t.Fatalf("expected 20 phase entries, got %d", len(ss.TLBO.Phase))
	}
	if len(ss.TLBO.PeerIdx) != 20 {
		t.Fatalf("expected 20 peer entries, got %d", len(ss.TLBO.PeerIdx))
	}
}

func TestClearTLBO(t *testing.T) {
	ss := makeTLBOState(10)
	InitTLBO(ss)
	ClearTLBO(ss)
	if ss.TLBO != nil {
		t.Fatal("TLBO should be nil after clear")
	}
	if ss.TLBOOn {
		t.Fatal("TLBOOn should be false after clear")
	}
}

func TestTickTLBONil(t *testing.T) {
	ss := makeTLBOState(10)
	// Should not panic when TLBO is nil
	TickTLBO(ss)
}

func TestTickTLBO(t *testing.T) {
	ss := makeTLBOState(20)
	InitTLBO(ss)
	for tick := 0; tick < 50; tick++ {
		TickTLBO(ss)
	}
	st := ss.TLBO
	if st.BestIdx < 0 || st.BestIdx >= 20 {
		t.Fatalf("best index out of range: %d", st.BestIdx)
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].TLBOFitness < 0 || ss.Bots[i].TLBOFitness > 100 {
			t.Fatalf("bot %d: TLBOFitness out of range: %d", i, ss.Bots[i].TLBOFitness)
		}
		if ss.Bots[i].TLBOPhase < 0 || ss.Bots[i].TLBOPhase > 1 {
			t.Fatalf("bot %d: TLBOPhase out of range: %d", i, ss.Bots[i].TLBOPhase)
		}
	}
}

func TestApplyTLBO(t *testing.T) {
	ss := makeTLBOState(20)
	InitTLBO(ss)
	for tick := 0; tick < 10; tick++ {
		TickTLBO(ss)
	}
	// Apply to a non-best bot — should move directly (Eigenbewegung)
	for i := range ss.Bots {
		if i != ss.TLBO.BestIdx {
			oldX := ss.Bots[i].X
			oldY := ss.Bots[i].Y
			ApplyTLBO(&ss.Bots[i], ss, i)
			if ss.Bots[i].Speed != 0 {
				t.Fatal("bot speed should be 0 after apply (direct movement)")
			}
			dx := ss.Bots[i].X - oldX
			dy := ss.Bots[i].Y - oldY
			if dx*dx+dy*dy < 0.001 {
				t.Fatal("bot should have moved via direct position update")
			}
			break
		}
	}
}

func TestApplyTLBOBestBot(t *testing.T) {
	ss := makeTLBOState(20)
	InitTLBO(ss)
	for tick := 0; tick < 10; tick++ {
		TickTLBO(ss)
	}
	bestIdx := ss.TLBO.BestIdx
	if bestIdx >= 0 {
		ApplyTLBO(&ss.Bots[bestIdx], ss, bestIdx)
		// Best bot (teacher) should get gold LED
		if ss.Bots[bestIdx].LEDColor != [3]uint8{255, 215, 0} {
			t.Fatal("teacher bot should have gold LED")
		}
	}
}

func TestTLBOGrowSlices(t *testing.T) {
	ss := makeTLBOState(5)
	InitTLBO(ss)
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	TickTLBO(ss)
	if len(ss.TLBO.Fitness) != 10 {
		t.Fatalf("expected 10 fitness after grow, got %d", len(ss.TLBO.Fitness))
	}
	if len(ss.TLBO.Phase) != 10 {
		t.Fatalf("expected 10 phases after grow, got %d", len(ss.TLBO.Phase))
	}
	if len(ss.TLBO.PeerIdx) != 10 {
		t.Fatalf("expected 10 peers after grow, got %d", len(ss.TLBO.PeerIdx))
	}
}

func TestTLBOCycleReset(t *testing.T) {
	ss := makeTLBOState(10)
	InitTLBO(ss)
	for tick := 0; tick < tlboMaxTicks+10; tick++ {
		TickTLBO(ss)
	}
	if ss.TLBO.Tick > tlboMaxTicks {
		t.Fatalf("tick should wrap around, got %d", ss.TLBO.Tick)
	}
}

func TestApplyTLBONil(t *testing.T) {
	ss := makeTLBOState(10)
	bot := &ss.Bots[0]
	// Should not panic with nil TLBO
	ApplyTLBO(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("speed should be default when TLBO is nil")
	}
}

func TestTLBOPhaseAlternation(t *testing.T) {
	ss := makeTLBOState(20)
	InitTLBO(ss)

	// After 1 tick, phase should be 1 (odd tick = learner)
	TickTLBO(ss)
	for i := range ss.Bots {
		if ss.TLBO.Phase[i] != 1 {
			t.Fatalf("tick 1: expected learner phase (1), got %d", ss.TLBO.Phase[i])
		}
	}

	// After 2 ticks, phase should be 0 (even tick = teacher)
	TickTLBO(ss)
	for i := range ss.Bots {
		if ss.TLBO.Phase[i] != 0 {
			t.Fatalf("tick 2: expected teacher phase (0), got %d", ss.TLBO.Phase[i])
		}
	}
}

func TestTLBOPeersDifferFromSelf(t *testing.T) {
	ss := makeTLBOState(20)
	InitTLBO(ss)
	TickTLBO(ss)
	for i := range ss.Bots {
		if ss.TLBO.PeerIdx[i] == i {
			t.Fatalf("bot %d: peer should differ from self", i)
		}
	}
}

func TestTLBOGlobalBest(t *testing.T) {
	ss := makeTLBOState(20)
	InitTLBO(ss)
	for tick := 0; tick < 50; tick++ {
		TickTLBO(ss)
	}
	st := ss.TLBO
	if st.GlobalBestF < -1e17 {
		t.Fatal("global best fitness should be set after 50 ticks")
	}
	if st.GlobalBestF < st.BestF-0.001 {
		t.Fatalf("global best (%.2f) should be >= current best (%.2f)", st.GlobalBestF, st.BestF)
	}
}

func TestTLBOMeanComputation(t *testing.T) {
	ss := makeTLBOState(10)
	InitTLBO(ss)
	TickTLBO(ss)

	// Compute expected mean manually
	var sumX, sumY float64
	for i := range ss.Bots {
		sumX += ss.Bots[i].X
		sumY += ss.Bots[i].Y
	}
	expectedMeanX := sumX / float64(len(ss.Bots))
	expectedMeanY := sumY / float64(len(ss.Bots))

	// Allow small float tolerance
	dx := ss.TLBO.MeanX - expectedMeanX
	dy := ss.TLBO.MeanY - expectedMeanY
	if dx*dx+dy*dy > 0.01 {
		t.Fatalf("mean mismatch: got (%.2f, %.2f), expected (%.2f, %.2f)",
			ss.TLBO.MeanX, ss.TLBO.MeanY, expectedMeanX, expectedMeanY)
	}
}
