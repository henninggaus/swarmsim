// Package render — Color scheme for SwarmSim.
//
// Design principles:
//   - Dark background (10-30 brightness) for long coding/observation sessions
//   - Bright saturated colors for interactive elements (bots, buttons, delivery stations)
//   - Muted/dim colors for passive elements (grid, borders, disabled UI)
//   - Alpha transparency for overlays, sensor radii, and pheromone layers
//   - 3D effects on obstacles via highlight (top-left) and shadow (bottom-right) edges
//   - Consistent color coding: green=health/success, red=error/danger, yellow=energy/warning
//   - Bot types use distinct hues for fast visual identification at any zoom level
package render

import "image/color"

// ═══════════════════════════════════════════════════════
// SHARED UI COLORS — used across many render files
// ═══════════════════════════════════════════════════════
var (
	ColorWhite       = color.RGBA{255, 255, 255, 255} // general-purpose white text
	ColorLightRed    = color.RGBA{255, 100, 100, 255} // errors, warnings, negative values
	ColorBrightBlue  = color.RGBA{100, 180, 255, 255} // panel headers, links
	ColorToggleBlue  = color.RGBA{80, 140, 220, 255}  // toggle/dashboard buttons
	ColorSectionGold = color.RGBA{255, 200, 80, 255}  // section headers, highlights
	ColorTextLight   = color.RGBA{200, 210, 230, 255} // secondary text, list items
	ColorHeaderBlue  = color.RGBA{180, 200, 255, 255} // section headers in algo labor
)

// ═══════════════════════════════════════════════════════
// STANDARD PANEL COLORS — consistent dark-panel styling
// ═══════════════════════════════════════════════════════
var (
	ColorPanelBg     = color.RGBA{10, 12, 22, 235}   // standard dark panel background
	ColorPanelBorder = color.RGBA{60, 80, 140, 200}   // standard panel border
	ColorPanelHeader = color.RGBA{255, 200, 100, 255}  // standard panel header text (gold)
)

// ═══════════════════════════════════════════════════════
// FREQUENTLY USED COLORS — deduplicated literals
// ═══════════════════════════════════════════════════════
var (
	ColorWhiteFaded = color.RGBA{255, 255, 255, 200}
	ColorGoldFaded  = color.RGBA{255, 215, 0, 200}
	ColorInfoCyan   = color.RGBA{136, 204, 255, 220}
	ColorDimOverlay = color.RGBA{60, 80, 120, 150}
	ColorMediumGray = color.RGBA{180, 180, 180, 255}
)

