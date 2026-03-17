package swarm

import (
	"math"
	"math/rand"
	"sort"
)

// SensitivityParam describes one parameter to vary.
type SensitivityParam struct {
	Name     string
	Index    int     // index into ParamValues[26]
	BaseVal  float64 // current/center value
	MinVal   float64
	MaxVal   float64
}

// SensitivityResult stores the outcome of varying one parameter.
type SensitivityResult struct {
	ParamName   string
	ParamIndex  int
	Impact      float64 // absolute fitness change when varied
	Direction   int     // +1 = increasing helps, -1 = decreasing helps, 0 = neutral
	BestVal     float64 // value that produced best fitness
	BaseFitness float64 // fitness at base value
	BestFitness float64 // fitness at best value
}

// SensitivityConfig configures the analysis run.
type SensitivityConfig struct {
	Steps     int     // number of values to test per param (default 5)
	DeltaPct  float64 // percentage variation around base (0.2 = +-20%)
	EvalFunc  func(bot *SwarmBot) float64
}

// DefaultSensitivityConfig returns a reasonable default config.
func DefaultSensitivityConfig() SensitivityConfig {
	return SensitivityConfig{
		Steps:    5,
		DeltaPct: 0.3,
		EvalFunc: EvaluateGPFitness,
	}
}

// SensitivityReport contains all results from a sensitivity analysis.
type SensitivityReport struct {
	Results       []SensitivityResult
	MostSensitive string  // param name with highest impact
	LeastSensitive string // param name with lowest impact
	TotalImpact   float64 // sum of all impacts (for normalization)
}

// RunSensitivityAnalysis performs one-at-a-time sensitivity analysis.
// For each parameter, it varies the value across steps while keeping others at base,
// and measures the fitness impact using the provided evaluator.
func RunSensitivityAnalysis(rng *rand.Rand, bots []SwarmBot, params []SensitivityParam, cfg SensitivityConfig) *SensitivityReport {
	if len(params) == 0 || len(bots) == 0 {
		return &SensitivityReport{}
	}
	if cfg.Steps < 2 {
		cfg.Steps = 2
	}
	if cfg.EvalFunc == nil {
		cfg.EvalFunc = EvaluateGPFitness
	}
	if cfg.DeltaPct <= 0 {
		cfg.DeltaPct = 0.3
	}

	results := make([]SensitivityResult, len(params))

	for pi, param := range params {
		// Compute base fitness (average across bots)
		baseFit := avgFitness(bots, cfg.EvalFunc)

		// Generate test values
		lo := param.BaseVal * (1.0 - cfg.DeltaPct)
		hi := param.BaseVal * (1.0 + cfg.DeltaPct)
		if lo < param.MinVal {
			lo = param.MinVal
		}
		if hi > param.MaxVal {
			hi = param.MaxVal
		}
		if lo > hi {
			lo, hi = hi, lo
		}

		bestFit := baseFit
		bestVal := param.BaseVal
		fitAtLow := baseFit
		fitAtHigh := baseFit

		for s := 0; s < cfg.Steps; s++ {
			testVal := lo + (hi-lo)*float64(s)/float64(cfg.Steps-1)

			// Temporarily set all bots to this param value
			origVals := make([]float64, len(bots))
			for i := range bots {
				origVals[i] = bots[i].ParamValues[param.Index]
				bots[i].ParamValues[param.Index] = testVal
			}

			fit := avgFitness(bots, cfg.EvalFunc)

			// Restore
			for i := range bots {
				bots[i].ParamValues[param.Index] = origVals[i]
			}

			if fit > bestFit {
				bestFit = fit
				bestVal = testVal
			}
			if s == 0 {
				fitAtLow = fit
			}
			if s == cfg.Steps-1 {
				fitAtHigh = fit
			}
		}

		impact := math.Abs(bestFit - baseFit)
		direction := 0
		if fitAtHigh > fitAtLow+0.01 {
			direction = 1
		} else if fitAtLow > fitAtHigh+0.01 {
			direction = -1
		}

		results[pi] = SensitivityResult{
			ParamName:   param.Name,
			ParamIndex:  param.Index,
			Impact:      impact,
			Direction:   direction,
			BestVal:     bestVal,
			BaseFitness: baseFit,
			BestFitness: bestFit,
		}
	}

	// Sort by impact (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Impact > results[j].Impact
	})

	report := &SensitivityReport{
		Results: results,
	}
	if len(results) > 0 {
		report.MostSensitive = results[0].ParamName
		report.LeastSensitive = results[len(results)-1].ParamName
		for _, r := range results {
			report.TotalImpact += r.Impact
		}
	}
	return report
}

// NormalizedImpact returns each result's impact as a fraction of total (0..1).
func (sr *SensitivityReport) NormalizedImpact() []float64 {
	if sr.TotalImpact == 0 || len(sr.Results) == 0 {
		return make([]float64, len(sr.Results))
	}
	norm := make([]float64, len(sr.Results))
	for i, r := range sr.Results {
		norm[i] = r.Impact / sr.TotalImpact
	}
	return norm
}

func avgFitness(bots []SwarmBot, eval func(*SwarmBot) float64) float64 {
	if len(bots) == 0 {
		return 0
	}
	total := 0.0
	for i := range bots {
		total += eval(&bots[i])
	}
	return total / float64(len(bots))
}
