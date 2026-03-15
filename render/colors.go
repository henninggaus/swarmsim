package render

import "image/color"

// Bot type colors.
var (
	ColorScout  = color.RGBA{0, 255, 255, 255}   // Cyan
	ColorWorker = color.RGBA{255, 165, 0, 255}   // Orange
	ColorLeader = color.RGBA{255, 215, 0, 255}   // Gold
	ColorTank   = color.RGBA{0, 100, 0, 255}     // Dark green
	ColorHealer = color.RGBA{255, 105, 180, 255} // Pink

	ColorResource   = color.RGBA{0, 200, 0, 255}
	ColorObstacle   = color.RGBA{128, 128, 128, 255}
	ColorHomeBase   = color.RGBA{80, 120, 255, 200}
	ColorHealthBar  = color.RGBA{0, 255, 0, 255}
	ColorHealthBg   = color.RGBA{255, 0, 0, 255}
	ColorHUD        = color.RGBA{255, 255, 255, 255}
	ColorCommLine   = color.RGBA{255, 255, 0, 150}
	ColorSensorRad  = color.RGBA{255, 255, 255, 30}
	ColorCommRad    = color.RGBA{255, 255, 0, 20}
	ColorBackground = color.RGBA{20, 20, 30, 255}
	ColorGrid       = color.RGBA{40, 40, 55, 255}

	// Energy bar
	ColorEnergyBar = color.RGBA{255, 220, 0, 255}
	ColorEnergyBg  = color.RGBA{80, 60, 0, 255}

	// Pheromone colors
	ColorPherSearch = color.RGBA{50, 80, 255, 0} // blue
	ColorPherFound  = color.RGBA{0, 220, 50, 0}  // green
	ColorPherDanger = color.RGBA{255, 40, 30, 0} // red

	// Genome overlay
	ColorGenomeBar = color.RGBA{100, 200, 255, 220}
	ColorGenomeBg  = color.RGBA{0, 0, 0, 200}

	// Fitness graph
	ColorFitnessLine = color.RGBA{0, 255, 100, 255}
	ColorFitnessBg   = color.RGBA{0, 0, 0, 160}

	// Bot disabled (zero energy)
	ColorBotDisabled = color.RGBA{100, 100, 100, 180}

	// Heavy resource
	ColorHeavyResource = color.RGBA{255, 215, 0, 255} // Gold

	// Cooperative pickup particles
	ColorCoopParticle = color.RGBA{255, 200, 50, 255} // Gold sparkle

	// Home base delivery glow
	ColorDeliveryGlow = color.RGBA{100, 255, 100, 200} // Green glow

	// Truck mode colors
	ColorTruckCabin = color.RGBA{60, 60, 70, 255}
	ColorTruckCargo = color.RGBA{180, 160, 130, 255}
	ColorTruckRamp  = color.RGBA{140, 130, 110, 255}
	ColorRampEdge   = color.RGBA{100, 90, 80, 255}

	// Package colors
	ColorPkgSmallBox  = color.RGBA{160, 120, 80, 255}
	ColorPkgMediumBox = color.RGBA{120, 85, 50, 255}
	ColorPkgLargeBox  = color.RGBA{200, 150, 80, 255}
	ColorPkgFragile   = color.RGBA{220, 50, 50, 255}
	ColorPkgPallet    = color.RGBA{150, 150, 150, 255}
	ColorPkgLongItem  = color.RGBA{80, 120, 200, 255}

	// Sort zone colors (fill tints)
	ColorZoneA = color.RGBA{100, 150, 255, 60}
	ColorZoneB = color.RGBA{100, 255, 100, 60}
	ColorZoneC = color.RGBA{255, 180, 80, 60}
	ColorZoneD = color.RGBA{180, 100, 255, 60}

	// Sort zone borders
	ColorZoneABorder = color.RGBA{100, 150, 255, 200}
	ColorZoneBBorder = color.RGBA{100, 255, 100, 200}
	ColorZoneCBorder = color.RGBA{255, 180, 80, 200}
	ColorZoneDBorder = color.RGBA{180, 100, 255, 200}

	// Charging station
	ColorChargingStation = color.RGBA{255, 220, 50, 200}

	// Truck delivery particles
	ColorCorrectDelivery = color.RGBA{50, 255, 50, 255}
	ColorWrongDelivery   = color.RGBA{255, 50, 50, 255}

	// Swarm mode - editor
	ColorSwarmEditorBg  = color.RGBA{25, 25, 35, 255}
	ColorSwarmEditorSep = color.RGBA{60, 60, 80, 255}
	ColorSwarmLineNum   = color.RGBA{80, 80, 100, 255}
	ColorSwarmCursor    = color.RGBA{255, 255, 255, 200}

	// Swarm mode - syntax highlighting
	ColorSwarmKeyword   = color.RGBA{0, 255, 255, 255}   // IF/THEN/AND = cyan
	ColorSwarmCondition = color.RGBA{0, 255, 100, 255}   // sensor names = green
	ColorSwarmAction    = color.RGBA{255, 180, 50, 255}  // action names = orange
	ColorSwarmNumber    = color.RGBA{255, 255, 100, 255} // numbers = yellow
	ColorSwarmComment   = color.RGBA{100, 100, 100, 255} // comments = gray
	ColorSwarmOperator  = color.RGBA{200, 200, 200, 255} // operators = white

	// Swarm mode - UI buttons
	ColorSwarmBtnDeploy = color.RGBA{40, 140, 40, 255}  // deploy button
	ColorSwarmBtnReset  = color.RGBA{180, 120, 30, 255} // reset button
	ColorSwarmBtnPreset = color.RGBA{50, 80, 160, 255}  // preset dropdown
	ColorSwarmBtnHover  = color.RGBA{80, 120, 200, 255} // button hover
	ColorSwarmError     = color.RGBA{255, 60, 60, 255}  // error message

	// Swarm mode - arena
	ColorSwarmArenaBg     = color.RGBA{10, 10, 15, 255}
	ColorSwarmArenaGrid   = color.RGBA{25, 25, 35, 255}
	ColorSwarmArenaBorder = color.RGBA{60, 60, 80, 255}
	ColorSwarmLight       = color.RGBA{255, 255, 100, 80} // light glow
	ColorSwarmBotBlink    = color.RGBA{0, 255, 0, 200}    // deploy blink

	// Swarm mode - obstacles & maze
	ColorSwarmObstacle   = color.RGBA{100, 100, 110, 255} // obstacle body
	ColorSwarmObstacleHi = color.RGBA{130, 130, 140, 255} // obstacle highlight (top-left)
	ColorSwarmObstacleLo = color.RGBA{60, 60, 70, 255}    // obstacle shadow (bottom-right)
	ColorSwarmMazeWall   = color.RGBA{150, 150, 150, 255} // maze wall color
	ColorSwarmMazeBorder = color.RGBA{180, 180, 180, 255} // maze wall border stroke

	// Swarm mode - toggle buttons
	ColorSwarmBtnToggleOn  = color.RGBA{40, 140, 40, 255} // toggle on (green)
	ColorSwarmBtnToggleOff = color.RGBA{80, 80, 90, 255}  // toggle off (gray)

	// Swarm mode - follow lines & trails
	ColorSwarmTrail    = color.RGBA{255, 255, 255, 60} // trail dot
	ColorSwarmSelected = color.RGBA{255, 255, 0, 200}  // selected bot ring
	ColorSwarmInfoBg   = color.RGBA{20, 20, 30, 220}   // info panel bg
)
