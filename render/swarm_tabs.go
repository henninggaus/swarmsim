package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Tab system layout constants
const (
	tabBarY      = 640 // Y position of tab bar
	tabBarH      = 20  // tab button height
	tabContentY  = 662 // Y where tab content starts
	tabContentH  = 230 // available height for tab content
	tabBtnW      = 66  // width per tab button
	tabBtnGap    = 2   // gap between tab buttons

	// Toggle grid inside tabs
	tabToggleW = 165
	tabToggleH = 20
	tabToggleGap = 4
	tabPadX    = 5
)

// Tab names
var tabNames = [4]string{"Arena", "Evo", "Anzeige", "Werkzeuge"}

// Tab colors (active vs inactive)
var (
	tabActiveColor   = color.RGBA{50, 70, 110, 255}
	tabInactiveColor = color.RGBA{25, 30, 45, 200}
	tabTextActive    = color.RGBA{220, 230, 255, 255}
	tabTextInactive  = color.RGBA{120, 130, 150, 255}
	tabBorderColor   = color.RGBA{80, 100, 160, 180}
)

// drawTabBar renders the 4 tab buttons.
func drawTabBar(screen *ebiten.Image, ss *swarm.SwarmState) {
	// Background strip
	vector.DrawFilledRect(screen, 0, float32(tabBarY), float32(editorPanelW), float32(tabBarH), color.RGBA{15, 18, 30, 255}, false)

	for i, name := range tabNames {
		x := tabPadX + i*(tabBtnW+tabBtnGap)
		bgCol := tabInactiveColor
		textCol := tabTextInactive
		if ss.EditorTab == i {
			bgCol = tabActiveColor
			textCol = tabTextActive
		}
		vector.DrawFilledRect(screen, float32(x), float32(tabBarY), float32(tabBtnW), float32(tabBarH), bgCol, false)
		if ss.EditorTab == i {
			// Active tab: bright top border
			vector.StrokeLine(screen, float32(x), float32(tabBarY), float32(x+tabBtnW), float32(tabBarY), 2, tabBorderColor, false)
		}
		// Center text
		textW := len(name) * charW
		tx := x + (tabBtnW-textW)/2
		printColoredAt(screen, name, tx, tabBarY+4, textCol)
	}
}

// drawTabContent renders the content for the active tab.
func drawTabContent(screen *ebiten.Image, ss *swarm.SwarmState) {
	// Content background
	vector.DrawFilledRect(screen, 0, float32(tabContentY), float32(editorPanelW), float32(tabContentH),
		color.RGBA{18, 20, 32, 240}, false)

	switch ss.EditorTab {
	case 0:
		drawTabArena(screen, ss)
	case 1:
		drawTabEvolution(screen, ss)
	case 2:
		drawTabDisplay(screen, ss)
	case 3:
		drawTabTools(screen, ss)
	}
}

// tabToggle is a helper to draw a toggle button in the tab content area.
type tabToggle struct {
	label    string
	on       bool
	onColor  color.RGBA
	disabled bool // greyed out, not clickable
}

func drawTabToggles(screen *ebiten.Image, toggles []tabToggle, startY int) {
	for i, t := range toggles {
		col := 0
		if i%2 == 1 {
			col = 1
		}
		row := i / 2
		x := tabPadX + col*(tabToggleW+5)
		y := startY + row*(tabToggleH+tabToggleGap)

		bgCol := ColorSwarmBtnToggleOff
		if t.disabled {
			bgCol = color.RGBA{25, 25, 35, 200} // very dark = disabled
		} else if t.on {
			bgCol = t.onColor
		}
		drawSwarmButton(screen, x, y, tabToggleW, tabToggleH, t.label, bgCol)
	}
}

// ==================== Tab 0: Arena ====================

