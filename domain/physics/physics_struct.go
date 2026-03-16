package physics

// Obstacle is a rectangular obstacle in the arena.
type Obstacle struct {
	X, Y, W, H float64
	Pushable   bool    // can be moved by Tank bots
	VX, VY     float64 // velocity for dynamic environment mode

	// Patrol movement (moves between two points)
	PatrolOn   bool
	PatrolX1   float64 // start point X
	PatrolY1   float64 // start point Y
	PatrolX2   float64 // end point X
	PatrolY2   float64 // end point Y
	PatrolT    float64 // 0.0-1.0 interpolation parameter
	PatrolDir  float64 // +1 or -1 (direction of movement)
	PatrolSpeed float64 // speed of patrol (0.0-1.0 per tick)

	// Rotation
	RotateOn    bool
	RotateAngle float64 // current angle in radians
	RotateSpeed float64 // radians per tick
	RotateCX    float64 // rotation center X (relative to obstacle center)
	RotateCY    float64 // rotation center Y
	RotateRadius float64 // distance from rotation center
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
