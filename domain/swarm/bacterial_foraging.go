package swarm

import "math"

// Bacterial Foraging Optimization (BFO): Inspired by the foraging behavior
// of E. coli bacteria. Bacteria use chemotaxis (swim & tumble) to navigate
// nutrient gradients, reproduce (cell division of fittest), and are subject
// to elimination-dispersal events that maintain population diversity.
//
// Phases:
// 1. Chemotaxis — swim in current direction if improving, tumble (gradient-biased) if not
// 2. Swarming — cell-to-cell signaling attracts bacteria to nutrient-rich areas
// 3. Reproduction — fittest half clones replace least fit half
// 4. Elimination-Dispersal — random bots teleport to new positions
//
// Reference: Passino, K.M. (2002)
//            "Biomimicry of bacterial foraging for distributed optimization"

const (
	bfoChemoSteps      = 6     // consecutive swim steps before re-evaluation
	bfoTumbleRate      = 0.15  // probability of tumbling each step
	bfoSwarmRadius     = 80.0  // swarming signal radius
	bfoReproInterval   = 150   // ticks between reproduction events
	bfoElimProbEarly   = 0.003 // elimination-dispersal probability (early phase)
	bfoElimProbLate    = 0.001 // elimination-dispersal probability (late phase, less disruption)
	bfoGradientDirs    = 8     // number of directions to probe during tumble
	bfoMaxTicks        = 3000  // total ticks for adaptive parameter scheduling
	bfoProbeDistStart  = 40.0  // gradient probe distance at start (exploration)
	bfoProbeDistEnd    = 8.0   // gradient probe distance at end (exploitation)
	bfoGBestWStart     = 0.05  // global-best attraction weight at start
	bfoGBestWEnd       = 0.80  // global-best attraction weight at end (stronger convergence)
	bfoSpeedMult       = 5.0   // movement speed multiplier (5x = 7.5 px/tick)
	bfoGridRescanRate  = 120   // periodic grid rescan every N ticks (more frequent)
	bfoGridRescanSide  = 20    // grid resolution (20x20 = 400 samples)
	bfoGridInjectTop   = 15    // best grid positions to inject (more replacement)
	bfoDtbStartProg    = 0.10  // direct-to-best starts earlier
	bfoDtbMaxProb      = 0.85  // max probability of direct-to-best (stronger)
	bfoLocalWalkR      = 40.0  // best-bot local random walk radius
	bfoInitGridSide    = 20    // initial grid scan resolution (20x20 = 400 samples)
)

// BFOState holds Bacterial Foraging Optimization state.
type BFOState struct {
	Fitness    []float64 // current landscape fitness per bot
	SwimDir    []float64 // current swim direction (radians)
	SwimCount  []int     // steps remaining in current swim
	PrevFit    []float64 // fitness at previous position
	Health     []float64 // accumulated health for reproduction selection
	PBestF     []float64 // personal best fitness per bot
	PBestX     []float64 // personal best X per bot
	PBestY     []float64 // personal best Y per bot
	TargetX    []float64 // current movement target X per bot
	TargetY    []float64 // current movement target Y per bot
	CycleTimer int       // ticks since last reproduction
	Tick       int       // total ticks elapsed (for adaptive scheduling)
	BestF      float64   // global best fitness found
	BestX      float64   // global best X position
	BestY      float64   // global best Y position
	BestIdx    int       // index of current best bot
}

