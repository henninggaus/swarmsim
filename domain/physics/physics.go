package physics

import (
	"math"
	"math/rand"
)

// NewArena creates an arena with the given dimensions and home base.
func NewArena(w, h, hbx, hby, hbr float64) *Arena {
	return &Arena{
		Width: w, Height: h,
		HomeBaseX: hbx, HomeBaseY: hby, HomeBaseR: hbr,
	}
}

// AddObstacle adds an obstacle to the arena.
func (a *Arena) AddObstacle(x, y, w, h float64) {
	a.Obstacles = append(a.Obstacles, &Obstacle{X: x, Y: y, W: w, H: h, Pushable: true})
}

// GenerateObstacles creates n random obstacles, avoiding the home base area.
func (a *Arena) GenerateObstacles(n int, rng *rand.Rand) {
	margin := 40.0
	for i := 0; i < n; i++ {
		for attempt := 0; attempt < 50; attempt++ {
			w := 30 + rng.Float64()*80
			h := 30 + rng.Float64()*80
			x := margin + rng.Float64()*(a.Width-2*margin-w)
			y := margin + rng.Float64()*(a.Height-2*margin-h)
			cx, cy := x+w/2, y+h/2
			dx := cx - a.HomeBaseX
			dy := cy - a.HomeBaseY
			if dx*dx+dy*dy > (a.HomeBaseR+60)*(a.HomeBaseR+60) {
				a.AddObstacle(x, y, w, h)
				break
			}
		}
	}
}

// InHomeBase returns true if position (px,py) is inside the home base circle.
func (a *Arena) InHomeBase(px, py float64) bool {
	dx := px - a.HomeBaseX
	dy := py - a.HomeBaseY
	return dx*dx+dy*dy <= a.HomeBaseR*a.HomeBaseR
}

// Clamp restricts a value to [min, max].
func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ClampToBounds keeps a position within arena bounds with a margin for the bot radius.
func ClampToBounds(x, y, radius, arenaW, arenaH float64) (float64, float64, bool) {
	nx := Clamp(x, radius, arenaW-radius)
	ny := Clamp(y, radius, arenaH-radius)
	clamped := nx != x || ny != y
	return nx, ny, clamped
}

// ReflectVelocity reverses velocity components when hitting a wall.
func ReflectVelocity(x, y, vx, vy, radius, arenaW, arenaH float64) (float64, float64) {
	if x-radius <= 0 || x+radius >= arenaW {
		vx = -vx * 0.5
	}
	if y-radius <= 0 || y+radius >= arenaH {
		vy = -vy * 0.5
	}
	return vx, vy
}

// CircleRectCollision tests if circle (cx,cy,cr) overlaps rectangle (rx,ry,rw,rh).
func CircleRectCollision(cx, cy, cr, rx, ry, rw, rh float64) (bool, float64, float64) {
	nearX := Clamp(cx, rx, rx+rw)
	nearY := Clamp(cy, ry, ry+rh)
	dx := cx - nearX
	dy := cy - nearY
	dist := math.Sqrt(dx*dx + dy*dy)
	return dist < cr, nearX, nearY
}

// ResolveCircleRectOverlap pushes the circle out of the rectangle.
func ResolveCircleRectOverlap(cx, cy, cr, rx, ry, rw, rh float64) (float64, float64) {
	nearX := Clamp(cx, rx, rx+rw)
	nearY := Clamp(cy, ry, ry+rh)
	dx := cx - nearX
	dy := cy - nearY
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist == 0 {
		toLeft := cx - rx
		toRight := rx + rw - cx
		toTop := cy - ry
		toBottom := ry + rh - cy
		minD := toLeft
		pushX, pushY := -1.0, 0.0
		if toRight < minD {
			minD = toRight
			pushX, pushY = 1, 0
		}
		if toTop < minD {
			minD = toTop
			pushX, pushY = 0, -1
		}
		if toBottom < minD {
			pushX, pushY = 0, 1
		}
		_ = minD
		return cx + pushX*(cr+1), cy + pushY*(cr+1)
	}
	if dist < cr {
		overlap := cr - dist
		nx := dx / dist
		ny := dy / dist
		return cx + nx*overlap, cy + ny*overlap
	}
	return cx, cy
}

// NewSpatialHash creates a spatial hash for the given arena size and cell size.
// Pre-allocates all cell slices to avoid per-frame heap allocations.
// Returns nil if any dimension or cellSize is invalid.
func NewSpatialHash(arenaW, arenaH, cellSize float64) *SpatialHash {
	if cellSize <= 0 || arenaW <= 0 || arenaH <= 0 {
		return nil
	}
	cols := int(math.Ceil(arenaW / cellSize))
	rows := int(math.Ceil(arenaH / cellSize))
	total := cols * rows
	cells := make([][]int, total)
	for i := range cells {
		cells[i] = make([]int, 0, 8)
	}
	return &SpatialHash{
		CellSize: cellSize,
		Cols:     cols,
		Rows:     rows,
		cells:    cells,
		queryBuf: make([]int, 0, 64),
	}
}

// Clear resets all cells without deallocating backing arrays.
func (s *SpatialHash) Clear() {
	for i := range s.cells {
		s.cells[i] = s.cells[i][:0]
	}
}

// Insert adds an entity at (x, y) with the given id.
func (s *SpatialHash) Insert(id int, x, y float64) {
	ci := s.cellIndex(x, y)
	s.cells[ci] = append(s.cells[ci], id)
}

// Query returns all entity IDs in cells overlapping the given radius.
// The returned slice is reused across calls — callers must consume it
// before the next Query call.
func (s *SpatialHash) Query(x, y, radius float64) []int {
	minCX := int((x - radius) / s.CellSize)
	maxCX := int((x + radius) / s.CellSize)
	minCY := int((y - radius) / s.CellSize)
	maxCY := int((y + radius) / s.CellSize)
	if minCX < 0 {
		minCX = 0
	}
	if minCY < 0 {
		minCY = 0
	}
	if maxCX >= s.Cols {
		maxCX = s.Cols - 1
	}
	if maxCY >= s.Rows {
		maxCY = s.Rows - 1
	}
	s.queryBuf = s.queryBuf[:0]
	for cy := minCY; cy <= maxCY; cy++ {
		for cx := minCX; cx <= maxCX; cx++ {
			ci := cy*s.Cols + cx
			s.queryBuf = append(s.queryBuf, s.cells[ci]...)
		}
	}
	return s.queryBuf
}

func (s *SpatialHash) cellIndex(x, y float64) int {
	cx := int(x / s.CellSize)
	cy := int(y / s.CellSize)
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	if cx >= s.Cols {
		cx = s.Cols - 1
	}
	if cy >= s.Rows {
		cy = s.Rows - 1
	}
	return cy*s.Cols + cx
}

// Distance returns the Euclidean distance between two points.
func Distance(x1, y1, x2, y2 float64) float64 {
	dx := x1 - x2
	dy := y1 - y2
	return math.Sqrt(dx*dx + dy*dy)
}

// Normalize returns a unit vector in the direction (dx, dy).
func Normalize(dx, dy float64) (float64, float64) {
	l := math.Sqrt(dx*dx + dy*dy)
	if l == 0 {
		return 0, 0
	}
	return dx / l, dy / l
}
