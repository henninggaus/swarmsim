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

func drawProductionDashboard(screen *ebiten.Image, fs *factory.FactoryState, sw, sh, tick int) {
	dY := sh - dashboardH
	dW := sw

	// Background
	vector.DrawFilledRect(screen, 0, float32(dY), float32(dW), float32(dashboardH),
		color.RGBA{12, 14, 22, 235}, false)
	vector.StrokeLine(screen, 0, float32(dY), float32(dW), float32(dY), 1,
		color.RGBA{50, 60, 90, 200}, false)

	padX := 12
	padY := dY + 8

	// --- Section 1: Throughput Graph (leftmost, 280px wide) ---
	graphW := 280
	graphH := dashboardH - 30
	drawThroughputGraph(screen, fs, padX, padY, graphW, graphH, tick)

	// --- Section 2: Machine Status (row of colored boxes) ---
	msX := padX + graphW + 20
	drawMachineStatusBoxes(screen, fs, msX, padY, tick)

	// --- Section 3: Energy Distribution ---
	edX := msX + 200
	drawEnergyDistribution(screen, fs, edX, padY)

	// --- Section 4: LKW Status ---
	lkwX := edX + 160
	drawLKWStatus(screen, fs, lkwX, padY, tick)

	// --- Section 5: Production Rate & Efficiency ---
	prX := lkwX + 140
	drawProductionStats(screen, fs, prX, padY, tick)

	// --- [NEW-7] Efficiency Sparkline ---
	drawEfficiencySparkline(screen, fs, prX, padY+lineH*5+8)

	// --- Section 6: KPI Dashboard ---
	kpiX := prX + 180
	drawKPIDashboard(screen, fs, kpiX, padY, tick)
}

func drawThroughputGraph(screen *ebiten.Image, fs *factory.FactoryState,
	x, y, w, h, tick int) {

	// Title
	printColoredAt(screen, locale.T("factory.dash.throughput"), x, y, color.RGBA{200, 210, 230, 200})

	gy := y + lineH + 2
	gh := h - lineH - 4

	// Graph background
	vector.DrawFilledRect(screen, float32(x), float32(gy), float32(w), float32(gh),
		color.RGBA{8, 10, 18, 200}, false)
	vector.StrokeRect(screen, float32(x), float32(gy), float32(w), float32(gh),
		1, color.RGBA{40, 50, 70, 150}, false)

	if len(fs.ThroughputHistory) == 0 {
		return
	}

	// Find max for scaling
	maxVal := 1
	for _, v := range fs.ThroughputHistory {
		if v > maxVal {
			maxVal = v
		}
	}

	// Draw line chart
	n := len(fs.ThroughputHistory)
	barW := float64(w) / float64(n)

	for i := 0; i < n; i++ {
		// Read from ring buffer in order
		idx := (fs.ThroughputIdx + i) % n
		val := fs.ThroughputHistory[idx]
		if val <= 0 {
			continue
		}
		frac := float64(val) / float64(maxVal)
		barH := frac * float64(gh-2)
		bx := float64(x) + float64(i)*barW
		by := float64(gy+gh-1) - barH

		col := color.RGBA{40, 180, 220, 160}
		if frac > 0.8 {
			col = color.RGBA{80, 255, 120, 160}
		}
		vector.DrawFilledRect(screen, float32(bx), float32(by), float32(barW-1), float32(barH), col, false)
	}
}

func drawMachineStatusBoxes(screen *ebiten.Image, fs *factory.FactoryState,
	x, y, tick int) {

	printColoredAt(screen, locale.T("factory.dash.machines"), x, y, color.RGBA{200, 210, 230, 200})
	by := y + lineH + 4
	boxSize := 18
	gap := 4
	names := []string{"CNC1", "CNC2", "ASSY", "DR1", "DR2", "QC"}

	for i, m := range fs.Machines {
		bx := x + i*(boxSize+gap)
		col := color.RGBA{80, 80, 90, 200} // idle gray

		if !m.Active && m.CurrentInput == 0 && !m.OutputReady {
			// Disabled / truly idle
			col = color.RGBA{60, 60, 70, 200}
		} else if m.Active && m.ProcessTimer > 0 {
			// Processing = green
			col = color.RGBA{40, 200, 80, 220}
		} else if m.OutputReady {
			// Output ready = cyan
			col = color.RGBA{40, 200, 220, 220}
		}

		vector.DrawFilledRect(screen, float32(bx), float32(by), float32(boxSize), float32(boxSize), col, false)
		vector.StrokeRect(screen, float32(bx), float32(by), float32(boxSize), float32(boxSize), 1,
			color.RGBA{100, 110, 130, 180}, false)

		// Label below
		if i < len(names) {
			lx := bx + boxSize/2 - runeLen(names[i])*charW/2
			printColoredAt(screen, names[i], lx, by+boxSize+2, colorHUDDim)
		}
	}
}

func drawEnergyDistribution(screen *ebiten.Image, fs *factory.FactoryState, x, y int) {
	printColoredAt(screen, locale.T("factory.dash.energy"), x, y, color.RGBA{200, 210, 230, 200})
	by := y + lineH + 4

	// Bucket bots into energy ranges: 0-25, 25-50, 50-75, 75-100
	var buckets [4]int
	for i := range fs.Bots {
		e := fs.Bots[i].Energy
		switch {
		case e < 25:
			buckets[0]++
		case e < 50:
			buckets[1]++
		case e < 75:
			buckets[2]++
		default:
			buckets[3]++
		}
	}

	total := len(fs.Bots)
	if total == 0 {
		total = 1
	}

	barW := 130
	barH := 14
	labels := []string{"0-25", "25-50", "50-75", "75+"}
	colors := []color.RGBA{
		{220, 60, 60, 200},
		{220, 200, 60, 200},
		{100, 200, 60, 200},
		{60, 220, 80, 200},
	}

	for i := 0; i < 4; i++ {
		ly := by + i*(barH+3)
		frac := float64(buckets[i]) / float64(total)
		vector.DrawFilledRect(screen, float32(x), float32(ly), float32(barW), float32(barH),
			color.RGBA{25, 28, 40, 200}, false)
		vector.DrawFilledRect(screen, float32(x), float32(ly), float32(float64(barW)*frac), float32(barH),
			colors[i], false)
		printColoredAt(screen, labels[i], x+2, ly+1, color.RGBA{220, 220, 230, 200})
		pctStr := fmt.Sprintf("%d", buckets[i])
		printColoredAt(screen, pctStr, x+barW+4, ly+1, colorHUDDim)
	}
}

