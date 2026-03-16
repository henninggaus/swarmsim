package render

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawGenomeBrowser renders the Genom-Browser overlay — a sortable list
// of all bots with fitness, age, genome summary. Works with Evolution, GP, and Neuro modes.
func DrawGenomeBrowser(screen *ebiten.Image, ss *swarm.SwarmState) {
	n := len(ss.Bots)
	if n == 0 {
		return
	}

	panelW := 500
	panelH := 550
	sh := screen.Bounds().Dy()
	sw := screen.Bounds().Dx()
	panelX := (sw - panelW) / 2
	panelY := (sh - panelH) / 2
	if panelY < 30 {
		panelY = 30
	}

	// Background
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), color.RGBA{10, 10, 20, 240}, false)
	vector.StrokeRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), 2, color.RGBA{100, 180, 255, 200}, false)
	// Top highlight
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY), float32(panelW), 2,
		color.RGBA{100, 180, 255, 120}, false)

	headerCol := color.RGBA{136, 204, 255, 255}
	dimCol := color.RGBA{120, 120, 140, 255}
	valCol := color.RGBA{200, 200, 220, 255}
	goldCol := color.RGBA{255, 215, 80, 255}
	greenCol := color.RGBA{80, 255, 120, 255}
	redCol := color.RGBA{255, 100, 100, 255}

	cx := panelX + 8
	cy := panelY + 8

	// Title
	modeStr := "Evolution"
	if ss.GPEnabled {
		modeStr = "GP"
	} else if ss.NeuroEnabled {
		modeStr = "Neuro"
	}
	genNum := ss.Generation
	if ss.GPEnabled {
		genNum = ss.GPGeneration
	} else if ss.NeuroEnabled {
		genNum = ss.NeuroGeneration
	}
	title := fmt.Sprintf("GENOM-BROWSER (%s Gen %d)", modeStr, genNum)
	printColoredAt(screen, title, cx, cy, headerCol)
	cy += 16

	// Sort mode indicator
	sortNames := []string{"Fitness", "Alter (Ticks)", "Deliveries"}
	sortInfo := fmt.Sprintf("Sortierung: %s  (Tab=wechseln, Mausrad=scrollen)", sortNames[ss.GenomeBrowserSort])
	printColoredAt(screen, sortInfo, cx, cy, dimCol)
	cy += 14

	// Column headers
	headerY := cy
	printColoredAt(screen, "#", cx, headerY, goldCol)
	printColoredAt(screen, "Bot", cx+20, headerY, goldCol)
	printColoredAt(screen, "Fitness", cx+60, headerY, goldCol)
	printColoredAt(screen, "Age", cx+120, headerY, goldCol)
	printColoredAt(screen, "Deliv", cx+170, headerY, goldCol)
	printColoredAt(screen, "Genom-Info", cx+220, headerY, goldCol)
	cy += 14

	// Separator
	vector.StrokeLine(screen, float32(cx), float32(cy), float32(panelX+panelW-8), float32(cy),
		1, color.RGBA{60, 80, 120, 150}, false)
	cy += 4

	// Sort bots by selected criterion
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	switch ss.GenomeBrowserSort {
	case 0: // Fitness (descending)
		sort.Slice(indices, func(a, b int) bool {
			return ss.Bots[indices[a]].Fitness > ss.Bots[indices[b]].Fitness
		})
	case 1: // Age / TicksAlive (descending)
		sort.Slice(indices, func(a, b int) bool {
			return ss.Bots[indices[a]].Stats.TicksAlive > ss.Bots[indices[b]].Stats.TicksAlive
		})
	case 2: // Deliveries (descending)
		sort.Slice(indices, func(a, b int) bool {
			return ss.Bots[indices[a]].Stats.TotalDeliveries > ss.Bots[indices[b]].Stats.TotalDeliveries
		})
	}

	// Clamp scroll
	listAreaH := panelY + panelH - cy - 30
	rowH := 16
	maxVisible := listAreaH / rowH
	if maxVisible < 1 {
		maxVisible = 1
	}
	maxScroll := n - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ss.GenomeBrowserScroll > maxScroll {
		ss.GenomeBrowserScroll = maxScroll
	}
	if ss.GenomeBrowserScroll < 0 {
		ss.GenomeBrowserScroll = 0
	}

	// Render visible rows
	listStartY := cy
	for row := 0; row < maxVisible && row+ss.GenomeBrowserScroll < n; row++ {
		rank := row + ss.GenomeBrowserScroll
		botIdx := indices[rank]
		bot := &ss.Bots[botIdx]
		ry := listStartY + row*rowH

		// Alternate row background
		if row%2 == 0 {
			vector.DrawFilledRect(screen, float32(cx-2), float32(ry-1),
				float32(panelW-16), float32(rowH), color.RGBA{25, 25, 40, 120}, false)
		}

		// Highlight selected bot
		if botIdx == ss.SelectedBot {
			vector.DrawFilledRect(screen, float32(cx-2), float32(ry-1),
				float32(panelW-16), float32(rowH), color.RGBA{0, 80, 160, 80}, false)
		}

		// Rank
		rankCol := valCol
		if rank == 0 {
			rankCol = goldCol
		} else if rank == 1 {
			rankCol = color.RGBA{200, 200, 200, 255}
		} else if rank == 2 {
			rankCol = color.RGBA{200, 150, 80, 255}
		}
		printColoredAt(screen, fmt.Sprintf("%d", rank+1), cx, ry, rankCol)

		// Bot index + LED swatch
		printColoredAt(screen, fmt.Sprintf("#%d", botIdx), cx+20, ry, valCol)
		ledCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 255}
		vector.DrawFilledRect(screen, float32(cx+52), float32(ry+2), 6, 6, ledCol, false)

		// Fitness
		fitCol := valCol
		if rank < n/5 {
			fitCol = greenCol // top 20%
		} else if rank > n*4/5 {
			fitCol = redCol // bottom 20%
		}
		printColoredAt(screen, fmt.Sprintf("%.0f", bot.Fitness), cx+60, ry, fitCol)

		// Age
		printColoredAt(screen, fmt.Sprintf("%dk", bot.Stats.TicksAlive/1000), cx+120, ry, dimCol)

		// Deliveries
		delivStr := fmt.Sprintf("%d/%d", bot.Stats.CorrectDeliveries, bot.Stats.TotalDeliveries)
		printColoredAt(screen, delivStr, cx+170, ry, valCol)

		// Genome summary (mode-specific)
		genomeStr := genomeSummary(ss, botIdx)
		printColoredAt(screen, genomeStr, cx+220, ry, dimCol)
	}

	// Scrollbar
	if n > maxVisible {
		sbX := float32(panelX + panelW - 8)
		sbH := float32(listAreaH)
		sbY := float32(listStartY)
		// Track
		vector.DrawFilledRect(screen, sbX, sbY, 4, sbH, color.RGBA{40, 40, 60, 150}, false)
		// Thumb
		thumbH := sbH * float32(maxVisible) / float32(n)
		if thumbH < 10 {
			thumbH = 10
		}
		thumbY := sbY + (sbH-thumbH)*float32(ss.GenomeBrowserScroll)/float32(maxScroll)
		vector.DrawFilledRect(screen, sbX, thumbY, 4, thumbH, color.RGBA{100, 150, 255, 200}, false)
	}

	// Footer
	footerY := panelY + panelH - 16
	footerStr := fmt.Sprintf("%d Bots | G=Schliessen | Tab=Sortierung", n)
	printColoredAt(screen, footerStr, cx, footerY, dimCol)
}

