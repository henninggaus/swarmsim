package swarm

import "math"

// Whale Optimization Algorithm (WOA): Inspired by the bubble-net hunting
// strategy of humpback whales. Whales create shrinking circles of bubbles
// around prey, then spiral inward to attack.
//
// Three behaviors:
// 1. Encircling prey — move toward the best known position
// 2. Bubble-net attack — logarithmic spiral around prey
// 3. Search for prey — random exploration when |A| >= 1
//
// Reference: Mirjalili, S. & Lewis, A. (2016)
//            "The Whale Optimization Algorithm", Advances in Engineering Software.

const (
	woaRadius         = 90.0  // neighbor detection radius
	woaMaxTicks       = 3000  // full benchmark length
	woaSteerRate      = 0.25  // max steering change per tick (radians)
	woaSpiralB        = 1.0   // spiral shape constant
	woaSpiralProb     = 0.5   // probability of spiral vs shrink
	woaSpeedMult      = 5.0   // movement speed multiplier (7.5 px/tick)
	woaGridRescanRate = 200   // periodic grid rescan every N ticks
	woaGridRescanSize = 16    // grid resolution (16×16 = 256 samples)
	woaGridInjectTop  = AlgoGridInjectTop // inject top N grid positions into worst bots
	woaDirectMaxProb  = 0.70  // max probability of direct-to-best at end
	woaDirectStartProg = 0.20 // progress threshold to start direct-to-best
	woaGBWeightMin    = 0.05  // global-best attraction weight at start
	woaGBWeightMax    = 0.65  // global-best attraction weight at end
	woaBestBotWalkR   = 40.0  // best-bot local random walk radius
)

// WOAState holds Whale Optimization Algorithm state.
type WOAState struct {
	Fitness     []float64 // current fitness per bot
	BestIdx     int       // index of current tick's best whale
	BestX       float64   // current tick best position
	BestY       float64
	BestF       float64   // current tick best fitness
	HuntTick    int       // current tick in hunt cycle
	Phase       []int     // per-bot: 0=encircle, 1=spiral, 2=search
	TargetX     []float64 // precomputed target X per bot
	TargetY     []float64 // precomputed target Y per bot
	IsDirect    []bool    // per-bot: true if direct-to-best
	// Persistent global best (never reset)
	GlobalBestF float64
	GlobalBestX float64
	GlobalBestY float64
}

// InitWOA allocates Whale Optimization state for all bots.
func InitWOA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.WOA = &WOAState{
		Fitness:     make([]float64, n),
		Phase:       make([]int, n),
		TargetX:     make([]float64, n),
		TargetY:     make([]float64, n),
		IsDirect:    make([]bool, n),
		BestIdx:     -1,
		BestF:       -1e18,
		GlobalBestF: -1e18,
	}
	ss.WOAOn = true
}

// ClearWOA frees Whale Optimization state.
func ClearWOA(ss *SwarmState) {
	ss.WOA = nil
	ss.WOAOn = false
}