func drawTabArena(screen *ebiten.Image, ss *swarm.SwarmState) {
	y := tabContentY + 2

	toggles := []tabToggle{
		{fmtToggle("Hindernisse", ss.ObstaclesOn), ss.ObstaclesOn, ColorSwarmBtnToggleOn, false},
		{fmtToggle("Labyrinth", ss.MazeOn), ss.MazeOn, ColorSwarmBtnToggleOn, false},
		{fmtToggle("Lichtquelle", ss.Light.Active), ss.Light.Active, ColorSwarmBtnToggleOn, false},
		{fmtWallToggle(ss.WrapMode), ss.WrapMode, color.RGBA{60, 60, 140, 255}, false},
		{fmtDelivToggle(ss), ss.DeliveryOn, color.RGBA{200, 120, 40, 255}, false},
		{fmtTruckToggle(ss), ss.TruckToggle, color.RGBA{180, 100, 40, 255}, false},
		{fmtToggle("Energie", ss.EnergyEnabled), ss.EnergyEnabled, color.RGBA{220, 180, 50, 255}, false},
		{fmtToggle("Dynamisch", ss.DynamicEnv), ss.DynamicEnv, color.RGBA{255, 150, 50, 255}, false},
		{fmtToggle("Tag/Nacht", ss.DayNightOn), ss.DayNightOn, color.RGBA{100, 100, 200, 255}, false},
		{fmtToggle("Arena-Editor", ss.ArenaEditMode), ss.ArenaEditMode, color.RGBA{180, 180, 80, 255}, false},
	}
	drawTabToggles(screen, toggles, y)
}

// ==================== Tab 1: Evolution ====================

func drawTabEvolution(screen *ebiten.Image, ss *swarm.SwarmState) {
	y := tabContentY + 2

	// Mutual exclusion: Evolution, GP, Neuro can't run simultaneously
	evoBlocked := ss.GPEnabled || ss.NeuroEnabled
	gpBlocked := ss.EvolutionOn || ss.NeuroEnabled
	neuroBlocked := ss.EvolutionOn || ss.GPEnabled
	// Pareto & Artbildung need parametric Evolution
	paretoBlocked := !ss.EvolutionOn && !ss.GPEnabled
	specBlocked := !ss.EvolutionOn

	evoLabel := fmtToggle("Evolution", ss.EvolutionOn)
	gpLabel := fmtToggle("GP", ss.GPEnabled)
	neuroLabel := fmtToggle("Neuro", ss.NeuroEnabled)
	if evoBlocked && !ss.EvolutionOn {
		evoLabel = "Evolution: ---"
	}
	if gpBlocked && !ss.GPEnabled {
		gpLabel = "GP: ---"
	}
	if neuroBlocked && !ss.NeuroEnabled {
		neuroLabel = "Neuro: ---"
	}

	toggles := []tabToggle{
		{evoLabel, ss.EvolutionOn, color.RGBA{180, 50, 180, 255}, evoBlocked && !ss.EvolutionOn},
		{gpLabel, ss.GPEnabled, color.RGBA{0, 180, 160, 255}, gpBlocked && !ss.GPEnabled},
		{neuroLabel, ss.NeuroEnabled, color.RGBA{255, 140, 50, 255}, neuroBlocked && !ss.NeuroEnabled},
		{fmtToggle("Teams", ss.TeamsEnabled), ss.TeamsEnabled, color.RGBA{100, 100, 255, 255}, false},
		{fmtToggle("Pareto", ss.ParetoEnabled), ss.ParetoEnabled, color.RGBA{200, 100, 200, 255}, paretoBlocked},
		{fmtToggle("Artbildung", ss.SpeciationOn), ss.SpeciationOn, color.RGBA{160, 200, 100, 255}, specBlocked},
		{fmtToggle("Sensorrauschen", ss.SensorNoiseOn), ss.SensorNoiseOn, color.RGBA{255, 120, 80, 255}, false},
		{fmtToggle("Gedaechtnis", ss.MemoryEnabled), ss.MemoryEnabled, color.RGBA{200, 150, 255, 255}, false},
		{fmtToggle("Bestenliste", ss.ShowLeaderboard), ss.ShowLeaderboard, color.RGBA{255, 200, 80, 255}, false},
	}
	drawTabToggles(screen, toggles, y)

	// Hint: explain why buttons are disabled
	if ss.NeuroEnabled {
		hintY := y + 5*(tabToggleH+tabToggleGap) + 2
		printColoredAt(screen, "Neuro aktiv: Evolution/GP gesperrt", 5, hintY, color.RGBA{255, 140, 50, 150})
	} else if ss.GPEnabled {
		hintY := y + 5*(tabToggleH+tabToggleGap) + 2
		printColoredAt(screen, "GP aktiv: Evolution/Neuro gesperrt", 5, hintY, color.RGBA{0, 180, 160, 150})
	} else if ss.EvolutionOn {
		hintY := y + 5*(tabToggleH+tabToggleGap) + 2
		printColoredAt(screen, "Evolution aktiv: GP/Neuro gesperrt", 5, hintY, color.RGBA{180, 50, 180, 150})
	}

	// Evo status line at bottom
	statusY := tabContentY + tabContentH - lineH - 2
	if ss.GPEnabled {
		paretoTag := ""
		if ss.ParetoEnabled {
			paretoTag = " [PARETO]"
		}
		gpInfo := fmt.Sprintf("GP Gen:%d Best:%.0f Avg:%.0f%s",
			ss.GPGeneration, ss.BestFitness, ss.AvgFitness, paretoTag)
		printColoredAt(screen, gpInfo, 5, statusY, color.RGBA{0, 180, 160, 200})
	} else if ss.NeuroEnabled {
		neuroInfo := fmt.Sprintf("Neuro Gen:%d Best:%.0f Avg:%.0f",
			ss.NeuroGeneration, ss.BestFitness, ss.AvgFitness)
		printColoredAt(screen, neuroInfo, 5, statusY, color.RGBA{255, 140, 50, 200})
	} else if ss.EvolutionOn {
		evoInfo := fmt.Sprintf("Evo Gen:%d Best:%.0f Avg:%.0f",
			ss.Generation, ss.BestFitness, ss.AvgFitness)
		printColoredAt(screen, evoInfo, 5, statusY, color.RGBA{180, 50, 180, 200})
	}
}

