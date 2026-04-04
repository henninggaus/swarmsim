package render

import (
	"image"
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"swarmsim/locale"

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

	// Aurora background effect (before grid, for subtle under-layer)
	if ss.AuroraOn {
		drawAurora(a, ss.Tick)
	}

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
	// Uses spatial hash for O(m·k) instead of O(m·n) where k = nearby bots.
	if ss.ShowCommGraph && len(ss.PrevMessages) > 0 {
		commRange := float32(swarm.SwarmCommRange)
		commRangeSq := commRange * commRange
		for _, msg := range ss.PrevMessages {
			sx := float32(msg.X)
			sy := float32(msg.Y)
			// Draw small broadcast ring at sender
			vector.StrokeCircle(a, sx, sy, commRange, 1, color.RGBA{100, 200, 255, 40}, false)
			// Draw lines to receiving bots (within comm range) via spatial hash
			nearIDs := ss.Hash.Query(msg.X, msg.Y, swarm.SwarmCommRange)
			for _, bi := range nearIDs {
				if bi < 0 || bi >= len(ss.Bots) {
					continue
				}
				bot := &ss.Bots[bi]
				dx := float32(bot.X) - sx
				dy := float32(bot.Y) - sy
				distSq := dx*dx + dy*dy
				if distSq < commRangeSq && distSq > 4 {
					alpha := uint8(120 - 80*distSq/commRangeSq)
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

	// Draw A* nav grid debug overlay (Shift+G toggle)
	if ss.ShowNavGrid && ss.AStarOn && ss.AStar != nil {
		drawNavGridOverlay(a, ss)
	}

	// Draw A* path overlay (Shift+P toggle)
	if ss.ShowPaths && ss.AStarOn && ss.AStar != nil {
		drawPathOverlay(a, ss)
	}

	// Draw flocking velocity overlay (Shift+F toggle)
	if ss.ShowFlock {
		drawFlockOverlay(a, ss)
	}

	// Draw role color overlay (Shift+R toggle)
	if ss.ShowRoles {
		drawRoleOverlay(a, ss)
	}

	// Draw firefly flash overlay (9 key toggle)
	if ss.ShowFirefly && ss.FireflyOn && ss.Firefly != nil {
		drawFireflyOverlay(a, ss)
	}

	// Draw vortex rotation overlay (0 key toggle)
	if ss.ShowVortex {
		drawVortexOverlay(a, ss)
	}

	// Draw morphogen pattern overlay (Shift+M toggle)
	if ss.ShowMorphogen && ss.MorphogenOn && ss.Morphogen != nil {
		drawMorphogenOverlay(a, ss)
	}

	// Draw evasion wave overlay (Shift+E toggle)
	if ss.ShowEvasion && ss.EvasionOn && ss.Evasion != nil {
		drawEvasionOverlay(a, ss)
	}

	// Draw slime trail overlay (Shift+S toggle)
	if ss.ShowSlime && ss.SlimeOn && ss.Slime != nil {
		drawSlimeOverlay(a, ss)
	}

	// Draw bridge overlay (Shift+B toggle)
	if ss.ShowBridge && ss.BridgeOn && ss.Bridge != nil {
		drawBridgeOverlay(a, ss)
	}

	// Draw wave overlay (Shift+W toggle)
	if ss.ShowWave && ss.WaveOn && ss.Wave != nil {
		drawWaveOverlay(a, ss)
	}

	// Draw shepherd overlay (Shift+H toggle)
	if ss.ShowShepherd && ss.ShepherdOn && ss.Shepherd != nil {
		drawShepherdOverlay(a, ss)
	}

	// Draw fitness landscape heatmap overlay — only in Algo-Labor mode (F4)
	if ss.AlgoLaborMode && ss.ShowPSO && ss.SwarmAlgo != nil {
		r.drawFitnessLandscapeOverlay(a, ss)
	}

	// Draw magnetic chain overlay (Shift+G toggle)
	if ss.ShowMagnetic && ss.MagneticOn && ss.Magnetic != nil {
		drawMagneticOverlay(a, ss)
	}

	// Draw division overlay (Shift+D toggle)
	if ss.ShowDivision && ss.DivisionOn && ss.Division != nil {
		drawDivisionOverlay(a, ss)
	}

	// Draw V-Formation overlay (Shift+V toggle)
	if ss.ShowVFormation && ss.VFormationOn && ss.VFormation != nil {
		drawVFormationOverlay(a, ss)
	}

	// Draw brood sorting overlay (Shift+O toggle)
	if ss.ShowBrood && ss.BroodOn && ss.Brood != nil {
		drawBroodOverlay(a, ss)
	}

	// Draw jellyfish pulse overlay (Shift+J toggle)
	if ss.ShowJellyfish && ss.JellyfishOn && ss.Jellyfish != nil {
		drawJellyfishOverlay(a, ss)
	}

	// Draw immune system overlay (Shift+I toggle)
	if ss.ShowImmune && ss.ImmuneSwarmOn && ss.ImmuneSwarm != nil {
		drawImmuneOverlay(a, ss)
	}

	// Draw gravity overlay (Shift+Y toggle — was genome, now also gravity w/ Ctrl)
	if ss.ShowGravity && ss.GravityOn && ss.Gravity != nil {
		drawGravityOverlay(a, ss)
	}

	// Draw crystal overlay (Shift+K toggle)
	if ss.ShowCrystal && ss.CrystalOn && ss.Crystal != nil {
		drawCrystalOverlay(a, ss)
	}

	// Draw amoeba overlay (Shift+A toggle)
	if ss.ShowAmoeba && ss.AmoebaOn && ss.Amoeba != nil {
		drawAmoebaOverlay(a, ss)
	}

	// Draw ACO overlay (Shift+Q toggle)
	if ss.ShowACO && ss.ACOOn && ss.ACO != nil {
		drawACOOverlay(a, ss)
	}

	// ── Algo overlays — only rendered in Algo-Labor mode (F4) ──
	if ss.AlgoLaborMode {

	// Draw GWO overlay (6 key toggle) — lines from wolves to alpha/beta/delta
	if ss.ShowGWO && ss.GWOOn && ss.GWO != nil {
		drawGWOOverlay(a, ss)
	}

	// Draw WOA overlay (Shift+X toggle) — spiral paths around best whale
	if ss.ShowWOA && ss.WOAOn && ss.WOA != nil {
		drawWOAOverlay(a, ss)
	}

	// Draw BFO overlay (Shift+Z toggle) — nutrient field and swim/tumble markers
	if ss.ShowBFO && ss.BFOOn && ss.BFO != nil {
		drawBFOOverlay(a, ss)
	}

	// Draw MFO overlay (Shift+R toggle) — lines from moths to their flames
	if ss.ShowMFO && ss.MFOOn && ss.MFO != nil {
		drawMFOOverlay(a, ss)
	}

	// Draw Cuckoo Search overlay — lines from nests to global best
	if ss.ShowCuckoo && ss.CuckooOn && ss.Cuckoo != nil {
		drawCuckooOverlay(a, ss)
	}

	// Draw Differential Evolution overlay — trial position targets
	if ss.ShowDE && ss.DEOn && ss.DE != nil {
		drawDEOverlay(a, ss)
	}

	// Draw Artificial Bee Colony overlay — best food source and role indicators
	if ss.ShowABC && ss.ABCOn && ss.ABC != nil {
		drawABCOverlay(a, ss)
	}

	// Draw Harmony Search overlay — best harmony and HM positions
	if ss.ShowHSO && ss.HSOOn && ss.HSO != nil {
		drawHSOOverlay(a, ss)
	}

	// Draw Bat Algorithm overlay — echolocation pulse rings and best bat marker
	if ss.ShowBat && ss.BatOn && ss.Bat != nil {
		drawBatOverlay(a, ss)
	}

	// Draw Harris Hawks Optimization overlay — hunting phase indicators and prey marker
	if ss.ShowHHO && ss.HHOOn && ss.HHO != nil {
		drawHHOOverlay(a, ss)
	}

	// Draw Salp Swarm Algorithm overlay — chain links and food source
	if ss.ShowSSA && ss.SSAOn && ss.SSA != nil {
		drawSSAOverlay(a, ss)
	}

	// Draw Gravitational Search Algorithm overlay — mass rings and force vectors
	if ss.ShowGSA && ss.GSAOn && ss.GSA != nil {
		drawGSAOverlay(a, ss)
	}

	// Draw Flower Pollination Algorithm overlay — pollination types and global best
	if ss.ShowFPA && ss.FPAOn && ss.FPA != nil {
		drawFPAOverlay(a, ss)
	}

	// Draw Simulated Annealing overlay — temperature halos and target lines
	if ss.ShowSA && ss.SAOn && ss.SA != nil {
		drawSAOverlay(a, ss)
	}

	// Draw Aquila Optimizer overlay — hunting phases and prey marker
	if ss.ShowAO && ss.AOOn && ss.AO != nil {
		drawAOOverlay(a, ss)
	}

	// Draw Sine Cosine Algorithm overlay — sine/cosine phase indicators
	if ss.ShowSCA && ss.SCAOn && ss.SCA != nil {
		drawSCAOverlay(a, ss)
	}

	// Draw Dragonfly Algorithm overlay — roles, step vectors, food/enemy
	if ss.ShowDA && ss.DAOn && ss.DA != nil {
		drawDAOverlay(a, ss)
	}

	// Draw TLBO overlay — teacher/learner phases and peer links
	if ss.ShowTLBO && ss.TLBOOn && ss.TLBO != nil {
		drawTLBOOverlay(a, ss)
	}

	// Draw Equilibrium Optimizer overlay — phases and pool positions
	if ss.ShowEO && ss.EOOn && ss.EO != nil {
		drawEOOverlay(a, ss)
	}

	// Draw Jaya overlay — best/worst markers and fitness gradient
	if ss.ShowJaya && ss.JayaOn && ss.Jaya != nil {
		drawJayaOverlay(a, ss)
	}

	} // end AlgoLaborMode guard

	// Draw message wave rings
	if ss.ShowMsgWaves {
		for _, w := range ss.MsgWaves {
			progress := 1.0 - float64(w.Timer)/30.0
			alpha := uint8(120 * (1.0 - progress))
			if alpha < 5 {
				continue
			}
			// Color based on broadcast value
			var wr, wg, wb uint8
			if w.Value > 0 {
				wr, wg, wb = 100, 200, 255 // blue-cyan for positive
			} else if w.Value < 0 {
				wr, wg, wb = 255, 100, 100 // red for negative
			} else {
				wr, wg, wb = 200, 200, 200 // gray for zero
			}
			wCol := color.RGBA{wr, wg, wb, alpha}
			vector.StrokeCircle(a, float32(w.X), float32(w.Y), float32(w.Radius), 1.5, wCol, false)
		}
	}

	// Prediction arrows overlay (before bots)
	if ss.ShowPrediction {
		drawPredictionArrows(a, ss)
	}

	// Congestion zone overlay (before bots)
	if ss.ShowZones {
		drawCongestionOverlay(a, ss)
	}

	// Swarm center of mass overlay (before bots)
	if ss.ShowSwarmCenter {
		drawSwarmCenterOverlay(a, ss)
	}

	// Voronoi territory overlay (before bots)
	if ss.ShowVoronoi {
		drawVoronoiOverlay(a, ss)
	}

	// Velocity flow field overlay (before bots)
	if ss.ShowFlowField {
		drawFlowField(a, ss)
	}

	// Fitness gradient field overlay (before bots)
	if ss.ShowFitnessGradient {
		drawFitnessGradientField(a, ss)
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
		vector.StrokeLine(a, bx, by, bx+dx, by+dy, 1.5, ColorWhiteFaded, false)

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

	// Dash speed lines (on top of bots)
	drawDashSpeedLines(a, ss)

	// Day/Night overlay (on top of everything in arena)
	if ss.DayNightOn {
		drawDayNightOverlay(a, ss)
	}

	// Selected bot visual overlays (on arena, before blit)
	if ss.SelectedBot >= 0 && ss.SelectedBot < len(ss.Bots) {
		drawSelectedBotOverlays(a, ss)
	}

	// Concept overlay (C key) — educational visual explanations on arena
	if ss.ShowConceptOverlay {
		DrawConceptOverlay(a, ss)
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

	// Track which right-side panels are active to avoid overlaps.
	botInfoVisible := !ss.DashboardOn && ss.SelectedBot >= 0 && ss.SelectedBot < len(ss.Bots)
	comparisonVisible := !ss.DashboardOn && ss.CompareBot >= 0 && ss.CompareBot < len(ss.Bots) && botInfoVisible

	// Delivery legend — hide when a right-side panel would overlap it.
	if ss.DeliveryOn && !botInfoVisible && !ss.DashboardOn {
		drawDeliveryLegend(screen, ss, sw)
	}

	// Selected bot info overlay / comparison panel — skip when dashboard is open.
	if comparisonVisible {
		drawBotComparisonPanel(screen, ss)
	} else if botInfoVisible {
		drawSelectedBotInfo(screen, ss)
	}

	// Live math trace overlay (K key)
	if ss.ShowMathTrace {
		DrawMathOverlay(screen, ss)
	}

	// Decision trace overlay (D key) — shows which rule fired and why
	if ss.ShowDecisionTrace {
		DrawDecisionTrace(screen, ss)
	}

	// Did You Know educational tips (auto-rotating)
	DrawDidYouKnow(screen, ss)

	// Glossary overlay (G key) — scrollable term reference
	if ss.ShowGlossary {
		DrawGlossary(screen, ss)
	}

	// Truck round complete overlay
	if ss.TruckToggle && ss.TruckState != nil && ss.TruckState.CurrentTruck != nil &&
		ss.TruckState.CurrentTruck.Phase == swarm.TruckRoundDone {
		roundText := locale.Tf("ui.round_complete", ss.TruckState.Score)
		rtW := runeLen(roundText) * charW
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

	// Tick HUD string cache once per frame
	upd := hudCacheTick()

	// Follow-cam HUD indicator
	if ss.FollowCamBot >= 0 && ss.FollowCamBot < len(ss.Bots) {
		label := cachedFollowCam(upd, ss.FollowCamBot)
		printColoredAt(screen, label, 420, 838, color.RGBA{0, 255, 255, 220})
	}

	// Evolution HUD + fitness graph
	if ss.EvolutionOn {
		evoInfo := cachedEvoInfo(upd, ss.Generation, ss.BestFitness, ss.AvgFitness, ss.EvolutionTimer, simulation.EvolutionInterval)
		printColoredAt(screen, evoInfo, 420, 48, color.RGBA{180, 50, 180, 255})
		// Fitness graph (150x50px)
		if len(ss.FitnessHistory) > 1 {
			drawSwarmFitnessGraph(screen, ss, 420, 60, 150, 50)
		}
	}

	// GP HUD + fitness graph
	if ss.GPEnabled {
		gpInfo := cachedGPInfo(upd, ss.GPGeneration, ss.BestFitness, ss.AvgFitness, ss.GPTimer, simulation.GPEvolutionInterval)
		printColoredAt(screen, gpInfo, 420, 48, color.RGBA{0, 180, 160, 255})
		if len(ss.FitnessHistory) > 1 {
			drawSwarmFitnessGraph(screen, ss, 420, 60, 150, 50)
		}
	}

	// Scenario chain HUD
	if ss.ScenarioChain != nil {
		chain := ss.ScenarioChain
		if chain.Active {
			step := &chain.Steps[chain.StepIdx]
			chainInfo := cachedChainInfo(upd, step.Name, step.TickLimit-chain.Timer, step.TickLimit, chain.TotalScore)
			vector.DrawFilledRect(screen, 360, float32(sh-76), float32(sw-365), 16,
				color.RGBA{20, 60, 80, 200}, false)
			printColoredAt(screen, chainInfo, 365, sh-75, color.RGBA{80, 220, 255, 255})
		} else if chain.Complete {
			result := cachedChainDone(upd, chain.TotalScore, chain.StepScores)
			vector.DrawFilledRect(screen, 360, float32(sh-76), float32(sw-365), 16,
				color.RGBA{20, 80, 20, 200}, false)
			printColoredAt(screen, result, 365, sh-75, color.RGBA{80, 255, 120, 255})
		}
	}

	// Auto-Optimizer HUD
	if ss.AutoOptimizer != nil && ss.AutoOptimizer.Active {
		opt := ss.AutoOptimizer
		optInfo := cachedOptInfo(upd, opt.Trial, opt.MaxTrials, opt.CurrentScore, opt.BestScore)
		// Background bar
		vector.DrawFilledRect(screen, 360, float32(sh-60), float32(sw-365), 16,
			color.RGBA{80, 20, 20, 200}, false)
		printColoredAt(screen, optInfo, 365, sh-59, ColorSectionGold)
	}

	// Genome visualization overlay (V key)
	if ss.ShowGenomeViz && ss.EvolutionOn {
		drawGenomeVisualization(screen, ss)
	}

	// Genom-Browser overlay (G key)
	if ss.GenomeBrowserOn {
		DrawGenomeBrowser(screen, ss)
	}

	// Speciation overlay (Shift+E)
	if ss.ShowSpeciation && ss.Speciation != nil {
		DrawSpeciationOverlay(screen, ss)
	}

	// Pattern detection overlay (Shift+F)
	if ss.ShowPatterns && ss.PatternResult != nil {
		DrawPatternOverlay(screen, ss)
	}

	// Achievement overlay (Shift+B)
	if ss.ShowAchievements && ss.AchievementState != nil {
		DrawAchievementOverlay(screen, ss)
	}

	// Achievement popup (always drawn when active)
	if ss.AchievementState != nil {
		ss.AchievementState.UpdatePopup()
		DrawAchievementPopup(screen, ss)
	}

	// Tournament overlay (U key)
	if ss.TournamentOn {
		DrawTournamentOverlay(screen, ss)
	}

	// Formation analysis overlay (F6 key)
	if ss.ShowFormation {
		DrawFormationOverlay(screen, ss)
	}

	// Live chart overlay (. key)
	if ss.ShowLiveChart {
		DrawLiveChart(screen, ss)
	}

	// Pareto front overlay (when Pareto evolution active)
	if ss.ParetoEnabled && ss.ParetoFront != nil {
		DrawParetoOverlay(screen, ss)
	}

	// Leaderboard overlay (Ctrl+L)
	if ss.ShowLeaderboard && ss.Leaderboard != nil {
		DrawLeaderboardOverlay(screen, ss.Leaderboard)
	}

	// Self-Programming Swarm: "Detecting..." progress indicator
	if ss.CollectiveAIOn && ss.IssueBoard != nil && len(ss.IssueBoard.Issues) == 0 {
		pulse := 0.5 + 0.5*math.Sin(float64(ss.Tick)*0.1)
		alpha := uint8(100 + int(pulse*100))
		msg := locale.T("collective.detecting")
		printColoredAt(screen, msg, sw/2-runeLen(msg)*charW/2, 40, color.RGBA{255, 180, 50, alpha})
	}

	// Self-Programming Swarm issue board (I key)
	DrawIssueBoard(screen, ss)

	// Arena editor mode indicator
	if ss.ArenaEditMode {
		toolNames := []string{locale.T("ui.tool_obstacle"), locale.T("ui.tool_station"), locale.T("ui.tool_delete")}
		toolColors := []color.RGBA{
			{180, 120, 60, 255},
			{60, 180, 120, 255},
			{255, 80, 80, 255},
		}
		label := locale.Tf("ui.arena_editor_bar", toolNames[ss.ArenaEditTool])
		// Background bar
		vector.DrawFilledRect(screen, 360, float32(sh-46), float32(sw-365), 16,
			color.RGBA{20, 20, 40, 200}, false)
		printColoredAt(screen, label, 365, sh-45, toolColors[ss.ArenaEditTool])
	}

	// Statistics dashboard
	if ss.DashboardOn && ss.StatsTracker != nil {
		DrawDashboard(screen, ss, sw-260, 55, 250, sh-70)
	}

	// Convergence graph for swarm algorithms (auto-shows when algorithm is active)
	if ss.SwarmAlgoOn && ss.SwarmAlgo != nil && len(ss.SwarmAlgo.ConvergenceHistory) >= 2 {
		drawConvergenceGraph(screen, ss)
		drawSearchTrajectory(screen, ss)
		drawFitnessHistogram(screen, ss)
	}

	// Algorithm overlays — only in Algo-Labor mode (F4)
	if ss.AlgoLaborMode {
		// Algorithm performance scoreboard (shows when 2+ algorithms have been tested)
		if len(ss.AlgoScoreboard) >= 1 {
			drawAlgoScoreboard(screen, ss)
		}

		// Algorithm radar chart (shows when 2+ algorithms have been tested and toggled on)
		if ss.ShowAlgoRadar && len(ss.AlgoScoreboard) >= 2 {
			drawAlgoRadarChart(screen, ss)
		}

		// Algorithm auto-tournament progress bar
		if ss.AlgoTournamentOn {
			drawAlgoTournamentProgress(screen, ss)
		}
	}

	// Parameter tweaker overlay (Shift+P)
	DrawParamTweaker(screen, ss)

	// Performance monitor overlay (F12)
	DrawPerfMonitor(screen, ss)

	// Shortcut reference card (?)
	DrawShortcutCard(screen, ss)

	// Minimap
	if r.ShowMinimap {
		r.drawSwarmMinimap(screen, ss)
	}

	// Separator line between editor and arena
	vector.StrokeLine(screen, 350, 0, 350, float32(sh), 2, ColorSwarmEditorSep, false)
}

