package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawDashboard renders the statistics dashboard on the right side of the screen.
func DrawDashboard(screen *ebiten.Image, ss *swarm.SwarmState, x, y, w, h int) {
	st := ss.StatsTracker
	if st == nil {
		return
	}

	// Dashboard background with subtle gradient effect
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h),
		color.RGBA{15, 15, 25, 235}, false)
	// Top highlight
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), 2,
		color.RGBA{80, 120, 200, 80}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1,
		color.RGBA{60, 80, 120, 150}, false)

	headerCol := color.RGBA{136, 204, 255, 255}
	sectionCol := color.RGBA{180, 200, 230, 220}
	dimCol := color.RGBA{110, 115, 130, 220}
	cx := x + 5
	cy := y + 5

	printColoredAt(screen, "DASHBOARD", cx, cy, headerCol)
	printColoredAt(screen, "(Taste D)", cx+70, cy, dimCol)
	cy += 18

	// 1. Fitness Graph (if evolution, GP, or neuro active)
	if (ss.EvolutionOn || ss.GPEnabled || ss.NeuroEnabled) && len(ss.FitnessHistory) > 1 {
		label := "FITNESS-VERLAUF (B=Baseline)"
		if len(ss.BaselineFitness) > 0 {
			label = "FITNESS-VERLAUF (Blau=Baseline)"
		}
		drawDashSectionHeader(screen, cx, cy, w-15, label, sectionCol)
		cy += 14
		printColoredAt(screen, "Gruen=Best  Gelb=Durchschnitt", cx, cy, dimCol)
		cy += 10
		desc := "Zeigt wie Evolution die Parameter optimiert"
		if ss.NeuroEnabled {
			desc = "Zeigt wie Neuro-Netze durch Evolution lernen"
		}
		printColoredAt(screen, desc, cx, cy, color.RGBA{80, 80, 100, 180})
		cy += 12
		drawDashFitnessGraph(screen, ss, cx, cy, w-15, 60)
		cy += 65
	}

	// 1b. Learning Speed Curve (fitness delta per generation)
	if (ss.EvolutionOn || ss.GPEnabled || ss.NeuroEnabled) && len(ss.FitnessHistory) > 2 {
		drawDashSectionHeader(screen, cx, cy, w-15, "LERNGESCHWINDIGKEIT", sectionCol)
		cy += 14
		printColoredAt(screen, "Fitness-Aenderung pro Generation", cx, cy, dimCol)
		cy += 12
		drawDashSpeedCurve(screen, ss, cx, cy, w-15, 50)
		cy += 55
	}

	// 2. Delivery Rate Bar Chart
	if ss.DeliveryOn && len(st.DeliveryBuckets) > 0 {
		drawDashSectionHeader(screen, cx, cy, w-15, "LIEFERRATE", sectionCol)
		cy += 14
		if ss.TeamsEnabled {
			printColoredAt(screen, "Blau=Team A  Rot=Team B (pro 500 Ticks)", cx, cy, dimCol)
		} else {
			printColoredAt(screen, "Gruen=Richtig  Rot=Falsch (pro 500 Ticks)", cx, cy, dimCol)
		}
		cy += 12
		drawDashDeliveryChart(screen, st, ss, cx, cy, w-15, 60)
		cy += 65
	}

	// 3. Heatmap (toggle between motion and action with A key)
	if st.ShowActionHeat && st.ActionHeatmapMax > 0 {
		drawDashSectionHeader(screen, cx, cy, w-15, "AKTIONS-HEATMAP (A)", sectionCol)
		cy += 14
		printColoredAt(screen, "Wo passieren Pickups & Deliveries?", cx, cy, dimCol)
		cy += 10
		printColoredAt(screen, "Blau=wenig  Gelb=mittel  Rot=Hotspot", cx, cy, color.RGBA{80, 80, 100, 180})
		cy += 12
		drawDashActionHeatmap(screen, st, cx, cy, w-15, w-15)
		cy += w - 10
	} else if st.HeatmapMax > 0 {
		drawDashSectionHeader(screen, cx, cy, w-15, "BEWEGUNGS-HEATMAP (A)", sectionCol)
		cy += 14
		printColoredAt(screen, "Wo bewegen sich Bots am meisten?", cx, cy, dimCol)
		cy += 10
		printColoredAt(screen, "Blau=wenig  Gruen=mittel  Rot=haeufig", cx, cy, color.RGBA{80, 80, 100, 180})
		cy += 12
		drawDashHeatmap(screen, st, cx, cy, w-15, w-15)
		cy += w - 10
	}

	// 4. Bot Efficiency Ranking
	if len(st.BotRankings) > 0 && ss.DeliveryOn {
		drawDashSectionHeader(screen, cx, cy, w-15, "TOP BOTS", sectionCol)
		cy += 14
		printColoredAt(screen, "Sortiert nach Lieferungen (avg = Lieferzeit)", cx, cy, dimCol)
		cy += 12
		drawDashRanking(screen, st, cx, cy, w-15, 80)
		cy += 85
	}

	// 5. Event Ticker
	if len(st.EventTicker) > 0 {
		drawDashSectionHeader(screen, cx, cy, w-15, "LIVE-EVENTS", sectionCol)
		cy += 14
		printColoredAt(screen, "Letzte Aktionen: Pickup, Delivery, Respawn...", cx, cy, dimCol)
		cy += 12
		drawDashTicker(screen, st, cx, cy, w-15, 80)
	}
}

