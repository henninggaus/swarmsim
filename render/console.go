package render

import (
	"image/color"
	"swarmsim/logger"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	consoleMaxLines = 6
	consolePadding  = 4
)

// Console colors by level
var (
	colorConsoleInfo  = color.RGBA{160, 160, 175, 255}
	colorConsoleWarn  = color.RGBA{255, 200, 80, 255}
	colorConsoleError = color.RGBA{255, 80, 80, 255}
	colorConsoleBg    = color.RGBA{5, 5, 12, 210}
	colorConsoleLine  = color.RGBA{60, 60, 80, 200}
)

// DrawConsole renders the in-game log console at the bottom of the screen.
// In swarm mode it draws only in the arena area (right of editor panel).
func DrawConsole(screen *ebiten.Image, entries []logger.LogEntry, isSwarmMode bool) {
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// In swarm mode, offset to avoid covering editor buttons
	panelX := 0
	panelW := sw
	if isSwarmMode {
		panelX = editorPanelW + 2
		panelW = sw - panelX
	}

	panelH := consoleMaxLines*lineH + consolePadding*2 + 4
	panelY := sh - panelH

	// Background
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), colorConsoleBg, false)

	// Top border
	vector.StrokeLine(screen, float32(panelX), float32(panelY), float32(panelX+panelW), float32(panelY), 1, colorConsoleLine, false)

	// Title
	printColoredAt(screen, "~ Log", panelX+consolePadding, panelY+2, color.RGBA{100, 100, 120, 255})

	// Show last N entries
	maxVisible := consoleMaxLines - 1 // -1 for title line
	n := len(entries)
	start := 0
	if n > maxVisible {
		start = n - maxVisible
	}
	visible := entries[start:]

	y := panelY + lineH + consolePadding
	for _, e := range visible {
		col := consoleColorForLevel(e.Level)
		line := "[" + e.Tag + "] " + e.Message
		// Truncate long lines
		maxChars := (panelW - consolePadding*2) / charW
		if len(line) > maxChars {
			line = line[:maxChars-3] + "..."
		}
		printColoredAt(screen, line, panelX+consolePadding, y, col)
		y += lineH
	}
}

func consoleColorForLevel(l logger.Level) color.RGBA {
	switch l {
	case logger.LevelWarn:
		return colorConsoleWarn
	case logger.LevelError:
		return colorConsoleError
	default:
		return colorConsoleInfo
	}
}
