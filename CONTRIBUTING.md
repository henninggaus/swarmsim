# Contributing to SwarmSim

This guide covers how to add features, algorithms, translations, and tests to SwarmSim.

## Getting Started

```bash
# Build
go build ./...

# Run all tests
go test ./... -count=1 -timeout 180s

# Run specific package tests
go test ./domain/swarm/ -count=1 -timeout 120s
go test ./domain/factory/ -count=1 -timeout 30s
go test ./locale/... -count=1 -timeout 30s

# Run the simulation
go run .
```

## File Organization

The codebase separates logic from rendering:

| Directory | Purpose | Dependencies |
|-----------|---------|-------------|
| `domain/swarm/` | Swarm logic, algorithms, subsystems | Pure Go, math, math/rand |
| `domain/factory/` | Factory mode logic | Pure Go, math, math/rand |
| `domain/bot/`, `domain/physics/`, `domain/genetics/` | Core domain types | Pure Go |
| `engine/simulation/` | Update loop, scenario loading | domain/* |
| `engine/swarmscript/` | DSL parser and interpreter | Pure Go |
| `engine/pheromone/` | Pheromone grid | Pure Go |
| `render/` | All rendering (Ebiten) | domain/*, engine/*, Ebiten |
| `locale/` | Translations | Pure Go |
| Root `.go` files | Game struct, input handling, draw routing | Everything |

**Rules**:
- `domain/` packages must never import `render/` or Ebiten
- `render/` reads domain state but does not modify simulation logic
- New rendering goes in `render/`, new logic goes in `domain/` or `engine/`
- Root-level files handle input dispatch and the Ebiten game interface

## Adding a Swarm Subsystem

Every subsystem follows the Init/Tick/Clear pattern. See ARCHITECTURE.md for details.

### Step 1: Create the feature file

Create `domain/swarm/my_feature.go` with `InitMyFeature`, `TickMyFeature`, and `ClearMyFeature` functions.

### Step 2: Add fields to SwarmState

In `domain/swarm/swarm_struct.go`, add:

```go
MyFeature   *MyFeatureState
MyFeatureOn bool
```

### Step 3: Create tests

Create `domain/swarm/my_feature_test.go` with at minimum:
- `TestInitMyFeature`
- `TestClearMyFeature`
- `TestTickMyFeature`
- `TestTickMyFeatureNil` (nil-safety)

### Step 4: Run tests

```bash
go test ./domain/swarm/ -run "TestMyFeature" -count=1 -timeout 30s
```

## Adding an Optimization Algorithm

SwarmSim uses a central algorithm registry. Adding a new algorithm is a three-step process.

### Step 1: Define the enum value

In `domain/swarm/swarm_algorithms.go`, add a new constant before `AlgoCount`:

```go
AlgoMyAlgo  // My Algorithm (Author Year)
```

### Step 2: Create the algorithm file

Create `domain/swarm/my_algo.go` with four functions:

```go
package swarm

type MyAlgoState struct {
    Fitness    []float64
    BestF      float64
    BestX, BestY float64
    BestIdx    int
    CycleTick  int
    GlobalBestF float64
    GlobalBestX, GlobalBestY float64
    // ... algorithm-specific state
}

func InitMyAlgo(ss *SwarmState) {
    n := len(ss.Bots)
    st := &MyAlgoState{
        Fitness: make([]float64, n),
        BestIdx: -1,
    }
    // Initialize per-bot state
    ss.MyAlgo = st
}

func ClearMyAlgo(ss *SwarmState) {
    ss.MyAlgo = nil
}

func TickMyAlgo(ss *SwarmState) {
    st := ss.MyAlgo
    if st == nil { return }
    // Evaluate fitness, update global best, advance cycle tick
}

func ApplyMyAlgo(bot *SwarmBot, ss *SwarmState, idx int) {
    st := ss.MyAlgo
    if st == nil { return }
    // Compute new position/angle for this bot
}
```

**Important patterns used by existing algorithms**:
- Use `WrapAngle()` from `swarm_steering.go` for angle normalization
- Use `steerToward()` for smooth heading changes
- Use `gridPt` and `idxFit` types from `algo_common.go`
- Use `AlgoGridRescanSize` and `AlgoGridInjectTop` constants for periodic grid rescans
- Evaluate fitness using `ss.SwarmAlgo.FitFunc` or the fitness landscape

### Step 3: Register in the algorithm registry

In `domain/swarm/algorithm_registry.go`, add an entry to `algoRegistry`:

```go
AlgoMyAlgo: {
    init:      InitMyAlgo,
    clear:     ClearMyAlgo,
    tick:      TickMyAlgo,
    apply:     ApplyMyAlgo,
    mathTrace: traceMyAlgo,  // optional, see Math Trace section
    bestFitness: func(ss *SwarmState) float64 {
        if ss.MyAlgo != nil { return ss.MyAlgo.BestF }
        return 0
    },
    avgFitnessVals: func(ss *SwarmState) []float64 {
        if ss.MyAlgo != nil { return ss.MyAlgo.Fitness }
        return nil
    },
    bestPos: func(ss *SwarmState) (float64, float64, bool) {
        if ss.MyAlgo != nil && ss.MyAlgo.BestIdx >= 0 {
            return ss.MyAlgo.BestX, ss.MyAlgo.BestY, true
        }
        return 0, 0, false
    },
},
```

You also need to add the `MyAlgo *MyAlgoState` field to `SwarmState` in `swarm_struct.go`.

## Math Trace System

The Math Trace provides live formula visualization for optimization algorithms.

### How it works

1. Each algorithm can register a `mathTrace` function in `algoRegistry`
2. When the user presses K, the trace function is called each tick for the selected bot
3. The trace function builds a `MathTrace` with labeled steps
4. `render/swarm_render_math.go` draws the overlay

### Adding a trace function

Add to `domain/swarm/math_trace_algos.go`:

```go
func traceMyAlgo(bot *SwarmBot, ss *SwarmState, idx int) {
    st := ss.MyAlgo
    if st == nil { return }

    mt := &MathTrace{AlgoName: "My Algorithm", PhaseName: "Exploration"}

    // Input parameters (blue)
    mt.AddStep("param", "alpha", fmt.Sprintf("%.3f", alpha), alpha, MathInput)

    // Intermediate calculations (yellow)
    mt.AddStep("step1", "x_new = x + alpha * r",
        fmt.Sprintf("%.1f + %.3f * %.3f", bot.X, alpha, r),
        newX, MathIntermediate)

    // Branch decisions (orange)
    mt.AddStep("phase", "exploration if E > 0.5",
        fmt.Sprintf("%.2f > 0.5 = %v", e, e > 0.5),
        0, MathBranch)

    // Final outputs (green)
    mt.AddStep("dx", "delta_x", fmt.Sprintf("%.2f", dx), dx, MathOutput)

    bot.MathTrace = mt
}
```

**Step kinds and their colors**:
- `MathInput` (blue): Input parameters and constants
- `MathIntermediate` (yellow): Intermediate computed values
- `MathOutput` (green): Final outputs (position deltas, new angles)
- `MathBranch` (orange): Phase or branch decisions

Then reference `traceMyAlgo` in the `mathTrace` field of your `algoRegistry` entry.

## Factory Mode Development

Factory mode files are in `domain/factory/`. The update loop is in `engine/simulation/factory_ai.go`.

### Adding a new structure type

1. Define the structure in `factory_struct.go` (position, size, properties)
2. Place it in `factory_layout.go`
3. Add rendering in `render/factory_render.go`

### Adding a new task type

1. Add a constant to the task type enum in `factory_struct.go`
2. Update `GenerateTasks()` in `factory_task.go` to generate the new task type
3. Update bot behavior in `factory_bot.go` to handle the new task
4. Add the new task type to the Kanban display in `render/factory_render_ui.go`

### Adding a new event type

1. Add the event to the event enum in `factory_struct.go`
2. Add generation logic in `factory_task.go` (TickEvents function)
3. Add alert text to locale files
4. Add rendering effects if needed in `render/factory_render_effects.go`

### Factory update phases

The factory runs 15 phases per tick (see `engine/simulation/factory_ai.go`):
- Phases 0-0.6: Spatial hash, emergency, shift system
- Phases 1-3.5: Trucks, task generation, task assignment, parking
- Phases 4-6.5: Bot behavior, machines, energy, orders
- Phases 7-9: Repair, events, weather, heatmap

When adding new phases, place them in the appropriate position and consider whether they should be paused during emergencies.

## Locale / i18n System

SwarmSim supports 7 languages: DE, EN, FR, ES, PT, IT, UK (Ukrainian).

### Architecture

- `locale/locale.go` -- Core: `T()`, `Tf()`, `Tn()`, `CycleLang()`, `SaveLang()`/`LoadLang()`
- `locale/{lang}.go` -- One file per language with all translations
- `cmd/locale-check/` -- CLI tool to find missing/untranslated keys

### Adding a New Language

1. Create `locale/xx.go` with `var xxStrings = map[string]string{...}`
2. Copy all keys from `locale/en.go` and translate the values
3. In `locale/locale.go`:
   - Add `XX Lang = "xx"` constant
   - Add `XX` to `langOrder` slice
   - Add `XX: xxStrings` to `translations` map
4. Run `go run ./cmd/locale-check/` to verify all keys are present
5. Run `go test ./locale/...` to verify consistency

### Adding a New Translatable String

1. Add the key+value to `locale/de.go` (reference language)
2. Add the key+value to `locale/en.go`
3. Run `go run ./cmd/locale-check/ -scaffold` to get stubs for other languages
4. Copy the stubs into each language file and translate
5. Run `go test ./locale/...` -- `TestAllLanguagesHaveSameKeys` will catch missing keys

### Key Naming Convention

| Prefix | Usage |
|--------|-------|
| `tab.*` | Tab names |
| `toggle.*` | Toggle button labels |
| `btn.*` | Action button labels |
| `tooltip.*` | Hover tooltip texts |
| `help.*` | Help overlay content |
| `tutorial.*` | Tutorial step text |
| `ach.name.*` / `ach.desc.*` | Achievement names/descriptions |
| `ui.*` | General UI labels |
| `stat.*` | Statistics labels |
| `bot.*` | Bot info labels |
| `preset.*` | Preset names/descriptions |
| `plural.*.one` / `plural.*.other` | Pluralized strings (use `Tn()`) |
| `factory.*` | Factory mode strings |
| `algo.*` | Algorithm names and descriptions |

### Functions

- `locale.T("key")` -- Simple lookup
- `locale.Tf("key", args...)` -- Lookup + fmt.Sprintf
- `locale.Tn("key", count)` -- Pluralized lookup (uses key.one / key.other)

### Tools

```bash
go run ./cmd/locale-check/           # Check all languages for missing keys
go run ./cmd/locale-check/ -scaffold # Print Go stubs for missing keys
go test ./locale/... -v              # Run all locale tests
go test ./locale/... -bench=.        # Run benchmarks
```

## Testing

### Running Tests

```bash
# All tests
go test ./... -count=1 -timeout 180s

# Specific packages
go test ./domain/swarm/ -count=1 -timeout 120s
go test ./domain/factory/ -count=1 -timeout 30s
go test ./engine/simulation/ -count=1 -timeout 30s
go test ./engine/pheromone/ -count=1 -timeout 30s
go test ./locale/... -count=1 -timeout 30s

# Specific test
go test ./domain/swarm/ -run "TestMyFeature" -count=1 -timeout 30s

# With coverage
go test ./domain/swarm/ -coverprofile=cover.out
go tool cover -html=cover.out
```

### Test Requirements

Every subsystem should have:
- **Init test**: Verify state is allocated correctly
- **Clear test**: Verify cleanup sets pointer to nil
- **Tick test**: Verify basic operation produces expected changes
- **Nil-safety test**: Verify `Tick<Feature>` does not panic when state is nil
- **Domain-specific tests**: At least one test for the core algorithm behavior

For algorithms, additionally test:
- Fitness evaluation produces non-zero values
- Global best tracking updates correctly
- Cycle reset does not lose the global best

### Build Verification

Always verify the build compiles after changes:

```bash
go build ./...
```

## Code Style

- Use `sort.Slice` instead of bubble sort
- Use `WrapAngle()` instead of manual angle normalization
- Use the centralized color constants in `render/colors.go`
- Use `locale.T()` for all user-visible strings
- Prefer `printColoredAt()` from `render/text_render.go` for text rendering
- Use `cachedTextImage()` for frequently rendered text
- Keep `domain/` packages free of rendering dependencies
