package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func makeSensitivityBots(rng *rand.Rand, n int) []SwarmBot {
	bots := make([]SwarmBot, n)
	for i := range bots {
		bots[i].Stats.TotalDeliveries = rng.Intn(10)
		bots[i].Stats.TotalPickups = rng.Intn(10)
		bots[i].Stats.TotalDistance = rng.Float64() * 1000
		bots[i].Stats.TicksAlive = 100
		for p := 0; p < 26; p++ {
			bots[i].ParamValues[p] = 50 + rng.Float64()*20
		}
	}
	return bots
}

func TestSensitivityEmptyParams(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bots := makeSensitivityBots(rng, 10)
	report := RunSensitivityAnalysis(rng, bots, nil, DefaultSensitivityConfig())
	if len(report.Results) != 0 {
		t.Error("empty params should give empty results")
	}
}

func TestSensitivityEmptyBots(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := []SensitivityParam{{Name: "p0", Index: 0, BaseVal: 50, MinVal: 0, MaxVal: 100}}
	report := RunSensitivityAnalysis(rng, nil, params, DefaultSensitivityConfig())
	if len(report.Results) != 0 {
		t.Error("empty bots should give empty results")
	}
}

func TestSensitivitySingleParam(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bots := makeSensitivityBots(rng, 10)
	params := []SensitivityParam{
		{Name: "speed", Index: 0, BaseVal: 50, MinVal: 0, MaxVal: 100},
	}
	report := RunSensitivityAnalysis(rng, bots, params, DefaultSensitivityConfig())
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].ParamName != "speed" {
		t.Errorf("expected param name 'speed', got '%s'", report.Results[0].ParamName)
	}
	if report.MostSensitive != "speed" {
		t.Error("single param should be most sensitive")
	}
}

func TestSensitivityMultipleParams(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bots := makeSensitivityBots(rng, 20)
	params := []SensitivityParam{
		{Name: "p0", Index: 0, BaseVal: 50, MinVal: 0, MaxVal: 100},
		{Name: "p1", Index: 1, BaseVal: 50, MinVal: 0, MaxVal: 100},
		{Name: "p2", Index: 2, BaseVal: 50, MinVal: 0, MaxVal: 100},
	}
	report := RunSensitivityAnalysis(rng, bots, params, DefaultSensitivityConfig())
	if len(report.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(report.Results))
	}
	// Results should be sorted by impact (descending)
	for i := 1; i < len(report.Results); i++ {
		if report.Results[i].Impact > report.Results[i-1].Impact {
			t.Error("results should be sorted by impact descending")
		}
	}
}

func TestSensitivityNormalizedImpact(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bots := makeSensitivityBots(rng, 15)
	params := []SensitivityParam{
		{Name: "p0", Index: 0, BaseVal: 50, MinVal: 0, MaxVal: 100},
		{Name: "p1", Index: 1, BaseVal: 50, MinVal: 0, MaxVal: 100},
	}
	report := RunSensitivityAnalysis(rng, bots, params, DefaultSensitivityConfig())
	norm := report.NormalizedImpact()
	if len(norm) != 2 {
		t.Fatalf("expected 2 normalized values, got %d", len(norm))
	}
	sum := norm[0] + norm[1]
	if report.TotalImpact > 0 && math.Abs(sum-1.0) > 0.01 {
		t.Errorf("normalized impacts should sum to ~1.0, got %f", sum)
	}
}

func TestSensitivityCustomEvalFunc(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bots := makeSensitivityBots(rng, 10)
	// Custom eval: fitness = param[0] * 10
	customEval := func(bot *SwarmBot) float64 {
		return bot.ParamValues[0] * 10
	}
	params := []SensitivityParam{
		{Name: "key_param", Index: 0, BaseVal: 50, MinVal: 0, MaxVal: 100},
	}
	cfg := DefaultSensitivityConfig()
	cfg.EvalFunc = customEval
	report := RunSensitivityAnalysis(rng, bots, params, cfg)
	if len(report.Results) != 1 {
		t.Fatal("expected 1 result")
	}
	// With custom eval, varying param 0 should have significant impact
	if report.Results[0].Impact < 0.01 {
		t.Error("expected non-zero impact when eval directly depends on param")
	}
}

func TestSensitivityParamsRestored(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bots := makeSensitivityBots(rng, 5)
	origVals := make([]float64, len(bots))
	for i := range bots {
		origVals[i] = bots[i].ParamValues[0]
	}
	params := []SensitivityParam{
		{Name: "p0", Index: 0, BaseVal: 50, MinVal: 0, MaxVal: 100},
	}
	RunSensitivityAnalysis(rng, bots, params, DefaultSensitivityConfig())
	for i := range bots {
		if bots[i].ParamValues[0] != origVals[i] {
			t.Errorf("bot %d param not restored: %f != %f", i, bots[i].ParamValues[0], origVals[i])
		}
	}
}

func TestSensitivityDirection(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bots := make([]SwarmBot, 5)
	for i := range bots {
		bots[i].ParamValues[0] = 50
	}
	// Custom eval where higher param = better fitness
	customEval := func(bot *SwarmBot) float64 {
		return bot.ParamValues[0]
	}
	params := []SensitivityParam{
		{Name: "positive", Index: 0, BaseVal: 50, MinVal: 0, MaxVal: 100},
	}
	cfg := SensitivityConfig{Steps: 5, DeltaPct: 0.3, EvalFunc: customEval}
	report := RunSensitivityAnalysis(rng, bots, params, cfg)
	if report.Results[0].Direction != 1 {
		t.Errorf("expected direction +1 (higher=better), got %d", report.Results[0].Direction)
	}
}

func TestSensitivityMinSteps(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	bots := makeSensitivityBots(rng, 5)
	params := []SensitivityParam{
		{Name: "p0", Index: 0, BaseVal: 50, MinVal: 0, MaxVal: 100},
	}
	cfg := DefaultSensitivityConfig()
	cfg.Steps = 1 // below minimum, should be forced to 2
	report := RunSensitivityAnalysis(rng, bots, params, cfg)
	if len(report.Results) != 1 {
		t.Error("should still produce 1 result with min steps")
	}
}
