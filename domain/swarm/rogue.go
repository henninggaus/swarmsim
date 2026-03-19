package swarm

import "math"

// Rogue Bot Detection: peer-based anomaly detection for swarm integrity.
// Each bot monitors neighbors' behavior and maintains reputation scores.
// "Rogue" bots can emerge naturally (stuck, broken program) or be injected.

const (
	rogueCheckRadius  = 60.0  // range to monitor neighbors
	rogueSpeedThresh  = 0.3   // speed deviation threshold
	rogueAngleThresh  = 0.8   // heading deviation threshold (radians)
	rogueReputDecay   = 0.01  // natural reputation recovery per tick
	rogueFlagPenalty  = 15    // reputation penalty when flagged by neighbor
	rogueInitReputation = 100 // starting reputation
)

// TickRogue computes reputation and suspect sensors for all bots.
// Uses simple peer comparison: bots whose behavior deviates from
// their local neighborhood average are considered suspicious.
func TickRogue(ss *SwarmState) {
	if ss.Hash == nil {
		return
	}
	for i := range ss.Bots {
		computeRogueSensors(ss, i)
	}

	// Natural reputation recovery (every 10 ticks)
	if ss.Tick%10 == 0 {
		for i := range ss.Bots {
			if ss.Bots[i].Reputation < rogueInitReputation {
				ss.Bots[i].Reputation++
			}
		}
	}
}

// computeRogueSensors checks if any neighbor is behaving anomalously.
func computeRogueSensors(ss *SwarmState, idx int) {
	bot := &ss.Bots[idx]
	nearIDs := ss.Hash.Query(bot.X, bot.Y, rogueCheckRadius)

	// Initialize reputation if 0
	if bot.Reputation == 0 {
		bot.Reputation = rogueInitReputation
	}

	// Compute local averages
	var avgSpeed, avgSin, avgCos float64
	n := 0
	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx := bot.X - nb.X
		dy := bot.Y - nb.Y
		if math.Sqrt(dx*dx+dy*dy) > rogueCheckRadius {
			continue
		}
		avgSpeed += nb.Speed
		avgSin += math.Sin(nb.Angle)
		avgCos += math.Cos(nb.Angle)
		n++
	}

	if n < 2 {
		bot.SuspectNearby = 0
		return
	}

	avgSpeed /= float64(n)
	avgAngle := math.Atan2(avgSin/float64(n), avgCos/float64(n))

	// Check each neighbor for anomalous behavior
	bot.SuspectNearby = 0
	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx := bot.X - nb.X
		dy := bot.Y - nb.Y
		if math.Sqrt(dx*dx+dy*dy) > rogueCheckRadius {
			continue
		}

		// Speed anomaly
		speedDev := math.Abs(nb.Speed-avgSpeed) / (avgSpeed + 0.1)

		// Heading anomaly
		angleDiff := math.Abs(nb.Angle - avgAngle)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}

		// Combined anomaly score
		if speedDev > rogueSpeedThresh || angleDiff > rogueAngleThresh {
			// Check reputation: low rep neighbors are more suspect
			if nb.Reputation < 70 {
				bot.SuspectNearby = 1
				break
			}
		}
	}
}

// FlagRogue decreases the nearest neighbor's reputation.
func FlagRogue(bot *SwarmBot, ss *SwarmState, idx int) {
	nearIDs := ss.Hash.Query(bot.X, bot.Y, rogueCheckRadius)
	bestDist := math.MaxFloat64
	bestJ := -1

	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx := bot.X - nb.X
		dy := bot.Y - nb.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < bestDist {
			bestDist = dist
			bestJ = j
		}
	}

	if bestJ >= 0 {
		ss.Bots[bestJ].Reputation -= rogueFlagPenalty
		if ss.Bots[bestJ].Reputation < 0 {
			ss.Bots[bestJ].Reputation = 0
		}
	}
}
