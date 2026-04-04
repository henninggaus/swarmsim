package swarm

// gridPt represents a grid point with its fitness value.
// Used by all optimization algorithms during grid scan phases.
type gridPt struct {
	x, y, f float64
}

// idxFit pairs a bot/agent index with its fitness for ranking.
// Used by algorithms to sort agents by fitness.
type idxFit struct {
	idx int
	f   float64
}

const (
	// AlgoGridRescanSize is the default grid size for periodic rescans (NxN points).
	// Used by bacterial_foraging, de, hso, sca algorithms.
	AlgoGridRescanSize = 20

	// AlgoGridInjectTop is the default number of top grid points injected into worst bots.
	// Used by bat, cuckoo_search, dragonfly, eo, fpa, grey_wolf, gsa, hho, moth_flame, pso, whale.
	AlgoGridInjectTop = 10
)
