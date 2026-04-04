package factory

import (
	"fmt"
	"math"
	"strings"
	"swarmsim/locale"
)

// NewTask creates a task with given type, source, destination, priority, and part color.
func NewTask(typ TaskType, sx, sy, dx, dy float64, priority int, partColor int) Task {
	return Task{
		Type:      typ,
		SourceX:   sx,
		SourceY:   sy,
		DestX:     dx,
		DestY:     dy,
		Priority:  priority,
		Assigned:  -1,
		PartColor: partColor,
	}
}

// AddTask appends a task to the queue.
func (tq *TaskQueue) AddTask(t Task) {
	tq.Tasks = append(tq.Tasks, t)
}

// AssignTask finds the highest-priority unassigned task and assigns it to botIdx.
// Returns the task index or -1 if no tasks available.
func (tq *TaskQueue) AssignTask(botIdx int) int {
	bestIdx := -1
	bestPri := -1
	for i := range tq.Tasks {
		if tq.Tasks[i].Assigned < 0 && tq.Tasks[i].Priority > bestPri {
			bestPri = tq.Tasks[i].Priority
			bestIdx = i
		}
	}
	if bestIdx >= 0 {
		tq.Tasks[bestIdx].Assigned = botIdx
	}
	return bestIdx
}

// CompleteTask marks a task as done and removes it from the queue.
func (tq *TaskQueue) CompleteTask(idx int) {
	if idx < 0 || idx >= len(tq.Tasks) {
		return
	}
	// Swap-remove for O(1) deletion
	last := len(tq.Tasks) - 1
	if idx != last {
		tq.Tasks[idx] = tq.Tasks[last]
	}
	tq.Tasks = tq.Tasks[:last]
}

// RemoveAssignment unassigns a bot from its task (e.g. when bot needs to charge).
func (tq *TaskQueue) RemoveAssignment(botIdx int) {
	for i := range tq.Tasks {
		if tq.Tasks[i].Assigned == botIdx {
			tq.Tasks[i].Assigned = -1
			return
		}
	}
}

// findBotTask returns the task index assigned to the given bot, or -1.
func findBotTask(fs *FactoryState, botIdx int) int {
	for i := range fs.Tasks.Tasks {
		if fs.Tasks.Tasks[i].Assigned == botIdx {
			return i
		}
	}
	return -1
}

// taskExistsForTruck checks if any unfinished tasks reference the given truck's dock.
func taskExistsForTruck(fs *FactoryState, truckIdx int) bool {
	truck := &fs.Trucks[truckIdx]
	if truck.DockIdx < 0 || truck.DockIdx >= len(fs.Docks) {
		return false
	}
	dock := &fs.Docks[truck.DockIdx]
	cx := dock.X + dock.W/2
	cy := dock.Y + dock.H/2
	for i := range fs.Tasks.Tasks {
		t := &fs.Tasks.Tasks[i]
		if t.Type == TaskUnloadTruck || t.Type == TaskLoadTruck {
			// Match by source location (close enough)
			dx := t.SourceX - cx
			dy := t.SourceY - cy
			if dx*dx+dy*dy < 100 {
				return true
			}
		}
	}
	return false
}

// taskExistsForMachine checks if a transport task already exists for the given machine.
// isInput=true checks TaskTransportToMachine, false checks TaskTransportFromMachine.
func taskExistsForMachine(fs *FactoryState, machIdx int, isInput bool) bool {
	m := &fs.Machines[machIdx]
	cx := m.X + m.W/2
	cy := m.Y + m.H/2
	wantType := TaskTransportFromMachine
	if isInput {
		wantType = TaskTransportToMachine
	}
	for i := range fs.Tasks.Tasks {
		t := &fs.Tasks.Tasks[i]
		if t.Type != wantType {
			continue
		}
		// Match by destination (input) or source (output)
		var tx, ty float64
		if isInput {
			tx, ty = t.DestX, t.DestY
		} else {
			tx, ty = t.SourceX, t.SourceY
		}
		dx := tx - cx
		dy := ty - cy
		if dx*dx+dy*dy < 100 {
			return true
		}
	}
	return false
}

// maxPendingTasks caps the total task queue to prevent flooding.
const maxPendingTasks = 200

// countTasksForLocation counts tasks of given type near a source location.
func countTasksForLocation(fs *FactoryState, typ TaskType, sx, sy float64) int {
	count := 0
	for i := range fs.Tasks.Tasks {
		t := &fs.Tasks.Tasks[i]
		if t.Type == typ {
			dx := t.SourceX - sx
			dy := t.SourceY - sy
			if dx*dx+dy*dy < 100 {
				count++
			}
		}
	}
	return count
}

