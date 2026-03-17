package swarm

import (
	"math"
	"math/rand"
	"testing"
)

// === genomeDistance tests ===

func TestGenomeDistanceIdentical(t *testing.T) {
	var a, b [26]float64
	for i := range a {
		a[i] = float64(i * 10)
		b[i] = float64(i * 10)
	}
	var used [26]bool
	for i := 0; i < 7; i++ {
		used[i] = true
	}
	d := genomeDistance(a, b, used)
	if d != 0 {
		t.Errorf("identical genomes should have distance 0, got %.6f", d)
	}
}

func TestGenomeDistanceSymmetric(t *testing.T) {
	var a, b [26]float64
	a[0] = 10
	b[0] = 20
	a[1] = 50
	b[1] = 30
	var used [26]bool
	used[0] = true
	used[1] = true

	d1 := genomeDistance(a, b, used)
	d2 := genomeDistance(b, a, used)
	if math.Abs(d1-d2) > 1e-10 {
		t.Errorf("distance should be symmetric: %.6f != %.6f", d1, d2)
	}
}

func TestGenomeDistanceNoUsedParams(t *testing.T) {
	var a, b [26]float64
	a[0] = 100
	b[0] = 0
	var used [26]bool // all false
	d := genomeDistance(a, b, used)
	if d != 0 {
		t.Errorf("no used params should return distance 0, got %.6f", d)
	}
}

func TestGenomeDistanceOnlyUsedParamsCount(t *testing.T) {
	var a, b [26]float64
	a[0] = 100
	b[0] = 0
	a[5] = 999 // unused, should be ignored
	b[5] = 0
	var used [26]bool
	used[0] = true

	d := genomeDistance(a, b, used)
	// sqrt((100-0)^2 / 1) / 100 = 100/100 = 1.0
	if math.Abs(d-1.0) > 0.001 {
		t.Errorf("expected distance 1.0, got %.6f", d)
	}
}

// === InitSpeciation tests ===

func TestInitSpeciation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	if ss.Speciation == nil {
		t.Fatal("Speciation should not be nil after init")
	}
	if ss.Speciation.Threshold != 0.3 {
		t.Errorf("expected initial threshold 0.3, got %.2f", ss.Speciation.Threshold)
	}
	if ss.Speciation.TargetSpecies != 7 {
		t.Errorf("expected target species 7, got %d", ss.Speciation.TargetSpecies)
	}
}

// === UpdateSpeciation tests ===

func TestUpdateSpeciationBasic(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSpeciation(ss)

	// Give bots distinct genomes
	for i := range ss.Bots {
		ss.UsedParams[0] = true
		ss.UsedParams[1] = true
		ss.Bots[i].ParamValues[0] = float64(i) * 50
		ss.Bots[i].ParamValues[1] = float64(i) * 30
		ss.Bots[i].Fitness = float64(i)
	}

	UpdateSpeciation(ss)

	// Should have created some species
	if len(ss.Speciation.Species) == 0 {
		t.Error("should have at least one species")
	}

	// All bots should be assigned to exactly one species
	assigned := make(map[int]bool)
	for _, sp := range ss.Speciation.Species {
		for _, bi := range sp.Members {
			if assigned[bi] {
				t.Errorf("bot %d assigned to multiple species", bi)
			}
			assigned[bi] = true
		}
	}
	for i := 0; i < 20; i++ {
		if !assigned[i] {
			t.Errorf("bot %d not assigned to any species", i)
		}
	}
}

func TestUpdateSpeciationIdenticalBots(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	// All bots have identical genomes → should all be in one species
	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = 50
	}

	UpdateSpeciation(ss)

	if len(ss.Speciation.Species) != 1 {
		t.Errorf("identical bots should form 1 species, got %d", len(ss.Speciation.Species))
	}
	if len(ss.Speciation.Species[0].Members) != 10 {
		t.Errorf("all 10 bots should be in the species, got %d", len(ss.Speciation.Species[0].Members))
	}
}

func TestUpdateSpeciationThresholdAdjustment(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSpeciation(ss)
	ss.Speciation.TargetSpecies = 3

	// Spread bots far apart → creates many species
	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = float64(i) * 200
	}

	initialThreshold := ss.Speciation.Threshold

	// Run several rounds
	for round := 0; round < 10; round++ {
		UpdateSpeciation(ss)
	}

	// If too many species formed, threshold should have increased
	if len(ss.Speciation.Species) > ss.Speciation.TargetSpecies+1 {
		if ss.Speciation.Threshold <= initialThreshold {
			t.Error("threshold should increase when too many species")
		}
	}
}

func TestUpdateSpeciationThresholdBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	// Force extreme threshold values
	ss.Speciation.Threshold = 0.01

	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = float64(i)
	}

	UpdateSpeciation(ss)

	if ss.Speciation.Threshold < 0.05 {
		t.Errorf("threshold should be clamped to >= 0.05, got %.4f", ss.Speciation.Threshold)
	}
}

