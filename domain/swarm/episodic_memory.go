package swarm

import (
	"math"
	"swarmsim/logger"
)

// EpisodicMemoryState manages a per-bot episodic memory system.
// Bots store memories of significant events (finding resources, successful
// deliveries, encountering danger). They replay recent memories to bias
// behavior — e.g., returning to a location where they found a resource.
type EpisodicMemoryState struct {
	MaxMemories   int     // max memories per bot (default 20)
	DecayRate     float64 // memory strength decay per tick (default 0.001)
	ReplayChance  float64 // probability of replaying a memory per tick (default 0.05)
	SpatialRadius float64 // how close to trigger spatial memory (default 60)

	Memories [][]Episode // per-bot memory stores

	// Stats
	TotalMemories int
	AvgMemoryAge  float64
	ReplayCount   int
}

// Episode is a single remembered event.
type Episode struct {
	Tick     int     // when it happened
	X, Y     float64 // where it happened
	Type     EpisodeType
	Value    float64 // importance/reward (higher = more significant)
	Strength float64 // memory strength (decays over time, 0-1)
}

// EpisodeType categorizes what happened.
type EpisodeType int

const (
	EpisodeFoundResource EpisodeType = iota // found a pickup location
	EpisodeDelivered                         // successfully delivered
	EpisodeDanger                            // encountered obstacle/predator
	EpisodeGoodArea                          // area with many resources
	EpisodeBadArea                           // area with no resources
	EpisodeSocialEvent                       // encountered many bots
)

// EpisodeTypeName returns the display name.
func EpisodeTypeName(et EpisodeType) string {
	switch et {
	case EpisodeFoundResource:
		return "Ressource"
	case EpisodeDelivered:
		return "Lieferung"
	case EpisodeDanger:
		return "Gefahr"
	case EpisodeGoodArea:
		return "Gutes Gebiet"
	case EpisodeBadArea:
		return "Schlechtes Gebiet"
	case EpisodeSocialEvent:
		return "Sozialevent"
	default:
		return "?"
	}
}

// InitEpisodicMemory sets up the episodic memory system.
func InitEpisodicMemory(ss *SwarmState) {
	n := len(ss.Bots)
	em := &EpisodicMemoryState{
		MaxMemories:   20,
		DecayRate:     0.001,
		ReplayChance:  0.05,
		SpatialRadius: 60,
		Memories:      make([][]Episode, n),
	}

	for i := range em.Memories {
		em.Memories[i] = make([]Episode, 0, em.MaxMemories)
	}

	ss.EpisodicMemory = em
	logger.Info("MEMORY", "Initialisiert: %d Bots, MaxErinnerungen=%d", n, em.MaxMemories)
}

// ClearEpisodicMemory disables the episodic memory system.
func ClearEpisodicMemory(ss *SwarmState) {
	ss.EpisodicMemory = nil
	ss.EpisodicMemoryOn = false
}

// TickEpisodicMemory runs one tick of the memory system.
func TickEpisodicMemory(ss *SwarmState) {
	em := ss.EpisodicMemory
	if em == nil {
		return
	}

	n := len(ss.Bots)
	if len(em.Memories) != n {
		return
	}

	totalMem := 0
	totalAge := 0.0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		mems := &em.Memories[i]

		// Record new memories based on current events
		recordMemories(ss, em, i, bot, mems)

		// Decay existing memories
		alive := 0
		for j := range *mems {
			(*mems)[j].Strength -= em.DecayRate
			if (*mems)[j].Strength > 0 {
				alive++
				totalAge += float64(ss.Tick - (*mems)[j].Tick)
			}
		}

		// Remove dead memories
		if alive < len(*mems) {
			filtered := make([]Episode, 0, alive)
			for _, ep := range *mems {
				if ep.Strength > 0 {
					filtered = append(filtered, ep)
				}
			}
			*mems = filtered
		}

		totalMem += len(*mems)

		// Memory replay: occasionally bias behavior based on memories
		if ss.Rng.Float64() < em.ReplayChance && len(*mems) > 0 {
			replayMemory(ss, em, bot, *mems)
			em.ReplayCount++
		}

		// Spatial memory trigger: react when near a remembered location
		applySpatialMemory(ss, em, bot, *mems)
	}

	em.TotalMemories = totalMem
	if totalMem > 0 {
		em.AvgMemoryAge = totalAge / float64(totalMem)
	}
}

