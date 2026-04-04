// Package render — Educational feature overlays:
// 1. Decision Trace (D key) — shows which SwarmScript rule fired and why
// 2. Did You Know tips — rotating educational facts
// 3. Concept Overlay (C key) — visual sensor/neighbor/heading explanations
// 4. Glossary (G key) — scrollable swarm robotics term reference
package render

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"swarmsim/domain/swarm"
	"swarmsim/locale"
)

// --- Color palette for educational overlays ---
var (
	colorEduBg       = ColorPanelBg
	colorEduBorder   = color.RGBA{80, 160, 255, 200}
	colorEduTitle    = color.RGBA{255, 220, 80, 255}
	colorEduText     = color.RGBA{200, 210, 230, 255}
	colorEduDim      = color.RGBA{120, 130, 150, 200}
	colorEduPass     = color.RGBA{80, 255, 100, 255}  // green checkmark
	colorEduFail     = color.RGBA{255, 80, 80, 255}   // red cross
	colorEduFired    = color.RGBA{40, 180, 60, 180}    // fired rule bg
	colorEduNotFired = color.RGBA{15, 18, 30, 200}     // non-fired rule bg

	colorTipBg     = color.RGBA{15, 20, 40, 200}
	colorTipBorder = color.RGBA{255, 200, 60, 150}
	colorTipText   = color.RGBA{255, 240, 200, 230}
	colorTipIcon   = color.RGBA{255, 200, 50, 255}

	colorGlossBg     = ColorPanelBg
	colorGlossBorder = color.RGBA{100, 180, 255, 200}
	colorGlossTerm   = color.RGBA{255, 220, 80, 255}
	colorGlossDef    = color.RGBA{190, 200, 220, 230}
	colorGlossHint   = color.RGBA{100, 110, 130, 180}

	colorConceptSensor   = color.RGBA{0, 150, 255, 50}
	colorConceptNeighbor = color.RGBA{255, 255, 255, 60}
	colorConceptHeading  = color.RGBA{0, 255, 100, 180}
	colorConceptNearest  = color.RGBA{0, 255, 255, 150}
	colorConceptComm     = color.RGBA{255, 200, 50, 40}
)

// =========================================================================
// 1. DECISION TRACE (D key)
// =========================================================================

// DrawDecisionTrace renders the decision trace panel when a bot is selected.
func DrawDecisionTrace(screen *ebiten.Image, ss *swarm.SwarmState) {
	if !ss.ShowDecisionTrace || ss.SelectedBot < 0 || len(ss.DecisionTrace) == 0 {
		return
	}

	sw := screen.Bounds().Dx()

	// Panel dimensions
	panelW := 420
	panelH := 20 + len(ss.DecisionTrace)*42
	if panelH > 500 {
		panelH = 500
	}
	panelX := sw - panelW - 10
	panelY := 120

	// Background
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), colorEduBg, false)
	vector.StrokeRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), 2, colorEduBorder, false)

	// Title
	title := fmt.Sprintf("%s #%d — %s", locale.T("edu.decision.title"), ss.SelectedBot, locale.T("edu.decision.subtitle"))
	printColoredAt(screen, title, panelX+8, panelY+4, colorEduTitle)

	y := panelY + 22
	maxY := panelY + panelH - 10

	firedIdx := -1
	// Find first matched rule
	for i, step := range ss.DecisionTrace {
		if step.Matched {
			firedIdx = i
			break
		}
	}

	for ri, step := range ss.DecisionTrace {
		if y > maxY {
			break
		}

		// Rule background (highlight fired rule)
		ruleBg := colorEduNotFired
		if step.Matched && ri == firedIdx {
			ruleBg = colorEduFired
		}
		ruleH := 14 + len(step.Conditions)*14
		if ruleH > 40 {
			ruleH = 40
		}
		vector.DrawFilledRect(screen, float32(panelX+4), float32(y-2), float32(panelW-8), float32(ruleH), ruleBg, false)

		// Rule number and match indicator
		indicator := "X"
		indicatorColor := colorEduFail
		if step.Matched {
			indicator = ">"
			indicatorColor = colorEduPass
		}
		ruleLabel := fmt.Sprintf("%s R%d: %s", indicator, step.RuleIdx+1, step.ActionName)
		printColoredAt(screen, ruleLabel, panelX+8, y, indicatorColor)

		// Condition details (compact)
		cx := panelX + 28
		cy := y + 14
		for _, c := range step.Conditions {
			if cy > maxY {
				break
			}
			passStr := "X"
			passColor := colorEduFail
			if c.Passed {
				passStr = "+"
				passColor = colorEduPass
			}
			var condText string
			if c.SensorName == "true" {
				condText = fmt.Sprintf("[%s] true", passStr)
			} else {
				condText = fmt.Sprintf("[%s] %s: %s %s %s", passStr, c.SensorName, c.ActualValue, c.Operator, c.Threshold)
			}
			printColoredAt(screen, condText, cx, cy, passColor)
			cy += 14
		}

		y += ruleH + 4
	}

	// ESC hint at bottom
	escHint := locale.T("overlay.esc_close")
	printColoredAt(screen, escHint, panelX+panelW/2-runeLen(escHint)*charW/2, panelY+panelH-14, color.RGBA{120, 130, 150, 180})
}

