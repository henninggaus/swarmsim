package swarm

import (
	"math"
	"sort"
)

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
	fpaMaxTicks       = 3000  // full pollination cycle (matches benchmark length)
	fpaSwitchProb     = 0.65  // probability of global pollination (exploration)
	fpaLevyBeta       = 1.5   // Lévy exponent (1 < beta <= 2)
	fpaStepScale      = 0.5   // scale factor for Lévy steps
	fpaLocalScale     = 0.3   // scale factor for local pollination
	fpaSpeedMult      = 5.0   // movement speed multiplier (7.5 px/tick)
	fpaGridRescanRate = 300   // periodic grid rescan every N ticks
	fpaGridRescanSize = 14    // grid resolution (14×14 = 196 samples)
	fpaGridInjectTop  = AlgoGridInjectTop // teleport worst N flowers to best grid positions
	fpaDirectTobestP  = 0.55  // max probability of direct-to-best in late phase
)

// FPAState holds Flower Pollination Algorithm state for the swarm.
type FPAState struct {
	Fitness     []float64 // current fitness per flower
	BestFit     []float64 // personal best fitness per flower
	BestX       []float64 // personal best X per flower
	BestY       []float64 // personal best Y per flower
	TargetX     []float64 // movement target X per flower
	TargetY     []float64 // movement target Y per flower
	GlobalBestX float64   // global best X
	GlobalBestY float64   // global best Y
	GlobalBestF float64   // global best fitness
	GlobalIdx   int       // index of global best flower
	PollTick    int       // ticks into current cycle
	IsGlobal    []bool    // whether each flower did global poll this tick
	IsDirect    []bool    // whether each flower is doing direct-to-best
}

