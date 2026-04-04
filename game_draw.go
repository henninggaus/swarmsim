package main

import (
	"fmt"
	"strings"
	"swarmsim/logger"
	"swarmsim/render"

	"github.com/hajimehoshi/ebiten/v2"
)

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

	// Factory mode has its own HUD — skip classic/swarm overlays
	if g.sim.FactoryMode {
		// Only console + help overlay in factory mode
		if g.showConsole {
			render.DrawConsole(screen, logger.Entries(), false, -1)
		}
		if g.showHelp {
			render.DrawHelpOverlay(screen, false, g.helpScrollY)
		}
		if g.panicTimer > 0 && g.panicMsg != "" {
			render.DrawPanicBanner(screen, g.panicMsg, g.panicTimer)
		}
		return
	}

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
			// Max frames reached -- auto-stop
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

	// Learning system overlays (lesson text, emergence popups, lesson menu)
	if g.sim.SwarmMode && g.sim.SwarmState != nil {
		render.DrawLessonOverlay(screen, g.sim.SwarmState)
		render.DrawEmergencePopup(screen, g.sim.SwarmState)
		render.DrawLessonMenu(screen, g.sim.SwarmState)
		// Educational tips (Did You Know) — update timer each frame
		render.UpdateDidYouKnow(g.sim.SwarmState)
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

// Layout returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}
