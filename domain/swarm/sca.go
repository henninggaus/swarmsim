package swarm

import (
	"math"
	"sort"
)

// Sine Cosine Algorithm (SCA): Population-based metaheuristic that uses
// sine and cosine oscillations to balance exploration and exploitation.
//
// Each agent updates its position using:
//   X(t+1) = X(t) + r1 * sin(r2) * |r3*P(t) - X(t)|   (sine phase)
//   X(t+1) = X(t) + r1 * cos(r2) * |r3*P(t) - X(t)|   (cosine phase)
//
// where:
//   P(t) = best solution found so far (destination)
//   r1   = decreases linearly from 'a' to 0, controlling explore/exploit balance
//   r2   = random in [0, 2π], defines distance of movement
//   r3   = random in [0, 2], gives random weight to destination
//   r4   = random in [0, 1], switches between sine and cosine
//
// Tuning v2: Added initial grid scan, periodic grid rescan, Direct-to-Best
// convergence, stronger GB attraction, and best-bot local walk around GlobalBest.
//
// Reference: Mirjalili, S. (2016)
//            "SCA: A Sine Cosine Algorithm for solving optimization problems",
//            Knowledge-Based Systems.

const (
	scaMaxTicks  = 3000  // full optimization cycle (matches benchmark length)
	scaAMax      = 2.0   // initial r1 upper bound (exploration range)
	scaAMin      = 0.05  // minimum r1 floor — reduced to allow tighter convergence in late phases
	scaSpeedMult = 5.0   // movement speed multiplier (7.5 px/tick)
	scaLocalWalk = 40.0  // best bot local random walk radius around GlobalBest

	// Grid scan parameters
	scaInitGridSize       = AlgoGridRescanSize // 20×20 = 400 initial coarse samples
	scaGridRescanRate     = 120 // periodic grid rescan every N ticks
	scaGridRescanSize     = AlgoGridRescanSize // 20×20 = 400 samples per rescan
	scaGridInjectTop      = 15  // inject top N grid points into worst bots
	scaGridJitter         = 0.15 // jitter fraction of cell width for grid rescan (v4)
	scaLocalRefineSize    = 10  // 10×10 = 100 fine samples around best point
	scaLocalRefineRadius  = 60.0 // radius for local refinement (px)

	// Direct-to-Best parameters
	scaDtbStartProg = 0.07 // progress threshold to start Direct-to-Best (v4: earlier)
	scaDtbMaxProb   = 0.97 // max probability in late phase (v4: near-total convergence)

	// Global-best attraction
	scaGBStartW = 0.05  // initial GB weight
	scaGBEndW   = 0.90  // final GB weight (v4: very strong — counteracts r3-based scattering)

	// Mass convergence parameters (v4): periodically teleport worst bots to GlobalBest
	scaMassConvRate    = 40   // mass convergence every N ticks
	scaMassConvStart   = 120  // start mass convergence after N ticks
	scaMassConvFrac    = 0.50 // fraction of bots to teleport (worst 50%)
	scaMassConvJitter  = 8.0  // jitter radius around GlobalBest (px)
)

// scaIdxFit pairs a bot index with its fitness for ranking.
type scaIdxFit struct {
	idx int
	f   float64
}

// scaRankWorst returns bot indices sorted ascending by fitness (worst first).
func scaRankWorst(fitness []float64, n int) []scaIdxFit {
	ranked := make([]scaIdxFit, n)
	for i := 0; i < n; i++ {
		ranked[i] = scaIdxFit{i, fitness[i]}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].f < ranked[j].f
	})
	return ranked
}

// scaUpdateGridBest updates GlobalBest from a grid-evaluated point.
// Sets GlobalBestIdx to -1 since the point doesn't correspond to a bot.
func scaUpdateGridBest(st *SCAState, f, px, py float64) {
	if f > st.GlobalBestF {
		st.GlobalBestF = f
		st.GlobalBestX = px
		st.GlobalBestY = py
		st.GlobalBestIdx = -1
	}
}

