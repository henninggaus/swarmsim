# Architecture Guide

This document explains SwarmSim's internal architecture and how to add new features.

## Overview

```
swarmsim/
├── domain/              Pure logic (no rendering dependencies)
│   ├── bot/             Bot types and behavior interfaces
│   ├── swarm/           SwarmBot + 80 subsystems (156 files)
│   ├── physics/         Collision detection, spatial hash, arena
│   ├── comm/            Decentralized message passing
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
│   ├── dashboard.go     Statistics dashboard
│   ├── tutorial.go      Interactive 15-step tutorial
│   └── ...              Minimap, tooltips, help, particles, capture
├── logger/              Structured logging
└── main.go              Ebiten game loop, input handling
```

## Core Concepts

### SwarmState — The Central Hub

`SwarmState` in `domain/swarm/swarm_struct.go` is the central data structure. It holds:

- `Bots []SwarmBot` — all bots in the simulation
- `Rng *rand.Rand` — shared random number generator
- `Tick int` — current simulation tick
- `ArenaW, ArenaH float64` — arena dimensions
- A pointer + bool for every subsystem (80+)

```go
type SwarmState struct {
    Bots    []SwarmBot
    Rng     *rand.Rand
    Tick    int
    ArenaW  float64
    ArenaH  float64

    // Each subsystem adds two fields:
    Oscillator   *OscillatorState   // subsystem state (nil = not initialized)
    OscillatorOn bool               // toggle flag
    // ... 80+ more subsystem pairs ...
}
```

### SwarmBot — Per-Bot Data

`SwarmBot` holds all per-bot runtime data:

```go
type SwarmBot struct {
    X, Y        float64     // position
    Angle       float64     // heading (radians)
    Speed       float64     // current speed
    CarryingPkg int         // package index (-1 = none)
    LEDColor    [3]uint8    // RGB LED color

    // Sensor readings (updated each tick)
    NearestPickupDist  float64
    NearestDropoffDist float64
    NeighborCount      int

    // Brain
    Brain    *NeuroBrain
    // ... more fields ...
}
```

## The Init / Tick / Clear Pattern

**Every subsystem** follows the same three-function pattern:

### 1. `Init<Feature>(ss *SwarmState)`

Called once to set up the subsystem. Allocates per-bot arrays, sets default parameters, stores the state pointer in `SwarmState`.

```go
func InitOscillators(ss *SwarmState) {
    n := len(ss.Bots)
    os := &OscillatorState{
        Phases:   make([]float64, n),
        NatFreqs: make([]float64, n),
        // ...
    }
    for i := 0; i < n; i++ {
        os.Phases[i] = ss.Rng.Float64() * 2 * math.Pi
    }
    ss.Oscillator = os
}
```

### 2. `Tick<Feature>(ss *SwarmState)`

Called every simulation tick. Reads bot state, modifies behavior (speed, angle, LEDColor). Always starts with a nil-check:

```go
func TickOscillators(ss *SwarmState) {
    os := ss.Oscillator
    if os == nil {
        return
    }
    // ... update phases, apply behavior ...
}
```

### 3. `Clear<Feature>(ss *SwarmState)`

Disables the subsystem, frees memory:

```go
func ClearOscillators(ss *SwarmState) {
    ss.Oscillator = nil
    ss.OscillatorOn = false
}
```

### Optional: `Evolve<Feature>(ss *SwarmState, sortedIndices []int)`

For evolutionary subsystems. Called at generation boundaries. `sortedIndices` is a fitness-sorted list of bot indices (best first). Top 20-25% become parents, bottom bots get replaced.

```go
func EvolveBodyPlans(ss *SwarmState, sortedIndices []int) {
    // Top 25% as parents
    parentCount := n * 25 / 100
    // Elite preservation (top 2 unchanged)
    // Crossover + mutation for the rest
}
```

## How to Add a New Subsystem

### Step 1: Create the feature file