// GenerateTasks checks all structures and creates tasks as needed.
// Rate-limited: only runs every 50 ticks and caps total tasks.
func GenerateTasks(fs *FactoryState) {
	// Rate limit: only generate every 50 ticks
	if fs.Tick%50 != 0 {
		return
	}
	// Cap total tasks
	if len(fs.Tasks.Tasks) >= maxPendingTasks {
		return
	}

	// Trucks with packages -> TaskUnloadTruck (max 3 tasks per truck per cycle)
	for i := range fs.Trucks {
		truck := &fs.Trucks[i]
		if truck.Phase == TruckUnloading && len(truck.Parts) > 0 {
			if truck.DockIdx >= 0 && truck.DockIdx < len(fs.Docks) {
				dock := &fs.Docks[truck.DockIdx]
				if fs.QCStorageIdx < len(fs.Storage) {
					// Feature 6: Unload to QC area instead of main inbound
					qcArea := &fs.Storage[fs.QCStorageIdx]
					// Count existing unload tasks for this truck
					existing := countTasksForLocation(fs, TaskUnloadTruck, dock.X+dock.W/2, dock.Y+dock.H/2)
					toCreate := len(truck.Parts) - existing
					if toCreate > 3 {
						toCreate = 3 // max 3 new tasks per cycle
					}
					for j := 0; j < toCreate && len(fs.Tasks.Tasks) < maxPendingTasks; j++ {
						partColor := truck.Parts[j%len(truck.Parts)]
						fs.Tasks.AddTask(NewTask(TaskUnloadTruck,
							dock.X+dock.W/2, dock.Y+dock.H/2,
							qcArea.X+qcArea.W/2, qcArea.Y+qcArea.H/2,
							10, partColor))
					}
				} else if fs.InboundStorageIdx < len(fs.Storage) {
					// Fallback: only one storage area
					storage := &fs.Storage[fs.InboundStorageIdx]
					existing := countTasksForLocation(fs, TaskUnloadTruck, dock.X+dock.W/2, dock.Y+dock.H/2)
					toCreate := len(truck.Parts) - existing
					if toCreate > 3 {
						toCreate = 3
					}
					for j := 0; j < toCreate && len(fs.Tasks.Tasks) < maxPendingTasks; j++ {
						partColor := truck.Parts[j%len(truck.Parts)]
						fs.Tasks.AddTask(NewTask(TaskUnloadTruck,
							dock.X+dock.W/2, dock.Y+dock.H/2,
							storage.X+storage.W/2, storage.Y+storage.H/2,
							10, partColor))
					}
				}
			}
		}
	}

	// Machines needing input -> TaskTransportToMachine (Kanban pull: only when machine requests)
	for i := range fs.Machines {
		m := &fs.Machines[i]
		if m.NeedsInput {
			if fs.InboundStorageIdx < len(fs.Storage) {
				storage := &fs.Storage[fs.InboundStorageIdx]
				// Check Slots for matching parts (prefer Slots over legacy Parts)
				hasMatching := false
				for _, slot := range storage.Slots {
					if slot.PartColor == m.InputColor || m.InputColor == 0 {
						hasMatching = true
						break
					}
				}
				// Fallback to legacy Parts if Slots is empty but Parts has content
				if !hasMatching && len(storage.Slots) == 0 && len(storage.Parts) > 0 {
					hasMatching = true
				}
				if hasMatching && !taskExistsForMachine(fs, i, true) {
					fs.Tasks.AddTask(NewTask(TaskTransportToMachine,
						storage.X+storage.W/2, storage.Y+storage.H/2,
						m.X+m.W/2, m.Y+m.H/2,
						8, m.InputColor))
				}
			}
		}
	}

	// Machines with output -> TaskTransportFromMachine (with multi-step recipe routing)
	for i := range fs.Machines {
		m := &fs.Machines[i]
		if m.OutputReady && !taskExistsForMachine(fs, i, false) {
			// Check if this machine is part of a recipe and has a next step
			nextMachineIdx := findNextMachineInRecipe(fs, i, m.OutputColor)
			if nextMachineIdx >= 0 && nextMachineIdx < len(fs.Machines) {
				// Route to the next machine in the recipe
				nextM := &fs.Machines[nextMachineIdx]
				task := NewTask(TaskTransportFromMachine,
					m.X+m.W/2, m.Y+m.H/2,
					nextM.X+nextM.W/2, nextM.Y+nextM.H/2,
					8, m.OutputColor)
				task.PartRecipe = findRecipeIdx(fs, i)
				fs.Tasks.AddTask(task)
			} else if fs.OutboundStorageIdx < len(fs.Storage) {
				// Last step or no recipe: route to outbound storage
				outStorage := &fs.Storage[fs.OutboundStorageIdx]
				fs.Tasks.AddTask(NewTask(TaskTransportFromMachine,
					m.X+m.W/2, m.Y+m.H/2,
					outStorage.X+outStorage.W/2, outStorage.Y+outStorage.H/2,
					7, m.OutputColor))
			}
		}
	}

	// Feature 6: QC inspection — parts in QC buffer need inspection
	if fs.QCStorageIdx < len(fs.Storage) {
		qcArea := &fs.Storage[fs.QCStorageIdx]
		if len(qcArea.Slots) > 0 {
			existing := countTasksForLocation(fs, TaskIncomingQC, qcArea.X+qcArea.W/2, qcArea.Y+qcArea.H/2)
			toCreate := len(qcArea.Slots) - existing
			if toCreate > 3 {
				toCreate = 3
			}
			inbound := &fs.Storage[fs.InboundStorageIdx]
			for j := 0; j < toCreate && len(fs.Tasks.Tasks) < maxPendingTasks; j++ {
				if j < len(qcArea.Slots) {
					fs.Tasks.AddTask(NewTask(TaskIncomingQC,
						qcArea.X+qcArea.W/2, qcArea.Y+qcArea.H/2,
						inbound.X+inbound.W/2, inbound.Y+inbound.H/2,
						9, qcArea.Slots[j].PartColor))
				}
			}
		}
	}

	// Outbound storage has parts AND an outbound truck is parked at a dock -> TaskLoadTruck
	for ti := range fs.Trucks {
		truck := &fs.Trucks[ti]
		if truck.Direction == 1 && truck.Phase == TruckLoading && truck.DockIdx >= 0 && truck.DockIdx < len(fs.Docks) {
			dock := &fs.Docks[truck.DockIdx]
			outStorage := &fs.Storage[fs.OutboundStorageIdx]
			if len(outStorage.Slots) > 0 {
				existing := countTasksForLocation(fs, TaskLoadTruck, outStorage.X+outStorage.W/2, outStorage.Y+outStorage.H/2)
				toCreate := len(outStorage.Slots) - existing
				if toCreate > 3 {
					toCreate = 3
				}
				for j := 0; j < toCreate && len(fs.Tasks.Tasks) < maxPendingTasks; j++ {
					if j < len(outStorage.Slots) {
						partColor := outStorage.Slots[j].PartColor
						fs.Tasks.AddTask(NewTask(TaskLoadTruck,
							outStorage.X+outStorage.W/2, outStorage.Y+outStorage.H/2,
							dock.X+dock.W/2, dock.Y+dock.H/2,
							9, partColor))
					}
				}
			}
		}
	}
}

