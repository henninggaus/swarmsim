package swarm

import "math"

// FitnessLandscapeType selects which fitness function the optimisation
// algorithms use. Cycling through them lets users compare convergence
// behaviour on different optimisation surfaces.
type FitnessLandscapeType int

const (
	FitGaussian   FitnessLandscapeType = iota // random multi-modal Gaussian peaks (default)
	FitRastrigin                              // Rastrigin function (highly multi-modal)
	FitAckley                                 // Ackley function (nearly flat + sharp global minimum)
	FitRosenbrock                             // Rosenbrock banana function (narrow curved valley)
	FitSchwefel                               // Schwefel function (deceptive global minimum)
	FitGriewank                               // Griewank function (many shallow local minima)
	FitLevy                                   // Levy function (complex multimodal landscape)
	FitCount
)

// FitnessLandscapeName returns the display name for a fitness function type.
func FitnessLandscapeName(ft FitnessLandscapeType) string {
	switch ft {
	case FitGaussian:
		return "Gaussian Peaks"
	case FitRastrigin:
		return "Rastrigin"
	case FitAckley:
		return "Ackley"
	case FitRosenbrock:
		return "Rosenbrock"
	case FitSchwefel:
		return "Schwefel"
	case FitGriewank:
		return "Griewank"
	case FitLevy:
		return "Lévy"
	default:
		return "Unknown"
	}
}

// SwarmAlgorithmType identifies a classic swarm intelligence algorithm.
type SwarmAlgorithmType int

const (
	AlgoNone     SwarmAlgorithmType = iota
	AlgoBoids                       // Craig Reynolds' Boids (1986)
	AlgoPSO                         // Particle Swarm Optimization
	AlgoACO                         // Ant Colony Optimization
	AlgoFirefly                     // Firefly Algorithm
	AlgoGWO                         // Grey Wolf Optimizer (Mirjalili 2014)
	AlgoWOA                         // Whale Optimization Algorithm (Mirjalili 2016)
	AlgoBFO                         // Bacterial Foraging Optimization (Passino 2002)
	AlgoMFO                         // Moth-Flame Optimization (Mirjalili 2015)
	AlgoCuckoo                      // Cuckoo Search (Yang & Deb 2009)
	AlgoDE                          // Differential Evolution (Storn & Price 1997)
	AlgoABC                         // Artificial Bee Colony (Karaboga 2005)
	AlgoHSO                         // Harmony Search Optimization (Geem 2001)
	AlgoBat                         // Bat Algorithm (Yang 2010)
	AlgoSSA                         // Salp Swarm Algorithm (Mirjalili 2017)
	AlgoGSA                         // Gravitational Search Algorithm (Rashedi 2009)
	AlgoFPA                         // Flower Pollination Algorithm (Yang 2012)
	AlgoHHO                         // Harris Hawks Optimization (Heidari 2019)
	AlgoSA                          // Simulated Annealing (Kirkpatrick 1983)
	AlgoAO                          // Aquila Optimizer (Abualigah 2021)
	AlgoSCA                         // Sine Cosine Algorithm (Mirjalili 2016)
	AlgoDA                          // Dragonfly Algorithm (Mirjalili 2016)
	AlgoTLBO                        // Teaching-Learning-Based Optimization (Rao 2011)
	AlgoEO                          // Equilibrium Optimizer (Faramarzi 2020)
	AlgoJaya                        // Jaya Algorithm (Rao 2016)
	AlgoCount
)

// AlgoPerformanceRecord stores the final performance of an algorithm run
// on a specific fitness landscape. Used by the scoreboard to compare algorithms.
type AlgoPerformanceRecord struct {
	Algo        SwarmAlgorithmType
	FitnessFunc FitnessLandscapeType
	BestFitness float64 // best fitness achieved
	Iterations  int     // total convergence samples
	Perturbations int   // number of stagnation perturbations triggered

	// Extended metrics for radar chart comparison
	ConvergenceSpeed float64 // iterations to reach 90% of final best (lower = faster), 0 if N/A
	FinalDiversity   float64 // population spatial diversity at recording time (0-1)
	AvgFitness       float64 // average fitness of the population at recording time
}

// ConvergenceArchiveEntry stores a snapshot of one algorithm's convergence
// curve so it can be overlaid on the graph after switching to another algorithm.
// This enables visual comparison of convergence speed across algorithms.
type ConvergenceArchiveEntry struct {
	Algo        SwarmAlgorithmType
	FitnessFunc FitnessLandscapeType
	BestHistory []float64 // sampled best fitness over time
}

// convergenceArchiveMax is the maximum number of archived convergence curves.
// Oldest entries are evicted when the limit is reached.
const convergenceArchiveMax = 8

// SwarmAlgorithmState holds the state for classic swarm algorithms.
type SwarmAlgorithmState struct {
	ActiveAlgo  SwarmAlgorithmType
	FitnessFunc FitnessLandscapeType // which benchmark function to use (default: FitGaussian)

	// Boids parameters
	BoidsSeparationDist float64 // minimum distance (default 15)
	BoidsAlignmentDist  float64 // alignment range (default 50)
	BoidsCohesionDist   float64 // cohesion range (default 80)
	BoidsSepWeight      float64 // separation weight (default 1.5)
	BoidsAlignWeight    float64 // alignment weight (default 1.0)
	BoidsCohWeight      float64 // cohesion weight (default 1.0)
	BoidsMaxSpeed       float64 // max speed (default 2.0)
	BoidsMaxTurn        float64 // max turn per tick in radians (default 0.2)

	// PSO parameters
	PSOGlobalBestX float64 // global best position
	PSOGlobalBestY float64
	PSOGlobalBestF float64 // global best fitness
	PSOInertia     float64 // inertia weight (default 0.7)
	PSOCognitive   float64 // cognitive coefficient (default 1.5)
	PSOSocial      float64 // social coefficient (default 1.5)
	PSOPersonalBest []PSOParticle // per-bot personal best

	// ACO parameters
	ACOPheromoneDeposit float64 // amount deposited per step (default 1.0)
	ACOEvaporation      float64 // evaporation rate per tick (default 0.01)
	ACOAlpha            float64 // pheromone influence (default 1.0)
	ACOBeta             float64 // distance influence (default 2.0)
	ACOGrid             []float64 // pheromone grid
	ACOGridCols         int
	ACOGridRows         int
	ACOCellSize         float64

	// Firefly parameters
	FireflyBeta0      float64   // base attractiveness (default 1.0)
	FireflyGamma      float64   // light absorption (default 0.01)
	FireflyAlpha      float64   // randomization parameter (default 0.5)
	FireflyAlpha0     float64   // initial alpha for decay tracking (default 0.5)
	FireflyBrightness []float64 // per-bot brightness (fitness-based)
	FireflyBestX      float64   // global best X
	FireflyBestY      float64   // global best Y
	FireflyBestF      float64   // global best fitness
	FireflyBestIdx    int       // index of best bot (-1 = none)
	FireflyCycleTick  int       // cycle tick for exploration ratio

	// Shared Gaussian fitness landscape (used by all optimization algorithms)
	// Generated once when any optimization algorithm starts, persisted across
	// algorithm switches so comparisons are on the same landscape.
	FitPeakX []float64 // peak center X coordinates
	FitPeakY []float64 // peak center Y coordinates
	FitPeakH []float64 // peak heights (strength)
	FitPeakS []float64 // peak sigmas (spread)

	// Dynamic fitness landscape: peaks drift over time (Gaussian only).
	// Toggle with Ctrl+D. Velocities are initialized when dynamic mode is
	// first enabled and are stored per-peak.
	DynamicLandscape bool      // whether peaks are moving
	DynVelX          []float64 // horizontal velocity per peak (px/tick)
	DynVelY          []float64 // vertical velocity per peak (px/tick)
	DynVelH          []float64 // height change rate per peak (units/tick)
	DynVelS          []float64 // sigma change rate per peak (units/tick)

	// Convergence tracking (shared across all algorithms)
	ConvergenceHistory     []float64 // sampled best fitness over time
	ConvergenceAvg         []float64 // sampled average fitness over time
	ConvergenceDiversity   []float64 // sampled population spatial diversity over time
	ConvergenceExploration []float64 // sampled exploration ratio (0-100%) over time
	ConvergenceTick        int       // tick counter for sampling interval

	// Search trajectory: X,Y path of the global best solution over time.
	// Recorded alongside convergence samples, visualised as a spatial inset.
	TrajectoryX []float64 // global best X at each convergence sample
	TrajectoryY []float64 // global best Y at each convergence sample

	// Stagnation detection
	StagnationCount   int     // samples since last best fitness improvement
	BestFitnessEver   float64 // highest best fitness observed
	TotalIterations   int     // total number of convergence samples taken
	PerturbationCount int     // how many times auto-perturbation was applied
}

