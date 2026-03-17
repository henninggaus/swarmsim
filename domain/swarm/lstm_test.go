package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestLSTMWeightCountCorrect(t *testing.T) {
	expected := 4*(NeuroInputs+LSTMHidden)*LSTMHidden + LSTMHidden*NeuroOutputs
	if LSTMWeights != expected {
		t.Errorf("LSTMWeights should be %d, got %d", expected, LSTMWeights)
	}
	if LSTMWeights != 288 {
		t.Errorf("LSTMWeights should be 288, got %d", LSTMWeights)
	}
}

func TestSigmoid(t *testing.T) {
	if math.Abs(sigmoid(0)-0.5) > 1e-10 {
		t.Errorf("sigmoid(0) should be 0.5, got %f", sigmoid(0))
	}
	if sigmoid(100) < 0.99 {
		t.Errorf("sigmoid(100) should be ~1.0, got %f", sigmoid(100))
	}
	if sigmoid(-100) > 0.01 {
		t.Errorf("sigmoid(-100) should be ~0.0, got %f", sigmoid(-100))
	}
}

func TestLSTMForwardAllZero(t *testing.T) {
	brain := &LSTMBrain{}
	var inputs [NeuroInputs]float64
	actionIdx := LSTMForward(brain, inputs)
	// With all zeros, output should be deterministic
	if actionIdx < 0 || actionIdx >= NeuroOutputs {
		t.Errorf("ActionIdx out of range: %d", actionIdx)
	}
}

func TestLSTMForwardPersistentState(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	brain := &LSTMBrain{}
	// Random weights
	scale := 2.0 / math.Sqrt(float64(LSTMConcatSize))
	for w := 0; w < LSTMWeights; w++ {
		brain.Weights[w] = (rng.Float64() - 0.5) * scale
	}

	var inputs [NeuroInputs]float64
	inputs[0] = 0.5
	inputs[11] = 1.0 // bias

	// First call
	action1 := LSTMForward(brain, inputs)
	hidden1 := brain.HiddenState

	// Second call with same input should produce different hidden state
	action2 := LSTMForward(brain, inputs)
	hidden2 := brain.HiddenState

	// Hidden state should differ (LSTM has memory)
	stateChanged := false
	for h := 0; h < LSTMHidden; h++ {
		if math.Abs(hidden1[h]-hidden2[h]) > 1e-10 {
			stateChanged = true
			break
		}
	}
	if !stateChanged {
		t.Error("Hidden state should change between consecutive calls (LSTM has memory)")
	}
	_ = action1
	_ = action2
}

func TestLSTMForwardResetState(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	brain := &LSTMBrain{}
	scale := 2.0 / math.Sqrt(float64(LSTMConcatSize))
	for w := 0; w < LSTMWeights; w++ {
		brain.Weights[w] = (rng.Float64() - 0.5) * scale
	}

	var inputs [NeuroInputs]float64
	inputs[0] = 0.5
	inputs[11] = 1.0

	// Run a few steps to build up state
	for i := 0; i < 5; i++ {
		LSTMForward(brain, inputs)
	}

	// Reset
	ResetLSTMState(brain)

	// Cell and Hidden should be zero
	for h := 0; h < LSTMHidden; h++ {
		if brain.CellState[h] != 0 {
			t.Errorf("CellState[%d] should be 0 after reset, got %f", h, brain.CellState[h])
		}
		if brain.HiddenState[h] != 0 {
			t.Errorf("HiddenState[%d] should be 0 after reset, got %f", h, brain.HiddenState[h])
		}
	}

	// After reset, first forward should match a fresh brain with same weights
	freshBrain := &LSTMBrain{}
	freshBrain.Weights = brain.Weights
	a1 := LSTMForward(brain, inputs)
	a2 := LSTMForward(freshBrain, inputs)
	if a1 != a2 {
		t.Errorf("After reset, forward should match fresh brain: %d != %d", a1, a2)
	}
}

func TestLSTMForwardOutputRange(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	brain := &LSTMBrain{}
	scale := 2.0 / math.Sqrt(float64(LSTMConcatSize))
	for w := 0; w < LSTMWeights; w++ {
		brain.Weights[w] = (rng.Float64() - 0.5) * scale
	}

	var inputs [NeuroInputs]float64
	for i := range inputs {
		inputs[i] = rng.Float64()
	}

	for step := 0; step < 100; step++ {
		LSTMForward(brain, inputs)
		for o := 0; o < NeuroOutputs; o++ {
			if math.IsNaN(brain.OutputAct[o]) || math.IsInf(brain.OutputAct[o], 0) {
				t.Fatalf("Output[%d] is NaN/Inf at step %d: %f", o, step, brain.OutputAct[o])
			}
		}
	}
}

