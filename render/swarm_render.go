package render

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"swarmsim/engine/swarmscript"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawSwarmMode renders the entire swarm mode screen (arena on the right).
func (r *Renderer) DrawSwarmMode(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	ss := s.SwarmState
	if ss == nil {
		return
	}

	// Fill entire screen with editor bg (left panel drawn by DrawSwarmEditor)
	screen.Fill(ColorSwarmEditorBg)

	// --- Offscreen arena image (supports follow-cam zoom/pan) ---
	arenaW := int(ss.ArenaW)
	arenaH := int(ss.ArenaH)
	if r.arenaImg == nil || r.arenaImg.Bounds().Dx() != arenaW {
		r.arenaImg = ebiten.NewImage(arenaW, arenaH)
	}
	r.arenaImg.Clear()

	// All arena content is drawn to r.arenaImg at (0,0) coordinates
	a := r.arenaImg

	// Arena background
	vector.DrawFilledRect(a, 0, 0, float32(ss.ArenaW), float32(ss.ArenaH), ColorSwarmArenaBg, false)

	// Arena grid
	gridStep := 50.0
	for gx := gridStep; gx < ss.ArenaW; gx += gridStep {
		sx := float32(gx)
		vector.StrokeLine(a, sx, 0, sx, float32(ss.ArenaH), 1, ColorSwarmArenaGrid, false)
	}
	for gy := gridStep; gy < ss.ArenaH; gy += gridStep {
		sy := float32(gy)
		vector.StrokeLine(a, 0, sy, float32(ss.ArenaW), sy, 1, ColorSwarmArenaGrid, false)
	}

	// Arena border
	vector.StrokeRect(a, 0, 0, float32(ss.ArenaW), float32(ss.ArenaH), 2, ColorSwarmArenaBorder, false)

	// Obstacles (3D effect: lighter top-left edges, darker bottom-right)
	for _, obs := range ss.Obstacles {
		ox := float32(obs.X)
		oy := float32(obs.Y)
		ow := float32(obs.W)
		oh := float32(obs.H)
		vector.DrawFilledRect(a, ox, oy, ow, oh, ColorSwarmObstacle, false)
		vector.StrokeLine(a, ox, oy, ox+ow, oy, 2, ColorSwarmObstacleHi, false)
		vector.StrokeLine(a, ox, oy, ox, oy+oh, 2, ColorSwarmObstacleHi, false)
		vector.StrokeLine(a, ox, oy+oh, ox+ow, oy+oh, 2, ColorSwarmObstacleLo, false)
		vector.StrokeLine(a, ox+ow, oy, ox+ow, oy+oh, 2, ColorSwarmObstacleLo, false)
	}

	// Maze walls (thin colored rects with bright border)
	for _, wall := range ss.MazeWalls {
		wx, wy, ww, wh := float32(wall.X), float32(wall.Y), float32(wall.W), float32(wall.H)
		vector.DrawFilledRect(a, wx, wy, ww, wh, ColorSwarmMazeWall, false)
		// 1px bright border for better visibility
		vector.StrokeLine(a, wx, wy, wx+ww, wy, 1, ColorSwarmMazeBorder, false)
		vector.StrokeLine(a, wx, wy, wx, wy+wh, 1, ColorSwarmMazeBorder, false)
		vector.StrokeLine(a, wx, wy+wh, wx+ww, wy+wh, 1, ColorSwarmMazeBorder, false)
		vector.StrokeLine(a, wx+ww, wy, wx+ww, wy+wh, 1, ColorSwarmMazeBorder, false)
	}

	// Light source (concentric circles with decreasing alpha)
	if ss.Light.Active {
		lx := float32(ss.Light.X)
		ly := float32(ss.Light.Y)
		for ri := 4; ri >= 1; ri-- {
			radius := float32(ri) * 25.0
			alpha := uint8(25 - ri*4)
			if alpha < 5 {
				alpha = 5
			}
			lightCol := color.RGBA{ColorSwarmLight.R, ColorSwarmLight.G, ColorSwarmLight.B, alpha}
			vector.DrawFilledCircle(a, lx, ly, radius, lightCol, false)
		}
		vector.DrawFilledCircle(a, lx, ly, 6, ColorSwarmLight, false)
		vector.StrokeCircle(a, lx, ly, 10, 1.5, color.RGBA{255, 255, 100, 150}, false)
	}

	// Delivery rendering
	if ss.DeliveryOn {
		if ss.ShowRoutes {
			drawPickupDropoffRoutes(a, ss, 0, 0)
		}
		if ss.ShowRoutes {
			hasCarrying := false
			for i := range ss.Bots {
				if ss.Bots[i].CarryingPkg >= 0 {
					hasCarrying = true
					break
				}
			}
			if hasCarrying {
				drawCarryRouteLines(a, ss, 0, 0)
			}
		}
		drawDeliveryStations(a, ss, 0, 0)
		drawStationLabels(a, ss, 0, 0)
		drawDeliveryPackages(a, ss, 0, 0)
	}

	// Truck rendering (ramp + vehicle)
	if ss.TruckToggle && ss.TruckState != nil {
		drawSwarmRamp(a, ss)
		if ss.TruckState.CurrentTruck != nil {
			drawSwarmTruckVehicle(a, ss)
		}
	}

	// Trails — connected fading lines with color gradient
	if ss.ShowTrails {
		trailLen := len(ss.Bots[0].Trail)
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			// Build ordered trail: newest first
			for t := 1; t < trailLen; t++ {
				currIdx := (bot.TrailIdx - t + trailLen) % trailLen
				prevIdx := (bot.TrailIdx - t - 1 + trailLen) % trailLen
				cx, cy := bot.Trail[currIdx][0], bot.Trail[currIdx][1]
				px, py := bot.Trail[prevIdx][0], bot.Trail[prevIdx][1]
				if (cx == 0 && cy == 0) || (px == 0 && py == 0) {
					continue
				}
				// Skip if distance too large (teleport/respawn)
				dx := cx - px
				dy := cy - py
				if dx*dx+dy*dy > 2500 {
					continue // >50px gap = teleport
				}
				// Fade alpha and width with age
				frac := float32(t) / float32(trailLen)
				alpha := uint8(float32(120) * (1.0 - frac))
				if alpha < 8 {
					alpha = 8
				}
				width := 2.5 * (1.0 - frac*0.7)
				if width < 0.5 {
					width = 0.5
				}
				trailCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], alpha}
				vector.StrokeLine(a, float32(cx), float32(cy), float32(px), float32(py), float32(width), trailCol, false)
			}
		}
	}

	// Pheromone overlay (green dots at cells with intensity > 0.05)
	if ss.ShowTrails && ss.PherGrid != nil {
		g := ss.PherGrid
		for r := 0; r < g.Rows; r++ {
			for c := 0; c < g.Cols; c++ {
				v := g.Data[r*g.Cols+c]
				if v > 0.05 {
					cx := float32(float64(c)*g.CellSize + g.CellSize/2)
					cy := float32(float64(r)*g.CellSize + g.CellSize/2)
					alpha := uint8(v * 180)
					if alpha > 180 {
						alpha = 180
					}
					pherCol := color.RGBA{0, 255, 100, alpha}
					vector.DrawFilledCircle(a, cx, cy, 3, pherCol, false)
				}
			}
		}
	}

	// Follow lines
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.FollowTargetIdx >= 0 && bot.FollowTargetIdx < len(ss.Bots) {
			target := &ss.Bots[bot.FollowTargetIdx]
			lineCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 120}
			vector.StrokeLine(a, float32(bot.X), float32(bot.Y), float32(target.X), float32(target.Y), 1, lineCol, false)
		}
	}

	// Communication graph overlay (K key)
	if ss.ShowCommGraph && len(ss.PrevMessages) > 0 {
		commRange := float32(swarm.SwarmCommRange)
		for _, msg := range ss.PrevMessages {
			sx := float32(msg.X)
			sy := float32(msg.Y)
			// Draw small broadcast ring at sender
			vector.StrokeCircle(a, sx, sy, commRange, 1, color.RGBA{100, 200, 255, 40}, false)
			// Draw lines to receiving bots (within comm range)
			for bi := range ss.Bots {
				bot := &ss.Bots[bi]
				dx := float32(bot.X) - sx
				dy := float32(bot.Y) - sy
				dist := dx*dx + dy*dy
				if dist < commRange*commRange && dist > 4 {
					alpha := uint8(120 - 80*dist/(commRange*commRange))
					lineCol := color.RGBA{100, 200, 255, alpha}
					vector.StrokeLine(a, sx, sy, float32(bot.X), float32(bot.Y), 1, lineCol, false)
				}
			}
		}
	}

	// Heatmap overlay (Y key) — drawn under bots
	if ss.ShowHeatmap && ss.HeatmapGrid != nil {
		cellW := float32(swarm.HeatmapCellSize)
		// Find max for normalization
		maxVal := 1.0
		for _, v := range ss.HeatmapGrid {
			if v > maxVal {
				maxVal = v
			}
		}
		for row := 0; row < ss.HeatmapRows; row++ {
			for col := 0; col < ss.HeatmapCols; col++ {
				v := ss.HeatmapGrid[row*ss.HeatmapCols+col]
				if v < 1 {
					continue
				}
				intensity := v / maxVal
				// Color gradient: blue → green → yellow → red
				var hr, hg, hb uint8
				if intensity < 0.33 {
					t := intensity / 0.33
					hb = uint8(200 * (1 - t))
					hg = uint8(200 * t)
				} else if intensity < 0.66 {
					t := (intensity - 0.33) / 0.33
					hg = uint8(200 * (1 - t))
					hr = uint8(255 * t)
					hg += uint8(100 * t) // yellow
				} else {
					t := (intensity - 0.66) / 0.34
					hr = 255
					hg = uint8(100 * (1 - t))
				}
				alpha := uint8(40 + 100*intensity)
				vector.DrawFilledRect(a, float32(col)*cellW, float32(row)*cellW,
					cellW, cellW, color.RGBA{hr, hg, hb, alpha}, false)
			}
		}
	}

	// Draw bots
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		bx := float32(bot.X)
		by := float32(bot.Y)
		radius := float32(swarm.SwarmBotRadius)

		// Color filter: check if bot passes filter
		botAlpha := uint8(255)
		if ss.ColorFilter > 0 {
			match := false
			switch ss.ColorFilter {
			case 1: // red dominant
				match = bot.LEDColor[0] > bot.LEDColor[1] && bot.LEDColor[0] > bot.LEDColor[2] && bot.LEDColor[0] > 100
			case 2: // green dominant
				match = bot.LEDColor[1] > bot.LEDColor[0] && bot.LEDColor[1] > bot.LEDColor[2] && bot.LEDColor[1] > 100
			case 3: // blue dominant
				match = bot.LEDColor[2] > bot.LEDColor[0] && bot.LEDColor[2] > bot.LEDColor[1] && bot.LEDColor[2] > 100
			case 4: // carrying package
				match = bot.CarryingPkg >= 0
			case 5: // idle (speed == 0)
				match = bot.Speed < 0.1
			}
			if !match {
				botAlpha = 30 // nearly invisible
			}
		}

		// Boost minimum LED brightness so dark bots are still visible
		r, g, b := bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2]
		if r < 60 && g < 60 && b < 60 {
			r, g, b = 80, 80, 80
		}
		botCol := color.RGBA{r, g, b, botAlpha}
		vector.DrawFilledCircle(a, bx, by, radius, botCol, false)

		// Team ring overlay
		if ss.TeamsEnabled && bot.Team > 0 {
			var teamCol color.RGBA
			if bot.Team == 1 {
				teamCol = color.RGBA{60, 100, 255, 180} // blue
			} else {
				teamCol = color.RGBA{255, 60, 60, 180} // red
			}
			vector.StrokeCircle(a, bx, by, radius+2, 2, teamCol, false)
		}

		dirLen := radius * 1.5
		dx := float32(math.Cos(bot.Angle)) * dirLen
		dy := float32(math.Sin(bot.Angle)) * dirLen
		vector.StrokeLine(a, bx, by, bx+dx, by+dy, 1.5, color.RGBA{255, 255, 255, 200}, false)

		if ss.DeliveryOn && bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			pkg := &ss.Packages[bot.CarryingPkg]
			pkgCol := deliveryColor(pkg.Color)
			pulse := 0.6 + 0.4*math.Sin(float64(ss.Tick)*0.1)
			glowAlpha := uint8(60 + 40*pulse)
			glowCol := color.RGBA{pkgCol.R, pkgCol.G, pkgCol.B, glowAlpha}
			vector.StrokeCircle(a, bx, by, radius+5, 2, glowCol, false)
			vector.DrawFilledRect(a, bx-4, by-radius-8, 8, 8, pkgCol, false)
			vector.StrokeRect(a, bx-4, by-radius-8, 8, 8, 1, color.RGBA{255, 255, 255, 180}, false)
			// Visual indicator: "!" = heading to dropoff, "?" = lost/searching
			if bot.DropoffMatch || bot.HeardBeaconDropoffColor > 0 {
				printColoredAt(a, "!", int(bx-3), int(by-radius-20), color.RGBA{50, 255, 50, 255})
			} else {
				printColoredAt(a, "?", int(bx-3), int(by-radius-20), color.RGBA{255, 100, 50, 255})
			}
		}

		if bot.BlinkTimer > 0 && (bot.BlinkTimer/4)%2 == 0 {
			blinkCol := ColorSwarmBotBlink
			blinkCol.A = 120
			vector.DrawFilledCircle(a, bx, by, radius+2, blinkCol, false)
		}

		// Energy bar under bot
		if ss.EnergyEnabled {
			barW := float32(16)
			barH := float32(3)
			barX := bx - barW/2
			barY := by + radius + 3
			// Background
			vector.DrawFilledRect(a, barX, barY, barW, barH, color.RGBA{40, 40, 40, 180}, false)
			// Fill based on energy
			fill := float32(bot.Energy) / 100.0 * barW
			var eCol color.RGBA
			if bot.Energy > 50 {
				eCol = color.RGBA{80, 220, 80, 200}
			} else if bot.Energy > 20 {
				eCol = color.RGBA{220, 200, 50, 200}
			} else {
				eCol = color.RGBA{220, 50, 50, 200}
			}
			vector.DrawFilledRect(a, barX, barY, fill, barH, eCol, false)
		}

		if i == ss.SelectedBot {
			pulse := float32(2.0 + 2.0*math.Sin(float64(ss.Tick)*0.12))
			pulseAlpha := uint8(150 + int(50*math.Sin(float64(ss.Tick)*0.08)))
			selCol := color.RGBA{ColorSwarmSelected.R, ColorSwarmSelected.G, ColorSwarmSelected.B, pulseAlpha}
			vector.StrokeCircle(a, bx, by, radius+pulse+2, 2, selCol, false)
		}
		if i == ss.CompareBot {
			pulse := float32(2.0 + 2.0*math.Sin(float64(ss.Tick)*0.12+math.Pi))
			pulseAlpha := uint8(150 + int(50*math.Sin(float64(ss.Tick)*0.08+math.Pi)))
			cmpCol := color.RGBA{0, 220, 255, pulseAlpha} // cyan ring
			vector.StrokeCircle(a, bx, by, radius+pulse+2, 2, cmpCol, false)
		}
	}

	// Ghost bots (wrap mode only)
	if ss.WrapMode {
		ghostMargin := 40.0
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			ghostCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 128}
			radius := float32(swarm.SwarmBotRadius)

			drawGhost := func(gx, gy float64) {
				vector.DrawFilledCircle(a, float32(gx), float32(gy), radius, ghostCol, false)
			}

			if bot.X < ghostMargin {
				drawGhost(bot.X+ss.ArenaW, bot.Y)
			}
			if bot.X > ss.ArenaW-ghostMargin {
				drawGhost(bot.X-ss.ArenaW, bot.Y)
			}
			if bot.Y < ghostMargin {
				drawGhost(bot.X, bot.Y+ss.ArenaH)
			}
			if bot.Y > ss.ArenaH-ghostMargin {
				drawGhost(bot.X, bot.Y-ss.ArenaH)
			}
		}
	}

	// Delivery overlays (on top of bots, on arena)
	if ss.DeliveryOn {
		drawScorePopups(a, ss, 0, 0)
		processDeliveryEvents(r, ss)
	}

	// Sound events: evolution gong, broadcast blips
	if r.Sound != nil && r.Sound.Enabled {
		if ss.EvolutionSoundPending {
			r.Sound.PlayEvolution()
			ss.EvolutionSoundPending = false
		}
		if ss.BroadcastCount > 0 {
			r.Sound.PlayBroadcast()
			ss.BroadcastCount = 0
		}
	}

	// Delivery particles (on arena)
	if r.SwarmParticles != nil {
		r.SwarmParticles.Update()
		drawSwarmParticles(a, r.SwarmParticles, 0, 0)
	}

	// Selected bot visual overlays (on arena, before blit)
	if ss.SelectedBot >= 0 && ss.SelectedBot < len(ss.Bots) {
		drawSelectedBotOverlays(a, ss)
	}

	// --- Blit arena image to screen with camera transform ---
	viewportX := 415.0
	viewportY := 50.0
	viewportW := 800.0
	viewportH := 800.0

	camX := ss.SwarmCamX
	camY := ss.SwarmCamY
	zoom := ss.SwarmCamZoom

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-camX, -camY)
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(viewportX+viewportW/2, viewportY+viewportH/2)

	// Clip to arena viewport
	viewport := screen.SubImage(image.Rect(int(viewportX)-1, int(viewportY)-1, sw, sh)).(*ebiten.Image)
	viewport.DrawImage(r.arenaImg, op)

	// --- HUD elements (drawn directly to screen, NOT zoomed) ---

	// Delivery legend
	if ss.DeliveryOn {
		drawDeliveryLegend(screen, ss, sw)
	}

	// Selected bot info overlay / comparison panel
	if ss.CompareBot >= 0 && ss.CompareBot < len(ss.Bots) && ss.SelectedBot >= 0 && ss.SelectedBot < len(ss.Bots) {
		drawBotComparisonPanel(screen, ss)
	} else if ss.SelectedBot >= 0 && ss.SelectedBot < len(ss.Bots) {
		drawSelectedBotInfo(screen, ss)
	}

	// Truck round complete overlay
	if ss.TruckToggle && ss.TruckState != nil && ss.TruckState.CurrentTruck != nil &&
		ss.TruckState.CurrentTruck.Phase == swarm.TruckRoundDone {
		roundText := fmt.Sprintf("ROUND COMPLETE! Score: %d [N: New Round]", ss.TruckState.Score)
		rtW := len(roundText) * charW
		rtX := sw/2 - rtW/2 + 100
		rtY := sh/2 - 30
		vector.DrawFilledRect(screen, float32(rtX-10), float32(rtY-8), float32(rtW+20), 30,
			color.RGBA{30, 30, 20, 220}, false)
		vector.StrokeRect(screen, float32(rtX-10), float32(rtY-8), float32(rtW+20), 30,
			2, color.RGBA{255, 200, 50, 200}, false)
		printColoredAt(screen, roundText, rtX, rtY, color.RGBA{255, 220, 50, 255})
	}

	// Teams scoreboard
	if ss.TeamsEnabled {
		drawTeamsScoreboard(screen, ss, sw)
	}

	// Follow-cam HUD indicator
	if ss.FollowCamBot >= 0 && ss.FollowCamBot < len(ss.Bots) {
		label := fmt.Sprintf("Following Bot #%d [F to stop]", ss.FollowCamBot)
		printColoredAt(screen, label, 500, 855, color.RGBA{0, 255, 255, 220})
	}

	// Evolution HUD + fitness graph
	if ss.EvolutionOn {
		evoInfo := fmt.Sprintf("Gen: %d | Best: %.0f | Avg: %.1f | Timer: %d/1500",
			ss.Generation, ss.BestFitness, ss.AvgFitness, ss.EvolutionTimer)
		printColoredAt(screen, evoInfo, 420, 48, color.RGBA{180, 50, 180, 255})
		// Fitness graph (150x50px)
		if len(ss.FitnessHistory) > 1 {
			drawSwarmFitnessGraph(screen, ss, 420, 60, 150, 50)
		}
	}

	// GP HUD + fitness graph
	if ss.GPEnabled {
		gpInfo := fmt.Sprintf("GP Gen:%d | Best:%.0f | Avg:%.0f | %d/2000",
			ss.GPGeneration, ss.BestFitness, ss.AvgFitness, ss.GPTimer)
		printColoredAt(screen, gpInfo, 420, 48, color.RGBA{0, 180, 160, 255})
		if len(ss.FitnessHistory) > 1 {
			drawSwarmFitnessGraph(screen, ss, 420, 60, 150, 50)
		}
	}

	// Auto-Optimizer HUD
	if ss.AutoOptimizer != nil && ss.AutoOptimizer.Active {
		opt := ss.AutoOptimizer
		optInfo := fmt.Sprintf("AUTO-OPTIMIZER: Trial %d/%d | Score:%.0f | Best:%.0f | F4=Stop",
			opt.Trial+1, opt.MaxTrials, opt.CurrentScore, opt.BestScore)
		// Background bar
		vector.DrawFilledRect(screen, 360, float32(sh-60), float32(sw-365), 16,
			color.RGBA{80, 20, 20, 200}, false)
		printColoredAt(screen, optInfo, 365, sh-59, color.RGBA{255, 200, 80, 255})
	}

	// Genome visualization overlay (V key)
	if ss.ShowGenomeViz && ss.EvolutionOn {
		drawGenomeVisualization(screen, ss)
	}

	// Genom-Browser overlay (G key)
	if ss.GenomeBrowserOn {
		DrawGenomeBrowser(screen, ss)
	}

	// Tournament overlay (U key)
	if ss.TournamentOn {
		DrawTournamentOverlay(screen, ss)
	}

	// Arena editor mode indicator
	if ss.ArenaEditMode {
		toolNames := []string{"Hindernis", "Station", "Loeschen"}
		toolColors := []color.RGBA{
			{180, 120, 60, 255},
			{60, 180, 120, 255},
			{255, 80, 80, 255},
		}
		label := fmt.Sprintf("ARENA-EDITOR  Tool: %s  (1/2/3=wechseln, O=aus)", toolNames[ss.ArenaEditTool])
		// Background bar
		vector.DrawFilledRect(screen, 360, float32(sh-46), float32(sw-365), 16,
			color.RGBA{20, 20, 40, 200}, false)
		printColoredAt(screen, label, 365, sh-45, toolColors[ss.ArenaEditTool])
	}

	// Statistics dashboard
	if ss.DashboardOn && ss.StatsTracker != nil {
		DrawDashboard(screen, ss, sw-260, 55, 250, sh-70)
	}

	// Minimap
	if r.ShowMinimap {
		r.drawSwarmMinimap(screen, ss)
	}

	// Separator line between editor and arena
	vector.StrokeLine(screen, 350, 0, 350, float32(sh), 2, ColorSwarmEditorSep, false)
}

