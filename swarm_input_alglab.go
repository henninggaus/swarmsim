package main

import (
	"fmt"
	"swarmsim/domain/swarm"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
	"swarmsim/render"
)

func (g *Game) handleBlockEditorClick(mx, my int) {
	ss := g.sim.SwarmState

	// If dropdown is open, check for dropdown click first
	if ss.ActiveDropdown != nil {
		idx := render.BlockDropdownHitTest(mx, my, ss.ActiveDropdown)
		if idx >= 0 {
			g.applyBlockDropdownSelection(idx)
		}
		ss.ActiveDropdown = nil
		return
	}

	action, ri, ci := render.BlockEditorHitTest(mx, my, ss)
	switch action {
	case "sensor":
		// Open sensor dropdown
		items := flattenGroups(swarmscript.SensorGrouped)
		ddX := blockEditorDropdownX("sensor")
		ddY := my
		ss.ActiveDropdown = &swarm.BlockDropdown{
			RuleIdx: ri, CondIdx: ci, FieldType: "sensor",
			X: ddX, Y: ddY, Items: items, HoverIdx: -1,
		}

	case "op":
		items := []string{">", "<", "=="}
		ss.ActiveDropdown = &swarm.BlockDropdown{
			RuleIdx: ri, CondIdx: ci, FieldType: "op",
			X: blockEditorDropdownX("op"), Y: my, Items: items, HoverIdx: -1,
		}

	case "value":
		ss.BlockValueEdit = true
		ss.BlockValueRuleIdx = ri
		ss.BlockValueCondIdx = ci
		ss.BlockValueText = fmt.Sprintf("%d", ss.BlockRules[ri].Conditions[ci].Value)
		ss.Editor.Focused = false

	case "action":
		items := flattenGroups(swarmscript.ActionGrouped)
		ss.ActiveDropdown = &swarm.BlockDropdown{
			RuleIdx: ri, CondIdx: -1, FieldType: "action",
			X: blockEditorDropdownX("action"), Y: my, Items: items, HoverIdx: -1,
		}

	case "delete":
		if ri >= 0 && ri < len(ss.BlockRules) {
			ss.BlockRules = append(ss.BlockRules[:ri], ss.BlockRules[ri+1:]...)
		}

	case "add_cond":
		if ri >= 0 && ri < len(ss.BlockRules) {
			ss.BlockRules[ri].Conditions = append(ss.BlockRules[ri].Conditions, swarm.BlockCondition{
				SensorName: "true", OpStr: "==", Value: 1,
			})
		}

	case "new_rule":
		ss.BlockRules = append(ss.BlockRules, swarm.BlockRule{
			Conditions:   []swarm.BlockCondition{{SensorName: "true", OpStr: "==", Value: 1}},
			ActionName:   "FWD",
			ActionParams: [3]int{},
		})
	}
}

func (g *Game) applyBlockDropdownSelection(idx int) {
	ss := g.sim.SwarmState
	dd := ss.ActiveDropdown
	if dd == nil || idx < 0 || idx >= len(dd.Items) {
		return
	}
	selected := dd.Items[idx]
	ri := dd.RuleIdx
	ci := dd.CondIdx

	if ri < 0 || ri >= len(ss.BlockRules) {
		return
	}

	switch dd.FieldType {
	case "sensor":
		if ci >= 0 && ci < len(ss.BlockRules[ri].Conditions) {
			ss.BlockRules[ri].Conditions[ci].SensorName = selected
		}
	case "op":
		if ci >= 0 && ci < len(ss.BlockRules[ri].Conditions) {
			ss.BlockRules[ri].Conditions[ci].OpStr = selected
		}
	case "action":
		ss.BlockRules[ri].ActionName = selected
		ss.BlockRules[ri].ActionParams = [3]int{}
	}
}

func blockEditorDropdownX(fieldType string) int {
	switch fieldType {
	case "sensor":
		return 4 + 20 // blockPadX + blockIfW
	case "op":
		return 4 + 20 + 72 + 2 // after sensor
	case "action":
		return 4 + 20 + 72 + 22 + 30 + 8 + 30 // after THEN
	}
	return 30
}

// handleAlgoLaborClick processes click hits from the Algo-Labor panel.
func (g *Game) handleAlgoLaborClick(hit string) {
	ss := g.sim.SwarmState

	switch {
	case hit == "alglab:f2back":
		// Switch back to Swarm Lab
		ss.AlgoLaborMode = false
		// Turn off all algo overlays
		entries := render.GetAlgoEntries(ss)
		for _, e := range entries {
			*e.ShowPtr = false
		}
		ss.ShowPSO = false
		logger.Info("ALGO-LABOR", "Zurueck zu Swarm Lab")

	case len(hit) > 11 && hit[:11] == "alglab:fit:":
		// Fitness function selection
		idx := 0
		fmt.Sscanf(hit[11:], "%d", &idx)
		fitFuncs := []swarm.FitnessLandscapeType{swarm.FitGaussian, swarm.FitRastrigin, swarm.FitAckley, swarm.FitRosenbrock}
		if idx >= 0 && idx < len(fitFuncs) && ss.SwarmAlgo != nil {
			ss.SwarmAlgo.FitnessFunc = fitFuncs[idx]
			// Force fresh Gaussian peaks when switching back to Gauss
			ss.SwarmAlgo.FitPeakX = nil
			ss.SwarmAlgo.FitPeakY = nil
			ss.SwarmAlgo.FitPeakH = nil
			ss.SwarmAlgo.FitPeakS = nil
			// Re-generate peaks for Gaussian mode
			swarm.RegenerateFitnessPeaks(ss)
			// Invalidate renderer cache
			g.renderer.InvalidateFitnessCache()
			logger.Info("ALGO-LABOR", "Fitness-Funktion: %s", swarm.FitnessLandscapeName(fitFuncs[idx]))
		}

	case len(hit) > 12 && hit[:12] == "alglab:algo:":
		// Algorithm toggle (ON <-> OFF)
		idx := 0
		fmt.Sscanf(hit[12:], "%d", &idx)
		entries := render.GetAlgoEntries(ss)
		if idx >= 0 && idx < len(entries) {
			e := entries[idx]
			if !*e.ShowPtr {
				// Turning ON: init if needed, then show
				if !*e.OnPtr {
					e.Init(ss)
				}
				*e.ShowPtr = true
			} else {
				// Turning OFF
				*e.ShowPtr = false
			}
			logger.Info("ALGO-LABOR", "Algo %s: %v", e.Name, *e.ShowPtr)
		}

	case hit == "alglab:radar":
		ss.ShowAlgoRadar = !ss.ShowAlgoRadar

	case hit == "alglab:tourney":
		if !ss.AlgoTournamentOn {
			swarm.StartAlgoTournament(ss)
			logger.Info("ALGO-LABOR", "Auto-Turnier gestartet")
		}

	case hit == "alglab:speed:1":
		g.sim.Speed = 1.0
		ss.CurrentSpeed = 1.0
	case hit == "alglab:speed:2":
		g.sim.Speed = 2.0
		ss.CurrentSpeed = 2.0
	case hit == "alglab:speed:5":
		g.sim.Speed = 5.0
		ss.CurrentSpeed = 5.0
	case hit == "alglab:speed:10":
		g.sim.Speed = 10.0
		ss.CurrentSpeed = 10.0
	case hit == "alglab:speed:50":
		g.sim.Speed = 50.0
		ss.CurrentSpeed = 50.0
	}
}
