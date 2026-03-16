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
