package factory

import (
	"math"
	"swarmsim/domain/physics"
	"swarmsim/domain/swarm"
)

// navigateTo moves bot toward target. Returns true when arrived (within 5px).
// Includes bot-bot collision avoidance using the spatial hash.
func navigateTo(bot *swarm.SwarmBot, tx, ty float64, fs *FactoryState) bool {
	return navigateToIdx(bot, -1, tx, ty, fs)
}

// navigateToIdx is like navigateTo but knows the bot's index for collision avoidance.
func navigateToIdx(bot *swarm.SwarmBot, botIdx int, tx, ty float64, fs *FactoryState) bool {
	dx := tx - bot.X
	dy := ty - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 5 {
		bot.Speed = 0
		return true
	}

	// Direct navigation with smooth steering
	targetAngle := math.Atan2(dy, dx)
	diff := targetAngle - bot.Angle
	// Wrap angle difference to [-pi, pi]
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	maxTurn := 0.15
	if diff > maxTurn {
		diff = maxTurn
	}
	if diff < -maxTurn {
		diff = -maxTurn
	}
	bot.Angle += diff

	// Role-based base speed
	baseSpeed := FactoryBotSpeed
	if botIdx >= 0 && botIdx < len(fs.BotRoles) {
		switch fs.BotRoles[botIdx] {
		case RoleForklift:
			baseSpeed = 2.0
		case RoleExpress:
			baseSpeed = 5.0
		}
	}

	// Experience/efficiency multiplier
	speedMult := 1.0
	if botIdx >= 0 && botIdx < len(fs.BotDeliveries) {
		d := fs.BotDeliveries[botIdx]
		if d >= 100 {
			speedMult = 1.3
		} else if d >= 50 {
			speedMult = 1.2
		} else if d >= 10 {
			speedMult = 1.1
		}
	}
	bot.Speed = baseSpeed * speedMult

	// Feature 11: Efficiency Bonus event — 1.5x speed
	if IsEventActive(fs, EventEfficiencyBonus) {
		bot.Speed *= 1.5
	}

	// Lane steering bias: pull toward nearest lane center
	laneYs := []float64{HallY + 100, AisleY, AisleY + AisleH, HallY + HallH - 100}
	nearestLaneDist := 9999.0
	nearestLaneY := bot.Y
	for _, ly := range laneYs {
		d := math.Abs(bot.Y - ly)
		if d < nearestLaneDist {
			nearestLaneDist = d
			nearestLaneY = ly
		}
	}
	if nearestLaneDist < 40 && nearestLaneDist > 5 {
		// Small pull toward lane center
		pull := (nearestLaneY - bot.Y) * 0.02
		bot.Y += pull
	}

	// Bot-bot collision avoidance using spatial hash
	if botIdx >= 0 && fs.BotHash != nil {
		neighbors := fs.BotHash.Query(bot.X, bot.Y, 12.0)
		for _, nIdx := range neighbors {
			if nIdx == botIdx || nIdx < 0 || nIdx >= len(fs.Bots) {
				continue
			}
			nb := &fs.Bots[nIdx]
			ndx := nb.X - bot.X
			ndy := nb.Y - bot.Y
			ndist := math.Sqrt(ndx*ndx + ndy*ndy)
			if ndist < 8 && ndist > 0 {
				// Check if neighbor is ahead (within ~90 degrees of heading)
				angleToNeighbor := math.Atan2(ndy, ndx)
				angleDiff := angleToNeighbor - bot.Angle
				for angleDiff > math.Pi {
					angleDiff -= 2 * math.Pi
				}
				for angleDiff < -math.Pi {
					angleDiff += 2 * math.Pi
				}
				if math.Abs(angleDiff) < math.Pi/2 {
					// Bot ahead - steer right and slow down
					bot.Angle += 0.3
					bot.Speed *= 0.5
					break
				}
			}
		}
	}

	// Feature: Jam detection — if bot hasn't moved significantly in 50 ticks, reroute
	if botIdx >= 0 && botIdx < len(fs.BotCounters) {
		if bot.Speed < 0.5 {
			fs.BotCounters[botIdx]++
			if fs.BotCounters[botIdx] > 50 {
				// Stuck! Random reroute
				bot.Angle += (fs.Rng.Float64() - 0.5) * math.Pi
				bot.Speed = FactoryBotSpeed
				fs.BotCounters[botIdx] = 0
			}
		} else {
			if fs.BotCounters[botIdx] > 0 {
				fs.BotCounters[botIdx] = 0
			}
		}
	}

	// Feature: Congestion separation — when >5 bots within 30px, apply strong separation
	if botIdx >= 0 && fs.BotHash != nil {
		congNeighbors := fs.BotHash.Query(bot.X, bot.Y, 30.0)
		if len(congNeighbors) > 5 {
			for _, nIdx := range congNeighbors {
				if nIdx == botIdx || nIdx < 0 || nIdx >= len(fs.Bots) {
					continue
				}
				nb := &fs.Bots[nIdx]
				cdx := bot.X - nb.X
				cdy := bot.Y - nb.Y
				cdist := math.Sqrt(cdx*cdx + cdy*cdy)
				if cdist < 30 && cdist > 0.1 {
					bot.X += cdx / cdist * 1.5
					bot.Y += cdy / cdist * 1.5
				}
			}
		}
	}

	// --- Feature 1: Traffic Rules — one-way lane enforcement in the main aisle ---
	inAisle := bot.Y > AisleY && bot.Y < AisleY+AisleH
	if inAisle {
		goingRight := math.Cos(bot.Angle) > 0 // heading right
		inUpperLane := bot.Y < AisleY+AisleH/2
		// If going right, should be in upper lane. If going left, lower lane.
		if goingRight && !inUpperLane {
			bot.Y -= 0.5 // nudge up toward upper lane
		} else if !goingRight && inUpperLane {
			bot.Y += 0.5 // nudge down toward lower lane
		}
	}

	// --- Feature 1: Honk detection — flash when bot directly ahead within 6px ---
	if botIdx >= 0 && botIdx < len(fs.HonkFlash) && fs.BotHash != nil {
		neighbors := fs.BotHash.Query(bot.X, bot.Y, 8.0)
		for _, nIdx := range neighbors {
			if nIdx == botIdx || nIdx < 0 || nIdx >= len(fs.Bots) {
				continue
			}
			nb := &fs.Bots[nIdx]
			ndx := nb.X - bot.X
			ndy := nb.Y - bot.Y
			ndist := math.Sqrt(ndx*ndx + ndy*ndy)
			if ndist < 6 && ndist > 0 {
				// Check if neighbor is directly ahead
				angleToN := math.Atan2(ndy, ndx)
				adiff := angleToN - bot.Angle
				for adiff > math.Pi {
					adiff -= 2 * math.Pi
				}
				for adiff < -math.Pi {
					adiff += 2 * math.Pi
				}
				if math.Abs(adiff) < math.Pi/4 {
					fs.HonkFlash[botIdx] = 5
					break
				}
			}
		}
	}

	bot.X += math.Cos(bot.Angle) * bot.Speed
	bot.Y += math.Sin(bot.Angle) * bot.Speed

	// Wall collision — use spatial hash for O(1) lookup instead of O(n) brute force
	if fs.ObsHash != nil {
		nearWalls := fs.ObsHash.Query(bot.X, bot.Y, 30)
		for _, wIdx := range nearWalls {
			if wIdx < 0 || wIdx >= len(fs.Walls) {
				continue
			}
			wall := fs.Walls[wIdx]
			hit, _, _ := physics.CircleRectCollision(bot.X, bot.Y, FactoryBotRadius, wall.X, wall.Y, wall.W, wall.H)
			if hit {
				bot.X -= math.Cos(bot.Angle) * bot.Speed * 2
				bot.Y -= math.Sin(bot.Angle) * bot.Speed * 2
				bot.Angle += math.Pi/2 + fs.Rng.Float64()*math.Pi/2
				break
			}
		}
	} else {
		for _, wall := range fs.Walls {
			hit, _, _ := physics.CircleRectCollision(bot.X, bot.Y, FactoryBotRadius, wall.X, wall.Y, wall.W, wall.H)
			if hit {
				bot.X -= math.Cos(bot.Angle) * bot.Speed * 2
				bot.Y -= math.Sin(bot.Angle) * bot.Speed * 2
				bot.Angle += math.Pi/2 + fs.Rng.Float64()*math.Pi/2
				break
			}
		}
	}

	// Clamp to world bounds
	if bot.X < FactoryBotRadius {
		bot.X = FactoryBotRadius
	}
	if bot.X > WorldW-FactoryBotRadius {
		bot.X = WorldW - FactoryBotRadius
	}
	if bot.Y < FactoryBotRadius {
		bot.Y = FactoryBotRadius
	}
	if bot.Y > WorldH-FactoryBotRadius {
		bot.Y = WorldH - FactoryBotRadius
	}

	return false
}

