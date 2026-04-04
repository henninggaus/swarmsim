package render

import (
	"image/color"
	"math"
	"swarmsim/domain/factory"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func drawSparkEffects(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32) {

	alive := fs.SparkEffects[:0]
	for _, sp := range fs.SparkEffects {
		sp.Tick++
		if sp.Tick > 15 {
			continue
		}
		alive = append(alive, sp)
		alpha := uint8(255 * (1.0 - float64(sp.Tick)/15.0))
		for s := 0; s < 4; s++ {
			sx := sp.X + sp.VX[s]*float64(sp.Tick)
			sy := sp.Y + sp.VY[s]*float64(sp.Tick)
			vector.DrawFilledCircle(screen, wx(sx), wy(sy), ws(1.5),
				color.RGBA{255, 220, 40, alpha}, false)
		}
	}
	fs.SparkEffects = alive
}

// Machine finish FX (big pulse ring)
func drawMachineFinishFX(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32) {

	alive := fs.MachineFinishFX[:0]
	for _, fx := range fs.MachineFinishFX {
		fx.Tick++
		if fx.Tick > 30 {
			continue
		}
		alive = append(alive, fx)
		frac := float64(fx.Tick) / 30.0
		r := 60.0 * frac
		alpha := uint8(220 * (1.0 - frac))
		vector.StrokeCircle(screen, wx(fx.X), wy(fx.Y), ws(r), 3,
			color.RGBA{100, 255, 180, alpha}, false)
		// Inner ring
		r2 := 30.0 * frac
		alpha2 := uint8(180 * (1.0 - frac))
		vector.StrokeCircle(screen, wx(fx.X), wy(fx.Y), ws(r2), 2,
			color.RGBA{255, 255, 200, alpha2}, false)
	}
	fs.MachineFinishFX = alive
}

// Truck arrival flash (orange glow at top of screen)
func drawTruckArriveFlash(screen *ebiten.Image, fs *factory.FactoryState, sw int) {
	t := fs.TruckArriveFlash.Tick
	if t >= 20 {
		return
	}
	alpha := uint8(80 * (1.0 - float64(t)/20.0))
	vector.DrawFilledRect(screen, 0, 0, float32(sw), 6, color.RGBA{255, 160, 40, alpha}, false)
}

// ============================================================================
// [10] Selected Bot Path Trail
// ============================================================================
func drawAisleFlowParticles(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	visMinX, visMaxX float64, tick int) {

	aisleStartX := factory.HallX + 20
	aisleEndX := factory.HallX + factory.HallW - 20
	aisleLen := aisleEndX - aisleStartX
	aisleMidY := factory.AisleY + factory.AisleH/2

	particleCount := 20
	speed := 0.003
	flowCol := color.RGBA{160, 200, 240, 50}

	for p := 0; p < particleCount; p++ {
		phase := math.Mod(float64(tick)*speed+float64(p)/float64(particleCount), 1.0)
		px := aisleStartX + phase*aisleLen
		if px < visMinX || px > visMaxX {
			continue
		}
		py := aisleMidY + 4*math.Sin(phase*math.Pi*4+float64(p))
		edgeFade := 1.0
		if phase < 0.05 {
			edgeFade = phase / 0.05
		} else if phase > 0.95 {
			edgeFade = (1.0 - phase) / 0.05
		}
		alpha := uint8(float64(flowCol.A) * edgeFade)
		vector.DrawFilledCircle(screen, wx(px), wy(py), ws(2.5), color.RGBA{flowCol.R, flowCol.G, flowCol.B, alpha}, false)
	}
}

// drawFactoryHUD renders the info panel at top-left corner.
func drawAnimatedDashedLine(screen *ebiten.Image, x1, y1, x2, y2 float32, col color.RGBA, tick int) {
	dx := x2 - x1
	dy := y2 - y1
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length < 1 {
		return
	}
	dashLen := float32(8.0)
	gapLen := float32(5.0)
	segLen := dashLen + gapLen
	offset := float32(math.Mod(float64(tick)*0.5, float64(segLen)))
	ux := dx / length
	uy := dy / length
	for d := -offset; d < length; d += segLen {
		start := d
		if start < 0 {
			start = 0
		}
		end := d + dashLen
		if end > length {
			end = length
		}
		if start >= end {
			continue
		}
		sx := x1 + ux*start
		sy := y1 + uy*start
		ex := x1 + ux*end
		ey := y1 + uy*end
		vector.StrokeLine(screen, sx, sy, ex, ey, 1.5, col, false)
	}
}

// --- Truck exhaust particles ---
func drawTruckExhaust(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32,
	tick int) {

	alive := fs.ExhaustParticles[:0]
	for i := range fs.ExhaustParticles {
		p := &fs.ExhaustParticles[i]
		p.Life--
		drift := p.DriftY
		if drift == 0 {
			drift = -0.3
		}
		p.Y += drift
		p.X += (float64(tick+i*7) * 0.01)
		if p.Life > 0 {
			alive = append(alive, *p)
		}
	}
	fs.ExhaustParticles = alive

	for _, truck := range fs.Trucks {
		isMoving := truck.Phase == factory.TruckEntering || truck.Phase == factory.TruckExiting
		if !isMoving {
			continue
		}

		// Feature 8: Thicker exhaust on acceleration (first 30 ticks of movement)
		accelerating := truck.MoveTick < 30

		// Feature 8: Brake detection — entering truck approaching dock
		braking := false
		if truck.Phase == factory.TruckEntering && truck.DockIdx >= 0 && truck.DockIdx < len(fs.Docks) {
			dock := &fs.Docks[truck.DockIdx]
			targetX := dock.X + dock.W/2 - truck.W/2
			distToDock := math.Abs(truck.X - targetX)
			braking = distToDock < 150
		}

		// Determine exhaust position
		var ex float64
		if truck.Direction == 0 {
			ex = truck.X - 2
		} else {
			ex = truck.X + truck.W + 2
		}
		ey := truck.Y + truck.H*0.8

		if accelerating {
			// Spawn MORE particles (5 per tick), darker and larger, with upward drift
			if tick%2 == 0 {
				for p := 0; p < 5; p++ {
					offsetX := float64(p-2) * 1.5
					fs.ExhaustParticles = append(fs.ExhaustParticles, factory.ExhaustParticle{
						X: ex + offsetX, Y: ey - float64(p)*0.5,
						Life: 40, Alpha: 200,
						Size: 3.5, DriftY: -0.5,
					})
				}
			}
		} else {
			// Normal exhaust: 1 particle every 3 ticks
			if tick%3 == 0 {
				fs.ExhaustParticles = append(fs.ExhaustParticles, factory.ExhaustParticle{
					X: ex, Y: ey, Life: 30, Alpha: 120,
					Size: 2.0, DriftY: -0.3,
				})
			}
		}

		// Feature 8: Brake light flash when approaching dock
		if braking {
			// Draw red brake lights at rear of truck
			var blx float64
			if truck.Direction == 0 {
				blx = truck.X - 3
			} else {
				blx = truck.X + truck.W + 3
			}
			bly1 := truck.Y + truck.H*0.3
			bly2 := truck.Y + truck.H*0.7
			brakeAlpha := uint8(180 + int(40*math.Sin(float64(tick)*0.5)))
			brakeCol := color.RGBA{255, 30, 20, brakeAlpha}
			vector.DrawFilledCircle(screen, wx(blx), wy(bly1), ws(3), brakeCol, false)
			vector.DrawFilledCircle(screen, wx(blx), wy(bly2), ws(3), brakeCol, false)
		}
	}

	for _, p := range fs.ExhaustParticles {
		maxLife := float64(30)
		if p.Size > 2.5 {
			maxLife = 40
		}
		fade := uint8(float64(p.Alpha) * float64(p.Life) / maxLife)
		baseR := p.Size
		if baseR <= 0 {
			baseR = 2.0
		}
		age := maxLife - float64(p.Life)
		r := baseR + age*0.1
		// Darker color for larger (acceleration) particles
		gray := uint8(160)
		if baseR > 2.5 {
			gray = 100
		}
		vector.DrawFilledCircle(screen, wx(p.X), wy(p.Y), ws(r), color.RGBA{gray, gray, gray + 10, fade}, false)
	}
}

// --- Pulse effects (pickup/delivery expanding rings) ---
func drawPulseEffects(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32) {

	alive := fs.PulseEffects[:0]
	for i := range fs.PulseEffects {
		p := &fs.PulseEffects[i]
		p.Tick++
		if p.Tick > 20 {
			continue
		}
		alive = append(alive, *p)
		frac := float64(p.Tick) / 20.0
		r := p.MaxR * frac
		alpha := uint8(200 * (1.0 - frac))
		col := color.RGBA{p.Color[0], p.Color[1], p.Color[2], alpha}
		vector.StrokeCircle(screen, wx(p.X), wy(p.Y), ws(r), 2, col, false)
	}
	fs.PulseEffects = alive
}

// --- Hover tooltip ---
func drawChargingLightning(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32, tick int) {

	// Only show every other 5-tick period (flicker)
	if (tick/5)%2 != 0 {
		return
	}

	for ci := range fs.Chargers {
		ch := &fs.Chargers[ci]
		chCX := ch.X + ch.W/2
		chCY := ch.Y + ch.H/2

		for _, botIdx := range ch.Occupants {
			if botIdx < 0 || botIdx >= len(fs.Bots) {
				continue
			}
			bot := &fs.Bots[botIdx]
			if bot.State != factory.BotCharging {
				continue
			}

			// Draw zigzag line from charger center to bot position
			sx := wx(chCX)
			sy := wy(chCY)
			ex := wx(bot.X)
			ey := wy(bot.Y)

			// Number of zigzag segments
			segments := 5
			dx := (ex - sx) / float32(segments)
			dy := (ey - sy) / float32(segments)

			boltCol := color.RGBA{80, 255, 255, 180}
			if tick%10 < 5 {
				boltCol = color.RGBA{255, 255, 80, 200}
			}

			px, py := sx, sy
			for s := 1; s <= segments; s++ {
				nx := sx + dx*float32(s)
				ny := sy + dy*float32(s)
				if s < segments {
					// Add perpendicular offset for zigzag
					perpX := -dy / float32(segments)
					perpY := dx / float32(segments)
					offset := float32(3.0)
					if s%2 == 0 {
						offset = -offset
					}
					nx += perpX * offset
					ny += perpY * offset
				}
				vector.StrokeLine(screen, px, py, nx, ny, 1.5, boltCol, false)
				px, py = nx, ny
			}
		}
	}
}

// ============================================================================
// [NEW-3] Production Chain Visualization (input/output flow dots on machines)
// ============================================================================
func drawProductionChainFlow(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32, tick int) {

	if fs.CamZoom < 0.5 {
		return // only at reasonable zoom
	}

	for i := range fs.Machines {
		m := &fs.Machines[i]
		if !m.Active || m.ProcessTime <= 0 {
			continue
		}

		progress := 1.0 - float64(m.ProcessTimer)/float64(m.ProcessTime)

		// Input dots flowing in from the left
		inputCol := partColorToRGBA(m.InputColor)
		inputCol.A = 160
		for d := 0; d < 3; d++ {
			phase := math.Mod(progress+float64(d)*0.33, 1.0)
			dotX := m.X - 20 + phase*20
			dotY := m.Y + m.H*0.3 + float64(d)*8
			alpha := uint8(160 * (1.0 - phase))
			vector.DrawFilledCircle(screen, wx(dotX), wy(dotY), ws(2.5),
				color.RGBA{inputCol.R, inputCol.G, inputCol.B, alpha}, false)
		}

		// Output dots flowing out to the right
		outputCol := partColorToRGBA(m.OutputColor)
		outputCol.A = 160
		for d := 0; d < 3; d++ {
			phase := math.Mod(progress+float64(d)*0.33, 1.0)
			dotX := m.X + m.W + phase*20
			dotY := m.Y + m.H*0.3 + float64(d)*8
			alpha := uint8(160 * phase)
			vector.DrawFilledCircle(screen, wx(dotX), wy(dotY), ws(2.5),
				color.RGBA{outputCol.R, outputCol.G, outputCol.B, alpha}, false)
		}
	}
}

// ============================================================================
// [NEW-4] Bot Task Preview on Hover (route preview line)
// ============================================================================
func drawBotHoverTaskPreview(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, tick int) {

	if fs.HoverBot < 0 || fs.HoverBot >= len(fs.Bots) {
		return
	}
	// Don't draw if it's also the selected bot (selected bot already has task lines)
	if fs.HoverBot == fs.SelectedBot {
		return
	}

	bot := &fs.Bots[fs.HoverBot]
	taskIdx := factory.FindBotTask(fs, fs.HoverBot)
	if taskIdx < 0 {
		return
	}

	task := &fs.Tasks.Tasks[taskIdx]
	bx := wx(bot.X)
	by := wy(bot.Y)

	// Line from bot to source (cyan, thin)
	if bot.State == factory.BotMovingToSource || bot.State == factory.BotIdle || bot.State == factory.BotPickingUp {
		srcX := wx(task.SourceX)
		srcY := wy(task.SourceY)
		vector.StrokeLine(screen, bx, by, srcX, srcY, 1, color.RGBA{0, 200, 220, 100}, false)
		// Source to dest (yellow, thin)
		dstX := wx(task.DestX)
		dstY := wy(task.DestY)
		vector.StrokeLine(screen, srcX, srcY, dstX, dstY, 1, color.RGBA{220, 200, 40, 80}, false)
	} else if bot.State == factory.BotMovingToDest || bot.State == factory.BotDelivering {
		dstX := wx(task.DestX)
		dstY := wy(task.DestY)
		vector.StrokeLine(screen, bx, by, dstX, dstY, 1, color.RGBA{220, 200, 40, 100}, false)
	}
}

// ============================================================================
// [NEW-5] Loading Dock Animations — Truck Cargo Doors
// ============================================================================
func drawDockCargoAnimation(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32, tick int) {

	for _, truck := range fs.Trucks {
		if truck.DockIdx < 0 || truck.DockIdx >= len(fs.Docks) {
			continue
		}
		if truck.Phase != factory.TruckUnloading && truck.Phase != factory.TruckLoading {
			continue
		}

		dock := &fs.Docks[truck.DockIdx]

		// Cargo door animation: two rectangles sliding apart
		doorW := dock.W * 0.35
		doorH := float64(8)
		doorY := dock.Y + dock.H - doorH - 2
		doorCenterX := dock.X + dock.W/2

		// Door opening progress (fully open after ~40 ticks at dock)
		openProgress := float64(truck.Counter) / 40.0
		if openProgress > 1 {
			openProgress = 1
		}
		slideOffset := doorW * 0.6 * openProgress

		doorCol := color.RGBA{80, 70, 50, 200}
		// Left door
		vector.DrawFilledRect(screen, wx(doorCenterX-doorW-slideOffset), wy(doorY),
			ws(doorW), ws(doorH), doorCol, false)
		// Right door
		vector.DrawFilledRect(screen, wx(doorCenterX+slideOffset), wy(doorY),
			ws(doorW), ws(doorH), doorCol, false)

		// Show remaining packages as colored squares inside truck area
		if len(truck.Parts) > 0 {
			pkgSize := 4.0
			pkgCols := 5 // columns of packages
			for pi, pc := range truck.Parts {
				if pi >= 20 {
					break
				}
				pr := pi / pkgCols
				pcol := pi % pkgCols
				ppx := truck.X + 8 + float64(pcol)*(pkgSize+2)
				ppy := truck.Y + 6 + float64(pr)*(pkgSize+2)
				pColor := partColorToRGBA(pc)
				pColor.A = 180
				vector.DrawFilledRect(screen, wx(ppx), wy(ppy), ws(pkgSize), ws(pkgSize), pColor, false)
			}
		}

		// Animated packages moving between truck and dock
		if openProgress > 0.5 {
			numAnimPkgs := 2
			for p := 0; p < numAnimPkgs; p++ {
				phase := math.Mod(float64(tick%60)/60.0+float64(p)*0.5, 1.0)
				var startY, endY float64
				if truck.Phase == factory.TruckUnloading {
					startY = truck.Y + truck.H
					endY = dock.Y + dock.H + 20
				} else {
					startY = dock.Y + dock.H + 20
					endY = truck.Y + truck.H
				}
				pkgY := startY + (endY-startY)*phase
				pkgX := doorCenterX + float64(p-1)*10
				alpha := uint8(200 * (1.0 - math.Abs(phase-0.5)*2))
				vector.DrawFilledRect(screen, wx(pkgX-2), wy(pkgY-2), ws(5), ws(5),
					color.RGBA{220, 180, 80, alpha}, false)
			}
		}
	}
}

// ============================================================================
// [NEW-6] Floor Safety Zone Markings
// ============================================================================
func drawFloorSafetyZones(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32) {

	safetyAlpha := uint8(30)

	// Red/white striped zones around machines (danger area)
	for _, m := range fs.Machines {
		margin := 15.0
		// Red danger zone
		vector.DrawFilledRect(screen,
			wx(m.X-margin), wy(m.Y-margin),
			ws(m.W+margin*2), ws(m.H+margin*2),
			color.RGBA{200, 40, 40, safetyAlpha}, false)
		// Striped corner marks
		stripeCol := color.RGBA{200, 40, 40, safetyAlpha + 15}
		vector.StrokeLine(screen, wx(m.X-margin), wy(m.Y-margin),
			wx(m.X-margin+10), wy(m.Y-margin), 1, stripeCol, false)
		vector.StrokeLine(screen, wx(m.X-margin), wy(m.Y-margin),
			wx(m.X-margin), wy(m.Y-margin+10), 1, stripeCol, false)
		vector.StrokeLine(screen, wx(m.X+m.W+margin), wy(m.Y+m.H+margin),
			wx(m.X+m.W+margin-10), wy(m.Y+m.H+margin), 1, stripeCol, false)
		vector.StrokeLine(screen, wx(m.X+m.W+margin), wy(m.Y+m.H+margin),
			wx(m.X+m.W+margin), wy(m.Y+m.H+margin-10), 1, stripeCol, false)
	}

	// Blue marked zones at chargers
	for _, ch := range fs.Chargers {
		margin := 10.0
		vector.DrawFilledRect(screen,
			wx(ch.X-margin), wy(ch.Y-margin),
			ws(ch.W+margin*2), ws(ch.H+margin*2),
			color.RGBA{40, 80, 200, safetyAlpha}, false)
	}

	// Green marked safe zones at parking areas
	for _, pz := range fs.ParkingZones {
		pw := 70.0
		ph := 50.0
		vector.DrawFilledRect(screen,
			wx(pz[0]-pw/2), wy(pz[1]-ph/2),
			ws(pw), ws(ph),
			color.RGBA{40, 180, 40, safetyAlpha}, false)
	}
}

// ============================================================================
// [NEW-7] Efficiency Sparkline (50px wide, 20px tall)
// ============================================================================
func drawAmbientNoiseViz(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32, tick int) {

	// Machines emit small vibration circles when active
	for _, m := range fs.Machines {
		if !m.Active {
			continue
		}
		cx := m.X + m.W/2
		cy := m.Y + m.H/2

		// Two concentric rings, expanding based on tick
		for ring := 0; ring < 2; ring++ {
			phase := math.Mod(float64(tick)*0.06+float64(ring)*0.5, 1.0)
			r := m.W*0.6 + phase*m.W*0.3
			alpha := uint8(25 * (1.0 - phase))
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(r), 1,
				color.RGBA{120, 140, 180, alpha}, false)
		}
	}

	// Trucks emit larger rumble circles when driving
	for _, truck := range fs.Trucks {
		if truck.Phase != factory.TruckEntering && truck.Phase != factory.TruckExiting {
			continue
		}
		cx := truck.X + truck.W/2
		cy := truck.Y + truck.H/2

		for ring := 0; ring < 3; ring++ {
			phase := math.Mod(float64(tick)*0.04+float64(ring)*0.33, 1.0)
			r := 20.0 + phase*30.0
			alpha := uint8(20 * (1.0 - phase))
			vector.StrokeCircle(screen, wx(cx), wy(cy), ws(r), 1,
				color.RGBA{140, 140, 150, alpha}, false)
		}
	}
}

