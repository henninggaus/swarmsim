package swarm

import (
	"math"
	"math/rand"
	"swarmsim/domain/physics"
	"swarmsim/logger"
)

// CurriculumStage defines one difficulty level in the curriculum.
type CurriculumStage struct {
	Name           string
	Level          int
	ObstacleCount  int     // number of obstacles to place
	ArenaScale     float64 // 1.0 = normal, 1.5 = 50% larger
	PackageTimeout int     // ticks before packages expire (0=never)
	RespawnDelay   int     // ticks between package respawns
	SpeedModifier  float64 // global bot speed modifier (1.0=normal, 0.8=slower)
	MaxBots        int     // max bots allowed (0=unlimited)
}

// CurriculumState manages automatic difficulty progression.
type CurriculumState struct {
	Stages       []CurriculumStage
	CurrentStage int
	TicksInStage int
	MinTicks     int // minimum ticks before advancing (default 3000)

	// Performance tracking for advancement decision
	PerformanceWindow []float64 // recent fitness values (sliding window)
	WindowSize        int       // how many generations to track (default 5)
	PlateauThreshold  float64   // min improvement to not count as plateau (default 5%)
	PlateauCount      int       // consecutive generations without improvement
	PlateauLimit      int       // advance after this many plateau generations (default 3)

	// Stats
	StagesCompleted int
	TotalAdvances   int
}

// DefaultCurriculumStages returns a progression of 6 difficulty levels.
func DefaultCurriculumStages() []CurriculumStage {
	return []CurriculumStage{
		{
			Name: "Anfaenger", Level: 1,
			ObstacleCount: 0, ArenaScale: 1.0,
			PackageTimeout: 0, RespawnDelay: 50,
			SpeedModifier: 1.0,
		},
		{
			Name: "Leicht", Level: 2,
			ObstacleCount: 5, ArenaScale: 1.0,
			PackageTimeout: 0, RespawnDelay: 80,
			SpeedModifier: 1.0,
		},
		{
			Name: "Mittel", Level: 3,
			ObstacleCount: 10, ArenaScale: 1.2,
			PackageTimeout: 3000, RespawnDelay: 100,
			SpeedModifier: 1.0,
		},
		{
			Name: "Schwer", Level: 4,
			ObstacleCount: 15, ArenaScale: 1.3,
			PackageTimeout: 2000, RespawnDelay: 150,
			SpeedModifier: 0.9,
		},
		{
			Name: "Experte", Level: 5,
			ObstacleCount: 20, ArenaScale: 1.5,
			PackageTimeout: 1500, RespawnDelay: 200,
			SpeedModifier: 0.8,
		},
		{
			Name: "Meister", Level: 6,
			ObstacleCount: 25, ArenaScale: 1.7,
			PackageTimeout: 1000, RespawnDelay: 250,
			SpeedModifier: 0.7,
		},
	}
}

// InitCurriculum sets up the curriculum learning system.
func InitCurriculum(ss *SwarmState) {
	cs := &CurriculumState{
		Stages:           DefaultCurriculumStages(),
		CurrentStage:     0,
		MinTicks:         3000,
		WindowSize:       5,
		PlateauThreshold: 0.05,
		PlateauLimit:     3,
	}
	ss.Curriculum = cs
	ApplyCurriculumStage(ss)
	logger.Info("CURRICULUM", "Initialisiert: %d Stufen, Start=%s",
		len(cs.Stages), cs.Stages[0].Name)
}

// ClearCurriculum disables the curriculum system.
func ClearCurriculum(ss *SwarmState) {
	ss.Curriculum = nil
	ss.CurriculumOn = false
}

// ApplyCurriculumStage configures the simulation for the current stage.
func ApplyCurriculumStage(ss *SwarmState) {
	cs := ss.Curriculum
	if cs == nil || cs.CurrentStage >= len(cs.Stages) {
		return
	}
	stage := cs.Stages[cs.CurrentStage]

	// Generate obstacles for this stage
	if stage.ObstacleCount > 0 {
		ss.ObstaclesOn = true
		generateCurriculumObstacles(ss, stage.ObstacleCount)
	} else {
		ss.ObstaclesOn = false
		ss.Obstacles = nil
	}

	cs.TicksInStage = 0
	logger.Info("CURRICULUM", "Stage %d: %s (Obstacles=%d, Arena=%.1fx, Timeout=%d)",
		stage.Level, stage.Name, stage.ObstacleCount, stage.ArenaScale, stage.PackageTimeout)
}

// generateCurriculumObstacles places a specific number of obstacles.
func generateCurriculumObstacles(ss *SwarmState, count int) {
	margin := 40.0
	ss.Obstacles = nil
	for i := 0; i < count; i++ {
		w := 30 + ss.Rng.Float64()*50
		h := 30 + ss.Rng.Float64()*50
		x := margin + ss.Rng.Float64()*(ss.ArenaW-2*margin-w)
		y := margin + ss.Rng.Float64()*(ss.ArenaH-2*margin-h)
		ss.Obstacles = append(ss.Obstacles, &physics.Obstacle{X: x, Y: y, W: w, H: h})
	}
}

