package swarm

import (
	"math"
	"swarmsim/domain/physics"
	"testing"
)

// newTestSwarmState creates a minimal SwarmState for A* testing.
func newTestSwarmState(numBots int) *SwarmState {
	ss := &SwarmState{
		ArenaW:     800,
		ArenaH:     800,
		DeliveryOn: true,
		Bots:       make([]SwarmBot, numBots),
		Obstacles:  make([]*physics.Obstacle, 0),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = 100
		ss.Bots[i].Y = 100
		ss.Bots[i].CarryingPkg = -1
	}
	return ss
}

func TestInitAStar(t *testing.T) {
	ss := newTestSwarmState(10)
	InitAStar(ss)

	if ss.AStar == nil {
		t.Fatal("AStar state should not be nil after init")
	}
	if !ss.AStarOn {
		t.Fatal("AStarOn should be true after init")
	}
	if ss.AStar.GridCols <= 0 || ss.AStar.GridRows <= 0 {
		t.Fatal("Grid dimensions should be positive")
	}
	if len(ss.AStar.Blocked) != ss.AStar.GridCols*ss.AStar.GridRows {
		t.Fatal("Blocked grid size mismatch")
	}
	if len(ss.AStar.Paths) != 10 {
		t.Fatalf("Expected 10 path slots, got %d", len(ss.AStar.Paths))
	}
}

func TestClearAStar(t *testing.T) {
	ss := newTestSwarmState(5)
	InitAStar(ss)
	ClearAStar(ss)

	if ss.AStar != nil {
		t.Fatal("AStar should be nil after clear")
	}
	if ss.AStarOn {
		t.Fatal("AStarOn should be false after clear")
	}
}

func TestTickAStarNil(t *testing.T) {
	ss := newTestSwarmState(5)
	// Should not panic with nil AStar
	TickAStar(ss)
}

func TestBuildGrid(t *testing.T) {
	ss := newTestSwarmState(1)
	// Add an obstacle at (200, 200) with size 40x40
	ss.Obstacles = append(ss.Obstacles, &physics.Obstacle{X: 200, Y: 200, W: 40, H: 40})

	InitAStar(ss)
	st := ss.AStar

	// Check that cells overlapping the obstacle area are blocked
	// Obstacle at 200-240 x 200-240, with padding of SwarmBotRadius (10)
	// So blocked range is roughly 190-250 x 190-250
	// Cell at col=10, row=10 (200/20=10) should be blocked
	midCol := int(220 / st.CellSize)
	midRow := int(220 / st.CellSize)
	idx := midRow*st.GridCols + midCol
	if !st.Blocked[idx] {
		t.Fatalf("Cell at col=%d, row=%d should be blocked (obstacle at 200,200,40,40)", midCol, midRow)
	}

	// Cell far away should be free
	freeCol := 0
	freeRow := 0
	freeIdx := freeRow*st.GridCols + freeCol
	if st.Blocked[freeIdx] {
		t.Fatal("Cell at (0,0) should be free")
	}
}

func TestAStarOpenPath(t *testing.T) {
	ss := newTestSwarmState(1)
	InitAStar(ss)
	st := ss.AStar

	path := astarSearch(st, 50, 50, 700, 700)
	if path == nil {
		t.Fatal("Expected a path on open grid, got nil")
	}
	if len(path) < 2 {
		t.Fatalf("Path should have at least 2 waypoints, got %d", len(path))
	}

	// Last waypoint should be near the goal
	last := path[len(path)-1]
	if math.Abs(last.X-700) > 1 || math.Abs(last.Y-700) > 1 {
		t.Fatalf("Last waypoint (%.1f, %.1f) should be near goal (700, 700)", last.X, last.Y)
	}
}

