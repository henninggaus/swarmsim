package swarm

import (
	"math/rand"
	"swarmsim/domain/physics"
	"testing"
)

// helper to create a minimal SwarmState for delivery testing
func newDeliveryTestState() *SwarmState {
	return &SwarmState{
		ArenaW: 800,
		ArenaH: 800,
	}
}

// --- Respawn timer tests ---

func TestRespawnTimer(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Stations = []DeliveryStation{
		{X: 100, Y: 100, Color: 1, IsPickup: true, HasPackage: false, RespawnIn: 3},
	}

	// Tick 1: RespawnIn 3 -> 2
	UpdateDeliverySystem(ss)
	if ss.Stations[0].RespawnIn != 2 {
		t.Errorf("expected RespawnIn=2 after 1 tick, got %d", ss.Stations[0].RespawnIn)
	}
	if ss.Stations[0].HasPackage {
		t.Error("should not have package yet")
	}

	// Tick 2: RespawnIn 2 -> 1
	UpdateDeliverySystem(ss)
	if ss.Stations[0].RespawnIn != 1 {
		t.Errorf("expected RespawnIn=1, got %d", ss.Stations[0].RespawnIn)
	}

	// Tick 3: RespawnIn 1 -> 0, should spawn package
	UpdateDeliverySystem(ss)
	if !ss.Stations[0].HasPackage {
		t.Error("station should have package after respawn completes")
	}
	if len(ss.Packages) != 1 {
		t.Fatalf("expected 1 package spawned, got %d", len(ss.Packages))
	}
	pkg := ss.Packages[0]
	if pkg.Color != 1 {
		t.Errorf("package color should match station color (1), got %d", pkg.Color)
	}
	if pkg.CarriedBy != -1 {
		t.Errorf("spawned package should not be carried, got CarriedBy=%d", pkg.CarriedBy)
	}
	if !pkg.Active {
		t.Error("spawned package should be active")
	}
	if pkg.X != 100 || pkg.Y != 100 {
		t.Errorf("package position should match station (100,100), got (%.0f,%.0f)", pkg.X, pkg.Y)
	}
}

func TestRespawnNoEffectOnDropoff(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Stations = []DeliveryStation{
		{X: 200, Y: 200, Color: 2, IsPickup: false, HasPackage: false, RespawnIn: 1},
	}

	// Dropoff stations should not trigger respawn logic
	UpdateDeliverySystem(ss)
	if len(ss.Packages) != 0 {
		t.Error("dropoff station should not spawn packages")
	}
}

func TestRespawnNoEffectWhenHasPackage(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Stations = []DeliveryStation{
		{X: 100, Y: 100, Color: 1, IsPickup: true, HasPackage: true, RespawnIn: 5},
	}

	// Should not decrement timer when station already has a package
	UpdateDeliverySystem(ss)
	if ss.Stations[0].RespawnIn != 5 {
		t.Errorf("timer should not decrease when HasPackage=true, got %d", ss.Stations[0].RespawnIn)
	}
}

// --- Flash timer tests ---

func TestFlashTimerDecrement(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Stations = []DeliveryStation{
		{X: 100, Y: 100, Color: 1, IsPickup: false, FlashTimer: 5},
	}

	UpdateDeliverySystem(ss)
	if ss.Stations[0].FlashTimer != 4 {
		t.Errorf("expected FlashTimer=4, got %d", ss.Stations[0].FlashTimer)
	}
}

func TestFlashTimerStopsAtZero(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Stations = []DeliveryStation{
		{X: 100, Y: 100, Color: 1, IsPickup: false, FlashTimer: 0},
	}

	UpdateDeliverySystem(ss)
	if ss.Stations[0].FlashTimer != 0 {
		t.Errorf("FlashTimer should stay at 0, got %d", ss.Stations[0].FlashTimer)
	}
}

// --- Carried package position update ---

