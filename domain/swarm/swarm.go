package swarm

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"swarmsim/domain/physics"
	"swarmsim/engine/swarmscript"
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
	}

	// Set up presets
	ss.PresetNames = []string{
		"Aggregation", "Dispersion", "Orbit", "Color Wave", "Flocking",
		"Snake Formation", "Obstacle Nav", "Pulse Sync", "Trail Follow", "Ant Colony",
		"Simple Delivery", "Delivery Comm", "Delivery Roles",
	}
	ss.PresetPrograms = []string{
		presetAggregation, presetDispersion, presetOrbit, presetColorWave, presetFlocking,
		presetSnakeFormation, presetObstacleNav, presetPulseSync, presetTrailFollow, presetAntColony,
		presetSimpleDelivery, presetDeliveryComm, presetDeliveryRoles,
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
	// Reset delivery packages
	if ss.DeliveryOn {
		ss.resetDeliveryPackages()
	}
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
func GenerateSwarmObstacles(ss *SwarmState) {
	count := 10 + ss.Rng.Intn(6)
	margin := 40.0
	ss.Obstacles = make([]*physics.Obstacle, 0, count)
	for i := 0; i < count; i++ {
		w := 30 + ss.Rng.Float64()*50
		h := 30 + ss.Rng.Float64()*50
		x := margin + ss.Rng.Float64()*(ss.ArenaW-2*margin-w)
		y := margin + ss.Rng.Float64()*(ss.ArenaH-2*margin-h)
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

	// Convert to wall obstacles — only internal walls
	for c := 0; c < mazeCols; c++ {
		for r := 0; r < mazeRows; r++ {
			x := float64(c) * cellW
			y := float64(r) * cellH
			// Right wall (vertical)
			if c < mazeCols-1 && cells[c][r].walls[1] {
				ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{
					X: x + cellW - wallThick/2, Y: y, W: wallThick, H: cellH,
				})
			}
			// Bottom wall (horizontal)
			if r < mazeRows-1 && cells[c][r].walls[2] {
				ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{
					X: x, Y: y + cellH - wallThick/2, W: cellW, H: wallThick,
				})
			}
		}
	}

	// Add border walls
	ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: 0, Y: 0, W: ss.ArenaW, H: wallThick})                     // top
	ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: 0, Y: ss.ArenaH - wallThick, W: ss.ArenaW, H: wallThick}) // bottom
	ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: 0, Y: 0, W: wallThick, H: ss.ArenaH})                     // left
	ss.MazeWalls = append(ss.MazeWalls, &physics.Obstacle{X: ss.ArenaW - wallThick, Y: 0, W: wallThick, H: ss.ArenaH}) // right
}

// GenerateDeliveryStations places 4 pickup + 4 dropoff stations in the arena.
// Pickup and dropoff of the same color must be at least 300px apart.
func GenerateDeliveryStations(ss *SwarmState) {
	ss.Stations = nil
	ss.Packages = nil
	ss.DeliveryStats = DeliveryStats{}

	colors := []int{1, 2, 3, 4} // red, blue, yellow, green
	margin := 60.0
	stationRadius := 25.0

	// Helper: check if position overlaps any wall
	posOK := func(x, y float64) bool {
		for _, obs := range ss.AllObstacles() {
			if x+stationRadius > obs.X && x-stationRadius < obs.X+obs.W &&
				y+stationRadius > obs.Y && y-stationRadius < obs.Y+obs.H {
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
# --- Explore (lowest priority) ---
IF rnd < 6 THEN TURN_RANDOM
IF true THEN FWD
# --- Separation ---
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF near_dist < 15 THEN FWD
# --- LED gradient + broadcast ---
IF d_dist < 200 THEN LED_DROPOFF
IF d_dist > 200 THEN COPY_LED
IF has_pkg == 1 THEN SEND_PICKUP 1
IF d_dist < 200 THEN SEND_DROPOFF 1
# --- Pickup ---
IF carry == 0 AND p_dist < 20 THEN PICKUP
IF has_pkg == 1 THEN GOTO_PICKUP
IF has_pkg == 1 THEN FWD
IF heard_pickup > 0 THEN GOTO_HEARD_PICKUP
IF heard_pickup > 0 THEN FWD
# --- Deliver: LED/messages ---
IF led_dist < 200 THEN GOTO_LED
IF led_dist < 200 THEN FWD
IF heard_dropoff > 0 THEN GOTO_HEARD_DROPOFF
IF heard_dropoff > 0 THEN FWD
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
# --- Pickup ---
IF carry == 0 AND p_dist < 20 THEN PICKUP
IF has_pkg == 1 THEN GOTO_PICKUP
IF has_pkg == 1 THEN FWD
IF heard_pickup > 0 THEN GOTO_HEARD_PICKUP
IF heard_pickup > 0 THEN FWD
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
