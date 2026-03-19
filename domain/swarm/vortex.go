package swarm

import "math"

// Vortex Swarming: Spontaneous rotating vortex formation.
// Inspired by fish schools and bacterial vortices. Bots tend to turn
// slightly toward the average perpendicular of their neighbors' headings,
// creating emergent clockwise/counter-clockwise rotation patterns.
// Produces a hypnotic visual effect.

const (
	vortexRadius     = 70.0  // neighbor detection radius
	vortexSteerRate  = 0.10  // max steering change per tick (radians)
	vortexBias       = 0.3   // perpendicular bias strength
	vortexCohereW    = 0.2   // cohesion weight (keep group together)
	vortexMinNeighbors = 2   // minimum neighbors to form vortex
)

// TickVortex computes vortex sensor values for all bots.
// Sets VortexStrength (0-100, how strong the local rotation is).
func TickVortex(ss *SwarmState) {
	if ss.Hash == nil {
		return
	}
	for i := range ss.Bots {
		computeVortexSensors(ss, i)
	}
}

// computeVortexSensors calculates local vortex strength for one bot.
func computeVortexSensors(ss *SwarmState, idx int) {
	bot := &ss.Bots[idx]
	nearIDs := ss.Hash.Query(bot.X, bot.Y, vortexRadius)

	var sinSum, cosSum float64
	n := 0
	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx := nb.X - bot.X
		dy := nb.Y - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.001 || dist > vortexRadius {
			continue
		}
		// Compute tangential (perpendicular) component of neighbor heading
		// relative to the vector from bot to neighbor
		vecAngle := math.Atan2(dy, dx)
		tangAngle := vecAngle + math.Pi/2 // perpendicular (CCW)
		// How much does the neighbor's heading align with the tangent?
		alignDiff := nb.Angle - tangAngle
		for alignDiff > math.Pi {
			alignDiff -= 2 * math.Pi
		}
		for alignDiff < -math.Pi {
			alignDiff += 2 * math.Pi
		}
		sinSum += math.Sin(alignDiff)
		cosSum += math.Cos(alignDiff)
		n++
	}

	if n < vortexMinNeighbors {
		bot.VortexStrength = 0
		return
	}

	// Vortex strength: how aligned neighbors are tangentially
	// cos(alignDiff) close to 1 means tangential alignment
	avgCos := cosSum / float64(n)
	bot.VortexStrength = int(math.Max(0, avgCos*100))
}

// ApplyVortex steers the bot to join or maintain a vortex pattern.
// Turns toward the perpendicular of neighbors' center of mass with a bias.
func ApplyVortex(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Hash == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	nearIDs := ss.Hash.Query(bot.X, bot.Y, vortexRadius)

	var cohX, cohY float64
	var tangSin, tangCos float64
	n := 0
	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx := nb.X - bot.X
		dy := nb.Y - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.001 || dist > vortexRadius {
			continue
		}
		cohX += dx
		cohY += dy

		// Tangential direction (perpendicular to radial vector)
		vecAngle := math.Atan2(dy, dx)
		tangAngle := vecAngle + math.Pi/2 // CCW rotation
		tangSin += math.Sin(tangAngle)
		tangCos += math.Cos(tangAngle)
		n++
	}

	if n < vortexMinNeighbors {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Combined steering: tangential + cohesion
	steerX := math.Cos(bot.Angle)
	steerY := math.Sin(bot.Angle)

	// Tangential component (vortex rotation)
	tangDir := math.Atan2(tangSin/float64(n), tangCos/float64(n))
	steerX += math.Cos(tangDir) * vortexBias
	steerY += math.Sin(tangDir) * vortexBias

	// Cohesion component (stay in group)
	cohX /= float64(n)
	cohY /= float64(n)
	cohLen := math.Sqrt(cohX*cohX + cohY*cohY)
	if cohLen > 0.001 {
		steerX += (cohX / cohLen) * vortexCohereW
		steerY += (cohY / cohLen) * vortexCohereW
	}

	// Apply clamped steering
	desired := math.Atan2(steerY, steerX)
	diff := desired - bot.Angle
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	if diff > vortexSteerRate {
		diff = vortexSteerRate
	} else if diff < -vortexSteerRate {
		diff = -vortexSteerRate
	}
	bot.Angle += diff
	bot.Speed = SwarmBotSpeed

	// Purple LED when vortexing
	bot.LEDColor = [3]uint8{180, 0, 255}
}