func TestCarriedPackagePosition(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Bots = []SwarmBot{
		{X: 300, Y: 400, CarryingPkg: 0, Speed: 2.0},
	}
	ss.Packages = []DeliveryPackage{
		{Color: 1, CarriedBy: 0, X: 100, Y: 100, Active: true},
	}

	UpdateDeliverySystem(ss)

	// Package position should match bot position
	if ss.Packages[0].X != 300 || ss.Packages[0].Y != 400 {
		t.Errorf("expected package at bot position (300,400), got (%.0f,%.0f)",
			ss.Packages[0].X, ss.Packages[0].Y)
	}
}

func TestCarriedPackageSpeedReduction(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Bots = []SwarmBot{
		{X: 100, Y: 100, CarryingPkg: 0, Speed: 2.0},
	}
	ss.Packages = []DeliveryPackage{
		{Color: 1, CarriedBy: 0, Active: true},
	}

	UpdateDeliverySystem(ss)
	expectedSpeed := 2.0 * 0.7
	if ss.Bots[0].Speed != expectedSpeed {
		t.Errorf("expected speed %.2f after slowdown, got %.2f", expectedSpeed, ss.Bots[0].Speed)
	}
}

func TestNotCarryingNoSpeedChange(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Bots = []SwarmBot{
		{X: 100, Y: 100, CarryingPkg: -1, Speed: 2.0},
	}

	UpdateDeliverySystem(ss)
	if ss.Bots[0].Speed != 2.0 {
		t.Errorf("speed should not change when not carrying, got %.2f", ss.Bots[0].Speed)
	}
}

func TestCarryingZeroSpeedStaysZero(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Bots = []SwarmBot{
		{X: 100, Y: 100, CarryingPkg: 0, Speed: 0},
	}
	ss.Packages = []DeliveryPackage{
		{Color: 1, CarriedBy: 0, Active: true},
	}

	UpdateDeliverySystem(ss)
	if ss.Bots[0].Speed != 0 {
		t.Errorf("zero speed should stay zero, got %.2f", ss.Bots[0].Speed)
	}
}

// --- Score popups ---

func TestScorePopupRiseAndFade(t *testing.T) {
	ss := newDeliveryTestState()
	ss.ScorePopups = []ScorePopup{
		{X: 100, Y: 100, Text: "+10", Timer: 3},
	}

	UpdateDeliverySystem(ss)
	if len(ss.ScorePopups) != 1 {
		t.Fatal("popup should still exist after 1 tick")
	}
	if ss.ScorePopups[0].Y != 99.5 {
		t.Errorf("popup should rise by 0.5: expected Y=99.5, got %.1f", ss.ScorePopups[0].Y)
	}
	if ss.ScorePopups[0].Timer != 2 {
		t.Errorf("timer should be 2, got %d", ss.ScorePopups[0].Timer)
	}

	// Tick until timer reaches 0
	UpdateDeliverySystem(ss) // Timer: 2->1
	UpdateDeliverySystem(ss) // Timer: 1->0, removed

	if len(ss.ScorePopups) != 0 {
		t.Errorf("popup should be removed when timer reaches 0, got %d popups", len(ss.ScorePopups))
	}
}

func TestMultipleScorePopups(t *testing.T) {
	ss := newDeliveryTestState()
	ss.ScorePopups = []ScorePopup{
		{X: 100, Y: 100, Text: "+10", Timer: 1}, // will expire this tick
		{X: 200, Y: 200, Text: "+5", Timer: 5},  // will survive
	}

	UpdateDeliverySystem(ss)
	if len(ss.ScorePopups) != 1 {
		t.Fatalf("expected 1 surviving popup, got %d", len(ss.ScorePopups))
	}
	if ss.ScorePopups[0].Text != "+5" {
		t.Errorf("surviving popup should be '+5', got '%s'", ss.ScorePopups[0].Text)
	}
}

// --- Invalid package index ---