// drawDashSectionHeader draws a section header with a subtle background line.
func drawDashSectionHeader(screen *ebiten.Image, x, y, w int, title string, col color.RGBA) {
	vector.DrawFilledRect(screen, float32(x-2), float32(y), float32(w+4), float32(lineH-2),
		color.RGBA{25, 30, 50, 200}, false)
	vector.DrawFilledRect(screen, float32(x-2), float32(y+lineH-3), float32(w+4), 1,
		color.RGBA{60, 80, 120, 100}, false)
	printColoredAt(screen, title, x, y, col)
}

// drawDashFitnessGraph draws a mini fitness graph.
func drawDashFitnessGraph(screen *ebiten.Image, ss *swarm.SwarmState, gx, gy, gw, gh int) {
	vector.DrawFilledRect(screen, float32(gx), float32(gy), float32(gw), float32(gh),
		color.RGBA{5, 5, 15, 200}, false)

	history := ss.FitnessHistory
	n := len(history)
	if n < 2 {
		return
	}
	// Find max
	maxFit := 1.0
	for _, h := range history {
		if h.Best > maxFit {
			maxFit = h.Best
		}
	}

	// Include baseline in max calculation
	baseline := ss.BaselineFitness
	for _, h := range baseline {
		if h.Best > maxFit {
			maxFit = h.Best
		}
	}

	// Axis labels
	printColoredAt(screen, fmt.Sprintf("%.0f", maxFit), gx+2, gy+2, color.RGBA{80, 80, 100, 150})
	printColoredAt(screen, "0", gx+2, gy+gh-lineH, color.RGBA{80, 80, 100, 150})

	// Draw baseline comparison (dim blue, behind current)
	if len(baseline) > 1 {
		bStart := 0
		bN := len(baseline)
		if bN > 50 {
			bStart = bN - 50
		}
		bPts := bN - bStart
		for i := 1; i < bPts; i++ {
			x0 := float32(gx) + float32(i-1)/float32(bPts-1)*float32(gw)
			x1 := float32(gx) + float32(i)/float32(bPts-1)*float32(gw)
			y0b := float32(gy+gh) - float32(baseline[bStart+i-1].Best/maxFit)*float32(gh)
			y1b := float32(gy+gh) - float32(baseline[bStart+i].Best/maxFit)*float32(gh)
			vector.StrokeLine(screen, x0, y0b, x1, y1b, 1, color.RGBA{80, 120, 200, 100}, false)
		}
		// Label
		printColoredAt(screen, ss.BaselineLabel, gx+gw/2, gy+2, color.RGBA{80, 120, 200, 120})
	}

	// Draw current lines
	start := 0
	if n > 50 {
		start = n - 50
	}
	pts := n - start
	for i := 1; i < pts; i++ {
		x0 := float32(gx) + float32(i-1)/float32(pts-1)*float32(gw)
		x1 := float32(gx) + float32(i)/float32(pts-1)*float32(gw)
		// Best (green)
		y0b := float32(gy+gh) - float32(history[start+i-1].Best/maxFit)*float32(gh)
		y1b := float32(gy+gh) - float32(history[start+i].Best/maxFit)*float32(gh)
		vector.StrokeLine(screen, x0, y0b, x1, y1b, 1.5, color.RGBA{80, 255, 80, 220}, false)
		// Avg (yellow)
		y0a := float32(gy+gh) - float32(history[start+i-1].Avg/maxFit)*float32(gh)
		y1a := float32(gy+gh) - float32(history[start+i].Avg/maxFit)*float32(gh)
		vector.StrokeLine(screen, x0, y0a, x1, y1a, 1, color.RGBA{255, 200, 50, 180}, false)
	}

	// Generation count label
	genLabel := fmt.Sprintf("Gen %d-%d", start+1, n)
	printColoredAt(screen, genLabel, gx+gw-len(genLabel)*charW-2, gy+gh-lineH, color.RGBA{80, 80, 100, 150})
}

