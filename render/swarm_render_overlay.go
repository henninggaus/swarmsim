package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
	lx := int(bx) - runeLen(label)*charW/2
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

