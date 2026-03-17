package swarm

import (
	"math"
	"swarmsim/logger"
)

// NeuralPruningState manages neural Darwinism — synaptic pruning.
// Each bot starts with an oversized brain (many connections). Connections
// that are frequently used get strengthened; unused connections are pruned.
// Like infant brain development: from chaotic to highly efficient.
type NeuralPruningState struct {
	Brains []PrunableBrain // per-bot brains

	// Parameters
	InitialConnections int     // connections at birth (default 50)
	PruneThreshold     float64 // usage below this → prune (default 0.05)
	PruneInterval      int     // ticks between pruning rounds (default 100)
	StrengthDecay      float64 // connection strength decay (default 0.01)
	UseGrowth          float64 // growth from usage (default 0.05)

	// Stats
	AvgConnections float64
	TotalPruned    int
	AvgEfficiency  float64 // output/connections ratio
	Generation     int
}

// PrunableBrain is an oversized neural network with pruning.
type PrunableBrain struct {
	// Connections: from input/hidden to hidden/output
	Connections []NeuralConnection
	HiddenSize  int
	InputSize   int
	OutputSize  int

	// Activity tracking
	TotalOutput float64
	TicksAlive  int
}

// NeuralConnection is a single synapse that can be pruned.
type NeuralConnection struct {
	From     int     // source neuron index
	To       int     // target neuron index
	Weight   float64 // connection weight
	Strength float64 // usage-based strength (0-1)
	UseCount int     // times this connection was significantly activated
	Alive    bool    // false = pruned
}

// InitNeuralPruning sets up the neural pruning system.
func InitNeuralPruning(ss *SwarmState) {
	n := len(ss.Bots)
	np := &NeuralPruningState{
		Brains:             make([]PrunableBrain, n),
		InitialConnections: 50,
		PruneThreshold:     0.05,
		PruneInterval:      100,
		StrengthDecay:      0.01,
		UseGrowth:          0.05,
	}

	inputSize := 5  // pickupDist, dropoffDist, neighbors, carrying, speed
	hiddenSize := 8
	outputSize := 3 // speed, turn, action

	for i := 0; i < n; i++ {
		brain := &np.Brains[i]
		brain.InputSize = inputSize
		brain.HiddenSize = hiddenSize
		brain.OutputSize = outputSize
		brain.Connections = make([]NeuralConnection, np.InitialConnections)

		totalNeurons := inputSize + hiddenSize + outputSize
		for c := 0; c < np.InitialConnections; c++ {
			from := ss.Rng.Intn(inputSize + hiddenSize)
			to := inputSize + ss.Rng.Intn(hiddenSize+outputSize)
			if to == from {
				to = (to + 1) % totalNeurons
				if to < inputSize {
					to = inputSize
				}
			}

			brain.Connections[c] = NeuralConnection{
				From:     from,
				To:       to,
				Weight:   (ss.Rng.Float64() - 0.5) * 2.0,
				Strength: 0.5,
				Alive:    true,
			}
		}
	}

	ss.NeuralPruning = np
	logger.Info("PRUNE", "Initialisiert: %d Bots mit je %d Verbindungen", n, np.InitialConnections)
}

// ClearNeuralPruning disables the pruning system.
func ClearNeuralPruning(ss *SwarmState) {
	ss.NeuralPruning = nil
	ss.NeuralPruningOn = false
}

// TickNeuralPruning runs one tick of neural activation and pruning.
func TickNeuralPruning(ss *SwarmState) {
	np := ss.NeuralPruning
	if np == nil {
		return
	}

	n := len(ss.Bots)
	if len(np.Brains) != n {
		return
	}

	totalConn := 0
	totalEff := 0.0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		brain := &np.Brains[i]

		// Forward pass
		outputs := forwardPrunable(brain, bot)

		// Apply outputs
		bot.Speed *= 0.7 + outputs[0]*0.6
		bot.Angle += outputs[1] * 0.2

		brain.TotalOutput += math.Abs(outputs[0]) + math.Abs(outputs[1]) + math.Abs(outputs[2])
		brain.TicksAlive++

		// Decay connection strength
		alive := 0
		for c := range brain.Connections {
			if !brain.Connections[c].Alive {
				continue
			}
			brain.Connections[c].Strength -= np.StrengthDecay
			if brain.Connections[c].Strength < 0 {
				brain.Connections[c].Strength = 0
			}
			alive++
		}

		totalConn += alive
		if alive > 0 {
			totalEff += brain.TotalOutput / float64(alive)
		}
	}

	// Periodic pruning
	if ss.Tick%np.PruneInterval == 0 && ss.Tick > 0 {
		pruned := pruneConnections(np)
		np.TotalPruned += pruned
	}

	if n > 0 {
		np.AvgConnections = float64(totalConn) / float64(n)
		np.AvgEfficiency = totalEff / float64(n)
	}
}