// validateTask checks whether a task's preconditions are still met in the world state.
// Returns true if the task is still valid and should be kept/assigned.
func validateTask(fs *FactoryState, t *Task) bool {
	switch t.Type {
	case TaskUnloadTruck:
		// Check if any truck at this source dock still has parts
		for _, truck := range fs.Trucks {
			if truck.Phase == TruckUnloading && len(truck.Parts) > 0 && truck.DockIdx >= 0 && truck.DockIdx < len(fs.Docks) {
				dock := &fs.Docks[truck.DockIdx]
				dx := (dock.X + dock.W/2) - t.SourceX
				dy := (dock.Y + dock.H/2) - t.SourceY
				if dx*dx+dy*dy < 100 {
					return true
				}
			}
		}
		return false

	case TaskLoadTruck:
		// Check outbound storage has parts AND a loading truck is at the dock
		if fs.OutboundStorageIdx >= len(fs.Storage) {
			return false
		}
		outStorage := &fs.Storage[fs.OutboundStorageIdx]
		if len(outStorage.Slots) == 0 && len(outStorage.Parts) == 0 {
			return false
		}
		for _, truck := range fs.Trucks {
			if truck.Phase == TruckLoading && truck.DockIdx >= 0 && truck.DockIdx < len(fs.Docks) {
				dock := &fs.Docks[truck.DockIdx]
				dx := (dock.X + dock.W/2) - t.DestX
				dy := (dock.Y + dock.H/2) - t.DestY
				if dx*dx+dy*dy < 100 {
					return true
				}
			}
		}
		return false

	case TaskTransportToMachine:
		// Check that inbound storage has parts of the right color
		for si := range fs.Storage {
			st := &fs.Storage[si]
			cx := st.X + st.W/2
			cy := st.Y + st.H/2
			dx := cx - t.SourceX
			dy := cy - t.SourceY
			if dx*dx+dy*dy < 100 {
				for _, slot := range st.Slots {
					if slot.PartColor == t.PartColor || t.PartColor == 0 {
						return true
					}
				}
				// Fallback to legacy Parts
				for _, p := range st.Parts {
					if p == t.PartColor || t.PartColor == 0 {
						return true
					}
				}
				return false
			}
		}
		return false

	case TaskTransportFromMachine:
		// Check machine has OutputReady
		for mi := range fs.Machines {
			m := &fs.Machines[mi]
			cx := m.X + m.W/2
			cy := m.Y + m.H/2
			dx := cx - t.SourceX
			dy := cy - t.SourceY
			if dx*dx+dy*dy < 100 {
				return m.OutputReady
			}
		}
		return false
	case TaskIncomingQC:
		// Check that QC area has parts
		if fs.QCStorageIdx < len(fs.Storage) {
			st := &fs.Storage[fs.QCStorageIdx]
			return len(st.Slots) > 0
		}
		return false
	}
	return true
}

