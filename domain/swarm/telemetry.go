package swarm

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// TelemetryWriter writes periodic simulation metrics to a JSONL file.
// Each line is a self-contained JSON object with a timestamp, tick number,
// active algorithm, fitness landscape, and key performance indicators.
type TelemetryWriter struct {
	file     *os.File
	interval int // write every N ticks
	lastTick int // last tick written
}

// TelemetrySample is one line in the telemetry JSONL file.
type TelemetrySample struct {
	Timestamp     string  `json:"ts"`
	Tick          int     `json:"tick"`
	Algorithm     string  `json:"algo"`
	AlgoType      int     `json:"algo_type"`
	FitnessFunc   string  `json:"fitness_func"`
	FitFuncType   int     `json:"fit_func_type"`
	BestFitness   float64 `json:"best_fitness"`
	AvgFitness    float64 `json:"avg_fitness"`
	Diversity     float64 `json:"diversity"`
	ExplRatio     float64 `json:"expl_ratio"`
	StagnCount    int     `json:"stagn_count"`
	PerturbCount  int     `json:"perturb_count"`
	Iterations    int     `json:"iterations"`
	BotCount      int     `json:"bot_count"`
	BestEver      float64 `json:"best_ever"`
	TournamentOn  bool    `json:"tournament_on,omitempty"`
	TournamentPct float64 `json:"tournament_pct,omitempty"`
}

// NewTelemetryWriter creates a new telemetry writer that appends to the given
// file path. The interval controls how often samples are written (in ticks).
func NewTelemetryWriter(path string, interval int) (*TelemetryWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("telemetry open: %w", err)
	}
	if interval < 1 {
		interval = 10
	}
	return &TelemetryWriter{file: f, interval: interval}, nil
}

// Sample checks if it's time to write and, if so, captures the current state.
func (tw *TelemetryWriter) Sample(ss *SwarmState) {
	if tw == nil || tw.file == nil || ss == nil {
		return
	}
	if ss.Tick-tw.lastTick < tw.interval {
		return
	}
	tw.lastTick = ss.Tick

	s := TelemetrySample{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Tick:      ss.Tick,
		BotCount:  len(ss.Bots),
	}

	if ss.SwarmAlgo != nil {
		sa := ss.SwarmAlgo
		s.Algorithm = SwarmAlgorithmName(sa.ActiveAlgo)
		s.AlgoType = int(sa.ActiveAlgo)
		s.FitnessFunc = FitnessLandscapeName(sa.FitnessFunc)
		s.FitFuncType = int(sa.FitnessFunc)
		s.BestFitness = GetAlgoBestFitness(ss)
		s.AvgFitness = GetAlgoAvgFitness(ss)
		s.Diversity = GetAlgoDiversity(ss)
		s.ExplRatio = GetAlgoExplorationRatio(ss)
		s.StagnCount = sa.StagnationCount
		s.PerturbCount = sa.PerturbationCount
		s.Iterations = sa.TotalIterations
		s.BestEver = sa.BestFitnessEver
	}

	if ss.AlgoTournamentOn {
		s.TournamentOn = true
		if ss.AlgoTournamentTotal > 0 {
			s.TournamentPct = float64(ss.AlgoTournamentDone) / float64(ss.AlgoTournamentTotal) * 100
		}
	}

	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	tw.file.Write(data)
	tw.file.Write([]byte("\n"))
}

// Close flushes and closes the telemetry file.
func (tw *TelemetryWriter) Close() error {
	if tw == nil || tw.file == nil {
		return nil
	}
	return tw.file.Close()
}

// AlgoBenchmarkResult is the final summary written to benchmark_results.json.
type AlgoBenchmarkResult struct {
	Algorithm        string  `json:"algorithm"`
	AlgoType         int     `json:"algo_type"`
	FitnessFunc      string  `json:"fitness_func"`
	FitFuncType      int     `json:"fit_func_type"`
	BestFitness      float64 `json:"best_fitness"`
	AvgFitness       float64 `json:"avg_fitness"`
	ConvergenceSpeed float64 `json:"convergence_speed"`
	FinalDiversity   float64 `json:"final_diversity"`
	Iterations       int     `json:"iterations"`
	Perturbations    int     `json:"perturbations"`
}

// ExportAlgoBenchmarkResults converts the scoreboard into a JSON-serialisable slice.
func ExportAlgoBenchmarkResults(ss *SwarmState) []AlgoBenchmarkResult {
	var results []AlgoBenchmarkResult
	for _, r := range ss.AlgoScoreboard {
		results = append(results, AlgoBenchmarkResult{
			Algorithm:        SwarmAlgorithmName(r.Algo),
			AlgoType:         int(r.Algo),
			FitnessFunc:      FitnessLandscapeName(r.FitnessFunc),
			FitFuncType:      int(r.FitnessFunc),
			BestFitness:      r.BestFitness,
			AvgFitness:       r.AvgFitness,
			ConvergenceSpeed: r.ConvergenceSpeed,
			FinalDiversity:   r.FinalDiversity,
			Iterations:       r.Iterations,
			Perturbations:    r.Perturbations,
		})
	}
	return results
}
