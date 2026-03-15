package bot

import (
	"math"
	"math/rand"
	"swarmsim/domain/comm"
)

// Tank is slow with a large body, can push obstacles.
type Tank struct {
	*BaseBot
	wanderAngle float64
}

func NewTank(id int, x, y float64) *Tank {
	t := &Tank{
		BaseBot:     NewBaseBot(id, TypeTank, x, y, 0.8, 50, 50, 0),
		wanderAngle: rand.Float64() * 2 * math.Pi,
	}
	t.Radius = 10
	return t
}

func (t *Tank) GetBase() *BaseBot { return t.BaseBot }

func (t *Tank) Update(ctx *UpdateContext) []comm.Message {
	if !t.Alive {
		return nil
	}

	if !t.HasEnergy() {
		return t.HandleNoEnergy()
	}

	var outbox []comm.Message

	t.DepositDangerPheromone(ctx)

	// Return for energy if needed
	if t.ShouldReturnForEnergy() {
		t.State = StateReturning
		home := Vec2{ctx.HomeX, ctx.HomeY}
		steer := t.SteerToward(home, 0.3)
		sep := Separation(t, ctx.Nearby, 25)
		t.Vel = t.Vel.Add(steer).Add(sep.Scale(1.0))
		t.ApplyVelocity(ctx.ECfg)
		return outbox
	}

	// Check for help requests
	var helpTarget *Vec2
	for _, msg := range ctx.Inbox {
		if msg.Type == comm.MsgHelpNeeded && t.Genome.CooperationBias > 0.3 {
			ht := Vec2{msg.X, msg.Y}
			helpTarget = &ht
		}
		if msg.Type == comm.MsgDanger && t.ShouldCommunicate(ctx.Tick) {
			outbox = append(outbox, msg)
			t.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
			t.FitMessagesRelayed++
		}
	}

	if helpTarget != nil {
		t.State = StatePushing
		steer := t.SteerToward(*helpTarget, 0.3)
		sep := Separation(t, ctx.Nearby, 25)
		t.Vel = t.Vel.Add(steer).Add(sep.Scale(1.0))
		t.ApplyVelocity(ctx.ECfg)
		return outbox
	}

	// Patrol / wander
	t.State = StateFlocking
	t.wanderAngle += (rand.Float64() - 0.5) * 0.3
	wx := math.Cos(t.wanderAngle) * t.MaxSpeed * 0.6
	wy := math.Sin(t.wanderAngle) * t.MaxSpeed * 0.6
	steer := Vec2{wx, wy}

	// Avoid danger pheromone
	if ctx.Pheromones != nil {
		dx, dy := ctx.Pheromones.Gradient(t.Pos.X, t.Pos.Y, PherDanger)
		steer = steer.Add(Vec2{-dx * 3, -dy * 3})
	}

	sep := Separation(t, ctx.Nearby, 30)
	align := Alignment(t, ctx.Nearby, 50)
	fw := t.Genome.FlockingWeight
	t.Vel = t.Vel.Add(steer.Scale(0.1)).Add(sep.Scale(1.5 * fw)).Add(align.Scale(0.2 * fw))
	t.ApplyVelocity(ctx.ECfg)
	return outbox
}