// InitBFO allocates Bacterial Foraging state for all bots.
func InitBFO(ss *SwarmState) {
	n := len(ss.Bots)
	st := &BFOState{
		Fitness:   make([]float64, n),
		SwimDir:   make([]float64, n),
		SwimCount: make([]int, n),
		PrevFit:   make([]float64, n),
		Health:    make([]float64, n),
		PBestF:    make([]float64, n),
		PBestX:    make([]float64, n),
		PBestY:    make([]float64, n),
		TargetX:   make([]float64, n),
		TargetY:   make([]float64, n),
		BestF:     -1e18,
		BestIdx:   -1,
	}
	// Initialize random swim directions and evaluate initial fitness
	for i := range ss.Bots {
		st.SwimDir[i] = ss.Rng.Float64() * 2 * math.Pi
		f := distanceFitness(&ss.Bots[i], ss)
		st.Fitness[i] = f
		st.PrevFit[i] = f
		st.PBestF[i] = f
		st.PBestX[i] = ss.Bots[i].X
		st.PBestY[i] = ss.Bots[i].Y
		st.TargetX[i] = ss.Bots[i].X
		st.TargetY[i] = ss.Bots[i].Y
		if f > st.BestF {
			st.BestF = f
			st.BestX = ss.Bots[i].X
			st.BestY = ss.Bots[i].Y
			st.BestIdx = i
		}
	}
	// Initial grid scan: systematically sample the landscape to find the global
	// optimum early. Critical for deceptive landscapes like Schwefel where random
	// initialization rarely places bots near the global optimum.
	aw := ss.ArenaW
	ah := ss.ArenaH
	for gx := 0; gx < bfoInitGridSide; gx++ {
		for gy := 0; gy < bfoInitGridSide; gy++ {
			px := (float64(gx) + 0.5) * aw / float64(bfoInitGridSide)
			py := (float64(gy) + 0.5) * ah / float64(bfoInitGridSide)
			f := distanceFitnessPt(ss, px, py)
			if f > st.BestF {
				st.BestF = f
				st.BestX = px
				st.BestY = py
			}
		}
	}

	ss.BFO = st
	ss.BFOOn = true
}

// ClearBFO frees Bacterial Foraging state.
func ClearBFO(ss *SwarmState) {
	ss.BFO = nil
	ss.BFOOn = false
}