func TestInitLSTM(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 20),
		BotCount: 20,
		Rng:      rng,
	}
	InitLSTM(ss)

	for i := range ss.Bots {
		if ss.Bots[i].LSTMBrain == nil {
			t.Errorf("Bot %d should have LSTMBrain after init", i)
		}
	}
	if ss.LSTMGeneration != 0 {
		t.Errorf("Generation should be 0, got %d", ss.LSTMGeneration)
	}
}

func TestClearLSTM(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 10),
		BotCount: 10,
		Rng:      rng,
	}
	InitLSTM(ss)
	ClearLSTM(ss)

	for i := range ss.Bots {
		if ss.Bots[i].LSTMBrain != nil {
			t.Errorf("Bot %d LSTMBrain should be nil after clear", i)
		}
	}
}

func TestRunLSTMEvolutionMinBots(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 2), // too few
		BotCount: 2,
		Rng:      rng,
	}
	// Should not panic
	RunLSTMEvolution(ss)
}

func TestRunLSTMEvolutionPreservesElite(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 20),
		BotCount: 20,
		Rng:      rng,
		ArenaW:   800,
		ArenaH:   800,
		LSTMEnabled: true,
	}
	InitLSTM(ss)

	// Give bot 0 very high fitness
	ss.Bots[0].Stats.TotalDeliveries = 100
	ss.Bots[0].Stats.TotalPickups = 50

	// Save elite weights
	eliteWeights := ss.Bots[0].LSTMBrain.Weights

	RunLSTMEvolution(ss)

	// Check that elite weights survived somewhere in the population
	found := false
	for i := range ss.Bots {
		if ss.Bots[i].LSTMBrain != nil && ss.Bots[i].LSTMBrain.Weights == eliteWeights {
			found = true
			break
		}
	}
	if !found {
		t.Error("Elite weights should be preserved in next generation")
	}
}

func TestRunLSTMEvolutionResetsCellState(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 20),
		BotCount: 20,
		Rng:      rng,
		ArenaW:   800,
		ArenaH:   800,
		LSTMEnabled: true,
	}
	InitLSTM(ss)

	// Build up some state
	var inputs [NeuroInputs]float64
	inputs[11] = 1.0
	for i := range ss.Bots {
		if ss.Bots[i].LSTMBrain != nil {
			for step := 0; step < 10; step++ {
				LSTMForward(ss.Bots[i].LSTMBrain, inputs)
			}
		}
		ss.Bots[i].Stats.TotalDeliveries = rng.Intn(10)
		ss.Bots[i].Stats.TicksAlive = 100
	}

	RunLSTMEvolution(ss)

	// All cell/hidden states should be zeroed
	for i := range ss.Bots {
		if ss.Bots[i].LSTMBrain == nil {
			continue
		}
		for h := 0; h < LSTMHidden; h++ {
			if ss.Bots[i].LSTMBrain.CellState[h] != 0 {
				t.Errorf("Bot %d CellState[%d] should be 0 after evolution, got %f",
					i, h, ss.Bots[i].LSTMBrain.CellState[h])
			}
		}
	}
}

func TestResetLSTMStateNil(t *testing.T) {
	// Should not panic
	ResetLSTMState(nil)
}

func TestLSTMForwardGateSaturation(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	brain := &LSTMBrain{}
	// Set all weights randomly, then set large forget gate weights
	scale := 2.0 / math.Sqrt(float64(LSTMConcatSize))
	for w := 0; w < LSTMWeights; w++ {
		brain.Weights[w] = (rng.Float64() - 0.5) * scale
	}
	// Saturate forget gate → forget gate ≈ 1.0 → keep cell state
	for w := LSTMGateW; w < 2*LSTMGateW; w++ {
		brain.Weights[w] = 5.0
	}

	var inputs [NeuroInputs]float64
	inputs[0] = 0.8
	inputs[11] = 1.0

	// Run several steps to build up cell state
	for step := 0; step < 5; step++ {
		LSTMForward(brain, inputs)
	}
	cell5 := brain.CellState

	// Cell state should be non-zero after multiple steps with saturated forget gate
	anyNonZero := false
	for h := 0; h < LSTMHidden; h++ {
		if math.Abs(cell5[h]) > 0.01 {
			anyNonZero = true
			break
		}
	}
	if !anyNonZero {
		t.Error("With high forget gate and active inputs, cell state should accumulate")
	}
}
