package swarm

import (
	"math/rand"
	"testing"
)

func TestInitImmune(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitImmune(ss)

	if ss.Immune == nil {
		t.Fatal("immune should be initialized")
	}
	if len(ss.Immune.Cells) != 20 {
		t.Fatalf("expected 20 cells, got %d", len(ss.Immune.Cells))
	}
	// Should have some detectors
	detCount := 0
	for _, c := range ss.Immune.Cells {
		if c.IsDetector {
			detCount++
		}
	}
	if detCount == 0 {
		t.Fatal("should have some detector cells")
	}
}

func TestClearImmune(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.ImmuneOn = true
	InitImmune(ss)
	ClearImmune(ss)

	if ss.Immune != nil {
		t.Fatal("should be nil")
	}
	if ss.ImmuneOn {
		t.Fatal("should be false")
	}
}

func TestTickImmune(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitImmune(ss)

	// Give bots varying speeds to create behavioral variance
	for i := range ss.Bots {
		ss.Bots[i].Speed = float64(i%5) * 0.3
		ss.Bots[i].NeighborCount = i % 8
	}

	for tick := 0; tick < 100; tick++ {
		ss.Tick = tick
		TickImmune(ss)
	}

	is := ss.Immune
	if is.AvgHealth <= 0 {
		t.Fatal("avg health should be > 0")
	}
}

func TestTickImmuneNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickImmune(ss) // should not panic
}

func TestBotSignature(t *testing.T) {
	cell := &ImmuneCell{
		AvgSpeed:     1.0,
		AvgTurnRate:  0.5,
		AvgNeighbors: 3.0,
		AvgDelivery:  0.1,
	}
	sig := botSignature(cell)
	if sig[0] != 1.0 || sig[2] != 3.0 {
		t.Fatal("signature should match cell averages")
	}
}

func TestStoreThreatPattern(t *testing.T) {
	is := &ImmuneState{MaxMemoryCells: 3}
	sig := [4]float64{1, 2, 3, 4}

	storeThreatPattern(is, sig, 0)
	if len(is.ThreatPatterns) != 1 {
		t.Fatal("should store one pattern")
	}

	// Same pattern again — should strengthen, not add
	storeThreatPattern(is, sig, 1)
	if len(is.ThreatPatterns) != 1 {
		t.Fatal("similar pattern should strengthen existing")
	}
	if is.ThreatPatterns[0].Responses != 1 {
		t.Fatal("should have 1 response")
	}

	// Fill to capacity
	storeThreatPattern(is, [4]float64{10, 20, 30, 40}, 2)
	storeThreatPattern(is, [4]float64{100, 200, 300, 400}, 3)
	if len(is.ThreatPatterns) != 3 {
		t.Fatalf("expected 3, got %d", len(is.ThreatPatterns))
	}

	// Overflow — should evict weakest
	storeThreatPattern(is, [4]float64{1000, 2000, 3000, 4000}, 4)
	if len(is.ThreatPatterns) != 3 {
		t.Fatal("should not exceed max")
	}
}

func TestImmuneAnomalyCount(t *testing.T) {
	if ImmuneAnomalyCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestImmuneAvgHealth(t *testing.T) {
	if ImmuneAvgHealth(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestImmuneThreatCount(t *testing.T) {
	if ImmuneThreatCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	is := &ImmuneState{ThreatPatterns: make([]ThreatPattern, 5)}
	if ImmuneThreatCount(is) != 5 {
		t.Fatal("expected 5")
	}
}

func TestImmuneDetectorRotation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitImmune(ss)

	// Age all detectors past rotation threshold
	for i := range ss.Immune.Cells {
		if ss.Immune.Cells[i].IsDetector {
			ss.Immune.Cells[i].DetectorAge = 600
		}
	}

	rotateDetectors(ss, ss.Immune)

	// Should still have ~20% detectors
	detCount := 0
	for _, c := range ss.Immune.Cells {
		if c.IsDetector {
			detCount++
		}
	}
	if detCount < 3 {
		t.Fatalf("should have ~20%% detectors after rotation, got %d", detCount)
	}
}
