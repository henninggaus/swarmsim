package swarm

import (
	"math"
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeACOState(n int) *SwarmState {
	ss := &SwarmState{
		Bots:   make([]SwarmBot, n),
		ArenaW: 800,
		ArenaH: 800,
		Rng:    rand.New(rand.NewSource(42)),
		Hash:   physics.NewSpatialHash(800, 800, 30),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = 100 + ss.Rng.Float64()*600
		ss.Bots[i].Y = 100 + ss.Rng.Float64()*600
		ss.Bots[i].Angle = ss.Rng.Float64() * 2 * math.Pi
		ss.Bots[i].Energy = 80
		ss.Bots[i].CarryingPkg = -1
	}
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}
	return ss
}

func TestInitACO(t *testing.T) {
	ss := makeACOState(10)
	InitACO(ss)
	if ss.ACO == nil {
		t.Fatal("ACO state should not be nil after init")
	}
	if !ss.ACOOn {
		t.Fatal("ACOOn should be true after init")
	}
	st := ss.ACO
	expectedCols := int(800/acoCellSize) + 1
	expectedRows := int(800/acoCellSize) + 1
	if st.GridCols != expectedCols {
		t.Fatalf("expected %d cols, got %d", expectedCols, st.GridCols)
	}
	if st.GridRows != expectedRows {
		t.Fatalf("expected %d rows, got %d", expectedRows, st.GridRows)
	}
	if len(st.Trail) != expectedCols*expectedRows {
		t.Fatalf("trail grid size mismatch: expected %d, got %d", expectedCols*expectedRows, len(st.Trail))
	}
	// All trails should start at zero
	for i, v := range st.Trail {
		if v != 0 {
			t.Fatalf("trail[%d] should be 0 at init, got %f", i, v)
		}
	}
}

func TestClearACO(t *testing.T) {
	ss := makeACOState(10)
	InitACO(ss)
	ClearACO(ss)
	if ss.ACO != nil {
		t.Fatal("ACO should be nil after clear")
	}
	if ss.ACOOn {
		t.Fatal("ACOOn should be false after clear")
	}
}

func TestTickACODeposit(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	// Set bot 0 to be carrying a package
	ss.Bots[0].CarryingPkg = 1
	ss.Bots[0].X = 200
	ss.Bots[0].Y = 200

	TickACO(ss)

	// Cell at bot's position should have pheromone
	col := int(200 / acoCellSize)
	row := int(200 / acoCellSize)
	idx := row*ss.ACO.GridCols + col
	if ss.ACO.Trail[idx] < acoDepositRate*0.9 {
		t.Fatalf("carrying bot should deposit pheromone, got %f", ss.ACO.Trail[idx])
	}
}

func TestTickACONoDepositWhenNotCarrying(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	// No bot is carrying
	TickACO(ss)
	// All trails should be 0 (only evaporation on zeros)
	for _, v := range ss.ACO.Trail {
		if v > 0.01 {
			t.Fatalf("no deposit expected when no bot is carrying, got %f", v)
		}
	}
}

func TestTickACOEvaporation(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	// Manually deposit pheromone
	ss.ACO.Trail[50] = 50.0
	initial := ss.ACO.Trail[50]

	TickACO(ss)

	// Should have evaporated
	if ss.ACO.Trail[50] >= initial {
		t.Fatal("pheromone should evaporate over time")
	}
	expected := initial * acoEvapRate
	if math.Abs(ss.ACO.Trail[50]-expected) > 0.1 {
		t.Fatalf("expected ~%f after evaporation, got %f", expected, ss.ACO.Trail[50])
	}
}

func TestTickACOPheromoneCapAtMax(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	// Set bot to carry and deposit repeatedly at same cell
	ss.Bots[0].CarryingPkg = 1
	ss.Bots[0].X = 200
	ss.Bots[0].Y = 200

	for i := 0; i < 1000; i++ {
		TickACO(ss)
	}

	col := int(200 / acoCellSize)
	row := int(200 / acoCellSize)
	idx := row*ss.ACO.GridCols + col
	if ss.ACO.Trail[idx] > acoMaxPheromone {
		t.Fatalf("pheromone should be capped at %f, got %f", acoMaxPheromone, ss.ACO.Trail[idx])
	}
}

func TestTickACOReinforceOnDelivery(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	// Simulate a delivery (TimeSinceDelivery == 1)
	ss.Bots[0].TimeSinceDelivery = 1
	ss.Bots[0].X = 400
	ss.Bots[0].Y = 400

	TickACO(ss)

	// Cells around delivery point should have reinforced pheromone
	col := int(400 / acoCellSize)
	row := int(400 / acoCellSize)
	idx := row*ss.ACO.GridCols + col
	if ss.ACO.Trail[idx] < acoReinforceFactor*0.5 {
		t.Fatalf("delivery should reinforce pheromone, got %f", ss.ACO.Trail[idx])
	}
}

func TestTickACONilSafe(t *testing.T) {
	ss := makeACOState(10)
	// Should not panic when ACO is nil
	TickACO(ss)
}

func TestTickACOSensorCache(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	// Deposit pheromone manually near bot 0
	ss.Bots[0].X = 200
	ss.Bots[0].Y = 200
	col := int(200 / acoCellSize)
	row := int(200 / acoCellSize)
	ss.ACO.Trail[row*ss.ACO.GridCols+col] = 50.0

	TickACO(ss)

	// ACOTrail sensor should reflect pheromone
	if ss.Bots[0].ACOTrail <= 0 {
		t.Fatal("ACOTrail sensor should reflect pheromone at bot position")
	}
}

func TestApplyACO(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	// Create a pheromone gradient: strong to the right of bot 0
	ss.Bots[0].X = 200
	ss.Bots[0].Y = 200
	ss.Bots[0].Angle = 0 // facing right
	col := int(200 / acoCellSize)
	row := int(200 / acoCellSize)
	// Place strong pheromone one cell to the right
	if col+1 < ss.ACO.GridCols {
		ss.ACO.Trail[row*ss.ACO.GridCols+(col+1)] = 50.0
	}

	bot := &ss.Bots[0]
	ApplyACO(bot, ss, 0)

	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("expected speed %f, got %f", SwarmBotSpeed, bot.Speed)
	}
}

func TestApplyACONilState(t *testing.T) {
	ss := makeACOState(5)
	bot := &ss.Bots[0]
	ApplyACO(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("should default to SwarmBotSpeed when ACO is nil, got %f", bot.Speed)
	}
}

func TestApplyACONoTrail(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	// No pheromone anywhere
	bot := &ss.Bots[0]
	bot.X = 400
	bot.Y = 400
	ApplyACO(bot, ss, 0)
	// Should still move (wander mode)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("bot should wander at SwarmBotSpeed when no trail, got %f", bot.Speed)
	}
}

func TestApplyACOLEDColor(t *testing.T) {
	ss := makeACOState(5)
	InitACO(ss)
	ss.Bots[0].X = 200
	ss.Bots[0].Y = 200
	col := int(200 / acoCellSize)
	row := int(200 / acoCellSize)
	ss.ACO.Trail[row*ss.ACO.GridCols+col] = 80.0

	ApplyACO(&ss.Bots[0], ss, 0)

	// LED should have some color set (orange tint for trails)
	led := ss.Bots[0].LEDColor
	if led[0] == 0 && led[1] == 0 && led[2] == 0 {
		t.Fatal("LED should be set after ApplyACO")
	}
}
