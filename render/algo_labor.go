package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Algo-Labor panel layout constants — shared between rendering and hit test
const (
	alglabPanelW = editorPanelW // 350px wide

	// Y positions for layout elements
	alglabTitleY     = 5
	alglabBackY      = 20
	alglabSep1Y      = 40
	alglabFitHeaderY = 42
	alglabFitBtnY    = 56
	alglabFitBtnH    = 18
	alglabSep2Y      = 80
	alglabAlgoHeaderY = 82
	alglabAlgoListY  = 96
	alglabAlgoEntryH = 20
	alglabAlgoVisible = 10
	alglabFitBtnW    = 78
	alglabSpeedBtnW  = 58
)

// Computed Y positions that depend on algo list
func alglabAfterAlgoY() int { return alglabAlgoListY + alglabAlgoVisible*alglabAlgoEntryH }

// DrawAlgoLabor renders the left panel when in Algo-Labor mode (F4).
func DrawAlgoLabor(screen *ebiten.Image, ss *swarm.SwarmState) {
	sh := screen.Bounds().Dy()
	vector.DrawFilledRect(screen, 0, 0, float32(alglabPanelW), float32(sh), color.RGBA{12, 14, 24, 245}, false)

	y := alglabTitleY
	printColoredAt(screen, "ALGO-LABOR", 5, y, color.RGBA{0, 220, 255, 255})

	y = alglabBackY
	printColoredAt(screen, "F2=Swarm Lab zurueck", 5, y, color.RGBA{80, 80, 100, 255})

	// Separator
	vector.StrokeLine(screen, 0, float32(alglabSep1Y), float32(alglabPanelW), float32(alglabSep1Y), 1, color.RGBA{60, 70, 100, 180}, false)

	// Fitness landscape header + buttons
	printColoredAt(screen, "FITNESS-LANDSCHAFT", 5, alglabFitHeaderY, color.RGBA{180, 200, 255, 255})

	fitNames := [4]string{"Gauss", "Rastrigin", "Ackley", "Rosenbrock"}
	fitFuncs := [4]swarm.FitnessLandscapeType{swarm.FitGaussian, swarm.FitRastrigin, swarm.FitAckley, swarm.FitRosenbrock}
	for i, name := range fitNames {
		bx := 5 + i*(alglabFitBtnW+4)
		col := color.RGBA{50, 55, 75, 255}
		if ss.SwarmAlgo != nil && ss.SwarmAlgo.FitnessFunc == fitFuncs[i] {
			col = color.RGBA{60, 140, 220, 255}
		}
		drawSwarmButton(screen, bx, alglabFitBtnY, alglabFitBtnW, alglabFitBtnH, name, col)
	}

	// Separator
	vector.StrokeLine(screen, 0, float32(alglabSep2Y), float32(alglabPanelW), float32(alglabSep2Y), 1, color.RGBA{60, 70, 100, 180}, false)

	// Algorithms header
	printColoredAt(screen, "ALGORITHMEN", 5, alglabAlgoHeaderY, color.RGBA{180, 200, 255, 255})
	printColoredAt(screen, "(Scroll: Mausrad)", 120, alglabAlgoHeaderY, color.RGBA{80, 80, 100, 255})

	// Scrollable algorithm list
	entries := GetAlgoEntries(ss)
	startIdx := ss.AlgoLaborScrollY
	if startIdx > len(entries)-alglabAlgoVisible {
		startIdx = len(entries) - alglabAlgoVisible
	}
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(entries) && i < startIdx+alglabAlgoVisible; i++ {
		e := entries[i]
		ey := alglabAlgoListY + (i-startIdx)*alglabAlgoEntryH

		vector.DrawFilledCircle(screen, 8, float32(ey+7), 4, e.Color, false)
		printColoredAt(screen, e.Name, 18, ey+2, color.RGBA{200, 210, 230, 255})

		statusLabel := "AUS"
		statusCol := color.RGBA{120, 60, 60, 255}
		bgCol := color.RGBA{40, 40, 50, 200}
		if *e.ShowPtr {
			statusLabel = "AN"
			statusCol = color.RGBA{80, 220, 80, 255}
			bgCol = color.RGBA{30, 60, 40, 200}
		}
		vector.DrawFilledRect(screen, float32(alglabPanelW-50), float32(ey), 45, float32(alglabAlgoEntryH), bgCol, false)
		printColoredAt(screen, statusLabel, alglabPanelW-45, ey+4, statusCol)
	}

	// Scroll indicator
	if len(entries) > alglabAlgoVisible {
		scrollInfo := fmt.Sprintf("%d-%d / %d", startIdx+1, min(startIdx+alglabAlgoVisible, len(entries)), len(entries))
		printColoredAt(screen, scrollInfo, alglabPanelW-len(scrollInfo)*charW-10, alglabAfterAlgoY()+2,
			color.RGBA{100, 100, 120, 200})
	}

	// === After algo list: Vergleich, Tempo, Erklärung ===
	afterY := alglabAfterAlgoY()

	// Separator
	sepY := afterY + 6
	vector.StrokeLine(screen, 0, float32(sepY), float32(alglabPanelW), float32(sepY), 1, color.RGBA{60, 70, 100, 180}, false)

	// Comparison header + buttons
	compY := sepY + 4
	printColoredAt(screen, "VERGLEICH", 5, compY, color.RGBA{180, 200, 255, 255})
	btnY := compY + 14
	radarCol := color.RGBA{60, 80, 140, 255}
	if ss.ShowAlgoRadar {
		radarCol = color.RGBA{80, 140, 220, 255}
	}
	drawSwarmButton(screen, 5, btnY, 150, 20, "Radar-Vergleich", radarCol)
	tourneyCol := color.RGBA{60, 80, 140, 255}
	if ss.AlgoTournamentOn {
		tourneyCol = color.RGBA{200, 120, 40, 255}
	}
	drawSwarmButton(screen, 160, btnY, 150, 20, "Auto-Turnier", tourneyCol)

	// Separator
	sep2Y := btnY + 24
	vector.StrokeLine(screen, 0, float32(sep2Y), float32(alglabPanelW), float32(sep2Y), 1, color.RGBA{60, 70, 100, 180}, false)

	// Speed header + buttons
	speedHeaderY := sep2Y + 4
	printColoredAt(screen, "TEMPO", 5, speedHeaderY, color.RGBA{180, 200, 255, 255})
	speedBtnY := speedHeaderY + 14
	speeds := [5]struct {
		label string
		val   float64
	}{
		{"1x", 1.0}, {"2x", 2.0}, {"5x", 5.0}, {"10x", 10.0}, {"50x", 50.0},
	}
	for i, sp := range speeds {
		bx := 5 + i*(alglabSpeedBtnW+4)
		col := color.RGBA{50, 60, 80, 255}
		if ss.CurrentSpeed >= sp.val-0.01 && ss.CurrentSpeed <= sp.val+0.01 {
			col = color.RGBA{80, 160, 80, 255}
		}
		drawSwarmButton(screen, bx, speedBtnY, alglabSpeedBtnW, 20, sp.label, col)
	}

	// Separator
	sep3Y := speedBtnY + 24
	vector.StrokeLine(screen, 0, float32(sep3Y), float32(alglabPanelW), float32(sep3Y), 1, color.RGBA{60, 70, 100, 180}, false)

	// Dynamic explanation: show active algorithm info, or default hint
	explY := sep3Y + 4
	activeAlgos := getActiveAlgoDescriptions(ss)
	if len(activeAlgos) > 0 {
		for _, ad := range activeAlgos {
			// Algorithm name as header
			printColoredAt(screen, ad.name, 5, explY, ad.color)
			explY += lineH
			// Description lines
			for _, line := range ad.lines {
				printColoredAt(screen, line, 5, explY, color.RGBA{160, 170, 190, 255})
				explY += lineH
			}
			explY += 4 // gap between algorithms
		}
	} else {
		printColoredAt(screen, "ERKLAERUNG", 5, explY, color.RGBA{0, 160, 200, 200})
		dimGray := color.RGBA{100, 105, 120, 255}
		hints := []string{
			"Jeder Algorithmus optimiert",
			"Bot-Positionen auf der Fitness-",
			"Landschaft. Hoehere Fitness =",
			"bessere Position gefunden.",
			"",
			"Aktiviere einen Algorithmus",
			"oben, um hier seine Erklaerung",
			"zu sehen!",
			"",
			"Tipp: Starte ein Auto-Turnier",
			"fuer automatischen Vergleich.",
		}
		for i, line := range hints {
			printColoredAt(screen, line, 5, explY+14+i*lineH, dimGray)
		}
	}
}

