package swarm

import (
	"math"
	"math/rand"
	"swarmsim/domain/physics"
	"testing"
)

// newEmergentTestState creates a SwarmState with SpatialHash and Rng initialized.
func newEmergentTestState(numBots int) *SwarmState {
	ss := newFlockTestState(numBots)
	ss.Rng = rand.New(rand.NewSource(42))
	return ss
}

// --- Ant Bridge Tests ---

func TestInitClearBridge(t *testing.T) {
	ss := newEmergentTestState(5)
	InitBridge(ss)
	if ss.Bridge == nil || !ss.BridgeOn {
		t.Fatal("InitBridge should set Bridge and BridgeOn")
	}
	ClearBridge(ss)
	if ss.Bridge != nil || ss.BridgeOn {
		t.Fatal("ClearBridge should nil Bridge and clear BridgeOn")
	}
}

func TestTickBridgeNil(t *testing.T) {
	ss := newEmergentTestState(3)
	// Should not panic
	TickBridge(ss)
}

func TestApplyFormBridgeNearObstacle(t *testing.T) {
	ss := newEmergentTestState(3)
	InitBridge(ss)
	bot := &ss.Bots[0]
	bot.X = 100
	bot.Y = 100
	bot.Angle = 0
	bot.ObstacleAhead = true
	bot.ObstacleDist = 20

	ApplyFormBridge(bot, ss, 0)

	if !ss.Bridge.InBridge[0] {
		t.Fatal("Bot near obstacle should join bridge")
	}
	if bot.Speed != 0 {
		t.Fatal("Bridge bot should be stationary")
	}
}

func TestBridgeChainExtension(t *testing.T) {
	ss := newEmergentTestState(3)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitBridge(ss)

	// Place bots close together
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100
	ss.Bots[1].X = 115 // within bridgeLockDist*1.5
	ss.Bots[1].Y = 100
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Bot 0 is already in bridge
	ss.Bridge.InBridge[0] = true
	ss.Bridge.ChainID[0] = 0
	ss.Bridge.ChainPos[0] = 0
	ss.Bridge.NextChain = 1

	// Bot 1 tries to form bridge → should extend chain
	ApplyFormBridge(&ss.Bots[1], ss, 1)

	if !ss.Bridge.InBridge[1] {
		t.Fatal("Bot near existing bridge should extend chain")
	}
	if ss.Bridge.ChainID[1] != 0 {
		t.Fatalf("Expected chain ID 0, got %d", ss.Bridge.ChainID[1])
	}
	if ss.Bridge.ChainPos[1] != 1 {
		t.Fatalf("Expected chain pos 1, got %d", ss.Bridge.ChainPos[1])
	}
}

func TestBridgeSensorCache(t *testing.T) {
	ss := newEmergentTestState(3)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitBridge(ss)

	ss.Bridge.InBridge[0] = true
	ss.Bridge.ChainPos[0] = 2

	for i := range ss.Bots {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickBridge(ss)

	if ss.Bots[0].BridgeActive != 1 {
		t.Fatal("Bridge bot should have BridgeActive=1")
	}
	if ss.Bots[0].BridgePos != 2 {
		t.Fatalf("Expected BridgePos=2, got %d", ss.Bots[0].BridgePos)
	}
}

func TestApplyCrossBridgeNilSafe(t *testing.T) {
	ss := newEmergentTestState(1)
	bot := &ss.Bots[0]
	ApplyCrossBridge(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected SwarmBotSpeed with nil Bridge, got %f", bot.Speed)
	}
}

// --- Shape Formation Tests (existing system integration) ---

func TestShapeFormationSensorCache(t *testing.T) {
	ss := newEmergentTestState(5)
	// We test that ShapeDist/ShapeAngle/ShapeProgress fields exist and work
	bot := &ss.Bots[0]
	bot.ShapeDist = 100
	bot.ShapeAngle = 45
	bot.ShapeProgress = 80
	if bot.ShapeDist != 100 || bot.ShapeAngle != 45 || bot.ShapeProgress != 80 {
		t.Fatal("Shape sensor cache fields should be readable")
	}
}

// --- Mexican Wave Tests ---

func TestInitClearWave(t *testing.T) {
	ss := newEmergentTestState(5)
	InitWave(ss)
	if ss.Wave == nil || !ss.WaveOn {
		t.Fatal("InitWave should set Wave and WaveOn")
	}
	ClearWave(ss)
	if ss.Wave != nil || ss.WaveOn {
		t.Fatal("ClearWave should nil Wave and clear WaveOn")
	}
}

func TestTickWaveNil(t *testing.T) {
	ss := newEmergentTestState(3)
	// Should not panic
	TickWave(ss)
}

func TestTickWaveAdvancesPhase(t *testing.T) {
	ss := newEmergentTestState(5)
	InitWave(ss)

	for i := range ss.Bots {
		ss.Bots[i].X = float64(i) * 160
		ss.Bots[i].Y = 400
	}
	ss.Tick = 1

	oldPhase := ss.Wave.Phase
	TickWave(ss)

	if ss.Wave.Phase <= oldPhase {
		t.Fatal("Wave phase should advance each tick")
	}
}

func TestWaveFlashPropagation(t *testing.T) {
	ss := newEmergentTestState(10)
	InitWave(ss)
	ss.Wave.Mode = WaveLinear

	// Spread bots across arena
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i) * (ss.ArenaW / 10)
		ss.Bots[i].Y = 400
	}

	// Run enough ticks for wave to sweep across
	flashed := make([]bool, len(ss.Bots))
	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		TickWave(ss)
		for i := range ss.Bots {
			if ss.Bots[i].WaveFlash == 1 {
				flashed[i] = true
			}
		}
	}

	// Most bots should have flashed at some point
	count := 0
	for _, f := range flashed {
		if f {
			count++
		}
	}
	if count < 5 {
		t.Fatalf("Expected at least 5 bots to flash during wave sweep, got %d", count)
	}
}

