package swarm

import (
	"math/rand"
	"swarmsim/domain/physics"
	"swarmsim/engine/swarmscript"
)

const (
	SwarmBotRadius           = 10.0
	SwarmBotSpeed            = 1.5
	SwarmSensorRange         = 60.0
	SwarmDeliverySensorRange = 200.0 // extended range for delivery station + LED scanning
	SwarmBeaconRange         = 600.0 // dropoff stations broadcast beacons to carrying bots
	SwarmCommRange           = 40.0
	SwarmArenaSize           = 800.0
	SwarmEdgeMargin          = 20.0
	SwarmDefaultBots         = 50
	SwarmMaxBots             = 500
	SwarmMinBots             = 5

	// Truck / Ramp constants
	SwarmRampX      = 0.0
	SwarmRampY      = 200.0
	SwarmRampW      = 200.0
	SwarmRampH      = 350.0
	SwarmTruckParkX = 20.0
)

// TruckAnimPhase represents the animation state of a truck.
type TruckAnimPhase int

const (
	TruckDrivingIn  TruckAnimPhase = iota // truck entering from left
	TruckParked                           // waiting for packages to be picked up
	TruckComplete                         // all packages picked, short delay
	TruckDrivingOut                       // truck leaving to the right
	TruckWaiting                          // pause between trucks
	TruckRoundDone                        // all trucks in round delivered
)

// SwarmBot is a simple programmable robot with no identity.
type SwarmBot struct {
	X, Y  float64 // world position
	Angle float64 // facing direction (radians)
	Speed float64 // current speed (0 or SwarmBotSpeed)

	LEDColor [3]uint8 // visual LED color (R, G, B)

	// Internal state variables (user-programmable)
	State   int
	Counter int
	Value1  int
	Value2  int
	Timer   int // counts down each tick when > 0

	// Per-tick sensor cache (rebuilt by environment builder)
	NeighborCount int
	NearestDist   float64
	NearestAngle  float64 // absolute angle to nearest neighbor
	AvgNeighborX  float64 // average X of neighbors (relative to bot)
	AvgNeighborY  float64 // average Y of neighbors (relative to bot)
	OnEdge        bool
	ReceivedMsg   int // 0 = none, >0 = message value
	LightValue    int // 0-100

	// Per-tick output
	PendingMsg int // 0 = no message, >0 = broadcast value
	BlinkTimer int // deploy-confirmation blink (frames countdown)

	// Follow mechanic
	FollowTargetIdx int // index of bot being followed (-1 = none)
	FollowerIdx     int // index of bot following this one (-1 = none)

	// Extended sensor cache
	NearestLEDR   uint8
	NearestLEDG   uint8
	NearestLEDB   uint8
	ObstacleAhead bool    // obstacle within 50px in facing direction
	ObstacleDist  float64 // distance to nearest obstacle ahead (999 if none)
	NearestIdx    int     // index of nearest neighbor (-1 if none)

	// Anti-stuck tracking
	StuckTicks      int     // how many ticks bot barely moved
	StuckPrevX      float64 // position last tick for stuck detection
	StuckPrevY      float64
	StuckCooldown   int // cooldown: forced solo movement after unfollow (counts down)
	AntiStuckTimer  int // >0 = breakout mode active (counts down)
	AntiStuckAngle  float64
	CloseNeighbors  int // neighbors within 30px (rebuilt per tick)

	// Trail history (ring buffer)
	Trail    [10][2]float64
	TrailIdx int

	// Delivery system
	CarryingPkg int // -1 = not carrying, otherwise package index

	// Delivery sensor cache (rebuilt per tick)
	NearestPickupDist   float64
	NearestPickupColor  int
	NearestPickupHasPkg bool
	NearestPickupIdx    int // station index
	NearestDropoffDist  float64
	NearestDropoffColor int
	NearestDropoffIdx   int // station index
	DropoffMatch        bool
	HeardPickupColor    int
	HeardPickupAngle    float64
	HeardDropoffColor   int
	HeardDropoffAngle   float64

	// LED pheromone matching (for delivery navigation gradient)
	NearestMatchLEDDist  float64 // dist to nearest bot whose LED matches carrying color
	NearestMatchLEDAngle float64 // angle to that bot

	// Beacon sensor cache (rebuilt per tick when carrying)
	HeardBeaconDropoffColor int     // color of nearest beacon-heard dropoff (0=none)
	HeardBeaconDropoffAngle float64 // angle toward it
	HeardBeaconDropoffDist  float64 // distance to it

	// Truck sensor cache (rebuilt per tick when TruckToggle)
	TruckHere           bool
	TruckPkgCount       int
	OnRamp              bool
	NearestTruckPkgDist float64
	NearestTruckPkgIdx  int // index in SwarmTruck.Packages (-1 if none)

	// Lifetime statistics (persist across sim reset, cleared on bot count change)
	Stats BotLifetimeStats
}

