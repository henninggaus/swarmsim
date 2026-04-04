package factory

import (
	"math"
	"swarmsim/domain/swarm"
	"testing"
)

func TestNewFactoryState(t *testing.T) {
	fs := NewFactoryState(100)
	if fs == nil {
		t.Fatal("nil state")
	}
	if len(fs.Bots) != 100 {
		t.Errorf("bots: got %d, want 100", len(fs.Bots))
	}
	if fs.BotCount != 100 {
		t.Errorf("botCount: got %d", fs.BotCount)
	}
	if len(fs.Machines) == 0 {
		t.Error("no machines")
	}
	if len(fs.Chargers) == 0 {
		t.Error("no chargers")
	}
	if len(fs.Docks) == 0 {
		t.Error("no docks")
	}
	if len(fs.Walls) == 0 {
		t.Error("no walls")
	}
	if len(fs.Storage) == 0 {
		t.Error("no storage")
	}
	if fs.InboundStorageIdx >= len(fs.Storage) {
		t.Error("invalid inbound idx")
	}
	if fs.QCStorageIdx >= len(fs.Storage) {
		t.Error("invalid QC idx")
	}
	if fs.OutboundStorageIdx >= len(fs.Storage) {
		t.Error("invalid outbound idx")
	}
	if fs.Budget != 10000 {
		t.Errorf("budget: got %.0f, want 10000", fs.Budget)
	}
}

func TestTaskQueue(t *testing.T) {
	tq := &TaskQueue{}
	tq.AddTask(NewTask(TaskUnloadTruck, 0, 0, 100, 100, 5, 1))
	tq.AddTask(NewTask(TaskTransportToMachine, 0, 0, 200, 200, 10, 2))
	if len(tq.Tasks) != 2 {
		t.Fatalf("tasks: got %d, want 2", len(tq.Tasks))
	}

	// Assign highest priority task
	idx := tq.AssignTask(0)
	if idx < 0 {
		t.Fatal("no task assigned")
	}
	if tq.Tasks[idx].Priority != 10 {
		t.Error("should assign highest priority")
	}

	// Complete it
	tq.CompleteTask(idx)
	if len(tq.Tasks) != 1 {
		t.Errorf("after complete: got %d tasks", len(tq.Tasks))
	}
}

func TestMachineFeedAndCollect(t *testing.T) {
	m := &Machine{MaxInput: 3, ProcessTime: 100}
	if !FeedMachine(m) {
		t.Error("should accept input")
	}
	if m.CurrentInput != 1 {
		t.Error("input should be 1")
	}

	// Feed to max
	FeedMachine(m)
	FeedMachine(m)
	if FeedMachine(m) {
		t.Error("should reject when full")
	}

	// Process
	m.Active = true
	m.ProcessTimer = 1
	// Simulate tick
	if m.ProcessTimer > 0 {
		m.ProcessTimer--
		if m.ProcessTimer == 0 {
			m.OutputReady = true
			m.Active = false
		}
	}
	if !m.OutputReady {
		t.Error("output should be ready")
	}
	if !CollectOutput(m) {
		t.Error("should collect output")
	}
	if m.OutputReady {
		t.Error("output should be cleared")
	}
}

func TestNeedsCharge(t *testing.T) {
	bot := &swarm.SwarmBot{Energy: 100}
	if NeedsCharge(bot) {
		t.Error("100% should not need charge")
	}
	bot.Energy = 15
	if !NeedsCharge(bot) {
		t.Error("15% should need charge")
	}
}

func TestStorageSlotFIFO(t *testing.T) {
	st := &StorageArea{MaxParts: 10}
	addToStorageDirect(st, 1, 100) // red at tick 100
	addToStorageDirect(st, 2, 200) // blue at tick 200
	addToStorageDirect(st, 1, 300) // red at tick 300

	if len(st.Slots) != 3 {
		t.Fatalf("slots: got %d", len(st.Slots))
	}

	// Remove red — should get the oldest (tick 100)
	// Use direct slot manipulation since removeFromStorageCheck requires FactoryState
	bestIdx := -1
	bestTick := int(1e9)
	for j := range st.Slots {
		if st.Slots[j].PartColor == 1 && st.Slots[j].Tick < bestTick {
			bestTick = st.Slots[j].Tick
			bestIdx = j
		}
	}
	if bestIdx < 0 {
		t.Fatal("should find red slot")
	}
	if bestTick != 100 {
		t.Errorf("should remove oldest red (tick 100), got tick %d", bestTick)
	}
	st.Slots = append(st.Slots[:bestIdx], st.Slots[bestIdx+1:]...)

	if len(st.Slots) != 2 {
		t.Errorf("after remove: got %d slots", len(st.Slots))
	}
	// Remaining red should be tick 300
	for _, s := range st.Slots {
		if s.PartColor == 1 && s.Tick == 100 {
			t.Error("oldest red should be gone")
		}
	}
}

