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

// drawFlowField renders a velocity flow field overlay showing the average
// direction and speed of bots in each grid cell as colored arrows. This makes
// swarm dynamics visible at a glance: flow patterns around obstacles, vortices,
// convergence zones, and divergence zones all become apparent.
func drawFlowField(a *ebiten.Image, ss *swarm.SwarmState) {
	const cellSize = 40.0 // grid cell size in pixels
	cols := int(ss.ArenaW / cellSize)
	rows := int(ss.ArenaH / cellSize)
	if cols < 1 || rows < 1 {
		return
	}

	n := cols * rows
	// Accumulate velocity vectors per cell
	vxSum := make([]float64, n)
	vySum := make([]float64, n)
	count := make([]int, n)

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.Speed < 0.01 {
			continue
		}
		col := int(bot.X / cellSize)
		row := int(bot.Y / cellSize)
		if col < 0 || col >= cols || row < 0 || row >= rows {
			continue
		}
		idx := row*cols + col
		vxSum[idx] += math.Cos(bot.Angle) * bot.Speed
		vySum[idx] += math.Sin(bot.Angle) * bot.Speed
		count[idx]++
	}

	// Find max magnitude for normalization
	maxMag := 0.0
	for i := 0; i < n; i++ {
		if count[i] == 0 {
			continue
		}
		vx := vxSum[i] / float64(count[i])
		vy := vySum[i] / float64(count[i])
		mag := math.Sqrt(vx*vx + vy*vy)
		if mag > maxMag {
			maxMag = mag
		}
	}
	if maxMag < 0.01 {
		return
	}

	half := cellSize / 2.0
	arrowMax := cellSize * 0.4 // max arrow length

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if count[idx] == 0 {
				continue
			}
			vx := vxSum[idx] / float64(count[idx])
			vy := vySum[idx] / float64(count[idx])
			mag := math.Sqrt(vx*vx + vy*vy)
			if mag < 0.01 {
				continue
			}

			// Normalize direction, scale length by magnitude
			ratio := mag / maxMag
			length := arrowMax * ratio
			dx := (vx / mag) * length
			dy := (vy / mag) * length

			cx := float64(col)*cellSize + half
			cy := float64(row)*cellSize + half
			x1 := float32(cx - dx*0.5)
			y1 := float32(cy - dy*0.5)
			x2 := float32(cx + dx*0.5)
			y2 := float32(cy + dy*0.5)

			// Color: blue (slow) → cyan → green → yellow (fast)
			alpha := uint8(80 + ratio*140)
			var r8, g8, b8 uint8
			if ratio < 0.33 {
				t := ratio / 0.33
				r8, g8, b8 = 0, uint8(100+155*t), uint8(255-100*t)
			} else if ratio < 0.66 {
				t := (ratio - 0.33) / 0.33
				r8, g8, b8 = uint8(200*t), 255, uint8(155-155*t)
			} else {
				t := (ratio - 0.66) / 0.34
				r8, g8, b8 = uint8(200+55*t), uint8(255-55*t), 0
			}
			arrowCol := color.RGBA{r8, g8, b8, alpha}

			// Shaft
			vector.StrokeLine(a, x1, y1, x2, y2, 1.5, arrowCol, false)

			// Arrowhead (two small lines from tip)
			ang := math.Atan2(float64(dy), float64(dx))
			headLen := float32(length * 0.3)
			if headLen < 3 {
				headLen = 3
			}
			for _, offset := range []float64{2.5, -2.5} {
				ha := ang + math.Pi + offset
				hx := x2 + headLen*float32(math.Cos(ha))
				hy := y2 + headLen*float32(math.Sin(ha))
				vector.StrokeLine(a, x2, y2, hx, hy, 1.2, arrowCol, false)
			}

			// Bot count indicator (small dot, alpha proportional to density)
			dotAlpha := uint8(40 + count[idx]*15)
			if dotAlpha > 200 {
				dotAlpha = 200
			}
			vector.DrawFilledCircle(a, float32(cx), float32(cy), 2, color.RGBA{255, 255, 255, dotAlpha}, false)
		}
	}
}

