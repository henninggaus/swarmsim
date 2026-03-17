package swarm

import (
	"math/rand"
	"testing"
)

func TestInitHomeostasis(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitHomeostasis(ss)

	hs := ss.Homeostasis
	if hs == nil {
		t.Fatal("homeostasis should be initialized")
	}
	if len(hs.Drives) != 15 {
		t.Fatalf("expected 15 drives, got %d", len(hs.Drives))
	}
	for i, d := range hs.Drives {
		if d.Energy < 0 || d.Energy > 1 {
			t.Fatalf("bot %d: energy %.2f out of range", i, d.Energy)
		}
	}
}

func TestClearHomeostasis(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.HomeostasisOn = true
	InitHomeostasis(ss)
	ClearHomeostasis(ss)

	if ss.Homeostasis != nil {
		t.Fatal("should be nil")
	}
	if ss.HomeostasisOn {
		t.Fatal("should be false")
	}
}

func TestTickHomeostasis(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitHomeostasis(ss)

	// Set some conditions
	for i := 0; i < 5; i++ {
		ss.Bots[i].NearestPickupDist = 20 // near resources
	}
	for i := 10; i < 15; i++ {
		ss.Bots[i].NeighborCount = 12 // crowded → stress
	}

	initialEnergy := ss.Homeostasis.Drives[0].Energy

	for tick := 0; tick < 200; tick++ {
		TickHomeostasis(ss)
	}

	// Energy should have changed
	if ss.Homeostasis.Drives[0].Energy == initialEnergy {
		t.Fatal("energy should have changed after 200 ticks")
	}

	// Stats should be computed
	if ss.Homeostasis.AvgEnergy <= 0 {
		t.Fatal("avg energy should be positive")
	}
}

func TestTickHomeostasisNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickHomeostasis(ss) // should not panic
}

func TestDriveTypeName(t *testing.T) {
	if DriveTypeName(DriveEnergy) != "Energie" {
		t.Fatal("expected Energie")
	}
	if DriveTypeName(DriveStress) != "Stress" {
		t.Fatal("expected Stress")
	}
	if DriveTypeName(DriveCuriosity) != "Neugier" {
		t.Fatal("expected Neugier")
	}
	if DriveTypeName(DriveSafety) != "Sicherheit" {
		t.Fatal("expected Sicherheit")
	}
	if DriveTypeName(99) != "?" {
		t.Fatal("expected ?")
	}
}

func TestDetermineDominant(t *testing.T) {
	// Low energy should dominate
	d := &BotDrives{Energy: 0.1, Stress: 0.2, Curiosity: 0.3, Safety: 0.8}
	determineDominant(d)
	if d.DominantDrive != DriveEnergy {
		t.Fatalf("expected DriveEnergy, got %d", d.DominantDrive)
	}

	// High stress should dominate
	d = &BotDrives{Energy: 0.8, Stress: 0.9, Curiosity: 0.3, Safety: 0.7}
	determineDominant(d)
	if d.DominantDrive != DriveStress {
		t.Fatalf("expected DriveStress, got %d", d.DominantDrive)
	}

	// High curiosity should dominate
	d = &BotDrives{Energy: 0.8, Stress: 0.1, Curiosity: 0.9, Safety: 0.8}
	determineDominant(d)
	if d.DominantDrive != DriveCuriosity {
		t.Fatalf("expected DriveCuriosity, got %d", d.DominantDrive)
	}

	// Low safety should dominate
	d = &BotDrives{Energy: 0.8, Stress: 0.1, Curiosity: 0.1, Safety: 0.05}
	determineDominant(d)
	if d.DominantDrive != DriveSafety {
		t.Fatalf("expected DriveSafety, got %d", d.DominantDrive)
	}
}

func TestHomeoAvgEnergy(t *testing.T) {
	if HomeoAvgEnergy(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestHomeoAvgStress(t *testing.T) {
	if HomeoAvgStress(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestHomeoCriticalBots(t *testing.T) {
	if HomeoCriticalBots(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBotDominantDrive(t *testing.T) {
	if BotDominantDrive(nil, 0) != DriveEnergy {
		t.Fatal("nil should return DriveEnergy")
	}
}

func TestBotDriveEnergy(t *testing.T) {
	if BotDriveEnergy(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitHomeostasis(ss)

	e := BotDriveEnergy(ss.Homeostasis, 0)
	if e < 0 || e > 1 {
		t.Fatalf("energy %.2f out of range", e)
	}
}

func TestEnergyDepletion(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitHomeostasis(ss)

	// Run for many ticks without resources nearby
	for i := range ss.Bots {
		ss.Bots[i].NearestPickupDist = 500
		ss.Bots[i].Speed = SwarmBotSpeed
	}

	for tick := 0; tick < 500; tick++ {
		TickHomeostasis(ss)
	}

	// Energy should be very low
	for _, d := range ss.Homeostasis.Drives {
		if d.Energy > 0.5 {
			t.Fatal("energy should have depleted significantly")
		}
	}
}
