package render

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
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

	// Truck round complete overlay
	if ss.TruckToggle && ss.TruckState != nil && ss.TruckState.CurrentTruck != nil &&
		ss.TruckState.CurrentTruck.Phase == swarm.TruckRoundDone {
		roundText := fmt.Sprintf("RUNDE ABGESCHLOSSEN! Punkte: %d", ss.TruckState.Score)
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
	compareRow("Lieferungen", float64(botA.Stats.TotalDeliveries), float64(botB.Stats.TotalDeliveries), true)
	compareRow("Richtig", float64(botA.Stats.CorrectDeliveries), float64(botB.Stats.CorrectDeliveries), true)
	compareRow("Falsch", float64(botA.Stats.WrongDeliveries), float64(botB.Stats.WrongDeliveries), false)
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

	printColoredAt(screen, "[Klick = abwaehlen, Shift+Klick = vergleichen]", px+5, py+ph-lineH-2, color.RGBA{80, 80, 100, 200})
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
// When evolution or GP is active the scoreboard shifts down to avoid the fitness graph.
func drawTeamsScoreboard(screen *ebiten.Image, ss *swarm.SwarmState, sw int) {
	cx := sw/2 + 100 // offset right (editor panel on left)
	y := 55
	if ss.EvolutionOn || ss.GPEnabled {
		y = 115 // below the fitness graph
	}
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

// drawPathOverlay renders computed A* paths with animated particles and color coding.
// Cyan = searching for pickup, Gold = carrying to dropoff.
func drawPathOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.AStar
	if st == nil {
		return
	}

	// Animation phase based on tick (flowing particles)
	phase := float64(ss.Tick%60) / 60.0

	for i := range ss.Bots {
		if i >= len(st.Paths) || st.Paths[i] == nil {
			continue
		}
		path := st.Paths[i]
		pidx := st.PathIdx[i]
		if pidx >= len(path) {
			continue
		}

		bot := &ss.Bots[i]

		// Color coding: cyan = searching, gold = carrying
		var baseR, baseG, baseB uint8
		if bot.CarryingPkg >= 0 {
			baseR, baseG, baseB = 255, 200, 50 // gold
		} else {
			baseR, baseG, baseB = 0, 220, 220 // cyan
		}

		// Draw line from bot to first waypoint (bright lead segment)
		wp := path[pidx]
		vector.StrokeLine(a,
			float32(bot.X), float32(bot.Y),
			float32(wp.X), float32(wp.Y),
			2.0, color.RGBA{baseR, baseG, baseB, 120}, false)

		// Draw path segments with distance-based fade
		for j := pidx; j < len(path)-1; j++ {
			dist := j - pidx
			alpha := uint8(90)
			width := float32(1.5)
			if dist > 3 {
				alpha = 55
				width = 1.0
			}
			if dist > 10 {
				alpha = 30
				width = 0.8
			}
			vector.StrokeLine(a,
				float32(path[j].X), float32(path[j].Y),
				float32(path[j+1].X), float32(path[j+1].Y),
				width, color.RGBA{baseR, baseG, baseB, alpha}, false)
		}

		// Draw waypoint dots
		for j := pidx; j < len(path); j++ {
			r := float32(1.5)
			alpha := uint8(50)
			if j == pidx {
				r = 3.0
				alpha = 180
			} else if j == len(path)-1 {
				r = 4.0
				alpha = 200 // bright goal marker
			}
			vector.DrawFilledCircle(a, float32(path[j].X), float32(path[j].Y),
				r, color.RGBA{baseR, baseG, baseB, alpha}, false)
		}

		// Animated flowing particles along path
		// Place 3 particles per bot, evenly spaced and moving with phase
		totalSegs := len(path) - pidx
		if totalSegs > 0 {
			for p := 0; p < 3; p++ {
				// Particle position along path (0.0 to 1.0)
				t := math.Mod(phase+float64(p)*0.33, 1.0)
				segF := t * float64(totalSegs)
				segIdx := int(segF)
				segFrac := segF - float64(segIdx)

				j := pidx + segIdx
				if j >= len(path)-1 {
					j = len(path) - 2
					segFrac = 1.0
				}
				if j < pidx {
					continue
				}

				// Interpolate position
				px := path[j].X + (path[j+1].X-path[j].X)*segFrac
				py := path[j].Y + (path[j+1].Y-path[j].Y)*segFrac

				// Pulsing glow
				pulse := uint8(140 + 80*math.Sin(phase*math.Pi*2+float64(p)*2.1))
				vector.DrawFilledCircle(a, float32(px), float32(py),
					3.5, color.RGBA{baseR, baseG, baseB, pulse}, false)
			}
		}

		// Goal marker: pulsing ring at destination
		if len(path) > 0 {
			goal := path[len(path)-1]
			pulseR := float32(6 + 3*math.Sin(phase*math.Pi*2))
			vector.StrokeCircle(a, float32(goal.X), float32(goal.Y),
				pulseR, 1.5, color.RGBA{baseR, baseG, baseB, 100}, false)
		}
	}
}

// drawNavGridOverlay renders the A* navigation grid debug visualization.
// Green cells = free, Red cells = blocked by obstacles.
func drawNavGridOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.AStar
	if st == nil {
		return
	}

	cellF := float32(st.CellSize)
	for r := 0; r < st.GridRows; r++ {
		for c := 0; c < st.GridCols; c++ {
			idx := r*st.GridCols + c
			x := float32(c) * cellF
			y := float32(r) * cellF

			if st.Blocked[idx] {
				// Blocked: semi-transparent red
				vector.DrawFilledRect(a, x, y, cellF, cellF,
					color.RGBA{200, 40, 40, 50}, false)
			} else {
				// Free: very faint green
				vector.DrawFilledRect(a, x, y, cellF, cellF,
					color.RGBA{30, 160, 30, 15}, false)
			}
		}
	}

	// Draw grid lines (very subtle)
	for r := 0; r <= st.GridRows; r++ {
		y := float32(r) * cellF
		vector.StrokeLine(a, 0, y, float32(st.GridCols)*cellF, y,
			0.5, color.RGBA{80, 80, 80, 30}, false)
	}
	for c := 0; c <= st.GridCols; c++ {
		x := float32(c) * cellF
		vector.StrokeLine(a, x, 0, x, float32(st.GridRows)*cellF,
			0.5, color.RGBA{80, 80, 80, 30}, false)
	}
}

// drawFlockOverlay draws velocity lines and separation zones for flocking bots.
func drawFlockOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		bx := float32(bot.X)
		by := float32(bot.Y)

		// Velocity line (heading * speed, cyan)
		if bot.Speed > 0.1 {
			vlen := float32(20.0)
			ex := bx + float32(math.Cos(bot.Angle))*vlen
			ey := by + float32(math.Sin(bot.Angle))*vlen
			vector.StrokeLine(a, bx, by, ex, ey, 1.0,
				color.RGBA{0, 200, 220, 120}, false)
		}

		// Separation urgency ring (red, radius proportional to urgency)
		if bot.FlockSeparation > 20 {
			alpha := uint8(bot.FlockSeparation * 2)
			if alpha > 180 {
				alpha = 180
			}
			r := float32(4 + bot.FlockSeparation/10)
			vector.StrokeCircle(a, bx, by, r, 1.0,
				color.RGBA{255, 60, 60, alpha}, false)
		}

		// Alignment indicator (green arc in heading direction)
		if bot.FlockAlign != 0 {
			alignRad := float64(bot.FlockAlign) * math.Pi / 180
			ex := bx + float32(math.Cos(bot.Angle+alignRad))*12
			ey := by + float32(math.Sin(bot.Angle+alignRad))*12
			vector.StrokeLine(a, bx, by, ex, ey, 0.8,
				color.RGBA{0, 220, 80, 100}, false)
		}
	}
}

// drawRoleOverlay draws colored rings and labels for bot roles.
func drawRoleOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		bx := float32(bot.X)
		by := float32(bot.Y)
		r := float32(swarm.SwarmBotRadius + 3)

		var c color.RGBA
		var label string
		switch bot.Role {
		case swarm.BotRoleScout:
			c = color.RGBA{0, 200, 255, 150}
			label = "S"
		case swarm.BotRoleWorker:
			c = color.RGBA{255, 200, 0, 150}
			label = "W"
		case swarm.BotRoleGuard:
			c = color.RGBA{255, 50, 50, 150}
			label = "G"
		default:
			continue // no role, skip
		}

		// Colored ring
		vector.StrokeCircle(a, bx, by, r, 1.5, c, false)

		// Role letter
		printColoredAt(a, label, int(bx)-3, int(by)-14, c)

		// Reputation indicator (small bar below bot)
		if bot.Reputation < 80 && bot.Reputation > 0 {
			// Low reputation: orange/red warning
			repFrac := float32(bot.Reputation) / 100.0
			barW := float32(16) * repFrac
			barColor := color.RGBA{255, uint8(200 * repFrac), 0, 180}
			vector.DrawFilledRect(a, bx-8, by+r+2, barW, 2, barColor, false)
		}
	}

	// Quorum decision indicators
	if ss.QuorumOn && ss.Quorum != nil {
		for _, dec := range ss.Quorum.Decisions {
			cx := float32(dec.CenterX)
			cy := float32(dec.CenterY)
			radius := float32(20 + dec.Participants*3)
			alpha := uint8(100 * dec.Strength)
			if alpha < 30 {
				alpha = 30
			}

			// Pulsing circle at decision center
			phase := float32(ss.Tick%60) / 60.0
			pulseR := radius * (0.8 + 0.4*phase)

			var qc color.RGBA
			switch dec.Proposal {
			case 0: // Migrate
				qc = color.RGBA{100, 200, 255, alpha}
			case 1: // Cluster
				qc = color.RGBA{255, 255, 100, alpha}
			case 2: // Disperse
				qc = color.RGBA{100, 255, 100, alpha}
			case 3: // Alarm
				qc = color.RGBA{255, 80, 80, alpha}
			}
			vector.StrokeCircle(a, cx, cy, pulseR, 1.5, qc, false)
		}
	}
}

// drawFireflyOverlay draws flash pulse indicators for firefly synchronization.
func drawFireflyOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		bx := float32(bot.X)
		by := float32(bot.Y)

		// Phase indicator: ring that fills up as phase approaches flash
		phase := float32(bot.FlashPhase) / 255.0
		r := float32(swarm.SwarmBotRadius) + 2 + phase*6

		if bot.FlashSync == 1 {
			// Flashing! Bright yellow burst
			vector.DrawFilledCircle(a, bx, by, r+4, color.RGBA{255, 255, 100, 180}, false)
			vector.StrokeCircle(a, bx, by, r+8, 1.0, color.RGBA{255, 255, 200, 80}, false)
		} else {
			// Oscillator phase glow: dim→bright as approaching flash
			alpha := uint8(30 + phase*150)
			c := color.RGBA{255, 220, uint8(50 + phase*200), alpha}
			vector.StrokeCircle(a, bx, by, r, 1.0, c, false)
		}
	}
}

// drawVortexOverlay draws rotation arcs indicating vortex formation.
func drawVortexOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.VortexStrength < 10 {
			continue
		}
		bx := float32(bot.X)
		by := float32(bot.Y)

		// Purple arc showing rotation direction
		strength := float32(bot.VortexStrength) / 100.0
		alpha := uint8(60 + strength*150)
		r := float32(swarm.SwarmBotRadius) + 3 + strength*5
		c := color.RGBA{180, 0, 255, alpha}
		vector.StrokeCircle(a, bx, by, r, 1.2, c, false)

		// Tangential arrow
		perpAngle := bot.Angle + math.Pi/2
		ex := bx + float32(math.Cos(perpAngle))*r*0.8
		ey := by + float32(math.Sin(perpAngle))*r*0.8
		vector.StrokeLine(a, bx, by, ex, ey, 0.8,
			color.RGBA{200, 50, 255, alpha}, false)
	}
}