// TickWOA updates the Whale Optimization Algorithm for all bots.
func TickWOA(ss *SwarmState) {
	if ss.WOA == nil {
		return
	}
	st := ss.WOA

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		st.Fitness = append(st.Fitness, 0)
		st.Phase = append(st.Phase, 0)
		st.TargetX = append(st.TargetX, 0)
		st.TargetY = append(st.TargetY, 0)
		st.IsDirect = append(st.IsDirect, false)
	}

	st.HuntTick++
	if st.HuntTick > woaMaxTicks {
		st.HuntTick = 1
	}

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find best whale (current tick) and update persistent global best
	st.BestIdx = 0
	st.BestF = st.Fitness[0]
	for i := 1; i < len(ss.Bots); i++ {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
		}
	}
	st.BestX = ss.Bots[st.BestIdx].X
	st.BestY = ss.Bots[st.BestIdx].Y
	// Update persistent global best
	if st.BestF > st.GlobalBestF {
		st.GlobalBestF = st.BestF
		st.GlobalBestX = st.BestX
		st.GlobalBestY = st.BestY
	}

	n := len(ss.Bots)

	// Periodic grid rescan — systematic landscape sampling
	if st.HuntTick > 0 && st.HuntTick%woaGridRescanRate == 0 && n > 0 {
		woaGridRescan(ss, st)
	}

	// Linearly decreasing a: 2 → 0
	a := 2.0 * (1.0 - float64(st.HuntTick)/float64(woaMaxTicks))
	if a < 0 {
		a = 0
	}

	// Progress for adaptive parameters
	progress := float64(st.HuntTick) / float64(woaMaxTicks)
	if progress > 1 {
		progress = 1
	}
	gbWeight := woaGBWeightMin + (woaGBWeightMax-woaGBWeightMin)*progress

	// Precompute targets for all bots
	for i := range ss.Bots {
		st.IsDirect[i] = false

		// Best-bot local random walk around GlobalBest
		if st.BestIdx >= 0 && i == st.BestIdx {
			angle := ss.Rng.Float64() * 2 * math.Pi
			r := ss.Rng.Float64() * woaBestBotWalkR
			tx := st.GlobalBestX + r*math.Cos(angle)
			ty := st.GlobalBestY + r*math.Sin(angle)
			f := distanceFitnessPt(ss, tx, ty)
			if f > st.GlobalBestF {
				st.GlobalBestF = f
				st.GlobalBestX = tx
				st.GlobalBestY = ty
			}
			st.TargetX[i] = tx
			st.TargetY[i] = ty
			continue
		}

		// Direct-to-best: late-phase convergence
		if progress > woaDirectStartProg && st.GlobalBestF > -1e18 {
			prob := woaDirectMaxProb * (progress - woaDirectStartProg) / (1.0 - woaDirectStartProg)
			if ss.Rng.Float64() < prob {
				jitter := 7.5
				tx := st.GlobalBestX + (ss.Rng.Float64()*2-1)*jitter
				ty := st.GlobalBestY + (ss.Rng.Float64()*2-1)*jitter
				f := distanceFitnessPt(ss, tx, ty)
				if f > st.GlobalBestF {
					st.GlobalBestF = f
					st.GlobalBestX = tx
					st.GlobalBestY = ty
				}
				st.TargetX[i] = tx
				st.TargetY[i] = ty
				st.IsDirect[i] = true
				continue
			}
		}

		// Standard WOA phase assignment
		r := ss.Rng.Float64()
		A := 2.0*a*r - a
		p := ss.Rng.Float64()

		var targetX, targetY float64

		if math.Abs(A) >= 1.0 {
			st.Phase[i] = 2 // search (exploration)
			randIdx := ss.Rng.Intn(len(ss.Bots))
			targetX = ss.Bots[randIdx].X
			targetY = ss.Bots[randIdx].Y
		} else if p < woaSpiralProb {
			st.Phase[i] = 1 // spiral (bubble-net)
			dx := st.GlobalBestX - ss.Bots[i].X
			dy := st.GlobalBestY - ss.Bots[i].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1.0 {
				dist = 1.0
			}
			l := ss.Rng.Float64()*2.0 - 1.0
			spiralAngle := math.Atan2(dy, dx) + math.Pi*2.0*l*0.1
			spiralDist := dist * math.Exp(woaSpiralB*l) * math.Cos(2*math.Pi*l)
			if spiralDist > dist*2 {
				spiralDist = dist * 2
			}
			targetX = ss.Bots[i].X + math.Cos(spiralAngle)*spiralDist*0.3
			targetY = ss.Bots[i].Y + math.Sin(spiralAngle)*spiralDist*0.3
		} else {
			st.Phase[i] = 0 // encircle
			targetX, targetY = st.GlobalBestX, st.GlobalBestY
		}

		// Apply global-best attraction
		if st.GlobalBestF > -1e18 {
			targetX = targetX*(1-gbWeight) + st.GlobalBestX*gbWeight
			targetY = targetY*(1-gbWeight) + st.GlobalBestY*gbWeight
		}

		st.TargetX[i] = targetX
		st.TargetY[i] = targetY
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].WOAPhase = st.Phase[i]
		ss.Bots[i].WOAFitness = fitToSensor(st.Fitness[i])
		dx := st.GlobalBestX - ss.Bots[i].X
		dy := st.GlobalBestY - ss.Bots[i].Y
		ss.Bots[i].WOABestDist = int(math.Sqrt(dx*dx + dy*dy))
	}
}

// ApplyWOA moves a bot using the Whale Optimization Algorithm via direct
// position updates (eigenbewegung). Bots move directly so they converge
// in both GUI and benchmark mode.
func ApplyWOA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.WOA == nil {
		bot.Speed = 0
		return
	}
	st := ss.WOA
	if idx >= len(st.Phase) || idx >= len(st.TargetX) {
		bot.Speed = 0
		return
	}

	// LED colors
	if st.IsDirect[idx] {
		bot.LEDColor = [3]uint8{0, 255, 0} // green for direct-to-best
	} else if idx == st.BestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for best
	} else {
		switch st.Phase[idx] {
		case 0:
			bot.LEDColor = [3]uint8{0, 60, 180} // dark blue for encircle
		case 1:
			bot.LEDColor = [3]uint8{0, 200, 255} // cyan for spiral
		case 2:
			bot.LEDColor = [3]uint8{100, 150, 255} // light blue for search
		}
	}

	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, woaSpeedMult)
}

// woaGridRescan evaluates a grid of points across the arena and teleports
// the worst whales to the best-discovered grid positions.
func woaGridRescan(ss *SwarmState, st *WOAState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	gridPts := make([]gridPt, 0, woaGridRescanSize*woaGridRescanSize)
	for gx := 0; gx < woaGridRescanSize; gx++ {
		for gy := 0; gy < woaGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(woaGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(woaGridRescanSize)
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
	if len(gridPts) > 0 && gridPts[0].f > st.GlobalBestF {
		st.GlobalBestF = gridPts[0].f
		st.GlobalBestX = gridPts[0].x
		st.GlobalBestY = gridPts[0].y
	}

	// Find worst bots by fitness
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
	inject := woaGridInjectTop
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
	}
}
