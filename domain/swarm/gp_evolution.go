package swarm

import (
	"sort"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
)

// EvaluateGPFitness computes fitness for a bot based on its lifetime stats.
func EvaluateGPFitness(bot *SwarmBot) float64 {
	f := float64(bot.Stats.TotalDeliveries)*30 +
		float64(bot.Stats.TotalPickups)*15 +
		bot.Stats.TotalDistance*0.01 -
		float64(bot.Stats.AntiStuckCount)*10 -
		float64(bot.Stats.TicksIdle)*0.05
	return f
}

// RunGPEvolution performs one generation of genetic programming evolution.
// Each bot has its own program; the best programs survive and reproduce.
func RunGPEvolution(ss *SwarmState) {
	n := len(ss.Bots)
	if n < 4 {
		return
	}

	// 1. Evaluate fitness for all bots
	fitnesses := make([]float64, n)
	if ss.ParetoEnabled {
		// Multi-objective Pareto ranking (NSGA-II style)
		pf := ComputeParetoFronts(ss)
		ss.ParetoFront = pf
		for i := range ss.Bots {
			fitnesses[i] = ParetoRankFitness(pf, i)
		}
	} else {
		for i := range ss.Bots {
			fitnesses[i] = EvaluateGPFitness(&ss.Bots[i])
		}
	}

	// 2. Sort indices by fitness (descending)
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		return fitnesses[indices[a]] > fitnesses[indices[b]]
	})

	// 3. Record stats
	ss.BestFitness = fitnesses[indices[0]]
	total := 0.0
	for _, f := range fitnesses {
		total += f
	}
	ss.AvgFitness = total / float64(n)

	ss.FitnessHistory = append(ss.FitnessHistory, FitnessRecord{
		Best: ss.BestFitness,
		Avg:  ss.AvgFitness,
	})

	// 4. Top 20% are parents (elite)
	parentCount := n * 20 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 3
	if eliteCount > parentCount {
		eliteCount = parentCount
	}

	// 5. Save elite programs (deep copy, no mutation)
	elitePrograms := make([]*swarmscript.SwarmProgram, eliteCount)
	for i := 0; i < eliteCount; i++ {
		elitePrograms[i] = swarmscript.CopyProgram(ss.Bots[indices[i]].OwnProgram)
	}

	// 6. Save all parent programs for crossover
	parentPrograms := make([]*swarmscript.SwarmProgram, parentCount)
	for i := 0; i < parentCount; i++ {
		parentPrograms[i] = swarmscript.CopyProgram(ss.Bots[indices[i]].OwnProgram)
	}

	// 7. Generate new population
	freshCount := n * 10 / 100 // 10% fresh random
	if freshCount < 1 {
		freshCount = 1
	}

	for rank, botIdx := range indices {
		if rank < eliteCount {
			// Elite: keep as-is (already copied above)
			ss.Bots[botIdx].OwnProgram = elitePrograms[rank]
		} else if rank >= n-freshCount {
			// Fresh random programs
			numRules := 8 + ss.Rng.Intn(8) // 8-15 rules
			ss.Bots[botIdx].OwnProgram = swarmscript.GenerateRandomProgram(ss.Rng, numRules)
		} else {
			// Crossover + mutation
			p1 := parentPrograms[ss.Rng.Intn(parentCount)]
			p2 := parentPrograms[ss.Rng.Intn(parentCount)]
			child := swarmscript.CrossoverPrograms(ss.Rng, p1, p2)
			swarmscript.MutateProgram(ss.Rng, child)
			ss.Bots[botIdx].OwnProgram = child
		}

		// Reset fitness-relevant stats for next generation
		ss.Bots[botIdx].Stats.TotalDeliveries = 0
		ss.Bots[botIdx].Stats.TotalPickups = 0
		ss.Bots[botIdx].Stats.TotalDistance = 0
		ss.Bots[botIdx].Stats.AntiStuckCount = 0
		ss.Bots[botIdx].Stats.TicksIdle = 0
		ss.Bots[botIdx].Stats.TicksAlive = 0
		ss.Bots[botIdx].Fitness = 0
	}

	ss.GPGeneration++

	// Log GP generation milestone
	bestRules := 0
	if ss.Bots[indices[0]].OwnProgram != nil {
		bestRules = len(ss.Bots[indices[0]].OwnProgram.Rules)
	}
	logger.Info("GP", "Gen %d — Best: %.0f (%d Regeln), Avg: %.0f, %d Elite + %d Crossover + %d Neue",
		ss.GPGeneration, ss.BestFitness, bestRules, ss.AvgFitness, eliteCount, n-eliteCount-freshCount, freshCount)
}

// InitGP initializes genetic programming: each bot gets a random program.
func InitGP(ss *SwarmState) {
	for i := range ss.Bots {
		numRules := 8 + ss.Rng.Intn(8) // 8-15 rules
		ss.Bots[i].OwnProgram = swarmscript.GenerateRandomProgram(ss.Rng, numRules)
		ss.Bots[i].Fitness = 0
	}
	ss.GPGeneration = 0
	ss.GPTimer = 0
	ss.FitnessHistory = nil
}

// InitGPSeeded initializes GP with 50% mutated seed program and 50% random.
func InitGPSeeded(ss *SwarmState, seedProgram *swarmscript.SwarmProgram) {
	for i := range ss.Bots {
		if i%2 == 0 && seedProgram != nil {
			// Seeded: copy + mutate the seed program
			ss.Bots[i].OwnProgram = swarmscript.CopyProgram(seedProgram)
			swarmscript.MutateProgram(ss.Rng, ss.Bots[i].OwnProgram)
			swarmscript.MutateProgram(ss.Rng, ss.Bots[i].OwnProgram) // double mutate for variety
		} else {
			// Random
			numRules := 8 + ss.Rng.Intn(8)
			ss.Bots[i].OwnProgram = swarmscript.GenerateRandomProgram(ss.Rng, numRules)
		}
		ss.Bots[i].Fitness = 0
	}
	ss.GPGeneration = 0
	ss.GPTimer = 0
	ss.FitnessHistory = nil
}

// ClearGP disables GP and removes per-bot programs.
func ClearGP(ss *SwarmState) {
	for i := range ss.Bots {
		ss.Bots[i].OwnProgram = nil
	}
	ss.GPEnabled = false
	ss.GPGeneration = 0
	ss.GPTimer = 0
}

// GPBestProgramText returns the program text of the bot with highest fitness.
func GPBestProgramText(ss *SwarmState) string {
	if len(ss.Bots) == 0 {
		return ""
	}
	bestIdx := 0
	bestFit := EvaluateGPFitness(&ss.Bots[0])
	for i := 1; i < len(ss.Bots); i++ {
		f := EvaluateGPFitness(&ss.Bots[i])
		if f > bestFit {
			bestFit = f
			bestIdx = i
		}
	}
	if ss.Bots[bestIdx].OwnProgram == nil {
		return ""
	}
	return swarmscript.ProgramToText(ss.Bots[bestIdx].OwnProgram)
}
