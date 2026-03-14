# Schwarm-Robotik-Simulator

A 2D swarm robotics simulator built in Go with [Ebiten](https://ebitengine.org/).

## Build

```bash
# Standard build
go build -o swarmsim.exe .

# Cross-compile for Windows from Linux
GOOS=windows GOARCH=amd64 go build -o swarmsim.exe .
```

## Controls

| Key | Action |
|-----|--------|
| **Space** | Pause / Resume |
| **Left Click** | Select bot (shows info panel) |
| **Right Click Drag** | Pan camera |
| **Mouse Wheel** | Zoom in/out |
| **WASD** | Pan camera |
| **1-5** | Spawn Scout/Worker/Leader/Tank/Healer at cursor |
| **R** | Spawn resource at cursor |
| **H** | Place obstacle at cursor |
| **F** | Toggle communication radius visualization |
| **G** | Toggle sensor radius visualization |
| **D** | Toggle debug comm lines |
| **P** | Cycle pheromone visualization (OFF / FOUND / ALL) |
| **E** | Force end generation (evolve now) |
| **V** | Toggle genome overlay for selected bot |
| **F1** | Scenario: Foraging Paradise |
| **F2** | Scenario: Labyrinth |
| **F3** | Scenario: Energy Crisis |
| **F4** | Scenario: Sandbox |
| **F5** | Scenario: Evolution Arena |
| **+/-** | Increase/decrease simulation speed |
| **ESC** | Quit |

## Bot Types

| Type | Speed | Sensor | Comm Range | Carry | Special |
|------|-------|--------|------------|-------|---------|
| **Scout** (Cyan) | 3.0 | 150px | 80px | 0 | Explores, marks resources, deposits search pheromone |
| **Worker** (Orange) | 1.5 | 60px | 60px | 2 | Collects & transports, follows found-resource pheromone |
| **Leader** (Gold) | 1.0 | 100px | 200px | 0 | Coordinates workers, relays messages |
| **Tank** (Dark Green) | 0.8 | 50px | 50px | 0 | Pushes obstacles, responds to help requests |
| **Healer** (Pink) | 1.2 | 80px | 80px | 0 | Heals bots and recharges energy |

## Systems

### Pheromone System (ACO)
Three pheromone types: **Search** (blue), **Found Resource** (green), **Danger** (red). Bots deposit pheromones that evaporate over time and diffuse to neighbors. Scouts avoid search pheromone (to explore new areas), workers follow found-resource pheromone trails back to resources, and all bots avoid danger pheromone. Toggle visualization with **P**.

### Energy System
All bots have energy (0-100). Movement, messaging, carrying resources, depositing pheromones, and pushing obstacles cost energy. Bots recharge at the home base. Healers can transfer energy to nearby bots. At zero energy, bots become immobilized and send help requests. Low energy causes blinking; zero energy grays out the bot.

### Genetic Algorithm
Each bot has a 7-gene genome: FlockingWeight, PheromoneFollow, ExplorationDrive, CommFrequency, EnergyConservation, SpeedPreference, CooperationBias. Fitness is tracked per bot (distance traveled, resources delivered, messages relayed, bots healed, minus zero-energy ticks). At generation end, top 30% elite genomes are preserved; others are replaced by crossover offspring with mutation. Press **E** to force evolve, **V** to view a selected bot's genome.

### Scenario System
Five preset scenarios accessed via **F1-F5**. Each reconfigures arena size, bot counts, resource placement, obstacles, and energy/evolution parameters. Genomes are preserved across scenario switches.

## Architecture

- **Emergent behavior**: Swarm intelligence arises from simple local rules, no central control
- **Spatial hashing**: O(1) neighbor lookups for efficient large swarms
- **Fixed tick-rate**: Simulation runs at 30 ticks/sec independent of render FPS
- **Radius-based communication**: Bots can only communicate within their comm range
- **Message relay**: Messages have TTL of 3 ticks and can be forwarded by other bots
- **Pheromone grid**: Efficient off-screen image rendering with WritePixels
- **Per-type evolution**: Each bot type evolves independently with its own genome pool
