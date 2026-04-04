package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/hajimehoshi/ebiten/v2"
)

// drawNeuroVisualization renders the neural network diagram in the editor code area.
// Shows: explanation header, network architecture, sensor→action mapping,
// and real-time activations of the selected bot (or bot 0).
func drawNeuroVisualization(screen *ebiten.Image, ss *swarm.SwarmState) {
	// Background
	vector.DrawFilledRect(screen, 0, float32(editorCodeY), float32(editorPanelW), float32(editorCodeH),
		color.RGBA{15, 15, 25, 255}, false)

	x := 5
	y := editorCodeY + 4

	// ── Header: Was ist Neuroevolution? ──
	printColoredAt(screen, locale.T("neuro.title"), x+2, y, color.RGBA{255, 140, 50, 255})
	y += lineH
	printColoredAt(screen, locale.T("neuro.desc1"), x+2, y, color.RGBA{160, 160, 180, 255})
	y += lineH
	printColoredAt(screen, locale.T("neuro.desc2"), x+2, y, color.RGBA{160, 160, 180, 255})
	y += lineH
	printColoredAt(screen, locale.T("neuro.desc3"), x+2, y, color.RGBA{100, 255, 100, 220})
	y += lineH + 4

	// ── Architecture summary ──
	printColoredAt(screen, locale.T("neuro.architecture"), x+2, y, ColorInfoCyan)
	y += lineH
	printColoredAt(screen, locale.T("neuro.arch_layout"), x+2, y, color.RGBA{180, 180, 200, 200})
	y += lineH
	printColoredAt(screen, locale.Tf("neuro.arch_weights", swarm.NeuroWeights), x+2, y, color.RGBA{140, 140, 160, 200})
	y += lineH + 6

	// ── Separator ──
	vector.StrokeLine(screen, float32(x), float32(y), float32(editorPanelW-5), float32(y), 1, color.RGBA{60, 60, 80, 200}, false)
	y += 6

	// ── Network diagram ──
	// Get reference bot (selected or first)
	if len(ss.Bots) == 0 {
		printColoredAt(screen, locale.T("neuro.no_bots"), x+2, y, color.RGBA{120, 120, 140, 200})
		return
	}
	botIdx := 0
	if ss.SelectedBot >= 0 && ss.SelectedBot < len(ss.Bots) {
		botIdx = ss.SelectedBot
	}
	bot := &ss.Bots[botIdx]
	brain := bot.Brain

	printColoredAt(screen, fmt.Sprintf("BOT #%d NETZ (live)", botIdx), x+2, y, color.RGBA{255, 200, 100, 220})
	y += lineH + 2

	if brain == nil {
		printColoredAt(screen, locale.T("neuro.no_net"), x+2, y, color.RGBA{120, 120, 140, 200})
		return
	}

	// Draw the neural network visualization
	drawNeuroNetDiagram(screen, brain, x+2, y, editorPanelW-10, 220, ss.TruckToggle)
	y += 225

	// ── Separator ──
	vector.StrokeLine(screen, float32(x), float32(y), float32(editorPanelW-5), float32(y), 1, color.RGBA{60, 60, 80, 200}, false)
	y += 6

	// ── Evolution info ──
	printColoredAt(screen, locale.T("neuro.evolution"), x+2, y, ColorInfoCyan)
	y += lineH
	printColoredAt(screen, locale.T("neuro.evo1"), x+2, y, color.RGBA{140, 140, 160, 200})
	y += lineH
	printColoredAt(screen, locale.T("neuro.evo2"), x+2, y, color.RGBA{140, 140, 160, 200})
	y += lineH
	printColoredAt(screen, locale.T("neuro.evo3"), x+2, y, color.RGBA{140, 140, 160, 200})
	y += lineH
	printColoredAt(screen, locale.T("neuro.evo4"), x+2, y, color.RGBA{140, 140, 160, 200})
	y += lineH + 2

	// Fitness function explanation (context-sensitive)
	printColoredAt(screen, locale.T("neuro.fitness_formula"), x+2, y, color.RGBA{200, 160, 255, 220})
	y += lineH
	if ss.TruckToggle {
		printColoredAt(screen, locale.T("neuro.fit_truck1"), x+2, y, color.RGBA{80, 255, 80, 200})
		y += lineH
		printColoredAt(screen, locale.T("neuro.fit_truck2"), x+2, y, color.RGBA{80, 255, 80, 200})
		y += lineH
		printColoredAt(screen, locale.T("neuro.fit_truck3"), x+2, y, color.RGBA{255, 100, 80, 200})
	} else {
		printColoredAt(screen, locale.T("neuro.fit_deliv1"), x+2, y, color.RGBA{80, 255, 80, 200})
		y += lineH
		printColoredAt(screen, locale.T("neuro.fit_deliv2"), x+2, y, color.RGBA{80, 255, 80, 200})
		y += lineH
		printColoredAt(screen, locale.T("neuro.fit_deliv3"), x+2, y, color.RGBA{255, 100, 80, 200})
	}
	y += lineH + 4

	// Current generation stats
	printColoredAt(screen, fmt.Sprintf("Generation: %d", ss.NeuroGeneration), x+2, y, color.RGBA{255, 140, 50, 240})
	y += lineH
	if ss.NeuroGeneration > 0 {
		printColoredAt(screen, fmt.Sprintf("Best Fitness: %.0f", ss.BestFitness), x+2, y, color.RGBA{80, 255, 80, 220})
		y += lineH
		printColoredAt(screen, fmt.Sprintf("Avg Fitness:  %.0f", ss.AvgFitness), x+2, y, color.RGBA{255, 200, 50, 220})
	} else {
		printColoredAt(screen, locale.T("neuro.waiting_gen"), x+2, y, color.RGBA{120, 120, 140, 180})
		y += lineH
		progress := float64(ss.NeuroTimer) / 2000.0 * 100.0
		printColoredAt(screen, locale.Tf("neuro.progress", progress), x+2, y, color.RGBA{255, 140, 50, 180})
	}
}

