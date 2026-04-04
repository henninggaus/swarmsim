package main

import (
	"fmt"
	"strconv"
	"swarmsim/domain/swarm"
	"swarmsim/logger"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Game) handleSwarmEditorKeys() {
	ss := g.sim.SwarmState
	ed := ss.Editor

	// Character input
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		g.editorInsertChar(ch)
	}

	// Enter: new line
	if isKeyRepeating(ebiten.KeyEnter) {
		line := ed.Lines[ed.CursorLine]
		before := line[:ed.CursorCol]
		after := line[ed.CursorCol:]
		ed.Lines[ed.CursorLine] = before
		// Insert new line after current
		newLines := make([]string, len(ed.Lines)+1)
		copy(newLines, ed.Lines[:ed.CursorLine+1])
		newLines[ed.CursorLine+1] = after
		copy(newLines[ed.CursorLine+2:], ed.Lines[ed.CursorLine+1:])
		ed.Lines = newLines
		ed.CursorLine++
		ed.CursorCol = 0
		g.editorEnsureCursorVisible()
		ss.ProgramName = "Custom"
		ss.IsDeliveryProgram = false
	}

	// Backspace
	if isKeyRepeating(ebiten.KeyBackspace) {
		if ed.CursorCol > 0 {
			line := ed.Lines[ed.CursorLine]
			ed.Lines[ed.CursorLine] = line[:ed.CursorCol-1] + line[ed.CursorCol:]
			ed.CursorCol--
			ss.ProgramName = "Custom"
			ss.IsDeliveryProgram = false
		} else if ed.CursorLine > 0 {
			// Merge with previous line
			prevLine := ed.Lines[ed.CursorLine-1]
			curLine := ed.Lines[ed.CursorLine]
			ed.Lines[ed.CursorLine-1] = prevLine + curLine
			ed.Lines = append(ed.Lines[:ed.CursorLine], ed.Lines[ed.CursorLine+1:]...)
			ed.CursorLine--
			ed.CursorCol = len(prevLine)
			g.editorEnsureCursorVisible()
			ss.ProgramName = "Custom"
			ss.IsDeliveryProgram = false
		}
	}

	// Delete
	if isKeyRepeating(ebiten.KeyDelete) {
		line := ed.Lines[ed.CursorLine]
		if ed.CursorCol < len(line) {
			ed.Lines[ed.CursorLine] = line[:ed.CursorCol] + line[ed.CursorCol+1:]
			ss.ProgramName = "Custom"
			ss.IsDeliveryProgram = false
		} else if ed.CursorLine < len(ed.Lines)-1 {
			// Merge with next line
			nextLine := ed.Lines[ed.CursorLine+1]
			ed.Lines[ed.CursorLine] = line + nextLine
			ed.Lines = append(ed.Lines[:ed.CursorLine+1], ed.Lines[ed.CursorLine+2:]...)
			ss.ProgramName = "Custom"
			ss.IsDeliveryProgram = false
		}
	}

	// Arrow keys
	if isKeyRepeating(ebiten.KeyLeft) {
		if ed.CursorCol > 0 {
			ed.CursorCol--
		} else if ed.CursorLine > 0 {
			ed.CursorLine--
			ed.CursorCol = len(ed.Lines[ed.CursorLine])
		}
		ed.BlinkTick = 0
		g.editorEnsureCursorVisible()
	}
	if isKeyRepeating(ebiten.KeyRight) {
		lineLen := len(ed.Lines[ed.CursorLine])
		if ed.CursorCol < lineLen {
			ed.CursorCol++
		} else if ed.CursorLine < len(ed.Lines)-1 {
			ed.CursorLine++
			ed.CursorCol = 0
		}
		ed.BlinkTick = 0
		g.editorEnsureCursorVisible()
	}
	if isKeyRepeating(ebiten.KeyUp) {
		if ed.CursorLine > 0 {
			ed.CursorLine--
			if ed.CursorCol > len(ed.Lines[ed.CursorLine]) {
				ed.CursorCol = len(ed.Lines[ed.CursorLine])
			}
		}
		ed.BlinkTick = 0
		g.editorEnsureCursorVisible()
	}
	if isKeyRepeating(ebiten.KeyDown) {
		if ed.CursorLine < len(ed.Lines)-1 {
			ed.CursorLine++
			if ed.CursorCol > len(ed.Lines[ed.CursorLine]) {
				ed.CursorCol = len(ed.Lines[ed.CursorLine])
			}
		}
		ed.BlinkTick = 0
		g.editorEnsureCursorVisible()
	}

	// Home / End
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		ed.CursorCol = 0
		ed.BlinkTick = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		ed.CursorCol = len(ed.Lines[ed.CursorLine])
		ed.BlinkTick = 0
	}

	// Tab: insert 4 spaces
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		for i := 0; i < 4; i++ {
			g.editorInsertChar(' ')
		}
	}
}

