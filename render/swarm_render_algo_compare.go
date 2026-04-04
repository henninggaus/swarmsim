package render

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawConvergenceGraph renders a real-time convergence chart for the active
// swarm algorithm. Positioned in the bottom-left of the arena viewport area,
// it shows three lines: green = best fitness, yellow = average fitness,
// cyan = population diversity (spatial spread, independently scaled 0-100).
func drawConvergenceGraph(screen *ebiten.Image, ss *swarm.SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil || len(sa.ConvergenceHistory) < 2 {
		return
	}

	// Graph dimensions and position (bottom-left of arena area)
	const gw = 220
	const gh = 100
	gx := 420 // just right of editor panel (350px + margin)
	gy := int(ss.ArenaH) + 50 - gh - 10

	// Background
	vector.DrawFilledRect(screen, float32(gx), float32(gy), gw, gh,
		color.RGBA{10, 10, 20, 220}, false)
	vector.StrokeRect(screen, float32(gx), float32(gy), gw, gh, 1,
		color.RGBA{60, 80, 120, 150}, false)

	// Title
	algoName := swarm.SwarmAlgorithmName(sa.ActiveAlgo)
	printColoredAt(screen, algoName+" Konvergenz", gx+3, gy+2, color.RGBA{136, 204, 255, 220})

	// Determine visible window (last N samples)
	best := sa.ConvergenceHistory
	avg := sa.ConvergenceAvg
	div := sa.ConvergenceDiversity
	expl := sa.ConvergenceExploration
	n := len(best)
	maxPts := gw - 10 // one pixel per sample
	start := 0
	if n > maxPts {
		start = n - maxPts
	}
	pts := n - start

	// Find min/max for fitness scaling (include archived curves for unified range)
	minV := best[start]
	maxV := best[start]
	for i := start; i < n; i++ {
		if best[i] < minV {
			minV = best[i]
		}
		if best[i] > maxV {
			maxV = best[i]
		}
		if i < len(avg) {
			if avg[i] < minV {
				minV = avg[i]
			}
			if avg[i] > maxV {
				maxV = avg[i]
			}
		}
	}
	// Extend range to include archived convergence curves.
	for _, arch := range ss.ConvergenceArchive {
		if sa.ActiveAlgo == arch.Algo && sa.FitnessFunc == arch.FitnessFunc {
			continue
		}
		for _, v := range arch.BestHistory {
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
	}
	if maxV <= minV {
		maxV = minV + 1
	}

	// Chart area (inset from borders)
	chartX := float32(gx + 5)
	chartY := float32(gy + 14)
	chartW := float32(gw - 10)
	chartH := float32(gh - 20)

	// Axis labels
	printColoredAt(screen, fmt.Sprintf("%.0f", maxV), gx+3, gy+14, color.RGBA{70, 70, 90, 150})
	printColoredAt(screen, fmt.Sprintf("%.0f", minV), gx+3, gy+gh-lineH-2, color.RGBA{70, 70, 90, 150})

	// Draw archived convergence curves from previously tested algorithms.
	// Each archived curve is drawn as a thin semi-transparent line with a
	// distinct color so users can visually compare convergence trajectories.
	archiveColors := []color.RGBA{
		{255, 120, 80, 100},  // orange
		{180, 80, 255, 100},  // purple
		{80, 255, 180, 100},  // mint
		{255, 255, 80, 100},  // yellow
		{80, 180, 255, 100},  // sky blue
		{255, 80, 180, 100},  // pink
		{180, 255, 80, 100},  // lime
		{80, 255, 255, 100},  // teal
	}
	for ai, arch := range ss.ConvergenceArchive {
		if sa.ActiveAlgo == arch.Algo && sa.FitnessFunc == arch.FitnessFunc {
			continue
		}
		ah := arch.BestHistory
		an := len(ah)
		if an < 2 {
			continue
		}
		aStart := 0
		if an > maxPts {
			aStart = an - maxPts
		}
		aPts := an - aStart
		ac := archiveColors[ai%len(archiveColors)]
		for i := 1; i < aPts; i++ {
			x0 := chartX + float32(i-1)/float32(aPts-1)*chartW
			x1 := chartX + float32(i)/float32(aPts-1)*chartW
			v0 := ah[aStart+i-1]
			v1 := ah[aStart+i]
			y0 := chartY + chartH - float32((v0-minV)/(maxV-minV))*chartH
			y1 := chartY + chartH - float32((v1-minV)/(maxV-minV))*chartH
			vector.StrokeLine(screen, x0, y0, x1, y1, 1, ac, false)
		}
	}

	// Draw exploration ratio line (magenta, behind everything, independently scaled 0-100)
	if len(expl) >= n {
		for i := 1; i < pts; i++ {
			e0 := expl[start+i-1]
			e1 := expl[start+i]
			// Skip if algorithm does not report exploration ratio (-1)
			if e0 < 0 || e1 < 0 {
				continue
			}
			x0 := chartX + float32(i-1)/float32(pts-1)*chartW
			x1 := chartX + float32(i)/float32(pts-1)*chartW
			if e0 > 100 {
				e0 = 100
			}
			if e1 > 100 {
				e1 = 100
			}
			y0 := chartY + chartH - float32(e0/100)*chartH
			y1 := chartY + chartH - float32(e1/100)*chartH
			vector.StrokeLine(screen, x0, y0, x1, y1, 1, color.RGBA{220, 100, 220, 90}, false)
		}
	}

	// Draw diversity line (cyan, behind average/best, independently scaled 0-100)
	if len(div) >= n {
		for i := 1; i < pts; i++ {
			x0 := chartX + float32(i-1)/float32(pts-1)*chartW
			x1 := chartX + float32(i)/float32(pts-1)*chartW
			// Diversity is already 0-100, scale directly to chart height
			d0 := div[start+i-1]
			d1 := div[start+i]
			if d0 > 100 {
				d0 = 100
			}
			if d1 > 100 {
				d1 = 100
			}
			y0 := chartY + chartH - float32(d0/100)*chartH
			y1 := chartY + chartH - float32(d1/100)*chartH
			vector.StrokeLine(screen, x0, y0, x1, y1, 1, color.RGBA{80, 200, 220, 100}, false)
		}
	}

	// Draw average line (yellow, behind best)
	if len(avg) >= n {
		for i := 1; i < pts; i++ {
			x0 := chartX + float32(i-1)/float32(pts-1)*chartW
			x1 := chartX + float32(i)/float32(pts-1)*chartW
			y0 := chartY + chartH - float32((avg[start+i-1]-minV)/(maxV-minV))*chartH
			y1 := chartY + chartH - float32((avg[start+i]-minV)/(maxV-minV))*chartH
			vector.StrokeLine(screen, x0, y0, x1, y1, 1, color.RGBA{220, 200, 50, 140}, false)
		}
	}

	// Draw best line (green, on top)
	for i := 1; i < pts; i++ {
		x0 := chartX + float32(i-1)/float32(pts-1)*chartW
		x1 := chartX + float32(i)/float32(pts-1)*chartW
		y0 := chartY + chartH - float32((best[start+i-1]-minV)/(maxV-minV))*chartH
		y1 := chartY + chartH - float32((best[start+i]-minV)/(maxV-minV))*chartH
		vector.StrokeLine(screen, x0, y0, x1, y1, 1.5, color.RGBA{50, 220, 80, 200}, false)
	}

	// Stagnation warning bar: red overlay when stagnation is high
	stagnPct := float32(sa.StagnationCount) / float32(swarm.StagnationThreshold)
	if stagnPct > 1 {
		stagnPct = 1
	}
	if stagnPct > 0.3 {
		// Draw stagnation progress bar at bottom of chart
		barY := chartY + chartH - 3
		barW := chartW * stagnPct
		alpha := uint8(40 + int(stagnPct*120))
		vector.DrawFilledRect(screen, chartX, barY, barW, 3, color.RGBA{255, 60, 40, alpha}, false)
	}

	// Legend
	ly := gy + gh + 2
	vector.DrawFilledRect(screen, float32(gx+3), float32(ly+2), 8, 2, color.RGBA{50, 220, 80, 200}, false)
	printColoredAt(screen, "Best", gx+14, ly, color.RGBA{50, 220, 80, 180})
	vector.DrawFilledRect(screen, float32(gx+50), float32(ly+2), 8, 2, color.RGBA{220, 200, 50, 140}, false)
	printColoredAt(screen, "Avg", gx+61, ly, color.RGBA{220, 200, 50, 140})
	vector.DrawFilledRect(screen, float32(gx+90), float32(ly+2), 8, 2, color.RGBA{80, 200, 220, 140}, false)
	printColoredAt(screen, "Div", gx+101, ly, color.RGBA{80, 200, 220, 140})
	// Show Expl legend only if algorithm reports exploration ratio
	hasExpl := len(expl) > 0 && expl[len(expl)-1] >= 0
	if hasExpl {
		vector.DrawFilledRect(screen, float32(gx+125), float32(ly+2), 8, 2, color.RGBA{220, 100, 220, 140}, false)
		printColoredAt(screen, "Expl", gx+136, ly, color.RGBA{220, 100, 220, 140})
	}

	// Current values
	curDiv := ""
	if len(div) > 0 {
		curDiv = fmt.Sprintf(" D:%.0f%%", div[len(div)-1])
	}
	curExpl := ""
	if hasExpl {
		curExpl = fmt.Sprintf(" E:%.0f%%", expl[len(expl)-1])
	}
	printColoredAt(screen, fmt.Sprintf("Best:%.1f%s%s", best[n-1], curDiv, curExpl), gx+3, ly+lineH, color.RGBA{160, 180, 200, 180})

	// Statistics line: iteration count, stagnation, perturbations
	ly += 2 * lineH
	iterStr := fmt.Sprintf("Iter:%d", sa.TotalIterations)
	stagnStr := ""
	if sa.StagnationCount > 0 {
		stagnStr = fmt.Sprintf(" Stagn:%d/%d", sa.StagnationCount, swarm.StagnationThreshold)
	}
	pertStr := ""
	if sa.PerturbationCount > 0 {
		pertStr = fmt.Sprintf(" Perturb:%d", sa.PerturbationCount)
	}

	stagnCol := color.RGBA{130, 140, 160, 180}
	if stagnPct > 0.7 {
		stagnCol = color.RGBA{255, 120, 60, 220}
	}
	printColoredAt(screen, iterStr+stagnStr+pertStr, gx+3, ly, stagnCol)

	// Archived curve legend: show abbreviation + color swatch for each archived algo.
	archiveDrawn := 0
	for ai, arch := range ss.ConvergenceArchive {
		if sa.ActiveAlgo == arch.Algo && sa.FitnessFunc == arch.FitnessFunc {
			continue
		}
		if len(arch.BestHistory) < 2 {
			continue
		}
		ac := archiveColors[ai%len(archiveColors)]
		acLabel := ac
		acLabel.A = 180
		lx := gx + 3 + archiveDrawn*45
		ly2 := ly + lineH
		vector.DrawFilledRect(screen, float32(lx), float32(ly2+2), 8, 2, ac, false)
		printColoredAt(screen, swarm.SwarmAlgorithmAbbrev(arch.Algo), lx+11, ly2, acLabel)
		archiveDrawn++
	}
}

// drawSearchTrajectory renders a small inset showing the X,Y path of the
// global best solution through the search space over time. Earlier positions
// are drawn in blue, transitioning to red for recent positions. A yellow
// marker highlights the current best. Positioned to the right of the
// convergence graph.
func drawSearchTrajectory(screen *ebiten.Image, ss *swarm.SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil || len(sa.TrajectoryX) < 2 {
		return
	}

	// Panel dimensions and position (right of convergence graph)
	const tw = 100
	const th = 100
	tx := 420 + 220 + 5 // gx + gw + gap
	ty := int(ss.ArenaH) + 50 - th - 10

	// Background
	vector.DrawFilledRect(screen, float32(tx), float32(ty), tw, th,
		color.RGBA{10, 10, 20, 220}, false)
	vector.StrokeRect(screen, float32(tx), float32(ty), tw, th, 1,
		color.RGBA{60, 80, 120, 150}, false)

	// Title
	printColoredAt(screen, "Trajektorie", tx+3, ty+2, color.RGBA{136, 204, 255, 220})

	// Chart inset
	chartX := float32(tx + 5)
	chartY := float32(ty + 14)
	chartW := float32(tw - 10)
	chartH := float32(th - 18)

	// Map arena coordinates [0, ArenaW/H] to chart area
	aw := float32(ss.ArenaW)
	ah := float32(ss.ArenaH)
	if aw < 1 {
		aw = 800
	}
	if ah < 1 {
		ah = 800
	}

	mapX := func(wx float64) float32 { return chartX + float32(wx)/aw*chartW }
	mapY := func(wy float64) float32 { return chartY + float32(wy)/ah*chartH }

	// Draw trajectory polyline with color gradient (blue->red)
	n := len(sa.TrajectoryX)
	for i := 1; i < n; i++ {
		x0, y0 := sa.TrajectoryX[i-1], sa.TrajectoryY[i-1]
		x1, y1 := sa.TrajectoryX[i], sa.TrajectoryY[i]
		// Skip invalid sentinel values
		if x0 < 0 || y0 < 0 || x1 < 0 || y1 < 0 {
			continue
		}
		// Color interpolation: blue (early) -> red (recent)
		t := float32(i) / float32(n)
		r := uint8(60 + t*195)
		g := uint8(60 * (1 - t))
		b := uint8(220 * (1 - t))
		a := uint8(80 + t*140)
		c := color.RGBA{r, g, b, a}
		vector.StrokeLine(screen, mapX(x0), mapY(y0), mapX(x1), mapY(y1), 1, c, false)
	}

	// Current best position marker (yellow dot)
	lastX, lastY := sa.TrajectoryX[n-1], sa.TrajectoryY[n-1]
	if lastX >= 0 && lastY >= 0 {
		cx, cy := mapX(lastX), mapY(lastY)
		vector.DrawFilledCircle(screen, cx, cy, 3, color.RGBA{255, 220, 50, 240}, false)
	}

	// Start position marker (small blue dot)
	for i := 0; i < n; i++ {
		if sa.TrajectoryX[i] >= 0 && sa.TrajectoryY[i] >= 0 {
			sx, sy := mapX(sa.TrajectoryX[i]), mapY(sa.TrajectoryY[i])
			vector.DrawFilledCircle(screen, sx, sy, 2, color.RGBA{80, 120, 255, 180}, false)
			break
		}
	}
}

// drawFitnessHistogram renders a small histogram showing the distribution of
// per-bot fitness values for the active optimisation algorithm. 10 bins are
// used, with bar height proportional to the count in each bin. A colour
// gradient from red (low fitness) to green (high fitness) makes the
// distribution shape immediately visible. Positioned right of the trajectory
// plot.
func drawFitnessHistogram(screen *ebiten.Image, ss *swarm.SwarmState) {
	vals := swarm.GetAlgoFitnessValues(ss)
	if len(vals) < 2 {
		return
	}

	// Panel dimensions and position (right of trajectory plot)
	const hw = 100
	const hh = 100
	hx := 420 + 220 + 5 + 100 + 5 // convergence + gap + trajectory + gap = 750
	hy := int(ss.ArenaH) + 50 - hh - 10

	// Background
	vector.DrawFilledRect(screen, float32(hx), float32(hy), hw, hh,
		color.RGBA{10, 10, 20, 220}, false)
	vector.StrokeRect(screen, float32(hx), float32(hy), hw, hh, 1,
		color.RGBA{60, 80, 120, 150}, false)

	// Title
	printColoredAt(screen, "Fitness-Vertlg", hx+3, hy+2, color.RGBA{136, 204, 255, 220})

	// Compute min/max
	minV, maxV := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if maxV <= minV {
		maxV = minV + 1
	}
	rangeV := maxV - minV

	// Bin into 10 buckets
	const numBins = 10
	var bins [numBins]int
	for _, v := range vals {
		idx := int((v - minV) / rangeV * float64(numBins))
		if idx >= numBins {
			idx = numBins - 1
		}
		if idx < 0 {
			idx = 0
		}
		bins[idx]++
	}

	// Find max bin count for scaling
	maxBin := 1
	for _, c := range bins {
		if c > maxBin {
			maxBin = c
		}
	}

	// Chart area (inset)
	chartX := float32(hx + 5)
	chartY := float32(hy + 14)
	chartW := float32(hw - 10)
	chartH := float32(hh - 24)
	barW := chartW / float32(numBins)

	// Draw bars with red->green gradient
	for i := 0; i < numBins; i++ {
		t := float32(i) / float32(numBins-1)
		barH := float32(bins[i]) / float32(maxBin) * chartH
		bx := chartX + float32(i)*barW
		by := chartY + chartH - barH

		// Color: red (low fitness) -> yellow (mid) -> green (high fitness)
		var r, g, b uint8
		if t < 0.5 {
			r = 220
			g = uint8(200 * t * 2)
			b = 40
		} else {
			r = uint8(220 * (1 - t) * 2)
			g = 200
			b = 40
		}
		col := color.RGBA{r, g, b, 200}
		if barH > 0.5 {
			vector.DrawFilledRect(screen, bx+0.5, by, barW-1, barH, col, false)
		}
	}

	// Axis labels (min/max fitness)
	printColoredAt(screen, fmt.Sprintf("%.0f", minV), hx+3, hy+hh-lineH, color.RGBA{100, 100, 120, 160})
	printColoredAt(screen, fmt.Sprintf("%.0f", maxV), hx+hw-30, hy+hh-lineH, color.RGBA{100, 100, 120, 160})
}

// drawAlgoScoreboard renders a compact ranking of algorithms that have been
// tested on the current fitness landscape. Positioned below the convergence
// graph (or at the same location if no graph is visible). Sorted by best
// fitness descending so the top performer is first.
func drawAlgoScoreboard(screen *ebiten.Image, ss *swarm.SwarmState) {
	board := ss.AlgoScoreboard
	if len(board) == 0 {
		return
	}

	// Sort by best fitness descending
	sorted := make([]swarm.AlgoPerformanceRecord, len(board))
	copy(sorted, board)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BestFitness > sorted[j].BestFitness
	})

	// Position: right of histogram panel (convergence + trajectory + histogram + gaps)
	const sbW = 200
	sbX := 420 + 220 + 5 + 100 + 5 + 100 + 5 // gx + convW + gap + trajW + gap + histW + gap = 855
	sbY := int(ss.ArenaH) + 50 - 100 - 10      // same top as convergence graph

	// Background
	rowH := lineH + 2
	headerH := lineH + 4
	sbH := headerH + len(sorted)*rowH + 4
	vector.DrawFilledRect(screen, float32(sbX), float32(sbY), float32(sbW), float32(sbH),
		color.RGBA{10, 10, 20, 220}, false)
	vector.StrokeRect(screen, float32(sbX), float32(sbY), float32(sbW), float32(sbH), 1,
		color.RGBA{60, 80, 120, 150}, false)

	// Title
	printColoredAt(screen, locale.T("ui.algo_ranking"), sbX+3, sbY+2, color.RGBA{136, 204, 255, 220})

	// Column headers
	y := sbY + headerH
	printColoredAt(screen, "#", sbX+3, y, color.RGBA{100, 100, 120, 160})
	printColoredAt(screen, locale.T("ui.algorithm"), sbX+16, y, color.RGBA{100, 100, 120, 160})
	printColoredAt(screen, locale.T("genome.col_fitness"), sbX+130, y, color.RGBA{100, 100, 120, 160})
	y += rowH

	// Rows
	maxEntries := 8 // limit visible entries
	if len(sorted) < maxEntries {
		maxEntries = len(sorted)
	}
	for rank, rec := range sorted[:maxEntries] {
		// Rank color: gold/silver/bronze for top 3
		var rankCol color.RGBA
		switch rank {
		case 0:
			rankCol = color.RGBA{255, 215, 0, 220} // gold
		case 1:
			rankCol = color.RGBA{200, 200, 210, 200} // silver
		case 2:
			rankCol = color.RGBA{205, 127, 50, 200} // bronze
		default:
			rankCol = color.RGBA{140, 150, 170, 180}
		}

		name := swarm.SwarmAlgorithmName(rec.Algo)
		// Truncate long names
		if len(name) > 16 {
			name = name[:15] + "."
		}

		printColoredAt(screen, fmt.Sprintf("%d", rank+1), sbX+3, y, rankCol)
		printColoredAt(screen, name, sbX+16, y, rankCol)
		printColoredAt(screen, fmt.Sprintf("%.1f", rec.BestFitness), sbX+130, y, rankCol)
		y += rowH
	}
}