// =========================================================================
// 2. DID YOU KNOW TIPS
// =========================================================================

// TipCount is the number of educational tips available.
const TipCount = 23

// UpdateDidYouKnow advances the tip timer and rotates tips.
func UpdateDidYouKnow(ss *swarm.SwarmState) {
	if ss.DidYouKnowTimer > 0 {
		ss.DidYouKnowTimer--
	}
	// Show new tip every 2000 ticks
	if ss.Tick > 0 && ss.Tick%2000 == 0 {
		ss.DidYouKnowIdx = (ss.DidYouKnowIdx + 1) % TipCount
		ss.DidYouKnowTimer = 300 // show for 5 seconds at 60fps
	}
}

// DrawDidYouKnow renders the educational tip banner at the bottom.
func DrawDidYouKnow(screen *ebiten.Image, ss *swarm.SwarmState) {
	if ss.DidYouKnowTimer <= 0 {
		return
	}

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	tipKey := fmt.Sprintf("tip.%d", ss.DidYouKnowIdx+1)
	tipText := locale.T(tipKey)

	// Fade in/out
	alpha := uint8(200)
	if ss.DidYouKnowTimer < 30 {
		alpha = uint8(200 * ss.DidYouKnowTimer / 30)
	}
	if ss.DidYouKnowTimer > 270 {
		fade := 300 - ss.DidYouKnowTimer
		alpha = uint8(200 * fade / 30)
	}

	bannerW := runeLen(tipText)*charW + 50
	if bannerW > sw-100 {
		bannerW = sw - 100
	}
	bannerH := 28
	bannerX := (sw - bannerW) / 2
	bannerY := sh - bannerH - 8

	bg := color.RGBA{colorTipBg.R, colorTipBg.G, colorTipBg.B, alpha}
	border := color.RGBA{colorTipBorder.R, colorTipBorder.G, colorTipBorder.B, alpha / 2}
	txt := color.RGBA{colorTipText.R, colorTipText.G, colorTipText.B, alpha}
	icon := color.RGBA{colorTipIcon.R, colorTipIcon.G, colorTipIcon.B, alpha}

	vector.DrawFilledRect(screen, float32(bannerX), float32(bannerY), float32(bannerW), float32(bannerH), bg, false)
	vector.StrokeRect(screen, float32(bannerX), float32(bannerY), float32(bannerW), float32(bannerH), 1, border, false)

	// Lightbulb icon (asterisk as stand-in)
	printColoredAt(screen, "*", bannerX+6, bannerY+6, icon)

	// Tip text
	printColoredAt(screen, tipText, bannerX+20, bannerY+6, txt)
}

// =========================================================================
// 3. CONCEPT OVERLAY (C key) — visual explanations on the arena
// =========================================================================

