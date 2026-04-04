package swarm

import (
	"container/heap"
	"math"
	"swarmsim/logger"
)

// AStarState holds the navigation grid and per-bot path data.
type AStarState struct {
	// Navigation grid
	GridCols int
	GridRows int
	CellSize float64
	Blocked  []bool // flat [row*GridCols+col], true if obstacle

	// Per-bot path data
	Paths       [][]PathNode // Paths[botIdx] = list of waypoints (world coords)
	PathIdx     []int        // PathIdx[botIdx] = index of next waypoint
	PathTargetX []float64    // cached target X per bot
	PathTargetY []float64    // cached target Y per bot
	PathTick    []int        // tick when path was last computed

	// Grid rebuild tracking
	ObstacleHash uint64

	// Staggered computation
	NextBotBatch      int
	BotsPerTick       int // how many bots to pathfind per tick (default 10)
	RecomputeInterval int // fallback recompute interval in ticks (default 120)
}

// PathNode is a waypoint in world coordinates.
type PathNode struct {
	X, Y float64
}

const (
	astarCellSize          = 20.0
	astarWaypointThreshold = 15.0 // distance to advance to next waypoint
	astarDefaultBatch      = 10
	astarDefaultRecompute  = 120
)

// InitAStar initializes the A* pathfinding subsystem.
func InitAStar(ss *SwarmState) {
	cols := int(math.Ceil(ss.ArenaW / astarCellSize))
	rows := int(math.Ceil(ss.ArenaH / astarCellSize))
	n := len(ss.Bots)

	st := &AStarState{
		GridCols:          cols,
		GridRows:          rows,
		CellSize:          astarCellSize,
		Blocked:           make([]bool, cols*rows),
		Paths:             make([][]PathNode, n),
		PathIdx:           make([]int, n),
		PathTargetX:       make([]float64, n),
		PathTargetY:       make([]float64, n),
		PathTick:          make([]int, n),
		BotsPerTick:       astarDefaultBatch,
		RecomputeInterval: astarDefaultRecompute,
	}

	buildGrid(st, ss)
	st.ObstacleHash = obstacleHash(ss)

	ss.AStar = st
	ss.AStarOn = true
	logger.Info("ASTAR", "A* Pathfinding initialisiert (%dx%d Grid)", cols, rows)
}

// ClearAStar disables and frees the A* subsystem.
func ClearAStar(ss *SwarmState) {
	ss.AStar = nil
	ss.AStarOn = false
}

// TickAStar runs one tick of the A* pathfinding system.
// Processes a batch of bots per tick (staggered) and updates sensor caches.
func TickAStar(ss *SwarmState) {
	st := ss.AStar
	if st == nil {
		return
	}

	// Ensure arrays match bot count (bots may have been added/removed)
	if len(st.Paths) != len(ss.Bots) {
		n := len(ss.Bots)
		st.Paths = make([][]PathNode, n)
		st.PathIdx = make([]int, n)
		st.PathTargetX = make([]float64, n)
		st.PathTargetY = make([]float64, n)
		st.PathTick = make([]int, n)
	}

	// Check if grid needs rebuild
	newHash := obstacleHash(ss)
	gridRebuilt := false
	if newHash != st.ObstacleHash {
		buildGrid(st, ss)
		st.ObstacleHash = newHash
		gridRebuilt = true
	}

	// Process a batch of bots
	n := len(ss.Bots)
	if n == 0 {
		return
	}
	batchSize := st.BotsPerTick
	if batchSize > n {
		batchSize = n
	}

	for b := 0; b < batchSize; b++ {
		idx := (st.NextBotBatch + b) % n
		bot := &ss.Bots[idx]

		// Determine target based on delivery state
		goalX, goalY, hasGoal := determineTarget(bot, ss)
		if !hasGoal {
			st.Paths[idx] = nil
			bot.PathDist = 0
			bot.PathAngle = 0
			continue
		}

		// Check if path needs recompute
		needsRecompute := st.Paths[idx] == nil ||
			st.PathIdx[idx] >= len(st.Paths[idx]) ||
			gridRebuilt ||
			(ss.Tick-st.PathTick[idx]) > st.RecomputeInterval ||
			math.Abs(goalX-st.PathTargetX[idx]) > astarCellSize ||
			math.Abs(goalY-st.PathTargetY[idx]) > astarCellSize

		if needsRecompute {
			path := astarSearch(st, bot.X, bot.Y, goalX, goalY)
			st.Paths[idx] = path
			st.PathIdx[idx] = 0
			st.PathTargetX[idx] = goalX
			st.PathTargetY[idx] = goalY
			st.PathTick[idx] = ss.Tick
		}
	}
	st.NextBotBatch = (st.NextBotBatch + batchSize) % n

	// Update sensor caches for ALL bots (cheap operation)
	for i := range ss.Bots {
		updatePathSensors(&ss.Bots[i], st, i)
	}
}

