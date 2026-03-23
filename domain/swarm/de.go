package swarm

import (
	"math"
	"math/rand"
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
// Parameters:
//   F  (DifferentialWeight) — scaling factor for difference vectors [0.4, 1.0]
//   CR (CrossoverRate)      — probability of inheriting from mutant [0.1, 0.9]
//
// In the swarm context, each bot is a candidate solution. The algorithm
// evaluates fitness using the same landscape as PSO (distance to light or
// center), allowing direct comparison between the two approaches.
//
// Reference: Storn, R. & Price, K. (1997)
//
//	"Differential Evolution — A Simple and Efficient Heuristic for
//	 Global Optimization over Continuous Spaces",
//	 Journal of Global Optimization, 11(4), pp. 341–359.

const (
	deMaxTicks  = 500   // generation cycle length in ticks
	deSteerRate = 0.15  // max steering change per tick (radians)
	deTrialStep = 30.0  // how far a bot moves toward its trial position per tick
)

// DEState holds Differential Evolution state for the swarm.
type DEState struct {
	// Per-bot state
	Fitness  []float64 // current fitness per bot (higher = better)
	TrialX   []float64 // trial (mutant) position X per bot
	TrialY   []float64 // trial (mutant) position Y per bot
	TrialF   []float64 // trial fitness (evaluated when bot arrives)
	Moving   []bool    // true while bot is moving toward trial position
	BestIdx  int       // index of best individual
	BestF    float64   // best fitness found

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
		}
	}
	ss.DEOn = true
}

// ClearDE frees Differential Evolution state.
func ClearDE(ss *SwarmState) {
	ss.DE = nil
	ss.DEOn = false
}

// TickDE runs one tick of the Differential Evolution algorithm.
// At the start of each generation, mutant trial vectors are generated via
// DE/rand/1/bin. Bots then steer toward their trial positions. When a bot
// arrives (or the generation timer expires), selection determines whether
// the trial replaces the current position.
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
	}

	st.GenTick++

	// New generation: create trial vectors.
	if st.GenTick >= deMaxTicks || st.GenTick == 1 {
		if st.GenTick >= deMaxTicks {
			// Selection: evaluate trial fitness and decide replacement.
			for i := range ss.Bots {
				if st.Moving[i] {
					st.TrialF[i] = deFitness(&ss.Bots[i], ss)
					if st.TrialF[i] >= st.Fitness[i] {
						// Trial is better or equal — accept (greedy selection).
						st.Fitness[i] = st.TrialF[i]
					}
					// If trial is worse, the bot already moved, so we don't
					// teleport it back — instead it will get a new trial next gen.
				}
				st.Moving[i] = false
			}
			st.GenTick = 1
		}

		// Update global best.
		for i := range ss.Bots {
			if st.Fitness[i] > st.BestF {
				st.BestF = st.Fitness[i]
				st.BestIdx = i
			}
		}

		// Generate trial vectors via DE/rand/1/bin mutation + crossover.
		for i := 0; i < n; i++ {
			// Pick three distinct random indices r1, r2, r3 ≠ i.
			r1, r2, r3 := dePickThree(ss.Rng, n, i)

			// Mutation: v = x_r1 + F * (x_r2 - x_r3)
			vx := ss.Bots[r1].X + st.DifferentialWeight*(ss.Bots[r2].X-ss.Bots[r3].X)
			vy := ss.Bots[r1].Y + st.DifferentialWeight*(ss.Bots[r2].Y-ss.Bots[r3].Y)

			// Binomial crossover: decide per-dimension whether to use mutant.
			tx, ty := ss.Bots[i].X, ss.Bots[i].Y
			jrand := ss.Rng.Intn(2) // ensure at least one dimension from mutant

			if ss.Rng.Float64() < st.CrossoverRate || jrand == 0 {
				tx = vx
			}
			if ss.Rng.Float64() < st.CrossoverRate || jrand == 1 {
				ty = vy
			}

			// Clamp trial position to arena bounds.
			tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
			ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))

			st.TrialX[i] = tx
			st.TrialY[i] = ty
			st.Moving[i] = true
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

// ApplyDE steers a bot toward its trial position during the current generation.
// Once the bot arrives within deTrialStep distance, it stops and waits for
// the generation cycle to complete selection.
func ApplyDE(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.DE == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.DE
	if idx >= len(st.Moving) {
		bot.Speed = SwarmBotSpeed
		return
	}

	if !st.Moving[idx] {
		// Not moving — maintain slow drift for visual interest.
		bot.Speed = SwarmBotSpeed * 0.3
		// LED: dim blue for idle.
		bot.LEDColor = [3]uint8{60, 60, 120}
		return
	}

	// Steer toward trial position.
	dx := st.TrialX[idx] - bot.X
	dy := st.TrialY[idx] - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist < deTrialStep {
		// Arrived at trial position — mark as arrived.
		st.Moving[idx] = false
		st.TrialF[idx] = deFitness(bot, ss)
		if st.TrialF[idx] >= st.Fitness[idx] {
			st.Fitness[idx] = st.TrialF[idx]
		}
		bot.Speed = SwarmBotSpeed * 0.3
		return
	}

	desired := math.Atan2(dy, dx)
	steerToward(bot, desired, deSteerRate)
	bot.Speed = SwarmBotSpeed

	// LED: color by generation progress — green when moving, brighter = higher fitness.
	fit01 := math.Min(math.Max(st.Fitness[idx]/100.0, 0), 1)
	g := uint8(80 + fit01*175)
	bot.LEDColor = [3]uint8{30, g, 50}
}

// deFitness evaluates the fitness of a bot using the shared Gaussian fitness
// landscape for consistent comparison across algorithms.
func deFitness(bot *SwarmBot, ss *SwarmState) float64 {
	return distanceFitness(bot, ss)
}

// dePickThree selects three distinct random indices from [0, n) that differ
// from exclude. It uses rejection sampling which is efficient when n ≥ 4.
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
