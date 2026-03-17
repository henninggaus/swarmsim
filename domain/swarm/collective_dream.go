package swarm

import (
	"math"
	"swarmsim/logger"
)

// CollectiveDreamState manages offline strategy replay and recombination.
// When swarm activity drops (low average speed), the system enters a
// "dream phase" where it replays the best strategies from all bots,
// recombines them, and injects creative new solutions — like REM sleep
// for the collective consciousness.
type CollectiveDreamState struct {
	Memories     []StrategyMemory // collected strategy snapshots
	MaxMemories  int              // max stored memories (default 100)
	DreamActive  bool             // currently in dream phase
	DreamTicks   int              // ticks spent dreaming
	DreamCooldown int             // ticks until next dream allowed

	// Dream parameters
	ActivityThreshold float64 // below this avg speed ratio → dream (default 0.3)
	DreamDuration     int     // ticks per dream phase (default 50)
	RecombineRate     float64 // chance to recombine two memories (default 0.2)

	// Stats
	TotalDreams     int
	MemoriesStored  int
	InsightsCreated int // new strategies born from dreams
	AvgMemoryFitness float64
}

// StrategyMemory is a snapshot of a successful bot's behavior.
type StrategyMemory struct {
	Angle       float64 // heading when successful
	Speed       float64
	NearPickup  float64 // distance to nearest pickup
	NearDropoff float64
	Neighbors   int
	Carrying    bool
	Fitness     float64 // how successful this strategy was
	BotIdx      int     // which bot this came from
	Tick        int     // when this was recorded
}

// InitCollectiveDream sets up the collective dream system.
func InitCollectiveDream(ss *SwarmState) {
	cd := &CollectiveDreamState{
		Memories:          make([]StrategyMemory, 0, 100),
		MaxMemories:       100,
		ActivityThreshold: 0.3,
		DreamDuration:     50,
		RecombineRate:     0.2,
	}

	ss.CollectiveDream = cd
	logger.Info("DREAM", "Kollektives Unterbewusstsein initialisiert")
}

// ClearCollectiveDream disables the dream system.
func ClearCollectiveDream(ss *SwarmState) {
	ss.CollectiveDream = nil
	ss.CollectiveDreamOn = false
}

// TickCollectiveDream runs one tick of the dream system.
func TickCollectiveDream(ss *SwarmState) {
	cd := ss.CollectiveDream
	if cd == nil {
		return
	}

	// Collect memories from successful bots
	collectMemories(ss, cd)

	// Check if we should enter dream phase
	if cd.DreamCooldown > 0 {
		cd.DreamCooldown--
	}

	if !cd.DreamActive {
		avgActivity := computeSwarmActivity(ss)
		if avgActivity < cd.ActivityThreshold && cd.DreamCooldown <= 0 && len(cd.Memories) >= 5 {
			cd.DreamActive = true
			cd.DreamTicks = 0
			cd.TotalDreams++
			logger.Info("DREAM", "Traumphase %d beginnt mit %d Erinnerungen",
				cd.TotalDreams, len(cd.Memories))
		}
	}

	if cd.DreamActive {
		executeDream(ss, cd)
		cd.DreamTicks++
		if cd.DreamTicks >= cd.DreamDuration {
			cd.DreamActive = false
			cd.DreamCooldown = 200 // cooldown before next dream
		}
	}

	// Update stats
	cd.MemoriesStored = len(cd.Memories)
	if len(cd.Memories) > 0 {
		sum := 0.0
		for _, m := range cd.Memories {
			sum += m.Fitness
		}
		cd.AvgMemoryFitness = sum / float64(len(cd.Memories))
	}
}

