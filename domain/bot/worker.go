package bot

import (
	"math"
	"math/rand"
	"swarmsim/domain/comm"
)

// Worker can carry resources and transport them to the home base.
type Worker struct {
	*BaseBot
	wanderAngle    float64
	knownResources []Vec2
}

func NewWorker(id int, x, y float64) *Worker {
	return &Worker{
		BaseBot:     NewBaseBot(id, TypeWorker, x, y, 1.5, 60, 60, 2),
		wanderAngle: rand.Float64() * 2 * math.Pi,
	}
}

func (w *Worker) GetBase() *BaseBot { return w.BaseBot }

func (w *Worker) Update(ctx *UpdateContext) []comm.Message {
	if !w.Alive {
		return nil
	}

	if !w.HasEnergy() {
		return w.HandleNoEnergy()
	}

	var outbox []comm.Message

	// Process messages
	if w.ShouldCommunicate(ctx.Tick) {
		for _, msg := range ctx.Inbox {
			if msg.Type == comm.MsgResourceFound || msg.Type == comm.MsgHeavyResourceFound {
				w.knownResources = append(w.knownResources, Vec2{msg.X, msg.Y})
				// Cap at 50 entries to prevent unbounded growth
				if len(w.knownResources) > 50 {
					w.knownResources = w.knownResources[len(w.knownResources)-50:]
				}
				// Relay heavy resource messages more eagerly based on CooperationBias
				if msg.Type == comm.MsgHeavyResourceFound {
					if w.Genome.CooperationBias > 0.3 {
						outbox = append(outbox, msg)
						w.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
						w.FitMessagesRelayed++
					}
				} else {
					outbox = append(outbox, msg)
					w.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
					w.FitMessagesRelayed++
				}
			}
		}
	}

	w.DepositDangerPheromone(ctx)

	home := Vec2{ctx.HomeX, ctx.HomeY}

	// Return to base for energy if needed
	if w.ShouldReturnForEnergy() && len(w.Inventory) == 0 {
		w.State = StateReturning
		steer := w.SteerToward(home, 0.5)
		sep := Separation(w, ctx.Nearby, 20)
		w.Vel = w.Vel.Add(steer).Add(sep.Scale(1.0))
		w.ApplyVelocity(ctx.ECfg)
		return outbox
	}

	// If carrying resources, return to base
	if len(w.Inventory) >= w.Capacity || (len(w.Inventory) > 0 && len(ctx.Resources) == 0 && len(w.knownResources) == 0) {
		w.State = StateReturning
		steer := w.SteerToward(home, 0.5)
		sep := Separation(w, ctx.Nearby, 20)
		w.Vel = w.Vel.Add(steer).Add(sep.Scale(1.5))

		// Deposit FOUND_RESOURCE pheromone on return path
		if ctx.Pheromones != nil {
			w.DepositPheromone(ctx.Pheromones, PherFoundResource, 0.15, ctx.ECfg)
		}

		w.ApplyVelocity(ctx.ECfg)

		if w.Pos.Dist(home) < 70 {
			w.DeliverResources()
			w.State = StateForaging
		}
		return outbox
	}

	// Try to pick up nearby resources
	for _, r := range ctx.Resources {
		if r.IsAvailable() && w.CanCarry() {
			rPos := Vec2{r.X, r.Y}
			dist := w.Pos.Dist(rPos)
			if dist < 15 {
				if r.Heavy {
					// Heavy resource: stay near and wait for more workers
					w.State = StateCooperating
					steer := w.SteerToward(rPos, 0.2)
					w.Vel = steer.Scale(0.3)
					w.ApplyVelocity(ctx.ECfg)
					// Broadcast to attract more workers
					if w.ShouldCommunicate(ctx.Tick) {
						outbox = append(outbox, comm.NewHeavyResourceFound(w.BotID, r.X, r.Y))
						w.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
					}
					return outbox
				}
				w.PickUpResource(r)
				w.State = StateReturning
				return outbox
			}
			w.State = StateForaging
			steer := w.SteerToward(rPos, 0.5)
			sep := Separation(w, ctx.Nearby, 20)
			w.Vel = w.Vel.Add(steer).Add(sep.Scale(1.0))
			w.ApplyVelocity(ctx.ECfg)
			return outbox
		}
	}

	// Broadcast heavy resources in sensor range
	if w.ShouldCommunicate(ctx.Tick) {
		for _, r := range ctx.Resources {
			if r.Heavy && r.IsAvailable() {
				outbox = append(outbox, comm.NewHeavyResourceFound(w.BotID, r.X, r.Y))
				w.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
			}
		}
	}

	// Move toward known resource positions
	if len(w.knownResources) > 0 {
		target := w.knownResources[0]
		w.State = StateForaging
		if w.Pos.Dist(target) < 20 {
			w.knownResources = w.knownResources[1:]
		} else {
			steer := w.SteerToward(target, 0.4)
			sep := Separation(w, ctx.Nearby, 20)
			w.Vel = w.Vel.Add(steer).Add(sep.Scale(1.0))
			w.ApplyVelocity(ctx.ECfg)
			return outbox
		}
	}

	// Follow FOUND_RESOURCE pheromone gradient
	if ctx.Pheromones != nil && w.Genome.PheromoneFollow > 0.1 {
		gx, gy := ctx.Pheromones.Gradient(w.Pos.X, w.Pos.Y, PherFoundResource)
		pherSteer := Vec2{gx, gy}
		if pherSteer.Len() > 0.001 {
			w.State = StateForaging
			steer := pherSteer.Normalized().Scale(w.MaxSpeed * w.Genome.PheromoneFollow)
			sep := Separation(w, ctx.Nearby, 20)

			// Avoid danger pheromone
			dx, dy := ctx.Pheromones.Gradient(w.Pos.X, w.Pos.Y, PherDanger)
			dangerAvoid := Vec2{-dx * 3, -dy * 3}

			w.Vel = w.Vel.Add(steer.Scale(0.2)).Add(sep.Scale(1.0)).Add(dangerAvoid.Scale(0.3))
			w.ApplyVelocity(ctx.ECfg)
			return outbox
		}
	}

	// Wander with flocking
	w.State = StateFlocking
	w.wanderAngle += (rand.Float64() - 0.5) * 0.4
	wanderX := math.Cos(w.wanderAngle) * w.MaxSpeed * 0.5
	wanderY := math.Sin(w.wanderAngle) * w.MaxSpeed * 0.5
	steer := Vec2{wanderX, wanderY}
	sep := Separation(w, ctx.Nearby, 25)
	align := Alignment(w, ctx.Nearby, 60)
	coh := Cohesion(w, ctx.Nearby, 60)
	fw := w.Genome.FlockingWeight
	w.Vel = w.Vel.Add(steer.Scale(0.1)).Add(sep.Scale(1.5 * fw)).Add(align.Scale(0.3 * fw)).Add(coh.Scale(0.2 * fw))
	w.ApplyVelocity(ctx.ECfg)
	return outbox
}
