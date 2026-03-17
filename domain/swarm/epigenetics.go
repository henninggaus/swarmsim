package swarm

import (
	"math"
	"swarmsim/logger"
)

// EpigeneticsState manages heritable environmental marks.
// Bots can methylate (silence) or acetylate (activate) genes based on
// experience. These epigenetic marks are partially inherited by offspring.
// A bot that experienced starvation passes "thrifty genes" to children.
// Lamarckian evolution where environment shapes heritable traits.
type EpigeneticsState struct {
	Marks []BotEpiMarks // per-bot epigenetic marks

	// Parameters
	NumGenes        int     // genes that can be marked (default 8)
	MethylateRate   float64 // rate of silencing (default 0.02)
	AcetylateRate   float64 // rate of activation (default 0.02)
	InheritRate     float64 // fraction of marks inherited (default 0.7)
	DecayRate       float64 // marks slowly revert (default 0.005)

	// Stats
	AvgMethylation float64
	AvgAcetylation float64
	EpiDiversity   float64 // how different bots' marks are
	Generation     int
}

// BotEpiMarks holds one bot's epigenetic state.
type BotEpiMarks struct {
	// Methylation levels per gene (0=fully active, 1=fully silenced)
	Methylation [8]float64

	// Acetylation levels per gene (0=baseline, 1=fully boosted)
	Acetylation [8]float64

	// Experience counters that drive marking
	StarvationExp  float64 // accumulated starvation experience
	CrowdingExp    float64 // accumulated crowding stress
	ExplorationExp float64 // accumulated exploration success
	CooperationExp float64 // accumulated cooperation events
}

// Gene indices for epigenetic control
const (
	EpiGeneSpeed      = 0 // base speed modifier
	EpiGeneEfficiency = 1 // energy efficiency
	EpiGeneSocial     = 2 // social behavior
	EpiGeneExplore    = 3 // exploration tendency
	EpiGeneAggression = 4 // competitive behavior
	EpiGeneMemory     = 5 // learning rate
	EpiGeneSensor     = 6 // sensor sensitivity
	EpiGeneResilience = 7 // stress resistance
)

// EpiGeneName returns a gene name.
func EpiGeneName(idx int) string {
	names := []string{
		"Geschwindigkeit", "Effizienz", "Sozialverhalten", "Exploration",
		"Aggression", "Gedaechtnis", "Sensorik", "Resilienz",
	}
	if idx >= 0 && idx < len(names) {
		return names[idx]
	}
	return "?"
}

// InitEpigenetics sets up the epigenetics system.
func InitEpigenetics(ss *SwarmState) {
	n := len(ss.Bots)
	ep := &EpigeneticsState{
		Marks:         make([]BotEpiMarks, n),
		NumGenes:      8,
		MethylateRate: 0.02,
		AcetylateRate: 0.02,
		InheritRate:   0.7,
		DecayRate:     0.005,
	}

	// Small random initial marks
	for i := 0; i < n; i++ {
		for g := 0; g < 8; g++ {
			ep.Marks[i].Methylation[g] = ss.Rng.Float64() * 0.1
			ep.Marks[i].Acetylation[g] = ss.Rng.Float64() * 0.1
		}
	}

	ss.Epigenetics = ep
	logger.Info("EPI", "Initialisiert: %d Bots mit %d epigenetischen Markern", n, ep.NumGenes)
}

// ClearEpigenetics disables the epigenetics system.
func ClearEpigenetics(ss *SwarmState) {
	ss.Epigenetics = nil
	ss.EpigeneticsOn = false
}

