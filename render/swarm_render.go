package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawSwarmMode renders the entire swarm mode screen (arena on the right).
func (r *Renderer) DrawSwarmMode(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	ss := s.SwarmState
	if ss == nil {
		return
	}

	// Fill entire screen with editor bg (left panel drawn by DrawSwarmEditor)
	screen.Fill(ColorSwarmEditorBg)

	// Arena viewport starts at x=350, arena is 800x800, centered in 930x900
	arenaOffX := 415.0 // 350 + (930-800)/2
	arenaOffY := 50.0  // (900-800)/2

	// Arena background
	vector.DrawFilledRect(screen, float32(arenaOffX), float32(arenaOffY),
		float32(ss.ArenaW), float32(ss.ArenaH), ColorSwarmArenaBg, false)

	// Arena grid
	gridStep := 50.0
	for gx := gridStep; gx < ss.ArenaW; gx += gridStep {
		sx := float32(arenaOffX + gx)
		vector.StrokeLine(screen, sx, float32(arenaOffY), sx, float32(arenaOffY+ss.ArenaH), 1, ColorSwarmArenaGrid, false)
	}
	for gy := gridStep; gy < ss.ArenaH; gy += gridStep {
		sy := float32(arenaOffY + gy)
		vector.StrokeLine(screen, float32(arenaOffX), sy, float32(arenaOffX+ss.ArenaW), sy, 1, ColorSwarmArenaGrid, false)
	}

	// Arena border
	vector.StrokeRect(screen, float32(arenaOffX), float32(arenaOffY),
		float32(ss.ArenaW), float32(ss.ArenaH), 2, ColorSwarmArenaBorder, false)

	// Obstacles (3D effect: lighter top-left edges, darker bottom-right)
	for _, obs := range ss.Obstacles {
		ox := float32(arenaOffX + obs.X)
		oy := float32(arenaOffY + obs.Y)
		ow := float32(obs.W)
		oh := float32(obs.H)
		// Body
		vector.DrawFilledRect(screen, ox, oy, ow, oh, ColorSwarmObstacle, false)
		// Top edge highlight
		vector.StrokeLine(screen, ox, oy, ox+ow, oy, 2, ColorSwarmObstacleHi, false)
		// Left edge highlight
		vector.StrokeLine(screen, ox, oy, ox, oy+oh, 2, ColorSwarmObstacleHi, false)
		// Bottom edge shadow
		vector.StrokeLine(screen, ox, oy+oh, ox+ow, oy+oh, 2, ColorSwarmObstacleLo, false)
		// Right edge shadow
		vector.StrokeLine(screen, ox+ow, oy, ox+ow, oy+oh, 2, ColorSwarmObstacleLo, false)
	}

	// Maze walls (thin colored rects)
	for _, wall := range ss.MazeWalls {
		wx := float32(arenaOffX + wall.X)
		wy := float32(arenaOffY + wall.Y)
		ww := float32(wall.W)
		wh := float32(wall.H)
		vector.DrawFilledRect(screen, wx, wy, ww, wh, ColorSwarmMazeWall, false)
	}

	// Light source (concentric circles with decreasing alpha)
	if ss.Light.Active {
		lx := float32(arenaOffX + ss.Light.X)
		ly := float32(arenaOffY + ss.Light.Y)
		for ri := 4; ri >= 1; ri-- {
			radius := float32(ri) * 25.0
			alpha := uint8(25 - ri*4)
			if alpha < 5 {
				alpha = 5
			}
			lightCol := color.RGBA{ColorSwarmLight.R, ColorSwarmLight.G, ColorSwarmLight.B, alpha}
			vector.DrawFilledCircle(screen, lx, ly, radius, lightCol, false)
		}
		// Bright center
		vector.DrawFilledCircle(screen, lx, ly, 6, ColorSwarmLight, false)
		vector.StrokeCircle(screen, lx, ly, 10, 1.5, color.RGBA{255, 255, 100, 150}, false)
	}

	// Delivery rendering
	if ss.DeliveryOn {
		// Route lines behind everything (toggle with 'C')
		if ss.ShowRoutes {
			drawPickupDropoffRoutes(screen, ss, arenaOffX, arenaOffY)
		}
		// Dashed lines from carrying bots to their matching dropoff (only when routes toggled)
		if ss.ShowRoutes {
			hasCarrying := false
			for i := range ss.Bots {
				if ss.Bots[i].CarryingPkg >= 0 {
					hasCarrying = true
					break
				}
			}
			if hasCarrying {
				drawCarryRouteLines(screen, ss, arenaOffX, arenaOffY)
			}
		}
		// Stations with labels, timer bars, counters
		drawDeliveryStations(screen, ss, arenaOffX, arenaOffY)
		drawStationLabels(screen, ss, arenaOffX, arenaOffY)
		// Ground packages with pulse and arrow
		drawDeliveryPackages(screen, ss, arenaOffX, arenaOffY)
	}

	// Trails (small circles with decreasing alpha for last 10 positions)
	if ss.ShowTrails {
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			for t := 0; t < len(bot.Trail); t++ {
				tx := bot.Trail[t][0]
				ty := bot.Trail[t][1]
				if tx == 0 && ty == 0 {
					continue // unused slot
				}
				// Calculate age: newest trail point has highest alpha
				age := (bot.TrailIdx - t - 1 + len(bot.Trail)) % len(bot.Trail)
				alpha := uint8(60 - age*5)
				if alpha < 10 {
					alpha = 10
				}
				trailCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], alpha}
				sx := float32(arenaOffX + tx)
				sy := float32(arenaOffY + ty)
				vector.DrawFilledCircle(screen, sx, sy, 2, trailCol, false)
			}
		}
	}

	// Follow lines (thin line from follower to leader in LED color)
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.FollowTargetIdx >= 0 && bot.FollowTargetIdx < len(ss.Bots) {
			target := &ss.Bots[bot.FollowTargetIdx]
			bx := float32(arenaOffX + bot.X)
			by := float32(arenaOffY + bot.Y)
			tx := float32(arenaOffX + target.X)
			ty := float32(arenaOffY + target.Y)
			lineCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 120}
			vector.StrokeLine(screen, bx, by, tx, ty, 1, lineCol, false)
		}
	}

	// Draw bots
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		bx := float32(arenaOffX + bot.X)
		by := float32(arenaOffY + bot.Y)
		radius := float32(swarm.SwarmBotRadius)

		// Bot body circle with LED color
		botCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 255}
		vector.DrawFilledCircle(screen, bx, by, radius, botCol, false)

		// Direction indicator line
		dirLen := radius * 1.5
		dx := float32(math.Cos(bot.Angle)) * dirLen
		dy := float32(math.Sin(bot.Angle)) * dirLen
		vector.StrokeLine(screen, bx, by, bx+dx, by+dy, 1.5, color.RGBA{255, 255, 255, 200}, false)

		// Carried package indicator: glow ring + small colored square
		if ss.DeliveryOn && bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			pkg := &ss.Packages[bot.CarryingPkg]
			pkgCol := deliveryColor(pkg.Color)
			// Pulsing glow ring
			pulse := 0.6 + 0.4*math.Sin(float64(ss.Tick)*0.1)
			glowAlpha := uint8(60 + 40*pulse)
			glowCol := color.RGBA{pkgCol.R, pkgCol.G, pkgCol.B, glowAlpha}
			vector.StrokeCircle(screen, bx, by, radius+5, 2, glowCol, false)
			// Package square above bot
			vector.DrawFilledRect(screen, bx-4, by-radius-8, 8, 8, pkgCol, false)
			vector.StrokeRect(screen, bx-4, by-radius-8, 8, 8, 1, color.RGBA{255, 255, 255, 180}, false)
		}

		// Deploy blink overlay (green flash)
		if bot.BlinkTimer > 0 && (bot.BlinkTimer/4)%2 == 0 {
			blinkCol := ColorSwarmBotBlink
			blinkCol.A = 120
			vector.DrawFilledCircle(screen, bx, by, radius+2, blinkCol, false)
		}

		// Selected bot highlight (pulsing ring)
		if i == ss.SelectedBot {
			pulse := float32(2.0 + 2.0*math.Sin(float64(ss.Tick)*0.12))
			pulseAlpha := uint8(150 + int(50*math.Sin(float64(ss.Tick)*0.08)))
			selCol := color.RGBA{ColorSwarmSelected.R, ColorSwarmSelected.G, ColorSwarmSelected.B, pulseAlpha}
			vector.StrokeCircle(screen, bx, by, radius+pulse+2, 2, selCol, false)
		}
	}

	// Ghost bots (wrap mode only — draw 50% alpha copies near opposite edges)
	if ss.WrapMode {
		ghostMargin := 40.0
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			ghostCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 128}
			radius := float32(swarm.SwarmBotRadius)

			// Check if near each edge and draw ghost on opposite side
			drawGhost := func(gx, gy float64) {
				sx := float32(arenaOffX + gx)
				sy := float32(arenaOffY + gy)
				vector.DrawFilledCircle(screen, sx, sy, radius, ghostCol, false)
			}

			if bot.X < ghostMargin {
				drawGhost(bot.X+ss.ArenaW, bot.Y)
			}
			if bot.X > ss.ArenaW-ghostMargin {
				drawGhost(bot.X-ss.ArenaW, bot.Y)
			}
			if bot.Y < ghostMargin {
				drawGhost(bot.X, bot.Y+ss.ArenaH)
			}
			if bot.Y > ss.ArenaH-ghostMargin {
				drawGhost(bot.X, bot.Y-ss.ArenaH)
			}
		}
	}

	// Delivery overlays (on top of bots)
	if ss.DeliveryOn {
		// Score popups (floating text)
		drawScorePopups(screen, ss, arenaOffX, arenaOffY)
		// Delivery legend (top-right)
		drawDeliveryLegend(screen, ss, sw)
		// Process delivery events → particle effects
		processDeliveryEvents(r, ss)
	}

	// Draw delivery particles (on top)
	if r.SwarmParticles != nil {
		r.SwarmParticles.Update()
		drawSwarmParticles(screen, r.SwarmParticles, arenaOffX, arenaOffY)
	}

	// Selected bot info overlay
	if ss.SelectedBot >= 0 && ss.SelectedBot < len(ss.Bots) {
		drawSelectedBotInfo(screen, ss)
	}

	// Minimap
	if r.ShowMinimap {
		r.drawSwarmMinimap(screen, ss)
	}

	// Separator line between editor and arena
	vector.StrokeLine(screen, 350, 0, 350, float32(sh), 2, ColorSwarmEditorSep, false)
}

