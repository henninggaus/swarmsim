package swarm

import "math"

// Cell Division (Mitosis): The swarm periodically splits into two groups
// that separate, creating a dramatic division animation. After separation,
// the groups can re-merge, creating a pulsing division/merge cycle.
// Inspired by biological cell division and embryonic morphogenesis.

const (
	divisionCycle      = 300   // ticks per division cycle
	divisionSplitPhase = 0.3   // fraction of cycle for splitting
	divisionMergePhase = 0.7   // fraction of cycle where merge starts
	divisionForce      = 0.12  // force pushing groups apart
	divisionCohesion   = 0.06  // force keeping each group together
)

// DivisionState holds cell division state.
type DivisionState struct {
	GroupID   []int     // 0 or 1 — which group each bot belongs to
	Phase     float64   // 0.0-1.0 cycle phase
	SplitAxis float64   // angle of the division axis
	CenterAX  float64   // group A center X
	CenterAY  float64   // group A center Y
	CenterBX  float64   // group B center X
	CenterBY  float64   // group B center Y
	CycleCount int      // how many divisions completed
}

// InitDivision allocates cell division state.
func InitDivision(ss *SwarmState) {
	n := len(ss.Bots)
	st := &DivisionState{
		GroupID:   make([]int, n),
		SplitAxis: 0, // horizontal split initially
	}

	// Assign groups by position: above center = group 0, below = group 1
	midY := ss.ArenaH / 2
	for i := range ss.Bots {
		if ss.Bots[i].Y < midY {
			st.GroupID[i] = 0
		} else {
			st.GroupID[i] = 1
		}
	}

	ss.Division = st
	ss.DivisionOn = true
}

// ClearDivision frees cell division state.
func ClearDivision(ss *SwarmState) {
	ss.Division = nil
	ss.DivisionOn = false
}

// TickDivision advances division cycle and updates forces.
func TickDivision(ss *SwarmState) {
	if ss.Division == nil {
		return
	}
	st := ss.Division

	// Grow slices
	for len(st.GroupID) < len(ss.Bots) {
		st.GroupID = append(st.GroupID, 0)
	}

	// Advance phase
	st.Phase += 1.0 / float64(divisionCycle)
	if st.Phase >= 1.0 {
		st.Phase = 0
		st.CycleCount++
		// Rotate split axis each cycle
		st.SplitAxis += math.Pi / 3

		// Reassign groups based on new split axis
		cx, cy := ss.ArenaW/2, ss.ArenaH/2
		for i := range ss.Bots {
			dx := ss.Bots[i].X - cx
			dy := ss.Bots[i].Y - cy
			// Project onto split axis normal
			proj := dx*math.Cos(st.SplitAxis+math.Pi/2) + dy*math.Sin(st.SplitAxis+math.Pi/2)
			if proj >= 0 {
				st.GroupID[i] = 0
			} else {
				st.GroupID[i] = 1
			}
		}
	}

	// Compute group centers
	var sumAX, sumAY, sumBX, sumBY float64
	countA, countB := 0, 0
	for i := range ss.Bots {
		if st.GroupID[i] == 0 {
			sumAX += ss.Bots[i].X
			sumAY += ss.Bots[i].Y
			countA++
		} else {
			sumBX += ss.Bots[i].X
			sumBY += ss.Bots[i].Y
			countB++
		}
	}
	if countA > 0 {
		st.CenterAX = sumAX / float64(countA)
		st.CenterAY = sumAY / float64(countA)
	}
	if countB > 0 {
		st.CenterBX = sumBX / float64(countB)
		st.CenterBY = sumBY / float64(countB)
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].DivGroup = st.GroupID[i]
		ss.Bots[i].DivPhase = int(st.Phase * 100)

		// Distance to own group center
		var gcx, gcy float64
		if st.GroupID[i] == 0 {
			gcx, gcy = st.CenterAX, st.CenterAY
		} else {
			gcx, gcy = st.CenterBX, st.CenterBY
		}
		dx := ss.Bots[i].X - gcx
		dy := ss.Bots[i].Y - gcy
		ss.Bots[i].DivDist = int(math.Min(9999, math.Sqrt(dx*dx+dy*dy)))
	}
}

// ApplyDivision applies division forces to the bot.
func ApplyDivision(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Division == nil || idx >= len(ss.Division.GroupID) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Division

	var steerX, steerY float64

	// Own group center
	var gcx, gcy float64
	if st.GroupID[idx] == 0 {
		gcx, gcy = st.CenterAX, st.CenterAY
	} else {
		gcx, gcy = st.CenterBX, st.CenterBY
	}

	// Other group center
	var ocx, ocy float64
	if st.GroupID[idx] == 0 {
		ocx, ocy = st.CenterBX, st.CenterBY
	} else {
		ocx, ocy = st.CenterAX, st.CenterAY
	}

	if st.Phase < divisionSplitPhase {
		// Splitting phase: push apart from other group + cohere to own group
		// Repulsion from other group center
		dx := bot.X - ocx
		dy := bot.Y - ocy
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 0.001 {
			force := divisionForce * (1 - st.Phase/divisionSplitPhase) * 2
			steerX += (dx / dist) * force
			steerY += (dy / dist) * force
		}

		// Cohesion to own group
		dx = gcx - bot.X
		dy = gcy - bot.Y
		dist = math.Sqrt(dx*dx + dy*dy)
		if dist > 10 {
			steerX += (dx / dist) * divisionCohesion
			steerY += (dy / dist) * divisionCohesion
		}

	} else if st.Phase > divisionMergePhase {
		// Merging phase: attract toward global center
		cx, cy := ss.ArenaW/2, ss.ArenaH/2
		dx := cx - bot.X
		dy := cy - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 10 {
			mergeStr := divisionForce * (st.Phase - divisionMergePhase) / (1 - divisionMergePhase)
			steerX += (dx / dist) * mergeStr
			steerY += (dy / dist) * mergeStr
		}

	} else {
		// Stable separated phase: maintain cohesion in own group
		dx := gcx - bot.X
		dy := gcy - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 30 {
			steerX += (dx / dist) * divisionCohesion
			steerY += (dy / dist) * divisionCohesion
		}
	}

	// Apply steering
	steerMag := math.Sqrt(steerX*steerX + steerY*steerY)
	if steerMag > 0.01 {
		targetAngle := math.Atan2(steerY, steerX)
		diff := targetAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > 0.2 {
			diff = 0.2
		} else if diff < -0.2 {
			diff = -0.2
		}
		bot.Angle += diff
		bot.Speed = SwarmBotSpeed
	} else {
		bot.Speed = SwarmBotSpeed * 0.5
	}

	// LED: group A = magenta, group B = cyan, brightness varies with phase
	phase := st.Phase
	intensity := uint8(150 + 105*math.Sin(phase*2*math.Pi))
	if st.GroupID[idx] == 0 {
		bot.LEDColor = [3]uint8{intensity, 30, intensity / 2}
	} else {
		bot.LEDColor = [3]uint8{30, intensity / 2, intensity}
	}
}
