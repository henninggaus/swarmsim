package swarm

import (
	"math"
	"math/rand"
	"swarmsim/domain/physics"
	"testing"
)

// newBatch6TestState creates a SwarmState with SpatialHash and Rng initialized.
func newBatch6TestState(numBots int) *SwarmState {
	ss := newFlockTestState(numBots)
	ss.Rng = rand.New(rand.NewSource(42))
	return ss
}

// --- Gravity Tests ---

func TestInitClearGravity(t *testing.T) {
	ss := newBatch6TestState(10)
	InitGravity(ss)
	if ss.Gravity == nil || !ss.GravityOn {
		t.Fatal("InitGravity should set Gravity and GravityOn")
	}
	// Should have heavy bots
	heavyCount := 0
	for _, m := range ss.Gravity.Mass {
		if m >= 3.0*0.8 {
			heavyCount++
		}
	}
	if heavyCount < 1 {
		t.Fatal("Should have at least 1 heavy bot")
	}
	ClearGravity(ss)
	if ss.Gravity != nil || ss.GravityOn {
		t.Fatal("ClearGravity should nil Gravity and clear GravityOn")
	}
}

func TestTickGravityNil(t *testing.T) {
	ss := newBatch6TestState(3)
	TickGravity(ss) // should not panic
}

func TestGravityForceComputation(t *testing.T) {
	ss := newBatch6TestState(3)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitGravity(ss)

	// Place heavy bot at center, light bot nearby
	ss.Gravity.Mass[0] = 3.0
	ss.Gravity.Mass[1] = 0.5
	ss.Bots[0].X, ss.Bots[0].Y = 400, 400
	ss.Bots[1].X, ss.Bots[1].Y = 450, 400
	ss.Bots[2].X, ss.Bots[2].Y = 100, 100 // far away

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickGravity(ss)

	// Bot 1 should have force pointing toward bot 0 (negative X direction)
	if ss.Gravity.ForceX[1] >= 0 {
		t.Fatal("Light bot should be attracted toward heavy bot (negative X)")
	}
}

func TestGravitySensorCache(t *testing.T) {
	ss := newBatch6TestState(5)
	InitGravity(ss)
	TickGravity(ss)

	// GravMass should be set
	if ss.Bots[0].GravMass <= 0 {
		t.Fatal("GravMass should be positive")
	}
}

func TestApplyGravityNilSafe(t *testing.T) {
	ss := newBatch6TestState(1)
	bot := &ss.Bots[0]
	ApplyGravity(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil Gravity")
	}
}

// --- Crystal Tests ---

func TestInitClearCrystal(t *testing.T) {
	ss := newBatch6TestState(10)
	InitCrystal(ss)
	if ss.Crystal == nil || !ss.CrystalOn {
		t.Fatal("InitCrystal should set Crystal and CrystalOn")
	}
	ClearCrystal(ss)
	if ss.Crystal != nil || ss.CrystalOn {
		t.Fatal("ClearCrystal should nil Crystal and clear CrystalOn")
	}
}

func TestTickCrystalNil(t *testing.T) {
	ss := newBatch6TestState(3)
	TickCrystal(ss) // should not panic
}

func TestCrystalNeighborDetection(t *testing.T) {
	ss := newBatch6TestState(7)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitCrystal(ss)

	// Place 7 bots in hexagonal arrangement (center + 6 neighbors)
	cx, cy := 400.0, 400.0
	ss.Bots[0].X, ss.Bots[0].Y = cx, cy
	spacing := 22.0
	for k := 0; k < 6; k++ {
		angle := float64(k) * math.Pi / 3
		ss.Bots[k+1].X = cx + spacing*math.Cos(angle)
		ss.Bots[k+1].Y = cy + spacing*math.Sin(angle)
	}

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickCrystal(ss)

	// Center bot should have 6 neighbors
	if ss.Crystal.NeighCount[0] != 6 {
		t.Fatalf("Center bot should have 6 neighbors, got %d", ss.Crystal.NeighCount[0])
	}
	// Should be settled
	if !ss.Crystal.Settled[0] {
		t.Fatal("Center bot with 6 neighbors should be settled")
	}
}

func TestCrystalSensorCache(t *testing.T) {
	ss := newBatch6TestState(5)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitCrystal(ss)

	// Place all bots close together
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*20
		ss.Bots[i].Y = 400
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickCrystal(ss)

	// CrystalNeigh should be set for at least one bot
	hasNeigh := false
	for i := range ss.Bots {
		if ss.Bots[i].CrystalNeigh > 0 {
			hasNeigh = true
		}
	}
	if !hasNeigh {
		t.Fatal("At least one bot should have CrystalNeigh > 0")
	}
}

