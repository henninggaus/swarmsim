package pheromone

import "math"

// NewPheromoneGrid creates a pheromone grid covering the arena.
func NewPheromoneGrid(arenaW, arenaH, cellSize, decay, diffusion float64) *PheromoneGrid {
	cols := int(math.Ceil(arenaW / cellSize))
	rows := int(math.Ceil(arenaH / cellSize))
	n := cols * rows
	g := &PheromoneGrid{
		Cols:      cols,
		Rows:      rows,
		CellSize:  cellSize,
		Temp:      make([]float64, n),
		Decay:     decay,
		Diffusion: diffusion,
	}
	for t := 0; t < PherCount; t++ {
		g.Data[t] = make([]float64, n)
	}
	return g
}

func (g *PheromoneGrid) clampCell(c, r int) (int, int) {
	if c < 0 {
		c = 0
	}
	if r < 0 {
		r = 0
	}
	if c >= g.Cols {
		c = g.Cols - 1
	}
	if r >= g.Rows {
		r = g.Rows - 1
	}
	return c, r
}

func (g *PheromoneGrid) worldToCell(x, y float64) (int, int) {
	c := int(x / g.CellSize)
	r := int(y / g.CellSize)
	return g.clampCell(c, r)
}

// Deposit adds pheromone at world position (x,y).
func (g *PheromoneGrid) Deposit(x, y float64, pType PheromoneType, amount float64) {
	c, r := g.worldToCell(x, y)
	idx := r*g.Cols + c
	g.Data[pType][idx] += amount
	if g.Data[pType][idx] > 1.0 {
		g.Data[pType][idx] = 1.0
	}
}

// Get returns pheromone intensity at world position (x,y).
func (g *PheromoneGrid) Get(x, y float64, pType PheromoneType) float64 {
	c, r := g.worldToCell(x, y)
	return g.Data[pType][r*g.Cols+c]
}

// GetCell returns pheromone intensity at cell (c,r).
func (g *PheromoneGrid) GetCell(c, r int, pType PheromoneType) float64 {
	c, r = g.clampCell(c, r)
	return g.Data[pType][r*g.Cols+c]
}

// Gradient returns the direction of increasing pheromone at world position (x,y).
func (g *PheromoneGrid) Gradient(x, y float64, pType PheromoneType) (float64, float64) {
	c, r := g.worldToCell(x, y)
	center := g.Data[pType][r*g.Cols+c]
	if center < 0.001 {
		var maxVal float64
		var bestDX, bestDY float64
		for dr := -1; dr <= 1; dr++ {
			for dc := -1; dc <= 1; dc++ {
				if dr == 0 && dc == 0 {
					continue
				}
				nc, nr := g.clampCell(c+dc, r+dr)
				if nc == c+dc && nr == r+dr {
					v := g.Data[pType][nr*g.Cols+nc]
					if v > maxVal {
						maxVal = v
						bestDX = float64(dc)
						bestDY = float64(dr)
					}
				}
			}
		}
		if maxVal > 0.001 {
			l := math.Sqrt(bestDX*bestDX + bestDY*bestDY)
			if l > 0 {
				return bestDX / l * maxVal, bestDY / l * maxVal
			}
		}
		return 0, 0
	}

	var left, right, up, down float64
	if c > 0 {
		left = g.Data[pType][r*g.Cols+c-1]
	}
	if c < g.Cols-1 {
		right = g.Data[pType][r*g.Cols+c+1]
	}
	if r > 0 {
		up = g.Data[pType][(r-1)*g.Cols+c]
	}
	if r < g.Rows-1 {
		down = g.Data[pType][(r+1)*g.Cols+c]
	}
	return (right - left) / 2, (down - up) / 2
}

// Update applies evaporation and diffusion to all pheromone types.
func (g *PheromoneGrid) Update() {
	n := g.Cols * g.Rows
	for t := 0; t < PherCount; t++ {
		copy(g.Temp, g.Data[t])
		for r := 0; r < g.Rows; r++ {
			for c := 0; c < g.Cols; c++ {
				idx := r*g.Cols + c
				val := g.Temp[idx] * g.Decay
				if val < 0.001 {
					g.Data[t][idx] = 0
					continue
				}
				spread := val * g.Diffusion
				neighbors := 0
				if c > 0 {
					neighbors++
				}
				if c < g.Cols-1 {
					neighbors++
				}
				if r > 0 {
					neighbors++
				}
				if r < g.Rows-1 {
					neighbors++
				}
				g.Data[t][idx] = val - spread*float64(neighbors)
				if c > 0 {
					g.Data[t][idx-1] += spread
				}
				if c < g.Cols-1 {
					g.Data[t][idx+1] += spread
				}
				if r > 0 {
					g.Data[t][idx-g.Cols] += spread
				}
				if r < g.Rows-1 {
					g.Data[t][idx+g.Cols] += spread
				}
			}
		}
		for i := 0; i < n; i++ {
			if g.Data[t][i] > 1.0 {
				g.Data[t][i] = 1.0
			} else if g.Data[t][i] < 0 {
				g.Data[t][i] = 0
			}
		}
	}
}

// Clear resets all pheromone values.
func (g *PheromoneGrid) Clear() {
	for t := 0; t < PherCount; t++ {
		for i := range g.Data[t] {
			g.Data[t][i] = 0
		}
	}
}
