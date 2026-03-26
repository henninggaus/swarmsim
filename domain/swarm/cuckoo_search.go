package swarm

import "math"

// Cuckoo Search Algorithm (CS): Meta-heuristic inspired by the brood
// parasitism of cuckoo birds combined with Lévy flight exploration.
// Each bot represents a "nest" with a solution (position). New solutions
// are generated via Lévy flights. If a cuckoo's egg (new solution) is
// better than a host nest, it replaces it. A fraction of the worst nests
// are abandoned each cycle and rebuilt at random positions.
//
// The algorithm balances exploration (Lévy flights produce occasional
// long jumps) with exploitation (keeping good solutions and replacing
// bad ones). This makes it particularly effective for multi-modal
// optimization landscapes.
//
// Three sensor values are exposed to SwarmScript:
//   - CuckooFitness: current nest quality (0-100)
//   - CuckooNestAge: ticks since last nest rebuild (0-100 capped)
//   - CuckooBest:    1 if this nest is in the top 25%, 0 otherwise
//
// Reference: Yang, X.-S. & Deb, S. (2009)
//
//	"Cuckoo Search via Lévy Flights", Proc. World Congress
//	on Nature & Biologically Inspired Computing (NaBIC).

const (
	csDiscoveryProb = 0.05  // probability of worst nests being abandoned (pa), reduced from 0.10
	csLevyAlpha     = 1.5   // Lévy exponent (1 < alpha <= 2)
	csStepScale     = 0.5   // scaling factor for Lévy step size
	csTopFraction   = 0.25  // fraction considered "best" nests
	csSpeedMult     = 5.0   // movement speed multiplier (7.5 px/tick)
	csMaxTicks      = 3000  // full benchmark length

	// Grid rescan parameters
	csGridRescanRate = 150 // periodic grid rescan every N ticks
	csGridRescanSize = 18  // grid resolution (18×18 = 324 samples)
	csGridInjectTop  = 10  // top grid points injected into worst nests

	// Direct-to-best parameters
	csDirectToBestStart = 0.20 // progress threshold to start direct-to-best
	csDirectToBestMax   = 0.70 // max probability of direct-to-best

	// Global-best attraction
	csGBWeightMin = 0.05 // early-phase GB attraction
	csGBWeightMax = 0.65 // late-phase GB attraction
)

// CuckooState holds the state for Cuckoo Search optimization.
type CuckooState struct {
	Fitness    []float64 // current fitness per bot (nest quality)
	BestX      float64   // current tick best position X
	BestY      float64   // current tick best position Y
	BestF      float64   // current tick best fitness
	BestIdx    int       // index of current tick best nest
	GlobalBestX float64  // persistent global best position X
	GlobalBestY float64  // persistent global best position Y
	GlobalBestF float64  // persistent global best fitness
	NestAge    []int     // ticks since last rebuild per nest
	Tick       int       // current tick counter
	Rankings   []int     // sorted indices by fitness (reused buffer)
	IsDirect   []bool    // per-bot: true if doing direct-to-best this tick
}

// InitCuckoo allocates Cuckoo Search state for all bots.
func InitCuckoo(ss *SwarmState) {
	n := len(ss.Bots)
	ss.Cuckoo = &CuckooState{
		Fitness:    make([]float64, n),
		NestAge:    make([]int, n),
		Rankings:   make([]int, n),
		IsDirect:   make([]bool, n),
		BestIdx:    -1,
		BestF:      -1e9,
		GlobalBestF: -1e18,
	}
	ss.CuckooOn = true
}

// ClearCuckoo frees Cuckoo Search state.
func ClearCuckoo(ss *SwarmState) {
	ss.Cuckoo = nil
	ss.CuckooOn = false
}

