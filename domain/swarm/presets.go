// Package swarm provides simulation logic for bot swarms.
// This file handles saving and loading parameter presets.
package swarm

import (
	"encoding/json"
	"errors"
	"os"
	"swarmsim/logger"
)

// presetsFile is the name of the JSON file where presets are stored.
const presetsFile = "swarmsim_presets.json"

// ParamPreset stores a named set of parameter values for bots.
// It includes the parameter values, which parameters are used,
// and optionally the program source code.
type ParamPreset struct {
	Name       string       `json:"name"`       // Name of the preset (e.g., "default", "aggressive")
	Params     [26]float64  `json:"params"`     // Array of 26 parameter values
	UsedParams [26]bool     `json:"used_params"` // Boolean flags indicating which parameters are active
	ProgramSrc string       `json:"program_src,omitempty"` // Source code of the program (optional)
}

// PresetStore holds all saved presets in the system.
type PresetStore struct {
	Presets []ParamPreset `json:"presets"` // List of all saved presets
}

// SavePreset saves the current swarm parameters as a named preset.
func SavePreset(ss *SwarmState, name string) {
	store := loadPresetStore()

	preset := ParamPreset{
		Name:       name,
		UsedParams: ss.UsedParams,
	}

	// Average params across bots (only used parameters)
	preset.Params = AverageParamsAcrossBots(ss.Bots, &ss.UsedParams)

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
		if !errors.Is(err, os.ErrNotExist) {
			logger.Warn("PRESET", "Failed to read %s: %v", presetsFile, err)
		}
		return PresetStore{}
	}
	var store PresetStore
	if err := json.Unmarshal(data, &store); err != nil {
		logger.Warn("PRESET", "Parse error in %s: %v (resetting)", presetsFile, err)
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