// drawMorphogenOverlay draws activator/inhibitor concentration as colored auras.
func drawMorphogenOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		bx := float32(bot.X)
		by := float32(bot.Y)

		activ := float32(bot.MorphA) / 100.0
		inhib := float32(bot.MorphH) / 100.0
		r := float32(swarm.SwarmBotRadius) + 2 + activ*6
		c := color.RGBA{
			uint8(activ * 255),
			uint8((1 - abs32(activ-inhib)) * 150),
			uint8(inhib * 255),
			uint8(60 + activ*120),
		}
		vector.DrawFilledCircle(a, bx, by, r, c, false)
	}
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// drawEvasionOverlay draws expanding red rings for evasion wave propagation.
func drawEvasionOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.EvasionAlert == 0 {
			continue
		}
		bx := float32(bot.X)
		by := float32(bot.Y)

		// Expanding red ring based on wave progress
		wave := float32(bot.EvasionWave)
		r := float32(swarm.SwarmBotRadius) + wave*0.8
		alpha := uint8(200 - wave*4)
		if alpha < 30 {
			alpha = 30
		}
		vector.StrokeCircle(a, bx, by, r, 2.0,
			color.RGBA{255, 30, 0, alpha}, false)

		// Direction arrow (flee heading)
		if ss.Evasion != nil && i < len(ss.Evasion.FleeAngle) {
			fleeAng := ss.Evasion.FleeAngle[i]
			ex := bx + float32(math.Cos(fleeAng))*r
			ey := by + float32(math.Sin(fleeAng))*r
			vector.StrokeLine(a, bx, by, ex, ey, 1.0,
				color.RGBA{255, 100, 50, alpha}, false)
		}
	}
}

// drawSlimeOverlay draws the slime trail grid as a translucent green heatmap.
func drawSlimeOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Slime
	cellW := float32(st.CellSize)
	for row := 0; row < st.Rows; row++ {
		for col := 0; col < st.Cols; col++ {
			val := st.Grid[row*st.Cols+col]
			if val < 0.01 {
				continue
			}
			alpha := uint8(val * 180)
			green := uint8(100 + val*155)
			vector.DrawFilledRect(a, float32(col)*cellW, float32(row)*cellW,
				cellW, cellW, color.RGBA{0, green, green / 3, alpha}, false)
		}
	}
}

// drawBridgeOverlay draws ant bridge chains as gold connecting lines.
func drawBridgeOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Bridge
	for i := range ss.Bots {
		if !st.InBridge[i] {
			continue
		}
		bot := &ss.Bots[i]
		// Draw gold circle at bridge bot position
		vector.DrawFilledCircle(a, float32(bot.X), float32(bot.Y), 6,
			color.RGBA{255, 200, 0, 150}, false)
		// Connect to nearest bridge neighbor in same chain
		for j := i + 1; j < len(ss.Bots); j++ {
			if !st.InBridge[j] || st.ChainID[j] != st.ChainID[i] {
				continue
			}
			dx := ss.Bots[j].X - bot.X
			dy := ss.Bots[j].Y - bot.Y
			if dx*dx+dy*dy < 900 { // within 30px
				vector.StrokeLine(a, float32(bot.X), float32(bot.Y),
					float32(ss.Bots[j].X), float32(ss.Bots[j].Y),
					3, color.RGBA{255, 200, 0, 120}, false)
			}
		}
	}
}

// drawWaveOverlay draws Mexican wave flash effects.
func drawWaveOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Wave
	for i := range ss.Bots {
		if i >= len(st.FlashTick) {
			continue
		}
		ticksSince := ss.Tick - st.FlashTick[i]
		if ticksSince >= 20 {
			continue
		}
		bot := &ss.Bots[i]
		intensity := float32(1.0 - float64(ticksSince)/20.0)
		radius := 4 + intensity*8
		alpha := uint8(intensity * 200)
		var r, g, b uint8
		switch st.Mode {
		case swarm.WaveLinear:
			r, g, b = alpha, alpha, 0
		case swarm.WaveRadial:
			r, g, b = 0, alpha, alpha
		default:
			r, g, b = alpha, 0, alpha
		}
		vector.DrawFilledCircle(a, float32(bot.X), float32(bot.Y), radius,
			color.RGBA{r, g, b, alpha}, false)
	}
}

// drawShepherdOverlay draws shepherd drive zones and flock center.
func drawShepherdOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Shepherd
	// Draw flock center
	vector.DrawFilledCircle(a, float32(st.FlockCX), float32(st.FlockCY), 8,
		color.RGBA{100, 100, 255, 150}, false)
	// Draw target
	vector.DrawFilledCircle(a, float32(st.TargetX), float32(st.TargetY), 10,
		color.RGBA{255, 50, 50, 150}, false)
	// Line from flock center to target
	vector.StrokeLine(a, float32(st.FlockCX), float32(st.FlockCY),
		float32(st.TargetX), float32(st.TargetY),
		2, color.RGBA{255, 100, 100, 80}, false)
	// Draw shepherd drive radius
	for i := range ss.Bots {
		if i >= len(st.IsShepherd) || !st.IsShepherd[i] {
			continue
		}
		bot := &ss.Bots[i]
		vector.StrokeCircle(a, float32(bot.X), float32(bot.Y), 120,
			1.5, color.RGBA{255, 80, 80, 60}, false)
	}
}

// fitnessLandscapeHashKey computes a simple hash from the fitness function type
// and (for Gaussian) the shared peak parameters.
func fitnessLandscapeHashKey(sa *swarm.SwarmAlgorithmState) uint64 {
	h := uint64(sa.FitnessFunc) * 1000003
	h ^= uint64(len(sa.FitPeakX))
	for i := range sa.FitPeakX {
		h ^= math.Float64bits(sa.FitPeakX[i]) * 31
		h ^= math.Float64bits(sa.FitPeakY[i]) * 37
		h ^= math.Float64bits(sa.FitPeakH[i]) * 41
		h ^= math.Float64bits(sa.FitPeakS[i]) * 43
	}
	return h
}

// landscapeColor maps a normalized fitness value (0-1) to a color using a
// blue → cyan → green → yellow → red gradient.
func landscapeColor(t float64, alpha uint8) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	var r, g, b float64
	switch {
	case t < 0.25:
		s := t / 0.25
		r, g, b = 0, s, 1 // blue → cyan
	case t < 0.5:
		s := (t - 0.25) / 0.25
		r, g, b = 0, 1, 1-s // cyan → green
	case t < 0.75:
		s := (t - 0.5) / 0.25
		r, g, b = s, 1, 0 // green → yellow
	default:
		s := (t - 0.75) / 0.25
		r, g, b = 1, 1-s, 0 // yellow → red
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), alpha}
}

// contourSegment represents a single line segment of a contour line,
// in arena pixel coordinates.
type contourSegment struct {
	x0, y0, x1, y1 float32
	level          int // contour level index (0=lowest, numLevels-1=highest)
}

// buildFitnessLandscape generates a heatmap image from the shared Gaussian
// fitness landscape. Computed at 1/4 resolution for performance and cached
// until peaks change. Also computes iso-fitness contour lines via marching
// squares.
func (r *Renderer) buildFitnessLandscape(sa *swarm.SwarmAlgorithmState, arenaW, arenaH int) *ebiten.Image {
	const step = 4 // sample every 4 pixels
	imgW := arenaW / step
	imgH := arenaH / step

	// Find min/max fitness for normalization
	minF, maxF := math.MaxFloat64, -math.MaxFloat64
	values := make([]float64, imgW*imgH)
	for iy := 0; iy < imgH; iy++ {
		wy := float64(iy*step) + float64(step)/2
		for ix := 0; ix < imgW; ix++ {
			wx := float64(ix*step) + float64(step)/2
			f := swarm.EvaluateFitnessLandscape(sa, wx, wy)
			values[iy*imgW+ix] = f
			if f < minF {
				minF = f
			}
			if f > maxF {
				maxF = f
			}
		}
	}

	// Build RGBA image
	rgba := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	rangeF := maxF - minF
	if rangeF < 1e-9 {
		rangeF = 1
	}
	for iy := 0; iy < imgH; iy++ {
		for ix := 0; ix < imgW; ix++ {
			t := (values[iy*imgW+ix] - minF) / rangeF
			c := landscapeColor(t, 100)
			off := (iy*imgW + ix) * 4
			rgba.Pix[off+0] = c.R
			rgba.Pix[off+1] = c.G
			rgba.Pix[off+2] = c.B
			rgba.Pix[off+3] = c.A
		}
	}

	// Compute contour lines via marching squares
	const numContourLevels = 8
	r.psoContourSegs = r.psoContourSegs[:0]
	r.psoContourW = imgW
	r.psoContourH = imgH
	for li := 1; li <= numContourLevels; li++ {
		frac := float64(li) / float64(numContourLevels+1) // e.g. 1/9, 2/9, ..., 8/9
		level := minF + frac*rangeF
		marchingSquaresContour(&r.psoContourSegs, values, imgW, imgH, level, li-1, float32(step))
	}

	img := ebiten.NewImageFromImage(rgba)
	return img
}

// marchingSquaresContour extracts iso-value contour line segments from a 2D
// scalar field using the marching squares algorithm. Results are appended to
// segs. Coordinates are scaled by pixelStep to convert from grid cells to
// arena pixels.
func marchingSquaresContour(segs *[]contourSegment, values []float64, w, h int, level float64, levelIdx int, pixelStep float32) {
	for iy := 0; iy < h-1; iy++ {
		for ix := 0; ix < w-1; ix++ {
			// Four corners: NW, NE, SE, SW
			nw := values[iy*w+ix]
			ne := values[iy*w+ix+1]
			se := values[(iy+1)*w+ix+1]
			sw := values[(iy+1)*w+ix]

			// 4-bit case index
			ci := 0
			if nw >= level {
				ci |= 1
			}
			if ne >= level {
				ci |= 2
			}
			if se >= level {
				ci |= 4
			}
			if sw >= level {
				ci |= 8
			}
			if ci == 0 || ci == 15 {
				continue
			}

			// Linear interpolation along an edge
			lerpT := func(a, b float64) float32 {
				d := b - a
				if d > -1e-10 && d < 1e-10 {
					return 0.5
				}
				t := (level - a) / d
				if t < 0 {
					t = 0
				} else if t > 1 {
					t = 1
				}
				return float32(t)
			}

			fx := float32(ix) * pixelStep
			fy := float32(iy) * pixelStep

			// Edge crossing points (in arena pixel coords)
			northX := fx + lerpT(nw, ne)*pixelStep
			northY := fy
			eastX := fx + pixelStep
			eastY := fy + lerpT(ne, se)*pixelStep
			southX := fx + lerpT(sw, se)*pixelStep
			southY := fy + pixelStep
			westX := fx
			westY := fy + lerpT(nw, sw)*pixelStep

			addSeg := func(ax, ay, bx, by float32) {
				*segs = append(*segs, contourSegment{ax, ay, bx, by, levelIdx})
			}

			switch ci {
			case 1, 14: // NW
				addSeg(northX, northY, westX, westY)
			case 2, 13: // NE
				addSeg(northX, northY, eastX, eastY)
			case 3, 12: // NW+NE → west-east
				addSeg(westX, westY, eastX, eastY)
			case 4, 11: // SE → east-south
				addSeg(eastX, eastY, southX, southY)
			case 5: // NW+SE (saddle) → two segments
				addSeg(northX, northY, westX, westY)
				addSeg(eastX, eastY, southX, southY)
			case 6, 9: // NE+SE or NW+SW → north-south
				addSeg(northX, northY, southX, southY)
			case 7, 8: // three corners or SW → west-south
				addSeg(westX, westY, southX, southY)
			case 10: // NE+SW (saddle) → two segments
				addSeg(northX, northY, eastX, eastY)
				addSeg(westX, westY, southX, southY)
			}
		}
	}
}

