package bot

import (
	"math"
	"testing"
)

// === FormationCircle tests ===

func TestFormationCircleSlot0(t *testing.T) {
	// Slot 0, heading=0, centered at (100,100)
	x, y := FormationSlotPos(FormationCircle, 0, 100, 100, 0, 50)
	// slot 0 at angle heading+0 → (100+cos(0)*50, 100+sin(0)*50) = (150, 100)
	if math.Abs(x-150) > 0.1 || math.Abs(y-100) > 0.1 {
		t.Errorf("circle slot 0 at heading 0: expected (150,100), got (%.1f,%.1f)", x, y)
	}
}

func TestFormationCircleSlotsEquidistant(t *testing.T) {
	// All 8 slots should be at the same distance from center
	cx, cy := 200.0, 200.0
	spacing := 60.0
	for slot := 0; slot < 8; slot++ {
		x, y := FormationSlotPos(FormationCircle, slot, cx, cy, 0, spacing)
		dist := math.Sqrt((x-cx)*(x-cx) + (y-cy)*(y-cy))
		if math.Abs(dist-spacing) > 0.1 {
			t.Errorf("slot %d: distance to center = %.1f, expected %.1f", slot, dist, spacing)
		}
	}
}

func TestFormationCircleSlotsUnique(t *testing.T) {
	// All 8 slots should be at different positions
	type pos struct{ x, y float64 }
	positions := make([]pos, 8)
	for slot := 0; slot < 8; slot++ {
		x, y := FormationSlotPos(FormationCircle, slot, 200, 200, 0, 60)
		positions[slot] = pos{x, y}
	}

	for i := 0; i < 8; i++ {
		for j := i + 1; j < 8; j++ {
			dx := positions[i].x - positions[j].x
			dy := positions[i].y - positions[j].y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1.0 {
				t.Errorf("slots %d and %d overlap (dist=%.2f)", i, j, dist)
			}
		}
	}
}

func TestFormationCircleHeadingRotates(t *testing.T) {
	// Slot 0 at heading=0 vs heading=Pi/2 should be rotated 90°
	x1, y1 := FormationSlotPos(FormationCircle, 0, 0, 0, 0, 50)
	x2, y2 := FormationSlotPos(FormationCircle, 0, 0, 0, math.Pi/2, 50)

	// At heading=0: slot 0 → (50, 0)
	// At heading=Pi/2: slot 0 → (0, 50)
	if math.Abs(x1-50) > 0.1 || math.Abs(y1) > 0.1 {
		t.Errorf("heading=0 slot 0 expected (50,0), got (%.1f,%.1f)", x1, y1)
	}
	if math.Abs(x2) > 0.1 || math.Abs(y2-50) > 0.1 {
		t.Errorf("heading=Pi/2 slot 0 expected (0,50), got (%.1f,%.1f)", x2, y2)
	}
}

// === FormationLine tests ===

func TestFormationLineSymmetric(t *testing.T) {
	// Line formation should be symmetric around center (for symmetric slot indices)
	cx, cy := 200.0, 200.0
	spacing := 40.0
	heading := 0.0

	// Slot 3 is the center (offset = (3-3)*spacing*0.5 = 0)
	x3, y3 := FormationSlotPos(FormationLine, 3, cx, cy, heading, spacing)
	if math.Abs(x3-cx) > 0.1 || math.Abs(y3-cy) > 0.1 {
		t.Errorf("line slot 3 should be at center, got (%.1f,%.1f)", x3, y3)
	}
}

func TestFormationLinePerpendicularToHeading(t *testing.T) {
	// With heading=0, line should spread along Y axis (perpendicular)
	heading := 0.0
	x0, _ := FormationSlotPos(FormationLine, 0, 0, 0, heading, 40)
	x6, _ := FormationSlotPos(FormationLine, 6, 0, 0, heading, 40)

	// Slot 0: offset = (0-3)*40*0.5 = -60 → perpX = -sin(0) = 0, perpY = cos(0) = 1
	// So X should stay at 0 (spread is along Y)
	if math.Abs(x0) > 0.1 {
		t.Errorf("line slot 0 at heading=0 should have x≈0, got %.1f", x0)
	}
	if math.Abs(x6) > 0.1 {
		t.Errorf("line slot 6 at heading=0 should have x≈0, got %.1f", x6)
	}
}

// === FormationV tests ===

func TestFormationVLeaderAtCenter(t *testing.T) {
	// Slot 0: rank = 0/2+1 = 1, side = 1 (even)
	// The leader conceptually is behind the V-point, so
	// slot 0 should be offset from center
	x, y := FormationSlotPos(FormationV, 0, 200, 200, 0, 50)
	// Just verify it's a valid position (not NaN or Inf)
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		t.Errorf("V slot 0 should be valid position, got (%.1f,%.1f)", x, y)
	}
}

func TestFormationVSymmetricSides(t *testing.T) {
	// Slots 0 and 1 should be symmetric around the heading axis
	cx, cy := 200.0, 200.0
	heading := 0.0
	spacing := 50.0

	_, y0 := FormationSlotPos(FormationV, 0, cx, cy, heading, spacing)
	_, y1 := FormationSlotPos(FormationV, 1, cx, cy, heading, spacing)

	// They should be equidistant from center Y but on opposite sides
	dy0 := y0 - cy
	dy1 := y1 - cy
	if math.Abs(math.Abs(dy0)-math.Abs(dy1)) > 0.1 {
		t.Errorf("V sides should be symmetric: dy0=%.1f, dy1=%.1f", dy0, dy1)
	}
	// And on opposite sides
	if dy0*dy1 > 0 && math.Abs(dy0) > 0.1 {
		t.Errorf("V sides should be on opposite sides of heading: dy0=%.1f, dy1=%.1f", dy0, dy1)
	}
}

func TestFormationVRankIncreases(t *testing.T) {
	// Higher slots should be further back from center
	cx, cy := 200.0, 200.0
	heading := 0.0
	spacing := 50.0

	x0, _ := FormationSlotPos(FormationV, 0, cx, cy, heading, spacing)
	x2, _ := FormationSlotPos(FormationV, 2, cx, cy, heading, spacing)
	x4, _ := FormationSlotPos(FormationV, 4, cx, cy, heading, spacing)

	// dx component: -cos(heading) * rank → slot 0 has rank 1, slot 2 has rank 2
	// With heading=0, dx should be increasingly negative
	if x2 >= x0 {
		t.Errorf("slot 2 should be further back than slot 0: x0=%.1f, x2=%.1f", x0, x2)
	}
	if x4 >= x2 {
		t.Errorf("slot 4 should be further back than slot 2: x2=%.1f, x4=%.1f", x2, x4)
	}
}

// === Unknown formation type ===

func TestFormationUnknownReturnsCenter(t *testing.T) {
	x, y := FormationSlotPos(FormationType(99), 0, 200, 200, 0, 50)
	if x != 200 || y != 200 {
		t.Errorf("unknown formation should return center (200,200), got (%.1f,%.1f)", x, y)
	}
}

// === Spacing tests ===

func TestFormationCircleSpacingScales(t *testing.T) {
	x1, _ := FormationSlotPos(FormationCircle, 0, 0, 0, 0, 50)
	x2, _ := FormationSlotPos(FormationCircle, 0, 0, 0, 0, 100)

	// Double spacing = double distance
	if math.Abs(x2-x1*2) > 0.1 {
		t.Errorf("doubling spacing should double distance: x1=%.1f, x2=%.1f", x1, x2)
	}
}