// drawSelectedBotInfo draws an info panel for the selected bot.
func drawSelectedBotInfo(screen *ebiten.Image, ss *swarm.SwarmState) {
	bot := &ss.Bots[ss.SelectedBot]
	x := 1050
	y := 60
	w := 210
	h := 200
	col := color.RGBA{200, 200, 220, 255}

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), ColorSwarmInfoBg, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, ColorSwarmEditorSep, false)

	lx := x + 5
	ly := y + 5

	printColoredAt(screen, fmt.Sprintf("Bot #%d", ss.SelectedBot), lx, ly, ColorSwarmSelected)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Pos: %.0f, %.0f", bot.X, bot.Y), lx, ly, col)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Angle: %.0f deg", bot.Angle*180/math.Pi), lx, ly, col)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("State:%d Counter:%d", bot.State, bot.Counter), lx, ly, col)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("V1:%d V2:%d Timer:%d", bot.Value1, bot.Value2, bot.Timer), lx, ly, col)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("LED: %d,%d,%d", bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2]), lx, ly, col)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Neighbors: %d", bot.NeighborCount), lx, ly, col)
	ly += lineH

	followStr := "None"
	if bot.FollowTargetIdx >= 0 {
		followStr = fmt.Sprintf("#%d", bot.FollowTargetIdx)
	}
	followerStr := "None"
	if bot.FollowerIdx >= 0 {
		followerStr = fmt.Sprintf("#%d", bot.FollowerIdx)
	}
	printColoredAt(screen, fmt.Sprintf("Follow:%s Follower:%s", followStr, followerStr), lx, ly, col)
	ly += lineH

	obsStr := "No"
	if bot.ObstacleAhead {
		obsStr = fmt.Sprintf("Yes (%.0f)", bot.ObstacleDist)
	}
	printColoredAt(screen, fmt.Sprintf("Obstacle: %s", obsStr), lx, ly, col)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Speed: %.1f  Msg: %d", bot.Speed, bot.ReceivedMsg), lx, ly, col)

	if ss.DeliveryOn {
		ly += lineH
		carryStr := "None"
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			carryStr = swarm.DeliveryColorName(ss.Packages[bot.CarryingPkg].Color)
		}
		printColoredAt(screen, fmt.Sprintf("Carrying: %s", carryStr), lx, ly, col)
	}
}

