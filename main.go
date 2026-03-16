package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"swarmsim/domain/bot"
	"swarmsim/domain/physics"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
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

	// Capture requests (set in Update, executed in Draw where screen is available)
	screenshotRequested bool
	gifToggleRequested  bool

	// Welcome screen
	showWelcome  bool
	welcomeTick  int
	welcomeReady bool // set after first frame (init bots needs screen size)

	// Help overlay
	showHelp    bool
	helpScrollY int

	// In-game console
	showConsole      bool
	consoleFilterBot int // -1 = all logs, >= 0 = filter for this bot

	// Classic Mode scenario dropdown
	classicDropdownOpen  bool
	classicDropdownHover int
	classicScenarioIdx   int // 0-4 index into classicScenarios
	classicScenarios     []simulation.Scenario

	// Panic recovery overlay
	panicMsg   string
	panicTimer int

	// Tutorial
	tutorial render.TutorialState

	// Tooltips
	tooltip render.TooltipState

	// Single-step debugger (Q key)
	stepMode bool // true = advance one tick per Q press
	stepOnce bool // true = execute exactly one tick this frame

	// Replay / time travel
	replayMode     bool
	replayIdx      int // current snapshot index in replay buffer
	replayWasPause bool // was sim paused before entering replay?
}

// NewGame creates a new game instance.
func NewGame() *Game {
	cfg := simulation.DefaultConfig()
	s := simulation.NewSimulation(cfg)
	cam := render.NewCamera(cfg.ArenaWidth, cfg.ArenaHeight)
	r := render.NewRenderer(cam)
	return &Game{
		sim:              s,
		renderer:         r,
		camera:           cam,
		scenarios:        simulation.GetScenarios(),
		classicScenarios: simulation.GetClassicScenarios(),
		showWelcome:      true,
		showConsole:      true,
		consoleFilterBot: -1,
	}
}

// Update handles input and advances simulation.
func (g *Game) Update() (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("PANIC", "Update panic: %v", r)
			g.panicMsg = fmt.Sprintf("Error in Update: %v", r)
			g.panicTimer = 300
			retErr = nil // don't crash
		}
	}()

	// Panic overlay timer
	if g.panicTimer > 0 {
		g.panicTimer--
	}

	// ESC to quit (or cancel follow-cam first)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.showWelcome {
			return ebiten.Termination
		}
		if g.showHelp {
			g.showHelp = false
			return nil
		}
		// Exit replay mode
		if g.replayMode {
			g.replayMode = false
			g.sim.Paused = g.replayWasPause
			return nil
		}
		// Close tournament overlay
		if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.TournamentOn {
			swarm.TournamentStop(g.sim.SwarmState)
			return nil
		}
		// Cancel follow-cam before quitting
		if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.FollowCamBot >= 0 {
			g.sim.SwarmState.FollowCamBot = -1
			return nil
		}
		return ebiten.Termination
	}

	// Welcome screen: update bots, check for dismiss
	if g.showWelcome {
		g.welcomeTick++
		g.renderer.UpdateWelcomeBots(screenW, screenH)

		// F1: Classic Mode, F2/F7: Swarm Lab
		if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
			idx := g.classicScenarioIdx
			if idx < 0 || idx >= len(g.classicScenarios) {
				idx = 0
			}
			g.sim.LoadScenario(g.classicScenarios[idx])
			g.camera.X = g.sim.Cfg.ArenaWidth / 2
			g.camera.Y = g.sim.Cfg.ArenaHeight / 2
			g.camera.Zoom = 0.7
			g.tickAcc = 0
			g.showWelcome = false
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF2) || inpututil.IsKeyJustPressed(ebiten.KeyF7) {
			g.sim.LoadSwarmScenario()
			g.tickAcc = 0
			g.showWelcome = false
			return nil
		}

		// Any other key or mouse click → load Swarm Lab (default) and dismiss
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.sim.LoadSwarmScenario()
			g.tickAcc = 0
			g.showWelcome = false
			g.maybeStartTutorial()
			return nil
		}
		// Any key press (except ESC which is handled above)
		for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
			if inpututil.IsKeyJustPressed(k) && k != ebiten.KeyEscape {
				g.sim.LoadSwarmScenario()
				g.tickAcc = 0
				g.showWelcome = false
				g.maybeStartTutorial()
				return nil
			}
		}
		return nil
	}

	// Help overlay: only H dismisses, arrow keys scroll
	if g.showHelp {
		if inpututil.IsKeyJustPressed(ebiten.KeyH) {
			g.showHelp = false
			g.helpScrollY = 0
		}
		// Scroll with arrow keys / page up/down
		if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyJ) {
			g.helpScrollY += 4
		}
		if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyK) {
			g.helpScrollY -= 4
			if g.helpScrollY < 0 {
				g.helpScrollY = 0
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
			g.helpScrollY += 200
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
			g.helpScrollY -= 200
			if g.helpScrollY < 0 {
				g.helpScrollY = 0
			}
		}
		// Mouse wheel scrolling
		_, wy := ebiten.Wheel()
		if wy < 0 {
			g.helpScrollY += 48
		} else if wy > 0 {
			g.helpScrollY -= 48
			if g.helpScrollY < 0 {
				g.helpScrollY = 0
			}
		}
		return nil
	}

	// Tutorial update
	if g.tutorial.Active {
		g.updateTutorial()
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
	// Reset flash timer
	if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.ResetFlashTimer > 0 {
		g.sim.SwarmState.ResetFlashTimer--
	}
	// Clipboard flash timer
	if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.ClipboardFlash > 0 {
		g.sim.SwarmState.ClipboardFlash--
	}

	// Follow-cam lerp update
	if g.sim.SwarmMode && g.sim.SwarmState != nil {
		ss := g.sim.SwarmState
		if ss.FollowCamBot >= 0 && ss.FollowCamBot < len(ss.Bots) {
			bot := &ss.Bots[ss.FollowCamBot]
			ss.SwarmCamX += (bot.X - ss.SwarmCamX) * 0.10
			ss.SwarmCamY += (bot.Y - ss.SwarmCamY) * 0.10
			ss.SwarmCamZoom += (1.5 - ss.SwarmCamZoom) * 0.10
		} else {
			// Lerp back to center / zoom 1.0
			center := swarm.SwarmArenaSize / 2
			ss.SwarmCamX += (center - ss.SwarmCamX) * 0.10
			ss.SwarmCamY += (center - ss.SwarmCamY) * 0.10
			ss.SwarmCamZoom += (1.0 - ss.SwarmCamZoom) * 0.10
			if math.Abs(ss.SwarmCamZoom-1.0) < 0.01 {
				ss.SwarmCamZoom = 1.0
				ss.SwarmCamX = center
				ss.SwarmCamY = center
			}
		}
	}

	// Single-step mode: force exactly one tick
	if g.stepMode && g.stepOnce && !g.replayMode {
		g.stepOnce = false
		g.sim.Paused = false
		g.sim.Update()
		g.sim.Paused = true
		// Record replay snapshot
		if g.sim.SwarmMode && g.sim.SwarmState != nil {
			ss := g.sim.SwarmState
			if ss.ReplayBuf == nil {
				ss.ReplayBuf = swarm.NewReplayBuffer(500)
			}
			if ss.Tick%10 == 0 {
				ss.ReplayBuf.Record(ss)
			}
		}
	}

	// Fixed timestep for simulation (skip during replay)
	if !g.replayMode {
		dt := 1.0 / 60.0 * g.sim.Speed
		g.tickAcc += dt
		tickInterval := 1.0 / float64(g.sim.Cfg.TickRate)
		for g.tickAcc >= tickInterval {
			g.sim.Update()
			g.tickAcc -= tickInterval
			// Record replay snapshot every 10 ticks
			if g.sim.SwarmMode && g.sim.SwarmState != nil {
				ss := g.sim.SwarmState
				if ss.ReplayBuf == nil {
					ss.ReplayBuf = swarm.NewReplayBuffer(500)
				}
				if ss.Tick%10 == 0 {
					ss.ReplayBuf.Record(ss)
				}
			}
		}
	}

	return nil
}

