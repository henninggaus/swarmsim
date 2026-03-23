package swarm

import "math"

// Dragonfly Algorithm (DA): Swarm intelligence metaheuristic inspired by the
// static (feeding) and dynamic (migratory) swarming behaviour of dragonflies.
//
// Each dragonfly adjusts its position using five behavioural vectors:
//   s = separation   — avoid collisions with neighbours
//   a = alignment    — match velocity of neighbours
//   c = cohesion     — fly toward centre of neighbourhood
//   f = food attraction — steer toward the best-known food source
//   e = enemy distraction — flee from the worst-known position
//
// Step vector:  ΔX = (s·ws + a·wa + c·wc + f·wf + e·we) + w·ΔX(t-1)
// Position:     X(t+1) = X(t) + ΔX(t)
//
// Exploration weights (s,a high) linearly transition to exploitation weights
// (f,c high) over the optimisation cycle, mimicking the shift from dynamic
// (migratory) to static (feeding) swarms.
//
// When no neighbours exist, a dragonfly performs a Lévy flight for global
// exploration — the heavy tail produces occasional long jumps that prevent
// stagnation in local optima.
//
// Reference: Mirjalili, S. (2016)
//            "Dragonfly algorithm: a new meta-heuristic optimization technique
//             for solving single-objective, discrete, and multi-objective problems",
//            Neural Computing and Applications.

const (
	daMaxTicks   = 600  // full optimisation cycle
	daSteerRate  = 0.2  // max steering change per tick (radians)
	daNeighDist  = 80.0 // neighbourhood radius
)

// DAState holds Dragonfly Algorithm state for the swarm.
type DAState struct {
	Fitness  []float64 // current fitness per bot
	StepX    []float64 // step vector X per bot (velocity carry-over)
	StepY    []float64 // step vector Y per bot
	BestX    float64   // food source (global best position) X
	BestY    float64   // food source (global best position) Y
	BestF    float64   // global best fitness
	BestIdx  int       // index of best bot
	WorstX   float64   // enemy (global worst position) X
	WorstY   float64   // enemy (global worst position) Y
	WorstF   float64   // global worst fitness
	WorstIdx int       // index of worst bot
	Tick     int       // ticks into current cycle
	Role     []int     // 0=static(feeding), 1=dynamic(migratory), 2=levy per bot
}

// InitDA allocates Dragonfly Algorithm state for all bots.
func InitDA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.DA = &DAState{
		Fitness:  make([]float64, n),
		StepX:    make([]float64, n),
		StepY:    make([]float64, n),
		Role:     make([]int, n),
		BestF:    -1e18,
		BestIdx:  -1,
		WorstF:   1e18,
		WorstIdx: -1,
	}
	ss.DAOn = true
}

// ClearDA frees Dragonfly Algorithm state.
func ClearDA(ss *SwarmState) {
	ss.DA = nil
	ss.DAOn = false
}

// TickDA updates the Dragonfly Algorithm for all bots.
func TickDA(ss *SwarmState) {
	if ss.DA == nil {
		return
	}
	st := ss.DA

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		st.Fitness = append(st.Fitness, 0)
		st.StepX = append(st.StepX, 0)
		st.StepY = append(st.StepY, 0)
		st.Role = append(st.Role, 0)
	}

	st.Tick++
	if st.Tick > daMaxTicks {
		st.Tick = 1
	}

	// Compute fitness using shared landscape
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		neighbourFit := float64(bot.NeighborCount) / 10.0
		if neighbourFit > 1.0 {
			neighbourFit = 1.0
		}
		carryFit := 0.0
		if bot.CarryingPkg >= 0 {
			carryFit = 0.3
		}
		landFit := distanceFitness(bot, ss) / 100.0
		if landFit < 0 {
			landFit = 0
		}
		st.Fitness[i] = neighbourFit*0.4 + carryFit + landFit*0.3
	}

	// Find global best (food) and worst (enemy)
	st.BestIdx = -1
	st.BestF = -1e18
	st.WorstIdx = -1
	st.WorstF = 1e18
	for i := range ss.Bots {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
		}
		if st.Fitness[i] < st.WorstF {
			st.WorstF = st.Fitness[i]
			st.WorstIdx = i
		}
	}
	if st.BestIdx >= 0 {
		st.BestX = ss.Bots[st.BestIdx].X
		st.BestY = ss.Bots[st.BestIdx].Y
	}
	if st.WorstIdx >= 0 {
		st.WorstX = ss.Bots[st.WorstIdx].X
		st.WorstY = ss.Bots[st.WorstIdx].Y
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].DAFitness = int(st.Fitness[i] * 100)
		if ss.Bots[i].DAFitness > 100 {
			ss.Bots[i].DAFitness = 100
		}
		ss.Bots[i].DARole = st.Role[i]
		if st.BestIdx >= 0 {
			dx := st.BestX - ss.Bots[i].X
			dy := st.BestY - ss.Bots[i].Y
			ss.Bots[i].DAFoodDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].DAFoodDist = 9999
		}
	}
}

