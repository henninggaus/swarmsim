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
	SwarmDropoffBeaconRange  = 150.0 // virtual dropoff beacon broadcast range
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

// SwarmPreset pairs a preset name with its SwarmScript source code.
// This replaces the parallel PresetNames/PresetPrograms arrays to prevent
// accidental divergence (wrong name mapped to wrong program).
type SwarmPreset struct {
	Name   string
	Source string
}

// SwarmBot is a simple programmable robot with no identity.
type SwarmBot struct {
	X, Y  float64 // world position
	Angle float64 // facing direction (radians)
	Speed float64 // current speed (0 or SwarmBotSpeed)

	LEDColor [3]uint8 // visual LED color (R, G, B)

	// Energy system
	Energy float64 // 0.0 - 100.0, drains with movement, recharges at stations

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
	WallRight     bool    // wall within 25px to the right of heading
	WallLeft      bool    // wall within 25px to the left of heading
	NearestIdx    int     // index of nearest neighbor (-1 if none)
	PherAhead     float64 // pheromone intensity 20px ahead (0.0-1.0)

	// Directional sensors (90° cones around heading)
	BotAhead  int // neighbors in front cone
	BotBehind int // neighbors in rear cone
	BotLeft   int // neighbors in left cone
	BotRight  int // neighbors in right cone

	// Spatial memory (visited grid per bot)
	MemoryGrid []uint8
	MemoryCols int
	MemoryRows int

	// Cooperative sensors (sensor fusion from neighbors)
	GroupCarry int // % of neighbors that are carrying (0-100)
	GroupSpeed int // avg speed of neighbors * 100
	GroupSize  int // connected cluster size (BFS)

	// Swarm awareness sensors (computed per tick)
	SwarmCenterDist    int // distance to swarm center of mass
	SwarmSpreadSensor  int // overall swarm spread (same for all bots)
	IsolationLevel     int // 0 if near others, >0 if isolated
	ResourceGradientX  int // direction toward resources (0-359 degrees, -1=none)
	ResourceGradientY  int // magnitude of resource gradient (0-100)

	// Extended sensor cache (advanced)
	NeighborMinDist   int // distance to closest neighbor (9999 if none)
	BotCarryingCount  int // count of neighbors currently carrying
	TimeSinceDelivery int // ticks since last successful delivery
	RecentCollision   int // 1 if collided with obstacle in last 10 ticks, 0 otherwise
	CollisionTimer    int // countdown from 10 when collision detected

	// A* pathfinding sensor cache
	PathDist  int // remaining path distance (0 = no path)
	PathAngle int // angle to next waypoint relative to heading (-180..180)

	// Flocking (Boids) sensor cache
	FlockAlign     int // alignment angle diff to neighbor avg heading (-180..180)
	FlockCohesion  int // distance to neighbor center of mass (0-500)
	FlockSeparation int // separation urgency (0-100, 100=critical)

	// Dynamic Role sensor cache
	Role       int // current role: 0=none, 1=scout, 2=worker, 3=guard
	RoleDemand int // most needed role locally (1-3)

	// Quorum Sensing sensor cache
	Vote          int // current vote value (0 = no vote)
	QuorumCount   int // nearby bots with same vote
	QuorumReached int // 1 if quorum threshold met

	// Rogue Detection sensor cache
	Reputation    int // 0-100, high = trusted
	SuspectNearby int // 1 if anomalous neighbor detected

	// Lévy-Flight sensor cache
	LevyPhase int // 0=idle, 1=short walk, 2=long jump
	LevyStep  int // remaining step distance

	// Firefly Synchronization sensor cache
	FlashPhase int // oscillator phase (0-255)
	FlashSync  int // 1 if currently flashing

	// Collective Transport sensor cache
	TransportNearby int // count of heavy objects in range
	TransportCount  int // bots assisting nearest task

	// Vortex Swarming sensor cache
	VortexStrength int // local vortex rotation strength (0-100)

	// Brake mechanics
	BrakeTimer int // >0 = braking (speed ramps down over 3 ticks)

	// Anti-stuck tracking
	StuckTicks      int     // how many ticks bot barely moved
	StuckPrevX      float64 // position last tick for stuck detection
	StuckPrevY      float64
	StuckCooldown   int // cooldown: forced solo movement after unfollow (counts down)
	AntiStuckTimer  int // >0 = breakout mode active (counts down)
	AntiStuckAngle  float64
	CloseNeighbors  int // neighbors within 30px (rebuilt per tick)

	// Dash mechanics (DASH action: double-speed burst)
	DashTimer    int // >0 = dash active (counts down from 10)
	DashCooldown int // >0 = cooldown after dash (counts down from 60)

	// Scatter mechanics
	ScatterTimer    int // >0 = forced scatter mode (TURN_FROM_NEAREST + FWD, counts down)
	ScatterCooldown int // >0 = cooldown after scatter (counts down)
	IdleBoostTimer  int // >0 = idle boost active (counts down): faster speed + more turning

	// Idle exploration (carry==0 bots that stopped moving)
	IdleMoveTicks int // consecutive ticks with carry==0 AND speed<0.5
	IdleMoveTimer int // >0 = forced random exploration (counts down)

	// Trail history (ring buffer)
	Trail    [30][2]float64
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

	// Exploration state (for spiral search when carrying but lost)
	ExplorationTimer int     // ticks since last successful dropoff detection
	ExplorationAngle float64 // current spiral angle offset

	// Truck sensor cache (rebuilt per tick when TruckToggle)
	TruckHere           bool
	TruckPkgCount       int
	OnRamp              bool
	NearestTruckPkgDist float64
	NearestTruckPkgIdx  int // index in SwarmTruck.Packages (-1 if none)
	RampCooldown        int // >0 = ticks until bot can try GOTO_RAMP again

	// Evolution parameters (per-bot, used when $A-$Z syntax and EvolutionOn)
	ParamValues    [26]float64
	Fitness        float64
	DiploidGenome  *DiploidGenome // nil when sexual reproduction OFF

	// Genetic Programming (per-bot program, used when GPEnabled)
	OwnProgram       *swarmscript.SwarmProgram // nil when GP OFF
	LastMatchedRules []int                     // indices of rules that matched last tick

	// Neuroevolution (per-bot neural network, used when NeuroEnabled)
	Brain     *NeuroBrain // nil when NEURO OFF
	LSTMBrain *LSTMBrain  // nil when LSTM OFF

	// Genealogy (lineage tracking)
	BotID   int // unique genealogy identifier
	ParentA int // parent A BotID (-1 = none)
	ParentB int // parent B BotID (-1 = none)

	// Novelty Search (behavior characterization)
	Behavior BehaviorDescriptor

	// Morphological evolution (evolvable body parameters)
	Morph Morphology

	// Teams (multiplayer mode)
	Team int // 0=no team, 1=A (blue), 2=B (red)

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
	TicksAlive        int     // incremented every tick
	TicksCarrying     int     // ticks while CarryingPkg >= 0
	TicksIdle         int     // ticks while Speed == 0
	SumNeighborCount  float64 // sum of neighbor count per tick (for novelty behavior)
}

