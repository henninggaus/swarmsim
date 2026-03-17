package swarm

import (
	"math"
	"swarmsim/logger"
)

// LanguageEvo manages the emergent communication protocol evolution.
// Instead of predefined message types, bots evolve continuous signal
// vectors. Receivers evolve to interpret these signals. Over generations,
// shared "meaning" emerges — a language created by evolution.
type LanguageEvo struct {
	SignalSize    int     // number of float values per signal (default 4)
	BroadcastRange float64 // max range for signal reception (default 60)
	DecayRate    float64 // signal strength decay per tick (default 0.1)
	MaxSignals   int     // max active signals per tick (default 200)

	// Per-bot evolved encoding/decoding networks
	Encoders []SignalEncoder // per-bot: situation → signal
	Decoders []SignalDecoder // per-bot: signal → action bias

	// Active signals this tick
	Signals  []LanguageSignal
	PrevSignals []LanguageSignal // signals from previous tick

	// Stats & visualization
	Generation       int
	SignalHistory    []SignalCluster // clusters of similar signals over time
	MutualInfoScore  float64         // mutual information between signal and context
	VocabularySize   int             // number of distinct signal clusters
}

// SignalEncoder is a small neural network that maps bot state to a signal.
// Architecture: 6 inputs → 4 hidden (tanh) → SignalSize outputs (tanh)
type SignalEncoder struct {
	WeightsIH [6 * 4]float64  // input to hidden
	WeightsHO [4 * 8]float64  // hidden to output (max SignalSize=8)
}

// SignalDecoder is a small network that maps received signals to action biases.
// Architecture: SignalSize inputs → 4 hidden → 4 outputs (bias for actions)
type SignalDecoder struct {
	WeightsIH [8 * 4]float64  // input to hidden (max SignalSize=8)
	WeightsHO [4 * 4]float64  // hidden to output biases
}

// LanguageSignal is a broadcast signal from one bot.
type LanguageSignal struct {
	SenderIdx int
	X, Y      float64   // sender position
	Values    []float64 // signal vector
	Context   int       // sender's context (0=exploring, 1=carrying, 2=near_pickup, 3=near_dropoff)
	Age       int       // ticks since broadcast
}

// SignalCluster represents a group of similar signals (for vocabulary analysis).
type SignalCluster struct {
	Centroid []float64
	Count    int
	Context  int // most common context in this cluster
}

const (
	langEncoderInputs  = 6
	langEncoderHidden  = 4
	langMaxSignalSize  = 8
	langDecoderOutputs = 4
)

// InitLanguageEvo sets up the emergent language system.
func InitLanguageEvo(ss *SwarmState, signalSize int) {
	if signalSize < 2 {
		signalSize = 2
	}
	if signalSize > langMaxSignalSize {
		signalSize = langMaxSignalSize
	}

	n := len(ss.Bots)
	le := &LanguageEvo{
		SignalSize:     signalSize,
		BroadcastRange: 60,
		DecayRate:      0.1,
		MaxSignals:     200,
		Encoders:       make([]SignalEncoder, n),
		Decoders:       make([]SignalDecoder, n),
	}

	// Initialize random encoder/decoder weights
	for i := 0; i < n; i++ {
		randomizeEncoder(ss, &le.Encoders[i])
		randomizeDecoder(ss, &le.Decoders[i])
	}

	ss.LanguageEvo = le
	logger.Info("LANGUAGE", "Initialisiert: %d Bots, Signal-Groesse=%d", n, signalSize)
}

func randomizeEncoder(ss *SwarmState, enc *SignalEncoder) {
	for w := range enc.WeightsIH {
		enc.WeightsIH[w] = (ss.Rng.Float64() - 0.5) * 1.0
	}
	for w := range enc.WeightsHO {
		enc.WeightsHO[w] = (ss.Rng.Float64() - 0.5) * 1.0
	}
}

func randomizeDecoder(ss *SwarmState, dec *SignalDecoder) {
	for w := range dec.WeightsIH {
		dec.WeightsIH[w] = (ss.Rng.Float64() - 0.5) * 1.0
	}
	for w := range dec.WeightsHO {
		dec.WeightsHO[w] = (ss.Rng.Float64() - 0.5) * 1.0
	}
}

// ClearLanguageEvo disables the language evolution system.
func ClearLanguageEvo(ss *SwarmState) {
	ss.LanguageEvo = nil
	ss.LanguageEvoOn = false
}

