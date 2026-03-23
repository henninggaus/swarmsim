package swarm

import "math"

// Grey Wolf Optimizer (GWO): Meta-heuristic inspired by the social hierarchy
// and hunting strategy of grey wolves. The pack has four ranks:
//   Alpha (α) — best solution, leads the hunt
//   Beta  (β) — second best, assists alpha
//   Delta (δ) — third best, scouts and sentinels
//   Omega (ω) — remaining wolves, follow the leaders
//
// Hunting phases: (1) Search for prey (exploration), (2) Encircle prey,
// (3) Attack (exploitation). The parameter 'a' decreases linearly from
// 2→0 over time, transitioning from exploration to exploitation.
//
// Reference: Mirjalili, S., Mirjalili, S.M. & Lewis, A. (2014)
//            "Grey Wolf Optimizer", Advances in Engineering Software.

const (
	gwoRadius       = 100.0 // neighbor detection radius
	gwoMaxTicks     = 600   // full hunt cycle length
	gwoSteerRate    = 0.15  // max steering change per tick (radians)
	gwoEncircleWt   = 0.6   // weight for encircling behavior
	gwoCohesionWt   = 0.3   // weight for pack cohesion
	gwoMinNeighbors = 3     // minimum neighbors to form a pack
)

// GWOState holds Grey Wolf Optimizer state for the swarm.
type GWOState struct {
	Rank      []int     // 0=alpha, 1=beta, 2=delta, 3=omega
	Fitness   []float64 // current fitness per bot (higher = better)
	HuntTick  int       // ticks into current hunt cycle
	AlphaIdx  int       // index of alpha wolf
	BetaIdx   int       // index of beta wolf
	DeltaIdx  int       // index of delta wolf
	AlphaX    float64   // alpha position
	AlphaY    float64
	BetaX     float64   // beta position
	BetaY     float64
	DeltaX    float64   // delta position
	DeltaY    float64
}

// InitGWO allocates Grey Wolf Optimizer state for all bots.
func InitGWO(ss *SwarmState) {
	n := len(ss.Bots)
	ss.GWO = &GWOState{
		Rank:    make([]int, n),
		Fitness: make([]float64, n),
	}
	// Initially all omega; leaders determined by fitness
	for i := range ss.GWO.Rank {
		ss.GWO.Rank[i] = 3
	}
	ss.GWOOn = true
}

// ClearGWO frees Grey Wolf Optimizer state.
func ClearGWO(ss *SwarmState) {
	ss.GWO = nil
	ss.GWOOn = false
}

// TickGWO updates the Grey Wolf Optimizer for all bots.
// Computes fitness, assigns ranks, updates sensor cache.
func TickGWO(ss *SwarmState) {
	if ss.GWO == nil {
		return
	}
	st := ss.GWO

	// Grow slices if bots were added
	for len(st.Rank) < len(ss.Bots) {
		st.Rank = append(st.Rank, 3)
		st.Fitness = append(st.Fitness, 0)
	}

	st.HuntTick++
	if st.HuntTick > gwoMaxTicks {
		st.HuntTick = 1
	}

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find top 3 fitness indices (alpha, beta, delta)
	st.AlphaIdx, st.BetaIdx, st.DeltaIdx = -1, -1, -1
	bestF, secF, thiF := -1.0, -1.0, -1.0

	for i := range ss.Bots {
		f := st.Fitness[i]
		if f > bestF {
			thiF = secF
			st.DeltaIdx = st.BetaIdx
			secF = bestF
			st.BetaIdx = st.AlphaIdx
			bestF = f
			st.AlphaIdx = i
		} else if f > secF {
			thiF = secF
			st.DeltaIdx = st.BetaIdx
			secF = f
			st.BetaIdx = i
		} else if f > thiF {
			thiF = f
			st.DeltaIdx = i
		}
	}

	// Assign ranks
	for i := range st.Rank {
		st.Rank[i] = 3 // omega
	}
	if st.AlphaIdx >= 0 {
		st.Rank[st.AlphaIdx] = 0
		st.AlphaX = ss.Bots[st.AlphaIdx].X
		st.AlphaY = ss.Bots[st.AlphaIdx].Y
	}
	if st.BetaIdx >= 0 {
		st.Rank[st.BetaIdx] = 1
		st.BetaX = ss.Bots[st.BetaIdx].X
		st.BetaY = ss.Bots[st.BetaIdx].Y
	}
	if st.DeltaIdx >= 0 {
		st.Rank[st.DeltaIdx] = 2
		st.DeltaX = ss.Bots[st.DeltaIdx].X
		st.DeltaY = ss.Bots[st.DeltaIdx].Y
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].GWORank = st.Rank[i]
		ss.Bots[i].GWOFitness = fitToSensor(st.Fitness[i])
		if st.AlphaIdx >= 0 {
			dx := st.AlphaX - ss.Bots[i].X
			dy := st.AlphaY - ss.Bots[i].Y
			ss.Bots[i].GWOAlphaDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].GWOAlphaDist = 9999
		}
	}
}