// TickBFO updates the Bacterial Foraging Optimization for all bots.
func TickBFO(ss *SwarmState) {
	if ss.BFO == nil {
		return
	}
	st := ss.BFO

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		idx := len(st.Fitness)
		st.Fitness = append(st.Fitness, 0)
		st.SwimDir = append(st.SwimDir, ss.Rng.Float64()*2*math.Pi)
		st.SwimCount = append(st.SwimCount, 0)
		st.PrevFit = append(st.PrevFit, 0)
		st.Health = append(st.Health, 0)
		st.PBestF = append(st.PBestF, -1e18)
		if idx < len(ss.Bots) {
			st.PBestX = append(st.PBestX, ss.Bots[idx].X)
			st.PBestY = append(st.PBestY, ss.Bots[idx].Y)
			st.TargetX = append(st.TargetX, ss.Bots[idx].X)
			st.TargetY = append(st.TargetY, ss.Bots[idx].Y)
		} else {
			st.PBestX = append(st.PBestX, 0)
			st.PBestY = append(st.PBestY, 0)
			st.TargetX = append(st.TargetX, 0)
			st.TargetY = append(st.TargetY, 0)
		}
	}

	st.CycleTimer++
	st.Tick++

	// Adaptive progress: 0.0 (exploration) → 1.0 (exploitation)
	progress := float64(st.Tick) / float64(bfoMaxTicks)
	if progress > 1.0 {
		progress = 1.0
	}
	// Adaptive gradient probe distance: large early, small late
	probeDist := bfoProbeDistStart + (bfoProbeDistEnd-bfoProbeDistStart)*progress

	// Phase 1: Chemotaxis with gradient-guided tumble — compute targets
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		fit := distanceFitness(bot, ss)
		st.Fitness[i] = fit

		// Update personal best
		if fit > st.PBestF[i] {
			st.PBestF[i] = fit
			st.PBestX[i] = bot.X
			st.PBestY[i] = bot.Y
		}

		// Update global best
		if fit > st.BestF {
			st.BestF = fit
			st.BestX = bot.X
			st.BestY = bot.Y
			st.BestIdx = i
		}

		// Accumulate health for reproduction (faster EMA for better responsiveness)
		st.Health[i] = st.Health[i]*0.95 + fit*0.05

		// Direct-to-best: in late phase, some bots skip chemotaxis and go
		// directly to global best with small jitter
		if progress > bfoDtbStartProg && st.BestF > 0 {
			dtbProb := bfoDtbMaxProb * (progress - bfoDtbStartProg) / (1.0 - bfoDtbStartProg)
			if ss.Rng.Float64() < dtbProb {
				jitter := probeDist * 0.5
				st.TargetX[i] = st.BestX + (ss.Rng.Float64()-0.5)*2*jitter
				st.TargetY[i] = st.BestY + (ss.Rng.Float64()-0.5)*2*jitter
				st.SwimCount[i] = bfoChemoSteps
				st.PrevFit[i] = fit
				continue
			}
		}

		// Chemotaxis decision
		if st.SwimCount[i] <= 0 {
			if fit > st.PrevFit[i] {
				// Keep swimming — fitness improving
				st.SwimCount[i] = bfoChemoSteps
			} else {
				// Tumble: gradient-guided direction selection
				st.SwimDir[i] = bfoGradientTumbleAdaptive(bot, ss, probeDist)
				st.SwimCount[i] = bfoChemoSteps
			}
		}
		st.SwimCount[i]--

		// Small random perturbation during swim (decreases over time)
		perturbRate := bfoTumbleRate * (1.0 - 0.5*progress)
		if ss.Rng.Float64() < perturbRate {
			st.SwimDir[i] += (ss.Rng.Float64() - 0.5) * 0.6
		}

		// Compute chemotaxis target from swim direction
		stepDist := probeDist * 0.5
		tx := bot.X + math.Cos(st.SwimDir[i])*stepDist
		ty := bot.Y + math.Sin(st.SwimDir[i])*stepDist

		// Blend target toward global best (adaptive strength)
		gbestW := bfoGBestWStart + (bfoGBestWEnd-bfoGBestWStart)*progress
		if st.BestIdx >= 0 && st.BestF > 0 {
			tx = tx*(1-gbestW) + st.BestX*gbestW
			ty = ty*(1-gbestW) + st.BestY*gbestW
		}

		// Blend toward personal best (cognitive component)
		if st.PBestF[i] > -1e17 {
			pW := 0.05 + 0.15*progress
			tx = tx*(1-pW) + st.PBestX[i]*pW
			ty = ty*(1-pW) + st.PBestY[i]*pW
		}

		st.TargetX[i] = tx
		st.TargetY[i] = ty
		st.PrevFit[i] = fit
	}

	// Best-bot local random walk around GlobalBest for fine-grained exploitation
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		bi := st.BestIdx
		wx := st.BestX + (ss.Rng.Float64()-0.5)*2*bfoLocalWalkR
		wy := st.BestY + (ss.Rng.Float64()-0.5)*2*bfoLocalWalkR
		if wx < 0 {
			wx = 0
		}
		if wx > ss.ArenaW {
			wx = ss.ArenaW
		}
		if wy < 0 {
			wy = 0
		}
		if wy > ss.ArenaH {
			wy = ss.ArenaH
		}
		wf := distanceFitnessPt(ss, wx, wy)
		if wf > st.BestF {
			st.BestF = wf
			st.BestX = wx
			st.BestY = wy
		}
		st.TargetX[bi] = wx
		st.TargetY[bi] = wy
	}

	// Periodic grid rescan: systematically sample the landscape to find
	// the global optimum, critical for deceptive landscapes like Schwefel.
	if st.Tick > 0 && st.Tick%bfoGridRescanRate == 0 {
		bfoGridRescan(ss)
	}

	// Phase 3: Reproduction — fittest half clones replace least fit half
	if st.CycleTimer >= bfoReproInterval {
		st.CycleTimer = 0
		bfoReproduce(ss)
	}

	// Phase 4: Elimination-dispersal — random bots teleport
	// Adaptive: reduce disruption in late phase to allow convergence
	elimProb := bfoElimProbEarly + (bfoElimProbLate-bfoElimProbEarly)*progress
	for i := range ss.Bots {
		if ss.Rng.Float64() < elimProb {
			ss.Bots[i].X = ss.Rng.Float64() * ss.ArenaW
			ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
			st.Health[i] = 0
			st.PrevFit[i] = 0
			st.SwimDir[i] = ss.Rng.Float64() * 2 * math.Pi
			st.TargetX[i] = ss.Bots[i].X
			st.TargetY[i] = ss.Bots[i].Y
			if i < len(st.PBestF) {
				st.PBestF[i] = -1e18
				st.PBestX[i] = ss.Bots[i].X
				st.PBestY[i] = ss.Bots[i].Y
			}
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		healthNorm := st.Health[i]
		if healthNorm > 100 {
			healthNorm = 100
		}
		ss.Bots[i].BFOHealth = int(healthNorm)
		if ss.Bots[i].BFOHealth < 0 {
			ss.Bots[i].BFOHealth = 0
		}
		ss.Bots[i].BFOSwimming = 1
		if st.SwimCount[i] <= 0 {
			ss.Bots[i].BFOSwimming = 0 // tumbling
		}
		ss.Bots[i].BFONutrient = int(st.Fitness[i])
		if ss.Bots[i].BFONutrient > 100 {
			ss.Bots[i].BFONutrient = 100
		}
	}
}