// ============================================================================
// [NEW-10] Achievement Popup
// ============================================================================
func drawQCRejectEffects(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32, tick int) {
	alive := fs.QCRejectEffects[:0]
	for _, eff := range fs.QCRejectEffects {
		age := tick - eff.Tick
		if age > 60 {
			continue // expired
		}
		alive = append(alive, eff)
		alpha := uint8(255 - age*4)
		cx := wx(eff.X)
		cy := wy(eff.Y)
		sz := ws(15)
		// Draw red X
		vector.StrokeLine(screen, cx-sz, cy-sz, cx+sz, cy+sz, 3, color.RGBA{255, 40, 40, alpha}, false)
		vector.StrokeLine(screen, cx+sz, cy-sz, cx-sz, cy+sz, 3, color.RGBA{255, 40, 40, alpha}, false)
	}
	fs.QCRejectEffects = alive
}

// ============================================================================
// Feature 7: Stock Warning Borders — pulsing orange border on low-stock storage
// ============================================================================
func drawStockWarningBorders(screen *ebiten.Image, fs *factory.FactoryState,
	wx func(float64) float32, wy func(float64) float32, ws func(float64) float32, tick int) {

	// Pulsing orange border on inbound storage when stock is low
	if fs.StockWarning && fs.InboundStorageIdx < len(fs.Storage) {
		st := &fs.Storage[fs.InboundStorageIdx]
		pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.12)
		alpha := uint8(100 + pulse*155)
		vector.StrokeRect(screen, wx(st.X)-2, wy(st.Y)-2, ws(st.W)+4, ws(st.H)+4,
			3, color.RGBA{255, 160, 40, alpha}, false)
	}

	// Pulsing red border on outbound storage when full
	if fs.OutboundFull && fs.OutboundStorageIdx < len(fs.Storage) {
		st := &fs.Storage[fs.OutboundStorageIdx]
		pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.12)
		alpha := uint8(100 + pulse*155)
		vector.StrokeRect(screen, wx(st.X)-2, wy(st.Y)-2, ws(st.W)+4, ws(st.H)+4,
			3, color.RGBA{255, 60, 40, alpha}, false)
	}
}

// ============================================================================
// Feature 10: Maintenance Planner Overlay — toggled with P key
// ============================================================================