// drawDashSpeedCurve draws fitness delta (learning speed) per generation.
func drawDashSpeedCurve(screen *ebiten.Image, ss *swarm.SwarmState, gx, gy, gw, gh int) {
	vector.DrawFilledRect(screen, float32(gx), float32(gy), float32(gw), float32(gh),
		color.RGBA{5, 5, 15, 200}, false)

	history := ss.FitnessHistory
	n := len(history)
	if n < 3 {
		return
	}

	// Compute deltas
	start := 0
	if n > 50 {
		start = n - 50
	}
	pts := n - start
	deltas := make([]float64, pts-1)
	maxAbs := 1.0
	for i := 1; i < pts; i++ {
		d := history[start+i].Best - history[start+i-1].Best
		deltas[i-1] = d
		if d > maxAbs {
			maxAbs = d
		}
		if -d > maxAbs {
			maxAbs = -d
		}
	}

	// Zero line at center
	midY := float32(gy) + float32(gh)/2
	vector.StrokeLine(screen, float32(gx), midY, float32(gx+gw), midY, 1,
		color.RGBA{60, 60, 80, 150}, false)

	// Draw bars
	barW := float32(gw) / float32(len(deltas))
	if barW < 1 {
		barW = 1
	}
	for i, d := range deltas {
		bx := float32(gx) + float32(i)*barW
		h := float32(d/maxAbs) * float32(gh/2)
		if d >= 0 {
			// Green bar up
			vector.DrawFilledRect(screen, bx, midY-h, barW, h,
				color.RGBA{80, 220, 80, 200}, false)
		} else {
			// Red bar down
			vector.DrawFilledRect(screen, bx, midY, barW, -h,
				color.RGBA{220, 80, 80, 200}, false)
		}
	}

	// Labels
	printColoredAt(screen, fmt.Sprintf("+%.0f", maxAbs), gx+2, gy+1, color.RGBA{80, 80, 100, 150})
	printColoredAt(screen, fmt.Sprintf("-%.0f", maxAbs), gx+2, gy+gh-lineH, color.RGBA{80, 80, 100, 150})
	printColoredAt(screen, "Gruen=besser Rot=schlechter", gx+gw/3, gy+gh-lineH, color.RGBA{60, 60, 80, 120})
}

// drawDashDeliveryChart draws a bar chart of deliveries per window.
func drawDashDeliveryChart(screen *ebiten.Image, st *swarm.StatsTracker, ss *swarm.SwarmState, gx, gy, gw, gh int) {
	vector.DrawFilledRect(screen, float32(gx), float32(gy), float32(gw), float32(gh),
		color.RGBA{5, 5, 15, 200}, false)

	n := len(st.DeliveryBuckets)
	if n == 0 {
		return
	}
	// Show last 6 buckets
	start := 0
	if n > 6 {
		start = n - 6
	}
	visible := n - start
	barW := float32(gw) / float32(visible) * 0.8
	gap := float32(gw) / float32(visible) * 0.2

	// Find max
	maxVal := 1
	for i := start; i < n; i++ {
		if st.DeliveryBuckets[i] > maxVal {
			maxVal = st.DeliveryBuckets[i]
		}
	}

	for i := start; i < n; i++ {
		idx := i - start
		bx := float32(gx) + float32(idx)*(barW+gap) + gap/2

		if ss.TeamsEnabled && i < len(st.TeamABuckets) && i < len(st.TeamBBuckets) {
			// Team bars side-by-side
			halfW := barW / 2
			hA := float32(st.TeamABuckets[i]) / float32(maxVal) * float32(gh)
			hB := float32(st.TeamBBuckets[i]) / float32(maxVal) * float32(gh)
			vector.DrawFilledRect(screen, bx, float32(gy+gh)-hA, halfW, hA,
				color.RGBA{80, 120, 255, 200}, false)
			vector.DrawFilledRect(screen, bx+halfW, float32(gy+gh)-hB, halfW, hB,
				color.RGBA{255, 80, 80, 200}, false)
		} else {
			// Correct (green) + wrong (red) stacked
			correct := 0
			wrong := 0
			if i < len(st.CorrectBuckets) {
				correct = st.CorrectBuckets[i]
			}
			if i < len(st.WrongBuckets) {
				wrong = st.WrongBuckets[i]
			}
			hC := float32(correct) / float32(maxVal) * float32(gh)
			hW := float32(wrong) / float32(maxVal) * float32(gh)
			vector.DrawFilledRect(screen, bx, float32(gy+gh)-hC-hW, barW, hC,
				color.RGBA{80, 200, 80, 200}, false)
			vector.DrawFilledRect(screen, bx, float32(gy+gh)-hW, barW, hW,
				color.RGBA{200, 80, 80, 200}, false)
		}

		// Bucket value label
		val := st.DeliveryBuckets[i]
		if val > 0 {
			label := fmt.Sprintf("%d", val)
			printColoredAt(screen, label, int(bx)+int(barW/2)-len(label)*charW/2, gy+gh-lineH,
				color.RGBA{180, 180, 200, 150})
		}
	}
}

