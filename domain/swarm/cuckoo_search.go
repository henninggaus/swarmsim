package swarm

import "math"

// Cuckoo Search Algorithm (CS): Meta-heuristic inspired by the brood
// parasitism of cuckoo birds combined with Lévy flight exploration.
// Each bot represents a "nest" with a solution (position). New solutions
// are generated via Lévy flights. If a cuckoo's egg (new solution) is
// better than a host nest, it replaces it. A fraction of the worst nests
// are abandoned each cycle and rebuilt at random positions.
//
// The algorithm balances exploration (Lévy flights produce occasional
// long jumps) with exploitation (keeping good solutions and replacing
// bad ones). This makes it particularly effective for multi-modal
// optimization landscapes.
//
// Three sensor values are exposed to SwarmScript:
//   - CuckooFitness: current nest quality (0-100)
//   - CuckooNestAge: ticks since last nest rebuild (0-100 capped)
//   - CuckooBest:    1 if this nest is in the top 25%, 0 otherwise
//
// Reference: Yang, X.-S. & Deb, S. (2009)
//
//	"Cuckoo Search via Lévy Flights", Proc. World Congress
//	on Nature & Biologically Inspired Computing (NaBIC).

const (
	csDiscoveryProb = 0.25  // probability of worst nests being abandoned (pa)
	csLevyAlpha     = 1.5   // Lévy exponent (1 < alpha <= 2)
	csStepScale     = 0.5   // scaling factor for Lévy step size
	csMaxCycle      = 400   // ticks per optimization cycle
	csSteerRate     = 0.12  // max steering change per tick (radians)
	csTopFraction   = 0.25  // fraction considered "best" nests
)

// CuckooState holds the state for Cuckoo Search optimization.
type CuckooState struct {
	Fitness  []float64 // current fitness per bot (nest quality)
	BestX    float64   // global best position X
	BestY    float64   // global best position Y
	BestF    float64   // global best fitness
	BestIdx  int       // index of global best nest
	NestAge  []int     // ticks since last rebuild per nest
	Cycle    int       // current tick in optimization cycle
	Rankings []int     // sorted indices by fitness (reused buffer)
}

// InitCuckoo allocates Cuckoo Search state for all bots.
func InitCuckoo(ss *SwarmState) {
	n := len(ss.Bots)
	ss.Cuckoo = &CuckooState{
		Fitness:  make([]float64, n),
		NestAge:  make([]int, n),
		Rankings: make([]int, n),
		BestIdx:  -1,
		BestF:    -1e9,
	}
	ss.CuckooOn = true
}

// ClearCuckoo frees Cuckoo Search state.
func ClearCuckoo(ss *SwarmState) {
	ss.Cuckoo = nil
	ss.CuckooOn = false
}

// TickCuckoo updates the Cuckoo Search algorithm for all bots.
// Computes fitness, updates global best, abandons worst nests,
// and writes sensor cache values for SwarmScript access.
func TickCuckoo(ss *SwarmState) {
	if ss.Cuckoo == nil {
		return
	}
	st := ss.Cuckoo
	n := len(ss.Bots)

	// Grow slices if bots were added after init
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.NestAge = append(st.NestAge, 0)
		st.Rankings = append(st.Rankings, len(st.Rankings))
	}

	st.Cycle++
	if st.Cycle > csMaxCycle {
		st.Cycle = 1
	}

	// Evaluate fitness: proximity to target (light or center) + neighbor density
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		st.Fitness[i] = cuckooFitness(bot, ss)
		st.NestAge[i]++
	}

	// Find global best
	st.BestIdx = 0
	st.BestF = st.Fitness[0]
	for i := 1; i < n; i++ {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
		}
	}
	st.BestX = ss.Bots[st.BestIdx].X
	st.BestY = ss.Bots[st.BestIdx].Y

	// Rank nests by fitness (simple insertion sort — fine for n <= 500)
	for i := 0; i < n; i++ {
		st.Rankings[i] = i
	}
	for i := 1; i < n; i++ {
		key := st.Rankings[i]
		j := i - 1
		for j >= 0 && st.Fitness[st.Rankings[j]] > st.Fitness[key] {
			st.Rankings[j+1] = st.Rankings[j]
			j--
		}
		st.Rankings[j+1] = key
	}

	// Abandon worst nests (discovery probability pa)
	abandonCount := int(float64(n) * csDiscoveryProb)
	if abandonCount < 1 {
		abandonCount = 1
	}
	// Worst nests are at the beginning of sorted rankings (ascending order)
	for k := 0; k < abandonCount && k < n; k++ {
		idx := st.Rankings[k]
		// Reset nest to random position within arena
		ss.Bots[idx].X = SpawnAreaMargin + ss.Rng.Float64()*(ss.ArenaW-2*SpawnAreaMargin)
		ss.Bots[idx].Y = SpawnAreaMargin + ss.Rng.Float64()*(ss.ArenaH-2*SpawnAreaMargin)
		ss.Bots[idx].Angle = ss.Rng.Float64() * 2 * math.Pi
		st.NestAge[idx] = 0
	}

	// Determine top fraction threshold
	topN := int(float64(n) * csTopFraction)
	if topN < 1 {
		topN = 1
	}
	topThreshold := -1e9
	if n-topN >= 0 && n-topN < n {
		topThreshold = st.Fitness[st.Rankings[n-topN]]
	}

	// Update sensor cache for SwarmScript
	for i := range ss.Bots {
		fitNorm := (st.Fitness[i] + 50) / 150.0 // normalize roughly to 0-1
		if fitNorm < 0 {
			fitNorm = 0
		}
		if fitNorm > 1 {
			fitNorm = 1
		}
		ss.Bots[i].CuckooFitness = int(fitNorm * 100)

		age := st.NestAge[i]
		if age > 100 {
			age = 100
		}
		ss.Bots[i].CuckooNestAge = age

		if st.Fitness[i] >= topThreshold {
			ss.Bots[i].CuckooBest = 1
		} else {
			ss.Bots[i].CuckooBest = 0
		}
	}
}