// drawFitnessLandscapeOverlay draws the shared Gaussian fitness landscape as a
// color heatmap with peak markers. Works for any algorithm using the shared
// fitness landscape (not just PSO).
func (r *Renderer) drawFitnessLandscapeOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	sa := ss.SwarmAlgo

	// Rebuild cached heatmap if peaks changed
	h := fitnessLandscapeHashKey(sa)
	if r.psoLandscapeImg == nil || r.psoLandscapeHash != h {
		arenaW := int(ss.ArenaW)
		arenaH := int(ss.ArenaH)
		r.psoLandscapeImg = r.buildFitnessLandscape(sa, arenaW, arenaH)
		r.psoLandscapeHash = h
	}

	// Draw scaled heatmap (4x upscale)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(4, 4)
	a.DrawImage(r.psoLandscapeImg, op)

	// Draw contour lines on top of the heatmap
	if len(r.psoContourSegs) > 0 {
		for _, seg := range r.psoContourSegs {
			// Major contour levels (every other one) are brighter and thicker
			major := seg.level%2 == 1
			var alpha uint8
			var width float32
			if major {
				alpha = 160
				width = 1.5
			} else {
				alpha = 90
				width = 0.8
			}
			col := color.RGBA{255, 255, 255, alpha}
			vector.StrokeLine(a, seg.x0, seg.y0, seg.x1, seg.y1, width, col, false)
		}
	}

	// Draw peak center crosshairs (only for Gaussian peaks)
	if sa.FitnessFunc == swarm.FitGaussian {
		for p := range sa.FitPeakX {
			px, py := float32(sa.FitPeakX[p]), float32(sa.FitPeakY[p])
			arm := float32(8)
			crossCol := color.RGBA{255, 255, 255, 140}
			vector.StrokeLine(a, px-arm, py, px+arm, py, 1.5, crossCol, false)
			vector.StrokeLine(a, px, py-arm, px, py+arm, 1.5, crossCol, false)
		}
	}

	// Draw global best marker if PSO is active (PSO tracks global best position)
	if ss.PSO != nil && ss.PSOOn {
		st := ss.PSO
		vector.DrawFilledCircle(a, float32(st.GlobalX), float32(st.GlobalY), 8,
			color.RGBA{255, 255, 0, 200}, false)
		vector.StrokeCircle(a, float32(st.GlobalX), float32(st.GlobalY), 12,
			2, color.RGBA{255, 255, 0, 120}, false)
	}

	// Legend in top-left corner of arena
	algoName := swarm.SwarmAlgorithmName(sa.ActiveAlgo)
	fitName := swarm.FitnessLandscapeName(sa.FitnessFunc)
	legendY := 10
	printColoredAt(a, fitName+" ("+algoName+")", 10, legendY, color.RGBA{255, 255, 255, 200})
	legendY += 14
	printColoredAt(a, "Low", 10, legendY, color.RGBA{0, 0, 255, 200})
	printColoredAt(a, " -> ", 28, legendY, color.RGBA{200, 200, 200, 180})
	printColoredAt(a, "High", 50, legendY, color.RGBA{255, 0, 0, 200})
	legendY += 14
	printColoredAt(a, "Funktion wechseln: linkes Panel", 10, legendY, color.RGBA{180, 180, 180, 160})
	if sa.DynamicLandscape {
		legendY += 14
		printColoredAt(a, "DYNAMISCH", 10, legendY, color.RGBA{255, 180, 0, 220})
	}
}

// drawMagneticOverlay draws magnetic chain links between bots.
func drawMagneticOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Magnetic
	for i := range ss.Bots {
		if i >= len(st.ChainNext) {
			break
		}
		j := st.ChainNext[i]
		if j < 0 || j >= len(ss.Bots) {
			continue
		}
		// Draw link line
		chainLen := st.ChainLen[i]
		intensity := uint8(100 + int(float64(chainLen)*20))
		if intensity < 100 {
			intensity = 100
		}
		vector.StrokeLine(a, float32(ss.Bots[i].X), float32(ss.Bots[i].Y),
			float32(ss.Bots[j].X), float32(ss.Bots[j].Y),
			2.5, color.RGBA{50, intensity / 2, intensity, 150}, false)
	}
}

// drawDivisionOverlay draws the division split line and group centers.
func drawDivisionOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Division
	cx := float32(ss.ArenaW / 2)
	cy := float32(ss.ArenaH / 2)

	// Draw split axis line
	splitNormal := float64(st.SplitAxis) + math.Pi/2
	lineLen := float32(400)
	dx := lineLen * float32(math.Cos(splitNormal))
	dy := lineLen * float32(math.Sin(splitNormal))
	alpha := uint8(60 + int(st.Phase*120))
	vector.StrokeLine(a, cx-dx, cy-dy, cx+dx, cy+dy,
		2, color.RGBA{200, 200, 200, alpha}, false)

	// Group A center (magenta)
	vector.DrawFilledCircle(a, float32(st.CenterAX), float32(st.CenterAY), 8,
		color.RGBA{200, 50, 150, 150}, false)
	// Group B center (cyan)
	vector.DrawFilledCircle(a, float32(st.CenterBX), float32(st.CenterBY), 8,
		color.RGBA{50, 150, 200, 150}, false)
}

// drawVFormationOverlay draws leader marker, wing lines, and draft zones.
func drawVFormationOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.VFormation
	if st.LeaderIdx >= len(ss.Bots) {
		return
	}
	leader := &ss.Bots[st.LeaderIdx]

	// Gold circle around leader
	vector.StrokeCircle(a, float32(leader.X), float32(leader.Y), 12,
		2, color.RGBA{255, 255, 100, 180}, false)

	// Draw wing lines from leader to wing bots
	for i := range ss.Bots {
		if i == st.LeaderIdx || !st.InFormation[i] {
			continue
		}
		bot := &ss.Bots[i]
		var c color.RGBA
		if st.FormPos[i] > 0 {
			c = color.RGBA{200, 150, 50, 60} // orange right wing
		} else {
			c = color.RGBA{50, 150, 200, 60} // blue left wing
		}
		vector.StrokeLine(a, float32(leader.X), float32(leader.Y),
			float32(bot.X), float32(bot.Y), 1, c, false)
	}

	// Migration direction arrow from leader
	arrLen := float32(30)
	ax := float32(leader.X) + arrLen*float32(math.Cos(st.MigAngle))
	ay := float32(leader.Y) + arrLen*float32(math.Sin(st.MigAngle))
	vector.StrokeLine(a, float32(leader.X), float32(leader.Y), ax, ay,
		2, color.RGBA{255, 255, 100, 150}, false)
}

// drawBroodOverlay draws brood sorting items on the arena.
func drawBroodOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Brood
	for i := range st.Items {
		item := &st.Items[i]
		if item.Held {
			continue // held items follow carrier, skip overlay
		}
		var c color.RGBA
		switch item.Color {
		case 0:
			c = color.RGBA{255, 80, 80, 150} // red
		case 1:
			c = color.RGBA{80, 255, 80, 150} // green
		case 2:
			c = color.RGBA{80, 80, 255, 150} // blue
		}
		vector.DrawFilledCircle(a, float32(item.X), float32(item.Y), 4, c, false)
	}
}

// drawJellyfishOverlay draws the pulsing center and expansion/contraction rings.
func drawJellyfishOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Jellyfish
	cx, cy := float32(st.CenterX), float32(st.CenterY)

	// Draw center marker
	vector.DrawFilledCircle(a, cx, cy, 5, color.RGBA{0, 220, 220, 120}, false)

	// Draw pulse rings at min and max radius
	vector.StrokeCircle(a, cx, cy, 40, 1, color.RGBA{0, 200, 200, 40}, false)  // min
	vector.StrokeCircle(a, cx, cy, 200, 1, color.RGBA{0, 200, 200, 40}, false) // max

	// Draw current average radius ring
	var totalDist float64
	for i := range ss.Bots {
		dx := ss.Bots[i].X - st.CenterX
		dy := ss.Bots[i].Y - st.CenterY
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}
	if len(ss.Bots) > 0 {
		avgR := float32(totalDist / float64(len(ss.Bots)))
		vector.StrokeCircle(a, cx, cy, avgR, 2, color.RGBA{0, 220, 220, 80}, false)
	}
}

// drawImmuneOverlay draws antibody/pathogen markers and alert connections.
func drawImmuneOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.ImmuneSwarm
	for i := range ss.Bots {
		if i >= len(st.IsAntibody) {
			break
		}
		bot := &ss.Bots[i]
		if st.IsPathogen[i] {
			// Purple X for pathogens
			x, y := float32(bot.X), float32(bot.Y)
			vector.StrokeLine(a, x-6, y-6, x+6, y+6, 2, color.RGBA{180, 0, 180, 180}, false)
			vector.StrokeLine(a, x+6, y-6, x-6, y+6, 2, color.RGBA{180, 0, 180, 180}, false)
			// Neutralization progress ring
			if st.NeutralizeTimer[i] > 0 {
				progress := float32(st.NeutralizeTimer[i]) / 30.0
				vector.StrokeCircle(a, x, y, 10+progress*5, 2,
					color.RGBA{255, 100, 100, uint8(100 + progress*155)}, false)
			}
		} else if st.IsAntibody[i] && st.AlertLevel[i] > 0.1 {
			// Draw alert line from antibody to signal source
			vector.StrokeLine(a, float32(bot.X), float32(bot.Y),
				float32(st.SignalX[i]), float32(st.SignalY[i]),
				1, color.RGBA{255, 80, 80, uint8(st.AlertLevel[i] * 80)}, false)
		}
	}
}

// drawGravityOverlay draws gravitational force vectors and heavy body markers.
func drawGravityOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Gravity
	for i := range ss.Bots {
		if i >= len(st.Mass) {
			break
		}
		bot := &ss.Bots[i]
		// Heavy bodies: yellow ring
		if st.Mass[i] >= 3.0*0.8 {
			vector.StrokeCircle(a, float32(bot.X), float32(bot.Y), 10,
				2, color.RGBA{255, 255, 100, 150}, false)
		}
		// Force vector line
		fx, fy := st.ForceX[i], st.ForceY[i]
		mag := math.Sqrt(fx*fx + fy*fy)
		if mag > 0.02 {
			scale := float32(math.Min(30, mag*100))
			ex := float32(bot.X) + scale*float32(fx/mag)
			ey := float32(bot.Y) + scale*float32(fy/mag)
			vector.StrokeLine(a, float32(bot.X), float32(bot.Y), ex, ey,
				1, color.RGBA{200, 200, 100, 60}, false)
		}
	}
}

// drawCrystalOverlay draws lattice bonds between neighboring bots.
func drawCrystalOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Crystal
	// Draw bonds between settled neighbors
	for i := range ss.Bots {
		if i >= len(st.Settled) || !st.Settled[i] {
			continue
		}
		if ss.Hash == nil {
			continue
		}
		nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, 35)
		for _, j := range nearIDs {
			if j <= i || j >= len(ss.Bots) || j >= len(st.Settled) || !st.Settled[j] {
				continue
			}
			vector.StrokeLine(a,
				float32(ss.Bots[i].X), float32(ss.Bots[i].Y),
				float32(ss.Bots[j].X), float32(ss.Bots[j].Y),
				1, color.RGBA{50, 220, 80, 40}, false)
		}
	}
	// Defect markers
	for i := range ss.Bots {
		if i >= len(st.Defect) || !st.Defect[i] {
			continue
		}
		vector.StrokeCircle(a, float32(ss.Bots[i].X), float32(ss.Bots[i].Y), 6,
			1, color.RGBA{220, 60, 60, 100}, false)
	}
}

// drawAmoebaOverlay draws the amoeba blob outline, center, and direction.
func drawAmoebaOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Amoeba
	cx, cy := float32(st.CenterX), float32(st.CenterY)

	// Center marker
	vector.DrawFilledCircle(a, cx, cy, 5, color.RGBA{100, 255, 80, 120}, false)

	// Direction arrow
	arrLen := float32(40)
	ax := cx + arrLen*float32(math.Cos(st.Direction))
	ay := cy + arrLen*float32(math.Sin(st.Direction))
	vector.StrokeLine(a, cx, cy, ax, ay, 2, color.RGBA{150, 255, 100, 150}, false)

	// Pseudopod zone outline (arc)
	for _, bot := range ss.Bots {
		if bot.AmoebaPseudo == 1 {
			vector.DrawFilledCircle(a, float32(bot.X), float32(bot.Y), 3,
				color.RGBA{150, 255, 100, 60}, false)
		}
	}
}

// drawACOOverlay draws the pheromone trail heatmap.
func drawACOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.ACO
	cellSize := float32(10)
	for row := 0; row < st.GridRows; row++ {
		for col := 0; col < st.GridCols; col++ {
			val := st.Trail[row*st.GridCols+col]
			if val < 1 {
				continue
			}
			intensity := uint8(math.Min(255, val*3))
			x := float32(col) * cellSize
			y := float32(row) * cellSize
			vector.DrawFilledRect(a, x, y, cellSize, cellSize,
				color.RGBA{intensity, uint8(float64(intensity) * 0.5), 0, uint8(math.Min(80, val*2))}, false)
		}
	}
}

// drawGWOOverlay visualizes the Grey Wolf Optimizer pack hierarchy.
// Alpha (gold), beta (silver), delta (bronze) are drawn as larger circles.
// Omega wolves get thin lines connecting them to the alpha leader.
func drawGWOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.GWO
	if st == nil {
		return
	}

	// Draw lines from omega wolves to alpha
	if st.AlphaIdx >= 0 {
		ax := float32(ss.Bots[st.AlphaIdx].X)
		ay := float32(ss.Bots[st.AlphaIdx].Y)
		for i := range ss.Bots {
			if st.Rank[i] == 3 { // omega
				bx := float32(ss.Bots[i].X)
				by := float32(ss.Bots[i].Y)
				dx := ax - bx
				dy := ay - by
				if dx*dx+dy*dy < 200*200 { // only draw if within 200px
					vector.StrokeLine(a, bx, by, ax, ay, 0.5,
						color.RGBA{255, 215, 0, 30}, false)
				}
			}
		}
	}

	// Draw hierarchy rings around alpha/beta/delta
	leaders := []struct {
		idx int
		r, g, b uint8
		radius float32
	}{
		{st.AlphaIdx, 255, 215, 0, 12},  // gold
		{st.BetaIdx, 192, 192, 192, 10},  // silver
		{st.DeltaIdx, 205, 127, 50, 8},   // bronze
	}
	for _, l := range leaders {
		if l.idx >= 0 && l.idx < len(ss.Bots) {
			x := float32(ss.Bots[l.idx].X)
			y := float32(ss.Bots[l.idx].Y)
			vector.StrokeCircle(a, x, y, l.radius, 2,
				color.RGBA{l.r, l.g, l.b, 180}, false)
		}
	}
}