// drawDashHeatmap draws a miniature heatmap of bot movement.
func drawDashHeatmap(screen *ebiten.Image, st *swarm.StatsTracker, gx, gy, gw, gh int) {
	if st.HeatmapMax <= 0 {
		return
	}
	cellW := float32(gw) / 80.0
	cellH := float32(gh) / 60.0

	for cx := 0; cx < 80; cx++ {
		for cy := 0; cy < 60; cy++ {
			val := st.HeatmapGrid[cx][cy]
			if val == 0 {
				continue
			}
			// Normalize to 0-1
			frac := float32(val) / float32(st.HeatmapMax)

			// Color gradient: blue -> green -> yellow -> red
			var col color.RGBA
			switch {
			case frac < 0.25:
				t := frac * 4
				col = color.RGBA{0, 0, uint8(100 + 155*t), 180}
			case frac < 0.5:
				t := (frac - 0.25) * 4
				col = color.RGBA{0, uint8(255 * t), uint8(255 * (1 - t)), 180}
			case frac < 0.75:
				t := (frac - 0.5) * 4
				col = color.RGBA{uint8(255 * t), 255, 0, 180}
			default:
				t := (frac - 0.75) * 4
				col = color.RGBA{255, uint8(255 * (1 - t)), 0, 200}
			}

			px := float32(gx) + float32(cx)*cellW
			py := float32(gy) + float32(cy)*cellH
			vector.DrawFilledRect(screen, px, py, cellW+1, cellH+1, col, false)
		}
	}
}

// drawDashActionHeatmap draws a heatmap of pickup/drop events.
func drawDashActionHeatmap(screen *ebiten.Image, st *swarm.StatsTracker, gx, gy, gw, gh int) {
	if st.ActionHeatmapMax <= 0 {
		return
	}
	cellW := float32(gw) / 80.0
	cellH := float32(gh) / 60.0

	for cx := 0; cx < 80; cx++ {
		for cy := 0; cy < 60; cy++ {
			val := st.ActionHeatmap[cx][cy]
			if val == 0 {
				continue
			}
			frac := float32(val) / float32(st.ActionHeatmapMax)

			// Color gradient: blue -> yellow -> orange -> red
			var col color.RGBA
			switch {
			case frac < 0.33:
				t := frac * 3
				col = color.RGBA{0, 0, uint8(120 + 135*t), 200}
			case frac < 0.66:
				t := (frac - 0.33) * 3
				col = color.RGBA{uint8(255 * t), uint8(200 * t), uint8(255 * (1 - t)), 200}
			default:
				t := (frac - 0.66) * 3
				col = color.RGBA{255, uint8(200 * (1 - t)), 0, 220}
			}

			px := float32(gx) + float32(cx)*cellW
			py := float32(gy) + float32(cy)*cellH
			vector.DrawFilledRect(screen, px, py, cellW+1, cellH+1, col, false)
		}
	}
}

// drawDashRanking draws the top 5 bots by deliveries.
func drawDashRanking(screen *ebiten.Image, st *swarm.StatsTracker, gx, gy, gw, gh int) {
	maxShow := 5
	if maxShow > len(st.BotRankings) {
		maxShow = len(st.BotRankings)
	}

	for i := 0; i < maxShow; i++ {
		entry := &st.BotRankings[i]
		var col color.RGBA
		switch {
		case entry.Deliveries >= 10:
			col = color.RGBA{80, 255, 80, 220} // green
		case entry.Deliveries >= 5:
			col = color.RGBA{255, 200, 50, 220} // yellow
		default:
			col = color.RGBA{255, 100, 80, 220} // red
		}

		// Medal emoji equivalent
		medal := " "
		if i == 0 {
			medal = "*"
		}

		text := fmt.Sprintf("%s#%d Bot%d: %d Lieferungen (%d avg)",
			medal, i+1, entry.BotIdx, entry.Deliveries, entry.AvgTime)
		printColoredAt(screen, text, gx, gy+i*14, col)
	}
}

// drawDashTicker draws the scrolling event ticker.
func drawDashTicker(screen *ebiten.Image, st *swarm.StatsTracker, gx, gy, gw, gh int) {
	// Show last 6 events
	start := 0
	if len(st.EventTicker) > 6 {
		start = len(st.EventTicker) - 6
	}
	tickerCol := color.RGBA{160, 170, 190, 200}
	for i := start; i < len(st.EventTicker); i++ {
		text := st.EventTicker[i]
		if len(text) > 38 {
			text = text[:38]
		}
		printColoredAt(screen, text, gx, gy+(i-start)*13, tickerCol)
	}
}
