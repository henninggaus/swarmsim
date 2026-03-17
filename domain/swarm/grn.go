package swarm

import (
	"math"
	"swarmsim/logger"
)

// GRNState manages the Genetic Regulatory Network system.
// Instead of neural networks, each bot has a GRN where "genes"
// are activated or repressed by signals. Gene expression levels
// determine bot behavior. GRNs can evolve structure and weights.
type GRNState struct {
	NumGenes       int     // genes per bot (default 8)
	NumInputGenes  int     // input genes mapped to sensors (default 6)
	NumOutputGenes int     // output genes mapped to actions (default 4)
	DecayRate      float64 // expression decay per tick (default 0.1)
	MaxExpression  float64 // cap on expression level (default 5.0)

	Networks []GRNetwork // per-bot GRN
	Generation int
}

// GRNetwork is a single bot's gene regulatory network.
type GRNetwork struct {
	// Regulatory matrix: Regulation[i][j] = how gene i affects gene j
	// Positive = activation, negative = repression
	Regulation [][]float64
	// Current expression levels per gene
	Expression []float64
	// Thresholds for gene activation
	Thresholds []float64
}

// InitGRN sets up the genetic regulatory network system.
func InitGRN(ss *SwarmState, numGenes int) {
	if numGenes < 6 {
		numGenes = 6
	}
	if numGenes > 20 {
		numGenes = 20
	}

	n := len(ss.Bots)
	gs := &GRNState{
		NumGenes:       numGenes,
		NumInputGenes:  6,
		NumOutputGenes: 4,
		DecayRate:      0.1,
		MaxExpression:  5.0,
		Networks:       make([]GRNetwork, n),
	}

	for i := 0; i < n; i++ {
		gs.Networks[i] = randomGRNetwork(ss, numGenes)
	}

	ss.GRN = gs
	logger.Info("GRN", "Initialisiert: %d Bots, %d Gene pro Bot", n, numGenes)
}

// ClearGRN disables the GRN system.
func ClearGRN(ss *SwarmState) {
	ss.GRN = nil
	ss.GRNOn = false
}

// randomGRNetwork creates a random GRN.
func randomGRNetwork(ss *SwarmState, numGenes int) GRNetwork {
	net := GRNetwork{
		Regulation: make([][]float64, numGenes),
		Expression: make([]float64, numGenes),
		Thresholds: make([]float64, numGenes),
	}

	for i := 0; i < numGenes; i++ {
		net.Regulation[i] = make([]float64, numGenes)
		for j := 0; j < numGenes; j++ {
			// Sparse regulation: only ~30% of connections active
			if ss.Rng.Float64() < 0.3 {
				net.Regulation[i][j] = (ss.Rng.Float64() - 0.5) * 2.0
			}
		}
		net.Expression[i] = ss.Rng.Float64() * 0.5
		net.Thresholds[i] = ss.Rng.Float64() * 0.5
	}
	return net
}

// TickGRN runs one tick of gene expression for all bots.
func TickGRN(ss *SwarmState) {
	gs := ss.GRN
	if gs == nil {
		return
	}

	n := len(ss.Bots)
	if len(gs.Networks) != n {
		return
	}

	for i := range ss.Bots {
		net := &gs.Networks[i]
		bot := &ss.Bots[i]

		// Set input gene expression from sensors
		setGRNInputs(gs, net, bot)

		// Update gene expression
		updateExpression(gs, net)

		// Read output genes to control behavior
		applyGRNOutputs(gs, net, bot)
	}
}

