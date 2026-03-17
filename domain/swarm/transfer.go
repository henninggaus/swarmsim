package swarm

import (
	"encoding/json"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
)

// TransferGenome holds an exported genome that can be imported into a different scenario.
type TransferGenome struct {
	Type        string             `json:"type"`         // "neuro", "gp", "params"
	SourceScene string             `json:"source_scene"` // scenario it was trained in
	Generation  int                `json:"generation"`   // generation when exported
	Fitness     float64            `json:"fitness"`      // fitness at export time
	BotCount    int                `json:"bot_count"`    // how many bots in the export

	// Neuro weights (one per bot)
	NeuroWeights [][]float64        `json:"neuro_weights,omitempty"`

	// GP programs (one per bot)
	GPPrograms   []string           `json:"gp_programs,omitempty"`

	// Parameter values (one set per bot)
	ParamValues  [][26]float64      `json:"param_values,omitempty"`

	// Morphology (if enabled)
	Morphologies []Morphology       `json:"morphologies,omitempty"`
}

// TransferState manages transfer learning imports/exports.
type TransferState struct {
	LastExport   *TransferGenome
	LastImport   *TransferGenome
	ExportCount  int
	ImportCount  int

	// Performance comparison
	TransferFitness  []FitnessRecord // fitness after import
	BaselineFitness  []FitnessRecord // fitness of fresh population (control)
	ComparisonActive bool
}

// ExportNeuro exports all neural network weights from the current population.
func ExportNeuro(ss *SwarmState) *TransferGenome {
	tg := &TransferGenome{
		Type:        "neuro",
		SourceScene: ss.ProgramName,
		Generation:  ss.NeuroGeneration,
		Fitness:     ss.BestFitness,
		BotCount:    len(ss.Bots),
	}

	for _, bot := range ss.Bots {
		if bot.Brain != nil {
			weights := make([]float64, NeuroWeights)
			copy(weights, bot.Brain.Weights[:])
			tg.NeuroWeights = append(tg.NeuroWeights, weights)
		}
	}

	if ss.MorphEnabled {
		for _, bot := range ss.Bots {
			tg.Morphologies = append(tg.Morphologies, bot.Morph)
		}
	}

	logger.Info("TRANSFER", "Exported %d neuro brains from %s (Gen %d, Fitness %.0f)",
		len(tg.NeuroWeights), tg.SourceScene, tg.Generation, tg.Fitness)
	return tg
}

// ExportGP exports all GP programs from the current population.
func ExportGP(ss *SwarmState) *TransferGenome {
	tg := &TransferGenome{
		Type:        "gp",
		SourceScene: ss.ProgramName,
		Generation:  ss.GPGeneration,
		Fitness:     ss.BestFitness,
		BotCount:    len(ss.Bots),
	}

	for _, bot := range ss.Bots {
		if bot.OwnProgram != nil {
			tg.GPPrograms = append(tg.GPPrograms, swarmscript.ProgramToText(bot.OwnProgram))
		}
	}

	logger.Info("TRANSFER", "Exported %d GP programs from %s (Gen %d, Fitness %.0f)",
		len(tg.GPPrograms), tg.SourceScene, tg.Generation, tg.Fitness)
	return tg
}

// ExportParams exports parameter evolution values.
func ExportParams(ss *SwarmState) *TransferGenome {
	tg := &TransferGenome{
		Type:        "params",
		SourceScene: ss.ProgramName,
		Generation:  ss.Generation,
		Fitness:     ss.BestFitness,
		BotCount:    len(ss.Bots),
	}

	for _, bot := range ss.Bots {
		tg.ParamValues = append(tg.ParamValues, bot.ParamValues)
	}

	logger.Info("TRANSFER", "Exported %d param sets from %s", len(tg.ParamValues), tg.SourceScene)
	return tg
}

// ImportNeuro imports neural network weights into the current population.
// If population sizes differ, weights are distributed cyclically.
func ImportNeuro(ss *SwarmState, tg *TransferGenome) bool {
	if tg == nil || tg.Type != "neuro" || len(tg.NeuroWeights) == 0 {
		return false
	}

	for i := range ss.Bots {
		if ss.Bots[i].Brain == nil {
			ss.Bots[i].Brain = &NeuroBrain{}
		}
		srcIdx := i % len(tg.NeuroWeights)
		copy(ss.Bots[i].Brain.Weights[:], tg.NeuroWeights[srcIdx])
	}

	// Import morphologies if available
	if len(tg.Morphologies) > 0 && ss.MorphEnabled {
		for i := range ss.Bots {
			srcIdx := i % len(tg.Morphologies)
			ss.Bots[i].Morph = tg.Morphologies[srcIdx]
		}
	}

	logger.Info("TRANSFER", "Imported %d neuro brains from %s (Gen %d) → %d Bots",
		len(tg.NeuroWeights), tg.SourceScene, tg.Generation, len(ss.Bots))
	return true
}

