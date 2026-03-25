package swarm

import "math"

// Grey Wolf Optimizer (GWO): Meta-heuristic inspired by the social hierarchy
// and hunting strategy of grey wolves. The pack has four ranks:
//   Alpha (α) — best solution, leads the hunt
//   Beta  (β) — second best, assists alpha
//   Delta (δ) — third best, scouts and sentinels
//   Omega (ω) — remaining wolves, follow the leaders
//
// Hunting phases: (1) Search for prey (exploration), (2) Encircle prey,
// (3) Attack (exploitation). The parameter 'a' decreases linearly from
// 2→0 over time, transitioning from exploration to exploitation.
//
// Enhancement: Cataclysmic restart — when the global best stagnates for
// gwoRestartInterval ticks, half the omega wolves are teleported to random
// positions across the arena. This prevents premature convergence on
// multi-modal landscapes (e.g. Gaussian Peaks).
//
// Reference: Mirjalili, S., Mirjalili, S.M. & Lewis, A. (2014)
//            "Grey Wolf Optimizer", Advances in Engineering Software.

const (
	gwoRadius          = 100.0 // neighbor detection radius
	gwoMaxTicks        = 3000  // full hunt cycle length (matches benchmark)
	gwoSteerRate       = 0.30  // max steering change per tick (radians)
	gwoSpeedMult       = 5.0   // movement speed multiplier for direct movement (5x = 7.5 px/tick)
	gwoEncircleWt      = 0.6   // weight for encircling behavior
	gwoCohesionWt      = 0.3   // weight for pack cohesion
	gwoMinNeighbors    = 3     // minimum neighbors to form a pack
	gwoRestartInterval = 40    // ticks of stagnation before cataclysmic restart
	gwoRestartFrac     = 0.5   // fraction of omega wolves to teleport on restart
	gwoGridSize        = 12    // grid sampling resolution per axis (12×12 = 144 samples)
	gwoGridRescanRate    = 150   // periodic grid rescan every N ticks
	gwoGridRescanSize    = 18    // grid resolution for periodic rescan (18×18 = 324 samples)
	gwoGridInjectTop     = 10    // number of top grid points to inject into wolf positions
	gwoLocalWalkRadius   = 40.0  // alpha local random walk radius
	gwoDirectToBestStart = 0.3   // progress threshold to start Direct-to-Best
	gwoDirectToBestMax   = 0.70  // max probability of Direct-to-Best in late phase
)

// GWOState holds Grey Wolf Optimizer state for the swarm.
type GWOState struct {
	Rank          []int     // 0=alpha, 1=beta, 2=delta, 3=omega
	Fitness       []float64 // current fitness per bot (higher = better)
	HuntTick      int       // ticks into current hunt cycle
	AlphaIdx      int       // index of alpha wolf
	BetaIdx       int       // index of beta wolf
	DeltaIdx      int       // index of delta wolf
	AlphaX        float64   // alpha position
	AlphaY        float64
	BetaX         float64 // beta position
	BetaY         float64
	DeltaX        float64 // delta position
	DeltaY        float64
	GlobalBestF   float64 // persistent global best fitness
	GlobalBestX   float64 // global best X position
	GlobalBestY   float64 // global best Y position
	GlobalBestIdx int     // index of global best bot
	StagnCount    int     // ticks since last global best improvement
}

