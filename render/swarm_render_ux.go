// Package render — UX improvement overlays:
// 1. Parameter Tweaker (Shift+P) — live +/- controls for $A-$Z params
// 2. Performance Monitor (F12) — FPS, tick time, bot count, spatial hash stats
package render

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"swarmsim/domain/swarm"
	"swarmsim/locale"
)

// --- Color palette for UX overlays ---
var (
	colorTweakerBg     = ColorPanelBg
	colorTweakerBorder = color.RGBA{120, 200, 80, 200}
	colorTweakerTitle  = color.RGBA{120, 255, 80, 255}
	colorTweakerLabel  = color.RGBA{200, 210, 230, 255}
	colorTweakerValue  = color.RGBA{255, 255, 255, 255}
	colorTweakerBtn    = color.RGBA{60, 120, 60, 255}
	colorTweakerBtnDec = color.RGBA{120, 60, 60, 255}

	colorPerfBg     = ColorPanelBg
	colorPerfBorder = color.RGBA{200, 200, 60, 180}
	colorPerfTitle  = color.RGBA{255, 220, 80, 255}
	colorPerfText   = color.RGBA{200, 210, 230, 255}
	colorPerfGood   = color.RGBA{80, 255, 80, 255}
	colorPerfWarn   = color.RGBA{255, 200, 60, 255}
	colorPerfBad    = color.RGBA{255, 80, 80, 255}
)

// =========================================================================
// 1. PARAMETER TWEAKER (Shift+P)
// =========================================================================

// paramTweaker layout constants
const (
	tweakerX       = 360
	tweakerY       = 60
	tweakerW       = 210
	tweakerRowH    = 22
	tweakerBtnW    = 22
	tweakerBtnH    = 18
	tweakerPadding = 8
)

// DrawParamTweaker renders the parameter tweaker overlay panel.
func DrawParamTweaker(screen *ebiten.Image, ss *swarm.SwarmState) {
	if !ss.ShowParamTweaker || ss.Program == nil {
		return
	}

	// Find which parameters are used in the current program
	usedParams := findUsedParamIndices(ss)
	if len(usedParams) == 0 {
		// Show a hint that no params are used
		panelH := 60
		vector.DrawFilledRect(screen, float32(tweakerX), float32(tweakerY), float32(tweakerW), float32(panelH), colorTweakerBg, false)
		vector.StrokeRect(screen, float32(tweakerX), float32(tweakerY), float32(tweakerW), float32(panelH), 2, colorTweakerBorder, false)
		printColoredAt(screen, locale.T("tweaker.title"), tweakerX+tweakerPadding, tweakerY+tweakerPadding, colorTweakerTitle)
		printColoredAt(screen, locale.T("tweaker.no_params"), tweakerX+tweakerPadding, tweakerY+28, colorTweakerLabel)
		return
	}

	panelH := len(usedParams)*tweakerRowH + 44
	// Background
	vector.DrawFilledRect(screen, float32(tweakerX), float32(tweakerY), float32(tweakerW), float32(panelH), colorTweakerBg, false)
	vector.StrokeRect(screen, float32(tweakerX), float32(tweakerY), float32(tweakerW), float32(panelH), 2, colorTweakerBorder, false)

	// Title
	printColoredAt(screen, locale.T("tweaker.title"), tweakerX+tweakerPadding, tweakerY+tweakerPadding, colorTweakerTitle)
	printColoredAt(screen, locale.T("tweaker.hint"), tweakerX+tweakerW-80, tweakerY+tweakerPadding, color.RGBA{80, 100, 80, 180})

	// Parameter rows
	y := tweakerY + 30
	for _, paramIdx := range usedParams {
		paramName := fmt.Sprintf("$%c", 'A'+paramIdx)
		value := float64(0)
		if len(ss.Bots) > 0 && paramIdx < len(ss.Bots[0].ParamValues) {
			value = ss.Bots[0].ParamValues[paramIdx]
		}

		// Parameter name
		printColoredAt(screen, paramName, tweakerX+tweakerPadding, y+3, colorTweakerLabel)

		// [-] button
		btnX := tweakerX + 40
		drawSwarmButton(screen, btnX, y, tweakerBtnW, tweakerBtnH, "-", colorTweakerBtnDec)

		// Value display
		valStr := fmt.Sprintf("%.0f", value)
		valX := btnX + tweakerBtnW + 4
		valW := 60
		vector.DrawFilledRect(screen, float32(valX), float32(y), float32(valW), float32(tweakerBtnH), color.RGBA{20, 25, 40, 255}, false)
		// Center the value text in the value area
		textX := valX + (valW-len(valStr)*charW)/2
		printColoredAt(screen, valStr, textX, y+3, colorTweakerValue)

		// [+] button
		plusX := valX + valW + 4
		drawSwarmButton(screen, plusX, y, tweakerBtnW, tweakerBtnH, "+", colorTweakerBtn)

		y += tweakerRowH
	}

	// ESC hint at bottom of panel
	escHint := locale.T("overlay.esc_close")
	printColoredAt(screen, escHint, tweakerX+tweakerW/2-runeLen(escHint)*charW/2, y+4, color.RGBA{120, 130, 150, 180})
}

