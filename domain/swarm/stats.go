package swarm

import "fmt"

// StatsTracker tracks real-time statistics for the dashboard.
type StatsTracker struct {
	// Delivery rate tracking (per 500-tick window)
	DeliveryBuckets   []int // deliveries per 500-tick window
	CorrectBuckets    []int // correct deliveries per window
	WrongBuckets      []int // wrong deliveries per window
	TeamABuckets      []int // Team A deliveries per window (teams mode)
	TeamBBuckets      []int // Team B deliveries per window (teams mode)
	CurrentBucketIdx  int   // current window index
	LastBucketTick    int   // tick when current bucket started

	// Heatmap (80x60 cells covering arena)
	HeatmapGrid [80][60]int
	HeatmapMax  int // max cell value (for normalization)

	// Bot efficiency ranking (updated every 500 ticks)
	BotRankings   []BotRankEntry
	RankingUpdate int // tick of last ranking update

	// Event ticker (scrolling log of recent events)
	EventTicker []string // last 20 events

	// Action heatmap (tracks where specific actions happen)
	ActionHeatmap    [80][60]int // pickup + drop events per cell
	ActionHeatmapMax int
	ShowActionHeat   bool // A key toggle: show action heatmap instead of motion heatmap
}

// BotRankEntry stores a bot's ranking info.
type BotRankEntry struct {
	BotIdx     int
	Deliveries int
	AvgTime    int
}

// NewStatsTracker creates a new stats tracker.
func NewStatsTracker() *StatsTracker {
	return &StatsTracker{
		DeliveryBuckets: make([]int, 0, 20),
		CorrectBuckets:  make([]int, 0, 20),
		WrongBuckets:    make([]int, 0, 20),
		TeamABuckets:    make([]int, 0, 20),
		TeamBBuckets:    make([]int, 0, 20),
		BotRankings:     make([]BotRankEntry, 0),
		EventTicker:     make([]string, 0, 20),
	}
}

// Update is called every tick to track bucket transitions.
func (st *StatsTracker) Update(ss *SwarmState) {
	// Bucket transition every 500 ticks
	if ss.Tick-st.LastBucketTick >= 500 {
		st.LastBucketTick = ss.Tick
		// Start a new bucket
		st.DeliveryBuckets = append(st.DeliveryBuckets, 0)
		st.CorrectBuckets = append(st.CorrectBuckets, 0)
		st.WrongBuckets = append(st.WrongBuckets, 0)
		st.TeamABuckets = append(st.TeamABuckets, 0)
		st.TeamBBuckets = append(st.TeamBBuckets, 0)
		st.CurrentBucketIdx = len(st.DeliveryBuckets) - 1
		// Keep only last 12 buckets
		if len(st.DeliveryBuckets) > 12 {
			st.DeliveryBuckets = st.DeliveryBuckets[len(st.DeliveryBuckets)-12:]
			st.CorrectBuckets = st.CorrectBuckets[len(st.CorrectBuckets)-12:]
			st.WrongBuckets = st.WrongBuckets[len(st.WrongBuckets)-12:]
			st.TeamABuckets = st.TeamABuckets[len(st.TeamABuckets)-12:]
			st.TeamBBuckets = st.TeamBBuckets[len(st.TeamBBuckets)-12:]
			st.CurrentBucketIdx = len(st.DeliveryBuckets) - 1
		}
	}
}

// RecordDelivery records a delivery event in the current bucket.
func (st *StatsTracker) RecordDelivery(correct bool, team int) {
	if len(st.DeliveryBuckets) == 0 {
		st.DeliveryBuckets = append(st.DeliveryBuckets, 0)
		st.CorrectBuckets = append(st.CorrectBuckets, 0)
		st.WrongBuckets = append(st.WrongBuckets, 0)
		st.TeamABuckets = append(st.TeamABuckets, 0)
		st.TeamBBuckets = append(st.TeamBBuckets, 0)
		st.CurrentBucketIdx = 0
	}
	idx := len(st.DeliveryBuckets) - 1
	st.DeliveryBuckets[idx]++
	if correct {
		st.CorrectBuckets[idx]++
	} else {
		st.WrongBuckets[idx]++
	}
	if team == 1 {
		st.TeamABuckets[idx]++
	} else if team == 2 {
		st.TeamBBuckets[idx]++
	}
}

// UpdateHeatmap updates the heatmap grid with current bot positions.
func (st *StatsTracker) UpdateHeatmap(ss *SwarmState) {
	cellW := ss.ArenaW / 80.0
	cellH := ss.ArenaH / 60.0
	for i := range ss.Bots {
		cx := int(ss.Bots[i].X / cellW)
		cy := int(ss.Bots[i].Y / cellH)
		if cx >= 0 && cx < 80 && cy >= 0 && cy < 60 {
			st.HeatmapGrid[cx][cy]++
			if st.HeatmapGrid[cx][cy] > st.HeatmapMax {
				st.HeatmapMax = st.HeatmapGrid[cx][cy]
			}
		}
	}
}

// UpdateRankings rebuilds the bot efficiency ranking.
func (st *StatsTracker) UpdateRankings(ss *SwarmState) {
	st.BotRankings = st.BotRankings[:0]
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		avgTime := 0
		if len(bot.Stats.DeliveryTimes) > 0 {
			sum := 0
			for _, t := range bot.Stats.DeliveryTimes {
				sum += t
			}
			avgTime = sum / len(bot.Stats.DeliveryTimes)
		}
		st.BotRankings = append(st.BotRankings, BotRankEntry{
			BotIdx:     i,
			Deliveries: bot.Stats.TotalDeliveries,
			AvgTime:    avgTime,
		})
	}
	// Sort by deliveries descending
	for i := 0; i < len(st.BotRankings)-1; i++ {
		for j := i + 1; j < len(st.BotRankings); j++ {
			if st.BotRankings[j].Deliveries > st.BotRankings[i].Deliveries {
				st.BotRankings[i], st.BotRankings[j] = st.BotRankings[j], st.BotRankings[i]
			}
		}
	}
}

// AddEvent adds an event to the ticker.
func (st *StatsTracker) AddEvent(text string) {
	st.EventTicker = append(st.EventTicker, text)
	if len(st.EventTicker) > 20 {
		st.EventTicker = st.EventTicker[len(st.EventTicker)-20:]
	}
}

// AddDeliveryEvent adds a formatted delivery event.
func (st *StatsTracker) AddDeliveryEvent(botIdx int, colorName string, correct bool, ticks int) {
	mark := "X"
	if correct {
		mark = "OK"
	}
	text := fmt.Sprintf("#%d %s %s (%dt)", botIdx, colorName, mark, ticks)
	st.AddEvent(text)
}

// AddPickupEvent adds a formatted pickup event.
func (st *StatsTracker) AddPickupEvent(botIdx int, colorName string) {
	text := fmt.Sprintf("#%d pickup %s", botIdx, colorName)
	st.AddEvent(text)
}

// RecordActionAt records an action event at a world position for the action heatmap.
func (st *StatsTracker) RecordActionAt(x, y, arenaW, arenaH float64) {
	cellW := arenaW / 80.0
	cellH := arenaH / 60.0
	cx := int(x / cellW)
	cy := int(y / cellH)
	if cx >= 0 && cx < 80 && cy >= 0 && cy < 60 {
		st.ActionHeatmap[cx][cy]++
		if st.ActionHeatmap[cx][cy] > st.ActionHeatmapMax {
			st.ActionHeatmapMax = st.ActionHeatmap[cx][cy]
		}
	}
}
