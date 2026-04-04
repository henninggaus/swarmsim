package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"swarmsim/engine/swarmscript"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Editor layout constants — reorganized panel
const (
	editorPanelW = 350

	// Title row
	editorTitleY = 2

	// Button bar
	editorBarY = 24
	editorBarH = 25

	// Code editor area
	editorCodeY = 54
	editorCodeH = 544 // 34 visible lines * 16px

	// Error + status
	editorErrorY  = 600
	editorStatusY = 618

	// Separator 1
	editorSep1Y = 618

	// Bot count input (above the tab bar)
	editorBotCountY = 622

	// Toggle buttons row 1
	editorToggle1Y = 662

	// Toggle buttons row 2
	editorToggle2Y = 688

	// Toggle buttons row 3
	editorToggle3Y = 714

	// Toggle buttons row 4
	editorToggle4Y = 740

	// Toggle buttons row 5
	editorToggle5Y = 766

	// Separator 2
	editorSep2Y = 792

	// Stats panel
	editorStatsY = 798

	editorTextX    = 40 // code text start x (after line numbers)
	editorLineNumX = 5  // line number x

	// Toggle button dimensions
	toggleBtnW = 165
	toggleBtnH = 22
)

// DrawSwarmEditor renders the editor panel on the left side of the screen.
func DrawSwarmEditor(screen *ebiten.Image, ss *swarm.SwarmState) {
	if ss.AlgoLaborMode {
		DrawAlgoLabor(screen, ss)
		return
	}
	if ss == nil || ss.Editor == nil {
		return
	}
	tickTextCache() // maintain text image cache
	ed := ss.Editor

	// Title bar with TEXT/BLOCKS toggle
	printColoredAt(screen, "SwarmScript Editor", 10, editorTitleY, ColorWhite)
	// TEXT/BLOCKS mode toggle
	textBtnCol := ColorSwarmBtnDeploy
	blockBtnCol := color.RGBA{50, 50, 65, 255}
	if ss.BlockEditorActive {
		textBtnCol = color.RGBA{50, 50, 65, 255}
		blockBtnCol = ColorSwarmBtnDeploy
	}
	drawSwarmButton(screen, 210, editorTitleY, 45, 16, "TEXT", textBtnCol)
	drawSwarmButton(screen, 258, editorTitleY, 60, 16, "BLOCKS", blockBtnCol)

	// Button bar: [▼ PresetName] [DEPLOY] [RESET]
	drawSwarmDropdownButton(screen, ss)
	drawSwarmButton(screen, 195, editorBarY, 60, editorBarH, "DEPLOY", ColorSwarmBtnDeploy)
	drawSwarmButton(screen, 258, editorBarY, 40, editorBarH, "RESET", ColorSwarmBtnReset)
	// Export/Import: [COPY] [PASTE]
	copyCol := color.RGBA{60, 100, 140, 255}
	pasteCol := color.RGBA{100, 60, 140, 255}
	if ss.ClipboardFlash > 0 {
		copyCol = color.RGBA{80, 200, 100, 255}
	}
	drawSwarmButton(screen, 300, editorBarY, 23, editorBarH, "CP", copyCol)
	drawSwarmButton(screen, 325, editorBarY, 23, editorBarH, "PA", pasteCol)

	// Confirmation message for destructive actions
	if ss.PendingConfirm != "" {
		msg := locale.T("confirm." + ss.PendingConfirm)
		printColoredAt(screen, msg, 10, editorBarY+editorBarH+2, color.RGBA{255, 200, 50, 255})
		ss.PendingConfirmTick--
		if ss.PendingConfirmTick <= 0 {
			ss.PendingConfirm = ""
		}
	}

	// Code area: neuro visualization, text editor, or block editor
	if ss.NeuroEnabled {
		drawNeuroVisualization(screen, ss)
	} else if ss.BlockEditorActive {
		DrawBlockEditor(screen, ss)
	} else {
		drawTextEditor(screen, ss, ed)
	}

	// Shared bottom section (error, status, toggles, stats, dropdown)
	drawSwarmEditorBottom(screen, ss, ed)
}

