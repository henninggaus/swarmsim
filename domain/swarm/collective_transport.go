package swarm

import "math"

// Collective Transport: Multiple bots coordinate to carry heavy objects.
// Inspired by ant colonies where multiple workers cooperate to move large
// food items. Bots sense nearby "transport tasks" (heavy packages) and
// recruit neighbors. When enough bots surround a heavy object, they
// collectively move it toward the nearest dropoff.

const (
	transportRadius    = 50.0  // detection radius for heavy objects
	transportMinBots   = 3     // minimum bots needed to start transport
	transportMaxBots   = 8     // maximum bots that can help
	transportSpeed     = 0.8   // speed multiplier when transporting
	transportAlignStr  = 0.20  // alignment strength toward transport heading
)

// TransportTask represents a heavy object that needs multiple bots to move.
type TransportTask struct {
	X, Y     float64 // current position
	TargetX  float64 // destination X
	TargetY  float64 // destination Y
	Weight   int     // how many bots needed (3-8)
	BotIDs   []int   // bots currently assisting
	Active   bool    // is this task active?
	Progress float64 // 0-1, delivery progress
}

// TransportState holds collective transport system state.
type TransportState struct {
	Tasks []TransportTask
}

// InitTransport allocates transport system state.
func InitTransport(ss *SwarmState) {
	ss.Transport = &TransportState{
		Tasks: make([]TransportTask, 0, 8),
	}
	ss.TransportOn = true
}

// ClearTransport frees transport state.
func ClearTransport(ss *SwarmState) {
	ss.Transport = nil
	ss.TransportOn = false
}

// TickTransport updates sensor cache for collective transport.
// Computes TransportNearby (count of heavy objects in range) and
// TransportCount (bots assisting nearest task).
func TickTransport(ss *SwarmState) {
	if ss.Transport == nil {
		return
	}
	st := ss.Transport

	// Update active tasks: move objects, check completion
	for ti := range st.Tasks {
		task := &st.Tasks[ti]
		if !task.Active {
			continue
		}
		// Count valid assisting bots (still nearby)
		alive := 0
		for _, bid := range task.BotIDs {
			if bid < 0 || bid >= len(ss.Bots) {
				continue
			}
			dx := ss.Bots[bid].X - task.X
			dy := ss.Bots[bid].Y - task.Y
			if math.Sqrt(dx*dx+dy*dy) < transportRadius*1.5 {
				alive++
			}
		}
		if alive < transportMinBots {
			// Not enough bots, stall
			continue
		}
		// Move task toward target
		dx := task.TargetX - task.X
		dy := task.TargetY - task.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 5.0 {
			task.Active = false
			task.Progress = 1.0
			continue
		}
		// Speed proportional to helper count
		speed := SwarmBotSpeed * transportSpeed * float64(alive) / float64(task.Weight)
		if speed > SwarmBotSpeed {
			speed = SwarmBotSpeed
		}
		task.X += (dx / dist) * speed
		task.Y += (dy / dist) * speed
		task.Progress = 1.0 - (dist / math.Max(1, math.Sqrt(
			(task.TargetX-task.X)*(task.TargetX-task.X)+
				(task.TargetY-task.Y)*(task.TargetY-task.Y))))
	}

	// Update bot sensors
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		nearbyTasks := 0
		bestDist := math.MaxFloat64
		bestTaskAssist := 0

		for _, task := range st.Tasks {
			if !task.Active {
				continue
			}
			dx := bot.X - task.X
			dy := bot.Y - task.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < transportRadius {
				nearbyTasks++
				if dist < bestDist {
					bestDist = dist
					bestTaskAssist = len(task.BotIDs)
				}
			}
		}
		bot.TransportNearby = nearbyTasks
		bot.TransportCount = bestTaskAssist
	}
}

// ApplyAssistTransport makes a bot join the nearest transport task.
func ApplyAssistTransport(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Transport == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Transport

	// Find nearest active task
	bestDist := math.MaxFloat64
	bestTI := -1
	for ti, task := range st.Tasks {
		if !task.Active {
			continue
		}
		dx := bot.X - task.X
		dy := bot.Y - task.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < transportRadius && dist < bestDist {
			bestDist = dist
			bestTI = ti
		}
	}

	if bestTI < 0 {
		// No task nearby, move forward normally
		bot.Speed = SwarmBotSpeed
		return
	}

	task := &st.Tasks[bestTI]

	// Check if already assisting
	alreadyIn := false
	for _, bid := range task.BotIDs {
		if bid == idx {
			alreadyIn = true
			break
		}
	}
	if !alreadyIn && len(task.BotIDs) < transportMaxBots {
		task.BotIDs = append(task.BotIDs, idx)
	}

	// Steer toward task position
	dx := task.X - bot.X
	dy := task.Y - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist > 1.0 {
		target := math.Atan2(dy, dx)
		diff := target - bot.Angle
		diff = WrapAngle(diff)
		if diff > transportAlignStr {
			diff = transportAlignStr
		} else if diff < -transportAlignStr {
			diff = -transportAlignStr
		}
		bot.Angle += diff
	}
	bot.Speed = SwarmBotSpeed * transportSpeed

	// Orange LED when assisting
	bot.LEDColor = [3]uint8{255, 160, 0}
}
