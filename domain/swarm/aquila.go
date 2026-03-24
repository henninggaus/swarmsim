package swarm

import "math"

// Aquila Optimizer (AO): Meta-heuristic inspired by the hunting behaviours
// of Aquila eagles. The algorithm models four distinct strategies that
// transition from exploration to exploitation as the hunt progresses:
//
//  1. High soar with vertical stoop (expanded exploration) — eagles soar
//     at high altitude scanning a wide area, then perform steep dives.
//  2. Contour flight with short glide (narrowed exploration) — low-altitude
//     flight following terrain contours with short gliding bursts.
//  3. Low flight with slow descent (expanded exploitation) — eagles descend
//     slowly toward prey, gradually tightening their search area.
//  4. Walk and grab prey (narrowed exploitation) — final precise approach,
//     eagles walk on the ground to grab the prey.
//
// The transition between phases is governed by a parameter t/T where T is
// the maximum number of ticks. At t/T < 2/3, exploration dominates; beyond
// that, exploitation takes over. Within each regime, a random threshold
// determines the specific sub-strategy.
//
// Reference: Abualigah, L., Yousri, D., Abd Elaziz, M., Ewees, A.A.,
//
//	Al-qaness, M.A.A. & Gandomi, A.H. (2021)
//	"Aquila Optimizer: A novel meta-heuristic optimization algorithm",
//	Computers & Industrial Engineering.
const (
	aoMaxTicks  = 3000  // full hunt cycle (matches benchmark length)
	aoSteerRate = 0.25  // max steering change per tick (radians)
	aoLevyBeta  = 1.5   // Lévy flight exponent for high soar
	aoAlpha     = 0.1   // exploitation step scaling
	aoDelta     = 0.1   // walk-and-grab random walk scale
	aoSpeedMult = 3.0   // movement speed multiplier
)

// AOState holds Aquila Optimizer state for the swarm.
type AOState struct {
	Fitness      []float64 // current fitness per eagle
	Phase        []int     // 0=high soar, 1=contour, 2=low flight, 3=walk&grab
	HuntTick     int       // ticks into current hunt cycle
	BestIdx      int       // index of best eagle (prey location)
	CurBestIdx   int       // current tick best (for LED display)
	BestX        float64   // best position X
	BestY        float64   // best position Y
	BestF        float64   // best fitness found (persistent)
	GlobalBestF  float64   // persistent global best fitness
	GlobalBestX  float64   // persistent global best X
	GlobalBestY  float64   // persistent global best Y
	MeanX        float64   // swarm mean X (used for contour flight)
	MeanY        float64   // swarm mean Y
}

// InitAO allocates Aquila Optimizer state for all bots.
func InitAO(ss *SwarmState) {
	n := len(ss.Bots)
	ss.AO = &AOState{
		Fitness:     make([]float64, n),
		Phase:       make([]int, n),
		BestIdx:     -1,
		CurBestIdx:  -1,
		BestF:       -1e18,
		GlobalBestF: -1e18,
	}
	ss.AOOn = true
}

// ClearAO frees Aquila Optimizer state.
func ClearAO(ss *SwarmState) {
	ss.AO = nil
	ss.AOOn = false
}

// TickAO updates the Aquila Optimizer for all bots.
func TickAO(ss *SwarmState) {
	if ss.AO == nil {
		return
	}
	st := ss.AO
	n := len(ss.Bots)

	// Grow slices if bots were added
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.Phase = append(st.Phase, 0)
	}

	st.HuntTick++
	if st.HuntTick > aoMaxTicks {
		st.HuntTick = 1
		// Do NOT reset BestF — persistent global best
	}

	// Compute fitness and swarm mean
	st.MeanX, st.MeanY = 0, 0
	curBestF := -1e18
	st.CurBestIdx = -1
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
		st.MeanX += ss.Bots[i].X
		st.MeanY += ss.Bots[i].Y
		if st.Fitness[i] > curBestF {
			curBestF = st.Fitness[i]
			st.CurBestIdx = i
		}
	}
	if n > 0 {
		st.MeanX /= float64(n)
		st.MeanY /= float64(n)
	}

	// Update persistent global best
	for i := range ss.Bots {
		if st.Fitness[i] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[i]
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
			st.BestIdx = i
		}
	}
	st.BestF = st.GlobalBestF
	st.BestX = st.GlobalBestX
	st.BestY = st.GlobalBestY

	// Phase assignment based on t/T ratio
	tRatio := float64(st.HuntTick) / float64(aoMaxTicks)

	for i := range ss.Bots {
		r := ss.Rng.Float64()
		if tRatio < 2.0/3.0 {
			// Exploration phase
			if r < 0.5 {
				st.Phase[i] = 0 // high soar with vertical stoop
			} else {
				st.Phase[i] = 1 // contour flight with short glide
			}
		} else {
			// Exploitation phase
			if r < 0.5 {
				st.Phase[i] = 2 // low flight with slow descent
			} else {
				st.Phase[i] = 3 // walk and grab prey
			}
		}

		// Update sensor cache
		ss.Bots[i].AOPhase = st.Phase[i]
		ss.Bots[i].AOFitness = int(st.Fitness[i])
		if st.BestIdx >= 0 {
			dx := st.BestX - ss.Bots[i].X
			dy := st.BestY - ss.Bots[i].Y
			ss.Bots[i].AOBestDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].AOBestDist = 9999
		}
	}
}

