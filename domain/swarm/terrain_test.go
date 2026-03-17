package swarm

import (
	"math/rand"
	"testing"
)

func TestNewTerrainGrid(t *testing.T) {
	tg := NewTerrainGrid(800, 800, 20)
	if tg.Cols != 40 || tg.Rows != 40 {
		t.Errorf("expected 40x40, got %dx%d", tg.Cols, tg.Rows)
	}
	if len(tg.Cells) != 1600 {
		t.Errorf("expected 1600 cells, got %d", len(tg.Cells))
	}
}

func TestTerrainGetHeight(t *testing.T) {
	tg := NewTerrainGrid(800, 800, 20)
	tg.Cells[0].Height = 0.75
	h := tg.GetHeight(5, 5)
	if h != 0.75 {
		t.Errorf("expected 0.75, got %f", h)
	}
}

func TestTerrainGetBiome(t *testing.T) {
	tg := NewTerrainGrid(800, 800, 20)
	tg.Cells[0].Biome = BiomeIce
	b := tg.GetBiome(5, 5)
	if b != BiomeIce {
		t.Errorf("expected BiomeIce, got %d", b)
	}
}

func TestBiomeSpeedMod(t *testing.T) {
	if BiomeSpeedMod(BiomeGrass) != 1.0 {
		t.Error("grass should be 1.0")
	}
	if BiomeSpeedMod(BiomeSand) != 0.6 {
		t.Error("sand should be 0.6")
	}
	if BiomeSpeedMod(BiomeWater) != 0.0 {
		t.Error("water should be 0.0")
	}
	if BiomeSpeedMod(BiomeIce) != 1.3 {
		t.Error("ice should be 1.3")
	}
	if BiomeSpeedMod(BiomeMud) != 0.4 {
		t.Error("mud should be 0.4")
	}
}

func TestTerrainSpeedAt(t *testing.T) {
	tg := NewTerrainGrid(800, 800, 20)
	// Flat grass terrain
	for i := range tg.Cells {
		tg.Cells[i].Biome = BiomeGrass
		tg.Cells[i].Height = 0.5
	}
	s := tg.SpeedAt(400, 400)
	if s < 0.9 || s > 1.1 {
		t.Errorf("flat grass should be ~1.0, got %f", s)
	}
}

func TestTerrainIsPassable(t *testing.T) {
	tg := NewTerrainGrid(800, 800, 20)
	tg.Cells[0].Biome = BiomeGrass
	if !tg.IsPassable(5, 5) {
		t.Error("grass should be passable")
	}
	tg.Cells[0].Biome = BiomeWater
	if tg.IsPassable(5, 5) {
		t.Error("water should not be passable")
	}
}

func TestGeneratePerlinTerrain(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	tg := NewTerrainGrid(800, 800, 20)
	GeneratePerlinTerrain(rng, tg)

	hasWater := false
	hasGrass := false
	for _, c := range tg.Cells {
		if c.Height < 0 || c.Height > 1 {
			t.Errorf("height out of range: %f", c.Height)
		}
		if c.Biome == BiomeWater {
			hasWater = true
		}
		if c.Biome == BiomeGrass {
			hasGrass = true
		}
	}
	if !hasGrass {
		t.Error("should have some grass biome")
	}
	_ = hasWater // water may or may not appear depending on noise
}

func TestTerrainHeightGradient(t *testing.T) {
	tg := NewTerrainGrid(800, 800, 20)
	// Create a slope: increasing height to the right
	for row := 0; row < tg.Rows; row++ {
		for col := 0; col < tg.Cols; col++ {
			tg.Cells[tg.cellIndex(col, row)].Height = float64(col) / float64(tg.Cols)
		}
	}
	gx, _ := tg.HeightGradient(400, 400) // middle of grid
	if gx <= 0 {
		t.Errorf("gradient should point right (increasing), got gx=%f", gx)
	}
}

func TestTerrainEdgeCoordinates(t *testing.T) {
	tg := NewTerrainGrid(800, 800, 20)
	// Should not panic at edges
	_ = tg.GetHeight(0, 0)
	_ = tg.GetHeight(799, 799)
	_ = tg.GetHeight(-10, -10)
	_ = tg.SpeedAt(0, 0)
}