func TestUpdateSpeciationStagnation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = 50
		ss.Bots[i].Fitness = 10
	}

	// Run multiple generations with no fitness improvement
	for gen := 0; gen < 5; gen++ {
		UpdateSpeciation(ss)
	}

	for _, sp := range ss.Speciation.Species {
		if sp.Stagnant < 3 {
			t.Errorf("species should be stagnant after 5 gens without improvement, stagnant=%d", sp.Stagnant)
		}
	}
}

func TestUpdateSpeciationEmptyPopulation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 0)
	InitSpeciation(ss)

	// Should not panic
	UpdateSpeciation(ss)
}

func TestUpdateSpeciationHistoryRecorded(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = float64(i) * 100
	}

	for gen := 0; gen < 5; gen++ {
		ss.Generation = gen
		UpdateSpeciation(ss)
	}

	if len(ss.Speciation.History) != 5 {
		t.Errorf("expected 5 history snapshots, got %d", len(ss.Speciation.History))
	}
}

func TestUpdateSpeciationHistoryCap(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = float64(i) * 100
	}

	for gen := 0; gen < 150; gen++ {
		ss.Generation = gen
		UpdateSpeciation(ss)
	}

	if len(ss.Speciation.History) > 100 {
		t.Errorf("history should be capped at 100, got %d", len(ss.Speciation.History))
	}
}

func TestUpdateSpeciationLEDColors(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = float64(i) * 100
	}

	UpdateSpeciation(ss)

	// Bots in the same species should have the same LED color
	for _, sp := range ss.Speciation.Species {
		if len(sp.Members) < 2 {
			continue
		}
		color := ss.Bots[sp.Members[0]].LEDColor
		for _, bi := range sp.Members[1:] {
			if ss.Bots[bi].LEDColor != color {
				t.Errorf("bots in same species should have same LED color")
				break
			}
		}
	}
}

func TestSpeciesActiveCount(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)

	if SpeciesActiveCount(ss) != 0 {
		t.Error("should be 0 before init")
	}

	InitSpeciation(ss)
	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = float64(i) * 100
	}
	UpdateSpeciation(ss)

	count := SpeciesActiveCount(ss)
	if count == 0 {
		t.Error("should have species after update")
	}
}

func TestUpdateSpeciationNilSpeciation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	// Should not panic when speciation is nil
	UpdateSpeciation(ss)
}

// === Fitness Sharing tests ===

func TestApplyFitnessSharingReducesFitness(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	// All bots identical → 1 big species
	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = 50
		ss.Bots[i].Fitness = 100
	}

	// Assign to species first
	UpdateSpeciation(ss)

	// After fitness sharing, fitness should be 100/10 = 10
	for i, bot := range ss.Bots {
		if bot.Fitness > 15 {
			t.Errorf("bot %d: fitness should be shared (reduced), got %.1f", i, bot.Fitness)
		}
	}
}

func TestApplyFitnessSharingProtectsSmallSpecies(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSpeciation(ss)

	// Create 2 groups: 8 bots in cluster A, 2 bots far away in cluster B
	ss.UsedParams[0] = true
	for i := 0; i < 8; i++ {
		ss.Bots[i].ParamValues[0] = 50
		ss.Bots[i].Fitness = 100
	}
	for i := 8; i < 10; i++ {
		ss.Bots[i].ParamValues[0] = 5000 // far away
		ss.Bots[i].Fitness = 100
	}

	UpdateSpeciation(ss)

	// Large species (8 members): fitness ≈ 100/8 = 12.5
	// Small species (2 members): fitness ≈ 100/2 = 50
	// Small species should have HIGHER shared fitness → protected!
	var largeFitness, smallFitness float64
	for i := 0; i < 8; i++ {
		largeFitness += ss.Bots[i].Fitness
	}
	largeFitness /= 8
	for i := 8; i < 10; i++ {
		smallFitness += ss.Bots[i].Fitness
	}
	smallFitness /= 2

	if smallFitness <= largeFitness {
		t.Errorf("small species should have higher shared fitness: small=%.1f, large=%.1f",
			smallFitness, largeFitness)
	}
}

func TestApplyFitnessSharingNilSpeciation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	// Should not panic
	ApplyFitnessSharing(ss)
}

func TestApplyFitnessSharingSingleMemberSpecies(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 3)
	InitSpeciation(ss)

	// Each bot very far apart → each in own species
	ss.UsedParams[0] = true
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = float64(i) * 10000
		ss.Bots[i].Fitness = 100
	}

	UpdateSpeciation(ss)

	// Single-member species: fitness should NOT be divided (stays 100)
	for i, bot := range ss.Bots {
		if bot.Fitness < 90 {
			t.Errorf("bot %d in solo species: fitness should stay ~100, got %.1f", i, bot.Fitness)
		}
	}
}
