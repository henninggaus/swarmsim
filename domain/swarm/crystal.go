package swarm

import "math"

// Crystallization: Bots self-organize into hexagonal lattice structures
// like real crystals. Each bot tries to maintain 6 equidistant neighbors.
// Defects heal over time, and grain boundaries form between domains.
// Based on Lennard-Jones potential for inter-particle forces.

const (
	crystalSpacing    = 22.0  // ideal neighbor distance
	crystalRepelStr   = 0.25  // repulsion strength (too close)
	crystalAttractStr = 0.12  // attraction strength (too far)
	crystalAlignStr   = 0.05  // hexagonal alignment torque
	crystalDamping    = 0.92  // velocity damping (settling)
	crystalMaxNeighbors = 6   // target neighbor count (hexagonal)
)

// CrystalState holds crystallization state.
type CrystalState struct {
	VelX       []float64 // per-bot velocity X
	VelY       []float64 // per-bot velocity Y
	NeighCount []int     // per-bot actual neighbor count
	Defect     []bool    // true if bot has wrong neighbor count (defect site)
	Settled    []bool    // true if bot has ~6 neighbors at correct distance
}

// InitCrystal allocates crystallization state.
func InitCrystal(ss *SwarmState) {
	n := len(ss.Bots)
	st := &CrystalState{
		VelX:       make([]float64, n),
		VelY:       make([]float64, n),
		NeighCount: make([]int, n),
		Defect:     make([]bool, n),
		Settled:    make([]bool, n),
	}
	ss.Crystal = st
	ss.CrystalOn = true
}

// ClearCrystal frees crystallization state.
func ClearCrystal(ss *SwarmState) {
	ss.Crystal = nil
	ss.CrystalOn = false
}

// TickCrystal computes Lennard-Jones-like forces for hexagonal lattice formation.
func TickCrystal(ss *SwarmState) {
	if ss.Crystal == nil {
		return
	}
	st := ss.Crystal

	// Grow slices
	for len(st.VelX) < len(ss.Bots) {
		st.VelX = append(st.VelX, 0)
		st.VelY = append(st.VelY, 0)
		st.NeighCount = append(st.NeighCount, 0)
		st.Defect = append(st.Defect, false)
		st.Settled = append(st.Settled, false)
	}

	// Compute forces for each bot
	for i := range ss.Bots {
		if i >= len(st.VelX) {
			break
		}

		var fx, fy float64
		neighCount := 0

		var neighbors []int
		if ss.Hash != nil {
			neighbors = ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, crystalSpacing*3)
		} else {
			neighbors = make([]int, len(ss.Bots))
			for j := range neighbors {
				neighbors[j] = j
			}
		}

		for _, j := range neighbors {
			if j == i || j < 0 || j >= len(ss.Bots) {
				continue
			}
			dx := ss.Bots[j].X - ss.Bots[i].X
			dy := ss.Bots[j].Y - ss.Bots[i].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 0.1 {
				continue
			}

			// Count neighbors within 1.5x spacing
			if dist < crystalSpacing*1.5 {
				neighCount++
			}

			// Lennard-Jones-like: attract if far, repel if close
			ratio := crystalSpacing / dist
			if dist < crystalSpacing {
				// Too close: repel
				force := crystalRepelStr * (ratio*ratio - 1)
				fx -= force * dx / dist
				fy -= force * dy / dist
			} else if dist < crystalSpacing*2.5 {
				// In range: attract to ideal distance
				force := crystalAttractStr * (1 - ratio)
				fx += force * dx / dist
				fy += force * dy / dist
			}
		}

		// Hexagonal alignment: nudge toward 60° multiples
		if neighCount > 0 && ss.Hash != nil {
			nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, crystalSpacing*1.8)
			for _, j := range nearIDs {
				if j == i || j < 0 || j >= len(ss.Bots) {
					continue
				}
				dx := ss.Bots[j].X - ss.Bots[i].X
				dy := ss.Bots[j].Y - ss.Bots[i].Y
				angle := math.Atan2(dy, dx)
				// Snap to nearest 60° slot
				slot := math.Round(angle / (math.Pi / 3))
				idealAngle := slot * (math.Pi / 3)
				angleDiff := idealAngle - angle
				// Tangential force to align
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > 0.1 && dist < crystalSpacing*1.5 {
					tx := -math.Sin(angle) * angleDiff * crystalAlignStr
					ty := math.Cos(angle) * angleDiff * crystalAlignStr
					fx += tx
					fy += ty
				}
			}
		}

		// Update velocity with damping
		st.VelX[i] = (st.VelX[i] + fx) * crystalDamping
		st.VelY[i] = (st.VelY[i] + fy) * crystalDamping

		// Cap velocity
		vel := math.Sqrt(st.VelX[i]*st.VelX[i] + st.VelY[i]*st.VelY[i])
		if vel > 2 {
			st.VelX[i] *= 2 / vel
			st.VelY[i] *= 2 / vel
		}

		// Track neighbor count
		st.NeighCount[i] = neighCount
		st.Defect[i] = neighCount != crystalMaxNeighbors && neighCount > 0
		st.Settled[i] = neighCount >= 5 && neighCount <= 7
	}

	// Update sensor cache
	for i := range ss.Bots {
		if i >= len(st.NeighCount) {
			break
		}
		ss.Bots[i].CrystalNeigh = st.NeighCount[i]
		if st.Defect[i] {
			ss.Bots[i].CrystalDefect = 1
		} else {
			ss.Bots[i].CrystalDefect = 0
		}
		if st.Settled[i] {
			ss.Bots[i].CrystalSettled = 1
		} else {
			ss.Bots[i].CrystalSettled = 0
		}
	}
}

// ApplyCrystal moves bot according to crystallization forces.
func ApplyCrystal(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Crystal == nil || idx >= len(ss.Crystal.VelX) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Crystal

	vx, vy := st.VelX[idx], st.VelY[idx]
	vel := math.Sqrt(vx*vx + vy*vy)

	if vel > 0.05 {
		bot.Angle = math.Atan2(vy, vx)
		bot.Speed = SwarmBotSpeed * math.Min(1.0, vel)
	} else {
		bot.Speed = 0 // settled
	}

	// LED: settled=green, defect=red, forming=cyan
	if st.Settled[idx] {
		bot.LEDColor = [3]uint8{50, 220, 80} // crystal green
	} else if st.Defect[idx] {
		bot.LEDColor = [3]uint8{220, 60, 60} // defect red
	} else {
		bot.LEDColor = [3]uint8{80, 180, 220} // forming cyan
	}
}
