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
	woaRadius      = 90.0  // neighbor detection radius
	woaMaxTicks    = 500   // full hunt cycle
	woaSteerRate   = 0.12  // max steering change per tick (radians)
	woaSpiralB     = 1.0   // spiral shape constant
	woaSpiralProb  = 0.5   // probability of spiral vs shrink
)

// WOAState holds Whale Optimization Algorithm state.
type WOAState struct {
	Fitness   []float64 // current fitness per bot
	BestIdx   int       // index of best whale (prey position)
	BestX     float64   // best whale position
	BestY     float64
	HuntTick  int       // current tick in hunt cycle
	Phase     []int     // per-bot: 0=encircle, 1=spiral, 2=search
}

// InitWOA allocates Whale Optimization state for all bots.
func InitWOA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.WOA = &WOAState{
		Fitness: make([]float64, n),
		Phase:   make([]int, n),
		BestIdx: -1,
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
	}

	st.HuntTick++
	if st.HuntTick > woaMaxTicks {
		st.HuntTick = 1
	}

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find best whale
	st.BestIdx = 0
	bestF := st.Fitness[0]
	for i := 1; i < len(ss.Bots); i++ {
		if st.Fitness[i] > bestF {
			bestF = st.Fitness[i]
			st.BestIdx = i
		}
	}
	st.BestX = ss.Bots[st.BestIdx].X
	st.BestY = ss.Bots[st.BestIdx].Y

	// Linearly decreasing a: 2 → 0
	a := 2.0 * (1.0 - float64(st.HuntTick)/float64(woaMaxTicks))
	if a < 0 {
		a = 0
	}

	// Assign phase per bot
	for i := range ss.Bots {
		r := ss.Rng.Float64()
		A := 2.0*a*r - a
		p := ss.Rng.Float64()

		if math.Abs(A) >= 1.0 {
			st.Phase[i] = 2 // search (exploration)
		} else if p < woaSpiralProb {
			st.Phase[i] = 1 // spiral (bubble-net)
		} else {
			st.Phase[i] = 0 // encircle
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].WOAPhase = st.Phase[i]
		ss.Bots[i].WOAFitness = fitToSensor(st.Fitness[i])
		dx := st.BestX - ss.Bots[i].X
		dy := st.BestY - ss.Bots[i].Y
		ss.Bots[i].WOABestDist = int(math.Sqrt(dx*dx + dy*dy))
	}
}

// ApplyWOA steers a bot using the Whale Optimization Algorithm.
func ApplyWOA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.WOA == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.WOA
	if idx >= len(st.Phase) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// The best whale keeps its natural behavior
	if idx == st.BestIdx {
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{0, 100, 255} // bright blue for best
		return
	}

	phase := st.Phase[idx]
	switch phase {
	case 0: // Encircling prey — move toward best position
		desired := math.Atan2(st.BestY-bot.Y, st.BestX-bot.X)
		steerToward(bot, desired, woaSteerRate)
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{0, 60, 180} // dark blue

	case 1: // Spiral (bubble-net) — logarithmic spiral around prey
		dx := st.BestX - bot.X
		dy := st.BestY - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 1.0 {
			dist = 1.0
		}
		// l ∈ [-1, 1] for spiral shape variation
		l := ss.Rng.Float64()*2.0 - 1.0
		// Spiral angle offset
		spiralAngle := math.Atan2(dy, dx) + math.Pi*2.0*l*0.1
		steerToward(bot, spiralAngle, woaSteerRate*1.5)
		bot.Speed = SwarmBotSpeed * (0.8 + 0.4*math.Exp(woaSpiralB*l)*math.Cos(2*math.Pi*l))
		if bot.Speed > SwarmBotSpeed*2.0 {
			bot.Speed = SwarmBotSpeed * 2.0
		}
		if bot.Speed < SwarmBotSpeed*0.5 {
			bot.Speed = SwarmBotSpeed * 0.5
		}
		bot.LEDColor = [3]uint8{0, 200, 255} // cyan for spiral

	case 2: // Search for prey — move toward a random whale
		randIdx := ss.Rng.Intn(len(ss.Bots))
		target := &ss.Bots[randIdx]
		desired := math.Atan2(target.Y-bot.Y, target.X-bot.X)
		steerToward(bot, desired, woaSteerRate)
		bot.Speed = SwarmBotSpeed * 1.2
		bot.LEDColor = [3]uint8{100, 150, 255} // light blue for search
	}
}