Create `domain/swarm/my_feature.go`:

```go
package swarm

import "swarmsim/logger"

type MyFeatureState struct {
    // Per-bot data
    Values []float64
    // Parameters
    Rate float64
    // Stats
    AvgValue float64
}

func InitMyFeature(ss *SwarmState) {
    n := len(ss.Bots)
    mf := &MyFeatureState{
        Values: make([]float64, n),
        Rate:   0.05,
    }
    ss.MyFeature = mf
    logger.Info("MF", "Initialized: %d bots", n)
}

func ClearMyFeature(ss *SwarmState) {
    ss.MyFeature = nil
    ss.MyFeatureOn = false
}

func TickMyFeature(ss *SwarmState) {
    mf := ss.MyFeature
    if mf == nil {
        return
    }
    // ... logic ...
}
```

### Step 2: Add fields to SwarmState

In `domain/swarm/swarm_struct.go`, add before `// Block editor`:

```go
// My Feature
MyFeature   *MyFeatureState
MyFeatureOn bool
```

### Step 3: Create tests

Create `domain/swarm/my_feature_test.go`:

```go
package swarm

import (
    "math/rand"
    "testing"
)

func TestInitMyFeature(t *testing.T) {
    rng := rand.New(rand.NewSource(42))
    ss := NewSwarmState(rng, 10)
    InitMyFeature(ss)
    if ss.MyFeature == nil {
        t.Fatal("should be initialized")
    }
}

func TestTickMyFeatureNil(t *testing.T) {
    rng := rand.New(rand.NewSource(42))
    ss := NewSwarmState(rng, 5)
    TickMyFeature(ss) // should not panic
}
```

### Step 4: Run tests

```bash
go test ./domain/swarm/ -run "TestMyFeature" -count=1 -timeout 30s
```

## Naming Conventions

- **State types**: `<Feature>State` (e.g., `OscillatorState`, `HomeostasisState`)
- **Functions**: `Init<Feature>`, `Tick<Feature>`, `Clear<Feature>`, `Evolve<Feature>`
- **SwarmState fields**: `<Feature> *<Feature>State` and `<Feature>On bool`
- **Per-bot types**: `Bot<Feature>` (e.g., `BotBody`, `BotDrives`, `BotVocab`)
- **Accessor functions**: `<Feature><Metric>(state, botIdx)` (e.g., `BotPhase(os, 0)`)

## Common Pitfalls

1. **Name collisions**: Check existing files before naming types/functions. Use `grep -r "func YourName"` to verify uniqueness.

2. **Go integer truncation**: `int(-10.0 / 40.0) == 0` in Go (truncates toward zero). Use large negative values (-100) for out-of-bounds tests.

3. **`clampF` exists in `novelty.go`**: Don't redeclare it. Just use it.

4. **Always read before edit**: The `SwarmState` file is frequently modified. Always read it before editing to avoid stale content.

5. **O(n^2) neighbor loops**: Many features iterate all bot pairs. For large swarms (500+), consider using the spatial hash from `physics/`.

## Test Pattern

Every subsystem should have at minimum:

- `TestInit<Feature>` — verifies initialization
- `TestClear<Feature>` — verifies cleanup
- `TestTick<Feature>` — verifies basic operation
- `TestTick<Feature>Nil` — verifies nil-safety (no panic when not initialized)
- At least one domain-specific test (e.g., synchronization, decay, matching)

Run all tests:

```bash
go test ./domain/swarm/ -count=1 -timeout 120s
```

## Performance Notes

- **SpatialHash** — O(1) neighbor lookup in `physics/`
- **Text Cache** — HUD text rendered as cached GPU images (120-frame eviction)
- **Pheromone Cache** — Pixel buffer recalculated every 5 ticks only
- **Bot Sprites** — Pre-rendered 24x24px triangles with ColorScale tinting
- Features that tick every N ticks (not every tick) use `ss.Tick % N` for load distribution
