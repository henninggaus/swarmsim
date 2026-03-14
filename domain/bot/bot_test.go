package bot

import (
	"math"
	"swarmsim/domain/genetics"
	"swarmsim/domain/resource"
	"testing"
)

// --- Vec2 tests ---

func TestVec2Add(t *testing.T) {
	v := Vec2{1, 2}.Add(Vec2{3, 4})
	if v.X != 4 || v.Y != 6 {
		t.Errorf("expected (4,6), got (%v,%v)", v.X, v.Y)
	}
}

func TestVec2Sub(t *testing.T) {
	v := Vec2{5, 7}.Sub(Vec2{2, 3})
	if v.X != 3 || v.Y != 4 {
		t.Errorf("expected (3,4), got (%v,%v)", v.X, v.Y)
	}
}

func TestVec2Scale(t *testing.T) {
	v := Vec2{3, 4}.Scale(2)
	if v.X != 6 || v.Y != 8 {
		t.Errorf("expected (6,8), got (%v,%v)", v.X, v.Y)
	}
}

func TestVec2Len(t *testing.T) {
	l := Vec2{3, 4}.Len()
	if math.Abs(l-5) > 1e-9 {
		t.Errorf("expected 5, got %v", l)
	}
}

func TestVec2Dist(t *testing.T) {
	d := Vec2{0, 0}.Dist(Vec2{3, 4})
	if math.Abs(d-5) > 1e-9 {
		t.Errorf("expected 5, got %v", d)
	}
}

func TestVec2Normalized(t *testing.T) {
	n := Vec2{3, 4}.Normalized()
	if math.Abs(n.X-0.6) > 1e-9 || math.Abs(n.Y-0.8) > 1e-9 {
		t.Errorf("expected (0.6,0.8), got (%v,%v)", n.X, n.Y)
	}
}

func TestVec2NormalizedZero(t *testing.T) {
	n := Vec2{0, 0}.Normalized()
	if n.X != 0 || n.Y != 0 {
		t.Error("normalized zero vector should be zero")
	}
}

// --- BotType tests ---

func TestBotTypeString(t *testing.T) {
	tests := []struct {
		typ    BotType
		expect string
	}{
		{TypeScout, "Scout"},
		{TypeWorker, "Worker"},
		{TypeLeader, "Leader"},
		{TypeTank, "Tank"},
		{TypeHealer, "Healer"},
		{BotType(99), "Unknown"},
	}
	for _, tc := range tests {
		if tc.typ.String() != tc.expect {
			t.Errorf("BotType(%d).String() = %q, want %q", tc.typ, tc.typ.String(), tc.expect)
		}
	}
}

// --- BotState tests ---

func TestBotStateString(t *testing.T) {
	tests := []struct {
		state  BotState
		expect string
	}{
		{StateIdle, "Idle"},
		{StateFlocking, "Flocking"},
		{StateForaging, "Foraging"},
		{StateReturning, "Returning"},
		{StateFormation, "Formation"},
		{StateRepairing, "Repairing"},
		{StatePushing, "Pushing"},
		{StateScouting, "Scouting"},
		{StateNoEnergy, "No Energy"},
		{StateCooperating, "Cooperating"},
		{StateLifting, "Lifting"},
		{StateCarryingPkg, "Carrying Pkg"},
		{StateWaitingHelp, "Waiting Help"},
		{StateCoordinating, "Coordinating"},
		{BotState(99), "Unknown"},
	}
	for _, tc := range tests {
		if tc.state.String() != tc.expect {
			t.Errorf("BotState(%d).String() = %q, want %q", tc.state, tc.state.String(), tc.expect)
		}
	}
}

// --- NewBaseBot tests ---

