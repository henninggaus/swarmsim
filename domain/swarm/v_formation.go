package swarm

import "math"

// V-Formation (Gänseflug): Bots self-organize into V-shaped flight
// formation like migrating geese. Trailing bots save energy by drafting
// behind leaders. Leader rotates when tired. Creates beautiful emergent V.
// Based on aerodynamic uplift models and observed avian behavior.

const (
	vfDraftAngle   = 0.65  // ~37° half-angle of V
	vfDraftRange   = 50.0  // range to detect draft benefit
	vfLeaderTicks  = 200   // ticks before leader rotation
	vfSteerRate    = 0.12  // steering toward formation position
	vfSpacing      = 25.0  // ideal spacing between bots in V
)

// VFormationState holds V-formation state.
type VFormationState struct {
	LeaderIdx    int       // current leader bot index
	LeaderTimer  int       // ticks remaining as leader
	MigAngle     float64   // migration direction angle
	InFormation  []bool    // is this bot in V position?
	FormPos      []int     // position in V: 0=leader, +N=right wing, -N=left wing
	Energy       []float64 // energy savings from drafting (0-1)
}

// InitVFormation allocates V-formation state.
func InitVFormation(ss *SwarmState) {
	n := len(ss.Bots)
	st := &VFormationState{
		LeaderIdx:   0,
		LeaderTimer: vfLeaderTicks,
		MigAngle:    0, // migrate rightward initially
		InFormation: make([]bool, n),
		FormPos:     make([]int, n),
		Energy:      make([]float64, n),
	}
	ss.VFormation = st
	ss.VFormationOn = true
}

// ClearVFormation frees V-formation state.
func ClearVFormation(ss *SwarmState) {
	ss.VFormation = nil
	ss.VFormationOn = false
}

// TickVFormation updates formation positions, leader rotation, draft energy.
func TickVFormation(ss *SwarmState) {
	if ss.VFormation == nil {
		return
	}
	st := ss.VFormation

	// Grow slices
	for len(st.InFormation) < len(ss.Bots) {
		st.InFormation = append(st.InFormation, false)
		st.FormPos = append(st.FormPos, 0)
		st.Energy = append(st.Energy, 0)
	}

	// Leader rotation
	st.LeaderTimer--
	if st.LeaderTimer <= 0 {
		// Find next leader: bot with most energy
		bestEnergy := -1.0
		bestIdx := st.LeaderIdx
		for i := range ss.Bots {
			if i == st.LeaderIdx {
				continue
			}
			if st.Energy[i] > bestEnergy {
				bestEnergy = st.Energy[i]
				bestIdx = i
			}
		}
		st.LeaderIdx = bestIdx
		st.LeaderTimer = vfLeaderTicks
	}

	// Ensure leader is valid
	if st.LeaderIdx >= len(ss.Bots) {
		st.LeaderIdx = 0
	}

	leader := &ss.Bots[st.LeaderIdx]

	// Compute ideal V positions relative to leader
	for i := range ss.Bots {
		if i == st.LeaderIdx {
			st.FormPos[i] = 0
			st.InFormation[i] = true
			st.Energy[i] *= 0.99 // leader loses energy
			continue
		}

		// Assign wing position: alternate left/right
		pos := i
		if pos > st.LeaderIdx {
			pos = pos - st.LeaderIdx
		} else {
			pos = pos + (len(ss.Bots) - st.LeaderIdx)
		}
		if pos%2 == 0 {
			st.FormPos[i] = pos / 2 // right wing
		} else {
			st.FormPos[i] = -(pos + 1) / 2 // left wing
		}

		// Check if bot is in draft zone of any bot ahead
		bot := &ss.Bots[i]
		dx := bot.X - leader.X
		dy := bot.Y - leader.Y
		distToLeader := math.Sqrt(dx*dx + dy*dy)
		if distToLeader < vfDraftRange*float64(abs(st.FormPos[i])+1) {
			st.InFormation[i] = true
			st.Energy[i] += 0.01 // gain energy from drafting
			if st.Energy[i] > 1 {
				st.Energy[i] = 1
			}
		} else {
			st.InFormation[i] = false
			st.Energy[i] *= 0.995
		}
	}

	// Update sensor cache
	for i := range ss.Bots {
		ss.Bots[i].VFormPos = st.FormPos[i]
		if st.InFormation[i] {
			ss.Bots[i].VFormDraft = 1
		} else {
			ss.Bots[i].VFormDraft = 0
		}
		if i == st.LeaderIdx {
			ss.Bots[i].VFormLeader = 1
		} else {
			ss.Bots[i].VFormLeader = 0
		}
	}
}

// abs is defined in temporal_memory.go

// ApplyVFormation steers the bot into V-formation position.
func ApplyVFormation(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.VFormation == nil || idx >= len(ss.VFormation.FormPos) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.VFormation

	if idx == st.LeaderIdx {
		// Leader: fly in migration direction
		diff := st.MigAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > vfSteerRate {
			diff = vfSteerRate
		} else if diff < -vfSteerRate {
			diff = -vfSteerRate
		}
		bot.Angle += diff
		bot.Speed = SwarmBotSpeed
		bot.LEDColor = [3]uint8{255, 255, 100} // gold leader
		return
	}

	// Compute target position in V
	leader := &ss.Bots[st.LeaderIdx]
	pos := st.FormPos[idx]
	wingAngle := st.MigAngle + math.Pi // behind leader

	var targetX, targetY float64
	if pos > 0 {
		// Right wing
		offsetAngle := wingAngle + vfDraftAngle
		dist := float64(pos) * vfSpacing
		targetX = leader.X + math.Cos(offsetAngle)*dist
		targetY = leader.Y + math.Sin(offsetAngle)*dist
	} else {
		// Left wing
		offsetAngle := wingAngle - vfDraftAngle
		dist := float64(-pos) * vfSpacing
		targetX = leader.X + math.Cos(offsetAngle)*dist
		targetY = leader.Y + math.Sin(offsetAngle)*dist
	}

	// Steer toward target
	dx := targetX - bot.X
	dy := targetY - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist > 5 {
		targetAngle := math.Atan2(dy, dx)
		diff := targetAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > vfSteerRate {
			diff = vfSteerRate
		} else if diff < -vfSteerRate {
			diff = -vfSteerRate
		}
		bot.Angle += diff
		bot.Speed = SwarmBotSpeed
	} else {
		// In position: match leader heading
		diff := st.MigAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		bot.Angle += diff * 0.1
		bot.Speed = SwarmBotSpeed * 0.95
	}

	// LED: wing position → color gradient
	intensity := uint8(math.Min(255, 100+float64(abs(pos))*30))
	if pos > 0 {
		bot.LEDColor = [3]uint8{intensity, 150, 50} // orange-ish right
	} else {
		bot.LEDColor = [3]uint8{50, 150, intensity} // blue-ish left
	}
}
