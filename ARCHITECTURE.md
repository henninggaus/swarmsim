# Architecture Guide

This document describes SwarmSim's internal architecture, project layout, simulation modes, and key subsystems.

## Project Structure

```
swarmsim/
├── domain/                     Pure logic (no rendering dependencies)
│   ├── bot/                    Bot types and behavior interfaces
│   ├── swarm/                  SwarmBot + algorithms + math trace (239 files, 63k LOC)
│   │   ├── swarm_struct.go       SwarmState central hub
│   │   ├── swarm_algorithms.go   Algorithm enum + switching
│   │   ├── algorithm_registry.go Central algo dispatch + trace hooks
│   │   ├── algo_common.go        Shared types (gridPt, idxFit, constants)
│   │   ├── swarm_steering.go     WrapAngle, steerToward, Levy flights
│   │   ├── math_trace.go         MathTrace / MathStep types
│   │   ├── math_trace_algos.go   20 algorithm trace functions
│   │   ├── presets.go            22 SwarmScript presets
│   │   ├── delivery.go           Package delivery system
│   │   ├── achievements.go       Achievement tracking
│   │   └── ...                   80+ subsystem files (Init/Tick/Clear pattern)
│   ├── factory/                Factory Mode (F5)
│   │   ├── factory_struct.go     FactoryState, all types
│   │   ├── factory_layout.go     Predefined factory layout
│   │   ├── factory_bot.go        Bot behavior state machine
│   │   ├── factory_bot_nav.go    Navigation + collision
│   │   ├── factory_bot_systems.go Shift/emergency/communication
│   │   ├── factory_task.go       Task queue + assignment + events
│   │   ├── factory_machine.go    Machine processing + overheating
│   │   ├── factory_truck.go      Truck lifecycle
│   │   ├── factory_charge.go     Energy + economics
│   │   ├── factory_repair.go     Maintenance + malfunctions
│   │   └── factory_util.go       Shared helpers
│   ├── physics/                Collision detection, spatial hash, arena
│   ├── comm/                   Decentralized message passing
│   ├── genetics/               Genome, crossover, mutation, fitness
│   └── resource/               Resource spawning and management
├── engine/
│   ├── simulation/
│   │   ├── simulation.go         Main sim struct + scenarios
│   │   ├── simulation_struct.go  Simulation fields + config
│   │   ├── swarm_ai.go           Swarm update loop
│   │   ├── swarm_ai_program.go   SwarmScript execution
│   │   ├── swarm_ai_physics.go   Bot physics (collision, movement)
│   │   ├── factory_ai.go         Factory update loop (15 phases)
│   │   ├── evolution.go          Evolutionary loop
│   │   ├── scenarios.go          Scenario definitions
│   │   └── config.go             Simulation configuration
│   ├── swarmscript/            Parser, interpreter, GP operators
│   └── pheromone/              Pheromone grid (diffusion, evaporation)
├── render/                     All rendering (Ebiten) — 52 files
│   ├── renderer.go               Camera, bot sprites, pheromone rendering
│   ├── text_render.go            JetBrains Mono font, text cache, printColoredAt
│   ├── colors.go                 Centralized color constants
│   ├── swarm_render.go           Swarm mode arena + HUD
│   ├── swarm_render_algo.go      Algorithm visualization
│   ├── swarm_render_algo_compare.go  Side-by-side comparison
│   ├── swarm_render_algo_overlays.go Algorithm-specific overlays
│   ├── swarm_render_delivery.go  Delivery visualization
│   ├── swarm_render_info.go      Info panels
│   ├── swarm_render_overlay.go   General overlays
│   ├── swarm_render_math.go      Live Math Overlay
│   ├── swarm_editor.go           Code editor with syntax highlighting
│   ├── swarm_block_editor.go     Block editor UI
│   ├── swarm_tabs.go             Tabbed panel
│   ├── factory_render.go         Factory mode main renderer
│   ├── factory_render_effects.go Factory visual effects
│   ├── factory_render_ui.go      Factory HUD / panels
│   ├── help.go                   Help overlay framework
│   ├── help_features.go          Feature explanations (650 lines)
│   ├── help_reference.go         SwarmScript reference
│   ├── tutorial.go               Interactive 15-step tutorial
│   ├── dashboard.go              Statistics dashboard
│   ├── hud.go / hud_cache.go     HUD rendering + caching
│   ├── minimap.go                Minimap
│   ├── algo_labor.go             Algo-Labor radar chart + tournament
│   ├── tournament.go             Tournament bracket
│   ├── livechart.go              Live fitness chart
│   ├── tooltips.go               Hover tooltips
│   ├── bot_tooltip.go            Bot detail tooltip
│   ├── achievements.go           Achievement popup
│   ├── leaderboard.go            Leaderboard
│   ├── genome_browser.go         Genome visualization
│   ├── neuro_viz.go              Neural network visualization
│   ├── pareto.go                 Pareto front visualization
│   ├── patterns.go               Pattern overlays
│   ├── formation.go              Formation display
│   ├── speciation.go             Species visualization
│   ├── capture.go                Screenshot/GIF recording
│   ├── particles.go              Particle effects
│   ├── replay.go                 Replay playback
│   ├── console.go                In-game log console
│   ├── welcome.go                Welcome screen
│   └── sound.go                  Audio system
├── locale/                     i18n system (7 languages, 1325+ keys)
│   ├── locale.go                 T(), Tf(), Tn(), CycleLang(), Save/LoadLang
│   ├── de.go, en.go, fr.go, es.go, pt.go, it.go, uk.go
│   └── locale_test.go            Consistency tests
├── cmd/
│   └── locale-check/           CLI tool for translation consistency
├── logger/                     Structured logging
├── main.go                     Game entry point (158 lines)
├── game_update.go              Update loop + global input + factory input
├── game_draw.go                Draw routing
├── input.go                    Classic mode input
├── swarm_input.go              Swarm mode input (editor, light, math trace)
├── swarm_input_alglab.go       Block editor click handling
├── swarm_input_editor.go       SwarmScript editor keyboard
├── block_editor.go             Block editor helpers
├── profiling.go                CPU profiling (build tag: profile)
├── benchmark_runner.go         Benchmark harness
└── Makefile                    Build targets
```

