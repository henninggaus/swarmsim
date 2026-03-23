package swarm

import "math"

// Harris Hawks Optimization (HHO): Meta-heuristic inspired by the cooperative
// hunting strategy of Harris's hawks. The algorithm models three phases:
//
//  1. Exploration — hawks perch randomly and search for prey using two
//     strategies: perch based on random tall trees or perch based on
//     the position of other family members (rabbit position).
//  2. Transition — controlled by "escaping energy" E that decreases from
//     2→0 over time. When |E|≥1 hawks explore; when |E|<1 they exploit.
//  3. Exploitation — four strategies based on the combination of:
//     - Soft/Hard besiege: |E|≥0.5 vs |E|<0.5
//     - Prey escapes or not: r<0.5 vs r≥0.5
//     Soft besiege: hawks surround prey and gradually tighten.
//     Hard besiege: hawks converge aggressively on prey.
//     Rapid dive: hawks perform Lévy-flight surprise attacks.
//
// Reference: Heidari, A.A. et al. (2019)
//
//	"Harris hawks optimization: Algorithm and applications",
//	Future Generation Computer Systems.
const (
	hhoMaxTicks  = 600   // full hunt cycle
	hhoSteerRate = 0.2   // max steering change per tick (radians)
	hhoLevyBeta  = 1.5   // Lévy flight exponent
)

// HHOState holds Harris Hawks Optimization state for the swarm.
type HHOState struct {
	Fitness  []float64 // current fitness per hawk
	Phase    []int     // 0=explore, 1=soft besiege, 2=hard besiege, 3=rapid dive
	HuntTick int       // ticks into current hunt cycle
	BestIdx  int       // index of rabbit (best hawk)
	BestX    float64   // rabbit position
	BestY    float64
	BestF    float64   // rabbit fitness
}

// InitHHO allocates Harris Hawks Optimization state for all bots.
func InitHHO(ss *SwarmState) {
	n := len(ss.Bots)
	ss.HHO = &HHOState{
		Fitness: make([]float64, n),
		Phase:   make([]int, n),
		BestIdx: -1,
		BestF:   -1e18,
	}
	ss.HHOOn = true
}

// ClearHHO frees Harris Hawks Optimization state.
func ClearHHO(ss *SwarmState) {
	ss.HHO = nil
	ss.HHOOn = false
}

// TickHHO updates the Harris Hawks Optimization for all bots.
func TickHHO(ss *SwarmState) {
	if ss.HHO == nil {
		return
	}
	st := ss.HHO

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		st.Fitness = append(st.Fitness, 0)
		st.Phase = append(st.Phase, 0)
	}

	st.HuntTick++
	if st.HuntTick > hhoMaxTicks {
		st.HuntTick = 1
		st.BestF = -1e18 // reset for new cycle
	}

	// Compute fitness for each hawk
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find rabbit (best fitness)
	for i := range ss.Bots {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
			st.BestX = ss.Bots[i].X
			st.BestY = ss.Bots[i].Y
		}
	}

	// Escaping energy: E = 2 * E0 * (1 - t/T), where E0 ∈ [-1, 1]
	// This linearly decreases the energy magnitude over the hunt cycle.
	// |E| ≥ 1 → exploration; |E| < 1 → exploitation
	tRatio := float64(st.HuntTick) / float64(hhoMaxTicks)

	// Assign phases and update sensor cache
	for i := range ss.Bots {
		E0 := 2*ss.Rng.Float64() - 1 // random in [-1, 1]
		E := 2 * E0 * (1 - tRatio)
		absE := math.Abs(E)

		if absE >= 1 {
			st.Phase[i] = 0 // exploration
		} else if absE >= 0.5 {
			st.Phase[i] = 1 // soft besiege
		} else {
			r := ss.Rng.Float64()
			if r >= 0.5 {
				st.Phase[i] = 2 // hard besiege
			} else {
				st.Phase[i] = 3 // rapid dive (Lévy flight)
			}
		}

		// Update sensor cache
		ss.Bots[i].HHOPhase = st.Phase[i]
		ss.Bots[i].HHOFitness = int(st.Fitness[i])
		if st.BestIdx >= 0 {
			dx := st.BestX - ss.Bots[i].X
			dy := st.BestY - ss.Bots[i].Y
			ss.Bots[i].HHOBestDist = int(math.Sqrt(dx*dx + dy*dy))
		} else {
			ss.Bots[i].HHOBestDist = 9999
		}
	}
}

