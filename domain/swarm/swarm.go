package swarm

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"swarmsim/domain/physics"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
)

// NewSwarmState creates and initializes a fresh swarm scenario.
func NewSwarmState(rng *rand.Rand, botCount int) *SwarmState {
	ss := &SwarmState{
		BotCount:     botCount,
		ArenaW:       SwarmArenaSize,
		ArenaH:       SwarmArenaSize,
		Rng:          rng,
		Hash:         physics.NewSpatialHash(SwarmArenaSize, SwarmArenaSize, 30),
		ProgramName:  "Aggregation",
		BotCountText: fmt.Sprintf("%d", botCount),
		SelectedBot:  -1,
		CompareBot:   -1,
		FollowCamBot: -1,
		SwarmCamX:    SwarmArenaSize / 2,
		SwarmCamY:    SwarmArenaSize / 2,
		SwarmCamZoom: 1.0,
	}

	// Set up presets
	ss.PresetNames = []string{
		"Aggregation", "Dispersion", "Orbit", "Color Wave", "Flocking",
		"Snake Formation", "Obstacle Nav", "Pulse Sync", "Trail Follow", "Ant Colony",
		"Simple Delivery", "Delivery Comm", "Delivery Roles",
		"Simple Unload", "Coordinated Unload",
		"Evolving Delivery", "Evolving Truck",
		"Maze Explorer",
	}
	ss.PresetPrograms = []string{
		presetAggregation, presetDispersion, presetOrbit, presetColorWave, presetFlocking,
		presetSnakeFormation, presetObstacleNav, presetPulseSync, presetTrailFollow, presetAntColony,
		presetSimpleDelivery, presetDeliveryComm, presetDeliveryRoles,
		presetSimpleUnload, presetCoordinatedUnload,
		presetEvolvingDelivery, presetEvolvingTruckUnload,
		presetMazeExplorer,
	}

	// Initialize editor with default preset
	ss.Editor = &EditorState{
		Lines:      strings.Split(presetAggregation, "\n"),
		MaxVisible: 34,
		Focused:    true,
	}

	// Spawn bots
	ss.spawnBots(botCount)

	// Auto-deploy the default program
	prog, err := swarmscript.ParseSwarmScript(presetAggregation)
	if err == nil {
		ss.Program = prog
		ss.ProgramText = presetAggregation
	}

	return ss
}

// spawnBots creates bots at random positions.
func (ss *SwarmState) spawnBots(count int) {
	margin := 30.0
	ss.Bots = make([]SwarmBot, count)
	for i := range ss.Bots {
		startX := margin + ss.Rng.Float64()*(ss.ArenaW-2*margin)
		startY := margin + ss.Rng.Float64()*(ss.ArenaH-2*margin)
		ss.Bots[i] = SwarmBot{
			X:                   startX,
			Y:                   startY,
			Angle:               ss.Rng.Float64() * 2 * math.Pi,
			LEDColor:            [3]uint8{255, 255, 255},
			FollowTargetIdx:     -1,
			FollowerIdx:         -1,
			ObstacleDist:        999,
			NearestIdx:          -1,
			StuckPrevX:          startX,
			StuckPrevY:          startY,
			CarryingPkg:         -1,
			NearestPickupDist:   999,
			NearestDropoffDist:  999,
			NearestPickupIdx:    -1,
			NearestDropoffIdx:   -1,
			NearestMatchLEDDist: 999,
		}
	}
	ss.BotCount = count
}

// RespawnBots creates a new set of bots with the given count.
func (ss *SwarmState) RespawnBots(count int) {
	if count < SwarmMinBots {
		count = SwarmMinBots
	}
	if count > SwarmMaxBots {
		count = SwarmMaxBots
	}
	ss.spawnBots(count)
	ss.BotCountText = fmt.Sprintf("%d", count)
	ss.Tick = 0
	ss.PrevMessages = nil
	ss.NextMessages = nil
}

// ResetBots resets all bot positions and internal state but keeps bot count.
func (ss *SwarmState) ResetBots() {
	margin := 30.0
	for i := range ss.Bots {
		ss.Bots[i].X = margin + ss.Rng.Float64()*(ss.ArenaW-2*margin)
		ss.Bots[i].Y = margin + ss.Rng.Float64()*(ss.ArenaH-2*margin)
		ss.Bots[i].Angle = ss.Rng.Float64() * 2 * math.Pi
		ss.Bots[i].Speed = 0
		ss.Bots[i].LEDColor = [3]uint8{255, 255, 255}
		ss.Bots[i].State = 0
		ss.Bots[i].Counter = 0
		ss.Bots[i].Value1 = 0
		ss.Bots[i].Value2 = 0
		ss.Bots[i].Timer = 0
		ss.Bots[i].PendingMsg = 0
		ss.Bots[i].BlinkTimer = 0
		ss.Bots[i].ReceivedMsg = 0
		ss.Bots[i].FollowTargetIdx = -1
		ss.Bots[i].FollowerIdx = -1
		ss.Bots[i].NearestLEDR = 0
		ss.Bots[i].NearestLEDG = 0
		ss.Bots[i].NearestLEDB = 0
		ss.Bots[i].ObstacleAhead = false
		ss.Bots[i].ObstacleDist = 999
		ss.Bots[i].NearestIdx = -1
		ss.Bots[i].StuckTicks = 0
		ss.Bots[i].StuckPrevX = ss.Bots[i].X
		ss.Bots[i].StuckPrevY = ss.Bots[i].Y
		ss.Bots[i].StuckCooldown = 0
		ss.Bots[i].Trail = [10][2]float64{}
		ss.Bots[i].TrailIdx = 0
		ss.Bots[i].CarryingPkg = -1
		ss.Bots[i].NearestPickupDist = 999
		ss.Bots[i].NearestDropoffDist = 999
		ss.Bots[i].NearestPickupIdx = -1
		ss.Bots[i].NearestDropoffIdx = -1
		ss.Bots[i].NearestMatchLEDDist = 999
		ss.Bots[i].NearestMatchLEDAngle = 0
	}
	ss.Tick = 0
	ss.PrevMessages = nil
	ss.NextMessages = nil
	// Reset delivery state (packages, stats, station timers)
	ss.ResetDeliveryState()
}

