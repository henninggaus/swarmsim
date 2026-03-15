package bot

import (
	"math"
	"swarmsim/domain/comm"
	"swarmsim/domain/genetics"
	"swarmsim/domain/resource"
)

func (v Vec2) Add(o Vec2) Vec2      { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) Sub(o Vec2) Vec2      { return Vec2{v.X - o.X, v.Y - o.Y} }
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }
func (v Vec2) Len() float64         { return math.Sqrt(v.X*v.X + v.Y*v.Y) }
func (v Vec2) Dist(o Vec2) float64  { return v.Sub(o).Len() }

func (v Vec2) Normalized() Vec2 {
	l := v.Len()
	if l == 0 {
		return Vec2{}
	}
	return Vec2{v.X / l, v.Y / l}
}

// NewBaseBot creates a base bot with common defaults.
func NewBaseBot(id int, typ BotType, x, y, maxSpeed, sensorRange, commRange float64, capacity int) *BaseBot {
	b := &BaseBot{
		BotID:       id,
		BotType:     typ,
		Pos:         Vec2{x, y},
		MaxSpeed:    maxSpeed,
		BaseSpeed:   maxSpeed,
		SensorRange: sensorRange,
		CommRange:   commRange,
		Capacity:    capacity,
		Hp:          100,
		MaxHp:       100,
		Energy:      100,
		MaxEnergy:   100,
		Alive:       true,
		Radius:      6,
		State:       StateFlocking,
	}
	b.LastPos = b.Pos
	for i := range b.Trail {
		b.Trail[i] = b.Pos
	}
	return b
}

func (b *BaseBot) ID() int                            { return b.BotID }
func (b *BaseBot) Type() BotType                      { return b.BotType }
func (b *BaseBot) Position() Vec2                     { return b.Pos }
func (b *BaseBot) Velocity() Vec2                     { return b.Vel }
func (b *BaseBot) Health() float64                    { return b.Hp }
func (b *BaseBot) MaxHealth() float64                 { return b.MaxHp }
func (b *BaseBot) IsAlive() bool                      { return b.Alive }
func (b *BaseBot) GetRadius() float64                 { return b.Radius }
func (b *BaseBot) GetSensorRange() float64            { return b.SensorRange }
func (b *BaseBot) GetCommRange() float64              { return b.CommRange }
func (b *BaseBot) GetState() BotState                 { return b.State }
func (b *BaseBot) GetInventory() []*resource.Resource { return b.Inventory }
func (b *BaseBot) GetEnergy() float64                 { return b.Energy }
func (b *BaseBot) GetGenome() *genetics.Genome        { return &b.Genome }

func (b *BaseBot) HasEnergy() bool { return b.Energy > 0 }

func (b *BaseBot) ConsumeEnergy(amount, decayMult float64) {
	b.Energy -= amount * decayMult
	if b.Energy < 0 {
		b.Energy = 0
	}
}

func (b *BaseBot) RechargeEnergy(amount float64) {
	b.Energy += amount
	if b.Energy > b.MaxEnergy {
		b.Energy = b.MaxEnergy
	}
}

func (b *BaseBot) ApplyGenomeSpeed() {
	b.MaxSpeed = b.BaseSpeed * b.Genome.SpeedPreference
}

func (b *BaseBot) Fitness() float64 {
	return float64(b.FitResourcesCollected)*10 +
		float64(b.FitResourcesDelivered)*25 +
		float64(b.FitMessagesRelayed)*2 +
		float64(b.FitBotsHealed)*15 +
		b.FitDistanceExplored*0.1 -
		float64(b.FitZeroEnergyTicks)*5
}

func (b *BaseBot) ResetFitness() {
	b.FitResourcesCollected = 0
	b.FitResourcesDelivered = 0
	b.FitMessagesRelayed = 0
	b.FitBotsHealed = 0
	b.FitDistanceExplored = 0
	b.FitZeroEnergyTicks = 0
}

func (b *BaseBot) TrackDistance() {
	d := b.Pos.Dist(b.LastPos)
	b.FitDistanceExplored += d
	b.LastPos = b.Pos
}

func (b *BaseBot) UpdateTrail() {
	b.Trail[b.TrailIdx] = b.Pos
	b.TrailIdx = (b.TrailIdx + 1) % TrailLen
}

func (b *BaseBot) GetTrail() [TrailLen]Vec2 {
	var result [TrailLen]Vec2
	for i := 0; i < TrailLen; i++ {
		result[i] = b.Trail[(b.TrailIdx+i)%TrailLen]
	}
	return result
}

// ApplyVelocity moves the bot, costs energy based on speed.
func (b *BaseBot) ApplyVelocity(eCfg EnergyCfg) {
	if !b.HasEnergy() {
		b.Vel = Vec2{}
		return
	}
	speed := b.Vel.Len()
	if speed > b.MaxSpeed {
		b.Vel = b.Vel.Normalized().Scale(b.MaxSpeed)
		speed = b.MaxSpeed
	}
	b.ConsumeEnergy(eCfg.MoveCost*speed, eCfg.DecayMult)
	if len(b.Inventory) > 0 {
		b.ConsumeEnergy(eCfg.CarryCost*float64(len(b.Inventory)), eCfg.DecayMult)
	}
	b.Pos = b.Pos.Add(b.Vel)
	b.UpdateTrail()
	b.TrackDistance()
}