// PSOParticle stores personal best for one PSO particle.
type PSOParticle struct {
	BestX, BestY float64
	BestFitness  float64
	VelX, VelY   float64
}

// SwarmAlgorithmName returns the display name of an algorithm.
func SwarmAlgorithmName(algo SwarmAlgorithmType) string {
	switch algo {
	case AlgoBoids:
		return "Boids (Reynolds)"
	case AlgoPSO:
		return "Particle Swarm (PSO)"
	case AlgoACO:
		return "Ant Colony (ACO)"
	case AlgoFirefly:
		return "Firefly"
	case AlgoGWO:
		return "Grey Wolf (GWO)"
	case AlgoWOA:
		return "Whale (WOA)"
	case AlgoBFO:
		return "Bacterial Foraging (BFO)"
	case AlgoMFO:
		return "Moth-Flame (MFO)"
	case AlgoCuckoo:
		return "Cuckoo Search"
	case AlgoDE:
		return "Differential Evolution (DE)"
	case AlgoABC:
		return "Artificial Bee Colony (ABC)"
	case AlgoHSO:
		return "Harmony Search (HSO)"
	case AlgoBat:
		return "Bat Algorithm (BA)"
	case AlgoSSA:
		return "Salp Swarm (SSA)"
	case AlgoGSA:
		return "Gravitational Search (GSA)"
	case AlgoFPA:
		return "Flower Pollination (FPA)"
	case AlgoHHO:
		return "Harris Hawks (HHO)"
	case AlgoSA:
		return "Simulated Annealing (SA)"
	case AlgoAO:
		return "Aquila Optimizer (AO)"
	case AlgoSCA:
		return "Sine Cosine (SCA)"
	case AlgoDA:
		return "Dragonfly (DA)"
	case AlgoTLBO:
		return "Teaching-Learning (TLBO)"
	case AlgoEO:
		return "Equilibrium Optimizer (EO)"
	case AlgoJaya:
		return "Jaya Algorithm (Rao)"
	default:
		return "Keiner"
	}
}

// SwarmAlgorithmAbbrev returns a very short abbreviation (3-5 chars) suitable
// for compact legends in the convergence graph overlay.
func SwarmAlgorithmAbbrev(algo SwarmAlgorithmType) string {
	switch algo {
	case AlgoBoids:
		return "Boid"
	case AlgoPSO:
		return "PSO"
	case AlgoACO:
		return "ACO"
	case AlgoFirefly:
		return "FF"
	case AlgoGWO:
		return "GWO"
	case AlgoWOA:
		return "WOA"
	case AlgoBFO:
		return "BFO"
	case AlgoMFO:
		return "MFO"
	case AlgoCuckoo:
		return "CS"
	case AlgoDE:
		return "DE"
	case AlgoABC:
		return "ABC"
	case AlgoHSO:
		return "HSO"
	case AlgoBat:
		return "BA"
	case AlgoSSA:
		return "SSA"
	case AlgoGSA:
		return "GSA"
	case AlgoFPA:
		return "FPA"
	case AlgoHHO:
		return "HHO"
	case AlgoSA:
		return "SA"
	case AlgoAO:
		return "AO"
	case AlgoSCA:
		return "SCA"
	case AlgoDA:
		return "DA"
	case AlgoTLBO:
		return "TLBO"
	case AlgoEO:
		return "EO"
	case AlgoJaya:
		return "Jaya"
	default:
		return "?"
	}
}

// InitSwarmAlgorithm initializes a swarm algorithm by type. Lifecycle functions
// (init, clear, tick, apply) are looked up via algoRegistry (see
// algorithm_registry.go) so that adding a new algorithm never requires editing
// these switch-free dispatch paths.
func InitSwarmAlgorithm(ss *SwarmState, algo SwarmAlgorithmType) {
	// Record outgoing algorithm performance before clearing.
	recordAlgoPerformance(ss)

	// Clear any previously active algorithm first to avoid stale state.
	ClearSwarmAlgorithm(ss)

	sa := &SwarmAlgorithmState{
		ActiveAlgo: algo,
		// Boids defaults
		BoidsSeparationDist: 15,
		BoidsAlignmentDist:  50,
		BoidsCohesionDist:   80,
		BoidsSepWeight:      1.5,
		BoidsAlignWeight:    1.0,
		BoidsCohWeight:      1.0,
		BoidsMaxSpeed:       2.0,
		BoidsMaxTurn:        0.2,
		// PSO defaults (kept for reference; PSO now uses dedicated PSOState)
		PSOInertia:   0.7,
		PSOCognitive: 1.5,
		PSOSocial:    1.5,
		// ACO defaults (kept for reference; ACO now uses dedicated ACOState)
		ACOPheromoneDeposit: 1.0,
		ACOEvaporation:      0.01,
		ACOAlpha:            1.0,
		ACOBeta:             2.0,
		ACOCellSize:         20,
		// Firefly defaults
		FireflyBeta0:   1.0,
		FireflyGamma:   0.01,
		FireflyAlpha:   0.5,
		FireflyAlpha0:  0.5,
		FireflyBestIdx: -1,
	}

	// Assign state before calling the init handler so that handlers like
	// Firefly can access ss.SwarmAlgo to set up their fields.
	ss.SwarmAlgo = sa
	ss.SwarmAlgoOn = true

	// Generate shared Gaussian fitness landscape for optimisation algorithms.
	// Boids and ACO do not optimise a fitness function and are excluded.
	if algo != AlgoBoids && algo != AlgoACO {
		initFitnessLandscape(ss)
	}

	if h, ok := algoRegistry[algo]; ok && h.init != nil {
		h.init(ss)
	}
}

