package factory

import (
	"math"
	"swarmsim/locale"
)

const (
	MalfunctionChance    = 0.000005 // per tick per bot (~1 malfunction per 1000 bots every 200 ticks)
	MalfunctionSpeedMult = 0.5
	WorkshopRepairTime   = 200
)

// TickRepair handles malfunction checks, preventive maintenance, and workshop processing.
func TickRepair(fs *FactoryState) {
	// Ensure Malfunctioning slice is sized correctly
	for len(fs.Malfunctioning) < len(fs.Bots) {
		fs.Malfunctioning = append(fs.Malfunctioning, false)
	}
	// Ensure BotOpHours slice is sized correctly
	for len(fs.BotOpHours) < len(fs.Bots) {
		fs.BotOpHours = append(fs.BotOpHours, 0)
	}

	// Increment operating hours for working bots
	for i := range fs.Bots {
		bot := &fs.Bots[i]
		if bot.State != BotIdle && bot.State != BotOffShift && bot.State != BotCharging && bot.State != BotRepairing && bot.Speed > 0 {
			fs.BotOpHours[i]++
		}
	}

	// Preventive maintenance check: mandatory at 30000 hours
	for i := range fs.Bots {
		bot := &fs.Bots[i]
		if fs.Malfunctioning[i] || bot.State == BotRepairing || bot.State == BotCharging || bot.State == BotOffShift {
			continue
		}
		if i < len(fs.BotOpHours) && fs.BotOpHours[i] >= MaintenanceMandatoryHours {
			// Mandatory maintenance: auto-send to workshop
			fs.Malfunctioning[i] = true
			fs.Tasks.RemoveAssignment(i)
			bot.CarryingPkg = -1
			fs.RepairQueue = append(fs.RepairQueue, i)
			AddAlert(fs, locale.Tf("factory.alert.mandatory_maint", i), [3]uint8{220, 180, 40})
		}
	}

	// Random malfunction check
	for i := range fs.Bots {
		bot := &fs.Bots[i]
		if fs.Malfunctioning[i] || bot.State == BotRepairing || bot.State == BotCharging {
			continue
		}
		if fs.Rng.Float64() < MalfunctionChance {
			fs.Malfunctioning[i] = true
			// Remove current task
			fs.Tasks.RemoveAssignment(i)
			bot.CarryingPkg = -1

			AddAlert(fs, locale.Tf("factory.alert.malfunction", i), [3]uint8{255, 60, 60})

			// Add to repair queue
			fs.RepairQueue = append(fs.RepairQueue, i)
		}
	}

	// Apply malfunction effects (50% speed, erratic jitter)
	for i := range fs.Bots {
		if !fs.Malfunctioning[i] {
			continue
		}
		bot := &fs.Bots[i]
		if bot.State == BotRepairing {
			continue // being repaired, don't jitter
		}
		// Reduce speed and add jitter
		if bot.Speed > FactoryBotSpeed*MalfunctionSpeedMult {
			bot.Speed = FactoryBotSpeed * MalfunctionSpeedMult
		}
		bot.Angle += (fs.Rng.Float64() - 0.5) * 0.5

		// Navigate toward workshop if not already heading there
		if bot.State != BotMovingToSource {
			bot.State = BotMovingToSource
			// Create repair task
			fs.Tasks.AddTask(NewTask(TaskGoRepair,
				bot.X, bot.Y,
				fs.Workshop.X+fs.Workshop.W/2, fs.Workshop.Y+fs.Workshop.H/2,
				90, 0))
			taskIdx := len(fs.Tasks.Tasks) - 1
			fs.Tasks.Tasks[taskIdx].Assigned = i
		}
	}

	// Feature: Forklift Battery Swap — process forklifts at workshop for quick battery swap
	for len(fs.BatterySwapTimer) < len(fs.Bots) {
		fs.BatterySwapTimer = append(fs.BatterySwapTimer, 0)
	}
	for i := range fs.Bots {
		if i >= len(fs.BatterySwapTimer) || fs.BatterySwapTimer[i] <= 0 {
			continue
		}
		bot := &fs.Bots[i]
		// Check if forklift is near workshop and not actually malfunctioning
		if i < len(fs.BotRoles) && fs.BotRoles[i] == RoleForklift && !fs.Malfunctioning[i] {
			dx := (fs.Workshop.X + fs.Workshop.W/2) - bot.X
			dy := (fs.Workshop.Y + fs.Workshop.H/2) - bot.Y
			if math.Sqrt(dx*dx+dy*dy) < 30 {
				bot.Speed = 0
				fs.BatterySwapTimer[i]--
				if fs.BatterySwapTimer[i] <= 0 {
					bot.Energy = 120 // full forklift energy
					bot.State = BotIdle
					fs.Tasks.RemoveAssignment(i)
					AddAlert(fs, locale.Tf("factory.alert.battery_swap", i), [3]uint8{200, 200, 50})
				}
				continue
			}
		}
	}

	// Process workshop
	wks := &fs.Workshop
	if wks.CurrentBot >= 0 {
		// Currently repairing a bot
		wks.RepairTimer--
		if wks.RepairTimer <= 0 {
			// Repair complete
			botIdx := wks.CurrentBot
			if botIdx < len(fs.Bots) && botIdx >= 0 {
				fs.Malfunctioning[botIdx] = false
				fs.Bots[botIdx].State = BotIdle
				fs.Bots[botIdx].Speed = 0
				// Reset operating hours
				if botIdx < len(fs.BotOpHours) {
					fs.BotOpHours[botIdx] = 0
				}
			}
			wks.CurrentBot = -1
		}
	}

	// Accept next bot from queue if workshop is free
	if wks.CurrentBot < 0 && len(fs.RepairQueue) > 0 {
		for len(fs.RepairQueue) > 0 {
			botIdx := fs.RepairQueue[0]
			fs.RepairQueue = fs.RepairQueue[1:]

			if botIdx < 0 || botIdx >= len(fs.Bots) || !fs.Malfunctioning[botIdx] {
				continue // skip invalid entries
			}

			bot := &fs.Bots[botIdx]
			// Check if bot is close enough to workshop
			dx := (wks.X + wks.W/2) - bot.X
			dy := (wks.Y + wks.H/2) - bot.Y
			if math.Sqrt(dx*dx+dy*dy) < 30 {
				bot.State = BotRepairing
				bot.Speed = 0
				wks.CurrentBot = botIdx
				// Preventive maintenance is faster (100 ticks vs 200 for breakdowns)
				if botIdx < len(fs.BotOpHours) && fs.BotOpHours[botIdx] >= MaintenanceMandatoryHours {
					wks.RepairTimer = PreventiveRepairTime
				} else {
					wks.RepairTimer = WorkshopRepairTime
				}
				// Remove repair task
				fs.Tasks.RemoveAssignment(botIdx)
				break
			} else {
				// Bot not at workshop yet, re-queue at the end
				fs.RepairQueue = append(fs.RepairQueue, botIdx)
				break
			}
		}
	}
}