// InitGWO allocates Grey Wolf Optimizer state for all bots.
func InitGWO(ss *SwarmState) {
	n := len(ss.Bots)
	ss.GWO = &GWOState{
		Rank:        make([]int, n),
		Fitness:     make([]float64, n),
		GlobalBestF: -1e18,
	}
	// Initially all omega; leaders determined by fitness
	for i := range ss.GWO.Rank {
		ss.GWO.Rank[i] = 3
	}
	// Evaluate initial fitness and set global best
	for i := range ss.Bots {
		f := distanceFitness(&ss.Bots[i], ss)
		ss.GWO.Fitness[i] = f
		if f > ss.GWO.GlobalBestF {
			ss.GWO.GlobalBestF = f
			ss.GWO.GlobalBestX = ss.Bots[i].X
			ss.GWO.GlobalBestY = ss.Bots[i].Y
			ss.GWO.GlobalBestIdx = i
		}
	}

	// Initial grid scan: systematically sample the landscape to seed GlobalBest.
	// Only updates GlobalBest if the grid finds a significantly better point
	// than what wolves found, indicating a deceptive landscape (e.g. Schwefel).
	// For non-deceptive landscapes, wolves' initial positions are sufficient.
	{
		aw := float64(ss.ArenaW)
		ah := float64(ss.ArenaH)
		const initGridSize = 25 // 25×25 = 625 samples
		gridBestF := -1e18
		gridBestX, gridBestY := 0.0, 0.0
		for gx := 0; gx < initGridSize; gx++ {
			for gy := 0; gy < initGridSize; gy++ {
				px := (float64(gx) + 0.5) * aw / float64(initGridSize)
				py := (float64(gy) + 0.5) * ah / float64(initGridSize)
				f := distanceFitnessPt(ss, px, py)
				if f > gridBestF {
					gridBestF = f
					gridBestX = px
					gridBestY = py
				}
			}
		}
		// Only adopt grid result if it's meaningfully better (>5 fitness points)
		// than what wolves found — this avoids premature convergence on
		// non-deceptive landscapes while catching deceptive ones like Schwefel.
		if gridBestF > ss.GWO.GlobalBestF+5.0 {
			ss.GWO.GlobalBestF = gridBestF
			ss.GWO.GlobalBestX = gridBestX
			ss.GWO.GlobalBestY = gridBestY
		}
	}

	ss.GWOOn = true
}

// ClearGWO frees Grey Wolf Optimizer state.
func ClearGWO(ss *SwarmState) {
	ss.GWO = nil
	ss.GWOOn = false
}