// DrawConceptOverlay draws educational visual overlays on the arena image
// for the selected bot: sensor radius, neighbor lines, heading arrow, etc.
func DrawConceptOverlay(target *ebiten.Image, ss *swarm.SwarmState) {
	if !ss.ShowConceptOverlay || ss.SelectedBot < 0 || ss.SelectedBot >= len(ss.Bots) {
		return
	}

	bot := &ss.Bots[ss.SelectedBot]
	bx := float32(bot.X)
	by := float32(bot.Y)

	// 1. Sensor radius circle (120px, semi-transparent blue)
	sensorRange := float32(swarm.SwarmSensorRange)
	vector.StrokeCircle(target, bx, by, sensorRange, 1.5, colorConceptSensor, false)
	// Label
	printColoredAt(target, locale.T("concept.sensor"), int(bx+sensorRange+4), int(by-8), colorEduDim)

	// 2. Communication range circle (yellow dashed approximation)
	commRange := float32(swarm.SwarmCommRange)
	vector.StrokeCircle(target, bx, by, commRange, 1, colorConceptComm, false)
	printColoredAt(target, locale.T("concept.comm"), int(bx+commRange+4), int(by+8), colorEduDim)

	// 3. Neighbor connection lines (thin white lines to all neighbors within sensor range)
	sensorRangeSq := float64(sensorRange * sensorRange)
	for i := range ss.Bots {
		if i == ss.SelectedBot {
			continue
		}
		other := &ss.Bots[i]
		dx := other.X - bot.X
		dy := other.Y - bot.Y
		distSq := dx*dx + dy*dy
		if distSq < sensorRangeSq {
			vector.StrokeLine(target, bx, by, float32(other.X), float32(other.Y), 0.5, colorConceptNeighbor, false)
		}
	}

	// 4. Nearest bot highlight (thick cyan line)
	if bot.NearestIdx >= 0 && bot.NearestIdx < len(ss.Bots) {
		near := &ss.Bots[bot.NearestIdx]
		vector.StrokeLine(target, bx, by, float32(near.X), float32(near.Y), 2, colorConceptNearest, false)
		// Distance label
		dist := math.Sqrt(math.Pow(near.X-bot.X, 2) + math.Pow(near.Y-bot.Y, 2))
		midX := (bx + float32(near.X)) / 2
		midY := (by + float32(near.Y)) / 2
		printColoredAt(target, fmt.Sprintf("%.0f", dist), int(midX)+4, int(midY)-8, colorConceptNearest)
	}

	// 5. Heading arrow (long green arrow in movement direction)
	arrowLen := float32(40)
	endX := bx + arrowLen*float32(math.Cos(bot.Angle))
	endY := by + arrowLen*float32(math.Sin(bot.Angle))
	vector.StrokeLine(target, bx, by, endX, endY, 2, colorConceptHeading, false)
	// Arrowhead
	headAngle1 := bot.Angle + math.Pi*0.85
	headAngle2 := bot.Angle - math.Pi*0.85
	headLen := float32(10)
	vector.StrokeLine(target, endX, endY,
		endX+headLen*float32(math.Cos(headAngle1)),
		endY+headLen*float32(math.Sin(headAngle1)),
		2, colorConceptHeading, false)
	vector.StrokeLine(target, endX, endY,
		endX+headLen*float32(math.Cos(headAngle2)),
		endY+headLen*float32(math.Sin(headAngle2)),
		2, colorConceptHeading, false)
	// Label
	printColoredAt(target, locale.T("concept.heading"), int(endX)+6, int(endY)-4, colorConceptHeading)
}

// =========================================================================
// 4. GLOSSARY (G key) — scrollable term reference
// =========================================================================

// GlossaryTermCount is the number of glossary terms.
const GlossaryTermCount = 15

// DrawGlossary renders the glossary overlay panel.
func DrawGlossary(screen *ebiten.Image, ss *swarm.SwarmState) {
	if !ss.ShowGlossary {
		return
	}

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	panelW := 460
	panelH := sh - 80
	panelX := (sw - panelW) / 2
	panelY := 40

	// Background
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), colorGlossBg, false)
	vector.StrokeRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), 2, colorGlossBorder, false)

	// Title
	title := locale.T("glossary.title")
	titleX := panelX + (panelW-runeLen(title)*charW)/2
	printColoredAt(screen, title, titleX, panelY+8, colorEduTitle)

	// Scrollable content
	y := panelY + 30 - ss.GlossaryScroll
	maxY := panelY + panelH - 20
	minY := panelY + 28

	for i := 1; i <= GlossaryTermCount; i++ {
		termKey := fmt.Sprintf("glossary.term.%d", i)
		defKey := fmt.Sprintf("glossary.def.%d", i)
		term := locale.T(termKey)
		def := locale.T(defKey)

		if y >= minY && y < maxY {
			printColoredAt(screen, term, panelX+12, y, colorGlossTerm)
		}
		if y+lineH >= minY && y+lineH < maxY {
			// Split definition on | for multi-line
			lines := splitLines(def)
			for li, line := range lines {
				ly := y + lineH + li*lineH
				if ly >= minY && ly < maxY {
					printColoredAt(screen, "  "+line, panelX+12, ly, colorGlossDef)
				}
			}
			y += lineH * (1 + len(lines))
		} else {
			y += lineH * 3 // approximate skip
		}
		y += 6 // gap between terms
	}

	// Scroll down indicator: show if content extends below visible area
	if y > maxY {
		arrowX := panelX + panelW/2 - 15
		arrowY := panelY + panelH - 42
		printColoredAt(screen, "v v v", arrowX, arrowY, color.RGBA{150, 160, 180, 150})
	}

	// Scroll hint
	hint := locale.T("glossary.hint")
	printColoredAt(screen, hint, panelX+10, panelY+panelH-28, colorGlossHint)

	// ESC hint at bottom
	escHint := locale.T("overlay.esc_close")
	printColoredAt(screen, escHint, panelX+panelW/2-runeLen(escHint)*charW/2, panelY+panelH-14, color.RGBA{120, 130, 150, 180})
}
