package swarm

import (
	"math"
	"sort"
	"swarmsim/logger"
)

// PredatorPreyState manages the co-evolutionary predator-prey system.
type PredatorPreyState struct {
	PredatorCount  int     // number of predator bots
	PreyCount      int     // number of prey bots
	CatchRadius    float64 // distance for a predator to catch prey
	CatchCooldown  int     // ticks after a catch before predator can catch again
	RespawnDelay   int     // ticks before caught prey respawns
	PredatorSpeed  float64 // speed multiplier for predators (default 1.3)
	PreySpeed      float64 // speed multiplier for prey (default 1.0)

	// Evolution parameters
	EvoInterval int // ticks between evolution steps (default 2000)
	EvoTimer    int // current tick counter
	Generation  int

	// Per-bot predator/prey state (indexed by bot index)
	Roles       []PredatorPreyRole
	CatchCount  []int // per-predator catch count
	EscapeCount []int // per-prey escape count (ticks survived)
	Cooldowns   []int // per-bot cooldown timer

	// Stats
	TotalCatches  int
	TotalEscapes  int
	PredFitHistory []FitnessRecord
	PreyFitHistory []FitnessRecord
}

// PredatorPreyRole identifies whether a bot is predator or prey.
type PredatorPreyRole int

const (
	RolePrey     PredatorPreyRole = 0
	RolePredator PredatorPreyRole = 1
)

// InitPredatorPrey sets up the predator-prey system.
// Predators are the first predatorFrac% of bots, rest are prey.
func InitPredatorPrey(ss *SwarmState, predatorFrac float64) {
	n := len(ss.Bots)
	if n < 4 {
		return
	}
	if predatorFrac < 0.1 {
		predatorFrac = 0.1
	}
	if predatorFrac > 0.5 {
		predatorFrac = 0.5
	}

	predCount := int(float64(n) * predatorFrac)
	if predCount < 2 {
		predCount = 2
	}

	pp := &PredatorPreyState{
		PredatorCount: predCount,
		PreyCount:     n - predCount,
		CatchRadius:   15.0,
		CatchCooldown: 60,
		RespawnDelay:  30,
		PredatorSpeed: 1.3,
		PreySpeed:     1.0,
		EvoInterval:   2000,
		Roles:         make([]PredatorPreyRole, n),
		CatchCount:    make([]int, n),
		EscapeCount:   make([]int, n),
		Cooldowns:     make([]int, n),
	}

	// Assign roles
	for i := 0; i < n; i++ {
		if i < predCount {
			pp.Roles[i] = RolePredator
			ss.Bots[i].LEDColor = [3]uint8{255, 50, 50} // red = predator
		} else {
			pp.Roles[i] = RolePrey
			ss.Bots[i].LEDColor = [3]uint8{50, 200, 50} // green = prey
		}
	}

	// Init neural brains for all bots (predators get different input set)
	for i := range ss.Bots {
		brain := &NeuroBrain{}
		for w := 0; w < NeuroWeights; w++ {
			brain.Weights[w] = (ss.Rng.Float64() - 0.5) * 2.0 / math.Sqrt(float64(NeuroInputs))
		}
		ss.Bots[i].Brain = brain
	}

	ss.PredatorPrey = pp
	logger.Info("PREDPREY", "Initialisiert: %d Predators, %d Prey", predCount, n-predCount)
}

// ClearPredatorPrey disables the predator-prey system.
func ClearPredatorPrey(ss *SwarmState) {
	ss.PredatorPrey = nil
	ss.PredatorPreyOn = false
}

