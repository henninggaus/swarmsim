package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestSwarmAlgorithmName(t *testing.T) {
	if SwarmAlgorithmName(AlgoBoids) != "Boids (Reynolds)" {
		t.Fatal("wrong name for Boids")
	}
	if SwarmAlgorithmName(AlgoNone) != "Keiner" {
		t.Fatal("wrong name for None")
	}
}

func TestInitSwarmAlgorithmBoids(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSwarmAlgorithm(ss, AlgoBoids)

	if ss.SwarmAlgo == nil {
		t.Fatal("swarm algo should be initialized")
	}
	if ss.SwarmAlgo.ActiveAlgo != AlgoBoids {
		t.Fatal("should be Boids")
	}
}

func TestTickBoids(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSwarmAlgorithm(ss, AlgoBoids)

	// Run a few ticks
	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	// Bots should be moving
	moving := false
	for _, bot := range ss.Bots {
		if bot.Speed > 0 {
			moving = true
			break
		}
	}
	if !moving {
		t.Fatal("bots should be moving after Boids ticks")
	}
}

func TestInitSwarmAlgorithmPSO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSwarmAlgorithm(ss, AlgoPSO)

	if ss.PSO == nil {
		t.Fatal("PSO state should be initialized")
	}
	if len(ss.PSO.VelX) != 20 {
		t.Fatalf("expected 20 particles, got %d", len(ss.PSO.VelX))
	}
}

func TestTickPSO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400
	InitSwarmAlgorithm(ss, AlgoPSO)

	for i := 0; i < 50; i++ {
		ss.Tick = i
		TickSwarmAlgorithm(ss)
	}
	// Global best should be updated (dedicated PSO uses Gaussian peaks)
	if ss.PSO.GlobalFit <= 0 {
		t.Fatal("global best fitness should be positive after ticks")
	}
}

func TestInitSwarmAlgorithmACO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSwarmAlgorithm(ss, AlgoACO)

	if ss.ACO == nil {
		t.Fatal("ACO state should be initialized")
	}
	if ss.ACO.GridCols <= 0 {
		t.Fatal("ACO grid should have columns")
	}
}

func TestTickACO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	// Set a bot as carrying so pheromone is deposited
	InitSwarmAlgorithm(ss, AlgoACO)
	ss.Bots[0].CarryingPkg = 1

	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	// Some pheromone should have been deposited by the carrying bot
	hasPher := false
	for _, v := range ss.ACO.Trail {
		if v > 0 {
			hasPher = true
			break
		}
	}
	if !hasPher {
		t.Fatal("should have pheromone deposited")
	}
}

func TestTickFirefly(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ss.Light.Active = true
	ss.Light.X = 400
	ss.Light.Y = 400
	InitSwarmAlgorithm(ss, AlgoFirefly)

	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	// All bots should have some brightness
	for _, b := range ss.SwarmAlgo.FireflyBrightness {
		if math.IsNaN(b) {
			t.Fatal("brightness should not be NaN")
		}
	}
}

func TestHsvToRGB(t *testing.T) {
	r, g, b := hsvToRGB(0, 1, 1) // pure red
	if r != 255 || g != 0 || b != 0 {
		t.Fatalf("expected (255,0,0) got (%d,%d,%d)", r, g, b)
	}

	r, g, b = hsvToRGB(0.333, 1, 1) // green-ish
	if g < 200 {
		t.Fatalf("expected green > 200, got %d", g)
	}
}

func TestAlgorithmNames(t *testing.T) {
	names := AlgorithmNames()
	if len(names) != int(AlgoCount) {
		t.Fatalf("expected %d names, got %d", AlgoCount, len(names))
	}
}

func TestInitAndTickGWO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitSwarmAlgorithm(ss, AlgoGWO)
	if ss.GWO == nil {
		t.Fatal("GWO state should be initialized")
	}
	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	moving := false
	for _, bot := range ss.Bots {
		if bot.Speed > 0 {
			moving = true
			break
		}
	}
	if !moving {
		t.Fatal("bots should be moving after GWO ticks")
	}
	ClearSwarmAlgorithm(ss)
	if ss.GWO != nil {
		t.Fatal("GWO state should be cleared")
	}
}

func TestInitAndTickWOA(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitSwarmAlgorithm(ss, AlgoWOA)
	if ss.WOA == nil {
		t.Fatal("WOA state should be initialized")
	}
	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	ClearSwarmAlgorithm(ss)
	if ss.WOA != nil {
		t.Fatal("WOA state should be cleared")
	}
}

func TestInitAndTickBFO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitSwarmAlgorithm(ss, AlgoBFO)
	if ss.BFO == nil {
		t.Fatal("BFO state should be initialized")
	}
	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	ClearSwarmAlgorithm(ss)
	if ss.BFO != nil {
		t.Fatal("BFO state should be cleared")
	}
}