// drawTextEditor renders the traditional text code editor.
func drawTextEditor(screen *ebiten.Image, ss *swarm.SwarmState, ed *swarm.EditorState) {
	// Code editor area background
	vector.DrawFilledRect(screen, 0, float32(editorCodeY), float32(editorPanelW), float32(editorCodeH), ColorSwarmEditorBg, false)

	// Draw visible code lines
	maxVisible := editorCodeH / lineH
	ed.MaxVisible = maxVisible
	startLine := ed.ScrollY
	endLine := startLine + maxVisible
	if endLine > len(ed.Lines) {
		endLine = len(ed.Lines)
	}

	scrollXOff := ed.ScrollX * charW // pixel offset for horizontal scroll

	for li := startLine; li < endLine; li++ {
		screenY := editorCodeY + (li-startLine)*lineH

		// Line number
		lineNumStr := fmt.Sprintf("%3d", li+1)
		printColoredAt(screen, lineNumStr, editorLineNumX, screenY, ColorSwarmLineNum)

		// Syntax-highlighted tokens (with horizontal scroll)
		tokens := swarmscript.TokenizeLine(ed.Lines[li])
		for _, tok := range tokens {
			tokX := editorTextX + tok.Col*charW - scrollXOff
			tokEndX := tokX + len(tok.Text)*charW

			// Skip tokens entirely off-screen to the right
			if tokX >= editorPanelW-2 {
				continue
			}
			// Skip tokens entirely off-screen to the left
			if tokEndX <= editorTextX {
				continue
			}

			tokCol := swarmTokenColor(tok.Type)
			text := tok.Text

			// Clip token text on the left (when partially scrolled off)
			if tokX < editorTextX {
				clipChars := (editorTextX - tokX + charW - 1) / charW
				if clipChars >= len(text) {
					continue
				}
				text = text[clipChars:]
				tokX += clipChars * charW
			}

			// Clip token text on the right (when extending past panel)
			maxChars := (editorPanelW - 2 - tokX) / charW
			if maxChars <= 0 {
				continue
			}
			if maxChars < len(text) {
				text = text[:maxChars]
			}
			printColoredAt(screen, text, tokX, screenY, tokCol)
		}
	}

	// Horizontal scroll indicator
	if ed.ScrollX > 0 {
		printColoredAt(screen, "«", editorTextX-1, editorCodeY+editorCodeH-lineH, ColorSwarmLineNum)
	}

	// Cursor (blinking, visible when editor focused)
	if ed.Focused {
		ed.BlinkTick++
		if (ed.BlinkTick/30)%2 == 0 {
			cursorScreenLine := ed.CursorLine - ed.ScrollY
			if cursorScreenLine >= 0 && cursorScreenLine < maxVisible {
				cx := float32(editorTextX + ed.CursorCol*charW - scrollXOff)
				cy := float32(editorCodeY + cursorScreenLine*lineH)
				if cx >= float32(editorTextX) && cx < float32(editorPanelW-2) {
					vector.StrokeLine(screen, cx, cy, cx, cy+float32(lineH), 2, ColorSwarmCursor, false)
				}
			}
		}
	}

}

// drawSwarmEditorBottom renders the shared bottom part (error, status, toggles, stats).
func drawSwarmEditorBottom(screen *ebiten.Image, ss *swarm.SwarmState, ed *swarm.EditorState) {
	// Error message or hint (compact)
	if ss.ErrorMsg != "" {
		errText := ss.ErrorMsg
		if len(errText) > 55 {
			errText = errText[:55] + "..."
		}
		printColoredAt(screen, errText, 10, editorErrorY, ColorSwarmError)
	} else if ss.ProgramName == "" || ss.ProgramName == "Custom" {
		printColoredAt(screen, locale.T("editor.hint_preset"), 10, editorErrorY, color.RGBA{80, 100, 130, 200})
	}

	// Status bar
	lightStatus := locale.T("editor.off")
	if ss.Light.Active {
		lightStatus = locale.T("editor.on")
	}
	statusText := locale.Tf("editor.status_bar",
		ss.Tick, ss.BotCount, ss.ProgramName, lightStatus)
	printColoredAt(screen, statusText, 10, editorStatusY, ColorMediumGray)

	// Collective AI status indicator (always visible when enabled)
	if ss.CollectiveAIOn {
		aiStatus := locale.T("collective.active_short")
		issueCount := 0
		resolvedCount := 0
		if ss.IssueBoard != nil {
			issueCount = len(ss.IssueBoard.Issues)
			for _, iss := range ss.IssueBoard.Issues {
				if iss.Status == swarm.IssueResolved {
					resolvedCount++
				}
			}
		}
		aiStatusText := fmt.Sprintf("%s [%d/%d]", aiStatus, resolvedCount, issueCount)
		printColoredAt(screen, aiStatusText, 220, editorStatusY, color.RGBA{255, 180, 50, 200})
	}

	// Separator
	vector.StrokeLine(screen, 5, float32(editorSep1Y), float32(editorPanelW-5), float32(editorSep1Y), 1, ColorSwarmEditorSep, false)

	// Bot count input with +/- buttons
	botLabel := fmt.Sprintf("Bots: [%s]", ss.BotCountText)
	if ss.BotCountEdit {
		botLabel = fmt.Sprintf("Bots: [%s_]", ss.BotCountText)
	}
	printColoredAt(screen, botLabel, 10, editorBotCountY, color.RGBA{200, 200, 200, 255})
	drawSwarmButton(screen, 160, editorBotCountY-1, 18, 18, "-", color.RGBA{180, 80, 80, 255})
	drawSwarmButton(screen, 182, editorBotCountY-1, 18, 18, "+", color.RGBA{80, 180, 80, 255})

	// === TABBED PANEL (replaces old toggle rows + stats) ===
	drawTabBar(screen, ss)
	drawTabContent(screen, ss)
	drawCompactStats(screen, ss)

	// Dropdown overlay (drawn on top of everything)
	if ss.DropdownOpen {
		drawSwarmDropdownOverlay(screen, ss)
	}

	// Tooltips for toggle buttons (when hovering)
	drawSwarmTooltips(screen)
}