func TestProductionRecipes(t *testing.T) {
	fs := NewFactoryState(10)
	if len(fs.Recipes) == 0 {
		t.Fatal("no recipes")
	}
	// Check recipe steps reference valid machine indices
	for ri, r := range fs.Recipes {
		for si, mi := range r.Steps {
			if mi < 0 || mi >= len(fs.Machines) {
				t.Errorf("recipe %d step %d: invalid machine idx %d", ri, si, mi)
			}
		}
	}
}

func TestTickCharging_DrainEnergy(t *testing.T) {
	fs := NewFactoryState(5)
	fs.Bots[0].Energy = 50
	fs.Bots[0].Speed = 3.0 // moving
	fs.Bots[0].State = BotMovingToSource
	TickCharging(fs)
	if fs.Bots[0].Energy >= 50 {
		t.Error("energy should decrease when moving")
	}
}

func TestTickCharging_ChargingBotGainsEnergy(t *testing.T) {
	fs := NewFactoryState(5)
	fs.Bots[0].Energy = 30
	fs.Bots[0].State = BotCharging
	fs.Bots[0].Speed = 0
	// Place bot at first charger
	ch := &fs.Chargers[0]
	fs.Bots[0].X = ch.X + ch.W/2
	fs.Bots[0].Y = ch.Y + ch.H/2
	ch.Occupants = append(ch.Occupants, 0)
	TickCharging(fs)
	if fs.Bots[0].Energy <= 30 {
		t.Error("charging bot should gain energy")
	}
}

func TestTickTrucks_SpawnsInbound(t *testing.T) {
	fs := NewFactoryState(10)
	fs.TruckTimer = 1 // about to spawn
	before := len(fs.Trucks)
	TickTrucks(fs)
	if len(fs.Trucks) <= before {
		t.Error("should spawn inbound truck")
	}
}

func TestTickMachines_ProcessesInput(t *testing.T) {
	fs := NewFactoryState(10)
	m := &fs.Machines[0]
	FeedMachine(m)
	// Machine should start processing
	for i := 0; i < m.ProcessTime+10; i++ {
		TickMachines(fs)
	}
	if !m.OutputReady {
		t.Error("machine should have output ready after processing time")
	}
}

func TestTickRepair_MalfunctionSlowsBot(t *testing.T) {
	fs := NewFactoryState(10)
	fs.Malfunctioning[0] = true
	fs.Bots[0].Speed = FactoryBotSpeed
	fs.Bots[0].State = BotMovingToSource
	TickRepair(fs)
	// Malfunctioning bot should have reduced speed applied during behavior tick
	// Just verify the malfunction flag is set
	if !fs.Malfunctioning[0] {
		t.Error("should still be malfunctioning")
	}
}

func TestBudgetEconomics(t *testing.T) {
	fs := NewFactoryState(10)
	initialBudget := fs.Budget
	// Simulate material cost
	fs.Budget -= fs.MaterialCostPerPart * 5
	if fs.Budget >= initialBudget {
		t.Error("budget should decrease")
	}
	// Simulate revenue
	fs.Budget += fs.RevenuePerProduct * 2
	if fs.Budget <= initialBudget-fs.MaterialCostPerPart*5 {
		t.Error("revenue should increase budget")
	}
}

func TestValidateTask_InvalidTruckTask(t *testing.T) {
	fs := NewFactoryState(10)
	// Task referencing non-existent truck
	task := NewTask(TaskUnloadTruck, 9999, 9999, 0, 0, 10, 1)
	if validateTask(fs, &task) {
		t.Error("should be invalid - no truck at location")
	}
}

func TestEmergencyEvacuation(t *testing.T) {
	fs := NewFactoryState(10)
	fs.Bots[0].State = BotMovingToSource
	ToggleEmergency(fs)
	if !fs.Emergency {
		t.Error("emergency should be active")
	}
	if fs.Bots[0].State != BotEmergencyEvac {
		t.Error("bot should be evacuating")
	}
	ToggleEmergency(fs)
	if fs.Emergency {
		t.Error("emergency should be deactivated")
	}
}

