package swarm

import "math"

// Gravitational Search Algorithm (GSA): Physics-inspired metaheuristic where
// agents are masses that interact via Newtonian gravity. Fitter agents have
// higher mass and attract weaker agents more strongly. Over time, the
// gravitational constant G decays, transitioning from exploration (strong
// long-range forces) to exploitation (weak, local-only forces).
//
// Key mechanics:
//   - Mass is proportional to fitness (distance to target)
//   - Gravitational force F = G * Mi * Mj / (r + eps)
//   - G decays exponentially: G(t) = G0 * exp(-alpha * t / T)
//   - Only the K-best (heaviest) agents exert force (Kbest decreases over time)
//   - Acceleration = sum(forces) / mass_i
//
// Reference: Rashedi, E., Nezamabadi-pour, H. & Saryazdi, S. (2009)
//            "GSA: A Gravitational Search Algorithm", Information Sciences.

const (
	gsaG0          = 100.0  // initial gravitational constant
	gsaAlphaDecay  = 20.0   // G decay rate
	gsaMaxTicks    = 600    // full cycle length
	gsaSteerRate   = 0.2    // max steering change per tick (radians)
	gsaEps         = 1.0    // softening constant to avoid division by zero
	gsaMaxAccel    = 3.0    // acceleration clamp
)

// GSAState holds Gravitational Search Algorithm state for the swarm.
type GSAState struct {
	Mass    []float64 // normalised mass per bot (0-1)
	Fitness []float64 // raw fitness per bot (higher = better)
	AccX    []float64 // acceleration X per bot
	AccY    []float64 // acceleration Y per bot
	Tick    int       // current tick in cycle
	BestIdx int       // index of heaviest agent
	BestX   float64   // position of heaviest agent
	BestY   float64
	G       float64   // current gravitational constant
	sortBuf []float64 // reusable buffer for K-best partial sort (avoids per-tick allocation)
	kBest   []bool    // reusable boolean mask for K-best agents
}

// InitGSA allocates Gravitational Search Algorithm state.
func InitGSA(ss *SwarmState) {
	n := len(ss.Bots)
	ss.GSA = &GSAState{
		Mass:    make([]float64, n),
		Fitness: make([]float64, n),
		AccX:    make([]float64, n),
		AccY:    make([]float64, n),
		G:       gsaG0,
	}
	ss.GSAOn = true
}

// ClearGSA frees Gravitational Search Algorithm state.
func ClearGSA(ss *SwarmState) {
	ss.GSA = nil
	ss.GSAOn = false
}

// gsaMinSearchRadius is the minimum spatial hash query radius.
// Even late in the cycle (low G), we search at least this far.
const gsaMinSearchRadius = 100.0

