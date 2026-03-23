package swarm

import (
	"math"
	"swarmsim/logger"
)

// BenchmarkState manages standardized swarm benchmarks.
// Each benchmark defines a scenario with specific success criteria,
// enabling objective comparison of different swarm configurations.
type BenchmarkState struct {
	ActiveBenchmark BenchmarkType
	Scenarios       []BenchmarkScenario
	Results         []BenchmarkResult
	CurrentRun      *BenchmarkRun
	BestScores      map[BenchmarkType]float64
}

// BenchmarkType identifies a benchmark scenario.
type BenchmarkType int

const (
	BenchForaging     BenchmarkType = iota // collect and deliver as many packages as possible
	BenchExploration                        // cover as much arena area as possible
	BenchClustering                         // form tight clusters at designated points
	BenchSorting                            // sort packages by type to correct destinations
	BenchScalability                        // performance with increasing bot count
	BenchAdaptation                         // recover from sudden environment changes
	BenchTypeCount
)

// BenchmarkScenario defines a benchmark's setup and scoring.
type BenchmarkScenario struct {
	Type        BenchmarkType
	Name        string
	Description string
	Duration    int     // ticks to run
	BotCount    int     // required bot count (0 = use current)
	ScoreFunc   string  // scoring function identifier
	MaxScore    float64 // theoretical maximum score
}

// BenchmarkRun tracks a running benchmark.
type BenchmarkRun struct {
	Scenario     BenchmarkType
	StartTick    int
	EndTick      int
	Score        float64
	Metrics      BenchmarkMetrics
	IsRunning    bool
	Progress     float64 // 0.0-1.0
}

// BenchmarkMetrics holds detailed performance metrics.
type BenchmarkMetrics struct {
	Deliveries       int
	AreaCovered      float64 // fraction of arena explored
	AvgClusterDist   float64 // average distance to cluster centers
	CollisionCount   int
	EnergyEfficiency float64 // deliveries per energy unit
	Throughput       float64 // deliveries per 1000 ticks
	Convergence      float64 // how quickly bots reached solution
}

// BenchmarkResult is the final result of a completed benchmark run.
type BenchmarkResult struct {
	Scenario  BenchmarkType
	Score     float64
	Metrics   BenchmarkMetrics
	Tick      int // when completed
	NormScore float64 // score / max_score (0-1)
}

// BenchmarkTypeName returns the display name.
func BenchmarkTypeName(bt BenchmarkType) string {
	switch bt {
	case BenchForaging:
		return "Sammeln"
	case BenchExploration:
		return "Erkundung"
	case BenchClustering:
		return "Clustering"
	case BenchSorting:
		return "Sortierung"
	case BenchScalability:
		return "Skalierbarkeit"
	case BenchAdaptation:
		return "Anpassung"
	default:
		return "?"
	}
}

// AllBenchmarkNames returns all benchmark names.
func AllBenchmarkNames() []string {
	names := make([]string, BenchTypeCount)
	for i := BenchmarkType(0); i < BenchTypeCount; i++ {
		names[i] = BenchmarkTypeName(i)
	}
	return names
}

// InitBenchmark sets up the benchmark system.
func InitBenchmark(ss *SwarmState) {
	bs := &BenchmarkState{
		BestScores: make(map[BenchmarkType]float64),
	}

	bs.Scenarios = []BenchmarkScenario{
		{BenchForaging, "Sammeln", "Maximiere Lieferungen in 5000 Ticks", 5000, 0, "deliveries", 500},
		{BenchExploration, "Erkundung", "Erkunde die gesamte Arena in 3000 Ticks", 3000, 0, "area", 1.0},
		{BenchClustering, "Clustering", "Bilde 3 enge Cluster in 2000 Ticks", 2000, 0, "cluster", 100},
		{BenchSorting, "Sortierung", "Liefere Pakete an korrekte Stationen", 5000, 0, "sorting", 200},
		{BenchScalability, "Skalierbarkeit", "Effizienz bei steigender Bot-Zahl", 3000, 0, "scale", 100},
		{BenchAdaptation, "Anpassung", "Erholung nach Umgebungsaenderung", 4000, 0, "adapt", 100},
	}

	ss.Benchmark = bs
	logger.Info("BENCHMARK", "Initialisiert: %d Szenarien", len(bs.Scenarios))
}

// ClearBenchmark disables the benchmark system.
func ClearBenchmark(ss *SwarmState) {
	ss.Benchmark = nil
	ss.BenchmarkOn = false
}

// StartBenchmark begins a benchmark run.
func StartBenchmark(ss *SwarmState, benchType BenchmarkType) {
	bs := ss.Benchmark
	if bs == nil || benchType < 0 || int(benchType) >= len(bs.Scenarios) {
		return
	}

	scenario := bs.Scenarios[benchType]
	bs.ActiveBenchmark = benchType
	bs.CurrentRun = &BenchmarkRun{
		Scenario:  benchType,
		StartTick: ss.Tick,
		EndTick:   ss.Tick + scenario.Duration,
		IsRunning: true,
	}

	// Reset bot stats for fair measurement
	for i := range ss.Bots {
		ss.Bots[i].Stats = BotLifetimeStats{}
		ss.Bots[i].Fitness = 0
	}

	logger.Info("BENCHMARK", "Start: %s (%d Ticks)", scenario.Name, scenario.Duration)
}

// TickBenchmark updates the running benchmark.
func TickBenchmark(ss *SwarmState) {
	bs := ss.Benchmark
	if bs == nil || bs.CurrentRun == nil || !bs.CurrentRun.IsRunning {
		return
	}

	run := bs.CurrentRun
	elapsed := ss.Tick - run.StartTick
	scenario := bs.Scenarios[run.Scenario]
	run.Progress = float64(elapsed) / float64(scenario.Duration)

	// Update metrics continuously
	updateBenchmarkMetrics(ss, run)

	// Check completion
	if ss.Tick >= run.EndTick {
		completeBenchmark(ss, bs, run)
	}
}

