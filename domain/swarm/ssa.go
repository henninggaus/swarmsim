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
	ssaRadius       = 120.0 // neighbor detection radius
	ssaMaxTicks     = 3000  // full cycle length (matches benchmark duration)
	ssaSteerRate    = 0.25  // max steering per tick (radians)
	ssaSpeedMult    = 3.0   // movement speed multiplier for faster convergence
	ssaGBestWeightA = 0.05  // global-best attraction start weight
	ssaGBestWeightB = 0.25  // global-best attraction end weight
)

// SSAState holds Salp Swarm Algorithm state for the swarm.
type SSAState struct {
	Fitness    []float64 // current fitness per bot
	Role       []int     // 0=leader, 1=follower
	ChainIdx   []int     // index of salp ahead in chain (-1 for first leader)
	CycleTick  int       // ticks into current cycle
	FoodX      float64   // best-known food source position
	FoodY      float64
	FoodFit    float64 // best fitness value found
	BestIdx    int     // index of best salp
	CurBestIdx int     // current tick's best (for LED)
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

	// Compute fitness using the shared fitness landscape.
	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
	}

	// Find the food source (persistent global best)
	st.CurBestIdx = 0
	curBestF := st.Fitness[0]
	for i := range ss.Bots {
		if st.Fitness[i] > curBestF {
			curBestF = st.Fitness[i]
			st.CurBestIdx = i
		}
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
		ss.Bots[i].SSAFitness = fitToSensor(st.Fitness[i])
		dx := st.FoodX - ss.Bots[i].X
		dy := st.FoodY - ss.Bots[i].Y
		ss.Bots[i].SSAFoodDist = int(math.Sqrt(dx*dx + dy*dy))
	}
}

// ssaMovBot moves a bot directly via position updates and sets Speed=0
// to prevent double movement in GUI mode.
func ssaMovBot(bot *SwarmBot, ss *SwarmState, tx, ty float64) {
	dx := tx - bot.X
	dy := ty - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	maxStep := SwarmBotSpeed * ssaSpeedMult
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

	// Clamp to arena
	bot.X = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, bot.X))
	bot.Y = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, bot.Y))

	bot.Angle = math.Atan2(dy, dx)
	bot.Speed = 0 // prevent double movement in GUI mode
}

// ApplySSA moves a bot according to the Salp Swarm Algorithm.
// Bots move directly via position updates (bot.X/bot.Y) to work in both
// GUI and headless benchmark modes.
func ApplySSA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.SSA == nil {
		bot.Speed = 0
		return
	}
	st := ss.SSA
	if idx >= len(st.Role) {
		bot.Speed = 0
		return
	}

	// c1 = 2 * exp(-(4t/T)^2) — decays over cycle for exploration→exploitation
	t := float64(st.CycleTick)
	T := float64(ssaMaxTicks)
	c1 := 2.0 * math.Exp(-math.Pow(4.0*t/T, 2))

	// Adaptive global-best attraction: increases over time
	progress := t / T
	if progress > 1 {
		progress = 1
	}
	gbestW := ssaGBestWeightA + (ssaGBestWeightB-ssaGBestWeightA)*progress

	var targetX, targetY float64

	if st.Role[idx] == 0 {
		// === Leader behavior ===
		// Leaders oscillate around the food source position.
		c2 := ss.Rng.Float64() // [0,1]
		c3 := ss.Rng.Float64() // [0,1]

		targetX, targetY = st.FoodX, st.FoodY

		// Oscillation around food source
		amplitude := c1 * ss.ArenaW * 0.3
		if c3 < 0.5 {
			targetX += amplitude * c2
			targetY += amplitude * c2
		} else {
			targetX -= amplitude * c2
			targetY -= amplitude * c2
		}

		// Leader LED: cyan-teal gradient based on c1
		intensity := uint8(100 + c1*77)
		bot.LEDColor = [3]uint8{0, intensity, intensity}
	} else {
		// === Follower behavior ===
		// Followers track the salp ahead: x_new = 0.5 * (x_i + x_{i-1})
		prev := st.ChainIdx[idx]
		if prev < 0 || prev >= len(ss.Bots) {
			bot.Speed = 0
			bot.LEDColor = [3]uint8{60, 60, 80}
			return
		}

		targetX = 0.5 * (bot.X + ss.Bots[prev].X)
		targetY = 0.5 * (bot.Y + ss.Bots[prev].Y)

		// Follower LED: dim blue, brighter for higher chain position
		chainPos := float64(idx) / float64(len(ss.Bots))
		blue := uint8(80 + chainPos*100)
		bot.LEDColor = [3]uint8{30, 30, blue}
	}

	// Blend target toward global best (adaptive attraction)
	if st.FoodFit > -1e8 && gbestW > 0 {
		targetX = targetX*(1-gbestW) + st.FoodX*gbestW
		targetY = targetY*(1-gbestW) + st.FoodY*gbestW
	}

	// Clamp target to arena
	targetX = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, targetX))
	targetY = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, targetY))

	// Move directly to target
	ssaMovBot(bot, ss, targetX, targetY)

	// Best salp gets bright gold LED
	if idx == st.BestIdx {
		bot.LEDColor = [3]uint8{255, 200, 50}
	}
}
