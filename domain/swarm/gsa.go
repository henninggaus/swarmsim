package swarm

import (
	"math"
	"sort"
)

// Gravitational Search Algorithm (GSA): Physics-inspired metaheuristic where
// agents are masses that interact via Newtonian gravity. Fitter agents have
// higher mass and attract weaker agents more strongly. Over time, the
// gravitational constant G decays, transitioning from exploration (strong
// long-range forces) to exploitation (weak, local-only forces).
//
// Key mechanics:
//   - Mass is proportional to fitness (distance to target)
//   - Gravitational force F = G * Mi * Mj / (r + eps)
//   - G decays exponentially: G(t) = G0 * exp(-alpha * t / T)
//   - Only the K-best (heaviest) agents exert force (Kbest decreases over time)
//   - Acceleration = sum(forces) / mass_i
//
// Reference: Rashedi, E., Nezamabadi-pour, H. & Saryazdi, S. (2009)
//            "GSA: A Gravitational Search Algorithm", Information Sciences.

const (
	gsaG0          = 100.0  // initial gravitational constant
	gsaAlphaDecay  = 20.0   // G decay rate
	gsaMaxTicks    = 3000   // full cycle length (matches benchmark length)
	gsaSteerRate   = 0.30   // max steering change per tick (radians)
	gsaEps         = 1.0    // softening constant to avoid division by zero
	gsaMaxAccel    = 3.0    // acceleration clamp
	gsaSpeedMult   = 5.0    // movement speed multiplier (5.0 * 1.5 = 7.5 px/tick)
	gsaGridRescanRate = 300  // periodic grid rescan every N ticks
	gsaGridRescanSize = 14   // grid resolution (14×14 = 196 samples)
	gsaGridInjectTop  = AlgoGridInjectTop // best grid positions to inject per rescan
	gsaDirectProb     = 0.55 // max direct-to-best probability in late phase
)

// GSAState holds Gravitational Search Algorithm state for the swarm.
type GSAState struct {
	Mass    []float64 // normalised mass per bot (0-1)
	Fitness []float64 // raw fitness per bot (higher = better)
	AccX    []float64 // acceleration X per bot
	AccY    []float64 // acceleration Y per bot
	Tick    int       // current tick in cycle
	BestIdx int       // index of heaviest agent (current tick)
	BestX   float64   // position of heaviest agent (current tick)
	BestY   float64
	G       float64 // current gravitational constant
	// Persistent global best across entire run
	GlobalBestF float64
	GlobalBestX float64
	GlobalBestY float64
	sortBuf     []float64 // reusable buffer for K-best partial sort (avoids per-tick allocation)
	kBest       []bool    // reusable boolean mask for K-best agents
	IsDirect    []bool    // per-bot flag: true if doing direct-to-best this tick
	TargetX     []float64 // per-bot target X for algoMovBot
	TargetY     []float64 // per-bot target Y for algoMovBot
}

// InitGSA allocates Gravitational Search Algorithm state.
func InitGSA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.GSA = &GSAState{
		Mass:        make([]float64, n),
		Fitness:     make([]float64, n),
		AccX:        make([]float64, n),
		AccY:        make([]float64, n),
		G:           gsaG0,
		GlobalBestF: -1e18,
		IsDirect:    make([]bool, n),
		TargetX:     make([]float64, n),
		TargetY:     make([]float64, n),
	}
	ss.GSAOn = true
}

// ClearGSA frees Gravitational Search Algorithm state.
func ClearGSA(ss *SwarmState) {
	ss.GSA = nil
	ss.GSAOn = false
}

// gsaMinSearchRadius is the minimum spatial hash query radius.
// Even late in the cycle (low G), we search at least this far.
const gsaMinSearchRadius = 100.0