// forwardPrunable runs the prunable network forward.
func forwardPrunable(brain *PrunableBrain, bot *SwarmBot) [3]float64 {
	totalNeurons := brain.InputSize + brain.HiddenSize + brain.OutputSize
	activations := make([]float64, totalNeurons)

	// Set inputs
	activations[0] = clampF(bot.NearestPickupDist/400.0, 0, 1)
	activations[1] = clampF(bot.NearestDropoffDist/400.0, 0, 1)
	activations[2] = clampF(float64(bot.NeighborCount)/15.0, 0, 1)
	carry := 0.0
	if bot.CarryingPkg >= 0 {
		carry = 1.0
	}
	activations[3] = carry
	activations[4] = clampF(bot.Speed/SwarmBotSpeed, 0, 1)

	// Propagate through alive connections
	for c := range brain.Connections {
		conn := &brain.Connections[c]
		if !conn.Alive || conn.From >= totalNeurons || conn.To >= totalNeurons {
			continue
		}

		signal := activations[conn.From] * conn.Weight
		activations[conn.To] += signal

		// Track usage
		if math.Abs(signal) > 0.01 {
			conn.UseCount++
			conn.Strength += brain.Strength(signal)
			if conn.Strength > 1 {
				conn.Strength = 1
			}
		}
	}

	// Apply tanh to hidden and output
	for j := brain.InputSize; j < totalNeurons; j++ {
		activations[j] = math.Tanh(activations[j])
	}

	outStart := brain.InputSize + brain.HiddenSize
	return [3]float64{
		activations[outStart],
		activations[outStart+1],
		activations[outStart+2],
	}
}

// Strength computes how much to strengthen a connection based on signal.
func (b *PrunableBrain) Strength(signal float64) float64 {
	return math.Abs(signal) * 0.05
}

// pruneConnections removes weak connections.
func pruneConnections(np *NeuralPruningState) int {
	pruned := 0
	for i := range np.Brains {
		brain := &np.Brains[i]

		// Count alive connections
		alive := 0
		for _, c := range brain.Connections {
			if c.Alive {
				alive++
			}
		}

		// Don't prune below minimum
		minConnections := 10
		if alive <= minConnections {
			continue
		}

		for c := range brain.Connections {
			conn := &brain.Connections[c]
			if !conn.Alive {
				continue
			}
			if conn.Strength < np.PruneThreshold && alive > minConnections {
				conn.Alive = false
				pruned++
				alive--
			}
		}
	}
	return pruned
}

// EvolveNeuralPruning evolves brain structure based on fitness.
func EvolveNeuralPruning(ss *SwarmState, sortedIndices []int) {
	np := ss.NeuralPruning
	if np == nil {
		return
	}

	n := len(ss.Bots)
	if len(np.Brains) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 25 / 100
	if parentCount < 2 {
		parentCount = 2
	}

	for rank, botIdx := range sortedIndices {
		if rank < 2 {
			continue
		}

		p := sortedIndices[ss.Rng.Intn(parentCount)]
		parent := &np.Brains[p]
		child := &np.Brains[botIdx]

		// Copy alive connections from parent
		child.Connections = make([]NeuralConnection, 0, len(parent.Connections))
		for _, c := range parent.Connections {
			if c.Alive {
				nc := c
				nc.UseCount = 0
				nc.Strength = 0.5

				// Mutate weight
				if ss.Rng.Float64() < 0.1 {
					nc.Weight += ss.Rng.NormFloat64() * 0.3
					nc.Weight = clampF(nc.Weight, -3, 3)
				}
				child.Connections = append(child.Connections, nc)
			}
		}

		// Add some new random connections (neurogenesis)
		totalNeurons := child.InputSize + child.HiddenSize + child.OutputSize
		newConns := 3 + ss.Rng.Intn(5)
		for c := 0; c < newConns; c++ {
			from := ss.Rng.Intn(child.InputSize + child.HiddenSize)
			to := child.InputSize + ss.Rng.Intn(child.HiddenSize+child.OutputSize)
			if to < child.InputSize {
				to = child.InputSize
			}
			if to >= totalNeurons {
				to = totalNeurons - 1
			}

			child.Connections = append(child.Connections, NeuralConnection{
				From:     from,
				To:       to,
				Weight:   (ss.Rng.Float64() - 0.5) * 1.0,
				Strength: 0.5,
				Alive:    true,
			})
		}

		child.TotalOutput = 0
		child.TicksAlive = 0
	}

	np.Generation++
	logger.Info("PRUNE", "Gen %d: AvgConn=%.1f, TotalPruned=%d, Efficiency=%.3f",
		np.Generation, np.AvgConnections, np.TotalPruned, np.AvgEfficiency)
}

// PruningAvgConnections returns average connections per bot.
func PruningAvgConnections(np *NeuralPruningState) float64 {
	if np == nil {
		return 0
	}
	return np.AvgConnections
}

// PruningTotalPruned returns total pruned connections.
func PruningTotalPruned(np *NeuralPruningState) int {
	if np == nil {
		return 0
	}
	return np.TotalPruned
}

// PruningEfficiency returns average efficiency.
func PruningEfficiency(np *NeuralPruningState) float64 {
	if np == nil {
		return 0
	}
	return np.AvgEfficiency
}
