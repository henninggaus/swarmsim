package swarm

import (
	"math"
	"swarmsim/logger"
)

// LanguageState manages emergent communication in the swarm.
// Each bot can broadcast a symbol (0-7, a 3-bit message). Over generations,
// bots evolve associations between symbols and meanings (contexts).
// Successful communication — where a receiver acts beneficially on a
// message — reinforces the shared vocabulary.
type LanguageState struct {
	Vocabs []BotVocab // per-bot vocabularies

	// Symbol usage statistics
	SymbolFreq    [8]int     // how often each symbol is broadcast
	SymbolSuccess [8]float64 // average reward when symbol was used
	SharedMeaning float64    // how much bots agree on symbol meanings (0-1)
	Generation    int
}

// BotVocab holds a bot's communication mappings.
type BotVocab struct {
	// Encoding: context → which symbol to send (8 contexts × 8 symbols)
	// Each row is a context, values are preferences for each symbol
	Encode [8][8]float64

	// Decoding: symbol → what action bias to apply
	// Each row is a received symbol, values are action biases
	Decode [8][4]float64 // 4 actions: seek-pickup, seek-dropoff, flee, cluster

	CurrentSymbol  int     // what this bot is currently broadcasting
	LastReceived   int     // last symbol received from neighbor
	CommSuccess    float64 // accumulated communication success
	BroadcastRange float64 // how far the signal reaches
}

// Communication contexts
const (
	CtxFoundFood    = 0 // near a pickup
	CtxCarrying     = 1 // carrying a package
	CtxNearDropoff  = 2 // near a dropoff
	CtxCrowded      = 3 // many neighbors
	CtxAlone        = 4 // few neighbors
	CtxDanger       = 5 // low fitness area
	CtxExploring    = 6 // far from others, moving
	CtxIdle         = 7 // default/no special context
)

// InitLanguage sets up the emergent communication system.
func InitLanguage(ss *SwarmState) {
	n := len(ss.Bots)
	ls := &LanguageState{
		Vocabs: make([]BotVocab, n),
	}

	for i := 0; i < n; i++ {
		v := &ls.Vocabs[i]
		v.BroadcastRange = 80.0

		// Random initial vocabularies
		for ctx := 0; ctx < 8; ctx++ {
			for sym := 0; sym < 8; sym++ {
				v.Encode[ctx][sym] = ss.Rng.Float64() * 0.5
			}
		}
		for sym := 0; sym < 8; sym++ {
			for act := 0; act < 4; act++ {
				v.Decode[sym][act] = (ss.Rng.Float64() - 0.5) * 0.5
			}
		}
	}

	ss.Language = ls
	logger.Info("LANG", "Initialisiert: %d Bots mit 8-Symbol Vokabular", n)
}

// ClearLanguage disables the language system.
func ClearLanguage(ss *SwarmState) {
	ss.Language = nil
	ss.LanguageOn = false
}

// TickLanguage runs one tick of the communication system.
func TickLanguage(ss *SwarmState) {
	ls := ss.Language
	if ls == nil {
		return
	}

	n := len(ss.Bots)
	if len(ls.Vocabs) != n {
		return
	}

	// Phase 1: Each bot determines context and broadcasts a symbol
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		vocab := &ls.Vocabs[i]

		ctx := determineContext(bot)
		symbol := chooseSymbol(ss, vocab, ctx)
		vocab.CurrentSymbol = symbol
		ls.SymbolFreq[symbol]++
	}

	// Phase 2: Each bot receives symbols from neighbors and reacts
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		vocab := &ls.Vocabs[i]

		// Find nearest broadcasting neighbor
		bestDist := vocab.BroadcastRange
		bestSymbol := -1
		for j := range ss.Bots {
			if i == j {
				continue
			}
			dx := ss.Bots[j].X - bot.X
			dy := ss.Bots[j].Y - bot.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < bestDist {
				bestDist = dist
				bestSymbol = ls.Vocabs[j].CurrentSymbol
			}
		}

		if bestSymbol >= 0 && bestSymbol < 8 {
			vocab.LastReceived = bestSymbol
			applySymbolAction(ss, bot, vocab, bestSymbol)
		}
	}

	// Update shared meaning metric
	updateLanguageStats(ls)
}

// determineContext figures out the bot's current situation.
func determineContext(bot *SwarmBot) int {
	if bot.NearestPickupDist < 60 && bot.CarryingPkg < 0 {
		return CtxFoundFood
	}
	if bot.CarryingPkg >= 0 {
		if bot.NearestDropoffDist < 80 {
			return CtxNearDropoff
		}
		return CtxCarrying
	}
	if bot.NeighborCount > 6 {
		return CtxCrowded
	}
	if bot.NeighborCount < 2 {
		return CtxAlone
	}
	if bot.Speed > SwarmBotSpeed*0.8 {
		return CtxExploring
	}
	return CtxIdle
}

