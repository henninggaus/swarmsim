// Package render — Learning system overlay renderer.
// Draws lesson text boxes, progress indicators, star ratings,
// challenge progress bars, emergence popups, and the lesson menu.
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

// Learning system color palette
var (
	colorLearnBg       = color.RGBA{10, 15, 30, 230}
	colorLearnBorder   = color.RGBA{100, 200, 120, 255}
	colorLearnText     = color.RGBA{220, 230, 210, 255}
	colorLearnDim      = color.RGBA{140, 150, 130, 255}
	colorLearnBtn      = color.RGBA{40, 140, 80, 255}
	colorLearnBtnH     = color.RGBA{60, 180, 100, 255}
	colorLearnStep     = color.RGBA{100, 200, 120, 200}
	colorLearnTitle    = color.RGBA{255, 220, 100, 255}
	colorLearnStar     = color.RGBA{255, 200, 50, 255}
	colorLearnStarDim  = color.RGBA{80, 80, 60, 200}
	colorLearnLocked   = color.RGBA{100, 100, 100, 200}
	colorLearnMenuBg   = color.RGBA{5, 8, 20, 240}
	colorLearnMenuCard = color.RGBA{20, 30, 50, 240}
	colorLearnChall    = color.RGBA{255, 150, 50, 255}
	colorLearnBarBg    = color.RGBA{40, 40, 40, 200}
	colorLearnBarFill  = color.RGBA{100, 200, 120, 255}
	colorEmergeBg      = color.RGBA{15, 20, 40, 220}
	colorEmergeBorder  = color.RGBA{255, 200, 80, 255}
	colorEmergeTitle   = color.RGBA{255, 220, 100, 255}
	colorEmergeText    = color.RGBA{200, 210, 230, 255}
)

// DrawLessonOverlay renders the active lesson text box at the bottom center.
func DrawLessonOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	ls := ss.Learning
	if ls == nil || !ls.Active {
		return
	}
	lessons := swarm.GetAllLessons()
	if int(ls.CurrentLesson) >= len(lessons) {
		return
	}
	lesson := lessons[ls.CurrentLesson]

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// If challenge is active, draw challenge overlay instead
	if ls.ChallengeActive {
		drawChallengeOverlay(screen, ls, lesson, sw, sh)
		return
	}

	if ls.CurrentStep >= len(lesson.Steps) {
		return
	}
	step := lesson.Steps[ls.CurrentStep]

	// Text box at bottom center
	boxW := 680
	boxH := 100
	boxX := (sw - boxW) / 2
	boxY := sh - boxH - 45

	// Box background with green border
	vector.DrawFilledRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), colorLearnBg, false)
	vector.StrokeRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), 2, colorLearnBorder, false)

	// Progress indicator
	progress := fmt.Sprintf("%s %d/%d  %s %d/%d",
		locale.T("learn.lesson"), int(ls.CurrentLesson)+1, int(swarm.LessonCount),
		locale.T("learn.step"), ls.CurrentStep+1, len(lesson.Steps))
	printColoredAt(screen, progress, boxX+10, boxY+6, colorLearnStep)

	// Lesson title
	title := locale.T(lesson.TitleKey)
	printColoredAt(screen, title, boxX+boxW-runeLen(title)*charW-15, boxY+6, colorLearnTitle)

	// Step text (up to 3 lines)
	txt := locale.T(step.TextKey)
	// Split text at | for multi-line
	lines := splitLines(txt)
	for i, line := range lines {
		if i >= 3 {
			break
		}
		printColoredAt(screen, line, boxX+15, boxY+24+i*lineH, colorLearnText)
	}

	// Highlight indicator
	if step.Highlight != "" {
		hintText := fmt.Sprintf("[%s: %s]", locale.T("learn.watch"), step.Highlight)
		printColoredAt(screen, hintText, boxX+15, boxY+boxH-20, colorLearnDim)
	}

	// Auto-advance timer indicator
	if step.WaitTicks > 0 && ls.StepTimer > 0 {
		pct := float64(ls.StepTimer) / float64(step.WaitTicks)
		barW := 100
		barX := boxX + boxW - barW - 90
		barY := boxY + boxH - 18
		vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(barW), 8, colorLearnBarBg, false)
		vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(float64(barW)*pct), 8, colorLearnBarFill, false)
	}

	// Next button (only when not waiting for auto-advance)
	btnY := boxY + boxH - 26
	if step.WaitTicks == 0 || ls.StepTimer >= step.WaitTicks {
		btnW := 80
		btnX := boxX + boxW - btnW - 10
		mx, my := ebiten.CursorPosition()
		hovered := mx >= btnX && mx < btnX+btnW && my >= btnY && my < btnY+22
		btnColor := colorLearnBtn
		if hovered {
			btnColor = colorLearnBtnH
		}
		vector.DrawFilledRect(screen, float32(btnX), float32(btnY), float32(btnW), 22, btnColor, false)
		printColoredAt(screen, locale.T("learn.next"), btnX+10, btnY+4, colorLearnText)
	} else {
		hint := locale.T("learn.wait")
		printColoredAt(screen, hint, boxX+boxW-runeLen(hint)*charW-15, btnY+4, colorLearnDim)
	}

	// Exit button
	exitW := 80
	exitX := boxX + 10
	vector.DrawFilledRect(screen, float32(exitX), float32(btnY), float32(exitW), 22, color.RGBA{80, 30, 30, 200}, false)
	printColoredAt(screen, locale.T("learn.exit"), exitX+8, btnY+4, colorLearnDim)
}