// drawSwarmStats renders the stats panel at the bottom of the editor.
func drawSwarmStats(screen *ebiten.Image, ss *swarm.SwarmState) {
	y := editorStatsY
	col := color.RGBA{160, 180, 200, 255}
	dimCol := color.RGBA{120, 120, 140, 255}
	headerCol := ColorInfoCyan

	// Section header with subtle background
	vector.DrawFilledRect(screen, 5, float32(y), float32(editorPanelW-10), float32(lineH), color.RGBA{30, 35, 50, 200}, false)
	printColoredAt(screen, locale.T("stats.title"), 10, y, headerCol)
	y += lineH + 2

	// Count chains and max chain length
	chains := 0
	maxChain := 0
	totalNeighbors := 0
	visited := make(map[int]bool)

	for i := range ss.Bots {
		totalNeighbors += ss.Bots[i].NeighborCount

		// Count chain heads (bots that have a follower but no leader)
		if ss.Bots[i].FollowerIdx >= 0 && ss.Bots[i].FollowTargetIdx < 0 && !visited[i] {
			chains++
			chainLen := 1
			cur := ss.Bots[i].FollowerIdx
			for cur >= 0 && cur < len(ss.Bots) && !visited[cur] {
				visited[cur] = true
				chainLen++
				cur = ss.Bots[cur].FollowerIdx
			}
			if chainLen > maxChain {
				maxChain = chainLen
			}
		}
	}

	avgNeighbors := 0.0
	if ss.BotCount > 0 {
		avgNeighbors = float64(totalNeighbors) / float64(ss.BotCount)
	}

	chainInfo := locale.Tf("stat.chains_info", chains, maxChain)
	if chains == 0 {
		chainInfo = locale.T("stat.chains_none")
	}
	printColoredAt(screen, chainInfo, 10, y, col)
	y += lineH
	printColoredAt(screen, locale.Tf("stat.avg_neighbors", avgNeighbors), 10, y, col)
	y += lineH

	// Coverage: how much of the arena is "covered" (divide into 20x20 grid, count occupied cells)
	gridRes := 20
	cellW := ss.ArenaW / float64(gridRes)
	cellH := ss.ArenaH / float64(gridRes)
	occupied := make(map[int]bool)
	for i := range ss.Bots {
		cx := int(ss.Bots[i].X / cellW)
		cy := int(ss.Bots[i].Y / cellH)
		if cx < 0 {
			cx = 0
		}
		if cx >= gridRes {
			cx = gridRes - 1
		}
		if cy < 0 {
			cy = 0
		}
		if cy >= gridRes {
			cy = gridRes - 1
		}
		occupied[cy*gridRes+cx] = true
	}
	coverage := float64(len(occupied)) / float64(gridRes*gridRes) * 100

	coverageHint := locale.T("stat.distributed")
	if coverage < 25 {
		coverageHint = locale.T("stat.clustered")
	} else if coverage > 75 {
		coverageHint = locale.T("stat.well_distributed")
	}
	printColoredAt(screen, locale.Tf("stat.coverage", coverage, coverageHint), 10, y, col)
	y += lineH

	// Trails status
	trailStatus := locale.T("toggle.off")
	if ss.ShowTrails {
		trailStatus = locale.T("toggle.on")
	}
	wrapStatus := locale.T("stat.bounce")
	if ss.WrapMode {
		wrapStatus = locale.T("stat.wrap")
	}
	printColoredAt(screen, locale.Tf("stat.trails_walls", trailStatus, wrapStatus), 10, y, dimCol)
	y += lineH

	// Delivery stats
	if ss.DeliveryOn {
		ds := &ss.DeliveryStats
		// Delivery section header
		vector.DrawFilledRect(screen, 5, float32(y), float32(editorPanelW-10), float32(lineH), color.RGBA{30, 35, 50, 180}, false)
		printColoredAt(screen, locale.T("stat.delivery_header"), 10, y, color.RGBA{255, 200, 100, 200})
		y += lineH + 2

		correctRate := 0
		if ds.TotalDelivered > 0 {
			correctRate = ds.CorrectDelivered * 100 / ds.TotalDelivered
		}
		printColoredAt(screen, locale.Tf("stat.delivered_correct", ds.TotalDelivered, ds.CorrectDelivered, correctRate), 10, y, col)
		y += lineH
		if ds.WrongDelivered > 0 {
			printColoredAt(screen, locale.Tf("stat.wrong_count", ds.WrongDelivered), 10, y, color.RGBA{255, 100, 80, 200})
			y += lineH
		}
		carrying := 0
		idle := 0
		for i := range ss.Bots {
			if ss.Bots[i].CarryingPkg >= 0 {
				carrying++
			} else {
				idle++
			}
		}
		printColoredAt(screen, locale.Tf("stat.carrying_searching", carrying, idle), 10, y, col)
		y += lineH
		avgTime := 0
		if len(ds.DeliveryTimes) > 0 {
			sum := 0
			for _, t := range ds.DeliveryTimes {
				sum += t
			}
			avgTime = sum / len(ds.DeliveryTimes)
		}
		printColoredAt(screen, locale.Tf("stat.avg_delivery_time", avgTime), 10, y, dimCol)
		y += lineH
	}

	// Truck stats (in left panel, replaces old arena snackbar)
	if ss.TruckToggle && ss.TruckState != nil {
		ts := ss.TruckState
		vector.DrawFilledRect(screen, 5, float32(y), float32(editorPanelW-10), float32(lineH), color.RGBA{35, 30, 20, 180}, false)
		printColoredAt(screen, locale.T("stat.truck_header"), 10, y, color.RGBA{255, 180, 80, 200})
		y += lineH + 2

		remaining := 0
		total := 0
		if ts.CurrentTruck != nil {
			total = len(ts.CurrentTruck.Packages)
			for _, p := range ts.CurrentTruck.Packages {
				if !p.PickedUp {
					remaining++
				}
			}
		}
		printColoredAt(screen, locale.Tf("stat.truck_packages", ts.TruckNum, ts.TrucksPerRound, total-remaining, total), 10, y, col)
		y += lineH
		printColoredAt(screen, locale.Tf("stat.points", ts.Score), 10, y, color.RGBA{255, 200, 100, 200})
	}
}

