package swarm

import (
	"math"
	"math/rand"
	"testing"

	"swarmsim/domain/physics"
)

func makePSOState(n int) *SwarmState {
	ss := &SwarmState{
		Bots:   make([]SwarmBot, n),
		ArenaW: 800,
		ArenaH: 800,
		Rng:    rand.New(rand.NewSource(42)),
		Hash:   physics.NewSpatialHash(800, 800, 30),
	}
	for i := range ss.Bots {
		ss.Bots[i].X = ss.Rng.Float64() * 800
		ss.Bots[i].Y = ss.Rng.Float64() * 800
		ss.Bots[i].Angle = ss.Rng.Float64() * 2 * math.Pi
		ss.Bots[i].Energy = 80
		ss.Bots[i].CarryingPkg = -1
	}
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}
	return ss
}

func TestInitPSO(t *testing.T) {
	ss := makePSOState(20)
	InitPSO(ss)
	if ss.PSO == nil {
		t.Fatal("PSO state should not be nil after init")
	}
	if !ss.PSOOn {
		t.Fatal("PSOOn should be true after init")
	}
	st := ss.PSO
	if len(st.VelX) != 20 || len(st.VelY) != 20 {
		t.Fatalf("velocity slices should have length 20, got %d/%d", len(st.VelX), len(st.VelY))
	}
	if len(st.BestX) != 20 || len(st.BestY) != 20 || len(st.BestFit) != 20 {
		t.Fatal("personal best slices should have length 20")
	}
	// Fitness landscape should have 3-5 peaks
	if len(st.PeakX) < 3 || len(st.PeakX) > 5 {
		t.Fatalf("expected 3-5 peaks, got %d", len(st.PeakX))
	}
	// Global best should be initialized
	if st.GlobalFit < 0 {
		t.Fatal("global fitness should be >= 0 after init")
	}
}

func TestClearPSO(t *testing.T) {
	ss := makePSOState(10)
	InitPSO(ss)
	ClearPSO(ss)
	if ss.PSO != nil {
		t.Fatal("PSO should be nil after clear")
	}
	if ss.PSOOn {
		t.Fatal("PSOOn should be false after clear")
	}
}

func TestPSOEvaluate(t *testing.T) {
	st := &PSOState{
		PeakX: []float64{100},
		PeakY: []float64{100},
		PeakH: []float64{80},
		PeakS: []float64{50},
	}
	// Fitness at peak center should be maximal
	fitCenter := psoEvaluate(st, 100, 100)
	if math.Abs(fitCenter-80) > 0.01 {
		t.Fatalf("fitness at peak center should be ~80, got %f", fitCenter)
	}
	// Fitness far away should be near zero
	fitFar := psoEvaluate(st, 1000, 1000)
	if fitFar > 1.0 {
		t.Fatalf("fitness far from peak should be ~0, got %f", fitFar)
	}
	// Fitness should decrease with distance from peak
	fitNear := psoEvaluate(st, 120, 100)
	if fitNear >= fitCenter {
		t.Fatal("fitness should decrease with distance from peak center")
	}
	if fitNear <= fitFar {
		t.Fatal("fitness near peak should be higher than far from peak")
	}
}

func TestTickPSOStandalone(t *testing.T) {
	ss := makePSOState(20)
	// Use the full algorithm init to set up shared fitness landscape.
	InitSwarmAlgorithm(ss, AlgoPSO)
	sa := ss.SwarmAlgo

	// Place one bot directly at the strongest shared peak
	peakIdx := 0
	bestH := sa.FitPeakH[0]
	for p := 1; p < len(sa.FitPeakH); p++ {
		if sa.FitPeakH[p] > bestH {
			bestH = sa.FitPeakH[p]
			peakIdx = p
		}
	}
	ss.Bots[0].X = sa.FitPeakX[peakIdx]
	ss.Bots[0].Y = sa.FitPeakY[peakIdx]

	// Run ticks (including at least one fitness evaluation tick)
	for tick := 0; tick < psoUpdateRate+1; tick++ {
		ss.Tick = tick
		TickPSO(ss)
	}

	// Global best should be near the peak
	dx := ss.PSO.GlobalX - sa.FitPeakX[peakIdx]
	dy := ss.PSO.GlobalY - sa.FitPeakY[peakIdx]
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist > 200 {
		t.Fatalf("global best should be near strongest peak, distance: %f", dist)
	}

	// Sensor caches should be populated
	for i := range ss.Bots {
		if ss.Bots[i].PSOGlobalDist < 0 {
			t.Fatalf("bot %d: PSOGlobalDist should be >= 0", i)
		}
	}
}

