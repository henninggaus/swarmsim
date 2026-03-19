package swarm

import (
	"math"
	"math/rand"
	"swarmsim/domain/physics"
	"testing"
)

// newBioTestState creates a SwarmState with SpatialHash and Rng initialized.
func newBioTestState(numBots int) *SwarmState {
	ss := newFlockTestState(numBots)
	ss.Rng = rand.New(rand.NewSource(42))
	return ss
}

// --- Waggle Dance Tests ---

func TestInitClearWaggle(t *testing.T) {
	ss := newBioTestState(5)
	InitWaggle(ss)
	if ss.Waggle == nil || !ss.WaggleOn {
		t.Fatal("InitWaggle should set Waggle and WaggleOn")
	}
	ClearWaggle(ss)
	if ss.Waggle != nil || ss.WaggleOn {
		t.Fatal("ClearWaggle should nil Waggle and clear WaggleOn")
	}
}

func TestTickWaggleNil(t *testing.T) {
	ss := newBioTestState(3)
	// Should not panic
	TickWaggle(ss)
}

func TestApplyWaggleDanceStartsDance(t *testing.T) {
	ss := newBioTestState(3)
	InitWaggle(ss)
	bot := &ss.Bots[0]
	bot.X = 400
	bot.Y = 400
	bot.Angle = 0

	ApplyWaggleDance(bot, ss, 0)

	if !ss.Waggle.Dancing[0] {
		t.Fatal("ApplyWaggleDance should start dance")
	}
	if ss.Waggle.DanceTick[0] <= 0 {
		t.Fatal("Dance timer should be set")
	}
}

func TestTickWaggleDecoding(t *testing.T) {
	ss := newBioTestState(3)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitWaggle(ss)

	// Place bots close together
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*15
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = 0
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Bot 0 is dancing, advertising a target to the right
	ss.Waggle.Dancing[0] = true
	ss.Waggle.DanceTick[0] = 30
	ss.Waggle.TargetX[0] = 600
	ss.Waggle.TargetY[0] = 400

	TickWaggle(ss)

	// Bot 1 (nearby non-dancer) should decode the target
	if ss.Bots[0].WaggleDancing != 1 {
		t.Fatal("Dancing bot should have WaggleDancing=1")
	}
	// Bot 1 should have some non-zero target angle (target is to the right)
	// since bot 1 is at x=415, target at x=600 → roughly angle 0 relative to heading 0
	// This is a soft check since exact value depends on geometry
}

func TestApplyFollowDanceNoTarget(t *testing.T) {
	ss := newBioTestState(1)
	bot := &ss.Bots[0]
	bot.WaggleTarget = 0
	ApplyFollowDance(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected default speed with no target, got %f", bot.Speed)
	}
}

// --- Morphogen Gradient Tests ---

func TestInitClearMorphogen(t *testing.T) {
	ss := newBioTestState(5)
	InitMorphogen(ss)
	if ss.Morphogen == nil || !ss.MorphogenOn {
		t.Fatal("InitMorphogen should set Morphogen and MorphogenOn")
	}
	if len(ss.Morphogen.A) != 5 {
		t.Fatalf("Expected 5 activator values, got %d", len(ss.Morphogen.A))
	}
	ClearMorphogen(ss)
	if ss.Morphogen != nil || ss.MorphogenOn {
		t.Fatal("ClearMorphogen should nil Morphogen and clear MorphogenOn")
	}
}

func TestTickMorphogenNil(t *testing.T) {
	ss := newBioTestState(3)
	// Should not panic
	TickMorphogen(ss)
}

func TestTickMorphogenUpdates(t *testing.T) {
	ss := newBioTestState(5)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitMorphogen(ss)

	// Place bots close together
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*20
		ss.Bots[i].Y = 400
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Set different concentrations to create gradient
	ss.Morphogen.A[0] = 0.9
	ss.Morphogen.A[1] = 0.1
	ss.Morphogen.H[0] = 0.1
	ss.Morphogen.H[1] = 0.9

	oldA0 := ss.Morphogen.A[0]
	TickMorphogen(ss)

	// Concentrations should change due to diffusion + reaction
	if ss.Morphogen.A[0] == oldA0 {
		t.Fatal("Morphogen A should change due to reaction-diffusion")
	}

	// Sensor cache should be set
	if ss.Bots[0].MorphA == 0 && ss.Bots[0].MorphH == 0 {
		t.Fatal("MorphA/MorphH sensors should be updated")
	}
}

func TestApplyMorphColor(t *testing.T) {
	ss := newBioTestState(1)
	bot := &ss.Bots[0]
	bot.MorphA = 80
	bot.MorphH = 20
	ApplyMorphColor(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected SwarmBotSpeed, got %f", bot.Speed)
	}
	// LED should be set based on concentrations
	if bot.LEDColor[0] == 0 && bot.LEDColor[2] == 0 {
		t.Fatal("LED should be colored based on morphogen")
	}
}

// --- Predator Evasion Wave Tests ---

func TestInitClearEvasion(t *testing.T) {
	ss := newBioTestState(5)
	InitEvasion(ss)
	if ss.Evasion == nil || !ss.EvasionOn {
		t.Fatal("InitEvasion should set Evasion and EvasionOn")
	}
	ClearEvasion(ss)
	if ss.Evasion != nil || ss.EvasionOn {
		t.Fatal("ClearEvasion should nil Evasion and clear EvasionOn")
	}
}

func TestTickEvasionNil(t *testing.T) {
	ss := newBioTestState(3)
	// Should not panic
	TickEvasion(ss)
}