func drawLKWStatus(screen *ebiten.Image, fs *factory.FactoryState, x, y, tick int) {
	printColoredAt(screen, locale.T("factory.dash.lkw"), x, y, color.RGBA{200, 210, 230, 200})
	by := y + lineH + 4

	for i, truck := range fs.Trucks {
		if i >= 6 {
			break
		}
		ty := by + i*(lineH+2)

		var label string
		var col color.RGBA
		switch truck.Phase {
		case factory.TruckEntering:
			label = locale.T("factory.truck.driving")
			col = color.RGBA{80, 200, 220, 220}
		case factory.TruckUnloading:
			label = locale.T("factory.truck.unload")
			col = color.RGBA{220, 160, 60, 220}
		case factory.TruckLoading:
			label = locale.T("factory.truck.load")
			col = color.RGBA{60, 200, 80, 220}
		case factory.TruckExiting:
			label = locale.T("factory.truck.depart")
			col = color.RGBA{160, 160, 170, 180}
		case factory.TruckParked:
			label = locale.T("factory.truck.parked")
			col = color.RGBA{140, 140, 150, 200}
		default:
			label = locale.T("factory.truck.wait")
			col = color.RGBA{100, 100, 110, 150}
		}

		// Truck icon (tiny rectangle)
		vector.DrawFilledRect(screen, float32(x), float32(ty), 12, float32(lineH-2), col, false)
		printColoredAt(screen, label, x+16, ty, col)
	}

	if len(fs.Trucks) == 0 {
		printColoredAt(screen, locale.T("factory.truck.none"), x, by, colorHUDDim)
	}
}

func drawProductionStats(screen *ebiten.Image, fs *factory.FactoryState, x, y, tick int) {
	printColoredAt(screen, locale.T("factory.dash.production"), x, y, color.RGBA{200, 210, 230, 200})
	by := y + lineH + 4

	// Production rate (parts/minute approximation)
	// Use last throughput sample * 60/100 to approximate per-minute
	rate := 0
	if len(fs.ThroughputHistory) > 0 && fs.ThroughputIdx > 0 {
		lastIdx := (fs.ThroughputIdx - 1) % len(fs.ThroughputHistory)
		rate = fs.ThroughputHistory[lastIdx] * 60 / 100 // crude per-minute
	}
	printColoredAt(screen, locale.Tf("factory.dash.rate", rate), x, by, colorHUDText)
	by += lineH

	// Efficiency: working bots / (working + idle)
	working := fs.Stats.BotsWorking
	idle := fs.Stats.BotsIdle
	total := working + idle
	if total == 0 {
		total = 1
	}
	eff := float64(working) / float64(total) * 100
	effCol := color.RGBA{80, 220, 80, 220}
	if eff < 50 {
		effCol = color.RGBA{220, 60, 60, 220}
	} else if eff < 75 {
		effCol = color.RGBA{220, 200, 60, 220}
	}
	printColoredAt(screen, locale.Tf("factory.dash.efficiency", eff), x, by, effCol)
	by += lineH

	// Total processed
	printColoredAt(screen, locale.Tf("factory.dash.total", fs.Stats.PartsProcessed), x, by, colorHUDDim)
	by += lineH

	// Trucks unloaded/loaded
	printColoredAt(screen, locale.Tf("factory.dash.inout", fs.Stats.TrucksUnloaded, fs.Stats.TrucksLoaded), x, by, colorHUDDim)
}

// ============================================================================
// KPI Dashboard (OEE, Cycle Time, Bottleneck, WIP)
// ============================================================================
func drawKPIDashboard(screen *ebiten.Image, fs *factory.FactoryState, x, y, tick int) {
	printColoredAt(screen, locale.T("factory.dash.kpi"), x, y, color.RGBA{200, 210, 230, 200})
	by := y + lineH + 4

	// OEE = Availability x Performance x Quality
	availability := 0.0
	if fs.Tick > 0 {
		totalUptime := 0
		for i := 0; i < len(fs.Stats.MachineUptime) && i < len(fs.Machines); i++ {
			totalUptime += fs.Stats.MachineUptime[i]
		}
		nMachines := len(fs.Machines)
		if nMachines == 0 {
			nMachines = 1
		}
		availability = float64(totalUptime) / float64(fs.Tick*nMachines)
		if availability > 1 {
			availability = 1
		}
	}

	// Performance: actual output vs theoretical max
	performance := 0.0
	if fs.Tick > 0 {
		// Theoretical max: sum of 1/processTime for each machine * tick
		theoreticalMax := 0.0
		for i := range fs.Machines {
			if fs.Machines[i].ProcessTime > 0 {
				theoreticalMax += float64(fs.Tick) / float64(fs.Machines[i].ProcessTime)
			}
		}
		if theoreticalMax > 0 {
			performance = float64(fs.Stats.TotalParts) / theoreticalMax
		}
		if performance > 1 {
			performance = 1
		}
	}

	// Quality: good parts / total parts
	quality := 1.0
	if fs.Stats.TotalParts > 0 {
		quality = float64(fs.Stats.GoodParts) / float64(fs.Stats.TotalParts)
	}

	oee := availability * performance * quality * 100
	oeeCol := color.RGBA{80, 220, 80, 220}
	if oee < 50 {
		oeeCol = color.RGBA{220, 60, 60, 220}
	} else if oee < 75 {
		oeeCol = color.RGBA{220, 200, 60, 220}
	}
	printColoredAt(screen, locale.Tf("factory.dash.oee", oee), x, by, oeeCol)
	by += lineH

	// Quality rate
	qCol := color.RGBA{80, 220, 80, 220}
	if quality < 0.9 {
		qCol = color.RGBA{220, 200, 60, 220}
	}
	printColoredAt(screen, locale.Tf("factory.dash.quality", quality*100), x, by, qCol)
	by += lineH

	// Defect count
	printColoredAt(screen, locale.Tf("factory.dash.defects", fs.Stats.DefectCount), x, by, color.RGBA{200, 0, 200, 220})
	by += lineH

	// WIP (work in progress)
	printColoredAt(screen, locale.Tf("factory.dash.wip", fs.Stats.WIP), x, by, colorHUDDim)
	by += lineH

	// Bottleneck detection: machine with the most queued input
	bottleneckIdx := -1
	maxQueue := 0
	for i := range fs.Machines {
		if fs.Machines[i].CurrentInput > maxQueue {
			maxQueue = fs.Machines[i].CurrentInput
			bottleneckIdx = i
		}
	}
	kpiMachNames := []string{"CNC1", "CNC2", "ASSY", "DR1", "DR2", "QC"}
	if bottleneckIdx >= 0 && bottleneckIdx < len(kpiMachNames) {
		printColoredAt(screen, locale.Tf("factory.dash.bottleneck", kpiMachNames[bottleneckIdx], maxQueue), x, by, color.RGBA{255, 100, 60, 220})
	} else {
		printColoredAt(screen, locale.T("factory.dash.bottl_none"), x, by, colorHUDDim)
	}
	by += lineH

	// Shift info
	shiftPct := 0
	for i := range fs.ShiftOnDuty {
		if i < len(fs.Bots) && fs.ShiftOnDuty[i] {
			shiftPct++
		}
	}
	if len(fs.Bots) > 0 {
		shiftPct = shiftPct * 100 / len(fs.Bots)
	}
	printColoredAt(screen, locale.Tf("factory.dash.shift", shiftPct), x, by, colorHUDDim)
}

