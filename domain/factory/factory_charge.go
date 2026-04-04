package factory

import (
	"math"
	"swarmsim/domain/swarm"
)

// Energy drain/charge constants
const (
	EnergyDrainMoving = 0.02 // per tick when Speed > 0
	EnergyDrainIdle   = 0.005 // per tick when idle
	EnergyLowThreshold = 20.0
	EnergyCritical     = 5.0
)

// NeedsCharge returns true if bot energy is below the low threshold.
func NeedsCharge(bot *swarm.SwarmBot) bool {
	return bot.Energy < EnergyLowThreshold
}

// nearestChargerDist returns the distance to the nearest available charger.
func nearestChargerDist(fs *FactoryState, x, y float64) float64 {
	points := make([][2]float64, len(fs.Chargers))
	for i := range fs.Chargers {
		ch := &fs.Chargers[i]
		points[i] = [2]float64{ch.X + ch.W/2, ch.Y + ch.H/2}
	}
	_, dist := nearestPoint(x, y, points)
	if dist == math.MaxFloat64 {
		return 1e18
	}
	return dist
}

// needsSmartCharge checks if bot should charge now based on energy prediction:
// Can the bot complete its current task AND return to a charger?
func needsSmartCharge(fs *FactoryState, bot *swarm.SwarmBot, botIdx int) bool {
	// Still use hard threshold as a fallback
	if bot.Energy < EnergyLowThreshold {
		return true
	}

	// Feature 9: Smart energy economics — if energy 20-35% and daytime, wait for night
	// (only if no urgent task assigned)
	if bot.Energy >= EnergyLowThreshold && bot.Energy < 35 {
		dayPhase := float64(fs.Tick%10000) / 10000.0
		isNight := dayPhase > 0.7 || dayPhase < 0.2
		if !isNight {
			taskIdx := findBotTask(fs, botIdx)
			if taskIdx < 0 {
				// No task: defer charging until night
				return false
			}
		}
	}

	// Predictive charging: estimate if bot can complete task + reach charger
	taskIdx := findBotTask(fs, botIdx)
	if taskIdx < 0 {
		// No task: check if energy is below 35% and nearest charger is far
		chargerDist := nearestChargerDist(fs, bot.X, bot.Y)
		needed := chargerDist*0.02 + 10
		return bot.Energy < needed
	}

	task := &fs.Tasks.Tasks[taskIdx]
	var distToTarget float64
	if bot.State == BotMovingToSource || bot.State == BotIdle {
		dx := task.SourceX - bot.X
		dy := task.SourceY - bot.Y
		distToTarget = math.Sqrt(dx*dx + dy*dy)
		// Also need to go from source to dest
		ddx := task.DestX - task.SourceX
		ddy := task.DestY - task.SourceY
		distToTarget += math.Sqrt(ddx*ddx + ddy*ddy)
	} else {
		dx := task.DestX - bot.X
		dy := task.DestY - bot.Y
		distToTarget = math.Sqrt(dx*dx + dy*dy)
	}

	// Distance from task destination to nearest charger
	chargerDist := nearestChargerDist(fs, task.DestX, task.DestY)

	// Energy needed: travel cost + safety margin
	needed := distToTarget*0.02 + chargerDist*0.02 + 10

	return bot.Energy < needed
}

