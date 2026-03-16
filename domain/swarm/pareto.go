package swarm

import (
	"sort"
)

// ParetoObjective represents one objective value for multi-objective optimization.
type ParetoObjective struct {
	Name  string
	Value float64
}

// ParetoFront tracks non-dominated solutions in multi-objective evolution.
type ParetoFront struct {
	Fronts           [][]int     // fronts[0] = rank-0 (Pareto front), fronts[1] = rank-1, etc.
	BotCount         int
	Objectives       [][]float64 // [botIdx][objectiveIdx]
	CrowdingDistance  []float64   // [botIdx] pre-computed crowding distance
}

// ComputeParetoFronts assigns Pareto ranks to all bots using NSGA-II fast non-dominated sorting.
// Returns ParetoFront with fronts (layers of non-dominated solutions).
func ComputeParetoFronts(ss *SwarmState) *ParetoFront {
	n := len(ss.Bots)
	if n == 0 {
		return &ParetoFront{}
	}

	// Compute multi-objective fitness for each bot
	objectives := make([][]float64, n)
	for i := range ss.Bots {
		objectives[i] = botObjectives(&ss.Bots[i])
	}

	// Fast non-dominated sorting (NSGA-II style)
	dominationCount := make([]int, n)    // how many solutions dominate i
	dominated := make([][]int, n)        // solutions dominated by i

	for i := 0; i < n; i++ {
		dominated[i] = nil
		dominationCount[i] = 0
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			if dominates(objectives[i], objectives[j]) {
				dominated[i] = append(dominated[i], j)
			} else if dominates(objectives[j], objectives[i]) {
				dominationCount[i]++
			}
		}
	}

	// Build fronts
	var fronts [][]int
	var currentFront []int
	for i := 0; i < n; i++ {
		if dominationCount[i] == 0 {
			currentFront = append(currentFront, i)
		}
	}

	for len(currentFront) > 0 {
		fronts = append(fronts, currentFront)
		var nextFront []int
		for _, i := range currentFront {
			for _, j := range dominated[i] {
				dominationCount[j]--
				if dominationCount[j] == 0 {
					nextFront = append(nextFront, j)
				}
			}
		}
		currentFront = nextFront
	}

	pf := &ParetoFront{
		Fronts:          fronts,
		BotCount:        n,
		Objectives:      objectives,
		CrowdingDistance: make([]float64, n),
	}

	// Pre-compute crowding distances for all fronts
	for _, front := range fronts {
		dists := computeAllCrowdingDistances(pf, front)
		for i, idx := range front {
			pf.CrowdingDistance[idx] = dists[i]
		}
	}

	return pf
}

// botObjectives returns the multi-objective fitness vector for a bot.
// Objectives (all maximize):
//   [0] Deliveries (correct * 3 + wrong * 1)
//   [1] Exploration (total distance / 100)
//   [2] Efficiency (correct / max(total,1) * 100)
func botObjectives(bot *SwarmBot) []float64 {
	deliveries := float64(bot.Stats.CorrectDeliveries)*3 + float64(bot.Stats.WrongDeliveries)
	exploration := bot.Stats.TotalDistance / 100.0
	total := bot.Stats.TotalDeliveries
	if total == 0 {
		total = 1
	}
	efficiency := float64(bot.Stats.CorrectDeliveries) / float64(total) * 100
	return []float64{deliveries, exploration, efficiency}
}

// dominates returns true if solution a Pareto-dominates solution b.
// a dominates b if a is >= b in all objectives and strictly > in at least one.
func dominates(a, b []float64) bool {
	strictlyBetter := false
	for i := range a {
		if a[i] < b[i] {
			return false
		}
		if a[i] > b[i] {
			strictlyBetter = true
		}
	}
	return strictlyBetter
}

// ParetoRankFitness converts Pareto rank to a scalar fitness value for selection.
// Rank 0 (Pareto front) gets highest fitness. Uses pre-computed crowding distance.
func ParetoRankFitness(pf *ParetoFront, botIdx int) float64 {
	if botIdx < 0 || botIdx >= pf.BotCount {
		return 0
	}
	for rank, front := range pf.Fronts {
		for _, idx := range front {
			if idx == botIdx {
				baseFitness := float64(pf.BotCount-rank) * 100
				return baseFitness + pf.CrowdingDistance[botIdx]
			}
		}
	}
	return 0
}

// computeAllCrowdingDistances computes crowding distance for every member of a front at once.
// This avoids redundant re-sorting per bot — sorts once per objective for the whole front.
// Returns a map from bot index to crowding distance.
func computeAllCrowdingDistances(pf *ParetoFront, front []int) []float64 {
	n := len(front)
	// Indexed by position in front, not by bot index
	distances := make([]float64, n)

	if n <= 2 {
		for i := range distances {
			distances[i] = 50
		}
		return distances
	}

	// Build reverse lookup: botIdx → position in front
	posOf := make(map[int]int, n)
	for i, idx := range front {
		posOf[idx] = i
	}

	numObj := len(pf.Objectives[0])
	sorted := make([]int, n) // reuse across objectives

	for obj := 0; obj < numObj; obj++ {
		copy(sorted, front)
		sort.Slice(sorted, func(a, b int) bool {
			return pf.Objectives[sorted[a]][obj] < pf.Objectives[sorted[b]][obj]
		})

		// Boundary solutions get max distance
		distances[posOf[sorted[0]]] += 1000
		distances[posOf[sorted[n-1]]] += 1000

		// Range for normalization
		objRange := pf.Objectives[sorted[n-1]][obj] - pf.Objectives[sorted[0]][obj]
		if objRange < 0.001 {
			continue
		}

		// Inner solutions
		for i := 1; i < n-1; i++ {
			d := (pf.Objectives[sorted[i+1]][obj] - pf.Objectives[sorted[i-1]][obj]) / objRange
			distances[posOf[sorted[i]]] += d * 50
		}
	}

	return distances
}

// ParetoFrontSize returns the number of solutions on the Pareto front (rank 0).
func ParetoFrontSize(pf *ParetoFront) int {
	if len(pf.Fronts) == 0 {
		return 0
	}
	return len(pf.Fronts[0])
}

// ObjectiveNames returns the names of the objectives used in Pareto optimization.
func ObjectiveNames() []string {
	return []string{"Lieferungen", "Exploration", "Effizienz"}
}
