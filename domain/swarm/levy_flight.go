package swarm

import "math"

// Lévy-Flight Foraging: Natural search strategy combining short random walks
// with rare long-distance jumps. Inspired by how animals forage in patchy
// environments. The heavy-tailed step distribution (Pareto/Cauchy) maximizes
// area coverage when resource locations are unknown.

const (
	levyMinStep     = 2.0   // minimum step length (pixels)
	levyMaxStep     = 300.0 // maximum step length (pixels)
	levyLongProb    = 0.05  // probability of a long jump per tick
	levyLongMin     = 80.0  // minimum long-jump distance
	levyShortMax    = 15.0  // maximum short-walk distance
	levySteerRate   = 0.12  // max angle change per tick during walk (radians)
	levyPhaseTicks  = 30    // ticks in a short walk phase
)

// LevyState holds per-bot Lévy flight state.
type LevyState struct {
	Phase     []int     // 0=idle, 1=short walk, 2=long jump
	StepLen   []float64 // remaining step distance
	TargetAng []float64 // target heading for current step
	Timer     []int     // ticks remaining in current phase
}

// InitLevy allocates Lévy flight state for all bots.
func InitLevy(ss *SwarmState) {
	n := len(ss.Bots)
	ss.Levy = &LevyState{
		Phase:     make([]int, n),
		StepLen:   make([]float64, n),
		TargetAng: make([]float64, n),
		Timer:     make([]int, n),
	}
	ss.LevyOn = true
}

// ClearLevy frees Lévy flight state.
func ClearLevy(ss *SwarmState) {
	ss.Levy = nil
	ss.LevyOn = false
}

// TickLevy updates Lévy flight sensor cache for all bots.
// Computes LevyPhase (0=idle, 1=short, 2=long) and LevyStep (remaining distance).
func TickLevy(ss *SwarmState) {
	if ss.Levy == nil {
		return
	}
	st := ss.Levy
	// Grow slices if bots were added
	for len(st.Phase) < len(ss.Bots) {
		st.Phase = append(st.Phase, 0)
		st.StepLen = append(st.StepLen, 0)
		st.TargetAng = append(st.TargetAng, 0)
		st.Timer = append(st.Timer, 0)
	}

	for i := range ss.Bots {
		// Decay timer
		if st.Timer[i] > 0 {
			st.Timer[i]--
		}
		if st.Timer[i] <= 0 {
			st.Phase[i] = 0
			st.StepLen[i] = 0
		}

		// Update sensor cache
		ss.Bots[i].LevyPhase = st.Phase[i]
		ss.Bots[i].LevyStep = int(st.StepLen[i])
	}
}

// ApplyLevyWalk executes a Lévy flight step for a bot.
// Randomly chooses short walk or long jump based on heavy-tailed distribution.
func ApplyLevyWalk(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Levy == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Levy
	if idx >= len(st.Phase) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// If idle, start a new step
	if st.Phase[idx] == 0 {
		r := ss.Rng.Float64()
		if r < levyLongProb {
			// Long jump (Lévy flight)
			st.Phase[idx] = 2
			st.StepLen[idx] = levyLongMin + ss.Rng.Float64()*(levyMaxStep-levyLongMin)
			st.TargetAng[idx] = ss.Rng.Float64() * 2 * math.Pi
			st.Timer[idx] = int(st.StepLen[idx] / (SwarmBotSpeed * 2)) + 1
		} else {
			// Short random walk
			st.Phase[idx] = 1
			st.StepLen[idx] = levyMinStep + ss.Rng.Float64()*levyShortMax
			st.TargetAng[idx] = bot.Angle + (ss.Rng.Float64()-0.5)*math.Pi*0.5
			st.Timer[idx] = levyPhaseTicks
		}
	}

	// Steer toward target angle
	diff := st.TargetAng[idx] - bot.Angle
	diff = WrapAngle(diff)
	if diff > levySteerRate {
		diff = levySteerRate
	} else if diff < -levySteerRate {
		diff = -levySteerRate
	}
	bot.Angle += diff

	// Move speed depends on phase
	if st.Phase[idx] == 2 {
		bot.Speed = SwarmBotSpeed * 2.0 // fast long jump
	} else {
		bot.Speed = SwarmBotSpeed
	}

	// Reduce remaining distance
	st.StepLen[idx] -= bot.Speed
	if st.StepLen[idx] <= 0 {
		st.StepLen[idx] = 0
		st.Phase[idx] = 0
		st.Timer[idx] = 0
	}
}
