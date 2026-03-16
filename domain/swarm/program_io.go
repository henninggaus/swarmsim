package swarm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"swarmsim/logger"
)

// SwarmFile represents a .swarm file on disk.
type SwarmFile struct {
	Name       string     `json:"name"`
	Version    int        `json:"version"`
	Source     string     `json:"source"`      // SwarmScript source code
	BotCount   int        `json:"bot_count"`
	Params     [26]float64 `json:"params"`     // $A-$Z values
	UsedParams [26]bool   `json:"used_params"`
	Settings   SwarmFileSettings `json:"settings"`
}

// SwarmFileSettings stores optional simulation settings.
type SwarmFileSettings struct {
	DeliveryOn  bool `json:"delivery_on"`
	ObstaclesOn bool `json:"obstacles_on"`
	MazeOn      bool `json:"maze_on"`
	WrapMode    bool `json:"wrap_mode"`
	EnergyOn    bool `json:"energy_on"`
}

// ExportProgram saves the current program and settings to a .swarm file.
func ExportProgram(ss *SwarmState, filename string) error {
	// Build source from editor lines
	src := ""
	if ss.Editor != nil && len(ss.Editor.Lines) > 0 {
		src = strings.Join(ss.Editor.Lines, "\n")
	}

	// Average params across bots
	var avgParams [26]float64
	if len(ss.Bots) > 0 {
		for i := range ss.Bots {
			for p := 0; p < 26; p++ {
				avgParams[p] += ss.Bots[i].ParamValues[p]
			}
		}
		for p := 0; p < 26; p++ {
			avgParams[p] /= float64(len(ss.Bots))
		}
	}

	sf := SwarmFile{
		Name:       ss.ProgramName,
		Version:    1,
		Source:     src,
		BotCount:   ss.BotCount,
		Params:     avgParams,
		UsedParams: ss.UsedParams,
		Settings: SwarmFileSettings{
			DeliveryOn:  ss.DeliveryOn,
			ObstaclesOn: ss.ObstaclesOn,
			MazeOn:      ss.MazeOn,
			WrapMode:    ss.WrapMode,
			EnergyOn:    ss.EnergyEnabled,
		},
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}

	// Ensure .swarm extension
	if !strings.HasSuffix(filename, ".swarm") {
		filename += ".swarm"
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	logger.Info("EXPORT", "Programm exportiert: %s (%d Bytes)", filename, len(data))
	return nil
}

// ImportProgram loads a .swarm file and applies it to the simulation state.
// Returns the source code for the editor.
func ImportProgram(ss *SwarmState, filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	var sf SwarmFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return "", err
	}

	// Apply settings
	ss.ProgramName = sf.Name
	if sf.BotCount > 0 {
		// Don't change bot count, just note it
		logger.Info("IMPORT", "Datei empfiehlt %d Bots (aktuell: %d)", sf.BotCount, ss.BotCount)
	}

	// Apply params to all bots
	for i := range ss.Bots {
		for p := 0; p < 26; p++ {
			if sf.UsedParams[p] {
				ss.Bots[i].ParamValues[p] = sf.Params[p]
			}
		}
	}
	ss.UsedParams = sf.UsedParams

	// Apply settings
	ss.DeliveryOn = sf.Settings.DeliveryOn
	ss.ObstaclesOn = sf.Settings.ObstaclesOn
	ss.MazeOn = sf.Settings.MazeOn
	ss.WrapMode = sf.Settings.WrapMode
	ss.EnergyEnabled = sf.Settings.EnergyOn

	if sf.Settings.DeliveryOn && len(ss.Stations) == 0 {
		GenerateDeliveryStations(ss)
	}

	logger.Info("IMPORT", "Programm importiert: %s (%d Zeilen)", sf.Name, strings.Count(sf.Source, "\n")+1)
	return sf.Source, nil
}

// ListSwarmFiles returns .swarm files in the current directory.
func ListSwarmFiles() []string {
	matches, err := filepath.Glob("*.swarm")
	if err != nil {
		return nil
	}
	return matches
}

// ExportProgramText exports just the SwarmScript source as a plain text .txt file.
func ExportProgramText(ss *SwarmState, filename string) error {
	src := ""
	if ss.Editor != nil && len(ss.Editor.Lines) > 0 {
		src = strings.Join(ss.Editor.Lines, "\n")
	}

	if !strings.HasSuffix(filename, ".txt") {
		filename += ".txt"
	}

	err := os.WriteFile(filename, []byte(src), 0644)
	if err != nil {
		return err
	}

	logger.Info("EXPORT", "SwarmScript exportiert: %s", filename)
	return nil
}