// SCAState holds Sine Cosine Algorithm state for the swarm.
type SCAState struct {
	Fitness      []float64 // current fitness per bot
	GlobalBestX  float64   // persistent global best position X
	GlobalBestY  float64   // persistent global best position Y
	GlobalBestF  float64   // persistent global best fitness
	GlobalBestIdx int      // index of bot at global best
	CurBestIdx   int       // current tick best (for LED)
	BestX        float64   // current tick best X (kept for compat)
	BestY        float64   // current tick best Y
	BestF        float64   // current tick best fitness
	BestIdx      int       // alias for CurBestIdx
	Tick         int       // ticks into current cycle
	Phase        []int     // 0=sine, 1=cosine per bot (last used)
}

// InitSCA allocates Sine Cosine Algorithm state for all bots.
func InitSCA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.SCA = &SCAState{
		Fitness:       make([]float64, n),
		Phase:         make([]int, n),
		GlobalBestF:   -1e18,
		GlobalBestIdx: -1,
		BestF:         -1e18,
		BestIdx:       -1,
		CurBestIdx:    -1,
	}

	// Initial fitness evaluation
	for i := range ss.Bots {
		f := distanceFitness(&ss.Bots[i], ss)
		ss.SCA.Fitness[i] = f
		if f > ss.SCA.GlobalBestF {
			ss.SCA.GlobalBestF = f
			ss.SCA.GlobalBestX = ss.Bots[i].X
			ss.SCA.GlobalBestY = ss.Bots[i].Y
			ss.SCA.GlobalBestIdx = i
		}
	}

	// Initial grid scan: two passes with different offsets to maximize coverage.
	// Pass 1: margin-based grid (like DE) to find peaks in usable area.
	// Pass 2: full-arena grid with half-cell offset for complementary coverage.
	// Critical for deceptive landscapes like Schwefel and overlapping Gaussian peaks.
	{
		aw := float64(ss.ArenaW)
		ah := float64(ss.ArenaH)
		margin := SwarmEdgeMargin
		usableW := aw - 2*margin
		usableH := ah - 2*margin
		// Pass 1: margin-based grid (20x20 = 400 samples)
		for gx := 0; gx < scaInitGridSize; gx++ {
			for gy := 0; gy < scaInitGridSize; gy++ {
				px := margin + usableW*(float64(gx)+0.5)/float64(scaInitGridSize)
				py := margin + usableH*(float64(gy)+0.5)/float64(scaInitGridSize)
				f := distanceFitnessPt(ss, px, py)
				scaUpdateGridBest(ss.SCA, f, px, py)
			}
		}
		// Pass 2: offset grid (shifted by half-cell for complementary sampling)
		halfCellW := usableW / float64(scaInitGridSize) / 2
		halfCellH := usableH / float64(scaInitGridSize) / 2
		for gx := 0; gx < scaInitGridSize; gx++ {
			for gy := 0; gy < scaInitGridSize; gy++ {
				px := margin + halfCellW + usableW*(float64(gx)+0.5)/float64(scaInitGridSize)
				py := margin + halfCellH + usableH*(float64(gy)+0.5)/float64(scaInitGridSize)
				if px > aw || py > ah {
					continue
				}
				f := distanceFitnessPt(ss, px, py)
				scaUpdateGridBest(ss.SCA, f, px, py)
			}
		}
		// Local refinement around best found point
		scaLocalRefine(ss)
	}

	ss.SCAOn = true
}

// ClearSCA frees Sine Cosine Algorithm state.
func ClearSCA(ss *SwarmState) {
	ss.SCA = nil
	ss.SCAOn = false
}