// ClearSwarmAlgorithm disables the active swarm algorithm and frees its state.
// It delegates to the algorithm's registered clear handler (see algoRegistry)
// to clean up any dedicated state structs.
func ClearSwarmAlgorithm(ss *SwarmState) {
	if ss.SwarmAlgo != nil {
		if h, ok := algoRegistry[ss.SwarmAlgo.ActiveAlgo]; ok && h.clear != nil {
			h.clear(ss)
		}
	}
	ss.SwarmAlgo = nil
	ss.SwarmAlgoOn = false
}

// TickSwarmAlgorithm runs one tick of the active algorithm. The registered
// handler's tick function runs first (global state update), then, if an apply
// function is registered, it is called for every bot to apply per-bot steering.
// Algorithms like Boids and Firefly integrate per-bot steering into their tick
// function and leave apply nil.
func TickSwarmAlgorithm(ss *SwarmState) {
	if ss.SwarmAlgo == nil {
		return
	}
	// Move Gaussian peaks if dynamic landscape is active.
	TickDynamicLandscape(ss)

	h, ok := algoRegistry[ss.SwarmAlgo.ActiveAlgo]
	if !ok {
		return
	}
	if h.tick != nil {
		h.tick(ss)
	}
	if h.apply != nil {
		for i := range ss.Bots {
			h.apply(&ss.Bots[i], ss, i)
		}
	}
	// Sample convergence data for the real-time graph.
	recordConvergence(ss)
}

// convergenceSampleInterval is the number of ticks between fitness samples
// for the convergence graph. Sampling every 10 ticks keeps memory bounded
// while providing sufficient resolution for visualisation.
const convergenceSampleInterval = 10

// convergenceMaxSamples caps the convergence history length to prevent
// unbounded memory growth in long-running simulations.
const convergenceMaxSamples = 500

// GetAlgoBestFitness returns the current global best fitness of the active
// swarm algorithm. Returns 0 if no algorithm is active or the algorithm
// does not track a global best (e.g. Boids).
func GetAlgoBestFitness(ss *SwarmState) float64 {
	if ss.SwarmAlgo == nil {
		return 0
	}
	if h, ok := algoRegistry[ss.SwarmAlgo.ActiveAlgo]; ok && h.bestFitness != nil {
		return h.bestFitness(ss)
	}
	return 0
}

// GetAlgoBestPos returns the (X, Y) position of the global best solution
// for the active optimization algorithm. The third return value is false if
// no position is available (e.g. no algorithm active or Boids/ACO).
func GetAlgoBestPos(ss *SwarmState) (float64, float64, bool) {
	if ss.SwarmAlgo == nil {
		return 0, 0, false
	}
	if h, ok := algoRegistry[ss.SwarmAlgo.ActiveAlgo]; ok && h.bestPos != nil {
		return h.bestPos(ss)
	}
	return 0, 0, false
}

// GetAlgoFitnessValues returns the per-bot fitness slice for the active
// algorithm, or nil if unavailable. The returned slice is the algorithm's
// internal storage — callers must NOT modify it.
func GetAlgoFitnessValues(ss *SwarmState) []float64 {
	if ss.SwarmAlgo == nil {
		return nil
	}
	if h, ok := algoRegistry[ss.SwarmAlgo.ActiveAlgo]; ok && h.avgFitnessVals != nil {
		return h.avgFitnessVals(ss)
	}
	return nil
}

