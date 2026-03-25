package swarm

import "math"

// Equilibrium Optimizer (EO): Physics-inspired metaheuristic based on the
// mass balance equation describing control volume dynamics. Particles
// (concentrations) converge toward an equilibrium pool composed of the best
// solutions found so far. The algorithm balances exploration and exploitation
// via an exponential decay term and a generation rate with random perturbation.
//
// Key mechanisms:
//   - Equilibrium pool: the top-K best positions found so far act as
//     attractors. Each tick, a bot targets a randomly chosen pool member.
//   - Exponential decay: F = a1 * sign(r-0.5) * (e^(-λt) - 1) controls the
//     transition from exploration (large oscillations) to exploitation (small
//     corrections). λ increases over the hunt cycle.
//   - Generation rate: G = G0 * F adds random perturbation around the
//     equilibrium concentration, allowing escape from local optima.
//   - Memory: each bot remembers its personal best position.
//
// Reference: Faramarzi, A. et al. (2020)
//   "Equilibrium optimizer: A novel optimization algorithm",
//   Knowledge-Based Systems 191, 105190.

const (
	eoPoolSize       = 5    // number of best solutions in equilibrium pool
	eoMaxTicks       = 3000 // full cycle length (matches benchmark length)
	eoSteerRate      = 0.30 // max steering per tick (radians)
	eoA1             = 2.0  // exploration control constant
	eoA2             = 1.0  // exploitation control constant
	eoGP             = 0.5  // generation probability
	eoSpeedMult      = 5.0  // movement speed multiplier (7.5 px/tick)
	eoGridRescanRate = 300  // periodic grid rescan every N ticks
	eoGridSize       = 14   // grid resolution (14x14 = 196 samples)
	eoGridInjectTop  = 10   // best grid positions to inject per rescan
)

// EOState holds Equilibrium Optimizer state for the swarm.
type EOState struct {
	Fitness   []float64 // current fitness per bot
	PersonalX []float64 // personal best X per bot
	PersonalY []float64 // personal best Y per bot
	PersonalF []float64 // personal best fitness per bot
	PoolX     []float64 // equilibrium pool X positions
	PoolY     []float64 // equilibrium pool Y positions
	PoolF     []float64 // equilibrium pool fitness values
	CycleTick int       // ticks into current cycle
	BestIdx   int       // index of overall best bot
	BestFit   float64   // best fitness found
	BestX     float64   // persistent global best X position
	BestY     float64   // persistent global best Y position
	Phase     []int     // per-bot phase: 0=exploration, 1=exploitation
	IsDirect  []bool    // per-bot: true if using direct-to-best this tick
	TargetX   []float64 // per-bot target X (computed in Tick, applied in Apply)
	TargetY   []float64 // per-bot target Y (computed in Tick, applied in Apply)
}

// InitEO allocates Equilibrium Optimizer state.
func InitEO(ss *SwarmState) {
	n := len(ss.Bots)
	st := &EOState{
		Fitness:   make([]float64, n),
		PersonalX: make([]float64, n),
		PersonalY: make([]float64, n),
		PersonalF: make([]float64, n),
		PoolX:     make([]float64, eoPoolSize),
		PoolY:     make([]float64, eoPoolSize),
		PoolF:     make([]float64, eoPoolSize),
		Phase:     make([]int, n),
		IsDirect:  make([]bool, n),
		TargetX:   make([]float64, n),
		TargetY:   make([]float64, n),
		BestFit:   -1e9,
	}

	// Initialize personal bests to current positions
	for i := range ss.Bots {
		st.PersonalX[i] = ss.Bots[i].X
		st.PersonalY[i] = ss.Bots[i].Y
		st.PersonalF[i] = -1e9
		st.TargetX[i] = ss.Bots[i].X
		st.TargetY[i] = ss.Bots[i].Y
	}

	// Initialize pool with default spread
	for k := 0; k < eoPoolSize; k++ {
		st.PoolF[k] = -1e9
	}

	ss.EO = st
	ss.EOOn = true
}

