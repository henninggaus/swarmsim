package main

import (
	"fmt"
	"math"
	"strings"
	"swarmsim/domain/factory"
	"swarmsim/domain/swarm"
	"swarmsim/locale"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
	"swarmsim/render"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

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

	// ESC to quit (or close overlays / cancel follow-cam / deselect first)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.showWelcome {
			return ebiten.Termination
		}
		if g.showHelp {
			g.showHelp = false
			return nil
		}
		// Close active educational overlay before anything else
		if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.ActiveOverlay != "" {
			toggleOverlay(g.sim.SwarmState, "") // close current overlay
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
		// Factory mode: deselect bot before quitting
		if g.sim.FactoryMode && g.sim.FactoryState != nil && g.sim.FactoryState.SelectedBot >= 0 {
			g.sim.FactoryState.SelectedBot = -1
			g.sim.FactoryState.FollowCamBot = -1
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
		// F3: Swarm Lab + Tutorial
		if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
			g.sim.LoadSwarmScenario()
			g.tickAcc = 0
			g.showWelcome = false
			g.maybeStartTutorial()
			return nil
		}
		// F5: Factory Mode
		if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
			g.sim.LoadFactoryScenario()
			g.tickAcc = 0
			g.showWelcome = false
			return nil
		}
		// F4: Algo-Labor (dedicated optimization algorithm visualization)
		if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
			g.sim.LoadSwarmScenario()
			g.tickAcc = 0
			g.showWelcome = false
			ss := g.sim.SwarmState
			ss.AlgoLaborMode = true
			ss.DeliveryOn = false
			ss.TruckToggle = false
			ss.NeuroEnabled = false
			ss.GPEnabled = false
			ss.EvolutionOn = false
			// Initialize fitness landscape
			if ss.SwarmAlgo == nil {
				ss.SwarmAlgo = &swarm.SwarmAlgorithmState{}
			}
			swarm.InitAlgoLabor(ss)
			return nil
		}

		// Mouse click on welcome screen -- check which button was hit
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			// Factory button
			btn5 := g.renderer.WelcomeBtn5
			if btn5[2] > 0 && mx >= btn5[0] && mx < btn5[0]+btn5[2] && my >= btn5[1] && my < btn5[1]+btn5[3] {
				g.sim.LoadFactoryScenario()
				g.tickAcc = 0
				g.showWelcome = false
				return nil
			}
			// Algo-Labor button
			btn4 := g.renderer.WelcomeBtn4
			if btn4[2] > 0 && mx >= btn4[0] && mx < btn4[0]+btn4[2] && my >= btn4[1] && my < btn4[1]+btn4[3] {
				// Algo-Labor button clicked
				g.sim.LoadSwarmScenario()
				g.tickAcc = 0
				g.showWelcome = false
				ss := g.sim.SwarmState
				ss.AlgoLaborMode = true
				ss.DeliveryOn = false
				ss.TruckToggle = false
				ss.NeuroEnabled = false
				ss.GPEnabled = false
				ss.EvolutionOn = false
				if ss.SwarmAlgo == nil {
					ss.SwarmAlgo = &swarm.SwarmAlgorithmState{}
				}
				swarm.InitAlgoLabor(ss)
				return nil
			}
			// Default: load Swarm Lab
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

	if g.sim.FactoryMode && g.sim.FactoryState != nil {
		g.handleFactoryInput()
	} else if g.sim.SwarmMode && g.sim.SwarmState != nil {
		g.handleSwarmInput()
	} else {
		g.handleInput()
		g.handleCamera()
	}

	// Cursor shape: pointer when over editor panel buttons, default otherwise
	{
		mx, _ := ebiten.CursorPosition()
		if g.sim.SwarmMode && g.sim.SwarmState != nil && mx < 350 {
			ebiten.SetCursorShape(ebiten.CursorShapePointer)
		} else if g.sim.FactoryMode && g.sim.FactoryState != nil && mx < 300 {
			ebiten.SetCursorShape(ebiten.CursorShapePointer)
		} else {
			ebiten.SetCursorShape(ebiten.CursorShapeDefault)
		}
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
	// Session save flash timer
	if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.SessionSaveFlash > 0 {
		g.sim.SwarmState.SessionSaveFlash--
	}
	// Deploy success flash timer
	if g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.DeployFlash > 0 {
		g.sim.SwarmState.DeployFlash--
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

	// Attach telemetry writer to SwarmState when swarm mode starts
	if g.telemetryWriter != nil && g.sim.SwarmMode && g.sim.SwarmState != nil && g.sim.SwarmState.Telemetry == nil {
		g.sim.SwarmState.Telemetry = g.telemetryWriter
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
			g.sim.Speed *= 2 // 0.125 -> 0.25 -> 0.5 -> 1.0
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
				g.sim.Speed /= 2 // 1.0 -> 0.5 -> 0.25 -> 0.125
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
			ebiten.SetWindowTitle(locale.T("ui.window_title") + " \u2014 Classic")
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
			ebiten.SetWindowTitle(locale.T("ui.window_title") + " \u2014 Swarm Lab")
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
			ebiten.SetWindowTitle(locale.T("ui.window_title") + " \u2014 Swarm Lab")
		}
	}

	// F4: Algo-Labor (dedicated optimization algorithm mode)
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) && g.renderer.FadeDir == 0 {
		logger.Info("KEY", "F4 pressed -> Loading Algo-Labor")
		g.renderer.FadeDir = -1
		g.renderer.FadeAlpha = 0
		g.renderer.FadeLoad = func() {
			g.sim.LoadSwarmScenario()
			g.tickAcc = 0
			ss := g.sim.SwarmState
			ss.AlgoLaborMode = true
			ss.DeliveryOn = false
			ss.TruckToggle = false
			ss.NeuroEnabled = false
			ss.GPEnabled = false
			ss.EvolutionOn = false
			if ss.SwarmAlgo == nil {
				ss.SwarmAlgo = &swarm.SwarmAlgorithmState{}
			}
			swarm.InitAlgoLabor(ss)
			ebiten.SetWindowTitle(locale.T("ui.window_title") + " \u2014 Algo-Labor")
		}
	}

	// F5: Factory Mode (logistics simulation)
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) && g.renderer.FadeDir == 0 {
		logger.Info("KEY", "F5 pressed -> Loading Factory Mode")
		g.renderer.FadeDir = -1
		g.renderer.FadeAlpha = 0
		g.renderer.FadeLoad = func() {
			g.sim.LoadFactoryScenario()
			g.tickAcc = 0
			ebiten.SetWindowTitle(locale.T("ui.window_title") + " \u2014 Factory")
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

	// Backquote (`) or Semicolon: toggle in-game log console
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

// --- Tutorial ---

func (g *Game) updateTutorial() {
	tut := &g.tutorial
	steps := render.GetTutorialSteps()
	if !tut.Active || tut.Step < 0 || tut.Step >= len(steps) {
		return
	}
	tut.PulseTimer++
	step := steps[tut.Step]

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
	if tut.Step >= len(render.GetTutorialSteps()) {
		tut.Active = false
		tut.Dismissed = true
		render.MarkTutorialDone()
		logger.Info("TUTORIAL", "Completed!")
	} else {
		logger.Info("TUTORIAL", "Step %d/%d", tut.Step+1, len(render.GetTutorialSteps()))
	}
}

// --- Factory Mode Input ---

func (g *Game) handleFactoryInput() {
	fs := g.sim.FactoryState
	if fs == nil {
		return
	}

	// Auto-start factory tutorial on first entry
	if fs.FactoryTutorial == nil && fs.ShowFactoryTut {
		fs.FactoryTutorial = &factory.FactoryTutorial{Active: true, Step: 0}
		fs.ShowFactoryTut = false
	}

	// Factory tutorial click handling
	if fs.FactoryTutorial != nil && fs.FactoryTutorial.Active {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			sw, sh := ebiten.WindowSize()
			action := render.FactoryTutHitTest(mx, my, sw, sh, fs.FactoryTutorial)
			switch action {
			case "next":
				fs.FactoryTutorial.Step++
				if fs.FactoryTutorial.Step >= 10 {
					fs.FactoryTutorial.Active = false
				}
			case "skip":
				fs.FactoryTutorial.Active = false
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fs.FactoryTutorial.Active = false
		}
	}

	g.handleFactoryCameraInput(fs)
	g.handleFactorySpeedInput(fs)
	g.handleFactoryBotManagement(fs)
	g.handleFactoryToggles(fs)
	g.handleFactoryMouseInput(fs)
}

// handleFactoryCameraInput handles WASD pan, mouse-wheel zoom, right-drag pan,
// follow-cam lerp, and camera clamping.
func (g *Game) handleFactoryCameraInput(fs *factory.FactoryState) {
	// WASD camera pan
	panSpeed := 8.0 / fs.CamZoom
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		fs.CamY -= panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) && !inpututil.IsKeyJustPressed(ebiten.KeyS) {
		fs.CamY += panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		fs.CamX -= panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		fs.CamX += panSpeed
	}

	// Mouse wheel zoom
	_, wy := ebiten.Wheel()
	if wy > 0 {
		fs.CamZoom *= 1.1
		if fs.CamZoom > 3.0 {
			fs.CamZoom = 3.0
		}
	} else if wy < 0 {
		fs.CamZoom /= 1.1
		if fs.CamZoom < 0.15 {
			fs.CamZoom = 0.15
		}
	}

	// Right-drag pan
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mx, my := ebiten.CursorPosition()
		g.dragging = true
		g.dragStartX = mx
		g.dragStartY = my
		g.camStartX = fs.CamX
		g.camStartY = fs.CamY
	}
	if g.dragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		mx, my := ebiten.CursorPosition()
		dx := float64(mx-g.dragStartX) / fs.CamZoom
		dy := float64(my-g.dragStartY) / fs.CamZoom
		fs.CamX = g.camStartX - dx
		fs.CamY = g.camStartY - dy
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		g.dragging = false
	}

	// Follow-cam smooth lerp
	if fs.FollowCamBot >= 0 && fs.FollowCamBot < len(fs.Bots) {
		bot := &fs.Bots[fs.FollowCamBot]
		fs.CamX += (bot.X - fs.CamX) * 0.10
		fs.CamY += (bot.Y - fs.CamY) * 0.10
		fs.CamZoom += (1.5 - fs.CamZoom) * 0.05
	}

	// Key 0: zoom-to-fit all factory bots
	if inpututil.IsKeyJustPressed(ebiten.Key0) {
		g.factoryZoomToFit(fs)
	}

	// Clamp camera to world bounds
	if fs.CamX < 0 {
		fs.CamX = 0
	}
	if fs.CamX > float64(3000) {
		fs.CamX = float64(3000)
	}
	if fs.CamY < 0 {
		fs.CamY = 0
	}
	if fs.CamY > float64(2000) {
		fs.CamY = float64(2000)
	}
}

