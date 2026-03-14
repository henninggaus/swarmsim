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
	editorSep1Y = 636

	// Bot count input
	editorBotCountY = 642

	// Toggle buttons row 1
	editorToggle1Y = 662

	// Toggle buttons row 2
	editorToggle2Y = 688

	// Toggle buttons row 3
	editorToggle3Y = 714

	// Separator 2
	editorSep2Y = 740

	// Stats panel
	editorStatsY = 746

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
	if ss == nil || ss.Editor == nil {
		return
	}
	tickTextCache() // maintain text image cache
	ed := ss.Editor

	// Title bar
	ebitenutil.DebugPrintAt(screen, "SwarmScript Editor", 10, editorTitleY)

	// Button bar: [▼ PresetName] [DEPLOY] [RESET]
	drawSwarmDropdownButton(screen, ss)
	drawSwarmButton(screen, 195, editorBarY, 75, editorBarH, "DEPLOY", ColorSwarmBtnDeploy)
	drawSwarmButton(screen, 275, editorBarY, 65, editorBarH, "RESET", ColorSwarmBtnReset)

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

	// Calculate visible column range
	maxVisibleCols := (editorPanelW - 2 - editorTextX) / charW
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

	// Horizontal scroll indicator (shows "<<" or ">>" if content is scrolled)
	_ = maxVisibleCols
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

	// Error message
	if ss.ErrorMsg != "" {
		errText := ss.ErrorMsg
		if len(errText) > 55 {
			errText = errText[:55] + "..."
		}
		printColoredAt(screen, errText, 10, editorErrorY, ColorSwarmError)
	}

	// Status bar
	lightStatus := "OFF"
	if ss.Light.Active {
		lightStatus = "ON"
	}
	statusText := fmt.Sprintf("Tick:%d Bots:%d Prog:%s Light:%s",
		ss.Tick, ss.BotCount, ss.ProgramName, lightStatus)
	printColoredAt(screen, statusText, 10, editorStatusY, color.RGBA{180, 180, 180, 255})

	// Separator 1
	vector.StrokeLine(screen, 5, float32(editorSep1Y), float32(editorPanelW-5), float32(editorSep1Y), 1, ColorSwarmEditorSep, false)

	// Bot count input
	botLabel := fmt.Sprintf("Bots: [%s]", ss.BotCountText)
	if ss.BotCountEdit {
		botLabel = fmt.Sprintf("Bots: [%s_]", ss.BotCountText)
	}
	printColoredAt(screen, botLabel, 10, editorBotCountY, color.RGBA{200, 200, 200, 255})

	// Toggle buttons row 1: [Obstacles: OFF] [Maze: OFF]
	obsLabel := "Obstacles: OFF"
	obsColor := ColorSwarmBtnToggleOff
	if ss.ObstaclesOn {
		obsLabel = "Obstacles: ON"
		obsColor = ColorSwarmBtnToggleOn
	}
	drawSwarmButton(screen, 5, editorToggle1Y, toggleBtnW, toggleBtnH, obsLabel, obsColor)

	mazeLabel := "Maze: OFF"
	mazeColor := ColorSwarmBtnToggleOff
	if ss.MazeOn {
		mazeLabel = "Maze: ON"
		mazeColor = ColorSwarmBtnToggleOn
	}
	drawSwarmButton(screen, 175, editorToggle1Y, toggleBtnW, toggleBtnH, mazeLabel, mazeColor)

	// Toggle buttons row 2: [Light: OFF] [Walls: BOUNCE]
	lightLabel := "Light: OFF"
	lightColor := ColorSwarmBtnToggleOff
	if ss.Light.Active {
		lightLabel = "Light: ON"
		lightColor = ColorSwarmBtnToggleOn
	}
	drawSwarmButton(screen, 5, editorToggle2Y, toggleBtnW, toggleBtnH, lightLabel, lightColor)

	wallLabel := "Walls: BOUNCE"
	wallColor := ColorSwarmBtnToggleOff
	if ss.WrapMode {
		wallLabel = "Walls: WRAP"
		wallColor = color.RGBA{60, 60, 140, 255}
	}
	drawSwarmButton(screen, 175, editorToggle2Y, toggleBtnW, toggleBtnH, wallLabel, wallColor)

	// Toggle buttons row 3: [Delivery: OFF/ON/—]
	if ss.ProgramName != "Custom" && !ss.IsDeliveryProgram {
		// Non-delivery preset: grayed out
		drawSwarmButton(screen, 5, editorToggle3Y, toggleBtnW, toggleBtnH, "Delivery: \u2014", color.RGBA{60, 60, 70, 128})
	} else {
		delivLabel := "Delivery: OFF"
		delivColor := ColorSwarmBtnToggleOff
		if ss.DeliveryOn {
			delivLabel = "Delivery: ON"
			delivColor = color.RGBA{200, 120, 40, 255} // orange
		}
		drawSwarmButton(screen, 5, editorToggle3Y, toggleBtnW, toggleBtnH, delivLabel, delivColor)
	}

	// Separator 2
	vector.StrokeLine(screen, 5, float32(editorSep2Y), float32(editorPanelW-5), float32(editorSep2Y), 1, ColorSwarmEditorSep, false)

	// Stats panel
	drawSwarmStats(screen, ss)

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

	printColoredAt(screen, "--- Stats ---", 10, y, dimCol)
	y += lineH

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

	printColoredAt(screen, fmt.Sprintf("Chains: %d  MaxLen: %d", chains, maxChain), 10, y, col)
	y += lineH
	printColoredAt(screen, fmt.Sprintf("Avg Neighbors: %.1f", avgNeighbors), 10, y, col)
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

	printColoredAt(screen, fmt.Sprintf("Coverage: %.0f%%", coverage), 10, y, col)
	y += lineH

	// Trails status
	trailStatus := "OFF"
	if ss.ShowTrails {
		trailStatus = "ON"
	}
	wrapStatus := "BOUNCE"
	if ss.WrapMode {
		wrapStatus = "WRAP"
	}
	printColoredAt(screen, fmt.Sprintf("Trails:%s Walls:%s", trailStatus, wrapStatus), 10, y, dimCol)
	y += lineH

	// Delivery stats
	if ss.DeliveryOn {
		ds := &ss.DeliveryStats
		printColoredAt(screen, fmt.Sprintf("Deliveries: %d (%d ok)", ds.TotalDelivered, ds.CorrectDelivered), 10, y, col)
		y += lineH
		carrying := 0
		idle := 0
		for i := range ss.Bots {
			if ss.Bots[i].CarryingPkg >= 0 {
				carrying++
			} else {
				idle++
			}
		}
		printColoredAt(screen, fmt.Sprintf("Carrying:%d Idle:%d", carrying, idle), 10, y, col)
		y += lineH
		avgTime := 0
		if len(ds.DeliveryTimes) > 0 {
			sum := 0
			for _, t := range ds.DeliveryTimes {
				sum += t
			}
			avgTime = sum / len(ds.DeliveryTimes)
		}
		printColoredAt(screen, fmt.Sprintf("Avg delivery: %d ticks", avgTime), 10, y, dimCol)
	}
}

