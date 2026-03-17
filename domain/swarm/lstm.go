package swarm

import (
	"math"
	"sort"
	"swarmsim/logger"
)

// ═══════════════════════════════════════════════════════════
// LSTM — Long Short-Term Memory Netz pro Bot
// ═══════════════════════════════════════════════════════════
//
// Alternative zum feedforward NeuroBrain mit temporalem
// Gedaechtnis. Jeder Bot hat persistente Cell- und Hidden-
// States die zwischen Ticks erhalten bleiben.
//
// Architektur:
//   12 Sensor-Inputs → 4 LSTM-Zellen → 8 Action-Outputs
//
// LSTM-Gates:
//   - Input Gate: welche neuen Infos aufnehmen?
//   - Forget Gate: welche alten Infos vergessen?
//   - Cell Gate: Kandidaten-Werte fuer Cell State
//   - Output Gate: was vom Cell State ausgeben?
//
// Gewichte: 4 Gates × (12+4) × 4 + 4 × 8 = 256 + 32 = 288
//
// Evolution identisch zu Neuro: Crossover + adaptive Mutation,
// aber nach jeder Generation werden Cell/Hidden States genullt.

const (
	LSTMHidden     = 4
	LSTMConcatSize = NeuroInputs + LSTMHidden                  // 16
	LSTMGateW      = LSTMConcatSize * LSTMHidden               // 64 per gate
	LSTMTotalGateW = 4 * LSTMGateW                             // 256
	LSTMOutputW    = LSTMHidden * NeuroOutputs                  // 32
	LSTMWeights    = LSTMTotalGateW + LSTMOutputW               // 288
)

// LSTMBrain holds LSTM weights and persistent state for a single bot.
type LSTMBrain struct {
	Weights [LSTMWeights]float64

	// Persistent state (carries across ticks, reset on evolution)
	CellState   [LSTMHidden]float64
	HiddenState [LSTMHidden]float64

	// Cached activations for visualization
	InputGate  [LSTMHidden]float64
	ForgetGate [LSTMHidden]float64
	OutputGate [LSTMHidden]float64
	CellGate   [LSTMHidden]float64 // candidate cell values (tanh)
	OutputAct  [NeuroOutputs]float64
	InputVals  [NeuroInputs]float64
	ActionIdx  int
}