// EncodeSignal generates a signal from a bot's current state using its encoder network.
func EncodeSignal(le *LanguageEvo, enc *SignalEncoder, bot *SwarmBot, ss *SwarmState) []float64 {
	if le == nil || enc == nil {
		return nil
	}

	// Build encoder input: [carry, near_dist, p_dist, d_dist, neighbors, speed]
	var input [langEncoderInputs]float64
	if bot.CarryingPkg >= 0 {
		input[0] = 1.0
	}
	nd := bot.NearestDist
	if nd > 200 {
		nd = 200
	}
	input[1] = nd / 200.0

	pd := bot.NearestPickupDist
	if pd > 500 {
		pd = 500
	}
	input[2] = pd / 500.0

	dd := bot.NearestDropoffDist
	if dd > 500 {
		dd = 500
	}
	input[3] = dd / 500.0

	nc := float64(bot.NeighborCount)
	if nc > 10 {
		nc = 10
	}
	input[4] = nc / 10.0
	input[5] = bot.Speed / SwarmBotSpeed

	// Forward pass: input → hidden (tanh)
	var hidden [langEncoderHidden]float64
	for h := 0; h < langEncoderHidden; h++ {
		sum := 0.0
		for inp := 0; inp < langEncoderInputs; inp++ {
			sum += input[inp] * enc.WeightsIH[inp*langEncoderHidden+h]
		}
		hidden[h] = math.Tanh(sum)
	}

	// Hidden → output (tanh, produces signal values in [-1, 1])
	signal := make([]float64, le.SignalSize)
	for o := 0; o < le.SignalSize; o++ {
		sum := 0.0
		for h := 0; h < langEncoderHidden; h++ {
			sum += hidden[h] * enc.WeightsHO[h*langMaxSignalSize+o]
		}
		signal[o] = math.Tanh(sum)
	}

	return signal
}

// DecodeSignal interprets a received signal and returns action biases.
func DecodeSignal(le *LanguageEvo, dec *SignalDecoder, signal []float64) [langDecoderOutputs]float64 {
	var output [langDecoderOutputs]float64
	if le == nil || dec == nil || len(signal) == 0 {
		return output
	}

	// Forward pass: signal → hidden (tanh)
	var hidden [langEncoderHidden]float64
	for h := 0; h < langEncoderHidden; h++ {
		sum := 0.0
		for inp := 0; inp < le.SignalSize && inp < langMaxSignalSize; inp++ {
			sum += signal[inp] * dec.WeightsIH[inp*langEncoderHidden+h]
		}
		hidden[h] = math.Tanh(sum)
	}

	// Hidden → output biases
	for o := 0; o < langDecoderOutputs; o++ {
		sum := 0.0
		for h := 0; h < langEncoderHidden; h++ {
			sum += hidden[h] * dec.WeightsHO[h*langDecoderOutputs+o]
		}
		output[o] = math.Tanh(sum)
	}

	return output
}

// BotContext returns the contextual state of a bot (for signal analysis).
func BotContext(bot *SwarmBot) int {
	if bot.CarryingPkg >= 0 {
		return 1 // carrying
	}
	if bot.NearestPickupDist < 50 {
		return 2 // near pickup
	}
	if bot.NearestDropoffDist < 50 {
		return 3 // near dropoff
	}
	return 0 // exploring
}

// TickLanguageEvo runs one tick of the language evolution system.
func TickLanguageEvo(ss *SwarmState) {
	le := ss.LanguageEvo
	if le == nil {
		return
	}

	// Age and remove old signals
	le.PrevSignals = le.Signals
	le.Signals = nil

	// Each bot broadcasts a signal
	for i := range ss.Bots {
		if i >= len(le.Encoders) {
			break
		}

		// Only broadcast every few ticks (reduce noise)
		if ss.Tick%3 != i%3 {
			continue
		}

		signal := EncodeSignal(le, &le.Encoders[i], &ss.Bots[i], ss)
		if signal == nil {
			continue
		}

		le.Signals = append(le.Signals, LanguageSignal{
			SenderIdx: i,
			X:         ss.Bots[i].X,
			Y:         ss.Bots[i].Y,
			Values:    signal,
			Context:   BotContext(&ss.Bots[i]),
			Age:       0,
		})

		if len(le.Signals) >= le.MaxSignals {
			break
		}
	}

	// Each bot receives nearby signals and applies decoder
	for i := range ss.Bots {
		if i >= len(le.Decoders) {
			break
		}

		// Find strongest nearby signal
		var bestSignal []float64
		bestDist := le.BroadcastRange

		for _, sig := range le.Signals {
			if sig.SenderIdx == i {
				continue // don't listen to yourself
			}
			dx := ss.Bots[i].X - sig.X
			dy := ss.Bots[i].Y - sig.Y
			d := math.Sqrt(dx*dx + dy*dy)
			if d < bestDist {
				bestDist = d
				bestSignal = sig.Values
			}
		}

		if bestSignal == nil {
			continue
		}

		// Decode signal into action biases
		biases := DecodeSignal(le, &le.Decoders[i], bestSignal)

		// Apply biases to bot behavior:
		// [0] speed bias, [1] turn bias, [2] pickup bias, [3] goto-dropoff bias
		ss.Bots[i].Speed += biases[0] * 0.3 * SwarmBotSpeed
		if ss.Bots[i].Speed < 0 {
			ss.Bots[i].Speed = 0
		}
		ss.Bots[i].Angle += biases[1] * 0.15

		// Visual: encode signal as LED color
		if bestSignal != nil && len(bestSignal) >= 3 {
			r := uint8(128 + bestSignal[0]*127)
			g := uint8(128 + bestSignal[1]*127)
			b := uint8(128 + bestSignal[2]*127)
			ss.Bots[i].LEDColor = [3]uint8{r, g, b}
		}
	}
}