// deliveryColor returns the RGBA color for a delivery color ID (1-4).
func deliveryColor(c int) color.RGBA {
	switch c {
	case 1:
		return color.RGBA{255, 60, 60, 255} // red
	case 2:
		return color.RGBA{60, 100, 255, 255} // blue
	case 3:
		return color.RGBA{255, 220, 40, 255} // yellow
	case 4:
		return color.RGBA{40, 200, 60, 255} // green
	}
	return color.RGBA{200, 200, 200, 255}
}

// drawDeliveryStations renders pickup and dropoff stations.
func drawDeliveryStations(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for si := range ss.Stations {
		st := &ss.Stations[si]
		sx := float32(offX + st.X)
		sy := float32(offY + st.Y)
		col := deliveryColor(st.Color)
		r := float32(25)

		if st.IsPickup {
			// Pickup: filled circle
			vector.DrawFilledCircle(screen, sx, sy, r, col, false)
			// Outline
			vector.StrokeCircle(screen, sx, sy, r, 2, color.RGBA{col.R, col.G, col.B, 180}, false)
			// Package icon inside (small square) if has package
			if st.HasPackage {
				// Pulse animation
				alpha := uint8(255)
				if ss.Tick%40 < 20 {
					alpha = 200
				}
				pkgCol := color.RGBA{255, 255, 255, alpha}
				vector.DrawFilledRect(screen, sx-5, sy-5, 10, 10, pkgCol, false)
				// Up arrow ↑
				vector.StrokeLine(screen, sx, sy-8, sx, sy+2, 2, color.RGBA{255, 255, 255, 200}, false)
				vector.StrokeLine(screen, sx-3, sy-5, sx, sy-8, 1.5, color.RGBA{255, 255, 255, 200}, false)
				vector.StrokeLine(screen, sx+3, sy-5, sx, sy-8, 1.5, color.RGBA{255, 255, 255, 200}, false)
			} else {
				// Empty: dashed outline look (dimmed)
				dimCol := color.RGBA{col.R / 2, col.G / 2, col.B / 2, 120}
				vector.StrokeCircle(screen, sx, sy, r-4, 1, dimCol, false)
			}
		} else {
			// Dropoff: ring (not filled)
			vector.StrokeCircle(screen, sx, sy, r, 3, col, false)
			// Inner target ring
			vector.StrokeCircle(screen, sx, sy, r*0.5, 2, col, false)
			// Down arrow ↓
			vector.StrokeLine(screen, sx, sy-4, sx, sy+6, 2, col, false)
			vector.StrokeLine(screen, sx-3, sy+3, sx, sy+6, 1.5, col, false)
			vector.StrokeLine(screen, sx+3, sy+3, sx, sy+6, 1.5, col, false)

			// Flash effect on delivery
			if st.FlashTimer > 0 {
				flashAlpha := uint8(st.FlashTimer * 8)
				if flashAlpha > 200 {
					flashAlpha = 200
				}
				var flashCol color.RGBA
				if st.FlashOK {
					flashCol = color.RGBA{0, 255, 0, flashAlpha}
				} else {
					flashCol = color.RGBA{255, 0, 0, flashAlpha}
				}
				flashR := r + float32(30-st.FlashTimer)
				vector.StrokeCircle(screen, sx, sy, flashR, 3, flashCol, false)
			}
		}
	}
}