// BotLifetimeStats tracks cumulative performance metrics for a single bot.
type BotLifetimeStats struct {
	TotalDistance     float64 // sum of per-tick displacement in px
	TotalPickups      int
	TotalDeliveries   int
	CorrectDeliveries int
	WrongDeliveries   int
	DeliveryTimes     []int // ticks per completed delivery
	MessagesSent      int
	MessagesReceived  int
	AntiStuckCount    int
	TicksAlive        int // incremented every tick
	TicksCarrying     int // ticks while CarryingPkg >= 0
	TicksIdle         int // ticks while Speed == 0
}

// LightSource represents an optional light source in the arena.
type LightSource struct {
	Active bool
	X, Y   float64
}

// SwarmMessage is a broadcast int message with position.
type SwarmMessage struct {
	Value int
	X, Y  float64
}

// EditorState tracks the text editor's internal state.
type EditorState struct {
	Lines      []string
	CursorLine int
	CursorCol  int
	ScrollY    int // first visible line index
	ScrollX    int // first visible column index (horizontal scroll)
	Focused    bool
	BlinkTick  int // cursor blink animation counter
	MaxVisible int // number of visible lines in editor area
}

// DeliveryStation represents a pickup or dropoff station.
type DeliveryStation struct {
	X, Y         float64
	Color        int  // 1=red, 2=blue, 3=yellow, 4=green
	IsPickup     bool // true=Pickup, false=Dropoff
	HasPackage   bool // only for Pickup: is a package ready?
	RespawnIn    int  // countdown until new package spawns
	FlashTimer   int  // visual flash effect on delivery
	FlashOK      bool // true=correct delivery, false=wrong
	DeliverCount int  // total packages delivered to this dropoff
}

// DeliveryPackage represents a package in the delivery system.
type DeliveryPackage struct {
	Color      int     // 1=red, 2=blue, 3=yellow, 4=green
	CarriedBy  int     // bot index, -1 if not carried
	X, Y       float64 // position when on ground
	OnGround   bool    // true if dropped on ground (not at station)
	PickupTick int     // tick when picked up (for avg time)
	Active     bool    // false if delivered and waiting respawn
}

// DeliveryStats tracks delivery performance.
type DeliveryStats struct {
	TotalDelivered   int
	CorrectDelivered int
	WrongDelivered   int
	DeliveryTimes    []int
	ColorDelivered   [5]int // index 1-4 for each color
}

// ScorePopup is a floating score text that rises and fades.
type ScorePopup struct {
	X, Y  float64
	Text  string
	Timer int // counts down from 60
	Color [3]uint8
}

// SwarmDeliveryEvent signals a delivery to the renderer for particle effects.
type SwarmDeliveryEvent struct {
	X, Y     float64
	Color    int  // delivery color 1-4
	Correct  bool // true=matched, false=wrong
	IsPickup bool // true=pickup event, false=drop event
}

// TruckPackage is a single package on a truck.
type TruckPackage struct {
	Color    int     // 1-4
	PickedUp bool
	RelX     float64 // position relative to truck body for rendering
	RelY     float64
}

// SwarmTruck represents a truck that drives into the ramp area.
type SwarmTruck struct {
	X, Y       float64        // current position (X animates)
	Phase      TruckAnimPhase
	PhaseTimer int
	Packages   []TruckPackage
	TruckType  int // 0=Small(6), 1=Medium(8), 2=Large(10)
}

