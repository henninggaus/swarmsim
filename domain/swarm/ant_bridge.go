package swarm

import "math"

// Ant Bridge (Self-Assembly): Bots detect gaps or obstacles and form chains
// by locking together, creating living bridges that other bots can cross.
// Inspired by army ants (Eciton burchellii) that build bridges with their bodies.

const (
	bridgeDetectRadius = 50.0  // range to detect gap/obstacle
	bridgeLockDist     = 18.0  // distance to lock onto neighbor
	bridgeMaxLen       = 12    // max chain length
	bridgeDecayTicks   = 300   // bridge dissolves after this many idle ticks
	bridgeSteerRate    = 0.20  // steering rate toward bridge position
)

// BridgeState holds per-bot bridge assembly state.
type BridgeState struct {
	InBridge  []bool    // is this bot part of a bridge?
	ChainID   []int     // which bridge chain (-1 = none)
	ChainPos  []int     // position in chain (0 = anchor)
	LockAngle []float64 // angle bot is locked facing
	IdleTicks []int     // ticks since last bot crossed
	NextChain int       // next chain ID to assign
}

// InitBridge allocates bridge state.
func InitBridge(ss *SwarmState) {
	n := len(ss.Bots)
	ss.Bridge = &BridgeState{
		InBridge:  make([]bool, n),
		ChainID:   make([]int, n),
		ChainPos:  make([]int, n),
		LockAngle: make([]float64, n),
		IdleTicks: make([]int, n),
	}
	for i := range ss.Bridge.ChainID {
		ss.Bridge.ChainID[i] = -1
	}
	ss.BridgeOn = true
}

// ClearBridge frees bridge state.
func ClearBridge(ss *SwarmState) {
	ss.Bridge = nil
	ss.BridgeOn = false
}

// TickBridge updates bridge structures: decay idle bridges, update sensor cache.
func TickBridge(ss *SwarmState) {
	if ss.Bridge == nil {
		return
	}
	st := ss.Bridge

	// Grow slices if bots added
	for len(st.InBridge) < len(ss.Bots) {
		st.InBridge = append(st.InBridge, false)
		st.ChainID = append(st.ChainID, -1)
		st.ChainPos = append(st.ChainPos, 0)
		st.LockAngle = append(st.LockAngle, 0)
		st.IdleTicks = append(st.IdleTicks, 0)
	}

	// Increment idle ticks for bridge bots
	for i := range ss.Bots {
		if st.InBridge[i] {
			st.IdleTicks[i]++
			// Decay: release from bridge after timeout
			if st.IdleTicks[i] > bridgeDecayTicks {
				st.InBridge[i] = false
				st.ChainID[i] = -1
				st.ChainPos[i] = 0
			}
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		if st.InBridge[i] {
			ss.Bots[i].BridgeActive = 1
			ss.Bots[i].BridgePos = st.ChainPos[i]
		} else {
			ss.Bots[i].BridgeActive = 0
			ss.Bots[i].BridgePos = 0
		}

		// Detect nearby bridge bots
		if ss.Hash != nil {
			nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, bridgeDetectRadius)
			bridgeNear := 0
			for _, j := range nearIDs {
				if j == i || j < 0 || j >= len(ss.Bots) {
					continue
				}
				if st.InBridge[j] {
					bridgeNear++
				}
			}
			ss.Bots[i].BridgeNearby = bridgeNear
		}
	}
}

// ApplyFormBridge causes this bot to join or start a bridge chain near an obstacle.
func ApplyFormBridge(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Bridge == nil || idx >= len(ss.Bridge.InBridge) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Bridge

	// Already in bridge: stay locked
	if st.InBridge[idx] {
		bot.Angle = st.LockAngle[idx]
		bot.Speed = 0
		bot.LEDColor = [3]uint8{255, 200, 0} // gold bridge
		return
	}

	// Check if near obstacle → anchor a new bridge
	if bot.ObstacleAhead && bot.ObstacleDist < 30 {
		st.InBridge[idx] = true
		st.ChainID[idx] = st.NextChain
		st.NextChain++
		st.ChainPos[idx] = 0
		st.LockAngle[idx] = bot.Angle
		st.IdleTicks[idx] = 0
		bot.Speed = 0
		bot.LEDColor = [3]uint8{255, 200, 0}
		return
	}

	// Check if near an existing bridge bot → extend chain
	if ss.Hash != nil {
		nearIDs := ss.Hash.Query(bot.X, bot.Y, bridgeLockDist*1.5)
		bestDist := math.MaxFloat64
		bestJ := -1
		for _, j := range nearIDs {
			if j == idx || j < 0 || j >= len(ss.Bots) || !st.InBridge[j] {
				continue
			}
			// Don't extend too long
			if st.ChainPos[j] >= bridgeMaxLen-1 {
				continue
			}
			dx := bot.X - ss.Bots[j].X
			dy := bot.Y - ss.Bots[j].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < bestDist {
				bestDist = dist
				bestJ = j
			}
		}
		if bestJ >= 0 && bestDist < bridgeLockDist*1.5 {
			st.InBridge[idx] = true
			st.ChainID[idx] = st.ChainID[bestJ]
			st.ChainPos[idx] = st.ChainPos[bestJ] + 1
			st.LockAngle[idx] = st.LockAngle[bestJ]
			st.IdleTicks[idx] = 0
			bot.Speed = 0
			bot.LEDColor = [3]uint8{255, 200, 0}
			return
		}
	}

	// Not near obstacle or bridge: move toward nearest obstacle
	if bot.ObstacleAhead {
		bot.Speed = SwarmBotSpeed * 0.5
	} else {
		bot.Speed = SwarmBotSpeed
	}
}

// ApplyCrossBridge steers a non-bridge bot toward and across a nearby bridge.
func ApplyCrossBridge(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Bridge == nil || ss.Hash == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Bridge

	// Already in bridge: don't move
	if st.InBridge[idx] {
		bot.Speed = 0
		return
	}

	// Find nearest bridge bot and steer toward it
	nearIDs := ss.Hash.Query(bot.X, bot.Y, bridgeDetectRadius)
	bestDist := math.MaxFloat64
	bestAngle := bot.Angle
	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) || !st.InBridge[j] {
			continue
		}
		dx := ss.Bots[j].X - bot.X
		dy := ss.Bots[j].Y - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < bestDist {
			bestDist = dist
			bestAngle = math.Atan2(dy, dx)
		}
	}

	if bestDist < bridgeDetectRadius {
		diff := bestAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > bridgeSteerRate {
			diff = bridgeSteerRate
		} else if diff < -bridgeSteerRate {
			diff = -bridgeSteerRate
		}
		bot.Angle += diff

		// Reset idle ticks of bridge bots we pass near (they're being used)
		for _, j := range nearIDs {
			if j >= 0 && j < len(ss.Bots) && st.InBridge[j] {
				nb := &ss.Bots[j]
				dx := bot.X - nb.X
				dy := bot.Y - nb.Y
				if math.Sqrt(dx*dx+dy*dy) < bridgeLockDist {
					st.IdleTicks[j] = 0
				}
			}
		}
	}

	bot.Speed = SwarmBotSpeed
	bot.LEDColor = [3]uint8{200, 150, 50}
}
