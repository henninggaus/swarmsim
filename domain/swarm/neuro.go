package swarm

import (
	"math"
	"sort"
	"swarmsim/logger"
)

// ═══════════════════════════════════════════════════════════
// NEUROEVOLUTION — Neuronales Netz pro Bot
// ═══════════════════════════════════════════════════════════
//
// Jeder Bot bekommt ein eigenes kleines neuronales Netz:
//   12 Sensor-Inputs → 6 Hidden-Neuronen (tanh) → 8 Action-Outputs
//
// Architektur:
//   - Input-Layer (12 Neuronen): normalisierte Sensorwerte
//     [near_dist, neighbors, edge, carry, p_dist, d_dist,
//      match, has_pkg, obs_ahead, light, rnd, bias]
//   - Hidden-Layer (6 Neuronen): tanh-Aktivierung
//   - Output-Layer (8 Neuronen): eine Aktion pro Neuron
//     [FWD, TURN_LEFT, TURN_RIGHT, TURN_TO_NEAREST,
//      TURN_FROM_NEAREST, PICKUP, DROP, GOTO_DROPOFF]
//
// Gewichte: 12×6 + 6×8 = 72 + 48 = 120 Gewichte pro Bot
//
// Evolution:
//   - Alle 2000 Ticks: Fitness bewerten, Top 20% sind Eltern
//   - Crossover: Gewichte von zwei Eltern mischen
//   - Mutation: ADAPTIVE Rate (5-40%) und Staerke (0.1-0.8)
//     → steigt bei Stagnation, sinkt bei Fortschritt
//   - 10% jeder Generation sind komplett neue Zufalls-Netze
//   - Fitness = Deliveries×30 + Pickups×15 + Distance×0.01
//               - AntiStuck×10 - Idle×0.05
//
// Das Netz lernt NICHT durch Backpropagation, sondern durch
// Evolution: Die besten Netze werden kopiert und leicht
// verändert — wie in der Natur.

const (
	NeuroInputs  = 12
	NeuroHidden  = 6
	NeuroOutputs = 8
	NeuroWeights = NeuroInputs*NeuroHidden + NeuroHidden*NeuroOutputs // 120
)

// NeuroAction names for visualization and logging.
var NeuroActionNames = [NeuroOutputs]string{
	"FWD", "TURN_L", "TURN_R", "TO_NEAR",
	"FROM_NEAR", "PICKUP", "DROP", "GO_DROP",
}

// NeuroTruckActionNames for truck mode visualization.
var NeuroTruckActionNames = [NeuroOutputs]string{
	"FWD", "TURN_L", "TURN_R", "TO_NEAR",
	"FROM_NEAR", "GO_RAMP", "DROP", "GO_DROP",
}

// NeuroInputNames for visualization.
var NeuroInputNames = [NeuroInputs]string{
	"near_dist", "neighbors", "edge", "carry",
	"p_dist", "d_dist", "match", "has_pkg",
	"obs_ahead", "light", "rnd", "bias",
}

// NeuroTruckInputNames for truck mode visualization.
var NeuroTruckInputNames = [NeuroInputs]string{
	"near_dist", "neighbors", "edge", "carry",
	"trk_pkg_d", "d_dist", "match", "on_ramp",
	"obs_ahead", "trk_here", "rnd", "bias",
}

// NeuroBrain holds the neural network weights for a single bot.
type NeuroBrain struct {
	Weights [NeuroWeights]float64 // all weights flattened: [input→hidden | hidden→output]

	// Cached activations for visualization (updated each forward pass)
	HiddenAct [NeuroHidden]float64  // hidden layer activations after tanh
	OutputAct [NeuroOutputs]float64 // output layer activations (raw)
	InputVals [NeuroInputs]float64  // last input values
	ActionIdx int                   // index of chosen action (highest output)
}

