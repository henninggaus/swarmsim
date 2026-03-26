package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"time"
)

// Number of runs per algorithm/landscape combination.
// Higher values reduce variance and give more reliable comparisons.
const benchmarkRunsPerCombo = 5

// runBenchmark executes a headless benchmark of all optimisation algorithms
// on all fitness landscapes with multiple runs per combination.
// Results are averaged and written to benchmark_results.json.
func runBenchmark() {
	fmt.Println("=== SwarmSim Headless Benchmark ===")
	fmt.Printf("Runs pro Kombination: %d\n\n", benchmarkRunsPerCombo)

	algos := benchmarkAlgos()
	landscapes := benchmarkLandscapes()

	ticksPerAlgo := 3000
	totalCombos := len(algos) * len(landscapes)
	totalRuns := totalCombos * benchmarkRunsPerCombo
	comboIdx := 0

	var allResults []comboResult

	benchStart := time.Now()

	for _, fitFunc := range landscapes {
		for _, algo := range algos {
			comboIdx++
			algoName := swarm.SwarmAlgorithmName(algo)
			fitName := swarm.FitnessLandscapeName(fitFunc)

			cr := comboResult{algo: algo, fitFunc: fitFunc}

			for run := 0; run < benchmarkRunsPerCombo; run++ {
				runNum := (comboIdx-1)*benchmarkRunsPerCombo + run + 1
				fmt.Printf("[%d/%d] %s × %s (Run %d/%d) ... ",
					runNum, totalRuns, algoName, fitName, run+1, benchmarkRunsPerCombo)

				// Fresh simulation for each run (different RNG seed)
				s := createHeadlessSim()
				ss := s.SwarmState

				// Init algorithm with this fitness landscape
				swarm.InitSwarmAlgorithm(ss, algo)
				if ss.SwarmAlgo != nil {
					ss.SwarmAlgo.FitnessFunc = fitFunc
					swarm.ReinitFitnessLandscape(ss)
					swarm.ReinitActiveAlgorithm(ss)
				}

				start := time.Now()

				// Run algorithm
				for t := 0; t < ticksPerAlgo; t++ {
					ss.Tick++
					ss.Hash.Clear()
					for i := range ss.Bots {
						ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
					}
					swarm.TickSwarmAlgorithm(ss)
				}

				// Collect results
				swarm.RecordAlgoPerformanceExported(ss)
				exported := swarm.ExportAlgoBenchmarkResults(ss)

				elapsed := time.Since(start)

				if len(exported) > 0 {
					r := exported[len(exported)-1]
					cr.bestRuns = append(cr.bestRuns, r.BestFitness)
					cr.avgRuns = append(cr.avgRuns, r.AvgFitness)
					cr.convRuns = append(cr.convRuns, r.ConvergenceSpeed)
					cr.divRuns = append(cr.divRuns, r.FinalDiversity)
					cr.pertRuns = append(cr.pertRuns, r.Perturbations)
					fmt.Printf("Avg=%.2f Best=%.2f (%.0fms)\n",
						r.AvgFitness, r.BestFitness, float64(elapsed.Milliseconds()))
				} else {
					fmt.Printf("no results (%.0fms)\n", float64(elapsed.Milliseconds()))
				}
			}

			allResults = append(allResults, cr)
		}
	}

	// Build averaged results
	var results []swarm.AlgoBenchmarkResult
	for _, cr := range allResults {
		r := swarm.AlgoBenchmarkResult{
			Algorithm:        swarm.SwarmAlgorithmName(cr.algo),
			AlgoType:         int(cr.algo),
			FitnessFunc:      swarm.FitnessLandscapeName(cr.fitFunc),
			FitFuncType:      int(cr.fitFunc),
			BestFitness:      avgFloat(cr.bestRuns),
			AvgFitness:       avgFloat(cr.avgRuns),
			ConvergenceSpeed: avgFloat(cr.convRuns),
			FinalDiversity:   avgFloat(cr.divRuns),
			Iterations:       ticksPerAlgo / 10, // same as before
			Perturbations:    avgInt(cr.pertRuns),
		}
		results = append(results, r)
	}

	// Write results
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: marshal results: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("benchmark_results.json", data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: write benchmark_results.json: %v\n", err)
		os.Exit(1)
	}

	totalElapsed := time.Since(benchStart)
	fmt.Println()
	fmt.Printf("Ergebnisse: benchmark_results.json (%d Eintraege, %d Runs gemittelt)\n",
		len(results), benchmarkRunsPerCombo)
	fmt.Printf("Gesamtzeit: %s\n", totalElapsed.Round(time.Second))
	fmt.Println()

	// Print summary table with stddev
	printBenchmarkSummaryMulti(allResults)
}

