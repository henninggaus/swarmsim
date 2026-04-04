package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/factory"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Factory mode colors -- industrial palette
var (
	// Ground
	colorYard       = color.RGBA{140, 145, 135, 255} // asphalt gray
	colorHallFloor  = color.RGBA{65, 70, 80, 255}    // dark concrete
	colorRoad       = color.RGBA{90, 90, 85, 255}    // road surface
	colorRoadLine   = color.RGBA{230, 210, 50, 255}  // yellow road marking
	colorGrass      = color.RGBA{60, 90, 50, 255}    // grass edges

	// Structures
	colorWall       = color.RGBA{45, 50, 60, 255}
	colorWallEdge   = color.RGBA{70, 75, 90, 255}
	colorDoorOpen   = color.RGBA{40, 180, 60, 255}
	colorDoorClosed = color.RGBA{180, 40, 40, 255}
	colorMachine    = color.RGBA{60, 100, 160, 255}
	colorMachineEdge= color.RGBA{90, 140, 200, 255}
	colorCharger    = color.RGBA{200, 180, 30, 255}
	colorChargerBolt= color.RGBA{255, 240, 60, 255}
	colorWorkshop   = color.RGBA{190, 110, 30, 255}
	colorDock       = color.RGBA{120, 85, 50, 255}
	colorDockEdge   = color.RGBA{160, 120, 80, 255}
	colorStorage    = color.RGBA{150, 130, 90, 255}
	colorStorEdge   = color.RGBA{180, 160, 120, 255}

	// Bots
	colorBot        = color.RGBA{0, 200, 220, 255}
	colorBotDir     = color.RGBA{255, 255, 255, 180}

	// Trucks
	colorTruckBody   = color.RGBA{55, 60, 70, 255}
	colorTruckCab    = color.RGBA{70, 80, 100, 255}
	colorTruckEdge   = color.RGBA{90, 100, 120, 255}
	colorTruckCargo  = color.RGBA{45, 50, 55, 255}
	colorHeadlight   = color.RGBA{255, 255, 220, 240}

	// UI
	colorFactBg     = color.RGBA{25, 28, 35, 255}
	colorHUDText    = color.RGBA{210, 215, 230, 240}
	colorHUDDim     = color.RGBA{140, 150, 170, 200}
	colorHUDPanel   = color.RGBA{15, 18, 28, 220}
)

// Dashboard height at bottom of screen
const dashboardH = 150