// GetAlgoAvgFitness returns the average fitness across all bots for the
// active algorithm. Returns 0 if no per-bot fitness array is available.
func GetAlgoAvgFitness(ss *SwarmState) float64 {
	if ss.SwarmAlgo == nil {
		return 0
	}
	var vals []float64
	if h, ok := algoRegistry[ss.SwarmAlgo.ActiveAlgo]; ok && h.avgFitnessVals != nil {
		vals = h.avgFitnessVals(ss)
	}
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// GetAlgoDiversity computes the spatial diversity of the bot population as the
// root-mean-square distance from the swarm centroid, normalised to [0,100].
// High diversity indicates exploration; low diversity indicates convergence.
// Returns 0 if fewer than 2 bots are present.
func GetAlgoDiversity(ss *SwarmState) float64 {
	n := len(ss.Bots)
	if n < 2 {
		return 0
	}
	// Compute centroid
	cx, cy := 0.0, 0.0
	for i := range ss.Bots {
		cx += ss.Bots[i].X
		cy += ss.Bots[i].Y
	}
	cx /= float64(n)
	cy /= float64(n)
	// RMS distance from centroid
	sumSq := 0.0
	for i := range ss.Bots {
		dx := ss.Bots[i].X - cx
		dy := ss.Bots[i].Y - cy
		sumSq += dx*dx + dy*dy
	}
	rms := math.Sqrt(sumSq / float64(n))
	// Normalise: arena diagonal is max possible spread
	diag := math.Sqrt(ss.ArenaW*ss.ArenaW + ss.ArenaH*ss.ArenaH)
	if diag < 1 {
		diag = 1
	}
	return (rms / diag) * 100
}

// GetAlgoExplorationRatio returns the current exploration-exploitation balance
// of the active swarm algorithm as a percentage (0-100). 100 = pure exploration,
// 0 = pure exploitation. The ratio is derived from each algorithm's internal
// phase-control parameter (e.g. GWO's 'a' parameter, SA's temperature, etc.).
// Returns -1 if no algorithm is active or the algorithm has no meaningful ratio.
func GetAlgoExplorationRatio(ss *SwarmState) float64 {
	if ss.SwarmAlgo == nil {
		return -1
	}
	if h, ok := algoRegistry[ss.SwarmAlgo.ActiveAlgo]; ok && h.explorationRatio != nil {
		return h.explorationRatio(ss)
	}
	return -1
}

// StagnationThreshold is the number of convergence samples without best fitness
// improvement before auto-perturbation is triggered. At 10 ticks per sample,
// this equals 500 ticks (~8 seconds at 60fps).
const StagnationThreshold = 50

// perturbFraction is the fraction of the population randomly scattered when
// stagnation is detected. A small fraction preserves good solutions while
// injecting fresh exploration.
const perturbFraction = 0.3

// recordConvergence samples the current best and average fitness into the
// convergence history. Called from TickSwarmAlgorithm every
// convergenceSampleInterval ticks. Also tracks stagnation and triggers
// auto-perturbation when no improvement is detected for stagnationThreshold
// consecutive samples.
func recordConvergence(ss *SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil {
		return
	}
	sa.ConvergenceTick++
	if sa.ConvergenceTick%convergenceSampleInterval != 0 {
		return
	}
	best := GetAlgoBestFitness(ss)
	avg := GetAlgoAvgFitness(ss)
	div := GetAlgoDiversity(ss)
	explRatio := GetAlgoExplorationRatio(ss)
	sa.ConvergenceHistory = append(sa.ConvergenceHistory, best)
	sa.ConvergenceAvg = append(sa.ConvergenceAvg, avg)
	sa.ConvergenceDiversity = append(sa.ConvergenceDiversity, div)
	sa.ConvergenceExploration = append(sa.ConvergenceExploration, explRatio)
	sa.TotalIterations++

	// Record global best position for search trajectory plot
	if bx, by, ok := GetAlgoBestPos(ss); ok {
		sa.TrajectoryX = append(sa.TrajectoryX, bx)
		sa.TrajectoryY = append(sa.TrajectoryY, by)
	} else {
		// No position available — store NaN sentinel so indices stay aligned
		sa.TrajectoryX = append(sa.TrajectoryX, -1)
		sa.TrajectoryY = append(sa.TrajectoryY, -1)
	}

	// Stagnation detection: track if best fitness has improved
	if best > sa.BestFitnessEver+1e-9 {
		sa.BestFitnessEver = best
		sa.StagnationCount = 0
	} else {
		sa.StagnationCount++
	}

	// Auto-perturbation: scatter a fraction of bots when stagnated.
	// Skip for Boids (no fitness optimization) and ACO (grid-based).
	if sa.StagnationCount >= StagnationThreshold &&
		sa.ActiveAlgo != AlgoBoids && sa.ActiveAlgo != AlgoACO {
		applyStagnationPerturbation(ss)
		sa.StagnationCount = 0
		sa.PerturbationCount++
	}

	// Cap length
	if len(sa.ConvergenceHistory) > convergenceMaxSamples {
		sa.ConvergenceHistory = sa.ConvergenceHistory[len(sa.ConvergenceHistory)-convergenceMaxSamples:]
		sa.ConvergenceAvg = sa.ConvergenceAvg[len(sa.ConvergenceAvg)-convergenceMaxSamples:]
		sa.ConvergenceDiversity = sa.ConvergenceDiversity[len(sa.ConvergenceDiversity)-convergenceMaxSamples:]
		sa.ConvergenceExploration = sa.ConvergenceExploration[len(sa.ConvergenceExploration)-convergenceMaxSamples:]
	}
	if len(sa.TrajectoryX) > convergenceMaxSamples {
		sa.TrajectoryX = sa.TrajectoryX[len(sa.TrajectoryX)-convergenceMaxSamples:]
		sa.TrajectoryY = sa.TrajectoryY[len(sa.TrajectoryY)-convergenceMaxSamples:]
	}
}

// applyStagnationPerturbation uses Opposition-Based Learning (OBL) to
// reposition perturbFraction of the bot population when stagnation is
// detected. Instead of random scatter, OBL computes the opposite position
// in the search space: opposite = lb + ub - current. The bot is moved to
// whichever position (opposite or a random candidate) has higher fitness.
// This is proven to escape local optima more effectively than pure random
// perturbation (Tizhoosh 2005, Rahnamayan 2008).
func applyStagnationPerturbation(ss *SwarmState) {
	n := len(ss.Bots)
	numPerturb := int(float64(n) * perturbFraction)
	if numPerturb < 1 {
		numPerturb = 1
	}
	margin := SwarmEdgeMargin
	lb := margin        // lower bound of usable arena
	ubX := ss.ArenaW - margin // upper bound X
	ubY := ss.ArenaH - margin // upper bound Y

	for k := 0; k < numPerturb; k++ {
		idx := ss.Rng.Intn(n)
		bot := &ss.Bots[idx]
		origX, origY := bot.X, bot.Y

		// Opposition-Based Learning: opposite position
		oblX := lb + ubX - origX
		oblY := lb + ubY - origY
		// Clamp to arena bounds
		oblX = math.Max(margin, math.Min(ubX, oblX))
		oblY = math.Max(margin, math.Min(ubY, oblY))

		// Random candidate (fallback diversity)
		randX := margin + ss.Rng.Float64()*(ubX-lb)
		randY := margin + ss.Rng.Float64()*(ubY-lb)

		// Evaluate fitness at all three positions; pick the best non-original
		fitOBL := EvaluateFitnessLandscape(ss.SwarmAlgo, oblX, oblY)
		fitRand := EvaluateFitnessLandscape(ss.SwarmAlgo, randX, randY)

		if fitOBL >= fitRand {
			bot.X = oblX
			bot.Y = oblY
		} else {
			bot.X = randX
			bot.Y = randY
		}
		bot.Angle = ss.Rng.Float64() * 2 * math.Pi
	}
}

// recordAlgoPerformance snapshots the current algorithm's performance into the
// scoreboard on SwarmState. Called before switching algorithms so users can
// compare how different algorithms performed on the same fitness landscape.
// It also archives the convergence curve for visual overlay comparison.
func recordAlgoPerformance(ss *SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil || sa.ActiveAlgo == AlgoNone || sa.ActiveAlgo == AlgoBoids || sa.ActiveAlgo == AlgoACO {
		return
	}
	if sa.TotalIterations < 2 {
		return // not enough data to record
	}
	// Compute convergence speed: iterations to reach 90% of final best fitness.
	convSpeed := 0.0
	if len(sa.ConvergenceHistory) > 1 {
		target := sa.BestFitnessEver * 0.9
		for i, v := range sa.ConvergenceHistory {
			if v >= target {
				convSpeed = float64(i)
				break
			}
		}
		if convSpeed == 0 {
			convSpeed = float64(len(sa.ConvergenceHistory)) // never reached 90%
		}
	}

	// Compute average fitness of the population.
	avgFit := GetAlgoAvgFitness(ss)

	// Compute final diversity (normalised 0-1 from GetAlgoDiversity's 0-100).
	finalDiv := GetAlgoDiversity(ss) / 100.0
	if finalDiv > 1 {
		finalDiv = 1
	}

	rec := AlgoPerformanceRecord{
		Algo:             sa.ActiveAlgo,
		FitnessFunc:      sa.FitnessFunc,
		BestFitness:      sa.BestFitnessEver,
		Iterations:       sa.TotalIterations,
		Perturbations:    sa.PerturbationCount,
		ConvergenceSpeed: convSpeed,
		FinalDiversity:   finalDiv,
		AvgFitness:       avgFit,
	}
	// Replace existing entry for same algo+fitness combo, or append.
	for i, r := range ss.AlgoScoreboard {
		if r.Algo == rec.Algo && r.FitnessFunc == rec.FitnessFunc {
			ss.AlgoScoreboard[i] = rec
			goto archiveConvergence
		}
	}
	ss.AlgoScoreboard = append(ss.AlgoScoreboard, rec)

archiveConvergence:
	// Archive the convergence curve for visual comparison overlay.
	archiveConvergenceCurve(ss)
}

// RecordAlgoPerformanceExported is the exported wrapper around
// recordAlgoPerformance for use by the headless benchmark runner.
func RecordAlgoPerformanceExported(ss *SwarmState) {
	recordAlgoPerformance(ss)
}

// ReinitFitnessLandscape forces re-initialisation of the Gaussian fitness
// peaks. For non-Gaussian landscapes (Rastrigin, Ackley, etc.) this is a
// no-op since those use analytical formulas. Call this after changing
// SwarmAlgo.FitnessFunc to ensure the landscape matches.
func ReinitFitnessLandscape(ss *SwarmState) {
	if ss.SwarmAlgo == nil {
		return
	}
	if ss.SwarmAlgo.FitnessFunc == FitGaussian {
		// Clear peaks to force regeneration
		ss.SwarmAlgo.FitPeakX = nil
		ss.SwarmAlgo.FitPeakY = nil
		ss.SwarmAlgo.FitPeakH = nil
		ss.SwarmAlgo.FitPeakS = nil
		initFitnessLandscape(ss)
	}
	// Non-Gaussian landscapes use analytical formulas; nothing to reinit.
}

// archiveConvergenceCurve copies the current algorithm's best-fitness
// convergence history into the archive on SwarmState. Replaces an existing
// entry for the same algo+fitness combo. Evicts the oldest entry when full.
func archiveConvergenceCurve(ss *SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil || len(sa.ConvergenceHistory) < 2 {
		return
	}
	// Make a copy so the slice is independent of the SwarmAlgorithmState.
	hist := make([]float64, len(sa.ConvergenceHistory))
	copy(hist, sa.ConvergenceHistory)
	entry := ConvergenceArchiveEntry{
		Algo:        sa.ActiveAlgo,
		FitnessFunc: sa.FitnessFunc,
		BestHistory: hist,
	}
	// Replace existing entry for same algo+fitness combo.
	for i, e := range ss.ConvergenceArchive {
		if e.Algo == entry.Algo && e.FitnessFunc == entry.FitnessFunc {
			ss.ConvergenceArchive[i] = entry
			return
		}
	}
	// Evict oldest if at capacity.
	if len(ss.ConvergenceArchive) >= convergenceArchiveMax {
		ss.ConvergenceArchive = ss.ConvergenceArchive[1:]
	}
	ss.ConvergenceArchive = append(ss.ConvergenceArchive, entry)
}

// ─── AUTO-TOURNAMENT ───────────────────────────────────

// AlgoTournamentTicksPerAlgo is the number of simulation ticks each algorithm
// gets during an auto-tournament. At 10 ticks per convergence sample, this
// yields 300 samples — enough to show convergence behaviour.
const AlgoTournamentTicksPerAlgo = 3000

// tournamentAlgos returns the list of optimisation algorithms eligible for
// tournament benchmarking. Boids and ACO are excluded because they do not
// optimise a fitness function.
func tournamentAlgos() []SwarmAlgorithmType {
	var algos []SwarmAlgorithmType
	for a := AlgoPSO; a < AlgoCount; a++ {
		if a == AlgoACO {
			continue
		}
		algos = append(algos, a)
	}
	return algos
}

// StartAlgoTournament begins an auto-tournament that cycles through all
// optimisation algorithms on the current fitness landscape. The scoreboard
// and convergence archive are cleared first. If a tournament is already
// running it is cancelled instead.
func StartAlgoTournament(ss *SwarmState) {
	if ss.AlgoTournamentOn {
		StopAlgoTournament(ss)
		return
	}
	ClearAlgoScoreboard(ss)
	algos := tournamentAlgos()
	ss.AlgoTournamentOn = true
	ss.AlgoTournamentQueue = algos[1:] // everything after the first
	ss.AlgoTournamentTotal = len(algos)
	ss.AlgoTournamentDone = 0
	ss.AlgoTournamentCur = algos[0]
	ss.AlgoTournamentTicks = AlgoTournamentTicksPerAlgo

	// Preserve the fitness landscape type across init.
	fitFunc := FitGaussian
	if ss.SwarmAlgo != nil {
		fitFunc = ss.SwarmAlgo.FitnessFunc
	}
	InitSwarmAlgorithm(ss, algos[0])
	ss.SwarmAlgo.FitnessFunc = fitFunc
}

// StopAlgoTournament cancels a running tournament, keeping whatever scoreboard
// results have been collected so far.
func StopAlgoTournament(ss *SwarmState) {
	ss.AlgoTournamentOn = false
	ss.AlgoTournamentQueue = nil
	ss.AlgoTournamentTicks = 0
}

// TickAlgoTournament advances the algorithm tournament by one tick. It
// decrements the tick counter for the current algorithm and, when it expires,
// records performance and moves to the next algorithm. Returns true while
// the tournament is still running.
func TickAlgoTournament(ss *SwarmState) bool {
	if !ss.AlgoTournamentOn {
		return false
	}
	ss.AlgoTournamentTicks--
	if ss.AlgoTournamentTicks > 0 {
		return true
	}
	// Current algorithm finished — record and advance.
	ss.AlgoTournamentDone++
	if len(ss.AlgoTournamentQueue) == 0 {
		// All done — record the last algorithm and stop.
		recordAlgoPerformance(ss)
		ss.AlgoTournamentOn = false
		return false
	}
	// Switch to the next algorithm in the queue.
	fitFunc := ss.SwarmAlgo.FitnessFunc
	next := ss.AlgoTournamentQueue[0]
	ss.AlgoTournamentQueue = ss.AlgoTournamentQueue[1:]
	ss.AlgoTournamentCur = next
	ss.AlgoTournamentTicks = AlgoTournamentTicksPerAlgo
	InitSwarmAlgorithm(ss, next)
	ss.SwarmAlgo.FitnessFunc = fitFunc
	return true
}

// ClearAlgoScoreboard resets the performance scoreboard and convergence archive
// (e.g. when the fitness landscape changes and old comparisons are no longer valid).
func ClearAlgoScoreboard(ss *SwarmState) {
	ss.AlgoScoreboard = nil
	ss.ConvergenceArchive = nil
}

// ─── BOIDS ─────────────────────────────────────────────

// tickBoids uses the spatial hash for O(n·k) neighbor lookups instead of O(n²).
// Only bots within BoidsCohesionDist (the largest range) are considered as
// candidates, then filtered by the tighter separation/alignment radii.
func tickBoids(ss *SwarmState, sa *SwarmAlgorithmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]

		sepX, sepY := 0.0, 0.0 // separation
		aliX, aliY := 0.0, 0.0 // alignment
		cohX, cohY := 0.0, 0.0 // cohesion
		aliCount, cohCount := 0, 0

		// Query spatial hash with the largest radius (cohesion) to get candidates.
		maxRange := sa.BoidsCohesionDist
		if sa.BoidsAlignmentDist > maxRange {
			maxRange = sa.BoidsAlignmentDist
		}
		candidates := ss.Hash.Query(bot.X, bot.Y, maxRange)
		for _, j := range candidates {
			if j == i {
				continue
			}
			dx := ss.Bots[j].X - bot.X
			dy := ss.Bots[j].Y - bot.Y
			d := math.Sqrt(dx*dx + dy*dy)

			// Separation
			if d < sa.BoidsSeparationDist && d > 0 {
				sepX -= dx / d
				sepY -= dy / d
			}
			// Alignment
			if d < sa.BoidsAlignmentDist {
				aliX += math.Cos(ss.Bots[j].Angle)
				aliY += math.Sin(ss.Bots[j].Angle)
				aliCount++
			}
			// Cohesion
			if d < sa.BoidsCohesionDist {
				cohX += dx
				cohY += dy
				cohCount++
			}
		}

		// Average and weight
		desiredAngle := bot.Angle
		fx, fy := 0.0, 0.0

		fx += sepX * sa.BoidsSepWeight
		fy += sepY * sa.BoidsSepWeight

		if aliCount > 0 {
			fx += (aliX / float64(aliCount)) * sa.BoidsAlignWeight
			fy += (aliY / float64(aliCount)) * sa.BoidsAlignWeight
		}
		if cohCount > 0 {
			fx += (cohX / float64(cohCount)) * sa.BoidsCohWeight
			fy += (cohY / float64(cohCount)) * sa.BoidsCohWeight
		}

		if fx != 0 || fy != 0 {
			desiredAngle = math.Atan2(fy, fx)
		}

		// Smooth turning via the shared steering helper (steerToward normalises
		// the angular difference and clamps the step to BoidsMaxTurn radians).
		steerToward(bot, desiredAngle, sa.BoidsMaxTurn)
		bot.Speed = sa.BoidsMaxSpeed

		// LED: color by heading for visual effect
		hue := (bot.Angle + math.Pi) / (2 * math.Pi)
		r, g, b := hsvToRGB(hue, 0.8, 1.0)
		bot.LEDColor = [3]uint8{r, g, b}
	}
}

