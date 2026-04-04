package factory

import "math"

// nearestPoint finds the index of the closest point to (x, y) from a list of center coordinates.
// Returns the index and distance. Returns -1 if points is empty.
func nearestPoint(x, y float64, points [][2]float64) (int, float64) {
	bestIdx := -1
	bestDist := math.MaxFloat64
	for i, p := range points {
		dx := p[0] - x
		dy := p[1] - y
		d := dx*dx + dy*dy // squared distance (cheaper, same ordering)
		if d < bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	if bestIdx >= 0 {
		bestDist = math.Sqrt(bestDist) // only sqrt the winner
	}
	return bestIdx, bestDist
}