// botWander makes an idle bot navigate to the nearest parking zone and park in grid formation.
// Uses precomputed BotParkingSlot for O(1) slot lookup instead of O(n) inner loop.
func botWander(bot *swarm.SwarmBot, botIdx int, fs *FactoryState) {
	// Use precomputed parking assignment if available
	if len(fs.ParkingZones) > 0 && botIdx < len(fs.BotParkingSlot) {
		pa := fs.BotParkingSlot[botIdx]
		bestZone := fs.ParkingZones[pa.ZoneIdx]

		// Parking grid: 6 columns, as many rows as needed
		cols := 6
		slotW := 12.0
		row := pa.SlotIdx / cols
		col := pa.SlotIdx % cols
		targetX := bestZone[0] - 30 + float64(col)*slotW
		targetY := bestZone[1] - 30 + float64(row)*slotW

		dx := targetX - bot.X
		dy := targetY - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist < 3 {
			// Snap to grid position and stop
			bot.X = targetX
			bot.Y = targetY
			bot.Speed = 0
			return
		}

		// Navigate to parking slot
		targetAngle := math.Atan2(dy, dx)
		diff := targetAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > 0.15 {
			diff = 0.15
		}
		if diff < -0.15 {
			diff = -0.15
		}
		bot.Angle += diff
		bot.Speed = FactoryBotSpeed * 0.3
	} else {
		// Fallback: old wander behavior
		if fs.Rng.Float64() < 0.03 {
			bot.Angle += (fs.Rng.Float64() - 0.5) * 1.0
		}
		bot.Speed = FactoryBotSpeed * 0.3
	}

	bot.X += math.Cos(bot.Angle) * bot.Speed
	bot.Y += math.Sin(bot.Angle) * bot.Speed

	// Wall collision
	for _, wall := range fs.Walls {
		hit, _, _ := physics.CircleRectCollision(bot.X, bot.Y, FactoryBotRadius, wall.X, wall.Y, wall.W, wall.H)
		if hit {
			bot.X -= math.Cos(bot.Angle) * bot.Speed * 2
			bot.Y -= math.Sin(bot.Angle) * bot.Speed * 2
			bot.Angle += math.Pi + (fs.Rng.Float64()-0.5)*0.5
		}
	}
}

// findNearestGateTarget returns the coordinate of the nearest door's yard-side target.
func findNearestGateTarget(fs *FactoryState, bot *swarm.SwarmBot) [2]float64 {
	if len(fs.Doors) == 0 {
		return [2]float64{WorldW / 2, 300} // default: center of yard
	}
	points := make([][2]float64, len(fs.Doors))
	for i, door := range fs.Doors {
		points[i] = [2]float64{door.X + door.W/2, door.Y - 50}
	}
	idx, _ := nearestPoint(bot.X, bot.Y, points)
	if idx < 0 {
		return [2]float64{WorldW / 2, 300}
	}
	return points[idx]
}

// findNearestStorage returns a pointer to the nearest storage area to (x, y).
func findNearestStorage(fs *FactoryState, x, y float64) *StorageArea {
	if len(fs.Storage) == 0 {
		return nil
	}
	points := make([][2]float64, len(fs.Storage))
	for i := range fs.Storage {
		st := &fs.Storage[i]
		points[i] = [2]float64{st.X + st.W/2, st.Y + st.H/2}
	}
	idx, _ := nearestPoint(x, y, points)
	if idx < 0 {
		return nil
	}
	return &fs.Storage[idx]
}