// drawWOAOverlay visualizes the Whale Optimization Algorithm.
// Shows the best whale with a large ring and phase-colored indicators per bot.
func drawWOAOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.WOA
	if st == nil {
		return
	}

	// Draw ring around best whale (prey position)
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		bx := float32(st.BestX)
		by := float32(st.BestY)
		vector.StrokeCircle(a, bx, by, 20, 2,
			color.RGBA{0, 150, 255, 200}, false)
		vector.StrokeCircle(a, bx, by, 30, 1,
			color.RGBA{0, 100, 255, 100}, false)
	}

	// Draw phase indicators: encircle=blue dot, spiral=cyan arc, search=light blue
	for i := range ss.Bots {
		if i == st.BestIdx {
			continue
		}
		x := float32(ss.Bots[i].X)
		y := float32(ss.Bots[i].Y)
		phase := st.Phase[i]
		switch phase {
		case 0: // encircle
			vector.DrawFilledCircle(a, x, y-8, 2,
				color.RGBA{0, 60, 180, 120}, false)
		case 1: // spiral
			vector.StrokeCircle(a, x, y, 6, 1,
				color.RGBA{0, 200, 255, 100}, false)
		case 2: // search
			vector.DrawFilledCircle(a, x, y-8, 2,
				color.RGBA{100, 150, 255, 100}, false)
		}
	}
}

// drawBFOOverlay visualizes Bacterial Foraging Optimization.
// Shows health as colored rings and swim direction indicators.
func drawBFOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.BFO
	if st == nil {
		return
	}

	for i := range ss.Bots {
		if i >= len(st.Health) {
			break
		}
		x := float32(ss.Bots[i].X)
		y := float32(ss.Bots[i].Y)

		// Health ring: green (healthy) to red (unhealthy)
		health := st.Health[i]
		if health > 10 {
			health = 10
		}
		g := uint8(math.Min(255, health*25))
		r := uint8(math.Min(255, (10-health)*25))
		vector.StrokeCircle(a, x, y, 7, 1.5,
			color.RGBA{r, g, 50, 100}, false)

		// Swim direction indicator (small line in swim direction)
		if i < len(st.SwimDir) {
			dir := st.SwimDir[i]
			ex := x + float32(math.Cos(dir))*10
			ey := y + float32(math.Sin(dir))*10
			vector.StrokeLine(a, x, y, ex, ey, 1,
				color.RGBA{r, g, 50, 80}, false)
		}
	}
}

// drawMFOOverlay visualizes Moth-Flame Optimization.
// Draws flames as orange circles and lines from moths to their assigned flames.
func drawMFOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.MFO
	if st == nil {
		return
	}

	// Draw flames
	for _, flame := range st.Flames {
		fx := float32(flame.X)
		fy := float32(flame.Y)
		intensity := uint8(math.Min(255, flame.Intensity*255))
		radius := float32(4 + flame.Fitness*6)
		vector.DrawFilledCircle(a, fx, fy, radius,
			color.RGBA{255, intensity, 0, uint8(math.Min(160, flame.Intensity*200))}, false)
		vector.StrokeCircle(a, fx, fy, radius+3, 1,
			color.RGBA{255, 200, 50, uint8(math.Min(80, flame.Intensity*100))}, false)
	}

	// Draw lines from moths to their flames
	for i := range ss.Bots {
		if i >= len(st.MothFlame) {
			break
		}
		fi := st.MothFlame[i]
		if fi < 0 || fi >= len(st.Flames) {
			continue
		}
		mx := float32(ss.Bots[i].X)
		my := float32(ss.Bots[i].Y)
		fx := float32(st.Flames[fi].X)
		fy := float32(st.Flames[fi].Y)
		dx := fx - mx
		dy := fy - my
		if dx*dx+dy*dy < 150*150 { // only draw if close enough
			vector.StrokeLine(a, mx, my, fx, fy, 0.5,
				color.RGBA{255, 180, 30, 40}, false)
		}
	}
}

// drawCuckooOverlay visualizes the Cuckoo Search algorithm.
// Draws the global best nest as a gold star and lines from nests to it.
// Top 25% nests are highlighted with green rings.
func drawCuckooOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Cuckoo
	if st == nil {
		return
	}

	// Draw global best nest as a gold circle
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		bx := float32(ss.Bots[st.BestIdx].X)
		by := float32(ss.Bots[st.BestIdx].Y)
		vector.DrawFilledCircle(a, bx, by, 8, color.RGBA{255, 215, 0, 180}, false)
		vector.StrokeCircle(a, bx, by, 12, 1.5, color.RGBA{255, 255, 100, 120}, false)
	}

	// Draw lines from each nest to global best, and highlight top nests
	for i := range ss.Bots {
		if i == st.BestIdx {
			continue
		}
		nx := float32(ss.Bots[i].X)
		ny := float32(ss.Bots[i].Y)

		// Top 25% nests get a green ring
		if ss.Bots[i].CuckooBest == 1 {
			vector.StrokeCircle(a, nx, ny, 7, 1,
				color.RGBA{0, 200, 80, 100}, false)
		}

		// Line to global best (faint, only if within range)
		if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
			bx := float32(ss.Bots[st.BestIdx].X)
			by := float32(ss.Bots[st.BestIdx].Y)
			dx := bx - nx
			dy := by - ny
			if dx*dx+dy*dy < 200*200 {
				vector.StrokeLine(a, nx, ny, bx, by, 0.5,
					color.RGBA{180, 200, 50, 35}, false)
			}
		}
	}
}

// drawDEOverlay visualizes the Differential Evolution algorithm.
// Shows the best individual as a gold circle and draws lines from each bot
// to its trial position (the mutant vector target).
func drawDEOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.DE
	if st == nil {
		return
	}

	// Draw best individual as a gold circle.
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		bx := float32(ss.Bots[st.BestIdx].X)
		by := float32(ss.Bots[st.BestIdx].Y)
		vector.DrawFilledCircle(a, bx, by, 8, color.RGBA{255, 200, 0, 180}, false)
		vector.StrokeCircle(a, bx, by, 12, 1.5, color.RGBA{255, 255, 80, 120}, false)
	}

	// Draw trial position markers and lines for bots that are moving.
	for i := range ss.Bots {
		if i >= len(st.Moving) || !st.Moving[i] {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		tx := float32(st.TrialX[i])
		ty := float32(st.TrialY[i])

		// Small cross at trial position.
		vector.StrokeLine(a, tx-3, ty, tx+3, ty, 1, color.RGBA{100, 255, 100, 100}, false)
		vector.StrokeLine(a, tx, ty-3, tx, ty+3, 1, color.RGBA{100, 255, 100, 100}, false)

		// Line from bot to trial.
		vector.StrokeLine(a, bx, by, tx, ty, 0.5, color.RGBA{80, 200, 80, 40}, false)
	}
}

// drawABCOverlay visualises the Artificial Bee Colony algorithm.
// Draws the best food source as a gold circle and shows role-dependent
// indicators: employed bees get small rings, scouts get white diamonds.
func drawABCOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.ABC
	if st == nil {
		return
	}

	// Draw best food source as a gold circle.
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		bx := float32(ss.Bots[st.BestIdx].X)
		by := float32(ss.Bots[st.BestIdx].Y)
		vector.DrawFilledCircle(a, bx, by, 8, color.RGBA{255, 200, 0, 180}, false)
		vector.StrokeCircle(a, bx, by, 12, 1.5, color.RGBA{255, 220, 60, 120}, false)
	}

	// Draw role indicators and trial position lines.
	for i := range ss.Bots {
		if i >= len(st.Role) {
			break
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		tx := float32(st.TrialX[i])
		ty := float32(st.TrialY[i])

		switch st.Role[i] {
		case 0: // Employed — small yellow ring
			vector.StrokeCircle(a, bx, by, 7, 1, color.RGBA{255, 200, 30, 80}, false)
			vector.StrokeLine(a, bx, by, tx, ty, 0.5, color.RGBA{255, 200, 30, 30}, false)
		case 1: // Onlooker — small orange ring
			vector.StrokeCircle(a, bx, by, 7, 1, color.RGBA{255, 140, 0, 80}, false)
		case 2: // Scout — white diamond marker
			vector.StrokeLine(a, bx, by-6, bx+4, by, 1, color.RGBA{255, 255, 255, 120}, false)
			vector.StrokeLine(a, bx+4, by, bx, by+6, 1, color.RGBA{255, 255, 255, 120}, false)
			vector.StrokeLine(a, bx, by+6, bx-4, by, 1, color.RGBA{255, 255, 255, 120}, false)
			vector.StrokeLine(a, bx-4, by, bx, by-6, 1, color.RGBA{255, 255, 255, 120}, false)
		}
	}
}

// drawHSOOverlay visualises the Harmony Search Optimization algorithm.
// Draws Harmony Memory positions as cyan dots, the best harmony as a bright
// cyan circle, and lines from each bot to its current target position.
func drawHSOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.HSO
	if st == nil {
		return
	}

	// Draw Harmony Memory entries as small cyan dots.
	for _, h := range st.HM {
		hx := float32(h.X)
		hy := float32(h.Y)
		vector.DrawFilledCircle(a, hx, hy, 3, color.RGBA{0, 200, 220, 100}, false)
	}

	// Draw best harmony as a bright cyan circle.
	if st.BestIdx >= 0 && st.BestIdx < len(st.HM) {
		bx := float32(st.BestX)
		by := float32(st.BestY)
		vector.DrawFilledCircle(a, bx, by, 8, color.RGBA{0, 220, 255, 180}, false)
		vector.StrokeCircle(a, bx, by, 12, 1.5, color.RGBA{0, 200, 240, 120}, false)
	}

	// Draw lines from bots to their target positions.
	for i := range ss.Bots {
		if i >= len(st.TargetX) {
			break
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		tx := float32(st.TargetX[i])
		ty := float32(st.TargetY[i])

		if st.Phase[i] == 0 {
			// Improvising — line to target.
			vector.StrokeLine(a, bx, by, tx, ty, 0.5, color.RGBA{0, 180, 200, 40}, false)
		} else {
			// Arrived — small ring.
			vector.StrokeCircle(a, bx, by, 7, 1, color.RGBA{0, 255, 220, 80}, false)
		}
	}
}

// drawBatOverlay renders echolocation pulse rings proportional to loudness
// and highlights the best bat with a bright cyan marker.
func drawBatOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Bat
	if st == nil {
		return
	}

	// Draw echolocation pulse rings for each bat — radius scales with loudness
	for i := range ss.Bots {
		if i >= len(st.Loud) {
			break
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)

		// Pulse ring: louder bats emit larger sonar rings
		radius := float32(st.Loud[i] * 30)
		if radius < 3 {
			radius = 3
		}
		alpha := uint8(st.Loud[i] * 120)
		if alpha < 10 {
			alpha = 10
		}
		vector.StrokeCircle(a, bx, by, radius, 0.8, color.RGBA{140, 60, 220, alpha}, false)

		// Inner pulse dot — brighter for higher pulse rate
		pulseAlpha := uint8(40)
		if i < len(st.Pulse) {
			pulseAlpha = uint8(st.Pulse[i] * 180)
		}
		if pulseAlpha < 20 {
			pulseAlpha = 20
		}
		vector.DrawFilledCircle(a, bx, by, 2, color.RGBA{180, 80, 255, pulseAlpha}, false)
	}

	// Highlight global best bat
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		bx := float32(st.BestX)
		by := float32(st.BestY)
		vector.DrawFilledCircle(a, bx, by, 8, color.RGBA{0, 255, 255, 180}, false)
		vector.StrokeCircle(a, bx, by, 14, 1.5, color.RGBA{0, 200, 255, 100}, false)
	}
}

