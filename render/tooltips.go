package render

import (
	"image/color"
	"strings"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TooltipState tracks hover state for tooltips.
type TooltipState struct {
	HoverID    string // ID of element being hovered
	HoverTicks int    // how many ticks hovering on this element
	Visible    bool   // tooltip is showing
	X, Y       int    // tooltip position
	Lines      []string
}

const tooltipDelay = 25 // ~0.42 seconds at 60fps — fast feedback for discoverability
const tooltipMaxW = 42  // chars per line before wrap

// UpdateTooltip updates tooltip state based on current hover position.
func UpdateTooltip(ts *TooltipState, mx, my int, currentHitID string) {
	if currentHitID == "" || currentHitID == "editor" || currentHitID == "botcount" {
		ts.HoverID = ""
		ts.HoverTicks = 0
		ts.Visible = false
		return
	}

	if currentHitID == ts.HoverID {
		ts.HoverTicks++
	} else {
		ts.HoverID = currentHitID
		ts.HoverTicks = 0
		ts.Visible = false
	}

	if ts.HoverTicks >= tooltipDelay {
		text := getTooltipText(currentHitID)
		if text != "" {
			ts.Visible = true
			ts.Lines = wrapTooltipText(text, tooltipMaxW)
			ts.X = mx + 12
			ts.Y = my - len(ts.Lines)*lineH - 8
			// Clamp to screen
			tipW := tooltipPixelWidth(ts.Lines)
			if ts.X+tipW > 1270 {
				ts.X = 1270 - tipW
			}
			if ts.Y < 5 {
				ts.Y = my + 20
			}
		}
	}
}

// DrawTooltip renders the tooltip if visible.
func DrawTooltip(screen *ebiten.Image, ts *TooltipState) {
	if !ts.Visible || len(ts.Lines) == 0 {
		return
	}

	w := tooltipPixelWidth(ts.Lines) + 12
	h := len(ts.Lines)*lineH + 8

	// Background
	vector.DrawFilledRect(screen, float32(ts.X-2), float32(ts.Y-2), float32(w+4), float32(h+4), color.RGBA{0, 0, 0, 230}, false)
	vector.StrokeRect(screen, float32(ts.X-2), float32(ts.Y-2), float32(w+4), float32(h+4), 1, color.RGBA{100, 120, 160, 200}, false)

	// Text
	for i, line := range ts.Lines {
		printColoredAt(screen, line, ts.X+4, ts.Y+4+i*lineH, color.RGBA{220, 225, 240, 255})
	}
}

// getTooltipText returns the tooltip text for an ID via locale lookup.
func getTooltipText(id string) string {
	key := "tooltip." + id
	localized := locale.T(key)
	if localized != key {
		return localized
	}
	return ""
}

func wrapTooltipText(text string, maxChars int) []string {
	words := strings.Fields(text)
	var lines []string
	var current string
	for _, word := range words {
		if current == "" {
			current = word
		} else if len(current)+1+len(word) <= maxChars {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func tooltipPixelWidth(lines []string) int {
	maxLen := 0
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	return maxLen * charW
}