// TickCuckoo updates the Cuckoo Search algorithm for all bots.
// Computes fitness, updates global best, abandons worst nests,
// runs periodic grid rescan, and writes sensor cache values.
func TickCuckoo(ss *SwarmState) {
	if ss.Cuckoo == nil {
		return
	}
	st := ss.Cuckoo
	n := len(ss.Bots)

	// Grow slices if bots were added after init
	for len(st.Fitness) < n {
		st.Fitness = append(st.Fitness, 0)
		st.NestAge = append(st.NestAge, 0)
		st.Rankings = append(st.Rankings, len(st.Rankings))
		st.IsDirect = append(st.IsDirect, false)
	}

	st.Tick++

	// Evaluate fitness
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		st.Fitness[i] = cuckooFitness(bot, ss)
		st.NestAge[i]++
	}

	// Find current tick best
	st.BestIdx = 0
	st.BestF = st.Fitness[0]
	for i := 1; i < n; i++ {
		if st.Fitness[i] > st.BestF {
			st.BestF = st.Fitness[i]
			st.BestIdx = i
		}
	}
	st.BestX = ss.Bots[st.BestIdx].X
	st.BestY = ss.Bots[st.BestIdx].Y

	// Update persistent global best
	if st.BestF > st.GlobalBestF {
		st.GlobalBestF = st.BestF
		st.GlobalBestX = st.BestX
		st.GlobalBestY = st.BestY
	}

	// Periodic grid rescan — systematic landscape sampling
	if st.Tick > 0 && st.Tick%csGridRescanRate == 0 && n > 0 {
		csGridRescan(ss, st)
	}

	// Rank nests by fitness (simple insertion sort — fine for n <= 500)
	for i := 0; i < n; i++ {
		st.Rankings[i] = i
	}
	for i := 1; i < n; i++ {
		key := st.Rankings[i]
		j := i - 1
		for j >= 0 && st.Fitness[st.Rankings[j]] > st.Fitness[key] {
			st.Rankings[j+1] = st.Rankings[j]
			j--
		}
		st.Rankings[j+1] = key
	}

	// Abandon worst nests (discovery probability pa)
	abandonCount := int(float64(n) * csDiscoveryProb)
	if abandonCount < 1 {
		abandonCount = 1
	}
	for k := 0; k < abandonCount && k < n; k++ {
		idx := st.Rankings[k]
		ss.Bots[idx].X = SpawnAreaMargin + ss.Rng.Float64()*(ss.ArenaW-2*SpawnAreaMargin)
		ss.Bots[idx].Y = SpawnAreaMargin + ss.Rng.Float64()*(ss.ArenaH-2*SpawnAreaMargin)
		ss.Bots[idx].Angle = ss.Rng.Float64() * 2 * math.Pi
		st.NestAge[idx] = 0
	}

	// Determine direct-to-best flags for ApplyCuckoo
	progress := float64(st.Tick) / float64(csMaxTicks)
	for i := 0; i < n; i++ {
		st.IsDirect[i] = false
	}
	if progress > csDirectToBestStart && st.GlobalBestF > -1e18 {
		dtbProb := csDirectToBestMax * (progress - csDirectToBestStart) / (1.0 - csDirectToBestStart)
		if dtbProb > csDirectToBestMax {
			dtbProb = csDirectToBestMax
		}
		for i := 0; i < n; i++ {
			if i != st.BestIdx && ss.Rng.Float64() < dtbProb {
				st.IsDirect[i] = true
			}
		}
	}

	// Determine top fraction threshold
	topN := int(float64(n) * csTopFraction)
	if topN < 1 {
		topN = 1
	}
	topThreshold := -1e9
	if n-topN >= 0 && n-topN < n {
		topThreshold = st.Fitness[st.Rankings[n-topN]]
	}

	// Update sensor cache for SwarmScript
	for i := range ss.Bots {
		fitNorm := (st.Fitness[i] + 50) / 150.0
		if fitNorm < 0 {
			fitNorm = 0
		}
		if fitNorm > 1 {
			fitNorm = 1
		}
		ss.Bots[i].CuckooFitness = int(fitNorm * 100)

		age := st.NestAge[i]
		if age > 100 {
			age = 100
		}
		ss.Bots[i].CuckooNestAge = age

		if st.Fitness[i] >= topThreshold {
			ss.Bots[i].CuckooBest = 1
		} else {
			ss.Bots[i].CuckooBest = 0
		}
	}
}