// drawChallengeOverlay renders the challenge progress bar and description.
func drawChallengeOverlay(screen *ebiten.Image, ls *swarm.LearningState, lesson swarm.Lesson, sw, sh int) {
	ch := lesson.Challenge
	if ch == nil {
		return
	}

	boxW := 500
	boxH := 80
	boxX := (sw - boxW) / 2
	boxY := sh - boxH - 45

	vector.DrawFilledRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), colorLearnBg, false)
	vector.StrokeRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), 2, colorLearnChall, false)

	// Challenge title
	printColoredAt(screen, locale.T("learn.challenge"), boxX+10, boxY+6, colorLearnChall)

	// Challenge description
	desc := locale.T(ch.DescKey)
	printColoredAt(screen, desc, boxX+15, boxY+24, colorLearnText)

	// Progress bar
	pct := ls.ChallengeValue / ch.ThresholdGold
	if pct > 1 {
		pct = 1
	}
	barW := boxW - 30
	barX := boxX + 15
	barY := boxY + 48
	vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(barW), 10, colorLearnBarBg, false)
	vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(float64(barW)*pct), 10, colorLearnBarFill, false)

	// Threshold markers
	for _, th := range []float64{ch.ThresholdBronze, ch.ThresholdSilver, ch.ThresholdGold} {
		markerPct := th / ch.ThresholdGold
		mx := float32(barX) + float32(float64(barW)*markerPct)
		vector.StrokeLine(screen, mx, float32(barY-2), mx, float32(barY+12), 1, colorLearnStar, false)
	}

	// Timer
	secs := ls.ChallengeTicks / 60
	timerText := fmt.Sprintf("%ds", secs)
	printColoredAt(screen, timerText, boxX+boxW-50, boxY+6, colorLearnDim)

	// Exit button
	btnY := boxY + boxH - 22
	exitW := 80
	exitX := boxX + 10
	vector.DrawFilledRect(screen, float32(exitX), float32(btnY), float32(exitW), 18, color.RGBA{80, 30, 30, 200}, false)
	printColoredAt(screen, locale.T("learn.exit"), exitX+8, btnY+2, colorLearnDim)
}