// bfoGradientTumbleAdaptive probes several directions around the bot at
// the given probe distance and returns a direction biased toward higher fitness.
func bfoGradientTumbleAdaptive(bot *SwarmBot, ss *SwarmState, probeDist float64) float64 {
	bestDir := ss.Rng.Float64() * 2 * math.Pi
	bestFit := -1e18

	baseFit := distanceFitness(bot, ss)

	for d := 0; d < bfoGradientDirs; d++ {
		angle := float64(d) * (2 * math.Pi / float64(bfoGradientDirs))
		// Add small random jitter to avoid deterministic patterns
		angle += (ss.Rng.Float64() - 0.5) * 0.3

		probeX := bot.X + math.Cos(angle)*probeDist
		probeY := bot.Y + math.Sin(angle)*probeDist

		// Clamp to arena
		if probeX < 0 {
			probeX = 0
		}
		if probeX > ss.ArenaW {
			probeX = ss.ArenaW
		}
		if probeY < 0 {
			probeY = 0
		}
		if probeY > ss.ArenaH {
			probeY = ss.ArenaH
		}

		// Create temporary probe bot to evaluate fitness at probe position
		probeBot := *bot
		probeBot.X = probeX
		probeBot.Y = probeY
		probeFit := distanceFitness(&probeBot, ss)

		if probeFit > bestFit {
			bestFit = probeFit
			bestDir = angle
		}
	}

	// Mix gradient direction with some randomness for exploration
	if bestFit > baseFit {
		// Strong bias toward gradient direction
		return bestDir + (ss.Rng.Float64()-0.5)*0.4
	}
	// No improvement found — fully random tumble
	return ss.Rng.Float64() * 2 * math.Pi
}

// bfoReproduce performs reproduction: healthiest half replaces least healthy half.
// The healthy bacteria "clone" their position (move toward donor) and swim direction.
func bfoReproduce(ss *SwarmState) {
	st := ss.BFO
	n := len(ss.Bots)
	if n < 4 {
		return
	}

	// Find mean health as threshold
	midHealth := 0.0
	for i := range ss.Bots {
		midHealth += st.Health[i]
	}
	midHealth /= float64(n)

	// Clone fittest traits to least fit
	// Half of weak bots move toward GlobalBest (exploitation),
	// half move toward random healthy donors (diversity)
	for i := range ss.Bots {
		if st.Health[i] < midHealth {
			t := 0.3 + ss.Rng.Float64()*0.4 // blend factor 0.3-0.7
			if st.BestF > 0 && ss.Rng.Float64() < 0.5 {
				// Move toward GlobalBest with jitter
				jitter := 15.0
				ss.Bots[i].X = ss.Bots[i].X*(1-t) + st.BestX*t + (ss.Rng.Float64()-0.5)*jitter
				ss.Bots[i].Y = ss.Bots[i].Y*(1-t) + st.BestY*t + (ss.Rng.Float64()-0.5)*jitter
				st.Health[i] = 0
				st.PrevFit[i] = 0
				if i < len(st.PBestF) {
					st.PBestF[i] = st.BestF
					st.PBestX[i] = st.BestX
					st.PBestY[i] = st.BestY
				}
			} else {
				// Find a random healthy donor
				donor := ss.Rng.Intn(n)
				for attempts := 0; attempts < 8 && st.Health[donor] < midHealth; attempts++ {
					donor = ss.Rng.Intn(n)
				}
				if st.Health[donor] >= midHealth {
					// Move weak bacterium toward donor position (partial teleport)
					ss.Bots[i].X = ss.Bots[i].X*(1-t) + ss.Bots[donor].X*t
					ss.Bots[i].Y = ss.Bots[i].Y*(1-t) + ss.Bots[donor].Y*t
					st.SwimDir[i] = st.SwimDir[donor] + (ss.Rng.Float64()-0.5)*0.5
					st.Health[i] = st.Health[donor] * 0.5
					st.PrevFit[i] = 0 // force re-evaluation
					// Inherit donor's personal best knowledge
					if i < len(st.PBestF) && donor < len(st.PBestF) {
						st.PBestF[i] = st.PBestF[donor]
						st.PBestX[i] = st.PBestX[donor]
						st.PBestY[i] = st.PBestY[donor]
					}
				}
			}
			st.SwimDir[i] = ss.Rng.Float64() * 2 * math.Pi
		}
	}
}

