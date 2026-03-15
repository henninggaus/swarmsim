package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Help overlay colors
var (
	colorHelpBg      = color.RGBA{0, 0, 0, 225}
	colorHelpTitle   = color.RGBA{136, 204, 255, 255} // cyan
	colorHelpSection = color.RGBA{255, 200, 80, 255}  // gold section headers
	colorHelpKey     = color.RGBA{136, 204, 255, 255} // cyan keys
	colorHelpText    = color.RGBA{200, 200, 210, 255}
	colorHelpDim     = color.RGBA{120, 120, 140, 255}
	colorHelpSyntax  = color.RGBA{0, 255, 255, 255}   // cyan for SwarmScript keywords
	colorHelpSensor  = color.RGBA{0, 255, 100, 255}   // green for sensors
	colorHelpAction  = color.RGBA{255, 180, 50, 255}  // orange for actions
	colorHelpSep     = color.RGBA{50, 55, 70, 255}
	colorHelpFeature = color.RGBA{180, 220, 255, 255}
)

// DrawHelpOverlay renders the full-screen help overlay.
// Two-column layout: left = keyboard reference, right = feature explanations.
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
	title := "SWARMSIM HILFE"
	titleImg := cachedTextImage(title)
	titleScale := 1.5
	titleTotalW := float64(titleImg.Bounds().Dx()) * titleScale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(titleScale, titleScale)
	op.GeoM.Translate(float64(sw)/2-titleTotalW/2, float64(y))
	op.ColorScale.Scale(136.0/255, 204.0/255, 1.0, 1.0)
	screen.DrawImage(titleImg, op)
	y += 30

	// Separator
	vector.StrokeLine(screen, float32(px), float32(y), float32(sw-px), float32(y), 1, colorHelpSep, false)
	y += 8

	// ========================
	// LEFT COLUMN: Keyboard Reference
	// ========================
	ly := y

	// -- ALLGEMEIN --
	printColoredAt(screen, "ALLGEMEIN", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKV(screen, px, &ly, []kv{
		{"Space", "Pause / Fortsetzen"},
		{"+/-", "Geschwindigkeit (0.5x - 5.0x)"},
		{"H", "Diese Hilfe"},
		{"F3", "Tutorial starten"},
		{"F10", "Screenshot (PNG)"},
		{"F11", "GIF Aufnahme"},
		{"S", "Sound ein/aus"},
		{"`/Oe", "Log-Konsole"},
		{"ESC", "Beenden"},
	})
	ly += 6

	// -- MODI --
	printColoredAt(screen, "MODI", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKV(screen, px, &ly, []kv{
		{"F1", "Classic Mode (5 Bot-Typen, Pheromone, Evolution)"},
		{"F2", "Swarm Lab (SwarmScript Editor, empfohlen)"},
	})
	ly += 6

	// -- NAVIGATION --
	printColoredAt(screen, "NAVIGATION", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKV(screen, px, &ly, []kv{
		{"Mausrad", "Zoom rein/raus"},
		{"Rechte Maus", "Kamera verschieben"},
		{"Linksklick", "Bot auswaehlen"},
		{"F", "Kamera folgt selektiertem Bot"},
		{"Tab", "Log: Alle / Nur selektierter Bot"},
	})
	ly += 6

	// -- SWARM LAB --
	printColoredAt(screen, "SWARM LAB (F2)", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKV(screen, px, &ly, []kv{
		{"L", "Lichtquelle setzen"},
		{"T", "Trails anzeigen"},
		{"C", "Lieferrouten / Challenge (Teams)"},
		{"D", "Dashboard ein/aus"},
		{"M", "Minimap"},
		{"N", "Neue Runde (Trucks/Teams)"},
		{"V", "Genom-Visualisierung (Evolution)"},
	})
	ly += 8

	// -- SWARMSCRIPT --
	vector.StrokeLine(screen, float32(px), float32(ly), float32(midX-20), float32(ly), 1, colorHelpSep, false)
	ly += 6
	printColoredAt(screen, "SWARMSCRIPT", px, ly, colorHelpSection)
	ly += lineH + 2

	printColoredAt(screen, "Syntax:", px+5, ly, colorHelpDim)
	printColoredAt(screen, "IF", px+50, ly, colorHelpSyntax)
	printColoredAt(screen, "<sensor> <op> <wert>", px+50+3*charW, ly, colorHelpSensor)
	printColoredAt(screen, "THEN", px+50+24*charW, ly, colorHelpSyntax)
	printColoredAt(screen, "<aktion>", px+50+29*charW, ly, colorHelpAction)
	ly += lineH

	printColoredAt(screen, "# Kommentare | $A:15 = evolvierbar", px+50, ly, color.RGBA{90, 90, 100, 255})
	ly += lineH + 4

	// Example
	printColoredAt(screen, "Beispiel:", px+5, ly, colorHelpDim)
	ly += lineH
	exampleLines := []struct{ text string; clr color.RGBA }{
		{"IF carry == 0 AND p_dist < 20 THEN PICKUP", colorHelpText},
		{"IF match == 1 THEN GOTO_DROPOFF", colorHelpText},
		{"IF near_dist < 15 THEN TURN_FROM_NEAREST", colorHelpText},
		{"IF true THEN FWD", colorHelpText},
	}
	for _, ex := range exampleLines {
		printColoredAt(screen, "  "+ex.text, px+5, ly, ex.clr)
		ly += lineH
	}
	ly += 6

	// -- SENSOREN (compact) --
	printColoredAt(screen, "SENSOREN (Auswahl)", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKVSensor(screen, px, &ly, []kv{
		{"near_dist", "Abstand zum naechsten Bot"},
		{"neighbors", "Anzahl Nachbarn im Sensorradius"},
		{"carry", "1 = traegt Paket, 0 = leer"},
		{"match", "1 = passende Dropoff sichtbar"},
		{"p_dist", "Abstand zur naechsten Pickup-Station"},
		{"d_dist", "Abstand zur naechsten Dropoff-Station"},
		{"has_pkg", "1 = Pickup hat ein Paket bereit"},
		{"obs_ahead", "1 = Hindernis voraus"},
		{"wall_right", "1 = Wand rechts (Maze-Navigation)"},
		{"edge", "1 = Am Arena-Rand"},
		{"rnd", "Zufallszahl 0-100"},
		{"team", "Team-Zugehoerigkeit (1=A, 2=B)"},
	})
	ly += 6

	// -- AKTIONEN (compact) --
	printColoredAt(screen, "AKTIONEN (Auswahl)", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKVAction(screen, px, &ly, []kv{
		{"FWD", "Vorwaerts bewegen"},
		{"STOP", "Anhalten"},
		{"TURN_RIGHT N", "Um N Grad drehen"},
		{"TURN_RANDOM", "Zufaellige Richtung"},
		{"TURN_FROM_NEAREST", "Vom naechsten Bot weg"},
		{"PICKUP", "Paket aufheben"},
		{"DROP", "Paket ablegen"},
		{"GOTO_DROPOFF", "Zur Dropoff-Station drehen"},
		{"AVOID_OBSTACLE", "Hindernis ausweichen"},
		{"WALL_FOLLOW_RIGHT", "Rechte-Hand-Regel (Maze)"},
		{"SET_LED R G B", "LED-Farbe setzen"},
		{"SEND_MESSAGE N", "Nachricht broadcasten"},
	})

	// ========================
	// RIGHT COLUMN: Feature Explanations
	// ========================
	ry := y

	printColoredAt(screen, "FEATURES", midX, ry, colorHelpSection)
	ry += lineH + 4

	// Delivery
	printColoredAt(screen, "DELIVERY", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Pakete von Pickup (gefuellt) zur gleichfarbigen",
		"Dropoff (Ring) transportieren. 4 Farben: R, B, Y, G.",
		"Sensoren: carry, match, p_dist, d_dist, has_pkg.",
	})
	ry += 6

	// Trucks
	printColoredAt(screen, "TRUCKS", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"LKW faehrt an Rampe, Bots entladen Pakete.",
		"Max 3 Bots gleichzeitig auf der Rampe (Semaphor).",
		"Sensoren: on_ramp, truck_here, truck_pkg_count.",
	})
	ry += 6

	// Evolution
	printColoredAt(screen, "EVOLUTION", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Zahlenwerte ($A-$Z) optimieren sich ueber",
		"Generationen. Die erfolgreichsten Bots vererben",
		"ihre Werte. Mutation + Crossover + Elitismus.",
	})
	ry += 6

	// GP
	printColoredAt(screen, "GP (GENETISCHE PROGRAMMIERUNG)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Die Regeln SELBST evolvieren. Jeder Bot hat ein",
		"eigenes Programm. Crossover + Mutation erzeugen",
		"neue Programme. 'Export Best' speichert Ergebnis.",
		"Fitness: Deliveries*30 + Pickups*15 + Dist*0.01",
	})
	ry += 6

	// Teams
	printColoredAt(screen, "TEAMS", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Zwei Teams (Blau vs Rot) konkurrieren um Pakete.",
		"Verschiedene Programme pro Team moeglich.",
		"C = Challenge (5000 Ticks), N = Neue Runde.",
		"Sensoren: team, team_score, enemy_score.",
	})
	ry += 6

	// Dashboard
	printColoredAt(screen, "DASHBOARD (D)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Echtzeit-Statistiken: Fitness-Graph, Heatmap,",
		"Bot-Rankings, Delivery-Rate, Event-Ticker.",
	})
	ry += 6

	// Tutorial
	printColoredAt(screen, "TUTORIAL (F3)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"15-Schritte-Tutorial beim ersten Start.",
		"F3 zum erneuten Starten, ESC zum Ueberspringen.",
	})
	ry += 8

	// Classic Mode
	vector.StrokeLine(screen, float32(midX), float32(ry), float32(sw-px), float32(ry), 1, colorHelpSep, false)
	ry += 6
	printColoredAt(screen, "CLASSIC MODE (F1)", midX, ry, colorHelpSection)
	ry += lineH + 2
	helpKV(screen, midX, &ry, []kv{
		{"1-5", "Scout/Worker/Leader/Tank/Healer spawnen"},
		{"R", "Ressource platzieren"},
		{"O", "Hindernis platzieren"},
		{"P", "Pheromone (OFF -> FOUND -> ALL)"},
		{"E", "Generation erzwingen (Evolution)"},
		{"V", "Genom-Overlay"},
		{"N", "Naechstes Szenario"},
		{"WASD", "Kamera bewegen"},
	})

	// Vertical separator between columns
	vector.StrokeLine(screen, float32(midX-15), float32(y), float32(midX-15), float32(sh-30), 1, colorHelpSep, false)

	// Footer
	footerY := sh - 20
	footer := "H = Hilfe schliessen  |  Pfeiltasten / Mausrad = Scrollen  |  F3 = Tutorial"
	footerW := len(footer) * charW
	vector.DrawFilledRect(screen, 0, float32(footerY-5), float32(sw), float32(lineH+10), color.RGBA{0, 0, 0, 240}, false)
	printColoredAt(screen, footer, sw/2-footerW/2, footerY, colorHelpDim)
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
		printColoredAt(screen, item.desc, px+100, *y, colorHelpDim)
		*y += lineH
	}
}

func helpKVAction(screen *ebiten.Image, px int, y *int, items []kv) {
	for _, item := range items {
		printColoredAt(screen, item.key, px+5, *y, colorHelpAction)
		printColoredAt(screen, item.desc, px+130, *y, colorHelpDim)
		*y += lineH
	}
}

func helpParagraph(screen *ebiten.Image, px int, y *int, lines []string) {
	for _, line := range lines {
		printColoredAt(screen, line, px+5, *y, colorHelpText)
		*y += lineH
	}
}
