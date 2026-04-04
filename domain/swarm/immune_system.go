package swarm

import "math"

// Immune Swarm Response: Some bots act as "antibody" cells that patrol
// the swarm. When a "pathogen" (anomalous bot or designated threat) is
// detected, antibodies swarm to it and neutralize it, then return to patrol.
// Inspired by adaptive immune response and inflammation cascades.
// (Note: This is a SwarmScript-compatible subsystem separate from the
// existing behavioral ImmuneState in immune.go.)

const (
	iswPatrolSpeed    = 0.8   // patrol speed multiplier
	iswChaseSpeed     = 1.6   // chase speed when pursuing pathogen
	iswDetectRadius   = 100.0 // radius to detect pathogens
	iswSignalRadius   = 150.0 // radius to signal other antibodies
	iswNeutralizeTime = 30    // ticks to neutralize pathogen
	iswPathogenRate   = 200   // new pathogen appears every N ticks
)

// ImmuneSwarmState holds immune swarm state.
type ImmuneSwarmState struct {
	IsAntibody      []bool
	IsPathogen      []bool
	NeutralizeTimer []int     // >0 means being neutralized
	AlertLevel      []float64 // per-antibody alert level (0-1)
	SignalX         []float64 // last known pathogen X
	SignalY         []float64 // last known pathogen Y
	ActiveSignal    bool
}

// InitImmuneSwarm allocates immune swarm state. ~20% become antibodies.
func InitImmuneSwarm(ss *SwarmState) {
	n := len(ss.Bots)
	st := &ImmuneSwarmState{
		IsAntibody:      make([]bool, n),
		IsPathogen:      make([]bool, n),
		NeutralizeTimer: make([]int, n),
		AlertLevel:      make([]float64, n),
		SignalX:         make([]float64, n),
		SignalY:         make([]float64, n),
	}

	numAntibodies := n * 20 / 100
	if numAntibodies < 2 {
		numAntibodies = 2
	}
	for i := 0; i < numAntibodies && i < n; i++ {
		st.IsAntibody[i] = true
	}

	// Designate 1-2 pathogens initially
	numPathogens := n * 5 / 100
	if numPathogens < 1 {
		numPathogens = 1
	}
	for i := n - numPathogens; i < n; i++ {
		st.IsPathogen[i] = true
	}

	ss.ImmuneSwarm = st
	ss.ImmuneSwarmOn = true
}

// ClearImmuneSwarm frees immune swarm state.
func ClearImmuneSwarm(ss *SwarmState) {
	ss.ImmuneSwarm = nil
	ss.ImmuneSwarmOn = false
}

