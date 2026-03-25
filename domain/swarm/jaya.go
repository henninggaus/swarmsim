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
	jayaMaxTicks       = 3000 // full optimization cycle (matches benchmark length)
	jayaSteerRate      = 0.25 // max steering change per tick (radians)
	jayaSpeedMult      = 5.0  // movement speed multiplier (7.5 px/tick)
	jayaGBWeightMin    = 0.05 // initial global-best attraction weight
	jayaGBWeightMax    = 0.40 // final global-best attraction weight
	jayaBestWalkRadius = 40.0 // random walk radius for best bot
	jayaGridInterval   = 400  // ticks between systematic grid re-scans
	jayaGridSide       = 12   // grid side for periodic re-scan (12x12=144 candidates)
	jayaGridInject     = 10   // number of grid points to teleport bots to
)

// JayaState holds Jaya Algorithm state for the swarm.
type JayaState struct {
	Fitness       []float64 // current fitness per bot
	PersonalBestX []float64 // personal best position X
	PersonalBestY []float64 // personal best position Y
	PersonalBestF []float64 // personal best fitness
	BestX         float64   // current tick best position X
	BestY         float64   // current tick best position Y
	BestF         float64   // current tick best fitness
	BestIdx       int       // index of best bot this tick
	WorstX        float64   // worst position X
	WorstY        float64   // worst position Y
	WorstF        float64   // worst fitness
	WorstIdx      int       // index of worst bot
	GlobalBestF   float64   // persistent global best fitness (never resets)
	GlobalBestX   float64   // persistent global best position X
	GlobalBestY   float64   // persistent global best position Y
	GlobalBestIdx int       // index of bot that found global best
	Tick          int       // ticks into current cycle
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
		GlobalBestF:   -1e18,
		GlobalBestIdx: -1,
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

	// Find best and worst individuals this tick
	tickBestF := -1e18
	tickBestIdx := -1
	st.WorstIdx = -1
	st.WorstF = 1e18
	for i := 0; i < n; i++ {
		if st.Fitness[i] > tickBestF {
			tickBestF = st.Fitness[i]
			tickBestIdx = i
		}
		if st.Fitness[i] < st.WorstF {
			st.WorstF = st.Fitness[i]
			st.WorstIdx = i
		}
	}
	st.BestF = tickBestF
	st.BestIdx = tickBestIdx
	if tickBestIdx >= 0 {
		st.BestX = ss.Bots[tickBestIdx].X
		st.BestY = ss.Bots[tickBestIdx].Y
	}
	if st.WorstIdx >= 0 {
		st.WorstX = ss.Bots[st.WorstIdx].X
		st.WorstY = ss.Bots[st.WorstIdx].Y
	}

	// Update persistent global best
	if tickBestF > st.GlobalBestF && tickBestIdx >= 0 {
		st.GlobalBestF = tickBestF
		st.GlobalBestX = ss.Bots[tickBestIdx].X
		st.GlobalBestY = ss.Bots[tickBestIdx].Y
		st.GlobalBestIdx = tickBestIdx
	}

	// Periodic grid rescan: systematically scan the arena to find regions
	// that random exploration missed. Critical for deceptive landscapes
	// like Schwefel where the global optimum is far from local optima.
	if st.Tick > 1 && st.Tick%jayaGridInterval == 0 && n > 0 {
		jayaGridRescan(ss, st)
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].JayaFitness = fitToSensor(st.Fitness[i])
		if ss.Bots[i].JayaFitness > 100 {
			ss.Bots[i].JayaFitness = 100
		}
		gbx, gby := st.GlobalBestX, st.GlobalBestY
		if st.GlobalBestIdx < 0 {
			gbx, gby = st.BestX, st.BestY
		}
		dx := gbx - ss.Bots[i].X
		dy := gby - ss.Bots[i].Y
		ss.Bots[i].JayaBestDist = int(math.Sqrt(dx*dx + dy*dy))

		if st.WorstIdx >= 0 {
			dx2 := st.WorstX - ss.Bots[i].X
			dy2 := st.WorstY - ss.Bots[i].Y
			ss.Bots[i].JayaWorstDist = int(math.Sqrt(dx2*dx2 + dy2*dy2))
		} else {
			ss.Bots[i].JayaWorstDist = 9999
		}
	}
}