// ============================================================================
// [8] Visual Feedback Effects
// ============================================================================

// Spark effects (malfunction)
func drawFactoryHUD(screen *ebiten.Image, fs *factory.FactoryState, sw, sh int, tick int) {
	panelW := 380
	panelH := 300
	vector.DrawFilledRect(screen, 8, 8, float32(panelW), float32(panelH), colorHUDPanel, false)
	vector.StrokeRect(screen, 8, 8, float32(panelW), float32(panelH), 1, color.RGBA{60, 70, 100, 180}, false)

	x := 16
	y := 16

	printColoredAt(screen, locale.T("factory.title"), x, y, color.RGBA{255, 220, 100, 255})

	// Clock/time display
	simMinutes := fs.Tick / 60
	simHours := simMinutes / 60
	clockStr := fmt.Sprintf("%02d:%02d", simHours%24, simMinutes%60)
	printColoredAt(screen, clockStr, x+panelW-60, y, color.RGBA{180, 200, 255, 200})

	// Day/Night indicator
	cyclePos := math.Mod(float64(fs.Tick), 10000.0) / 10000.0
	var timeOfDay string
	switch {
	case cyclePos < 0.15:
		timeOfDay = locale.T("factory.hud.dawn")
	case cyclePos < 0.45:
		timeOfDay = locale.T("factory.hud.day")
	case cyclePos < 0.55:
		timeOfDay = locale.T("factory.hud.dusk")
	default:
		timeOfDay = locale.T("factory.hud.night")
	}
	printColoredAt(screen, timeOfDay, x+panelW-120, y, color.RGBA{140, 150, 180, 150})
	y += lineH + 2

	printColoredAt(screen, locale.Tf("factory.hud.bots_tick", fs.BotCount, fs.Tick), x, y, colorHUDText)
	y += lineH

	status := locale.T("factory.hud.running")
	statusCol := color.RGBA{80, 220, 80, 255}
	if fs.Paused {
		status = locale.T("factory.hud.paused")
		statusCol = color.RGBA{220, 80, 80, 255}
	}
	printColoredAt(screen, status, x, y, statusCol)
	printColoredAt(screen, locale.Tf("factory.hud.speed", fs.Speed), x+80, y, colorHUDDim)

	// Weather indicator
	weatherStr := locale.T("factory.hud.weather_clear")
	if fs.Weather == factory.WeatherRain {
		weatherStr = locale.T("factory.hud.weather_rain")
	}
	printColoredAt(screen, weatherStr, x+180, y, color.RGBA{140, 160, 200, 180})
	y += lineH

	printColoredAt(screen, locale.Tf("factory.hud.zoom", fs.CamZoom), x, y, colorHUDDim)

	// Heatmap indicator
	if fs.ShowHeatmap {
		printColoredAt(screen, locale.T("factory.hud.heatmap_on"), x+100, y, color.RGBA{220, 80, 80, 200})
	}
	y += lineH + 4

	// Live stats
	printColoredAt(screen, locale.Tf("factory.hud.working",
		fs.Stats.BotsWorking, fs.Stats.BotsIdle, fs.Stats.BotsCharging), x, y, colorHUDText)
	y += lineH
	printColoredAt(screen, locale.Tf("factory.hud.tasks",
		len(fs.Tasks.Tasks), fs.Stats.PartsProcessed, fs.Stats.TrucksUnloaded, fs.Stats.TrucksLoaded), x, y, colorHUDText)
	y += lineH + 4

	// Mini production pipeline
	drawProductionPipeline(screen, x, y, panelW-20, fs, tick)
	y += 28

	// Score display
	printColoredAt(screen, locale.Tf("factory.hud.score", fs.Score, fs.CompletedOrders), x, y, color.RGBA{255, 220, 100, 200})
	y += lineH

	// Feature 9: Budget display
	budgetCol := color.RGBA{80, 220, 80, 255}
	if fs.Budget < 1000 {
		// Flash red when low
		pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.15)
		budgetCol = color.RGBA{uint8(180 + pulse*75), 40, 40, 255}
	} else if fs.Budget < 3000 {
		budgetCol = color.RGBA{220, 180, 40, 255}
	}
	printColoredAt(screen, locale.Tf("factory.hud.budget", fs.Budget, fs.TotalEnergyCost), x, y, budgetCol)

	// Energy price indicator (sun/moon with price)
	dayPhase := math.Mod(float64(fs.Tick), 10000.0) / 10000.0
	isNight := dayPhase > 0.7 || dayPhase < 0.2
	priceStr := fmt.Sprintf("$%.2f/u", fs.EnergyCostDay)
	priceCol := color.RGBA{220, 200, 60, 200}
	if isNight {
		priceStr = fmt.Sprintf("$%.2f/u", fs.EnergyCostNight)
		priceCol = color.RGBA{100, 140, 220, 200}
	}
	printColoredAt(screen, priceStr, x+panelW-70, y, priceCol)
	y += lineH

	// Feature 6: QC rejects
	if fs.Stats.IncomingRejects > 0 {
		printColoredAt(screen, locale.Tf("factory.hud.qc_rejects", fs.Stats.IncomingRejects), x, y, color.RGBA{220, 100, 100, 200})
		y += lineH
	}

	// Feature 7: Stock warnings
	if fs.StockWarning {
		pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.12)
		alpha := uint8(150 + pulse*105)
		printColoredAt(screen, locale.T("factory.hud.low_stock"), x+panelW-90, y-lineH, color.RGBA{255, 160, 40, alpha})
	}
	if fs.OutboundFull {
		pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.12)
		alpha := uint8(150 + pulse*105)
		printColoredAt(screen, locale.T("factory.hud.outbound_full"), x+panelW-110, y-lineH-lineH, color.RGBA{255, 80, 40, alpha})
	}

	y += 4

	// Controls
	printColoredAt(screen, locale.T("factory.hud.controls1"), x, y, color.RGBA{100, 110, 130, 180})
	y += lineH
	printColoredAt(screen, locale.T("factory.hud.controls2"), x, y, color.RGBA{100, 110, 130, 180})
	y += lineH
	printColoredAt(screen, locale.T("factory.hud.controls3"), x, y, color.RGBA{100, 110, 130, 180})
}

