package swarm

import (
	"fmt"
	"math"
	"swarmsim/locale"
)

// traceGWO populates the math trace for the Grey Wolf Optimizer.
func traceGWO(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.GWO
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Grey Wolf (GWO)"

	// Best bot: local walk
	if idx == st.AlphaIdx || idx == st.GlobalBestIdx {
		mt.PhaseName = locale.T("math.phase.local_walk")
		mt.AddStep("role", "Alpha (best)", "Local random walk", 0, MathBranch)
		return
	}

	rank := 3
	if idx < len(st.Rank) {
		rank = st.Rank[idx]
	}
	rankNames := []string{"Alpha", "Beta", "Delta", "Omega"}
	if rank >= 0 && rank < len(rankNames) {
		mt.PhaseName = fmt.Sprintf("%s: %s", locale.T("math.phase.wolf_hunt"), rankNames[rank])
	}

	progress := float64(st.HuntTick) / float64(gwoMaxTicks)
	a := 2.0 * (1.0 - progress)
	if a < 0 {
		a = 0
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.HuntTick, gwoMaxTicks), progress, MathInput)
	mt.AddStep("a", "2 * (1 - progress)", fmt.Sprintf("2 * (1 - %.3f)", progress), a, MathIntermediate)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)
	mt.AddStep("alpha_pos", "(ax, ay)", fmt.Sprintf("(%.1f, %.1f)", st.AlphaX, st.AlphaY), st.AlphaX, MathInput)
	mt.AddStep("beta_pos", "(bx, by)", fmt.Sprintf("(%.1f, %.1f)", st.BetaX, st.BetaY), st.BetaX, MathInput)
	mt.AddStep("delta_pos", "(dx, dy)", fmt.Sprintf("(%.1f, %.1f)", st.DeltaX, st.DeltaY), st.DeltaX, MathInput)

	// Target: average of alpha, beta, delta positions
	targetX := (st.AlphaX + st.BetaX + st.DeltaX) / 3.0
	targetY := (st.AlphaY + st.BetaY + st.DeltaY) / 3.0
	mt.AddStep("target", "(a+b+d)/3", fmt.Sprintf("(%.1f, %.1f)", targetX, targetY), targetX, MathOutput)
	mt.AddStep("globalBest", "best fitness", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceWOA populates the math trace for the Whale Optimization Algorithm.
func traceWOA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.WOA
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Whale (WOA)"

	// Best bot: direct-to-best or local walk
	isDirect := false
	if idx < len(st.IsDirect) {
		isDirect = st.IsDirect[idx]
	}
	if idx == st.BestIdx && !isDirect {
		mt.PhaseName = locale.T("math.phase.local_walk")
		mt.AddStep("role", "Best whale", "Local random walk", 0, MathBranch)
		return
	}
	if isDirect {
		mt.PhaseName = locale.T("math.phase.direct_to_best")
		mt.AddStep("mode", "Direct", "Moving toward best", 0, MathBranch)
	}

	progress := float64(st.HuntTick) / float64(woaMaxTicks)
	a := 2.0 * (1.0 - progress)
	if a < 0 {
		a = 0
	}

	phase := 0
	if idx < len(st.Phase) {
		phase = st.Phase[idx]
	}
	phaseNames := []string{locale.T("math.phase.encircle"), locale.T("math.phase.spiral"), locale.T("math.phase.search")}
	if !isDirect && phase >= 0 && phase < len(phaseNames) {
		mt.PhaseName = phaseNames[phase]
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.HuntTick, woaMaxTicks), progress, MathInput)
	mt.AddStep("a", "2 * (1 - progress)", fmt.Sprintf("2 * (1 - %.3f)", progress), a, MathIntermediate)

	// A and C coefficients
	A := 2*a*0.5 - a // representative: r=0.5
	C := 2 * 0.5
	mt.AddStep("A", "2*a*r - a", fmt.Sprintf("2*%.2f*0.5 - %.2f", a, a), A, MathIntermediate)
	mt.AddStep("C", "2*r", "2*0.5", C, MathIntermediate)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)
	mt.AddStep("best_pos", "(bx, by)", fmt.Sprintf("(%.1f, %.1f)", st.BestX, st.BestY), st.BestF, MathOutput)

	if phase == 1 {
		mt.AddStep("spiral_b", "const", fmt.Sprintf("%.1f", woaSpiralB), woaSpiralB, MathBranch)
	}
	mt.AddStep("globalBest", "best fitness", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceSCA populates the math trace for the Sine Cosine Algorithm.
func traceSCA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.SCA
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Sine Cosine (SCA)"

	// Best bot: local walk
	if idx == st.GlobalBestIdx {
		mt.PhaseName = locale.T("math.phase.local_walk")
		mt.AddStep("role", "Global best", "Local random walk", 0, MathBranch)
		return
	}

	progress := float64(st.Tick) / float64(scaMaxTicks)
	r1 := scaAMax * (1.0 - progress)
	if r1 < scaAMin {
		r1 = scaAMin
	}

	phase := 0
	if idx < len(st.Phase) {
		phase = st.Phase[idx]
	}
	if phase == 0 {
		mt.PhaseName = locale.T("math.phase.sine_phase")
	} else {
		mt.PhaseName = locale.T("math.phase.cosine_phase")
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.Tick, scaMaxTicks), progress, MathInput)
	mt.AddStep("r1", "a * (1 - progress)", fmt.Sprintf("%.2f * (1 - %.3f)", scaAMax, progress), r1, MathIntermediate)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)
	mt.AddStep("dest", "(Px, Py)", fmt.Sprintf("(%.1f, %.1f)", st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathInput)

	dist := math.Sqrt((bot.X-st.GlobalBestX)*(bot.X-st.GlobalBestX) + (bot.Y-st.GlobalBestY)*(bot.Y-st.GlobalBestY))
	mt.AddStep("dist", "|P - X|", fmt.Sprintf("%.2f", dist), dist, MathIntermediate)

	if phase == 0 {
		offset := r1 * math.Sin(1.0) * dist
		mt.AddStep("offset", "r1*sin(r2)*|P-X|", fmt.Sprintf("%.3f*sin(1)*%.2f", r1, dist), offset, MathOutput)
	} else {
		offset := r1 * math.Cos(1.0) * dist
		mt.AddStep("offset", "r1*cos(r2)*|P-X|", fmt.Sprintf("%.3f*cos(1)*%.2f", r1, dist), offset, MathOutput)
	}
	mt.AddStep("globalBest", "best fitness", fmt.Sprintf("%.2f", st.GlobalBestF), st.GlobalBestF, MathOutput)
}

