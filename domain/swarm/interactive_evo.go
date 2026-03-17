package swarm

import (
	"math"
	"swarmsim/logger"
)

// InteractiveEvoState manages user-guided interactive evolution.
// Instead of fitness functions, the user selects which simulations
// they find most interesting — like Karl Sims' evolved creatures.
type InteractiveEvoState struct {
	// Candidates: each candidate is a full set of bot brains/programs
	Candidates    []InteractiveCandidate
	NumCandidates int // how many parallel candidates to show (default 6)

	// Selection
	Selected      []bool // which candidates the user selected
	SelectionDone bool   // user has confirmed selection

	// Evolution state
	Generation      int
	TicksPerPreview int // ticks to run each candidate before selection (default 1000)
	CurrentTick     int // tick within current preview
	ActiveCandidate int // which candidate is currently being previewed (-1 = all parallel)

	// Mode
	Mode InteractiveEvoMode // parallel or sequential preview
}

// InteractiveEvoMode controls how candidates are presented.
type InteractiveEvoMode int

const (
	InteractiveParallel   InteractiveEvoMode = iota // show all at once (miniature)
	InteractiveSequential                            // cycle through one at a time
)

// InteractiveCandidate holds one evolutionary candidate.
type InteractiveCandidate struct {
	ID       int
	Weights  [][NeuroWeights]float64 // per-bot neuro weights
	Morphs   []Morphology            // per-bot morphologies (optional)
	Fitness  float64                 // auto-computed fitness (for reference)
	UserScore int                    // user preference score (0-5 stars)
	Label    string                  // user-assigned label
}

// InitInteractiveEvo sets up the interactive evolution system.
func InitInteractiveEvo(ss *SwarmState, numCandidates int) {
	if numCandidates < 2 {
		numCandidates = 2
	}
	if numCandidates > 8 {
		numCandidates = 8
	}

	ie := &InteractiveEvoState{
		NumCandidates:   numCandidates,
		TicksPerPreview: 1000,
		ActiveCandidate: -1,
		Mode:            InteractiveParallel,
	}

	// Generate initial random candidates
	ie.Candidates = make([]InteractiveCandidate, numCandidates)
	ie.Selected = make([]bool, numCandidates)

	n := len(ss.Bots)
	for c := 0; c < numCandidates; c++ {
		ie.Candidates[c] = InteractiveCandidate{
			ID:      c,
			Weights: make([][NeuroWeights]float64, n),
			Label:   "",
		}
		// Random initial weights
		for i := 0; i < n; i++ {
			for w := 0; w < NeuroWeights; w++ {
				ie.Candidates[c].Weights[i][w] = (ss.Rng.Float64() - 0.5) * 2.0 / math.Sqrt(float64(NeuroInputs))
			}
		}
		if ss.MorphEnabled {
			ie.Candidates[c].Morphs = make([]Morphology, n)
			for i := 0; i < n; i++ {
				ie.Candidates[c].Morphs[i] = RandomMorphology(ss.Rng)
			}
		}
	}

	ss.InteractiveEvo = ie
	logger.Info("INTERACTIVE", "Initialisiert: %d Kandidaten, %d Ticks/Preview",
		numCandidates, ie.TicksPerPreview)
}

// ClearInteractiveEvo disables the interactive evolution system.
func ClearInteractiveEvo(ss *SwarmState) {
	ss.InteractiveEvo = nil
	ss.InteractiveEvoOn = false
}

// LoadCandidate loads a candidate's weights into the active bots.
func LoadCandidate(ss *SwarmState, candidateIdx int) {
	ie := ss.InteractiveEvo
	if ie == nil || candidateIdx < 0 || candidateIdx >= len(ie.Candidates) {
		return
	}
	cand := &ie.Candidates[candidateIdx]
	n := len(ss.Bots)

	for i := 0; i < n && i < len(cand.Weights); i++ {
		if ss.Bots[i].Brain == nil {
			ss.Bots[i].Brain = &NeuroBrain{}
		}
		ss.Bots[i].Brain.Weights = cand.Weights[i]

		if cand.Morphs != nil && i < len(cand.Morphs) && ss.MorphEnabled {
			ss.Bots[i].Morph = cand.Morphs[i]
		}
	}

	ie.ActiveCandidate = candidateIdx
	ie.CurrentTick = 0

	// Reset bot stats for fair evaluation
	for i := range ss.Bots {
		ss.Bots[i].Stats = BotLifetimeStats{}
		ss.Bots[i].Fitness = 0
	}
}

// SaveCandidateFitness records the fitness of the current candidate after preview.
func SaveCandidateFitness(ss *SwarmState) {
	ie := ss.InteractiveEvo
	if ie == nil || ie.ActiveCandidate < 0 {
		return
	}

	// Compute average fitness
	totalFit := 0.0
	for i := range ss.Bots {
		totalFit += EvaluateGPFitness(&ss.Bots[i])
	}
	ie.Candidates[ie.ActiveCandidate].Fitness = totalFit / float64(len(ss.Bots))
}

