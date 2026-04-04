package swarm

import "math"

// Predator Evasion Waves: Cascading escape waves through the swarm.
// Inspired by starling murmurations where alarm propagates as a wave.
// When a bot detects a "threat" (or is manually alarmed), it flees and
// triggers neighbors to flee too, creating expanding wavefronts.
// The swarm splits, flows around the threat, and reforms beautifully.

const (
	evasionRadius     = 70.0  // alarm propagation radius
	evasionFleeSpeed  = 2.5   // speed multiplier when fleeing
	evasionDuration   = 40    // ticks of evasion behavior per bot
	evasionCooldown   = 60    // ticks before a bot can be alarmed again
	evasionPropDelay  = 3     // ticks delay before propagating alarm
	evasionSteerRate  = 0.30  // max flee steering per tick (radians)
)

// EvasionState holds per-bot evasion state.
type EvasionState struct {
	Alarmed   []bool    // is this bot currently evading?
	Timer     []int     // remaining evasion ticks
	Cooldown  []int     // cooldown before re-alarm
	FleeAngle []float64 // direction to flee
	PropTimer []int     // delay before propagating to neighbors
}

// InitEvasion allocates evasion wave state.
func InitEvasion(ss *SwarmState) {
	n := len(ss.Bots)
	ss.Evasion = &EvasionState{
		Alarmed:   make([]bool, n),
		Timer:     make([]int, n),
		Cooldown:  make([]int, n),
		FleeAngle: make([]float64, n),
		PropTimer: make([]int, n),
	}
	ss.EvasionOn = true
}

// ClearEvasion frees evasion state.
func ClearEvasion(ss *SwarmState) {
	ss.Evasion = nil
	ss.EvasionOn = false
}

// TickEvasion propagates alarm waves and updates sensor cache.
// Sets EvasionAlert (0/1) and EvasionWave (ticks since alarm started).
func TickEvasion(ss *SwarmState) {
	if ss.Evasion == nil {
		return
	}
	st := ss.Evasion

	// Grow slices
	for len(st.Alarmed) < len(ss.Bots) {
		st.Alarmed = append(st.Alarmed, false)
		st.Timer = append(st.Timer, 0)
		st.Cooldown = append(st.Cooldown, 0)
		st.FleeAngle = append(st.FleeAngle, 0)
		st.PropTimer = append(st.PropTimer, 0)
	}

	// Propagation: alarmed bots trigger nearby non-alarmed bots
	if ss.Hash != nil {
		newAlarms := make([]int, 0, 16)
		for i := range ss.Bots {
			if !st.Alarmed[i] || st.PropTimer[i] > 0 {
				continue
			}
			bot := &ss.Bots[i]
			nearIDs := ss.Hash.Query(bot.X, bot.Y, evasionRadius)
			for _, j := range nearIDs {
				if j == i || j < 0 || j >= len(ss.Bots) {
					continue
				}
				if st.Alarmed[j] || st.Cooldown[j] > 0 {
					continue
				}
				nb := &ss.Bots[j]
				dx := bot.X - nb.X
				dy := bot.Y - nb.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > evasionRadius {
					continue
				}
				newAlarms = append(newAlarms, j)
			}
			// Only propagate once (set prop timer high so it doesn't re-trigger)
			st.PropTimer[i] = evasionDuration
		}

		// Apply new alarms
		for _, j := range newAlarms {
			if st.Alarmed[j] {
				continue
			}
			st.Alarmed[j] = true
			st.Timer[j] = evasionDuration
			st.PropTimer[j] = evasionPropDelay

			// Flee direction: away from the nearest alarmed bot
			bot := &ss.Bots[j]
			bestDist := math.MaxFloat64
			fleeAngle := bot.Angle + math.Pi // default: reverse
			nearIDs := ss.Hash.Query(bot.X, bot.Y, evasionRadius)
			for _, k := range nearIDs {
				if k == j || k < 0 || k >= len(ss.Bots) || !st.Alarmed[k] {
					continue
				}
				dx := bot.X - ss.Bots[k].X
				dy := bot.Y - ss.Bots[k].Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < bestDist && dist > 0.001 {
					bestDist = dist
					fleeAngle = math.Atan2(dy, dx) // away from threat
				}
			}
			st.FleeAngle[j] = fleeAngle
		}
	}

	// Update timers and sensor cache
	for i := range ss.Bots {
		if st.Cooldown[i] > 0 {
			st.Cooldown[i]--
		}
		if st.PropTimer[i] > 0 {
			st.PropTimer[i]--
		}
		if st.Alarmed[i] {
			st.Timer[i]--
			if st.Timer[i] <= 0 {
				st.Alarmed[i] = false
				st.Cooldown[i] = evasionCooldown
			}
		}

		if st.Alarmed[i] {
			ss.Bots[i].EvasionAlert = 1
			ss.Bots[i].EvasionWave = evasionDuration - st.Timer[i]
		} else {
			ss.Bots[i].EvasionAlert = 0
			ss.Bots[i].EvasionWave = 0
		}
	}
}

// ApplyEvade triggers this bot to start an evasion alarm (the "predator detected" event).
func ApplyEvade(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Evasion == nil || idx >= len(ss.Evasion.Alarmed) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Evasion

	if !st.Alarmed[idx] && st.Cooldown[idx] <= 0 {
		st.Alarmed[idx] = true
		st.Timer[idx] = evasionDuration
		st.PropTimer[idx] = evasionPropDelay
		st.FleeAngle[idx] = bot.Angle + math.Pi // flee backward
	}

	// If alarmed, steer toward flee direction
	if st.Alarmed[idx] {
		diff := st.FleeAngle[idx] - bot.Angle
		diff = WrapAngle(diff)
		if diff > evasionSteerRate {
			diff = evasionSteerRate
		} else if diff < -evasionSteerRate {
			diff = -evasionSteerRate
		}
		bot.Angle += diff
		bot.Speed = SwarmBotSpeed * evasionFleeSpeed
	} else {
		bot.Speed = SwarmBotSpeed
	}

	// Red flash when alarmed
	if st.Alarmed[idx] {
		bot.LEDColor = [3]uint8{255, 30, 0}
	}
}