// genomeSummary returns a short genome description for the given bot.
func genomeSummary(ss *swarm.SwarmState, idx int) string {
	bot := &ss.Bots[idx]

	if ss.NeuroEnabled && bot.Brain != nil {
		// Show dominant action and avg weight magnitude
		action := swarm.NeuroActionNames[bot.Brain.ActionIdx]
		avgW := 0.0
		for _, w := range bot.Brain.Weights {
			avgW += math.Abs(w)
		}
		avgW /= float64(swarm.NeuroWeights)
		return fmt.Sprintf("%s |w|=%.2f", action, avgW)
	}

	if ss.GPEnabled && bot.OwnProgram != nil {
		// Show rule count and matched rules
		nRules := len(bot.OwnProgram.Rules)
		matched := len(bot.LastMatchedRules)
		return fmt.Sprintf("%dR %dM", nRules, matched)
	}

	if ss.EvolutionOn {
		// Show used param values summary
		parts := ""
		count := 0
		for p := 0; p < 26; p++ {
			if !ss.UsedParams[p] {
				continue
			}
			if count > 0 {
				parts += " "
			}
			parts += fmt.Sprintf("%c=%.0f", 'A'+p, bot.ParamValues[p])
			count++
			if count >= 4 {
				break // max 4 params shown
			}
		}
		return parts
	}

	return ""
}
