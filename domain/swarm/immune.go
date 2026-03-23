package swarm

import (
	"math"
	"swarmsim/logger"
)

// ImmuneState manages a swarm immune system.
// Bots monitor their neighbors' behavior and detect anomalies.
// "Detector cells" learn normal behavior patterns and flag deviants.
// "Memory cells" remember past threats for faster future response.
type ImmuneState struct {
	DetectionRange float64 // sensing range for behavior monitoring (default 60)
	Threshold      float64 // anomaly threshold (default 2.0 std devs)
	MemoryDecay    float64 // threat memory decay (default 0.005)
	ResponseGain   float64 // immune response strength (default 0.3)
	MaxMemoryCells int     // max remembered threat patterns (default 50)

	// Per-bot immune state
	Cells []ImmuneCell

	// Population-level immune memory
	ThreatPatterns []ThreatPattern

	// Stats
	AnomalyCount int
	ResponseCount int
	AvgHealth    float64
}

// ImmuneCell is a bot's individual immune state.
type ImmuneCell struct {
	Health       float64 // 0-1, overall health score
	IsAnomalous  bool    // flagged as anomalous by neighbors
	AnomalyScore float64 // how anomalous this bot appears
	IsDetector   bool    // this bot acts as a detector cell
	DetectorAge  int     // how long this bot has been a detector

	// Behavioral signature: running average of key metrics
	AvgSpeed    float64
	AvgTurnRate float64
	AvgNeighbors float64
	AvgDelivery  float64
}

// ThreatPattern is a remembered anomalous behavior pattern.
type ThreatPattern struct {
	Signature [4]float64 // behavioral signature
	Strength  float64    // memory strength (decays over time)
	Tick      int        // when first detected
	Responses int        // how many times this pattern triggered response
}

// InitImmune sets up the swarm immune system.
func InitImmune(ss *SwarmState) {
	n := len(ss.Bots)
	is := &ImmuneState{
		DetectionRange: 60,
		Threshold:      2.0,
		MemoryDecay:    0.005,
		ResponseGain:   0.3,
		MaxMemoryCells: 50,
		Cells:          make([]ImmuneCell, n),
	}

	// Assign ~20% of bots as initial detectors
	for i := range is.Cells {
		is.Cells[i].Health = 1.0
		if ss.Rng.Float64() < 0.2 {
			is.Cells[i].IsDetector = true
		}
	}

	ss.Immune = is
	logger.Info("IMMUNE", "Initialisiert: %d Bots, %d%% Detektoren",
		n, 20)
}

// ClearImmune disables the immune system.
func ClearImmune(ss *SwarmState) {
	ss.Immune = nil
	ss.ImmuneOn = false
}

