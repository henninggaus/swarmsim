package swarm

import "math"

// Harris Hawks Optimization (HHO): Meta-heuristic inspired by the cooperative
// hunting strategy of Harris's hawks. The algorithm models three phases:
//
//  1. Exploration — hawks perch randomly and search for prey using two
//     strategies: perch based on random tall trees or perch based on
//     the position of other family members (rabbit position).
//  2. Transition — controlled by "escaping energy" E that decreases from
//     2→0 over time. When |E|≥1 hawks explore; when |E|<1 they exploit.
//  3. Exploitation — four strategies based on the combination of:
//     - Soft/Hard besiege: |E|≥0.5 vs |E|<0.5
//     - Prey escapes or not: r<0.5 vs r≥0.5
//     Soft besiege: hawks surround prey and gradually tighten.
//     Hard besiege: hawks converge aggressively on prey.
//     Rapid dive: hawks perform Lévy-flight surprise attacks.
//
// Reference: Heidari, A.A. et al. (2019)
//
//	"Harris hawks optimization: Algorithm and applications",
//	Future Generation Computer Systems.
const (
	hhoMaxTicks        = 3000 // full hunt cycle (matches benchmark length)
	hhoSteerRate       = 0.25 // max steering change per tick (radians)
	hhoLevyBeta        = 1.5  // Lévy flight exponent
	hhoSpeedMult       = 5.0  // movement speed multiplier (5x = 7.5 px/tick)
	hhoGridRescanRate  = 150  // periodic grid rescan every N ticks
	hhoGridRescanSize  = 16   // grid resolution (16×16 = 256 samples)
	hhoGridInjectTop   = 10   // inject top N grid positions into worst hawks
	hhoDirectMaxProb   = 0.80 // max probability of direct-to-best at end
	hhoDirectStartProg = 0.10 // progress threshold to start direct-to-best
	hhoGBWeightMin     = 0.05 // global-best attraction weight at start
	hhoGBWeightMax     = 0.75 // global-best attraction weight at end
	hhoBestBotWalkR    = 40.0 // best-bot local random walk radius
)

// HHOState holds Harris Hawks Optimization state for the swarm.
type HHOState struct {
	Fitness       []float64 // current fitness per hawk
	Phase         []int     // 0=explore, 1=soft besiege, 2=hard besiege, 3=rapid dive
	TargetX       []float64 // precomputed target X per hawk
	TargetY       []float64 // precomputed target Y per hawk
	IsDirect      []bool    // true if hawk is in direct-to-best mode
	HuntTick      int       // ticks into current hunt cycle
	BestIdx       int       // index of rabbit (best hawk) this tick
	BestX         float64   // current tick-best rabbit position
	BestY         float64
	BestF         float64   // current tick-best rabbit fitness
	GlobalBestF   float64   // persistent global best fitness
	GlobalBestX   float64   // persistent global best position X
	GlobalBestY   float64   // persistent global best position Y
	GlobalBestIdx int       // persistent global best index
	CurBestIdx    int       // current tick's best (for LED display)
}

// InitHHO allocates Harris Hawks Optimization state for all bots.
func InitHHO(ss *SwarmState) {
	n := len(ss.Bots)
	ss.HHO = &HHOState{
		Fitness:       make([]float64, n),
		Phase:         make([]int, n),
		TargetX:       make([]float64, n),
		TargetY:       make([]float64, n),
		IsDirect:      make([]bool, n),
		BestIdx:       -1,
		BestF:         -1e18,
		GlobalBestF:   -1e18,
		GlobalBestIdx: -1,
	}
	ss.HHOOn = true
}

// ClearHHO frees Harris Hawks Optimization state.
func ClearHHO(ss *SwarmState) {
	ss.HHO = nil
	ss.HHOOn = false
}

