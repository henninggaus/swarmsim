package swarm

import (
	"math"
	"math/rand"
	"sort"
)

// DiploidGenome represents a diploid (two-copy) genome for sexual reproduction.
// Each gene has two alleles; the expressed value is their average (co-dominance).
type DiploidGenome struct {
	AllelesA [26]float64 // maternal alleles
	AllelesB [26]float64 // paternal alleles
}

// Express returns the expressed (phenotypic) value of gene p as the average of both alleles.
func (d *DiploidGenome) Express(p int) float64 {
	return (d.AllelesA[p] + d.AllelesB[p]) / 2.0
}

// MateSelection picks a mate for bot at index i using fitness-proportionate selection
// from bots within mating range. Returns -1 if no mate found.
func MateSelection(ss *SwarmState, i int, matingRange float64) int {
	bot := &ss.Bots[i]
	n := len(ss.Bots)

	type candidate struct {
		idx     int
		fitness float64
	}
	var candidates []candidate

	for j := 0; j < n; j++ {
		if j == i {
			continue
		}
		dx := ss.Bots[j].X - bot.X
		dy := ss.Bots[j].Y - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > matingRange {
			continue
		}
		f := EvaluateGPFitness(&ss.Bots[j])
		if f <= 0 {
			f = 0.01 // minimum weight
		}
		candidates = append(candidates, candidate{idx: j, fitness: f})
	}

	if len(candidates) == 0 {
		return -1
	}

	// Fitness-proportionate (roulette wheel) selection
	totalFit := 0.0
	for _, c := range candidates {
		totalFit += c.fitness
	}
	spin := ss.Rng.Float64() * totalFit
	cumulative := 0.0
	for _, c := range candidates {
		cumulative += c.fitness
		if cumulative >= spin {
			return c.idx
		}
	}
	return candidates[len(candidates)-1].idx
}

// DiploidCrossover creates a child DiploidGenome from two parents.
// Each allele in the child comes from one random parent allele (meiosis simulation).
func DiploidCrossover(rng *rand.Rand, parentA, parentB *DiploidGenome) DiploidGenome {
	var child DiploidGenome
	for p := 0; p < 26; p++ {
		// Child allele A: from parentA (50% maternal, 50% paternal allele)
		if rng.Float64() < 0.5 {
			child.AllelesA[p] = parentA.AllelesA[p]
		} else {
			child.AllelesA[p] = parentA.AllelesB[p]
		}
		// Child allele B: from parentB (50% maternal, 50% paternal allele)
		if rng.Float64() < 0.5 {
			child.AllelesB[p] = parentB.AllelesA[p]
		} else {
			child.AllelesB[p] = parentB.AllelesB[p]
		}
	}
	return child
}

// MutateDiploid applies random mutation to a diploid genome.
// mutRate is the per-gene probability, sigma is the mutation strength.
func MutateDiploid(rng *rand.Rand, genome *DiploidGenome, mutRate, sigma float64) {
	for p := 0; p < 26; p++ {
		if rng.Float64() < mutRate {
			genome.AllelesA[p] += rng.NormFloat64() * sigma
		}
		if rng.Float64() < mutRate {
			genome.AllelesB[p] += rng.NormFloat64() * sigma
		}
	}
}

// RunSexualEvolution performs one generation of sexual reproduction.
// Uses diploid genetics with mate selection, crossover, and mutation.
func RunSexualEvolution(ss *SwarmState) {
	n := len(ss.Bots)
	if n < 4 {
		return
	}

	// 1. Evaluate fitness
	fitnesses := make([]float64, n)
	for i := range ss.Bots {
		fitnesses[i] = EvaluateGPFitness(&ss.Bots[i])
	}

	// 2. Sort by fitness
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

	// 4. Initialize diploid genomes if needed
	for i := range ss.Bots {
		if ss.Bots[i].DiploidGenome == nil {
			g := &DiploidGenome{}
			for p := 0; p < 26; p++ {
				g.AllelesA[p] = ss.Bots[i].ParamValues[p]
				g.AllelesB[p] = ss.Bots[i].ParamValues[p]
			}
			ss.Bots[i].DiploidGenome = g
		}
	}

	// 5. Elite preservation (top 10%)
	eliteCount := n * 10 / 100
	if eliteCount < 2 {
		eliteCount = 2
	}

	// Save elite diploid genomes
	type savedDiploid struct {
		genome DiploidGenome
	}
	eliteGenomes := make([]savedDiploid, eliteCount)
	for i := 0; i < eliteCount; i++ {
		if ss.Bots[indices[i]].DiploidGenome != nil {
			eliteGenomes[i].genome = *ss.Bots[indices[i]].DiploidGenome
		}
	}

	// 6. Generate offspring via sexual reproduction
	matingRange := 200.0
	for rank, botIdx := range indices {
		if rank < eliteCount {
			// Elite: keep genome
			if ss.Bots[botIdx].DiploidGenome == nil {
				ss.Bots[botIdx].DiploidGenome = &DiploidGenome{}
			}
			*ss.Bots[botIdx].DiploidGenome = eliteGenomes[rank].genome
		} else {
			// Select parents via fitness-proportionate mate selection
			p1 := indices[ss.Rng.Intn(eliteCount*2)] // bias toward top performers
			if p1 >= n {
				p1 = indices[0]
			}
			p2 := MateSelection(ss, p1, matingRange)
			if p2 < 0 {
				// No mate in range: self with mutation
				p2 = indices[ss.Rng.Intn(eliteCount)]
			}

			parentA := ss.Bots[p1].DiploidGenome
			parentB := ss.Bots[p2].DiploidGenome
			if parentA == nil || parentB == nil {
				continue
			}

			child := DiploidCrossover(ss.Rng, parentA, parentB)
			MutateDiploid(ss.Rng, &child, 0.10, 2.0)

			if ss.Bots[botIdx].DiploidGenome == nil {
				ss.Bots[botIdx].DiploidGenome = &DiploidGenome{}
			}
			*ss.Bots[botIdx].DiploidGenome = child
		}

		// Express diploid genome to param values
		if ss.Bots[botIdx].DiploidGenome != nil {
			for p := 0; p < 26; p++ {
				ss.Bots[botIdx].ParamValues[p] = ss.Bots[botIdx].DiploidGenome.Express(p)
			}
		}
		ss.Bots[botIdx].Fitness = 0
	}

	ss.Generation++
	ss.EvolutionTimer = 0
}
