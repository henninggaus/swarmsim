package swarm

import (
	"math"
	"math/rand"
	"swarmsim/domain/physics"
	"testing"
)

// newBatch5TestState creates a SwarmState with SpatialHash and Rng initialized.
func newBatch5TestState(numBots int) *SwarmState {
	ss := newFlockTestState(numBots)
	ss.Rng = rand.New(rand.NewSource(42))
	return ss
}

// --- V-Formation Tests ---

func TestInitClearVFormation(t *testing.T) {
	ss := newBatch5TestState(10)
	InitVFormation(ss)
	if ss.VFormation == nil || !ss.VFormationOn {
		t.Fatal("InitVFormation should set VFormation and VFormationOn")
	}
	if len(ss.VFormation.InFormation) != 10 {
		t.Fatal("InFormation should have length matching bots")
	}
	ClearVFormation(ss)
	if ss.VFormation != nil || ss.VFormationOn {
		t.Fatal("ClearVFormation should nil VFormation and clear VFormationOn")
	}
}

func TestTickVFormationNil(t *testing.T) {
	ss := newBatch5TestState(3)
	TickVFormation(ss) // should not panic
}

func TestVFormationLeaderRotation(t *testing.T) {
	ss := newBatch5TestState(5)
	InitVFormation(ss)

	// Give bot 2 lots of energy
	ss.VFormation.Energy[2] = 1.0

	// Drain leader timer
	ss.VFormation.LeaderTimer = 1
	TickVFormation(ss)

	// Leader should have changed to bot with most energy
	if ss.VFormation.LeaderIdx != 2 {
		t.Fatalf("Leader should rotate to bot 2 (most energy), got %d", ss.VFormation.LeaderIdx)
	}
}

func TestVFormationSensorCache(t *testing.T) {
	ss := newBatch5TestState(5)
	InitVFormation(ss)
	TickVFormation(ss)

	// Leader should have VFormLeader=1
	leader := ss.VFormation.LeaderIdx
	if ss.Bots[leader].VFormLeader != 1 {
		t.Fatal("Leader bot should have VFormLeader=1")
	}

	// Non-leader should have VFormLeader=0
	other := (leader + 1) % len(ss.Bots)
	if ss.Bots[other].VFormLeader != 0 {
		t.Fatal("Non-leader bot should have VFormLeader=0")
	}
}

func TestApplyVFormationNilSafe(t *testing.T) {
	ss := newBatch5TestState(1)
	bot := &ss.Bots[0]
	ApplyVFormation(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil VFormation")
	}
}

func TestVFormationWingPositions(t *testing.T) {
	ss := newBatch5TestState(5)
	InitVFormation(ss)
	TickVFormation(ss)

	// Leader should have FormPos=0
	if ss.VFormation.FormPos[ss.VFormation.LeaderIdx] != 0 {
		t.Fatal("Leader should have FormPos=0")
	}

	// Other bots should have non-zero FormPos
	hasPositive := false
	hasNegative := false
	for i := range ss.Bots {
		if i == ss.VFormation.LeaderIdx {
			continue
		}
		if ss.VFormation.FormPos[i] > 0 {
			hasPositive = true
		}
		if ss.VFormation.FormPos[i] < 0 {
			hasNegative = true
		}
	}
	if !hasPositive || !hasNegative {
		t.Fatal("V-formation should have both positive (right wing) and negative (left wing) positions")
	}
}

// --- Brood Sorting Tests ---

func TestInitClearBrood(t *testing.T) {
	ss := newBatch5TestState(5)
	InitBrood(ss)
	if ss.Brood == nil || !ss.BroodOn {
		t.Fatal("InitBrood should set Brood and BroodOn")
	}
	if len(ss.Brood.Items) != 15 { // 3 items per bot
		t.Fatalf("Expected 15 items, got %d", len(ss.Brood.Items))
	}
	ClearBrood(ss)
	if ss.Brood != nil || ss.BroodOn {
		t.Fatal("ClearBrood should nil Brood and clear BroodOn")
	}
}

func TestTickBroodNil(t *testing.T) {
	ss := newBatch5TestState(3)
	TickBrood(ss) // should not panic
}

func TestBroodItemColors(t *testing.T) {
	ss := newBatch5TestState(5)
	InitBrood(ss)

	colors := map[int]int{}
	for i := range ss.Brood.Items {
		colors[ss.Brood.Items[i].Color]++
	}
	// Should have all 3 colors
	if len(colors) != 3 {
		t.Fatalf("Expected 3 colors, got %d", len(colors))
	}
}

