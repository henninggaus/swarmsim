# SwarmSim — Swarm Robotics Simulator

**[Live Demo (WebAssembly)](https://henning-heisenberg.github.io/swarmsim/)**

> A 2D swarm robotics simulator with its own scripting language (SwarmScript),
> genetic programming, multiplayer arenas, and real-time analytics.
> Built in Go with [Ebiten](https://ebitengine.org/).

## Highlights

- **Programmable Bots** — Write behavior rules in SwarmScript, a domain-specific language with 30+ sensors and 25+ actions
- **Genetic Programming** — Programs evolve themselves through crossover and mutation
- **Multiplayer** — Blue vs Red team competitions with shared arena
- **Real-time Analytics** — Heatmaps, fitness graphs, delivery rates, bot rankings
- **Logistics Simulation** — Truck unloading, package delivery, color-coded stations
- **Visual Block Editor** — Click-based programming without typing
- **Interactive Tutorial** — 15-step guided tour on first launch (F3)
- **WebAssembly** — Runs in the browser, no install needed

## Quick Start

```bash
git clone https://github.com/henning-heisenberg/swarmsim.git
cd swarmsim
go run .
```

Press **F2** for Swarm Lab, select **"Simple Delivery"** from the dropdown, click **DEPLOY**.
Press **H** for the full keyboard reference.

### Requirements

- Go 1.21+ (tested with Go 1.25)
- No CGO dependencies

## SwarmScript Language

SwarmScript is a rule-based DSL for programming swarm behavior.
Every line is a rule: `IF <condition> [AND ...] THEN <action>`.
All matching rules execute each tick (not just the first match).

### Example: Package Delivery

```
# Pick up package when near a pickup station that has one
IF carry == 0 AND p_dist < 20 AND has_pkg == 1 THEN PICKUP

# Navigate to matching dropoff
IF carry == 1 AND match == 1 THEN GOTO_DROPOFF

# Drop package at matching station
IF carry == 1 AND d_dist < 25 AND match == 1 THEN DROP

# Collision avoidance
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF obs_ahead == 1 THEN AVOID_OBSTACLE

# Default: move forward
IF true THEN FWD
```

### Sensor Reference

**Navigation & Neighbors**

| Sensor | Alias | Description |
|--------|-------|-------------|
| `nearest_distance` | `near_dist` | Distance to nearest bot |
| `neighbors_count` | `nbrs` | Number of neighbors in sensor range |
| `on_edge` | `edge` | At arena boundary? (0/1) |
| `obstacle_ahead` | `obs_ahead` | Obstacle in front? (0/1) |
| `obstacle_distance` | `obs_dist` | Distance to nearest obstacle |
| `light_value` | `light` | Light intensity (0-100) |
| `random` | `rnd` | Random value (0-100) |
| `tick` | — | Global simulation tick |

**Internal State**

| Sensor | Alias | Description |
|--------|-------|-------------|
| `state` | `my_state` | Internal state variable (0-255) |
| `counter` | — | Counter variable (0-255) |
| `timer` | — | Countdown timer (ticks) |
| `value1`, `value2` | — | User-defined variables |
| `has_leader` | `leader` | Following another bot? |
| `chain_length` | `chain_len` | Follow-chain length |
| `received_message` | `msg` | Message received? |

**Delivery**

| Sensor | Alias | Description |
|--------|-------|-------------|
| `carrying` | `carry` | Carrying a package? (0/1) |
| `nearest_pickup_dist` | `p_dist` | Distance to nearest pickup |
| `nearest_pickup_has_package` | `has_pkg` | Pickup has package ready? |
| `nearest_dropoff_dist` | `d_dist` | Distance to nearest dropoff |
| `dropoff_match` | `match` | Dropoff matches package color? |
| `nearest_matching_led_dist` | `led_dist` | Distance to bot with matching LED |
| `heard_pickup_color` | `heard_pickup` | Pickup color heard via message |
| `heard_dropoff_color` | `heard_dropoff` | Dropoff color heard via message |
| `exploring` | `lost` | No dropoff/beacon in sight? |

**Truck & Maze**

| Sensor | Alias | Description |
|--------|-------|-------------|
| `on_ramp` | — | Bot on truck ramp? |
| `truck_here` | — | Truck parked and ready? |
| `truck_pkg_count` | — | Packages remaining on truck |
| `heard_beacon` | — | Beacon signal from dropoff? |
| `wall_right` | — | Wall within 25px on right? |
| `wall_left` | — | Wall within 25px on left? |
| `pheromone` | `pher` | Pheromone intensity ahead (0-100) |

**Teams**

| Sensor | Description |
|--------|-------------|
| `team` | Team membership (1=A, 2=B) |
| `team_score` | Own team's score |
| `enemy_score` | Opposing team's score |

**Evolution Parameters**

```
IF near_dist < $A:15 THEN TURN_FROM_NEAREST
```

`$A:15` = Parameter A with default value 15. When Evolution is enabled, each bot gets its own value for each parameter. Natural selection optimizes these values over generations.

### Action Reference

**Movement**

| Action | Description |
|--------|-------------|
| `FWD` / `FWD_SLOW` | Move forward (normal / slow) |
| `STOP` | Stop moving |
| `TURN_LEFT N` | Turn left by N degrees |
| `TURN_RIGHT N` | Turn right by N degrees |
| `TURN_RANDOM` | Turn to random direction |
| `TURN_TO_NEAREST` | Turn toward nearest bot |
| `TURN_FROM_NEAREST` | Turn away from nearest bot |
| `TURN_TO_CENTER` | Turn toward neighbor centroid |
| `TURN_TO_LIGHT` | Turn toward light source |
| `AVOID_OBSTACLE` | Dodge obstacle ahead |
| `SPIRAL` | Expanding spiral search |

**Delivery**

| Action | Description |
|--------|-------------|
| `PICKUP` | Pick up package |
| `DROP` | Drop package |
| `GOTO_PICKUP` | Turn toward nearest pickup |
| `GOTO_DROPOFF` / `GOTO_MATCH` | Turn toward matching dropoff |
| `GOTO_LED` | Turn toward bot with matching LED |
| `GOTO_RAMP` | Navigate to truck ramp |
| `GOTO_BEACON` | Navigate to beacon signal |
| `LED_PICKUP` | Set LED to nearest pickup color |
| `LED_DROPOFF` | Set LED to nearest dropoff color |

**Communication & State**

| Action | Description |
|--------|-------------|
| `SET_STATE N` | Set internal state (0-9) |
| `SET_LED R G B` | Set LED color (RGB 0-255) |
| `COPY_LED` | Copy nearest bot's LED color |
| `SEND_MESSAGE N` | Broadcast message |
| `SEND_PICKUP N` | Broadcast pickup color info |
| `SEND_DROPOFF N` | Broadcast dropoff color info |
| `FOLLOW_NEAREST` | Start following nearest bot |
| `UNFOLLOW` | Stop following |

**Maze Navigation**

| Action | Description |
|--------|-------------|
| `WALL_FOLLOW_RIGHT` | Right-hand wall following |
| `WALL_FOLLOW_LEFT` | Left-hand wall following |
| `FOLLOW_PHER` | Follow pheromone gradient |

### Preset Programs (20)

| Preset | Category | Description |
|--------|----------|-------------|
| Aggregation | Behavior | Bots cluster toward center |
| Dispersion | Behavior | Bots spread out evenly |
| Orbit | Behavior | Circle around light source |
| Color Wave | Communication | LED color wave through swarm |
| Flocking | Behavior | Boids-style swarming |
| Snake Formation | Following | Bots form chains |
| Obstacle Nav | Navigation | Navigate around obstacles to light |
| Pulse Sync | Communication | Synchronized LED pulses (fireflies) |
| Trail Follow | Following | Bots follow and copy LED trails |
| Ant Colony | Foraging | Simplified ant colony optimization |
| Simple Delivery | Delivery | Random exploration + package delivery |
| Delivery Comm | Delivery | With station position sharing |
| Delivery Roles | Delivery | 50% scouts, 50% carriers |
| Simple Unload | Truck | Basic truck unloading |
| Coordinated Unload | Truck | LED-gradient + beacon coordination |
| Evolving Delivery | Evolution | Delivery with evolvable parameters |
| Evolving Truck | Evolution | Truck unloading with evolution |
| Maze Explorer | Maze | Wall-following maze navigation |
| GP: Random Start | GP | Fully random genetic programs |
| GP: Seeded Start | GP | Seeded from Simple Delivery |

## Modes

### F1: Classic Mode

Traditional swarm simulation with 5 bot types (Scout, Worker, Leader, Tank, Healer).
Scenarios include Sandbox, Foraging Paradise, Labyrinth, Energy Crisis, and Evolution Arena.
Features pheromone trails, energy management, and genetic algorithms.

### F2: Swarm Lab

The main mode. Full SwarmScript editor with visual block editor.
Features: Delivery system, Trucks, Maze, Obstacles, Light source,
Parameter Evolution, Genetic Programming, Teams, and Dashboard.

## Features In Detail

### Genetic Programming (GP)

Each bot gets its own randomly generated SwarmScript program.
Every 2000 ticks: fitness evaluation, selection (top 20%), crossover, mutation.
Top 3 programs preserved as elite. After enough generations, bots develop
delivery strategies no human wrote. Use **"Export Best"** to save the evolved program.

**Fitness:** `Deliveries*30 + Pickups*15 + Distance*0.01 - StuckCount*10 - IdleTicks*0.05`

### Multiplayer (Teams)

Two teams (Blue A, Red B) compete in the same arena with separate programs.
**Challenge mode** (C key): 5000 ticks, team with more correct deliveries wins.
New round (N key) resets scores and positions. Each team can run a different program.

### Truck Unloading

Trucks drive in and park at the ramp. Bots pick up packages from the truck
and deliver them to color-coded dropoff stations. Ramp semaphore limits
concurrent access to 3 bots. After all packages are unloaded,
the truck leaves and the next one arrives.

### Statistics Dashboard (D key)

Real-time dashboard with five panels:
- **Fitness Graph** — Best/average fitness over generations
- **Delivery Rate** — Bar chart per 500-tick window
- **Heatmap** — Bot movement density (blue to red)
- **Ranking** — Top 5 bots by deliveries
- **Event Ticker** — Live pickup/delivery events

### Interactive Tutorial (F3)

15-step guided tour covering SwarmScript basics, delivery mode,
bot selection, follow-cam, block editor, and feature toggles.
Starts automatically on first launch, skippable with ESC.

## Architecture

```
swarmsim/
├── domain/              Core logic (no rendering dependencies)
│   ├── bot/             Bot types and behavior interfaces
│   ├── swarm/           SwarmBot, delivery, GP evolution, teams, stats
│   ├── physics/         Collision detection, spatial hash, arena
│   ├── comm/            Decentralized message passing (TTL, range)
│   ├── genetics/        Genome, crossover, mutation, fitness
│   └── resource/        Resource spawning and management
├── engine/              Simulation engine
│   ├── simulation/      Game loop, scenarios, configuration
│   ├── swarmscript/     Parser, interpreter, GP operators
│   └── pheromone/       Pheromone grid (diffusion, evaporation)
├── render/              All rendering (Ebiten)
│   ├── renderer.go      Camera, bot sprites, pheromone rendering
│   ├── swarm_render.go  Swarm mode arena + HUD
│   ├── swarm_editor.go  Code editor with syntax highlighting
│   ├── dashboard.go     Statistics dashboard (graphs, heatmap)
│   ├── tutorial.go      Interactive 15-step tutorial
│   ├── tooltips.go      Hover tooltips for all UI elements
│   ├── help.go          Help overlay (H key)
│   ├── minimap.go       150x100px overview map
│   ├── capture.go       Screenshot (PNG) and GIF recording
│   └── particles.go     Particle effects
└── main.go              Ebiten game loop, input handling
```

### Performance

- **SpatialHash** — O(1) neighbor lookup with pre-allocated flat slices
- **Text Cache** — HUD text rendered as cached GPU images (120-frame eviction)
- **Pheromone Cache** — Pixel buffer recalculated every 5 ticks only
- **Bot Sprites** — Pre-rendered 24x24px triangles with ColorScale tinting

## Building

```bash
make build          # Linux/Mac binary
make windows        # Windows .exe (cross-compile)
make wasm           # WebAssembly → docs/swarmsim.wasm
go test ./...       # Run all tests
```

Or directly:

```bash
go build -o swarmsim .
./swarmsim
```

## Keyboard Shortcuts

### Global

| Key | Action |
|-----|--------|
| **Space** | Pause / Resume |
| **+/-** | Simulation speed (0.5x - 5.0x) |
| **F1** | Classic Mode |
| **F2** | Swarm Lab |
| **F3** | Start Tutorial |
| **F10** | Screenshot (PNG) |
| **F11** | GIF Recording |
| **H** | Help overlay |
| **ESC** | Quit |

### Swarm Lab (F2)

| Key | Action |
|-----|--------|
| **Click** | Select bot / UI element |
| **T** | Toggle trails |
| **L** | Place / remove light source |
| **C** | Show routes / start challenge (teams) |
| **D** | Toggle statistics dashboard |
| **M** | Toggle minimap |
| **N** | New round (trucks / teams) |
| **F** | Follow selected bot |
| **Tab** | Filter log to selected bot |

## Tech Stack

- **Go 1.21+** (no CGO)
- **Ebiten v2** (2D game library)
- Cross-compile: Windows, Linux, WebAssembly
- No external dependencies beyond Ebiten

## License

MIT
