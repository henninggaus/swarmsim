package swarm

import (
	"math"
	"math/rand"
)

// MultiSwarmState manages multiple competing swarms in the arena.
type MultiSwarmState struct {
	Swarms      []SwarmTeam
	SharedArena bool    // all swarms share one arena
	MigrationRate float64 // probability of bot migration per tick (default 0.001)
	MigrationCount int
	CompetitionMode int // 0=delivery race, 1=territory control
	Round       int
	RoundTicks  int
	RoundLimit  int // ticks per round (default 5000)
}

// SwarmTeam represents one team in the multi-swarm arena.
type SwarmTeam struct {
	ID         int
	Name       string
	Color      [3]uint8
	BotStart   int // index of first bot in ss.Bots
	BotEnd     int // index past last bot
	Score      int
	Deliveries int
	Territory  float64 // 0-1 fraction of arena controlled
	AvgFitness float64
}

// InitMultiSwarm sets up multi-swarm with the given number of teams.
func InitMultiSwarm(ss *SwarmState, teamCount int) {
	if teamCount < 2 {
		teamCount = 2
	}
	if teamCount > 4 {
		teamCount = 4
	}

	botsPerTeam := len(ss.Bots) / teamCount
	if botsPerTeam < 2 {
		return
	}

	colors := [][3]uint8{
		{255, 80, 80},   // red
		{80, 80, 255},   // blue
		{80, 255, 80},   // green
		{255, 255, 80},  // yellow
	}
	names := []string{"Rot", "Blau", "Gruen", "Gelb"}

	ms := &MultiSwarmState{
		Swarms:        make([]SwarmTeam, teamCount),
		SharedArena:   true,
		MigrationRate: 0.001,
		RoundLimit:    5000,
	}

	for i := 0; i < teamCount; i++ {
		start := i * botsPerTeam
		end := start + botsPerTeam
		if i == teamCount-1 {
			end = len(ss.Bots) // last team gets remainder
		}
		ms.Swarms[i] = SwarmTeam{
			ID:       i,
			Name:     names[i],
			Color:    colors[i],
			BotStart: start,
			BotEnd:   end,
		}
		// Assign team to bots
		for j := start; j < end && j < len(ss.Bots); j++ {
			ss.Bots[j].Team = i
			ss.Bots[j].LEDColor = colors[i]
		}
	}

	ss.MultiSwarm = ms
}

// ClearMultiSwarm disables the multi-swarm system.
func ClearMultiSwarm(ss *SwarmState) {
	ss.MultiSwarm = nil
}

// TickMultiSwarm advances multi-swarm state by one tick.
func TickMultiSwarm(ss *SwarmState, rng *rand.Rand) {
	ms := ss.MultiSwarm
	if ms == nil {
		return
	}

	ms.RoundTicks++

	// Update team scores
	for i := range ms.Swarms {
		team := &ms.Swarms[i]
		team.Deliveries = 0
		fitSum := 0.0
		count := 0
		for j := team.BotStart; j < team.BotEnd && j < len(ss.Bots); j++ {
			fitSum += ss.Bots[j].Fitness
			team.Deliveries += ss.Bots[j].Stats.TotalDeliveries
			count++
		}
		if count > 0 {
			team.AvgFitness = fitSum / float64(count)
		}
		team.Score = team.Deliveries
	}

	// Territory calculation
	computeTerritory(ss, ms)

	// Migration: occasionally move a bot from one team to another
	if ms.MigrationRate > 0 && rng.Float64() < ms.MigrationRate && len(ms.Swarms) >= 2 {
		migrateBotRandom(ss, ms, rng)
	}

	// Round check
	if ms.RoundLimit > 0 && ms.RoundTicks >= ms.RoundLimit {
		ms.Round++
		ms.RoundTicks = 0
	}
}