// drawProductionPipeline draws a mini flow diagram.
func drawProductionPipeline(screen *ebiten.Image, x, y, width int, fs *factory.FactoryState, tick int) {
	stages := []struct {
		label string
		col   color.RGBA
	}{
		{locale.T("factory.pipe.lkw"), color.RGBA{120, 120, 130, 255}},
		{locale.T("factory.pipe.recv"), color.RGBA{80, 200, 80, 255}},
		{locale.T("factory.pipe.stor"), color.RGBA{80, 140, 220, 255}},
		{locale.T("factory.pipe.prod"), color.RGBA{220, 160, 60, 255}},
		{locale.T("factory.pipe.ship"), color.RGBA{200, 80, 80, 255}},
		{locale.T("factory.pipe.lkw"), color.RGBA{120, 120, 130, 255}},
	}

	stageW := width / len(stages)
	stageH := 16

	for i, st := range stages {
		sx := x + i*stageW
		bright := uint8(0)
		if i >= 1 && i <= 4 {
			pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.05+float64(i)*0.8)
			bright = uint8(pulse * 40)
		}
		bgCol := color.RGBA{st.col.R/4 + bright, st.col.G/4 + bright, st.col.B/4 + bright, 200}
		vector.DrawFilledRect(screen, float32(sx), float32(y), float32(stageW-2), float32(stageH), bgCol, false)
		vector.StrokeRect(screen, float32(sx), float32(y), float32(stageW-2), float32(stageH), 1, st.col, false)
		lx := sx + stageW/2 - runeLen(st.label)*charW/2
		printColoredAt(screen, st.label, lx, y+2, st.col)
		if i < len(stages)-1 {
			arrowX := float32(sx + stageW - 1)
			arrowY := float32(y + stageH/2)
			vector.StrokeLine(screen, arrowX, arrowY, arrowX+4, arrowY, 1, color.RGBA{160, 170, 190, 150}, false)
		}
	}
}

// drawFactoryLegend renders a bot state legend on the right side of the screen.
func drawFactoryLegend(screen *ebiten.Image, fs *factory.FactoryState, sw, sh int) {
	var counts [7]int
	for i := range fs.Bots {
		s := fs.Bots[i].State
		if s >= 0 && s < 7 {
			counts[s]++
		}
	}

	type legendEntry struct {
		label string
		col   color.RGBA
		count int
	}
	entries := []legendEntry{
		{locale.T("factory.legend.idle"), color.RGBA{100, 100, 110, 255}, counts[0]},
		{locale.T("factory.legend.navigating"), color.RGBA{0, 200, 220, 255}, counts[1]},
		{locale.T("factory.legend.picking_up"), color.RGBA{0, 220, 100, 255}, counts[2]},
		{locale.T("factory.legend.carrying"), color.RGBA{220, 200, 40, 255}, counts[3]},
		{locale.T("factory.legend.delivering"), color.RGBA{0, 255, 120, 255}, counts[4]},
		{locale.T("factory.legend.charging"), color.RGBA{255, 220, 40, 255}, counts[5]},
		{locale.T("factory.legend.repairing"), color.RGBA{220, 120, 40, 255}, counts[6]},
	}

	panelW := 150
	panelH := len(entries)*lineH + 24
	px := sw - panelW - 12
	py := 8

	vector.DrawFilledRect(screen, float32(px), float32(py), float32(panelW), float32(panelH), colorHUDPanel, false)
	vector.StrokeRect(screen, float32(px), float32(py), float32(panelW), float32(panelH), 1, color.RGBA{60, 70, 100, 180}, false)

	printColoredAt(screen, locale.T("factory.legend.title"), px+6, py+4, color.RGBA{200, 210, 230, 200})
	ey := py + 20

	for _, e := range entries {
		vector.DrawFilledCircle(screen, float32(px+14), float32(ey+lineH/2-1), 4, e.col, false)
		text := fmt.Sprintf("%-11s %d", e.label, e.count)
		printColoredAt(screen, text, px+24, ey, colorHUDDim)
		ey += lineH
	}
}

// --- Animated dashed line helper ---
func drawBotTooltip(screen *ebiten.Image, fs *factory.FactoryState, botIdx, sw, sh int) {
	if botIdx < 0 || botIdx >= len(fs.Bots) {
		return
	}
	bot := &fs.Bots[botIdx]

	ttStateNames := []string{
		locale.T("factory.tooltip.state.idle"), locale.T("factory.tooltip.state.nav"),
		locale.T("factory.tooltip.state.pickup"), locale.T("factory.tooltip.state.carry"),
		locale.T("factory.tooltip.state.deliver"), locale.T("factory.tooltip.state.charge"),
		locale.T("factory.tooltip.state.repair"),
	}
	stateName := "?"
	if bot.State >= 0 && bot.State < len(ttStateNames) {
		stateName = ttStateNames[bot.State]
	}
	energy := int(bot.Energy)
	if energy > 100 {
		energy = 100
	}
	if energy < 0 {
		energy = 0
	}
	roleStr := ""
	if botIdx < len(fs.BotRoles) {
		ttRoleNames := []string{locale.T("factory.tooltip.role.t"), locale.T("factory.tooltip.role.fl"), locale.T("factory.tooltip.role.ex")}
		if int(fs.BotRoles[botIdx]) < len(ttRoleNames) {
			roleStr = ttRoleNames[fs.BotRoles[botIdx]]
		}
	}
	text := locale.Tf("factory.tooltip.bot", botIdx, stateName, roleStr, energy)

	mx, my := ebiten.CursorPosition()
	ttX := mx + 16
	ttY := my - 20
	ttW := runeLen(text)*charW + 12
	ttH := lineH + 8
	if ttX+ttW > sw {
		ttX = mx - ttW - 4
	}
	if ttY < 0 {
		ttY = my + 16
	}

	vector.DrawFilledRect(screen, float32(ttX), float32(ttY), float32(ttW), float32(ttH),
		color.RGBA{20, 24, 35, 230}, false)
	vector.StrokeRect(screen, float32(ttX), float32(ttY), float32(ttW), float32(ttH),
		1, color.RGBA{80, 100, 140, 200}, false)
	printColoredAt(screen, text, ttX+6, ttY+4, color.RGBA{220, 230, 255, 240})
}

