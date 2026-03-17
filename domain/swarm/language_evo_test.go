package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestInitLanguageEvo(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitLanguageEvo(ss, 4)

	if ss.LanguageEvo == nil {
		t.Fatal("language evo should be initialized")
	}
	le := ss.LanguageEvo
	if le.SignalSize != 4 {
		t.Fatalf("expected signal size 4, got %d", le.SignalSize)
	}
	if len(le.Encoders) != 10 {
		t.Fatal("should have 10 encoders")
	}
	if len(le.Decoders) != 10 {
		t.Fatal("should have 10 decoders")
	}
}

func TestEncodeSignal(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitLanguageEvo(ss, 4)
	le := ss.LanguageEvo

	signal := EncodeSignal(le, &le.Encoders[0], &ss.Bots[0], ss)
	if signal == nil {
		t.Fatal("signal should not be nil")
	}
	if len(signal) != 4 {
		t.Fatalf("expected 4 values, got %d", len(signal))
	}
	// All values should be in [-1, 1] (tanh output)
	for i, v := range signal {
		if v < -1 || v > 1 {
			t.Fatalf("signal[%d] = %f out of range", i, v)
		}
	}
}

func TestDecodeSignal(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitLanguageEvo(ss, 4)
	le := ss.LanguageEvo

	signal := []float64{0.5, -0.3, 0.8, -0.1}
	biases := DecodeSignal(le, &le.Decoders[0], signal)

	// All biases should be in [-1, 1]
	for i, b := range biases {
		if b < -1 || b > 1 {
			t.Fatalf("bias[%d] = %f out of range", i, b)
		}
	}
}

func TestBotContext(t *testing.T) {
	bot := SwarmBot{CarryingPkg: 2}
	if BotContext(&bot) != 1 {
		t.Fatal("carrying bot should have context 1")
	}

	bot2 := SwarmBot{CarryingPkg: -1, NearestPickupDist: 30}
	if BotContext(&bot2) != 2 {
		t.Fatal("near-pickup bot should have context 2")
	}

	bot3 := SwarmBot{CarryingPkg: -1, NearestPickupDist: 999, NearestDropoffDist: 30}
	if BotContext(&bot3) != 3 {
		t.Fatal("near-dropoff bot should have context 3")
	}

	bot4 := SwarmBot{CarryingPkg: -1, NearestPickupDist: 999, NearestDropoffDist: 999}
	if BotContext(&bot4) != 0 {
		t.Fatal("exploring bot should have context 0")
	}
}

func TestTickLanguageEvo(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitLanguageEvo(ss, 3)

	// Run a few ticks
	for i := 0; i < 10; i++ {
		ss.Tick = i
		TickLanguageEvo(ss)
	}

	// Should have some signals
	if len(ss.LanguageEvo.Signals) == 0 {
		t.Fatal("should have some signals after ticks")
	}
}

func TestSignalSimilarity(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{1, 0, 0}
	sim := SignalSimilarity(a, b)
	if math.Abs(sim-1.0) > 0.001 {
		t.Fatalf("identical vectors should have similarity 1.0, got %f", sim)
	}

	c := []float64{-1, 0, 0}
	sim = SignalSimilarity(a, c)
	if math.Abs(sim+1.0) > 0.001 {
		t.Fatalf("opposite vectors should have similarity -1.0, got %f", sim)
	}

	d := []float64{0, 1, 0}
	sim = SignalSimilarity(a, d)
	if math.Abs(sim) > 0.001 {
		t.Fatalf("orthogonal vectors should have similarity 0, got %f", sim)
	}
}

func TestAnalyzeVocabulary(t *testing.T) {
	le := &LanguageEvo{
		SignalSize: 2,
		Signals: []LanguageSignal{
			{Values: []float64{0.5, 0.5}},   // HH
			{Values: []float64{0.5, 0.5}},   // HH (same)
			{Values: []float64{-0.5, -0.5}}, // LL
			{Values: []float64{0.0, 0.0}},   // MM
		},
	}
	vocab := analyzeVocabulary(le)
	if vocab != 3 {
		t.Fatalf("expected 3 distinct clusters, got %d", vocab)
	}
}

func TestEvolveLanguage(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitLanguageEvo(ss, 3)

	// Create sorted indices (just sequential for test)
	indices := make([]int, 10)
	for i := range indices {
		indices[i] = i
	}

	EvolveLanguage(ss, indices)
	if ss.LanguageEvo.Generation != 1 {
		t.Fatal("generation should be 1 after evolution")
	}
}

func TestLanguageSignalCount(t *testing.T) {
	le := &LanguageEvo{
		Signals: []LanguageSignal{{}, {}, {}},
	}
	if LanguageSignalCount(le) != 3 {
		t.Fatal("should count 3 signals")
	}
	if LanguageSignalCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}