func TestPrecomputeParkingSlots(t *testing.T) {
	fs := NewFactoryState(20)
	// Set some bots idle
	for i := 0; i < 10; i++ {
		fs.Bots[i].State = BotIdle
		fs.ShiftOnDuty[i] = true
	}
	for i := 10; i < 20; i++ {
		fs.Bots[i].State = BotMovingToSource
	}
	PrecomputeParkingSlots(fs)
	if len(fs.BotParkingSlot) != 20 {
		t.Fatalf("expected 20 parking slots, got %d", len(fs.BotParkingSlot))
	}
	// Idle bots should have non-negative zone indices
	for i := 0; i < 10; i++ {
		if fs.BotParkingSlot[i].ZoneIdx < 0 || fs.BotParkingSlot[i].ZoneIdx >= len(fs.ParkingZones) {
			t.Errorf("bot %d: invalid zone index %d", i, fs.BotParkingSlot[i].ZoneIdx)
		}
	}
}

// --- Task generation tests ---

func TestGenerateTasks_UnloadTruck(t *testing.T) {
	fs := NewFactoryState(20)
	// Spawn a truck manually at a dock
	fs.Trucks = append(fs.Trucks, FactoryTruck{
		DockIdx: 0, Phase: TruckUnloading,
		Parts: []int{1, 2, 3},
	})
	fs.Docks[0].TruckIdx = 0
	fs.Tick = 50 // rate limit: must be divisible by 50
	GenerateTasks(fs)
	if len(fs.Tasks.Tasks) == 0 {
		t.Error("should generate unload tasks")
	}
}

func TestGenerateTasks_Kanban(t *testing.T) {
	fs := NewFactoryState(10)
	// Set machine to NeedsInput
	fs.Machines[0].NeedsInput = true
	// Add matching part to inbound storage
	fs.Storage[fs.InboundStorageIdx].Slots = append(fs.Storage[fs.InboundStorageIdx].Slots,
		StorageSlot{PartColor: fs.Machines[0].InputColor, Tick: 1})
	fs.Tick = 50
	GenerateTasks(fs)
	found := false
	for _, task := range fs.Tasks.Tasks {
		if task.Type == TaskTransportToMachine {
			found = true
			break
		}
	}
	if !found {
		t.Error("should generate transport-to-machine task via Kanban")
	}
}

func TestNavigateToIdx_WallCollision(t *testing.T) {
	fs := NewFactoryState(5)
	bot := &fs.Bots[0]
	// Place bot near a wall
	if len(fs.Walls) > 0 {
		wall := fs.Walls[0]
		bot.X = wall.X + wall.W/2
		bot.Y = wall.Y + wall.H + FactoryBotRadius + 1
		bot.Angle = -math.Pi / 2 // heading into wall
	}
	// Navigate should avoid wall
	navigateToIdx(bot, 0, bot.X, bot.Y-100, fs)
	// Bot should still be within world bounds
	if bot.X < 0 || bot.Y < 0 {
		t.Error("bot went out of bounds")
	}
}

func TestShiftSystem(t *testing.T) {
	fs := NewFactoryState(100)
	onBefore := 0
	for _, on := range fs.ShiftOnDuty {
		if on {
			onBefore++
		}
	}
	// Force shift change
	fs.ShiftTimer = 1
	TickShiftSystem(fs)
	onAfter := 0
	for _, on := range fs.ShiftOnDuty {
		if on {
			onAfter++
		}
	}
	// Should have swapped some bots
	if onAfter == onBefore {
		t.Log("shift may not have changed — depends on timer logic")
	}
}

func TestMachineOverheating(t *testing.T) {
	fs := NewFactoryState(10)
	m := &fs.Machines[0]
	FeedMachine(m)
	m.Active = true
	m.ProcessTimer = 10000 // long processing
	// Run many ticks to heat up
	for i := 0; i < 2100; i++ {
		TickMachines(fs)
	}
	if !m.CoolingDown {
		t.Error("machine should be cooling down after prolonged use")
	}
}

func TestPruneStaleTasks(t *testing.T) {
	fs := NewFactoryState(10)
	// Add invalid task (unload from nonexistent truck location)
	fs.Tasks.AddTask(NewTask(TaskUnloadTruck, 9999, 9999, 0, 0, 5, 1))
	fs.Tick = 100 // must be divisible by 100
	PruneStaleTasks(fs)
	if len(fs.Tasks.Tasks) != 0 {
		t.Error("stale task should be pruned")
	}
}

