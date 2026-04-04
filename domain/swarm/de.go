package swarm

import (
	"math"
	"math/rand"
	"sort"
)

// Differential Evolution (DE): A population-based stochastic optimization
// algorithm that uses vector differences between population members for
// mutation. DE is particularly effective for continuous optimization problems
// because it self-adapts its step size through the difference vectors.
//
// Strategy: DE/rand/1/bin — the classic variant:
//   1. Mutation:  v_i = x_r1 + F * (x_r2 - x_r3)
//   2. Crossover: u_i = crossover(x_i, v_i) with probability CR
//   3. Selection: x_i = u_i if f(u_i) > f(x_i), else keep x_i
//
// Enhanced with: 5x speed, initial grid scan, periodic grid rescan,
// direct-to-best convergence, GB attraction, best-bot local walk.
//
// Reference: Storn, R. & Price, K. (1997)

const (
	deMaxTicks  = 100   // generation cycle length in ticks (was 500)
	deTrialStep = 30.0  // how far a bot moves toward its trial position per tick
	deSpeedMult = 5.0   // movement speed multiplier (7.5 px/tick)

	// Grid scan parameters
	deInitGridSide       = AlgoGridRescanSize // initial grid scan side (20x20=400)
	deGridRescanInterval = 150 // ticks between grid re-scans
	deGridRescanSide     = AlgoGridRescanSize // grid side for re-scan (20x20=400)
	deGridInjectCount    = 15  // best grid positions to inject per rescan

	// Direct-to-Best parameters
	deDtbStartProg = 0.10  // progress threshold for direct-to-best
	deDtbMaxProb   = 0.80  // max probability of direct-to-best at progress=1

	// Global-Best attraction
	deGBWeightStart = 0.05 // GB attraction at progress=0
	deGBWeightEnd   = 0.70 // GB attraction at progress=1

	// Local walk
	deLocalWalkRadius = 40.0 // radius for best-bot local walk around GlobalBest
)

// DEState holds Differential Evolution state for the swarm.
type DEState struct {
	// Per-bot state
	Fitness  []float64 // current fitness per bot (higher = better)
	TrialX   []float64 // trial (mutant) position X per bot
	TrialY   []float64 // trial (mutant) position Y per bot
	TrialF   []float64 // trial fitness (evaluated when bot arrives)
	Moving   []bool    // true while bot is moving toward trial position
	IsDirect []bool    // true if bot is doing direct-to-best this gen

	// Global best tracking (persistent across generations)
	BestIdx int     // index of best individual
	BestF   float64 // best fitness found
	BestX   float64 // best known X position
	BestY   float64 // best known Y position

	// Parameters
	DifferentialWeight float64 // F: mutation scaling factor (default 0.8)
	CrossoverRate      float64 // CR: crossover probability (default 0.5)
	GenTick            int     // ticks into current generation
}

// InitDE allocates Differential Evolution state for all bots.
func InitDE(ss *SwarmState) {
	n := len(ss.Bots)
	ss.DE = &DEState{
		Fitness:            make([]float64, n),
		TrialX:             make([]float64, n),
		TrialY:             make([]float64, n),
		TrialF:             make([]float64, n),
		Moving:             make([]bool, n),
		IsDirect:           make([]bool, n),
		BestF:              -1e9,
		DifferentialWeight: 0.8,
		CrossoverRate:      0.5,
	}

	// Evaluate initial fitness for all bots.
	for i := range ss.Bots {
		ss.DE.Fitness[i] = deFitness(&ss.Bots[i], ss)
		if ss.DE.Fitness[i] > ss.DE.BestF {
			ss.DE.BestF = ss.DE.Fitness[i]
			ss.DE.BestIdx = i
			ss.DE.BestX = ss.Bots[i].X
			ss.DE.BestY = ss.Bots[i].Y
		}
	}

	// Initial grid scan to find global optimum on deceptive landscapes.
	deInitialGridScan(ss)

	ss.DEOn = true
}