// PSO and ACO are implemented in their dedicated files (pso.go, aco.go)
// using the Init/Tick/Apply pattern shared by all meta-heuristic algorithms.

// initFitnessLandscape generates 3-5 Gaussian peaks as a shared fitness
// landscape if one does not already exist. The landscape is stored in
// SwarmAlgorithmState and persists across algorithm switches so users can
// compare convergence behaviour on the same landscape.
func initFitnessLandscape(ss *SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil {
		return
	}
	// Already initialised (preserved across algorithm switches).
	if len(sa.FitPeakX) > 0 {
		return
	}
	numPeaks := 3 + ss.Rng.Intn(3) // 3-5 peaks
	sa.FitPeakX = make([]float64, numPeaks)
	sa.FitPeakY = make([]float64, numPeaks)
	sa.FitPeakH = make([]float64, numPeaks)
	sa.FitPeakS = make([]float64, numPeaks)
	for p := 0; p < numPeaks; p++ {
		sa.FitPeakX[p] = ss.ArenaW*0.1 + ss.Rng.Float64()*ss.ArenaW*0.8
		sa.FitPeakY[p] = ss.ArenaH*0.1 + ss.Rng.Float64()*ss.ArenaH*0.8
		sa.FitPeakH[p] = 50 + ss.Rng.Float64()*50
		sa.FitPeakS[p] = 40 + ss.Rng.Float64()*80
	}
}

