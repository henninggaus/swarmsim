package swarm

import (
	"math"
	"swarmsim/logger"
)

// ShapeType identifies a target formation shape.
type ShapeType int

const (
	ShapeCircle   ShapeType = iota
	ShapeSquare
	ShapeTriangle
	ShapeArrow
	ShapeLine
	ShapeStar
	ShapeSpiral
	ShapeLetterV
	ShapeCount
)

// ShapeFormationState manages the shape formation system.
type ShapeFormationState struct {
	ActiveShape     ShapeType
	TargetPositions [][2]float64 // per-bot target positions
	CenterX         float64
	CenterY         float64
	Radius          float64   // formation radius (default 200)
	Convergence     float64   // 0.0-1.0 how well bots match their targets
	Speed           float64   // approach speed multiplier (default 0.5)
	RotationAngle   float64   // current rotation offset (animated)
	RotationSpeed   float64   // radians per tick (0 = static, default 0.005)
	Assigned        []int     // per-bot: which target slot (-1 = unassigned)
}

// ShapeTypeName returns the display name of a shape.
func ShapeTypeName(s ShapeType) string {
	switch s {
	case ShapeCircle:
		return "Kreis"
	case ShapeSquare:
		return "Quadrat"
	case ShapeTriangle:
		return "Dreieck"
	case ShapeArrow:
		return "Pfeil"
	case ShapeLine:
		return "Linie"
	case ShapeStar:
		return "Stern"
	case ShapeSpiral:
		return "Spirale"
	case ShapeLetterV:
		return "V-Form"
	default:
		return "?"
	}
}

// AllShapeNames returns all shape display names.
func AllShapeNames() []string {
	names := make([]string, ShapeCount)
	for i := ShapeType(0); i < ShapeCount; i++ {
		names[i] = ShapeTypeName(i)
	}
	return names
}

// InitShapeFormation sets up the shape formation system.
func InitShapeFormation(ss *SwarmState, shape ShapeType) {
	sf := &ShapeFormationState{
		ActiveShape:   shape,
		CenterX:       ss.ArenaW / 2,
		CenterY:       ss.ArenaH / 2,
		Radius:        200,
		Speed:         0.5,
		RotationSpeed: 0.005,
	}

	n := len(ss.Bots)
	sf.TargetPositions = GenerateShapePositions(shape, n, sf.CenterX, sf.CenterY, sf.Radius)
	sf.Assigned = assignTargets(ss, sf.TargetPositions)

	ss.ShapeFormation = sf
	logger.Info("SHAPE", "Formation: %s, %d Bots, Radius=%.0f",
		ShapeTypeName(shape), n, sf.Radius)
}

// ClearShapeFormation disables the shape formation system.
func ClearShapeFormation(ss *SwarmState) {
	ss.ShapeFormation = nil
	ss.ShapeFormationOn = false
}