func TestAStarWithObstacle(t *testing.T) {
	ss := newTestSwarmState(1)
	// Create a wall blocking direct path from (50,400) to (750,400)
	// Wall at x=380, y=100, w=40, h=600 (blocks the middle vertically)
	ss.Obstacles = append(ss.Obstacles, &physics.Obstacle{X: 380, Y: 100, W: 40, H: 600})

	InitAStar(ss)
	st := ss.AStar

	path := astarSearch(st, 50, 400, 750, 400)
	if path == nil {
		t.Fatal("Expected a path around obstacle, got nil")
	}
	if len(path) < 3 {
		t.Fatalf("Path around obstacle should have more waypoints, got %d", len(path))
	}

	// Verify no waypoint is inside the obstacle (accounting for padding)
	for _, wp := range path {
		if wp.X > 370 && wp.X < 430 && wp.Y > 90 && wp.Y < 710 {
			t.Fatalf("Waypoint (%.1f, %.1f) is inside obstacle zone", wp.X, wp.Y)
		}
	}
}

func TestAStarNoPath(t *testing.T) {
	ss := newTestSwarmState(1)
	// Surround the goal with obstacles to make it unreachable
	// Create a box around (700, 700) that's fully enclosed
	ss.Obstacles = append(ss.Obstacles,
		&physics.Obstacle{X: 640, Y: 640, W: 120, H: 20},  // top
		&physics.Obstacle{X: 640, Y: 760, W: 120, H: 20},  // bottom
		&physics.Obstacle{X: 640, Y: 640, W: 20, H: 140},  // left
		&physics.Obstacle{X: 740, Y: 640, W: 20, H: 140},  // right
	)

	InitAStar(ss)
	st := ss.AStar

	path := astarSearch(st, 50, 50, 700, 700)
	// Should either find a path to nearest free cell or return nil
	// The exact behavior depends on findNearestFree
	_ = path // just verify it doesn't panic
}

func TestPathDistance(t *testing.T) {
	bot := &SwarmBot{X: 0, Y: 0}
	st := &AStarState{
		Paths: [][]PathNode{
			{{X: 100, Y: 0}, {X: 100, Y: 100}},
		},
		PathIdx: []int{0},
	}

	updatePathSensors(bot, st, 0)

	// Distance should be ~200 (100 + 100)
	if bot.PathDist < 190 || bot.PathDist > 210 {
		t.Fatalf("Expected PathDist ~200, got %d", bot.PathDist)
	}
}

func TestPathAngle(t *testing.T) {
	bot := &SwarmBot{X: 0, Y: 0, Angle: 0} // facing right (+X)
	st := &AStarState{
		Paths: [][]PathNode{
			{{X: 100, Y: 0}}, // waypoint directly ahead
		},
		PathIdx: []int{0},
	}

	updatePathSensors(bot, st, 0)

	// Angle should be ~0 (waypoint is directly ahead)
	if bot.PathAngle < -5 || bot.PathAngle > 5 {
		t.Fatalf("Expected PathAngle ~0 for waypoint ahead, got %d", bot.PathAngle)
	}

	// Now waypoint 90° to the left (up, since Atan2(negative_y, 0) = -pi/2)
	st.Paths[0] = []PathNode{{X: 0, Y: 100}} // below in screen coords
	updatePathSensors(bot, st, 0)
	if bot.PathAngle < 80 || bot.PathAngle > 100 {
		t.Fatalf("Expected PathAngle ~90 for waypoint below, got %d", bot.PathAngle)
	}
}

func TestStaggeredComputation(t *testing.T) {
	ss := newTestSwarmState(20)
	// Add stations so bots have targets
	ss.Stations = []DeliveryStation{
		{X: 700, Y: 700, Color: 1, IsPickup: true, HasPackage: true},
		{X: 100, Y: 700, Color: 1, IsPickup: false},
	}

	InitAStar(ss)
	st := ss.AStar
	st.BotsPerTick = 5

	// After one tick, only 5 bots should have been processed
	ss.Tick = 1
	TickAStar(ss)

	if st.NextBotBatch != 5 {
		t.Fatalf("Expected NextBotBatch=5 after first tick, got %d", st.NextBotBatch)
	}

	// After second tick, next 5 bots
	ss.Tick = 2
	TickAStar(ss)

	if st.NextBotBatch != 10 {
		t.Fatalf("Expected NextBotBatch=10 after second tick, got %d", st.NextBotBatch)
	}
}