// TickCharging updates energy for all bots and manages charging station queues.
func TickCharging(fs *FactoryState) {
	for i := range fs.Bots {
		bot := &fs.Bots[i]

		// Skip off-shift and emergency-evacuating bots for energy drain but still drain
		if bot.State == BotOffShift {
			bot.Energy -= EnergyDrainIdle * 0.1 // minimal drain while parked
			if bot.Energy < 0 {
				bot.Energy = 0
			}
			continue
		}

		// Drain energy
		if bot.Speed > 0.1 {
			bot.Energy -= EnergyDrainMoving
		} else {
			bot.Energy -= EnergyDrainIdle
		}
		if bot.Energy < 0 {
			bot.Energy = 0
		}

		// Skip charge assignment during emergency
		if fs.Emergency {
			continue
		}

		// Skip if charger busy delay is active (bot communication)
		if i < len(fs.ChargerBusyDelay) && fs.ChargerBusyDelay[i] > 0 {
			continue
		}

		// Smart energy management: predict if bot can complete task + reach charger
		if needsSmartCharge(fs, bot, i) && bot.State != BotCharging && bot.State != BotRepairing && bot.State != BotOffShift && bot.State != BotEmergencyEvac {
			// Remove current task assignment
			fs.Tasks.RemoveAssignment(i)
			bot.CarryingPkg = -1 // drop any carried item

			// Feature: Forklift Battery Swap — forklifts go to workshop instead of charger
			if i < len(fs.BotRoles) && fs.BotRoles[i] == RoleForklift {
				// Create a task to navigate to workshop for battery swap
				fs.Tasks.AddTask(NewTask(TaskGoRepair, bot.X, bot.Y,
					fs.Workshop.X+fs.Workshop.W/2, fs.Workshop.Y+fs.Workshop.H/2, 100, 0))
				taskIdx := len(fs.Tasks.Tasks) - 1
				fs.Tasks.Tasks[taskIdx].Assigned = i
				bot.State = BotMovingToSource
				// Initialize battery swap timer
				for len(fs.BatterySwapTimer) <= i {
					fs.BatterySwapTimer = append(fs.BatterySwapTimer, 0)
				}
				fs.BatterySwapTimer[i] = 30 // 30 tick swap
				continue // skip charger logic
			}

			// Find nearest charger with capacity
			bestCharger := -1
			bestDist := 1e18
			allFull := true
			for ci := range fs.Chargers {
				ch := &fs.Chargers[ci]
				if len(ch.Occupants) >= ch.MaxBots {
					continue
				}
				allFull = false
				dx := (ch.X + ch.W/2) - bot.X
				dy := (ch.Y + ch.H/2) - bot.Y
				d := dx*dx + dy*dy
				if d < bestDist {
					bestDist = d
					bestCharger = ci
				}
			}

			// Bot communication: if all chargers are full, broadcast to nearby bots
			if allFull && fs.BotHash != nil {
				neighbors := fs.BotHash.Query(bot.X, bot.Y, 100.0)
				for _, nIdx := range neighbors {
					if nIdx == i || nIdx < 0 || nIdx >= len(fs.ChargerBusyDelay) {
						continue
					}
					fs.ChargerBusyDelay[nIdx] = 200 // delay charging for 200 ticks
				}
			}

			if bestCharger >= 0 {
				ch := &fs.Chargers[bestCharger]
				// Create a charge task targeting the charger
				fs.Tasks.AddTask(NewTask(TaskGoCharge,
					bot.X, bot.Y,
					ch.X+ch.W/2, ch.Y+ch.H/2,
					100, 0)) // highest priority
				// Assign it immediately
				taskIdx := len(fs.Tasks.Tasks) - 1
				fs.Tasks.Tasks[taskIdx].Assigned = i
				bot.State = BotMovingToSource // will navigate to charger via dest
			}
		}
	}

	// Feature: Idle Optimization — proactive charging for idle bots with < 80% energy
	if fs.Tick%200 == 0 {
		for i := range fs.Bots {
			bot := &fs.Bots[i]
			if bot.State != BotIdle || bot.Energy >= 80 {
				continue
			}
			// Only if not too many bots already charging
			if fs.Stats.BotsCharging >= len(fs.Chargers)*3 {
				break
			}
			// Skip if already has a task
			if findBotTask(fs, i) >= 0 {
				continue
			}
			// Find nearest charger with capacity
			bestCh := -1
			bestD := 1e18
			for ci := range fs.Chargers {
				ch := &fs.Chargers[ci]
				if len(ch.Occupants) >= ch.MaxBots {
					continue
				}
				dx := (ch.X + ch.W/2) - bot.X
				dy := (ch.Y + ch.H/2) - bot.Y
				d := dx*dx + dy*dy
				if d < bestD {
					bestD = d
					bestCh = ci
				}
			}
			if bestCh >= 0 {
				ch := &fs.Chargers[bestCh]
				fs.Tasks.AddTask(NewTask(TaskGoCharge,
					bot.X, bot.Y,
					ch.X+ch.W/2, ch.Y+ch.H/2,
					50, 0)) // lower priority than urgent charging
				taskIdx := len(fs.Tasks.Tasks) - 1
				fs.Tasks.Tasks[taskIdx].Assigned = i
				bot.State = BotMovingToSource
			}
		}
	}

	// Process bots at chargers
	for ci := range fs.Chargers {
		ch := &fs.Chargers[ci]

		// Check for bots that arrived at this charger
		for i := range fs.Bots {
			bot := &fs.Bots[i]
			if bot.State != BotCharging {
				// Check if bot is assigned a charge task and close to this charger
				taskIdx := findBotTask(fs, i)
				if taskIdx < 0 {
					continue
				}
				task := &fs.Tasks.Tasks[taskIdx]
				if task.Type != TaskGoCharge {
					continue
				}
				dx := (ch.X + ch.W/2) - bot.X
				dy := (ch.Y + ch.H/2) - bot.Y
				if dx*dx+dy*dy < 25 { // within 5px
					if len(ch.Occupants) < ch.MaxBots {
						bot.State = BotCharging
						bot.Speed = 0
						ch.Occupants = append(ch.Occupants, i)
						fs.Tasks.CompleteTask(taskIdx)
					}
				}
			}
		}

		// Charge occupants (Feature 11: skip charging during Power Outage)
		powerOutage := IsEventActive(fs, EventPowerOutage)
		remaining := ch.Occupants[:0]
		for _, botIdx := range ch.Occupants {
			if botIdx < 0 || botIdx >= len(fs.Bots) {
				continue
			}
			bot := &fs.Bots[botIdx]
			if powerOutage {
				// No charging during power outage, but bots stay at charger
				remaining = append(remaining, botIdx)
				continue
			}
			bot.Energy += ch.ChargeRate

			// Feature 9: Energy economics — charge costs money
			dayPhase := float64(fs.Tick%10000) / 10000.0
			isNight := dayPhase > 0.7 || dayPhase < 0.2
			costPerUnit := fs.EnergyCostDay
			if isNight {
				costPerUnit = fs.EnergyCostNight
			}
			fs.Budget -= ch.ChargeRate * costPerUnit
			fs.TotalEnergyCost += ch.ChargeRate * costPerUnit
			maxEnergy := 100.0
			if botIdx < len(fs.BotRoles) && fs.BotRoles[botIdx] == RoleForklift {
				maxEnergy = 120.0
			} else if botIdx < len(fs.BotRoles) && fs.BotRoles[botIdx] == RoleExpress {
				maxEnergy = 60.0
			}
			if bot.Energy >= maxEnergy {
				bot.Energy = maxEnergy
				bot.State = BotIdle
				bot.Speed = 0
			} else {
				remaining = append(remaining, botIdx)
			}
		}
		ch.Occupants = remaining
	}
}
