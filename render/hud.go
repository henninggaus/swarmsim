package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/bot"
	"swarmsim/engine/simulation"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawHUD renders the heads-up display overlay.
func DrawHUD(screen *ebiten.Image, s *simulation.Simulation, fps float64, r *Renderer) {
	// Swarm mode: separate editor panel + HUD
	if s.SwarmMode && s.SwarmState != nil {
		DrawSwarmEditor(screen, s.SwarmState)
		DrawSwarmHUD(screen, s, fps)
		DrawCaptureOverlay(screen, r)
		drawFPSWarning(screen, r, fps, screen.Bounds().Dx())
		drawFadeOverlay(screen, r)
		return
	}

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// Top-left: FPS, Tick, Speed
	info := fmt.Sprintf("FPS: %.0f  Tick: %d  Speed: %.1fx", fps, s.Tick, s.Speed)
	if s.Paused {
		info += "  [PAUSED]"
	}
	ebitenutil.DebugPrintAt(screen, info, 10, 10)

	// Mode-specific top HUD
	if s.TruckMode && s.TruckState != nil {
		drawTruckHUD(screen, s, sw, sh)
	} else {
		drawWaveHUD(screen, s, sw)
	}

	// Top-center: Generation info (shift down if wave/truck HUD active)
	genY := 10
	if s.Cfg.WaveEnabled || s.TruckMode {
		genY = 95
	}
	genInfo := fmt.Sprintf("Gen: %d  Tick: %d/%d  Best: %.0f  Avg: %.0f",
		s.Generation, s.GenerationTick, s.Cfg.GenerationLength, s.BestFitness, s.AvgFitness)
	genW := len(genInfo) * 6
	ebitenutil.DebugPrintAt(screen, genInfo, sw/2-genW/2, genY)

	// Top-right: Bot counts
	counts := s.BotCount()
	y := 10
	drawBotCount(screen, sw-160, y, "Scout", counts[bot.TypeScout], ColorScout)
	y += 16
	drawBotCount(screen, sw-160, y, "Worker", counts[bot.TypeWorker], ColorWorker)
	y += 16
	drawBotCount(screen, sw-160, y, "Leader", counts[bot.TypeLeader], ColorLeader)
	y += 16
	drawBotCount(screen, sw-160, y, "Tank", counts[bot.TypeTank], ColorTank)
	y += 16
	drawBotCount(screen, sw-160, y, "Healer", counts[bot.TypeHealer], ColorHealer)

	// Bottom-left: Resources & messages
	available := 0
	for _, r := range s.Resources {
		if r.IsAvailable() {
			available++
		}
	}
	resInfo := fmt.Sprintf("Resources: %d  Delivered: %d  Score: %d  Msgs: %d (total: %d)", available, s.Delivered, s.Score, s.ActiveMsgs, s.TotalMsgsSent)
	ebitenutil.DebugPrintAt(screen, resInfo, 10, sh-45)

	pherModes := []string{"Pher:OFF", "Pher:FOUND", "Pher:ALL"}
	ebitenutil.DebugPrintAt(screen, pherModes[s.PheromoneVizMode], 10, sh-30)

	if s.TruckMode {
		ebitenutil.DebugPrintAt(screen, "SPACE:Pause N:NewTruck F:Comm G:Sensor D:Debug P:Pher V:Genome S:Sound +/-:Speed H:Hilfe F1-F5:Scenario F6:Truck", 10, sh-15)
	} else {
		ebitenutil.DebugPrintAt(screen, "SPACE:Pause 1-5:Bot R:Res O:Obs F:Comm G:Sensor D:Debug P:Pher E:Evolve V:Genome S:Sound +/-:Speed H:Hilfe", 10, sh-15)
	}

	// Selected bot info panel
	if s.SelectedBotID >= 0 {
		bot := s.GetBotByID(s.SelectedBotID)
		if bot != nil && bot.IsAlive() {
			drawBotInfoPanel(screen, bot, sw)
			if s.ShowGenomeOverlay {
				drawGenomeOverlay(screen, bot, sw)
			}
		}
	}

	// Fitness graph
	if len(s.FitnessHistory) > 1 {
		drawFitnessGraph(screen, s, sw, sh)
	}

	// Scenario title overlay
	if s.ScenarioTimer > 0 {
		drawScenarioTitle(screen, s.ScenarioTitle, sw, sh, s.ScenarioTimer)
	}

	// Screenshot / GIF overlay
	DrawCaptureOverlay(screen, r)

	// FPS warning
	drawFPSWarning(screen, r, fps, sw)

	// Fade transition overlay
	drawFadeOverlay(screen, r)
}

// drawFPSWarning shows a yellow warning when FPS is consistently low.
func drawFPSWarning(screen *ebiten.Image, r *Renderer, fps float64, sw int) {
	if fps > 0 && fps < 30 {
		r.LowFPSCounter++
	} else {
		r.LowFPSCounter = 0
	}
	if r.LowFPSCounter > 60 {
		warn := fmt.Sprintf("Low FPS: %.0f", fps)
		warnW := len(warn) * charW
		warnX := sw - warnW - 15
		vector.DrawFilledRect(screen, float32(warnX-4), 2, float32(warnW+8), float32(lineH+4),
			color.RGBA{100, 80, 0, 200}, false)
		printColoredAt(screen, warn, warnX, 4, color.RGBA{255, 220, 80, 255})
	}
}

