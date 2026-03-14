package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"swarmsim/domain/bot"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"swarmsim/engine/swarmscript"
	"swarmsim/render"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenW = 1280
	screenH = 900
)

// Game implements the ebiten.Game interface.
type Game struct {
	sim      *simulation.Simulation
	renderer *render.Renderer
	camera   *render.Camera

	scenarios []simulation.Scenario

	// Camera panning
	dragging   bool
	dragStartX int
	dragStartY int
	camStartX  float64
	camStartY  float64

	// Tick accumulator for fixed timestep
	tickAcc float64
}

// NewGame creates a new game instance.
func NewGame() *Game {
	cfg := simulation.DefaultConfig()
	s := simulation.NewSimulation(cfg)
	cam := render.NewCamera(cfg.ArenaWidth, cfg.ArenaHeight)
	r := render.NewRenderer(cam)
	return &Game{
		sim:       s,
		renderer:  r,
		camera:    cam,
		scenarios: simulation.GetScenarios(),
	}
}

// Update handles input and advances simulation.
func (g *Game) Update() error {
	// ESC to quit
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	// Global keys: SPACE, +/-, F1-F7 work in all modes
	g.handleGlobalInput()

	if g.sim.SwarmMode && g.sim.SwarmState != nil {
		g.handleSwarmInput()
	} else {
		g.handleInput()
		g.handleCamera()
	}

	// Scenario title timer
	if g.sim.ScenarioTimer > 0 {
		g.sim.ScenarioTimer--
	}

	// Fixed timestep for simulation
	dt := 1.0 / 60.0 * g.sim.Speed
	g.tickAcc += dt
	tickInterval := 1.0 / float64(g.sim.Cfg.TickRate)
	for g.tickAcc >= tickInterval {
		g.sim.Update()
		g.tickAcc -= tickInterval
	}

	return nil
}

// handleGlobalInput handles keys that work in ALL modes (including swarm).
func (g *Game) handleGlobalInput() {
	// Space: pause/resume
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.sim.Paused = !g.sim.Paused
	}

	// +/-: speed
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		g.sim.Speed += 0.5
		if g.sim.Speed > 5.0 {
			g.sim.Speed = 5.0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		// In swarm mode, don't consume minus if editor is focused (for typing)
		if !(g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.Editor != nil && g.sim.SwarmState.Editor.Focused) {
			g.sim.Speed -= 0.5
			if g.sim.Speed < 0.5 {
				g.sim.Speed = 0.5
			}
		}
	}

	// F1-F5: load scenarios
	scenarioKeys := []ebiten.Key{ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5}
	for i, key := range scenarioKeys {
		if inpututil.IsKeyJustPressed(key) && i < len(g.scenarios) {
			fmt.Printf("[KEY] F%d pressed -> Loading scenario: %s\n", i+1, g.scenarios[i].Name)
			g.sim.LoadScenario(g.scenarios[i])
			g.camera.X = g.sim.Cfg.ArenaWidth / 2
			g.camera.Y = g.sim.Cfg.ArenaHeight / 2
			g.camera.Zoom = 0.7
			g.tickAcc = 0
		}
	}

	// F6: load truck scenario
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		fmt.Println("[KEY] F6 pressed -> Loading truck scenario: LKW-ENTLADUNG")
		g.sim.LoadTruckScenario()
		g.camera.X = g.sim.Cfg.ArenaWidth / 2
		g.camera.Y = g.sim.Cfg.ArenaHeight / 2
		g.camera.Zoom = 0.7
		g.tickAcc = 0
	}

	// F7: load swarm scenario
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		fmt.Println("[KEY] F7 pressed -> Loading swarm scenario: PROGRAMMABLE SWARM")
		g.sim.LoadSwarmScenario()
		g.tickAcc = 0
	}
}

