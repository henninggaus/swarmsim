package genetics

import (
	"math/rand"
	"testing"
)

// --- DefaultGenome tests ---

func TestDefaultGenome(t *testing.T) {
	g := DefaultGenome()
	if g.FlockingWeight != 0.5 {
		t.Errorf("FlockingWeight: want 0.5, got %v", g.FlockingWeight)
	}
	if g.PheromoneFollow != 0.5 {
		t.Errorf("PheromoneFollow: want 0.5, got %v", g.PheromoneFollow)
	}
	if g.ExplorationDrive != 0.5 {
		t.Errorf("ExplorationDrive: want 0.5, got %v", g.ExplorationDrive)
	}
	if g.CommFrequency != 0.5 {
		t.Errorf("CommFrequency: want 0.5, got %v", g.CommFrequency)
	}
	if g.EnergyConservation != 0.5 {
		t.Errorf("EnergyConservation: want 0.5, got %v", g.EnergyConservation)
	}
	if g.SpeedPreference != 1.0 {
		t.Errorf("SpeedPreference: want 1.0, got %v", g.SpeedPreference)
	}
	if g.CooperationBias != 0.5 {
		t.Errorf("CooperationBias: want 0.5, got %v", g.CooperationBias)
	}
}

// --- GenomeLabels ---

func TestGenomeLabels(t *testing.T) {
	labels := GenomeLabels()
	if len(labels) != 7 {
		t.Fatalf("expected 7 labels, got %d", len(labels))
	}
	expected := [7]string{"Flock", "Pher", "Explor", "Comm", "EnCons", "Speed", "Coop"}
	if labels != expected {
		t.Errorf("labels mismatch: got %v", labels)
	}
}

// --- Values ---

func TestGenomeValues(t *testing.T) {
	g := DefaultGenome()
	v := g.Values()
	if len(v) != 7 {
		t.Fatalf("expected 7 values, got %d", len(v))
	}
	// SpeedPreference is normalized: (1.0 - 0.5) = 0.5
	if v[5] != 0.5 {
		t.Errorf("SpeedPreference normalized value: want 0.5, got %v", v[5])
	}
	// Other values should be 0.5
	for i := 0; i < 5; i++ {
		if v[i] != 0.5 {
			t.Errorf("value[%d]: want 0.5, got %v", i, v[i])
		}
	}
}

// --- Crossover tests ---

func TestCrossover(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	a := Genome{
		FlockingWeight: 0.0, PheromoneFollow: 0.0, ExplorationDrive: 0.0,
		CommFrequency: 0.0, EnergyConservation: 0.0,
		SpeedPreference: 0.5, CooperationBias: 0.0,
	}
	b := Genome{
		FlockingWeight: 1.0, PheromoneFollow: 1.0, ExplorationDrive: 1.0,
		CommFrequency: 1.0, EnergyConservation: 1.0,
		SpeedPreference: 1.5, CooperationBias: 1.0,
	}

	child := Crossover(a, b, rng)

	// Each gene should come from either parent a (0.0/0.5) or parent b (1.0/1.5)
	checkGene := func(name string, val, pa, pb float64) {
		if val != pa && val != pb {
			t.Errorf("%s: got %v, expected either %v or %v", name, val, pa, pb)
		}
	}
	checkGene("FlockingWeight", child.FlockingWeight, 0.0, 1.0)
	checkGene("PheromoneFollow", child.PheromoneFollow, 0.0, 1.0)
	checkGene("ExplorationDrive", child.ExplorationDrive, 0.0, 1.0)
	checkGene("CommFrequency", child.CommFrequency, 0.0, 1.0)
	checkGene("EnergyConservation", child.EnergyConservation, 0.0, 1.0)
	checkGene("SpeedPreference", child.SpeedPreference, 0.5, 1.5)
	checkGene("CooperationBias", child.CooperationBias, 0.0, 1.0)
}

