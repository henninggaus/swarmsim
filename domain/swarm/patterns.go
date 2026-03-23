package swarm

import "math"

// PatternType identifies a detected swarm formation pattern.
type PatternType int

const (
	PatternNone       PatternType = iota
	PatternCluster                // tight group
	PatternLine                   // linear arrangement
	PatternCircle                 // circular arrangement
	PatternScattered              // spread out randomly
	PatternStream                 // moving in same direction
	PatternVortex                 // circular movement
)

// PatternResult holds the analysis of current swarm patterns.
type PatternResult struct {
	Primary       PatternType // dominant pattern
	PrimaryScore  float64     // confidence 0-1
	Secondary     PatternType // second-most dominant
	SecondScore   float64
	ClusterCount  int         // number of distinct clusters
	Entropy       float64     // movement entropy (0=uniform, 1=chaotic)
	Alignment     float64     // heading alignment (0=random, 1=all same direction)
	Cohesion      float64     // spatial cohesion (0=scattered, 1=tight)
	Circularity   float64     // how circular the arrangement is (0-1)
}

// PatternName returns the German display name for a pattern type.
func PatternName(p PatternType) string {
	switch p {
	case PatternCluster:
		return "Cluster"
	case PatternLine:
		return "Linie"
	case PatternCircle:
		return "Kreis"
	case PatternScattered:
		return "Verstreut"
	case PatternStream:
		return "Strom"
	case PatternVortex:
		return "Wirbel"
	}
	return "Keins"
}

