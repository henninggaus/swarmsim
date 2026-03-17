package swarm

import (
	"math"
	"swarmsim/logger"
)

// BlockType represents the type of a placed block.
type BlockType int

const (
	BlockWall   BlockType = iota // solid block, acts as obstacle
	BlockBridge                  // walkable, increases speed
	BlockRamp                    // connects height levels
	BlockCount
)

// StigmergyBlock is a single block placed by a bot.
type StigmergyBlock struct {
	X, Y      float64
	Type      BlockType
	PlacedBy  int // bot index that placed it
	PlacedAt  int // tick when placed
	Integrity int // health points (default 100, decays or can be removed)
}

// StigmergyGrid manages the collective building system.
type StigmergyGrid struct {
	Blocks      []StigmergyBlock
	MaxBlocks   int     // max blocks in arena (default 200)
	BlockSize   float64 // block side length in pixels (default 20)
	PlaceCost   int     // energy cost to place a block (default 10)
	RemoveCost  int     // energy cost to remove a block (default 5)
	DecayRate   int     // ticks until block loses 1 integrity (0=no decay)
	PlaceCooldown int   // ticks between placements per bot (default 30)

	// Grid for fast lookup (cell → block index, -1 = empty)
	Grid     []int
	GridCols int
	GridRows int

	// Stats
	TotalPlaced   int
	TotalRemoved  int
	TotalDecayed  int

	// Resource system: bots must collect material before building
	MaterialEnabled bool
	BotMaterial     []int // per-bot material count
	MaterialMax     int   // max material per bot (default 5)
	MaterialPickupRange float64 // range to pick up material from source (default 30)
}

// MaterialSource is a location where bots can pick up building material.
type MaterialSource struct {
	X, Y      float64
	Supply    int // remaining material (0 = depleted)
	MaxSupply int
	RespawnIn int // ticks until respawn
}

// StigmergyState holds all stigmergy data.
type StigmergyState struct {
	Grid            *StigmergyGrid
	MaterialSources []MaterialSource
}

// NewStigmergyGrid creates a stigmergy building grid for the arena.
func NewStigmergyGrid(arenaW, arenaH float64, botCount int) *StigmergyGrid {
	blockSize := 20.0
	cols := int(math.Ceil(arenaW / blockSize))
	rows := int(math.Ceil(arenaH / blockSize))

	grid := make([]int, cols*rows)
	for i := range grid {
		grid[i] = -1
	}

	return &StigmergyGrid{
		MaxBlocks:     200,
		BlockSize:     blockSize,
		PlaceCost:     10,
		RemoveCost:    5,
		DecayRate:     0, // no decay by default
		PlaceCooldown: 30,
		Grid:          grid,
		GridCols:      cols,
		GridRows:      rows,
		MaterialMax:   5,
		MaterialPickupRange: 30,
		BotMaterial:   make([]int, botCount),
	}
}

// gridIndex returns the flat grid index for world coordinates.
func (sg *StigmergyGrid) gridIndex(x, y float64) int {
	col := int(x / sg.BlockSize)
	row := int(y / sg.BlockSize)
	if col < 0 {
		col = 0
	}
	if row < 0 {
		row = 0
	}
	if col >= sg.GridCols {
		col = sg.GridCols - 1
	}
	if row >= sg.GridRows {
		row = sg.GridRows - 1
	}
	return row*sg.GridCols + col
}

// HasBlock returns whether there is a block at the given position.
func (sg *StigmergyGrid) HasBlock(x, y float64) bool {
	idx := sg.gridIndex(x, y)
	return sg.Grid[idx] >= 0
}

// GetBlock returns the block at the given position, or nil.
func (sg *StigmergyGrid) GetBlock(x, y float64) *StigmergyBlock {
	idx := sg.gridIndex(x, y)
	blockIdx := sg.Grid[idx]
	if blockIdx < 0 || blockIdx >= len(sg.Blocks) {
		return nil
	}
	return &sg.Blocks[blockIdx]
}

