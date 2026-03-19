package swarm

import "math"

// Morphogen Gradients: Reaction-Diffusion pattern formation in the swarm.
// Inspired by Turing patterns — two "chemical" signals (activator + inhibitor)
// diffuse between neighboring bots at different rates. The interaction creates
// emergent spatial patterns: spots, stripes, spirals.
// Each bot carries activator (A) and inhibitor (H) concentrations.

const (
	morphRadius    = 60.0  // diffusion neighborhood radius
	morphDiffA     = 0.08  // activator diffusion rate
	morphDiffH     = 0.16  // inhibitor diffusion rate (faster = spots/stripes)
	morphFeedA     = 0.04  // activator production rate
	morphDecayH    = 0.06  // inhibitor decay rate
	morphSaturation = 1.0  // max concentration
)

// MorphogenState holds per-bot chemical concentrations.
type MorphogenState struct {
	A []float64 // activator concentration [0, 1]
	H []float64 // inhibitor concentration [0, 1]
	// Scratch buffers for double-buffered update
	dA []float64
	dH []float64
}

// InitMorphogen allocates morphogen state with small random perturbations.
func InitMorphogen(ss *SwarmState) {
	n := len(ss.Bots)
	st := &MorphogenState{
		A:  make([]float64, n),
		H:  make([]float64, n),
		dA: make([]float64, n),
		dH: make([]float64, n),
	}
	for i := 0; i < n; i++ {
		st.A[i] = 0.5 + (ss.Rng.Float64()-0.5)*0.1
		st.H[i] = 0.5 + (ss.Rng.Float64()-0.5)*0.1
	}
	ss.Morphogen = st
	ss.MorphogenOn = true
}

// ClearMorphogen frees morphogen state.
func ClearMorphogen(ss *SwarmState) {
	ss.Morphogen = nil
	ss.MorphogenOn = false
}

// TickMorphogen runs one reaction-diffusion step across all bots.
// Updates MorphA (0-100) and MorphH (0-100) sensor cache.
func TickMorphogen(ss *SwarmState) {
	if ss.Morphogen == nil || ss.Hash == nil {
		return
	}
	st := ss.Morphogen
	n := len(ss.Bots)

	// Grow slices if bots added
	for len(st.A) < n {
		st.A = append(st.A, 0.5)
		st.H = append(st.H, 0.5)
		st.dA = append(st.dA, 0)
		st.dH = append(st.dH, 0)
	}

	// Compute deltas (reaction + diffusion)
	for i := 0; i < n; i++ {
		st.dA[i] = 0
		st.dH[i] = 0
	}

	for i := 0; i < n; i++ {
		bot := &ss.Bots[i]
		nearIDs := ss.Hash.Query(bot.X, bot.Y, morphRadius)

		var sumA, sumH float64
		count := 0
		for _, j := range nearIDs {
			if j == i || j < 0 || j >= n {
				continue
			}
			nb := &ss.Bots[j]
			dx := bot.X - nb.X
			dy := bot.Y - nb.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > morphRadius || dist < 0.001 {
				continue
			}
			// Distance-weighted diffusion
			w := 1.0 - dist/morphRadius
			sumA += (st.A[j] - st.A[i]) * w
			sumH += (st.H[j] - st.H[i]) * w
			count++
		}

		if count > 0 {
			// Diffusion
			st.dA[i] += morphDiffA * sumA / float64(count)
			st.dH[i] += morphDiffH * sumH / float64(count)
		}

		// Reaction: Gierer-Meinhardt model
		// dA/dt = A^2/H - A + feedA
		// dH/dt = A^2 - H*decayH
		a := st.A[i]
		h := st.H[i]
		if h < 0.01 {
			h = 0.01
		}
		st.dA[i] += (a*a/h - a + morphFeedA) * 0.1
		st.dH[i] += (a*a - h*morphDecayH) * 0.1
	}

	// Apply deltas
	for i := 0; i < n; i++ {
		st.A[i] += st.dA[i]
		st.H[i] += st.dH[i]
		// Clamp
		if st.A[i] < 0 {
			st.A[i] = 0
		}
		if st.A[i] > morphSaturation {
			st.A[i] = morphSaturation
		}
		if st.H[i] < 0 {
			st.H[i] = 0
		}
		if st.H[i] > morphSaturation {
			st.H[i] = morphSaturation
		}

		// Update sensor cache
		ss.Bots[i].MorphA = int(st.A[i] * 100)
		ss.Bots[i].MorphH = int(st.H[i] * 100)
	}
}

// ApplyMorphColor sets bot LED based on its morphogen concentrations.
// Creates visible Turing patterns across the swarm.
func ApplyMorphColor(bot *SwarmBot, ss *SwarmState, idx int) {
	a := float64(bot.MorphA) / 100.0
	h := float64(bot.MorphH) / 100.0
	// High activator: warm colors, high inhibitor: cool colors
	bot.LEDColor = [3]uint8{
		uint8(math.Min(255, a*255)),
		uint8(math.Min(255, (1-math.Abs(a-h))*150)),
		uint8(math.Min(255, h*255)),
	}
	bot.Speed = SwarmBotSpeed
}