// ResetDeliveryState resets all delivery counters, package states, and bot
// carrying state. Called on DEPLOY and preset switch to start with a clean slate.
func (ss *SwarmState) ResetDeliveryState() {
	if !ss.DeliveryOn {
		return
	}
	// Clear bot carrying state and delivery sensor caches
	for i := range ss.Bots {
		ss.Bots[i].CarryingPkg = -1
		ss.Bots[i].NearestPickupDist = 999
		ss.Bots[i].NearestDropoffDist = 999
		ss.Bots[i].NearestPickupIdx = -1
		ss.Bots[i].NearestDropoffIdx = -1
		ss.Bots[i].NearestPickupColor = 0
		ss.Bots[i].NearestDropoffColor = 0
		ss.Bots[i].NearestPickupHasPkg = false
		ss.Bots[i].DropoffMatch = false
		ss.Bots[i].HeardPickupColor = 0
		ss.Bots[i].HeardDropoffColor = 0
		ss.Bots[i].NearestMatchLEDDist = 999
		ss.Bots[i].NearestMatchLEDAngle = 0
	}
	ss.resetDeliveryPackages()
}

// resetDeliveryPackages resets all packages to their initial state.
func (ss *SwarmState) resetDeliveryPackages() {
	ss.Packages = nil
	ss.DeliveryStats = DeliveryStats{}
	for i := range ss.Stations {
		if ss.Stations[i].IsPickup {
			ss.Stations[i].HasPackage = true
			ss.Stations[i].RespawnIn = 0
			ss.Stations[i].FlashTimer = 0
			pkg := DeliveryPackage{
				Color:     ss.Stations[i].Color,
				CarriedBy: -1,
				X:         ss.Stations[i].X,
				Y:         ss.Stations[i].Y,
				Active:    true,
			}
			ss.Packages = append(ss.Packages, pkg)
		}
	}
}

// --- Maze & Obstacle Generation ---

// GenerateSwarmObstacles creates 10-15 random rectangular obstacles.
// When TruckToggle is active, obstacles that overlap the ramp area are skipped.
func GenerateSwarmObstacles(ss *SwarmState) {
	count := 10 + ss.Rng.Intn(6)
	margin := 40.0
	ss.Obstacles = make([]*physics.Obstacle, 0, count)
	rampMargin := 30.0
	for i := 0; i < count; i++ {
		w := 30 + ss.Rng.Float64()*50
		h := 30 + ss.Rng.Float64()*50
		x := margin + ss.Rng.Float64()*(ss.ArenaW-2*margin-w)
		y := margin + ss.Rng.Float64()*(ss.ArenaH-2*margin-h)
		// Skip obstacles overlapping ramp area when trucks are active
		if ss.TruckToggle {
			if x < SwarmRampX+SwarmRampW+rampMargin && x+w > SwarmRampX-rampMargin &&
				y < SwarmRampY+SwarmRampH+rampMargin && y+h > SwarmRampY-rampMargin {
				continue
			}
		}
		ss.Obstacles = append(ss.Obstacles, &physics.Obstacle{X: x, Y: y, W: w, H: h})
	}
}