// drawHHOOverlay visualizes the Harris Hawks Optimization algorithm.
// Shows the rabbit (prey) with a gold marker, and phase-colored indicators
// for each hawk: blue=explore, orange=soft besiege, red=hard besiege, purple=dive.
func drawHHOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.HHO
	if st == nil {
		return
	}

	// Phase colors for each hawk
	phaseColors := [4]color.RGBA{
		{80, 130, 200, 100},  // explore — blue
		{255, 165, 0, 100},   // soft besiege — orange
		{255, 50, 50, 100},   // hard besiege — red
		{200, 50, 200, 100},  // rapid dive — purple
	}

	// Draw connection lines from hawks to rabbit
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		rx := float32(st.BestX)
		ry := float32(st.BestY)
		for i := range ss.Bots {
			if i == st.BestIdx || i >= len(st.Phase) {
				continue
			}
			bx := float32(ss.Bots[i].X)
			by := float32(ss.Bots[i].Y)
			dx := rx - bx
			dy := ry - by
			if dx*dx+dy*dy < 250*250 {
				p := st.Phase[i]
				if p < 0 || p > 3 {
					p = 0
				}
				c := phaseColors[p]
				c.A = 40
				vector.StrokeLine(a, bx, by, rx, ry, 0.5, c, false)
			}
		}
	}

	// Draw phase indicators for each hawk
	for i := range ss.Bots {
		if i == st.BestIdx || i >= len(st.Phase) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		p := st.Phase[i]
		if p < 0 || p > 3 {
			p = 0
		}
		c := phaseColors[p]
		vector.StrokeCircle(a, bx, by, 8, 1.2, c, false)
	}

	// Highlight rabbit (prey) with gold marker and crosshair
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		rx := float32(st.BestX)
		ry := float32(st.BestY)
		vector.DrawFilledCircle(a, rx, ry, 6, color.RGBA{255, 215, 0, 200}, false)
		vector.StrokeCircle(a, rx, ry, 14, 1.5, color.RGBA{255, 215, 0, 100}, false)
		// Crosshair lines
		vector.StrokeLine(a, rx-18, ry, rx-8, ry, 1, color.RGBA{255, 215, 0, 120}, false)
		vector.StrokeLine(a, rx+8, ry, rx+18, ry, 1, color.RGBA{255, 215, 0, 120}, false)
		vector.StrokeLine(a, rx, ry-18, rx, ry-8, 1, color.RGBA{255, 215, 0, 120}, false)
		vector.StrokeLine(a, rx, ry+8, rx, ry+18, 1, color.RGBA{255, 215, 0, 120}, false)
	}
}

// drawSSAOverlay visualizes the Salp Swarm Algorithm.
// Shows chain links between leader→follower salps and food source marker.
func drawSSAOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.SSA
	if st == nil {
		return
	}

	// Draw chain links from each follower to its predecessor
	for i := range ss.Bots {
		if i >= len(st.ChainIdx) {
			continue
		}
		pred := st.ChainIdx[i]
		if pred < 0 || pred >= len(ss.Bots) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		px := float32(ss.Bots[pred].X)
		py := float32(ss.Bots[pred].Y)
		dx := px - bx
		dy := py - by
		if dx*dx+dy*dy < 200*200 {
			c := color.RGBA{0, 180, 220, 40}
			if i < len(st.Role) && st.Role[i] == 0 {
				c = color.RGBA{0, 220, 255, 60} // leaders brighter
			}
			vector.StrokeLine(a, bx, by, px, py, 0.5, c, false)
		}
	}

	// Role indicators: cyan ring for leaders, blue dot for followers
	for i := range ss.Bots {
		if i >= len(st.Role) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		if st.Role[i] == 0 { // leader
			vector.StrokeCircle(a, bx, by, 8, 1.2, color.RGBA{0, 220, 255, 120}, false)
		} else { // follower
			vector.DrawFilledCircle(a, bx, by-8, 2, color.RGBA{50, 100, 200, 100}, false)
		}
	}

	// Food source marker (best fitness position)
	fx := float32(st.FoodX)
	fy := float32(st.FoodY)
	vector.DrawFilledCircle(a, fx, fy, 6, color.RGBA{255, 215, 0, 200}, false)
	vector.StrokeCircle(a, fx, fy, 14, 1.5, color.RGBA{255, 215, 0, 100}, false)
}

// drawGSAOverlay visualizes the Gravitational Search Algorithm.
// Shows mass-proportional rings and gravitational force vectors.
func drawGSAOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.GSA
	if st == nil {
		return
	}

	// Draw mass-proportional rings around each agent
	for i := range ss.Bots {
		if i >= len(st.Mass) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		m := st.Mass[i]
		radius := float32(4 + m*10) // 4-14px based on mass
		alpha := uint8(40 + m*160)
		vector.StrokeCircle(a, bx, by, radius, 1, color.RGBA{180, 100, 255, alpha}, false)
	}

	// Draw acceleration vectors
	for i := range ss.Bots {
		if i >= len(st.AccX) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		ax := float32(st.AccX[i]) * 8
		ay := float32(st.AccY[i]) * 8
		if ax*ax+ay*ay > 1 {
			vector.StrokeLine(a, bx, by, bx+ax, by+ay, 1,
				color.RGBA{180, 100, 255, 80}, false)
		}
	}

	// Highlight heaviest agent (best)
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		hx := float32(st.BestX)
		hy := float32(st.BestY)
		vector.DrawFilledCircle(a, hx, hy, 5, color.RGBA{255, 200, 50, 200}, false)
		vector.StrokeCircle(a, hx, hy, 18, 2, color.RGBA{255, 200, 50, 120}, false)
	}
}

// drawFPAOverlay visualizes the Flower Pollination Algorithm.
// Shows global/local pollination type per flower and global best marker.
func drawFPAOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.FPA
	if st == nil {
		return
	}

	// Draw pollination type indicators
	for i := range ss.Bots {
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)

		if i < len(st.IsGlobal) && st.IsGlobal[i] {
			// Global pollination: small butterfly symbol (magenta ring)
			vector.StrokeCircle(a, bx, by, 7, 1.2, color.RGBA{255, 100, 200, 100}, false)
		} else {
			// Local pollination: small green dot
			vector.DrawFilledCircle(a, bx, by-7, 2, color.RGBA{100, 220, 80, 100}, false)
		}

		// Draw line to personal best (if close enough)
		if i < len(st.BestX) {
			px := float32(st.BestX[i])
			py := float32(st.BestY[i])
			dx := px - bx
			dy := py - by
			if dx*dx+dy*dy > 4 && dx*dx+dy*dy < 150*150 {
				vector.StrokeLine(a, bx, by, px, py, 0.5,
					color.RGBA{100, 220, 80, 30}, false)
			}
		}
	}

	// Global best flower marker
	gx := float32(st.GlobalBestX)
	gy := float32(st.GlobalBestY)
	vector.DrawFilledCircle(a, gx, gy, 6, color.RGBA{255, 100, 200, 200}, false)
	vector.StrokeCircle(a, gx, gy, 14, 1.5, color.RGBA{255, 100, 200, 100}, false)
	vector.StrokeCircle(a, gx, gy, 22, 1, color.RGBA{255, 100, 200, 50}, false)
}

// drawSAOverlay visualizes Simulated Annealing.
// Shows temperature as color halos and lines to perturbation targets.
func drawSAOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.SA
	if st == nil {
		return
	}

	// Draw temperature halos and target lines
	for i := range ss.Bots {
		if i >= len(st.Temp) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)

		// Temperature halo: hot=red, cold=blue
		tRatio := st.Temp[i] / st.InitialTemp
		if tRatio > 1 {
			tRatio = 1
		}
		r := uint8(255 * tRatio)
		b := uint8(255 * (1 - tRatio))
		radius := float32(5 + tRatio*10) // larger halo when hot
		vector.StrokeCircle(a, bx, by, radius, 1, color.RGBA{r, 40, b, 80}, false)

		// Draw line to perturbation target if moving
		if i < len(st.Moving) && st.Moving[i] && i < len(st.TargetX) {
			tx := float32(st.TargetX[i])
			ty := float32(st.TargetY[i])
			lineCol := color.RGBA{255, 165, 0, 50} // orange
			if i < len(st.Accepted) && !st.Accepted[i] {
				lineCol = color.RGBA{255, 50, 50, 50} // red = rejected
			}
			vector.StrokeLine(a, bx, by, tx, ty, 0.5, lineCol, false)
		}
	}

	// Highlight global best with gold marker
	if st.GlobalBestIdx >= 0 && st.GlobalBestIdx < len(ss.Bots) {
		gx := float32(ss.Bots[st.GlobalBestIdx].X)
		gy := float32(ss.Bots[st.GlobalBestIdx].Y)
		vector.DrawFilledCircle(a, gx, gy, 5, color.RGBA{255, 215, 0, 200}, false)
		vector.StrokeCircle(a, gx, gy, 16, 2, color.RGBA{255, 215, 0, 120}, false)
	}
}

// drawAOOverlay visualizes the Aquila Optimizer.
// Shows four hunting phases as colored rings and prey marker.
func drawAOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.AO
	if st == nil {
		return
	}

	// Phase colors: 0=sky blue (high soar), 1=green (contour), 2=orange (low flight), 3=red (grab)
	phaseColors := [4]color.RGBA{
		{100, 180, 255, 100}, // high soar — sky blue
		{80, 220, 120, 100},  // contour flight — green
		{255, 165, 50, 100},  // low flight — orange
		{255, 60, 60, 100},   // walk & grab — red
	}

	for i := range ss.Bots {
		if i >= len(st.Phase) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)
		p := st.Phase[i]
		if p < 0 || p > 3 {
			p = 0
		}
		c := phaseColors[p]
		radius := float32(6 + (3-p)*2) // larger ring = exploration phases
		vector.StrokeCircle(a, bx, by, radius, 1.2, c, false)

		// Draw line to prey (best) for grab phase
		if p == 3 && st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
			px := float32(st.BestX)
			py := float32(st.BestY)
			vector.StrokeLine(a, bx, by, px, py, 0.5, color.RGBA{255, 60, 60, 40}, false)
		}
	}

	// Prey marker (global best)
	if st.BestIdx >= 0 {
		px := float32(st.BestX)
		py := float32(st.BestY)
		vector.DrawFilledCircle(a, px, py, 5, color.RGBA{255, 215, 0, 200}, false)
		vector.StrokeCircle(a, px, py, 14, 1.5, color.RGBA{255, 215, 0, 100}, false)
		// Crosshair
		vector.StrokeLine(a, px-10, py, px+10, py, 0.8, color.RGBA{255, 215, 0, 80}, false)
		vector.StrokeLine(a, px, py-10, px, py+10, 0.8, color.RGBA{255, 215, 0, 80}, false)
	}
}

// drawSCAOverlay visualizes the Sine Cosine Algorithm.
// Shows sine/cosine phase per bot and oscillation arcs toward global best.
func drawSCAOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.SCA
	if st == nil {
		return
	}

	for i := range ss.Bots {
		if i >= len(st.Phase) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)

		if st.Phase[i] == 0 {
			// Sine phase (exploration): cyan ring
			vector.StrokeCircle(a, bx, by, 7, 1.2, color.RGBA{0, 220, 255, 100}, false)
		} else {
			// Cosine phase (exploitation): magenta dot
			vector.DrawFilledCircle(a, bx, by-7, 2.5, color.RGBA{255, 80, 220, 120}, false)
		}

		// Draw line to global best (short range only)
		dx := float32(st.BestX) - bx
		dy := float32(st.BestY) - by
		if dx*dx+dy*dy > 4 && dx*dx+dy*dy < 180*180 {
			c := color.RGBA{0, 220, 255, 25}
			if st.Phase[i] == 1 {
				c = color.RGBA{255, 80, 220, 25}
			}
			vector.StrokeLine(a, bx, by, float32(st.BestX), float32(st.BestY), 0.5, c, false)
		}
	}

	// Global best marker
	if st.BestIdx >= 0 {
		gx := float32(st.BestX)
		gy := float32(st.BestY)
		vector.DrawFilledCircle(a, gx, gy, 6, color.RGBA{255, 200, 50, 200}, false)
		vector.StrokeCircle(a, gx, gy, 14, 1.5, color.RGBA{255, 200, 50, 100}, false)
	}
}

