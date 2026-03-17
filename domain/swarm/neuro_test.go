package swarm

import (
	"math"
	"math/rand"
	"testing"
)

// === NeuroForward tests ===

func TestNeuroForwardAllZeroInputs(t *testing.T) {
	brain := &NeuroBrain{}
	// With all zero weights and zero inputs, all outputs should be 0
	var inputs [NeuroInputs]float64
	actionIdx := NeuroForward(brain, inputs)

	// All outputs are 0 → picks index 0 (first max)
	if actionIdx != 0 {
		t.Errorf("expected action 0 with zero brain, got %d", actionIdx)
	}

	// Verify cached activations
	for h := 0; h < NeuroHidden; h++ {
		if brain.HiddenAct[h] != 0 {
			t.Errorf("hidden[%d] should be 0 with zero weights, got %.4f", h, brain.HiddenAct[h])
		}
	}
}

func TestNeuroForwardBiasOnly(t *testing.T) {
	brain := &NeuroBrain{}
	// Set one weight from bias input to hidden[0] to a large value
	// bias is input[11], hidden[0] → weight index = 11*NeuroHidden + 0 = 66
	brain.Weights[11*NeuroHidden+0] = 5.0
	// Set weight from hidden[0] to output[2] (TURN_RIGHT)
	// offset = NeuroInputs*NeuroHidden = 72
	// hidden[0] to output[2] → weight index = 72 + 0*NeuroOutputs + 2 = 74
	brain.Weights[72+0*NeuroOutputs+2] = 10.0

	var inputs [NeuroInputs]float64
	inputs[11] = 1.0 // bias

	actionIdx := NeuroForward(brain, inputs)
	if actionIdx != 2 {
		t.Errorf("expected action 2 (TURN_RIGHT) with strong bias path, got %d", actionIdx)
	}
}

func TestNeuroForwardTanhSaturation(t *testing.T) {
	brain := &NeuroBrain{}
	// Set very large weight: should saturate tanh to ~1.0
	brain.Weights[0] = 100.0 // input[0] → hidden[0]

	var inputs [NeuroInputs]float64
	inputs[0] = 1.0

	NeuroForward(brain, inputs)

	if math.Abs(brain.HiddenAct[0]-1.0) > 0.01 {
		t.Errorf("tanh should saturate near 1.0 with large input, got %.4f", brain.HiddenAct[0])
	}
}

func TestNeuroForwardNegativeWeights(t *testing.T) {
	brain := &NeuroBrain{}
	brain.Weights[0] = -100.0 // input[0] → hidden[0]

	var inputs [NeuroInputs]float64
	inputs[0] = 1.0

	NeuroForward(brain, inputs)

	if math.Abs(brain.HiddenAct[0]-(-1.0)) > 0.01 {
		t.Errorf("tanh should saturate near -1.0 with large negative weight, got %.4f", brain.HiddenAct[0])
	}
}

func TestNeuroForwardInputsCached(t *testing.T) {
	brain := &NeuroBrain{}
	var inputs [NeuroInputs]float64
	inputs[3] = 0.75
	inputs[9] = 0.42

	NeuroForward(brain, inputs)

	if brain.InputVals[3] != 0.75 || brain.InputVals[9] != 0.42 {
		t.Error("input values should be cached in brain.InputVals")
	}
}

func TestNeuroForwardDeterministic(t *testing.T) {
	brain := &NeuroBrain{}
	rng := rand.New(rand.NewSource(42))
	for w := 0; w < NeuroWeights; w++ {
		brain.Weights[w] = (rng.Float64() - 0.5) * 2.0
	}

	var inputs [NeuroInputs]float64
	for i := range inputs {
		inputs[i] = rng.Float64()
	}

	action1 := NeuroForward(brain, inputs)
	outputs1 := brain.OutputAct

	action2 := NeuroForward(brain, inputs)
	outputs2 := brain.OutputAct

	if action1 != action2 {
		t.Error("same inputs + same weights should produce same action")
	}
	for o := 0; o < NeuroOutputs; o++ {
		if outputs1[o] != outputs2[o] {
			t.Errorf("output[%d] not deterministic: %.6f vs %.6f", o, outputs1[o], outputs2[o])
		}
	}
}

