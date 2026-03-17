package swarm

import (
	"math"
	"swarmsim/logger"
)

// MorphogenesisState manages Turing-pattern formation on the swarm.
// Each bot carries activator and inhibitor concentrations. Through local
// diffusion (activator short-range, inhibitor long-range), spatial patterns
// emerge — stripes, spots, or waves. The pattern determines bot roles:
// bots in "activated" regions become workers, inhibited regions become scouts.
type MorphogenesisState struct {
	Activator []float64 // per-bot activator concentration
	Inhibitor []float64 // per-bot inhibitor concentration

	// Turing parameters
	DiffA     float64 // activator diffusion rate (short range, default 0.02)
	DiffI     float64 // inhibitor diffusion rate (long range, default 0.08)
	FeedRate  float64 // activator production rate (default 0.04)
	KillRate  float64 // inhibitor kill rate (default 0.06)
	CoupleRadius float64 // neighbor coupling radius (default 80)

	// Pattern stats
	AvgActivator  float64
	AvgInhibitor  float64
	PatternContrast float64 // difference between max and min activator
	ActivatedCount int     // bots above threshold
	Generation     int
}

// InitMorphogenesis sets up the Turing-pattern system.
func InitMorphogenesis(ss *SwarmState) {
	n := len(ss.Bots)
	ms := &MorphogenesisState{
		Activator:    make([]float64, n),
		Inhibitor:    make([]float64, n),
		DiffA:        0.02,
		DiffI:        0.08,
		FeedRate:     0.04,
		KillRate:     0.06,
		CoupleRadius: 80,
	}

	// Random initial concentrations with small perturbation
	for i := 0; i < n; i++ {
		ms.Activator[i] = 0.5 + (ss.Rng.Float64()-0.5)*0.1
		ms.Inhibitor[i] = 0.25 + (ss.Rng.Float64()-0.5)*0.1
	}

	ss.Morphogenesis = ms
	logger.Info("MORPHO", "Initialisiert: %d Bots mit Turing-Muster Dynamik", n)
}

// ClearMorphogenesis disables the morphogenesis system.
func ClearMorphogenesis(ss *SwarmState) {
	ss.Morphogenesis = nil
	ss.MorphogenesisOn = false
}

// TickMorphogenesis runs one step of the reaction-diffusion on bots.
func TickMorphogenesis(ss *SwarmState) {
	ms := ss.Morphogenesis
	if ms == nil {
		return
	}

	n := len(ss.Bots)
	if len(ms.Activator) != n {
		return
	}

	radiusSq := ms.CoupleRadius * ms.CoupleRadius

	// Compute diffusion from neighbors
	newA := make([]float64, n)
	newI := make([]float64, n)
	copy(newA, ms.Activator)
	copy(newI, ms.Inhibitor)

	for i := range ss.Bots {
		a := ms.Activator[i]
		inh := ms.Inhibitor[i]

		// Diffusion: average neighbor concentrations
		sumA, sumI := 0.0, 0.0
		neighbors := 0

		for j := range ss.Bots {
			if i == j {
				continue
			}
			dx := ss.Bots[j].X - ss.Bots[i].X
			dy := ss.Bots[j].Y - ss.Bots[i].Y
			distSq := dx*dx + dy*dy
			if distSq < radiusSq {
				weight := 1.0 - math.Sqrt(distSq)/ms.CoupleRadius
				sumA += ms.Activator[j] * weight
				sumI += ms.Inhibitor[j] * weight
				neighbors++
			}
		}

		if neighbors > 0 {
			avgA := sumA / float64(neighbors)
			avgI := sumI / float64(neighbors)

			// Diffusion toward neighbor average
			newA[i] += ms.DiffA * (avgA - a)
			newI[i] += ms.DiffI * (avgI - inh)
		}

		// Reaction: Gray-Scott inspired
		// Activator: feeds on itself, consumed by inhibitor
		// Inhibitor: produced by activator, decays
		reaction := a * a * inh
		newA[i] += ms.FeedRate*(1-a) - reaction
		newI[i] += reaction - ms.KillRate*inh

		// Clamp
		newA[i] = clampF(newA[i], 0, 1)
		newI[i] = clampF(newI[i], 0, 1)
	}

	copy(ms.Activator, newA)
	copy(ms.Inhibitor, newI)

	// Apply pattern to bot behavior
	applyMorphoBehavior(ss, ms)
	updateMorphoStats(ms)
}

// applyMorphoBehavior modifies bots based on their activator level.
func applyMorphoBehavior(ss *SwarmState, ms *MorphogenesisState) {
	threshold := 0.5
	ms.ActivatedCount = 0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		a := ms.Activator[i]

		if a > threshold {
			// Activated: worker behavior — focused, efficient
			ms.ActivatedCount++
			bot.Speed *= 0.9 + a*0.2
			// Warm colors (orange-red)
			r := uint8(150 + a*105)
			g := uint8(80 * (1 - a))
			bot.LEDColor = [3]uint8{r, g, 20}
		} else {
			// Inhibited: scout behavior — explore widely
			bot.Speed *= 1.0 + (0.5-a)*0.3
			bot.Angle += (ss.Rng.Float64() - 0.5) * (0.5 - a) * 0.1
			// Cool colors (blue-cyan)
			b := uint8(150 + (1-a)*105)
			g := uint8(100 * a)
			bot.LEDColor = [3]uint8{20, g, b}
		}
	}
}

// updateMorphoStats computes pattern statistics.
func updateMorphoStats(ms *MorphogenesisState) {
	n := len(ms.Activator)
	if n == 0 {
		return
	}

	sumA, sumI := 0.0, 0.0
	minA, maxA := ms.Activator[0], ms.Activator[0]

	for i := range ms.Activator {
		sumA += ms.Activator[i]
		sumI += ms.Inhibitor[i]
		if ms.Activator[i] < minA {
			minA = ms.Activator[i]
		}
		if ms.Activator[i] > maxA {
			maxA = ms.Activator[i]
		}
	}

	fn := float64(n)
	ms.AvgActivator = sumA / fn
	ms.AvgInhibitor = sumI / fn
	ms.PatternContrast = maxA - minA
}

// MorphoContrast returns how strong the pattern is.
func MorphoContrast(ms *MorphogenesisState) float64 {
	if ms == nil {
		return 0
	}
	return ms.PatternContrast
}

// MorphoActivatedRatio returns fraction of activated bots.
func MorphoActivatedRatio(ms *MorphogenesisState) float64 {
	if ms == nil || len(ms.Activator) == 0 {
		return 0
	}
	return float64(ms.ActivatedCount) / float64(len(ms.Activator))
}

// BotActivator returns a bot's activator concentration.
func BotActivator(ms *MorphogenesisState, botIdx int) float64 {
	if ms == nil || botIdx < 0 || botIdx >= len(ms.Activator) {
		return 0
	}
	return ms.Activator[botIdx]
}
