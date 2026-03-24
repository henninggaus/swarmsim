package swarm

import "math"

// Artificial Bee Colony (ABC): A swarm intelligence algorithm inspired by
// the foraging behaviour of honey bee colonies. The colony consists of three
// groups of bees:
//
//   Employed bees  — exploit known food sources and share fitness via waggle dance
//   Onlooker bees  — select food sources probabilistically based on reported fitness
//   Scout bees     — abandon exhausted sources and explore random new positions
//
// Each food source represents a candidate solution. Employed bees perform
// local search around their assigned source. Onlooker bees are recruited to
// better sources with probability proportional to fitness (roulette-wheel
// selection). If a source is not improved for 'limit' consecutive trials,
// the employed bee becomes a scout and the source is replaced randomly.
//
// The ABC algorithm balances exploration (scouts) and exploitation (employed
// + onlooker phases) naturally through the abandonment limit mechanism.
//
// Reference: Karaboga, D. (2005)
//
//	"An Idea Based on Honey Bee Swarm for Numerical Optimization",
//	Technical Report TR06, Erciyes University.

const (
	abcSteerRate     = 0.14  // max steering change per tick (radians)
	abcAbandonLimit  = 60    // ticks without improvement before source is abandoned
	abcLocalStep     = 40.0  // local search perturbation radius
	abcScoutSpeed    = 1.5   // speed multiplier for scouting bees
	abcOnlookerRatio = 0.5   // fraction of colony acting as onlookers
	abcSpeedMult     = 5.0   // movement speed multiplier (7.5 px/tick)
	abcMaxTicks      = 3000  // optimization cycle length (matches benchmark)
	abcArrivalDist   = 5.0   // distance threshold for arrival at target
)

// ABCState holds Artificial Bee Colony state for the swarm.
type ABCState struct {
	// Per-bee state (indexed by bot index).
	Fitness []float64 // fitness of each food source
	TrialX  []float64 // trial (neighbor) position X
	TrialY  []float64 // trial (neighbor) position Y
	Stale   []int     // ticks since last improvement for each source
	Role    []int     // 0=employed, 1=onlooker, 2=scout

	// Global tracking (per-tick best).
	BestIdx int     // index of best food source this tick
	BestF   float64 // best fitness found this tick
	BestX   float64 // best position this tick
	BestY   float64

	// Persistent global best (never reset).
	GlobalBestF   float64
	GlobalBestX   float64
	GlobalBestY   float64
	GlobalBestIdx int

	Tick int // current tick counter
}

// InitABC allocates Artificial Bee Colony state for all bots.
// Half the bots are designated as employed bees (each owns a food source),
// and the other half act as onlookers that probabilistically select sources.
func InitABC(ss *SwarmState) {
	n := len(ss.Bots)
	st := &ABCState{
		Fitness:       make([]float64, n),
		TrialX:        make([]float64, n),
		TrialY:        make([]float64, n),
		Stale:         make([]int, n),
		Role:          make([]int, n),
		BestF:         -1e9,
		BestIdx:       -1,
		GlobalBestF:   -1e9,
		GlobalBestIdx: -1,
	}

	// Assign initial roles: first half employed, second half onlooker.
	onlookerStart := int(float64(n) * abcOnlookerRatio)
	for i := 0; i < n; i++ {
		if i >= onlookerStart {
			st.Role[i] = 1 // onlooker
		}
		// Evaluate initial fitness.
		st.Fitness[i] = abcFitness(&ss.Bots[i], ss)
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
			st.BestX = ss.Bots[i].X
			st.BestY = ss.Bots[i].Y
		}
		if st.Fitness[i] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[i]
			st.GlobalBestIdx = i
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
		}
	}

	ss.ABC = st
	ss.ABCOn = true
}

// ClearABC frees Artificial Bee Colony state.
func ClearABC(ss *SwarmState) {
	ss.ABC = nil
	ss.ABCOn = false
}