// ApplyCuckoo steers a bot using the Cuckoo Search algorithm.
// Good nests move toward the global best via small Lévy steps.
// Direct-to-best bots skip Lévy and go straight to GlobalBest.
// Other nests perform Lévy flights with global-best attraction.
func ApplyCuckoo(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Cuckoo == nil {
		bot.Speed = 0
		return
	}
	st := ss.Cuckoo
	if idx >= len(st.Fitness) {
		bot.Speed = 0
		return
	}

	progress := float64(st.Tick) / float64(csMaxTicks)

	// Global best bot does random walk around GlobalBest (radius 40px)
	if idx == st.BestIdx {
		rx := st.GlobalBestX + (ss.Rng.Float64()-0.5)*80
		ry := st.GlobalBestY + (ss.Rng.Float64()-0.5)*80
		algoMovBot(bot, rx, ry, ss.ArenaW, ss.ArenaH, csSpeedMult)
		// Evaluate and update GlobalBest
		f := cuckooFitness(bot, ss)
		if f > st.GlobalBestF {
			st.GlobalBestF = f
			st.GlobalBestX = bot.X
			st.GlobalBestY = bot.Y
		}
		bot.LEDColor = [3]uint8{255, 215, 0} // gold for best
		return
	}

	// Direct-to-best: skip Lévy, go straight to GlobalBest with jitter
	if st.IsDirect[idx] && st.GlobalBestF > -1e18 {
		jitter := 7.5
		tX := st.GlobalBestX + (ss.Rng.Float64()*2-1)*jitter
		tY := st.GlobalBestY + (ss.Rng.Float64()*2-1)*jitter
		algoMovBot(bot, tX, tY, ss.ArenaW, ss.ArenaH, csSpeedMult)
		// Evaluate and update GlobalBest
		f := cuckooFitness(bot, ss)
		if f > st.GlobalBestF {
			st.GlobalBestF = f
			st.GlobalBestX = bot.X
			st.GlobalBestY = bot.Y
		}
		bot.LEDColor = [3]uint8{0, 255, 0} // green for direct-to-best
		return
	}

	// Lévy flight step toward global best with random perturbation
	step := csStepScale * MantegnaLevy(ss.Rng, csLevyAlpha)

	dx := st.GlobalBestX - bot.X
	dy := st.GlobalBestY - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	var targetX, targetY float64
	if dist > 1.0 {
		baseAngle := math.Atan2(dy, dx)
		desired := baseAngle + step*0.15
		absStep := math.Abs(step)
		stepDist := SwarmBotSpeed * (1.0 + math.Min(absStep*0.5, 1.0)) * csSpeedMult
		targetX = bot.X + math.Cos(desired)*stepDist
		targetY = bot.Y + math.Sin(desired)*stepDist
	} else {
		angle := ss.Rng.Float64() * 2 * math.Pi
		targetX = bot.X + math.Cos(angle)*SwarmBotSpeed*csSpeedMult
		targetY = bot.Y + math.Sin(angle)*SwarmBotSpeed*csSpeedMult
	}

	// Apply global-best attraction: pull target toward GlobalBest
	if st.GlobalBestF > -1e18 {
		gbWeight := csGBWeightMin + (csGBWeightMax-csGBWeightMin)*progress
		if gbWeight > csGBWeightMax {
			gbWeight = csGBWeightMax
		}
		targetX = targetX*(1-gbWeight) + st.GlobalBestX*gbWeight
		targetY = targetY*(1-gbWeight) + st.GlobalBestY*gbWeight
	}

	algoMovBot(bot, targetX, targetY, ss.ArenaW, ss.ArenaH, csSpeedMult)

	// LED color: green gradient by fitness, blue tint for older nests
	fitNorm := (st.Fitness[idx] + 50) / 150.0
	if fitNorm < 0 {
		fitNorm = 0
	}
	if fitNorm > 1 {
		fitNorm = 1
	}
	g := uint8(80 + fitNorm*175)
	b := uint8(math.Min(float64(st.NestAge[idx])*2.5, 200))
	bot.LEDColor = [3]uint8{30, g, b}
}