func (b *BaseBot) SteerToward(target Vec2, weight float64) Vec2 {
	desired := target.Sub(b.Pos)
	dist := desired.Len()
	if dist < 1 {
		return Vec2{}
	}
	desired = desired.Normalized().Scale(b.MaxSpeed)
	steer := desired.Sub(b.Vel)
	if steer.Len() > weight {
		steer = steer.Normalized().Scale(weight)
	}
	return steer
}

func (b *BaseBot) Damage(amount float64) {
	b.Hp -= amount
	if b.Hp <= 0 {
		b.Hp = 0
		b.Alive = false
	}
}

func (b *BaseBot) Heal(amount float64) {
	b.Hp += amount
	if b.Hp > b.MaxHp {
		b.Hp = b.MaxHp
	}
}

func (b *BaseBot) CanCarry() bool { return len(b.Inventory) < b.Capacity }

func (b *BaseBot) PickUpResource(r *resource.Resource) bool {
	if !b.CanCarry() {
		return false
	}
	r.PickUp(b.BotID)
	b.Inventory = append(b.Inventory, r)
	b.FitResourcesCollected++
	return true
}

func (b *BaseBot) DropAllResources() []*resource.Resource {
	dropped := b.Inventory
	for _, r := range dropped {
		r.Drop(b.Pos.X, b.Pos.Y)
	}
	b.Inventory = nil
	return dropped
}

func (b *BaseBot) DeliverResources() int {
	count := len(b.Inventory)
	for _, r := range b.Inventory {
		r.Deliver()
	}
	b.FitResourcesDelivered += count
	b.Inventory = nil
	return count
}

// DepositPheromone deposits pheromone and consumes energy.
func (b *BaseBot) DepositPheromone(grid PheromoneGrid, pType PheromoneType, amount float64, eCfg EnergyCfg) {
	if !b.HasEnergy() || grid == nil {
		return
	}
	depositAmount := amount * (1.0 - b.Genome.EnergyConservation*0.5)
	if depositAmount < 0.005 {
		return
	}
	grid.Deposit(b.Pos.X, b.Pos.Y, pType, depositAmount)
	b.ConsumeEnergy(eCfg.PherCost, eCfg.DecayMult)
}

// HandleNoEnergy handles the common no-energy pattern: set state, track fitness, stop.
// Returns the help message to send. Callers should return immediately after this.
func (b *BaseBot) HandleNoEnergy() []comm.Message {
	b.State = StateNoEnergy
	b.FitZeroEnergyTicks++
	b.Vel = Vec2{}
	return []comm.Message{comm.NewHelpNeeded(b.BotID, b.Pos.X, b.Pos.Y)}
}

// DepositDangerPheromone deposits danger pheromone if health is low.
func (b *BaseBot) DepositDangerPheromone(ctx *UpdateContext) {
	if b.Hp < 30 && ctx.Pheromones != nil {
		b.DepositPheromone(ctx.Pheromones, PherDanger, 0.3, ctx.ECfg)
	}
}

// ShouldCommunicate returns true based on genome CommFrequency.
func (b *BaseBot) ShouldCommunicate(tick int) bool {
	if !b.HasEnergy() {
		return false
	}
	interval := int(10 - b.Genome.CommFrequency*9)
	if interval < 1 {
		interval = 1
	}
	return tick%interval == 0
}

// ShouldReturnForEnergy checks if energy is below genome-determined threshold.
func (b *BaseBot) ShouldReturnForEnergy() bool {
	threshold := 10.0 + b.Genome.EnergyConservation*60.0
	return b.Energy < threshold
}

// Separation computes a steering force to avoid crowding nearby bots.
func Separation(self Bot, nearby []Bot, desiredDist float64) Vec2 {
	var steer Vec2
	count := 0
	for _, other := range nearby {
		if other.ID() == self.ID() || !other.IsAlive() {
			continue
		}
		d := self.Position().Dist(other.Position())
		if d > 0 && d < desiredDist {
			diff := self.Position().Sub(other.Position()).Normalized().Scale(1.0 / d)
			steer = steer.Add(diff)
			count++
		}
	}
	if count > 0 {
		steer = steer.Scale(1.0 / float64(count))
	}
	return steer
}

// Alignment computes steering to match nearby bots' velocities.
func Alignment(self Bot, nearby []Bot, radius float64) Vec2 {
	var avg Vec2
	count := 0
	for _, other := range nearby {
		if other.ID() == self.ID() || !other.IsAlive() {
			continue
		}
		if self.Position().Dist(other.Position()) < radius {
			avg = avg.Add(other.Velocity())
			count++
		}
	}
	if count == 0 {
		return Vec2{}
	}
	avg = avg.Scale(1.0 / float64(count))
	return avg.Normalized().Scale(self.Velocity().Len())
}

// Cohesion computes steering toward center of nearby bots.
func Cohesion(self Bot, nearby []Bot, radius float64) Vec2 {
	var center Vec2
	count := 0
	for _, other := range nearby {
		if other.ID() == self.ID() || !other.IsAlive() {
			continue
		}
		if self.Position().Dist(other.Position()) < radius {
			center = center.Add(other.Position())
			count++
		}
	}
	if count == 0 {
		return Vec2{}
	}
	center = center.Scale(1.0 / float64(count))
	return center.Sub(self.Position()).Normalized()
}
