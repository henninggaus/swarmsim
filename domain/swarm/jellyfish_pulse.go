package swarm

import "math"

// Jellyfish Pulse: Coordinated rhythmic expansion and contraction of the
// entire swarm, like a jellyfish bell. All bots pulse between moving outward
// (expansion) and inward (contraction) in sync, creating hypnotic breathing.
// Uses coupled oscillator model — bots sync their phase via neighbors.

const (
	jellyPeriod      = 90    // ticks per pulse cycle
	jellyCoupleStr   = 0.06  // phase coupling strength
	jellyExpandForce = 0.18  // expansion steering rate
	jellyContractForce = 0.22 // contraction steering rate
	jellyMinRadius   = 40.0  // minimum contraction radius
	jellyMaxRadius   = 200.0 // maximum expansion radius
)

// JellyfishState holds jellyfish pulse state.
type JellyfishState struct {
	Phase    []float64 // per-bot oscillator phase (0 to 2π)
	CenterX  float64   // swarm center X
	CenterY  float64   // swarm center Y
}

// InitJellyfish allocates jellyfish state.
func InitJellyfish(ss *SwarmState) {
	n := len(ss.Bots)
	st := &JellyfishState{
		Phase:   make([]float64, n),
		CenterX: ss.ArenaW / 2,
		CenterY: ss.ArenaH / 2,
	}
	// Start all in sync
	for i := range st.Phase {
		st.Phase[i] = 0
	}
	ss.Jellyfish = st
	ss.JellyfishOn = true
}

// ClearJellyfish frees jellyfish state.
func ClearJellyfish(ss *SwarmState) {
	ss.Jellyfish = nil
	ss.JellyfishOn = false
}

// TickJellyfish advances phases and couples neighboring oscillators.
func TickJellyfish(ss *SwarmState) {
	if ss.Jellyfish == nil {
		return
	}
	st := ss.Jellyfish

	// Grow slices
	for len(st.Phase) < len(ss.Bots) {
		st.Phase = append(st.Phase, 0)
	}

	// Compute swarm center
	var sumX, sumY float64
	for i := range ss.Bots {
		sumX += ss.Bots[i].X
		sumY += ss.Bots[i].Y
	}
	n := float64(len(ss.Bots))
	if n > 0 {
		st.CenterX = sumX / n
		st.CenterY = sumY / n
	}

	// Phase advance + coupling (Kuramoto-like)
	omega := 2 * math.Pi / float64(jellyPeriod) // natural frequency
	for i := range ss.Bots {
		st.Phase[i] += omega

		// Couple with neighbors
		if ss.Hash != nil {
			nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, 80)
			for _, j := range nearIDs {
				if j == i || j < 0 || j >= len(ss.Bots) {
					continue
				}
				phaseDiff := st.Phase[j] - st.Phase[i]
				st.Phase[i] += jellyCoupleStr * math.Sin(phaseDiff)
			}
		}

		// Wrap phase
		if st.Phase[i] > 2*math.Pi {
			st.Phase[i] -= 2 * math.Pi
		}
		if st.Phase[i] < 0 {
			st.Phase[i] += 2 * math.Pi
		}

		// Update sensor cache
		// PulsePhase: 0-100 where 0-50=expanding, 50-100=contracting
		ss.Bots[i].JellyPhase = int(st.Phase[i] / (2 * math.Pi) * 100)

		// Is this the expansion half?
		if math.Sin(st.Phase[i]) > 0 {
			ss.Bots[i].JellyExpanding = 1
		} else {
			ss.Bots[i].JellyExpanding = 0
		}

		// Distance to center
		dx := ss.Bots[i].X - st.CenterX
		dy := ss.Bots[i].Y - st.CenterY
		ss.Bots[i].JellyRadius = int(math.Min(9999, math.Sqrt(dx*dx+dy*dy)))
	}
}

// ApplyJellyfishPulse moves bot outward/inward based on pulse phase.
func ApplyJellyfishPulse(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Jellyfish == nil || idx >= len(ss.Jellyfish.Phase) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Jellyfish

	dx := bot.X - st.CenterX
	dy := bot.Y - st.CenterY
	dist := math.Sqrt(dx*dx + dy*dy)

	// Angle from center to bot
	outAngle := math.Atan2(dy, dx)
	inAngle := outAngle + math.Pi

	pulseVal := math.Sin(st.Phase[idx])

	var targetAngle float64
	var force float64

	if pulseVal > 0 {
		// Expanding phase: move outward
		if dist < jellyMaxRadius {
			targetAngle = outAngle
			force = jellyExpandForce * pulseVal
		} else {
			// At max radius: slow down
			targetAngle = outAngle
			force = 0.02
		}
	} else {
		// Contracting phase: move inward
		if dist > jellyMinRadius {
			targetAngle = inAngle
			force = jellyContractForce * (-pulseVal)
		} else {
			// At min radius: slow down
			targetAngle = inAngle
			force = 0.02
		}
	}

	// Apply steering
	diff := targetAngle - bot.Angle
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	if diff > force {
		diff = force
	} else if diff < -force {
		diff = -force
	}
	bot.Angle += diff
	bot.Speed = SwarmBotSpeed * (0.5 + 0.5*math.Abs(pulseVal))

	// LED: pulsing glow — cyan during expansion, purple during contraction
	t := (math.Sin(st.Phase[idx]) + 1) / 2 // 0-1
	r := uint8(100 + t*100)
	g := uint8(50 + (1-t)*200)
	b := uint8(200)
	bot.LEDColor = [3]uint8{r, g, b}
}
