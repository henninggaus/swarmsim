package swarm

import (
	"math"
	"swarmsim/logger"
)

// AdaptiveImmuneState models an adaptive immune system with B/T cell
// analogies. "B-cells" (detector bots) produce antibodies against threats.
// "T-cells" (helper bots) coordinate the response. "Memory cells" remember
// past threats for rapid re-response. First exposure is slow; second is 10x faster.
type AdaptiveImmuneState struct {
	ThreatMemory []ThreatSignature // memory of past threats
	ActiveThreats []ActiveThreat   // currently detected threats
	BotRoles      []ImmuneCellRole // per-bot immune role

	// Parameters
	DetectionRadius float64 // how far bots can sense threats (default 100)
	ResponseSpeed   float64 // base response speed multiplier (default 1.0)
	MemoryDuration  int     // ticks memory lasts (default 5000)
	MaxMemory       int     // max stored threat signatures (default 30)

	// Stats
	TotalThreats     int
	MemorizedThreats int
	ReExposures      int     // threats seen before (fast response)
	AvgResponseTime  float64 // average ticks to respond
}

// ThreatSignature represents a memorized threat pattern.
type ThreatSignature struct {
	// Threat characteristics (signature)
	SpeedProfile  float64 // how fast the threat appeared
	SizeProfile   float64 // how many bots were affected
	LocationHash  int     // rough area (grid cell)

	Exposures   int     // how many times seen
	LastSeen    int     // tick of last exposure
	ResponseEff float64 // effectiveness of response (0-1)
}

// ActiveThreat is a currently ongoing threat.
type ActiveThreat struct {
	X, Y        float64
	Severity    float64 // 0-1
	DetectedTick int
	Responding  int     // bots responding to this
	Memorized   bool    // have we seen this type before?
	SignatureIdx int    // index in memory (-1 if new)
}

// ImmuneCellRole defines a bot's role in the immune response.
type ImmuneCellRole struct {
	Role         int // 0=naive, 1=B-cell(detector), 2=T-cell(helper), 3=memory
	TargetThreat int // which active threat this bot is responding to (-1 = none)
	Activation   float64 // how activated this cell is (0-1)
}

const (
	ImmuneNaive   = 0
	ImmuneBCell   = 1
	ImmuneTCell   = 2
	ImmuneMemCell = 3
)

// InitAdaptiveImmune sets up the adaptive immune system.
func InitAdaptiveImmune(ss *SwarmState) {
	n := len(ss.Bots)
	ai := &AdaptiveImmuneState{
		ThreatMemory:    make([]ThreatSignature, 0, 30),
		ActiveThreats:   make([]ActiveThreat, 0, 10),
		BotRoles:        make([]ImmuneCellRole, n),
		DetectionRadius: 100,
		ResponseSpeed:   1.0,
		MemoryDuration:  5000,
		MaxMemory:       30,
	}

	// Assign initial roles
	for i := 0; i < n; i++ {
		r := ss.Rng.Float64()
		if r < 0.3 {
			ai.BotRoles[i].Role = ImmuneBCell
		} else if r < 0.5 {
			ai.BotRoles[i].Role = ImmuneTCell
		} else {
			ai.BotRoles[i].Role = ImmuneNaive
		}
		ai.BotRoles[i].TargetThreat = -1
	}

	ss.AdaptiveImmune = ai
	logger.Info("AIMM", "Adaptives Immunsystem: %d Zellen initialisiert", n)
}

// ClearAdaptiveImmune disables the adaptive immune system.
func ClearAdaptiveImmune(ss *SwarmState) {
	ss.AdaptiveImmune = nil
	ss.AdaptiveImmuneOn = false
}

// TickAdaptiveImmune runs one tick of the adaptive immune system.
func TickAdaptiveImmune(ss *SwarmState) {
	ai := ss.AdaptiveImmune
	if ai == nil {
		return
	}

	n := len(ss.Bots)
	if len(ai.BotRoles) != n {
		return
	}

	// Phase 1: Detect threats (anomalous patterns)
	detectImmuneThreats(ss, ai)

	// Phase 2: Match threats against memory
	matchThreatsToMemory(ai)

	// Phase 3: Respond to active threats
	respondToThreats(ss, ai)

	// Phase 4: Update memory
	updateImmuneMemory(ss, ai)

	// Phase 5: Visualize
	visualizeImmune(ss, ai)
}