## Simulation Modes

SwarmSim has four distinct simulation modes, switched with function keys:

### F1: Classic Mode

The original simulation with 5 bot types, pheromone-based communication, and evolutionary learning.

- **Bot types**: Scout, Worker, Leader, Tank, Healer
- **Pheromones**: Diffusion/evaporation grid for indirect communication
- **Evolution**: Genetic programming with elite preservation
- **Scenarios**: Truck Unloading, Maze, Multiplayer (Blue vs Red)
- **SwarmScript**: DSL with 30+ sensors and 25+ actions

### F2: Swarm Lab

Interactive laboratory for SwarmScript programming with a code editor and 22 presets.

- **Code editor**: Syntax highlighting, line numbers, copy/paste
- **Block editor**: Visual drag-and-drop rule builder
- **22 presets**: Pre-configured SwarmScript programs
- **80+ subsystems**: Each follows the Init/Tick/Clear pattern
- **Tabbed panel**: Clickable toggle UI for all subsystems
- **Live Math Overlay**: Real-time formula visualization (K key)

### F4: Algo-Labor

Dedicated optimization algorithm comparison mode with 20 metaheuristic algorithms.

**Algorithms** (all with live math trace):
1. PSO (Particle Swarm Optimization)
2. ACO (Ant Colony Optimization)
3. Firefly Algorithm
4. GWO (Grey Wolf Optimizer)
5. WOA (Whale Optimization Algorithm)
6. BFO (Bacterial Foraging Optimization)
7. MFO (Moth-Flame Optimization)
8. Cuckoo Search
9. DE (Differential Evolution)
10. ABC (Artificial Bee Colony)
11. HSO (Harmony Search Optimization)
12. Bat Algorithm
13. SSA (Salp Swarm Algorithm)
14. GSA (Gravitational Search Algorithm)
15. FPA (Flower Pollination Algorithm)
16. HHO (Harris Hawks Optimization)
17. SA (Simulated Annealing)
18. AO (Aquila Optimizer)
19. SCA (Sine Cosine Algorithm)
20. DA (Dragonfly Algorithm)
21. TLBO (Teaching-Learning-Based Optimization)
22. EO (Equilibrium Optimizer)
23. Jaya Algorithm
24. Boids (flocking, no fitness)