// TickPredatorPrey runs one tick of the predator-prey simulation.
func TickPredatorPrey(ss *SwarmState) {
	pp := ss.PredatorPrey
	if pp == nil {
		return
	}

	n := len(ss.Bots)
	if len(pp.Roles) != n {
		return
	}

	// Update cooldowns
	for i := range pp.Cooldowns {
		if pp.Cooldowns[i] > 0 {
			pp.Cooldowns[i]--
		}
	}

	// Predators try to catch prey
	for i := 0; i < n; i++ {
		if pp.Roles[i] != RolePredator || pp.Cooldowns[i] > 0 {
			continue
		}

		for j := 0; j < n; j++ {
			if pp.Roles[j] != RolePrey {
				continue
			}

			dx := ss.Bots[i].X - ss.Bots[j].X
			dy := ss.Bots[i].Y - ss.Bots[j].Y
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist < pp.CatchRadius {
				// Catch!
				pp.CatchCount[i]++
				pp.TotalCatches++
				pp.Cooldowns[i] = pp.CatchCooldown

				// Respawn prey at random location
				margin := 30.0
				ss.Bots[j].X = margin + ss.Rng.Float64()*(ss.ArenaW-2*margin)
				ss.Bots[j].Y = margin + ss.Rng.Float64()*(ss.ArenaH-2*margin)
				ss.Bots[j].Angle = ss.Rng.Float64() * 2 * math.Pi

				// Flash effect
				ss.Bots[j].BlinkTimer = 10
				break // one catch per tick per predator
			}
		}
	}

	// Count escape ticks for prey
	for i := 0; i < n; i++ {
		if pp.Roles[i] == RolePrey {
			pp.EscapeCount[i]++
		}
	}

	// Evolution check
	pp.EvoTimer++
	if pp.EvoTimer >= pp.EvoInterval {
		EvolvePredatorPrey(ss)
		pp.EvoTimer = 0
	}

	// Update LED colors
	for i := 0; i < n; i++ {
		if pp.Roles[i] == RolePredator {
			if pp.Cooldowns[i] > 0 {
				ss.Bots[i].LEDColor = [3]uint8{150, 30, 30} // dim red = cooling down
			} else {
				ss.Bots[i].LEDColor = [3]uint8{255, 50, 50} // bright red
			}
		} else {
			ss.Bots[i].LEDColor = [3]uint8{50, 200, 50} // green
		}
	}
}

// PredatorPreyFitness computes fitness for a bot in the predator-prey context.
func PredatorPreyFitness(pp *PredatorPreyState, botIdx int) float64 {
	if pp == nil || botIdx < 0 || botIdx >= len(pp.Roles) {
		return 0
	}
	if pp.Roles[botIdx] == RolePredator {
		// Predators: maximize catches
		return float64(pp.CatchCount[botIdx]) * 100
	}
	// Prey: maximize survival time
	return float64(pp.EscapeCount[botIdx]) * 0.1
}

// EvolvePredatorPrey evolves predators and prey populations independently.
func EvolvePredatorPrey(ss *SwarmState) {
	pp := ss.PredatorPrey
	if pp == nil {
		return
	}
	n := len(ss.Bots)

	// Collect predator and prey indices
	var predIdxs, preyIdxs []int
	for i := 0; i < n; i++ {
		if pp.Roles[i] == RolePredator {
			predIdxs = append(predIdxs, i)
		} else {
			preyIdxs = append(preyIdxs, i)
		}
	}

	// Evolve predators
	evolvePPPopulation(ss, pp, predIdxs, "PRED")
	// Evolve prey
	evolvePPPopulation(ss, pp, preyIdxs, "PREY")

	// Record fitness history
	predBest, predAvg := ppPopFitness(pp, predIdxs)
	preyBest, preyAvg := ppPopFitness(pp, preyIdxs)
	pp.PredFitHistory = append(pp.PredFitHistory, FitnessRecord{Best: predBest, Avg: predAvg})
	pp.PreyFitHistory = append(pp.PreyFitHistory, FitnessRecord{Best: preyBest, Avg: preyAvg})

	// Reset per-bot counters
	for i := range pp.CatchCount {
		pp.CatchCount[i] = 0
	}
	for i := range pp.EscapeCount {
		pp.EscapeCount[i] = 0
	}

	pp.Generation++
	logger.Info("PREDPREY", "Gen %d — Pred Best: %.0f Avg: %.0f | Prey Best: %.0f Avg: %.0f",
		pp.Generation, predBest, predAvg, preyBest, preyAvg)
}

// ppPopFitness computes best and average fitness for a population subset.
func ppPopFitness(pp *PredatorPreyState, indices []int) (best, avg float64) {
	if len(indices) == 0 {
		return 0, 0
	}
	total := 0.0
	best = -1e9
	for _, idx := range indices {
		f := PredatorPreyFitness(pp, idx)
		total += f
		if f > best {
			best = f
		}
	}
	return best, total / float64(len(indices))
}

