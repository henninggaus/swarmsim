package render

import (
	"image/color"
	"math"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawAurora renders animated aurora borealis color bands on the arena background.
func drawAurora(a *ebiten.Image, tick int) {
	aw := a.Bounds().Dx()
	ah := a.Bounds().Dy()
	t := float64(tick)

	type band struct {
		baseY      float64
		amplitude  float64
		wavelength float64
		speed      float64
		bandH      float64
		r, g, b    uint8
		alpha      uint8
	}

	bands := []band{
		{float64(ah) * 0.3, 40, 200, 0.008, 60, 0, 255, 100, 18},
		{float64(ah) * 0.45, 50, 160, -0.006, 50, 50, 200, 255, 14},
		{float64(ah) * 0.55, 35, 240, 0.010, 45, 180, 80, 255, 12},
		{float64(ah) * 0.7, 30, 180, -0.005, 40, 100, 255, 180, 10},
	}

	step := 4
	for _, b := range bands {
		for x := 0; x < aw; x += step {
			fx := float64(x)
			y := b.baseY + b.amplitude*math.Sin(fx/b.wavelength*2*math.Pi+t*b.speed)
			// Secondary wave for organic movement
			y += b.amplitude * 0.3 * math.Sin(fx/b.wavelength*1.3*math.Pi+t*b.speed*1.7)
			col := color.RGBA{b.r, b.g, b.b, b.alpha}
			vector.DrawFilledRect(a, float32(x), float32(y), float32(step), float32(b.bandH), col, false)
		}
	}
}

// drawPredictionArrows renders fading arrows showing where each bot will be in ~20 ticks.
func drawPredictionArrows(a *ebiten.Image, ss *swarm.SwarmState) {
	predTicks := 20.0
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.Speed < 0.1 {
			continue
		}

		bx := bot.X
		by := bot.Y
		px := bx + math.Cos(bot.Angle)*bot.Speed*predTicks
		py := by + math.Sin(bot.Angle)*bot.Speed*predTicks

		// Clamp to arena
		if px < 0 {
			px = 0
		}
		if px > ss.ArenaW {
			px = ss.ArenaW
		}
		if py < 0 {
			py = 0
		}
		if py > ss.ArenaH {
			py = ss.ArenaH
		}

		col := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 50}

		// Main line
		vector.StrokeLine(a, float32(bx), float32(by), float32(px), float32(py), 1, col, false)

		// 3 intermediate dots with decreasing alpha
		for d := 1; d <= 3; d++ {
			frac := float64(d) / 4.0
			dx := bx + (px-bx)*frac
			dy := by + (py-by)*frac
			dotAlpha := uint8(40 - d*10)
			dotCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], dotAlpha}
			vector.DrawFilledCircle(a, float32(dx), float32(dy), 2, dotCol, false)
		}

		// Arrowhead at predicted position
		arrowLen := 6.0
		angle := bot.Angle
		ax1 := px - math.Cos(angle-0.4)*arrowLen
		ay1 := py - math.Sin(angle-0.4)*arrowLen
		ax2 := px - math.Cos(angle+0.4)*arrowLen
		ay2 := py - math.Sin(angle+0.4)*arrowLen
		arrowCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 60}
		vector.StrokeLine(a, float32(px), float32(py), float32(ax1), float32(ay1), 1, arrowCol, false)
		vector.StrokeLine(a, float32(px), float32(py), float32(ax2), float32(ay2), 1, arrowCol, false)
	}
}

// drawDayNightOverlay draws darkness overlay and bot LED glows for the day/night cycle.
func drawDayNightOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	brightness := 0.5 + 0.5*math.Cos(ss.DayNightPhase*2*math.Pi)
	if brightness >= 0.98 {
		return // full daylight, nothing to draw
	}

	aw := float32(ss.ArenaW)
	ah := float32(ss.ArenaH)

	// Darkness overlay
	darkness := 1.0 - brightness
	alpha := uint8(darkness * 180)
	vector.DrawFilledRect(a, 0, 0, aw, ah, color.RGBA{0, 0, 20, alpha}, false)

	// Bot LED glow (firefly effect) — only when dark enough
	if darkness > 0.3 {
		glowAlpha := uint8(darkness * 80)
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			r, g, b := bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2]
			if r < 30 && g < 30 && b < 30 {
				continue // skip very dark LEDs
			}
			glowCol := color.RGBA{r, g, b, glowAlpha}
			vector.DrawFilledCircle(a, float32(bot.X), float32(bot.Y), 15, glowCol, false)
		}
	}
}

// drawSwarmCenterOverlay draws a pulsing crosshair at the swarm center of mass and a spread circle.
func drawSwarmCenterOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	cx := float32(ss.SwarmCenterX)
	cy := float32(ss.SwarmCenterY)
	spread := float32(ss.SwarmSpread)

	// Pulsing crosshair
	pulse := float32(0.5 + 0.5*math.Sin(float64(ss.Tick)*0.1))
	alpha := uint8(100 + float32(80)*pulse)
	crossCol := color.RGBA{255, 255, 0, alpha}
	crossLen := float32(15)
	vector.StrokeLine(a, cx-crossLen, cy, cx+crossLen, cy, 1.5, crossCol, false)
	vector.StrokeLine(a, cx, cy-crossLen, cx, cy+crossLen, 1.5, crossCol, false)

	// Spread circle
	if spread > 5 {
		circleCol := color.RGBA{0, 255, 255, 25}
		vector.StrokeCircle(a, cx, cy, spread, 1, circleCol, false)
	}

	// Inner marker
	vector.DrawFilledCircle(a, cx, cy, 3, color.RGBA{255, 255, 0, 150}, false)
}

// drawCongestionOverlay draws pulsing red-orange halos where bots are stuck.
func drawCongestionOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	if ss.CongestionGrid == nil || ss.CongestionCols == 0 {
		return
	}

	cellSize := ss.ArenaW / float64(ss.CongestionCols)
	pulse := 0.5 + 0.5*math.Sin(float64(ss.Tick)*0.15)

	for row := 0; row < ss.CongestionRows; row++ {
		for col := 0; col < ss.CongestionCols; col++ {
			v := ss.CongestionGrid[row*ss.CongestionCols+col]
			if v < 0.1 {
				continue
			}
			if v > 1.0 {
				v = 1.0
			}
			alpha := uint8(v * 80 * (0.6 + 0.4*pulse))
			congCol := color.RGBA{255, uint8(80 + 40*pulse), 0, alpha}
			x := float32(float64(col) * cellSize)
			y := float32(float64(row) * cellSize)
			vector.DrawFilledRect(a, x, y, float32(cellSize), float32(cellSize), congCol, false)
		}
	}
}

// drawDashSpeedLines draws speed-line afterimages behind dashing bots.
func drawDashSpeedLines(a *ebiten.Image, ss *swarm.SwarmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.DashTimer <= 0 {
			continue
		}
		bx := float32(bot.X)
		by := float32(bot.Y)
		// 3 fading trail positions behind the bot
		for t := 1; t <= 3; t++ {
			dist := float64(t) * 8.0
			tx := bx - float32(math.Cos(bot.Angle)*dist)
			ty := by - float32(math.Sin(bot.Angle)*dist)
			alpha := uint8(60 - t*15)
			col := color.RGBA{255, 255, 255, alpha}
			vector.DrawFilledCircle(a, tx, ty, float32(swarm.SwarmBotRadius)*0.6, col, false)
		}
	}
}

