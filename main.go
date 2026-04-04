package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"
	"swarmsim/locale"
	"swarmsim/logger"
	"swarmsim/render"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenW = 1280
	screenH = 900
)

// Game implements the ebiten.Game interface.
type Game struct {
	sim      *simulation.Simulation
	renderer *render.Renderer
	camera   *render.Camera

	scenarios []simulation.Scenario

	// Camera panning
	dragging   bool
	dragStartX int
	dragStartY int
	camStartX  float64
	camStartY  float64

	// Tick accumulator for fixed timestep
	tickAcc float64

	// Capture requests (set in Update, executed in Draw where screen is available)
	screenshotRequested bool
	gifToggleRequested  bool

	// Welcome screen
	showWelcome  bool
	welcomeTick  int
	welcomeReady bool // set after first frame (init bots needs screen size)

	// Help overlay
	showHelp    bool
	helpScrollY int

	// In-game console
	showConsole      bool
	consoleFilterBot int // -1 = all logs, >= 0 = filter for this bot

	// Classic Mode scenario dropdown
	classicDropdownOpen  bool
	classicDropdownHover int
	classicScenarioIdx   int // 0-4 index into classicScenarios
	classicScenarios     []simulation.Scenario

	// Panic recovery overlay
	panicMsg   string
	panicTimer int

	// Tutorial
	tutorial render.TutorialState

	// Tooltips
	tooltip render.TooltipState

	// Single-step debugger (Q key)
	stepMode bool // true = advance one tick per Q press
	stepOnce bool // true = execute exactly one tick this frame

	// Replay / time travel
	replayMode     bool
	replayIdx      int  // current snapshot index in replay buffer
	replayWasPause bool // was sim paused before entering replay?

	// Telemetry (enabled with --telemetry flag)
	telemetryWriter *swarm.TelemetryWriter
}

// NewGame creates a new game instance.
func NewGame() *Game {
	cfg := simulation.DefaultConfig()
	s := simulation.NewSimulation(cfg)
	cam := render.NewCamera(cfg.ArenaWidth, cfg.ArenaHeight)
	r := render.NewRenderer(cam)
	return &Game{
		sim:              s,
		renderer:         r,
		camera:           cam,
		scenarios:        simulation.GetScenarios(),
		classicScenarios: simulation.GetClassicScenarios(),
		showWelcome:      true,
		showConsole:      true,
		consoleFilterBot: -1,
	}
}

func main() {
	defer logger.CloseLog()
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error("CRASH", "Panic: %v\n%s", r, stack)
			fmt.Fprintf(os.Stderr, "FATAL: %v\n", r)
		}
	}()

	// Parse CLI flags
	telemetryEnabled := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--benchmark":
			runBenchmark()
			return
		case "--telemetry":
			telemetryEnabled = true
		}
	}

	logger.Info("INIT", "SwarmSim starting")

	locale.LoadLang()

	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle(locale.T("ui.window_title"))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)

	game := NewGame()

	// Enable telemetry if requested via --telemetry flag
	if telemetryEnabled {
		tw, err := swarm.NewTelemetryWriter("telemetry.jsonl", 10)
		if err != nil {
			logger.Error("TELEMETRY", "Cannot open telemetry.jsonl: %v", err)
		} else {
			logger.Info("TELEMETRY", "Writing telemetry to telemetry.jsonl")
			game.telemetryWriter = tw
		}
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

	// Close telemetry
	if game.telemetryWriter != nil {
		game.telemetryWriter.Close()
	}
	StopProfile() // ensure profiling stops on clean exit
	logger.Info("INIT", "SwarmSim exiting cleanly")
}