// ==================== Tab 2: Anzeige ====================

func drawTabDisplay(screen *ebiten.Image, ss *swarm.SwarmState) {
	y := tabContentY + 2

	// Genom-Balken/Liste only make sense with parametric Evolution (not Neuro/GP)
	genomDisabled := ss.NeuroEnabled || ss.GPEnabled

	toggles := []tabToggle{
		{fmtToggle("Dashboard", ss.DashboardOn), ss.DashboardOn, color.RGBA{80, 140, 220, 255}, false},
		{fmtToggle("Minimap", ss.ShowMinimap), ss.ShowMinimap, color.RGBA{80, 140, 220, 255}, false},
		{fmtToggle("Spuren", ss.ShowTrails), ss.ShowTrails, color.RGBA{80, 140, 220, 255}, false},
		{fmtToggle("Heatmap", ss.ShowHeatmap), ss.ShowHeatmap, color.RGBA{200, 80, 80, 255}, false},
		{fmtToggle("Lieferwege", ss.ShowRoutes), ss.ShowRoutes, color.RGBA{80, 140, 220, 255}, false},
		{fmtToggle("Live-Diagramm", ss.ShowLiveChart), ss.ShowLiveChart, color.RGBA{80, 200, 80, 255}, false},
		{fmtToggle("Kom.-Graph", ss.ShowCommGraph), ss.ShowCommGraph, color.RGBA{100, 200, 255, 255}, false},
		{fmtToggle("Broadcast", ss.ShowMsgWaves), ss.ShowMsgWaves, color.RGBA{100, 200, 255, 255}, false},
		{fmtToggle("Genom-Balken", ss.ShowGenomeViz), ss.ShowGenomeViz, color.RGBA{200, 160, 255, 255}, genomDisabled},
		{fmtToggle("Genom-Liste", ss.GenomeBrowserOn), ss.GenomeBrowserOn, color.RGBA{200, 160, 255, 255}, genomDisabled},
		{fmtToggle("Schwarm-Mitte", ss.ShowSwarmCenter), ss.ShowSwarmCenter, color.RGBA{255, 255, 100, 255}, false},
		{fmtToggle("Stau-Zonen", ss.ShowZones), ss.ShowZones, color.RGBA{255, 120, 80, 255}, false},
		{fmtToggle("Vorhersage", ss.ShowPrediction), ss.ShowPrediction, color.RGBA{150, 200, 255, 255}, false},
		{fmtColorFilter(ss.ColorFilter), ss.ColorFilter > 0, color.RGBA{200, 200, 80, 255}, false},
	}
	drawTabToggles(screen, toggles, y)
}

