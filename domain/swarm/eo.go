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
	eoPoolSize  = 5      // number of best solutions in equilibrium pool
	eoMaxTicks  = 3000   // full cycle length (matches benchmark length)
	eoSteerRate = 0.30   // max steering per tick (radians)
	eoA1        = 2.0    // exploration control constant
	eoA2        = 1.0    // exploitation control constant
	eoGP        = 0.5    // generation probability
)

// EOState holds Equilibrium Optimizer state for the swarm.
type EOState struct {
	Fitness   []float64   // current fitness per bot
	PersonalX []float64   // personal best X per bot
	PersonalY []float64   // personal best Y per bot
	PersonalF []float64   // personal best fitness per bot
	PoolX     []float64   // equilibrium pool X positions
	PoolY     []float64   // equilibrium pool Y positions
	PoolF     []float64   // equilibrium pool fitness values
	CycleTick int         // ticks into current cycle
	BestIdx   int         // index of overall best bot
	BestFit   float64     // best fitness found
	BestX     float64     // persistent global best X position
	BestY     float64     // persistent global best Y position
	Phase     []int       // per-bot phase: 0=exploration, 1=exploitation
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
		BestFit:   -1e9,
	}

	// Initialize personal bests to current positions
	for i := range ss.Bots {
		st.PersonalX[i] = ss.Bots[i].X
		st.PersonalY[i] = ss.Bots[i].Y
		st.PersonalF[i] = -1e9
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
	}

	st.CycleTick++
	if st.CycleTick > eoMaxTicks {
		st.CycleTick = 1
	}

	// Time ratio t ∈ (0, 1]
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

	// Determine phase per bot based on time ratio
	for i := range ss.Bots {
		if t < 0.5 {
			st.Phase[i] = 0 // exploration
		} else {
			st.Phase[i] = 1 // exploitation
		}
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

	// Select a random equilibrium candidate from the pool
	poolIdx := ss.Rng.Intn(eoPoolSize)
	ceqX := st.PoolX[poolIdx]
	ceqY := st.PoolY[poolIdx]

	// Exponential decay term: F = a1 * sign(r - 0.5) * (exp(-λt) - 1)
	// λ increases over time to narrow search
	r := ss.Rng.Float64()
	lambda := ss.Rng.Float64() * 2.0 // randomized turnover rate
	sign := 1.0
	if r < 0.5 {
		sign = -1.0
	}
	F := eoA1 * sign * (math.Exp(-lambda*t) - 1.0)

	// Generation rate: G = G0 * F with probability eoGP
	G0X, G0Y := 0.0, 0.0
	if ss.Rng.Float64() < eoGP {
		r1 := ss.Rng.Float64()
		r2 := ss.Rng.Float64()
		G0X = (ceqX - bot.X) * r1
		G0Y = (ceqY - bot.Y) * r2
	}
	GX := G0X * F
	GY := G0Y * F

	// Update concentration (position target):
	// C_new = Ceq + (C - Ceq) * F + (G / (lambda * V)) * (1 - F)
	// Simplified: target = Ceq + (bot - Ceq) * F + G * (1 - F)
	targetX := ceqX + (bot.X-ceqX)*F + GX*(1.0-F)
	targetY := ceqY + (bot.Y-ceqY)*F + GY*(1.0-F)

	// Adaptive global-best attraction: weight increases from 5% to 20% over time
	if st.BestFit > -1e8 {
		gbWeight := 0.05 + 0.15*t
		targetX = targetX*(1-gbWeight) + st.BestX*gbWeight
		targetY = targetY*(1-gbWeight) + st.BestY*gbWeight
	}

	// Clamp to arena
	targetX = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, targetX))
	targetY = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, targetY))

	// Move bot directly toward target (Eigenbewegung)
	dx := targetX - bot.X
	dy := targetY - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	maxStep := SwarmBotSpeed * 1.5
	if dist > maxStep {
		// Move at max speed toward target
		ratio := maxStep / dist
		bot.X += dx * ratio
		bot.Y += dy * ratio
	} else {
		// Snap to target when close
		bot.X = targetX
		bot.Y = targetY
	}

	// Update angle to match movement direction
	if dist > 1.0 {
		bot.Angle = math.Atan2(dy, dx)
	}

	// Set Speed=0 to prevent double-move in GUI physics step
	bot.Speed = 0

	// Arena clamping
	eoClampArena(bot, ss)

	// LED visualization
	if idx == st.BestIdx {
		// Best particle: gold
		bot.LEDColor = [3]uint8{255, 200, 50}
	} else if st.Phase[idx] == 0 {
		// Exploration phase: blue-violet gradient
		intensity := uint8(100 + F*50)
		if intensity < 80 {
			intensity = 80
		}
		bot.LEDColor = [3]uint8{intensity, 60, 220}
	} else {
		// Exploitation phase: green-teal gradient
		progress := t
		g := uint8(100 + progress*155)
		bot.LEDColor = [3]uint8{30, g, uint8(80 + progress*80)}
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