// TickGSA updates the Gravitational Search Algorithm for all bots.
// Computes fitness, normalises masses, decays G, computes gravitational forces.
//
// Performance: Uses the spatial hash for force computation (O(n·k) where k is
// the average number of nearby agents) instead of brute-force O(n²). The search
// radius shrinks with the gravitational constant G: early in the cycle (large G,
// exploration) forces reach far, late in the cycle (small G, exploitation) only
// nearby agents matter. Falls back to brute force if Hash is nil.
func TickGSA(ss *SwarmState) {
	if ss.GSA == nil {
		return
	}
	st := ss.GSA
	n := len(ss.Bots)

	// Grow slices if bots were added
	for len(st.Mass) < n {
		st.Mass = append(st.Mass, 0)
		st.Fitness = append(st.Fitness, 0)
		st.AccX = append(st.AccX, 0)
		st.AccY = append(st.AccY, 0)
		st.IsDirect = append(st.IsDirect, false)
		st.TargetX = append(st.TargetX, 0)
		st.TargetY = append(st.TargetY, 0)
	}
	// Grow reusable buffers
	if cap(st.sortBuf) < n {
		st.sortBuf = make([]float64, n)
		st.kBest = make([]bool, n)
	}
	st.sortBuf = st.sortBuf[:n]
	st.kBest = st.kBest[:n]

	st.Tick++
	if st.Tick > gsaMaxTicks {
		st.Tick = 1
	}

	// Decay gravitational constant: G(t) = G0 * exp(-alpha * t / T)
	st.G = gsaG0 * math.Exp(-gsaAlphaDecay*float64(st.Tick)/float64(gsaMaxTicks))

	// Compute fitness using shared Gaussian landscape
	bestFit := -math.MaxFloat64
	worstFit := math.MaxFloat64

	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
		if st.Fitness[i] > bestFit {
			bestFit = st.Fitness[i]
			st.BestIdx = i
		}
		if st.Fitness[i] < worstFit {
			worstFit = st.Fitness[i]
		}
		// Update persistent global best
		if st.Fitness[i] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[i]
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
		}
	}

	st.BestX = ss.Bots[st.BestIdx].X
	st.BestY = ss.Bots[st.BestIdx].Y

	// Normalise masses: m_i = (fit_i - worst) / (best - worst + eps)
	fitRange := bestFit - worstFit + gsaEps
	totalMass := 0.0
	for i := range ss.Bots {
		st.Mass[i] = (st.Fitness[i] - worstFit) / fitRange
		totalMass += st.Mass[i]
	}
	// Normalise so masses sum to 1
	if totalMass > 0 {
		for i := range st.Mass {
			st.Mass[i] /= totalMass
		}
	}

	// K-best: only the top K agents exert gravitational force.
	// K decreases linearly from N to 1 over the cycle.
	kCount := n - int(float64(n-1)*float64(st.Tick)/float64(gsaMaxTicks))
	if kCount < 1 {
		kCount = 1
	}

	// Find K-best using reusable buffer (partial selection sort, no allocation)
	copy(st.sortBuf, st.Mass[:n])
	for i := 0; i < kCount && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < n; j++ {
			if st.sortBuf[j] > st.sortBuf[maxIdx] {
				maxIdx = j
			}
		}
		st.sortBuf[i], st.sortBuf[maxIdx] = st.sortBuf[maxIdx], st.sortBuf[i]
	}
	massThreshold := 0.0
	if kCount < n {
		massThreshold = st.sortBuf[kCount-1]
	}

	// Build boolean K-best mask for O(1) lookup during force computation
	for i := 0; i < n; i++ {
		st.kBest[i] = st.Mass[i] >= massThreshold
	}

	// Compute gravitational acceleration for each bot
	for i := range ss.Bots {
		st.AccX[i] = 0
		st.AccY[i] = 0
	}

	// Search radius proportional to G: large early (exploration), small late (exploitation).
	// Arena diagonal is the maximum useful distance.
	arenaDiag := math.Sqrt(ss.ArenaW*ss.ArenaW + ss.ArenaH*ss.ArenaH)
	gRatio := st.G / gsaG0 // 1.0 → 0.0 over the cycle
	searchRadius := gsaMinSearchRadius + (arenaDiag-gsaMinSearchRadius)*math.Sqrt(gRatio)

	useSpatialHash := ss.Hash != nil && searchRadius < arenaDiag*0.9

	for i := range ss.Bots {
		if useSpatialHash {
			// Spatial hash query: only check nearby candidates
			candidates := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, searchRadius)
			for _, j := range candidates {
				if j == i || !st.kBest[j] {
					continue
				}
				dx := ss.Bots[j].X - ss.Bots[i].X
				dy := ss.Bots[j].Y - ss.Bots[i].Y
				r := math.Sqrt(dx*dx + dy*dy)
				force := st.G * st.Mass[i] * st.Mass[j] / (r + gsaEps)
				randJ := ss.Rng.Float64()
				if r > 0 {
					st.AccX[i] += randJ * force * dx / r
					st.AccY[i] += randJ * force * dy / r
				}
			}
		} else {
			// Brute-force fallback (Hash nil or radius covers whole arena)
			for j := range ss.Bots {
				if j == i || !st.kBest[j] {
					continue
				}
				dx := ss.Bots[j].X - ss.Bots[i].X
				dy := ss.Bots[j].Y - ss.Bots[i].Y
				r := math.Sqrt(dx*dx + dy*dy)
				force := st.G * st.Mass[i] * st.Mass[j] / (r + gsaEps)
				randJ := ss.Rng.Float64()
				if r > 0 {
					st.AccX[i] += randJ * force * dx / r
					st.AccY[i] += randJ * force * dy / r
				}
			}
		}

		// a_i = F_i / M_i (avoid division by zero for zero-mass agents)
		if st.Mass[i] > 1e-10 {
			st.AccX[i] /= st.Mass[i]
			st.AccY[i] /= st.Mass[i]
		}

		// Clamp acceleration
		accMag := math.Sqrt(st.AccX[i]*st.AccX[i] + st.AccY[i]*st.AccY[i])
		if accMag > gsaMaxAccel {
			scale := gsaMaxAccel / accMag
			st.AccX[i] *= scale
			st.AccY[i] *= scale
		}
	}

	// Periodic grid rescan: systematically sample the arena to find the
	// global optimum on deceptive landscapes like Schwefel.
	if st.Tick > 0 && st.Tick%gsaGridRescanRate == 0 && n > 0 {
		gsaGridRescan(ss, st)
	}

	// Compute per-bot targets: either direct-to-best or gravity-based movement
	progress := float64(st.Tick) / float64(gsaMaxTicks)
	for i := range ss.Bots {
		st.IsDirect[i] = false

		// Direct-to-best: ab progress > 0.3, steigend bis gsaDirectProb
		if progress > 0.3 && st.GlobalBestF > -1e17 {
			directP := gsaDirectProb * (progress - 0.3) / 0.7
			if ss.Rng.Float64() < directP {
				st.IsDirect[i] = true
				jitter := 7.5
				st.TargetX[i] = st.GlobalBestX + (ss.Rng.Float64()*2-1)*jitter
				st.TargetY[i] = st.GlobalBestY + (ss.Rng.Float64()*2-1)*jitter
				// Evaluate the direct target and update GlobalBest if better
				f := distanceFitnessPt(ss, st.TargetX[i], st.TargetY[i])
				if f > st.GlobalBestF {
					st.GlobalBestF = f
					st.GlobalBestX = st.TargetX[i]
					st.GlobalBestY = st.TargetY[i]
				}
				continue
			}
		}

		// Gravity-based target: acceleration direction + global-best attraction
		ax, ay := st.AccX[i], st.AccY[i]
		accMag := math.Sqrt(ax*ax + ay*ay)

		// Base target from acceleration
		tx, ty := ss.Bots[i].X, ss.Bots[i].Y
		if accMag > 1e-10 {
			step := SwarmBotSpeed * gsaSpeedMult
			tx += ax / accMag * step
			ty += ay / accMag * step
		} else {
			// Random walk if no acceleration
			tx += (ss.Rng.Float64()*2 - 1) * SwarmBotSpeed * gsaSpeedMult
			ty += (ss.Rng.Float64()*2 - 1) * SwarmBotSpeed * gsaSpeedMult
		}

		// Global-best attraction: weight increases from 5% to 55% over time
		gbWeight := 0.05 + 0.50*progress
		if st.GlobalBestF > -1e17 {
			tx = tx*(1-gbWeight) + st.GlobalBestX*gbWeight
			ty = ty*(1-gbWeight) + st.GlobalBestY*gbWeight
		}

		st.TargetX[i] = tx
		st.TargetY[i] = ty
	}

	// Best-bot local random walk around GlobalBest
	if st.GlobalBestF > -1e17 && n > 0 {
		bi := st.BestIdx
		radius := 40.0
		st.TargetX[bi] = st.GlobalBestX + (ss.Rng.Float64()*2-1)*radius
		st.TargetY[bi] = st.GlobalBestY + (ss.Rng.Float64()*2-1)*radius
		st.IsDirect[bi] = false
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].GSAMass = int(st.Mass[i] * 1000) // mass * 1000 for integer sensor
		ss.Bots[i].GSAForce = int(math.Sqrt(st.AccX[i]*st.AccX[i]+st.AccY[i]*st.AccY[i]) * 100)
		dx := st.BestX - ss.Bots[i].X
		dy := st.BestY - ss.Bots[i].Y
		ss.Bots[i].GSABestDist = int(math.Sqrt(dx*dx + dy*dy))
	}
}

