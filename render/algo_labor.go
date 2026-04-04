package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

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
	printColoredAt(screen, locale.T("alglab.title"), 5, y, color.RGBA{0, 220, 255, 255})

	y = alglabBackY
	printColoredAt(screen, locale.T("alglab.back"), 5, y, color.RGBA{80, 80, 100, 255})

	// Separator
	vector.StrokeLine(screen, 0, float32(alglabSep1Y), float32(alglabPanelW), float32(alglabSep1Y), 1, color.RGBA{60, 70, 100, 180}, false)

	// Fitness landscape header + buttons
	printColoredAt(screen, locale.T("alglab.fitness"), 5, alglabFitHeaderY, ColorHeaderBlue)

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
	printColoredAt(screen, locale.T("alglab.algorithms"), 5, alglabAlgoHeaderY, ColorHeaderBlue)
	printColoredAt(screen, locale.T("alglab.scroll_hint"), 120, alglabAlgoHeaderY, color.RGBA{80, 80, 100, 255})

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
		printColoredAt(screen, e.Name, 18, ey+2, ColorTextLight)

		statusLabel := locale.T("toggle.off")
		statusCol := color.RGBA{120, 60, 60, 255}
		bgCol := color.RGBA{40, 40, 50, 200}
		if *e.ShowPtr {
			statusLabel = locale.T("toggle.on")
			statusCol = color.RGBA{80, 220, 80, 255}
			bgCol = color.RGBA{30, 60, 40, 200}
		}
		vector.DrawFilledRect(screen, float32(alglabPanelW-50), float32(ey), 45, float32(alglabAlgoEntryH), bgCol, false)
		printColoredAt(screen, statusLabel, alglabPanelW-45, ey+4, statusCol)
	}

	// Scroll indicator
	if len(entries) > alglabAlgoVisible {
		scrollInfo := fmt.Sprintf("%d-%d / %d", startIdx+1, min(startIdx+alglabAlgoVisible, len(entries)), len(entries))
		printColoredAt(screen, scrollInfo, alglabPanelW-runeLen(scrollInfo)*charW-10, alglabAfterAlgoY()+2,
			color.RGBA{100, 100, 120, 200})
	}

	// === After algo list: Vergleich, Tempo, Erklärung ===
	afterY := alglabAfterAlgoY()

	// Separator
	sepY := afterY + 6
	vector.StrokeLine(screen, 0, float32(sepY), float32(alglabPanelW), float32(sepY), 1, color.RGBA{60, 70, 100, 180}, false)

	// Comparison header + buttons
	compY := sepY + 4
	printColoredAt(screen, locale.T("alglab.compare"), 5, compY, ColorHeaderBlue)
	btnY := compY + 14
	radarCol := color.RGBA{60, 80, 140, 255}
	if ss.ShowAlgoRadar {
		radarCol = ColorToggleBlue
	}
	drawSwarmButton(screen, 5, btnY, 150, 20, locale.T("alglab.radar"), radarCol)
	tourneyCol := color.RGBA{60, 80, 140, 255}
	if ss.AlgoTournamentOn {
		tourneyCol = color.RGBA{200, 120, 40, 255}
	}
	drawSwarmButton(screen, 160, btnY, 150, 20, locale.T("alglab.tourney"), tourneyCol)

	// Separator
	sep2Y := btnY + 24
	vector.StrokeLine(screen, 0, float32(sep2Y), float32(alglabPanelW), float32(sep2Y), 1, color.RGBA{60, 70, 100, 180}, false)

	// Speed header + buttons
	speedHeaderY := sep2Y + 4
	printColoredAt(screen, locale.T("alglab.speed"), 5, speedHeaderY, ColorHeaderBlue)
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

	// K key hint for math trace
	mathHintY := sep3Y + 4
	printColoredAt(screen, locale.T("alglab.mathtrace_hint"), 5, mathHintY, color.RGBA{100, 120, 160, 200})

	// Dynamic explanation: show active algorithm info, or default hint
	explY := mathHintY + lineH + 2
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
		printColoredAt(screen, locale.T("alglab.explain"), 5, explY, color.RGBA{0, 160, 200, 200})
		dimGray := color.RGBA{100, 105, 120, 255}
		hints := []string{
			locale.T("alglab.hint.1"),
			locale.T("alglab.hint.2"),
			locale.T("alglab.hint.3"),
			locale.T("alglab.hint.4"),
			locale.T("alglab.hint.5"),
			locale.T("alglab.hint.6"),
			locale.T("alglab.hint.7"),
			"",
			locale.T("alglab.hint.8"),
			locale.T("alglab.hint.9"),
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
	explanations := algoExplanations()
	entries := GetAlgoEntries(ss)
	var result []algoDescription
	for _, e := range entries {
		if *e.ShowPtr {
			desc, ok := explanations[e.Name]
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

// algoExplanationLineCounts maps algorithm names to their number of locale description lines.
var algoExplanationLineCounts = map[string]int{
	"GWO (Grey Wolf)":      9,
	"WOA (Whale)":          8,
	"BFO (Bacterial)":      6,
	"MFO (Moth-Flame)":     6,
	"Cuckoo Search":        7,
	"Diff. Evolution":      6,
	"ABC (Bee Colony)":     7,
	"HSO (Harmony)":        6,
	"Bat Algorithm":        6,
	"HHO (Harris Hawks)":   6,
	"SSA (Salp Swarm)":     6,
	"GSA (Gravitational)":  6,
	"FPA (Flower)":         6,
	"SA (Sim. Annealing)":  6,
	"AO (Aquila)":          6,
	"SCA (Sine Cosine)":    6,
	"DA (Dragonfly)":       7,
	"TLBO (Teaching)":      7,
	"EO (Equilibrium)":     7,
	"Jaya":                 9,
}

var (
	cachedAlgoExplanations map[string][]string
	cachedAlgoExplLang     locale.Lang
)

// algoExplanations builds the algorithm explanation map from locale at runtime.
// Results are cached and only rebuilt when the active language changes.
func algoExplanations() map[string][]string {
	if cachedAlgoExplanations != nil && cachedAlgoExplLang == locale.GetLang() {
		return cachedAlgoExplanations
	}
	m := make(map[string][]string, len(algoExplanationLineCounts))
	for name, count := range algoExplanationLineCounts {
		lines := make([]string, count)
		for i := 0; i < count; i++ {
			lines[i] = locale.T(fmt.Sprintf("algoexpl.%s.%d", name, i))
		}
		m[name] = lines
	}
	cachedAlgoExplanations = m
	cachedAlgoExplLang = locale.GetLang()
	return m
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

	// Tab bar (language button, tab buttons) — shared with swarm editor
	if tabHit := TabBarHitTest(mx, my); tabHit != "" {
		return tabHit
	}

	return ""
}
