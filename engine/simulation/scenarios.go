package simulation

import (
	"math/rand"
	"swarmsim/domain/physics"
)

// GetScenarios returns all available scenarios.
func GetScenarios() []Scenario {
	return []Scenario{
		scenarioForaging(),
		scenarioLabyrinth(),
		scenarioEnergy(),
		scenarioSandbox(),
		scenarioEvolution(),
	}
}

// GetClassicScenarios returns the 5 classic scenarios for the Classic Mode dropdown.
// Order: Sandbox (default), Foraging, Labyrinth, Energy Crisis, Evolution.
func GetClassicScenarios() []Scenario {
	return []Scenario{
		scenarioSandbox(),
		scenarioForaging(),
		scenarioLabyrinth(),
		scenarioEnergy(),
		scenarioEvolution(),
	}
}

func scenarioForaging() Scenario {
	cfg := DefaultConfig()
	cfg.ArenaWidth = 2000
	cfg.ArenaHeight = 1500
	cfg.InitObstacles = 0
	cfg.InitResources = 50
	cfg.RespawnInterval = 60
	cfg.HomeBaseX = 1000
	cfg.HomeBaseY = 750
	return Scenario{
		ID:   ScenarioForaging,
		Name: "FORAGING PARADISE",
		Cfg:  cfg,
	}
}

func scenarioLabyrinth() Scenario {
	cfg := DefaultConfig()
	cfg.InitScouts = 15
	cfg.InitWorkers = 10
	cfg.InitLeaders = 2
	cfg.InitTanks = 8
	cfg.InitHealers = 5
	cfg.InitObstacles = 0 // custom maze instead
	cfg.InitResources = 20
	return Scenario{
		ID:   ScenarioLabyrinth,
		Name: "LABYRINTH",
		Cfg:  cfg,
		CustomSetup: func(s *Simulation) {
			generateMaze(s)
		},
	}
}

func scenarioEnergy() Scenario {
	cfg := DefaultConfig()
	cfg.InitResources = 10
	cfg.RespawnInterval = 200
	cfg.EnergyDecayMult = 2.0
	cfg.InitHealers = 8
	return Scenario{
		ID:   ScenarioEnergy,
		Name: "ENERGY CRISIS",
		Cfg:  cfg,
	}
}

func scenarioSandbox() Scenario {
	cfg := DefaultConfig()
	return Scenario{
		ID:   ScenarioSandbox,
		Name: "SANDBOX",
		Cfg:  cfg,
	}
}

func scenarioEvolution() Scenario {
	cfg := DefaultConfig()
	cfg.GenerationLength = 500
	cfg.AutoEvolve = true
	cfg.InitResources = 25
	return Scenario{
		ID:   ScenarioEvolution,
		Name: "EVOLUTION ARENA",
		Cfg:  cfg,
	}
}

// generateMaze creates a maze-like obstacle layout using recursive backtracker.
func generateMaze(s *Simulation) {
	mazeCols := 10
	mazeRows := 8
	cellW := s.Cfg.ArenaWidth / float64(mazeCols)
	cellH := s.Cfg.ArenaHeight / float64(mazeRows)
	wallThick := 8.0

	type cell struct {
		visited bool
		walls   [4]bool // N, E, S, W
	}

	cells := make([][]cell, mazeCols)
	for c := range cells {
		cells[c] = make([]cell, mazeRows)
		for r := range cells[c] {
			cells[c][r].walls = [4]bool{true, true, true, true}
		}
	}

	type pos struct{ c, r int }
	dirs := [4]pos{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	opposite := [4]int{2, 3, 0, 1}

	stack := []pos{{0, 0}}
	cells[0][0].visited = true
	rng := s.Rng

	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		var neighbors []int
		for d, dir := range dirs {
			nc, nr := curr.c+dir.c, curr.r+dir.r
			if nc >= 0 && nc < mazeCols && nr >= 0 && nr < mazeRows && !cells[nc][nr].visited {
				neighbors = append(neighbors, d)
			}
		}
		if len(neighbors) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		d := neighbors[rng.Intn(len(neighbors))]
		nc, nr := curr.c+dirs[d].c, curr.r+dirs[d].r
		cells[curr.c][curr.r].walls[d] = false
		cells[nc][nr].walls[opposite[d]] = false
		cells[nc][nr].visited = true
		stack = append(stack, pos{nc, nr})
	}

	// Convert walls to obstacles (only East and South interior walls)
	for c := 0; c < mazeCols; c++ {
		for r := 0; r < mazeRows; r++ {
			x := float64(c) * cellW
			y := float64(r) * cellH
			// East wall
			if c < mazeCols-1 && cells[c][r].walls[1] {
				s.Arena.Obstacles = append(s.Arena.Obstacles, &physics.Obstacle{
					X: x + cellW - wallThick/2, Y: y,
					W: wallThick, H: cellH, Pushable: true,
				})
			}
			// South wall
			if r < mazeRows-1 && cells[c][r].walls[2] {
				s.Arena.Obstacles = append(s.Arena.Obstacles, &physics.Obstacle{
					X: x, Y: y + cellH - wallThick/2,
					W: cellW, H: wallThick, Pushable: true,
				})
			}
		}
	}

	// Place resources at dead ends (cells with 3 walls)
	for c := 0; c < mazeCols; c++ {
		for r := 0; r < mazeRows; r++ {
			wallCount := 0
			for _, w := range cells[c][r].walls {
				if w {
					wallCount++
				}
			}
			if wallCount >= 3 {
				rx := float64(c)*cellW + cellW/2
				ry := float64(r)*cellH + cellH/2
				s.SpawnResourceAt(rx, ry)
			}
		}
	}
}

// RandomScenarioSeed returns a new random seed for variety.
func RandomScenarioSeed() int64 {
	return rand.Int63()
}
