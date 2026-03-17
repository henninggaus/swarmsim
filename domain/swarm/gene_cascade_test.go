package swarm

import (
	"math/rand"
	"testing"
)

func TestInitGeneCascade(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitGeneCascade(ss)

	gc := ss.GeneCascade
	if gc == nil {
		t.Fatal("gene cascade should be initialized")
	}
	if len(gc.Cascades) != 15 {
		t.Fatalf("expected 15 cascades, got %d", len(gc.Cascades))
	}
	for i, c := range gc.Cascades {
		if len(c.Expression) != 6 {
			t.Fatalf("bot %d: expected 6 genes, got %d", i, len(c.Expression))
		}
		if len(c.Regulation) != 6 {
			t.Fatalf("bot %d: expected 6x6 regulation", i)
		}
	}
}

func TestClearGeneCascade(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.GeneCascadeOn = true
	InitGeneCascade(ss)
	ClearGeneCascade(ss)

	if ss.GeneCascade != nil {
		t.Fatal("should be nil")
	}
	if ss.GeneCascadeOn {
		t.Fatal("should be false")
	}
}

func TestTickGeneCascade(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitGeneCascade(ss)

	// Give bots varied environments
	for i := range ss.Bots {
		ss.Bots[i].NearestPickupDist = float64(i) * 20
		ss.Bots[i].Speed = SwarmBotSpeed
	}

	for tick := 0; tick < 100; tick++ {
		TickGeneCascade(ss)
	}

	gc := ss.GeneCascade
	if gc.ActiveGenes == 0 && gc.CascadeEvents == 0 {
		t.Fatal("should have some gene activity")
	}
}

func TestTickGeneCascadeNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickGeneCascade(ss) // should not panic
}

func TestEvolveGeneCascades(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitGeneCascade(ss)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveGeneCascades(ss, sorted)
	if ss.GeneCascade.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.GeneCascade.Generation)
	}
}

func TestCascadeAvgExpr(t *testing.T) {
	if CascadeAvgExpr(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestCascadeActiveGenes(t *testing.T) {
	if CascadeActiveGenes(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestCloneCascade(t *testing.T) {
	src := BotCascade{
		Expression: []float64{0.5, 0.3},
		Regulation: [][]float64{
			{0, 0.5},
			{-0.3, 0},
		},
		EnvSensitivity: [4]float64{1, 0, 0, 0},
	}

	dst := cloneCascade(src)
	dst.Expression[0] = 0.9
	dst.Regulation[0][1] = 999

	if src.Expression[0] == 0.9 {
		t.Fatal("clone should be independent (expression)")
	}
	if src.Regulation[0][1] == 999 {
		t.Fatal("clone should be independent (regulation)")
	}
}

func TestComputePhenotype(t *testing.T) {
	c := &BotCascade{
		Expression: []float64{0.8, 0.7, 0.6, 0.9, 0.5, 0.4},
	}
	computePhenotype(c, 6)

	if c.Phenotype.SpeedMod < 0.7 {
		t.Fatal("speed mod should be positive")
	}
	if c.Phenotype.SocialPull <= 0 {
		t.Fatal("social pull should be positive with high gene 3")
	}
}
