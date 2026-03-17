package swarm

import (
	"math/rand"
	"testing"
)

func TestInitReactionDiffusion(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitReactionDiffusion(ss, 40)

	rd := ss.ReactionDiffusion
	if rd == nil {
		t.Fatal("reaction diffusion should be initialized")
	}
	if rd.GridW != 40 || rd.GridH != 40 {
		t.Fatalf("expected 40x40 grid, got %dx%d", rd.GridW, rd.GridH)
	}
	if len(rd.A) != 1600 {
		t.Fatalf("expected 1600 cells, got %d", len(rd.A))
	}
	if rd.FeedRate != 0.055 {
		t.Fatal("default feed rate should be 0.055")
	}
}

func TestInitReactionDiffusionClamp(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitReactionDiffusion(ss, 5) // below min
	if ss.ReactionDiffusion.GridW != 10 {
		t.Fatal("should clamp to min 10")
	}
}

func TestClearReactionDiffusion(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.ReactionDiffusionOn = true
	InitReactionDiffusion(ss, 20)
	ClearReactionDiffusion(ss)

	if ss.ReactionDiffusion != nil {
		t.Fatal("should be nil after clear")
	}
	if ss.ReactionDiffusionOn {
		t.Fatal("should be false after clear")
	}
}

func TestTickReactionDiffusion(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitReactionDiffusion(ss, 20)

	// Run a few ticks
	for i := 0; i < 20; i++ {
		TickReactionDiffusion(ss)
	}

	rd := ss.ReactionDiffusion
	if rd.Tick != 20 {
		t.Fatalf("expected tick 20, got %d", rd.Tick)
	}
	if rd.AvgA <= 0 {
		t.Fatal("average A should be > 0")
	}
	if rd.Pattern == "" {
		t.Fatal("pattern should be detected")
	}
}

func TestTickReactionDiffusionNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickReactionDiffusion(ss) // should not panic
}

func TestRDGetConcentration(t *testing.T) {
	if a, b := RDGetConcentration(nil, 100, 100); a != 0 || b != 0 {
		t.Fatal("nil should return 0,0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitReactionDiffusion(ss, 20)

	a, _ := RDGetConcentration(ss.ReactionDiffusion, 50, 50)
	if a < 0 || a > 1 {
		t.Fatalf("A out of range: %.3f", a)
	}

	// Out of bounds
	a, b := RDGetConcentration(ss.ReactionDiffusion, -100, -100)
	if a != 0 || b != 0 {
		t.Fatal("out of bounds should return 0,0")
	}
}

func TestRDSetParameters(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitReactionDiffusion(ss, 20)
	rd := ss.ReactionDiffusion

	RDSetParameters(rd, "spots")
	if rd.FeedRate != 0.035 {
		t.Fatalf("expected 0.035, got %f", rd.FeedRate)
	}

	RDSetParameters(rd, "maze")
	if rd.FeedRate != 0.029 {
		t.Fatalf("expected 0.029, got %f", rd.FeedRate)
	}

	RDSetParameters(nil, "spots") // should not panic
}

func TestDetectPattern(t *testing.T) {
	rd := &ReactionDiffusionState{
		GridW: 10,
		GridH: 10,
		A:     make([]float64, 100),
		B:     make([]float64, 100),
		AvgB:  0.001,
	}
	if detectPattern(rd) != "Homogen" {
		t.Fatal("low B should be Homogen")
	}

	rd.AvgB = 0.5
	// Set few B cells high
	for i := 0; i < 5; i++ {
		rd.B[i] = 0.5
	}
	p := detectPattern(rd)
	if p != "Punkte" {
		t.Fatalf("expected Punkte, got %s", p)
	}
}

func TestConcentrationClamp(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitReactionDiffusion(ss, 10)

	// Force extreme values
	rd := ss.ReactionDiffusion
	for i := range rd.A {
		rd.A[i] = 2.0
		rd.B[i] = 2.0
	}

	TickReactionDiffusion(ss)

	for i := range rd.A {
		if rd.A[i] < 0 || rd.A[i] > 1 {
			t.Fatalf("A[%d]=%.3f out of [0,1]", i, rd.A[i])
		}
		if rd.B[i] < 0 || rd.B[i] > 1 {
			t.Fatalf("B[%d]=%.3f out of [0,1]", i, rd.B[i])
		}
	}
}
