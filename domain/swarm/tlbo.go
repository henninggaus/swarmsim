package swarm

import "math"

// Teaching-Learning-Based Optimization (TLBO): A parameter-free population-based
// metaheuristic inspired by the teaching-learning process in a classroom.
//
// TLBO is unique among metaheuristics because it requires NO algorithm-specific
// parameters (no inertia weight, no mutation rate, no crossover probability, etc.).
// It uses only common control parameters: population size and number of iterations.
//
// Two phases per iteration:
//
//  1. Teacher Phase — The best individual (teacher) tries to raise the class mean.
//     Each learner moves toward the teacher, adjusted by the difference between
//     the teacher and the mean position scaled by a Teaching Factor (TF ∈ {1, 2}).
//     TF = round(1 + rand()) is randomized per iteration to add exploration.
//     New position: X_new = X_old + r * (Teacher - TF * Mean)
//
//  2. Learner Phase — Each learner interacts with a randomly selected peer.
//     If the peer has better fitness, the learner moves toward the peer;
//     otherwise it moves away. This creates a self-improving dynamic where
//     knowledge flows from better to worse individuals.
//     If peer is better:  X_new = X_old + r * (Peer - X_old)
//     If self is better:  X_new = X_old + r * (X_old - Peer)
//
// Reference: Rao, R.V., Savsani, V.J., & Vakharia, D.P. (2011)
//
//	"Teaching-learning-based optimization: A novel method for constrained
//	 mechanical design optimization problems",
//	 Computer-Aided Design, 43(3), 303-315.
const (
	tlboMaxTicks        = 3000 // full optimization cycle (matches benchmark length)
	tlboSteerRate       = 0.25 // max steering change per tick (radians)
	tlboSpeedMult       = 5.0  // movement speed multiplier (7.5 px/tick)
	tlboGridRescanRate  = 200  // periodic grid rescan every N ticks
	tlboGridRescanSize  = 16   // grid resolution (16×16 = 256 samples)
	tlboGridRescanTopK  = 10   // number of best grid points to use
)

// TLBOState holds Teaching-Learning-Based Optimization state for the swarm.
type TLBOState struct {
	Fitness       []float64 // current fitness per bot
	BestX         float64   // current tick teacher position X
	BestY         float64   // current tick teacher position Y
	BestF         float64   // current tick teacher fitness
	BestIdx       int       // index of current tick teacher bot
	GlobalBestF   float64   // persistent best fitness over entire run
	GlobalBestX   float64   // persistent best position X
	GlobalBestY   float64   // persistent best position Y
	GlobalBestIdx int       // persistent best bot index
	MeanX         float64   // class mean position X
	MeanY         float64   // class mean position Y
	Phase         []int     // 0=teacher phase, 1=learner phase per bot (last used)
	PeerIdx       []int     // index of randomly chosen peer for learner phase
	TargetX       []float64 // precomputed target X per bot
	TargetY       []float64 // precomputed target Y per bot
	IsDirect      []bool    // true if bot uses direct-to-best this tick
	Tick          int       // ticks into current cycle
}

// InitTLBO allocates TLBO state for all bots.
func InitTLBO(ss *SwarmState) {
	n := len(ss.Bots)
	ss.TLBO = &TLBOState{
		Fitness:       make([]float64, n),
		Phase:         make([]int, n),
		PeerIdx:       make([]int, n),
		TargetX:       make([]float64, n),
		TargetY:       make([]float64, n),
		IsDirect:      make([]bool, n),
		BestF:         -1e18,
		BestIdx:       -1,
		GlobalBestF:   -1e18,
		GlobalBestIdx: -1,
	}
	ss.TLBOOn = true
}

// ClearTLBO frees TLBO state.
func ClearTLBO(ss *SwarmState) {
	ss.TLBO = nil
	ss.TLBOOn = false
}