// GenerateSwarmMaze creates an 8x8 maze using recursive backtracker.
func GenerateSwarmMaze(ss *SwarmState) {
	mazeCols := 8
	mazeRows := 8
	cellW := ss.ArenaW / float64(mazeCols)
	cellH := ss.ArenaH / float64(mazeRows)
	wallThick := 6.0
	ss.MazeWalls = nil

	type cell struct {
		visited bool
		walls   [4]bool // top, right, bottom, left
	}

	cells := make([][]cell, mazeCols)
	for c := range cells {
		cells[c] = make([]cell, mazeRows)
		for r := range cells[c] {
			cells[c][r].walls = [4]bool{true, true, true, true}
		}
	}

	type pos struct{ c, r int }
	dirs := [4]pos{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	opp := [4]int{2, 3, 0, 1}

	stack := []pos{{0, 0}}
	cells[0][0].visited = true

	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		var nbrs []int
		for d, dir := range dirs {
			nc, nr := curr.c+dir.c, curr.r+dir.r
			if nc >= 0 && nc < mazeCols && nr >= 0 && nr < mazeRows && !cells[nc][nr].visited {
				nbrs = append(nbrs, d)
			}
		}
		if len(nbrs) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		d := nbrs[ss.Rng.Intn(len(nbrs))]
		nc, nr := curr.c+dirs[d].c, curr.r+dirs[d].r
		cells[curr.c][curr.r].walls[d] = false
		cells[nc][nr].walls[opp[d]] = false
		cells[nc][nr].visited = true
		stack = append(stack, pos{nc, nr})
	}

	// Ramp exclusion zone (with margin for bots to pass through)
	rampMargin := 20.0
	rampExclX := SwarmRampX
	rampExclY := SwarmRampY - rampMargin
	rampExclW := SwarmRampW + rampMargin
	rampExclH := SwarmRampH + 2*rampMargin

	wallOverlapsRamp := func(wx, wy, ww, wh float64) bool {
		if !ss.TruckToggle {
			return false
		}
		return wx < rampExclX+rampExclW && wx+ww > rampExclX &&
			wy < rampExclY+rampExclH && wy+wh > rampExclY
	}

	// Convert to wall obstacles — only internal walls (skip those overlapping ramp)
	for c := 0; c < mazeCols; c++ {
		for r := 0; r < mazeRows; r++ {
			x := float64(c) * cellW
			y := float64(r) * cellH
			// Right wall (vertical)
			if c < mazeCols-1 && cells[c][r].walls[1] {
				wx := x + cellW - wallThick/2
				if !wallOverlapsRamp(wx, y, wallThick, cellH) {
					ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{
						X: wx, Y: y, W: wallThick, H: cellH,
					})
				}
			}
			// Bottom wall (horizontal)
			if r < mazeRows-1 && cells[c][r].walls[2] {
				wy := y + cellH - wallThick/2
				if !wallOverlapsRamp(x, wy, cellW, wallThick) {
					ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{
						X: x, Y: wy, W: cellW, H: wallThick,
					})
				}
			}
		}
	}

	// Add border walls (split left wall to leave gap for ramp)
	ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: 0, Y: 0, W: ss.ArenaW, H: wallThick})                     // top
	ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: 0, Y: ss.ArenaH - wallThick, W: ss.ArenaW, H: wallThick}) // bottom
	if ss.TruckToggle {
		// Left wall with gap for ramp entrance
		if SwarmRampY > wallThick {
			ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: 0, Y: 0, W: wallThick, H: SwarmRampY})
		}
		rampBottom := SwarmRampY + SwarmRampH
		if rampBottom < ss.ArenaH-wallThick {
			ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: 0, Y: rampBottom, W: wallThick, H: ss.ArenaH - rampBottom})
		}
	} else {
		ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: 0, Y: 0, W: wallThick, H: ss.ArenaH}) // left
	}
	ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: ss.ArenaW - wallThick, Y: 0, W: wallThick, H: ss.ArenaH}) // right
}

// GenerateDeliveryStations places 4 pickup + 4 dropoff stations in the arena.
// Pickup and dropoff of the same color must be at least 300px apart.
func GenerateDeliveryStations(ss *SwarmState) {
	ss.Stations = nil
	ss.Packages = nil
	ss.DeliveryStats = DeliveryStats{}

	// Initialize pheromone grid if not already present
	if ss.PherGrid == nil {
		ss.PherGrid = NewSwarmPheromoneGrid(ss.ArenaW, ss.ArenaH)
	} else {
		ss.PherGrid.Clear()
	}

	colors := []int{1, 2, 3, 4} // red, blue, yellow, green
	margin := 60.0
	stationRadius := 25.0

	// Helper: check if position overlaps any wall or ramp
	posOK := func(x, y float64) bool {
		for _, obs := range ss.AllObstacles() {
			if x+stationRadius > obs.X && x-stationRadius < obs.X+obs.W &&
				y+stationRadius > obs.Y && y-stationRadius < obs.Y+obs.H {
				return false
			}
		}
		// Don't place stations inside the ramp area when trucks active
		if ss.TruckToggle {
			rampPad := 40.0
			if x+stationRadius > SwarmRampX-rampPad && x-stationRadius < SwarmRampX+SwarmRampW+rampPad &&
				y+stationRadius > SwarmRampY-rampPad && y-stationRadius < SwarmRampY+SwarmRampH+rampPad {
				return false
			}
		}
		return true
	}

	// Helper: distance between two points
	dist := func(x1, y1, x2, y2 float64) float64 {
		dx := x1 - x2
		dy := y1 - y2
		return math.Sqrt(dx*dx + dy*dy)
	}

	// Place pickups first
	var pickups []DeliveryStation
	for _, c := range colors {
		for attempts := 0; attempts < 200; attempts++ {
			x := margin + ss.Rng.Float64()*(ss.ArenaW-2*margin)
			y := margin + ss.Rng.Float64()*(ss.ArenaH-2*margin)
			if !posOK(x, y) {
				continue
			}
			// Check distance to existing stations
			tooClose := false
			for _, s := range pickups {
				if dist(x, y, s.X, s.Y) < 100 {
					tooClose = true
					break
				}
			}
			if tooClose {
				continue
			}
			pickups = append(pickups, DeliveryStation{
				X: x, Y: y, Color: c, IsPickup: true, HasPackage: true,
			})
			break
		}
	}

	// Place dropoffs (min 300px from same-color pickup)
	var dropoffs []DeliveryStation
	for _, c := range colors {
		// Find same-color pickup
		var pickupX, pickupY float64
		for _, p := range pickups {
			if p.Color == c {
				pickupX, pickupY = p.X, p.Y
				break
			}
		}
		for attempts := 0; attempts < 200; attempts++ {
			x := margin + ss.Rng.Float64()*(ss.ArenaW-2*margin)
			y := margin + ss.Rng.Float64()*(ss.ArenaH-2*margin)
			if !posOK(x, y) {
				continue
			}
			if dist(x, y, pickupX, pickupY) < 300 {
				continue
			}
			// Check distance to other stations
			tooClose := false
			for _, s := range pickups {
				if dist(x, y, s.X, s.Y) < 80 {
					tooClose = true
					break
				}
			}
			for _, s := range dropoffs {
				if dist(x, y, s.X, s.Y) < 80 {
					tooClose = true
					break
				}
			}
			if tooClose {
				continue
			}
			dropoffs = append(dropoffs, DeliveryStation{
				X: x, Y: y, Color: c, IsPickup: false,
			})
			break
		}
	}

	ss.Stations = append(pickups, dropoffs...)

	// Spawn initial packages
	for i, st := range ss.Stations {
		if st.IsPickup && st.HasPackage {
			ss.Packages = append(ss.Packages, DeliveryPackage{
				Color:     st.Color,
				CarriedBy: -1,
				X:         st.X,
				Y:         st.Y,
				Active:    true,
			})
			_ = i
		}
	}
}