// tracePSO populates the math trace for Particle Swarm Optimization.
func tracePSO(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.PSO
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Particle Swarm (PSO)"
	mt.PhaseName = locale.T("math.phase.velocity_update")

	w := psoInertia
	c1 := psoCognitive
	c2 := psoSocial

	mt.AddStep("w", "inertia", fmt.Sprintf("%.2f", w), w, MathInput)
	mt.AddStep("c1", "cognitive", fmt.Sprintf("%.2f", c1), c1, MathInput)
	mt.AddStep("c2", "social", fmt.Sprintf("%.2f", c2), c2, MathInput)

	velX, velY := 0.0, 0.0
	if idx < len(st.VelX) {
		velX = st.VelX[idx]
		velY = st.VelY[idx]
	}
	mt.AddStep("vel", "(vx, vy)", fmt.Sprintf("(%.2f, %.2f)", velX, velY), math.Sqrt(velX*velX+velY*velY), MathIntermediate)

	pBestX, pBestY, pBestF := 0.0, 0.0, 0.0
	if idx < len(st.BestX) {
		pBestX = st.BestX[idx]
		pBestY = st.BestY[idx]
		pBestF = st.BestFit[idx]
	}
	mt.AddStep("pBest", "(px, py)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", pBestX, pBestY, pBestF), pBestF, MathInput)
	mt.AddStep("gBest", "(gx, gy)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.GlobalX, st.GlobalY, st.GlobalFit), st.GlobalFit, MathInput)

	// Cognitive and social components
	cogX := c1 * (pBestX - bot.X)
	socX := c2 * (st.GlobalX - bot.X)
	mt.AddStep("cognitive", "c1*(pBest-x)", fmt.Sprintf("%.2f*(%.1f-%.1f)", c1, pBestX, bot.X), cogX, MathIntermediate)
	mt.AddStep("social", "c2*(gBest-x)", fmt.Sprintf("%.2f*(%.1f-%.1f)", c2, st.GlobalX, bot.X), socX, MathIntermediate)

	newVelX := w*velX + cogX*0.5 + socX*0.5 // approximate (r1,r2 ~ 0.5)
	mt.AddStep("newVel_x", "w*v + cog + soc", fmt.Sprintf("%.2f*%.2f + %.2f + %.2f", w, velX, cogX*0.5, socX*0.5), newVelX, MathOutput)
}

