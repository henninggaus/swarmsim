package swarm

import "math"

// Moth-Flame Optimization (MFO): Inspired by the navigation method of moths
// in nature called transverse orientation. Moths maintain a fixed angle with
// respect to distant light sources (the moon) for straight-line navigation.
// When attracted to artificial lights (flames), this mechanism causes them
// to fly in logarithmic spirals around the light.
//
// In MFO, moths spiral around "flames" (best positions found so far).
// As the algorithm progresses, the number of flames decreases, focusing
// the search on the best solutions.
//
// Reference: Mirjalili, S. (2015)
//            "Moth-flame optimization algorithm", Knowledge-Based Systems.

const (
	mfoRadius      = 80.0  // flame detection radius
	mfoSteerRate   = 0.25  // max steering per tick (was 0.14)
	mfoSpiralB     = 1.0   // spiral shape constant
	mfoMaxFlames   = 12    // maximum number of flames (was 10)
	mfoFlameDecay  = 0.998 // flame intensity decay per tick
	mfoMergeDist   = 900   // flame merge distance squared (30px, was 20px)
	mfoMaxTicks    = 3000  // total ticks for adaptive t-range
	mfoFlameMinFit = 25.0  // minimum fitness to create a flame (avoid bad local optima)
	mfoEliteRate   = 0.2   // fraction of moths assigned to best flame instead of nearest
)

// FlamePoint represents a flame (best-known position) in MFO.
type FlamePoint struct {
	X, Y      float64 // flame position
	Fitness   float64 // flame fitness value
	Intensity float64 // flame brightness (decays over time)
}

// MFOState holds Moth-Flame Optimization state.
type MFOState struct {
	Flames       []FlamePoint // sorted by fitness (best first)
	MothFlame    []int        // index of assigned flame per bot (-1 = none)
	SpiralT      []float64    // spiral parameter t per bot ∈ [-1, 1]
	BotFitness   []float64    // current fitness per bot
	Tick         int          // iteration counter
	BestF        float64      // global best fitness
	BestX        float64      // global best X
	BestY        float64      // global best Y
	StagnCounter int          // ticks since last best improvement
	LastBestF    float64      // best fitness at last improvement
}

// InitMFO allocates Moth-Flame Optimization state.
func InitMFO(ss *SwarmState) {
	n := len(ss.Bots)
	st := &MFOState{
		Flames:     make([]FlamePoint, 0, mfoMaxFlames),
		MothFlame:  make([]int, n),
		SpiralT:    make([]float64, n),
		BotFitness: make([]float64, n),
	}
	for i := range st.MothFlame {
		st.MothFlame[i] = -1
	}
	ss.MFO = st
	ss.MFOOn = true
}

// ClearMFO frees Moth-Flame Optimization state.
func ClearMFO(ss *SwarmState) {
	ss.MFO = nil
	ss.MFOOn = false
}