// PruneStaleTasks removes tasks whose preconditions are no longer met.
// Called every 100 ticks to keep the task queue clean.
func PruneStaleTasks(fs *FactoryState) {
	if fs.Tick%100 != 0 {
		return
	}
	valid := fs.Tasks.Tasks[:0]
	for i := range fs.Tasks.Tasks {
		t := &fs.Tasks.Tasks[i]
		// Keep assigned tasks (bot is already working on them) OR valid unassigned tasks
		if t.Assigned >= 0 || validateTask(fs, t) {
			valid = append(valid, *t)
		}
	}
	fs.Tasks.Tasks = valid
}

// cancelTasksAtLocation cancels all unassigned tasks with a given source location,
// and unassigns bots that haven't picked up yet. This implements swarm communication:
// when a source is exhausted, nearby bots learn immediately.
func cancelTasksAtLocation(fs *FactoryState, sx, sy float64) {
	for i := len(fs.Tasks.Tasks) - 1; i >= 0; i-- {
		t := &fs.Tasks.Tasks[i]
		dx := t.SourceX - sx
		dy := t.SourceY - sy
		if dx*dx+dy*dy < 100 {
			if t.Assigned >= 0 {
				// Bot is heading to source but hasn't picked up yet — release it
				botIdx := t.Assigned
				if botIdx >= 0 && botIdx < len(fs.Bots) {
					bot := &fs.Bots[botIdx]
					if bot.State == BotMovingToSource {
						bot.State = BotIdle
						bot.CarryingPkg = -1
					} else {
						// Bot already picked up or is delivering — don't cancel
						continue
					}
				}
			}
			// Remove this task (swap-remove)
			last := len(fs.Tasks.Tasks) - 1
			if i != last {
				fs.Tasks.Tasks[i] = fs.Tasks.Tasks[last]
			}
			fs.Tasks.Tasks = fs.Tasks.Tasks[:last]
		}
	}
}