// TickImmune runs one tick of the immune system.
func TickImmune(ss *SwarmState) {
	is := ss.Immune
	if is == nil {
		return
	}

	n := len(ss.Bots)
	if len(is.Cells) != n {
		return
	}

	// Phase 1: Update behavioral signatures (rolling average)
	for i := range ss.Bots {
		cell := &is.Cells[i]
		alpha := 0.05 // smoothing factor
		cell.AvgSpeed = cell.AvgSpeed*(1-alpha) + ss.Bots[i].Speed*alpha
		turnRate := math.Abs(ss.Bots[i].Angle - cell.AvgTurnRate)
		cell.AvgTurnRate = cell.AvgTurnRate*(1-alpha) + turnRate*alpha
		cell.AvgNeighbors = cell.AvgNeighbors*(1-alpha) + float64(ss.Bots[i].NeighborCount)*alpha
		cell.AvgDelivery = cell.AvgDelivery*(1-alpha) + float64(ss.Bots[i].Stats.TotalDeliveries)*0.01*alpha
	}

	// Phase 2: Compute population statistics
	var meanSig [4]float64
	var stdSig [4]float64
	for i := range is.Cells {
		sig := botSignature(&is.Cells[i])
		for k := 0; k < 4; k++ {
			meanSig[k] += sig[k]
		}
	}
	fn := float64(n)
	for k := 0; k < 4; k++ {
		meanSig[k] /= fn
	}
	for i := range is.Cells {
		sig := botSignature(&is.Cells[i])
		for k := 0; k < 4; k++ {
			d := sig[k] - meanSig[k]
			stdSig[k] += d * d
		}
	}
	for k := 0; k < 4; k++ {
		stdSig[k] = math.Sqrt(stdSig[k] / fn)
		if stdSig[k] < 0.01 {
			stdSig[k] = 0.01
		}
	}

	// Phase 3: Detectors check neighbors for anomalies
	rangeSq := is.DetectionRange * is.DetectionRange
	is.AnomalyCount = 0

	for i := range ss.Bots {
		is.Cells[i].IsAnomalous = false
		is.Cells[i].AnomalyScore = 0
	}

	useSpatial := ss.Hash != nil

	for i := range ss.Bots {
		if !is.Cells[i].IsDetector {
			continue
		}
		is.Cells[i].DetectorAge++

		if useSpatial {
			// O(n·k): query only nearby bots via spatial hash
			nearIDs := ss.Hash.Query(ss.Bots[i].X, ss.Bots[i].Y, is.DetectionRange)
			for _, j := range nearIDs {
				if j == i || j >= n {
					continue
				}
				dx := ss.Bots[i].X - ss.Bots[j].X
				dy := ss.Bots[i].Y - ss.Bots[j].Y
				if dx*dx+dy*dy > rangeSq {
					continue
				}

				sig := botSignature(&is.Cells[j])
				score := 0.0
				for k := 0; k < 4; k++ {
					z := math.Abs(sig[k]-meanSig[k]) / stdSig[k]
					score += z
				}
				score /= 4.0

				is.Cells[j].AnomalyScore = math.Max(is.Cells[j].AnomalyScore, score)

				if score > is.Threshold {
					is.Cells[j].IsAnomalous = true
					is.AnomalyCount++
					storeThreatPattern(is, sig, ss.Tick)
				}
			}
		} else {
			// Fallback: brute-force O(n²)
			for j := range ss.Bots {
				if i == j {
					continue
				}
				dx := ss.Bots[i].X - ss.Bots[j].X
				dy := ss.Bots[i].Y - ss.Bots[j].Y
				if dx*dx+dy*dy > rangeSq {
					continue
				}

				sig := botSignature(&is.Cells[j])
				score := 0.0
				for k := 0; k < 4; k++ {
					z := math.Abs(sig[k]-meanSig[k]) / stdSig[k]
					score += z
				}
				score /= 4.0

				is.Cells[j].AnomalyScore = math.Max(is.Cells[j].AnomalyScore, score)

				if score > is.Threshold {
					is.Cells[j].IsAnomalous = true
					is.AnomalyCount++
					storeThreatPattern(is, sig, ss.Tick)
				}
			}
		}
	}

	// Phase 4: Immune response — slow down anomalous bots
	is.ResponseCount = 0
	for i := range ss.Bots {
		cell := &is.Cells[i]
		if cell.IsAnomalous {
			// Reduce speed
			ss.Bots[i].Speed *= (1 - is.ResponseGain)
			cell.Health -= 0.01
			if cell.Health < 0 {
				cell.Health = 0
			}
			is.ResponseCount++

			// Red LED for flagged bots
			ss.Bots[i].LEDColor = [3]uint8{255, 50, 50}
		} else {
			// Recover health
			cell.Health += 0.002
			if cell.Health > 1 {
				cell.Health = 1
			}
		}
	}

	// Phase 5: Check against threat memory for faster detection
	if ss.Tick%10 == 0 {
		checkThreatMemory(ss, is, meanSig, stdSig)
	}

	// Phase 6: Decay threat memories
	alive := 0
	for j := range is.ThreatPatterns {
		is.ThreatPatterns[j].Strength -= is.MemoryDecay
		if is.ThreatPatterns[j].Strength > 0 {
			alive++
		}
	}
	if alive < len(is.ThreatPatterns) {
		filtered := make([]ThreatPattern, 0, alive)
		for _, tp := range is.ThreatPatterns {
			if tp.Strength > 0 {
				filtered = append(filtered, tp)
			}
		}
		is.ThreatPatterns = filtered
	}

	// Phase 7: Rotate detectors occasionally
	if ss.Tick%500 == 0 {
		rotateDetectors(ss, is)
	}

	// Stats
	totalHealth := 0.0
	for _, c := range is.Cells {
		totalHealth += c.Health
	}
	is.AvgHealth = totalHealth / fn
}

