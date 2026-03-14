package simulation

import (
	"math/rand"
	"swarmsim/domain/bot"
	"swarmsim/domain/comm"
	"swarmsim/domain/physics"
	"swarmsim/domain/resource"
	"swarmsim/domain/swarm"
	"swarmsim/engine/pheromone"
)

// CoopPickupEvent is emitted when workers cooperatively pick up a heavy resource.
type CoopPickupEvent struct {
	X, Y float64
	Tick int
}

// DeliveryEvent is emitted when resources are delivered to the home base.
type DeliveryEvent struct {
	Tick       int
	PointValue int
}

// ScenarioID identifies a scenario.
type ScenarioID int

const (
	ScenarioForaging ScenarioID = iota
	ScenarioLabyrinth
	ScenarioEnergy
	ScenarioSandbox
	ScenarioEvolution
	ScenarioSwarm
)

// Scenario defines a preset configuration.
type Scenario struct {
	ID   ScenarioID
	Name string
	Cfg  Config
	// CustomSetup runs after standard init to add custom obstacles etc.
	CustomSetup func(s *Simulation)
}

// Simulation holds all simulation state.
type Simulation struct {
	Cfg        Config
	Arena      *physics.Arena
	Bots       []bot.Bot
	Resources  []*resource.Resource
	Channel    *comm.Channel
	Hash       *physics.SpatialHash
	Pheromones *pheromone.PheromoneGrid

	Tick          int
	Speed         float64
	Paused        bool
	NextBotID     int
	NextResID     int
	Delivered     int
	ActiveMsgs    int
	TotalMsgsSent int
	Rng           *rand.Rand

	ShowCommRadius    bool
	ShowSensorRadius  bool
	ShowDebugComm     bool
	PheromoneVizMode  int // 0=off, 1=found_resource, 2=all
	ShowGenomeOverlay bool

	SelectedBotID int

	// Evolution
	Generation     int
	GenerationTick int
	BestFitness    float64
	AvgFitness     float64
	FitnessHistory []float64

	// Scenario
	CurrentScenario ScenarioID
	ScenarioTitle   string
	ScenarioTimer   int // frames remaining for title display

	GenomePool map[bot.BotType][]bot.Genome

	// Wave system
	WaveNumber    int
	WaveTicksLeft int
	Score         int

	// Visual events (consumed by renderer)
	CoopPickupEvents []CoopPickupEvent
	DeliveryEvents   []DeliveryEvent

	// Programmable swarm mode
	SwarmMode  bool
	SwarmState *swarm.SwarmState
}