// setGRNInputs maps bot sensor values to input gene expression.
func setGRNInputs(gs *GRNState, net *GRNetwork, bot *SwarmBot) {
	if gs.NumInputGenes > len(net.Expression) {
		return
	}

	// Input gene mapping (same as neuro inputs):
	// [0] carrying, [1] nearest_dist, [2] pickup_dist, [3] dropoff_dist,
	// [4] neighbors, [5] speed

	if bot.CarryingPkg >= 0 {
		net.Expression[0] = gs.MaxExpression
	} else {
		net.Expression[0] = 0
	}

	nd := bot.NearestDist
	if nd > 200 {
		nd = 200
	}
	net.Expression[1] = (nd / 200.0) * gs.MaxExpression

	pd := bot.NearestPickupDist
	if pd > 500 {
		pd = 500
	}
	net.Expression[2] = (1 - pd/500.0) * gs.MaxExpression // closer = higher

	dd := bot.NearestDropoffDist
	if dd > 500 {
		dd = 500
	}
	net.Expression[3] = (1 - dd/500.0) * gs.MaxExpression

	nc := float64(bot.NeighborCount)
	if nc > 10 {
		nc = 10
	}
	net.Expression[4] = (nc / 10.0) * gs.MaxExpression

	net.Expression[5] = (bot.Speed / SwarmBotSpeed) * gs.MaxExpression
}

// updateExpression computes new expression levels using the regulatory matrix.
func updateExpression(gs *GRNState, net *GRNetwork) {
	numGenes := gs.NumGenes
	newExpr := make([]float64, numGenes)

	for j := gs.NumInputGenes; j < numGenes; j++ {
		// Sum regulatory inputs
		activation := 0.0
		for i := 0; i < numGenes; i++ {
			if net.Expression[i] > net.Thresholds[i] {
				activation += net.Regulation[i][j] * net.Expression[i]
			}
		}

		// Sigmoid activation
		newExpr[j] = gs.MaxExpression / (1.0 + math.Exp(-activation))

		// Decay toward zero
		newExpr[j] = newExpr[j]*(1-gs.DecayRate) + net.Expression[j]*gs.DecayRate
	}

	// Copy input genes (keep sensor values)
	for i := 0; i < gs.NumInputGenes && i < numGenes; i++ {
		newExpr[i] = net.Expression[i]
	}

	// Clamp
	for i := range newExpr {
		if newExpr[i] < 0 {
			newExpr[i] = 0
		}
		if newExpr[i] > gs.MaxExpression {
			newExpr[i] = gs.MaxExpression
		}
	}

	net.Expression = newExpr
}

// applyGRNOutputs maps output gene expression to bot actions.
func applyGRNOutputs(gs *GRNState, net *GRNetwork, bot *SwarmBot) {
	numGenes := gs.NumGenes
	outStart := numGenes - gs.NumOutputGenes
	if outStart < gs.NumInputGenes {
		outStart = gs.NumInputGenes
	}

	// Output genes:
	// [outStart+0] speed control
	// [outStart+1] turn direction
	// [outStart+2] turn magnitude
	// [outStart+3] LED intensity

	if outStart < numGenes {
		speedExpr := net.Expression[outStart] / gs.MaxExpression
		bot.Speed = speedExpr * SwarmBotSpeed
	}

	if outStart+1 < numGenes {
		turnDir := net.Expression[outStart+1] / gs.MaxExpression
		turnDir = turnDir*2 - 1 // map [0,1] to [-1,1]
		magnitude := 0.1
		if outStart+2 < numGenes {
			magnitude = net.Expression[outStart+2] / gs.MaxExpression * 0.3
		}
		bot.Angle += turnDir * magnitude
	}

	// LED color from expression levels
	if numGenes >= 3 {
		r := uint8(net.Expression[numGenes-3] / gs.MaxExpression * 255)
		g := uint8(net.Expression[numGenes-2] / gs.MaxExpression * 255)
		b := uint8(net.Expression[numGenes-1] / gs.MaxExpression * 255)
		bot.LEDColor = [3]uint8{r, g, b}
	}
}

