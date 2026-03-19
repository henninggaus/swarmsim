package swarm

import "math"

// Bee Waggle Dance: Bots communicate resource locations through movement.
// Inspired by honeybee waggle dance where the angle relative to the hive
// encodes direction to food, and duration encodes distance.
// "Dancing" bots perform figure-8 patterns. "Watching" bots nearby
// decode the dance and gain a heading toward the advertised target.

const (
	waggleRadius      = 60.0  // detection radius for watching a dance
	waggleDuration    = 60    // ticks for a full dance cycle
	waggleSteerRate   = 0.25  // radians per tick during dance figure-8
	waggleDecodeRange = 50.0  // how close to watch a dance
	wagglePhaseSwitch = 30    // ticks per half of the figure-8
)

// WaggleState holds per-bot dance state.
type WaggleState struct {
	Dancing    []bool    // is this bot currently dancing?
	DanceTick  []int     // ticks remaining in dance
	TargetX    []float64 // resource location being advertised
	TargetY    []float64
	DanceAngle []float64 // encoded direction (angle from bot to target)
}

// InitWaggle allocates waggle dance state.
func InitWaggle(ss *SwarmState) {
	n := len(ss.Bots)
	ss.Waggle = &WaggleState{
		Dancing:    make([]bool, n),
		DanceTick:  make([]int, n),
		TargetX:    make([]float64, n),
		TargetY:    make([]float64, n),
		DanceAngle: make([]float64, n),
	}
	ss.WaggleOn = true
}

// ClearWaggle frees waggle dance state.
func ClearWaggle(ss *SwarmState) {
	ss.Waggle = nil
	ss.WaggleOn = false
}

// TickWaggle updates dance state and sensor cache for all bots.
// Sets WaggleDancing (0/1), WaggleTarget (angle to decoded target, -180..180).
func TickWaggle(ss *SwarmState) {
	if ss.Waggle == nil {
		return
	}
	st := ss.Waggle

	// Grow slices if bots added
	for len(st.Dancing) < len(ss.Bots) {
		st.Dancing = append(st.Dancing, false)
		st.DanceTick = append(st.DanceTick, 0)
		st.TargetX = append(st.TargetX, 0)
		st.TargetY = append(st.TargetY, 0)
		st.DanceAngle = append(st.DanceAngle, 0)
	}

	// Update active dances
	for i := range ss.Bots {
		if st.Dancing[i] {
			st.DanceTick[i]--
			if st.DanceTick[i] <= 0 {
				st.Dancing[i] = false
			}
		}
	}

	// Decode: non-dancing bots near a dancer get heading info
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if st.Dancing[i] {
			bot.WaggleDancing = 1
			bot.WaggleTarget = 0 // dancer doesn't need decoded target
			continue
		}

		bot.WaggleDancing = 0
		bot.WaggleTarget = 0

		// Find nearest dancer within range
		if ss.Hash == nil {
			continue
		}
		nearIDs := ss.Hash.Query(bot.X, bot.Y, waggleRadius)
		bestDist := math.MaxFloat64
		bestJ := -1
		for _, j := range nearIDs {
			if j == i || j < 0 || j >= len(ss.Bots) || !st.Dancing[j] {
				continue
			}
			dx := bot.X - ss.Bots[j].X
			dy := bot.Y - ss.Bots[j].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < waggleDecodeRange && dist < bestDist {
				bestDist = dist
				bestJ = j
			}
		}

		if bestJ >= 0 {
			// Decode: angle from this bot to the dancer's advertised target
			dx := st.TargetX[bestJ] - bot.X
			dy := st.TargetY[bestJ] - bot.Y
			targetAngle := math.Atan2(dy, dx)
			diff := targetAngle - bot.Angle
			for diff > math.Pi {
				diff -= 2 * math.Pi
			}
			for diff < -math.Pi {
				diff += 2 * math.Pi
			}
			bot.WaggleTarget = int(diff * 180 / math.Pi)
		}
	}
}

// ApplyWaggleDance starts a waggle dance advertising the bot's last known
// resource location (nearest pickup with package, or nearest dropoff).
func ApplyWaggleDance(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Waggle == nil || idx >= len(ss.Waggle.Dancing) {
		bot.Speed = SwarmBotSpeed * 0.3
		return
	}
	st := ss.Waggle

	// Start dance if not already dancing
	if !st.Dancing[idx] {
		st.Dancing[idx] = true
		st.DanceTick[idx] = waggleDuration

		// Encode target: dance advertises the direction the bot is heading
		// (proxy for "I found something interesting this way")
		st.TargetX[idx] = bot.X + math.Cos(bot.Angle)*150
		st.TargetY[idx] = bot.Y + math.Sin(bot.Angle)*150
		st.DanceAngle[idx] = math.Atan2(st.TargetY[idx]-bot.Y, st.TargetX[idx]-bot.X)
	}

	// Perform figure-8 motion
	phase := st.DanceTick[idx] % waggleDuration
	if phase < wagglePhaseSwitch {
		// First half: waggle run (straight toward target direction)
		bot.Angle = st.DanceAngle[idx] + math.Sin(float64(phase)*0.5)*0.3
	} else {
		// Second half: return loop
		bot.Angle = st.DanceAngle[idx] + math.Pi + math.Sin(float64(phase)*0.5)*0.3
	}
	bot.Speed = SwarmBotSpeed * 0.5

	// Yellow-green LED while dancing
	bot.LEDColor = [3]uint8{200, 255, 0}
}

// ApplyFollowDance steers toward the decoded waggle dance target.
func ApplyFollowDance(bot *SwarmBot, ss *SwarmState, idx int) {
	if bot.WaggleTarget == 0 {
		bot.Speed = SwarmBotSpeed
		return
	}
	// Steer toward decoded direction
	diff := float64(bot.WaggleTarget) * math.Pi / 180
	if diff > waggleSteerRate {
		diff = waggleSteerRate
	} else if diff < -waggleSteerRate {
		diff = -waggleSteerRate
	}
	bot.Angle += diff
	bot.Speed = SwarmBotSpeed
}