// drawDeliveryPackages renders packages on the ground with pulsing alpha and arrow to dropoff.
func drawDeliveryPackages(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for pi := range ss.Packages {
		pkg := &ss.Packages[pi]
		if !pkg.Active || pkg.CarriedBy >= 0 || !pkg.OnGround {
			continue
		}
		px := float32(offX + pkg.X)
		py := float32(offY + pkg.Y)
		col := deliveryColor(pkg.Color)
		// Pulsing alpha
		pulse := 0.6 + 0.4*math.Sin(float64(ss.Tick)*0.08)
		col.A = uint8(float64(col.A) * pulse)
		vector.DrawFilledRect(screen, px-5, py-5, 10, 10, col, false)
		vector.StrokeRect(screen, px-5, py-5, 10, 10, 1, color.RGBA{255, 255, 255, 150}, false)
		// Small arrow pointing to matching dropoff
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup || st.Color != pkg.Color {
				continue
			}
			dx := st.X - pkg.X
			dy := st.Y - pkg.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 1 {
				nx := dx / dist
				ny := dy / dist
				ax := px + float32(nx*15)
				ay := py + float32(ny*15)
				arrowCol := color.RGBA{col.R, col.G, col.B, 140}
				vector.StrokeLine(screen, px, py, ax, ay, 1.5, arrowCol, false)
			}
			break // first matching dropoff
		}
	}
}

