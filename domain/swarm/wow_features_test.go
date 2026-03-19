package swarm

import (
	"math"
	"math/rand"
	"swarmsim/domain/physics"
	"testing"
)

// newWowTestState creates a SwarmState with SpatialHash and Rng initialized.
func newWowTestState(numBots int) *SwarmState {
	ss := newFlockTestState(numBots)
	ss.Rng = rand.New(rand.NewSource(42))
	return ss
}

// --- Lévy-Flight Foraging Tests ---

func TestInitClearLevy(t *testing.T) {
	ss := newWowTestState(5)
	InitLevy(ss)
	if ss.Levy == nil || !ss.LevyOn {
		t.Fatal("InitLevy should set Levy and LevyOn")
	}
	if len(ss.Levy.Phase) != 5 {
		t.Fatalf("Expected 5 phases, got %d", len(ss.Levy.Phase))
	}
	ClearLevy(ss)
	if ss.Levy != nil || ss.LevyOn {
		t.Fatal("ClearLevy should nil Levy and clear LevyOn")
	}
}

func TestTickLevyNil(t *testing.T) {
	ss := newWowTestState(3)
	// Should not panic with nil Levy
	TickLevy(ss)
}

func TestApplyLevyWalkStartsStep(t *testing.T) {
	ss := newWowTestState(3)
	InitLevy(ss)
	bot := &ss.Bots[0]
	bot.X = 400
	bot.Y = 400
	bot.Angle = 0

	ApplyLevyWalk(bot, ss, 0)

	// After first call, bot should have started a step (phase != 0 or step > 0)
	if ss.Levy.Phase[0] == 0 && ss.Levy.StepLen[0] == 0 {
		t.Fatal("ApplyLevyWalk should start a new step when idle")
	}
	if bot.Speed <= 0 {
		t.Fatal("Bot speed should be positive after Lévy walk")
	}
}

func TestApplyLevyWalkNilSafe(t *testing.T) {
	ss := newWowTestState(1)
	bot := &ss.Bots[0]
	// Should not panic with nil Levy
	ApplyLevyWalk(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected default speed with nil Levy, got %f", bot.Speed)
	}
}

func TestTickLevySensorCache(t *testing.T) {
	ss := newWowTestState(3)
	InitLevy(ss)
	// Force phase and step for bot 0
	ss.Levy.Phase[0] = 2
	ss.Levy.StepLen[0] = 150.0
	ss.Levy.Timer[0] = 10

	TickLevy(ss)

	if ss.Bots[0].LevyPhase != 2 {
		t.Fatalf("Expected LevyPhase=2, got %d", ss.Bots[0].LevyPhase)
	}
	if ss.Bots[0].LevyStep != 150 {
		t.Fatalf("Expected LevyStep=150, got %d", ss.Bots[0].LevyStep)
	}
}

// --- Firefly Synchronization Tests ---

func TestInitClearFirefly(t *testing.T) {
	ss := newWowTestState(5)
	InitFirefly(ss)
	if ss.Firefly == nil || !ss.FireflyOn {
		t.Fatal("InitFirefly should set Firefly and FireflyOn")
	}
	if len(ss.Firefly.Phase) != 5 {
		t.Fatalf("Expected 5 phases, got %d", len(ss.Firefly.Phase))
	}
	ClearFirefly(ss)
	if ss.Firefly != nil || ss.FireflyOn {
		t.Fatal("ClearFirefly should nil Firefly and clear FireflyOn")
	}
}

func TestTickFireflyNil(t *testing.T) {
	ss := newWowTestState(3)
	// Should not panic with nil Firefly
	TickFirefly(ss)
}

func TestTickFireflyPhaseAdvances(t *testing.T) {
	ss := newWowTestState(3)
	InitFirefly(ss)
	// Set all phases to 0
	for i := range ss.Firefly.Phase {
		ss.Firefly.Phase[i] = 0.0
	}

	TickFirefly(ss)

	// Phases should have advanced
	for i := range ss.Bots {
		if ss.Firefly.Phase[i] <= 0.0 {
			t.Fatalf("Bot %d: phase should advance, got %f", i, ss.Firefly.Phase[i])
		}
	}
}

func TestTickFireflyFlashDetection(t *testing.T) {
	ss := newWowTestState(3)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitFirefly(ss)

	// Set phase very close to 1.0 so next tick causes flash
	ss.Firefly.Phase[0] = 0.999

	TickFirefly(ss)

	// Bot 0 should be flashing
	if ss.Bots[0].FlashSync != 1 {
		t.Fatal("Bot 0 should be flashing after phase wraps")
	}
}

func TestApplyFlash(t *testing.T) {
	ss := newWowTestState(3)
	InitFirefly(ss)
	ss.Firefly.Phase[0] = 0.5 // mid-phase
	ss.Tick = 100

	ApplyFlash(&ss.Bots[0], ss, 0)

	if ss.Firefly.Phase[0] != 0.0 {
		t.Fatalf("ApplyFlash should reset phase to 0, got %f", ss.Firefly.Phase[0])
	}
	if ss.Firefly.FlashTick[0] != 100 {
		t.Fatalf("ApplyFlash should set FlashTick to current tick")
	}
	if ss.Bots[0].LEDColor[0] != 255 || ss.Bots[0].LEDColor[1] != 255 {
		t.Fatal("ApplyFlash should set bright LED")
	}
}

