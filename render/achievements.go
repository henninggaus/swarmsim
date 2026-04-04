package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawAchievementPopup draws the toast notification when an achievement is unlocked.
func DrawAchievementPopup(screen *ebiten.Image, ss *swarm.SwarmState) {
	as := ss.AchievementState
	if as == nil || as.RecentPopup == nil {
		return
	}

	pop := as.RecentPopup
	def := swarm.AllAchievements[pop.ID]
	diffCol := swarm.DifficultyColor(def.Difficulty)

	// Slide-in animation: first 20 frames slide down, last 30 frames fade out
	sw := screen.Bounds().Dx()
	w := 320
	x := (sw - w) / 2
	y := 10

	alpha := uint8(240)
	if pop.Timer > 160 {
		// Slide in from top
		frac := float64(180-pop.Timer) / 20.0
		y = int(float64(y)*frac - float64(40)*(1-frac))
	} else if pop.Timer < 30 {
		// Fade out
		alpha = uint8(float64(pop.Timer) / 30.0 * 240)
	}

	h := 50

	// Background with difficulty-tinted border
	bgCol := color.RGBA{15, 15, 30, alpha}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), bgCol, false)
	borderCol := color.RGBA{diffCol[0], diffCol[1], diffCol[2], alpha}
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, borderCol, false)

	// Glow pulse
	pulse := 0.5 + 0.5*math.Sin(float64(pop.Timer)*0.2)
	glowAlpha := uint8(float64(alpha) * 0.15 * pulse)
	glowCol := color.RGBA{diffCol[0], diffCol[1], diffCol[2], glowAlpha}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), glowCol, false)

	// Icon + title
	textAlpha := alpha
	titleCol := color.RGBA{diffCol[0], diffCol[1], diffCol[2], textAlpha}
	line1 := fmt.Sprintf("%s  %s", def.Icon, def.DisplayName())
	printColoredAt(screen, line1, x+10, y+5, titleCol)

	// Achievement unlocked label
	unlockCol := color.RGBA{255, 255, 255, textAlpha}
	printColoredAt(screen, locale.T("ach.unlocked"), x+10, y+18, unlockCol)

	// Description
	descCol := color.RGBA{180, 180, 200, textAlpha}
	printColoredAt(screen, def.DisplayDesc(), x+10, y+33, descCol)
}

// DrawAchievementOverlay draws the full achievement list panel (Shift+B).
func DrawAchievementOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	as := ss.AchievementState
	if as == nil {
		return
	}

	sw := screen.Bounds().Dx()
	x := sw/2 - 200
	y := 60
	w := 400
	lineH := 18
	h := 50 + int(swarm.AchCount)*lineH

	// Clamp height
	sh := screen.Bounds().Dy()
	if y+h > sh-20 {
		h = sh - 20 - y
	}

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h),
		color.RGBA{10, 10, 20, 240}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h),
		1, color.RGBA{80, 80, 120, 255}, false)

	// Title
	titleStr := fmt.Sprintf("%s  [%d/%d]", locale.T("ach.title"), as.TotalUnlocked, int(swarm.AchCount))
	printColoredAt(screen, titleStr, x+10, y+5, color.RGBA{255, 220, 100, 255})

	// Progress bar
	barY := y + 20
	barW := w - 20
	vector.DrawFilledRect(screen, float32(x+10), float32(barY), float32(barW), 8,
		color.RGBA{30, 30, 50, 200}, false)
	if swarm.AchCount > 0 {
		fillW := int(float64(barW) * float64(as.TotalUnlocked) / float64(swarm.AchCount))
		vector.DrawFilledRect(screen, float32(x+10), float32(barY), float32(fillW), 8,
			color.RGBA{100, 255, 100, 200}, false)
	}

	// Achievement list
	ly := barY + 16
	maxY := y + h - 5
	for i := 0; i < int(swarm.AchCount); i++ {
		if ly > maxY {
			break
		}
		def := swarm.AllAchievements[i]
		ach := as.Achievements[i]
		diffCol := swarm.DifficultyColor(def.Difficulty)

		if ach.Unlocked {
			// Difficulty color dot
			vector.DrawFilledCircle(screen, float32(x+16), float32(ly+5), 4,
				color.RGBA{diffCol[0], diffCol[1], diffCol[2], 255}, false)
			// Name
			nameCol := ColorWhite
			printColoredAt(screen, def.Icon+" "+def.DisplayName(), x+24, ly, nameCol)
			// Description (shorter, right-aligned)
			descCol := color.RGBA{140, 140, 160, 200}
			printColoredAt(screen, def.DisplayDesc(), x+180, ly, descCol)
		} else {
			// Locked: gray
			vector.DrawFilledCircle(screen, float32(x+16), float32(ly+5), 4,
				color.RGBA{50, 50, 60, 200}, false)
			lockCol := color.RGBA{80, 80, 100, 150}
			printColoredAt(screen, "??? "+def.DisplayName(), x+24, ly, lockCol)
		}
		ly += lineH
	}
}
