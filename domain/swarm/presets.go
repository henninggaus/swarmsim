package swarm

import (
	"encoding/json"
	"os"
	"swarmsim/logger"
)

const presetsFile = "swarmsim_presets.json"

// ParamPreset stores a named set of parameter values.
type ParamPreset struct {
	Name       string       `json:"name"`
	Params     [26]float64  `json:"params"`
	UsedParams [26]bool     `json:"used_params"`
	ProgramSrc string       `json:"program_src,omitempty"`
}

// PresetStore holds all saved presets.
type PresetStore struct {
	Presets []ParamPreset `json:"presets"`
}

// SavePreset saves current params as a named preset.
func SavePreset(ss *SwarmState, name string) {
	store := loadPresetStore()

	preset := ParamPreset{
		Name:       name,
		UsedParams: ss.UsedParams,
	}

	// Average params across bots
	if len(ss.Bots) > 0 {
		for p := 0; p < 26; p++ {
			if ss.UsedParams[p] {
				total := 0.0
				for i := range ss.Bots {
					total += ss.Bots[i].ParamValues[p]
				}
				preset.Params[p] = total / float64(len(ss.Bots))
			}
		}
	}

	// Save program source
	if ss.Editor != nil && len(ss.Editor.Lines) > 0 {
		src := ""
		for i, line := range ss.Editor.Lines {
			if i > 0 {
				src += "\n"
			}
			src += line
		}
		preset.ProgramSrc = src
	}

	// Replace existing preset with same name, or append
	found := false
	for i, p := range store.Presets {
		if p.Name == name {
			store.Presets[i] = preset
			found = true
			break
		}
	}
	if !found {
		store.Presets = append(store.Presets, preset)
	}

	savePresetStore(store)
	logger.Info("PRESET", "Saved: %s (%d params)", name, countUsed(ss.UsedParams))
}

// LoadPreset applies a named preset to all bots.
func LoadPreset(ss *SwarmState, name string) bool {
	store := loadPresetStore()

	for _, p := range store.Presets {
		if p.Name == name {
			// Apply params to all bots
			for i := range ss.Bots {
				for k := 0; k < 26; k++ {
					if p.UsedParams[k] {
						ss.Bots[i].ParamValues[k] = p.Params[k]
					}
				}
			}
			logger.Info("PRESET", "Loaded: %s", name)
			return true
		}
	}
	logger.Warn("PRESET", "Not found: %s", name)
	return false
}

// ListPresets returns names of all saved presets.
func ListPresets() []string {
	store := loadPresetStore()
	names := make([]string, len(store.Presets))
	for i, p := range store.Presets {
		names[i] = p.Name
	}
	return names
}

// DeletePreset removes a preset by name.
func DeletePreset(name string) {
	store := loadPresetStore()
	for i, p := range store.Presets {
		if p.Name == name {
			store.Presets = append(store.Presets[:i], store.Presets[i+1:]...)
			savePresetStore(store)
			logger.Info("PRESET", "Deleted: %s", name)
			return
		}
	}
}

func loadPresetStore() PresetStore {
	data, err := os.ReadFile(presetsFile)
	if err != nil {
		return PresetStore{}
	}
	var store PresetStore
	if err := json.Unmarshal(data, &store); err != nil {
		logger.Warn("PRESET", "Parse error: %v", err)
		return PresetStore{}
	}
	return store
}

func savePresetStore(store PresetStore) {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		logger.Error("PRESET", "Marshal error: %v", err)
		return
	}
	if err := os.WriteFile(presetsFile, data, 0644); err != nil {
		logger.Error("PRESET", "Write error: %v", err)
	}
}

func countUsed(used [26]bool) int {
	n := 0
	for _, u := range used {
		if u {
			n++
		}
	}
	return n
}