// TickDynamicLandscape moves the Gaussian fitness peaks when dynamic mode is
// active. Each peak drifts with a per-peak velocity vector, bouncing off the
// arena boundaries. Peak heights oscillate slowly within [30, 100] and sigmas
// within [30, 120]. Only affects FitGaussian; benchmark functions have fixed
// formulas and are not modified.
func TickDynamicLandscape(ss *SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil || !sa.DynamicLandscape || sa.FitnessFunc != FitGaussian {
		return
	}
	n := len(sa.FitPeakX)
	if n == 0 {
		return
	}

	// Lazy-init velocities when dynamic mode is first enabled.
	if len(sa.DynVelX) != n {
		sa.DynVelX = make([]float64, n)
		sa.DynVelY = make([]float64, n)
		sa.DynVelH = make([]float64, n)
		sa.DynVelS = make([]float64, n)
		for i := 0; i < n; i++ {
			sa.DynVelX[i] = (ss.Rng.Float64() - 0.5) * 1.0 // -0.5 .. +0.5 px/tick
			sa.DynVelY[i] = (ss.Rng.Float64() - 0.5) * 1.0
			sa.DynVelH[i] = (ss.Rng.Float64() - 0.5) * 0.1 // slow height drift
			sa.DynVelS[i] = (ss.Rng.Float64() - 0.5) * 0.2 // slow sigma drift
		}
	}

	margin := 0.05 // 5% arena margin for bounce
	minX := ss.ArenaW * margin
	maxX := ss.ArenaW * (1 - margin)
	minY := ss.ArenaH * margin
	maxY := ss.ArenaH * (1 - margin)

	for i := 0; i < n; i++ {
		// Update position
		sa.FitPeakX[i] += sa.DynVelX[i]
		sa.FitPeakY[i] += sa.DynVelY[i]

		// Bounce off arena boundaries
		if sa.FitPeakX[i] < minX {
			sa.FitPeakX[i] = minX
			sa.DynVelX[i] = -sa.DynVelX[i]
		} else if sa.FitPeakX[i] > maxX {
			sa.FitPeakX[i] = maxX
			sa.DynVelX[i] = -sa.DynVelX[i]
		}
		if sa.FitPeakY[i] < minY {
			sa.FitPeakY[i] = minY
			sa.DynVelY[i] = -sa.DynVelY[i]
		} else if sa.FitPeakY[i] > maxY {
			sa.FitPeakY[i] = maxY
			sa.DynVelY[i] = -sa.DynVelY[i]
		}

		// Update height with bounce within [30, 100]
		sa.FitPeakH[i] += sa.DynVelH[i]
		if sa.FitPeakH[i] < 30 {
			sa.FitPeakH[i] = 30
			sa.DynVelH[i] = -sa.DynVelH[i]
		} else if sa.FitPeakH[i] > 100 {
			sa.FitPeakH[i] = 100
			sa.DynVelH[i] = -sa.DynVelH[i]
		}

		// Update sigma with bounce within [30, 120]
		sa.FitPeakS[i] += sa.DynVelS[i]
		if sa.FitPeakS[i] < 30 {
			sa.FitPeakS[i] = 30
			sa.DynVelS[i] = -sa.DynVelS[i]
		} else if sa.FitPeakS[i] > 120 {
			sa.FitPeakS[i] = 120
			sa.DynVelS[i] = -sa.DynVelS[i]
		}
	}
}

