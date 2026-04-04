package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawParetoOverlay renders the Pareto front visualization panel.
func DrawParetoOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	pf := ss.ParetoFront
	if pf == nil || len(pf.Fronts) == 0 {
		return
	}

	panelW := 220
	panelH := 150
	px := screen.Bounds().Dx() - panelW - 10
	py := 270 // below formation panel area

	// Background
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(panelW), float32(panelH), color.RGBA{10, 15, 35, 220}, false)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(panelW), float32(panelH), 1, color.RGBA{200, 100, 255, 180}, false)

	// Title
	printColoredAt(screen, locale.T("pareto.title"), px+6, py+4, color.RGBA{200, 130, 255, 255})

	y := py + 22
	objNames := swarm.ObjectiveNames()

	// Front stats
	printColoredAt(screen, locale.Tf("pareto.front_size", swarm.ParetoFrontSize(pf)),
		px+8, y, ColorMediumGray)
	y += lineH + 2

	printColoredAt(screen, locale.Tf("pareto.fronts_count", len(pf.Fronts)),
		px+8, y, ColorMediumGray)
	y += lineH + 4

	// Objective ranges for Pareto front (rank 0)
	if len(pf.Fronts) > 0 && len(pf.Fronts[0]) > 0 && len(pf.Objectives) > 0 {
		printColoredAt(screen, locale.T("pareto.objectives"), px+8, y, color.RGBA{140, 150, 170, 255})
		y += lineH + 2

		numObj := len(pf.Objectives[0])
		for obj := 0; obj < numObj && obj < len(objNames); obj++ {
			minV := pf.Objectives[pf.Fronts[0][0]][obj]
			maxV := minV
			for _, idx := range pf.Fronts[0] {
				v := pf.Objectives[idx][obj]
				if v < minV {
					minV = v
				}
				if v > maxV {
					maxV = v
				}
			}

			label := fmt.Sprintf("%s: %.0f-%.0f", objNames[obj], minV, maxV)
			col := ColorMediumGray
			switch obj {
			case 0:
				col = color.RGBA{100, 255, 150, 255} // green for deliveries
			case 1:
				col = color.RGBA{100, 200, 255, 255} // blue for exploration
			case 2:
				col = color.RGBA{255, 200, 100, 255} // yellow for efficiency
			}
			printColoredAt(screen, label, px+12, y, col)
			y += lineH
		}
	}

	// Mini scatter plot: objective 0 vs objective 1
	plotX := float32(px + 8)
	plotY := float32(py + panelH - 40)
	plotW := float32(panelW - 16)
	plotH := float32(30)

	// Plot background
	vector.DrawFilledRect(screen, plotX, plotY, plotW, plotH, color.RGBA{20, 20, 35, 200}, false)

	if len(pf.Objectives) > 0 && len(pf.Objectives[0]) >= 2 {
		// Find ranges
		minO0, maxO0 := pf.Objectives[0][0], pf.Objectives[0][0]
		minO1, maxO1 := pf.Objectives[0][1], pf.Objectives[0][1]
		for _, objs := range pf.Objectives {
			if objs[0] < minO0 {
				minO0 = objs[0]
			}
			if objs[0] > maxO0 {
				maxO0 = objs[0]
			}
			if objs[1] < minO1 {
				minO1 = objs[1]
			}
			if objs[1] > maxO1 {
				maxO1 = objs[1]
			}
		}
		rangeO0 := maxO0 - minO0
		rangeO1 := maxO1 - minO1
		if rangeO0 < 0.001 {
			rangeO0 = 1
		}
		if rangeO1 < 0.001 {
			rangeO1 = 1
		}

		// Draw all bots as dots (dim)
		front0 := pf.Fronts[0] // safe: pf.Fronts checked at top
		for i, objs := range pf.Objectives {
			x := plotX + float32((objs[0]-minO0)/rangeO0)*plotW
			y := plotY + plotH - float32((objs[1]-minO1)/rangeO1)*plotH
			col := color.RGBA{60, 60, 80, 150}
			// Pareto front bots are bright
			for _, idx := range front0 {
				if idx == i {
					col = color.RGBA{200, 130, 255, 255}
					break
				}
			}
			vector.DrawFilledCircle(screen, x, y, 2, col, false)
		}
	}
}