// NeuroForward runs the forward pass: inputs → hidden (tanh) → outputs.
// Returns the index of the output neuron with the highest activation.
func NeuroForward(brain *NeuroBrain, inputs [NeuroInputs]float64) int {
	brain.InputVals = inputs

	// Input → Hidden (weights [0 .. NeuroInputs*NeuroHidden))
	for h := 0; h < NeuroHidden; h++ {
		sum := 0.0
		for inp := 0; inp < NeuroInputs; inp++ {
			sum += inputs[inp] * brain.Weights[inp*NeuroHidden+h]
		}
		brain.HiddenAct[h] = math.Tanh(sum)
	}

	// Hidden → Output (weights [NeuroInputs*NeuroHidden .. end))
	offset := NeuroInputs * NeuroHidden
	bestIdx := 0
	bestVal := -1e9
	for o := 0; o < NeuroOutputs; o++ {
		sum := 0.0
		for h := 0; h < NeuroHidden; h++ {
			sum += brain.HiddenAct[h] * brain.Weights[offset+h*NeuroOutputs+o]
		}
		brain.OutputAct[o] = sum
		if sum > bestVal {
			bestVal = sum
			bestIdx = o
		}
	}
	brain.ActionIdx = bestIdx
	return bestIdx
}

// BuildNeuroInputs constructs the normalized input vector from bot sensor values.
func BuildNeuroInputs(bot *SwarmBot, ss *SwarmState) [NeuroInputs]float64 {
	var inp [NeuroInputs]float64

	// near_dist: normalized 0-1 (0=touching, 1=far away/no neighbor)
	nd := bot.NearestDist
	if nd > 200 {
		nd = 200
	}
	inp[0] = nd / 200.0

	// neighbors: normalized 0-1 (0=none, 1=10+)
	nc := float64(bot.NeighborCount)
	if nc > 10 {
		nc = 10
	}
	inp[1] = nc / 10.0

	// edge: 0 or 1
	if bot.OnEdge {
		inp[2] = 1.0
	}

	// carry: 0 or 1
	if bot.CarryingPkg >= 0 {
		inp[3] = 1.0
	}

	// p_dist: normalized pickup distance (0=at pickup, 1=far)
	pd := bot.NearestPickupDist
	if pd > 500 {
		pd = 500
	}
	inp[4] = pd / 500.0

	// d_dist: normalized dropoff distance
	dd := bot.NearestDropoffDist
	if dd > 500 {
		dd = 500
	}
	inp[5] = dd / 500.0

	// match: dropoff matches carried package color
	if bot.DropoffMatch {
		inp[6] = 1.0
	}

	// has_pkg: nearest pickup has a package available
	if bot.NearestPickupHasPkg {
		inp[7] = 1.0
	}

	// obs_ahead: obstacle ahead
	if bot.ObstacleAhead {
		inp[8] = 1.0
	}

	// light: normalized 0-1 (clamped)
	lv := float64(bot.LightValue)
	if lv > 100 {
		lv = 100
	}
	inp[9] = lv / 100.0

	// rnd: random noise for exploration
	inp[10] = ss.Rng.Float64()

	// bias: always 1.0
	inp[11] = 1.0

	return inp
}

// BuildNeuroTruckInputs builds sensor inputs for truck unloading mode.
// Same 12-input architecture, but sensors are mapped to truck-relevant data.
func BuildNeuroTruckInputs(bot *SwarmBot, ss *SwarmState) [NeuroInputs]float64 {
	var inp [NeuroInputs]float64

	// [0] near_dist: normalized distance to nearest neighbor
	nd := bot.NearestDist
	if nd > 200 {
		nd = 200
	}
	inp[0] = nd / 200.0

	// [1] neighbors: count in range
	nc := float64(bot.NeighborCount)
	if nc > 10 {
		nc = 10
	}
	inp[1] = nc / 10.0

	// [2] edge: at arena border
	if bot.OnEdge {
		inp[2] = 1.0
	}

	// [3] carry: carrying a package
	if bot.CarryingPkg >= 0 {
		inp[3] = 1.0
	}

	// [4] trk_pkg_d: distance to nearest truck package (replaces pickup dist)
	td := bot.NearestTruckPkgDist
	if td > 500 {
		td = 500
	}
	inp[4] = td / 500.0

	// [5] d_dist: distance to nearest dropoff
	dd := bot.NearestDropoffDist
	if dd > 500 {
		dd = 500
	}
	inp[5] = dd / 500.0

	// [6] match: dropoff matches carried package color
	if bot.DropoffMatch {
		inp[6] = 1.0
	}

	// [7] on_ramp: bot is on the truck ramp (replaces has_pkg)
	if bot.OnRamp {
		inp[7] = 1.0
	}

	// [8] obs_ahead: obstacle ahead
	if bot.ObstacleAhead {
		inp[8] = 1.0
	}

	// [9] trk_here: truck is present (replaces light)
	if bot.TruckHere {
		inp[9] = 1.0
	}

	// [10] rnd: exploration noise
	inp[10] = ss.Rng.Float64()

	// [11] bias: always 1.0
	inp[11] = 1.0

	return inp
}

