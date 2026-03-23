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
	tlboMaxTicks  = 600  // full optimization cycle
	tlboSteerRate = 0.15 // max steering change per tick (radians)
)

// TLBOState holds Teaching-Learning-Based Optimization state for the swarm.
type TLBOState struct {
	Fitness   []float64 // current fitness per bot
	BestX     float64   // teacher (best) position X
	BestY     float64   // teacher (best) position Y
	BestF     float64   // teacher fitness
	BestIdx   int       // index of teacher bot
	MeanX     float64   // class mean position X
	MeanY     float64   // class mean position Y
	Phase     []int     // 0=teacher phase, 1=learner phase per bot (last used)
	PeerIdx   []int     // index of randomly chosen peer for learner phase
	Tick      int       // ticks into current cycle
}

// InitTLBO allocates TLBO state for all bots.
func InitTLBO(ss *SwarmState) {
	n := len(ss.Bots)
	ss.TLBO = &TLBOState{
		Fitness: make([]float64, n),
		Phase:   make([]int, n),
		PeerIdx: make([]int, n),
		BestF:   -1e18,
		BestIdx: -1,
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
	}

	st.Tick++
	if st.Tick > tlboMaxTicks {
		st.Tick = 1
	}

	// Compute fitness using shared landscape
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		neighborFit := float64(bot.NeighborCount) / 10.0
		if neighborFit > 1.0 {
			neighborFit = 1.0
		}
		carryFit := 0.0
		if bot.CarryingPkg >= 0 {
			carryFit = 0.3
		}
		landFit := distanceFitness(bot, ss) / 100.0
		if landFit < 0 {
			landFit = 0
		}
		st.Fitness[i] = neighborFit*0.4 + carryFit + landFit*0.3
	}

	// Find teacher (best individual)
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

	// Alternate between teacher and learner phases based on tick parity
	// Even ticks = teacher phase, odd ticks = learner phase
	currentPhase := st.Tick % 2
	for i := 0; i < n; i++ {
		st.Phase[i] = currentPhase
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].TLBOFitness = int(st.Fitness[i] * 100)
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

// ApplyTLBO steers a bot according to the TLBO algorithm.
func ApplyTLBO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.TLBO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.TLBO
	if idx >= len(st.Fitness) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Teacher (best bot) keeps natural behavior
	if idx == st.BestIdx {
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for teacher
		return
	}

	if st.BestIdx < 0 {
		bot.Speed = SwarmBotSpeed
		return
	}

	var targetX, targetY float64

	if st.Phase[idx] == 0 {
		// Teacher Phase: move toward teacher, adjusted by class mean
		// Teaching Factor TF = round(1 + rand()) → 1 or 2
		tf := float64(1 + ss.Rng.Intn(2)) // 1 or 2
		r := ss.Rng.Float64()

		// X_new = X_old + r * (Teacher - TF * Mean)
		diffX := st.BestX - tf*st.MeanX
		diffY := st.BestY - tf*st.MeanY
		targetX = bot.X + r*diffX
		targetY = bot.Y + r*diffY
	} else {
		// Learner Phase: interact with a random peer
		peer := st.PeerIdx[idx]
		if peer >= len(st.Fitness) {
			peer = 0
		}
		r := ss.Rng.Float64()

		peerBot := &ss.Bots[peer]
		if st.Fitness[idx] < st.Fitness[peer] {
			// Peer is better: move toward peer
			targetX = bot.X + r*(peerBot.X-bot.X)
			targetY = bot.Y + r*(peerBot.Y-bot.Y)
		} else {
			// Self is better: move away from peer
			targetX = bot.X + r*(bot.X-peerBot.X)
			targetY = bot.Y + r*(bot.Y-peerBot.Y)
		}
	}

	// Steer toward target
	dx := targetX - bot.X
	dy := targetY - bot.Y
	if dx != 0 || dy != 0 {
		desired := math.Atan2(dy, dx)
		steerToward(bot, desired, tlboSteerRate)
	}
	bot.Speed = SwarmBotSpeed

	// LED color: teacher phase = green tones, learner phase = blue tones
	// Brightness scales with fitness
	intensity := uint8(80 + st.Fitness[idx]*175)
	if intensity < 80 {
		intensity = 80
	}
	if st.Phase[idx] == 0 {
		// Teacher phase: green (learning from the best)
		bot.LEDColor = [3]uint8{0, intensity, intensity / 3}
	} else {
		// Learner phase: blue (peer interaction)
		bot.LEDColor = [3]uint8{intensity / 3, intensity / 2, intensity}
	}
}
