package swarm

import (
	"math/rand"
	"testing"
)

func TestInitAdaptiveImmune(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitAdaptiveImmune(ss)

	ai := ss.AdaptiveImmune
	if ai == nil {
		t.Fatal("adaptive immune should be initialized")
	}
	if len(ai.BotRoles) != 15 {
		t.Fatalf("expected 15 roles, got %d", len(ai.BotRoles))
	}

	// Should have a mix of roles
	bCells, tCells := 0, 0
	for _, r := range ai.BotRoles {
		if r.Role == ImmuneBCell {
			bCells++
		}
		if r.Role == ImmuneTCell {
			tCells++
		}
	}
	if bCells == 0 && tCells == 0 {
		t.Fatal("should have some B and T cells")
	}
}

func TestClearAdaptiveImmune(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.AdaptiveImmuneOn = true
	InitAdaptiveImmune(ss)
	ClearAdaptiveImmune(ss)

	if ss.AdaptiveImmune != nil {
		t.Fatal("should be nil")
	}
	if ss.AdaptiveImmuneOn {
		t.Fatal("should be false")
	}
}

func TestTickAdaptiveImmune(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitAdaptiveImmune(ss)

	// Create an anomalous cluster to trigger threat detection
	for i := 0; i < 15; i++ {
		ss.Bots[i].X = 100
		ss.Bots[i].Y = 100
		ss.Bots[i].Speed = SwarmBotSpeed * 3 // anomalous speed
	}

	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		TickAdaptiveImmune(ss)
	}

	// System should have detected something
	ai := ss.AdaptiveImmune
	if ai.TotalThreats == 0 {
		t.Fatal("should have detected threats")
	}
}

func TestTickAdaptiveImmuneNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickAdaptiveImmune(ss) // should not panic
}

func TestMatchThreatsToMemory(t *testing.T) {
	ai := &AdaptiveImmuneState{
		ThreatMemory: []ThreatSignature{
			{SizeProfile: 0.7, Exposures: 1, LastSeen: 0},
		},
		ActiveThreats: []ActiveThreat{
			{Severity: 0.72, SignatureIdx: -1},
		},
	}

	matchThreatsToMemory(ai)

	if !ai.ActiveThreats[0].Memorized {
		t.Fatal("threat should match memory")
	}
	if ai.ReExposures != 1 {
		t.Fatal("should count as re-exposure")
	}
}

func TestAdaptiveImmuneThreats(t *testing.T) {
	if AdaptiveImmuneThreats(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestAdaptiveImmuneMemory(t *testing.T) {
	if AdaptiveImmuneMemory(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestAdaptiveImmuneReExposures(t *testing.T) {
	if AdaptiveImmuneReExposures(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestMemoryFormation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitAdaptiveImmune(ss)

	// Create threat scenario
	for i := 0; i < 15; i++ {
		ss.Bots[i].X = 100
		ss.Bots[i].Y = 100
		ss.Bots[i].Speed = SwarmBotSpeed * 3
	}

	for tick := 0; tick < 300; tick++ {
		ss.Tick = tick
		TickAdaptiveImmune(ss)
	}

	ai := ss.AdaptiveImmune
	if ai.MemorizedThreats > ai.MaxMemory {
		t.Fatal("should not exceed max memory")
	}
}
