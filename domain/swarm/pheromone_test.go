package swarm

import (
	"math"
	"testing"
)

func TestNewSwarmPheromoneGrid(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	if g.Cols != 40 || g.Rows != 40 {
		t.Errorf("expected 40x40 grid, got %dx%d", g.Cols, g.Rows)
	}
	if g.CellSize != 20 {
		t.Errorf("expected cell size 20, got %.0f", g.CellSize)
	}
	if len(g.Data) != 1600 {
		t.Errorf("expected 1600 cells, got %d", len(g.Data))
	}
}

func TestDeposit(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	g.Deposit(100, 100, 0.5)
	val := g.Get(100, 100)
	if val != 0.5 {
		t.Errorf("expected 0.5, got %f", val)
	}
}

func TestDepositCapsAtOne(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	g.Deposit(100, 100, 0.7)
	g.Deposit(100, 100, 0.7)
	val := g.Get(100, 100)
	if val != 1.0 {
		t.Errorf("expected 1.0 (capped), got %f", val)
	}
}

func TestGetEmptyGrid(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	val := g.Get(400, 400)
	if val != 0 {
		t.Errorf("expected 0 for empty grid, got %f", val)
	}
}

func TestGradientEmpty(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	gx, gy := g.Gradient(400, 400)
	if gx != 0 || gy != 0 {
		t.Errorf("expected zero gradient on empty grid, got (%.4f, %.4f)", gx, gy)
	}
}

func TestGradientPointsToward(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	// Deposit pheromone to the right of query point
	g.Deposit(160, 100, 0.8) // cell (8,5)
	gx, gy := g.Gradient(140, 100)
	if gx <= 0 {
		t.Errorf("gradient should point right toward deposit, gx=%.4f", gx)
	}
	_ = gy
}

func TestUpdateDecay(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	g.Deposit(100, 100, 1.0)
	g.Update()
	val := g.Get(100, 100)
	if val >= 1.0 {
		t.Errorf("value should decay below 1.0, got %f", val)
	}
	if val < 0.9 {
		t.Errorf("value should not decay too much in one step, got %f", val)
	}
}

func TestUpdateDiffusion(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	// Place deposit in middle so neighbors exist on all sides
	g.Deposit(400, 400, 1.0)
	g.Update()
	center := g.Get(400, 400)
	// Center should lose some value from diffusion
	if center >= 1.0 {
		t.Error("center should decrease after diffusion")
	}
	if center <= 0 {
		t.Error("center should still have pheromone")
	}
}

func TestClear(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	g.Deposit(100, 100, 0.5)
	g.Deposit(200, 200, 0.8)
	g.Clear()
	if g.Get(100, 100) != 0 || g.Get(200, 200) != 0 {
		t.Error("Clear should reset all values to 0")
	}
}

func TestDepositEdge(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	// Should not panic at edges
	g.Deposit(0, 0, 0.5)
	g.Deposit(799, 799, 0.5)
	g.Deposit(-10, -10, 0.3) // negative coords clamped
	if g.Get(0, 0) < 0.5 {
		t.Error("deposit at origin should work")
	}
}

func TestMultipleUpdatesDecayToZero(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	g.Deposit(400, 400, 0.1)
	for i := 0; i < 1000; i++ {
		g.Update()
	}
	val := g.Get(400, 400)
	if val > 0.001 {
		t.Errorf("value should decay near zero after many updates, got %f", val)
	}
}

func TestGradientCentralDifference(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	// Create a gradient: more pheromone to the right
	g.Deposit(100, 200, 0.2) // cell (5,10)
	g.Deposit(120, 200, 0.6) // cell (6,10)
	g.Deposit(140, 200, 0.9) // cell (7,10)
	gx, _ := g.Gradient(120, 200)
	if gx <= 0 {
		t.Errorf("gradient should point right (increasing pheromone), gx=%.4f", gx)
	}
}

func TestSmallArena(t *testing.T) {
	g := NewSwarmPheromoneGrid(100, 100)
	if g.Cols != 5 || g.Rows != 5 {
		t.Errorf("expected 5x5 grid for 100x100 arena, got %dx%d", g.Cols, g.Rows)
	}
	g.Deposit(50, 50, 0.5)
	g.Update()
	_ = g.Get(50, 50) // should not panic
}

func TestGradientNormalized(t *testing.T) {
	g := NewSwarmPheromoneGrid(800, 800)
	// Deposit far from query to test neighbor search
	g.Deposit(120, 100, 0.5)
	gx, gy := g.Gradient(100, 100) // no pheromone here, should find neighbor
	// Gradient magnitude should be reasonable
	mag := math.Sqrt(gx*gx + gy*gy)
	if mag > 1.0 {
		t.Errorf("gradient magnitude too large: %.4f", mag)
	}
}