// DrawEmergencePopup renders a floating explanation popup near the action point.
func DrawEmergencePopup(screen *ebiten.Image, ss *swarm.SwarmState) {
	if ss.EmergencePopup == nil || ss.EmergencePopup.Timer <= 0 {
		return
	}
	ep := ss.EmergencePopup

	// Convert world coords to screen position (approximate)
	// Popup is drawn at a fixed screen position with a connection line
	sw := screen.Bounds().Dx()

	// Fade in/out
	alpha := uint8(220)
	if ep.Timer < 30 {
		alpha = uint8(220 * ep.Timer / 30)
	}

	boxW := 380
	boxH := 60
	boxX := sw - boxW - 20
	boxY := 60

	bg := color.RGBA{colorEmergeBg.R, colorEmergeBg.G, colorEmergeBg.B, alpha}
	border := color.RGBA{colorEmergeBorder.R, colorEmergeBorder.G, colorEmergeBorder.B, alpha}

	vector.DrawFilledRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), bg, false)
	vector.StrokeRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), 2, border, false)

	// Lightbulb icon (text)
	titleColor := color.RGBA{colorEmergeTitle.R, colorEmergeTitle.G, colorEmergeTitle.B, alpha}
	textColor := color.RGBA{colorEmergeText.R, colorEmergeText.G, colorEmergeText.B, alpha}

	title := locale.T(ep.TitleKey)
	printColoredAt(screen, "* "+title, boxX+10, boxY+6, titleColor)

	txt := locale.T(ep.TextKey)
	lines := splitLines(txt)
	for i, line := range lines {
		if i >= 2 {
			break
		}
		printColoredAt(screen, line, boxX+15, boxY+24+i*lineH, textColor)
	}
}

// DrawLessonMenu renders the full-screen lesson selection menu.
func DrawLessonMenu(screen *ebiten.Image, ss *swarm.SwarmState) {
	ls := ss.Learning
	if ls == nil || !ls.ShowMenu {
		return
	}

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()
	lessons := swarm.GetAllLessons()

	// Full-screen overlay
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), colorLearnMenuBg, false)

	// Title
	title := locale.T("learn.menu.title")
	titleX := (sw - runeLen(title)*charW) / 2
	printColoredAt(screen, title, titleX, 20, colorLearnTitle)

	// Star count
	starText := fmt.Sprintf("%s: %d / %d", locale.T("learn.stars"), ls.TotalStars, int(swarm.LessonCount)*3)
	printColoredAt(screen, starText, sw-runeLen(starText)*charW-20, 22, colorLearnStar)

	// Level labels
	levelNames := []string{
		locale.T("learn.level.beginner"),
		locale.T("learn.level.intermediate"),
		locale.T("learn.level.advanced"),
	}

	// Draw 3 rows of 4 cards each
	cardW := 160
	cardH := 120
	gapX := 15
	gapY := 15
	startY := 55

	for level := 0; level < 3; level++ {
		rowY := startY + level*(cardH+gapY+20)

		// Level header
		printColoredAt(screen, levelNames[level], 30, rowY, colorLearnStep)
		rowY += 18

		for col := 0; col < 4; col++ {
			idx := level*4 + col
			if idx >= len(lessons) {
				break
			}
			lesson := lessons[idx]

			cardX := 30 + col*(cardW+gapX)

			// Card background
			cardColor := colorLearnMenuCard
			mx, my := ebiten.CursorPosition()
			hovered := mx >= cardX && mx < cardX+cardW && my >= rowY && my < rowY+cardH
			if hovered {
				cardColor = color.RGBA{30, 45, 70, 240}
			}

			// Check if locked (need previous lesson bronze+)
			locked := false
			if idx > 0 && ls.Completed[idx-1] == 0 {
				locked = true
				cardColor = color.RGBA{15, 15, 20, 200}
			}

			vector.DrawFilledRect(screen, float32(cardX), float32(rowY), float32(cardW), float32(cardH), cardColor, false)

			// Border color based on completion
			borderCol := colorLearnBorder
			if locked {
				borderCol = colorLearnLocked
			}
			vector.StrokeRect(screen, float32(cardX), float32(rowY), float32(cardW), float32(cardH), 1, borderCol, false)

			// Lesson number
			numText := fmt.Sprintf("#%d", idx+1)
			printColoredAt(screen, numText, cardX+5, rowY+5, colorLearnDim)

			// Title
			titleText := locale.T(lesson.TitleKey)
			tc := colorLearnText
			if locked {
				tc = colorLearnLocked
			}
			printColoredAt(screen, titleText, cardX+5, rowY+20, tc)

			// Description (truncated)
			descText := locale.T(lesson.DescKey)
			if runeLen(descText) > 24 {
				descText = string([]rune(descText)[:22]) + ".."
			}
			printColoredAt(screen, descText, cardX+5, rowY+36, colorLearnDim)

			// Stars
			rating := ls.Completed[idx]
			for s := 0; s < 3; s++ {
				starX := cardX + 5 + s*14
				starY := rowY + cardH - 22
				sc := colorLearnStarDim
				if s < rating {
					sc = colorLearnStar
				}
				drawStar(screen, starX+6, starY+6, 5, sc)
			}

			// Lock icon
			if locked {
				printColoredAt(screen, locale.T("learn.locked"), cardX+cardW-40, rowY+cardH-20, colorLearnLocked)
			}
		}
	}

	// Close hint
	hint := locale.T("learn.menu.hint")
	printColoredAt(screen, hint, (sw-runeLen(hint)*charW)/2, sh-38, colorLearnDim)

	// ESC hint at bottom
	escHint := locale.T("overlay.esc_close")
	printColoredAt(screen, escHint, (sw-runeLen(escHint)*charW)/2, sh-22, color.RGBA{120, 130, 150, 180})
}

