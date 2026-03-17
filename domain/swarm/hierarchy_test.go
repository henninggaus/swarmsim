package swarm

import (
	"math/rand"
	"testing"
)

func TestInitHierarchy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitHierarchy(ss, 5)

	hs := ss.Hierarchy
	if hs == nil {
		t.Fatal("hierarchy should be initialized")
	}
	if hs.TotalSquads != 4 {
		t.Fatalf("expected 4 squads (20/5), got %d", hs.TotalSquads)
	}
	if len(hs.BotSquad) != 20 {
		t.Fatalf("expected 20 bot assignments, got %d", len(hs.BotSquad))
	}
	// Every bot should be assigned
	for i, sq := range hs.BotSquad {
		if sq < 0 {
			t.Fatalf("bot %d should be assigned to a squad", i)
		}
	}
}

func TestInitHierarchyClamp(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitHierarchy(ss, 1) // below min
	if ss.Hierarchy.SquadSize != 2 {
		t.Fatal("should clamp squad size to 2")
	}
}

func TestClearHierarchy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ss.HierarchyOn = true
	InitHierarchy(ss, 5)
	ClearHierarchy(ss)

	if ss.Hierarchy != nil {
		t.Fatal("should be nil after clear")
	}
	if ss.HierarchyOn {
		t.Fatal("should be false")
	}
}

func TestTickHierarchy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitHierarchy(ss, 5)

	for tick := 0; tick < 50; tick++ {
		ss.Tick = tick
		TickHierarchy(ss)
	}

	hs := ss.Hierarchy
	if hs.LeaderCount != 4 {
		t.Fatalf("expected 4 leaders, got %d", hs.LeaderCount)
	}
	if hs.AvgSquadSize != 5 {
		t.Fatalf("expected avg 5, got %.1f", hs.AvgSquadSize)
	}
}

func TestTickHierarchyNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickHierarchy(ss) // should not panic
}

func TestElectLeaders(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitHierarchy(ss, 5)

	// Give bot 3 high deliveries
	ss.Bots[3].Stats.TotalDeliveries = 50

	electLeaders(ss, ss.Hierarchy)

	// Bot 3 is in squad 0, should be leader
	if ss.Hierarchy.Squads[0].LeaderID != 3 {
		t.Fatalf("expected bot 3 as leader, got %d", ss.Hierarchy.Squads[0].LeaderID)
	}
}

func TestSquadCohesion(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitHierarchy(ss, 5)

	// Place all squad 0 members at same point
	for _, m := range ss.Hierarchy.Squads[0].Members {
		ss.Bots[m].X = 400
		ss.Bots[m].Y = 400
	}

	updateSquadStats(ss, &ss.Hierarchy.Squads[0])
	if ss.Hierarchy.Squads[0].Cohesion < 0.9 {
		t.Fatalf("cohesion should be ~1.0 for co-located bots, got %.3f",
			ss.Hierarchy.Squads[0].Cohesion)
	}
}

func TestSquadCount(t *testing.T) {
	if SquadCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	hs := &HierarchyState{Squads: make([]Squad, 3)}
	if SquadCount(hs) != 3 {
		t.Fatal("expected 3")
	}
}

func TestBotSquadID(t *testing.T) {
	if BotSquadID(nil, 0) != -1 {
		t.Fatal("nil should return -1")
	}
	hs := &HierarchyState{BotSquad: []int{2, 0, 1}}
	if BotSquadID(hs, 0) != 2 {
		t.Fatal("expected 2")
	}
	if BotSquadID(hs, 5) != -1 {
		t.Fatal("out of bounds should return -1")
	}
}

func TestHierarchyDepth(t *testing.T) {
	if HierarchyDepth(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 30)
	InitHierarchy(ss, 5)
	depth := HierarchyDepth(ss.Hierarchy)
	if depth < 2 {
		t.Fatalf("expected depth >= 2, got %d", depth)
	}
}

func TestTaskName(t *testing.T) {
	if TaskName(TaskExplore) != "Erkunden" {
		t.Fatal("expected Erkunden")
	}
	if TaskName(99) != "?" {
		t.Fatal("expected ?")
	}
}

func TestGroupIDs(t *testing.T) {
	groups := groupIDs([]int{0, 1, 2, 3, 4, 5, 6}, 3)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if len(groups[2]) != 1 {
		t.Fatalf("last group should have 1 element, got %d", len(groups[2]))
	}
}
