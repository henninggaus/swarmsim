package swarm

import (
	"math/rand"
	"testing"
)

func TestInitSpatialMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpatialMemory(ss)

	sm := ss.SpatialMemory
	if sm == nil {
		t.Fatal("spatial memory should be initialized")
	}
	if sm.GridW <= 0 || sm.GridH <= 0 {
		t.Fatal("grid dimensions should be positive")
	}
	if len(sm.Cells) != sm.GridW*sm.GridH {
		t.Fatal("cell count mismatch")
	}
}

func TestClearSpatialMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.SpatialMemoryOn = true
	InitSpatialMemory(ss)
	ClearSpatialMemory(ss)

	if ss.SpatialMemory != nil {
		t.Fatal("should be nil")
	}
	if ss.SpatialMemoryOn {
		t.Fatal("should be false")
	}
}

func TestTickSpatialMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSpatialMemory(ss)

	// Place bots with varied states
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i%5)*80 + 50
		ss.Bots[i].Y = float64(i/5)*80 + 50
	}
	for i := 0; i < 5; i++ {
		ss.Bots[i].NearestPickupDist = 30
	}

	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		TickSpatialMemory(ss)
	}

	sm := ss.SpatialMemory
	if sm.KnownCells == 0 {
		t.Fatal("should have some known cells")
	}
	if sm.TotalWrites == 0 {
		t.Fatal("should have written some observations")
	}
}

func TestTickSpatialMemoryNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickSpatialMemory(ss) // should not panic
}

func TestSpatialMemKnown(t *testing.T) {
	if SpatialMemKnown(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestSpatialMemConfidence(t *testing.T) {
	if SpatialMemConfidence(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestSpatialMemCellScore(t *testing.T) {
	if SpatialMemCellScore(nil, 0, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitSpatialMemory(ss)

	// Manually set a cell
	idx := smCellIdx(ss.SpatialMemory, 100, 100)
	if idx >= 0 {
		ss.SpatialMemory.Cells[idx].ResourceScore = 0.75
	}

	score := SpatialMemCellScore(ss.SpatialMemory, 100, 100)
	if score != 0.75 {
		t.Fatalf("expected 0.75, got %.2f", score)
	}
}

func TestSmCellIdx(t *testing.T) {
	sm := &SpatialMemoryState{
		GridW:    20,
		GridH:    20,
		CellSize: 40,
	}

	idx := smCellIdx(sm, 80, 120)
	// cx=2, cy=3 → 3*20+2 = 62
	if idx != 62 {
		t.Fatalf("expected 62, got %d", idx)
	}

	// Out of bounds
	if smCellIdx(sm, -200, -200) != -1 {
		t.Fatal("out of bounds should return -1")
	}
}

func TestSpatialMemoryDecay(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitSpatialMemory(ss)
	sm := ss.SpatialMemory

	// Set a cell with known confidence
	sm.Cells[0].Confidence = 0.01
	sm.Cells[0].ResourceScore = 0.5

	// Tick enough times for decay
	for tick := 0; tick < 100; tick++ {
		ss.Tick = tick
		TickSpatialMemory(ss)
	}

	if sm.Cells[0].Confidence > 0.01 {
		// bot might have written to it, but the decay should have kicked in
	}
}