func TestApplyWaveFlashNilSafe(t *testing.T) {
	ss := newEmergentTestState(1)
	bot := &ss.Bots[0]
	ApplyWaveFlash(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected SwarmBotSpeed with nil Wave, got %f", bot.Speed)
	}
}

func TestCycleWaveMode(t *testing.T) {
	ss := newEmergentTestState(3)
	InitWave(ss)

	if ss.Wave.Mode != WaveLinear {
		t.Fatal("Initial mode should be WaveLinear")
	}
	CycleWaveMode(ss)
	if ss.Wave.Mode != WaveRadial {
		t.Fatal("After cycle, mode should be WaveRadial")
	}
	CycleWaveMode(ss)
	if ss.Wave.Mode != WaveSpiral {
		t.Fatal("After second cycle, mode should be WaveSpiral")
	}
}

// --- Shepherd-Flock Tests ---

func TestInitClearShepherd(t *testing.T) {
	ss := newEmergentTestState(50)
	InitShepherd(ss)
	if ss.Shepherd == nil || !ss.ShepherdOn {
		t.Fatal("InitShepherd should set Shepherd and ShepherdOn")
	}
	// At least 1 shepherd
	shepherdCount := 0
	for _, s := range ss.Shepherd.IsShepherd {
		if s {
			shepherdCount++
		}
	}
	if shepherdCount < 1 {
		t.Fatal("Should have at least 1 shepherd")
	}
	ClearShepherd(ss)
	if ss.Shepherd != nil || ss.ShepherdOn {
		t.Fatal("ClearShepherd should nil Shepherd and clear ShepherdOn")
	}
}

func TestTickShepherdNil(t *testing.T) {
	ss := newEmergentTestState(3)
	// Should not panic
	TickShepherd(ss)
}

func TestTickShepherdComputesFlockCenter(t *testing.T) {
	ss := newEmergentTestState(5)
	InitShepherd(ss)

	// Place non-shepherd bots in a known pattern
	for i := range ss.Bots {
		ss.Bots[i].X = 200
		ss.Bots[i].Y = 300
	}
	// First bot is shepherd
	ss.Bots[0].X = 500
	ss.Bots[0].Y = 500

	TickShepherd(ss)

	// Flock center should be around (200, 300)
	if math.Abs(ss.Shepherd.FlockCX-200) > 1 {
		t.Fatalf("Expected FlockCX≈200, got %.1f", ss.Shepherd.FlockCX)
	}
	if math.Abs(ss.Shepherd.FlockCY-300) > 1 {
		t.Fatalf("Expected FlockCY≈300, got %.1f", ss.Shepherd.FlockCY)
	}
}

func TestShepherdSensorCache(t *testing.T) {
	ss := newEmergentTestState(5)
	InitShepherd(ss)

	for i := range ss.Bots {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
	}

	TickShepherd(ss)

	// First bot is shepherd
	if ss.Bots[0].ShepherdRole != 1 {
		t.Fatal("Shepherd bot should have ShepherdRole=1")
	}
	if ss.Bots[1].ShepherdRole != 0 {
		t.Fatal("Non-shepherd bot should have ShepherdRole=0")
	}
}

func TestApplyShepherdNilSafe(t *testing.T) {
	ss := newEmergentTestState(1)
	bot := &ss.Bots[0]
	ApplyShepherd(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected SwarmBotSpeed with nil Shepherd, got %f", bot.Speed)
	}
}

func TestShepherdDrivesBots(t *testing.T) {
	ss := newEmergentTestState(5)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitShepherd(ss)

	// Place all flock bots at center
	for i := 1; i < len(ss.Bots); i++ {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = 0
	}
	// Shepherd behind flock (left side, target is right)
	ss.Bots[0].X = 300
	ss.Bots[0].Y = 400
	ss.Bots[0].Angle = 0

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickShepherd(ss)
	ApplyShepherd(&ss.Bots[0], ss, 0)

	// Shepherd should be moving faster than normal
	if ss.Bots[0].Speed <= SwarmBotSpeed {
		t.Fatal("Shepherd should move faster than normal speed")
	}
}
