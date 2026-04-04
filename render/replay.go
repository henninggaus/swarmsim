package render

import (
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawReplayOverlay renders the replay timeline and ghost bots from the snapshot.
func DrawReplayOverlay(screen *ebiten.Image, ss *swarm.SwarmState, replayIdx int) {
	rb := ss.ReplayBuf
	if rb == nil || rb.Count == 0 {
		return
	}

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	snap := rb.Get(replayIdx)
	if snap == nil {
		return
	}

	// Draw ghost bots from snapshot on the arena
	// Need to apply same camera transform as swarm_render
	viewportX := 415.0
	viewportW := float64(sw) - viewportX
	viewportH := float64(sh)

	camX := ss.SwarmCamX
	camY := ss.SwarmCamY
	zoom := ss.SwarmCamZoom
	if zoom < 0.5 {
		zoom = 1.0
	}

	for _, bot := range snap.BotData {
		// Transform arena coords to screen coords
		screenX := viewportX + (bot.X-camX)*zoom + viewportW/2 + camX*zoom - ss.ArenaW*zoom/2
		screenY := (bot.Y-camY)*zoom + viewportH/2 + camY*zoom - ss.ArenaH*zoom/2

		// Skip if offscreen
		if screenX < viewportX || screenX > float64(sw) || screenY < 0 || screenY > float64(sh) {
			continue
		}

		bx := float32(screenX)
		by := float32(screenY)
		radius := float32(swarm.SwarmBotRadius) * float32(zoom)

		// Semi-transparent ghost with LED color
		r, g, b := bot.LEDR, bot.LEDG, bot.LEDB
		if r < 60 && g < 60 && b < 60 {
			r, g, b = 80, 80, 80
		}
		ghostCol := color.RGBA{r, g, b, 140}
		vector.DrawFilledCircle(screen, bx, by, radius, ghostCol, false)

		// Direction line
		dirLen := radius * 1.5
		dx := float32(math.Cos(bot.Angle)) * dirLen
		dy := float32(math.Sin(bot.Angle)) * dirLen
		vector.StrokeLine(screen, bx, by, bx+dx, by+dy, 1, color.RGBA{255, 255, 255, 100}, false)

		// Carrying indicator
		if bot.Carrying {
			vector.StrokeCircle(screen, bx, by, radius+3, 1.5, color.RGBA{255, 200, 50, 120}, false)
		}
	}

	// Timeline bar at bottom
	barY := float32(sh - 35)
	barX := float32(viewportX + 20)
	barW := float32(viewportW - 40)
	barH := float32(16)

	// Background
	vector.DrawFilledRect(screen, barX-2, barY-20, barW+4, barH+30, color.RGBA{0, 0, 0, 200}, false)

	// Progress bar track
	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{30, 30, 50, 220}, false)

	// Fill
	progress := float32(replayIdx) / float32(rb.Count-1)
	vector.DrawFilledRect(screen, barX, barY, barW*progress, barH, color.RGBA{80, 160, 255, 180}, false)

	// Cursor
	cursorX := barX + barW*progress
	vector.DrawFilledRect(screen, cursorX-2, barY-3, 4, barH+6, ColorWhite, false)

	// Labels
	labelCol := ColorTextLight
	dimCol := color.RGBA{120, 130, 150, 200}

	title := locale.Tf("replay.title", snap.Tick, replayIdx+1, rb.Count)
	printColoredAt(screen, title, int(barX), int(barY-18), labelCol)

	hint := locale.T("replay.hint")
	printColoredAt(screen, hint, int(barX), int(barY+barH+4), dimCol)
}
