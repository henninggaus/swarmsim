package swarm

import "math"

// DiversityMetrics holds population diversity measurements.
type DiversityMetrics struct {
	AvgDistance   float64 // average pairwise distance (0-1 normalized)
	MinDistance   float64 // minimum pairwise distance (most similar pair)
	UniqueCount  int     // number of distinct genotypes (above threshold)
	Stagnant     bool    // true if diversity is critically low
	AutoInjected int     // count of random genomes injected this generation
}

// MeasureParamDiversity measures diversity for Evolution mode (ParamValues).
func MeasureParamDiversity(ss *SwarmState) DiversityMetrics {
	n := len(ss.Bots)
	if n < 2 {
		return DiversityMetrics{AvgDistance: 1.0, MinDistance: 1.0, UniqueCount: n}
	}

	// Find which params are used
	usedCount := 0
	for _, u := range ss.UsedParams {
		if u {
			usedCount++
		}
	}
	if usedCount == 0 {
		return DiversityMetrics{AvgDistance: 1.0, MinDistance: 1.0, UniqueCount: n}
	}

	// Sample up to 30 bots for pairwise comparison (O(n²) gets expensive)
	sampleSize := n
	if sampleSize > 30 {
		sampleSize = 30
	}

	totalDist := 0.0
	minDist := math.MaxFloat64
	pairs := 0
	threshold := 0.01 // distance below which genomes are "same"

	uniqueMap := make(map[int]bool) // track unique clusters
	for i := 0; i < sampleSize; i++ {
		isUnique := true
		for j := i + 1; j < sampleSize; j++ {
			d := paramDistance(ss, i, j, usedCount)
			totalDist += d
			pairs++
			if d < minDist {
				minDist = d
			}
			if d < threshold {
				isUnique = false
			}
		}
		if isUnique {
			uniqueMap[i] = true
		}
	}

	avgDist := 0.0
	if pairs > 0 {
		avgDist = totalDist / float64(pairs)
	}
	if minDist == math.MaxFloat64 {
		minDist = 0
	}

	return DiversityMetrics{
		AvgDistance:  avgDist,
		MinDistance:  minDist,
		UniqueCount: len(uniqueMap),
		Stagnant:    avgDist < 0.05,
	}
}

// paramDistance computes normalized distance between two bots' param values.
func paramDistance(ss *SwarmState, i, j, usedCount int) float64 {
	dist := 0.0
	for p := 0; p < 26; p++ {
		if !ss.UsedParams[p] {
			continue
		}
		d := ss.Bots[i].ParamValues[p] - ss.Bots[j].ParamValues[p]
		dist += d * d
	}
	// Normalize by param count
	return math.Sqrt(dist / float64(usedCount)) / 100.0 // params typically 0-100
}

// MeasureNeuroDiversity measures diversity for Neuroevolution mode (brain weights).
func MeasureNeuroDiversity(ss *SwarmState) DiversityMetrics {
	n := len(ss.Bots)
	if n < 2 {
		return DiversityMetrics{AvgDistance: 1.0, MinDistance: 1.0, UniqueCount: n}
	}

	sampleSize := n
	if sampleSize > 30 {
		sampleSize = 30
	}

	totalDist := 0.0
	minDist := math.MaxFloat64
	pairs := 0
	threshold := 0.02

	uniqueCount := 0
	for i := 0; i < sampleSize; i++ {
		if ss.Bots[i].Brain == nil {
			continue
		}
		isUnique := true
		for j := i + 1; j < sampleSize; j++ {
			if ss.Bots[j].Brain == nil {
				continue
			}
			d := neuroDistance(ss.Bots[i].Brain, ss.Bots[j].Brain)
			totalDist += d
			pairs++
			if d < minDist {
				minDist = d
			}
			if d < threshold {
				isUnique = false
			}
		}
		if isUnique {
			uniqueCount++
		}
	}

	avgDist := 0.0
	if pairs > 0 {
		avgDist = totalDist / float64(pairs)
	}
	if minDist == math.MaxFloat64 {
		minDist = 0
	}

	return DiversityMetrics{
		AvgDistance:  avgDist,
		MinDistance:  minDist,
		UniqueCount: uniqueCount,
		Stagnant:    avgDist < 0.03,
	}
}

