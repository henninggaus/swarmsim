package bot

import (
	"swarmsim/domain/comm"
	"swarmsim/domain/genetics"
	"swarmsim/domain/resource"
)

// PheromoneType identifies a pheromone channel (mirrors engine/pheromone constants).
type PheromoneType int

const (
	PherSearch        PheromoneType = 0 // blue — scouts exploring
	PherFoundResource PheromoneType = 1 // green — path to resource
	PherDanger        PheromoneType = 2 // red — low-health warning
)

// PheromoneGrid abstracts pheromone operations so domain doesn't import engine/.
type PheromoneGrid interface {
	Deposit(x, y float64, pType PheromoneType, amount float64)
	Gradient(x, y float64, pType PheromoneType) (float64, float64)
}

// Genome is an alias for genetics.Genome so consumers can use bot.Genome.
type Genome = genetics.Genome

// GenomeLabels re-exports genetics.GenomeLabels for convenience.
var GenomeLabels = genetics.GenomeLabels

// BotType identifies the kind of bot.
type BotType int

const (
	TypeScout BotType = iota
	TypeWorker
	TypeLeader
	TypeTank
	TypeHealer
)

func (t BotType) String() string {
	switch t {
	case TypeScout:
		return "Scout"
	case TypeWorker:
		return "Worker"
	case TypeLeader:
		return "Leader"
	case TypeTank:
		return "Tank"
	case TypeHealer:
		return "Healer"
	}
	return "Unknown"
}

// BotState represents the current behavioral state.
type BotState int

const (
	StateIdle BotState = iota
	StateFlocking
	StateForaging
	StateReturning
	StateFormation
	StateRepairing
	StatePushing
	StateScouting
	StateNoEnergy
	StateCooperating
	StateLifting      // lifting a package (truck mode)
	StateCarryingPkg  // carrying a package to zone (truck mode)
	StateWaitingHelp  // waiting for cooperative lift (truck mode)
	StateCoordinating // leader directing traffic (truck mode)
)

func (s BotState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateFlocking:
		return "Flocking"
	case StateForaging:
		return "Foraging"
	case StateReturning:
		return "Returning"
	case StateFormation:
		return "Formation"
	case StateRepairing:
		return "Repairing"
	case StatePushing:
		return "Pushing"
	case StateScouting:
		return "Scouting"
	case StateNoEnergy:
		return "No Energy"
	case StateCooperating:
		return "Cooperating"
	case StateLifting:
		return "Lifting"
	case StateCarryingPkg:
		return "Carrying Pkg"
	case StateWaitingHelp:
		return "Waiting Help"
	case StateCoordinating:
		return "Coordinating"
	}
	return "Unknown"
}

// Vec2 is a 2D vector.
type Vec2 struct{ X, Y float64 }

// UpdateContext holds environmental data passed to bots each tick.
type UpdateContext struct {
	Nearby     []Bot
	Resources  []*resource.Resource
	Inbox      []comm.Message
	HomeX      float64
	HomeY      float64
	Pheromones PheromoneGrid
	ECfg       EnergyCfg
	Tick       int
}

// EnergyCfg holds energy cost/gain parameters.
type EnergyCfg struct {
	MoveCost  float64
	MsgCost   float64
	CarryCost float64
	PherCost  float64
	TankPush  float64
	DecayMult float64
}

const TrailLen = 5

// BaseBot contains shared fields for all bot types.
type BaseBot struct {
	BotID       int
	BotType     BotType
	Pos         Vec2
	Vel         Vec2
	MaxSpeed    float64
	BaseSpeed   float64 // original max speed before genome modifier
	SensorRange float64
	CommRange   float64
	Capacity    int
	Hp          float64
	MaxHp       float64
	Alive       bool
	Radius      float64
	State       BotState

	Energy    float64
	MaxEnergy float64

	CarryCap float64 // carrying capacity (kg) for truck mode

	Genome genetics.Genome

	// Fitness tracking
	FitResourcesCollected int
	FitResourcesDelivered int
	FitMessagesRelayed    int
	FitBotsHealed         int
	FitDistanceExplored   float64
	FitZeroEnergyTicks    int
	LastPos               Vec2

	Inventory []*resource.Resource
	Trail     [TrailLen]Vec2
	TrailIdx  int

	TargetPos   *Vec2
	TargetValid bool

	FormationID   int
	FormationSlot int
}

// Bot is the interface all bot types implement.
type Bot interface {
	ID() int
	Type() BotType
	Position() Vec2
	Velocity() Vec2
	Health() float64
	MaxHealth() float64
	IsAlive() bool
	GetRadius() float64
	GetSensorRange() float64
	GetCommRange() float64
	GetState() BotState
	GetInventory() []*resource.Resource
	GetTrail() [TrailLen]Vec2
	GetBase() *BaseBot
	GetEnergy() float64
	GetGenome() *genetics.Genome

	Update(ctx *UpdateContext) []comm.Message
}
