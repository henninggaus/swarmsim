package factory

import (
	"fmt"
	"swarmsim/locale"
)

// TickMachines updates all machines — counts down processing timers, marks output ready.
// Includes quality control: 5% chance of defect (OutputColor = 5).
func TickMachines(fs *FactoryState) {
	// Pause machines during emergency
	if fs.Emergency {
		return
	}

	for i := range fs.Machines {
		m := &fs.Machines[i]

		// Kanban pull: machine requests input when buffer drops below 2
		m.NeedsInput = m.CurrentInput < 2 && !m.OutputReady

		// Track uptime for KPI
		if m.Active && i < len(fs.Stats.MachineUptime) {
			fs.Stats.MachineUptime[i]++
		}

		// Feature 4: Machine overheating
		if m.Active {
			m.Temperature += 0.05
			if m.Temperature >= 100 {
				m.CoolingDown = true
				m.Active = false // pause processing
				machNames := []string{locale.T("factory.machine.cnc1"), locale.T("factory.machine.cnc2"), locale.T("factory.machine.assembly"), locale.T("factory.machine.drill1"), locale.T("factory.machine.drill2"), locale.T("factory.machine.qcfinal")}
				mname := fmt.Sprintf("Machine %d", i+1)
				if i < len(machNames) {
					mname = machNames[i]
				}
				AddAlert(fs, locale.Tf("factory.alert.overheating", mname), [3]uint8{220, 60, 30})
			}
		} else {
			if m.Temperature > 0 {
				m.Temperature -= 0.2
				if m.Temperature < 0 {
					m.Temperature = 0
				}
			}
			if m.CoolingDown && m.Temperature < 30 {
				m.CoolingDown = false
				machNames := []string{locale.T("factory.machine.cnc1"), locale.T("factory.machine.cnc2"), locale.T("factory.machine.assembly"), locale.T("factory.machine.drill1"), locale.T("factory.machine.drill2"), locale.T("factory.machine.qcfinal")}
				mname := fmt.Sprintf("Machine %d", i+1)
				if i < len(machNames) {
					mname = machNames[i]
				}
				AddAlert(fs, locale.Tf("factory.alert.cooled", mname), [3]uint8{60, 200, 60})
			}
		}

		// Feature 7b: Machine power consumption while active
		if m.Active && m.ProcessTimer > 0 {
			cost := m.PowerCostPerTick
			if cost <= 0 {
				cost = 0.02
			}
			fs.Budget -= cost
			fs.TotalEnergyCost += cost
		}

		// If actively processing, count down (skip if cooling down)
		if m.Active && m.ProcessTimer > 0 && !m.CoolingDown {
			m.ProcessTimer--
			if m.ProcessTimer <= 0 {
				// Done processing
				m.Active = false
				m.OutputReady = true

				// Quality control: 5% defect rate
				if fs.Rng.Float64() < 0.05 {
					m.OutputColor = 5 // defective
				}

				fs.Stats.TotalParts++
				if m.OutputColor != 5 {
					fs.Stats.GoodParts++
				}

				// Alert for machine output
				machNames := []string{locale.T("factory.machine.cnc1"), locale.T("factory.machine.cnc2"), locale.T("factory.machine.assembly"), locale.T("factory.machine.drill1"), locale.T("factory.machine.drill2"), locale.T("factory.machine.qcfinal")}
				mname := fmt.Sprintf("Machine %d", i+1)
				if i < len(machNames) {
					mname = machNames[i]
				}
				// Only alert occasionally (not every single completion)
				if fs.Tick%500 < 50 {
					AddAlert(fs, locale.Tf("factory.alert.output_ready", mname), [3]uint8{80, 255, 120})
				}
			}
		}

		// If not active, not outputting, has input, and NOT cooling down, start processing
		if !m.Active && !m.OutputReady && !m.CoolingDown && m.CurrentInput > 0 {
			m.Active = true
			m.ProcessTimer = m.ProcessTime
			m.CurrentInput--

			// Reset output color to normal for this production run
			// (the defect check happens when processing finishes)
		}
	}
}

// FeedMachine adds a part to a machine's input. Returns true if accepted.
func FeedMachine(m *Machine) bool {
	if m.CurrentInput >= m.MaxInput {
		return false
	}
	m.CurrentInput++
	return true
}

// CollectOutput takes the finished product from a machine. Returns true if collected.
func CollectOutput(m *Machine) bool {
	if !m.OutputReady {
		return false
	}
	m.OutputReady = false
	return true
}
