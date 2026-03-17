package swarm

import (
	"math/rand"
	"testing"
)

func TestInitBenchmark(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitBenchmark(ss)

	if ss.Benchmark == nil {
		t.Fatal("benchmark should be initialized")
	}
	if len(ss.Benchmark.Scenarios) != int(BenchTypeCount) {
		t.Fatalf("expected %d scenarios, got %d", BenchTypeCount, len(ss.Benchmark.Scenarios))
	}
}

func TestClearBenchmark(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.BenchmarkOn = true
	InitBenchmark(ss)
	ClearBenchmark(ss)

	if ss.Benchmark != nil {
		t.Fatal("should be nil after clear")
	}
	if ss.BenchmarkOn {
		t.Fatal("should be false")
	}
}

func TestStartBenchmark(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitBenchmark(ss)
	ss.Tick = 100

	StartBenchmark(ss, BenchForaging)

	run := ss.Benchmark.CurrentRun
	if run == nil {
		t.Fatal("current run should exist")
	}
	if !run.IsRunning {
		t.Fatal("should be running")
	}
	if run.StartTick != 100 {
		t.Fatalf("expected start tick 100, got %d", run.StartTick)
	}
	if run.EndTick != 5100 {
		t.Fatalf("expected end tick 5100, got %d", run.EndTick)
	}
}

func TestTickBenchmark(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitBenchmark(ss)

	StartBenchmark(ss, BenchExploration)

	for tick := 0; tick < 100; tick++ {
		ss.Tick = tick
		TickBenchmark(ss)
	}

	run := ss.Benchmark.CurrentRun
	if run.Progress <= 0 {
		t.Fatal("progress should be > 0 after 100 ticks")
	}
	if !run.IsRunning {
		t.Fatal("should still be running")
	}
}

func TestTickBenchmarkNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickBenchmark(ss) // should not panic
}

func TestBenchmarkCompletion(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitBenchmark(ss)

	StartBenchmark(ss, BenchForaging)

	// Give some bots deliveries
	for i := 0; i < 5; i++ {
		ss.Bots[i].Stats.TotalDeliveries = 10
	}

	// Advance to end
	run := ss.Benchmark.CurrentRun
	ss.Tick = run.EndTick
	TickBenchmark(ss)

	if run.IsRunning {
		t.Fatal("should be complete")
	}
	if len(ss.Benchmark.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ss.Benchmark.Results))
	}
	if ss.Benchmark.Results[0].Score != 50 {
		t.Fatalf("expected score 50 (5*10), got %.0f", ss.Benchmark.Results[0].Score)
	}
}

func TestBenchmarkTypeName(t *testing.T) {
	if BenchmarkTypeName(BenchForaging) != "Sammeln" {
		t.Fatal("expected Sammeln")
	}
	if BenchmarkTypeName(BenchAdaptation) != "Anpassung" {
		t.Fatal("expected Anpassung")
	}
	if BenchmarkTypeName(99) != "?" {
		t.Fatal("expected ?")
	}
}

func TestAllBenchmarkNames(t *testing.T) {
	names := AllBenchmarkNames()
	if len(names) != int(BenchTypeCount) {
		t.Fatalf("expected %d names, got %d", BenchTypeCount, len(names))
	}
}

func TestBenchmarkProgress(t *testing.T) {
	if BenchmarkProgress(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBenchmarkIsRunning(t *testing.T) {
	if BenchmarkIsRunning(nil) {
		t.Fatal("nil should return false")
	}
}

func TestBenchmarkResultCount(t *testing.T) {
	if BenchmarkResultCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBenchmarkBestScore(t *testing.T) {
	if BenchmarkBestScore(nil, BenchForaging) != 0 {
		t.Fatal("nil should return 0")
	}

	bs := &BenchmarkState{
		BestScores: map[BenchmarkType]float64{BenchForaging: 42},
	}
	if BenchmarkBestScore(bs, BenchForaging) != 42 {
		t.Fatal("expected 42")
	}
}

func TestComputeClusterScore(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)

	// Place all bots at same spot
	for i := range ss.Bots {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
	}

	score := computeClusterScore(ss)
	if score < 90 {
		t.Fatalf("co-located bots should have high cluster score, got %.1f", score)
	}
}