// ClearEO frees Equilibrium Optimizer state.
func ClearEO(ss *SwarmState) {
	ss.EO = nil
	ss.EOOn = false
}

// TickEO updates the Equilibrium Optimizer for all bots.
func TickEO(ss *SwarmState) {
	if ss.EO == nil {
		return
	}
	st := ss.EO

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		idx := len(st.Fitness)
		st.Fitness = append(st.Fitness, 0)
		st.PersonalX = append(st.PersonalX, ss.Bots[idx].X)
		st.PersonalY = append(st.PersonalY, ss.Bots[idx].Y)
		st.PersonalF = append(st.PersonalF, -1e9)
		st.Phase = append(st.Phase, 0)
		st.IsDirect = append(st.IsDirect, false)
		st.TargetX = append(st.TargetX, ss.Bots[idx].X)
		st.TargetY = append(st.TargetY, ss.Bots[idx].Y)
	}

	st.CycleTick++
	if st.CycleTick > eoMaxTicks {
		st.CycleTick = 1
	}

	// Time ratio t in (0, 1]
	t := float64(st.CycleTick) / float64(eoMaxTicks)

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Update personal bests
	for i := range ss.Bots {
		if st.Fitness[i] > st.PersonalF[i] {
			st.PersonalF[i] = st.Fitness[i]
			st.PersonalX[i] = ss.Bots[i].X
			st.PersonalY[i] = ss.Bots[i].Y
		}
	}

	// Build equilibrium pool: top-K best personal positions + average
	// Collect all personal bests and find top K-1
	type idxFit struct {
		idx int
		fit float64
	}
	candidates := make([]idxFit, len(ss.Bots))
	for i := range ss.Bots {
		candidates[i] = idxFit{i, st.PersonalF[i]}
	}
	// Simple selection sort for top K-1 (K is small)
	poolCount := eoPoolSize - 1
	if poolCount > len(candidates) {
		poolCount = len(candidates)
	}
	for k := 0; k < poolCount; k++ {
		best := k
		for j := k + 1; j < len(candidates); j++ {
			if candidates[j].fit > candidates[best].fit {
				best = j
			}
		}
		candidates[k], candidates[best] = candidates[best], candidates[k]
		st.PoolX[k] = st.PersonalX[candidates[k].idx]
		st.PoolY[k] = st.PersonalY[candidates[k].idx]
		st.PoolF[k] = candidates[k].fit
	}

	// Last pool entry = average of the pool members (Ceq_avg)
	avgX, avgY, avgF := 0.0, 0.0, 0.0
	for k := 0; k < poolCount; k++ {
		avgX += st.PoolX[k]
		avgY += st.PoolY[k]
		avgF += st.PoolF[k]
	}
	if poolCount > 0 {
		avgX /= float64(poolCount)
		avgY /= float64(poolCount)
		avgF /= float64(poolCount)
	}
	lastIdx := poolCount
	if lastIdx >= eoPoolSize {
		lastIdx = eoPoolSize - 1
	}
	st.PoolX[lastIdx] = avgX
	st.PoolY[lastIdx] = avgY
	st.PoolF[lastIdx] = avgF

	// Track overall best (persistent across entire run)
	for k := 0; k < poolCount; k++ {
		if st.PoolF[k] > st.BestFit {
			st.BestFit = st.PoolF[k]
			st.BestX = st.PoolX[k]
			st.BestY = st.PoolY[k]
		}
	}
	st.BestIdx = candidates[0].idx

	// Periodic grid rescan: systematically sample the arena
	n := len(ss.Bots)
	if st.CycleTick > 0 && st.CycleTick%eoGridRescanRate == 0 && n > 0 {
		eoGridRescan(ss, st)
	}

	// Determine phase per bot and compute targets
	progress := t
	for i := range ss.Bots {
		st.IsDirect[i] = false

		if t < 0.5 {
			st.Phase[i] = 0 // exploration
		} else {
			st.Phase[i] = 1 // exploitation
		}

		// Direct-to-Best: after progress > 0.3, increasing probability
		if progress > 0.3 && st.BestFit > -1e8 {
			directProb := 0.60 * (progress - 0.3) / 0.7 // 0 -> 60%
			if ss.Rng.Float64() < directProb && i != st.BestIdx {
				jitter := 7.5
				tx := st.BestX + (ss.Rng.Float64()*2-1)*jitter
				ty := st.BestY + (ss.Rng.Float64()*2-1)*jitter
				tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
				ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))
				// Evaluate the direct-to-best point
				f := distanceFitnessPt(ss, tx, ty)
				if f > st.BestFit {
					st.BestFit = f
					st.BestX = tx
					st.BestY = ty
				}
				st.TargetX[i] = tx
				st.TargetY[i] = ty
				st.IsDirect[i] = true
				continue
			}
		}

		// Best bot: local random walk around GlobalBest
		if i == st.BestIdx && st.BestFit > -1e8 {
			walkR := 40.0
			tx := st.BestX + (ss.Rng.Float64()*2-1)*walkR
			ty := st.BestY + (ss.Rng.Float64()*2-1)*walkR
			tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
			ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))
			f := distanceFitnessPt(ss, tx, ty)
			if f > st.BestFit {
				st.BestFit = f
				st.BestX = tx
				st.BestY = ty
			}
			st.TargetX[i] = tx
			st.TargetY[i] = ty
			continue
		}

		// Standard EO dynamics: compute target via equilibrium equation
		poolIdx := ss.Rng.Intn(eoPoolSize)
		ceqX := st.PoolX[poolIdx]
		ceqY := st.PoolY[poolIdx]

		r := ss.Rng.Float64()
		lambda := ss.Rng.Float64() * 2.0
		sign := 1.0
		if r < 0.5 {
			sign = -1.0
		}
		F := eoA1 * sign * (math.Exp(-lambda*t) - 1.0)

		G0X, G0Y := 0.0, 0.0
		if ss.Rng.Float64() < eoGP {
			r1 := ss.Rng.Float64()
			r2 := ss.Rng.Float64()
			G0X = (ceqX - ss.Bots[i].X) * r1
			G0Y = (ceqY - ss.Bots[i].Y) * r2
		}
		GX := G0X * F
		GY := G0Y * F

		targetX := ceqX + (ss.Bots[i].X-ceqX)*F + GX*(1.0-F)
		targetY := ceqY + (ss.Bots[i].Y-ceqY)*F + GY*(1.0-F)

		// Adaptive global-best attraction: 5% -> 55%
		if st.BestFit > -1e8 {
			gbWeight := 0.05 + 0.50*t
			targetX = targetX*(1-gbWeight) + st.BestX*gbWeight
			targetY = targetY*(1-gbWeight) + st.BestY*gbWeight
		}

		targetX = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, targetX))
		targetY = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, targetY))

		st.TargetX[i] = targetX
		st.TargetY[i] = targetY
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].EOFitness = fitToSensor(st.Fitness[i])
		ss.Bots[i].EOPhase = st.Phase[i]
		dx := st.PoolX[0] - ss.Bots[i].X
		dy := st.PoolY[0] - ss.Bots[i].Y
		ss.Bots[i].EOEquilDist = int(math.Sqrt(dx*dx + dy*dy))
	}
}

