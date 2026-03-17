package swarm

import (
	"math/rand"
	"testing"
)

func TestDefaultMorphology(t *testing.T) {
	m := DefaultMorphology()
	if m.BodySize != 1.0 || m.SpeedGene != 1.0 || m.SensorRange != 1.0 {
		t.Fatal("default morphology should be all 1.0")
	}
}

func TestRandomMorphology(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	m := RandomMorphology(rng)
	if m.BodySize < 0.8 || m.BodySize > 1.2 {
		t.Fatalf("random morphology out of range: %f", m.BodySize)
	}
}

func TestMutateMorphology(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	cfg := DefaultMorphologyConfig()
	cfg.MutationRate = 1.0 // always mutate
	m := DefaultMorphology()
	m2 := MutateMorphology(rng, m, cfg)
	// At least one gene should differ
	same := m2.BodySize == m.BodySize && m2.SpeedGene == m.SpeedGene &&
		m2.SensorRange == m.SensorRange && m2.EnergyPool == m.EnergyPool
	if same {
		t.Fatal("mutation should change at least one gene")
	}
	// All genes should be in range
	genes := MorphologyGenes(m2)
	for i, g := range genes {
		if g < cfg.MinGene || g > cfg.MaxGene {
			t.Fatalf("gene %d out of range: %f", i, g)
		}
	}
}

func TestCrossoverMorphology(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	a := Morphology{BodySize: 0.5, SpeedGene: 0.5, SensorRange: 0.5, EnergyPool: 0.5, CommRange: 0.5, CarryCost: 0.5}
	b := Morphology{BodySize: 2.0, SpeedGene: 2.0, SensorRange: 2.0, EnergyPool: 2.0, CommRange: 2.0, CarryCost: 2.0}
	c := CrossoverMorphology(rng, a, b)
	genes := MorphologyGenes(c)
	for _, g := range genes {
		if g != 0.5 && g != 2.0 {
			t.Fatalf("crossover should pick from parents, got %f", g)
		}
	}
}

func TestEffectiveSpeed(t *testing.T) {
	big := Morphology{BodySize: 2.0, SpeedGene: 1.0}
	small := Morphology{BodySize: 0.5, SpeedGene: 1.0}
	if EffectiveSpeed(big) >= EffectiveSpeed(small) {
		t.Fatal("big bots should be slower")
	}
}

func TestMorphologyFitnessCost(t *testing.T) {
	neutral := DefaultMorphology()
	if MorphologyFitnessCost(neutral) != 0 {
		t.Fatal("neutral morphology should have zero cost")
	}
	big := Morphology{BodySize: 2.0, SpeedGene: 2.0, SensorRange: 2.0, EnergyPool: 2.0, CommRange: 2.0, CarryCost: 2.0}
	if MorphologyFitnessCost(big) <= 0 {
		t.Fatal("oversized morphology should have positive cost")
	}
}

func TestMorphologyDistance(t *testing.T) {
	a := DefaultMorphology()
	b := DefaultMorphology()
	if MorphologyDistance(a, b) != 0 {
		t.Fatal("same morphologies should have zero distance")
	}
	b.BodySize = 2.0
	if MorphologyDistance(a, b) <= 0 {
		t.Fatal("different morphologies should have positive distance")
	}
}

func TestInitMorphology(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitMorphology(ss)
	for _, bot := range ss.Bots {
		if bot.Morph.BodySize == 0 {
			t.Fatal("morphology should be initialized")
		}
	}
	if ss.MorphConfig == nil {
		t.Fatal("morph config should be set")
	}
}