// DeliveryColorName returns the name of a delivery color.
func DeliveryColorName(c int) string {
	switch c {
	case 1:
		return "Red"
	case 2:
		return "Blue"
	case 3:
		return "Yellow"
	case 4:
		return "Green"
	}
	return "?"
}

// IsDeliveryPresetIdx returns true if the preset index is a delivery program (10-12, 15, 17).
func IsDeliveryPresetIdx(idx int) bool {
	return idx >= 10 && idx <= 12 || idx == 15 || idx == 17 // idx 15 = Evolving Delivery, 17 = Maze Explorer
}

// IsTruckPresetIdx returns true if the preset index is a truck program (13-14).
func IsTruckPresetIdx(idx int) bool {
	return idx >= 13 && idx <= 14 || idx == 16 // idx 16 = Evolving Truck
}

// IsEvolutionPresetIdx returns true for evolution presets (idx 15-16).
func IsEvolutionPresetIdx(idx int) bool {
	return idx >= 15 && idx <= 16
}

// ScanUsedParams scans the current program and sets ss.UsedParams for each $A-$Z found.
func ScanUsedParams(ss *SwarmState) {
	ss.UsedParams = [26]bool{}
	if ss.Program == nil {
		return
	}
	for _, rule := range ss.Program.Rules {
		for _, cond := range rule.Conditions {
			if cond.IsParamRef {
				ss.UsedParams[cond.ParamIdx] = true
			}
		}
	}
}

// GetParamHint returns the hint value for a given parameter index from the program.
func GetParamHint(ss *SwarmState, paramIdx int) float64 {
	if ss.Program == nil {
		return 0
	}
	for _, rule := range ss.Program.Rules {
		for _, cond := range rule.Conditions {
			if cond.IsParamRef && cond.ParamIdx == paramIdx {
				return cond.ParamHint
			}
		}
	}
	return 0
}

// NewSwarmTruckState creates a fresh truck unloading round.
func NewSwarmTruckState(rng *rand.Rand) *SwarmTruckState {
	ts := &SwarmTruckState{
		TrucksPerRound: 999, // effectively infinite trucks per round
		RoundNum:       1,
		TruckNum:       0,
		RampX:          SwarmRampX,
		RampY:          SwarmRampY,
		RampW:          SwarmRampW,
		RampH:          SwarmRampH,
	}
	ts.SpawnNextTruck(rng)
	return ts
}

// SpawnNextTruck creates the next truck in the round sequence.
func (ts *SwarmTruckState) SpawnNextTruck(rng *rand.Rand) {
	ts.TruckNum++
	truckType := rng.Intn(3) // 0=Small, 1=Medium, 2=Large
	ts.CurrentTruck = NewSwarmTruck(rng, truckType)
}

// NewSwarmTruck creates a truck with packages at the off-screen start position.
func NewSwarmTruck(rng *rand.Rand, truckType int) *SwarmTruck {
	pkgCounts := [3]int{6, 8, 10}
	count := pkgCounts[truckType]

	t := &SwarmTruck{
		X:         -120,
		Y:         SwarmRampY + SwarmRampH/2 - 20,
		Phase:     TruckDrivingIn,
		TruckType: truckType,
		Packages:  make([]TruckPackage, count),
	}

	// Layout packages in a 2-row grid on the cargo area
	for i := 0; i < count; i++ {
		col := i / 2
		row := i % 2
		t.Packages[i] = TruckPackage{
			Color: 1 + rng.Intn(4), // 1-4
			RelX:  float64(20 + col*12),
			RelY:  float64(5 + row*14),
		}
	}

	return t
}

