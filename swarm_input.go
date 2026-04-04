package main

import (
	"fmt"
	"math"
	"strings"
	"swarmsim/domain/physics"
	"swarmsim/domain/swarm"
	"swarmsim/engine/swarmscript"
	"swarmsim/locale"
	"swarmsim/logger"
	"swarmsim/render"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// toggleOverlay implements mutual exclusion for educational overlays.
// Only one overlay can be active at a time. Passing the current overlay name closes it.
// Passing "" closes whatever is active.
func toggleOverlay(ss *swarm.SwarmState, name string) {
	if ss.ActiveOverlay == name || name == "" {
		ss.ActiveOverlay = ""
	} else {
		ss.ActiveOverlay = name
	}
	// Sync the individual bool flags for rendering
	ss.ShowMathTrace = ss.ActiveOverlay == "math"
	ss.ShowDecisionTrace = ss.ActiveOverlay == "decision"
	ss.ShowConceptOverlay = ss.ActiveOverlay == "concept"
	ss.ShowGlossary = ss.ActiveOverlay == "glossary"
	if ss.Learning != nil {
		ss.Learning.ShowMenu = ss.ActiveOverlay == "lessons"
	}
	ss.ShowIssueBoard = ss.ActiveOverlay == "issues"
	ss.ShowPerfMonitor = ss.ActiveOverlay == "perf"
	ss.ShowParamTweaker = ss.ActiveOverlay == "tweaker"
	ss.ShowShortcutCard = ss.ActiveOverlay == "shortcuts"
}

// --- Swarm Mode Input Handling ---

func (g *Game) handleSwarmInput() {
	ss := g.sim.SwarmState
	if ss == nil {
		return
	}
	ed := ss.Editor
	mx, my := ebiten.CursorPosition()

	// Mouse wheel: scroll editor if mouse is in editor area
	// Hold Shift for horizontal scrolling
	_, wy := ebiten.Wheel()
	if mx < 350 && wy != 0 && ss.BlockEditorActive {
		// Block editor scroll
		if wy < 0 {
			ss.BlockScrollY += 24
		} else {
			ss.BlockScrollY -= 24
		}
		if ss.BlockScrollY < 0 {
			ss.BlockScrollY = 0
		}
		maxScroll := len(ss.BlockRules)*26 - 200
		if maxScroll < 0 {
			maxScroll = 0
		}
		if ss.BlockScrollY > maxScroll {
			ss.BlockScrollY = maxScroll
		}
	} else if mx < 350 && wy != 0 {
		shiftHeld := ebiten.IsKeyPressed(ebiten.KeyShift)
		if shiftHeld {
			// Horizontal scroll
			if wy < 0 {
				ed.ScrollX += 5
			} else {
				ed.ScrollX -= 5
			}
			if ed.ScrollX < 0 {
				ed.ScrollX = 0
			}
			// Clamp to max line length
			maxCol := 0
			for _, line := range ed.Lines {
				if len(line) > maxCol {
					maxCol = len(line)
				}
			}
			if ed.ScrollX > maxCol {
				ed.ScrollX = maxCol
			}
		} else {
			// Vertical scroll
			if wy < 0 {
				ed.ScrollY += 3
			} else {
				ed.ScrollY -= 3
			}
			if ed.ScrollY < 0 {
				ed.ScrollY = 0
			}
			maxScroll := len(ed.Lines) - ed.MaxVisible/2
			if maxScroll < 0 {
				maxScroll = 0
			}
			if ed.ScrollY > maxScroll {
				ed.ScrollY = maxScroll
			}
		}
	}

	// Tab content scroll (algorithm list in Algo tab)
	if mx < 350 && my >= 660 && wy != 0 && ss.EditorTab == 3 {
		if wy < 0 {
			ss.TabScrollY++
		} else {
			ss.TabScrollY--
		}
		if ss.TabScrollY < 0 {
			ss.TabScrollY = 0
		}
	}

	// Algo-Labor scroll (algorithm list: 20 algos, 10 visible)
	if ss.AlgoLaborMode && mx < 350 && wy != 0 {
		if wy < 0 {
			ss.AlgoLaborScrollY++
		} else {
			ss.AlgoLaborScrollY--
		}
		if ss.AlgoLaborScrollY < 0 {
			ss.AlgoLaborScrollY = 0
		}
		if ss.AlgoLaborScrollY > 10 { // 20 total - 10 visible = 10 max scroll
			ss.AlgoLaborScrollY = 10
		}
	}

	// Glossary scroll (mouse wheel anywhere when glossary is open)
	if ss.ShowGlossary && wy != 0 {
		if wy < 0 {
			ss.GlossaryScroll += 24
		} else {
			ss.GlossaryScroll -= 24
		}
		if ss.GlossaryScroll < 0 {
			ss.GlossaryScroll = 0
		}
		if ss.GlossaryScroll > 600 {
			ss.GlossaryScroll = 600
		}
	}

	// Update dropdown hover tracking
	if ss.DropdownOpen {
		ss.DropdownHover = render.SwarmDropdownHitTest(mx, my, len(ss.Presets))
	}

	// Bot hover detection for tooltip
	ss.HoveredBot = -1
	awx, awy, inside := render.SwarmScreenToArena(mx, my, ss)
	if inside && !ss.DropdownOpen && !ss.ArenaEditMode {
		bestDist := 12.0
		for i := range ss.Bots {
			dx := ss.Bots[i].X - awx
			dy := ss.Bots[i].Y - awy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < bestDist {
				bestDist = dist
				ss.HoveredBot = i
			}
		}
	}

	// Left click handling
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if ss.DropdownOpen {
			// Dropdown is open -- check if we clicked on an item
			idx := render.SwarmDropdownHitTest(mx, my, len(ss.Presets))
			if idx >= 0 {
				g.loadSwarmPreset(idx)
			}
			ss.DropdownOpen = false
		} else {
			g.handleSwarmClick(mx, my)
		}
	}

	// All feature toggles are now handled via the tabbed panel (mouse clicks).
	// Keyboard shortcuts have been removed in favor of the clickable UI.

	// L key kept: toggle light source at mouse position (needs arena coordinates)
	if inpututil.IsKeyJustPressed(ebiten.KeyL) && !ed.Focused && !ss.BotCountEdit {
		awx, awy, inside := render.SwarmScreenToArena(mx, my, ss)
		if inside {
			if ss.Light.Active {
				ss.Light.Active = false
				logger.Info("SWARM", "Light OFF")
			} else {
				ss.Light.Active = true
				ss.Light.X = awx
				ss.Light.Y = awy
				logger.Info("SWARM", "Light ON at (%.0f, %.0f)", awx, awy)
			}
		}
	}

	// K key: toggle live math trace overlay (only when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyK) && !ed.Focused && !ss.BotCountEdit {
		toggleOverlay(ss, "math")
		logger.Info("SWARM", "MathTrace: %v", ss.ShowMathTrace)
	}

	// D key: toggle decision trace overlay (educational — shows which rule fired and why)
	if inpututil.IsKeyJustPressed(ebiten.KeyD) && !ed.Focused && !ss.BotCountEdit {
		toggleOverlay(ss, "decision")
		logger.Info("SWARM", "DecisionTrace: %v", ss.ShowDecisionTrace)
	}

	// C key: toggle concept overlay (educational — sensor radius, neighbors, heading)
	if inpututil.IsKeyJustPressed(ebiten.KeyC) && !ed.Focused && !ss.BotCountEdit {
		toggleOverlay(ss, "concept")
		logger.Info("SWARM", "ConceptOverlay: %v", ss.ShowConceptOverlay)
	}

	// G key: toggle glossary overlay (educational — swarm robotics term reference)
	if inpututil.IsKeyJustPressed(ebiten.KeyG) && !ed.Focused && !ss.BotCountEdit {
		toggleOverlay(ss, "glossary")
		logger.Info("SWARM", "Glossary: %v", ss.ShowGlossary)
	}

	// Shift+L: toggle lesson menu (learning system)
	if inpututil.IsKeyJustPressed(ebiten.KeyL) && ebiten.IsKeyPressed(ebiten.KeyShift) && !ed.Focused && !ss.BotCountEdit {
		if ss.Learning == nil {
			ss.Learning = &swarm.LearningState{}
		}
		toggleOverlay(ss, "lessons")
		if ss.Learning.ShowMenu {
			ss.Learning.Active = false // pause active lesson when opening menu
		}
		logger.Info("LEARN", "Lesson menu: %v", ss.Learning.ShowMenu)
	}

	// I key: toggle issue board overlay (Self-Programming Swarm)
	if inpututil.IsKeyJustPressed(ebiten.KeyI) && !ebiten.IsKeyPressed(ebiten.KeyShift) && !ed.Focused && !ss.BotCountEdit {
		toggleOverlay(ss, "issues")
		logger.Info("SWARM", "IssueBoard: %v", ss.ShowIssueBoard)
	}

	// Shift+I: toggle Collective AI master switch
	if inpututil.IsKeyJustPressed(ebiten.KeyI) && ebiten.IsKeyPressed(ebiten.KeyShift) && !ed.Focused && !ss.BotCountEdit {
		ss.CollectiveAIOn = !ss.CollectiveAIOn
		logger.Info("SWARM", "CollectiveAI: %v", ss.CollectiveAIOn)
	}

	// Shift+P: toggle parameter tweaker overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyP) && ebiten.IsKeyPressed(ebiten.KeyShift) && !ed.Focused && !ss.BotCountEdit {
		toggleOverlay(ss, "tweaker")
		logger.Info("SWARM", "ParamTweaker: %v", ss.ShowParamTweaker)
	}

	// F12: toggle performance monitor overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) && !ed.Focused && !ss.BotCountEdit {
		toggleOverlay(ss, "perf")
		logger.Info("SWARM", "PerfMonitor: %v", ss.ShowPerfMonitor)
	}

	// ? (Shift+Slash): toggle shortcut reference card
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && ebiten.IsKeyPressed(ebiten.KeyShift) && !ed.Focused && !ss.BotCountEdit {
		toggleOverlay(ss, "shortcuts")
		logger.Info("SWARM", "ShortcutCard: %v", ss.ShowShortcutCard)
	}

	// (Keyboard shortcuts for T, F4, F5, F6, F8, F9, etc. removed -- use tabbed panel instead)

	// Ctrl+S: save session (program text to disk)
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && (ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)) {
		source := strings.Join(ed.Lines, "\n")
		swarm.SaveSession(source)
		ss.SessionSaveFlash = 60
		logger.Info("SWARM", "Session saved to disk")
	}

	// 0 key: zoom-to-fit all bots (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.Key0) && !ed.Focused && !ss.BotCountEdit {
		g.swarmZoomToFit(ss)
	}

	// Editor keyboard input (when editor is focused)
	if ed.Focused {
		g.handleSwarmEditorKeys()
	}

	// Bot count field input (when focused)
	if ss.BotCountEdit {
		g.handleBotCountInput()
	}

	// Block editor value field input
	if ss.BlockValueEdit {
		g.handleBlockValueInput()
	}

	// Tooltip hover detection
	hoverID := ""
	if !ed.Focused && !ss.BotCountEdit {
		hoverID = render.SwarmEditorHitTest(mx, my, ss)
		// Also check dropdown hover for preset tooltips
		if ss.DropdownOpen {
			idx := render.SwarmDropdownHitTest(mx, my, len(ss.Presets))
			if idx >= 0 && idx < len(ss.Presets) {
				hoverID = "preset:" + ss.Presets[idx].Name
			}
		}
	}
	render.UpdateTooltip(&g.tooltip, mx, my, hoverID)
}

