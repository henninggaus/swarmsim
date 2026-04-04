package swarm

import (
	"math"
	"swarmsim/logger"
)

// TemporalMemoryState manages temporal pattern recognition.
// Bots record events with timestamps and detect periodic patterns.
// When a pattern is detected (e.g., resources appear every N ticks),
// bots anticipate and pre-position. Like migratory animals sensing seasons.
type TemporalMemoryState struct {
	Events    []TemporalEvent // shared event log
	Patterns  []TemporalPattern // detected periodic patterns
	MaxEvents int // max events stored (default 500)

	// Per-bot anticipation state
	Anticipations []BotAnticipation

	// Stats
	PatternsFound  int
	Anticipations_ int // bots currently anticipating
	PredictionHits int // correct predictions
	TotalPredictions int
}

// TemporalEvent is a timestamped observation.
type TemporalEvent struct {
	Tick     int
	EventType int // 0=resource_appeared, 1=cluster_formed, 2=high_activity, 3=low_activity
	X, Y     float64
	Strength float64
}

// TemporalPattern is a detected periodicity.
type TemporalPattern struct {
	EventType  int
	Period     int     // ticks between occurrences
	Phase      int     // offset within period
	Confidence float64 // how reliable this pattern is
	LastSeen   int     // tick of last occurrence
	HitCount   int     // how many times the pattern was confirmed
}

// BotAnticipation tracks what a bot expects to happen.
type BotAnticipation struct {
	ExpectingEvent bool
	EventType      int
	ExpectedTick   int
	TargetX        float64
	TargetY        float64
}

// Temporal event types
const (
	TEvtResource    = 0
	TEvtCluster     = 1
	TEvtHighActivity = 2
	TEvtLowActivity  = 3
)

// InitTemporalMemory sets up the temporal pattern recognition system.
func InitTemporalMemory(ss *SwarmState) {
	n := len(ss.Bots)
	tm := &TemporalMemoryState{
		Events:        make([]TemporalEvent, 0, 500),
		Patterns:      make([]TemporalPattern, 0, 20),
		MaxEvents:     500,
		Anticipations: make([]BotAnticipation, n),
	}

	ss.TemporalMemory = tm
	logger.Info("TMEM", "Initialisiert: Zeitgedaechtnis fuer %d Bots", n)
}

// ClearTemporalMemory disables the system.
func ClearTemporalMemory(ss *SwarmState) {
	ss.TemporalMemory = nil
	ss.TemporalMemoryOn = false
}

// TickTemporalMemory runs one tick of temporal pattern recognition.
func TickTemporalMemory(ss *SwarmState) {
	tm := ss.TemporalMemory
	if tm == nil {
		return
	}

	n := len(ss.Bots)
	if len(tm.Anticipations) != n {
		return
	}

	// Record current events
	recordTemporalEvents(ss, tm)

	// Detect patterns periodically
	if ss.Tick%50 == 0 && len(tm.Events) > 10 {
		detectPatterns(tm)
	}

	// Update bot anticipations
	anticipating := 0
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		ant := &tm.Anticipations[i]

		if ant.ExpectingEvent && ss.Tick >= ant.ExpectedTick-20 {
			// Move toward anticipated event location
			dx := ant.TargetX - bot.X
			dy := ant.TargetY - bot.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 5 {
				targetAngle := math.Atan2(dy, dx)
				diff := targetAngle - bot.Angle
				diff = WrapAngle(diff)
				bot.Angle += diff * 0.1
				bot.Speed *= 1.1
			}
			anticipating++

			// Check if prediction was correct
			if ss.Tick >= ant.ExpectedTick {
				checkPrediction(ss, tm, ant)
				ant.ExpectingEvent = false
			}

			// Anticipation color: gold
			bot.LEDColor = [3]uint8{255, 215, 0}
		}
	}

	// Assign new anticipations from patterns
	if ss.Tick%30 == 0 {
		assignAnticipations(ss, tm)
	}

	tm.Anticipations_ = anticipating
}

// recordTemporalEvents captures current swarm state as events.
func recordTemporalEvents(ss *SwarmState, tm *TemporalMemoryState) {
	if ss.Tick%10 != 0 {
		return
	}

	// Check for resource clusters
	resourceBots := 0
	avgRX, avgRY := 0.0, 0.0
	for i := range ss.Bots {
		if ss.Bots[i].NearestPickupDist < 50 {
			resourceBots++
			avgRX += ss.Bots[i].X
			avgRY += ss.Bots[i].Y
		}
	}

	if resourceBots > len(ss.Bots)/4 {
		if resourceBots > 0 {
			avgRX /= float64(resourceBots)
			avgRY /= float64(resourceBots)
		}
		addTemporalEvent(tm, TemporalEvent{
			Tick:      ss.Tick,
			EventType: TEvtResource,
			X:         avgRX,
			Y:         avgRY,
			Strength:  float64(resourceBots) / float64(len(ss.Bots)),
		})
	}

	// Check for clustering
	clustered := 0
	for i := range ss.Bots {
		if ss.Bots[i].NeighborCount > 6 {
			clustered++
		}
	}
	if clustered > len(ss.Bots)/3 {
		addTemporalEvent(tm, TemporalEvent{
			Tick:      ss.Tick,
			EventType: TEvtCluster,
			Strength:  float64(clustered) / float64(len(ss.Bots)),
		})
	}

	// Check activity level
	avgSpeed := 0.0
	for i := range ss.Bots {
		avgSpeed += ss.Bots[i].Speed
	}
	avgSpeed /= float64(len(ss.Bots))
	if avgSpeed > SwarmBotSpeed*1.2 {
		addTemporalEvent(tm, TemporalEvent{
			Tick:      ss.Tick,
			EventType: TEvtHighActivity,
			Strength:  avgSpeed / SwarmBotSpeed,
		})
	}
}