// updateBenchmarkMetrics computes current metrics.
func updateBenchmarkMetrics(ss *SwarmState, run *BenchmarkRun) {
	m := &run.Metrics

	// Deliveries
	totalDel := 0
	for i := range ss.Bots {
		totalDel += ss.Bots[i].Stats.TotalDeliveries
	}
	m.Deliveries = totalDel

	// Area coverage: divide arena into grid and check which cells have been visited
	gridSize := 20
	cellW := ss.ArenaW / float64(gridSize)
	cellH := ss.ArenaH / float64(gridSize)
	visited := make([]bool, gridSize*gridSize)
	for i := range ss.Bots {
		gx := int(ss.Bots[i].X / cellW)
		gy := int(ss.Bots[i].Y / cellH)
		if gx >= 0 && gx < gridSize && gy >= 0 && gy < gridSize {
			visited[gy*gridSize+gx] = true
		}
	}
	coveredCount := 0
	for _, v := range visited {
		if v {
			coveredCount++
		}
	}
	m.AreaCovered = float64(coveredCount) / float64(gridSize*gridSize)

	// Throughput
	elapsed := ss.Tick - run.StartTick
	if elapsed > 0 {
		m.Throughput = float64(totalDel) / float64(elapsed) * 1000
	}
}

// completeBenchmark finalizes a benchmark run.
func completeBenchmark(ss *SwarmState, bs *BenchmarkState, run *BenchmarkRun) {
	run.IsRunning = false
	scenario := bs.Scenarios[run.Scenario]

	// Compute final score
	switch run.Scenario {
	case BenchForaging:
		run.Score = float64(run.Metrics.Deliveries)
	case BenchExploration:
		run.Score = run.Metrics.AreaCovered
	case BenchClustering:
		run.Score = computeClusterScore(ss)
	case BenchSorting:
		run.Score = float64(run.Metrics.Deliveries) * 0.8 // simplified
	case BenchScalability:
		if len(ss.Bots) > 0 {
			run.Score = float64(run.Metrics.Deliveries) / float64(len(ss.Bots)) * 10
		}
	case BenchAdaptation:
		run.Score = run.Metrics.Throughput
	}

	// Normalize
	normScore := 0.0
	if scenario.MaxScore > 0 {
		normScore = math.Min(run.Score/scenario.MaxScore, 1.0)
	}

	result := BenchmarkResult{
		Scenario:  run.Scenario,
		Score:     run.Score,
		Metrics:   run.Metrics,
		Tick:      ss.Tick,
		NormScore: normScore,
	}
	bs.Results = append(bs.Results, result)

	// Update best
	if run.Score > bs.BestScores[run.Scenario] {
		bs.BestScores[run.Scenario] = run.Score
	}

	logger.Info("BENCHMARK", "Ergebnis: %s Score=%.1f (%.0f%%)",
		scenario.Name, run.Score, normScore*100)
}

// computeClusterScore evaluates how well bots have formed clusters.
// Uses spatial hash when available for O(n·k) instead of O(n²).
func computeClusterScore(ss *SwarmState) float64 {
	if len(ss.Bots) == 0 {
		return 0
	}

	totalNN := 0.0
	for i := range ss.Bots {
		minDist := math.MaxFloat64

		if ss.Hash != nil {
			// Expanding-radius search: try small radius first, widen if no neighbor found.
			for _, radius := range []float64{50, 150, 400} {
				candidates := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, radius)
				for _, j := range candidates {
					if j == i || j < 0 || j >= len(ss.Bots) {
						continue
					}
					dx := ss.Bots[i].X - ss.Bots[j].X
					dy := ss.Bots[i].Y - ss.Bots[j].Y
					d := dx*dx + dy*dy
					if d < minDist {
						minDist = d
					}
				}
				if minDist < math.MaxFloat64 {
					break
				}
			}
		}

		// Fallback: brute-force if hash unavailable or no neighbor found.
		if minDist == math.MaxFloat64 {
			for j := range ss.Bots {
				if i == j {
					continue
				}
				dx := ss.Bots[i].X - ss.Bots[j].X
				dy := ss.Bots[i].Y - ss.Bots[j].Y
				d := dx*dx + dy*dy
				if d < minDist {
					minDist = d
				}
			}
		}

		totalNN += math.Sqrt(minDist)
	}
	avgNN := totalNN / float64(len(ss.Bots))

	// Lower average NN distance = better clustering
	return math.Max(100-avgNN, 0)
}

// BenchmarkProgress returns 0.0-1.0 progress of current run.
func BenchmarkProgress(bs *BenchmarkState) float64 {
	if bs == nil || bs.CurrentRun == nil {
		return 0
	}
	return bs.CurrentRun.Progress
}

// BenchmarkIsRunning returns whether a benchmark is active.
func BenchmarkIsRunning(bs *BenchmarkState) bool {
	if bs == nil || bs.CurrentRun == nil {
		return false
	}
	return bs.CurrentRun.IsRunning
}

// BenchmarkResultCount returns how many benchmarks have been completed.
func BenchmarkResultCount(bs *BenchmarkState) int {
	if bs == nil {
		return 0
	}
	return len(bs.Results)
}

// BenchmarkBestScore returns the best score for a given benchmark type.
func BenchmarkBestScore(bs *BenchmarkState, bt BenchmarkType) float64 {
	if bs == nil {
		return 0
	}
	return bs.BestScores[bt]
}
