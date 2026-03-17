package swarm

import (
	"math"
	"swarmsim/logger"
)

// ReactionDiffusionState manages a Turing-pattern reaction-diffusion system.
// Two chemical concentrations (activator A and inhibitor B) diffuse across
// a grid. Bots act as catalysts that locally boost activator production.
// The resulting patterns (spots, stripes, labyrinths) influence bot behavior.
type ReactionDiffusionState struct {
	GridW, GridH int       // grid dimensions
	CellSize     float64   // world units per cell
	A            []float64 // activator concentration (GridW*GridH)
	B            []float64 // inhibitor concentration (GridW*GridH)

	// Gray-Scott parameters
	FeedRate float64 // feed rate for A (default 0.055)
	KillRate float64 // kill rate for B (default 0.062)
	DiffuseA float64 // diffusion rate of A (default 1.0)
	DiffuseB float64 // diffusion rate of B (default 0.5)
	DeltaT   float64 // time step per tick (default 1.0)

	// Bot interaction
	CatalystStrength float64 // how much bots boost local A (default 0.01)
	ChemotaxisGain   float64 // how strongly bots follow A gradients (default 0.3)

	// Stats
	AvgA     float64 // average activator concentration
	AvgB     float64 // average inhibitor concentration
	Pattern  string  // detected pattern type
	Tick     int
}

// InitReactionDiffusion creates the reaction-diffusion grid.
func InitReactionDiffusion(ss *SwarmState, gridSize int) {
	if gridSize < 10 {
		gridSize = 10
	}
	if gridSize > 100 {
		gridSize = 100
	}

	cellSize := ss.ArenaW / float64(gridSize)
	n := gridSize * gridSize

	rd := &ReactionDiffusionState{
		GridW:    gridSize,
		GridH:    gridSize,
		CellSize: cellSize,
		A:        make([]float64, n),
		B:        make([]float64, n),

		FeedRate: 0.055,
		KillRate: 0.062,
		DiffuseA: 1.0,
		DiffuseB: 0.5,
		DeltaT:   1.0,

		CatalystStrength: 0.01,
		ChemotaxisGain:   0.3,
	}

	// Initialize: uniform A=1, B=0 with random seed patches
	for i := range rd.A {
		rd.A[i] = 1.0
		rd.B[i] = 0.0
	}

	// Seed some B patches to start pattern formation
	numSeeds := 5 + ss.Rng.Intn(5)
	for s := 0; s < numSeeds; s++ {
		cx := ss.Rng.Intn(gridSize)
		cy := ss.Rng.Intn(gridSize)
		r := 2 + ss.Rng.Intn(3)
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				gx := (cx + dx + gridSize) % gridSize
				gy := (cy + dy + gridSize) % gridSize
				if dx*dx+dy*dy <= r*r {
					idx := gy*gridSize + gx
					rd.A[idx] = 0.5 + ss.Rng.Float64()*0.5
					rd.B[idx] = 0.25 + ss.Rng.Float64()*0.25
				}
			}
		}
	}

	ss.ReactionDiffusion = rd
	logger.Info("RD", "Initialisiert: %dx%d Grid, Feed=%.3f, Kill=%.3f",
		gridSize, gridSize, rd.FeedRate, rd.KillRate)
}

// ClearReactionDiffusion disables the reaction-diffusion system.
func ClearReactionDiffusion(ss *SwarmState) {
	ss.ReactionDiffusion = nil
	ss.ReactionDiffusionOn = false
}

