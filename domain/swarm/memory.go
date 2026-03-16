package swarm

import "math"

// BotMemory: per-bot visited grid that tracks which arena cells a bot has explored.
const MemoryCellSize = 40 // 40px cells → 800/40 = 20×20 grid

// InitBotMemory initializes the memory grid for all bots.
func InitBotMemory(ss *SwarmState) {
	cols := int(ss.ArenaW) / MemoryCellSize
	rows := int(ss.ArenaH) / MemoryCellSize
	for i := range ss.Bots {
		ss.Bots[i].MemoryGrid = make([]uint8, cols*rows)
		ss.Bots[i].MemoryCols = cols
		ss.Bots[i].MemoryRows = rows
	}
}

// ClearBotMemory resets memory for all bots.
func ClearBotMemory(ss *SwarmState) {
	for i := range ss.Bots {
		ss.Bots[i].MemoryGrid = nil
		ss.Bots[i].MemoryCols = 0
		ss.Bots[i].MemoryRows = 0
	}
}

// UpdateBotMemory marks the current cell as visited for each bot (call every ~5 ticks).
func UpdateBotMemory(ss *SwarmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.MemoryGrid == nil {
			continue
		}
		if bot.X < 0 || bot.Y < 0 {
			continue
		}
		c := int(bot.X) / MemoryCellSize
		r := int(bot.Y) / MemoryCellSize
		if c >= 0 && c < bot.MemoryCols && r >= 0 && r < bot.MemoryRows {
			if bot.MemoryGrid[r*bot.MemoryCols+c] < 255 {
				bot.MemoryGrid[r*bot.MemoryCols+c]++
			}
		}
	}
}

// BotVisitedHere returns how many times the bot visited its current cell.
func BotVisitedHere(bot *SwarmBot) int {
	if bot.MemoryGrid == nil {
		return 0
	}
	c := int(bot.X) / MemoryCellSize
	r := int(bot.Y) / MemoryCellSize
	if c >= 0 && c < bot.MemoryCols && r >= 0 && r < bot.MemoryRows {
		return int(bot.MemoryGrid[r*bot.MemoryCols+c])
	}
	return 0
}

// BotVisitedAhead returns how many times the bot visited the cell 40px ahead.
func BotVisitedAhead(bot *SwarmBot) int {
	if bot.MemoryGrid == nil {
		return 0
	}
	ax := bot.X + math.Cos(bot.Angle)*float64(MemoryCellSize)
	ay := bot.Y + math.Sin(bot.Angle)*float64(MemoryCellSize)
	c := int(ax) / MemoryCellSize
	r := int(ay) / MemoryCellSize
	if c >= 0 && c < bot.MemoryCols && r >= 0 && r < bot.MemoryRows {
		return int(bot.MemoryGrid[r*bot.MemoryCols+c])
	}
	return 0
}

// BotExploredPercent returns what percentage of grid the bot has visited.
func BotExploredPercent(bot *SwarmBot) int {
	if bot.MemoryGrid == nil {
		return 0
	}
	visited := 0
	for _, v := range bot.MemoryGrid {
		if v > 0 {
			visited++
		}
	}
	total := bot.MemoryCols * bot.MemoryRows
	if total == 0 {
		return 0
	}
	return visited * 100 / total
}