// chooseSymbol selects a symbol based on context using softmax.
func chooseSymbol(ss *SwarmState, vocab *BotVocab, ctx int) int {
	// Softmax selection from encode preferences
	maxVal := vocab.Encode[ctx][0]
	for s := 1; s < 8; s++ {
		if vocab.Encode[ctx][s] > maxVal {
			maxVal = vocab.Encode[ctx][s]
		}
	}

	sum := 0.0
	probs := [8]float64{}
	for s := 0; s < 8; s++ {
		probs[s] = math.Exp(vocab.Encode[ctx][s] - maxVal)
		sum += probs[s]
	}

	r := ss.Rng.Float64() * sum
	cumul := 0.0
	for s := 0; s < 8; s++ {
		cumul += probs[s]
		if cumul >= r {
			return s
		}
	}
	return 7
}

// applySymbolAction interprets a received symbol and adjusts behavior.
func applySymbolAction(ss *SwarmState, bot *SwarmBot, vocab *BotVocab, symbol int) {
	decode := vocab.Decode[symbol]
	strength := 0.15

	// Action 0: seek-pickup bias
	if decode[0] > 0.1 && bot.CarryingPkg < 0 {
		bot.Speed *= 1.0 + decode[0]*strength
	}

	// Action 1: seek-dropoff bias
	if decode[1] > 0.1 && bot.CarryingPkg >= 0 {
		bot.Speed *= 1.0 + decode[1]*strength
	}

	// Action 2: flee (turn away)
	if decode[2] > 0.2 {
		bot.Angle += decode[2] * strength * 0.5
	}

	// Action 3: cluster (slow down to stay near)
	if decode[3] > 0.1 {
		bot.Speed *= 1.0 - decode[3]*strength*0.3
	}

	// Color based on received symbol
	hue := float64(symbol) / 8.0 * 2 * math.Pi
	r := uint8(128 + math.Sin(hue)*127)
	g := uint8(128 + math.Sin(hue+2.094)*127)
	b := uint8(128 + math.Sin(hue+4.189)*127)
	bot.LEDColor = [3]uint8{r, g, b}
}

// EvolveLanguage evolves vocabularies based on communication success.
func EvolveSymbolLanguage(ss *SwarmState, sortedIndices []int) {
	ls := ss.Language
	if ls == nil {
		return
	}

	n := len(ss.Bots)
	if len(ls.Vocabs) != n || len(sortedIndices) != n {
		return
	}

	parentCount := n * 25 / 100
	if parentCount < 2 {
		parentCount = 2
	}
	eliteCount := 2

	parents := make([]BotVocab, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parents[i] = ls.Vocabs[sortedIndices[i]]
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			continue
		}

		p := ss.Rng.Intn(parentCount)
		child := parents[p]
		child.CommSuccess = 0

		// Mutate encode table
		for ctx := 0; ctx < 8; ctx++ {
			for sym := 0; sym < 8; sym++ {
				if ss.Rng.Float64() < 0.1 {
					child.Encode[ctx][sym] += ss.Rng.NormFloat64() * 0.2
				}
			}
		}

		// Mutate decode table
		for sym := 0; sym < 8; sym++ {
			for act := 0; act < 4; act++ {
				if ss.Rng.Float64() < 0.1 {
					child.Decode[sym][act] += ss.Rng.NormFloat64() * 0.2
					child.Decode[sym][act] = clampF(child.Decode[sym][act], -2, 2)
				}
			}
		}

		ls.Vocabs[botIdx] = child
	}

	ls.SymbolFreq = [8]int{}
	ls.SymbolSuccess = [8]float64{}
	ls.Generation++

	logger.Info("LANG", "Gen %d: SharedMeaning=%.3f", ls.Generation, ls.SharedMeaning)
}

// updateLanguageStats computes how much bots agree on symbol usage.
func updateLanguageStats(ls *LanguageState) {
	n := len(ls.Vocabs)
	if n < 2 {
		return
	}

	// Measure agreement: for each symbol, how concentrated is usage across contexts?
	totalAgreement := 0.0
	for sym := 0; sym < 8; sym++ {
		// Count which context most commonly maps to this symbol
		contextVotes := [8]int{}
		for _, v := range ls.Vocabs {
			bestCtx := 0
			bestVal := v.Encode[0][sym]
			for ctx := 1; ctx < 8; ctx++ {
				if v.Encode[ctx][sym] > bestVal {
					bestVal = v.Encode[ctx][sym]
					bestCtx = ctx
				}
			}
			contextVotes[bestCtx]++
		}

		// Agreement = max votes / total
		maxVotes := 0
		for _, votes := range contextVotes {
			if votes > maxVotes {
				maxVotes = votes
			}
		}
		totalAgreement += float64(maxVotes) / float64(n)
	}

	ls.SharedMeaning = totalAgreement / 8.0
}

// SharedMeaning returns how much the swarm agrees on symbol meanings.
func SharedMeaning(ls *LanguageState) float64 {
	if ls == nil {
		return 0
	}
	return ls.SharedMeaning
}

// BotCurrentSymbol returns what symbol a bot is broadcasting.
func BotCurrentSymbol(ls *LanguageState, botIdx int) int {
	if ls == nil || botIdx < 0 || botIdx >= len(ls.Vocabs) {
		return -1
	}
	return ls.Vocabs[botIdx].CurrentSymbol
}
