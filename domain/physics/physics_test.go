package physics

import (
	"math"
	"math/rand"
	"testing"
)

// --- SpatialHash tests ---

func TestSpatialHashInsertAndQuery(t *testing.T) {
	sh := NewSpatialHash(800, 800, 50)

	// Insert three entities
	sh.Insert(0, 100, 100)
	sh.Insert(1, 110, 110) // close to 0
	sh.Insert(2, 500, 500) // far away

	// Query around entity 0 with radius 50 — should find 0 and 1, not 2
	results := sh.Query(100, 100, 50)
	found0, found1, found2 := false, false, false
	for _, id := range results {
		switch id {
		case 0:
			found0 = true
		case 1:
			found1 = true
		case 2:
			found2 = true
		}
	}
	if !found0 {
		t.Error("expected to find entity 0 near (100,100)")
	}
	if !found1 {
		t.Error("expected to find entity 1 near (100,100)")
	}
	if found2 {
		t.Error("entity 2 at (500,500) should not be in range of (100,100) r=50")
	}
}

func TestSpatialHashClear(t *testing.T) {
	sh := NewSpatialHash(400, 400, 50)
	sh.Insert(0, 100, 100)
	sh.Insert(1, 200, 200)
	sh.Clear()
	results := sh.Query(100, 100, 200)
	if len(results) != 0 {
		t.Errorf("expected 0 results after clear, got %d", len(results))
	}
}

func TestSpatialHashEdgeClamping(t *testing.T) {
	sh := NewSpatialHash(400, 400, 50)
	// Inserting at negative coords should clamp to cell 0
	sh.Insert(42, -10, -10)
	results := sh.Query(0, 0, 50)
	found := false
	for _, id := range results {
		if id == 42 {
			found = true
		}
	}
	if !found {
		t.Error("entity at negative coords should be clamped and queryable near origin")
	}
}

func TestSpatialHashQueryOutOfBounds(t *testing.T) {
	sh := NewSpatialHash(400, 400, 50)
	sh.Insert(0, 200, 200)
	// Query at origin with small radius should not find distant entity
	results := sh.Query(0, 0, 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results for distant query, got %d", len(results))
	}
}

func TestSpatialHashMultipleInSameCell(t *testing.T) {
	sh := NewSpatialHash(400, 400, 100)
	// All three are in the same cell (cell size 100)
	sh.Insert(0, 10, 10)
	sh.Insert(1, 20, 20)
	sh.Insert(2, 30, 30)
	results := sh.Query(15, 15, 50)
	if len(results) != 3 {
		t.Errorf("expected 3 entities in same cell, got %d", len(results))
	}
}

// --- CircleRectCollision tests ---

func TestCircleRectCollision_Overlap(t *testing.T) {
	// Circle at (50, 50) with radius 20, rect at (40, 40) size 30x30
	hit, _, _ := CircleRectCollision(50, 50, 20, 40, 40, 30, 30)
	if !hit {
		t.Error("expected collision when circle center is inside rect")
	}
}

func TestCircleRectCollision_NoOverlap(t *testing.T) {
	// Circle at (0, 0) radius 5, rect at (100, 100) size 10x10
	hit, _, _ := CircleRectCollision(0, 0, 5, 100, 100, 10, 10)
	if hit {
		t.Error("expected no collision when circle and rect are far apart")
	}
}

func TestCircleRectCollision_EdgeTouch(t *testing.T) {
	// Circle at (0, 50) radius 10, rect at (10, 40) size 20x20
	// Nearest rect point to circle center is (10, 50), distance = 10 = radius
	hit, _, _ := CircleRectCollision(0, 50, 10, 10, 40, 20, 20)
	// dist == cr -> should NOT collide (strict <)
	if hit {
		t.Error("edge-touching should not count as collision (strict less-than)")
	}
}

func TestCircleRectCollision_JustInside(t *testing.T) {
	// Circle at (0, 50) radius 11, rect at (10, 40) size 20x20
	// Nearest point = (10, 50), distance = 10 < 11
	hit, nx, ny := CircleRectCollision(0, 50, 11, 10, 40, 20, 20)
	if !hit {
		t.Error("expected collision when circle overlaps rect edge")
	}
	if nx != 10 || ny != 50 {
		t.Errorf("expected nearest point (10, 50), got (%.1f, %.1f)", nx, ny)
	}
}

// --- ResolveCircleRectOverlap ---

func TestResolveCircleRectOverlap_Push(t *testing.T) {
	// Circle at (25, 50) radius 10, rect at (30, 40) size 20x20
	// Nearest rect point to (25, 50) = (30, 50), dist = 5, overlap = 10-5 = 5
	nx, ny := ResolveCircleRectOverlap(25, 50, 10, 30, 40, 20, 20)
	// Should push left (negative x direction)
	if nx >= 25 {
		t.Errorf("expected to be pushed left from 25, got %.1f", nx)
	}
	if math.Abs(ny-50) > 0.01 {
		t.Errorf("expected y to stay at 50, got %.1f", ny)
	}
}

func TestResolveCircleRectOverlap_NoOverlap(t *testing.T) {
	// Circle at (0, 0) radius 5, rect at (100, 100) size 10x10
	nx, ny := ResolveCircleRectOverlap(0, 0, 5, 100, 100, 10, 10)
	if nx != 0 || ny != 0 {
		t.Errorf("no overlap should return original pos, got (%.1f, %.1f)", nx, ny)
	}
}

