package swarm

import (
	"math"
	"sort"
)

// Bat Algorithm (BA): Meta-heuristic inspired by the echolocation behavior
// of microbats. Each bat emits ultrasonic pulses and listens for echoes to
// detect prey (the optimum). Key parameters:
//
//   Frequency (f): Controls the velocity/step size. Each bat tunes its
//     frequency within [fMin, fMax] to vary exploration/exploitation.
//   Pulse rate (r): Probability of performing a local search near the best
//     solution. Starts low and increases over time as the bat converges.
//   Loudness (A): Controls acceptance of new solutions. Starts high
//     (aggressive exploration) and decreases as the bat closes in on prey.
//
// Reference: Yang, X.-S. (2010) "A New Metaheuristic Bat-Inspired Algorithm",
//            Nature Inspired Cooperative Strategies for Optimization (NICSO).

const (
	batFMin      = 0.0  // minimum frequency
	batFMax      = 2.0  // maximum frequency
	batAlpha     = 0.95 // loudness decay rate (0 < alpha < 1)
	batGamma     = 0.9  // pulse rate increase coefficient
	batSteerRate = 0.25 // max steering change per tick (radians)
	batMaxSpeed  = 2.5  // max bot speed under BA
	batLocalStep = 15.0 // local random walk step size
	batSpeedMult = 5.0  // movement speed multiplier for eigenbewegung (was 3.0)
	batMaxTicks  = 3000 // full benchmark length

	batGridRescanRate    = 150  // periodic grid rescan every N ticks (was 300)
	batGridRescanSize    = 20   // grid resolution (20×20 = 400 samples, was 14)
	batGridInjectTop     = 15   // top grid points injected into worst bats (was 10)
	batGridJitter        = 0.15 // grid jitter as fraction of cell width (was 0.02)
	batLocalRefineN      = 10   // local refinement grid side (10×10=100 points)
	batLocalRefineR      = 60.0 // local refinement radius around best grid point
	batDirectToBestStart = 0.10 // progress threshold to start direct-to-best (was 0.3)
	batDirectToBestMax   = 0.85 // max probability of direct-to-best (was 0.55)
	batGBEndW            = 0.80 // max global-best attraction weight (was 0.55)
)

// BatState holds per-bot echolocation state for the Bat Algorithm.
type BatState struct {
	Freq    []float64    // frequency per bat
	Vel     [2][]float64 // velocity components [0]=X, [1]=Y
	Loud    []float64    // loudness per bat (decreases over time)
	Pulse   []float64    // pulse emission rate per bat (increases over time)
	Fitness []float64    // fitness per bat
	BestX   float64      // current tick best position X
	BestY   float64      // current tick best position Y
	BestF   float64      // current tick best fitness
	BestIdx int          // index of best bat
	Tick    int          // iteration counter
	AvgLoud float64      // precomputed average loudness (avoids O(n²) in ApplyBat)
	// Persistent global best (never reset)
	GlobalBestF float64
	GlobalBestX float64
	GlobalBestY float64
	// Personal best tracking: each bat remembers its own best position.
	PBestX []float64
	PBestY []float64
	PBestF []float64
	// Stagnation tracking for grid rescan triggering
	StagnCounter int
	LastBestF    float64
}

// InitBat allocates Bat Algorithm state.
func InitBat(ss *SwarmState) {
	n := len(ss.Bots)
	st := &BatState{
		Freq:        make([]float64, n),
		Vel:         [2][]float64{make([]float64, n), make([]float64, n)},
		Loud:        make([]float64, n),
		Pulse:       make([]float64, n),
		Fitness:     make([]float64, n),
		PBestX:      make([]float64, n),
		PBestY:      make([]float64, n),
		PBestF:      make([]float64, n),
		BestF:       -1e18,
		BestIdx:     -1,
		AvgLoud:     1.0,
		GlobalBestF: -1e18,
	}
	// Initialize each bat with full loudness and zero pulse rate.
	// Personal best starts at the bot's initial position.
	for i := range ss.Bots {
		st.Loud[i] = 1.0
		st.Pulse[i] = 0.0
		st.Freq[i] = batFMin + ss.Rng.Float64()*(batFMax-batFMin)
		st.PBestX[i] = ss.Bots[i].X
		st.PBestY[i] = ss.Bots[i].Y
		st.PBestF[i] = -1e18
	}
	ss.Bat = st
	ss.BatOn = true
}

// ClearBat frees Bat Algorithm state.
func ClearBat(ss *SwarmState) {
	ss.Bat = nil
	ss.BatOn = false
}