// csGridRescan evaluates a grid of points across the arena and teleports
// the worst nests to the best-discovered grid positions. Critical for
// deceptive landscapes like Schwefel.
func csGridRescan(ss *SwarmState, st *CuckooState) {
	margin := 10.0
	usableW := ss.ArenaW - 2*margin
	usableH := ss.ArenaH - 2*margin
	n := len(ss.Bots)

	type gridPt struct {
		x, y, f float64
	}
	gridPts := make([]gridPt, 0, csGridRescanSize*csGridRescanSize)
	for gx := 0; gx < csGridRescanSize; gx++ {
		for gy := 0; gy < csGridRescanSize; gy++ {
			x := margin + usableW*(float64(gx)+0.5)/float64(csGridRescanSize)
			y := margin + usableH*(float64(gy)+0.5)/float64(csGridRescanSize)
			x += (ss.Rng.Float64()*2.0 - 1.0) * usableW * 0.02
			y += (ss.Rng.Float64()*2.0 - 1.0) * usableH * 0.02
			f := distanceFitnessPt(ss, x, y)
			gridPts = append(gridPts, gridPt{x, y, f})
		}
	}

	// Sort grid points by fitness descending
	for i := 0; i < len(gridPts)-1; i++ {
		for j := i + 1; j < len(gridPts); j++ {
			if gridPts[j].f > gridPts[i].f {
				gridPts[i], gridPts[j] = gridPts[j], gridPts[i]
			}
		}
	}

	// Update GlobalBest from grid findings
	if len(gridPts) > 0 && gridPts[0].f > st.GlobalBestF {
		st.GlobalBestF = gridPts[0].f
		st.GlobalBestX = gridPts[0].x
		st.GlobalBestY = gridPts[0].y
	}

	// Find worst nests by fitness
	type idxFit struct {
		idx int
		f   float64
	}
	nests := make([]idxFit, n)
	for i := range ss.Bots {
		nests[i] = idxFit{i, st.Fitness[i]}
	}
	for i := 0; i < len(nests)-1; i++ {
		for j := i + 1; j < len(nests); j++ {
			if nests[j].f < nests[i].f {
				nests[i], nests[j] = nests[j], nests[i]
			}
		}
	}

	// Teleport worst nests to best grid points
	inject := csGridInjectTop
	if inject > len(gridPts) {
		inject = len(gridPts)
	}
	if inject > n {
		inject = n
	}
	for i := 0; i < inject; i++ {
		idx := nests[i].idx
		jitter := 5.0
		ss.Bots[idx].X = gridPts[i].x + (ss.Rng.Float64()*2-1)*jitter
		ss.Bots[idx].Y = gridPts[i].y + (ss.Rng.Float64()*2-1)*jitter
		// Clamp to arena
		if ss.Bots[idx].X < SpawnAreaMargin {
			ss.Bots[idx].X = SpawnAreaMargin
		}
		if ss.Bots[idx].X > ss.ArenaW-SpawnAreaMargin {
			ss.Bots[idx].X = ss.ArenaW - SpawnAreaMargin
		}
		if ss.Bots[idx].Y < SpawnAreaMargin {
			ss.Bots[idx].Y = SpawnAreaMargin
		}
		if ss.Bots[idx].Y > ss.ArenaH-SpawnAreaMargin {
			ss.Bots[idx].Y = ss.ArenaH - SpawnAreaMargin
		}
		st.NestAge[idx] = 0
	}
}

// cuckooFitness evaluates the fitness of a nest position using the shared
// Gaussian fitness landscape plus a small neighbor density bonus.
func cuckooFitness(bot *SwarmBot, ss *SwarmState) float64 {
	proxFit := distanceFitness(bot, ss)
	neighFit := math.Min(float64(bot.NeighborCount)/6.0, 1.0) * 20
	return proxFit + neighFit
}
