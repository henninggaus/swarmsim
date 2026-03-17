package swarm

import (
	"swarmsim/logger"
)

// ScenarioChainStep defines one step in a scenario chain.
type ScenarioChainStep struct {
	Name       string
	Setup      func(ss *SwarmState) // configures arena for this step
	TickLimit  int
	Score      int // accumulated after step

	// Branching: optional condition to choose next step dynamically
	BranchFunc func(ss *SwarmState, score int) int // returns step index to jump to, or -1 for sequential
	MinScore   int                                  // if score < MinScore, skip to FailStep
	FailStep   int                                  // step index to jump to on failure (-1 = end chain)
}

// ScenarioChainState tracks a running scenario chain.
type ScenarioChainState struct {
	Active     bool
	StepIdx    int
	Steps      []ScenarioChainStep
	Timer      int
	TotalScore int
	StepScores []int
	Complete   bool

	// Branching info
	StepPath []int // which step indices were actually executed
}

// ScenarioTemplate is an editable chain definition (for the scenario editor).
type ScenarioTemplate struct {
	Name        string
	Description string
	Steps       []ScenarioStepDef
}

// ScenarioStepDef is a serializable step definition (no func pointers).
type ScenarioStepDef struct {
	Name         string
	TickLimit    int
	Obstacles    bool
	Maze         bool
	Delivery     bool
	MinScore     int // branch: if score < MinScore, jump to FailStep
	FailStep     int // -1 = end chain
}

// GetDefaultChainSteps returns the default scenario chain.
func GetDefaultChainSteps() []ScenarioChainStep {
	return []ScenarioChainStep{
		{
			Name: "1. Einfache Lieferung",
			Setup: func(ss *SwarmState) {
				ss.ObstaclesOn = false
				ss.Obstacles = nil
				ss.MazeOn = false
				ss.MazeWalls = nil
				ss.DeliveryOn = true
				ss.ResetDeliveryState()
				GenerateDeliveryStations(ss)
				ss.ResetBots()
			},
			TickLimit: 3000,
		},
		{
			Name: "2. Mit Hindernissen",
			Setup: func(ss *SwarmState) {
				ss.ObstaclesOn = true
				ss.MazeOn = false
				ss.MazeWalls = nil
				GenerateSwarmObstacles(ss)
				ss.DeliveryOn = true
				ss.ResetDeliveryState()
				GenerateDeliveryStations(ss)
				ss.ResetBots()
			},
			TickLimit: 3000,
		},
		{
			Name: "3. Labyrinth",
			Setup: func(ss *SwarmState) {
				ss.ObstaclesOn = false
				ss.Obstacles = nil
				ss.MazeOn = true
				GenerateSwarmMaze(ss)
				ss.DeliveryOn = true
				ss.ResetDeliveryState()
				GenerateDeliveryStations(ss)
				ss.ResetBots()
			},
			TickLimit: 4000,
		},
	}
}

// ScenarioChainStart begins the default scenario chain.
func ScenarioChainStart(ss *SwarmState) {
	steps := GetDefaultChainSteps()
	ss.ScenarioChain = &ScenarioChainState{
		Active:     true,
		StepIdx:    0,
		Steps:      steps,
		StepScores: make([]int, len(steps)),
		StepPath:   []int{0},
	}
	// Setup first step
	steps[0].Setup(ss)
	ss.ScenarioChain.Timer = steps[0].TickLimit
	ss.DeliveryStats = DeliveryStats{}
	logger.Info("CHAIN", "Started: %s (%d ticks)", steps[0].Name, steps[0].TickLimit)
}