// DetectPatterns analyzes the swarm and detects emergent formations.
func DetectPatterns(ss *SwarmState) PatternResult {
	n := len(ss.Bots)
	if n < 5 {
		return PatternResult{Primary: PatternNone}
	}

	result := PatternResult{}

	// 1. Compute heading alignment (how aligned are bot headings?)
	sumCos, sumSin := 0.0, 0.0
	for i := range ss.Bots {
		sumCos += math.Cos(ss.Bots[i].Angle)
		sumSin += math.Sin(ss.Bots[i].Angle)
	}
	result.Alignment = math.Sqrt(sumCos*sumCos+sumSin*sumSin) / float64(n)

	// 2. Compute spatial cohesion (inverse of normalized spread)
	cx, cy := ss.SwarmCenterX, ss.SwarmCenterY
	if cx == 0 && cy == 0 {
		ComputeSwarmCenter(ss)
		cx, cy = ss.SwarmCenterX, ss.SwarmCenterY
	}
	maxDist := ss.ArenaW * 0.5
	avgDist := 0.0
	for i := range ss.Bots {
		dx := ss.Bots[i].X - cx
		dy := ss.Bots[i].Y - cy
		avgDist += math.Sqrt(dx*dx + dy*dy)
	}
	avgDist /= float64(n)
	result.Cohesion = 1.0 - math.Min(avgDist/maxDist, 1.0)

	// 3. Compute circularity (how uniformly distributed around center?)
	if avgDist > 20 {
		angleVariance := 0.0
		// Bin angles into 8 sectors
		sectors := [8]int{}
		for i := range ss.Bots {
			dx := ss.Bots[i].X - cx
			dy := ss.Bots[i].Y - cy
			angle := math.Atan2(dy, dx)
			if angle < 0 {
				angle += 2 * math.Pi
			}
			sector := int(angle / (math.Pi / 4))
			if sector >= 8 {
				sector = 7
			}
			sectors[sector]++
		}
		expected := float64(n) / 8.0
		for _, count := range sectors {
			diff := float64(count) - expected
			angleVariance += diff * diff
		}
		angleVariance /= 8.0
		// Normalize: low variance = circular
		maxVar := expected * expected
		result.Circularity = 1.0 - math.Min(angleVariance/maxVar, 1.0)

		// Check if distances from center are uniform (ring vs blob)
		distVariance := 0.0
		for i := range ss.Bots {
			dx := ss.Bots[i].X - cx
			dy := ss.Bots[i].Y - cy
			d := math.Sqrt(dx*dx+dy*dy) - avgDist
			distVariance += d * d
		}
		distVariance /= float64(n)
		ringScore := 1.0 - math.Min(distVariance/(avgDist*avgDist+1), 1.0)
		result.Circularity *= ringScore
	}

	// 4. Vortex detection (angular velocity around center)
	vortexScore := 0.0
	for i := range ss.Bots {
		dx := ss.Bots[i].X - cx
		dy := ss.Bots[i].Y - cy
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 10 {
			continue
		}
		// Expected tangential heading for CCW rotation
		tangent := math.Atan2(-dx, dy)
		headingDiff := angleDiff(ss.Bots[i].Angle, tangent)
		if headingDiff < math.Pi/4 {
			vortexScore += 1.0
		}
		// Check CW rotation
		tangentCW := math.Atan2(dx, -dy)
		if angleDiff(ss.Bots[i].Angle, tangentCW) < math.Pi/4 {
			vortexScore += 1.0
		}
	}
	vortexScore /= float64(n)

	// 5. Cluster counting (simple distance-based clustering)
	visited := make([]bool, n)
	result.ClusterCount = 0
	clusterThreshold := 60.0
	clusterThreshSq := clusterThreshold * clusterThreshold
	for i := range ss.Bots {
		if visited[i] {
			continue
		}
		result.ClusterCount++
		// BFS flood fill
		queue := []int{i}
		visited[i] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			bx, by := ss.Bots[curr].X, ss.Bots[curr].Y
			if ss.Hash != nil {
				// Spatial hash: only check nearby bots — O(k) per BFS step
				candidates := ss.Hash.Query(bx, by, clusterThreshold)
				for _, j := range candidates {
					if visited[j] || j < 0 || j >= n {
						continue
					}
					dx := bx - ss.Bots[j].X
					dy := by - ss.Bots[j].Y
					if dx*dx+dy*dy < clusterThreshSq {
						visited[j] = true
						queue = append(queue, j)
					}
				}
			} else {
				// Fallback: brute-force O(n) per BFS step
				for j := range ss.Bots {
					if visited[j] {
						continue
					}
					dx := bx - ss.Bots[j].X
					dy := by - ss.Bots[j].Y
					if dx*dx+dy*dy < clusterThreshSq {
						visited[j] = true
						queue = append(queue, j)
					}
				}
			}
		}
	}

	// 6. Movement entropy (diversity of headings)
	headingBins := [12]int{} // 30° bins
	for i := range ss.Bots {
		if ss.Bots[i].Speed < 0.1 {
			continue
		}
		a := ss.Bots[i].Angle
		if a < 0 {
			a += 2 * math.Pi
		}
		bin := int(a / (math.Pi / 6))
		if bin >= 12 {
			bin = 11
		}
		headingBins[bin]++
	}
	movingCount := 0
	for _, c := range headingBins {
		movingCount += c
	}
	if movingCount > 0 {
		entropy := 0.0
		for _, c := range headingBins {
			if c > 0 {
				p := float64(c) / float64(movingCount)
				entropy -= p * math.Log2(p)
			}
		}
		maxEntropy := math.Log2(12)
		result.Entropy = entropy / maxEntropy
	}

	// Score each pattern type
	scores := map[PatternType]float64{
		PatternCluster:   result.Cohesion * (1 - result.Circularity) * 0.8,
		PatternScattered: (1 - result.Cohesion) * result.Entropy,
		PatternStream:    result.Alignment * (1 - result.Circularity),
		PatternCircle:    result.Circularity * result.Cohesion,
		PatternVortex:    vortexScore * result.Cohesion,
	}
	// Line detection: high cohesion along one axis
	if result.Cohesion > 0.3 && result.ClusterCount <= 2 {
		// Check if spread is much larger in one direction
		varX, varY := 0.0, 0.0
		for i := range ss.Bots {
			dx := ss.Bots[i].X - cx
			dy := ss.Bots[i].Y - cy
			varX += dx * dx
			varY += dy * dy
		}
		ratio := math.Max(varX, varY) / (math.Min(varX, varY) + 1)
		if ratio > 3 {
			scores[PatternLine] = math.Min(ratio/10, 1.0) * result.Cohesion
		}
	}

	// Find primary and secondary
	result.Primary = PatternNone
	result.PrimaryScore = 0
	for pt, score := range scores {
		if score > result.PrimaryScore {
			result.Secondary = result.Primary
			result.SecondScore = result.PrimaryScore
			result.Primary = pt
			result.PrimaryScore = score
		} else if score > result.SecondScore {
			result.Secondary = pt
			result.SecondScore = score
		}
	}

	// Minimum threshold
	if result.PrimaryScore < 0.15 {
		result.Primary = PatternScattered
		result.PrimaryScore = 1.0 - result.Cohesion
	}

	return result
}

// angleDiff returns the absolute difference between two angles (0 to pi).
func angleDiff(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > math.Pi {
		d = 2*math.Pi - d
	}
	return d
}
