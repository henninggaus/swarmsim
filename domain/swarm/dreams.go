package swarm

import (
	"math"
	"swarmsim/logger"
)

// DreamState manages offline experience replay for the swarm.
// When bots are idle or during low-activity phases, they "dream" by
// replaying stored experiences. This consolidates learning and allows
// bots to optimize behavior without new environmental input — similar
// to how animals consolidate memory during sleep.
type DreamState struct {
	Experiences [][]DreamExperience // per-bot experience buffers
	MaxBuffer   int                 // max experiences per bot (default 50)
	DreamRate   float64             // probability of dreaming per tick when idle (default 0.1)
	LearnRate   float64             // learning rate during dreams (default 0.02)
	BatchSize   int                 // experiences to replay per dream (default 5)

	// Stats
	TotalDreams      int
	TotalExperiences int
	AvgReplayValue   float64
	DreamPhase       bool // true during global dream phase
}

// DreamExperience stores a snapshot of a bot's state and outcome.
type DreamExperience struct {
	Tick       int
	X, Y       float64
	Angle      float64
	Action     int     // what the bot did (0=forward, 1=left, 2=right, 3=idle)
	Reward     float64 // outcome value (positive=good, negative=bad)
	PickupDist float64 // context: distance to nearest pickup
	Neighbors  int     // context: neighbor count
	Carrying   bool    // context: was carrying a package
}

// InitDreams sets up the dream/replay system.
func InitDreams(ss *SwarmState) {
	n := len(ss.Bots)
	ds := &DreamState{
		Experiences: make([][]DreamExperience, n),
		MaxBuffer:   50,
		DreamRate:   0.1,
		LearnRate:   0.02,
		BatchSize:   5,
	}

	for i := range ds.Experiences {
		ds.Experiences[i] = make([]DreamExperience, 0, ds.MaxBuffer)
	}

	ss.Dreams = ds
	logger.Info("DREAMS", "Initialisiert: %d Bots mit Erfahrungs-Replay, Buffer=%d", n, ds.MaxBuffer)
}

// ClearDreams disables the dream system.
func ClearDreams(ss *SwarmState) {
	ss.Dreams = nil
	ss.DreamsOn = false
}

// TickDreams runs one tick of the dream system.
func TickDreams(ss *SwarmState) {
	ds := ss.Dreams
	if ds == nil {
		return
	}

	n := len(ss.Bots)
	if len(ds.Experiences) != n {
		return
	}

	totalExp := 0
	totalReplay := 0.0
	replayCount := 0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		exps := &ds.Experiences[i]

		// Record current experience
		recordExperience(ss, bot, exps, ds)

		totalExp += len(*exps)

		// Dream: replay when idle or moving slowly
		isIdle := bot.Speed < SwarmBotSpeed*0.5
		if isIdle && ss.Rng.Float64() < ds.DreamRate && len(*exps) >= ds.BatchSize {
			val := replayExperiences(ss, bot, *exps, ds)
			totalReplay += val
			replayCount++
			ds.TotalDreams++
		}
	}

	ds.TotalExperiences = totalExp
	if replayCount > 0 {
		ds.AvgReplayValue = totalReplay / float64(replayCount)
	}

	// Global dream phase: every 500 ticks, all bots dream briefly
	ds.DreamPhase = ss.Tick%500 >= 490 && ss.Tick%500 < 500
	if ds.DreamPhase {
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			exps := ds.Experiences[i]
			if len(exps) >= ds.BatchSize {
				replayExperiences(ss, bot, exps, ds)
				ds.TotalDreams++

				// Visual: dreaming bots pulse purple
				phase := math.Sin(float64(ss.Tick) * 0.3)
				intensity := uint8(128 + phase*80)
				bot.LEDColor = [3]uint8{intensity, 50, intensity}
			}
		}
	}
}

// recordExperience stores the current state as an experience.
func recordExperience(ss *SwarmState, bot *SwarmBot, exps *[]DreamExperience, ds *DreamState) {
	// Only record every 10 ticks to avoid redundancy
	if ss.Tick%10 != 0 {
		return
	}

	// Compute reward based on current situation
	reward := 0.0
	if bot.CarryingPkg >= 0 && bot.NearestDropoffDist < 100 {
		reward += 0.5 // good: heading to dropoff with package
	}
	if bot.CarryingPkg < 0 && bot.NearestPickupDist < 50 {
		reward += 0.3 // good: near a pickup without package
	}
	if bot.NeighborCount > 8 {
		reward -= 0.2 // bad: too crowded
	}
	if bot.Speed < SwarmBotSpeed*0.3 {
		reward -= 0.1 // bad: barely moving
	}

	// Determine action from current angle change approximation
	action := 0 // forward by default

	exp := DreamExperience{
		Tick:       ss.Tick,
		X:          bot.X,
		Y:          bot.Y,
		Angle:      bot.Angle,
		Action:     action,
		Reward:     reward,
		PickupDist: bot.NearestPickupDist,
		Neighbors:  bot.NeighborCount,
		Carrying:   bot.CarryingPkg >= 0,
	}

	if len(*exps) >= ds.MaxBuffer {
		// Replace oldest experience
		copy((*exps)[0:], (*exps)[1:])
		(*exps)[len(*exps)-1] = exp
	} else {
		*exps = append(*exps, exp)
	}
}

// replayExperiences processes a batch of stored experiences to adjust behavior.
func replayExperiences(ss *SwarmState, bot *SwarmBot, exps []DreamExperience, ds *DreamState) float64 {
	if len(exps) == 0 {
		return 0
	}

	totalValue := 0.0
	batchCount := ds.BatchSize
	if batchCount > len(exps) {
		batchCount = len(exps)
	}

	// Sample random experiences from buffer
	for b := 0; b < batchCount; b++ {
		idx := ss.Rng.Intn(len(exps))
		exp := exps[idx]
		totalValue += exp.Reward

		// If this was a positive experience, bias toward that location
		if exp.Reward > 0.2 {
			dx := exp.X - bot.X
			dy := exp.Y - bot.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 20 {
				targetAngle := math.Atan2(dy, dx)
				diff := targetAngle - bot.Angle
				diff = WrapAngle(diff)
				bot.Angle += diff * ds.LearnRate * exp.Reward
			}
		}

		// If negative experience, bias away from that location
		if exp.Reward < -0.1 {
			dx := exp.X - bot.X
			dy := exp.Y - bot.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 150 && dist > 5 {
				awayAngle := math.Atan2(-dy, -dx)
				diff := awayAngle - bot.Angle
				diff = WrapAngle(diff)
				bot.Angle += diff * ds.LearnRate * math.Abs(exp.Reward)
			}
		}
	}

	return totalValue / float64(batchCount)
}

// DreamCount returns total dreams across all bots.
func DreamCount(ds *DreamState) int {
	if ds == nil {
		return 0
	}
	return ds.TotalDreams
}

// ExperienceCount returns total stored experiences.
func ExperienceCount(ds *DreamState) int {
	if ds == nil {
		return 0
	}
	return ds.TotalExperiences
}

// BotExperienceCount returns how many experiences a bot has stored.
func BotExperienceCount(ds *DreamState, botIdx int) int {
	if ds == nil || botIdx < 0 || botIdx >= len(ds.Experiences) {
		return 0
	}
	return len(ds.Experiences[botIdx])
}

// IsDreamPhase returns whether the swarm is in a global dream phase.
func IsDreamPhase(ds *DreamState) bool {
	if ds == nil {
		return false
	}
	return ds.DreamPhase
}