// TickBat runs one iteration of the Bat Algorithm across all bots.
// Updates frequencies, velocities, loudness, pulse rates, and global best.
func TickBat(ss *SwarmState) {
	st := ss.Bat
	if st == nil {
		return
	}

	// Grow slices if bots were added dynamically
	for len(st.Freq) < len(ss.Bots) {
		idx := len(st.Freq)
		st.Freq = append(st.Freq, batFMin+ss.Rng.Float64()*(batFMax-batFMin))
		st.Vel[0] = append(st.Vel[0], 0)
		st.Vel[1] = append(st.Vel[1], 0)
		st.Loud = append(st.Loud, 1.0)
		st.Pulse = append(st.Pulse, 0.0)
		st.Fitness = append(st.Fitness, 0)
		st.PBestX = append(st.PBestX, ss.Bots[idx].X)
		st.PBestY = append(st.PBestY, ss.Bots[idx].Y)
		st.PBestF = append(st.PBestF, -1e18)
	}

	st.Tick++

	// Evaluate fitness for each bat using the shared fitness landscape.
	// Update personal bests (each bat remembers its own best position).
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
		if st.Fitness[i] > st.PBestF[i] {
			st.PBestF[i] = st.Fitness[i]
			st.PBestX[i] = ss.Bots[i].X
			st.PBestY[i] = ss.Bots[i].Y
		}
	}

	// Find current tick best and update persistent global best
	for i := range ss.Bots {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestX = ss.Bots[i].X
			st.BestY = ss.Bots[i].Y
			st.BestIdx = i
		}
		if st.Fitness[i] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[i]
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
		}
	}

	// Track stagnation for grid rescan
	if st.GlobalBestF > st.LastBestF+0.01 {
		st.LastBestF = st.GlobalBestF
		st.StagnCounter = 0
	} else {
		st.StagnCounter++
	}

	// Initial grid scan at tick 1: find the global optimum early
	// before bats fixate on local optima.
	n := len(ss.Bots)
	if st.Tick == 1 && n > 0 {
		batGridRescan(ss, st)
	}

	// Periodic grid rescan: systematically sample the arena to find the
	// global optimum on deceptive landscapes like Schwefel.
	if st.Tick > 1 && st.Tick%batGridRescanRate == 0 && n > 0 {
		batGridRescan(ss, st)
	}

	// Precompute average loudness once per tick (O(n)).
	// Previously this was computed inside ApplyBat per bot → O(n²).
	loudSum := 0.0
	for _, l := range st.Loud {
		loudSum += l
	}
	if len(st.Loud) > 0 {
		st.AvgLoud = loudSum / float64(len(st.Loud))
	}

	// Update sensor cache for SwarmScript
	for i := range ss.Bots {
		ss.Bots[i].BatLoud = int(st.Loud[i] * 100)
		ss.Bots[i].BatPulse = int(st.Pulse[i] * 100)
		ss.Bots[i].BatFitness = int(st.Fitness[i])
		if st.BestIdx >= 0 {
			dx := st.BestX - ss.Bots[i].X
			dy := st.BestY - ss.Bots[i].Y
			ss.Bots[i].BatBestDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].BatBestDist = 9999
		}
	}
}