// drawSelectedBotInfo draws an enhanced info panel for the selected bot.
func drawSelectedBotInfo(screen *ebiten.Image, ss *swarm.SwarmState) {
	bot := &ss.Bots[ss.SelectedBot]
	x := 1050
	y := 60
	w := 220
	h := 380
	if ss.GPEnabled && bot.OwnProgram != nil {
		h = 520 // taller to fit GP program info
	}
	if ss.NeuroEnabled && bot.Brain != nil {
		h = 500 // taller to fit neuro info
	}
	valCol := color.RGBA{200, 200, 220, 255}
	dimCol := color.RGBA{140, 140, 160, 255}
	headerCol := color.RGBA{0, 220, 255, 255}

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), ColorSwarmInfoBg, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, ColorSwarmEditorSep, false)

	lx := x + 5
	ly := y + 5

	// Title + LED swatch
	printColoredAt(screen, fmt.Sprintf("Bot #%d", ss.SelectedBot), lx, ly, ColorSwarmSelected)
	ledCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 255}
	vector.DrawFilledRect(screen, float32(lx+70), float32(ly+2), 10, 10, ledCol, false)
	vector.StrokeRect(screen, float32(lx+70), float32(ly+2), 10, 10, 1, color.RGBA{255, 255, 255, 120}, false)
	ly += lineH + 2

	// --- Position & Bewegung ---
	printColoredAt(screen, "Position & Bewegung", lx, ly, headerCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("X:%.0f Y:%.0f", bot.X, bot.Y), lx, ly, valCol)
	ly += lineH
	degAngle := bot.Angle * 180 / math.Pi
	if degAngle < 0 {
		degAngle += 360
	}
	printColoredAt(screen, fmt.Sprintf("Richtung:%.0f Tempo:%.1f", degAngle, bot.Speed), lx, ly, valCol)
	ly += lineH + 2

	// --- Interner Zustand ---
	printColoredAt(screen, "Interner Zustand", lx, ly, headerCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("State:%d  Counter:%d  Timer:%d", bot.State, bot.Counter, bot.Timer), lx, ly, valCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Value1:%d  Value2:%d", bot.Value1, bot.Value2), lx, ly, valCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("LED: R%d G%d B%d", bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2]), lx, ly, dimCol)
	ly += lineH + 2

	// --- Sensoren ---
	printColoredAt(screen, "Sensoren (Live-Werte)", lx, ly, headerCol)
	ly += lineH
	nearStr := "keiner"
	if bot.NearestIdx >= 0 {
		nearStr = fmt.Sprintf("%.0fpx (Bot #%d)", bot.NearestDist, bot.NearestIdx)
	}
	printColoredAt(screen, fmt.Sprintf("Naechster: %s", nearStr), lx, ly, valCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Nachbarn: %d (in 120px)", bot.NeighborCount), lx, ly, valCol)
	ly += lineH
	obsStr := "Nein"
	if bot.ObstacleAhead {
		obsStr = fmt.Sprintf("Ja (%.0fpx)", bot.ObstacleDist)
	}
	edgeStr := "Nein"
	if bot.OnEdge {
		edgeStr = "Ja"
	}
	printColoredAt(screen, fmt.Sprintf("Hindernis: %s  Rand: %s", obsStr, edgeStr), lx, ly, valCol)
	ly += lineH
	lightStr := "---"
	if ss.Light.Active {
		lightStr = fmt.Sprintf("%d/100", bot.LightValue)
	}
	msgStr := "Nein"
	if bot.ReceivedMsg > 0 {
		msgStr = fmt.Sprintf("Typ %d", bot.ReceivedMsg)
	}
	printColoredAt(screen, fmt.Sprintf("Licht: %s  Msg: %s", lightStr, msgStr), lx, ly, valCol)
	ly += lineH + 2

	// --- Delivery (conditional) ---
	if ss.DeliveryOn {
		printColoredAt(screen, "Paket-Delivery", lx, ly, headerCol)
		ly += lineH
		carryStr := "Leer (sucht)"
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			carryStr = fmt.Sprintf("Traegt: %s", swarm.DeliveryColorName(ss.Packages[bot.CarryingPkg].Color))
		}
		printColoredAt(screen, carryStr, lx, ly, valCol)
		ly += lineH
		pDist := "---"
		if bot.NearestPickupDist < 999 {
			pDist = fmt.Sprintf("%.0fpx", bot.NearestPickupDist)
		}
		dDist := "---"
		if bot.NearestDropoffDist < 999 {
			dDist = fmt.Sprintf("%.0fpx", bot.NearestDropoffDist)
		}
		matchStr := ""
		if bot.DropoffMatch {
			matchStr = " MATCH!"
		}
		printColoredAt(screen, fmt.Sprintf("Pickup:%s Dropoff:%s%s", pDist, dDist, matchStr), lx, ly, valCol)
		ly += lineH + 2
	}

	// --- Sozial ---
	printColoredAt(screen, "Sozial & Ketten", lx, ly, headerCol)
	ly += lineH
	followStr := "None"
	if bot.FollowTargetIdx >= 0 {
		followStr = fmt.Sprintf("#%d", bot.FollowTargetIdx)
	}
	followerStr := "None"
	if bot.FollowerIdx >= 0 {
		followerStr = fmt.Sprintf("#%d", bot.FollowerIdx)
	}
	printColoredAt(screen, fmt.Sprintf("Follow:%s Follower:%s", followStr, followerStr), lx, ly, valCol)
	ly += lineH + 2

	// --- Lifetime Stats ---
	printColoredAt(screen, "Lifetime", lx, ly, headerCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Dist:%.0f Alive:%d", bot.Stats.TotalDistance, bot.Stats.TicksAlive), lx, ly, dimCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Pickups:%d Deliv:%d", bot.Stats.TotalPickups, bot.Stats.TotalDeliveries), lx, ly, dimCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("OK:%d Wrong:%d", bot.Stats.CorrectDeliveries, bot.Stats.WrongDeliveries), lx, ly, dimCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Msg TX:%d RX:%d", bot.Stats.MessagesSent, bot.Stats.MessagesReceived), lx, ly, dimCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Stuck:%d Idle:%d", bot.Stats.AntiStuckCount, bot.Stats.TicksIdle), lx, ly, dimCol)
	ly += lineH + 2

	// --- GP Program (when GP enabled) ---
	if ss.GPEnabled && bot.OwnProgram != nil {
		gpCol := color.RGBA{0, 180, 160, 255}
		printColoredAt(screen, "GP Program", lx, ly, gpCol)
		ly += lineH
		fit := EvaluateGPFitnessRender(bot)
		printColoredAt(screen, fmt.Sprintf("%d rules gen:%d fit:%.0f",
			len(bot.OwnProgram.Rules), ss.GPGeneration, fit), lx, ly, dimCol)
		ly += lineH
		// Show first 5 rules, highlight matched ones
		maxShow := 5
		if maxShow > len(bot.OwnProgram.Rules) {
			maxShow = len(bot.OwnProgram.Rules)
		}
		for ri := 0; ri < maxShow; ri++ {
			ruleCol := color.RGBA{120, 120, 140, 200}
			// Check if this rule matched last tick
			for _, mi := range bot.LastMatchedRules {
				if mi == ri {
					ruleCol = color.RGBA{80, 255, 80, 255} // green = matched
					break
				}
			}
			rule := &bot.OwnProgram.Rules[ri]
			ruleText := swarmscript.RuleToShortText(rule)
			if len(ruleText) > 35 {
				ruleText = ruleText[:35]
			}
			printColoredAt(screen, ruleText, lx, ly, ruleCol)
			ly += lineH - 2
		}
		if len(bot.OwnProgram.Rules) > maxShow {
			printColoredAt(screen, fmt.Sprintf("  ...+%d more", len(bot.OwnProgram.Rules)-maxShow), lx, ly, dimCol)
		}
	}

	// --- Neuro Brain (when NEURO enabled) ---
	if ss.NeuroEnabled && bot.Brain != nil {
		neuroCol := color.RGBA{255, 140, 50, 255}
		printColoredAt(screen, "Neuronales Netz", lx, ly, neuroCol)
		ly += lineH
		fit := EvaluateGPFitnessRender(bot)
		printColoredAt(screen, fmt.Sprintf("Gen:%d  Fitness:%.0f", ss.NeuroGeneration, fit), lx, ly, dimCol)
		ly += lineH

		// Show chosen action
		actionName := "---"
		if bot.Brain.ActionIdx >= 0 && bot.Brain.ActionIdx < len(swarm.NeuroActionNames) {
			actionName = swarm.NeuroActionNames[bot.Brain.ActionIdx]
		}
		printColoredAt(screen, fmt.Sprintf("Aktion: %s", actionName), lx, ly, color.RGBA{255, 255, 100, 255})
		ly += lineH

		// Show top 3 input values
		printColoredAt(screen, "Inputs:", lx, ly, color.RGBA{100, 200, 255, 200})
		ly += lineH
		for inp := 0; inp < swarm.NeuroInputs && inp < 6; inp++ {
			v := bot.Brain.InputVals[inp]
			printColoredAt(screen, fmt.Sprintf(" %s: %.2f", swarm.NeuroInputNames[inp], v), lx, ly, dimCol)
			ly += lineH - 3
		}
		ly += 3

		// Show output activations
		printColoredAt(screen, "Outputs:", lx, ly, color.RGBA{255, 180, 100, 200})
		ly += lineH
		for o := 0; o < swarm.NeuroOutputs; o++ {
			v := bot.Brain.OutputAct[o]
			outCol := dimCol
			if o == bot.Brain.ActionIdx {
				outCol = color.RGBA{255, 255, 100, 255}
			}
			printColoredAt(screen, fmt.Sprintf(" %s: %.2f", swarm.NeuroActionNames[o], v), lx, ly, outCol)
			ly += lineH - 3
		}
	}
}