func TestNewBaseBot(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 100, 200, 3.0, 80, 50, 2)
	if b.ID() != 1 {
		t.Errorf("ID: want 1, got %d", b.ID())
	}
	if b.Type() != TypeScout {
		t.Errorf("Type: want Scout, got %v", b.Type())
	}
	if b.Position().X != 100 || b.Position().Y != 200 {
		t.Errorf("Position: want (100,200), got %v", b.Position())
	}
	if b.Health() != 100 {
		t.Errorf("Health: want 100, got %v", b.Health())
	}
	if b.MaxHealth() != 100 {
		t.Errorf("MaxHealth: want 100, got %v", b.MaxHealth())
	}
	if !b.IsAlive() {
		t.Error("new bot should be alive")
	}
	if b.GetRadius() != 6 {
		t.Errorf("Radius: want 6, got %v", b.GetRadius())
	}
	if b.GetSensorRange() != 80 {
		t.Errorf("SensorRange: want 80, got %v", b.GetSensorRange())
	}
	if b.GetCommRange() != 50 {
		t.Errorf("CommRange: want 50, got %v", b.GetCommRange())
	}
	if b.GetState() != StateFlocking {
		t.Errorf("State: want Flocking, got %v", b.GetState())
	}
	if b.GetEnergy() != 100 {
		t.Errorf("Energy: want 100, got %v", b.GetEnergy())
	}
}

// --- Concrete bot constructors ---

func TestNewScout(t *testing.T) {
	s := NewScout(1, 50, 60)
	if s.GetBase().ID() != 1 {
		t.Errorf("ID: want 1, got %d", s.ID())
	}
	if s.Type() != TypeScout {
		t.Errorf("Type: want Scout, got %v", s.Type())
	}
	if s.GetSensorRange() != 150 {
		t.Errorf("Scout SensorRange: want 150, got %v", s.GetSensorRange())
	}
}

func TestNewWorker(t *testing.T) {
	w := NewWorker(2, 50, 60)
	if w.Type() != TypeWorker {
		t.Errorf("Type: want Worker, got %v", w.Type())
	}
	if w.Capacity != 2 {
		t.Errorf("Worker capacity: want 2, got %d", w.Capacity)
	}
}

// --- Energy management ---

func TestConsumeEnergy(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.ConsumeEnergy(10, 1.0)
	if b.Energy != 90 {
		t.Errorf("expected 90, got %v", b.Energy)
	}
	// Consume more than available
	b.ConsumeEnergy(200, 1.0)
	if b.Energy != 0 {
		t.Errorf("energy should not go below 0, got %v", b.Energy)
	}
}

func TestConsumeEnergyWithDecay(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.ConsumeEnergy(10, 2.0)
	if b.Energy != 80 {
		t.Errorf("expected 80 (10*2.0), got %v", b.Energy)
	}
}

func TestRechargeEnergy(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Energy = 50
	b.RechargeEnergy(30)
	if b.Energy != 80 {
		t.Errorf("expected 80, got %v", b.Energy)
	}
	// Recharge beyond max
	b.RechargeEnergy(200)
	if b.Energy != 100 {
		t.Errorf("energy should cap at MaxEnergy 100, got %v", b.Energy)
	}
}

func TestHasEnergy(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	if !b.HasEnergy() {
		t.Error("new bot should have energy")
	}
	b.Energy = 0
	if b.HasEnergy() {
		t.Error("bot with 0 energy should not have energy")
	}
}

// --- Damage / Heal ---

func TestDamage(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Damage(30)
	if b.Hp != 70 {
		t.Errorf("expected 70 HP, got %v", b.Hp)
	}
	if !b.Alive {
		t.Error("should still be alive")
	}
	// Lethal damage
	b.Damage(100)
	if b.Hp != 0 {
		t.Errorf("HP should be 0, got %v", b.Hp)
	}
	if b.Alive {
		t.Error("should be dead after lethal damage")
	}
}