// botSignature returns a behavioral signature vector.
func botSignature(cell *ImmuneCell) [4]float64 {
	return [4]float64{
		cell.AvgSpeed,
		cell.AvgTurnRate,
		cell.AvgNeighbors,
		cell.AvgDelivery,
	}
}

// storeThreatPattern adds a new threat pattern to immune memory.
func storeThreatPattern(is *ImmuneState, sig [4]float64, tick int) {
	// Check if similar pattern already exists
	for j := range is.ThreatPatterns {
		dist := 0.0
		for k := 0; k < 4; k++ {
			d := is.ThreatPatterns[j].Signature[k] - sig[k]
			dist += d * d
		}
		if dist < 0.1 {
			is.ThreatPatterns[j].Strength = math.Min(is.ThreatPatterns[j].Strength+0.1, 1.0)
			is.ThreatPatterns[j].Responses++
			return
		}
	}

	// New pattern
	if len(is.ThreatPatterns) >= is.MaxMemoryCells {
		// Evict weakest
		weakest := 0
		for j := 1; j < len(is.ThreatPatterns); j++ {
			if is.ThreatPatterns[j].Strength < is.ThreatPatterns[weakest].Strength {
				weakest = j
			}
		}
		is.ThreatPatterns[weakest] = ThreatPattern{
			Signature: sig,
			Strength:  0.5,
			Tick:      tick,
		}
	} else {
		is.ThreatPatterns = append(is.ThreatPatterns, ThreatPattern{
			Signature: sig,
			Strength:  0.5,
			Tick:      tick,
		})
	}
}

// checkThreatMemory compares current bots against remembered threats.
func checkThreatMemory(ss *SwarmState, is *ImmuneState, meanSig, stdSig [4]float64) {
	for i := range ss.Bots {
		sig := botSignature(&is.Cells[i])
		for j := range is.ThreatPatterns {
			dist := 0.0
			for k := 0; k < 4; k++ {
				d := sig[k] - is.ThreatPatterns[j].Signature[k]
				dist += d * d
			}
			if dist < 0.2 && is.ThreatPatterns[j].Strength > 0.3 {
				is.Cells[i].AnomalyScore = math.Max(is.Cells[i].AnomalyScore,
					is.ThreatPatterns[j].Strength*is.Threshold)
			}
		}
	}
}

// rotateDetectors reassigns which bots are detectors.
func rotateDetectors(ss *SwarmState, is *ImmuneState) {
	for i := range is.Cells {
		if is.Cells[i].IsDetector && is.Cells[i].DetectorAge > 500 {
			is.Cells[i].IsDetector = false
			is.Cells[i].DetectorAge = 0
		}
	}
	// Ensure ~20% are detectors
	detCount := 0
	for _, c := range is.Cells {
		if c.IsDetector {
			detCount++
		}
	}
	target := len(is.Cells) * 20 / 100
	for detCount < target {
		idx := ss.Rng.Intn(len(is.Cells))
		if !is.Cells[idx].IsDetector {
			is.Cells[idx].IsDetector = true
			detCount++
		}
	}
}

// ImmuneAnomalyCount returns the number of anomalous bots.
func ImmuneAnomalyCount(is *ImmuneState) int {
	if is == nil {
		return 0
	}
	return is.AnomalyCount
}

// ImmuneAvgHealth returns average swarm health.
func ImmuneAvgHealth(is *ImmuneState) float64 {
	if is == nil {
		return 0
	}
	return is.AvgHealth
}

// ImmuneThreatCount returns number of remembered threat patterns.
func ImmuneThreatCount(is *ImmuneState) int {
	if is == nil {
		return 0
	}
	return len(is.ThreatPatterns)
}