// handleFactorySpeedInput handles Space (pause/resume) and 1-5 speed presets.
func (g *Game) handleFactorySpeedInput(fs *factory.FactoryState) {
	// Space = pause/resume
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		fs.Paused = !fs.Paused
	}

	// Speed controls: 1-5 keys for presets
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		fs.Speed = 1.0
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		fs.Speed = 2.0
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		fs.Speed = 5.0
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		fs.Speed = 10.0
	}
	if inpututil.IsKeyJustPressed(ebiten.Key5) {
		fs.Speed = 20.0
	}
}

// handleFactoryBotManagement handles B (buy), V (sell), +/- (add/remove 100 bots).
func (g *Game) handleFactoryBotManagement(fs *factory.FactoryState) {
	// [13] Bot count slider: +/= to add 100 bots, - to remove 100
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) { // = or + key
		newCount := fs.BotCount + 100
		if newCount > 2000 {
			newCount = 2000
		}
		for len(fs.Bots) < newCount {
			// Determine role for new bot
			r := fs.Rng.Float64()
			role := factory.RoleTransporter
			spd := factory.FactoryBotSpeed
			energy := 100.0
			if r < 0.15 {
				role = factory.RoleExpress
				spd = 5.0
				energy = 60
			} else if r < 0.40 {
				role = factory.RoleForklift
				spd = 2.0
				energy = 120
			}
			fs.Bots = append(fs.Bots, swarm.SwarmBot{
				X:      factory.HallX + 50 + fs.Rng.Float64()*(factory.HallW-100),
				Y:      factory.HallY + 50 + fs.Rng.Float64()*(factory.HallH-100),
				Speed:  spd,
				Angle:  fs.Rng.Float64() * 6.283,
				Energy: energy,
				CarryingPkg: -1,
			})
			fs.Malfunctioning = append(fs.Malfunctioning, false)
			fs.BotRoles = append(fs.BotRoles, role)
			fs.BotOpHours = append(fs.BotOpHours, 0)
			fs.BotDeliveries = append(fs.BotDeliveries, 0)
			fs.ShiftOnDuty = append(fs.ShiftOnDuty, true)
			fs.BatterySwapTimer = append(fs.BatterySwapTimer, 0)
		}
		fs.BotCount = len(fs.Bots)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		newCount := fs.BotCount - 100
		if newCount < 100 {
			newCount = 100
		}
		if newCount < len(fs.Bots) {
			// Deselect if selected bot is being removed
			if fs.SelectedBot >= newCount {
				fs.SelectedBot = -1
				fs.FollowCamBot = -1
			}
			fs.Bots = fs.Bots[:newCount]
			if len(fs.Malfunctioning) > newCount {
				fs.Malfunctioning = fs.Malfunctioning[:newCount]
			}
			if len(fs.BotRoles) > newCount {
				fs.BotRoles = fs.BotRoles[:newCount]
			}
			if len(fs.BotOpHours) > newCount {
				fs.BotOpHours = fs.BotOpHours[:newCount]
			}
			if len(fs.BotDeliveries) > newCount {
				fs.BotDeliveries = fs.BotDeliveries[:newCount]
			}
			if len(fs.ShiftOnDuty) > newCount {
				fs.ShiftOnDuty = fs.ShiftOnDuty[:newCount]
			}
			if len(fs.BatterySwapTimer) > newCount {
				fs.BatterySwapTimer = fs.BatterySwapTimer[:newCount]
			}
		}
		fs.BotCount = len(fs.Bots)
	}

	// Feature: Buy 10 bots for $500 each (B key)
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		cost := 10 * 500.0
		if fs.Budget >= cost && len(fs.Bots) < 2000 {
			fs.Budget -= cost
			for j := 0; j < 10; j++ {
				r := fs.Rng.Float64()
				role := factory.RoleTransporter
				spd := factory.FactoryBotSpeed
				energy := 100.0
				if r < 0.15 {
					role = factory.RoleExpress
					spd = 5.0
					energy = 60
				} else if r < 0.40 {
					role = factory.RoleForklift
					spd = 2.0
					energy = 120
				}
				fs.Bots = append(fs.Bots, swarm.SwarmBot{
					X:           factory.HallX + 50 + fs.Rng.Float64()*(factory.HallW-100),
					Y:           factory.HallY + 50 + fs.Rng.Float64()*(factory.HallH-100),
					Speed:       spd,
					Angle:       fs.Rng.Float64() * 6.283,
					Energy:      energy,
					CarryingPkg: -1,
				})
				fs.Malfunctioning = append(fs.Malfunctioning, false)
				fs.BotRoles = append(fs.BotRoles, role)
				fs.BotOpHours = append(fs.BotOpHours, 0)
				fs.BotDeliveries = append(fs.BotDeliveries, 0)
				fs.ShiftOnDuty = append(fs.ShiftOnDuty, true)
				fs.BatterySwapTimer = append(fs.BatterySwapTimer, 0)
			}
			fs.BotCount = len(fs.Bots)
			factory.AddAlert(fs, locale.T("factory.alert.bought_bots"), [3]uint8{60, 200, 60})
		}
	}

	// Feature: Sell 10 idle bots for $200 each (V key)
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		if len(fs.Bots) > 110 {
			// Find and remove last 10 idle bots
			removed := 0
			for i := len(fs.Bots) - 1; i >= 0 && removed < 10; i-- {
				if fs.Bots[i].State == factory.BotIdle || fs.Bots[i].State == factory.BotOffShift {
					// Deselect if needed
					if fs.SelectedBot == i {
						fs.SelectedBot = -1
						fs.FollowCamBot = -1
					}
					// Swap-remove from all per-bot slices
					last := len(fs.Bots) - 1
					fs.Bots[i] = fs.Bots[last]
					fs.Bots = fs.Bots[:last]
					if i < len(fs.Malfunctioning) && last < len(fs.Malfunctioning) {
						fs.Malfunctioning[i] = fs.Malfunctioning[last]
						fs.Malfunctioning = fs.Malfunctioning[:last]
					}
					if i < len(fs.BotRoles) && last < len(fs.BotRoles) {
						fs.BotRoles[i] = fs.BotRoles[last]
						fs.BotRoles = fs.BotRoles[:last]
					}
					if i < len(fs.BotOpHours) && last < len(fs.BotOpHours) {
						fs.BotOpHours[i] = fs.BotOpHours[last]
						fs.BotOpHours = fs.BotOpHours[:last]
					}
					if i < len(fs.BotDeliveries) && last < len(fs.BotDeliveries) {
						fs.BotDeliveries[i] = fs.BotDeliveries[last]
						fs.BotDeliveries = fs.BotDeliveries[:last]
					}
					if i < len(fs.ShiftOnDuty) && last < len(fs.ShiftOnDuty) {
						fs.ShiftOnDuty[i] = fs.ShiftOnDuty[last]
						fs.ShiftOnDuty = fs.ShiftOnDuty[:last]
					}
					if i < len(fs.BatterySwapTimer) && last < len(fs.BatterySwapTimer) {
						fs.BatterySwapTimer[i] = fs.BatterySwapTimer[last]
						fs.BatterySwapTimer = fs.BatterySwapTimer[:last]
					}
					removed++
				}
			}
			if removed > 0 {
				fs.Budget += float64(removed) * 200
				fs.BotCount = len(fs.Bots)
				factory.AddAlert(fs, locale.Tf("factory.alert.sold_bots", removed, removed*200), [3]uint8{200, 200, 60})
			}
		}
	}
}

