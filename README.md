# SwarmSim — Swarm Robotics & Artificial Life Simulator

**[Live Demo (WebAssembly)](https://henning-heisenberg.github.io/swarmsim/)**

> A 2D swarm robotics simulator with 80+ scientific subsystems — from neural
> networks and genetic algorithms to immune systems, stock markets, and
> emergent language. Features its own scripting language (SwarmScript),
> genetic programming, multiplayer arenas, and real-time analytics.
> Built in Go with [Ebiten](https://ebitengine.org/).
>
> **~113,000 lines of Go** | **12 test suites** | **373 source files**

## Highlights

- **Programmable Bots** — Write behavior rules in SwarmScript, a domain-specific language with 30+ sensors and 25+ actions
- **Genetic Programming** — Programs evolve themselves through crossover and mutation
- **Neural Brains** — NeuroBrain, LSTM, Hebbian Learning, Neural Pruning, Neural Architecture Search
- **Evolution Lab** — Genetic algorithms, sexual reproduction, diploid genetics, speciation, epigenetics
- **Biological Systems** — Morphogenesis, gene regulatory networks, immune systems, ecosystems, homeostasis
- **Social Intelligence** — Hierarchy, quorum sensing, democracy (ranked choice), emergent language
- **Economy** — Energy trading, stock market with strategy shares, Gini coefficient
- **Swarm Cognition** — Episodic memory, temporal pattern recognition, spatial memory, collective dreaming
- **Multiplayer** — Blue vs Red team competitions with shared arena
- **Real-time Analytics** — Heatmaps, fitness graphs, delivery rates, bot rankings
- **Logistics Simulation** — Truck unloading, package delivery, color-coded stations
- **Visual Block Editor** — Click-based programming without typing
- **Interactive Tutorial** — 15-step guided tour on first launch (F3)
- **Factory Mode (F5)** — Full warehouse logistics simulation with 1000+ autonomous robots, production chains, LKW loading/unloading, 3 robot types, energy economics, customer orders
- **20 Optimization Algorithms** — GWO, WOA, PSO, DE, Cuckoo Search, ABC, Bat, HHO, and 12 more — with live math formula display (K key)
- **7 Languages** — German, English, French, Spanish, Portuguese, Italian, Ukrainian — switchable at runtime
- **Unicode Font Rendering** — JetBrains Mono with full Cyrillic support
- **Live Math Overlay** — See exactly how each algorithm calculates bot positions, with color-coded formulas
- **Self-Programming Swarm** — Bots detect problems and request AI-generated code. Proven solutions spread through the swarm. Optional Claude API for real LLM code generation
- **Interactive Learning** — 12 guided lessons from beginner to expert, with challenges and star ratings
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

- Go 1.23+ (tested with Go 1.25)
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

### F3: Tutorial

15-step interactive guided tour covering SwarmScript basics,
delivery mode, bot selection, follow-cam, block editor, and feature toggles.

### F4: Algo-Labor

Side-by-side comparison of 20 bio-inspired optimization algorithms
on configurable fitness landscapes. Live math overlay, radar chart,
and auto-tournament benchmarking.

### F5: Factory Mode

Full autonomous warehouse logistics simulation with 1000+ robots,
production chains, LKW operations, energy economics, and real-time KPIs.

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

## Factory Mode (F5)

A complete autonomous warehouse simulation:

- **1000-2000 robots** navigate a multi-zone factory (Receiving -> Storage -> Production -> Shipping)
- **3 robot types**: Transporter (60%), Forklift with pallets (25%), Express courier (15%)
- **Production chain**: Inbound LKW -> QC inspection -> Kanban-pull machines -> Multi-step recipes -> Outbound LKW
- **Smart task assignment**: Bots pick nearest tasks, avoid congestion, communicate charger availability
- **Energy economics**: Day/night pricing, budget management, bot purchase/sale
- **Shift system**: 70/30 rotation with handover, maintenance scheduling
- **Random events**: Supply shortage, rush orders, power outages, efficiency bonuses
- **Real-time KPIs**: OEE, throughput, quality rate, bottleneck detection

## Algo-Labor (F4)

Compare 20 bio-inspired optimization algorithms side-by-side:

GWO (Grey Wolf) | WOA (Whale) | PSO (Particle Swarm) | DE (Differential Evolution) | Cuckoo Search | ABC (Bee Colony) | BFO (Bacterial) | MFO (Moth-Flame) | Bat | HHO (Harris Hawks) | SSA (Salp Swarm) | GSA (Gravitational) | FPA (Flower) | SA (Simulated Annealing) | AO (Aquila) | SCA (Sine Cosine) | DA (Dragonfly) | TLBO (Teaching) | EO (Equilibrium) | Jaya

- **Live Math Overlay (K)**: See the exact formula with live values for the selected bot
- **Radar Chart**: Compare algorithm performance across 4 metrics
- **Auto-Tournament**: Benchmark all algorithms automatically
- **4 Fitness Landscapes**: Gaussian Peaks, Rastrigin, Ackley, Rosenbrock

## Languages

SwarmSim supports 7 languages, switchable at runtime via the language button in the tab bar:

Deutsch | English | Francais | Espanol | Portugues | Italiano | Ukrainska

1325+ translated strings covering all UI, help text, tooltips, and achievements.

## Advanced Subsystems (80+)

SwarmSim contains a rich library of scientific subsystems, each following the `Init*/Tick*/Clear*` pattern and stored in `domain/swarm/`. All subsystems are independent, toggleable, and fully tested.

### Neural & Learning Systems

| Module | File | Description |
|--------|------|-------------|
| **NeuroBrain** | `neuro.go` | Feedforward neural network with configurable hidden layer |
| **LSTM** | `lstm.go` | Long Short-Term Memory networks for sequential decision-making |
| **Hebbian Learning** | `hebbian.go` | Reward-modulated Hebbian plasticity with eligibility traces |
| **Neural Pruning** | `neural_pruning.go` | Neuronaler Darwinismus — oversize brains get pruned to efficient circuits |
| **Neural Architecture Search** | `nas.go`, `nas_evolution.go` | NEAT-style topology evolution, bots evolve their own network structure |
| **Reinforcement Learning** | `rl.go` | Q-Learning with epsilon-greedy exploration |
| **Behavior Trees** | `behavior_tree.go` | Sequence/Selector/Inverter/Repeater nodes as brain type |
| **Learning Classifier System** | `classifier.go` | IF-THEN rule populations per bot, rules compete by strength |

### Evolution & Genetics

| Module | File | Description |
|--------|------|-------------|
| **Genetic Programming** | (engine/swarmscript) | SwarmScript programs evolve via crossover and mutation |
| **Sexual Reproduction** | `sexual_reproduction.go` | Diploid genetics with mate selection |
| **Advanced Diploid Genetics** | `diploid.go` | Dominance, co-dominance, heterozygote advantage |
| **Speciation** | `speciation.go` | Genetic distance-based species formation |
| **Meta-Evolution** | `meta_evolution.go` | Mutation rate, crossover rate, selection pressure evolve themselves |
| **Epigenetics** | `epigenetics.go` | Heritable methylation/acetylation marks from environmental experience |
| **GP Evolution** | `gp_evolution.go` | Genetic programming operators for tree-based programs |
| **Pareto (NSGA-II)** | `pareto.go` | Multi-objective optimization with Pareto fronts |
| **Novelty Search** | `novelty.go` | Reward behavioral novelty instead of just fitness |
| **Body Evolution** | `body_evolution.go` | Evolvable size, speed, sensor range, carry capacity with trade-offs |
| **Morphological Evolution** | `morphology.go` | Evolvable body parameters per bot |
| **Interactive Evolution** | `interactive_evo.go` | User-guided selection (Karl Sims style) |

### Biological Systems

| Module | File | Description |
|--------|------|-------------|
| **Gene Regulatory Networks** | `grn.go` | Regulation matrix, gene expression, sigmoid activation |
| **Gene Cascades** | `gene_cascade.go` | Regulatory chains: Gene A activates B activates C |
| **Morphogenesis** | `morphogenesis.go` | Turing patterns via activator-inhibitor reaction-diffusion on bots |
| **Reaction-Diffusion** | `reaction_diffusion.go` | Gray-Scott model with bot chemotaxis |
| **Immune System** | `immune.go` | Detector cells, anomaly scoring, threat memory |
| **Adaptive Immunity** | `adaptive_immune.go` | B/T-cells, memory cells, clonal selection, 10x faster secondary response |
| **Ecosystem** | `ecosystem.go` | Plants, herbivores, predators with metabolic cost and respawn |
| **Predator-Prey** | `predator_prey.go` | Co-evolutionary arms race |
| **Homeostasis** | `homeostasis.go` | 4 internal drives (energy, stress, curiosity, safety) regulate behavior |

### Communication & Social

| Module | File | Description |
|--------|------|-------------|
| **Emergent Language** | `language.go` | 8-symbol vocabulary, encode/decode tables, shared meaning metric |
| **Language Evolution** | `language_evo.go` | Continuous signal vectors with evolved encoding/decoding networks |
| **Quorum Sensing** | `quorum.go` | Local voting with 4 proposals, social influence, quorum detection |
| **Democracy** | `democracy.go` | Ranked choice voting with instant-runoff, emergent parties |
| **Hierarchy** | `hierarchy.go` | Squads, platoons, companies with leader election |
| **Specialization** | `specialization.go` | Division of labor with experience-based role assignment |
| **Cooperative Learning** | `cooperative.go` | Knowledge transfer between bots |

### Memory & Cognition

| Module | File | Description |
|--------|------|-------------|
| **Episodic Memory** | `episodic_memory.go` | Per-bot memories with decay, replay, spatial triggers |
| **Spatial Memory** | `spatial_memory.go` | Shared knowledge grid with resource/traffic/delivery scores |
| **Temporal Memory** | `temporal_memory.go` | Temporal pattern recognition — bots anticipate periodic events |
| **Collective Dreaming** | `collective_dream.go` | Offline strategy replay and recombination during low activity |
| **Swarm Dreams** | `dreams.go` | Experience replay with consolidation |
| **Bot Memory** | `memory.go` | Spatial memory with exponential decay |

### Navigation & Stigmergy

| Module | File | Description |
|--------|------|-------------|
| **Pheromone Trails** | `swarm_pheromone.go` | Multi-channel pheromone grid with directional data |
| **Stigmergy** | `stigmergy.go` | Collective building like termites |
| **Stigmergy 2.0** | `stigmergy2.go` | Compound pheromone messages (type + direction + age) |
| **Shape Formation** | `shape_formation.go` | 8 shapes, greedy nearest-neighbor assignment, rotation |
| **Formations** | `formation.go` | 5 formation types with smooth morphing |
| **Oscillators** | `oscillator.go` | Kuramoto model — firefly synchronization, order parameter |

### Economy & Competition

| Module | File | Description |
|--------|------|-------------|
| **Energy Economy** | `energy_economy.go` | Wallets, trading, 4 roles (Worker/Trader/Hoarder/Altruist), Gini coefficient |
| **Stock Market** | `stock_market.go` | Strategy shares traded by bots, emergent bubbles and crashes |
| **Multi-Swarm Arena** | `multi_swarm.go` | Competing teams with territory control |
| **Tournament** | `tournament.go` | Competitive evaluation system |

### Analytics & Tools

| Module | File | Description |
|--------|------|-------------|
| **Benchmarks** | `benchmark.go` | 6 standardized scenarios (Foraging, Exploration, Clustering, ...) |
| **Diversity Metrics** | `diversity.go` | Population diversity tracking |
| **Statistics** | `stats.go` | Runtime statistics and aggregation |
| **Timelapse** | `timelapse.go` | Window-based metric aggregation and trends |
| **Replay** | `replay.go` | Seek/rewind/step-through for recorded simulations |
| **Checkpoints** | `checkpoint.go` | Save/restore full simulation state as JSON |
| **Genealogy** | `genealogy.go` | Family tree tracking across generations |
| **Heatmap** | `heatmap.go` | Spatial activity density visualization |
| **Sensitivity Analysis** | `sensitivity.go` | Automated parameter variation framework |
| **Auto-Optimizer** | `auto_optimizer.go` | Convergence detection + config save/restore |
| **AST Visualizer** | `ast_visualizer.go` | SwarmScript program tree layout |

### Environment & Effects

| Module | File | Description |
|--------|------|-------------|
| **Weather** | `weather.go` | Wind, rain, fog, storms with visibility and force effects |
| **Terrain** | `terrain.go` | Heightmap + biomes with speed modifiers |
| **Sensor Noise** | `sensor_noise.go` | Realistic sensor failures and noise pattern learning |
| **Moving Obstacles** | `moving_obstacles.go` | Patrol and rotation obstacles |
| **Dynamic Environment** | `dynamic_env.go` | Changing environment conditions |

### Infrastructure

| Module | File | Description |
|--------|------|-------------|
| **Plugin System** | `plugin.go` | Mod/extension system for SwarmState |
| **Curriculum Learning** | `curriculum.go` | Automatic difficulty progression |
| **Transfer Learning** | `transfer.go` | Genome export/import between scenarios |
| **Scenario Chains** | `scenario_chain.go` | Branching scenario templates |
| **Presets** | `presets.go` | 20 built-in SwarmScript programs |
| **Shader Config** | `shader_config.go` | 6 Kage GPU shader effects |
| **Leaderboard** | `leaderboard.go` | Persistent high-score system |
| **Achievements** | `achievements.go` | Progress tracking and unlockables |

## Architecture

```
swarmsim/
├── domain/              Core logic (no rendering dependencies)
│   ├── bot/             Bot types and behavior interfaces
│   ├── swarm/           SwarmBot + 80+ subsystems (see above)
│   ├── factory/         Factory mode logic (robots, machines, trucks, orders)
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
| **F1** | Classic Mode |
| **F2** | Swarm Lab |
| **F3** | Tutorial |
| **F4** | Algo-Labor |
| **F5** | Factory Mode |
| **F10** | Screenshot (PNG) |
| **F11** | GIF Recording |
| **H** | Help overlay |
| **Space** | Pause / Resume |
| **+/-** | Simulation speed (0.5x - 5.0x) |
| **ESC** | Back / Quit |

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

### Swarm Lab / Algo-Labor

| Key | Action |
|-----|--------|
| **K** | Live Math Overlay (formula display) |
| **1-5** | Speed presets |
| **Click** | Select bot |
| **Mouse wheel** | Zoom |

### Factory Mode (F5)

| Key | Action |
|-----|--------|
| **WASD** | Pan camera |
| **1-5** | Speed (1x-20x) |
| **Click** | Select bot / Toggle machine |
| **F** | Follow selected bot |
| **M** | Heatmap overlay |
| **H** | Help overlay |
| **P** | Maintenance planner |
| **E** | Emergency evacuation |
| **T** | Spawn inbound truck |
| **Shift+T** | Spawn outbound truck |
| **B** | Buy 10 bots ($5000) |
| **V** | Sell 10 bots (+$2000) |
| **X** | Export stats to clipboard |

## Tech Stack

- **Go 1.23+** (tested with Go 1.25, no CGO)
- **Ebiten v2.9** (2D game library)
- **JetBrains Mono** font (SIL OFL) with full Cyrillic support
- **text/v2** for Unicode rendering
- Cross-compile: Windows, Linux, WebAssembly
- **~113,000 lines of Go** | **12 test suites** | **373 source files**

## Contributing

See [ARCHITECTURE.md](ARCHITECTURE.md) for the subsystem pattern and how to add new features.

## License

MIT — see [LICENSE](LICENSE)