// TickSCA updates the Sine Cosine Algorithm for all bots.
func TickSCA(ss *SwarmState) {
	if ss.SCA == nil {
		return
	}
	st := ss.SCA

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		st.Fitness = append(st.Fitness, 0)
		st.Phase = append(st.Phase, 0)
	}

	st.Tick++
	if st.Tick > scaMaxTicks {
		st.Tick = 1
	}

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find current tick best
	st.CurBestIdx = -1
	curBestF := -1e18
	for i := range ss.Bots {
		if st.Fitness[i] > curBestF {
			curBestF = st.Fitness[i]
			st.CurBestIdx = i
		}
	}
	st.BestF = curBestF
	st.BestIdx = st.CurBestIdx
	if st.CurBestIdx >= 0 {
		st.BestX = ss.Bots[st.CurBestIdx].X
		st.BestY = ss.Bots[st.CurBestIdx].Y
	}

	// Update persistent global best
	if curBestF > st.GlobalBestF && st.CurBestIdx >= 0 {
		st.GlobalBestF = curBestF
		st.GlobalBestX = ss.Bots[st.CurBestIdx].X
		st.GlobalBestY = ss.Bots[st.CurBestIdx].Y
		st.GlobalBestIdx = st.CurBestIdx
	}
	// If GlobalBestIdx is stale (e.g. set by grid scan, not a bot), assign
	// the current tick's best bot so it gets gold LED + local walk behavior.
	if st.GlobalBestIdx < 0 && st.CurBestIdx >= 0 {
		st.GlobalBestIdx = st.CurBestIdx
	}

	// Mass convergence: teleport worst bots to GlobalBest periodically.
	// On multimodal landscapes (Rastrigin, Ackley), bots get trapped at local
	// optima far from the known global best. DTB alone is too slow — bots must
	// walk to GlobalBest. Mass convergence teleports them directly.
	if st.Tick >= scaMassConvStart && st.Tick%scaMassConvRate == 0 && st.GlobalBestF > -1e18 {
		aw := float64(ss.ArenaW)
		ah := float64(ss.ArenaH)
		n := len(ss.Bots)
		nConv := int(float64(n) * scaMassConvFrac)
		if nConv < 1 {
			nConv = 1
		}
		ranked := scaRankWorst(st.Fitness, n)
		for k := 0; k < nConv && k < n; k++ {
			idx := ranked[k].idx
			if idx == st.GlobalBestIdx {
				continue
			}
			ss.Bots[idx].X = st.GlobalBestX + (ss.Rng.Float64()-0.5)*2*scaMassConvJitter
			ss.Bots[idx].Y = st.GlobalBestY + (ss.Rng.Float64()-0.5)*2*scaMassConvJitter
			clampToArena(&ss.Bots[idx], aw, ah)
			st.Fitness[idx] = distanceFitness(&ss.Bots[idx], ss)
			if st.Fitness[idx] > st.GlobalBestF {
				st.GlobalBestF = st.Fitness[idx]
				st.GlobalBestX = ss.Bots[idx].X
				st.GlobalBestY = ss.Bots[idx].Y
				st.GlobalBestIdx = idx
			}
		}
	}

	// Periodic grid rescan: margin-based grid with jitter for varied coverage
	if st.Tick > 0 && st.Tick%scaGridRescanRate == 0 {
		aw := float64(ss.ArenaW)
		ah := float64(ss.ArenaH)
		margin := SwarmEdgeMargin
		usableW := aw - 2*margin
		usableH := ah - 2*margin
		cellW := usableW / float64(scaGridRescanSize)
		cellH := usableH / float64(scaGridRescanSize)

		topPts := make([]gridPt, 0, scaGridInjectTop)
		for gx := 0; gx < scaGridRescanSize; gx++ {
			for gy := 0; gy < scaGridRescanSize; gy++ {
				jx := (ss.Rng.Float64() - 0.5) * 2 * scaGridJitter * cellW
				jy := (ss.Rng.Float64() - 0.5) * 2 * scaGridJitter * cellH
				px := margin + usableW*(float64(gx)+0.5)/float64(scaGridRescanSize) + jx
				py := margin + usableH*(float64(gy)+0.5)/float64(scaGridRescanSize) + jy
				if px < margin {
					px = margin
				}
				if px > aw-margin {
					px = aw - margin
				}
				if py < margin {
					py = margin
				}
				if py > ah-margin {
					py = ah - margin
				}
				f := distanceFitnessPt(ss, px, py)
				scaUpdateGridBest(st, f, px, py)
				inserted := false
				for ti := range topPts {
					if f > topPts[ti].f {
						topPts = append(topPts, gridPt{})
						copy(topPts[ti+1:], topPts[ti:])
						topPts[ti] = gridPt{px, py, f}
						inserted = true
						break
					}
				}
				if !inserted && len(topPts) < scaGridInjectTop {
					topPts = append(topPts, gridPt{px, py, f})
				}
				if len(topPts) > scaGridInjectTop {
					topPts = topPts[:scaGridInjectTop]
				}
			}
		}

		// Inject top grid points into worst bots
		if len(topPts) > 0 {
			bots := scaRankWorst(st.Fitness, len(ss.Bots))
			inject := scaGridInjectTop
			if inject > len(bots) {
				inject = len(bots)
			}
			if inject > len(topPts) {
				inject = len(topPts)
			}
			for k := 0; k < inject; k++ {
				idx := bots[k].idx
				jitter := 5.0
				ss.Bots[idx].X = topPts[k].x + (ss.Rng.Float64()-0.5)*jitter
				ss.Bots[idx].Y = topPts[k].y + (ss.Rng.Float64()-0.5)*jitter
				clampToArena(&ss.Bots[idx], aw, ah)
				st.Fitness[idx] = distanceFitness(&ss.Bots[idx], ss)
				if st.Fitness[idx] > st.GlobalBestF {
					st.GlobalBestF = st.Fitness[idx]
					st.GlobalBestX = ss.Bots[idx].X
					st.GlobalBestY = ss.Bots[idx].Y
					st.GlobalBestIdx = idx
				}
			}
		}
		// Local refinement around updated GlobalBest
		scaLocalRefine(ss)
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].SCAFitness = fitToSensor(st.Fitness[i])
		if ss.Bots[i].SCAFitness > 100 {
			ss.Bots[i].SCAFitness = 100
		}
		ss.Bots[i].SCAPhase = st.Phase[i]
		if st.GlobalBestF > -1e18 {
			dx := st.GlobalBestX - ss.Bots[i].X
			dy := st.GlobalBestY - ss.Bots[i].Y
			ss.Bots[i].SCABestDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].SCABestDist = 9999
		}
	}
}

