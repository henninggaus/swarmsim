package swarm

import (
	"math"
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeTestSwarmStateABC(n int) *SwarmState {
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

func TestInitABC(t *testing.T) {
	ss := makeTestSwarmStateABC(20)
	InitABC(ss)

	if ss.ABC == nil {
		t.Fatal("ABC state should not be nil after init")
	}
	if !ss.ABCOn {
		t.Fatal("ABCOn should be true after init")
	}
	if len(ss.ABC.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.ABC.Fitness))
	}
	if len(ss.ABC.Role) != 20 {
		t.Fatalf("expected 20 role entries, got %d", len(ss.ABC.Role))
	}

	// Verify role assignment: first half employed, second half onlooker.
	employedCount := 0
	onlookerCount := 0
	for _, r := range ss.ABC.Role {
		switch r {
		case 0:
			employedCount++
		case 1:
			onlookerCount++
		}
	}
	if employedCount == 0 {
		t.Fatal("expected some employed bees")
	}
	if onlookerCount == 0 {
		t.Fatal("expected some onlooker bees")
	}
}

func TestClearABC(t *testing.T) {
	ss := makeTestSwarmStateABC(10)
	InitABC(ss)
	ClearABC(ss)

	if ss.ABC != nil {
		t.Fatal("ABC state should be nil after clear")
	}
	if ss.ABCOn {
		t.Fatal("ABCOn should be false after clear")
	}
}

func TestTickABC_NilSafe(t *testing.T) {
	ss := makeTestSwarmStateABC(10)
	// Should not panic when ABC is nil.
	TickABC(ss)
	ApplyABC(&ss.Bots[0], ss, 0)
}

func TestTickABC_Basic(t *testing.T) {
	ss := makeTestSwarmStateABC(20)
	InitABC(ss)

	// Run several ticks and verify no panics, bots move.
	for tick := 0; tick < 50; tick++ {
		ss.Tick = tick
		ss.Hash.Clear()
		for i := range ss.Bots {
			ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
		}

		TickABC(ss)
		for i := range ss.Bots {
			ApplyABC(&ss.Bots[i], ss, i)
		}

		// Apply movement.
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			bot.X += bot.Speed * math.Cos(bot.Angle)
			bot.Y += bot.Speed * math.Sin(bot.Angle)
			// Clamp to arena.
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
		if ss.Bots[i].ABCFitness != 0 {
			anyFitness = true
			break
		}
	}
	if !anyFitness {
		t.Fatal("expected at least one bot with non-zero ABCFitness")
	}
}

func TestTickABC_ScoutPhase(t *testing.T) {
	ss := makeTestSwarmStateABC(10)
	InitABC(ss)

	// Force a bot to become stale enough to trigger scouting.
	ss.ABC.Stale[0] = abcAbandonLimit + 1

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickABC(ss)

	// Bot 0 should now be a scout.
	if ss.ABC.Role[0] != 2 {
		t.Fatalf("expected bot 0 to be scout (role=2), got role=%d", ss.ABC.Role[0])
	}
}

func TestInitSwarmAlgorithm_ABC(t *testing.T) {
	ss := makeTestSwarmStateABC(15)
	InitSwarmAlgorithm(ss, AlgoABC)
	if ss.ABC == nil {
		t.Fatal("ABC should be initialized via InitSwarmAlgorithm")
	}
	if !ss.SwarmAlgoOn {
		t.Fatal("SwarmAlgoOn should be true")
	}
}

func TestClearSwarmAlgorithm_ABC(t *testing.T) {
	ss := makeTestSwarmStateABC(15)
	InitSwarmAlgorithm(ss, AlgoABC)
	ClearSwarmAlgorithm(ss)
	if ss.ABC != nil {
		t.Fatal("ABC should be nil after ClearSwarmAlgorithm")
	}
}

func TestTickSwarmAlgorithm_ABC(t *testing.T) {
	ss := makeTestSwarmStateABC(15)
	InitSwarmAlgorithm(ss, AlgoABC)

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Should not panic.
	TickSwarmAlgorithm(ss)
}

func TestABCFitness_LightSource(t *testing.T) {
	ss := makeTestSwarmStateABC(5)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400

	// Bot near the light should have higher fitness than bot far away.
	ss.Bots[0].X = 400
	ss.Bots[0].Y = 400
	ss.Bots[1].X = 100
	ss.Bots[1].Y = 100

	f0 := abcFitness(&ss.Bots[0], ss)
	f1 := abcFitness(&ss.Bots[1], ss)

	if f0 <= f1 {
		t.Fatalf("bot near light should have higher fitness: f0=%f, f1=%f", f0, f1)
	}
}
