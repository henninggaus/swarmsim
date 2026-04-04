// Package swarm — Real-time emergence detection.
// Detects when interesting swarm behaviors form and triggers explanatory popups.
package swarm

// EmergenceEvent identifies a type of emergent behavior.
type EmergenceEvent int

const (
	EmergenceClusterFormed EmergenceEvent = iota // bots formed a tight group
	EmergenceAlignmentHigh                        // bots moving in same direction
	EmergenceVortexDetected                       // circular motion pattern
	EmergenceDeliveryChain                        // bots forming delivery lines
	EmergenceEvolutionJump                        // fitness doubled in one generation
	EmergenceCoverageHigh                         // bots spread to 80%+ of arena
	EmergenceEventCount
)

// EmergencePopup holds data for a floating explanation near the action.
type EmergencePopup struct {
	Event    EmergenceEvent
	TitleKey string  // locale key for title
	TextKey  string  // locale key for explanation
	Timer    int     // display duration (ticks, counts down)
	X, Y     float64 // world coordinates where it happened
}

// DetectEmergence checks for emergent behaviors and returns a popup if a new one
// is detected. Only each type is shown once per session (tracked via EmergenceShown).
func DetectEmergence(ss *SwarmState) *EmergencePopup {
	if ss.PatternResult == nil {
		return nil
	}
	if ss.EmergenceShown == nil {
		ss.EmergenceShown = make(map[EmergenceEvent]bool)
	}
	pr := ss.PatternResult

	// 1. Cluster formation: high cohesion with multiple clusters
	if !ss.EmergenceShown[EmergenceClusterFormed] &&
		pr.Cohesion > 0.7 && pr.ClusterCount >= 3 {
		ss.EmergenceShown[EmergenceClusterFormed] = true
		return &EmergencePopup{
			Event:    EmergenceClusterFormed,
			TitleKey: "emerge.cluster.title",
			TextKey:  "emerge.cluster.text",
			Timer:    300,
			X:        ss.SwarmCenterX,
			Y:        ss.SwarmCenterY,
		}
	}

	// 2. Alignment: bots moving in the same direction
	if !ss.EmergenceShown[EmergenceAlignmentHigh] &&
		pr.Alignment > 0.8 {
		ss.EmergenceShown[EmergenceAlignmentHigh] = true
		return &EmergencePopup{
			Event:    EmergenceAlignmentHigh,
			TitleKey: "emerge.alignment.title",
			TextKey:  "emerge.alignment.text",
			Timer:    300,
			X:        ss.SwarmCenterX,
			Y:        ss.SwarmCenterY,
		}
	}

	// 3. Vortex: circular motion pattern
	if !ss.EmergenceShown[EmergenceVortexDetected] &&
		pr.Primary == PatternVortex && pr.PrimaryScore > 0.5 {
		ss.EmergenceShown[EmergenceVortexDetected] = true
		return &EmergencePopup{
			Event:    EmergenceVortexDetected,
			TitleKey: "emerge.vortex.title",
			TextKey:  "emerge.vortex.text",
			Timer:    300,
			X:        ss.SwarmCenterX,
			Y:        ss.SwarmCenterY,
		}
	}

	// 4. Delivery chain: multiple bots carrying in a line
	if !ss.EmergenceShown[EmergenceDeliveryChain] && ss.DeliveryOn {
		carryCount := 0
		for i := range ss.Bots {
			if ss.Bots[i].CarryingPkg >= 0 {
				carryCount++
			}
		}
		if carryCount >= 5 && pr.Alignment > 0.5 {
			ss.EmergenceShown[EmergenceDeliveryChain] = true
			return &EmergencePopup{
				Event:    EmergenceDeliveryChain,
				TitleKey: "emerge.delivery.title",
				TextKey:  "emerge.delivery.text",
				Timer:    300,
				X:        ss.SwarmCenterX,
				Y:        ss.SwarmCenterY,
			}
		}
	}

	// 5. Evolution jump: fitness doubled in one generation
	if !ss.EmergenceShown[EmergenceEvolutionJump] && ss.EvolutionOn {
		if len(ss.FitnessHistory) >= 2 {
			last := ss.FitnessHistory[len(ss.FitnessHistory)-1].Best
			prev := ss.FitnessHistory[len(ss.FitnessHistory)-2].Best
			if prev > 0 && last >= prev*2 {
				ss.EmergenceShown[EmergenceEvolutionJump] = true
				return &EmergencePopup{
					Event:    EmergenceEvolutionJump,
					TitleKey: "emerge.evojump.title",
					TextKey:  "emerge.evojump.text",
					Timer:    300,
					X:        ss.SwarmCenterX,
					Y:        ss.SwarmCenterY,
				}
			}
		}
	}

	// 6. Coverage high: bots spread to 80%+ of arena
	if !ss.EmergenceShown[EmergenceCoverageHigh] {
		coverage := computeCoverage(ss)
		if coverage > 0.8 {
			ss.EmergenceShown[EmergenceCoverageHigh] = true
			return &EmergencePopup{
				Event:    EmergenceCoverageHigh,
				TitleKey: "emerge.coverage.title",
				TextKey:  "emerge.coverage.text",
				Timer:    300,
				X:        ss.ArenaW / 2,
				Y:        ss.ArenaH / 2,
			}
		}
	}

	return nil
}

// computeCoverage estimates what fraction of the arena has bots nearby.
// Uses a coarse 10x10 grid and checks if each cell has at least one bot.
func computeCoverage(ss *SwarmState) float64 {
	const gridN = 10
	cellW := ss.ArenaW / gridN
	cellH := ss.ArenaH / gridN
	var covered [gridN][gridN]bool

	for i := range ss.Bots {
		gx := int(ss.Bots[i].X / cellW)
		gy := int(ss.Bots[i].Y / cellH)
		if gx >= 0 && gx < gridN && gy >= 0 && gy < gridN {
			covered[gx][gy] = true
		}
	}

	count := 0
	for x := 0; x < gridN; x++ {
		for y := 0; y < gridN; y++ {
			if covered[x][y] {
				count++
			}
		}
	}
	return float64(count) / float64(gridN*gridN)
}

// TickEmergencePopup decrements the popup timer.
func TickEmergencePopup(ss *SwarmState) {
	if ss.EmergencePopup != nil {
		ss.EmergencePopup.Timer--
		if ss.EmergencePopup.Timer <= 0 {
			ss.EmergencePopup = nil
		}
	}
}