// DrawFactoryMode renders the entire factory mode screen.
func DrawFactoryMode(screen *ebiten.Image, fs *factory.FactoryState) {
	if fs == nil {
		return
	}

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// Camera transform helpers
	camX := fs.CamX
	camY := fs.CamY
	zoom := fs.CamZoom

	// World to screen coordinate conversion
	wx := func(worldX float64) float32 {
		return float32((worldX-camX)*zoom + float64(sw)/2)
	}
	wy := func(worldY float64) float32 {
		return float32((worldY-camY)*zoom + float64(sh)/2)
	}
	ws := func(size float64) float32 { // world size to screen size
		return float32(size * zoom)
	}

	// Visible bounds (for culling)
	margin := 50.0
	visMinX := camX - float64(sw)/2/zoom - margin
	visMaxX := camX + float64(sw)/2/zoom + margin
	visMinY := camY - float64(sh)/2/zoom - margin
	visMaxY := camY + float64(sh)/2/zoom + margin

	tick := fs.Tick

	// --- Background ---
	screen.Fill(colorFactBg)

	// --- Grass border around everything ---
	vector.DrawFilledRect(screen, wx(0), wy(0), ws(factory.WorldW), ws(factory.WorldH), colorGrass, false)

	// --- Yard (asphalt) ---
	vector.DrawFilledRect(screen, wx(0), wy(0), ws(factory.WorldW), ws(factory.YardH), colorYard, false)

	// --- Parking lot markings in the yard ---
	parkingY := 80.0
	parkingSpacing := 60.0
	parkingCol := color.RGBA{180, 180, 170, 50}
	for px := 100.0; px < factory.WorldW-200; px += parkingSpacing {
		if px < visMinX || px > visMaxX {
			continue
		}
		vector.StrokeLine(screen, wx(px), wy(parkingY), wx(px), wy(parkingY+90), 1, parkingCol, false)
	}
	vector.StrokeLine(screen, wx(100), wy(parkingY), wx(factory.WorldW-200), wy(parkingY), 1, parkingCol, false)
	vector.StrokeLine(screen, wx(100), wy(parkingY+90), wx(factory.WorldW-200), wy(parkingY+90), 1, parkingCol, false)

	// --- Road surface ---
	roadTop := factory.RoadY - factory.RoadH/2
	vector.DrawFilledRect(screen, wx(0), wy(roadTop), ws(factory.WorldW), ws(factory.RoadH), colorRoad, false)

	// Road edge lines (white)
	roadEdgeCol := color.RGBA{200, 200, 200, 80}
	vector.StrokeLine(screen, wx(0), wy(roadTop), wx(factory.WorldW), wy(roadTop), 2, roadEdgeCol, false)
	vector.StrokeLine(screen, wx(0), wy(roadTop+factory.RoadH), wx(factory.WorldW), wy(roadTop+factory.RoadH), 2, roadEdgeCol, false)

	// Road markings (dashed center line)
	dashW := 50.0
	gapW := 30.0
	for x := 0.0; x < factory.WorldW; x += dashW + gapW {
		if x > visMaxX || x+dashW < visMinX {
			continue
		}
		endX := x + dashW
		if endX > factory.WorldW {
			endX = factory.WorldW
		}
		vector.DrawFilledRect(screen, wx(x), wy(factory.RoadY-2), ws(endX-x), ws(4), colorRoadLine, false)
	}

	// --- Truck Rendering ---
	drawFactoryTrucks(screen, fs, wx, wy, ws, visMinX, visMaxX, tick)

	// --- Hall floor ---
	vector.DrawFilledRect(screen, wx(factory.HallX), wy(factory.HallY),
		ws(factory.HallW), ws(factory.HallH), colorHallFloor, false)

	// --- Floor stains/marks ---
	stainCol := color.RGBA{55, 60, 70, 255}
	stainSeeds := [12][2]float64{
		{380, 750}, {600, 900}, {1100, 700}, {1500, 850},
		{900, 1200}, {1300, 1100}, {700, 1400}, {2000, 800},
		{2200, 1000}, {1800, 1300}, {500, 1600}, {2400, 750},
	}
	for _, s := range stainSeeds {
		sx, sy := s[0], s[1]
		if sx < visMinX || sx > visMaxX || sy < visMinY || sy > visMaxY {
			continue
		}
		r := 3.0 + float64(int(sx*7+sy*13)%5)
		vector.DrawFilledCircle(screen, wx(sx), wy(sy), ws(r), stainCol, false)
	}

	// Zone floor tints (subtle color coding for each area)
	zoneAlpha := uint8(25)
	vector.DrawFilledRect(screen, wx(factory.ZoneReceivingX), wy(factory.HallY),
		ws(factory.ZoneReceivingW), ws(factory.HallH), color.RGBA{40, 120, 40, zoneAlpha}, false)
	vector.DrawFilledRect(screen, wx(factory.ZoneStorageX), wy(factory.HallY),
		ws(factory.ZoneStorageW), ws(factory.HallH), color.RGBA{40, 80, 160, zoneAlpha}, false)
	vector.DrawFilledRect(screen, wx(factory.ZoneProductionX), wy(factory.HallY),
		ws(factory.ZoneProductionW), ws(factory.HallH), color.RGBA{160, 100, 30, zoneAlpha}, false)
	vector.DrawFilledRect(screen, wx(factory.ZoneShippingX), wy(factory.HallY),
		ws(factory.ZoneShippingW), ws(factory.HallH), color.RGBA{150, 40, 40, zoneAlpha}, false)

	// Main aisle (horizontal corridor)
	aisleCol := color.RGBA{85, 90, 100, 255}
	vector.DrawFilledRect(screen, wx(factory.HallX), wy(factory.AisleY),
		ws(factory.HallW), ws(factory.AisleH), aisleCol, false)
	safetyCol := color.RGBA{200, 180, 40, 120}
	vector.StrokeLine(screen, wx(factory.HallX), wy(factory.AisleY),
		wx(factory.HallX+factory.HallW), wy(factory.AisleY), 2, safetyCol, false)
	vector.StrokeLine(screen, wx(factory.HallX), wy(factory.AisleY+factory.AisleH),
		wx(factory.HallX+factory.HallW), wy(factory.AisleY+factory.AisleH), 2, safetyCol, false)

	// --- [5] Animated Conveyor Belts Along Main Aisle ---
	drawConveyorBelts(screen, fs, wx, wy, ws, visMinX, visMaxX, tick)

	// --- Aisle flow particles ---
	drawAisleFlowParticles(screen, fs, wx, wy, ws, visMinX, visMaxX, tick)

	// Material flow arrows along aisle
	arrowCol := color.RGBA{180, 200, 220, 40}
	arrowY := factory.AisleY + factory.AisleH/2
	for ax := factory.HallX + 50.0; ax < factory.HallX+factory.HallW-100; ax += 150.0 {
		if ax < visMinX || ax > visMaxX {
			continue
		}
		vector.StrokeLine(screen, wx(ax), wy(arrowY), wx(ax+60), wy(arrowY), 2, arrowCol, false)
		vector.StrokeLine(screen, wx(ax+60), wy(arrowY), wx(ax+48), wy(arrowY-8), 2, arrowCol, false)
		vector.StrokeLine(screen, wx(ax+60), wy(arrowY), wx(ax+48), wy(arrowY+8), 2, arrowCol, false)
	}

	// --- [9] Animated Factory Signage (pulsing zone labels) ---
	drawAnimatedZoneLabels(screen, fs, wx, wy, tick)

	// --- [NEW] Lane markings (thin dashed lines on hall floor) ---
	laneYs := []float64{factory.HallY + 100, factory.AisleY, factory.AisleY + factory.AisleH, factory.HallY + factory.HallH - 100}
	laneCol := color.RGBA{100, 110, 130, 40}
	laneDash := 20.0
	laneGap := 15.0
	for _, ly := range laneYs {
		if ly < visMinY || ly > visMaxY {
			continue
		}
		for lx := factory.HallX + 20.0; lx < factory.HallX+factory.HallW-20; lx += laneDash + laneGap {
			if lx > visMaxX || lx+laneDash < visMinX {
				continue
			}
			endX := lx + laneDash
			if endX > factory.HallX+factory.HallW-20 {
				endX = factory.HallX + factory.HallW - 20
			}
			vector.StrokeLine(screen, wx(lx), wy(ly), wx(endX), wy(ly), 1, laneCol, false)
		}
	}

	// --- [NEW] Parking zone floor markings (diagonal lines) ---
	if fs.ParkingZones != nil {
		parkCol := color.RGBA{80, 90, 110, 50}
		for _, pz := range fs.ParkingZones {
			pzx := pz[0] - 60
			pzy := pz[1] - 40
			pzw := 120.0
			pzh := 80.0
			if pzx > visMaxX || pzx+pzw < visMinX || pzy > visMaxY || pzy+pzh < visMinY {
				continue
			}
			// Diagonal hatching
			for d := 0.0; d < pzw+pzh; d += 12 {
				x1 := pzx + d
				y1 := pzy
				x2 := pzx
				y2 := pzy + d
				if x1 > pzx+pzw {
					y1 += x1 - (pzx + pzw)
					x1 = pzx + pzw
				}
				if y2 > pzy+pzh {
					x2 += y2 - (pzy + pzh)
					y2 = pzy + pzh
				}
				vector.StrokeLine(screen, wx(x1), wy(y1), wx(x2), wy(y2), 1, parkCol, false)
			}
			// Border
			vector.StrokeRect(screen, wx(pzx), wy(pzy), ws(pzw), ws(pzh), 1, color.RGBA{90, 100, 120, 80}, false)
			// Label
			plabel := locale.T("factory.parking")
			plx := int(wx(pz[0])) - runeLen(plabel)*charW/2
			ply := int(wy(pzy - 10))
			printColoredAt(screen, plabel, plx, ply, color.RGBA{90, 100, 120, 120})
		}
	}

	// --- Feature 9: Tire tracks (subtle dark overlay from heatmap data) ---
	if fs.HeatmapW > 0 && fs.HeatmapH > 0 {
		cellW := factory.WorldW / float64(fs.HeatmapW)
		cellH := factory.WorldH / float64(fs.HeatmapH)
		maxVal := 1
		for _, v := range fs.HeatmapGrid {
			if v > maxVal {
				maxVal = v
			}
		}
		for gy := 0; gy < fs.HeatmapH; gy++ {
			for gx := 0; gx < fs.HeatmapW; gx++ {
				val := fs.HeatmapGrid[gy*fs.HeatmapW+gx]
				if val < maxVal/10 {
					continue // skip low-traffic cells
				}
				intensity := float64(val) / float64(maxVal)
				if intensity > 1 {
					intensity = 1
				}
				alpha := uint8(intensity * 25) // very subtle, max alpha 25
				wx0 := float64(gx) * cellW
				wy0 := float64(gy) * cellH
				// Only draw inside the hall
				if wy0 < factory.HallY || wy0 > factory.HallY+factory.HallH {
					continue
				}
				if wx0+cellW < visMinX || wx0 > visMaxX || wy0+cellH < visMinY || wy0 > visMaxY {
					continue
				}
				vector.DrawFilledRect(screen, wx(wx0), wy(wy0), ws(cellW), ws(cellH),
					color.RGBA{20, 15, 10, alpha}, false)
			}
		}
	}

	// Floor grid pattern inside hall
	gridSize := 80.0
	gridCol := color.RGBA{75, 80, 90, 255}
	for gx := factory.HallX; gx <= factory.HallX+factory.HallW; gx += gridSize {
		if gx < visMinX || gx > visMaxX {
			continue
		}
		vector.StrokeLine(screen, wx(gx), wy(factory.HallY), wx(gx), wy(factory.HallY+factory.HallH), 1, gridCol, false)
	}
	for gy := factory.HallY; gy <= factory.HallY+factory.HallH; gy += gridSize {
		if gy < visMinY || gy > visMaxY {
			continue
		}
		vector.StrokeLine(screen, wx(factory.HallX), wy(gy), wx(factory.HallX+factory.HallW), wy(gy), 1, gridCol, false)
	}

	// --- Walls ---
	for _, wall := range fs.Walls {
		if wall.X+wall.W < visMinX || wall.X > visMaxX || wall.Y+wall.H < visMinY || wall.Y > visMaxY {
			continue
		}
		vector.DrawFilledRect(screen, wx(wall.X), wy(wall.Y), ws(wall.W), ws(wall.H), colorWall, false)
		vector.StrokeRect(screen, wx(wall.X), wy(wall.Y), ws(wall.W), ws(wall.H), 1, colorWallEdge, false)
	}

	// --- Danger stripes near gates ---
	dangerCol1 := color.RGBA{200, 180, 40, 80}
	dangerCol2 := color.RGBA{40, 40, 40, 80}
	for _, door := range fs.Doors {
		stripeW := 6.0
		for s := 0; s < 4; s++ {
			col := dangerCol1
			if s%2 == 1 {
				col = dangerCol2
			}
			vector.DrawFilledRect(screen, wx(door.X+float64(s)*stripeW), wy(door.Y-8),
				ws(stripeW), ws(6), col, false)
			vector.DrawFilledRect(screen, wx(door.X+float64(s)*stripeW), wy(door.Y+factory.WallThick+2),
				ws(stripeW), ws(6), col, false)
		}
	}

	// --- Doors ---
	for _, door := range fs.Doors {
		col := colorDoorClosed
		if door.Open {
			col = colorDoorOpen
		}
		dh := factory.WallThick
		vector.DrawFilledRect(screen, wx(door.X), wy(door.Y), ws(door.W), ws(dh), col, false)
		if door.Open {
			cx := door.X + door.W/2
			cy := door.Y + dh/2
			arrowSize := 8.0
			vector.StrokeLine(screen, wx(cx), wy(cy-arrowSize), wx(cx), wy(cy+arrowSize), 2, color.RGBA{100, 255, 120, 200}, false)
		}
	}

	// --- Storage areas ---
	for i, st := range fs.Storage {
		vector.DrawFilledRect(screen, wx(st.X), wy(st.Y), ws(st.W), ws(st.H), colorStorage, false)
		vector.StrokeRect(screen, wx(st.X), wy(st.Y), ws(st.W), ws(st.H), 2, colorStorEdge, false)
		shelfGap := 20.0
		for sy := st.Y + shelfGap; sy < st.Y+st.H-5; sy += shelfGap {
			vector.StrokeLine(screen, wx(st.X+5), wy(sy), wx(st.X+st.W-5), wy(sy), 1, colorStorEdge, false)
		}
		if st.MaxParts > 0 && len(st.Parts) > 0 {
			fillFrac := float64(len(st.Parts)) / float64(st.MaxParts)
			fillCol := color.RGBA{80, 200, 80, 60}
			if fillFrac > 0.8 {
				fillCol = color.RGBA{200, 80, 80, 60}
			} else if fillFrac > 0.5 {
				fillCol = color.RGBA{200, 200, 80, 60}
			}
			vector.DrawFilledRect(screen, wx(st.X), wy(st.Y+st.H*(1-fillFrac)),
				ws(st.W), ws(st.H*fillFrac), fillCol, false)
		}
		// FIFO slot visualization: colored dots for each stored part
		if len(st.Slots) > 0 && zoom > 0.4 {
			cols := int(st.W / 10)
			if cols < 1 {
				cols = 1
			}
			for si, slot := range st.Slots {
				col := si % cols
				row := si / cols
				slotX := st.X + 5 + float64(col)*10
				slotY := st.Y + 5 + float64(row)*10
				if slotY > st.Y+st.H-5 {
					break
				}
				slotCol := partColorToRGBA(slot.PartColor)
				vector.DrawFilledCircle(screen, wx(slotX), wy(slotY), ws(3), slotCol, false)
			}
		}
		storageLabels := []string{
			locale.T("factory.storage.receiving"), locale.T("factory.storage.qc_inbound"),
			locale.T("factory.storage.shelf1"), locale.T("factory.storage.shelf2"), locale.T("factory.storage.shelf3"),
			locale.T("factory.storage.shelf4"), locale.T("factory.storage.shelf5"), locale.T("factory.storage.shelf6"),
			locale.T("factory.storage.shipping"), locale.T("factory.storage.packing"), locale.T("factory.storage.buffer"),
		}
		if i < len(storageLabels) {
			label := storageLabels[i]
			lx := int(wx(st.X+st.W/2)) - runeLen(label)*charW/2
			ly := int(wy(st.Y + st.H + 5))
			printColoredAt(screen, label, lx, ly, colorStorEdge)
		}
	}

	// --- [NEW-6] Floor Safety Zone Markings ---
	drawFloorSafetyZones(screen, fs, wx, wy, ws)

	// --- Machines ---
	drawFactoryMachines(screen, fs, wx, wy, ws, tick)

	// --- [NEW-3] Production Chain Flow Visualization ---
	drawProductionChainFlow(screen, fs, wx, wy, ws, tick)

	// --- Chargers ---
	drawFactoryChargers(screen, fs, wx, wy, ws, tick)

	// --- [NEW-2] Charging Lightning Bolts ---
	drawChargingLightning(screen, fs, wx, wy, ws, tick)

	// --- Workshop ---
	wks := fs.Workshop
	vector.DrawFilledRect(screen, wx(wks.X), wy(wks.Y), ws(wks.W), ws(wks.H), colorWorkshop, false)
	vector.StrokeRect(screen, wx(wks.X), wy(wks.Y), ws(wks.W), ws(wks.H), 2, color.RGBA{240, 160, 60, 255}, false)
	wcx := wks.X + wks.W/2
	wcy := wks.Y + wks.H/2
	wr := wks.W * 0.25
	vector.StrokeLine(screen, wx(wcx-wr), wy(wcy), wx(wcx+wr), wy(wcy), 2, color.RGBA{240, 160, 60, 255}, false)
	vector.StrokeLine(screen, wx(wcx), wy(wcy-wr), wx(wcx), wy(wcy+wr), 2, color.RGBA{240, 160, 60, 255}, false)
	if wks.CurrentBot >= 0 {
		pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.15)
		sparkAlpha := uint8(80 + int(pulse*120))
		vector.StrokeCircle(screen, wx(wcx), wy(wcy), ws(wks.W*0.45), 2, color.RGBA{255, 180, 60, sparkAlpha}, false)
	}
	wlabel := locale.T("factory.workshop")
	wlx := int(wx(wcx)) - runeLen(wlabel)*charW/2
	wly := int(wy(wks.Y + wks.H + 5))
	printColoredAt(screen, wlabel, wlx, wly, color.RGBA{240, 160, 60, 200})

	// --- Docks ---
	drawFactoryDocks(screen, fs, wx, wy, ws, visMinX, visMaxX, tick)

	// --- [NEW-5] Loading Dock Cargo Door Animations ---
	drawDockCargoAnimation(screen, fs, wx, wy, ws, tick)

	// --- Charger ambient glow ---
	for _, ch := range fs.Chargers {
		cx := ch.X + ch.W/2
		cy := ch.Y + ch.H/2
		glowR := ch.W * 1.5
		for ring := 0; ring < 5; ring++ {
			r := glowR * (1.0 - float64(ring)*0.18)
			a := uint8(10 + ring*4)
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(r), 1, color.RGBA{255, 240, 60, a}, false)
		}
	}

	// --- Machine conveyor belt chevron pattern ---
	for _, m := range fs.Machines {
		if !m.Active {
			continue
		}
		chevY := m.Y + m.H - 6
		chevCount := int(m.W / 12)
		offset := float64(tick%20) * 0.6
		for c := 0; c < chevCount; c++ {
			cx := m.X + 6 + float64(c)*12 + math.Mod(offset, 12)
			if cx > m.X+m.W-4 {
				continue
			}
			chevCol := color.RGBA{120, 170, 220, 80}
			vector.StrokeLine(screen, wx(cx), wy(chevY+3), wx(cx+3), wy(chevY), 1, chevCol, false)
			vector.StrokeLine(screen, wx(cx+3), wy(chevY), wx(cx+6), wy(chevY+3), 1, chevCol, false)
		}
	}

	// --- Door open/close animation ---
	for _, door := range fs.Doors {
		if door.Open {
			dh := factory.WallThick
			halfW := door.W / 2
			slideOff := halfW * 0.8
			vector.DrawFilledRect(screen, wx(door.X-slideOff), wy(door.Y), ws(halfW), ws(dh),
				color.RGBA{40, 180, 60, 150}, false)
			vector.DrawFilledRect(screen, wx(door.X+halfW+slideOff*0.2), wy(door.Y), ws(halfW), ws(dh),
				color.RGBA{40, 180, 60, 150}, false)
			if door.CloseTimer > 55 {
				flashA := uint8(float64(door.CloseTimer-55) * 25)
				vector.DrawFilledRect(screen, wx(door.X-10), wy(door.Y-10), ws(door.W+20), ws(dh+20),
					color.RGBA{60, 255, 90, flashA}, false)
			}
		}
	}

	// --- Truck exhaust particles ---
	drawTruckExhaust(screen, fs, wx, wy, ws, tick)

	// --- [3] Heatmap Overlay ---
	if fs.ShowHeatmap {
		drawHeatmapOverlay(screen, fs, wx, wy, ws)
	}

	// --- [10] Selected Bot Path Trail ---
	if fs.SelectedBot >= 0 && fs.SelectedBot < len(fs.Bots) {
		drawSelectedBotPath(screen, fs, wx, wy, ws)
	}

	// --- [NEW-8] Ambient Factory Noise Visualization ---
	drawAmbientNoiseViz(screen, fs, wx, wy, ws, tick)

	// --- Bots (with zoom-dependent detail) ---
	drawFactoryBots(screen, fs, wx, wy, ws, visMinX, visMaxX, visMinY, visMaxY, tick)

	// --- [8] Spark Effects (malfunction) ---
	drawSparkEffects(screen, fs, wx, wy, ws)

	// --- [8] Machine Finish FX (big pulse rings) ---
	drawMachineFinishFX(screen, fs, wx, wy, ws)

	// --- Pulse effects (pickup/delivery rings) ---
	drawPulseEffects(screen, fs, wx, wy, ws)

	// --- [2] Weather: Rain in yard only ---
	if fs.Weather == factory.WeatherRain {
		drawRainEffect(screen, fs, wx, wy, ws, visMinX, visMaxX, tick, sw, sh)
	}

	// --- Emergency Red Flashing Overlay ---
	if fs.Emergency {
		flashAlpha := uint8(40 + int(30*math.Sin(float64(tick)*0.2)))
		vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh),
			color.RGBA{255, 0, 0, flashAlpha}, false)
		// Emergency text
		eLabel := locale.T("factory.emergency")
		eLx := sw/2 - runeLen(eLabel)*charW/2
		eLy := 30
		printColoredAt(screen, eLabel, eLx, eLy, color.RGBA{255, 80, 80, 255})
	}

	// --- [1] Day/Night Cycle Overlay ---
	drawFactoryDayNightOverlay(screen, fs, sw, sh)

	// --- [8] Truck Arrival Flash ---
	if fs.TruckArriveFlash.Tick < 20 {
		drawTruckArriveFlash(screen, fs, sw)
	}

	// --- [12] Edge-of-Screen Indicators for selected bot ---
	if fs.SelectedBot >= 0 && fs.SelectedBot < len(fs.Bots) {
		drawOffScreenIndicator(screen, fs, wx, wy, sw, sh)
	}

	// --- [NEW-4] Bot Task Preview on Hover ---
	drawBotHoverTaskPreview(screen, fs, wx, wy, tick)

	// --- Hover tooltip ---
	if fs.HoverBot >= 0 && fs.HoverBot < len(fs.Bots) {
		drawBotTooltip(screen, fs, fs.HoverBot, sw, sh)
	}

	// --- [7] Real-Time Production Dashboard (bottom 150px) ---
	drawProductionDashboard(screen, fs, sw, sh, tick)

	// --- HUD (top-left) ---
	drawFactoryHUD(screen, fs, sw, sh, tick)

	// --- Feature 11: Active Event Banner ---
	drawEventBanner(screen, fs, sw, tick)

	// --- Feature 12: Bot Productivity Podium ---
	drawBotPodium(screen, fs, sw, sh)

	// --- [NEW] Order Panel (left side, below HUD) ---
	drawOrderPanel(screen, fs, sw, sh, tick)

	// --- [NEW] Alert Ticker (top-right, below legend) ---
	drawAlertTicker(screen, fs, sw, sh, tick)

	// --- [NEW-10] Achievement Popup ---
	drawAchievementPopup(screen, fs, sw, sh)

	// --- [NEW] Clipboard flash ---
	if fs.ClipboardFlash > 0 {
		fs.ClipboardFlash--
		alpha := uint8(min(fs.ClipboardFlash*3, 200))
		printColoredAt(screen, locale.T("factory.clipboard_flash"), sw/2-120, 50, color.RGBA{100, 255, 200, alpha})
	}

	// --- Stats Legend (right side) ---
	drawFactoryLegend(screen, fs, sw, sh)

	// --- Selected Bot Info Panel ---
	if fs.SelectedBot >= 0 && fs.SelectedBot < len(fs.Bots) {
		drawSelectedBotPanel(screen, fs, sw, sh, tick)
	}

	// --- [4] Mini-Map (bottom-right, above dashboard) ---
	drawMiniMap(screen, fs, sw, sh)

	// --- Feature 6: QC Reject Effects ---
	drawQCRejectEffects(screen, fs, wx, wy, ws, tick)

	// --- Feature 7: Stock Warning Borders ---
	drawStockWarningBorders(screen, fs, wx, wy, ws, tick)

	// --- Feature 10: Maintenance Planner Overlay ---
	if fs.ShowMaintPlanner {
		drawMaintenancePlanner(screen, fs, sw, sh, tick)
	}

	// --- Factory Tutorial Overlay ---
	if fs.FactoryTutorial != nil && fs.FactoryTutorial.Active {
		DrawFactoryTutorial(screen, fs, sw, sh)
	}

	// --- Factory Help Overlay ---
	if fs.ShowHelp {
		drawFactoryHelp(screen, sw, sh)
	}

	// --- Quick-stats bar at very top of screen ---
	drawFactoryQuickStats(screen, fs, sw)
}

