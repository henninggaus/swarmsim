package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	colorMathInput  = color.RGBA{100, 160, 255, 255} // blue
	colorMathInterm = color.RGBA{255, 220, 80, 255}  // yellow
	colorMathOutput = color.RGBA{80, 255, 120, 255}  // green
	colorMathBranch = color.RGBA{255, 160, 60, 255}  // orange
	colorMathHeader = color.RGBA{255, 200, 100, 255} // gold
	colorMathBg     = ColorPanelBg
	colorMathBorder = ColorPanelBorder
)

// mathMaxSteps limits the visible step count to avoid oversized panels.
const mathMaxSteps = 10

func mathStepColor(kind swarm.MathStepKind) color.RGBA {
	switch kind {
	case swarm.MathInput:
		return colorMathInput
	case swarm.MathIntermediate:
		return colorMathInterm
	case swarm.MathOutput:
		return colorMathOutput
	case swarm.MathBranch:
		return colorMathBranch
	}
	return colorMathInput
}

// DrawMathOverlay renders the live math trace panel.
func DrawMathOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	if !ss.ShowMathTrace {
		return
	}

	// Show hint when no trace data is available
	if ss.MathTrace == nil || len(ss.MathTrace.Steps) == 0 {
		// Position-aware: shift right of algo-labor panel when active
		x := 780
		if ss.AlgoLaborMode {
			x = editorPanelW + 10
		}
		y := 60
		w := 300
		h := 50
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), colorMathBg, false)
		vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, colorMathBorder, false)
		printColoredAt(screen, locale.T("math.no_trace"), x+8, y+8, colorMathHeader)
		printColoredAt(screen, locale.T("math.hint_select"), x+8, y+24, color.RGBA{140, 150, 170, 200})
		return
	}

	trace := ss.MathTrace

	// Panel position: shift right of the left panel in Algo-Labor mode
	x := 780
	if ss.AlgoLaborMode {
		x = editorPanelW + 10
	}
	y := 60
	w := 480

	// Limit visible steps
	visibleSteps := trace.Steps
	truncated := false
	if len(visibleSteps) > mathMaxSteps {
		visibleSteps = visibleSteps[:mathMaxSteps]
		truncated = true
	}

	// Height: header + phase + separator + steps + legend + optional truncation notice
	h := (len(visibleSteps)*2+4)*lineH + 16 + 24 // extra space for legend
	if truncated {
		h += lineH
	}

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), colorMathBg, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, colorMathBorder, false)

	lx := x + 8
	ly := y + 8

	// Header
	header := fmt.Sprintf("%s  %s", trace.AlgoName, locale.T("math.live_calc"))
	printColoredAt(screen, header, lx, ly, colorMathHeader)
	ly += lineH + 2

	// Phase sub-header
	if trace.PhaseName != "" {
		printColoredAt(screen, trace.PhaseName, lx, ly, colorMathBranch)
		ly += lineH
	}

	// Separator
	vector.StrokeLine(screen, float32(lx), float32(ly), float32(x+w-8), float32(ly), 1, colorMathBorder, false)
	ly += 4

	// Steps
	for _, step := range visibleSteps {
		col := mathStepColor(step.Kind)

		// Format value for display
		valueStr := fmt.Sprintf("%.4f", step.Value)
		if step.Value == float64(int(step.Value)) {
			valueStr = fmt.Sprintf("%.0f", step.Value)
		} else if step.Value > 100 || step.Value < -100 {
			valueStr = fmt.Sprintf("%.1f", step.Value)
		}

		// Line 1: label = symbolic
		labelStr := fmt.Sprintf("%-12s", step.Label)
		printColoredAt(screen, labelStr, lx, ly, col)
		printColoredAt(screen, "= "+step.Symbolic, lx+78, ly, color.RGBA{160, 165, 180, 200})
		ly += lineH

		// Line 2: substituted = value (indented)
		printColoredAt(screen, "= "+step.Substituted, lx+14, ly, color.RGBA{180, 185, 200, 220})
		printColoredAt(screen, "= "+valueStr, lx+300, ly, col)
		ly += lineH + 2
	}

	// Truncation notice
	if truncated {
		printColoredAt(screen, fmt.Sprintf("... +%d more steps", len(trace.Steps)-mathMaxSteps), lx, ly, color.RGBA{120, 120, 140, 180})
		ly += lineH
	}

	// Legend
	ly += 6
	vector.StrokeLine(screen, float32(lx), float32(ly), float32(x+w-8), float32(ly), 1, colorMathBorder, false)
	ly += 4
	legendItems := []struct {
		col   color.RGBA
		label string
	}{
		{colorMathInput, locale.T("math.legend.input")},
		{colorMathInterm, locale.T("math.legend.intermediate")},
		{colorMathOutput, locale.T("math.legend.output")},
		{colorMathBranch, locale.T("math.legend.branch")},
	}
	for i, item := range legendItems {
		lx2 := lx + i*110
		vector.DrawFilledCircle(screen, float32(lx2), float32(ly+5), 3, item.col, false)
		printColoredAt(screen, item.label, lx2+8, ly, color.RGBA{160, 165, 180, 200})
	}

	// ESC hint at bottom
	escHint := locale.T("overlay.esc_close")
	printColoredAt(screen, escHint, x+w/2-runeLen(escHint)*charW/2, y+h-14, color.RGBA{120, 130, 150, 180})
}
