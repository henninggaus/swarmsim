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
	mfoRadius     = 80.0  // flame detection radius
	mfoSteerRate  = 0.14  // max steering per tick
	mfoSpiralB    = 1.0   // spiral shape constant
	mfoMaxFlames  = 10    // maximum number of flames
	mfoFlameDecay = 0.998 // flame intensity decay per tick
)

// FlamePoint represents a flame (best-known position) in MFO.
type FlamePoint struct {
	X, Y      float64 // flame position
	Fitness   float64 // flame fitness value
	Intensity float64 // flame brightness (decays over time)
}

// MFOState holds Moth-Flame Optimization state.
type MFOState struct {
	Flames     []FlamePoint // sorted by fitness (best first)
	MothFlame  []int        // index of assigned flame per bot (-1 = none)
	SpiralT    []float64    // spiral parameter t per bot ∈ [-1, 1]
	BotFitness []float64    // current fitness per bot
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

	// Grow slices if bots were added
	for len(st.MothFlame) < len(ss.Bots) {
		st.MothFlame = append(st.MothFlame, -1)
		st.SpiralT = append(st.SpiralT, 0)
		st.BotFitness = append(st.BotFitness, 0)
	}

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.BotFitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Update flames: add current best positions, merge nearby, cull weak
	for i := range ss.Bots {
		if st.BotFitness[i] > 10 { // only good positions become flames
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

	// The number of active flames decreases linearly with iterations.
	// In our swarm context, we use a fraction of available flames.
	numFlames := len(st.Flames)
	if numFlames > mfoMaxFlames {
		numFlames = mfoMaxFlames
	}
	if numFlames == 0 {
		for i := range ss.Bots {
			st.MothFlame[i] = -1
		}
	} else {
		// Assign each moth to a flame (round-robin or nearest)
		for i := range ss.Bots {
			// Assign to nearest flame within radius
			bestDist := math.MaxFloat64
			bestF := -1
			for f := 0; f < numFlames; f++ {
				dx := st.Flames[f].X - ss.Bots[i].X
				dy := st.Flames[f].Y - ss.Bots[i].Y
				d := math.Sqrt(dx*dx + dy*dy)
				if d < bestDist {
					bestDist = d
					bestF = f
				}
			}
			st.MothFlame[i] = bestF
			// Randomize spiral parameter t ∈ [-1, 1]
			st.SpiralT[i] = ss.Rng.Float64()*2.0 - 1.0
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
		if dx*dx+dy*dy < 400 { // within 20px
			// Strengthen existing flame
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
		// No flame assigned — random exploration
		if ss.Rng.Float64() < 0.05 {
			bot.Angle += (ss.Rng.Float64() - 0.5) * math.Pi * 0.5
		}
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{80, 80, 80} // dim grey
		return
	}

	flame := st.Flames[flameIdx]
	dx := flame.X - bot.X
	dy := flame.Y - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1.0 {
		dist = 1.0
	}

	// Logarithmic spiral: position = dist * e^(bt) * cos(2πt) + flame
	t := st.SpiralT[idx]
	spiralAngle := math.Atan2(dy, dx) + 2*math.Pi*t*0.15
	spiralDist := dist * math.Exp(mfoSpiralB*t) * 0.02

	// Combine spiral direction with direct approach
	desired := spiralAngle
	if spiralDist > dist {
		// Close to flame: circle tighter
		desired = math.Atan2(dy, dx) + math.Pi*0.4*float64(sign(t))
	}

	steerToward(bot, desired, mfoSteerRate)

	// Speed varies with spiral phase
	bot.Speed = SwarmBotSpeed * (0.8 + 0.4*math.Abs(math.Cos(2*math.Pi*t)))

	// LED: orange-yellow (like a moth near flame)
	intensity := uint8(math.Min(255, flame.Intensity*255))
	bot.LEDColor = [3]uint8{255, intensity, 30}
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