// ExecuteNeuroAction performs the action selected by the neural network.
func ExecuteNeuroAction(actionIdx int, bot *SwarmBot, ss *SwarmState, botIdx int) {
	switch actionIdx {
	case 0: // FWD
		bot.Speed = SwarmBotSpeed
	case 1: // TURN_LEFT
		bot.Angle -= 0.3
	case 2: // TURN_RIGHT
		bot.Angle += 0.3
	case 3: // TURN_TO_NEAREST
		if bot.NearestIdx >= 0 {
			dx, dy := NeighborDelta(bot.X, bot.Y, ss.Bots[bot.NearestIdx].X, ss.Bots[bot.NearestIdx].Y, ss)
			bot.Angle = math.Atan2(dy, dx)
		}
		bot.Speed = SwarmBotSpeed
	case 4: // TURN_FROM_NEAREST
		if bot.NearestIdx >= 0 {
			dx, dy := NeighborDelta(bot.X, bot.Y, ss.Bots[bot.NearestIdx].X, ss.Bots[bot.NearestIdx].Y, ss)
			bot.Angle = math.Atan2(-dy, -dx)
		}
		bot.Speed = SwarmBotSpeed
	case 5: // PICKUP
		if bot.CarryingPkg < 0 && bot.NearestPickupDist < 25 && bot.NearestPickupHasPkg {
			bot.Speed = SwarmBotSpeed
			// Actual pickup handled by delivery system
		}
		// Navigate to pickup if not carrying
		if bot.CarryingPkg < 0 && bot.NearestPickupIdx >= 0 {
			st := &ss.Stations[bot.NearestPickupIdx]
			dx, dy := NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
			bot.Speed = SwarmBotSpeed
		}
	case 6: // DROP
		if bot.CarryingPkg >= 0 && bot.DropoffMatch && bot.NearestDropoffDist < 30 {
			bot.Speed = SwarmBotSpeed
			// Actual drop handled by delivery system
		}
	case 7: // GOTO_DROPOFF
		if bot.CarryingPkg >= 0 && bot.NearestDropoffIdx >= 0 {
			st := &ss.Stations[bot.NearestDropoffIdx]
			dx, dy := NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
			bot.Speed = SwarmBotSpeed
		}
	}

	// Obstacle avoidance overlay — always active for neuro bots
	if bot.ObstacleAhead && actionIdx != 4 { // not already turning away
		bot.Angle += 0.5 // slight turn to avoid
	}
	// Edge bounce
	if bot.OnEdge {
		bot.Angle += math.Pi
	}

	// LED color: carrying → package color, otherwise dim orange (neuro indicator)
	if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
		switch ss.Packages[bot.CarryingPkg].Color {
		case 1:
			bot.LEDColor = [3]uint8{255, 60, 60} // red
		case 2:
			bot.LEDColor = [3]uint8{60, 60, 255} // blue
		case 3:
			bot.LEDColor = [3]uint8{255, 255, 60} // yellow
		case 4:
			bot.LEDColor = [3]uint8{60, 255, 60} // green
		}
	} else {
		bot.LEDColor = [3]uint8{200, 120, 40} // dim orange = neuro bot
	}
}