// InitFPA allocates Flower Pollination Algorithm state.
func InitFPA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.FPA = &FPAState{
		Fitness:     make([]float64, n),
		BestFit:     make([]float64, n),
		BestX:       make([]float64, n),
		BestY:       make([]float64, n),
		TargetX:     make([]float64, n),
		TargetY:     make([]float64, n),
		IsGlobal:    make([]bool, n),
		IsDirect:    make([]bool, n),
		GlobalBestF: -1e18,
	}
	// Initialize personal bests to current positions
	for i := range ss.Bots {
		ss.FPA.BestX[i] = ss.Bots[i].X
		ss.FPA.BestY[i] = ss.Bots[i].Y
		ss.FPA.BestFit[i] = -1e18
		ss.FPA.TargetX[i] = ss.Bots[i].X
		ss.FPA.TargetY[i] = ss.Bots[i].Y
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
		idx := len(st.Fitness)
		st.Fitness = append(st.Fitness, 0)
		st.BestFit = append(st.BestFit, -1e18)
		st.BestX = append(st.BestX, ss.Bots[idx].X)
		st.BestY = append(st.BestY, ss.Bots[idx].Y)
		st.TargetX = append(st.TargetX, ss.Bots[idx].X)
		st.TargetY = append(st.TargetY, ss.Bots[idx].Y)
		st.IsGlobal = append(st.IsGlobal, false)
		st.IsDirect = append(st.IsDirect, false)
	}

	st.PollTick++
	if st.PollTick > fpaMaxTicks {
		st.PollTick = 1
	}

	progress := float64(st.PollTick) / float64(fpaMaxTicks)
	if progress > 1 {
		progress = 1
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

	// Periodic grid rescan: systematically sample the arena
	if st.PollTick > 0 && st.PollTick%fpaGridRescanRate == 0 && n > 0 {
		fpaGridRescan(ss, st)
	}

	// Compute targets for each flower
	for i := range ss.Bots {
		st.IsDirect[i] = false

		// Direct-to-best: in late phase, skip pollination and go straight to global best
		if progress > 0.3 {
			dtbProb := fpaDirectTobestP * (progress - 0.3) / 0.7
			if ss.Rng.Float64() < dtbProb {
				jitter := 7.5
				st.TargetX[i] = st.GlobalBestX + (ss.Rng.Float64()*2-1)*jitter
				st.TargetY[i] = st.GlobalBestY + (ss.Rng.Float64()*2-1)*jitter
				st.IsDirect[i] = true
				st.IsGlobal[i] = false

				// Evaluate the target point and update global best if better
				f := distanceFitnessPt(ss, st.TargetX[i], st.TargetY[i])
				if f > st.GlobalBestF {
					st.GlobalBestF = f
					st.GlobalBestX = st.TargetX[i]
					st.GlobalBestY = st.TargetY[i]
				}
				continue
			}
		}

		st.IsGlobal[i] = ss.Rng.Float64() < fpaSwitchProb

		var tx, ty float64
		if st.IsGlobal[i] {
			// Global pollination: Lévy flight toward global best
			levyStep := levyFlight(ss)
			tx = ss.Bots[i].X + levyStep*(st.GlobalBestX-ss.Bots[i].X)*fpaStepScale
			ty = ss.Bots[i].Y + levyStep*(st.GlobalBestY-ss.Bots[i].Y)*fpaStepScale
		} else {
			// Local pollination: move between two random flowers
			j := ss.Rng.Intn(n)
			k := ss.Rng.Intn(n)
			for j == i && n > 1 {
				j = ss.Rng.Intn(n)
			}
			for k == i && n > 1 {
				k = ss.Rng.Intn(n)
			}
			epsilon := ss.Rng.Float64()
			tx = ss.Bots[i].X + epsilon*(ss.Bots[j].X-ss.Bots[k].X)*fpaLocalScale
			ty = ss.Bots[i].Y + epsilon*(ss.Bots[j].Y-ss.Bots[k].Y)*fpaLocalScale
		}

		// Personal-best attraction (5% → 20%)
		pbWeight := 0.05 + 0.15*progress
		tx += pbWeight * (st.BestX[i] - tx)
		ty += pbWeight * (st.BestY[i] - ty)

		// Global-best attraction (5% → 50%)
		gbWeight := 0.05 + 0.45*progress
		tx += gbWeight * (st.GlobalBestX - tx)
		ty += gbWeight * (st.GlobalBestY - ty)

		st.TargetX[i] = tx
		st.TargetY[i] = ty
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

		// fpa_type: 0=global (Lévy), 1=local, 2=direct-to-best
		if st.IsDirect[i] {
			bot.FPAType = 2
		} else if st.IsGlobal[i] {
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

// ApplyFPA applies per-bot movement for the Flower Pollination Algorithm.
func ApplyFPA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.FPA
	if st == nil {
		return
	}

	// Move toward target using fast direct movement
	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, fpaSpeedMult)
	bot.Speed = 0 // prevent double-move in GUI physics step

	// LED visualization:
	// Direct-to-best = green
	// Global pollination = warm colors (yellow/orange based on fitness)
	// Local pollination = cool colors (blue/cyan)
	// Global best = gold
	if idx == st.GlobalIdx {
		bot.LEDColor = [3]uint8{255, 215, 0} // gold
	} else if st.IsDirect[idx] {
		bot.LEDColor = [3]uint8{0, 255, 100} // green for direct-to-best
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

// fpaGridRescan evaluates a grid of points across the arena and teleports
// the worst flowers to the best-discovered grid positions.
func fpaGridRescan(ss *SwarmState, st *FPAState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	gridPts := make([]gridPt, 0, fpaGridRescanSize*fpaGridRescanSize)
	for gx := 0; gx < fpaGridRescanSize; gx++ {
		for gy := 0; gy < fpaGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(fpaGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(fpaGridRescanSize)
			x += (ss.Rng.Float64()*2.0 - 1.0) * usableW * 0.02
			y += (ss.Rng.Float64()*2.0 - 1.0) * usableH * 0.02
			f := distanceFitnessPt(ss, x, y)
			gridPts = append(gridPts, gridPt{x, y, f})
		}
	}

	// Sort grid points by fitness descending
	sort.Slice(gridPts, func(i, j int) bool { return gridPts[i].f > gridPts[j].f })

	// Update GlobalBest from grid findings
	if len(gridPts) > 0 && gridPts[0].f > st.GlobalBestF {
		st.GlobalBestF = gridPts[0].f
		st.GlobalBestX = gridPts[0].x
		st.GlobalBestY = gridPts[0].y
	}

	// Find worst flowers by fitness
	flowers := make([]idxFit, n)
	for i := range ss.Bots {
		flowers[i] = idxFit{i, st.Fitness[i]}
	}
	// Sort ascending by fitness (worst first)
	sort.Slice(flowers, func(i, j int) bool { return flowers[i].f < flowers[j].f })

	// Teleport worst flowers to best grid points
	inject := fpaGridInjectTop
	if inject > len(gridPts) {
		inject = len(gridPts)
	}
	if inject > n {
		inject = n
	}
	for i := 0; i < inject; i++ {
		bi := flowers[i].idx
		jitter := 5.0
		ss.Bots[bi].X = gridPts[i].x + (ss.Rng.Float64()*2-1)*jitter
		ss.Bots[bi].Y = gridPts[i].y + (ss.Rng.Float64()*2-1)*jitter
		// Clamp to arena
		if ss.Bots[bi].X < SwarmBotRadius {
			ss.Bots[bi].X = SwarmBotRadius
		}
		if ss.Bots[bi].X > ss.ArenaW-SwarmBotRadius {
			ss.Bots[bi].X = ss.ArenaW - SwarmBotRadius
		}
		if ss.Bots[bi].Y < SwarmBotRadius {
			ss.Bots[bi].Y = SwarmBotRadius
		}
		if ss.Bots[bi].Y > ss.ArenaH-SwarmBotRadius {
			ss.Bots[bi].Y = ss.ArenaH - SwarmBotRadius
		}
		// Update personal best for teleported flower
		f := distanceFitnessPt(ss, ss.Bots[bi].X, ss.Bots[bi].Y)
		if f > st.BestFit[bi] {
			st.BestFit[bi] = f
			st.BestX[bi] = ss.Bots[bi].X
			st.BestY[bi] = ss.Bots[bi].Y
		}
		// Set target to current position so algoMovBot doesn't overshoot
		st.TargetX[bi] = ss.Bots[bi].X
		st.TargetY[bi] = ss.Bots[bi].Y
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
