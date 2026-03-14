package physics

// Obstacle is a rectangular obstacle in the arena.
type Obstacle struct {
	X, Y, W, H float64
	Pushable   bool // can be moved by Tank bots
}

// Arena holds the world boundaries and obstacles.
type Arena struct {
	Width, Height float64
	Obstacles     []*Obstacle
	HomeBaseX     float64
	HomeBaseY     float64
	HomeBaseR     float64
}

// SpatialHash provides O(1) neighbor lookups for entities.
// Uses pre-allocated flat slices to avoid per-frame allocations.
type SpatialHash struct {
	CellSize float64
	Cols     int
	Rows     int
	cells    [][]int // pre-allocated [Cols*Rows], each with initial cap 8
	queryBuf []int   // reusable query result buffer
}
