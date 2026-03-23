package swarm

import "math"

// Salp Swarm Algorithm (SSA): Meta-heuristic inspired by the swarming and
// chaining behavior of salps — barrel-shaped planktonic tunicates that form
// long chains in the ocean. Leaders at the front of the chain navigate toward
// the food source (best fitness position), while followers trail behind their
// predecessor, creating emergent chain formations.
//
// Key mechanisms:
//   - The population is split into leaders (first half) and followers.
//   - Leaders oscillate around the food source using a parameter c1 that
//     decays over the hunt cycle, transitioning from exploration → exploitation.
//   - Followers track the position of the salp ahead of them in the chain,
//     producing emergent trail-following behavior.
//
// Reference: Mirjalili, S. et al. (2017)
//
//	"Salp Swarm Algorithm: A bio-inspired optimizer for engineering
//	 design problems", Advances in Engineering Software.
const (
	ssaRadius    = 120.0 // neighbor detection radius
	ssaMaxTicks  = 800   // full cycle length
	ssaSteerRate = 0.18  // max steering per tick (radians)
)

// SSAState holds Salp Swarm Algorithm state for the swarm.
type SSAState struct {
	Fitness  []float64 // current fitness per bot
	Role     []int     // 0=leader, 1=follower
	ChainIdx []int     // index of salp ahead in chain (-1 for first leader)
	CycleTick int      // ticks into current cycle
	FoodX    float64   // best-known food source position
	FoodY    float64
	FoodFit  float64   // best fitness value found
	BestIdx  int       // index of best salp
}

// InitSSA allocates Salp Swarm Algorithm state.
func InitSSA(ss *SwarmState) {
	n := len(ss.Bots)
	half := n / 2
	if half < 1 {
		half = 1
	}

	st := &SSAState{
		Fitness:  make([]float64, n),
		Role:     make([]int, n),
		ChainIdx: make([]int, n),
		FoodFit:  -1e9,
		BestIdx:  0,
	}

	// First half = leaders, second half = followers
	for i := 0; i < n; i++ {
		if i < half {
			st.Role[i] = 0 // leader
			st.ChainIdx[i] = -1
		} else {
			st.Role[i] = 1 // follower
			st.ChainIdx[i] = i - 1 // follows the salp ahead
		}
	}

	ss.SSA = st
	ss.SSAOn = true
}

// ClearSSA frees Salp Swarm Algorithm state.
func ClearSSA(ss *SwarmState) {
	ss.SSA = nil
	ss.SSAOn = false
}

// TickSSA updates the Salp Swarm Algorithm for all bots.
func TickSSA(ss *SwarmState) {
	if ss.SSA == nil {
		return
	}
	st := ss.SSA

	// Grow slices if bots were added
	for len(st.Fitness) < len(ss.Bots) {
		st.Fitness = append(st.Fitness, 0)
		st.Role = append(st.Role, 1)
		st.ChainIdx = append(st.ChainIdx, len(st.Fitness)-2)
	}

	st.CycleTick++
	if st.CycleTick > ssaMaxTicks {
		st.CycleTick = 1
	}

	n := len(ss.Bots)
	half := n / 2
	if half < 1 {
		half = 1
	}

	// Compute fitness for each salp using shared landscape
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

	// Find the food source (global best)
	for i := range ss.Bots {
		if st.Fitness[i] > st.FoodFit {
			st.FoodFit = st.Fitness[i]
			st.FoodX = ss.Bots[i].X
			st.FoodY = ss.Bots[i].Y
			st.BestIdx = i
		}
	}

	// Update roles (first half = leaders, rest = followers with chain)
	for i := 0; i < n; i++ {
		if i < half {
			st.Role[i] = 0
			st.ChainIdx[i] = -1
		} else {
			st.Role[i] = 1
			prev := i - 1
			if prev < 0 {
				prev = 0
			}
			st.ChainIdx[i] = prev
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].SSARole = st.Role[i]
		ss.Bots[i].SSAFitness = int(st.Fitness[i] * 100)
		dx := st.FoodX - ss.Bots[i].X
		dy := st.FoodY - ss.Bots[i].Y
		ss.Bots[i].SSAFoodDist = int(math.Sqrt(dx*dx + dy*dy))
	}
}

// ApplySSA steers a bot according to the Salp Swarm Algorithm.
func ApplySSA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.SSA == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.SSA
	if idx >= len(st.Role) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// c1 = 2 * exp(-(4t/T)^2) — decays over cycle for exploration→exploitation
	t := float64(st.CycleTick)
	T := float64(ssaMaxTicks)
	c1 := 2.0 * math.Exp(-math.Pow(4.0*t/T, 2))

	if st.Role[idx] == 0 {
		// === Leader behavior ===
		// Leaders oscillate around the food source position.
		// x_new = Food + c1 * ((ub - lb) * c2 + lb) where c2, c3 are random
		c2 := ss.Rng.Float64() // [0,1]
		c3 := ss.Rng.Float64() // [0,1]

		targetX, targetY := st.FoodX, st.FoodY

		// Oscillation around food source
		amplitude := c1 * ss.ArenaW * 0.3
		if c3 < 0.5 {
			targetX += amplitude * c2
			targetY += amplitude * c2
		} else {
			targetX -= amplitude * c2
			targetY -= amplitude * c2
		}

		// Clamp to arena
		targetX = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, targetX))
		targetY = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, targetY))

		desired := math.Atan2(targetY-bot.Y, targetX-bot.X)
		steerToward(bot, desired, ssaSteerRate)
		bot.Speed = SwarmBotSpeed

		// Leader LED: cyan-teal gradient based on c1
		intensity := uint8(100 + c1*77)
		bot.LEDColor = [3]uint8{0, intensity, intensity}
	} else {
		// === Follower behavior ===
		// Followers track the salp ahead: x_new = 0.5 * (x_i + x_{i-1})
		prev := st.ChainIdx[idx]
		if prev < 0 || prev >= len(ss.Bots) {
			bot.Speed = SwarmBotSpeed
			bot.LEDColor = [3]uint8{60, 60, 80}
			return
		}

		targetX := 0.5 * (bot.X + ss.Bots[prev].X)
		targetY := 0.5 * (bot.Y + ss.Bots[prev].Y)

		desired := math.Atan2(targetY-bot.Y, targetX-bot.X)
		steerToward(bot, desired, ssaSteerRate*0.8) // followers steer slightly slower
		bot.Speed = SwarmBotSpeed * 0.9

		// Follower LED: dim blue, brighter for higher chain position
		chainPos := float64(idx) / float64(len(ss.Bots))
		blue := uint8(80 + chainPos*100)
		bot.LEDColor = [3]uint8{30, 30, blue}
	}

	// Best salp gets bright gold LED
	if idx == st.BestIdx {
		bot.LEDColor = [3]uint8{255, 200, 50}
	}
}
