package swarm

import "math"

// Slime Mold Network: Physarum-inspired adaptive transport network.
// Bots deposit "slime trails" as they move. Trails between frequently
// visited locations (resource nodes) strengthen over time, while unused
// trails decay. This creates an efficient, self-organizing network
// similar to how Physarum polycephalum finds shortest paths.

const (
	slimeGridSize  = 20    // cell size for slime grid (pixels)
	slimeDeposit   = 0.15  // trail deposit per tick
	slimeDecay     = 0.005 // trail decay per tick
	slimeMax       = 1.0   // max trail intensity
	slimeRadius    = 40.0  // sensing radius for trail gradient
	slimeSteerRate = 0.12  // max steering toward trail (radians)
)

// SlimeState holds the global slime trail grid.
type SlimeState struct {
	Cols, Rows int
	CellSize   float64
	Grid       []float64 // flat array [row*cols+col], trail intensity [0, 1]
}

// InitSlime allocates slime mold network state.
func InitSlime(ss *SwarmState) {
	cols := int(ss.ArenaW) / slimeGridSize
	rows := int(ss.ArenaH) / slimeGridSize
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	ss.Slime = &SlimeState{
		Cols:     cols,
		Rows:     rows,
		CellSize: float64(slimeGridSize),
		Grid:     make([]float64, cols*rows),
	}
	ss.SlimeOn = true
}

// ClearSlime frees slime state.
func ClearSlime(ss *SwarmState) {
	ss.Slime = nil
	ss.SlimeOn = false
}

// TickSlime decays trails, deposits new trail from bot positions,
// and updates SlimeTrail (0-100) and SlimeGrad (angle to strongest gradient).
func TickSlime(ss *SwarmState) {
	if ss.Slime == nil {
		return
	}
	st := ss.Slime

	// Decay all trails
	for i := range st.Grid {
		st.Grid[i] -= slimeDecay
		if st.Grid[i] < 0 {
			st.Grid[i] = 0
		}
	}

	// Deposit trail from each bot's position
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		col := int(bot.X / st.CellSize)
		row := int(bot.Y / st.CellSize)
		if col < 0 || col >= st.Cols || row < 0 || row >= st.Rows {
			continue
		}
		idx := row*st.Cols + col
		// Carrying bots deposit more (reinforcing delivery paths)
		deposit := slimeDeposit
		if bot.CarryingPkg >= 0 {
			deposit *= 2.0
		}
		st.Grid[idx] += deposit
		if st.Grid[idx] > slimeMax {
			st.Grid[idx] = slimeMax
		}
	}

	// Update sensor cache: trail at bot position + gradient direction
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		col := int(bot.X / st.CellSize)
		row := int(bot.Y / st.CellSize)

		// Trail intensity at current position
		if col >= 0 && col < st.Cols && row >= 0 && row < st.Rows {
			bot.SlimeTrail = int(st.Grid[row*st.Cols+col] * 100)
		} else {
			bot.SlimeTrail = 0
		}

		// Gradient: find direction of strongest nearby trail
		bestVal := 0.0
		bestAngle := bot.Angle
		// Sample 8 directions
		for d := 0; d < 8; d++ {
			ang := float64(d) * math.Pi / 4.0
			sx := bot.X + math.Cos(ang)*slimeRadius*0.7
			sy := bot.Y + math.Sin(ang)*slimeRadius*0.7
			sc := int(sx / st.CellSize)
			sr := int(sy / st.CellSize)
			if sc < 0 || sc >= st.Cols || sr < 0 || sr >= st.Rows {
				continue
			}
			val := st.Grid[sr*st.Cols+sc]
			if val > bestVal {
				bestVal = val
				bestAngle = ang
			}
		}

		// Encode as angle relative to heading
		diff := bestAngle - bot.Angle
		diff = WrapAngle(diff)
		bot.SlimeGrad = int(diff * 180 / math.Pi)
	}
}

// ApplyFollowSlime steers the bot toward the strongest slime trail gradient.
func ApplyFollowSlime(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Slime == nil || bot.SlimeTrail == 0 {
		bot.Speed = SwarmBotSpeed
		return
	}

	diff := float64(bot.SlimeGrad) * math.Pi / 180
	if diff > slimeSteerRate {
		diff = slimeSteerRate
	} else if diff < -slimeSteerRate {
		diff = -slimeSteerRate
	}
	bot.Angle += diff
	bot.Speed = SwarmBotSpeed

	// Green glow when following slime
	trail := uint8(math.Min(255, float64(bot.SlimeTrail)*2.5))
	bot.LEDColor = [3]uint8{0, trail, trail / 3}
}
