package factory

import (
	"swarmsim/locale"
)

// Bot states stored in SwarmBot.State
const (
	BotIdle           = 0 // waiting for task
	BotMovingToSource = 1 // navigating to pickup location
	BotPickingUp      = 2 // at source, picking up
	BotMovingToDest   = 3 // carrying item to destination
	BotDelivering     = 4 // at destination, dropping off
	BotCharging       = 5 // at charger, charging
	BotRepairing      = 6 // at workshop, being repaired
	BotOffShift       = 7 // off duty, parked along walls
	BotEmergencyEvac  = 8 // evacuating to yard during emergency
)

// TickBotBehavior updates all bot movement and task execution.
func TickBotBehavior(fs *FactoryState) {
	// Grow per-bot slices if needed
	for len(fs.HonkFlash) < len(fs.Bots) {
		fs.HonkFlash = append(fs.HonkFlash, 0)
	}
	for len(fs.BotCounters) < len(fs.Bots) {
		fs.BotCounters = append(fs.BotCounters, 0)
	}
	for len(fs.BotPallet) < len(fs.Bots) {
		fs.BotPallet = append(fs.BotPallet, nil)
	}

	// Decrement honk flash timers
	for i := range fs.HonkFlash {
		if fs.HonkFlash[i] > 0 {
			fs.HonkFlash[i]--
		}
	}

	for i := range fs.Bots {
		bot := &fs.Bots[i]

		// Off-shift bots: stay parked
		if bot.State == BotOffShift {
			bot.Speed = 0
			continue
		}

		// Emergency evacuation: navigate to yard
		if bot.State == BotEmergencyEvac {
			// Navigate toward nearest gate (yard area, Y < YardH)
			evacuateTarget := findNearestGateTarget(fs, bot)
			navigateToIdx(bot, i, evacuateTarget[0], evacuateTarget[1], fs)
			continue
		}

		// Critical energy: stop completely
		if bot.Energy < EnergyCritical && bot.State != BotCharging && bot.State != BotRepairing {
			bot.Speed = 0
			continue
		}

		// Charging and repairing are handled by their respective tick functions
		if bot.State == BotCharging || bot.State == BotRepairing {
			continue
		}

		// Bot communication: decrement charger busy delay
		if i < len(fs.ChargerBusyDelay) && fs.ChargerBusyDelay[i] > 0 {
			fs.ChargerBusyDelay[i]--
		}

		taskIdx := findBotTask(fs, i)
		if taskIdx < 0 {
			// No task: wander slowly
			botWander(bot, i, fs)

			// Bot communication: if idle and see 3+ bots ahead, broadcast congestion
			broadcastCongestion(fs, i)
			continue
		}

		task := &fs.Tasks.Tasks[taskIdx]

		switch bot.State {
		case BotIdle, BotMovingToSource:
			bot.State = BotMovingToSource
			if navigateToIdx(bot, i, task.SourceX, task.SourceY, fs) {
				bot.State = BotPickingUp
			}

		case BotPickingUp:
			// Attempt pickup — verify source still has material
			pickupOK := true
			switch task.Type {
			case TaskUnloadTruck:
				if !removeTruckPartCheck(fs, task) {
					pickupOK = false
				}
			case TaskTransportToMachine:
				if !removeFromStorageCheck(fs, task.SourceX, task.SourceY, task.PartColor) {
					pickupOK = false
				}
			case TaskTransportFromMachine:
				if !collectFromMachineCheck(fs, task.SourceX, task.SourceY) {
					pickupOK = false
				}
			case TaskLoadTruck:
				if !removeFromStorageCheck(fs, task.SourceX, task.SourceY, task.PartColor) {
					pickupOK = false
				}
			case TaskReturnDefect:
				if !removeFromStorageCheck(fs, task.SourceX, task.SourceY, task.PartColor) {
					pickupOK = false
				}
			case TaskIncomingQC:
				if !removeFromStorageCheck(fs, task.SourceX, task.SourceY, task.PartColor) {
					pickupOK = false
				}
			}

			if !pickupOK {
				// Source is empty — cancel this task and release the bot
				fs.Tasks.CompleteTask(taskIdx)
				bot.State = BotIdle
				bot.CarryingPkg = -1
				// Swarm communication: cancel all tasks at this exhausted source
				cancelTasksAtLocation(fs, task.SourceX, task.SourceY)
				continue
			}

			bot.CarryingPkg = task.PartColor
			bot.State = BotMovingToDest

			// --- Feature 5: Pallet Transport — forklifts pick up extra parts ---
			if i < len(fs.BotRoles) && fs.BotRoles[i] == RoleForklift && i < len(fs.BotPallet) {
				fs.BotPallet[i] = []int{task.PartColor} // first part already picked up
				// Try to pick up 3 more matching parts from the same source
				for extra := 0; extra < 3; extra++ {
					extraOK := false
					switch task.Type {
					case TaskUnloadTruck:
						extraOK = removeTruckPartCheck(fs, task)
					case TaskTransportToMachine:
						extraOK = removeFromStorageCheck(fs, task.SourceX, task.SourceY, task.PartColor)
					case TaskTransportFromMachine:
						extraOK = collectFromMachineCheck(fs, task.SourceX, task.SourceY)
					case TaskLoadTruck:
						extraOK = removeFromStorageCheck(fs, task.SourceX, task.SourceY, task.PartColor)
					case TaskIncomingQC:
						extraOK = removeFromStorageCheck(fs, task.SourceX, task.SourceY, task.PartColor)
					}
					if extraOK {
						fs.BotPallet[i] = append(fs.BotPallet[i], task.PartColor)
					} else {
						break
					}
				}
			}

			// Spawn pulse effect at pickup location
			fs.PulseEffects = append(fs.PulseEffects, PulseEffect{
				X: bot.X, Y: bot.Y, MaxR: 25, Color: [3]uint8{0, 220, 255},
			})

			// After successful pickup, check if source is now empty -> broadcast
			switch task.Type {
			case TaskUnloadTruck:
				if truckEmptyAtDock(fs, task.SourceX, task.SourceY) {
					cancelTasksAtLocation(fs, task.SourceX, task.SourceY)
				}
			case TaskTransportToMachine, TaskLoadTruck, TaskReturnDefect, TaskIncomingQC:
				if storageEmptyAt(fs, task.SourceX, task.SourceY) {
					cancelTasksAtLocation(fs, task.SourceX, task.SourceY)
				}
			case TaskTransportFromMachine:
				if machineOutputCollectedAt(fs, task.SourceX, task.SourceY) {
					cancelTasksAtLocation(fs, task.SourceX, task.SourceY)
				}
			}

		case BotMovingToDest:
			if navigateToIdx(bot, i, task.DestX, task.DestY, fs) {
				bot.State = BotDelivering
			}

		case BotDelivering:
			// --- Feature 2: Queue Buffers — wait at machine if input is full ---
			if task.Type == TaskTransportToMachine {
				machFull := false
				for mi := range fs.Machines {
					m := &fs.Machines[mi]
					cx := m.X + m.W/2
					cy := m.Y + m.H/2
					ddx := task.DestX - cx
					ddy := task.DestY - cy
					if ddx*ddx+ddy*ddy < 100 {
						if m.CurrentInput >= m.MaxInput {
							machFull = true
						}
						break
					}
				}
				if machFull {
					bot.Speed = 0
					if i < len(fs.BotCounters) {
						fs.BotCounters[i]++
						if fs.BotCounters[i] > 100 {
							// Waited too long, abandon task
							fs.BotCounters[i] = 0
							fs.Tasks.RemoveAssignment(i)
							bot.State = BotIdle
							bot.CarryingPkg = -1
							continue
						}
					}
					continue // wait at current position
				}
				// Reset counter on successful delivery
				if i < len(fs.BotCounters) {
					fs.BotCounters[i] = 0
				}
			}

			// Check for QC-Final defect rejection
			if task.Type == TaskTransportFromMachine && bot.CarryingPkg == 5 {
				// Defective part: redirect to inbound storage instead
				if fs.InboundStorageIdx < len(fs.Storage) {
					inbound := &fs.Storage[fs.InboundStorageIdx]
					addToStorage(fs, inbound.X+inbound.W/2, inbound.Y+inbound.H/2, bot.CarryingPkg)
				}
				fs.Stats.DefectCount++
				bot.CarryingPkg = -1
				bot.State = BotIdle
				fs.Tasks.CompleteTask(taskIdx)
				fs.PulseEffects = append(fs.PulseEffects, PulseEffect{
					X: bot.X, Y: bot.Y, MaxR: 30, Color: [3]uint8{255, 60, 60},
				})
				continue
			}

			// Deliver (instant)
			switch task.Type {
			case TaskIncomingQC:
				// Feature 6: 8% chance of rejection (Feature 11: 16% during Inspection Visit)
				rejectRate := 0.08
				if IsEventActive(fs, EventInspection) {
					rejectRate = 0.16
				}
				if fs.Rng.Float64() < rejectRate {
					// Part rejected - discard
					fs.Stats.IncomingRejects++
					AddAlert(fs, locale.T("factory.alert.qc_reject"), [3]uint8{220, 60, 60})
					// Add QC reject effect for rendering
					if fs.QCStorageIdx < len(fs.Storage) {
						qcArea := &fs.Storage[fs.QCStorageIdx]
						fs.QCRejectEffects = append(fs.QCRejectEffects, QCRejectEffect{
							X: qcArea.X + qcArea.W/2, Y: qcArea.Y + qcArea.H/2, Tick: fs.Tick,
						})
					}
				} else {
					// Part passes - add to inbound storage
					addToStorage(fs, task.DestX, task.DestY, bot.CarryingPkg)
				}
			case TaskUnloadTruck, TaskReturnDefect:
				// Add to storage
				addToStorage(fs, task.DestX, task.DestY, bot.CarryingPkg)
			case TaskTransportFromMachine:
				// Multi-step production: check if destination is another machine
				fedMachine := false
				if task.PartRecipe >= 0 {
					fedMachine = feedMachineAt(fs, task.DestX, task.DestY)
				}
				if !fedMachine {
					addToStorage(fs, task.DestX, task.DestY, bot.CarryingPkg)
				}
			case TaskTransportToMachine:
				// Feed machine
				feedMachineAt(fs, task.DestX, task.DestY)
			case TaskLoadTruck:
				// Add to truck
				addToTruck(fs, task)
			}

			// --- Feature 5: Pallet delivery — forklifts deposit all pallet parts ---
			if i < len(fs.BotRoles) && fs.BotRoles[i] == RoleForklift && i < len(fs.BotPallet) && len(fs.BotPallet[i]) > 1 {
				// Deliver extra pallet parts (first part already handled above)
				for pi := 1; pi < len(fs.BotPallet[i]); pi++ {
					pColor := fs.BotPallet[i][pi]
					switch task.Type {
					case TaskIncomingQC:
						// QC check per pallet part too
						if fs.Rng.Float64() < 0.08 {
							fs.Stats.IncomingRejects++
						} else {
							addToStorage(fs, task.DestX, task.DestY, pColor)
						}
					case TaskUnloadTruck, TaskReturnDefect:
						addToStorage(fs, task.DestX, task.DestY, pColor)
					case TaskTransportFromMachine:
						fedM := false
						if task.PartRecipe >= 0 {
							fedM = feedMachineAt(fs, task.DestX, task.DestY)
						}
						if !fedM {
							addToStorage(fs, task.DestX, task.DestY, pColor)
						}
					case TaskTransportToMachine:
						feedMachineAt(fs, task.DestX, task.DestY)
					case TaskLoadTruck:
						addToTruck(fs, task)
					}
					fs.Stats.PartsProcessed++
				}
				fs.BotPallet[i] = nil
			}

			isFullCycle := task.Type == TaskLoadTruck
			// Feature: Revenue — earn money when loading outbound truck
			if isFullCycle {
				fs.TotalRevenue += fs.RevenuePerProduct
				fs.Budget += fs.RevenuePerProduct
			}
			bot.CarryingPkg = -1
			bot.State = BotIdle
			fs.Tasks.CompleteTask(taskIdx)
			fs.Stats.PartsProcessed++
			if isFullCycle {
				fs.Stats.CompletedCycles++
			}
			// Track delivery count for experience
			if i < len(fs.BotDeliveries) {
				fs.BotDeliveries[i]++
			}
			// Check if this delivery fulfills an order
			checkOrderFulfillment(fs, task.PartColor)
			// Spawn pulse effect at delivery location
			fs.PulseEffects = append(fs.PulseEffects, PulseEffect{
				X: bot.X, Y: bot.Y, MaxR: 30, Color: [3]uint8{80, 255, 120},
			})
		}
	}
}


