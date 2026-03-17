package swarm

import (
	"math"
	"math/rand"
)

// WeatherType identifies the current weather condition.
type WeatherType int

const (
	WeatherClear WeatherType = iota
	WeatherRain              // reduces sensor range
	WeatherFog               // heavily reduces sensor range
	WeatherWind              // applies directional force to bots
	WeatherStorm             // wind + rain combined
	WeatherTypeCount
)

// WeatherState tracks the current weather and its effects.
type WeatherState struct {
	Current       WeatherType
	WindAngle     float64 // direction wind blows (radians)
	WindStrength  float64 // 0.0-1.0 force applied to bot movement
	Visibility    float64 // 0.0-1.0 multiplier on sensor ranges (1=clear, 0.3=fog)
	Duration      int     // ticks remaining for current weather
	ChangeRate    int     // ticks between weather changes (default 3000)
	Timer         int     // ticks until next change
	History       []WeatherRecord
	MaxHistory    int
}

// WeatherRecord stores a past weather event for visualization.
type WeatherRecord struct {
	Type     WeatherType
	Duration int
	Tick     int
}

// NewWeatherState creates a weather system with defaults.
func NewWeatherState() *WeatherState {
	return &WeatherState{
		Current:    WeatherClear,
		Visibility: 1.0,
		ChangeRate: 3000,
		Timer:      3000,
		MaxHistory: 50,
	}
}

// WeatherName returns the display name for a weather type.
func WeatherName(w WeatherType) string {
	switch w {
	case WeatherClear:
		return "Klar"
	case WeatherRain:
		return "Regen"
	case WeatherFog:
		return "Nebel"
	case WeatherWind:
		return "Wind"
	case WeatherStorm:
		return "Sturm"
	default:
		return "Unbekannt"
	}
}

// TickWeather advances the weather system by one tick.
func TickWeather(rng *rand.Rand, ws *WeatherState) {
	if ws == nil {
		return
	}

	ws.Timer--
	if ws.Timer > 0 {
		// Wind angle drifts slowly
		if ws.WindStrength > 0 {
			ws.WindAngle += (rng.Float64() - 0.5) * 0.02
		}
		return
	}

	// Weather change
	ws.History = append(ws.History, WeatherRecord{
		Type:     ws.Current,
		Duration: ws.ChangeRate - ws.Timer,
		Tick:     0, // caller should set
	})
	if len(ws.History) > ws.MaxHistory {
		ws.History = ws.History[1:]
	}

	// Pick new weather (weighted: clear is most common)
	roll := rng.Float64()
	switch {
	case roll < 0.4:
		ws.Current = WeatherClear
	case roll < 0.6:
		ws.Current = WeatherRain
	case roll < 0.75:
		ws.Current = WeatherFog
	case roll < 0.9:
		ws.Current = WeatherWind
	default:
		ws.Current = WeatherStorm
	}

	applyWeatherEffects(rng, ws)
	ws.Timer = ws.ChangeRate + rng.Intn(1000) - 500 // ±500 ticks variation
	if ws.Timer < 500 {
		ws.Timer = 500
	}
}

func applyWeatherEffects(rng *rand.Rand, ws *WeatherState) {
	switch ws.Current {
	case WeatherClear:
		ws.Visibility = 1.0
		ws.WindStrength = 0
	case WeatherRain:
		ws.Visibility = 0.7
		ws.WindStrength = 0.1
		ws.WindAngle = rng.Float64() * 2 * math.Pi
	case WeatherFog:
		ws.Visibility = 0.3
		ws.WindStrength = 0
	case WeatherWind:
		ws.Visibility = 0.9
		ws.WindStrength = 0.4 + rng.Float64()*0.3
		ws.WindAngle = rng.Float64() * 2 * math.Pi
	case WeatherStorm:
		ws.Visibility = 0.5
		ws.WindStrength = 0.6 + rng.Float64()*0.3
		ws.WindAngle = rng.Float64() * 2 * math.Pi
	}
}

// ApplyWindToBot applies wind force to a bot's position.
func ApplyWindToBot(ws *WeatherState, bot *SwarmBot) {
	if ws == nil || ws.WindStrength < 0.01 {
		return
	}
	bot.X += math.Cos(ws.WindAngle) * ws.WindStrength * 0.5
	bot.Y += math.Sin(ws.WindAngle) * ws.WindStrength * 0.5
}

// VisibilityRange returns the effective sensor range after weather effects.
func VisibilityRange(ws *WeatherState, baseRange float64) float64 {
	if ws == nil {
		return baseRange
	}
	return baseRange * ws.Visibility
}