// gsaMovBot moves a bot directly by (dx, dy), clamps to arena, and sets Speed=0
// to prevent double-movement when applySwarmPhysics runs in GUI mode.
func gsaMovBot(bot *SwarmBot, dx, dy, arenaW, arenaH float64) {
	bot.X += dx
	bot.Y += dy
	if bot.X < 0 {
		bot.X = 0
	} else if bot.X > arenaW {
		bot.X = arenaW
	}
	if bot.Y < 0 {
		bot.Y = 0
	} else if bot.Y > arenaH {
		bot.Y = arenaH
	}
	bot.Speed = 0
}

// gsaGridRescan evaluates a grid of points across the arena and teleports
// the worst agents to the best-discovered grid positions. Critical for
// deceptive landscapes like Schwefel where the global optimum is far
// from local optima.
func gsaGridRescan(ss *SwarmState, st *GSAState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	gridPts := make([]gridPt, 0, gsaGridRescanSize*gsaGridRescanSize)
	for gx := 0; gx < gsaGridRescanSize; gx++ {
		for gy := 0; gy < gsaGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(gsaGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(gsaGridRescanSize)
			// Small jitter
			x += (ss.Rng.Float64()*2.0 - 1.0) * usableW * 0.02
			y += (ss.Rng.Float64()*2.0 - 1.0) * usableH * 0.02
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

	// Find worst agents by fitness
	agents := make([]idxFit, n)
	for i := range ss.Bots {
		agents[i] = idxFit{i, st.Fitness[i]}
	}
	// Sort ascending by fitness (worst first)
	sort.Slice(agents, func(i, j int) bool { return agents[i].f < agents[j].f })

	// Teleport worst agents to best grid points
	inject := gsaGridInjectTop
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
		// Reset acceleration for teleported agents
		st.AccX[bi] = 0
		st.AccY[bi] = 0
	}
}

// ApplyGSA moves a bot toward its computed target via algoMovBot with 5x speed.
// Targets are computed in TickGSA: either direct-to-best or gravity+attraction.
func ApplyGSA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.GSA == nil || idx >= len(ss.GSA.AccX) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.GSA

	// Move toward pre-computed target
	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, gsaSpeedMult)

	// LED: mass as color intensity (heavy = bright red, light = dim blue)
	mass01 := st.Mass[idx] * float64(len(ss.Bots)) // denormalise
	if mass01 > 1 {
		mass01 = 1
	}
	r := uint8(mass01 * 255)
	b := uint8((1 - mass01) * 180)
	// Best agent gets gold, direct-to-best gets green
	if idx == st.BestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0}
	} else if st.IsDirect[idx] {
		bot.LEDColor = [3]uint8{0, 200, 80}
	} else {
		bot.LEDColor = [3]uint8{r, 40, b}
	}
}