// drawFadeOverlay draws a full-screen black rect for fade transitions.
func drawFadeOverlay(screen *ebiten.Image, r *Renderer) {
	if r.FadeAlpha <= 0 && r.FadeDir == 0 {
		return
	}
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// Animate
	if r.FadeDir == -1 {
		r.FadeAlpha += 0.1
		if r.FadeAlpha >= 1.0 {
			r.FadeAlpha = 1.0
			// Execute load callback and reverse fade direction
			if r.FadeLoad != nil {
				r.FadeLoad()
				r.FadeLoad = nil
			}
			r.FadeDir = 1
		}
	} else if r.FadeDir == 1 {
		r.FadeAlpha -= 0.08
		if r.FadeAlpha <= 0 {
			r.FadeAlpha = 0
			r.FadeDir = 0
		}
	}

	if r.FadeAlpha > 0 {
		alpha := uint8(r.FadeAlpha * 255)
		vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh),
			color.RGBA{0, 0, 0, alpha}, false)
	}
}

func drawBotCount(screen *ebiten.Image, x, y int, name string, count int, col color.RGBA) {
	vector.DrawFilledRect(screen, float32(x), float32(y+2), 10, 10, col, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s: %d", name, count), x+15, y)
}

func drawBotInfoPanel(screen *ebiten.Image, b bot.Bot, screenW int) {
	px := screenW - 200
	py := 110

	vector.DrawFilledRect(screen, float32(px-5), float32(py-5), 195, 140, color.RGBA{0, 0, 0, 180}, false)
	vector.StrokeRect(screen, float32(px-5), float32(py-5), 195, 140, 1, color.RGBA{100, 100, 100, 255}, false)

	col := BotColor(b.Type())
	vector.DrawFilledRect(screen, float32(px), float32(py), 10, 10, col, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("ID: %d  %s", b.ID(), b.Type()), px+15, py)
	py += 18
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("State: %s", b.GetState()), px, py)
	py += 16
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Health: %.0f/%.0f", b.Health(), b.MaxHealth()), px, py)
	py += 16
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Energy: %.0f/100", b.GetEnergy()), px, py)
	py += 16
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Pos: (%.0f, %.0f)", b.Position().X, b.Position().Y), px, py)
	py += 16
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Speed: %.1f", b.Velocity().Len()), px, py)
	py += 16
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Fitness: %.0f  Inv: %d", b.GetBase().Fitness(), len(b.GetInventory())), px, py)
}

func drawGenomeOverlay(screen *ebiten.Image, b bot.Bot, screenW int) {
	px := screenW - 200
	py := 260

	vector.DrawFilledRect(screen, float32(px-5), float32(py-5), 195, 130, ColorGenomeBg, false)
	vector.StrokeRect(screen, float32(px-5), float32(py-5), 195, 130, 1, color.RGBA{100, 100, 100, 255}, false)

	ebitenutil.DebugPrintAt(screen, "Genome:", px, py)
	py += 14

	genome := b.GetGenome()
	labels := bot.GenomeLabels()
	values := genome.Values()

	for i := 0; i < 7; i++ {
		ebitenutil.DebugPrintAt(screen, labels[i], px, py+i*15)
		bx := float32(px + 50)
		by := float32(py + i*15 + 2)
		vector.DrawFilledRect(screen, bx, by, 100, 10, color.RGBA{40, 40, 40, 255}, false)
		fillW := float32(100) * float32(math.Max(0, math.Min(1, values[i])))
		vector.DrawFilledRect(screen, bx, by, fillW, 10, ColorGenomeBar, false)
	}
}

