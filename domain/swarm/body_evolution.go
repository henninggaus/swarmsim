package swarm

import (
	"math"
	"swarmsim/logger"
)

// BodyEvoState manages morphological evolution of bot bodies.
// Each bot has evolvable physical traits: size, sensor range, speed,
// carrying capacity, and energy efficiency. Larger bots are slower but
// carry more; smaller bots are fast scouts. Trade-offs enforce diversity.
type BodyEvoState struct {
	Bodies []BotBody // per-bot body plans

	// Population stats
	AvgSize       float64
	AvgSpeed      float64
	AvgSensorRange float64
	SizeDiversity float64 // std dev of sizes
	Generation    int
}

// BotBody defines the physical traits of a bot.
type BotBody struct {
	Size          float64 // body size multiplier (0.5-2.0, default 1.0)
	MaxSpeed      float64 // derived max speed (smaller = faster)
	SensorRange   float64 // derived sensor range (bigger = better sensors)
	CarryCapacity int     // how many packages can carry (1-3)
	Efficiency    float64 // energy efficiency (0.5-1.5)
	Armor         float64 // damage resistance (0-1)

	// Genome: raw evolvable parameters
	Genes [6]float64
}

// InitBodyEvolution sets up the morphological evolution system.
func InitBodyEvolution(ss *SwarmState) {
	n := len(ss.Bots)
	be := &BodyEvoState{
		Bodies: make([]BotBody, n),
	}

	for i := 0; i < n; i++ {
		for g := 0; g < 6; g++ {
			be.Bodies[i].Genes[g] = ss.Rng.Float64()
		}
		expressBody(&be.Bodies[i])
	}

	ss.BodyEvo = be
	logger.Info("BODY", "Initialisiert: %d Bots mit evolvierbaren Koerper-Plaenen", n)
}

// ClearBodyEvolution disables the body evolution system.
func ClearBodyEvolution(ss *SwarmState) {
	ss.BodyEvo = nil
	ss.BodyEvoOn = false
}

// expressBody computes physical traits from genes.
func expressBody(body *BotBody) {
	// Gene 0: size (0.5 - 2.0)
	body.Size = 0.5 + body.Genes[0]*1.5

	// Trade-offs: bigger = slower but better sensors and carry capacity
	body.MaxSpeed = SwarmBotSpeed * (1.5 - body.Size*0.4)
	if body.MaxSpeed < SwarmBotSpeed*0.3 {
		body.MaxSpeed = SwarmBotSpeed * 0.3
	}

	body.SensorRange = SwarmSensorRange * (0.6 + body.Size*0.5)

	// Gene 1: carry capacity
	body.CarryCapacity = 1
	if body.Genes[1] > 0.7 {
		body.CarryCapacity = 2
	}
	if body.Genes[1] > 0.9 && body.Size > 1.3 {
		body.CarryCapacity = 3
	}

	// Gene 2: efficiency
	body.Efficiency = 0.5 + body.Genes[2]

	// Gene 3: armor (trade-off with speed)
	body.Armor = body.Genes[3] * 0.8
	body.MaxSpeed *= 1.0 - body.Armor*0.2

	// Gene 4-5 reserved for future traits
}

// TickBodyEvolution applies body traits to bot behavior.
func TickBodyEvolution(ss *SwarmState) {
	be := ss.BodyEvo
	if be == nil {
		return
	}

	n := len(ss.Bots)
	if len(be.Bodies) != n {
		return
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		body := &be.Bodies[i]

		// Apply max speed
		if bot.Speed > body.MaxSpeed {
			bot.Speed = body.MaxSpeed
		}

		// Visual: size affects LED brightness, color shows type
		sizeRatio := (body.Size - 0.5) / 1.5 // 0-1
		if sizeRatio > 1 {
			sizeRatio = 1
		}

		// Big bots: warm orange, small bots: cool blue
		r := uint8(100 + sizeRatio*155)
		g := uint8(100)
		b := uint8(100 + (1-sizeRatio)*155)
		bot.LEDColor = [3]uint8{r, g, b}
	}
}

// EvolveBodyPlans evolves the body plans based on fitness.
func EvolveBodyPlans(ss *SwarmState, sortedIndices []int) {
	be := ss.BodyEvo
	if be == nil {
		return
	}

	n := len(ss.Bots)
	if len(be.Bodies) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 25 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 2

	parents := make([]BotBody, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parents[i] = be.Bodies[sortedIndices[i]]
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			continue
		}

		// Crossover two parents
		p1 := ss.Rng.Intn(parentCount)
		p2 := ss.Rng.Intn(parentCount)
		for p2 == p1 && parentCount > 1 {
			p2 = ss.Rng.Intn(parentCount)
		}

		child := BotBody{}
		crossPoint := ss.Rng.Intn(6)
		for g := 0; g < 6; g++ {
			if g < crossPoint {
				child.Genes[g] = parents[p1].Genes[g]
			} else {
				child.Genes[g] = parents[p2].Genes[g]
			}

			// Mutation
			if ss.Rng.Float64() < 0.15 {
				child.Genes[g] += ss.Rng.NormFloat64() * 0.1
				child.Genes[g] = clampF(child.Genes[g], 0, 1)
			}
		}

		expressBody(&child)
		be.Bodies[botIdx] = child
	}

	updateBodyStats(be)
	be.Generation++

	logger.Info("BODY", "Gen %d: AvgSize=%.2f, AvgSpeed=%.2f, SizeDiversity=%.3f",
		be.Generation, be.AvgSize, be.AvgSpeed, be.SizeDiversity)
}

// updateBodyStats computes population statistics.
func updateBodyStats(be *BodyEvoState) {
	n := len(be.Bodies)
	if n == 0 {
		return
	}

	sumSize, sumSpeed, sumSensor := 0.0, 0.0, 0.0
	for _, b := range be.Bodies {
		sumSize += b.Size
		sumSpeed += b.MaxSpeed
		sumSensor += b.SensorRange
	}

	fn := float64(n)
	be.AvgSize = sumSize / fn
	be.AvgSpeed = sumSpeed / fn
	be.AvgSensorRange = sumSensor / fn

	// Size diversity
	varSize := 0.0
	for _, b := range be.Bodies {
		d := b.Size - be.AvgSize
		varSize += d * d
	}
	be.SizeDiversity = math.Sqrt(varSize / fn)
}

// BodySize returns a bot's body size.
func BodySize(be *BodyEvoState, botIdx int) float64 {
	if be == nil || botIdx < 0 || botIdx >= len(be.Bodies) {
		return 1.0
	}
	return be.Bodies[botIdx].Size
}

// BodyMaxSpeed returns a bot's max speed.
func BodyMaxSpeed(be *BodyEvoState, botIdx int) float64 {
	if be == nil || botIdx < 0 || botIdx >= len(be.Bodies) {
		return SwarmBotSpeed
	}
	return be.Bodies[botIdx].MaxSpeed
}

// AvgBodySize returns the population average size.
func AvgBodySize(be *BodyEvoState) float64 {
	if be == nil {
		return 0
	}
	return be.AvgSize
}