// AssignIdleBots assigns up to maxPerTick idle bots to pending tasks using smart dispatch.
// Express bots get highest-priority tasks first.
// Each idle bot picks the task with the best score = priority*10 - distance*0.1.
// Bots in the receiving zone prefer unload tasks; bots near machines prefer transport tasks.
// Never assign a task to a bot further than 1500px away.
func AssignIdleBots(fs *FactoryState, maxPerTick int) {
	assigned := 0

	// Two passes: first Express bots (high priority tasks), then everyone else
	for pass := 0; pass < 2; pass++ {
		for i := range fs.Bots {
			if assigned >= maxPerTick {
				break
			}

			// First pass: Express bots only; Second pass: non-Express
			if pass == 0 {
				if i >= len(fs.BotRoles) || fs.BotRoles[i] != RoleExpress {
					continue
				}
			} else {
				if i < len(fs.BotRoles) && fs.BotRoles[i] == RoleExpress {
					continue // already processed
				}
			}

			bot := &fs.Bots[i]
			// Skip off-shift bots, bots in emergency, or bots not idle
			if bot.State != BotIdle || bot.Energy <= 20 || findBotTask(fs, i) >= 0 {
				continue
			}
			// Skip off-duty bots
			if i < len(fs.ShiftOnDuty) && !fs.ShiftOnDuty[i] {
				continue
			}
			// Skip during emergency
			if fs.Emergency {
				continue
			}

			bestIdx := -1
			bestScore := -1e18

			// Determine bot zone preference
			inReceivingZone := bot.X >= ZoneReceivingX && bot.X <= ZoneReceivingX+ZoneReceivingW
			nearMachines := bot.X >= ZoneProductionX && bot.X <= ZoneProductionX+ZoneProductionW

			for j := range fs.Tasks.Tasks {
				t := &fs.Tasks.Tasks[j]
				if t.Assigned >= 0 {
					continue
				}

				// Validate task preconditions before assignment
				if !validateTask(fs, t) {
					continue
				}

				dx := t.SourceX - bot.X
				dy := t.SourceY - bot.Y
				dist := math.Sqrt(dx*dx + dy*dy)

				// Never assign if too far away
				if dist > 1500 {
					continue
				}

				score := float64(t.Priority)*10 - dist*0.1

				// Zone preference bonuses
				if inReceivingZone && (t.Type == TaskUnloadTruck || t.Type == TaskIncomingQC) {
					score += 50
				}
				if nearMachines && (t.Type == TaskTransportToMachine || t.Type == TaskTransportFromMachine) {
					score += 30
				}

				// Express bots prioritize highest-priority tasks
				if pass == 0 {
					score += float64(t.Priority) * 5
				}

				if score > bestScore {
					bestScore = score
					bestIdx = j
				}
			}

			if bestIdx >= 0 {
				fs.Tasks.Tasks[bestIdx].Assigned = i
				bot.State = BotMovingToSource
				assigned++
			}
		}
	}
}

// TickOrders manages customer order spawning and deadline checks.
func TickOrders(fs *FactoryState) {
	fs.OrderTimer--
	if fs.OrderTimer <= 0 {
		fs.OrderTimer = OrderSpawnInterval

		// Spawn a new customer order
		fs.NextOrderID++
		outputColor := 1 + fs.Rng.Intn(4) // colors 1-4
		qty := OrderMinQty + fs.Rng.Intn(OrderMaxQty-OrderMinQty+1)
		order := CustomerOrder{
			ID:          fs.NextOrderID,
			OutputColor: outputColor,
			Quantity:    qty,
			Fulfilled:   0,
			Deadline:    fs.Tick + OrderDeadlineTicks,
			Completed:   false,
		}
		fs.Orders = append(fs.Orders, order)
		colorNames := []string{"", locale.T("factory.part.red"), locale.T("factory.part.blue"), locale.T("factory.part.yellow"), locale.T("factory.part.green")}
		cname := "Mixed"
		if outputColor > 0 && outputColor < len(colorNames) {
			cname = colorNames[outputColor]
		}
		AddAlert(fs, locale.Tf("factory.alert.new_order", order.ID, qty, cname), [3]uint8{100, 200, 255})
	}

	// Check deadlines for overdue orders (no penalty, just visual)
	// Prune completed orders that are old (keep last 20)
	if len(fs.Orders) > 30 {
		alive := make([]CustomerOrder, 0, 20)
		for i := range fs.Orders {
			if !fs.Orders[i].Completed || fs.Tick-fs.Orders[i].Deadline < 2000 {
				alive = append(alive, fs.Orders[i])
			}
		}
		fs.Orders = alive
	}
}

