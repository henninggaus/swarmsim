package swarm

import (
	"math"
	"swarmsim/domain/physics"
	"testing"
)

// newFlockTestState creates a SwarmState with SpatialHash initialized.
func newFlockTestState(numBots int) *SwarmState {
	ss := newTestSwarmState(numBots)
	ss.Hash = physics.NewSpatialHash(800, 800, 50)
	return ss
}

func TestTickFlockingEmpty(t *testing.T) {
	ss := newFlockTestState(5)
	// Bots all at same position → will be near each other
	// But let's spread them to test "no neighbors" by putting them far apart
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i) * 200
		ss.Bots[i].Y = 400
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}
	TickFlocking(ss)

	for i := range ss.Bots {
		if ss.Bots[i].FlockAlign != 0 {
			t.Fatalf("Bot %d: expected FlockAlign=0 with no neighbors, got %d", i, ss.Bots[i].FlockAlign)
		}
	}
}

func TestTickFlockingAlignment(t *testing.T) {
	ss := newFlockTestState(3)
	// Place bots close together, facing same direction
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*15
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = 0 // all facing right
	}
	// Rebuild spatial hash
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickFlocking(ss)

	// All facing same direction → alignment should be ~0
	for i := range ss.Bots {
		if ss.Bots[i].FlockAlign > 10 || ss.Bots[i].FlockAlign < -10 {
			t.Fatalf("Bot %d: expected FlockAlign ~0 (all aligned), got %d", i, ss.Bots[i].FlockAlign)
		}
	}
}

func TestTickFlockingSeparation(t *testing.T) {
	ss := newFlockTestState(3)
	// Place bots very close together (within separation radius)
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*5
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = 0
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickFlocking(ss)

	// Edge bots (0 and 2) should have non-zero separation
	// Middle bot (1) may cancel out due to symmetry
	anySeparation := false
	for i := range ss.Bots {
		if ss.Bots[i].FlockSeparation > 0 {
			anySeparation = true
		}
	}
	if !anySeparation {
		t.Fatal("Expected at least some bots to have FlockSeparation > 0 when close together")
	}
}

func TestApplyFlock(t *testing.T) {
	ss := newFlockTestState(3)
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*20
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = math.Pi / 4
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	bot := &ss.Bots[0]
	ApplyFlock(bot, ss, 0)

	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("Expected speed %f after FLOCK, got %f", SwarmBotSpeed, bot.Speed)
	}
}

func TestApplyFlockNoNeighbors(t *testing.T) {
	ss := newFlockTestState(1)
	ss.Bots[0].X = 400
	ss.Bots[0].Y = 400
	ss.Hash.Clear()
	ss.Hash.Insert(0, ss.Bots[0].X, ss.Bots[0].Y)

	origAngle := ss.Bots[0].Angle
	ApplyFlock(&ss.Bots[0], ss, 0)

	// No neighbors: angle should stay the same
	if math.Abs(ss.Bots[0].Angle-origAngle) > 0.01 {
		t.Fatalf("Expected angle unchanged with no neighbors, got %f (was %f)", ss.Bots[0].Angle, origAngle)
	}
}

func TestSetRole(t *testing.T) {
	bot := &SwarmBot{}
	SetRole(bot, BotRoleScout)
	if bot.Role != BotRoleScout {
		t.Fatalf("Expected role %d, got %d", BotRoleScout, bot.Role)
	}
	if bot.LEDColor[0] != 0 || bot.LEDColor[1] != 200 || bot.LEDColor[2] != 255 {
		t.Fatalf("Expected cyan LED for scout, got %v", bot.LEDColor)
	}

	SetRole(bot, BotRoleWorker)
	if bot.Role != BotRoleWorker {
		t.Fatalf("Expected role %d, got %d", BotRoleWorker, bot.Role)
	}

	SetRole(bot, BotRoleGuard)
	if bot.Role != BotRoleGuard {
		t.Fatalf("Expected role %d, got %d", BotRoleGuard, bot.Role)
	}
}

func TestTickRoles(t *testing.T) {
	ss := newFlockTestState(10)
	// Assign some roles
	for i := 0; i < 5; i++ {
		ss.Bots[i].Role = BotRoleScout
	}
	for i := 5; i < 10; i++ {
		ss.Bots[i].Role = BotRoleWorker
	}

	TickRoles(ss)

	// All bots should have role_demand set
	for i := range ss.Bots {
		if ss.Bots[i].RoleDemand < 1 || ss.Bots[i].RoleDemand > 3 {
			t.Fatalf("Bot %d: invalid RoleDemand %d", i, ss.Bots[i].RoleDemand)
		}
	}
}

func TestTickRogue(t *testing.T) {
	ss := newFlockTestState(5)
	// Place bots close together
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*10
		ss.Bots[i].Y = 400
		ss.Bots[i].Angle = 0
		ss.Bots[i].Speed = SwarmBotSpeed
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	TickRogue(ss)

	// All bots should have reputation initialized
	for i := range ss.Bots {
		if ss.Bots[i].Reputation == 0 {
			t.Fatalf("Bot %d: reputation should be initialized, got 0", i)
		}
	}
}

func TestFlagRogue(t *testing.T) {
	ss := newFlockTestState(3)
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*10
		ss.Bots[i].Y = 400
		ss.Bots[i].Reputation = 100
	}
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Bot 0 flags nearest neighbor
	FlagRogue(&ss.Bots[0], ss, 0)

	// Nearest to bot 0 is bot 1 (10px away)
	if ss.Bots[1].Reputation >= 100 {
		t.Fatal("Nearest neighbor's reputation should decrease after flagging")
	}
}
