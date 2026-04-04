package render

import (
	"fmt"
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/engine/swarmscript"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawSelectedBotInfo draws an enhanced info panel for the selected bot.
func drawSelectedBotInfo(screen *ebiten.Image, ss *swarm.SwarmState) {
	bot := &ss.Bots[ss.SelectedBot]
	x := 1050
	y := 60
	w := 220
	h := 380
	if ss.GPEnabled && bot.OwnProgram != nil {
		h = 520 // taller to fit GP program info
	}
	if ss.NeuroEnabled && bot.Brain != nil {
		h = 500 // taller to fit neuro info
	}
	if ss.CollectiveAIOn && ss.SelectedBot < len(ss.BotChatLog) && ss.BotChatLog[ss.SelectedBot] != nil && len(ss.BotChatLog[ss.SelectedBot]) > 0 {
		h += 120 // extra space for chat log
	}
	valCol := color.RGBA{200, 200, 220, 255}
	dimCol := color.RGBA{140, 140, 160, 255}
	headerCol := color.RGBA{0, 220, 255, 255}

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), ColorSwarmInfoBg, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, ColorSwarmEditorSep, false)

	lx := x + 5
	ly := y + 5

	// Title + LED swatch
	printColoredAt(screen, fmt.Sprintf("Bot #%d", ss.SelectedBot), lx, ly, ColorSwarmSelected)
	ledCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 255}
	vector.DrawFilledRect(screen, float32(lx+70), float32(ly+2), 10, 10, ledCol, false)
	vector.StrokeRect(screen, float32(lx+70), float32(ly+2), 10, 10, 1, color.RGBA{255, 255, 255, 120}, false)
	ly += lineH + 2

	// --- Position & Bewegung ---
	printColoredAt(screen, locale.T("ui.position_movement"), lx, ly, headerCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("X:%.0f Y:%.0f", bot.X, bot.Y), lx, ly, valCol)
	ly += lineH
	degAngle := bot.Angle * 180 / math.Pi
	if degAngle < 0 {
		degAngle += 360
	}
	printColoredAt(screen, locale.Tf("ui.direction_speed", degAngle, bot.Speed), lx, ly, valCol)
	ly += lineH + 2

	// --- Interner Zustand ---
	printColoredAt(screen, locale.T("ui.internal_state"), lx, ly, headerCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("State:%d  Counter:%d  Timer:%d", bot.State, bot.Counter, bot.Timer), lx, ly, valCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Value1:%d  Value2:%d", bot.Value1, bot.Value2), lx, ly, valCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("LED: R%d G%d B%d", bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2]), lx, ly, dimCol)
	ly += lineH + 2

	// --- Sensoren ---
	printColoredAt(screen, locale.T("ui.sensors_live"), lx, ly, headerCol)
	ly += lineH
	nearStr := locale.T("ui.none")
	if bot.NearestIdx >= 0 {
		nearStr = fmt.Sprintf("%.0fpx (Bot #%d)", bot.NearestDist, bot.NearestIdx)
	}
	printColoredAt(screen, locale.Tf("ui.nearest", nearStr), lx, ly, valCol)
	ly += lineH
	printColoredAt(screen, locale.Tf("ui.neighbors_count", bot.NeighborCount), lx, ly, valCol)
	ly += lineH
	obsStr := locale.T("ui.no")
	if bot.ObstacleAhead {
		obsStr = locale.Tf("ui.yes_dist", bot.ObstacleDist)
	}
	edgeStr := locale.T("ui.no")
	if bot.OnEdge {
		edgeStr = locale.T("ui.yes")
	}
	printColoredAt(screen, locale.Tf("ui.obstacle_edge", obsStr, edgeStr), lx, ly, valCol)
	ly += lineH
	lightStr := "---"
	if ss.Light.Active {
		lightStr = fmt.Sprintf("%d/100", bot.LightValue)
	}
	msgStr := locale.T("ui.no")
	if bot.ReceivedMsg > 0 {
		msgStr = locale.Tf("ui.msg_type", bot.ReceivedMsg)
	}
	printColoredAt(screen, locale.Tf("ui.light_msg", lightStr, msgStr), lx, ly, valCol)
	ly += lineH + 2

	// --- Delivery (conditional) ---
	if ss.DeliveryOn {
		printColoredAt(screen, locale.T("ui.packet_delivery"), lx, ly, headerCol)
		ly += lineH
		carryStr := locale.T("ui.empty_searching")
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			carryStr = locale.Tf("ui.carrying_color", swarm.DeliveryColorName(ss.Packages[bot.CarryingPkg].Color))
		}
		printColoredAt(screen, carryStr, lx, ly, valCol)
		ly += lineH
		pDist := "---"
		if bot.NearestPickupDist < 999 {
			pDist = fmt.Sprintf("%.0fpx", bot.NearestPickupDist)
		}
		dDist := "---"
		if bot.NearestDropoffDist < 999 {
			dDist = fmt.Sprintf("%.0fpx", bot.NearestDropoffDist)
		}
		matchStr := ""
		if bot.DropoffMatch {
			matchStr = " MATCH!"
		}
		printColoredAt(screen, fmt.Sprintf("Pickup:%s Dropoff:%s%s", pDist, dDist, matchStr), lx, ly, valCol)
		ly += lineH + 2
	}

	// --- Sozial ---
	printColoredAt(screen, locale.T("ui.social_chains"), lx, ly, headerCol)
	ly += lineH
	followStr := "None"
	if bot.FollowTargetIdx >= 0 {
		followStr = fmt.Sprintf("#%d", bot.FollowTargetIdx)
	}
	followerStr := "None"
	if bot.FollowerIdx >= 0 {
		followerStr = fmt.Sprintf("#%d", bot.FollowerIdx)
	}
	printColoredAt(screen, fmt.Sprintf("Follow:%s Follower:%s", followStr, followerStr), lx, ly, valCol)
	ly += lineH + 2

	// --- Lifetime Stats ---
	printColoredAt(screen, "Lifetime", lx, ly, headerCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Dist:%.0f Alive:%d", bot.Stats.TotalDistance, bot.Stats.TicksAlive), lx, ly, dimCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Pickups:%d Deliv:%d", bot.Stats.TotalPickups, bot.Stats.TotalDeliveries), lx, ly, dimCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("OK:%d Wrong:%d", bot.Stats.CorrectDeliveries, bot.Stats.WrongDeliveries), lx, ly, dimCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Msg TX:%d RX:%d", bot.Stats.MessagesSent, bot.Stats.MessagesReceived), lx, ly, dimCol)
	ly += lineH
	printColoredAt(screen, fmt.Sprintf("Stuck:%d Idle:%d", bot.Stats.AntiStuckCount, bot.Stats.TicksIdle), lx, ly, dimCol)
	ly += lineH + 2

	// --- GP Program (when GP enabled) ---
	if ss.GPEnabled && bot.OwnProgram != nil {
		gpCol := color.RGBA{0, 180, 160, 255}
		printColoredAt(screen, "GP Program", lx, ly, gpCol)
		ly += lineH
		fit := EvaluateGPFitnessRender(bot)
		printColoredAt(screen, fmt.Sprintf("%d rules gen:%d fit:%.0f",
			len(bot.OwnProgram.Rules), ss.GPGeneration, fit), lx, ly, dimCol)
		ly += lineH
		// Show first 5 rules, highlight matched ones
		maxShow := 5
		if maxShow > len(bot.OwnProgram.Rules) {
			maxShow = len(bot.OwnProgram.Rules)
		}
		for ri := 0; ri < maxShow; ri++ {
			ruleCol := color.RGBA{120, 120, 140, 200}
			// Check if this rule matched last tick
			for _, mi := range bot.LastMatchedRules {
				if mi == ri {
					ruleCol = color.RGBA{80, 255, 80, 255} // green = matched
					break
				}
			}
			rule := &bot.OwnProgram.Rules[ri]
			ruleText := swarmscript.RuleToShortText(rule)
			if len(ruleText) > 35 {
				ruleText = ruleText[:35]
			}
			printColoredAt(screen, ruleText, lx, ly, ruleCol)
			ly += lineH - 2
		}
		if len(bot.OwnProgram.Rules) > maxShow {
			printColoredAt(screen, fmt.Sprintf("  ...+%d more", len(bot.OwnProgram.Rules)-maxShow), lx, ly, dimCol)
		}
	}

	// --- Neuro Brain (when NEURO enabled) ---
	if ss.NeuroEnabled && bot.Brain != nil {
		neuroCol := color.RGBA{255, 140, 50, 255}
		printColoredAt(screen, locale.T("ui.neural_net"), lx, ly, neuroCol)
		ly += lineH
		fit := EvaluateGPFitnessRender(bot)
		printColoredAt(screen, fmt.Sprintf("Gen:%d  Fitness:%.0f", ss.NeuroGeneration, fit), lx, ly, dimCol)
		ly += lineH

		// Show chosen action
		actionName := "---"
		if bot.Brain.ActionIdx >= 0 && bot.Brain.ActionIdx < len(swarm.NeuroActionNames) {
			actionName = swarm.NeuroActionNames[bot.Brain.ActionIdx]
		}
		printColoredAt(screen, locale.Tf("ui.action_name", actionName), lx, ly, color.RGBA{255, 255, 100, 255})
		ly += lineH

		// Show top 3 input values
		printColoredAt(screen, "Inputs:", lx, ly, color.RGBA{100, 200, 255, 200})
		ly += lineH
		for inp := 0; inp < swarm.NeuroInputs && inp < 6; inp++ {
			v := bot.Brain.InputVals[inp]
			printColoredAt(screen, fmt.Sprintf(" %s: %.2f", swarm.NeuroInputNames[inp], v), lx, ly, dimCol)
			ly += lineH - 3
		}
		ly += 3

		// Show output activations
		printColoredAt(screen, "Outputs:", lx, ly, color.RGBA{255, 180, 100, 200})
		ly += lineH
		for o := 0; o < swarm.NeuroOutputs; o++ {
			v := bot.Brain.OutputAct[o]
			outCol := dimCol
			if o == bot.Brain.ActionIdx {
				outCol = color.RGBA{255, 255, 100, 255}
			}
			printColoredAt(screen, fmt.Sprintf(" %s: %.2f", swarm.NeuroActionNames[o], v), lx, ly, outCol)
			ly += lineH - 3
		}
	}

	// --- Collective AI Chat Log (when Collective AI enabled) ---
	if ss.CollectiveAIOn {
		DrawBotChatLog(screen, ss, ss.SelectedBot, lx, ly+5)
	}
}

