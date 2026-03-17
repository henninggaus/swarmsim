package swarm

import (
	"math/rand"
	"testing"
)

func TestInitEpisodicMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitEpisodicMemory(ss)

	em := ss.EpisodicMemory
	if em == nil {
		t.Fatal("episodic memory should be initialized")
	}
	if len(em.Memories) != 15 {
		t.Fatalf("expected 15 memory stores, got %d", len(em.Memories))
	}
	if em.MaxMemories != 20 {
		t.Fatal("default max should be 20")
	}
}

func TestClearEpisodicMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.EpisodicMemoryOn = true
	InitEpisodicMemory(ss)
	ClearEpisodicMemory(ss)

	if ss.EpisodicMemory != nil {
		t.Fatal("should be nil after clear")
	}
	if ss.EpisodicMemoryOn {
		t.Fatal("should be false")
	}
}

func TestTickEpisodicMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitEpisodicMemory(ss)

	// Set some bots near pickups
	for i := 0; i < 5; i++ {
		ss.Bots[i].NearestPickupDist = 20
	}

	for tick := 0; tick < 100; tick++ {
		ss.Tick = tick
		TickEpisodicMemory(ss)
	}

	em := ss.EpisodicMemory
	if em.TotalMemories == 0 {
		t.Fatal("should have recorded some memories")
	}
}

func TestTickEpisodicMemoryNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickEpisodicMemory(ss) // should not panic
}

func TestAddMemoryEviction(t *testing.T) {
	em := &EpisodicMemoryState{MaxMemories: 3}
	mems := &[]Episode{}

	for i := 0; i < 5; i++ {
		addMemory(em, mems, Episode{
			Tick:     i,
			Strength: float64(i) * 0.2,
			Value:    1.0,
		})
	}

	if len(*mems) != 3 {
		t.Fatalf("should cap at 3 memories, got %d", len(*mems))
	}

	// Weakest should have been evicted
	for _, ep := range *mems {
		if ep.Strength < 0.3 {
			t.Fatalf("weak memory (strength=%.1f) should have been evicted", ep.Strength)
		}
	}
}

func TestMemoryDecay(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitEpisodicMemory(ss)
	em := ss.EpisodicMemory

	// Manually add a memory
	em.Memories[0] = append(em.Memories[0], Episode{
		Tick:     0,
		X:        100,
		Y:        100,
		Type:     EpisodeFoundResource,
		Strength: 0.0005, // very weak, below decay rate
	})

	ss.Tick = 1
	TickEpisodicMemory(ss)

	// Should have decayed below 0 and been removed
	if len(em.Memories[0]) != 0 {
		t.Fatal("dead memory should have been removed")
	}
}

func TestMemoryCount(t *testing.T) {
	if MemoryCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	em := &EpisodicMemoryState{TotalMemories: 42}
	if MemoryCount(em) != 42 {
		t.Fatal("expected 42")
	}
}

func TestBotMemoryCount(t *testing.T) {
	if BotMemoryCount(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}
	em := &EpisodicMemoryState{
		Memories: [][]Episode{
			{{}, {}, {}},
			{{}},
		},
	}
	if BotMemoryCount(em, 0) != 3 {
		t.Fatal("expected 3")
	}
	if BotMemoryCount(em, 5) != 0 {
		t.Fatal("out of bounds should return 0")
	}
}

func TestStrongestMemory(t *testing.T) {
	if StrongestMemory(nil, 0) != nil {
		t.Fatal("nil should return nil")
	}
	em := &EpisodicMemoryState{
		Memories: [][]Episode{
			{
				{Strength: 0.3, Type: EpisodeDanger},
				{Strength: 0.9, Type: EpisodeDelivered},
				{Strength: 0.5, Type: EpisodeFoundResource},
			},
		},
	}
	best := StrongestMemory(em, 0)
	if best == nil || best.Type != EpisodeDelivered {
		t.Fatal("should return strongest memory (delivery)")
	}
}

func TestEpisodeTypeName(t *testing.T) {
	if EpisodeTypeName(EpisodeFoundResource) != "Ressource" {
		t.Fatal("expected Ressource")
	}
	if EpisodeTypeName(EpisodeDanger) != "Gefahr" {
		t.Fatal("expected Gefahr")
	}
	if EpisodeTypeName(99) != "?" {
		t.Fatal("expected ?")
	}
}
