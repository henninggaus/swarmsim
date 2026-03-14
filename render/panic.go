package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawPanicBanner renders a red error banner at the top of the screen.
func DrawPanicBanner(screen *ebiten.Image, msg string, timer int) {
	sw := screen.Bounds().Dx()

	// Fade alpha
	alpha := uint8(255)
	if timer < 60 {
		alpha = uint8(timer * 255 / 60)
	}

	// Truncate message
	maxChars := (sw - 20) / charW
	if len(msg) > maxChars {
		msg = msg[:maxChars-3] + "..."
	}

	// Red background banner
	bannerH := float32(lineH + 8)
	vector.DrawFilledRect(screen, 0, 0, float32(sw), bannerH,
		color.RGBA{150, 0, 0, alpha}, false)
	vector.StrokeLine(screen, 0, bannerH, float32(sw), bannerH, 1,
		color.RGBA{255, 80, 80, alpha}, false)

	// Error text
	textCol := color.RGBA{255, 200, 200, alpha}
	printColoredAt(screen, msg, 10, 4, textCol)
}