// EvaluateGPFitnessRender computes GP fitness for rendering (same formula as domain).
func EvaluateGPFitnessRender(bot *swarm.SwarmBot) float64 {
	return float64(bot.Stats.TotalDeliveries)*30 +
		float64(bot.Stats.TotalPickups)*15 +
		bot.Stats.TotalDistance*0.01 -
		float64(bot.Stats.AntiStuckCount)*10 -
		float64(bot.Stats.TicksIdle)*0.05
}

// drawBotComparisonPanel renders a side-by-side comparison of two bots.
func drawBotComparisonPanel(screen *ebiten.Image, ss *swarm.SwarmState) {
	botA := &ss.Bots[ss.SelectedBot]
	botB := &ss.Bots[ss.CompareBot]

	px := 830
	py := 60
	pw := 430
	ph := 300
	headerCol := color.RGBA{0, 220, 255, 255}
	dimCol := color.RGBA{140, 140, 160, 255}
	greenCol := color.RGBA{80, 255, 80, 255}

	// Background
	vector.DrawFilledRect(screen, float32(px), float32(py), float32(pw), float32(ph), ColorSwarmInfoBg, false)
	vector.StrokeRect(screen, float32(px), float32(py), float32(pw), float32(ph), 1, ColorSwarmEditorSep, false)

	lx := px + 5
	ly := py + 5
	colA := lx + 100  // column for Bot A values
	colB := lx + 270  // column for Bot B values

	// Title row
	printColoredAt(screen, "Comparison", lx, ly, headerCol)
	// LED swatches next to bot labels
	printColoredAt(screen, fmt.Sprintf("Bot #%d", ss.SelectedBot), colA, ly, ColorSwarmSelected)
	ledA := color.RGBA{botA.LEDColor[0], botA.LEDColor[1], botA.LEDColor[2], 255}
	vector.DrawFilledRect(screen, float32(colA+55), float32(ly+2), 8, 8, ledA, false)
	printColoredAt(screen, fmt.Sprintf("Bot #%d", ss.CompareBot), colB, ly, color.RGBA{0, 220, 255, 255})
	ledB := color.RGBA{botB.LEDColor[0], botB.LEDColor[1], botB.LEDColor[2], 255}
	vector.DrawFilledRect(screen, float32(colB+55), float32(ly+2), 8, 8, ledB, false)
	ly += lineH + 4

	// Separator
	vector.StrokeLine(screen, float32(px+3), float32(ly), float32(px+pw-3), float32(ly), 1, color.RGBA{60, 60, 80, 200}, false)
	ly += 4

	// Comparison row helper
	compareRow := func(label string, valA, valB float64, higherIsBetter bool) {
		printColoredAt(screen, label, lx, ly, dimCol)
		aStr := fmt.Sprintf("%.0f", valA)
		bStr := fmt.Sprintf("%.0f", valB)
		aCol := dimCol
		bCol := dimCol
		if higherIsBetter {
			if valA > valB {
				aCol = greenCol
			} else if valB > valA {
				bCol = greenCol
			}
		} else {
			if valA < valB {
				aCol = greenCol
			} else if valB < valA {
				bCol = greenCol
			}
		}
		printColoredAt(screen, aStr, colA, ly, aCol)
		printColoredAt(screen, bStr, colB, ly, bCol)
		ly += lineH
	}

	// Stats rows
	compareRow("Distance", botA.Stats.TotalDistance, botB.Stats.TotalDistance, true)
	compareRow("Pickups", float64(botA.Stats.TotalPickups), float64(botB.Stats.TotalPickups), true)
	compareRow("Deliveries", float64(botA.Stats.TotalDeliveries), float64(botB.Stats.TotalDeliveries), true)
	compareRow("Correct", float64(botA.Stats.CorrectDeliveries), float64(botB.Stats.CorrectDeliveries), true)
	compareRow("Wrong", float64(botA.Stats.WrongDeliveries), float64(botB.Stats.WrongDeliveries), false)
	compareRow("Msgs TX", float64(botA.Stats.MessagesSent), float64(botB.Stats.MessagesSent), true)
	compareRow("Msgs RX", float64(botA.Stats.MessagesReceived), float64(botB.Stats.MessagesReceived), true)
	compareRow("Stuck", float64(botA.Stats.AntiStuckCount), float64(botB.Stats.AntiStuckCount), false)
	compareRow("Idle", float64(botA.Stats.TicksIdle), float64(botB.Stats.TicksIdle), false)
	compareRow("Carrying", float64(botA.CarryingPkg+1), float64(botB.CarryingPkg+1), true)
	ly += 2

	// Separator
	vector.StrokeLine(screen, float32(px+3), float32(ly), float32(px+pw-3), float32(ly), 1, color.RGBA{60, 60, 80, 200}, false)
	ly += 4

	// Position info
	printColoredAt(screen, "Position", lx, ly, headerCol)
	printColoredAt(screen, fmt.Sprintf("(%.0f,%.0f)", botA.X, botA.Y), colA, ly, dimCol)
	printColoredAt(screen, fmt.Sprintf("(%.0f,%.0f)", botB.X, botB.Y), colB, ly, dimCol)
	ly += lineH

	// State info
	printColoredAt(screen, "State", lx, ly, headerCol)
	printColoredAt(screen, fmt.Sprintf("S:%d C:%d", botA.State, botA.Counter), colA, ly, dimCol)
	printColoredAt(screen, fmt.Sprintf("S:%d C:%d", botB.State, botB.Counter), colB, ly, dimCol)
	ly += lineH + 2

	printColoredAt(screen, "[Shift+Click to compare, Click to clear]", px+5, py+ph-lineH-2, color.RGBA{80, 80, 100, 200})
}

