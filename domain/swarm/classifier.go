package swarm

import (
	"math"
	"swarmsim/logger"
)

// ClassifierState manages a Learning Classifier System (LCS) for each bot.
// Each bot carries a population of IF-THEN rules. Rules compete for
// activation based on current conditions. Successful rules gain strength,
// weak rules are replaced. Over generations, bots evolve optimal rule sets.
type ClassifierState struct {
	RuleSets []BotRuleSet // per-bot rule populations

	MaxRules int     // rules per bot (default 16)
	LearnRate float64 // strength adjustment rate (default 0.05)
	TaxRate   float64 // cost per activation (default 0.01)

	// Stats
	AvgRuleStrength float64
	AvgActiveRules  float64
	BestRuleStr     float64
	Generation      int
}

// BotRuleSet holds one bot's classifier rules.
type BotRuleSet struct {
	Rules      []ClassifierRule
	LastAction int // last chosen action
}

// ClassifierRule is an IF-THEN rule with fitness tracking.
type ClassifierRule struct {
	// Condition: each entry is (min, max) for an input dimension
	// Input dimensions: 0=pickupDist, 1=dropoffDist, 2=neighbors, 3=carrying, 4=speed
	Conditions [5][2]float64 // [dim][min,max]

	// Action: what to do when activated
	Action     int     // 0=forward, 1=left, 2=right, 3=seek-pickup, 4=seek-dropoff
	Strength   float64 // rule fitness/quality (0-1)
	MatchCount int     // times this rule was activated
	RewardSum  float64 // accumulated reward
}

const numClassifierActions = 5

// InitClassifier sets up the classifier system.
func InitClassifier(ss *SwarmState) {
	n := len(ss.Bots)
	cs := &ClassifierState{
		RuleSets:  make([]BotRuleSet, n),
		MaxRules:  16,
		LearnRate: 0.05,
		TaxRate:   0.01,
	}

	for i := 0; i < n; i++ {
		cs.RuleSets[i].Rules = make([]ClassifierRule, cs.MaxRules)
		for j := 0; j < cs.MaxRules; j++ {
			cs.RuleSets[i].Rules[j] = randomRule(ss)
		}
	}

	ss.Classifier = cs
	logger.Info("LCS", "Initialisiert: %d Bots mit je %d Regeln", n, cs.MaxRules)
}

// ClearClassifier disables the classifier system.
func ClearClassifier(ss *SwarmState) {
	ss.Classifier = nil
	ss.ClassifierOn = false
}

// randomRule creates a random classifier rule.
func randomRule(ss *SwarmState) ClassifierRule {
	r := ClassifierRule{
		Action:   ss.Rng.Intn(numClassifierActions),
		Strength: 0.5,
	}

	// Random condition ranges (normalized 0-1 inputs)
	// Some dimensions are "don't care" (full range) to ensure matching
	for d := 0; d < 5; d++ {
		if ss.Rng.Float64() < 0.4 {
			// Don't care — matches anything
			r.Conditions[d] = [2]float64{0, 1}
		} else {
			lo := ss.Rng.Float64() * 0.5
			hi := lo + 0.3 + ss.Rng.Float64()*0.4
			if hi > 1 {
				hi = 1
			}
			r.Conditions[d] = [2]float64{lo, hi}
		}
	}

	return r
}

// TickClassifier runs one tick of the classifier system.
func TickClassifier(ss *SwarmState) {
	cs := ss.Classifier
	if cs == nil {
		return
	}

	n := len(ss.Bots)
	if len(cs.RuleSets) != n {
		return
	}

	totalStr := 0.0
	totalActive := 0.0
	bestStr := 0.0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		rs := &cs.RuleSets[i]

		// Build normalized input vector
		inputs := classifierInputs(bot)

		// Find matching rules
		matches := []int{}
		for j, rule := range rs.Rules {
			if ruleMatches(rule, inputs) {
				matches = append(matches, j)
			}
		}

		if len(matches) == 0 {
			continue
		}

		// Select best matching rule (highest strength)
		bestIdx := matches[0]
		for _, j := range matches[1:] {
			if rs.Rules[j].Strength > rs.Rules[bestIdx].Strength {
				bestIdx = j
			}
		}

		// Execute action
		action := rs.Rules[bestIdx].Action
		applyClassifierAction(ss, bot, action)
		rs.LastAction = action

		// Tax the activated rule
		rs.Rules[bestIdx].Strength -= cs.TaxRate
		if rs.Rules[bestIdx].Strength < 0.01 {
			rs.Rules[bestIdx].Strength = 0.01
		}
		rs.Rules[bestIdx].MatchCount++

		// Reward based on outcome
		reward := classifierReward(bot)
		rs.Rules[bestIdx].Strength += cs.LearnRate * reward
		rs.Rules[bestIdx].RewardSum += reward
		if rs.Rules[bestIdx].Strength > 1 {
			rs.Rules[bestIdx].Strength = 1
		}

		// Track stats
		for _, rule := range rs.Rules {
			totalStr += rule.Strength
			if rule.Strength > bestStr {
				bestStr = rule.Strength
			}
		}
		totalActive += float64(len(matches))
	}

	totalRules := float64(n * cs.MaxRules)
	if totalRules > 0 {
		cs.AvgRuleStrength = totalStr / totalRules
	}
	if n > 0 {
		cs.AvgActiveRules = totalActive / float64(n)
	}
	cs.BestRuleStr = bestStr
}