func TestApplyCrystalNilSafe(t *testing.T) {
	ss := newBatch6TestState(1)
	bot := &ss.Bots[0]
	ApplyCrystal(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil Crystal")
	}
}

// --- Amoeba Tests ---

func TestInitClearAmoeba(t *testing.T) {
	ss := newBatch6TestState(10)
	InitAmoeba(ss)
	if ss.Amoeba == nil || !ss.AmoebaOn {
		t.Fatal("InitAmoeba should set Amoeba and AmoebaOn")
	}
	ClearAmoeba(ss)
	if ss.Amoeba != nil || ss.AmoebaOn {
		t.Fatal("ClearAmoeba should nil Amoeba and clear AmoebaOn")
	}
}

func TestTickAmoebaNil(t *testing.T) {
	ss := newBatch6TestState(3)
	TickAmoeba(ss) // should not panic
}

func TestAmoebaCenterComputation(t *testing.T) {
	ss := newBatch6TestState(4)
	InitAmoeba(ss)

	ss.Bots[0].X, ss.Bots[0].Y = 200, 200
	ss.Bots[1].X, ss.Bots[1].Y = 400, 200
	ss.Bots[2].X, ss.Bots[2].Y = 200, 400
	ss.Bots[3].X, ss.Bots[3].Y = 400, 400

	TickAmoeba(ss)

	if math.Abs(ss.Amoeba.CenterX-300) > 1 || math.Abs(ss.Amoeba.CenterY-300) > 1 {
		t.Fatalf("Center should be (300,300), got (%.1f, %.1f)", ss.Amoeba.CenterX, ss.Amoeba.CenterY)
	}
}

func TestAmoebaSkinDetection(t *testing.T) {
	ss := newBatch6TestState(10)
	InitAmoeba(ss)

	// Spread bots: some near center, some far
	for i := 0; i < 5; i++ {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
	}
	for i := 5; i < 10; i++ {
		ss.Bots[i].X = 400 + float64(i)*50
		ss.Bots[i].Y = 400
	}

	TickAmoeba(ss)

	// Far bots should be skin
	hasSkin := false
	for i := 5; i < 10; i++ {
		if ss.Bots[i].AmoebaSkin == 1 {
			hasSkin = true
		}
	}
	if !hasSkin {
		t.Fatal("Far bots should be detected as skin/membrane")
	}
}

func TestApplyAmoebaNilSafe(t *testing.T) {
	ss := newBatch6TestState(1)
	bot := &ss.Bots[0]
	ApplyAmoeba(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil Amoeba")
	}
}

// --- ACO Tests ---

func TestInitClearACO(t *testing.T) {
	ss := newBatch6TestState(5)
	InitACO(ss)
	if ss.ACO == nil || !ss.ACOOn {
		t.Fatal("InitACO should set ACO and ACOOn")
	}
	if ss.ACO.GridCols < 1 || ss.ACO.GridRows < 1 {
		t.Fatal("ACO grid should have positive dimensions")
	}
	ClearACO(ss)
	if ss.ACO != nil || ss.ACOOn {
		t.Fatal("ClearACO should nil ACO and clear ACOOn")
	}
}

func TestTickACONil(t *testing.T) {
	ss := newBatch6TestState(3)
	TickACO(ss) // should not panic
}

func TestACOPheromoneDeposit(t *testing.T) {
	ss := newBatch6TestState(3)
	InitACO(ss)

	// Bot 0 carries a package at position (100, 100)
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100
	ss.Bots[0].CarryingPkg = 0

	TickACO(ss)

	// Check pheromone at that grid cell
	col := int(100 / 10)
	row := int(100 / 10)
	idx := row*ss.ACO.GridCols + col
	if ss.ACO.Trail[idx] < 1 {
		t.Fatal("Carrying bot should deposit pheromone")
	}
}

func TestACOEvaporation(t *testing.T) {
	ss := newBatch6TestState(1)
	InitACO(ss)

	// Manually set high pheromone
	ss.ACO.Trail[0] = 50.0

	// Tick without any carrying bot
	ss.Bots[0].CarryingPkg = -1
	TickACO(ss)

	if ss.ACO.Trail[0] >= 50.0 {
		t.Fatal("Pheromone should evaporate")
	}
}

func TestACOSensorCache(t *testing.T) {
	ss := newBatch6TestState(3)
	InitACO(ss)

	// Deposit some pheromone
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100
	ss.Bots[0].CarryingPkg = 0
	TickACO(ss)

	// Bot at that location should see trail
	if ss.Bots[0].ACOTrail < 1 {
		t.Fatal("Bot at pheromone location should have ACOTrail > 0")
	}
}

func TestApplyACONilSafe(t *testing.T) {
	ss := newBatch6TestState(1)
	bot := &ss.Bots[0]
	ApplyACO(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil ACO")
	}
}
