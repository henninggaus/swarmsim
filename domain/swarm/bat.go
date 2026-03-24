package swarm

import "math"

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
	batFMin      = 0.0   // minimum frequency
	batFMax      = 2.0   // maximum frequency
	batAlpha     = 0.95  // loudness decay rate (0 < alpha < 1)
	batGamma     = 0.9   // pulse rate increase coefficient
	batSteerRate = 0.25  // max steering change per tick (radians)
	batMaxSpeed  = 2.5   // max bot speed under BA
	batLocalStep = 15.0  // local random walk step size
	batSpeedMult = 3.0   // movement speed multiplier for eigenbewegung
	batMaxTicks  = 3000  // full benchmark length
)

// BatState holds per-bot echolocation state for the Bat Algorithm.
type BatState struct {
	Freq     []float64 // frequency per bat
	Vel      [2][]float64 // velocity components [0]=X, [1]=Y
	Loud     []float64 // loudness per bat (decreases over time)
	Pulse    []float64 // pulse emission rate per bat (increases over time)
	Fitness  []float64 // fitness per bat
	BestX    float64   // current tick best position X
	BestY    float64   // current tick best position Y
	BestF    float64   // current tick best fitness
	BestIdx  int       // index of best bat
	Tick     int       // iteration counter
	AvgLoud  float64   // precomputed average loudness (avoids O(n²) in ApplyBat)
	// Persistent global best (never reset)
	GlobalBestF float64
	GlobalBestX float64
	GlobalBestY float64
	// Personal best tracking: each bat remembers its own best position.
	PBestX   []float64
	PBestY   []float64
	PBestF   []float64
}

// batMovBot moves a bot directly toward a target position.
// Sets Speed=0 afterward to prevent double-movement in GUI mode.
func batMovBot(bot *SwarmBot, targetX, targetY, arenaW, arenaH float64) {
	dx := targetX - bot.X
	dy := targetY - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 2 {
		bot.X = targetX
		bot.Y = targetY
		bot.Speed = 0
		return
	}
	maxStep := SwarmBotSpeed * batSpeedMult
	if dist <= maxStep {
		bot.X = targetX
		bot.Y = targetY
	} else {
		ratio := maxStep / dist
		bot.X += dx * ratio
		bot.Y += dy * ratio
	}
	// Clamp to arena
	if bot.X < SwarmBotRadius {
		bot.X = SwarmBotRadius
	}
	if bot.X > arenaW-SwarmBotRadius {
		bot.X = arenaW - SwarmBotRadius
	}
	if bot.Y < SwarmBotRadius {
		bot.Y = SwarmBotRadius
	}
	if bot.Y > arenaH-SwarmBotRadius {
		bot.Y = arenaH - SwarmBotRadius
	}
	bot.Angle = math.Atan2(dy, dx)
	bot.Speed = 0
}

// InitBat allocates Bat Algorithm state.
func InitBat(ss *SwarmState) {
	n := len(ss.Bots)
	st := &BatState{
		Freq:    make([]float64, n),
		Vel:     [2][]float64{make([]float64, n), make([]float64, n)},
		Loud:    make([]float64, n),
		Pulse:   make([]float64, n),
		Fitness: make([]float64, n),
		PBestX:  make([]float64, n),
		PBestY:  make([]float64, n),
		PBestF:  make([]float64, n),
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
	// Weight increases from 5% to 25% over batMaxTicks
	if st.GlobalBestF > -1e18 {
		gbWeight := 0.05 + 0.20*progress
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
		batMovBot(bot, newX, newY, ss.ArenaW, ss.ArenaH)

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