func TestHeal(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Hp = 50
	b.Heal(30)
	if b.Hp != 80 {
		t.Errorf("expected 80 HP, got %v", b.Hp)
	}
	// Heal beyond max
	b.Heal(200)
	if b.Hp != 100 {
		t.Errorf("HP should cap at MaxHp 100, got %v", b.Hp)
	}
}

// --- Fitness ---

func TestFitness(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.FitResourcesCollected = 2
	b.FitResourcesDelivered = 1
	b.FitMessagesRelayed = 5
	b.FitBotsHealed = 3
	b.FitDistanceExplored = 100
	b.FitZeroEnergyTicks = 2

	// 2*10 + 1*25 + 5*2 + 3*15 + 100*0.1 - 2*5 = 20+25+10+45+10-10 = 100
	f := b.Fitness()
	if math.Abs(f-100) > 1e-9 {
		t.Errorf("expected fitness 100, got %v", f)
	}
}

func TestResetFitness(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.FitResourcesCollected = 5
	b.FitDistanceExplored = 200
	b.ResetFitness()
	if b.Fitness() != 0 {
		t.Errorf("fitness should be 0 after reset, got %v", b.Fitness())
	}
}

// --- Resource management ---

func TestPickUpResource(t *testing.T) {
	b := NewBaseBot(1, TypeWorker, 0, 0, 3, 80, 50, 2)
	r := resource.NewResource(1, 50, 50, 10)
	ok := b.PickUpResource(r)
	if !ok {
		t.Error("should be able to pick up resource")
	}
	if len(b.Inventory) != 1 {
		t.Errorf("inventory should have 1 item, got %d", len(b.Inventory))
	}
	if b.FitResourcesCollected != 1 {
		t.Errorf("FitResourcesCollected should be 1, got %d", b.FitResourcesCollected)
	}
}

func TestPickUpResourceCapacity(t *testing.T) {
	b := NewBaseBot(1, TypeWorker, 0, 0, 3, 80, 50, 1) // capacity 1
	r1 := resource.NewResource(1, 0, 0, 10)
	r2 := resource.NewResource(2, 0, 0, 10)
	b.PickUpResource(r1)
	ok := b.PickUpResource(r2)
	if ok {
		t.Error("should not be able to pick up when at capacity")
	}
	if len(b.Inventory) != 1 {
		t.Errorf("inventory should still have 1 item, got %d", len(b.Inventory))
	}
}

func TestCanCarry(t *testing.T) {
	b := NewBaseBot(1, TypeWorker, 0, 0, 3, 80, 50, 2)
	if !b.CanCarry() {
		t.Error("empty bot should be able to carry")
	}
	r1 := resource.NewResource(1, 0, 0, 10)
	r2 := resource.NewResource(2, 0, 0, 10)
	b.PickUpResource(r1)
	b.PickUpResource(r2)
	if b.CanCarry() {
		t.Error("full bot should not be able to carry")
	}
}

func TestDropAllResources(t *testing.T) {
	b := NewBaseBot(1, TypeWorker, 100, 200, 3, 80, 50, 5)
	r1 := resource.NewResource(1, 0, 0, 10)
	r2 := resource.NewResource(2, 0, 0, 10)
	b.PickUpResource(r1)
	b.PickUpResource(r2)

	dropped := b.DropAllResources()
	if len(dropped) != 2 {
		t.Errorf("expected 2 dropped, got %d", len(dropped))
	}
	if len(b.Inventory) != 0 {
		t.Error("inventory should be empty after drop")
	}
	// Dropped resources should be at bot position
	if r1.X != 100 || r1.Y != 200 {
		t.Errorf("dropped resource position should be bot pos, got (%.0f,%.0f)", r1.X, r1.Y)
	}
}

