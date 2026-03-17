package swarm

import (
	"math/rand"
	"testing"
)

func TestInitHebbian(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNeuro(ss)
	InitHebbian(ss)

	if ss.Hebbian == nil {
		t.Fatal("hebbian should be initialized")
	}
	if len(ss.Hebbian.Traces) != 10 {
		t.Fatalf("expected 10 traces, got %d", len(ss.Hebbian.Traces))
	}
	// Each bot should have a learning rate
	for i, tr := range ss.Hebbian.Traces {
		if tr.LearningRate <= 0 {
			t.Fatalf("bot %d: learning rate should be > 0", i)
		}
	}
}

func TestClearHebbian(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.HebbianOn = true
	InitNeuro(ss)
	InitHebbian(ss)
	ClearHebbian(ss)

	if ss.Hebbian != nil {
		t.Fatal("should be nil after clear")
	}
	if ss.HebbianOn {
		t.Fatal("should be false")
	}
}

func TestTickHebbian(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNeuro(ss)
	InitHebbian(ss)

	// Run neuro forward first to populate activations
	for i := range ss.Bots {
		inputs := BuildNeuroInputs(&ss.Bots[i], ss)
		NeuroForward(ss.Bots[i].Brain, inputs)
	}

	// Set some bots in rewarding states
	ss.Bots[0].CarryingPkg = 1
	ss.Bots[0].NearestDropoffDist = 30
	ss.Bots[1].NearestPickupDist = 20

	initialWeight := ss.Bots[0].Brain.Weights[0]
	for tick := 0; tick < 50; tick++ {
		// Forward pass
		for i := range ss.Bots {
			inputs := BuildNeuroInputs(&ss.Bots[i], ss)
			NeuroForward(ss.Bots[i].Brain, inputs)
		}
		TickHebbian(ss)
	}

	// Weights should have changed
	if ss.Bots[0].Brain.Weights[0] == initialWeight {
		t.Fatal("weights should change through Hebbian learning")
	}
}

func TestTickHebbianNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickHebbian(ss) // should not panic
}

func TestComputeReward(t *testing.T) {
	bot := &SwarmBot{CarryingPkg: 1, Speed: 1.0, NearestDropoffDist: 30}
	reward := computeReward(bot)
	if reward <= 0 {
		t.Fatal("carrying near dropoff should have positive reward")
	}

	bot2 := &SwarmBot{CarryingPkg: -1, Speed: 0, NearestPickupDist: 500}
	reward2 := computeReward(bot2)
	if reward2 >= 0 {
		t.Fatal("standing still far from everything should have negative reward")
	}
}

func TestWeightToNeurons(t *testing.T) {
	pre, post := weightToNeurons(0)
	if pre != 0 || post != 0 {
		t.Fatal("weight 0 should map to pre=0, post=0")
	}

	// Last IH weight
	lastIH := NeuroInputs*NeuroHidden - 1
	pre, post = weightToNeurons(lastIH)
	if pre != NeuroInputs-1 || post != NeuroHidden-1 {
		t.Fatalf("last IH weight: expected pre=%d post=%d, got pre=%d post=%d",
			NeuroInputs-1, NeuroHidden-1, pre, post)
	}

	// First HO weight
	firstHO := NeuroInputs * NeuroHidden
	pre, post = weightToNeurons(firstHO)
	if pre != NeuroInputs {
		t.Fatalf("first HO weight: expected pre=%d, got %d", NeuroInputs, pre)
	}
}

func TestEvolveHebbianRates(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNeuro(ss)
	InitHebbian(ss)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveHebbianRates(ss, sorted)

	if ss.Hebbian.Generation != 1 {
		t.Fatalf("expected generation 1, got %d", ss.Hebbian.Generation)
	}
}

func TestHebbianAvgDelta(t *testing.T) {
	if HebbianAvgDelta(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestHebbianAvgLearningRate(t *testing.T) {
	if HebbianAvgLearningRate(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	hs := &HebbianState{
		Traces: []HebbianTrace{
			{LearningRate: 0.01},
			{LearningRate: 0.03},
		},
	}
	avg := HebbianAvgLearningRate(hs)
	if avg < 0.019 || avg > 0.021 {
		t.Fatalf("expected ~0.02, got %.4f", avg)
	}
}