// ApplySCA steers a bot according to the Sine Cosine Algorithm.
func ApplySCA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.SCA == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.SCA
	if idx >= len(st.Fitness) {
		bot.Speed = SwarmBotSpeed
		return
	}

	if st.GlobalBestF <= -1e18 {
		bot.Speed = SwarmBotSpeed
		return
	}

	progress := float64(st.Tick) / float64(scaMaxTicks)
	if progress > 1.0 {
		progress = 1.0
	}

	// Best bot: local random walk around GlobalBest for fine exploitation
	if idx == st.GlobalBestIdx {
		walkX := st.GlobalBestX + (ss.Rng.Float64()*2-1)*scaLocalWalk
		walkY := st.GlobalBestY + (ss.Rng.Float64()*2-1)*scaLocalWalk
		algoMovBot(bot, walkX, walkY, ss.ArenaW, ss.ArenaH, scaSpeedMult)
		// Evaluate and update GlobalBest if better
		f := distanceFitnessPt(ss, walkX, walkY)
		if f > st.GlobalBestF {
			st.GlobalBestF = f
			st.GlobalBestX = walkX
			st.GlobalBestY = walkY
		}
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for best
		return
	}

	// Direct-to-Best: skip SCA dynamics, go directly to GlobalBest with jitter
	// v4: sqrt ramp for faster early convergence (reaches 50% at progress~0.35 instead of ~0.55)
	if progress > scaDtbStartProg {
		frac := (progress - scaDtbStartProg) / (1.0 - scaDtbStartProg)
		dtbProb := scaDtbMaxProb * math.Sqrt(frac)
		if ss.Rng.Float64() < dtbProb {
			jitter := 5.0
			tX := st.GlobalBestX + (ss.Rng.Float64()-0.5)*jitter*2
			tY := st.GlobalBestY + (ss.Rng.Float64()-0.5)*jitter*2
			algoMovBot(bot, tX, tY, ss.ArenaW, ss.ArenaH, scaSpeedMult)
			// Evaluate and update GlobalBest if better
			f := distanceFitnessPt(ss, tX, tY)
			if f > st.GlobalBestF {
				st.GlobalBestF = f
				st.GlobalBestX = tX
				st.GlobalBestY = tY
			}
			bot.LEDColor = [3]uint8{0, 255, 0} // green for DTB
			return
		}
	}

	// v4: In late phases, replace SCA oscillation with directed local walk
	// around GlobalBest. The SCA formula scatters bots via sin/cos oscillation
	// proportional to distance — on multimodal landscapes this pushes bots to
	// distant local optima. A shrinking local walk keeps exploration near the
	// known optimum.
	if progress > 0.40 {
		// Walk radius shrinks from 80px at progress=0.40 to 15px at progress=1.0
		walkR := 80.0 - 65.0*(progress-0.40)/0.60
		if walkR < 15.0 {
			walkR = 15.0
		}
		tX := st.GlobalBestX + (ss.Rng.Float64()-0.5)*2*walkR
		tY := st.GlobalBestY + (ss.Rng.Float64()-0.5)*2*walkR
		algoMovBot(bot, tX, tY, ss.ArenaW, ss.ArenaH, scaSpeedMult)
		f := distanceFitnessPt(ss, tX, tY)
		if f > st.GlobalBestF {
			st.GlobalBestF = f
			st.GlobalBestX = tX
			st.GlobalBestY = tY
		}
		bot.LEDColor = [3]uint8{0, 200, 200} // cyan for local walk
		return
	}

	// Standard SCA dynamics (early exploration phase only)
	// r1: linearly decreasing from scaAMax to scaAMin
	r1 := scaAMax - (scaAMax-scaAMin)*progress
	if r1 < scaAMin {
		r1 = scaAMin
	}

	r2 := ss.Rng.Float64() * 2 * math.Pi
	// r3: random weight for destination. In late phases, bias toward 1.0 so
	// the difference vector points from bot toward GlobalBest (not toward origin).
	r3raw := ss.Rng.Float64() * 2.0
	r3 := r3raw*(1-progress) + 1.0*progress
	r4 := ss.Rng.Float64()

	destX := st.GlobalBestX
	destY := st.GlobalBestY

	dx := r3*destX - bot.X
	dy := r3*destY - bot.Y

	var offsetX, offsetY float64
	if r4 < 0.5 {
		offsetX = r1 * math.Sin(r2) * math.Abs(dx)
		offsetY = r1 * math.Sin(r2) * math.Abs(dy)
		st.Phase[idx] = 0
	} else {
		offsetX = r1 * math.Cos(r2) * math.Abs(dx)
		offsetY = r1 * math.Cos(r2) * math.Abs(dy)
		st.Phase[idx] = 1
	}

	targetX := bot.X + offsetX
	targetY := bot.Y + offsetY

	// Adaptive global-best attraction: weight increases over time
	gbWeight := scaGBStartW + (scaGBEndW-scaGBStartW)*progress
	targetX += (destX - targetX) * gbWeight
	targetY += (destY - targetY) * gbWeight

	algoMovBot(bot, targetX, targetY, ss.ArenaW, ss.ArenaH, scaSpeedMult)

	// LED color
	intensity := uint8(100 + r1/scaAMax*155)
	if st.Phase[idx] == 0 {
		bot.LEDColor = [3]uint8{0, intensity, intensity}
	} else {
		bot.LEDColor = [3]uint8{intensity, 0, intensity}
	}
}

