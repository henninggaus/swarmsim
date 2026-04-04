package render

import (
	"image/color"
	"math"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
		vector.DrawFilledCircle(a, rx, ry, 6, ColorGoldFaded, false)
		vector.StrokeCircle(a, rx, ry, 14, 1.5, color.RGBA{255, 215, 0, 100}, false)
		// Crosshair lines
		vector.StrokeLine(a, rx-18, ry, rx-8, ry, 1, color.RGBA{255, 215, 0, 120}, false)
		vector.StrokeLine(a, rx+8, ry, rx+18, ry, 1, color.RGBA{255, 215, 0, 120}, false)
		vector.StrokeLine(a, rx, ry-18, rx, ry-8, 1, color.RGBA{255, 215, 0, 120}, false)
		vector.StrokeLine(a, rx, ry+8, rx, ry+18, 1, color.RGBA{255, 215, 0, 120}, false)
	}
}

// drawSSAOverlay visualizes the Salp Swarm Algorithm.
// Shows chain links between leader->follower salps and food source marker.
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
	vector.DrawFilledCircle(a, fx, fy, 6, ColorGoldFaded, false)
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
		vector.DrawFilledCircle(a, gx, gy, 5, ColorGoldFaded, false)
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
		vector.DrawFilledCircle(a, px, py, 5, ColorGoldFaded, false)
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
		vector.DrawFilledCircle(a, fx, fy, 6, ColorGoldFaded, false)
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
		vector.DrawFilledCircle(a, tx, ty, 6, ColorGoldFaded, false)
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
		vector.DrawFilledCircle(a, gx, gy, 5, ColorGoldFaded, false)
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
		vector.DrawFilledCircle(a, bx, by, 6, ColorGoldFaded, false)
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
