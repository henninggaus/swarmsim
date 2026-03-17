package swarm

import (
	"math/rand"
	"testing"
)

func TestInitStigmergy2(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitStigmergy2(ss)

	sg := ss.Stigmergy2
	if sg == nil {
		t.Fatal("stigmergy2 should be initialized")
	}
	if sg.GridW <= 0 || sg.GridH <= 0 {
		t.Fatal("grid dimensions should be positive")
	}
}

func TestClearStigmergy2(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.Stigmergy2On = true
	InitStigmergy2(ss)
	ClearStigmergy2(ss)

	if ss.Stigmergy2 != nil {
		t.Fatal("should be nil")
	}
	if ss.Stigmergy2On {
		t.Fatal("should be false")
	}
}

func TestTickStigmergy2(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitStigmergy2(ss)

	// Place bots in valid positions
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i*30) + 50
		ss.Bots[i].Y = 100
	}

	// Some carrying packages
	for i := 0; i < 5; i++ {
		ss.Bots[i].CarryingPkg = 0
		ss.Bots[i].NearestPickupDist = 50
	}

	for tick := 0; tick < 100; tick++ {
		ss.Tick = tick
		TickStigmergy2(ss)
	}

	sg := ss.Stigmergy2
	if sg.TotalTrails == 0 {
		t.Fatal("should have deposited some trails")
	}
}

func TestTickStigmergy2Nil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickStigmergy2(ss) // should not panic
}

func TestTrailTypeName(t *testing.T) {
	if TrailTypeName(TrailFood) != "Nahrung" {
		t.Fatal("expected Nahrung")
	}
	if TrailTypeName(TrailDanger) != "Gefahr" {
		t.Fatal("expected Gefahr")
	}
	if TrailTypeName(TrailPath) != "Pfad" {
		t.Fatal("expected Pfad")
	}
	if TrailTypeName(TrailHome) != "Heimat" {
		t.Fatal("expected Heimat")
	}
	if TrailTypeName(99) != "?" {
		t.Fatal("expected ?")
	}
}

func TestStig2TrailCount(t *testing.T) {
	if Stig2TrailCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestStig2ActiveCells(t *testing.T) {
	if Stig2ActiveCells(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestStig2CellIdx(t *testing.T) {
	sg := &Stigmergy2State{
		GridW:    10,
		GridH:    10,
		CellSize: 30,
	}

	idx := stig2CellIdx(sg, 45, 60)
	if idx != 2*10+1 { // cx=1, cy=2
		t.Fatalf("expected cell index 21, got %d", idx)
	}

	// Out of bounds
	idx = stig2CellIdx(sg, -100, -100)
	if idx != -1 {
		t.Fatal("negative coords should return -1")
	}
}

func TestTrailDecay(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitStigmergy2(ss)
	sg := ss.Stigmergy2

	// Manually place a weak trail
	idx := stig2CellIdx(sg, 100, 100)
	if idx >= 0 && idx < len(sg.Grid) {
		sg.Grid[idx] = []PheromoneCell{{
			Trails: []CompoundTrail{{
				Type:      TrailFood,
				Intensity: 0.003, // very weak, below decay
				DirX:      1,
				DirY:      0,
			}},
		}}
	}

	ss.Tick = 1
	TickStigmergy2(ss)

	// Trail should have been removed
	if idx >= 0 && sg.Grid[idx] != nil && len(sg.Grid[idx]) > 0 {
		if len(sg.Grid[idx][0].Trails) > 0 {
			t.Fatal("very weak trail should have decayed away")
		}
	}
}