// handleFactoryToggles handles H/M/P/E/T/X/F toggle keys.
func (g *Game) handleFactoryToggles(fs *factory.FactoryState) {
	// [14] Truck manual dispatch: T = inbound, Shift+T = outbound
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			// Force spawn outbound truck
			factory.ForceSpawnOutboundTruck(fs)
		} else {
			// Force spawn inbound truck
			factory.ForceSpawnInboundTruck(fs)
		}
	}

	// Help toggle: H key (replaces heatmap)
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		if fs.ShowHelp {
			fs.ShowHelp = false
		} else if fs.ShowHeatmap {
			fs.ShowHeatmap = false
		} else {
			fs.ShowHelp = true
		}
	}

	// [3] Heatmap toggle: M key
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		fs.ShowHeatmap = !fs.ShowHeatmap
	}

	// Emergency toggle: E key
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		factory.ToggleEmergency(fs)
	}

	// Feature 10: Maintenance planner toggle: P key
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		fs.ShowMaintPlanner = !fs.ShowMaintPlanner
	}

	// X: copy factory stats to clipboard
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		report := factory.GenerateStatsReport(fs)
		render.ClipboardWrite(report)
		fs.ClipboardFlash = 120 // show flash for 2 seconds
		factory.AddAlert(fs, "Stats copied to clipboard!", [3]uint8{100, 255, 200})
	}

	// F: toggle follow-cam on selected bot
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if fs.SelectedBot >= 0 {
			if fs.FollowCamBot == fs.SelectedBot {
				fs.FollowCamBot = -1 // toggle off
			} else {
				fs.FollowCamBot = fs.SelectedBot
			}
		}
	}
}

