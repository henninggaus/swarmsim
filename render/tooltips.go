package render

import (
	"image/color"
	"strings"

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

const tooltipDelay = 40 // ~0.66 seconds at 60fps
const tooltipMaxW = 42  // chars per line before wrap

var tooltipRegistry = map[string]string{
	// Buttons
	"deploy":     "Laedt das aktuelle Programm auf alle Bots (Strg+Enter)",
	"reset":      "Setzt alle Bots auf Startpositionen zurueck",
	"text_mode":  "SwarmScript als Text editieren",
	"block_mode": "Visueller Block-Editor — Regeln zusammenklicken",
	"bots_plus":  "Mehr Bots hinzufuegen (+10)",
	"bots_minus": "Bots entfernen (-10)",

	// Toggles
	"obstacles": "Zufaellige Hindernisse in der Arena. Bots muessen ausweichen (obs_ahead Sensor).",
	"maze":      "Labyrinth-Generierung. Bots navigieren durch Gaenge. Nutze wall_right/wall_left fuer Wall-Following.",
	"light":     "Lichtquelle in der Arena. Taste L zum Positionieren. TURN_TO_LIGHT navigiert dorthin.",
	"walls":     "BOUNCE = Bots prallen am Rand ab. WRAP = Bots gehen durch, kommen auf der anderen Seite raus.",
	"delivery":  "Paket-Liefermodus. 4 Pickup-Stationen (gefuellt) und 4 Dropoff-Stationen (Ring). Farblich passend transportieren.",
	"trucks":    "LKW faehrt an Rampe, Bots entladen Pakete. Nach Entladung faehrt LKW weg, naechster kommt.",
	"evolution": "Parameter-Evolution. Zahlenwerte ($A-$Z) optimieren sich ueber Generationen durch natuerliche Selektion.",
	"gp":        "Genetische Programmierung. Die Regeln SELBST evolvieren. Bots erfinden eigene Programme. 'Export Best' speichert das beste.",
	"teams":     "Zwei Teams (Blau vs Rot) mit verschiedenen Programmen. Gleiche Arena — wer liefert mehr?",

	// Dropdown presets
	"preset:Aggregation":        "Bots finden sich zu Clustern zusammen",
	"preset:Dispersion":         "Bots verteilen sich gleichmaessig",
	"preset:Orbit":              "Bots umkreisen die Lichtquelle (Light ON noetig)",
	"preset:Color Wave":         "Farbwelle breitet sich durch den Schwarm aus",
	"preset:Flocking":           "Schwarmverhalten wie Voegel — Boids-Algorithmus",
	"preset:Snake Formation":    "Bots bilden Ketten (Follow-Mechanik)",
	"preset:Obstacle Nav":       "Hindernisnavigation (Obstacles ON noetig)",
	"preset:Pulse Sync":         "Synchronisierte Blitz-Rhythmen wie Gluehwuermchen",
	"preset:Trail Follow":       "Bots folgen Spuren anderer Bots",
	"preset:Ant Colony":         "Vereinfachter Ameisenalgorithmus (Light ON noetig)",
	"preset:Simple Delivery":    "Zufaellige Exploration, Pakete einsammeln und abliefern",
	"preset:Delivery Comm":      "Wie Simple, aber Bots teilen Stationspositionen",
	"preset:Delivery Roles":     "Spezialisierung: 50% Scouts, 50% Carrier",
	"preset:Simple Unload":      "LKW-Entladung ohne Kommunikation (Trucks ON)",
	"preset:Coordinated Unload": "LKW-Entladung mit LED-Gradient und Beacons",
	"preset:Evolving Delivery":  "Delivery mit evolvierbaren Parametern ($A-$Z)",
	"preset:Evolving Truck":     "Truck-Entladung mit evolvierbaren Parametern",
	"preset:Maze Explorer":      "Wall-Following Navigation im Labyrinth",
	"preset:GP: Random Start":   "Genetische Programmierung — komplett zufaellig",
	"preset:GP: Seeded Start":   "GP basierend auf Simple Delivery als Seed",
}

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
		text, ok := tooltipRegistry[currentHitID]
		if ok {
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