func (g *Game) handleInput() {
	mx, my := ebiten.CursorPosition()
	wx, wy := g.camera.ScreenToWorld(float64(mx), float64(my), screenW, screenH)

	// Left click: select bot
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		bot := g.sim.FindBotAt(wx, wy, 20)
		if bot != nil {
			g.sim.SelectedBotID = bot.ID()
		} else {
			g.sim.SelectedBotID = -1
		}
	}

	// Number keys: spawn bots
	spawnKeys := []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5}
	types := []bot.BotType{bot.TypeScout, bot.TypeWorker, bot.TypeLeader, bot.TypeTank, bot.TypeHealer}
	for i, key := range spawnKeys {
		if inpututil.IsKeyJustPressed(key) {
			g.sim.SpawnBot(types[i], wx, wy)
		}
	}

	// R: spawn resource
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.sim.SpawnResourceAt(wx, wy)
	}

	// H: add obstacle
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.sim.AddObstacleAt(wx, wy)
	}

	// F: toggle comm radius
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		g.sim.ShowCommRadius = !g.sim.ShowCommRadius
		fmt.Printf("[KEY] F pressed -> ShowCommRadius=%v\n", g.sim.ShowCommRadius)
	}

	// G: toggle sensor radius
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.sim.ShowSensorRadius = !g.sim.ShowSensorRadius
		fmt.Printf("[KEY] G pressed -> ShowSensorRadius=%v\n", g.sim.ShowSensorRadius)
	}

	// D: toggle debug comm lines
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.sim.ShowDebugComm = !g.sim.ShowDebugComm
		fmt.Printf("[KEY] D pressed -> ShowDebugComm=%v\n", g.sim.ShowDebugComm)
	}

	// P: cycle pheromone visualization (OFF -> FOUND -> ALL -> OFF)
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.sim.PheromoneVizMode = (g.sim.PheromoneVizMode + 1) % 3
		modes := []string{"OFF", "FOUND", "ALL"}
		fmt.Printf("[KEY] P pressed -> PheromoneVizMode=%s (%d)\n", modes[g.sim.PheromoneVizMode], g.sim.PheromoneVizMode)
	}

	// E: force end generation (evolve)
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.sim.ForceEndGeneration()
		fmt.Printf("[KEY] E pressed -> Generation=%d Best=%.1f Avg=%.1f\n", g.sim.Generation, g.sim.BestFitness, g.sim.AvgFitness)
	}

	// V: toggle genome overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		g.sim.ShowGenomeOverlay = !g.sim.ShowGenomeOverlay
		fmt.Printf("[KEY] V pressed -> ShowGenomeOverlay=%v (SelectedBot=%d)\n", g.sim.ShowGenomeOverlay, g.sim.SelectedBotID)
	}

	// N: regenerate truck (only in truck mode)
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && g.sim.TruckMode {
		fmt.Println("[KEY] N pressed -> Generating new truck")
		g.sim.RegenerateTruck()
	}
}

func (g *Game) handleCamera() {
	// Zoom with mouse wheel
	_, wy := ebiten.Wheel()
	if wy > 0 {
		g.camera.Zoom *= 1.1
		if g.camera.Zoom > 3.0 {
			g.camera.Zoom = 3.0
		}
	} else if wy < 0 {
		g.camera.Zoom *= 0.9
		if g.camera.Zoom < 0.2 {
			g.camera.Zoom = 0.2
		}
	}

	// Pan with right mouse button drag
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.dragging = true
		g.dragStartX, g.dragStartY = ebiten.CursorPosition()
		g.camStartX = g.camera.X
		g.camStartY = g.camera.Y
	}
	if g.dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
			mx, my := ebiten.CursorPosition()
			dx := float64(mx-g.dragStartX) / g.camera.Zoom
			dy := float64(my-g.dragStartY) / g.camera.Zoom
			g.camera.X = g.camStartX - dx
			g.camera.Y = g.camStartY - dy
		} else {
			g.dragging = false
		}
	}

	// WASD pan
	panSpeed := 5.0 / g.camera.Zoom
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.camera.Y -= panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camera.Y += panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.camera.X -= panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) && !inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.camera.X += panSpeed
	}
}

// --- Swarm Mode Input Handling ---

