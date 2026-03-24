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
	woaMaxTicks    = 3000  // full benchmark length
	woaSteerRate   = 0.25  // max steering change per tick (radians)
	woaSpiralB     = 1.0   // spiral shape constant
	woaSpiralProb  = 0.5   // probability of spiral vs shrink
	woaSpeedMult   = 3.0   // movement speed multiplier for eigenbewegung
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

// ApplyWOA moves a bot using the Whale Optimization Algorithm via direct
// position updates (eigenbewegung). Bots move directly so they converge
// in both GUI and benchmark mode.
func ApplyWOA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.WOA == nil {
		bot.Speed = 0
		return
	}
	st := ss.WOA
	if idx >= len(st.Phase) {
		bot.Speed = 0
		return
	}

	// Progress for adaptive parameters
	progress := float64(st.HuntTick) / float64(woaMaxTicks)
	if progress > 1 {
		progress = 1
	}

	// The best whale does a small random walk to explore locally
	if idx == st.BestIdx {
		rx := bot.X + (ss.Rng.Float64()-0.5)*10
		ry := bot.Y + (ss.Rng.Float64()-0.5)*10
		algoMovBot(bot, rx, ry, ss.ArenaW, ss.ArenaH, 1.0)
		bot.LEDColor = [3]uint8{0, 100, 255} // bright blue for best
		return
	}

	var targetX, targetY float64

	phase := st.Phase[idx]
	switch phase {
	case 0: // Encircling prey — move toward best position
		targetX, targetY = st.BestX, st.BestY
		bot.LEDColor = [3]uint8{0, 60, 180} // dark blue

	case 1: // Spiral (bubble-net) — logarithmic spiral around prey
		dx := st.BestX - bot.X
		dy := st.BestY - bot.Y
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
		targetX = bot.X + math.Cos(spiralAngle)*spiralDist*0.3
		targetY = bot.Y + math.Sin(spiralAngle)*spiralDist*0.3
		bot.LEDColor = [3]uint8{0, 200, 255} // cyan for spiral

	case 2: // Search for prey — move toward a random whale
		randIdx := ss.Rng.Intn(len(ss.Bots))
		targetX = ss.Bots[randIdx].X
		targetY = ss.Bots[randIdx].Y
		bot.LEDColor = [3]uint8{100, 150, 255} // light blue for search
	}

	// Adaptive global-best attraction (5% → 25%)
	if st.GlobalBestF > -1e18 {
		gbWeight := 0.05 + 0.20*progress
		targetX = targetX*(1-gbWeight) + st.GlobalBestX*gbWeight
		targetY = targetY*(1-gbWeight) + st.GlobalBestY*gbWeight
	}

	algoMovBot(bot, targetX, targetY, ss.ArenaW, ss.ArenaH, woaSpeedMult)
}
