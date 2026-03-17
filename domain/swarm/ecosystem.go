package swarm

import (
	"math"
	"swarmsim/logger"
)

// EcosystemState manages a co-evolutionary ecosystem.
// Plants grow and spread, herbivore-bots eat plants for energy,
// predator-bots hunt herbivores. Population dynamics emerge naturally
// (Lotka-Volterra-like cycles).
type EcosystemState struct {
	// Plants
	Plants    []Plant
	MaxPlants int     // cap on plant population (default 100)
	GrowRate  float64 // plant growth probability per tick (default 0.02)
	SpreadDist float64 // plant seed spread distance (default 50)

	// Species assignment
	Herbivores []int // bot indices that are herbivores
	Predators  []int // bot indices that are predators

	// Parameters
	EatRange      float64 // range to eat a plant (default 20)
	HuntRange     float64 // range for predator to catch prey (default 25)
	PlantEnergy   float64 // energy gained from eating a plant (default 20)
	HuntEnergy    float64 // energy gained from catching prey (default 30)
	MetabolicCost float64 // energy cost per tick (default 0.05)
	ReproThreshold float64 // energy needed to reproduce (default 80)

	// Per-bot ecosystem state
	EcoEnergy []float64 // per-bot energy level
	EcoAlive  []bool    // per-bot alive status

	// Stats
	PlantCount    int
	HerbivoreCount int
	PredatorCount int
	Tick          int
	PopHistory    []PopSnapshot // population over time
}

// Plant is a food source in the ecosystem.
type Plant struct {
	X, Y    float64
	Energy  float64 // energy content
	Age     int
	Alive   bool
}

// PopSnapshot records population counts at a point in time.
type PopSnapshot struct {
	Tick       int
	Plants     int
	Herbivores int
	Predators  int
}

// InitEcosystem sets up the ecosystem.
func InitEcosystem(ss *SwarmState, herbRatio float64) {
	if herbRatio < 0.3 {
		herbRatio = 0.3
	}
	if herbRatio > 0.9 {
		herbRatio = 0.9
	}

	n := len(ss.Bots)
	herbCount := int(float64(n) * herbRatio)
	predCount := n - herbCount

	eco := &EcosystemState{
		MaxPlants:      100,
		GrowRate:       0.02,
		SpreadDist:     50,
		EatRange:       20,
		HuntRange:      25,
		PlantEnergy:    20,
		HuntEnergy:     30,
		MetabolicCost:  0.05,
		ReproThreshold: 80,
		EcoEnergy:      make([]float64, n),
		EcoAlive:       make([]bool, n),
	}

	// Assign species
	for i := 0; i < n; i++ {
		eco.EcoEnergy[i] = 50
		eco.EcoAlive[i] = true
		if i < herbCount {
			eco.Herbivores = append(eco.Herbivores, i)
		} else {
			eco.Predators = append(eco.Predators, i)
		}
	}

	// Spawn initial plants
	for i := 0; i < 50; i++ {
		eco.Plants = append(eco.Plants, Plant{
			X:      ss.Rng.Float64() * ss.ArenaW,
			Y:      ss.Rng.Float64() * ss.ArenaH,
			Energy: 15 + ss.Rng.Float64()*10,
			Alive:  true,
		})
	}

	eco.HerbivoreCount = herbCount
	eco.PredatorCount = predCount
	eco.PlantCount = len(eco.Plants)

	ss.Ecosystem = eco
	logger.Info("ECO", "Initialisiert: %d Herbivoren, %d Raeuber, %d Pflanzen",
		herbCount, predCount, len(eco.Plants))
}

// ClearEcosystem disables the ecosystem.
func ClearEcosystem(ss *SwarmState) {
	ss.Ecosystem = nil
	ss.EcosystemOn = false
}

