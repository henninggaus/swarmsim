package swarm

import (
	"math"
	"swarmsim/logger"
)

// Stigmergy2State manages advanced pheromone trails with compound messages.
// Unlike simple pheromones, these carry structured information: a type
// (food/danger/path/home), intensity, a directional vector, and age.
// Bots read and interpret these compound trails context-dependently.
type Stigmergy2State struct {
	Grid       [][]PheromoneCell // 2D grid of pheromone cells
	GridW      int               // grid width in cells
	GridH      int               // grid height in cells
	CellSize   float64           // world units per cell (default 30)
	DecayRate  float64           // decay per tick (default 0.005)
	MaxTrails  int               // max trails per cell (default 5)
	DepositStr float64           // deposit strength (default 0.8)

	// Stats
	TotalTrails  int
	ActiveCells  int
	AvgIntensity float64
}

// PheromoneCell holds trails at one grid position.
type PheromoneCell struct {
	Trails []CompoundTrail
}

// CompoundTrail is a structured pheromone deposit.
type CompoundTrail struct {
	Type      TrailType
	Intensity float64 // strength (0-1, decays)
	DirX      float64 // suggested direction X component
	DirY      float64 // suggested direction Y component
	Age       int     // ticks since deposit
	BotID     int     // who deposited (to avoid self-following)
}

// TrailType categorizes the pheromone message.
type TrailType int

const (
	TrailFood   TrailType = iota // "food this way"
	TrailDanger                   // "danger here"
	TrailPath                     // "I traveled this path"
	TrailHome                     // "base/dropoff this way"
)

// TrailTypeName returns the display name.
func TrailTypeName(tt TrailType) string {
	switch tt {
	case TrailFood:
		return "Nahrung"
	case TrailDanger:
		return "Gefahr"
	case TrailPath:
		return "Pfad"
	case TrailHome:
		return "Heimat"
	default:
		return "?"
	}
}

// InitStigmergy2 sets up the advanced stigmergy system.
func InitStigmergy2(ss *SwarmState) {
	cellSize := 30.0
	gw := int(ss.ArenaW/cellSize) + 1
	gh := int(ss.ArenaH/cellSize) + 1

	sg := &Stigmergy2State{
		GridW:      gw,
		GridH:      gh,
		CellSize:   cellSize,
		DecayRate:  0.005,
		MaxTrails:  5,
		DepositStr: 0.8,
		Grid:       make([][]PheromoneCell, gw*gh),
	}

	// Initialize flat grid
	for i := range sg.Grid {
		sg.Grid[i] = nil // lazy init
	}

	ss.Stigmergy2 = sg
	logger.Info("STIG2", "Initialisiert: %dx%d Grid, CellSize=%.0f", gw, gh, cellSize)
}

// ClearStigmergy2 disables the stigmergy system.
func ClearStigmergy2(ss *SwarmState) {
	ss.Stigmergy2 = nil
	ss.Stigmergy2On = false
}

// TickStigmergy2 runs one tick of the stigmergy system.
func TickStigmergy2(ss *SwarmState) {
	sg := ss.Stigmergy2
	if sg == nil {
		return
	}

	// Phase 1: Bots deposit pheromones
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		depositPheromones(ss, sg, i, bot)
	}

	// Phase 2: Decay all trails
	totalTrails := 0
	activeCells := 0
	totalIntensity := 0.0

	for idx := range sg.Grid {
		if sg.Grid[idx] == nil {
			continue
		}
		cells := sg.Grid[idx]
		alive := 0
		for j := range cells {
			cell := &cells[j]
			for k := len(cell.Trails) - 1; k >= 0; k-- {
				cell.Trails[k].Intensity -= sg.DecayRate
				cell.Trails[k].Age++
				if cell.Trails[k].Intensity <= 0 {
					cell.Trails = append(cell.Trails[:k], cell.Trails[k+1:]...)
				} else {
					totalIntensity += cell.Trails[k].Intensity
					alive++
				}
			}
		}
		if alive > 0 {
			activeCells++
		}
		totalTrails += alive
	}

	sg.TotalTrails = totalTrails
	sg.ActiveCells = activeCells
	if totalTrails > 0 {
		sg.AvgIntensity = totalIntensity / float64(totalTrails)
	}

	// Phase 3: Bots read and respond to pheromones
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		readPheromones(ss, sg, i, bot)
	}
}

func stig2CellIdx(sg *Stigmergy2State, x, y float64) int {
	cx := int(x / sg.CellSize)
	cy := int(y / sg.CellSize)
	if cx < 0 || cx >= sg.GridW || cy < 0 || cy >= sg.GridH {
		return -1
	}
	return cy*sg.GridW + cx
}

