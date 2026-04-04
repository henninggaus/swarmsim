package pheromone

import (
	"math"
	"testing"
)

func TestNewPheromoneGrid(t *testing.T) {
	g := NewPheromoneGrid(100, 100, 10, 0.99, 0.01)
	if g == nil {
		t.Fatal("NewPheromoneGrid returned nil")
	}
	if g.Cols != 10 {
		t.Errorf("expected 10 cols, got %d", g.Cols)
	}
	if g.Rows != 10 {
		t.Errorf("expected 10 rows, got %d", g.Rows)
	}
	if g.CellSize != 10 {
		t.Errorf("expected CellSize 10, got %f", g.CellSize)
	}
	if g.Decay != 0.99 {
		t.Errorf("expected Decay 0.99, got %f", g.Decay)
	}
	if g.Diffusion != 0.01 {
		t.Errorf("expected Diffusion 0.01, got %f", g.Diffusion)
	}
	for pt := 0; pt < PherCount; pt++ {
		if len(g.Data[pt]) != 100 {
			t.Errorf("expected 100 cells for pheromone type %d, got %d", pt, len(g.Data[pt]))
		}
	}
}

func TestNewPheromoneGrid_NonAlignedDimensions(t *testing.T) {
	g := NewPheromoneGrid(95, 75, 10, 0.99, 0.01)
	if g.Cols != 10 { // ceil(95/10) = 10
		t.Errorf("expected 10 cols, got %d", g.Cols)
	}
	if g.Rows != 8 { // ceil(75/10) = 8
		t.Errorf("expected 8 rows, got %d", g.Rows)
	}
}

func TestDeposit(t *testing.T) {
	g := NewPheromoneGrid(100, 100, 10, 0.99, 0.01)

	g.Deposit(15, 25, PherSearch, 0.5)
	val := g.Get(15, 25, PherSearch)
	if val != 0.5 {
		t.Errorf("expected 0.5, got %f", val)
	}

	// Other types should remain zero.
	if g.Get(15, 25, PherFoundResource) != 0 {
		t.Error("expected zero for untouched pheromone type")
	}
}

func TestDeposit_ClampToOne(t *testing.T) {
	g := NewPheromoneGrid(100, 100, 10, 0.99, 0.01)

	g.Deposit(15, 25, PherSearch, 0.8)
	g.Deposit(15, 25, PherSearch, 0.5)
	val := g.Get(15, 25, PherSearch)
	if val != 1.0 {
		t.Errorf("expected 1.0 (clamped), got %f", val)
	}
}

func TestGet_EmptyGrid(t *testing.T) {
	g := NewPheromoneGrid(100, 100, 10, 0.99, 0.01)
	for pt := 0; pt < PherCount; pt++ {
		if g.Get(50, 50, PheromoneType(pt)) != 0 {
			t.Errorf("expected 0 for empty grid, pheromone type %d", pt)
		}
	}
}

func TestGetCell(t *testing.T) {
	g := NewPheromoneGrid(100, 100, 10, 0.99, 0.01)
	g.Deposit(15, 25, PherDanger, 0.7)
	// world (15,25) maps to cell (1,2)
	val := g.GetCell(1, 2, PherDanger)
	if val != 0.7 {
		t.Errorf("expected 0.7, got %f", val)
	}
}

func TestUpdate_Evaporation(t *testing.T) {
	// Use zero diffusion to isolate evaporation behavior.
	decay := 0.5
	g := NewPheromoneGrid(100, 100, 10, decay, 0.0)

	g.Deposit(55, 55, PherSearch, 1.0)
	g.Update()

	val := g.Get(55, 55, PherSearch)
	if math.Abs(val-decay) > 0.01 {
		t.Errorf("expected ~%f after one evaporation step, got %f", decay, val)
	}

	// After many updates the value should approach zero.
	for i := 0; i < 50; i++ {
		g.Update()
	}
	val = g.Get(55, 55, PherSearch)
	if val > 0.01 {
		t.Errorf("expected near-zero after many evaporation steps, got %f", val)
	}
}

func TestUpdate_Diffusion(t *testing.T) {
	g := NewPheromoneGrid(100, 100, 10, 0.99, 0.1)

	// Deposit in center cell (5,5).
	g.Deposit(55, 55, PherFoundResource, 1.0)

	// Run several update steps so diffusion spreads outward.
	for i := 0; i < 10; i++ {
		g.Update()
	}

	center := g.GetCell(5, 5, PherFoundResource)
	if center <= 0 {
		t.Error("center should still have pheromone")
	}

	// At least one neighbor should have received some pheromone after
	// multiple diffusion steps.
	totalNeighbor := g.GetCell(4, 5, PherFoundResource) +
		g.GetCell(6, 5, PherFoundResource) +
		g.GetCell(5, 4, PherFoundResource) +
		g.GetCell(5, 6, PherFoundResource)
	if totalNeighbor <= 0 {
		t.Error("expected diffusion to spread pheromone to at least one neighbor")
	}
}

func TestClear(t *testing.T) {
	g := NewPheromoneGrid(100, 100, 10, 0.99, 0.01)
	g.Deposit(15, 25, PherSearch, 0.5)
	g.Deposit(45, 55, PherDanger, 0.8)
	g.Clear()

	for pt := 0; pt < PherCount; pt++ {
		for _, v := range g.Data[pt] {
			if v != 0 {
				t.Fatal("expected all zeros after Clear")
			}
		}
	}
}

func TestDeposit_EdgePositions(t *testing.T) {
	g := NewPheromoneGrid(100, 100, 10, 0.99, 0.01)

	// Deposit at origin.
	g.Deposit(0, 0, PherSearch, 0.3)
	if g.Get(0, 0, PherSearch) != 0.3 {
		t.Error("failed to deposit at origin")
	}

	// Deposit at far edge (should clamp to last cell).
	g.Deposit(99, 99, PherDanger, 0.4)
	if g.Get(99, 99, PherDanger) != 0.4 {
		t.Error("failed to deposit at far edge")
	}
}
