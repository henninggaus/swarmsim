package swarm

import (
	"math"
	"swarmsim/logger"
)

// OscillatorState manages coupled oscillators using the Kuramoto model.
// Each bot has an internal phase oscillator with a natural frequency.
// Nearby bots couple their phases, leading to synchronization — like
// fireflies flashing in unison. Synchronized groups coordinate better.
type OscillatorState struct {
	Phases     []float64 // per-bot phase (0 to 2*pi)
	NatFreqs   []float64 // per-bot natural frequency
	CoupleStr  float64   // coupling strength (default 0.3)
	CoupleRange float64  // coupling radius (default 80)

	// Stats
	OrderParam    float64 // Kuramoto order parameter r (0=random, 1=full sync)
	MeanPhase     float64 // mean phase angle
	SyncGroups    int     // number of synchronized clusters
	Generation    int
}

// InitOscillators sets up the Kuramoto oscillator system.
func InitOscillators(ss *SwarmState) {
	n := len(ss.Bots)
	os := &OscillatorState{
		Phases:      make([]float64, n),
		NatFreqs:    make([]float64, n),
		CoupleStr:   0.3,
		CoupleRange: 80,
	}

	for i := 0; i < n; i++ {
		os.Phases[i] = ss.Rng.Float64() * 2 * math.Pi
		os.NatFreqs[i] = 0.05 + ss.Rng.Float64()*0.1 // natural frequency spread
	}

	ss.Oscillator = os
	logger.Info("OSC", "Initialisiert: %d Oszillatoren, Kopplung=%.2f", n, os.CoupleStr)
}

// ClearOscillators disables the oscillator system.
func ClearOscillators(ss *SwarmState) {
	ss.Oscillator = nil
	ss.OscillatorOn = false
}

// TickOscillators runs one step of the Kuramoto model.
func TickOscillators(ss *SwarmState) {
	os := ss.Oscillator
	if os == nil {
		return
	}

	n := len(ss.Bots)
	if len(os.Phases) != n {
		return
	}

	radiusSq := os.CoupleRange * os.CoupleRange
	newPhases := make([]float64, n)
	copy(newPhases, os.Phases)

	for i := range ss.Bots {
		// Natural frequency advancement
		newPhases[i] += os.NatFreqs[i]

		// Kuramoto coupling: sum of sin(phase_j - phase_i) for neighbors
		coupling := 0.0
		neighbors := 0

		for j := range ss.Bots {
			if i == j {
				continue
			}
			dx := ss.Bots[j].X - ss.Bots[i].X
			dy := ss.Bots[j].Y - ss.Bots[i].Y
			if dx*dx+dy*dy < radiusSq {
				coupling += math.Sin(os.Phases[j] - os.Phases[i])
				neighbors++
			}
		}

		if neighbors > 0 {
			newPhases[i] += os.CoupleStr * coupling / float64(neighbors)
		}

		// Wrap to [0, 2*pi]
		for newPhases[i] < 0 {
			newPhases[i] += 2 * math.Pi
		}
		for newPhases[i] >= 2*math.Pi {
			newPhases[i] -= 2 * math.Pi
		}
	}

	copy(os.Phases, newPhases)

	// Apply oscillator effects to bot behavior
	applyOscBehavior(ss, os)
	updateOscStats(os)
}

// applyOscBehavior uses oscillator phase for coordinated behavior.
func applyOscBehavior(ss *SwarmState, os *OscillatorState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		phase := os.Phases[i]

		// Phase-dependent speed: bots pulse between fast and slow
		pulseFactor := 0.8 + 0.4*math.Sin(phase)
		bot.Speed *= pulseFactor

		// LED: firefly effect — brightness follows phase
		brightness := math.Sin(phase)
		if brightness < 0 {
			brightness = 0
		}
		intensity := uint8(brightness * 255)

		// Synchronized bots glow yellow, desync bots glow blue
		syncScore := math.Abs(math.Sin(phase - os.MeanPhase))
		if syncScore < 0.3 {
			bot.LEDColor = [3]uint8{intensity, intensity, uint8(float64(intensity) * 0.3)}
		} else {
			bot.LEDColor = [3]uint8{uint8(float64(intensity) * 0.3), uint8(float64(intensity) * 0.5), intensity}
		}
	}
}

// updateOscStats computes the Kuramoto order parameter and sync groups.
func updateOscStats(os *OscillatorState) {
	n := len(os.Phases)
	if n == 0 {
		return
	}

	// Order parameter: r*e^(i*psi) = (1/N) * sum(e^(i*theta_j))
	sumCos, sumSin := 0.0, 0.0
	for _, phase := range os.Phases {
		sumCos += math.Cos(phase)
		sumSin += math.Sin(phase)
	}

	fn := float64(n)
	avgCos := sumCos / fn
	avgSin := sumSin / fn
	os.OrderParam = math.Sqrt(avgCos*avgCos + avgSin*avgSin)
	os.MeanPhase = math.Atan2(avgSin, avgCos)

	// Count sync groups: bots within pi/4 phase difference
	visited := make([]bool, n)
	groups := 0
	for i := 0; i < n; i++ {
		if visited[i] {
			continue
		}
		groups++
		visited[i] = true
		// BFS: find all bots close in phase
		queue := []int{i}
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			for j := 0; j < n; j++ {
				if visited[j] {
					continue
				}
				diff := math.Abs(os.Phases[curr] - os.Phases[j])
				if diff > math.Pi {
					diff = 2*math.Pi - diff
				}
				if diff < math.Pi/4 {
					visited[j] = true
					queue = append(queue, j)
				}
			}
		}
	}
	os.SyncGroups = groups
}

// OscOrderParam returns the Kuramoto order parameter (0=random, 1=sync).
func OscOrderParam(os *OscillatorState) float64 {
	if os == nil {
		return 0
	}
	return os.OrderParam
}

// OscSyncGroups returns the number of synchronized groups.
func OscSyncGroups(os *OscillatorState) int {
	if os == nil {
		return 0
	}
	return os.SyncGroups
}

// BotPhase returns a bot's oscillator phase.
func BotPhase(os *OscillatorState, botIdx int) float64 {
	if os == nil || botIdx < 0 || botIdx >= len(os.Phases) {
		return 0
	}
	return os.Phases[botIdx]
}
