package swarm

import (
	"math/rand"
	"testing"
)

func TestInitDiploid(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitDiploid(ss, 24)

	ds := ss.Diploid
	if ds == nil {
		t.Fatal("diploid should be initialized")
	}
	if ds.NumGenes != 24 {
		t.Fatalf("expected 24 genes, got %d", ds.NumGenes)
	}
	if len(ds.Genomes) != 10 {
		t.Fatalf("expected 10 genomes, got %d", len(ds.Genomes))
	}
	// Each genome should have expressed values
	for i, g := range ds.Genomes {
		if len(g.Expressed) != 24 {
			t.Fatalf("bot %d: expected 24 expressed values, got %d", i, len(g.Expressed))
		}
	}
}

func TestInitDiploidClamp(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitDiploid(ss, 3)
	if ss.Diploid.NumGenes != 8 {
		t.Fatal("should clamp to min 8")
	}
}

func TestClearDiploid(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.DiploidOn = true
	InitDiploid(ss, 16)
	ClearDiploid(ss)

	if ss.Diploid != nil {
		t.Fatal("should be nil")
	}
	if ss.DiploidOn {
		t.Fatal("should be false")
	}
}

func TestExpressGenome(t *testing.T) {
	g := &AdvDiploidGenome{
		ChromA: []Allele{
			{Value: 1.0, Dominant: true},
			{Value: 0.5, Dominant: false},
			{Value: 0.8, Dominant: true},
		},
		ChromB: []Allele{
			{Value: 0.2, Dominant: false},
			{Value: 0.9, Dominant: true},
			{Value: 0.6, Dominant: true},
		},
		Expressed: make([]float64, 3),
	}

	expressAdvGenome(g)

	// Gene 0: A dominant, B recessive → express A (1.0)
	if g.Expressed[0] != 1.0 {
		t.Fatalf("gene 0: expected 1.0, got %.2f", g.Expressed[0])
	}
	// Gene 1: A recessive, B dominant → express B (0.9)
	if g.Expressed[1] != 0.9 {
		t.Fatalf("gene 1: expected 0.9, got %.2f", g.Expressed[1])
	}
	// Gene 2: both dominant → average (0.7)
	if g.Expressed[2] != 0.7 {
		t.Fatalf("gene 2: expected 0.7, got %.2f", g.Expressed[2])
	}
}

func TestTickDiploid(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitDiploid(ss, 16)

	TickDiploid(ss)
	// Should not panic and bots should have modified speeds
}

func TestTickDiploidNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickDiploid(ss) // should not panic
}

func TestEvolveDiploid(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitDiploid(ss, 16)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveDiploid(ss, sorted)

	if ss.Diploid.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.Diploid.Generation)
	}
}

func TestMeiosis(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)

	parent := &AdvDiploidGenome{
		ChromA: []Allele{
			{Value: 1.0, Dominant: true},
			{Value: 2.0, Dominant: false},
			{Value: 3.0, Dominant: true},
		},
		ChromB: []Allele{
			{Value: -1.0, Dominant: false},
			{Value: -2.0, Dominant: true},
			{Value: -3.0, Dominant: false},
		},
	}

	gamete := advMeiosis(ss, parent, 0) // no mutation
	if len(gamete) != 3 {
		t.Fatalf("expected 3 alleles, got %d", len(gamete))
	}
}

func TestHeterozygosity(t *testing.T) {
	if Heterozygosity(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestDiploidDiversity(t *testing.T) {
	if DiploidDiversity(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestHeterozygoteAdvantage(t *testing.T) {
	if HeterozygoteAdvantage(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitDiploid(ss, 16)

	bonus := HeterozygoteAdvantage(ss.Diploid, 0)
	if bonus < 0 {
		t.Fatal("bonus should be >= 0")
	}
}

func TestCloneDiploidGenome(t *testing.T) {
	src := AdvDiploidGenome{
		ChromA:    []Allele{{Value: 1.0, Dominant: true}},
		ChromB:    []Allele{{Value: 2.0, Dominant: false}},
		Expressed: []float64{1.0},
	}

	dst := cloneAdvDiploidGenome(src)
	dst.ChromA[0].Value = 999

	if src.ChromA[0].Value == 999 {
		t.Fatal("clone should be independent")
	}
}
