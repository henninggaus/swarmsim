package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawLeaderboardOverlay renders the highscore table.
func DrawLeaderboardOverlay(screen *ebiten.Image, lb *swarm.LeaderboardState) {
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// Semi-transparent background
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), color.RGBA{0, 0, 0, 200}, false)

	panelW := 700
	panelH := 500
	px := sw/2 - panelW/2
	py := sh/2 - panelH/2

	// Panel background
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(panelW), float32(panelH), color.RGBA{15, 20, 40, 240}, false)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(panelW), float32(panelH), 2, color.RGBA{255, 200, 50, 200}, false)

	// Title
	title := "LEADERBOARD — HIGHSCORES"
	titleW := len(title) * charW
	printColoredAt(screen, title, sw/2-titleW/2, py+10, color.RGBA{255, 200, 50, 255})

	// Column headers
	y := py + 32
	headerCol := color.RGBA{100, 180, 255, 255}
	printColoredAt(screen, "#", px+10, y, headerCol)
	printColoredAt(screen, "Name", px+30, y, headerCol)
	printColoredAt(screen, "Score", px+220, y, headerCol)
	printColoredAt(screen, "Korrekt", px+290, y, headerCol)
	printColoredAt(screen, "Falsch", px+370, y, headerCol)
	printColoredAt(screen, "Eff.%", px+440, y, headerCol)
	printColoredAt(screen, "Modus", px+510, y, headerCol)
	printColoredAt(screen, "Gen", px+590, y, headerCol)
	printColoredAt(screen, "Bots", px+640, y, headerCol)

	// Separator
	y += lineH + 2
	vector.StrokeLine(screen, float32(px+5), float32(y), float32(px+panelW-5), float32(y),
		1, color.RGBA{60, 70, 90, 200}, false)
	y += 4

	entries := swarm.LeaderboardTop(lb, 15)
	for i, e := range entries {
		rankCol := color.RGBA{180, 180, 180, 255}
		nameCol := color.RGBA{200, 210, 230, 255}
		scoreCol := color.RGBA{180, 180, 180, 255}

		// Gold/Silver/Bronze for top 3
		switch i {
		case 0:
			rankCol = color.RGBA{255, 215, 0, 255}   // gold
			nameCol = color.RGBA{255, 215, 0, 255}
			scoreCol = color.RGBA{255, 215, 0, 255}
		case 1:
			rankCol = color.RGBA{192, 192, 192, 255}  // silver
			nameCol = color.RGBA{200, 200, 210, 255}
		case 2:
			rankCol = color.RGBA{205, 127, 50, 255}   // bronze
			nameCol = color.RGBA{205, 160, 100, 255}
		}

		printColoredAt(screen, fmt.Sprintf("%d", i+1), px+10, y, rankCol)

		// Truncate name to fit
		name := e.Name
		if len(name) > 22 {
			name = name[:22] + ".."
		}
		printColoredAt(screen, name, px+30, y, nameCol)
		printColoredAt(screen, fmt.Sprintf("%d", e.Score), px+220, y, scoreCol)
		printColoredAt(screen, fmt.Sprintf("%d", e.Correct), px+290, y, color.RGBA{100, 255, 100, 255})
		printColoredAt(screen, fmt.Sprintf("%d", e.Wrong), px+370, y, color.RGBA{255, 100, 100, 255})
		printColoredAt(screen, fmt.Sprintf("%.0f%%", e.Efficiency), px+440, y, color.RGBA{180, 180, 180, 255})
		printColoredAt(screen, e.Mode, px+510, y, color.RGBA{150, 150, 170, 255})
		printColoredAt(screen, fmt.Sprintf("%d", e.Generation), px+590, y, color.RGBA{150, 150, 170, 255})
		printColoredAt(screen, fmt.Sprintf("%d", e.BotCount), px+640, y, color.RGBA{150, 150, 170, 255})

		y += lineH + 1
	}

	if len(entries) == 0 {
		printColoredAt(screen, "Noch keine Eintraege — spiele und liefere Pakete!",
			px+40, y+20, color.RGBA{140, 150, 170, 255})
	}

	// Footer
	footerY := py + panelH - 22
	footer := "ESC = Schliessen  |  Scores werden bei Programm-Wechsel/Reset gespeichert"
	printColoredAt(screen, footer, px+40, footerY, color.RGBA{100, 100, 120, 255})
}
