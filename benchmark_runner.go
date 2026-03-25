package main

import (
	"encoding/json"
	"fmt"
	"os"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"time"
)

// runBenchmark executes a headless benchmark of all optimisation algorithms
// on all fitness landscapes and writes results to benchmark_results.json
// and telemetry to telemetry.jsonl. No GUI is started.
func runBenchmark() {
	fmt.Println("=== SwarmSim Headless Benchmark ===")
	fmt.Println()

	// Create simulation in swarm mode
	s := createHeadlessSim()
	ss := s.SwarmState

	// Open telemetry writer
	tw, err := swarm.NewTelemetryWriter("telemetry.jsonl", 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot open telemetry.jsonl: %v\n", err)
		os.Exit(1)
	}
	defer tw.Close()
	ss.Telemetry = tw

	algos := benchmarkAlgos()
	landscapes := benchmarkLandscapes()

	ticksPerAlgo := 3000
	totalRuns := len(algos) * len(landscapes)
	runIdx := 0

	for _, fitFunc := range landscapes {
		// For Gaussian Peaks: generate peaks once using the first algorithm,
		// then reuse the SAME peaks for all subsequent algorithms so the
		// comparison is fair (all algorithms optimise the same landscape).
		var savedPeaks *swarm.GaussianPeaks

		for _, algo := range algos {
			runIdx++
			algoName := swarm.SwarmAlgorithmName(algo)
			fitName := swarm.FitnessLandscapeName(fitFunc)
			fmt.Printf("[%d/%d] %s auf %s ... ", runIdx, totalRuns, algoName, fitName)

			// Init algorithm with this fitness landscape.
			swarm.InitSwarmAlgorithm(ss, algo)
			if ss.SwarmAlgo != nil {
				ss.SwarmAlgo.FitnessFunc = fitFunc
				if fitFunc == swarm.FitGaussian && savedPeaks != nil {
					// Restore the same peaks generated for the first algorithm
					swarm.RestoreGaussianPeaks(ss, savedPeaks)
				} else {
					swarm.ReinitFitnessLandscape(ss)
				}
				swarm.ReinitActiveAlgorithm(ss)

				// Save peaks after first algorithm generates them
				if fitFunc == swarm.FitGaussian && savedPeaks == nil {
					savedPeaks = swarm.SaveGaussianPeaks(ss)
				}
			}

			start := time.Now()

			// Run algorithm for ticksPerAlgo ticks using the centralized dispatch.
			// This bypasses the full simulation loop (which requires SwarmScript
			// programs) and directly drives the algorithm via the registry.
			for t := 0; t < ticksPerAlgo; t++ {
				ss.Tick++

				// Rebuild spatial hash (required for neighbour queries)
				ss.Hash.Clear()
				for i := range ss.Bots {
					ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
				}

				// Tick the algorithm (global update + per-bot steering + convergence recording)
				swarm.TickSwarmAlgorithm(ss)

				// Sample telemetry
				tw.Sample(ss)
			}

			bestFit := swarm.GetAlgoBestFitness(ss)
			elapsed := time.Since(start)

			fmt.Printf("Best=%.2f (%.0fms)\n", bestFit, float64(elapsed.Milliseconds()))

			// Record performance into scoreboard
			swarm.RecordAlgoPerformanceExported(ss)
		}
	}

	// Export results
	results := swarm.ExportAlgoBenchmarkResults(ss)
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: marshal results: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("benchmark_results.json", data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: write benchmark_results.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("Ergebnisse: benchmark_results.json (%d Eintraege)\n", len(results))
	fmt.Printf("Telemetrie: telemetry.jsonl\n")
	fmt.Println()

	// Print summary table
	printBenchmarkSummary(results)
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

// printBenchmarkSummary prints a compact table of results.
func printBenchmarkSummary(results []swarm.AlgoBenchmarkResult) {
	fmt.Println("=== Zusammenfassung ===")
	fmt.Printf("%-28s %-14s %8s %8s %6s %5s\n",
		"Algorithmus", "Landschaft", "Best", "Avg", "Conv", "Pert")
	fmt.Println("--------------------------------------------------------------------------")
	for _, r := range results {
		fmt.Printf("%-28s %-14s %8.2f %8.2f %6.0f %5d\n",
			r.Algorithm, r.FitnessFunc, r.BestFitness, r.AvgFitness,
			r.ConvergenceSpeed, r.Perturbations)
	}
}