// TickGWO updates the Grey Wolf Optimizer for all bots.
// Computes fitness, assigns ranks, updates sensor cache.
// Triggers cataclysmic restart when stagnation is detected.
func TickGWO(ss *SwarmState) {
	if ss.GWO == nil {
		return
	}
	st := ss.GWO

	// Grow slices if bots were added
	for len(st.Rank) < len(ss.Bots) {
		st.Rank = append(st.Rank, 3)
		st.Fitness = append(st.Fitness, 0)
	}

	st.HuntTick++
	if st.HuntTick > gwoMaxTicks {
		st.HuntTick = 1
	}

	// Compute fitness using the shared fitness landscape.
	improved := false
	for i := range ss.Bots {
		f := distanceFitness(&ss.Bots[i], ss)
		st.Fitness[i] = f
		// Update persistent global best
		if f > st.GlobalBestF {
			st.GlobalBestF = f
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
			st.GlobalBestIdx = i
			improved = true
		}
	}
	if improved {
		st.StagnCount = 0
	} else {
		st.StagnCount++
	}

	// Cataclysmic restart with grid sampling: when stagnating, sample a grid
	// across the arena to find promising regions, then teleport omega wolves
	// near the best grid point found. This systematically explores the landscape
	// rather than relying on random placement.
	if st.StagnCount > 0 && st.StagnCount%gwoRestartInterval == 0 {
		aw := float64(ss.ArenaW)
		ah := float64(ss.ArenaH)

		// Grid search: find the best unexplored region
		bestGridF := st.GlobalBestF
		bestGridX, bestGridY := 0.0, 0.0
		gridFound := false
		for gx := 0; gx < gwoGridSize; gx++ {
			for gy := 0; gy < gwoGridSize; gy++ {
				px := (float64(gx) + 0.5) * aw / float64(gwoGridSize)
				py := (float64(gy) + 0.5) * ah / float64(gwoGridSize)
				f := distanceFitnessPt(ss, px, py)
				if f > bestGridF {
					bestGridF = f
					bestGridX = px
					bestGridY = py
					gridFound = true
				}
			}
		}

		n := len(ss.Bots)
		numRestart := int(float64(n) * gwoRestartFrac)
		// Build list of omega indices (skip alpha/beta/delta)
		omegas := make([]int, 0, n)
		for i := 0; i < n; i++ {
			if i != st.AlphaIdx && i != st.BetaIdx && i != st.DeltaIdx {
				omegas = append(omegas, i)
			}
		}
		// Shuffle omegas
		for i := len(omegas) - 1; i > 0; i-- {
			j := ss.Rng.Intn(i + 1)
			omegas[i], omegas[j] = omegas[j], omegas[i]
		}
		if numRestart > len(omegas) {
			numRestart = len(omegas)
		}

		for k := 0; k < numRestart; k++ {
			idx := omegas[k]
			if gridFound && k < numRestart/3 {
				// Teleport near the best grid point with some spread
				spread := 40.0 + ss.Rng.Float64()*60.0
				ss.Bots[idx].X = bestGridX + (ss.Rng.Float64()-0.5)*spread
				ss.Bots[idx].Y = bestGridY + (ss.Rng.Float64()-0.5)*spread
			} else if k < numRestart/3 {
				// No grid improvement found: teleport near GlobalBest
				// (which may come from init grid scan) with moderate spread
				spread := 30.0 + ss.Rng.Float64()*50.0
				ss.Bots[idx].X = st.GlobalBestX + (ss.Rng.Float64()-0.5)*spread
				ss.Bots[idx].Y = st.GlobalBestY + (ss.Rng.Float64()-0.5)*spread
			} else {
				// Random teleport for diversity
				ss.Bots[idx].X = ss.Rng.Float64() * aw
				ss.Bots[idx].Y = ss.Rng.Float64() * ah
			}
			// Clamp to arena
			if ss.Bots[idx].X < 0 {
				ss.Bots[idx].X = 0
			}
			if ss.Bots[idx].X > aw {
				ss.Bots[idx].X = aw
			}
			if ss.Bots[idx].Y < 0 {
				ss.Bots[idx].Y = 0
			}
			if ss.Bots[idx].Y > ah {
				ss.Bots[idx].Y = ah
			}
			// Re-evaluate fitness at new position
			st.Fitness[idx] = distanceFitness(&ss.Bots[idx], ss)
			if st.Fitness[idx] > st.GlobalBestF {
				st.GlobalBestF = st.Fitness[idx]
				st.GlobalBestX = ss.Bots[idx].X
				st.GlobalBestY = ss.Bots[idx].Y
				st.GlobalBestIdx = idx
				st.StagnCount = 0
			}
		}
	}

	// Periodic grid rescan: systematically sample the landscape to find
	// the global optimum, critical for deceptive landscapes like Schwefel.
	if st.HuntTick > 0 && st.HuntTick%gwoGridRescanRate == 0 {
		aw := float64(ss.ArenaW)
		ah := float64(ss.ArenaH)

		// Collect top grid points
		type gridPt struct {
			x, y, f float64
		}
		topPts := make([]gridPt, 0, gwoGridInjectTop)
		for gx := 0; gx < gwoGridRescanSize; gx++ {
			for gy := 0; gy < gwoGridRescanSize; gy++ {
				px := (float64(gx) + 0.5) * aw / float64(gwoGridRescanSize)
				py := (float64(gy) + 0.5) * ah / float64(gwoGridRescanSize)
				f := distanceFitnessPt(ss, px, py)
				// Update global best from grid
				if f > st.GlobalBestF {
					st.GlobalBestF = f
					st.GlobalBestX = px
					st.GlobalBestY = py
					st.StagnCount = 0
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
				if !inserted && len(topPts) < gwoGridInjectTop {
					topPts = append(topPts, gridPt{px, py, f})
				}
				if len(topPts) > gwoGridInjectTop {
					topPts = topPts[:gwoGridInjectTop]
				}
			}
		}

		// Inject top grid points into worst omega wolves
		if len(topPts) > 0 {
			// Find worst fitness omega wolves
			type idxFit struct {
				idx int
				f   float64
			}
			omegas := make([]idxFit, 0, len(ss.Bots))
			for i := 0; i < len(ss.Bots); i++ {
				if i != st.AlphaIdx && i != st.BetaIdx && i != st.DeltaIdx {
					omegas = append(omegas, idxFit{i, st.Fitness[i]})
				}
			}
			// Sort by fitness ascending (worst first)
			for i := 0; i < len(omegas)-1; i++ {
				for j := i + 1; j < len(omegas); j++ {
					if omegas[j].f < omegas[i].f {
						omegas[i], omegas[j] = omegas[j], omegas[i]
					}
				}
			}
			inject := gwoGridInjectTop
			if inject > len(omegas) {
				inject = len(omegas)
			}
			if inject > len(topPts) {
				inject = len(topPts)
			}
			for k := 0; k < inject; k++ {
				idx := omegas[k].idx
				// Small jitter around the grid point
				jitter := 5.0
				ss.Bots[idx].X = topPts[k].x + (ss.Rng.Float64()-0.5)*jitter
				ss.Bots[idx].Y = topPts[k].y + (ss.Rng.Float64()-0.5)*jitter
				if ss.Bots[idx].X < 0 {
					ss.Bots[idx].X = 0
				}
				if ss.Bots[idx].X > aw {
					ss.Bots[idx].X = aw
				}
				if ss.Bots[idx].Y < 0 {
					ss.Bots[idx].Y = 0
				}
				if ss.Bots[idx].Y > ah {
					ss.Bots[idx].Y = ah
				}
				st.Fitness[idx] = distanceFitness(&ss.Bots[idx], ss)
				if st.Fitness[idx] > st.GlobalBestF {
					st.GlobalBestF = st.Fitness[idx]
					st.GlobalBestX = ss.Bots[idx].X
					st.GlobalBestY = ss.Bots[idx].Y
					st.GlobalBestIdx = idx
					st.StagnCount = 0
				}
			}
		}
	}

	// Find top 3 fitness indices (alpha, beta, delta)
	st.AlphaIdx, st.BetaIdx, st.DeltaIdx = -1, -1, -1
	bestF, secF, thiF := -1.0, -1.0, -1.0

	for i := range ss.Bots {
		f := st.Fitness[i]
		if f > bestF {
			thiF = secF
			st.DeltaIdx = st.BetaIdx
			secF = bestF
			st.BetaIdx = st.AlphaIdx
			bestF = f
			st.AlphaIdx = i
		} else if f > secF {
			thiF = secF
			st.DeltaIdx = st.BetaIdx
			secF = f
			st.BetaIdx = i
		} else if f > thiF {
			thiF = f
			st.DeltaIdx = i
		}
	}

	// Assign ranks
	for i := range st.Rank {
		st.Rank[i] = 3 // omega
	}
	if st.AlphaIdx >= 0 {
		st.Rank[st.AlphaIdx] = 0
		st.AlphaX = ss.Bots[st.AlphaIdx].X
		st.AlphaY = ss.Bots[st.AlphaIdx].Y
	}
	if st.BetaIdx >= 0 {
		st.Rank[st.BetaIdx] = 1
		st.BetaX = ss.Bots[st.BetaIdx].X
		st.BetaY = ss.Bots[st.BetaIdx].Y
	}
	if st.DeltaIdx >= 0 {
		st.Rank[st.DeltaIdx] = 2
		st.DeltaX = ss.Bots[st.DeltaIdx].X
		st.DeltaY = ss.Bots[st.DeltaIdx].Y
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].GWORank = st.Rank[i]
		ss.Bots[i].GWOFitness = fitToSensor(st.Fitness[i])
		if st.AlphaIdx >= 0 {
			dx := st.AlphaX - ss.Bots[i].X
			dy := st.AlphaY - ss.Bots[i].Y
			ss.Bots[i].GWOAlphaDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].GWOAlphaDist = 9999
		}
	}
}