// GenerateShapePositions creates target positions for a given shape.
func GenerateShapePositions(shape ShapeType, count int, cx, cy, radius float64) [][2]float64 {
	positions := make([][2]float64, count)

	switch shape {
	case ShapeCircle:
		for i := 0; i < count; i++ {
			angle := float64(i) * 2 * math.Pi / float64(count)
			positions[i] = [2]float64{
				cx + radius*math.Cos(angle),
				cy + radius*math.Sin(angle),
			}
		}

	case ShapeSquare:
		perSide := count / 4
		if perSide < 1 {
			perSide = 1
		}
		idx := 0
		halfR := radius
		for side := 0; side < 4 && idx < count; side++ {
			for j := 0; j < perSide && idx < count; j++ {
				t := float64(j) / float64(perSide)
				switch side {
				case 0: // top
					positions[idx] = [2]float64{cx - halfR + t*2*halfR, cy - halfR}
				case 1: // right
					positions[idx] = [2]float64{cx + halfR, cy - halfR + t*2*halfR}
				case 2: // bottom
					positions[idx] = [2]float64{cx + halfR - t*2*halfR, cy + halfR}
				case 3: // left
					positions[idx] = [2]float64{cx - halfR, cy + halfR - t*2*halfR}
				}
				idx++
			}
		}
		// Remaining bots fill remaining sides
		for idx < count {
			angle := float64(idx) * 2 * math.Pi / float64(count)
			positions[idx] = [2]float64{cx + halfR*math.Cos(angle), cy + halfR*math.Sin(angle)}
			idx++
		}

	case ShapeTriangle:
		perSide := count / 3
		if perSide < 1 {
			perSide = 1
		}
		// Three vertices
		verts := [3][2]float64{
			{cx, cy - radius},                                           // top
			{cx - radius*math.Sin(math.Pi/3), cy + radius*0.5},         // bottom left
			{cx + radius*math.Sin(math.Pi/3), cy + radius*0.5},         // bottom right
		}
		idx := 0
		for side := 0; side < 3 && idx < count; side++ {
			v1 := verts[side]
			v2 := verts[(side+1)%3]
			for j := 0; j < perSide && idx < count; j++ {
				t := float64(j) / float64(perSide)
				positions[idx] = [2]float64{
					v1[0] + t*(v2[0]-v1[0]),
					v1[1] + t*(v2[1]-v1[1]),
				}
				idx++
			}
		}
		for idx < count {
			positions[idx] = [2]float64{cx, cy}
			idx++
		}

	case ShapeArrow:
		// Arrow pointing right: shaft + head
		shaftLen := radius * 1.5
		shaftBots := count * 60 / 100
		headBots := count - shaftBots
		idx := 0
		// Shaft (horizontal line)
		for i := 0; i < shaftBots && idx < count; i++ {
			t := float64(i) / float64(shaftBots)
			positions[idx] = [2]float64{cx - shaftLen/2 + t*shaftLen*0.7, cy}
			idx++
		}
		// Arrowhead (V shape)
		for i := 0; i < headBots && idx < count; i++ {
			t := float64(i)/float64(headBots)*2 - 1 // -1 to 1
			tipX := cx + shaftLen/2
			positions[idx] = [2]float64{
				tipX - math.Abs(t)*radius*0.5,
				cy + t*radius*0.4,
			}
			idx++
		}

	case ShapeLine:
		for i := 0; i < count; i++ {
			t := float64(i)/float64(count-1)*2 - 1 // -1 to 1
			positions[i] = [2]float64{cx + t*radius, cy}
		}

	case ShapeStar:
		points := 5
		for i := 0; i < count; i++ {
			angle := float64(i) * 2 * math.Pi / float64(count)
			// Alternate between inner and outer radius
			pointIdx := int(angle / (2 * math.Pi / float64(points*2)))
			r := radius
			if pointIdx%2 == 1 {
				r = radius * 0.4
			}
			positions[i] = [2]float64{
				cx + r*math.Cos(angle-math.Pi/2),
				cy + r*math.Sin(angle-math.Pi/2),
			}
		}

	case ShapeSpiral:
		for i := 0; i < count; i++ {
			t := float64(i) / float64(count)
			angle := t * 4 * math.Pi // 2 full turns
			r := radius * 0.2 + t*radius*0.8
			positions[i] = [2]float64{
				cx + r*math.Cos(angle),
				cy + r*math.Sin(angle),
			}
		}

	case ShapeLetterV:
		half := count / 2
		for i := 0; i < count; i++ {
			var t float64
			if i < half {
				t = float64(i) / float64(half)
				positions[i] = [2]float64{
					cx - radius + t*radius,
					cy - radius + t*radius,
				}
			} else {
				t = float64(i-half) / float64(count-half)
				positions[i] = [2]float64{
					cx + t*radius,
					cy + radius - t*radius,
				}
			}
		}
	}

	return positions
}