// detectImmuneThreats finds anomalous patterns in the swarm.
func detectImmuneThreats(ss *SwarmState, ai *AdaptiveImmuneState) {
	if ss.Tick%20 != 0 {
		return
	}

	// Detect "threats": areas with sudden speed changes or clustering anomalies
	for i := range ss.Bots {
		if ai.BotRoles[i].Role != ImmuneBCell {
			continue
		}

		bot := &ss.Bots[i]

		// B-cells detect anomalies in their neighborhood
		anomalyScore := 0.0
		neighbors := 0
		avgNeighSpeed := 0.0

		for j := range ss.Bots {
			if i == j {
				continue
			}
			dx := ss.Bots[j].X - bot.X
			dy := ss.Bots[j].Y - bot.Y
			if dx*dx+dy*dy < ai.DetectionRadius*ai.DetectionRadius {
				avgNeighSpeed += ss.Bots[j].Speed
				neighbors++
			}
		}

		if neighbors > 0 {
			avgNeighSpeed /= float64(neighbors)
			speedDev := math.Abs(avgNeighSpeed - SwarmBotSpeed)
			anomalyScore = speedDev / SwarmBotSpeed

			// High neighbor count anomaly
			if neighbors > len(ss.Bots)/3 {
				anomalyScore += 0.3
			}
		}

		if anomalyScore > 0.4 {
			// New threat detected
			threat := ActiveThreat{
				X:            bot.X,
				Y:            bot.Y,
				Severity:      clampF(anomalyScore, 0, 1),
				DetectedTick: ss.Tick,
				SignatureIdx:  -1,
			}

			// Check if this overlaps existing threat
			overlap := false
			for t := range ai.ActiveThreats {
				dx := ai.ActiveThreats[t].X - threat.X
				dy := ai.ActiveThreats[t].Y - threat.Y
				if dx*dx+dy*dy < 2500 { // within 50 units
					overlap = true
					break
				}
			}

			if !overlap && len(ai.ActiveThreats) < 10 {
				ai.ActiveThreats = append(ai.ActiveThreats, threat)
				ai.TotalThreats++
			}
		}
	}
}

// matchThreatsToMemory checks if we've seen this type of threat before.
func matchThreatsToMemory(ai *AdaptiveImmuneState) {
	for t := range ai.ActiveThreats {
		if ai.ActiveThreats[t].SignatureIdx >= 0 {
			continue
		}

		threat := &ai.ActiveThreats[t]
		bestMatch := -1
		bestSim := 0.0

		for m, mem := range ai.ThreatMemory {
			sim := 1.0 - math.Abs(threat.Severity-mem.SizeProfile)
			if sim > 0.6 && sim > bestSim {
				bestSim = sim
				bestMatch = m
			}
		}

		if bestMatch >= 0 {
			threat.Memorized = true
			threat.SignatureIdx = bestMatch
			ai.ThreatMemory[bestMatch].Exposures++
			ai.ReExposures++
		}
	}
}