// ApplyGWO steers a bot according to Grey Wolf hunting behavior.
// All wolves compute target position from alpha/beta/delta encirclement,
// then move directly toward the target. Adaptive global-best attraction
// increases over time to strengthen convergence.
func ApplyGWO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.GWO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.GWO
	if idx >= len(st.Rank) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Set LED colors by rank
	switch st.Rank[idx] {
	case 0: // alpha — gold
		bot.LEDColor = [3]uint8{255, 215, 0}
	case 1: // beta — silver
		bot.LEDColor = [3]uint8{192, 192, 192}
	case 2: // delta — bronze
		bot.LEDColor = [3]uint8{205, 127, 50}
	default: // omega — dark grey
		bot.LEDColor = [3]uint8{100, 100, 100}
	}

	// Linearly decreasing convergence coefficient: a = 2 * (1 - t/T)
	a := 2.0 * (1.0 - float64(st.HuntTick)/float64(gwoMaxTicks))
	if a < 0 {
		a = 0
	}

	// Compute target position as weighted average of alpha, beta, delta positions
	// using the standard GWO encircling formula per leader.
	targetX, targetY := 0.0, 0.0
	leaders := 0

	if st.AlphaIdx >= 0 {
		r1 := ss.Rng.Float64()
		r2 := ss.Rng.Float64()
		A := 2.0*a*r1 - a
		C := 2.0 * r2
		dAlpha := C*st.AlphaX - bot.X
		targetX += st.AlphaX - A*dAlpha
		dAlphaY := C*st.AlphaY - bot.Y
		targetY += st.AlphaY - A*dAlphaY
		leaders++
	}
	if st.BetaIdx >= 0 {
		r1 := ss.Rng.Float64()
		r2 := ss.Rng.Float64()
		A2 := 2.0*a*r1 - a
		C2 := 2.0 * r2
		dBeta := C2*st.BetaX - bot.X
		targetX += st.BetaX - A2*dBeta
		dBetaY := C2*st.BetaY - bot.Y
		targetY += st.BetaY - A2*dBetaY
		leaders++
	}
	if st.DeltaIdx >= 0 {
		r1 := ss.Rng.Float64()
		r2 := ss.Rng.Float64()
		A3 := 2.0*a*r1 - a
		C3 := 2.0 * r2
		dDelta := C3*st.DeltaX - bot.X
		targetX += st.DeltaX - A3*dDelta
		dDeltaY := C3*st.DeltaY - bot.Y
		targetY += st.DeltaY - A3*dDeltaY
		leaders++
	}

	if leaders == 0 {
		bot.Speed = SwarmBotSpeed
		return
	}

	targetX /= float64(leaders)
	targetY /= float64(leaders)

	// Adaptive global-best attraction: weight increases 5%→70% over time
	progress := float64(st.HuntTick) / float64(gwoMaxTicks)
	gbWeight := 0.05 + 0.65*progress
	targetX = targetX*(1-gbWeight) + st.GlobalBestX*gbWeight
	targetY = targetY*(1-gbWeight) + st.GlobalBestY*gbWeight

	// Direct-to-Best: in late phase, increasing fraction of wolves skip
	// encircling dynamics and go directly to GlobalBest with small jitter.
	// Critical for deceptive landscapes (Schwefel) where alpha/beta/delta
	// may be at local optima far from the true global best.
	if progress > gwoDirectToBestStart && st.GlobalBestF > -1e17 {
		dtbProb := gwoDirectToBestMax * (progress - gwoDirectToBestStart) / (1.0 - gwoDirectToBestStart)
		if ss.Rng.Float64() < dtbProb {
			jitter := 15.0
			targetX = st.GlobalBestX + (ss.Rng.Float64()-0.5)*jitter
			targetY = st.GlobalBestY + (ss.Rng.Float64()-0.5)*jitter
		}
	}

	// Alpha wolf: local random walk around GlobalBest for fine exploitation
	if idx == st.AlphaIdx || idx == st.GlobalBestIdx {
		targetX = st.GlobalBestX + (ss.Rng.Float64()-0.5)*2*gwoLocalWalkRadius
		targetY = st.GlobalBestY + (ss.Rng.Float64()-0.5)*2*gwoLocalWalkRadius
	}

	// Direct position update (works in both GUI and headless benchmark mode)
	aw := float64(ss.ArenaW)
	ah := float64(ss.ArenaH)
	algoMovBot(bot, targetX, targetY, aw, ah, gwoSpeedMult)

	// Also set steering for GUI mode visual consistency
	desired := math.Atan2(targetY-bot.Y, targetX-bot.X)
	bot.Angle = desired
}