// deInitialGridScan evaluates a grid over the arena and updates GlobalBest.
func deInitialGridScan(ss *SwarmState) {
	st := ss.DE
	margin := SwarmEdgeMargin
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin

	pts := make([]gridPt, 0, deInitGridSide*deInitGridSide)
	for gx := 0; gx < deInitGridSide; gx++ {
		for gy := 0; gy < deInitGridSide; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(deInitGridSide)
			y := margin + usableH*(float64(gy)+0.5)/float64(deInitGridSide)
			f := distanceFitnessPt(ss, x, y)
			pts = append(pts, gridPt{x, y, f})
		}
	}

	// Sort descending by fitness.
	sort.Slice(pts, func(i, j int) bool { return pts[i].f > pts[j].f })

	// Update GlobalBest if grid found something better.
	if len(pts) > 0 && pts[0].f > st.BestF {
		st.BestF = pts[0].f
		st.BestX = pts[0].x
		st.BestY = pts[0].y
	}

	// Inject best grid positions into worst bots.
	n := len(ss.Bots)
	inject := deGridInjectCount
	if inject > n {
		inject = n
	}
	if inject > len(pts) {
		inject = len(pts)
	}

	// Find worst bots by fitness.
	ranked := make([]idxFit, n)
	for i := 0; i < n; i++ {
		ranked[i] = idxFit{i, st.Fitness[i]}
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].f < ranked[j].f })

	for i := 0; i < inject; i++ {
		bi := ranked[i].idx
		jx := pts[i].x + (ss.Rng.Float64()*2-1)*5
		jy := pts[i].y + (ss.Rng.Float64()*2-1)*5
		jx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, jx))
		jy = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, jy))
		ss.Bots[bi].X = jx
		ss.Bots[bi].Y = jy
		st.Fitness[bi] = distanceFitnessPt(ss, jx, jy)
		if st.Fitness[bi] > st.BestF {
			st.BestF = st.Fitness[bi]
			st.BestX = jx
			st.BestY = jy
			st.BestIdx = bi
		}
	}
}

// ClearDE frees Differential Evolution state.
func ClearDE(ss *SwarmState) {
	ss.DE = nil
	ss.DEOn = false
}

