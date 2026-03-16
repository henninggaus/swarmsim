package simulation

// Timing constants for the swarm simulation loop.
// Centralizing these avoids magic numbers scattered throughout swarm_ai.go.
const (
	// DropoffBeaconInterval is how often (in ticks) dropoff stations broadcast beacon messages.
	DropoffBeaconInterval = 30

	// DropoffPickupDist is the maximum distance for a bot to interact with a station.
	DropoffPickupDist = 30

	// ScatterDuration is how many ticks a bot stays in scatter mode.
	ScatterDuration = 15
	// ScatterCooldownTicks is the cooldown after scattering before a bot can scatter again.
	ScatterCooldownTicks = 30
	// IdleThreshold is how many ticks of no movement before idle exploration kicks in.
	IdleThreshold = 60
	// IdleExploreDuration is how many ticks a bot explores when idle.
	IdleExploreDuration = 30

	// EvolutionInterval is the number of ticks between parameter evolution generations.
	EvolutionInterval = 1500
	// GPEvolutionInterval is the number of ticks between GP evolution generations.
	GPEvolutionInterval = 2000
	// NeuroEvolutionInterval is the number of ticks between neuroevolution generations.
	NeuroEvolutionInterval = 2000

	// DeliveryLogInterval is how often (in ticks) delivery statistics are logged.
	DeliveryLogInterval = 600
	// TruckDebugLogInterval is how often (in ticks) truck debug info is logged.
	TruckDebugLogInterval = 200

	// HeatmapUpdateInterval is how often (in ticks) the statistics heatmap is updated.
	HeatmapUpdateInterval = 100
	// RankingsUpdateInterval is how often (in ticks) bot rankings are updated.
	RankingsUpdateInterval = 500

	// SwarmHeatmapInterval is how often (in ticks) the motion heatmap accumulates data.
	SwarmHeatmapInterval = 5
	// MemoryUpdateInterval is how often (in ticks) bot spatial memory is updated.
	MemoryUpdateInterval = 5
)