func TestCarryingInvalidPackageIndex(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Bots = []SwarmBot{
		{X: 100, Y: 100, CarryingPkg: 99, Speed: 2.0}, // index out of range
	}
	ss.Packages = []DeliveryPackage{
		{Color: 1, CarriedBy: -1, Active: true},
	}

	// Should not panic; invalid index is >= len(Packages) so the check fails
	UpdateDeliverySystem(ss)
	// Speed should not change since condition bot.CarryingPkg < len(ss.Packages) fails
	if ss.Bots[0].Speed != 2.0 {
		t.Errorf("speed should not change for invalid pkg index, got %.2f", ss.Bots[0].Speed)
	}
}

// --- DeliveryColorName tests ---

func TestDeliveryColorName(t *testing.T) {
	tests := []struct {
		color int
		name  string
	}{
		{1, "Red"},
		{2, "Blue"},
		{3, "Yellow"},
		{4, "Green"},
		{0, "?"},
		{99, "?"},
	}
	for _, tc := range tests {
		got := DeliveryColorName(tc.color)
		if got != tc.name {
			t.Errorf("DeliveryColorName(%d) = %q, want %q", tc.color, got, tc.name)
		}
	}
}

// --- NeighborDelta tests ---

func TestNeighborDeltaNormal(t *testing.T) {
	ss := newDeliveryTestState()
	ss.WrapMode = false
	dx, dy := NeighborDelta(100, 100, 200, 300, ss)
	if dx != 100 || dy != 200 {
		t.Errorf("expected (100,200), got (%.0f,%.0f)", dx, dy)
	}
}

func TestNeighborDeltaWrap(t *testing.T) {
	ss := newDeliveryTestState()
	ss.WrapMode = true
	// Bot A at (700,100), Bot B at (100,100) — dx = -600, but with wrap: -600+800 = 200
	dx, dy := NeighborDelta(700, 100, 100, 100, ss)
	if dx != 200 {
		t.Errorf("expected wrapped dx=200, got %.0f", dx)
	}
	if dy != 0 {
		t.Errorf("expected dy=0, got %.0f", dy)
	}
}

func TestNeighborDeltaWrapPositive(t *testing.T) {
	ss := newDeliveryTestState()
	ss.WrapMode = true
	// Bot A at (100,100), Bot B at (700,100) — dx = 600, but with wrap: 600-800 = -200
	dx, _ := NeighborDelta(100, 100, 700, 100, ss)
	if dx != -200 {
		t.Errorf("expected wrapped dx=-200, got %.0f", dx)
	}
}

// --- AllObstacles tests ---

func TestAllObstacles(t *testing.T) {
	ss := newDeliveryTestState()
	ss.Obstacles = []*physics.Obstacle{
		{X: 10, Y: 10, W: 20, H: 20},
	}
	ss.MazeWalls = []*physics.Obstacle{
		{X: 50, Y: 50, W: 10, H: 100},
		{X: 100, Y: 100, W: 100, H: 10},
	}

	all := ss.AllObstacles()
	if len(all) != 3 {
		t.Errorf("expected 3 total obstacles, got %d", len(all))
	}
}

func TestAllObstaclesEmpty(t *testing.T) {
	ss := newDeliveryTestState()
	all := ss.AllObstacles()
	if len(all) != 0 {
		t.Errorf("expected 0 obstacles, got %d", len(all))
	}
}

// --- NewSwarmState ---

func TestNewSwarmState(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	if ss.BotCount != 20 {
		t.Errorf("expected 20 bots, got %d", ss.BotCount)
	}
	if len(ss.Bots) != 20 {
		t.Errorf("expected 20 bots in slice, got %d", len(ss.Bots))
	}
	if ss.ArenaW != SwarmArenaSize || ss.ArenaH != SwarmArenaSize {
		t.Error("arena should be SwarmArenaSize")
	}
	if ss.Program == nil {
		t.Error("default program should be compiled")
	}
	if ss.Editor == nil {
		t.Error("editor should be initialized")
	}
	if ss.Hash == nil {
		t.Error("spatial hash should be initialized")
	}
	if len(ss.Presets) == 0 {
		t.Error("presets should be populated")
	}
	if ss.SelectedBot != -1 {
		t.Errorf("SelectedBot should be -1, got %d", ss.SelectedBot)
	}
}