// FollowPath steers a bot toward its next A* waypoint.
func FollowPath(bot *SwarmBot, ss *SwarmState, botIdx int) {
	st := ss.AStar
	if st == nil || botIdx >= len(st.Paths) || st.Paths[botIdx] == nil {
		bot.Speed = SwarmBotSpeed
		return
	}

	path := st.Paths[botIdx]
	pidx := st.PathIdx[botIdx]
	if pidx >= len(path) {
		bot.Speed = SwarmBotSpeed
		return
	}

	wp := path[pidx]
	dx := wp.X - bot.X
	dy := wp.Y - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	// Advance waypoint if close enough
	if dist < astarWaypointThreshold && pidx < len(path)-1 {
		st.PathIdx[botIdx]++
		pidx++
		wp = path[pidx]
		dx = wp.X - bot.X
		dy = wp.Y - bot.Y
	}

	// Steer toward waypoint
	bot.Angle = math.Atan2(dy, dx)
	bot.Speed = SwarmBotSpeed
}

// determineTarget picks the best goal for a bot based on delivery state.
func determineTarget(bot *SwarmBot, ss *SwarmState) (float64, float64, bool) {
	if !ss.DeliveryOn || len(ss.Stations) == 0 {
		return 0, 0, false
	}

	if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
		// Carrying: find nearest matching dropoff
		carryColor := ss.Packages[bot.CarryingPkg].Color
		bestDist := math.MaxFloat64
		bestX, bestY := 0.0, 0.0
		found := false
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup || st.Color != carryColor {
				continue
			}
			dx := st.X - bot.X
			dy := st.Y - bot.Y
			d := dx*dx + dy*dy
			if d < bestDist {
				bestDist = d
				bestX = st.X
				bestY = st.Y
				found = true
			}
		}
		return bestX, bestY, found
	}

	// Not carrying: find nearest pickup with package
	bestDist := math.MaxFloat64
	bestX, bestY := 0.0, 0.0
	found := false
	for si := range ss.Stations {
		st := &ss.Stations[si]
		if !st.IsPickup || !st.HasPackage {
			continue
		}
		dx := st.X - bot.X
		dy := st.Y - bot.Y
		d := dx*dx + dy*dy
		if d < bestDist {
			bestDist = d
			bestX = st.X
			bestY = st.Y
			found = true
		}
	}
	return bestX, bestY, found
}

// updatePathSensors fills the bot's PathDist and PathAngle cache fields.
func updatePathSensors(bot *SwarmBot, st *AStarState, botIdx int) {
	if botIdx >= len(st.Paths) || st.Paths[botIdx] == nil || st.PathIdx[botIdx] >= len(st.Paths[botIdx]) {
		bot.PathDist = 0
		bot.PathAngle = 0
		return
	}

	path := st.Paths[botIdx]
	pidx := st.PathIdx[botIdx]

	// Compute remaining distance along path
	totalDist := 0.0
	prevX, prevY := bot.X, bot.Y
	for i := pidx; i < len(path); i++ {
		dx := path[i].X - prevX
		dy := path[i].Y - prevY
		totalDist += math.Sqrt(dx*dx + dy*dy)
		prevX = path[i].X
		prevY = path[i].Y
	}
	bot.PathDist = int(totalDist)

	// Compute angle to next waypoint relative to bot heading
	wp := path[pidx]
	dx := wp.X - bot.X
	dy := wp.Y - bot.Y
	absAngle := math.Atan2(dy, dx)
	relAngle := absAngle - bot.Angle
	// Normalize to [-PI, PI]
	relAngle = WrapAngle(relAngle)
	bot.PathAngle = int(relAngle * 180 / math.Pi)
}

// --- Grid building ---

// buildGrid marks cells as blocked based on obstacles.
func buildGrid(st *AStarState, ss *SwarmState) {
	for i := range st.Blocked {
		st.Blocked[i] = false
	}

	pad := SwarmBotRadius // padding to account for bot size
	allObs := ss.AllObstacles()

	for _, obs := range allObs {
		// Expand obstacle bounds by bot radius
		ox := obs.X - pad
		oy := obs.Y - pad
		ow := obs.W + pad*2
		oh := obs.H + pad*2

		// Find overlapping grid cells
		minCol := int(ox / st.CellSize)
		maxCol := int((ox + ow) / st.CellSize)
		minRow := int(oy / st.CellSize)
		maxRow := int((oy + oh) / st.CellSize)

		if minCol < 0 {
			minCol = 0
		}
		if minRow < 0 {
			minRow = 0
		}
		if maxCol >= st.GridCols {
			maxCol = st.GridCols - 1
		}
		if maxRow >= st.GridRows {
			maxRow = st.GridRows - 1
		}

		for r := minRow; r <= maxRow; r++ {
			for c := minCol; c <= maxCol; c++ {
				st.Blocked[r*st.GridCols+c] = true
			}
		}
	}
}

