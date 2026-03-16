package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawFormationOverlay renders swarm formation metrics panel.
func DrawFormationOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	m := swarm.ComputeFormation(ss)

	panelW := 240
	panelH := 200
	px := screen.Bounds().Dx() - panelW - 10
	py := 60

	// Background
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(panelW), float32(panelH), color.RGBA{10, 15, 35, 220}, false)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(panelW), float32(panelH), 1, color.RGBA{80, 120, 200, 180}, false)

	// Title
	printColoredAt(screen, "FORMATION-ANALYSE", px+6, py+4, color.RGBA{100, 180, 255, 255})

	y := py + 20
	lines := []struct {
		label string
		value string
		bar   float64 // 0-1 for bar display, -1 for no bar
		col   color.RGBA
	}{
		{"Zentroid", fmt.Sprintf("(%.0f, %.0f)", m.CentroidX, m.CentroidY), -1, color.RGBA{180, 180, 180, 255}},
		{"Ausbreitung", fmt.Sprintf("%.0fpx (max %.0f)", m.SpreadRadius, m.MaxSpread), -1, color.RGBA{180, 180, 180, 255}},
		{"Cluster", fmt.Sprintf("%d", m.ClusterCount), -1, color.RGBA{255, 200, 100, 255}},
		{"Avg Speed", fmt.Sprintf("%.1f", m.AvgSpeed), -1, color.RGBA{180, 180, 180, 255}},
		{"Avg Nachbar", fmt.Sprintf("%.0fpx", m.AvgNeighDist), -1, color.RGBA{180, 180, 180, 255}},
		{"Alignment", fmt.Sprintf("%.0f%%", m.Alignment*100), m.Alignment, color.RGBA{100, 255, 150, 255}},
		{"Kohaesion", fmt.Sprintf("%.0f%%", m.Cohesion*100), m.Cohesion, color.RGBA{100, 200, 255, 255}},
	}

	for _, l := range lines {
		printColoredAt(screen, l.label+":", px+8, y, color.RGBA{140, 150, 170, 255})
		printColoredAt(screen, l.value, px+100, y, l.col)

		// Draw bar if applicable
		if l.bar >= 0 {
			barX := float32(px + 170)
			barW := float32(55)
			barH := float32(8)
			barY := float32(y + 2)
			vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{30, 30, 50, 200}, false)
			fillW := barW * float32(l.bar)
			vector.DrawFilledRect(screen, barX, barY, fillW, barH, l.col, false)
		}

		y += lineH + 2
	}

	// Draw centroid marker on arena
	// (This is rendered in screen space, need to convert)
}