// ApplyGWO steers a bot according to Grey Wolf hunting behavior.
// Omega wolves move toward the average position of alpha, beta, and delta.
// The convergence coefficient 'a' decreases over the hunt cycle, transitioning
// from wide exploration to tight encirclement.
func ApplyGWO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.GWO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.GWO
	if idx >= len(st.Rank) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Alpha, beta, delta keep their natural behavior (they lead)
	if st.Rank[idx] <= 2 {
		bot.Speed = SwarmBotSpeed
		// Leaders get bright LED colors to indicate rank
		switch st.Rank[idx] {
		case 0: // alpha — gold
			bot.LEDColor = [3]uint8{255, 215, 0}
		case 1: // beta — silver
			bot.LEDColor = [3]uint8{192, 192, 192}
		case 2: // delta — bronze
			bot.LEDColor = [3]uint8{205, 127, 50}
		}
		return
	}

	// Linearly decreasing convergence coefficient: a = 2 * (1 - t/T)
	a := 2.0 * (1.0 - float64(st.HuntTick)/float64(gwoMaxTicks))
	if a < 0 {
		a = 0
	}

	// Random coefficients for stochastic exploration
	r1 := ss.Rng.Float64()
	r2 := ss.Rng.Float64()
	A := 2.0*a*r1 - a // coefficient vector A ∈ [-a, a]
	C := 2.0 * r2      // coefficient vector C ∈ [0, 2]

	// Compute target position as weighted average of alpha, beta, delta positions
	targetX, targetY := 0.0, 0.0
	leaders := 0

	if st.AlphaIdx >= 0 {
		dAlpha := C*st.AlphaX - bot.X
		targetX += st.AlphaX - A*dAlpha
		dAlphaY := C*st.AlphaY - bot.Y
		targetY += st.AlphaY - A*dAlphaY
		leaders++
	}
	if st.BetaIdx >= 0 {
		r1 = ss.Rng.Float64()
		r2 = ss.Rng.Float64()
		A2 := 2.0*a*r1 - a
		C2 := 2.0 * r2
		dBeta := C2*st.BetaX - bot.X
		targetX += st.BetaX - A2*dBeta
		dBetaY := C2*st.BetaY - bot.Y
		targetY += st.BetaY - A2*dBetaY
		leaders++
	}
	if st.DeltaIdx >= 0 {
		r1 = ss.Rng.Float64()
		r2 = ss.Rng.Float64()
		A3 := 2.0*a*r1 - a
		C3 := 2.0 * r2
		dDelta := C3*st.DeltaX - bot.X
		targetX += st.DeltaX - A3*dDelta
		dDeltaY := C3*st.DeltaY - bot.Y
		targetY += st.DeltaY - A3*dDeltaY
		leaders++
	}

	if leaders == 0 {
		bot.Speed = SwarmBotSpeed
		return
	}

	targetX /= float64(leaders)
	targetY /= float64(leaders)

	// Steer toward target
	desired := math.Atan2(targetY-bot.Y, targetX-bot.X)
	steerToward(bot, desired, gwoSteerRate)
	bot.Speed = SwarmBotSpeed

	// Omega wolves: dark grey LED
	bot.LEDColor = [3]uint8{100, 100, 100}
}
