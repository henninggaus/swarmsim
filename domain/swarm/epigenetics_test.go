package swarm

import (
	"math/rand"
	"testing"
)

func TestInitEpigenetics(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitEpigenetics(ss)

	ep := ss.Epigenetics
	if ep == nil {
		t.Fatal("epigenetics should be initialized")
	}
	if len(ep.Marks) != 15 {
		t.Fatalf("expected 15 marks, got %d", len(ep.Marks))
	}
}

func TestClearEpigenetics(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.EpigeneticsOn = true
	InitEpigenetics(ss)
	ClearEpigenetics(ss)

	if ss.Epigenetics != nil {
		t.Fatal("should be nil")
	}
	if ss.EpigeneticsOn {
		t.Fatal("should be false")
	}
}

func TestTickEpigenetics(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitEpigenetics(ss)

	// Create starvation conditions
	for i := range ss.Bots {
		ss.Bots[i].NearestPickupDist = 500
		ss.Bots[i].Speed = SwarmBotSpeed
	}

	for tick := 0; tick < 200; tick++ {
		TickEpigenetics(ss)
	}

	ep := ss.Epigenetics
	// Starvation should have caused methylation of speed gene
	totalMeth := 0.0
	for _, m := range ep.Marks {
		totalMeth += m.Methylation[EpiGeneSpeed]
	}
	if totalMeth == 0 {
		t.Fatal("starvation should cause speed gene methylation")
	}
}

func TestTickEpigeneticsNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickEpigenetics(ss) // should not panic
}

func TestEvolveEpigenetics(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitEpigenetics(ss)

	// Set some marks on parent
	ss.Epigenetics.Marks[0].Methylation[EpiGeneSpeed] = 0.8
	ss.Epigenetics.Marks[0].Acetylation[EpiGeneEfficiency] = 0.7

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveEpigenetics(ss, sorted)
	if ss.Epigenetics.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.Epigenetics.Generation)
	}

	// Children should have inherited some marks
	child := &ss.Epigenetics.Marks[5] // non-elite child
	if child.StarvationExp != 0 {
		t.Fatal("child experience should be reset")
	}
}

func TestEpiGeneName(t *testing.T) {
	if EpiGeneName(EpiGeneSpeed) != "Geschwindigkeit" {
		t.Fatal("expected Geschwindigkeit")
	}
	if EpiGeneName(EpiGeneEfficiency) != "Effizienz" {
		t.Fatal("expected Effizienz")
	}
	if EpiGeneName(99) != "?" {
		t.Fatal("expected ?")
	}
}

func TestEpiAvgMethylation(t *testing.T) {
	if EpiAvgMethylation(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestEpiAvgAcetylation(t *testing.T) {
	if EpiAvgAcetylation(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestEpiDiversityFunc(t *testing.T) {
	if EpiDiversity(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBotMethylation(t *testing.T) {
	if BotMethylation(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitEpigenetics(ss)
	ss.Epigenetics.Marks[0].Methylation = [8]float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}

	m := BotMethylation(ss.Epigenetics, 0)
	if m != 0.5 {
		t.Fatalf("expected 0.5, got %.2f", m)
	}
}

func TestExperienceAccumulation(t *testing.T) {
	marks := &BotEpiMarks{}
	bot := &SwarmBot{
		NearestPickupDist: 500,
		CarryingPkg:       -1,
		Speed:             SwarmBotSpeed,
		NeighborCount:     1,
	}

	for i := 0; i < 100; i++ {
		accumulateEpiExperience(bot, marks)
	}

	if marks.StarvationExp < 0.3 {
		t.Fatal("should have accumulated starvation experience")
	}
}

func TestComputeEpiDiversity(t *testing.T) {
	ep := &EpigeneticsState{
		Marks: []BotEpiMarks{
			{Methylation: [8]float64{0, 0, 0, 0, 0, 0, 0, 0}},
			{Methylation: [8]float64{1, 1, 1, 1, 1, 1, 1, 1}},
		},
	}

	computeEpiDiversity(ep)
	if ep.EpiDiversity <= 0 {
		t.Fatal("very different marks should produce diversity > 0")
	}
}
