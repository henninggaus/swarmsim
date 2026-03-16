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
	Fronts     [][]int // fronts[0] = rank-0 (Pareto front), fronts[1] = rank-1, etc.
	BotCount   int
	Objectives [][]float64 // [botIdx][objectiveIdx]
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

	return &ParetoFront{
		Fronts:     fronts,
		BotCount:   n,
		Objectives: objectives,
	}
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
// Rank 0 (Pareto front) gets highest fitness. Uses crowding distance within each front.
func ParetoRankFitness(pf *ParetoFront, botIdx int) float64 {
	for rank, front := range pf.Fronts {
		for _, idx := range front {
			if idx == botIdx {
				// Base fitness from rank (higher rank = lower fitness)
				baseFitness := float64(pf.BotCount-rank) * 100
				// Add crowding distance bonus within front
				crowding := crowdingDistance(pf, front, botIdx)
				return baseFitness + crowding
			}
		}
	}
	return 0
}

// crowdingDistance estimates the crowding distance for a solution within its front.
// Solutions with more diverse objective values get higher crowding distance.
func crowdingDistance(pf *ParetoFront, front []int, targetIdx int) float64 {
	if len(front) <= 2 {
		return 50 // small front, give decent bonus
	}

	numObj := len(pf.Objectives[0])
	distances := make(map[int]float64)
	for _, idx := range front {
		distances[idx] = 0
	}

	for obj := 0; obj < numObj; obj++ {
		// Sort front by this objective
		sorted := make([]int, len(front))
		copy(sorted, front)
		sort.Slice(sorted, func(a, b int) bool {
			return pf.Objectives[sorted[a]][obj] < pf.Objectives[sorted[b]][obj]
		})

		// Boundary solutions get max distance
		distances[sorted[0]] += 1000
		distances[sorted[len(sorted)-1]] += 1000

		// Range for normalization
		objRange := pf.Objectives[sorted[len(sorted)-1]][obj] - pf.Objectives[sorted[0]][obj]
		if objRange < 0.001 {
			continue
		}

		// Inner solutions
		for i := 1; i < len(sorted)-1; i++ {
			d := (pf.Objectives[sorted[i+1]][obj] - pf.Objectives[sorted[i-1]][obj]) / objRange
			distances[sorted[i]] += d * 50
		}
	}

	return distances[targetIdx]
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
