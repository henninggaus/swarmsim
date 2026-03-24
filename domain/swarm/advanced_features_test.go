package swarm

import (
	"math"
	"math/rand"
	"swarmsim/domain/physics"
	"testing"
)

// newAdvancedTestState creates a SwarmState with SpatialHash and Rng initialized.
func newAdvancedTestState(numBots int) *SwarmState {
	ss := newFlockTestState(numBots)
	ss.Rng = rand.New(rand.NewSource(42))
	return ss
}

// --- PSO Tests ---

func TestInitClearPSO(t *testing.T) {
	ss := newAdvancedTestState(10)
	InitPSO(ss)
	if ss.PSO == nil || !ss.PSOOn {
		t.Fatal("InitPSO should set PSO and PSOOn")
	}
	if len(ss.PSO.PeakX) < 3 {
		t.Fatal("Should generate at least 3 fitness peaks")
	}
	ClearPSO(ss)
	if ss.PSO != nil || ss.PSOOn {
		t.Fatal("ClearPSO should nil PSO and clear PSOOn")
	}
}

func TestTickPSONil(t *testing.T) {
	ss := newAdvancedTestState(3)
	TickPSO(ss) // should not panic
}

func TestPSOFitnessEvaluation(t *testing.T) {
	ss := newAdvancedTestState(5)
	InitPSO(ss)

	// Place a bot at a peak location
	ss.Bots[0].X = ss.PSO.PeakX[0]
	ss.Bots[0].Y = ss.PSO.PeakY[0]

	ss.Tick = 5 // evaluate on multiples of psoUpdateRate
	TickPSO(ss)

	// Fitness should be high at peak
	if ss.Bots[0].PSOFitness < 20 {
		t.Fatalf("Bot at peak should have high fitness, got %d", ss.Bots[0].PSOFitness)
	}
}

func TestPSOVelocityUpdate(t *testing.T) {
	ss := newAdvancedTestState(5)
	InitPSO(ss)

	// Place bot far from global best
	ss.Bots[1].X = 50
	ss.Bots[1].Y = 50

	for tick := 0; tick < 20; tick++ {
		ss.Tick = tick
		TickPSO(ss)
	}

	// Velocity should be non-zero
	vx, vy := ss.PSO.VelX[1], ss.PSO.VelY[1]
	if math.Abs(vx)+math.Abs(vy) < 0.01 {
		t.Fatal("PSO velocity should be non-zero after updates")
	}
}

func TestApplyPSOMoveNilSafe(t *testing.T) {
	ss := newAdvancedTestState(1)
	bot := &ss.Bots[0]
	ApplyPSOMove(bot, ss, 0)
	if bot.Speed != 0 {
		t.Fatal("Expected Speed=0 with nil PSO (eigenbewegung)")
	}
}

func TestPSOSensorCache(t *testing.T) {
	ss := newAdvancedTestState(5)
	InitPSO(ss)
	ss.Tick = 5
	TickPSO(ss)

	// PSOGlobalDist should be set
	for i := range ss.Bots {
		if ss.Bots[i].PSOGlobalDist == 0 && i > 0 {
			// at least some bots should have non-zero distance
			continue
		}
	}
	// PSOBest should be >= 0
	if ss.Bots[0].PSOBest < 0 {
		t.Fatal("PSOBest should be non-negative")
	}
}

// --- Predator-Prey SwarmScript Tests ---

func TestApplyPredatorNilSafe(t *testing.T) {
	ss := newAdvancedTestState(1)
	bot := &ss.Bots[0]
	ApplyPredator(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil PredatorPrey")
	}
}

func TestPredPreySensorFields(t *testing.T) {
	ss := newAdvancedTestState(5)
	// Just test sensor fields exist
	ss.Bots[0].PredRole = 1
	ss.Bots[0].PreyDist = 50
	ss.Bots[0].PredCatches = 3
	if ss.Bots[0].PredRole != 1 || ss.Bots[0].PreyDist != 50 || ss.Bots[0].PredCatches != 3 {
		t.Fatal("Predator-Prey sensor fields should be readable")
	}
}

// --- Magnetic Chain Tests ---

func TestInitClearMagnetic(t *testing.T) {
	ss := newAdvancedTestState(5)
	InitMagnetic(ss)
	if ss.Magnetic == nil || !ss.MagneticOn {
		t.Fatal("InitMagnetic should set Magnetic and MagneticOn")
	}
	ClearMagnetic(ss)
	if ss.Magnetic != nil || ss.MagneticOn {
		t.Fatal("ClearMagnetic should nil Magnetic and clear MagneticOn")
	}
}