// drawFitnessGradientField renders the gradient of the fitness landscape as
// colored arrows on a grid. Each arrow points in the direction of steepest
// ascent (improving fitness) with length proportional to gradient magnitude.
// Color encodes magnitude: purple (flat) → magenta → red → orange → yellow
// (steep). This helps users understand the optimisation surface topology and
// see where algorithms should be heading vs where they actually move.
func drawFitnessGradientField(a *ebiten.Image, ss *swarm.SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil {
		return
	}

	const cellSize = 30.0 // grid resolution in pixels
	const epsilon = 2.0   // finite-difference step for gradient estimation
	cols := int(ss.ArenaW / cellSize)
	rows := int(ss.ArenaH / cellSize)
	if cols < 1 || rows < 1 {
		return
	}

	n := cols * rows
	gxArr := make([]float64, n) // gradient X component
	gyArr := make([]float64, n) // gradient Y component
	magArr := make([]float64, n)

	// Compute gradient via central finite differences at each grid center
	maxMag := 0.0
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cx := float64(col)*cellSize + cellSize/2
			cy := float64(row)*cellSize + cellSize/2

			// Central difference: df/dx ≈ (f(x+ε)-f(x-ε)) / (2ε)
			fxp := swarm.EvaluateFitnessLandscape(sa, cx+epsilon, cy)
			fxm := swarm.EvaluateFitnessLandscape(sa, cx-epsilon, cy)
			fyp := swarm.EvaluateFitnessLandscape(sa, cx, cy+epsilon)
			fym := swarm.EvaluateFitnessLandscape(sa, cx, cy-epsilon)

			dx := (fxp - fxm) / (2 * epsilon)
			dy := (fyp - fym) / (2 * epsilon)
			mag := math.Sqrt(dx*dx + dy*dy)

			idx := row*cols + col
			gxArr[idx] = dx
			gyArr[idx] = dy
			magArr[idx] = mag
			if mag > maxMag {
				maxMag = mag
			}
		}
	}

	if maxMag < 1e-12 {
		return
	}

	half := cellSize / 2.0
	arrowMax := cellSize * 0.4

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			mag := magArr[idx]
			if mag < maxMag*0.02 { // skip near-zero gradient cells
				continue
			}

			dx := gxArr[idx]
			dy := gyArr[idx]
			ratio := mag / maxMag
			length := arrowMax * ratio
			if length < 2 {
				length = 2
			}

			// Normalised direction scaled by length
			ndx := (dx / mag) * length
			ndy := (dy / mag) * length

			cx := float64(col)*cellSize + half
			cy := float64(row)*cellSize + half
			x1 := float32(cx - ndx*0.5)
			y1 := float32(cy - ndy*0.5)
			x2 := float32(cx + ndx*0.5)
			y2 := float32(cy + ndy*0.5)

			// Color: purple (flat) → magenta → red → orange → yellow (steep)
			alpha := uint8(60 + ratio*160)
			var r8, g8, b8 uint8
			if ratio < 0.25 {
				t := ratio / 0.25
				r8, g8, b8 = uint8(80+80*t), 0, uint8(180-60*t) // purple→magenta
			} else if ratio < 0.5 {
				t := (ratio - 0.25) / 0.25
				r8, g8, b8 = uint8(160+95*t), 0, uint8(120-120*t) // magenta→red
			} else if ratio < 0.75 {
				t := (ratio - 0.5) / 0.25
				r8, g8, b8 = 255, uint8(120 * t), 0 // red→orange
			} else {
				t := (ratio - 0.75) / 0.25
				r8, g8, b8 = 255, uint8(120+135*t), 0 // orange→yellow
			}
			arrowCol := color.RGBA{r8, g8, b8, alpha}

			// Shaft
			vector.StrokeLine(a, x1, y1, x2, y2, 1.3, arrowCol, false)

			// Arrowhead
			ang := math.Atan2(ndy, ndx)
			headLen := float32(length * 0.3)
			if headLen < 2.5 {
				headLen = 2.5
			}
			for _, off := range []float64{2.5, -2.5} {
				ha := ang + math.Pi + off
				hx := x2 + headLen*float32(math.Cos(ha))
				hy := y2 + headLen*float32(math.Sin(ha))
				vector.StrokeLine(a, x2, y2, hx, hy, 1.0, arrowCol, false)
			}
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

// drawVoronoiOverlay renders a Voronoi tessellation showing each bot's
// nearest-neighbor territory. Each cell of a reduced-resolution grid is
// assigned to the closest bot and tinted with that bot's LED color at low
// alpha. Cell boundaries where the owner changes are drawn as thin lines,
// creating a territory map that makes spatial partitioning visible at a glance.
// Useful for understanding coverage patterns, density variations, and how
// algorithms distribute agents across the search space.
func drawVoronoiOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	if len(ss.Bots) == 0 {
		return
	}

	const cellSize = 10.0 // resolution in pixels
	cols := int(ss.ArenaW / cellSize)
	rows := int(ss.ArenaH / cellSize)
	if cols < 1 || rows < 1 {
		return
	}

	// Build owner grid: owner[r*cols+c] = index of nearest bot
	n := cols * rows
	owner := make([]int, n)

	// For each cell center, find the nearest bot via brute scan of bot list.
	// At 10px resolution on 800x800 arena this is 6400 cells x N bots.
	// For up to ~500 bots this is ~3.2M distance checks per frame which is
	// fast enough given the simple arithmetic (no sqrt needed — compare d2).
	for r := 0; r < rows; r++ {
		cy := (float64(r) + 0.5) * cellSize
		for c := 0; c < cols; c++ {
			cx := (float64(c) + 0.5) * cellSize
			bestIdx := 0
			bestD2 := math.MaxFloat64
			for i := range ss.Bots {
				dx := ss.Bots[i].X - cx
				dy := ss.Bots[i].Y - cy
				d2 := dx*dx + dy*dy
				if d2 < bestD2 {
					bestD2 = d2
					bestIdx = i
				}
			}
			owner[r*cols+c] = bestIdx
		}
	}

	// Draw cell fills — tinted with owner LED color at low alpha
	const fillAlpha = uint8(30)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			idx := owner[r*cols+c]
			led := ss.Bots[idx].LEDColor
			col := color.RGBA{led[0], led[1], led[2], fillAlpha}
			fx := float32(c) * cellSize
			fy := float32(r) * cellSize
			vector.DrawFilledRect(a, fx, fy, cellSize, cellSize, col, false)
		}
	}

	// Draw Voronoi edges: where adjacent cells have different owners
	edgeCol := color.RGBA{180, 200, 255, 60}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			me := owner[r*cols+c]
			fx := float32(c) * cellSize
			fy := float32(r) * cellSize
			// Right neighbor
			if c+1 < cols && owner[r*cols+c+1] != me {
				x := fx + cellSize
				vector.StrokeLine(a, x, fy, x, fy+cellSize, 1, edgeCol, false)
			}
			// Bottom neighbor
			if r+1 < rows && owner[(r+1)*cols+c] != me {
				y := fy + cellSize
				vector.StrokeLine(a, fx, y, fx+cellSize, y, 1, edgeCol, false)
			}
		}
	}

	// Label
	printColoredAt(a, "VORONOI (Ctrl+V)", 5, int(ss.ArenaH)-14, color.RGBA{180, 200, 255, 180})
}