// traceDE populates the math trace for Differential Evolution.
func traceDE(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.DE
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Differential Evolution (DE)"

	moving := false
	if idx < len(st.Moving) {
		moving = st.Moving[idx]
	}
	if moving {
		mt.PhaseName = locale.T("math.phase.mutation_crossover")
	} else {
		mt.PhaseName = locale.T("math.phase.mutation_crossover")
	}

	mt.AddStep("F", "mutation scale", fmt.Sprintf("%.2f", st.DifferentialWeight), st.DifferentialWeight, MathInput)
	mt.AddStep("CR", "crossover rate", fmt.Sprintf("%.2f", st.CrossoverRate), st.CrossoverRate, MathInput)

	progress := float64(st.GenTick) / float64(deMaxTicks)
	mt.AddStep("progress", "gen / maxGen", fmt.Sprintf("%d / %d", st.GenTick, deMaxTicks), progress, MathInput)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	trialX, trialY := 0.0, 0.0
	if idx < len(st.TrialX) {
		trialX = st.TrialX[idx]
		trialY = st.TrialY[idx]
	}
	mt.AddStep("trial", "(tx, ty)", fmt.Sprintf("(%.1f, %.1f)", trialX, trialY), trialX, MathIntermediate)
	mt.AddStep("bestPos", "(bx, by)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.BestX, st.BestY, st.BestF), st.BestF, MathOutput)
}

// traceCuckoo populates the math trace for Cuckoo Search.
func traceCuckoo(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.Cuckoo
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Cuckoo Search (CS)"
	mt.PhaseName = locale.T("math.phase.levy_flight") + " + " + locale.T("math.phase.nest_abandon")

	mt.AddStep("levy_a", "Levy exponent", fmt.Sprintf("%.2f", csLevyAlpha), csLevyAlpha, MathInput)
	mt.AddStep("step_s", "step scale", fmt.Sprintf("%.2f", csStepScale), csStepScale, MathInput)
	mt.AddStep("pa", "abandon prob", fmt.Sprintf("%.3f", csDiscoveryProb), csDiscoveryProb, MathInput)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	nestAge := 0
	if idx < len(st.NestAge) {
		nestAge = st.NestAge[idx]
	}
	mt.AddStep("nestAge", "ticks", fmt.Sprintf("%d", nestAge), float64(nestAge), MathIntermediate)

	mt.AddStep("bestNest", "(bx, by)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.BestX, st.BestY, st.BestF), st.BestF, MathOutput)

	gbF := st.GlobalBestF
	if gbF > st.BestF {
		mt.AddStep("globalBest", "persistent", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
	}
}

// traceABC populates the math trace for Artificial Bee Colony.
func traceABC(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.ABC
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Artificial Bee Colony (ABC)"

	role := 0
	if idx < len(st.Role) {
		role = st.Role[idx]
	}
	roleNames := []string{locale.T("math.phase.employed"), locale.T("math.phase.onlooker"), locale.T("math.phase.scout")}
	if role >= 0 && role < len(roleNames) {
		mt.PhaseName = roleNames[role]
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.Tick, abcMaxTicks), float64(st.Tick)/float64(abcMaxTicks), MathInput)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	stale := 0
	if idx < len(st.Stale) {
		stale = st.Stale[idx]
	}
	mt.AddStep("stale", "no improvement", fmt.Sprintf("%d / %d", stale, abcAbandonLimit), float64(stale), MathIntermediate)

	trialX, trialY := 0.0, 0.0
	if idx < len(st.TrialX) {
		trialX = st.TrialX[idx]
		trialY = st.TrialY[idx]
	}
	mt.AddStep("trial", "(tx, ty)", fmt.Sprintf("(%.1f, %.1f)", trialX, trialY), trialX, MathIntermediate)
	mt.AddStep("localStep", "perturbation R", fmt.Sprintf("%.1f", abcLocalStep), abcLocalStep, MathInput)
	mt.AddStep("bestFood", "(gx, gy)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.GlobalBestX, st.GlobalBestY, st.GlobalBestF), st.GlobalBestF, MathOutput)
}

// traceBFO populates the math trace for Bacterial Foraging Optimization.
func traceBFO(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.BFO
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Bacterial Foraging (BFO)"

	progress := float64(st.Tick) / float64(bfoMaxTicks)
	probeD := bfoProbeDistStart + (bfoProbeDistEnd-bfoProbeDistStart)*progress

	swimCount := 0
	if idx < len(st.SwimCount) {
		swimCount = st.SwimCount[idx]
	}
	if swimCount > 0 {
		mt.PhaseName = locale.T("math.phase.swim")
	} else {
		mt.PhaseName = locale.T("math.phase.tumble")
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.Tick, bfoMaxTicks), progress, MathInput)
	mt.AddStep("probeD", "adaptive step", fmt.Sprintf("%.1f -> %.1f @ %.1f", bfoProbeDistStart, bfoProbeDistEnd, probeD), probeD, MathIntermediate)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	swimDir := 0.0
	if idx < len(st.SwimDir) {
		swimDir = st.SwimDir[idx]
	}
	mt.AddStep("swimDir", "direction", fmt.Sprintf("%.2f rad", swimDir), swimDir, MathIntermediate)
	mt.AddStep("swimLeft", "steps remain", fmt.Sprintf("%d / %d", swimCount, bfoChemoSteps), float64(swimCount), MathIntermediate)

	gbW := bfoGBestWStart + (bfoGBestWEnd-bfoGBestWStart)*progress
	mt.AddStep("gbWeight", "GB attract", fmt.Sprintf("%.3f", gbW), gbW, MathIntermediate)
	mt.AddStep("bestPos", "(bx, by)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.BestX, st.BestY, st.BestF), st.BestF, MathOutput)
}

// traceMFO populates the math trace for Moth-Flame Optimization.
func traceMFO(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.MFO
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Moth-Flame (MFO)"

	progress := float64(st.Tick) / float64(mfoMaxTicks)

	flameIdx := -1
	if idx < len(st.MothFlame) {
		flameIdx = st.MothFlame[idx]
	}
	if flameIdx >= 0 && flameIdx < len(st.Flames) {
		mt.PhaseName = fmt.Sprintf("%s (flame %d)", locale.T("math.phase.spiral_flight"), flameIdx)
	} else {
		mt.PhaseName = locale.T("math.phase.spiral_flight")
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.Tick, mfoMaxTicks), progress, MathInput)
	mt.AddStep("spiral_b", "shape const", fmt.Sprintf("%.2f", mfoSpiralB), mfoSpiralB, MathInput)

	spiralT := 0.0
	if idx < len(st.SpiralT) {
		spiralT = st.SpiralT[idx]
	}
	mt.AddStep("t", "spiral param", fmt.Sprintf("%.3f", spiralT), spiralT, MathIntermediate)

	fitness := 0.0
	if idx < len(st.BotFitness) {
		fitness = st.BotFitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	if flameIdx >= 0 && flameIdx < len(st.Flames) {
		fl := st.Flames[flameIdx]
		dist := math.Sqrt((bot.X-fl.X)*(bot.X-fl.X) + (bot.Y-fl.Y)*(bot.Y-fl.Y))
		mt.AddStep("flameDist", "|moth - flame|", fmt.Sprintf("%.2f", dist), dist, MathIntermediate)
		spiralVal := dist * math.Exp(mfoSpiralB*spiralT) * math.Cos(2*math.Pi*spiralT)
		mt.AddStep("spiralD", "D*e^(bt)*cos(2pi*t)", fmt.Sprintf("%.2f", spiralVal), spiralVal, MathOutput)
	}
	mt.AddStep("globalBest", "best fitness", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceBat populates the math trace for the Bat Algorithm.
func traceBat(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.Bat
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Bat Algorithm (BA)"
	mt.PhaseName = locale.T("math.phase.echolocation")

	freq := 0.0
	if idx < len(st.Freq) {
		freq = st.Freq[idx]
	}
	mt.AddStep("freq", "f in [fMin, fMax]", fmt.Sprintf("%.3f in [%.1f, %.1f]", freq, batFMin, batFMax), freq, MathInput)

	loud := 0.0
	if idx < len(st.Loud) {
		loud = st.Loud[idx]
	}
	mt.AddStep("loudness", "A (exploration)", fmt.Sprintf("%.3f (decay=%.3f)", loud, batAlpha), loud, MathIntermediate)

	pulse := 0.0
	if idx < len(st.Pulse) {
		pulse = st.Pulse[idx]
	}
	mt.AddStep("pulse_r", "rate (local)", fmt.Sprintf("%.3f (gamma=%.2f)", pulse, batGamma), pulse, MathIntermediate)

	velMag := 0.0
	if idx < len(st.Vel[0]) {
		vx := st.Vel[0][idx]
		vy := st.Vel[1][idx]
		velMag = math.Sqrt(vx*vx + vy*vy)
		mt.AddStep("velocity", "(vx, vy)", fmt.Sprintf("(%.2f, %.2f)", vx, vy), velMag, MathIntermediate)
	}

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)
	mt.AddStep("bestPos", "(bx, by)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.BestX, st.BestY, st.BestF), st.BestF, MathOutput)
	mt.AddStep("globalBest", "persistent", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceHHO populates the math trace for Harris Hawks Optimization.
func traceHHO(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.HHO
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Harris Hawks (HHO)"

	progress := float64(st.HuntTick) / float64(hhoMaxTicks)
	E := 2.0 * (1.0 - progress)
	if E < 0 {
		E = 0
	}

	phase := 0
	if idx < len(st.Phase) {
		phase = st.Phase[idx]
	}
	phaseNames := []string{locale.T("math.phase.exploration"), locale.T("math.phase.soft_siege"), locale.T("math.phase.hard_siege"), locale.T("math.phase.rapid_dive")}
	if phase >= 0 && phase < len(phaseNames) {
		mt.PhaseName = phaseNames[phase]
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.HuntTick, hhoMaxTicks), progress, MathInput)
	mt.AddStep("E_escape", "2*(1-progress)", fmt.Sprintf("2*(1-%.3f)", progress), E, MathIntermediate)
	mt.AddStep("|E|", "escape energy", fmt.Sprintf("%.3f", math.Abs(E)), math.Abs(E), MathBranch)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)
	mt.AddStep("rabbit", "(rx, ry)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.BestX, st.BestY, st.BestF), st.BestF, MathOutput)

	gbW := hhoGBWeightMin + (hhoGBWeightMax-hhoGBWeightMin)*progress
	mt.AddStep("gbWeight", "GB attract", fmt.Sprintf("%.3f", gbW), gbW, MathIntermediate)
	mt.AddStep("globalBest", "persistent", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceSSA populates the math trace for the Salp Swarm Algorithm.
func traceSSA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.SSA
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Salp Swarm (SSA)"

	role := 0
	if idx < len(st.Role) {
		role = st.Role[idx]
	}
	if role == 0 {
		mt.PhaseName = locale.T("math.phase.leader")
	} else {
		mt.PhaseName = locale.T("math.phase.follower")
	}

	t := float64(st.CycleTick)
	T := float64(ssaMaxTicks)
	c1 := 2.0 * math.Exp(-math.Pow(4*t/T, 2))

	mt.AddStep("c1", "2*exp(-(4t/T)^2)", fmt.Sprintf("2*exp(-(4*%.0f/%.0f)^2)", t, T), c1, MathIntermediate)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)
	mt.AddStep("food", "(fx, fy)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.FoodX, st.FoodY, st.FoodFit), st.FoodFit, MathInput)

	if role == 0 {
		c2 := 0.5 // representative
		c3 := 0.5
		offset := c1 * ((st.FoodX - bot.X) * c3)
		mt.AddStep("c2", "rand [0,1]", "~0.5", c2, MathIntermediate)
		mt.AddStep("c3", "rand [0,1]", "~0.5", c3, MathIntermediate)
		mt.AddStep("offset", "c1*(F-x)*c3", fmt.Sprintf("%.3f*(%.1f-%.1f)*0.5", c1, st.FoodX, bot.X), offset, MathOutput)
	} else {
		chainIdx := -1
		if idx < len(st.ChainIdx) {
			chainIdx = st.ChainIdx[idx]
		}
		mt.AddStep("chain", "follow idx", fmt.Sprintf("%d", chainIdx), float64(chainIdx), MathBranch)
		mt.AddStep("update", "(x_i + x_{i-1})/2", "average with leader", 0, MathOutput)
	}
}

// traceGSA populates the math trace for the Gravitational Search Algorithm.
func traceGSA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.GSA
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Gravitational Search (GSA)"
	mt.PhaseName = locale.T("math.phase.gravitational")

	mt.AddStep("G", "grav. const", fmt.Sprintf("%.3f (G0=%.1f)", st.G, gsaG0), st.G, MathInput)

	mass := 0.0
	if idx < len(st.Mass) {
		mass = st.Mass[idx]
	}
	mt.AddStep("mass", "normalised", fmt.Sprintf("%.4f", mass), mass, MathInput)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	accX, accY := 0.0, 0.0
	if idx < len(st.AccX) {
		accX = st.AccX[idx]
		accY = st.AccY[idx]
	}
	accMag := math.Sqrt(accX*accX + accY*accY)
	mt.AddStep("accel", "(ax, ay)", fmt.Sprintf("(%.3f, %.3f)", accX, accY), accMag, MathIntermediate)

	// Force = G * M_j * M_i / R (representative)
	mt.AddStep("F", "G*Mj*Mi/R", fmt.Sprintf("%.3f * %.4f * Mj / R", st.G, mass), 0, MathIntermediate)

	progress := float64(st.Tick) / float64(gsaMaxTicks)
	mt.AddStep("progress", "tick / max", fmt.Sprintf("%d / %d", st.Tick, gsaMaxTicks), progress, MathInput)
	mt.AddStep("globalBest", "best fitness", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceFPA populates the math trace for Flower Pollination Algorithm.
func traceFPA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.FPA
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Flower Pollination (FPA)"

	isGlobal := false
	if idx < len(st.IsGlobal) {
		isGlobal = st.IsGlobal[idx]
	}
	if isGlobal {
		mt.PhaseName = locale.T("math.phase.global_levy")
	} else {
		mt.PhaseName = locale.T("math.phase.local_wind")
	}

	mt.AddStep("p_switch", "switch prob", fmt.Sprintf("%.2f", fpaSwitchProb), fpaSwitchProb, MathInput)

	progress := float64(st.PollTick) / float64(fpaMaxTicks)
	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.PollTick, fpaMaxTicks), progress, MathInput)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	pBestF := 0.0
	if idx < len(st.BestFit) {
		pBestF = st.BestFit[idx]
	}
	mt.AddStep("pBest", "personal best", fmt.Sprintf("%.2f", pBestF), pBestF, MathIntermediate)

	if isGlobal {
		mt.AddStep("levy", "Levy flight", "L(lambda)", 0, MathBranch)
		dist := math.Sqrt((bot.X-st.GlobalBestX)*(bot.X-st.GlobalBestX) + (bot.Y-st.GlobalBestY)*(bot.Y-st.GlobalBestY))
		mt.AddStep("step", "L*(x-gBest)", fmt.Sprintf("L * %.2f", dist), dist, MathIntermediate)
	} else {
		mt.AddStep("epsilon", "rand diff", "x_j - x_k", 0, MathBranch)
	}
	mt.AddStep("globalBest", "best fitness", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceSA populates the math trace for Simulated Annealing.
func traceSA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.SA
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Simulated Annealing (SA)"

	temp := 0.0
	if idx < len(st.Temp) {
		temp = st.Temp[idx]
	}
	accepted := false
	if idx < len(st.Accepted) {
		accepted = st.Accepted[idx]
	}
	if accepted {
		mt.PhaseName = locale.T("math.phase.annealing")
	} else {
		mt.PhaseName = locale.T("math.phase.annealing")
	}

	mt.AddStep("T", "temperature", fmt.Sprintf("%.3f (T0=%.1f)", temp, st.InitialTemp), temp, MathInput)
	mt.AddStep("Tmin", "min temp", fmt.Sprintf("%.3f", st.MinTemp), st.MinTemp, MathInput)
	mt.AddStep("alpha", "cooling rate", fmt.Sprintf("%.4f", st.CoolingRate), st.CoolingRate, MathInput)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	pBestF := 0.0
	if idx < len(st.BestF) {
		pBestF = st.BestF[idx]
	}
	deltaE := fitness - pBestF
	mt.AddStep("dE", "f_new - f_best", fmt.Sprintf("%.2f - %.2f", fitness, pBestF), deltaE, MathIntermediate)

	if deltaE < 0 && temp > 0 {
		prob := math.Exp(deltaE / temp)
		mt.AddStep("P_accept", "exp(dE/T)", fmt.Sprintf("exp(%.2f / %.3f)", deltaE, temp), prob, MathBranch)
	} else {
		mt.AddStep("P_accept", "1 (better)", "1.0", 1.0, MathBranch)
	}

	mt.AddStep("T_next", "alpha * T", fmt.Sprintf("%.4f * %.3f", st.CoolingRate, temp), st.CoolingRate*temp, MathOutput)
	mt.AddStep("globalBest", "best fitness", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceAO populates the math trace for the Aquila Optimizer.
func traceAO(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.AO
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Aquila Optimizer (AO)"

	progress := float64(st.HuntTick) / float64(aoMaxTicks)
	phase := 0
	if idx < len(st.Phase) {
		phase = st.Phase[idx]
	}
	phaseNames := []string{locale.T("math.phase.soar"), locale.T("math.phase.contour"), locale.T("math.phase.glide"), locale.T("math.phase.attack")}
	if phase >= 0 && phase < len(phaseNames) {
		mt.PhaseName = phaseNames[phase]
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.HuntTick, aoMaxTicks), progress, MathInput)
	mt.AddStep("phase", "hunt mode", fmt.Sprintf("%d (%s)", phase, phaseNames[phase]), float64(phase), MathBranch)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)
	mt.AddStep("mean", "(mx, my)", fmt.Sprintf("(%.1f, %.1f)", st.MeanX, st.MeanY), st.MeanX, MathInput)
	mt.AddStep("prey", "(px, py)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.BestX, st.BestY, st.BestF), st.BestF, MathOutput)

	if phase == 0 || phase == 1 {
		mt.AddStep("levy", "exploration", "Levy component", 0, MathIntermediate)
	}
	mt.AddStep("globalBest", "persistent", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceDA populates the math trace for the Dragonfly Algorithm.
func traceDA(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.DA
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Dragonfly (DA)"

	progress := float64(st.Tick) / float64(daMaxTicks)
	w := 0.9 - 0.4*progress // inertia weight decreases

	role := 0
	if idx < len(st.Role) {
		role = st.Role[idx]
	}
	if role >= 0 && role < 3 {
		mt.PhaseName = locale.T("math.phase.dragonfly_swarm")
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.Tick, daMaxTicks), progress, MathInput)
	mt.AddStep("w_inertia", "0.9 - 0.4*p", fmt.Sprintf("0.9 - 0.4*%.3f", progress), w, MathIntermediate)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	// Behavioral weights (scaled by progress)
	s := 2.0 * progress  // separation increases
	a := 2.0 * progress  // alignment increases
	c := 2.0 * progress  // cohesion increases
	f := 2.0 * (1 - progress) // food attraction decreases with convergence
	e := progress             // enemy repulsion

	mt.AddStep("S_sep", "separation wt", fmt.Sprintf("%.3f", s), s, MathIntermediate)
	mt.AddStep("A_align", "alignment wt", fmt.Sprintf("%.3f", a), a, MathIntermediate)
	mt.AddStep("C_coh", "cohesion wt", fmt.Sprintf("%.3f", c), c, MathIntermediate)
	mt.AddStep("F_food", "food attract", fmt.Sprintf("toward (%.1f, %.1f)", st.BestX, st.BestY), f, MathOutput)
	mt.AddStep("E_enemy", "enemy repel", fmt.Sprintf("away (%.1f, %.1f)", st.WorstX, st.WorstY), e, MathOutput)
}

// traceTLBO populates the math trace for Teaching-Learning-Based Optimization.
func traceTLBO(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.TLBO
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "TLBO"

	phase := 0
	if idx < len(st.Phase) {
		phase = st.Phase[idx]
	}
	if phase == 0 {
		mt.PhaseName = locale.T("math.phase.teacher_phase")
	} else {
		mt.PhaseName = locale.T("math.phase.learner_phase")
	}

	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.Tick, tlboMaxTicks), float64(st.Tick)/float64(tlboMaxTicks), MathInput)
	mt.AddStep("teacher", "(tx, ty)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.BestX, st.BestY, st.BestF), st.BestF, MathInput)
	mt.AddStep("mean", "(mx, my)", fmt.Sprintf("(%.1f, %.1f)", st.MeanX, st.MeanY), st.MeanX, MathInput)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	if phase == 0 {
		// Teacher phase: TF = round(1 + rand) -> 1 or 2
		tf := 1.0 // representative
		diffX := st.BestX - tf*st.MeanX
		mt.AddStep("TF", "round(1+rand)", "1 or 2", tf, MathBranch)
		mt.AddStep("diff_x", "teacher - TF*mean", fmt.Sprintf("%.1f - %.0f*%.1f", st.BestX, tf, st.MeanX), diffX, MathIntermediate)
		mt.AddStep("new_x", "x + r*diff", fmt.Sprintf("%.1f + r*%.1f", bot.X, diffX), bot.X+0.5*diffX, MathOutput)
	} else {
		// Learner phase: compare with random peer
		peerIdx := 0
		if idx < len(st.PeerIdx) {
			peerIdx = st.PeerIdx[idx]
		}
		mt.AddStep("peer", "random peer", fmt.Sprintf("bot %d", peerIdx), float64(peerIdx), MathBranch)
		mt.AddStep("compare", "f(self) vs f(peer)", fmt.Sprintf("%.2f vs ?", fitness), fitness, MathIntermediate)
		mt.AddStep("update", "move toward/away", "x +/- r*(x-peer)", 0, MathOutput)
	}
	mt.AddStep("globalBest", "persistent", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}

// traceEO populates the math trace for the Equilibrium Optimizer.
func traceEO(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.EO
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Equilibrium Optimizer (EO)"

	phase := 0
	if idx < len(st.Phase) {
		phase = st.Phase[idx]
	}
	if phase == 0 {
		mt.PhaseName = locale.T("math.phase.equilibrium")
	} else {
		mt.PhaseName = locale.T("math.phase.equilibrium")
	}

	progress := float64(st.CycleTick) / float64(eoMaxTicks)
	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.CycleTick, eoMaxTicks), progress, MathInput)

	// Equilibrium pool (up to 4 best)
	poolSize := len(st.PoolF)
	if poolSize > 4 {
		poolSize = 4
	}
	for p := 0; p < poolSize; p++ {
		label := fmt.Sprintf("pool_%d", p+1)
		mt.AddStep(label, "eq. candidate", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.PoolX[p], st.PoolY[p], st.PoolF[p]), st.PoolF[p], MathInput)
	}

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)

	// F decay factor
	t := progress
	F := math.Exp(-4.0 * t)
	mt.AddStep("F_decay", "exp(-4t)", fmt.Sprintf("exp(-4*%.3f)", t), F, MathIntermediate)
	mt.AddStep("G_rate", "gen. rate", "G0*F*(eq-x)", 0, MathIntermediate)
	mt.AddStep("bestFit", "global best", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.BestFit, st.BestX, st.BestY), st.BestFit, MathOutput)
}

// traceJaya populates the math trace for the Jaya Algorithm.
func traceJaya(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.Jaya
	if st == nil || ss.MathTrace == nil {
		return
	}
	mt := ss.MathTrace
	mt.AlgoName = "Jaya Algorithm"
	mt.PhaseName = locale.T("math.phase.toward_best_away_worst")

	progress := float64(st.Tick) / float64(jayaMaxTicks)
	mt.AddStep("progress", "tick / maxTicks", fmt.Sprintf("%d / %d", st.Tick, jayaMaxTicks), progress, MathInput)

	fitness := 0.0
	if idx < len(st.Fitness) {
		fitness = st.Fitness[idx]
	}
	mt.AddStep("fitness", "f(x, y)", fmt.Sprintf("f(%.1f, %.1f)", bot.X, bot.Y), fitness, MathInput)
	mt.AddStep("best", "(bx, by)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.BestX, st.BestY, st.BestF), st.BestF, MathInput)
	mt.AddStep("worst", "(wx, wy)", fmt.Sprintf("(%.1f, %.1f) f=%.2f", st.WorstX, st.WorstY, st.WorstF), st.WorstF, MathInput)

	// X_new = X + r1*(best - |X|) - r2*(worst - |X|)
	r1, r2 := 0.5, 0.5 // representative
	toBest := r1 * (st.BestX - math.Abs(bot.X))
	fromWorst := r2 * (st.WorstX - math.Abs(bot.X))
	mt.AddStep("r1", "rand [0,1]", "~0.5", r1, MathIntermediate)
	mt.AddStep("r2", "rand [0,1]", "~0.5", r2, MathIntermediate)
	mt.AddStep("toBest_x", "r1*(best-|x|)", fmt.Sprintf("0.5*(%.1f-|%.1f|)", st.BestX, bot.X), toBest, MathIntermediate)
	mt.AddStep("fromWorst", "r2*(worst-|x|)", fmt.Sprintf("0.5*(%.1f-|%.1f|)", st.WorstX, bot.X), fromWorst, MathIntermediate)
	newX := bot.X + toBest - fromWorst
	mt.AddStep("new_x", "x + to - from", fmt.Sprintf("%.1f + %.2f - %.2f", bot.X, toBest, fromWorst), newX, MathOutput)

	mt.AddStep("globalBest", "persistent", fmt.Sprintf("%.2f @ (%.1f, %.1f)", st.GlobalBestF, st.GlobalBestX, st.GlobalBestY), st.GlobalBestF, MathOutput)
}