// handleGlobalInput handles keys that work in ALL modes (including swarm).
func (g *Game) handleGlobalInput() {
	// Space: pause/resume
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.sim.Paused = !g.sim.Paused
		if !g.sim.Paused {
			g.stepMode = false // resume clears step mode
		}
	}

	// Q: single-step mode (toggle) / advance one tick
	skipQ := g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.Editor != nil && g.sim.SwarmState.Editor.Focused
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) && !skipQ {
		if !g.stepMode {
			g.stepMode = true
			g.sim.Paused = true
			g.stepOnce = true // execute the first tick immediately
		} else {
			g.stepOnce = true // advance one tick
		}
	}

	// +/-: speed (finer steps below 1x for slow-motion)
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		if g.sim.Speed < 1.0 {
			g.sim.Speed *= 2 // 0.125 → 0.25 → 0.5 → 1.0
		} else {
			g.sim.Speed += 0.5
		}
		if g.sim.Speed > 10.0 {
			g.sim.Speed = 10.0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		// In swarm mode, don't consume minus if editor is focused (for typing)
		if !(g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.Editor != nil && g.sim.SwarmState.Editor.Focused) {
			if g.sim.Speed <= 1.0 {
				g.sim.Speed /= 2 // 1.0 → 0.5 → 0.25 → 0.125
			} else {
				g.sim.Speed -= 0.5
			}
			if g.sim.Speed < 0.125 {
				g.sim.Speed = 0.125
			}
		}
	}

	// F1: load Classic Mode (with fade transition)
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) && g.renderer.FadeDir == 0 {
		idx := g.classicScenarioIdx
		if idx < 0 || idx >= len(g.classicScenarios) {
			idx = 0
		}
		logger.Info("KEY", "F1 pressed -> Loading Classic Mode: %s", g.classicScenarios[idx].Name)
		g.renderer.FadeDir = -1
		g.renderer.FadeAlpha = 0
		g.renderer.FadeLoad = func() {
			g.sim.LoadScenario(g.classicScenarios[idx])
			g.camera.X = g.sim.Cfg.ArenaWidth / 2
			g.camera.Y = g.sim.Cfg.ArenaHeight / 2
			g.camera.Zoom = 0.7
			g.tickAcc = 0
		}
	}

	// F2: load Swarm Lab (with fade)
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) && g.renderer.FadeDir == 0 {
		logger.Info("KEY", "F2 pressed -> Loading Swarm Lab")
		g.renderer.FadeDir = -1
		g.renderer.FadeAlpha = 0
		g.renderer.FadeLoad = func() {
			g.sim.LoadSwarmScenario()
			g.tickAcc = 0
		}
	}

	// F7: alias for F2 (legacy Swarm Lab key)
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) && g.renderer.FadeDir == 0 {
		logger.Info("KEY", "F7 pressed -> Loading Swarm Lab (legacy)")
		g.renderer.FadeDir = -1
		g.renderer.FadeAlpha = 0
		g.renderer.FadeLoad = func() {
			g.sim.LoadSwarmScenario()
			g.tickAcc = 0
		}
	}

	// F3: start tutorial
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.tutorial.Active = true
		g.tutorial.Step = 0
		g.tutorial.InputDone = false
		g.tutorial.WaitTimer = 0
		g.tutorial.Dismissed = false
		logger.Info("KEY", "F3 pressed -> Tutorial started")
	}

	// F10: screenshot
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		g.screenshotRequested = true
	}

	// F11: toggle GIF recording
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		g.gifToggleRequested = true
	}

	// S: toggle sound
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		// Only toggle if not in swarm editor text input
		if !(g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.Editor != nil && g.sim.SwarmState.Editor.Focused) &&
			!(g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.BotCountEdit) {
			g.renderer.Sound.Enabled = !g.renderer.Sound.Enabled
			if g.renderer.Sound.Enabled {
				g.renderer.Sound.StartAmbient()
			} else {
				g.renderer.Sound.StopAmbient()
			}
			logger.Info("KEY", "S pressed -> Sound=%v", g.renderer.Sound.Enabled)
		}
	}

	// H: toggle help overlay (when not in editor text input)
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		if !(g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.Editor != nil && g.sim.SwarmState.Editor.Focused) &&
			!(g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.BotCountEdit) {
			g.showHelp = !g.showHelp
		}
	}

	// Backquote (`) or Semicolon (Ö on German keyboards): toggle in-game log console
	if inpututil.IsKeyJustPressed(ebiten.KeyGraveAccent) ||
		inpututil.IsKeyJustPressed(ebiten.KeySemicolon) {
		if !(g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.Editor != nil && g.sim.SwarmState.Editor.Focused) &&
			!(g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.BotCountEdit) {
			g.showConsole = !g.showConsole
			logger.Info("KEY", "Console toggled: %v", g.showConsole)
		}
	}

	// F12: toggle CPU profiling (requires: go build -tags profile)
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		ToggleProfile()
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

	// O: add obstacle (was H, now H is help)
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		g.sim.AddObstacleAt(wx, wy)
	}

	// F: toggle comm radius
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		g.sim.ShowCommRadius = !g.sim.ShowCommRadius
		logger.Info("KEY", "F pressed -> ShowCommRadius=%v", g.sim.ShowCommRadius)
	}

	// G: toggle sensor radius
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.sim.ShowSensorRadius = !g.sim.ShowSensorRadius
		logger.Info("KEY", "G pressed -> ShowSensorRadius=%v", g.sim.ShowSensorRadius)
	}

	// D: toggle debug comm lines
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.sim.ShowDebugComm = !g.sim.ShowDebugComm
		logger.Info("KEY", "D pressed -> ShowDebugComm=%v", g.sim.ShowDebugComm)
	}

	// T: toggle trail rendering
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		g.renderer.ShowTrails = !g.renderer.ShowTrails
		logger.Info("KEY", "T pressed -> ShowTrails=%v", g.renderer.ShowTrails)
	}

	// M: toggle minimap
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.renderer.ShowMinimap = !g.renderer.ShowMinimap
		logger.Info("KEY", "M pressed -> ShowMinimap=%v", g.renderer.ShowMinimap)
	}

	// P: cycle pheromone visualization (OFF -> FOUND -> ALL -> OFF)
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.sim.PheromoneVizMode = (g.sim.PheromoneVizMode + 1) % 3
		modes := []string{"OFF", "FOUND", "ALL"}
		logger.Info("KEY", "P pressed -> PheromoneVizMode=%s (%d)", modes[g.sim.PheromoneVizMode], g.sim.PheromoneVizMode)
	}

	// E: force end generation (evolve)
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.sim.ForceEndGeneration()
		logger.Info("KEY", "E pressed -> Generation=%d Best=%.1f Avg=%.1f", g.sim.Generation, g.sim.BestFitness, g.sim.AvgFitness)
	}

	// V: toggle genome overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		g.sim.ShowGenomeOverlay = !g.sim.ShowGenomeOverlay
		logger.Info("KEY", "V pressed -> ShowGenomeOverlay=%v (SelectedBot=%d)", g.sim.ShowGenomeOverlay, g.sim.SelectedBotID)
	}

	// N: switch scenario in Classic Mode (cycle through dropdown)
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		if len(g.classicScenarios) > 0 && g.renderer.FadeDir == 0 {
			g.classicScenarioIdx = (g.classicScenarioIdx + 1) % len(g.classicScenarios)
			idx := g.classicScenarioIdx
			logger.Info("KEY", "N pressed -> Switching to scenario: %s", g.classicScenarios[idx].Name)
			g.renderer.FadeDir = -1
			g.renderer.FadeAlpha = 0
			g.renderer.FadeLoad = func() {
				g.sim.LoadScenario(g.classicScenarios[idx])
				g.camera.X = g.sim.Cfg.ArenaWidth / 2
				g.camera.Y = g.sim.Cfg.ArenaHeight / 2
				g.camera.Zoom = 0.7
				g.tickAcc = 0
			}
		}
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
	if ebiten.IsKeyPressed(ebiten.KeyS) && !inpututil.IsKeyJustPressed(ebiten.KeyS) {
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
	if mx < 350 && wy != 0 && ss.BlockEditorActive {
		// Block editor scroll
		if wy < 0 {
			ss.BlockScrollY += 24
		} else {
			ss.BlockScrollY -= 24
		}
		if ss.BlockScrollY < 0 {
			ss.BlockScrollY = 0
		}
		maxScroll := len(ss.BlockRules)*26 - 200
		if maxScroll < 0 {
			maxScroll = 0
		}
		if ss.BlockScrollY > maxScroll {
			ss.BlockScrollY = maxScroll
		}
	} else if mx < 350 && wy != 0 {
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

	// Bot hover detection for tooltip
	ss.HoveredBot = -1
	awx, awy, inside := render.SwarmScreenToArena(mx, my, ss)
	if inside && !ss.DropdownOpen && !ss.ArenaEditMode {
		bestDist := 12.0
		for i := range ss.Bots {
			dx := ss.Bots[i].X - awx
			dy := ss.Bots[i].Y - awy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < bestDist {
				bestDist = dist
				ss.HoveredBot = i
			}
		}
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
		awx, awy, inside := render.SwarmScreenToArena(mx, my, ss)
		if inside {
			if ss.Light.Active {
				ss.Light.Active = false
				logger.Info("SWARM", "Light OFF")
			} else {
				ss.Light.Active = true
				ss.Light.X = awx
				ss.Light.Y = awy
				logger.Info("SWARM", "Light ON at (%.0f, %.0f)", awx, awy)
			}
		}
	}

	// T key: toggle trails (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyT) && !ed.Focused && !ss.BotCountEdit {
		ss.ShowTrails = !ss.ShowTrails
		logger.Info("SWARM", "Trails: %v", ss.ShowTrails)
	}

	// F5 key: start/stop scenario chain
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) && !ed.Focused {
		if ss.ScenarioChain != nil && ss.ScenarioChain.Active {
			swarm.ScenarioChainStop(ss)
			logger.Info("SWARM", "Scenario chain stopped")
		} else {
			swarm.ScenarioChainStart(ss)
		}
	}

	// F4 key: start/stop auto-optimizer
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) && !ed.Focused {
		if ss.AutoOptimizer != nil && ss.AutoOptimizer.Active {
			ss.AutoOptimizer.Active = false
			logger.Info("SWARM", "Auto-Optimizer stopped")
		} else {
			if !ss.DeliveryOn {
				ss.DeliveryOn = true
				swarm.GenerateDeliveryStations(ss)
			}
			ss.EvolutionOn = true
			swarm.ScanUsedParams(ss)
			swarm.AutoOptimizerStart(ss)
			g.sim.Speed = 10.0 // max speed during optimization
		}
	}

	// F6 key: toggle formation analysis overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) && !ed.Focused {
		ss.ShowFormation = !ss.ShowFormation
		logger.Info("SWARM", "Formation-Analyse: %v", ss.ShowFormation)
	}

	// F8 key: save parameter preset
	if inpututil.IsKeyJustPressed(ebiten.KeyF8) && !ed.Focused {
		name := fmt.Sprintf("Preset_%s_%d", ss.ProgramName, ss.Tick)
		swarm.SavePreset(ss, name)
		g.renderer.OverlayText = "Preset gespeichert: " + name
		g.renderer.OverlayTimer = 60
	}

	// F9 key: cycle and load saved presets
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) && !ed.Focused {
		names := swarm.ListPresets()
		if len(names) > 0 {
			ss.PresetIdx = ss.PresetIdx % len(names)
			swarm.LoadPreset(ss, names[ss.PresetIdx])
			g.renderer.OverlayText = "Preset geladen: " + names[ss.PresetIdx]
			g.renderer.OverlayTimer = 60
			ss.PresetIdx = (ss.PresetIdx + 1) % len(names)
		} else {
			g.renderer.OverlayText = "Keine Presets gespeichert (F8 zum Speichern)"
			g.renderer.OverlayTimer = 60
		}
	}

	// Period key: toggle live chart
	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) && !ed.Focused && !ss.BotCountEdit {
		ss.ShowLiveChart = !ss.ShowLiveChart
		logger.Info("SWARM", "Live-Chart: %v", ss.ShowLiveChart)
	}

	// Slash key: toggle Pareto multi-objective evolution
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && !ed.Focused && !ss.BotCountEdit {
		ss.ParetoEnabled = !ss.ParetoEnabled
		if ss.ParetoEnabled {
			logger.Info("SWARM", "Pareto-Modus: ON (Multi-Objective)")
		} else {
			ss.ParetoFront = nil
			logger.Info("SWARM", "Pareto-Modus: OFF (Skalare Fitness)")
		}
	}

	// Comma key: toggle bot spatial memory
	if inpututil.IsKeyJustPressed(ebiten.KeyComma) && !ed.Focused && !ss.BotCountEdit {
		ss.MemoryEnabled = !ss.MemoryEnabled
		if ss.MemoryEnabled {
			swarm.InitBotMemory(ss)
		} else {
			swarm.ClearBotMemory(ss)
		}
		logger.Info("SWARM", "Bot-Gedaechtnis: %v", ss.MemoryEnabled)
	}

	// Y key: toggle heatmap overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyY) && !ed.Focused && !ss.BotCountEdit {
		ss.ShowHeatmap = !ss.ShowHeatmap
		if ss.ShowHeatmap && ss.HeatmapGrid == nil {
			swarm.InitHeatmap(ss)
		}
		if !ss.ShowHeatmap {
			swarm.ClearHeatmap(ss)
		}
		logger.Info("SWARM", "Heatmap: %v", ss.ShowHeatmap)
	}

	// W key: cycle color filter (off → red → green → blue → carrying → idle → off)
	if inpututil.IsKeyJustPressed(ebiten.KeyW) && !ed.Focused && !ss.BotCountEdit {
		ss.ColorFilter = (ss.ColorFilter + 1) % 6
		filterNames := []string{"OFF", "Rot", "Gruen", "Blau", "Traegt Paket", "Idle"}
		logger.Info("SWARM", "Color filter: %s", filterNames[ss.ColorFilter])
	}

	// P key: toggle message wave visualization
	if inpututil.IsKeyJustPressed(ebiten.KeyP) && !ed.Focused && !ss.BotCountEdit {
		ss.ShowMsgWaves = !ss.ShowMsgWaves
		if !ss.ShowMsgWaves {
			ss.MsgWaves = nil
		}
		logger.Info("SWARM", "Message waves: %v", ss.ShowMsgWaves)
	}

	// C key: challenge (teams) or toggle routes (delivery)
	if inpututil.IsKeyJustPressed(ebiten.KeyC) && !ed.Focused && !ss.BotCountEdit {
		if ss.TeamsEnabled {
			// Start challenge: 5000 ticks
			ss.ChallengeActive = true
			ss.ChallengeTicks = 5000
			ss.ChallengeResult = ""
			ss.TeamAScore = 0
			ss.TeamBScore = 0
			logger.Info("SWARM", "Challenge started! 5000 ticks")
		} else {
			ss.ShowRoutes = !ss.ShowRoutes
			logger.Info("SWARM", "Routes: %v", ss.ShowRoutes)
		}
	}

	// N key: new round (teams) or new truck round
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && !ed.Focused && !ss.BotCountEdit {
		if ss.TeamsEnabled {
			swarm.ResetTeamScores(ss)
			if ss.DeliveryOn {
				ss.ResetDeliveryState()
				swarm.GenerateDeliveryStations(ss)
			}
			logger.Info("SWARM", "Teams: New round!")
		} else if ss.TruckToggle && ss.TruckState != nil {
			// Always allow N to restart trucks (not just in RoundDone)
			oldRound := ss.TruckState.RoundNum
			ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
			ss.TruckState.RoundNum = oldRound + 1
			ss.ResetBots()
			if ss.DeliveryOn {
				swarm.GenerateDeliveryStations(ss)
			}
			logger.Info("SWARM", "New truck round %d", ss.TruckState.RoundNum)
		}
	}

	// M key: toggle minimap (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyM) && !ed.Focused && !ss.BotCountEdit {
		g.renderer.ShowMinimap = !g.renderer.ShowMinimap
		logger.Info("SWARM", "Minimap: %v", g.renderer.ShowMinimap)
	}

	// D key: toggle dashboard (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyD) && !ed.Focused && !ss.BotCountEdit {
		ss.DashboardOn = !ss.DashboardOn
		if ss.DashboardOn && ss.StatsTracker == nil {
			ss.StatsTracker = swarm.NewStatsTracker()
		}
		logger.Info("SWARM", "Dashboard: %v", ss.DashboardOn)
	}

	// Z key: toggle replay mode
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) && !ed.Focused && !ss.BotCountEdit {
		if !g.replayMode {
			if ss.ReplayBuf != nil && ss.ReplayBuf.Count > 0 {
				g.replayMode = true
				g.replayIdx = ss.ReplayBuf.Count - 1 // start at newest
				g.replayWasPause = g.sim.Paused
				g.sim.Paused = true
				logger.Info("SWARM", "Replay ON — %d snapshots, Arrow keys to scrub, ESC to exit", ss.ReplayBuf.Count)
			}
		} else {
			g.replayMode = false
			g.sim.Paused = g.replayWasPause
			logger.Info("SWARM", "Replay OFF")
		}
	}

	// Replay navigation (when in replay mode)
	if g.replayMode && ss.ReplayBuf != nil {
		step := 1
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			step = 10
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || (ebiten.IsKeyPressed(ebiten.KeyLeft) && ss.Tick%3 == 0) {
			g.replayIdx -= step
			if g.replayIdx < 0 {
				g.replayIdx = 0
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) || (ebiten.IsKeyPressed(ebiten.KeyRight) && ss.Tick%3 == 0) {
			g.replayIdx += step
			if g.replayIdx >= ss.ReplayBuf.Count {
				g.replayIdx = ss.ReplayBuf.Count - 1
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
			g.replayIdx = 0
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
			g.replayIdx = ss.ReplayBuf.Count - 1
		}
	}

	// I key: toggle energy system
	if inpututil.IsKeyJustPressed(ebiten.KeyI) && !ed.Focused && !ss.BotCountEdit {
		ss.EnergyEnabled = !ss.EnergyEnabled
		if ss.EnergyEnabled {
			// Initialize all bots with full energy
			for i := range ss.Bots {
				ss.Bots[i].Energy = 100
			}
			logger.Info("SWARM", "Energy system ON — Bots verbrauchen Energie bei Bewegung")
		} else {
			logger.Info("SWARM", "Energy system OFF")
		}
	}

	// B key: bookmark current fitness curve as baseline for comparison
	if inpututil.IsKeyJustPressed(ebiten.KeyB) && !ed.Focused && !ss.BotCountEdit {
		if len(ss.FitnessHistory) > 1 {
			ss.BaselineFitness = make([]swarm.FitnessRecord, len(ss.FitnessHistory))
			copy(ss.BaselineFitness, ss.FitnessHistory)
			gen := ss.Generation
			if ss.GPEnabled {
				gen = ss.GPGeneration
			}
			if ss.NeuroEnabled {
				gen = ss.NeuroGeneration
			}
			ss.BaselineLabel = fmt.Sprintf("Baseline Gen %d", gen)
			logger.Info("SWARM", "Fitness baseline saved (%d generations)", len(ss.BaselineFitness))
		}
	}

	// K key: toggle communication graph overlay
	if inpututil.IsKeyJustPressed(ebiten.KeyK) && !ed.Focused && !ss.BotCountEdit {
		ss.ShowCommGraph = !ss.ShowCommGraph
		logger.Info("SWARM", "Comm graph: %v", ss.ShowCommGraph)
	}

	// A key: toggle action heatmap vs motion heatmap on dashboard
	if inpututil.IsKeyJustPressed(ebiten.KeyA) && !ed.Focused && !ss.BotCountEdit {
		if ss.DashboardOn && ss.StatsTracker != nil {
			ss.StatsTracker.ShowActionHeat = !ss.StatsTracker.ShowActionHeat
			logger.Info("SWARM", "Action heatmap: %v", ss.StatsTracker.ShowActionHeat)
		}
	}

	// Speed presets: number keys 1-5 (when editor not focused)
	if !ed.Focused && !ss.BotCountEdit {
		speedPresets := map[ebiten.Key]float64{
			ebiten.KeyDigit1: 0.5,
			ebiten.KeyDigit2: 1.0,
			ebiten.KeyDigit3: 2.0,
			ebiten.KeyDigit4: 5.0,
			ebiten.KeyDigit5: 10.0,
		}
		for key, speed := range speedPresets {
			if inpututil.IsKeyJustPressed(key) {
				g.sim.Speed = speed
				logger.Info("SWARM", "Speed preset: %.1fx (key %d)", speed, key-ebiten.KeyDigit0)
			}
		}
	}

	// X key: export stats as CSV to clipboard
	if inpututil.IsKeyJustPressed(ebiten.KeyX) && !ed.Focused && !ss.BotCountEdit {
		csv := g.buildStatsCSV()
		if csv != "" {
			render.ClipboardWrite(csv)
			ss.ClipboardFlash = 30
			logger.Info("SWARM", "Stats exported to clipboard (%d bytes)", len(csv))
		}
	}

	// V key: toggle genome visualization (when editor not focused + evolution on)
	if inpututil.IsKeyJustPressed(ebiten.KeyV) && !ed.Focused && !ss.BotCountEdit {
		if ss.EvolutionOn {
			ss.ShowGenomeViz = !ss.ShowGenomeViz
			logger.Info("SWARM", "Genome viz: %v", ss.ShowGenomeViz)
		}
	}

	// G key: toggle Genom-Browser (when any evolution mode is active)
	if inpututil.IsKeyJustPressed(ebiten.KeyG) && !ed.Focused && !ss.BotCountEdit {
		if ss.EvolutionOn || ss.GPEnabled || ss.NeuroEnabled {
			ss.GenomeBrowserOn = !ss.GenomeBrowserOn
			ss.GenomeBrowserScroll = 0
			logger.Info("SWARM", "Genom-Browser: %v", ss.GenomeBrowserOn)
		}
	}

	// G+Up/Down: scroll Genom-Browser, G+Tab: change sort
	if ss.GenomeBrowserOn {
		_, wy := ebiten.Wheel()
		if wy < 0 {
			ss.GenomeBrowserScroll += 3
		} else if wy > 0 {
			ss.GenomeBrowserScroll -= 3
			if ss.GenomeBrowserScroll < 0 {
				ss.GenomeBrowserScroll = 0
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) && !ed.Focused {
			ss.GenomeBrowserSort = (ss.GenomeBrowserSort + 1) % 3
			logger.Info("SWARM", "Genom-Browser sort: %d", ss.GenomeBrowserSort)
			return
		}
	}

	// Tab key: toggle console bot filter (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) && !ed.Focused && !ss.BotCountEdit {
		if g.consoleFilterBot >= 0 {
			g.consoleFilterBot = -1
		} else if ss.SelectedBot >= 0 {
			g.consoleFilterBot = ss.SelectedBot
		}
	}

	// J key: toggle dynamic environment (moving obstacles + package expiry)
	if inpututil.IsKeyJustPressed(ebiten.KeyJ) && !ed.Focused && !ss.BotCountEdit {
		ss.DynamicEnv = !ss.DynamicEnv
		logger.Info("SWARM", "Dynamic environment: %v", ss.DynamicEnv)
	}

	// F key: toggle follow-cam (when editor not focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && !ed.Focused && !ss.BotCountEdit {
		if ss.FollowCamBot >= 0 {
			ss.FollowCamBot = -1
			logger.Info("SWARM", "Follow-cam OFF")
		} else if ss.SelectedBot >= 0 {
			ss.FollowCamBot = ss.SelectedBot
			logger.Info("SWARM", "Follow-cam ON: Bot #%d", ss.FollowCamBot)
		}
	}

	// U key: tournament mode
	if inpututil.IsKeyJustPressed(ebiten.KeyU) && !ed.Focused && !ss.BotCountEdit {
		if !ss.TournamentOn {
			// Add current program to tournament roster and open tournament
			name := ss.ProgramName
			if name == "" {
				name = "Custom"
			}
			source := strings.Join(ed.Lines, "\n")
			swarm.TournamentAddEntry(ss, name, source)
			ss.TournamentOn = true
		} else if ss.TournamentPhase == 0 {
			// Already in tournament idle — add current program
			name := ss.ProgramName
			if name == "" {
				name = "Custom"
			}
			source := strings.Join(ed.Lines, "\n")
			swarm.TournamentAddEntry(ss, name, source)
		} else if ss.TournamentPhase == 2 {
			// Results shown — reset for new tournament
			ss.TournamentEntries = nil
			ss.TournamentResults = nil
			ss.TournamentPhase = 0
			logger.Info("SWARM", "Tournament reset")
		}
	}

	// Enter key in tournament idle: start the tournament
	if ss.TournamentOn && ss.TournamentPhase == 0 &&
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) && !ed.Focused {
		if len(ss.TournamentEntries) >= 2 {
			if !ss.DeliveryOn {
				ss.DeliveryOn = true
				swarm.GenerateDeliveryStations(ss)
			}
			swarm.TournamentStart(ss)
		}
	}

	// O key: toggle arena edit mode
	if inpututil.IsKeyJustPressed(ebiten.KeyO) && !ed.Focused && !ss.BotCountEdit {
		ss.ArenaEditMode = !ss.ArenaEditMode
		ss.ArenaDragIdx = -1
		if ss.ArenaEditMode {
			ss.ArenaEditTool = 0 // default: obstacle
			logger.Info("SWARM", "Arena-Editor ON (Tool: Obstacle)")
		} else {
			logger.Info("SWARM", "Arena-Editor OFF")
		}
	}

	// Arena edit tool switching: 1=obstacle, 2=station, 3=delete (when in edit mode)
	if ss.ArenaEditMode && !ed.Focused {
		if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
			ss.ArenaEditTool = 0
			logger.Info("SWARM", "Arena-Editor Tool: Obstacle")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
			ss.ArenaEditTool = 1
			logger.Info("SWARM", "Arena-Editor Tool: Station")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDigit3) {
			ss.ArenaEditTool = 2
			logger.Info("SWARM", "Arena-Editor Tool: Delete")
		}
	}

	// Editor keyboard input (when editor is focused)
	if ed.Focused {
		g.handleSwarmEditorKeys()
	}

	// Bot count field input (when focused)
	if ss.BotCountEdit {
		g.handleBotCountInput()
	}

	// Block editor value field input
	if ss.BlockValueEdit {
		g.handleBlockValueInput()
	}

	// Tooltip hover detection
	hoverID := ""
	if !ed.Focused && !ss.BotCountEdit {
		hoverID = render.SwarmEditorHitTest(mx, my)
		// Also check dropdown hover for preset tooltips
		if ss.DropdownOpen {
			idx := render.SwarmDropdownHitTest(mx, my, len(ss.PresetNames))
			if idx >= 0 && idx < len(ss.PresetNames) {
				hoverID = "preset:" + ss.PresetNames[idx]
			}
		}
	}
	render.UpdateTooltip(&g.tooltip, mx, my, hoverID)
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
		g.renderer.Sound.PlayDeploy()

	case "reset":
		ss.ResetBots()
		// Re-create truck state so a fresh truck arrives immediately
		if ss.TruckToggle {
			ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
		}
		logger.Info("SWARM", "RESET — %d bots scattered", ss.BotCount)
		g.renderer.Sound.PlayReset()
		ss.DropdownOpen = false
		ss.BotCountEdit = false

	case "copy":
		// Export program to clipboard
		text := strings.Join(ed.Lines, "\n")
		render.ClipboardWrite(text)
		ss.ClipboardFlash = 30
		logger.Info("SWARM", "Program copied to clipboard (%d lines)", len(ed.Lines))

	case "paste":
		// Import program from clipboard
		render.ClipboardRead(func(text string) {
			if text == "" {
				return
			}
			lines := strings.Split(text, "\n")
			ed.Lines = lines
			ed.CursorLine = 0
			ed.CursorCol = 0
			ed.ScrollY = 0
			ss.ProgramName = "Custom"
			logger.Info("SWARM", "Program pasted from clipboard (%d lines)", len(lines))
		})

	case "botcount":
		ss.BotCountEdit = true
		ed.Focused = false
		ss.DropdownOpen = false

	case "bots_minus":
		newCount := ss.BotCount - 10
		if newCount < swarm.SwarmMinBots {
			newCount = swarm.SwarmMinBots
		}
		ss.RespawnBots(newCount)
		logger.Info("SWARM", "Bot count decreased to %d", newCount)
		ed.Focused = false
		ss.BotCountEdit = false

	case "bots_plus":
		newCount := ss.BotCount + 10
		if newCount > swarm.SwarmMaxBots {
			newCount = swarm.SwarmMaxBots
		}
		ss.RespawnBots(newCount)
		logger.Info("SWARM", "Bot count increased to %d", newCount)
		ed.Focused = false
		ss.BotCountEdit = false

	case "text_mode":
		if ss.BlockEditorActive {
			// Serialize blocks back to text
			ss.BlockEditorActive = false
			ss.ActiveDropdown = nil
			text := serializeBlockRules(ss.BlockRules)
			ss.Editor.Lines = strings.Split(text, "\n")
			ss.Editor.CursorLine = 0
			ss.Editor.CursorCol = 0
			ss.Editor.ScrollY = 0
			logger.Info("SWARM", "Switched to TEXT mode")
		}

	case "block_mode":
		if !ss.BlockEditorActive {
			// Parse current text into blocks
			source := strings.Join(ss.Editor.Lines, "\n")
			prog, err := swarmscript.ParseSwarmScript(source)
			if err != nil {
				ss.ErrorMsg = "Block-Modus: " + err.Error()
				logger.Warn("SWARM", "Cannot switch to block mode: %s", err.Error())
			} else {
				ss.BlockRules = rulesToBlockRules(prog.Rules)
				ss.BlockEditorActive = true
				ss.ActiveDropdown = nil
				ss.BlockScrollY = 0
				ss.BlockValueEdit = false
				ss.ErrorMsg = ""
				ed.Focused = false
				logger.Info("SWARM", "Switched to BLOCKS mode (%d rules)", len(ss.BlockRules))
			}
		}

	case "editor":
		if ss.BlockEditorActive {
			g.handleBlockEditorClick(mx, my)
		} else {
			ed.Focused = true
			ss.BotCountEdit = false
			ss.DropdownOpen = false
			line, col := render.SwarmEditorClickToPos(mx, my, ed)
			ed.CursorLine = line
			ed.CursorCol = col
			ed.BlinkTick = 0
		}

	case "obstacles":
		ss.ObstaclesOn = !ss.ObstaclesOn
		if ss.ObstaclesOn {
			ss.MazeOn = false
			ss.MazeWalls = nil
			swarm.GenerateSwarmObstacles(ss)
			logger.Info("SWARM", "Obstacles ON (%d obstacles)", len(ss.Obstacles))
		} else {
			ss.Obstacles = nil
			logger.Info("SWARM", "Obstacles OFF")
		}
		g.autoReset("Obstacles toggled")
		ed.Focused = false
		ss.BotCountEdit = false

	case "maze":
		ss.MazeOn = !ss.MazeOn
		if ss.MazeOn {
			ss.ObstaclesOn = false
			ss.Obstacles = nil
			swarm.GenerateSwarmMaze(ss)
			logger.Info("SWARM", "Maze ON (%d walls)", len(ss.MazeWalls))
		} else {
			ss.MazeWalls = nil
			logger.Info("SWARM", "Maze OFF")
		}
		g.autoReset("Maze toggled")
		ed.Focused = false
		ss.BotCountEdit = false

	case "light":
		if ss.Light.Active {
			ss.Light.Active = false
			logger.Info("SWARM", "Light OFF (button)")
		} else {
			ss.Light.Active = true
			ss.Light.X = ss.ArenaW / 2
			ss.Light.Y = ss.ArenaH / 2
			logger.Info("SWARM", "Light ON at arena center")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "walls":
		ss.WrapMode = !ss.WrapMode
		mode := "BOUNCE"
		if ss.WrapMode {
			mode = "WRAP"
		}
		// Scatter bot positions only (no full reset)
		for i := range ss.Bots {
			margin := 30.0
			ss.Bots[i].X = margin + ss.Rng.Float64()*(ss.ArenaW-2*margin)
			ss.Bots[i].Y = margin + ss.Rng.Float64()*(ss.ArenaH-2*margin)
		}
		ss.ResetFlashTimer = 30
		logger.Info("SWARM", "Walls mode: %s (positions reset)", mode)
		ed.Focused = false
		ss.BotCountEdit = false

	case "delivery":
		// Non-delivery preset: button is grayed, ignore clicks
		if ss.ProgramName != "Custom" && !ss.IsDeliveryProgram {
			break
		}
		// Delivery preset: cannot disable delivery
		if ss.IsDeliveryProgram && ss.DeliveryOn {
			logger.Info("SWARM", "Cannot disable delivery while a delivery program is active")
			break
		}
		// Neuro needs delivery for fitness
		if ss.NeuroEnabled && ss.DeliveryOn {
			logger.Info("SWARM", "Cannot disable delivery while Neuro is active")
			break
		}
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
			logger.Info("SWARM", "Delivery ON (%d stations, %d packages)", len(ss.Stations), len(ss.Packages))
		} else {
			ss.Stations = nil
			ss.Packages = nil
			ss.DeliveryStats = swarm.DeliveryStats{}
			// Reset bot carrying state
			for i := range ss.Bots {
				ss.Bots[i].CarryingPkg = -1
			}
			logger.Info("SWARM", "Delivery OFF")
		}
		g.autoReset("Delivery toggled")
		ed.Focused = false
		ss.BotCountEdit = false

	case "trucks":
		// Non-truck preset: button is grayed, ignore clicks
		if ss.ProgramName != "Custom" && !ss.IsTruckProgram {
			break
		}
		// Truck preset: cannot disable trucks
		if ss.IsTruckProgram && ss.TruckToggle {
			logger.Info("SWARM", "Cannot disable trucks while a truck program is active")
			break
		}
		ss.TruckToggle = !ss.TruckToggle
		if ss.TruckToggle {
			// Force delivery on (need dropoff stations)
			if !ss.DeliveryOn {
				ss.DeliveryOn = true
			}
			// Maze OFF when trucks active (bots + trucks use open arena)
			ss.MazeOn = false
			ss.MazeWalls = nil
			ss.ObstaclesOn = false
			ss.Obstacles = nil
			// Generate stations and truck state
			swarm.GenerateDeliveryStations(ss)
			ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
			logger.Info("SWARM", "Trucks ON (round %d)", ss.TruckState.RoundNum)
		} else {
			ss.TruckState = nil
			// Regenerate maze without ramp exemption
			if ss.MazeOn {
				swarm.GenerateSwarmMaze(ss)
			}
			logger.Info("SWARM", "Trucks OFF")
		}
		g.autoReset("Trucks toggled")
		ed.Focused = false
		ss.BotCountEdit = false

	case "evolution":
		ss.EvolutionOn = !ss.EvolutionOn
		if ss.EvolutionOn {
			ss.GPEnabled = false // mutually exclusive
			ss.NeuroEnabled = false
			swarm.ClearGP(ss)
			swarm.ClearNeuro(ss)
			swarm.InitBotParams(ss)
			ss.Generation = 0
			ss.EvolutionTimer = 0
			ss.FitnessHistory = nil
			logger.Info("SWARM", "Evolution ON — %d used params", countUsedParams(ss))
		} else {
			ss.ShowGenomeViz = false
			ss.GenomeBrowserOn = false
			logger.Info("SWARM", "Evolution OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "teams":
		ss.TeamsEnabled = !ss.TeamsEnabled
		if ss.TeamsEnabled {
			swarm.InitTeams(ss)
			// Set team programs from current shared program
			if ss.Program != nil {
				ss.TeamAProgram = ss.Program
				ss.TeamBProgram = ss.Program
			}
			logger.Info("SWARM", "Teams ON — %d bots split into two teams", ss.BotCount)
		} else {
			swarm.ClearTeams(ss)
			logger.Info("SWARM", "Teams OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "gp":
		ss.GPEnabled = !ss.GPEnabled
		if ss.GPEnabled {
			// Turn off regular evolution and neuro (mutually exclusive)
			ss.EvolutionOn = false
			ss.ShowGenomeViz = false
			ss.NeuroEnabled = false
			swarm.ClearNeuro(ss)
			swarm.InitGP(ss)
			logger.Info("SWARM", "GP ON — %d bots, each with own random program", ss.BotCount)
		} else {
			swarm.ClearGP(ss)
			logger.Info("SWARM", "GP OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	case "neuro":
		ss.NeuroEnabled = !ss.NeuroEnabled
		if ss.NeuroEnabled {
			// Turn off regular evolution and GP (mutually exclusive)
			ss.EvolutionOn = false
			ss.ShowGenomeViz = false
			ss.GPEnabled = false
			swarm.ClearGP(ss)
			// Auto-enable delivery if not already on
			if !ss.DeliveryOn {
				ss.DeliveryOn = true
				ss.IsDeliveryProgram = true
				swarm.GenerateDeliveryStations(ss)
			}
			swarm.InitNeuro(ss)
			logger.Info("SWARM", "NEURO ON — %d Bots × %d Gewichte, Delivery automatisch aktiviert",
				ss.BotCount, swarm.NeuroWeights)
		} else {
			swarm.ClearNeuro(ss)
			logger.Info("SWARM", "NEURO OFF")
		}
		ed.Focused = false
		ss.BotCountEdit = false

	default:
		// Clicked outside editor panel — check arena for bot selection
		ed.Focused = false
		ss.BotCountEdit = false
		ss.DropdownOpen = false

		// Arena edit mode: place/remove objects
		awx, awy, inside := render.SwarmScreenToArena(mx, my, ss)
		if inside && ss.ArenaEditMode {
			g.handleArenaEdit(awx, awy)
			return
		}

		// Try to select a bot in the arena
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

			// Ctrl+click: clone bot genome to all others
			ctrlHeld := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
			if ctrlHeld && bestIdx >= 0 {
				g.cloneBotGenome(bestIdx)
				return
			}

			// Shift+click: set compare bot (if a primary bot is already selected)
			shiftHeld := ebiten.IsKeyPressed(ebiten.KeyShift)
			if shiftHeld && bestIdx >= 0 && ss.SelectedBot >= 0 && bestIdx != ss.SelectedBot {
				ss.CompareBot = bestIdx
				logger.Info("SWARM", "Compare bot #%d vs #%d", ss.SelectedBot, bestIdx)
			} else {
				ss.SelectedBot = bestIdx
				ss.CompareBot = -1 // normal click clears comparison
				// Cancel follow-cam if we selected a different bot
				if ss.FollowCamBot >= 0 && bestIdx != ss.FollowCamBot {
					ss.FollowCamBot = -1
				}
				if bestIdx >= 0 {
					logger.Info("SWARM", "Selected bot #%d", bestIdx)
				}
			}
		} else {
			ss.SelectedBot = -1
			ss.CompareBot = -1
		}
	}
}

// cloneBotGenome copies a bot's genome (params/weights/program) to all other bots.
func (g *Game) cloneBotGenome(srcIdx int) {
	ss := g.sim.SwarmState
	src := &ss.Bots[srcIdx]
	cloned := 0

	for i := range ss.Bots {
		if i == srcIdx {
			continue
		}
		// Clone evolution params
		if ss.EvolutionOn {
			ss.Bots[i].ParamValues = src.ParamValues
			cloned++
		}
		// Clone neuro brain weights
		if ss.NeuroEnabled && src.Brain != nil {
			if ss.Bots[i].Brain == nil {
				ss.Bots[i].Brain = &swarm.NeuroBrain{}
			}
			ss.Bots[i].Brain.Weights = src.Brain.Weights
			cloned++
		}
		// Clone GP program
		if ss.GPEnabled && src.OwnProgram != nil {
			ss.Bots[i].OwnProgram = src.OwnProgram // share program (read-only between evolutions)
			cloned++
		}
	}
	ss.SelectedBot = srcIdx
	logger.Info("SWARM", "Cloned bot #%d genome to %d bots", srcIdx, cloned)
}

// handleArenaEdit handles arena clicks when arena edit mode is active.
func (g *Game) handleArenaEdit(awx, awy float64) {
	ss := g.sim.SwarmState

	switch ss.ArenaEditTool {
	case 0: // Place obstacle
		obs := &physics.Obstacle{
			X: awx - 20,
			Y: awy - 20,
			W: 40,
			H: 40,
		}
		ss.Obstacles = append(ss.Obstacles, obs)
		ss.ObstaclesOn = true
		logger.Info("ARENA-EDIT", "Placed obstacle at (%.0f, %.0f)", awx, awy)

	case 1: // Place station
		if !ss.DeliveryOn {
			ss.DeliveryOn = true
			ss.Stations = nil
			ss.Packages = nil
		}
		// Cycle through colors: count existing stations to pick next color
		nextColor := (len(ss.Stations) % 4) + 1
		isPickup := len(ss.Stations)%2 == 0 // alternate pickup/dropoff
		st := swarm.DeliveryStation{
			X:        awx,
			Y:        awy,
			Color:    nextColor,
			IsPickup: isPickup,
		}
		if isPickup {
			st.HasPackage = true
			ss.Packages = append(ss.Packages, swarm.DeliveryPackage{
				Color:     nextColor,
				CarriedBy: -1,
				X:         awx,
				Y:         awy,
				Active:    true,
				SpawnTick: ss.Tick,
			})
		}
		ss.Stations = append(ss.Stations, st)
		kind := "Dropoff"
		if isPickup {
			kind = "Pickup"
		}
		logger.Info("ARENA-EDIT", "Placed %s station (color %d) at (%.0f, %.0f)", kind, nextColor, awx, awy)

	case 2: // Delete nearest object
		// Check obstacles first
		bestObsDist := 50.0
		bestObsIdx := -1
		for i, obs := range ss.Obstacles {
			cx := obs.X + obs.W/2
			cy := obs.Y + obs.H/2
			dx := cx - awx
			dy := cy - awy
			d := math.Sqrt(dx*dx + dy*dy)
			if d < bestObsDist {
				bestObsDist = d
				bestObsIdx = i
			}
		}
		// Check stations
		bestStDist := 50.0
		bestStIdx := -1
		for i := range ss.Stations {
			dx := ss.Stations[i].X - awx
			dy := ss.Stations[i].Y - awy
			d := math.Sqrt(dx*dx + dy*dy)
			if d < bestStDist {
				bestStDist = d
				bestStIdx = i
			}
		}
		// Delete whichever is closer
		if bestObsIdx >= 0 && (bestStIdx < 0 || bestObsDist < bestStDist) {
			ss.Obstacles = append(ss.Obstacles[:bestObsIdx], ss.Obstacles[bestObsIdx+1:]...)
			logger.Info("ARENA-EDIT", "Deleted obstacle at (%.0f, %.0f)", awx, awy)
			if len(ss.Obstacles) == 0 {
				ss.ObstaclesOn = false
			}
		} else if bestStIdx >= 0 {
			ss.Stations = append(ss.Stations[:bestStIdx], ss.Stations[bestStIdx+1:]...)
			logger.Info("ARENA-EDIT", "Deleted station at (%.0f, %.0f)", awx, awy)
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
		ss.IsDeliveryProgram = false
	}

	// Backspace
	if isKeyRepeating(ebiten.KeyBackspace) {
		if ed.CursorCol > 0 {
			line := ed.Lines[ed.CursorLine]
			ed.Lines[ed.CursorLine] = line[:ed.CursorCol-1] + line[ed.CursorCol:]
			ed.CursorCol--
			ss.ProgramName = "Custom"
			ss.IsDeliveryProgram = false
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
			ss.IsDeliveryProgram = false
		}
	}

	// Delete
	if isKeyRepeating(ebiten.KeyDelete) {
		line := ed.Lines[ed.CursorLine]
		if ed.CursorCol < len(line) {
			ed.Lines[ed.CursorLine] = line[:ed.CursorCol] + line[ed.CursorCol+1:]
			ss.ProgramName = "Custom"
			ss.IsDeliveryProgram = false
		} else if ed.CursorLine < len(ed.Lines)-1 {
			// Merge with next line
			nextLine := ed.Lines[ed.CursorLine+1]
			ed.Lines[ed.CursorLine] = line + nextLine
			ed.Lines = append(ed.Lines[:ed.CursorLine+1], ed.Lines[ed.CursorLine+2:]...)
			ss.ProgramName = "Custom"
			ss.IsDeliveryProgram = false
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
	ss.IsDeliveryProgram = false
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
			ss.ResetFlashTimer = 30
			logger.Info("SWARM", "Bot count changed to %d", count)
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

	// If block editor is active, serialize blocks to text first
	if ss.BlockEditorActive && len(ss.BlockRules) > 0 {
		text := serializeBlockRules(ss.BlockRules)
		ss.Editor.Lines = strings.Split(text, "\n")
	}

	source := strings.Join(ss.Editor.Lines, "\n")

	prog, err := swarmscript.ParseSwarmScript(source)
	if err != nil {
		ss.ErrorMsg = err.Error()
		// Try to extract line number from error
		ss.ErrorLine = 0
		logger.Warn("SWARM", "Parse error: %s", err.Error())
		return
	}

	ss.Program = prog
	ss.ProgramText = source
	ss.ErrorMsg = ""
	ss.ErrorLine = 0

	// Reset delivery state so counters/packages start fresh
	ss.ResetDeliveryState()

	// Blink all bots green to confirm deploy
	for i := range ss.Bots {
		ss.Bots[i].BlinkTimer = 30
	}

	logger.Info("SWARM", "Program deployed: %d rules", len(prog.Rules))
}

// autoReset resets bots and regenerates delivery state on toggle changes.
func countUsedParams(ss *swarm.SwarmState) int {
	n := 0
	for _, used := range ss.UsedParams {
		if used {
			n++
		}
	}
	return n
}

func (g *Game) autoReset(reason string) {
	ss := g.sim.SwarmState
	if ss == nil {
		return
	}
	ss.ResetBots()
	if ss.DeliveryOn {
		swarm.GenerateDeliveryStations(ss)
	}
	// Re-create truck state so a fresh truck arrives immediately
	if ss.TruckToggle {
		ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
	}
	// Re-initialize neuro brains after respawn
	if ss.NeuroEnabled {
		swarm.InitNeuro(ss)
	}
	ss.ResetFlashTimer = 30
	logger.Info("SWARM", "Auto-reset: %s", reason)
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

	// Delivery-program coupling
	if swarm.IsDeliveryPresetIdx(idx) || swarm.IsTruckPresetIdx(idx) {
		ss.IsDeliveryProgram = swarm.IsDeliveryPresetIdx(idx)
		ss.IsTruckProgram = swarm.IsTruckPresetIdx(idx)
		// Set truck toggle BEFORE maze generation so ramp is exempted
		if swarm.IsTruckPresetIdx(idx) {
			ss.TruckToggle = true
		} else {
			ss.TruckToggle = false
			ss.TruckState = nil
		}
		// Force delivery ON, obstacles OFF
		ss.DeliveryOn = true
		ss.ObstaclesOn = false
		ss.Obstacles = nil
		// Truck presets: maze OFF (open arena); delivery presets: maze ON
		if swarm.IsTruckPresetIdx(idx) {
			ss.MazeOn = false
			ss.MazeWalls = nil
		} else {
			ss.MazeOn = true
			swarm.GenerateSwarmMaze(ss)
		}
		swarm.GenerateDeliveryStations(ss)
		for i := range ss.Bots {
			ss.Bots[i].CarryingPkg = -1
		}
		// Truck preset: create truck state
		if swarm.IsTruckPresetIdx(idx) {
			ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
			logger.Info("SWARM", "Truck preset: trucks enabled (round %d)", ss.TruckState.RoundNum)
		}
	} else {
		ss.IsDeliveryProgram = false
		ss.IsTruckProgram = false
		ss.TruckToggle = false
		ss.TruckState = nil
		// Force delivery OFF for non-delivery presets
		if ss.DeliveryOn {
			ss.DeliveryOn = false
			ss.Stations = nil
			ss.Packages = nil
			ss.DeliveryStats = swarm.DeliveryStats{}
			for i := range ss.Bots {
				ss.Bots[i].CarryingPkg = -1
			}
		}
	}

	// Evolution-program coupling
	if swarm.IsEvolutionPresetIdx(idx) {
		ss.EvolutionOn = true
		ss.Generation = 0
		ss.EvolutionTimer = 0
		ss.FitnessHistory = nil
		g.sim.Speed = 5.0 // auto 5x speed for evolution
	} else {
		ss.EvolutionOn = false
	}

	// GP-program coupling
	if swarm.IsGPPresetIdx(idx) {
		ss.GPEnabled = true
		ss.EvolutionOn = false  // mutually exclusive
		ss.NeuroEnabled = false // mutually exclusive
		swarm.ClearNeuro(ss)
		g.sim.Speed = 5.0 // auto 5x speed
	} else {
		if ss.GPEnabled {
			swarm.ClearGP(ss)
		}
	}

	// Neuro-program coupling
	if swarm.IsNeuroPresetIdx(idx) {
		ss.NeuroEnabled = true
		ss.EvolutionOn = false // mutually exclusive
		ss.GPEnabled = false   // mutually exclusive
		swarm.ClearGP(ss)
		swarm.InitNeuro(ss)
		g.sim.Speed = 5.0 // auto 5x speed for neuro
		logger.Info("SWARM", "Neuro Delivery preset — NEURO auto-aktiviert, %d Bots × %d Gewichte",
			ss.BotCount, swarm.NeuroWeights)
	} else {
		if ss.NeuroEnabled {
			swarm.ClearNeuro(ss)
			ss.NeuroEnabled = false
		}
	}

	// Auto-deploy the preset
	prog, err := swarmscript.ParseSwarmScript(presetText)
	if err == nil {
		ss.Program = prog
		ss.ProgramText = presetText

		// Reset delivery state so counters/packages start fresh
		ss.ResetDeliveryState()

		// Initialize evolution params after program is parsed (needs UsedParams scan)
		if ss.EvolutionOn {
			swarm.InitBotParams(ss)
		}

		// Initialize GP after program is parsed (seed program for GP:Seeded)
		if ss.GPEnabled {
			if idx == 19 {
				// GP: Seeded Start — use the parsed program as seed
				swarm.InitGPSeeded(ss, prog)
			} else {
				swarm.InitGP(ss)
			}
		}

		for i := range ss.Bots {
			ss.Bots[i].BlinkTimer = 30
		}
		logger.Info("SWARM", "Preset '%s' loaded and deployed: %d rules", ss.ProgramName, len(prog.Rules))
	}
}

// --- Block editor helpers ---

func rulesToBlockRules(rules []swarmscript.Rule) []swarm.BlockRule {
	var result []swarm.BlockRule
	for _, r := range rules {
		br := swarm.BlockRule{
			ActionName:   swarmscript.ActionTypeName(r.Action.Type),
			ActionParams: [3]int{r.Action.Param1, r.Action.Param2, r.Action.Param3},
		}
		for _, c := range r.Conditions {
			br.Conditions = append(br.Conditions, swarm.BlockCondition{
				SensorName: swarmscript.ConditionTypeName(c.Type),
				OpStr:      swarmscript.OpString(c.Op),
				Value:      c.Value,
			})
		}
		if len(br.Conditions) == 0 {
			br.Conditions = append(br.Conditions, swarm.BlockCondition{
				SensorName: "true", OpStr: "==", Value: 1,
			})
		}
		result = append(result, br)
	}
	return result
}

func serializeBlockRules(blocks []swarm.BlockRule) string {
	var lines []string
	for _, br := range blocks {
		line := "IF "
		for i, cond := range br.Conditions {
			if i > 0 {
				line += " AND "
			}
			if cond.SensorName == "true" {
				line += "true"
			} else {
				line += fmt.Sprintf("%s %s %d", cond.SensorName, cond.OpStr, cond.Value)
			}
		}
		line += " THEN " + br.ActionName
		pc := swarmscript.ActionParamCountByName(br.ActionName)
		if pc >= 1 {
			line += fmt.Sprintf(" %d", br.ActionParams[0])
		}
		if pc >= 2 {
			line += fmt.Sprintf(" %d", br.ActionParams[1])
		}
		if pc >= 3 {
			line += fmt.Sprintf(" %d", br.ActionParams[2])
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (g *Game) handleBlockEditorClick(mx, my int) {
	ss := g.sim.SwarmState

	// If dropdown is open, check for dropdown click first
	if ss.ActiveDropdown != nil {
		idx := render.BlockDropdownHitTest(mx, my, ss.ActiveDropdown)
		if idx >= 0 {
			g.applyBlockDropdownSelection(idx)
		}
		ss.ActiveDropdown = nil
		return
	}

	action, ri, ci := render.BlockEditorHitTest(mx, my, ss)
	switch action {
	case "sensor":
		// Open sensor dropdown
		items := flattenGroups(swarmscript.SensorGrouped)
		ddX := blockEditorDropdownX("sensor")
		ddY := my
		ss.ActiveDropdown = &swarm.BlockDropdown{
			RuleIdx: ri, CondIdx: ci, FieldType: "sensor",
			X: ddX, Y: ddY, Items: items, HoverIdx: -1,
		}

	case "op":
		items := []string{">", "<", "=="}
		ss.ActiveDropdown = &swarm.BlockDropdown{
			RuleIdx: ri, CondIdx: ci, FieldType: "op",
			X: blockEditorDropdownX("op"), Y: my, Items: items, HoverIdx: -1,
		}

	case "value":
		ss.BlockValueEdit = true
		ss.BlockValueRuleIdx = ri
		ss.BlockValueCondIdx = ci
		ss.BlockValueText = fmt.Sprintf("%d", ss.BlockRules[ri].Conditions[ci].Value)
		ss.Editor.Focused = false

	case "action":
		items := flattenGroups(swarmscript.ActionGrouped)
		ss.ActiveDropdown = &swarm.BlockDropdown{
			RuleIdx: ri, CondIdx: -1, FieldType: "action",
			X: blockEditorDropdownX("action"), Y: my, Items: items, HoverIdx: -1,
		}

	case "delete":
		if ri >= 0 && ri < len(ss.BlockRules) {
			ss.BlockRules = append(ss.BlockRules[:ri], ss.BlockRules[ri+1:]...)
		}

	case "add_cond":
		if ri >= 0 && ri < len(ss.BlockRules) {
			ss.BlockRules[ri].Conditions = append(ss.BlockRules[ri].Conditions, swarm.BlockCondition{
				SensorName: "true", OpStr: "==", Value: 1,
			})
		}

	case "new_rule":
		ss.BlockRules = append(ss.BlockRules, swarm.BlockRule{
			Conditions:   []swarm.BlockCondition{{SensorName: "true", OpStr: "==", Value: 1}},
			ActionName:   "FWD",
			ActionParams: [3]int{},
		})
	}
}

func (g *Game) applyBlockDropdownSelection(idx int) {
	ss := g.sim.SwarmState
	dd := ss.ActiveDropdown
	if dd == nil || idx < 0 || idx >= len(dd.Items) {
		return
	}
	selected := dd.Items[idx]
	ri := dd.RuleIdx
	ci := dd.CondIdx

	if ri < 0 || ri >= len(ss.BlockRules) {
		return
	}

	switch dd.FieldType {
	case "sensor":
		if ci >= 0 && ci < len(ss.BlockRules[ri].Conditions) {
			ss.BlockRules[ri].Conditions[ci].SensorName = selected
		}
	case "op":
		if ci >= 0 && ci < len(ss.BlockRules[ri].Conditions) {
			ss.BlockRules[ri].Conditions[ci].OpStr = selected
		}
	case "action":
		ss.BlockRules[ri].ActionName = selected
		ss.BlockRules[ri].ActionParams = [3]int{}
	}
}

func flattenGroups(groups [][]string) []string {
	var result []string
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

func (g *Game) handleBlockValueInput() {
	ss := g.sim.SwarmState

	// Accept digits and minus
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if (ch >= '0' && ch <= '9') || (ch == '-' && len(ss.BlockValueText) == 0) {
			ss.BlockValueText += string(ch)
		}
	}

	// Backspace
	if isKeyRepeating(ebiten.KeyBackspace) && len(ss.BlockValueText) > 0 {
		ss.BlockValueText = ss.BlockValueText[:len(ss.BlockValueText)-1]
	}

	// Enter or Escape: commit
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ri := ss.BlockValueRuleIdx
		ci := ss.BlockValueCondIdx
		if ri >= 0 && ri < len(ss.BlockRules) && ci >= 0 && ci < len(ss.BlockRules[ri].Conditions) {
			val, err := strconv.Atoi(ss.BlockValueText)
			if err == nil {
				ss.BlockRules[ri].Conditions[ci].Value = val
			}
		}
		ss.BlockValueEdit = false
	}
}

func blockEditorDropdownX(fieldType string) int {
	switch fieldType {
	case "sensor":
		return 4 + 20 // blockPadX + blockIfW
	case "op":
		return 4 + 20 + 72 + 2 // after sensor
	case "action":
		return 4 + 20 + 72 + 22 + 30 + 8 + 30 // after THEN
	}
	return 30
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

func (g *Game) updateTutorial() {
	tut := &g.tutorial
	if !tut.Active || tut.Step < 0 || tut.Step >= len(render.TutorialSteps) {
		return
	}
	tut.PulseTimer++
	step := render.TutorialSteps[tut.Step]

	mx, my := ebiten.CursorPosition()

	// ESC to skip tutorial
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		tut.Active = false
		tut.Dismissed = true
		render.MarkTutorialDone()
		logger.Info("TUTORIAL", "Skipped at step %d", tut.Step+1)
		return
	}

	// Click handling for Weiter/Skip buttons
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		hit := render.TutorialWeiterHitTest(mx, my, screenW, screenH, tut)
		if hit == "skip" {
			tut.Active = false
			tut.Dismissed = true
			render.MarkTutorialDone()
			logger.Info("TUTORIAL", "Skipped via button at step %d", tut.Step+1)
			return
		}
		if hit == "weiter" {
			g.advanceTutorial()
			return
		}
	}

	// Enter/Space to advance (if no input needed or input done)
	if step.WaitInput == "" || tut.InputDone {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.advanceTutorial()
			return
		}
	}

	// Check step-specific input
	switch step.WaitInput {
	case "timer:300":
		tut.WaitTimer++
		if tut.WaitTimer >= 300 {
			tut.InputDone = true
		}
	case "key:F2":
		if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
			tut.InputDone = true
		}
	case "key:F":
		if inpututil.IsKeyJustPressed(ebiten.KeyF) {
			tut.InputDone = true
		}
	case "key:H":
		if inpututil.IsKeyJustPressed(ebiten.KeyH) {
			tut.InputDone = true
		}
	case "click:deploy_any":
		// Detected by handleSwarmClick when deploy is clicked
		// We check if it was just deployed by looking at blink timer
		if g.sim.SwarmMode && g.sim.SwarmState != nil {
			for _, b := range g.sim.SwarmState.Bots {
				if b.BlinkTimer == 30 {
					tut.InputDone = true
					break
				}
			}
		}
	case "click:delivery":
		if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.DeliveryOn {
			tut.InputDone = true
		}
	case "click:bot":
		if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.SelectedBot >= 0 {
			tut.InputDone = true
		}
	case "click:blocks":
		if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.BlockEditorActive {
			tut.InputDone = true
		}
	}

	// Auto-advance when input done
	if tut.InputDone && step.WaitInput != "" {
		// Small delay before auto-advance for some step types
		if step.WaitInput == "key:H" {
			// Close help overlay that was just opened
			g.showHelp = false
		}
		g.advanceTutorial()
	}
}

func (g *Game) maybeStartTutorial() {
	if !render.IsTutorialDone() {
		g.tutorial.Active = true
		g.tutorial.Step = 0
		g.tutorial.InputDone = false
		g.tutorial.WaitTimer = 0
		g.tutorial.Dismissed = false
		logger.Info("TUTORIAL", "Auto-started (first launch)")
	}
}

func (g *Game) advanceTutorial() {
	tut := &g.tutorial
	tut.Step++
	tut.InputDone = false
	tut.WaitTimer = 0
	if tut.Step >= len(render.TutorialSteps) {
		tut.Active = false
		tut.Dismissed = true
		render.MarkTutorialDone()
		logger.Info("TUTORIAL", "Completed!")
	} else {
		logger.Info("TUTORIAL", "Step %d/%d", tut.Step+1, len(render.TutorialSteps))
	}
}

// Draw renders the simulation.
func (g *Game) Draw(screen *ebiten.Image) {
	// Welcome screen
	if g.showWelcome {
		if !g.welcomeReady {
			g.renderer.InitWelcomeBots(screenW, screenH)
			g.welcomeReady = true
		}
		g.renderer.DrawWelcomeScreen(screen, g.welcomeTick)
		return
	}

	g.renderer.Draw(screen, g.sim)
	render.DrawHUD(screen, g.sim, ebiten.ActualFPS(), g.renderer)

	// Step mode indicator
	if g.stepMode {
		render.DrawStepModeIndicator(screen)
	}

	// Replay overlay
	if g.replayMode && g.sim.SwarmState != nil && g.sim.SwarmState.ReplayBuf != nil {
		render.DrawReplayOverlay(screen, g.sim.SwarmState, g.replayIdx)
	}

	// Sound: ambient volume + collision clicks
	if g.renderer.Sound != nil && g.renderer.Sound.Enabled {
		botCount := len(g.sim.Bots)
		if g.sim.SwarmMode && g.sim.SwarmState != nil {
			botCount = len(g.sim.SwarmState.Bots)
		}
		g.renderer.Sound.SetBotCount(botCount)

		// Collision click (swarm mode, throttled inside PlayCollision)
		if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.CollisionCount > 0 {
			g.renderer.Sound.PlayCollision()
		}
	}

	// Screenshot (capture after full render including HUD)
	if g.screenshotRequested {
		g.screenshotRequested = false
		fname := render.CaptureScreenshot(screen)
		if fname != "" {
			g.renderer.OverlayText = "Screenshot saved: " + fname
			g.renderer.OverlayTimer = 60
		}
	}

	// GIF recording toggle
	if g.gifToggleRequested {
		g.gifToggleRequested = false
		if g.renderer.Recording {
			render.StopRecording(g.renderer) // async: encodes in goroutine
		} else if !g.renderer.GIFEncoding {
			render.StartRecording(g.renderer)
		}
	}

	// Capture GIF frame if recording
	if g.renderer.Recording {
		if render.CaptureGIFFrame(screen, g.renderer) {
			// Max frames reached — auto-stop
			render.StopRecording(g.renderer) // async: encodes in goroutine
		}
	}

	// In-game log console
	if g.showConsole {
		filterID := g.consoleFilterBot
		var logEntries []logger.LogEntry
		if filterID >= 0 && g.sim.SwarmMode {
			logEntries = logger.EntriesForBot(filterID)
		} else {
			logEntries = logger.Entries()
			filterID = -1
		}
		render.DrawConsole(screen, logEntries, g.sim.SwarmMode, filterID)
	}

	// Help overlay (drawn on top of everything, including console)
	if g.showHelp {
		render.DrawHelpOverlay(screen, g.sim.SwarmMode, g.helpScrollY)
	}

	// Tooltips (below tutorial overlay)
	if g.sim.SwarmMode && g.tooltip.Visible {
		render.DrawTooltip(screen, &g.tooltip)
	}

	// Bot hover tooltip
	if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.HoveredBot >= 0 {
		bmx, bmy := ebiten.CursorPosition()
		render.DrawBotTooltip(screen, g.sim.SwarmState, bmx, bmy)
	}

	// Tutorial overlay (on top of everything)
	if g.tutorial.Active {
		render.DrawTutorial(screen, &g.tutorial, 0)
	}

	// Panic error banner
	if g.panicTimer > 0 && g.panicMsg != "" {
		render.DrawPanicBanner(screen, g.panicMsg, g.panicTimer)
	}
}

// buildStatsCSV builds a CSV export of fitness history and delivery stats.
func (g *Game) buildStatsCSV() string {
	ss := g.sim.SwarmState
	if ss == nil {
		return ""
	}
	var b strings.Builder

	// Fitness history
	if len(ss.FitnessHistory) > 0 {
		b.WriteString("# FITNESS HISTORY\n")
		b.WriteString("Generation,BestFitness,AvgFitness\n")
		for i, h := range ss.FitnessHistory {
			b.WriteString(fmt.Sprintf("%d,%.2f,%.2f\n", i+1, h.Best, h.Avg))
		}
		b.WriteString("\n")
	}

	// Delivery stats
	if ss.DeliveryOn {
		ds := &ss.DeliveryStats
		b.WriteString("# DELIVERY STATS\n")
		b.WriteString(fmt.Sprintf("TotalDelivered,%d\n", ds.TotalDelivered))
		b.WriteString(fmt.Sprintf("CorrectDelivered,%d\n", ds.CorrectDelivered))
		b.WriteString(fmt.Sprintf("WrongDelivered,%d\n", ds.WrongDelivered))
		b.WriteString(fmt.Sprintf("Tick,%d\n", ss.Tick))
		b.WriteString("\n")
	}

	// Delivery rate buckets
	if ss.StatsTracker != nil && len(ss.StatsTracker.DeliveryBuckets) > 0 {
		b.WriteString("# DELIVERY RATE (per 500 ticks)\n")
		b.WriteString("Window,Total,Correct,Wrong\n")
		st := ss.StatsTracker
		for i := 0; i < len(st.DeliveryBuckets); i++ {
			correct := 0
			wrong := 0
			if i < len(st.CorrectBuckets) {
				correct = st.CorrectBuckets[i]
			}
			if i < len(st.WrongBuckets) {
				wrong = st.WrongBuckets[i]
			}
			b.WriteString(fmt.Sprintf("%d,%d,%d,%d\n", i+1, st.DeliveryBuckets[i], correct, wrong))
		}
		b.WriteString("\n")
	}

	// Bot rankings
	if ss.StatsTracker != nil && len(ss.StatsTracker.BotRankings) > 0 {
		b.WriteString("# BOT RANKINGS\n")
		b.WriteString("Rank,BotIdx,Deliveries,AvgTime\n")
		for i, r := range ss.StatsTracker.BotRankings {
			if i >= 20 {
				break
			}
			b.WriteString(fmt.Sprintf("%d,%d,%d,%d\n", i+1, r.BotIdx, r.Deliveries, r.AvgTime))
		}
		b.WriteString("\n")
	}

	// Diversity
	if ss.Diversity != nil {
		b.WriteString("# DIVERSITY\n")
		b.WriteString(fmt.Sprintf("AvgDistance,%.4f\n", ss.Diversity.AvgDistance))
		b.WriteString(fmt.Sprintf("MinDistance,%.4f\n", ss.Diversity.MinDistance))
		b.WriteString(fmt.Sprintf("UniqueGenotypes,%d\n", ss.Diversity.UniqueCount))
		b.WriteString(fmt.Sprintf("Stagnant,%v\n", ss.Diversity.Stagnant))
	}

	return b.String()
}

// Layou returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}

func main() {
	defer logger.CloseLog()
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error("CRASH", "Panic: %v\n%s", r, stack)
			fmt.Fprintf(os.Stderr, "FATAL: %v\n", r)
		}
	}()

	logger.Info("INIT", "SwarmSim starting")

	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Schwarm-Robotik-Simulator")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
	StopProfile() // ensure profiling stops on clean exit
	logger.Info("INIT", "SwarmSim exiting cleanly")
}
