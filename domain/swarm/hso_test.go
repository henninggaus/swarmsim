package swarm

import (
	"math"
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeTestSwarmStateHSO(n int) *SwarmState {
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

func TestInitHSO(t *testing.T) {
	ss := makeTestSwarmStateHSO(20)
	InitHSO(ss)

	if ss.HSO == nil {
		t.Fatal("HSO state should not be nil after init")
	}
	if !ss.HSOOn {
		t.Fatal("HSOOn should be true after init")
	}
	if len(ss.HSO.TargetX) != 20 {
		t.Fatalf("expected 20 target entries, got %d", len(ss.HSO.TargetX))
	}
	if len(ss.HSO.HM) == 0 {
		t.Fatal("Harmony Memory should be seeded with initial positions")
	}
	if len(ss.HSO.HM) > hsoHMSize {
		t.Fatalf("HM should not exceed hsoHMSize=%d, got %d", hsoHMSize, len(ss.HSO.HM))
	}
}

func TestClearHSO(t *testing.T) {
	ss := makeTestSwarmStateHSO(10)
	InitHSO(ss)
	ClearHSO(ss)

	if ss.HSO != nil {
		t.Fatal("HSO state should be nil after clear")
	}
	if ss.HSOOn {
		t.Fatal("HSOOn should be false after clear")
	}
}

func TestTickHSO_NilSafe(t *testing.T) {
	ss := makeTestSwarmStateHSO(10)
	// Should not panic when HSO is nil.
	TickHSO(ss)
	ApplyHSO(&ss.Bots[0], ss, 0)
}

func TestTickHSO_Basic(t *testing.T) {
	ss := makeTestSwarmStateHSO(20)
	InitHSO(ss)

	// Run several ticks and verify no panics, bots move.
	for tick := 0; tick < 50; tick++ {
		ss.Tick = tick
		ss.Hash.Clear()
		for i := range ss.Bots {
			ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
		}

		TickHSO(ss)
		for i := range ss.Bots {
			ApplyHSO(&ss.Bots[i], ss, i)
		}

		// Apply movement.
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			bot.X += bot.Speed * math.Cos(bot.Angle)
			bot.Y += bot.Speed * math.Sin(bot.Angle)
			if bot.X < SwarmEdgeMargin {
				bot.X = SwarmEdgeMargin
			}
			if bot.X > ss.ArenaW-SwarmEdgeMargin {
				bot.X = ss.ArenaW - SwarmEdgeMargin
			}
			if bot.Y < SwarmEdgeMargin {
				bot.Y = SwarmEdgeMargin
			}
			if bot.Y > ss.ArenaH-SwarmEdgeMargin {
				bot.Y = ss.ArenaH - SwarmEdgeMargin
			}
		}
	}

	// Verify sensor cache was populated.
	anyFitness := false
	for i := range ss.Bots {
		if ss.Bots[i].HSOFitness != 0 {
			anyFitness = true
			break
		}
	}
	if !anyFitness {
		t.Fatal("expected at least one bot with non-zero HSOFitness")
	}
}

func TestTickHSO_SliceGrowth(t *testing.T) {
	ss := makeTestSwarmStateHSO(10)
	InitHSO(ss)

	// Add more bots after init.
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X:     ss.Rng.Float64() * ss.ArenaW,
			Y:     ss.Rng.Float64() * ss.ArenaH,
			Speed: SwarmBotSpeed,
		})
	}

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Should not panic with new bots.
	TickHSO(ss)
	for i := range ss.Bots {
		ApplyHSO(&ss.Bots[i], ss, i)
	}
}

func TestTickHSO_HarmonyMemoryUpdate(t *testing.T) {
	ss := makeTestSwarmStateHSO(15)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400
	InitHSO(ss)

	// Run many ticks to allow harmonies to be improvised and HM updated.
	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		ss.Hash.Clear()
		for i := range ss.Bots {
			ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
		}

		TickHSO(ss)
		for i := range ss.Bots {
			ApplyHSO(&ss.Bots[i], ss, i)
		}

		for i := range ss.Bots {
			bot := &ss.Bots[i]
			bot.X += bot.Speed * math.Cos(bot.Angle)
			bot.Y += bot.Speed * math.Sin(bot.Angle)
			if bot.X < SwarmEdgeMargin {
				bot.X = SwarmEdgeMargin
			}
			if bot.X > ss.ArenaW-SwarmEdgeMargin {
				bot.X = ss.ArenaW - SwarmEdgeMargin
			}
			if bot.Y < SwarmEdgeMargin {
				bot.Y = SwarmEdgeMargin
			}
			if bot.Y > ss.ArenaH-SwarmEdgeMargin {
				bot.Y = ss.ArenaH - SwarmEdgeMargin
			}
		}
	}

	// After many ticks, best fitness should have improved.
	if ss.HSO.BestF < 0 {
		t.Fatalf("expected positive best fitness after convergence, got %f", ss.HSO.BestF)
	}
}

func TestInitSwarmAlgorithm_HSO(t *testing.T) {
	ss := makeTestSwarmStateHSO(15)
	InitSwarmAlgorithm(ss, AlgoHSO)
	if ss.HSO == nil {
		t.Fatal("HSO should be initialized via InitSwarmAlgorithm")
	}
	if !ss.SwarmAlgoOn {
		t.Fatal("SwarmAlgoOn should be true")
	}
}

func TestClearSwarmAlgorithm_HSO(t *testing.T) {
	ss := makeTestSwarmStateHSO(15)
	InitSwarmAlgorithm(ss, AlgoHSO)
	ClearSwarmAlgorithm(ss)
	if ss.HSO != nil {
		t.Fatal("HSO should be nil after ClearSwarmAlgorithm")
	}
}

func TestTickSwarmAlgorithm_HSO(t *testing.T) {
	ss := makeTestSwarmStateHSO(15)
	InitSwarmAlgorithm(ss, AlgoHSO)

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Should not panic.
	TickSwarmAlgorithm(ss)
}

func TestHSOFitness_LightSource(t *testing.T) {
	ss := makeTestSwarmStateHSO(5)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400

	// Bot near the light should have higher fitness than bot far away.
	ss.Bots[0].X = 400
	ss.Bots[0].Y = 400
	ss.Bots[1].X = 100
	ss.Bots[1].Y = 100

	f0 := hsoFitness(&ss.Bots[0], ss)
	f1 := hsoFitness(&ss.Bots[1], ss)

	if f0 <= f1 {
		t.Fatalf("bot near light should have higher fitness: f0=%f, f1=%f", f0, f1)
	}
}
