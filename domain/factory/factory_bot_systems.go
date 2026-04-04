package factory

import (
	"math"
	"swarmsim/locale"
)

// broadcastCongestion checks if a bot sees 3+ neighbors ahead and tells nearby bots to reroute.
func broadcastCongestion(fs *FactoryState, botIdx int) {
	if fs.BotHash == nil {
		return
	}
	bot := &fs.Bots[botIdx]
	neighbors := fs.BotHash.Query(bot.X, bot.Y, 15.0)
	aheadCount := 0
	for _, nIdx := range neighbors {
		if nIdx == botIdx || nIdx < 0 || nIdx >= len(fs.Bots) {
			continue
		}
		nb := &fs.Bots[nIdx]
		ndx := nb.X - bot.X
		ndy := nb.Y - bot.Y
		ndist := math.Sqrt(ndx*ndx + ndy*ndy)
		if ndist < 15 && ndist > 0 {
			aheadCount++
		}
	}
	// Congestion detected: nudge nearby bots to steer away
	if aheadCount >= 3 {
		nearby := fs.BotHash.Query(bot.X, bot.Y, 50.0)
		for _, nIdx := range nearby {
			if nIdx == botIdx || nIdx < 0 || nIdx >= len(fs.Bots) {
				continue
			}
			// Slightly randomize angle to break traffic jams
			fs.Bots[nIdx].Angle += (fs.Rng.Float64() - 0.5) * 0.4
		}
	}
}

// --- Order fulfillment ---

// checkOrderFulfillment checks if a delivered part fulfills any pending order.
func checkOrderFulfillment(fs *FactoryState, partColor int) {
	for i := range fs.Orders {
		o := &fs.Orders[i]
		if o.Completed || o.OutputColor != partColor {
			continue
		}
		o.Fulfilled++
		if o.Fulfilled >= o.Quantity {
			o.Completed = true
			fs.CompletedOrders++
			fs.Score += 200
			// Feature 9: completed orders add to budget
			fs.Budget += 500
			AddAlert(fs, locale.Tf("factory.alert.order_complete", o.ID), [3]uint8{80, 255, 120})
		}
		return
	}
}

// --- Shift System ---

// TickShiftSystem handles the 5000-tick shift rotation.
func TickShiftSystem(fs *FactoryState) {
	// Grow slices if needed (bot count can change)
	for len(fs.ShiftOnDuty) < len(fs.Bots) {
		fs.ShiftOnDuty = append(fs.ShiftOnDuty, true)
	}
	for len(fs.ChargerBusyDelay) < len(fs.Bots) {
		fs.ChargerBusyDelay = append(fs.ChargerBusyDelay, 0)
	}
	for len(fs.BotRoles) < len(fs.Bots) {
		fs.BotRoles = append(fs.BotRoles, RoleTransporter)
	}
	for len(fs.BotOpHours) < len(fs.Bots) {
		fs.BotOpHours = append(fs.BotOpHours, 0)
	}
	for len(fs.BotDeliveries) < len(fs.Bots) {
		fs.BotDeliveries = append(fs.BotDeliveries, 0)
	}

	fs.ShiftTimer--
	if fs.ShiftTimer <= 0 {
		fs.ShiftTimer = 5000

		// Alert for shift change
		onCount := 0
		for i := range fs.ShiftOnDuty {
			if i < len(fs.Bots) && !fs.ShiftOnDuty[i] {
				onCount++
			}
		}
		AddAlert(fs, locale.Tf("factory.alert.shift_change", onCount), [3]uint8{140, 180, 255})

		// Shift change: swap on-duty and off-duty bots
		for i := range fs.Bots {
			if i >= len(fs.ShiftOnDuty) {
				break
			}
			wasOnShift := fs.ShiftOnDuty[i]
			fs.ShiftOnDuty[i] = !wasOnShift

			if wasOnShift {
				// Feature 8: Shift handover — drop carried part to nearest storage
				if fs.Bots[i].CarryingPkg > 0 {
					nearestSt := findNearestStorage(fs, fs.Bots[i].X, fs.Bots[i].Y)
					if nearestSt != nil {
						addToStorageDirect(nearestSt, fs.Bots[i].CarryingPkg, fs.Tick)
					}
					fs.Bots[i].CarryingPkg = -1
				}
				// Un-assign task so incoming shift can pick it up
				fs.Tasks.RemoveAssignment(i)
				fs.Bots[i].State = BotOffShift
				fs.Bots[i].Speed = 0
				// Park position: along bottom wall of hall
				fs.Bots[i].X = HallX + 30 + fs.Rng.Float64()*(HallW-60)
				fs.Bots[i].Y = HallY + HallH - 30 - fs.Rng.Float64()*20
			} else {
				// Coming on shift: activate
				if fs.Bots[i].State == BotOffShift {
					fs.Bots[i].State = BotIdle
				}
			}
		}
	}

	// Keep off-duty bots in off-shift state
	for i := range fs.Bots {
		if i >= len(fs.ShiftOnDuty) {
			break
		}
		if !fs.ShiftOnDuty[i] && fs.Bots[i].State != BotOffShift && fs.Bots[i].State != BotCharging && fs.Bots[i].State != BotRepairing {
			fs.Bots[i].State = BotOffShift
			fs.Bots[i].Speed = 0
			fs.Tasks.RemoveAssignment(i)
			fs.Bots[i].CarryingPkg = -1
		}
	}
}

// --- Emergency System ---

// ToggleEmergency starts or ends emergency mode.
func ToggleEmergency(fs *FactoryState) {
	if fs.Emergency {
		// End emergency
		endEmergency(fs)
	} else {
		// Start emergency
		startEmergency(fs)
	}
}

func startEmergency(fs *FactoryState) {
	fs.Emergency = true
	fs.EmergencyTimer = 500

	// All bots stop current tasks and evacuate
	for i := range fs.Bots {
		bot := &fs.Bots[i]
		if bot.State == BotOffShift {
			continue // off-shift bots stay parked
		}
		fs.Tasks.RemoveAssignment(i)
		bot.CarryingPkg = -1
		bot.State = BotEmergencyEvac
	}
}

func endEmergency(fs *FactoryState) {
	fs.Emergency = false
	fs.EmergencyTimer = 0

	// Resume all bots
	for i := range fs.Bots {
		bot := &fs.Bots[i]
		if bot.State == BotEmergencyEvac {
			bot.State = BotIdle
		}
	}
}

// TickEmergency counts down emergency timer and auto-ends.
func TickEmergency(fs *FactoryState) {
	if !fs.Emergency {
		return
	}
	fs.EmergencyTimer--
	if fs.EmergencyTimer <= 0 {
		endEmergency(fs)
	}
}