// ApplyHHO steers a hawk according to its current hunting phase.
func ApplyHHO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.HHO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.HHO
	if idx >= len(st.Phase) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// The rabbit itself just moves normally
	if idx == st.BestIdx {
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{255, 215, 0} // gold = rabbit/prey
		return
	}

	tRatio := float64(st.HuntTick) / float64(hhoMaxTicks)
	E0 := 2*ss.Rng.Float64() - 1
	E := 2 * E0 * (1 - tRatio)

	var targetX, targetY float64

	switch st.Phase[idx] {
	case 0: // Exploration
		// Strategy: q < 0.5 → perch near random hawk, else near rabbit with random offset
		q := ss.Rng.Float64()
		if q < 0.5 {
			// Random hawk position with perturbation
			rIdx := ss.Rng.Intn(len(ss.Bots))
			targetX = ss.Bots[rIdx].X - ss.Rng.Float64()*math.Abs(ss.Bots[rIdx].X-2*ss.Rng.Float64()*bot.X)
			targetY = ss.Bots[rIdx].Y - ss.Rng.Float64()*math.Abs(ss.Bots[rIdx].Y-2*ss.Rng.Float64()*bot.Y)
		} else {
			// Rabbit position with random offset
			targetX = st.BestX - ss.Rng.Float64()*(ss.Rng.Float64()*ss.ArenaW*0.2)
			targetY = st.BestY - ss.Rng.Float64()*(ss.Rng.Float64()*ss.ArenaH*0.2)
		}
		bot.LEDColor = [3]uint8{80, 130, 200} // blue = exploring

	case 1: // Soft besiege
		// Hawks encircle prey with gradually tightening spiral
		J := 2 * (1 - ss.Rng.Float64()) // random jump strength
		dx := st.BestX - bot.X
		dy := st.BestY - bot.Y
		targetX = st.BestX - E*math.Abs(J*st.BestX-bot.X)
		targetY = st.BestY - E*math.Abs(J*st.BestY-bot.Y)
		// Blend with direct approach
		targetX = (targetX + st.BestX + dx*0.3) / 2
		targetY = (targetY + st.BestY + dy*0.3) / 2
		bot.LEDColor = [3]uint8{255, 165, 0} // orange = soft besiege

	case 2: // Hard besiege
		// Hawks converge directly on prey with tight radius
		targetX = st.BestX - E*math.Abs(st.BestX-bot.X)
		targetY = st.BestY - E*math.Abs(st.BestY-bot.Y)
		bot.LEDColor = [3]uint8{255, 50, 50} // red = hard besiege

	case 3: // Rapid dive (Lévy flight)
		// Surprise attack: Lévy flight toward prey
		levy := MantegnaLevy(ss.Rng, 1.5)
		dx := st.BestX - bot.X
		dy := st.BestY - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 1 {
			targetX = bot.X + (dx/dist)*levy*30
			targetY = bot.Y + (dy/dist)*levy*30
		} else {
			targetX = st.BestX
			targetY = st.BestY
		}
		bot.LEDColor = [3]uint8{200, 50, 200} // purple = rapid dive
	}

	// Clamp to arena
	targetX = math.Max(10, math.Min(ss.ArenaW-10, targetX))
	targetY = math.Max(10, math.Min(ss.ArenaH-10, targetY))

	// Steer toward target
	desired := math.Atan2(targetY-bot.Y, targetX-bot.X)
	steerToward(bot, desired, hhoSteerRate)
	bot.Speed = SwarmBotSpeed
}

// levyStep generates a Lévy-flight step using the shared Mantegna algorithm.
func levyStep(ss *SwarmState) float64 {
	return MantegnaLevy(ss.Rng, hhoLevyBeta)
}