// EvolveGRN evolves GRN networks using fitness-ranked selection.
func EvolveGRN(ss *SwarmState, sortedIndices []int) {
	gs := ss.GRN
	if gs == nil {
		return
	}

	n := len(ss.Bots)
	if len(gs.Networks) != n {
		return
	}

	parentCount := n * 20 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 3
	if eliteCount > parentCount {
		eliteCount = parentCount
	}
	freshCount := n * 10 / 100
	if freshCount < 1 {
		freshCount = 1
	}

	// Save parent networks
	parents := make([]GRNetwork, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parents[i] = cloneGRNetwork(gs.Networks[sortedIndices[i]])
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			gs.Networks[botIdx] = cloneGRNetwork(parents[rank])
		} else if rank >= n-freshCount {
			gs.Networks[botIdx] = randomGRNetwork(ss, gs.NumGenes)
		} else {
			p1 := ss.Rng.Intn(parentCount)
			p2 := ss.Rng.Intn(parentCount)
			gs.Networks[botIdx] = crossoverGRN(ss, &parents[p1], &parents[p2], gs.NumGenes)
		}
	}

	gs.Generation++
	logger.Info("GRN", "Gen %d: %d Eltern, %d Elite, %d Frisch",
		gs.Generation, parentCount, eliteCount, freshCount)
}

// cloneGRNetwork deep-copies a GRN.
func cloneGRNetwork(src GRNetwork) GRNetwork {
	numGenes := len(src.Expression)
	dst := GRNetwork{
		Regulation: make([][]float64, numGenes),
		Expression: make([]float64, numGenes),
		Thresholds: make([]float64, numGenes),
	}
	copy(dst.Expression, src.Expression)
	copy(dst.Thresholds, src.Thresholds)
	for i := 0; i < numGenes; i++ {
		dst.Regulation[i] = make([]float64, numGenes)
		copy(dst.Regulation[i], src.Regulation[i])
	}
	return dst
}

// crossoverGRN breeds two parent GRNs.
func crossoverGRN(ss *SwarmState, p1, p2 *GRNetwork, numGenes int) GRNetwork {
	child := GRNetwork{
		Regulation: make([][]float64, numGenes),
		Expression: make([]float64, numGenes),
		Thresholds: make([]float64, numGenes),
	}

	for i := 0; i < numGenes; i++ {
		child.Regulation[i] = make([]float64, numGenes)

		// Crossover threshold
		if ss.Rng.Float64() < 0.5 {
			child.Thresholds[i] = p1.Thresholds[i]
		} else {
			child.Thresholds[i] = p2.Thresholds[i]
		}
		// Mutation
		if ss.Rng.Float64() < 0.1 {
			child.Thresholds[i] += ss.Rng.NormFloat64() * 0.1
		}

		// Crossover regulation
		for j := 0; j < numGenes; j++ {
			if ss.Rng.Float64() < 0.5 {
				child.Regulation[i][j] = p1.Regulation[i][j]
			} else {
				child.Regulation[i][j] = p2.Regulation[i][j]
			}
			// Mutation
			if ss.Rng.Float64() < 0.1 {
				child.Regulation[i][j] += ss.Rng.NormFloat64() * 0.3
			}
			// Structural mutation: add/remove connection
			if ss.Rng.Float64() < 0.02 {
				if child.Regulation[i][j] == 0 {
					child.Regulation[i][j] = (ss.Rng.Float64() - 0.5) * 2.0
				} else {
					child.Regulation[i][j] = 0
				}
			}
		}

		child.Expression[i] = ss.Rng.Float64() * 0.5
	}

	return child
}

// GRNGeneCount returns number of genes per bot.
func GRNGeneCount(gs *GRNState) int {
	if gs == nil {
		return 0
	}
	return gs.NumGenes
}

// GRNExpression returns the expression levels of a bot's genes.
func GRNExpression(gs *GRNState, botIdx int) []float64 {
	if gs == nil || botIdx < 0 || botIdx >= len(gs.Networks) {
		return nil
	}
	return gs.Networks[botIdx].Expression
}

// GRNConnectivity returns the fraction of non-zero regulatory connections.
func GRNConnectivity(gs *GRNState) float64 {
	if gs == nil || len(gs.Networks) == 0 {
		return 0
	}

	totalConn := 0
	totalPossible := 0
	for _, net := range gs.Networks {
		for _, row := range net.Regulation {
			for _, w := range row {
				totalPossible++
				if w != 0 {
					totalConn++
				}
			}
		}
	}

	if totalPossible == 0 {
		return 0
	}
	return float64(totalConn) / float64(totalPossible)
}