// === Xavier initialization range test ===

func TestInitNeuroXavierRange(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNeuro(ss)

	// Xavier init: weights should be in range [-2/sqrt(12), 2/sqrt(12)] ≈ [-0.577, 0.577]
	maxExpected := 2.0 / math.Sqrt(float64(NeuroInputs))
	for i, bot := range ss.Bots {
		if bot.Brain == nil {
			t.Fatalf("bot %d has no brain after InitNeuro", i)
		}
		for w := 0; w < NeuroWeights; w++ {
			if bot.Brain.Weights[w] < -maxExpected || bot.Brain.Weights[w] > maxExpected {
				t.Errorf("bot %d weight %d = %.4f outside Xavier range [%.4f, %.4f]",
					i, w, bot.Brain.Weights[w], -maxExpected, maxExpected)
				break
			}
		}
	}
}

// === Evolution tests ===

func TestRunNeuroEvolutionMinPopulation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 3)
	InitNeuro(ss)

	// Should be a no-op with < 4 bots
	RunNeuroEvolution(ss)
	if ss.NeuroGeneration != 0 {
		t.Error("should not evolve with < 4 bots")
	}
}

func TestRunNeuroEvolutionElitePreserved(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNeuro(ss)

	// Give bot 0 a distinctive weight pattern and highest fitness
	for w := 0; w < NeuroWeights; w++ {
		ss.Bots[0].Brain.Weights[w] = 99.99
	}
	ss.Bots[0].Stats.TotalDeliveries = 100
	ss.Bots[0].Stats.CorrectDeliveries = 100
	ss.Bots[0].Stats.TotalDistance = 10000

	// Give all others low fitness
	for i := 1; i < 20; i++ {
		ss.Bots[i].Stats.TotalDeliveries = 0
	}

	RunNeuroEvolution(ss)

	if ss.NeuroGeneration != 1 {
		t.Errorf("expected generation 1, got %d", ss.NeuroGeneration)
	}

	// The elite (best bot's weights) should be preserved somewhere in population
	found := false
	for _, bot := range ss.Bots {
		if bot.Brain != nil && bot.Brain.Weights[0] == 99.99 {
			found = true
			break
		}
	}
	if !found {
		t.Error("elite bot weights should be preserved without mutation")
	}
}

func TestRunNeuroEvolutionFitnessReset(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNeuro(ss)

	for i := range ss.Bots {
		ss.Bots[i].Stats.TotalDeliveries = i
	}

	RunNeuroEvolution(ss)

	for i, bot := range ss.Bots {
		if bot.Fitness != 0 {
			t.Errorf("bot %d fitness should be 0 after evolution, got %.1f", i, bot.Fitness)
		}
	}
}

func TestRunNeuroEvolutionFreshBots(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNeuro(ss)

	for i := range ss.Bots {
		ss.Bots[i].Stats.TotalDeliveries = i
	}

	RunNeuroEvolution(ss)

	// 10% should be fresh random → 2 bots
	// Verify all bots still have brains
	for i, bot := range ss.Bots {
		if bot.Brain == nil {
			t.Errorf("bot %d has no brain after evolution", i)
		}
	}
}

func TestRunNeuroEvolutionMultipleGenerations(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 30)
	InitNeuro(ss)

	for gen := 0; gen < 10; gen++ {
		// Simulate some activity
		for i := range ss.Bots {
			ss.Bots[i].Stats.TotalDeliveries = rng.Intn(20)
			ss.Bots[i].Stats.TotalDistance = rng.Float64() * 1000
		}
		RunNeuroEvolution(ss)
	}

	if ss.NeuroGeneration != 10 {
		t.Errorf("expected generation 10, got %d", ss.NeuroGeneration)
	}
	if len(ss.FitnessHistory) != 10 {
		t.Errorf("expected 10 fitness records, got %d", len(ss.FitnessHistory))
	}

	// Verify best fitness is tracked
	for i, rec := range ss.FitnessHistory {
		if rec.Best < rec.Avg {
			t.Errorf("gen %d: best (%.1f) should be >= avg (%.1f)", i, rec.Best, rec.Avg)
		}
	}
}