func TestNewSwarmStateBotInit(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	for i, bot := range ss.Bots {
		if bot.CarryingPkg != -1 {
			t.Errorf("bot %d: CarryingPkg should be -1", i)
		}
		if bot.FollowTargetIdx != -1 {
			t.Errorf("bot %d: FollowTargetIdx should be -1", i)
		}
		if bot.ObstacleDist != 999 {
			t.Errorf("bot %d: ObstacleDist should be 999", i)
		}
	}
}

// --- RespawnBots ---

func TestRespawnBots(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	ss.Tick = 100
	ss.RespawnBots(30)
	if ss.BotCount != 30 {
		t.Errorf("expected 30 bots, got %d", ss.BotCount)
	}
	if ss.Tick != 0 {
		t.Errorf("tick should be reset to 0, got %d", ss.Tick)
	}
}

func TestRespawnBotsClampsMin(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	ss.RespawnBots(1) // below SwarmMinBots (5)
	if ss.BotCount < SwarmMinBots {
		t.Errorf("bot count should be at least %d, got %d", SwarmMinBots, ss.BotCount)
	}
}

func TestRespawnBotsClampsMax(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	ss.RespawnBots(9999) // above SwarmMaxBots (500)
	if ss.BotCount > SwarmMaxBots {
		t.Errorf("bot count should be at most %d, got %d", SwarmMaxBots, ss.BotCount)
	}
}

// --- ResetBots ---

func TestResetBots(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	// Modify some state
	ss.Tick = 500
	ss.Bots[0].State = 5
	ss.Bots[0].Counter = 10
	ss.Bots[0].CarryingPkg = 2
	ss.ResetBots()
	if ss.Tick != 0 {
		t.Errorf("tick should be 0 after reset, got %d", ss.Tick)
	}
	if ss.Bots[0].State != 0 {
		t.Errorf("state should be 0, got %d", ss.Bots[0].State)
	}
	if ss.Bots[0].Counter != 0 {
		t.Errorf("counter should be 0, got %d", ss.Bots[0].Counter)
	}
	if ss.Bots[0].CarryingPkg != -1 {
		t.Errorf("CarryingPkg should be -1, got %d", ss.Bots[0].CarryingPkg)
	}
}

// --- GenerateSwarmObstacles ---

func TestGenerateSwarmObstacles(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	GenerateSwarmObstacles(ss)
	if len(ss.Obstacles) < 10 || len(ss.Obstacles) > 15 {
		t.Errorf("expected 10-15 obstacles, got %d", len(ss.Obstacles))
	}
}

// --- GenerateSwarmMaze ---

func TestGenerateSwarmMaze(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	GenerateSwarmMaze(ss)
	if len(ss.MazeWalls) == 0 {
		t.Error("maze should generate walls")
	}
	// Should have at least 4 border walls
	if len(ss.MazeWalls) < 4 {
		t.Errorf("expected at least 4 border walls, got %d", len(ss.MazeWalls))
	}
}

// --- GenerateDeliveryStations ---

func TestGenerateDeliveryStations(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	GenerateDeliveryStations(ss)
	if len(ss.Stations) != 8 {
		t.Errorf("expected 8 stations (4 pickup + 4 dropoff), got %d", len(ss.Stations))
	}
	pickups := 0
	dropoffs := 0
	for _, st := range ss.Stations {
		if st.IsPickup {
			pickups++
			if !st.HasPackage {
				t.Error("pickup stations should start with a package")
			}
		} else {
			dropoffs++
		}
	}
	if pickups != 4 {
		t.Errorf("expected 4 pickups, got %d", pickups)
	}
	if dropoffs != 4 {
		t.Errorf("expected 4 dropoffs, got %d", dropoffs)
	}
	if len(ss.Packages) != 4 {
		t.Errorf("expected 4 initial packages, got %d", len(ss.Packages))
	}
}
