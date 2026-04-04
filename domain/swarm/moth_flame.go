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
	mfoSpiralB     = 1.0   // spiral shape constant
	mfoMaxFlames   = 12    // maximum number of flames (was 10)
	mfoFlameDecay  = 0.998 // flame intensity decay per tick
	mfoMergeDist   = 900   // flame merge distance squared (30px)
	mfoMaxTicks    = 3000  // total ticks for adaptive t-range
	mfoFlameMinFit = 25.0  // minimum fitness to create a flame
	mfoEliteRate   = 0.2   // fraction of moths assigned to best flame
	mfoSpeedMult   = 5.0   // movement speed multiplier (7.5 px/tick)

	// Grid rescan parameters
	mfoGridRescanRate = 250 // periodic grid rescan every N ticks
	mfoGridRescanSize = 14  // grid resolution (14×14 = 196 samples)
	mfoGridInjectTop  = AlgoGridInjectTop // top grid points injected into worst moths

	// Direct-to-Best parameters
	mfoDirectStartProgress = 0.25 // start Direct-to-Best at 25% progress
	mfoDirectMaxProb       = 0.65 // max probability of Direct-to-Best
	mfoDirectJitter        = 7.5  // jitter around GlobalBest for Direct-to-Best

	// Global-Best attraction
	mfoGBWeightMin = 0.05 // initial GB attraction weight
	mfoGBWeightMax = 0.60 // final GB attraction weight
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
	TargetX      []float64    // target X per bot
	TargetY      []float64    // target Y per bot
	IsDirect     []bool       // whether bot is in Direct-to-Best mode
	Tick         int          // iteration counter
	BestF        float64      // per-tick best fitness
	BestX        float64      // per-tick best X
	BestY        float64      // per-tick best Y
	GlobalBestF  float64      // persistent global best fitness
	GlobalBestX  float64      // persistent global best X
	GlobalBestY  float64      // persistent global best Y
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
		TargetX:    make([]float64, n),
		TargetY:    make([]float64, n),
		IsDirect:   make([]bool, n),
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
		st.TargetX = append(st.TargetX, 0)
		st.TargetY = append(st.TargetY, 0)
		st.IsDirect = append(st.IsDirect, false)
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
		}
		if f > st.GlobalBestF {
			st.GlobalBestF = f
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
			improved = true
		}
	}
	// Track stagnation
	if improved {
		st.StagnCounter = 0
		st.LastBestF = st.GlobalBestF
	} else {
		st.StagnCounter++
	}

	n := len(ss.Bots)
	progress := float64(st.Tick) / float64(mfoMaxTicks)
	if progress > 1.0 {
		progress = 1.0
	}

	// Periodic grid rescan — systematic landscape sampling
	if st.Tick > 0 && st.Tick%mfoGridRescanRate == 0 && n > 0 {
		mfoGridRescan(ss, st)
	}

	// Best-bot local random walk around GlobalBest
	if st.GlobalBestF > 0 && n > 0 {
		bestIdx := 0
		bestDist := math.MaxFloat64
		for i := range ss.Bots {
			dx := ss.Bots[i].X - st.GlobalBestX
			dy := ss.Bots[i].Y - st.GlobalBestY
			d := dx*dx + dy*dy
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		rx := st.GlobalBestX + (ss.Rng.Float64()-0.5)*80
		ry := st.GlobalBestY + (ss.Rng.Float64()-0.5)*80
		st.TargetX[bestIdx] = rx
		st.TargetY[bestIdx] = ry
		st.IsDirect[bestIdx] = true
		// Evaluate the random walk point
		rf := distanceFitness(&SwarmBot{X: rx, Y: ry}, ss)
		if rf > st.GlobalBestF {
			st.GlobalBestF = rf
			st.GlobalBestX = rx
			st.GlobalBestY = ry
		}
	}

	// Update flames: add current best positions, merge nearby, cull weak
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
	tMax := 1.0 - 2.0*progress // tMax goes from 1.0 → -1.0

	// Stagnation-triggered flame injection
	if st.StagnCounter > 40 && numFlames >= 2 && st.StagnCounter%20 == 0 {
		rx := SwarmEdgeMargin + ss.Rng.Float64()*(ss.ArenaW-2*SwarmEdgeMargin)
		ry := SwarmEdgeMargin + ss.Rng.Float64()*(ss.ArenaH-2*SwarmEdgeMargin)
		rf := distanceFitness(&SwarmBot{X: rx, Y: ry}, ss)
		st.Flames[numFlames-1] = FlamePoint{X: rx, Y: ry, Fitness: rf, Intensity: 1.0}
		if rf > st.GlobalBestF {
			st.GlobalBestF = rf
			st.GlobalBestX = rx
			st.GlobalBestY = ry
		}
	}

	// GB attraction weight (linear ramp)
	gbWeight := mfoGBWeightMin + (mfoGBWeightMax-mfoGBWeightMin)*progress

	// Compute targets for each moth
	for i := range ss.Bots {
		st.IsDirect[i] = false

		// Direct-to-Best: skip flame dynamics and go straight to GlobalBest
		if progress > mfoDirectStartProgress && st.GlobalBestF > 0 {
			directProb := mfoDirectMaxProb * (progress - mfoDirectStartProgress) / (1.0 - mfoDirectStartProgress)
			if ss.Rng.Float64() < directProb {
				tX := st.GlobalBestX + (ss.Rng.Float64()*2-1)*mfoDirectJitter
				tY := st.GlobalBestY + (ss.Rng.Float64()*2-1)*mfoDirectJitter
				st.TargetX[i] = tX
				st.TargetY[i] = tY
				st.IsDirect[i] = true
				// Evaluate and update GlobalBest
				rf := distanceFitness(&SwarmBot{X: tX, Y: tY}, ss)
				if rf > st.GlobalBestF {
					st.GlobalBestF = rf
					st.GlobalBestX = tX
					st.GlobalBestY = tY
				}
				continue
			}
		}

		// Flame-based spiral movement
		var targetX, targetY float64

		if numFlames == 0 {
			st.MothFlame[i] = -1
			// Random exploration
			angle := ss.Rng.Float64() * 2 * math.Pi
			targetX = ss.Bots[i].X + math.Cos(angle)*SwarmBotSpeed*mfoSpeedMult
			targetY = ss.Bots[i].Y + math.Sin(angle)*SwarmBotSpeed*mfoSpeedMult
		} else {
			// Assign flame
			if ss.Rng.Float64() < mfoEliteRate {
				st.MothFlame[i] = 0
			} else {
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

			// Spiral parameter
			st.SpiralT[i] = -1.0 + ss.Rng.Float64()*(tMax+1.0)
			t := st.SpiralT[i]

			flameIdx := st.MothFlame[i]
			if flameIdx >= 0 && flameIdx < len(st.Flames) {
				flame := st.Flames[flameIdx]
				dx := flame.X - ss.Bots[i].X
				dy := flame.Y - ss.Bots[i].Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < 1.0 {
					dist = 1.0
				}

				angleToFlame := math.Atan2(dy, dx)
				spiralOffset := 2 * math.Pi * t * 0.2
				desired := angleToFlame + spiralOffset
				if dist < 12 {
					desired = angleToFlame
				}

				stepDist := SwarmBotSpeed * mfoSpeedMult * (0.8 + 0.4*math.Abs(math.Cos(2*math.Pi*t)))
				targetX = ss.Bots[i].X + math.Cos(desired)*stepDist
				targetY = ss.Bots[i].Y + math.Sin(desired)*stepDist
			} else {
				angle := ss.Rng.Float64() * 2 * math.Pi
				targetX = ss.Bots[i].X + math.Cos(angle)*SwarmBotSpeed*mfoSpeedMult
				targetY = ss.Bots[i].Y + math.Sin(angle)*SwarmBotSpeed*mfoSpeedMult
			}
		}

		// Apply Global-Best attraction
		if st.GlobalBestF > 0 {
			targetX = targetX*(1-gbWeight) + st.GlobalBestX*gbWeight
			targetY = targetY*(1-gbWeight) + st.GlobalBestY*gbWeight
		}

		st.TargetX[i] = targetX
		st.TargetY[i] = targetY
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
		if dx*dx+dy*dy < mfoMergeDist {
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

// ApplyMFO moves a moth toward its computed target using algoMovBot.
func ApplyMFO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.MFO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.MFO
	if idx >= len(st.TargetX) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// LED color
	if st.IsDirect[idx] {
		bot.LEDColor = [3]uint8{0, 255, 0} // green for Direct-to-Best
	} else {
		flameIdx := st.MothFlame[idx]
		if flameIdx == 0 {
			bot.LEDColor = [3]uint8{255, 215, 0} // gold for best flame
		} else if flameIdx >= 0 && flameIdx < len(st.Flames) {
			intensity := uint8(math.Min(255, st.Flames[flameIdx].Intensity*255))
			bot.LEDColor = [3]uint8{255, intensity, 30}
		} else {
			bot.LEDColor = [3]uint8{80, 80, 80}
		}
	}

	algoMovBot(bot, st.TargetX[idx], st.TargetY[idx], ss.ArenaW, ss.ArenaH, mfoSpeedMult)
}

// mfoGridRescan evaluates a grid of points across the arena and teleports
// the worst moths to the best-discovered grid positions.
func mfoGridRescan(ss *SwarmState, st *MFOState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	if usableW <= 0 || usableH <= 0 {
		return
	}
	gridPts := make([]gridPt, 0, mfoGridRescanSize*mfoGridRescanSize)
	for gx := 0; gx < mfoGridRescanSize; gx++ {
		for gy := 0; gy < mfoGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(mfoGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(mfoGridRescanSize)
			x += (ss.Rng.Float64()*2.0 - 1.0) * usableW * 0.02
			y += (ss.Rng.Float64()*2.0 - 1.0) * usableH * 0.02
			f := distanceFitness(&SwarmBot{X: x, Y: y}, ss)
			gridPts = append(gridPts, gridPt{x: x, y: y, f: f})
			if f > st.GlobalBestF {
				st.GlobalBestF = f
				st.GlobalBestX = x
				st.GlobalBestY = y
			}
		}
	}
	// Sort grid points descending by fitness
	for i := 1; i < len(gridPts); i++ {
		for j := i; j > 0 && gridPts[j].f > gridPts[j-1].f; j-- {
			gridPts[j], gridPts[j-1] = gridPts[j-1], gridPts[j]
		}
	}
	// Inject top grid points into worst moths
	n := len(ss.Bots)
	if n == 0 {
		return
	}
	// Find worst moths by fitness
	worst := make([]idxFit, n)
	for i := range ss.Bots {
		worst[i] = idxFit{i, st.BotFitness[i]}
	}
	// Sort ascending by fitness (worst first)
	for i := 1; i < len(worst); i++ {
		for j := i; j > 0 && worst[j].f < worst[j-1].f; j-- {
			worst[j], worst[j-1] = worst[j-1], worst[j]
		}
	}
	inject := mfoGridInjectTop
	if inject > len(gridPts) {
		inject = len(gridPts)
	}
	if inject > n {
		inject = n
	}
	for k := 0; k < inject; k++ {
		bi := worst[k].idx
		jx := gridPts[k].x + (ss.Rng.Float64()*2-1)*5
		jy := gridPts[k].y + (ss.Rng.Float64()*2-1)*5
		ss.Bots[bi].X = jx
		ss.Bots[bi].Y = jy
		st.BotFitness[bi] = gridPts[k].f
		st.TargetX[bi] = jx
		st.TargetY[bi] = jy
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