// ExecuteNeuroTruckAction performs truck-mode actions for the neural network.
// Actions 0-4 are same as delivery (movement). 5=GOTO_RAMP, 6=DROP, 7=GOTO_DROPOFF.
func ExecuteNeuroTruckAction(actionIdx int, bot *SwarmBot, ss *SwarmState, botIdx int) {
	switch actionIdx {
	case 0: // FWD
		bot.Speed = SwarmBotSpeed
	case 1: // TURN_LEFT
		bot.Angle -= 0.3
	case 2: // TURN_RIGHT
		bot.Angle += 0.3
	case 3: // TURN_TO_NEAREST
		if bot.NearestIdx >= 0 {
			dx, dy := NeighborDelta(bot.X, bot.Y, ss.Bots[bot.NearestIdx].X, ss.Bots[bot.NearestIdx].Y, ss)
			bot.Angle = math.Atan2(dy, dx)
		}
		bot.Speed = SwarmBotSpeed
	case 4: // TURN_FROM_NEAREST
		if bot.NearestIdx >= 0 {
			dx, dy := NeighborDelta(bot.X, bot.Y, ss.Bots[bot.NearestIdx].X, ss.Bots[bot.NearestIdx].Y, ss)
			bot.Angle = math.Atan2(-dy, -dx)
		}
		bot.Speed = SwarmBotSpeed
	case 5: // GOTO_RAMP — navigate to the truck ramp to pick up packages
		if ss.TruckState != nil && ss.TruckState.CurrentTruck != nil {
			ts := ss.TruckState
			// Target ramp edge, spread bots
			cx := ts.RampX + ts.RampW + 20
			slots := 20
			slot := botIdx % slots
			yFrac := (float64(slot) + 0.5) / float64(slots)
			cy := ts.RampY + ts.RampH*0.1 + ts.RampH*0.8*yFrac
			dx := cx - bot.X
			dy := cy - bot.Y
			bot.Angle = math.Atan2(dy, dx)
			bot.Speed = SwarmBotSpeed
		}
	case 6: // DROP
		if bot.CarryingPkg >= 0 && bot.DropoffMatch && bot.NearestDropoffDist < 30 {
			bot.Speed = SwarmBotSpeed
			// Actual drop handled by delivery system
		}
	case 7: // GOTO_DROPOFF
		if bot.CarryingPkg >= 0 && bot.NearestDropoffIdx >= 0 {
			st := &ss.Stations[bot.NearestDropoffIdx]
			dx, dy := NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
			bot.Speed = SwarmBotSpeed
		}
	}

	// Obstacle avoidance
	if bot.ObstacleAhead && actionIdx != 4 {
		bot.Angle += 0.5
	}
	if bot.OnEdge {
		bot.Angle += math.Pi
	}

	// LED: carrying → package color, on ramp → cyan, otherwise purple (truck-neuro)
	if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
		switch ss.Packages[bot.CarryingPkg].Color {
		case 1:
			bot.LEDColor = [3]uint8{255, 60, 60}
		case 2:
			bot.LEDColor = [3]uint8{60, 60, 255}
		case 3:
			bot.LEDColor = [3]uint8{255, 255, 60}
		case 4:
			bot.LEDColor = [3]uint8{60, 255, 60}
		}
	} else if bot.OnRamp {
		bot.LEDColor = [3]uint8{40, 200, 200} // cyan = on ramp
	} else {
		bot.LEDColor = [3]uint8{160, 80, 200} // purple = truck-neuro bot
	}
}

// EvaluateNeuroTruckFitness evaluates fitness for truck unloading mode.
// Rewards: score (packages delivered to correct station), pickups, proximity to ramp.
// Penalties: idle time, anti-stuck.
func EvaluateNeuroTruckFitness(bot *SwarmBot, ss *SwarmState) float64 {
	f := float64(bot.Stats.TotalDeliveries)*40 +
		float64(bot.Stats.TotalPickups)*20 +
		bot.Stats.TotalDistance*0.005 -
		float64(bot.Stats.AntiStuckCount)*10 -
		float64(bot.Stats.TicksIdle)*0.05
	// Bonus for being near truck/ramp when not carrying
	if bot.CarryingPkg < 0 && bot.TruckHere {
		f += 5
	}
	return f
}

// ═══════════════════════════════════════════════════════════
// NEUROEVOLUTION — Initialisierung & Evolution
// ═══════════════════════════════════════════════════════════

// InitNeuro initializes random neural networks for all bots.
func InitNeuro(ss *SwarmState) {
	for i := range ss.Bots {
		brain := &NeuroBrain{}
		for w := 0; w < NeuroWeights; w++ {
			// Xavier-ähnliche Initialisierung: kleiner Bereich für stabile Aktivierungen
			brain.Weights[w] = (ss.Rng.Float64() - 0.5) * 2.0 / math.Sqrt(float64(NeuroInputs))
		}
		ss.Bots[i].Brain = brain
		ss.Bots[i].Fitness = 0
	}
	ss.NeuroGeneration = 0
	ss.NeuroTimer = 0
	ss.FitnessHistory = nil
	logger.Info("NEURO", "Initialisiert: %d Bots × %d Gewichte = %d Parameter total",
		len(ss.Bots), NeuroWeights, len(ss.Bots)*NeuroWeights)
}

