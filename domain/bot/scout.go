package bot

import (
	"math"
	"math/rand"
	"swarmsim/domain/comm"
)

// Scout has a large sensor radius, is fast, but cannot carry resources.
type Scout struct {
	*BaseBot
	wanderAngle float64
}

func NewScout(id int, x, y float64) *Scout {
	return &Scout{
		BaseBot:     NewBaseBot(id, TypeScout, x, y, 3.0, 150, 80, 0),
		wanderAngle: rand.Float64() * 2 * math.Pi,
	}
}

func (s *Scout) GetBase() *BaseBot { return s.BaseBot }

func (s *Scout) Update(ctx *UpdateContext) []comm.Message {
	if !s.Alive {
		return nil
	}

	if !s.HasEnergy() {
		return s.HandleNoEnergy()
	}

	var outbox []comm.Message

	// Always broadcast resource positions when visible (even when returning)
	if s.ShouldCommunicate(ctx.Tick) {
		for _, r := range ctx.Resources {
			if r.IsAvailable() {
				outbox = append(outbox, comm.NewResourceFound(s.BotID, r.X, r.Y))
				s.ConsumeEnergy(ctx.ECfg.MsgCost, ctx.ECfg.DecayMult)
				s.FitMessagesRelayed++
			}
		}
	}

	s.DepositDangerPheromone(ctx)

	if s.Hp < 50 {
		outbox = append(outbox, comm.NewHelpNeeded(s.BotID, s.Pos.X, s.Pos.Y))
	}

	// Return to base for energy if needed
	if s.ShouldReturnForEnergy() {
		s.State = StateReturning
		home := Vec2{ctx.HomeX, ctx.HomeY}
		steer := s.SteerToward(home, 0.5)
		sep := Separation(s, ctx.Nearby, 30)
		s.Vel = s.Vel.Add(steer).Add(sep.Scale(1.5))
		s.ApplyVelocity(ctx.ECfg)
		return outbox
	}

	s.State = StateScouting

	// Deposit SEARCH pheromone
	if ctx.Pheromones != nil {
		s.DepositPheromone(ctx.Pheromones, PherSearch, 0.1, ctx.ECfg)
	}

	// Wander — avoid areas with high SEARCH pheromone
	s.wanderAngle += (rand.Float64() - 0.5) * (0.4 + s.Genome.ExplorationDrive*0.4)
	wanderX := math.Cos(s.wanderAngle) * s.MaxSpeed
	wanderY := math.Sin(s.wanderAngle) * s.MaxSpeed
	steer := Vec2{wanderX, wanderY}

	if ctx.Pheromones != nil {
		// Avoid SEARCH pheromone (already explored)
		gx, gy := ctx.Pheromones.Gradient(s.Pos.X, s.Pos.Y, PherSearch)
		avoidStr := s.Genome.ExplorationDrive * 3.0
		steer = steer.Add(Vec2{-gx * avoidStr, -gy * avoidStr})

		// Avoid DANGER pheromone
		dx, dy := ctx.Pheromones.Gradient(s.Pos.X, s.Pos.Y, PherDanger)
		steer = steer.Add(Vec2{-dx * 5, -dy * 5})
	}

	sep := Separation(s, ctx.Nearby, 30)
	steer = steer.Add(sep.Scale(2.0 * s.Genome.FlockingWeight))

	s.Vel = s.Vel.Add(steer.Scale(0.1))
	s.ApplyVelocity(ctx.ECfg)
	return outbox
}
