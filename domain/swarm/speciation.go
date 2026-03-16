package swarm

import (
	"math"
	"sort"
)

// Species represents a group of genetically similar bots.
type Species struct {
	ID            int       // unique species identifier
	Members       []int     // bot indices belonging to this species
	CenterGenome  [26]float64 // representative genome (centroid)
	Color         [3]uint8  // display color for this species
	AvgFitness    float64
	BestFitness   float64
	Age           int       // generations this species has existed
	Stagnant      int       // generations without improvement
	PrevBestFit   float64
}

// SpeciationState tracks all species and their evolution history.
type SpeciationState struct {
	Species        []Species
	NextSpeciesID  int
	Threshold      float64 // distance threshold for same species (auto-adjusted)
	TargetSpecies  int     // target number of species (5-10)
	History        []SpeciesSnapshot // per-generation species data
}

// SpeciesSnapshot records species composition at one generation.
type SpeciesSnapshot struct {
	Generation int
	Counts     []SpeciesCount
}

// SpeciesCount is how many members a species has at one point.
type SpeciesCount struct {
	ID    int
	Count int
	Color [3]uint8
}

// speciesColors are distinct colors for up to 12 species.
var speciesColors = [][3]uint8{
	{255, 80, 80},    // red
	{80, 180, 255},   // blue
	{80, 255, 80},    // green
	{255, 200, 50},   // yellow
	{200, 80, 255},   // purple
	{255, 150, 50},   // orange
	{50, 255, 200},   // cyan
	{255, 80, 200},   // pink
	{200, 255, 80},   // lime
	{80, 120, 255},   // indigo
	{255, 255, 150},  // light yellow
	{150, 100, 80},   // brown
}

// InitSpeciation initializes the speciation system.
func InitSpeciation(ss *SwarmState) {
	ss.Speciation = &SpeciationState{
		NextSpeciesID: 1,
		Threshold:     0.3,
		TargetSpecies: 7,
	}
}

// UpdateSpeciation assigns bots to species based on genome similarity.
// Call once per generation after evolution.
func UpdateSpeciation(ss *SwarmState) {
	spec := ss.Speciation
	if spec == nil {
		return
	}

	n := len(ss.Bots)
	if n == 0 {
		return
	}

	// Age existing species
	for i := range spec.Species {
		spec.Species[i].Age++
		spec.Species[i].Members = nil
	}

	// Assign each bot to nearest species or create new species
	for bi := range ss.Bots {
		bot := &ss.Bots[bi]
		bestIdx := -1
		bestDist := math.MaxFloat64

		for si := range spec.Species {
			d := genomeDistance(bot.ParamValues, spec.Species[si].CenterGenome, ss.UsedParams)
			if d < bestDist {
				bestDist = d
				bestIdx = si
			}
		}

		if bestIdx >= 0 && bestDist < spec.Threshold {
			spec.Species[bestIdx].Members = append(spec.Species[bestIdx].Members, bi)
		} else {
			// New species
			colorIdx := spec.NextSpeciesID % len(speciesColors)
			newSpecies := Species{
				ID:           spec.NextSpeciesID,
				Members:      []int{bi},
				CenterGenome: bot.ParamValues,
				Color:        speciesColors[colorIdx],
			}
			spec.Species = append(spec.Species, newSpecies)
			spec.NextSpeciesID++
		}
	}

	// Remove empty species
	alive := spec.Species[:0]
	for _, sp := range spec.Species {
		if len(sp.Members) > 0 {
			alive = append(alive, sp)
		}
	}
	spec.Species = alive

	// Update species centroids and fitness
	for si := range spec.Species {
		sp := &spec.Species[si]
		// Centroid
		for p := 0; p < 26; p++ {
			sum := 0.0
			for _, bi := range sp.Members {
				sum += ss.Bots[bi].ParamValues[p]
			}
			sp.CenterGenome[p] = sum / float64(len(sp.Members))
		}
		// Fitness
		sp.AvgFitness = 0
		sp.BestFitness = 0
		for _, bi := range sp.Members {
			f := ss.Bots[bi].Fitness
			sp.AvgFitness += f
			if f > sp.BestFitness {
				sp.BestFitness = f
			}
		}
		sp.AvgFitness /= float64(len(sp.Members))

		// Stagnation check
		if sp.BestFitness > sp.PrevBestFit+0.1 {
			sp.Stagnant = 0
		} else {
			sp.Stagnant++
		}
		sp.PrevBestFit = sp.BestFitness
	}

	// Auto-adjust threshold to target species count
	if len(spec.Species) < spec.TargetSpecies-1 {
		spec.Threshold *= 0.95 // lower threshold → more species
	} else if len(spec.Species) > spec.TargetSpecies+1 {
		spec.Threshold *= 1.05 // raise threshold → fewer species
	}
	if spec.Threshold < 0.05 {
		spec.Threshold = 0.05
	}
	if spec.Threshold > 1.0 {
		spec.Threshold = 1.0
	}

	// Record history
	snap := SpeciesSnapshot{
		Generation: ss.Generation,
	}
	for _, sp := range spec.Species {
		snap.Counts = append(snap.Counts, SpeciesCount{
			ID:    sp.ID,
			Count: len(sp.Members),
			Color: sp.Color,
		})
	}
	// Sort by count (largest first)
	sort.Slice(snap.Counts, func(i, j int) bool {
		return snap.Counts[i].Count > snap.Counts[j].Count
	})
	spec.History = append(spec.History, snap)
	// Keep last 100 generations
	if len(spec.History) > 100 {
		spec.History = spec.History[len(spec.History)-100:]
	}

	// Set bot LED colors to species colors
	for _, sp := range spec.Species {
		for _, bi := range sp.Members {
			ss.Bots[bi].LEDColor = sp.Color
		}
	}
}

// genomeDistance computes normalized distance between two parameter vectors.
func genomeDistance(a, b [26]float64, used [26]bool) float64 {
	dist := 0.0
	count := 0
	for i := 0; i < 26; i++ {
		if !used[i] {
			continue
		}
		d := a[i] - b[i]
		dist += d * d
		count++
	}
	if count == 0 {
		return 0
	}
	return math.Sqrt(dist/float64(count)) / 100.0
}

// SpeciesCount returns the number of active species.
func SpeciesActiveCount(ss *SwarmState) int {
	if ss.Speciation == nil {
		return 0
	}
	return len(ss.Speciation.Species)
}
