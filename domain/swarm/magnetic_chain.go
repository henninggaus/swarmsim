package swarm

import "math"

// Magnetic Dipole Chains: Bots behave like magnetic dipoles, attracting
// head-to-tail and repelling side-to-side. Creates self-organizing chains
// and rings reminiscent of ferrofluid or magnetotactic bacteria.

const (
	magAttractRange = 80.0  // attraction range
	magRepelRange   = 30.0  // repulsion range (too close)
	magAlignStr     = 0.12  // dipole alignment strength
	magAttractStr   = 0.08  // head-to-tail attraction
	magRepelStr     = 0.15  // side repulsion
	magChainDist    = 20.0  // ideal chain spacing
)

// MagState holds per-bot magnetic dipole state.
type MagState struct {
	Dipole    []float64 // dipole orientation angle
	ChainNext []int     // next bot in chain (-1 = tail)
	ChainPrev []int     // prev bot in chain (-1 = head)
	ChainLen  []int     // length of chain this bot belongs to
}

// InitMagnetic allocates magnetic state.
func InitMagnetic(ss *SwarmState) {
	n := len(ss.Bots)
	st := &MagState{
		Dipole:    make([]float64, n),
		ChainNext: make([]int, n),
		ChainPrev: make([]int, n),
		ChainLen:  make([]int, n),
	}
	for i := range st.ChainNext {
		st.Dipole[i] = ss.Bots[i].Angle
		st.ChainNext[i] = -1
		st.ChainPrev[i] = -1
		st.ChainLen[i] = 1
	}
	ss.Magnetic = st
	ss.MagneticOn = true
}

// ClearMagnetic frees magnetic state.
func ClearMagnetic(ss *SwarmState) {
	ss.Magnetic = nil
	ss.MagneticOn = false
}

// TickMagnetic updates dipole interactions and chain detection.
func TickMagnetic(ss *SwarmState) {
	if ss.Magnetic == nil || ss.Hash == nil {
		return
	}
	st := ss.Magnetic

	// Grow slices
	for len(st.Dipole) < len(ss.Bots) {
		st.Dipole = append(st.Dipole, 0)
		st.ChainNext = append(st.ChainNext, -1)
		st.ChainPrev = append(st.ChainPrev, -1)
		st.ChainLen = append(st.ChainLen, 1)
	}

	// Reset chains
	for i := range st.ChainNext {
		st.ChainNext[i] = -1
		st.ChainPrev[i] = -1
	}

	// Detect chains: two bots are linked if aligned and close head-to-tail
	for i := range ss.Bots {
		nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, magChainDist*1.5)
		for _, j := range nearIDs {
			if j <= i || j >= len(ss.Bots) {
				continue
			}
			dx := ss.Bots[j].X - ss.Bots[i].X
			dy := ss.Bots[j].Y - ss.Bots[i].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > magChainDist*1.5 || dist < 1 {
				continue
			}

			// Check if j is in front of i (head-to-tail)
			angleToJ := math.Atan2(dy, dx)
			diffI := angleToJ - st.Dipole[i]
			diffI = WrapAngle(diffI)

			// Dipole alignment check
			dipoleDiff := st.Dipole[j] - st.Dipole[i]
			dipoleDiff = WrapAngle(dipoleDiff)

			if math.Abs(diffI) < math.Pi/3 && math.Abs(dipoleDiff) < math.Pi/3 {
				if st.ChainNext[i] == -1 && st.ChainPrev[j] == -1 {
					st.ChainNext[i] = j
					st.ChainPrev[j] = i
				}
			}
		}
	}

	// Compute chain lengths
	for i := range st.ChainLen {
		st.ChainLen[i] = 1
	}
	for i := range ss.Bots {
		if st.ChainPrev[i] != -1 {
			continue // not chain head
		}
		// Walk chain from head
		length := 1
		curr := i
		for st.ChainNext[curr] != -1 {
			length++
			curr = st.ChainNext[curr]
		}
		// Set length for all members
		curr = i
		for curr != -1 {
			st.ChainLen[curr] = length
			curr = st.ChainNext[curr]
		}
	}

	// Update dipole orientations and sensor cache
	for i := range ss.Bots {
		st.Dipole[i] = ss.Bots[i].Angle
		ss.Bots[i].MagChainLen = st.ChainLen[i]
		if st.ChainNext[i] != -1 || st.ChainPrev[i] != -1 {
			ss.Bots[i].MagLinked = 1
		} else {
			ss.Bots[i].MagLinked = 0
		}

		// Dipole alignment with neighbors
		nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, magAttractRange)
		alignSum := 0.0
		alignCount := 0
		for _, j := range nearIDs {
			if j == i || j < 0 || j >= len(ss.Bots) {
				continue
			}
			alignSum += math.Cos(st.Dipole[j] - st.Dipole[i])
			alignCount++
		}
		if alignCount > 0 {
			ss.Bots[i].MagAlign = int((alignSum / float64(alignCount)) * 100)
		} else {
			ss.Bots[i].MagAlign = 0
		}
	}
}

// ApplyMagnetic applies magnetic dipole forces to the bot.
func ApplyMagnetic(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Magnetic == nil || ss.Hash == nil || idx >= len(ss.Magnetic.Dipole) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Magnetic

	nearIDs := ss.Hash.Query(bot.X, bot.Y, magAttractRange)
	var steerX, steerY float64

	for _, j := range nearIDs {
		if j == idx || j < 0 || j >= len(ss.Bots) {
			continue
		}
		other := &ss.Bots[j]
		dx := other.X - bot.X
		dy := other.Y - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 1 {
			continue
		}

		// Angle from bot to other
		angleToOther := math.Atan2(dy, dx)
		headToTail := angleToOther - st.Dipole[idx]
		headToTail = WrapAngle(headToTail)

		if dist < magRepelRange {
			// Repulsion at close range
			steerX -= (dx / dist) * magRepelStr
			steerY -= (dy / dist) * magRepelStr
		} else if math.Abs(headToTail) < math.Pi/3 {
			// Head-to-tail attraction
			w := magAttractStr * (1 - dist/magAttractRange)
			steerX += (dx / dist) * w
			steerY += (dy / dist) * w
		}

		// Dipole alignment torque
		_ = other
		dipoleDiff := st.Dipole[j] - st.Dipole[idx]
		dipoleDiff = WrapAngle(dipoleDiff)
		bot.Angle += dipoleDiff * magAlignStr * 0.1
	}

	// Apply steering
	steerMag := math.Sqrt(steerX*steerX + steerY*steerY)
	if steerMag > 0.01 {
		targetAngle := math.Atan2(steerY, steerX)
		diff := targetAngle - bot.Angle
		diff = WrapAngle(diff)
		if diff > 0.15 {
			diff = 0.15
		} else if diff < -0.15 {
			diff = -0.15
		}
		bot.Angle += diff
	}

	bot.Speed = SwarmBotSpeed * 0.8

	// LED: chain members glow blue, stronger for longer chains
	chainLen := st.ChainLen[idx]
	intensity := uint8(math.Min(255, float64(chainLen)*40))
	if chainLen > 1 {
		bot.LEDColor = [3]uint8{50, intensity / 2, intensity}
	} else {
		bot.LEDColor = [3]uint8{100, 100, 100} // gray unlinked
	}
}
