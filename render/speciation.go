package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawSpeciationOverlay draws the species composition chart and species info.
func DrawSpeciationOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	spec := ss.Speciation
	if spec == nil {
		return
	}

	x := 420
	y := 120
	w := 300
	h := 200

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h),
		color.RGBA{10, 10, 20, 230}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h),
		1, color.RGBA{80, 80, 120, 255}, false)

	// Title
	printColoredAt(screen, "ARTBILDUNG (Speciation)", x+5, y+5, color.RGBA{255, 220, 100, 255})
	printColoredAt(screen, fmt.Sprintf("Arten: %d  Schwelle: %.2f", len(spec.Species), spec.Threshold),
		x+5, y+18, color.RGBA{180, 180, 200, 255})

	// Species list (top section)
	ly := y + 34
	for i, sp := range spec.Species {
		if i >= 8 {
			printColoredAt(screen, fmt.Sprintf("... +%d weitere", len(spec.Species)-8), x+5, ly, color.RGBA{120, 120, 140, 200})
			break
		}
		col := color.RGBA{sp.Color[0], sp.Color[1], sp.Color[2], 255}
		vector.DrawFilledRect(screen, float32(x+5), float32(ly+2), 8, 8, col, false)
		info := fmt.Sprintf("Art#%d: %d Bots  Fit:%.0f  Alter:%d", sp.ID, len(sp.Members), sp.BestFitness, sp.Age)
		printColoredAt(screen, info, x+16, ly, color.RGBA{200, 200, 220, 255})
		ly += 13
	}

	// Species history chart (stacked area at bottom)
	chartY := y + h - 60
	chartW := w - 10
	chartH := 50

	vector.DrawFilledRect(screen, float32(x+5), float32(chartY), float32(chartW), float32(chartH),
		color.RGBA{5, 5, 10, 200}, false)

	histLen := len(spec.History)
	if histLen < 2 {
		printColoredAt(screen, "Warte auf Generationsdaten...", x+10, chartY+20, color.RGBA{100, 100, 120, 200})
		return
	}

	// Determine max total for normalization
	maxTotal := 1
	for _, snap := range spec.History {
		total := 0
		for _, sc := range snap.Counts {
			total += sc.Count
		}
		if total > maxTotal {
			maxTotal = total
		}
	}

	// Draw stacked bars for each generation
	startIdx := 0
	if histLen > chartW {
		startIdx = histLen - chartW
	}
	visible := spec.History[startIdx:]
	barW := float32(chartW) / float32(len(visible))
	if barW < 1 {
		barW = 1
	}

	for gi, snap := range visible {
		bx := float32(x+5) + float32(gi)*barW
		cumY := float32(0)
		for _, sc := range snap.Counts {
			frac := float32(sc.Count) / float32(maxTotal)
			segH := frac * float32(chartH)
			col := color.RGBA{sc.Color[0], sc.Color[1], sc.Color[2], 180}
			vector.DrawFilledRect(screen, bx, float32(chartY)+float32(chartH)-cumY-segH, barW, segH, col, false)
			cumY += segH
		}
	}

	// Chart label
	printColoredAt(screen, "Artenzusammensetzung", x+5, chartY-10, color.RGBA{150, 150, 170, 200})
}
