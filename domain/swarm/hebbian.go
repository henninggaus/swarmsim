package swarm

import (
	"math"
	"swarmsim/logger"
)

// HebbianState manages lifetime synaptic plasticity for neural bots.
// Instead of only changing weights through evolution, bots modify their
// own neural network weights during their lifetime using Hebbian learning:
// "neurons that fire together wire together."
type HebbianState struct {
	LearningRate  float64 // base Hebbian learning rate (default 0.01)
	DecayRate     float64 // weight decay to prevent runaway growth (default 0.001)
	Modulation    float64 // reward modulation factor (default 0.5)
	MaxWeightDelta float64 // max weight change per tick (default 0.05)
	Eligibility   float64 // eligibility trace decay (default 0.9)

	// Per-bot plasticity state
	Traces []HebbianTrace

	// Stats
	AvgDelta    float64 // average weight change magnitude
	TotalUpdates int
	Generation  int
}

// HebbianTrace holds per-bot learning traces for eligibility-modulated learning.
type HebbianTrace struct {
	// Eligibility traces: record recent pre/post activations
	PrePost   [NeuroWeights]float64 // Hebbian correlation trace
	Reward    float64               // recent reward signal
	PrevInputs  [NeuroInputs]float64
	PrevOutputs [NeuroOutputs]float64
	LearningRate float64 // per-bot learning rate (can evolve)
}

// InitHebbian sets up the Hebbian learning system.
func InitHebbian(ss *SwarmState) {
	n := len(ss.Bots)
	hs := &HebbianState{
		LearningRate:  0.01,
		DecayRate:     0.001,
		Modulation:    0.5,
		MaxWeightDelta: 0.05,
		Eligibility:   0.9,
		Traces:        make([]HebbianTrace, n),
	}

	// Initialize per-bot learning rates with slight variation
	for i := range hs.Traces {
		hs.Traces[i].LearningRate = hs.LearningRate * (0.8 + ss.Rng.Float64()*0.4)
	}

	ss.Hebbian = hs
	logger.Info("HEBBIAN", "Initialisiert: %d Bots, LR=%.3f, Decay=%.4f",
		n, hs.LearningRate, hs.DecayRate)
}

// ClearHebbian disables the Hebbian learning system.
func ClearHebbian(ss *SwarmState) {
	ss.Hebbian = nil
	ss.HebbianOn = false
}

// TickHebbian applies Hebbian plasticity after each neuro forward pass.
func TickHebbian(ss *SwarmState) {
	hs := ss.Hebbian
	if hs == nil {
		return
	}

	n := len(ss.Bots)
	if len(hs.Traces) != n {
		return
	}

	totalDelta := 0.0
	updates := 0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.Brain == nil {
			continue
		}

		trace := &hs.Traces[i]

		// Compute reward signal from bot performance
		reward := computeReward(bot)
		trace.Reward = trace.Reward*hs.Eligibility + reward*(1-hs.Eligibility)

		// Get current activations
		inputs := BuildNeuroInputs(bot, ss)
		outputs := bot.Brain.OutputAct

		// Hebbian update: dW = lr * reward * pre * post
		for w := 0; w < NeuroWeights; w++ {
			// Determine which pre and post neuron this weight connects
			preIdx, postIdx := weightToNeurons(w)
			pre := getActivation(inputs[:], preIdx)
			post := getOutputActivation(outputs[:], postIdx)

			// Update eligibility trace
			trace.PrePost[w] = trace.PrePost[w]*hs.Eligibility + pre*post*(1-hs.Eligibility)

			// Modulated Hebbian: weight change proportional to reward * trace
			delta := trace.LearningRate * trace.Reward * hs.Modulation * trace.PrePost[w]

			// Clamp delta
			if delta > hs.MaxWeightDelta {
				delta = hs.MaxWeightDelta
			}
			if delta < -hs.MaxWeightDelta {
				delta = -hs.MaxWeightDelta
			}

			// Apply weight change with decay
			bot.Brain.Weights[w] += delta
			bot.Brain.Weights[w] *= (1 - hs.DecayRate) // prevent runaway

			totalDelta += math.Abs(delta)
			updates++
		}

		// Store for next tick
		copy(trace.PrevInputs[:], inputs[:])
		copy(trace.PrevOutputs[:], outputs[:])
	}

	if updates > 0 {
		hs.AvgDelta = totalDelta / float64(updates)
	}
	hs.TotalUpdates += updates
}