// findUsedParamIndices returns sorted indices of parameters used in the current program.
func findUsedParamIndices(ss *swarm.SwarmState) []int {
	var indices []int
	for i, used := range ss.UsedParams {
		if used {
			indices = append(indices, i)
		}
	}
	return indices
}

// ParamTweakerHitTest checks if a click is on a +/- button in the tweaker.
// Returns "param:X:+" or "param:X:-" where X is the parameter letter, or "".
func ParamTweakerHitTest(mx, my int, ss *swarm.SwarmState) string {
	if !ss.ShowParamTweaker || ss.Program == nil {
		return ""
	}

	usedParams := findUsedParamIndices(ss)
	if len(usedParams) == 0 {
		return ""
	}

	y := tweakerY + 30
	for _, paramIdx := range usedParams {
		// [-] button hit area
		btnX := tweakerX + 40
		if mx >= btnX && mx < btnX+tweakerBtnW && my >= y && my < y+tweakerBtnH {
			return fmt.Sprintf("param:%c:-", 'A'+paramIdx)
		}

		// [+] button hit area
		valX := btnX + tweakerBtnW + 4
		valW := 60
		plusX := valX + valW + 4
		if mx >= plusX && mx < plusX+tweakerBtnW && my >= y && my < y+tweakerBtnH {
			return fmt.Sprintf("param:%c:+", 'A'+paramIdx)
		}

		y += tweakerRowH
	}
	return ""
}

// ApplyParamTweak modifies a parameter value for ALL bots.
// paramIdx is 0-25 (A-Z), delta is the change amount.
func ApplyParamTweak(ss *swarm.SwarmState, paramIdx int, delta float64) {
	if paramIdx < 0 || paramIdx >= 26 {
		return
	}
	for i := range ss.Bots {
		if paramIdx < len(ss.Bots[i].ParamValues) {
			ss.Bots[i].ParamValues[paramIdx] += delta
		}
	}
}

// =========================================================================
// 2. PERFORMANCE MONITOR (F12)
// =========================================================================