// DrawSwarmHUD draws the bottom help line for swarm mode.
func DrawSwarmHUD(screen *ebiten.Image, s *simulation.Simulation, fps float64) {
	sh := screen.Bounds().Dy()
	ss := s.SwarmState

	// FPS + speed at top of arena area
	speedStr := fmt.Sprintf("%.1fx", s.Speed)
	if s.Speed < 1.0 {
		speedStr = fmt.Sprintf("%.3gx", s.Speed) // show 0.25x, 0.125x properly
	}
	info := locale.Tf("ui.fps_speed", fps, speedStr)
	if s.Paused {
		info += " [PAUSE]"
	}
	printColoredAt(screen, info, 360, 10, ColorWhite)

	// Slow-motion indicator
	if s.Speed < 1.0 && !s.Paused {
		label := locale.Tf("ui.slowmo", speedStr)
		printColoredAt(screen, label, 600, 10, color.RGBA{100, 200, 255, 230})
	}

	// Arena info line
	if ss != nil {
		arenaInfo := fmt.Sprintf("Bots: %s | %s | Tick: %s (%s)",
			fmtNum(ss.BotCount), ss.ProgramName, fmtNum(ss.Tick), fmtTime(ss.Tick))
		printColoredAt(screen, arenaInfo, 420, 35, ColorWhite)

		// Active algorithm indicator
		if ss.SwarmAlgo != nil {
			algoLabel := locale.Tf("ui.algorithm_label",
				swarm.SwarmAlgorithmName(ss.SwarmAlgo.ActiveAlgo))
			printColoredAt(screen, algoLabel, 420, 50, color.RGBA{200, 255, 100, 230})
		}

		// Dynamic environment indicator
		if ss.DynamicEnv {
			printColoredAt(screen, locale.T("ui.dynamic"), 750, 35, color.RGBA{255, 150, 50, 220})
		}

		// Color filter indicator
		if ss.ColorFilter > 0 {
			filterNames := []string{"", locale.T("colorfilter.red"), locale.T("colorfilter.green"), locale.T("colorfilter.blue"), locale.T("colorfilter.carry"), locale.T("colorfilter.idle")}
			filterColors := []color.RGBA{
				{}, {255, 80, 80, 255}, {80, 255, 80, 255}, {80, 80, 255, 255},
				{255, 200, 50, 255}, {150, 150, 150, 255},
			}
			printColoredAt(screen, filterNames[ss.ColorFilter], 850, 35, filterColors[ss.ColorFilter])
		}

		// Message wave indicator
		if ss.ShowMsgWaves {
			printColoredAt(screen, locale.T("ui.waves"), 950, 35, color.RGBA{100, 200, 255, 220})
		}

		// Memory indicator
		if ss.MemoryEnabled {
			printColoredAt(screen, "MEMORY", 1030, 35, color.RGBA{200, 150, 255, 220})
		}

		// Sensor noise indicator
		if ss.SensorNoiseOn {
			printColoredAt(screen, locale.T("ui.noise"), 1110, 35, color.RGBA{255, 120, 80, 220})
		}

		// Day/Night indicator
		if ss.DayNightOn {
			brightness := swarm.DayNightBrightness(ss)
			timeLabel := locale.T("ui.day")
			timeCol := color.RGBA{255, 220, 80, 220}
			if brightness < 0.3 {
				timeLabel = locale.T("ui.night")
				timeCol = color.RGBA{80, 80, 200, 220}
			} else if brightness < 0.7 {
				timeLabel = locale.T("ui.dusk")
				timeCol = color.RGBA{200, 150, 100, 220}
			}
			printColoredAt(screen, timeLabel, 1200, 35, timeCol)
		}

		// Aurora indicator
		if ss.AuroraOn {
			printColoredAt(screen, "AURORA", 1200, 22, color.RGBA{100, 255, 200, 220})
		}

		// Achievement counter
		if ss.AchievementState != nil && ss.AchievementState.TotalUnlocked > 0 {
			achLabel := fmt.Sprintf("%d/%d", ss.AchievementState.TotalUnlocked, int(swarm.AchCount))
			printColoredAt(screen, achLabel, 1200, 48, ColorGoldFaded)
		}

		// Reset flash indicator
		if ss.ResetFlashTimer > 0 {
			flashAlpha := uint8(255)
			if ss.ResetFlashTimer < 10 {
				flashAlpha = uint8(ss.ResetFlashTimer * 25)
			}
			printColoredAt(screen, "RESET", 700, 35, color.RGBA{255, 255, 50, flashAlpha})
		}

		// Delivery & Truck stats are shown in the left editor panel (STATISTIKEN section)

		// Session save flash (Ctrl+S)
		if ss.SessionSaveFlash > 0 {
			flashAlpha := uint8(255)
			if ss.SessionSaveFlash < 15 {
				flashAlpha = uint8(ss.SessionSaveFlash * 17)
			}
			printColoredAt(screen, "Session saved", 600, 35, color.RGBA{80, 220, 120, flashAlpha})
		}

		// Deploy success flash
		if ss.DeployFlash > 0 {
			alpha := uint8(255)
			if ss.DeployFlash < 15 {
				alpha = uint8(ss.DeployFlash * 17)
			}
			msg := locale.Tf("deploy.success", ss.DeployRuleCount)
			printColoredAt(screen, msg, 370, 30, color.RGBA{80, 220, 80, alpha})
		}
	}

	// --- Quick-stats bar at very top of arena area ---
	if ss != nil {
		sw := screen.Bounds().Dx()
		drawSwarmQuickStats(screen, ss, s, fps, sw)
	}

	// Help hint at very bottom
	printColoredAt(screen, locale.T("ui.bottom_hint"),
		360, sh-14, color.RGBA{120, 130, 150, 180})

	// Scenario title overlay
	if s.ScenarioTimer > 0 {
		sw := screen.Bounds().Dx()
		drawScenarioTitle(screen, s.ScenarioTitle, sw, sh, s.ScenarioTimer)
	}
}

