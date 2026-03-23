package swarm

import (
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makeTestSwarmStateBat(n int) *SwarmState {
	ss := &SwarmState{
		Bots:   make([]SwarmBot, n),
		ArenaW: 800,
		ArenaH: 800,
		Rng:    rand.New(rand.NewSource(42)),
		Hash:   physics.NewSpatialHash(800, 800, 60),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = ss.Rng.Float64() * ss.ArenaW
		ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
		ss.Bots[i].Speed = SwarmBotSpeed
		ss.Bots[i].CarryingPkg = -1
	}
	return ss
}

func TestBatInitClear(t *testing.T) {
	ss := makeTestSwarmStateBat(20)
	InitBat(ss)

	if ss.Bat == nil {
		t.Fatal("BatState should not be nil after init")
	}
	if !ss.BatOn {
		t.Fatal("BatOn should be true after init")
	}
	if len(ss.Bat.Freq) != 20 {
		t.Fatalf("expected 20 frequencies, got %d", len(ss.Bat.Freq))
	}
	if len(ss.Bat.Loud) != 20 {
		t.Fatalf("expected 20 loudness values, got %d", len(ss.Bat.Loud))
	}

	// Verify initial loudness = 1.0 and pulse = 0.0
	for i := range ss.Bat.Loud {
		if ss.Bat.Loud[i] != 1.0 {
			t.Errorf("bat %d: expected initial loudness 1.0, got %f", i, ss.Bat.Loud[i])
		}
		if ss.Bat.Pulse[i] != 0.0 {
			t.Errorf("bat %d: expected initial pulse 0.0, got %f", i, ss.Bat.Pulse[i])
		}
	}

	ClearBat(ss)
	if ss.Bat != nil {
		t.Fatal("BatState should be nil after clear")
	}
	if ss.BatOn {
		t.Fatal("BatOn should be false after clear")
	}
}

func TestBatTickConvergence(t *testing.T) {
	ss := makeTestSwarmStateBat(30)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400
	InitBat(ss)

	// Run 200 ticks
	for tick := 0; tick < 200; tick++ {
		TickBat(ss)
		for i := range ss.Bots {
			ApplyBat(&ss.Bots[i], ss, i)
		}
	}

	// After 200 ticks, loudness should have decreased for most bats
	lowLoud := 0
	for _, l := range ss.Bat.Loud {
		if l < 0.5 {
			lowLoud++
		}
	}
	if lowLoud == 0 {
		t.Error("expected at least some bats with decreased loudness after convergence")
	}

	// Pulse rate should have increased for some bats
	highPulse := 0
	for _, p := range ss.Bat.Pulse {
		if p > 0.1 {
			highPulse++
		}
	}
	if highPulse == 0 {
		t.Error("expected at least some bats with increased pulse rate")
	}
}

func TestBatSensorCache(t *testing.T) {
	ss := makeTestSwarmStateBat(10)
	InitBat(ss)

	TickBat(ss)

	// All bots should have sensor values populated
	for i := range ss.Bots {
		if ss.Bots[i].BatLoud < 0 || ss.Bots[i].BatLoud > 100 {
			t.Errorf("bat %d: BatLoud out of range: %d", i, ss.Bots[i].BatLoud)
		}
		if ss.Bots[i].BatPulse < 0 || ss.Bots[i].BatPulse > 100 {
			t.Errorf("bat %d: BatPulse out of range: %d", i, ss.Bots[i].BatPulse)
		}
	}
}

func TestBatAvgLoudPrecomputed(t *testing.T) {
	ss := makeTestSwarmStateBat(50)
	InitBat(ss)

	// After init, AvgLoud should be 1.0 (all bats start at loudness 1.0)
	if ss.Bat.AvgLoud != 1.0 {
		t.Errorf("expected initial AvgLoud 1.0, got %f", ss.Bat.AvgLoud)
	}

	// Run some ticks with apply
	for tick := 0; tick < 50; tick++ {
		TickBat(ss)
		for i := range ss.Bots {
			ApplyBat(&ss.Bots[i], ss, i)
		}
	}

	// AvgLoud is computed in TickBat BEFORE ApplyBat modifies Loud values.
	// After ApplyBat, individual Loud values may have changed.
	// Verify AvgLoud matches manual avg right after TickBat (before apply).
	TickBat(ss)
	manualAvg := 0.0
	for _, l := range ss.Bat.Loud {
		manualAvg += l
	}
	manualAvg /= float64(len(ss.Bat.Loud))

	diff := ss.Bat.AvgLoud - manualAvg
	if diff < -0.001 || diff > 0.001 {
		t.Errorf("AvgLoud (%f) doesn't match manual avg (%f) after TickBat", ss.Bat.AvgLoud, manualAvg)
	}
}

func TestBatPersonalBest(t *testing.T) {
	ss := makeTestSwarmStateBat(20)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400
	InitBat(ss)

	// Personal best slices should be initialized
	if len(ss.Bat.PBestX) != 20 {
		t.Fatalf("expected 20 PBestX, got %d", len(ss.Bat.PBestX))
	}
	if len(ss.Bat.PBestF) != 20 {
		t.Fatalf("expected 20 PBestF, got %d", len(ss.Bat.PBestF))
	}

	// Run ticks; personal best fitness should increase or stay the same
	for tick := 0; tick < 100; tick++ {
		TickBat(ss)
		for i := range ss.Bots {
			ApplyBat(&ss.Bots[i], ss, i)
		}
	}

	// At least some bats should have updated their personal best
	improved := 0
	for i := range ss.Bat.PBestF {
		if ss.Bat.PBestF[i] > -1e18 {
			improved++
		}
	}
	if improved == 0 {
		t.Error("expected at least some bats to have updated personal bests")
	}
}

func TestBatDynamicGrow(t *testing.T) {
	ss := makeTestSwarmStateBat(10)
	InitBat(ss)

	// Add bots dynamically
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * ss.ArenaW,
			Y: ss.Rng.Float64() * ss.ArenaH,
		})
	}

	// TickBat should grow all slices including PBest
	TickBat(ss)

	if len(ss.Bat.PBestX) != 15 {
		t.Errorf("expected PBestX grown to 15, got %d", len(ss.Bat.PBestX))
	}
	if len(ss.Bat.PBestY) != 15 {
		t.Errorf("expected PBestY grown to 15, got %d", len(ss.Bat.PBestY))
	}
	if len(ss.Bat.PBestF) != 15 {
		t.Errorf("expected PBestF grown to 15, got %d", len(ss.Bat.PBestF))
	}
}