// UpdateSwarmTruck advances the truck animation state machine.
func UpdateSwarmTruck(ss *SwarmState) {
	ts := ss.TruckState
	if ts == nil || ts.CurrentTruck == nil {
		return
	}
	t := ts.CurrentTruck

	switch t.Phase {
	case TruckDrivingIn:
		t.X += 2
		if t.X >= SwarmTruckParkX {
			t.X = SwarmTruckParkX
			t.Phase = TruckParked
		}

	case TruckParked:
		// Check if all packages picked up
		allPicked := true
		for i := range t.Packages {
			if !t.Packages[i].PickedUp {
				allPicked = false
				break
			}
		}
		if allPicked {
			t.Phase = TruckComplete
			t.PhaseTimer = 20 // brief celebration before departure
		}

	case TruckComplete:
		t.PhaseTimer--
		if t.PhaseTimer <= 0 {
			t.Phase = TruckDrivingOut
			logger.Info("TRUCK", "Truck departing (all %d packages picked)", len(t.Packages))
		}

	case TruckDrivingOut:
		t.X -= 3 // drive back LEFT (the way it came)
		if t.X < -120 {
			if ts.TruckNum >= ts.TrucksPerRound {
				t.Phase = TruckRoundDone
			} else {
				t.Phase = TruckWaiting
				t.PhaseTimer = 1 // next truck arrives immediately
			}
		}

	case TruckWaiting:
		t.PhaseTimer--
		if t.PhaseTimer <= 0 {
			ts.SpawnNextTruck(ss.Rng)
		}

	case TruckRoundDone:
		// Wait for N key
	}
}

// NeighborDelta computes dx, dy from (ax,ay) to (bx,by) with optional wrap-mode.
func NeighborDelta(ax, ay, bx, by float64, ss *SwarmState) (float64, float64) {
	dx := bx - ax
	dy := by - ay
	if ss.WrapMode {
		if dx > ss.ArenaW/2 {
			dx -= ss.ArenaW
		} else if dx < -ss.ArenaW/2 {
			dx += ss.ArenaW
		}
		if dy > ss.ArenaH/2 {
			dy -= ss.ArenaH
		} else if dy < -ss.ArenaH/2 {
			dy += ss.ArenaH
		}
	}
	return dx, dy
}

// --- Preset Programs ---

var presetAggregation = `# Bots cluster together
# Speed resets to 0 each tick
IF neighbors == 0 THEN FWD
IF neighbors == 0 AND rnd < 5 THEN TURN_RANDOM
IF near_dist > 30 THEN TURN_TO_NEAREST
IF near_dist > 30 THEN FWD
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF near_dist < 15 THEN FWD
IF edge == 1 THEN TURN_RIGHT 180`

var presetDispersion = `# Bots spread out evenly
IF near_dist < 40 THEN TURN_FROM_NEAREST
IF near_dist < 40 THEN FWD
IF neighbors == 0 THEN TURN_RANDOM
IF neighbors == 0 THEN FWD
IF edge == 1 THEN TURN_RIGHT 180`

var presetOrbit = `# Bots orbit the light source (L key)
IF light > 80 THEN TURN_RIGHT 90
IF light > 80 THEN FWD
IF light < 30 THEN TURN_TO_LIGHT
IF light < 30 THEN FWD
IF near_dist < 12 THEN TURN_FROM_NEAREST
IF near_dist < 12 THEN FWD
IF true THEN FWD`

var presetColorWave = `# Red flash wave through swarm
IF state == 0 AND msg == 1 THEN SET_STATE 1
IF state == 0 AND msg == 1 THEN SET_LED 255 0 0
IF state == 0 AND msg == 1 THEN SEND_MESSAGE 1
IF state == 0 AND msg == 1 THEN SET_TIMER 60
IF state == 1 AND timer == 0 THEN SET_STATE 0
IF state == 1 AND timer == 0 THEN SET_LED 60 60 60
IF state == 0 AND rnd < 1 THEN SEND_MESSAGE 1
IF state == 0 AND rnd < 1 THEN SET_STATE 1
IF state == 0 AND rnd < 1 THEN SET_LED 255 0 0
IF state == 0 AND rnd < 1 THEN SET_TIMER 60
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF near_dist < 15 THEN FWD`

var presetFlocking = `# Boids-like flocking behavior
IF near_dist < 12 THEN TURN_FROM_NEAREST
IF near_dist < 12 THEN FWD
IF near_dist > 40 THEN TURN_TO_CENTER
IF near_dist > 40 THEN FWD
IF neighbors == 0 THEN TURN_RANDOM
IF edge == 1 THEN TURN_RIGHT 180
IF true THEN FWD`