// --- Selected Bot Info Panel ---
func drawSelectedBotPanel(screen *ebiten.Image, fs *factory.FactoryState, sw, sh, tick int) {
	bot := &fs.Bots[fs.SelectedBot]

	panelW := 200
	panelH := 250
	legendH := 7*lineH + 24
	px := sw - panelW - 12
	py := 8 + legendH + 8

	vector.DrawFilledRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		colorHUDPanel, false)
	vector.StrokeRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		1, color.RGBA{80, 140, 220, 180}, false)

	x := px + 8
	y := py + 6

	title := locale.Tf("factory.bot.title", fs.SelectedBot)
	printColoredAt(screen, title, x, y, color.RGBA{100, 200, 255, 255})
	y += lineH

	stateNames := []string{
		locale.T("factory.bot.state.idle"), locale.T("factory.bot.state.navigating"),
		locale.T("factory.bot.state.picking_up"), locale.T("factory.bot.state.carrying"),
		locale.T("factory.bot.state.delivering"), locale.T("factory.bot.state.charging"),
		locale.T("factory.bot.state.repairing"),
	}
	stateColors := []color.RGBA{
		{100, 100, 110, 255}, {0, 200, 220, 255}, {0, 220, 100, 255},
		{220, 200, 40, 255}, {0, 255, 120, 255}, {255, 220, 40, 255}, {220, 120, 40, 255},
	}
	stateName := locale.T("factory.bot.state.unknown")
	stateCol := color.RGBA{150, 150, 160, 255}
	if bot.State >= 0 && bot.State < len(stateNames) {
		stateName = stateNames[bot.State]
		stateCol = stateColors[bot.State]
	}
	vector.DrawFilledCircle(screen, float32(x+4), float32(y+lineH/2), 4, stateCol, false)
	printColoredAt(screen, stateName, x+14, y, stateCol)
	y += lineH

	energy := bot.Energy
	if energy > 100 {
		energy = 100
	}
	if energy < 0 {
		energy = 0
	}
	barW := panelW - 50
	barH := 10
	printColoredAt(screen, locale.T("factory.bot.energy_label"), x, y+1, colorHUDDim)
	barX := x + 18
	vector.DrawFilledRect(screen, float32(barX), float32(y+2), float32(barW), float32(barH),
		color.RGBA{30, 35, 50, 200}, false)
	eCol := color.RGBA{80, 220, 80, 220}
	if energy < 30 {
		eCol = color.RGBA{220, 60, 60, 220}
	} else if energy < 60 {
		eCol = color.RGBA{220, 200, 60, 220}
	}
	fillW := float32(float64(barW) * energy / 100.0)
	vector.DrawFilledRect(screen, float32(barX), float32(y+2), fillW, float32(barH), eCol, false)
	vector.StrokeRect(screen, float32(barX), float32(y+2), float32(barW), float32(barH),
		1, color.RGBA{80, 90, 120, 200}, false)
	pctStr := fmt.Sprintf("%d%%", int(energy))
	printColoredAt(screen, pctStr, barX+barW+4, y+1, colorHUDDim)
	y += lineH + 2

	taskIdx := factory.FindBotTask(fs, fs.SelectedBot)
	if taskIdx >= 0 {
		task := &fs.Tasks.Tasks[taskIdx]
		taskNames := []string{
			locale.T("factory.task.none"), locale.T("factory.task.unload"),
			locale.T("factory.task.to_machine"), locale.T("factory.task.from_machine"),
			locale.T("factory.task.load"), locale.T("factory.task.charge"),
			locale.T("factory.task.repair"),
		}
		tName := locale.T("factory.task.none")
		if int(task.Type) < len(taskNames) {
			tName = taskNames[task.Type]
		}
		printColoredAt(screen, locale.Tf("factory.bot.task", tName), x, y, colorHUDText)
	} else {
		printColoredAt(screen, locale.T("factory.bot.task_none"), x, y, colorHUDDim)
	}
	y += lineH

	if bot.CarryingPkg > 0 {
		pkgNames := []string{"", locale.T("factory.part.red"), locale.T("factory.part.blue"), locale.T("factory.part.yellow"), locale.T("factory.part.green")}
		pkgName := "?"
		if bot.CarryingPkg < len(pkgNames) {
			pkgName = pkgNames[bot.CarryingPkg]
		}
		printColoredAt(screen, locale.Tf("factory.bot.carrying", pkgName), x, y, color.RGBA{220, 200, 80, 240})
	} else {
		printColoredAt(screen, locale.T("factory.bot.carrying_none"), x, y, colorHUDDim)
	}
	y += lineH

	printColoredAt(screen, locale.Tf("factory.bot.pos", bot.X, bot.Y), x, y, colorHUDDim)
	y += lineH

	// Role info
	if fs.SelectedBot < len(fs.BotRoles) {
		roleNames := []string{locale.T("factory.role.transporter"), locale.T("factory.role.forklift"), locale.T("factory.role.express")}
		rn := locale.T("factory.bot.state.unknown")
		if int(fs.BotRoles[fs.SelectedBot]) < len(roleNames) {
			rn = roleNames[fs.BotRoles[fs.SelectedBot]]
		}
		printColoredAt(screen, locale.Tf("factory.bot.role", rn), x, y, colorHUDDim)
		y += lineH
	}

	// Experience info
	if fs.SelectedBot < len(fs.BotDeliveries) {
		deliveries := fs.BotDeliveries[fs.SelectedBot]
		rank := locale.T("factory.exp.novice")
		if deliveries >= 100 {
			rank = locale.T("factory.exp.master")
		} else if deliveries >= 50 {
			rank = locale.T("factory.exp.expert")
		} else if deliveries >= 10 {
			rank = locale.T("factory.exp.experienced")
		}
		printColoredAt(screen, locale.Tf("factory.bot.exp", rank, deliveries), x, y, colorHUDDim)
		y += lineH
	}

	// Maintenance hours
	if fs.SelectedBot < len(fs.BotOpHours) {
		opH := fs.BotOpHours[fs.SelectedBot]
		mCol := colorHUDDim
		if opH >= factory.MaintenanceMandatoryHours {
			mCol = color.RGBA{255, 60, 60, 220}
		} else if opH >= factory.MaintenanceWarningHours {
			mCol = color.RGBA{220, 200, 40, 220}
		}
		printColoredAt(screen, locale.Tf("factory.bot.ophrs", opH), x, y, mCol)
		y += lineH
	}

	if fs.FollowCamBot == fs.SelectedBot {
		pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.1)
		a := uint8(150 + int(pulse*105))
		printColoredAt(screen, locale.T("factory.bot.following"), x, y, color.RGBA{100, 200, 255, a})
	} else {
		printColoredAt(screen, locale.T("factory.bot.follow"), x, y, color.RGBA{80, 90, 110, 150})
	}
}

// SpawnPulseEffect creates a brief expanding ring at (x, y) with given color.
func drawOrderPanel(screen *ebiten.Image, fs *factory.FactoryState, sw, sh, tick int) {
	if len(fs.Orders) == 0 {
		return
	}

	panelW := 220
	maxOrders := 8
	visibleOrders := 0
	for _, o := range fs.Orders {
		if !o.Completed {
			visibleOrders++
		}
	}
	if visibleOrders == 0 {
		// Show last completed orders if all done
		for i := len(fs.Orders) - 1; i >= 0 && visibleOrders < 3; i-- {
			if fs.Orders[i].Completed {
				visibleOrders++
			}
		}
	}
	if visibleOrders > maxOrders {
		visibleOrders = maxOrders
	}
	panelH := visibleOrders*lineH + 28

	px := 8
	py := 260 // below HUD

	vector.DrawFilledRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		colorHUDPanel, false)
	vector.StrokeRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		1, color.RGBA{60, 70, 100, 180}, false)

	printColoredAt(screen, locale.T("factory.order.title"), px+6, py+4, color.RGBA{200, 210, 230, 200})
	ey := py + 20

	shown := 0
	for i := range fs.Orders {
		if shown >= maxOrders {
			break
		}
		o := &fs.Orders[i]

		colorNames := []string{"", locale.T("factory.order.red"), locale.T("factory.order.blue"), locale.T("factory.order.yellow"), locale.T("factory.order.green")}
		cname := "?"
		if o.OutputColor > 0 && o.OutputColor < len(colorNames) {
			cname = colorNames[o.OutputColor]
		}

		text := fmt.Sprintf("#%d: %dx%s [%d/%d]", o.ID, o.Quantity, cname, o.Fulfilled, o.Quantity)
		textCol := colorHUDText

		if o.Completed {
			textCol = color.RGBA{80, 200, 80, 180}
			text += " " + locale.T("factory.order.done")
		} else if fs.Tick > o.Deadline {
			// Overdue: flash red
			if tick%20 < 10 {
				textCol = color.RGBA{255, 60, 60, 255}
			} else {
				textCol = color.RGBA{200, 40, 40, 200}
			}
			text += " " + locale.T("factory.order.late")
		}

		// Color dot
		partCol := partColorToRGBA(o.OutputColor)
		vector.DrawFilledCircle(screen, float32(px+10), float32(ey+lineH/2-1), 3, partCol, false)
		printColoredAt(screen, text, px+18, ey, textCol)
		ey += lineH
		shown++
	}
}