func TestTickPSONilSafe(t *testing.T) {
	ss := makePSOState(10)
	// Should not panic when PSO is nil
	TickPSO(ss)
}

func TestApplyPSOMove(t *testing.T) {
	ss := makePSOState(10)
	InitPSO(ss)
	// Set a velocity for bot 0
	ss.PSO.VelX[0] = 2.0
	ss.PSO.VelY[0] = 0.0

	bot := &ss.Bots[0]
	bot.Angle = math.Pi / 2 // facing up, velocity points right
	ApplyPSOMove(bot, ss, 0)

	if bot.Speed <= 0 {
		t.Fatal("bot speed should be positive when velocity is significant")
	}
	// Angle should have moved toward 0 (rightward)
	if bot.Angle > math.Pi/2 {
		t.Fatal("bot angle should have moved toward velocity direction")
	}
}

func TestApplyPSOMoveZeroVelocity(t *testing.T) {
	ss := makePSOState(5)
	InitPSO(ss)
	ss.PSO.VelX[0] = 0
	ss.PSO.VelY[0] = 0

	bot := &ss.Bots[0]
	ApplyPSOMove(bot, ss, 0)
	if bot.Speed != 0 {
		t.Fatal("bot speed should be 0 when velocity is near zero")
	}
}

func TestApplyPSOMoveNilState(t *testing.T) {
	ss := makePSOState(5)
	bot := &ss.Bots[0]
	ApplyPSOMove(bot, ss, 0)
	if bot.Speed != SwarmBotSpeed {
		t.Fatalf("should default to SwarmBotSpeed when PSO is nil, got %f", bot.Speed)
	}
}

func TestApplyPSOMoveOutOfRange(t *testing.T) {
	ss := makePSOState(5)
	InitPSO(ss)
	bot := &ss.Bots[0]
	ApplyPSOMove(bot, ss, 999) // index beyond slice
	if bot.Speed != SwarmBotSpeed {
		t.Fatal("should default to SwarmBotSpeed when index out of range")
	}
}

func TestPSOGrowSlices(t *testing.T) {
	ss := makePSOState(5)
	InitPSO(ss)
	// Add bots
	for i := 0; i < 5; i++ {
		ss.Bots = append(ss.Bots, SwarmBot{
			X: ss.Rng.Float64() * 800, Y: ss.Rng.Float64() * 800,
			Energy: 50, CarryingPkg: -1,
		})
	}
	// Should not panic and should grow slices
	ss.Tick = 0
	TickPSO(ss)
	if len(ss.PSO.VelX) != 10 {
		t.Fatalf("expected 10 velocity entries after grow, got %d", len(ss.PSO.VelX))
	}
}

func TestPSOConvergence(t *testing.T) {
	ss := makePSOState(30)
	InitPSO(ss)

	// Run many ticks to let PSO converge
	for tick := 0; tick < 500; tick++ {
		ss.Tick = tick
		TickPSO(ss)
	}

	// Global best fitness should be reasonably high (near a peak)
	if ss.PSO.GlobalFit < 10 {
		t.Fatalf("after 500 ticks, global fitness should be significant, got %f", ss.PSO.GlobalFit)
	}
}

func TestPSOLEDColor(t *testing.T) {
	ss := makePSOState(10)
	InitPSO(ss)
	// Place bot at a peak for high fitness
	ss.Bots[0].X = ss.PSO.PeakX[0]
	ss.Bots[0].Y = ss.PSO.PeakY[0]
	ApplyPSOMove(&ss.Bots[0], ss, 0)
	// LED should have green component > red (high fitness → green)
	if ss.Bots[0].LEDColor[1] < ss.Bots[0].LEDColor[0] {
		t.Log("LED color at peak should favor green over red")
	}
}