var presetSnakeFormation = `# Bots form chains and slither
# state: 0=lone 1=head 2=follower
IF leader == 0 AND follower == 0 THEN SET_STATE 0
IF leader == 0 AND follower == 1 THEN SET_STATE 1
IF leader == 1 THEN SET_STATE 2
# Lone bot: search for chain
IF state == 0 THEN FWD
IF state == 0 AND rnd < 5 THEN TURN_RANDOM
IF state == 0 THEN SET_LED 255 255 0
IF state == 0 AND nbrs > 0 THEN FOLLOW_NEAREST
# Chain head: steer the snake
IF state == 1 THEN SET_LED 0 255 0
IF state == 1 THEN FWD
IF state == 1 AND rnd < 3 THEN TURN_RIGHT 15
IF state == 1 AND rnd < 3 THEN TURN_LEFT 15
# Head also merges with other chains
IF state == 1 AND nbrs > 0 THEN FOLLOW_NEAREST
# Followers: blue
IF state == 2 THEN SET_LED 100 100 255
# Tail trim
IF chain_len > 12 AND follower == 0 THEN UNFOLLOW
# Environment avoidance
IF edge == 1 THEN TURN_RIGHT 180
IF obs_ahead == 1 THEN AVOID_OBSTACLE`

var presetObstacleNav = `# Navigate obstacles toward light
# Enable Obstacles or Maze + Light!
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF obs_ahead == 1 THEN FWD_SLOW
IF obs_ahead == 1 THEN SET_LED 255 0 0
IF light > 0 THEN TURN_TO_LIGHT
IF obs_ahead == 0 THEN SET_LED 0 255 100
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF edge == 1 THEN TURN_RIGHT 180
IF rnd < 3 THEN TURN_RIGHT 10
IF true THEN FWD`

var presetPulseSync = `# Synchronized LED pulses (fireflies)
IF state == 0 THEN SET_TIMER 60
IF state == 0 THEN SET_STATE 1
IF state == 0 THEN SET_LED 20 20 20
IF state == 1 AND timer == 0 THEN SET_LED 255 99 0
IF state == 1 AND timer == 0 THEN SEND_MESSAGE 1
IF state == 1 AND timer == 0 THEN SET_STATE 2
IF state == 1 AND timer == 0 THEN SET_TIMER 5
IF state == 2 AND timer == 0 THEN SET_LED 20 20 20
IF state == 2 AND timer == 0 THEN SET_TIMER 55
IF state == 2 AND timer == 0 THEN SET_STATE 1
IF state == 1 AND msg == 1 THEN SET_TIMER 40
IF near_dist < 12 THEN TURN_FROM_NEAREST
IF near_dist < 12 THEN FWD_SLOW
IF rnd < 2 THEN TURN_RANDOM`

var presetTrailFollow = `# Copy nearest neighbor LED color
IF near_dist > 25 THEN TURN_TO_NEAREST
IF near_dist > 25 THEN FWD
IF neighbors > 0 THEN COPY_LED
IF neighbors == 0 THEN SET_LED 255 255 255
IF neighbors == 0 THEN TURN_RANDOM
IF neighbors == 0 THEN FWD
IF near_dist < 12 THEN TURN_FROM_NEAREST
IF near_dist < 12 THEN FWD
IF edge == 1 THEN TURN_RIGHT 180
IF rnd < 1 THEN SET_LED 255 0 0`

var presetAntColony = `# Ant foraging (use Light!)
# State 0=search 1=return 2=base
IF state == 0 THEN FWD
IF state == 0 AND rnd < 5 THEN TURN_RANDOM
IF state == 0 AND light > 50 THEN SET_STATE 1
IF state == 0 AND light > 50 THEN SET_LED 0 255 0
IF state == 0 AND light > 50 THEN TURN_RIGHT 180
IF state == 1 THEN FWD
IF state == 1 THEN SEND_MESSAGE 2
IF state == 1 AND edge == 1 THEN SET_STATE 2
IF state == 1 AND edge == 1 THEN SET_LED 99 99 99
IF state == 1 AND edge == 1 THEN SET_TIMER 30
IF state == 2 AND timer == 0 THEN SET_STATE 0
IF state == 2 AND timer == 0 THEN SET_LED 255 99 0
IF state == 2 AND timer == 0 THEN TURN_RIGHT 180
IF state == 0 AND msg == 2 THEN TURN_TO_NEAREST
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF obs_ahead == 1 THEN AVOID_OBSTACLE`

var presetSimpleDelivery = `# Smart Delivery — enable Delivery!
# LED gradient, separation, deliver
# --- Explore (lowest priority) ---
IF rnd < 8 THEN TURN_RANDOM
IF true THEN FWD
# --- Separation ---
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF near_dist < 15 THEN FWD
# --- LED gradient ---
IF d_dist < 200 THEN LED_DROPOFF
IF d_dist > 200 THEN COPY_LED
# --- Pickup (pkg available only) ---
IF carry == 0 AND p_dist < 20 THEN PICKUP
IF has_pkg == 1 THEN GOTO_PICKUP
IF has_pkg == 1 THEN FWD
# --- Deliver: LED gradient ---
IF led_dist < 200 THEN GOTO_LED
IF led_dist < 200 THEN FWD
# --- Deliver: direct ---
IF match == 1 AND d_dist < 25 THEN DROP
IF match == 1 THEN GOTO_DROPOFF
IF match == 1 THEN FWD
# --- Navigation LAST ---
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF edge == 1 THEN TURN_RIGHT 180`