// deliveryColor returns the RGBA color for a delivery color ID (1-4).
func deliveryColor(c int) color.RGBA {
	switch c {
	case 1:
		return color.RGBA{255, 60, 60, 255} // red
	case 2:
		return color.RGBA{60, 100, 255, 255} // blue
	case 3:
		return color.RGBA{255, 220, 40, 255} // yellow
	case 4:
		return color.RGBA{40, 200, 60, 255} // green
	}
	return color.RGBA{200, 200, 200, 255}
}

// drawDeliveryStations renders pickup and dropoff stations.
func drawDeliveryStations(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for si := range ss.Stations {
		st := &ss.Stations[si]
		sx := float32(offX + st.X)
		sy := float32(offY + st.Y)
		col := deliveryColor(st.Color)
		r := float32(25)

		if st.IsPickup {
			// Pickup: filled circle
			vector.DrawFilledCircle(screen, sx, sy, r, col, false)
			// Outline
			vector.StrokeCircle(screen, sx, sy, r, 2, color.RGBA{col.R, col.G, col.B, 180}, false)
			// Package icon inside (small square) if has package
			if st.HasPackage {
				// Pulse animation
				alpha := uint8(255)
				if ss.Tick%40 < 20 {
					alpha = 200
				}
				pkgCol := color.RGBA{255, 255, 255, alpha}
				vector.DrawFilledRect(screen, sx-5, sy-5, 10, 10, pkgCol, false)
				// Up arrow ↑
				vector.StrokeLine(screen, sx, sy-8, sx, sy+2, 2, color.RGBA{255, 255, 255, 200}, false)
				vector.StrokeLine(screen, sx-3, sy-5, sx, sy-8, 1.5, color.RGBA{255, 255, 255, 200}, false)
				vector.StrokeLine(screen, sx+3, sy-5, sx, sy-8, 1.5, color.RGBA{255, 255, 255, 200}, false)
			} else {
				// Empty: dashed outline look (dimmed)
				dimCol := color.RGBA{col.R / 2, col.G / 2, col.B / 2, 120}
				vector.StrokeCircle(screen, sx, sy, r-4, 1, dimCol, false)
			}
		} else {
			// Dropoff: ring (not filled)
			vector.StrokeCircle(screen, sx, sy, r, 3, col, false)
			// Inner target ring
			vector.StrokeCircle(screen, sx, sy, r*0.5, 2, col, false)
			// Down arrow ↓
			vector.StrokeLine(screen, sx, sy-4, sx, sy+6, 2, col, false)
			vector.StrokeLine(screen, sx-3, sy+3, sx, sy+6, 1.5, col, false)
			vector.StrokeLine(screen, sx+3, sy+3, sx, sy+6, 1.5, col, false)

			// Flash effect on delivery
			if st.FlashTimer > 0 {
				flashAlpha := uint8(st.FlashTimer * 8)
				if flashAlpha > 200 {
					flashAlpha = 200
				}
				var flashCol color.RGBA
				if st.FlashOK {
					flashCol = color.RGBA{0, 255, 0, flashAlpha}
				} else {
					flashCol = color.RGBA{255, 0, 0, flashAlpha}
				}
				flashR := r + float32(30-st.FlashTimer)
				vector.StrokeCircle(screen, sx, sy, flashR, 3, flashCol, false)
			}
		}
	}
}

