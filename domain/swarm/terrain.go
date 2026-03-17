package swarm

import (
	"math"
	"math/rand"
)

// BiomeType represents a terrain biome.
type BiomeType int

const (
	BiomeGrass BiomeType = iota // normal speed
	BiomeSand                   // 0.6x speed
	BiomeIce                    // 1.3x speed but random slip
	BiomeWater                  // impassable
	BiomeMud                    // 0.4x speed
	BiomeCount
)

// BiomeSpeedMod returns the speed multiplier for a biome.
func BiomeSpeedMod(b BiomeType) float64 {
	switch b {
	case BiomeGrass:
		return 1.0
	case BiomeSand:
		return 0.6
	case BiomeIce:
		return 1.3
	case BiomeMud:
		return 0.4
	case BiomeWater:
		return 0.0 // impassable
	default:
		return 1.0
	}
}

// TerrainCell stores height and biome for one grid cell.
type TerrainCell struct {
	Height float64   // 0.0 (low) to 1.0 (high)
	Biome  BiomeType
}

// TerrainGrid represents the terrain heightmap and biomes.
type TerrainGrid struct {
	Cols     int
	Rows     int
	CellSize float64
	Cells    []TerrainCell
}

// NewTerrainGrid creates a terrain grid for the given arena dimensions.
func NewTerrainGrid(arenaW, arenaH int, cellSize float64) *TerrainGrid {
	if cellSize < 1 {
		cellSize = 20
	}
	cols := int(math.Ceil(float64(arenaW) / cellSize))
	rows := int(math.Ceil(float64(arenaH) / cellSize))
	return &TerrainGrid{
		Cols:     cols,
		Rows:     rows,
		CellSize: cellSize,
		Cells:    make([]TerrainCell, cols*rows),
	}
}

// cellIndex returns the flat index for grid coordinates.
func (tg *TerrainGrid) cellIndex(col, row int) int {
	if col < 0 {
		col = 0
	}
	if row < 0 {
		row = 0
	}
	if col >= tg.Cols {
		col = tg.Cols - 1
	}
	if row >= tg.Rows {
		row = tg.Rows - 1
	}
	return row*tg.Cols + col
}

// GetCell returns the terrain cell at world coordinates.
func (tg *TerrainGrid) GetCell(x, y float64) *TerrainCell {
	col := int(x / tg.CellSize)
	row := int(y / tg.CellSize)
	idx := tg.cellIndex(col, row)
	return &tg.Cells[idx]
}

// GetHeight returns the height at world coordinates.
func (tg *TerrainGrid) GetHeight(x, y float64) float64 {
	return tg.GetCell(x, y).Height
}

// GetBiome returns the biome at world coordinates.
func (tg *TerrainGrid) GetBiome(x, y float64) BiomeType {
	return tg.GetCell(x, y).Biome
}

// SpeedAt returns the speed multiplier at world coordinates (biome + slope).
func (tg *TerrainGrid) SpeedAt(x, y float64) float64 {
	cell := tg.GetCell(x, y)
	biomeMod := BiomeSpeedMod(cell.Biome)
	// Slope penalty: steeper = slower (use height gradient)
	slopePenalty := 1.0
	col := int(x / tg.CellSize)
	row := int(y / tg.CellSize)
	if col > 0 && col < tg.Cols-1 {
		left := tg.Cells[tg.cellIndex(col-1, row)].Height
		right := tg.Cells[tg.cellIndex(col+1, row)].Height
		slope := math.Abs(right - left)
		slopePenalty = 1.0 - slope*0.5 // max 50% penalty for steep slopes
		if slopePenalty < 0.3 {
			slopePenalty = 0.3
		}
	}
	return biomeMod * slopePenalty
}

// IsPassable returns whether a bot can move through this position.
func (tg *TerrainGrid) IsPassable(x, y float64) bool {
	return tg.GetBiome(x, y) != BiomeWater
}

// GeneratePerlinTerrain fills the grid with Perlin-like noise heights and biomes.
func GeneratePerlinTerrain(rng *rand.Rand, tg *TerrainGrid) {
	// Simple value noise (not true Perlin, but good enough for gameplay)
	scale := 0.1
	for row := 0; row < tg.Rows; row++ {
		for col := 0; col < tg.Cols; col++ {
			// Multi-octave noise approximation
			h := 0.0
			h += math.Sin(float64(col)*scale*1.7+rng.Float64()*0.1) * 0.5
			h += math.Cos(float64(row)*scale*1.3+rng.Float64()*0.1) * 0.5
			h += rng.Float64() * 0.2
			h = (h + 1.0) / 2.0 // normalize to 0-1
			if h < 0 {
				h = 0
			}
			if h > 1 {
				h = 1
			}

			idx := tg.cellIndex(col, row)
			tg.Cells[idx].Height = h

			// Assign biome based on height
			switch {
			case h < 0.15:
				tg.Cells[idx].Biome = BiomeWater
			case h < 0.3:
				tg.Cells[idx].Biome = BiomeMud
			case h < 0.6:
				tg.Cells[idx].Biome = BiomeGrass
			case h < 0.8:
				tg.Cells[idx].Biome = BiomeSand
			default:
				tg.Cells[idx].Biome = BiomeIce
			}
		}
	}
}

// HeightGradient returns the direction of steepest ascent at world coordinates.
func (tg *TerrainGrid) HeightGradient(x, y float64) (float64, float64) {
	col := int(x / tg.CellSize)
	row := int(y / tg.CellSize)

	hCenter := tg.Cells[tg.cellIndex(col, row)].Height
	hRight := tg.Cells[tg.cellIndex(col+1, row)].Height
	hUp := tg.Cells[tg.cellIndex(col, row+1)].Height

	return hRight - hCenter, hUp - hCenter
}
