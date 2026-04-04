package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"swarmsim/locale"
)

// Help overlay colors
var (
	colorHelpBg      = color.RGBA{0, 0, 0, 225}
	colorHelpTitle   = color.RGBA{136, 204, 255, 255} // cyan
	colorHelpSection = ColorSectionGold                // gold section headers
	colorHelpKey     = color.RGBA{136, 204, 255, 255}  // cyan keys
	colorHelpText    = color.RGBA{200, 200, 210, 255}
	colorHelpDim     = color.RGBA{120, 120, 140, 255}
	colorHelpSyntax  = color.RGBA{0, 255, 255, 255}   // cyan for SwarmScript keywords
	colorHelpSensor  = color.RGBA{0, 255, 100, 255}   // green for sensors
	colorHelpAction  = color.RGBA{255, 180, 50, 255}  // orange for actions
	colorHelpSep     = color.RGBA{50, 55, 70, 255}
	colorHelpFeature = color.RGBA{180, 220, 255, 255}
	colorHelpAlgo    = color.RGBA{200, 160, 255, 255} // purple for algorithm names
	colorHelpNote    = color.RGBA{160, 200, 160, 255} // soft green for notes/tips
	colorHelpMath    = color.RGBA{255, 220, 100, 255} // gold for math formulas
	colorHelpApply   = color.RGBA{140, 220, 180, 255} // green for "how it's applied"
	colorHelpAlgoHdr = color.RGBA{180, 140, 255, 255} // purple for algo sub-headers
)

// DrawHelpOverlay renders the full-screen help overlay.
// Two-column layout: left = SwarmScript reference, right = feature explanations.
func DrawHelpOverlay(screen *ebiten.Image, isSwarmMode bool, scrollY int) {
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// Semi-transparent background
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), colorHelpBg, false)

	// Layout
	px := 30
	midX := sw/2 + 10
	y := 20 - scrollY

	// Title
	title := locale.T("help.title")
	titleImg := cachedTextImage(title)
	titleScale := 1.5
	titleTotalW := float64(titleImg.Bounds().Dx()) * titleScale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(titleScale, titleScale)
	op.GeoM.Translate(float64(sw)/2-titleTotalW/2, float64(y))
	op.ColorScale.Scale(136.0/255, 204.0/255, 1.0, 1.0)
	screen.DrawImage(titleImg, op)
	y += 22

	// Subtitle
	sub := locale.T("help.subtitle")
	subW := runeLen(sub) * charW
	printColoredAt(screen, sub, sw/2-subW/2, y, colorHelpDim)
	y += lineH + 4

	// Separator
	vector.StrokeLine(screen, float32(px), float32(y), float32(sw-px), float32(y), 1, colorHelpSep, false)
	y += 8

	// LEFT COLUMN: Quick Start + SwarmScript Reference
	leftEndY := drawHelpLeftColumn(screen, px, midX, y, scrollY)

	// RIGHT COLUMN: Feature Explanations + Math + Concepts + Footer
	drawHelpRightColumnAndMath(screen, px, midX, y, scrollY, leftEndY)
}

type kv struct{ key, desc string }

func helpKV(screen *ebiten.Image, px int, y *int, items []kv) {
	for _, item := range items {
		printColoredAt(screen, item.key, px+5, *y, colorHelpKey)
		printColoredAt(screen, item.desc, px+100, *y, colorHelpText)
		*y += lineH
	}
}

func helpKVSensor(screen *ebiten.Image, px int, y *int, items []kv) {
	for _, item := range items {
		printColoredAt(screen, item.key, px+5, *y, colorHelpSensor)
		printColoredAt(screen, item.desc, px+110, *y, colorHelpDim)
		*y += lineH
	}
}

func helpKVAction(screen *ebiten.Image, px int, y *int, items []kv) {
	for _, item := range items {
		printColoredAt(screen, item.key, px+5, *y, colorHelpAction)
		printColoredAt(screen, item.desc, px+145, *y, colorHelpDim)
		*y += lineH
	}
}

func helpKVDim(screen *ebiten.Image, px int, y *int, items []kv) {
	dimKey := color.RGBA{90, 120, 150, 200}
	dimDesc := color.RGBA{90, 90, 105, 200}
	for _, item := range items {
		printColoredAt(screen, item.key, px+5, *y, dimKey)
		printColoredAt(screen, item.desc, px+100, *y, dimDesc)
		*y += lineH
	}
}

func helpParagraph(screen *ebiten.Image, px int, y *int, lines []string) {
	for _, line := range lines {
		printColoredAt(screen, line, px+5, *y, colorHelpText)
		*y += lineH
	}
}
