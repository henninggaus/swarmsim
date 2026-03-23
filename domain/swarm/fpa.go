package swarm

import "math"

// Flower Pollination Algorithm (FPA): Nature-inspired optimization by Yang (2012).
// Mimics the pollination process of flowering plants:
//
//   - Global pollination (biotic, cross-pollination): pollinators carry pollen
//     over long distances following Lévy flight paths. This enables broad
//     exploration of the search space.
//   - Local pollination (abiotic, self-pollination): pollen is transferred
//     between nearby flowers via wind or diffusion. This refines solutions.
//   - Switch probability p: each flower randomly chooses between global and
//     local pollination each tick, balancing exploration vs exploitation.
//
// Reference: Yang, X.-S. (2012) "Flower Pollination Algorithm for Global
//            Optimization", Unconventional Computation and Natural Computation.

const (
	fpaMaxTicks   = 800   // full pollination cycle
	fpaSwitchProb = 0.8   // probability of global pollination (exploration)
	fpaSteerRate  = 0.20  // max steering per tick (radians)
	fpaLevyBeta   = 1.5   // Lévy exponent (1 < beta <= 2)
	fpaStepScale  = 0.5   // scale factor for Lévy steps
	fpaLocalScale = 0.3   // scale factor for local pollination
)

// FPAState holds Flower Pollination Algorithm state for the swarm.
type FPAState struct {
	Fitness    []float64 // current fitness per flower
	BestFit    []float64 // personal best fitness per flower
	BestX      []float64 // personal best X per flower
	BestY      []float64 // personal best Y per flower
	GlobalBestX float64  // global best X
	GlobalBestY float64  // global best Y
	GlobalBestF float64  // global best fitness
	GlobalIdx   int      // index of global best flower
	PollTick    int      // ticks into current cycle
	IsGlobal    []bool   // whether each flower did global poll this tick
}

// InitFPA allocates Flower Pollination Algorithm state.
func InitFPA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.FPA = &FPAState{
		Fitness:     make([]float64, n),
		BestFit:     make([]float64, n),
		BestX:       make([]float64, n),
		BestY:       make([]float64, n),
		IsGlobal:    make([]bool, n),
		GlobalBestF: -1e18,
	}
	// Initialize personal bests to current positions
	for i := range ss.Bots {
		ss.FPA.BestX[i] = ss.Bots[i].X
		ss.FPA.BestY[i] = ss.Bots[i].Y
		ss.FPA.BestFit[i] = -1e18
	}
	ss.FPAOn = true
}

// ClearFPA frees FPA state.
func ClearFPA(ss *SwarmState) {
	ss.FPA = nil
	ss.FPAOn = false
}

// TickFPA updates the Flower Pollination Algorithm for one tick.
func TickFPA(ss *SwarmState) {
	if ss.FPA == nil {
		return
	}
	st := ss.FPA
	n := len(ss.Bots)

	// Grow slices if bots were added
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.BestFit = append(st.BestFit, -1e18)
		st.BestX = append(st.BestX, ss.Bots[len(st.BestX)].X)
		st.BestY = append(st.BestY, ss.Bots[len(st.BestY)].Y)
		st.IsGlobal = append(st.IsGlobal, false)
	}

	st.PollTick++
	if st.PollTick > fpaMaxTicks {
		st.PollTick = 1
	}

	// Evaluate fitness and update personal/global bests
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
		if st.Fitness[i] > st.BestFit[i] {
			st.BestFit[i] = st.Fitness[i]
			st.BestX[i] = ss.Bots[i].X
			st.BestY[i] = ss.Bots[i].Y
		}
		if st.Fitness[i] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[i]
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
			st.GlobalIdx = i
		}
	}

	// Determine global/local pollination for each flower
	for i := range ss.Bots {
		st.IsGlobal[i] = ss.Rng.Float64() < fpaSwitchProb
	}

	// Update sensor cache for SwarmScript
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		// fpa_fitness: normalized to 0-100
		f := st.Fitness[i]
		if f < 0 {
			f = 0
		}
		bot.FPAFitness = int(f)
		if bot.FPAFitness > 100 {
			bot.FPAFitness = 100
		}

		// fpa_type: 0=global (Lévy), 1=local
		if st.IsGlobal[i] {
			bot.FPAType = 0
		} else {
			bot.FPAType = 1
		}

		// fpa_best_dist: distance to global best
		dx := bot.X - st.GlobalBestX
		dy := bot.Y - st.GlobalBestY
		bot.FPABestDist = int(math.Sqrt(dx*dx + dy*dy))
	}
}

// ApplyFPA applies per-bot steering for the Flower Pollination Algorithm.
func ApplyFPA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.FPA
	if st == nil {
		return
	}

	var dx, dy float64

	if st.IsGlobal[idx] {
		// Global pollination: Lévy flight toward global best
		levyStep := levyFlight(ss)
		dx = levyStep * (st.GlobalBestX - bot.X) * fpaStepScale
		dy = levyStep * (st.GlobalBestY - bot.Y) * fpaStepScale
	} else {
		// Local pollination: move between two random flowers
		n := len(ss.Bots)
		j := ss.Rng.Intn(n)
		k := ss.Rng.Intn(n)
		for j == idx && n > 1 {
			j = ss.Rng.Intn(n)
		}
		for k == idx && n > 1 {
			k = ss.Rng.Intn(n)
		}
		epsilon := ss.Rng.Float64()
		dx = epsilon * (ss.Bots[j].X - ss.Bots[k].X) * fpaLocalScale
		dy = epsilon * (ss.Bots[j].Y - ss.Bots[k].Y) * fpaLocalScale
	}

	if dx != 0 || dy != 0 {
		desired := math.Atan2(dy, dx)
		steerToward(bot, desired, fpaSteerRate)
		speed := math.Sqrt(dx*dx + dy*dy)
		if speed > SwarmBotSpeed*2 {
			speed = SwarmBotSpeed * 2
		}
		if speed < SwarmBotSpeed*0.3 {
			speed = SwarmBotSpeed * 0.3
		}
		bot.Speed = speed
	} else {
		// Global best or zero movement — do a small random walk
		bot.Angle += (ss.Rng.Float64() - 0.5) * 0.4
		bot.Speed = SwarmBotSpeed * 0.5
	}

	// LED visualization:
	// Global pollination = warm colors (yellow/orange based on fitness)
	// Local pollination = cool colors (blue/cyan)
	// Global best = gold
	if idx == st.GlobalIdx {
		bot.LEDColor = [3]uint8{255, 215, 0} // gold
	} else if st.IsGlobal[idx] {
		fit01 := st.Fitness[idx] / 100
		if fit01 < 0 {
			fit01 = 0
		}
		if fit01 > 1 {
			fit01 = 1
		}
		bot.LEDColor = [3]uint8{255, uint8(180 * fit01), uint8(50 * (1 - fit01))}
	} else {
		fit01 := st.Fitness[idx] / 100
		if fit01 < 0 {
			fit01 = 0
		}
		if fit01 > 1 {
			fit01 = 1
		}
		bot.LEDColor = [3]uint8{uint8(50 * (1 - fit01)), uint8(200 * fit01), 255}
	}
}

// levyFlight generates a step size from a Lévy distribution using the shared
// Mantegna algorithm, clamped to [-3, 3] for reasonable step sizes.
func levyFlight(ss *SwarmState) float64 {
	step := MantegnaLevy(ss.Rng, fpaLevyBeta)
	if step > 3.0 {
		step = 3.0
	}
	if step < -3.0 {
		step = -3.0
	}
	return step
}