// TickEpigenetics runs one tick of epigenetic regulation.
func TickEpigenetics(ss *SwarmState) {
	ep := ss.Epigenetics
	if ep == nil {
		return
	}

	n := len(ss.Bots)
	if len(ep.Marks) != n {
		return
	}

	sumMeth, sumAcet := 0.0, 0.0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		marks := &ep.Marks[i]

		// Accumulate experience
		accumulateEpiExperience(bot, marks)

		// Apply epigenetic modifications based on experience
		applyEpiModifications(ep, marks)

		// Decay marks toward baseline
		for g := 0; g < 8; g++ {
			marks.Methylation[g] -= ep.DecayRate
			marks.Acetylation[g] -= ep.DecayRate
			marks.Methylation[g] = clampF(marks.Methylation[g], 0, 1)
			marks.Acetylation[g] = clampF(marks.Acetylation[g], 0, 1)
		}

		// Apply epigenetic effects to behavior
		applyEpiEffects(ss, bot, marks)

		// Stats
		for g := 0; g < 8; g++ {
			sumMeth += marks.Methylation[g]
			sumAcet += marks.Acetylation[g]
		}
	}

	totalMarks := float64(n * 8)
	if totalMarks > 0 {
		ep.AvgMethylation = sumMeth / totalMarks
		ep.AvgAcetylation = sumAcet / totalMarks
	}

	// Compute diversity
	computeEpiDiversity(ep)
}

// accumulateEpiExperience tracks what the bot is experiencing.
func accumulateEpiExperience(bot *SwarmBot, marks *BotEpiMarks) {
	// Starvation: far from resources for long
	if bot.NearestPickupDist > 200 && bot.CarryingPkg < 0 {
		marks.StarvationExp += 0.01
	} else {
		marks.StarvationExp *= 0.99
	}

	// Crowding: too many neighbors
	if bot.NeighborCount > 8 {
		marks.CrowdingExp += 0.01
	} else {
		marks.CrowdingExp *= 0.99
	}

	// Exploration success: moving fast while alone
	if bot.Speed > SwarmBotSpeed*0.8 && bot.NeighborCount < 3 {
		marks.ExplorationExp += 0.005
	}

	// Cooperation: near others with good outcomes
	if bot.NeighborCount > 3 && bot.CarryingPkg >= 0 {
		marks.CooperationExp += 0.005
	}

	marks.StarvationExp = clampF(marks.StarvationExp, 0, 1)
	marks.CrowdingExp = clampF(marks.CrowdingExp, 0, 1)
	marks.ExplorationExp = clampF(marks.ExplorationExp, 0, 1)
	marks.CooperationExp = clampF(marks.CooperationExp, 0, 1)
}

// applyEpiModifications changes marks based on experience.
func applyEpiModifications(ep *EpigeneticsState, marks *BotEpiMarks) {
	// Starvation → methylate speed (slow down), acetylate efficiency
	if marks.StarvationExp > 0.3 {
		marks.Methylation[EpiGeneSpeed] += ep.MethylateRate
		marks.Acetylation[EpiGeneEfficiency] += ep.AcetylateRate
		marks.Acetylation[EpiGeneResilience] += ep.AcetylateRate * 0.5
	}

	// Crowding → methylate social, acetylate exploration
	if marks.CrowdingExp > 0.3 {
		marks.Methylation[EpiGeneSocial] += ep.MethylateRate
		marks.Acetylation[EpiGeneExplore] += ep.AcetylateRate
	}

	// Exploration success → acetylate sensor and memory
	if marks.ExplorationExp > 0.3 {
		marks.Acetylation[EpiGeneSensor] += ep.AcetylateRate
		marks.Acetylation[EpiGeneMemory] += ep.AcetylateRate * 0.5
	}

	// Cooperation → acetylate social, methylate aggression
	if marks.CooperationExp > 0.3 {
		marks.Acetylation[EpiGeneSocial] += ep.AcetylateRate
		marks.Methylation[EpiGeneAggression] += ep.MethylateRate
	}

	// Clamp all
	for g := 0; g < 8; g++ {
		marks.Methylation[g] = clampF(marks.Methylation[g], 0, 1)
		marks.Acetylation[g] = clampF(marks.Acetylation[g], 0, 1)
	}
}

// applyEpiEffects modifies bot behavior based on epigenetic state.
func applyEpiEffects(ss *SwarmState, bot *SwarmBot, marks *BotEpiMarks) {
	// Net expression = acetylation - methylation (positive = boosted)
	speedExpr := marks.Acetylation[EpiGeneSpeed] - marks.Methylation[EpiGeneSpeed]
	exploreExpr := marks.Acetylation[EpiGeneExplore] - marks.Methylation[EpiGeneExplore]

	bot.Speed *= 1.0 + speedExpr*0.2
	bot.Angle += exploreExpr * (ss.Rng.Float64() - 0.5) * 0.1

	// Color based on dominant epigenetic profile
	methLevel := 0.0
	acetLevel := 0.0
	for g := 0; g < 8; g++ {
		methLevel += marks.Methylation[g]
		acetLevel += marks.Acetylation[g]
	}
	methLevel /= 8
	acetLevel /= 8

	// More methylated = cooler (blue), more acetylated = warmer (red)
	r := uint8(100 + acetLevel*155)
	b := uint8(100 + methLevel*155)
	g := uint8(100)
	bot.LEDColor = [3]uint8{r, g, b}
}

