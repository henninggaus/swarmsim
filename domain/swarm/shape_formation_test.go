package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestGenerateShapePositionsCircle(t *testing.T) {
	pos := GenerateShapePositions(ShapeCircle, 10, 400, 400, 200)
	if len(pos) != 10 {
		t.Fatalf("expected 10 positions, got %d", len(pos))
	}
	// All points should be ~200 from center
	for i, p := range pos {
		dx := p[0] - 400
		dy := p[1] - 400
		dist := math.Sqrt(dx*dx + dy*dy)
		if math.Abs(dist-200) > 1 {
			t.Fatalf("point %d: dist=%.2f, expected ~200", i, dist)
		}
	}
}

func TestGenerateShapePositionsAllShapes(t *testing.T) {
	for shape := ShapeType(0); shape < ShapeCount; shape++ {
		pos := GenerateShapePositions(shape, 20, 400, 400, 200)
		if len(pos) != 20 {
			t.Fatalf("shape %s: expected 20 positions, got %d", ShapeTypeName(shape), len(pos))
		}
	}
}

func TestShapeTypeName(t *testing.T) {
	if ShapeTypeName(ShapeCircle) != "Kreis" {
		t.Fatal("expected Kreis")
	}
	if ShapeTypeName(ShapeStar) != "Stern" {
		t.Fatal("expected Stern")
	}
	if ShapeTypeName(ShapeCount) != "?" {
		t.Fatal("expected ? for unknown")
	}
}

func TestAllShapeNames(t *testing.T) {
	names := AllShapeNames()
	if len(names) != int(ShapeCount) {
		t.Fatalf("expected %d names, got %d", ShapeCount, len(names))
	}
}

func TestInitShapeFormation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitShapeFormation(ss, ShapeCircle)

	if ss.ShapeFormation == nil {
		t.Fatal("shape formation should be initialized")
	}
	sf := ss.ShapeFormation
	if sf.ActiveShape != ShapeCircle {
		t.Fatal("active shape should be circle")
	}
	if len(sf.TargetPositions) != 20 {
		t.Fatalf("expected 20 target positions, got %d", len(sf.TargetPositions))
	}
	if len(sf.Assigned) != 20 {
		t.Fatalf("expected 20 assignments, got %d", len(sf.Assigned))
	}
	if sf.Radius != 200 {
		t.Fatalf("expected radius 200, got %.0f", sf.Radius)
	}
}

func TestClearShapeFormation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ss.ShapeFormationOn = true
	InitShapeFormation(ss, ShapeSquare)
	ClearShapeFormation(ss)

	if ss.ShapeFormation != nil {
		t.Fatal("shape formation should be nil after clear")
	}
	if ss.ShapeFormationOn {
		t.Fatal("ShapeFormationOn should be false")
	}
}

func TestTickShapeFormation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitShapeFormation(ss, ShapeCircle)

	// Run several ticks
	for i := 0; i < 100; i++ {
		TickShapeFormation(ss)
	}

	sf := ss.ShapeFormation
	if sf.RotationAngle == 0 {
		t.Fatal("rotation should have advanced")
	}
	// Convergence should be > 0 after some ticks
	if sf.Convergence < 0 || sf.Convergence > 1 {
		t.Fatalf("convergence out of range: %.3f", sf.Convergence)
	}
}

func TestTickShapeFormationNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	// Should not panic
	TickShapeFormation(ss)
}

func TestSetShape(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitShapeFormation(ss, ShapeCircle)

	SetShape(ss, ShapeTriangle)
	if ss.ShapeFormation.ActiveShape != ShapeTriangle {
		t.Fatal("shape should be triangle")
	}
	if ss.ShapeFormation.RotationAngle != 0 {
		t.Fatal("rotation should reset on shape change")
	}
}

func TestSetShapeRadius(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitShapeFormation(ss, ShapeLine)

	SetShapeRadius(ss, 100)
	if ss.ShapeFormation.Radius != 100 {
		t.Fatalf("expected radius 100, got %.0f", ss.ShapeFormation.Radius)
	}

	// Clamp min
	SetShapeRadius(ss, 10)
	if ss.ShapeFormation.Radius != 50 {
		t.Fatalf("expected clamped radius 50, got %.0f", ss.ShapeFormation.Radius)
	}

	// Clamp max
	SetShapeRadius(ss, 500)
	if ss.ShapeFormation.Radius != 350 {
		t.Fatalf("expected clamped radius 350, got %.0f", ss.ShapeFormation.Radius)
	}
}

func TestShapeConvergence(t *testing.T) {
	if ShapeConvergence(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	sf := &ShapeFormationState{Convergence: 0.75}
	if ShapeConvergence(sf) != 0.75 {
		t.Fatal("expected 0.75")
	}
}

func TestAssignTargetsUnique(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	targets := GenerateShapePositions(ShapeCircle, 10, 400, 400, 200)
	assigned := assignTargets(ss, targets)

	// Each target should be assigned at most once
	seen := make(map[int]bool)
	for _, a := range assigned {
		if a < 0 {
			continue
		}
		if seen[a] {
			t.Fatalf("target %d assigned to multiple bots", a)
		}
		seen[a] = true
	}
}
