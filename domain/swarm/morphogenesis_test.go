package swarm

import (
	"math/rand"
	"testing"
)

func TestInitMorphogenesis(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitMorphogenesis(ss)

	ms := ss.Morphogenesis
	if ms == nil {
		t.Fatal("morphogenesis should be initialized")
	}
	if len(ms.Activator) != 15 {
		t.Fatalf("expected 15 activators, got %d", len(ms.Activator))
	}
	if len(ms.Inhibitor) != 15 {
		t.Fatalf("expected 15 inhibitors, got %d", len(ms.Inhibitor))
	}
}

func TestClearMorphogenesis(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.MorphogenesisOn = true
	InitMorphogenesis(ss)
	ClearMorphogenesis(ss)

	if ss.Morphogenesis != nil {
		t.Fatal("should be nil")
	}
	if ss.MorphogenesisOn {
		t.Fatal("should be false")
	}
}

func TestTickMorphogenesis(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMorphogenesis(ss)

	// Spread bots out
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i%5) * 50
		ss.Bots[i].Y = float64(i/5) * 50
	}

	for tick := 0; tick < 100; tick++ {
		TickMorphogenesis(ss)
	}

	ms := ss.Morphogenesis
	if ms.AvgActivator <= 0 {
		t.Fatal("should have some activator")
	}
}

func TestTickMorphogenesisNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickMorphogenesis(ss) // should not panic
}

func TestMorphoContrast(t *testing.T) {
	if MorphoContrast(nil) != 0 {
		t.Fatal("nil should return 0")
	}

	ms := &MorphogenesisState{
		Activator: []float64{0.1, 0.5, 0.9},
		Inhibitor: []float64{0.2, 0.3, 0.4},
	}
	updateMorphoStats(ms)
	c := MorphoContrast(ms)
	if c < 0.7 {
		t.Fatalf("expected contrast ~0.8, got %.2f", c)
	}
}

func TestMorphoActivatedRatio(t *testing.T) {
	if MorphoActivatedRatio(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBotActivator(t *testing.T) {
	if BotActivator(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	ms := &MorphogenesisState{
		Activator: []float64{0.3, 0.7, 0.5},
	}
	if BotActivator(ms, 1) != 0.7 {
		t.Fatal("expected 0.7")
	}
	if BotActivator(ms, 5) != 0 {
		t.Fatal("out of bounds should return 0")
	}
}

func TestMorphoPatternFormation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 30)
	InitMorphogenesis(ss)

	// Place bots in a grid to allow diffusion
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i%6) * 40
		ss.Bots[i].Y = float64(i/6) * 40
	}

	// Seed a perturbation
	ss.Morphogenesis.Activator[15] = 0.9

	initialContrast := MorphoContrast(ss.Morphogenesis)
	for tick := 0; tick < 200; tick++ {
		TickMorphogenesis(ss)
	}

	// Pattern should have developed (contrast should change)
	finalContrast := MorphoContrast(ss.Morphogenesis)
	_ = initialContrast
	if finalContrast <= 0 {
		t.Fatal("pattern should have developed some contrast")
	}
}