// assignTargets uses greedy nearest-neighbor to assign bots to target positions.
func assignTargets(ss *SwarmState, targets [][2]float64) []int {
	n := len(ss.Bots)
	assigned := make([]int, n)
	taken := make([]bool, len(targets))

	for i := range assigned {
		assigned[i] = -1
	}

	// Greedy: for each bot, find nearest untaken target
	for i := range ss.Bots {
		bestDist := math.MaxFloat64
		bestJ := -1
		for j, t := range targets {
			if taken[j] {
				continue
			}
			dx := ss.Bots[i].X - t[0]
			dy := ss.Bots[i].Y - t[1]
			d := dx*dx + dy*dy
			if d < bestDist {
				bestDist = d
				bestJ = j
			}
		}
		if bestJ >= 0 {
			assigned[i] = bestJ
			taken[bestJ] = true
		}
	}
	return assigned
}

// TickShapeFormation moves bots toward their target positions.
func TickShapeFormation(ss *SwarmState) {
	sf := ss.ShapeFormation
	if sf == nil {
		return
	}

	// Animate rotation
	sf.RotationAngle += sf.RotationSpeed

	// Compute convergence
	totalDist := 0.0
	n := len(ss.Bots)

	for i := range ss.Bots {
		if i >= len(sf.Assigned) || sf.Assigned[i] < 0 {
			continue
		}
		target := sf.TargetPositions[sf.Assigned[i]]

		// Apply rotation around center
		dx := target[0] - sf.CenterX
		dy := target[1] - sf.CenterY
		cos := math.Cos(sf.RotationAngle)
		sin := math.Sin(sf.RotationAngle)
		rotX := sf.CenterX + dx*cos - dy*sin
		rotY := sf.CenterY + dx*sin + dy*cos

		// Move toward target
		ddx := rotX - ss.Bots[i].X
		ddy := rotY - ss.Bots[i].Y
		dist := math.Sqrt(ddx*ddx + ddy*ddy)
		totalDist += dist

		if dist > 2 {
			ss.Bots[i].Angle = math.Atan2(ddy, ddx)
			ss.Bots[i].Speed = math.Min(dist*sf.Speed*0.1, SwarmBotSpeed)
		} else {
			ss.Bots[i].Speed = 0
		}

		// LED: color by distance to target (green=close, red=far)
		t := dist / sf.Radius
		if t > 1 {
			t = 1
		}
		ss.Bots[i].LEDColor = [3]uint8{uint8(t * 255), uint8((1 - t) * 255), 50}
	}

	if n > 0 {
		sf.Convergence = 1.0 - math.Min(totalDist/float64(n)/sf.Radius, 1.0)
	}
}

// SetShape changes the active formation shape.
func SetShape(ss *SwarmState, shape ShapeType) {
	sf := ss.ShapeFormation
	if sf == nil {
		return
	}
	sf.ActiveShape = shape
	sf.TargetPositions = GenerateShapePositions(shape, len(ss.Bots), sf.CenterX, sf.CenterY, sf.Radius)
	sf.Assigned = assignTargets(ss, sf.TargetPositions)
	sf.RotationAngle = 0
}

// SetShapeRadius changes the formation size.
func SetShapeRadius(ss *SwarmState, radius float64) {
	sf := ss.ShapeFormation
	if sf == nil {
		return
	}
	if radius < 50 {
		radius = 50
	}
	if radius > 350 {
		radius = 350
	}
	sf.Radius = radius
	sf.TargetPositions = GenerateShapePositions(sf.ActiveShape, len(ss.Bots), sf.CenterX, sf.CenterY, sf.Radius)
	sf.Assigned = assignTargets(ss, sf.TargetPositions)
}

// ShapeConvergence returns 0.0-1.0 indicating formation quality.
func ShapeConvergence(sf *ShapeFormationState) float64 {
	if sf == nil {
		return 0
	}
	return sf.Convergence
}