// ScenarioChainTick advances the scenario chain.
func ScenarioChainTick(ss *SwarmState) {
	chain := ss.ScenarioChain
	if chain == nil || !chain.Active {
		return
	}

	chain.Timer--
	if chain.Timer > 0 {
		return
	}

	// Step complete — record score
	score := ss.DeliveryStats.CorrectDelivered*30 - ss.DeliveryStats.WrongDelivered*10
	chain.StepScores[chain.StepIdx] = score
	chain.TotalScore += score

	currentStep := &chain.Steps[chain.StepIdx]
	logger.Info("CHAIN", "Step %d complete: %s — Score:%d (Total:%d)",
		chain.StepIdx+1, currentStep.Name, score, chain.TotalScore)

	// Determine next step (branching logic)
	nextIdx := chain.StepIdx + 1

	// Branch function takes priority
	if currentStep.BranchFunc != nil {
		branchTarget := currentStep.BranchFunc(ss, score)
		if branchTarget >= 0 && branchTarget < len(chain.Steps) {
			nextIdx = branchTarget
		}
	}

	// MinScore check: if score below threshold, jump to FailStep
	if currentStep.MinScore > 0 && score < currentStep.MinScore {
		if currentStep.FailStep >= 0 && currentStep.FailStep < len(chain.Steps) {
			nextIdx = currentStep.FailStep
			logger.Info("CHAIN", "Score %d < min %d — branching to step %d",
				score, currentStep.MinScore, nextIdx)
		} else {
			// FailStep = -1 means end chain
			chain.Active = false
			chain.Complete = true
			logger.Info("CHAIN", "Score %d < min %d — chain failed! Total: %d",
				score, currentStep.MinScore, chain.TotalScore)
			return
		}
	}

	if nextIdx >= len(chain.Steps) {
		chain.Active = false
		chain.Complete = true
		logger.Info("CHAIN", "All steps complete! Total score: %d", chain.TotalScore)
		return
	}

	chain.StepIdx = nextIdx
	chain.StepPath = append(chain.StepPath, nextIdx)

	// Setup next step
	step := &chain.Steps[chain.StepIdx]
	step.Setup(ss)
	chain.Timer = step.TickLimit
	ss.DeliveryStats = DeliveryStats{}
	logger.Info("CHAIN", "Next: %s (%d ticks)", step.Name, step.TickLimit)
}

// ScenarioChainStop ends the chain.
func ScenarioChainStop(ss *SwarmState) {
	if ss.ScenarioChain != nil {
		ss.ScenarioChain.Active = false
	}
}

// BuildChainFromTemplate converts a ScenarioTemplate into executable steps.
func BuildChainFromTemplate(tmpl *ScenarioTemplate) []ScenarioChainStep {
	if tmpl == nil {
		return nil
	}
	steps := make([]ScenarioChainStep, len(tmpl.Steps))
	for i, def := range tmpl.Steps {
		d := def // capture
		steps[i] = ScenarioChainStep{
			Name:      d.Name,
			TickLimit: d.TickLimit,
			MinScore:  d.MinScore,
			FailStep:  d.FailStep,
			Setup: func(ss *SwarmState) {
				ss.ObstaclesOn = d.Obstacles
				if d.Obstacles {
					GenerateSwarmObstacles(ss)
				} else {
					ss.Obstacles = nil
				}
				ss.MazeOn = d.Maze
				if d.Maze {
					GenerateSwarmMaze(ss)
				} else {
					ss.MazeWalls = nil
				}
				ss.DeliveryOn = d.Delivery
				if d.Delivery {
					ss.ResetDeliveryState()
					GenerateDeliveryStations(ss)
				}
				ss.ResetBots()
			},
		}
	}
	return steps
}

// ScenarioChainStartCustom begins a chain from a template.
func ScenarioChainStartCustom(ss *SwarmState, tmpl *ScenarioTemplate) {
	steps := BuildChainFromTemplate(tmpl)
	if len(steps) == 0 {
		return
	}
	ss.ScenarioChain = &ScenarioChainState{
		Active:     true,
		StepIdx:    0,
		Steps:      steps,
		StepScores: make([]int, len(steps)),
		StepPath:   []int{0},
	}
	steps[0].Setup(ss)
	ss.ScenarioChain.Timer = steps[0].TickLimit
	ss.DeliveryStats = DeliveryStats{}
	logger.Info("CHAIN", "Custom chain '%s' started: %s (%d ticks)",
		tmpl.Name, steps[0].Name, steps[0].TickLimit)
}

// GetDefaultTemplate returns the default 3-step scenario as a template.
func GetDefaultTemplate() *ScenarioTemplate {
	return &ScenarioTemplate{
		Name:        "Standard-Kette",
		Description: "3 Stufen: einfach, Hindernisse, Labyrinth",
		Steps: []ScenarioStepDef{
			{Name: "1. Einfache Lieferung", TickLimit: 3000, Delivery: true},
			{Name: "2. Mit Hindernissen", TickLimit: 3000, Obstacles: true, Delivery: true},
			{Name: "3. Labyrinth", TickLimit: 4000, Maze: true, Delivery: true},
		},
	}
}

// ChainProgress returns current step / total steps as 0..1.
func ChainProgress(chain *ScenarioChainState) float64 {
	if chain == nil || len(chain.Steps) == 0 {
		return 0
	}
	return float64(chain.StepIdx) / float64(len(chain.Steps))
}
