package swarm

import (
	"math"
	"math/rand"
)

// NASNode represents a node in a variable-topology neural network.
type NASNode struct {
	ID         int
	NodeType   int     // 0=input, 1=hidden, 2=output
	Bias       float64
	Activation float64 // cached last activation
}

// NASConnection represents a weighted connection between nodes.
type NASConnection struct {
	From    int
	To      int
	Weight  float64
	Enabled bool
	InnovID int // innovation number for NEAT-style tracking
}

// NASGenome is a variable-topology neural network genome.
type NASGenome struct {
	Nodes       []NASNode
	Connections []NASConnection
	NextNodeID  int
	Fitness     float64
}

// NASState tracks the NAS evolution process.
type NASState struct {
	InnovCounter int     // global innovation counter
	AddNodeRate  float64 // probability of adding a node (default 0.03)
	AddConnRate  float64 // probability of adding a connection (default 0.05)
	MutWeightRate float64 // probability of mutating a weight (default 0.8)
	MutWeightSigma float64 // mutation strength (default 0.5)
	Generation   int
}

// NewNASState creates a NAS evolution state with defaults.
func NewNASState() *NASState {
	return &NASState{
		AddNodeRate:    0.03,
		AddConnRate:    0.05,
		MutWeightRate:  0.8,
		MutWeightSigma: 0.5,
	}
}

// NewMinimalNASGenome creates a genome with inputs directly connected to outputs.
func NewMinimalNASGenome(rng *rand.Rand, numInputs, numOutputs int, nas *NASState) *NASGenome {
	g := &NASGenome{}

	// Input nodes
	for i := 0; i < numInputs; i++ {
		g.Nodes = append(g.Nodes, NASNode{ID: i, NodeType: 0})
	}
	// Output nodes
	for i := 0; i < numOutputs; i++ {
		g.Nodes = append(g.Nodes, NASNode{ID: numInputs + i, NodeType: 2})
	}
	g.NextNodeID = numInputs + numOutputs

	// Connect each input to each output with small random weights
	for i := 0; i < numInputs; i++ {
		for o := 0; o < numOutputs; o++ {
			nas.InnovCounter++
			g.Connections = append(g.Connections, NASConnection{
				From:    i,
				To:      numInputs + o,
				Weight:  (rng.Float64() - 0.5) * 2,
				Enabled: true,
				InnovID: nas.InnovCounter,
			})
		}
	}
	return g
}

// NASForward evaluates the network given inputs and returns output activations.
func NASForward(g *NASGenome, inputs []float64) []float64 {
	if g == nil || len(g.Nodes) == 0 {
		return nil
	}

	// Reset activations
	for i := range g.Nodes {
		g.Nodes[i].Activation = 0
	}

	// Set input activations
	nodeMap := make(map[int]*NASNode, len(g.Nodes))
	for i := range g.Nodes {
		nodeMap[g.Nodes[i].ID] = &g.Nodes[i]
	}

	inputIdx := 0
	for i := range g.Nodes {
		if g.Nodes[i].NodeType == 0 && inputIdx < len(inputs) {
			g.Nodes[i].Activation = inputs[inputIdx]
			inputIdx++
		}
	}

	// Propagate through connections (simple feedforward, max 3 passes for hidden layers)
	for pass := 0; pass < 3; pass++ {
		for _, conn := range g.Connections {
			if !conn.Enabled {
				continue
			}
			fromNode := nodeMap[conn.From]
			toNode := nodeMap[conn.To]
			if fromNode == nil || toNode == nil {
				continue
			}
			if toNode.NodeType == 0 {
				continue // don't write to input nodes
			}
			toNode.Activation += fromNode.Activation * conn.Weight
		}

		// Apply activation function to hidden and output nodes
		for i := range g.Nodes {
			if g.Nodes[i].NodeType != 0 {
				g.Nodes[i].Activation = nasTanh(g.Nodes[i].Activation + g.Nodes[i].Bias)
			}
		}
	}

	// Collect outputs
	var outputs []float64
	for i := range g.Nodes {
		if g.Nodes[i].NodeType == 2 {
			outputs = append(outputs, g.Nodes[i].Activation)
		}
	}
	return outputs
}

func nasTanh(x float64) float64 {
	return math.Tanh(x)
}