// DrawSwarmHUD draws the bottom help line for swarm mode.
func DrawSwarmHUD(screen *ebiten.Image, s *simulation.Simulation, fps float64) {
	sh := screen.Bounds().Dy()
	ss := s.SwarmState

	// FPS + speed at top of arena area
	info := fmt.Sprintf("FPS:%.0f Speed:%.1fx", fps, s.Speed)
	if s.Paused {
		info += " [PAUSED]"
	}
	ebitenutil.DebugPrintAt(screen, info, 360, 10)

	// Arena info line
	if ss != nil {
		arenaInfo := fmt.Sprintf("SwarmBots:%d | %s | Tick:%d",
			ss.BotCount, ss.ProgramName, ss.Tick)
		ebitenutil.DebugPrintAt(screen, arenaInfo, 420, 35)

		// Delivery HUD line
		if ss.DeliveryOn {
			ds := &ss.DeliveryStats
			avgTime := 0
			if len(ds.DeliveryTimes) > 0 {
				sum := 0
				for _, t := range ds.DeliveryTimes {
					sum += t
				}
				avgTime = sum / len(ds.DeliveryTimes)
			}
			dInfo := fmt.Sprintf("Deliveries:%d | Correct:%d | Wrong:%d | AvgTime:%d",
				ds.TotalDelivered, ds.CorrectDelivered, ds.WrongDelivered, avgTime)
			printColoredAt(screen, dInfo, 420, 15, color.RGBA{255, 200, 100, 255})
		}
	}

	// Help text at very bottom
	helpText := "F7:Swarm SPACE:Pause L:Light T:Trails C:Routes S:Sound +/-:Speed H:Hilfe"
	ebitenutil.DebugPrintAt(screen, helpText, 10, sh-15)

	// Scenario title overlay
	if s.ScenarioTimer > 0 {
		sw := screen.Bounds().Dx()
		drawScenarioTitle(screen, s.ScenarioTitle, sw, sh, s.ScenarioTimer)
	}
}

