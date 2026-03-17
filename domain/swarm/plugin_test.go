package swarm

import (
	"testing"
)

func TestNewPluginRegistry(t *testing.T) {
	reg := NewPluginRegistry(10)
	if reg.MaxPlugins != 10 {
		t.Errorf("expected 10, got %d", reg.MaxPlugins)
	}
}

func TestNewPluginRegistryMin(t *testing.T) {
	reg := NewPluginRegistry(0)
	if reg.MaxPlugins != 20 {
		t.Errorf("0 should default to 20, got %d", reg.MaxPlugins)
	}
}

func TestRegisterPlugin(t *testing.T) {
	reg := NewPluginRegistry(5)
	ok := RegisterPlugin(reg, PluginInfo{ID: "test1", Name: "Test 1", Type: PluginBehavior})
	if !ok {
		t.Error("registration should succeed")
	}
	if PluginCount(reg) != 1 {
		t.Errorf("expected 1 plugin, got %d", PluginCount(reg))
	}
}

func TestRegisterPluginDuplicate(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "test1", Name: "Test 1"})
	ok := RegisterPlugin(reg, PluginInfo{ID: "test1", Name: "Duplicate"})
	if ok {
		t.Error("duplicate ID should fail")
	}
}

func TestRegisterPluginMax(t *testing.T) {
	reg := NewPluginRegistry(2)
	RegisterPlugin(reg, PluginInfo{ID: "a"})
	RegisterPlugin(reg, PluginInfo{ID: "b"})
	ok := RegisterPlugin(reg, PluginInfo{ID: "c"})
	if ok {
		t.Error("exceeding max should fail")
	}
}

func TestRegisterPluginNil(t *testing.T) {
	if RegisterPlugin(nil, PluginInfo{ID: "x"}) {
		t.Error("nil registry should fail")
	}
}

func TestUnregisterPlugin(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "test1"})
	RegisterHook(reg, PluginHook{PluginID: "test1", OnTick: func(ss *SwarmState) {}})

	ok := UnregisterPlugin(reg, "test1")
	if !ok {
		t.Error("unregister should succeed")
	}
	if PluginCount(reg) != 0 {
		t.Error("plugin should be removed")
	}
	if len(reg.Hooks) != 0 {
		t.Error("hooks should be removed")
	}
}

func TestUnregisterPluginNotFound(t *testing.T) {
	reg := NewPluginRegistry(5)
	if UnregisterPlugin(reg, "nonexistent") {
		t.Error("nonexistent should fail")
	}
}

func TestRegisterHook(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "test1"})
	ok := RegisterHook(reg, PluginHook{PluginID: "test1", OnTick: func(ss *SwarmState) {}})
	if !ok {
		t.Error("hook registration should succeed")
	}
	if len(reg.Hooks) != 1 {
		t.Error("should have 1 hook")
	}
}

func TestRegisterHookNoPlugin(t *testing.T) {
	reg := NewPluginRegistry(5)
	ok := RegisterHook(reg, PluginHook{PluginID: "nonexistent"})
	if ok {
		t.Error("hook for nonexistent plugin should fail")
	}
}

func TestEnableDisablePlugin(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "test1"})
	if !IsPluginEnabled(reg, "test1") {
		t.Error("plugin should be enabled by default")
	}
	DisablePlugin(reg, "test1")
	if IsPluginEnabled(reg, "test1") {
		t.Error("plugin should be disabled")
	}
	EnablePlugin(reg, "test1")
	if !IsPluginEnabled(reg, "test1") {
		t.Error("plugin should be re-enabled")
	}
}

func TestEnableDisableNil(t *testing.T) {
	if EnablePlugin(nil, "x") {
		t.Error("nil should fail")
	}
	if DisablePlugin(nil, "x") {
		t.Error("nil should fail")
	}
}

func TestRunTickHooks(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "test1"})
	called := false
	RegisterHook(reg, PluginHook{
		PluginID: "test1",
		OnTick:   func(ss *SwarmState) { called = true },
	})
	RunTickHooks(reg, &SwarmState{})
	if !called {
		t.Error("tick hook should be called")
	}
}

func TestRunTickHooksDisabled(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "test1"})
	called := false
	RegisterHook(reg, PluginHook{
		PluginID: "test1",
		OnTick:   func(ss *SwarmState) { called = true },
	})
	DisablePlugin(reg, "test1")
	RunTickHooks(reg, &SwarmState{})
	if called {
		t.Error("disabled plugin hooks should not be called")
	}
}

func TestRunTickHooksNil(t *testing.T) {
	RunTickHooks(nil, &SwarmState{}) // should not panic
}

func TestRunScoreHooks(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "bonus"})
	RegisterHook(reg, PluginHook{
		PluginID: "bonus",
		OnScore:  func(bot *SwarmBot, ss *SwarmState) float64 { return 42 },
	})
	score := RunScoreHooks(reg, &SwarmBot{}, &SwarmState{})
	if score != 42 {
		t.Errorf("expected score 42, got %f", score)
	}
}

func TestRunScoreHooksMultiple(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "a"})
	RegisterPlugin(reg, PluginInfo{ID: "b"})
	RegisterHook(reg, PluginHook{
		PluginID: "a",
		OnScore:  func(bot *SwarmBot, ss *SwarmState) float64 { return 10 },
	})
	RegisterHook(reg, PluginHook{
		PluginID: "b",
		OnScore:  func(bot *SwarmBot, ss *SwarmState) float64 { return 20 },
	})
	score := RunScoreHooks(reg, &SwarmBot{}, &SwarmState{})
	if score != 30 {
		t.Errorf("expected score 30, got %f", score)
	}
}

func TestPluginCount(t *testing.T) {
	if PluginCount(nil) != 0 {
		t.Error("nil should return 0")
	}
	reg := NewPluginRegistry(5)
	if PluginCount(reg) != 0 {
		t.Error("empty should return 0")
	}
}

func TestEnabledPluginCount(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "a"})
	RegisterPlugin(reg, PluginInfo{ID: "b"})
	DisablePlugin(reg, "b")
	if EnabledPluginCount(reg) != 1 {
		t.Errorf("expected 1 enabled, got %d", EnabledPluginCount(reg))
	}
}

func TestPluginTypeName(t *testing.T) {
	if PluginTypeName(PluginBehavior) != "Verhalten" {
		t.Error("wrong name for PluginBehavior")
	}
	if PluginTypeName(PluginFitness) != "Fitness" {
		t.Error("wrong name for PluginFitness")
	}
	if PluginTypeName(PluginType(99)) != "Unbekannt" {
		t.Error("unknown should be Unbekannt")
	}
}

func TestRunBotHooks(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "test1"})
	called := false
	RegisterHook(reg, PluginHook{
		PluginID: "test1",
		OnBotAct: func(bot *SwarmBot, ss *SwarmState) { called = true },
	})
	RunBotHooks(reg, &SwarmBot{}, &SwarmState{})
	if !called {
		t.Error("bot hook should be called")
	}
}

func TestRunEvolveHooks(t *testing.T) {
	reg := NewPluginRegistry(5)
	RegisterPlugin(reg, PluginInfo{ID: "test1"})
	capturedGen := -1
	RegisterHook(reg, PluginHook{
		PluginID: "test1",
		OnEvolve: func(ss *SwarmState, gen int) { capturedGen = gen },
	})
	RunEvolveHooks(reg, &SwarmState{}, 5)
	if capturedGen != 5 {
		t.Errorf("expected gen 5, got %d", capturedGen)
	}
}