// ApplyDA steers a bot according to the Dragonfly Algorithm.
func ApplyDA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.DA == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.DA
	if idx >= len(st.Fitness) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Best bot keeps natural behaviour
	if idx == st.BestIdx {
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for food source
		st.Role[idx] = 0
		return
	}

	if st.BestIdx < 0 {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Adaptive weights: exploration (s,a high) → exploitation (f,c high)
	t := float64(st.Tick) / float64(daMaxTicks)
	ws := 2.0 * (1.0 - t) // separation weight: 2→0
	wa := 2.0 * (1.0 - t) // alignment weight: 2→0
	wc := 2.0 * t          // cohesion weight: 0→2
	wf := 2.0 * t          // food attraction: 0→2
	we := 1.0 - t           // enemy distraction: 1→0
	w := 0.9 - 0.5*t        // inertia weight: 0.9→0.4

	// Find neighbours within daNeighDist using spatial hash
	candidates := ss.Hash.Query(bot.X, bot.Y, daNeighDist)

	sepX, sepY := 0.0, 0.0
	aliX, aliY := 0.0, 0.0
	cohX, cohY := 0.0, 0.0
	neighCount := 0

	for _, j := range candidates {
		if j == idx {
			continue
		}
		dx := ss.Bots[j].X - bot.X
		dy := ss.Bots[j].Y - bot.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d > daNeighDist || d < 0.001 {
			continue
		}
		neighCount++

		// Separation: avoid neighbours
		sepX -= dx / d
		sepY -= dy / d

		// Alignment: match velocity direction
		aliX += math.Cos(ss.Bots[j].Angle)
		aliY += math.Sin(ss.Bots[j].Angle)

		// Cohesion: toward centre of neighbourhood
		cohX += dx
		cohY += dy
	}

	if neighCount > 0 {
		// Normalise
		n := float64(neighCount)
		sepX /= n
		sepY /= n
		aliX /= n
		aliY /= n
		cohX /= n
		cohY /= n

		// Food attraction vector
		foodX := (st.BestX - bot.X)
		foodY := (st.BestY - bot.Y)
		foodD := math.Sqrt(foodX*foodX + foodY*foodY)
		if foodD > 0 {
			foodX /= foodD
			foodY /= foodD
		}

		// Enemy distraction vector (flee from worst)
		enemyX := bot.X - st.WorstX
		enemyY := bot.Y - st.WorstY
		enemyD := math.Sqrt(enemyX*enemyX + enemyY*enemyY)
		if enemyD > 0 {
			enemyX /= enemyD
			enemyY /= enemyD
		}

		// Compute step vector with inertia
		st.StepX[idx] = w*st.StepX[idx] + ws*sepX + wa*aliX + wc*cohX + wf*foodX + we*enemyX
		st.StepY[idx] = w*st.StepY[idx] + ws*sepY + wa*aliY + wc*cohY + wf*foodY + we*enemyY

		// Role: early = dynamic/migratory, late = static/feeding
		if t < 0.5 {
			st.Role[idx] = 1 // dynamic
		} else {
			st.Role[idx] = 0 // static
		}
	} else {
		// No neighbours: Lévy flight for exploration
		st.Role[idx] = 2 // levy

		step := MantegnaLevy(ss.Rng, 1.5)

		levyAngle := ss.Rng.Float64() * 2 * math.Pi
		st.StepX[idx] = step * math.Cos(levyAngle) * 3.0
		st.StepY[idx] = step * math.Sin(levyAngle) * 3.0
	}

	// Clamp step magnitude
	mag := math.Sqrt(st.StepX[idx]*st.StepX[idx] + st.StepY[idx]*st.StepY[idx])
	maxStep := SwarmBotSpeed * 2.0
	if mag > maxStep {
		st.StepX[idx] = st.StepX[idx] / mag * maxStep
		st.StepY[idx] = st.StepY[idx] / mag * maxStep
	}

	// Steer toward step direction
	if mag > 0.01 {
		desired := math.Atan2(st.StepY[idx], st.StepX[idx])
		steerToward(bot, desired, daSteerRate)
	}
	bot.Speed = SwarmBotSpeed

	// LED colour by role
	switch st.Role[idx] {
	case 0: // static/feeding — green
		intensity := uint8(100 + t*155)
		bot.LEDColor = [3]uint8{0, intensity, 50}
	case 1: // dynamic/migratory — blue
		intensity := uint8(100 + (1-t)*155)
		bot.LEDColor = [3]uint8{50, 50, intensity}
	case 2: // lévy flight — magenta
		bot.LEDColor = [3]uint8{200, 0, 200}
	}
}
