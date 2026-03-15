package bot

import (
	"math"
	"math/rand"
	"swarmsim/domain/comm"
)

// Leader coordinates worker groups with an extended communication radius.
type Leader struct {
	*BaseBot
	wanderAngle float64
	formationID int
	tickCounter int
}

func NewLeader(id int, x, y float64) *Leader {
	return &Leader{
		BaseBot:     NewBaseBot(id, TypeLeader, x, y, 1.0, 100, 200, 0),
		wanderAngle: rand.Float64() * 2 * math.Pi,
		formationID: id,
	}
}

func (l *Leader) GetBase() *BaseBot { return l.BaseBot }

func (l *Leader) Update(ctx *UpdateContext) []comm.Message {
	if !l.Alive {
		return nil
	}

	if !l.HasEnergy() {
		return l.HandleNoEnergy()
	}

	l.tickCounter++
	var outbox []comm.Message

	l.DepositDangerPheromone(ctx)

	// Return for energy if needed
	if l.ShouldReturnForEnergy() {
		l.State = StateReturning
		home := Vec2{ctx.HomeX, ctx.HomeY}
		steer := l.SteerToward(home, 0.4)
		sep := Separation(l, ctx.Nearby, 30)
		l.Vel = l.Vel.Add(steer).Add(sep.Scale(1.0))
		l.ApplyVelocity(ctx.ECfg)
		return outbox
	}

	// Relay resource-found messages
	if l.ShouldCommunicate(ctx.Tick) {
		for _, msg := range ctx.Inbox {
			if msg.Type == comm.MsgResourceFound {
				outbox = append(outbox, msg)
				l.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
				l.FitMessagesRelayed++
			}
		}

		// Broadcast heartbeat
		if l.tickCounter%10 == 0 {
			outbox = append(outbox, comm.NewHeartbeat(l.BotID, l.Pos.X, l.Pos.Y))
			l.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
		}

		// Broadcast visible resources
		for _, r := range ctx.Resources {
			if r.IsAvailable() {
				outbox = append(outbox, comm.NewResourceFound(l.BotID, r.X, r.Y))
				l.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
				l.FitMessagesRelayed++
			}
		}

		// Formation invites
		if l.tickCounter%30 == 0 {
			slot := 0
			for _, b := range ctx.Nearby {
				if b.ID() != l.BotID && b.IsAlive() && b.Type() == TypeWorker {
					outbox = append(outbox, comm.NewFormationJoin(l.BotID, l.formationID, slot, l.Pos.X, l.Pos.Y))
					l.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
					slot++
					if slot >= 6 {
						break
					}
				}
			}
		}
	}

	l.State = StateFlocking

	// Move toward clusters of workers
	var workerCenter Vec2
	workerCount := 0
	for _, b := range ctx.Nearby {
		if b.Type() == TypeWorker && b.IsAlive() {
			workerCenter = workerCenter.Add(b.Position())
			workerCount++
		}
	}

	if workerCount > 2 {
		workerCenter = workerCenter.Scale(1.0 / float64(workerCount))
		steer := l.SteerToward(workerCenter, 0.3)
		sep := Separation(l, ctx.Nearby, 30)
		l.Vel = l.Vel.Add(steer).Add(sep.Scale(1.5 * l.Genome.FlockingWeight))
	} else {
		l.wanderAngle += (rand.Float64() - 0.5) * 0.3
		wx := math.Cos(l.wanderAngle) * l.MaxSpeed * 0.5
		wy := math.Sin(l.wanderAngle) * l.MaxSpeed * 0.5
		sep := Separation(l, ctx.Nearby, 30)
		l.Vel = l.Vel.Add(Vec2{wx, wy}.Scale(0.1)).Add(sep.Scale(1.5))
	}

	l.ApplyVelocity(ctx.ECfg)
	return outbox
}