// drawDAOverlay visualizes the Dragonfly Algorithm.
// Shows role indicators (static/dynamic/levy), step vectors, and food/enemy markers.
func drawDAOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.DA
	if st == nil {
		return
	}

	// Role colors: 0=green (static/feeding), 1=blue (dynamic/migratory), 2=magenta (levy)
	roleColors := [3]color.RGBA{
		{80, 220, 100, 100},  // static — green
		{80, 140, 255, 100},  // dynamic — blue
		{220, 80, 255, 100},  // levy flight — magenta
	}

	for i := range ss.Bots {
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)

		// Role indicator
		r := 0
		if i < len(st.Role) {
			r = st.Role[i]
		}
		if r < 0 || r > 2 {
			r = 0
		}
		c := roleColors[r]
		vector.StrokeCircle(a, bx, by, 7, 1, c, false)

		// Draw step vector
		if i < len(st.StepX) && i < len(st.StepY) {
			sx := float32(st.StepX[i]) * 5
			sy := float32(st.StepY[i]) * 5
			if sx*sx+sy*sy > 2 {
				vector.StrokeLine(a, bx, by, bx+sx, by+sy, 0.8, color.RGBA{c.R, c.G, c.B, 60}, false)
			}
		}
	}

	// Food source (best) — gold
	if st.BestIdx >= 0 {
		fx := float32(st.BestX)
		fy := float32(st.BestY)
		vector.DrawFilledCircle(a, fx, fy, 6, color.RGBA{255, 215, 0, 200}, false)
		vector.StrokeCircle(a, fx, fy, 14, 1.5, color.RGBA{255, 215, 0, 100}, false)
	}

	// Enemy (worst) — red X
	if st.WorstIdx >= 0 {
		ex := float32(st.WorstX)
		ey := float32(st.WorstY)
		vector.StrokeLine(a, ex-6, ey-6, ex+6, ey+6, 1.5, color.RGBA{255, 60, 60, 160}, false)
		vector.StrokeLine(a, ex-6, ey+6, ex+6, ey-6, 1.5, color.RGBA{255, 60, 60, 160}, false)
	}
}

// drawTLBOOverlay visualizes Teaching-Learning-Based Optimization.
// Shows teacher/learner phases, peer links, and class mean position.
func drawTLBOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.TLBO
	if st == nil {
		return
	}

	for i := range ss.Bots {
		if i >= len(st.Phase) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)

		if st.Phase[i] == 0 {
			// Teacher phase: green ring (learning from best)
			vector.StrokeCircle(a, bx, by, 7, 1, color.RGBA{80, 220, 100, 90}, false)
		} else {
			// Learner phase: blue ring (peer interaction)
			vector.StrokeCircle(a, bx, by, 7, 1, color.RGBA{80, 140, 255, 90}, false)
		}

		// Draw line to peer in learner phase
		if st.Phase[i] == 1 && i < len(st.PeerIdx) {
			pi := st.PeerIdx[i]
			if pi >= 0 && pi < len(ss.Bots) {
				px := float32(ss.Bots[pi].X)
				py := float32(ss.Bots[pi].Y)
				dx := px - bx
				dy := py - by
				if dx*dx+dy*dy < 200*200 {
					vector.StrokeLine(a, bx, by, px, py, 0.5, color.RGBA{80, 140, 255, 30}, false)
				}
			}
		}
	}

	// Class mean position — white diamond
	mx := float32(st.MeanX)
	my := float32(st.MeanY)
	vector.StrokeLine(a, mx, my-6, mx+6, my, 1, color.RGBA{200, 200, 220, 120}, false)
	vector.StrokeLine(a, mx+6, my, mx, my+6, 1, color.RGBA{200, 200, 220, 120}, false)
	vector.StrokeLine(a, mx, my+6, mx-6, my, 1, color.RGBA{200, 200, 220, 120}, false)
	vector.StrokeLine(a, mx-6, my, mx, my-6, 1, color.RGBA{200, 200, 220, 120}, false)

	// Teacher (best) — gold marker
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		tx := float32(st.BestX)
		ty := float32(st.BestY)
		vector.DrawFilledCircle(a, tx, ty, 6, color.RGBA{255, 215, 0, 200}, false)
		vector.StrokeCircle(a, tx, ty, 14, 2, color.RGBA{255, 215, 0, 120}, false)
	}
}

// drawEOOverlay visualizes the Equilibrium Optimizer.
// Shows exploration/exploitation phase, equilibrium pool positions, and personal best lines.
func drawEOOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.EO
	if st == nil {
		return
	}

	for i := range ss.Bots {
		if i >= len(st.Phase) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)

		if st.Phase[i] == 0 {
			// Exploration: violet ring
			vector.StrokeCircle(a, bx, by, 7, 1, color.RGBA{160, 80, 255, 90}, false)
		} else {
			// Exploitation: teal ring
			vector.StrokeCircle(a, bx, by, 7, 1, color.RGBA{0, 200, 180, 90}, false)
		}

		// Line to personal best
		if i < len(st.PersonalX) && i < len(st.PersonalY) {
			px := float32(st.PersonalX[i])
			py := float32(st.PersonalY[i])
			dx := px - bx
			dy := py - by
			if dx*dx+dy*dy > 4 && dx*dx+dy*dy < 120*120 {
				vector.StrokeLine(a, bx, by, px, py, 0.4, color.RGBA{160, 80, 255, 25}, false)
			}
		}
	}

	// Equilibrium pool positions — cyan diamonds
	for k := 0; k < len(st.PoolX) && k < len(st.PoolY); k++ {
		if st.PoolF[k] <= -1e17 {
			continue // uninitialized
		}
		px := float32(st.PoolX[k])
		py := float32(st.PoolY[k])
		sz := float32(5)
		vector.StrokeLine(a, px, py-sz, px+sz, py, 1, color.RGBA{0, 220, 220, 150}, false)
		vector.StrokeLine(a, px+sz, py, px, py+sz, 1, color.RGBA{0, 220, 220, 150}, false)
		vector.StrokeLine(a, px, py+sz, px-sz, py, 1, color.RGBA{0, 220, 220, 150}, false)
		vector.StrokeLine(a, px-sz, py, px, py-sz, 1, color.RGBA{0, 220, 220, 150}, false)
	}

	// Global best — gold marker
	if st.BestIdx >= 0 && st.BestIdx < len(ss.Bots) {
		gx := float32(ss.Bots[st.BestIdx].X)
		gy := float32(ss.Bots[st.BestIdx].Y)
		vector.DrawFilledCircle(a, gx, gy, 5, color.RGBA{255, 215, 0, 200}, false)
		vector.StrokeCircle(a, gx, gy, 16, 2, color.RGBA{255, 215, 0, 120}, false)
	}
}

// drawJayaOverlay visualizes the Jaya Algorithm.
// Shows attraction toward best (green) and repulsion from worst (red).
func drawJayaOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	st := ss.Jaya
	if st == nil {
		return
	}

	for i := range ss.Bots {
		if i >= len(st.Fitness) {
			continue
		}
		bx := float32(ss.Bots[i].X)
		by := float32(ss.Bots[i].Y)

		// Color by relative fitness (red=near worst, green=near best)
		fRange := st.BestF - st.WorstF
		var ratio float64
		if fRange > 1e-9 {
			ratio = (st.Fitness[i] - st.WorstF) / fRange
		}
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		r := uint8(200 * (1 - ratio))
		g := uint8(200 * ratio)
		vector.StrokeCircle(a, bx, by, 7, 1, color.RGBA{r, g, 40, 90}, false)

		// Short line toward personal best
		if i < len(st.PersonalBestX) {
			px := float32(st.PersonalBestX[i])
			py := float32(st.PersonalBestY[i])
			dx := px - bx
			dy := py - by
			if dx*dx+dy*dy > 4 && dx*dx+dy*dy < 120*120 {
				vector.StrokeLine(a, bx, by, px, py, 0.4, color.RGBA{80, 200, 80, 25}, false)
			}
		}
	}

	// Best position — gold
	if st.BestIdx >= 0 {
		bx := float32(st.BestX)
		by := float32(st.BestY)
		vector.DrawFilledCircle(a, bx, by, 6, color.RGBA{255, 215, 0, 200}, false)
		vector.StrokeCircle(a, bx, by, 14, 1.5, color.RGBA{255, 215, 0, 100}, false)
	}

	// Worst position — red X
	if st.WorstIdx >= 0 {
		wx := float32(st.WorstX)
		wy := float32(st.WorstY)
		vector.StrokeLine(a, wx-6, wy-6, wx+6, wy+6, 1.5, color.RGBA{255, 60, 60, 160}, false)
		vector.StrokeLine(a, wx-6, wy+6, wx+6, wy-6, 1.5, color.RGBA{255, 60, 60, 160}, false)
	}
}