func classifierInputs(bot *SwarmBot) [5]float64 {
	return [5]float64{
		clampF(bot.NearestPickupDist/400.0, 0, 1),
		clampF(bot.NearestDropoffDist/400.0, 0, 1),
		clampF(float64(bot.NeighborCount)/15.0, 0, 1),
		clampF(float64(bot.CarryingPkg+1)/2.0, 0, 1), // -1→0, 0+→0.5+
		clampF(bot.Speed/SwarmBotSpeed, 0, 1),
	}
}

func ruleMatches(rule ClassifierRule, inputs [5]float64) bool {
	for d := 0; d < 5; d++ {
		if inputs[d] < rule.Conditions[d][0] || inputs[d] > rule.Conditions[d][1] {
			return false
		}
	}
	return true
}

func applyClassifierAction(ss *SwarmState, bot *SwarmBot, action int) {
	switch action {
	case 0: // forward
		bot.Speed = SwarmBotSpeed
	case 1: // left
		bot.Speed = SwarmBotSpeed
		bot.Angle -= 0.2
	case 2: // right
		bot.Speed = SwarmBotSpeed
		bot.Angle += 0.2
	case 3: // seek pickup (random search)
		bot.Speed = SwarmBotSpeed * 1.1
		bot.Angle += (ss.Rng.Float64() - 0.5) * 0.3
	case 4: // seek dropoff (tighter turns)
		bot.Speed = SwarmBotSpeed * 0.9
		bot.Angle += (ss.Rng.Float64() - 0.5) * 0.15
	}
}

func classifierReward(bot *SwarmBot) float64 {
	reward := 0.0
	if bot.CarryingPkg >= 0 && bot.NearestDropoffDist < 80 {
		reward += 0.5
	}
	if bot.CarryingPkg < 0 && bot.NearestPickupDist < 60 {
		reward += 0.3
	}
	if bot.Speed > SwarmBotSpeed*0.5 {
		reward += 0.05
	}
	return reward
}

// EvolveClassifier evolves rule sets based on bot fitness.
func EvolveClassifier(ss *SwarmState, sortedIndices []int) {
	cs := ss.Classifier
	if cs == nil {
		return
	}

	n := len(ss.Bots)
	if len(cs.RuleSets) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 25 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 2

	parents := make([]BotRuleSet, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parents[i] = cloneRuleSet(cs.RuleSets[sortedIndices[i]])
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			continue
		}

		p := ss.Rng.Intn(parentCount)
		child := cloneRuleSet(parents[p])

		// Replace weak rules with mutations of strong rules
		for j := range child.Rules {
			if child.Rules[j].Strength < 0.2 && ss.Rng.Float64() < 0.3 {
				// Find a strong rule to mutate
				strongest := 0
				for k := 1; k < len(child.Rules); k++ {
					if child.Rules[k].Strength > child.Rules[strongest].Strength {
						strongest = k
					}
				}
				child.Rules[j] = mutateRule(ss, child.Rules[strongest])
			}

			// Regular mutation
			if ss.Rng.Float64() < 0.1 {
				child.Rules[j] = mutateRule(ss, child.Rules[j])
			}
		}

		cs.RuleSets[botIdx] = child
	}

	cs.Generation++
	logger.Info("LCS", "Gen %d: AvgStr=%.3f, BestStr=%.3f, AvgActive=%.1f",
		cs.Generation, cs.AvgRuleStrength, cs.BestRuleStr, cs.AvgActiveRules)
}

func mutateRule(ss *SwarmState, src ClassifierRule) ClassifierRule {
	r := src
	r.MatchCount = 0
	r.RewardSum = 0

	// Mutate conditions
	for d := 0; d < 5; d++ {
		if ss.Rng.Float64() < 0.3 {
			r.Conditions[d][0] += ss.Rng.NormFloat64() * 0.1
			r.Conditions[d][1] += ss.Rng.NormFloat64() * 0.1
			r.Conditions[d][0] = clampF(r.Conditions[d][0], 0, 1)
			r.Conditions[d][1] = clampF(r.Conditions[d][1], 0, 1)
			if r.Conditions[d][0] > r.Conditions[d][1] {
				r.Conditions[d][0], r.Conditions[d][1] = r.Conditions[d][1], r.Conditions[d][0]
			}
		}
	}

	// Possibly change action
	if ss.Rng.Float64() < 0.1 {
		r.Action = ss.Rng.Intn(numClassifierActions)
	}

	r.Strength = math.Max(src.Strength*0.8, 0.1)
	return r
}

func cloneRuleSet(src BotRuleSet) BotRuleSet {
	dst := BotRuleSet{
		Rules:      make([]ClassifierRule, len(src.Rules)),
		LastAction: src.LastAction,
	}
	copy(dst.Rules, src.Rules)
	return dst
}

// ClassifierAvgStrength returns the average rule strength.
func ClassifierAvgStrength(cs *ClassifierState) float64 {
	if cs == nil {
		return 0
	}
	return cs.AvgRuleStrength
}

// ClassifierBestRule returns the strongest rule strength.
func ClassifierBestRule(cs *ClassifierState) float64 {
	if cs == nil {
		return 0
	}
	return cs.BestRuleStr
}

// BotRuleCount returns the number of rules for a bot.
func BotRuleCount(cs *ClassifierState, botIdx int) int {
	if cs == nil || botIdx < 0 || botIdx >= len(cs.RuleSets) {
		return 0
	}
	return len(cs.RuleSets[botIdx].Rules)
}