func drawSwarmDropdownButton(screen *ebiten.Image, ss *swarm.SwarmState) {
	// Dropdown button: [▼ PresetName]
	x, y, w, h := 5, editorBarY, 185, editorBarH
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), ColorSwarmBtnPreset, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, color.RGBA{200, 200, 200, 100}, false)

	label := fmt.Sprintf("v %s", ss.ProgramName)
	if len(label) > 28 {
		label = label[:28] + ".."
	}
	textX := x + 5
	textY := y + (h-12)/2
	printColoredAt(screen, label, textX, textY, ColorWhite)
}

// presetCategories assigns category labels to preset indices for the dropdown.
// presetCategoryKeys maps preset index to locale key for category labels.
var presetCategoryKeys = map[int]string{
	0:  "preset.cat_simple",
	10: "preset.cat_delivery",
	13: "preset.cat_truck",
	15: "preset.cat_selflearn",
	17: "preset.cat_advanced",
	20: "preset.cat_neural",
	22: "preset.cat_explore",
}

func presetCategory(idx int) (string, bool) {
	key, ok := presetCategoryKeys[idx]
	if !ok {
		return "", false
	}
	return locale.T(key), true
}

// presetShortDesc gives a brief one-line description for each preset.
// presetShortDescKeys maps preset name to locale key for short descriptions.
var presetShortDescKeys = map[string]string{
	"Aggregation":        "preset.desc_aggregation",
	"Dispersion":         "preset.desc_dispersion",
	"Orbit":              "preset.desc_orbit",
	"Color Wave":         "preset.desc_color_wave",
	"Flocking":           "preset.desc_flocking",
	"Snake Formation":    "preset.desc_snake",
	"Obstacle Nav":       "preset.desc_obstacle",
	"Pulse Sync":         "preset.desc_pulse_sync",
	"Trail Follow":       "preset.desc_trail_follow",
	"Ant Colony":         "preset.desc_ant_colony",
	"Simple Delivery":    "preset.desc_simple_delivery",
	"Delivery Comm":      "preset.desc_delivery_comm",
	"Delivery Roles":     "preset.desc_delivery_roles",
	"Simple Unload":      "preset.desc_simple_unload",
	"Coordinated Unload": "preset.desc_coord_unload",
	"Evolving Delivery":  "preset.desc_evolving_delivery",
	"Evolving Truck":     "preset.desc_evolving_truck",
	"Maze Explorer":      "preset.desc_maze",
	"GP: Random Start":   "preset.desc_gp_random",
	"GP: Seeded Start":   "preset.desc_gp_seeded",
	"Neuro: Delivery":    "preset.desc_neuro_delivery",
	"Neuro: LKW":         "preset.desc_neuro_truck",
	"Phototaxis":         "preset.desc_phototaxis",
	"Braitenberg 2b":     "preset.desc_braitenberg",
	"Stigmergy":          "preset.desc_stigmergy",
}

func drawSwarmDropdownOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	x := 5
	y := editorBarY + editorBarH
	w := 220
	itemH := 28 // taller to fit name + description

	// Semi-transparent backdrop behind dropdown
	totalH := len(ss.Presets)*itemH + 10
	vector.DrawFilledRect(screen, float32(x-2), float32(y-2), float32(w+4), float32(totalH+4),
		color.RGBA{20, 25, 40, 250}, false)
	vector.StrokeRect(screen, float32(x-2), float32(y-2), float32(w+4), float32(totalH+4),
		1, color.RGBA{100, 140, 200, 150}, false)

	for i, preset := range ss.Presets {
		name := preset.Name
		iy := y + i*itemH

		// Category separator
		if cat, ok := presetCategory(i); ok && i > 0 {
			vector.DrawFilledRect(screen, float32(x), float32(iy-1), float32(w), 1, color.RGBA{80, 100, 140, 150}, false)
			printColoredAt(screen, cat, x+w-runeLen(cat)*charW-5, iy+5, color.RGBA{100, 140, 200, 150})
		}

		bgCol := color.RGBA{35, 45, 70, 240}
		if i == ss.DropdownHover {
			bgCol = color.RGBA{60, 90, 160, 240}
		}
		vector.DrawFilledRect(screen, float32(x), float32(iy), float32(w), float32(itemH), bgCol, false)

		// Highlight current program
		if name == ss.ProgramName {
			printColoredAt(screen, ">", x+3, iy+3, color.RGBA{136, 204, 255, 255})
		}
		printColoredAt(screen, name, x+12, iy+3, color.RGBA{220, 225, 240, 255})
		// Short description on second line (if available)
		if key, ok := presetShortDescKeys[name]; ok {
			desc := locale.T(key)
			printColoredAt(screen, desc, x+12, iy+13, color.RGBA{100, 110, 130, 200})
		}
	}
}