// drawAlgoTournamentProgress renders a compact progress bar showing which
// algorithm is currently being benchmarked and overall tournament progress.
func drawAlgoTournamentProgress(screen *ebiten.Image, ss *swarm.SwarmState) {
	const barW = 300
	const barH = 28
	barX := float32(250)
	barY := float32(5)

	// Background
	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{10, 15, 30, 230}, false)
	vector.StrokeRect(screen, barX, barY, barW, barH, 1, color.RGBA{80, 140, 255, 180}, false)

	// Progress fraction: completed algorithms + partial progress of current one
	total := float32(ss.AlgoTournamentTotal)
	if total < 1 {
		total = 1
	}
	tickFrac := 1.0 - float32(ss.AlgoTournamentTicks)/float32(swarm.AlgoTournamentTicksPerAlgo)
	progress := (float32(ss.AlgoTournamentDone) + tickFrac) / total

	// Progress bar fill
	fillW := (barW - 4) * progress
	fillCol := color.RGBA{40, 180, 80, 200}
	vector.DrawFilledRect(screen, barX+2, barY+2, fillW, barH-4, fillCol, false)

	// Text: "AUTO-TURNIER: PSO (3/20)"
	algoName := swarm.SwarmAlgorithmAbbrev(ss.AlgoTournamentCur)
	label := fmt.Sprintf("AUTO-TURNIER: %s (%d/%d)", algoName, ss.AlgoTournamentDone+1, ss.AlgoTournamentTotal)
	printColoredAt(screen, label, int(barX)+6, int(barY)+4, color.RGBA{220, 230, 255, 255})

	// Sub-line: ticks remaining for current algo
	tickPct := int(tickFrac * 100)
	sub := fmt.Sprintf("%d%%", tickPct)
	printColoredAt(screen, sub, int(barX)+barW-35, int(barY)+4, color.RGBA{160, 180, 200, 200})
}

