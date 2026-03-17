package swarm

import (
	"math"
	"swarmsim/logger"
)

// MetaEvoState manages meta-evolution: evolving evolution parameters.
// Each bot carries its own mutation rate, crossover rate, selection
// pressure, and other evolutionary hyperparameters. These meta-parameters
// are themselves subject to evolution — the swarm learns HOW to learn.
type MetaEvoState struct {
	Params []MetaParams // per-bot evolution parameters

	// Population-level statistics
	AvgMutationRate    float64
	AvgCrossoverRate   float64
	AvgSelectionPress  float64
	AvgExplorationRate float64
	Diversity          float64 // diversity of meta-params across population
	Generation         int
}

// MetaParams holds per-bot evolutionary hyperparameters.
type MetaParams struct {
	MutationRate    float64 // probability of weight mutation (0.01-0.5)
	MutationSize    float64 // std dev of mutation (0.05-1.0)
	CrossoverRate   float64 // probability of crossover per weight (0.1-0.9)
	SelectionPress  float64 // how strongly fitness affects selection (0.5-5.0)
	ExplorationRate float64 // tendency to explore vs exploit (0.0-1.0)
	EliteRatio      float64 // fraction of population preserved (0.01-0.2)

	// Self-adaptation tracking
	FitnessHistory []float64 // recent fitness values
	Improvement    float64   // fitness improvement rate
}

// InitMetaEvolution sets up the meta-evolution system.
func InitMetaEvolution(ss *SwarmState) {
	n := len(ss.Bots)
	me := &MetaEvoState{
		Params: make([]MetaParams, n),
	}

	for i := 0; i < n; i++ {
		me.Params[i] = MetaParams{
			MutationRate:    0.05 + ss.Rng.Float64()*0.15,
			MutationSize:    0.1 + ss.Rng.Float64()*0.3,
			CrossoverRate:   0.3 + ss.Rng.Float64()*0.4,
			SelectionPress:  1.0 + ss.Rng.Float64()*2.0,
			ExplorationRate: 0.1 + ss.Rng.Float64()*0.4,
			EliteRatio:      0.02 + ss.Rng.Float64()*0.08,
			FitnessHistory:  make([]float64, 0, 10),
		}
	}

	ss.MetaEvo = me
	logger.Info("META-EVO", "Initialisiert: %d Bots mit individuellen Evolutions-Parametern", n)
}

// ClearMetaEvolution disables the meta-evolution system.
func ClearMetaEvolution(ss *SwarmState) {
	ss.MetaEvo = nil
	ss.MetaEvoOn = false
}

// GetMetaMutationRate returns the mutation rate for a specific bot.
func GetMetaMutationRate(me *MetaEvoState, botIdx int) float64 {
	if me == nil || botIdx < 0 || botIdx >= len(me.Params) {
		return 0.1 // default
	}
	return me.Params[botIdx].MutationRate
}

// GetMetaMutationSize returns the mutation size for a specific bot.
func GetMetaMutationSize(me *MetaEvoState, botIdx int) float64 {
	if me == nil || botIdx < 0 || botIdx >= len(me.Params) {
		return 0.2 // default
	}
	return me.Params[botIdx].MutationSize
}

// GetMetaCrossoverRate returns the crossover rate for a specific bot.
func GetMetaCrossoverRate(me *MetaEvoState, botIdx int) float64 {
	if me == nil || botIdx < 0 || botIdx >= len(me.Params) {
		return 0.5 // default
	}
	return me.Params[botIdx].CrossoverRate
}

// RecordFitness records a bot's fitness for improvement tracking.
func RecordFitness(me *MetaEvoState, botIdx int, fitness float64) {
	if me == nil || botIdx < 0 || botIdx >= len(me.Params) {
		return
	}
	p := &me.Params[botIdx]
	p.FitnessHistory = append(p.FitnessHistory, fitness)
	if len(p.FitnessHistory) > 10 {
		p.FitnessHistory = p.FitnessHistory[1:]
	}

	// Compute improvement rate
	if len(p.FitnessHistory) >= 2 {
		recent := p.FitnessHistory[len(p.FitnessHistory)-1]
		older := p.FitnessHistory[0]
		if older > 0 {
			p.Improvement = (recent - older) / older
		}
	}
}

