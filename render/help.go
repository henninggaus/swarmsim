package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Help overlay colors
var (
	colorHelpBg      = color.RGBA{0, 0, 0, 220}
	colorHelpTitle   = color.RGBA{136, 204, 255, 255} // cyan
	colorHelpSection = color.RGBA{255, 200, 80, 255}  // gold section headers
	colorHelpKey     = color.RGBA{136, 204, 255, 255}  // cyan keys
	colorHelpText    = color.RGBA{200, 200, 210, 255}
	colorHelpDim     = color.RGBA{120, 120, 140, 255}
	colorHelpSyntax  = color.RGBA{0, 255, 255, 255}   // cyan for SwarmScript keywords
	colorHelpSensor  = color.RGBA{0, 255, 100, 255}   // green for sensors
	colorHelpAction  = color.RGBA{255, 180, 50, 255}  // orange for actions
)

// DrawHelpOverlay renders the full-screen help overlay.
func DrawHelpOverlay(screen *ebiten.Image, isSwarmMode bool, scrollY int) {
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// Semi-transparent background
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), colorHelpBg, false)

	// Content area with padding
	px := 40 // padding x
	y := 20 - scrollY

	// Title
	title := "HILFE - Tastaturkuerzel & Referenz"
	titleW := len(title) * charW
	titleImg := cachedTextImage(title)
	titleScale := 1.5
	titleTotalW := float64(titleImg.Bounds().Dx()) * titleScale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(titleScale, titleScale)
	op.GeoM.Translate(float64(sw)/2-titleTotalW/2, float64(y))
	op.ColorScale.Scale(136.0/255, 204.0/255, 1.0, 1.0)
	screen.DrawImage(titleImg, op)
	_ = titleW
	y += 35

	// Separator
	vector.StrokeLine(screen, float32(px), float32(y), float32(sw-px), float32(y), 1, color.RGBA{60, 60, 80, 255}, false)
	y += 10

	// === GLOBAL SHORTCUTS ===
	printColoredAt(screen, "-- Globale Tasten (alle Modi) --", px, y, colorHelpSection)
	y += lineH + 2

	globalKeys := []struct{ key, desc string }{
		{"Space", "Pause / Fortsetzen"},
		{"+/-", "Geschwindigkeit aendern (0.5x - 5.0x)"},
		{"F1", "Classic Mode (Sandbox, Foraging, Labyrinth, Energy, Evolution)"},
		{"F2", "Swarm Lab (SwarmScript Editor)"},
		{"F10", "Screenshot speichern (PNG)"},
		{"F11", "GIF Aufnahme starten/stoppen"},
		{"F12", "CPU Profiling (Build-Tag: profile)"},
		{"H", "Hilfe ein-/ausblenden"},
		{"S", "Sound ein-/ausschalten"},
		{"`/Oe", "Log-Konsole ein-/ausblenden"},
		{"F/Tab/T/M", "Comm-Radius / Bot-Info / Trails / Minimap"},
		{"ESC", "Beenden"},
	}
	for _, kv := range globalKeys {
		printColoredAt(screen, kv.key, px+5, y, colorHelpKey)
		printColoredAt(screen, kv.desc, px+120, y, colorHelpText)
		y += lineH
	}
	y += 8

	// === STANDARD MODE ===
	printColoredAt(screen, "-- Classic Mode (F1) --", px, y, colorHelpSection)
	y += lineH + 2

	stdKeys := []struct{ key, desc string }{
		{"1-5", "Bot spawnen (Scout, Worker, Leader, Tank, Healer)"},
		{"R", "Ressource platzieren"},
		{"O", "Hindernis platzieren"},
		{"F", "Kommunikationsradius anzeigen"},
		{"G", "Sensorradius anzeigen"},
		{"D", "Debug-Kommunikationslinien"},
		{"T", "Bot-Trails anzeigen"},
		{"M", "Minimap anzeigen"},
		{"P", "Pheromone (OFF -> FOUND -> ALL)"},
		{"E", "Generation beenden (Evolution erzwingen)"},
		{"V", "Genom-Overlay anzeigen"},
		{"N", "Naechstes Szenario (Classic Mode)"},
		{"WASD", "Kamera bewegen"},
		{"Mausrad", "Zoom"},
		{"Rechtsklick", "Kamera ziehen"},
		{"Linksklick", "Bot auswaehlen"},
	}
	for _, kv := range stdKeys {
		printColoredAt(screen, kv.key, px+5, y, colorHelpKey)
		printColoredAt(screen, kv.desc, px+120, y, colorHelpText)
		y += lineH
	}
	y += 8

	// === SWARM MODE ===
	printColoredAt(screen, "-- Swarm Lab (F2) --", px, y, colorHelpSection)
	y += lineH + 2

	swarmKeys := []struct{ key, desc string }{
		{"L", "Lichtquelle ein-/ausschalten (Klick fuer Position)"},
		{"T", "Trails anzeigen"},
		{"C", "Lieferrouten anzeigen"},
		{"M", "Minimap anzeigen"},
		{"Editor", "Linksklick zum Fokussieren, Tab=4 Spaces"},
	}
	for _, kv := range swarmKeys {
		printColoredAt(screen, kv.key, px+5, y, colorHelpKey)
		printColoredAt(screen, kv.desc, px+120, y, colorHelpText)
		y += lineH
	}
	y += 12

	// === SWARMSCRIPT REFERENCE ===
	vector.StrokeLine(screen, float32(px), float32(y), float32(sw-px), float32(y), 1, color.RGBA{60, 60, 80, 255}, false)
	y += 8
	printColoredAt(screen, "-- SwarmScript Referenz --", px, y, colorHelpSection)
	y += lineH + 2

	// Syntax
	printColoredAt(screen, "Syntax:", px+5, y, colorHelpDim)
	printColoredAt(screen, "IF", px+60, y, colorHelpSyntax)
	printColoredAt(screen, "<sensor> <op> <wert>", px+60+3*charW, y, colorHelpSensor)
	printColoredAt(screen, "[AND ...]", px+60+24*charW, y, colorHelpSyntax)
	printColoredAt(screen, "THEN", px+60+34*charW, y, colorHelpSyntax)
	printColoredAt(screen, "<aktion> [param]", px+60+39*charW, y, colorHelpAction)
	y += lineH
	printColoredAt(screen, "# Kommentare beginnen mit #", px+60, y, color.RGBA{100, 100, 100, 255})
	y += lineH + 6

	// Sensors in two columns
	col1X := px + 5
	col2X := sw/2 + 20

	printColoredAt(screen, "Sensoren:", col1X, y, colorHelpDim)
	printColoredAt(screen, "Aktionen:", col2X, y, colorHelpDim)
	y += lineH + 2

	// Left column: sensors
	sensors := []struct{ name, desc string }{
		{"near_dist", "Abstand zum naechsten Bot"},
		{"neighbors / nbrs", "Anzahl Nachbarn"},
		{"edge", "Am Rand? (0/1)"},
		{"obs_ahead", "Hindernis voraus? (0/1)"},
		{"obs_dist", "Abstand zum Hindernis"},
		{"light", "Lichtwert (0-255)"},
		{"rnd", "Zufallszahl (0-100)"},
		{"state / my_state", "Eigener Zustand (0-9)"},
		{"counter", "Zaehler-Wert"},
		{"timer", "Timer-Ticks"},
		{"tick", "Globaler Tick"},
		{"leader / follower", "In Kette? (0/1)"},
		{"chain_len", "Kettenlaenge"},
		{"msg", "Nachricht empfangen? (0/1)"},
		{"carry", "Traegt Paket? (0/1)"},
		{"match", "Passendes Dropoff? (0/1)"},
		{"p_dist / d_dist", "Pickup/Dropoff Abstand"},
		{"has_pkg", "Pickup hat Paket? (0/1)"},
		{"heard_pickup", "Pickup-Nachricht gehoert"},
		{"heard_dropoff", "Dropoff-Nachricht gehoert"},
		{"led_dist", "Passende LED-Distanz"},
		{"value1 / value2", "Benutzervariablen"},
	}

	// Right column: actions
	actions := []struct{ name, desc string }{
		{"FWD / FWD_SLOW", "Vorwaerts (normal/langsam)"},
		{"STOP", "Anhalten"},
		{"TURN_LEFT N", "Links drehen (N Grad)"},
		{"TURN_RIGHT N", "Rechts drehen (N Grad)"},
		{"TURN_TO_NEAREST", "Zum naechsten Bot drehen"},
		{"TURN_FROM_NEAREST", "Vom naechsten weg drehen"},
		{"TURN_TO_CENTER", "Zur Mitte drehen"},
		{"TURN_TO_LIGHT", "Zum Licht drehen"},
		{"TURN_RANDOM", "Zufaellig drehen"},
		{"AVOID_OBSTACLE", "Hindernis ausweichen"},
		{"FOLLOW_NEAREST", "Naechsten Bot folgen"},
		{"UNFOLLOW", "Folgen beenden"},
		{"SET_STATE N", "Zustand setzen (0-9)"},
		{"SET_LED R G B", "LED-Farbe setzen"},
		{"COPY_LED", "LED vom Naechsten kopieren"},
		{"SEND_MESSAGE N", "Nachricht senden"},
		{"PICKUP / DROP", "Paket aufnehmen/ablegen"},
		{"GOTO_PICKUP", "Zum Pickup navigieren"},
		{"GOTO_DROPOFF", "Zum passenden Dropoff"},
		{"GOTO_LED", "Zur passenden LED"},
		{"SEND_PICKUP N", "Pickup-Info broadcasten"},
		{"SEND_DROPOFF N", "Dropoff-Info broadcasten"},
	}

	// Render both columns
	sensorY := y
	for _, s := range sensors {
		if sensorY+lineH < sh-30 { // don't render below screen
			printColoredAt(screen, s.name, col1X, sensorY, colorHelpSensor)
			descX := col1X + 20*charW
			printColoredAt(screen, s.desc, descX, sensorY, colorHelpDim)
		}
		sensorY += lineH
	}

	actionY := y
	for _, a := range actions {
		if actionY+lineH < sh-30 {
			printColoredAt(screen, a.name, col2X, actionY, colorHelpAction)
			descX := col2X + 20*charW
			printColoredAt(screen, a.desc, descX, actionY, colorHelpDim)
		}
		actionY += lineH
	}

	// Footer
	footerY := sh - 20
	footer := "H = Hilfe schliessen  |  Pfeil hoch/runter = Scrollen"
	footerW := len(footer) * charW
	vector.DrawFilledRect(screen, 0, float32(footerY-5), float32(sw), float32(lineH+10), color.RGBA{0, 0, 0, 240}, false)
	printColoredAt(screen, footer, sw/2-footerW/2, footerY, colorHelpDim)
}