var presetDeliveryComm = `# Delivery+Comm — enable Delivery!
# Messages AND LED gradient
# --- Anti-cluster: idle bots leave empty stations ---
IF carry == 0 AND p_dist < 30 AND has_pkg == 0 THEN SET_LED 255 255 255
IF carry == 0 AND p_dist < 30 AND has_pkg == 0 THEN TURN_FROM_NEAREST
# --- Explore (lowest priority) ---
IF rnd < 6 THEN TURN_RANDOM
IF true THEN FWD
# --- Separation ---
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF near_dist < 15 THEN FWD
# --- LED gradient + broadcast ---
IF d_dist < 200 THEN LED_DROPOFF
IF carry == 1 AND d_dist > 200 THEN COPY_LED
IF has_pkg == 1 THEN SEND_PICKUP 1
IF d_dist < 200 THEN SEND_DROPOFF 1
# --- Pickup (only non-carrying) ---
IF carry == 0 AND p_dist < 20 THEN PICKUP
IF carry == 0 AND has_pkg == 1 THEN GOTO_PICKUP
IF carry == 0 AND has_pkg == 1 THEN FWD
IF carry == 0 AND heard_pickup > 0 THEN GOTO_HEARD_PICKUP
IF carry == 0 AND heard_pickup > 0 THEN FWD
# --- Deliver: LED/messages ---
IF led_dist < 200 THEN GOTO_LED
IF led_dist < 200 THEN FWD
IF carry == 1 AND heard_dropoff > 0 THEN GOTO_HEARD_DROPOFF
IF carry == 1 AND heard_dropoff > 0 THEN FWD
# --- Deliver: direct ---
IF match == 1 AND d_dist < 25 THEN DROP
IF match == 1 THEN GOTO_DROPOFF
IF match == 1 THEN FWD
# --- Navigation LAST ---
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF edge == 1 THEN TURN_RIGHT 180`

var presetDeliveryRoles = `# Roles+LED — enable Delivery!
# value1: 1=Beacon 2=Carrier
# --- Role assignment ---
IF value1 == 0 AND rnd < 40 THEN SET_VALUE1 1
IF value1 == 0 THEN SET_VALUE1 2
IF value1 == 1 THEN SET_STATE 1
IF value1 == 2 THEN SET_STATE 2
# --- Carrier: explore (lowest) ---
IF state == 2 AND rnd < 5 THEN TURN_RANDOM
IF state == 2 THEN FWD
IF state == 2 AND carry == 0 THEN SET_LED 90 90 90
# --- Separation (both roles) ---
IF near_dist < 15 THEN TURN_FROM_NEAREST
# --- LED gradient (both roles) ---
IF d_dist < 200 THEN LED_DROPOFF
IF d_dist > 200 THEN COPY_LED
IF has_pkg == 1 THEN SEND_PICKUP 1
IF d_dist < 200 THEN SEND_DROPOFF 1
# --- Pickup (only non-carrying) ---
IF carry == 0 AND p_dist < 20 THEN PICKUP
IF carry == 0 AND has_pkg == 1 THEN GOTO_PICKUP
IF carry == 0 AND has_pkg == 1 THEN FWD
IF carry == 0 AND heard_pickup > 0 THEN GOTO_HEARD_PICKUP
IF carry == 0 AND heard_pickup > 0 THEN FWD
# --- Beacon: explore (overrides pickup) ---
IF state == 1 AND rnd < 8 THEN TURN_RANDOM
IF state == 1 THEN FWD
# --- Deliver: LED gradient ---
IF led_dist < 200 THEN GOTO_LED
IF led_dist < 200 THEN FWD
IF state == 2 AND carry == 1 THEN SET_LED 255 99 0
IF state == 2 AND carry == 1 THEN FWD
# --- Deliver: direct ---
IF match == 1 AND d_dist < 25 THEN DROP
IF match == 1 THEN GOTO_DROPOFF
IF match == 1 THEN FWD
# --- Navigation LAST ---
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF edge == 1 THEN TURN_RIGHT 180`

var presetSimpleUnload = `# Simple Unload — enable Trucks!
# 1. Separation (prevents clustering)
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF near_dist < 15 THEN FWD
# 2. Carrying + match: deliver at correct dropoff
IF carry == 1 AND match == 1 AND d_dist < 25 THEN DROP
IF carry == 1 AND match == 1 THEN GOTO_DROPOFF
IF carry == 1 AND match == 1 THEN FWD
# 3. Carrying + hear beacon: follow beacon
IF carry == 1 AND heard_beacon == 1 THEN GOTO_BEACON
IF carry == 1 AND heard_beacon == 1 THEN FWD
# 4. Carrying + lost: spiral search
IF carry == 1 AND exploring == 1 THEN SPIRAL
IF carry == 1 THEN FWD
# 5. Not carrying: pickup from truck
IF carry == 0 AND on_ramp == 1 AND truck_here == 1 THEN PICKUP
IF carry == 0 THEN GOTO_RAMP
IF carry == 0 THEN FWD
# 6. Navigation fallback
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF edge == 1 THEN TURN_RIGHT 180
IF rnd < 5 THEN TURN_RANDOM
IF true THEN FWD`

