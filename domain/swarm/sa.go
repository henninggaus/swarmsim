package swarm

import (
	"math"
)

// Simulated Annealing (SA): A probabilistic single-solution metaheuristic
// inspired by the annealing process in metallurgy. The system starts at high
// "temperature" (exploration) and gradually cools (exploitation). At high
// temperatures, worse solutions are accepted with high probability, enabling
// escape from local optima. As the temperature decreases, acceptance of worse
// solutions becomes increasingly rare, converging on the global optimum.
//
// In the swarm context, each bot independently runs its own SA instance:
//   1. Generate a neighbor solution by random perturbation of current position.
//   2. Evaluate fitness at the new position.
//   3. If better, always accept. If worse, accept with probability
//      P = exp(-deltaE / T), where deltaE = f_current - f_new and T = temperature.
//   4. Cool the temperature: T_new = alpha * T_old (geometric cooling).
//   5. After reaching minimum temperature, reheat (restart) to escape stagnation.
//
// Parameters:
//   T0    (InitialTemp)    — starting temperature (default 100.0)
//   Tmin  (MinTemp)        — minimum temperature before reheat (default 0.1)
//   alpha (CoolingRate)    — cooling factor per tick (default 0.995)
//   step  (PerturbRadius)  — max random step size for neighbor generation (default 60.0)
//
// Reference: Kirkpatrick, S., Gelatt, C. D., & Vecchi, M. P. (1983)
//            "Optimization by Simulated Annealing",
//            Science, 220(4598), pp. 671–680.

const (
	saSteerRate = 0.2 // max steering change per tick (radians)
)

// SAState holds Simulated Annealing state for the swarm.
type SAState struct {
	// Per-bot state
	Fitness   []float64 // current fitness per bot
	BestX     []float64 // personal best position X
	BestY     []float64 // personal best position Y
	BestF     []float64 // personal best fitness
	TargetX   []float64 // current perturbation target X
	TargetY   []float64 // current perturbation target Y
	Moving    []bool    // true while bot moves toward target
	Temp      []float64 // current temperature per bot
	Accepted  []bool    // whether last move was accepted

	// Global tracking
	GlobalBestIdx int     // index of globally best bot
	GlobalBestF   float64 // best fitness found across all bots

	// Parameters
	InitialTemp   float64 // T0 (default 100.0)
	MinTemp       float64 // Tmin (default 0.1)
	CoolingRate   float64 // alpha (default 0.995)
	PerturbRadius float64 // neighbor step size (default 60.0)
}

// InitSA allocates Simulated Annealing state for all bots.
func InitSA(ss *SwarmState) {
	n := len(ss.Bots)
	st := &SAState{
		Fitness:       make([]float64, n),
		BestX:         make([]float64, n),
		BestY:         make([]float64, n),
		BestF:         make([]float64, n),
		TargetX:       make([]float64, n),
		TargetY:       make([]float64, n),
		Moving:        make([]bool, n),
		Temp:          make([]float64, n),
		Accepted:      make([]bool, n),
		GlobalBestF:   -1e9,
		InitialTemp:   100.0,
		MinTemp:       0.1,
		CoolingRate:   0.995,
		PerturbRadius: 60.0,
	}

	for i := range ss.Bots {
		st.Fitness[i] = saFitness(&ss.Bots[i], ss)
		st.BestX[i] = ss.Bots[i].X
		st.BestY[i] = ss.Bots[i].Y
		st.BestF[i] = st.Fitness[i]
		st.Temp[i] = st.InitialTemp
		if st.Fitness[i] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[i]
			st.GlobalBestIdx = i
		}
	}

	ss.SA = st
	ss.SAOn = true
}

// ClearSA frees Simulated Annealing state.
func ClearSA(ss *SwarmState) {
	ss.SA = nil
	ss.SAOn = false
}