// evolvePPPopulation runs evolution on a subset of bots (predators or prey).
func evolvePPPopulation(ss *SwarmState, pp *PredatorPreyState, indices []int, tag string) {
	n := len(indices)
	if n < 4 {
		return
	}

	// Compute fitness
	fitnesses := make([]float64, n)
	for i, idx := range indices {
		fitnesses[i] = PredatorPreyFitness(pp, idx)
	}

	// Sort by fitness descending
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		return fitnesses[order[a]] > fitnesses[order[b]]
	})

	// Top 20% are parents
	parentCount := n * 20 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 2
	if eliteCount > parentCount {
		eliteCount = parentCount
	}

	// Save parent weights
	type savedW struct {
		w [NeuroWeights]float64
	}
	parentW := make([]savedW, parentCount)
	for i := 0; i < parentCount; i++ {
		botIdx := indices[order[i]]
		if ss.Bots[botIdx].Brain != nil {
			parentW[i].w = ss.Bots[botIdx].Brain.Weights
		}
	}

	freshCount := n * 10 / 100
	if freshCount < 1 {
		freshCount = 1
	}

	for rank := 0; rank < n; rank++ {
		botIdx := indices[order[rank]]
		if ss.Bots[botIdx].Brain == nil {
			ss.Bots[botIdx].Brain = &NeuroBrain{}
		}

		if rank < eliteCount {
			// Keep elite unchanged
			ss.Bots[botIdx].Brain.Weights = parentW[rank].w
		} else if rank >= n-freshCount {
			// Fresh random
			for w := 0; w < NeuroWeights; w++ {
				ss.Bots[botIdx].Brain.Weights[w] = (ss.Rng.Float64() - 0.5) * 2.0 / math.Sqrt(float64(NeuroInputs))
			}
		} else {
			// Crossover + mutation
			p1 := ss.Rng.Intn(parentCount)
			p2 := ss.Rng.Intn(parentCount)
			for w := 0; w < NeuroWeights; w++ {
				if ss.Rng.Float64() < 0.5 {
					ss.Bots[botIdx].Brain.Weights[w] = parentW[p1].w[w]
				} else {
					ss.Bots[botIdx].Brain.Weights[w] = parentW[p2].w[w]
				}
				if ss.Rng.Float64() < 0.15 {
					ss.Bots[botIdx].Brain.Weights[w] += ss.Rng.NormFloat64() * 0.3
				}
			}
		}

		// Reset stats
		ss.Bots[botIdx].Stats = BotLifetimeStats{}
		ss.Bots[botIdx].Fitness = 0
	}
}

