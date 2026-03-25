package swarm

import "math"

// Dragonfly Algorithm (DA): Swarm intelligence metaheuristic inspired by the
// static (feeding) and dynamic (migratory) swarming behaviour of dragonflies.
//
// Each dragonfly adjusts its position using five behavioural vectors:
//   s = separation   — avoid collisions with neighbours
//   a = alignment    — match velocity of neighbours
//   c = cohesion     — fly toward centre of neighbourhood
//   f = food attraction — steer toward the best-known food source
//   e = enemy distraction — flee from the worst-known position
//
// Step vector:  ΔX = (s·ws + a·wa + c·wc + f·wf + e·we) + w·ΔX(t-1)
// Position:     X(t+1) = X(t) + ΔX(t)
//
// Exploration weights (s,a high) linearly transition to exploitation weights
// (f,c high) over the optimisation cycle, mimicking the shift from dynamic
// (migratory) to static (feeding) swarms.
//
// When no neighbours exist, a dragonfly performs a Lévy flight for global
// exploration — the heavy tail produces occasional long jumps that prevent
// stagnation in local optima.
//
// Reference: Mirjalili, S. (2016)
//            "Dragonfly algorithm: a new meta-heuristic optimization technique
//             for solving single-objective, discrete, and multi-objective problems",
//            Neural Computing and Applications.

const (
	daMaxTicks         = 3000 // full optimisation cycle (matches benchmark length)
	daSteerRate        = 0.3  // max steering change per tick (radians)
	daNeighDist        = 80.0 // neighbourhood radius
	daSpeedMult        = 5.0  // algoMovBot speed multiplier (7.5 px/tick)
	daGridRescanRate   = 200  // periodic grid rescan every N ticks
	daGridRescanSize   = 16   // grid resolution (16×16 = 256 samples)
	daGridInjectTop    = 10   // inject top N grid positions into worst bots
	daDirectMaxProb    = 0.70 // max probability of direct-to-best at end
	daDirectStartProg  = 0.20 // progress threshold to start direct-to-best
	daGBWeightMin      = 0.05 // global-best attraction weight at start
	daGBWeightMax      = 0.65 // global-best attraction weight at end
	daBestBotWalkR     = 40.0 // best-bot local random walk radius
)

// DAState holds Dragonfly Algorithm state for the swarm.
type DAState struct {
	Fitness    []float64 // current fitness per bot
	StepX      []float64 // step vector X per bot (velocity carry-over)
	StepY      []float64 // step vector Y per bot
	TargetX    []float64 // precomputed target X per bot
	TargetY    []float64 // precomputed target Y per bot
	IsDirect   []bool    // true if bot is in direct-to-best mode
	BestX      float64   // food source (historical global best position) X
	BestY      float64   // food source (historical global best position) Y
	BestF      float64   // historical global best fitness
	BestIdx    int       // index of historical best bot
	CurBestIdx int       // index of current tick's best bot (for LED)
	WorstX     float64   // enemy (global worst position) X
	WorstY     float64   // enemy (global worst position) Y
	WorstF     float64   // global worst fitness
	WorstIdx   int       // index of worst bot
	Tick       int       // ticks into current cycle
	Role       []int     // 0=static(feeding), 1=dynamic(migratory), 2=levy per bot
}

// InitDA allocates Dragonfly Algorithm state for all bots.
func InitDA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.DA = &DAState{
		Fitness:  make([]float64, n),
		StepX:    make([]float64, n),
		StepY:    make([]float64, n),
		TargetX:  make([]float64, n),
		TargetY:  make([]float64, n),
		IsDirect: make([]bool, n),
		Role:     make([]int, n),
		BestF:    -1e18,
		BestIdx:  -1,
		WorstF:   1e18,
		WorstIdx: -1,
	}
	ss.DAOn = true
}

// ClearDA frees Dragonfly Algorithm state.
func ClearDA(ss *SwarmState) {
	ss.DA = nil
	ss.DAOn = false
}