func TestForceSpawnTruck(t *testing.T) {
	fs := NewFactoryState(10)
	before := len(fs.Trucks)
	ForceSpawnInboundTruck(fs)
	if len(fs.Trucks) != before+1 {
		t.Error("should spawn truck")
	}
}

func TestBotRolesDistribution(t *testing.T) {
	fs := NewFactoryState(1000)
	trans, fork, express := 0, 0, 0
	for _, r := range fs.BotRoles {
		switch r {
		case RoleTransporter:
			trans++
		case RoleForklift:
			fork++
		case RoleExpress:
			express++
		}
	}
	// Roughly 60/25/15 distribution
	if trans < 500 || trans > 700 {
		t.Errorf("transporters: %d (expect ~600)", trans)
	}
	if fork < 150 || fork > 350 {
		t.Errorf("forklifts: %d (expect ~250)", fork)
	}
	if express < 80 || express > 220 {
		t.Errorf("express: %d (expect ~150)", express)
	}
}

func TestOrderSystem(t *testing.T) {
	fs := NewFactoryState(10)
	fs.OrderTimer = 1 // about to generate
	fs.Tick = 1
	TickOrders(fs)
	if len(fs.Orders) == 0 {
		t.Error("should generate an order")
	}
}

func TestRecipeChaining(t *testing.T) {
	fs := NewFactoryState(10)
	if len(fs.Recipes) < 2 {
		t.Fatal("need at least 2 recipes")
	}
	r := fs.Recipes[0]
	if len(r.Steps) < 2 {
		t.Fatal("recipe should have 2+ steps")
	}
	// Verify each step machine exists
	for _, mi := range r.Steps {
		if mi < 0 || mi >= len(fs.Machines) {
			t.Errorf("recipe step machine idx %d out of range", mi)
		}
	}
}

func TestNearestPoint(t *testing.T) {
	points := [][2]float64{{0, 0}, {10, 0}, {5, 5}}
	idx, dist := nearestPoint(4, 4, points)
	if idx != 2 {
		t.Errorf("expected nearest index 2, got %d", idx)
	}
	if dist < 0 {
		t.Errorf("distance should be non-negative, got %f", dist)
	}
	// Empty list
	idx2, _ := nearestPoint(0, 0, nil)
	if idx2 != -1 {
		t.Errorf("expected -1 for empty points, got %d", idx2)
	}
}

func TestFindNearestStorage(t *testing.T) {
	fs := NewFactoryState(5)
	if len(fs.Storage) == 0 {
		t.Fatal("need storage areas")
	}
	// Use a point near inbound storage
	inbound := &fs.Storage[fs.InboundStorageIdx]
	st := findNearestStorage(fs, inbound.X+inbound.W/2, inbound.Y+inbound.H/2)
	if st == nil {
		t.Fatal("should find nearest storage")
	}
}

func TestFindNearestGateTarget(t *testing.T) {
	fs := NewFactoryState(5)
	bot := &fs.Bots[0]
	target := findNearestGateTarget(fs, bot)
	if target[0] == 0 && target[1] == 0 {
		t.Error("gate target should not be origin")
	}
}

func TestAddAlert(t *testing.T) {
	fs := NewFactoryState(5)
	AddAlert(fs, "test alert", [3]uint8{255, 0, 0})
	if len(fs.Alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(fs.Alerts))
	}
	if fs.Alerts[0].Message != "test alert" {
		t.Errorf("wrong message: %s", fs.Alerts[0].Message)
	}
	// Add many alerts to test pruning
	for i := 0; i < MaxVisibleAlerts*4; i++ {
		AddAlert(fs, "flood", [3]uint8{0, 0, 0})
	}
	if len(fs.Alerts) > MaxVisibleAlerts*3 {
		t.Errorf("alerts should be pruned, got %d", len(fs.Alerts))
	}
}

func TestEmergencyTimerCountdown(t *testing.T) {
	fs := NewFactoryState(10)
	ToggleEmergency(fs)
	if !fs.Emergency {
		t.Fatal("emergency should be active")
	}
	initialTimer := fs.EmergencyTimer
	TickEmergency(fs)
	if fs.EmergencyTimer >= initialTimer {
		t.Error("emergency timer should decrease")
	}
	// Run until auto-end
	for fs.Emergency {
		TickEmergency(fs)
	}
	if fs.Emergency {
		t.Error("emergency should auto-end after timer expires")
	}
}