// TickABC runs one tick of the Artificial Bee Colony algorithm.
//
// Phase 1 — Employed bees: each employed bee generates a neighbor solution
// near its current food source using a random perturbation and a partner's
// position. If the neighbor is better, the source is updated (greedy selection).
//
// Phase 2 — Onlooker bees: each onlooker selects an employed bee's source
// with probability proportional to its fitness (roulette-wheel), then performs
// a similar local search around that source.
//
// Phase 3 — Scout bees: any source that has not been improved for
// abcAbandonLimit ticks is abandoned. The corresponding bee teleports to a
// random position and becomes a scout for one tick.
func TickABC(ss *SwarmState) {
	if ss.ABC == nil {
		return
	}
	st := ss.ABC
	n := len(ss.Bots)
	st.Tick++

	// Grow slices if bots were added.
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.TrialX = append(st.TrialX, 0)
		st.TrialY = append(st.TrialY, 0)
		st.Stale = append(st.Stale, 0)
		st.Role = append(st.Role, 0)
	}

	// Re-evaluate fitness for all bees.
	for i := range ss.Bots {
		st.Fitness[i] = abcFitness(&ss.Bots[i], ss)
	}

	// Adaptive Global-Best attraction weight: 5% → 30% over abcMaxTicks.
	progress := float64(st.Tick) / float64(abcMaxTicks)
	if progress > 1 {
		progress = 1
	}
	gbWeight := 0.05 + 0.25*progress

	// ── Phase 1: Employed bees — local search ──
	for i := 0; i < n; i++ {
		if st.Role[i] != 0 { // only employed bees
			continue
		}
		// Pick a random partner k ≠ i.
		k := i
		for k == i {
			k = ss.Rng.Intn(n)
		}

		// Generate neighbor: x_new = x_i + phi * (x_i - x_k)
		phi := (ss.Rng.Float64()*2.0 - 1.0) // phi ∈ [-1, 1]
		tx := ss.Bots[i].X + phi*(ss.Bots[i].X-ss.Bots[k].X)
		ty := ss.Bots[i].Y + phi*(ss.Bots[i].Y-ss.Bots[k].Y)

		// Shift toward GlobalBest.
		if st.GlobalBestIdx >= 0 {
			tx += gbWeight * (st.GlobalBestX - tx)
			ty += gbWeight * (st.GlobalBestY - ty)
		}

		// Clamp to arena.
		tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
		ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))

		st.TrialX[i] = tx
		st.TrialY[i] = ty
	}

	// ── Phase 2: Onlooker bees — probabilistic selection ──
	// Build roulette wheel from employed fitness values.
	totalFit := 0.0
	for i := 0; i < n; i++ {
		if st.Role[i] == 0 {
			// Shift fitness to be positive for roulette selection.
			f := st.Fitness[i] + 100.0
			if f < 0.01 {
				f = 0.01
			}
			totalFit += f
		}
	}

	for i := 0; i < n; i++ {
		if st.Role[i] != 1 { // only onlooker bees
			continue
		}

		// Roulette-wheel selection: pick an employed bee's source.
		spin := ss.Rng.Float64() * totalFit
		cumul := 0.0
		selected := 0
		for j := 0; j < n; j++ {
			if st.Role[j] != 0 {
				continue
			}
			f := st.Fitness[j] + 100.0
			if f < 0.01 {
				f = 0.01
			}
			cumul += f
			if cumul >= spin {
				selected = j
				break
			}
		}

		// Perform local search around the selected source.
		phi := (ss.Rng.Float64()*2.0 - 1.0)
		k := i
		for k == i {
			k = ss.Rng.Intn(n)
		}
		tx := ss.Bots[selected].X + phi*(ss.Bots[selected].X-ss.Bots[k].X)
		ty := ss.Bots[selected].Y + phi*(ss.Bots[selected].Y-ss.Bots[k].Y)

		// Shift toward GlobalBest.
		if st.GlobalBestIdx >= 0 {
			tx += gbWeight * (st.GlobalBestX - tx)
			ty += gbWeight * (st.GlobalBestY - ty)
		}

		tx = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaW-SwarmEdgeMargin, tx))
		ty = math.Max(SwarmEdgeMargin, math.Min(ss.ArenaH-SwarmEdgeMargin, ty))

		st.TrialX[i] = tx
		st.TrialY[i] = ty
	}

	// ── Phase 3: Scout bees — abandon exhausted sources ──
	for i := 0; i < n; i++ {
		if st.Role[i] == 0 {
			st.Stale[i]++
		}
		if st.Stale[i] >= abcAbandonLimit {
			// Abandon source: random new position.
			st.Role[i] = 2 // scout
			st.Stale[i] = 0
			margin := SwarmEdgeMargin
			st.TrialX[i] = margin + ss.Rng.Float64()*(ss.ArenaW-2*margin)
			st.TrialY[i] = margin + ss.Rng.Float64()*(ss.ArenaH-2*margin)
		}
	}

	// Update per-tick best and persistent GlobalBest.
	st.BestF = -1e9
	for i := range ss.Bots {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
			st.BestX = ss.Bots[i].X
			st.BestY = ss.Bots[i].Y
		}
		if st.Fitness[i] > st.GlobalBestF {
			st.GlobalBestF = st.Fitness[i]
			st.GlobalBestIdx = i
			st.GlobalBestX = ss.Bots[i].X
			st.GlobalBestY = ss.Bots[i].Y
		}
	}

	// Update sensor cache.
	for i := range ss.Bots {
		ss.Bots[i].ABCFitness = int(st.Fitness[i] * 100)
		ss.Bots[i].ABCRole = st.Role[i]
		if st.GlobalBestIdx >= 0 && st.GlobalBestIdx < n {
			dx := st.GlobalBestX - ss.Bots[i].X
			dy := st.GlobalBestY - ss.Bots[i].Y
			ss.Bots[i].ABCBestDist = int(math.Sqrt(dx*dx + dy*dy))
		}
	}
}