// TickTLBO updates the TLBO algorithm for all bots.
func TickTLBO(ss *SwarmState) {
	if ss.TLBO == nil {
		return
	}
	st := ss.TLBO
	n := len(ss.Bots)

	// Grow slices if bots were added
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.Phase = append(st.Phase, 0)
		st.PeerIdx = append(st.PeerIdx, 0)
		st.TargetX = append(st.TargetX, 0)
		st.TargetY = append(st.TargetY, 0)
		st.IsDirect = append(st.IsDirect, false)
	}

	st.Tick++
	if st.Tick > tlboMaxTicks {
		st.Tick = 1
	}

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find teacher (best individual in current tick)
	st.BestIdx = -1
	st.BestF = -1e18
	for i := 0; i < n; i++ {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
		}
	}
	if st.BestIdx >= 0 {
		st.BestX = ss.Bots[st.BestIdx].X
		st.BestY = ss.Bots[st.BestIdx].Y
	}

	// Update persistent global best
	if st.BestF > st.GlobalBestF {
		st.GlobalBestF = st.BestF
		st.GlobalBestX = st.BestX
		st.GlobalBestY = st.BestY
		st.GlobalBestIdx = st.BestIdx
	}

	// Compute class mean position
	st.MeanX = 0
	st.MeanY = 0
	for i := 0; i < n; i++ {
		st.MeanX += ss.Bots[i].X
		st.MeanY += ss.Bots[i].Y
	}
	if n > 0 {
		st.MeanX /= float64(n)
		st.MeanY /= float64(n)
	}

	// Assign random peers for learner phase (each bot picks a different peer)
	for i := 0; i < n; i++ {
		peer := ss.Rng.Intn(n)
		for peer == i && n > 1 {
			peer = ss.Rng.Intn(n)
		}
		st.PeerIdx[i] = peer
	}

	// Periodic grid rescan: systematically sample the arena
	if st.Tick > 0 && st.Tick%tlboGridRescanRate == 0 && n > 0 {
		tlboGridRescan(ss, st)
	}

	// Alternate between teacher and learner phases based on tick parity
	// Even ticks = teacher phase, odd ticks = learner phase
	currentPhase := st.Tick % 2
	progress := float64(st.Tick) / float64(tlboMaxTicks)

	for i := 0; i < n; i++ {
		st.Phase[i] = currentPhase
		st.IsDirect[i] = false

		// Direct-to-Best: after progress > 0.2, increasing probability (0→70%)
		if progress > 0.2 && st.GlobalBestF > -1e17 && i != st.GlobalBestIdx {
			directProb := 0.70 * (progress - 0.2) / 0.8 // 0 -> 70%
			if ss.Rng.Float64() < directProb {
				jitter := 7.5
				tx := st.GlobalBestX + (ss.Rng.Float64()*2-1)*jitter
				ty := st.GlobalBestY + (ss.Rng.Float64()*2-1)*jitter
				tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
				ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))
				// Evaluate direct-to-best point
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

		// Best bot: local random walk around GlobalBest
		if i == st.GlobalBestIdx && st.GlobalBestF > -1e17 {
			walkR := 40.0
			tx := st.GlobalBestX + (ss.Rng.Float64()*2-1)*walkR
			ty := st.GlobalBestY + (ss.Rng.Float64()*2-1)*walkR
			tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
			ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))
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

		// Standard TLBO dynamics: compute target
		var targetX, targetY float64
		if currentPhase == 0 {
			// Teacher Phase: move toward teacher, adjusted by class mean
			tf := float64(1 + ss.Rng.Intn(2)) // TF = 1 or 2
			r := ss.Rng.Float64()
			diffX := st.BestX - tf*st.MeanX
			diffY := st.BestY - tf*st.MeanY
			targetX = ss.Bots[i].X + r*diffX
			targetY = ss.Bots[i].Y + r*diffY
		} else {
			// Learner Phase: interact with random peer
			peer := st.PeerIdx[i]
			if peer >= len(st.Fitness) {
				peer = 0
			}
			r := ss.Rng.Float64()
			peerBot := &ss.Bots[peer]
			if st.Fitness[i] < st.Fitness[peer] {
				targetX = ss.Bots[i].X + r*(peerBot.X-ss.Bots[i].X)
				targetY = ss.Bots[i].Y + r*(peerBot.Y-ss.Bots[i].Y)
			} else {
				targetX = ss.Bots[i].X + r*(ss.Bots[i].X-peerBot.X)
				targetY = ss.Bots[i].Y + r*(ss.Bots[i].Y-peerBot.Y)
			}
		}

		// Adaptive global-best attraction: 5% -> 65%
		gbWeight := 0.05 + 0.60*progress
		if st.GlobalBestF > -1e17 {
			targetX = targetX*(1-gbWeight) + st.GlobalBestX*gbWeight
			targetY = targetY*(1-gbWeight) + st.GlobalBestY*gbWeight
		}

		targetX = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, targetX))
		targetY = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, targetY))

		st.TargetX[i] = targetX
		st.TargetY[i] = targetY
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].TLBOFitness = fitToSensor(st.Fitness[i])
		if ss.Bots[i].TLBOFitness > 100 {
			ss.Bots[i].TLBOFitness = 100
		}
		ss.Bots[i].TLBOPhase = st.Phase[i]
		if st.BestIdx >= 0 {
			dx := st.BestX - ss.Bots[i].X
			dy := st.BestY - ss.Bots[i].Y
			ss.Bots[i].TLBOTeacherDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].TLBOTeacherDist = 9999
		}
	}
}

