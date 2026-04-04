package render

import (
	"fmt"
	"image/color"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawIssueBoard draws the Self-Programming Swarm issue board overlay.
func DrawIssueBoard(screen *ebiten.Image, ss *swarm.SwarmState) {
	if !ss.ShowIssueBoard || ss.IssueBoard == nil {
		return
	}

	// Semi-transparent panel on the right side
	x := 750
	y := 60
	w := 500
	h := 600

	// Background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h),
		ColorPanelBg, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h),
		1, color.RGBA{80, 80, 120, 255}, false)

	// Title
	titleStr := locale.T("issueboard.title")
	printColoredAt(screen, titleStr, x+8, y+8, color.RGBA{255, 200, 50, 255})

	// Brief description
	printColoredAt(screen, locale.T("issueboard.desc"), x+8, y+22, color.RGBA{140, 150, 170, 200})

	// Legend: colored dots with labels
	legendY := y + 36
	legendItems := []struct {
		status string
		col    color.RGBA
	}{
		{locale.T("issue.status.open"), color.RGBA{150, 150, 150, 255}},
		{locale.T("claude.waiting"), color.RGBA{160, 100, 220, 255}},
		{locale.T("issue.status.testing"), color.RGBA{220, 200, 40, 255}},
		{locale.T("issue.status.resolved"), color.RGBA{60, 220, 60, 255}},
		{locale.T("issue.status.failed"), color.RGBA{220, 60, 60, 255}},
	}
	for i, item := range legendItems {
		lx := x + 8 + i*90
		vector.DrawFilledCircle(screen, float32(lx), float32(legendY+4), 3, item.col, false)
		printColoredAt(screen, item.status, lx+8, legendY, color.RGBA{160, 165, 180, 200})
	}

	// Status summary
	ib := ss.IssueBoard
	openCount, testCount, resolvedCount, failedCount, codegenCount := 0, 0, 0, 0, 0
	for i := range ib.Issues {
		switch ib.Issues[i].Status {
		case swarm.IssueOpen:
			openCount++
		case swarm.IssueCodeGen:
			codegenCount++
		case swarm.IssueTesting:
			testCount++
		case swarm.IssueResolved:
			resolvedCount++
		case swarm.IssueFailed:
			failedCount++
		}
	}
	summary := fmt.Sprintf("Open:%d  Testing:%d  Resolved:%d  Failed:%d", openCount+codegenCount, testCount, resolvedCount, failedCount)
	printColoredAt(screen, summary, x+8, y+50, color.RGBA{180, 180, 200, 255})

	// Collective AI toggle status
	var aiStatus string
	if ss.CollectiveAIOn {
		aiStatus = locale.T("collective.on")
	} else {
		aiStatus = locale.T("collective.off")
	}
	printColoredAt(screen, aiStatus, x+350, y+8, color.RGBA{100, 255, 100, 255})

	// Hint: AI is off
	if !ss.CollectiveAIOn {
		hint := locale.T("issueboard.ai_off_hint")
		printColoredAt(screen, hint, x+w/2-runeLen(hint)*charW/2, y+h/2, color.RGBA{220, 180, 50, 255})
	}

	// Hint: AI on but no issues yet
	if len(ib.Issues) == 0 && ss.CollectiveAIOn {
		emptyMsg := locale.T("issueboard.empty")
		waitMsg := locale.T("issueboard.wait")
		printColoredAt(screen, emptyMsg, x+20, y+h/3, color.RGBA{160, 170, 190, 220})
		printColoredAt(screen, waitMsg, x+20, y+h/3+lineH, color.RGBA{130, 140, 160, 180})
	}

	// Issues list (scrollable, max 15 visible)
	ly := y + 66
	maxVisible := 15
	startIdx := ss.IssueBoardScroll
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx > len(ib.Issues)-maxVisible {
		startIdx = len(ib.Issues) - maxVisible
	}
	if startIdx < 0 {
		startIdx = 0
	}

	for idx := startIdx; idx < len(ib.Issues) && idx < startIdx+maxVisible; idx++ {
		issue := &ib.Issues[idx]
		ly += 2

		// Status color
		var statusCol color.RGBA
		var statusStr string
		switch issue.Status {
		case swarm.IssueOpen:
			statusCol = color.RGBA{150, 150, 150, 255}
			statusStr = locale.T("issue.status.open")
		case swarm.IssueCodeGen:
			statusCol = color.RGBA{180, 140, 255, 255}
			statusStr = locale.T("claude.waiting")
		case swarm.IssueTesting:
			statusCol = color.RGBA{255, 220, 50, 255}
			statusStr = locale.T("issue.status.testing")
		case swarm.IssueResolved:
			statusCol = color.RGBA{50, 255, 50, 255}
			statusStr = locale.T("issue.status.resolved")
		case swarm.IssueFailed:
			statusCol = color.RGBA{255, 80, 80, 255}
			statusStr = locale.T("issue.status.failed")
		default:
			statusCol = color.RGBA{100, 100, 100, 255}
			statusStr = "?"
		}

		// Problem type label
		var problemStr string
		switch issue.Problem {
		case "stuck":
			problemStr = locale.T("issue.stuck")
		case "no_package":
			problemStr = locale.T("issue.no_package")
		case "obstacle":
			problemStr = locale.T("issue.obstacle")
		case "isolated":
			problemStr = locale.T("issue.isolated")
		case "slow_delivery":
			problemStr = locale.T("issue.slow_delivery")
		case "energy_crisis":
			problemStr = locale.T("issue.energy_crisis")
		default:
			problemStr = issue.Problem
		}

		// Draw issue row
		// Status indicator dot
		vector.DrawFilledCircle(screen, float32(x+10), float32(ly+5), 4, statusCol, false)

		// Bot# + problem
		line := fmt.Sprintf("Bot#%d: %s", issue.BotIdx, problemStr)
		printColoredAt(screen, line, x+20, ly, color.RGBA{220, 220, 240, 255})

		// Status text
		printColoredAt(screen, statusStr, x+280, ly, statusCol)

		// Tick info
		tickStr := fmt.Sprintf("T%d", issue.Tick)
		printColoredAt(screen, tickStr, x+360, ly, color.RGBA{100, 100, 130, 255})

		// Code preview (first 30 chars)
		if issue.GeneratedCode != "" {
			codePreview := issue.GeneratedCode
			if len(codePreview) > 40 {
				codePreview = codePreview[:40] + "..."
			}
			// Replace newlines with semicolons for single-line display
			cleanCode := ""
			for _, ch := range codePreview {
				if ch == '\n' {
					cleanCode += "; "
				} else {
					cleanCode += string(ch)
				}
			}
			if len(cleanCode) > 50 {
				cleanCode = cleanCode[:50] + "..."
			}
			ly += lineH
			printColoredAt(screen, cleanCode, x+20, ly, color.RGBA{120, 160, 200, 200})
		}

		ly += lineH
	}

	// Proven solutions count
	provenCount := len(ib.ProvenSolutions)
	if provenCount > 0 {
		provenStr := fmt.Sprintf("Proven solutions: %d", provenCount)
		printColoredAt(screen, provenStr, x+8, y+h-20, color.RGBA{100, 255, 150, 200})
	}

	// Claude API status line
	if ib.UseClaudeAPI && ib.ClaudeBackend != nil {
		apiStr := locale.T("claude.active")
		printColoredAt(screen, apiStr, x+8, y+h-36, color.RGBA{180, 140, 255, 255})
		if ib.ClaudeLastErr != "" {
			errStr := fmt.Sprintf(locale.T("claude.error"), ib.ClaudeLastErr)
			if len(errStr) > 60 {
				errStr = errStr[:60] + "..."
			}
			printColoredAt(screen, errStr, x+200, y+h-36, color.RGBA{255, 120, 80, 200})
		}
	} else {
		apiStr := locale.T("claude.inactive")
		printColoredAt(screen, apiStr, x+8, y+h-36, color.RGBA{120, 120, 140, 180})
	}

	// Scroll hint + down arrow indicator
	if len(ib.Issues) > maxVisible {
		scrollHint := fmt.Sprintf("[scroll: %d/%d]", startIdx+1, len(ib.Issues))
		printColoredAt(screen, scrollHint, x+380, y+h-20, color.RGBA{100, 100, 140, 200})
		// Show down arrow if more items below
		if startIdx+maxVisible < len(ib.Issues) {
			arrowX := x + w/2 - 15
			arrowY := y + h - 8
			printColoredAt(screen, "v v v", arrowX, arrowY, color.RGBA{150, 160, 180, 150})
		}
	}

	// ESC hint at bottom
	escHint := locale.T("overlay.esc_close")
	printColoredAt(screen, escHint, x+w/2-runeLen(escHint)*charW/2, y+h-14, color.RGBA{120, 130, 150, 180})
}

