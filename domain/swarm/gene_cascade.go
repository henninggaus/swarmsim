package swarm

import (
	"math"
	"swarmsim/logger"
)

// GeneCascadeState manages gene regulatory cascades.
// Each bot has a chain of genes where activation of one gene can
// trigger or suppress downstream genes. Environmental signals trigger
// the first gene, causing a cascade of behavioral changes — like
// real cell differentiation where one signal causes a complex response.
type GeneCascadeState struct {
	Cascades []BotCascade // per-bot gene cascades

	CascadeLen   int     // genes per cascade (default 6)
	ActivateThresh float64 // threshold for gene activation (default 0.5)
	PropagateRate  float64 // how fast signals propagate (default 0.1)
	DecayRate      float64 // expression decay per tick (default 0.02)

	// Stats
	AvgExpression float64 // average gene expression level
	ActiveGenes   int     // total active genes across all bots
	CascadeEvents int     // total cascade propagation events
	Generation    int
}

// BotCascade holds one bot's gene regulatory cascade.
type BotCascade struct {
	// Expression levels for each gene (0-1)
	Expression []float64

	// Regulation matrix: how gene i affects gene j
	// Positive = activating, negative = suppressing
	Regulation [][]float64

	// Environmental sensitivity: which inputs trigger gene 0
	EnvSensitivity [4]float64 // pickup, dropoff, neighbors, speed

	// Current phenotype (behavioral output)
	Phenotype CascadePhenotype
}

// CascadePhenotype is the behavioral output of the cascade.
type CascadePhenotype struct {
	SpeedMod    float64 // speed multiplier
	TurnBias    float64 // turning tendency
	SocialPull  float64 // attraction to neighbors
	AggressionLvl float64 // competitive behavior
}

// InitGeneCascade sets up the gene cascade system.
func InitGeneCascade(ss *SwarmState) {
	n := len(ss.Bots)
	cascadeLen := 6

	gc := &GeneCascadeState{
		Cascades:       make([]BotCascade, n),
		CascadeLen:     cascadeLen,
		ActivateThresh: 0.5,
		PropagateRate:  0.1,
		DecayRate:      0.02,
	}

	for i := 0; i < n; i++ {
		c := &gc.Cascades[i]
		c.Expression = make([]float64, cascadeLen)
		c.Regulation = make([][]float64, cascadeLen)

		for g := 0; g < cascadeLen; g++ {
			c.Regulation[g] = make([]float64, cascadeLen)
			for h := 0; h < cascadeLen; h++ {
				if g != h && ss.Rng.Float64() < 0.4 {
					c.Regulation[g][h] = (ss.Rng.Float64() - 0.3) * 1.5
				}
			}
		}

		// Random environmental sensitivity
		for s := 0; s < 4; s++ {
			c.EnvSensitivity[s] = (ss.Rng.Float64() - 0.3) * 2.0
		}
	}

	ss.GeneCascade = gc
	logger.Info("GCAS", "Initialisiert: %d Bots mit %d-Gen Kaskaden", n, cascadeLen)
}

// ClearGeneCascade disables the gene cascade system.
func ClearGeneCascade(ss *SwarmState) {
	ss.GeneCascade = nil
	ss.GeneCascadeOn = false
}

// TickGeneCascade runs one tick of gene regulation.
func TickGeneCascade(ss *SwarmState) {
	gc := ss.GeneCascade
	if gc == nil {
		return
	}

	n := len(ss.Bots)
	if len(gc.Cascades) != n {
		return
	}

	totalExpr := 0.0
	activeGenes := 0
	cascadeEvents := 0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		c := &gc.Cascades[i]

		// Environmental input to gene 0
		envSignal := computeEnvSignal(bot, c)
		c.Expression[0] = c.Expression[0]*(1-gc.PropagateRate) + envSignal*gc.PropagateRate

		// Propagate through cascade
		newExpr := make([]float64, gc.CascadeLen)
		copy(newExpr, c.Expression)

		for g := 0; g < gc.CascadeLen; g++ {
			if c.Expression[g] < gc.ActivateThresh {
				newExpr[g] -= gc.DecayRate
				continue
			}

			// Active gene regulates downstream genes
			for h := 0; h < gc.CascadeLen; h++ {
				if g == h {
					continue
				}
				influence := c.Regulation[g][h] * c.Expression[g] * gc.PropagateRate
				if math.Abs(influence) > 0.01 {
					newExpr[h] += influence
					cascadeEvents++
				}
			}
		}

		// Clamp and apply
		for g := 0; g < gc.CascadeLen; g++ {
			c.Expression[g] = clampF(newExpr[g], 0, 1)
			totalExpr += c.Expression[g]
			if c.Expression[g] > gc.ActivateThresh {
				activeGenes++
			}
		}

		// Compute phenotype from expression
		computePhenotype(c, gc.CascadeLen)

		// Apply phenotype to bot
		applyCascadePhenotype(ss, bot, &c.Phenotype)
	}

	totalGenes := float64(n * gc.CascadeLen)
	if totalGenes > 0 {
		gc.AvgExpression = totalExpr / totalGenes
	}
	gc.ActiveGenes = activeGenes
	gc.CascadeEvents = cascadeEvents
}