// --- Helper drawing functions ---

func drawSwarmButton(screen *ebiten.Image, x, y, w, h int, label string, bgCol color.RGBA) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), bgCol, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, color.RGBA{200, 200, 200, 100}, false)
	textX := x + (w-len(label)*charW)/2
	textY := y + (h-12)/2
	ebitenutil.DebugPrintAt(screen, label, textX, textY)
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

func drawSwarmDropdownOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	x := 5
	y := editorBarY + editorBarH
	w := 185
	itemH := 22

	for i, name := range ss.PresetNames {
		iy := y + i*itemH
		bgCol := color.RGBA{40, 50, 80, 240}
		if i == ss.DropdownHover {
			bgCol = ColorSwarmBtnHover
		}
		vector.DrawFilledRect(screen, float32(x), float32(iy), float32(w), float32(itemH), bgCol, false)
		vector.StrokeRect(screen, float32(x), float32(iy), float32(w), float32(itemH), 1, color.RGBA{100, 100, 120, 200}, false)
		ebitenutil.DebugPrintAt(screen, name, x+5, iy+5)
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
// "obstacles", "maze", "light", "walls", or ""
func SwarmEditorHitTest(mx, my int) string {
	if mx > editorPanelW {
		return ""
	}

	// Dropdown button
	if my >= editorBarY && my < editorBarY+editorBarH && mx >= 5 && mx < 190 {
		return "dropdown"
	}

	// Deploy button
	if my >= editorBarY && my < editorBarY+editorBarH && mx >= 195 && mx < 270 {
		return "deploy"
	}

	// Reset button
	if my >= editorBarY && my < editorBarY+editorBarH && mx >= 275 && mx < 340 {
		return "reset"
	}

	// Code editor area
	if my >= editorCodeY && my < editorCodeY+editorCodeH {
		return "editor"
	}

	// Bot count field
	if my >= editorBotCountY && my < editorBotCountY+18 && mx >= 5 && mx < 200 {
		return "botcount"
	}

	// Toggle row 1: Obstacles / Maze
	if my >= editorToggle1Y && my < editorToggle1Y+toggleBtnH {
		if mx >= 5 && mx < 5+toggleBtnW {
			return "obstacles"
		}
		if mx >= 175 && mx < 175+toggleBtnW {
			return "maze"
		}
	}

	// Toggle row 2: Light / Walls
	if my >= editorToggle2Y && my < editorToggle2Y+toggleBtnH {
		if mx >= 5 && mx < 5+toggleBtnW {
			return "light"
		}
		if mx >= 175 && mx < 175+toggleBtnW {
			return "walls"
		}
	}

	// Toggle row 3: Delivery
	if my >= editorToggle3Y && my < editorToggle3Y+toggleBtnH {
		if mx >= 5 && mx < 5+toggleBtnW {
			return "delivery"
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
	w := 185
	itemH := 22

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
