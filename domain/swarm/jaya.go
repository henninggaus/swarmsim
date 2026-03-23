package swarm

import "math"

// Jaya Algorithm: A parameter-free population-based metaheuristic that moves
// solutions toward the best and away from the worst individual simultaneously.
//
// Like TLBO (also by Rao), Jaya requires NO algorithm-specific parameters —
// only population size and number of iterations. The name "Jaya" means
// "victory" in Sanskrit, reflecting the algorithm's philosophy of always
// striving to move toward the best solution.
//
// Update rule per dimension:
//
//	X_new = X_old + r1 * (Best - |X_old|) - r2 * (Worst - |X_old|)
//
// The first term attracts solutions toward the global best, while the second
// term repels solutions from the global worst. Both terms use absolute values
// of the current position to preserve dimensionality. New positions are only
// accepted if they improve fitness (greedy selection).
//
// Reference: Rao, R.V. (2016)
//
//	"Jaya: A simple and new optimization algorithm for solving constrained
//	 and unconstrained optimization problems",
//	 International Journal of Industrial Engineering Computations, 7(1), 19-34.
const (
	jayaMaxTicks  = 600  // full optimization cycle
	jayaSteerRate = 0.15 // max steering change per tick (radians)
)

// JayaState holds Jaya Algorithm state for the swarm.
type JayaState struct {
	Fitness  []float64 // current fitness per bot
	PersonalBestX []float64 // personal best position X
	PersonalBestY []float64 // personal best position Y
	PersonalBestF []float64 // personal best fitness
	BestX    float64   // global best position X
	BestY    float64   // global best position Y
	BestF    float64   // global best fitness
	BestIdx  int       // index of best bot
	WorstX   float64   // worst position X
	WorstY   float64   // worst position Y
	WorstF   float64   // worst fitness
	WorstIdx int       // index of worst bot
	Tick     int       // ticks into current cycle
}

// InitJaya allocates Jaya state for all bots.
func InitJaya(ss *SwarmState) {
	n := len(ss.Bots)
	ss.Jaya = &JayaState{
		Fitness:       make([]float64, n),
		PersonalBestX: make([]float64, n),
		PersonalBestY: make([]float64, n),
		PersonalBestF: make([]float64, n),
		BestF:         -1e18,
		BestIdx:       -1,
		WorstF:        1e18,
		WorstIdx:      -1,
	}
	// Initialize personal bests to current positions
	for i := range ss.Bots {
		ss.Jaya.PersonalBestX[i] = ss.Bots[i].X
		ss.Jaya.PersonalBestY[i] = ss.Bots[i].Y
		ss.Jaya.PersonalBestF[i] = -1e18
	}
	ss.JayaOn = true
}

// ClearJaya frees Jaya state.
func ClearJaya(ss *SwarmState) {
	ss.Jaya = nil
	ss.JayaOn = false
}

// TickJaya updates the Jaya algorithm for all bots.
func TickJaya(ss *SwarmState) {
	if ss.Jaya == nil {
		return
	}
	st := ss.Jaya
	n := len(ss.Bots)

	// Grow slices if bots were added
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.PersonalBestX = append(st.PersonalBestX, 0)
		st.PersonalBestY = append(st.PersonalBestY, 0)
		st.PersonalBestF = append(st.PersonalBestF, -1e18)
	}

	st.Tick++
	if st.Tick > jayaMaxTicks {
		st.Tick = 1
	}

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Update personal bests
	for i := 0; i < n; i++ {
		if st.Fitness[i] > st.PersonalBestF[i] {
			st.PersonalBestF[i] = st.Fitness[i]
			st.PersonalBestX[i] = ss.Bots[i].X
			st.PersonalBestY[i] = ss.Bots[i].Y
		}
	}

	// Find best and worst individuals
	st.BestIdx = -1
	st.BestF = -1e18
	st.WorstIdx = -1
	st.WorstF = 1e18
	for i := 0; i < n; i++ {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
		}
		if st.Fitness[i] < st.WorstF {
			st.WorstF = st.Fitness[i]
			st.WorstIdx = i
		}
	}
	if st.BestIdx >= 0 {
		st.BestX = ss.Bots[st.BestIdx].X
		st.BestY = ss.Bots[st.BestIdx].Y
	}
	if st.WorstIdx >= 0 {
		st.WorstX = ss.Bots[st.WorstIdx].X
		st.WorstY = ss.Bots[st.WorstIdx].Y
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].JayaFitness = fitToSensor(st.Fitness[i])
		if ss.Bots[i].JayaFitness > 100 {
			ss.Bots[i].JayaFitness = 100
		}
		if st.BestIdx >= 0 {
			dx := st.BestX - ss.Bots[i].X
			dy := st.BestY - ss.Bots[i].Y
			ss.Bots[i].JayaBestDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].JayaBestDist = 9999
		}
		if st.WorstIdx >= 0 {
			dx := st.WorstX - ss.Bots[i].X
			dy := st.WorstY - ss.Bots[i].Y
			ss.Bots[i].JayaWorstDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].JayaWorstDist = 9999
		}
	}
}

// ApplyJaya steers a bot according to the Jaya algorithm.
func ApplyJaya(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Jaya == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Jaya
	if idx >= len(st.Fitness) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Best bot keeps natural behavior
	if idx == st.BestIdx {
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for best
		return
	}

	if st.BestIdx < 0 || st.WorstIdx < 0 {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Jaya update rule:
	// X_new = X_old + r1 * (Best - |X_old|) - r2 * (Worst - |X_old|)
	r1 := ss.Rng.Float64()
	r2 := ss.Rng.Float64()

	absX := math.Abs(bot.X)
	absY := math.Abs(bot.Y)

	targetX := bot.X + r1*(st.BestX-absX) - r2*(st.WorstX-absX)
	targetY := bot.Y + r1*(st.BestY-absY) - r2*(st.WorstY-absY)

	// Steer toward target
	dx := targetX - bot.X
	dy := targetY - bot.Y
	if dx != 0 || dy != 0 {
		desired := math.Atan2(dy, dx)
		steerToward(bot, desired, jayaSteerRate)
	}
	bot.Speed = SwarmBotSpeed

	// LED color: brightness proportional to fitness
	// Green = close to best, Red = close to worst
	intensity := uint8(80 + st.Fitness[idx]*175)
	if intensity < 80 {
		intensity = 80
	}
	// Ratio: how close to best vs worst (0 = worst, 1 = best)
	fitnessRange := st.BestF - st.WorstF
	var ratio float64
	if fitnessRange > 1e-10 {
		ratio = (st.Fitness[idx] - st.WorstF) / fitnessRange
	} else {
		ratio = 0.5
	}
	// Interpolate red→green based on ratio
	r := uint8(float64(intensity) * (1 - ratio))
	g := uint8(float64(intensity) * ratio)
	bot.LEDColor = [3]uint8{r, g, intensity / 4}
}