// drawConvergenceGraph renders a real-time convergence chart for the active
// swarm algorithm. Positioned in the bottom-left of the arena viewport area,
// it shows three lines: green = best fitness, yellow = average fitness,
// cyan = population diversity (spatial spread, independently scaled 0-100).
func drawConvergenceGraph(screen *ebiten.Image, ss *swarm.SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil || len(sa.ConvergenceHistory) < 2 {
		return
	}

	// Graph dimensions and position (bottom-left of arena area)
	const gw = 220
	const gh = 100
	gx := 420 // just right of editor panel (350px + margin)
	gy := int(ss.ArenaH) + 50 - gh - 10

	// Background
	vector.DrawFilledRect(screen, float32(gx), float32(gy), gw, gh,
		color.RGBA{10, 10, 20, 220}, false)
	vector.StrokeRect(screen, float32(gx), float32(gy), gw, gh, 1,
		color.RGBA{60, 80, 120, 150}, false)

	// Title
	algoName := swarm.SwarmAlgorithmName(sa.ActiveAlgo)
	printColoredAt(screen, algoName+" Konvergenz", gx+3, gy+2, color.RGBA{136, 204, 255, 220})

	// Determine visible window (last N samples)
	best := sa.ConvergenceHistory
	avg := sa.ConvergenceAvg
	div := sa.ConvergenceDiversity
	expl := sa.ConvergenceExploration
	n := len(best)
	maxPts := gw - 10 // one pixel per sample
	start := 0
	if n > maxPts {
		start = n - maxPts
	}
	pts := n - start

	// Find min/max for fitness scaling (include archived curves for unified range)
	minV := best[start]
	maxV := best[start]
	for i := start; i < n; i++ {
		if best[i] < minV {
			minV = best[i]
		}
		if best[i] > maxV {
			maxV = best[i]
		}
		if i < len(avg) {
			if avg[i] < minV {
				minV = avg[i]
			}
			if avg[i] > maxV {
				maxV = avg[i]
			}
		}
	}
	// Extend range to include archived convergence curves.
	for _, arch := range ss.ConvergenceArchive {
		if sa.ActiveAlgo == arch.Algo && sa.FitnessFunc == arch.FitnessFunc {
			continue
		}
		for _, v := range arch.BestHistory {
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
	}
	if maxV <= minV {
		maxV = minV + 1
	}

	// Chart area (inset from borders)
	chartX := float32(gx + 5)
	chartY := float32(gy + 14)
	chartW := float32(gw - 10)
	chartH := float32(gh - 20)

	// Axis labels
	printColoredAt(screen, fmt.Sprintf("%.0f", maxV), gx+3, gy+14, color.RGBA{70, 70, 90, 150})
	printColoredAt(screen, fmt.Sprintf("%.0f", minV), gx+3, gy+gh-lineH-2, color.RGBA{70, 70, 90, 150})

	// Draw archived convergence curves from previously tested algorithms.
	// Each archived curve is drawn as a thin semi-transparent line with a
	// distinct color so users can visually compare convergence trajectories.
	archiveColors := []color.RGBA{
		{255, 120, 80, 100},  // orange
		{180, 80, 255, 100},  // purple
		{80, 255, 180, 100},  // mint
		{255, 255, 80, 100},  // yellow
		{80, 180, 255, 100},  // sky blue
		{255, 80, 180, 100},  // pink
		{180, 255, 80, 100},  // lime
		{80, 255, 255, 100},  // teal
	}
	for ai, arch := range ss.ConvergenceArchive {
		if sa.ActiveAlgo == arch.Algo && sa.FitnessFunc == arch.FitnessFunc {
			continue
		}
		ah := arch.BestHistory
		an := len(ah)
		if an < 2 {
			continue
		}
		aStart := 0
		if an > maxPts {
			aStart = an - maxPts
		}
		aPts := an - aStart
		ac := archiveColors[ai%len(archiveColors)]
		for i := 1; i < aPts; i++ {
			x0 := chartX + float32(i-1)/float32(aPts-1)*chartW
			x1 := chartX + float32(i)/float32(aPts-1)*chartW
			v0 := ah[aStart+i-1]
			v1 := ah[aStart+i]
			y0 := chartY + chartH - float32((v0-minV)/(maxV-minV))*chartH
			y1 := chartY + chartH - float32((v1-minV)/(maxV-minV))*chartH
			vector.StrokeLine(screen, x0, y0, x1, y1, 1, ac, false)
		}
	}

	// Draw exploration ratio line (magenta, behind everything, independently scaled 0-100)
	if len(expl) >= n {
		for i := 1; i < pts; i++ {
			e0 := expl[start+i-1]
			e1 := expl[start+i]
			// Skip if algorithm does not report exploration ratio (-1)
			if e0 < 0 || e1 < 0 {
				continue
			}
			x0 := chartX + float32(i-1)/float32(pts-1)*chartW
			x1 := chartX + float32(i)/float32(pts-1)*chartW
			if e0 > 100 {
				e0 = 100
			}
			if e1 > 100 {
				e1 = 100
			}
			y0 := chartY + chartH - float32(e0/100)*chartH
			y1 := chartY + chartH - float32(e1/100)*chartH
			vector.StrokeLine(screen, x0, y0, x1, y1, 1, color.RGBA{220, 100, 220, 90}, false)
		}
	}

	// Draw diversity line (cyan, behind average/best, independently scaled 0-100)
	if len(div) >= n {
		for i := 1; i < pts; i++ {
			x0 := chartX + float32(i-1)/float32(pts-1)*chartW
			x1 := chartX + float32(i)/float32(pts-1)*chartW
			// Diversity is already 0-100, scale directly to chart height
			d0 := div[start+i-1]
			d1 := div[start+i]
			if d0 > 100 {
				d0 = 100
			}
			if d1 > 100 {
				d1 = 100
			}
			y0 := chartY + chartH - float32(d0/100)*chartH
			y1 := chartY + chartH - float32(d1/100)*chartH
			vector.StrokeLine(screen, x0, y0, x1, y1, 1, color.RGBA{80, 200, 220, 100}, false)
		}
	}

	// Draw average line (yellow, behind best)
	if len(avg) >= n {
		for i := 1; i < pts; i++ {
			x0 := chartX + float32(i-1)/float32(pts-1)*chartW
			x1 := chartX + float32(i)/float32(pts-1)*chartW
			y0 := chartY + chartH - float32((avg[start+i-1]-minV)/(maxV-minV))*chartH
			y1 := chartY + chartH - float32((avg[start+i]-minV)/(maxV-minV))*chartH
			vector.StrokeLine(screen, x0, y0, x1, y1, 1, color.RGBA{220, 200, 50, 140}, false)
		}
	}

	// Draw best line (green, on top)
	for i := 1; i < pts; i++ {
		x0 := chartX + float32(i-1)/float32(pts-1)*chartW
		x1 := chartX + float32(i)/float32(pts-1)*chartW
		y0 := chartY + chartH - float32((best[start+i-1]-minV)/(maxV-minV))*chartH
		y1 := chartY + chartH - float32((best[start+i]-minV)/(maxV-minV))*chartH
		vector.StrokeLine(screen, x0, y0, x1, y1, 1.5, color.RGBA{50, 220, 80, 200}, false)
	}

	// Stagnation warning bar: red overlay when stagnation is high
	stagnPct := float32(sa.StagnationCount) / float32(swarm.StagnationThreshold)
	if stagnPct > 1 {
		stagnPct = 1
	}
	if stagnPct > 0.3 {
		// Draw stagnation progress bar at bottom of chart
		barY := chartY + chartH - 3
		barW := chartW * stagnPct
		alpha := uint8(40 + int(stagnPct*120))
		vector.DrawFilledRect(screen, chartX, barY, barW, 3, color.RGBA{255, 60, 40, alpha}, false)
	}

	// Legend
	ly := gy + gh + 2
	vector.DrawFilledRect(screen, float32(gx+3), float32(ly+2), 8, 2, color.RGBA{50, 220, 80, 200}, false)
	printColoredAt(screen, "Best", gx+14, ly, color.RGBA{50, 220, 80, 180})
	vector.DrawFilledRect(screen, float32(gx+50), float32(ly+2), 8, 2, color.RGBA{220, 200, 50, 140}, false)
	printColoredAt(screen, "Avg", gx+61, ly, color.RGBA{220, 200, 50, 140})
	vector.DrawFilledRect(screen, float32(gx+90), float32(ly+2), 8, 2, color.RGBA{80, 200, 220, 140}, false)
	printColoredAt(screen, "Div", gx+101, ly, color.RGBA{80, 200, 220, 140})
	// Show Expl legend only if algorithm reports exploration ratio
	hasExpl := len(expl) > 0 && expl[len(expl)-1] >= 0
	if hasExpl {
		vector.DrawFilledRect(screen, float32(gx+125), float32(ly+2), 8, 2, color.RGBA{220, 100, 220, 140}, false)
		printColoredAt(screen, "Expl", gx+136, ly, color.RGBA{220, 100, 220, 140})
	}

	// Current values
	curDiv := ""
	if len(div) > 0 {
		curDiv = fmt.Sprintf(" D:%.0f%%", div[len(div)-1])
	}
	curExpl := ""
	if hasExpl {
		curExpl = fmt.Sprintf(" E:%.0f%%", expl[len(expl)-1])
	}
	printColoredAt(screen, fmt.Sprintf("Best:%.1f%s%s", best[n-1], curDiv, curExpl), gx+3, ly+lineH, color.RGBA{160, 180, 200, 180})

	// Statistics line: iteration count, stagnation, perturbations
	ly += 2 * lineH
	iterStr := fmt.Sprintf("Iter:%d", sa.TotalIterations)
	stagnStr := ""
	if sa.StagnationCount > 0 {
		stagnStr = fmt.Sprintf(" Stagn:%d/%d", sa.StagnationCount, swarm.StagnationThreshold)
	}
	pertStr := ""
	if sa.PerturbationCount > 0 {
		pertStr = fmt.Sprintf(" Perturb:%d", sa.PerturbationCount)
	}

	stagnCol := color.RGBA{130, 140, 160, 180}
	if stagnPct > 0.7 {
		stagnCol = color.RGBA{255, 120, 60, 220}
	}
	printColoredAt(screen, iterStr+stagnStr+pertStr, gx+3, ly, stagnCol)

	// Archived curve legend: show abbreviation + color swatch for each archived algo.
	archiveDrawn := 0
	for ai, arch := range ss.ConvergenceArchive {
		if sa.ActiveAlgo == arch.Algo && sa.FitnessFunc == arch.FitnessFunc {
			continue
		}
		if len(arch.BestHistory) < 2 {
			continue
		}
		ac := archiveColors[ai%len(archiveColors)]
		acLabel := ac
		acLabel.A = 180
		lx := gx + 3 + archiveDrawn*45
		ly2 := ly + lineH
		vector.DrawFilledRect(screen, float32(lx), float32(ly2+2), 8, 2, ac, false)
		printColoredAt(screen, swarm.SwarmAlgorithmAbbrev(arch.Algo), lx+11, ly2, acLabel)
		archiveDrawn++
	}
}

// drawSearchTrajectory renders a small inset showing the X,Y path of the
// global best solution through the search space over time. Earlier positions
// are drawn in blue, transitioning to red for recent positions. A yellow
// marker highlights the current best. Positioned to the right of the
// convergence graph.
func drawSearchTrajectory(screen *ebiten.Image, ss *swarm.SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil || len(sa.TrajectoryX) < 2 {
		return
	}

	// Panel dimensions and position (right of convergence graph)
	const tw = 100
	const th = 100
	tx := 420 + 220 + 5 // gx + gw + gap
	ty := int(ss.ArenaH) + 50 - th - 10

	// Background
	vector.DrawFilledRect(screen, float32(tx), float32(ty), tw, th,
		color.RGBA{10, 10, 20, 220}, false)
	vector.StrokeRect(screen, float32(tx), float32(ty), tw, th, 1,
		color.RGBA{60, 80, 120, 150}, false)

	// Title
	printColoredAt(screen, "Trajektorie", tx+3, ty+2, color.RGBA{136, 204, 255, 220})

	// Chart inset
	chartX := float32(tx + 5)
	chartY := float32(ty + 14)
	chartW := float32(tw - 10)
	chartH := float32(th - 18)

	// Map arena coordinates [0, ArenaW/H] to chart area
	aw := float32(ss.ArenaW)
	ah := float32(ss.ArenaH)
	if aw < 1 {
		aw = 800
	}
	if ah < 1 {
		ah = 800
	}

	mapX := func(wx float64) float32 { return chartX + float32(wx)/aw*chartW }
	mapY := func(wy float64) float32 { return chartY + float32(wy)/ah*chartH }

	// Draw trajectory polyline with color gradient (blue→red)
	n := len(sa.TrajectoryX)
	for i := 1; i < n; i++ {
		x0, y0 := sa.TrajectoryX[i-1], sa.TrajectoryY[i-1]
		x1, y1 := sa.TrajectoryX[i], sa.TrajectoryY[i]
		// Skip invalid sentinel values
		if x0 < 0 || y0 < 0 || x1 < 0 || y1 < 0 {
			continue
		}
		// Color interpolation: blue (early) → red (recent)
		t := float32(i) / float32(n)
		r := uint8(60 + t*195)
		g := uint8(60 * (1 - t))
		b := uint8(220 * (1 - t))
		a := uint8(80 + t*140)
		c := color.RGBA{r, g, b, a}
		vector.StrokeLine(screen, mapX(x0), mapY(y0), mapX(x1), mapY(y1), 1, c, false)
	}

	// Current best position marker (yellow dot)
	lastX, lastY := sa.TrajectoryX[n-1], sa.TrajectoryY[n-1]
	if lastX >= 0 && lastY >= 0 {
		cx, cy := mapX(lastX), mapY(lastY)
		vector.DrawFilledCircle(screen, cx, cy, 3, color.RGBA{255, 220, 50, 240}, false)
	}

	// Start position marker (small blue dot)
	for i := 0; i < n; i++ {
		if sa.TrajectoryX[i] >= 0 && sa.TrajectoryY[i] >= 0 {
			sx, sy := mapX(sa.TrajectoryX[i]), mapY(sa.TrajectoryY[i])
			vector.DrawFilledCircle(screen, sx, sy, 2, color.RGBA{80, 120, 255, 180}, false)
			break
		}
	}
}

// drawFitnessHistogram renders a small histogram showing the distribution of
// per-bot fitness values for the active optimisation algorithm. 10 bins are
// used, with bar height proportional to the count in each bin. A colour
// gradient from red (low fitness) to green (high fitness) makes the
// distribution shape immediately visible. Positioned right of the trajectory
// plot.
func drawFitnessHistogram(screen *ebiten.Image, ss *swarm.SwarmState) {
	vals := swarm.GetAlgoFitnessValues(ss)
	if len(vals) < 2 {
		return
	}

	// Panel dimensions and position (right of trajectory plot)
	const hw = 100
	const hh = 100
	hx := 420 + 220 + 5 + 100 + 5 // convergence + gap + trajectory + gap = 750
	hy := int(ss.ArenaH) + 50 - hh - 10

	// Background
	vector.DrawFilledRect(screen, float32(hx), float32(hy), hw, hh,
		color.RGBA{10, 10, 20, 220}, false)
	vector.StrokeRect(screen, float32(hx), float32(hy), hw, hh, 1,
		color.RGBA{60, 80, 120, 150}, false)

	// Title
	printColoredAt(screen, "Fitness-Vertlg", hx+3, hy+2, color.RGBA{136, 204, 255, 220})

	// Compute min/max
	minV, maxV := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if maxV <= minV {
		maxV = minV + 1
	}
	rangeV := maxV - minV

	// Bin into 10 buckets
	const numBins = 10
	var bins [numBins]int
	for _, v := range vals {
		idx := int((v - minV) / rangeV * float64(numBins))
		if idx >= numBins {
			idx = numBins - 1
		}
		if idx < 0 {
			idx = 0
		}
		bins[idx]++
	}

	// Find max bin count for scaling
	maxBin := 1
	for _, c := range bins {
		if c > maxBin {
			maxBin = c
		}
	}

	// Chart area (inset)
	chartX := float32(hx + 5)
	chartY := float32(hy + 14)
	chartW := float32(hw - 10)
	chartH := float32(hh - 24)
	barW := chartW / float32(numBins)

	// Draw bars with red→green gradient
	for i := 0; i < numBins; i++ {
		t := float32(i) / float32(numBins-1)
		barH := float32(bins[i]) / float32(maxBin) * chartH
		bx := chartX + float32(i)*barW
		by := chartY + chartH - barH

		// Color: red (low fitness) → yellow (mid) → green (high fitness)
		var r, g, b uint8
		if t < 0.5 {
			r = 220
			g = uint8(200 * t * 2)
			b = 40
		} else {
			r = uint8(220 * (1 - t) * 2)
			g = 200
			b = 40
		}
		col := color.RGBA{r, g, b, 200}
		if barH > 0.5 {
			vector.DrawFilledRect(screen, bx+0.5, by, barW-1, barH, col, false)
		}
	}

	// Axis labels (min/max fitness)
	printColoredAt(screen, fmt.Sprintf("%.0f", minV), hx+3, hy+hh-lineH, color.RGBA{100, 100, 120, 160})
	printColoredAt(screen, fmt.Sprintf("%.0f", maxV), hx+hw-30, hy+hh-lineH, color.RGBA{100, 100, 120, 160})
}

// drawAlgoScoreboard renders a compact ranking of algorithms that have been
// tested on the current fitness landscape. Positioned below the convergence
// graph (or at the same location if no graph is visible). Sorted by best
// fitness descending so the top performer is first.
func drawAlgoScoreboard(screen *ebiten.Image, ss *swarm.SwarmState) {
	board := ss.AlgoScoreboard
	if len(board) == 0 {
		return
	}

	// Sort by best fitness descending
	sorted := make([]swarm.AlgoPerformanceRecord, len(board))
	copy(sorted, board)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BestFitness > sorted[j].BestFitness
	})

	// Position: right of histogram panel (convergence + trajectory + histogram + gaps)
	const sbW = 200
	sbX := 420 + 220 + 5 + 100 + 5 + 100 + 5 // gx + convW + gap + trajW + gap + histW + gap = 855
	sbY := int(ss.ArenaH) + 50 - 100 - 10      // same top as convergence graph

	// Background
	rowH := lineH + 2
	headerH := lineH + 4
	sbH := headerH + len(sorted)*rowH + 4
	vector.DrawFilledRect(screen, float32(sbX), float32(sbY), float32(sbW), float32(sbH),
		color.RGBA{10, 10, 20, 220}, false)
	vector.StrokeRect(screen, float32(sbX), float32(sbY), float32(sbW), float32(sbH), 1,
		color.RGBA{60, 80, 120, 150}, false)

	// Title
	printColoredAt(screen, "Algorithmus-Ranking", sbX+3, sbY+2, color.RGBA{136, 204, 255, 220})

	// Column headers
	y := sbY + headerH
	printColoredAt(screen, "#", sbX+3, y, color.RGBA{100, 100, 120, 160})
	printColoredAt(screen, "Algorithmus", sbX+16, y, color.RGBA{100, 100, 120, 160})
	printColoredAt(screen, "Fitness", sbX+130, y, color.RGBA{100, 100, 120, 160})
	y += rowH

	// Rows
	maxEntries := 8 // limit visible entries
	if len(sorted) < maxEntries {
		maxEntries = len(sorted)
	}
	for rank, rec := range sorted[:maxEntries] {
		// Rank color: gold/silver/bronze for top 3
		var rankCol color.RGBA
		switch rank {
		case 0:
			rankCol = color.RGBA{255, 215, 0, 220} // gold
		case 1:
			rankCol = color.RGBA{200, 200, 210, 200} // silver
		case 2:
			rankCol = color.RGBA{205, 127, 50, 200} // bronze
		default:
			rankCol = color.RGBA{140, 150, 170, 180}
		}

		name := swarm.SwarmAlgorithmName(rec.Algo)
		// Truncate long names
		if len(name) > 16 {
			name = name[:15] + "."
		}

		printColoredAt(screen, fmt.Sprintf("%d", rank+1), sbX+3, y, rankCol)
		printColoredAt(screen, name, sbX+16, y, rankCol)
		printColoredAt(screen, fmt.Sprintf("%.1f", rec.BestFitness), sbX+130, y, rankCol)
		y += rowH
	}
}

