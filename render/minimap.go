package render

import (
	"image/color"
	"swarmsim/domain/swarm"
	"swarmsim/engine/simulation"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	minimapW      = 150
	minimapH      = 100
	minimapMargin = 10
)

var (
	minimapBg     = color.RGBA{0, 0, 0, 180}
	minimapBorder = color.RGBA{100, 100, 100, 200}
)

// drawMinimapFrame draws the background and border shared by both minimap modes.
func drawMinimapFrame(screen *ebiten.Image, x, y float32) {
	vector.DrawFilledRect(screen, x, y, minimapW, minimapH, minimapBg, false)
	vector.StrokeRect(screen, x, y, minimapW, minimapH, 1, minimapBorder, false)
}

// drawMinimap renders the standard-mode minimap in the bottom-right corner.
// Shows obstacles, home base, bots, and the camera viewport rectangle.
func (r *Renderer) drawMinimap(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	mx := float32(sw - minimapW - minimapMargin)
	my := float32(sh - minimapH - minimapMargin)

	drawMinimapFrame(screen, mx, my)

	arenaW := s.Cfg.ArenaWidth
	arenaH := s.Cfg.ArenaHeight
	scaleX := float64(minimapW) / arenaW
	scaleY := float64(minimapH) / arenaH

	// Obstacles
	for _, obs := range s.Arena.Obstacles {
		ox := mx + float32(obs.X*scaleX)
		oy := my + float32(obs.Y*scaleY)
		ow := float32(obs.W * scaleX)
		oh := float32(obs.H * scaleY)
		if ow < 1 {
			ow = 1
		}
		if oh < 1 {
			oh = 1
		}
		vector.DrawFilledRect(screen, ox, oy, ow, oh, color.RGBA{128, 128, 128, 200}, false)
	}

	// Home base
	hx := mx + float32(s.Arena.HomeBaseX*scaleX)
	hy := my + float32(s.Arena.HomeBaseY*scaleY)
	vector.DrawFilledCircle(screen, hx, hy, 3, color.RGBA{80, 120, 255, 200}, false)

	// Bots as 2px colored dots
	for _, b := range s.Bots {
		if !b.IsAlive() {
			continue
		}
		pos := b.Position()
		bx := mx + float32(pos.X*scaleX)
		by := my + float32(pos.Y*scaleY)
		col := BotColor(b.Type())
		vector.DrawFilledRect(screen, bx-1, by-1, 2, 2, col, false)
	}

	// Camera viewport rectangle
	tlx, tly := r.Camera.ScreenToWorld(0, 0, sw, sh)
	brx, bry := r.Camera.ScreenToWorld(float64(sw), float64(sh), sw, sh)

	// Clamp to arena bounds
	if tlx < 0 {
		tlx = 0
	}
	if tly < 0 {
		tly = 0
	}
	if brx > arenaW {
		brx = arenaW
	}
	if bry > arenaH {
		bry = arenaH
	}

	vx := mx + float32(tlx*scaleX)
	vy := my + float32(tly*scaleY)
	vw := float32((brx - tlx) * scaleX)
	vh := float32((bry - tly) * scaleY)
	vector.StrokeRect(screen, vx, vy, vw, vh, 1, ColorWhiteFaded, false)
}

// drawSwarmMinimap renders the swarm-mode minimap in the bottom-right of the arena viewport.
// Shows obstacles, maze walls, delivery stations, bots (LED color), and selected bot highlight.
func (r *Renderer) drawSwarmMinimap(screen *ebiten.Image, ss *swarm.SwarmState) {
	arenaOffX := float32(415.0)
	arenaOffY := float32(50.0)

	mx := arenaOffX + float32(ss.ArenaW) - minimapW - 5
	my := arenaOffY + float32(ss.ArenaH) - minimapH - 35 // raised to avoid bottom HUD bars

	drawMinimapFrame(screen, mx, my)

	scaleX := float64(minimapW) / ss.ArenaW
	scaleY := float64(minimapH) / ss.ArenaH

	// Obstacles
	for _, obs := range ss.Obstacles {
		ox := mx + float32(obs.X*scaleX)
		oy := my + float32(obs.Y*scaleY)
		ow := float32(obs.W * scaleX)
		oh := float32(obs.H * scaleY)
		if ow < 1 {
			ow = 1
		}
		if oh < 1 {
			oh = 1
		}
		vector.DrawFilledRect(screen, ox, oy, ow, oh, color.RGBA{100, 100, 110, 200}, false)
	}

	// Maze walls
	for _, wall := range ss.MazeWalls {
		wx := mx + float32(wall.X*scaleX)
		wy := my + float32(wall.Y*scaleY)
		ww := float32(wall.W * scaleX)
		wh := float32(wall.H * scaleY)
		if ww < 1 {
			ww = 1
		}
		if wh < 1 {
			wh = 1
		}
		vector.DrawFilledRect(screen, wx, wy, ww, wh, color.RGBA{160, 160, 180, 180}, false)
	}

	// Delivery stations as colored dots
	if ss.DeliveryOn {
		for si := range ss.Stations {
			st := &ss.Stations[si]
			sx := mx + float32(st.X*scaleX)
			sy := my + float32(st.Y*scaleY)
			col := deliveryColor(st.Color)
			if st.IsPickup {
				vector.DrawFilledCircle(screen, sx, sy, 3, col, false)
			} else {
				vector.StrokeCircle(screen, sx, sy, 3, 1, col, false)
			}
		}
	}

	// Bots as 2px dots in LED color
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		bx := mx + float32(bot.X*scaleX)
		by := my + float32(bot.Y*scaleY)
		botCol := color.RGBA{bot.LEDColor[0], bot.LEDColor[1], bot.LEDColor[2], 255}
		vector.DrawFilledRect(screen, bx-1, by-1, 2, 2, botCol, false)
	}

	// Selected bot highlight (slightly larger yellow dot)
	if ss.SelectedBot >= 0 && ss.SelectedBot < len(ss.Bots) {
		bot := &ss.Bots[ss.SelectedBot]
		bx := mx + float32(bot.X*scaleX)
		by := my + float32(bot.Y*scaleY)
		vector.DrawFilledRect(screen, bx-2, by-2, 4, 4, color.RGBA{255, 255, 0, 255}, false)
	}

	// Mini legend below minimap
	ly := int(my) + minimapH + 2
	lx := int(mx)
	legendCol := color.RGBA{100, 110, 130, 180}
	printColoredAt(screen, "Minimap", lx, ly, legendCol)
	if ss.DeliveryOn {
		ly += 10
		printColoredAt(screen, "o=Pickup  O=Dropoff", lx, ly, legendCol)
	}
}