// sigmoid activation function.
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// LSTMForward runs one LSTM time step.
// Unlike NeuroForward, this mutates persistent state (CellState, HiddenState).
func LSTMForward(brain *LSTMBrain, inputs [NeuroInputs]float64) int {
	brain.InputVals = inputs

	// Concatenate [inputs ; hiddenState] → length 16
	var concat [LSTMConcatSize]float64
	copy(concat[:NeuroInputs], inputs[:])
	copy(concat[NeuroInputs:], brain.HiddenState[:])

	// Compute all 4 gates: input(0), forget(1), cell(2), output(3)
	var gates [4][LSTMHidden]float64
	for g := 0; g < 4; g++ {
		gateOffset := g * LSTMGateW
		for h := 0; h < LSTMHidden; h++ {
			sum := 0.0
			for j := 0; j < LSTMConcatSize; j++ {
				sum += concat[j] * brain.Weights[gateOffset+j*LSTMHidden+h]
			}
			gates[g][h] = sum
		}
	}

	// Apply activations
	for h := 0; h < LSTMHidden; h++ {
		brain.InputGate[h] = sigmoid(gates[0][h])
		brain.ForgetGate[h] = sigmoid(gates[1][h])
		brain.CellGate[h] = math.Tanh(gates[2][h])
		brain.OutputGate[h] = sigmoid(gates[3][h])
	}

	// Update cell state: C_t = f * C_{t-1} + i * c_candidate
	for h := 0; h < LSTMHidden; h++ {
		brain.CellState[h] = brain.ForgetGate[h]*brain.CellState[h] +
			brain.InputGate[h]*brain.CellGate[h]
	}

	// Update hidden state: h_t = o * tanh(C_t)
	for h := 0; h < LSTMHidden; h++ {
		brain.HiddenState[h] = brain.OutputGate[h] * math.Tanh(brain.CellState[h])
	}

	// Output layer: HiddenState → 8 outputs
	outOffset := LSTMTotalGateW
	bestIdx := 0
	bestVal := -1e9
	for o := 0; o < NeuroOutputs; o++ {
		sum := 0.0
		for h := 0; h < LSTMHidden; h++ {
			sum += brain.HiddenState[h] * brain.Weights[outOffset+h*NeuroOutputs+o]
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

// ResetLSTMState zeros the cell and hidden state.
func ResetLSTMState(brain *LSTMBrain) {
	if brain == nil {
		return
	}
	brain.CellState = [LSTMHidden]float64{}
	brain.HiddenState = [LSTMHidden]float64{}
}

// InitLSTM initializes LSTM brains for all bots.
func InitLSTM(ss *SwarmState) {
	scale := 2.0 / math.Sqrt(float64(LSTMConcatSize))
	for i := range ss.Bots {
		brain := &LSTMBrain{}
		for w := 0; w < LSTMWeights; w++ {
			brain.Weights[w] = (ss.Rng.Float64() - 0.5) * scale
		}
		// Initialize forget gate bias to ~1.0 (helps LSTM remember by default)
		// Forget gate weights start at offset LSTMGateW (gate index 1)
		// Bias is added via the constant 1.0 in the input vector (bias neuron)
		ss.Bots[i].LSTMBrain = brain
		ss.Bots[i].Fitness = 0
	}
	ss.LSTMGeneration = 0
	ss.LSTMTimer = 0
	ss.FitnessHistory = nil
	logger.Info("LSTM", "Initialisiert: %d Bots × %d Gewichte = %d Parameter total",
		len(ss.Bots), LSTMWeights, len(ss.Bots)*LSTMWeights)
}

// ClearLSTM removes LSTM brains from all bots.
func ClearLSTM(ss *SwarmState) {
	for i := range ss.Bots {
		ss.Bots[i].LSTMBrain = nil
	}
	ss.LSTMGeneration = 0
	ss.LSTMTimer = 0
}

// RunLSTMEvolution performs one generation of LSTM neuroevolution.
func RunLSTMEvolution(ss *SwarmState) {
	n := len(ss.Bots)
	if n < 4 {
		return
	}

	// 1. Evaluate fitness
	fitnesses := make([]float64, n)
	for i := range ss.Bots {
		fitnesses[i] = EvaluateGPFitness(&ss.Bots[i])
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

	// 4. Top 20% are parents
	parentCount := n * 20 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 3
	if eliteCount > parentCount {
		eliteCount = parentCount
	}

	// 5. Save elite weights
	type savedWeights struct {
		weights [LSTMWeights]float64
	}
	eliteWeights := make([]savedWeights, eliteCount)
	for i := 0; i < eliteCount; i++ {
		if ss.Bots[indices[i]].LSTMBrain != nil {
			eliteWeights[i].weights = ss.Bots[indices[i]].LSTMBrain.Weights
		}
	}

	// 6. Save parent weights
	parentWeights := make([]savedWeights, parentCount)
	for i := 0; i < parentCount; i++ {
		if ss.Bots[indices[i]].LSTMBrain != nil {
			parentWeights[i].weights = ss.Bots[indices[i]].LSTMBrain.Weights
		}
	}

	// 7. Adaptive mutation (reuse neuro stagnation logic)
	mutRate, mutStrength := lstmAdaptiveMutation(ss)

	// 8. Generate new population
	freshCount := n * 10 / 100
	if freshCount < 1 {
		freshCount = 1
	}
	crossoverCount := n - eliteCount - freshCount
	scale := 2.0 / math.Sqrt(float64(LSTMConcatSize))

	assigned := 0

	// Genealogy: save old BotIDs for parent tracking
	oldBotIDs := make([]int, n)
	for i := range ss.Bots {
		oldBotIDs[i] = ss.Bots[i].BotID
	}

	// Elite
	for i := 0; i < eliteCount && assigned < n; i++ {
		idx := indices[assigned]
		if ss.Bots[idx].LSTMBrain == nil {
			ss.Bots[idx].LSTMBrain = &LSTMBrain{}
		}
		ss.Bots[idx].LSTMBrain.Weights = eliteWeights[i].weights
		ResetLSTMState(ss.Bots[idx].LSTMBrain)
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
		if ss.Bots[idx].LSTMBrain == nil {
			ss.Bots[idx].LSTMBrain = &LSTMBrain{}
		}
		p1 := ss.Rng.Intn(parentCount)
		p2 := ss.Rng.Intn(parentCount)
		for w := 0; w < LSTMWeights; w++ {
			if ss.Rng.Float64() < 0.5 {
				ss.Bots[idx].LSTMBrain.Weights[w] = parentWeights[p1].weights[w]
			} else {
				ss.Bots[idx].LSTMBrain.Weights[w] = parentWeights[p2].weights[w]
			}
			if ss.Rng.Float64() < mutRate {
				ss.Bots[idx].LSTMBrain.Weights[w] += ss.Rng.NormFloat64() * mutStrength
			}
		}
		ResetLSTMState(ss.Bots[idx].LSTMBrain)
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
		if ss.Bots[idx].LSTMBrain == nil {
			ss.Bots[idx].LSTMBrain = &LSTMBrain{}
		}
		for w := 0; w < LSTMWeights; w++ {
			ss.Bots[idx].LSTMBrain.Weights[w] = (ss.Rng.Float64() - 0.5) * scale
		}
		ResetLSTMState(ss.Bots[idx].LSTMBrain)
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
		RecordGeneration(ss.Genealogy, ss.Bots, ss.LSTMGeneration)
	}

	// 9. Reset lifetime stats
	for i := range ss.Bots {
		ss.Bots[i].Fitness = 0
		ss.Bots[i].Stats = BotLifetimeStats{}
	}

	ss.LSTMGeneration++
	ss.LSTMTimer = 0

	logger.Info("LSTM", "Gen %d — Best: %.0f, Avg: %.0f (Mut: %.0f%%/%.2f, %d Elite + %d Crossover + %d Neue)",
		ss.LSTMGeneration, ss.BestFitness, ss.AvgFitness,
		mutRate*100, mutStrength, eliteCount, crossoverCount, freshCount)
}

// lstmAdaptiveMutation computes mutation rate and strength based on stagnation.
func lstmAdaptiveMutation(ss *SwarmState) (rate, strength float64) {
	const (
		minRate     = 0.05
		maxRate     = 0.40
		minStrength = 0.10
		maxStrength = 0.80
	)

	// Count stagnant generations
	h := ss.FitnessHistory
	n := len(h)
	if n < 2 {
		return minRate, minStrength
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

	t := float64(stagnant) / 5.0
	if t > 1.0 {
		t = 1.0
	}
	rate = minRate + t*(maxRate-minRate)
	strength = minStrength + t*(maxStrength-minStrength)
	return rate, strength
}