// collectMemories records snapshots from bots doing well.
func collectMemories(ss *SwarmState, cd *CollectiveDreamState) {
	if ss.Tick%20 != 0 {
		return
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Record if bot is being productive
		fitness := 0.0
		if bot.CarryingPkg >= 0 && bot.NearestDropoffDist < 100 {
			fitness = 0.8
		} else if bot.CarryingPkg < 0 && bot.NearestPickupDist < 60 {
			fitness = 0.5
		} else if bot.Speed > SwarmBotSpeed*0.8 {
			fitness = 0.2
		}

		if fitness <= 0.1 {
			continue
		}

		mem := StrategyMemory{
			Angle:       bot.Angle,
			Speed:       bot.Speed,
			NearPickup:  bot.NearestPickupDist,
			NearDropoff: bot.NearestDropoffDist,
			Neighbors:   bot.NeighborCount,
			Carrying:    bot.CarryingPkg >= 0,
			Fitness:     fitness,
			BotIdx:      i,
			Tick:        ss.Tick,
		}

		if len(cd.Memories) < cd.MaxMemories {
			cd.Memories = append(cd.Memories, mem)
		} else {
			// Replace weakest memory
			weakest := 0
			for j := 1; j < len(cd.Memories); j++ {
				if cd.Memories[j].Fitness < cd.Memories[weakest].Fitness {
					weakest = j
				}
			}
			if mem.Fitness > cd.Memories[weakest].Fitness {
				cd.Memories[weakest] = mem
			}
		}
	}
}

// computeSwarmActivity returns average speed ratio.
func computeSwarmActivity(ss *SwarmState) float64 {
	if len(ss.Bots) == 0 {
		return 0
	}
	sum := 0.0
	for i := range ss.Bots {
		sum += ss.Bots[i].Speed / SwarmBotSpeed
	}
	return sum / float64(len(ss.Bots))
}

// executeDream applies dream-recombined strategies to idle bots.
func executeDream(ss *SwarmState, cd *CollectiveDreamState) {
	if len(cd.Memories) < 2 {
		return
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Only dream-inject into slow/idle bots
		if bot.Speed > SwarmBotSpeed*0.5 {
			continue
		}

		if ss.Rng.Float64() < cd.RecombineRate {
			// Recombine two random high-fitness memories
			m1 := selectMemory(ss, cd)
			m2 := selectMemory(ss, cd)

			// Create "insight": blend of two strategies
			blend := ss.Rng.Float64()
			newAngle := m1.Angle*blend + m2.Angle*(1-blend)
			newSpeed := m1.Speed*blend + m2.Speed*(1-blend)

			bot.Angle = newAngle + (ss.Rng.Float64()-0.5)*0.2
			bot.Speed = newSpeed

			cd.InsightsCreated++

			// Dream visual: purple glow
			bot.LEDColor = [3]uint8{180, 50, 220}
		} else {
			// Replay single memory
			m := selectMemory(ss, cd)

			// Apply similar context-dependent behavior
			if m.Carrying == (bot.CarryingPkg >= 0) {
				bot.Angle = m.Angle + (ss.Rng.Float64()-0.5)*0.3
				bot.Speed = m.Speed * 0.8
			}

			// Dream visual: soft blue glow
			bot.LEDColor = [3]uint8{80, 80, 200}
		}
	}
}

// selectMemory picks a memory weighted by fitness.
func selectMemory(ss *SwarmState, cd *CollectiveDreamState) StrategyMemory {
	// Fitness-proportional selection
	totalFitness := 0.0
	for _, m := range cd.Memories {
		totalFitness += m.Fitness
	}

	if totalFitness <= 0 {
		return cd.Memories[ss.Rng.Intn(len(cd.Memories))]
	}

	r := ss.Rng.Float64() * totalFitness
	cumul := 0.0
	for _, m := range cd.Memories {
		cumul += m.Fitness
		if cumul >= r {
			return m
		}
	}
	return cd.Memories[len(cd.Memories)-1]
}

// DreamIsActive returns whether the swarm is currently dreaming.
func DreamIsActive(cd *CollectiveDreamState) bool {
	if cd == nil {
		return false
	}
	return cd.DreamActive
}

// DreamInsights returns the total number of creative insights generated.
func DreamInsights(cd *CollectiveDreamState) int {
	if cd == nil {
		return 0
	}
	return cd.InsightsCreated
}

// DreamMemoryCount returns stored memory count.
func DreamMemoryCount(cd *CollectiveDreamState) int {
	if cd == nil {
		return 0
	}
	return cd.MemoriesStored
}

// DreamAvgFitness returns the average fitness of stored memories.
func DreamAvgFitness(cd *CollectiveDreamState) float64 {
	if cd == nil {
		return 0
	}
	return math.Max(cd.AvgMemoryFitness, 0)
}