// respondToThreats assigns immune cells to threats.
func respondToThreats(ss *SwarmState, ai *AdaptiveImmuneState) {
	if len(ai.ActiveThreats) == 0 {
		// No threats: deactivate
		for i := range ai.BotRoles {
			ai.BotRoles[i].TargetThreat = -1
			ai.BotRoles[i].Activation *= 0.95
		}
		return
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		role := &ai.BotRoles[i]

		if role.Role == ImmuneNaive {
			continue
		}

		// Find nearest threat
		nearestThreat := -1
		nearestDist := math.MaxFloat64
		for t := range ai.ActiveThreats {
			dx := ai.ActiveThreats[t].X - bot.X
			dy := ai.ActiveThreats[t].Y - bot.Y
			dist := dx*dx + dy*dy
			if dist < nearestDist {
				nearestDist = dist
				nearestThreat = t
			}
		}

		if nearestThreat < 0 {
			continue
		}

		threat := &ai.ActiveThreats[nearestThreat]
		role.TargetThreat = nearestThreat

		// Response speed: 10x faster for memorized threats
		speed := ai.ResponseSpeed
		if threat.Memorized {
			speed *= 10.0
		}

		role.Activation = clampF(role.Activation+0.05*speed, 0, 1)

		// T-cells coordinate: move toward threat
		if role.Role == ImmuneTCell && role.Activation > 0.3 {
			targetAngle := math.Atan2(threat.Y-bot.Y, threat.X-bot.X)
			diff := targetAngle - bot.Angle
			for diff > math.Pi {
				diff -= 2 * math.Pi
			}
			for diff < -math.Pi {
				diff += 2 * math.Pi
			}
			bot.Angle += diff * 0.15 * speed
			bot.Speed *= 1.0 + 0.2*speed
			threat.Responding++
		}

		// B-cells produce "antibodies": slow down threat area
		if role.Role == ImmuneBCell && nearestDist < ai.DetectionRadius*ai.DetectionRadius {
			bot.Speed *= 0.9 // conserve energy, stay and detect
		}

		// Memory cells: rapid re-activation
		if role.Role == ImmuneMemCell && threat.Memorized {
			role.Activation = 1.0
		}
	}

	// Remove resolved threats (too old)
	alive := ai.ActiveThreats[:0]
	for _, t := range ai.ActiveThreats {
		if ss.Tick-t.DetectedTick < 200 {
			alive = append(alive, t)
		}
	}
	ai.ActiveThreats = alive
}

// updateImmuneMemory stores new threat signatures.
func updateImmuneMemory(ss *SwarmState, ai *AdaptiveImmuneState) {
	if ss.Tick%100 != 0 {
		return
	}

	// Promote resolved threats to memory
	for _, t := range ai.ActiveThreats {
		if t.SignatureIdx >= 0 {
			ai.ThreatMemory[t.SignatureIdx].LastSeen = ss.Tick
			continue
		}

		if len(ai.ThreatMemory) < ai.MaxMemory {
			sig := ThreatSignature{
				SizeProfile:  t.Severity,
				SpeedProfile: float64(t.Responding),
				Exposures:    1,
				LastSeen:     ss.Tick,
				ResponseEff:  0.5,
			}
			ai.ThreatMemory = append(ai.ThreatMemory, sig)
		}
	}

	// Promote some naive cells to memory cells after exposure
	for i := range ai.BotRoles {
		if ai.BotRoles[i].Role == ImmuneNaive && ai.BotRoles[i].Activation > 0.8 {
			ai.BotRoles[i].Role = ImmuneMemCell
		}
	}

	ai.MemorizedThreats = len(ai.ThreatMemory)

	// Expire old memories
	if len(ai.ThreatMemory) > 0 {
		fresh := ai.ThreatMemory[:0]
		for _, m := range ai.ThreatMemory {
			if ss.Tick-m.LastSeen < ai.MemoryDuration {
				fresh = append(fresh, m)
			}
		}
		ai.ThreatMemory = fresh
		ai.MemorizedThreats = len(ai.ThreatMemory)
	}
}

// visualizeImmune sets LED colors based on immune role.
func visualizeImmune(ss *SwarmState, ai *AdaptiveImmuneState) {
	for i := range ss.Bots {
		role := &ai.BotRoles[i]
		act := uint8(role.Activation * 200)

		switch role.Role {
		case ImmuneBCell:
			ss.Bots[i].LEDColor = [3]uint8{act, act, 200} // blue: detector
		case ImmuneTCell:
			ss.Bots[i].LEDColor = [3]uint8{200, act, act} // red: helper
		case ImmuneMemCell:
			ss.Bots[i].LEDColor = [3]uint8{act, 200, act} // green: memory
		}
	}
}

// AdaptiveImmuneThreats returns active threat count.
func AdaptiveImmuneThreats(ai *AdaptiveImmuneState) int {
	if ai == nil {
		return 0
	}
	return len(ai.ActiveThreats)
}

// AdaptiveImmuneMemory returns memorized threat count.
func AdaptiveImmuneMemory(ai *AdaptiveImmuneState) int {
	if ai == nil {
		return 0
	}
	return ai.MemorizedThreats
}

// AdaptiveImmuneReExposures returns re-exposure count.
func AdaptiveImmuneReExposures(ai *AdaptiveImmuneState) int {
	if ai == nil {
		return 0
	}
	return ai.ReExposures
}