func (g *Game) handleSwarmClick(mx, my int) {
	ss := g.sim.SwarmState
	ed := ss.Editor
	sw, sh := ebiten.WindowSize()

	// Parameter tweaker click handling
	if ss.ShowParamTweaker {
		hit := render.ParamTweakerHitTest(mx, my, ss)
		if len(hit) > 0 && strings.HasPrefix(hit, "param:") {
			// Parse "param:A:+" or "param:B:-"
			paramLetter := hit[6]
			paramIdx := int(paramLetter - 'A')
			direction := hit[8]
			delta := float64(1)
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				delta = 10
			}
			if direction == '-' {
				delta = -delta
			}
			render.ApplyParamTweak(ss, paramIdx, delta)
			logger.Info("SWARM", "Tweaked $%c by %.0f -> %.0f", paramLetter, delta, ss.Bots[0].ParamValues[paramIdx])
			return
		}
	}

	// Learning system: lesson menu click
	if ss.Learning != nil && ss.Learning.ShowMenu {
		idx := render.LessonMenuHitTest(mx, my, sw, sh)
		if idx >= 0 && idx < int(swarm.LessonCount) {
			// Check if locked
			locked := idx > 0 && ss.Learning.Completed[idx-1] == 0
			if !locked {
				swarm.StartLesson(ss.Learning, swarm.LessonID(idx))
				// Load preset for this lesson
				lessons := swarm.GetAllLessons()
				if idx < len(lessons) {
					for pi, p := range ss.Presets {
						if p.Name == lessons[idx].PresetName {
							g.loadSwarmPreset(pi)
							break
						}
					}
				}
				logger.Info("LEARN", "Started lesson %d: %s", idx+1, lessons[idx].TitleKey)
			}
		} else {
			// Click outside cards closes menu
			ss.Learning.ShowMenu = false
		}
		return
	}

	// Learning system: lesson overlay click (next/exit)
	if ss.Learning != nil && ss.Learning.Active {
		action := render.LessonNextHitTest(mx, my, sw, sh, ss.Learning)
		switch action {
		case "next":
			swarm.AdvanceStep(ss.Learning)
			return
		case "exit":
			ss.Learning.Active = false
			ss.Learning.ChallengeActive = false
			return
		}
	}

	hit := render.SwarmEditorHitTest(mx, my, ss)
	switch hit {
	case "dropdown":
		ss.DropdownOpen = !ss.DropdownOpen
		ed.Focused = false
		ss.BotCountEdit = false

	case "deploy":
		g.deploySwarmProgram()
		ss.DropdownOpen = false
		ss.BotCountEdit = false
		g.renderer.Sound.PlayDeploy()

	case "reset":
		if ss.PendingConfirm == "reset" {
			// Confirmed! Execute reset
			ss.PendingConfirm = ""
			ss.ResetBots()
			// Re-create truck state so a fresh truck arrives immediately
			if ss.TruckToggle {
				ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
			}
			logger.Info("SWARM", "RESET -- %d bots scattered", ss.BotCount)
			g.renderer.Sound.PlayReset()
			ss.DropdownOpen = false
			ss.BotCountEdit = false
		} else {
			ss.PendingConfirm = "reset"
			ss.PendingConfirmTick = 180
		}

	case "copy":
		// Export program to clipboard
		text := strings.Join(ed.Lines, "\n")
		render.ClipboardWrite(text)
		ss.ClipboardFlash = 30
		logger.Info("SWARM", "Program copied to clipboard (%d lines)", len(ed.Lines))

	case "paste":
		// Import program from clipboard
		render.ClipboardRead(func(text string) {
			if text == "" {
				return
			}
			lines := strings.Split(text, "\n")
			ed.Lines = lines
			ed.CursorLine = 0
			ed.CursorCol = 0
			ed.ScrollY = 0
			ss.ProgramName = "Custom"
			logger.Info("SWARM", "Program pasted from clipboard (%d lines)", len(lines))
		})

	case "botcount":
		ss.BotCountEdit = true
		ed.Focused = false
		ss.DropdownOpen = false

	case "bots_minus":
		newCount := ss.BotCount - 10
		if newCount < swarm.SwarmMinBots {
			newCount = swarm.SwarmMinBots
		}
		ss.RespawnBots(newCount)
		logger.Info("SWARM", "Bot count decreased to %d", newCount)
		ed.Focused = false
		ss.BotCountEdit = false

	case "bots_plus":
		newCount := ss.BotCount + 10
		if newCount > swarm.SwarmMaxBots {
			newCount = swarm.SwarmMaxBots
		}
		ss.RespawnBots(newCount)
		logger.Info("SWARM", "Bot count increased to %d", newCount)
		ed.Focused = false
		ss.BotCountEdit = false

	case "text_mode":
		if ss.BlockEditorActive {
			// Serialize blocks back to text
			ss.BlockEditorActive = false
			ss.ActiveDropdown = nil
			text := serializeBlockRules(ss.BlockRules)
			ss.Editor.Lines = strings.Split(text, "\n")
			ss.Editor.CursorLine = 0
			ss.Editor.CursorCol = 0
			ss.Editor.ScrollY = 0
			logger.Info("SWARM", "Switched to TEXT mode")
		}

	case "block_mode":
		if !ss.BlockEditorActive {
			// Parse current text into blocks
			source := strings.Join(ss.Editor.Lines, "\n")
			prog, err := swarmscript.ParseSwarmScript(source)
			if err != nil {
				ss.ErrorMsg = "Block-Modus: " + err.Error()
				logger.Warn("SWARM", "Cannot switch to block mode: %s", err.Error())
			} else {
				ss.BlockRules = rulesToBlockRules(prog.Rules)
				ss.BlockEditorActive = true
				ss.ActiveDropdown = nil
				ss.BlockScrollY = 0
				ss.BlockValueEdit = false
				ss.ErrorMsg = ""
				ed.Focused = false
				logger.Info("SWARM", "Switched to BLOCKS mode (%d rules)", len(ss.BlockRules))
			}
		}

	case "editor":
		if ss.BlockEditorActive {
			g.handleBlockEditorClick(mx, my)
		} else {
			ed.Focused = true
			ss.BotCountEdit = false
			ss.DropdownOpen = false
			line, col := render.SwarmEditorClickToPos(mx, my, ed)
			ed.CursorLine = line
			ed.CursorCol = col
			ed.BlinkTick = 0
		}

	case "obstacles":
		ss.ObstaclesOn = !ss.ObstaclesOn
		if ss.ObstaclesOn {
			ss.MazeOn = false
			ss.MazeWalls = nil
			swarm.GenerateSwarmObstacles(ss)
			logger.Info("SWARM", "Obstacles ON (%d obstacles)", len(ss.Obstacles))
		} else {
			ss.Obstacles = nil
			logger.Info("SWARM", "Obstacles OFF")
		}
		g.autoReset("Obstacles toggled")
		ed.Focused = false
		ss.BotCountEdit = false

	case "maze":
		ss.MazeOn = !ss.MazeOn
		if ss.MazeOn {
			ss.ObstaclesOn = false
			ss.Obstacles = nil
			swarm.GenerateSwarmMaze(ss)
			logger.Info("SWARM", "Maze ON (%d walls)", len(ss.MazeWalls))
		} else {
			ss.MazeWalls = nil
			logger.Info("SWARM", "Maze OFF")
		}
		g.autoReset("Maze toggled")
		ed.Focused = false
		ss.BotCountEdit = false

	case "light":
		if ss.Light.Active {
			ss.Light.Active = false
			logger.Info("SWARM", "Light OFF (button)")
		} else {
			ss.Light.Active = true
			ss.Light.X = ss.ArenaW / 2
			ss.Light.Y = ss.ArenaH / 2
			logger.Info("SWARM", "Light ON at arena center")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "walls":
		ss.WrapMode = !ss.WrapMode
		mode := "BOUNCE"
		if ss.WrapMode {
			mode = "WRAP"
		}
		// Scatter bot positions only (no full reset)
		for i := range ss.Bots {
			margin := 30.0
			ss.Bots[i].X = margin + ss.Rng.Float64()*(ss.ArenaW-2*margin)
			ss.Bots[i].Y = margin + ss.Rng.Float64()*(ss.ArenaH-2*margin)
		}
		ss.ResetFlashTimer = 30
		logger.Info("SWARM", "Walls mode: %s (positions reset)", mode)
		ed.Focused = false
		ss.BotCountEdit = false

	case "delivery":
		// Non-delivery preset: button is grayed, ignore clicks
		if ss.ProgramName != "Custom" && !ss.IsDeliveryProgram {
			break
		}
		// Delivery preset: cannot disable delivery
		if ss.IsDeliveryProgram && ss.DeliveryOn {
			logger.Info("SWARM", "Cannot disable delivery while a delivery program is active")
			break
		}
		// Neuro needs delivery for fitness
		if ss.NeuroEnabled && ss.DeliveryOn {
			logger.Info("SWARM", "Cannot disable delivery while Neuro is active")
			break
		}
		ss.DeliveryOn = !ss.DeliveryOn
		if ss.DeliveryOn {
			// Generate stations and packages
			swarm.GenerateDeliveryStations(ss)
			// Reset bot carrying state
			for i := range ss.Bots {
				ss.Bots[i].CarryingPkg = -1
			}
			logger.Info("SWARM", "Delivery ON (%d stations, %d packages)", len(ss.Stations), len(ss.Packages))
		} else {
			ss.Stations = nil
			ss.Packages = nil
			ss.DeliveryStats = swarm.DeliveryStats{}
			// Reset bot carrying state
			for i := range ss.Bots {
				ss.Bots[i].CarryingPkg = -1
			}
			logger.Info("SWARM", "Delivery OFF")
		}
		g.autoReset("Delivery toggled")
		ed.Focused = false
		ss.BotCountEdit = false

	case "trucks":
		// Non-truck preset: button is grayed, ignore clicks
		if ss.ProgramName != "Custom" && !ss.IsTruckProgram {
			break
		}
		// Truck preset: cannot disable trucks
		if ss.IsTruckProgram && ss.TruckToggle {
			logger.Info("SWARM", "Cannot disable trucks while a truck program is active")
			break
		}
		ss.TruckToggle = !ss.TruckToggle
		if ss.TruckToggle {
			// Force delivery on (need dropoff stations)
			if !ss.DeliveryOn {
				ss.DeliveryOn = true
			}
			// Maze OFF when trucks active (bots + trucks use open arena)
			ss.MazeOn = false
			ss.MazeWalls = nil
			ss.ObstaclesOn = false
			ss.Obstacles = nil
			// Generate stations and truck state
			swarm.GenerateDeliveryStations(ss)
			ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
			logger.Info("SWARM", "Trucks ON (round %d)", ss.TruckState.RoundNum)
		} else {
			ss.TruckState = nil
			// Regenerate maze without ramp exemption
			if ss.MazeOn {
				swarm.GenerateSwarmMaze(ss)
			}
			logger.Info("SWARM", "Trucks OFF")
		}
		g.autoReset("Trucks toggled")
		ed.Focused = false
		ss.BotCountEdit = false

	case "evolution":
		// Block if GP or Neuro is active (user must turn those off first)
		if !ss.EvolutionOn && (ss.GPEnabled || ss.NeuroEnabled) {
			logger.Info("SWARM", "Evolution gesperrt -- erst GP/Neuro ausschalten")
			break
		}
		ss.EvolutionOn = !ss.EvolutionOn
		if ss.EvolutionOn {
			ss.GPEnabled = false // mutually exclusive
			ss.NeuroEnabled = false
			swarm.ClearGP(ss)
			swarm.ClearNeuro(ss)
			swarm.InitBotParams(ss)
			ss.Generation = 0
			ss.EvolutionTimer = 0
			ss.FitnessHistory = nil
			logger.Info("SWARM", "Evolution ON -- %d used params", countUsedParams(ss))
		} else {
			ss.ShowGenomeViz = false
			ss.GenomeBrowserOn = false
			logger.Info("SWARM", "Evolution OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "teams":
		ss.TeamsEnabled = !ss.TeamsEnabled
		if ss.TeamsEnabled {
			swarm.InitTeams(ss)
			// Set team programs from current shared program
			if ss.Program != nil {
				ss.TeamAProgram = ss.Program
				ss.TeamBProgram = ss.Program
			}
			logger.Info("SWARM", "Teams ON -- %d bots split into two teams", ss.BotCount)
		} else {
			swarm.ClearTeams(ss)
			logger.Info("SWARM", "Teams OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "gp":
		if !ss.GPEnabled && (ss.EvolutionOn || ss.NeuroEnabled) {
			logger.Info("SWARM", "GP gesperrt -- erst Evolution/Neuro ausschalten")
			break
		}
		ss.GPEnabled = !ss.GPEnabled
		if ss.GPEnabled {
			// Turn off regular evolution and neuro (mutually exclusive)
			ss.EvolutionOn = false
			ss.ShowGenomeViz = false
			ss.NeuroEnabled = false
			swarm.ClearNeuro(ss)
			swarm.InitGP(ss)
			logger.Info("SWARM", "GP ON -- %d bots, each with own random program", ss.BotCount)
		} else {
			swarm.ClearGP(ss)
			logger.Info("SWARM", "GP OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "neuro":
		if !ss.NeuroEnabled && (ss.EvolutionOn || ss.GPEnabled) {
			logger.Info("SWARM", "Neuro gesperrt -- erst Evolution/GP ausschalten")
			break
		}
		ss.NeuroEnabled = !ss.NeuroEnabled
		if ss.NeuroEnabled {
			// Turn off regular evolution and GP (mutually exclusive)
			ss.EvolutionOn = false
			ss.ShowGenomeViz = false
			ss.GPEnabled = false
			swarm.ClearGP(ss)
			// Auto-enable delivery if not already on
			if !ss.DeliveryOn {
				ss.DeliveryOn = true
				ss.IsDeliveryProgram = true
				swarm.GenerateDeliveryStations(ss)
			}
			swarm.InitNeuro(ss)
			logger.Info("SWARM", "NEURO ON -- %d Bots x %d Gewichte, Delivery automatisch aktiviert",
				ss.BotCount, swarm.NeuroWeights)
		} else {
			swarm.ClearNeuro(ss)
			logger.Info("SWARM", "NEURO OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	// ==================== TAB SYSTEM ====================
	case "tab:0", "tab:1", "tab:2", "tab:3":
		tabIdx := int(hit[4] - '0')
		if tabIdx > 3 {
			tabIdx = 3
		}
		ss.EditorTab = tabIdx
		ss.TabScrollY = 0
		ed.Focused = false
		ss.BotCountEdit = false

	// --- Tab 1 (Evo) new toggles ---
	case "pareto":
		if !ss.EvolutionOn && !ss.GPEnabled {
			logger.Info("SWARM", "Pareto braucht Evolution oder GP")
			break
		}
		ss.ParetoEnabled = !ss.ParetoEnabled
		logger.Info("SWARM", "Pareto: %v", ss.ParetoEnabled)
	case "speciation":
		if !ss.EvolutionOn {
			logger.Info("SWARM", "Artbildung braucht Evolution")
			break
		}
		ss.SpeciationOn = !ss.SpeciationOn
		if ss.SpeciationOn && ss.Speciation == nil {
			swarm.InitSpeciation(ss)
		}
		logger.Info("SWARM", "Speciation: %v", ss.SpeciationOn)
	case "sensornoise":
		ss.SensorNoiseOn = !ss.SensorNoiseOn
		if ss.SensorNoiseOn && ss.SensorNoiseCfg.NoiseLevel == 0 {
			ss.SensorNoiseCfg = swarm.SensorNoiseConfig{
				NoiseLevel:  0.15,
				FailureRate: 0.02,
			}
		}
		logger.Info("SWARM", "SensorNoise: %v", ss.SensorNoiseOn)
	case "memory":
		ss.MemoryEnabled = !ss.MemoryEnabled
		logger.Info("SWARM", "Memory: %v", ss.MemoryEnabled)
	case "leaderboard":
		ss.ShowLeaderboard = !ss.ShowLeaderboard
		logger.Info("SWARM", "Leaderboard: %v", ss.ShowLeaderboard)
	case "collective_ai":
		ss.CollectiveAIOn = !ss.CollectiveAIOn
		logger.Info("SWARM", "CollectiveAI: %v", ss.CollectiveAIOn)

	// --- Tab 2 (Anzeige) toggles ---
	case "dashboard":
		ss.DashboardOn = !ss.DashboardOn
		logger.Info("SWARM", "Dashboard: %v", ss.DashboardOn)
	case "minimap":
		g.renderer.ShowMinimap = !g.renderer.ShowMinimap
		ss.ShowMinimap = g.renderer.ShowMinimap
		logger.Info("SWARM", "Minimap: %v", g.renderer.ShowMinimap)
	case "trails":
		ss.ShowTrails = !ss.ShowTrails
		logger.Info("SWARM", "Trails: %v", ss.ShowTrails)
	case "heatmap":
		ss.ShowHeatmap = !ss.ShowHeatmap
		if ss.ShowHeatmap {
			swarm.InitHeatmap(ss)
		}
		logger.Info("SWARM", "Heatmap: %v", ss.ShowHeatmap)
	case "routes":
		ss.ShowRoutes = !ss.ShowRoutes
		logger.Info("SWARM", "Routes: %v", ss.ShowRoutes)
	case "livechart":
		ss.ShowLiveChart = !ss.ShowLiveChart
		logger.Info("SWARM", "LiveChart: %v", ss.ShowLiveChart)
	case "commgraph":
		ss.ShowCommGraph = !ss.ShowCommGraph
		logger.Info("SWARM", "CommGraph: %v", ss.ShowCommGraph)
	case "msgwaves":
		ss.ShowMsgWaves = !ss.ShowMsgWaves
		logger.Info("SWARM", "MsgWaves: %v", ss.ShowMsgWaves)
	case "genomeviz":
		if ss.NeuroEnabled || ss.GPEnabled {
			break // only for parametric evolution
		}
		ss.ShowGenomeViz = !ss.ShowGenomeViz
		logger.Info("SWARM", "GenomeViz: %v", ss.ShowGenomeViz)
	case "genomebrowser":
		if ss.NeuroEnabled || ss.GPEnabled {
			break // only for parametric evolution
		}
		ss.GenomeBrowserOn = !ss.GenomeBrowserOn
		logger.Info("SWARM", "GenomeBrowser: %v", ss.GenomeBrowserOn)
	case "swarmcenter":
		ss.ShowSwarmCenter = !ss.ShowSwarmCenter
		logger.Info("SWARM", "SwarmCenter: %v", ss.ShowSwarmCenter)
	case "congestion":
		ss.ShowZones = !ss.ShowZones
		logger.Info("SWARM", "CongestionZones: %v", ss.ShowZones)
	case "prediction":
		ss.ShowPrediction = !ss.ShowPrediction
		logger.Info("SWARM", "Prediction: %v", ss.ShowPrediction)
	case "colorfilter":
		ss.ColorFilter = (ss.ColorFilter + 1) % 6
		logger.Info("SWARM", "ColorFilter: %d", ss.ColorFilter)

	// Overlay quick-access buttons (Display tab)
	case "overlay_math":
		toggleOverlay(ss, "math")
		logger.Info("SWARM", "Overlay: math=%v", ss.ShowMathTrace)
	case "overlay_decision":
		toggleOverlay(ss, "decision")
		logger.Info("SWARM", "Overlay: decision=%v", ss.ShowDecisionTrace)
	case "overlay_concept":
		toggleOverlay(ss, "concept")
		logger.Info("SWARM", "Overlay: concept=%v", ss.ShowConceptOverlay)
	case "overlay_glossary":
		toggleOverlay(ss, "glossary")
		logger.Info("SWARM", "Overlay: glossary=%v", ss.ShowGlossary)
	case "overlay_issues":
		toggleOverlay(ss, "issues")
		logger.Info("SWARM", "Overlay: issues=%v", ss.ShowIssueBoard)

	case "language":
		locale.CycleLang()
		locale.SaveLang()
		render.ClearTextCache()
		ebiten.SetWindowTitle(locale.T("ui.window_title"))
		logger.Info("SWARM", "Language: %s", locale.LangDisplayName())

	// --- Tab 3 (Algo) ---
	case "algo:radar":
		ss.ShowAlgoRadar = !ss.ShowAlgoRadar
		logger.Info("SWARM", "AlgoRadar: %v", ss.ShowAlgoRadar)
	case "mathtrace":
		toggleOverlay(ss, "math")
		logger.Info("SWARM", "MathTrace: %v", ss.ShowMathTrace)
	case "algo:tourney":
		if !ss.AlgoTournamentOn {
			swarm.StartAlgoTournament(ss)
			logger.Info("SWARM", "Algo-Tournament gestartet")
		}

	// --- Tab 3 (Werkzeuge) ---
	case "speed:0.5":
		g.sim.Speed = 0.5
		ss.CurrentSpeed = 0.5
	case "speed:1":
		g.sim.Speed = 1.0
		ss.CurrentSpeed = 1.0
	case "speed:2":
		g.sim.Speed = 2.0
		ss.CurrentSpeed = 2.0
	case "speed:5":
		g.sim.Speed = 5.0
		ss.CurrentSpeed = 5.0
	case "speed:10":
		g.sim.Speed = 10.0
		ss.CurrentSpeed = 10.0
	case "newround":
		if ss.PendingConfirm == "newround" {
			// Confirmed! Execute new round
			ss.PendingConfirm = ""
			if ss.TeamsEnabled {
				swarm.ResetTeamScores(ss)
				if ss.DeliveryOn {
					ss.ResetDeliveryState()
					swarm.GenerateDeliveryStations(ss)
				}
			} else if ss.TruckToggle && ss.TruckState != nil {
				oldRound := ss.TruckState.RoundNum
				ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
				ss.TruckState.RoundNum = oldRound + 1
				ss.ResetBots()
				if ss.DeliveryOn {
					swarm.GenerateDeliveryStations(ss)
				}
			}
			logger.Info("SWARM", "Neue Runde")
		} else {
			ss.PendingConfirm = "newround"
			ss.PendingConfirmTick = 180
		}
	case "screenshot":
		g.screenshotRequested = true
	case "gif":
		g.gifToggleRequested = true
	case "replay":
		ss.ReplayMode = !ss.ReplayMode
		if ss.ReplayMode && ss.ReplayBuf != nil {
			ss.ReplayPlayer = swarm.NewReplayPlayer(ss.ReplayBuf)
		}
		logger.Info("SWARM", "Replay: %v", ss.ReplayMode)
	case "tournament":
		ss.TournamentOn = !ss.TournamentOn
		logger.Info("SWARM", "Tournament: %v", ss.TournamentOn)

	// --- Tab 0 (Arena) new toggles ---
	case "energy":
		ss.EnergyEnabled = !ss.EnergyEnabled
		if ss.EnergyEnabled {
			for i := range ss.Bots {
				ss.Bots[i].Energy = 100
			}
		}
		logger.Info("SWARM", "Energy: %v", ss.EnergyEnabled)
	case "dynamicenv":
		ss.DynamicEnv = !ss.DynamicEnv
		logger.Info("SWARM", "DynamicEnv: %v", ss.DynamicEnv)
	case "daynight":
		ss.DayNightOn = !ss.DayNightOn
		if ss.DayNightOn {
			ss.DayNightPhase = 0
			ss.DayNightSpeed = 0.0002
		}
		logger.Info("SWARM", "DayNight: %v", ss.DayNightOn)
	case "arenaeditor":
		ss.ArenaEditMode = !ss.ArenaEditMode
		logger.Info("SWARM", "ArenaEditMode: %v", ss.ArenaEditMode)

	default:
		// Handle Algo-Labor clicks (alglab:* IDs)
		if len(hit) > 7 && hit[:7] == "alglab:" {
			g.handleAlgoLaborClick(hit)
			return
		}
		// NOTE: algo:N clicks from the old Algo tab are dead code since the
		// Algo tab was removed (algorithms now live in F4 Algo-Labor mode,
		// handled via "alglab:algo:N" above). Kept as safety fallback.
		if len(hit) > 5 && hit[:5] == "algo:" {
			idx := 0
			fmt.Sscanf(hit[5:], "%d", &idx)
			entries := render.GetAlgoEntries(ss)
			if idx >= 0 && idx < len(entries) {
				e := entries[idx]
				if !*e.OnPtr {
					e.Init(ss)
				}
				*e.ShowPtr = !*e.ShowPtr
				logger.Info("SWARM", "Algo %s: %v (legacy path)", e.Name, *e.ShowPtr)
			}
			return
		}
		// exportswarm / importswarm / exportcsv are handled via existing keyboard shortcuts
		// (Ctrl+S, Ctrl+O, X) -- not yet wired as click actions

		// Clicked outside editor panel -- check arena for bot selection
		ed.Focused = false
		ss.BotCountEdit = false
		ss.DropdownOpen = false

		// Arena edit mode: place/remove objects
		awx, awy, inside := render.SwarmScreenToArena(mx, my, ss)
		if inside && ss.ArenaEditMode {
			g.handleArenaEdit(awx, awy)
			return
		}

		// Try to select a bot in the arena
		if inside {
			bestIdx := -1
			bestDist := 15.0 // max click distance
			for i := range ss.Bots {
				dx := ss.Bots[i].X - awx
				dy := ss.Bots[i].Y - awy
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < bestDist {
					bestDist = dist
					bestIdx = i
				}
			}

			// Ctrl+click: clone bot genome to all others
			ctrlHeld := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
			if ctrlHeld && bestIdx >= 0 {
				g.cloneBotGenome(bestIdx)
				return
			}

			// Shift+click: set compare bot (if a primary bot is already selected)
			shiftHeld := ebiten.IsKeyPressed(ebiten.KeyShift)
			if shiftHeld && bestIdx >= 0 && ss.SelectedBot >= 0 && bestIdx != ss.SelectedBot {
				ss.CompareBot = bestIdx
				logger.Info("SWARM", "Compare bot #%d vs #%d", ss.SelectedBot, bestIdx)
			} else {
				ss.SelectedBot = bestIdx
				ss.CompareBot = -1 // normal click clears comparison
				// Cancel follow-cam if we selected a different bot
				if ss.FollowCamBot >= 0 && bestIdx != ss.FollowCamBot {
					ss.FollowCamBot = -1
				}
				if bestIdx >= 0 {
					logger.Info("SWARM", "Selected bot #%d", bestIdx)
				}
			}
		} else {
			ss.SelectedBot = -1
			ss.CompareBot = -1
		}
	}
}

// cloneBotGenome copies a bot's genome (params/weights/program) to all other bots.
func (g *Game) cloneBotGenome(srcIdx int) {
	ss := g.sim.SwarmState
	src := &ss.Bots[srcIdx]
	cloned := 0

	for i := range ss.Bots {
		if i == srcIdx {
			continue
		}
		// Clone evolution params
		if ss.EvolutionOn {
			ss.Bots[i].ParamValues = src.ParamValues
			cloned++
		}
		// Clone neuro brain weights
		if ss.NeuroEnabled && src.Brain != nil {
			if ss.Bots[i].Brain == nil {
				ss.Bots[i].Brain = &swarm.NeuroBrain{}
			}
			ss.Bots[i].Brain.Weights = src.Brain.Weights
			cloned++
		}
		// Clone GP program
		if ss.GPEnabled && src.OwnProgram != nil {
			ss.Bots[i].OwnProgram = src.OwnProgram // share program (read-only between evolutions)
			cloned++
		}
	}
	ss.SelectedBot = srcIdx
	logger.Info("SWARM", "Cloned bot #%d genome to %d bots", srcIdx, cloned)
}

// handleArenaEdit handles arena clicks when arena edit mode is active.
func (g *Game) handleArenaEdit(awx, awy float64) {
	ss := g.sim.SwarmState

	switch ss.ArenaEditTool {
	case 0: // Place obstacle
		obs := &physics.Obstacle{
			X: awx - 20,
			Y: awy - 20,
			W: 40,
			H: 40,
		}
		ss.Obstacles = append(ss.Obstacles, obs)
		ss.ObstaclesOn = true
		logger.Info("ARENA-EDIT", "Placed obstacle at (%.0f, %.0f)", awx, awy)

	case 1: // Place station
		if !ss.DeliveryOn {
			ss.DeliveryOn = true
			ss.Stations = nil
			ss.Packages = nil
		}
		// Cycle through colors: count existing stations to pick next color
		nextColor := (len(ss.Stations) % 4) + 1
		isPickup := len(ss.Stations)%2 == 0 // alternate pickup/dropoff
		st := swarm.DeliveryStation{
			X:        awx,
			Y:        awy,
			Color:    nextColor,
			IsPickup: isPickup,
		}
		if isPickup {
			st.HasPackage = true
			ss.Packages = append(ss.Packages, swarm.DeliveryPackage{
				Color:     nextColor,
				CarriedBy: -1,
				X:         awx,
				Y:         awy,
				Active:    true,
				SpawnTick: ss.Tick,
			})
		}
		ss.Stations = append(ss.Stations, st)
		kind := "Dropoff"
		if isPickup {
			kind = "Pickup"
		}
		logger.Info("ARENA-EDIT", "Placed %s station (color %d) at (%.0f, %.0f)", kind, nextColor, awx, awy)

	case 2: // Delete nearest object
		// Check obstacles first
		bestObsDist := 50.0
		bestObsIdx := -1
		for i, obs := range ss.Obstacles {
			cx := obs.X + obs.W/2
			cy := obs.Y + obs.H/2
			dx := cx - awx
			dy := cy - awy
			d := math.Sqrt(dx*dx + dy*dy)
			if d < bestObsDist {
				bestObsDist = d
				bestObsIdx = i
			}
		}
		// Check stations
		bestStDist := 50.0
		bestStIdx := -1
		for i := range ss.Stations {
			dx := ss.Stations[i].X - awx
			dy := ss.Stations[i].Y - awy
			d := math.Sqrt(dx*dx + dy*dy)
			if d < bestStDist {
				bestStDist = d
				bestStIdx = i
			}
		}
		// Delete whichever is closer
		if bestObsIdx >= 0 && (bestStIdx < 0 || bestObsDist < bestStDist) {
			ss.Obstacles = append(ss.Obstacles[:bestObsIdx], ss.Obstacles[bestObsIdx+1:]...)
			logger.Info("ARENA-EDIT", "Deleted obstacle at (%.0f, %.0f)", awx, awy)
			if len(ss.Obstacles) == 0 {
				ss.ObstaclesOn = false
			}
		} else if bestStIdx >= 0 {
			ss.Stations = append(ss.Stations[:bestStIdx], ss.Stations[bestStIdx+1:]...)
			logger.Info("ARENA-EDIT", "Deleted station at (%.0f, %.0f)", awx, awy)
		}
	}
}

// swarmZoomToFit calculates the bounding box of all bots and sets the camera
// to center on the swarm with enough zoom to fit all bots in the viewport.
func (g *Game) swarmZoomToFit(ss *swarm.SwarmState) {
	if len(ss.Bots) == 0 {
		return
	}
	minX, maxX := ss.Bots[0].X, ss.Bots[0].X
	minY, maxY := ss.Bots[0].Y, ss.Bots[0].Y
	for i := range ss.Bots {
		if ss.Bots[i].X < minX {
			minX = ss.Bots[i].X
		}
		if ss.Bots[i].X > maxX {
			maxX = ss.Bots[i].X
		}
		if ss.Bots[i].Y < minY {
			minY = ss.Bots[i].Y
		}
		if ss.Bots[i].Y > maxY {
			maxY = ss.Bots[i].Y
		}
	}
	// Center camera on bounding box center
	ss.SwarmCamX = (minX + maxX) / 2
	ss.SwarmCamY = (minY + maxY) / 2
	// Zoom to fit with 10% margin; viewport is ~800x800
	bboxW := maxX - minX + 100 // padding so bots are not on the edge
	bboxH := maxY - minY + 100
	if bboxW < 50 {
		bboxW = 50
	}
	if bboxH < 50 {
		bboxH = 50
	}
	zoomW := 800.0 / bboxW
	zoomH := 800.0 / bboxH
	ss.SwarmCamZoom = math.Min(zoomW, zoomH) * 0.9
	// Disable follow-cam so zoom-to-fit isn't overridden
	ss.FollowCamBot = -1
	logger.Info("SWARM", "Zoom-to-fit: center=(%.0f,%.0f) zoom=%.2f", ss.SwarmCamX, ss.SwarmCamY, ss.SwarmCamZoom)
}