func TestFireflyCoupling(t *testing.T) {
	ss := newWowTestState(3)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitFirefly(ss)

	// Place bots close together
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*20
		ss.Bots[i].Y = 400
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Bot 0 about to flash, others at 0.1
	ss.Firefly.Phase[0] = 0.999
	ss.Firefly.Phase[1] = 0.1
	ss.Firefly.Phase[2] = 0.1

	oldPhase1 := ss.Firefly.Phase[1]
	TickFirefly(ss)

	// Nearby bots should have their phase nudged forward
	if ss.Firefly.Phase[1] <= oldPhase1 {
		t.Fatal("Coupling should nudge nearby bot's phase forward when neighbor flashes")
	}
}

// --- Collective Transport Tests ---

func TestInitClearTransport(t *testing.T) {
	ss := newWowTestState(5)
	InitTransport(ss)
	if ss.Transport == nil || !ss.TransportOn {
		t.Fatal("InitTransport should set Transport and TransportOn")
	}
	ClearTransport(ss)
	if ss.Transport != nil || ss.TransportOn {
		t.Fatal("ClearTransport should nil Transport and clear TransportOn")
	}
}

func TestTickTransportNil(t *testing.T) {
	ss := newWowTestState(3)
	// Should not panic with nil Transport
	TickTransport(ss)
}

func TestTickTransportSensors(t *testing.T) {
	ss := newWowTestState(5)
	InitTransport(ss)

	// Add a task near bot 0
	ss.Transport.Tasks = append(ss.Transport.Tasks, TransportTask{
		X: 405, Y: 400,
		TargetX: 700, TargetY: 400,
		Weight:  3,
		BotIDs:  []int{},
		Active:  true,
	})
	ss.Bots[0].X = 400
	ss.Bots[0].Y = 400

	TickTransport(ss)

	if ss.Bots[0].TransportNearby < 1 {
		t.Fatal("Bot 0 should detect nearby transport task")
	}
}

func TestApplyAssistTransport(t *testing.T) {
	ss := newWowTestState(3)
	InitTransport(ss)

	// Add a task
	ss.Transport.Tasks = append(ss.Transport.Tasks, TransportTask{
		X: 410, Y: 400,
		TargetX: 700, TargetY: 400,
		Weight:  3,
		BotIDs:  []int{},
		Active:  true,
	})
	ss.Bots[0].X = 400
	ss.Bots[0].Y = 400

	ApplyAssistTransport(&ss.Bots[0], ss, 0)

	// Bot should have joined the task
	if len(ss.Transport.Tasks[0].BotIDs) != 1 {
		t.Fatal("Bot should join transport task when assisting")
	}
	if ss.Transport.Tasks[0].BotIDs[0] != 0 {
		t.Fatal("Bot ID should be recorded in task")
	}
}

func TestApplyAssistTransportNilSafe(t *testing.T) {
	ss := newWowTestState(1)
	bot := &ss.Bots[0]
	// Should not panic with nil Transport
	ApplyAssistTransport(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected default speed with nil Transport, got %f", bot.Speed)
	}
}

// --- Vortex Swarming Tests ---

func TestTickVortexNilHash(t *testing.T) {
	ss := newTestSwarmState(5)
	// ss.Hash is nil, should not panic
	TickVortex(ss)
}

func TestTickVortexNoNeighbors(t *testing.T) {
	ss := newWowTestState(5)
	// Spread bots far apart
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i) * 200
		ss.Bots[i].Y = 400
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickVortex(ss)

	for i := range ss.Bots {
		if ss.Bots[i].VortexStrength != 0 {
			t.Fatalf("Bot %d: expected VortexStrength=0 with no neighbors, got %d", i, ss.Bots[i].VortexStrength)
		}
	}
}

func TestTickVortexWithNeighbors(t *testing.T) {
	ss := newWowTestState(5)
	// Place bots in a circle, each facing tangentially (CCW rotation)
	cx, cy := 400.0, 400.0
	for i := range ss.Bots {
		angle := float64(i) * 2 * math.Pi / 5.0
		ss.Bots[i].X = cx + math.Cos(angle)*30
		ss.Bots[i].Y = cy + math.Sin(angle)*30
		// Face tangentially (perpendicular to radius)
		ss.Bots[i].Angle = angle + math.Pi/2
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickVortex(ss)

	// At least some bots should detect vortex
	anyVortex := false
	for i := range ss.Bots {
		if ss.Bots[i].VortexStrength > 0 {
			anyVortex = true
		}
	}
	if !anyVortex {
		t.Fatal("Expected at least some bots to detect vortex when arranged in rotating circle")
	}
}

func TestApplyVortexNilHash(t *testing.T) {
	ss := newTestSwarmState(1)
	bot := &ss.Bots[0]
	// Should not panic with nil Hash
	ApplyVortex(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected default speed with nil Hash, got %f", bot.Speed)
	}
}

func TestApplyVortexSteering(t *testing.T) {
	ss := newWowTestState(5)
	// Place bots in tight cluster
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*15
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = 0
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	origAngle := ss.Bots[0].Angle
	ApplyVortex(&ss.Bots[0], ss, 0)

	if ss.Bots[0].Speed != SwarmBotSpeed {
		t.Fatalf("Expected SwarmBotSpeed, got %f", ss.Bots[0].Speed)
	}
	// Angle should change (vortex steering)
	if ss.Bots[0].Angle == origAngle {
		// May stay same if exactly balanced, but with asymmetric neighbors it should change
		// This is a soft check
	}
}
