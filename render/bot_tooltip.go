package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawBotTooltip renders a mini info panel near the hovered bot.
func DrawBotTooltip(screen *ebiten.Image, ss *swarm.SwarmState, mx, my int) {
	if ss.HoveredBot < 0 || ss.HoveredBot >= len(ss.Bots) {
		return
	}
	bot := &ss.Bots[ss.HoveredBot]

	// Build info lines
	lines := []string{
		fmt.Sprintf("Bot #%d", ss.HoveredBot),
		fmt.Sprintf("Pos: (%.0f, %.0f)", bot.X, bot.Y),
		fmt.Sprintf("Winkel: %.0f°  Speed: %.1f", bot.Angle*180/math.Pi, bot.Speed),
		fmt.Sprintf("State: %d  Counter: %d", bot.State, bot.Counter),
		fmt.Sprintf("LED: (%d,%d,%d)", bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2]),
		fmt.Sprintf("Nachbarn: %d  Naechster: %.0fpx", bot.NeighborCount, bot.NearestDist),
	}

	// Fitness/age
	lines = append(lines, fmt.Sprintf("Fitness: %.0f  Alter: %d", bot.Fitness, bot.Stats.TicksAlive))

	// Delivery info
	if ss.DeliveryOn {
		carryStr := "Nein"
		if bot.CarryingPkg >= 0 {
			carryStr = fmt.Sprintf("Paket #%d", bot.CarryingPkg)
		}
		lines = append(lines, fmt.Sprintf("Traegt: %s", carryStr))
		lines = append(lines, fmt.Sprintf("Lief: %d (%d richtig, %d falsch)",
			bot.Stats.TotalDeliveries, bot.Stats.CorrectDeliveries, bot.Stats.WrongDeliveries))
	}

	// Energy (only show when energy system is active)
	if ss.EnergyEnabled {
		lines = append(lines, fmt.Sprintf("Energie: %.0f%%", bot.Energy))
	}

	// Sensors
	sensorLine := fmt.Sprintf("Licht:%d  Msg:%d  Obs:%v",
		bot.LightValue, bot.ReceivedMsg, bot.ObstacleAhead)
	lines = append(lines, sensorLine)

	// Mode-specific info
	if ss.NeuroEnabled && bot.Brain != nil {
		lines = append(lines, "Modus: NEURO")
	} else if ss.GPEnabled && bot.OwnProgram != nil {
		lines = append(lines, fmt.Sprintf("Modus: GP (%d Regeln)", len(bot.OwnProgram.Rules)))
	} else if ss.EvolutionOn {
		lines = append(lines, "Modus: EVOLUTION")
	}

	// Calculate panel dimensions
	maxW := 0
	for _, l := range lines {
		if len(l) > maxW {
			maxW = len(l)
		}
	}
	panelW := maxW*charW + 16
	panelH := len(lines)*lineH + 10

	// Position tooltip near mouse, but clamp to screen
	tx := mx + 16
	ty := my - panelH - 8
	if tx+panelW > 1270 {
		tx = mx - panelW - 8
	}
	if ty < 5 {
		ty = my + 20
	}

	// Draw background
	vector.DrawFilledRect(screen, float32(tx-2), float32(ty-2),
		float32(panelW+4), float32(panelH+4), color.RGBA{10, 15, 30, 230}, false)
	vector.StrokeRect(screen, float32(tx-2), float32(ty-2),
		float32(panelW+4), float32(panelH+4), 1, color.RGBA{80, 140, 220, 200}, false)

	// LED color preview bar
	vector.DrawFilledRect(screen, float32(tx-2), float32(ty-2), float32(panelW+4), 3,
		color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 255}, false)

	// Draw lines
	for i, line := range lines {
		col := color.RGBA{200, 210, 230, 255}
		if i == 0 {
			col = color.RGBA{100, 180, 255, 255} // header color
		}
		printColoredAt(screen, line, tx+6, ty+4+i*lineH, col)
	}
}
