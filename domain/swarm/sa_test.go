package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeTestSwarmStateSA(n int) *SwarmState {
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

func TestInitSA(t *testing.T) {
	ss := makeTestSwarmStateSA(20)
	InitSA(ss)

	if ss.SA == nil {
		t.Fatal("SA state should not be nil after init")
	}
	if !ss.SAOn {
		t.Fatal("SAOn should be true after init")
	}
	if len(ss.SA.Fitness) != 20 {
		t.Fatalf("expected 20 fitness entries, got %d", len(ss.SA.Fitness))
	}
	if ss.SA.InitialTemp != 100.0 {
		t.Fatalf("expected InitialTemp=100.0, got %f", ss.SA.InitialTemp)
	}
	if ss.SA.CoolingRate != 0.995 {
		t.Fatalf("expected CoolingRate=0.995, got %f", ss.SA.CoolingRate)
	}

	// All bots should start at initial temperature.
	for i, temp := range ss.SA.Temp {
		if temp != 100.0 {
			t.Fatalf("bot %d should have temp=100.0, got %f", i, temp)
		}
	}
}

func TestClearSA(t *testing.T) {
	ss := makeTestSwarmStateSA(10)
	InitSA(ss)
	ClearSA(ss)

	if ss.SA != nil {
		t.Fatal("SA state should be nil after clear")
	}
	if ss.SAOn {
		t.Fatal("SAOn should be false after clear")
	}
}

func TestTickSA_NilSafe(t *testing.T) {
	ss := makeTestSwarmStateSA(10)
	TickSA(ss)
}

func TestTickSA_TemperatureCooling(t *testing.T) {
	ss := makeTestSwarmStateSA(10)
	InitSA(ss)

	initialTemp := ss.SA.Temp[0]

	// Run a few ticks.
	for tick := 0; tick < 10; tick++ {
		TickSA(ss)
		// Apply to move bots to their targets quickly.
		for i := range ss.Bots {
			if ss.SA.Moving[i] {
				ss.Bots[i].X = ss.SA.TargetX[i]
				ss.Bots[i].Y = ss.SA.TargetY[i]
				ss.SA.Moving[i] = false
			}
		}
	}

	// Temperature should have decreased (cooling).
	if ss.SA.Temp[0] >= initialTemp {
		t.Fatalf("temperature should decrease, got %f >= %f", ss.SA.Temp[0], initialTemp)
	}
}

func TestApplySA_NilSafe(t *testing.T) {
	ss := makeTestSwarmStateSA(5)
	bot := &ss.Bots[0]
	ApplySA(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("expected default speed, got %f", bot.Speed)
	}
}

func TestApplySA_DirectMovement(t *testing.T) {
	ss := makeTestSwarmStateSA(10)
	InitSA(ss)
	TickSA(ss) // generate targets

	// Find a bot that is moving.
	movIdx := -1
	for i := range ss.Bots {
		if ss.SA.Moving[i] {
			movIdx = i
			break
		}
	}
	if movIdx < 0 {
		t.Skip("no bot accepted a move this tick")
	}

	bot := &ss.Bots[movIdx]
	origX, origY := bot.X, bot.Y
	ApplySA(bot, ss, movIdx)

	// Bot should have moved via direct position update.
	dx := bot.X - origX
	dy := bot.Y - origY
	if dx*dx+dy*dy < 0.01 {
		t.Fatal("bot with Moving=true should have changed position directly")
	}
	// Speed should be 0 to prevent double movement.
	if bot.Speed != 0 {
		t.Fatalf("expected Speed=0 after direct move, got %f", bot.Speed)
	}
}

func TestSAFitness(t *testing.T) {
	ss := makeTestSwarmStateSA(1)
	ss.Bots[0].X = ss.ArenaW / 2
	ss.Bots[0].Y = ss.ArenaH / 2

	f := saFitness(&ss.Bots[0], ss)
	if f < 99 {
		t.Fatalf("bot at center should have high fitness, got %f", f)
	}

	ss.Bots[0].X = 0
	ss.Bots[0].Y = 0
	f2 := saFitness(&ss.Bots[0], ss)
	if f2 >= f {
		t.Fatal("bot at corner should have lower fitness than center")
	}
}

func TestSAFitnessAt(t *testing.T) {
	ss := makeTestSwarmStateSA(1)
	f1 := saFitnessAt(ss.ArenaW/2, ss.ArenaH/2, ss)
	f2 := saFitnessAt(0, 0, ss)
	if f2 >= f1 {
		t.Fatal("corner fitness should be lower than center fitness")
	}
}

func TestTickSA_SensorCache(t *testing.T) {
	ss := makeTestSwarmStateSA(10)
	InitSA(ss)
	TickSA(ss)

	// Best bot should have zero distance to itself.
	bestIdx := ss.SA.GlobalBestIdx
	if ss.Bots[bestIdx].SABestDist != 0 {
		t.Fatalf("best bot should have SABestDist=0, got %d", ss.Bots[bestIdx].SABestDist)
	}

	// All bots should have SATemp set.
	for i := range ss.Bots {
		if ss.Bots[i].SATemp <= 0 {
			t.Fatalf("bot %d should have positive SATemp, got %d", i, ss.Bots[i].SATemp)
		}
	}
}

func TestTickSA_GrowSlices(t *testing.T) {
	ss := makeTestSwarmStateSA(5)
	InitSA(ss)

	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X:           ss.Rng.Float64() * ss.ArenaW,
			Y:           ss.Rng.Float64() * ss.ArenaH,
			CarryingPkg: -1,
		})
	}

	TickSA(ss)

	if len(ss.SA.Fitness) != 10 {
		t.Fatalf("expected 10 fitness entries after grow, got %d", len(ss.SA.Fitness))
	}
}

func TestInitSwarmAlgorithm_SA(t *testing.T) {
	ss := makeTestSwarmStateSA(10)
	InitSwarmAlgorithm(ss, AlgoSA)

	if ss.SA == nil {
		t.Fatal("SA should be initialized via InitSwarmAlgorithm")
	}
	if !ss.SwarmAlgoOn {
		t.Fatal("SwarmAlgoOn should be true")
	}
}

func TestClearSwarmAlgorithm_SA(t *testing.T) {
	ss := makeTestSwarmStateSA(10)
	InitSwarmAlgorithm(ss, AlgoSA)
	ClearSwarmAlgorithm(ss)

	if ss.SA != nil {
		t.Fatal("SA should be nil after ClearSwarmAlgorithm")
	}
}

func TestTickSwarmAlgorithm_SA(t *testing.T) {
	ss := makeTestSwarmStateSA(10)
	InitSwarmAlgorithm(ss, AlgoSA)
	TickSwarmAlgorithm(ss)
}
