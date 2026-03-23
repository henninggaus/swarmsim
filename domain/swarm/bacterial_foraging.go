package swarm

import "math"

// Bacterial Foraging Optimization (BFO): Inspired by the foraging behavior
// of E. coli bacteria. Bacteria use chemotaxis (swim & tumble) to navigate
// nutrient gradients, reproduce (cell division of fittest), and are subject
// to elimination-dispersal events that maintain population diversity.
//
// Phases:
// 1. Chemotaxis — swim in current direction if improving, tumble (random dir) if not
// 2. Swarming — cell-to-cell signaling attracts bacteria to nutrient-rich areas
// 3. Reproduction — fittest half clones replace least fit half
// 4. Elimination-Dispersal — random bots teleport to new positions
//
// Reference: Passino, K.M. (2002)
//            "Biomimicry of bacterial foraging for distributed optimization"

const (
	bfoChemoSteps    = 4     // consecutive swim steps before re-evaluation
	bfoTumbleRate    = 0.25  // probability of tumbling each step
	bfoSwimSteerRate = 0.08  // max angle change during swim (radians)
	bfoSwarmRadius   = 60.0  // swarming signal radius
	bfoSwarmAttract  = 0.02  // swarming attraction coefficient
	bfoSwarmRepel    = 0.01  // swarming repulsion coefficient
	bfoReproInterval = 200   // ticks between reproduction events
	bfoElimProb      = 0.02  // probability of elimination-dispersal per bot per cycle
	bfoNutrientDecay = 0.995 // nutrient memory decay per tick
)

// BFOState holds Bacterial Foraging Optimization state.
type BFOState struct {
	Health       []float64 // accumulated nutrient health per bot
	SwimDir      []float64 // current swim direction (radians)
	SwimCount    []int     // steps remaining in current swim
	PrevNutrient []float64 // nutrient value at previous position
	CycleTimer   int       // ticks since last reproduction
}

// InitBFO allocates Bacterial Foraging state for all bots.
func InitBFO(ss *SwarmState) {
	n := len(ss.Bots)
	ss.BFO = &BFOState{
		Health:       make([]float64, n),
		SwimDir:      make([]float64, n),
		SwimCount:    make([]int, n),
		PrevNutrient: make([]float64, n),
	}
	// Initialize random swim directions
	for i := range ss.BFO.SwimDir {
		ss.BFO.SwimDir[i] = ss.Rng.Float64() * 2 * math.Pi
	}
	ss.BFOOn = true
}

// ClearBFO frees Bacterial Foraging state.
func ClearBFO(ss *SwarmState) {
	ss.BFO = nil
	ss.BFOOn = false
}

// TickBFO updates the Bacterial Foraging Optimization for all bots.
func TickBFO(ss *SwarmState) {
	if ss.BFO == nil {
		return
	}
	st := ss.BFO

	// Grow slices if bots were added
	for len(st.Health) < len(ss.Bots) {
		st.Health = append(st.Health, 0)
		st.SwimDir = append(st.SwimDir, ss.Rng.Float64()*2*math.Pi)
		st.SwimCount = append(st.SwimCount, 0)
		st.PrevNutrient = append(st.PrevNutrient, 0)
	}

	st.CycleTimer++

	// Compute current nutrient value for each bot.
	// Nutrient is a function of neighbor density (swarming signal)
	// and resource proximity.
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		nutrient := computeNutrient(bot, ss, i)

		// Chemotaxis: compare with previous nutrient
		if st.SwimCount[i] <= 0 {
			// Decide swim vs tumble
			if nutrient > st.PrevNutrient[i] {
				// Keep swimming in current direction (nutrient improving)
				st.SwimCount[i] = bfoChemoSteps
			} else {
				// Tumble: pick a new random direction
				st.SwimDir[i] = ss.Rng.Float64() * 2 * math.Pi
				st.SwimCount[i] = bfoChemoSteps
			}
		}
		st.SwimCount[i]--

		// Random tumble chance even during swim
		if ss.Rng.Float64() < bfoTumbleRate {
			st.SwimDir[i] += (ss.Rng.Float64() - 0.5) * math.Pi
		}

		st.PrevNutrient[i] = nutrient

		// Accumulate health (higher nutrient = healthier bacterium)
		st.Health[i] = st.Health[i]*bfoNutrientDecay + nutrient
	}

	// Reproduction: fittest half replaces least fit half
	if st.CycleTimer >= bfoReproInterval {
		st.CycleTimer = 0
		bfoReproduce(ss)
	}

	// Elimination-dispersal: random bots teleport
	for i := range ss.Bots {
		if ss.Rng.Float64() < bfoElimProb {
			ss.Bots[i].X = ss.Rng.Float64() * ss.ArenaW
			ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
			st.Health[i] = 0
			st.SwimDir[i] = ss.Rng.Float64() * 2 * math.Pi
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].BFOHealth = int(st.Health[i] * 10)
		if ss.Bots[i].BFOHealth > 100 {
			ss.Bots[i].BFOHealth = 100
		}
		if ss.Bots[i].BFOHealth < 0 {
			ss.Bots[i].BFOHealth = 0
		}
		ss.Bots[i].BFOSwimming = 1
		if st.SwimCount[i] <= 0 {
			ss.Bots[i].BFOSwimming = 0 // tumbling
		}
		ss.Bots[i].BFONutrient = int(st.PrevNutrient[i] * 100)
	}
}

