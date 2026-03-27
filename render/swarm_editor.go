package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"swarmsim/engine/swarmscript"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	charW          = 6  // monospace char width for DebugPrintAt
	lineH          = 16 // line height in pixels

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
	ebitenutil.DebugPrintAt(screen, "SwarmScript Editor", 10, editorTitleY)
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
		printColoredAt(screen, "Tipp: Preset waehlen oder eigenes Programm schreiben", 10, editorErrorY, color.RGBA{80, 100, 130, 200})
	}

	// Status bar
	lightStatus := "AUS"
	if ss.Light.Active {
		lightStatus = "AN"
	}
	statusText := fmt.Sprintf("Tick: %d | Bots: %d | Programm: %s | Licht: %s",
		ss.Tick, ss.BotCount, ss.ProgramName, lightStatus)
	printColoredAt(screen, statusText, 10, editorStatusY, color.RGBA{180, 180, 180, 255})

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
	headerCol := color.RGBA{136, 204, 255, 220}

	// Section header with subtle background
	vector.DrawFilledRect(screen, 5, float32(y), float32(editorPanelW-10), float32(lineH), color.RGBA{30, 35, 50, 200}, false)
	printColoredAt(screen, "STATISTIKEN", 10, y, headerCol)
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

	chainInfo := fmt.Sprintf("Ketten: %d  Laengste: %d", chains, maxChain)
	if chains == 0 {
		chainInfo = "Ketten: keine (FOLLOW_NEAREST noetig)"
	}
	printColoredAt(screen, chainInfo, 10, y, col)
	y += lineH
	printColoredAt(screen, fmt.Sprintf("Avg Nachbarn: %.1f (in 120px)", avgNeighbors), 10, y, col)
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

	coverageHint := "verteilt"
	if coverage < 25 {
		coverageHint = "geclustert"
	} else if coverage > 75 {
		coverageHint = "gut verteilt"
	}
	printColoredAt(screen, fmt.Sprintf("Abdeckung: %.0f%% (%s)", coverage, coverageHint), 10, y, col)
	y += lineH

	// Trails status
	trailStatus := "AUS"
	if ss.ShowTrails {
		trailStatus = "AN"
	}
	wrapStatus := "Abprall"
	if ss.WrapMode {
		wrapStatus = "Durchlauf"
	}
	printColoredAt(screen, fmt.Sprintf("Spuren: %s | Rand: %s", trailStatus, wrapStatus), 10, y, dimCol)
	y += lineH

	// Delivery stats
	if ss.DeliveryOn {
		ds := &ss.DeliveryStats
		// Delivery section header
		vector.DrawFilledRect(screen, 5, float32(y), float32(editorPanelW-10), float32(lineH), color.RGBA{30, 35, 50, 180}, false)
		printColoredAt(screen, "LIEFERUNG", 10, y, color.RGBA{255, 200, 100, 200})
		y += lineH + 2

		correctRate := 0
		if ds.TotalDelivered > 0 {
			correctRate = ds.CorrectDelivered * 100 / ds.TotalDelivered
		}
		printColoredAt(screen, fmt.Sprintf("Geliefert: %d | Richtig: %d (%d%%)", ds.TotalDelivered, ds.CorrectDelivered, correctRate), 10, y, col)
		y += lineH
		if ds.WrongDelivered > 0 {
			printColoredAt(screen, fmt.Sprintf("Falsch: %d", ds.WrongDelivered), 10, y, color.RGBA{255, 100, 80, 200})
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
		printColoredAt(screen, fmt.Sprintf("Tragen: %d | Suchen: %d", carrying, idle), 10, y, col)
		y += lineH
		avgTime := 0
		if len(ds.DeliveryTimes) > 0 {
			sum := 0
			for _, t := range ds.DeliveryTimes {
				sum += t
			}
			avgTime = sum / len(ds.DeliveryTimes)
		}
		printColoredAt(screen, fmt.Sprintf("Durchschn. Lieferzeit: %d Ticks", avgTime), 10, y, dimCol)
		y += lineH
	}

	// Truck stats (in left panel, replaces old arena snackbar)
	if ss.TruckToggle && ss.TruckState != nil {
		ts := ss.TruckState
		vector.DrawFilledRect(screen, 5, float32(y), float32(editorPanelW-10), float32(lineH), color.RGBA{35, 30, 20, 180}, false)
		printColoredAt(screen, "LKW-ENTLADUNG", 10, y, color.RGBA{255, 180, 80, 200})
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
		printColoredAt(screen, fmt.Sprintf("LKW %d/%d | Pakete: %d/%d", ts.TruckNum, ts.TrucksPerRound, total-remaining, total), 10, y, col)
		y += lineH
		printColoredAt(screen, fmt.Sprintf("Punkte: %d", ts.Score), 10, y, color.RGBA{255, 200, 100, 200})
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
	info := fmt.Sprintf("FPS: %.0f  Tempo: %s", fps, speedStr)
	if s.Paused {
		info += " [PAUSE]"
	}
	ebitenutil.DebugPrintAt(screen, info, 360, 10)

	// Slow-motion indicator
	if s.Speed < 1.0 && !s.Paused {
		label := fmt.Sprintf("ZEITLUPE %s", speedStr)
		printColoredAt(screen, label, 600, 10, color.RGBA{100, 200, 255, 230})
	}

	// Arena info line
	if ss != nil {
		arenaInfo := fmt.Sprintf("Bots: %d | %s | Tick: %d",
			ss.BotCount, ss.ProgramName, ss.Tick)
		ebitenutil.DebugPrintAt(screen, arenaInfo, 420, 35)

		// Active algorithm indicator
		if ss.SwarmAlgo != nil {
			algoLabel := fmt.Sprintf("Algorithmus: %s",
				swarm.SwarmAlgorithmName(ss.SwarmAlgo.ActiveAlgo))
			printColoredAt(screen, algoLabel, 420, 50, color.RGBA{200, 255, 100, 230})
		}

		// Dynamic environment indicator
		if ss.DynamicEnv {
			printColoredAt(screen, "DYNAMISCH", 750, 35, color.RGBA{255, 150, 50, 220})
		}

		// Color filter indicator
		if ss.ColorFilter > 0 {
			filterNames := []string{"", "Filter: Rot", "Filter: Gruen", "Filter: Blau", "Filter: Traegt", "Filter: Wartend"}
			filterColors := []color.RGBA{
				{}, {255, 80, 80, 255}, {80, 255, 80, 255}, {80, 80, 255, 255},
				{255, 200, 50, 255}, {150, 150, 150, 255},
			}
			printColoredAt(screen, filterNames[ss.ColorFilter], 850, 35, filterColors[ss.ColorFilter])
		}

		// Message wave indicator
		if ss.ShowMsgWaves {
			printColoredAt(screen, "WELLEN", 950, 35, color.RGBA{100, 200, 255, 220})
		}

		// Memory indicator
		if ss.MemoryEnabled {
			printColoredAt(screen, "MEMORY", 1030, 35, color.RGBA{200, 150, 255, 220})
		}

		// Sensor noise indicator
		if ss.SensorNoiseOn {
			printColoredAt(screen, "RAUSCHEN", 1110, 35, color.RGBA{255, 120, 80, 220})
		}

		// Day/Night indicator
		if ss.DayNightOn {
			brightness := swarm.DayNightBrightness(ss)
			timeLabel := "TAG"
			timeCol := color.RGBA{255, 220, 80, 220}
			if brightness < 0.3 {
				timeLabel = "NACHT"
				timeCol = color.RGBA{80, 80, 200, 220}
			} else if brightness < 0.7 {
				timeLabel = "DAEMMERUNG"
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
			printColoredAt(screen, achLabel, 1200, 48, color.RGBA{255, 215, 0, 200})
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
	}

	// Help hint at very bottom
	printColoredAt(screen, "SPACE:Pause  H:Hilfe  |  Alle Features per Maus in den 4 Tabs links steuerbar",
		360, sh-14, color.RGBA{120, 130, 150, 180})

	// Scenario title overlay
	if s.ScenarioTimer > 0 {
		sw := screen.Bounds().Dx()
		drawScenarioTitle(screen, s.ScenarioTitle, sw, sh, s.ScenarioTimer)
	}
}

// --- Helper drawing functions ---

func drawSwarmButton(screen *ebiten.Image, x, y, w, h int, label string, bgCol color.RGBA) {
	// Main body
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), bgCol, false)
	// Top highlight (subtle light edge)
	highlightCol := color.RGBA{
		uint8(min(int(bgCol.R)+30, 255)),
		uint8(min(int(bgCol.G)+30, 255)),
		uint8(min(int(bgCol.B)+30, 255)),
		bgCol.A,
	}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), 1, highlightCol, false)
	// Bottom shadow
	shadowCol := color.RGBA{
		uint8(max(int(bgCol.R)-25, 0)),
		uint8(max(int(bgCol.G)-25, 0)),
		uint8(max(int(bgCol.B)-25, 0)),
		bgCol.A,
	}
	vector.DrawFilledRect(screen, float32(x), float32(y+h-1), float32(w), 1, shadowCol, false)
	// Border
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, color.RGBA{180, 180, 200, 80}, false)
	textX := x + (w-len(label)*charW)/2
	textY := y + (h-12)/2
	ebitenutil.DebugPrintAt(screen, label, textX, textY)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	ebitenutil.DebugPrintAt(screen, label, textX, textY)
}

