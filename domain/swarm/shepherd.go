package swarm

import "math"

// Shepherd-Flock (Hütehund): One or more bots act as shepherds that herd
// the rest of the swarm toward a target location. Uses Strömbom et al.'s
// model: shepherd positions behind the flock (relative to target) and
// drives them forward. Creates beautiful herding dynamics.

const (
	shepherdDriveRadius = 120.0 // shepherd influence range
	shepherdDrivePower  = 0.25  // steering strength on flock
	shepherdSpeed       = 2.0   // speed multiplier for shepherd
	shepherdCollectDist = 80.0  // threshold: flock too spread → collect first
	shepherdSteerRate   = 0.20  // max steering per tick
)

// ShepherdState holds shepherd/flock herding state.
type ShepherdState struct {
	IsShepherd []bool   // which bots are shepherds
	TargetX    float64  // herd target location X
	TargetY    float64  // herd target location Y
	FlockCX    float64  // flock center of mass X
	FlockCY    float64  // flock center of mass Y
	FlockSpread float64 // flock spread (avg dist from center)
}

// InitShepherd allocates shepherd state. First bot becomes shepherd.
func InitShepherd(ss *SwarmState) {
	n := len(ss.Bots)
	st := &ShepherdState{
		IsShepherd: make([]bool, n),
		TargetX:    ss.ArenaW * 0.8, // default target: right side
		TargetY:    ss.ArenaH / 2,
	}
	// First ~2% of bots (min 1) are shepherds
	numShepherds := n / 50
	if numShepherds < 1 {
		numShepherds = 1
	}
	for i := 0; i < numShepherds && i < n; i++ {
		st.IsShepherd[i] = true
	}
	ss.Shepherd = st
	ss.ShepherdOn = true
}

// ClearShepherd frees shepherd state.
func ClearShepherd(ss *SwarmState) {
	ss.Shepherd = nil
	ss.ShepherdOn = false
}

// TickShepherd computes flock center and updates sensor cache.
func TickShepherd(ss *SwarmState) {
	if ss.Shepherd == nil {
		return
	}
	st := ss.Shepherd

	// Grow slices
	for len(st.IsShepherd) < len(ss.Bots) {
		st.IsShepherd = append(st.IsShepherd, false)
	}

	// Compute flock center of mass (non-shepherd bots only)
	var sumX, sumY float64
	flockCount := 0
	for i := range ss.Bots {
		if !st.IsShepherd[i] {
			sumX += ss.Bots[i].X
			sumY += ss.Bots[i].Y
			flockCount++
		}
	}
	if flockCount > 0 {
		st.FlockCX = sumX / float64(flockCount)
		st.FlockCY = sumY / float64(flockCount)
	}

	// Compute flock spread
	var spreadSum float64
	for i := range ss.Bots {
		if !st.IsShepherd[i] {
			dx := ss.Bots[i].X - st.FlockCX
			dy := ss.Bots[i].Y - st.FlockCY
			spreadSum += math.Sqrt(dx*dx + dy*dy)
		}
	}
	if flockCount > 0 {
		st.FlockSpread = spreadSum / float64(flockCount)
	}

	// Update sensor cache
	for i := range ss.Bots {
		if st.IsShepherd[i] {
			ss.Bots[i].ShepherdRole = 1
		} else {
			ss.Bots[i].ShepherdRole = 0
		}

		// Distance to flock center
		dx := ss.Bots[i].X - st.FlockCX
		dy := ss.Bots[i].Y - st.FlockCY
		ss.Bots[i].ShepherdDist = int(math.Min(9999, math.Sqrt(dx*dx+dy*dy)))

		// Distance from flock center to target
		tdx := st.TargetX - st.FlockCX
		tdy := st.TargetY - st.FlockCY
		ss.Bots[i].FlockToTarget = int(math.Min(9999, math.Sqrt(tdx*tdx+tdy*tdy)))
	}
}

