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
	scaMaxTicks  = 3000  // full optimization cycle (matches benchmark length)
	scaSteerRate = 0.25  // max steering change per tick (radians)
	scaAMax      = 2.0   // initial r1 upper bound (exploration range)
	scaAMin      = 0.15  // minimum r1 floor — keeps oscillation alive in late stages
	scaSpeedMult = 5.0   // movement speed multiplier (was 3.0)
	scaScoutRate = 100   // every N ticks, 20% of bots do random exploration
	scaScoutFrac = 0.20  // fraction of bots that scout
	scaLocalWalk = 40.0  // best bot local random walk radius
)

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

	// Best bot does local random walk to explore nearby peaks
	if idx == st.CurBestIdx {
		walkX := bot.X + (ss.Rng.Float64()*2-1)*scaLocalWalk
		walkY := bot.Y + (ss.Rng.Float64()*2-1)*scaLocalWalk
		algoMovBot(bot, walkX, walkY, ss.ArenaW, ss.ArenaH, scaSpeedMult)
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for best
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

	// Periodic random scouting: every scaScoutRate ticks, some bots explore randomly
	if st.Tick%scaScoutRate == 0 && ss.Rng.Float64() < scaScoutFrac {
		scoutX := ss.Rng.Float64() * ss.ArenaW
		scoutY := ss.Rng.Float64() * ss.ArenaH
		algoMovBot(bot, scoutX, scoutY, ss.ArenaW, ss.ArenaH, scaSpeedMult)
		bot.LEDColor = [3]uint8{0, 255, 0} // green for scouts
		return
	}

	// r1: linearly decreasing from scaAMax to scaAMin (floor keeps oscillation alive)
	r1 := scaAMax - (scaAMax-scaAMin)*progress
	if r1 < scaAMin {
		r1 = scaAMin
	}

	// r2: random in [0, 2π]
	r2 := ss.Rng.Float64() * 2 * math.Pi
	// r3: random in [0, 2] — random weight for destination
	r3 := ss.Rng.Float64() * 2.0
	// r4: random in [0, 1] — switches sine/cosine
	r4 := ss.Rng.Float64()

	// Use persistent global best as destination
	destX := st.GlobalBestX
	destY := st.GlobalBestY

	// Distance components to best position
	dx := r3*destX - bot.X
	dy := r3*destY - bot.Y

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

	// Adaptive global-best attraction: weight increases 5%→40% over time
	gbWeight := 0.05 + 0.35*progress
	targetX += (destX - targetX) * gbWeight
	targetY += (destY - targetY) * gbWeight

	// Move bot directly to target
	algoMovBot(bot, targetX, targetY, ss.ArenaW, ss.ArenaH, scaSpeedMult)

	// Also steer angle for GUI mode
	desired := math.Atan2(targetY-bot.Y, targetX-bot.X)
	steerToward(bot, desired, scaSteerRate)

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