// ApplyTLBO moves a bot according to precomputed TLBO targets with 5x speed.
func ApplyTLBO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.TLBO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.TLBO
	if idx >= len(st.Fitness) || idx >= len(st.TargetX) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Move to precomputed target with 5x speed (7.5 px/tick)
	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, tlboSpeedMult)

	// LED colors
	if idx == st.GlobalBestIdx {
		// Global best: gold
		bot.LEDColor = [3]uint8{255, 215, 0}
	} else if st.IsDirect[idx] {
		// Direct-to-best: green
		bot.LEDColor = [3]uint8{50, 255, 50}
	} else if st.Phase[idx] == 0 {
		// Teacher phase: green tones
		intensity := uint8(80 + st.Fitness[idx]*175)
		if intensity < 80 {
			intensity = 80
		}
		bot.LEDColor = [3]uint8{0, intensity, intensity / 3}
	} else {
		// Learner phase: blue tones
		intensity := uint8(80 + st.Fitness[idx]*175)
		if intensity < 80 {
			intensity = 80
		}
		bot.LEDColor = [3]uint8{intensity / 3, intensity / 2, intensity}
	}
}

// tlboGridRescan evaluates a grid of points across the arena and teleports
// the worst learners to the best-discovered grid positions. Critical for
// deceptive landscapes like Schwefel.
func tlboGridRescan(ss *SwarmState, st *TLBOState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	type gridPt struct {
		x, y, f float64
	}
	gridPts := make([]gridPt, 0, tlboGridRescanSize*tlboGridRescanSize)
	for gx := 0; gx < tlboGridRescanSize; gx++ {
		for gy := 0; gy < tlboGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(tlboGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(tlboGridRescanSize)
			// Small jitter
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

	// Update GlobalBest from grid findings
	if len(gridPts) > 0 && gridPts[0].f > st.GlobalBestF {
		st.GlobalBestF = gridPts[0].f
		st.GlobalBestX = gridPts[0].x
		st.GlobalBestY = gridPts[0].y
	}

	// Find worst bots by fitness
	type idxFit struct {
		idx int
		f   float64
	}
	worstBots := make([]idxFit, n)
	for i := 0; i < n; i++ {
		worstBots[i] = idxFit{i, st.Fitness[i]}
	}
	// Sort ascending (worst first)
	for i := 0; i < len(worstBots)-1; i++ {
		for j := i + 1; j < len(worstBots); j++ {
			if worstBots[j].f < worstBots[i].f {
				worstBots[i], worstBots[j] = worstBots[j], worstBots[i]
			}
		}
	}

	// Teleport worst bots to best grid positions
	topK := tlboGridRescanTopK
	if topK > len(gridPts) {
		topK = len(gridPts)
	}
	if topK > n {
		topK = n
	}
	for k := 0; k < topK; k++ {
		bi := worstBots[k].idx
		jitter := 5.0
		ss.Bots[bi].X = gridPts[k].x + (ss.Rng.Float64()*2-1)*jitter
		ss.Bots[bi].Y = gridPts[k].y + (ss.Rng.Float64()*2-1)*jitter
		ss.Bots[bi].X = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, ss.Bots[bi].X))
		ss.Bots[bi].Y = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ss.Bots[bi].Y))
	}
}