// ApplyCuckoo steers a bot using the Cuckoo Search algorithm.
// Good nests move toward the global best via small Lévy steps.
// Other nests perform larger Lévy flights for exploration.
func ApplyCuckoo(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Cuckoo == nil {
		bot.Speed = 0
		return
	}
	st := ss.Cuckoo
	if idx >= len(st.Fitness) {
		bot.Speed = 0
		return
	}

	// Global best does small random walk
	if idx == st.BestIdx {
		rx := bot.X + (ss.Rng.Float64()-0.5)*10
		ry := bot.Y + (ss.Rng.Float64()-0.5)*10
		algoMovBot(bot, rx, ry, ss.ArenaW, ss.ArenaH, 1.0)
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for best
		return
	}

	// Levy flight step toward global best with random perturbation.
	step := csStepScale * MantegnaLevy(ss.Rng, csLevyAlpha)

	// Compute target position using Levy flight
	dx := st.BestX - bot.X
	dy := st.BestY - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	var targetX, targetY float64
	if dist > 1.0 {
		baseAngle := math.Atan2(dy, dx)
		desired := baseAngle + step*0.3
		// Step distance proportional to Levy magnitude
		absStep := math.Abs(step)
		stepDist := SwarmBotSpeed * (1.0 + math.Min(absStep*0.5, 1.0)) * 3.0
		targetX = bot.X + math.Cos(desired)*stepDist
		targetY = bot.Y + math.Sin(desired)*stepDist
	} else {
		// Very close to best: random exploration
		angle := ss.Rng.Float64() * 2 * math.Pi
		targetX = bot.X + math.Cos(angle)*SwarmBotSpeed*3.0
		targetY = bot.Y + math.Sin(angle)*SwarmBotSpeed*3.0
	}

	// Move directly (eigenbewegung)
	algoMovBot(bot, targetX, targetY, ss.ArenaW, ss.ArenaH, 3.0)

	// LED color: green gradient by fitness, blue tint for older nests
	fitNorm := (st.Fitness[idx] + 50) / 150.0
	if fitNorm < 0 {
		fitNorm = 0
	}
	if fitNorm > 1 {
		fitNorm = 1
	}
	g := uint8(80 + fitNorm*175)
	b := uint8(math.Min(float64(st.NestAge[idx])*2.5, 200))
	bot.LEDColor = [3]uint8{30, g, b}
}

// cuckooFitness evaluates the fitness of a nest position using the shared
// Gaussian fitness landscape plus a small neighbor density bonus.
func cuckooFitness(bot *SwarmBot, ss *SwarmState) float64 {
	proxFit := distanceFitness(bot, ss)
	neighFit := math.Min(float64(bot.NeighborCount)/6.0, 1.0) * 20
	return proxFit + neighFit
}
