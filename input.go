package main

import (
	"swarmsim/domain/bot"
	"swarmsim/logger"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

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

	// R: spawn resource (only without Shift, Shift+R = MFO overlay)
	if inpututil.IsKeyJustPressed(ebiten.KeyR) && !ebiten.IsKeyPressed(ebiten.KeyShift) {
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
		if g.sim != nil && g.sim.SwarmState != nil {
			g.sim.SwarmState.ShowMinimap = g.renderer.ShowMinimap
		}
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