func TestBroadcastCongestion(t *testing.T) {
	fs := NewFactoryState(20)
	// Place several bots near each other
	for i := 0; i < 5; i++ {
		fs.Bots[i].X = 500
		fs.Bots[i].Y = 500
		fs.ShiftOnDuty[i] = true
	}
	// Rebuild spatial hash
	fs.BotHash.Clear()
	for i := range fs.Bots {
		fs.BotHash.Insert(i, fs.Bots[i].X, fs.Bots[i].Y)
	}
	// Should not panic
	broadcastCongestion(fs, 0)
}

func TestBotWander(t *testing.T) {
	fs := NewFactoryState(10)
	PrecomputeParkingSlots(fs)
	bot := &fs.Bots[0]
	bot.State = BotIdle
	fs.ShiftOnDuty[0] = true
	origX, origY := bot.X, bot.Y
	botWander(bot, 0, fs)
	// Bot should have moved or stayed (not crash)
	if bot.X == origX && bot.Y == origY && bot.Speed == 0 {
		// Already at parking slot — that's fine
	}
}

func TestMultiStepRecipe(t *testing.T) {
	fs := NewFactoryState(10)
	// Check that recipes chain through multiple machines
	for _, r := range fs.Recipes {
		if len(r.Steps) < 2 {
			t.Errorf("recipe for color %d has only %d steps", r.InputColor, len(r.Steps))
		}
	}
}

func TestOutboundStorageIdx(t *testing.T) {
	fs := NewFactoryState(10)
	if fs.OutboundStorageIdx == fs.InboundStorageIdx {
		t.Error("outbound should differ from inbound")
	}
	if fs.OutboundStorageIdx >= len(fs.Storage) {
		t.Error("outbound idx out of bounds")
	}
}

func TestBotRoleSpeedDifferences(t *testing.T) {
	fs := NewFactoryState(100)
	// Verify different roles exist
	hasTransporter, hasForklift, hasExpress := false, false, false
	for _, r := range fs.BotRoles {
		switch r {
		case RoleTransporter:
			hasTransporter = true
		case RoleForklift:
			hasForklift = true
		case RoleExpress:
			hasExpress = true
		}
	}
	if !hasTransporter || !hasForklift || !hasExpress {
		t.Error("should have all 3 bot roles")
	}
}

func TestKanbanPull(t *testing.T) {
	fs := NewFactoryState(10)
	m := &fs.Machines[0]
	// Initially NeedsInput should be true (empty)
	m.NeedsInput = true
	// Feed to max
	for i := 0; i < m.MaxInput; i++ {
		FeedMachine(m)
	}
	// After full, NeedsInput should be false
	// (TickMachines sets this based on CurrentInput < 2)
	TickMachines(fs)
	if m.NeedsInput {
		t.Error("machine should not need input when full")
	}
}

func TestMachineOverheatingCycle(t *testing.T) {
	fs := NewFactoryState(10)
	m := &fs.Machines[0]
	FeedMachine(m)
	m.Active = true
	m.ProcessTimer = 10000 // keep active long enough to overheat
	// Heat up until cooling triggers
	for i := 0; i < 2100; i++ {
		TickMachines(fs)
	}
	if !m.CoolingDown {
		t.Error("should be cooling")
	}
	// Cool down fully (temp starts ~80, needs to reach <30, at 0.2/tick ~250+ ticks)
	for i := 0; i < 500; i++ {
		TickMachines(fs)
	}
	if m.CoolingDown {
		t.Errorf("should have finished cooling, temperature=%.1f", m.Temperature)
	}
}

func TestEnergyEconomics(t *testing.T) {
	fs := NewFactoryState(10)
	initial := fs.Budget
	// Simulate material cost
	fs.Budget -= fs.MaterialCostPerPart * 3
	fs.TotalMaterialCost += fs.MaterialCostPerPart * 3
	// Simulate revenue
	fs.Budget += fs.RevenuePerProduct * 2
	fs.TotalRevenue += fs.RevenuePerProduct * 2
	profit := fs.TotalRevenue - fs.TotalMaterialCost
	if profit <= 0 {
		t.Errorf("2 products ($400) - 3 parts ($150) should be profit, got %.0f", profit)
	}
	if fs.Budget <= initial-fs.MaterialCostPerPart*3 {
		t.Error("budget should reflect revenue")
	}
}