// ============================================================================
// [5] Animated Conveyor Belts
// ============================================================================
func drawConveyorBelts(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	visMinX, visMaxX float64, tick int) {

	beltY := factory.AisleY + factory.AisleH/2
	beltHalfH := 8.0
	beltTop := beltY - beltHalfH
	beltBot := beltY + beltHalfH

	// Yellow edge rails
	railCol := color.RGBA{200, 180, 40, 100}
	vector.StrokeLine(screen, wx(factory.HallX+10), wy(beltTop-1),
		wx(factory.HallX+factory.HallW-10), wy(beltTop-1), 1.5, railCol, false)
	vector.StrokeLine(screen, wx(factory.HallX+10), wy(beltBot+1),
		wx(factory.HallX+factory.HallW-10), wy(beltBot+1), 1.5, railCol, false)

	// Dark rubber belt (dashed pattern that moves)
	beltCol := color.RGBA{40, 42, 48, 160}
	segLen := 20.0
	offset := math.Mod(float64(tick)*0.8, segLen*2)
	for x := factory.HallX + 10.0; x < factory.HallX+factory.HallW-10; x += segLen * 2 {
		sx := x + offset
		if sx > factory.HallX+factory.HallW-10 {
			sx -= (factory.HallW - 20)
		}
		if sx < visMinX || sx > visMaxX {
			continue
		}
		endX := sx + segLen
		if endX > factory.HallX+factory.HallW-10 {
			endX = factory.HallX + factory.HallW - 10
		}
		vector.DrawFilledRect(screen, wx(sx), wy(beltTop), ws(endX-sx), ws(beltBot-beltTop), beltCol, false)
	}

	// Scrolling direction arrows
	arrowCol := color.RGBA{120, 140, 60, 80}
	arrowSpacing := 80.0
	arrowOffset := math.Mod(float64(tick)*0.5, arrowSpacing)
	for x := factory.HallX + 30.0 + arrowOffset; x < factory.HallX+factory.HallW-30; x += arrowSpacing {
		if x < visMinX || x > visMaxX {
			continue
		}
		// Small right-pointing arrow
		vector.StrokeLine(screen, wx(x), wy(beltY-3), wx(x+8), wy(beltY), 1, arrowCol, false)
		vector.StrokeLine(screen, wx(x+8), wy(beltY), wx(x), wy(beltY+3), 1, arrowCol, false)
	}
}

