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

	logger.Info("CHAIN", "Step %d complete: %s — Score:%d (Total:%d)",
		chain.StepIdx+1, chain.Steps[chain.StepIdx].Name, score, chain.TotalScore)

	chain.StepIdx++
	if chain.StepIdx >= len(chain.Steps) {
		chain.Active = false
		chain.Complete = true
		logger.Info("CHAIN", "All steps complete! Total score: %d", chain.TotalScore)
		return
	}

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