// DrawPerfMonitor renders the performance monitor overlay in the top-right corner.
func DrawPerfMonitor(screen *ebiten.Image, ss *swarm.SwarmState) {
	if !ss.ShowPerfMonitor {
		return
	}

	sw := screen.Bounds().Dx()

	panelW := 200
	panelH := 110
	panelX := sw - panelW - 10
	panelY := 10

	// Background
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), colorPerfBg, false)
	vector.StrokeRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), 1, colorPerfBorder, false)

	lx := panelX + 6
	ly := panelY + 4

	// Title
	printColoredAt(screen, locale.T("perf.title"), lx, ly, colorPerfTitle)
	ly += lineH

	// FPS
	fps := ebiten.ActualFPS()
	fpsCol := colorPerfGood
	if fps < 50 {
		fpsCol = colorPerfWarn
	}
	if fps < 30 {
		fpsCol = colorPerfBad
	}
	printColoredAt(screen, fmt.Sprintf("FPS: %.0f", fps), lx, ly, fpsCol)

	// TPS
	tps := ebiten.ActualTPS()
	tpsCol := colorPerfGood
	if tps < 50 {
		tpsCol = colorPerfWarn
	}
	if tps < 30 {
		tpsCol = colorPerfBad
	}
	printColoredAt(screen, fmt.Sprintf("TPS: %.0f", tps), lx+80, ly, tpsCol)
	ly += lineH

	// Tick time
	tickMs := ss.LastTickDuration * 1000.0
	tickCol := colorPerfGood
	if tickMs > 8.0 {
		tickCol = colorPerfWarn
	}
	if tickMs > 16.0 {
		tickCol = colorPerfBad
	}
	printColoredAt(screen, fmt.Sprintf(locale.T("perf.tick_time"), tickMs), lx, ly, tickCol)
	ly += lineH

	// Bot count + active tasks
	printColoredAt(screen, fmt.Sprintf(locale.T("perf.bots"), len(ss.Bots)), lx, ly, colorPerfText)

	// Count active tasks (carrying bots)
	activeTasks := 0
	for i := range ss.Bots {
		if ss.Bots[i].CarryingPkg >= 0 {
			activeTasks++
		}
	}
	printColoredAt(screen, fmt.Sprintf(locale.T("perf.tasks"), activeTasks), lx+90, ly, colorPerfText)
	ly += lineH

	// Spatial hash stats
	if ss.Hash != nil {
		cells, avgPer := ss.Hash.Stats()
		printColoredAt(screen, fmt.Sprintf(locale.T("perf.hash"), cells, avgPer), lx, ly, colorPerfText)
	} else {
		printColoredAt(screen, locale.T("perf.no_hash"), lx, ly, color.RGBA{100, 100, 120, 200})
	}
	ly += lineH

	// Memory estimate (bots * approx struct size)
	memKB := float64(len(ss.Bots)) * 2.5 // ~2.5 KB per bot struct (estimate)
	if memKB > 1024 {
		printColoredAt(screen, fmt.Sprintf(locale.T("perf.mem_mb"), memKB/1024), lx, ly, colorPerfText)
	} else {
		printColoredAt(screen, fmt.Sprintf(locale.T("perf.mem_kb"), memKB), lx, ly, colorPerfText)
	}

	// ESC hint at bottom
	escHint := locale.T("overlay.esc_close")
	printColoredAt(screen, escHint, panelX+panelW/2-runeLen(escHint)*charW/2, panelY+panelH-14, color.RGBA{120, 130, 150, 180})
}

// =========================================================================
// 3. SHORTCUT REFERENCE CARD (? key)
// =========================================================================

// DrawShortcutCard renders a compact keyboard shortcut reference card.
func DrawShortcutCard(screen *ebiten.Image, ss *swarm.SwarmState) {
	if ss.ActiveOverlay != "shortcuts" {
		return
	}

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()
	w, h := 500, 400
	x := (sw - w) / 2
	y := (sh - h) / 2

	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), ColorPanelBg, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, ColorPanelBorder, false)

	lx := x + 15
	ly := y + 15

	printColoredAt(screen, locale.T("shortcuts.title"), lx, ly, ColorPanelHeader)
	ly += lineH + 4

	// Two columns
	shortcuts := []struct{ key, desc string }{
		{"F1-F5", locale.T("shortcuts.modes")},
		{"H", locale.T("shortcuts.help")},
		{"Space", locale.T("shortcuts.pause")},
		{"1-5", locale.T("shortcuts.speed")},
		{"K", locale.T("shortcuts.math")},
		{"D", locale.T("shortcuts.decision")},
		{"C", locale.T("shortcuts.concept")},
		{"G", locale.T("shortcuts.glossary")},
		{"I", locale.T("shortcuts.issues")},
		{"Shift+L", locale.T("shortcuts.lessons")},
		{"Shift+P", locale.T("shortcuts.params")},
		{"Shift+I", locale.T("shortcuts.collective")},
		{"Ctrl+S", locale.T("shortcuts.save")},
		{"F12", locale.T("shortcuts.perf")},
		{"0", locale.T("shortcuts.zoomfit")},
		{"?", locale.T("shortcuts.this")},
		{"ESC", locale.T("shortcuts.close")},
	}

	mid := len(shortcuts)/2 + 1
	for i, s := range shortcuts {
		col := 0
		row := i
		if i >= mid {
			col = 1
			row = i - mid
		}
		sx := lx + col*240
		sy := ly + row*(lineH+2)
		printColoredAt(screen, s.key, sx, sy, color.RGBA{136, 204, 255, 255})
		printColoredAt(screen, s.desc, sx+80, sy, color.RGBA{180, 185, 200, 220})
	}

	// ESC hint
	escHint := locale.T("overlay.esc_close")
	printColoredAt(screen, escHint, x+w/2-runeLen(escHint)*charW/2, y+h-16, color.RGBA{120, 130, 150, 180})
}
