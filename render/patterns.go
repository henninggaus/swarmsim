package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawPatternOverlay draws the emergent pattern detection panel.
func DrawPatternOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	pr := ss.PatternResult
	if pr == nil {
		return
	}

	x := 420
	y := 120
	w := 280
	h := 170

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h),
		color.RGBA{10, 10, 20, 230}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h),
		1, color.RGBA{80, 80, 120, 255}, false)

	// Title
	printColoredAt(screen, locale.T("patterns.title"), x+5, y+5, color.RGBA{255, 220, 100, 255})

	// Primary pattern with large indicator
	patternCol := patternColor(pr.Primary)
	ly := y + 22
	vector.DrawFilledRect(screen, float32(x+5), float32(ly), float32(int(float64(w-10)*pr.PrimaryScore)), 12,
		patternCol, false)
	vector.StrokeRect(screen, float32(x+5), float32(ly), float32(w-10), 12,
		1, color.RGBA{80, 80, 100, 200}, false)
	label := fmt.Sprintf("%s (%.0f%%)", swarm.PatternName(pr.Primary), pr.PrimaryScore*100)
	printColoredAt(screen, label, x+10, ly+1, ColorWhite)

	// Secondary pattern
	ly += 18
	if pr.SecondScore > 0.1 {
		label2 := fmt.Sprintf("  + %s (%.0f%%)", swarm.PatternName(pr.Secondary), pr.SecondScore*100)
		printColoredAt(screen, label2, x+5, ly, color.RGBA{180, 180, 200, 200})
	}
	ly += 16

	// Metrics
	drawMetricBar(screen, x+5, ly, w-10, locale.T("patterns.alignment"), pr.Alignment, color.RGBA{100, 200, 255, 200})
	ly += 16
	drawMetricBar(screen, x+5, ly, w-10, locale.T("patterns.cohesion"), pr.Cohesion, color.RGBA{100, 255, 100, 200})
	ly += 16
	drawMetricBar(screen, x+5, ly, w-10, locale.T("patterns.circularity"), pr.Circularity, color.RGBA{255, 200, 100, 200})
	ly += 16
	drawMetricBar(screen, x+5, ly, w-10, locale.T("patterns.entropy"), pr.Entropy, color.RGBA{255, 100, 100, 200})
	ly += 16

	// Cluster count
	clusterInfo := locale.Tf("patterns.cluster_count", pr.ClusterCount)
	printColoredAt(screen, clusterInfo, x+5, ly, color.RGBA{180, 180, 200, 255})
}

// drawMetricBar draws a labeled metric bar.
func drawMetricBar(screen *ebiten.Image, x, y, w int, label string, value float64, col color.RGBA) {
	labelW := 90
	barW := w - labelW - 5
	printColoredAt(screen, label, x, y, color.RGBA{150, 150, 170, 200})
	// Bar background
	vector.DrawFilledRect(screen, float32(x+labelW), float32(y+2), float32(barW), 8,
		color.RGBA{30, 30, 40, 200}, false)
	// Bar fill
	fill := int(math.Max(0, math.Min(1, value)) * float64(barW))
	vector.DrawFilledRect(screen, float32(x+labelW), float32(y+2), float32(fill), 8, col, false)
}

// patternColor returns a display color for a pattern type.
func patternColor(p swarm.PatternType) color.RGBA {
	switch p {
	case swarm.PatternCluster:
		return color.RGBA{80, 200, 80, 200}
	case swarm.PatternLine:
		return color.RGBA{200, 200, 80, 200}
	case swarm.PatternCircle:
		return color.RGBA{80, 200, 255, 200}
	case swarm.PatternScattered:
		return color.RGBA{200, 80, 80, 200}
	case swarm.PatternStream:
		return color.RGBA{100, 100, 255, 200}
	case swarm.PatternVortex:
		return color.RGBA{200, 80, 200, 200}
	}
	return color.RGBA{150, 150, 150, 200}
}