// handleFactoryMouseInput handles bot selection, machine click, minimap click,
// and hover tooltip detection.
func (g *Game) handleFactoryMouseInput(fs *factory.FactoryState) {
	// Left-click: select bot or toggle machine
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		// Convert screen to world coordinates
		worldX := (float64(mx) - float64(screenW)/2) / fs.CamZoom + fs.CamX
		worldY := (float64(my) - float64(screenH)/2) / fs.CamZoom + fs.CamY

		// [4] Check minimap click (bottom-right corner)
		mmW := 200.0
		mmH := 133.0
		mmX := float64(screenW) - mmW - 12
		mmY := float64(screenH) - 150 - mmH - 12
		if float64(mx) >= mmX && float64(mx) <= mmX+mmW && float64(my) >= mmY && float64(my) <= mmY+mmH {
			// Click on minimap: jump camera there
			relX := (float64(mx) - mmX) / mmW
			relY := (float64(my) - mmY) / mmH
			fs.CamX = relX * factory.WorldW
			fs.CamY = relY * factory.WorldH
		} else {
			// [15] Check if clicking a machine
			clickedMachine := -1
			for mi := range fs.Machines {
				m := &fs.Machines[mi]
				if worldX >= m.X && worldX <= m.X+m.W && worldY >= m.Y && worldY <= m.Y+m.H {
					clickedMachine = mi
					break
				}
			}

			if clickedMachine >= 0 {
				// Toggle machine active state
				m := &fs.Machines[clickedMachine]
				if m.Active || m.CurrentInput > 0 || m.OutputReady {
					// Disable: stop processing
					m.Active = false
					m.ProcessTimer = 0
				} else {
					// Re-enable: allow processing to resume naturally
					m.Active = false // will start on next TickMachines if input available
				}
			} else {
				// Find closest bot within click radius
				bestDist := 20.0 // max click distance in world pixels
				bestIdx := -1
				for i := range fs.Bots {
					dx := fs.Bots[i].X - worldX
					dy := fs.Bots[i].Y - worldY
					dist := math.Sqrt(dx*dx + dy*dy)
					if dist < bestDist {
						bestDist = dist
						bestIdx = i
					}
				}
				fs.SelectedBot = bestIdx
				// Reset selected bot path when selecting a new bot
				if bestIdx >= 0 {
					fs.SelectedBotPathIdx = 0
					for i := range fs.SelectedBotPath {
						fs.SelectedBotPath[i] = [2]float64{}
					}
				}
			}
		}
	}

	// Hover detection for tooltip
	{
		mx, my := ebiten.CursorPosition()
		worldX := (float64(mx) - float64(screenW)/2) / fs.CamZoom + fs.CamX
		worldY := (float64(my) - float64(screenH)/2) / fs.CamZoom + fs.CamY
		bestDist := 15.0
		bestIdx := -1
		for i := range fs.Bots {
			dx := fs.Bots[i].X - worldX
			dy := fs.Bots[i].Y - worldY
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < bestDist {
				bestDist = dist
				bestIdx = i
			}
		}
		fs.HoverBot = bestIdx
	}
}