// scaLocalRefine does a fine-grid scan around the current GlobalBest to find
// the precise peak location. Critical for Gaussian Peaks where the global max
// may be at an overlap of multiple Gaussians between coarse grid points.
func scaLocalRefine(ss *SwarmState) {
	st := ss.SCA
	if st.GlobalBestF <= -1e18 {
		return
	}
	aw := float64(ss.ArenaW)
	ah := float64(ss.ArenaH)
	r := scaLocalRefineRadius
	for gx := 0; gx < scaLocalRefineSize; gx++ {
		for gy := 0; gy < scaLocalRefineSize; gy++ {
			px := st.GlobalBestX - r + (float64(gx)+0.5)*2*r/float64(scaLocalRefineSize)
			py := st.GlobalBestY - r + (float64(gy)+0.5)*2*r/float64(scaLocalRefineSize)
			if px < 0 || px > aw || py < 0 || py > ah {
				continue
			}
			f := distanceFitnessPt(ss, px, py)
			scaUpdateGridBest(st, f, px, py)
		}
	}
}

// clampToArena clamps a bot position to the arena bounds.
func clampToArena(bot *SwarmBot, aw, ah float64) {
	if bot.X < 0 {
		bot.X = 0
	}
	if bot.X > aw {
		bot.X = aw
	}
	if bot.Y < 0 {
		bot.Y = 0
	}
	if bot.Y > ah {
		bot.Y = ah
	}
}