// ==================== Algo entries (used by F4 Algo-Labor) ====================

// AlgoEntry defines one algorithm for the scrollable list.
type AlgoEntry struct {
	Name    string
	OnPtr   *bool      // pointer to the On flag
	ShowPtr *bool      // pointer to Show flag
	Color   color.RGBA
	Init    func(*swarm.SwarmState) // init function (called when toggling ON)
}

// GetAlgoEntries returns the list of all optimization algorithms with their state pointers.
func GetAlgoEntries(ss *swarm.SwarmState) []AlgoEntry {
	return []AlgoEntry{
		{"GWO (Grey Wolf)", &ss.GWOOn, &ss.ShowGWO, color.RGBA{120, 120, 120, 255}, swarm.InitGWO},
		{"WOA (Whale)", &ss.WOAOn, &ss.ShowWOA, color.RGBA{60, 120, 200, 255}, swarm.InitWOA},
		{"BFO (Bacterial)", &ss.BFOOn, &ss.ShowBFO, color.RGBA{80, 200, 80, 255}, swarm.InitBFO},
		{"MFO (Moth-Flame)", &ss.MFOOn, &ss.ShowMFO, color.RGBA{255, 160, 50, 255}, swarm.InitMFO},
		{"Cuckoo Search", &ss.CuckooOn, &ss.ShowCuckoo, color.RGBA{140, 100, 200, 255}, swarm.InitCuckoo},
		{"Diff. Evolution", &ss.DEOn, &ss.ShowDE, color.RGBA{200, 80, 80, 255}, swarm.InitDE},
		{"ABC (Bee Colony)", &ss.ABCOn, &ss.ShowABC, color.RGBA{255, 200, 50, 255}, swarm.InitABC},
		{"HSO (Harmony)", &ss.HSOOn, &ss.ShowHSO, color.RGBA{100, 200, 200, 255}, swarm.InitHSO},
		{"Bat Algorithm", &ss.BatOn, &ss.ShowBat, color.RGBA{80, 80, 160, 255}, swarm.InitBat},
		{"HHO (Harris Hawks)", &ss.HHOOn, &ss.ShowHHO, color.RGBA{180, 100, 50, 255}, swarm.InitHHO},
		{"SSA (Salp Swarm)", &ss.SSAOn, &ss.ShowSSA, color.RGBA{100, 180, 100, 255}, swarm.InitSSA},
		{"GSA (Gravitational)", &ss.GSAOn, &ss.ShowGSA, color.RGBA{160, 160, 200, 255}, swarm.InitGSA},
		{"FPA (Flower)", &ss.FPAOn, &ss.ShowFPA, color.RGBA{255, 120, 180, 255}, swarm.InitFPA},
		{"SA (Sim. Annealing)", &ss.SAOn, &ss.ShowSA, color.RGBA{255, 100, 100, 255}, swarm.InitSA},
		{"AO (Aquila)", &ss.AOOn, &ss.ShowAO, color.RGBA{140, 100, 60, 255}, swarm.InitAO},
		{"SCA (Sine Cosine)", &ss.SCAOn, &ss.ShowSCA, color.RGBA{100, 200, 255, 255}, swarm.InitSCA},
		{"DA (Dragonfly)", &ss.DAOn, &ss.ShowDA, color.RGBA{80, 180, 120, 255}, swarm.InitDA},
		{"TLBO (Teaching)", &ss.TLBOOn, &ss.ShowTLBO, color.RGBA{200, 180, 100, 255}, swarm.InitTLBO},
		{"EO (Equilibrium)", &ss.EOOn, &ss.ShowEO, color.RGBA{120, 120, 200, 255}, swarm.InitEO},
		{"Jaya", &ss.JayaOn, &ss.ShowJaya, color.RGBA{200, 200, 100, 255}, swarm.InitJaya},
	}
}