// UpdateStats recalculates factory statistics.
func UpdateStats(fs *FactoryState) {
	fs.Stats.BotsIdle = 0
	fs.Stats.BotsWorking = 0
	fs.Stats.BotsCharging = 0
	fs.Stats.BotsRepairing = 0
	fs.Stats.BotsOffShift = 0
	wip := 0
	for i := range fs.Bots {
		switch fs.Bots[i].State {
		case BotIdle:
			fs.Stats.BotsIdle++
		case BotCharging:
			fs.Stats.BotsCharging++
		case BotRepairing:
			fs.Stats.BotsRepairing++
		case BotOffShift:
			fs.Stats.BotsOffShift++
		default:
			fs.Stats.BotsWorking++
		}
		// Count WIP: bots carrying parts
		if fs.Bots[i].CarryingPkg > 0 {
			wip++
		}
	}
	fs.Stats.WIP = wip

	// Feature 12: Bot productivity ranking — find top 3 bots by delivery count
	fs.Stats.TopWorkers = [3][2]int{}
	for i := range fs.BotDeliveries {
		d := fs.BotDeliveries[i]
		if d > fs.Stats.TopWorkers[2][1] {
			// Check if better than rank 3
			if d > fs.Stats.TopWorkers[0][1] {
				fs.Stats.TopWorkers[2] = fs.Stats.TopWorkers[1]
				fs.Stats.TopWorkers[1] = fs.Stats.TopWorkers[0]
				fs.Stats.TopWorkers[0] = [2]int{i, d}
			} else if d > fs.Stats.TopWorkers[1][1] {
				fs.Stats.TopWorkers[2] = fs.Stats.TopWorkers[1]
				fs.Stats.TopWorkers[1] = [2]int{i, d}
			} else {
				fs.Stats.TopWorkers[2] = [2]int{i, d}
			}
		}
	}

	// Feature 7: Inventory tracking — stock level warnings
	if fs.InboundStorageIdx < len(fs.Storage) {
		fs.StockWarning = len(fs.Storage[fs.InboundStorageIdx].Slots) < fs.MinStockLevel
	}
	// Outbound full warning
	if fs.OutboundStorageIdx < len(fs.Storage) {
		outStorage := &fs.Storage[fs.OutboundStorageIdx]
		fs.OutboundFull = len(outStorage.Slots) >= outStorage.MaxParts-2
	}
}

// GenerateStatsReport produces a text summary of the factory state for clipboard export.
func GenerateStatsReport(fs *FactoryState) string {
	var sb strings.Builder
	sb.WriteString("=== FACTORY REPORT ===\n")

	simMinutes := fs.Tick / 60
	simHours := simMinutes / 60
	sb.WriteString(fmt.Sprintf("Tick: %d | Uptime: %dh %dm\n", fs.Tick, simHours, simMinutes%60))

	sb.WriteString(fmt.Sprintf("Bots: %d (Working: %d, Idle: %d, Charging: %d, Off-Shift: %d)\n",
		len(fs.Bots), fs.Stats.BotsWorking, fs.Stats.BotsIdle, fs.Stats.BotsCharging, fs.Stats.BotsOffShift))

	// Defect rate
	defectRate := 0.0
	if fs.Stats.TotalParts > 0 {
		defectRate = float64(fs.Stats.DefectCount) / float64(fs.Stats.TotalParts) * 100
	}
	sb.WriteString(fmt.Sprintf("Parts Processed: %d | Defects: %d (%.1f%%)\n",
		fs.Stats.PartsProcessed, fs.Stats.DefectCount, defectRate))

	// OEE
	availability := 0.0
	if fs.Tick > 0 {
		totalUptime := 0
		for i := 0; i < len(fs.Stats.MachineUptime) && i < len(fs.Machines); i++ {
			totalUptime += fs.Stats.MachineUptime[i]
		}
		nMachines := len(fs.Machines)
		if nMachines == 0 {
			nMachines = 1
		}
		availability = float64(totalUptime) / float64(fs.Tick*nMachines)
		if availability > 1 {
			availability = 1
		}
	}
	quality := 1.0
	if fs.Stats.TotalParts > 0 {
		quality = float64(fs.Stats.GoodParts) / float64(fs.Stats.TotalParts)
	}
	performance := 0.0
	if fs.Tick > 0 {
		theoreticalMax := 0.0
		for i := range fs.Machines {
			if fs.Machines[i].ProcessTime > 0 {
				theoreticalMax += float64(fs.Tick) / float64(fs.Machines[i].ProcessTime)
			}
		}
		if theoreticalMax > 0 {
			performance = float64(fs.Stats.TotalParts) / theoreticalMax
		}
		if performance > 1 {
			performance = 1
		}
	}
	oee := availability * performance * quality * 100
	sb.WriteString(fmt.Sprintf("OEE: %.1f%% | Quality: %.1f%%\n", oee, quality*100))

	// Orders
	pending := 0
	for _, o := range fs.Orders {
		if !o.Completed {
			pending++
		}
	}
	sb.WriteString(fmt.Sprintf("Orders: %d completed, %d pending\n", fs.CompletedOrders, pending))

	sb.WriteString(fmt.Sprintf("Trucks: %d unloaded, %d loaded\n", fs.Stats.TrucksUnloaded, fs.Stats.TrucksLoaded))
	profit := fs.TotalRevenue - fs.TotalMaterialCost - fs.TotalEnergyCost
	sb.WriteString(fmt.Sprintf("QC Rejects: %d | Budget: $%.0f | Energy Cost: $%.0f\n",
		fs.Stats.IncomingRejects, fs.Budget, fs.TotalEnergyCost))
	sb.WriteString(fmt.Sprintf("Revenue: $%.0f | Material Cost: $%.0f | Profit: $%.0f\n",
		fs.TotalRevenue, fs.TotalMaterialCost, profit))

	// Bot roles
	var transporters, forklifts, express int
	for _, r := range fs.BotRoles {
		switch r {
		case RoleTransporter:
			transporters++
		case RoleForklift:
			forklifts++
		case RoleExpress:
			express++
		}
	}
	sb.WriteString(fmt.Sprintf("Roles: %d Transporter, %d Forklift, %d Express\n", transporters, forklifts, express))

	sb.WriteString(fmt.Sprintf("Score: %d\n", fs.Score))

	return sb.String()
}

