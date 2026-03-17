package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestNewWeatherState(t *testing.T) {
	ws := NewWeatherState()
	if ws.Current != WeatherClear {
		t.Error("should start clear")
	}
	if ws.Visibility != 1.0 {
		t.Error("visibility should be 1.0")
	}
	if ws.Timer != 3000 {
		t.Errorf("timer should be 3000, got %d", ws.Timer)
	}
}

func TestWeatherName(t *testing.T) {
	names := []string{"Klar", "Regen", "Nebel", "Wind", "Sturm"}
	for i, expected := range names {
		got := WeatherName(WeatherType(i))
		if got != expected {
			t.Errorf("WeatherName(%d) = %q, want %q", i, got, expected)
		}
	}
	if WeatherName(WeatherType(99)) != "Unbekannt" {
		t.Error("unknown weather should be Unbekannt")
	}
}

func TestTickWeatherNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	TickWeather(rng, nil) // should not panic
}

func TestTickWeatherCountdown(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := NewWeatherState()
	ws.Timer = 100
	TickWeather(rng, ws)
	if ws.Timer != 99 {
		t.Errorf("timer should decrement to 99, got %d", ws.Timer)
	}
}

func TestTickWeatherChange(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := NewWeatherState()
	ws.Timer = 1
	TickWeather(rng, ws)
	// After timer hits 0, weather should change and timer reset
	if ws.Timer < 500 {
		t.Errorf("timer should be >= 500 after reset, got %d", ws.Timer)
	}
	if len(ws.History) != 1 {
		t.Errorf("history should have 1 entry, got %d", len(ws.History))
	}
}

func TestTickWeatherHistoryPruning(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := NewWeatherState()
	ws.MaxHistory = 3
	for i := 0; i < 5; i++ {
		ws.Timer = 1
		TickWeather(rng, ws)
	}
	if len(ws.History) > 3 {
		t.Errorf("history should be <= 3, got %d", len(ws.History))
	}
}

func TestApplyWeatherEffectsClear(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := &WeatherState{Current: WeatherClear}
	applyWeatherEffects(rng, ws)
	if ws.Visibility != 1.0 {
		t.Errorf("clear visibility should be 1.0, got %f", ws.Visibility)
	}
	if ws.WindStrength != 0 {
		t.Error("clear should have no wind")
	}
}

func TestApplyWeatherEffectsRain(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := &WeatherState{Current: WeatherRain}
	applyWeatherEffects(rng, ws)
	if ws.Visibility != 0.7 {
		t.Errorf("rain visibility should be 0.7, got %f", ws.Visibility)
	}
	if ws.WindStrength != 0.1 {
		t.Errorf("rain wind should be 0.1, got %f", ws.WindStrength)
	}
}

func TestApplyWeatherEffectsFog(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := &WeatherState{Current: WeatherFog}
	applyWeatherEffects(rng, ws)
	if ws.Visibility != 0.3 {
		t.Errorf("fog visibility should be 0.3, got %f", ws.Visibility)
	}
	if ws.WindStrength != 0 {
		t.Error("fog should have no wind")
	}
}

func TestApplyWeatherEffectsWind(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := &WeatherState{Current: WeatherWind}
	applyWeatherEffects(rng, ws)
	if ws.Visibility != 0.9 {
		t.Errorf("wind visibility should be 0.9, got %f", ws.Visibility)
	}
	if ws.WindStrength < 0.4 || ws.WindStrength > 0.7 {
		t.Errorf("wind strength should be 0.4-0.7, got %f", ws.WindStrength)
	}
}

func TestApplyWeatherEffectsStorm(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := &WeatherState{Current: WeatherStorm}
	applyWeatherEffects(rng, ws)
	if ws.Visibility != 0.5 {
		t.Errorf("storm visibility should be 0.5, got %f", ws.Visibility)
	}
	if ws.WindStrength < 0.6 || ws.WindStrength > 0.9 {
		t.Errorf("storm wind should be 0.6-0.9, got %f", ws.WindStrength)
	}
}

func TestApplyWindToBot(t *testing.T) {
	ws := &WeatherState{
		WindAngle:    0, // blows east
		WindStrength: 1.0,
	}
	bot := &SwarmBot{X: 100, Y: 100}
	ApplyWindToBot(ws, bot)
	if bot.X <= 100 {
		t.Error("bot should move east with wind angle 0")
	}
	if math.Abs(bot.Y-100) > 0.01 {
		t.Error("bot should not move vertically with wind angle 0")
	}
}

func TestApplyWindToBotNil(t *testing.T) {
	bot := &SwarmBot{X: 100, Y: 100}
	ApplyWindToBot(nil, bot)
	if bot.X != 100 {
		t.Error("nil weather should not move bot")
	}
}

func TestApplyWindToBotNoWind(t *testing.T) {
	ws := &WeatherState{WindStrength: 0.001}
	bot := &SwarmBot{X: 100, Y: 100}
	ApplyWindToBot(ws, bot)
	if bot.X != 100 {
		t.Error("very low wind should not move bot")
	}
}

func TestVisibilityRange(t *testing.T) {
	ws := &WeatherState{Visibility: 0.5}
	r := VisibilityRange(ws, 100)
	if r != 50 {
		t.Errorf("expected 50, got %f", r)
	}
}

func TestVisibilityRangeNil(t *testing.T) {
	r := VisibilityRange(nil, 100)
	if r != 100 {
		t.Errorf("nil weather should return base range, got %f", r)
	}
}

func TestWindDrift(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ws := NewWeatherState()
	ws.WindStrength = 0.5
	ws.WindAngle = 1.0
	ws.Timer = 100
	original := ws.WindAngle
	TickWeather(rng, ws)
	if ws.WindAngle == original {
		t.Error("wind angle should drift when wind is active")
	}
	if math.Abs(ws.WindAngle-original) > 0.02 {
		t.Error("wind drift should be small")
	}
}

func TestWeatherTypeCount(t *testing.T) {
	if WeatherTypeCount != 5 {
		t.Errorf("expected 5 weather types, got %d", WeatherTypeCount)
	}
}
