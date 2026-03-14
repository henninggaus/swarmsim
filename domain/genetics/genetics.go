package genetics

import (
	"math"
	"math/rand"
)

// DefaultGenome returns sensible starting values.
func DefaultGenome() Genome {
	return Genome{
		FlockingWeight:     0.5,
		PheromoneFollow:    0.5,
		ExplorationDrive:   0.5,
		CommFrequency:      0.5,
		EnergyConservation: 0.5,
		SpeedPreference:    1.0,
		CooperationBias:    0.5,
	}
}

// NewRandomGenome creates a genome with values sampled around defaults.
func NewRandomGenome(rng *rand.Rand) Genome {
	d := DefaultGenome()
	return Genome{
		FlockingWeight:     clamp01(d.FlockingWeight + rng.NormFloat64()*0.2),
		PheromoneFollow:    clamp01(d.PheromoneFollow + rng.NormFloat64()*0.2),
		ExplorationDrive:   clamp01(d.ExplorationDrive + rng.NormFloat64()*0.2),
		CommFrequency:      clamp01(d.CommFrequency + rng.NormFloat64()*0.2),
		EnergyConservation: clamp01(d.EnergyConservation + rng.NormFloat64()*0.2),
		SpeedPreference:    ClampRange(d.SpeedPreference+rng.NormFloat64()*0.2, 0.5, 1.5),
		CooperationBias:    clamp01(d.CooperationBias + rng.NormFloat64()*0.2),
	}
}

// Crossover produces a child genome by mixing two parents.
func Crossover(a, b Genome, rng *rand.Rand) Genome {
	pick := func(va, vb float64) float64 {
		if rng.Float64() < 0.5 {
			return va
		}
		return vb
	}
	return Genome{
		FlockingWeight:     pick(a.FlockingWeight, b.FlockingWeight),
		PheromoneFollow:    pick(a.PheromoneFollow, b.PheromoneFollow),
		ExplorationDrive:   pick(a.ExplorationDrive, b.ExplorationDrive),
		CommFrequency:      pick(a.CommFrequency, b.CommFrequency),
		EnergyConservation: pick(a.EnergyConservation, b.EnergyConservation),
		SpeedPreference:    pick(a.SpeedPreference, b.SpeedPreference),
		CooperationBias:    pick(a.CooperationBias, b.CooperationBias),
	}
}

// Mutate applies random perturbation to a genome.
func Mutate(g *Genome, rng *rand.Rand, rate, sigma float64) {
	mutF := func(v float64) float64 {
		if rng.Float64() < rate {
			return v + rng.NormFloat64()*sigma
		}
		return v
	}
	g.FlockingWeight = clamp01(mutF(g.FlockingWeight))
	g.PheromoneFollow = clamp01(mutF(g.PheromoneFollow))
	g.ExplorationDrive = clamp01(mutF(g.ExplorationDrive))
	g.CommFrequency = clamp01(mutF(g.CommFrequency))
	g.EnergyConservation = clamp01(mutF(g.EnergyConservation))
	g.SpeedPreference = ClampRange(mutF(g.SpeedPreference), 0.5, 1.5)
	g.CooperationBias = clamp01(mutF(g.CooperationBias))
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

// ClampRange clamps v to [lo, hi]. Exported for use by evolution.
func ClampRange(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
