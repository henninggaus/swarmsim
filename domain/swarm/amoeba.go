package swarm

import "math"

// Amoeba Locomotion: The entire swarm moves as a single amoeba-like blob.
// Pseudopods extend from the front, cytoplasm flows through the interior,
// and the rear contracts. Bots on the membrane form a skin while interior
// bots stream forward. Creates collective organism-like movement.
// Based on sol-gel cytoplasmic streaming models.

const (
	amoebaCoherence    = 0.15  // cohesion force toward center
	amoebaPseudoStr    = 0.20  // pseudopod extension strength
	amoebaStreamStr    = 0.12  // interior streaming strength
	amoebaSkinDist     = 30.0  // distance threshold to detect membrane
	amoebaPseudoAngle  = 0.4   // half-angle of pseudopod cone (radians)
	amoebaDirectionTick = 300  // ticks between direction changes
)

// AmoebaState holds amoeba locomotion state.
type AmoebaState struct {
	CenterX    float64 // blob center X
	CenterY    float64 // blob center Y
	Direction  float64 // current movement direction angle
	DirTimer   int     // ticks until direction change
	IsSkin     []bool  // per-bot: on membrane?
	IsPseudo   []bool  // per-bot: in pseudopod zone?
}

// InitAmoeba allocates amoeba state.
func InitAmoeba(ss *SwarmState) {
	n := len(ss.Bots)
	st := &AmoebaState{
		CenterX:   ss.ArenaW / 2,
		CenterY:   ss.ArenaH / 2,
		Direction: 0,
		DirTimer:  amoebaDirectionTick,
		IsSkin:    make([]bool, n),
		IsPseudo:  make([]bool, n),
	}
	ss.Amoeba = st
	ss.AmoebaOn = true
}

// ClearAmoeba frees amoeba state.
func ClearAmoeba(ss *SwarmState) {
	ss.Amoeba = nil
	ss.AmoebaOn = false
}

// TickAmoeba computes center, membrane detection, pseudopod zones.
func TickAmoeba(ss *SwarmState) {
	if ss.Amoeba == nil {
		return
	}
	st := ss.Amoeba

	// Grow slices
	for len(st.IsSkin) < len(ss.Bots) {
		st.IsSkin = append(st.IsSkin, false)
		st.IsPseudo = append(st.IsPseudo, false)
	}

	n := float64(len(ss.Bots))
	if n == 0 {
		return
	}

	// Compute center of mass
	var sumX, sumY float64
	for i := range ss.Bots {
		sumX += ss.Bots[i].X
		sumY += ss.Bots[i].Y
	}
	st.CenterX = sumX / n
	st.CenterY = sumY / n

	// Direction change timer
	st.DirTimer--
	if st.DirTimer <= 0 {
		if ss.Rng != nil {
			st.Direction += (ss.Rng.Float64() - 0.5) * math.Pi * 0.5
		}
		st.DirTimer = amoebaDirectionTick
	}

	// Compute spread for membrane detection
	var maxDist float64
	for i := range ss.Bots {
		dx := ss.Bots[i].X - st.CenterX
		dy := ss.Bots[i].Y - st.CenterY
		d := math.Sqrt(dx*dx + dy*dy)
		if d > maxDist {
			maxDist = d
		}
	}
	skinThreshold := maxDist * 0.7
	if skinThreshold < amoebaSkinDist {
		skinThreshold = amoebaSkinDist
	}

	// Classify each bot
	for i := range ss.Bots {
		if i >= len(st.IsSkin) {
			break
		}
		dx := ss.Bots[i].X - st.CenterX
		dy := ss.Bots[i].Y - st.CenterY
		distToCenter := math.Sqrt(dx*dx + dy*dy)

		// Membrane: bots far from center
		st.IsSkin[i] = distToCenter > skinThreshold

		// Pseudopod: bots in front cone
		angleToBot := math.Atan2(dy, dx)
		angleDiff := angleToBot - st.Direction
		for angleDiff > math.Pi {
			angleDiff -= 2 * math.Pi
		}
		for angleDiff < -math.Pi {
			angleDiff += 2 * math.Pi
		}
		st.IsPseudo[i] = math.Abs(angleDiff) < amoebaPseudoAngle && distToCenter > skinThreshold*0.5

		// Sensor cache
		ss.Bots[i].AmoebaDistCenter = int(math.Min(9999, distToCenter))
		if st.IsSkin[i] {
			ss.Bots[i].AmoebaSkin = 1
		} else {
			ss.Bots[i].AmoebaSkin = 0
		}
		if st.IsPseudo[i] {
			ss.Bots[i].AmoebaPseudo = 1
		} else {
			ss.Bots[i].AmoebaPseudo = 0
		}
	}
}

// ApplyAmoeba executes amoeba-like movement: pseudopod extend, interior stream, rear contract.
func ApplyAmoeba(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Amoeba == nil || idx >= len(ss.Amoeba.IsSkin) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Amoeba

	dx := bot.X - st.CenterX
	dy := bot.Y - st.CenterY
	distToCenter := math.Sqrt(dx*dx + dy*dy)

	if st.IsPseudo[idx] {
		// Pseudopod: extend outward in movement direction
		diff := st.Direction - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > amoebaPseudoStr {
			diff = amoebaPseudoStr
		} else if diff < -amoebaPseudoStr {
			diff = -amoebaPseudoStr
		}
		bot.Angle += diff
		bot.Speed = SwarmBotSpeed * 1.3
		bot.LEDColor = [3]uint8{150, 255, 100} // bright green pseudopod
	} else if st.IsSkin[idx] {
		// Membrane: cohesion + slow movement in direction
		centerAngle := math.Atan2(-dy, -dx) // toward center
		moveAngle := st.Direction

		// Blend center attraction and movement direction
		blendAngle := math.Atan2(
			math.Sin(centerAngle)*0.6+math.Sin(moveAngle)*0.4,
			math.Cos(centerAngle)*0.6+math.Cos(moveAngle)*0.4,
		)

		diff := blendAngle - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > 0.15 {
			diff = 0.15
		} else if diff < -0.15 {
			diff = -0.15
		}
		bot.Angle += diff
		bot.Speed = SwarmBotSpeed * 0.8
		bot.LEDColor = [3]uint8{100, 200, 80} // membrane green
	} else {
		// Interior: stream toward movement direction
		diff := st.Direction - bot.Angle
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		if diff > amoebaStreamStr {
			diff = amoebaStreamStr
		} else if diff < -amoebaStreamStr {
			diff = -amoebaStreamStr
		}
		bot.Angle += diff

		// Also attract toward center slightly
		if distToCenter > amoebaSkinDist*2 {
			centerAngle := math.Atan2(-dy, -dx)
			cdiff := centerAngle - bot.Angle
			for cdiff > math.Pi {
				cdiff -= 2 * math.Pi
			}
			for cdiff < -math.Pi {
				cdiff += 2 * math.Pi
			}
			bot.Angle += cdiff * amoebaCoherence
		}

		bot.Speed = SwarmBotSpeed * 1.1
		bot.LEDColor = [3]uint8{60, 120, 60} // dark interior
	}
}