// DrawBotChatLog draws the AI chat log for a selected bot inside the bot info panel.
func DrawBotChatLog(screen *ebiten.Image, ss *swarm.SwarmState, botIdx int, x, y int) {
	if botIdx < 0 || botIdx >= len(ss.BotChatLog) || ss.BotChatLog[botIdx] == nil {
		return
	}

	entries := ss.BotChatLog[botIdx]
	if len(entries) == 0 {
		return
	}

	// Title
	printColoredAt(screen, locale.T("chatlog.title"), x, y, color.RGBA{255, 200, 50, 255})
	ly := y + lineH + 2

	// Show last 5 entries
	start := len(entries) - 5
	if start < 0 {
		start = 0
	}

	for _, entry := range entries[start:] {
		// Problem line
		probLine := fmt.Sprintf("[T%d] %s", entry.Tick, entry.Problem)
		printColoredAt(screen, probLine, x, ly, color.RGBA{200, 200, 220, 255})
		ly += lineH

		// Result with color coding
		var resultCol color.RGBA
		switch {
		case entry.Result == "RESOLVED":
			resultCol = color.RGBA{50, 255, 50, 255}
		case entry.Result == "FAILED" || len(entry.Result) > 5 && entry.Result[:6] == "FAILED":
			resultCol = color.RGBA{255, 80, 80, 255}
		case entry.Result == "TESTING":
			resultCol = color.RGBA{255, 220, 50, 255}
		default:
			resultCol = color.RGBA{100, 200, 255, 255} // ADOPTED
		}
		printColoredAt(screen, entry.Result, x, ly, resultCol)
		ly += lineH + 2
	}
}
