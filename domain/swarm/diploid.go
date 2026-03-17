package swarm

import (
	"math"
	"swarmsim/logger"
)

// DiploidState manages advanced diploid genetics for the swarm.
// Each bot has two sets of chromosomes with dominance. Gene expression
// uses dominance rules: dominant alleles mask recessive ones.
// Heterozygote advantage provides fitness benefits for genetic diversity.
type DiploidState struct {
	NumGenes   int     // genes per chromosome (default 24)
	DomRate    float64 // fraction of genes that are dominant (default 0.5)
	HetBonus   float64 // fitness bonus for heterozygous genes (default 0.1)
	MutRate    float64 // mutation rate per allele (default 0.05)

	Genomes []AdvDiploidGenome // per-bot genomes
	Generation int

	// Stats
	AvgHeterozygosity float64
	AvgDominance      float64
	GeneticDiversity  float64
}

// AdvDiploidGenome holds two chromosomes with dominance for a bot.
type AdvDiploidGenome struct {
	ChromA    []Allele  // maternal chromosome
	ChromB    []Allele  // paternal chromosome
	Expressed []float64 // phenotype: expressed gene values
}

// Allele represents a single gene variant with dominance.
type Allele struct {
	Value    float64 // gene value
	Dominant bool    // whether this allele is dominant
}

// InitDiploid sets up the diploid genetics system.
func InitDiploid(ss *SwarmState, numGenes int) {
	if numGenes < 8 {
		numGenes = 8
	}
	if numGenes > 64 {
		numGenes = 64
	}

	n := len(ss.Bots)
	ds := &DiploidState{
		NumGenes: numGenes,
		DomRate:  0.5,
		HetBonus: 0.1,
		MutRate:  0.05,
		Genomes:  make([]AdvDiploidGenome, n),
	}

	for i := 0; i < n; i++ {
		ds.Genomes[i] = randomAdvDiploidGenome(ss, numGenes, ds.DomRate)
		expressAdvGenome(&ds.Genomes[i])
	}

	ss.Diploid = ds
	logger.Info("DIPLOID", "Initialisiert: %d Bots, %d Gene, DomRate=%.0f%%",
		n, numGenes, ds.DomRate*100)
}

// ClearDiploid disables the diploid system.
func ClearDiploid(ss *SwarmState) {
	ss.Diploid = nil
	ss.DiploidOn = false
}

// randomAdvDiploidGenome creates a random diploid genome.
func randomAdvDiploidGenome(ss *SwarmState, numGenes int, domRate float64) AdvDiploidGenome {
	g := AdvDiploidGenome{
		ChromA:    make([]Allele, numGenes),
		ChromB:    make([]Allele, numGenes),
		Expressed: make([]float64, numGenes),
	}
	for j := 0; j < numGenes; j++ {
		g.ChromA[j] = Allele{
			Value:    (ss.Rng.Float64() - 0.5) * 2.0,
			Dominant: ss.Rng.Float64() < domRate,
		}
		g.ChromB[j] = Allele{
			Value:    (ss.Rng.Float64() - 0.5) * 2.0,
			Dominant: ss.Rng.Float64() < domRate,
		}
	}
	return g
}

// expressAdvGenome computes the phenotype from two chromosomes using dominance.
func expressAdvGenome(g *AdvDiploidGenome) {
	for i := range g.ChromA {
		a := g.ChromA[i]
		b := g.ChromB[i]

		switch {
		case a.Dominant && !b.Dominant:
			g.Expressed[i] = a.Value
		case !a.Dominant && b.Dominant:
			g.Expressed[i] = b.Value
		case a.Dominant && b.Dominant:
			g.Expressed[i] = (a.Value + b.Value) / 2
		default:
			g.Expressed[i] = (a.Value + b.Value) / 2 * 0.9
		}
	}
}

// TickDiploid applies gene expression to bot behavior.
func TickDiploid(ss *SwarmState) {
	ds := ss.Diploid
	if ds == nil {
		return
	}

	n := len(ss.Bots)
	if len(ds.Genomes) != n {
		return
	}

	for i := range ss.Bots {
		g := &ds.Genomes[i]

		if len(g.Expressed) >= 4 {
			speedMod := math.Tanh(g.Expressed[0]) * 0.3
			ss.Bots[i].Speed = SwarmBotSpeed * (0.7 + speedMod)

			turnBias := math.Tanh(g.Expressed[1]) * 0.1
			ss.Bots[i].Angle += turnBias

			if len(g.Expressed) >= 7 {
				r := uint8(128 + math.Tanh(g.Expressed[4])*127)
				green := uint8(128 + math.Tanh(g.Expressed[5])*127)
				b := uint8(128 + math.Tanh(g.Expressed[6])*127)
				ss.Bots[i].LEDColor = [3]uint8{r, green, b}
			}
		}
	}
}

