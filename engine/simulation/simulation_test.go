package simulation

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ArenaWidth <= 0 {
		t.Error("ArenaWidth must be positive")
	}
	if cfg.ArenaHeight <= 0 {
		t.Error("ArenaHeight must be positive")
	}
	if cfg.TickRate <= 0 {
		t.Error("TickRate must be positive")
	}
	if cfg.HomeBaseR <= 0 {
		t.Error("HomeBaseR must be positive")
	}
	if cfg.SpatialCellSize <= 0 {
		t.Error("SpatialCellSize must be positive")
	}
	if cfg.PherCellSize <= 0 {
		t.Error("PherCellSize must be positive")
	}
	if cfg.PherDecay <= 0 || cfg.PherDecay > 1 {
		t.Errorf("PherDecay should be in (0,1], got %f", cfg.PherDecay)
	}
	if cfg.GenerationLength <= 0 {
		t.Error("GenerationLength must be positive")
	}
	if cfg.MutationRate < 0 || cfg.MutationRate > 1 {
		t.Errorf("MutationRate should be in [0,1], got %f", cfg.MutationRate)
	}
	if cfg.EliteRatio < 0 || cfg.EliteRatio > 1 {
		t.Errorf("EliteRatio should be in [0,1], got %f", cfg.EliteRatio)
	}
}

func TestDefaultConfig_BotCounts(t *testing.T) {
	cfg := DefaultConfig()

	total := cfg.InitScouts + cfg.InitWorkers + cfg.InitLeaders + cfg.InitTanks + cfg.InitHealers
	if total <= 0 {
		t.Error("default config should spawn at least one bot")
	}
}

func TestNewSimulation(t *testing.T) {
	cfg := DefaultConfig()
	s := NewSimulation(cfg)

	if s == nil {
		t.Fatal("NewSimulation returned nil")
	}
	if s.Arena == nil {
		t.Error("Arena should not be nil")
	}
	if s.Pheromones == nil {
		t.Error("Pheromones should not be nil")
	}
	if s.Channel == nil {
		t.Error("Channel should not be nil")
	}
	if s.Hash == nil {
		t.Error("Hash should not be nil")
	}
	if s.Rng == nil {
		t.Error("Rng should not be nil")
	}
}

func TestNewSimulation_SpawnsBots(t *testing.T) {
	cfg := DefaultConfig()
	s := NewSimulation(cfg)

	expectedBots := cfg.InitScouts + cfg.InitWorkers + cfg.InitLeaders + cfg.InitTanks + cfg.InitHealers
	if len(s.Bots) != expectedBots {
		t.Errorf("expected %d bots, got %d", expectedBots, len(s.Bots))
	}
}

func TestNewSimulation_SpawnsResources(t *testing.T) {
	cfg := DefaultConfig()
	s := NewSimulation(cfg)

	if len(s.Resources) != cfg.InitResources {
		t.Errorf("expected %d resources, got %d", cfg.InitResources, len(s.Resources))
	}
}

func TestNewSimulation_InitialState(t *testing.T) {
	cfg := DefaultConfig()
	s := NewSimulation(cfg)

	if s.Tick != 0 {
		t.Errorf("expected Tick 0, got %d", s.Tick)
	}
	if s.Paused {
		t.Error("simulation should not start paused")
	}
	if s.Speed != 1.0 {
		t.Errorf("expected Speed 1.0, got %f", s.Speed)
	}
	if s.Delivered != 0 {
		t.Errorf("expected 0 delivered, got %d", s.Delivered)
	}
	if s.SelectedBotID != -1 {
		t.Errorf("expected SelectedBotID -1, got %d", s.SelectedBotID)
	}
}

func TestNewSimulation_ConfigPreserved(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ArenaWidth = 2000
	cfg.ArenaHeight = 1500
	s := NewSimulation(cfg)

	if s.Cfg.ArenaWidth != 2000 {
		t.Errorf("expected ArenaWidth 2000, got %f", s.Cfg.ArenaWidth)
	}
	if s.Cfg.ArenaHeight != 1500 {
		t.Errorf("expected ArenaHeight 1500, got %f", s.Cfg.ArenaHeight)
	}
}

func TestNewSimulation_WaveSystem(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WaveEnabled = true
	cfg.WaveInterval = 300
	s := NewSimulation(cfg)

	if s.WaveTicksLeft != 300 {
		t.Errorf("expected WaveTicksLeft 300, got %d", s.WaveTicksLeft)
	}
}

func TestNewSimulation_WaveDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WaveEnabled = false
	s := NewSimulation(cfg)

	if s.WaveTicksLeft != 0 {
		t.Errorf("expected WaveTicksLeft 0 when waves disabled, got %d", s.WaveTicksLeft)
	}
}

func TestLoadFactoryScenario(t *testing.T) {
	cfg := DefaultConfig()
	s := NewSimulation(cfg)
	s.LoadFactoryScenario()
	if !s.FactoryMode {
		t.Error("should be in factory mode")
	}
	if s.FactoryState == nil {
		t.Error("factory state should not be nil")
	}
	if len(s.FactoryState.Bots) == 0 {
		t.Error("should have bots")
	}
}