// TickImmuneSwarm updates pathogen detection, antibody signaling, neutralization.
func TickImmuneSwarm(ss *SwarmState) {
	if ss.ImmuneSwarm == nil {
		return
	}
	st := ss.ImmuneSwarm

	// Grow slices
	for len(st.IsAntibody) < len(ss.Bots) {
		st.IsAntibody = append(st.IsAntibody, false)
		st.IsPathogen = append(st.IsPathogen, false)
		st.NeutralizeTimer = append(st.NeutralizeTimer, 0)
		st.AlertLevel = append(st.AlertLevel, 0)
		st.SignalX = append(st.SignalX, 0)
		st.SignalY = append(st.SignalY, 0)
	}

	// Spawn new pathogens periodically
	if ss.Rng != nil && ss.Tick%iswPathogenRate == 0 && ss.Tick > 0 {
		for attempts := 0; attempts < 5; attempts++ {
			idx := ss.Rng.Intn(len(ss.Bots))
			if !st.IsAntibody[idx] && !st.IsPathogen[idx] {
				st.IsPathogen[idx] = true
				break
			}
		}
	}

	// Process neutralization
	for i := range ss.Bots {
		if !st.IsPathogen[i] {
			continue
		}

		if ss.Hash != nil {
			nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, 20)
			abCount := 0
			for _, j := range nearIDs {
				if j >= 0 && j < len(ss.Bots) && st.IsAntibody[j] {
					abCount++
				}
			}
			if abCount >= 2 {
				st.NeutralizeTimer[i]++
				if st.NeutralizeTimer[i] >= iswNeutralizeTime {
					st.IsPathogen[i] = false
					st.NeutralizeTimer[i] = 0
				}
			} else {
				if st.NeutralizeTimer[i] > 0 {
					st.NeutralizeTimer[i]--
				}
			}
		}
	}

	// Find active pathogen for signaling
	st.ActiveSignal = false
	for i := range ss.Bots {
		if st.IsPathogen[i] {
			st.ActiveSignal = true
			if ss.Hash != nil {
				// O(n·k): query only nearby antibodies via spatial hash
				nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, iswSignalRadius)
				for _, j := range nearIDs {
					if j >= len(st.IsAntibody) || !st.IsAntibody[j] {
						continue
					}
					dx := ss.Bots[i].X - ss.Bots[j].X
					dy := ss.Bots[i].Y - ss.Bots[j].Y
					dist := math.Sqrt(dx*dx + dy*dy)
					if dist < iswSignalRadius {
						st.AlertLevel[j] = math.Min(1.0, st.AlertLevel[j]+0.1)
						st.SignalX[j] = ss.Bots[i].X
						st.SignalY[j] = ss.Bots[i].Y
					}
				}
			} else {
				for j := range ss.Bots {
					if st.IsAntibody[j] {
						dx := ss.Bots[i].X - ss.Bots[j].X
						dy := ss.Bots[i].Y - ss.Bots[j].Y
						dist := math.Sqrt(dx*dx + dy*dy)
						if dist < iswSignalRadius {
							st.AlertLevel[j] = math.Min(1.0, st.AlertLevel[j]+0.1)
							st.SignalX[j] = ss.Bots[i].X
							st.SignalY[j] = ss.Bots[i].Y
						}
					}
				}
			}
			break // signal one pathogen at a time
		}
	}

	// Decay alert levels
	for i := range st.AlertLevel {
		st.AlertLevel[i] *= 0.98
	}

	// Collect pathogen indices once (O(n)) to avoid re-scanning per bot
	var pathogenIdxs []int
	for j := range ss.Bots {
		if j < len(st.IsPathogen) && st.IsPathogen[j] {
			pathogenIdxs = append(pathogenIdxs, j)
		}
	}

	// Update sensor cache — O(n × p) where p = pathogen count (typically very small)
	for i := range ss.Bots {
		if i >= len(st.IsAntibody) {
			break
		}
		if st.IsAntibody[i] {
			ss.Bots[i].ImmuneRole = 1
		} else if st.IsPathogen[i] {
			ss.Bots[i].ImmuneRole = 2
		} else {
			ss.Bots[i].ImmuneRole = 0
		}
		ss.Bots[i].ImmuneAlert = int(st.AlertLevel[i] * 100)

		// Distance to nearest pathogen
		nearestPath := 9999.0
		for _, j := range pathogenIdxs {
			if j == i {
				continue
			}
			dx := ss.Bots[j].X - ss.Bots[i].X
			dy := ss.Bots[j].Y - ss.Bots[i].Y
			d := math.Sqrt(dx*dx + dy*dy)
			if d < nearestPath {
				nearestPath = d
			}
		}
		ss.Bots[i].ImmunePathDist = int(math.Min(9999, nearestPath))
	}
}

// ApplyImmuneSwarm executes immune behavior: antibodies chase, pathogens flee.
func ApplyImmuneSwarm(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.ImmuneSwarm == nil || idx >= len(ss.ImmuneSwarm.IsAntibody) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.ImmuneSwarm

	if st.IsAntibody[idx] {
		if st.AlertLevel[idx] > 0.1 {
			// Chase: steer toward signaled pathogen location
			dx := st.SignalX[idx] - bot.X
			dy := st.SignalY[idx] - bot.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 5 {
				targetAngle := math.Atan2(dy, dx)
				diff := targetAngle - bot.Angle
				diff = WrapAngle(diff)
				if diff > 0.2 {
					diff = 0.2
				} else if diff < -0.2 {
					diff = -0.2
				}
				bot.Angle += diff
			}
			bot.Speed = SwarmBotSpeed * iswChaseSpeed
			alert := uint8(100 + st.AlertLevel[idx]*155)
			bot.LEDColor = [3]uint8{alert, 50, 50}
		} else {
			// Patrol
			bot.Speed = SwarmBotSpeed * iswPatrolSpeed
			bot.LEDColor = [3]uint8{200, 200, 200}
		}
	} else if st.IsPathogen[idx] {
		// Pathogen: flee from nearby antibodies
		if ss.Hash != nil {
			nearIDs := ss.Hash.Query(bot.X, bot.Y, iswDetectRadius)
			var fleeX, fleeY float64
			abCount := 0
			for _, j := range nearIDs {
				if j == idx || j < 0 || j >= len(ss.Bots) || !st.IsAntibody[j] {
					continue
				}
				dx := bot.X - ss.Bots[j].X
				dy := bot.Y - ss.Bots[j].Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > 0.001 {
					fleeX += dx / dist
					fleeY += dy / dist
					abCount++
				}
			}
			if abCount > 0 {
				fleeAngle := math.Atan2(fleeY, fleeX)
				diff := fleeAngle - bot.Angle
				diff = WrapAngle(diff)
				if diff > 0.25 {
					diff = 0.25
				} else if diff < -0.25 {
					diff = -0.25
				}
				bot.Angle += diff
				bot.Speed = SwarmBotSpeed * 1.3
			} else {
				bot.Speed = SwarmBotSpeed
			}
		}
		bot.LEDColor = [3]uint8{180, 0, 180}
	} else {
		// Normal cell
		bot.Speed = SwarmBotSpeed * 0.7
		bot.LEDColor = [3]uint8{100, 150, 100}
	}
}