// drawDeliveryPackages renders packages on the ground with pulsing alpha and arrow to dropoff.
func drawDeliveryPackages(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for pi := range ss.Packages {
		pkg := &ss.Packages[pi]
		if !pkg.Active || pkg.CarriedBy >= 0 || !pkg.OnGround {
			continue
		}
		px := float32(offX + pkg.X)
		py := float32(offY + pkg.Y)
		col := deliveryColor(pkg.Color)
		// Pulsing alpha
		pulse := 0.6 + 0.4*math.Sin(float64(ss.Tick)*0.08)
		col.A = uint8(float64(col.A) * pulse)
		vector.DrawFilledRect(screen, px-5, py-5, 10, 10, col, false)
		vector.StrokeRect(screen, px-5, py-5, 10, 10, 1, color.RGBA{255, 255, 255, 150}, false)
		// Small arrow pointing to matching dropoff
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup || st.Color != pkg.Color {
				continue
			}
			dx := st.X - pkg.X
			dy := st.Y - pkg.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 1 {
				nx := dx / dist
				ny := dy / dist
				ax := px + float32(nx*15)
				ay := py + float32(ny*15)
				arrowCol := color.RGBA{col.R, col.G, col.B, 140}
				vector.StrokeLine(screen, px, py, ax, ay, 1.5, arrowCol, false)
			}
			break // first matching dropoff
		}
	}
}