**Features**:
- Radar chart comparing convergence, diversity, speed, stability
- Tournament mode: round-robin algorithm competition
- 7 fitness landscapes: Gaussian Peaks, Rastrigin, Ackley, Rosenbrock, Schwefel, Griewank, Levy
- Side-by-side algorithm comparison
- Algorithm-specific overlays

### F5: Factory Mode

Large-scale warehouse logistics simulation with 1000+ autonomous robots.

- **3 robot types**: Transporter (standard), Forklift (slow, high energy), Express (fast, low energy)
- **Production chain**: Trucks arrive with raw materials, machines process them, trucks depart with finished goods
- **Task queue**: Kanban-style assignment with priority, multi-step recipes
- **Energy economics**: Day/night pricing, charging stations, budget management
- **Shift system**: Handover between shifts, off-duty parking
- **Maintenance**: Malfunctions, repair scheduling, maintenance planner (P key)
- **Customer orders**: Deadlines, scoring, order fulfillment tracking
- **Random events**: 6 event types affecting production
- **Weather system**: Rain affects bot speed
- **Heatmap**: Bot traffic density visualization

## Core Concepts

### SwarmState -- The Central Hub

`SwarmState` in `domain/swarm/swarm_struct.go` is the central data structure for Swarm Lab and Algo-Labor modes. It holds:

- `Bots []SwarmBot` -- all bots in the simulation
- `Rng *rand.Rand` -- shared random number generator
- `Tick int` -- current simulation tick
- `ArenaW, ArenaH float64` -- arena dimensions
- A pointer + bool for every subsystem (80+)

### SwarmBot -- Per-Bot Data

`SwarmBot` holds all per-bot runtime data: position, heading, speed, carrying state, LED color, sensor readings, brain, and per-subsystem data.

### FactoryState -- Factory Hub

`FactoryState` in `domain/factory/factory_struct.go` is the central data structure for Factory Mode. It holds bot array, machine array, truck array, task queue, budget, shift state, weather, events, and all rendering state (camera, selections, effects).

### Algorithm Registry

`domain/swarm/algorithm_registry.go` provides a dispatch table that maps each `SwarmAlgorithmType` to an `algoHandler` struct containing lifecycle functions (Init, Clear, Tick, Apply) and optional query functions (bestFitness, avgFitnessVals, bestPos, explorationRatio) plus a `mathTrace` function for the Live Math Overlay.

Adding a new algorithm requires only:
1. Defining `AlgoXxx` in the enum (`swarm_algorithms.go`)
2. Creating Init/Clear/Tick/Apply functions in a dedicated file
3. Adding one entry to `algoRegistry`

No switch statement edits are needed.

## The Init / Tick / Clear Pattern

**Every swarm subsystem** follows the same three-function pattern:

### 1. `Init<Feature>(ss *SwarmState)`

Called once to set up the subsystem. Allocates per-bot arrays, sets default parameters, stores the state pointer in `SwarmState`.

### 2. `Tick<Feature>(ss *SwarmState)`

Called every simulation tick. Reads bot state, modifies behavior (speed, angle, LEDColor). Always starts with a nil-check on the state pointer.

### 3. `Clear<Feature>(ss *SwarmState)`

Disables the subsystem, frees memory by setting the state pointer to nil and the toggle bool to false.

### Optional: `Evolve<Feature>(ss *SwarmState, sortedIndices []int)`

For evolutionary subsystems. Called at generation boundaries. `sortedIndices` is fitness-sorted (best first). Top 20-25% become parents, bottom bots get replaced.