// SelectCandidate toggles the selection state of a candidate.
func SelectCandidate(ie *InteractiveEvoState, idx int) {
	if ie == nil || idx < 0 || idx >= len(ie.Selected) {
		return
	}
	ie.Selected[idx] = !ie.Selected[idx]
}

// SetUserScore sets a user preference score (0-5) for a candidate.
func SetUserScore(ie *InteractiveEvoState, idx, score int) {
	if ie == nil || idx < 0 || idx >= len(ie.Candidates) {
		return
	}
	if score < 0 {
		score = 0
	}
	if score > 5 {
		score = 5
	}
	ie.Candidates[idx].UserScore = score
}

// EvolveInteractive breeds selected candidates to create the next generation.
func EvolveInteractive(ss *SwarmState) {
	ie := ss.InteractiveEvo
	if ie == nil {
		return
	}

	// Collect selected parent indices
	var parents []int
	for i, sel := range ie.Selected {
		if sel {
			parents = append(parents, i)
		}
	}

	// Fallback: if nothing selected, use top 2 by user score
	if len(parents) == 0 {
		parents = topByUserScore(ie, 2)
	}
	if len(parents) < 1 {
		logger.Info("INTERACTIVE", "Keine Kandidaten ausgewaehlt!")
		return
	}

	n := len(ss.Bots)
	numCand := ie.NumCandidates

	// Create new generation
	newCandidates := make([]InteractiveCandidate, numCand)

	for c := 0; c < numCand; c++ {
		newCandidates[c] = InteractiveCandidate{
			ID:      c,
			Weights: make([][NeuroWeights]float64, n),
		}

		// Pick two parents (with replacement)
		p1 := parents[ss.Rng.Intn(len(parents))]
		p2 := parents[ss.Rng.Intn(len(parents))]

		for i := 0; i < n; i++ {
			// Crossover
			for w := 0; w < NeuroWeights; w++ {
				if ss.Rng.Float64() < 0.5 {
					newCandidates[c].Weights[i][w] = ie.Candidates[p1].Weights[i][w]
				} else {
					newCandidates[c].Weights[i][w] = ie.Candidates[p2].Weights[i][w]
				}
				// Mutation
				if ss.Rng.Float64() < 0.1 {
					newCandidates[c].Weights[i][w] += ss.Rng.NormFloat64() * 0.2
				}
			}
		}

		// Morphology crossover
		if ss.MorphEnabled && len(ie.Candidates[p1].Morphs) > 0 {
			newCandidates[c].Morphs = make([]Morphology, n)
			cfg := DefaultMorphologyConfig()
			for i := 0; i < n; i++ {
				m1 := ie.Candidates[p1].Morphs[i%len(ie.Candidates[p1].Morphs)]
				m2 := ie.Candidates[p2].Morphs[i%len(ie.Candidates[p2].Morphs)]
				child := CrossoverMorphology(ss.Rng, m1, m2)
				newCandidates[c].Morphs[i] = MutateMorphology(ss.Rng, child, cfg)
			}
		}
	}

	ie.Candidates = newCandidates
	ie.Selected = make([]bool, numCand)
	ie.SelectionDone = false
	ie.Generation++

	logger.Info("INTERACTIVE", "Gen %d: %d Eltern → %d neue Kandidaten",
		ie.Generation, len(parents), numCand)
}

// topByUserScore returns indices of top N candidates by user score.
func topByUserScore(ie *InteractiveEvoState, n int) []int {
	if ie == nil || len(ie.Candidates) == 0 {
		return nil
	}

	type scored struct {
		idx   int
		score int
	}
	items := make([]scored, len(ie.Candidates))
	for i := range ie.Candidates {
		items[i] = scored{i, ie.Candidates[i].UserScore}
	}

	// Simple selection sort
	for i := 0; i < n && i < len(items); i++ {
		best := i
		for j := i + 1; j < len(items); j++ {
			if items[j].score > items[best].score {
				best = j
			}
		}
		items[i], items[best] = items[best], items[i]
	}

	result := make([]int, 0, n)
	for i := 0; i < n && i < len(items); i++ {
		if items[i].score > 0 {
			result = append(result, items[i].idx)
		}
	}

	// If no scored candidates, just pick first two
	if len(result) == 0 && len(items) >= 2 {
		result = []int{items[0].idx, items[1].idx}
	}
	return result
}

// InteractiveCandidateCount returns the number of candidates.
func InteractiveCandidateCount(ie *InteractiveEvoState) int {
	if ie == nil {
		return 0
	}
	return len(ie.Candidates)
}

// InteractiveSelectedCount returns how many candidates are selected.
func InteractiveSelectedCount(ie *InteractiveEvoState) int {
	if ie == nil {
		return 0
	}
	count := 0
	for _, s := range ie.Selected {
		if s {
			count++
		}
	}
	return count
}