// computeReward derives a scalar reward from bot performance.
func computeReward(bot *SwarmBot) float64 {
	reward := 0.0

	// Reward for carrying (making progress)
	if bot.CarryingPkg >= 0 && bot.Speed > 0 {
		reward += 0.3
	}

	// Reward for being near pickup when not carrying
	if bot.CarryingPkg < 0 && bot.NearestPickupDist < 50 {
		reward += 0.5
	}

	// Reward for being near dropoff when carrying
	if bot.CarryingPkg >= 0 && bot.NearestDropoffDist < 50 {
		reward += 0.8
	}

	// Penalty for standing still
	if bot.Speed < 0.1 {
		reward -= 0.1
	}

	return reward
}

// weightToNeurons maps a flat weight index to pre/post neuron indices.
func weightToNeurons(w int) (int, int) {
	// First layer: NeuroInputs * NeuroHidden weights
	ihWeights := NeuroInputs * NeuroHidden
	if w < ihWeights {
		pre := w / NeuroHidden
		post := w % NeuroHidden
		return pre, post
	}
	// Second layer: NeuroHidden * NeuroOutputs weights
	w2 := w - ihWeights
	pre := w2 / NeuroOutputs
	post := w2 % NeuroOutputs
	return NeuroInputs + pre, NeuroHidden + post
}

// getActivation returns activation of a neuron by index.
func getActivation(inputs []float64, idx int) float64 {
	if idx < len(inputs) {
		return inputs[idx]
	}
	return 0
}

// getOutputActivation returns output neuron activation.
func getOutputActivation(outputs []float64, idx int) float64 {
	adjusted := idx - NeuroHidden
	if adjusted >= 0 && adjusted < len(outputs) {
		return outputs[adjusted]
	}
	return 0
}

// EvolveHebbianRates evolves per-bot learning rates alongside main evolution.
func EvolveHebbianRates(ss *SwarmState, sortedIndices []int) {
	hs := ss.Hebbian
	if hs == nil {
		return
	}

	n := len(ss.Bots)
	if len(hs.Traces) != n {
		return
	}

	parentCount := n * 20 / 100
	if parentCount < 2 {
		parentCount = 2
	}

	// Save parent learning rates
	parentRates := make([]float64, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parentRates[i] = hs.Traces[sortedIndices[i]].LearningRate
	}

	for rank, botIdx := range sortedIndices {
		if rank < 3 {
			// Elite: keep rate
			continue
		}
		// Inherit from parent with mutation
		p := ss.Rng.Intn(parentCount)
		rate := parentRates[p]
		if ss.Rng.Float64() < 0.2 {
			rate *= 0.8 + ss.Rng.Float64()*0.4 // mutate ±20%
		}
		if rate < 0.001 {
			rate = 0.001
		}
		if rate > 0.1 {
			rate = 0.1
		}
		hs.Traces[botIdx].LearningRate = rate
		// Reset traces for new generation
		hs.Traces[botIdx].PrePost = [NeuroWeights]float64{}
		hs.Traces[botIdx].Reward = 0
	}

	hs.Generation++
	logger.Info("HEBBIAN", "Gen %d: AvgDelta=%.6f, Updates=%d",
		hs.Generation, hs.AvgDelta, hs.TotalUpdates)
}

// HebbianAvgDelta returns average weight change magnitude.
func HebbianAvgDelta(hs *HebbianState) float64 {
	if hs == nil {
		return 0
	}
	return hs.AvgDelta
}

// HebbianAvgLearningRate returns the average per-bot learning rate.
func HebbianAvgLearningRate(hs *HebbianState) float64 {
	if hs == nil || len(hs.Traces) == 0 {
		return 0
	}
	total := 0.0
	for _, t := range hs.Traces {
		total += t.LearningRate
	}
	return total / float64(len(hs.Traces))
}