func TestResolveCircleRectOverlap_CenterInside(t *testing.T) {
	// Circle center exactly on nearest point (dist=0): special push logic
	nx, ny := ResolveCircleRectOverlap(50, 50, 10, 40, 40, 30, 30)
	// Should be pushed away from rect
	dist := math.Sqrt((nx-50)*(nx-50) + (ny-50)*(ny-50))
	if dist < 1 {
		t.Error("expected to be pushed away from center when dist=0")
	}
}

// --- Normalize tests ---

func TestNormalize(t *testing.T) {
	tests := []struct {
		dx, dy   float64
		expectNx float64
		expectNy float64
	}{
		{3, 4, 0.6, 0.8},
		{0, 5, 0, 1},
		{-1, 0, -1, 0},
		{0, 0, 0, 0}, // zero vector
	}
	for _, tc := range tests {
		nx, ny := Normalize(tc.dx, tc.dy)
		if math.Abs(nx-tc.expectNx) > 1e-9 || math.Abs(ny-tc.expectNy) > 1e-9 {
			t.Errorf("Normalize(%v,%v) = (%v,%v), want (%v,%v)",
				tc.dx, tc.dy, nx, ny, tc.expectNx, tc.expectNy)
		}
	}
}

// --- Distance tests ---

func TestDistance(t *testing.T) {
	tests := []struct {
		x1, y1, x2, y2 float64
		expected       float64
	}{
		{0, 0, 3, 4, 5},
		{0, 0, 0, 0, 0},
		{1, 1, 4, 5, 5},
	}
	for _, tc := range tests {
		d := Distance(tc.x1, tc.y1, tc.x2, tc.y2)
		if math.Abs(d-tc.expected) > 1e-9 {
			t.Errorf("Distance(%v,%v,%v,%v) = %v, want %v",
				tc.x1, tc.y1, tc.x2, tc.y2, d, tc.expected)
		}
	}
}

// --- Clamp tests ---

func TestClamp(t *testing.T) {
	if Clamp(5, 0, 10) != 5 {
		t.Error("value within range should be unchanged")
	}
	if Clamp(-1, 0, 10) != 0 {
		t.Error("below min should clamp to min")
	}
	if Clamp(15, 0, 10) != 10 {
		t.Error("above max should clamp to max")
	}
}

// --- ClampToBounds tests ---

func TestClampToBounds(t *testing.T) {
	// Inside bounds
	x, y, clamped := ClampToBounds(100, 100, 5, 800, 600)
	if clamped {
		t.Error("position inside bounds should not be clamped")
	}
	if x != 100 || y != 100 {
		t.Error("position should be unchanged")
	}

	// Outside left
	x, _, clamped = ClampToBounds(-10, 100, 5, 800, 600)
	if !clamped {
		t.Error("position outside bounds should be clamped")
	}
	if x != 5 {
		t.Errorf("expected x=5 (radius), got %v", x)
	}

	// Outside bottom
	_, y, clamped = ClampToBounds(100, 700, 5, 800, 600)
	if !clamped {
		t.Error("position below bottom should be clamped")
	}
	if y != 595 {
		t.Errorf("expected y=595 (arenaH-radius), got %v", y)
	}
}

// --- ReflectVelocity tests ---

func TestReflectVelocity(t *testing.T) {
	// Hitting left wall: x - radius <= 0
	vx, vy := ReflectVelocity(2, 100, -5, 3, 5, 800, 600)
	if vx != 2.5 {
		t.Errorf("expected vx = 2.5 after left wall hit, got %v", vx)
	}
	if vy != 3 {
		t.Errorf("vy should be unchanged, got %v", vy)
	}

	// Hitting top wall: y - radius <= 0
	vx, vy = ReflectVelocity(100, 3, 4, -6, 5, 800, 600)
	if vx != 4 {
		t.Errorf("vx should be unchanged, got %v", vx)
	}
	if vy != 3 {
		t.Errorf("expected vy = 3 after top wall hit, got %v", vy)
	}
}

// --- Arena tests ---

func TestNewArena(t *testing.T) {
	a := NewArena(800, 600, 400, 300, 50)
	if a.Width != 800 || a.Height != 600 {
		t.Error("wrong arena dimensions")
	}
	if a.HomeBaseX != 400 || a.HomeBaseY != 300 || a.HomeBaseR != 50 {
		t.Error("wrong home base")
	}
}

func TestArenaAddObstacle(t *testing.T) {
	a := NewArena(800, 600, 400, 300, 50)
	a.AddObstacle(100, 100, 50, 50)
	if len(a.Obstacles) != 1 {
		t.Errorf("expected 1 obstacle, got %d", len(a.Obstacles))
	}
	if !a.Obstacles[0].Pushable {
		t.Error("obstacles should be pushable by default")
	}
}

func TestArenaInHomeBase(t *testing.T) {
	a := NewArena(800, 600, 400, 300, 50)
	if !a.InHomeBase(400, 300) {
		t.Error("center of home base should be inside")
	}
	if a.InHomeBase(0, 0) {
		t.Error("origin should be outside home base")
	}
	// Edge: exactly on boundary (distance == radius)
	if !a.InHomeBase(450, 300) {
		t.Error("point on boundary should be inside (<=)")
	}
}

func TestArenaGenerateObstacles(t *testing.T) {
	a := NewArena(800, 600, 400, 300, 50)
	rng := rand.New(rand.NewSource(42))
	a.GenerateObstacles(5, rng)
	if len(a.Obstacles) == 0 {
		t.Error("expected obstacles to be generated")
	}
	if len(a.Obstacles) > 5 {
		t.Errorf("expected at most 5 obstacles, got %d", len(a.Obstacles))
	}
}
