package swarm

import "math"

// ComputeSwarmCenter calculates the center of mass and spread of all bots.
func ComputeSwarmCenter(ss *SwarmState) {
	n := len(ss.Bots)
	if n == 0 {
		return
	}

	sumX, sumY := 0.0, 0.0
	for i := range ss.Bots {
		sumX += ss.Bots[i].X
		sumY += ss.Bots[i].Y
	}
	ss.SwarmCenterX = sumX / float64(n)
	ss.SwarmCenterY = sumY / float64(n)

	// Spread = average distance from center
	totalDist := 0.0
	for i := range ss.Bots {
		dx := ss.Bots[i].X - ss.SwarmCenterX
		dy := ss.Bots[i].Y - ss.SwarmCenterY
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}
	ss.SwarmSpread = totalDist / float64(n)
}

// UpdateCongestionGrid updates the congestion overlay grid.
func UpdateCongestionGrid(ss *SwarmState) {
	cols := 20
	rows := 20
	cellSize := ss.ArenaW / float64(cols)

	if ss.CongestionGrid == nil || ss.CongestionCols != cols {
		ss.CongestionGrid = make([]float64, cols*rows)
		ss.CongestionCols = cols
		ss.CongestionRows = rows
	}

	// Temporary count grid
	counts := make([]float64, cols*rows)
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.Speed > 0.3 && bot.StuckTicks < 10 {
			continue // not stuck
		}
		col := int(bot.X / cellSize)
		row := int(bot.Y / cellSize)
		if col < 0 {
			col = 0
		}
		if col >= cols {
			col = cols - 1
		}
		if row < 0 {
			row = 0
		}
		if row >= rows {
			row = rows - 1
		}
		counts[row*cols+col] += 1.0
	}

	// Normalize and apply exponential moving average
	for i := range ss.CongestionGrid {
		target := counts[i] / 5.0 // 5 stuck bots = full congestion
		if target > 1.0 {
			target = 1.0
		}
		ss.CongestionGrid[i] = ss.CongestionGrid[i]*0.9 + target*0.1
	}
}

// DayNightBrightness returns the current brightness factor (0.0=midnight, 1.0=noon).
func DayNightBrightness(ss *SwarmState) float64 {
	if !ss.DayNightOn {
		return 1.0
	}
	return 0.5 + 0.5*math.Cos(ss.DayNightPhase*2*math.Pi)
}

// ComputeSwarmAwarenessSensors computes per-bot swarm awareness sensor values.
func ComputeSwarmAwarenessSensors(ss *SwarmState) {
	n := len(ss.Bots)
	if n == 0 {
		return
	}

	// Compute center first (always needed for sensors)
	ComputeSwarmCenter(ss)
	spreadInt := int(ss.SwarmSpread)

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Distance to swarm center
		dx := bot.X - ss.SwarmCenterX
		dy := bot.Y - ss.SwarmCenterY
		bot.SwarmCenterDist = int(math.Sqrt(dx*dx + dy*dy))

		// Spread (same for all bots)
		bot.SwarmSpreadSensor = spreadInt

		// Isolation: if nearest neighbor > SwarmSensorRange, how far
		if bot.NearestDist > SwarmSensorRange {
			bot.IsolationLevel = int(bot.NearestDist - SwarmSensorRange)
		} else {
			bot.IsolationLevel = 0
		}

		// Resource gradient: direction toward nearest pickup station with package
		bot.ResourceGradientX = -1
		bot.ResourceGradientY = 0
		if ss.DeliveryOn {
			bestDist := math.MaxFloat64
			bestAngle := 0.0
			for si := range ss.Stations {
				st := &ss.Stations[si]
				if !st.IsPickup || !st.HasPackage {
					continue
				}
				sdx := st.X - bot.X
				sdy := st.Y - bot.Y
				dist := math.Sqrt(sdx*sdx + sdy*sdy)
				if dist < bestDist {
					bestDist = dist
					bestAngle = math.Atan2(sdy, sdx)
				}
			}
			if bestDist < math.MaxFloat64 {
				deg := bestAngle * 180 / math.Pi
				if deg < 0 {
					deg += 360
				}
				bot.ResourceGradientX = int(deg)
				// Magnitude: inverse of distance, scaled 0-100
				mag := 100.0 - bestDist/10.0
				if mag < 0 {
					mag = 0
				}
				if mag > 100 {
					mag = 100
				}
				bot.ResourceGradientY = int(mag)
			}
		}
	}
}