// aoMovBot moves a bot directly toward a target position.
func aoMovBot(bot *SwarmBot, ss *SwarmState, tx, ty float64) {
	dx := tx - bot.X
	dy := ty - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	maxStep := SwarmBotSpeed * aoSpeedMult
	if dist < 2.0 {
		bot.X = tx
		bot.Y = ty
	} else if dist <= maxStep {
		bot.X = tx
		bot.Y = ty
	} else {
		ratio := maxStep / dist
		bot.X += dx * ratio
		bot.Y += dy * ratio
	}

	// Arena clamping
	if bot.X < 0 {
		bot.X = 0
	} else if bot.X > ss.ArenaW {
		bot.X = ss.ArenaW
	}
	if bot.Y < 0 {
		bot.Y = 0
	} else if bot.Y > ss.ArenaH {
		bot.Y = ss.ArenaH
	}
	bot.Speed = 0
}

// ApplyAO steers an eagle according to its current hunting phase.
func ApplyAO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.AO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.AO
	if idx >= len(st.Phase) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Current tick best gets gold LED but still participates
	if idx == st.CurBestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0} // gold = current best
	}

	tRatio := float64(st.HuntTick) / float64(aoMaxTicks)
	var targetX, targetY float64

	switch st.Phase[idx] {
	case 0: // High soar with vertical stoop
		levy := aoLevyStep(ss)
		targetX = st.BestX*(1-tRatio) + (st.MeanX-st.BestX*ss.Rng.Float64()) + levy*20*ss.Rng.NormFloat64()
		targetY = st.BestY*(1-tRatio) + (st.MeanY-st.BestY*ss.Rng.Float64()) + levy*20*ss.Rng.NormFloat64()
		if idx != st.CurBestIdx {
			bot.LEDColor = [3]uint8{100, 180, 255} // sky blue = high soar
		}

	case 1: // Contour flight with short glide
		levy := aoLevyStep(ss)
		rIdx := ss.Rng.Intn(len(ss.Bots))
		theta := -math.Pi + ss.Rng.Float64()*2*math.Pi
		r := ss.Rng.Float64() * 50
		spiralX := r * math.Cos(theta)
		spiralY := r * math.Sin(theta)
		targetX = st.BestX + levy*math.Abs(st.BestX-bot.X) + spiralX + (ss.Bots[rIdx].X-bot.X)*ss.Rng.Float64()
		targetY = st.BestY + levy*math.Abs(st.BestY-bot.Y) + spiralY + (ss.Bots[rIdx].Y-bot.Y)*ss.Rng.Float64()
		if idx != st.CurBestIdx {
			bot.LEDColor = [3]uint8{50, 200, 100} // green = contour flight
		}

	case 2: // Low flight with slow descent
		targetX = (st.BestX-st.MeanX)*aoAlpha - ss.Rng.Float64()*math.Abs(st.BestX-bot.X) + bot.X
		targetY = (st.BestY-st.MeanY)*aoAlpha - ss.Rng.Float64()*math.Abs(st.BestY-bot.Y) + bot.Y
		targetX += (st.BestX - bot.X) * ss.Rng.Float64() * 0.3
		targetY += (st.BestY - bot.Y) * ss.Rng.Float64() * 0.3
		if idx != st.CurBestIdx {
			bot.LEDColor = [3]uint8{255, 140, 0} // orange = low flight
		}

	case 3: // Walk and grab prey
		QF := math.Pow(tRatio, 2.0) * 2 * ss.Rng.Float64()
		levy := aoLevyStep(ss)
		targetX = st.BestX - QF*math.Abs(st.BestX-st.MeanX)*ss.Rng.Float64() - levy*aoDelta*math.Abs(st.BestX-bot.X)
		targetY = st.BestY - QF*math.Abs(st.BestY-st.MeanY)*ss.Rng.Float64() - levy*aoDelta*math.Abs(st.BestY-bot.Y)
		if idx != st.CurBestIdx {
			bot.LEDColor = [3]uint8{255, 50, 50} // red = grab prey
		}
	}

	// Adaptive global-best attraction (5% -> 25% over aoMaxTicks)
	progress := float64(st.HuntTick) / float64(aoMaxTicks)
	if progress > 1 {
		progress = 1
	}
	gbWeight := 0.05 + 0.20*progress
	targetX = targetX*(1-gbWeight) + st.GlobalBestX*gbWeight
	targetY = targetY*(1-gbWeight) + st.GlobalBestY*gbWeight

	// Clamp to arena
	targetX = math.Max(10, math.Min(ss.ArenaW-10, targetX))
	targetY = math.Max(10, math.Min(ss.ArenaH-10, targetY))

	// Direct movement + steering for GUI mode
	desired := math.Atan2(targetY-bot.Y, targetX-bot.X)
	steerToward(bot, desired, aoSteerRate)
	aoMovBot(bot, ss, targetX, targetY)
}

// aoLevyStep generates a Lévy-flight step using the shared Mantegna algorithm.
func aoLevyStep(ss *SwarmState) float64 {
	return MantegnaLevy(ss.Rng, aoLevyBeta)
}