// EvolveDiploid performs sexual reproduction with advanced diploid genetics.
func EvolveDiploid(ss *SwarmState, sortedIndices []int) {
	ds := ss.Diploid
	if ds == nil {
		return
	}

	n := len(ss.Bots)
	if len(ds.Genomes) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 30 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 2
	if eliteCount > parentCount {
		eliteCount = parentCount
	}

	parents := make([]AdvDiploidGenome, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parents[i] = cloneAdvDiploidGenome(ds.Genomes[sortedIndices[i]])
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			continue
		}

		p1 := ss.Rng.Intn(parentCount)
		p2 := ss.Rng.Intn(parentCount)
		for p2 == p1 && parentCount > 1 {
			p2 = ss.Rng.Intn(parentCount)
		}

		childA := advMeiosis(ss, &parents[p1], ds.MutRate)
		childB := advMeiosis(ss, &parents[p2], ds.MutRate)

		ds.Genomes[botIdx] = AdvDiploidGenome{
			ChromA:    childA,
			ChromB:    childB,
			Expressed: make([]float64, ds.NumGenes),
		}
		expressAdvGenome(&ds.Genomes[botIdx])
	}

	updateDiploidStats(ds)
	ds.Generation++

	logger.Info("DIPLOID", "Gen %d: Heterozygositaet=%.2f, Diversitaet=%.3f",
		ds.Generation, ds.AvgHeterozygosity, ds.GeneticDiversity)
}

// advMeiosis creates a gamete by crossing over two chromosomes and mutating.
func advMeiosis(ss *SwarmState, parent *AdvDiploidGenome, mutRate float64) []Allele {
	numGenes := len(parent.ChromA)
	gamete := make([]Allele, numGenes)

	crossPoint := ss.Rng.Intn(numGenes)

	for j := 0; j < numGenes; j++ {
		if j < crossPoint {
			gamete[j] = parent.ChromA[j]
		} else {
			gamete[j] = parent.ChromB[j]
		}

		if ss.Rng.Float64() < mutRate {
			gamete[j].Value += ss.Rng.NormFloat64() * 0.3
			if ss.Rng.Float64() < 0.05 {
				gamete[j].Dominant = !gamete[j].Dominant
			}
		}
	}

	return gamete
}

// cloneAdvDiploidGenome deep-copies a genome.
func cloneAdvDiploidGenome(src AdvDiploidGenome) AdvDiploidGenome {
	dst := AdvDiploidGenome{
		ChromA:    make([]Allele, len(src.ChromA)),
		ChromB:    make([]Allele, len(src.ChromB)),
		Expressed: make([]float64, len(src.Expressed)),
	}
	copy(dst.ChromA, src.ChromA)
	copy(dst.ChromB, src.ChromB)
	copy(dst.Expressed, src.Expressed)
	return dst
}

// updateDiploidStats computes population genetics statistics.
func updateDiploidStats(ds *DiploidState) {
	n := len(ds.Genomes)
	if n == 0 {
		return
	}

	totalHet := 0.0
	totalDom := 0.0
	numGenes := ds.NumGenes

	for _, g := range ds.Genomes {
		for j := 0; j < numGenes && j < len(g.ChromA); j++ {
			diff := math.Abs(g.ChromA[j].Value - g.ChromB[j].Value)
			if diff > 0.1 {
				totalHet++
			}
			if g.ChromA[j].Dominant {
				totalDom++
			}
			if g.ChromB[j].Dominant {
				totalDom++
			}
		}
	}

	totalAlleles := float64(n * numGenes)
	ds.AvgHeterozygosity = totalHet / totalAlleles
	ds.AvgDominance = totalDom / (totalAlleles * 2)

	if numGenes > 0 {
		varSum := 0.0
		for g := 0; g < numGenes; g++ {
			mean := 0.0
			for i := range ds.Genomes {
				if g < len(ds.Genomes[i].Expressed) {
					mean += ds.Genomes[i].Expressed[g]
				}
			}
			mean /= float64(n)
			for i := range ds.Genomes {
				if g < len(ds.Genomes[i].Expressed) {
					d := ds.Genomes[i].Expressed[g] - mean
					varSum += d * d
				}
			}
		}
		ds.GeneticDiversity = math.Sqrt(varSum / totalAlleles)
	}
}

// Heterozygosity returns the average heterozygosity.
func Heterozygosity(ds *DiploidState) float64 {
	if ds == nil {
		return 0
	}
	return ds.AvgHeterozygosity
}

// DiploidDiversity returns the genetic diversity.
func DiploidDiversity(ds *DiploidState) float64 {
	if ds == nil {
		return 0
	}
	return ds.GeneticDiversity
}

// HeterozygoteAdvantage computes the fitness bonus from heterozygosity for a bot.
func HeterozygoteAdvantage(ds *DiploidState, botIdx int) float64 {
	if ds == nil || botIdx < 0 || botIdx >= len(ds.Genomes) {
		return 0
	}
	g := &ds.Genomes[botIdx]
	hetCount := 0
	for j := range g.ChromA {
		if j < len(g.ChromB) {
			diff := math.Abs(g.ChromA[j].Value - g.ChromB[j].Value)
			if diff > 0.1 {
				hetCount++
			}
		}
	}
	return float64(hetCount) * ds.HetBonus / float64(ds.NumGenes)
}