// TickDA updates the Dragonfly Algorithm for all bots.
func TickDA(ss *SwarmState) {
	if ss.DA == nil {
		return
	}
	st := ss.DA

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		st.Fitness = append(st.Fitness, 0)
		st.StepX = append(st.StepX, 0)
		st.StepY = append(st.StepY, 0)
		st.TargetX = append(st.TargetX, 0)
		st.TargetY = append(st.TargetY, 0)
		st.IsDirect = append(st.IsDirect, false)
		st.Role = append(st.Role, 0)
	}

	st.Tick++
	if st.Tick > daMaxTicks {
		st.Tick = 1
	}

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find current tick's best/worst and update historical global best
	curBestIdx := -1
	curBestF := -1e18
	st.WorstIdx = -1
	st.WorstF = 1e18
	for i := range ss.Bots {
		if st.Fitness[i] > curBestF {
			curBestF = st.Fitness[i]
			curBestIdx = i
		}
		if st.Fitness[i] < st.WorstF {
			st.WorstF = st.Fitness[i]
			st.WorstIdx = i
		}
	}
	// Update historical global best (persistent across ticks)
	if curBestIdx >= 0 && curBestF > st.BestF {
		st.BestF = curBestF
		st.BestIdx = curBestIdx
		st.BestX = ss.Bots[curBestIdx].X
		st.BestY = ss.Bots[curBestIdx].Y
	}
	// Track current tick's best index for gold LED
	st.CurBestIdx = curBestIdx
	if st.WorstIdx >= 0 {
		st.WorstX = ss.Bots[st.WorstIdx].X
		st.WorstY = ss.Bots[st.WorstIdx].Y
	}

	n := len(ss.Bots)

	// Periodic grid rescan — systematic landscape sampling
	if st.Tick > 0 && st.Tick%daGridRescanRate == 0 && n > 0 {
		daGridRescan(ss, st)
	}

	// Precompute targets for all bots
	progress := float64(st.Tick) / float64(daMaxTicks)
	gbWeight := daGBWeightMin + (daGBWeightMax-daGBWeightMin)*progress

	for i := range ss.Bots {
		st.IsDirect[i] = false

		// Best-bot local random walk
		if st.BestIdx >= 0 && i == st.BestIdx {
			angle := ss.Rng.Float64() * 2 * math.Pi
			r := ss.Rng.Float64() * daBestBotWalkR
			tx := st.BestX + r*math.Cos(angle)
			ty := st.BestY + r*math.Sin(angle)
			// Evaluate and update global best
			f := distanceFitnessPt(ss, tx, ty)
			if f > st.BestF {
				st.BestF = f
				st.BestX = tx
				st.BestY = ty
			}
			st.TargetX[i] = tx
			st.TargetY[i] = ty
			st.Role[i] = 0
			continue
		}

		// Direct-to-best: late-phase convergence
		if progress > daDirectStartProg && st.BestIdx >= 0 {
			prob := daDirectMaxProb * (progress - daDirectStartProg) / (1.0 - daDirectStartProg)
			if ss.Rng.Float64() < prob {
				jitter := 7.5
				tx := st.BestX + (ss.Rng.Float64()*2-1)*jitter
				ty := st.BestY + (ss.Rng.Float64()*2-1)*jitter
				f := distanceFitnessPt(ss, tx, ty)
				if f > st.BestF {
					st.BestF = f
					st.BestX = tx
					st.BestY = ty
				}
				st.TargetX[i] = tx
				st.TargetY[i] = ty
				st.IsDirect[i] = true
				st.Role[i] = 0
				continue
			}
		}

		// Standard DA step vector computation
		t := progress
		ws := 2.0 * (1.0 - t)
		wa := 2.0 * (1.0 - t)
		wc := 2.0 * t
		wf := 2.0 * t
		we := 1.0 - t
		w := 0.9 - 0.5*t

		candidates := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, daNeighDist)
		sepX, sepY := 0.0, 0.0
		aliX, aliY := 0.0, 0.0
		cohX, cohY := 0.0, 0.0
		neighCount := 0
		for _, j := range candidates {
			if j == i {
				continue
			}
			dx := ss.Bots[j].X - ss.Bots[i].X
			dy := ss.Bots[j].Y - ss.Bots[i].Y
			d := math.Sqrt(dx*dx + dy*dy)
			if d > daNeighDist || d < 0.001 {
				continue
			}
			neighCount++
			sepX -= dx / d
			sepY -= dy / d
			aliX += math.Cos(ss.Bots[j].Angle)
			aliY += math.Sin(ss.Bots[j].Angle)
			cohX += dx
			cohY += dy
		}

		if neighCount > 0 {
			nn := float64(neighCount)
			sepX /= nn
			sepY /= nn
			aliX /= nn
			aliY /= nn
			cohX /= nn
			cohY /= nn

			foodX := st.BestX - ss.Bots[i].X
			foodY := st.BestY - ss.Bots[i].Y
			foodD := math.Sqrt(foodX*foodX + foodY*foodY)
			if foodD > 0 {
				foodX /= foodD
				foodY /= foodD
			}

			enemyX := ss.Bots[i].X - st.WorstX
			enemyY := ss.Bots[i].Y - st.WorstY
			enemyD := math.Sqrt(enemyX*enemyX + enemyY*enemyY)
			if enemyD > 0 {
				enemyX /= enemyD
				enemyY /= enemyD
			}

			st.StepX[i] = w*st.StepX[i] + ws*sepX + wa*aliX + wc*cohX + wf*foodX + we*enemyX
			st.StepY[i] = w*st.StepY[i] + ws*sepY + wa*aliY + wc*cohY + wf*foodY + we*enemyY

			if t < 0.5 {
				st.Role[i] = 1
			} else {
				st.Role[i] = 0
			}
		} else {
			st.Role[i] = 2
			step := MantegnaLevy(ss.Rng, 1.5)
			levyAngle := ss.Rng.Float64() * 2 * math.Pi
			st.StepX[i] = step * math.Cos(levyAngle) * 3.0
			st.StepY[i] = step * math.Sin(levyAngle) * 3.0
		}

		// Compute target from step vector (scale up for speed)
		mag := math.Sqrt(st.StepX[i]*st.StepX[i] + st.StepY[i]*st.StepY[i])
		maxStep := SwarmBotSpeed * daSpeedMult
		if mag > 0.01 {
			scale := maxStep / mag
			if scale > maxStep {
				scale = maxStep
			}
			st.TargetX[i] = ss.Bots[i].X + st.StepX[i]*scale
			st.TargetY[i] = ss.Bots[i].Y + st.StepY[i]*scale
		} else {
			st.TargetX[i] = ss.Bots[i].X
			st.TargetY[i] = ss.Bots[i].Y
		}

		// Global-Best attraction: blend target toward BestX/BestY
		if st.BestIdx >= 0 {
			st.TargetX[i] = st.TargetX[i]*(1-gbWeight) + st.BestX*gbWeight
			st.TargetY[i] = st.TargetY[i]*(1-gbWeight) + st.BestY*gbWeight
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].DAFitness = fitToSensor(st.Fitness[i])
		if ss.Bots[i].DAFitness > 100 {
			ss.Bots[i].DAFitness = 100
		}
		ss.Bots[i].DARole = st.Role[i]
		if st.BestIdx >= 0 {
			dx := st.BestX - ss.Bots[i].X
			dy := st.BestY - ss.Bots[i].Y
			ss.Bots[i].DAFoodDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].DAFoodDist = 9999
		}
	}
}