## Key Subsystems

### i18n (Internationalization)

- **7 languages**: DE, EN, FR, ES, PT, IT, UK (Ukrainian)
- **1325+ translation keys** organized by prefix (tab.*, toggle.*, btn.*, tooltip.*, help.*, tutorial.*, ach.*, ui.*, stat.*, bot.*, preset.*, plural.*)
- **Functions**: `T(key)` for simple lookup, `Tf(key, args...)` for formatted, `Tn(key, count)` for pluralized
- **Persistence**: `SaveLang()` / `LoadLang()` store preference to disk
- **Thread safety**: `sync.RWMutex` protects concurrent access
- **CLI tool**: `cmd/locale-check/` finds missing or untranslated keys

### Font Rendering (text/v2)

- **JetBrains Mono TTF** loaded from `render/fonts/`
- **text/v2 API**: Full Unicode support including Cyrillic (Ukrainian)
- **Text cache**: GPU image cache with `printColoredAt` function
- **512-entry LRU**: Prevents unbounded memory growth, 120-frame eviction for stale entries

### Math Trace System

Live visualization of algorithm calculations for the selected bot.

- **MathTrace / MathStep types**: Defined in `domain/swarm/math_trace.go`
- **4 step kinds**: Input (blue), Intermediate (yellow), Output (green), Branch (orange)
- **20 trace functions**: In `math_trace_algos.go`, one per algorithm
- **Rendering**: `render/swarm_render_math.go` draws the overlay panel
- **Activation**: K key toggles, only active when an algorithm is running

### Factory Simulation

The factory update loop (`engine/simulation/factory_ai.go`) runs 15 phases per tick:

0. Spatial hash rebuild (bot-bot collision)
0. Emergency system tick
0. Shift system tick
1. Truck arrivals/departures
2. Task generation from structures
2. Stale task pruning (every 100 ticks)
3. Task assignment to idle bots (50/tick max)
3. Parking slot pre-computation
4. Bot behavior execution (navigate, pickup, deliver)
5. Machine processing + completion FX
6. Energy and charging
6. Order system tick
7. Repair and malfunctions
7. Random events
8. Weather system
9. Heatmap accumulation (every 5 ticks)

## Performance Patterns

- **Spatial hash**: O(1) neighbor lookup for bot-bot and obstacle collision (domain/physics/)
- **FOV culling**: Factory renderer skips bots/structures outside camera view
- **Staggered updates**: A* pathfinding, malfunction checks, and heatmap accumulation run every N ticks
- **Text cache**: GPU image cache with 512-entry LRU cap and 120-frame eviction
- **Pheromone cache**: Pixel buffer recalculated every 5 ticks only
- **Bot sprites**: Pre-rendered 24x24px triangles with ColorScale tinting
- **Caching**: Font face, algorithm explanations, tutorial steps, parking slot assignments all cached
- **Pre-computed parking**: O(n) parking slot assignment replaces O(n^2) per-bot search
- **Task assignment cap**: Max 50 assignments per tick prevents frame drops

## Naming Conventions

- **State types**: `<Feature>State` (e.g., `OscillatorState`, `HomeostasisState`)
- **Functions**: `Init<Feature>`, `Tick<Feature>`, `Clear<Feature>`, `Evolve<Feature>`
- **SwarmState fields**: `<Feature> *<Feature>State` and `<Feature>On bool`
- **Per-bot types**: `Bot<Feature>` (e.g., `BotBody`, `BotDrives`, `BotVocab`)
- **Accessor functions**: `<Feature><Metric>(state, botIdx)`
- **Factory types**: Prefixed with `Factory` or short role names (`RoleTransporter`)
- **Algorithm types**: `Algo<Name>` enum values, `Init<Name>/Clear<Name>/Tick<Name>/Apply<Name>` functions

## Keyboard Shortcuts

### Global (All Modes)

