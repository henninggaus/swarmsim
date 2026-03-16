package swarm

import (
	"swarmsim/logger"
)

// AutoOptimizerState tracks the auto-optimization process.
type AutoOptimizerState struct {
	Active       bool
	Trial        int // current trial number
	MaxTrials    int // total trials to run
	TicksPerTrial int
	Timer        int // ticks remaining in current trial
	BestScore    float64
	BestParams   [26]float64
	CurrentScore float64
	Scores       []float64 // score per trial for visualization
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
	}

	opt.Trial++
	if opt.Trial >= opt.MaxTrials {
		// Done — apply best params
		for p := 0; p < 26; p++ {
			if ss.UsedParams[p] {
				for i := range ss.Bots {
					ss.Bots[i].ParamValues[p] = opt.BestParams[p]
				}
			}
		}
		opt.Active = false
		logger.Info("OPTIMIZER", "Complete! Best score: %.0f — params applied", opt.BestScore)
		return
	}

	// Start next trial
	opt.Timer = opt.TicksPerTrial
	opt.CurrentScore = 0
	autoOptimizerRandomize(ss)
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