// TickEcosystem runs one tick of the ecosystem.
func TickEcosystem(ss *SwarmState) {
	eco := ss.Ecosystem
	if eco == nil {
		return
	}

	n := len(ss.Bots)
	if len(eco.EcoEnergy) != n {
		return
	}

	// Phase 1: Plant growth and spreading
	if ss.Rng.Float64() < eco.GrowRate && len(eco.Plants) < eco.MaxPlants {
		// New plant near existing plant (seed spread)
		if len(eco.Plants) > 0 {
			parent := eco.Plants[ss.Rng.Intn(len(eco.Plants))]
			if parent.Alive {
				nx := parent.X + (ss.Rng.Float64()-0.5)*eco.SpreadDist*2
				ny := parent.Y + (ss.Rng.Float64()-0.5)*eco.SpreadDist*2
				nx = math.Max(0, math.Min(nx, ss.ArenaW))
				ny = math.Max(0, math.Min(ny, ss.ArenaH))
				eco.Plants = append(eco.Plants, Plant{
					X: nx, Y: ny,
					Energy: 10 + ss.Rng.Float64()*10,
					Alive:  true,
				})
			}
		}
	}

	// Age plants
	for i := range eco.Plants {
		if eco.Plants[i].Alive {
			eco.Plants[i].Age++
			// Old plants die
			if eco.Plants[i].Age > 2000 {
				eco.Plants[i].Alive = false
			}
		}
	}

	// Phase 2: Herbivores eat plants
	eatRangeSq := eco.EatRange * eco.EatRange
	for _, hi := range eco.Herbivores {
		if hi >= n || !eco.EcoAlive[hi] {
			continue
		}
		bot := &ss.Bots[hi]

		// Find nearest alive plant
		for j := range eco.Plants {
			if !eco.Plants[j].Alive {
				continue
			}
			dx := bot.X - eco.Plants[j].X
			dy := bot.Y - eco.Plants[j].Y
			if dx*dx+dy*dy < eatRangeSq {
				eco.EcoEnergy[hi] += eco.Plants[j].Energy
				eco.Plants[j].Alive = false
				break // one plant per tick
			}
		}

		// Metabolic cost
		eco.EcoEnergy[hi] -= eco.MetabolicCost
		if eco.EcoEnergy[hi] <= 0 {
			eco.EcoAlive[hi] = false
			bot.Speed = 0
		}

		// Color: green for herbivores
		green := uint8(math.Min(eco.EcoEnergy[hi]*3, 255))
		bot.LEDColor = [3]uint8{0, green, 50}
	}

	// Phase 3: Predators hunt herbivores
	huntRangeSq := eco.HuntRange * eco.HuntRange
	for _, pi := range eco.Predators {
		if pi >= n || !eco.EcoAlive[pi] {
			continue
		}
		predBot := &ss.Bots[pi]

		// Find nearest alive herbivore
		bestDist := huntRangeSq
		bestH := -1
		for _, hi := range eco.Herbivores {
			if hi >= n || !eco.EcoAlive[hi] {
				continue
			}
			dx := predBot.X - ss.Bots[hi].X
			dy := predBot.Y - ss.Bots[hi].Y
			dSq := dx*dx + dy*dy
			if dSq < bestDist {
				bestDist = dSq
				bestH = hi
			}
		}

		if bestH >= 0 {
			eco.EcoEnergy[pi] += eco.HuntEnergy
			eco.EcoAlive[bestH] = false
			ss.Bots[bestH].Speed = 0
		}

		// Metabolic cost (predators burn more)
		eco.EcoEnergy[pi] -= eco.MetabolicCost * 1.5
		if eco.EcoEnergy[pi] <= 0 {
			eco.EcoAlive[pi] = false
			predBot.Speed = 0
		}

		// Color: red for predators
		red := uint8(math.Min(eco.EcoEnergy[pi]*3, 255))
		predBot.LEDColor = [3]uint8{red, 0, 50}
	}

	// Phase 4: Respawn dead organisms (simplified reproduction)
	if eco.Tick%100 == 0 {
		respawnEcosystem(ss, eco)
	}

	// Phase 5: Clean dead plants
	alivePlants := 0
	for _, p := range eco.Plants {
		if p.Alive {
			alivePlants++
		}
	}
	if alivePlants < len(eco.Plants)/2 {
		filtered := make([]Plant, 0, alivePlants)
		for _, p := range eco.Plants {
			if p.Alive {
				filtered = append(filtered, p)
			}
		}
		eco.Plants = filtered
	}

	// Update stats
	eco.PlantCount = alivePlants
	eco.HerbivoreCount = 0
	eco.PredatorCount = 0
	for _, hi := range eco.Herbivores {
		if hi < n && eco.EcoAlive[hi] {
			eco.HerbivoreCount++
		}
	}
	for _, pi := range eco.Predators {
		if pi < n && eco.EcoAlive[pi] {
			eco.PredatorCount++
		}
	}

	// Record history
	if eco.Tick%50 == 0 {
		eco.PopHistory = append(eco.PopHistory, PopSnapshot{
			Tick:       eco.Tick,
			Plants:     eco.PlantCount,
			Herbivores: eco.HerbivoreCount,
			Predators:  eco.PredatorCount,
		})
		if len(eco.PopHistory) > 200 {
			eco.PopHistory = eco.PopHistory[1:]
		}
	}

	eco.Tick++
}

// respawnEcosystem respawns dead bots with some energy.
func respawnEcosystem(ss *SwarmState, eco *EcosystemState) {
	n := len(ss.Bots)
	for i := 0; i < n; i++ {
		if !eco.EcoAlive[i] {
			// 30% chance to respawn
			if ss.Rng.Float64() < 0.3 {
				eco.EcoAlive[i] = true
				eco.EcoEnergy[i] = 30
				ss.Bots[i].X = ss.Rng.Float64() * ss.ArenaW
				ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
				ss.Bots[i].Speed = SwarmBotSpeed * 0.5
			}
		}
	}
}

// EcoPlantCount returns the number of alive plants.
func EcoPlantCount(eco *EcosystemState) int {
	if eco == nil {
		return 0
	}
	return eco.PlantCount
}

// EcoSpeciesCount returns herbivore and predator counts.
func EcoSpeciesCount(eco *EcosystemState) (int, int) {
	if eco == nil {
		return 0, 0
	}
	return eco.HerbivoreCount, eco.PredatorCount
}

// EcoBotEnergy returns a bot's ecosystem energy.
func EcoBotEnergy(eco *EcosystemState, botIdx int) float64 {
	if eco == nil || botIdx < 0 || botIdx >= len(eco.EcoEnergy) {
		return 0
	}
	return eco.EcoEnergy[botIdx]
}