// drawStationLabels renders "PR","PB","PY","PG" / "DR","DB","DY","DG" labels
// above each station, plus respawn timer bars and delivery counters.
func drawStationLabels(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	colorLetters := [5]string{"", "R", "B", "Y", "G"}
	for si := range ss.Stations {
		st := &ss.Stations[si]
		sx := int(offX + st.X)
		sy := int(offY + st.Y)
		prefix := "D"
		if st.IsPickup {
			prefix = "P"
		}
		letter := ""
		if st.Color >= 1 && st.Color <= 4 {
			letter = colorLetters[st.Color]
		}
		label := prefix + letter
		col := deliveryColor(st.Color)
		// Label above station
		printColoredAt(screen, label, sx-6, sy-40, col)

		if st.IsPickup {
			// Respawn timer bar below empty pickup stations
			if !st.HasPackage && st.RespawnIn > 0 {
				barW := 30
				barH := 4
				bx := float32(sx - barW/2)
				by := float32(sy + 30)
				// Background
				vector.DrawFilledRect(screen, bx, by, float32(barW), float32(barH),
					color.RGBA{40, 40, 40, 180}, false)
				// Progress fill
				progress := 1.0 - float64(st.RespawnIn)/100.0
				fillW := float32(float64(barW) * progress)
				vector.DrawFilledRect(screen, bx, by, fillW, float32(barH), col, false)
			}
		} else {
			// Delivery counter below dropoff
			if st.DeliverCount > 0 {
				countStr := fmt.Sprintf("%d", st.DeliverCount)
				printColoredAt(screen, countStr, sx-len(countStr)*3, sy+30,
					color.RGBA{255, 255, 255, 220})
			}
		}
	}
}

// drawPickupDropoffRoutes draws thin dashed lines between same-color pickup/dropoff pairs.
func drawPickupDropoffRoutes(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for si := range ss.Stations {
		st := &ss.Stations[si]
		if !st.IsPickup {
			continue
		}
		// Find matching dropoff
		for di := range ss.Stations {
			dst := &ss.Stations[di]
			if dst.IsPickup || dst.Color != st.Color {
				continue
			}
			x1 := float32(offX + st.X)
			y1 := float32(offY + st.Y)
			x2 := float32(offX + dst.X)
			y2 := float32(offY + dst.Y)
			col := deliveryColor(st.Color)
			col.A = 40
			drawDashedLine(screen, x1, y1, x2, y2, 8, 6, 1, col)
		}
	}
}

// drawCarryRouteLines draws dashed lines from carrying bots to their matching dropoff.
func drawCarryRouteLines(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.CarryingPkg < 0 || bot.CarryingPkg >= len(ss.Packages) {
			continue
		}
		pkg := &ss.Packages[bot.CarryingPkg]
		// Find nearest matching dropoff
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup || st.Color != pkg.Color {
				continue
			}
			bx := float32(offX + bot.X)
			by := float32(offY + bot.Y)
			dx := float32(offX + st.X)
			dy := float32(offY + st.Y)
			col := deliveryColor(pkg.Color)
			col.A = 60
			drawDashedLine(screen, bx, by, dx, dy, 6, 4, 1, col)
			break // first matching dropoff
		}
	}
}

// drawDashedLine draws a dashed line between two points.
func drawDashedLine(screen *ebiten.Image, x1, y1, x2, y2, segLen, gapLen, width float32, col color.RGBA) {
	dx := x2 - x1
	dy := y2 - y1
	totalLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if totalLen < 1 {
		return
	}
	nx := dx / totalLen
	ny := dy / totalLen
	pos := float32(0)
	for pos < totalLen {
		end := pos + segLen
		if end > totalLen {
			end = totalLen
		}
		sx := x1 + nx*pos
		sy := y1 + ny*pos
		ex := x1 + nx*end
		ey := y1 + ny*end
		vector.StrokeLine(screen, sx, sy, ex, ey, width, col, false)
		pos += segLen + gapLen
	}
}

// drawScorePopups renders floating score text that rises and fades.
func drawScorePopups(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for i := range ss.ScorePopups {
		sp := &ss.ScorePopups[i]
		alpha := float64(sp.Timer) / 60.0
		if alpha > 1 {
			alpha = 1
		}
		col := color.RGBA{sp.Color[0], sp.Color[1], sp.Color[2], uint8(255 * alpha)}
		sx := int(offX+sp.X) - len(sp.Text)*3
		sy := int(offY + sp.Y)
		printColoredAt(screen, sp.Text, sx, sy, col)
	}
}