// ============================================================================
// [9] Animated Zone Labels (pulsing glow)
// ============================================================================
func drawAnimatedZoneLabels(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, tick int) {

	zoneLabelY := int(wy(factory.HallY + 12))
	zoneLabels := []struct {
		name string
		x    float64
		w    float64
		col  color.RGBA
	}{
		{locale.T("factory.zone.receiving"), factory.ZoneReceivingX, factory.ZoneReceivingW, color.RGBA{80, 200, 80, 180}},
		{locale.T("factory.zone.storage"), factory.ZoneStorageX, factory.ZoneStorageW, color.RGBA{80, 140, 220, 180}},
		{locale.T("factory.zone.production"), factory.ZoneProductionX, factory.ZoneProductionW, color.RGBA{220, 160, 60, 180}},
		{locale.T("factory.zone.shipping"), factory.ZoneShippingX, factory.ZoneShippingW, color.RGBA{200, 80, 80, 180}},
	}

	for i, zl := range zoneLabels {
		cx := zl.x + zl.w/2
		lx := int(wx(cx)) - runeLen(zl.name)*charW/2

		// Pulsing glow intensity based on zone activity
		pulse := 0.6 + 0.4*math.Sin(float64(tick)*0.06+float64(i)*1.2)
		a := uint8(float64(zl.col.A) * pulse)

		// Glow rectangle behind label
		glowA := uint8(15 * pulse)
		glowW := float32(runeLen(zl.name)*charW + 12)
		glowH := float32(lineH + 4)
		vector.DrawFilledRect(screen, float32(lx-6), float32(zoneLabelY-2), glowW, glowH,
			color.RGBA{zl.col.R, zl.col.G, zl.col.B, glowA}, false)

		printColoredAt(screen, zl.name, lx, zoneLabelY, color.RGBA{zl.col.R, zl.col.G, zl.col.B, a})
	}
}

// ============================================================================
// [6] Forklift-Style Bots + [8] Visual Feedback + [11] Zoom-Dependent Detail
// ============================================================================
func drawFactoryBots(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	visMinX, visMaxX, visMinY, visMaxY float64, tick int) {

	zoom := fs.CamZoom

	for i := range fs.Bots {
		bot := &fs.Bots[i]
		if bot.X < visMinX || bot.X > visMaxX || bot.Y < visMinY || bot.Y > visMaxY {
			continue
		}
		bx := wx(bot.X)
		by := wy(bot.Y)

		// Color by state, with role tint
		botCol := getBotStateColor(bot.State)

		// Role-based color tinting (for idle/moving bots)
		if bot.State != factory.BotOffShift && bot.State != factory.BotEmergencyEvac && bot.CarryingPkg <= 0 {
			if i < len(fs.BotRoles) {
				switch fs.BotRoles[i] {
				case factory.RoleForklift:
					botCol = color.RGBA{
						uint8(min(int(botCol.R)+40, 255)),
						uint8(max(int(botCol.G)-20, 0)),
						uint8(max(int(botCol.B)-30, 0)),
						botCol.A,
					} // orange tint
				case factory.RoleExpress:
					botCol = color.RGBA{
						uint8(max(int(botCol.R)-20, 0)),
						botCol.G,
						uint8(min(int(botCol.B)+40, 255)),
						botCol.A,
					} // blue tint
				}
			}
		}

		// Off-shift bots: render as dark gray
		if bot.State == factory.BotOffShift {
			botCol = color.RGBA{40, 42, 48, 100}
		}

		// Emergency evacuating bots: flashing red
		if bot.State == factory.BotEmergencyEvac {
			if tick%10 < 5 {
				botCol = color.RGBA{255, 60, 60, 255}
			} else {
				botCol = color.RGBA{200, 40, 40, 180}
			}
		}

		// Package color
		pkgCol := color.RGBA{}
		if bot.CarryingPkg > 0 {
			switch bot.CarryingPkg {
			case 1:
				pkgCol = color.RGBA{220, 60, 60, 255}
			case 2:
				pkgCol = color.RGBA{60, 100, 220, 255}
			case 3:
				pkgCol = color.RGBA{220, 200, 40, 255}
			case 4:
				pkgCol = color.RGBA{60, 180, 60, 255}
			case 5: // defective part
				pkgCol = color.RGBA{200, 0, 200, 255} // magenta for defects
			}
			botCol = pkgCol
		}

		// --- [11] Zoom-Dependent Detail Levels ---
		if zoom < 0.3 {
			// Zoomed out: just dots
			dotR := ws(factory.FactoryBotRadius)
			if dotR < 1 {
				dotR = 1
			}
			vector.DrawFilledCircle(screen, bx, by, dotR, botCol, false)

			// [8] Critical energy warning even zoomed out
			if bot.Energy < factory.EnergyCritical && bot.State != factory.BotCharging && bot.State != factory.BotRepairing {
				if tick%20 < 10 {
					vector.DrawFilledCircle(screen, bx, by-dotR-2, 2, color.RGBA{255, 40, 40, 220}, false)
				}
			}

			// [NEW] Energy bar on all bots (skip only if zoom < 0.2)
			if zoom >= 0.2 {
				drawBotEnergyMicroBarAlways(screen, bot, bx, by, ws)
			}
		} else if zoom < 0.8 {
			// Medium zoom: direction line, no fork details
			botR := ws(factory.FactoryBotRadius)
			if botR < 1.5 {
				botR = 1.5
			}
			dirLen := botR * 2.5

			// Motion trail for moving bots
			if bot.State == 1 || bot.State == 3 {
				trailLen := botR * 4
				tx := bx - trailLen*float32(math.Cos(bot.Angle))
				ty := by - trailLen*float32(math.Sin(bot.Angle))
				vector.StrokeLine(screen, bx, by, tx, ty, 1, color.RGBA{botCol.R, botCol.G, botCol.B, 60}, false)
			}

			vector.DrawFilledCircle(screen, bx, by, botR, botCol, false)
			ddx := bx + dirLen*float32(math.Cos(bot.Angle))
			ddy := by + dirLen*float32(math.Sin(bot.Angle))
			vector.StrokeLine(screen, bx, by, ddx, ddy, 1, colorBotDir, false)

			// Malfunction indicator
			if i < len(fs.Malfunctioning) && fs.Malfunctioning[i] {
				if tick%20 < 10 {
					vector.StrokeCircle(screen, bx, by, botR+2, 1, color.RGBA{255, 40, 40, 180}, false)
				}
			}

			// [8] Critical energy: red exclamation
			if bot.Energy < factory.EnergyCritical && bot.State != factory.BotCharging && bot.State != factory.BotRepairing {
				drawExclamation(screen, bx, by-botR-6, tick)
			}

			// [NEW] Energy bar on all bots at medium zoom
			drawBotEnergyMicroBarAlways(screen, bot, bx, by, ws)
		} else {
			// Full zoom: forklift-style rendering
			drawForkliftBot(screen, bot, bx, by, ws, zoom, botCol, tick)

			// Package square
			if bot.CarryingPkg > 0 && pkgCol.A > 0 {
				botR := ws(factory.FactoryBotRadius)
				pkgSize := botR * 1.2
				vector.DrawFilledRect(screen, bx-pkgSize/2, by-botR-pkgSize-1, pkgSize, pkgSize, pkgCol, false)
			}

			// Malfunction indicator
			if i < len(fs.Malfunctioning) && fs.Malfunctioning[i] {
				botR := ws(factory.FactoryBotRadius)
				if tick%20 < 10 {
					vector.StrokeCircle(screen, bx, by, botR+3, 1, color.RGBA{255, 40, 40, 180}, false)
				}
			}

			// [8] Critical energy: red exclamation
			if bot.Energy < factory.EnergyCritical && bot.State != factory.BotCharging && bot.State != factory.BotRepairing {
				botR := ws(factory.FactoryBotRadius)
				drawExclamation(screen, bx, by-botR-8, tick)
			}

			// Energy micro-bar
			drawBotEnergyMicroBar(screen, bot, bx, by, ws)
		}

		// --- [NEW] Experience stars (tiny yellow dots above expert/master bots) ---
		if zoom >= 0.5 && i < len(fs.BotDeliveries) {
			d := fs.BotDeliveries[i]
			if d >= 10 {
				botR := ws(factory.FactoryBotRadius)
				if botR < 1.5 {
					botR = 1.5
				}
				starY := by - botR - 4
				starCol := color.RGBA{255, 220, 60, 200}
				numStars := 1
				if d >= 100 {
					numStars = 3
					starCol = color.RGBA{255, 180, 0, 255}
				} else if d >= 50 {
					numStars = 2
				}
				for si := 0; si < numStars; si++ {
					sx := bx - float32(numStars-1)*2 + float32(si)*4
					vector.DrawFilledCircle(screen, sx, starY, 1.5, starCol, false)
				}
			}
		}

		// --- [NEW] Maintenance warning (yellow indicator above bots near 20000 hours) ---
		if zoom >= 0.4 && i < len(fs.BotOpHours) {
			opH := fs.BotOpHours[i]
			if opH >= factory.MaintenanceWarningHours && opH < factory.MaintenanceMandatoryHours {
				botR := ws(factory.FactoryBotRadius)
				if botR < 1.5 {
					botR = 1.5
				}
				// Yellow wrench icon: simplified as a yellow cross
				wy2 := by - botR - 10
				if tick%30 < 20 {
					vector.StrokeLine(screen, bx-3, wy2, bx+3, wy2, 2, color.RGBA{220, 200, 40, 220}, false)
					vector.StrokeLine(screen, bx, wy2-3, bx, wy2+3, 2, color.RGBA{220, 200, 40, 220}, false)
				}
			}
		}

		// --- [NEW] Speed lines behind Express bots ---
		if zoom >= 0.5 && i < len(fs.BotRoles) && fs.BotRoles[i] == factory.RoleExpress {
			if bot.Speed > 2 && (bot.State == factory.BotMovingToSource || bot.State == factory.BotMovingToDest) {
				botR := ws(factory.FactoryBotRadius)
				tailDir := bot.Angle + math.Pi // behind the bot
				for sl := 0; sl < 3; sl++ {
					lineLen := botR * (2.0 + float32(sl)*1.5)
					offset := float32(sl) * 2
					lx1 := bx + float32(math.Cos(tailDir))*botR*1.5 - float32(math.Sin(tailDir))*offset
					ly1 := by + float32(math.Sin(tailDir))*botR*1.5 + float32(math.Cos(tailDir))*offset
					lx2 := lx1 + float32(math.Cos(tailDir))*lineLen
					ly2 := ly1 + float32(math.Sin(tailDir))*lineLen
					alpha := uint8(80 - sl*20)
					vector.StrokeLine(screen, lx1, ly1, lx2, ly2, 1, color.RGBA{140, 180, 255, alpha}, false)
				}
			}
		}

		// --- [NEW] Forklift bots: larger rectangle at high zoom ---
		if zoom >= 0.8 && i < len(fs.BotRoles) && fs.BotRoles[i] == factory.RoleForklift {
			botR := ws(factory.FactoryBotRadius)
			// Draw a slightly larger outline
			vector.StrokeRect(screen, bx-botR*1.5, by-botR*1.5, botR*3, botR*3, 1, color.RGBA{220, 160, 60, 100}, false)
		}

		// --- Feature 1: Honk flash (yellow dot ahead of bot) ---
		if zoom >= 0.3 && i < len(fs.HonkFlash) && fs.HonkFlash[i] > 0 {
			botR := ws(factory.FactoryBotRadius)
			hx := bx + float32(math.Cos(bot.Angle))*botR*3
			hy := by + float32(math.Sin(bot.Angle))*botR*3
			vector.DrawFilledCircle(screen, hx, hy, 3, color.RGBA{255, 255, 60, 220}, false)
		}

		// --- Feature 5: Pallet stack rendering (colored squares above forklift bots) ---
		if zoom >= 0.4 && i < len(fs.BotPallet) && len(fs.BotPallet[i]) > 1 {
			botR := ws(factory.FactoryBotRadius)
			for pi, pc := range fs.BotPallet[i] {
				pCol := partColorToRGBA(pc)
				pSize := botR * 0.8
				py := by - botR - 2 - float32(pi)*(pSize+1)
				vector.DrawFilledRect(screen, bx-pSize/2, py-pSize, pSize, pSize, pCol, false)
			}
		}

		// --- Feature 12: Golden glow for #1 productivity bot ---
		if fs.Stats.TopWorkers[0][1] > 0 && i == fs.Stats.TopWorkers[0][0] {
			goldPulse := float32(1.0 + 0.2*math.Sin(float64(tick)*0.1))
			goldR := ws(factory.FactoryBotRadius) * 2.5 * goldPulse
			if goldR < 3 {
				goldR = 3
			}
			goldAlpha := uint8(80 + int(40*math.Sin(float64(tick)*0.1)))
			vector.StrokeCircle(screen, bx, by, goldR, 2, color.RGBA{255, 215, 0, goldAlpha}, false)
		}

		// --- Selection ring (all zoom levels) ---
		if i == fs.SelectedBot {
			botR := ws(factory.FactoryBotRadius)
			if botR < 1.5 {
				botR = 1.5
			}
			pulse := float32(1.0 + 0.3*math.Sin(float64(tick)*0.15))
			ringR := botR * 3 * pulse
			vector.StrokeCircle(screen, bx, by, ringR, 2, color.RGBA{255, 255, 255, 200}, false)

			// Task target line
			taskIdx := factory.FindBotTask(fs, i)
			if taskIdx >= 0 {
				task := &fs.Tasks.Tasks[taskIdx]
				if bot.State == 1 || bot.State == 2 {
					drawAnimatedDashedLine(screen, bx, by, wx(task.SourceX), wy(task.SourceY),
						color.RGBA{0, 220, 255, 150}, tick)
				} else if bot.State == 3 || bot.State == 4 {
					drawAnimatedDashedLine(screen, bx, by, wx(task.DestX), wy(task.DestY),
						color.RGBA{255, 220, 60, 150}, tick)
				}
			}
		}
	}
}