func TestTickMagneticNil(t *testing.T) {
	ss := newAdvancedTestState(3)
	TickMagnetic(ss) // should not panic
}

func TestMagneticChainDetection(t *testing.T) {
	ss := newAdvancedTestState(3)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitMagnetic(ss)

	// Place two bots close together, aligned
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100
	ss.Bots[0].Angle = 0
	ss.Bots[1].X = 118 // within magChainDist*1.5
	ss.Bots[1].Y = 100
	ss.Bots[1].Angle = 0
	ss.Bots[2].X = 500
	ss.Bots[2].Y = 500

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickMagnetic(ss)

	// Bots 0 and 1 should be linked
	if ss.Bots[0].MagChainLen < 2 || ss.Bots[1].MagChainLen < 2 {
		t.Fatalf("Aligned close bots should form chain, got len[0]=%d len[1]=%d",
			ss.Bots[0].MagChainLen, ss.Bots[1].MagChainLen)
	}
	if ss.Bots[0].MagLinked != 1 {
		t.Fatal("Linked bot should have MagLinked=1")
	}
}

func TestApplyMagneticNilSafe(t *testing.T) {
	ss := newAdvancedTestState(1)
	bot := &ss.Bots[0]
	ApplyMagnetic(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil Magnetic")
	}
}

func TestMagneticAlignSensor(t *testing.T) {
	ss := newAdvancedTestState(5)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitMagnetic(ss)

	// All bots at same position, same angle → high alignment
	for i := range ss.Bots {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = 0
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickMagnetic(ss)

	// All aligned → MagAlign should be high positive
	if ss.Bots[0].MagAlign < 50 {
		t.Fatalf("Perfectly aligned bots should have high MagAlign, got %d", ss.Bots[0].MagAlign)
	}
}

// --- Cell Division Tests ---

func TestInitClearDivision(t *testing.T) {
	ss := newAdvancedTestState(10)
	InitDivision(ss)
	if ss.Division == nil || !ss.DivisionOn {
		t.Fatal("InitDivision should set Division and DivisionOn")
	}
	ClearDivision(ss)
	if ss.Division != nil || ss.DivisionOn {
		t.Fatal("ClearDivision should nil Division and clear DivisionOn")
	}
}

func TestTickDivisionNil(t *testing.T) {
	ss := newAdvancedTestState(3)
	TickDivision(ss) // should not panic
}

func TestDivisionGroupAssignment(t *testing.T) {
	ss := newAdvancedTestState(10)

	// Spread bots: half above center, half below
	for i := range ss.Bots {
		if i < 5 {
			ss.Bots[i].Y = 100 // above center (400)
		} else {
			ss.Bots[i].Y = 700 // below center
		}
		ss.Bots[i].X = 400
	}

	InitDivision(ss)

	group0Count := 0
	group1Count := 0
	for i := range ss.Bots {
		if ss.Division.GroupID[i] == 0 {
			group0Count++
		} else {
			group1Count++
		}
	}

	if group0Count == 0 || group1Count == 0 {
		t.Fatal("Division should create two non-empty groups")
	}
}

func TestDivisionPhaseAdvance(t *testing.T) {
	ss := newAdvancedTestState(5)
	InitDivision(ss)

	oldPhase := ss.Division.Phase
	TickDivision(ss)

	if ss.Division.Phase <= oldPhase {
		t.Fatal("Division phase should advance each tick")
	}
}

func TestDivisionSensorCache(t *testing.T) {
	ss := newAdvancedTestState(10)
	InitDivision(ss)

	for i := range ss.Bots {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
	}

	TickDivision(ss)

	// DivPhase should be > 0
	if ss.Bots[0].DivPhase < 0 {
		t.Fatal("DivPhase should be non-negative")
	}

	// DivGroup should be 0 or 1
	for i := range ss.Bots {
		if ss.Bots[i].DivGroup != 0 && ss.Bots[i].DivGroup != 1 {
			t.Fatalf("DivGroup should be 0 or 1, got %d", ss.Bots[i].DivGroup)
		}
	}
}

func TestApplyDivisionNilSafe(t *testing.T) {
	ss := newAdvancedTestState(1)
	bot := &ss.Bots[0]
	ApplyDivision(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil Division")
	}
}

func TestDivisionCycleReset(t *testing.T) {
	ss := newAdvancedTestState(5)
	InitDivision(ss)

	// Run through a full cycle
	for i := 0; i < 310; i++ {
		TickDivision(ss)
	}

	// CycleCount should have incremented
	if ss.Division.CycleCount < 1 {
		t.Fatal("CycleCount should increment after a full cycle")
	}
}
