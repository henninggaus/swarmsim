package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"
	"strconv"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed fonts/JetBrainsMono-Regular.ttf
var fontTTF []byte

// fontSource is the parsed TTF face source, initialized on first use.
var fontSource *text.GoTextFaceSource

// fontSize chosen so that character advance ≈ charW (6px).
const fontSize = 10.0

var cachedFontFace *text.GoTextFace

// getFontFace returns a GoTextFace at the configured size.
// The result is cached to avoid allocating a new struct on every call.
func getFontFace() *text.GoTextFace {
	if cachedFontFace != nil {
		return cachedFontFace
	}
	if fontSource == nil {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(fontTTF))
		if err != nil {
			panic("failed to load embedded font: " + err.Error())
		}
		fontSource = src
	}
	cachedFontFace = &text.GoTextFace{
		Source: fontSource,
		Size:   fontSize,
	}
	return cachedFontFace
}

const (
	charW = 6  // monospace char width for DebugPrintAt
	lineH = 16 // line height in pixels
)

// runeLen returns the number of runes in s. Used for monospace text width calculations.
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// fmtNum formats an integer with thousands separators: 1000000 → "1,000,000"
func fmtNum(n int) string {
	if n < 0 {
		return "-" + fmtNum(-n)
	}
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	// Insert commas from right
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// fmtTime converts simulation ticks to a human-readable elapsed time string.
// Assumes 60 ticks per second (TPS=60).
func fmtTime(ticks int) string {
	totalSec := ticks / 60
	hours := totalSec / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	}
	return fmt.Sprintf("%dm%02ds", minutes, seconds)
}

// --- Helper drawing functions ---

func drawSwarmButton(screen *ebiten.Image, x, y, w, h int, label string, bgCol color.RGBA) {
	// Hover detection: brighten on mouse-over, darken on press
	mx, my := ebiten.CursorPosition()
	isOver := mx >= x && mx < x+w && my >= y && my < y+h
	if isOver {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			// Pressed: darken (inset effect)
			bgCol.R = uint8(max(0, int(bgCol.R)-20))
			bgCol.G = uint8(max(0, int(bgCol.G)-20))
			bgCol.B = uint8(max(0, int(bgCol.B)-20))
		} else {
			// Hover: brighten
			bgCol.R = uint8(min(255, int(bgCol.R)+25))
			bgCol.G = uint8(min(255, int(bgCol.G)+25))
			bgCol.B = uint8(min(255, int(bgCol.B)+25))
		}
	}
	// Main body
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), bgCol, false)
	// Top highlight (subtle light edge)
	highlightCol := color.RGBA{
		uint8(min(int(bgCol.R)+30, 255)),
		uint8(min(int(bgCol.G)+30, 255)),
		uint8(min(int(bgCol.B)+30, 255)),
		bgCol.A,
	}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), 1, highlightCol, false)
	// Bottom shadow
	shadowCol := color.RGBA{
		uint8(max(int(bgCol.R)-25, 0)),
		uint8(max(int(bgCol.G)-25, 0)),
		uint8(max(int(bgCol.B)-25, 0)),
		bgCol.A,
	}
	vector.DrawFilledRect(screen, float32(x), float32(y+h-1), float32(w), 1, shadowCol, false)
	// Border
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, color.RGBA{180, 180, 200, 80}, false)
	textX := x + (w-runeLen(label)*charW)/2
	textY := y + (h-12)/2
	printColoredAt(screen, label, textX, textY, ColorWhite)
}

// textCacheEntry holds a cached colored text image.
type textCacheEntry struct {
	img      *ebiten.Image
	lastUsed int
}

const textCacheMaxEntries = 512 // hard cap to prevent unbounded growth

var (
	textCache      = make(map[string]*textCacheEntry, 128)
	textCacheFrame int
)

// ClearTextCache evicts all cached text images. Call after language switch
// so that stale translations are not displayed.
func ClearTextCache() {
	for k, e := range textCache {
		if e.img != nil {
			e.img.Deallocate()
		}
		delete(textCache, k)
	}
}

// tickTextCache increments the frame counter and evicts stale entries.
// Call once per frame from the draw entry point.
func tickTextCache() {
	textCacheFrame++
	// Time-based eviction every 120 frames
	if textCacheFrame%120 == 0 {
		for k, e := range textCache {
			if textCacheFrame-e.lastUsed > 120 {
				if e.img != nil {
					e.img.Deallocate()
				}
				delete(textCache, k)
			}
		}
	}
	// Hard cap: if cache exceeds max, evict oldest entries
	if len(textCache) > textCacheMaxEntries {
		oldest := textCacheFrame
		oldestKey := ""
		for k, e := range textCache {
			if e.lastUsed < oldest {
				oldest = e.lastUsed
				oldestKey = k
			}
		}
		if oldestKey != "" {
			if e := textCache[oldestKey]; e.img != nil {
				e.img.Deallocate()
			}
			delete(textCache, oldestKey)
		}
	}
}

// cachedTextImage returns a cached white-text image for the given string.
// Used by HUD functions that need a scaled text image.
func cachedTextImage(s string) *ebiten.Image {
	key := "__hud__" + s
	entry, ok := textCache[key]
	if !ok {
		tw := runeLen(s)*6 + 10
		if tw < 1 {
			tw = 1
		}
		th := 16
		img := ebiten.NewImage(tw, th)
		face := getFontFace()
		op := &text.DrawOptions{}
		op.GeoM.Translate(5, 3)
		op.ColorScale.Scale(1, 1, 1, 1) // white
		text.Draw(img, s, face, op)
		entry = &textCacheEntry{img: img}
		textCache[key] = entry
	}
	entry.lastUsed = textCacheFrame
	return entry.img
}

// printColoredAt draws colored text at the given position.
// Uses a text image cache to avoid per-frame GPU image allocations.
func printColoredAt(screen *ebiten.Image, s string, x, y int, col color.RGBA) {
	if s == "" {
		return
	}

	// Build cache key: text + color bytes
	key := s + string([]byte{col.R, col.G, col.B, col.A})

	entry, ok := textCache[key]
	if !ok {
		tw := runeLen(s)*charW + 2
		if tw < 1 {
			tw = 1
		}
		th := lineH
		img := ebiten.NewImage(tw, th)
		face := getFontFace()
		top := &text.DrawOptions{}
		top.ColorScale.Scale(1, 1, 1, 1) // white; tinted later
		text.Draw(img, s, face, top)
		entry = &textCacheEntry{img: img}
		textCache[key] = entry
	}
	entry.lastUsed = textCacheFrame

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))

	// Cached image is white text; tint to desired color
	r := float64(col.R) / 255.0
	g := float64(col.G) / 255.0
	b := float64(col.B) / 255.0
	a := float64(col.A) / 255.0
	op.ColorScale.Scale(float32(r), float32(g), float32(b), float32(a))

	screen.DrawImage(entry.img, op)
}