func TestDeliverResources(t *testing.T) {
	b := NewBaseBot(1, TypeWorker, 0, 0, 3, 80, 50, 5)
	r1 := resource.NewResource(1, 0, 0, 10)
	r2 := resource.NewResource(2, 0, 0, 10)
	b.PickUpResource(r1)
	b.PickUpResource(r2)

	count := b.DeliverResources()
	if count != 2 {
		t.Errorf("expected 2 delivered, got %d", count)
	}
	if len(b.Inventory) != 0 {
		t.Error("inventory should be empty after deliver")
	}
	if b.FitResourcesDelivered != 2 {
		t.Errorf("FitResourcesDelivered should be 2, got %d", b.FitResourcesDelivered)
	}
	if !r1.IsDelivered() {
		t.Error("r1 should be delivered")
	}
}

// --- Genome speed ---

func TestApplyGenomeSpeed(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Genome.SpeedPreference = 1.5
	b.ApplyGenomeSpeed()
	if b.MaxSpeed != 4.5 {
		t.Errorf("expected MaxSpeed=4.5, got %v", b.MaxSpeed)
	}
}

// --- Communication ---

func TestShouldCommunicate(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Genome.CommFrequency = 1.0 // interval = max(1, 10-9) = 1 → always
	if !b.ShouldCommunicate(0) {
		t.Error("with CommFrequency=1.0, should always communicate")
	}
	if !b.ShouldCommunicate(1) {
		t.Error("with CommFrequency=1.0, should always communicate")
	}
}

func TestShouldCommunicateNoEnergy(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Energy = 0
	if b.ShouldCommunicate(0) {
		t.Error("should not communicate with no energy")
	}
}

// --- Energy return threshold ---

func TestShouldReturnForEnergy(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Genome.EnergyConservation = 0.5
	// threshold = 10 + 0.5*60 = 40
	b.Energy = 30
	if !b.ShouldReturnForEnergy() {
		t.Error("with energy 30 < threshold 40, should return")
	}
	b.Energy = 50
	if b.ShouldReturnForEnergy() {
		t.Error("with energy 50 > threshold 40, should not return")
	}
}

// --- SteerToward ---

func TestSteerToward(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	steer := b.SteerToward(Vec2{100, 0}, 1.0)
	if steer.X <= 0 {
		t.Errorf("should steer right toward (100,0), got X=%v", steer.X)
	}
}

func TestSteerTowardClose(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 100, 100, 3, 80, 50, 2)
	// Already at target
	steer := b.SteerToward(Vec2{100, 100}, 1.0)
	if steer.X != 0 || steer.Y != 0 {
		t.Errorf("steer to self should be zero, got (%v,%v)", steer.X, steer.Y)
	}
}

// --- Trail ---

func TestUpdateTrail(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 10, 20, 3, 80, 50, 2)
	b.Pos = Vec2{30, 40}
	b.UpdateTrail()
	trail := b.GetTrail()
	// The most recently written trail entry should be (30,40)
	found := false
	for _, pos := range trail {
		if pos.X == 30 && pos.Y == 40 {
			found = true
		}
	}
	if !found {
		t.Error("trail should contain the updated position")
	}
}

func TestTrackDistance(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Pos = Vec2{3, 4} // distance from (0,0) = 5
	b.TrackDistance()
	if math.Abs(b.FitDistanceExplored-5) > 1e-9 {
		t.Errorf("expected distance 5, got %v", b.FitDistanceExplored)
	}
}

// --- GetGenome ---

func TestGetGenome(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Genome = genetics.DefaultGenome()
	g := b.GetGenome()
	if g.FlockingWeight != 0.5 {
		t.Errorf("genome should be default, FlockingWeight=%v", g.FlockingWeight)
	}
}

// --- Velocity ---

func TestGetVelocity(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Vel = Vec2{1, 2}
	if b.Velocity().X != 1 || b.Velocity().Y != 2 {
		t.Error("Velocity should return Vel")
	}
}

// --- GetInventory ---

func TestGetInventory(t *testing.T) {
	b := NewBaseBot(1, TypeWorker, 0, 0, 3, 80, 50, 5)
	inv := b.GetInventory()
	if len(inv) != 0 {
		t.Error("new bot inventory should be empty")
	}
}

