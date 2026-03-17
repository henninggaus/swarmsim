package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestNewMinimalNASGenome(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	nas := NewNASState()
	g := NewMinimalNASGenome(rng, 3, 2, nas)
	if len(g.Nodes) != 5 {
		t.Errorf("expected 5 nodes (3+2), got %d", len(g.Nodes))
	}
	if len(g.Connections) != 6 {
		t.Errorf("expected 6 connections (3*2), got %d", len(g.Connections))
	}
	if g.NextNodeID != 5 {
		t.Errorf("expected NextNodeID=5, got %d", g.NextNodeID)
	}
}

func TestNASForwardBasic(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	nas := NewNASState()
	g := NewMinimalNASGenome(rng, 2, 1, nas)
	outputs := NASForward(g, []float64{1.0, 0.5})
	if len(outputs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(outputs))
	}
	if math.IsNaN(outputs[0]) {
		t.Error("output should not be NaN")
	}
}

func TestNASForwardNilGenome(t *testing.T) {
	outputs := NASForward(nil, []float64{1.0})
	if outputs != nil {
		t.Error("nil genome should return nil outputs")
	}
}

func TestNASMutateWeights(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	nas := NewNASState()
	g := NewMinimalNASGenome(rng, 2, 2, nas)

	origWeights := make([]float64, len(g.Connections))
	for i, c := range g.Connections {
		origWeights[i] = c.Weight
	}

	NASMutate(rng, g, nas)

	changed := 0
	for i, c := range g.Connections {
		if i < len(origWeights) && math.Abs(c.Weight-origWeights[i]) > 0.001 {
			changed++
		}
	}
	if changed == 0 {
		t.Error("mutation should change at least some weights")
	}
}

func TestNASAddNode(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	nas := NewNASState()
	nas.AddNodeRate = 1.0 // force add
	g := NewMinimalNASGenome(rng, 2, 2, nas)
	origNodes := len(g.Nodes)

	NASMutate(rng, g, nas)

	if len(g.Nodes) <= origNodes {
		t.Error("add node mutation should increase node count")
	}
}

func TestNASAddConnection(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	nas := NewNASState()
	nas.AddConnRate = 1.0 // force add
	nas.AddNodeRate = 1.0 // add node first to create hidden
	g := NewMinimalNASGenome(rng, 2, 2, nas)

	for i := 0; i < 5; i++ {
		NASMutate(rng, g, nas)
	}
	// Should have more connections than the minimal 4
	if len(g.Connections) <= 4 {
		t.Error("should have gained connections")
	}
}

func TestNASCrossover(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	nas := NewNASState()
	a := NewMinimalNASGenome(rng, 2, 2, nas)
	b := NewMinimalNASGenome(rng, 2, 2, nas)
	a.Fitness = 100
	b.Fitness = 50

	child := NASCrossover(rng, a, b)
	if child == nil {
		t.Fatal("child should not be nil")
	}
	if len(child.Nodes) == 0 {
		t.Error("child should have nodes")
	}
	if len(child.Connections) == 0 {
		t.Error("child should have connections")
	}
}

func TestNASComplexity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	nas := NewNASState()
	g := NewMinimalNASGenome(rng, 3, 2, nas)

	nodes, conns, enabled := NASComplexity(g)
	if nodes != 5 {
		t.Errorf("expected 5 nodes, got %d", nodes)
	}
	if conns != 6 {
		t.Errorf("expected 6 connections, got %d", conns)
	}
	if enabled != 6 {
		t.Errorf("expected 6 enabled, got %d", enabled)
	}
}

func TestNASComplexityNil(t *testing.T) {
	n, c, e := NASComplexity(nil)
	if n != 0 || c != 0 || e != 0 {
		t.Error("nil should return zeros")
	}
}

func TestNASMutateNilSafe(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	NASMutate(rng, nil, nil) // should not panic
}
