package swarm

import (
	"math/rand"
	"testing"
)

func TestInitCollectiveDream(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitCollectiveDream(ss)

	cd := ss.CollectiveDream
	if cd == nil {
		t.Fatal("collective dream should be initialized")
	}
	if cd.MaxMemories != 100 {
		t.Fatalf("expected 100 max memories, got %d", cd.MaxMemories)
	}
}

func TestClearCollectiveDream(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.CollectiveDreamOn = true
	InitCollectiveDream(ss)
	ClearCollectiveDream(ss)

	if ss.CollectiveDream != nil {
		t.Fatal("should be nil")
	}
	if ss.CollectiveDreamOn {
		t.Fatal("should be false")
	}
}

func TestTickCollectiveDream(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitCollectiveDream(ss)

	// Set some bots as productive
	for i := 0; i < 10; i++ {
		ss.Bots[i].CarryingPkg = 0
		ss.Bots[i].NearestDropoffDist = 50
		ss.Bots[i].Speed = SwarmBotSpeed
	}

	for tick := 0; tick < 300; tick++ {
		ss.Tick = tick
		TickCollectiveDream(ss)
	}

	cd := ss.CollectiveDream
	if cd.MemoriesStored == 0 {
		t.Fatal("should have collected some memories")
	}
}

func TestTickCollectiveDreamNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickCollectiveDream(ss) // should not panic
}

func TestDreamPhaseActivation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitCollectiveDream(ss)

	// First collect some memories
	for i := range ss.Bots {
		ss.Bots[i].CarryingPkg = 0
		ss.Bots[i].NearestDropoffDist = 30
		ss.Bots[i].Speed = SwarmBotSpeed
	}

	for tick := 0; tick < 100; tick++ {
		ss.Tick = tick
		TickCollectiveDream(ss)
	}

	// Now make bots idle to trigger dream
	for i := range ss.Bots {
		ss.Bots[i].Speed = SwarmBotSpeed * 0.1
	}

	for tick := 100; tick < 200; tick++ {
		ss.Tick = tick
		TickCollectiveDream(ss)
	}

	cd := ss.CollectiveDream
	if cd.TotalDreams == 0 {
		t.Fatal("should have entered at least one dream phase")
	}
}

func TestDreamIsActive(t *testing.T) {
	if DreamIsActive(nil) {
		t.Fatal("nil should return false")
	}
}

func TestDreamInsights(t *testing.T) {
	if DreamInsights(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestDreamMemoryCount(t *testing.T) {
	if DreamMemoryCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestDreamAvgFitness(t *testing.T) {
	if DreamAvgFitness(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestSelectMemory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitCollectiveDream(ss)

	cd := ss.CollectiveDream
	cd.Memories = []StrategyMemory{
		{Fitness: 0.9, Angle: 1.0},
		{Fitness: 0.1, Angle: 2.0},
		{Fitness: 0.5, Angle: 3.0},
	}

	// High fitness should be selected more often
	counts := [3]int{}
	for i := 0; i < 1000; i++ {
		m := selectMemory(ss, cd)
		for j, mem := range cd.Memories {
			if m.Angle == mem.Angle {
				counts[j]++
				break
			}
		}
	}

	if counts[0] < counts[1] {
		t.Fatal("high fitness memory should be selected more often")
	}
}

func TestComputeSwarmActivity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	for i := range ss.Bots {
		ss.Bots[i].Speed = SwarmBotSpeed
	}

	act := computeSwarmActivity(ss)
	if act < 0.9 || act > 1.1 {
		t.Fatalf("expected activity ~1.0, got %.2f", act)
	}
}
