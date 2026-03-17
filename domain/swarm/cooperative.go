package swarm

import (
	"math"
	"math/rand"
)

// CooperativeConfig controls knowledge transfer between bots.
type CooperativeConfig struct {
	TeachRange    float64 // max distance for teaching (default 60)
	TeachRate     float64 // probability per tick of teaching attempt (default 0.01)
	LearnStrength float64 // how much the learner adopts (0-1, default 0.3)
	MinFitGap     float64 // teacher must be this much fitter than learner (default 10)
}

// DefaultCooperativeConfig returns sensible defaults.
func DefaultCooperativeConfig() CooperativeConfig {
	return CooperativeConfig{
		TeachRange:    60,
		TeachRate:     0.01,
		LearnStrength: 0.3,
		MinFitGap:     10,
	}
}

// TeachEvent records a knowledge transfer for visualization.
type TeachEvent struct {
	TeacherIdx int
	LearnerIdx int
	Tick       int
	FitGain    float64 // fitness improvement estimate
}

// CooperativeState tracks cooperative learning across the swarm.
type CooperativeState struct {
	Config       CooperativeConfig
	TeachEvents  []TeachEvent // recent events (for visualization)
	TotalTeaches int
	MaxEvents    int // ring buffer size for events (default 100)
}

// NewCooperativeState creates a cooperative learning state.
func NewCooperativeState() *CooperativeState {
	return &CooperativeState{
		Config:    DefaultCooperativeConfig(),
		MaxEvents: 100,
	}
}

// RunCooperativeLearning performs one tick of cooperative knowledge transfer.
// High-fitness bots "teach" nearby low-fitness bots by blending neural weights.
func RunCooperativeLearning(rng *rand.Rand, ss *SwarmState) {
	coop := ss.CoopState
	if coop == nil {
		return
	}
	cfg := &coop.Config
	n := len(ss.Bots)
	if n < 2 {
		return
	}

	for i := 0; i < n; i++ {
		if rng.Float64() >= cfg.TeachRate {
			continue
		}

		teacher := &ss.Bots[i]
		teacherFit := EvaluateGPFitness(teacher)

		// Find nearest bot within range
		bestJ := -1
		bestDist := math.MaxFloat64
		for j := 0; j < n; j++ {
			if j == i {
				continue
			}
			dx := ss.Bots[j].X - teacher.X
			dy := ss.Bots[j].Y - teacher.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < cfg.TeachRange && dist < bestDist {
				learnerFit := EvaluateGPFitness(&ss.Bots[j])
				if teacherFit-learnerFit >= cfg.MinFitGap {
					bestJ = j
					bestDist = dist
				}
			}
		}

		if bestJ < 0 {
			continue
		}

		// Transfer knowledge (blend neural weights)
		transferred := transferNeuroWeights(teacher, &ss.Bots[bestJ], cfg.LearnStrength)
		if !transferred {
			transferred = transferParamValues(teacher, &ss.Bots[bestJ], cfg.LearnStrength)
		}

		if transferred {
			coop.TotalTeaches++
			event := TeachEvent{
				TeacherIdx: i,
				LearnerIdx: bestJ,
				Tick:       ss.Tick,
				FitGain:    teacherFit - EvaluateGPFitness(&ss.Bots[bestJ]),
			}
			addTeachEvent(coop, event)
		}
	}
}

// transferNeuroWeights blends Brain weights from teacher to learner.
func transferNeuroWeights(teacher, learner *SwarmBot, strength float64) bool {
	if teacher.Brain == nil || learner.Brain == nil {
		return false
	}
	for w := range learner.Brain.Weights {
		learner.Brain.Weights[w] = learner.Brain.Weights[w]*(1-strength) +
			teacher.Brain.Weights[w]*strength
	}
	return true
}

// transferParamValues blends ParamValues as fallback.
func transferParamValues(teacher, learner *SwarmBot, strength float64) bool {
	transferred := false
	for p := 0; p < 26; p++ {
		diff := math.Abs(teacher.ParamValues[p] - learner.ParamValues[p])
		if diff > 0.01 {
			learner.ParamValues[p] = learner.ParamValues[p]*(1-strength) +
				teacher.ParamValues[p]*strength
			transferred = true
		}
	}
	return transferred
}

func addTeachEvent(coop *CooperativeState, event TeachEvent) {
	if len(coop.TeachEvents) >= coop.MaxEvents {
		coop.TeachEvents = coop.TeachEvents[1:]
	}
	coop.TeachEvents = append(coop.TeachEvents, event)
}

// RecentTeachCount returns how many teach events happened in the last N ticks.
func RecentTeachCount(coop *CooperativeState, currentTick, windowTicks int) int {
	if coop == nil {
		return 0
	}
	count := 0
	for _, e := range coop.TeachEvents {
		if currentTick-e.Tick <= windowTicks {
			count++
		}
	}
	return count
}