// drawAlgoRadarChart renders a radar (spider) chart comparing algorithm
// performance across 4 axes: Best Fitness, Convergence Speed, Avg Fitness,
// and Final Diversity. Each algorithm is drawn as a colored polygon.
// Toggled with Ctrl+=. Requires at least 2 entries in AlgoScoreboard.
func drawAlgoRadarChart(screen *ebiten.Image, ss *swarm.SwarmState) {
	board := ss.AlgoScoreboard
	if len(board) < 2 {
		return
	}

	// Sort by best fitness descending for consistent ordering
	sorted := make([]swarm.AlgoPerformanceRecord, len(board))
	copy(sorted, board)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BestFitness > sorted[j].BestFitness
	})

	// Limit to top 6 algorithms for readability
	maxAlgos := 6
	if len(sorted) < maxAlgos {
		maxAlgos = len(sorted)
	}
	sorted = sorted[:maxAlgos]

	// Axes: BestFitness, ConvergenceSpeed (inverted: lower=better->higher value),
	// AvgFitness, FinalDiversity
	const numAxes = 4
	axisLabels := [numAxes]string{"Beste Fitness", "Konv.-Speed", "Avg Fitness", "Diversitaet"}

	// Find max values for normalisation
	maxBest := 0.0
	maxSpeed := 0.0
	maxAvg := 0.0
	maxDiv := 0.0
	for _, r := range sorted {
		if r.BestFitness > maxBest {
			maxBest = r.BestFitness
		}
		if r.ConvergenceSpeed > maxSpeed {
			maxSpeed = r.ConvergenceSpeed
		}
		if r.AvgFitness > maxAvg {
			maxAvg = r.AvgFitness
		}
		if r.FinalDiversity > maxDiv {
			maxDiv = r.FinalDiversity
		}
	}
	if maxBest < 1 {
		maxBest = 1
	}
	if maxSpeed < 1 {
		maxSpeed = 1
	}
	if maxAvg < 1 {
		maxAvg = 1
	}
	if maxDiv < 0.01 {
		maxDiv = 0.01
	}

	// Chart geometry — position in upper-right of arena
	cx := float32(ss.ArenaW) - 10 - 110 // center X
	cy := float32(130)                   // center Y
	radius := float32(90)

	// Background circle
	const bgAlpha = 200
	vector.DrawFilledCircle(screen, cx, cy, radius+15, color.RGBA{10, 10, 25, bgAlpha}, false)
	vector.StrokeCircle(screen, cx, cy, radius+15, 1, color.RGBA{60, 80, 120, 150}, false)

	// Title
	printColoredAt(screen, locale.T("ui.algo_radar"), int(cx)-55, int(cy-radius)-25,
		color.RGBA{136, 204, 255, 220})

	// Draw axis lines and labels
	for a := 0; a < numAxes; a++ {
		angle := float64(a)*2*math.Pi/float64(numAxes) - math.Pi/2
		ex := cx + radius*float32(math.Cos(angle))
		ey := cy + radius*float32(math.Sin(angle))
		vector.StrokeLine(screen, cx, cy, ex, ey, 1, color.RGBA{50, 60, 80, 180}, false)

		// Label position (slightly past endpoint)
		lx := cx + (radius+12)*float32(math.Cos(angle))
		ly := cy + (radius+12)*float32(math.Sin(angle))
		label := axisLabels[a]
		// Center text approximately
		labelOff := runeLen(label) * charW / 2
		printColoredAt(screen, label, int(lx)-labelOff, int(ly)-5,
			color.RGBA{140, 160, 190, 200})
	}

	// Draw concentric guide rings at 25%, 50%, 75%
	for _, frac := range []float32{0.25, 0.5, 0.75} {
		r := radius * frac
		vector.StrokeCircle(screen, cx, cy, r, 0.5, color.RGBA{40, 50, 70, 100}, false)
	}

	// Algorithm colors (distinct, semi-transparent for fill)
	algoColors := []color.RGBA{
		{255, 100, 80, 255},  // red-orange
		{80, 180, 255, 255},  // sky blue
		{120, 255, 120, 255}, // green
		{255, 200, 60, 255},  // gold
		{200, 120, 255, 255}, // purple
		{255, 140, 200, 255}, // pink
	}

	// Draw polygon for each algorithm
	for ai, rec := range sorted {
		col := algoColors[ai%len(algoColors)]
		fillCol := color.RGBA{col.R, col.G, col.B, 40}

		// Normalise values to [0, 1]
		vals := [numAxes]float64{
			rec.BestFitness / maxBest,
			1.0 - rec.ConvergenceSpeed/maxSpeed, // invert: lower speed = better
			rec.AvgFitness / maxAvg,
			rec.FinalDiversity / maxDiv,
		}

		// Compute polygon vertices
		var verts [numAxes][2]float32
		for a := 0; a < numAxes; a++ {
			angle := float64(a)*2*math.Pi/float64(numAxes) - math.Pi/2
			v := vals[a]
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			r := float32(v) * radius
			verts[a] = [2]float32{
				cx + r*float32(math.Cos(angle)),
				cy + r*float32(math.Sin(angle)),
			}
		}

		// Draw filled triangles (fan from center) for the polygon
		for a := 0; a < numAxes; a++ {
			next := (a + 1) % numAxes
			drawTriangleFill(screen, cx, cy, verts[a][0], verts[a][1],
				verts[next][0], verts[next][1], fillCol)
		}

		// Draw polygon outline
		for a := 0; a < numAxes; a++ {
			next := (a + 1) % numAxes
			vector.StrokeLine(screen, verts[a][0], verts[a][1],
				verts[next][0], verts[next][1], 1.5, col, false)
		}

		// Draw vertex dots
		for a := 0; a < numAxes; a++ {
			vector.DrawFilledCircle(screen, verts[a][0], verts[a][1], 2.5, col, false)
		}
	}

	// Legend
	ly := int(cy + radius + 20)
	for ai, rec := range sorted {
		col := algoColors[ai%len(algoColors)]
		name := swarm.SwarmAlgorithmAbbrev(rec.Algo)
		lx := int(cx) - 55 + (ai%3)*42
		if ai >= 3 {
			ly = int(cy+radius) + 20 + lineH + 2
			lx = int(cx) - 55 + (ai%3)*42
		}
		// Color swatch
		vector.DrawFilledRect(screen, float32(lx), float32(ly+1), 8, 8, col, false)
		printColoredAt(screen, name, lx+10, ly, color.RGBA{180, 190, 210, 220})
	}
}

// radarWhitePixel is a 1x1 white image used as source texture for DrawTriangles.
var radarWhitePixel *ebiten.Image

func getRadarWhitePixel() *ebiten.Image {
	if radarWhitePixel == nil {
		radarWhitePixel = ebiten.NewImage(1, 1)
		radarWhitePixel.Fill(color.White)
	}
	return radarWhitePixel
}

// drawTriangleFill renders a filled triangle using ebiten's vertex-based rendering.
func drawTriangleFill(screen *ebiten.Image, x0, y0, x1, y1, x2, y2 float32, col color.RGBA) {
	vs := []ebiten.Vertex{
		{DstX: x0, DstY: y0, SrcX: 0, SrcY: 0, ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255, ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
		{DstX: x1, DstY: y1, SrcX: 0, SrcY: 0, ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255, ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
		{DstX: x2, DstY: y2, SrcX: 0, SrcY: 0, ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255, ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
	}
	is := []uint16{0, 1, 2}
	screen.DrawTriangles(vs, is, getRadarWhitePixel(), nil)
}