func (g *Game) handleSwarmInput() {
	ss := g.sim.SwarmState
	if ss == nil {
		return
	}
	ed := ss.Editor
	mx, my := ebiten.CursorPosition()

	// Mouse wheel: scroll editor if mouse is in editor area
	// Hold Shift for horizontal scrolling
	_, wy := ebiten.Wheel()
	if mx < 350 && wy != 0 {
		shiftHeld := ebiten.IsKeyPressed(ebiten.KeyShift)
		if shiftHeld {
			// Horizontal scroll
			if wy < 0 {
				ed.ScrollX += 5
			} else {
				ed.ScrollX -= 5
			}
			if ed.ScrollX < 0 {
				ed.ScrollX = 0
			}
			// Clamp to max line length
			maxCol := 0
			for _, line := range ed.Lines {
				if len(line) > maxCol {
					maxCol = len(line)
				}
			}
			if ed.ScrollX > maxCol {
				ed.ScrollX = maxCol
			}
		} else {
			// Vertical scroll
			if wy < 0 {
				ed.ScrollY += 3
			} else {
				ed.ScrollY -= 3
			}
			if ed.ScrollY < 0 {
				ed.ScrollY = 0
			}
			maxScroll := len(ed.Lines) - ed.MaxVisible/2
			if maxScroll < 0 {
				maxScroll = 0
			}
			if ed.ScrollY > maxScroll {
				ed.ScrollY = maxScroll
			}
		}
	}

	// Update dropdown hover tracking
	if ss.DropdownOpen {
		ss.DropdownHover = render.SwarmDropdownHitTest(mx, my, len(ss.PresetNames))
	}

	// Left click handling
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if ss.DropdownOpen {
			// Dropdown is open — check if we clicked on an item
			idx := render.SwarmDropdownHitTest(mx, my, len(ss.PresetNames))
			if idx >= 0 {
				g.loadSwarmPreset(idx)
			}
			ss.DropdownOpen = false
		} else {
			g.handleSwarmClick(mx, my)
		}
	}

	// L key: toggle light source at mouse position (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyL) && !ed.Focused && !ss.BotCountEdit {
		awx, awy, inside := render.SwarmScreenToArena(mx, my)
		if inside {
			if ss.Light.Active {
				ss.Light.Active = false
				fmt.Println("[SWARM] Light OFF")
			} else {
				ss.Light.Active = true
				ss.Light.X = awx
				ss.Light.Y = awy
				fmt.Printf("[SWARM] Light ON at (%.0f, %.0f)\n", awx, awy)
			}
		}
	}

	// T key: toggle trails (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyT) && !ed.Focused && !ss.BotCountEdit {
		ss.ShowTrails = !ss.ShowTrails
		fmt.Printf("[SWARM] Trails: %v\n", ss.ShowTrails)
	}

	// C key: toggle delivery route lines (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyC) && !ed.Focused && !ss.BotCountEdit {
		ss.ShowRoutes = !ss.ShowRoutes
		fmt.Printf("[SWARM] Routes: %v\n", ss.ShowRoutes)
	}

	// Editor keyboard input (when editor is focused)
	if ed.Focused {
		g.handleSwarmEditorKeys()
	}

	// Bot count field input (when focused)
	if ss.BotCountEdit {
		g.handleBotCountInput()
	}
}