// TickMFO updates Moth-Flame Optimization for all bots.
func TickMFO(ss *SwarmState) {
	if ss.MFO == nil {
		return
	}
	st := ss.MFO
	st.Tick++

	// Grow slices if bots were added
	for len(st.MothFlame) < len(ss.Bots) {
		st.MothFlame = append(st.MothFlame, -1)
		st.SpiralT = append(st.SpiralT, 0)
		st.BotFitness = append(st.BotFitness, 0)
	}

	// Compute fitness using the shared fitness landscape.
	improved := false
	for i := range ss.Bots {
		f := distanceFitness(&ss.Bots[i], ss)
		st.BotFitness[i] = f
		if f > st.BestF {
			st.BestF = f
			st.BestX = ss.Bots[i].X
			st.BestY = ss.Bots[i].Y
			improved = true
		}
	}
	// Track stagnation
	if improved {
		st.StagnCounter = 0
		st.LastBestF = st.BestF
	} else {
		st.StagnCounter++
	}

	// Update flames: add current best positions, merge nearby, cull weak
	// Use higher threshold to avoid creating flames at poor local optima
	for i := range ss.Bots {
		if st.BotFitness[i] > mfoFlameMinFit {
			mfoAddFlame(st, ss.Bots[i].X, ss.Bots[i].Y, st.BotFitness[i])
		}
	}

	// Decay flame intensity and remove dead flames
	alive := st.Flames[:0]
	for i := range st.Flames {
		st.Flames[i].Intensity *= mfoFlameDecay
		if st.Flames[i].Intensity > 0.01 {
			alive = append(alive, st.Flames[i])
		}
	}
	st.Flames = alive

	// Sort flames by fitness (simple insertion sort for small slice)
	for i := 1; i < len(st.Flames); i++ {
		for j := i; j > 0 && st.Flames[j].Fitness > st.Flames[j-1].Fitness; j-- {
			st.Flames[j], st.Flames[j-1] = st.Flames[j-1], st.Flames[j]
		}
	}

	// Cap active flames
	numFlames := len(st.Flames)
	if numFlames > mfoMaxFlames {
		numFlames = mfoMaxFlames
	}

	// Adaptive t-range: t ∈ [-1, tMax] where tMax shrinks from 1 to -1 over time.
	// This is the key Mirjalili mechanism: early on moths orbit widely (t>0),
	// later they approach directly (t<0).
	progress := float64(st.Tick) / float64(mfoMaxTicks)
	if progress > 1.0 {
		progress = 1.0
	}
	tMax := 1.0 - 2.0*progress // tMax goes from 1.0 → -1.0

	// Stagnation-triggered flame injection: when stuck, replace the weakest
	// flame with one at a random position to guide moths to unexplored areas.
	// Moths stay in spiral mode (preserving avg) but explore new territory.
	if st.StagnCounter > 40 && numFlames >= 2 && st.StagnCounter%20 == 0 {
		// Replace last (worst) flame with a random position
		rx := SwarmEdgeMargin + ss.Rng.Float64()*(ss.ArenaW-2*SwarmEdgeMargin)
		ry := SwarmEdgeMargin + ss.Rng.Float64()*(ss.ArenaH-2*SwarmEdgeMargin)
		rf := distanceFitness(&SwarmBot{X: rx, Y: ry}, ss)
		st.Flames[numFlames-1] = FlamePoint{X: rx, Y: ry, Fitness: rf, Intensity: 1.0}
	}

	if numFlames == 0 {
		for i := range ss.Bots {
			st.MothFlame[i] = -1
		}
	} else {
		for i := range ss.Bots {
			if ss.Rng.Float64() < mfoEliteRate {
				// Elite moth: assigned to best flame (index 0)
				st.MothFlame[i] = 0
			} else {
				// Standard moth: assign to nearest flame
				bestDist := math.MaxFloat64
				bestF := -1
				for f := 0; f < numFlames; f++ {
					dx := st.Flames[f].X - ss.Bots[i].X
					dy := st.Flames[f].Y - ss.Bots[i].Y
					d := dx*dx + dy*dy
					if d < bestDist {
						bestDist = d
						bestF = f
					}
				}
				st.MothFlame[i] = bestF
			}
			// Randomize spiral parameter t ∈ [-1, tMax] (range shrinks over time)
			st.SpiralT[i] = -1.0 + ss.Rng.Float64()*(tMax+1.0)
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].MFOFlame = st.MothFlame[i]
		ss.Bots[i].MFOFitness = fitToSensor(st.BotFitness[i])
		if st.MothFlame[i] >= 0 && st.MothFlame[i] < len(st.Flames) {
			f := st.Flames[st.MothFlame[i]]
			dx := f.X - ss.Bots[i].X
			dy := f.Y - ss.Bots[i].Y
			ss.Bots[i].MFOFlameDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].MFOFlameDist = 9999
		}
	}
}

