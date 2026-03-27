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
	// LEFT COLUMN: Quick Start + SwarmScript Reference
	// ========================
	ly := y

	// -- SCHNELLSTART --
	printColoredAt(screen, "WO ANFANGEN?", px, ly, color.RGBA{120, 255, 120, 255})
	ly += lineH + 2
	helpParagraph(screen, px, &ly, []string{
		"1. Klicke F3 fuer das interaktive Tutorial",
		"2. Oder F2 und waehle ein Preset im Dropdown",
		"3. Klicke DEPLOY — beobachte die Bots!",
		"4. Im Tab 'Evo' schalte Evolution ON ein",
		"5. Im Tab 'Anzeige' aktiviere das Dashboard",
	})
	ly += 6

	// -- BEDIENUNG --
	printColoredAt(screen, "BEDIENUNG", px, ly, colorHelpSection)
	ly += lineH + 2
	helpParagraph(screen, px, &ly, []string{
		"Alles wird per Maus gesteuert:",
		"",
		"  Linksklick    Bot auswaehlen (zeigt Info-Panel)",
		"  Rechte Maus   Kamera verschieben (drag)",
		"  Mausrad       Zoom rein/raus",
		"  Space         Pause / Fortsetzen",
		"  H             Diese Hilfe ein/ausblenden",
		"  ESC           Beenden / Overlay schliessen",
		"",
		"Alle Features sind ueber die 4 Tabs links steuerbar (Algo via F4):",
	})
	ly += 2
	helpKV(screen, px, &ly, []kv{
		{"Arena", "Hindernisse, Labyrinth, Licht, Pakete, LKW"},
		{"Evo", "Evolution, Gen. Programmierung, Neuro, Teams"},
		{"Anzeige", "Dashboard, Spuren, Heatmap, Minimap, ..."},
		{"Werkzeuge", "Tempo, Zeitreise, Bildschirmfoto, Export"},
		{"F4 Algo-Labor", "20 Optimierungs-Algorithmen per Klick"},
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
		{"near_dist", "Abstand zum naechsten Bot (Pixel)"},
		{"neighbors", "Nachbarn im Sensorradius (120px)"},
		{"carry", "1 = traegt Paket, 0 = leer"},
		{"match", "1 = passende Dropoff in Reichweite"},
		{"p_dist", "Abstand zur naechsten Pickup"},
		{"d_dist", "Abstand zur naechsten Dropoff"},
		{"has_pkg", "1 = Pickup hat Paket bereit"},
		{"light", "Lichtstaerke (0-100)"},
		{"obs_ahead", "1 = Hindernis voraus"},
		{"wall_right", "1 = Wand rechts (Maze)"},
		{"wall_left", "1 = Wand links (Maze)"},
		{"edge", "1 = Am Arena-Rand"},
		{"rnd", "Zufallszahl 0-100"},
		{"tick", "Aktueller Simulations-Tick"},
		{"state", "Interner Zustand (0-9)"},
		{"counter", "Interner Zaehler"},
		{"heading", "Blickrichtung (0-359 Grad)"},
		{"team", "Team (1=A, 2=B)"},
		{"team_score", "Eigenes Team-Score"},
		{"enemy_score", "Gegner-Score"},
		{"msg", "1 = Nachricht empfangen"},
		{"on_ramp", "1 = Auf LKW-Rampe"},
		{"truck_here", "1 = LKW an Rampe"},
		{"truck_pkg", "Pakete im LKW"},
		{"speed", "Aktuelle Geschwindigkeit"},
		{"bot_ahead", "Bots im 90-Grad-Kegel vorn"},
		{"bot_behind", "Bots hinten"},
		{"bot_left", "Bots links"},
		{"bot_right", "Bots rechts"},
		{"visited_here", "Besuche aktuelle Zelle"},
		{"visited_ahead", "Besuche Zelle voraus"},
		{"explored", "% erkundet (Memory)"},
		{"group_carry", "% Nachbarn die tragen"},
		{"group_speed", "Avg Speed Nachbarn"},
		{"group_size", "Cluster-Groesse"},
	})
	ly += 6

	// -- AKTIONEN (complete) --
	printColoredAt(screen, "AKTIONEN (vollstaendig)", px, ly, colorHelpSection)
	ly += lineH + 2
	helpKVAction(screen, px, &ly, []kv{
		{"FWD", "Geradeaus bewegen"},
		{"STOP", "Anhalten"},
		{"TURN_RIGHT N", "N Grad rechts drehen"},
		{"TURN_LEFT N", "N Grad links drehen"},
		{"TURN_RANDOM", "Zufaellige Richtung"},
		{"TURN_TO_NEAREST", "Zum naechsten Bot"},
		{"TURN_FROM_NEAREST", "Vom naechsten Bot weg"},
		{"TURN_TO_LIGHT", "Zur Lichtquelle"},
		{"TURN_TO_CENTER", "Zur Nachbar-Mitte"},
		{"FOLLOW_NEAREST", "Naechstem Bot folgen"},
		{"PICKUP", "Paket aufheben"},
		{"DROP", "Paket ablegen"},
		{"GOTO_DROPOFF", "Zur passenden Dropoff"},
		{"GOTO_PICKUP", "Zur naechsten Pickup"},
		{"AVOID_OBSTACLE", "Hindernis ausweichen"},
		{"WALL_FOLLOW_RIGHT", "Rechte-Hand-Regel"},
		{"WALL_FOLLOW_LEFT", "Linke-Hand-Regel"},
		{"SET_LED R G B", "LED-Farbe setzen"},
		{"COPY_LED", "LED kopieren"},
		{"SEND_MESSAGE N", "Nachricht broadcasten"},
		{"SET_STATE N", "Zustand setzen"},
		{"INC_COUNTER", "Zaehler +1"},
		{"RESET_COUNTER", "Zaehler auf 0"},
		{"GOTO_BEACON", "Zum Beacon navigieren"},
	})

	// ========================
	// RIGHT COLUMN: Feature Explanations
	// ========================
	ry := y

	// ==============================================
	// GLOSSAR: Begriffe in Alltagssprache
	// ==============================================
	printColoredAt(screen, "BEGRIFFE — EINFACH ERKLAERT", midX, ry, colorHelpSection)
	ry += lineH + 4

	glossarCol := color.RGBA{255, 220, 140, 255} // warm gold for terms
	glossarDesc := color.RGBA{190, 195, 210, 255}

	// Each term: bold name + plain-language explanation
	glossarItems := []struct{ term, desc1, desc2 string }{
		{"Bot",
			"Ein kleiner autonomer Roboter. Wie eine Ameise:",
			"sieht nur 120 Pixel weit, kennt keinen Masterplan."},
		{"Sensor",
			"Was der Bot wahrnimmt: Abstand zum Nachbarn, Licht,",
			"ob er ein Paket traegt. Wie Augen und Fuehler."},
		{"Fitness",
			"Eine Zahl die sagt 'wie gut macht der Bot seinen Job'.",
			"Hoeher = besser. Wie eine Schulnote, nur andersrum."},
		{"Evolution",
			"Die 20% besten Bots 'vererben' ihre Einstellungen.",
			"Wie in der Natur: was funktioniert, ueberlebt."},
		{"Parameter ($A-$Z)",
			"Zahlenwerte im Programm die Evolution veraendern darf.",
			"Wie Regler an einem Mischpult — Evolution dreht dran."},
		{"Emergenz",
			"Wenn einfache Regeln zusammen etwas Komplexes ergeben.",
			"Jeder Vogel folgt 3 Regeln — trotzdem fliegt der Schwarm."},
		{"Exploration",
			"Neues ausprobieren, weit suchen. Wie: 'Ich probiere",
			"heute mal ein neues Restaurant aus.'"},
		{"Exploitation",
			"Das Bekannte verfeinern. Wie: 'Ich gehe zum Lieblings-",
			"italiener und nehme mein Stammgericht.'"},
		{"Konvergenz",
			"Alle Loesungen naehern sich dem Optimum an.",
			"Wie Leute die sich an einer Bushaltestelle sammeln."},
		{"Lokales Optimum",
			"Eine gute Loesung, aber nicht die beste. Wie der beste",
			"Italiener deiner Strasse — es gibt vielleicht einen besseren."},
	}

	for _, g := range glossarItems {
		printColoredAt(screen, g.term, midX+5, ry, glossarCol)
		ry += lineH
		printColoredAt(screen, "  "+g.desc1, midX+5, ry, glossarDesc)
		ry += lineH
		printColoredAt(screen, "  "+g.desc2, midX+5, ry, glossarDesc)
		ry += lineH + 3
	}

	ry += 4
	vector.StrokeLine(screen, float32(midX), float32(ry), float32(sw-px), float32(ry), 1, colorHelpSep, false)
	ry += 8

	printColoredAt(screen, "FEATURES & ALGORITHMEN", midX, ry, colorHelpSection)
	ry += lineH + 4

	// --- Emergentes Verhalten ---
	printColoredAt(screen, "WAS IST EMERGENTES VERHALTEN?", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Stell dir einen Vogelschwarm vor: Kein Vogel weiss",
		"wohin der Schwarm fliegt. Jeder folgt 3 Regeln:",
		"Abstand halten, gleiche Richtung, zusammenbleiben.",
		"Trotzdem fliegen Tausende synchron — OHNE Anweiser!",
		"",
		"Genau das passiert hier: Deine Bots kennen nur ihre",
		"direkte Umgebung. Komplexe Muster entstehen von selbst.",
	})
	ry += 6

	// --- Delivery ---
	printColoredAt(screen, "PAKET-DELIVERY (Tab: Arena)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"4 farbige Pickup-Stationen (gefuellte Kreise) und",
		"4 Dropoff-Stationen (Ringe). Bots muessen Pakete",
		"zur gleichfarbigen Dropoff transportieren.",
		"Sensoren: carry, match, p_dist, d_dist, has_pkg.",
	})
	ry += 6

	// --- Trucks ---
	printColoredAt(screen, "LKW-ENTLADUNG (Tab: Arena)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Ein LKW faehrt an die Rampe und wird von Bots",
		"entladen. Max. 3 Bots gleichzeitig auf der Rampe.",
		"Sensoren: on_ramp, truck_here, truck_pkg.",
	})
	ry += 6

	// --- Evolution ---
	printColoredAt(screen, "EVOLUTION (Tab: Evo)", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Genetischer Algorithmus: $A-$Z Parameter werden",
		"ueber Generationen optimiert. Top 20% vererben",
		"Werte, Crossover + Mutation + Elitismus.",
	})
	ry += 6

	// --- GP ---
	printColoredAt(screen, "GENETISCHE PROGRAMMIERUNG (Tab: Evo)", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Programme selbst evolvieren! Jeder Bot hat eigene",
		"SwarmScript-Regeln. Crossover tauscht Regeln,",
		"Mutation aendert Sensoren/Aktionen.",
	})
	ry += 6

	// --- Neuroevolution ---
	printColoredAt(screen, "NEUROEVOLUTION (Tab: Evo)", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Jeder Bot hat ein neuronales Netz: 12 Sensoren",
		"-> 6 Hidden -> 8 Aktionen = 120 Gewichte.",
		"Gewichte evolvieren statt Regeln zu schreiben.",
		"Preset: 'Neuro: Delivery' waehlen.",
	})
	ry += 6

	// --- Algorithmen ---
	printColoredAt(screen, "20 OPTIMIERUNGS-ALGORITHMEN (Tab: Algo)", midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"GWO, WOA, BFO, MFO, Cuckoo Search, Differential",
		"Evolution, ABC, Harmony Search, Bat, Harris Hawks,",
		"SSA, GSA, FPA, Simulated Annealing, Aquila, SCA,",
		"Dragonfly, TLBO, Equilibrium, Jaya.",
		"",
		"Alle per Klick im 'Algo' Tab aktivierbar.",
		"Radar Chart vergleicht aktive Algorithmen.",
	})
	ry += 6

	// --- Teams ---
	printColoredAt(screen, "TEAM-WETTBEWERB (Tab: Evo)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Zwei Teams (Blau A vs Rot B) mit verschiedenen",
		"Programmen konkurrieren in derselben Arena.",
		"Sensoren: team (1=A, 2=B), team_score, enemy_score.",
	})
	ry += 6

	// --- Kommunikation ---
	printColoredAt(screen, "KOMMUNIKATION", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"SEND_MESSAGE N broadcastet an Bots in Reichweite.",
		"Typen: ResourceFound, HelpNeeded, PackageFound,",
		"FormationJoin, Danger, RampCongested, TaskAssign.",
	})
	ry += 6

	// --- Dashboard ---
	printColoredAt(screen, "DASHBOARD (Tab: Anzeige)", midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		"Echtzeit-Statistiken: Fitness-Graph, Delivery-Rate,",
		"Heatmap, Bot-Ranking, Event-Ticker.",
	})
	ry += 6

	// --- Tipps ---
	vector.StrokeLine(screen, float32(midX), float32(ry), float32(sw-px), float32(ry), 1, colorHelpSep, false)
	ry += 6
	printColoredAt(screen, "TIPPS FUER EINSTEIGER", midX, ry, colorHelpNote)
	ry += lineH + 2
	helpParagraph(screen, midX, &ry, []string{
		"1. Starte mit dem Tutorial (F3)",
		"2. Probiere erst Aggregation, dann Simple Delivery",
		"3. Tab 'Evo': Evolution ON + Evolving Delivery",
		"4. Beobachte wie die Fitness steigt",
		"5. Tab 'Anzeige': Dashboard fuer Statistiken",
		"6. GP: Random Start = Programme von Null evolvieren",
		"7. Neuro: Delivery = Bots lernen ohne Code!",
		"8. Klicke einen Bot an fuer Details",
		"9. Tab 'Algo': Algorithmen vergleichen",
	})

	// Vertical separator between columns (up to where math section starts)
	mathStartY := ly
	if ry > ly {
		mathStartY = ry
	}
	mathStartY += 10
	vector.StrokeLine(screen, float32(midX-15), float32(y), float32(midX-15), float32(mathStartY-10), 1, colorHelpSep, false)

	// ========================
	// FULL-WIDTH: Mathematical Foundations
	// ========================
	my_ := mathStartY
	vector.StrokeLine(screen, float32(px), float32(my_), float32(sw-px), float32(my_), 1, colorHelpSep, false)
	my_ += 8

	// Section title
	mathTitle := "MATHEMATISCHE GRUNDLAGEN DER ALGORITHMEN"
	mathTitleW := len(mathTitle) * charW
	printColoredAt(screen, mathTitle, sw/2-mathTitleW/2, my_, colorHelpSection)
	my_ += lineH + 2
	mathIntro := "Jeder Algorithmus basiert auf einer mathematischen Update-Regel. Hier: Formel + Anwendung im Simulator."
	printColoredAt(screen, mathIntro, px+5, my_, colorHelpDim)
	my_ += lineH + 6

	// Two-column math layout
	mlx := px          // left math column
	mrx := sw/2 + 10   // right math column
	colW := sw/2 - 40  // column width
	_ = colW

	// Helper: draw one algorithm math block
	type mathBlock struct {
		name    string
		formula []string
		applied []string
	}

	mathAlgos := []mathBlock{
		{
			"GWO — Grey Wolf Optimizer",
			[]string{
				"X(t+1) = X_p - A * |C * X_p - X(t)|",
				"A = 2a*r1 - a,  C = 2*r2",
				"a: 2 -> 0 linear ueber Iterationen",
			},
			[]string{
				"X_p = Position des Alpha-Wolfs (bester Bot).",
				"A > 1: Exploration (Wolfe schweifen aus).",
				"A < 1: Exploitation (Rudel kreist ein).",
				"Im Sim: Bot-Parameter konvergieren zum Besten.",
			},
		},
		{
			"WOA — Whale Optimization",
			[]string{
				"Spirale: X(t+1) = D'*e^(b*l)*cos(2*pi*l) + X*",
				"Einkreisen: X(t+1) = X* - A*|C*X* - X|",
				"D' = |X*(t) - X(t)|,  p < 0.5: Kreis, sonst Spirale",
			},
			[]string{
				"Modelliert Blasennetz-Jagd der Buckelwale.",
				"50% Chance: Einkreisen ODER Spirale.",
				"Spirale = logarithmisch enger werdend.",
				"Im Sim: Parameter umkreisen das Optimum.",
			},
		},
		{
			"Cuckoo Search + Levy-Fluege",
			[]string{
				"x(t+1) = x(t) + alpha * L(lambda)",
				"L(s) ~ s^(-lambda),  1 < lambda < 3",
				"Levy: u/|v|^(1/beta), u~N(0,sigma), v~N(0,1)",
			},
			[]string{
				"Levy-Fluege: viele kurze + seltene weite Spruenge.",
				"Optimal fuer Suche in unbekanntem Terrain!",
				"pa = 0.25: 25% schlechteste Nester werden ersetzt.",
				"Im Sim: Grosse Spruenge vermeiden lokale Optima.",
			},
		},
		{
			"DE — Differential Evolution",
			[]string{
				"Mutation: v = x_r1 + F*(x_r2 - x_r3)",
				"Crossover: u_j = v_j wenn rand < CR, sonst x_j",
				"Selektion: x = u wenn f(u) < f(x), sonst x",
			},
			[]string{
				"F = Skalierungsfaktor (0.5-1.0) der Differenz.",
				"CR = Crossover-Rate, steuert Parametermischung.",
				"Greedy: Nur Verbesserungen werden akzeptiert.",
				"Im Sim: 3 zufaellige Bots erzeugen Mutanten.",
			},
		},
		{
			"SA — Simulated Annealing",
			[]string{
				"P(accept) = exp(-deltaE / T)",
				"T(t) = T0 * alpha^t,  0.9 < alpha < 0.99",
				"deltaE = f(x_new) - f(x_current)",
			},
			[]string{
				"Hohe T: Fast alles akzeptiert (Exploration).",
				"Niedrige T: Nur Verbesserungen (Exploitation).",
				"Boltzmann-Verteilung aus der Thermodynamik!",
				"Im Sim: Bots 'kuehlen ab' ueber Generationen.",
			},
		},
		{
			"BFO — Bacterial Foraging",
			[]string{
				"Chemotaxis: theta(j+1) = theta(j) + C*delta/|delta|",
				"Schwimmen: gleiche Richtung, Taumeln: neue Richtung",
				"Reproduktion: Top 50% verdoppeln sich",
			},
			[]string{
				"E.coli navigiert per Tumble-and-Run.",
				"C = Schrittweite, delta = zufaelliger Vektor.",
				"Elimination: Zufaellig stirbt ein Bakterium.",
				"Im Sim: Bots 'schwimmen' durch den Parameterraum.",
			},
		},
		{
			"ABC — Artificial Bee Colony",
			[]string{
				"v_ij = x_ij + phi*(x_ij - x_kj)",
				"P(i) = fitness(i) / sum(fitness)",
				"Scout: x_j = x_min + rand*(x_max - x_min)",
			},
			[]string{
				"3 Phasen: Employed -> Onlooker -> Scout.",
				"Employed: Lokale Suche um aktuelle Position.",
				"Onlooker: Fitness-proportionale Selektion.",
				"Scout: Erschoepfte Quellen werden aufgegeben.",
			},
		},
		{
			"HSO — Harmony Search",
			[]string{
				"x_new = x_i aus Memory (HMCR)",
				"       +/- bw*rand (Pitch Adjust, PAR)",
				"       oder rand(min,max) (1-HMCR)",
			},
			[]string{
				"HMCR=0.95: 95% aus Erinnerung, 5% frisch.",
				"PAR=0.3: 30% Chance auf Feintuning.",
				"bw = Bandbreite der Pitch-Anpassung.",
				"Im Sim: 'Noten' = Parameter, 'Harmonie' = Fitness.",
			},
		},
		{
			"MFO — Moth-Flame Optimization",
			[]string{
				"S(M_i, F_j) = D*e^(bt)*cos(2*pi*t) + F_j",
				"D = |F_j - M_i|, t in [-1, 1]",
				"Flammen = sortierte beste Positionen",
			},
			[]string{
				"Motten fliegen Spiralen um Flammen (Licht).",
				"t=-1: engste Bahn, t=1: weiteste Bahn.",
				"Flammenanzahl sinkt: Fokussierung ueber Zeit.",
				"Im Sim: Bots spiralen um beste Loesungen.",
			},
		},
		{
			"SCA — Sine Cosine Algorithm",
			[]string{
				"x(t+1) = x(t) + r1*sin(r2)*|r3*P - x(t)|",
				"  oder   x(t) + r1*cos(r2)*|r3*P - x(t)|",
				"r1 = a*(1 - t/T), a=2, Amplitude nimmt ab",
			},
			[]string{
				"sin/cos oszillieren zwischen Explore & Exploit.",
				"r1 > 1: Exploration (Sinus geht ueber Ziel).",
				"r1 < 1: Exploitation (konvergiert).",
				"Im Sim: Parameter schwingen um das Optimum.",
			},
		},
		{
			"HHO — Harris Hawks Optimization",
			[]string{
				"Exploration: X = X_rand - r1*|X_rand - 2*r2*X|",
				"Soft Siege:  X = dX - E*|J*X_best - X|",
				"Rapid Dive:  X = X_best - E*|dX| + S*Levy(d)",
			},
			[]string{
				"E = Fluchtenergie des Beutetiers (2->0).",
				"|E|>=1: Explore, |E|<1: Exploit (Angriff).",
				"4 Phasen: Perch, Surprise, Siege, Dive.",
				"Im Sim: Bots jagen kooperativ das Optimum.",
			},
		},
		{
			"GSA — Gravitational Search",
			[]string{
				"F_ij = G(t) * M_i*M_j / (R_ij+eps) * (x_j-x_i)",
				"a_i = sum(F_ij) / M_i",
				"G(t) = G0 * exp(-alpha * t/T)",
			},
			[]string{
				"Masse proportional zu Fitness (besser=schwerer).",
				"Schwere Massen ziehen leichtere an.",
				"G sinkt: Anfangs starke, spaeter schwache Kraft.",
				"Im Sim: Gute Bots ziehen andere Parameter an.",
			},
		},
		{
			"FPA — Flower Pollination",
			[]string{
				"Global: x(t+1) = x(t) + gamma*L*(x_best - x(t))",
				"Lokal:  x(t+1) = x(t) + eps*(x_j - x_k)",
				"p=0.8: 80% global (Insekten), 20% lokal (Wind)",
			},
			[]string{
				"L = Levy-Flug (wie bei Cuckoo Search).",
				"Globale Bestaeubung: Insekten fliegen weit.",
				"Lokale: Wind = kleine zufaellige Aenderungen.",
				"Im Sim: Mischung aus weiten und nahen Spruengen.",
			},
		},
		{
			"TLBO — Teaching-Learning-Based",
			[]string{
				"Teacher: x_new = x + r*(x_best - TF*x_mean)",
				"Learner: x_new = x + r*(x_i - x_j) wenn f(i)<f(j)",
				"TF = round(1 + rand) in {1, 2}. KEINE Parameter!",
			},
			[]string{
				"Teacher-Phase: Lehrer hebt Klassendurchschnitt.",
				"Learner-Phase: Schueler lernt von besserem Peer.",
				"Einziger Algo OHNE Tuning-Parameter!",
				"Im Sim: Bots lernen von den Besten direkt.",
			},
		},
		{
			"Bat Algorithm",
			[]string{
				"f_i = f_min + (f_max-f_min)*beta",
				"v_i(t+1) = v_i(t) + (x_i - x_best)*f_i",
				"x_i(t+1) = x_i(t) + v_i(t+1)",
			},
			[]string{
				"Frequenz f: steuert Schrittweite.",
				"Lautstaerke A: sinkt bei Erfolg (leiser=feiner).",
				"Pulsrate r: steigt (mehr lokale Suche).",
				"Im Sim: Echoortung sucht den Parameterraum ab.",
			},
		},
		{
			"SSA — Salp Swarm Algorithm",
			[]string{
				"Leader:   x = F + c1*(ub-lb)*c2 + lb  (c3>=0.5)",
				"          x = F - c1*(ub-lb)*c2 + lb  (c3< 0.5)",
				"Follower: x_i = (x_i + x_(i-1)) / 2",
			},
			[]string{
				"c1 = 2*exp(-(4t/T)^2): Exploration nimmt ab.",
				"Leader orientiert sich an Food-Quelle (Bester).",
				"Follower mitteln mit Vordermann (Kettenbildung).",
				"Im Sim: Bot-Kette konvergiert schrittweise.",
			},
		},
		{
			"EO — Equilibrium Optimizer",
			[]string{
				"x(t+1) = x_eq + (x-x_eq)*F + G/lambda*(1-F)",
				"F = a*sign(r-0.5)*(e^(-lambda*t) - 1)",
				"x_eq = Mittelwert der 4 besten + zufaellig",
			},
			[]string{
				"Partikel streben zum Gleichgewicht (x_eq).",
				"F: Exponentieller Decay steuert Konvergenz.",
				"G: Generation-Rate fuer Zufallsperturbation.",
				"Im Sim: Schnelle Konvergenz mit Diversitaet.",
			},
		},
		{
			"AO — Aquila Optimizer",
			[]string{
				"Hoehenflug: x = x_best*(1-t/T) + x_mean - x_rand",
				"Konturflug: x = x_best*Levy + x_rand + (y-x)*r",
				"Sturzflug:  x = (x_best-x_mean)*a - rand + lb*rand",
			},
			[]string{
				"4 Jagdphasen des Steinadlers:",
				"1. Hoehenflug (Explore, weiter Blick)",
				"2. Konturflug (Beute lokalisieren)",
				"3. Langsamer Abstieg (Verfolgen)",
				"4. Sturzflug (finaler Angriff = Exploitation).",
			},
		},
		{
			"DA — Dragonfly Algorithm",
			[]string{
				"dX = s*S + a*A + c*C + f*F + e*E",
				"S = -sum(X - X_j), A = sum(V_j)/N",
				"C = sum(X_j)/N - X, F = X_food - X, E = X+X_enemy",
			},
			[]string{
				"5 Kraeftemischung wie bei Schwarmsimulation!",
				"S=Separation, A=Alignment, C=Cohesion,",
				"F=Nahrung (Attraktion), E=Feind (Abstossung).",
				"Im Sim: Boids-artig, aber fuer Optimierung.",
			},
		},
		{
			"Jaya — 'Sieg' (Sanskrit)",
			[]string{
				"x_new = x + r1*(x_best - |x|) - r2*(x_worst - |x|)",
				"if f(x_new) < f(x): x = x_new",
				"KEIN EINZIGER PARAMETER! Nur best und worst.",
			},
			[]string{
				"Bewegt sich ZUM Besten und WEG vom Schlechtesten.",
				"Einfachster aller Metaheuristiken.",
				"Ueberraschend effektiv trotz Simplizitaet!",
				"Im Sim: Minimaler Overhead, schnelle Iteration.",
			},
		},
	}

	// Draw math blocks in two columns
	mlY := my_
	mrY := my_
	for i, mb := range mathAlgos {
		var cx int
		var cy *int
		if i%2 == 0 {
			cx = mlx
			cy = &mlY
		} else {
			cx = mrx
			cy = &mrY
		}

		// Algo name header
		printColoredAt(screen, mb.name, cx+5, *cy, colorHelpAlgoHdr)
		*cy += lineH + 1

		// Formulas (gold)
		for _, f := range mb.formula {
			printColoredAt(screen, "  "+f, cx+5, *cy, colorHelpMath)
			*cy += lineH
		}
		*cy += 2

		// Application (green)
		for _, a := range mb.applied {
			printColoredAt(screen, "  "+a, cx+5, *cy, colorHelpApply)
			*cy += lineH
		}
		*cy += 10
	}

	// Vertical separator in math section
	finalY := mlY
	if mrY > mlY {
		finalY = mrY
	}
	vector.StrokeLine(screen, float32(midX-15), float32(mathStartY), float32(midX-15), float32(finalY), 1, colorHelpSep, false)

	// General math concepts section
	finalY += 4
	vector.StrokeLine(screen, float32(px), float32(finalY), float32(sw-px), float32(finalY), 1, colorHelpSep, false)
	finalY += 8
	conceptTitle := "GEMEINSAME MATHEMATISCHE KONZEPTE"
	conceptTitleW := len(conceptTitle) * charW
	printColoredAt(screen, conceptTitle, sw/2-conceptTitleW/2, finalY, colorHelpSection)
	finalY += lineH + 4

	// Left: Exploration vs Exploitation
	cly := finalY
	printColoredAt(screen, "EXPLORATION vs EXPLOITATION", px+5, cly, colorHelpAlgoHdr)
	cly += lineH
	helpParagraph(screen, px, &cly, []string{
		"Das zentrale Dilemma aller Optimierung!",
		"Exploration = weite Suche im gesamten Raum.",
		"Exploitation = lokale Verfeinerung nahe dem Besten.",
		"",
		"Typisches Muster: Start explorativ, Ende exploitativ.",
		"Parameter a, r1 o.ae. steuern diese Balance.",
		"Zu viel Explore = langsam, zu viel Exploit = lokales Opt.",
	})
	cly += 6
	printColoredAt(screen, "LEVY-FLUEGE (Cuckoo, FPA, HHO, AO)", px+5, cly, colorHelpAlgoHdr)
	cly += lineH
	helpParagraph(screen, px, &cly, []string{
		"Schrittlaenge folgt der Potenzverteilung: P(s) ~ s^(-beta).",
		"Viele kleine Schritte + seltene riesige Spruenge.",
		"Mathematisch optimal fuer Suche in unbekanntem Terrain!",
		"Auch in der Natur: Albatrosse, Haie, Bienen, T-Zellen.",
	})

	// Right: Convergence + No Free Lunch
	cry := finalY
	printColoredAt(screen, "KONVERGENZVERHALTEN", mrx+5, cry, colorHelpAlgoHdr)
	cry += lineH
	helpParagraph(screen, mrx, &cry, []string{
		"Alle Algorithmen konvergieren in Richtung Optimum.",
		"Schnelle Konvergenz: SA, GWO, HHO, Jaya.",
		"Breite Suche: Cuckoo, DE, BFO, DA.",
		"",
		"Fitness = Deliveries*30 + CorrectColor*50",
		"         + Pickups*15 - WrongColor*20",
	})
	cry += 6
	printColoredAt(screen, "NO FREE LUNCH THEOREM", mrx+5, cry, colorHelpAlgoHdr)
	cry += lineH
	helpParagraph(screen, mrx, &cry, []string{
		"Wolpert & Macready (1997):",
		"Kein Algorithmus ist fuer ALLE Probleme der Beste!",
		"Jeder Algo hat Staerken bei bestimmten Landschaften.",
		"Deshalb: F4 Algo-Labor -> mehrere testen -> Radar Chart!",
		"",
		"Nutze das Auto-Turnier um alle 20 zu vergleichen.",
	})

	endY := cly
	if cry > endY {
		endY = cry
	}
	// Separator between concept columns
	vector.StrokeLine(screen, float32(midX-15), float32(finalY), float32(midX-15), float32(endY), 1, colorHelpSep, false)

	endY += 10

	// Footer
	footerY := sh - 20
	footer := "H = Hilfe schliessen  |  Mausrad = Scrollen  |  F3 = Tutorial"
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