// --- Deploy / Preset / Auto-reset ---

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

	// Submit leaderboard score before resetting (if there were deliveries)
	if ss.DeliveryOn && ss.DeliveryStats.TotalDelivered > 0 {
		if ss.Leaderboard == nil {
			ss.Leaderboard = swarm.LoadLeaderboard()
		}
		mode := "Script"
		gen := ss.Generation
		if ss.GPEnabled {
			mode = "GP"
			gen = ss.GPGeneration
		} else if ss.NeuroEnabled {
			mode = "Neuro"
			gen = ss.NeuroGeneration
		} else if ss.EvolutionOn {
			mode = "Evolution"
		}
		swarm.SubmitScore(ss.Leaderboard, swarm.LeaderboardEntry{
			Name:       ss.ProgramName,
			Deliveries: ss.DeliveryStats.TotalDelivered,
			Correct:    ss.DeliveryStats.CorrectDelivered,
			Wrong:      ss.DeliveryStats.WrongDelivered,
			BotCount:   ss.BotCount,
			Ticks:      ss.Tick,
			Generation: gen,
			Mode:       mode,
		})
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

	// Show success toast
	ss.DeployFlash = 60 // 1 second
	ss.DeployRuleCount = len(prog.Rules)

	logger.Info("SWARM", "Program deployed: %d rules", len(prog.Rules))
}