// ApplyDA steers a bot according to the Dragonfly Algorithm using precomputed targets.
func ApplyDA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.DA == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.DA
	if idx >= len(st.TargetX) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Move toward precomputed target via algoMovBot
	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, daSpeedMult)

	// LED colour by role and direct-to-best status
	if st.IsDirect[idx] {
		bot.LEDColor = [3]uint8{0, 255, 0} // green for direct-to-best
	} else if idx == st.CurBestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for food source
	} else {
		t := float64(st.Tick) / float64(daMaxTicks)
		switch st.Role[idx] {
		case 0: // static/feeding — green
			intensity := uint8(100 + t*155)
			bot.LEDColor = [3]uint8{0, intensity, 50}
		case 1: // dynamic/migratory — blue
			intensity := uint8(100 + (1-t)*155)
			bot.LEDColor = [3]uint8{50, 50, intensity}
		case 2: // lévy flight — magenta
			bot.LEDColor = [3]uint8{200, 0, 200}
		}
	}
}

// daGridRescan evaluates a grid of points across the arena and teleports
// the worst dragonflies to the best-discovered grid positions.
func daGridRescan(ss *SwarmState, st *DAState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	type gridPt struct {
		x, y, f float64
	}
	gridPts := make([]gridPt, 0, daGridRescanSize*daGridRescanSize)
	for gx := 0; gx < daGridRescanSize; gx++ {
		for gy := 0; gy < daGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(daGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(daGridRescanSize)
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

	// Update global best from grid findings
	if len(gridPts) > 0 && gridPts[0].f > st.BestF {
		st.BestF = gridPts[0].f
		st.BestX = gridPts[0].x
		st.BestY = gridPts[0].y
	}

	// Find worst bots by fitness
	type idxFit struct {
		idx int
		f   float64
	}
	agents := make([]idxFit, n)
	for i := range ss.Bots {
		agents[i] = idxFit{i, st.Fitness[i]}
	}
	for i := 0; i < len(agents)-1; i++ {
		for j := i + 1; j < len(agents); j++ {
			if agents[j].f < agents[i].f {
				agents[i], agents[j] = agents[j], agents[i]
			}
		}
	}

	// Teleport worst bots to best grid points
	inject := daGridInjectTop
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
		// Reset step vectors for teleported bots
		st.StepX[bi] = 0
		st.StepY[bi] = 0
	}
}