// drawStationLabels renders "PR","PB","PY","PG" / "DR","DB","DY","DG" labels
// above each station, plus respawn timer bars and delivery counters.
// Labels shift left/right when two stations are closer than 100px to avoid overlap.
func drawStationLabels(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	colorLetters := [5]string{"", "R", "B", "Y", "G"}

	// Pre-compute label X offsets to avoid overlap for nearby stations
	labelOffsetX := make([]int, len(ss.Stations))
	for i := range ss.Stations {
		for j := i + 1; j < len(ss.Stations); j++ {
			dx := ss.Stations[i].X - ss.Stations[j].X
			dy := ss.Stations[i].Y - ss.Stations[j].Y
			if math.Abs(dx) < 100 && math.Abs(dy) < 60 {
				if dx <= 0 { // i is left of j
					labelOffsetX[i] -= 20
					labelOffsetX[j] += 20
				} else {
					labelOffsetX[i] += 20
					labelOffsetX[j] -= 20
				}
			}
		}
	}

	for si := range ss.Stations {
		st := &ss.Stations[si]
		sx := int(offX + st.X)
		sy := int(offY + st.Y)
		prefix := "D"
		if st.IsPickup {
			prefix = "P"
		}
		letter := ""
		if st.Color >= 1 && st.Color <= 4 {
			letter = colorLetters[st.Color]
		}
		label := prefix + letter
		col := deliveryColor(st.Color)
		// Label above station (with horizontal offset for nearby stations)
		lx := sx - 6 + labelOffsetX[si]
		printColoredAt(screen, label, lx, sy-40, col)
		// Subtle type hint below label
		typeHint := "Drop"
		if st.IsPickup {
			typeHint = "Pick"
		}
		printColoredAt(screen, typeHint, lx-2, sy-30, color.RGBA{col.R, col.G, col.B, 100})

		if st.IsPickup {
			// Respawn timer bar below pickup station (15px gap from edge: radius 25 + 15 = 40)
			if !st.HasPackage && st.RespawnIn > 0 {
				barW := 30
				barH := 4
				bx := float32(sx - barW/2)
				by := float32(sy + 40)
				// Background
				vector.DrawFilledRect(screen, bx, by, float32(barW), float32(barH),
					color.RGBA{40, 40, 40, 180}, false)
				// Progress fill
				progress := 1.0 - float64(st.RespawnIn)/100.0
				fillW := float32(float64(barW) * progress)
				vector.DrawFilledRect(screen, bx, by, fillW, float32(barH), col, false)
			}
		} else {
			// Delivery counter below dropoff (15px gap from edge: radius 25 + 15 = 40)
			if st.DeliverCount > 0 {
				countStr := fmt.Sprintf("%d", st.DeliverCount)
				printColoredAt(screen, countStr, sx-len(countStr)*3, sy+40,
					color.RGBA{255, 255, 255, 220})
			}
		}
	}
}

// drawPickupDropoffRoutes draws thin dashed lines between same-color pickup/dropoff pairs.
func drawPickupDropoffRoutes(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for si := range ss.Stations {
		st := &ss.Stations[si]
		if !st.IsPickup {
			continue
		}
		// Find matching dropoff
		for di := range ss.Stations {
			dst := &ss.Stations[di]
			if dst.IsPickup || dst.Color != st.Color {
				continue
			}
			x1 := float32(offX + st.X)
			y1 := float32(offY + st.Y)
			x2 := float32(offX + dst.X)
			y2 := float32(offY + dst.Y)
			col := deliveryColor(st.Color)
			col.A = 40
			drawDashedLine(screen, x1, y1, x2, y2, 8, 6, 1, col)
		}
	}
}

// drawCarryRouteLines draws dashed lines from carrying bots to their matching dropoff.
func drawCarryRouteLines(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.CarryingPkg < 0 || bot.CarryingPkg >= len(ss.Packages) {
			continue
		}
		pkg := &ss.Packages[bot.CarryingPkg]
		// Find nearest matching dropoff
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup || st.Color != pkg.Color {
				continue
			}
			bx := float32(offX + bot.X)
			by := float32(offY + bot.Y)
			dx := float32(offX + st.X)
			dy := float32(offY + st.Y)
			col := deliveryColor(pkg.Color)
			col.A = 60
			drawDashedLine(screen, bx, by, dx, dy, 6, 4, 1, col)
			break // first matching dropoff
		}
	}
}

// drawDashedLine draws a dashed line between two points.
func drawDashedLine(screen *ebiten.Image, x1, y1, x2, y2, segLen, gapLen, width float32, col color.RGBA) {
	dx := x2 - x1
	dy := y2 - y1
	totalLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if totalLen < 1 {
		return
	}
	nx := dx / totalLen
	ny := dy / totalLen
	pos := float32(0)
	for pos < totalLen {
		end := pos + segLen
		if end > totalLen {
			end = totalLen
		}
		sx := x1 + nx*pos
		sy := y1 + ny*pos
		ex := x1 + nx*end
		ey := y1 + ny*end
		vector.StrokeLine(screen, sx, sy, ex, ey, width, col, false)
		pos += segLen + gapLen
	}
}

// drawScorePopups renders floating score text that rises and fades.
func drawScorePopups(screen *ebiten.Image, ss *swarm.SwarmState, offX, offY float64) {
	for i := range ss.ScorePopups {
		sp := &ss.ScorePopups[i]
		alpha := float64(sp.Timer) / 60.0
		if alpha > 1 {
			alpha = 1
		}
		col := color.RGBA{sp.Color[0], sp.Color[1], sp.Color[2], uint8(255 * alpha)}
		sx := int(offX+sp.X) - len(sp.Text)*3
		sy := int(offY + sp.Y)
		printColoredAt(screen, sp.Text, sx, sy, col)
	}
}

// drawDeliveryLegend draws a per-color delivery status legend in the top-right corner.
func drawDeliveryLegend(screen *ebiten.Image, ss *swarm.SwarmState, sw int) {
	colorNames := [5]string{"", "Red", "Blue", "Yel", "Grn"}
	x := sw - 130
	y := 55

	// Background
	vector.DrawFilledRect(screen, float32(x-5), float32(y-5), 128, 78,
		color.RGBA{20, 20, 30, 200}, false)
	vector.StrokeRect(screen, float32(x-5), float32(y-5), 128, 78, 1,
		color.RGBA{100, 100, 120, 150}, false)

	printColoredAt(screen, "Delivery", x, y, color.RGBA{255, 220, 100, 255})
	y += 14

	for c := 1; c <= 4; c++ {
		// Count in-transit for this color
		inTransit := 0
		for bi := range ss.Bots {
			if ss.Bots[bi].CarryingPkg >= 0 && ss.Bots[bi].CarryingPkg < len(ss.Packages) {
				if ss.Packages[ss.Bots[bi].CarryingPkg].Color == c {
					inTransit++
				}
			}
		}
		delivered := ss.DeliveryStats.ColorDelivered[c]
		col := deliveryColor(c)
		// Color swatch
		vector.DrawFilledRect(screen, float32(x), float32(y+2), 8, 8, col, false)
		// Text: "Red: 2→ 5"
		info := fmt.Sprintf("%s:%d>%d", colorNames[c], inTransit, delivered)
		printColoredAt(screen, info, x+12, y, color.RGBA{220, 220, 220, 255})
		y += 14
	}
}

// processDeliveryEvents processes pending delivery events for particle effects.
func processDeliveryEvents(r *Renderer, ss *swarm.SwarmState) {
	if r.SwarmParticles == nil {
		return
	}
	for _, ev := range ss.DeliveryEvents {
		col := deliveryColor(ev.Color)
		if ev.IsPickup {
			// Small burst on pickup
			r.SwarmParticles.Emit(ev.X, ev.Y, 8, col, 1.5, 3, 20)
			if r.Sound != nil && r.Sound.Enabled {
				r.Sound.PlayPickup()
			}
		} else if ev.Correct {
			// Big green burst on correct delivery
			r.SwarmParticles.Emit(ev.X, ev.Y, 20, color.RGBA{80, 255, 80, 255}, 2.0, 4, 30)
			if r.Sound != nil && r.Sound.Enabled {
				r.Sound.PlayDropOK()
			}
		} else {
			// Red burst on wrong delivery
			r.SwarmParticles.Emit(ev.X, ev.Y, 12, color.RGBA{255, 80, 80, 255}, 1.8, 3, 25)
			if r.Sound != nil && r.Sound.Enabled {
				r.Sound.PlayDropFail()
			}
		}
	}
	// Clear consumed events
	ss.DeliveryEvents = ss.DeliveryEvents[:0]
}

// drawSwarmParticles renders particles using fixed arena offsets (no camera).
func drawSwarmParticles(screen *ebiten.Image, ps *ParticleSystem, offX, offY float64) {
	for i := range ps.Particles {
		p := &ps.Particles[i]
		if !p.Active {
			continue
		}
		sx := float32(offX + p.X)
		sy := float32(offY + p.Y)
		alpha := float64(p.Life) / float64(p.MaxLife)
		col := p.Color
		col.A = uint8(float64(col.A) * alpha)
		size := float32(p.Size * alpha)
		if size < 0.5 {
			size = 0.5
		}
		vector.DrawFilledCircle(screen, sx, sy, size, col, false)
	}
}