// presetCategories assigns category labels to preset indices for the dropdown.
var presetCategories = map[int]string{
	0:  "Einfache Beispiele",
	10: "Paket-Lieferung",
	13: "LKW-Entladung",
	15: "Selbstlernend",
	17: "Fortgeschritten",
	20: "Neuronale Netze",
}

// presetShortDesc gives a brief one-line description for each preset.
var presetShortDesc = map[string]string{
	"Aggregation":        "Bots finden sich zu Gruppen",
	"Dispersion":         "Bots verteilen sich im Raum",
	"Orbit":              "Bots kreisen ums Licht",
	"Color Wave":         "Farbwelle breitet sich aus",
	"Flocking":           "Vogelschwarm-Verhalten",
	"Snake Formation":    "Bots bilden Ketten",
	"Obstacle Nav":       "Hindernisse umfahren",
	"Pulse Sync":         "Gluehwuermchen-Blinken",
	"Trail Follow":       "Farben vom Nachbarn kopieren",
	"Ant Colony":         "Ameisenkolonie mit Nachrichten",
	"Simple Delivery":    "Pakete aufheben und abliefern",
	"Delivery Comm":      "Liefern mit Kommunikation",
	"Delivery Roles":     "Spaeh-Bots + Liefer-Bots",
	"Simple Unload":      "LKW entladen ohne Absprache",
	"Coordinated Unload": "LKW entladen mit Wegweisern",
	"Evolving Delivery":  "Werte optimieren sich selbst",
	"Evolving Truck":     "Truck-Werte optimieren sich",
	"Maze Explorer":      "Immer rechts an der Wand lang",
	"GP: Random Start":   "Programme entstehen von Null",
	"GP: Seeded Start":   "Gutes Programm wird besser",
	"Neuro: Delivery":    "Gehirn lernt ohne Regeln!",
	"Neuro: LKW":         "Gehirn lernt LKW entladen!",
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
		if cat, ok := presetCategories[i]; ok && i > 0 {
			vector.DrawFilledRect(screen, float32(x), float32(iy-1), float32(w), 1, color.RGBA{80, 100, 140, 150}, false)
			printColoredAt(screen, cat, x+w-len(cat)*charW-5, iy+5, color.RGBA{100, 140, 200, 150})
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
		if desc, ok := presetShortDesc[name]; ok {
			printColoredAt(screen, desc, x+12, iy+13, color.RGBA{100, 110, 130, 200})
		}
	}
}