// EvolveMetaParams evolves the meta-parameters alongside the main evolution.
func EvolveMetaParams(ss *SwarmState, sortedIndices []int) {
	me := ss.MetaEvo
	if me == nil {
		return
	}

	n := len(ss.Bots)
	if len(me.Params) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 20 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 3
	if eliteCount > parentCount {
		eliteCount = parentCount
	}

	// Save parent meta-params
	parentParams := make([]MetaParams, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parentParams[i] = me.Params[sortedIndices[i]]
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			// Elite: keep params, record success
			continue
		}

		// Inherit from parent
		p := ss.Rng.Intn(parentCount)
		newP := parentParams[p]
		newP.FitnessHistory = make([]float64, 0, 10)

		// Meta-mutation: mutate the meta-params themselves
		metaMutRate := 0.15 // probability of meta-mutation
		if ss.Rng.Float64() < metaMutRate {
			newP.MutationRate *= 0.7 + ss.Rng.Float64()*0.6
			newP.MutationRate = clampF(newP.MutationRate, 0.01, 0.5)
		}
		if ss.Rng.Float64() < metaMutRate {
			newP.MutationSize *= 0.7 + ss.Rng.Float64()*0.6
			newP.MutationSize = clampF(newP.MutationSize, 0.05, 1.0)
		}
		if ss.Rng.Float64() < metaMutRate {
			newP.CrossoverRate *= 0.7 + ss.Rng.Float64()*0.6
			newP.CrossoverRate = clampF(newP.CrossoverRate, 0.1, 0.9)
		}
		if ss.Rng.Float64() < metaMutRate {
			newP.SelectionPress *= 0.7 + ss.Rng.Float64()*0.6
			newP.SelectionPress = clampF(newP.SelectionPress, 0.5, 5.0)
		}
		if ss.Rng.Float64() < metaMutRate {
			newP.ExplorationRate *= 0.7 + ss.Rng.Float64()*0.6
			newP.ExplorationRate = clampF(newP.ExplorationRate, 0.0, 1.0)
		}
		if ss.Rng.Float64() < metaMutRate {
			newP.EliteRatio *= 0.7 + ss.Rng.Float64()*0.6
			newP.EliteRatio = clampF(newP.EliteRatio, 0.01, 0.2)
		}

		// Self-adaptation: if parent was improving, bias toward its params
		if parentParams[p].Improvement > 0 {
			// Keep params closer to parent (less mutation)
			metaMutRate *= 0.5
		}

		me.Params[botIdx] = newP
	}

	// Update population statistics
	updateMetaStats(me)
	me.Generation++

	logger.Info("META-EVO", "Gen %d: MutRate=%.3f, CrossRate=%.3f, SelPress=%.2f, Diversity=%.3f",
		me.Generation, me.AvgMutationRate, me.AvgCrossoverRate,
		me.AvgSelectionPress, me.Diversity)
}

// updateMetaStats computes population-level statistics.
func updateMetaStats(me *MetaEvoState) {
	n := len(me.Params)
	if n == 0 {
		return
	}

	sumMut, sumCross, sumSel, sumExpl := 0.0, 0.0, 0.0, 0.0
	for _, p := range me.Params {
		sumMut += p.MutationRate
		sumCross += p.CrossoverRate
		sumSel += p.SelectionPress
		sumExpl += p.ExplorationRate
	}

	fn := float64(n)
	me.AvgMutationRate = sumMut / fn
	me.AvgCrossoverRate = sumCross / fn
	me.AvgSelectionPress = sumSel / fn
	me.AvgExplorationRate = sumExpl / fn

	// Diversity: coefficient of variation of mutation rates
	varMut := 0.0
	for _, p := range me.Params {
		d := p.MutationRate - me.AvgMutationRate
		varMut += d * d
	}
	if me.AvgMutationRate > 0 {
		me.Diversity = math.Sqrt(varMut/fn) / me.AvgMutationRate
	}
}

// MetaAvgMutationRate returns the population average mutation rate.
func MetaAvgMutationRate(me *MetaEvoState) float64 {
	if me == nil {
		return 0
	}
	return me.AvgMutationRate
}

// MetaDiversity returns the diversity of meta-parameters.
func MetaDiversity(me *MetaEvoState) float64 {
	if me == nil {
		return 0
	}
	return me.Diversity
}