// neuroDistance computes normalized Euclidean distance between two brains' weights.
func neuroDistance(a, b *NeuroBrain) float64 {
	dist := 0.0
	for k := 0; k < NeuroWeights; k++ {
		d := a.Weights[k] - b.Weights[k]
		dist += d * d
	}
	return math.Sqrt(dist / float64(NeuroWeights))
}

// MeasureGPDiversity measures diversity for GP mode (program structure).
func MeasureGPDiversity(ss *SwarmState) DiversityMetrics {
	n := len(ss.Bots)
	if n < 2 {
		return DiversityMetrics{AvgDistance: 1.0, MinDistance: 1.0, UniqueCount: n}
	}

	sampleSize := n
	if sampleSize > 30 {
		sampleSize = 30
	}

	totalDist := 0.0
	minDist := math.MaxFloat64
	pairs := 0
	threshold := 0.05

	uniqueCount := 0
	for i := 0; i < sampleSize; i++ {
		if ss.Bots[i].OwnProgram == nil {
			continue
		}
		isUnique := true
		for j := i + 1; j < sampleSize; j++ {
			if ss.Bots[j].OwnProgram == nil {
				continue
			}
			d := gpDistance(ss, i, j)
			totalDist += d
			pairs++
			if d < minDist {
				minDist = d
			}
			if d < threshold {
				isUnique = false
			}
		}
		if isUnique {
			uniqueCount++
		}
	}

	avgDist := 0.0
	if pairs > 0 {
		avgDist = totalDist / float64(pairs)
	}
	if minDist == math.MaxFloat64 {
		minDist = 0
	}

	return DiversityMetrics{
		AvgDistance:  avgDist,
		MinDistance:  minDist,
		UniqueCount: uniqueCount,
		Stagnant:    avgDist < 0.05,
	}
}

// MeasureLSTMDiversity measures diversity for LSTM mode (brain weights).
func MeasureLSTMDiversity(ss *SwarmState) DiversityMetrics {
	n := len(ss.Bots)
	if n < 2 {
		return DiversityMetrics{AvgDistance: 1.0, MinDistance: 1.0, UniqueCount: n}
	}

	sampleSize := n
	if sampleSize > 30 {
		sampleSize = 30
	}

	totalDist := 0.0
	minDist := math.MaxFloat64
	pairs := 0
	threshold := 0.02

	uniqueCount := 0
	for i := 0; i < sampleSize; i++ {
		if ss.Bots[i].LSTMBrain == nil {
			continue
		}
		isUnique := true
		for j := i + 1; j < sampleSize; j++ {
			if ss.Bots[j].LSTMBrain == nil {
				continue
			}
			d := lstmDistance(ss.Bots[i].LSTMBrain, ss.Bots[j].LSTMBrain)
			totalDist += d
			pairs++
			if d < minDist {
				minDist = d
			}
			if d < threshold {
				isUnique = false
			}
		}
		if isUnique {
			uniqueCount++
		}
	}

	avgDist := 0.0
	if pairs > 0 {
		avgDist = totalDist / float64(pairs)
	}
	if minDist == math.MaxFloat64 {
		minDist = 0
	}

	return DiversityMetrics{
		AvgDistance:  avgDist,
		MinDistance:  minDist,
		UniqueCount: uniqueCount,
		Stagnant:    avgDist < 0.03,
	}
}

// lstmDistance computes normalized Euclidean distance between two LSTM brains' weights.
func lstmDistance(a, b *LSTMBrain) float64 {
	dist := 0.0
	for k := 0; k < LSTMWeights; k++ {
		d := a.Weights[k] - b.Weights[k]
		dist += d * d
	}
	return math.Sqrt(dist / float64(LSTMWeights))
}

// gpDistance computes a simple structural distance between two GP programs.
func gpDistance(ss *SwarmState, i, j int) float64 {
	pa := ss.Bots[i].OwnProgram
	pb := ss.Bots[j].OwnProgram
	if pa == nil || pb == nil {
		return 1.0
	}

	maxRules := len(pa.Rules)
	if len(pb.Rules) > maxRules {
		maxRules = len(pb.Rules)
	}
	if maxRules == 0 {
		return 0
	}

	// Compare rule-by-rule: action type match + condition match
	matches := 0
	minRules := len(pa.Rules)
	if len(pb.Rules) < minRules {
		minRules = len(pb.Rules)
	}
	for r := 0; r < minRules; r++ {
		if pa.Rules[r].Action.Type == pb.Rules[r].Action.Type {
			matches++
		}
	}

	return 1.0 - float64(matches)/float64(maxRules)
}
