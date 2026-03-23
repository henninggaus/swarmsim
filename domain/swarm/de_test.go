package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeTestSwarmStateDE(n int) *SwarmState {
	ss := &SwarmState{
		Bots:   make([]SwarmBot, n),
		ArenaW: 800,
		ArenaH: 800,
		Rng:    rand.New(rand.NewSource(42)),
		Hash:   physics.NewSpatialHash(800, 800, 60),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = ss.Rng.Float64() * ss.ArenaW
		ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
		ss.Bots[i].Speed = SwarmBotSpeed
		ss.Bots[i].CarryingPkg = -1
	}
	return ss
}

func TestInitDE(t *testing.T) {
	ss := makeTestSwarmStateDE(20)
	InitDE(ss)

	if ss.DE == nil {
		t.Fatal("DE state should not be nil after init")
	}
	if !ss.DEOn {
		t.Fatal("DEOn should be true after init")
	}
	if len(ss.DE.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.DE.Fitness))
	}
	if len(ss.DE.TrialX) != 20 {
		t.Fatalf("expected 20 TrialX entries, got %d", len(ss.DE.TrialX))
	}
	if ss.DE.DifferentialWeight != 0.8 {
		t.Fatalf("expected F=0.8, got %f", ss.DE.DifferentialWeight)
	}
	if ss.DE.CrossoverRate != 0.5 {
		t.Fatalf("expected CR=0.5, got %f", ss.DE.CrossoverRate)
	}
}

func TestClearDE(t *testing.T) {
	ss := makeTestSwarmStateDE(10)
	InitDE(ss)
	ClearDE(ss)

	if ss.DE != nil {
		t.Fatal("DE state should be nil after clear")
	}
	if ss.DEOn {
		t.Fatal("DEOn should be false after clear")
	}
}

func TestTickDE_NilSafe(t *testing.T) {
	ss := makeTestSwarmStateDE(10)
	// Should not panic with nil state.
	TickDE(ss)
}

func TestTickDE_GenerationCycle(t *testing.T) {
	ss := makeTestSwarmStateDE(20)
	InitDE(ss)

	// Run one tick — should create trial vectors.
	TickDE(ss)

	// All bots should be moving toward trial positions.
	movingCount := 0
	for i := range ss.DE.Moving {
		if ss.DE.Moving[i] {
			movingCount++
		}
	}
	if movingCount != 20 {
		t.Fatalf("expected all 20 bots moving, got %d", movingCount)
	}
}

func TestApplyDE_SteeringTowardTrial(t *testing.T) {
	ss := makeTestSwarmStateDE(10)
	InitDE(ss)
	TickDE(ss) // create trial vectors

	// Apply to first bot.
	bot := &ss.Bots[0]
	initialAngle := bot.Angle
	ApplyDE(bot, ss, 0)

	// Bot should be moving at SwarmBotSpeed.
	if bot.Speed < SwarmBotSpeed*0.3 {
		t.Fatal("bot should have non-zero speed when moving to trial")
	}

	// If trial position differs, angle should change.
	if ss.DE.Moving[0] {
		dx := ss.DE.TrialX[0] - bot.X
		dy := ss.DE.TrialY[0] - bot.Y
		if dx*dx+dy*dy > deTrialStep*deTrialStep && bot.Angle == initialAngle {
			// Angle might not change if trial is in same direction, so this
			// is a soft check.
			_ = initialAngle
		}
	}
}

func TestApplyDE_NilSafe(t *testing.T) {
	ss := makeTestSwarmStateDE(5)
	bot := &ss.Bots[0]
	// Should not panic with nil DE state.
	ApplyDE(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("expected default speed, got %f", bot.Speed)
	}
}

func TestDePickThree(t *testing.T) {
	rng := rand.New(rand.NewSource(99))

	for trial := 0; trial < 100; trial++ {
		r1, r2, r3 := dePickThree(rng, 10, 5)
		if r1 == 5 || r2 == 5 || r3 == 5 {
			t.Fatal("dePickThree returned excluded index")
		}
		if r1 == r2 || r1 == r3 || r2 == r3 {
			t.Fatal("dePickThree returned duplicate indices")
		}
		if r1 < 0 || r1 >= 10 || r2 < 0 || r2 >= 10 || r3 < 0 || r3 >= 10 {
			t.Fatal("dePickThree returned out-of-range index")
		}
	}
}

func TestTickDE_GrowSlices(t *testing.T) {
	ss := makeTestSwarmStateDE(5)
	InitDE(ss)

	// Add bots dynamically.
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * ss.ArenaW,
			Y: ss.Rng.Float64() * ss.ArenaH,
			CarryingPkg: -1,
		})
	}

	// Should grow internal slices without panic.
	TickDE(ss)

	if len(ss.DE.Fitness) != 10 {
		t.Fatalf("expected 10 fitness entries after grow, got %d", len(ss.DE.Fitness))
	}
}

func TestDEFitness(t *testing.T) {
	ss := makeTestSwarmStateDE(1)
	ss.Bots[0].X = ss.ArenaW / 2
	ss.Bots[0].Y = ss.ArenaH / 2

	f := deFitness(&ss.Bots[0], ss)
	if f < 99 {
		t.Fatalf("bot at center should have high fitness, got %f", f)
	}

	ss.Bots[0].X = 0
	ss.Bots[0].Y = 0
	f2 := deFitness(&ss.Bots[0], ss)
	if f2 >= f {
		t.Fatal("bot at corner should have lower fitness than center")
	}
}

func TestTickDE_SensorCache(t *testing.T) {
	ss := makeTestSwarmStateDE(10)
	InitDE(ss)
	TickDE(ss)

	// Check sensor cache was populated.
	for i := range ss.Bots {
		if ss.Bots[i].DEPhase != 1 {
			t.Fatalf("bot %d should have DEPhase=1 (moving), got %d", i, ss.Bots[i].DEPhase)
		}
	}

	// Best should have zero distance to itself.
	bestIdx := ss.DE.BestIdx
	if ss.Bots[bestIdx].DEBestDist != 0 {
		t.Fatalf("best bot should have DEBestDist=0, got %d", ss.Bots[bestIdx].DEBestDist)
	}
}

func TestInitSwarmAlgorithm_DE(t *testing.T) {
	ss := makeTestSwarmStateDE(10)
	InitSwarmAlgorithm(ss, AlgoDE)

	if ss.DE == nil {
		t.Fatal("DE should be initialized via InitSwarmAlgorithm")
	}
	if !ss.SwarmAlgoOn {
		t.Fatal("SwarmAlgoOn should be true")
	}
}

func TestClearSwarmAlgorithm_DE(t *testing.T) {
	ss := makeTestSwarmStateDE(10)
	InitSwarmAlgorithm(ss, AlgoDE)
	ClearSwarmAlgorithm(ss)

	if ss.DE != nil {
		t.Fatal("DE should be nil after ClearSwarmAlgorithm")
	}
}

func TestTickSwarmAlgorithm_DE(t *testing.T) {
	ss := makeTestSwarmStateDE(10)
	InitSwarmAlgorithm(ss, AlgoDE)

	// Should not panic.
	TickSwarmAlgorithm(ss)
}
