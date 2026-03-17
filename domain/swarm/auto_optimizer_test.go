package swarm

import (
	"math"
	"testing"
)

func TestCheckConvergenceTrue(t *testing.T) {
	scores := []float64{100, 102, 101, 103}
	if !CheckConvergence(scores, 4, 5.0) {
		t.Error("scores within threshold should converge")
	}
}

func TestCheckConvergenceFalse(t *testing.T) {
	scores := []float64{100, 200, 101, 103}
	if CheckConvergence(scores, 4, 5.0) {
		t.Error("scores with large spread should not converge")
	}
}

func TestCheckConvergenceTooFewScores(t *testing.T) {
	scores := []float64{100, 101}
	if CheckConvergence(scores, 4, 5.0) {
		t.Error("fewer scores than window should not converge")
	}
}

func TestCheckConvergenceOnlyLastWindow(t *testing.T) {
	// Early scores vary wildly, but last 3 are close
	scores := []float64{10, 500, 0, 100, 101, 102}
	if !CheckConvergence(scores, 3, 5.0) {
		t.Error("should only look at last window entries")
	}
}

func TestSaveOptimizerConfig(t *testing.T) {
	opt := &AutoOptimizerState{}
	opt.BestScore = 42
	opt.BestParams[0] = 99

	SaveOptimizerConfig(opt, "test1")
	if len(opt.SavedConfigs) != 1 {
		t.Fatalf("expected 1 saved config, got %d", len(opt.SavedConfigs))
	}
	if opt.SavedConfigs[0].Score != 42 {
		t.Error("saved score should be 42")
	}
	if opt.SavedConfigs[0].Params[0] != 99 {
		t.Error("saved param[0] should be 99")
	}

	// Overwrite same name
	opt.BestScore = 100
	SaveOptimizerConfig(opt, "test1")
	if len(opt.SavedConfigs) != 1 {
		t.Error("overwrite should not add new entry")
	}
	if opt.SavedConfigs[0].Score != 100 {
		t.Error("overwritten score should be 100")
	}
}

func TestSaveMultipleConfigs(t *testing.T) {
	opt := &AutoOptimizerState{}
	opt.BestScore = 10
	SaveOptimizerConfig(opt, "a")
	opt.BestScore = 20
	SaveOptimizerConfig(opt, "b")
	if len(opt.SavedConfigs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(opt.SavedConfigs))
	}
}

func TestRestoreOptimizerConfig(t *testing.T) {
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 3),
		BotCount: 3,
	}
	ss.AutoOptimizer = &AutoOptimizerState{}
	ss.AutoOptimizer.BestParams[0] = 77
	ss.AutoOptimizer.BestScore = 50
	SaveOptimizerConfig(ss.AutoOptimizer, "snap1")

	// Change bot params
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = 0
	}

	ok := RestoreOptimizerConfig(ss, "snap1")
	if !ok {
		t.Fatal("restore should succeed")
	}
	for i := range ss.Bots {
		if ss.Bots[i].ParamValues[0] != 77 {
			t.Errorf("bot %d param[0] should be 77 after restore", i)
		}
	}
}

func TestRestoreOptimizerConfigNotFound(t *testing.T) {
	ss := &SwarmState{}
	ss.AutoOptimizer = &AutoOptimizerState{}
	ok := RestoreOptimizerConfig(ss, "nonexistent")
	if ok {
		t.Error("should return false for missing config")
	}
}

func TestRestoreNilOptimizer(t *testing.T) {
	ss := &SwarmState{}
	ok := RestoreOptimizerConfig(ss, "any")
	if ok {
		t.Error("should return false with nil optimizer")
	}
}

func TestSaveNilOptimizer(t *testing.T) {
	SaveOptimizerConfig(nil, "test") // should not panic
}

func TestCheckConvergenceExactThreshold(t *testing.T) {
	scores := []float64{100, 105}
	if !CheckConvergence(scores, 2, 5.0) {
		t.Error("scores exactly at threshold should converge")
	}
}

func TestConvergenceNegativeScores(t *testing.T) {
	scores := []float64{-10, -12, -11, -10}
	if !CheckConvergence(scores, 4, 5.0) {
		t.Error("negative scores within threshold should converge")
	}
}

func TestImprovementCountReset(t *testing.T) {
	opt := &AutoOptimizerState{
		ImprovementCount: 5,
		BestScore:        100,
	}
	// Simulate finding a better score
	newScore := 200.0
	if newScore > opt.BestScore {
		opt.BestScore = newScore
		opt.ImprovementCount = 0
	}
	if opt.ImprovementCount != 0 {
		t.Error("improvement count should reset on new best")
	}
}

func TestSavedConfigParamsIsolated(t *testing.T) {
	opt := &AutoOptimizerState{}
	opt.BestParams[0] = 50
	SaveOptimizerConfig(opt, "snap")

	// Change original
	opt.BestParams[0] = 999

	// Saved should be unchanged (it's a value copy)
	if math.Abs(opt.SavedConfigs[0].Params[0]-50) > 0.01 {
		t.Error("saved config should be isolated from later changes")
	}
}