// getBotColor returns the color for a bot based on state (unused, see getBotStateColor).
func getBotColor(bot *swarm.SwarmBot, idx int, fs *factory.FactoryState) color.RGBA {
	return getBotStateColor(bot.State)
}

// getBotStateColor returns bot color by state.
func getBotStateColor(state int) color.RGBA {
	switch state {
	case 1:
		return color.RGBA{0, 200, 220, 255}
	case 2:
		return color.RGBA{0, 220, 100, 255}
	case 3:
		return color.RGBA{220, 200, 40, 255}
	case 4:
		return color.RGBA{0, 255, 120, 255}
	case 5:
		return color.RGBA{255, 220, 40, 255}
	case 6:
		return color.RGBA{220, 120, 40, 255}
	case factory.BotOffShift:
		return color.RGBA{40, 42, 48, 100}
	case factory.BotEmergencyEvac:
		return color.RGBA{255, 60, 60, 255}
	default:
		return color.RGBA{100, 100, 110, 255}
	}
}

// partColorToRGBA converts a part color int to an RGBA color.
func partColorToRGBA(c int) color.RGBA {
	switch c {
	case 1:
		return color.RGBA{220, 60, 60, 255}
	case 2:
		return color.RGBA{60, 100, 220, 255}
	case 3:
		return color.RGBA{220, 200, 40, 255}
	case 4:
		return color.RGBA{60, 180, 60, 255}
	case 5: // defective
		return color.RGBA{200, 0, 200, 255}
	default:
		return color.RGBA{150, 150, 150, 200}
	}
}

// [6] drawForkliftBot draws a tiny top-down forklift.
func drawForkliftBot(screen *ebiten.Image, bot *swarm.SwarmBot,
	bx, by float32, ws func(float64) float32, zoom float64, botCol color.RGBA, tick int) {
	// We need angle for orientation
	// Body: small rectangle 4x6 oriented by angle
	// Fork: 2 lines protruding from front

	// Forklift dimensions in world units
	bodyW := 4.0
	bodyH := 6.0
	forkLen := 4.0

	cosA := float32(math.Cos(bot.Angle))
	sinA := float32(math.Sin(bot.Angle))

	bw := ws(bodyW)
	bh := ws(bodyH)
	fl := ws(forkLen)

	// Body center at (bx, by)
	// Draw body as a rotated rectangle (approximated with 4 lines)
	// Half dimensions
	hw := bw / 2
	hh := bh / 2

	// Body corners (rotated)
	// Forward direction is along angle
	// perpendicular = angle + pi/2
	perpX := -sinA
	perpY := cosA

	// 4 corners
	c1x := bx + cosA*hh + perpX*hw
	c1y := by + sinA*hh + perpY*hw
	c2x := bx + cosA*hh - perpX*hw
	c2y := by + sinA*hh - perpY*hw
	c3x := bx - cosA*hh - perpX*hw
	c3y := by - sinA*hh - perpY*hw
	c4x := bx - cosA*hh + perpX*hw
	c4y := by - sinA*hh + perpY*hw

	// Fill body with a quad (using lines since we don't have polygon)
	vector.StrokeLine(screen, c1x, c1y, c2x, c2y, bw*0.8, botCol, false)
	vector.StrokeLine(screen, c3x, c3y, c4x, c4y, bw*0.8, botCol, false)
	vector.StrokeLine(screen, c1x, c1y, c4x, c4y, 1, color.RGBA{botCol.R + 30, botCol.G + 30, botCol.B + 30, 255}, false)
	vector.StrokeLine(screen, c2x, c2y, c3x, c3y, 1, color.RGBA{botCol.R + 30, botCol.G + 30, botCol.B + 30, 255}, false)

	// Fork prongs (2 lines extending from front)
	frontX := bx + cosA*hh
	frontY := by + sinA*hh

	forkOffset := hw * 0.6
	fork1sx := frontX + perpX*forkOffset
	fork1sy := frontY + perpY*forkOffset
	fork1ex := fork1sx + cosA*fl
	fork1ey := fork1sy + sinA*fl

	fork2sx := frontX - perpX*forkOffset
	fork2sy := frontY - perpY*forkOffset
	fork2ex := fork2sx + cosA*fl
	fork2ey := fork2sy + sinA*fl

	forkCol := color.RGBA{180, 180, 190, 220}
	vector.StrokeLine(screen, fork1sx, fork1sy, fork1ex, fork1ey, 1.5, forkCol, false)
	vector.StrokeLine(screen, fork2sx, fork2sy, fork2ex, fork2ey, 1.5, forkCol, false)
}

// drawBotEnergyMicroBar draws a tiny energy bar below a fully-zoomed bot.
func drawBotEnergyMicroBar(screen *ebiten.Image, bot *swarm.SwarmBot,
	bx, by float32, ws func(float64) float32) {
	botR := ws(factory.FactoryBotRadius)
	barW := botR * 3
	barH := float32(2)
	barX := bx - barW/2
	barY := by + botR + 2

	energy := bot.Energy
	if energy > 100 {
		energy = 100
	}
	if energy < 0 {
		energy = 0
	}
	fill := float32(energy / 100.0)

	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{30, 30, 40, 150}, false)
	eCol := color.RGBA{80, 220, 80, 180}
	if energy < 30 {
		eCol = color.RGBA{220, 60, 60, 180}
	} else if energy < 60 {
		eCol = color.RGBA{220, 200, 60, 180}
	}
	vector.DrawFilledRect(screen, barX, barY, barW*fill, barH, eCol, false)
}

// [8] drawExclamation draws a red "!" above a bot with critical energy.
func drawExclamation(screen *ebiten.Image, x, y float32, tick int) {
	if tick%30 < 20 { // blink
		// Simple exclamation mark: vertical line + dot
		vector.StrokeLine(screen, x, y, x, y-6, 2, color.RGBA{255, 40, 40, 240}, false)
		vector.DrawFilledCircle(screen, x, y+2, 1.5, color.RGBA{255, 40, 40, 240}, false)
	}
}

// ============================================================================
// [1] Day/Night Cycle Overlay
// ============================================================================
func drawFactoryDayNightOverlay(screen *ebiten.Image, fs *factory.FactoryState, sw, sh int) {
	// Every 10000 ticks = one day cycle
	cyclePos := math.Mod(float64(fs.Tick), 10000.0) / 10000.0

	var overlayCol color.RGBA

	switch {
	case cyclePos < 0.15: // Dawn (0 - 0.15)
		t := cyclePos / 0.15
		alpha := uint8(30 * (1.0 - t))
		overlayCol = color.RGBA{200, 120, 40, alpha}
	case cyclePos < 0.45: // Day (0.15 - 0.45)
		// No overlay (normal brightness)
		return
	case cyclePos < 0.55: // Dusk (0.45 - 0.55)
		t := (cyclePos - 0.45) / 0.10
		alpha := uint8(t * 30)
		overlayCol = color.RGBA{140, 40, 100, alpha}
	case cyclePos < 0.85: // Night (0.55 - 0.85)
		overlayCol = color.RGBA{10, 10, 40, 45}
	default: // Pre-dawn (0.85 - 1.0)
		t := (cyclePos - 0.85) / 0.15
		overlayCol = color.RGBA{uint8(10 + t*190), uint8(10 + t*110), uint8(40 + t*0), uint8(45 - t*15)}
	}

	if overlayCol.A > 0 {
		vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), overlayCol, false)
	}
}