// algoDescription holds a name + explanation lines for an active algorithm.
type algoDescription struct {
	name  string
	color color.RGBA
	lines []string
}

// getActiveAlgoDescriptions returns descriptions for all currently visible algorithms.
func getActiveAlgoDescriptions(ss *swarm.SwarmState) []algoDescription {
	entries := GetAlgoEntries(ss)
	var result []algoDescription
	for _, e := range entries {
		if *e.ShowPtr {
			desc, ok := algoExplanations[e.Name]
			if ok {
				result = append(result, algoDescription{
					name:  e.Name,
					color: e.Color,
					lines: desc,
				})
			}
		}
	}
	return result
}

// algoExplanations maps algorithm names to their educational descriptions.
var algoExplanations = map[string][]string{
	"GWO (Grey Wolf)": {
		"Graue Woelfe jagen im Rudel.",
		"Alpha = bester, Beta = zweit-",
		"bester, Delta = drittbester.",
		"Alle anderen folgen diesen drei",
		"Leitwolfen. Ueber Zeit wird der",
		"Suchradius kleiner (Einkreisen).",
		"",
		"Formel: X(t+1) = Xp - A*|C*Xp - X|",
		"A wird von 2->0 kleiner = Jagd!",
	},
	"WOA (Whale)": {
		"Buckelwale jagen mit Blasen-Netz.",
		"Phase 1: Einkreisen — Wale",
		"schwimmen zum besten Wal.",
		"Phase 2: Spirale — Wale drehen",
		"sich spiralfoermig zum Ziel.",
		"50/50 Zufall zwischen beiden.",
		"",
		"Formel: Spirale mit e^(b*l)*cos(2pi*l)",
	},
	"BFO (Bacterial)": {
		"Bakterien suchen Nahrung durch",
		"Chemotaxis: Schwimmen + Taumeln.",
		"Schwimmen = geradeaus wenn gut.",
		"Taumeln = zufaellige Richtung",
		"wenn Fitness schlechter wird.",
		"Gute Bakterien reproduzieren sich.",
	},
	"MFO (Moth-Flame)": {
		"Motten fliegen spiralfoermig",
		"um Flammen (die besten Positionen).",
		"Jede Motte hat eine Flamme als Ziel.",
		"Ueber Zeit schrumpft die Spirale.",
		"Weniger Flammen = mehr Fokus auf",
		"die besten Loesungen.",
	},
	"Cuckoo Search": {
		"Kuckucke legen Eier in fremde",
		"Nester (= zufaellige Loesungen).",
		"Levy-Fluege: grosse Spruenge",
		"fuer Exploration, kleine fuer",
		"lokale Suche. Schlechte Nester",
		"werden mit Wahrscheinlichkeit",
		"pa=0.25 durch neue ersetzt.",
	},
	"Diff. Evolution": {
		"Drei zufaellige Vektoren waehlen.",
		"Mutation: v = x1 + F*(x2-x3)",
		"Crossover: Gene mit Rate CR",
		"vom Mutant oder Original nehmen.",
		"Trial vs Original: der Bessere",
		"ueberlebt. F=0.8, CR=0.9 typisch.",
	},
	"ABC (Bee Colony)": {
		"Drei Bienenrollen:",
		"  Arbeiterinnen: lokale Suche",
		"  Zuschauer: folgen den Besten",
		"  Scouts: suchen neue Quellen",
		"Gute Futterquellen bekommen",
		"mehr Bienen zugewiesen.",
		"Erschoepfte Quellen = Neubeginn.",
	},
	"HSO (Harmony)": {
		"Wie Jazz-Improvisation!",
		"Jeder Musiker (Variable) waehlt:",
		"  1. Aus Erinnerung (HMS)",
		"  2. Leicht variiert (Pitch Adj.)",
		"  3. Komplett zufaellig",
		"Beste Harmonien bleiben erhalten.",
	},
	"Bat Algorithm": {
		"Fledermaeuse nutzen Echoortung.",
		"Laut + niedrige Frequenz = weit",
		"suchen (Exploration).",
		"Leise + hohe Frequenz = nah",
		"suchen (Exploitation).",
		"Ueber Zeit: leiser + genauer.",
	},
	"HHO (Harris Hawks)": {
		"Harris-Habichte jagen im Team.",
		"Phase 1: Erkunden — Beute suchen.",
		"Phase 2: Surprise Pounce — von",
		"mehreren Seiten gleichzeitig.",
		"Weiche/harte Belagerung je nach",
		"Energie der Beute (nimmt ab).",
	},
	"SSA (Salp Swarm)": {
		"Salpen bilden Ketten im Ozean.",
		"Der Fuehrer folgt der Nahrung.",
		"Alle anderen folgen ihrem",
		"Vordermann in der Kette.",
		"c1 nimmt ueber Zeit ab =",
		"Exploration -> Exploitation.",
	},
	"GSA (Gravitational)": {
		"Schwere Objekte (= gute Fitness)",
		"ziehen leichte Objekte an.",
		"Gravitationskraft: F = G*M1*M2/r²",
		"G nimmt ueber Zeit ab = weniger",
		"globale, mehr lokale Suche.",
		"Masse ~ Fitness der Loesung.",
	},
	"FPA (Flower)": {
		"Blumen werden bestäubt durch:",
		"  Global: Levy-Fluege (Insekten",
		"    tragen Pollen weit weg)",
		"  Lokal: Wind/Diffusion (Pollen",
		"    verbreitet sich nah)",
		"Switch-Wahrscheinlichkeit p=0.8.",
	},
	"SA (Simulated Annealing)": {
		"Metallabkuehlung simuliert.",
		"Heiss: akzeptiere auch SCHLECHTERE",
		"Loesungen (entkomme Sackgassen).",
		"Kalt: nur noch BESSERE akzeptiert.",
		"P(akzeptiere) = e^(-deltaE / T)",
		"T sinkt langsam: T = T0 * alpha^t",
	},
	"AO (Aquila)": {
		"Adler-Jagdstrategien:",
		"  1. Hoch kreisen (erkunden)",
		"  2. Sturzflug (schnell zum Ziel)",
		"  3. Niedrig gleiten (lokal suchen)",
		"  4. Fuss-Angriff (Feintuning)",
		"Phase wechselt mit Iterationen.",
	},
	"SCA (Sine Cosine)": {
		"Sinus-Cosinus Oszillation.",
		"Positionen schwingen zwischen",
		"Erkundung (grosse Amplitude)",
		"und Ausbeutung (kleine Amplitude).",
		"x(t+1) = x + r1*sin(r2)*|r3*P-x|",
		"r1 nimmt linear ab: 2 -> 0.",
	},
	"DA (Dragonfly)": {
		"Libellen-Schwarmverhalten:",
		"  Separation: Abstand halten",
		"  Ausrichtung: gleiche Richtung",
		"  Zusammenhalt: zur Mitte",
		"  Nahrung: zum Optimum hin",
		"  Feind: vom Schlechtesten weg",
		"Balanciert Exploration/Exploitation.",
	},
	"TLBO (Teaching)": {
		"Lehrer-Schueler Interaktion:",
		"  Lehrer-Phase: Bester lehrt alle,",
		"    Klasse bewegt sich zum Lehrer.",
		"  Schueler-Phase: Paare lernen",
		"    voneinander — der Bessere",
		"    beeinflusst den Anderen.",
		"Kein Parameter noetig (F, CR etc.)!",
	},
	"EO (Equilibrium)": {
		"Equilibrium Optimizer:",
		"Pool aus 4-5 besten Loesungen.",
		"Partikel bewegen sich zum Pool",
		"mit exponentiell abnehmender",
		"Rate. Generation Term fuer",
		"Exploration. Balanciert durch",
		"Konzentrations-Mechanismus.",
	},
	"Jaya": {
		"Einfachstes Prinzip ueberhaupt:",
		"\"Zum Besten hin, vom",
		"Schlechtesten weg.\"",
		"",
		"x_new = x + r1*(best-x)",
		"          - r2*(worst-x)",
		"",
		"Kein einziger Parameter noetig!",
		"Trotzdem erstaunlich effektiv.",
	},
}