// computeTerritory calculates what fraction of the arena each team "controls"
// using a grid-based approach.
func computeTerritory(ss *SwarmState, ms *MultiSwarmState) {
	const gridSize = 10
	cols := int(ss.ArenaW / gridSize)
	rows := int(ss.ArenaH / gridSize)
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	totalCells := cols * rows

	// Count team presence per cell
	cellOwner := make([]int, totalCells) // -1 = unowned
	cellCount := make([][]int, totalCells)
	for i := range cellOwner {
		cellOwner[i] = -1
		cellCount[i] = make([]int, len(ms.Swarms))
	}

	for i := range ss.Bots {
		col := int(ss.Bots[i].X / gridSize)
		row := int(ss.Bots[i].Y / gridSize)
		if col < 0 {
			col = 0
		}
		if col >= cols {
			col = cols - 1
		}
		if row < 0 {
			row = 0
		}
		if row >= rows {
			row = rows - 1
		}
		team := ss.Bots[i].Team
		if team >= 0 && team < len(ms.Swarms) {
			cellCount[row*cols+col][team]++
		}
	}

	// Assign cells to team with most bots
	teamCells := make([]int, len(ms.Swarms))
	for i := range cellOwner {
		maxCount := 0
		maxTeam := -1
		for t := range ms.Swarms {
			if cellCount[i][t] > maxCount {
				maxCount = cellCount[i][t]
				maxTeam = t
			}
		}
		if maxTeam >= 0 {
			teamCells[maxTeam]++
		}
	}

	for i := range ms.Swarms {
		ms.Swarms[i].Territory = float64(teamCells[i]) / float64(totalCells)
	}
}

// migrateBotRandom moves a random bot from one team to another.
func migrateBotRandom(ss *SwarmState, ms *MultiSwarmState, rng *rand.Rand) {
	if len(ss.Bots) == 0 {
		return
	}
	botIdx := rng.Intn(len(ss.Bots))
	oldTeam := ss.Bots[botIdx].Team
	newTeam := rng.Intn(len(ms.Swarms))
	if newTeam == oldTeam {
		newTeam = (newTeam + 1) % len(ms.Swarms)
	}
	ss.Bots[botIdx].Team = newTeam
	ss.Bots[botIdx].LEDColor = ms.Swarms[newTeam].Color
	ms.MigrationCount++
}

// MultiSwarmLeader returns the team index with the highest score.
func MultiSwarmLeader(ms *MultiSwarmState) int {
	if ms == nil || len(ms.Swarms) == 0 {
		return -1
	}
	best := 0
	for i := 1; i < len(ms.Swarms); i++ {
		if ms.Swarms[i].Score > ms.Swarms[best].Score {
			best = i
		}
	}
	return best
}

// MultiSwarmTotalBots returns the total number of bots across all teams.
func MultiSwarmTotalBots(ms *MultiSwarmState) int {
	if ms == nil {
		return 0
	}
	total := 0
	for _, team := range ms.Swarms {
		total += team.BotEnd - team.BotStart
	}
	return total
}

// MultiSwarmTeamDistance returns the average distance between team centroids.
func MultiSwarmTeamDistance(ss *SwarmState, ms *MultiSwarmState) float64 {
	if ms == nil || len(ms.Swarms) < 2 {
		return 0
	}

	// Compute centroids
	centroids := make([][2]float64, len(ms.Swarms))
	for i, team := range ms.Swarms {
		cx, cy := 0.0, 0.0
		count := 0
		for j := team.BotStart; j < team.BotEnd && j < len(ss.Bots); j++ {
			cx += ss.Bots[j].X
			cy += ss.Bots[j].Y
			count++
		}
		if count > 0 {
			centroids[i] = [2]float64{cx / float64(count), cy / float64(count)}
		}
	}

	// Average pairwise distance
	sum := 0.0
	pairs := 0
	for i := 0; i < len(centroids); i++ {
		for j := i + 1; j < len(centroids); j++ {
			dx := centroids[i][0] - centroids[j][0]
			dy := centroids[i][1] - centroids[j][1]
			sum += math.Sqrt(dx*dx + dy*dy)
			pairs++
		}
	}
	if pairs == 0 {
		return 0
	}
	return sum / float64(pairs)
}