// ==================== Tab 3: Werkzeuge/Tools ====================

func drawTabTools(screen *ebiten.Image, ss *swarm.SwarmState) {
	y := tabContentY + 2
	dimCol := color.RGBA{140, 150, 170, 255}

	// Speed section
	printColoredAt(screen, "Tempo:", tabPadX, y+2, dimCol)
	speeds := [5]struct {
		label string
		val   float64
	}{
		{"0.5x", 0.5}, {"1x", 1.0}, {"2x", 2.0}, {"5x", 5.0}, {"10x", 10.0},
	}
	for i, sp := range speeds {
		bx := 50 + i*58
		col := color.RGBA{50, 60, 80, 255}
		// Highlight active speed
		if ss.CurrentSpeed > 0 && ss.CurrentSpeed >= sp.val-0.01 && ss.CurrentSpeed <= sp.val+0.01 {
			col = color.RGBA{80, 160, 80, 255}
		}
		drawSwarmButton(screen, bx, y, 54, 18, sp.label, col)
	}
	y += 24

	// Replay
	toggles := []tabToggle{
		{fmtToggle("Zeitreise", ss.ReplayMode), ss.ReplayMode, color.RGBA{200, 100, 100, 255}, false},
		{fmtToggle("Turnier", ss.TournamentOn), ss.TournamentOn, color.RGBA{200, 160, 80, 255}, false},
	}
	drawTabToggles(screen, toggles, y)
	y += (tabToggleH + tabToggleGap) + 6

	// Action buttons
	printColoredAt(screen, "Aktionen:", tabPadX, y+2, dimCol)
	y += lineH + 2

	actionCol := color.RGBA{60, 80, 120, 255}
	drawSwarmButton(screen, tabPadX, y, 110, 20, "Neue Runde", actionCol)
	drawSwarmButton(screen, tabPadX+115, y, 110, 20, "Bildschirmfoto", actionCol)
	y += 24
	drawSwarmButton(screen, tabPadX, y, 110, 20, "GIF aufnehmen", actionCol)
	drawSwarmButton(screen, tabPadX+115, y, 110, 20, "CSV Export", actionCol)
	y += 24
	drawSwarmButton(screen, tabPadX, y, 110, 20, "Zustand speichern", actionCol)
	drawSwarmButton(screen, tabPadX+115, y, 110, 20, "Zustand laden", actionCol)
}

// ==================== Helpers ====================

func fmtToggle(name string, on bool) string {
	if on {
		return name + ": AN"
	}
	return name + ": AUS"
}

func fmtWallToggle(wrap bool) string {
	if wrap {
		return "Rand: Durchlauf"
	}
	return "Rand: Abprall"
}

func fmtDelivToggle(ss *swarm.SwarmState) string {
	if ss.DeliveryOn {
		return "Lieferung: AN"
	}
	return "Lieferung: AUS"
}

func fmtTruckToggle(ss *swarm.SwarmState) string {
	if ss.TruckToggle {
		return "LKW: AN"
	}
	return "LKW: AUS"
}

func fmtColorFilter(filter int) string {
	names := []string{"Farbfilter: AUS", "Filter: Rot", "Filter: Gruen", "Filter: Blau", "Filter: Traegt", "Filter: Wartend"}
	if filter >= 0 && filter < len(names) {
		return names[filter]
	}
	return "Farbfilter: AUS"
}

// drawCompactStats renders a single-line stats summary at the bottom.
func drawCompactStats(screen *ebiten.Image, ss *swarm.SwarmState) {
	y := tabContentY + tabContentH + 2
	dimCol := color.RGBA{120, 130, 150, 200}

	if ss.DeliveryOn {
		ds := &ss.DeliveryStats
		rate := 0
		if ds.TotalDelivered > 0 {
			rate = ds.CorrectDelivered * 100 / ds.TotalDelivered
		}
		carrying := 0
		for i := range ss.Bots {
			if ss.Bots[i].CarryingPkg >= 0 {
				carrying++
			}
		}
		stats := fmt.Sprintf("Del:%d (%d%%) Carry:%d", ds.TotalDelivered, rate, carrying)
		printColoredAt(screen, stats, 5, y, dimCol)
	} else {
		totalNeighbors := 0
		for i := range ss.Bots {
			totalNeighbors += ss.Bots[i].NeighborCount
		}
		avg := 0.0
		if ss.BotCount > 0 {
			avg = float64(totalNeighbors) / float64(ss.BotCount)
		}
		stats := fmt.Sprintf("Bots:%d AvgN:%.1f", ss.BotCount, avg)
		printColoredAt(screen, stats, 5, y, dimCol)
	}
}