// ApplyShepherd drives the flock toward the target (shepherd behavior).
func ApplyShepherd(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Shepherd == nil || idx >= len(ss.Shepherd.IsShepherd) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Shepherd

	if !st.IsShepherd[idx] {
		// Non-shepherd: flee from nearby shepherds
		applyFlockFlee(bot, ss, idx)
		return
	}

	// Shepherd behavior: position behind flock (opposite side from target) and drive
	fcx, fcy := st.FlockCX, st.FlockCY
	tx, ty := st.TargetX, st.TargetY

	// Direction from flock to target
	dtx := tx - fcx
	dty := ty - fcy
	distToTarget := math.Sqrt(dtx*dtx + dty*dty)

	if distToTarget < 30 {
		// Flock is at target — orbit around
		bot.Angle += 0.05
		bot.Speed = SwarmBotSpeed * 0.5
		bot.LEDColor = [3]uint8{0, 255, 0}
		return
	}

	var goalX, goalY float64

	if st.FlockSpread > shepherdCollectDist {
		// Flock too spread: find the most outlying bot and drive it toward center
		bestDist := 0.0
		bestX, bestY := fcx, fcy
		for i := range ss.Bots {
			if st.IsShepherd[i] {
				continue
			}
			dx := ss.Bots[i].X - fcx
			dy := ss.Bots[i].Y - fcy
			d := math.Sqrt(dx*dx + dy*dy)
			if d > bestDist {
				bestDist = d
				bestX = ss.Bots[i].X
				bestY = ss.Bots[i].Y
			}
		}
		// Position behind the outlier (away from flock center)
		dx := bestX - fcx
		dy := bestY - fcy
		d := math.Sqrt(dx*dx + dy*dy)
		if d > 0.001 {
			goalX = bestX + (dx/d)*30
			goalY = bestY + (dy/d)*30
		} else {
			goalX = bestX + 30
			goalY = bestY
		}
	} else {
		// Flock compact: position behind flock and push toward target
		nd := math.Sqrt(dtx*dtx + dty*dty)
		if nd > 0.001 {
			// Behind flock = opposite of target direction
			goalX = fcx - (dtx/nd)*60
			goalY = fcy - (dty/nd)*60
		} else {
			goalX = fcx - 60
			goalY = fcy
		}
	}

	// Steer toward goal position
	dx := goalX - bot.X
	dy := goalY - bot.Y
	targetAngle := math.Atan2(dy, dx)
	diff := targetAngle - bot.Angle
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	if diff > shepherdSteerRate {
		diff = shepherdSteerRate
	} else if diff < -shepherdSteerRate {
		diff = -shepherdSteerRate
	}
	bot.Angle += diff
	bot.Speed = SwarmBotSpeed * shepherdSpeed
	bot.LEDColor = [3]uint8{255, 50, 50} // red shepherd
}

// applyFlockFlee makes a flock bot flee from nearby shepherds.
func applyFlockFlee(bot *SwarmBot, ss *SwarmState, idx int) {
	st := ss.Shepherd
	if ss.Hash == nil {
		bot.Speed = SwarmBotSpeed
		return
	}

	nearIDs := ss.Hash.Query(bot.X, bot.Y, shepherdDriveRadius)
	var fleeX, fleeY float64
	shepherdCount := 0
	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) || !st.IsShepherd[j] {
			continue
		}
		dx := bot.X - ss.Bots[j].X
		dy := bot.Y - ss.Bots[j].Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 0.001 && dist < shepherdDriveRadius {
			// Flee force weighted by proximity
			w := 1.0 - dist/shepherdDriveRadius
			fleeX += (dx / dist) * w
			fleeY += (dy / dist) * w
			shepherdCount++
		}
	}

	if shepherdCount > 0 {
		fleeAngle := math.Atan2(fleeY, fleeX)
		diff := fleeAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		steer := shepherdDrivePower
		if diff > steer {
			diff = steer
		} else if diff < -steer {
			diff = -steer
		}
		bot.Angle += diff
		bot.Speed = SwarmBotSpeed * 1.3 // scared flock moves faster
		bot.LEDColor = [3]uint8{200, 200, 255} // pale blue flock
	} else {
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{150, 150, 200}
	}
}