// ClearNeuro removes neural networks from all bots.
func ClearNeuro(ss *SwarmState) {
	for i := range ss.Bots {
		ss.Bots[i].Brain = nil
	}
	ss.NeuroGeneration = 0
	ss.NeuroTimer = 0
}

// RunNeuroEvolution performs one generation of neuroevolution.
// Selection → Crossover → Mutation on weight vectors.
func RunNeuroEvolution(ss *SwarmState) {
	n := len(ss.Bots)
	if n < 4 {
		return
	}

	// 1. Evaluate fitness (truck mode uses truck-specific fitness)
	fitnesses := make([]float64, n)
	if ss.TruckToggle && ss.TruckState != nil {
		for i := range ss.Bots {
			fitnesses[i] = EvaluateNeuroTruckFitness(&ss.Bots[i], ss)
		}
	} else {
		for i := range ss.Bots {
			fitnesses[i] = EvaluateGPFitness(&ss.Bots[i])
		}
	}

	// 1b. Novelty Search blending (if enabled)
	if ss.NoveltyEnabled && ss.NoveltyArchive != nil {
		for i := range ss.Bots {
			ss.Bots[i].Behavior = ComputeBehavior(&ss.Bots[i], ss)
		}
		noveltyScores := ComputeNoveltyScores(ss)
		if noveltyScores != nil {
			alpha := ss.NoveltyArchive.Alpha
			for i := range fitnesses {
				fitnesses[i] = BlendFitness(fitnesses[i], noveltyScores[i], alpha)
			}
			behaviors := make([]BehaviorDescriptor, n)
			for i := range ss.Bots {
				behaviors[i] = ss.Bots[i].Behavior
			}
			UpdateNoveltyArchive(ss, behaviors, noveltyScores)
		}
	}

	// 2. Sort by fitness (descending)
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		return fitnesses[indices[a]] > fitnesses[indices[b]]
	})

	// 3. Record stats
	ss.BestFitness = fitnesses[indices[0]]
	total := 0.0
	for _, f := range fitnesses {
		total += f
	}
	ss.AvgFitness = total / float64(n)

	ss.FitnessHistory = append(ss.FitnessHistory, FitnessRecord{
		Best: ss.BestFitness,
		Avg:  ss.AvgFitness,
	})

	// 4. Top 20% are parents (elite)
	parentCount := n * 20 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 3
	if eliteCount > parentCount {
		eliteCount = parentCount
	}

	// 5. Save elite weights (deep copy, no mutation)
	type savedWeights struct {
		weights [NeuroWeights]float64
	}
	eliteWeights := make([]savedWeights, eliteCount)
	for i := 0; i < eliteCount; i++ {
		if ss.Bots[indices[i]].Brain != nil {
			eliteWeights[i].weights = ss.Bots[indices[i]].Brain.Weights
		}
	}

	// 6. Save parent weights for crossover
	parentWeights := make([]savedWeights, parentCount)
	for i := 0; i < parentCount; i++ {
		if ss.Bots[indices[i]].Brain != nil {
			parentWeights[i].weights = ss.Bots[indices[i]].Brain.Weights
		}
	}

	// 7. Adaptive mutation: increase exploration when stagnating
	mutRate, mutStrength := neuroAdaptiveMutation(ss)

	// 8. Generate new population
	freshCount := n * 10 / 100 // 10% fresh random
	if freshCount < 1 {
		freshCount = 1
	}
	crossoverCount := n - eliteCount - freshCount

	// Genealogy: save old BotIDs for parent tracking
	oldBotIDs := make([]int, n)
	for i := range ss.Bots {
		oldBotIDs[i] = ss.Bots[i].BotID
	}

	assigned := 0
	// Elite: copy unchanged
	for i := 0; i < eliteCount && assigned < n; i++ {
		idx := indices[assigned]
		if ss.Bots[idx].Brain == nil {
			ss.Bots[idx].Brain = &NeuroBrain{}
		}
		ss.Bots[idx].Brain.Weights = eliteWeights[i].weights
		// Genealogy
		if ss.Genealogy != nil {
			ss.Bots[idx].ParentA = oldBotIDs[indices[i]]
			ss.Bots[idx].ParentB = -1
			ss.Bots[idx].BotID = AssignBotID(ss.Genealogy)
		}
		assigned++
	}

	// Crossover children
	for i := 0; i < crossoverCount && assigned < n; i++ {
		idx := indices[assigned]
		if ss.Bots[idx].Brain == nil {
			ss.Bots[idx].Brain = &NeuroBrain{}
		}
		p1 := ss.Rng.Intn(parentCount)
		p2 := ss.Rng.Intn(parentCount)
		for w := 0; w < NeuroWeights; w++ {
			// Uniform crossover
			if ss.Rng.Float64() < 0.5 {
				ss.Bots[idx].Brain.Weights[w] = parentWeights[p1].weights[w]
			} else {
				ss.Bots[idx].Brain.Weights[w] = parentWeights[p2].weights[w]
			}
			// Adaptive mutation
			if ss.Rng.Float64() < mutRate {
				ss.Bots[idx].Brain.Weights[w] += ss.Rng.NormFloat64() * mutStrength
			}
		}
		// Genealogy
		if ss.Genealogy != nil {
			ss.Bots[idx].ParentA = oldBotIDs[indices[p1]]
			ss.Bots[idx].ParentB = oldBotIDs[indices[p2]]
			ss.Bots[idx].BotID = AssignBotID(ss.Genealogy)
		}
		assigned++
	}

	// Fresh random
	for assigned < n {
		idx := indices[assigned]
		if ss.Bots[idx].Brain == nil {
			ss.Bots[idx].Brain = &NeuroBrain{}
		}
		for w := 0; w < NeuroWeights; w++ {
			ss.Bots[idx].Brain.Weights[w] = (ss.Rng.Float64() - 0.5) * 2.0 / math.Sqrt(float64(NeuroInputs))
		}
		// Genealogy
		if ss.Genealogy != nil {
			ss.Bots[idx].ParentA = -1
			ss.Bots[idx].ParentB = -1
			ss.Bots[idx].BotID = AssignBotID(ss.Genealogy)
		}
		assigned++
	}

	// Record genealogy
	if ss.Genealogy != nil {
		RecordGeneration(ss.Genealogy, ss.Bots, ss.NeuroGeneration)
	}

	// 9. Reset lifetime stats for next generation
	for i := range ss.Bots {
		ss.Bots[i].Fitness = 0
		ss.Bots[i].Stats = BotLifetimeStats{}
	}

	ss.NeuroGeneration++
	ss.NeuroTimer = 0

	// Log generation milestone
	logger.Info("NEURO", "Gen %d — Best: %.0f, Avg: %.0f (Mut: %.0f%%/%.2f, %d Elite + %d Crossover + %d Neue)",
		ss.NeuroGeneration, ss.BestFitness, ss.AvgFitness,
		mutRate*100, mutStrength, eliteCount, crossoverCount, freshCount)
}