// EvaluateFitnessLandscape computes the fitness at (x, y) using the selected
// fitness function. For Gaussian peaks it sums Gaussian contributions; for
// standard benchmark functions it maps arena coordinates to the function's
// canonical domain and returns the *negated* value (since all algorithms
// maximise fitness, but benchmark functions are conventionally minimised).
func EvaluateFitnessLandscape(sa *SwarmAlgorithmState, x, y float64) float64 {
	if sa == nil {
		return 0
	}
	switch sa.FitnessFunc {
	case FitRastrigin:
		return evalRastrigin(x, y)
	case FitAckley:
		return evalAckley(x, y)
	case FitRosenbrock:
		return evalRosenbrock(x, y)
	case FitSchwefel:
		return evalSchwefel(x, y)
	case FitGriewank:
		return evalGriewank(x, y)
	case FitLevy:
		return evalLevy(x, y)
	default: // FitGaussian
		if len(sa.FitPeakX) == 0 {
			return 0
		}
		fit := 0.0
		for p := range sa.FitPeakX {
			dx := x - sa.FitPeakX[p]
			dy := y - sa.FitPeakY[p]
			fit += sa.FitPeakH[p] * math.Exp(-(dx*dx+dy*dy)/(2*sa.FitPeakS[p]*sa.FitPeakS[p]))
		}
		return fit
	}
}

// arenaToCanonical maps arena pixel coordinates (0..800) into a canonical
// optimisation domain [-d, d] for standard benchmark functions.
func arenaToCanonical(x, y, d float64) (float64, float64) {
	// Map [0, 800] → [-d, d] assuming 800×800 arena
	cx := (x/SwarmArenaSize)*2*d - d
	cy := (y/SwarmArenaSize)*2*d - d
	return cx, cy
}

// evalRastrigin evaluates the 2D Rastrigin function mapped to the arena.
// Domain: [-5.12, 5.12], global minimum at (0,0) = 0.
// Highly multi-modal with many local minima in a regular grid pattern.
func evalRastrigin(ax, ay float64) float64 {
	x, y := arenaToCanonical(ax, ay, 5.12)
	f := 20 + x*x - 10*math.Cos(2*math.Pi*x) + y*y - 10*math.Cos(2*math.Pi*y)
	// Negate and shift so higher = better, roughly 0..100 range
	return 100 - f
}

// evalAckley evaluates the 2D Ackley function mapped to the arena.
// Domain: [-5, 5], global minimum at (0,0) = 0.
// Nearly flat outer region with a sharp global minimum.
func evalAckley(ax, ay float64) float64 {
	x, y := arenaToCanonical(ax, ay, 5.0)
	sum1 := x*x + y*y
	sum2 := math.Cos(2*math.Pi*x) + math.Cos(2*math.Pi*y)
	f := -20*math.Exp(-0.2*math.Sqrt(0.5*sum1)) - math.Exp(0.5*sum2) + math.E + 20
	// Negate and shift: Ackley max ≈22.7, map to ~0..100
	return (22.7 - f) * (100.0 / 22.7)
}

// evalRosenbrock evaluates the 2D Rosenbrock function mapped to the arena.
// Domain: [-2.048, 2.048], global minimum at (1,1) = 0.
// Narrow curved valley — easy to find the valley, hard to converge to the minimum.
func evalRosenbrock(ax, ay float64) float64 {
	x, y := arenaToCanonical(ax, ay, 2.048)
	a := 1.0 - x
	b := y - x*x
	f := a*a + 100*b*b
	// Clamp and map: worst case ~3900, map to 0..100
	if f > 4000 {
		f = 4000
	}
	return 100 * (1 - f/4000)
}

// evalSchwefel evaluates the 2D Schwefel function mapped to the arena.
// Domain: [-500, 500], global minimum at (420.97, 420.97) ≈ -837.96.
// Deceptive: global minimum is far from the next-best local minimum.
func evalSchwefel(ax, ay float64) float64 {
	x, y := arenaToCanonical(ax, ay, 500.0)
	f := 418.9829*2 - (x*math.Sin(math.Sqrt(math.Abs(x))) + y*math.Sin(math.Sqrt(math.Abs(y))))
	// f in [0, ~1676], map to 0..100 (lower f = higher fitness)
	if f < 0 {
		f = 0
	}
	if f > 1676 {
		f = 1676
	}
	return 100 * (1 - f/1676)
}

// evalGriewank evaluates the 2D Griewank function mapped to the arena.
// Domain: [-600, 600], global minimum at (0,0) = 0.
// Many local minima but the product term makes them shallow, rewarding
// algorithms that can exploit the global structure rather than getting
// trapped in shallow local wells.
func evalGriewank(ax, ay float64) float64 {
	x, y := arenaToCanonical(ax, ay, 600.0)
	sum := (x*x + y*y) / 4000.0
	prod := math.Cos(x/1.0) * math.Cos(y/math.Sqrt(2.0))
	f := sum - prod + 1 // f in [0, ~180+], global min = 0
	if f > 200 {
		f = 200
	}
	return 100 * (1 - f/200)
}

// evalLevy evaluates the 2D Levy function mapped to the arena.
// Domain: [-10, 10], global minimum at (1,1) = 0.
// Complex multimodal landscape with sinusoidal terms.
func evalLevy(ax, ay float64) float64 {
	x, y := arenaToCanonical(ax, ay, 10.0)
	w1 := 1.0 + (x-1.0)/4.0
	w2 := 1.0 + (y-1.0)/4.0
	term1 := math.Sin(math.Pi*w1) * math.Sin(math.Pi*w1)
	term2 := (w1 - 1) * (w1 - 1) * (1 + 10*math.Sin(math.Pi*w1+1)*math.Sin(math.Pi*w1+1))
	term3 := (w2 - 1) * (w2 - 1) * (1 + math.Sin(2*math.Pi*w2)*math.Sin(2*math.Pi*w2))
	f := term1 + term2 + term3 // f in [0, ~50+], global min = 0
	if f > 60 {
		f = 60
	}
	return 100 * (1 - f/60)
}

// distanceFitness evaluates fitness using the shared Gaussian fitness landscape.
// Falls back to a simple distance-to-center/light metric if no landscape exists.
func distanceFitness(bot *SwarmBot, ss *SwarmState) float64 {
	if ss.SwarmAlgo != nil && len(ss.SwarmAlgo.FitPeakX) > 0 {
		return EvaluateFitnessLandscape(ss.SwarmAlgo, bot.X, bot.Y)
	}
	// Fallback: simple proximity to light or center.
	targetX, targetY := ss.ArenaW/2, ss.ArenaH/2
	if ss.Light.Active {
		targetX = ss.Light.X
		targetY = ss.Light.Y
	}
	dx := bot.X - targetX
	dy := bot.Y - targetY
	dist := math.Sqrt(dx*dx + dy*dy)
	return 100 - dist*0.2
}