// drawNeuroNetDiagram draws the actual neural network nodes and connections.
func drawNeuroNetDiagram(screen *ebiten.Image, brain *swarm.NeuroBrain, gx, gy, gw, gh int, truckMode ...bool) {
	isTruck := len(truckMode) > 0 && truckMode[0]
	// Background
	vector.DrawFilledRect(screen, float32(gx-2), float32(gy-2), float32(gw+4), float32(gh+4),
		color.RGBA{5, 5, 15, 220}, false)

	// Layout: 3 columns (input | hidden | output)
	colInput := float32(gx) + 55
	colHidden := float32(gx) + float32(gw)/2
	colOutput := float32(gx) + float32(gw) - 55

	inputSpacing := float32(gh) / float32(swarm.NeuroInputs+1)
	hiddenSpacing := float32(gh) / float32(swarm.NeuroHidden+1)
	outputSpacing := float32(gh) / float32(swarm.NeuroOutputs+1)

	// Column headers
	printColoredAt(screen, locale.T("neuro.col_sensor"), gx+2, gy, color.RGBA{100, 200, 255, 200})
	printColoredAt(screen, locale.T("neuro.col_hidden"), int(colHidden)-15, gy, color.RGBA{200, 200, 100, 200})
	printColoredAt(screen, locale.T("neuro.col_action"), int(colOutput)-10, gy, color.RGBA{255, 180, 100, 200})

	// Compute node positions
	type nodePos struct {
		x, y float32
	}
	inputNodes := make([]nodePos, swarm.NeuroInputs)
	hiddenNodes := make([]nodePos, swarm.NeuroHidden)
	outputNodes := make([]nodePos, swarm.NeuroOutputs)

	for i := 0; i < swarm.NeuroInputs; i++ {
		inputNodes[i] = nodePos{colInput, float32(gy) + float32(i+1)*inputSpacing}
	}
	for i := 0; i < swarm.NeuroHidden; i++ {
		hiddenNodes[i] = nodePos{colHidden, float32(gy) + float32(i+1)*hiddenSpacing}
	}
	for i := 0; i < swarm.NeuroOutputs; i++ {
		outputNodes[i] = nodePos{colOutput, float32(gy) + float32(i+1)*outputSpacing}
	}

	// Draw connections: input → hidden (color by weight)
	for inp := 0; inp < swarm.NeuroInputs; inp++ {
		for h := 0; h < swarm.NeuroHidden; h++ {
			w := brain.Weights[inp*swarm.NeuroHidden+h]
			drawWeightLine(screen, inputNodes[inp].x, inputNodes[inp].y,
				hiddenNodes[h].x, hiddenNodes[h].y, w)
		}
	}

	// Draw connections: hidden → output
	offset := swarm.NeuroInputs * swarm.NeuroHidden
	for h := 0; h < swarm.NeuroHidden; h++ {
		for o := 0; o < swarm.NeuroOutputs; o++ {
			w := brain.Weights[offset+h*swarm.NeuroOutputs+o]
			drawWeightLine(screen, hiddenNodes[h].x, hiddenNodes[h].y,
				outputNodes[o].x, outputNodes[o].y, w)
		}
	}

	// Draw input nodes with labels and activation values
	for i := 0; i < swarm.NeuroInputs; i++ {
		val := brain.InputVals[i]
		nodeCol := activationColor(val, false)
		r := float32(3.0)
		vector.DrawFilledCircle(screen, inputNodes[i].x, inputNodes[i].y, r, nodeCol, false)
		// Label (left of node)
		inputNames := swarm.NeuroInputNames
		if isTruck {
			inputNames = swarm.NeuroTruckInputNames
		}
		name := inputNames[i]
		if len(name) > 6 {
			name = name[:6]
		}
		printColoredAt(screen, name, gx+2, int(inputNodes[i].y)-5, color.RGBA{100, 140, 180, 200})
	}

	// Draw hidden nodes
	for i := 0; i < swarm.NeuroHidden; i++ {
		val := brain.HiddenAct[i]
		nodeCol := activationColor(val, true)
		r := float32(4.0)
		vector.DrawFilledCircle(screen, hiddenNodes[i].x, hiddenNodes[i].y, r, nodeCol, false)
	}

	// Draw output nodes with labels
	for i := 0; i < swarm.NeuroOutputs; i++ {
		val := brain.OutputAct[i]
		isWinner := i == brain.ActionIdx
		nodeCol := activationColor(val, true)
		r := float32(3.5)
		if isWinner {
			r = 5.0
			nodeCol = color.RGBA{255, 255, 100, 255} // bright yellow for chosen action
			// Highlight ring
			vector.StrokeCircle(screen, outputNodes[i].x, outputNodes[i].y, 7, 1.5, color.RGBA{255, 200, 50, 200}, false)
		}
		vector.DrawFilledCircle(screen, outputNodes[i].x, outputNodes[i].y, r, nodeCol, false)
		// Label (right of node)
		actionNames := swarm.NeuroActionNames
		if isTruck {
			actionNames = swarm.NeuroTruckActionNames
		}
		name := actionNames[i]
		labelCol := color.RGBA{140, 140, 160, 200}
		if isWinner {
			labelCol = color.RGBA{255, 255, 100, 255}
		}
		printColoredAt(screen, name, int(colOutput)+10, int(outputNodes[i].y)-5, labelCol)
	}
}