// drawSwarmTooltips shows tooltip text when hovering over toggle buttons.
func drawSwarmTooltips(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()

	// Define tooltip zones: (x, y, w, h, text)
	type tooltipZone struct {
		x, y, w, h int
		text        string
	}
	zones := []tooltipZone{
		{5, editorToggle1Y, toggleBtnW, toggleBtnH, locale.T("editor.tip_obstacles")},
		{175, editorToggle1Y, toggleBtnW, toggleBtnH, locale.T("editor.tip_maze")},
		{5, editorToggle2Y, toggleBtnW, toggleBtnH, locale.T("editor.tip_light")},
		{175, editorToggle2Y, toggleBtnW, toggleBtnH, locale.T("editor.tip_wrap")},
		{5, editorToggle3Y, toggleBtnW, toggleBtnH, locale.T("editor.tip_delivery")},
		{175, editorToggle3Y, toggleBtnW, toggleBtnH, locale.T("editor.tip_truck")},
	}

	for _, z := range zones {
		if mx >= z.x && mx < z.x+z.w && my >= z.y && my < z.y+z.h {
			// Draw tooltip near cursor
			tipX := mx + 10
			tipY := my - lineH - 6
			tipW := len(z.text)*charW + 8

			// Keep tooltip on screen
			sw := screen.Bounds().Dx()
			if tipX+tipW > sw {
				tipX = sw - tipW - 5
			}
			if tipY < 0 {
				tipY = my + 20
			}

			vector.DrawFilledRect(screen, float32(tipX-3), float32(tipY-2), float32(tipW), float32(lineH+4),
				color.RGBA{30, 30, 50, 230}, false)
			vector.StrokeRect(screen, float32(tipX-3), float32(tipY-2), float32(tipW), float32(lineH+4),
				1, color.RGBA{100, 100, 130, 200}, false)
			printColoredAt(screen, z.text, tipX, tipY, color.RGBA{200, 200, 220, 255})
			break // only one tooltip at a time
		}
	}
}

// swarmTokenColor maps a SwarmScript token type to a display color.
func swarmTokenColor(t swarmscript.SwarmTokenType) color.RGBA {
	switch t {
	case swarmscript.TokKeyword:
		return ColorSwarmKeyword
	case swarmscript.TokCondition:
		return ColorSwarmCondition
	case swarmscript.TokAction:
		return ColorSwarmAction
	case swarmscript.TokNumber:
		return ColorSwarmNumber
	case swarmscript.TokComment:
		return ColorSwarmComment
	case swarmscript.TokOperator:
		return ColorSwarmOperator
	default:
		return color.RGBA{200, 200, 200, 255}
	}
}

// --- Hit-test helpers for editor buttons ---