func TestBroodSensorCache(t *testing.T) {
	ss := newBatch5TestState(5)
	InitBrood(ss)

	// Place a bot near an item
	ss.Bots[0].X = ss.Brood.Items[0].X
	ss.Bots[0].Y = ss.Brood.Items[0].Y

	TickBrood(ss)

	// BroodDensity should be > 0 for bot near items
	if ss.Bots[0].BroodDensity < 1 {
		t.Fatal("Bot near items should have BroodDensity > 0")
	}

	// Not carrying initially
	if ss.Bots[0].BroodCarrying != 0 {
		t.Fatal("Bot should not be carrying initially")
	}
}

func TestApplyBroodSortNilSafe(t *testing.T) {
	ss := newBatch5TestState(1)
	bot := &ss.Bots[0]
	ApplyBroodSort(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil Brood")
	}
}

func TestBroodHeldItemFollowsCarrier(t *testing.T) {
	ss := newBatch5TestState(3)
	InitBrood(ss)

	// Force bot 0 to carry item 0
	ss.Brood.Items[0].Held = true
	ss.Brood.Items[0].Holder = 0
	ss.Brood.Carrying[0] = 0

	ss.Bots[0].X = 300
	ss.Bots[0].Y = 400

	TickBrood(ss)

	// Item should follow carrier
	if math.Abs(ss.Brood.Items[0].X-300) > 1 || math.Abs(ss.Brood.Items[0].Y-400) > 1 {
		t.Fatal("Held item should follow carrier position")
	}
}

// --- Jellyfish Pulse Tests ---

func TestInitClearJellyfish(t *testing.T) {
	ss := newBatch5TestState(10)
	InitJellyfish(ss)
	if ss.Jellyfish == nil || !ss.JellyfishOn {
		t.Fatal("InitJellyfish should set Jellyfish and JellyfishOn")
	}
	ClearJellyfish(ss)
	if ss.Jellyfish != nil || ss.JellyfishOn {
		t.Fatal("ClearJellyfish should nil Jellyfish and clear JellyfishOn")
	}
}

func TestTickJellyfishNil(t *testing.T) {
	ss := newBatch5TestState(3)
	TickJellyfish(ss) // should not panic
}

func TestJellyfishPhaseAdvance(t *testing.T) {
	ss := newBatch5TestState(5)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitJellyfish(ss)

	oldPhase := ss.Jellyfish.Phase[0]
	TickJellyfish(ss)

	if ss.Jellyfish.Phase[0] <= oldPhase {
		t.Fatal("Jellyfish phase should advance each tick")
	}
}

func TestJellyfishCenterComputation(t *testing.T) {
	ss := newBatch5TestState(4)
	InitJellyfish(ss)

	// Place bots symmetrically
	ss.Bots[0].X, ss.Bots[0].Y = 100, 100
	ss.Bots[1].X, ss.Bots[1].Y = 300, 100
	ss.Bots[2].X, ss.Bots[2].Y = 100, 300
	ss.Bots[3].X, ss.Bots[3].Y = 300, 300

	TickJellyfish(ss)

	// Center should be ~200, 200
	if math.Abs(ss.Jellyfish.CenterX-200) > 1 || math.Abs(ss.Jellyfish.CenterY-200) > 1 {
		t.Fatalf("Center should be (200,200), got (%.1f, %.1f)", ss.Jellyfish.CenterX, ss.Jellyfish.CenterY)
	}
}

func TestJellyfishSensorCache(t *testing.T) {
	ss := newBatch5TestState(5)
	InitJellyfish(ss)
	TickJellyfish(ss)

	// JellyPhase should be set (0-100)
	if ss.Bots[0].JellyPhase < 0 || ss.Bots[0].JellyPhase > 100 {
		t.Fatalf("JellyPhase should be 0-100, got %d", ss.Bots[0].JellyPhase)
	}

	// JellyExpanding should be 0 or 1
	for i := range ss.Bots {
		if ss.Bots[i].JellyExpanding != 0 && ss.Bots[i].JellyExpanding != 1 {
			t.Fatalf("JellyExpanding should be 0 or 1, got %d", ss.Bots[i].JellyExpanding)
		}
	}
}

