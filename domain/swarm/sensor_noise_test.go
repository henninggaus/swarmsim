package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func makeNoiseSS(rng *rand.Rand) *SwarmState {
	ss := NewSwarmState(rng, 10)
	for i := range ss.Bots {
		ss.Bots[i].NearestDist = 100
		ss.Bots[i].NeighborCount = 5
		ss.Bots[i].ObstacleDist = 200
		ss.Bots[i].LightValue = 50
		ss.Bots[i].NearestPickupDist = 150
		ss.Bots[i].NearestDropoffDist = 250
	}
	return ss
}

func TestApplySensorNoiseOff(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := makeNoiseSS(rng)
	ss.SensorNoiseOn = false
	ApplySensorNoise(ss)
	// Nothing should change
	if ss.Bots[0].NearestDist != 100 {
		t.Error("noise off should not modify sensors")
	}
}

func TestApplySensorNoiseAddsNoise(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := makeNoiseSS(rng)
	ss.SensorNoiseOn = true
	ss.SensorNoiseCfg = SensorNoiseConfig{NoiseLevel: 0.5, FailureRate: 0}
	ApplySensorNoise(ss)

	changed := 0
	for i := range ss.Bots {
		if math.Abs(ss.Bots[i].NearestDist-100) > 0.01 {
			changed++
		}
	}
	if changed == 0 {
		t.Error("noise should modify at least some sensors")
	}
}

func TestSensorFailureState(t *testing.T) {
	fs := SensorFailureState{}
	fs.FailedSensors[0] = true
	fs.FailedSensors[3] = true
	fs.FailedTicks[0] = 5
	fs.FailedTicks[3] = 2

	if CountFailedSensors(&fs) != 2 {
		t.Errorf("expected 2 failed sensors, got %d", CountFailedSensors(&fs))
	}
	if MaxFailedTicks(&fs) != 5 {
		t.Errorf("expected max 5 ticks, got %d", MaxFailedTicks(&fs))
	}
}

func TestCountFailedSensorsNone(t *testing.T) {
	fs := SensorFailureState{}
	if CountFailedSensors(&fs) != 0 {
		t.Error("fresh state should have 0 failed")
	}
}

func TestGetSensorFailureStateGrows(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := makeNoiseSS(rng)
	// SensorFailures starts empty
	fs := getSensorFailureState(ss, 5)
	if fs == nil {
		t.Fatal("should not be nil")
	}
	if len(ss.SensorFailures) < 10 {
		t.Error("should grow to fit all bots")
	}
}

func TestSensorRecovery(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	ss := makeNoiseSS(rng)
	ss.SensorNoiseOn = true
	ss.SensorNoiseCfg = SensorNoiseConfig{
		NoiseLevel:   0,
		FailureRate:  1.0, // 100% failure
		RecoveryRate: 1.0, // 100% recovery
	}

	// First tick: all sensors fail
	ApplySensorNoise(ss)
	// Second tick: all should recover
	// Reset sensor values first
	for i := range ss.Bots {
		ss.Bots[i].NearestDist = 100
		ss.Bots[i].NeighborCount = 5
	}
	ApplySensorNoise(ss)

	// With 100% recovery, failed sensors from tick 1 should recover in tick 2
	// Some new failures will happen but also immediately recover
	recovered := 0
	for i := range ss.Bots {
		fs := &ss.SensorFailures[i]
		for s := 0; s < 6; s++ {
			if !fs.FailedSensors[s] {
				recovered++
			}
		}
	}
	// With 100% recovery, most should recover
	if recovered == 0 {
		t.Error("100% recovery rate should recover some sensors")
	}
}

func TestNewNoisePatternLearner(t *testing.T) {
	npl := NewNoisePatternLearner()
	if npl == nil {
		t.Fatal("should not be nil")
	}
	if npl.LearningRate != 0.01 {
		t.Errorf("expected learning rate 0.01, got %f", npl.LearningRate)
	}
	if npl.SampleCount != 0 {
		t.Error("should start with 0 samples")
	}
}

func TestUpdateNoiseLearner(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := makeNoiseSS(rng)
	npl := NewNoisePatternLearner()

	for i := 0; i < 20; i++ {
		UpdateNoiseLearner(npl, ss)
	}

	if npl.SampleCount != 20 {
		t.Errorf("expected 20 samples, got %d", npl.SampleCount)
	}
}

func TestNoiseVarianceZeroSamples(t *testing.T) {
	npl := &NoisePatternLearner{}
	v := npl.NoiseVariance(10)
	for s := 0; s < 6; s++ {
		if v[s] != 0 {
			t.Error("zero samples should give zero variance")
		}
	}
}

func TestNoiseVarianceWithData(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := makeNoiseSS(rng)
	// Add variation
	for i := range ss.Bots {
		ss.Bots[i].NearestDist = float64(50 + i*20) // 50, 70, 90, ...
	}

	npl := NewNoisePatternLearner()
	UpdateNoiseLearner(npl, ss)

	v := npl.NoiseVariance(len(ss.Bots))
	// Sensor 0 (NearestDist) should have non-zero variance due to spread
	if v[0] <= 0 {
		t.Error("NearestDist should have positive variance with spread data")
	}
}

func TestAddNoiseNonNegative(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		result := addNoise(10, 0.5, rng)
		if result < 0 {
			t.Errorf("addNoise should not return negative, got %f", result)
		}
	}
}

func TestAddIntNoiseNonNegative(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		result := addIntNoise(0, 1.0, rng)
		if result < 0 {
			t.Errorf("addIntNoise should not return negative, got %d", result)
		}
	}
}

func TestUpdateNoiseLearnerNilSafe(t *testing.T) {
	UpdateNoiseLearner(nil, &SwarmState{}) // should not panic
}

func TestDefaultRecoveryRate(t *testing.T) {
	cfg := SensorNoiseConfig{FailureRate: 0.5}
	if cfg.RecoveryRate != 0 {
		t.Error("default RecoveryRate should be 0 (handled in code as 0.3)")
	}
}