var presetCoordinatedUnload = `# Coordinated Unload — enable Trucks!
# 1. Separation (highest priority — prevents clustering)
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF near_dist < 15 THEN FWD
# 2. Carrying + match: deliver at correct dropoff
IF carry == 1 AND match == 1 AND d_dist < 25 THEN DROP
IF carry == 1 AND match == 1 THEN GOTO_DROPOFF
IF carry == 1 AND match == 1 THEN SEND_DROPOFF 1
IF carry == 1 AND match == 1 THEN LED_DROPOFF
IF carry == 1 AND match == 1 THEN FWD
# 3. Carrying + beacon/LED/heard: navigate to dropoff
IF carry == 1 AND heard_beacon == 1 THEN GOTO_BEACON
IF carry == 1 AND heard_beacon == 1 THEN FWD
IF carry == 1 AND heard_dropoff > 0 THEN GOTO_HEARD_DROPOFF
IF carry == 1 AND heard_dropoff > 0 THEN FWD
IF carry == 1 AND led_match < 200 THEN GOTO_LED_MATCH
IF carry == 1 AND led_match < 200 THEN FWD
# 4. Carrying + lost: spiral search
IF carry == 1 AND exploring == 1 THEN SPIRAL
IF carry == 1 THEN FWD
# 5. Not carrying: pickup from truck
IF carry == 0 AND on_ramp == 1 AND truck_here == 1 THEN PICKUP
IF carry == 0 THEN GOTO_RAMP
IF carry == 0 THEN FWD
# 6. Navigation fallback
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF edge == 1 THEN TURN_RIGHT 180
IF rnd < 5 THEN TURN_RANDOM
IF true THEN FWD`

var presetEvolvingDelivery = `# Evolving Delivery — uses $A-$Z params!
# 1. Carrying + see dropoff: deliver
IF carry == 1 AND match == 1 AND d_dist < $A:25 THEN DROP
IF carry == 1 AND match == 1 THEN GOTO_DROPOFF
# 2. Carrying + hear beacon: follow
IF carry == 1 AND heard_beacon == 1 THEN GOTO_BEACON
IF carry == 1 AND heard_beacon == 1 THEN FWD
# 3. Carrying + lost: spiral
IF carry == 1 AND exploring == 1 THEN SPIRAL
IF carry == 1 THEN FWD
# 4. Not carrying: pickup
IF carry == 0 AND p_dist < $B:30 AND has_pkg == 1 THEN PICKUP
IF carry == 0 AND p_dist < $C:200 THEN GOTO_PICKUP
IF carry == 0 THEN FWD
# 5. Separation + Navigation
IF near_dist < $D:20 THEN TURN_FROM_NEAREST
IF rnd < $E:30 THEN TURN_RANDOM
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF edge == 1 THEN TURN_RIGHT 180
IF true THEN FWD`

var presetEvolvingTruckUnload = `# Evolving Truck Unload — Trucks + Evolution!
# 1. Carrying + see dropoff: deliver
IF carry == 1 AND match == 1 AND d_dist < $A:25 THEN DROP
IF carry == 1 AND match == 1 THEN GOTO_DROPOFF
# 2. Carrying + hear beacon: follow
IF carry == 1 AND heard_beacon == 1 THEN GOTO_BEACON
IF carry == 1 AND heard_beacon == 1 THEN FWD
# 3. Carrying + lost: spiral
IF carry == 1 AND exploring == 1 THEN SPIRAL
IF carry == 1 THEN FWD
# 4. Not carrying: pickup from truck
IF carry == 0 AND on_ramp == 1 AND truck_here == 1 THEN PICKUP
IF carry == 0 THEN GOTO_RAMP
IF carry == 0 THEN FWD
# 5. Separation + Navigation
IF near_dist < $B:15 THEN TURN_FROM_NEAREST
IF rnd < $C:20 THEN TURN_RANDOM
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF edge == 1 THEN TURN_RIGHT 180
IF true THEN FWD`

var presetMazeExplorer = `# Maze Explorer — right-hand wall following!
# Wall-following + Delivery im Labyrinth
# Schalte Maze + Delivery ein!
IF carry == 1 AND match == 1 AND d_dist < 25 THEN DROP
IF carry == 1 AND match == 1 THEN GOTO_DROPOFF
IF carry == 1 AND match == 1 THEN FWD
IF carry == 0 AND p_dist < 20 AND has_pkg == 1 THEN PICKUP
IF carry == 0 AND has_pkg == 1 THEN GOTO_PICKUP
IF carry == 0 AND has_pkg == 1 THEN FWD
IF near_dist < 12 THEN TURN_FROM_NEAREST
IF wall_front == 1 THEN TURN_LEFT 90
IF wall_right == 0 THEN TURN_RIGHT 90
IF true THEN FWD`