func TestInitAndTickMFO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitSwarmAlgorithm(ss, AlgoMFO)
	if ss.MFO == nil {
		t.Fatal("MFO state should be initialized")
	}
	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	ClearSwarmAlgorithm(ss)
	if ss.MFO != nil {
		t.Fatal("MFO state should be cleared")
	}
}

func TestInitAndTickCuckoo(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitSwarmAlgorithm(ss, AlgoCuckoo)
	if ss.Cuckoo == nil {
		t.Fatal("Cuckoo state should be initialized")
	}
	for i := 0; i < 10; i++ {
		TickSwarmAlgorithm(ss)
	}
	ClearSwarmAlgorithm(ss)
	if ss.Cuckoo != nil {
		t.Fatal("Cuckoo state should be cleared")
	}
}

func TestClearPSOAndACO(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)

	InitSwarmAlgorithm(ss, AlgoPSO)
	if ss.PSO == nil {
		t.Fatal("PSO should be set")
	}
	ClearSwarmAlgorithm(ss)
	if ss.PSO != nil {
		t.Fatal("PSO should be cleared")
	}

	InitSwarmAlgorithm(ss, AlgoACO)
	if ss.ACO == nil {
		t.Fatal("ACO should be set")
	}
	ClearSwarmAlgorithm(ss)
	if ss.ACO != nil {
		t.Fatal("ACO should be cleared")
	}
}

func TestSwitchAlgorithmClearsOld(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSwarmAlgorithm(ss, AlgoGWO)
	if ss.GWO == nil {
		t.Fatal("GWO should be set")
	}
	// Switching to WOA should clear GWO
	InitSwarmAlgorithm(ss, AlgoWOA)
	if ss.GWO != nil {
		t.Fatal("GWO should be cleared when switching to WOA")
	}
	if ss.WOA == nil {
		t.Fatal("WOA should be set")
	}
}

func TestRecordAlgoPerformanceExtendedMetrics(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitSwarmAlgorithm(ss, AlgoPSO)

	// Simulate some convergence history
	sa := ss.SwarmAlgo
	sa.BestFitnessEver = 80.0
	sa.TotalIterations = 50
	sa.PerturbationCount = 2
	sa.ConvergenceHistory = make([]float64, 50)
	for i := range sa.ConvergenceHistory {
		// Simulate gradual improvement reaching 90% (72.0) at iteration 15
		sa.ConvergenceHistory[i] = float64(i) * 1.6
		if sa.ConvergenceHistory[i] > 80 {
			sa.ConvergenceHistory[i] = 80
		}
	}

	// Spread bots around so diversity is non-zero
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i) * 40
		ss.Bots[i].Y = float64(i) * 30
	}

	// Switch algorithm to trigger recordAlgoPerformance
	InitSwarmAlgorithm(ss, AlgoGWO)

	if len(ss.AlgoScoreboard) != 1 {
		t.Fatalf("expected 1 scoreboard entry, got %d", len(ss.AlgoScoreboard))
	}
	rec := ss.AlgoScoreboard[0]
	if rec.Algo != AlgoPSO {
		t.Fatalf("expected PSO, got %v", rec.Algo)
	}
	if rec.BestFitness != 80.0 {
		t.Errorf("expected best fitness 80.0, got %.1f", rec.BestFitness)
	}
	// ConvergenceSpeed: 90% of 80 = 72, at i=45 (45*1.6=72.0)
	if rec.ConvergenceSpeed == 0 {
		t.Error("expected non-zero convergence speed")
	}
	if rec.FinalDiversity <= 0 {
		t.Error("expected positive final diversity")
	}
	// AvgFitness should be computed from PSO state (which was active)
	// It may be 0 since we didn't run TickPSO, but the field should be set
}

func TestRecordAlgoPerformanceConvergenceSpeedNever90(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitSwarmAlgorithm(ss, AlgoPSO)

	sa := ss.SwarmAlgo
	sa.BestFitnessEver = 100.0
	sa.TotalIterations = 10
	// All history values below 90% of 100 = 90
	sa.ConvergenceHistory = []float64{10, 20, 30, 40, 50, 60, 70, 75, 80, 85}

	InitSwarmAlgorithm(ss, AlgoGWO)

	if len(ss.AlgoScoreboard) != 1 {
		t.Fatalf("expected 1 scoreboard entry, got %d", len(ss.AlgoScoreboard))
	}
	rec := ss.AlgoScoreboard[0]
	// Never reached 90%, so speed should equal total history length
	if rec.ConvergenceSpeed != float64(len(sa.ConvergenceHistory)) {
		t.Errorf("expected convergence speed %d, got %.0f", len(sa.ConvergenceHistory), rec.ConvergenceSpeed)
	}
}