// drawAlgoTournamentProgress renders a compact progress bar showing which
// algorithm is currently being benchmarked and overall tournament progress.
func drawAlgoTournamentProgress(screen *ebiten.Image, ss *swarm.SwarmState) {
	const barW = 300
	const barH = 28
	barX := float32(250)
	barY := float32(5)

	// Background
	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{10, 15, 30, 230}, false)
	vector.StrokeRect(screen, barX, barY, barW, barH, 1, color.RGBA{80, 140, 255, 180}, false)

	// Progress fraction: completed algorithms + partial progress of current one
	total := float32(ss.AlgoTournamentTotal)
	if total < 1 {
		total = 1
	}
	tickFrac := 1.0 - float32(ss.AlgoTournamentTicks)/float32(swarm.AlgoTournamentTicksPerAlgo)
	progress := (float32(ss.AlgoTournamentDone) + tickFrac) / total

	// Progress bar fill
	fillW := (barW - 4) * progress
	fillCol := color.RGBA{40, 180, 80, 200}
	vector.DrawFilledRect(screen, barX+2, barY+2, fillW, barH-4, fillCol, false)

	// Text: "AUTO-TURNIER: PSO (3/20)"
	algoName := swarm.SwarmAlgorithmAbbrev(ss.AlgoTournamentCur)
	label := fmt.Sprintf("AUTO-TURNIER: %s (%d/%d)", algoName, ss.AlgoTournamentDone+1, ss.AlgoTournamentTotal)
	printColoredAt(screen, label, int(barX)+6, int(barY)+4, color.RGBA{220, 230, 255, 255})

	// Sub-line: ticks remaining for current algo
	tickPct := int(tickFrac * 100)
	sub := fmt.Sprintf("%d%%", tickPct)
	printColoredAt(screen, sub, int(barX)+barW-35, int(barY)+4, color.RGBA{160, 180, 200, 200})
}

// drawAlgoRadarChart renders a radar (spider) chart comparing algorithm
// performance across 4 axes: Best Fitness, Convergence Speed, Avg Fitness,
// and Final Diversity. Each algorithm is drawn as a colored polygon.
// Toggled with Ctrl+=. Requires at least 2 entries in AlgoScoreboard.
func drawAlgoRadarChart(screen *ebiten.Image, ss *swarm.SwarmState) {
	board := ss.AlgoScoreboard
	if len(board) < 2 {
		return
	}

	// Sort by best fitness descending for consistent ordering
	sorted := make([]swarm.AlgoPerformanceRecord, len(board))
	copy(sorted, board)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BestFitness > sorted[j].BestFitness
	})

	// Limit to top 6 algorithms for readability
	maxAlgos := 6
	if len(sorted) < maxAlgos {
		maxAlgos = len(sorted)
	}
	sorted = sorted[:maxAlgos]

	// Axes: BestFitness, ConvergenceSpeed (inverted: lower=better→higher value),
	// AvgFitness, FinalDiversity
	const numAxes = 4
	axisLabels := [numAxes]string{"Beste Fitness", "Konv.-Speed", "Avg Fitness", "Diversitaet"}

	// Find max values for normalisation
	maxBest := 0.0
	maxSpeed := 0.0
	maxAvg := 0.0
	maxDiv := 0.0
	for _, r := range sorted {
		if r.BestFitness > maxBest {
			maxBest = r.BestFitness
		}
		if r.ConvergenceSpeed > maxSpeed {
			maxSpeed = r.ConvergenceSpeed
		}
		if r.AvgFitness > maxAvg {
			maxAvg = r.AvgFitness
		}
		if r.FinalDiversity > maxDiv {
			maxDiv = r.FinalDiversity
		}
	}
	if maxBest < 1 {
		maxBest = 1
	}
	if maxSpeed < 1 {
		maxSpeed = 1
	}
	if maxAvg < 1 {
		maxAvg = 1
	}
	if maxDiv < 0.01 {
		maxDiv = 0.01
	}

	// Chart geometry — position in upper-right of arena
	cx := float32(ss.ArenaW) - 10 - 110 // center X
	cy := float32(130)                   // center Y
	radius := float32(90)

	// Background circle
	const bgAlpha = 200
	vector.DrawFilledCircle(screen, cx, cy, radius+15, color.RGBA{10, 10, 25, bgAlpha}, false)
	vector.StrokeCircle(screen, cx, cy, radius+15, 1, color.RGBA{60, 80, 120, 150}, false)

	// Title
	printColoredAt(screen, "Algorithmus-Radar", int(cx)-55, int(cy-radius)-25,
		color.RGBA{136, 204, 255, 220})

	// Draw axis lines and labels
	for a := 0; a < numAxes; a++ {
		angle := float64(a)*2*math.Pi/float64(numAxes) - math.Pi/2
		ex := cx + radius*float32(math.Cos(angle))
		ey := cy + radius*float32(math.Sin(angle))
		vector.StrokeLine(screen, cx, cy, ex, ey, 1, color.RGBA{50, 60, 80, 180}, false)

		// Label position (slightly past endpoint)
		lx := cx + (radius+12)*float32(math.Cos(angle))
		ly := cy + (radius+12)*float32(math.Sin(angle))
		label := axisLabels[a]
		// Center text approximately
		labelOff := len(label) * charW / 2
		printColoredAt(screen, label, int(lx)-labelOff, int(ly)-5,
			color.RGBA{140, 160, 190, 200})
	}

	// Draw concentric guide rings at 25%, 50%, 75%
	for _, frac := range []float32{0.25, 0.5, 0.75} {
		r := radius * frac
		vector.StrokeCircle(screen, cx, cy, r, 0.5, color.RGBA{40, 50, 70, 100}, false)
	}

	// Algorithm colors (distinct, semi-transparent for fill)
	algoColors := []color.RGBA{
		{255, 100, 80, 255},  // red-orange
		{80, 180, 255, 255},  // sky blue
		{120, 255, 120, 255}, // green
		{255, 200, 60, 255},  // gold
		{200, 120, 255, 255}, // purple
		{255, 140, 200, 255}, // pink
	}

	// Draw polygon for each algorithm
	for ai, rec := range sorted {
		col := algoColors[ai%len(algoColors)]
		fillCol := color.RGBA{col.R, col.G, col.B, 40}

		// Normalise values to [0, 1]
		vals := [numAxes]float64{
			rec.BestFitness / maxBest,
			1.0 - rec.ConvergenceSpeed/maxSpeed, // invert: lower speed = better
			rec.AvgFitness / maxAvg,
			rec.FinalDiversity / maxDiv,
		}

		// Compute polygon vertices
		var verts [numAxes][2]float32
		for a := 0; a < numAxes; a++ {
			angle := float64(a)*2*math.Pi/float64(numAxes) - math.Pi/2
			v := vals[a]
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			r := float32(v) * radius
			verts[a] = [2]float32{
				cx + r*float32(math.Cos(angle)),
				cy + r*float32(math.Sin(angle)),
			}
		}

		// Draw filled triangles (fan from center) for the polygon
		for a := 0; a < numAxes; a++ {
			next := (a + 1) % numAxes
			drawTriangleFill(screen, cx, cy, verts[a][0], verts[a][1],
				verts[next][0], verts[next][1], fillCol)
		}

		// Draw polygon outline
		for a := 0; a < numAxes; a++ {
			next := (a + 1) % numAxes
			vector.StrokeLine(screen, verts[a][0], verts[a][1],
				verts[next][0], verts[next][1], 1.5, col, false)
		}

		// Draw vertex dots
		for a := 0; a < numAxes; a++ {
			vector.DrawFilledCircle(screen, verts[a][0], verts[a][1], 2.5, col, false)
		}
	}

	// Legend
	ly := int(cy + radius + 20)
	for ai, rec := range sorted {
		col := algoColors[ai%len(algoColors)]
		name := swarm.SwarmAlgorithmAbbrev(rec.Algo)
		lx := int(cx) - 55 + (ai%3)*42
		if ai >= 3 {
			ly = int(cy+radius) + 20 + lineH + 2
			lx = int(cx) - 55 + (ai%3)*42
		}
		// Color swatch
		vector.DrawFilledRect(screen, float32(lx), float32(ly+1), 8, 8, col, false)
		printColoredAt(screen, name, lx+10, ly, color.RGBA{180, 190, 210, 220})
	}
}

// radarWhitePixel is a 1x1 white image used as source texture for DrawTriangles.
var radarWhitePixel *ebiten.Image

func getRadarWhitePixel() *ebiten.Image {
	if radarWhitePixel == nil {
		radarWhitePixel = ebiten.NewImage(1, 1)
		radarWhitePixel.Fill(color.White)
	}
	return radarWhitePixel
}

// drawTriangleFill renders a filled triangle using ebiten's vertex-based rendering.
func drawTriangleFill(screen *ebiten.Image, x0, y0, x1, y1, x2, y2 float32, col color.RGBA) {
	vs := []ebiten.Vertex{
		{DstX: x0, DstY: y0, SrcX: 0, SrcY: 0, ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255, ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
		{DstX: x1, DstY: y1, SrcX: 0, SrcY: 0, ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255, ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
		{DstX: x2, DstY: y2, SrcX: 0, SrcY: 0, ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255, ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255},
	}
	is := []uint16{0, 1, 2}
	screen.DrawTriangles(vs, is, getRadarWhitePixel(), nil)
}
