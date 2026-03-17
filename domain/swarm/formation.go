package swarm

import "math"

// FormationMetrics holds computed swarm formation stats.
type FormationMetrics struct {
	CentroidX    float64
	CentroidY    float64
	SpreadRadius float64 // avg distance from centroid
	MaxSpread    float64 // max distance from centroid
	ClusterCount int     // number of distinct clusters (DBSCAN-like)
	AvgSpeed     float64
	AvgNeighDist float64 // avg distance to nearest neighbor
	Alignment    float64 // 0-1, how aligned headings are
	Cohesion     float64 // 0-1, inverse of normalized spread
}

// ComputeFormation calculates swarm formation metrics.
func ComputeFormation(ss *SwarmState) FormationMetrics {
	n := len(ss.Bots)
	if n == 0 {
		return FormationMetrics{}
	}

	var m FormationMetrics

	// Centroid
	for i := range ss.Bots {
		m.CentroidX += ss.Bots[i].X
		m.CentroidY += ss.Bots[i].Y
	}
	m.CentroidX /= float64(n)
	m.CentroidY /= float64(n)

	// Spread, speed, alignment
	var sumSin, sumCos float64
	for i := range ss.Bots {
		dx := ss.Bots[i].X - m.CentroidX
		dy := ss.Bots[i].Y - m.CentroidY
		dist := math.Sqrt(dx*dx + dy*dy)
		m.SpreadRadius += dist
		if dist > m.MaxSpread {
			m.MaxSpread = dist
		}
		m.AvgSpeed += ss.Bots[i].Speed
		m.AvgNeighDist += ss.Bots[i].NearestDist
		sumSin += math.Sin(ss.Bots[i].Angle)
		sumCos += math.Cos(ss.Bots[i].Angle)
	}
	m.SpreadRadius /= float64(n)
	m.AvgSpeed /= float64(n)
	m.AvgNeighDist /= float64(n)

	// Alignment: length of mean heading vector (0=random, 1=all same direction)
	m.Alignment = math.Sqrt(sumSin*sumSin+sumCos*sumCos) / float64(n)

	// Cohesion: inverse normalized spread (1=tight, 0=spread across arena)
	maxPossibleSpread := float64(ss.ArenaW) * 0.5
	m.Cohesion = 1.0 - math.Min(m.SpreadRadius/maxPossibleSpread, 1.0)

	// Simple cluster detection (grid-based)
	m.ClusterCount = countClusters(ss)

	return m
}

// countClusters uses a grid-based approach to count bot clusters.
func countClusters(ss *SwarmState) int {
	cellSize := 60.0
	cols := int(float64(ss.ArenaW)/cellSize) + 1
	rows := int(float64(ss.ArenaH)/cellSize) + 1
	grid := make([]bool, cols*rows)

	for i := range ss.Bots {
		c := int(ss.Bots[i].X / cellSize)
		r := int(ss.Bots[i].Y / cellSize)
		if c >= 0 && c < cols && r >= 0 && r < rows {
			grid[r*cols+c] = true
		}
	}

	// Flood-fill to count connected occupied cells
	visited := make([]bool, cols*rows)
	clusters := 0
	for idx := range grid {
		if grid[idx] && !visited[idx] {
			clusters++
			// BFS flood fill
			queue := []int{idx}
			visited[idx] = true
			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				cr := cur / cols
				cc := cur % cols
				for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nr, nc := cr+d[0], cc+d[1]
					if nr >= 0 && nr < rows && nc >= 0 && nc < cols {
						ni := nr*cols + nc
						if grid[ni] && !visited[ni] {
							visited[ni] = true
							queue = append(queue, ni)
						}
					}
				}
			}
		}
	}
	return clusters
}

// --- Formation Morphing ---

// FormationType identifies a target formation shape.
type FormationType int

const (
	FormCircle    FormationType = iota // bots arranged in a circle
	FormLine                          // bots in a horizontal line
	FormGrid                          // bots in a grid
	FormV                             // V-formation (like geese)
	FormSpiral                        // Archimedean spiral
	FormTypeCount                     // sentinel
)

// FormationTarget holds per-bot target positions for morphing.
type FormationTarget struct {
	Targets [][2]float64 // (x,y) per bot
	Type    FormationType
}

// ComputeFormationTargets computes target positions for n bots in the given formation.
func ComputeFormationTargets(fType FormationType, n int, centerX, centerY, radius float64) FormationTarget {
	if n == 0 {
		return FormationTarget{Type: fType}
	}
	targets := make([][2]float64, n)

	switch fType {
	case FormCircle:
		for i := 0; i < n; i++ {
			angle := 2 * math.Pi * float64(i) / float64(n)
			targets[i] = [2]float64{
				centerX + radius*math.Cos(angle),
				centerY + radius*math.Sin(angle),
			}
		}

	case FormLine:
		spacing := 2 * radius / math.Max(float64(n-1), 1)
		startX := centerX - radius
		for i := 0; i < n; i++ {
			targets[i] = [2]float64{startX + float64(i)*spacing, centerY}
		}

	case FormGrid:
		cols := int(math.Ceil(math.Sqrt(float64(n))))
		spacing := 2 * radius / math.Max(float64(cols-1), 1)
		startX := centerX - radius
		startY := centerY - radius
		for i := 0; i < n; i++ {
			c := i % cols
			r := i / cols
			targets[i] = [2]float64{startX + float64(c)*spacing, startY + float64(r)*spacing}
		}

	case FormV:
		halfN := n / 2
		spacing := radius / math.Max(float64(halfN), 1)
		// Left wing
		for i := 0; i <= halfN; i++ {
			targets[i] = [2]float64{
				centerX - float64(i)*spacing,
				centerY + float64(i)*spacing*0.6,
			}
		}
		// Right wing
		for i := halfN + 1; i < n; i++ {
			j := i - halfN
			targets[i] = [2]float64{
				centerX + float64(j)*spacing,
				centerY + float64(j)*spacing*0.6,
			}
		}

	case FormSpiral:
		for i := 0; i < n; i++ {
			t := float64(i) / float64(n) * 4 * math.Pi // 2 full turns
			r := radius * float64(i) / float64(n)
			targets[i] = [2]float64{
				centerX + r*math.Cos(t),
				centerY + r*math.Sin(t),
			}
		}
	}

	return FormationTarget{Targets: targets, Type: fType}
}

// MorphToFormation smoothly moves bots toward their formation target positions.
// lerpFactor controls speed (0.01=slow, 0.1=fast).
// Returns average remaining distance to targets.
func MorphToFormation(ss *SwarmState, ft FormationTarget, lerpFactor float64) float64 {
	n := len(ss.Bots)
	if len(ft.Targets) == 0 || n == 0 {
		return 0
	}

	totalDist := 0.0
	count := 0
	for i := 0; i < n && i < len(ft.Targets); i++ {
		tx, ty := ft.Targets[i][0], ft.Targets[i][1]
		dx := tx - ss.Bots[i].X
		dy := ty - ss.Bots[i].Y
		dist := math.Sqrt(dx*dx + dy*dy)
		totalDist += dist
		count++

		if dist > 1.0 {
			ss.Bots[i].X += dx * lerpFactor
			ss.Bots[i].Y += dy * lerpFactor
			ss.Bots[i].Angle = math.Atan2(dy, dx)
			ss.Bots[i].Speed = SwarmBotSpeed
		} else {
			ss.Bots[i].Speed = 0
		}
	}

	if count == 0 {
		return 0
	}
	return totalDist / float64(count)
}