// ═══════════════════════════════════════════════════════
// CLASSIC MODE — Bot types & world elements
// ═══════════════════════════════════════════════════════
//
// Each bot type has a unique color for instant identification:
//   Scout=Cyan (fast explorer), Worker=Orange (resource gatherer),
//   Leader=Gold (coordination), Tank=DarkGreen (combat/defense),
//   Healer=Pink (support/repair)
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

	// ─── Status bars ───
	ColorEnergyBar = color.RGBA{255, 220, 0, 255} // yellow energy fill
	ColorEnergyBg  = color.RGBA{80, 60, 0, 255}   // dark energy background

	// ─── Pheromone visualization ───
	// Alpha starts at 0, dynamically adjusted by pheromone intensity
	ColorPherSearch = color.RGBA{50, 80, 255, 0} // blue = exploring/searching
	ColorPherFound  = color.RGBA{0, 220, 50, 0}  // green = resource found
	ColorPherDanger = color.RGBA{255, 40, 30, 0} // red = danger zone

	// ─── Genome & Evolution overlays ───
	ColorGenomeBar = color.RGBA{100, 200, 255, 220} // genome parameter bar fill
	ColorGenomeBg  = color.RGBA{0, 0, 0, 200}       // genome panel background

	// ─── Fitness tracking ───
	ColorFitnessLine = color.RGBA{0, 255, 100, 255} // fitness graph line (green)
	ColorFitnessBg   = color.RGBA{0, 0, 0, 160}     // fitness graph background

	// ─── Special states ───
	ColorBotDisabled = color.RGBA{100, 100, 100, 180} // grayed out (zero energy)

	// ─── Resource types ───
	ColorHeavyResource = color.RGBA{255, 215, 0, 255} // gold = heavy (needs 2 bots)
	ColorCoopParticle  = color.RGBA{255, 200, 50, 255} // sparkle during cooperative pickup
	ColorDeliveryGlow  = color.RGBA{100, 255, 100, 200} // green glow at home base on delivery

	// ═══════════════════════════════════════════════════════
	// TRUCK MODE — Logistics simulation colors
	// ═══════════════════════════════════════════════════════
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

	// ─── Sort zone colors ─── (4 zones A-D for package sorting)
	// Fill tints (low alpha for transparent zone overlays)
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

	// ═══════════════════════════════════════════════════════
	// SWARM LAB MODE — Editor & Arena colors
	// ═══════════════════════════════════════════════════════

	// ─── Editor panel ───
	ColorSwarmEditorBg  = color.RGBA{25, 25, 35, 255}
	ColorSwarmEditorSep = color.RGBA{60, 60, 80, 255}
	ColorSwarmLineNum   = color.RGBA{80, 80, 100, 255}
	ColorSwarmCursor    = color.RGBA{255, 255, 255, 200}

	// ─── Syntax highlighting (SwarmScript) ───
	ColorSwarmKeyword   = color.RGBA{0, 255, 255, 255}   // IF/THEN/AND = cyan
	ColorSwarmCondition = color.RGBA{0, 255, 100, 255}   // sensor names = green
	ColorSwarmAction    = color.RGBA{255, 180, 50, 255}  // action names = orange
	ColorSwarmNumber    = color.RGBA{255, 255, 100, 255} // numbers = yellow
	ColorSwarmComment   = color.RGBA{100, 100, 100, 255} // comments = gray
	ColorSwarmOperator  = color.RGBA{200, 200, 200, 255} // operators = white

	// ─── UI buttons ───
	ColorSwarmBtnDeploy = color.RGBA{40, 140, 40, 255}  // deploy button
	ColorSwarmBtnReset  = color.RGBA{180, 120, 30, 255} // reset button
	ColorSwarmBtnPreset = color.RGBA{50, 80, 160, 255}  // preset dropdown
	ColorSwarmBtnHover  = color.RGBA{80, 120, 200, 255} // button hover
	ColorSwarmError     = color.RGBA{255, 60, 60, 255}  // error message

	// ─── Arena rendering ───
	ColorSwarmArenaBg     = color.RGBA{10, 10, 15, 255}
	ColorSwarmArenaGrid   = color.RGBA{25, 25, 35, 255}
	ColorSwarmArenaBorder = color.RGBA{60, 60, 80, 255}
	ColorSwarmLight       = color.RGBA{255, 255, 100, 80} // light glow
	ColorSwarmBotBlink    = color.RGBA{0, 255, 0, 200}    // deploy blink

	// ─── Obstacles & maze ─── (3D effect: highlight=top-left, shadow=bottom-right)
	ColorSwarmObstacle   = color.RGBA{100, 100, 110, 255} // obstacle body
	ColorSwarmObstacleHi = color.RGBA{130, 130, 140, 255} // obstacle highlight (top-left)
	ColorSwarmObstacleLo = color.RGBA{60, 60, 70, 255}    // obstacle shadow (bottom-right)
	ColorSwarmMazeWall   = color.RGBA{150, 150, 150, 255} // maze wall color
	ColorSwarmMazeBorder = color.RGBA{180, 180, 180, 255} // maze wall border stroke

	// ─── Toggle buttons ───
	ColorSwarmBtnToggleOn  = color.RGBA{40, 140, 40, 255} // toggle on (green)
	ColorSwarmBtnToggleOff = color.RGBA{80, 80, 90, 255}  // toggle off (gray)

	// ─── Follow camera, trails & selection ───
	ColorSwarmTrail    = color.RGBA{255, 255, 255, 60} // trail dot
	ColorSwarmSelected = color.RGBA{255, 255, 0, 200}  // selected bot ring
	ColorSwarmInfoBg   = color.RGBA{20, 20, 30, 220}   // info panel bg
)
