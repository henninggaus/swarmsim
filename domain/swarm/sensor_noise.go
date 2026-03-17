package swarm

import (
	"math"
	"math/rand"
)

// SensorNoiseConfig controls noise and failure parameters.
type SensorNoiseConfig struct {
	NoiseLevel   float64 // 0.0-1.0: how much noise to add (0=none, 0.2=20% noise)
	FailureRate  float64 // 0.0-1.0: probability of sensor failure per tick (0=never, 0.05=5%)
	RecoveryRate float64 // 0.0-1.0: probability of recovering from failure per tick (default 0.3)
}

// SensorFailureState tracks per-bot sensor failure and recovery.
type SensorFailureState struct {
	FailedSensors [6]bool // which of the 6 sensor groups are currently failed
	FailedTicks   [6]int  // how many ticks each sensor has been failed
}

// NoisePatternLearner tracks noise patterns over time and adapts.
type NoisePatternLearner struct {
	SampleCount   int
	NoiseSum      [6]float64 // accumulated noise magnitude per sensor group
	NoiseSquared  [6]float64 // for variance calculation
	Corrections   [6]float64 // learned bias corrections
	LearningRate  float64    // how fast corrections adapt (default 0.01)
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

		// Sensor failures with recovery
		if cfg.FailureRate > 0 {
			fs := getSensorFailureState(ss, i)

			recoveryRate := cfg.RecoveryRate
			if recoveryRate <= 0 {
				recoveryRate = 0.3
			}

			for s := 0; s < 6; s++ {
				if fs.FailedSensors[s] {
					// Try recovery
					if rng.Float64() < recoveryRate {
						fs.FailedSensors[s] = false
						fs.FailedTicks[s] = 0
					} else {
						fs.FailedTicks[s]++
					}
				} else {
					// Try failure
					if rng.Float64() < cfg.FailureRate {
						fs.FailedSensors[s] = true
						fs.FailedTicks[s] = 1
					}
				}

				// Apply failure
				if fs.FailedSensors[s] {
					applySensorFailure(bot, s)
				}
			}
		}
	}

	// Update noise pattern learner
	if ss.NoisePatternLearn != nil {
		UpdateNoiseLearner(ss.NoisePatternLearn, ss)
	}
}

// applySensorFailure zeros out a specific sensor group.
func applySensorFailure(bot *SwarmBot, sensorGroup int) {
	switch sensorGroup {
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

// getSensorFailureState returns the failure state for bot i, initializing if needed.
func getSensorFailureState(ss *SwarmState, i int) *SensorFailureState {
	if len(ss.SensorFailures) <= i {
		// Grow slice
		needed := len(ss.Bots)
		newFS := make([]SensorFailureState, needed)
		copy(newFS, ss.SensorFailures)
		ss.SensorFailures = newFS
	}
	return &ss.SensorFailures[i]
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

// NewNoisePatternLearner creates a noise pattern learner with default learning rate.
func NewNoisePatternLearner() *NoisePatternLearner {
	return &NoisePatternLearner{
		LearningRate: 0.01,
	}
}

// UpdateNoiseLearner updates the learner with current bot sensor data.
// It tracks how much sensors deviate from the population mean to learn bias corrections.
func UpdateNoiseLearner(npl *NoisePatternLearner, ss *SwarmState) {
	if npl == nil || len(ss.Bots) == 0 {
		return
	}

	npl.SampleCount++

	// Compute population means for each sensor group
	n := float64(len(ss.Bots))
	var means [6]float64
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		means[0] += bot.NearestDist
		means[1] += float64(bot.NeighborCount)
		means[2] += bot.ObstacleDist
		means[3] += float64(bot.LightValue)
		means[4] += float64(bot.ReceivedMsg)
		means[5] += bot.NearestPickupDist
	}
	for s := 0; s < 6; s++ {
		means[s] /= n
	}

	// Compute variance (deviation from mean)
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		vals := [6]float64{
			bot.NearestDist,
			float64(bot.NeighborCount),
			bot.ObstacleDist,
			float64(bot.LightValue),
			float64(bot.ReceivedMsg),
			bot.NearestPickupDist,
		}
		for s := 0; s < 6; s++ {
			diff := vals[s] - means[s]
			npl.NoiseSum[s] += math.Abs(diff)
			npl.NoiseSquared[s] += diff * diff
		}
	}

	// Update corrections using exponential moving average
	if npl.SampleCount > 10 {
		for s := 0; s < 6; s++ {
			avgNoise := npl.NoiseSum[s] / float64(npl.SampleCount*len(ss.Bots))
			// Correction reduces the noise effect
			npl.Corrections[s] = npl.Corrections[s]*(1-npl.LearningRate) + avgNoise*npl.LearningRate
		}
	}
}

// NoiseVariance returns the estimated variance per sensor group.
func (npl *NoisePatternLearner) NoiseVariance(botCount int) [6]float64 {
	var v [6]float64
	total := float64(npl.SampleCount * botCount)
	if total < 1 {
		return v
	}
	for s := 0; s < 6; s++ {
		v[s] = npl.NoiseSquared[s] / total
	}
	return v
}

// CountFailedSensors returns how many sensors are currently failed for a bot.
func CountFailedSensors(fs *SensorFailureState) int {
	count := 0
	for _, f := range fs.FailedSensors {
		if f {
			count++
		}
	}
	return count
}

// MaxFailedTicks returns the longest continuous failure across all sensors.
func MaxFailedTicks(fs *SensorFailureState) int {
	max := 0
	for _, t := range fs.FailedTicks {
		if t > max {
			max = t
		}
	}
	return max
}