// TickSA runs one tick of the Simulated Annealing algorithm.
// Each bot that is not currently moving toward a target generates a new
// neighbor solution, evaluates it, and decides whether to accept.
func TickSA(ss *SwarmState) {
	if ss.SA == nil {
		return
	}
	st := ss.SA
	n := len(ss.Bots)

	// Grow slices if bots were added.
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.BestX = append(st.BestX, 0)
		st.BestY = append(st.BestY, 0)
		st.BestF = append(st.BestF, 0)
		st.TargetX = append(st.TargetX, 0)
		st.TargetY = append(st.TargetY, 0)
		st.Moving = append(st.Moving, false)
		st.Temp = append(st.Temp, st.InitialTemp)
		st.Accepted = append(st.Accepted, false)
	}

	for i := 0; i < n; i++ {
		if st.Moving[i] {
			continue // bot is still moving toward its target
		}

		// Evaluate fitness at current position.
		currentF := saFitness(&ss.Bots[i], ss)
		st.Fitness[i] = currentF

		// Update personal best.
		if currentF > st.BestF[i] {
			st.BestF[i] = currentF
			st.BestX[i] = ss.Bots[i].X
			st.BestY[i] = ss.Bots[i].Y
		}

		// Update global best.
		if currentF > st.GlobalBestF {
			st.GlobalBestF = currentF
			st.GlobalBestIdx = i
		}

		// Generate neighbor: random perturbation scaled by temperature ratio.
		// At high temperature, perturbations are larger (exploration).
		tempRatio := st.Temp[i] / st.InitialTemp
		radius := st.PerturbRadius * (0.3 + 0.7*tempRatio) // range [30%..100%] of PerturbRadius
		angle := ss.Rng.Float64() * 2 * math.Pi
		dist := ss.Rng.Float64() * radius

		nx := ss.Bots[i].X + dist*math.Cos(angle)
		ny := ss.Bots[i].Y + dist*math.Sin(angle)

		// Clamp to arena bounds.
		nx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, nx))
		ny = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ny))

		// Pre-evaluate neighbor fitness to decide acceptance.
		neighborF := saFitnessAt(nx, ny, ss)
		deltaE := currentF - neighborF // positive = neighbor is worse

		accepted := false
		if neighborF >= currentF {
			// Better or equal — always accept.
			accepted = true
		} else if st.Temp[i] > st.MinTemp {
			// Worse — accept with Boltzmann probability.
			p := math.Exp(-deltaE / st.Temp[i])
			if ss.Rng.Float64() < p {
				accepted = true
			}
		}

		st.Accepted[i] = accepted
		if accepted {
			st.TargetX[i] = nx
			st.TargetY[i] = ny
			st.Moving[i] = true
		}

		// Cool the temperature.
		st.Temp[i] *= st.CoolingRate
		if st.Temp[i] < st.MinTemp {
			// Reheat: restart temperature to escape local optima.
			st.Temp[i] = st.InitialTemp * 0.5 // reheat to 50% of initial
		}
	}

	// Update sensor cache for SwarmScript.
	for i := range ss.Bots {
		if i >= len(st.Fitness) {
			break
		}
		ss.Bots[i].SAFitness = int(st.Fitness[i] * 100)
		ss.Bots[i].SATemp = int(st.Temp[i])

		if st.GlobalBestIdx >= 0 && st.GlobalBestIdx < n {
			dx := ss.Bots[st.GlobalBestIdx].X - ss.Bots[i].X
			dy := ss.Bots[st.GlobalBestIdx].Y - ss.Bots[i].Y
			ss.Bots[i].SABestDist = int(math.Sqrt(dx*dx + dy*dy))
		}
	}
}

// ApplySA steers a bot toward its accepted target position.
func ApplySA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.SA == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.SA
	if idx >= len(st.Moving) {
		bot.Speed = SwarmBotSpeed
		return
	}

	if !st.Moving[idx] {
		// Not moving — slow drift.
		bot.Speed = SwarmBotSpeed * 0.2
		// LED: color by temperature (hot=red, cold=blue).
		tempRatio := st.Temp[idx] / st.InitialTemp
		r := uint8(math.Min(255, tempRatio*255))
		b := uint8(math.Min(255, (1-tempRatio)*255))
		bot.LEDColor = [3]uint8{r, 40, b}
		return
	}

	dx := st.TargetX[idx] - bot.X
	dy := st.TargetY[idx] - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist < 5.0 {
		// Arrived at target.
		st.Moving[idx] = false
		st.Fitness[idx] = saFitness(bot, ss)
		if st.Fitness[idx] > st.BestF[idx] {
			st.BestF[idx] = st.Fitness[idx]
			st.BestX[idx] = bot.X
			st.BestY[idx] = bot.Y
		}
		if st.Fitness[idx] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[idx]
			st.GlobalBestIdx = idx
		}
		bot.Speed = SwarmBotSpeed * 0.2
		return
	}

	desired := math.Atan2(dy, dx)
	steerToward(bot, desired, saSteerRate)
	bot.Speed = SwarmBotSpeed * 1.2

	// LED: temperature gradient with acceptance flash.
	tempRatio := st.Temp[idx] / st.InitialTemp
	if st.Accepted[idx] {
		// Green flash for accepted moves.
		g := uint8(120 + tempRatio*135)
		bot.LEDColor = [3]uint8{30, g, 50}
	} else {
		// Red-blue gradient by temperature.
		r := uint8(math.Min(255, tempRatio*255))
		b := uint8(math.Min(255, (1-tempRatio)*255))
		bot.LEDColor = [3]uint8{r, 40, b}
	}

	// Mark best bot with gold LED.
	if idx == st.GlobalBestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0}
	}
}

// saFitness evaluates fitness at a bot's current position using the shared
// Gaussian fitness landscape.
func saFitness(bot *SwarmBot, ss *SwarmState) float64 {
	return saFitnessAt(bot.X, bot.Y, ss)
}

// saFitnessAt evaluates fitness at an arbitrary position using the shared
// Gaussian fitness landscape.
func saFitnessAt(x, y float64, ss *SwarmState) float64 {
	if ss.SwarmAlgo != nil && len(ss.SwarmAlgo.FitPeakX) > 0 {
		return EvaluateFitnessLandscape(ss.SwarmAlgo, x, y)
	}
	// Fallback
	targetX, targetY := ss.ArenaW/2, ss.ArenaH/2
	if ss.Light.Active {
		targetX = ss.Light.X
		targetY = ss.Light.Y
	}
	dx := x - targetX
	dy := y - targetY
	dist := math.Sqrt(dx*dx + dy*dy)
	return 100 - dist*0.2
}
