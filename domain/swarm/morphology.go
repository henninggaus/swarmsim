package swarm

import (
	"math"
	"math/rand"
)

// Morphology holds evolvable body parameters for a bot.
// These affect physical capabilities and create trade-offs that
// drive specialization — like in real evolution.
type Morphology struct {
	BodySize     float64 // 0.5-2.0 multiplier (affects radius, carry capacity)
	SpeedGene    float64 // 0.5-2.0 multiplier (base speed modifier)
	SensorRange  float64 // 0.5-2.0 multiplier (perception range)
	EnergyPool   float64 // 0.5-2.0 multiplier (max energy capacity)
	CommRange    float64 // 0.5-2.0 multiplier (communication range)
	CarryCost    float64 // 0.5-2.0 multiplier (energy cost for carrying)
}

// MorphologyConfig controls the morphological evolution system.
type MorphologyConfig struct {
	MutationRate     float64 // probability of mutating each gene (default 0.15)
	MutationStrength float64 // gaussian noise sigma (default 0.1)
	MinGene          float64 // minimum gene value (default 0.5)
	MaxGene          float64 // maximum gene value (default 2.0)
}

// DefaultMorphologyConfig returns sensible defaults.
func DefaultMorphologyConfig() MorphologyConfig {
	return MorphologyConfig{
		MutationRate:     0.15,
		MutationStrength: 0.1,
		MinGene:          0.5,
		MaxGene:          2.0,
	}
}

// DefaultMorphology returns a neutral morphology (all 1.0).
func DefaultMorphology() Morphology {
	return Morphology{
		BodySize:    1.0,
		SpeedGene:   1.0,
		SensorRange: 1.0,
		EnergyPool:  1.0,
		CommRange:   1.0,
		CarryCost:   1.0,
	}
}

// RandomMorphology creates a slightly randomized morphology around 1.0.
func RandomMorphology(rng *rand.Rand) Morphology {
	jitter := func() float64 {
		return 0.8 + rng.Float64()*0.4 // 0.8-1.2
	}
	return Morphology{
		BodySize:    jitter(),
		SpeedGene:   jitter(),
		SensorRange: jitter(),
		EnergyPool:  jitter(),
		CommRange:   jitter(),
		CarryCost:   jitter(),
	}
}