// --- Helper functions for task interactions ---

// removeTruckPartCheck removes a part from the truck and returns true, or false if no part available.
func removeTruckPartCheck(fs *FactoryState, task *Task) bool {
	for i := range fs.Trucks {
		truck := &fs.Trucks[i]
		if truck.DockIdx < 0 || truck.DockIdx >= len(fs.Docks) {
			continue
		}
		dock := &fs.Docks[truck.DockIdx]
		dx := task.SourceX - (dock.X + dock.W/2)
		dy := task.SourceY - (dock.Y + dock.H/2)
		if dx*dx+dy*dy < 100 && len(truck.Parts) > 0 {
			truck.Parts = truck.Parts[:len(truck.Parts)-1]
			return true
		}
	}
	return false
}

// removeFromStorageCheck removes a matching part from storage and returns true, or false if none found.
func removeFromStorageCheck(fs *FactoryState, sx, sy float64, partColor int) bool {
	for i := range fs.Storage {
		st := &fs.Storage[i]
		cx := st.X + st.W/2
		cy := st.Y + st.H/2
		dx := sx - cx
		dy := sy - cy
		if dx*dx+dy*dy < 200*200 {
			// Try FIFO removal from Slots
			bestIdx := -1
			bestTick := int(1e9)
			for j := range st.Slots {
				if (st.Slots[j].PartColor == partColor || partColor == 0) && st.Slots[j].Tick < bestTick {
					bestTick = st.Slots[j].Tick
					bestIdx = j
				}
			}
			if bestIdx >= 0 {
				st.Slots = append(st.Slots[:bestIdx], st.Slots[bestIdx+1:]...)
				// Also maintain legacy Parts slice
				for j := range st.Parts {
					if st.Parts[j] == partColor || partColor == 0 {
						st.Parts = append(st.Parts[:j], st.Parts[j+1:]...)
						break
					}
				}
				return true
			}
			// Fallback: remove from legacy Parts
			if len(st.Parts) > 0 {
				for j := range st.Parts {
					if st.Parts[j] == partColor || partColor == 0 {
						st.Parts = append(st.Parts[:j], st.Parts[j+1:]...)
						return true
					}
				}
			}
			return false // storage found but no matching part
		}
	}
	return false
}