// SwarmEditorHitTest checks what was clicked in the editor panel.
// Returns: "dropdown", "deploy", "reset", "botcount", "editor",
// "obstacles", "maze", "light", "walls", "delivery", "trucks", or ""
func SwarmEditorHitTest(mx, my int, ss *swarm.SwarmState) string {
	if mx > editorPanelW {
		return ""
	}

	// Algo-Labor mode has its own hit test
	if ss.AlgoLaborMode {
		return AlgoLaborHitTest(mx, my, ss)
	}

	// TEXT/BLOCKS toggle buttons (in title bar area)
	if my >= editorTitleY && my < editorTitleY+16 {
		if mx >= 210 && mx < 255 {
			return "text_mode"
		}
		if mx >= 258 && mx < 318 {
			return "block_mode"
		}
	}

	// Dropdown button
	if my >= editorBarY && my < editorBarY+editorBarH && mx >= 5 && mx < 190 {
		return "dropdown"
	}

	// Deploy button
	if my >= editorBarY && my < editorBarY+editorBarH && mx >= 195 && mx < 255 {
		return "deploy"
	}

	// Reset button
	if my >= editorBarY && my < editorBarY+editorBarH && mx >= 258 && mx < 298 {
		return "reset"
	}

	// Copy button
	if my >= editorBarY && my < editorBarY+editorBarH && mx >= 300 && mx < 323 {
		return "copy"
	}

	// Paste button
	if my >= editorBarY && my < editorBarY+editorBarH && mx >= 325 && mx < 348 {
		return "paste"
	}

	// Code editor area
	if my >= editorCodeY && my < editorCodeY+editorCodeH {
		return "editor"
	}

	// Bot count field
	if my >= editorBotCountY-1 && my < editorBotCountY+18 {
		if mx >= 160 && mx < 178 {
			return "bots_minus"
		}
		if mx >= 182 && mx < 200 {
			return "bots_plus"
		}
		if mx >= 5 && mx < 160 {
			return "botcount"
		}
	}

	// Tab bar
	if tabHit := TabBarHitTest(mx, my); tabHit != "" {
		return tabHit
	}

	// Tab content (toggles, algo list, tools)
	if ss != nil {
		if contentHit := TabContentHitTest(mx, my, ss); contentHit != "" {
			return contentHit
		}
	}

	return ""
}

// SwarmEditorClickToPos converts a click in the editor area to (line, col) in editor coordinates.
func SwarmEditorClickToPos(mx, my int, ed *swarm.EditorState) (int, int) {
	relY := my - editorCodeY
	clickLine := ed.ScrollY + relY/lineH
	if clickLine < 0 {
		clickLine = 0
	}
	if clickLine >= len(ed.Lines) {
		clickLine = len(ed.Lines) - 1
	}

	relX := mx - editorTextX
	clickCol := relX/charW + ed.ScrollX // account for horizontal scroll
	if clickCol < 0 {
		clickCol = 0
	}
	if clickLine >= 0 && clickLine < len(ed.Lines) {
		lineLen := len(ed.Lines[clickLine])
		if clickCol > lineLen {
			clickCol = lineLen
		}
	}

	return clickLine, clickCol
}

// SwarmDropdownHitTest returns the preset index at the given mouse position, or -1.
func SwarmDropdownHitTest(mx, my int, presetCount int) int {
	x := 5
	y := editorBarY + editorBarH
	w := 220
	itemH := 28 // matches dropdown overlay item height

	if mx < x || mx > x+w {
		return -1
	}
	for i := 0; i < presetCount; i++ {
		iy := y + i*itemH
		if my >= iy && my < iy+itemH {
			return i
		}
	}
	return -1
}

// drawSwarmQuickStats renders a thin persistent stats bar at the very top of
// the arena area (right of the editor panel). It shows key metrics at a glance.
func drawSwarmQuickStats(screen *ebiten.Image, ss *swarm.SwarmState, s *simulation.Simulation, fps float64, sw int) {
	barH := 14
	barX := 355 // right of editor panel
	barW := sw - barX

	// Semi-transparent dark background
	vector.DrawFilledRect(screen, float32(barX), 0, float32(barW), float32(barH), color.RGBA{15, 18, 25, 200}, false)

	// Build stats line
	speedStr := fmt.Sprintf("%.0fx", s.Speed)
	if s.Speed < 1.0 {
		speedStr = fmt.Sprintf("%.3gx", s.Speed)
	}

	stats := fmt.Sprintf("Bots:%s | Tick:%s (%s) | Speed:%s | FPS:%.0f",
		fmtNum(ss.BotCount), fmtNum(ss.Tick), fmtTime(ss.Tick), speedStr, fps)

	// Add delivery stats if active
	if ss.DeliveryOn && ss.DeliveryStats.TotalDelivered > 0 {
		pct := 0
		if ss.DeliveryStats.TotalDelivered > 0 {
			pct = ss.DeliveryStats.CorrectDelivered * 100 / ss.DeliveryStats.TotalDelivered
		}
		stats += fmt.Sprintf(" | Del:%s/%s (%d%%)",
			fmtNum(ss.DeliveryStats.CorrectDelivered), fmtNum(ss.DeliveryStats.TotalDelivered), pct)
	}

	// Add evolution generation and fitness if active
	if ss.EvolutionOn {
		stats += fmt.Sprintf(" | Gen:%d | Fit:%.0f", ss.Generation, ss.BestFitness)
	}

	dimCol := color.RGBA{130, 140, 160, 200}
	printColoredAt(screen, stats, barX+6, 2, dimCol)
}