// === BuildNeuroInputs normalization tests ===

func TestBuildNeuroInputsNormalized(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	bot := &ss.Bots[0]

	// Set extreme sensor values
	bot.NearestDist = 500
	bot.NeighborCount = 20
	bot.OnEdge = true
	bot.CarryingPkg = 3
	bot.NearestPickupDist = 1000
	bot.NearestDropoffDist = 1000
	bot.DropoffMatch = true
	bot.NearestPickupHasPkg = true
	bot.ObstacleAhead = true
	bot.LightValue = 200

	inputs := BuildNeuroInputs(bot, ss)

	// All inputs should be in [0, 1] range (except bias which is 1.0)
	for i := 0; i < NeuroInputs; i++ {
		if i == 10 {
			continue // random noise
		}
		if inputs[i] < 0 || inputs[i] > 1.0 {
			t.Errorf("input[%d] (%s) = %.4f out of [0,1] range",
				i, NeuroInputNames[i], inputs[i])
		}
	}

	// Bias should always be 1.0
	if inputs[11] != 1.0 {
		t.Errorf("bias input should be 1.0, got %.4f", inputs[11])
	}
}

func TestBuildNeuroInputsClamping(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	bot := &ss.Bots[0]

	// near_dist clamped at 200
	bot.NearestDist = 999
	inputs := BuildNeuroInputs(bot, ss)
	if inputs[0] != 1.0 {
		t.Errorf("near_dist should clamp to 1.0, got %.4f", inputs[0])
	}

	// neighbors clamped at 10
	bot.NeighborCount = 50
	inputs = BuildNeuroInputs(bot, ss)
	if inputs[1] != 1.0 {
		t.Errorf("neighbors should clamp to 1.0, got %.4f", inputs[1])
	}
}

// === Weight indexing correctness ===

func TestNeuroWeightIndexing(t *testing.T) {
	// Verify the weight layout: [input→hidden | hidden→output]
	if NeuroWeights != NeuroInputs*NeuroHidden+NeuroHidden*NeuroOutputs {
		t.Errorf("weight count mismatch: %d != %d*%d + %d*%d",
			NeuroWeights, NeuroInputs, NeuroHidden, NeuroHidden, NeuroOutputs)
	}

	// Verify constants
	if NeuroInputs != 12 {
		t.Errorf("expected 12 inputs, got %d", NeuroInputs)
	}
	if NeuroHidden != 6 {
		t.Errorf("expected 6 hidden, got %d", NeuroHidden)
	}
	if NeuroOutputs != 8 {
		t.Errorf("expected 8 outputs, got %d", NeuroOutputs)
	}
	if NeuroWeights != 120 {
		t.Errorf("expected 120 weights, got %d", NeuroWeights)
	}
}

// === Specific input → output pathway test ===

func TestNeuroForwardSpecificPathway(t *testing.T) {
	brain := &NeuroBrain{}

	// Create a pathway: if carrying (input[3]=1) → high GOTO_DROPOFF (output[7])
	// Path: input[3] → hidden[2] → output[7]
	// Weight: input[3] to hidden[2] = 3*NeuroHidden + 2 = 20
	brain.Weights[3*NeuroHidden+2] = 3.0
	// Weight: hidden[2] to output[7] = 72 + 2*NeuroOutputs + 7 = 72 + 23 = 95
	brain.Weights[72+2*NeuroOutputs+7] = 5.0

	var inputs [NeuroInputs]float64
	inputs[3] = 1.0  // carrying
	inputs[11] = 1.0 // bias (always set)

	actionIdx := NeuroForward(brain, inputs)
	if actionIdx != 7 {
		t.Errorf("carrying bot should choose GOTO_DROPOFF (7), got %d (%s)",
			actionIdx, NeuroActionNames[actionIdx])
	}
}

