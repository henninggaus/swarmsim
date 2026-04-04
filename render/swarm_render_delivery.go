package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
				vector.StrokeLine(screen, sx, sy-8, sx, sy+2, 2, ColorWhiteFaded, false)
				vector.StrokeLine(screen, sx-3, sy-5, sx, sy-8, 1.5, ColorWhiteFaded, false)
				vector.StrokeLine(screen, sx+3, sy-5, sx, sy-8, 1.5, ColorWhiteFaded, false)
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
// Labels shift left/right when two stations are closer than 100px to avoid overlap.
func drawStationLabels(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	colorLetters := [5]string{"", "R", "B", "Y", "G"}

	// Pre-compute label X offsets to avoid overlap for nearby stations
	labelOffsetX := make([]int, len(ss.Stations))
	for i := range ss.Stations {
		for j := i + 1; j < len(ss.Stations); j++ {
			dx := ss.Stations[i].X - ss.Stations[j].X
			dy := ss.Stations[i].Y - ss.Stations[j].Y
			if math.Abs(dx) < 100 && math.Abs(dy) < 60 {
				if dx <= 0 { // i is left of j
					labelOffsetX[i] -= 20
					labelOffsetX[j] += 20
				} else {
					labelOffsetX[i] += 20
					labelOffsetX[j] -= 20
				}
			}
		}
	}

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
		// Label above station (with horizontal offset for nearby stations)
		lx := sx - 6 + labelOffsetX[si]
		printColoredAt(screen, label, lx, sy-40, col)
		// Subtle type hint below label
		typeHint := "Drop"
		if st.IsPickup {
			typeHint = "Pick"
		}
		printColoredAt(screen, typeHint, lx-2, sy-30, color.RGBA{col.R, col.G, col.B, 100})

		if st.IsPickup {
			// Respawn timer bar below pickup station (15px gap from edge: radius 25 + 15 = 40)
			if !st.HasPackage && st.RespawnIn > 0 {
				barW := 30
				barH := 4
				bx := float32(sx - barW/2)
				by := float32(sy + 40)
				// Background
				vector.DrawFilledRect(screen, bx, by, float32(barW), float32(barH),
					color.RGBA{40, 40, 40, 180}, false)
				// Progress fill
				progress := 1.0 - float64(st.RespawnIn)/100.0
				fillW := float32(float64(barW) * progress)
				vector.DrawFilledRect(screen, bx, by, fillW, float32(barH), col, false)
			}
		} else {
			// Delivery counter below dropoff (15px gap from edge: radius 25 + 15 = 40)
			if st.DeliverCount > 0 {
				countStr := fmt.Sprintf("%d", st.DeliverCount)
				printColoredAt(screen, countStr, sx-len(countStr)*3, sy+40,
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

// drawSwarmRamp renders the loading ramp area on the arena.
func drawSwarmRamp(screen *ebiten.Image, ss *swarm.SwarmState) {
	ts := ss.TruckState
	rx := float32(ts.RampX)
	ry := float32(ts.RampY)
	rw := float32(ts.RampW)
	rh := float32(ts.RampH)

	// Semi-transparent background (brighter)
	vector.DrawFilledRect(screen, rx, ry, rw, rh, color.RGBA{80, 70, 30, 100}, false)

	// Diagonal hatching lines
	hatchStep := float32(20)
	for d := float32(0); d < rw+rh; d += hatchStep {
		x1 := rx + d
		y1 := ry
		x2 := rx
		y2 := ry + d
		if x1 > rx+rw {
			y1 += x1 - (rx + rw)
			x1 = rx + rw
		}
		if y2 > ry+rh {
			x2 += y2 - (ry + rh)
			y2 = ry + rh
		}
		vector.StrokeLine(screen, x1, y1, x2, y2, 1, color.RGBA{140, 120, 50, 80}, false)
	}

	// Bright border (yellow-orange, thick)
	borderCol := color.RGBA{255, 200, 50, 220}
	vector.StrokeRect(screen, rx, ry, rw, rh, 3, borderCol, false)

	// Pulsing inner border
	innerCol := color.RGBA{255, 200, 50, 80}
	vector.StrokeRect(screen, rx+4, ry+4, rw-8, rh-8, 1, innerCol, false)

	// Entrance arrow on the right side (pointing left into ramp)
	arrowX := rx + rw - 10
	arrowCY := ry + rh/2
	arrowCol := color.RGBA{255, 220, 80, 200}
	// Arrow body (horizontal line)
	vector.StrokeLine(screen, arrowX, arrowCY, arrowX-30, arrowCY, 3, arrowCol, false)
	// Arrowhead (chevron pointing left)
	vector.StrokeLine(screen, arrowX-30, arrowCY, arrowX-20, arrowCY-10, 2.5, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY, arrowX-20, arrowCY+10, 2.5, arrowCol, false)

	// Second arrow below
	arrowCY2 := arrowCY + 40
	vector.StrokeLine(screen, arrowX, arrowCY2, arrowX-30, arrowCY2, 3, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY2, arrowX-20, arrowCY2-10, 2.5, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY2, arrowX-20, arrowCY2+10, 2.5, arrowCol, false)

	// Second arrow above
	arrowCY3 := arrowCY - 40
	vector.StrokeLine(screen, arrowX, arrowCY3, arrowX-30, arrowCY3, 3, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY3, arrowX-20, arrowCY3-10, 2.5, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY3, arrowX-20, arrowCY3+10, 2.5, arrowCol, false)

	// "RAMP" label (larger, brighter)
	label := "RAMP"
	lx := int(rx+rw/2) - runeLen(label)*charW/2
	ly := int(ry + 8)
	printColoredAt(screen, label, lx, ly, color.RGBA{255, 220, 80, 255})

	// Truck count / status below label
	if ts.CurrentTruck != nil {
		status := ""
		switch ts.CurrentTruck.Phase {
		case swarm.TruckDrivingIn:
			status = "Truck incoming..."
		case swarm.TruckParked:
			remaining := 0
			for _, p := range ts.CurrentTruck.Packages {
				if !p.PickedUp {
					remaining++
				}
			}
			status = fmt.Sprintf("Pkgs: %d left", remaining)
		case swarm.TruckComplete:
			status = "All picked up!"
		case swarm.TruckDrivingOut:
			status = "Truck leaving..."
		case swarm.TruckWaiting:
			status = "Next truck soon"
		case swarm.TruckRoundDone:
			status = "Round complete!"
		}
		if status != "" {
			printColoredAt(screen, status, int(rx+5), int(ry+rh-18), color.RGBA{200, 180, 80, 200})
		}
	}
}

// drawSwarmTruckVehicle renders the truck body with packages.
func drawSwarmTruckVehicle(screen *ebiten.Image, ss *swarm.SwarmState) {
	t := ss.TruckState.CurrentTruck

	tx := float32(t.X)
	ty := float32(t.Y)

	// Truck body dimensions
	bodyW := float32(80)
	bodyH := float32(40)
	cabinW := float32(18)

	// Cabin (front, darker gray)
	vector.DrawFilledRect(screen, tx, ty, cabinW, bodyH, color.RGBA{70, 70, 80, 255}, false)
	vector.StrokeRect(screen, tx, ty, cabinW, bodyH, 1, color.RGBA{120, 120, 130, 200}, false)
	// Windshield
	vector.DrawFilledRect(screen, tx+2, ty+4, cabinW-4, 12, color.RGBA{140, 180, 220, 200}, false)

	// Cargo area (light gray)
	cargoX := tx + cabinW
	vector.DrawFilledRect(screen, cargoX, ty, bodyW-cabinW, bodyH, color.RGBA{160, 160, 150, 255}, false)
	vector.StrokeRect(screen, cargoX, ty, bodyW-cabinW, bodyH, 1, color.RGBA{120, 120, 130, 200}, false)

	// Wheels
	wheelR := float32(5)
	vector.DrawFilledCircle(screen, tx+12, ty+bodyH+2, wheelR, color.RGBA{40, 40, 40, 255}, false)
	vector.DrawFilledCircle(screen, tx+bodyW-12, ty+bodyH+2, wheelR, color.RGBA{40, 40, 40, 255}, false)

	// Draw packages on cargo
	for _, pkg := range t.Packages {
		if pkg.PickedUp {
			continue
		}
		px := cargoX + float32(pkg.RelX)
		py := ty + float32(pkg.RelY)
		col := deliveryColor(pkg.Color)
		vector.DrawFilledRect(screen, px, py, 8, 8, col, false)
		vector.StrokeRect(screen, px, py, 8, 8, 1, color.RGBA{255, 255, 255, 150}, false)
	}

	// Phase overlay text
	switch t.Phase {
	case swarm.TruckComplete:
		printColoredAt(screen, "COMPLETE", int(tx+10), int(ty-14), color.RGBA{0, 255, 100, 255})
	case swarm.TruckRoundDone:
		// Big overlay handled elsewhere (HUD)
	}
}