// recordMemories checks for memorable events and stores them.
func recordMemories(ss *SwarmState, em *EpisodicMemoryState, botIdx int, bot *SwarmBot, mems *[]Episode) {
	// Found resource nearby
	if bot.NearestPickupDist < 40 && bot.CarryingPkg < 0 {
		addMemory(em, mems, Episode{
			Tick:     ss.Tick,
			X:        bot.X,
			Y:        bot.Y,
			Type:     EpisodeFoundResource,
			Value:    1.0,
			Strength: 1.0,
		})
	}

	// Successful delivery
	if bot.Stats.TotalDeliveries > 0 && ss.Tick > 0 {
		rate := float64(bot.Stats.TotalDeliveries) / float64(ss.Tick) * 1000
		if rate > 0.5 && ss.Tick%200 == botIdx%200 {
			addMemory(em, mems, Episode{
				Tick:     ss.Tick,
				X:        bot.X,
				Y:        bot.Y,
				Type:     EpisodeDelivered,
				Value:    rate,
				Strength: 1.0,
			})
		}
	}

	// Social event: many neighbors
	if bot.NeighborCount > 6 && ss.Tick%50 == botIdx%50 {
		addMemory(em, mems, Episode{
			Tick:     ss.Tick,
			X:        bot.X,
			Y:        bot.Y,
			Type:     EpisodeSocialEvent,
			Value:    float64(bot.NeighborCount) / 10.0,
			Strength: 0.8,
		})
	}

	// Bad area: wandering with no pickups nearby for a while
	if bot.NearestPickupDist > 300 && bot.CarryingPkg < 0 && ss.Tick%100 == botIdx%100 {
		addMemory(em, mems, Episode{
			Tick:     ss.Tick,
			X:        bot.X,
			Y:        bot.Y,
			Type:     EpisodeBadArea,
			Value:    -0.5,
			Strength: 0.6,
		})
	}
}

// addMemory adds a memory, evicting the weakest if at capacity.
func addMemory(em *EpisodicMemoryState, mems *[]Episode, ep Episode) {
	if len(*mems) >= em.MaxMemories {
		// Evict weakest
		weakest := 0
		for j := 1; j < len(*mems); j++ {
			if (*mems)[j].Strength < (*mems)[weakest].Strength {
				weakest = j
			}
		}
		(*mems)[weakest] = ep
	} else {
		*mems = append(*mems, ep)
	}
}

// replayMemory picks a random memory and biases the bot toward/away from it.
func replayMemory(ss *SwarmState, em *EpisodicMemoryState, bot *SwarmBot, mems []Episode) {
	// Pick memory weighted by strength
	totalStr := 0.0
	for _, ep := range mems {
		totalStr += ep.Strength
	}
	if totalStr <= 0 {
		return
	}

	r := ss.Rng.Float64() * totalStr
	cumul := 0.0
	var chosen *Episode
	for j := range mems {
		cumul += mems[j].Strength
		if cumul >= r {
			chosen = &mems[j]
			break
		}
	}
	if chosen == nil {
		return
	}

	// Bias toward good memories, away from bad
	dx := chosen.X - bot.X
	dy := chosen.Y - bot.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 5 {
		return
	}

	targetAngle := math.Atan2(dy, dx)
	if chosen.Value < 0 {
		targetAngle += math.Pi // go away from bad memories
	}

	diff := targetAngle - bot.Angle
	diff = WrapAngle(diff)
	bot.Angle += diff * chosen.Strength * 0.1
}

// applySpatialMemory triggers when a bot is near a remembered location.
func applySpatialMemory(ss *SwarmState, em *EpisodicMemoryState, bot *SwarmBot, mems []Episode) {
	radiusSq := em.SpatialRadius * em.SpatialRadius

	for _, ep := range mems {
		dx := bot.X - ep.X
		dy := bot.Y - ep.Y
		if dx*dx+dy*dy > radiusSq {
			continue
		}

		switch ep.Type {
		case EpisodeFoundResource:
			// Slow down to search area
			if bot.CarryingPkg < 0 {
				bot.Speed *= 0.9
			}
		case EpisodeDanger:
			// Speed up to leave
			bot.Speed = math.Min(bot.Speed*1.2, SwarmBotSpeed)
		case EpisodeGoodArea:
			// LED green pulse
			green := uint8(128 + ep.Strength*127)
			bot.LEDColor = [3]uint8{0, green, 50}
		}
	}
}

// MemoryCount returns total memories across all bots.
func MemoryCount(em *EpisodicMemoryState) int {
	if em == nil {
		return 0
	}
	return em.TotalMemories
}

// BotMemoryCount returns how many memories a specific bot has.
func BotMemoryCount(em *EpisodicMemoryState, botIdx int) int {
	if em == nil || botIdx < 0 || botIdx >= len(em.Memories) {
		return 0
	}
	return len(em.Memories[botIdx])
}

// StrongestMemory returns the strongest memory of a bot.
func StrongestMemory(em *EpisodicMemoryState, botIdx int) *Episode {
	if em == nil || botIdx < 0 || botIdx >= len(em.Memories) {
		return nil
	}
	mems := em.Memories[botIdx]
	if len(mems) == 0 {
		return nil
	}
	best := &mems[0]
	for i := 1; i < len(mems); i++ {
		if mems[i].Strength > best.Strength {
			best = &mems[i]
		}
	}
	return best
}