func TestApplyJellyfishPulseNilSafe(t *testing.T) {
	ss := newBatch5TestState(1)
	bot := &ss.Bots[0]
	ApplyJellyfishPulse(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil Jellyfish")
	}
}

// --- Immune Swarm Tests ---

func TestInitClearImmuneSwarm(t *testing.T) {
	ss := newBatch5TestState(10)
	InitImmuneSwarm(ss)
	if ss.ImmuneSwarm == nil || !ss.ImmuneSwarmOn {
		t.Fatal("InitImmuneSwarm should set ImmuneSwarm and ImmuneSwarmOn")
	}
	// ~20% antibodies
	abCount := 0
	for _, ab := range ss.ImmuneSwarm.IsAntibody {
		if ab {
			abCount++
		}
	}
	if abCount < 2 {
		t.Fatal("Should have at least 2 antibodies")
	}
	ClearImmuneSwarm(ss)
	if ss.ImmuneSwarm != nil || ss.ImmuneSwarmOn {
		t.Fatal("ClearImmuneSwarm should nil ImmuneSwarm and clear ImmuneSwarmOn")
	}
}

func TestTickImmuneSwarmNil(t *testing.T) {
	ss := newBatch5TestState(3)
	TickImmuneSwarm(ss) // should not panic
}

func TestImmuneSwarmPathogenDetection(t *testing.T) {
	ss := newBatch5TestState(5)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitImmuneSwarm(ss)

	// Place antibody and pathogen close together
	ss.ImmuneSwarm.IsAntibody[0] = true
	ss.ImmuneSwarm.IsPathogen[4] = true
	ss.Bots[0].X, ss.Bots[0].Y = 100, 100
	ss.Bots[4].X, ss.Bots[4].Y = 110, 100

	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickImmuneSwarm(ss)

	// Antibody should get alert
	if ss.ImmuneSwarm.AlertLevel[0] < 0.05 {
		t.Fatal("Antibody near pathogen should have AlertLevel > 0")
	}
}

func TestImmuneSwarmSensorCache(t *testing.T) {
	ss := newBatch5TestState(5)
	InitImmuneSwarm(ss)
	TickImmuneSwarm(ss)

	// Antibody bots should have ImmuneRole=1
	for i := range ss.Bots {
		if i >= len(ss.ImmuneSwarm.IsAntibody) {
			break
		}
		if ss.ImmuneSwarm.IsAntibody[i] && ss.Bots[i].ImmuneRole != 1 {
			t.Fatalf("Antibody bot %d should have ImmuneRole=1, got %d", i, ss.Bots[i].ImmuneRole)
		}
		if ss.ImmuneSwarm.IsPathogen[i] && ss.Bots[i].ImmuneRole != 2 {
			t.Fatalf("Pathogen bot %d should have ImmuneRole=2, got %d", i, ss.Bots[i].ImmuneRole)
		}
	}
}

func TestApplyImmuneSwarmNilSafe(t *testing.T) {
	ss := newBatch5TestState(1)
	bot := &ss.Bots[0]
	ApplyImmuneSwarm(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("Expected SwarmBotSpeed with nil ImmuneSwarm")
	}
}

func TestImmuneSwarmNeutralization(t *testing.T) {
	ss := newBatch5TestState(5)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	InitImmuneSwarm(ss)

	// Set up: 2 antibodies near 1 pathogen
	for i := range ss.ImmuneSwarm.IsAntibody {
		ss.ImmuneSwarm.IsAntibody[i] = false
		ss.ImmuneSwarm.IsPathogen[i] = false
	}
	ss.ImmuneSwarm.IsAntibody[0] = true
	ss.ImmuneSwarm.IsAntibody[1] = true
	ss.ImmuneSwarm.IsPathogen[2] = true

	// Place them very close
	ss.Bots[0].X, ss.Bots[0].Y = 100, 100
	ss.Bots[1].X, ss.Bots[1].Y = 105, 100
	ss.Bots[2].X, ss.Bots[2].Y = 102, 100

	// Run enough ticks to neutralize (30 ticks)
	for tick := 0; tick < 35; tick++ {
		ss.Hash.Clear()
		for i := range ss.Bots {
			ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
		}
		TickImmuneSwarm(ss)
	}

	// Pathogen should be neutralized
	if ss.ImmuneSwarm.IsPathogen[2] {
		t.Fatal("Pathogen should be neutralized after 30+ ticks with 2 antibodies nearby")
	}
}