// drawDeliveryLegend draws a per-color delivery status legend in the top-right corner.
func drawDeliveryLegend(screen *ebiten.Image, ss *swarm.SwarmState, sw int) {
	colorNames := [5]string{"", "Red", "Blue", "Yel", "Grn"}
	x := sw - 130
	y := 55

	// Background
	vector.DrawFilledRect(screen, float32(x-5), float32(y-5), 128, 78,
		color.RGBA{20, 20, 30, 200}, false)
	vector.StrokeRect(screen, float32(x-5), float32(y-5), 128, 78, 1,
		color.RGBA{100, 100, 120, 150}, false)

	printColoredAt(screen, "Delivery", x, y, color.RGBA{255, 220, 100, 255})
	y += 14

	for c := 1; c <= 4; c++ {
		// Count in-transit for this color
		inTransit := 0
		for bi := range ss.Bots {
			if ss.Bots[bi].CarryingPkg >= 0 && ss.Bots[bi].CarryingPkg < len(ss.Packages) {
				if ss.Packages[ss.Bots[bi].CarryingPkg].Color == c {
					inTransit++
				}
			}
		}
		delivered := ss.DeliveryStats.ColorDelivered[c]
		col := deliveryColor(c)
		// Color swatch
		vector.DrawFilledRect(screen, float32(x), float32(y+2), 8, 8, col, false)
		// Text: "Red: 2→ 5"
		info := fmt.Sprintf("%s:%d>%d", colorNames[c], inTransit, delivered)
		printColoredAt(screen, info, x+12, y, color.RGBA{220, 220, 220, 255})
		y += 14
	}
}

// processDeliveryEvents processes pending delivery events for particle effects.
func processDeliveryEvents(r *Renderer, ss *swarm.SwarmState) {
	if r.SwarmParticles == nil {
		return
	}
	for _, ev := range ss.DeliveryEvents {
		col := deliveryColor(ev.Color)
		if ev.IsPickup {
			// Small burst on pickup
			r.SwarmParticles.Emit(ev.X, ev.Y, 8, col, 1.5, 3, 20)
			if r.Sound != nil && r.Sound.Enabled {
				r.Sound.PlayPickup()
			}
		} else if ev.Correct {
			// Big green burst on correct delivery
			r.SwarmParticles.Emit(ev.X, ev.Y, 20, color.RGBA{80, 255, 80, 255}, 2.0, 4, 30)
			if r.Sound != nil && r.Sound.Enabled {
				r.Sound.PlayDropOK()
			}
		} else {
			// Red burst on wrong delivery
			r.SwarmParticles.Emit(ev.X, ev.Y, 12, color.RGBA{255, 80, 80, 255}, 1.8, 3, 25)
			if r.Sound != nil && r.Sound.Enabled {
				r.Sound.PlayDropFail()
			}
		}
	}
	// Clear consumed events
	ss.DeliveryEvents = ss.DeliveryEvents[:0]
}

// drawSwarmParticles renders particles using fixed arena offsets (no camera).
func drawSwarmParticles(screen *ebiten.Image, ps *ParticleSystem, offX, offY float64) {
	for i := range ps.Particles {
		p := &ps.Particles[i]
		if !p.Active {
			continue
		}
		sx := float32(offX + p.X)
		sy := float32(offY + p.Y)
		alpha := float64(p.Life) / float64(p.MaxLife)
		col := p.Color
		col.A = uint8(float64(col.A) * alpha)
		size := float32(p.Size * alpha)
		if size < 0.5 {
			size = 0.5
		}
		vector.DrawFilledCircle(screen, sx, sy, size, col, false)
	}
}

// SwarmScreenToArena converts screen coordinates to arena world coordinates.
// Returns the arena position and whether the point is inside the arena.
func SwarmScreenToArena(sx, sy int) (float64, float64, bool) {
	arenaOffX := 415.0
	arenaOffY := 50.0
	wx := float64(sx) - arenaOffX
	wy := float64(sy) - arenaOffY
	inside := wx >= 0 && wx <= swarm.SwarmArenaSize && wy >= 0 && wy <= swarm.SwarmArenaSize
	return wx, wy, inside
}