// TickReactionDiffusion runs one step of the Gray-Scott model.
func TickReactionDiffusion(ss *SwarmState) {
	rd := ss.ReactionDiffusion
	if rd == nil {
		return
	}

	w := rd.GridW
	h := rd.GridH
	n := w * h

	// Bot catalyst: bots boost local activator
	for i := range ss.Bots {
		gx := int(ss.Bots[i].X / rd.CellSize)
		gy := int(ss.Bots[i].Y / rd.CellSize)
		if gx >= 0 && gx < w && gy >= 0 && gy < h {
			idx := gy*w + gx
			rd.A[idx] += rd.CatalystStrength
			if rd.A[idx] > 1.0 {
				rd.A[idx] = 1.0
			}
		}
	}

	// Gray-Scott reaction-diffusion step
	newA := make([]float64, n)
	newB := make([]float64, n)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x

			// Laplacian (5-point stencil with wrapping)
			left := y*w + (x-1+w)%w
			right := y*w + (x+1)%w
			up := ((y-1+h)%h)*w + x
			down := ((y+1)%h)*w + x

			lapA := rd.A[left] + rd.A[right] + rd.A[up] + rd.A[down] - 4*rd.A[idx]
			lapB := rd.B[left] + rd.B[right] + rd.B[up] + rd.B[down] - 4*rd.B[idx]

			a := rd.A[idx]
			b := rd.B[idx]
			ab2 := a * b * b

			newA[idx] = a + rd.DeltaT*(rd.DiffuseA*lapA - ab2 + rd.FeedRate*(1-a))
			newB[idx] = b + rd.DeltaT*(rd.DiffuseB*lapB + ab2 - (rd.KillRate+rd.FeedRate)*b)

			// Clamp
			if newA[idx] < 0 {
				newA[idx] = 0
			}
			if newA[idx] > 1 {
				newA[idx] = 1
			}
			if newB[idx] < 0 {
				newB[idx] = 0
			}
			if newB[idx] > 1 {
				newB[idx] = 1
			}
		}
	}

	rd.A = newA
	rd.B = newB

	// Chemotaxis: bots follow activator gradient
	if rd.ChemotaxisGain > 0 {
		for i := range ss.Bots {
			gx := int(ss.Bots[i].X / rd.CellSize)
			gy := int(ss.Bots[i].Y / rd.CellSize)
			if gx < 1 || gx >= w-1 || gy < 1 || gy >= h-1 {
				continue
			}

			// Gradient of A
			gradX := rd.A[gy*w+gx+1] - rd.A[gy*w+gx-1]
			gradY := rd.A[(gy+1)*w+gx] - rd.A[(gy-1)*w+gx]

			if gradX != 0 || gradY != 0 {
				gradAngle := math.Atan2(gradY, gradX)
				diff := gradAngle - ss.Bots[i].Angle
				// Normalize to [-pi, pi]
				for diff > math.Pi {
					diff -= 2 * math.Pi
				}
				for diff < -math.Pi {
					diff += 2 * math.Pi
				}
				ss.Bots[i].Angle += diff * rd.ChemotaxisGain * 0.1
			}

			// Color bots by local concentration
			aVal := rd.A[gy*w+gx]
			bVal := rd.B[gy*w+gx]
			ss.Bots[i].LEDColor = [3]uint8{
				uint8(bVal * 200),
				uint8(aVal * 200),
				uint8((1 - aVal) * 100),
			}
		}
	}

	// Update stats
	sumA, sumB := 0.0, 0.0
	for i := range rd.A {
		sumA += rd.A[i]
		sumB += rd.B[i]
	}
	rd.AvgA = sumA / float64(n)
	rd.AvgB = sumB / float64(n)
	rd.Tick++

	// Detect pattern type based on B distribution
	rd.Pattern = detectPattern(rd)
}

// detectPattern classifies the current pattern.
func detectPattern(rd *ReactionDiffusionState) string {
	if rd.AvgB < 0.01 {
		return "Homogen"
	}

	// Count connected B-regions above threshold
	threshold := 0.1
	highCount := 0
	for _, b := range rd.B {
		if b > threshold {
			highCount++
		}
	}

	ratio := float64(highCount) / float64(len(rd.B))
	if ratio < 0.1 {
		return "Punkte"
	}
	if ratio > 0.6 {
		return "Invers-Punkte"
	}
	if ratio > 0.3 && ratio < 0.5 {
		return "Streifen"
	}
	return "Labyrinth"
}

// RDGetConcentration returns the A and B values at a world position.
func RDGetConcentration(rd *ReactionDiffusionState, wx, wy float64) (float64, float64) {
	if rd == nil {
		return 0, 0
	}
	gx := int(wx / rd.CellSize)
	gy := int(wy / rd.CellSize)
	if gx < 0 || gx >= rd.GridW || gy < 0 || gy >= rd.GridH {
		return 0, 0
	}
	idx := gy*rd.GridW + gx
	return rd.A[idx], rd.B[idx]
}

// RDSetParameters updates the Gray-Scott parameters for different pattern types.
func RDSetParameters(rd *ReactionDiffusionState, preset string) {
	if rd == nil {
		return
	}
	switch preset {
	case "spots":
		rd.FeedRate = 0.035
		rd.KillRate = 0.065
	case "stripes":
		rd.FeedRate = 0.055
		rd.KillRate = 0.062
	case "maze":
		rd.FeedRate = 0.029
		rd.KillRate = 0.057
	case "waves":
		rd.FeedRate = 0.014
		rd.KillRate = 0.054
	}
}