// ============================================================================
// [NEW] Alert Ticker (top-right, below legend)
// ============================================================================
func drawAlertTicker(screen *ebiten.Image, fs *factory.FactoryState, sw, sh, tick int) {
	if len(fs.Alerts) == 0 {
		return
	}

	// Show up to MaxVisibleAlerts recent alerts
	alertW := 300
	alertH := lineH + 6
	startX := sw - alertW - 14
	// Position below legend and selected bot panel
	startY := 8 + (7*lineH + 24) + 8

	shown := 0
	for i := len(fs.Alerts) - 1; i >= 0 && shown < factory.MaxVisibleAlerts; i-- {
		alert := &fs.Alerts[i]
		age := fs.Tick - alert.Tick
		if age > factory.AlertFadeTicks {
			continue
		}

		// Fade alpha
		fadeProgress := float64(age) / float64(factory.AlertFadeTicks)
		alpha := uint8(200 * (1.0 - fadeProgress))
		bgAlpha := uint8(180 * (1.0 - fadeProgress))

		ay := startY + shown*(alertH+2)

		// Background with colored left edge
		vector.DrawFilledRect(screen, float32(startX), float32(ay), float32(alertW), float32(alertH),
			color.RGBA{15, 18, 28, bgAlpha}, false)
		// Color strip on left
		vector.DrawFilledRect(screen, float32(startX), float32(ay), 4, float32(alertH),
			color.RGBA{alert.Color[0], alert.Color[1], alert.Color[2], alpha}, false)
		// Border
		vector.StrokeRect(screen, float32(startX), float32(ay), float32(alertW), float32(alertH),
			1, color.RGBA{alert.Color[0], alert.Color[1], alert.Color[2], bgAlpha / 2}, false)

		// Text
		printColoredAt(screen, alert.Message, startX+8, ay+3,
			color.RGBA{alert.Color[0], alert.Color[1], alert.Color[2], alpha})

		shown++
	}
}

// ============================================================================
// [NEW-1] Energy bar on ALL bots (1px tall, max 8px wide, color-coded)
// ============================================================================
func drawBotEnergyMicroBarAlways(screen *ebiten.Image, bot *swarm.SwarmBot,
	bx, by float32, ws func(float64) float32) {
	barW := float32(8) * ws(1)
	if barW < 2 {
		barW = 2
	}
	barH := float32(1.5)
	pct := float32(bot.Energy / 100.0)
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	botR := ws(factory.FactoryBotRadius)
	if botR < 1 {
		botR = 1
	}
	barX := bx - barW/2
	barY := by + botR + 2
	// Background
	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{30, 30, 30, 180}, false)
	// Fill
	fillCol := color.RGBA{60, 200, 60, 220} // green
	if bot.Energy < 20 {
		fillCol = color.RGBA{220, 40, 40, 220} // red
	} else if bot.Energy < 50 {
		fillCol = color.RGBA{220, 200, 40, 220} // yellow
	}
	vector.DrawFilledRect(screen, barX, barY, barW*pct, barH, fillCol, false)
}

// ============================================================================
// [NEW-2] Charging Animation — Lightning Bolts
// ============================================================================
func drawEfficiencySparkline(screen *ebiten.Image, fs *factory.FactoryState, x, y int) {
	sparkW := 50
	sparkH := 20

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(sparkW), float32(sparkH),
		color.RGBA{8, 10, 18, 200}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(sparkW), float32(sparkH),
		1, color.RGBA{40, 50, 70, 150}, false)

	n := len(fs.EfficiencyHistory)
	count := fs.EfficiencyIdx
	if count > n {
		count = n
	}
	if count < 2 {
		return
	}

	stepW := float32(sparkW) / float32(n)

	for i := 0; i < count-1; i++ {
		idx1 := (fs.EfficiencyIdx - count + i) % n
		idx2 := (fs.EfficiencyIdx - count + i + 1) % n
		if idx1 < 0 {
			idx1 += n
		}
		if idx2 < 0 {
			idx2 += n
		}

		v1 := fs.EfficiencyHistory[idx1]
		v2 := fs.EfficiencyHistory[idx2]

		sy1 := float32(y+sparkH) - float32(v1)*float32(sparkH)/100.0
		sy2 := float32(y+sparkH) - float32(v2)*float32(sparkH)/100.0
		sx1 := float32(x) + float32(i)*stepW
		sx2 := float32(x) + float32(i+1)*stepW

		lineCol := color.RGBA{80, 220, 80, 200}
		if v2 < 50 {
			lineCol = color.RGBA{220, 60, 60, 200}
		} else if v2 < 75 {
			lineCol = color.RGBA{220, 200, 60, 200}
		}
		vector.StrokeLine(screen, sx1, sy1, sx2, sy2, 1.5, lineCol, false)
	}
}

// ============================================================================
// [NEW-8] Ambient Factory Noise Visualization
// ============================================================================
func drawAchievementPopup(screen *ebiten.Image, fs *factory.FactoryState, sw, sh int) {
	if fs.AchievementTimer <= 0 || fs.AchievementPopup == "" {
		return
	}

	text := fs.AchievementPopup
	textW := runeLen(text)*charW + 40
	textH := lineH*2 + 16
	ppx := sw/2 - textW/2
	ppy := sh/3 - textH/2

	// Fade based on timer
	fadeProgress := float64(fs.AchievementTimer) / 100.0
	if fadeProgress > 1 {
		fadeProgress = 1
	}
	alpha := uint8(220 * fadeProgress)
	bgAlpha := uint8(200 * fadeProgress)

	// Golden background
	vector.DrawFilledRect(screen, float32(ppx), float32(ppy), float32(textW), float32(textH),
		color.RGBA{30, 25, 10, bgAlpha}, false)
	// Golden border
	vector.StrokeRect(screen, float32(ppx), float32(ppy), float32(textW), float32(textH),
		2, color.RGBA{255, 215, 0, alpha}, false)
	// Inner glow border
	vector.StrokeRect(screen, float32(ppx+2), float32(ppy+2), float32(textW-4), float32(textH-4),
		1, color.RGBA{255, 200, 50, alpha / 2}, false)

	// Trophy label
	trophyLabel := locale.T("factory.achieve.title")
	printColoredAt(screen, trophyLabel, ppx+textW/2-runeLen(trophyLabel)*charW/2, ppy+4,
		color.RGBA{255, 215, 0, alpha})
	// Achievement text
	printColoredAt(screen, text, ppx+textW/2-runeLen(text)*charW/2, ppy+4+lineH+4,
		color.RGBA{255, 240, 180, alpha})
}

