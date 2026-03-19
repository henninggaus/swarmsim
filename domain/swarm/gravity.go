package swarm

import "math"

// Gravitational N-Body: Bots simulate gravitational attraction like stars
// and planets. Each bot has a mass; heavier bots attract lighter ones.
// Creates orbits, binary systems, clusters, and galaxy-like spirals.
// Based on Newtonian gravity with softened potential to avoid singularities.

const (
	gravG         = 0.5    // gravitational constant
	gravSoftening = 15.0   // softening parameter to avoid singularities
	gravMaxForce  = 0.3    // max steering force per tick
	gravMassMin   = 0.5    // minimum bot mass
	gravMassMax   = 3.0    // maximum bot mass
	gravHeavyPct  = 10     // % of bots that are "heavy" (stars)
)

// GravityState holds gravitational N-body state.
type GravityState struct {
	Mass   []float64 // per-bot mass
	ForceX []float64 // accumulated force X
	ForceY []float64 // accumulated force Y
}

// InitGravity allocates gravity state. ~10% of bots get heavy mass.
func InitGravity(ss *SwarmState) {
	n := len(ss.Bots)
	st := &GravityState{
		Mass:   make([]float64, n),
		ForceX: make([]float64, n),
		ForceY: make([]float64, n),
	}

	numHeavy := n * gravHeavyPct / 100
	if numHeavy < 1 {
		numHeavy = 1
	}

	for i := 0; i < n; i++ {
		if i < numHeavy {
			st.Mass[i] = gravMassMax // heavy "stars"
		} else {
			st.Mass[i] = gravMassMin + (gravMassMax-gravMassMin)*0.3 // lighter
			if ss.Rng != nil {
				st.Mass[i] = gravMassMin + ss.Rng.Float64()*(gravMassMax-gravMassMin)*0.5
			}
		}
	}

	ss.Gravity = st
	ss.GravityOn = true
}

// ClearGravity frees gravity state.
func ClearGravity(ss *SwarmState) {
	ss.Gravity = nil
	ss.GravityOn = false
}

// TickGravity computes gravitational forces between all bots (using spatial hash for efficiency).
func TickGravity(ss *SwarmState) {
	if ss.Gravity == nil {
		return
	}
	st := ss.Gravity

	// Grow slices
	for len(st.Mass) < len(ss.Bots) {
		st.Mass = append(st.Mass, gravMassMin)
		st.ForceX = append(st.ForceX, 0)
		st.ForceY = append(st.ForceY, 0)
	}

	// Reset forces
	for i := range st.ForceX {
		st.ForceX[i] = 0
		st.ForceY[i] = 0
	}

	// Compute pairwise gravitational forces (use spatial hash if available)
	for i := range ss.Bots {
		if i >= len(st.Mass) {
			break
		}

		var neighbors []int
		if ss.Hash != nil {
			neighbors = ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, 200)
		} else {
			// Fallback: check all bots
			neighbors = make([]int, len(ss.Bots))
			for j := range neighbors {
				neighbors[j] = j
			}
		}

		for _, j := range neighbors {
			if j == i || j < 0 || j >= len(ss.Bots) || j >= len(st.Mass) {
				continue
			}
			dx := ss.Bots[j].X - ss.Bots[i].X
			dy := ss.Bots[j].Y - ss.Bots[i].Y
			distSq := dx*dx + dy*dy + gravSoftening*gravSoftening
			dist := math.Sqrt(distSq)

			// F = G * m1 * m2 / r^2
			force := gravG * st.Mass[i] * st.Mass[j] / distSq
			if force > gravMaxForce {
				force = gravMaxForce
			}

			// Direction: toward other bot
			st.ForceX[i] += force * dx / dist
			st.ForceY[i] += force * dy / dist
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		if i >= len(st.Mass) {
			break
		}
		ss.Bots[i].GravMass = int(st.Mass[i] * 100)

		// Force magnitude as sensor
		forceMag := math.Sqrt(st.ForceX[i]*st.ForceX[i] + st.ForceY[i]*st.ForceY[i])
		ss.Bots[i].GravForce = int(math.Min(100, forceMag*200))

		// Nearest heavy body distance
		nearHeavy := 9999.0
		for j := range ss.Bots {
			if j == i || j >= len(st.Mass) || st.Mass[j] < gravMassMax*0.8 {
				continue
			}
			dx := ss.Bots[j].X - ss.Bots[i].X
			dy := ss.Bots[j].Y - ss.Bots[i].Y
			d := math.Sqrt(dx*dx + dy*dy)
			if d < nearHeavy {
				nearHeavy = d
			}
		}
		ss.Bots[i].GravNearHeavy = int(math.Min(9999, nearHeavy))
	}
}

// ApplyGravity steers bot according to gravitational forces.
func ApplyGravity(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Gravity == nil || idx >= len(ss.Gravity.ForceX) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Gravity

	fx, fy := st.ForceX[idx], st.ForceY[idx]
	forceMag := math.Sqrt(fx*fx + fy*fy)

	if forceMag > 0.01 {
		targetAngle := math.Atan2(fy, fx)
		diff := targetAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		steerRate := math.Min(0.15, forceMag*0.5)
		if diff > steerRate {
			diff = steerRate
		} else if diff < -steerRate {
			diff = -steerRate
		}
		bot.Angle += diff
	}

	// Heavier bots move slower
	massRatio := st.Mass[idx] / gravMassMax
	bot.Speed = SwarmBotSpeed * (1.2 - 0.5*massRatio)

	// LED: mass-based color — heavy=yellow/white, light=blue
	if st.Mass[idx] >= gravMassMax*0.8 {
		bot.LEDColor = [3]uint8{255, 255, 150} // heavy star
	} else {
		intensity := uint8(80 + forceMag*300)
		if intensity > 255 {
			intensity = 255
		}
		bot.LEDColor = [3]uint8{50, 80, intensity} // light body
	}
}