// --- Separation (uses concrete Scout type to satisfy Bot interface) ---

func TestSeparation(t *testing.T) {
	self := NewScout(0, 50, 50)
	other := NewScout(1, 55, 50)
	steer := Separation(self, []Bot{self, other}, 20)
	// Should steer away from other (to the left, negative X)
	if steer.X >= 0 {
		t.Errorf("should separate to the left, got X=%v", steer.X)
	}
}

func TestSeparationNoNearby(t *testing.T) {
	self := NewScout(0, 50, 50)
	steer := Separation(self, []Bot{self}, 20)
	if steer.X != 0 || steer.Y != 0 {
		t.Error("no nearby bots should produce zero separation")
	}
}

// --- Alignment ---

func TestAlignmentNoNeighbors(t *testing.T) {
	self := NewScout(0, 50, 50)
	steer := Alignment(self, []Bot{self}, 60)
	if steer.X != 0 || steer.Y != 0 {
		t.Error("no neighbors should produce zero alignment")
	}
}

func TestAlignment(t *testing.T) {
	self := NewScout(0, 50, 50)
	self.Vel = Vec2{1, 0}
	other := NewScout(1, 55, 50)
	other.Vel = Vec2{0, 1}
	steer := Alignment(self, []Bot{self, other}, 60)
	// Should try to match other's velocity direction
	if steer.Y <= 0 {
		t.Errorf("should align upward, got Y=%v", steer.Y)
	}
}

// --- Cohesion ---

func TestCohesionNoNeighbors(t *testing.T) {
	self := NewScout(0, 50, 50)
	steer := Cohesion(self, []Bot{self}, 60)
	if steer.X != 0 || steer.Y != 0 {
		t.Error("no neighbors should produce zero cohesion")
	}
}

func TestCohesion(t *testing.T) {
	self := NewScout(0, 0, 0)
	other := NewScout(1, 50, 0)
	steer := Cohesion(self, []Bot{self, other}, 100)
	// Should steer toward other (positive X)
	if steer.X <= 0 {
		t.Errorf("should steer toward neighbor at (50,0), got X=%v", steer.X)
	}
}

// --- Flocking ---

func TestDefaultFlockingParams(t *testing.T) {
	p := DefaultFlockingParams()
	if p.SeparationDist != 30 {
		t.Errorf("SeparationDist: want 30, got %v", p.SeparationDist)
	}
	if p.AlignmentWeight != 0.3 {
		t.Errorf("AlignmentWeight: want 0.3, got %v", p.AlignmentWeight)
	}
}

func TestComputeFlocking(t *testing.T) {
	self := NewScout(0, 50, 50)
	self.Vel = Vec2{1, 0}
	other := NewScout(1, 55, 50)
	other.Vel = Vec2{1, 0}
	params := DefaultFlockingParams()
	// Just verify it doesn't panic and returns a vector
	steer := ComputeFlocking(self, []Bot{self, other}, params)
	_ = steer // no panic is enough
}

// --- Foraging ---

func TestFindNearestResource(t *testing.T) {
	b := NewWorker(1, 50, 50)
	r1 := resource.NewResource(1, 100, 50, 10)
	r2 := resource.NewResource(2, 55, 50, 10)
	r3 := resource.NewResource(3, 200, 200, 10)
	r3.PickUp(99) // taken

	nearest := FindNearestResource(b, []*resource.Resource{r1, r2, r3})
	if nearest == nil {
		t.Fatal("should find nearest resource")
	}
	if nearest.ID != 2 {
		t.Errorf("nearest should be r2 (ID=2), got %d", nearest.ID)
	}
}