// PlaceBlock places a new block at the given position.
func PlaceBlock(sg *StigmergyGrid, x, y float64, btype BlockType, botIdx, tick int) bool {
	if sg == nil {
		return false
	}
	if len(sg.Blocks) >= sg.MaxBlocks {
		return false
	}

	// Snap to grid
	col := int(x / sg.BlockSize)
	row := int(y / sg.BlockSize)
	if col < 0 || col >= sg.GridCols || row < 0 || row >= sg.GridRows {
		return false
	}

	gridIdx := row*sg.GridCols + col
	if sg.Grid[gridIdx] >= 0 {
		return false // already occupied
	}

	// Check material if enabled
	if sg.MaterialEnabled && botIdx >= 0 && botIdx < len(sg.BotMaterial) {
		if sg.BotMaterial[botIdx] <= 0 {
			return false
		}
		sg.BotMaterial[botIdx]--
	}

	block := StigmergyBlock{
		X:         float64(col)*sg.BlockSize + sg.BlockSize/2,
		Y:         float64(row)*sg.BlockSize + sg.BlockSize/2,
		Type:      btype,
		PlacedBy:  botIdx,
		PlacedAt:  tick,
		Integrity: 100,
	}

	blockIdx := len(sg.Blocks)
	sg.Blocks = append(sg.Blocks, block)
	sg.Grid[gridIdx] = blockIdx
	sg.TotalPlaced++

	return true
}

// RemoveBlock removes a block at the given position.
func RemoveBlock(sg *StigmergyGrid, x, y float64) bool {
	if sg == nil {
		return false
	}

	col := int(x / sg.BlockSize)
	row := int(y / sg.BlockSize)
	if col < 0 || col >= sg.GridCols || row < 0 || row >= sg.GridRows {
		return false
	}

	gridIdx := row*sg.GridCols + col
	blockIdx := sg.Grid[gridIdx]
	if blockIdx < 0 {
		return false
	}

	// Mark as removed (set integrity to 0)
	sg.Blocks[blockIdx].Integrity = 0
	sg.Grid[gridIdx] = -1
	sg.TotalRemoved++

	return true
}

// TickStigmergy updates the stigmergy system (decay, material respawn).
func TickStigmergy(ss *SwarmState) {
	st := ss.Stigmergy
	if st == nil || st.Grid == nil {
		return
	}
	sg := st.Grid

	// Block decay
	if sg.DecayRate > 0 {
		for i := range sg.Blocks {
			if sg.Blocks[i].Integrity <= 0 {
				continue
			}
			if ss.Tick > 0 && ss.Tick%sg.DecayRate == 0 {
				sg.Blocks[i].Integrity--
				if sg.Blocks[i].Integrity <= 0 {
					// Remove from grid
					col := int(sg.Blocks[i].X / sg.BlockSize)
					row := int(sg.Blocks[i].Y / sg.BlockSize)
					if col >= 0 && col < sg.GridCols && row >= 0 && row < sg.GridRows {
						sg.Grid[row*sg.GridCols+col] = -1
					}
					sg.TotalDecayed++
				}
			}
		}
	}

	// Material source respawn
	for i := range st.MaterialSources {
		src := &st.MaterialSources[i]
		if src.Supply <= 0 && src.RespawnIn > 0 {
			src.RespawnIn--
			if src.RespawnIn <= 0 {
				src.Supply = src.MaxSupply
			}
		}
	}
}

// BotPickupMaterial lets a bot pick up building material from a nearby source.
func BotPickupMaterial(ss *SwarmState, botIdx int) bool {
	st := ss.Stigmergy
	if st == nil || st.Grid == nil || !st.Grid.MaterialEnabled {
		return false
	}
	sg := st.Grid
	if botIdx < 0 || botIdx >= len(sg.BotMaterial) {
		return false
	}
	if sg.BotMaterial[botIdx] >= sg.MaterialMax {
		return false
	}

	bot := &ss.Bots[botIdx]
	for i := range st.MaterialSources {
		src := &st.MaterialSources[i]
		if src.Supply <= 0 {
			continue
		}
		dx := bot.X - src.X
		dy := bot.Y - src.Y
		if math.Sqrt(dx*dx+dy*dy) < sg.MaterialPickupRange {
			sg.BotMaterial[botIdx]++
			src.Supply--
			if src.Supply <= 0 {
				src.RespawnIn = 500 // respawn after 500 ticks
			}
			return true
		}
	}
	return false
}