// TabBarHitTest returns "tab:N" if the click is on a tab button, or "".
func TabBarHitTest(mx, my int) string {
	if my < tabBarY || my >= tabBarY+tabBarH {
		return ""
	}
	for i := range tabNames {
		x := tabPadX + i*(tabBtnW+tabBtnGap)
		if mx >= x && mx < x+tabBtnW {
			return fmt.Sprintf("tab:%d", i)
		}
	}
	return ""
}

// TabContentHitTest returns a hit ID for clickable elements inside the active tab.
func TabContentHitTest(mx, my int, ss *swarm.SwarmState) string {
	if my < tabContentY || my >= tabContentY+tabContentH {
		return ""
	}

	switch ss.EditorTab {
	case 0:
		return tabArenaHitTest(mx, my)
	case 1:
		return tabEvoHitTest(mx, my)
	case 2:
		return tabDisplayHitTest(mx, my)
	case 3:
		return tabToolsHitTest(mx, my)
	}
	return ""
}

func toggleHitAt(mx, my, startY, index int) bool {
	col := index % 2
	row := index / 2
	x := tabPadX + col*(tabToggleW+5)
	y := startY + row*(tabToggleH+tabToggleGap)
	return mx >= x && mx < x+tabToggleW && my >= y && my < y+tabToggleH
}

func tabArenaHitTest(mx, my int) string {
	y := tabContentY + 2

	ids := []string{"obstacles", "maze", "light", "walls", "delivery", "trucks",
		"energy", "dynamicenv", "daynight", "arenaeditor"}
	for i, id := range ids {
		if toggleHitAt(mx, my, y, i) {
			return id
		}
	}
	return ""
}

func tabEvoHitTest(mx, my int) string {
	y := tabContentY + 2
	ids := []string{"evolution", "gp", "neuro", "teams", "pareto",
		"speciation", "sensornoise", "memory", "leaderboard"}
	for i, id := range ids {
		if toggleHitAt(mx, my, y, i) {
			return id
		}
	}
	return ""
}

func tabDisplayHitTest(mx, my int) string {
	y := tabContentY + 2
	ids := []string{"dashboard", "minimap", "trails", "heatmap", "routes",
		"livechart", "commgraph", "msgwaves", "genomeviz", "genomebrowser",
		"swarmcenter", "congestion", "prediction", "colorfilter"}
	for i, id := range ids {
		if toggleHitAt(mx, my, y, i) {
			return id
		}
	}
	return ""
}

func tabToolsHitTest(mx, my int) string {
	y := tabContentY + 2

	// Speed buttons
	speeds := [5]string{"speed:0.5", "speed:1", "speed:2", "speed:5", "speed:10"}
	for i, id := range speeds {
		bx := 50 + i*58
		if mx >= bx && mx < bx+54 && my >= y && my < y+18 {
			return id
		}
	}
	y += 24

	// Replay/Turnier toggles
	toolIds := []string{"replay", "tournament"}
	for i, id := range toolIds {
		if toggleHitAt(mx, my, y, i) {
			return id
		}
	}
	y += (tabToggleH + tabToggleGap) + 6 + lineH + 2

	// Action buttons (2 columns x 3 rows)
	actionIds := []string{"newround", "screenshot", "gif", "exportcsv", "exportswarm", "importswarm"}
	for i, id := range actionIds {
		col := i % 2
		row := i / 2
		bx := tabPadX + col*115
		by := y + row*24
		if mx >= bx && mx < bx+110 && my >= by && my < by+20 {
			return id
		}
	}
	return ""
}

// min is defined in swarm_editor.go