// AlgoLaborHitTest returns a hit ID for clickable elements in the Algo-Labor panel.
// Uses the SAME layout constants as DrawAlgoLabor to guarantee coordinate match.
func AlgoLaborHitTest(mx, my int, ss *swarm.SwarmState) string {
	if mx < 0 || mx >= alglabPanelW+20 {
		return ""
	}

	// F2 back button area
	if my >= alglabBackY && my < alglabBackY+14 && mx >= 5 && mx < 200 {
		return "alglab:f2back"
	}

	// Fitness function buttons
	if my >= alglabFitBtnY && my < alglabFitBtnY+alglabFitBtnH {
		for i := 0; i < 4; i++ {
			bx := 5 + i*(alglabFitBtnW+4)
			if mx >= bx && mx < bx+alglabFitBtnW {
				return fmt.Sprintf("alglab:fit:%d", i)
			}
		}
	}

	// Algorithm toggles
	if my >= alglabAlgoListY && my < alglabAlgoListY+alglabAlgoVisible*alglabAlgoEntryH {
		startIdx := ss.AlgoLaborScrollY
		entries := GetAlgoEntries(ss)
		if startIdx > len(entries)-alglabAlgoVisible {
			startIdx = len(entries) - alglabAlgoVisible
		}
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(entries) && i < startIdx+alglabAlgoVisible; i++ {
			ey := alglabAlgoListY + (i-startIdx)*alglabAlgoEntryH
			if my >= ey && my < ey+alglabAlgoEntryH {
				return fmt.Sprintf("alglab:algo:%d", i)
			}
		}
	}

	// === After algo list: computed positions ===
	afterY := alglabAfterAlgoY()
	sepY := afterY + 6
	compY := sepY + 4
	btnY := compY + 14
	sep2Y := btnY + 24
	speedHeaderY := sep2Y + 4
	speedBtnY := speedHeaderY + 14

	// Radar button
	if my >= btnY && my < btnY+20 && mx >= 5 && mx < 155 {
		return "alglab:radar"
	}

	// Tournament button
	if my >= btnY && my < btnY+20 && mx >= 160 && mx < 310 {
		return "alglab:tourney"
	}

	// Speed buttons
	if my >= speedBtnY && my < speedBtnY+20 {
		speedIds := [5]string{"alglab:speed:1", "alglab:speed:2", "alglab:speed:5", "alglab:speed:10", "alglab:speed:50"}
		for i, id := range speedIds {
			bx := 5 + i*(alglabSpeedBtnW+4)
			if mx >= bx && mx < bx+alglabSpeedBtnW {
				return id
			}
		}
	}

	return ""
}