// ============================================================================
// [2] Weather Effects (Rain)
// ============================================================================
func drawRainEffect(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	visMinX, visMaxX float64, tick int, sw, sh int) {

	// Hundreds of tiny diagonal lines in the yard area (Y < HallY)
	rainCol := color.RGBA{140, 160, 200, 60}
	rainCount := 300
	seed := tick * 7

	for i := 0; i < rainCount; i++ {
		// Pseudo-random positions based on tick and index (no allocations)
		h := seed + i*137
		rx := float64(h%3000)
		ry := math.Mod(float64((h*73+i*31)%600)+float64(tick%200)*2.0, factory.YardH)

		if rx < visMinX || rx > visMaxX {
			continue
		}

		// Screen coords
		sx := wx(rx)
		sy := wy(ry)

		// Only draw if on-screen
		if sx < 0 || sx > float32(sw) || sy < 0 || sy > float32(sh) {
			continue
		}

		// Diagonal line (wind from top-left to bottom-right)
		vector.StrokeLine(screen, sx, sy, sx+4, sy+8, 1, rainCol, false)
	}
}

// ============================================================================
// [3] Heatmap Overlay
// ============================================================================
func drawHeatmapOverlay(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32) {

	if len(fs.HeatmapGrid) == 0 || fs.HeatmapW <= 0 || fs.HeatmapH <= 0 {
		return
	}

	cellW := factory.WorldW / float64(fs.HeatmapW)
	cellH := factory.WorldH / float64(fs.HeatmapH)

	// Find max value for normalization
	maxVal := 1
	for _, v := range fs.HeatmapGrid {
		if v > maxVal {
			maxVal = v
		}
	}

	for gy := 0; gy < fs.HeatmapH; gy++ {
		for gx := 0; gx < fs.HeatmapW; gx++ {
			idx := gy*fs.HeatmapW + gx
			if idx >= len(fs.HeatmapGrid) {
				continue
			}
			val := fs.HeatmapGrid[idx]
			if val == 0 {
				continue
			}

			t := float64(val) / float64(maxVal)
			// Blue (cold) -> Red (hot)
			r := uint8(t * 220)
			g := uint8(0)
			b := uint8((1 - t) * 220)
			a := uint8(20 + t*60)

			worldX := float64(gx) * cellW
			worldY := float64(gy) * cellH
			vector.DrawFilledRect(screen, wx(worldX), wy(worldY), ws(cellW), ws(cellH),
				color.RGBA{r, g, b, a}, false)
		}
	}
}

// ============================================================================
// [4] Mini-Map
// ============================================================================
func drawMiniMap(screen *ebiten.Image, fs *factory.FactoryState, sw, sh int) {
	mmW := 200
	mmH := 133 // 3000:2000 = 3:2 ratio
	mmX := sw - mmW - 12
	mmY := sh - dashboardH - mmH - 12

	// Background
	vector.DrawFilledRect(screen, float32(mmX), float32(mmY), float32(mmW), float32(mmH),
		color.RGBA{10, 12, 20, 200}, false)
	vector.StrokeRect(screen, float32(mmX), float32(mmY), float32(mmW), float32(mmH),
		1, color.RGBA{60, 70, 100, 180}, false)

	scaleX := float64(mmW) / factory.WorldW
	scaleY := float64(mmH) / factory.WorldH

	// Yard
	yardH := int(factory.YardH * scaleY)
	vector.DrawFilledRect(screen, float32(mmX), float32(mmY), float32(mmW), float32(yardH),
		color.RGBA{90, 92, 85, 200}, false)

	// Hall
	hallSX := int(factory.HallX*scaleX) + mmX
	hallSY := int(factory.HallY*scaleY) + mmY
	hallSW := int(factory.HallW * scaleX)
	hallSH := int(factory.HallH * scaleY)
	vector.DrawFilledRect(screen, float32(hallSX), float32(hallSY), float32(hallSW), float32(hallSH),
		color.RGBA{55, 60, 70, 200}, false)

	// Machines
	for _, m := range fs.Machines {
		mx := int(m.X*scaleX) + mmX
		my := int(m.Y*scaleY) + mmY
		mw := int(m.W * scaleX)
		mh := int(m.H * scaleY)
		if mw < 1 {
			mw = 1
		}
		if mh < 1 {
			mh = 1
		}
		col := colorMachine
		if m.Active {
			col = color.RGBA{80, 220, 80, 200}
		}
		vector.DrawFilledRect(screen, float32(mx), float32(my), float32(mw), float32(mh), col, false)
	}

	// Chargers
	for _, ch := range fs.Chargers {
		cx := int(ch.X*scaleX) + mmX
		cy := int(ch.Y*scaleY) + mmY
		vector.DrawFilledRect(screen, float32(cx), float32(cy), 3, 3, colorCharger, false)
	}

	// Bots as single pixels
	for i := range fs.Bots {
		bot := &fs.Bots[i]
		bx := int(bot.X*scaleX) + mmX
		by := int(bot.Y*scaleY) + mmY
		col := getBotStateColor(bot.State)
		vector.DrawFilledRect(screen, float32(bx), float32(by), 1, 1, col, false)
	}

	// Trucks
	for _, truck := range fs.Trucks {
		if truck.Phase == factory.TruckWaiting {
			continue
		}
		tx := int(truck.X*scaleX) + mmX
		ty := int(truck.Y*scaleY) + mmY
		tw := int(truck.W * scaleX)
		if tw < 2 {
			tw = 2
		}
		vector.DrawFilledRect(screen, float32(tx), float32(ty), float32(tw), 2,
			color.RGBA{180, 180, 190, 220}, false)
	}

	// Camera viewport rectangle (white outline)
	vpLeft := fs.CamX - float64(sw)/2/fs.CamZoom
	vpRight := fs.CamX + float64(sw)/2/fs.CamZoom
	vpTop := fs.CamY - float64(sh)/2/fs.CamZoom
	vpBottom := fs.CamY + float64(sh)/2/fs.CamZoom
	vpSX := float32(vpLeft*scaleX) + float32(mmX)
	vpSY := float32(vpTop*scaleY) + float32(mmY)
	vpSW := float32((vpRight - vpLeft) * scaleX)
	vpSH := float32((vpBottom - vpTop) * scaleY)
	vector.StrokeRect(screen, vpSX, vpSY, vpSW, vpSH, 1, color.RGBA{255, 255, 255, 160}, false)

	// Label
	printColoredAt(screen, locale.T("factory.minimap"), mmX+3, mmY+2, color.RGBA{140, 150, 170, 150})
}

// ============================================================================
// [7] Real-Time Production Dashboard
// ============================================================================
func drawSelectedBotPath(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32) {

	if len(fs.SelectedBotPath) == 0 || fs.SelectedBotPathIdx == 0 {
		return
	}

	n := len(fs.SelectedBotPath)
	count := fs.SelectedBotPathIdx
	if count > n {
		count = n
	}

	for i := 0; i < count-1; i++ {
		// Read from oldest to newest
		idx1 := (fs.SelectedBotPathIdx - count + i) % n
		idx2 := (fs.SelectedBotPathIdx - count + i + 1) % n
		if idx1 < 0 {
			idx1 += n
		}
		if idx2 < 0 {
			idx2 += n
		}

		p1 := fs.SelectedBotPath[idx1]
		p2 := fs.SelectedBotPath[idx2]

		// Skip zero entries
		if p1[0] == 0 && p1[1] == 0 {
			continue
		}
		if p2[0] == 0 && p2[1] == 0 {
			continue
		}

		// Fading alpha (older = more transparent)
		alpha := uint8(float64(i) / float64(count) * 120)
		vector.StrokeLine(screen, wx(p1[0]), wy(p1[1]), wx(p2[0]), wy(p2[1]), 1,
			color.RGBA{100, 200, 255, alpha}, false)
	}
}

// ============================================================================
// [12] Edge-of-Screen Indicators
// ============================================================================
func drawOffScreenIndicator(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, sw, sh int) {

	bot := &fs.Bots[fs.SelectedBot]
	bx := wx(bot.X)
	by := wy(bot.Y)

	marginX := float32(30)
	marginY := float32(30)

	// Check if bot is on screen
	if bx >= marginX && bx <= float32(sw)-marginX && by >= marginY && by <= float32(sh)-marginY {
		return // on screen, no indicator needed
	}

	// Clamp to screen edge
	ix := bx
	iy := by
	if ix < marginX {
		ix = marginX
	}
	if ix > float32(sw)-marginX {
		ix = float32(sw) - marginX
	}
	if iy < marginY {
		iy = marginY
	}
	if iy > float32(sh)-marginY {
		iy = float32(sh) - marginY
	}

	// Arrow direction
	dx := bx - ix
	dy := by - iy
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length < 1 {
		return
	}
	ux := dx / length
	uy := dy / length

	col := getBotStateColor(bot.State)

	// Arrow head
	arrowLen := float32(12)
	tipX := ix + ux*arrowLen
	tipY := iy + uy*arrowLen
	perpX := -uy
	perpY := ux
	side := float32(5)

	vector.StrokeLine(screen, tipX, tipY, ix+perpX*side, iy+perpY*side, 2, col, false)
	vector.StrokeLine(screen, tipX, tipY, ix-perpX*side, iy-perpY*side, 2, col, false)
	vector.StrokeLine(screen, ix+perpX*side, iy+perpY*side, ix-perpX*side, iy-perpY*side, 2, col, false)
}

// ============================================================================
// Existing helper functions (unchanged)
// ============================================================================