// TickDE runs one tick of the Differential Evolution algorithm.
func TickDE(ss *SwarmState) {
	if ss.DE == nil {
		return
	}
	st := ss.DE
	n := len(ss.Bots)

	// Grow slices if bots were added.
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.TrialX = append(st.TrialX, 0)
		st.TrialY = append(st.TrialY, 0)
		st.TrialF = append(st.TrialF, 0)
		st.Moving = append(st.Moving, false)
		st.IsDirect = append(st.IsDirect, false)
	}

	st.GenTick++

	// Periodic grid rescan.
	if ss.Tick > 0 && ss.Tick%deGridRescanInterval == 0 {
		dePeriodicGridRescan(ss)
	}

	progress := float64(ss.Tick) / 3000.0
	if progress > 1 {
		progress = 1
	}

	// New generation: create trial vectors.
	if st.GenTick >= deMaxTicks || st.GenTick == 1 {
		if st.GenTick >= deMaxTicks {
			// Selection: evaluate trial fitness and decide replacement.
			for i := range ss.Bots {
				if st.Moving[i] {
					st.TrialF[i] = deFitness(&ss.Bots[i], ss)
					if st.TrialF[i] >= st.Fitness[i] {
						st.Fitness[i] = st.TrialF[i]
					}
					if st.Fitness[i] > st.BestF {
						st.BestF = st.Fitness[i]
						st.BestX = ss.Bots[i].X
						st.BestY = ss.Bots[i].Y
						st.BestIdx = i
					}
				}
				st.Moving[i] = false
				st.IsDirect[i] = false
			}
			st.GenTick = 1
		}

		// Update global best.
		for i := range ss.Bots {
			if st.Fitness[i] > st.BestF {
				st.BestF = st.Fitness[i]
				st.BestIdx = i
				st.BestX = ss.Bots[i].X
				st.BestY = ss.Bots[i].Y
			}
		}

		// Direct-to-Best probability ramps linearly.
		dtbProb := 0.0
		if progress > deDtbStartProg {
			dtbProb = (progress - deDtbStartProg) / (1.0 - deDtbStartProg) * deDtbMaxProb
		}

		// GB attraction weight ramps linearly.
		gbWeight := deGBWeightStart + (deGBWeightEnd-deGBWeightStart)*progress

		// Generate trial vectors.
		for i := 0; i < n; i++ {
			// Direct-to-Best: skip DE mutation and go straight to GlobalBest.
			if st.BestF > -1e8 && ss.Rng.Float64() < dtbProb {
				tx := st.BestX + (ss.Rng.Float64()*2-1)*7.5
				ty := st.BestY + (ss.Rng.Float64()*2-1)*7.5
				tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
				ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))
				st.TrialX[i] = tx
				st.TrialY[i] = ty
				st.Moving[i] = true
				st.IsDirect[i] = true
				continue
			}

			// DE/rand/1/bin mutation + crossover.
			r1, r2, r3 := dePickThree(ss.Rng, n, i)

			vx := ss.Bots[r1].X + st.DifferentialWeight*(ss.Bots[r2].X-ss.Bots[r3].X)
			vy := ss.Bots[r1].Y + st.DifferentialWeight*(ss.Bots[r2].Y-ss.Bots[r3].Y)

			// Binomial crossover.
			tx, ty := ss.Bots[i].X, ss.Bots[i].Y
			jrand := ss.Rng.Intn(2)

			if ss.Rng.Float64() < st.CrossoverRate || jrand == 0 {
				tx = vx
			}
			if ss.Rng.Float64() < st.CrossoverRate || jrand == 1 {
				ty = vy
			}

			// Apply GB attraction: pull trial toward GlobalBest.
			if st.BestF > -1e8 {
				tx += gbWeight * (st.BestX - tx)
				ty += gbWeight * (st.BestY - ty)
			}

			// Clamp trial position to arena bounds.
			tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
			ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))

			st.TrialX[i] = tx
			st.TrialY[i] = ty
			st.Moving[i] = true
			st.IsDirect[i] = false
		}
	}

	// Best-bot local random walk around GlobalBest.
	if st.BestF > -1e8 {
		bestBot := -1
		bestDist := math.MaxFloat64
		for i := range ss.Bots {
			ddx := ss.Bots[i].X - st.BestX
			ddy := ss.Bots[i].Y - st.BestY
			d := ddx*ddx + ddy*ddy
			if d < bestDist {
				bestDist = d
				bestBot = i
			}
		}
		if bestBot >= 0 {
			rwx := st.BestX + (ss.Rng.Float64()*2-1)*deLocalWalkRadius
			rwy := st.BestY + (ss.Rng.Float64()*2-1)*deLocalWalkRadius
			rwx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, rwx))
			rwy = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, rwy))
			rwf := distanceFitnessPt(ss, rwx, rwy)
			if rwf > st.BestF {
				st.BestF = rwf
				st.BestX = rwx
				st.BestY = rwy
				st.BestIdx = bestBot
			}
			// Move best bot toward the local walk target.
			st.TrialX[bestBot] = rwx
			st.TrialY[bestBot] = rwy
			st.Moving[bestBot] = true
		}
	}

	// Update sensor cache.
	for i := range ss.Bots {
		ss.Bots[i].DEFitness = int(st.Fitness[i] * 100)
		if st.BestIdx >= 0 && st.BestIdx < n {
			dx := ss.Bots[st.BestIdx].X - ss.Bots[i].X
			dy := ss.Bots[st.BestIdx].Y - ss.Bots[i].Y
			ss.Bots[i].DEBestDist = int(math.Sqrt(dx*dx + dy*dy))
		}
		if st.Moving[i] {
			ss.Bots[i].DEPhase = 1 // moving toward trial
		} else {
			ss.Bots[i].DEPhase = 0 // idle
		}
	}
}