// depositPheromones places trails based on bot state.
func depositPheromones(ss *SwarmState, sg *Stigmergy2State, botIdx int, bot *SwarmBot) {
	// Only deposit every 5 ticks
	if ss.Tick%5 != botIdx%5 {
		return
	}

	idx := stig2CellIdx(sg, bot.X, bot.Y)
	if idx < 0 || idx >= len(sg.Grid) {
		return
	}

	// Lazy init
	if sg.Grid[idx] == nil {
		sg.Grid[idx] = []PheromoneCell{{}}
	}
	cell := &sg.Grid[idx][0]

	dirX := math.Cos(bot.Angle)
	dirY := math.Sin(bot.Angle)

	// Determine what to deposit based on context
	if bot.CarryingPkg >= 0 && bot.NearestPickupDist < 100 {
		// Just picked up: mark food trail pointing back to source
		addTrail(sg, cell, CompoundTrail{
			Type:      TrailFood,
			Intensity: sg.DepositStr,
			DirX:      -dirX, // point back where we came from
			DirY:      -dirY,
			BotID:     botIdx,
		})
	} else if bot.CarryingPkg >= 0 {
		// Carrying: leave path trail
		addTrail(sg, cell, CompoundTrail{
			Type:      TrailPath,
			Intensity: sg.DepositStr * 0.5,
			DirX:      dirX,
			DirY:      dirY,
			BotID:     botIdx,
		})
	} else if bot.NearestDropoffDist < 80 {
		// Near dropoff: mark home trail
		addTrail(sg, cell, CompoundTrail{
			Type:      TrailHome,
			Intensity: sg.DepositStr * 0.7,
			DirX:      -dirX,
			DirY:      -dirY,
			BotID:     botIdx,
		})
	}
}

func addTrail(sg *Stigmergy2State, cell *PheromoneCell, trail CompoundTrail) {
	if len(cell.Trails) >= sg.MaxTrails {
		// Replace weakest
		weakest := 0
		for j := 1; j < len(cell.Trails); j++ {
			if cell.Trails[j].Intensity < cell.Trails[weakest].Intensity {
				weakest = j
			}
		}
		if cell.Trails[weakest].Intensity < trail.Intensity {
			cell.Trails[weakest] = trail
		}
	} else {
		cell.Trails = append(cell.Trails, trail)
	}
}

// readPheromones interprets nearby trails and adjusts behavior.
func readPheromones(ss *SwarmState, sg *Stigmergy2State, botIdx int, bot *SwarmBot) {
	idx := stig2CellIdx(sg, bot.X, bot.Y)
	if idx < 0 || idx >= len(sg.Grid) || sg.Grid[idx] == nil {
		return
	}

	cell := &sg.Grid[idx][0]

	// Find strongest trail of each type
	var bestFood, bestDanger, bestHome *CompoundTrail
	for j := range cell.Trails {
		t := &cell.Trails[j]
		if t.BotID == botIdx {
			continue // ignore own trails
		}

		switch t.Type {
		case TrailFood:
			if bestFood == nil || t.Intensity > bestFood.Intensity {
				bestFood = t
			}
		case TrailDanger:
			if bestDanger == nil || t.Intensity > bestDanger.Intensity {
				bestDanger = t
			}
		case TrailHome:
			if bestHome == nil || t.Intensity > bestHome.Intensity {
				bestHome = t
			}
		}
	}

	// React to trails
	if bestFood != nil && bot.CarryingPkg < 0 {
		// Follow food trail
		targetAngle := math.Atan2(bestFood.DirY, bestFood.DirX)
		turnToward(bot, targetAngle, bestFood.Intensity*0.3)
		bot.LEDColor = [3]uint8{200, 200, 0} // yellow: following food
	}

	if bestDanger != nil {
		// Flee from danger
		fleeAngle := math.Atan2(-bestDanger.DirY, -bestDanger.DirX)
		turnToward(bot, fleeAngle, bestDanger.Intensity*0.5)
		bot.Speed *= 1.2
		bot.LEDColor = [3]uint8{255, 0, 0} // red: danger
	}

	if bestHome != nil && bot.CarryingPkg >= 0 {
		// Follow home trail when carrying
		targetAngle := math.Atan2(bestHome.DirY, bestHome.DirX)
		turnToward(bot, targetAngle, bestHome.Intensity*0.3)
		bot.LEDColor = [3]uint8{0, 200, 100} // green: heading home
	}
}

func turnToward(bot *SwarmBot, targetAngle, strength float64) {
	diff := targetAngle - bot.Angle
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	bot.Angle += diff * strength
}

// Stig2TrailCount returns total active trails.
func Stig2TrailCount(sg *Stigmergy2State) int {
	if sg == nil {
		return 0
	}
	return sg.TotalTrails
}

// Stig2ActiveCells returns the number of cells with active trails.
func Stig2ActiveCells(sg *Stigmergy2State) int {
	if sg == nil {
		return 0
	}
	return sg.ActiveCells
}