// drawFactoryTrucks renders all trucks on the road and at docks.
func drawFactoryTrucks(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	visMinX, visMaxX float64, tick int) {

	for _, truck := range fs.Trucks {
		if truck.Phase == factory.TruckWaiting {
			continue
		}
		tx, ty := truck.X, truck.Y
		tw, th := truck.W, truck.H
		if tw <= 0 {
			tw = 120
		}
		if th <= 0 {
			th = 40
		}
		if tx+tw < visMinX || tx > visMaxX {
			continue
		}

		cabW := tw * 0.25
		trailerW := tw * 0.75
		isMoving := truck.Phase == factory.TruckEntering || truck.Phase == factory.TruckExiting
		isParked := truck.Phase == factory.TruckParked || truck.Phase == factory.TruckUnloading || truck.Phase == factory.TruckLoading

		if truck.Direction == 0 {
			vector.DrawFilledRect(screen, wx(tx), wy(ty), ws(trailerW), ws(th), colorTruckBody, false)
			vector.StrokeRect(screen, wx(tx), wy(ty), ws(trailerW), ws(th), 1, colorTruckEdge, false)
			vector.DrawFilledRect(screen, wx(tx+trailerW), wy(ty+th*0.1), ws(cabW), ws(th*0.8), colorTruckCab, false)
			vector.StrokeRect(screen, wx(tx+trailerW), wy(ty+th*0.1), ws(cabW), ws(th*0.8), 1, colorTruckEdge, false)
			vector.DrawFilledRect(screen, wx(tx+trailerW+cabW*0.6), wy(ty+th*0.2), ws(cabW*0.3), ws(th*0.6),
				color.RGBA{100, 140, 180, 180}, false)
			wheelR := th * 0.15
			vector.DrawFilledCircle(screen, wx(tx+tw*0.15), wy(ty+th), ws(wheelR), color.RGBA{30, 30, 30, 255}, false)
			vector.DrawFilledCircle(screen, wx(tx+tw*0.35), wy(ty+th), ws(wheelR), color.RGBA{30, 30, 30, 255}, false)
			vector.DrawFilledCircle(screen, wx(tx+tw*0.85), wy(ty+th), ws(wheelR), color.RGBA{30, 30, 30, 255}, false)
			if isMoving {
				vector.DrawFilledCircle(screen, wx(tx+tw+2), wy(ty+th*0.3), ws(3), colorHeadlight, false)
				vector.DrawFilledCircle(screen, wx(tx+tw+2), wy(ty+th*0.7), ws(3), colorHeadlight, false)
			}
			if isParked {
				vector.DrawFilledRect(screen, wx(tx), wy(ty+2), ws(8), ws(th-4), colorTruckCargo, false)
			}
		} else {
			vector.DrawFilledRect(screen, wx(tx), wy(ty+th*0.1), ws(cabW), ws(th*0.8), colorTruckCab, false)
			vector.StrokeRect(screen, wx(tx), wy(ty+th*0.1), ws(cabW), ws(th*0.8), 1, colorTruckEdge, false)
			vector.DrawFilledRect(screen, wx(tx+cabW*0.1), wy(ty+th*0.2), ws(cabW*0.3), ws(th*0.6),
				color.RGBA{100, 140, 180, 180}, false)
			vector.DrawFilledRect(screen, wx(tx+cabW), wy(ty), ws(trailerW), ws(th), colorTruckBody, false)
			vector.StrokeRect(screen, wx(tx+cabW), wy(ty), ws(trailerW), ws(th), 1, colorTruckEdge, false)
			wheelR := th * 0.15
			vector.DrawFilledCircle(screen, wx(tx+tw*0.15), wy(ty+th), ws(wheelR), color.RGBA{30, 30, 30, 255}, false)
			vector.DrawFilledCircle(screen, wx(tx+tw*0.65), wy(ty+th), ws(wheelR), color.RGBA{30, 30, 30, 255}, false)
			vector.DrawFilledCircle(screen, wx(tx+tw*0.85), wy(ty+th), ws(wheelR), color.RGBA{30, 30, 30, 255}, false)
			if isMoving {
				vector.DrawFilledCircle(screen, wx(tx-2), wy(ty+th*0.3), ws(3), colorHeadlight, false)
				vector.DrawFilledCircle(screen, wx(tx-2), wy(ty+th*0.7), ws(3), colorHeadlight, false)
			}
			if isParked {
				vector.DrawFilledRect(screen, wx(tx+tw-8), wy(ty+2), ws(8), ws(th-4), colorTruckCargo, false)
			}
		}

		// Tail lights
		if isMoving {
			tailCol := color.RGBA{200, 30, 30, 200}
			if truck.Direction == 0 {
				vector.DrawFilledCircle(screen, wx(tx-1), wy(ty+th*0.3), ws(2), tailCol, false)
				vector.DrawFilledCircle(screen, wx(tx-1), wy(ty+th*0.7), ws(2), tailCol, false)
			} else {
				vector.DrawFilledCircle(screen, wx(tx+tw+1), wy(ty+th*0.3), ws(2), tailCol, false)
				vector.DrawFilledCircle(screen, wx(tx+tw+1), wy(ty+th*0.7), ws(2), tailCol, false)
			}
		}

		// Loading/unloading animation
		if truck.Phase == factory.TruckUnloading || truck.Phase == factory.TruckLoading {
			pkgPhase := float64(tick%40) / 40.0
			for p := 0; p < 3; p++ {
				pp := math.Mod(pkgPhase+float64(p)*0.33, 1.0)
				var pkgX, pkgY float64
				if truck.Phase == factory.TruckUnloading {
					pkgX = tx + tw*0.5
					pkgY = ty + th + pp*40
				} else {
					pkgX = tx + tw*0.5
					pkgY = ty + th + 40 - pp*40
				}
				pkgCol := color.RGBA{220, 180, 80, uint8(200 - int(pp*120))}
				vector.DrawFilledRect(screen, wx(pkgX-3), wy(pkgY-3), ws(6), ws(6), pkgCol, false)
			}
		}
	}
}

// drawFactoryMachines renders machines with spinning gear and progress bar.
func drawFactoryMachines(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	tick int) {

	machineNames := []string{locale.T("factory.machine.cnc1"), locale.T("factory.machine.cnc2"), locale.T("factory.machine.assembly"), locale.T("factory.machine.drill1"), locale.T("factory.machine.drill2"), locale.T("factory.machine.qcfinal")}

	// Bottleneck detection: find machine with longest queue
	bottleneckIdx := -1
	maxQueue := 0
	for bi, bm := range fs.Machines {
		if bm.CurrentInput > maxQueue {
			maxQueue = bm.CurrentInput
			bottleneckIdx = bi
		}
	}

	for i, m := range fs.Machines {
		vector.DrawFilledRect(screen, wx(m.X), wy(m.Y), ws(m.W), ws(m.H), colorMachine, false)
		// Bottleneck: red pulsing border
		if i == bottleneckIdx && maxQueue > 2 {
			pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.15)
			borderAlpha := uint8(120 + int(pulse*135))
			vector.StrokeRect(screen, wx(m.X-2), wy(m.Y-2), ws(m.W+4), ws(m.H+4), 3, color.RGBA{255, 60, 40, borderAlpha}, false)
		} else {
			vector.StrokeRect(screen, wx(m.X), wy(m.Y), ws(m.W), ws(m.H), 2, colorMachineEdge, false)
		}

		cx := m.X + m.W/2
		cy := m.Y + m.H/2
		r := m.W / 3

		if m.Active && m.ProcessTimer > 0 {
			gearAngle := float64(tick) * 0.08
			gearR := r
			gearTeeth := 6
			for t := 0; t < gearTeeth; t++ {
				a := gearAngle + float64(t)*math.Pi*2/float64(gearTeeth)
				x1 := cx + gearR*0.7*math.Cos(a)
				y1 := cy + gearR*0.7*math.Sin(a)
				x2 := cx + gearR*1.1*math.Cos(a)
				y2 := cy + gearR*1.1*math.Sin(a)
				vector.StrokeLine(screen, wx(x1), wy(y1), wx(x2), wy(y2), 2, colorMachineEdge, false)
			}
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(gearR*0.7), 2, colorMachineEdge, false)
			vector.DrawFilledCircle(screen, wx(cx), wy(cy), ws(gearR*0.3), colorMachineEdge, false)

			if m.ProcessTime > 0 {
				barW := m.W * 0.7
				barH := 6.0
				barX := m.X + (m.W-barW)/2
				barY := m.Y + m.H - 14
				progress := 1.0 - float64(m.ProcessTimer)/float64(m.ProcessTime)
				vector.DrawFilledRect(screen, wx(barX), wy(barY), ws(barW), ws(barH), color.RGBA{30, 35, 50, 200}, false)
				fillCol := color.RGBA{60, 200, 80, 200}
				if progress > 0.8 {
					fillCol = color.RGBA{80, 220, 255, 200}
				}
				vector.DrawFilledRect(screen, wx(barX), wy(barY), ws(barW*progress), ws(barH), fillCol, false)
				vector.StrokeRect(screen, wx(barX), wy(barY), ws(barW), ws(barH), 1, colorMachineEdge, false)
			}
		} else {
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(r), 2, colorMachineEdge, false)
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(r*0.4), 1, colorMachineEdge, false)
		}

		if m.OutputReady {
			pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.12)
			glowAlpha := uint8(40 + int(pulse*100))
			glowR := r * (1.5 + pulse*0.3)
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(glowR), 2, color.RGBA{80, 255, 120, glowAlpha}, false)
			vector.DrawFilledCircle(screen, wx(m.X+m.W-8), wy(m.Y+8), ws(4), color.RGBA{80, 255, 120, 200}, false)
		}

		if m.CurrentInput > 0 {
			for inp := 0; inp < m.CurrentInput && inp < 5; inp++ {
				dotY := m.Y + 8 + float64(inp)*8
				vector.DrawFilledCircle(screen, wx(m.X+6), wy(dotY), ws(2.5),
					color.RGBA{220, 180, 60, 200}, false)
			}
		}

		label := fmt.Sprintf("M%d", i+1)
		if i < len(machineNames) {
			label = machineNames[i]
		}
		lx := int(wx(cx)) - runeLen(label)*charW/2
		ly := int(wy(m.Y + m.H + 5))
		printColoredAt(screen, label, lx, ly, colorMachineEdge)

		// Feature 10: Traffic light status indicator
		lightX := m.X + m.W + 5
		lightY := m.Y + 5
		lightR := float32(4) * ws(1)
		if lightR < 2 {
			lightR = 2
		}

		redOn := m.CoolingDown
		yellowOn := m.NeedsInput && !m.Active
		greenOn := m.Active && m.ProcessTimer > 0

		// Draw 3 dark background circles
		for li := 0; li < 3; li++ {
			lyOff := lightY + float64(li)*12
			vector.DrawFilledCircle(screen, wx(lightX), wy(lyOff), lightR, color.RGBA{30, 30, 30, 200}, false)
		}
		// Draw active lights
		if redOn {
			vector.DrawFilledCircle(screen, wx(lightX), wy(lightY), lightR, color.RGBA{220, 40, 40, 255}, false)
		}
		if yellowOn {
			vector.DrawFilledCircle(screen, wx(lightX), wy(lightY+12), lightR, color.RGBA{220, 200, 40, 255}, false)
		}
		if greenOn {
			vector.DrawFilledCircle(screen, wx(lightX), wy(lightY+24), lightR, color.RGBA{40, 220, 40, 255}, false)
		}
	}
}