func drawFitnessGraph(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	graphW := 200
	graphH := 80
	gx := sw - graphW - 10
	gy := sh - graphH - 55

	vector.DrawFilledRect(screen, float32(gx), float32(gy), float32(graphW), float32(graphH), ColorFitnessBg, false)
	vector.StrokeRect(screen, float32(gx), float32(gy), float32(graphW), float32(graphH), 1, color.RGBA{60, 60, 60, 255}, false)
	ebitenutil.DebugPrintAt(screen, "Avg Fitness", gx+2, gy+2)

	hist := s.FitnessHistory
	n := len(hist)
	if n < 2 {
		return
	}

	minV, maxV := hist[0], hist[0]
	for _, v := range hist {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if maxV == minV {
		maxV = minV + 1
	}

	maxPoints := graphW / 2
	start := 0
	if n > maxPoints {
		start = n - maxPoints
	}
	points := hist[start:]

	stepX := float64(graphW-10) / float64(len(points)-1)
	for i := 1; i < len(points); i++ {
		x1 := float32(gx+5) + float32(float64(i-1)*stepX)
		y1 := float32(gy+graphH-10) - float32((points[i-1]-minV)/(maxV-minV)*float64(graphH-20))
		x2 := float32(gx+5) + float32(float64(i)*stepX)
		y2 := float32(gy+graphH-10) - float32((points[i]-minV)/(maxV-minV)*float64(graphH-20))
		vector.StrokeLine(screen, x1, y1, x2, y2, 1.5, ColorFitnessLine, false)
	}
}

func drawWaveHUD(screen *ebiten.Image, s *simulation.Simulation, sw int) {
	if !s.Cfg.WaveEnabled {
		return
	}

	// Large centered score
	scoreText := fmt.Sprintf("Score: %d", s.Score)
	scoreImg := cachedTextImage(scoreText)
	textW := scoreImg.Bounds().Dx()

	scale := 2.5
	totalW := float64(textW) * scale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(sw)/2-totalW/2, 30)
	screen.DrawImage(scoreImg, op)

	// Wave info line below score
	waveInfo := fmt.Sprintf("Wave %d  |  Next wave in: %d ticks", s.WaveNumber, s.WaveTicksLeft)
	waveW := len(waveInfo) * 6
	ebitenutil.DebugPrintAt(screen, waveInfo, sw/2-waveW/2, 75)
}

func drawTruckHUD(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	ts := s.TruckState
	if ts == nil {
		return
	}

	// Large centered score
	scoreText := fmt.Sprintf("Score: %d", s.Score)
	scoreImg := cachedTextImage(scoreText)
	textW := scoreImg.Bounds().Dx()

	scale := 2.5
	totalW := float64(textW) * scale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(sw)/2-totalW/2, 25)
	screen.DrawImage(scoreImg, op)

	// Packages delivered
	pkgText := fmt.Sprintf("Packages: %d/%d delivered", ts.DeliveredPkgs, ts.TotalPkgs)
	pkgW := len(pkgText) * 6
	ebitenutil.DebugPrintAt(screen, pkgText, sw/2-pkgW/2, 72)

	// Sort accuracy
	totalDelivered := ts.CorrectZone + ts.WrongZone
	accuracy := 0.0
	if totalDelivered > 0 {
		accuracy = float64(ts.CorrectZone) / float64(totalDelivered) * 100
	}
	accText := fmt.Sprintf("Sort Accuracy: %.0f%%", accuracy)
	accW := len(accText) * 6
	ebitenutil.DebugPrintAt(screen, accText, sw/2-accW/2, 86)

	// Timer
	seconds := ts.Timer / 30
	minutes := seconds / 60
	secs := seconds % 60
	timerText := fmt.Sprintf("Time: %02d:%02d", minutes, secs)
	timerW := len(timerText) * 6
	ebitenutil.DebugPrintAt(screen, timerText, sw-timerW-20, 10)

	// Completion overlay
	if ts.Completed {
		completeText := "COMPLETED!"
		ctImg := cachedTextImage(completeText)
		ctW := ctImg.Bounds().Dx()
		ctH := ctImg.Bounds().Dy()

		ctScale := 3.0
		ctTotalW := float64(ctW) * ctScale
		ctTotalH := float64(ctH) * ctScale

		bgX := float64(sw)/2 - ctTotalW/2 - 10
		bgY := float64(sh)/2 - ctTotalH/2 - 5
		vector.DrawFilledRect(screen, float32(bgX), float32(bgY), float32(ctTotalW+20), float32(ctTotalH+10),
			color.RGBA{0, 60, 0, 200}, false)

		ctOp := &ebiten.DrawImageOptions{}
		ctOp.GeoM.Scale(ctScale, ctScale)
		ctOp.GeoM.Translate(float64(sw)/2-ctTotalW/2, float64(sh)/2-ctTotalH/2)
		screen.DrawImage(ctImg, ctOp)

		// Final stats
		finalText := fmt.Sprintf("Score: %d | Accuracy: %.0f%% | Time: %02d:%02d", s.Score, accuracy, minutes, secs)
		finalW := len(finalText) * 6
		ebitenutil.DebugPrintAt(screen, finalText, sw/2-finalW/2, sh/2+30)
	}
}

func drawScenarioTitle(screen *ebiten.Image, title string, sw, sh, timer int) {
	if title == "" {
		return
	}
	alpha := uint8(255)
	if timer < 30 {
		alpha = uint8(timer * 255 / 30)
	}

	textImg := cachedTextImage(title)
	textW := textImg.Bounds().Dx()
	textH := textImg.Bounds().Dy()

	scale := 3.0
	totalW := float64(textW) * scale
	totalH := float64(textH) * scale

	bgX := float64(sw)/2 - totalW/2 - 10
	bgY := float64(sh)/2 - totalH/2 - 5
	vector.DrawFilledRect(screen, float32(bgX), float32(bgY), float32(totalW+20), float32(totalH+10),
		color.RGBA{0, 0, 0, alpha / 2}, false)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(sw)/2-totalW/2, float64(sh)/2-totalH/2)
	op.ColorScale.ScaleAlpha(float32(alpha) / 255.0)
	screen.DrawImage(textImg, op)
}
