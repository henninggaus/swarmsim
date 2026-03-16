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
	"deploy":     "Laedt das aktuelle Programm auf alle Bots (Strg+Enter). Alle Bots fuehren danach die Regeln im Editor aus.",
	"reset":      "Setzt alle Bots auf zufaellige Startpositionen zurueck. Statistiken und Fitness werden ebenfalls zurueckgesetzt.",
	"text_mode":  "SwarmScript als Text editieren — jede Zeile ist eine IF...THEN Regel. Volle Kontrolle ueber die Bot-Logik.",
	"block_mode": "Visueller Block-Editor — Regeln per Dropdown zusammenklicken. Ideal fuer Einsteiger, kein Tippen noetig.",
	"bots_plus":  "Mehr Bots hinzufuegen (+10). Mehr Bots = komplexeres emergentes Verhalten, aber hoehere CPU-Last.",
	"bots_minus": "Bots entfernen (-10). Weniger Bots = schnellere Simulation, aber weniger Schwarm-Effekte.",

	// Toggles
	"obstacles": "Zufaellige Hindernisse in der Arena. Bots muessen mit dem Sensor 'obs_ahead' erkennen und mit AVOID_OBSTACLE ausweichen. Testet Navigations-Algorithmen.",
	"maze":      "Generiert ein Labyrinth mit Gaengen. Bots nutzen 'wall_right' und 'wall_left' Sensoren fuer Wall-Following. Klassisches Robotik-Problem: den Ausgang finden!",
	"light":     "Lichtquelle in der Arena (L = Position setzen). Bots messen 'light' (0-100) und navigieren mit TURN_TO_LIGHT. Simuliert Phototaxis wie bei Insekten.",
	"walls":     "BOUNCE = Bots prallen am Arena-Rand ab wie an einer Wand. WRAP = Bots laufen durch den Rand und erscheinen auf der Gegenseite (Torus-Topologie).",
	"delivery":  "Paket-Liefersystem: 4 farbige Pickup-Stationen (gefuellte Kreise) und 4 Dropoff-Stationen (Ringe). Bots muessen Pakete zur gleichfarbigen Dropoff bringen. Sensoren: carry, match, p_dist, d_dist.",
	"trucks":    "LKW-Entladung: Ein LKW faehrt an die Rampe, Bots entladen Pakete (max. 3 gleichzeitig auf der Rampe = Semaphor). Nach Entladung faehrt der LKW weg und der naechste kommt.",
	"evolution": "Parameter-Evolution (Genetischer Algorithmus): Zahlenwerte $A-$Z im Programm werden ueber Generationen optimiert. Die Top 20% Bots vererben ihre Werte. Mutation + Crossover + Elitismus.",
	"gp":        "Genetische Programmierung: Die Regeln SELBST evolvieren! Jeder Bot erhaelt ein eigenes Programm. Crossover tauscht Regeln zwischen Bots, Mutation aendert Sensoren/Aktionen. Fitness: Deliveries*30 + Pickups*15.",
	"teams":     "Zwei Teams (Blau A vs Rot B) mit verschiedenen Programmen konkurrieren in der gleichen Arena um Pakete. C = Challenge-Modus (5000 Ticks), N = Neue Runde.",
	"neuro":     "NEUROEVOLUTION: Jeder Bot erhaelt ein eigenes kleines neuronales Netz (12 Sensoren -> 6 Hidden -> 8 Aktionen = 120 Gewichte). Statt Regeln zu programmieren, lernen die Bots durch Evolution der Netz-Gewichte. Die besten 20% vererben ihre Netze. Mutation veraendert Gewichte zufaellig. Aktiviert automatisch Delivery. Keine Programmierung noetig — das Netz entscheidet!",

	// Dropdown presets
	"preset:Aggregation":        "Aggregation: Bots finden sich zu Clustern. Jeder Bot dreht sich zum naechsten Nachbarn (TURN_TO_NEAREST). Simuliert soziale Anziehung — wie Schwarmfische sich zusammenfinden.",
	"preset:Dispersion":         "Dispersion: Bots verteilen sich gleichmaessig im Raum. Wenn ein Nachbar zu nah ist (<40px), dreht sich der Bot weg (TURN_FROM_NEAREST). Gegenteil von Aggregation.",
	"preset:Orbit":              "Orbit: Bots umkreisen die Lichtquelle. Wenn Licht stark (>80), dreht der Bot 90 Grad — erzeugt Kreisbahnen. Benoetigt Light ON. Simuliert Phototaxis bei Insekten.",
	"preset:Color Wave":         "Color Wave: Eine Farbwelle breitet sich durch den Schwarm aus. Ein Bot wird rot, Nachbarn kopieren die Farbe per Nachricht. Zeigt Informationsausbreitung in dezentralen Systemen.",
	"preset:Flocking":           "Flocking (Boids): Schwarmverhalten wie bei Voegeln nach Craig Reynolds. 3 Kraefte: Separation (Abstand halten), Alignment (gleiche Richtung), Cohesion (zusammenbleiben).",
	"preset:Snake Formation":    "Snake: Bots bilden Ketten durch Follow-Mechanik. Jeder Bot folgt dem naechsten (FOLLOW_NEAREST). State-Machine: State 0 = suchen, State 1 = folgen. Emergente Schlangen-Formation.",
	"preset:Obstacle Nav":       "Obstacle Navigation: Bots navigieren um Hindernisse zur Lichtquelle. Kombination aus AVOID_OBSTACLE und TURN_TO_LIGHT. Benoetigt Obstacles ON. Testet reaktive Navigation.",
	"preset:Pulse Sync":         "Pulse Sync: Synchronisierte Lichtblitze wie Gluehwuermchen. Bots haben interne Timer (counter), blitzen gleichzeitig. Erforscht Synchronisation in biologischen Systemen (Kuramoto-Modell).",
	"preset:Trail Follow":       "Trail Follow: Bots kopieren die LED-Farbe des naechsten Nachbarn. Farbspuren breiten sich aus. Zeigt wie lokale Nachahmung zu globalen Mustern fuehrt — emergentes Verhalten.",
	"preset:Ant Colony":         "Ant Colony: Vereinfachter Ameisenalgorithmus. Bots suchen die Lichtquelle (Futter), kommunizieren Fundstellen per SEND_MESSAGE an Nachbarn. Benoetigt Light ON.",
	"preset:Simple Delivery":    "Simple Delivery: Bots explorieren zufaellig, sammeln Pakete an Pickup-Stationen ein (PICKUP) und bringen sie zur passenden Dropoff (GOTO_DROPOFF, DROP). Grundlage fuer alle Delivery-Szenarien.",
	"preset:Delivery Comm":      "Delivery mit Kommunikation: Wie Simple Delivery, aber Bots senden Nachrichten wenn sie Pakete finden (SEND_MESSAGE). Nachbarn koennen reagieren — verbessert die Effizienz.",
	"preset:Delivery Roles":     "Delivery mit Rollen: 50% der Bots sind Scouts (State 0, erkunden) und 50% Carrier (State 1, transportieren). Spezialisierung durch einfache Rollenzuweisung zeigt Effizienzgewinn.",
	"preset:Simple Unload":      "Simple Unload: Grundlegende LKW-Entladung. Bots fahren zur Rampe, nehmen Pakete und bringen sie zur Dropoff. Benoetigt Trucks ON. Ohne Kommunikation — rein reaktiv.",
	"preset:Coordinated Unload": "Coordinated Unload: LKW-Entladung mit LED-Gradient (Wegweiser) und Beacon-Signalen. Bots koordinieren sich fuer effizientere Entladung. Zeigt wie Stigmergie (indirekte Kommunikation) hilft.",
	"preset:Evolving Delivery":  "Evolving Delivery: Delivery-Programm mit evolvierbaren Parametern ($A-$Z). Aktiviere Evolution ON — die Werte optimieren sich ueber Generationen. Beobachte wie Fitness steigt!",
	"preset:Evolving Truck":     "Evolving Truck: Truck-Entladung mit evolvierbaren Parametern. Kombination aus Genetischem Algorithmus und Logistik-Aufgabe. Aktiviere Evolution ON.",
	"preset:Maze Explorer":      "Maze Explorer: Rechte-Hand-Regel (Wall-Following). Bot haelt immer die rechte Wand und findet so jeden Ausgang. Klassischer Algorithmus aus der Robotik. Benoetigt Maze ON.",
	"preset:GP: Random Start":   "GP Random Start: Genetische Programmierung mit komplett zufaelligen Programmen. Jeder Bot startet mit anderen Regeln. Evolution findet die besten Strategien von Null.",
	"preset:GP: Seeded Start":   "GP Seeded Start: GP startet mit Simple Delivery als Basis. Die Evolution verbessert ein bereits funktionierendes Programm weiter. Oft schnellere Konvergenz als Random Start.",
	"preset:Neuro: Delivery":    "Neuroevolution: Jeder Bot hat ein eigenes neuronales Netz (12 Sensoren -> 6 Hidden -> 8 Aktionen = 120 Gewichte). Statt Regeln zu programmieren, lernt das Netz durch Evolution der Gewichte. NEURO wird automatisch aktiviert. Beobachte wie Bots von zufaelligem Verhalten zu gezieltem Liefern lernen!",

	// Stats panel items
	"stat:chains":       "Chains = Anzahl der Bot-Ketten (Follow-Formationen). Bots folgen dem naechsten Nachbarn mit FOLLOW_NEAREST. Laengere Ketten zeigen starke soziale Bindung.",
	"stat:coverage":     "Coverage = Prozent der Arena, die von Bots abgedeckt wird. 100% = Bots verteilt in alle Bereiche. Niedrig = Bots clustern zusammen. 20x20 Raster-Berechnung.",
	"stat:neighbors":    "Avg Neighbors = Durchschnittliche Nachbar-Anzahl pro Bot (120px Radius). Hoch = dicht gepackter Schwarm. Niedrig = Bots verteilt. Beeinflusst Kommunikations-Effizienz.",
	"stat:delivered":    "Geliefert = Anzahl erfolgreich zugestellter Pakete. Richtig = zur passenden Farbe geliefert. Rate = Korrektheit in Prozent. Ziel: 100% Korrektheit!",
	"stat:carrying":     "Tragen = Bots die gerade ein Paket transportieren. Suchen = Bots ohne Paket. Ideale Balance haengt vom Programm ab.",
	"stat:avgtime":      "Durchschn. Lieferzeit = Ticks von Pickup bis Dropoff gemittelt. Niedrigere Werte = effizientere Lieferstrategie. Optimiere mit Evolution!",
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
