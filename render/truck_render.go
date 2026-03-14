package render

import (
	"image/color"
	"math"
	"swarmsim/engine/simulation"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawTruckMode renders all truck-mode-specific elements.
func (r *Renderer) DrawTruckMode(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	ts := s.TruckState
	if ts == nil {
		return
	}
	r.drawTruckBody(screen, ts, sw, sh)
	r.drawRamp(screen, ts, sw, sh)
	r.drawDepotZones(screen, ts, sw, sh)
	r.drawChargingStations(screen, ts, s.Tick, sw, sh)
	r.drawTruckPackages(screen, ts, sw, sh)
}

func (r *Renderer) drawTruckBody(screen *ebiten.Image, ts *simulation.TruckState, sw, sh int) {
	cam := r.Camera

	// Cabin
	cx, cy := cam.WorldToScreen(ts.Truck.CabinX, ts.Truck.CabinY, sw, sh)
	cw := ts.Truck.CabinW * cam.Zoom
	ch := ts.Truck.CabinH * cam.Zoom
	vector.DrawFilledRect(screen, float32(cx), float32(cy), float32(cw), float32(ch), ColorTruckCabin, false)
	vector.StrokeRect(screen, float32(cx), float32(cy), float32(cw), float32(ch), 2, color.RGBA{80, 80, 90, 255}, false)

	// Cabin windows
	winW := 20.0 * cam.Zoom
	winH := 30.0 * cam.Zoom
	winX := cx + cw*0.2
	winY1 := cy + ch*0.2
	winY2 := cy + ch*0.6
	vector.DrawFilledRect(screen, float32(winX), float32(winY1), float32(winW), float32(winH), color.RGBA{120, 160, 200, 200}, false)
	vector.DrawFilledRect(screen, float32(winX), float32(winY2), float32(winW), float32(winH), color.RGBA{120, 160, 200, 200}, false)

	// Cargo area
	cargoX, cargoY := cam.WorldToScreen(ts.Truck.CargoX, ts.Truck.CargoY, sw, sh)
	cargoW := ts.Truck.CargoW * cam.Zoom
	cargoH := ts.Truck.CargoH * cam.Zoom
	vector.DrawFilledRect(screen, float32(cargoX), float32(cargoY), float32(cargoW), float32(cargoH), ColorTruckCargo, false)

	// Cargo border — top, bottom, left (right side is opening)
	wallThick := float32(2)
	vector.DrawFilledRect(screen, float32(cargoX), float32(cargoY), float32(cargoW), wallThick, color.RGBA{100, 90, 70, 255}, false)                           // top
	vector.DrawFilledRect(screen, float32(cargoX), float32(cargoY+cargoH-float64(wallThick)), float32(cargoW), wallThick, color.RGBA{100, 90, 70, 255}, false) // bottom
	vector.DrawFilledRect(screen, float32(cargoX), float32(cargoY), wallThick, float32(cargoH), color.RGBA{100, 90, 70, 255}, false)                           // left

	// Opening marker (arrow pointing right)
	openX := cargoX + cargoW
	openY := cargoY + cargoH/2
	vector.StrokeLine(screen, float32(openX), float32(openY-15*cam.Zoom), float32(openX+10*cam.Zoom), float32(openY), 1.5, color.RGBA{200, 200, 100, 200}, false)
	vector.StrokeLine(screen, float32(openX), float32(openY+15*cam.Zoom), float32(openX+10*cam.Zoom), float32(openY), 1.5, color.RGBA{200, 200, 100, 200}, false)
}

func (r *Renderer) drawRamp(screen *ebiten.Image, ts *simulation.TruckState, sw, sh int) {
	cam := r.Camera
	rx, ry := cam.WorldToScreen(ts.Truck.RampX, ts.Truck.RampY, sw, sh)
	rw := ts.Truck.RampW * cam.Zoom
	rh := ts.Truck.RampH * cam.Zoom

	// Ramp fill
	vector.DrawFilledRect(screen, float32(rx), float32(ry), float32(rw), float32(rh), ColorTruckRamp, false)

	// Diagonal hatch lines to suggest slope
	step := 15.0 * cam.Zoom
	for offset := 0.0; offset < rw+rh; offset += step {
		x1 := rx + offset
		y1 := ry
		x2 := rx + offset - rh
		y2 := ry + rh

		// Clip to ramp bounds
		if x1 > rx+rw {
			y1 += (x1 - (rx + rw))
			x1 = rx + rw
		}
		if x2 < rx {
			y2 -= (rx - x2)
			x2 = rx
		}

		if y1 < ry+rh && y2 > ry {
			vector.StrokeLine(screen, float32(x1), float32(y1), float32(x2), float32(y2), 0.5, color.RGBA{120, 110, 90, 100}, false)
		}
	}

	// Ramp border
	vector.StrokeRect(screen, float32(rx), float32(ry), float32(rw), float32(rh), 1.5, ColorRampEdge, false)
}

func (r *Renderer) drawDepotZones(screen *ebiten.Image, ts *simulation.TruckState, sw, sh int) {
	cam := r.Camera
	zoneFills := [4]color.RGBA{ColorZoneA, ColorZoneB, ColorZoneC, ColorZoneD}
	zoneBorders := [4]color.RGBA{ColorZoneABorder, ColorZoneBBorder, ColorZoneCBorder, ColorZoneDBorder}

	for i, zone := range ts.Depot.Zones {
		zx, zy := cam.WorldToScreen(zone.X, zone.Y, sw, sh)
		zw := zone.W * cam.Zoom
		zh := zone.H * cam.Zoom

		// Fill
		vector.DrawFilledRect(screen, float32(zx), float32(zy), float32(zw), float32(zh), zoneFills[i], false)
		// Border
		vector.StrokeRect(screen, float32(zx), float32(zy), float32(zw), float32(zh), 2, zoneBorders[i], false)

		// Label
		labelX := zx + zw/2 - 3
		labelY := zy + zh/2 - 6
		ebitenutil.DebugPrintAt(screen, zone.Label, int(labelX), int(labelY))
	}
}

func (r *Renderer) drawChargingStations(screen *ebiten.Image, ts *simulation.TruckState, tick int, sw, sh int) {
	cam := r.Camera
	for _, c := range ts.Depot.Chargers {
		sx, sy := cam.WorldToScreen(c.X, c.Y, sw, sh)
		rad := c.Radius * cam.Zoom

		// Pulsing effect
		pulse := 1.0 + 0.1*math.Sin(float64(tick)*0.1)
		drawRad := float32(rad * pulse)

		// Outer glow
		glowCol := ColorChargingStation
		glowCol.A = 40
		vector.DrawFilledCircle(screen, float32(sx), float32(sy), drawRad*1.5, glowCol, false)

		// Main circle
		vector.DrawFilledCircle(screen, float32(sx), float32(sy), drawRad*0.3, ColorChargingStation, false)
		vector.StrokeCircle(screen, float32(sx), float32(sy), drawRad, 1.5, ColorChargingStation, false)

		// Lightning bolt symbol (simple chevron)
		boltSize := rad * 0.3
		vector.StrokeLine(screen, float32(sx-boltSize*0.3), float32(sy-boltSize), float32(sx+boltSize*0.2), float32(sy), 2, color.RGBA{255, 255, 200, 255}, false)
		vector.StrokeLine(screen, float32(sx+boltSize*0.2), float32(sy), float32(sx-boltSize*0.3), float32(sy+boltSize), 2, color.RGBA{255, 255, 200, 255}, false)
	}
}

func (r *Renderer) drawTruckPackages(screen *ebiten.Image, ts *simulation.TruckState, sw, sh int) {
	cam := r.Camera

	for _, pkg := range ts.Packages {
		if pkg.State == simulation.PkgDelivered {
			continue // Skip delivered packages
		}

		sx, sy := cam.WorldToScreen(pkg.X, pkg.Y, sw, sh)
		pw := pkg.Def.Width * cam.Zoom
		ph := pkg.Def.Height * cam.Zoom

		// Package color
		col := color.RGBA{pkg.Def.ColorR, pkg.Def.ColorG, pkg.Def.ColorB, 255}

		// Dim blocked packages
		if pkg.State == simulation.PkgInTruck && !pkg.IsAccessible(ts.Packages, ts.Truck.CargoRight()) {
			col.A = 128
		}

		// Draw filled rectangle centered on position
		vector.DrawFilledRect(screen, float32(sx-pw/2), float32(sy-ph/2), float32(pw), float32(ph), col, false)
		vector.StrokeRect(screen, float32(sx-pw/2), float32(sy-ph/2), float32(pw), float32(ph), 1, color.RGBA{col.R / 2, col.G / 2, col.B / 2, col.A}, false)

		// Fragile packages: red border + "!" icon
		if pkg.Def.Type == simulation.PkgFragile {
			vector.StrokeRect(screen, float32(sx-pw/2-2), float32(sy-ph/2-2), float32(pw+4), float32(ph+4), 1.5, color.RGBA{255, 0, 0, 200}, false)
			ebitenutil.DebugPrintAt(screen, "!", int(sx-3), int(sy-6))
		}

		// Lifting animation: slight bounce effect
		if pkg.State == simulation.PkgLifting {
			bounceOffset := math.Sin(float64(pkg.LiftTick)*0.5) * 3 * cam.Zoom
			liftCol := col
			liftCol.A = 180
			vector.DrawFilledRect(screen, float32(sx-pw/2), float32(sy-ph/2-bounceOffset), float32(pw), float32(ph), liftCol, false)
		}
	}
}