// TickEfficiencySample samples the efficiency ratio every 200 ticks into the sparkline ring buffer.
func TickEfficiencySample(fs *FactoryState) {
	if fs.Tick-fs.EfficiencySampleTick < 200 {
		return
	}
	fs.EfficiencySampleTick = fs.Tick

	working := fs.Stats.BotsWorking
	idle := fs.Stats.BotsIdle
	total := working + idle
	eff := 0
	if total > 0 {
		eff = working * 100 / total
	}

	idx := fs.EfficiencyIdx % len(fs.EfficiencyHistory)
	fs.EfficiencyHistory[idx] = eff
	fs.EfficiencyIdx++
}

// HasAchievement returns true if the given achievement ID has been unlocked.
func HasAchievement(fs *FactoryState, id string) bool {
	for _, a := range fs.FactoryAchievements {
		if a == id {
			return true
		}
	}
	return false
}

// UnlockAchievement unlocks an achievement and shows the popup.
func UnlockAchievement(fs *FactoryState, id, displayText string) {
	if HasAchievement(fs, id) {
		return
	}
	fs.FactoryAchievements = append(fs.FactoryAchievements, id)
	fs.AchievementPopup = displayText
	fs.AchievementTimer = 100
	fs.Score += 100
	AddAlert(fs, locale.T("factory.achieve.prefix")+displayText, [3]uint8{255, 215, 0})
}

// TickAchievements checks for factory achievement milestones.
func TickAchievements(fs *FactoryState) {
	// Tick down popup timer
	if fs.AchievementTimer > 0 {
		fs.AchievementTimer--
	}

	// Only check achievements every 50 ticks to save CPU
	if fs.Tick%50 != 0 {
		return
	}

	// "First Delivery!" -- first part processed
	if fs.Stats.PartsProcessed >= 1 {
		UnlockAchievement(fs, "first_delivery", locale.T("factory.achieve.first_delivery"))
	}

	// "100 Parts!" -- 100 parts processed
	if fs.Stats.PartsProcessed >= 100 {
		UnlockAchievement(fs, "100_parts", locale.T("factory.achieve.100_parts"))
	}

	// "Full House!" -- all machines running simultaneously
	allRunning := true
	for i := range fs.Machines {
		if !fs.Machines[i].Active {
			allRunning = false
			break
		}
	}
	if allRunning && len(fs.Machines) > 0 {
		UnlockAchievement(fs, "full_house", locale.T("factory.achieve.full_house"))
	}

	// "Speed Demon!" -- complete an order before deadline/2
	for i := range fs.Orders {
		o := &fs.Orders[i]
		if o.Completed {
			halfDeadline := o.Deadline - OrderDeadlineTicks/2
			if fs.Tick < halfDeadline && !HasAchievement(fs, "speed_demon") {
				UnlockAchievement(fs, "speed_demon", locale.T("factory.achieve.speed_demon"))
			}
		}
	}
}

// --- Multi-Step Production Recipe Helpers ---

// findRecipeIdx returns the recipe index that contains the given machine as a step, or -1.
func findRecipeIdx(fs *FactoryState, machIdx int) int {
	for ri, r := range fs.Recipes {
		for _, step := range r.Steps {
			if step == machIdx {
				return ri
			}
		}
	}
	return -1
}