func (g *Game) handleSwarmClick(mx, my int) {
	ss := g.sim.SwarmState
	ed := ss.Editor

	hit := render.SwarmEditorHitTest(mx, my)
	switch hit {
	case "dropdown":
		ss.DropdownOpen = !ss.DropdownOpen
		ed.Focused = false
		ss.BotCountEdit = false

	case "deploy":
		g.deploySwarmProgram()
		ss.DropdownOpen = false
		ss.BotCountEdit = false

	case "reset":
		ss.ResetBots()
		fmt.Printf("[SWARM] RESET — %d bots scattered\n", ss.BotCount)
		ss.DropdownOpen = false
		ss.BotCountEdit = false

	case "botcount":
		ss.BotCountEdit = true
		ed.Focused = false
		ss.DropdownOpen = false

	case "editor":
		ed.Focused = true
		ss.BotCountEdit = false
		ss.DropdownOpen = false
		line, col := render.SwarmEditorClickToPos(mx, my, ed)
		ed.CursorLine = line
		ed.CursorCol = col
		ed.BlinkTick = 0

	case "obstacles":
		ss.ObstaclesOn = !ss.ObstaclesOn
		if ss.ObstaclesOn {
			ss.MazeOn = false
			ss.MazeWalls = nil
			swarm.GenerateSwarmObstacles(ss)
			fmt.Printf("[SWARM] Obstacles ON (%d obstacles)\n", len(ss.Obstacles))
		} else {
			ss.Obstacles = nil
			fmt.Println("[SWARM] Obstacles OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "maze":
		ss.MazeOn = !ss.MazeOn
		if ss.MazeOn {
			ss.ObstaclesOn = false
			ss.Obstacles = nil
			swarm.GenerateSwarmMaze(ss)
			fmt.Printf("[SWARM] Maze ON (%d walls)\n", len(ss.MazeWalls))
		} else {
			ss.MazeWalls = nil
			fmt.Println("[SWARM] Maze OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "light":
		if ss.Light.Active {
			ss.Light.Active = false
			fmt.Println("[SWARM] Light OFF (button)")
		} else {
			ss.Light.Active = true
			ss.Light.X = ss.ArenaW / 2
			ss.Light.Y = ss.ArenaH / 2
			fmt.Println("[SWARM] Light ON at arena center")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "walls":
		ss.WrapMode = !ss.WrapMode
		mode := "BOUNCE"
		if ss.WrapMode {
			mode = "WRAP"
		}
		fmt.Printf("[SWARM] Walls mode: %s\n", mode)
		ed.Focused = false
		ss.BotCountEdit = false

	case "delivery":
		ss.DeliveryOn = !ss.DeliveryOn
		if ss.DeliveryOn {
			// Force maze on
			if !ss.MazeOn {
				ss.MazeOn = true
				ss.ObstaclesOn = false
				ss.Obstacles = nil
				swarm.GenerateSwarmMaze(ss)
			}
			// Generate stations and packages
			swarm.GenerateDeliveryStations(ss)
			// Reset bot carrying state
			for i := range ss.Bots {
				ss.Bots[i].CarryingPkg = -1
			}
			fmt.Printf("[SWARM] Delivery ON (%d stations, %d packages)\n", len(ss.Stations), len(ss.Packages))
		} else {
			ss.Stations = nil
			ss.Packages = nil
			ss.DeliveryStats = swarm.DeliveryStats{}
			// Reset bot carrying state
			for i := range ss.Bots {
				ss.Bots[i].CarryingPkg = -1
			}
			fmt.Println("[SWARM] Delivery OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	default:
		// Clicked outside editor panel — check arena for bot selection
		ed.Focused = false
		ss.BotCountEdit = false
		ss.DropdownOpen = false

		// Try to select a bot in the arena
		awx, awy, inside := render.SwarmScreenToArena(mx, my)
		if inside {
			bestIdx := -1
			bestDist := 15.0 // max click distance
			for i := range ss.Bots {
				dx := ss.Bots[i].X - awx
				dy := ss.Bots[i].Y - awy
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < bestDist {
					bestDist = dist
					bestIdx = i
				}
			}
			ss.SelectedBot = bestIdx
			if bestIdx >= 0 {
				fmt.Printf("[SWARM] Selected bot #%d\n", bestIdx)
			}
		} else {
			ss.SelectedBot = -1
		}
	}
}

func (g *Game) handleSwarmEditorKeys() {
	ss := g.sim.SwarmState
	ed := ss.Editor

	// Character input
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		g.editorInsertChar(ch)
	}

	// Enter: new line
	if isKeyRepeating(ebiten.KeyEnter) {
		line := ed.Lines[ed.CursorLine]
		before := line[:ed.CursorCol]
		after := line[ed.CursorCol:]
		ed.Lines[ed.CursorLine] = before
		// Insert new line after current
		newLines := make([]string, len(ed.Lines)+1)
		copy(newLines, ed.Lines[:ed.CursorLine+1])
		newLines[ed.CursorLine+1] = after
		copy(newLines[ed.CursorLine+2:], ed.Lines[ed.CursorLine+1:])
		ed.Lines = newLines
		ed.CursorLine++
		ed.CursorCol = 0
		g.editorEnsureCursorVisible()
		ss.ProgramName = "Custom"
	}

	// Backspace
	if isKeyRepeating(ebiten.KeyBackspace) {
		if ed.CursorCol > 0 {
			line := ed.Lines[ed.CursorLine]
			ed.Lines[ed.CursorLine] = line[:ed.CursorCol-1] + line[ed.CursorCol:]
			ed.CursorCol--
			ss.ProgramName = "Custom"
		} else if ed.CursorLine > 0 {
			// Merge with previous line
			prevLine := ed.Lines[ed.CursorLine-1]
			curLine := ed.Lines[ed.CursorLine]
			ed.Lines[ed.CursorLine-1] = prevLine + curLine
			ed.Lines = append(ed.Lines[:ed.CursorLine], ed.Lines[ed.CursorLine+1:]...)
			ed.CursorLine--
			ed.CursorCol = len(prevLine)
			g.editorEnsureCursorVisible()
			ss.ProgramName = "Custom"
		}
	}

	// Delete
	if isKeyRepeating(ebiten.KeyDelete) {
		line := ed.Lines[ed.CursorLine]
		if ed.CursorCol < len(line) {
			ed.Lines[ed.CursorLine] = line[:ed.CursorCol] + line[ed.CursorCol+1:]
			ss.ProgramName = "Custom"
		} else if ed.CursorLine < len(ed.Lines)-1 {
			// Merge with next line
			nextLine := ed.Lines[ed.CursorLine+1]
			ed.Lines[ed.CursorLine] = line + nextLine
			ed.Lines = append(ed.Lines[:ed.CursorLine+1], ed.Lines[ed.CursorLine+2:]...)
			ss.ProgramName = "Custom"
		}
	}

	// Arrow keys
	if isKeyRepeating(ebiten.KeyLeft) {
		if ed.CursorCol > 0 {
			ed.CursorCol--
		} else if ed.CursorLine > 0 {
			ed.CursorLine--
			ed.CursorCol = len(ed.Lines[ed.CursorLine])
		}
		ed.BlinkTick = 0
		g.editorEnsureCursorVisible()
	}
	if isKeyRepeating(ebiten.KeyRight) {
		lineLen := len(ed.Lines[ed.CursorLine])
		if ed.CursorCol < lineLen {
			ed.CursorCol++
		} else if ed.CursorLine < len(ed.Lines)-1 {
			ed.CursorLine++
			ed.CursorCol = 0
		}
		ed.BlinkTick = 0
		g.editorEnsureCursorVisible()
	}
	if isKeyRepeating(ebiten.KeyUp) {
		if ed.CursorLine > 0 {
			ed.CursorLine--
			if ed.CursorCol > len(ed.Lines[ed.CursorLine]) {
				ed.CursorCol = len(ed.Lines[ed.CursorLine])
			}
		}
		ed.BlinkTick = 0
		g.editorEnsureCursorVisible()
	}
	if isKeyRepeating(ebiten.KeyDown) {
		if ed.CursorLine < len(ed.Lines)-1 {
			ed.CursorLine++
			if ed.CursorCol > len(ed.Lines[ed.CursorLine]) {
				ed.CursorCol = len(ed.Lines[ed.CursorLine])
			}
		}
		ed.BlinkTick = 0
		g.editorEnsureCursorVisible()
	}

	// Home / End
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		ed.CursorCol = 0
		ed.BlinkTick = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		ed.CursorCol = len(ed.Lines[ed.CursorLine])
		ed.BlinkTick = 0
	}

	// Tab: insert 4 spaces
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		for i := 0; i < 4; i++ {
			g.editorInsertChar(' ')
		}
	}
}

func (g *Game) editorInsertChar(ch rune) {
	ss := g.sim.SwarmState
	ed := ss.Editor

	if ch < 32 || ch > 126 {
		return // only printable ASCII
	}

	line := ed.Lines[ed.CursorLine]
	ed.Lines[ed.CursorLine] = line[:ed.CursorCol] + string(ch) + line[ed.CursorCol:]
	ed.CursorCol++
	ed.BlinkTick = 0
	ss.ProgramName = "Custom"
}

func (g *Game) editorEnsureCursorVisible() {
	ed := g.sim.SwarmState.Editor
	// Vertical scroll
	if ed.CursorLine < ed.ScrollY {
		ed.ScrollY = ed.CursorLine
	}
	if ed.CursorLine >= ed.ScrollY+ed.MaxVisible {
		ed.ScrollY = ed.CursorLine - ed.MaxVisible + 1
	}
	// Horizontal scroll — keep cursor within visible columns
	// editorPanelW=350, editorTextX=40, charW=6 → maxVisibleCols ≈ 51
	maxVisibleCols := (350 - 2 - 40) / 6 // = 51
	if ed.CursorCol < ed.ScrollX {
		ed.ScrollX = ed.CursorCol
	}
	if ed.CursorCol >= ed.ScrollX+maxVisibleCols {
		ed.ScrollX = ed.CursorCol - maxVisibleCols + 1
	}
	if ed.ScrollX < 0 {
		ed.ScrollX = 0
	}
}

func (g *Game) handleBotCountInput() {
	ss := g.sim.SwarmState

	// Consume character input
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if ch >= '0' && ch <= '9' {
			ss.BotCountText += string(ch)
		}
	}

	// Backspace
	if isKeyRepeating(ebiten.KeyBackspace) && len(ss.BotCountText) > 0 {
		ss.BotCountText = ss.BotCountText[:len(ss.BotCountText)-1]
	}

	// Enter: apply bot count
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		count, err := strconv.Atoi(ss.BotCountText)
		if err == nil && count >= swarm.SwarmMinBots && count <= swarm.SwarmMaxBots {
			ss.RespawnBots(count)
			fmt.Printf("[SWARM] Bot count changed to %d\n", count)
		} else {
			// Reset to current count
			ss.BotCountText = fmt.Sprintf("%d", ss.BotCount)
		}
		ss.BotCountEdit = false
		ss.Editor.Focused = true
	}
}

func (g *Game) deploySwarmProgram() {
	ss := g.sim.SwarmState
	source := strings.Join(ss.Editor.Lines, "\n")

	prog, err := swarmscript.ParseSwarmScript(source)
	if err != nil {
		ss.ErrorMsg = err.Error()
		// Try to extract line number from error
		ss.ErrorLine = 0
		fmt.Printf("[SWARM] Parse error: %s\n", err.Error())
		return
	}

	ss.Program = prog
	ss.ProgramText = source
	ss.ErrorMsg = ""
	ss.ErrorLine = 0

	// Blink all bots green to confirm deploy
	for i := range ss.Bots {
		ss.Bots[i].BlinkTimer = 30
	}

	fmt.Printf("[SWARM] Program deployed: %d rules\n", len(prog.Rules))
}

func (g *Game) loadSwarmPreset(idx int) {
	ss := g.sim.SwarmState
	if idx < 0 || idx >= len(ss.PresetPrograms) {
		return
	}

	ss.ProgramName = ss.PresetNames[idx]
	presetText := ss.PresetPrograms[idx]
	ss.Editor.Lines = strings.Split(presetText, "\n")
	ss.Editor.CursorLine = 0
	ss.Editor.CursorCol = 0
	ss.Editor.ScrollY = 0
	ss.Editor.ScrollX = 0
	ss.ErrorMsg = ""
	ss.ErrorLine = 0

	// Auto-deploy the preset
	prog, err := swarmscript.ParseSwarmScript(presetText)
	if err == nil {
		ss.Program = prog
		ss.ProgramText = presetText
		for i := range ss.Bots {
			ss.Bots[i].BlinkTimer = 30
		}
		fmt.Printf("[SWARM] Preset '%s' loaded and deployed: %d rules\n", ss.ProgramName, len(prog.Rules))
	}
}

// isKeyRepeating returns true if a key was just pressed OR is being held long enough to repeat.
func isKeyRepeating(key ebiten.Key) bool {
	d := inpututil.KeyPressDuration(key)
	if d == 1 {
		return true // just pressed
	}
	if d >= 20 && (d-20)%3 == 0 {
		return true // repeat after 20 ticks, every 3 ticks
	}
	return false
}

// Draw renders the simulation.
func (g *Game) Draw(screen *ebiten.Image) {
	g.renderer.Draw(screen, g.sim)
	render.DrawHUD(screen, g.sim, ebiten.ActualFPS())
}

// Layout returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Schwarm-Robotik-Simulator")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