func TestCrossoverMixes(t *testing.T) {
	// Run many crossovers to ensure we see genes from both parents
	a := Genome{FlockingWeight: 0.0}
	b := Genome{FlockingWeight: 1.0}

	sawA, sawB := false, false
	for seed := int64(0); seed < 100; seed++ {
		rng := rand.New(rand.NewSource(seed))
		child := Crossover(a, b, rng)
		if child.FlockingWeight == 0.0 {
			sawA = true
		}
		if child.FlockingWeight == 1.0 {
			sawB = true
		}
	}
	if !sawA || !sawB {
		t.Error("crossover should produce genes from both parents over many trials")
	}
}

// --- Mutate tests ---

func TestMutate(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := DefaultGenome()

	// High mutation rate and sigma to guarantee change
	Mutate(&g, rng, 1.0, 0.5)

	// At least one gene should have changed from default
	d := DefaultGenome()
	unchanged := 0
	if g.FlockingWeight == d.FlockingWeight {
		unchanged++
	}
	if g.PheromoneFollow == d.PheromoneFollow {
		unchanged++
	}
	if g.ExplorationDrive == d.ExplorationDrive {
		unchanged++
	}
	if g.CommFrequency == d.CommFrequency {
		unchanged++
	}
	if g.EnergyConservation == d.EnergyConservation {
		unchanged++
	}
	if g.SpeedPreference == d.SpeedPreference {
		unchanged++
	}
	if g.CooperationBias == d.CooperationBias {
		unchanged++
	}
	if unchanged == 7 {
		t.Error("expected at least one gene to mutate with rate=1.0")
	}
}

func TestMutateClamping(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := Genome{
		FlockingWeight:     0.0,
		PheromoneFollow:    1.0,
		ExplorationDrive:   0.0,
		CommFrequency:      1.0,
		EnergyConservation: 0.0,
		SpeedPreference:    0.5,
		CooperationBias:    1.0,
	}

	// Mutate many times to test clamping
	for i := 0; i < 100; i++ {
		Mutate(&g, rng, 1.0, 1.0)
	}

	// All 0-1 fields should be within [0, 1]
	check := func(name string, val, lo, hi float64) {
		if val < lo || val > hi {
			t.Errorf("%s = %v, outside [%v, %v]", name, val, lo, hi)
		}
	}
	check("FlockingWeight", g.FlockingWeight, 0, 1)
	check("PheromoneFollow", g.PheromoneFollow, 0, 1)
	check("ExplorationDrive", g.ExplorationDrive, 0, 1)
	check("CommFrequency", g.CommFrequency, 0, 1)
	check("EnergyConservation", g.EnergyConservation, 0, 1)
	check("SpeedPreference", g.SpeedPreference, 0.5, 1.5)
	check("CooperationBias", g.CooperationBias, 0, 1)
}

func TestMutateZeroRate(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := DefaultGenome()
	original := g

	Mutate(&g, rng, 0.0, 0.5) // rate=0 means no mutation

	if g != original {
		t.Error("mutation with rate=0 should not change genome")
	}
}

// --- NewRandomGenome tests ---

func TestNewRandomGenome(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := NewRandomGenome(rng)

	// Values should be within valid ranges
	if g.FlockingWeight < 0 || g.FlockingWeight > 1 {
		t.Errorf("FlockingWeight %v out of [0,1]", g.FlockingWeight)
	}
	if g.SpeedPreference < 0.5 || g.SpeedPreference > 1.5 {
		t.Errorf("SpeedPreference %v out of [0.5,1.5]", g.SpeedPreference)
	}
	if g.CooperationBias < 0 || g.CooperationBias > 1 {
		t.Errorf("CooperationBias %v out of [0,1]", g.CooperationBias)
	}
}

// --- ClampRange tests ---

func TestClampRange(t *testing.T) {
	if ClampRange(0.5, 0, 1) != 0.5 {
		t.Error("value within range should be unchanged")
	}
	if ClampRange(-1, 0, 1) != 0 {
		t.Error("below lo should clamp to lo")
	}
	if ClampRange(2, 0, 1) != 1 {
		t.Error("above hi should clamp to hi")
	}
	if ClampRange(0.3, 0.5, 1.5) != 0.5 {
		t.Error("below lo should clamp to lo")
	}
}