// ApplyABC steers a bot according to its ABC role.
//
// Employed bees steer toward their trial (neighbor) position and perform
// greedy selection when they arrive. Onlooker bees steer toward the trial
// position of their selected employed source. Scout bees move quickly to
// their randomly assigned new position, then revert to employed status.
func ApplyABC(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.ABC == nil {
		bot.Speed = 0
		return
	}
	st := ss.ABC
	if idx >= len(st.Role) {
		bot.Speed = 0
		return
	}

	// Move directly toward trial position (eigenbewegung)
	algoMovBot(bot, st.TrialX[idx], st.TrialY[idx], ss.ArenaW, ss.ArenaH, abcSpeedMult)

	// Check arrival — evaluate fitness at new position
	dx := st.TrialX[idx] - bot.X
	dy := st.TrialY[idx] - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	switch st.Role[idx] {
	case 0: // Employed bee — exploit food source neighborhood.
		if dist < abcArrivalDist {
			trialF := abcFitness(bot, ss)
			if trialF > st.Fitness[idx] {
				st.Fitness[idx] = trialF
				st.Stale[idx] = 0
			}
		}
		fit01 := math.Min(math.Max((st.Fitness[idx]+50)/150, 0), 1)
		g := uint8(150 + fit01*105)
		bot.LEDColor = [3]uint8{255, g, 30}

	case 1: // Onlooker bee — follow recruited source.
		if dist < abcArrivalDist {
			trialF := abcFitness(bot, ss)
			if trialF > st.Fitness[idx] {
				st.Fitness[idx] = trialF
			}
		}
		bot.LEDColor = [3]uint8{255, 140, 0}

	case 2: // Scout bee — explore new random position.
		if dist < abcArrivalDist {
			st.Role[idx] = 0
			st.Fitness[idx] = abcFitness(bot, ss)
		}
		bot.LEDColor = [3]uint8{255, 255, 255}
	}

	// Gold LED for global best bot.
	if idx == st.GlobalBestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0}
	}
}

// abcFitness evaluates the fitness of a bot's position using the shared
// Gaussian fitness landscape for consistent comparison across algorithms.
func abcFitness(bot *SwarmBot, ss *SwarmState) float64 {
	return distanceFitness(bot, ss)
}