// EvaluateGPFitnessRender computes GP fitness for rendering (same formula as domain).
func EvaluateGPFitnessRender(bot *swarm.SwarmBot) float64 {
	return float64(bot.Stats.TotalDeliveries)*30 +
		float64(bot.Stats.TotalPickups)*15 +
		bot.Stats.TotalDistance*0.01 -
		float64(bot.Stats.AntiStuckCount)*10 -
		float64(bot.Stats.TicksIdle)*0.05
}

// drawBotComparisonPanel renders a side-by-side comparison of two bots.
func drawBotComparisonPanel(screen *ebiten.Image, ss *swarm.SwarmState) {
	botA := &ss.Bots[ss.SelectedBot]
	botB := &ss.Bots[ss.CompareBot]

	px := 830
	py := 60
	pw := 430
	ph := 370
	headerCol := color.RGBA{0, 220, 255, 255}
	dimCol := color.RGBA{140, 140, 160, 255}
	greenCol := color.RGBA{80, 255, 80, 255}

	// Background
	vector.DrawFilledRect(screen, float32(px), float32(py), float32(pw), float32(ph), ColorSwarmInfoBg, false)
	vector.StrokeRect(screen, float32(px), float32(py), float32(pw), float32(ph), 1, ColorSwarmEditorSep, false)

	lx := px + 5
	ly := py + 5
	colA := lx + 100  // column for Bot A values
	colB := lx + 270  // column for Bot B values

	// Title row
	printColoredAt(screen, "Comparison", lx, ly, headerCol)
	// LED swatches next to bot labels
	printColoredAt(screen, fmt.Sprintf("Bot #%d", ss.SelectedBot), colA, ly, ColorSwarmSelected)
	ledA := color.RGBA{botA.LEDColor[0], botA.LEDColor[1], botA.LEDColor[2], 255}
	vector.DrawFilledRect(screen, float32(colA+55), float32(ly+2), 8, 8, ledA, false)
	printColoredAt(screen, fmt.Sprintf("Bot #%d", ss.CompareBot), colB, ly, color.RGBA{0, 220, 255, 255})
	ledB := color.RGBA{botB.LEDColor[0], botB.LEDColor[1], botB.LEDColor[2], 255}
	vector.DrawFilledRect(screen, float32(colB+55), float32(ly+2), 8, 8, ledB, false)
	ly += lineH + 4

	// Separator
	vector.StrokeLine(screen, float32(px+3), float32(ly), float32(px+pw-3), float32(ly), 1, color.RGBA{60, 60, 80, 200}, false)
	ly += 4

	// Comparison row helper
	compareRow := func(label string, valA, valB float64, higherIsBetter bool) {
		printColoredAt(screen, label, lx, ly, dimCol)
		aStr := fmt.Sprintf("%.0f", valA)
		bStr := fmt.Sprintf("%.0f", valB)
		aCol := dimCol
		bCol := dimCol
		if higherIsBetter {
			if valA > valB {
				aCol = greenCol
			} else if valB > valA {
				bCol = greenCol
			}
		} else {
			if valA < valB {
				aCol = greenCol
			} else if valB < valA {
				bCol = greenCol
			}
		}
		printColoredAt(screen, aStr, colA, ly, aCol)
		printColoredAt(screen, bStr, colB, ly, bCol)
		ly += lineH
	}

	// Position & movement rows
	compareRow(locale.T("compare.position"), math.Sqrt(botA.X*botA.X+botA.Y*botA.Y), math.Sqrt(botB.X*botB.X+botB.Y*botB.Y), false)
	compareRow(locale.T("compare.speed"), botA.Speed, botB.Speed, true)
	compareRow(locale.T("compare.energy"), botA.Energy, botB.Energy, true)
	ly += 2

	// Separator
	vector.StrokeLine(screen, float32(px+3), float32(ly), float32(px+pw-3), float32(ly), 1, color.RGBA{60, 60, 80, 200}, false)
	ly += 4

	// Stats rows
	compareRow("Distance", botA.Stats.TotalDistance, botB.Stats.TotalDistance, true)
	compareRow("Pickups", float64(botA.Stats.TotalPickups), float64(botB.Stats.TotalPickups), true)
	compareRow(locale.T("stat.deliveries"), float64(botA.Stats.TotalDeliveries), float64(botB.Stats.TotalDeliveries), true)
	compareRow(locale.T("stat.correct"), float64(botA.Stats.CorrectDeliveries), float64(botB.Stats.CorrectDeliveries), true)
	compareRow(locale.T("stat.wrong"), float64(botA.Stats.WrongDeliveries), float64(botB.Stats.WrongDeliveries), false)
	compareRow("Msgs TX", float64(botA.Stats.MessagesSent), float64(botB.Stats.MessagesSent), true)
	compareRow("Msgs RX", float64(botA.Stats.MessagesReceived), float64(botB.Stats.MessagesReceived), true)
	compareRow("Stuck", float64(botA.Stats.AntiStuckCount), float64(botB.Stats.AntiStuckCount), false)
	compareRow("Idle", float64(botA.Stats.TicksIdle), float64(botB.Stats.TicksIdle), false)
	compareRow("Carrying", float64(botA.CarryingPkg+1), float64(botB.CarryingPkg+1), true)
	compareRow(locale.T("compare.fitness"), botA.Fitness, botB.Fitness, true)
	ly += 2

	// Separator
	vector.StrokeLine(screen, float32(px+3), float32(ly), float32(px+pw-3), float32(ly), 1, color.RGBA{60, 60, 80, 200}, false)
	ly += 4

	// Position info
	printColoredAt(screen, "Position", lx, ly, headerCol)
	printColoredAt(screen, fmt.Sprintf("(%.0f,%.0f)", botA.X, botA.Y), colA, ly, dimCol)
	printColoredAt(screen, fmt.Sprintf("(%.0f,%.0f)", botB.X, botB.Y), colB, ly, dimCol)
	ly += lineH

	// State info
	printColoredAt(screen, "State", lx, ly, headerCol)
	printColoredAt(screen, fmt.Sprintf("S:%d C:%d", botA.State, botA.Counter), colA, ly, dimCol)
	printColoredAt(screen, fmt.Sprintf("S:%d C:%d", botB.State, botB.Counter), colB, ly, dimCol)
	ly += lineH + 2

	printColoredAt(screen, locale.T("ui.compare_hint"), px+5, py+ph-lineH-2, color.RGBA{80, 80, 100, 200})
}