func TestFindNearestResourceNoneAvailable(t *testing.T) {
	b := NewWorker(1, 50, 50)
	r := resource.NewResource(1, 60, 50, 10)
	r.PickUp(99)
	nearest := FindNearestResource(b, []*resource.Resource{r})
	if nearest != nil {
		t.Error("should return nil when no resources available")
	}
}

func TestFindNearestResourceEmpty(t *testing.T) {
	b := NewWorker(1, 50, 50)
	nearest := FindNearestResource(b, nil)
	if nearest != nil {
		t.Error("should return nil for empty slice")
	}
}

// --- IsNearHome ---

func TestIsNearHome(t *testing.T) {
	b := NewWorker(1, 100, 100)
	if !IsNearHome(b, 100, 100, 50) {
		t.Error("bot at home center should be near home")
	}
	if IsNearHome(b, 500, 500, 50) {
		t.Error("bot far from home should not be near home")
	}
}

// --- Formation ---

func TestFormationSlotPosCircle(t *testing.T) {
	x, y := FormationSlotPos(FormationCircle, 0, 100, 100, 0, 20)
	// slot 0, heading 0: angle = 0, so x = 100+cos(0)*20 = 120, y = 100+sin(0)*20 = 100
	if math.Abs(x-120) > 0.1 || math.Abs(y-100) > 0.1 {
		t.Errorf("expected ~(120,100), got (%.1f,%.1f)", x, y)
	}
}

func TestFormationSlotPosLine(t *testing.T) {
	x, y := FormationSlotPos(FormationLine, 3, 100, 100, 0, 20)
	// slot 3, center of line: offset = (3-3)*20*0.5 = 0
	if math.Abs(x-100) > 0.1 || math.Abs(y-100) > 0.1 {
		t.Errorf("center slot should be at center, got (%.1f,%.1f)", x, y)
	}
}

func TestFormationSlotPosV(t *testing.T) {
	x, y := FormationSlotPos(FormationV, 0, 100, 100, 0, 20)
	// Just verify it returns valid coordinates
	if math.IsNaN(x) || math.IsNaN(y) {
		t.Error("V formation should return valid coordinates")
	}
}

func TestFormationSlotPosDefault(t *testing.T) {
	x, y := FormationSlotPos(FormationType(99), 0, 100, 200, 0, 20)
	if x != 100 || y != 200 {
		t.Errorf("unknown formation should return center, got (%.0f,%.0f)", x, y)
	}
}

func TestSteerToFormationSlot(t *testing.T) {
	b := NewScout(1, 0, 0)
	steer := SteerToFormationSlot(b, 100, 0)
	if steer.X <= 0 {
		t.Error("should steer right toward slot")
	}
}

// --- ApplyVelocity ---

func TestApplyVelocity(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Vel = Vec2{1, 0}
	eCfg := EnergyCfg{MoveCost: 0.1, DecayMult: 1.0}
	b.ApplyVelocity(eCfg)
	if b.Pos.X != 1 {
		t.Errorf("expected X=1, got %v", b.Pos.X)
	}
}

func TestApplyVelocityNoEnergy(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Energy = 0
	b.Vel = Vec2{5, 0}
	eCfg := EnergyCfg{MoveCost: 0.1, DecayMult: 1.0}
	b.ApplyVelocity(eCfg)
	if b.Vel.X != 0 || b.Vel.Y != 0 {
		t.Error("velocity should be zeroed when no energy")
	}
}

func TestApplyVelocitySpeedCap(t *testing.T) {
	b := NewBaseBot(1, TypeScout, 0, 0, 3, 80, 50, 2)
	b.Vel = Vec2{10, 0} // exceeds MaxSpeed=3
	eCfg := EnergyCfg{MoveCost: 0.1, DecayMult: 1.0}
	b.ApplyVelocity(eCfg)
	// Position should move by MaxSpeed (3), not 10
	if math.Abs(b.Pos.X-3) > 0.01 {
		t.Errorf("expected X≈3 (capped), got %v", b.Pos.X)
	}
}