| Key | Action |
|-----|--------|
| F1 | Switch to Classic Mode |
| F2 / F7 | Switch to Swarm Lab |
| F3 | Start interactive tutorial |
| F4 | Switch to Algo-Labor |
| F5 | Switch to Factory Mode |
| F10 | Screenshot |
| F11 | Toggle GIF recording |
| F12 | Toggle CPU profiling |
| Space | Pause / Resume |
| Q | Single-step mode (toggle / advance one tick) |
| +/- | Speed up / slow down |
| S | Toggle sound |
| H | Toggle help overlay |
| ` or ; | Toggle in-game log console |
| Escape | Dismiss overlay / welcome / tutorial |

### Classic Mode (F1)

| Key | Action |
|-----|--------|
| 1-5 | Spawn bot (Scout/Worker/Leader/Tank/Healer) at cursor |
| R | Spawn resource at cursor |
| O | Add obstacle at cursor |
| F | Toggle comm radius display |
| G | Toggle sensor radius display |
| D | Toggle debug comm lines |
| T | Toggle trail rendering |
| M | Toggle minimap |
| P | Cycle pheromone visualization (OFF/FOUND/ALL) |
| E | Force end generation (evolve) |
| V | Toggle genome overlay |
| N | Switch scenario (cycle) |
| Click | Select bot |
| WASD | Pan camera |

### Swarm Lab (F2)

| Key | Action |
|-----|--------|
| L | Toggle light source at cursor |
| K | Toggle Live Math Overlay |
| Click | Interact with tabbed panel, editor, block editor |
| Mouse wheel | Scroll editor / block editor |

### Factory Mode (F5)

| Key | Action |
|-----|--------|
| WASD / Arrows | Pan camera |
| Mouse wheel | Zoom in/out |
| Right-drag | Pan camera |
| Space | Pause / Resume |
| 1-5 | Speed presets (1x, 2x, 5x, 10x, 20x) |
| +/- | Fine speed adjust / bot count (add/remove 100) |
| T | Spawn inbound truck |
| Shift+T | Spawn outbound truck |
| H | Toggle help overlay / dismiss heatmap |
| M | Toggle heatmap |
| E | Toggle emergency mode |
| P | Toggle maintenance planner |
| B | Buy 10 bots ($500 each) |
| V | Sell 10 idle bots ($200 each) |
| X | Copy factory stats to clipboard |
| F | Toggle follow-cam on selected bot |
| Click | Select bot / Toggle machine |
| Click minimap | Jump camera to location |

## Common Pitfalls

1. **Name collisions**: Check existing files before naming types/functions. Use `grep -r "func YourName"` to verify uniqueness.

2. **Go integer truncation**: `int(-10.0 / 40.0) == 0` in Go (truncates toward zero). Use large negative values (-100) for out-of-bounds tests.

3. **`clampF` exists in `novelty.go`**: Do not redeclare it. Just use it.

4. **Always read before edit**: The `SwarmState` file is frequently modified. Always read it before editing to avoid stale content.

5. **O(n^2) neighbor loops**: Many features iterate all bot pairs. For large swarms (500+), consider using the spatial hash from `physics/`.

6. **Factory per-bot slices**: Factory mode maintains parallel slices (Bots, BotRoles, Malfunctioning, etc.). When adding/removing bots, all slices must be updated in sync.

7. **Thread safety**: Locale lookups use `sync.RWMutex`. Any new global state accessed from both update and render goroutines needs similar protection.

## Test Pattern

Every subsystem should have at minimum:

- `TestInit<Feature>` -- verifies initialization
- `TestClear<Feature>` -- verifies cleanup
- `TestTick<Feature>` -- verifies basic operation
- `TestTick<Feature>Nil` -- verifies nil-safety (no panic when not initialized)
- At least one domain-specific test

Run all tests:

```bash
go test ./domain/swarm/ -count=1 -timeout 120s
go test ./domain/factory/ -count=1 -timeout 30s
go test ./engine/simulation/ -count=1 -timeout 30s
go test ./locale/... -count=1 -timeout 30s
go test ./... -count=1 -timeout 180s  # everything
```