// EvolveEpigenetics inherits epigenetic marks to offspring.
func EvolveEpigenetics(ss *SwarmState, sortedIndices []int) {
	ep := ss.Epigenetics
	if ep == nil {
		return
	}

	n := len(ss.Bots)
	if len(ep.Marks) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 25 / 100
	if parentCount < 2 {
		parentCount = 2
	}

	parents := make([]BotEpiMarks, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parents[i] = ep.Marks[sortedIndices[i]]
	}

	for rank, botIdx := range sortedIndices {
		if rank < 2 {
			continue
		}

		p := ss.Rng.Intn(parentCount)
		child := BotEpiMarks{}

		// Inherit marks with some noise
		for g := 0; g < 8; g++ {
			child.Methylation[g] = parents[p].Methylation[g] * ep.InheritRate
			child.Acetylation[g] = parents[p].Acetylation[g] * ep.InheritRate

			// Epigenetic noise
			child.Methylation[g] += (ss.Rng.Float64() - 0.5) * 0.05
			child.Acetylation[g] += (ss.Rng.Float64() - 0.5) * 0.05
			child.Methylation[g] = clampF(child.Methylation[g], 0, 1)
			child.Acetylation[g] = clampF(child.Acetylation[g], 0, 1)
		}

		// Reset experience (new bot, fresh start)
		child.StarvationExp = 0
		child.CrowdingExp = 0
		child.ExplorationExp = 0
		child.CooperationExp = 0

		ep.Marks[botIdx] = child
	}

	ep.Generation++
	logger.Info("EPI", "Gen %d: AvgMeth=%.3f, AvgAcet=%.3f, Diversity=%.3f",
		ep.Generation, ep.AvgMethylation, ep.AvgAcetylation, ep.EpiDiversity)
}

// computeEpiDiversity measures how different bots' marks are.
func computeEpiDiversity(ep *EpigeneticsState) {
	n := len(ep.Marks)
	if n < 2 {
		ep.EpiDiversity = 0
		return
	}

	// Average pairwise difference (sample)
	samples := 50
	if samples > n*(n-1)/2 {
		samples = n * (n - 1) / 2
	}

	totalDiff := 0.0
	for s := 0; s < samples; s++ {
		i := s % n
		j := (s + 1 + s/n) % n
		if i == j {
			j = (j + 1) % n
		}

		diff := 0.0
		for g := 0; g < 8; g++ {
			diff += math.Abs(ep.Marks[i].Methylation[g] - ep.Marks[j].Methylation[g])
			diff += math.Abs(ep.Marks[i].Acetylation[g] - ep.Marks[j].Acetylation[g])
		}
		totalDiff += diff / 16 // normalize by number of marks
	}

	ep.EpiDiversity = totalDiff / float64(samples)
}

// EpiAvgMethylation returns average methylation level.
func EpiAvgMethylation(ep *EpigeneticsState) float64 {
	if ep == nil {
		return 0
	}
	return ep.AvgMethylation
}

// EpiAvgAcetylation returns average acetylation level.
func EpiAvgAcetylation(ep *EpigeneticsState) float64 {
	if ep == nil {
		return 0
	}
	return ep.AvgAcetylation
}

// EpiDiversity returns epigenetic diversity.
func EpiDiversity(ep *EpigeneticsState) float64 {
	if ep == nil {
		return 0
	}
	return ep.EpiDiversity
}

// BotMethylation returns a bot's total methylation level.
func BotMethylation(ep *EpigeneticsState, botIdx int) float64 {
	if ep == nil || botIdx < 0 || botIdx >= len(ep.Marks) {
		return 0
	}
	sum := 0.0
	for g := 0; g < 8; g++ {
		sum += ep.Marks[botIdx].Methylation[g]
	}
	return sum / 8
}