// NASMutate applies structural and weight mutations.
func NASMutate(rng *rand.Rand, g *NASGenome, nas *NASState) {
	if g == nil || nas == nil {
		return
	}

	// Mutate existing weights
	for i := range g.Connections {
		if rng.Float64() < nas.MutWeightRate {
			g.Connections[i].Weight += rng.NormFloat64() * nas.MutWeightSigma
		}
	}

	// Add node mutation
	if rng.Float64() < nas.AddNodeRate && len(g.Connections) > 0 {
		nasAddNode(rng, g, nas)
	}

	// Add connection mutation
	if rng.Float64() < nas.AddConnRate {
		nasAddConnection(rng, g, nas)
	}
}

// nasAddNode splits an existing connection with a new hidden node.
func nasAddNode(rng *rand.Rand, g *NASGenome, nas *NASState) {
	// Pick a random enabled connection
	enabled := []int{}
	for i, c := range g.Connections {
		if c.Enabled {
			enabled = append(enabled, i)
		}
	}
	if len(enabled) == 0 {
		return
	}

	idx := enabled[rng.Intn(len(enabled))]
	conn := &g.Connections[idx]
	conn.Enabled = false

	newNodeID := g.NextNodeID
	g.NextNodeID++
	g.Nodes = append(g.Nodes, NASNode{ID: newNodeID, NodeType: 1})

	nas.InnovCounter++
	g.Connections = append(g.Connections, NASConnection{
		From: conn.From, To: newNodeID,
		Weight: 1.0, Enabled: true, InnovID: nas.InnovCounter,
	})
	nas.InnovCounter++
	g.Connections = append(g.Connections, NASConnection{
		From: newNodeID, To: conn.To,
		Weight: conn.Weight, Enabled: true, InnovID: nas.InnovCounter,
	})
}

// nasAddConnection adds a new connection between two unconnected nodes.
func nasAddConnection(rng *rand.Rand, g *NASGenome, nas *NASState) {
	if len(g.Nodes) < 2 {
		return
	}

	// Try a few times to find an unconnected pair
	for attempt := 0; attempt < 10; attempt++ {
		from := g.Nodes[rng.Intn(len(g.Nodes))]
		to := g.Nodes[rng.Intn(len(g.Nodes))]

		if from.ID == to.ID {
			continue
		}
		if to.NodeType == 0 {
			continue // don't connect to input
		}
		if from.NodeType == 2 && to.NodeType == 2 {
			continue // don't connect output→output
		}

		// Check if connection exists
		exists := false
		for _, c := range g.Connections {
			if c.From == from.ID && c.To == to.ID {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		nas.InnovCounter++
		g.Connections = append(g.Connections, NASConnection{
			From: from.ID, To: to.ID,
			Weight: (rng.Float64() - 0.5) * 2, Enabled: true, InnovID: nas.InnovCounter,
		})
		return
	}
}

// NASCrossover creates a child genome from two parents (fitter parent's structure preferred).
func NASCrossover(rng *rand.Rand, fitter, weaker *NASGenome) *NASGenome {
	child := &NASGenome{
		NextNodeID: fitter.NextNodeID,
	}

	// Copy all nodes from fitter parent
	child.Nodes = make([]NASNode, len(fitter.Nodes))
	copy(child.Nodes, fitter.Nodes)

	// Build innovation map for weaker parent
	weakMap := map[int]*NASConnection{}
	for i := range weaker.Connections {
		weakMap[weaker.Connections[i].InnovID] = &weaker.Connections[i]
	}

	// Match genes by innovation number
	for _, conn := range fitter.Connections {
		if wc, ok := weakMap[conn.InnovID]; ok {
			// Matching gene: randomly pick
			if rng.Float64() < 0.5 {
				child.Connections = append(child.Connections, conn)
			} else {
				child.Connections = append(child.Connections, *wc)
			}
		} else {
			// Excess/disjoint: take from fitter
			child.Connections = append(child.Connections, conn)
		}
	}

	return child
}

// NASComplexity returns (node count, connection count, enabled connections).
func NASComplexity(g *NASGenome) (int, int, int) {
	if g == nil {
		return 0, 0, 0
	}
	enabled := 0
	for _, c := range g.Connections {
		if c.Enabled {
			enabled++
		}
	}
	return len(g.Nodes), len(g.Connections), enabled
}
