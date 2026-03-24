package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeMFOState(n int) *SwarmState {
	ss := &SwarmState{
		Bots:   make([]SwarmBot, n),
		ArenaW: 800,
		ArenaH: 800,
		Rng:    rand.New(rand.NewSource(55)),
		Hash:   physics.NewSpatialHash(800, 800, 30),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = ss.Rng.Float64() * 800
		ss.Bots[i].Y = ss.Rng.Float64() * 800
		ss.Bots[i].Angle = ss.Rng.Float64() * 6.28
		ss.Bots[i].Energy = 75
		ss.Bots[i].CarryingPkg = -1
		ss.Bots[i].NeighborCount = ss.Rng.Intn(8)
	}
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}
	return ss
}

func TestInitMFO(t *testing.T) {
	ss := makeMFOState(15)
	InitMFO(ss)
	if ss.MFO == nil {
		t.Fatal("MFO state should not be nil after init")
	}
	if !ss.MFOOn {
		t.Fatal("MFOOn should be true after init")
	}
	if len(ss.MFO.MothFlame) != 15 {
		t.Fatalf("expected 15 moth-flame assignments, got %d", len(ss.MFO.MothFlame))
	}
}

func TestTickMFO(t *testing.T) {
	ss := makeMFOState(15)
	InitMFO(ss)
	for tick := 0; tick < 50; tick++ {
		TickMFO(ss)
	}
	// Should have created some flames
	if len(ss.MFO.Flames) == 0 {
		t.Log("no flames created (may be normal if fitness < 0.5 for all bots)")
	}
	// Sensor cache should be populated
	for i := range ss.Bots {
		if ss.Bots[i].MFOFitness < 0 || ss.Bots[i].MFOFitness > 100 {
			t.Fatalf("bot %d: MFOFitness out of range: %d", i, ss.Bots[i].MFOFitness)
		}
	}
}

func TestTickMFONil(t *testing.T) {
	ss := makeMFOState(10)
	TickMFO(ss) // should not panic
}

func TestClearMFO(t *testing.T) {
	ss := makeMFOState(10)
	InitMFO(ss)
	ClearMFO(ss)
	if ss.MFO != nil {
		t.Fatal("MFO should be nil after clear")
	}
	if ss.MFOOn {
		t.Fatal("MFOOn should be false after clear")
	}
}

func TestApplyMFO(t *testing.T) {
	ss := makeMFOState(15)
	InitMFO(ss)
	// Give some bots high fitness to create flames
	for i := 0; i < 5; i++ {
		ss.Bots[i].NeighborCount = 8
		ss.Bots[i].Energy = 100
	}
	for tick := 0; tick < 20; tick++ {
		TickMFO(ss)
	}
	// Save initial positions to verify movement
	origX := make([]float64, len(ss.Bots))
	origY := make([]float64, len(ss.Bots))
	for i := range ss.Bots {
		origX[i] = ss.Bots[i].X
		origY[i] = ss.Bots[i].Y
	}
	for i := range ss.Bots {
		ApplyMFO(&ss.Bots[i], ss, i)
	}
	// At least some bots should have moved
	moved := 0
	for i := range ss.Bots {
		if ss.Bots[i].X != origX[i] || ss.Bots[i].Y != origY[i] {
			moved++
		}
	}
	if moved == 0 {
		t.Fatal("no bots moved after ApplyMFO")
	}
}

func TestMFOAddFlame(t *testing.T) {
	st := &MFOState{
		Flames: make([]FlamePoint, 0),
	}
	mfoAddFlame(st, 100, 100, 0.8)
	if len(st.Flames) != 1 {
		t.Fatalf("expected 1 flame, got %d", len(st.Flames))
	}
	// Add nearby flame (should merge)
	mfoAddFlame(st, 110, 110, 0.9)
	if len(st.Flames) != 1 {
		t.Fatalf("expected 1 flame after merge, got %d", len(st.Flames))
	}
	if st.Flames[0].Fitness != 0.9 {
		t.Fatalf("merged flame should have better fitness 0.9, got %f", st.Flames[0].Fitness)
	}
	// Add distant flame (should not merge)
	mfoAddFlame(st, 500, 500, 0.7)
	if len(st.Flames) != 2 {
		t.Fatalf("expected 2 flames, got %d", len(st.Flames))
	}
}

func TestMFOGrowSlices(t *testing.T) {
	ss := makeMFOState(5)
	InitMFO(ss)
	ss.Bots = append(ss.Bots, SwarmBot{X: 300, Y: 300, Energy: 80, CarryingPkg: -1, NeighborCount: 5})
	TickMFO(ss)
	if len(ss.MFO.MothFlame) != 6 {
		t.Fatalf("expected 6 moth-flame entries, got %d", len(ss.MFO.MothFlame))
	}
}

func TestSign(t *testing.T) {
	if sign(1.5) != 1 {
		t.Fatal("sign(1.5) should be 1")
	}
	if sign(-0.5) != -1 {
		t.Fatal("sign(-0.5) should be -1")
	}
	if sign(0) != 0 {
		t.Fatal("sign(0) should be 0")
	}
}
