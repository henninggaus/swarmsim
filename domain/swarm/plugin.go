package swarm

// PluginType identifies the kind of plugin.
type PluginType int

const (
	PluginBehavior    PluginType = iota // custom bot behavior
	PluginFitness                       // custom fitness function
	PluginEnvironment                   // custom environment modifier
	PluginVisual                        // custom visual overlay
	PluginTypeCount
)

// PluginInfo describes a registered plugin.
type PluginInfo struct {
	ID          string
	Name        string
	Description string
	Type        PluginType
	Version     string
	Enabled     bool
	Priority    int // execution order (lower = earlier)
}

// PluginHook is a callback that plugins can register for various events.
type PluginHook struct {
	PluginID string
	OnTick   func(ss *SwarmState)                      // called every tick
	OnBotAct func(bot *SwarmBot, ss *SwarmState)        // called per bot per tick
	OnEvolve func(ss *SwarmState, generation int)       // called after evolution
	OnReset  func(ss *SwarmState)                       // called on simulation reset
	OnScore  func(bot *SwarmBot, ss *SwarmState) float64 // custom fitness contribution
}

// PluginRegistry manages all registered plugins and hooks.
type PluginRegistry struct {
	Plugins  []PluginInfo
	Hooks    []PluginHook
	MaxPlugins int
}

// NewPluginRegistry creates an empty plugin registry.
func NewPluginRegistry(maxPlugins int) *PluginRegistry {
	if maxPlugins < 1 {
		maxPlugins = 20
	}
	return &PluginRegistry{
		MaxPlugins: maxPlugins,
	}
}

// RegisterPlugin adds a plugin to the registry.
func RegisterPlugin(reg *PluginRegistry, info PluginInfo) bool {
	if reg == nil {
		return false
	}
	if len(reg.Plugins) >= reg.MaxPlugins {
		return false
	}
	// Check for duplicate ID
	for _, p := range reg.Plugins {
		if p.ID == info.ID {
			return false
		}
	}
	info.Enabled = true
	reg.Plugins = append(reg.Plugins, info)
	return true
}

// UnregisterPlugin removes a plugin and its hooks.
func UnregisterPlugin(reg *PluginRegistry, id string) bool {
	if reg == nil {
		return false
	}
	found := false
	newPlugins := make([]PluginInfo, 0, len(reg.Plugins))
	for _, p := range reg.Plugins {
		if p.ID == id {
			found = true
			continue
		}
		newPlugins = append(newPlugins, p)
	}
	reg.Plugins = newPlugins

	// Remove hooks
	newHooks := make([]PluginHook, 0, len(reg.Hooks))
	for _, h := range reg.Hooks {
		if h.PluginID != id {
			newHooks = append(newHooks, h)
		}
	}
	reg.Hooks = newHooks

	return found
}

// RegisterHook adds a hook for a plugin.
func RegisterHook(reg *PluginRegistry, hook PluginHook) bool {
	if reg == nil {
		return false
	}
	// Verify plugin exists
	found := false
	for _, p := range reg.Plugins {
		if p.ID == hook.PluginID {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	reg.Hooks = append(reg.Hooks, hook)
	return true
}

// EnablePlugin toggles a plugin on.
func EnablePlugin(reg *PluginRegistry, id string) bool {
	if reg == nil {
		return false
	}
	for i := range reg.Plugins {
		if reg.Plugins[i].ID == id {
			reg.Plugins[i].Enabled = true
			return true
		}
	}
	return false
}

// DisablePlugin toggles a plugin off.
func DisablePlugin(reg *PluginRegistry, id string) bool {
	if reg == nil {
		return false
	}
	for i := range reg.Plugins {
		if reg.Plugins[i].ID == id {
			reg.Plugins[i].Enabled = false
			return true
		}
	}
	return false
}

// IsPluginEnabled returns whether a specific plugin is enabled.
func IsPluginEnabled(reg *PluginRegistry, id string) bool {
	if reg == nil {
		return false
	}
	for _, p := range reg.Plugins {
		if p.ID == id {
			return p.Enabled
		}
	}
	return false
}

// RunTickHooks executes all enabled plugin tick hooks.
func RunTickHooks(reg *PluginRegistry, ss *SwarmState) {
	if reg == nil {
		return
	}
	for _, hook := range reg.Hooks {
		if hook.OnTick != nil && IsPluginEnabled(reg, hook.PluginID) {
			hook.OnTick(ss)
		}
	}
}

// RunBotHooks executes all enabled plugin per-bot hooks.
func RunBotHooks(reg *PluginRegistry, bot *SwarmBot, ss *SwarmState) {
	if reg == nil {
		return
	}
	for _, hook := range reg.Hooks {
		if hook.OnBotAct != nil && IsPluginEnabled(reg, hook.PluginID) {
			hook.OnBotAct(bot, ss)
		}
	}
}

// RunEvolveHooks executes all enabled plugin evolution hooks.
func RunEvolveHooks(reg *PluginRegistry, ss *SwarmState, gen int) {
	if reg == nil {
		return
	}
	for _, hook := range reg.Hooks {
		if hook.OnEvolve != nil && IsPluginEnabled(reg, hook.PluginID) {
			hook.OnEvolve(ss, gen)
		}
	}
}

// RunScoreHooks sums fitness contributions from all enabled score plugins.
func RunScoreHooks(reg *PluginRegistry, bot *SwarmBot, ss *SwarmState) float64 {
	if reg == nil {
		return 0
	}
	sum := 0.0
	for _, hook := range reg.Hooks {
		if hook.OnScore != nil && IsPluginEnabled(reg, hook.PluginID) {
			sum += hook.OnScore(bot, ss)
		}
	}
	return sum
}

// PluginCount returns the number of registered plugins.
func PluginCount(reg *PluginRegistry) int {
	if reg == nil {
		return 0
	}
	return len(reg.Plugins)
}

// EnabledPluginCount returns the number of enabled plugins.
func EnabledPluginCount(reg *PluginRegistry) int {
	if reg == nil {
		return 0
	}
	count := 0
	for _, p := range reg.Plugins {
		if p.Enabled {
			count++
		}
	}
	return count
}

// PluginTypeName returns the display name for a plugin type.
func PluginTypeName(pt PluginType) string {
	switch pt {
	case PluginBehavior:
		return "Verhalten"
	case PluginFitness:
		return "Fitness"
	case PluginEnvironment:
		return "Umgebung"
	case PluginVisual:
		return "Visuell"
	default:
		return "Unbekannt"
	}
}