func (g *Game) editorInsertChar(ch rune) {
	ss := g.sim.SwarmState
	ed := ss.Editor

	if ch < 32 || ch > 126 {
		return // only printable ASCII
	}

	line := ed.Lines[ed.CursorLine]
	ed.Lines[ed.CursorLine] = line[:ed.CursorCol] + string(ch) + line[ed.CursorCol:]
	ed.CursorCol++
	ed.BlinkTick = 0
	ss.ProgramName = "Custom"
	ss.IsDeliveryProgram = false
}

func (g *Game) editorEnsureCursorVisible() {
	ed := g.sim.SwarmState.Editor
	// Vertical scroll
	if ed.CursorLine < ed.ScrollY {
		ed.ScrollY = ed.CursorLine
	}
	if ed.CursorLine >= ed.ScrollY+ed.MaxVisible {
		ed.ScrollY = ed.CursorLine - ed.MaxVisible + 1
	}
	// Horizontal scroll -- keep cursor within visible columns
	// editorPanelW=350, editorTextX=40, charW=6 -> maxVisibleCols = 51
	maxVisibleCols := (350 - 2 - 40) / 6 // = 51
	if ed.CursorCol < ed.ScrollX {
		ed.ScrollX = ed.CursorCol
	}
	if ed.CursorCol >= ed.ScrollX+maxVisibleCols {
		ed.ScrollX = ed.CursorCol - maxVisibleCols + 1
	}
	if ed.ScrollX < 0 {
		ed.ScrollX = 0
	}
}

func (g *Game) handleBotCountInput() {
	ss := g.sim.SwarmState

	// Consume character input
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if ch >= '0' && ch <= '9' {
			ss.BotCountText += string(ch)
		}
	}

	// Backspace
	if isKeyRepeating(ebiten.KeyBackspace) && len(ss.BotCountText) > 0 {
		ss.BotCountText = ss.BotCountText[:len(ss.BotCountText)-1]
	}

	// Enter: apply bot count
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		count, err := strconv.Atoi(ss.BotCountText)
		if err == nil && count >= swarm.SwarmMinBots && count <= swarm.SwarmMaxBots {
			ss.RespawnBots(count)
			ss.ResetFlashTimer = 30
			logger.Info("SWARM", "Bot count changed to %d", count)
		} else {
			// Reset to current count
			ss.BotCountText = fmt.Sprintf("%d", ss.BotCount)
		}
		ss.BotCountEdit = false
		ss.Editor.Focused = true
	}
}

func (g *Game) handleBlockValueInput() {
	ss := g.sim.SwarmState

	// Accept digits and minus
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if (ch >= '0' && ch <= '9') || (ch == '-' && len(ss.BlockValueText) == 0) {
			ss.BlockValueText += string(ch)
		}
	}

	// Backspace
	if isKeyRepeating(ebiten.KeyBackspace) && len(ss.BlockValueText) > 0 {
		ss.BlockValueText = ss.BlockValueText[:len(ss.BlockValueText)-1]
	}

	// Enter or Escape: commit
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ri := ss.BlockValueRuleIdx
		ci := ss.BlockValueCondIdx
		if ri >= 0 && ri < len(ss.BlockRules) && ci >= 0 && ci < len(ss.BlockRules[ri].Conditions) {
			val, err := strconv.Atoi(ss.BlockValueText)
			if err == nil {
				ss.BlockRules[ri].Conditions[ci].Value = val
			}
		}
		ss.BlockValueEdit = false
	}
}

// isKeyRepeating returns true if a key was just pressed OR is being held long enough to repeat.
func isKeyRepeating(key ebiten.Key) bool {
	d := inpututil.KeyPressDuration(key)
	if d == 1 {
		return true // just pressed
	}
	if d >= 20 && (d-20)%3 == 0 {
		return true // repeat after 20 ticks, every 3 ticks
	}
	return false
}