// distanceFitnessPt evaluates the fitness at an arbitrary (x, y) coordinate
// without requiring a SwarmBot reference. Used by algorithms that generate
// candidate positions and need to evaluate them before moving the bot.
func distanceFitnessPt(ss *SwarmState, x, y float64) float64 {
	if ss.SwarmAlgo != nil && len(ss.SwarmAlgo.FitPeakX) > 0 {
		return EvaluateFitnessLandscape(ss.SwarmAlgo, x, y)
	}
	targetX, targetY := ss.ArenaW/2, ss.ArenaH/2
	if ss.Light.Active {
		targetX = ss.Light.X
		targetY = ss.Light.Y
	}
	dx := x - targetX
	dy := y - targetY
	dist := math.Sqrt(dx*dx + dy*dy)
	return 100 - dist*0.2
}

// ─── FIREFLY ───────────────────────────────────────────

const fireflyMaxTicks = 3000 // cycle length for alpha decay and exploration ratio

// InitFireflyAlgo allocates per-bot brightness and initialises global best tracking.
func InitFireflyAlgo(ss *SwarmState) {
	sa := ss.SwarmAlgo
	n := len(ss.Bots)
	sa.FireflyBrightness = make([]float64, n)
	sa.FireflyBestIdx = -1
	sa.FireflyBestF = -1e18
	sa.FireflyCycleTick = 0
	sa.FireflyAlpha0 = sa.FireflyAlpha
}

// ClearFireflyAlgo frees the per-bot brightness slice.
func ClearFireflyAlgo(ss *SwarmState) {
	sa := ss.SwarmAlgo
	sa.FireflyBrightness = nil
	sa.FireflyBestIdx = -1
}

// tickFirefly uses the spatial hash for O(n·k) neighbor lookups.
// Attractiveness decays as exp(-gamma*r²), so beyond a cutoff radius the
// contribution is negligible (<1% of beta0). The cutoff is derived from
// -gamma*r² = ln(0.01), i.e. r = sqrt(-ln(0.01)/gamma).
//
// Alpha (randomization) decays over the cycle: alpha = alpha0 * (1 - t/T)
// This transitions from exploration (high alpha, large random walks) to
// exploitation (low alpha, precise local search) — a standard FA modification
// (Yang 2009, Yang & He 2013).
func tickFirefly(ss *SwarmState, sa *SwarmAlgorithmState) {
	n := len(ss.Bots)
	if len(sa.FireflyBrightness) != n {
		sa.FireflyBrightness = make([]float64, n)
	}

	// Alpha decay: linearly decrease randomization over the cycle.
	sa.FireflyCycleTick++
	if sa.FireflyCycleTick > fireflyMaxTicks {
		sa.FireflyCycleTick = 0 // restart cycle
	}
	tRatio := float64(sa.FireflyCycleTick) / float64(fireflyMaxTicks)
	sa.FireflyAlpha = sa.FireflyAlpha0 * (1.0 - tRatio)
	if sa.FireflyAlpha < 0.01 {
		sa.FireflyAlpha = 0.01
	}

	// Compute brightness (fitness) for each firefly and track global best.
	for i := range ss.Bots {
		f := distanceFitness(&ss.Bots[i], ss)
		sa.FireflyBrightness[i] = f
		if f > sa.FireflyBestF {
			sa.FireflyBestF = f
			sa.FireflyBestX = ss.Bots[i].X
			sa.FireflyBestY = ss.Bots[i].Y
			sa.FireflyBestIdx = i
		}
	}

	// Compute cutoff radius where attractiveness drops below 1% of beta0.
	// For very small gamma, cap at arena diagonal to avoid huge queries.
	cutoff := math.Sqrt(ss.ArenaW*ss.ArenaW + ss.ArenaH*ss.ArenaH)
	if sa.FireflyGamma > 0 {
		cutoff = math.Min(cutoff, math.Sqrt(-math.Log(0.01)/sa.FireflyGamma))
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		moveX, moveY := 0.0, 0.0

		candidates := ss.Hash.Query(bot.X, bot.Y, cutoff)
		for _, j := range candidates {
			if j == i {
				continue
			}
			// Only move toward brighter fireflies
			if sa.FireflyBrightness[j] <= sa.FireflyBrightness[i] {
				continue
			}

			dx := ss.Bots[j].X - bot.X
			dy := ss.Bots[j].Y - bot.Y
			r2 := dx*dx + dy*dy
			r := math.Sqrt(r2)

			// Attractiveness decreases with distance
			beta := sa.FireflyBeta0 * math.Exp(-sa.FireflyGamma*r2)

			if r > 0 {
				moveX += beta * dx / r
				moveY += beta * dy / r
			}
		}

		// Add random walk (alpha decays over time)
		moveX += sa.FireflyAlpha * (ss.Rng.Float64() - 0.5) * 2
		moveY += sa.FireflyAlpha * (ss.Rng.Float64() - 0.5) * 2

		if moveX != 0 || moveY != 0 {
			// Use smooth steering rather than snapping the heading instantly.
			// The rate limit of 0.3 rad/tick gives fireflies a natural, gentle
			// turning arc while still tracking brighter neighbours closely.
			desired := math.Atan2(moveY, moveX)
			steerToward(bot, desired, 0.3)
			bot.Speed = math.Min(math.Sqrt(moveX*moveX+moveY*moveY), SwarmBotSpeed*1.5)
		}

		// LED: warm-to-cool gradient based on fitness; gold for global best
		if i == sa.FireflyBestIdx {
			bot.LEDColor = [3]uint8{255, 215, 0} // gold
		} else {
			b01 := (sa.FireflyBrightness[i] + 50) / 150
			if b01 < 0 {
				b01 = 0
			}
			if b01 > 1 {
				b01 = 1
			}
			c := uint8(b01 * 255)
			bot.LEDColor = [3]uint8{c, c, 0}
		}
	}
}

// ─── HELPERS ───────────────────────────────────────────

// hsvToRGB converts HSV (h: 0-1, s: 0-1, v: 0-1) to RGB.
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	h = h - math.Floor(h) // wrap to 0-1
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)

	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	}
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}

// AlgorithmNames returns the display names of all available algorithms,
// ordered to match the SwarmAlgorithmType enum (AlgoNone through AlgoCuckoo).
func AlgorithmNames() []string {
	return []string{
		"Keiner", "Boids", "PSO", "ACO", "Firefly",
		"Grey Wolf", "Whale", "Bacterial Foraging", "Moth-Flame", "Cuckoo Search",
		"Differential Evolution", "Bee Colony (ABC)", "Harmony Search (HSO)",
		"Bat Algorithm (BA)",
		"Salp Swarm (SSA)",
		"Gravitational Search (GSA)",
		"Flower Pollination (FPA)",
		"Harris Hawks (HHO)",
		"Simulated Annealing (SA)",
		"Aquila Optimizer (AO)",
		"Sine Cosine (SCA)",
		"Dragonfly (DA)",
		"Teaching-Learning (TLBO)",
		"Equilibrium Optimizer (EO)",
		"Jaya Algorithm (Rao)",
	}
}