func TestFollowPathBasic(t *testing.T) {
	ss := newTestSwarmState(1)
	InitAStar(ss)
	st := ss.AStar

	bot := &ss.Bots[0]
	bot.X = 100
	bot.Y = 100

	// Set up a simple path
	st.Paths[0] = []PathNode{{X: 200, Y: 100}, {X: 300, Y: 100}}
	st.PathIdx[0] = 0

	FollowPath(bot, ss, 0)

	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected speed %f, got %f", SwarmBotSpeed, bot.Speed)
	}

	// Bot should be facing toward (200, 100) → angle ~0
	if math.Abs(bot.Angle) > 0.1 {
		t.Fatalf("Expected angle ~0 (facing right), got %f", bot.Angle)
	}
}

func TestGridRebuildOnObstacleChange(t *testing.T) {
	ss := newTestSwarmState(1)
	InitAStar(ss)
	st := ss.AStar

	oldHash := st.ObstacleHash

	// Add obstacle
	ss.Obstacles = append(ss.Obstacles, &physics.Obstacle{X: 400, Y: 400, W: 50, H: 50})
	newHash := obstacleHash(ss)

	if newHash == oldHash {
		t.Fatal("Obstacle hash should change when obstacle is added")
	}
}

func TestFindNearestFree(t *testing.T) {
	st := &AStarState{
		GridCols: 10,
		GridRows: 10,
		CellSize: 20,
		Blocked:  make([]bool, 100),
	}

	// Block center cell
	st.Blocked[5*10+5] = true

	// Find nearest free to blocked cell
	col, row := findNearestFree(st, 5, 5)
	if col < 0 || row < 0 {
		t.Fatal("Should find a free cell near blocked cell")
	}
	if col == 5 && row == 5 {
		t.Fatal("Should not return the blocked cell itself")
	}
	// Should be adjacent
	dc := col - 5
	dr := row - 5
	if dc < -1 || dc > 1 || dr < -1 || dr > 1 {
		t.Fatalf("Nearest free cell (%d,%d) should be adjacent to (5,5)", col, row)
	}
}

func TestPathSmoothing(t *testing.T) {
	ss := newTestSwarmState(1)
	InitAStar(ss)
	st := ss.AStar

	// On an open grid, a path from (50,50) to (750,50) should be smoothed
	// to very few waypoints (ideally just the goal, since there's line-of-sight)
	path := astarSearch(st, 50, 50, 750, 50)
	if path == nil {
		t.Fatal("Expected a path, got nil")
	}

	// With smoothing, a straight-line path on open grid should be very short
	// (much shorter than the raw A* grid path would be)
	if len(path) > 5 {
		t.Fatalf("Smoothed straight-line path should be very short, got %d waypoints", len(path))
	}
}

func TestPathSmoothingWithObstacle(t *testing.T) {
	ss := newTestSwarmState(1)
	// Wall blocking direct path
	ss.Obstacles = append(ss.Obstacles, &physics.Obstacle{X: 380, Y: 0, W: 40, H: 300})
	InitAStar(ss)
	st := ss.AStar

	path := astarSearch(st, 50, 150, 750, 150)
	if path == nil {
		t.Fatal("Expected a path around obstacle, got nil")
	}
	// With obstacle, path needs more points than a straight line, but smoothing
	// should still reduce unnecessary zigzag waypoints
	if len(path) < 2 {
		t.Fatal("Path around obstacle should have at least 2 waypoints")
	}
}

func TestHasLineOfSight(t *testing.T) {
	st := &AStarState{
		GridCols: 10,
		GridRows: 10,
		CellSize: 20,
		Blocked:  make([]bool, 100),
	}

	// Open grid: line of sight should be clear
	if !hasLineOfSight(st, 10, 10, 190, 190) {
		t.Fatal("Expected clear line of sight on open grid")
	}

	// Block a cell in the middle
	st.Blocked[5*10+5] = true // cell at (5,5) = world (100-120, 100-120)

	// Line through blocked cell should be blocked
	if hasLineOfSight(st, 10, 10, 190, 190) {
		t.Fatal("Expected blocked line of sight through obstacle")
	}

	// Line that avoids the obstacle should be clear
	if !hasLineOfSight(st, 10, 10, 190, 10) {
		t.Fatal("Expected clear horizontal line of sight")
	}
}