// drawStar draws a simple 5-pointed star using lines.
func drawStar(screen *ebiten.Image, cx, cy, r int, col color.RGBA) {
	pts := 5
	for i := 0; i < pts; i++ {
		a1 := float64(i)*2*math.Pi/float64(pts) - math.Pi/2
		a2 := float64(i+2)*2*math.Pi/float64(pts) - math.Pi/2
		x1 := float32(cx) + float32(r)*float32(math.Cos(a1))
		y1 := float32(cy) + float32(r)*float32(math.Sin(a1))
		x2 := float32(cx) + float32(r)*float32(math.Cos(a2))
		y2 := float32(cy) + float32(r)*float32(math.Sin(a2))
		vector.StrokeLine(screen, x1, y1, x2, y2, 1.5, col, false)
	}
}

// splitLines splits a string on '|' delimiters for multi-line display.
func splitLines(s string) []string {
	var result []string
	cur := ""
	for _, r := range s {
		if r == '|' {
			result = append(result, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	result = append(result, cur)
	return result
}

// LessonMenuHitTest returns the lesson index clicked, or -1.
func LessonMenuHitTest(mx, my, sw, sh int) int {
	cardW := 160
	cardH := 120
	gapX := 15
	gapY := 15
	startY := 55

	for level := 0; level < 3; level++ {
		rowY := startY + level*(cardH+gapY+20) + 18

		for col := 0; col < 4; col++ {
			idx := level*4 + col
			if idx >= int(swarm.LessonCount) {
				break
			}
			cardX := 30 + col*(cardW+gapX)
			if mx >= cardX && mx < cardX+cardW && my >= rowY && my < rowY+cardH {
				return idx
			}
		}
	}
	return -1
}

// LessonNextHitTest checks if the "Next" or "Exit" button was clicked during a lesson.
// Returns "next", "exit", or "".
func LessonNextHitTest(mx, my, sw, sh int, ls *swarm.LearningState) string {
	if ls == nil || !ls.Active {
		return ""
	}

	boxW := 680
	boxH := 100
	boxX := (sw - boxW) / 2
	boxY := sh - boxH - 45

	if ls.ChallengeActive {
		// Challenge exit button
		cBoxW := 500
		cBoxH := 80
		cBoxX := (sw - cBoxW) / 2
		cBoxY := sh - cBoxH - 45
		btnY := cBoxY + cBoxH - 22
		if mx >= cBoxX+10 && mx < cBoxX+10+80 && my >= btnY && my < btnY+18 {
			return "exit"
		}
		return ""
	}

	btnY := boxY + boxH - 26

	// Exit button
	if mx >= boxX+10 && mx < boxX+10+80 && my >= btnY && my < btnY+22 {
		return "exit"
	}

	// Next button
	lessons := swarm.GetAllLessons()
	if int(ls.CurrentLesson) < len(lessons) && ls.CurrentStep < len(lessons[ls.CurrentLesson].Steps) {
		step := lessons[ls.CurrentLesson].Steps[ls.CurrentStep]
		if step.WaitTicks == 0 || ls.StepTimer >= step.WaitTicks {
			btnW := 80
			btnX := boxX + boxW - btnW - 10
			if mx >= btnX && mx < btnX+btnW && my >= btnY && my < btnY+22 {
				return "next"
			}
		}
	}

	return ""
}