// dePeriodicGridRescan evaluates a grid over the arena and injects best
// positions into worst bots.
func dePeriodicGridRescan(ss *SwarmState) {
	st := ss.DE
	n := len(ss.Bots)
	margin := SwarmEdgeMargin
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin

	pts := make([]gridPt, 0, deGridRescanSide*deGridRescanSide)
	for gx := 0; gx < deGridRescanSide; gx++ {
		for gy := 0; gy < deGridRescanSide; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(deGridRescanSide)
			y := margin + usableH*(float64(gy)+0.5)/float64(deGridRescanSide)
			f := distanceFitnessPt(ss, x, y)
			pts = append(pts, gridPt{x, y, f})
		}
	}

	sort.Slice(pts, func(i, j int) bool { return pts[i].f > pts[j].f })

	// Update GlobalBest from grid.
	if len(pts) > 0 && pts[0].f > st.BestF {
		st.BestF = pts[0].f
		st.BestX = pts[0].x
		st.BestY = pts[0].y
	}

	// Inject best grid positions into worst bots.
	inject := deGridInjectCount
	if inject > n {
		inject = n
	}
	if inject > len(pts) {
		inject = len(pts)
	}

	ranked := make([]idxFit, n)
	for i := 0; i < n; i++ {
		ranked[i] = idxFit{i, st.Fitness[i]}
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].f < ranked[j].f })

	for i := 0; i < inject; i++ {
		bi := ranked[i].idx
		jx := pts[i].x + (ss.Rng.Float64()*2-1)*5
		jy := pts[i].y + (ss.Rng.Float64()*2-1)*5
		jx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, jx))
		jy = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, jy))
		ss.Bots[bi].X = jx
		ss.Bots[bi].Y = jy
		st.Fitness[bi] = distanceFitnessPt(ss, jx, jy)
		st.Moving[bi] = false
		if st.Fitness[bi] > st.BestF {
			st.BestF = st.Fitness[bi]
			st.BestX = jx
			st.BestY = jy
			st.BestIdx = bi
		}
	}
}

// ApplyDE steers a bot toward its trial position during the current generation.
func ApplyDE(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.DE == nil {
		bot.Speed = 0
		return
	}
	st := ss.DE
	if idx >= len(st.Moving) {
		bot.Speed = 0
		return
	}

	if !st.Moving[idx] {
		bot.Speed = 0
		bot.LEDColor = [3]uint8{60, 60, 120}
		return
	}

	// Move directly toward trial position.
	algoMovBot(bot, st.TrialX[idx], st.TrialY[idx], ss.ArenaW, ss.ArenaH, deSpeedMult)

	// Check arrival.
	dx := st.TrialX[idx] - bot.X
	dy := st.TrialY[idx] - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist < deTrialStep {
		st.Moving[idx] = false
		st.TrialF[idx] = deFitness(bot, ss)
		if st.TrialF[idx] >= st.Fitness[idx] {
			st.Fitness[idx] = st.TrialF[idx]
		}
		if st.Fitness[idx] > st.BestF {
			st.BestF = st.Fitness[idx]
			st.BestX = bot.X
			st.BestY = bot.Y
			st.BestIdx = idx
		}
		return
	}

	// LED color.
	if st.IsDirect[idx] {
		bot.LEDColor = [3]uint8{30, 220, 50} // green for direct-to-best
	} else {
		fit01 := math.Min(math.Max(st.Fitness[idx]/100.0, 0), 1)
		g := uint8(80 + fit01*175)
		bot.LEDColor = [3]uint8{30, g, 50}
	}
}

// deFitness evaluates the fitness of a bot.
func deFitness(bot *SwarmBot, ss *SwarmState) float64 {
	return distanceFitness(bot, ss)
}

// dePickThree selects three distinct random indices from [0, n) that differ
// from exclude.
func dePickThree(rng *rand.Rand, n, exclude int) (int, int, int) {
	r1 := exclude
	for r1 == exclude {
		r1 = rng.Intn(n)
	}
	r2 := exclude
	for r2 == exclude || r2 == r1 {
		r2 = rng.Intn(n)
	}
	r3 := exclude
	for r3 == exclude || r3 == r1 || r3 == r2 {
		r3 = rng.Intn(n)
	}
	return r1, r2, r3
}