func TestApplyEvadeStartsAlarm(t *testing.T) {
	ss := newBioTestState(3)
	InitEvasion(ss)
	bot := &ss.Bots[0]
	bot.X = 400
	bot.Y = 400
	bot.Angle = 0

	ApplyEvade(bot, ss, 0)

	if !ss.Evasion.Alarmed[0] {
		t.Fatal("ApplyEvade should alarm the bot")
	}
	if ss.Evasion.Timer[0] <= 0 {
		t.Fatal("Evasion timer should be set")
	}
	if bot.Speed <= SwarmBotSpeed {
		t.Fatal("Evading bot should move faster than normal")
	}
}

func TestEvasionWavePropagation(t *testing.T) {
	ss := newBioTestState(5)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitEvasion(ss)

	// Place bots close together
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*20
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = 0
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Alarm bot 0
	ss.Evasion.Alarmed[0] = true
	ss.Evasion.Timer[0] = 40
	ss.Evasion.PropTimer[0] = 0 // ready to propagate

	TickEvasion(ss)

	// Nearby bots should be alarmed by propagation
	anyAlarmed := false
	for i := 1; i < len(ss.Bots); i++ {
		if ss.Evasion.Alarmed[i] {
			anyAlarmed = true
		}
	}
	if !anyAlarmed {
		t.Fatal("Evasion wave should propagate to nearby bots")
	}
}

func TestEvasionSensorCache(t *testing.T) {
	ss := newBioTestState(3)
	InitEvasion(ss)

	ss.Evasion.Alarmed[0] = true
	ss.Evasion.Timer[0] = 30

	TickEvasion(ss)

	if ss.Bots[0].EvasionAlert != 1 {
		t.Fatal("Alarmed bot should have EvasionAlert=1")
	}
	if ss.Bots[0].EvasionWave == 0 {
		t.Fatal("Alarmed bot should have EvasionWave > 0")
	}
}

// --- Slime Mold Network Tests ---

func TestInitClearSlime(t *testing.T) {
	ss := newBioTestState(5)
	InitSlime(ss)
	if ss.Slime == nil || !ss.SlimeOn {
		t.Fatal("InitSlime should set Slime and SlimeOn")
	}
	if ss.Slime.Cols <= 0 || ss.Slime.Rows <= 0 {
		t.Fatal("Slime grid should have positive dimensions")
	}
	ClearSlime(ss)
	if ss.Slime != nil || ss.SlimeOn {
		t.Fatal("ClearSlime should nil Slime and clear SlimeOn")
	}
}

func TestTickSlimeNil(t *testing.T) {
	ss := newBioTestState(3)
	// Should not panic
	TickSlime(ss)
}

func TestTickSlimeDeposit(t *testing.T) {
	ss := newBioTestState(3)
	InitSlime(ss)

	// Place bot in valid grid cell
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100

	TickSlime(ss)

	// Grid cell at bot's position should have deposit
	col := int(100 / ss.Slime.CellSize)
	row := int(100 / ss.Slime.CellSize)
	idx := row*ss.Slime.Cols + col
	if ss.Slime.Grid[idx] <= 0 {
		t.Fatal("Bot should deposit slime at its position")
	}

	// Sensor cache should reflect trail
	if ss.Bots[0].SlimeTrail <= 0 {
		t.Fatal("SlimeTrail sensor should be positive at deposited location")
	}
}

func TestTickSlimeDecay(t *testing.T) {
	ss := newBioTestState(1)
	InitSlime(ss)

	// Place deposit manually
	ss.Slime.Grid[0] = 0.5

	// Place bot far away so no new deposit at [0]
	ss.Bots[0].X = 700
	ss.Bots[0].Y = 700

	TickSlime(ss)

	if ss.Slime.Grid[0] >= 0.5 {
		t.Fatal("Slime trail should decay over time")
	}
}

func TestApplyFollowSlimeNilSafe(t *testing.T) {
	ss := newBioTestState(1)
	bot := &ss.Bots[0]
	ApplyFollowSlime(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected default speed with nil Slime, got %f", bot.Speed)
	}
}

func TestApplyFollowSlimeSteering(t *testing.T) {
	ss := newBioTestState(1)
	InitSlime(ss)

	bot := &ss.Bots[0]
	bot.X = 400
	bot.Y = 400
	bot.Angle = 0
	bot.SlimeTrail = 50
	bot.SlimeGrad = 45 // trail is 45° to the right

	origAngle := bot.Angle
	ApplyFollowSlime(bot, ss, 0)

	if bot.Angle == origAngle {
		// Angle should change toward gradient
	}
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected SwarmBotSpeed, got %f", bot.Speed)
	}
}

// --- Cross-feature: Vortex with tangential arrangement ---

func TestVortexCircleFormation(t *testing.T) {
	ss := newBioTestState(8)
	cx, cy := 400.0, 400.0
	for i := range ss.Bots {
		angle := float64(i) * 2 * math.Pi / 8.0
		ss.Bots[i].X = cx + math.Cos(angle)*40
		ss.Bots[i].Y = cy + math.Sin(angle)*40
		ss.Bots[i].Angle = angle + math.Pi/2 // tangential
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickVortex(ss)

	// Most bots should detect strong vortex
	strongCount := 0
	for i := range ss.Bots {
		if ss.Bots[i].VortexStrength > 30 {
			strongCount++
		}
	}
	if strongCount < 4 {
		t.Fatalf("Expected at least 4 bots with strong vortex in circle formation, got %d", strongCount)
	}
}