// findNextMachineInRecipe finds the next machine in a recipe sequence after the given machine.
// It uses the machine's output color to match a recipe. Returns -1 if no next step.
func findNextMachineInRecipe(fs *FactoryState, machIdx int, outputColor int) int {
	for _, r := range fs.Recipes {
		for si, step := range r.Steps {
			if step == machIdx && si+1 < len(r.Steps) {
				return r.Steps[si+1]
			}
		}
	}
	return -1
}

// --- Feature 11: Random Events System ---

// Event type constants
const (
	EventSupplyShortage = 0
	EventRushOrder      = 1
	EventPowerOutage    = 2
	EventMachineBreak   = 3
	EventEfficiencyBonus = 4
	EventInspection     = 5
)

// TickEvents manages random factory events.
func TickEvents(fs *FactoryState) {
	// Tick active event
	if fs.ActiveEvent.Active {
		fs.ActiveEvent.Timer--
		if fs.ActiveEvent.Timer <= 0 {
			// Event ended — clean up
			if fs.ActiveEvent.Type == EventMachineBreak && fs.EventBreakdownMachine >= 0 {
				// Re-enable broken machine
				if fs.EventBreakdownMachine < len(fs.Machines) {
					fs.Machines[fs.EventBreakdownMachine].CoolingDown = false
				}
				fs.EventBreakdownMachine = -1
			}
			fs.ActiveEvent.Active = false
			AddAlert(fs, locale.T("factory.event.ended"), [3]uint8{100, 200, 100})
		}
	}

	// Spawn new events
	fs.EventTimer--
	if fs.EventTimer <= 0 && !fs.ActiveEvent.Active {
		fs.EventTimer = 3000 + fs.Rng.Intn(5000) // next event in 3000-8000 ticks
		// Pick random event
		eventType := fs.Rng.Intn(6)
		durations := []int{3000, 2000, 500, 1000, 1000, 500}
		fs.ActiveEvent = FactoryEvent{Type: eventType, Timer: durations[eventType], Active: true}
		names := []string{
			locale.T("factory.event.supply_shortage"), locale.T("factory.event.rush_order"),
			locale.T("factory.event.power_outage"), locale.T("factory.event.machine_breakdown"),
			locale.T("factory.event.efficiency_bonus"), locale.T("factory.event.inspection_visit"),
		}
		colors := [][3]uint8{{220, 60, 60}, {255, 200, 40}, {220, 100, 30}, {200, 40, 40}, {40, 200, 40}, {200, 200, 40}}
		AddAlert(fs, "EVENT: "+names[eventType], colors[eventType])

		// Apply one-time event start effects
		switch eventType {
		case EventRushOrder:
			// Spawn a special rush order worth 3x
			fs.NextOrderID++
			outputColor := 1 + fs.Rng.Intn(4)
			qty := OrderMinQty + fs.Rng.Intn(3) // small qty but tight deadline
			order := CustomerOrder{
				ID:          fs.NextOrderID,
				OutputColor: outputColor,
				Quantity:    qty,
				Fulfilled:   0,
				Deadline:    fs.Tick + 2000, // tight deadline
				Completed:   false,
			}
			fs.Orders = append(fs.Orders, order)
			fs.Score += qty * 3 // 3x bonus score potential
			AddAlert(fs, locale.Tf("factory.alert.rush_order", order.ID, qty), [3]uint8{255, 200, 40})

		case EventMachineBreak:
			// Disable a random machine
			if len(fs.Machines) > 0 {
				mIdx := fs.Rng.Intn(len(fs.Machines))
				fs.Machines[mIdx].CoolingDown = true
				fs.Machines[mIdx].Active = false
				fs.EventBreakdownMachine = mIdx
				machNames := []string{locale.T("factory.machine.cnc1"), locale.T("factory.machine.cnc2"), locale.T("factory.machine.assembly"), locale.T("factory.machine.drill1"), locale.T("factory.machine.drill2"), locale.T("factory.machine.qcfinal")}
				mname := fmt.Sprintf("Machine %d", mIdx+1)
				if mIdx < len(machNames) {
					mname = machNames[mIdx]
				}
				AddAlert(fs, locale.Tf("factory.alert.machine_breakdown", mname), [3]uint8{200, 40, 40})
			}
		}
	}
}

// IsEventActive returns true if the given event type is currently active.
func IsEventActive(fs *FactoryState, eventType int) bool {
	return fs.ActiveEvent.Active && fs.ActiveEvent.Type == eventType
}