// computeNutrient calculates the nutrient landscape value at a bot's position.
// Combines swarming signals (attracted to nearby bacteria) with environmental cues.
func computeNutrient(bot *SwarmBot, ss *SwarmState, idx int) float64 {
	if ss.Hash == nil {
		return 0.5
	}

	nearIDs := ss.Hash.Query(bot.X, bot.Y, bfoSwarmRadius)
	attractSignal := 0.0
	repelSignal := 0.0

	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx := nb.X - bot.X
		dy := nb.Y - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 1.0 {
			dist = 1.0
		}
		if dist > bfoSwarmRadius {
			continue
		}
		// Attraction: nearby bacteria indicate nutrients
		attractSignal += bfoSwarmAttract * math.Exp(-dist/bfoSwarmRadius)
		// Repulsion: too close causes competition
		repelSignal += bfoSwarmRepel * math.Exp(-dist/(bfoSwarmRadius*0.3))
	}

	// Landscape fitness as base nutrient (shared Gaussian peaks)
	landFit := distanceFitness(bot, ss) / 100.0
	if landFit < 0 {
		landFit = 0
	}
	baseNutrient := landFit * 0.5
	energyBonus := (bot.Energy / 100.0) * 0.2
	carry := 0.0
	if bot.CarryingPkg >= 0 {
		carry = 0.2
	}

	total := baseNutrient + energyBonus + carry + attractSignal - repelSignal
	if total < 0 {
		total = 0
	}
	if total > 1 {
		total = 1
	}
	return total
}

// bfoReproduce performs reproduction: healthiest half replaces least healthy half.
// The healthy bacteria "clone" their swim direction and reset health.
func bfoReproduce(ss *SwarmState) {
	st := ss.BFO
	n := len(ss.Bots)
	if n < 4 {
		return
	}

	// Simple selection: find median health, clone above-median to below-median
	// Build sorted index by health (insertion sort for small N)
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	// Partial sort: just find the half boundary
	midHealth := 0.0
	for i := range ss.Bots {
		midHealth += st.Health[i]
	}
	midHealth /= float64(n)

	// Clone fittest traits to least fit
	for i := range ss.Bots {
		if st.Health[i] < midHealth {
			// Find a random healthy donor
			donor := ss.Rng.Intn(n)
			for attempts := 0; attempts < 5 && st.Health[donor] < midHealth; attempts++ {
				donor = ss.Rng.Intn(n)
			}
			if st.Health[donor] >= midHealth {
				st.SwimDir[i] = st.SwimDir[donor] + (ss.Rng.Float64()-0.5)*0.3
				st.Health[i] = st.Health[donor] * 0.5
			}
		}
	}
}

// ApplyBFO steers a bot using Bacterial Foraging chemotaxis.
func ApplyBFO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.BFO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.BFO
	if idx >= len(st.SwimDir) {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Swarming: attract toward nearby healthy bacteria
	swarmAngle := st.SwimDir[idx]
	if ss.Hash != nil {
		nearIDs := ss.Hash.Query(bot.X, bot.Y, bfoSwarmRadius)
		sx, sy := 0.0, 0.0
		n := 0
		for _, j := range nearIDs {
			if j == idx || j < 0 || j >= len(ss.Bots) {
				continue
			}
			nb := &ss.Bots[j]
			dx := nb.X - bot.X
			dy := nb.Y - bot.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1.0 || dist > bfoSwarmRadius {
				continue
			}
			// Weight attraction by neighbor health
			w := 1.0
			if j < len(st.Health) {
				w = st.Health[j] * 0.1
			}
			sx += (dx / dist) * w
			sy += (dy / dist) * w
			n++
		}
		if n > 0 {
			swarmAngle = math.Atan2(
				math.Sin(st.SwimDir[idx])+sy*0.3,
				math.Cos(st.SwimDir[idx])+sx*0.3,
			)
		}
	}

	// Steer toward computed direction
	steerToward(bot, swarmAngle, bfoSwimSteerRate)
	bot.Speed = SwarmBotSpeed

	// LED color based on health: green (healthy) → red (unhealthy)
	health := st.Health[idx]
	if health > 10 {
		health = 10
	}
	g := uint8(math.Min(255, health*25))
	r := uint8(math.Min(255, (10-health)*25))
	bot.LEDColor = [3]uint8{r, g, 50}
}