func computeEnvSignal(bot *SwarmBot, c *BotCascade) float64 {
	inputs := [4]float64{
		clampF(1.0-bot.NearestPickupDist/200.0, 0, 1),
		clampF(1.0-bot.NearestDropoffDist/200.0, 0, 1),
		clampF(float64(bot.NeighborCount)/10.0, 0, 1),
		clampF(bot.Speed/SwarmBotSpeed, 0, 1),
	}

	signal := 0.0
	for s := 0; s < 4; s++ {
		signal += inputs[s] * c.EnvSensitivity[s]
	}
	return clampF(signal, 0, 1)
}

func computePhenotype(c *BotCascade, cascadeLen int) {
	// Map gene expression to behavioral outputs
	// Genes 0-1: speed, Gene 2: turning, Gene 3: social, Gene 4-5: aggression
	if cascadeLen >= 2 {
		c.Phenotype.SpeedMod = 0.7 + (c.Expression[0]+c.Expression[1])*0.3
	}
	if cascadeLen >= 3 {
		c.Phenotype.TurnBias = (c.Expression[2] - 0.5) * 0.4
	}
	if cascadeLen >= 4 {
		c.Phenotype.SocialPull = c.Expression[3] * 0.5
	}
	if cascadeLen >= 6 {
		c.Phenotype.AggressionLvl = (c.Expression[4] + c.Expression[5]) * 0.3
	}
}

func applyCascadePhenotype(ss *SwarmState, bot *SwarmBot, p *CascadePhenotype) {
	bot.Speed *= p.SpeedMod
	bot.Angle += p.TurnBias * (ss.Rng.Float64() - 0.5)

	// Color based on dominant gene expression
	r := uint8(100 + p.AggressionLvl*300)
	if r > 255 {
		r = 255
	}
	g := uint8(100 + p.SocialPull*300)
	if g > 255 {
		g = 255
	}
	b := uint8(100 + p.SpeedMod*100)
	if b > 255 {
		b = 255
	}
	bot.LEDColor = [3]uint8{r, g, b}
}

// EvolveGeneCascades evolves the regulation matrices.
func EvolveGeneCascades(ss *SwarmState, sortedIndices []int) {
	gc := ss.GeneCascade
	if gc == nil {
		return
	}

	n := len(ss.Bots)
	if len(gc.Cascades) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 25 / 100
	if parentCount < 2 {
		parentCount = 2
	}

	parents := make([]BotCascade, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parents[i] = cloneCascade(gc.Cascades[sortedIndices[i]])
	}

	for rank, botIdx := range sortedIndices {
		if rank < 2 {
			continue
		}

		p := ss.Rng.Intn(parentCount)
		child := cloneCascade(parents[p])

		// Mutate regulation matrix
		for g := range child.Regulation {
			for h := range child.Regulation[g] {
				if ss.Rng.Float64() < 0.1 {
					child.Regulation[g][h] += ss.Rng.NormFloat64() * 0.2
					child.Regulation[g][h] = clampF(child.Regulation[g][h], -2, 2)
				}
			}
		}

		// Mutate env sensitivity
		for s := 0; s < 4; s++ {
			if ss.Rng.Float64() < 0.15 {
				child.EnvSensitivity[s] += ss.Rng.NormFloat64() * 0.3
				child.EnvSensitivity[s] = clampF(child.EnvSensitivity[s], -2, 2)
			}
		}

		// Reset expression
		for g := range child.Expression {
			child.Expression[g] = 0
		}

		gc.Cascades[botIdx] = child
	}

	gc.Generation++
	logger.Info("GCAS", "Gen %d: AvgExpr=%.3f, ActiveGenes=%d",
		gc.Generation, gc.AvgExpression, gc.ActiveGenes)
}

func cloneCascade(src BotCascade) BotCascade {
	dst := BotCascade{
		Expression:     make([]float64, len(src.Expression)),
		Regulation:     make([][]float64, len(src.Regulation)),
		EnvSensitivity: src.EnvSensitivity,
	}
	copy(dst.Expression, src.Expression)
	for g := range src.Regulation {
		dst.Regulation[g] = make([]float64, len(src.Regulation[g]))
		copy(dst.Regulation[g], src.Regulation[g])
	}
	return dst
}

// CascadeAvgExpr returns average gene expression.
func CascadeAvgExpr(gc *GeneCascadeState) float64 {
	if gc == nil {
		return 0
	}
	return gc.AvgExpression
}

// CascadeActiveGenes returns total active genes.
func CascadeActiveGenes(gc *GeneCascadeState) int {
	if gc == nil {
		return 0
	}
	return gc.ActiveGenes
}