// collectFromMachineCheck collects output from a machine and returns true, or false if not ready.
func collectFromMachineCheck(fs *FactoryState, sx, sy float64) bool {
	for i := range fs.Machines {
		m := &fs.Machines[i]
		cx := m.X + m.W/2
		cy := m.Y + m.H/2
		dx := sx - cx
		dy := sy - cy
		if dx*dx+dy*dy < 100 {
			return CollectOutput(m)
		}
	}
	return false
}

// truckEmptyAtDock returns true if the truck at the given dock location has no more parts.
func truckEmptyAtDock(fs *FactoryState, sx, sy float64) bool {
	for i := range fs.Trucks {
		truck := &fs.Trucks[i]
		if truck.DockIdx < 0 || truck.DockIdx >= len(fs.Docks) {
			continue
		}
		dock := &fs.Docks[truck.DockIdx]
		dx := sx - (dock.X + dock.W/2)
		dy := sy - (dock.Y + dock.H/2)
		if dx*dx+dy*dy < 100 {
			return len(truck.Parts) == 0
		}
	}
	return true // no truck found = effectively empty
}

// storageEmptyAt returns true if the storage area near (sx, sy) has no slots left.
func storageEmptyAt(fs *FactoryState, sx, sy float64) bool {
	for i := range fs.Storage {
		st := &fs.Storage[i]
		cx := st.X + st.W/2
		cy := st.Y + st.H/2
		dx := sx - cx
		dy := sy - cy
		if dx*dx+dy*dy < 200*200 {
			return len(st.Slots) == 0 && len(st.Parts) == 0
		}
	}
	return true
}

