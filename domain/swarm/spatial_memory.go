package swarm

import (
	"math"
	"swarmsim/logger"
)

// SpatialMemoryState manages a shared spatial knowledge map.
// Bots write their observations (resource density, danger level, throughput)
// into a grid. Other bots read this collective map to navigate more
// efficiently. The map decays over time, requiring continuous updates.
type SpatialMemoryState struct {
	Cells    []MemoryCell // flat grid of memory cells
	GridW    int
	GridH    int
	CellSize float64 // world units per cell (default 40)
	Decay    float64 // decay rate per tick (default 0.002)

	// Stats
	KnownCells   int     // cells with any data
	AvgConfidence float64 // average confidence across known cells
	TotalWrites  int
	TotalReads   int
}

// MemoryCell stores aggregated knowledge about one area.
type MemoryCell struct {
	ResourceScore  float64 // how many resources found here (higher = better)
	DangerScore    float64 // how dangerous this area is
	TrafficScore   float64 // how many bots pass through
	DeliveryScore  float64 // successful deliveries nearby
	Confidence     float64 // how recent/reliable the data is (0-1)
	LastUpdateTick int
	WriteCount     int
}

// InitSpatialMemory sets up the shared spatial memory.
func InitSpatialMemory(ss *SwarmState) {
	cellSize := 40.0
	gw := int(ss.ArenaW/cellSize) + 1
	gh := int(ss.ArenaH/cellSize) + 1

	sm := &SpatialMemoryState{
		Cells:    make([]MemoryCell, gw*gh),
		GridW:    gw,
		GridH:    gh,
		CellSize: cellSize,
		Decay:    0.002,
	}

	ss.SpatialMemory = sm
	logger.Info("SPMEM", "Initialisiert: %dx%d Wissensgrid, CellSize=%.0f", gw, gh, cellSize)
}

// ClearSpatialMemory disables the spatial memory.
func ClearSpatialMemory(ss *SwarmState) {
	ss.SpatialMemory = nil
	ss.SpatialMemoryOn = false
}

// TickSpatialMemory runs one tick of the spatial memory system.
func TickSpatialMemory(ss *SwarmState) {
	sm := ss.SpatialMemory
	if sm == nil {
		return
	}

	// Decay all cells
	for i := range sm.Cells {
		if sm.Cells[i].Confidence > 0 {
			sm.Cells[i].Confidence -= sm.Decay
			if sm.Cells[i].Confidence < 0 {
				sm.Cells[i].Confidence = 0
			}
			sm.Cells[i].ResourceScore *= 0.999
			sm.Cells[i].DangerScore *= 0.999
			sm.Cells[i].TrafficScore *= 0.998
		}
	}

	// Bots write and read
	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Write observations
		smWriteObservation(ss, sm, i, bot)

		// Read and navigate
		smReadAndNavigate(ss, sm, bot)
	}

	// Update stats
	known := 0
	totalConf := 0.0
	for _, c := range sm.Cells {
		if c.Confidence > 0.01 {
			known++
			totalConf += c.Confidence
		}
	}
	sm.KnownCells = known
	if known > 0 {
		sm.AvgConfidence = totalConf / float64(known)
	}
}

func smCellIdx(sm *SpatialMemoryState, x, y float64) int {
	cx := int(x / sm.CellSize)
	cy := int(y / sm.CellSize)
	if cx < 0 || cx >= sm.GridW || cy < 0 || cy >= sm.GridH {
		return -1
	}
	return cy*sm.GridW + cx
}

// smWriteObservation records the bot's current observation.
func smWriteObservation(ss *SwarmState, sm *SpatialMemoryState, botIdx int, bot *SwarmBot) {
	// Write every 10 ticks to reduce overhead
	if ss.Tick%10 != botIdx%10 {
		return
	}

	idx := smCellIdx(sm, bot.X, bot.Y)
	if idx < 0 || idx >= len(sm.Cells) {
		return
	}

	cell := &sm.Cells[idx]
	alpha := 0.3 // learning rate for exponential moving average

	// Resource info
	if bot.NearestPickupDist < 80 {
		resourceVal := 1.0 - bot.NearestPickupDist/80.0
		cell.ResourceScore = cell.ResourceScore*(1-alpha) + resourceVal*alpha
	}

	// Traffic
	cell.TrafficScore = cell.TrafficScore*(1-alpha) + float64(bot.NeighborCount)/10.0*alpha

	// Delivery success
	if bot.CarryingPkg >= 0 && bot.NearestDropoffDist < 60 {
		cell.DeliveryScore = cell.DeliveryScore*(1-alpha) + 1.0*alpha
	}

	cell.Confidence = math.Min(cell.Confidence+0.1, 1.0)
	cell.LastUpdateTick = ss.Tick
	cell.WriteCount++
	sm.TotalWrites++
}

// smReadAndNavigate uses the spatial map to guide the bot.
func smReadAndNavigate(ss *SwarmState, sm *SpatialMemoryState, bot *SwarmBot) {
	// Only read every 20 ticks
	if ss.Tick%20 != 0 {
		return
	}

	sm.TotalReads++

	// Look at neighboring cells to decide direction
	cx := int(bot.X / sm.CellSize)
	cy := int(bot.Y / sm.CellSize)

	bestScore := -999.0
	bestDX, bestDY := 0.0, 0.0

	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx := cx + dx
			ny := cy + dy
			if nx < 0 || nx >= sm.GridW || ny < 0 || ny >= sm.GridH {
				continue
			}

			cell := &sm.Cells[ny*sm.GridW+nx]
			if cell.Confidence < 0.05 {
				// Unknown area: exploration bonus
				score := 0.3
				if score > bestScore {
					bestScore = score
					bestDX = float64(dx)
					bestDY = float64(dy)
				}
				continue
			}

			// Score based on what the bot needs
			score := 0.0
			if bot.CarryingPkg < 0 {
				// Looking for resources
				score += cell.ResourceScore * 2.0
				score -= cell.TrafficScore * 0.5 // avoid crowded areas
			} else {
				// Looking for dropoff
				score += cell.DeliveryScore * 2.0
			}
			score -= cell.DangerScore

			if score > bestScore {
				bestScore = score
				bestDX = float64(dx)
				bestDY = float64(dy)
			}
		}
	}

	if bestScore > -999 && (bestDX != 0 || bestDY != 0) {
		targetAngle := math.Atan2(bestDY, bestDX)
		diff := targetAngle - bot.Angle
		diff = WrapAngle(diff)
		bot.Angle += diff * 0.05 // gentle nudge
	}
}

// SpatialMemKnown returns the number of known cells.
func SpatialMemKnown(sm *SpatialMemoryState) int {
	if sm == nil {
		return 0
	}
	return sm.KnownCells
}

// SpatialMemConfidence returns average confidence.
func SpatialMemConfidence(sm *SpatialMemoryState) float64 {
	if sm == nil {
		return 0
	}
	return sm.AvgConfidence
}

// SpatialMemCellScore returns the resource score at a position.
func SpatialMemCellScore(sm *SpatialMemoryState, x, y float64) float64 {
	if sm == nil {
		return 0
	}
	idx := smCellIdx(sm, x, y)
	if idx < 0 || idx >= len(sm.Cells) {
		return 0
	}
	return sm.Cells[idx].ResourceScore
}