// === Adaptive Mutation tests ===

func TestNeuroAdaptiveMutationNoHistory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	rate, strength := neuroAdaptiveMutation(ss)

	// No history → no stagnation → minimum values
	if rate < 0.04 || rate > 0.06 {
		t.Errorf("expected rate ~0.05 with no history, got %.4f", rate)
	}
	if strength < 0.09 || strength > 0.11 {
		t.Errorf("expected strength ~0.10 with no history, got %.4f", strength)
	}
}

func TestNeuroAdaptiveMutationIncreasesDuringStagnation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)

	// Simulate stagnation: 6 generations with same best fitness
	for i := 0; i < 6; i++ {
		ss.FitnessHistory = append(ss.FitnessHistory, FitnessRecord{Best: 100, Avg: 50})
	}

	rate, strength := neuroAdaptiveMutation(ss)

	// After 5+ stagnant gens → should be at max
	if rate < 0.35 {
		t.Errorf("expected high mutation rate during stagnation, got %.4f", rate)
	}
	if strength < 0.70 {
		t.Errorf("expected high mutation strength during stagnation, got %.4f", strength)
	}
}

func TestNeuroAdaptiveMutationLowWhenImproving(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)

	// Simulate consistent improvement
	for i := 0; i < 5; i++ {
		ss.FitnessHistory = append(ss.FitnessHistory, FitnessRecord{Best: float64(i * 100), Avg: float64(i * 50)})
	}

	rate, strength := neuroAdaptiveMutation(ss)

	// Improving → should stay at minimum
	if rate > 0.10 {
		t.Errorf("expected low mutation rate when improving, got %.4f", rate)
	}
	if strength > 0.20 {
		t.Errorf("expected low mutation strength when improving, got %.4f", strength)
	}
}

func TestNeuroStagnantGenerations(t *testing.T) {
	ss := &SwarmState{}

	// No history
	if neuroStagnantGenerations(ss) != 0 {
		t.Error("no history → 0 stagnant")
	}

	// Improving sequence
	ss.FitnessHistory = []FitnessRecord{
		{Best: 10}, {Best: 20}, {Best: 30}, {Best: 40},
	}
	if neuroStagnantGenerations(ss) != 0 {
		t.Errorf("improving → 0 stagnant, got %d", neuroStagnantGenerations(ss))
	}

	// Stagnating sequence
	ss.FitnessHistory = []FitnessRecord{
		{Best: 100}, {Best: 100}, {Best: 100}, {Best: 100},
	}
	stag := neuroStagnantGenerations(ss)
	if stag < 2 {
		t.Errorf("flat fitness → should be stagnant, got %d", stag)
	}
}

func TestNeuroEvolutionAdaptiveMutationIntegration(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNeuro(ss)

	// Run 10 generations with identical fitness → stagnation
	for gen := 0; gen < 10; gen++ {
		for i := range ss.Bots {
			ss.Bots[i].Stats.TotalDeliveries = 5 // same for all
			ss.Bots[i].Stats.TotalDistance = 100
		}
		RunNeuroEvolution(ss)
	}

	// The adaptive mutation should have kicked in:
	// check that weights have more diversity than with fixed 0.3 noise
	// (we can't test exact values, but we can verify it doesn't crash
	// and that generations progress)
	if ss.NeuroGeneration != 10 {
		t.Errorf("expected 10 generations, got %d", ss.NeuroGeneration)
	}
}

func TestClearNeuro(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNeuro(ss)

	ClearNeuro(ss)

	for i, bot := range ss.Bots {
		if bot.Brain != nil {
			t.Errorf("bot %d brain should be nil after ClearNeuro", i)
		}
	}
	if ss.NeuroGeneration != 0 {
		t.Error("generation should be 0 after clear")
	}
}