// drawSwarmRamp renders the loading ramp area on the arena.
func drawSwarmRamp(screen *ebiten.Image, ss *swarm.SwarmState) {
	ts := ss.TruckState
	rx := float32(ts.RampX)
	ry := float32(ts.RampY)
	rw := float32(ts.RampW)
	rh := float32(ts.RampH)

	// Semi-transparent background (brighter)
	vector.DrawFilledRect(screen, rx, ry, rw, rh, color.RGBA{80, 70, 30, 100}, false)

	// Diagonal hatching lines
	hatchStep := float32(20)
	for d := float32(0); d < rw+rh; d += hatchStep {
		x1 := rx + d
		y1 := ry
		x2 := rx
		y2 := ry + d
		if x1 > rx+rw {
			y1 += x1 - (rx + rw)
			x1 = rx + rw
		}
		if y2 > ry+rh {
			x2 += y2 - (ry + rh)
			y2 = ry + rh
		}
		vector.StrokeLine(screen, x1, y1, x2, y2, 1, color.RGBA{140, 120, 50, 80}, false)
	}

	// Bright border (yellow-orange, thick)
	borderCol := color.RGBA{255, 200, 50, 220}
	vector.StrokeRect(screen, rx, ry, rw, rh, 3, borderCol, false)

	// Pulsing inner border
	innerCol := color.RGBA{255, 200, 50, 80}
	vector.StrokeRect(screen, rx+4, ry+4, rw-8, rh-8, 1, innerCol, false)

	// Entrance arrow on the right side (pointing left into ramp)
	arrowX := rx + rw - 10
	arrowCY := ry + rh/2
	arrowCol := color.RGBA{255, 220, 80, 200}
	// Arrow body (horizontal line)
	vector.StrokeLine(screen, arrowX, arrowCY, arrowX-30, arrowCY, 3, arrowCol, false)
	// Arrowhead (chevron pointing left)
	vector.StrokeLine(screen, arrowX-30, arrowCY, arrowX-20, arrowCY-10, 2.5, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY, arrowX-20, arrowCY+10, 2.5, arrowCol, false)

	// Second arrow below
	arrowCY2 := arrowCY + 40
	vector.StrokeLine(screen, arrowX, arrowCY2, arrowX-30, arrowCY2, 3, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY2, arrowX-20, arrowCY2-10, 2.5, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY2, arrowX-20, arrowCY2+10, 2.5, arrowCol, false)

	// Second arrow above
	arrowCY3 := arrowCY - 40
	vector.StrokeLine(screen, arrowX, arrowCY3, arrowX-30, arrowCY3, 3, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY3, arrowX-20, arrowCY3-10, 2.5, arrowCol, false)
	vector.StrokeLine(screen, arrowX-30, arrowCY3, arrowX-20, arrowCY3+10, 2.5, arrowCol, false)

	// "RAMP" label (larger, brighter)
	label := "RAMP"
	lx := int(rx+rw/2) - len(label)*charW/2
	ly := int(ry + 8)
	printColoredAt(screen, label, lx, ly, color.RGBA{255, 220, 80, 255})

	// Truck count / status below label
	if ts.CurrentTruck != nil {
		status := ""
		switch ts.CurrentTruck.Phase {
		case swarm.TruckDrivingIn:
			status = "Truck incoming..."
		case swarm.TruckParked:
			remaining := 0
			for _, p := range ts.CurrentTruck.Packages {
				if !p.PickedUp {
					remaining++
				}
			}
			status = fmt.Sprintf("Pkgs: %d left", remaining)
		case swarm.TruckComplete:
			status = "All picked up!"
		case swarm.TruckDrivingOut:
			status = "Truck leaving..."
		case swarm.TruckWaiting:
			status = "Next truck soon"
		case swarm.TruckRoundDone:
			status = "Round complete!"
		}
		if status != "" {
			printColoredAt(screen, status, int(rx+5), int(ry+rh-18), color.RGBA{200, 180, 80, 200})
		}
	}
}

// drawSwarmTruckVehicle renders the truck body with packages.
func drawSwarmTruckVehicle(screen *ebiten.Image, ss *swarm.SwarmState) {
	t := ss.TruckState.CurrentTruck

	tx := float32(t.X)
	ty := float32(t.Y)

	// Truck body dimensions
	bodyW := float32(80)
	bodyH := float32(40)
	cabinW := float32(18)

	// Cabin (front, darker gray)
	vector.DrawFilledRect(screen, tx, ty, cabinW, bodyH, color.RGBA{70, 70, 80, 255}, false)
	vector.StrokeRect(screen, tx, ty, cabinW, bodyH, 1, color.RGBA{120, 120, 130, 200}, false)
	// Windshield
	vector.DrawFilledRect(screen, tx+2, ty+4, cabinW-4, 12, color.RGBA{140, 180, 220, 200}, false)

	// Cargo area (light gray)
	cargoX := tx + cabinW
	vector.DrawFilledRect(screen, cargoX, ty, bodyW-cabinW, bodyH, color.RGBA{160, 160, 150, 255}, false)
	vector.StrokeRect(screen, cargoX, ty, bodyW-cabinW, bodyH, 1, color.RGBA{120, 120, 130, 200}, false)

	// Wheels
	wheelR := float32(5)
	vector.DrawFilledCircle(screen, tx+12, ty+bodyH+2, wheelR, color.RGBA{40, 40, 40, 255}, false)
	vector.DrawFilledCircle(screen, tx+bodyW-12, ty+bodyH+2, wheelR, color.RGBA{40, 40, 40, 255}, false)

	// Draw packages on cargo
	for _, pkg := range t.Packages {
		if pkg.PickedUp {
			continue
		}
		px := cargoX + float32(pkg.RelX)
		py := ty + float32(pkg.RelY)
		col := deliveryColor(pkg.Color)
		vector.DrawFilledRect(screen, px, py, 8, 8, col, false)
		vector.StrokeRect(screen, px, py, 8, 8, 1, color.RGBA{255, 255, 255, 150}, false)
	}

	// Phase overlay text
	switch t.Phase {
	case swarm.TruckComplete:
		printColoredAt(screen, "COMPLETE", int(tx+10), int(ty-14), color.RGBA{0, 255, 100, 255})
	case swarm.TruckRoundDone:
		// Big overlay handled elsewhere (HUD)
	}
}

// SwarmScreenToArena converts screen coordinates to arena world coordinates.
// Returns the arena position and whether the point is inside the arena.
func SwarmScreenToArena(sx, sy int, ss *swarm.SwarmState) (float64, float64, bool) {
	viewportX := 415.0
	viewportY := 50.0
	viewportW := 800.0
	viewportH := 800.0

	zoom := 1.0
	camX := swarm.SwarmArenaSize / 2
	camY := swarm.SwarmArenaSize / 2
	if ss != nil {
		zoom = ss.SwarmCamZoom
		camX = ss.SwarmCamX
		camY = ss.SwarmCamY
	}

	// Reverse camera transform
	wx := (float64(sx) - viewportX - viewportW/2) / zoom + camX
	wy := (float64(sy) - viewportY - viewportH/2) / zoom + camY
	inside := wx >= 0 && wx <= swarm.SwarmArenaSize && wy >= 0 && wy <= swarm.SwarmArenaSize
	return wx, wy, inside
}