// ApplyBat steers a single bat according to the echolocation algorithm.
// Uses direct position updates (eigenbewegung) so bots move in benchmark mode.
// Uses the precomputed AvgLoud from TickBat to avoid O(n²) recomputation.
// Velocity update blends attraction to both global best and personal best
// for improved convergence (cf. Yang 2010, enhanced BA variants).
func ApplyBat(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.Bat
	if st == nil || idx >= len(st.Freq) {
		bot.Speed = 0
		return
	}

	// Progress for adaptive parameters (0→1 over batMaxTicks)
	progress := float64(st.Tick) / float64(batMaxTicks)
	if progress > 1 {
		progress = 1
	}

	// Direct-to-best: in late phase, some bats skip normal dynamics
	// and move directly to GlobalBest with jitter — critical for Schwefel convergence.
	if progress > batDirectToBestStart && st.GlobalBestF > -1e18 {
		dtbProb := batDirectToBestMax * (progress - batDirectToBestStart) / (1.0 - batDirectToBestStart)
		if ss.Rng.Float64() < dtbProb {
			jitter := 7.5
			tX := st.GlobalBestX + (ss.Rng.Float64()*2-1)*jitter
			tY := st.GlobalBestY + (ss.Rng.Float64()*2-1)*jitter
			// Clamp
			if tX < SwarmBotRadius {
				tX = SwarmBotRadius
			}
			if tX > ss.ArenaW-SwarmBotRadius {
				tX = ss.ArenaW - SwarmBotRadius
			}
			if tY < SwarmBotRadius {
				tY = SwarmBotRadius
			}
			if tY > ss.ArenaH-SwarmBotRadius {
				tY = ss.ArenaH - SwarmBotRadius
			}
			// Evaluate and accept if better
			f := distanceFitnessPt(ss, tX, tY)
			if f > st.GlobalBestF {
				st.GlobalBestF = f
				st.GlobalBestX = tX
				st.GlobalBestY = tY
			}
			algoMovBot(bot, tX, tY, ss.ArenaW, ss.ArenaH, batSpeedMult)
			bot.LEDColor = [3]uint8{0, 200, 100} // green for direct-to-best
			return
		}
	}

	// Update frequency: f_i = fMin + (fMax - fMin) * beta, beta ∈ [0,1]
	beta := ss.Rng.Float64()
	st.Freq[idx] = batFMin + (batFMax-batFMin)*beta

	// Update velocity toward global best with personal best influence.
	// Standard BA pulls toward global best; we add a cognitive component
	// toward the bat's personal best for better exploration/exploitation balance.
	if st.BestIdx >= 0 {
		// Global attraction (social)
		st.Vel[0][idx] += (st.GlobalBestX - bot.X) * st.Freq[idx] * 0.01
		st.Vel[1][idx] += (st.GlobalBestY - bot.Y) * st.Freq[idx] * 0.01
		// Personal best attraction (cognitive) — weaker than global
		if idx < len(st.PBestX) {
			st.Vel[0][idx] += (st.PBestX[idx] - bot.X) * st.Freq[idx] * 0.005
			st.Vel[1][idx] += (st.PBestY[idx] - bot.Y) * st.Freq[idx] * 0.005
		}
	}

	// Candidate new position
	newX := bot.X + st.Vel[0][idx]
	newY := bot.Y + st.Vel[1][idx]

	// Local search: if random > pulse rate, perturb around global best
	if ss.Rng.Float64() > st.Pulse[idx] && st.GlobalBestF > -1e18 {
		// Random walk around global best solution scaled by precomputed average loudness.
		newX = st.GlobalBestX + batLocalStep*st.AvgLoud*(ss.Rng.Float64()-0.5)*2
		newY = st.GlobalBestY + batLocalStep*st.AvgLoud*(ss.Rng.Float64()-0.5)*2
	}

	// Adaptive global-best attraction: shift target toward global best
	// Weight increases from 5% to 80% over batMaxTicks (was 5%→55%)
	if st.GlobalBestF > -1e18 {
		gbWeight := 0.05 + (batGBEndW-0.05)*progress
		newX = newX*(1-gbWeight) + st.GlobalBestX*gbWeight
		newY = newY*(1-gbWeight) + st.GlobalBestY*gbWeight
	}

	// Clamp to arena
	if newX < SwarmBotRadius {
		newX = SwarmBotRadius
	}
	if newX > ss.ArenaW-SwarmBotRadius {
		newX = ss.ArenaW - SwarmBotRadius
	}
	if newY < SwarmBotRadius {
		newY = SwarmBotRadius
	}
	if newY > ss.ArenaH-SwarmBotRadius {
		newY = ss.ArenaH - SwarmBotRadius
	}

	// Accept new position if it improves fitness or if random < loudness.
	newFit := distanceFitnessPt(ss, newX, newY)
	if newFit > st.Fitness[idx] || ss.Rng.Float64() < st.Loud[idx] {
		// Move directly to accepted position (eigenbewegung)
		algoMovBot(bot, newX, newY, ss.ArenaW, ss.ArenaH, batSpeedMult)

		// Decrease loudness and increase pulse rate
		st.Loud[idx] *= batAlpha
		if st.Loud[idx] < 0.01 {
			st.Loud[idx] = 0.01
		}
		st.Pulse[idx] = st.Pulse[idx] + (1.0-st.Pulse[idx])*(1.0-math.Exp(-batGamma*float64(st.Tick)*0.01))
		if st.Pulse[idx] > 0.99 {
			st.Pulse[idx] = 0.99
		}
	} else {
		// Rejected — stay in place
		bot.Speed = 0
	}

	// Best-bot local walk: the bat nearest to GlobalBest explores locally
	// around the global best for fine-grained exploitation (radius 40px).
	if st.GlobalBestF > -1e18 {
		// Find if this bot is closest to GlobalBest
		dxGB := bot.X - st.GlobalBestX
		dyGB := bot.Y - st.GlobalBestY
		myDist := dxGB*dxGB + dyGB*dyGB
		isBestBot := true
		for j := range ss.Bots {
			if j == idx {
				continue
			}
			djx := ss.Bots[j].X - st.GlobalBestX
			djy := ss.Bots[j].Y - st.GlobalBestY
			if djx*djx+djy*djy < myDist {
				isBestBot = false
				break
			}
		}
		if isBestBot {
			lwR := 40.0
			lwX := st.GlobalBestX + (ss.Rng.Float64()*2-1)*lwR
			lwY := st.GlobalBestY + (ss.Rng.Float64()*2-1)*lwR
			if lwX < SwarmBotRadius {
				lwX = SwarmBotRadius
			}
			if lwX > ss.ArenaW-SwarmBotRadius {
				lwX = ss.ArenaW - SwarmBotRadius
			}
			if lwY < SwarmBotRadius {
				lwY = SwarmBotRadius
			}
			if lwY > ss.ArenaH-SwarmBotRadius {
				lwY = ss.ArenaH - SwarmBotRadius
			}
			lwF := distanceFitnessPt(ss, lwX, lwY)
			if lwF > st.GlobalBestF {
				st.GlobalBestF = lwF
				st.GlobalBestX = lwX
				st.GlobalBestY = lwY
			}
			algoMovBot(bot, lwX, lwY, ss.ArenaW, ss.ArenaH, batSpeedMult)
			bot.LEDColor = [3]uint8{0, 255, 255} // cyan for best-bot
			return
		}
	}

	// LED color: pulse rate as blue intensity, loudness as red intensity
	r := uint8(st.Loud[idx] * 255)
	b := uint8(st.Pulse[idx] * 255)
	g := uint8(40)
	if st.BestIdx == idx {
		// Best bat glows bright cyan
		r, g, b = 0, 255, 255
	}
	bot.LEDColor = [3]uint8{r, g, b}
}