// mfoAddFlame adds a new flame or strengthens an existing nearby flame.
func mfoAddFlame(st *MFOState, x, y, fitness float64) {
	// Merge with nearby existing flame
	for i := range st.Flames {
		dx := st.Flames[i].X - x
		dy := st.Flames[i].Y - y
		if dx*dx+dy*dy < mfoMergeDist { // within 30px
			if fitness > st.Flames[i].Fitness {
				st.Flames[i].X = x
				st.Flames[i].Y = y
				st.Flames[i].Fitness = fitness
			}
			st.Flames[i].Intensity = 1.0
			return
		}
	}

	// Add new flame if under limit
	if len(st.Flames) < mfoMaxFlames*2 {
		st.Flames = append(st.Flames, FlamePoint{
			X: x, Y: y,
			Fitness:   fitness,
			Intensity: 1.0,
		})
	}
}

// ApplyMFO steers a moth in a logarithmic spiral around its assigned flame.
func ApplyMFO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.MFO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.MFO
	if idx >= len(st.MothFlame) {
		bot.Speed = SwarmBotSpeed
		return
	}

	flameIdx := st.MothFlame[idx]
	if flameIdx < 0 || flameIdx >= len(st.Flames) {
		// No flame assigned — random exploration with pull toward best
		if ss.Rng.Float64() < 0.1 {
			bot.Angle += (ss.Rng.Float64() - 0.5) * math.Pi
		}
		if st.BestF > 0 {
			desired := math.Atan2(st.BestY-bot.Y, st.BestX-bot.X)
			steerToward(bot, desired, mfoSteerRate*0.15)
		}
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{80, 80, 80}
		mfoMovBot(bot, ss)
		return
	}

	flame := st.Flames[flameIdx]
	dx := flame.X - bot.X
	dy := flame.Y - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1.0 {
		dist = 1.0
	}

	// Logarithmic spiral (Mirjalili 2015).
	// t < 0 → moth approaches flame; t > 0 → moth orbits flame.
	// As tMax shrinks over time, more moths approach directly.
	t := st.SpiralT[idx]
	angleToFlame := math.Atan2(dy, dx)

	// Spiral offset: t=0 → direct approach, |t|=1 → full orbit
	spiralOffset := 2 * math.Pi * t * 0.2
	desired := angleToFlame + spiralOffset

	// Close to flame: steer directly for precision
	if dist < 12 {
		desired = angleToFlame
	}

	steerToward(bot, desired, mfoSteerRate)

	// Speed varies with spiral phase
	bot.Speed = SwarmBotSpeed * (0.8 + 0.4*math.Abs(math.Cos(2*math.Pi*t)))

	// LED: orange-yellow, gold for best flame
	intensity := uint8(math.Min(255, flame.Intensity*255))
	if flameIdx == 0 {
		bot.LEDColor = [3]uint8{255, 215, 0}
	} else {
		bot.LEDColor = [3]uint8{255, intensity, 30}
	}

	mfoMovBot(bot, ss)
}

// mfoMovBot applies bot movement based on speed/angle and clamps to arena bounds.
// We move the bot directly here and then zero out Speed so that the GUI physics
// step (applySwarmPhysics) does not duplicate the movement.
func mfoMovBot(bot *SwarmBot, ss *SwarmState) {
	if bot.Speed > 0 {
		bot.X += math.Cos(bot.Angle) * bot.Speed
		bot.Y += math.Sin(bot.Angle) * bot.Speed
		bot.Speed = 0 // prevent double-move in GUI physics step
	}
	// Clamp to arena
	if bot.X < SwarmEdgeMargin {
		bot.X = SwarmEdgeMargin
	}
	if bot.X > ss.ArenaW-SwarmEdgeMargin {
		bot.X = ss.ArenaW - SwarmEdgeMargin
	}
	if bot.Y < SwarmEdgeMargin {
		bot.Y = SwarmEdgeMargin
	}
	if bot.Y > ss.ArenaH-SwarmEdgeMargin {
		bot.Y = ss.ArenaH - SwarmEdgeMargin
	}
}

// sign returns -1, 0, or 1 for the sign of a float.
func sign(x float64) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