// drawSelectedBotOverlays draws visual tracking overlays for the selected bot on the arena.
func drawSelectedBotOverlays(target *ebiten.Image, ss *swarm.SwarmState) {
	bot := &ss.Bots[ss.SelectedBot]
	bx := float32(bot.X)
	by := float32(bot.Y)

	// Sensor radius circle (cyan, semi-transparent)
	vector.StrokeCircle(target, bx, by, float32(swarm.SwarmSensorRange), 1, color.RGBA{0, 200, 255, 40}, false)

	// Comm radius circle (yellow, semi-transparent)
	vector.StrokeCircle(target, bx, by, float32(swarm.SwarmCommRange), 1, color.RGBA{255, 255, 0, 30}, false)

	// Line to nearest neighbor (white)
	if bot.NearestIdx >= 0 && bot.NearestIdx < len(ss.Bots) {
		near := &ss.Bots[bot.NearestIdx]
		vector.StrokeLine(target, bx, by, float32(near.X), float32(near.Y), 1, color.RGBA{255, 255, 255, 100}, false)
	}

	// Line to follow target (green, thick)
	if bot.FollowTargetIdx >= 0 && bot.FollowTargetIdx < len(ss.Bots) {
		leader := &ss.Bots[bot.FollowTargetIdx]
		vector.StrokeLine(target, bx, by, float32(leader.X), float32(leader.Y), 2.5, color.RGBA{0, 255, 0, 180}, false)
	}

	// Line from follower (red, thick)
	if bot.FollowerIdx >= 0 && bot.FollowerIdx < len(ss.Bots) {
		follower := &ss.Bots[bot.FollowerIdx]
		vector.StrokeLine(target, bx, by, float32(follower.X), float32(follower.Y), 2.5, color.RGBA{255, 0, 0, 180}, false)
	}

	// Highlighted delivery route (carrying bot -> matching dropoff)
	if ss.DeliveryOn && bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
		pkg := &ss.Packages[bot.CarryingPkg]
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if !st.IsPickup && st.Color == pkg.Color {
				col := deliveryColor(pkg.Color)
				col.A = 150
				drawDashedLine(target, bx, by, float32(st.X), float32(st.Y), 8, 4, 2, col)
				break
			}
		}
	}

	// "#N" label above bot
	label := fmt.Sprintf("#%d", ss.SelectedBot)
	lx := int(bx) - len(label)*charW/2
	ly := int(by) - int(swarm.SwarmBotRadius) - 16
	printColoredAt(target, label, lx, ly, color.RGBA{255, 255, 0, 220})
}

// drawGenomeVisualization draws a semi-transparent overlay showing parameter distributions.
func drawGenomeVisualization(screen *ebiten.Image, ss *swarm.SwarmState) {
	panelW := 300
	panelH := 320
	panelX := 700
	panelY := 80

	// Background
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), color.RGBA{20, 10, 30, 220}, false)
	vector.StrokeRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), 2, color.RGBA{180, 50, 180, 200}, false)

	// Title
	title := fmt.Sprintf("Genome (Gen %d)", ss.Generation)
	printColoredAt(screen, title, panelX+10, panelY+8, color.RGBA{220, 150, 255, 255})

	// Parameter bars
	y := panelY + 28
	barW := 200
	for p := 0; p < 26; p++ {
		if !ss.UsedParams[p] {
			continue
		}
		if y > panelY+panelH-40 {
			break
		}
		letter := string(rune('A' + p))

		// Compute min/avg/max across all bots
		minVal := math.MaxFloat64
		maxVal := -math.MaxFloat64
		total := 0.0
		for i := range ss.Bots {
			v := ss.Bots[i].ParamValues[p]
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
			total += v
		}
		avg := total / float64(len(ss.Bots))
		spread := maxVal - minVal
		if spread < 1 {
			spread = 1
		}

		// Label
		paramLabel := fmt.Sprintf("%s: %.1f (%.0f..%.0f)", letter, avg, minVal, maxVal)
		printColoredAt(screen, paramLabel, panelX+10, y, color.RGBA{200, 200, 220, 255})

		// Bar background
		barX := panelX + 10
		barY := y + 14
		vector.DrawFilledRect(screen, float32(barX), float32(barY),
			float32(barW), 8, color.RGBA{40, 30, 50, 255}, false)

		// Draw individual bot parameter dots
		for i := range ss.Bots {
			v := ss.Bots[i].ParamValues[p]
			frac := (v - minVal) / spread
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			dx := float32(float64(barW) * frac)
			dotColor := color.RGBA{120, 60, 180, 100}
			vector.DrawFilledRect(screen, float32(barX)+dx-1, float32(barY), 2, 8, dotColor, false)
		}

		// Average marker
		avgFrac := (avg - minVal) / spread
		avgX := float32(barX) + float32(float64(barW)*avgFrac)
		vector.DrawFilledRect(screen, avgX-2, float32(barY-1), 4, 10, color.RGBA{255, 200, 50, 255}, false)

		y += 28
	}

	// Top 3 bots by fitness
	y += 4
	printColoredAt(screen, "Top Bots:", panelX+10, y, color.RGBA{180, 180, 200, 255})
	y += 14
	type botFit struct {
		idx int
		fit float64
	}
	bots := make([]botFit, len(ss.Bots))
	for i := range ss.Bots {
		bots[i] = botFit{i, ss.Bots[i].Fitness}
	}
	// Simple top-3 selection
	for rank := 0; rank < 3 && rank < len(bots); rank++ {
		bestIdx := rank
		for j := rank + 1; j < len(bots); j++ {
			if bots[j].fit > bots[bestIdx].fit {
				bestIdx = j
			}
		}
		bots[rank], bots[bestIdx] = bots[bestIdx], bots[rank]
		if y > panelY+panelH-14 {
			break
		}
		info := fmt.Sprintf("#%d fit=%.0f", bots[rank].idx, bots[rank].fit)
		printColoredAt(screen, info, panelX+10, y, color.RGBA{160, 160, 180, 255})
		y += 14
	}
}

// drawSwarmFitnessGraph renders a small fitness-over-generations line chart.
func drawSwarmFitnessGraph(screen *ebiten.Image, ss *swarm.SwarmState, gx, gy, gw, gh int) {
	history := ss.FitnessHistory
	n := len(history)
	if n < 2 {
		return
	}

	// Background
	vector.DrawFilledRect(screen, float32(gx), float32(gy), float32(gw), float32(gh),
		color.RGBA{0, 0, 0, 160}, false)

	// Find max fitness for scaling
	maxFit := 1.0
	for _, r := range history {
		if r.Best > maxFit {
			maxFit = r.Best
		}
	}

	// Draw lines: best (green) and avg (yellow)
	for i := 1; i < n; i++ {
		x0 := float32(gx) + float32(i-1)/float32(n-1)*float32(gw)
		x1 := float32(gx) + float32(i)/float32(n-1)*float32(gw)
		// Best fitness line (green)
		y0b := float32(gy+gh) - float32(history[i-1].Best/maxFit)*float32(gh)
		y1b := float32(gy+gh) - float32(history[i].Best/maxFit)*float32(gh)
		vector.StrokeLine(screen, x0, y0b, x1, y1b, 1.5, color.RGBA{80, 255, 80, 220}, false)
		// Avg fitness line (yellow)
		y0a := float32(gy+gh) - float32(history[i-1].Avg/maxFit)*float32(gh)
		y1a := float32(gy+gh) - float32(history[i].Avg/maxFit)*float32(gh)
		vector.StrokeLine(screen, x0, y0a, x1, y1a, 1.5, color.RGBA{255, 200, 50, 200}, false)
	}
}

// drawTeamsScoreboard renders the team score display at the top center of the arena.
func drawTeamsScoreboard(screen *ebiten.Image, ss *swarm.SwarmState, sw int) {
	cx := sw/2 + 100 // offset right (editor panel on left)
	y := 55
	w := 300
	h := 30

	// Background
	vector.DrawFilledRect(screen, float32(cx-w/2), float32(y), float32(w), float32(h),
		color.RGBA{20, 20, 30, 220}, false)
	vector.StrokeRect(screen, float32(cx-w/2), float32(y), float32(w), float32(h), 1,
		color.RGBA{100, 100, 120, 150}, false)

	// Score text
	teamA := fmt.Sprintf("Team A: %d", ss.TeamAScore)
	teamB := fmt.Sprintf("Team B: %d", ss.TeamBScore)
	printColoredAt(screen, teamA, cx-w/2+10, y+8, color.RGBA{80, 120, 255, 255})
	printColoredAt(screen, "vs", cx-10, y+8, color.RGBA{200, 200, 200, 200})
	printColoredAt(screen, teamB, cx+30, y+8, color.RGBA{255, 80, 80, 255})

	// Score bar
	total := ss.TeamAScore + ss.TeamBScore
	if total > 0 {
		barX := float32(cx - w/2 + 5)
		barY := float32(y + h - 5)
		barW := float32(w - 10)
		frac := float32(ss.TeamAScore) / float32(total)
		vector.DrawFilledRect(screen, barX, barY, barW*frac, 3, color.RGBA{80, 120, 255, 200}, false)
		vector.DrawFilledRect(screen, barX+barW*frac, barY, barW*(1-frac), 3, color.RGBA{255, 80, 80, 200}, false)
	}

	// Challenge overlay
	if ss.ChallengeActive {
		if ss.ChallengeResult != "" {
			// Show winner
			resultCol := color.RGBA{255, 220, 50, 255}
			printColoredAt(screen, ss.ChallengeResult, cx-len(ss.ChallengeResult)*3, y+h+5, resultCol)
		} else {
			// Show remaining ticks
			chalText := fmt.Sprintf("Challenge: %d ticks left", ss.ChallengeTicks)
			printColoredAt(screen, chalText, cx-len(chalText)*3, y+h+5, color.RGBA{200, 200, 100, 220})
		}
	}
}

