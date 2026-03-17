package swarm

import (
	"encoding/json"
	"math/rand"
)

// Checkpoint stores a snapshot of simulation state for save/restore.
type Checkpoint struct {
	Name       string
	Tick       int
	BotCount   int
	BotData    []CheckpointBot
	Settings   CheckpointSettings
	Score      int
	Generation int
	Seed       int64
}

// CheckpointBot stores per-bot data for a checkpoint.
type CheckpointBot struct {
	X, Y      float64
	Angle     float64
	Speed     float64
	Energy    float64
	State     int
	Counter   int
	Value1    int
	Value2    int
	Timer     int
	Team      int
	LEDColor  [3]uint8
	ParamValues [26]float64
	Fitness   float64
	CarryingPkg int
	BrainWeights []float64 // nil if no brain
	Stats     BotLifetimeStats
}

// CheckpointSettings stores simulation toggle states.
type CheckpointSettings struct {
	DeliveryOn   bool
	ObstaclesOn  bool
	MazeOn       bool
	EvolutionOn  bool
	NeuroEnabled bool
	LSTMEnabled  bool
	GPEnabled    bool
	EnergyEnabled bool
	WrapMode     bool
	SensorNoiseOn bool
	TerrainOn    bool
	WeatherOn    bool
	CoopOn       bool
	RLEnabled    bool
}

// CheckpointStore manages multiple checkpoints.
type CheckpointStore struct {
	Slots    []*Checkpoint
	MaxSlots int
}

// NewCheckpointStore creates a store with the given number of slots.
func NewCheckpointStore(maxSlots int) *CheckpointStore {
	if maxSlots < 1 {
		maxSlots = 5
	}
	return &CheckpointStore{
		Slots:    make([]*Checkpoint, maxSlots),
		MaxSlots: maxSlots,
	}
}

// SaveCheckpoint captures current simulation state into a slot.
func SaveCheckpoint(ss *SwarmState, store *CheckpointStore, slot int, name string) bool {
	if store == nil || slot < 0 || slot >= store.MaxSlots {
		return false
	}

	cp := &Checkpoint{
		Name:       name,
		Tick:       ss.Tick,
		BotCount:   len(ss.Bots),
		Generation: ss.Generation,
	}

	cp.BotData = make([]CheckpointBot, len(ss.Bots))
	for i, bot := range ss.Bots {
		cpBot := CheckpointBot{
			X: bot.X, Y: bot.Y,
			Angle: bot.Angle, Speed: bot.Speed,
			Energy: bot.Energy, State: bot.State,
			Counter: bot.Counter, Value1: bot.Value1, Value2: bot.Value2,
			Timer: bot.Timer, Team: bot.Team,
			LEDColor: bot.LEDColor, ParamValues: bot.ParamValues,
			Fitness: bot.Fitness, CarryingPkg: bot.CarryingPkg,
			Stats: bot.Stats,
		}
		if bot.Brain != nil {
			cpBot.BrainWeights = make([]float64, len(bot.Brain.Weights))
			copy(cpBot.BrainWeights, bot.Brain.Weights[:])
		}
		cp.BotData[i] = cpBot
	}

	cp.Settings = CheckpointSettings{
		DeliveryOn:    ss.DeliveryOn,
		ObstaclesOn:   ss.ObstaclesOn,
		MazeOn:        ss.MazeOn,
		EvolutionOn:   ss.EvolutionOn,
		NeuroEnabled:  ss.NeuroEnabled,
		LSTMEnabled:   ss.LSTMEnabled,
		GPEnabled:     ss.GPEnabled,
		EnergyEnabled: ss.EnergyEnabled,
		WrapMode:      ss.WrapMode,
		SensorNoiseOn: ss.SensorNoiseOn,
		TerrainOn:     ss.TerrainOn,
		WeatherOn:     ss.WeatherOn,
		CoopOn:        ss.CoopOn,
		RLEnabled:     ss.RLEnabled,
	}

	store.Slots[slot] = cp
	return true
}

