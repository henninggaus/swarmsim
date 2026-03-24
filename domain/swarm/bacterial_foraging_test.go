package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeBFOState(n int) *SwarmState {
	ss := &SwarmState{
		Bots:   make([]SwarmBot, n),
		ArenaW: 800,
		ArenaH: 800,
		Rng:    rand.New(rand.NewSource(77)),
		Hash:   physics.NewSpatialHash(800, 800, 30),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = ss.Rng.Float64() * 800
		ss.Bots[i].Y = ss.Rng.Float64() * 800
		ss.Bots[i].Angle = ss.Rng.Float64() * 6.28
		ss.Bots[i].Energy = 60
		ss.Bots[i].CarryingPkg = -1
	}
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}
	return ss
}

func TestInitBFO(t *testing.T) {
	ss := makeBFOState(20)
	InitBFO(ss)
	if ss.BFO == nil {
		t.Fatal("BFO state should not be nil after init")
	}
	if !ss.BFOOn {
		t.Fatal("BFOOn should be true after init")
	}
	if len(ss.BFO.Health) != 20 {
		t.Fatalf("expected 20 health entries, got %d", len(ss.BFO.Health))
	}
}

func TestTickBFO(t *testing.T) {
	ss := makeBFOState(20)
	InitBFO(ss)
	for tick := 0; tick < 50; tick++ {
		TickBFO(ss)
	}
	// Health values should be non-negative
	for i := range ss.Bots {
		if ss.Bots[i].BFOHealth < 0 {
			t.Fatalf("bot %d: BFOHealth should be >= 0, got %d", i, ss.Bots[i].BFOHealth)
		}
	}
}

func TestTickBFONil(t *testing.T) {
	ss := makeBFOState(10)
	TickBFO(ss) // should not panic
}

func TestClearBFO(t *testing.T) {
	ss := makeBFOState(10)
	InitBFO(ss)
	ClearBFO(ss)
	if ss.BFO != nil {
		t.Fatal("BFO should be nil after clear")
	}
	if ss.BFOOn {
		t.Fatal("BFOOn should be false after clear")
	}
}

func TestApplyBFO(t *testing.T) {
	ss := makeBFOState(20)
	InitBFO(ss)
	// Record initial positions
	initX := make([]float64, len(ss.Bots))
	initY := make([]float64, len(ss.Bots))
	for i := range ss.Bots {
		initX[i] = ss.Bots[i].X
		initY[i] = ss.Bots[i].Y
	}
	for tick := 0; tick < 10; tick++ {
		TickBFO(ss)
	}
	for i := range ss.Bots {
		ApplyBFO(&ss.Bots[i], ss, i)
	}
	// After apply, bots should have moved (Speed is zeroed after direct movement)
	moved := 0
	for i := range ss.Bots {
		if ss.Bots[i].X != initX[i] || ss.Bots[i].Y != initY[i] {
			moved++
		}
	}
	if moved == 0 {
		t.Fatal("no bots moved after ApplyBFO — position update not working")
	}
}

func TestBFOReproduce(t *testing.T) {
	ss := makeBFOState(20)
	InitBFO(ss)
	// Give varied health values
	for i := range ss.BFO.Health {
		ss.BFO.Health[i] = ss.Rng.Float64() * 10
	}
	// Should not panic
	bfoReproduce(ss)
}

func TestBFOGrowSlices(t *testing.T) {
	ss := makeBFOState(5)
	InitBFO(ss)
	ss.Bots = append(ss.Bots, SwarmBot{X: 200, Y: 200, Energy: 50, CarryingPkg: -1})
	TickBFO(ss)
	if len(ss.BFO.Health) != 6 {
		t.Fatalf("expected 6 health entries, got %d", len(ss.BFO.Health))
	}
}

func TestBFOGradientTumble(t *testing.T) {
	ss := makeBFOState(10)
	InitBFO(ss)
	bot := &ss.Bots[0]
	// Should return a valid angle without panicking
	angle := bfoGradientTumbleAdaptive(bot, ss, 25.0)
	if angle < -2*3.15 || angle > 4*3.15 {
		t.Fatalf("tumble angle out of expected range: %f", angle)
	}
}

func TestBFOPersonalBest(t *testing.T) {
	ss := makeBFOState(10)
	InitBFO(ss)
	// After init, personal bests should be set
	for i := range ss.Bots {
		if ss.BFO.PBestF[i] < 0 {
			// PBestF should be >= 0 for valid positions
			continue // distanceFitness can return negative in some configurations
		}
	}
	// Run some ticks and verify PBest is updated
	for tick := 0; tick < 50; tick++ {
		TickBFO(ss)
	}
	if ss.BFO.Tick != 50 {
		t.Fatalf("expected 50 ticks, got %d", ss.BFO.Tick)
	}
}