// batGridRescan evaluates a grid of points across the arena and teleports
// the worst bats to the best-discovered grid positions. Critical for
// deceptive landscapes like Schwefel where the global optimum is far
// from local optima.
func batGridRescan(ss *SwarmState, st *BatState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	cellW := usableW / float64(batGridRescanSize)
	cellH := usableH / float64(batGridRescanSize)

	gridPts := make([]gridPt, 0, batGridRescanSize*batGridRescanSize)
	for gx := 0; gx < batGridRescanSize; gx++ {
		for gy := 0; gy < batGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(batGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(batGridRescanSize)
			// 15% jitter (was 2%) — evaluates different positions each rescan
			x += (ss.Rng.Float64()*2.0 - 1.0) * cellW * batGridJitter
			y += (ss.Rng.Float64()*2.0 - 1.0) * cellH * batGridJitter
			f := distanceFitnessPt(ss, x, y)
			gridPts = append(gridPts, gridPt{x, y, f})
		}
	}

	// Sort grid points by fitness descending
	sort.Slice(gridPts, func(i, j int) bool { return gridPts[i].f > gridPts[j].f })

	// Update GlobalBest from grid findings
	if len(gridPts) > 0 && gridPts[0].f > st.GlobalBestF {
		st.GlobalBestF = gridPts[0].f
		st.GlobalBestX = gridPts[0].x
		st.GlobalBestY = gridPts[0].y
	}

	// Local refinement: fine 10×10 grid around best grid point (radius 60px)
	// Finds precise peaks in overlapping zones (critical for Gaussian Peaks).
	if len(gridPts) > 0 {
		cX, cY := gridPts[0].x, gridPts[0].y
		for rx := 0; rx < batLocalRefineN; rx++ {
			for ry := 0; ry < batLocalRefineN; ry++ {
				lx := cX + (float64(rx)/float64(batLocalRefineN-1)*2-1)*batLocalRefineR
				ly := cY + (float64(ry)/float64(batLocalRefineN-1)*2-1)*batLocalRefineR
				if lx < SwarmBotRadius {
					lx = SwarmBotRadius
				}
				if lx > ss.ArenaW-SwarmBotRadius {
					lx = ss.ArenaW - SwarmBotRadius
				}
				if ly < SwarmBotRadius {
					ly = SwarmBotRadius
				}
				if ly > ss.ArenaH-SwarmBotRadius {
					ly = ss.ArenaH - SwarmBotRadius
				}
				lf := distanceFitnessPt(ss, lx, ly)
				if lf > st.GlobalBestF {
					st.GlobalBestF = lf
					st.GlobalBestX = lx
					st.GlobalBestY = ly
				}
			}
		}
	}

	// Find worst bats by fitness
	bats := make([]idxFit, n)
	for i := range ss.Bots {
		bats[i] = idxFit{i, st.Fitness[i]}
	}
	// Sort ascending by fitness (worst first)
	sort.Slice(bats, func(i, j int) bool { return bats[i].f < bats[j].f })

	// Teleport worst bats to best grid points
	inject := batGridInjectTop
	if inject > len(gridPts) {
		inject = len(gridPts)
	}
	if inject > n {
		inject = n
	}
	for i := 0; i < inject; i++ {
		bi := bats[i].idx
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
		// Reset velocity for teleported bats
		st.Vel[0][bi] = 0
		st.Vel[1][bi] = 0
		// Reset loudness high and pulse rate low for fresh exploration
		st.Loud[bi] = 0.8
		st.Pulse[bi] = 0.1
	}
}