// LoadCheckpoint restores simulation state from a checkpoint slot.
func LoadCheckpoint(ss *SwarmState, store *CheckpointStore, slot int) bool {
	if store == nil || slot < 0 || slot >= store.MaxSlots || store.Slots[slot] == nil {
		return false
	}

	cp := store.Slots[slot]

	// Resize bots if needed
	if len(ss.Bots) != cp.BotCount {
		ss.Bots = make([]SwarmBot, cp.BotCount)
		ss.BotCount = cp.BotCount
	}

	for i, cpBot := range cp.BotData {
		bot := &ss.Bots[i]
		bot.X = cpBot.X
		bot.Y = cpBot.Y
		bot.Angle = cpBot.Angle
		bot.Speed = cpBot.Speed
		bot.Energy = cpBot.Energy
		bot.State = cpBot.State
		bot.Counter = cpBot.Counter
		bot.Value1 = cpBot.Value1
		bot.Value2 = cpBot.Value2
		bot.Timer = cpBot.Timer
		bot.Team = cpBot.Team
		bot.LEDColor = cpBot.LEDColor
		bot.ParamValues = cpBot.ParamValues
		bot.Fitness = cpBot.Fitness
		bot.CarryingPkg = cpBot.CarryingPkg
		bot.Stats = cpBot.Stats
		if cpBot.BrainWeights != nil && bot.Brain != nil {
			copy(bot.Brain.Weights[:], cpBot.BrainWeights)
		}
	}

	ss.Tick = cp.Tick
	ss.Generation = cp.Generation

	// Restore settings
	s := cp.Settings
	ss.DeliveryOn = s.DeliveryOn
	ss.ObstaclesOn = s.ObstaclesOn
	ss.MazeOn = s.MazeOn
	ss.EvolutionOn = s.EvolutionOn
	ss.NeuroEnabled = s.NeuroEnabled
	ss.LSTMEnabled = s.LSTMEnabled
	ss.GPEnabled = s.GPEnabled
	ss.EnergyEnabled = s.EnergyEnabled
	ss.WrapMode = s.WrapMode
	ss.SensorNoiseOn = s.SensorNoiseOn
	ss.TerrainOn = s.TerrainOn
	ss.WeatherOn = s.WeatherOn
	ss.CoopOn = s.CoopOn
	ss.RLEnabled = s.RLEnabled

	return true
}

// CheckpointExists returns true if a checkpoint exists in the given slot.
func CheckpointExists(store *CheckpointStore, slot int) bool {
	if store == nil || slot < 0 || slot >= store.MaxSlots {
		return false
	}
	return store.Slots[slot] != nil
}

// CheckpointName returns the name of a checkpoint, or empty string.
func CheckpointName(store *CheckpointStore, slot int) string {
	if !CheckpointExists(store, slot) {
		return ""
	}
	return store.Slots[slot].Name
}

// DeleteCheckpoint removes a checkpoint from a slot.
func DeleteCheckpoint(store *CheckpointStore, slot int) bool {
	if store == nil || slot < 0 || slot >= store.MaxSlots {
		return false
	}
	store.Slots[slot] = nil
	return true
}

// UsedSlots returns the count of occupied checkpoint slots.
func UsedSlots(store *CheckpointStore) int {
	if store == nil {
		return 0
	}
	count := 0
	for _, s := range store.Slots {
		if s != nil {
			count++
		}
	}
	return count
}

// SerializeCheckpoint converts a checkpoint to JSON bytes.
func SerializeCheckpoint(cp *Checkpoint) ([]byte, error) {
	if cp == nil {
		return nil, nil
	}
	return json.Marshal(cp)
}

// DeserializeCheckpoint parses JSON bytes into a checkpoint.
func DeserializeCheckpoint(data []byte) (*Checkpoint, error) {
	if len(data) == 0 {
		return nil, nil
	}
	cp := &Checkpoint{}
	err := json.Unmarshal(data, cp)
	return cp, err
}

// QuickSave saves to slot 0 with auto-generated name.
func QuickSave(ss *SwarmState, store *CheckpointStore) bool {
	name := "Quick T" + itoa(ss.Tick)
	return SaveCheckpoint(ss, store, 0, name)
}

// QuickLoad restores from slot 0.
func QuickLoad(ss *SwarmState, store *CheckpointStore) bool {
	return LoadCheckpoint(ss, store, 0)
}

// RewindToCheckpoint loads a checkpoint and reseeds the RNG.
func RewindToCheckpoint(ss *SwarmState, store *CheckpointStore, slot int, seed int64) bool {
	if !LoadCheckpoint(ss, store, slot) {
		return false
	}
	ss.Rng = rand.New(rand.NewSource(seed))
	return true
}