// ============================================================================
// Feature 6: QC Reject Effects — brief red X flash at QC area
// ============================================================================
func drawMaintenancePlanner(screen *ebiten.Image, fs *factory.FactoryState, sw, sh int, tick int) {
	// Semi-transparent dark panel covering center of screen
	panelW := 600
	panelH := 500
	if panelH > sh-40 {
		panelH = sh - 40
	}
	px := (sw - panelW) / 2
	py := (sh - panelH) / 2

	// Background
	vector.DrawFilledRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		color.RGBA{15, 18, 28, 230}, false)
	vector.StrokeRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		2, color.RGBA{80, 100, 140, 200}, false)

	// Title
	maintTitle := locale.T("factory.maint.title")
	printColoredAt(screen, maintTitle, px+panelW/2-runeLen(maintTitle)*charW/2, py+8, color.RGBA{255, 220, 100, 255})

	// Close hint
	printColoredAt(screen, locale.T("factory.maint.close"), px+panelW-80, py+8, color.RGBA{120, 130, 150, 180})

	// Table header
	headerY := py + 30
	printColoredAt(screen, locale.T("factory.maint.id"), px+10, headerY, color.RGBA{180, 190, 210, 200})
	printColoredAt(screen, locale.T("factory.maint.role"), px+60, headerY, color.RGBA{180, 190, 210, 200})
	printColoredAt(screen, locale.T("factory.maint.hours"), px+160, headerY, color.RGBA{180, 190, 210, 200})
	printColoredAt(screen, locale.T("factory.maint.status"), px+240, headerY, color.RGBA{180, 190, 210, 200})
	printColoredAt(screen, locale.T("factory.maint.next"), px+380, headerY, color.RGBA{180, 190, 210, 200})

	// Separator line
	vector.StrokeLine(screen, float32(px+8), float32(headerY+lineH+2), float32(px+panelW-8), float32(headerY+lineH+2),
		1, color.RGBA{60, 70, 100, 150}, false)

	// Collect bot data and sort by operating hours (descending)
	type botEntry struct {
		idx    int
		role   string
		hours  int
		status string
	}

	entries := make([]botEntry, 0, len(fs.Bots))
	for i := range fs.Bots {
		hours := 0
		if i < len(fs.BotOpHours) {
			hours = fs.BotOpHours[i]
		}
		role := locale.T("factory.maint.transport")
		if i < len(fs.BotRoles) {
			switch fs.BotRoles[i] {
			case factory.RoleForklift:
				role = locale.T("factory.role.forklift")
			case factory.RoleExpress:
				role = locale.T("factory.role.express")
			}
		}
		status := locale.T("factory.maint.idle")
		if i < len(fs.Bots) {
			switch fs.Bots[i].State {
			case factory.BotMovingToSource, factory.BotPickingUp, factory.BotMovingToDest, factory.BotDelivering:
				status = locale.T("factory.maint.working")
			case factory.BotCharging:
				status = locale.T("factory.maint.charging")
			case factory.BotRepairing:
				status = locale.T("factory.maint.repairing")
			case factory.BotOffShift:
				status = locale.T("factory.maint.offshift")
			case factory.BotEmergencyEvac:
				status = locale.T("factory.maint.evacuating")
			}
		}
		entries = append(entries, botEntry{idx: i, role: role, hours: hours, status: status})
	}

	// Sort by hours descending (simple insertion sort for top 20)
	for i := 1; i < len(entries); i++ {
		j := i
		for j > 0 && entries[j].hours > entries[j-1].hours {
			entries[j], entries[j-1] = entries[j-1], entries[j]
			j--
		}
	}

	// Display top 20
	rowY := headerY + lineH + 8
	maxRows := 20
	if maxRows > len(entries) {
		maxRows = len(entries)
	}
	availableRows := (panelH - (rowY - py) - 20) / lineH
	if availableRows < maxRows {
		maxRows = availableRows
	}

	for i := 0; i < maxRows; i++ {
		e := entries[i]

		// Color-coded by operating hours
		rowCol := color.RGBA{80, 220, 80, 220} // Green: <15000h
		if e.hours >= 25000 {
			rowCol = color.RGBA{255, 60, 60, 220} // Red: >25000h
		} else if e.hours >= 15000 {
			rowCol = color.RGBA{220, 200, 40, 220} // Yellow: 15000-25000h
		}

		printColoredAt(screen, fmt.Sprintf("#%d", e.idx), px+10, rowY, rowCol)
		printColoredAt(screen, e.role, px+60, rowY, color.RGBA{170, 180, 200, 200})
		printColoredAt(screen, fmt.Sprintf("%d", e.hours), px+160, rowY, rowCol)
		printColoredAt(screen, e.status, px+240, rowY, color.RGBA{170, 180, 200, 200})

		// Estimated time until next mandatory maintenance
		remaining := factory.MaintenanceMandatoryHours - e.hours
		if remaining < 0 {
			remaining = 0
		}
		maintStr := locale.Tf("factory.maint.ticks", remaining)
		if remaining <= 0 {
			maintStr = locale.T("factory.maint.overdue")
			printColoredAt(screen, maintStr, px+380, rowY, color.RGBA{255, 60, 60, 255})
		} else if remaining < 5000 {
			printColoredAt(screen, maintStr, px+380, rowY, color.RGBA{220, 180, 40, 220})
		} else {
			printColoredAt(screen, maintStr, px+380, rowY, color.RGBA{120, 140, 160, 180})
		}

		rowY += lineH
	}

	// Scroll down indicator: show if more entries exist below visible area
	if maxRows < len(entries) {
		arrowX := px + panelW/2 - 15
		arrowY := rowY + 2
		printColoredAt(screen, "v v v", arrowX, arrowY, color.RGBA{150, 160, 180, 150})
	}

	// Summary at bottom
	summaryY := py + panelH - lineH - 8
	totalBots := len(fs.Bots)
	greenCount, yellowCount, redCount := 0, 0, 0
	for i := range fs.BotOpHours {
		h := fs.BotOpHours[i]
		if h >= 25000 {
			redCount++
		} else if h >= 15000 {
			yellowCount++
		} else {
			greenCount++
		}
	}
	printColoredAt(screen, locale.Tf("factory.maint.summary",
		totalBots, greenCount, yellowCount, redCount), px+10, summaryY, color.RGBA{160, 170, 190, 200})
}

