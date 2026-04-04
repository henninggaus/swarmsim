package swarm

import (
	"math"
)

// Flocking implements Reynolds' Boids rules: Separation, Alignment, Cohesion.
// Each bot gets per-tick sensor values computed from local neighbors.

const (
	flockRadius     = 80.0  // neighbor detection radius for flocking
	flockSepRadius  = 30.0  // separation radius (too close!)
	flockMaxSteer   = 0.15  // max angle change per tick (radians)
	flockSepWeight  = 2.0   // separation priority multiplier
	flockAlignW     = 1.0   // alignment weight
	flockCohereW    = 1.0   // cohesion weight
)

// TickFlocking computes flocking sensor cache for all bots.
// Must be called after spatial hash is rebuilt.
func TickFlocking(ss *SwarmState) {
	if ss.Hash == nil {
		return
	}
	for i := range ss.Bots {
		computeFlockSensors(ss, i)
	}
}

// computeFlockSensors calculates Separation, Alignment, Cohesion for one bot.
func computeFlockSensors(ss *SwarmState, idx int) {
	bot := &ss.Bots[idx]
	nearIDs := ss.Hash.Query(bot.X, bot.Y, flockRadius)

	var (
		sepX, sepY     float64 // separation steering vector
		alignSin       float64 // for circular mean of neighbor headings
		alignCos       float64
		cohX, cohY     float64 // center of mass of neighbors
		nFlock         int     // neighbors in flock radius
		nSep           int     // neighbors in separation radius
	)

	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx, dy := NeighborDelta(bot.X, bot.Y, nb.X, nb.Y, ss)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.001 || dist > flockRadius {
			continue
		}

		nFlock++
		cohX += dx
		cohY += dy
		alignSin += math.Sin(nb.Angle)
		alignCos += math.Cos(nb.Angle)

		// Separation: inverse-distance weighted repulsion
		if dist < flockSepRadius {
			nSep++
			weight := (flockSepRadius - dist) / flockSepRadius
			sepX -= (dx / dist) * weight
			sepY -= (dy / dist) * weight
		}
	}

	if nFlock == 0 {
		bot.FlockAlign = 0
		bot.FlockCohesion = 0
		bot.FlockSeparation = 0
		return
	}

	// Alignment: angle difference to average neighbor heading
	avgAngle := math.Atan2(alignSin/float64(nFlock), alignCos/float64(nFlock))
	alignDiff := avgAngle - bot.Angle
	// Normalize to [-π, π]
	alignDiff = WrapAngle(alignDiff)
	bot.FlockAlign = int(alignDiff * 180 / math.Pi) // -180..180

	// Cohesion: distance to center of mass of neighbors
	cohX /= float64(nFlock)
	cohY /= float64(nFlock)
	bot.FlockCohesion = int(math.Sqrt(cohX*cohX + cohY*cohY))

	// Separation: urgency (0 = no pressure, 100 = critical)
	if nSep > 0 {
		sepMag := math.Sqrt(sepX*sepX + sepY*sepY)
		bot.FlockSeparation = int(math.Min(sepMag*50, 100))
	} else {
		bot.FlockSeparation = 0
	}
}

// ApplyFlock applies all three Reynolds rules to steer the bot.
// Combines separation, alignment, and cohesion into a single heading change.
func ApplyFlock(bot *SwarmBot, ss *SwarmState, idx int) {
	nearIDs := ss.Hash.Query(bot.X, bot.Y, flockRadius)

	var (
		sepX, sepY float64
		alignSin   float64
		alignCos   float64
		cohX, cohY float64
		nFlock     int
		nSep       int
	)

	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx, dy := NeighborDelta(bot.X, bot.Y, nb.X, nb.Y, ss)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.001 || dist > flockRadius {
			continue
		}

		nFlock++
		cohX += dx
		cohY += dy
		alignSin += math.Sin(nb.Angle)
		alignCos += math.Cos(nb.Angle)

		if dist < flockSepRadius {
			nSep++
			weight := (flockSepRadius - dist) / flockSepRadius
			sepX -= (dx / dist) * weight
			sepY -= (dy / dist) * weight
		}
	}

	if nFlock == 0 {
		bot.Speed = SwarmBotSpeed
		return
	}

	// Desired heading from each rule
	steerX := math.Cos(bot.Angle)
	steerY := math.Sin(bot.Angle)

	// Separation
	if nSep > 0 {
		sepLen := math.Sqrt(sepX*sepX + sepY*sepY)
		if sepLen > 0.001 {
			steerX += (sepX / sepLen) * flockSepWeight
			steerY += (sepY / sepLen) * flockSepWeight
		}
	}

	// Alignment
	avgAngle := math.Atan2(alignSin/float64(nFlock), alignCos/float64(nFlock))
	steerX += math.Cos(avgAngle) * flockAlignW
	steerY += math.Sin(avgAngle) * flockAlignW

	// Cohesion
	cohX /= float64(nFlock)
	cohY /= float64(nFlock)
	cohLen := math.Sqrt(cohX*cohX + cohY*cohY)
	if cohLen > 0.001 {
		steerX += (cohX / cohLen) * flockCohereW
		steerY += (cohY / cohLen) * flockCohereW
	}

	// Apply clamped steering
	desired := math.Atan2(steerY, steerX)
	diff := desired - bot.Angle
	diff = WrapAngle(diff)
	if diff > flockMaxSteer {
		diff = flockMaxSteer
	} else if diff < -flockMaxSteer {
		diff = -flockMaxSteer
	}
	bot.Angle += diff
	bot.Speed = SwarmBotSpeed
}

// ApplyAlign steers toward the average heading of neighbors (alignment only).
func ApplyAlign(bot *SwarmBot, ss *SwarmState, idx int) {
	nearIDs := ss.Hash.Query(bot.X, bot.Y, flockRadius)
	var sinSum, cosSum float64
	n := 0
	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx, dy := NeighborDelta(bot.X, bot.Y, nb.X, nb.Y, ss)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.001 || dist > flockRadius {
			continue
		}
		sinSum += math.Sin(nb.Angle)
		cosSum += math.Cos(nb.Angle)
		n++
	}
	if n == 0 {
		bot.Speed = SwarmBotSpeed
		return
	}
	target := math.Atan2(sinSum/float64(n), cosSum/float64(n))
	diff := target - bot.Angle
	diff = WrapAngle(diff)
	if diff > flockMaxSteer {
		diff = flockMaxSteer
	} else if diff < -flockMaxSteer {
		diff = -flockMaxSteer
	}
	bot.Angle += diff
	bot.Speed = SwarmBotSpeed
}

// ApplyCohere steers toward the center of mass of neighbors (cohesion only).
func ApplyCohere(bot *SwarmBot, ss *SwarmState, idx int) {
	nearIDs := ss.Hash.Query(bot.X, bot.Y, flockRadius)
	var cx, cy float64
	n := 0
	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		nb := &ss.Bots[j]
		dx, dy := NeighborDelta(bot.X, bot.Y, nb.X, nb.Y, ss)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 0.001 || dist > flockRadius {
			continue
		}
		cx += dx
		cy += dy
		n++
	}
	if n == 0 {
		bot.Speed = SwarmBotSpeed
		return
	}
	cx /= float64(n)
	cy /= float64(n)
	target := math.Atan2(cy, cx)
	diff := target - bot.Angle
	diff = WrapAngle(diff)
	if diff > flockMaxSteer {
		diff = flockMaxSteer
	} else if diff < -flockMaxSteer {
		diff = -flockMaxSteer
	}
	bot.Angle += diff
	bot.Speed = SwarmBotSpeed
}