// drawFactoryChargers renders charging stations with pulsing energy animation.
func drawFactoryChargers(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	tick int) {

	// Day/night glow boost
	cyclePos := math.Mod(float64(fs.Tick), 10000.0) / 10000.0
	nightBoost := uint8(0)
	if cyclePos > 0.55 && cyclePos < 0.85 {
		nightBoost = 40
	}

	for _, ch := range fs.Chargers {
		vector.DrawFilledRect(screen, wx(ch.X), wy(ch.Y), ws(ch.W), ws(ch.H), colorCharger, false)
		boltBright := color.RGBA{colorChargerBolt.R, colorChargerBolt.G, colorChargerBolt.B + nightBoost, 255}
		vector.StrokeRect(screen, wx(ch.X), wy(ch.Y), ws(ch.W), ws(ch.H), 1, boltBright, false)

		cx := ch.X + ch.W/2
		cy := ch.Y + ch.H/2
		boltSize := ch.W * 0.3

		vector.StrokeLine(screen, wx(cx-boltSize*0.3), wy(cy-boltSize), wx(cx+boltSize*0.2), wy(cy), 2, boltBright, false)
		vector.StrokeLine(screen, wx(cx+boltSize*0.2), wy(cy), wx(cx-boltSize*0.2), wy(cy), 2, boltBright, false)
		vector.StrokeLine(screen, wx(cx-boltSize*0.2), wy(cy), wx(cx+boltSize*0.3), wy(cy+boltSize), 2, boltBright, false)

		numCharging := len(ch.Occupants)
		if numCharging > 0 {
			pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.1)
			pulseR1 := ch.W*0.5 + pulse*ch.W*0.15
			alpha1 := uint8(60 + int(pulse*120))
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(pulseR1), 1, color.RGBA{255, 240, 60, alpha1}, false)
			pulse2 := 0.5 + 0.5*math.Sin(float64(tick)*0.1+1.5)
			pulseR2 := ch.W*0.65 + pulse2*ch.W*0.15
			alpha2 := uint8(30 + int(pulse2*80))
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(pulseR2), 1, color.RGBA{255, 240, 60, alpha2}, false)

			for ci, botIdx := range ch.Occupants {
				if botIdx < 0 || botIdx >= len(fs.Bots) {
					continue
				}
				if ci >= 4 {
					break
				}
				bot := &fs.Bots[botIdx]
				batX := ch.X + ch.W + 4
				batY := ch.Y + float64(ci)*12
				batW := 14.0
				batH := 8.0
				vector.StrokeRect(screen, wx(batX), wy(batY), ws(batW), ws(batH), 1, colorChargerBolt, false)
				vector.DrawFilledRect(screen, wx(batX+batW), wy(batY+2), ws(2), ws(batH-4), colorChargerBolt, false)
				fill := bot.Energy / 100.0
				if fill > 1 {
					fill = 1
				}
				if fill < 0 {
					fill = 0
				}
				fillCol := color.RGBA{80, 220, 80, 200}
				if fill < 0.3 {
					fillCol = color.RGBA{220, 60, 60, 200}
				} else if fill < 0.6 {
					fillCol = color.RGBA{220, 200, 60, 200}
				}
				vector.DrawFilledRect(screen, wx(batX+1), wy(batY+1), ws((batW-2)*fill), ws(batH-2), fillCol, false)
			}
		}
	}
}

// drawFactoryDocks renders docks with activity indicators and warning lights.
func drawFactoryDocks(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	visMinX, visMaxX float64, tick int) {

	dockLabels := []string{locale.T("factory.dock.inbound1"), locale.T("factory.dock.inbound2"), locale.T("factory.dock.outbound1"), locale.T("factory.dock.outbound2")}

	for i, dock := range fs.Docks {
		if dock.X+dock.W < visMinX || dock.X > visMaxX {
			continue
		}

		vector.DrawFilledRect(screen, wx(dock.X), wy(dock.Y), ws(dock.W), ws(dock.H), colorDock, false)
		vector.StrokeRect(screen, wx(dock.X), wy(dock.Y), ws(dock.W), ws(dock.H), 2, colorDockEdge, false)

		stripeW := 10.0
		for sx := dock.X; sx < dock.X+dock.W; sx += stripeW * 2 {
			sw := stripeW
			if sx+sw > dock.X+dock.W {
				sw = dock.X + dock.W - sx
			}
			vector.DrawFilledRect(screen, wx(sx), wy(dock.Y), ws(sw), ws(5), colorRoadLine, false)
		}

		if dock.TruckIdx >= 0 {
			flash := tick % 30
			if flash < 15 {
				warnCol := color.RGBA{255, 160, 30, 200}
				vector.DrawFilledCircle(screen, wx(dock.X-5), wy(dock.Y+dock.H/2), ws(4), warnCol, false)
				vector.DrawFilledCircle(screen, wx(dock.X+dock.W+5), wy(dock.Y+dock.H/2), ws(4), warnCol, false)
			}
			vector.DrawFilledRect(screen, wx(dock.X), wy(dock.Y+dock.H-4), ws(dock.W), ws(4),
				color.RGBA{255, 140, 30, 120}, false)

			pkgPhase := float64(tick%50) / 50.0
			for p := 0; p < 2; p++ {
				pp := math.Mod(pkgPhase+float64(p)*0.5, 1.0)
				pkgX := dock.X + dock.W*0.3 + float64(p)*dock.W*0.4
				pkgY := dock.Y - pp*30
				alpha := uint8(180 - int(pp*120))
				vector.DrawFilledRect(screen, wx(pkgX-2), wy(pkgY-2), ws(5), ws(5),
					color.RGBA{220, 180, 80, alpha}, false)
			}
		}

		label := locale.Tf("factory.dock.label", i+1)
		if i < len(dockLabels) {
			label = dockLabels[i]
		}
		lx := int(wx(dock.X+dock.W/2)) - runeLen(label)*charW/2
		ly := int(wy(dock.Y - 14))
		printColoredAt(screen, label, lx, ly, colorDockEdge)
	}
}

// drawAisleFlowParticles draws animated dots flowing through the aisle.
func SpawnPulseEffect(fs *factory.FactoryState, x, y float64, r, g, b uint8) {
	fs.PulseEffects = append(fs.PulseEffects, factory.PulseEffect{
		X: x, Y: y, MaxR: 25, Color: [3]uint8{r, g, b},
	})
}

// ============================================================================
// [NEW] Order Panel (left side, below HUD)
// ============================================================================

// ============================================================================
// Factory Tutorial — 10-step guided tour
// ============================================================================

// DrawFactoryTutorial renders the factory tutorial overlay.
func DrawFactoryTutorial(screen *ebiten.Image, fs *factory.FactoryState, sw, sh int) {
	ft := fs.FactoryTutorial
	if ft == nil || !ft.Active || ft.Step < 0 || ft.Step >= 10 {
		return
	}

	// Semi-transparent overlay at bottom
	boxW := 650
	boxH := 90
	boxX := (sw - boxW) / 2
	boxY := sh - boxH - 30

	bg := color.RGBA{10, 15, 35, 220}
	border := color.RGBA{80, 180, 255, 255}
	vector.DrawFilledRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), bg, false)
	vector.StrokeRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), 2, border, false)

	// Step indicator
	stepText := locale.Tf("ftut.step_fmt", ft.Step+1, 10)
	printColoredAt(screen, stepText, boxX+10, boxY+6, color.RGBA{80, 180, 255, 200})

	// Step text (3 lines max)
	key := fmt.Sprintf("ftut.%d", ft.Step)
	txt := locale.T(key)
	lines := factoryTutSplit(txt)
	for i, line := range lines {
		if i >= 3 {
			break
		}
		printColoredAt(screen, line, boxX+15, boxY+24+i*lineH, color.RGBA{220, 230, 240, 255})
	}

	// Next button
	btnY := boxY + boxH - 26
	btnW := 80
	btnX := boxX + boxW - btnW - 10
	mx, my := ebiten.CursorPosition()
	hovered := mx >= btnX && mx < btnX+btnW && my >= btnY && my < btnY+22
	btnCol := color.RGBA{40, 120, 200, 255}
	if hovered {
		btnCol = color.RGBA{60, 150, 230, 255}
	}
	vector.DrawFilledRect(screen, float32(btnX), float32(btnY), float32(btnW), 22, btnCol, false)
	printColoredAt(screen, locale.T("ftut.next"), btnX+10, btnY+4, color.RGBA{220, 230, 240, 255})

	// Skip button
	skipW := 80
	skipX := boxX + 10
	vector.DrawFilledRect(screen, float32(skipX), float32(btnY), float32(skipW), 22, color.RGBA{60, 30, 30, 200}, false)
	printColoredAt(screen, locale.T("ftut.skip"), skipX+8, btnY+4, color.RGBA{140, 145, 160, 255})
}

// FactoryTutHitTest checks if "Next" or "Skip" was clicked.
// Returns "next", "skip", or "".
func FactoryTutHitTest(mx, my, sw, sh int, ft *factory.FactoryTutorial) string {
	if ft == nil || !ft.Active || ft.Step >= 10 {
		return ""
	}

	boxW := 650
	boxH := 90
	boxX := (sw - boxW) / 2
	boxY := sh - boxH - 30
	btnY := boxY + boxH - 26

	// Skip button
	if mx >= boxX+10 && mx < boxX+10+80 && my >= btnY && my < btnY+22 {
		return "skip"
	}

	// Next button
	btnW := 80
	btnX := boxX + boxW - btnW - 10
	if mx >= btnX && mx < btnX+btnW && my >= btnY && my < btnY+22 {
		return "next"
	}

	return ""
}

// factoryTutSplit splits a string on '|' for multi-line display.
func factoryTutSplit(s string) []string {
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

// drawFactoryQuickStats renders a thin persistent stats bar at the very top of
// the factory screen, always visible regardless of HUD/panel scroll state.
func drawFactoryQuickStats(screen *ebiten.Image, fs *factory.FactoryState, sw int) {
	barH := 14

	// Semi-transparent dark background
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(barH), color.RGBA{15, 18, 25, 200}, false)

	speedStr := fmt.Sprintf("%.0fx", fs.Speed)
	if fs.Speed < 1.0 {
		speedStr = fmt.Sprintf("%.3gx", fs.Speed)
	}

	stats := fmt.Sprintf("Bots:%s | Tick:%s | Speed:%s | Budget:$%.0f | Parts:%s",
		fmtNum(fs.BotCount), fmtNum(fs.Tick), speedStr, fs.Budget, fmtNum(fs.Stats.PartsProcessed))

	// OEE approximation: working bots / total bots
	if fs.BotCount > 0 {
		oee := fs.Stats.BotsWorking * 100 / fs.BotCount
		stats += fmt.Sprintf(" | OEE:%d%%", oee)
	}

	// Orders
	completed := fs.CompletedOrders
	total := completed + len(fs.Orders)
	stats += fmt.Sprintf(" | Orders:%d/%d", completed, total)

	dimCol := color.RGBA{130, 140, 160, 200}
	printColoredAt(screen, stats, 6, 2, dimCol)
}
