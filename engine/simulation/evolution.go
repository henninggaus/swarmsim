package simulation

import (
	"math/rand"
	"sort"
	"swarmsim/domain/bot"
	"swarmsim/domain/genetics"
)

// EvolveGeneration runs one evolution step on the given bots, grouped by type.
// Returns (bestFitness, avgFitness).
func EvolveGeneration(allBots []bot.Bot, rng *rand.Rand, mutRate, mutSigma, eliteRatio float64) (float64, float64) {
	// Group bots by type
	groups := make(map[bot.BotType][]bot.Bot)
	for _, b := range allBots {
		if b.IsAlive() {
			groups[b.Type()] = append(groups[b.Type()], b)
		}
	}

	var totalFit float64
	var totalCount int
	var bestFit float64

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}

		// Sort by fitness descending
		sort.Slice(group, func(i, j int) bool {
			return group[i].GetBase().Fitness() > group[j].GetBase().Fitness()
		})

		// Track stats
		for _, b := range group {
			f := b.GetBase().Fitness()
			totalFit += f
			totalCount++
			if f > bestFit {
				bestFit = f
			}
		}

		eliteCount := int(float64(len(group)) * eliteRatio)
		if eliteCount < 1 {
			eliteCount = 1
		}

		// Collect elite genomes
		eliteGenomes := make([]bot.Genome, eliteCount)
		for i := 0; i < eliteCount; i++ {
			eliteGenomes[i] = *group[i].GetGenome()
		}

		// Top eliteCount keep their genome, rest get children
		for i := eliteCount; i < len(group); i++ {
			p1 := eliteGenomes[rng.Intn(eliteCount)]
			p2 := eliteGenomes[rng.Intn(eliteCount)]
			child := genetics.Crossover(p1, p2, rng)
			genetics.Mutate(&child, rng, mutRate, mutSigma)
			base := group[i].GetBase()
			base.Genome = child
			base.ApplyGenomeSpeed()
		}

		// Reset fitness for all bots in this group
		for _, b := range group {
			b.GetBase().ResetFitness()
			b.GetBase().Energy = b.GetBase().MaxEnergy
		}
	}

	avgFit := 0.0
	if totalCount > 0 {
		avgFit = totalFit / float64(totalCount)
	}
	return bestFit, avgFit
}

// CollectGenomes returns all genomes grouped by type (for scenario persistence).
func CollectGenomes(allBots []bot.Bot) map[bot.BotType][]bot.Genome {
	result := make(map[bot.BotType][]bot.Genome)
	for _, b := range allBots {
		g := *b.GetGenome()
		result[b.Type()] = append(result[b.Type()], g)
	}
	return result
}
