package swarm

import (
	"math"
	"swarmsim/logger"
)

// AutoOptimizerState tracks the auto-optimization process.
type AutoOptimizerState struct {
	Active        bool
	Trial         int // current trial number
	MaxTrials     int // total trials to run
	TicksPerTrial int
	Timer         int // ticks remaining in current trial
	BestScore     float64
	BestParams    [26]float64
	CurrentScore  float64
	Scores        []float64 // score per trial for visualization

	// Convergence detection
	ConvergeWindow   int     // how many trials to look back (default 4)
	ConvergeThresh   float64 // max delta to consider converged (default 5.0)
	Converged        bool    // true if optimizer stopped due to convergence
	ImprovementCount int     // consecutive non-improvement trials

	// Save/restore best config
	SavedConfigs []SavedOptimizerConfig
}

// SavedOptimizerConfig stores a named parameter snapshot.
type SavedOptimizerConfig struct {
	Name   string
	Score  float64
	Params [26]float64
}

// AutoOptimizerStart begins an auto-optimization run.
func AutoOptimizerStart(ss *SwarmState) {
	if ss.AutoOptimizer == nil {
		ss.AutoOptimizer = &AutoOptimizerState{}
	}
	opt := ss.AutoOptimizer
	opt.Active = true
	opt.Trial = 0
	opt.MaxTrials = 10
	opt.TicksPerTrial = 2000
	opt.Timer = opt.TicksPerTrial
	opt.BestScore = -999999
	opt.Scores = nil
	opt.CurrentScore = 0
	opt.Converged = false
	opt.ImprovementCount = 0
	opt.ConvergeWindow = 4
	opt.ConvergeThresh = 5.0

	// Save current best params
	ScanUsedParams(ss)
	for p := 0; p < 26; p++ {
		if ss.UsedParams[p] {
			opt.BestParams[p] = GetParamHint(ss, p)
		}
	}

	// Randomize first trial
	autoOptimizerRandomize(ss)
	logger.Info("OPTIMIZER", "Started: %d trials x %d ticks", opt.MaxTrials, opt.TicksPerTrial)
}

// AutoOptimizerTick advances one tick of auto-optimization.
func AutoOptimizerTick(ss *SwarmState) {
	opt := ss.AutoOptimizer
	if opt == nil || !opt.Active {
		return
	}

	opt.Timer--

	// Accumulate score from delivery performance
	opt.CurrentScore = float64(ss.DeliveryStats.CorrectDelivered)*30 -
		float64(ss.DeliveryStats.WrongDelivered)*10

	if opt.Timer > 0 {
		return
	}

	// Trial complete
	logger.Info("OPTIMIZER", "Trial %d/%d score: %.0f (best: %.0f)",
		opt.Trial+1, opt.MaxTrials, opt.CurrentScore, opt.BestScore)

	opt.Scores = append(opt.Scores, opt.CurrentScore)

	if opt.CurrentScore > opt.BestScore {
		opt.BestScore = opt.CurrentScore
		opt.ImprovementCount = 0
		// Save winning params
		if len(ss.Bots) > 0 {
			for p := 0; p < 26; p++ {
				if ss.UsedParams[p] {
					// Average across all bots
					total := 0.0
					for i := range ss.Bots {
						total += ss.Bots[i].ParamValues[p]
					}
					opt.BestParams[p] = total / float64(len(ss.Bots))
				}
			}
		}
		logger.Info("OPTIMIZER", "New best score: %.0f", opt.BestScore)
	} else {
		opt.ImprovementCount++
	}

	opt.Trial++

	// Convergence detection: if recent scores are all within threshold, stop early
	if opt.Trial >= opt.ConvergeWindow && opt.ConvergeWindow > 0 {
		if CheckConvergence(opt.Scores, opt.ConvergeWindow, opt.ConvergeThresh) {
			opt.Converged = true
			// Apply best and stop
			applyBestParams(ss, opt)
			opt.Active = false
			logger.Info("OPTIMIZER", "Converged after %d trials! Best: %.0f", opt.Trial, opt.BestScore)
			return
		}
	}

	if opt.Trial >= opt.MaxTrials {
		applyBestParams(ss, opt)
		opt.Active = false
		logger.Info("OPTIMIZER", "Complete! Best score: %.0f — params applied", opt.BestScore)
		return
	}

	// Start next trial
	opt.Timer = opt.TicksPerTrial
	opt.CurrentScore = 0
	autoOptimizerRandomize(ss)
}

// applyBestParams writes BestParams to all bots.
func applyBestParams(ss *SwarmState, opt *AutoOptimizerState) {
	for p := 0; p < 26; p++ {
		if ss.UsedParams[p] {
			for i := range ss.Bots {
				ss.Bots[i].ParamValues[p] = opt.BestParams[p]
			}
		}
	}
}

// CheckConvergence returns true if the last `window` scores are all within `threshold` of each other.
func CheckConvergence(scores []float64, window int, threshold float64) bool {
	n := len(scores)
	if n < window || window < 2 {
		return false
	}
	recent := scores[n-window:]
	minS, maxS := recent[0], recent[0]
	for _, s := range recent[1:] {
		if s < minS {
			minS = s
		}
		if s > maxS {
			maxS = s
		}
	}
	return math.Abs(maxS-minS) <= threshold
}

// SaveOptimizerConfig saves the current best params under a name.
func SaveOptimizerConfig(opt *AutoOptimizerState, name string) {
	if opt == nil {
		return
	}
	cfg := SavedOptimizerConfig{
		Name:   name,
		Score:  opt.BestScore,
		Params: opt.BestParams,
	}
	// Replace if name exists
	for i, c := range opt.SavedConfigs {
		if c.Name == name {
			opt.SavedConfigs[i] = cfg
			return
		}
	}
	opt.SavedConfigs = append(opt.SavedConfigs, cfg)
}

// RestoreOptimizerConfig restores saved params by name to all bots.
// Returns false if name not found.
func RestoreOptimizerConfig(ss *SwarmState, name string) bool {
	opt := ss.AutoOptimizer
	if opt == nil {
		return false
	}
	for _, c := range opt.SavedConfigs {
		if c.Name == name {
			for p := 0; p < 26; p++ {
				for i := range ss.Bots {
					ss.Bots[i].ParamValues[p] = c.Params[p]
				}
			}
			return true
		}
	}
	return false
}

// autoOptimizerRandomize randomizes bot params for a new trial.
func autoOptimizerRandomize(ss *SwarmState) {
	opt := ss.AutoOptimizer
	ss.ResetBots()
	ss.DeliveryStats = DeliveryStats{}
	if ss.DeliveryOn {
		ss.ResetDeliveryState()
		GenerateDeliveryStations(ss)
	}

	for i := range ss.Bots {
		for p := 0; p < 26; p++ {
			if !ss.UsedParams[p] {
				continue
			}
			// Mutate around best known params
			base := opt.BestParams[p]
			noise := (ss.Rng.Float64() - 0.5) * 40 // ±20 range
			ss.Bots[i].ParamValues[p] = base + noise
		}
		ss.Bots[i].Fitness = 0
	}
}