// ImportGP imports GP programs into the current population.
func ImportGP(ss *SwarmState, tg *TransferGenome) bool {
	if tg == nil || tg.Type != "gp" || len(tg.GPPrograms) == 0 {
		return false
	}

	for i := range ss.Bots {
		srcIdx := i % len(tg.GPPrograms)
		prog, err := swarmscript.ParseSwarmScript(tg.GPPrograms[srcIdx])
		if err == nil {
			ss.Bots[i].OwnProgram = prog
		}
	}

	logger.Info("TRANSFER", "Imported %d GP programs from %s → %d Bots",
		len(tg.GPPrograms), tg.SourceScene, len(ss.Bots))
	return true
}

// ImportParams imports parameter values into the current population.
func ImportParams(ss *SwarmState, tg *TransferGenome) bool {
	if tg == nil || tg.Type != "params" || len(tg.ParamValues) == 0 {
		return false
	}

	for i := range ss.Bots {
		srcIdx := i % len(tg.ParamValues)
		ss.Bots[i].ParamValues = tg.ParamValues[srcIdx]
	}

	logger.Info("TRANSFER", "Imported %d param sets from %s → %d Bots",
		len(tg.ParamValues), tg.SourceScene, len(ss.Bots))
	return true
}

// SerializeTransfer converts a TransferGenome to JSON.
func SerializeTransfer(tg *TransferGenome) ([]byte, error) {
	return json.Marshal(tg)
}

// DeserializeTransfer parses JSON into a TransferGenome.
func DeserializeTransfer(data []byte) (*TransferGenome, error) {
	tg := &TransferGenome{}
	err := json.Unmarshal(data, tg)
	return tg, err
}

// InitTransfer sets up the transfer learning state.
func InitTransfer(ss *SwarmState) {
	ss.Transfer = &TransferState{}
}

// ExportBestNeuro exports only the top N% of neural networks.
func ExportBestNeuro(ss *SwarmState, topPercent int) *TransferGenome {
	if topPercent < 1 {
		topPercent = 1
	}
	if topPercent > 100 {
		topPercent = 100
	}

	n := len(ss.Bots)
	count := n * topPercent / 100
	if count < 1 {
		count = 1
	}

	// Find top bots by fitness
	type botFit struct {
		idx     int
		fitness float64
	}
	fits := make([]botFit, n)
	for i := range ss.Bots {
		fits[i] = botFit{i, EvaluateGPFitness(&ss.Bots[i])}
	}
	// Simple selection sort for top N (good enough for small N)
	for i := 0; i < count; i++ {
		best := i
		for j := i + 1; j < n; j++ {
			if fits[j].fitness > fits[best].fitness {
				best = j
			}
		}
		fits[i], fits[best] = fits[best], fits[i]
	}

	tg := &TransferGenome{
		Type:        "neuro",
		SourceScene: ss.ProgramName,
		Generation:  ss.NeuroGeneration,
		Fitness:     fits[0].fitness,
		BotCount:    count,
	}

	for i := 0; i < count; i++ {
		bot := &ss.Bots[fits[i].idx]
		if bot.Brain != nil {
			weights := make([]float64, NeuroWeights)
			copy(weights, bot.Brain.Weights[:])
			tg.NeuroWeights = append(tg.NeuroWeights, weights)
		}
		if ss.MorphEnabled {
			tg.Morphologies = append(tg.Morphologies, bot.Morph)
		}
	}

	logger.Info("TRANSFER", "Exported top %d%% (%d) neuro brains, Best Fitness: %.0f",
		topPercent, count, tg.Fitness)
	return tg
}

// TransferAdaptationScore measures how well transferred genomes adapt.
// Returns the ratio of current fitness to the original export fitness.
func TransferAdaptationScore(ts *TransferState, currentBest float64) float64 {
	if ts == nil || ts.LastImport == nil || ts.LastImport.Fitness <= 0 {
		return 0
	}
	return currentBest / ts.LastImport.Fitness
}