// BotPlaceBlock places a block at the bot's current forward position.
func BotPlaceBlock(ss *SwarmState, botIdx int, btype BlockType) bool {
	st := ss.Stigmergy
	if st == nil || st.Grid == nil {
		return false
	}
	bot := &ss.Bots[botIdx]

	// Place block 20px ahead of bot
	placeX := bot.X + math.Cos(bot.Angle)*20
	placeY := bot.Y + math.Sin(bot.Angle)*20

	return PlaceBlock(st.Grid, placeX, placeY, btype, botIdx, ss.Tick)
}

// BotRemoveBlock removes a block at the bot's current forward position.
func BotRemoveBlock(ss *SwarmState, botIdx int) bool {
	st := ss.Stigmergy
	if st == nil || st.Grid == nil {
		return false
	}
	bot := &ss.Bots[botIdx]

	removeX := bot.X + math.Cos(bot.Angle)*20
	removeY := bot.Y + math.Sin(bot.Angle)*20

	return RemoveBlock(st.Grid, removeX, removeY)
}

// BlocksNearBot counts blocks within a radius of a bot.
func BlocksNearBot(sg *StigmergyGrid, x, y, radius float64) int {
	if sg == nil {
		return 0
	}
	count := 0
	r2 := radius * radius
	for _, b := range sg.Blocks {
		if b.Integrity <= 0 {
			continue
		}
		dx := b.X - x
		dy := b.Y - y
		if dx*dx+dy*dy < r2 {
			count++
		}
	}
	return count
}

// NearestBlockAngle returns the angle to the nearest block from a position.
func NearestBlockAngle(sg *StigmergyGrid, x, y, maxDist float64) (float64, bool) {
	if sg == nil {
		return 0, false
	}
	bestDist := maxDist * maxDist
	bestAngle := 0.0
	found := false
	for _, b := range sg.Blocks {
		if b.Integrity <= 0 {
			continue
		}
		dx := b.X - x
		dy := b.Y - y
		d2 := dx*dx + dy*dy
		if d2 < bestDist {
			bestDist = d2
			bestAngle = math.Atan2(dy, dx)
			found = true
		}
	}
	return bestAngle, found
}

// ActiveBlockCount returns the number of blocks with positive integrity.
func ActiveBlockCount(sg *StigmergyGrid) int {
	if sg == nil {
		return 0
	}
	count := 0
	for _, b := range sg.Blocks {
		if b.Integrity > 0 {
			count++
		}
	}
	return count
}

// InitStigmergy initializes the stigmergy building system.
func InitStigmergy(ss *SwarmState) {
	sg := NewStigmergyGrid(ss.ArenaW, ss.ArenaH, len(ss.Bots))

	// Create material sources at random positions
	sources := make([]MaterialSource, 4)
	margin := 60.0
	for i := range sources {
		sources[i] = MaterialSource{
			X:         margin + ss.Rng.Float64()*(ss.ArenaW-2*margin),
			Y:         margin + ss.Rng.Float64()*(ss.ArenaH-2*margin),
			Supply:    20,
			MaxSupply: 20,
		}
	}

	ss.Stigmergy = &StigmergyState{
		Grid:            sg,
		MaterialSources: sources,
	}
	logger.Info("STIGMERGY", "Initialisiert: %dx%d Grid, %d Material-Quellen",
		sg.GridCols, sg.GridRows, len(sources))
}

// ClearStigmergy disables the stigmergy system.
func ClearStigmergy(ss *SwarmState) {
	ss.Stigmergy = nil
	ss.StigmergyOn = false
}