// LightSource represents an optional light source in the arena.
type LightSource struct {
	Active bool
	X, Y   float64
}

// FitnessRecord stores per-generation fitness data for graphing.
type FitnessRecord struct {
	Best float64
	Avg  float64
}

// SwarmMessage is a broadcast int message with position.
type SwarmMessage struct {
	Value int
	X, Y  float64
}

// MsgWave is a visual expanding ring from a broadcast message.
type MsgWave struct {
	X, Y   float64
	Radius float64 // current radius (expands each tick)
	Timer  int     // counts down from 30
	Value  int     // message value (for color)
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
	SpawnTick  int     // tick when spawned (for expiry in dynamic mode)
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

	// Presets holds all built-in SwarmScript programs as name+source pairs.
	// Using a struct slice prevents name/program arrays from diverging.
	Presets []SwarmPreset

	// Deprecated: use Presets[i].Name / Presets[i].Source instead.
	// Kept temporarily for backward compatibility during migration.
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
	DynamicEnv  bool // dynamic environment: moving obstacles + expiring packages
	MazeOn      bool
	ArenaEditMode bool // arena editor: click to place/remove obstacles & stations
	ArenaEditTool int  // 0=obstacle, 1=station, 2=delete
	ArenaDragIdx  int  // index of obstacle being dragged (-1=none)
	ArenaDragType int  // 0=obstacle, 1=station
	WrapMode    bool // false=BOUNCE, true=WRAP
	ShowTrails  bool

	// Color filter (W key): 0=off, 1=red, 2=green, 3=blue, 4=carrying, 5=idle
	ColorFilter int

	// Formation analysis overlay (F6)
	ShowFormation bool

	// Parameter preset system (F8 save, F9 load)
	PresetIdx int // current preset index for cycling

	// Live chart overlay (. key)
	ShowLiveChart bool

	// Auto-Optimizer
	AutoOptimizer *AutoOptimizerState

	// Scenario chain
	ScenarioChain *ScenarioChainState

	// Heatmap overlay (Y key)
	ShowHeatmap bool
	HeatmapGrid []float64 // flat grid of visit counts
	HeatmapCols int
	HeatmapRows int

	// Selected bot for info overlay
	SelectedBot int // -1 = none
	CompareBot  int // -1 = none, Shift+click to set second bot for comparison
	HoveredBot  int // -1 = none, set by mouse hover detection

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

	// Evolution system
	EvolutionOn      bool
	Generation       int
	EvolutionTimer   int     // ticks since last evolution
	BestFitness      float64
	AvgFitness       float64
	UsedParams       [26]bool // which $A-$Z are used in current program
	ShowGenomeViz    bool     // V key toggle: show genome visualization overlay
	FitnessHistory   []FitnessRecord // per-generation fitness history for graph

	// Fitness comparison baseline (B key saves current curve)
	BaselineFitness []FitnessRecord // saved fitness curve for overlay comparison
	BaselineLabel   string          // label for the baseline (e.g. "GP Gen 25")

	// Genetic Programming (each bot evolves its own program)
	GPEnabled    bool // GP toggle
	GPGeneration int  // current GP generation
	GPTimer      int  // ticks since last GP evolution

	// Neuroevolution (each bot has its own neural network)
	NeuroEnabled    bool // NEURO toggle
	NeuroGeneration int  // current neuro generation
	NeuroTimer      int  // ticks since last neuro evolution

	// LSTM neuroevolution (alternative brain with temporal memory)
	LSTMEnabled    bool // LSTM toggle (mutually exclusive with NeuroEnabled)
	LSTMGeneration int
	LSTMTimer      int

	// Sound events (consumed by renderer each frame)
	EvolutionSoundPending bool // set true when a generation completes
	BroadcastCount        int  // messages sent this tick (for sound throttling)

	// Multiplayer teams
	TeamsEnabled     bool
	TeamAScore       int
	TeamBScore       int
	TeamAProgram     *swarmscript.SwarmProgram
	TeamBProgram     *swarmscript.SwarmProgram
	TeamAPresetIdx   int // -1 = custom
	TeamBPresetIdx   int // -1 = custom
	ChallengeActive  bool
	ChallengeTicks   int
	ChallengeResult  string

	// Message wave visualization
	ShowMsgWaves bool
	MsgWaves     []MsgWave // active expanding wave rings

	// Ramp semaphore (truck mode)
	RampBotCount int // non-carrying bots currently on ramp (rebuilt per tick)
	RampMaxBots  int // max concurrent bots on ramp (default 3)

	// Pheromone system (carrying bots leave trails)
	PherGrid *SwarmPheromoneGrid

	ShowCommGraph bool // K key toggle: show communication lines between bots
	EnergyEnabled bool // energy system toggle
	MemoryEnabled bool // bot spatial memory system toggle

	// Pareto multi-objective evolution
	ParetoEnabled bool         // use Pareto ranking instead of scalar fitness
	ParetoFront   *ParetoFront // latest Pareto front (computed each generation)

	// Sensor noise & failures
	SensorNoiseOn      bool                // toggle sensor noise system
	SensorNoiseCfg     SensorNoiseConfig   // noise parameters
	SensorFailures     []SensorFailureState // per-bot failure state
	NoisePatternLearn  *NoisePatternLearner // adapts to noise over time

	// Leaderboard / Highscore
	Leaderboard     *LeaderboardState // loaded on startup
	ShowLeaderboard bool              // toggle overlay

	CollisionCount  int // obstacle collisions this tick (reset per tick)
	ResetFlashTimer int // counts down from 30 for "RESET" flash

	// Statistics dashboard
	StatsTracker *StatsTracker
	DashboardOn  bool // D key toggle

	// Replay system
	ReplayBuf    *ReplayBuffer // ring buffer of state snapshots
	ReplayPlayer *ReplayPlayer // playback controller (seek/rewind/step)
	ReplayMode   bool          // true = replay active, simulation paused

	// Population diversity (updated each generation)
	Diversity *DiversityMetrics

	// Genealogy tracking (lineage across generations)
	Genealogy *GenealogyTracker

	// Novelty Search (behavioral diversity fitness)
	NoveltyArchive *NoveltyArchive
	NoveltyEnabled bool

	// Speciation (NEAT-style species formation)
	Speciation     *SpeciationState
	SpeciationOn   bool // Shift+E toggle
	ShowSpeciation bool // species visualization overlay

	// Emergent pattern detection (Shift+F)
	ShowPatterns  bool
	PatternResult *PatternResult

	// Achievement system (Shift+B)
	AchievementState  *AchievementState
	ShowAchievements  bool

	// Tournament mode
	TournamentOn      bool
	TournamentEntries []TournamentEntry
	TournamentRound   int // current round being played
	TournamentPhase   int // 0=idle, 1=running, 2=results
	TournamentTimer   int // ticks remaining in current round
	TournamentResults []TournamentResult
	TournamentScroll  int

	// Genom-Browser overlay (G key)
	GenomeBrowserOn     bool
	GenomeBrowserSort   int // 0=fitness, 1=age, 2=deliveries
	GenomeBrowserScroll int // scroll offset in list

	// Clipboard flash (visual feedback on copy)
	ClipboardFlash int // >0 = flash timer (counts down)

	// Aurora background effect (Shift+A)
	AuroraOn bool

	// Prediction arrows (Shift+T in swarm mode, separate from classic trails)
	ShowPrediction bool

	// Day/Night cycle (Shift+I)
	DayNightOn    bool
	DayNightPhase float64 // 0.0-1.0 (0=noon, 0.5=midnight)
	DayNightSpeed float64 // phase advance per tick

	// Swarm center of mass overlay (Shift+C in swarm mode)
	ShowSwarmCenter bool
	SwarmCenterX    float64
	SwarmCenterY    float64
	SwarmSpread     float64

	// Congestion zone overlay (Shift+Y)
	ShowZones     bool
	CongestionGrid []float64
	CongestionCols int
	CongestionRows int

	// Cooperative learning (knowledge transfer between bots)
	CoopState *CooperativeState
	CoopOn    bool

	// Terrain system (heightmap + biomes)
	Terrain   *TerrainGrid
	TerrainOn bool

	// Weather system
	Weather   *WeatherState
	WeatherOn bool

	// Multi-swarm arena
	MultiSwarm *MultiSwarmState

	// Reinforcement learning
	RLState     *RLState
	RLEnabled   bool
	RLBotStates []RLBotState

	// Morphological evolution
	MorphEnabled bool
	MorphConfig  *MorphologyConfig

	// Curriculum learning (auto-difficulty)
	Curriculum   *CurriculumState
	CurriculumOn bool

	// Predator-prey co-evolution
	PredatorPrey   *PredatorPreyState
	PredatorPreyOn bool

	// Stigmergy / collective building
	Stigmergy   *StigmergyState
	StigmergyOn bool

	// Classic swarm algorithms (Boids, PSO, ACO, Firefly)
	SwarmAlgo   *SwarmAlgorithmState
	SwarmAlgoOn bool

	// Transfer learning between scenarios
	Transfer *TransferState

	// Interactive evolution (user-guided selection)
	InteractiveEvo   *InteractiveEvoState
	InteractiveEvoOn bool

	// Emergent language evolution
	LanguageEvo   *LanguageEvo
	LanguageEvoOn bool

	// Shape formation system
	ShapeFormation   *ShapeFormationState
	ShapeFormationOn bool

	// Quorum sensing
	Quorum   *QuorumState
	QuorumOn bool

	// Reaction-diffusion (Turing patterns)
	ReactionDiffusion   *ReactionDiffusionState
	ReactionDiffusionOn bool

	// Energy economy / trading
	EnergyEconomy   *EnergyEconomyState
	EnergyEconomyOn bool

	// Hierarchical swarm (multi-scale)
	Hierarchy   *HierarchyState
	HierarchyOn bool

	// Episodic memory
	EpisodicMemory   *EpisodicMemoryState
	EpisodicMemoryOn bool

	// Benchmarks
	Benchmark   *BenchmarkState
	BenchmarkOn bool

	// Genetic Regulatory Networks
	GRN   *GRNState
	GRNOn bool

	// Hebbian plasticity (lifetime learning)
	Hebbian   *HebbianState
	HebbianOn bool

	// Swarm immune system
	Immune   *ImmuneState
	ImmuneOn bool

	// Meta-evolution
	MetaEvo   *MetaEvoState
	MetaEvoOn bool

	// Co-evolutionary ecosystem
	Ecosystem   *EcosystemState
	EcosystemOn bool

	// Diploid genetics
	Diploid   *DiploidState
	DiploidOn bool

	// Emergent Specialization
	Specialization   *SpecializationState
	SpecializationOn bool

	// NAS Evolution
	NASEvo   *NASEvoState
	NASEvoOn bool

	// Schwarm-Traeume (Offline Replay)
	Dreams   *DreamState
	DreamsOn bool

	// Emergent Language
	Language   *LanguageState
	LanguageOn bool

	// Morphogenesis (Turing patterns)
	Morphogenesis   *MorphogenesisState
	MorphogenesisOn bool

	// Stigmergy 2.0 (compound pheromones)
	Stigmergy2   *Stigmergy2State
	Stigmergy2On bool

	// Spatial Memory (shared knowledge map)
	SpatialMemory   *SpatialMemoryState
	SpatialMemoryOn bool

	// Body Evolution (morphological traits)
	BodyEvo   *BodyEvoState
	BodyEvoOn bool

	// Kuramoto Oscillators
	Oscillator   *OscillatorState
	OscillatorOn bool

	// Learning Classifier System
	Classifier   *ClassifierState
	ClassifierOn bool

	// Homeostasis (internal drives)
	Homeostasis   *HomeostasisState
	HomeostasisOn bool

	// Collective Dream (offline strategy replay)
	CollectiveDream   *CollectiveDreamState
	CollectiveDreamOn bool

	// Gene Cascade (regulatory cascades)
	GeneCascade   *GeneCascadeState
	GeneCascadeOn bool

	// Democracy (ranked choice voting)
	Democracy   *DemocracyState
	DemocracyOn bool

	// Temporal Memory (pattern recognition)
	TemporalMemory   *TemporalMemoryState
	TemporalMemoryOn bool

	// Adaptive Immune System
	AdaptiveImmune   *AdaptiveImmuneState
	AdaptiveImmuneOn bool

	// Neural Pruning (synaptic Darwinism)
	NeuralPruning   *NeuralPruningState
	NeuralPruningOn bool

	// Stock Market (strategy trading)
	StockMarket   *StockMarketState
	StockMarketOn bool

	// Epigenetics (heritable marks)
	Epigenetics   *EpigeneticsState
	EpigeneticsOn bool

	// A* Pathfinding
	AStar       *AStarState
	AStarOn     bool
	ShowPaths   bool
	ShowNavGrid bool // Shift+G: debug overlay showing blocked/free cells
	ShowFlock   bool // Shift+F: flocking velocity lines overlay
	ShowRoles   bool // Shift+R: role color overlay
	ShowFirefly bool // 9 key: firefly flash overlay
	ShowVortex  bool // 0 key: vortex rotation overlay

	// Lévy-Flight Foraging
	Levy   *LevyState
	LevyOn bool

	// Firefly Synchronization
	Firefly   *FireflyState
	FireflyOn bool

	// Collective Transport
	Transport   *TransportState
	TransportOn bool

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

// TournamentEntry is a saved program for tournament mode.
type TournamentEntry struct {
	Name    string
	Source  string // SwarmScript source code
	Program *swarmscript.SwarmProgram
}

// TournamentResult tracks a program's tournament performance.
type TournamentResult struct {
	Name       string
	Deliveries int
	Correct    int
	Wrong      int
	Score      int
}

// AllObstacles returns combined obstacles and maze walls.
func (ss *SwarmState) AllObstacles() []*physics.Obstacle {
	result := make([]*physics.Obstacle, 0, len(ss.Obstacles)+len(ss.MazeWalls))
	result = append(result, ss.Obstacles...)
	result = append(result, ss.MazeWalls...)
	return result
}