// EvolveLanguage evolves encoder/decoder networks alongside the main evolution.
func EvolveLanguage(ss *SwarmState, sortedIndices []int) {
	le := ss.LanguageEvo
	if le == nil {
		return
	}

	n := len(ss.Bots)
	if len(le.Encoders) != n || len(le.Decoders) != n {
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
	parentEnc := make([]SignalEncoder, parentCount)
	parentDec := make([]SignalDecoder, parentCount)
	for i := 0; i < parentCount && i < len(sortedIndices); i++ {
		parentEnc[i] = le.Encoders[sortedIndices[i]]
		parentDec[i] = le.Decoders[sortedIndices[i]]
	}

	for rank, botIdx := range sortedIndices {
		if rank < eliteCount {
			le.Encoders[botIdx] = parentEnc[rank]
			le.Decoders[botIdx] = parentDec[rank]
		} else if rank >= n-freshCount {
			randomizeEncoder(ss, &le.Encoders[botIdx])
			randomizeDecoder(ss, &le.Decoders[botIdx])
		} else {
			p1 := ss.Rng.Intn(parentCount)
			p2 := ss.Rng.Intn(parentCount)
			crossoverEncoder(ss, &le.Encoders[botIdx], &parentEnc[p1], &parentEnc[p2])
			crossoverDecoder(ss, &le.Decoders[botIdx], &parentDec[p1], &parentDec[p2])
		}
	}

	// Update vocabulary analysis
	le.VocabularySize = analyzeVocabulary(le)
	le.Generation++

	logger.Info("LANGUAGE", "Gen %d: Vokabular=%d Signale, MutualInfo=%.3f",
		le.Generation, le.VocabularySize, le.MutualInfoScore)
}

func crossoverEncoder(ss *SwarmState, child, p1, p2 *SignalEncoder) {
	for w := range child.WeightsIH {
		if ss.Rng.Float64() < 0.5 {
			child.WeightsIH[w] = p1.WeightsIH[w]
		} else {
			child.WeightsIH[w] = p2.WeightsIH[w]
		}
		if ss.Rng.Float64() < 0.1 {
			child.WeightsIH[w] += ss.Rng.NormFloat64() * 0.2
		}
	}
	for w := range child.WeightsHO {
		if ss.Rng.Float64() < 0.5 {
			child.WeightsHO[w] = p1.WeightsHO[w]
		} else {
			child.WeightsHO[w] = p2.WeightsHO[w]
		}
		if ss.Rng.Float64() < 0.1 {
			child.WeightsHO[w] += ss.Rng.NormFloat64() * 0.2
		}
	}
}

func crossoverDecoder(ss *SwarmState, child, p1, p2 *SignalDecoder) {
	for w := range child.WeightsIH {
		if ss.Rng.Float64() < 0.5 {
			child.WeightsIH[w] = p1.WeightsIH[w]
		} else {
			child.WeightsIH[w] = p2.WeightsIH[w]
		}
		if ss.Rng.Float64() < 0.1 {
			child.WeightsIH[w] += ss.Rng.NormFloat64() * 0.2
		}
	}
	for w := range child.WeightsHO {
		if ss.Rng.Float64() < 0.5 {
			child.WeightsHO[w] = p1.WeightsHO[w]
		} else {
			child.WeightsHO[w] = p2.WeightsHO[w]
		}
		if ss.Rng.Float64() < 0.1 {
			child.WeightsHO[w] += ss.Rng.NormFloat64() * 0.2
		}
	}
}

// analyzeVocabulary counts distinct signal clusters using simple k-means-like binning.
func analyzeVocabulary(le *LanguageEvo) int {
	if le == nil || len(le.Signals) < 2 {
		return 0
	}

	// Simple quantization: bin each signal dimension into 3 levels
	bins := make(map[string]int)
	for _, sig := range le.Signals {
		key := ""
		for _, v := range sig.Values {
			if v < -0.3 {
				key += "L"
			} else if v > 0.3 {
				key += "H"
			} else {
				key += "M"
			}
		}
		bins[key]++
	}

	return len(bins)
}

// SignalSimilarity computes cosine similarity between two signal vectors.
func SignalSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	dot, magA, magB := 0.0, 0.0, 0.0
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// LanguageSignalCount returns the number of active signals.
func LanguageSignalCount(le *LanguageEvo) int {
	if le == nil {
		return 0
	}
	return len(le.Signals)
}
