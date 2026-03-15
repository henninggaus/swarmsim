package bot

import (
	"math"
	"math/rand"
	"swarmsim/domain/comm"
)

// Healer repairs damaged bots and recharges energy of nearby bots.
type Healer struct {
	*BaseBot
	wanderAngle float64
}

func NewHealer(id int, x, y float64) *Healer {
	return &Healer{
		BaseBot:     NewBaseBot(id, TypeHealer, x, y, 1.2, 80, 80, 0),
		wanderAngle: rand.Float64() * 2 * math.Pi,
	}
}

func (h *Healer) GetBase() *BaseBot { return h.BaseBot }

func (h *Healer) Update(ctx *UpdateContext) []comm.Message {
	if !h.Alive {
		return nil
	}

	if !h.HasEnergy() {
		return h.HandleNoEnergy()
	}

	var outbox []comm.Message

	h.DepositDangerPheromone(ctx)

	// Return for energy if needed
	if h.ShouldReturnForEnergy() {
		h.State = StateReturning
		home := Vec2{ctx.HomeX, ctx.HomeY}
		steer := h.SteerToward(home, 0.4)
		sep := Separation(h, ctx.Nearby, 20)
		h.Vel = h.Vel.Add(steer).Add(sep.Scale(1.0))
		h.ApplyVelocity(ctx.ECfg)
		return outbox
	}

	// Find the most damaged or energy-depleted nearby bot
	var healTarget Bot
	bestPriority := 0.0
	for _, b := range ctx.Nearby {
		if b.ID() == h.BotID || !b.IsAlive() {
			continue
		}
		hpRatio := b.Health() / b.MaxHealth()
		eRatio := b.GetEnergy() / 100.0
		// Prioritize bots that need health or energy
		priority := 0.0
		if hpRatio < 0.5 {
			priority += (1.0 - hpRatio) * 2
		}
		if eRatio < 0.3 {
			priority += (1.0 - eRatio)
		}
		priority *= h.Genome.CooperationBias
		if priority > bestPriority {
			bestPriority = priority
			healTarget = b
		}
	}

	// Check messages for help requests
	var helpPos *Vec2
	for _, msg := range ctx.Inbox {
		if msg.Type == comm.MsgHelpNeeded {
			hp := Vec2{msg.X, msg.Y}
			helpPos = &hp
		}
	}

	if healTarget != nil {
		h.State = StateRepairing
		dist := h.Pos.Dist(healTarget.Position())
		if dist < 25 {
			tb := healTarget.GetBase()
			// Heal health
			if tb.Hp < tb.MaxHp*0.5 {
				tb.Heal(2.0)
				h.FitBotsHealed++
			}
			// Recharge energy
			if tb.Energy < 50 && h.Energy > 20 {
				tb.RechargeEnergy(0.5)
				h.ConsumeEnergy(0.3, ctx.ECfg.DecayMult)
			}
		} else {
			steer := h.SteerToward(healTarget.Position(), 0.5)
			h.Vel = h.Vel.Add(steer)
		}
		sep := Separation(h, ctx.Nearby, 15)
		h.Vel = h.Vel.Add(sep.Scale(1.0))
		h.ApplyVelocity(ctx.ECfg)
		return outbox
	}

	if helpPos != nil {
		h.State = StateRepairing
		steer := h.SteerToward(*helpPos, 0.4*h.Genome.CooperationBias)
		sep := Separation(h, ctx.Nearby, 20)
		h.Vel = h.Vel.Add(steer).Add(sep.Scale(1.0))
		h.ApplyVelocity(ctx.ECfg)
		return outbox
	}

	// Stay near groups (flocking)
	h.State = StateFlocking
	h.wanderAngle += (rand.Float64() - 0.5) * 0.3
	wx := math.Cos(h.wanderAngle) * h.MaxSpeed * 0.5
	wy := math.Sin(h.wanderAngle) * h.MaxSpeed * 0.5
	steer := Vec2{wx, wy}

	// Avoid danger pheromone? Healers should actually go TOWARD danger
	if ctx.Pheromones != nil && h.Genome.CooperationBias > 0.3 {
		dx, dy := ctx.Pheromones.Gradient(h.Pos.X, h.Pos.Y, PherDanger)
		steer = steer.Add(Vec2{dx * 2 * h.Genome.CooperationBias, dy * 2 * h.Genome.CooperationBias})
	}

	sep := Separation(h, ctx.Nearby, 25)
	align := Alignment(h, ctx.Nearby, 60)
	coh := Cohesion(h, ctx.Nearby, 80)
	fw := h.Genome.FlockingWeight
	h.Vel = h.Vel.Add(steer.Scale(0.1)).Add(sep.Scale(1.5 * fw)).Add(align.Scale(0.3 * fw)).Add(coh.Scale(0.4 * fw))
	h.ApplyVelocity(ctx.ECfg)
	return outbox
}
