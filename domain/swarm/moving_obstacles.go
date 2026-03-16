package swarm

import (
	"math"
	"swarmsim/domain/physics"
)

// UpdateMovingObstacles updates patrol and rotation for all obstacles.
func UpdateMovingObstacles(ss *SwarmState) {
	for _, obs := range ss.Obstacles {
		// Patrol movement
		if obs.PatrolOn {
			obs.PatrolT += obs.PatrolDir * obs.PatrolSpeed
			if obs.PatrolT >= 1.0 {
				obs.PatrolT = 1.0
				obs.PatrolDir = -1
			} else if obs.PatrolT <= 0.0 {
				obs.PatrolT = 0.0
				obs.PatrolDir = 1
			}
			// Smooth interpolation (ease in-out)
			t := smoothstep(obs.PatrolT)
			obs.X = obs.PatrolX1 + (obs.PatrolX2-obs.PatrolX1)*t
			obs.Y = obs.PatrolY1 + (obs.PatrolY2-obs.PatrolY1)*t
		}

		// Rotation (circular orbit around a center point)
		if obs.RotateOn {
			obs.RotateAngle += obs.RotateSpeed
			if obs.RotateAngle > 2*math.Pi {
				obs.RotateAngle -= 2 * math.Pi
			}
			obs.X = obs.RotateCX + math.Cos(obs.RotateAngle)*obs.RotateRadius
			obs.Y = obs.RotateCY + math.Sin(obs.RotateAngle)*obs.RotateRadius
		}
	}
}

// smoothstep provides smooth interpolation between 0 and 1.
func smoothstep(t float64) float64 {
	return t * t * (3 - 2*t)
}

// GeneratePatrolObstacles creates a set of patrolling obstacles in the arena.
func GeneratePatrolObstacles(ss *SwarmState, count int) {
	for i := 0; i < count; i++ {
		angle := float64(i) * 2 * math.Pi / float64(count)
		cx := ss.ArenaW / 2
		cy := ss.ArenaH / 2
		radius := 200.0 + float64(i%3)*80

		x1 := cx + math.Cos(angle)*radius
		y1 := cy + math.Sin(angle)*radius
		x2 := cx + math.Cos(angle+math.Pi/3)*radius
		y2 := cy + math.Sin(angle+math.Pi/3)*radius

		obs := &physics.Obstacle{
			X: x1, Y: y1,
			W: 30 + float64(ss.Rng.Intn(20)),
			H: 30 + float64(ss.Rng.Intn(20)),
			PatrolOn:    true,
			PatrolX1:    x1,
			PatrolY1:    y1,
			PatrolX2:    x2,
			PatrolY2:    y2,
			PatrolT:     ss.Rng.Float64(),
			PatrolDir:   1,
			PatrolSpeed: 0.002 + ss.Rng.Float64()*0.003,
		}
		ss.Obstacles = append(ss.Obstacles, obs)
	}
}

// GenerateRotatingObstacles creates obstacles that orbit around center points.
func GenerateRotatingObstacles(ss *SwarmState, count int) {
	for i := 0; i < count; i++ {
		cx := 100 + ss.Rng.Float64()*(ss.ArenaW-200)
		cy := 100 + ss.Rng.Float64()*(ss.ArenaH-200)
		radius := 60 + ss.Rng.Float64()*100

		obs := &physics.Obstacle{
			X: cx + radius, Y: cy,
			W: 25 + float64(ss.Rng.Intn(15)),
			H: 25 + float64(ss.Rng.Intn(15)),
			RotateOn:     true,
			RotateAngle:  ss.Rng.Float64() * 2 * math.Pi,
			RotateSpeed:  0.01 + ss.Rng.Float64()*0.02,
			RotateCX:     cx,
			RotateCY:     cy,
			RotateRadius: radius,
		}
		ss.Obstacles = append(ss.Obstacles, obs)
	}
}

// ClearMovingObstacles removes only patrol/rotating obstacles, keeping static ones.
func ClearMovingObstacles(ss *SwarmState) {
	kept := make([]*physics.Obstacle, 0, len(ss.Obstacles))
	for _, obs := range ss.Obstacles {
		if !obs.PatrolOn && !obs.RotateOn {
			kept = append(kept, obs)
		}
	}
	ss.Obstacles = kept
}

// HasMovingObstacles returns true if there are any patrol or rotating obstacles.
func HasMovingObstacles(ss *SwarmState) bool {
	for _, obs := range ss.Obstacles {
		if obs.PatrolOn || obs.RotateOn {
			return true
		}
	}
	return false
}