// clampGene clamps a value to [min, max].
func clampGene(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// MutateMorphology applies gaussian noise to morphology genes.
func MutateMorphology(rng *rand.Rand, m Morphology, cfg MorphologyConfig) Morphology {
	mutate := func(v float64) float64 {
		if rng.Float64() < cfg.MutationRate {
			v += rng.NormFloat64() * cfg.MutationStrength
			v = clampGene(v, cfg.MinGene, cfg.MaxGene)
		}
		return v
	}
	return Morphology{
		BodySize:    mutate(m.BodySize),
		SpeedGene:   mutate(m.SpeedGene),
		SensorRange: mutate(m.SensorRange),
		EnergyPool:  mutate(m.EnergyPool),
		CommRange:   mutate(m.CommRange),
		CarryCost:   mutate(m.CarryCost),
	}
}

// CrossoverMorphology creates a child morphology from two parents (uniform crossover).
func CrossoverMorphology(rng *rand.Rand, a, b Morphology) Morphology {
	pick := func(va, vb float64) float64 {
		if rng.Float64() < 0.5 {
			return va
		}
		return vb
	}
	return Morphology{
		BodySize:    pick(a.BodySize, b.BodySize),
		SpeedGene:   pick(a.SpeedGene, b.SpeedGene),
		SensorRange: pick(a.SensorRange, b.SensorRange),
		EnergyPool:  pick(a.EnergyPool, b.EnergyPool),
		CommRange:   pick(a.CommRange, b.CommRange),
		CarryCost:   pick(a.CarryCost, b.CarryCost),
	}
}

// EffectiveSpeed returns the bot's actual speed based on morphology.
// Trade-off: larger bots are slower, but speed gene compensates.
func EffectiveSpeed(m Morphology) float64 {
	// Bigger body = slower (inverse square root), speed gene directly multiplies
	return SwarmBotSpeed * m.SpeedGene / math.Sqrt(m.BodySize)
}

// EffectiveRadius returns the bot's collision radius based on body size.
func EffectiveRadius(m Morphology) float64 {
	return SwarmBotRadius * math.Sqrt(m.BodySize)
}

// EffectiveSensorRange returns the bot's perception range.
func EffectiveSensorRange(m Morphology) float64 {
	return SwarmSensorRange * m.SensorRange
}

// EffectiveCommRange returns the bot's communication range.
func EffectiveCommRange(m Morphology) float64 {
	return SwarmCommRange * m.CommRange
}

// EffectiveMaxEnergy returns the bot's max energy capacity.
func EffectiveMaxEnergy(m Morphology) float64 {
	return 100.0 * m.EnergyPool
}

// CarrySpeedPenalty returns the speed multiplier when carrying (larger bots carry easier).
func CarrySpeedPenalty(m Morphology) float64 {
	// Default carry penalty is 0.7 (30% slowdown).
	// Large bots get less penalty, small bots get more.
	base := 0.7
	bonus := (m.BodySize - 1.0) * 0.15 // ±15% per unit of body size
	penalty := base + bonus
	if penalty > 0.95 {
		penalty = 0.95
	}
	if penalty < 0.4 {
		penalty = 0.4
	}
	return penalty
}

// MorphologyFitnessCost returns the metabolic cost of having a large body.
// This creates a natural pressure against growing infinitely large.
func MorphologyFitnessCost(m Morphology) float64 {
	// Total "mass" of all genes — neutral at 6.0 (6 genes × 1.0)
	total := m.BodySize + m.SpeedGene + m.SensorRange + m.EnergyPool + m.CommRange + m.CarryCost
	excess := total - 6.0
	if excess <= 0 {
		return 0
	}
	return excess * excess * 2.0 // quadratic cost for excess
}

// MorphologyDistance computes euclidean distance between two morphologies (for speciation).
func MorphologyDistance(a, b Morphology) float64 {
	d := func(x, y float64) float64 { return (x - y) * (x - y) }
	return math.Sqrt(
		d(a.BodySize, b.BodySize) +
			d(a.SpeedGene, b.SpeedGene) +
			d(a.SensorRange, b.SensorRange) +
			d(a.EnergyPool, b.EnergyPool) +
			d(a.CommRange, b.CommRange) +
			d(a.CarryCost, b.CarryCost),
	)
}

// MorphologyGenes returns the morphology as a slice for visualization.
func MorphologyGenes(m Morphology) []float64 {
	return []float64{m.BodySize, m.SpeedGene, m.SensorRange, m.EnergyPool, m.CommRange, m.CarryCost}
}

// MorphologyGeneNames returns display names for each gene.
func MorphologyGeneNames() []string {
	return []string{"Koerper", "Speed", "Sensor", "Energie", "Komm", "Trage-Effi"}
}

// InitMorphology assigns random morphologies to all bots.
func InitMorphology(ss *SwarmState) {
	for i := range ss.Bots {
		ss.Bots[i].Morph = RandomMorphology(ss.Rng)
	}
	if ss.MorphConfig == nil {
		cfg := DefaultMorphologyConfig()
		ss.MorphConfig = &cfg
	}
}

// ClearMorphology resets all morphologies to default.
func ClearMorphology(ss *SwarmState) {
	for i := range ss.Bots {
		ss.Bots[i].Morph = DefaultMorphology()
	}
	ss.MorphEnabled = false
	ss.MorphConfig = nil
}

// EvolveMorphology performs morphological crossover and mutation after a generation.
// Should be called alongside RunGPEvolution or RunNeuroEvolution.
func EvolveMorphology(ss *SwarmState, sortedIndices []int) {
	if !ss.MorphEnabled || ss.MorphConfig == nil {
		return
	}
	cfg := *ss.MorphConfig
	n := len(ss.Bots)
	if n < 4 {
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

	// Save parent morphologies
	parentMorphs := make([]Morphology, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parentMorphs[i] = ss.Bots[sortedIndices[i]].Morph
	}

	// Save elite morphologies (no mutation)
	eliteMorphs := make([]Morphology, eliteCount)
	for i := 0; i < eliteCount && i < len(sortedIndices); i++ {
		eliteMorphs[i] = ss.Bots[sortedIndices[i]].Morph
	}

	freshCount := n * 10 / 100
	if freshCount < 1 {
		freshCount = 1
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			ss.Bots[botIdx].Morph = eliteMorphs[rank]
		} else if rank >= n-freshCount {
			ss.Bots[botIdx].Morph = RandomMorphology(ss.Rng)
		} else {
			p1 := ss.Rng.Intn(parentCount)
			p2 := ss.Rng.Intn(parentCount)
			child := CrossoverMorphology(ss.Rng, parentMorphs[p1], parentMorphs[p2])
			child = MutateMorphology(ss.Rng, child, cfg)
			ss.Bots[botIdx].Morph = child
		}
	}
}
