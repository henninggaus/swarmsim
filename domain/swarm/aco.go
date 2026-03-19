package swarm

import "math"

// Ant Colony Optimization (ACO): Bots deposit virtual pheromone trails
// on a grid as they travel between locations. Successful paths (deliveries)
// get stronger trails. Other bots follow strong trails → emergent shortest paths.
// Based on Dorigo's Ant System with evaporation and reinforcement.

const (
	acoCellSize       = 10.0  // pheromone grid cell size
	acoDepositRate    = 5.0   // pheromone deposited per tick while carrying
	acoEvapRate       = 0.995 // evaporation multiplier per tick
	acoFollowStr      = 0.18  // steering strength toward pheromone gradient
	acoReinforceFactor = 20.0 // bonus deposit on successful delivery
	acoMaxPheromone   = 100.0 // cap pheromone per cell
)

// ACOState holds ant colony optimization state.
type ACOState struct {
	GridCols int
	GridRows int
	Trail    []float64 // flat [row*cols+col] pheromone intensity
}

// InitACO allocates ACO state.
func InitACO(ss *SwarmState) {
	cols := int(ss.ArenaW/acoCellSize) + 1
	rows := int(ss.ArenaH/acoCellSize) + 1
	st := &ACOState{
		GridCols: cols,
		GridRows: rows,
		Trail:    make([]float64, cols*rows),
	}
	ss.ACO = st
	ss.ACOOn = true
}

// ClearACO frees ACO state.
func ClearACO(ss *SwarmState) {
	ss.ACO = nil
	ss.ACOOn = false
}

// TickACO evaporates trails and deposits pheromone for carrying bots.
func TickACO(ss *SwarmState) {
	if ss.ACO == nil {
		return
	}
	st := ss.ACO

	// Evaporate
	for i := range st.Trail {
		st.Trail[i] *= acoEvapRate
		if st.Trail[i] < 0.01 {
			st.Trail[i] = 0
		}
	}

	// Deposit pheromone from bots that are carrying
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.CarryingPkg >= 0 {
			col := int(bot.X / acoCellSize)
			row := int(bot.Y / acoCellSize)
			if col >= 0 && col < st.GridCols && row >= 0 && row < st.GridRows {
				idx := row*st.GridCols + col
				st.Trail[idx] += acoDepositRate
				if st.Trail[idx] > acoMaxPheromone {
					st.Trail[idx] = acoMaxPheromone
				}
			}
		}

		// Reinforce on delivery (TimeSinceDelivery just reset)
		if bot.TimeSinceDelivery == 1 {
			col := int(bot.X / acoCellSize)
			row := int(bot.Y / acoCellSize)
			if col >= 0 && col < st.GridCols && row >= 0 && row < st.GridRows {
				// Deposit strong pheromone around delivery point
				for dr := -2; dr <= 2; dr++ {
					for dc := -2; dc <= 2; dc++ {
						r, c := row+dr, col+dc
						if r >= 0 && r < st.GridRows && c >= 0 && c < st.GridCols {
							idx := r*st.GridCols + c
							st.Trail[idx] += acoReinforceFactor
							if st.Trail[idx] > acoMaxPheromone {
								st.Trail[idx] = acoMaxPheromone
							}
						}
					}
				}
			}
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		col := int(bot.X / acoCellSize)
		row := int(bot.Y / acoCellSize)

		// Current trail intensity
		trailHere := 0.0
		if col >= 0 && col < st.GridCols && row >= 0 && row < st.GridRows {
			trailHere = st.Trail[row*st.GridCols+col]
		}
		ss.Bots[i].ACOTrail = int(math.Min(100, trailHere))

		// Gradient: strongest adjacent cell direction
		bestVal := trailHere
		bestAngle := 0.0
		for _, dir := range [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			r, c := row+dir[0], col+dir[1]
			if r >= 0 && r < st.GridRows && c >= 0 && c < st.GridCols {
				val := st.Trail[r*st.GridCols+c]
				if val > bestVal {
					bestVal = val
					bestAngle = math.Atan2(float64(dir[0]), float64(dir[1]))
				}
			}
		}
		// Convert to relative angle
		if bestVal > trailHere+0.5 {
			relAngle := bestAngle - bot.Angle
			for relAngle > math.Pi {
				relAngle -= 2 * math.Pi
			}
			for relAngle < -math.Pi {
				relAngle += 2 * math.Pi
			}
			ss.Bots[i].ACOGrad = int(relAngle * 180 / math.Pi)
		} else {
			ss.Bots[i].ACOGrad = 0
		}
	}
}

// ApplyACO steers bot along pheromone gradient (follow strongest trail).
func ApplyACO(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.ACO == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.ACO

	col := int(bot.X / acoCellSize)
	row := int(bot.Y / acoCellSize)

	// Find gradient direction
	bestVal := 0.0
	bestDx, bestDy := 0.0, 0.0
	for _, dir := range [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		r, c := row+dir[0], col+dir[1]
		if r >= 0 && r < st.GridRows && c >= 0 && c < st.GridCols {
			val := st.Trail[r*st.GridCols+c]
			if val > bestVal {
				bestVal = val
				bestDx = float64(dir[1])
				bestDy = float64(dir[0])
			}
		}
	}

	if bestVal > 1.0 {
		targetAngle := math.Atan2(bestDy, bestDx)
		diff := targetAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > acoFollowStr {
			diff = acoFollowStr
		} else if diff < -acoFollowStr {
			diff = -acoFollowStr
		}
		bot.Angle += diff
		bot.Speed = SwarmBotSpeed
	} else {
		// No trail: wander
		bot.Speed = SwarmBotSpeed
	}

	// LED: trail intensity → orange glow
	trailHere := 0.0
	if col >= 0 && col < st.GridCols && row >= 0 && row < st.GridRows {
		trailHere = st.Trail[row*st.GridCols+col]
	}
	intensity := uint8(math.Min(255, 50+trailHere*2))
	bot.LEDColor = [3]uint8{intensity, uint8(float64(intensity) * 0.6), 30}
}
