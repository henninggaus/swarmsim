package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Block editor layout constants
const (
	blockRuleH     = 22 // height per condition row
	blockGap       = 4  // gap between rules
	blockPadX      = 4
	blockSensorW   = 72
	blockOpW       = 22
	blockValueW    = 30
	blockActionW   = 105
	blockDeleteW   = 14
	blockIfW       = 20 // "IF"/"AND" label width
	blockThenW     = 30 // "THEN" label width
	blockNewRuleH  = 22
	ddItemH        = 18 // dropdown item height
	ddMaxVisible   = 12 // max visible dropdown items
)

// Colors for block editor elements
var (
	colorBlockBg       = color.RGBA{25, 28, 40, 255}
	colorBlockRuleBg   = color.RGBA{32, 36, 52, 255}
	colorBlockSensorBg = color.RGBA{20, 60, 35, 255}
	colorBlockOpBg     = color.RGBA{40, 40, 55, 255}
	colorBlockValueBg  = color.RGBA{50, 45, 20, 255}
	colorBlockActionBg = color.RGBA{60, 40, 15, 255}
	colorBlockDeleteBg = color.RGBA{80, 25, 25, 255}
	colorBlockBorder   = color.RGBA{70, 70, 90, 180}
	colorBlockAndBtn   = color.RGBA{30, 55, 45, 255}
	colorBlockNewRule  = color.RGBA{35, 55, 80, 255}
	colorDDHeaderBg    = color.RGBA{20, 22, 32, 255}
	colorDDItemBg      = color.RGBA{35, 40, 60, 240}
	colorDDItemHover   = color.RGBA{60, 80, 140, 240}
	colorDDBorder      = color.RGBA{80, 80, 110, 200}
)

// DrawBlockEditor renders the visual block editor in the code area.
func DrawBlockEditor(screen *ebiten.Image, ss *swarm.SwarmState) {
	// Background
	vector.DrawFilledRect(screen, 0, float32(editorCodeY), float32(editorPanelW), float32(editorCodeH), colorBlockBg, false)

	// Clip region offset
	y := editorCodeY + 2 - ss.BlockScrollY

	for ri, rule := range ss.BlockRules {
		ruleH := len(rule.Conditions) * blockRuleH
		if ruleH < blockRuleH {
			ruleH = blockRuleH
		}

		// Skip if completely above visible area
		if y+ruleH+blockGap < editorCodeY {
			y += ruleH + blockGap
			continue
		}
		// Stop if below visible area
		if y > editorCodeY+editorCodeH {
			break
		}

		// Rule background
		vector.DrawFilledRect(screen, float32(blockPadX-1), float32(y-1),
			float32(editorPanelW-blockPadX*2+2), float32(ruleH+2), colorBlockRuleBg, false)

		// Draw each condition line
		for ci, cond := range rule.Conditions {
			cy := y + ci*blockRuleH
			if cy < editorCodeY-blockRuleH || cy > editorCodeY+editorCodeH {
				continue
			}

			x := blockPadX

			// IF / AND label
			if ci == 0 {
				printColoredAt(screen, "IF", x, cy+3, ColorSwarmKeyword)
			} else {
				printColoredAt(screen, "AND", x, cy+3, ColorSwarmKeyword)
			}
			x += blockIfW

			// Sensor dropdown button
			drawBlockBtn(screen, x, cy+1, blockSensorW, blockRuleH-2, cond.SensorName, colorBlockSensorBg, ColorSwarmCondition)
			x += blockSensorW + 2

			// Op dropdown button
			drawBlockBtn(screen, x, cy+1, blockOpW, blockRuleH-2, cond.OpStr, colorBlockOpBg, ColorSwarmOperator)
			x += blockOpW + 2

			// Value field
			valStr := fmt.Sprintf("%d", cond.Value)
			if ss.BlockValueEdit && ss.BlockValueRuleIdx == ri && ss.BlockValueCondIdx == ci {
				valStr = ss.BlockValueText + "_"
			}
			drawBlockBtn(screen, x, cy+1, blockValueW, blockRuleH-2, valStr, colorBlockValueBg, ColorSwarmNumber)
		}

		// THEN + Action on the last condition row
		lastY := y + (len(rule.Conditions)-1)*blockRuleH
		if lastY >= editorCodeY-blockRuleH && lastY <= editorCodeY+editorCodeH {
			ax := blockPadX + blockIfW + blockSensorW + blockOpW + blockValueW + 8
			printColoredAt(screen, "THEN", ax, lastY+3, ColorSwarmKeyword)
			ax += blockThenW

			actLabel := rule.ActionName
			if len(actLabel) > 16 {
				actLabel = actLabel[:15] + "."
			}
			drawBlockBtn(screen, ax, lastY+1, blockActionW, blockRuleH-2, actLabel, colorBlockActionBg, ColorSwarmAction)

			// [X] delete button
			delX := editorPanelW - blockPadX - blockDeleteW - 2
			drawBlockBtn(screen, delX, y+1, blockDeleteW, blockDeleteW, "X", colorBlockDeleteBg, color.RGBA{255, 100, 100, 255})
		}

		// [+AND] mini button below the rule
		andY := y + ruleH
		if andY >= editorCodeY && andY < editorCodeY+editorCodeH-blockRuleH {
			_ = ri // needed for hit-test, rendered as subtle text
			printColoredAt(screen, "+AND", blockPadX+blockIfW, andY+1, color.RGBA{60, 100, 80, 180})
		}

		y += ruleH + blockGap
	}

	// [+ New Rule] button at bottom
	if y >= editorCodeY && y < editorCodeY+editorCodeH-blockNewRuleH {
		drawSwarmButton(screen, blockPadX, y, editorPanelW-blockPadX*2, blockNewRuleH,
			"+ Neue Regel", colorBlockNewRule)
	}

	// Active dropdown overlay (drawn on top of everything)
	if ss.ActiveDropdown != nil {
		drawBlockDropdownOverlay(screen, ss.ActiveDropdown)
	}
}

