package render

import (
	"fmt"
	"image/color"
	"sort"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawTournamentOverlay renders the tournament status and results.
func DrawTournamentOverlay(screen *ebiten.Image, ss *swarm.SwarmState) {
	sw := screen.Bounds().Dx()

	panelW := 400
	panelH := 300
	panelX := (sw - panelW) / 2
	panelY := 60

	headerCol := color.RGBA{255, 215, 80, 255}   // gold
	valCol := color.RGBA{200, 200, 220, 255}
	dimCol := color.RGBA{120, 120, 140, 255}
	greenCol := color.RGBA{80, 255, 120, 255}
	redCol := color.RGBA{255, 100, 100, 255}
	activeCol := color.RGBA{100, 200, 255, 255}

	// Background
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), color.RGBA{10, 10, 25, 240}, false)
	vector.StrokeRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), 2, color.RGBA{255, 200, 50, 200}, false)
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY), float32(panelW), 2,
		color.RGBA{255, 215, 80, 120}, false)

	cx := panelX + 10
	cy := panelY + 10

	switch ss.TournamentPhase {
	case 0: // Idle — show roster
		printColoredAt(screen, "TURNIER-MODUS", cx, cy, headerCol)
		cy += 16
		printColoredAt(screen, "U=Programm hinzufuegen | Enter=Start | Esc=Beenden", cx, cy, dimCol)
		cy += 18

		if len(ss.TournamentEntries) == 0 {
			printColoredAt(screen, "Keine Programme. Lade ein Preset und druecke U.", cx, cy, dimCol)
		} else {
			printColoredAt(screen, fmt.Sprintf("Programme (%d):", len(ss.TournamentEntries)), cx, cy, valCol)
			cy += 14
			for i, e := range ss.TournamentEntries {
				marker := fmt.Sprintf("  %d. %s", i+1, e.Name)
				printColoredAt(screen, marker, cx, cy, activeCol)
				cy += 14
				if cy > panelY+panelH-30 {
					printColoredAt(screen, "  ...", cx, cy, dimCol)
					break
				}
			}
		}

		if len(ss.TournamentEntries) >= 2 {
			cy = panelY + panelH - 20
			printColoredAt(screen, "Enter = Turnier starten!", cx, cy, greenCol)
		}

	case 1: // Running
		entry := &ss.TournamentEntries[ss.TournamentRound]
		printColoredAt(screen, "TURNIER LAEUFT", cx, cy, headerCol)
		cy += 16

		roundInfo := fmt.Sprintf("Runde %d/%d: %s", ss.TournamentRound+1,
			len(ss.TournamentEntries), entry.Name)
		printColoredAt(screen, roundInfo, cx, cy, activeCol)
		cy += 14

		timerInfo := fmt.Sprintf("Verbleibend: %d Ticks", ss.TournamentTimer)
		printColoredAt(screen, timerInfo, cx, cy, valCol)
		cy += 14

		// Progress bar
		barW := float32(panelW - 20)
		barH := float32(8)
		barX := float32(cx)
		barY := float32(cy)
		vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{40, 40, 60, 255}, false)
		progress := 1.0 - float32(ss.TournamentTimer)/float32(swarm.TournamentRoundTicks)
		vector.DrawFilledRect(screen, barX, barY, barW*progress, barH, color.RGBA{255, 200, 50, 255}, false)
		cy += 16

		// Current stats
		ds := &ss.DeliveryStats
		statsInfo := fmt.Sprintf("Deliveries: %d | Correct: %d | Wrong: %d",
			ds.TotalDelivered, ds.CorrectDelivered, ds.WrongDelivered)
		printColoredAt(screen, statsInfo, cx, cy, valCol)
		cy += 18

		// Previous rounds
		if ss.TournamentRound > 0 {
			printColoredAt(screen, "Bisherige Ergebnisse:", cx, cy, dimCol)
			cy += 14
			for i := 0; i < ss.TournamentRound && i < len(ss.TournamentResults); i++ {
				r := &ss.TournamentResults[i]
				line := fmt.Sprintf("  %s: Score %d (C:%d W:%d)", r.Name, r.Score, r.Correct, r.Wrong)
				scoreCol := valCol
				if r.Score > 0 {
					scoreCol = greenCol
				}
				printColoredAt(screen, line, cx, cy, scoreCol)
				cy += 14
				if cy > panelY+panelH-20 {
					break
				}
			}
		}

	case 2: // Results
		printColoredAt(screen, "TURNIER-ERGEBNISSE", cx, cy, headerCol)
		cy += 18

		// Sort by score descending
		sorted := make([]swarm.TournamentResult, len(ss.TournamentResults))
		copy(sorted, ss.TournamentResults)
		sort.Slice(sorted, func(a, b int) bool {
			return sorted[a].Score > sorted[b].Score
		})

		// Column headers
		printColoredAt(screen, "#", cx, cy, headerCol)
		printColoredAt(screen, "Programm", cx+20, cy, headerCol)
		printColoredAt(screen, "Score", cx+200, cy, headerCol)
		printColoredAt(screen, "Correct", cx+260, cy, headerCol)
		printColoredAt(screen, "Wrong", cx+320, cy, headerCol)
		cy += 14
		vector.StrokeLine(screen, float32(cx), float32(cy), float32(panelX+panelW-10), float32(cy),
			1, color.RGBA{60, 80, 120, 150}, false)
		cy += 4

		for i, r := range sorted {
			rankCol := valCol
			if i == 0 {
				rankCol = color.RGBA{255, 215, 80, 255} // gold
			} else if i == 1 {
				rankCol = color.RGBA{200, 200, 200, 255} // silver
			} else if i == 2 {
				rankCol = color.RGBA{200, 150, 80, 255} // bronze
			}
			printColoredAt(screen, fmt.Sprintf("%d", i+1), cx, cy, rankCol)
			printColoredAt(screen, r.Name, cx+20, cy, rankCol)
			scoreCol := greenCol
			if r.Score <= 0 {
				scoreCol = redCol
			}
			printColoredAt(screen, fmt.Sprintf("%d", r.Score), cx+200, cy, scoreCol)
			printColoredAt(screen, fmt.Sprintf("%d", r.Correct), cx+260, cy, valCol)
			printColoredAt(screen, fmt.Sprintf("%d", r.Wrong), cx+320, cy, valCol)
			cy += 16
			if cy > panelY+panelH-30 {
				break
			}
		}

		cy = panelY + panelH - 20
		printColoredAt(screen, "U=Neues Turnier | Esc=Beenden", cx, cy, dimCol)
	}
}
