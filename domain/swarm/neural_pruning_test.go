package swarm

import (
	"math/rand"
	"testing"
)

func TestInitNeuralPruning(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitNeuralPruning(ss)

	np := ss.NeuralPruning
	if np == nil {
		t.Fatal("neural pruning should be initialized")
	}
	if len(np.Brains) != 15 {
		t.Fatalf("expected 15 brains, got %d", len(np.Brains))
	}
	for i, b := range np.Brains {
		if len(b.Connections) != 50 {
			t.Fatalf("bot %d: expected 50 connections, got %d", i, len(b.Connections))
		}
	}
}

func TestClearNeuralPruning(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.NeuralPruningOn = true
	InitNeuralPruning(ss)
	ClearNeuralPruning(ss)

	if ss.NeuralPruning != nil {
		t.Fatal("should be nil")
	}
	if ss.NeuralPruningOn {
		t.Fatal("should be false")
	}
}

func TestTickNeuralPruning(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNeuralPruning(ss)

	for i := range ss.Bots {
		ss.Bots[i].Speed = SwarmBotSpeed
		ss.Bots[i].NearestPickupDist = 50
	}

	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		TickNeuralPruning(ss)
	}

	np := ss.NeuralPruning
	// Some connections should have been pruned
	if np.TotalPruned == 0 {
		t.Fatal("should have pruned some connections")
	}
	if np.AvgConnections >= 50 {
		t.Fatal("average connections should be less than initial 50")
	}
}

func TestTickNeuralPruningNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickNeuralPruning(ss) // should not panic
}

func TestPruneConnections(t *testing.T) {
	np := &NeuralPruningState{
		Brains: []PrunableBrain{
			{
				InputSize:  5,
				HiddenSize: 8,
				OutputSize: 3,
				Connections: make([]NeuralConnection, 20),
			},
		},
		PruneThreshold: 0.05,
	}

	for c := range np.Brains[0].Connections {
		np.Brains[0].Connections[c].Alive = true
		if c < 5 {
			np.Brains[0].Connections[c].Strength = 0.01 // weak → prune
		} else {
			np.Brains[0].Connections[c].Strength = 0.5 // strong → keep
		}
	}

	pruned := pruneConnections(np)
	if pruned == 0 {
		t.Fatal("should have pruned some weak connections")
	}
}

func TestEvolveNeuralPruning(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNeuralPruning(ss)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveNeuralPruning(ss, sorted)
	if ss.NeuralPruning.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.NeuralPruning.Generation)
	}
}

func TestPruningAvgConnections(t *testing.T) {
	if PruningAvgConnections(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestPruningTotalPruned(t *testing.T) {
	if PruningTotalPruned(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestPruningEfficiency(t *testing.T) {
	if PruningEfficiency(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestForwardPrunable(t *testing.T) {
	brain := &PrunableBrain{
		InputSize:  5,
		HiddenSize: 8,
		OutputSize: 3,
		Connections: []NeuralConnection{
			{From: 0, To: 5, Weight: 1.0, Strength: 0.5, Alive: true},
			{From: 5, To: 13, Weight: 1.0, Strength: 0.5, Alive: true},
		},
	}

	bot := &SwarmBot{
		NearestPickupDist: 100,
		Speed:             SwarmBotSpeed,
	}

	outputs := forwardPrunable(brain, bot)
	// Outputs should be valid (tanh range)
	for _, o := range outputs {
		if o < -1 || o > 1 {
			t.Fatalf("output %.2f out of tanh range", o)
		}
	}
}