func addTemporalEvent(tm *TemporalMemoryState, evt TemporalEvent) {
	if len(tm.Events) >= tm.MaxEvents {
		// Remove oldest
		tm.Events = tm.Events[1:]
	}
	tm.Events = append(tm.Events, evt)
}

// detectPatterns finds periodicities in the event log.
func detectPatterns(tm *TemporalMemoryState) {
	// For each event type, check for periodic occurrence
	for evtType := 0; evtType <= 3; evtType++ {
		ticks := []int{}
		for _, e := range tm.Events {
			if e.EventType == evtType {
				ticks = append(ticks, e.Tick)
			}
		}

		if len(ticks) < 3 {
			continue
		}

		// Compute intervals
		intervals := make([]int, 0, len(ticks)-1)
		for i := 1; i < len(ticks); i++ {
			intervals = append(intervals, ticks[i]-ticks[i-1])
		}

		// Find most common interval (simple histogram)
		bestPeriod := 0
		bestCount := 0
		tolerance := 15

		for _, interval := range intervals {
			count := 0
			for _, other := range intervals {
				if abs(interval-other) <= tolerance {
					count++
				}
			}
			if count > bestCount {
				bestCount = count
				bestPeriod = interval
			}
		}

		if bestCount >= 2 && bestPeriod > 20 {
			confidence := float64(bestCount) / float64(len(intervals))

			// Check if we already have this pattern
			found := false
			for p := range tm.Patterns {
				if tm.Patterns[p].EventType == evtType && abs(tm.Patterns[p].Period-bestPeriod) < tolerance {
					tm.Patterns[p].Confidence = confidence
					tm.Patterns[p].HitCount++
					tm.Patterns[p].LastSeen = ticks[len(ticks)-1]
					found = true
					break
				}
			}

			if !found && len(tm.Patterns) < 20 {
				tm.Patterns = append(tm.Patterns, TemporalPattern{
					EventType:  evtType,
					Period:     bestPeriod,
					Phase:      ticks[len(ticks)-1] % bestPeriod,
					Confidence: confidence,
					LastSeen:   ticks[len(ticks)-1],
					HitCount:   1,
				})
				tm.PatternsFound++
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// assignAnticipations gives bots prediction-based targets.
func assignAnticipations(ss *SwarmState, tm *TemporalMemoryState) {
	for _, pat := range tm.Patterns {
		if pat.Confidence < 0.3 {
			continue
		}

		// When is the next expected occurrence?
		nextTick := pat.LastSeen + pat.Period
		if nextTick <= ss.Tick {
			nextTick += pat.Period
		}

		if nextTick-ss.Tick > 100 {
			continue // too far in the future
		}

		// Find the most recent event of this type for location
		var targetX, targetY float64
		for j := len(tm.Events) - 1; j >= 0; j-- {
			if tm.Events[j].EventType == pat.EventType {
				targetX = tm.Events[j].X
				targetY = tm.Events[j].Y
				break
			}
		}

		// Assign to some random bots
		for i := range ss.Bots {
			if tm.Anticipations[i].ExpectingEvent {
				continue
			}
			if ss.Rng.Float64() < 0.1*pat.Confidence {
				tm.Anticipations[i] = BotAnticipation{
					ExpectingEvent: true,
					EventType:      pat.EventType,
					ExpectedTick:   nextTick,
					TargetX:        targetX,
					TargetY:        targetY,
				}
			}
		}
	}
}

// checkPrediction validates whether a bot's prediction was correct.
func checkPrediction(ss *SwarmState, tm *TemporalMemoryState, ant *BotAnticipation) {
	tm.TotalPredictions++

	// Check if an event of the expected type occurred recently
	for j := len(tm.Events) - 1; j >= 0; j-- {
		e := tm.Events[j]
		if e.Tick < ant.ExpectedTick-30 {
			break
		}
		if e.EventType == ant.EventType {
			tm.PredictionHits++
			return
		}
	}
}

// TemporalPatternsFound returns how many periodic patterns were detected.
func TemporalPatternsFound(tm *TemporalMemoryState) int {
	if tm == nil {
		return 0
	}
	return tm.PatternsFound
}

// TemporalPredictionRate returns the prediction accuracy.
func TemporalPredictionRate(tm *TemporalMemoryState) float64 {
	if tm == nil || tm.TotalPredictions == 0 {
		return 0
	}
	return float64(tm.PredictionHits) / float64(tm.TotalPredictions)
}

// TemporalEventCount returns stored event count.
func TemporalEventCount(tm *TemporalMemoryState) int {
	if tm == nil {
		return 0
	}
	return len(tm.Events)
}
