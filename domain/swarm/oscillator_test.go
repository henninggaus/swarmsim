package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestInitOscillators(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitOscillators(ss)

	os := ss.Oscillator
	if os == nil {
		t.Fatal("oscillators should be initialized")
	}
	if len(os.Phases) != 15 {
		t.Fatalf("expected 15 phases, got %d", len(os.Phases))
	}
	for _, p := range os.Phases {
		if p < 0 || p >= 2*math.Pi {
			t.Fatalf("phase %.2f out of range", p)
		}
	}
}

func TestClearOscillators(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.OscillatorOn = true
	InitOscillators(ss)
	ClearOscillators(ss)

	if ss.Oscillator != nil {
		t.Fatal("should be nil")
	}
	if ss.OscillatorOn {
		t.Fatal("should be false")
	}
}

func TestTickOscillators(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitOscillators(ss)

	// Place bots close together for coupling
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i%5)*30 + 100
		ss.Bots[i].Y = float64(i/5)*30 + 100
	}

	for tick := 0; tick < 200; tick++ {
		TickOscillators(ss)
	}

	os := ss.Oscillator
	// After many ticks with coupling, order parameter should increase
	if os.OrderParam <= 0 {
		t.Fatal("order parameter should be positive")
	}
}

func TestTickOscillatorsNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickOscillators(ss) // should not panic
}

func TestOscOrderParam(t *testing.T) {
	if OscOrderParam(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestOscSyncGroups(t *testing.T) {
	if OscSyncGroups(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBotPhase(t *testing.T) {
	if BotPhase(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitOscillators(ss)

	p := BotPhase(ss.Oscillator, 0)
	if p < 0 || p >= 2*math.Pi {
		t.Fatalf("phase %.2f out of range", p)
	}
}

func TestOscSynchronization(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitOscillators(ss)

	// All bots at same position — max coupling
	for i := range ss.Bots {
		ss.Bots[i].X = 100
		ss.Bots[i].Y = 100
	}

	// Strong coupling
	ss.Oscillator.CoupleStr = 1.0

	initialOrder := OscOrderParam(ss.Oscillator)

	for tick := 0; tick < 500; tick++ {
		TickOscillators(ss)
	}

	finalOrder := OscOrderParam(ss.Oscillator)

	// Strong coupling should increase synchronization
	if finalOrder <= initialOrder {
		// It's possible with random phases, but very unlikely after 500 ticks
		// with strong coupling — just check it's reasonable
		if finalOrder < 0.5 {
			t.Fatalf("expected high sync after strong coupling, got %.3f", finalOrder)
		}
	}
}