// machineOutputCollectedAt returns true if the machine near (sx, sy) has no output ready.
func machineOutputCollectedAt(fs *FactoryState, sx, sy float64) bool {
	for i := range fs.Machines {
		m := &fs.Machines[i]
		cx := m.X + m.W/2
		cy := m.Y + m.H/2
		dx := sx - cx
		dy := sy - cy
		if dx*dx+dy*dy < 100 {
			return !m.OutputReady
		}
	}
	return true
}

func addToStorage(fs *FactoryState, dx, dy float64, partColor int) {
	for i := range fs.Storage {
		st := &fs.Storage[i]
		cx := st.X + st.W/2
		cy := st.Y + st.H/2
		ddx := dx - cx
		ddy := dy - cy
		if ddx*ddx+ddy*ddy < 200*200 {
			if len(st.Parts) < st.MaxParts {
				st.Parts = append(st.Parts, partColor)
				st.Slots = append(st.Slots, StorageSlot{
					PartColor: partColor,
					Tick:      fs.Tick,
				})
			}
			return
		}
	}
}

func feedMachineAt(fs *FactoryState, dx, dy float64) bool {
	for i := range fs.Machines {
		m := &fs.Machines[i]
		cx := m.X + m.W/2
		cy := m.Y + m.H/2
		ddx := dx - cx
		ddy := dy - cy
		if ddx*ddx+ddy*ddy < 100 {
			return FeedMachine(m)
		}
	}
	return false
}

