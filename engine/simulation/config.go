package simulation

// Config holds all simulation parameters.
type Config struct {
	ArenaWidth  float64
	ArenaHeight float64
	TickRate    int // ticks per second

	InitScouts  int
	InitWorkers int
	InitLeaders int
	InitTanks   int
	InitHealers int

	InitResources   int
	InitObstacles   int
	ResourceValue   float64
	ResourceRespawn bool
	RespawnInterval int // ticks between respawn checks

	HomeBaseX float64
	HomeBaseY float64
	HomeBaseR float64

	SpatialCellSize float64

	// Pheromone config
	PherCellSize  float64
	PherDecay     float64
	PherDiffusion float64

	// Energy config
	EnergyMoveCost     float64
	EnergyMsgCost      float64
	EnergyCarryCost    float64
	EnergyPherCost     float64
	EnergyTankPush     float64
	EnergyBaseRecharge float64
	EnergyBaseRange    float64
	EnergyHealGive     float64
	EnergyHealCost     float64
	EnergyDecayMult    float64 // multiplier on all costs (for energy crisis)

	// Wave system
	WaveEnabled      bool
	WaveInterval     int // ticks between waves
	WaveMinResources int // min normal+heavy per wave
	WaveMaxResources int // max normal+heavy per wave
	WaveMinHeavy     int // min heavy per wave
	WaveMaxHeavy     int // max heavy per wave

	// Evolution config
	GenerationLength int
	MutationRate     float64
	MutationSigma    float64
	EliteRatio       float64 // top percentage that survives
	AutoEvolve       bool    // auto-end generations

	// Truck mode config
	TruckLiftTicks      int     // ticks for lift animation
	TruckLiftEnergy     float64 // energy cost per lift
	TruckCarryPerWeight float64 // energy cost per weight per tick
	TruckRampExtraCost  float64 // additional energy per tick on ramp
	TruckChargeRate     float64 // energy recharge rate at charging stations
	TruckChargeRange    float64 // range of charging stations
}

// DefaultConfig returns the default simulation configuration.
func DefaultConfig() Config {
	return Config{
		ArenaWidth:      1600,
		ArenaHeight:     1200,
		TickRate:        30,
		InitScouts:      10,
		InitWorkers:     20,
		InitLeaders:     3,
		InitTanks:       5,
		InitHealers:     5,
		InitResources:   30,
		InitObstacles:   8,
		ResourceValue:   1.0,
		ResourceRespawn: false,
		RespawnInterval: 120,
		HomeBaseX:       800,
		HomeBaseY:       600,
		HomeBaseR:       60,
		SpatialCellSize: 100,

		PherCellSize:  10,
		PherDecay:     0.995,
		PherDiffusion: 0.01,

		WaveEnabled:      true,
		WaveInterval:     500,
		WaveMinResources: 10,
		WaveMaxResources: 15,
		WaveMinHeavy:     2,
		WaveMaxHeavy:     3,

		EnergyMoveCost:     0.02,
		EnergyMsgCost:      0.5,
		EnergyCarryCost:    0.01,
		EnergyPherCost:     0.1,
		EnergyTankPush:     0.5,
		EnergyBaseRecharge: 2.0,
		EnergyBaseRange:    100,
		EnergyHealGive:     0.5,
		EnergyHealCost:     0.3,
		EnergyDecayMult:    1.0,

		GenerationLength: 1500,
		MutationRate:     0.10,
		MutationSigma:    0.10,
		EliteRatio:       0.30,
		AutoEvolve:       false,

		TruckLiftTicks:      15,
		TruckLiftEnergy:     2.0,
		TruckCarryPerWeight: 0.05,
		TruckRampExtraCost:  0.03,
		TruckChargeRate:     2.0,
		TruckChargeRange:    60.0,
	}
}