// TickGSA updates the Gravitational Search Algorithm for all bots.
// Computes fitness, normalises masses, decays G, computes gravitational forces.
//
// Performance: Uses the spatial hash for force computation (O(n·k) where k is
// the average number of nearby agents) instead of brute-force O(n²). The search
// radius shrinks with the gravitational constant G: early in the cycle (large G,
// exploration) forces reach far, late in the cycle (small G, exploitation) only
// nearby agents matter. Falls back to brute force if Hash is nil.
func TickGSA(ss *SwarmState) {
	if ss.GSA == nil {
		return
	}
	st := ss.GSA
	n := len(ss.Bots)

	// Grow slices if bots were added
	for len(st.Mass) < n {
		st.Mass = append(st.Mass, 0)
		st.Fitness = append(st.Fitness, 0)
		st.AccX = append(st.AccX, 0)
		st.AccY = append(st.AccY, 0)
	}
	// Grow reusable buffers
	if cap(st.sortBuf) < n {
		st.sortBuf = make([]float64, n)
		st.kBest = make([]bool, n)
	}
	st.sortBuf = st.sortBuf[:n]
	st.kBest = st.kBest[:n]

	st.Tick++
	if st.Tick > gsaMaxTicks {
		st.Tick = 1
	}

	// Decay gravitational constant: G(t) = G0 * exp(-alpha * t / T)
	st.G = gsaG0 * math.Exp(-gsaAlphaDecay*float64(st.Tick)/float64(gsaMaxTicks))

	// Compute fitness using shared Gaussian landscape
	bestFit := -math.MaxFloat64
	worstFit := math.MaxFloat64

	for i := range ss.Bots {
		st.Fitness[i] = distanceFitness(&ss.Bots[i], ss)
		if st.Fitness[i] > bestFit {
			bestFit = st.Fitness[i]
			st.BestIdx = i
		}
		if st.Fitness[i] < worstFit {
			worstFit = st.Fitness[i]
		}
	}

	st.BestX = ss.Bots[st.BestIdx].X
	st.BestY = ss.Bots[st.BestIdx].Y

	// Normalise masses: m_i = (fit_i - worst) / (best - worst + eps)
	fitRange := bestFit - worstFit + gsaEps
	totalMass := 0.0
	for i := range ss.Bots {
		st.Mass[i] = (st.Fitness[i] - worstFit) / fitRange
		totalMass += st.Mass[i]
	}
	// Normalise so masses sum to 1
	if totalMass > 0 {
		for i := range st.Mass {
			st.Mass[i] /= totalMass
		}
	}

	// K-best: only the top K agents exert gravitational force.
	// K decreases linearly from N to 1 over the cycle.
	kCount := n - int(float64(n-1)*float64(st.Tick)/float64(gsaMaxTicks))
	if kCount < 1 {
		kCount = 1
	}

	// Find K-best using reusable buffer (partial selection sort, no allocation)
	copy(st.sortBuf, st.Mass[:n])
	for i := 0; i < kCount && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < n; j++ {
			if st.sortBuf[j] > st.sortBuf[maxIdx] {
				maxIdx = j
			}
		}
		st.sortBuf[i], st.sortBuf[maxIdx] = st.sortBuf[maxIdx], st.sortBuf[i]
	}
	massThreshold := 0.0
	if kCount < n {
		massThreshold = st.sortBuf[kCount-1]
	}

	// Build boolean K-best mask for O(1) lookup during force computation
	for i := 0; i < n; i++ {
		st.kBest[i] = st.Mass[i] >= massThreshold
	}

	// Compute gravitational acceleration for each bot
	for i := range ss.Bots {
		st.AccX[i] = 0
		st.AccY[i] = 0
	}

	// Search radius proportional to G: large early (exploration), small late (exploitation).
	// Arena diagonal is the maximum useful distance.
	arenaDiag := math.Sqrt(ss.ArenaW*ss.ArenaW + ss.ArenaH*ss.ArenaH)
	gRatio := st.G / gsaG0 // 1.0 → 0.0 over the cycle
	searchRadius := gsaMinSearchRadius + (arenaDiag-gsaMinSearchRadius)*math.Sqrt(gRatio)

	useSpatialHash := ss.Hash != nil && searchRadius < arenaDiag*0.9

	for i := range ss.Bots {
		if useSpatialHash {
			// Spatial hash query: only check nearby candidates
			candidates := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, searchRadius)
			for _, j := range candidates {
				if j == i || !st.kBest[j] {
					continue
				}
				dx := ss.Bots[j].X - ss.Bots[i].X
				dy := ss.Bots[j].Y - ss.Bots[i].Y
				r := math.Sqrt(dx*dx + dy*dy)
				force := st.G * st.Mass[i] * st.Mass[j] / (r + gsaEps)
				randJ := ss.Rng.Float64()
				if r > 0 {
					st.AccX[i] += randJ * force * dx / r
					st.AccY[i] += randJ * force * dy / r
				}
			}
		} else {
			// Brute-force fallback (Hash nil or radius covers whole arena)
			for j := range ss.Bots {
				if j == i || !st.kBest[j] {
					continue
				}
				dx := ss.Bots[j].X - ss.Bots[i].X
				dy := ss.Bots[j].Y - ss.Bots[i].Y
				r := math.Sqrt(dx*dx + dy*dy)
				force := st.G * st.Mass[i] * st.Mass[j] / (r + gsaEps)
				randJ := ss.Rng.Float64()
				if r > 0 {
					st.AccX[i] += randJ * force * dx / r
					st.AccY[i] += randJ * force * dy / r
				}
			}
		}

		// a_i = F_i / M_i (avoid division by zero for zero-mass agents)
		if st.Mass[i] > 1e-10 {
			st.AccX[i] /= st.Mass[i]
			st.AccY[i] /= st.Mass[i]
		}

		// Clamp acceleration
		accMag := math.Sqrt(st.AccX[i]*st.AccX[i] + st.AccY[i]*st.AccY[i])
		if accMag > gsaMaxAccel {
			scale := gsaMaxAccel / accMag
			st.AccX[i] *= scale
			st.AccY[i] *= scale
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].GSAMass = int(st.Mass[i] * 1000) // mass * 1000 for integer sensor
		ss.Bots[i].GSAForce = int(math.Sqrt(st.AccX[i]*st.AccX[i]+st.AccY[i]*st.AccY[i]) * 100)
		dx := st.BestX - ss.Bots[i].X
		dy := st.BestY - ss.Bots[i].Y
		ss.Bots[i].GSABestDist = int(math.Sqrt(dx*dx + dy*dy))
	}
}

// ApplyGSA steers a bot according to gravitational acceleration.
func ApplyGSA(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.GSA == nil || idx >= len(ss.GSA.AccX) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.GSA

	ax, ay := st.AccX[idx], st.AccY[idx]
	if ax != 0 || ay != 0 {
		desired := math.Atan2(ay, ax)
		steerToward(bot, desired, gsaSteerRate)
	}

	// Speed proportional to acceleration magnitude
	accMag := math.Sqrt(ax*ax + ay*ay)
	bot.Speed = SwarmBotSpeed * (0.5 + math.Min(accMag, 2.0)/2.0)

	// LED: mass as color intensity (heavy = bright red, light = dim blue)
	mass01 := st.Mass[idx] * float64(len(ss.Bots)) // denormalise
	if mass01 > 1 {
		mass01 = 1
	}
	r := uint8(mass01 * 255)
	b := uint8((1 - mass01) * 180)
	// Best agent gets gold
	if idx == st.BestIdx {
		bot.LEDColor = [3]uint8{255, 215, 0}
	} else {
		bot.LEDColor = [3]uint8{r, 40, b}
	}
}
