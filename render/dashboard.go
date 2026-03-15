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

	// Dashboard background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h),
		color.RGBA{15, 15, 25, 230}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1,
		color.RGBA{80, 80, 100, 150}, false)

	headerCol := color.RGBA{0, 200, 220, 255}
	dimCol := color.RGBA{120, 120, 140, 220}
	cx := x + 5
	cy := y + 5

	printColoredAt(screen, "Dashboard", cx, cy, headerCol)
	cy += 18

	// 1. Fitness Graph (if evolution or GP active)
	if (ss.EvolutionOn || ss.GPEnabled) && len(ss.FitnessHistory) > 1 {
		printColoredAt(screen, "Fitness", cx, cy, dimCol)
		cy += 14
		drawDashFitnessGraph(screen, ss, cx, cy, w-15, 60)
		cy += 65
	}

	// 2. Delivery Rate Bar Chart
	if ss.DeliveryOn && len(st.DeliveryBuckets) > 0 {
		printColoredAt(screen, "Delivery Rate", cx, cy, dimCol)
		cy += 14
		drawDashDeliveryChart(screen, st, ss, cx, cy, w-15, 60)
		cy += 65
	}

	// 3. Heatmap
	if st.HeatmapMax > 0 {
		printColoredAt(screen, "Heatmap", cx, cy, dimCol)
		cy += 14
		drawDashHeatmap(screen, st, cx, cy, w-15, w-15)
		cy += w - 10
	}

	// 4. Bot Efficiency Ranking
	if len(st.BotRankings) > 0 && ss.DeliveryOn {
		printColoredAt(screen, "Top Bots", cx, cy, dimCol)
		cy += 14
		drawDashRanking(screen, st, cx, cy, w-15, 80)
		cy += 85
	}

	// 5. Event Ticker
	if len(st.EventTicker) > 0 {
		printColoredAt(screen, "Events", cx, cy, dimCol)
		cy += 14
		drawDashTicker(screen, st, cx, cy, w-15, 80)
	}
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
	// Draw lines
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
		vector.StrokeLine(screen, x0, y0b, x1, y1b, 1, color.RGBA{80, 255, 80, 220}, false)
		// Avg (yellow)
		y0a := float32(gy+gh) - float32(history[start+i-1].Avg/maxFit)*float32(gh)
		y1a := float32(gy+gh) - float32(history[start+i].Avg/maxFit)*float32(gh)
		vector.StrokeLine(screen, x0, y0a, x1, y1a, 1, color.RGBA{255, 200, 50, 200}, false)
	}
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

			// Color gradient: blue → green → yellow → red
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

		text := fmt.Sprintf("#%d Bot%d: %dd %davg",
			i+1, entry.BotIdx, entry.Deliveries, entry.AvgTime)
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
