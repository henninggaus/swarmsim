package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestComputeFormationTargetsCircle(t *testing.T) {
	ft := ComputeFormationTargets(FormCircle, 4, 400, 400, 100)
	if len(ft.Targets) != 4 {
		t.Fatalf("expected 4 targets, got %d", len(ft.Targets))
	}
	// All targets should be ~100px from center
	for i, tgt := range ft.Targets {
		dx := tgt[0] - 400
		dy := tgt[1] - 400
		dist := math.Sqrt(dx*dx + dy*dy)
		if math.Abs(dist-100) > 1 {
			t.Errorf("target %d: expected dist ~100, got %f", i, dist)
		}
	}
}

func TestComputeFormationTargetsLine(t *testing.T) {
	ft := ComputeFormationTargets(FormLine, 5, 400, 300, 200)
	if len(ft.Targets) != 5 {
		t.Fatalf("expected 5 targets, got %d", len(ft.Targets))
	}
	// All targets should have Y=300
	for i, tgt := range ft.Targets {
		if math.Abs(tgt[1]-300) > 0.1 {
			t.Errorf("target %d: Y should be 300, got %f", i, tgt[1])
		}
	}
	// Should span from 200 to 600
	if math.Abs(ft.Targets[0][0]-200) > 1 {
		t.Errorf("first target X should be ~200, got %f", ft.Targets[0][0])
	}
	if math.Abs(ft.Targets[4][0]-600) > 1 {
		t.Errorf("last target X should be ~600, got %f", ft.Targets[4][0])
	}
}

func TestComputeFormationTargetsGrid(t *testing.T) {
	ft := ComputeFormationTargets(FormGrid, 9, 400, 400, 100)
	if len(ft.Targets) != 9 {
		t.Fatalf("expected 9 targets, got %d", len(ft.Targets))
	}
}

func TestComputeFormationTargetsV(t *testing.T) {
	ft := ComputeFormationTargets(FormV, 7, 400, 400, 100)
	if len(ft.Targets) != 7 {
		t.Fatalf("expected 7 targets, got %d", len(ft.Targets))
	}
	// Leader (index 0) should be at center
	if math.Abs(ft.Targets[0][0]-400) > 1 {
		t.Errorf("V leader should be at center X, got %f", ft.Targets[0][0])
	}
}

func TestComputeFormationTargetsSpiral(t *testing.T) {
	ft := ComputeFormationTargets(FormSpiral, 10, 400, 400, 150)
	if len(ft.Targets) != 10 {
		t.Fatalf("expected 10 targets, got %d", len(ft.Targets))
	}
	// First target should be near center (small radius)
	dx := ft.Targets[0][0] - 400
	dy := ft.Targets[0][1] - 400
	dist0 := math.Sqrt(dx*dx + dy*dy)
	// Last target should be far from center
	dx = ft.Targets[9][0] - 400
	dy = ft.Targets[9][1] - 400
	dist9 := math.Sqrt(dx*dx + dy*dy)
	if dist9 <= dist0 {
		t.Errorf("spiral should expand outward: dist0=%f dist9=%f", dist0, dist9)
	}
}

func TestComputeFormationTargetsEmpty(t *testing.T) {
	ft := ComputeFormationTargets(FormCircle, 0, 400, 400, 100)
	if len(ft.Targets) != 0 {
		t.Error("empty bot count should return no targets")
	}
}

func TestMorphToFormation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	// Scatter bots randomly
	for i := range ss.Bots {
		ss.Bots[i].X = rng.Float64() * 800
		ss.Bots[i].Y = rng.Float64() * 800
	}

	ft := ComputeFormationTargets(FormCircle, 10, 400, 400, 100)

	// First morph: bots should move closer
	dist1 := MorphToFormation(ss, ft, 0.5)

	// Second morph: should be even closer
	dist2 := MorphToFormation(ss, ft, 0.5)
	if dist2 >= dist1 {
		t.Errorf("distance should decrease: %f >= %f", dist2, dist1)
	}
}

func TestMorphToFormationEmptyTargets(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ft := FormationTarget{}
	dist := MorphToFormation(ss, ft, 0.1)
	if dist != 0 {
		t.Error("empty targets should return 0 distance")
	}
}