func addToTruck(fs *FactoryState, task *Task) {
	for i := range fs.Trucks {
		truck := &fs.Trucks[i]
		if truck.Phase != TruckLoading {
			continue
		}
		if truck.DockIdx < 0 || truck.DockIdx >= len(fs.Docks) {
			continue
		}
		dock := &fs.Docks[truck.DockIdx]
		dx := task.DestX - (dock.X + dock.W/2)
		dy := task.DestY - (dock.Y + dock.H/2)
		if dx*dx+dy*dy < 100 {
			truck.Parts = append(truck.Parts, task.PartColor)
			return
		}
	}
}

// AddAlert adds a new alert to the factory state.
func AddAlert(fs *FactoryState, message string, col [3]uint8) {
	fs.Alerts = append(fs.Alerts, FactoryAlert{
		Message: message,
		Tick:    fs.Tick,
		Color:   col,
	})
	// Keep only the last MaxVisibleAlerts*2 alerts (old ones get pruned in rendering)
	if len(fs.Alerts) > MaxVisibleAlerts*3 {
		fs.Alerts = fs.Alerts[len(fs.Alerts)-MaxVisibleAlerts*2:]
	}
}

// addToStorageDirect adds a part directly to a specific storage area.
func addToStorageDirect(st *StorageArea, partColor int, tick int) {
	if len(st.Parts) < st.MaxParts {
		st.Parts = append(st.Parts, partColor)
		st.Slots = append(st.Slots, StorageSlot{
			PartColor: partColor,
			Tick:      tick,
		})
	}
}