// BuildPredatorPreyInputs constructs neural inputs for predator-prey mode.
// Predators get prey-seeking sensors, prey get predator-avoiding sensors.
func BuildPredatorPreyInputs(bot *SwarmBot, ss *SwarmState, botIdx int) [NeuroInputs]float64 {
	var inp [NeuroInputs]float64
	pp := ss.PredatorPrey
	if pp == nil {
		return inp
	}

	role := pp.Roles[botIdx]

	// Find nearest opponent
	nearestDist := 999.0
	nearestAngle := 0.0
	opponentCount := 0
	for j := range ss.Bots {
		if j == botIdx || pp.Roles[j] == role {
			continue
		}
		dx := ss.Bots[j].X - bot.X
		dy := ss.Bots[j].Y - bot.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d < SwarmSensorRange && d < nearestDist {
			nearestDist = d
			nearestAngle = math.Atan2(dy, dx) - bot.Angle
		}
		if d < SwarmSensorRange {
			opponentCount++
		}
	}

	// Find nearest ally
	nearestAllyDist := 999.0
	for j := range ss.Bots {
		if j == botIdx || pp.Roles[j] != role {
			continue
		}
		dx := ss.Bots[j].X - bot.X
		dy := ss.Bots[j].Y - bot.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d < SwarmSensorRange && d < nearestAllyDist {
			nearestAllyDist = d
		}
	}

	// [0] nearest opponent distance (normalized)
	if nearestDist > 200 {
		nearestDist = 200
	}
	inp[0] = nearestDist / 200.0

	// [1] opponent count (normalized)
	oc := float64(opponentCount)
	if oc > 10 {
		oc = 10
	}
	inp[1] = oc / 10.0

	// [2] edge
	if bot.OnEdge {
		inp[2] = 1.0
	}

	// [3] angle to nearest opponent (normalized to -1..1)
	inp[3] = nearestAngle / math.Pi

	// [4] nearest ally distance (normalized)
	nad := nearestAllyDist
	if nad > 200 {
		nad = 200
	}
	inp[4] = nad / 200.0

	// [5] cooldown (predators: can I catch?)
	if role == RolePredator && pp.Cooldowns[botIdx] > 0 {
		inp[5] = float64(pp.Cooldowns[botIdx]) / float64(pp.CatchCooldown)
	}

	// [6] obstacle ahead
	if bot.ObstacleAhead {
		inp[6] = 1.0
	}

	// [7] speed
	inp[7] = bot.Speed / SwarmBotSpeed

	// [8] heading (normalized)
	inp[8] = bot.Angle / (2 * math.Pi)

	// [9] x position (normalized)
	inp[9] = bot.X / ss.ArenaW

	// [10] random
	inp[10] = ss.Rng.Float64()

	// [11] bias
	inp[11] = 1.0

	return inp
}

// PredatorPreyBotCount returns predator and prey counts.
func PredatorPreyBotCount(pp *PredatorPreyState) (pred, prey int) {
	if pp == nil {
		return 0, 0
	}
	return pp.PredatorCount, pp.PreyCount
}

// IsPredator returns whether a bot is a predator.
func IsPredator(pp *PredatorPreyState, idx int) bool {
	if pp == nil || idx < 0 || idx >= len(pp.Roles) {
		return false
	}
	return pp.Roles[idx] == RolePredator
}

// IsPrey returns whether a bot is prey.
func IsPrey(pp *PredatorPreyState, idx int) bool {
	if pp == nil || idx < 0 || idx >= len(pp.Roles) {
		return false
	}
	return pp.Roles[idx] == RolePrey
}

// ApplyPredator executes SwarmScript-triggered predator/prey behavior.
// Predators chase nearest prey, prey flee from nearest predator.
func ApplyPredator(bot *SwarmBot, ss *SwarmState, idx int) {
	pp := ss.PredatorPrey
	if pp == nil || idx >= len(pp.Roles) || ss.Hash == nil {
		bot.Speed = SwarmBotSpeed
		return
	}

	isPred := pp.Roles[idx] == RolePredator
	searchRadius := 150.0

	nearIDs := ss.Hash.Query(bot.X, bot.Y, searchRadius)
	bestDist := math.MaxFloat64
	bestAngle := bot.Angle

	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) || j >= len(pp.Roles) {
			continue
		}
		// Opposite role
		if isPred == (pp.Roles[j] == RolePredator) {
			continue
		}
		dx := ss.Bots[j].X - bot.X
		dy := ss.Bots[j].Y - bot.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d < bestDist {
			bestDist = d
			bestAngle = math.Atan2(dy, dx)
		}
	}

	if bestDist < searchRadius {
		targetAngle := bestAngle
		if !isPred {
			// Prey: flee (opposite direction)
			targetAngle += math.Pi
		}
		diff := targetAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		steer := 0.2
		if diff > steer {
			diff = steer
		} else if diff < -steer {
			diff = -steer
		}
		bot.Angle += diff

		if isPred {
			bot.Speed = SwarmBotSpeed * 1.5
			bot.LEDColor = [3]uint8{255, 50, 50}
		} else {
			bot.Speed = SwarmBotSpeed * 1.3
			bot.LEDColor = [3]uint8{50, 255, 50}
		}
	} else {
		// No target nearby: patrol
		bot.Speed = SwarmBotSpeed
		if isPred {
			bot.LEDColor = [3]uint8{200, 80, 80}
		} else {
			bot.LEDColor = [3]uint8{80, 200, 80}
		}
	}
}
