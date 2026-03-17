package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestNewTimelapseStats(t *testing.T) {
	ts := NewTimelapseStats()
	if ts.WindowSize != 500 {
		t.Errorf("expected window size 500, got %d", ts.WindowSize)
	}
	if ts.MaxWindows != 200 {
		t.Errorf("expected max windows 200, got %d", ts.MaxWindows)
	}
}

func TestTickTimelapseNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	TickTimelapse(nil, ss) // should not panic
}

func TestTickTimelapseEmpty(t *testing.T) {
	ts := NewTimelapseStats()
	ss := &SwarmState{}
	TickTimelapse(ts, ss) // no bots, should not panic
	if ts.Current.Samples != 0 {
		t.Error("no bots should produce no samples")
	}
}

func TestTickTimelapseAccumulates(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ts := NewTimelapseStats()
	for i := 0; i < 10; i++ {
		ss.Tick = i
		TickTimelapse(ts, ss)
	}
	if ts.Current.Samples != 10 {
		t.Errorf("expected 10 samples, got %d", ts.Current.Samples)
	}
}

func TestTickTimelapseWindowComplete(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ts := NewTimelapseStats()
	ts.WindowSize = 10

	for i := 0; i < 25; i++ {
		ss.Tick = i
		// Give bots some fitness
		for j := range ss.Bots {
			ss.Bots[j].Fitness = float64(j)
			ss.Bots[j].Speed = 2.0
		}
		TickTimelapse(ts, ss)
	}

	if len(ts.Windows) == 0 {
		t.Fatal("should have completed at least one window")
	}
	w := ts.Windows[0]
	if w.EndTick == 0 {
		t.Error("window should have end tick")
	}
	if w.AvgSpeed == 0 {
		t.Error("avg speed should be non-zero")
	}
}

func TestTimelapseWindowPruning(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ts := NewTimelapseStats()
	ts.WindowSize = 5
	ts.MaxWindows = 3

	for i := 0; i < 50; i++ {
		ss.Tick = i
		TickTimelapse(ts, ss)
	}
	if len(ts.Windows) > 3 {
		t.Errorf("windows should be pruned to max 3, got %d", len(ts.Windows))
	}
}

func TestTimelapseWindowCount(t *testing.T) {
	if TimelapseWindowCount(nil) != 0 {
		t.Error("nil should return 0")
	}
	ts := &TimelapseStats{Windows: make([]TimeWindow, 5)}
	if TimelapseWindowCount(ts) != 5 {
		t.Errorf("expected 5, got %d", TimelapseWindowCount(ts))
	}
}

func TestTimelapseTrend(t *testing.T) {
	ts := &TimelapseStats{
		Windows: []TimeWindow{
			{AvgFitness: 10},
			{AvgFitness: 20},
			{AvgFitness: 30},
			{AvgFitness: 40},
		},
	}
	trend := TimelapseTrend(ts, func(w TimeWindow) float64 { return w.AvgFitness }, 4)
	if trend <= 0 {
		t.Errorf("expected positive trend, got %f", trend)
	}
	// Linear increase of 10 per step → slope = 10
	if math.Abs(trend-10.0) > 0.1 {
		t.Errorf("expected trend ~10, got %f", trend)
	}
}

func TestTimelapseTrendNil(t *testing.T) {
	if TimelapseTrend(nil, func(w TimeWindow) float64 { return 0 }, 10) != 0 {
		t.Error("nil should return 0")
	}
}

func TestTimelapseTrendFlat(t *testing.T) {
	ts := &TimelapseStats{
		Windows: []TimeWindow{
			{AvgFitness: 50},
			{AvgFitness: 50},
			{AvgFitness: 50},
		},
	}
	trend := TimelapseTrend(ts, func(w TimeWindow) float64 { return w.AvgFitness }, 3)
	if math.Abs(trend) > 0.01 {
		t.Errorf("flat data should have zero trend, got %f", trend)
	}
}

func TestTimelapseAvgMetric(t *testing.T) {
	ts := &TimelapseStats{
		Windows: []TimeWindow{
			{AvgFitness: 10},
			{AvgFitness: 20},
			{AvgFitness: 30},
		},
	}
	avg := TimelapseAvgMetric(ts, func(w TimeWindow) float64 { return w.AvgFitness }, 3)
	if math.Abs(avg-20.0) > 0.01 {
		t.Errorf("expected avg 20, got %f", avg)
	}
}

func TestTimelapseAvgMetricNil(t *testing.T) {
	if TimelapseAvgMetric(nil, func(w TimeWindow) float64 { return 0 }, 5) != 0 {
		t.Error("nil should return 0")
	}
}

func TestTimelapseMaxMetric(t *testing.T) {
	ts := &TimelapseStats{
		Windows: []TimeWindow{
			{MaxFitness: 100},
			{MaxFitness: 200},
			{MaxFitness: 150},
		},
	}
	m := TimelapseMaxMetric(ts, func(w TimeWindow) float64 { return w.MaxFitness })
	if m != 200 {
		t.Errorf("expected max 200, got %f", m)
	}
}

func TestTimelapseMaxMetricNil(t *testing.T) {
	if TimelapseMaxMetric(nil, func(w TimeWindow) float64 { return 0 }) != 0 {
		t.Error("nil should return 0")
	}
}

func TestTimelapseMinFitnessDefault(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ts := NewTimelapseStats()
	ts.WindowSize = 3
	for i := 0; i < 5; i++ {
		ss.Tick = i
		TickTimelapse(ts, ss)
	}
	if len(ts.Windows) > 0 && ts.Windows[0].MinFitness == math.MaxFloat64 {
		t.Error("finalized min fitness should not be MaxFloat64")
	}
}