// obstacleHash computes a fast hash of obstacle positions for change detection.
func obstacleHash(ss *SwarmState) uint64 {
	var h uint64
	allObs := ss.AllObstacles()
	for _, obs := range allObs {
		h ^= math.Float64bits(obs.X) * 31
		h ^= math.Float64bits(obs.Y) * 37
		h ^= math.Float64bits(obs.W) * 41
		h ^= math.Float64bits(obs.H) * 43
		h = (h << 7) | (h >> 57) // rotate
	}
	return h
}

// --- A* search algorithm ---

type astarNode struct {
	idx    int     // cell index (row*cols+col)
	gCost  float64 // cost from start
	fCost  float64 // gCost + heuristic
	parent int     // parent cell index (-1 for start)
}

// astarHeap implements heap.Interface for A* open set.
type astarHeap []astarNode

func (h astarHeap) Len() int            { return len(h) }
func (h astarHeap) Less(i, j int) bool   { return h[i].fCost < h[j].fCost }
func (h astarHeap) Swap(i, j int)        { h[i], h[j] = h[j], h[i] }
func (h *astarHeap) Push(x interface{})  { *h = append(*h, x.(astarNode)) }
func (h *astarHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// 8-directional neighbors: dx, dy, cost
var astarDirs = [8][3]float64{
	{1, 0, 1.0}, {-1, 0, 1.0}, {0, 1, 1.0}, {0, -1, 1.0},
	{1, 1, 1.414}, {1, -1, 1.414}, {-1, 1, 1.414}, {-1, -1, 1.414},
}

// astarSearch finds a path from (sx,sy) to (gx,gy) in world coordinates.
// Returns nil if no path found.
func astarSearch(st *AStarState, sx, sy, gx, gy float64) []PathNode {
	// Convert to grid coords
	startCol := int(sx / st.CellSize)
	startRow := int(sy / st.CellSize)
	goalCol := int(gx / st.CellSize)
	goalRow := int(gy / st.CellSize)

	// Clamp to grid bounds
	startCol = clampInt(startCol, 0, st.GridCols-1)
	startRow = clampInt(startRow, 0, st.GridRows-1)
	goalCol = clampInt(goalCol, 0, st.GridCols-1)
	goalRow = clampInt(goalRow, 0, st.GridRows-1)

	startIdx := startRow*st.GridCols + startCol
	goalIdx := goalRow*st.GridCols + goalCol

	// If goal is blocked, find nearest unblocked cell
	if st.Blocked[goalIdx] {
		goalCol, goalRow = findNearestFree(st, goalCol, goalRow)
		if goalCol < 0 {
			return nil
		}
		goalIdx = goalRow*st.GridCols + goalCol
	}

	// If start is blocked, find nearest unblocked cell
	if st.Blocked[startIdx] {
		startCol, startRow = findNearestFree(st, startCol, startRow)
		if startCol < 0 {
			return nil
		}
		startIdx = startRow*st.GridCols + startCol
	}

	if startIdx == goalIdx {
		return []PathNode{{
			X: float64(goalCol)*st.CellSize + st.CellSize/2,
			Y: float64(goalRow)*st.CellSize + st.CellSize/2,
		}}
	}

	totalCells := st.GridCols * st.GridRows
	gCosts := make([]float64, totalCells)
	parents := make([]int, totalCells)
	closed := make([]bool, totalCells)
	for i := range gCosts {
		gCosts[i] = math.MaxFloat64
		parents[i] = -1
	}

	gCosts[startIdx] = 0
	h := &astarHeap{}
	heap.Init(h)
	heap.Push(h, astarNode{
		idx:   startIdx,
		gCost: 0,
		fCost: euclidean(startCol, startRow, goalCol, goalRow),
	})

	for h.Len() > 0 {
		current := heap.Pop(h).(astarNode)
		ci := current.idx

		if ci == goalIdx {
			return reconstructPath(st, parents, startIdx, goalIdx, gx, gy)
		}

		if closed[ci] {
			continue
		}
		closed[ci] = true

		curRow := ci / st.GridCols
		curCol := ci % st.GridCols

		for _, dir := range astarDirs {
			nc := curCol + int(dir[0])
			nr := curRow + int(dir[1])

			if nc < 0 || nc >= st.GridCols || nr < 0 || nr >= st.GridRows {
				continue
			}

			ni := nr*st.GridCols + nc
			if closed[ni] || st.Blocked[ni] {
				continue
			}

			// For diagonal movement, check that both cardinal neighbors are free
			if dir[0] != 0 && dir[1] != 0 {
				adj1 := curRow*st.GridCols + nc
				adj2 := nr*st.GridCols + curCol
				if st.Blocked[adj1] || st.Blocked[adj2] {
					continue
				}
			}

			newG := gCosts[ci] + dir[2]*st.CellSize
			if newG < gCosts[ni] {
				gCosts[ni] = newG
				parents[ni] = ci
				fCost := newG + euclidean(nc, nr, goalCol, goalRow)*st.CellSize
				heap.Push(h, astarNode{idx: ni, gCost: newG, fCost: fCost})
			}
		}
	}

	return nil // no path found
}

// reconstructPath builds a PathNode slice from A* parent chain.
func reconstructPath(st *AStarState, parents []int, startIdx, goalIdx int, goalX, goalY float64) []PathNode {
	// Trace back from goal to start
	var indices []int
	for ci := goalIdx; ci != startIdx && ci >= 0; ci = parents[ci] {
		indices = append(indices, ci)
		if parents[ci] == ci {
			break // safety: prevent infinite loop
		}
	}

	// Reverse to get start→goal order
	path := make([]PathNode, 0, len(indices))
	for i := len(indices) - 1; i >= 0; i-- {
		ci := indices[i]
		row := ci / st.GridCols
		col := ci % st.GridCols
		path = append(path, PathNode{
			X: float64(col)*st.CellSize + st.CellSize/2,
			Y: float64(row)*st.CellSize + st.CellSize/2,
		})
	}

	// Replace last waypoint with exact goal position
	if len(path) > 0 {
		path[len(path)-1] = PathNode{X: goalX, Y: goalY}
	}

	// Apply line-of-sight smoothing to remove unnecessary zigzag waypoints
	path = smoothPath(st, path)

	return path
}

// smoothPath removes redundant waypoints using line-of-sight checks.
// If waypoint i can "see" waypoint i+2 (no blocked cells between), skip i+1.
func smoothPath(st *AStarState, path []PathNode) []PathNode {
	if len(path) <= 2 {
		return path
	}

	smoothed := []PathNode{path[0]}
	i := 0
	for i < len(path)-1 {
		// Try to skip ahead as far as possible
		best := i + 1
		for j := i + 2; j < len(path); j++ {
			if hasLineOfSight(st, path[i].X, path[i].Y, path[j].X, path[j].Y) {
				best = j
			} else {
				break
			}
		}
		smoothed = append(smoothed, path[best])
		i = best
	}
	return smoothed
}

// hasLineOfSight checks if a straight line between two world points crosses any blocked cell.
// Uses Bresenham-like grid traversal.
func hasLineOfSight(st *AStarState, x1, y1, x2, y2 float64) bool {
	c1 := int(x1 / st.CellSize)
	r1 := int(y1 / st.CellSize)
	c2 := int(x2 / st.CellSize)
	r2 := int(y2 / st.CellSize)

	dc := c2 - c1
	dr := r2 - r1
	stepC := 1
	stepR := 1
	if dc < 0 {
		stepC = -1
		dc = -dc
	}
	if dr < 0 {
		stepR = -1
		dr = -dr
	}

	c, r := c1, r1
	err := dc - dr

	for {
		if c >= 0 && c < st.GridCols && r >= 0 && r < st.GridRows {
			if st.Blocked[r*st.GridCols+c] {
				return false
			}
		}
		if c == c2 && r == r2 {
			break
		}
		e2 := 2 * err
		if e2 > -dr {
			err -= dr
			c += stepC
		}
		if e2 < dc {
			err += dc
			r += stepR
		}
	}
	return true
}

// findNearestFree finds the nearest unblocked cell to (col, row) via expanding square search.
func findNearestFree(st *AStarState, col, row int) (int, int) {
	for radius := 1; radius < st.GridCols+st.GridRows; radius++ {
		for dr := -radius; dr <= radius; dr++ {
			for dc := -radius; dc <= radius; dc++ {
				if dr != -radius && dr != radius && dc != -radius && dc != radius {
					continue // only check perimeter
				}
				nc := col + dc
				nr := row + dr
				if nc >= 0 && nc < st.GridCols && nr >= 0 && nr < st.GridRows {
					if !st.Blocked[nr*st.GridCols+nc] {
						return nc, nr
					}
				}
			}
		}
	}
	return -1, -1
}

func euclidean(c1, r1, c2, r2 int) float64 {
	dc := float64(c1 - c2)
	dr := float64(r1 - r2)
	return math.Sqrt(dc*dc + dr*dr)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
