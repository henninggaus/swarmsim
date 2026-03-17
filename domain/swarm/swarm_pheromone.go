package swarm

import "math"

// SwarmPheromoneGrid is a single-channel pheromone grid for the swarm arena.
// Carrying bots deposit pheromone; non-carrying bots can follow the gradient.
type SwarmPheromoneGrid struct {
	Cols, Rows int
	CellSize   float64
	Data       []float64
	Temp       []float64
}

// NewSwarmPheromoneGrid creates a pheromone grid with 20px cells.
func NewSwarmPheromoneGrid(arenaW, arenaH float64) *SwarmPheromoneGrid {
	cellSize := 20.0
	cols := int(math.Ceil(arenaW / cellSize))
	rows := int(math.Ceil(arenaH / cellSize))
	n := cols * rows
	return &SwarmPheromoneGrid{
		Cols:     cols,
		Rows:     rows,
		CellSize: cellSize,
		Data:     make([]float64, n),
		Temp:     make([]float64, n),
	}
}

func (g *SwarmPheromoneGrid) clampCell(c, r int) (int, int) {
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

func (g *SwarmPheromoneGrid) worldToCell(x, y float64) (int, int) {
	c := int(x / g.CellSize)
	r := int(y / g.CellSize)
	return g.clampCell(c, r)
}

// Deposit adds pheromone at world position (x,y), capped at 1.0.
func (g *SwarmPheromoneGrid) Deposit(x, y, amount float64) {
	c, r := g.worldToCell(x, y)
	idx := r*g.Cols + c
	g.Data[idx] += amount
	if g.Data[idx] > 1.0 {
		g.Data[idx] = 1.0
	}
}

// Get returns pheromone intensity at world position (x,y).
func (g *SwarmPheromoneGrid) Get(x, y float64) float64 {
	c, r := g.worldToCell(x, y)
	return g.Data[r*g.Cols+c]
}

// Gradient returns the direction of increasing pheromone at world position.
// Uses central differences for cells with pheromone, and neighbor search otherwise.
func (g *SwarmPheromoneGrid) Gradient(x, y float64) (float64, float64) {
	c, r := g.worldToCell(x, y)
	center := g.Data[r*g.Cols+c]

	if center < 0.001 {
		// No pheromone here — look at neighbors for any signal
		var maxVal float64
		var bestDX, bestDY float64
		for dr := -1; dr <= 1; dr++ {
			for dc := -1; dc <= 1; dc++ {
				if dr == 0 && dc == 0 {
					continue
				}
				nc, nr := g.clampCell(c+dc, r+dr)
				if nc == c+dc && nr == r+dr {
					v := g.Data[nr*g.Cols+nc]
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

	// Central differences
	var left, right, up, down float64
	if c > 0 {
		left = g.Data[r*g.Cols+c-1]
	}
	if c < g.Cols-1 {
		right = g.Data[r*g.Cols+c+1]
	}
	if r > 0 {
		up = g.Data[(r-1)*g.Cols+c]
	}
	if r < g.Rows-1 {
		down = g.Data[(r+1)*g.Cols+c]
	}
	return (right - left) / 2, (down - up) / 2
}

// Update applies decay (0.995) and diffusion (0.01) to the pheromone grid.
func (g *SwarmPheromoneGrid) Update() {
	const decay = 0.995
	const diffusion = 0.01
	n := g.Cols * g.Rows

	copy(g.Temp, g.Data)
	for r := 0; r < g.Rows; r++ {
		for c := 0; c < g.Cols; c++ {
			idx := r*g.Cols + c
			val := g.Temp[idx] * decay
			if val < 0.001 {
				g.Data[idx] = 0
				continue
			}
			spread := val * diffusion
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
			g.Data[idx] = val - spread*float64(neighbors)
			if c > 0 {
				g.Data[idx-1] += spread
			}
			if c < g.Cols-1 {
				g.Data[idx+1] += spread
			}
			if r > 0 {
				g.Data[idx-g.Cols] += spread
			}
			if r < g.Rows-1 {
				g.Data[idx+g.Cols] += spread
			}
		}
	}
	// Clamp
	for i := 0; i < n; i++ {
		if g.Data[i] > 1.0 {
			g.Data[i] = 1.0
		} else if g.Data[i] < 0 {
			g.Data[i] = 0
		}
	}
}

// Clear resets all pheromone values.
func (g *SwarmPheromoneGrid) Clear() {
	for i := range g.Data {
		g.Data[i] = 0
	}
}

// --- Multi-Channel Pheromone Extension ---

// PherChannel identifies a pheromone type.
type PherChannel int

const (
	PherCarry     PherChannel = 0 // deposited by carrying bots (default)
	PherHome      PherChannel = 1 // deposited by bots returning to pickup
	PherDanger    PherChannel = 2 // deposited near obstacles/collisions
	PherChannels              = 3 // total number of channels
)

// MultiPheromoneGrid extends the basic grid with multiple channels and
// per-cell directional data (heading of depositing bot).
type MultiPheromoneGrid struct {
	Cols, Rows int
	CellSize   float64
	Channels   [PherChannels][]float64
	DirX       []float64 // average heading X component per cell
	DirY       []float64 // average heading Y component per cell
	DirCount   []int     // number of deposits contributing to direction
	DecayRates [PherChannels]float64
}

// NewMultiPheromoneGrid creates a multi-channel pheromone grid.
func NewMultiPheromoneGrid(arenaW, arenaH float64) *MultiPheromoneGrid {
	cellSize := 20.0
	cols := int(math.Ceil(arenaW / cellSize))
	rows := int(math.Ceil(arenaH / cellSize))
	n := cols * rows
	g := &MultiPheromoneGrid{
		Cols:     cols,
		Rows:     rows,
		CellSize: cellSize,
		DirX:     make([]float64, n),
		DirY:     make([]float64, n),
		DirCount: make([]int, n),
		DecayRates: [PherChannels]float64{0.995, 0.990, 0.980}, // carry=slow, home=medium, danger=fast
	}
	for ch := 0; ch < PherChannels; ch++ {
		g.Channels[ch] = make([]float64, n)
	}
	return g
}

func (g *MultiPheromoneGrid) worldToCell(x, y float64) (int, int) {
	c := int(x / g.CellSize)
	r := int(y / g.CellSize)
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

// DepositDirectional deposits pheromone with bot heading direction.
func (g *MultiPheromoneGrid) DepositDirectional(x, y, heading, amount float64, ch PherChannel) {
	c, r := g.worldToCell(x, y)
	idx := r*g.Cols + c
	g.Channels[ch][idx] += amount
	if g.Channels[ch][idx] > 1.0 {
		g.Channels[ch][idx] = 1.0
	}
	// Rolling average of heading direction
	g.DirX[idx] += math.Cos(heading)
	g.DirY[idx] += math.Sin(heading)
	g.DirCount[idx]++
}

// GetChannel returns pheromone intensity for a specific channel.
func (g *MultiPheromoneGrid) GetChannel(x, y float64, ch PherChannel) float64 {
	c, r := g.worldToCell(x, y)
	return g.Channels[ch][r*g.Cols+c]
}

// GetDirection returns the average heading direction at a cell.
func (g *MultiPheromoneGrid) GetDirection(x, y float64) (float64, float64) {
	c, r := g.worldToCell(x, y)
	idx := r*g.Cols + c
	if g.DirCount[idx] == 0 {
		return 0, 0
	}
	n := float64(g.DirCount[idx])
	return g.DirX[idx] / n, g.DirY[idx] / n
}

// GradientChannel computes gradient for a specific channel.
func (g *MultiPheromoneGrid) GradientChannel(x, y float64, ch PherChannel) (float64, float64) {
	c, r := g.worldToCell(x, y)
	data := g.Channels[ch]

	var left, right, up, down float64
	if c > 0 {
		left = data[r*g.Cols+c-1]
	}
	if c < g.Cols-1 {
		right = data[r*g.Cols+c+1]
	}
	if r > 0 {
		up = data[(r-1)*g.Cols+c]
	}
	if r < g.Rows-1 {
		down = data[(r+1)*g.Cols+c]
	}
	return (right - left) / 2, (down - up) / 2
}

// UpdateMulti applies per-channel decay rates and diffusion.
func (g *MultiPheromoneGrid) UpdateMulti() {
	const diffusion = 0.01
	n := g.Cols * g.Rows
	temp := make([]float64, n)

	for ch := 0; ch < PherChannels; ch++ {
		decay := g.DecayRates[ch]
		data := g.Channels[ch]
		copy(temp, data)

		for r := 0; r < g.Rows; r++ {
			for c := 0; c < g.Cols; c++ {
				idx := r*g.Cols + c
				val := temp[idx] * decay
				if val < 0.001 {
					data[idx] = 0
					continue
				}
				spread := val * diffusion
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
				data[idx] = val - spread*float64(neighbors)
				if c > 0 {
					data[idx-1] += spread
				}
				if c < g.Cols-1 {
					data[idx+1] += spread
				}
				if r > 0 {
					data[idx-g.Cols] += spread
				}
				if r < g.Rows-1 {
					data[idx+g.Cols] += spread
				}
			}
		}
		// Clamp
		for i := 0; i < n; i++ {
			if data[i] > 1.0 {
				data[i] = 1.0
			} else if data[i] < 0 {
				data[i] = 0
			}
		}
	}

	// Decay directional data
	for i := 0; i < n; i++ {
		g.DirX[i] *= 0.99
		g.DirY[i] *= 0.99
		if g.DirCount[i] > 100 {
			g.DirCount[i] = 100 // cap count to prevent overflow
		}
	}
}

// ClearMulti resets all channels and directional data.
func (g *MultiPheromoneGrid) ClearMulti() {
	for ch := 0; ch < PherChannels; ch++ {
		for i := range g.Channels[ch] {
			g.Channels[ch][i] = 0
		}
	}
	for i := range g.DirX {
		g.DirX[i] = 0
		g.DirY[i] = 0
		g.DirCount[i] = 0
	}
}
