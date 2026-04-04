package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	chartW      = 350
	chartH      = 180
	chartMargin = 10
	chartMaxPts = 60 // max data points shown
)

// DrawLiveChart renders a scrolling line chart of delivery stats over time.
func DrawLiveChart(screen *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.StatsTracker
	if st == nil {
		return
	}

	sw := screen.Bounds().Dx()
	px := sw - chartW - chartMargin
	py := screen.Bounds().Dy() - chartH - chartMargin - 30

	// Background
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(chartW), float32(chartH), color.RGBA{10, 12, 25, 230}, false)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(chartW), float32(chartH), 1, color.RGBA{60, 100, 180, 180}, false)

	// Title
	printColoredAt(screen, "LIVE-STATISTIK", px+6, py+4, ColorBrightBlue)

	// Chart area
	cx := px + 35
	cy := py + 22
	cw := chartW - 45
	ch := chartH - 35

	// Grid lines
	for i := 0; i <= 4; i++ {
		gy := float32(cy) + float32(ch)*float32(i)/4.0
		vector.StrokeLine(screen, float32(cx), gy, float32(cx+cw), gy, 0.5,
			color.RGBA{40, 50, 70, 150}, false)
	}

	// Get data
	nCorrect := len(st.CorrectBuckets)
	nWrong := len(st.WrongBuckets)
	if nCorrect < 2 {
		printColoredAt(screen, "Warte auf Daten...", cx+20, cy+ch/2, color.RGBA{100, 100, 120, 200})
		return
	}

	// Find visible range and max value
	startIdx := 0
	if nCorrect > chartMaxPts {
		startIdx = nCorrect - chartMaxPts
	}
	maxVal := 1
	for i := startIdx; i < nCorrect; i++ {
		v := st.CorrectBuckets[i]
		if i < nWrong {
			v += st.WrongBuckets[i]
		}
		if v > maxVal {
			maxVal = v
		}
	}
	// Add headroom
	maxVal = maxVal + maxVal/5
	if maxVal < 5 {
		maxVal = 5
	}

	// Y-axis labels
	for i := 0; i <= 4; i++ {
		val := maxVal * (4 - i) / 4
		gy := cy + ch*i/4
		printColoredAt(screen, fmt.Sprintf("%d", val), px+4, gy-4, color.RGBA{100, 110, 130, 200})
	}

	// Draw lines
	pts := nCorrect - startIdx
	if pts < 2 {
		return
	}

	// Correct deliveries (green line)
	drawChartLine(screen, st.CorrectBuckets[startIdx:], cx, cy, cw, ch, maxVal,
		color.RGBA{80, 220, 120, 255})

	// Total deliveries (dim blue line)
	totalBuckets := make([]int, pts)
	for i := 0; i < pts; i++ {
		idx := startIdx + i
		totalBuckets[i] = st.CorrectBuckets[idx]
		if idx < nWrong {
			totalBuckets[i] += st.WrongBuckets[idx]
		}
	}
	drawChartLine(screen, totalBuckets, cx, cy, cw, ch, maxVal,
		color.RGBA{80, 140, 220, 180})

	// Wrong deliveries (red, if any)
	if nWrong > startIdx {
		drawChartLine(screen, st.WrongBuckets[startIdx:], cx, cy, cw, ch, maxVal,
			color.RGBA{255, 80, 80, 200})
	}

	// Legend
	ly := py + chartH - 12
	vector.DrawFilledRect(screen, float32(cx), float32(ly), 8, 8, color.RGBA{80, 220, 120, 255}, false)
	printColoredAt(screen, locale.T("stat.correct"), cx+12, ly, color.RGBA{160, 200, 160, 255})
	vector.DrawFilledRect(screen, float32(cx+70), float32(ly), 8, 8, color.RGBA{255, 80, 80, 200}, false)
	printColoredAt(screen, locale.T("stat.wrong"), cx+82, ly, color.RGBA{200, 160, 160, 255})
	vector.DrawFilledRect(screen, float32(cx+135), float32(ly), 8, 8, color.RGBA{80, 140, 220, 180}, false)
	printColoredAt(screen, locale.T("stat.total"), cx+147, ly, color.RGBA{160, 170, 200, 255})

	// Current values
	if nCorrect > 0 {
		lastC := st.CorrectBuckets[nCorrect-1]
		lastW := 0
		if nWrong > 0 {
			lastW = st.WrongBuckets[nWrong-1]
		}
		info := locale.Tf("stat.current_per_bucket", lastC, lastC+lastW)
		printColoredAt(screen, info, cx+10, py+12, color.RGBA{180, 190, 210, 200})
	}
}

func drawChartLine(screen *ebiten.Image, data []int, cx, cy, cw, ch, maxVal int, col color.RGBA) {
	n := len(data)
	if n < 2 {
		return
	}
	for i := 1; i < n; i++ {
		x0 := float32(cx) + float32(cw)*float32(i-1)/float32(n-1)
		x1 := float32(cx) + float32(cw)*float32(i)/float32(n-1)
		y0 := float32(cy+ch) - float32(ch)*float32(data[i-1])/float32(maxVal)
		y1 := float32(cy+ch) - float32(ch)*float32(data[i])/float32(maxVal)
		vector.StrokeLine(screen, x0, y0, x1, y1, 2, col, false)
	}
}
