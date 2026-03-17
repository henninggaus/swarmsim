package swarm

import (
	"math"
	"sort"
	"swarmsim/logger"
)

// InitBotParams initializes per-bot parameter values from program hints + noise.
func InitBotParams(ss *SwarmState) {
	ScanUsedParams(ss)
	for i := range ss.Bots {
		for p := 0; p < 26; p++ {
			if !ss.UsedParams[p] {
				continue
			}
			hint := GetParamHint(ss, p)
			noise := (ss.Rng.Float64() - 0.5) * math.Max(math.Abs(hint), 1) * 0.4 // ±20% of hint
			ss.Bots[i].ParamValues[p] = hint + noise
		}
		ss.Bots[i].Fitness = 0
	}
}

// RunEvolution performs one generation of genetic algorithm.
// Called every 1500 ticks when EvolutionOn.
func RunEvolution(ss *SwarmState) {
	n := len(ss.Bots)
	if n < 4 {
		return
	}

	// 1. Compute fitness (Pareto or scalar)
	if ss.ParetoEnabled {
		pf := ComputeParetoFronts(ss)
		ss.ParetoFront = pf
		for i := range ss.Bots {
			ss.Bots[i].Fitness = ParetoRankFitness(pf, i)
		}
	}

	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		return ss.Bots[indices[a]].Fitness > ss.Bots[indices[b]].Fitness
	})

	// 2. Top 30% are parents
	parentCount := n * 30 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	parents := indices[:parentCount]

	// 3. Record stats
	ss.BestFitness = ss.Bots[parents[0]].Fitness
	total := 0.0
	for i := range ss.Bots {
		total += ss.Bots[i].Fitness
	}
	ss.AvgFitness = total / float64(n)

	// Record fitness history for graph
	ss.FitnessHistory = append(ss.FitnessHistory, FitnessRecord{
		Best: ss.BestFitness,
		Avg:  ss.AvgFitness,
	})

	// 3b. Novelty Search blending (if enabled)
	if ss.NoveltyEnabled && ss.NoveltyArchive != nil {
		for i := range ss.Bots {
			ss.Bots[i].Behavior = ComputeBehavior(&ss.Bots[i], ss)
		}
		noveltyScores := ComputeNoveltyScores(ss)
		if noveltyScores != nil {
			alpha := ss.NoveltyArchive.Alpha
			for i := range ss.Bots {
				ss.Bots[i].Fitness = BlendFitness(ss.Bots[i].Fitness, noveltyScores[i], alpha)
			}
			behaviors := make([]BehaviorDescriptor, n)
			for i := range ss.Bots {
				behaviors[i] = ss.Bots[i].Behavior
			}
			UpdateNoveltyArchive(ss, behaviors, noveltyScores)
		}
		// Re-sort after blending
		sort.Slice(indices, func(a, b int) bool {
			return ss.Bots[indices[a]].Fitness > ss.Bots[indices[b]].Fitness
		})
		parents = indices[:parentCount]
	}

	// Genealogy: save old BotIDs
	oldBotIDs := make([]int, n)
	for i := range ss.Bots {
		oldBotIDs[i] = ss.Bots[i].BotID
	}

	// 4. Bottom 70% get crossover + mutation from parents
	for _, childIdx := range indices[parentCount:] {
		p1 := parents[ss.Rng.Intn(parentCount)]
		p2 := parents[ss.Rng.Intn(parentCount)]
		for p := 0; p < 26; p++ {
			if !ss.UsedParams[p] {
				continue
			}
			// Uniform crossover
			if ss.Rng.Float64() < 0.5 {
				ss.Bots[childIdx].ParamValues[p] = ss.Bots[p1].ParamValues[p]
			} else {
				ss.Bots[childIdx].ParamValues[p] = ss.Bots[p2].ParamValues[p]
			}
			// Mutation: 15% chance, Gaussian noise
			if ss.Rng.Float64() < 0.15 {
				sigma := math.Abs(ss.Bots[childIdx].ParamValues[p]) * 0.2
				if sigma < 1 {
					sigma = 1
				}
				ss.Bots[childIdx].ParamValues[p] += ss.Rng.NormFloat64() * sigma
			}
		}
		// Genealogy
		if ss.Genealogy != nil {
			ss.Bots[childIdx].ParentA = oldBotIDs[p1]
			ss.Bots[childIdx].ParentB = oldBotIDs[p2]
			ss.Bots[childIdx].BotID = AssignBotID(ss.Genealogy)
		}
	}

	// Parents keep their ID (elite) but get new BotIDs
	if ss.Genealogy != nil {
		for _, parentIdx := range parents {
			ss.Bots[parentIdx].ParentA = oldBotIDs[parentIdx]
			ss.Bots[parentIdx].ParentB = -1
			ss.Bots[parentIdx].BotID = AssignBotID(ss.Genealogy)
		}
		RecordGeneration(ss.Genealogy, ss.Bots, ss.Generation)
	}

	// 5. Reset all fitness for next generation
	for i := range ss.Bots {
		ss.Bots[i].Fitness = 0
	}
	ss.Generation++
	ss.EvolutionTimer = 0

	// Log evolution milestone
	logger.Info("EVOLUTION", "Gen %d abgeschlossen — Best: %.0f, Avg: %.0f (%d Eltern -> %d Kinder)",
		ss.Generation, ss.BestFitness, ss.AvgFitness, parentCount, n-parentCount)
}