// countUsedParams counts how many evolution parameters are flagged as used.
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
	if idx < 0 || idx >= len(ss.Presets) {
		return
	}

	ss.ProgramName = ss.Presets[idx].Name
	presetText := ss.Presets[idx].Source
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
		// Maze OFF by default, except Maze Explorer preset
		if ss.Presets[idx].Name == "Maze Explorer" {
			ss.MazeOn = true
			swarm.GenerateSwarmMaze(ss)
		} else {
			ss.MazeOn = false
			ss.MazeWalls = nil
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
		// Neuro: LKW needs truck mode + delivery
		if swarm.IsNeuroTruckPresetIdx(idx) {
			ss.TruckToggle = true
			ss.DeliveryOn = true
			ss.TruckState = swarm.NewSwarmTruckState(ss.Rng)
			swarm.GenerateDeliveryStations(ss)
			logger.Info("SWARM", "Neuro LKW preset -- NEURO+TRUCKS auto-aktiviert, %d Bots x %d Gewichte",
				ss.BotCount, swarm.NeuroWeights)
		} else {
			logger.Info("SWARM", "Neuro Delivery preset -- NEURO auto-aktiviert, %d Bots x %d Gewichte",
				ss.BotCount, swarm.NeuroWeights)
		}
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
				// GP: Seeded Start -- use the parsed program as seed
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

// factoryZoomToFit calculates the bounding box of all factory bots and sets the
// camera to center on the fleet with enough zoom to fit them in the viewport.
func (g *Game) factoryZoomToFit(fs *factory.FactoryState) {
	if len(fs.Bots) == 0 {
		return
	}
	minX, maxX := fs.Bots[0].X, fs.Bots[0].X
	minY, maxY := fs.Bots[0].Y, fs.Bots[0].Y
	for i := range fs.Bots {
		if fs.Bots[i].X < minX {
			minX = fs.Bots[i].X
		}
		if fs.Bots[i].X > maxX {
			maxX = fs.Bots[i].X
		}
		if fs.Bots[i].Y < minY {
			minY = fs.Bots[i].Y
		}
		if fs.Bots[i].Y > maxY {
			maxY = fs.Bots[i].Y
		}
	}
	fs.CamX = (minX + maxX) / 2
	fs.CamY = (minY + maxY) / 2
	bboxW := maxX - minX + 200
	bboxH := maxY - minY + 200
	if bboxW < 100 {
		bboxW = 100
	}
	if bboxH < 100 {
		bboxH = 100
	}
	sw, sh := ebiten.WindowSize()
	zoomW := float64(sw) / bboxW
	zoomH := float64(sh) / bboxH
	fs.CamZoom = math.Min(zoomW, zoomH) * 0.9
	fs.FollowCamBot = -1
}
