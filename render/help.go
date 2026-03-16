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
	colorHelpAlgo    = color.RGBA{200, 160, 255, 255} // purple for algorithm names
	colorHelpNote    = color.RGBA{160, 200, 160, 255} // soft green for notes/tips
)

// DrawHelpOverlay renders the full-screen help overlay.
// Two-column layout: left = keyboard reference + SwarmScript, right = feature & algorithm explanations.
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
	y += 22

	// Subtitle
	sub := "Schwarm-Robotik-Simulator — Emergentes Verhalten durch einfache Regeln"
	subW := len(sub) * charW
	printColoredAt(screen, sub, sw/2-subW/2, y, colorHelpDim)
	y += lineH + 4

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
		{"+/-", "Geschwindigkeit stufenweise aendern"},
		{"1-5", "Speed-Presets: 0.5x / 1x / 2x / 5x / 10x"},
		{"H", "Diese Hilfe ein/ausblenden"},
		{"F3", "Interaktives Tutorial starten (15 Schritte)"},
		{"F10", "Screenshot als PNG speichern"},
		{"F11", "GIF-Aufnahme starten/stoppen"},
		{"S", "Sound ein/aus"},
		{"`/Oe", "Debug Log-Konsole oeffnen"},
		{"ESC", "Beenden / Overlay schliessen"},
	})
	ly += 6

	// -- MODI --
	printColoredAt(screen, "MODI", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKV(screen, px, &ly, []kv{
		{"F1", "Classic Mode (5 Bot-Typen, Pheromone, Evolution)"},
		{"F2", "Swarm Lab (SwarmScript Editor, empfohlen)"},
	})
	ly += 2
	helpParagraph(screen, px, &ly, []string{
		"  Classic = vordefinierte Bot-Typen mit Genom-Evolution",
		"  Swarm Lab = frei programmierbare Bots mit Editor",
	})
	ly += 6

	// -- NAVIGATION --
	printColoredAt(screen, "NAVIGATION", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKV(screen, px, &ly, []kv{
		{"Mausrad", "Zoom rein/raus (stufenlos)"},
		{"Rechte Maus", "Kamera verschieben (drag)"},
		{"Linksklick", "Bot auswaehlen (Info-Panel oeffnen)"},
		{"F", "Follow-Cam: Kamera folgt selektiertem Bot"},
		{"Tab", "Log filtern: Alle / Nur selektierter Bot"},
	})
	ly += 6

	// -- SWARM LAB --
	printColoredAt(screen, "SWARM LAB (F2)", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKV(screen, px, &ly, []kv{
		{"L", "Lichtquelle positionieren (klicken)"},
		{"T", "Bot-Trails (Bewegungsspuren) anzeigen"},
		{"C", "Lieferrouten anzeigen / Challenge starten"},
		{"D", "Dashboard mit Statistiken ein/aus"},
		{"M", "Minimap (Uebersichtskarte) ein/aus"},
		{"N", "Neue Runde (Trucks/Teams zuruecksetzen)"},
		{"V", "Genom-Visualisierung (bei Evolution)"},
		{"G", "Genom-Browser: sortierbare Bot-Liste (Evo/GP/Neuro)"},
		{"J", "Dynamische Umgebung (bewegl. Hindernisse + Paket-Verfall)"},
		{"O", "Arena-Editor: Hindernisse/Stationen platzieren (1/2/3)"},
		{"U", "Turnier-Modus: Programme gegeneinander antreten lassen"},
		{"Q", "Einzelschritt-Debugger (Q=1 Tick, Space=weiter)"},
		{"W", "Farb-Filter: Rot/Gruen/Blau/Traegt/Idle (zyklisch)"},
		{"Y", "Heatmap: zeigt wo Bots sich am meisten aufhalten"},
		{"F4", "Auto-Optimizer: testet Parameter automatisch"},
		{"F5", "Szenario-Kette: 3 Szenarien nacheinander durchlaufen"},
		{"X", "Stats als CSV in Clipboard exportieren"},
	})
	ly += 8

	// -- SWARMSCRIPT --
	vector.StrokeLine(screen, float32(px), float32(ly), float32(midX-20), float32(ly), 1, colorHelpSep, false)
	ly += 6
	printColoredAt(screen, "SWARMSCRIPT SPRACHE", px, ly, colorHelpSection)
	ly += lineH + 2

	helpParagraph(screen, px, &ly, []string{
		"Jeder Bot fuehrt die Regeln von oben nach unten aus.",
		"Die erste passende Regel bestimmt die Aktion.",
	})
	ly += 2

	printColoredAt(screen, "Syntax:", px+5, ly, colorHelpDim)
	printColoredAt(screen, "IF", px+55, ly, colorHelpSyntax)
	printColoredAt(screen, "<sensor> <op> <wert>", px+55+3*charW, ly, colorHelpSensor)
	printColoredAt(screen, "THEN", px+55+24*charW, ly, colorHelpSyntax)
	printColoredAt(screen, "<aktion>", px+55+29*charW, ly, colorHelpAction)
	ly += lineH

	printColoredAt(screen, "Mehrere Bedingungen:", px+5, ly, colorHelpDim)
	printColoredAt(screen, "IF ... AND ... AND ... THEN ...", px+130, ly, colorHelpSyntax)
	ly += lineH

	printColoredAt(screen, "Evolvierbare Werte:", px+5, ly, colorHelpDim)
	printColoredAt(screen, "$A:15", px+130, ly, colorHelpSensor)
	printColoredAt(screen, "(Variable A, Startwert 15)", px+165, ly, colorHelpDim)
	ly += lineH

	printColoredAt(screen, "Kommentare:", px+5, ly, colorHelpDim)
	printColoredAt(screen, "# Text wird ignoriert", px+130, ly, color.RGBA{100, 100, 110, 255})
	ly += lineH + 4

	// Example
	printColoredAt(screen, "Beispiel (Delivery-Bot):", px+5, ly, colorHelpNote)
	ly += lineH
	exampleLines := []struct {
		text string
		clr  color.RGBA
	}{
		{"IF carry == 0 AND p_dist < 20 THEN PICKUP", colorHelpText},
		{"  # Kein Paket? Pickup in der Naehe? -> Aufheben", color.RGBA{90, 90, 100, 255}},
		{"IF match == 1 THEN GOTO_DROPOFF", colorHelpText},
		{"  # Passendes Ziel? -> Dorthin navigieren", color.RGBA{90, 90, 100, 255}},
		{"IF near_dist < 15 THEN TURN_FROM_NEAREST", colorHelpText},
		{"  # Zu nah an anderem Bot? -> Ausweichen", color.RGBA{90, 90, 100, 255}},
		{"IF true THEN FWD", colorHelpText},
		{"  # Sonst: einfach geradeaus (Fallback)", color.RGBA{90, 90, 100, 255}},
	}
	for _, ex := range exampleLines {
		printColoredAt(screen, "  "+ex.text, px+5, ly, ex.clr)
		ly += lineH
	}
	ly += 6

	// -- SENSOREN (complete) --
	printColoredAt(screen, "SENSOREN (vollstaendig)", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKVSensor(screen, px, &ly, []kv{
		{"near_dist", "Abstand zum naechsten Bot (in Pixeln)"},
		{"neighbors", "Anzahl Nachbarn im Sensorradius (120px)"},
		{"carry", "1 = traegt Paket, 0 = leer"},
		{"match", "1 = Dropoff der richtigen Farbe in Reichweite"},
		{"p_dist", "Abstand zur naechsten Pickup-Station"},
		{"d_dist", "Abstand zur naechsten Dropoff-Station"},
		{"has_pkg", "1 = Pickup hat ein Paket bereit"},
		{"light", "Lichtstaerke am Standort (0-100)"},
		{"obs_ahead", "1 = Hindernis voraus (Raycast)"},
		{"wall_right", "1 = Wand rechts (fuer Maze-Navigation)"},
		{"wall_left", "1 = Wand links (fuer Maze-Navigation)"},
		{"edge", "1 = Bot ist am Arena-Rand"},
		{"rnd", "Zufallszahl 0-100 (jedes Tick neu)"},
		{"tick", "Aktueller Simulations-Tick"},
		{"state", "Interner Zustand des Bots (0-9)"},
		{"counter", "Interner Zaehler (fuer Timer/Logik)"},
		{"heading", "Aktuelle Blickrichtung (0-359 Grad)"},
		{"team", "Team-Zugehoerigkeit (1=A, 2=B)"},
		{"team_score", "Punktestand des eigenen Teams"},
		{"enemy_score", "Punktestand des gegnerischen Teams"},
		{"msg", "1 = Nachricht empfangen (hat_message)"},
		{"on_ramp", "1 = Bot steht auf der LKW-Rampe"},
		{"truck_here", "1 = LKW ist an der Rampe geparkt"},
		{"truck_pkg", "Anzahl verbleibender Pakete im LKW"},
		{"speed", "Aktuelle Geschwindigkeit des Bots"},
	})
	ly += 6

	// -- AKTIONEN (complete) --
	printColoredAt(screen, "AKTIONEN (vollstaendig)", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKVAction(screen, px, &ly, []kv{
		{"FWD", "Geradeaus bewegen"},
		{"STOP", "Anhalten (0 Geschwindigkeit)"},
		{"TURN_RIGHT N", "Um N Grad nach rechts drehen"},
		{"TURN_LEFT N", "Um N Grad nach links drehen"},
		{"TURN_RANDOM", "Zufaellige neue Richtung"},
		{"TURN_TO_NEAREST", "Zum naechsten Bot drehen"},
		{"TURN_FROM_NEAREST", "Vom naechsten Bot weg drehen"},
		{"TURN_TO_LIGHT", "Richtung Lichtquelle drehen"},
		{"TURN_TO_CENTER", "Zur Mitte der Nachbarn drehen"},
		{"FOLLOW_NEAREST", "Dem naechsten Bot folgen"},
		{"PICKUP", "Paket aufheben (an Pickup)"},
		{"DROP", "Paket ablegen (an Dropoff)"},
		{"GOTO_DROPOFF", "Zur passenden Dropoff drehen"},
		{"GOTO_PICKUP", "Zur naechsten Pickup drehen"},
		{"AVOID_OBSTACLE", "Hindernis ausweichen (Lenkung)"},
		{"WALL_FOLLOW_RIGHT", "Rechte-Hand-Regel (Maze)"},
		{"WALL_FOLLOW_LEFT", "Linke-Hand-Regel (Maze)"},
		{"SET_LED R G B", "LED auf Farbe setzen (0-255)"},
		{"COPY_LED", "LED des naechsten Bots kopieren"},
		{"SEND_MESSAGE N", "Nachricht Typ N broadcasten"},
		{"SET_STATE N", "Internen Zustand auf N setzen"},
		{"INC_COUNTER", "Internen Zaehler erhoehen (+1)"},
		{"RESET_COUNTER", "Internen Zaehler auf 0 setzen"},
		{"GOTO_BEACON", "Zum Beacon-Signal navigieren"},
	})

	// ========================
	// RIGHT COLUMN: Feature & Algorithm Explanations
	// ========================
	ry := y

	printColoredAt(screen, "FEATURES & ALGORITHMEN", midX, ry, colorHelpSection)
	ry += lineH + 4

	// --- Emergentes Verhalten ---
	printColoredAt(screen, "WAS IST EMERGENTES VERHALTEN?", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Komplexe globale Muster entstehen aus einfachen",
		"lokalen Regeln. Kein Bot kennt den Gesamtplan —",
		"jeder reagiert nur auf seine direkte Umgebung.",
		"Beispiele: Vogelschwarm, Ameisenpfade, Fischschulen.",
	})
	ry += 6

	// --- Delivery ---
	printColoredAt(screen, "PAKET-DELIVERY", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"4 farbige Pickup-Stationen (gefuellte Kreise) und",
		"4 Dropoff-Stationen (Ringe). Bots muessen Pakete",
		"zur gleichfarbigen Dropoff transportieren.",
		"Farben: Rot, Blau, Gelb, Gruen.",
		"Traegen verlangsamt den Bot auf 70% Geschwindigkeit.",
		"Sensoren: carry, match, p_dist, d_dist, has_pkg.",
	})
	ry += 6

	// --- Trucks ---
	printColoredAt(screen, "LKW-ENTLADUNG", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Ein LKW faehrt an die Rampe und wird von Bots",
		"entladen. Maximal 3 Bots gleichzeitig auf der",
		"Rampe (Semaphor-Prinzip, verhindert Stau).",
		"Nach Entladung faehrt der LKW ab, naechster kommt.",
		"Sensoren: on_ramp, truck_here, truck_pkg_count.",
	})
	ry += 6

	// --- Boids ---
	printColoredAt(screen, "BOIDS-ALGORITHMUS (Flocking)", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Craig Reynolds, 1986. Drei einfache Regeln erzeugen",
		"realistisches Schwarmverhalten wie bei Voegeln:",
		"1. Separation: Abstand halten (Gewicht 1.5)",
		"2. Alignment: Gleiche Richtung (Gewicht 0.3)",
		"3. Cohesion: Zusammenbleiben (Gewicht 0.3)",
		"Jeder Bot sieht nur Nachbarn in 30-80px Radius.",
	})
	ry += 6

	// --- Evolution ---
	printColoredAt(screen, "GENETISCHER ALGORITHMUS", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Inspiriert von Darwins Evolution. Nach jeder",
		"Generation (500 Ticks) werden Bots nach Fitness",
		"bewertet. Die besten 20% vererben ihre Werte.",
		"Operatoren: Crossover (Gene mischen), Mutation",
		"(zufaellige Aenderungen), Elitismus (Top 3 direkt",
		"uebernommen). $A-$Z Werte im Programm evolvieren.",
	})
	ry += 6

	// --- GP ---
	printColoredAt(screen, "GENETISCHE PROGRAMMIERUNG (GP)", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Nicht nur Parameter, sondern die Programme selbst",
		"evolvieren! Jeder Bot hat eigene SwarmScript-Regeln.",
		"Crossover tauscht Regeln zwischen erfolgreichen Bots.",
		"Mutation aendert Sensoren, Schwellwerte oder Aktionen.",
		"Fitness: Deliveries*30 + Pickups*15 + Dist*0.01",
		"         - StuckCount*10 - IdleTicks*0.05",
		"10% jeder Generation sind komplett neue Programme.",
	})
	ry += 6

	// --- Neuroevolution ---
	printColoredAt(screen, "NEUROEVOLUTION", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Jeder Bot hat ein eigenes neuronales Netz statt",
		"handgeschriebener Regeln. Architektur: 12 Sensoren",
		"-> 6 Hidden (tanh) -> 8 Aktionen = 120 Gewichte.",
		"Die Gewichte evolvieren: Top 20% vererben, Crossover",
		"mischt, Mutation veraendert. Kein Programmieren noetig!",
		"Preset: Neuro: Delivery. Toggle: Neuro ON im Editor.",
	})
	ry += 6

	// --- Pheromone ---
	printColoredAt(screen, "PHEROMONE (Classic Mode)", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Indirekte Kommunikation ueber chemische Spuren",
		"(Stigmergie), wie bei Ameisen. Bots hinterlassen",
		"Pheromone die langsam verdunsten (Decay) und sich",
		"ausbreiten (Diffusion). 3 Kanaele: Suche (blau),",
		"Gefunden (gruen), Gefahr (rot). Nachfolgende Bots",
		"folgen dem Konzentrationsgradienten zum Ziel.",
	})
	ry += 6

	// --- Kommunikation ---
	printColoredAt(screen, "KOMMUNIKATION", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Bots senden Nachrichten per SEND_MESSAGE (Broadcast",
		"an alle in Reichweite). Nachrichtentypen haben",
		"verschiedene Reichweiten und Lebensdauer (TTL).",
		"Typen: ResourceFound, HelpNeeded, PackageFound,",
		"FormationJoin, Danger, RampCongested, TaskAssign.",
		"Empfang pruefen mit dem 'msg' Sensor.",
	})
	ry += 6

	// --- Teams ---
	printColoredAt(screen, "TEAM-WETTBEWERB", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Zwei Teams (Blau A vs Rot B) mit verschiedenen",
		"Programmen konkurrieren in derselben Arena.",
		"C = Challenge-Modus: 5000 Ticks, wer liefert mehr?",
		"N = Neue Runde starten. Teams sind farblich markiert.",
		"Sensoren: team (1=A, 2=B), team_score, enemy_score.",
	})
	ry += 6

	// --- Formationen ---
	printColoredAt(screen, "FORMATIONEN (Classic Mode)", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Bots koennen Formationen bilden: Kreis, Linie, V.",
		"Jeder Bot berechnet seine Zielposition relativ zum",
		"Leader und steuert dorthin. Slot-basiertes System",
		"mit Heading-abhaengiger Positionierung.",
	})
	ry += 6

	// --- Dashboard ---
	printColoredAt(screen, "DASHBOARD (D)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Echtzeit-Statistiken auf der rechten Seite:",
		"- Fitness-Graph: Best (gruen) + Avg (gelb)",
		"- Delivery-Rate: Lieferungen pro Zeitfenster",
		"- Heatmap: Bot-Bewegungsdichte (blau=wenig, rot=viel)",
		"- Bot-Ranking: Top 5 Bots nach Lieferungen",
		"- Event-Ticker: Live-Feed von Pickup/Delivery Events",
	})
	ry += 6

	// --- Bot-Typen Classic ---
	printColoredAt(screen, "BOT-TYPEN (Classic Mode)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Scout (Cyan): Schnell, erkundet, markiert Ressourcen",
		"Worker (Orange): Sammelt Ressourcen, traegt zur Basis",
		"Leader (Gold): Koordiniert, gibt Signale an Nachbarn",
		"Tank (Gruen): Langsam, robust, raeumt Hindernisse",
		"Healer (Pink): Repariert beschaedigte Nachbar-Bots",
	})
	ry += 8

	// --- Classic Mode Keys ---
	vector.StrokeLine(screen, float32(midX), float32(ry), float32(sw-px), float32(ry), 1, colorHelpSep, false)
	ry += 6
	printColoredAt(screen, "CLASSIC MODE TASTEN (F1)", midX, ry, colorHelpSection)
	ry += lineH + 2
	helpKV(screen, midX, &ry, []kv{
		{"1-5", "Scout/Worker/Leader/Tank/Healer spawnen"},
		{"R", "Ressource platzieren (bei Mausposition)"},
		{"O", "Hindernis platzieren"},
		{"P", "Pheromone (OFF -> FOUND -> ALL)"},
		{"E", "Generation erzwingen (Evolution)"},
		{"V", "Genom-Overlay (Parameterwerte anzeigen)"},
		{"N", "Naechstes Szenario laden"},
		{"WASD", "Kamera bewegen"},
	})
	ry += 8

	// --- Tips ---
	vector.StrokeLine(screen, float32(midX), float32(ry), float32(sw-px), float32(ry), 1, colorHelpSep, false)
	ry += 6
	printColoredAt(screen, "TIPPS FUER EINSTEIGER", midX, ry, colorHelpNote)
	ry += lineH + 2
	helpParagraph(screen, midX, &ry, []string{
		"1. Starte mit Swarm Lab (F2) und dem Tutorial (F3)",
		"2. Probiere erst Aggregation, dann Simple Delivery",
		"3. Schalte Evolution ON + Evolving Delivery ein",
		"4. Beobachte wie die Fitness ueber Generationen steigt",
		"5. Nutze das Dashboard (D) fuer Statistiken",
		"6. GP: Random Start zeigt Evolution von Null",
		"7. Neuro: Delivery — Bots lernen ganz ohne Code!",
		"8. Klicke einen Bot an und druecke F zum Folgen",
		"9. Experimentiere mit Maze + Maze Explorer",
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

func helpParagraph(screen *ebiten.Image, px int, y *int, lines []string) {
	for _, line := range lines {
		printColoredAt(screen, line, px+5, *y, colorHelpText)
		*y += lineH
	}
}