// textCacheEntry holds a cached colored text image.
type textCacheEntry struct {
	img      *ebiten.Image
	lastUsed int
}

var (
	textCache      = make(map[string]*textCacheEntry, 128)
	textCacheFrame int
)

// tickTextCache increments the frame counter and evicts stale entries.
// Call once per frame from the draw entry point.
func tickTextCache() {
	textCacheFrame++
	if textCacheFrame%120 == 0 {
		for k, e := range textCache {
			if textCacheFrame-e.lastUsed > 120 {
				e.img.Deallocate()
				delete(textCache, k)
			}
		}
	}
}

// cachedTextImage returns a cached white-text image for the given string.
// Used by HUD functions that need a scaled text image.
func cachedTextImage(text string) *ebiten.Image {
	key := "__hud__" + text
	entry, ok := textCache[key]
	if !ok {
		tw := len(text)*6 + 10
		if tw < 1 {
			tw = 1
		}
		th := 16
		img := ebiten.NewImage(tw, th)
		ebitenutil.DebugPrintAt(img, text, 5, 3)
		entry = &textCacheEntry{img: img}
		textCache[key] = entry
	}
	entry.lastUsed = textCacheFrame
	return entry.img
}

// printColoredAt draws colored text at the given position.
// Uses a text image cache to avoid per-frame GPU image allocations.
func printColoredAt(screen *ebiten.Image, text string, x, y int, col color.RGBA) {
	if text == "" {
		return
	}

	// Build cache key: text + color bytes
	key := text + string([]byte{col.R, col.G, col.B, col.A})

	entry, ok := textCache[key]
	if !ok {
		tw := len(text)*charW + 2
		if tw < 1 {
			tw = 1
		}
		th := lineH
		img := ebiten.NewImage(tw, th)
		ebitenutil.DebugPrintAt(img, text, 0, 0)
		entry = &textCacheEntry{img: img}
		textCache[key] = entry
	}
	entry.lastUsed = textCacheFrame

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))

	// DebugPrintAt draws white text; tint to desired color
	r := float64(col.R) / 255.0
	g := float64(col.G) / 255.0
	b := float64(col.B) / 255.0
	a := float64(col.A) / 255.0
	op.ColorScale.Scale(float32(r), float32(g), float32(b), float32(a))

	screen.DrawImage(entry.img, op)
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
		{5, editorToggle1Y, toggleBtnW, toggleBtnH, "Zufaellige Hindernisse im Feld"},
		{175, editorToggle1Y, toggleBtnW, toggleBtnH, "Labyrinth mit Gaengen und Waenden"},
		{5, editorToggle2Y, toggleBtnW, toggleBtnH, "Lichtquelle fuer light_value Sensor"},
		{175, editorToggle2Y, toggleBtnW, toggleBtnH, "BOUNCE=Abprallen WRAP=Durchlaufen"},
		{5, editorToggle3Y, toggleBtnW, toggleBtnH, "Paket-Liefersystem mit Stationen"},
		{175, editorToggle3Y, toggleBtnW, toggleBtnH, "LKW-Entladung mit Rampe"},
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