// SwarmTruckState tracks the overall truck unloading round state.
type SwarmTruckState struct {
	CurrentTruck   *SwarmTruck
	TruckNum       int // 1-based (which truck in round)
	TrucksPerRound int // 3
	RoundNum       int
	Score          int
	TotalPkgs      int
	DeliveredPkgs  int
	CorrectPkgs    int
	WrongPkgs      int
	RampX, RampY   float64 // ramp area in arena coords
	RampW, RampH   float64
}

// SwarmState holds all state for the programmable swarm scenario.
type SwarmState struct {
	Bots     []SwarmBot
	BotCount int

	ArenaW float64
	ArenaH float64

	Program     *swarmscript.SwarmProgram
	ProgramText string
	ProgramName string // active preset name or "Custom"

	Light LightSource
	Tick  int
	Rng   *rand.Rand
	Hash  *physics.SpatialHash

	PrevMessages []SwarmMessage // messages from previous tick (readable)
	NextMessages []SwarmMessage // messages being sent this tick

	Editor *EditorState

	ErrorMsg  string // parse error message (empty if no error)
	ErrorLine int    // 1-based line of error, 0 if none

	PresetNames    []string
	PresetPrograms []string
	DropdownOpen   bool
	DropdownHover  int

	BotCountText string // editable bot count field text
	BotCountEdit bool   // is bot count field focused

	// Obstacles / Maze / Environment toggles
	Obstacles   []*physics.Obstacle
	MazeWalls   []*physics.Obstacle
	ObstaclesOn bool
	MazeOn      bool
	WrapMode    bool // false=BOUNCE, true=WRAP
	ShowTrails  bool

	// Selected bot for info overlay
	SelectedBot int // -1 = none
	CompareBot  int // -1 = none, Shift+click to set second bot for comparison

	// Follow-cam
	FollowCamBot int     // bot index being followed (-1 = off)
	SwarmCamX    float64 // current camera center (arena coords)
	SwarmCamY    float64
	SwarmCamZoom float64 // current zoom level (1.0 = normal)

	// Delivery system
	DeliveryOn        bool
	IsDeliveryProgram bool // true when a delivery preset (idx 10-12) is active

	// Truck system (in Swarm Lab)
	TruckToggle    bool
	IsTruckProgram bool // true when a truck preset (idx 13-14) is active
	TruckState     *SwarmTruckState
	Stations       []DeliveryStation
	Packages       []DeliveryPackage
	DeliveryStats  DeliveryStats
	ShowRoutes     bool                 // 'C' key toggle: show pickup→dropoff route lines
	ScorePopups    []ScorePopup         // floating score text on delivery
	DeliveryEvents []SwarmDeliveryEvent // consumed by renderer for particle effects

	CollisionCount  int // obstacle collisions this tick (reset per tick)
	ResetFlashTimer int // counts down from 30 for "RESET" flash

	// Block editor
	BlockEditorActive bool
	BlockRules        []BlockRule
	ActiveDropdown    *BlockDropdown
	BlockScrollY      int
	BlockValueEdit    bool // true when editing a value field
	BlockValueRuleIdx int  // which rule's value is being edited
	BlockValueCondIdx int  // which condition's value (-1 = action param)
	BlockValueText    string
}

// BlockRule represents a single rule in the visual block editor.
type BlockRule struct {
	Conditions   []BlockCondition
	ActionName   string
	ActionParams [3]int // up to 3 params (e.g. SET_LED R G B)
}

// BlockCondition represents a single condition in a block rule.
type BlockCondition struct {
	SensorName string
	OpStr      string
	Value      int
}

// BlockDropdown tracks which dropdown is currently open in the block editor.
type BlockDropdown struct {
	RuleIdx   int    // which rule
	CondIdx   int    // which condition (-1 = action dropdown)
	FieldType string // "sensor", "op", "action"
	X, Y      int    // screen position of dropdown
	ScrollY   int    // scroll offset in dropdown
	HoverIdx  int    // hovered item index (-1 = none)
	Items     []string
}

// AllObstacles returns combined obstacles and maze walls.
func (ss *SwarmState) AllObstacles() []*physics.Obstacle {
	result := make([]*physics.Obstacle, 0, len(ss.Obstacles)+len(ss.MazeWalls))
	result = append(result, ss.Obstacles...)
	result = append(result, ss.MazeWalls...)
	return result
}
