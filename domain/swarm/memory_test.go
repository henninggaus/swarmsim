package swarm

import (
	"math"
	"math/rand"
	"testing"
)

// === InitBotMemory tests ===

func TestInitBotMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitBotMemory(ss)

	expectedCols := int(ss.ArenaW) / MemoryCellSize // 800/40 = 20
	expectedRows := int(ss.ArenaH) / MemoryCellSize // 800/40 = 20

	for i, bot := range ss.Bots {
		if bot.MemoryCols != expectedCols {
			t.Errorf("bot %d: expected %d cols, got %d", i, expectedCols, bot.MemoryCols)
		}
		if bot.MemoryRows != expectedRows {
			t.Errorf("bot %d: expected %d rows, got %d", i, expectedRows, bot.MemoryRows)
		}
		if len(bot.MemoryGrid) != expectedCols*expectedRows {
			t.Errorf("bot %d: grid size %d != %d", i, len(bot.MemoryGrid), expectedCols*expectedRows)
		}
		// All cells should start at 0
		for _, v := range bot.MemoryGrid {
			if v != 0 {
				t.Errorf("bot %d: initial grid cell not zero", i)
				break
			}
		}
	}
}

// === UpdateBotMemory tests ===

func TestUpdateBotMemoryIncrementsCell(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	// Place bot at known position
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100

	UpdateBotMemory(ss)

	visits := BotVisitedHere(&ss.Bots[0])
	if visits != 1 {
		t.Errorf("expected 1 visit after one update, got %d", visits)
	}

	// Update again
	UpdateBotMemory(ss)
	visits = BotVisitedHere(&ss.Bots[0])
	if visits != 2 {
		t.Errorf("expected 2 visits after two updates, got %d", visits)
	}
}

func TestUpdateBotMemorySaturationAt255(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100

	// Fill to max
	for i := 0; i < 300; i++ {
		UpdateBotMemory(ss)
	}

	visits := BotVisitedHere(&ss.Bots[0])
	if visits != 255 {
		t.Errorf("visits should saturate at 255, got %d", visits)
	}
}

func TestUpdateBotMemoryNilGrid(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	// Don't init memory — grid is nil
	// Should not panic
	UpdateBotMemory(ss)
}

func TestUpdateBotMemoryEdgePositions(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	// Test corners
	testPositions := [][2]float64{
		{0, 0},
		{ss.ArenaW - 1, 0},
		{0, ss.ArenaH - 1},
		{ss.ArenaW - 1, ss.ArenaH - 1},
	}

	for _, pos := range testPositions {
		ss.Bots[0].X = pos[0]
		ss.Bots[0].Y = pos[1]
		UpdateBotMemory(ss) // should not panic
	}
}

func TestUpdateBotMemoryNegativePosition(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	// Negative position should be safely ignored (bounds check in UpdateBotMemory)
	ss.Bots[0].X = -10
	ss.Bots[0].Y = -10
	UpdateBotMemory(ss) // should not panic

	visits := BotVisitedHere(&ss.Bots[0])
	if visits != 0 {
		t.Errorf("negative position should not register a visit, got %d", visits)
	}
}

// === BotVisitedHere tests ===

func TestBotVisitedHereNilGrid(t *testing.T) {
	bot := &SwarmBot{}
	visits := BotVisitedHere(bot)
	if visits != 0 {
		t.Errorf("nil grid should return 0, got %d", visits)
	}
}

func TestBotVisitedHereOutOfBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	ss.Bots[0].X = 9999
	ss.Bots[0].Y = 9999
	visits := BotVisitedHere(&ss.Bots[0])
	if visits != 0 {
		t.Errorf("out-of-bounds should return 0, got %d", visits)
	}
}

// === BotVisitedAhead tests ===

func TestBotVisitedAhead(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	// Bot at (200, 200), facing right (angle=0)
	ss.Bots[0].X = 200
	ss.Bots[0].Y = 200
	ss.Bots[0].Angle = 0

	// Visit the cell ahead
	aheadX := 200 + math.Cos(0)*float64(MemoryCellSize)
	ss.Bots[0].X = aheadX
	UpdateBotMemory(ss)

	// Move back
	ss.Bots[0].X = 200
	visits := BotVisitedAhead(&ss.Bots[0])
	if visits != 1 {
		t.Errorf("expected 1 visit ahead, got %d", visits)
	}
}

func TestBotVisitedAheadNilGrid(t *testing.T) {
	bot := &SwarmBot{X: 100, Y: 100, Angle: 0}
	visits := BotVisitedAhead(bot)
	if visits != 0 {
		t.Errorf("nil grid should return 0, got %d", visits)
	}
}

// === BotExploredPercent tests ===

func TestBotExploredPercentEmpty(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	pct := BotExploredPercent(&ss.Bots[0])
	if pct != 0 {
		t.Errorf("fresh bot should have 0%% explored, got %d%%", pct)
	}
}

func TestBotExploredPercentFullyExplored(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	// Visit every cell
	bot := &ss.Bots[0]
	for r := 0; r < bot.MemoryRows; r++ {
		for c := 0; c < bot.MemoryCols; c++ {
			bot.MemoryGrid[r*bot.MemoryCols+c] = 1
		}
	}

	pct := BotExploredPercent(bot)
	if pct != 100 {
		t.Errorf("fully explored bot should have 100%%, got %d%%", pct)
	}
}

func TestBotExploredPercentHalf(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitBotMemory(ss)

	bot := &ss.Bots[0]
	total := bot.MemoryCols * bot.MemoryRows
	half := total / 2
	for i := 0; i < half; i++ {
		bot.MemoryGrid[i] = 1
	}

	pct := BotExploredPercent(bot)
	expected := half * 100 / total
	if pct != expected {
		t.Errorf("expected %d%% explored, got %d%%", expected, pct)
	}
}

func TestBotExploredPercentNilGrid(t *testing.T) {
	bot := &SwarmBot{}
	pct := BotExploredPercent(bot)
	if pct != 0 {
		t.Errorf("nil grid should return 0%%, got %d%%", pct)
	}
}

// === ClearBotMemory tests ===

func TestClearBotMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitBotMemory(ss)

	// Visit some cells
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100
	UpdateBotMemory(ss)

	ClearBotMemory(ss)

	for i, bot := range ss.Bots {
		if bot.MemoryGrid != nil {
			t.Errorf("bot %d: grid should be nil after clear", i)
		}
		if bot.MemoryCols != 0 || bot.MemoryRows != 0 {
			t.Errorf("bot %d: cols/rows should be 0 after clear", i)
		}
	}
}

// === Distinct cells tracked per bot ===

func TestMemoryPerBotIndependence(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 2)
	InitBotMemory(ss)

	// Bot 0 at (100, 100), Bot 1 at (400, 400)
	ss.Bots[0].X = 100
	ss.Bots[0].Y = 100
	ss.Bots[1].X = 400
	ss.Bots[1].Y = 400

	UpdateBotMemory(ss)

	// Bot 0 should not see visits at Bot 1's position
	ss.Bots[0].X = 400
	ss.Bots[0].Y = 400
	visits0 := BotVisitedHere(&ss.Bots[0])
	if visits0 != 0 {
		t.Errorf("bot 0 should not see bot 1's visits, got %d", visits0)
	}

	visits1 := BotVisitedHere(&ss.Bots[1])
	if visits1 != 1 {
		t.Errorf("bot 1 should see its own visit, got %d", visits1)
	}
}