// === Multi-channel pheromone tests ===

func TestNewMultiPheromoneGrid(t *testing.T) {
	g := NewMultiPheromoneGrid(800, 800)
	if g.Cols != 40 || g.Rows != 40 {
		t.Errorf("expected 40x40, got %dx%d", g.Cols, g.Rows)
	}
	for ch := 0; ch < PherChannels; ch++ {
		if len(g.Channels[ch]) != 1600 {
			t.Errorf("channel %d: expected 1600 cells, got %d", ch, len(g.Channels[ch]))
		}
	}
}

func TestMultiDepositDirectional(t *testing.T) {
	g := NewMultiPheromoneGrid(800, 800)
	heading := math.Pi / 4 // 45 degrees
	g.DepositDirectional(200, 200, heading, 0.5, PherCarry)

	val := g.GetChannel(200, 200, PherCarry)
	if val < 0.49 || val > 0.51 {
		t.Errorf("expected ~0.5, got %f", val)
	}

	// Other channels should be 0
	if g.GetChannel(200, 200, PherHome) != 0 {
		t.Error("PherHome should be 0")
	}
	if g.GetChannel(200, 200, PherDanger) != 0 {
		t.Error("PherDanger should be 0")
	}
}

func TestMultiGetDirection(t *testing.T) {
	g := NewMultiPheromoneGrid(800, 800)
	// Deposit multiple times with same heading
	for i := 0; i < 5; i++ {
		g.DepositDirectional(300, 300, 0, 0.1, PherCarry) // heading = 0 (east)
	}
	dx, dy := g.GetDirection(300, 300)
	if dx <= 0 {
		t.Errorf("direction X should be positive (eastward), got %f", dx)
	}
	if math.Abs(dy) > 0.01 {
		t.Errorf("direction Y should be near 0, got %f", dy)
	}
}

func TestMultiGetDirectionEmpty(t *testing.T) {
	g := NewMultiPheromoneGrid(800, 800)
	dx, dy := g.GetDirection(400, 400)
	if dx != 0 || dy != 0 {
		t.Errorf("empty cell should return (0,0), got (%f,%f)", dx, dy)
	}
}

func TestMultiVariableDecay(t *testing.T) {
	g := NewMultiPheromoneGrid(800, 800)
	g.DepositDirectional(100, 100, 0, 0.5, PherCarry)
	g.DepositDirectional(100, 100, 0, 0.5, PherDanger)

	for i := 0; i < 100; i++ {
		g.UpdateMulti()
	}

	carry := g.GetChannel(100, 100, PherCarry)
	danger := g.GetChannel(100, 100, PherDanger)

	// Danger decays faster (0.980) than carry (0.995)
	if danger >= carry {
		t.Errorf("danger should decay faster than carry: danger=%f carry=%f", danger, carry)
	}
}

func TestMultiGradientChannel(t *testing.T) {
	g := NewMultiPheromoneGrid(800, 800)
	g.DepositDirectional(100, 200, 0, 0.2, PherHome)
	g.DepositDirectional(120, 200, 0, 0.6, PherHome)
	g.DepositDirectional(140, 200, 0, 0.9, PherHome)

	gx, _ := g.GradientChannel(120, 200, PherHome)
	if gx <= 0 {
		t.Errorf("gradient should point right, got gx=%f", gx)
	}
}

func TestMultiClear(t *testing.T) {
	g := NewMultiPheromoneGrid(800, 800)
	g.DepositDirectional(200, 200, 1.0, 0.5, PherCarry)
	g.DepositDirectional(300, 300, 2.0, 0.8, PherDanger)
	g.ClearMulti()

	for ch := 0; ch < PherChannels; ch++ {
		for _, v := range g.Channels[ch] {
			if v != 0 {
				t.Errorf("channel %d not cleared", ch)
				break
			}
		}
	}
	for _, d := range g.DirCount {
		if d != 0 {
			t.Error("direction counts not cleared")
			break
		}
	}
}

func TestMultiDepositClampsTo1(t *testing.T) {
	g := NewMultiPheromoneGrid(800, 800)
	g.DepositDirectional(100, 100, 0, 0.7, PherCarry)
	g.DepositDirectional(100, 100, 0, 0.7, PherCarry)
	val := g.GetChannel(100, 100, PherCarry)
	if val > 1.0 {
		t.Errorf("value should be clamped to 1.0, got %f", val)
	}
}