// createHeadlessSim creates a Simulation in swarm mode without any GUI.
func createHeadlessSim() *simulation.Simulation {
	cfg := simulation.DefaultConfig()
	cfg.ArenaWidth = swarm.SwarmArenaSize
	cfg.ArenaHeight = swarm.SwarmArenaSize
	cfg.InitObstacles = 0
	cfg.InitResources = 0
	cfg.InitScouts = 0
	cfg.InitWorkers = 0
	cfg.InitLeaders = 0
	cfg.InitTanks = 0
	cfg.InitHealers = 0
	cfg.ResourceRespawn = false
	cfg.WaveEnabled = false
	cfg.HomeBaseX = -100
	cfg.HomeBaseY = -100
	cfg.HomeBaseR = 0

	s := simulation.NewSimulation(cfg)
	s.SwarmMode = true
	s.SwarmState = swarm.NewSwarmState(s.Rng, swarm.SwarmDefaultBots)
	return s
}

// benchmarkAlgos returns all optimisation algorithms (excluding Boids/ACO
// which don't optimise a fitness function).
func benchmarkAlgos() []swarm.SwarmAlgorithmType {
	var algos []swarm.SwarmAlgorithmType
	for a := swarm.AlgoPSO; a < swarm.AlgoCount; a++ {
		if a == swarm.AlgoACO || a == swarm.AlgoBoids {
			continue
		}
		algos = append(algos, a)
	}
	return algos
}

// benchmarkLandscapes returns all fitness landscape types.
func benchmarkLandscapes() []swarm.FitnessLandscapeType {
	var landscapes []swarm.FitnessLandscapeType
	for f := swarm.FitGaussian; f < swarm.FitCount; f++ {
		landscapes = append(landscapes, f)
	}
	return landscapes
}

// printBenchmarkSummaryMulti prints a table with mean ± stddev.
func printBenchmarkSummaryMulti(results []comboResult) {
	fmt.Println("=== Zusammenfassung (Mittelwert ± Stddev) ===")
	fmt.Printf("%-28s %-16s %14s %14s\n",
		"Algorithmus", "Landschaft", "Best", "Avg")
	fmt.Println("--------------------------------------------------------------------------")
	for _, cr := range results {
		algoName := swarm.SwarmAlgorithmName(cr.algo)
		fitName := swarm.FitnessLandscapeName(cr.fitFunc)
		bestMean, bestStd := meanStddev(cr.bestRuns)
		avgMean, avgStd := meanStddev(cr.avgRuns)
		fmt.Printf("%-28s %-16s %6.2f ± %4.2f  %6.2f ± %4.2f\n",
			algoName, fitName, bestMean, bestStd, avgMean, avgStd)
	}

	// Per-algorithm summary
	fmt.Println()
	fmt.Println("=== Gesamt pro Algorithmus ===")
	fmt.Printf("%-30s %8s %8s\n", "Algorithmus", "Avg", "Stddev")
	fmt.Println("------------------------------------------------")

	type algoSummary struct {
		algo swarm.SwarmAlgorithmType
		avgs []float64
	}
	algoMap := make(map[swarm.SwarmAlgorithmType]*algoSummary)
	var order []swarm.SwarmAlgorithmType

	for _, cr := range results {
		s, ok := algoMap[cr.algo]
		if !ok {
			s = &algoSummary{algo: cr.algo}
			algoMap[cr.algo] = s
			order = append(order, cr.algo)
		}
		s.avgs = append(s.avgs, avgFloat(cr.avgRuns))
	}

	// Sort by average descending
	for i := 0; i < len(order)-1; i++ {
		for j := i + 1; j < len(order); j++ {
			ai := avgFloat(algoMap[order[i]].avgs)
			aj := avgFloat(algoMap[order[j]].avgs)
			if aj > ai {
				order[i], order[j] = order[j], order[i]
			}
		}
	}

	for _, a := range order {
		s := algoMap[a]
		mean, std := meanStddev(s.avgs)
		fmt.Printf("%-30s %8.2f %8.2f\n", swarm.SwarmAlgorithmName(a), mean, std)
	}
}

func avgFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func avgInt(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	sum := 0
	for _, v := range vals {
		sum += v
	}
	return sum / len(vals)
}

func meanStddev(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	mean := avgFloat(vals)
	if len(vals) == 1 {
		return mean, 0
	}
	sumSq := 0.0
	for _, v := range vals {
		d := v - mean
		sumSq += d * d
	}
	return mean, math.Sqrt(sumSq / float64(len(vals)-1))
}

type comboResult struct {
	algo     swarm.SwarmAlgorithmType
	fitFunc  swarm.FitnessLandscapeType
	bestRuns []float64
	avgRuns  []float64
	convRuns []float64
	divRuns  []float64
	pertRuns []int
}
