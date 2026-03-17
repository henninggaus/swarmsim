package swarm

import (
	"math/rand"
	"testing"
)

func TestInitSpecialization(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitSpecialization(ss)

	sp := ss.Specialization
	if sp == nil {
		t.Fatal("specialization should be initialized")
	}
	if len(sp.Profiles) != 15 {
		t.Fatalf("expected 15 profiles, got %d", len(sp.Profiles))
	}
}

func TestClearSpecialization(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.SpecializationOn = true
	InitSpecialization(ss)
	ClearSpecialization(ss)

	if ss.Specialization != nil {
		t.Fatal("should be nil")
	}
	if ss.SpecializationOn {
		t.Fatal("should be false")
	}
}

func TestTickSpecialization(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSpecialization(ss)

	// Set varied conditions
	for i := 0; i < 5; i++ {
		ss.Bots[i].CarryingPkg = 0 // foragers
		ss.Bots[i].NearestPickupDist = 30
	}
	for i := 5; i < 10; i++ {
		ss.Bots[i].NeighborCount = 1 // scouts
		ss.Bots[i].Speed = SwarmBotSpeed
	}
	for i := 10; i < 15; i++ {
		ss.Bots[i].NeighborCount = 8 // guards
	}

	for tick := 0; tick < 500; tick++ {
		TickSpecialization(ss)
	}

	sp := ss.Specialization
	if sp.AvgSpecialization <= 0 {
		t.Fatal("should have some specialization")
	}
}

func TestTickSpecializationNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickSpecialization(ss) // should not panic
}

func TestRoleName(t *testing.T) {
	if RoleName(RoleForager) != "Sammler" {
		t.Fatal("expected Sammler")
	}
	if RoleName(RoleScout) != "Kundschafter" {
		t.Fatal("expected Kundschafter")
	}
	if RoleName(RoleGuard) != "Waechter" {
		t.Fatal("expected Waechter")
	}
	if RoleName(RoleBuilder) != "Bauer" {
		t.Fatal("expected Bauer")
	}
	if RoleName(99) != "?" {
		t.Fatal("expected ?")
	}
}

func TestGetBotRole(t *testing.T) {
	if GetBotRole(nil, 0) != RoleForager {
		t.Fatal("nil should return RoleForager")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitSpecialization(ss)

	role := GetBotRole(ss.Specialization, 0)
	if role < 0 || role > 3 {
		t.Fatal("invalid role")
	}
}

func TestGetBotSpecialization(t *testing.T) {
	if GetBotSpecialization(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestPopulationSpecialization(t *testing.T) {
	if PopulationSpecialization(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestSpecializationRoleDistribution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 40)
	InitSpecialization(ss)

	// Run enough ticks for roles to differentiate
	for tick := 0; tick < 200; tick++ {
		TickSpecialization(ss)
	}

	sp := ss.Specialization
	total := 0
	for _, c := range sp.RoleCounts {
		total += c
	}
	if total != 40 {
		t.Fatalf("role counts should sum to 40, got %d", total)
	}
}