// TickCurriculum checks performance and advances stage if plateauing.
// Should be called after each evolution generation.
func TickCurriculum(ss *SwarmState, currentBestFitness float64) {
	cs := ss.Curriculum
	if cs == nil {
		return
	}

	cs.TicksInStage += ss.Tick // approximate

	// Add to performance window
	cs.PerformanceWindow = append(cs.PerformanceWindow, currentBestFitness)
	if len(cs.PerformanceWindow) > cs.WindowSize {
		cs.PerformanceWindow = cs.PerformanceWindow[1:]
	}

	// Need at least WindowSize samples
	if len(cs.PerformanceWindow) < cs.WindowSize {
		return
	}

	// Check for plateau: is recent improvement < threshold?
	improved := checkImprovement(cs)
	if !improved {
		cs.PlateauCount++
	} else {
		cs.PlateauCount = 0
	}

	// Advance if plateau detected and minimum time has passed
	if cs.PlateauCount >= cs.PlateauLimit {
		AdvanceCurriculum(ss)
	}
}

// checkImprovement returns true if fitness improved significantly.
func checkImprovement(cs *CurriculumState) bool {
	w := cs.PerformanceWindow
	if len(w) < 2 {
		return true
	}

	// Compare first half average to second half average
	mid := len(w) / 2
	firstAvg := 0.0
	for i := 0; i < mid; i++ {
		firstAvg += w[i]
	}
	firstAvg /= float64(mid)

	secondAvg := 0.0
	for i := mid; i < len(w); i++ {
		secondAvg += w[i]
	}
	secondAvg /= float64(len(w) - mid)

	if firstAvg <= 0 {
		return secondAvg > 0
	}

	improvement := (secondAvg - firstAvg) / math.Abs(firstAvg)
	return improvement > cs.PlateauThreshold
}

// AdvanceCurriculum moves to the next difficulty stage.
func AdvanceCurriculum(ss *SwarmState) {
	cs := ss.Curriculum
	if cs == nil {
		return
	}

	if cs.CurrentStage >= len(cs.Stages)-1 {
		logger.Info("CURRICULUM", "Bereits auf hoechster Stufe: %s",
			cs.Stages[cs.CurrentStage].Name)
		return
	}

	cs.CurrentStage++
	cs.PlateauCount = 0
	cs.PerformanceWindow = nil
	cs.StagesCompleted++
	cs.TotalAdvances++

	ApplyCurriculumStage(ss)
	logger.Info("CURRICULUM", "AUFSTIEG! Neue Stufe: %s (Level %d)",
		cs.Stages[cs.CurrentStage].Name, cs.Stages[cs.CurrentStage].Level)
}

// RetreatCurriculum moves back one difficulty stage (manual).
func RetreatCurriculum(ss *SwarmState) {
	cs := ss.Curriculum
	if cs == nil || cs.CurrentStage <= 0 {
		return
	}
	cs.CurrentStage--
	cs.PlateauCount = 0
	cs.PerformanceWindow = nil
	ApplyCurriculumStage(ss)
}

// CurriculumProgress returns completion as 0.0-1.0.
func CurriculumProgress(cs *CurriculumState) float64 {
	if cs == nil || len(cs.Stages) <= 1 {
		return 0
	}
	return float64(cs.CurrentStage) / float64(len(cs.Stages)-1)
}

// CurriculumStageName returns the name of the current stage.
func CurriculumStageName(cs *CurriculumState) string {
	if cs == nil || cs.CurrentStage >= len(cs.Stages) {
		return "?"
	}
	return cs.Stages[cs.CurrentStage].Name
}

// CurriculumStageLevel returns the level of the current stage.
func CurriculumStageLevel(cs *CurriculumState) int {
	if cs == nil || cs.CurrentStage >= len(cs.Stages) {
		return 0
	}
	return cs.Stages[cs.CurrentStage].Level
}

// CurriculumSpeedMod returns the speed modifier of the current stage.
func CurriculumSpeedMod(cs *CurriculumState) float64 {
	if cs == nil || cs.CurrentStage >= len(cs.Stages) {
		return 1.0
	}
	return cs.Stages[cs.CurrentStage].SpeedModifier
}

// CustomCurriculumStage creates a custom stage from parameters.
func CustomCurriculumStage(name string, level, obstacles int, arenaScale float64,
	timeout, respawn int, speedMod float64) CurriculumStage {
	return CurriculumStage{
		Name:           name,
		Level:          level,
		ObstacleCount:  obstacles,
		ArenaScale:     arenaScale,
		PackageTimeout: timeout,
		RespawnDelay:   respawn,
		SpeedModifier:  speedMod,
	}
}

// AddCurriculumStage appends a custom stage to the curriculum.
func AddCurriculumStage(cs *CurriculumState, stage CurriculumStage) {
	if cs != nil {
		cs.Stages = append(cs.Stages, stage)
	}
}

// ShuffleCurriculumOrder randomizes stage order (for experiments).
func ShuffleCurriculumOrder(cs *CurriculumState, rng *rand.Rand) {
	if cs == nil || len(cs.Stages) < 2 {
		return
	}
	rng.Shuffle(len(cs.Stages), func(i, j int) {
		cs.Stages[i], cs.Stages[j] = cs.Stages[j], cs.Stages[i]
	})
}
