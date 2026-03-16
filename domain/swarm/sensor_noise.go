package swarm

import "math/rand"

// SensorNoiseConfig controls noise and failure parameters.
type SensorNoiseConfig struct {
	NoiseLevel   float64 // 0.0-1.0: how much noise to add (0=none, 0.2=20% noise)
	FailureRate  float64 // 0.0-1.0: probability of sensor failure per tick (0=never, 0.05=5%)
}

// ApplySensorNoise adds Gaussian noise and random failures to bot sensor readings.
// Should be called after buildSwarmEnvironment.
func ApplySensorNoise(ss *SwarmState) {
	if !ss.SensorNoiseOn {
		return
	}
	cfg := &ss.SensorNoiseCfg
	rng := ss.Rng

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Distance sensors: add proportional noise
		if cfg.NoiseLevel > 0 {
			bot.NearestDist = addNoise(bot.NearestDist, cfg.NoiseLevel, rng)
			bot.NearestPickupDist = addNoise(bot.NearestPickupDist, cfg.NoiseLevel, rng)
			bot.NearestDropoffDist = addNoise(bot.NearestDropoffDist, cfg.NoiseLevel, rng)
			bot.ObstacleDist = addNoise(bot.ObstacleDist, cfg.NoiseLevel, rng)

			// Count sensors: add small integer noise
			bot.NeighborCount = addIntNoise(bot.NeighborCount, cfg.NoiseLevel, rng)
			bot.LightValue = addIntNoise(bot.LightValue, cfg.NoiseLevel, rng)
			bot.BotAhead = addIntNoise(bot.BotAhead, cfg.NoiseLevel, rng)
			bot.BotBehind = addIntNoise(bot.BotBehind, cfg.NoiseLevel, rng)
			bot.BotLeft = addIntNoise(bot.BotLeft, cfg.NoiseLevel, rng)
			bot.BotRight = addIntNoise(bot.BotRight, cfg.NoiseLevel, rng)
		}

		// Random sensor failures: sensor reads 0
		if cfg.FailureRate > 0 {
			if rng.Float64() < cfg.FailureRate {
				// Pick a random sensor to fail
				switch rng.Intn(6) {
				case 0:
					bot.NearestDist = 999
				case 1:
					bot.NeighborCount = 0
				case 2:
					bot.ObstacleAhead = false
					bot.ObstacleDist = 999
				case 3:
					bot.LightValue = 0
				case 4:
					bot.ReceivedMsg = 0
				case 5:
					bot.NearestPickupDist = 999
					bot.NearestDropoffDist = 999
				}
			}
		}
	}
}

// addNoise adds Gaussian noise proportional to the value.
func addNoise(val, level float64, rng *rand.Rand) float64 {
	noise := rng.NormFloat64() * level * val * 0.3
	result := val + noise
	if result < 0 {
		result = 0
	}
	return result
}

// addIntNoise adds small integer noise to a count sensor.
func addIntNoise(val int, level float64, rng *rand.Rand) int {
	if rng.Float64() > level*2 {
		return val // most of the time, no noise for int sensors
	}
	delta := rng.Intn(3) - 1 // -1, 0, or +1
	result := val + delta
	if result < 0 {
		result = 0
	}
	return result
}