func drawBlockBtn(screen *ebiten.Image, x, y, w, h int, label string, bgCol, textCol color.RGBA) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), bgCol, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, colorBlockBorder, false)
	maxChars := (w - 4) / charW
	if maxChars < 1 {
		maxChars = 1
	}
	if len(label) > maxChars {
		label = label[:maxChars-1] + "."
	}
	printColoredAt(screen, label, x+2, y+2, textCol)
}

func drawBlockDropdownOverlay(screen *ebiten.Image, dd *swarm.BlockDropdown) {
	x := dd.X
	y := dd.Y
	w := 140
	if dd.FieldType == "op" {
		w = 40
	}

	visible := len(dd.Items)
	if visible > ddMaxVisible {
		visible = ddMaxVisible
	}
	h := visible * ddItemH

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), colorDDItemBg, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, colorDDBorder, false)

	for i := dd.ScrollY; i < dd.ScrollY+visible && i < len(dd.Items); i++ {
		iy := y + (i-dd.ScrollY)*ddItemH
		item := dd.Items[i]
		isHeader := len(item) > 2 && item[:2] == "--"

		if isHeader {
			vector.DrawFilledRect(screen, float32(x), float32(iy), float32(w), float32(ddItemH), colorDDHeaderBg, false)
			printColoredAt(screen, item, x+4, iy+3, color.RGBA{90, 90, 110, 255})
		} else {
			if i == dd.HoverIdx {
				vector.DrawFilledRect(screen, float32(x), float32(iy), float32(w), float32(ddItemH), colorDDItemHover, false)
			}
			printColoredAt(screen, item, x+6, iy+3, color.RGBA{200, 200, 220, 255})
		}
	}
}

// BlockEditorHitTest returns what was clicked in the block editor.
// Returns format: "action_type:ruleIdx:condIdx" or special values.
func BlockEditorHitTest(mx, my int, ss *swarm.SwarmState) (action string, ruleIdx, condIdx int) {
	if mx < 0 || mx >= editorPanelW || my < editorCodeY || my >= editorCodeY+editorCodeH {
		return "", -1, -1
	}

	y := editorCodeY + 2 - ss.BlockScrollY

	for ri, rule := range ss.BlockRules {
		ruleH := len(rule.Conditions) * blockRuleH
		if ruleH < blockRuleH {
			ruleH = blockRuleH
		}

		// Check [+AND] button area (in the gap below the rule)
		andY := y + ruleH
		if my >= andY && my < andY+blockGap+4 && mx >= blockPadX+blockIfW && mx < blockPadX+blockIfW+30 {
			return "add_cond", ri, -1
		}

		// Check within rule bounds
		if my >= y && my < y+ruleH {
			// Which condition row?
			ci := (my - y) / blockRuleH
			if ci >= len(rule.Conditions) {
				ci = len(rule.Conditions) - 1
			}

			cx := blockPadX + blockIfW

			// Sensor button
			if mx >= cx && mx < cx+blockSensorW {
				return "sensor", ri, ci
			}
			cx += blockSensorW + 2

			// Op button
			if mx >= cx && mx < cx+blockOpW {
				return "op", ri, ci
			}
			cx += blockOpW + 2

			// Value field
			if mx >= cx && mx < cx+blockValueW {
				return "value", ri, ci
			}

			// THEN + Action (only on last condition row)
			if ci == len(rule.Conditions)-1 {
				ax := blockPadX + blockIfW + blockSensorW + blockOpW + blockValueW + 8 + blockThenW
				if mx >= ax && mx < ax+blockActionW {
					return "action", ri, -1
				}
			}

			// [X] delete
			delX := editorPanelW - blockPadX - blockDeleteW - 2
			if mx >= delX && mx < delX+blockDeleteW && my >= y && my < y+blockDeleteW+2 {
				return "delete", ri, -1
			}

			y += ruleH + blockGap
			continue
		}

		y += ruleH + blockGap
	}

	// [+ New Rule] button
	if my >= y && my < y+blockNewRuleH {
		return "new_rule", -1, -1
	}

	return "", -1, -1
}

// BlockDropdownHitTest returns the item index clicked in the active dropdown, or -1.
func BlockDropdownHitTest(mx, my int, dd *swarm.BlockDropdown) int {
	w := 140
	if dd.FieldType == "op" {
		w = 40
	}
	visible := len(dd.Items)
	if visible > ddMaxVisible {
		visible = ddMaxVisible
	}

	if mx < dd.X || mx >= dd.X+w || my < dd.Y || my >= dd.Y+visible*ddItemH {
		return -1 // clicked outside
	}

	idx := (my-dd.Y)/ddItemH + dd.ScrollY
	if idx < 0 || idx >= len(dd.Items) {
		return -1
	}
	// Don't select headers
	item := dd.Items[idx]
	if len(item) > 2 && item[:2] == "--" {
		return -1
	}
	return idx
}
