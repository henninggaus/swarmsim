package swarm

import (
	"math/rand"
	"testing"
)

func TestInitGRN(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitGRN(ss, 8)

	gs := ss.GRN
	if gs == nil {
		t.Fatal("GRN should be initialized")
	}
	if gs.NumGenes != 8 {
		t.Fatalf("expected 8 genes, got %d", gs.NumGenes)
	}
	if len(gs.Networks) != 10 {
		t.Fatalf("expected 10 networks, got %d", len(gs.Networks))
	}
	// Each network should have 8x8 regulation matrix
	for i, net := range gs.Networks {
		if len(net.Regulation) != 8 {
			t.Fatalf("network %d: expected 8 regulation rows, got %d", i, len(net.Regulation))
		}
		if len(net.Expression) != 8 {
			t.Fatalf("network %d: expected 8 expression values, got %d", i, len(net.Expression))
		}
	}
}

func TestInitGRNClamp(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitGRN(ss, 3) // below min
	if ss.GRN.NumGenes != 6 {
		t.Fatal("should clamp to min 6")
	}
}

func TestClearGRN(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.GRNOn = true
	InitGRN(ss, 8)
	ClearGRN(ss)

	if ss.GRN != nil {
		t.Fatal("should be nil after clear")
	}
	if ss.GRNOn {
		t.Fatal("should be false")
	}
}

func TestTickGRN(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitGRN(ss, 8)

	// Set some sensor values
	ss.Bots[0].CarryingPkg = 1
	ss.Bots[1].NearestPickupDist = 30
	ss.Bots[2].NeighborCount = 8

	initialExpr := make([]float64, 8)
	copy(initialExpr, ss.GRN.Networks[0].Expression)

	for i := 0; i < 20; i++ {
		TickGRN(ss)
	}

	// Expression should have changed
	changed := false
	for j, e := range ss.GRN.Networks[0].Expression {
		if e != initialExpr[j] {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("expression levels should change after ticks")
	}
}

func TestTickGRNNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickGRN(ss) // should not panic
}

func TestEvolveGRN(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitGRN(ss, 8)

	// Create sorted indices by fake fitness
	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveGRN(ss, sorted)

	if ss.GRN.Generation != 1 {
		t.Fatalf("expected generation 1, got %d", ss.GRN.Generation)
	}
}

func TestCloneGRNetwork(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitGRN(ss, 8)

	original := ss.GRN.Networks[0]
	clone := cloneGRNetwork(original)

	// Modify clone, should not affect original
	clone.Regulation[0][0] = 999
	if original.Regulation[0][0] == 999 {
		t.Fatal("clone should be independent of original")
	}
}

func TestGRNGeneCount(t *testing.T) {
	if GRNGeneCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	gs := &GRNState{NumGenes: 12}
	if GRNGeneCount(gs) != 12 {
		t.Fatal("expected 12")
	}
}

func TestGRNExpression(t *testing.T) {
	if GRNExpression(nil, 0) != nil {
		t.Fatal("nil should return nil")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitGRN(ss, 8)

	expr := GRNExpression(ss.GRN, 0)
	if len(expr) != 8 {
		t.Fatalf("expected 8 expression values, got %d", len(expr))
	}
	if GRNExpression(ss.GRN, 10) != nil {
		t.Fatal("out of bounds should return nil")
	}
}

func TestGRNConnectivity(t *testing.T) {
	if GRNConnectivity(nil) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitGRN(ss, 8)

	conn := GRNConnectivity(ss.GRN)
	if conn <= 0 || conn >= 1 {
		t.Fatalf("connectivity should be between 0 and 1 (sparse), got %.3f", conn)
	}
}

func TestExpressionClamped(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitGRN(ss, 8)
	gs := ss.GRN

	// Force extreme expression
	for i := range gs.Networks[0].Expression {
		gs.Networks[0].Expression[i] = 100
	}

	TickGRN(ss)

	for i, e := range gs.Networks[0].Expression {
		if e < 0 || e > gs.MaxExpression {
			t.Fatalf("expression[%d]=%.3f out of [0, %.1f]", i, e, gs.MaxExpression)
		}
	}
}