// TickHHO updates the Harris Hawks Optimization for all bots.
func TickHHO(ss *SwarmState) {
	if ss.HHO == nil {
		return
	}
	st := ss.HHO

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		st.Fitness = append(st.Fitness, 0)
		st.Phase = append(st.Phase, 0)
		st.TargetX = append(st.TargetX, 0)
		st.TargetY = append(st.TargetY, 0)
		st.IsDirect = append(st.IsDirect, false)
	}

	st.HuntTick++
	if st.HuntTick > hhoMaxTicks {
		st.HuntTick = 1
		st.BestF = -1e18 // reset cycle-local best
	}

	// Compute fitness for each hawk
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find rabbit (best fitness) — cycle-local
	for i := range ss.Bots {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
			st.BestX = ss.Bots[i].X
			st.BestY = ss.Bots[i].Y
		}
	}

	// Update persistent global best
	st.CurBestIdx = st.BestIdx
	for i := range ss.Bots {
		if st.Fitness[i] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[i]
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
			st.GlobalBestIdx = i
		}
	}

	n := len(ss.Bots)

	// Periodic grid rescan — systematic landscape sampling
	if st.HuntTick > 0 && st.HuntTick%hhoGridRescanRate == 0 && n > 0 {
		hhoGridRescan(ss, st)
	}

	// Escaping energy: E = 2 * E0 * (1 - t/T), where E0 ∈ [-1, 1]
	tRatio := float64(st.HuntTick) / float64(hhoMaxTicks)
	progress := tRatio
	gbWeight := hhoGBWeightMin + (hhoGBWeightMax-hhoGBWeightMin)*progress

	// Assign phases
	for i := range ss.Bots {
		E0 := 2*ss.Rng.Float64() - 1 // random in [-1, 1]
		E := 2 * E0 * (1 - tRatio)
		absE := math.Abs(E)

		if absE >= 1 {
			st.Phase[i] = 0 // exploration
		} else if absE >= 0.5 {
			st.Phase[i] = 1 // soft besiege
		} else {
			r := ss.Rng.Float64()
			if r >= 0.5 {
				st.Phase[i] = 2 // hard besiege
			} else {
				st.Phase[i] = 3 // rapid dive (Lévy flight)
			}
		}
	}

	// Precompute targets for all hawks
	for i := range ss.Bots {
		st.IsDirect[i] = false

		// Best-bot local random walk around GlobalBest
		if st.GlobalBestIdx >= 0 && i == st.GlobalBestIdx {
			angle := ss.Rng.Float64() * 2 * math.Pi
			r := ss.Rng.Float64() * hhoBestBotWalkR
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
		if progress > hhoDirectStartProg && st.GlobalBestIdx >= 0 {
			prob := hhoDirectMaxProb * (progress - hhoDirectStartProg) / (1.0 - hhoDirectStartProg)
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

		// Standard HHO phase-based target computation
		E0 := 2*ss.Rng.Float64() - 1
		E := 2 * E0 * (1 - tRatio)
		var targetX, targetY float64

		switch st.Phase[i] {
		case 0: // Exploration
			q := ss.Rng.Float64()
			if q < 0.5 {
				rIdx := ss.Rng.Intn(len(ss.Bots))
				targetX = ss.Bots[rIdx].X - ss.Rng.Float64()*math.Abs(ss.Bots[rIdx].X-2*ss.Rng.Float64()*ss.Bots[i].X)
				targetY = ss.Bots[rIdx].Y - ss.Rng.Float64()*math.Abs(ss.Bots[rIdx].Y-2*ss.Rng.Float64()*ss.Bots[i].Y)
			} else {
				targetX = st.GlobalBestX - ss.Rng.Float64()*(ss.Rng.Float64()*ss.ArenaW*0.2)
				targetY = st.GlobalBestY - ss.Rng.Float64()*(ss.Rng.Float64()*ss.ArenaH*0.2)
			}

		case 1: // Soft besiege
			J := 2 * (1 - ss.Rng.Float64())
			dx := st.GlobalBestX - ss.Bots[i].X
			dy := st.GlobalBestY - ss.Bots[i].Y
			targetX = st.GlobalBestX - E*math.Abs(J*st.GlobalBestX-ss.Bots[i].X)
			targetY = st.GlobalBestY - E*math.Abs(J*st.GlobalBestY-ss.Bots[i].Y)
			targetX = (targetX + st.GlobalBestX + dx*0.3) / 2
			targetY = (targetY + st.GlobalBestY + dy*0.3) / 2

		case 2: // Hard besiege
			targetX = st.GlobalBestX - E*math.Abs(st.GlobalBestX-ss.Bots[i].X)
			targetY = st.GlobalBestY - E*math.Abs(st.GlobalBestY-ss.Bots[i].Y)

		case 3: // Rapid dive (Lévy flight)
			levy := MantegnaLevy(ss.Rng, 1.5)
			dx := st.GlobalBestX - ss.Bots[i].X
			dy := st.GlobalBestY - ss.Bots[i].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 1 {
				targetX = ss.Bots[i].X + (dx/dist)*levy*30
				targetY = ss.Bots[i].Y + (dy/dist)*levy*30
			} else {
				targetX = st.GlobalBestX
				targetY = st.GlobalBestY
			}
		}

		// Global-Best attraction: blend target toward GlobalBest
		if st.GlobalBestIdx >= 0 {
			targetX = targetX*(1-gbWeight) + st.GlobalBestX*gbWeight
			targetY = targetY*(1-gbWeight) + st.GlobalBestY*gbWeight
		}

		st.TargetX[i] = targetX
		st.TargetY[i] = targetY
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].HHOPhase = st.Phase[i]
		ss.Bots[i].HHOFitness = int(st.Fitness[i])
		if st.GlobalBestIdx >= 0 {
			dx := st.GlobalBestX - ss.Bots[i].X
			dy := st.GlobalBestY - ss.Bots[i].Y
			ss.Bots[i].HHOBestDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].HHOBestDist = 9999
		}
	}
}

// ApplyHHO steers a hawk using precomputed targets.
func ApplyHHO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.HHO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.HHO
	if idx >= len(st.TargetX) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Move toward precomputed target
	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, hhoSpeedMult)

	// LED colour
	if st.IsDirect[idx] {
		bot.LEDColor = [3]uint8{0, 255, 0} // green = direct-to-best
	} else if idx == st.GlobalBestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0} // gold = rabbit/prey
	} else {
		switch st.Phase[idx] {
		case 0:
			bot.LEDColor = [3]uint8{80, 130, 200} // blue = exploring
		case 1:
			bot.LEDColor = [3]uint8{255, 165, 0} // orange = soft besiege
		case 2:
			bot.LEDColor = [3]uint8{255, 50, 50} // red = hard besiege
		case 3:
			bot.LEDColor = [3]uint8{200, 50, 200} // purple = rapid dive
		}
	}
}

// hhoGridRescan performs a systematic grid scan of the landscape and
// teleports the worst hawks to the best grid positions found.
func hhoGridRescan(ss *SwarmState, st *HHOState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	type gridPt struct {
		x, y, f float64
	}
	gridPts := make([]gridPt, 0, hhoGridRescanSize*hhoGridRescanSize)
	for gx := 0; gx < hhoGridRescanSize; gx++ {
		for gy := 0; gy < hhoGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(hhoGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(hhoGridRescanSize)
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

	// Find worst hawks by fitness
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

	// Teleport worst hawks to best grid points
	inject := hhoGridInjectTop
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

// levyStep generates a Lévy-flight step using the shared Mantegna algorithm.
func levyStep(ss *SwarmState) float64 {
	return MantegnaLevy(ss.Rng, hhoLevyBeta)
}
