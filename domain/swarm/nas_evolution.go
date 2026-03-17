package swarm

import (
	"math"
	"swarmsim/logger"
)

// NASEvoState integrates NAS genomes into the swarm, allowing each bot
// to evolve its own neural network topology. Networks grow in complexity
// as needed — starting minimal and adding nodes/connections over time.
type NASEvoState struct {
	Genomes []*NASGenome // per-bot variable-topology networks
	NAS     *NASState    // shared NAS parameters

	// Population stats
	AvgNodes       float64
	AvgConnections float64
	AvgComplexity  float64
	BestFitness    float64
	Generation     int
}

// InitNASEvolution sets up the NAS evolution system.
func InitNASEvolution(ss *SwarmState) {
	n := len(ss.Bots)
	nas := NewNASState()
	evo := &NASEvoState{
		Genomes: make([]*NASGenome, n),
		NAS:     nas,
	}

	numInputs := 6
	numOutputs := 4

	for i := 0; i < n; i++ {
		evo.Genomes[i] = NewMinimalNASGenome(ss.Rng, numInputs, numOutputs, nas)
	}

	ss.NASEvo = evo
	logger.Info("NAS-EVO", "Initialisiert: %d Bots mit variablen Netzwerk-Topologien", n)
}

// ClearNASEvolution disables the NAS evolution system.
func ClearNASEvolution(ss *SwarmState) {
	ss.NASEvo = nil
	ss.NASEvoOn = false
}

// TickNASEvolution runs the NAS networks to control bot behavior.
func TickNASEvolution(ss *SwarmState) {
	evo := ss.NASEvo
	if evo == nil {
		return
	}

	n := len(ss.Bots)
	if len(evo.Genomes) != n {
		return
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		genome := evo.Genomes[i]
		if genome == nil {
			continue
		}

		// Build inputs
		inputs := []float64{
			math.Tanh(bot.NearestPickupDist / 200.0),
			math.Tanh(float64(bot.NeighborCount) / 10.0),
			math.Sin(bot.Angle),
			math.Cos(bot.Angle),
			nasBoolToFloat(bot.CarryingPkg >= 0),
			math.Tanh(bot.NearestDropoffDist / 200.0),
		}

		outputs := NASForward(genome, inputs)
		if len(outputs) < 4 {
			continue
		}

		bestAction := 0
		bestVal := outputs[0]
		for a := 1; a < 4; a++ {
			if outputs[a] > bestVal {
				bestVal = outputs[a]
				bestAction = a
			}
		}

		switch bestAction {
		case 0: // forward
			bot.Speed = SwarmBotSpeed
		case 1: // left
			bot.Speed = SwarmBotSpeed
			bot.Angle -= 0.15
		case 2: // right
			bot.Speed = SwarmBotSpeed
			bot.Angle += 0.15
		case 3: // idle/slow
			bot.Speed = SwarmBotSpeed * 0.3
		}

		// Color by network complexity
		nodes, _, enabled := NASComplexity(genome)
		complexity := float64(nodes+enabled) / 40.0
		if complexity > 1.0 {
			complexity = 1.0
		}
		r := uint8(80 + complexity*175)
		g := uint8(200 - complexity*100)
		bot.LEDColor = [3]uint8{r, g, 180}
	}
}

func nasBoolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// EvolveNASEvolution evolves the neural architectures.
func EvolveNASEvolution(ss *SwarmState, sortedIndices []int) {
	evo := ss.NASEvo
	if evo == nil {
		return
	}

	n := len(ss.Bots)
	if len(evo.Genomes) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 25 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 2
	if eliteCount > parentCount {
		eliteCount = parentCount
	}

	// Clone parents
	parents := make([]*NASGenome, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		src := evo.Genomes[sortedIndices[i]]
		if src == nil {
			continue
		}
		parents[i] = cloneNASGenome(src)
	}

	if parentCount > 0 && parents[0] != nil {
		evo.BestFitness = parents[0].Fitness
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			continue
		}

		p1 := ss.Rng.Intn(parentCount)
		p2 := ss.Rng.Intn(parentCount)
		for p2 == p1 && parentCount > 1 {
			p2 = ss.Rng.Intn(parentCount)
		}

		if parents[p1] == nil || parents[p2] == nil {
			continue
		}

		child := NASCrossover(ss.Rng, parents[p1], parents[p2])
		NASMutate(ss.Rng, child, evo.NAS)

		evo.Genomes[botIdx] = child
	}

	updateNASEvoStats(evo)
	evo.Generation++

	logger.Info("NAS-EVO", "Gen %d: AvgNodes=%.1f, AvgConns=%.1f, BestFit=%.3f",
		evo.Generation, evo.AvgNodes, evo.AvgConnections, evo.BestFitness)
}

// cloneNASGenome deep-copies a genome.
func cloneNASGenome(src *NASGenome) *NASGenome {
	dst := &NASGenome{
		NextNodeID: src.NextNodeID,
		Fitness:    src.Fitness,
		Nodes:      make([]NASNode, len(src.Nodes)),
		Connections: make([]NASConnection, len(src.Connections)),
	}
	copy(dst.Nodes, src.Nodes)
	copy(dst.Connections, src.Connections)
	return dst
}

// updateNASEvoStats computes population-level statistics.
func updateNASEvoStats(evo *NASEvoState) {
	n := len(evo.Genomes)
	if n == 0 {
		return
	}

	totalNodes, totalConns := 0.0, 0.0
	count := 0
	for _, g := range evo.Genomes {
		if g == nil {
			continue
		}
		count++
		nodes, _, enabled := NASComplexity(g)
		totalNodes += float64(nodes)
		totalConns += float64(enabled)
	}

	if count > 0 {
		fn := float64(count)
		evo.AvgNodes = totalNodes / fn
		evo.AvgConnections = totalConns / fn
		evo.AvgComplexity = (evo.AvgNodes + evo.AvgConnections) / 2
	}
}

// NASEvoAvgComplexity returns the average network complexity.
func NASEvoAvgComplexity(evo *NASEvoState) float64 {
	if evo == nil {
		return 0
	}
	return evo.AvgComplexity
}

// NASEvoBotNodeCount returns the number of nodes in a bot's network.
func NASEvoBotNodeCount(evo *NASEvoState, botIdx int) int {
	if evo == nil || botIdx < 0 || botIdx >= len(evo.Genomes) || evo.Genomes[botIdx] == nil {
		return 0
	}
	return len(evo.Genomes[botIdx].Nodes)
}