// --- Feature 11: Active Event Banner ---
func drawEventBanner(screen *ebiten.Image, fs *factory.FactoryState, sw int, tick int) {
	if !fs.ActiveEvent.Active {
		return
	}

	bannerNames := []string{
		locale.T("factory.event_banner.supply_shortage"), locale.T("factory.event_banner.rush_order"),
		locale.T("factory.event_banner.power_outage"), locale.T("factory.event_banner.machine_breakdown"),
		locale.T("factory.event_banner.efficiency_bonus"), locale.T("factory.event_banner.inspection_visit"),
	}
	colors := [][3]uint8{{220, 60, 60}, {255, 200, 40}, {220, 100, 30}, {200, 40, 40}, {40, 200, 40}, {200, 200, 40}}

	evtType := fs.ActiveEvent.Type
	if evtType < 0 || evtType >= len(bannerNames) {
		return
	}

	label := locale.Tf("factory.event.banner", bannerNames[evtType], fs.ActiveEvent.Timer)
	bannerW := runeLen(label)*charW + 40
	bannerH := 22
	bx := sw/2 - bannerW/2
	by := 2

	// Pulsing alpha
	pulse := 0.7 + 0.3*math.Sin(float64(tick)*0.1)
	bgAlpha := uint8(float64(180) * pulse)

	c := colors[evtType]
	vector.DrawFilledRect(screen, float32(bx), float32(by), float32(bannerW), float32(bannerH),
		color.RGBA{c[0] / 4, c[1] / 4, c[2] / 4, bgAlpha}, false)
	vector.StrokeRect(screen, float32(bx), float32(by), float32(bannerW), float32(bannerH),
		1, color.RGBA{c[0], c[1], c[2], 200}, false)
	printColoredAt(screen, label, bx+20, by+4, color.RGBA{c[0], c[1], c[2], 255})
}

// --- Feature 12: Bot Productivity Podium ---
func drawBotPodium(screen *ebiten.Image, fs *factory.FactoryState, sw, sh int) {
	stats := &fs.Stats
	// Only draw if we have data
	if stats.TopWorkers[0][1] == 0 {
		return
	}

	panelW := 200
	panelH := 70
	px := sw - panelW - 12
	// Position below the legend panel (legend is about 200px from top)
	py := 200

	vector.DrawFilledRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		colorHUDPanel, false)
	vector.StrokeRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		1, color.RGBA{60, 70, 100, 180}, false)

	printColoredAt(screen, locale.T("factory.podium.title"), px+6, py+4, color.RGBA{255, 215, 0, 220})

	medalColors := []color.RGBA{
		{255, 215, 0, 255},   // gold
		{192, 192, 192, 255}, // silver
		{205, 127, 50, 255},  // bronze
	}
	rankLabels := []string{"#1", "#2", "#3"}

	y := py + 18
	for r := 0; r < 3; r++ {
		botIdx := stats.TopWorkers[r][0]
		deliveries := stats.TopWorkers[r][1]
		if deliveries == 0 {
			continue
		}
		// Medal circle
		vector.DrawFilledCircle(screen, float32(px+14), float32(y+lineH/2-1), 5, medalColors[r], false)
		text := locale.Tf("factory.podium.entry", rankLabels[r], botIdx, deliveries)
		printColoredAt(screen, text, px+24, y, colorHUDDim)
		y += lineH
	}
}

// ============================================================================
// Factory Help Overlay
// ============================================================================
func drawFactoryHelp(screen *ebiten.Image, sw, sh int) {
	// Semi-transparent dark overlay
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh),
		color.RGBA{10, 12, 20, 220}, false)

	panelW := 560
	panelH := sh - 40
	if panelH > 720 {
		panelH = 720
	}
	px := (sw - panelW) / 2
	py := (sh - panelH) / 2

	// Panel background
	vector.DrawFilledRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		color.RGBA{15, 18, 28, 240}, false)
	vector.StrokeRect(screen, float32(px), float32(py), float32(panelW), float32(panelH),
		2, color.RGBA{100, 140, 220, 200}, false)

	x := px + 16
	y := py + 12

	titleCol := color.RGBA{255, 220, 100, 255}
	headCol := color.RGBA{100, 200, 255, 240}
	textCol := color.RGBA{190, 200, 220, 220}
	closeCol := color.RGBA{120, 130, 160, 200}

	lines := []struct {
		key string
		col color.RGBA
	}{
		{"factory.help.title", titleCol},
		{"", textCol},
		{"factory.help.controls", headCol},
		{"factory.help.ctrl.wasd", textCol},
		{"factory.help.ctrl.wheel", textCol},
		{"factory.help.ctrl.rdrag", textCol},
		{"factory.help.ctrl.click", textCol},
		{"factory.help.ctrl.space", textCol},
		{"factory.help.ctrl.speed", textCol},
		{"factory.help.ctrl.follow", textCol},
		{"factory.help.ctrl.help", textCol},
		{"", textCol},
		{"factory.help.bots", headCol},
		{"factory.help.bot.buy", textCol},
		{"factory.help.bot.sell", textCol},
		{"factory.help.bot.add", textCol},
		{"", textCol},
		{"factory.help.factory", headCol},
		{"factory.help.fac.truck_in", textCol},
		{"factory.help.fac.truck_out", textCol},
		{"factory.help.fac.emergency", textCol},
		{"factory.help.fac.maint", textCol},
		{"factory.help.fac.export", textCol},
		{"", textCol},
		{"factory.help.displays", headCol},
		{"factory.help.disp.heatmap", textCol},
		{"factory.help.disp.minimap", textCol},
		{"", textCol},
		{"factory.help.flow", headCol},
		{"factory.help.flow.line1", textCol},
		{"factory.help.flow.line2", textCol},
		{"factory.help.flow.line3", textCol},
		{"", textCol},
		{"factory.help.bottypes", headCol},
		{"factory.help.type.transport", textCol},
		{"factory.help.type.forklift", textCol},
		{"factory.help.type.express", textCol},
		{"", textCol},
		{"factory.help.economy", headCol},
		{"factory.help.econ.raw", textCol},
		{"factory.help.econ.finished", textCol},
		{"factory.help.econ.energy_day", textCol},
		{"factory.help.econ.machine", textCol},
		{"factory.help.econ.buysell", textCol},
		{"", textCol},
		{"factory.help.events", headCol},
		{"factory.help.events.desc", textCol},
		{"factory.help.events.desc2", textCol},
		{"factory.help.events.desc3", textCol},
	}

	for _, l := range lines {
		if y > py+panelH-30 {
			break
		}
		if l.key == "" {
			y += lineH / 2
			continue
		}
		printColoredAt(screen, locale.T(l.key), x, y, l.col)
		y += lineH
	}

	// Close hint at bottom
	closeText := locale.T("factory.help.close")
	printColoredAt(screen, closeText, px+panelW/2-runeLen(closeText)*charW/2, py+panelH-20, closeCol)
}
