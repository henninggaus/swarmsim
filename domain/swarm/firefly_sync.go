package swarm

import "math"

// Firefly Synchronization: Emergent LED pulse waves across the swarm.
// Inspired by Southeast Asian fireflies that synchronize their flashing.
// Each bot has an internal oscillator (phase 0–255). When the phase reaches
// 255 the bot "flashes" and nudges nearby neighbors' phases forward (coupling).
// Over time the whole swarm synchronizes into beautiful pulse waves.

const (
	fireflyPeriod     = 120   // oscillator period in ticks
	fireflyCoupleStr  = 0.08  // coupling strength (phase nudge per flash)
	fireflyFlashDur   = 8     // flash duration in ticks
	fireflyRadius     = 100.0 // influence radius for coupling
	fireflyNudgeMax   = 20    // max phase nudge per flash event
)

// FireflyState holds per-bot oscillator state.
type FireflyState struct {
	Phase     []float64 // oscillator phase [0, 1)
	FlashTick []int     // tick when last flash occurred (0 = never)
}

// InitFirefly allocates firefly sync state with random initial phases.
func InitFirefly(ss *SwarmState) {
	n := len(ss.Bots)
	st := &FireflyState{
		Phase:     make([]float64, n),
		FlashTick: make([]int, n),
	}
	for i := 0; i < n; i++ {
		st.Phase[i] = ss.Rng.Float64() // random initial phase
	}
	ss.Firefly = st
	ss.FireflyOn = true
}

// ClearFirefly frees firefly sync state.
func ClearFirefly(ss *SwarmState) {
	ss.Firefly = nil
	ss.FireflyOn = false
}

// TickFirefly advances all oscillators, detects flashes, and applies coupling.
// Updates FlashPhase (0–255) and FlashSync (0/1) sensor cache on each bot.
func TickFirefly(ss *SwarmState) {
	if ss.Firefly == nil {
		return
	}
	st := ss.Firefly

	// Grow slices if bots were added
	for len(st.Phase) < len(ss.Bots) {
		st.Phase = append(st.Phase, ss.Rng.Float64())
		st.FlashTick = append(st.FlashTick, 0)
	}

	// Advance phases and detect flashes
	flashers := make([]int, 0, 16)
	dt := 1.0 / float64(fireflyPeriod)
	for i := range ss.Bots {
		old := st.Phase[i]
		st.Phase[i] += dt
		if st.Phase[i] >= 1.0 {
			st.Phase[i] -= 1.0
			st.FlashTick[i] = ss.Tick
			flashers = append(flashers, i)
		}
		_ = old
	}

	// Coupling: flashers nudge nearby phases forward
	if ss.Hash != nil {
		for _, fi := range flashers {
			bot := &ss.Bots[fi]
			nearIDs := ss.Hash.Query(bot.X, bot.Y, fireflyRadius)
			for _, j := range nearIDs {
				if j == fi || j < 0 || j >= len(ss.Bots) || j >= len(st.Phase) {
					continue
				}
				nb := &ss.Bots[j]
				dx := bot.X - nb.X
				dy := bot.Y - nb.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > fireflyRadius || dist < 0.001 {
					continue
				}
				// Nudge proportional to distance (closer = stronger)
				strength := fireflyCoupleStr * (1.0 - dist/fireflyRadius)
				st.Phase[j] += strength
				if st.Phase[j] >= 1.0 {
					st.Phase[j] -= 1.0
					st.FlashTick[j] = ss.Tick
				}
			}
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].FlashPhase = int(st.Phase[i] * 255)
		ticksSinceFlash := ss.Tick - st.FlashTick[i]
		if ticksSinceFlash >= 0 && ticksSinceFlash < fireflyFlashDur {
			ss.Bots[i].FlashSync = 1
		} else {
			ss.Bots[i].FlashSync = 0
		}
	}
}

// ApplyFlash triggers an immediate flash for this bot (resets phase to 0).
func ApplyFlash(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Firefly == nil || idx >= len(ss.Firefly.Phase) {
		return
	}
	ss.Firefly.Phase[idx] = 0.0
	ss.Firefly.FlashTick[idx] = ss.Tick

	// Bright white flash LED
	bot.LEDColor = [3]uint8{255, 255, 200}
}