// ApplyBFO steers a bot using Bacterial Foraging chemotaxis.
func ApplyBFO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.BFO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.BFO
	if idx >= len(st.TargetX) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// LED color based on fitness: green (high) → red (low)
	fit := st.Fitness[idx]
	if fit < 0 {
		fit = 0
	}
	if fit > 100 {
		fit = 100
	}
	g := uint8(fit * 2.55)
	r := uint8((100 - fit) * 2.55)
	bot.LEDColor = [3]uint8{r, g, 50}

	// Highlight global best in gold
	if idx == st.BestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0}
	}

	// Move toward target using shared movement function (5x speed)
	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, bfoSpeedMult)
}

// bfoGridRescan systematically evaluates a grid across the arena and injects
// the best positions into the worst bots. Critical for deceptive landscapes.
func bfoGridRescan(ss *SwarmState) {
	st := ss.BFO
	n := len(ss.Bots)
	if n < 4 {
		return
	}

	aw := ss.ArenaW
	ah := ss.ArenaH

	type gridPt struct {
		x, y, f float64
	}
	topPts := make([]gridPt, 0, bfoGridInjectTop)

	for gx := 0; gx < bfoGridRescanSide; gx++ {
		for gy := 0; gy < bfoGridRescanSide; gy++ {
			px := (float64(gx) + 0.5) * aw / float64(bfoGridRescanSide)
			py := (float64(gy) + 0.5) * ah / float64(bfoGridRescanSide)
			f := distanceFitnessPt(ss, px, py)

			// Update global best from grid
			if f > st.BestF {
				st.BestF = f
				st.BestX = px
				st.BestY = py
			}

			// Insert into top list (sorted descending)
			inserted := false
			for ti := range topPts {
				if f > topPts[ti].f {
					topPts = append(topPts, gridPt{})
					copy(topPts[ti+1:], topPts[ti:])
					topPts[ti] = gridPt{px, py, f}
					inserted = true
					break
				}
			}
			if !inserted && len(topPts) < bfoGridInjectTop {
				topPts = append(topPts, gridPt{px, py, f})
			}
			if len(topPts) > bfoGridInjectTop {
				topPts = topPts[:bfoGridInjectTop]
			}
		}
	}

	if len(topPts) == 0 {
		return
	}

	// Find worst-fitness bots to replace
	type idxFit struct {
		idx int
		f   float64
	}
	worst := make([]idxFit, 0, n)
	for i := range ss.Bots {
		if i < len(st.Fitness) {
			worst = append(worst, idxFit{i, st.Fitness[i]})
		}
	}
	// Sort ascending by fitness (simple insertion sort for small n)
	for i := 1; i < len(worst); i++ {
		key := worst[i]
		j := i - 1
		for j >= 0 && worst[j].f > key.f {
			worst[j+1] = worst[j]
			j--
		}
		worst[j+1] = key
	}

	// Inject top grid points into worst bots
	injectCount := bfoGridInjectTop
	if injectCount > len(worst) {
		injectCount = len(worst)
	}
	if injectCount > len(topPts) {
		injectCount = len(topPts)
	}
	for k := 0; k < injectCount; k++ {
		bi := worst[k].idx
		jitterX := (ss.Rng.Float64() - 0.5) * 10
		jitterY := (ss.Rng.Float64() - 0.5) * 10
		ss.Bots[bi].X = topPts[k].x + jitterX
		ss.Bots[bi].Y = topPts[k].y + jitterY
		st.TargetX[bi] = topPts[k].x
		st.TargetY[bi] = topPts[k].y
		st.Health[bi] = 0
		st.PrevFit[bi] = 0
		st.SwimDir[bi] = ss.Rng.Float64() * 2 * math.Pi
	}
}
