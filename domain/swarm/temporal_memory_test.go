package swarm

import (
	"math/rand"
	"testing"
)

func TestInitTemporalMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitTemporalMemory(ss)

	tm := ss.TemporalMemory
	if tm == nil {
		t.Fatal("temporal memory should be initialized")
	}
	if len(tm.Anticipations) != 10 {
		t.Fatalf("expected 10 anticipations, got %d", len(tm.Anticipations))
	}
}

func TestClearTemporalMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.TemporalMemoryOn = true
	InitTemporalMemory(ss)
	ClearTemporalMemory(ss)

	if ss.TemporalMemory != nil {
		t.Fatal("should be nil")
	}
	if ss.TemporalMemoryOn {
		t.Fatal("should be false")
	}
}

func TestTickTemporalMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitTemporalMemory(ss)

	// Simulate periodic resource events
	for tick := 0; tick < 500; tick++ {
		ss.Tick = tick

		// Create periodic resource appearance
		if tick%100 < 20 {
			for i := 0; i < 10; i++ {
				ss.Bots[i].NearestPickupDist = 20
			}
		} else {
			for i := range ss.Bots {
				ss.Bots[i].NearestPickupDist = 200
			}
		}

		TickTemporalMemory(ss)
	}

	tm := ss.TemporalMemory
	if len(tm.Events) == 0 {
		t.Fatal("should have recorded some events")
	}
}

func TestTickTemporalMemoryNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickTemporalMemory(ss) // should not panic
}

func TestDetectPatterns(t *testing.T) {
	tm := &TemporalMemoryState{
		Events:    make([]TemporalEvent, 0),
		MaxEvents: 500,
	}

	// Create a clear periodic pattern: events at ticks 0, 100, 200, 300, 400
	for tick := 0; tick <= 400; tick += 100 {
		tm.Events = append(tm.Events, TemporalEvent{
			Tick:      tick,
			EventType: TEvtResource,
			X:         50,
			Y:         50,
			Strength:  0.5,
		})
	}

	detectPatterns(tm)

	if len(tm.Patterns) == 0 {
		t.Fatal("should have detected a pattern")
	}

	found := false
	for _, p := range tm.Patterns {
		if p.EventType == TEvtResource && abs(p.Period-100) < 20 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("should have detected ~100 tick period")
	}
}

func TestAddTemporalEvent(t *testing.T) {
	tm := &TemporalMemoryState{
		Events:    make([]TemporalEvent, 0),
		MaxEvents: 3,
	}

	for i := 0; i < 5; i++ {
		addTemporalEvent(tm, TemporalEvent{Tick: i})
	}

	if len(tm.Events) != 3 {
		t.Fatalf("expected max 3 events, got %d", len(tm.Events))
	}
	if tm.Events[0].Tick != 2 {
		t.Fatal("oldest events should have been removed")
	}
}

func TestTemporalPatternsFound(t *testing.T) {
	if TemporalPatternsFound(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestTemporalPredictionRate(t *testing.T) {
	if TemporalPredictionRate(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestTemporalEventCount(t *testing.T) {
	if TemporalEventCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestAbs(t *testing.T) {
	if abs(-5) != 5 {
		t.Fatal("abs(-5) should be 5")
	}
	if abs(3) != 3 {
		t.Fatal("abs(3) should be 3")
	}
}
