package swarm

import "math"

// TimelapseStats aggregates simulation statistics over time windows.
type TimelapseStats struct {
	Windows      []TimeWindow
	WindowSize   int // ticks per window (default 500)
	MaxWindows   int // max stored windows (default 200)
	CurrentStart int // tick when current window started
	Current      TimeWindowAccum
}

// TimeWindow stores aggregated stats for one time window.
type TimeWindow struct {
	StartTick    int
	EndTick      int
	AvgFitness   float64
	MaxFitness   float64
	MinFitness   float64
	TotalDeliveries int
	TotalPickups    int
	AvgSpeed     float64
	AvgNeighbors float64
	BotsCrashed  int
	Diversity    float64
	MsgCount     int
}

// TimeWindowAccum accumulates stats during a window.
type TimeWindowAccum struct {
	FitnessSum   float64
	FitnessMax   float64
	FitnessMin   float64
	SpeedSum     float64
	NeighborSum  float64
	Deliveries   int
	Pickups      int
	Crashes      int
	MsgCount     int
	Samples      int
}

// NewTimelapseStats creates a timelapse tracker with defaults.
func NewTimelapseStats() *TimelapseStats {
	return &TimelapseStats{
		WindowSize: 500,
		MaxWindows: 200,
		Current:    TimeWindowAccum{FitnessMin: math.MaxFloat64},
	}
}

// TickTimelapse samples the current simulation state.
func TickTimelapse(ts *TimelapseStats, ss *SwarmState) {
	if ts == nil {
		return
	}

	// Sample current state
	n := len(ss.Bots)
	if n == 0 {
		return
	}

	fitSum, fitMax, fitMin := 0.0, -math.MaxFloat64, math.MaxFloat64
	speedSum, neighborSum := 0.0, 0.0
	for i := range ss.Bots {
		f := ss.Bots[i].Fitness
		fitSum += f
		if f > fitMax {
			fitMax = f
		}
		if f < fitMin {
			fitMin = f
		}
		speedSum += ss.Bots[i].Speed
		neighborSum += float64(ss.Bots[i].NeighborCount)
	}

	ts.Current.FitnessSum += fitSum / float64(n)
	if fitMax > ts.Current.FitnessMax {
		ts.Current.FitnessMax = fitMax
	}
	if fitMin < ts.Current.FitnessMin {
		ts.Current.FitnessMin = fitMin
	}
	ts.Current.SpeedSum += speedSum / float64(n)
	ts.Current.NeighborSum += neighborSum / float64(n)
	ts.Current.Deliveries += ss.DeliveryStats.CorrectDelivered
	ts.Current.Pickups += ss.DeliveryStats.TotalDelivered
	ts.Current.MsgCount += ss.BroadcastCount
	ts.Current.Crashes += ss.CollisionCount
	ts.Current.Samples++

	// Check if window is complete
	if ss.Tick-ts.CurrentStart >= ts.WindowSize && ts.Current.Samples > 0 {
		window := finalizeWindow(ts, ss.Tick)
		ts.Windows = append(ts.Windows, window)
		if len(ts.Windows) > ts.MaxWindows {
			ts.Windows = ts.Windows[1:]
		}
		ts.CurrentStart = ss.Tick
		ts.Current = TimeWindowAccum{FitnessMin: math.MaxFloat64}
	}
}

func finalizeWindow(ts *TimelapseStats, endTick int) TimeWindow {
	c := &ts.Current
	s := float64(c.Samples)
	w := TimeWindow{
		StartTick:       ts.CurrentStart,
		EndTick:         endTick,
		AvgFitness:      c.FitnessSum / s,
		MaxFitness:      c.FitnessMax,
		MinFitness:      c.FitnessMin,
		TotalDeliveries: c.Deliveries,
		TotalPickups:    c.Pickups,
		AvgSpeed:        c.SpeedSum / s,
		AvgNeighbors:    c.NeighborSum / s,
		BotsCrashed:     c.Crashes,
		MsgCount:        c.MsgCount,
	}
	if w.MinFitness == math.MaxFloat64 {
		w.MinFitness = 0
	}
	return w
}

// TimelapseWindowCount returns the number of completed windows.
func TimelapseWindowCount(ts *TimelapseStats) int {
	if ts == nil {
		return 0
	}
	return len(ts.Windows)
}

// TimelapseTrend computes the trend (slope) of a metric over recent windows.
// Positive = improving, negative = declining.
func TimelapseTrend(ts *TimelapseStats, metric func(TimeWindow) float64, lastN int) float64 {
	if ts == nil || len(ts.Windows) < 2 {
		return 0
	}
	start := len(ts.Windows) - lastN
	if start < 0 {
		start = 0
	}
	windows := ts.Windows[start:]
	n := len(windows)
	if n < 2 {
		return 0
	}

	// Simple linear regression slope
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i, w := range windows {
		x := float64(i)
		y := metric(w)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	fn := float64(n)
	denom := fn*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 0
	}
	return (fn*sumXY - sumX*sumY) / denom
}

// TimelapseAvgMetric computes the average of a metric over recent windows.
func TimelapseAvgMetric(ts *TimelapseStats, metric func(TimeWindow) float64, lastN int) float64 {
	if ts == nil || len(ts.Windows) == 0 {
		return 0
	}
	start := len(ts.Windows) - lastN
	if start < 0 {
		start = 0
	}
	sum := 0.0
	count := 0
	for _, w := range ts.Windows[start:] {
		sum += metric(w)
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// TimelapseMaxMetric returns the maximum of a metric across all windows.
func TimelapseMaxMetric(ts *TimelapseStats, metric func(TimeWindow) float64) float64 {
	if ts == nil || len(ts.Windows) == 0 {
		return 0
	}
	best := metric(ts.Windows[0])
	for _, w := range ts.Windows[1:] {
		v := metric(w)
		if v > best {
			best = v
		}
	}
	return best
}