// neuroAdaptiveMutation computes mutation rate and strength based on stagnation.
// When fitness improves → low mutation (exploit). When stagnating → high mutation (explore).
func neuroAdaptiveMutation(ss *SwarmState) (rate, strength float64) {
	const (
		minRate     = 0.05 // 5% minimum mutation rate
		maxRate     = 0.40 // 40% maximum when stagnating
		minStrength = 0.10 // gentle mutations when improving
		maxStrength = 0.80 // aggressive mutations when stuck
	)

	stagnant := neuroStagnantGenerations(ss)

	// Linear ramp from min to max over 5 stagnant generations
	t := float64(stagnant) / 5.0
	if t > 1.0 {
		t = 1.0
	}
	rate = minRate + t*(maxRate-minRate)
	strength = minStrength + t*(maxStrength-minStrength)
	return rate, strength
}

// neuroStagnantGenerations counts how many recent generations had no best-fitness improvement.
func neuroStagnantGenerations(ss *SwarmState) int {
	h := ss.FitnessHistory
	n := len(h)
	if n < 2 {
		return 0
	}

	stagnant := 0
	bestSeen := h[n-1].Best
	for i := n - 2; i >= 0 && i >= n-10; i-- {
		if h[i].Best >= bestSeen {
			stagnant++
			bestSeen = h[i].Best
		} else {
			break
		}
	}
	return stagnant
}
