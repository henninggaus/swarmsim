package swarm

import "math"

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
// As r1 decreases, oscillations shrink and agents converge (exploitation).
// When r1 is large, agents explore widely via sine/cosine sweeps.
//
// Reference: Mirjalili, S. (2016)
//            "SCA: A Sine Cosine Algorithm for solving optimization problems",
//            Knowledge-Based Systems.

const (
	scaMaxTicks  = 600   // full optimization cycle
	scaSteerRate = 0.15  // max steering change per tick (radians)
	scaAMax      = 2.0   // initial r1 upper bound (exploration range)
)

// SCAState holds Sine Cosine Algorithm state for the swarm.
type SCAState struct {
	Fitness    []float64 // current fitness per bot
	BestX      float64   // global best position X
	BestY      float64   // global best position Y
	BestF      float64   // global best fitness
	BestIdx    int       // index of best bot
	Tick       int       // ticks into current cycle
	Phase      []int     // 0=sine, 1=cosine per bot (last used)
}

// InitSCA allocates Sine Cosine Algorithm state for all bots.
func InitSCA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.SCA = &SCAState{
		Fitness: make([]float64, n),
		Phase:   make([]int, n),
		BestF:   -1e18,
		BestIdx: -1,
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

	// Find global best
	st.BestIdx = -1
	st.BestF = -1e18
	for i := range ss.Bots {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
		}
	}
	if st.BestIdx >= 0 {
		st.BestX = ss.Bots[st.BestIdx].X
		st.BestY = ss.Bots[st.BestIdx].Y
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].SCAFitness = int(st.Fitness[i] * 100)
		if ss.Bots[i].SCAFitness > 100 {
			ss.Bots[i].SCAFitness = 100
		}
		ss.Bots[i].SCAPhase = st.Phase[i]
		if st.BestIdx >= 0 {
			dx := st.BestX - ss.Bots[i].X
			dy := st.BestY - ss.Bots[i].Y
			ss.Bots[i].SCABestDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].SCABestDist = 9999
		}
	}
}

// ApplySCA steers a bot according to the Sine Cosine Algorithm.
// The r1 parameter decreases linearly over the cycle, transitioning
// from wide sinusoidal exploration to tight convergence on the best position.
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

	// Best bot keeps natural behavior
	if idx == st.BestIdx {
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for best
		return
	}

	if st.BestIdx < 0 {
		bot.Speed = SwarmBotSpeed
		return
	}

	// r1: linearly decreasing from scaAMax to 0
	r1 := scaAMax * (1.0 - float64(st.Tick)/float64(scaMaxTicks))
	if r1 < 0 {
		r1 = 0
	}

	// r2: random in [0, 2π]
	r2 := ss.Rng.Float64() * 2 * math.Pi
	// r3: random in [0, 2] — random weight for destination
	r3 := ss.Rng.Float64() * 2.0
	// r4: random in [0, 1] — switches sine/cosine
	r4 := ss.Rng.Float64()

	// Distance components to best position
	dx := r3*st.BestX - bot.X
	dy := r3*st.BestY - bot.Y

	var offsetX, offsetY float64
	if r4 < 0.5 {
		// Sine phase (exploration)
		offsetX = r1 * math.Sin(r2) * math.Abs(dx)
		offsetY = r1 * math.Sin(r2) * math.Abs(dy)
		st.Phase[idx] = 0
	} else {
		// Cosine phase (exploitation)
		offsetX = r1 * math.Cos(r2) * math.Abs(dx)
		offsetY = r1 * math.Cos(r2) * math.Abs(dy)
		st.Phase[idx] = 1
	}

	// Target position
	targetX := bot.X + offsetX
	targetY := bot.Y + offsetY

	// Steer toward target
	desired := math.Atan2(targetY-bot.Y, targetX-bot.X)
	steerToward(bot, desired, scaSteerRate)
	bot.Speed = SwarmBotSpeed

	// LED color: sine=cyan oscillation, cosine=magenta oscillation
	// Brightness scales with r1 (brighter during exploration)
	intensity := uint8(100 + r1/scaAMax*155)
	if st.Phase[idx] == 0 {
		// Sine: cyan tones
		bot.LEDColor = [3]uint8{0, intensity, intensity}
	} else {
		// Cosine: magenta tones
		bot.LEDColor = [3]uint8{intensity, 0, intensity}
	}
}