// drawWeightLine draws a connection line colored by weight value.
// Positive = green, negative = red, thin for small weights.
func drawWeightLine(screen *ebiten.Image, x1, y1, x2, y2 float32, weight float64) {
	absW := math.Abs(weight)
	if absW < 0.05 {
		return // skip near-zero connections
	}

	// Alpha based on weight magnitude
	alpha := uint8(math.Min(absW*80, 100))
	var col color.RGBA
	if weight > 0 {
		col = color.RGBA{50, 200, 50, alpha} // green = positive
	} else {
		col = color.RGBA{200, 50, 50, alpha} // red = negative
	}

	thickness := float32(math.Min(absW*1.5, 2.0))
	vector.StrokeLine(screen, x1, y1, x2, y2, thickness, col, false)
}

// activationColor maps an activation value to a node color.
// For inputs: 0=dim, 1=bright cyan. For hidden/output: negative=red, positive=green.
func activationColor(val float64, signed bool) color.RGBA {
	if !signed {
		// Input: 0-1 range
		brightness := uint8(math.Min(math.Max(val*255, 30), 255))
		return color.RGBA{0, brightness, brightness, 220}
	}
	// Signed: tanh range -1..+1
	if val >= 0 {
		g := uint8(math.Min(val*255+50, 255))
		return color.RGBA{30, g, 30, 220}
	}
	r := uint8(math.Min(-val*255+50, 255))
	return color.RGBA{r, 30, 30, 220}
}