// jayaGridRescan evaluates a grid of points across the arena and teleports
// the worst bots to the best-discovered grid positions.
func jayaGridRescan(ss *SwarmState, st *JayaState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	if usableW < 10 || usableH < 10 {
		return
	}

	type gridPt struct {
		x, y, f float64
	}
	gridPts := make([]gridPt, 0, jayaGridSide*jayaGridSide)
	for gx := 0; gx < jayaGridSide; gx++ {
		for gy := 0; gy < jayaGridSide; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(jayaGridSide)
			y := margin + usableH*(float64(gy)+0.5)/float64(jayaGridSide)
			// Small jitter
			x += (ss.Rng.Float64()*2.0 - 1.0) * usableW * 0.02
			y += (ss.Rng.Float64()*2.0 - 1.0) * usableH * 0.02
			x = math.Max(margin, math.Min(ss.ArenaW-margin, x))
			y = math.Max(margin, math.Min(ss.ArenaH-margin, y))
			f := distanceFitnessPt(ss, x, y)
			gridPts = append(gridPts, gridPt{x, y, f})

			// Update global best from grid scan
			if f > st.GlobalBestF {
				st.GlobalBestF = f
				st.GlobalBestX = x
				st.GlobalBestY = y
			}
		}
	}

	// Sort grid points by fitness (best first) — simple selection of top N
	n := len(ss.Bots)
	injCount := jayaGridInject
	if injCount > n/2 {
		injCount = n / 2
	}
	if injCount < 1 {
		injCount = 1
	}

	// Find the top injCount grid points
	for k := 0; k < injCount; k++ {
		bestGIdx := k
		for j := k + 1; j < len(gridPts); j++ {
			if gridPts[j].f > gridPts[bestGIdx].f {
				bestGIdx = j
			}
		}
		gridPts[k], gridPts[bestGIdx] = gridPts[bestGIdx], gridPts[k]
	}

	// Find the worst bots and teleport them to the best grid positions
	worstBots := make([]int, 0, injCount)
	used := make(map[int]bool)
	for k := 0; k < injCount; k++ {
		worstIdx := -1
		worstFit := 1e18
		for i := 0; i < n; i++ {
			if !used[i] && i != st.GlobalBestIdx && st.Fitness[i] < worstFit {
				worstFit = st.Fitness[i]
				worstIdx = i
			}
		}
		if worstIdx >= 0 {
			worstBots = append(worstBots, worstIdx)
			used[worstIdx] = true
		}
	}

	for k, bi := range worstBots {
		if k < len(gridPts) {
			ss.Bots[bi].X = gridPts[k].x
			ss.Bots[bi].Y = gridPts[k].y
			st.Fitness[bi] = gridPts[k].f
			st.PersonalBestX[bi] = gridPts[k].x
			st.PersonalBestY[bi] = gridPts[k].y
			st.PersonalBestF[bi] = gridPts[k].f
		}
	}
}

// ApplyJaya steers a bot according to the Jaya algorithm.
func ApplyJaya(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Jaya == nil {
		bot.Speed = 0
		return
	}
	st := ss.Jaya
	if idx >= len(st.Fitness) {
		bot.Speed = 0
		return
	}

	// Determine global best position to use
	gbx, gby := st.GlobalBestX, st.GlobalBestY
	if st.GlobalBestIdx < 0 {
		gbx, gby = st.BestX, st.BestY
	}

	// Best bot does random walk to explore local neighborhood
	if idx == st.BestIdx || idx == st.GlobalBestIdx {
		rx := gbx + (ss.Rng.Float64()-0.5)*2*jayaBestWalkRadius
		ry := gby + (ss.Rng.Float64()-0.5)*2*jayaBestWalkRadius
		algoMovBot(bot, rx, ry, ss.ArenaW, ss.ArenaH, jayaSpeedMult)
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for best
		return
	}

	if st.BestIdx < 0 || st.WorstIdx < 0 {
		bot.Speed = 0
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

	// Adaptive global-best attraction: increases over time
	progress := float64(st.Tick) / float64(jayaMaxTicks)
	if progress > 1 {
		progress = 1
	}
	gbWeight := jayaGBWeightMin + (jayaGBWeightMax-jayaGBWeightMin)*progress
	targetX = targetX*(1-gbWeight) + gbx*gbWeight
	targetY = targetY*(1-gbWeight) + gby*gbWeight

	// Move directly toward target
	algoMovBot(bot, targetX, targetY, ss.ArenaW, ss.ArenaH, jayaSpeedMult)

	// LED color: brightness proportional to fitness
	intensity := uint8(80 + st.Fitness[idx]*175)
	if intensity < 80 {
		intensity = 80
	}
	fitnessRange := st.BestF - st.WorstF
	var ratio float64
	if fitnessRange > 1e-10 {
		ratio = (st.Fitness[idx] - st.WorstF) / fitnessRange
	} else {
		ratio = 0.5
	}
	r := uint8(float64(intensity) * (1 - ratio))
	g := uint8(float64(intensity) * ratio)
	bot.LEDColor = [3]uint8{r, g, intensity / 4}
}