// ApplyEO steers a bot according to the Equilibrium Optimizer.
func ApplyEO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.EO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.EO
	if idx >= len(st.Phase) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Time ratio
	t := float64(st.CycleTick) / float64(eoMaxTicks)

	// Move bot to precomputed target via algoMovBot (5x speed)
	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, eoSpeedMult)

	// Update angle to match movement direction
	dx := st.TargetX[idx] - bot.X
	dy := st.TargetY[idx] - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist > 1.0 {
		bot.Angle = math.Atan2(dy, dx)
	}

	// Set Speed=0 to prevent double-move in GUI physics step
	bot.Speed = 0

	// LED visualization
	if idx == st.BestIdx {
		// Best particle: gold
		bot.LEDColor = [3]uint8{255, 200, 50}
	} else if st.IsDirect[idx] {
		// Direct-to-best: green
		bot.LEDColor = [3]uint8{50, 255, 50}
	} else if st.Phase[idx] == 0 {
		// Exploration phase: blue-violet
		bot.LEDColor = [3]uint8{100, 60, 220}
	} else {
		// Exploitation phase: green-teal gradient
		progress := t
		g := uint8(100 + progress*155)
		bot.LEDColor = [3]uint8{30, g, uint8(80 + progress*80)}
	}
}

// eoGridRescan evaluates a grid of points across the arena and teleports
// the worst particles to the best-discovered grid positions. Critical for
// deceptive landscapes like Schwefel.
func eoGridRescan(ss *SwarmState, st *EOState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	type gridPt struct {
		x, y, f float64
	}
	gridPts := make([]gridPt, 0, eoGridSize*eoGridSize)
	for gx := 0; gx < eoGridSize; gx++ {
		for gy := 0; gy < eoGridSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(eoGridSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(eoGridSize)
			// Small jitter
			x += (ss.Rng.Float64()*2.0 - 1.0) * usableW * 0.02
			y += (ss.Rng.Float64()*2.0 - 1.0) * usableH * 0.02
			f := distanceFitnessPt(ss, x, y)
			gridPts = append(gridPts, gridPt{x, y, f})
		}
	}

	// Sort grid points by fitness descending
	for i := 0; i < len(gridPts)-1; i++ {
		for j := i + 1; j < len(gridPts); j++ {
			if gridPts[j].f > gridPts[i].f {
				gridPts[i], gridPts[j] = gridPts[j], gridPts[i]
			}
		}
	}

	// Update GlobalBest from grid findings
	if len(gridPts) > 0 && gridPts[0].f > st.BestFit {
		st.BestFit = gridPts[0].f
		st.BestX = gridPts[0].x
		st.BestY = gridPts[0].y
	}

	// Find worst particles by fitness
	type idxFit struct {
		idx int
		f   float64
	}
	agents := make([]idxFit, n)
	for i := range ss.Bots {
		agents[i] = idxFit{i, st.Fitness[i]}
	}
	// Sort ascending (worst first)
	for i := 0; i < len(agents)-1; i++ {
		for j := i + 1; j < len(agents); j++ {
			if agents[j].f < agents[i].f {
				agents[i], agents[j] = agents[j], agents[i]
			}
		}
	}

	// Teleport worst particles to best grid points
	inject := eoGridInjectTop
	if inject > len(gridPts) {
		inject = len(gridPts)
	}
	if inject > n {
		inject = n
	}
	for i := 0; i < inject; i++ {
		bi := agents[i].idx
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
		// Update personal best for teleported particle
		f := distanceFitness(&ss.Bots[bi], ss)
		if f > st.PersonalF[bi] {
			st.PersonalF[bi] = f
			st.PersonalX[bi] = ss.Bots[bi].X
			st.PersonalY[bi] = ss.Bots[bi].Y
		}
	}
}

// eoClampArena clamps bot position to arena bounds.
func eoClampArena(bot *SwarmBot, ss *SwarmState) {
	if bot.X < SwarmEdgeMargin {
		bot.X = SwarmEdgeMargin
	}
	if bot.X > ss.ArenaW-SwarmEdgeMargin {
		bot.X = ss.ArenaW - SwarmEdgeMargin
	}
	if bot.Y < SwarmEdgeMargin {
		bot.Y = SwarmEdgeMargin
	}
	if bot.Y > ss.ArenaH-SwarmEdgeMargin {
		bot.Y = ss.ArenaH - SwarmEdgeMargin
	}
}
