package render

import (
	"image"
	"image/color"
	"math"
	"swarmsim/domain/bot"
	"swarmsim/domain/physics"
	"swarmsim/engine/pheromone"
	"swarmsim/engine/simulation"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Camera handles viewport position and zoom.
type Camera struct {
	X, Y float64
	Zoom float64
}

func NewCamera(arenaW, arenaH float64) *Camera {
	return &Camera{X: arenaW / 2, Y: arenaH / 2, Zoom: 0.7}
}

func (c *Camera) WorldToScreen(wx, wy float64, sw, sh int) (float64, float64) {
	return (wx-c.X)*c.Zoom + float64(sw)/2, (wy-c.Y)*c.Zoom + float64(sh)/2
}

func (c *Camera) ScreenToWorld(sx, sy float64, sw, sh int) (float64, float64) {
	return (sx-float64(sw)/2)/c.Zoom + c.X, (sy-float64(sh)/2)/c.Zoom + c.Y
}

const botSpriteSize = 24 // pixels for pre-rendered bot triangle sprites

// Renderer draws the simulation.
type Renderer struct {
	Camera   *Camera
	pherImg  *ebiten.Image
	pherRGBA *image.RGBA
	pherTick int

	Particles         *ParticleSystem
	SwarmParticles    *ParticleSystem // separate particle system for swarm delivery effects
	HomeBaseGlowAlpha float64

	ShowTrails  bool // toggle with T key (default off for performance)
	ShowMinimap bool // toggle with M key

	arenaImg *ebiten.Image // offscreen 800x800 for swarm arena (follow-cam support)

	botSprites [5]*ebiten.Image // pre-rendered triangle per BotType

	// Screenshot & GIF recording
	Recording      bool
	RecRawFrames   []*image.RGBA // raw frames (dithering happens after stop)
	RecFrameCount  int
	RecSkipCounter int
	RecBlinkTick   int
	OverlayText    string
	OverlayTimer   int
	GIFEncoding    bool   // true while background goroutine encodes
	GIFEncodedFile string // set by goroutine when done

	// Sound system
	Sound *SoundSystem

	// Welcome screen
	WelcomeBots []WelcomeBot

	// Fade transition
	FadeAlpha float32 // 0.0 = transparent, 1.0 = fully black
	FadeDir   int     // -1 = fading out (to black), +1 = fading in (from black), 0 = none
	FadeLoad  func()  // callback when fade reaches 1.0 (load scenario, then reverse)

	// FPS warning
	LowFPSCounter int

	// PSO fitness landscape heatmap cache
	psoLandscapeImg  *ebiten.Image
	psoLandscapeHash uint64 // hash of peak params to detect changes

	// Contour line cache (computed alongside heatmap)
	psoContourSegs []contourSegment // all contour line segments
	psoContourW    int              // grid width used when computing contours
	psoContourH    int              // grid height used when computing contours
}

// NewRenderer creates a new renderer with a particle system.
func NewRenderer(cam *Camera) *Renderer {
	r := &Renderer{
		Camera:         cam,
		Particles:      NewParticleSystem(),
		SwarmParticles: NewParticleSystem(),
		Sound:          NewSoundSystem(),
	}
	r.initBotSprites()
	return r
}

// initBotSprites pre-renders a white triangle sprite for each BotType.
// At draw time the sprite is tinted via ColorScale and rotated via GeoM.
func (r *Renderer) initBotSprites() {
	// Triangle geometry at angle=0 (pointing right), size=1 unit, centered at (half,half)
	half := float64(botSpriteSize) / 2
	// Unit triangle: apex at 1.5 forward, rear corners at ±2.5 radians
	ax := half + 1.5*half*0.75 // apex (use 0.75 scale to fit inside sprite)
	ay := half
	bx := half + half*0.75*math.Cos(2.5)
	by := half + half*0.75*math.Sin(2.5)
	cx := half + half*0.75*math.Cos(-2.5)
	cy := half + half*0.75*math.Sin(-2.5)

	for i := 0; i < 5; i++ {
		img := ebiten.NewImage(botSpriteSize, botSpriteSize)
		// Draw white triangle; color is applied at render time via ColorScale
		white := color.RGBA{255, 255, 255, 255}
		vector.StrokeLine(img, float32(ax), float32(ay), float32(bx), float32(by), 1.5, white, false)
		vector.StrokeLine(img, float32(bx), float32(by), float32(cx), float32(cy), 1.5, white, false)
		vector.StrokeLine(img, float32(cx), float32(cy), float32(ax), float32(ay), 1.5, white, false)
		r.botSprites[i] = img
	}
}

// Draw renders the entire simulation to the screen.
func (r *Renderer) Draw(screen *ebiten.Image, s *simulation.Simulation) {
	// Swarm mode: completely separate rendering path
	if s.SwarmMode && s.SwarmState != nil {
		sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
		r.DrawSwarmMode(screen, s, sw, sh)
		return
	}

	screen.Fill(ColorBackground)
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Process visual events from simulation
	for _, ev := range s.CoopPickupEvents {
		r.Particles.Emit(ev.X, ev.Y, 30, ColorCoopParticle, 2.0, 3.0, 40)
	}
	for range s.DeliveryEvents {
		r.HomeBaseGlowAlpha = 1.0
	}
	// Sound effects for standard mode events
	if r.Sound != nil && r.Sound.Enabled {
		for range s.CoopPickupEvents {
			r.Sound.PlayPickup()
		}
		for range s.DeliveryEvents {
			r.Sound.PlayDropOK()
		}
	}

	r.Particles.Update()

	r.drawGrid(screen, s, sw, sh)

	r.drawHomeBase(screen, s, sw, sh)

	r.drawObstacles(screen, s, sw, sh)

	if s.PheromoneVizMode > 0 {
		r.drawPheromones(screen, s, sw, sh)
	}

	r.drawResources(screen, s, sw, sh)

	if s.ShowSensorRadius {
		r.drawRadii(screen, s, sw, sh, false)
	}
	if s.ShowCommRadius {
		r.drawRadii(screen, s, sw, sh, true)
	}

	if r.ShowTrails {
		r.drawBotTrails(screen, s, sw, sh)
	}
	r.drawBots(screen, s, sw, sh)
	r.Particles.Draw(screen, r.Camera, sw, sh)

	if s.ShowDebugComm {
		r.drawCommLines(screen, s, sw, sh)
	}

	// Minimap (only when zoomed in)
	if r.ShowMinimap && r.Camera.Zoom > 1.0 {
		r.drawMinimap(screen, s, sw, sh)
	}
}

func (r *Renderer) drawPheromones(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	pg := s.Pheromones
	if pg == nil {
		return
	}

	if r.pherImg == nil || r.pherImg.Bounds().Dx() != pg.Cols || r.pherImg.Bounds().Dy() != pg.Rows {
		r.pherImg = ebiten.NewImage(pg.Cols, pg.Rows)
		r.pherRGBA = image.NewRGBA(image.Rect(0, 0, pg.Cols, pg.Rows))
	}

	// Update pixel data every 5 ticks for performance (decay is 0.995/tick, imperceptible)
	if s.Tick-r.pherTick >= 5 {
		r.pherTick = s.Tick

		for ri := 0; ri < pg.Rows; ri++ {
			for ci := 0; ci < pg.Cols; ci++ {
				idx := (ri*pg.Cols + ci) * 4
				var cr, cg, cb uint8
				var maxA float64

				if s.PheromoneVizMode >= 2 {
					sv := pg.GetCell(ci, ri, pheromone.PherSearch)
					if sv > 0.01 {
						cb = uint8(math.Min(255, float64(cb)+sv*200))
						if sv*100 > maxA {
							maxA = sv * 100
						}
					}
					dv := pg.GetCell(ci, ri, pheromone.PherDanger)
					if dv > 0.01 {
						cr = uint8(math.Min(255, float64(cr)+dv*220))
						if dv*120 > maxA {
							maxA = dv * 120
						}
					}
				}

				fv := pg.GetCell(ci, ri, pheromone.PherFoundResource)
				if fv > 0.01 {
					cg = uint8(math.Min(255, float64(cg)+fv*220))
					if fv*120 > maxA {
						maxA = fv * 120
					}
				}

				if maxA > 255 {
					maxA = 255
				}
				r.pherRGBA.Pix[idx] = cr
				r.pherRGBA.Pix[idx+1] = cg
				r.pherRGBA.Pix[idx+2] = cb
				r.pherRGBA.Pix[idx+3] = uint8(maxA)
			}
		}
		r.pherImg.WritePixels(r.pherRGBA.Pix)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(pg.CellSize, pg.CellSize)
	op.GeoM.Translate(-r.Camera.X, -r.Camera.Y)
	op.GeoM.Scale(r.Camera.Zoom, r.Camera.Zoom)
	op.GeoM.Translate(float64(sw)/2, float64(sh)/2)
	screen.DrawImage(r.pherImg, op)
}

func (r *Renderer) drawGrid(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	step := 100.0
	for x := 0.0; x <= s.Cfg.ArenaWidth; x += step {
		sx, sy := r.Camera.WorldToScreen(x, 0, sw, sh)
		_, ey := r.Camera.WorldToScreen(x, s.Cfg.ArenaHeight, sw, sh)
		vector.StrokeLine(screen, float32(sx), float32(sy), float32(sx), float32(ey), 1, ColorGrid, false)
	}
	for y := 0.0; y <= s.Cfg.ArenaHeight; y += step {
		sx, sy := r.Camera.WorldToScreen(0, y, sw, sh)
		ex, _ := r.Camera.WorldToScreen(s.Cfg.ArenaWidth, y, sw, sh)
		vector.StrokeLine(screen, float32(sx), float32(sy), float32(ex), float32(sy), 1, ColorGrid, false)
	}
	x0, y0 := r.Camera.WorldToScreen(0, 0, sw, sh)
	x1, y1 := r.Camera.WorldToScreen(s.Cfg.ArenaWidth, s.Cfg.ArenaHeight, sw, sh)
	vector.StrokeRect(screen, float32(x0), float32(y0), float32(x1-x0), float32(y1-y0), 2, color.RGBA{100, 100, 100, 255}, false)
}

func (r *Renderer) drawHomeBase(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	sx, sy := r.Camera.WorldToScreen(s.Arena.HomeBaseX, s.Arena.HomeBaseY, sw, sh)
	rad := s.Arena.HomeBaseR * r.Camera.Zoom
	pulse := 1.0 + 0.05*math.Sin(float64(s.Tick)*0.1)
	rad *= pulse
	vector.StrokeCircle(screen, float32(sx), float32(sy), float32(rad), 2, ColorHomeBase, false)
	c := ColorHomeBase
	c.A = 40
	vector.DrawFilledCircle(screen, float32(sx), float32(sy), float32(rad*0.8), c, false)

	// Delivery glow effect
	if r.HomeBaseGlowAlpha > 0.01 {
		glowCol := ColorDeliveryGlow
		glowCol.A = uint8(r.HomeBaseGlowAlpha * 200)
		glowRad := rad * (1.0 + 0.3*(1.0-r.HomeBaseGlowAlpha))
		vector.DrawFilledCircle(screen, float32(sx), float32(sy), float32(glowRad), glowCol, false)
		r.HomeBaseGlowAlpha *= 0.95
	}
}

func (r *Renderer) drawObstacles(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	for _, obs := range s.Arena.Obstacles {
		sx, sy := r.Camera.WorldToScreen(obs.X, obs.Y, sw, sh)
		w := obs.W * r.Camera.Zoom
		h := obs.H * r.Camera.Zoom
		vector.DrawFilledRect(screen, float32(sx), float32(sy), float32(w), float32(h), ColorObstacle, false)
	}
}

func (r *Renderer) drawResources(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	for _, res := range s.Resources {
		if !res.IsAvailable() {
			continue
		}
		sx, sy := r.Camera.WorldToScreen(res.X, res.Y, sw, sh)

		if res.Heavy {
			// Heavy resource: larger gold double-diamond
			size := 8.0 * r.Camera.Zoom
			col := ColorHeavyResource
			// Outer diamond
			vector.StrokeLine(screen, float32(sx), float32(sy-size), float32(sx+size), float32(sy), 2.0, col, false)
			vector.StrokeLine(screen, float32(sx+size), float32(sy), float32(sx), float32(sy+size), 2.0, col, false)
			vector.StrokeLine(screen, float32(sx), float32(sy+size), float32(sx-size), float32(sy), 2.0, col, false)
			vector.StrokeLine(screen, float32(sx-size), float32(sy), float32(sx), float32(sy-size), 2.0, col, false)
			// Inner diamond
			inner := size * 0.5
			vector.StrokeLine(screen, float32(sx), float32(sy-inner), float32(sx+inner), float32(sy), 1.5, col, false)
			vector.StrokeLine(screen, float32(sx+inner), float32(sy), float32(sx), float32(sy+inner), 1.5, col, false)
			vector.StrokeLine(screen, float32(sx), float32(sy+inner), float32(sx-inner), float32(sy), 1.5, col, false)
			vector.StrokeLine(screen, float32(sx-inner), float32(sy), float32(sx), float32(sy-inner), 1.5, col, false)
		} else {
			// Normal resource: green diamond
			size := 5.0 * r.Camera.Zoom
			vector.StrokeLine(screen, float32(sx), float32(sy-size), float32(sx+size), float32(sy), 1.5, ColorResource, false)
			vector.StrokeLine(screen, float32(sx+size), float32(sy), float32(sx), float32(sy+size), 1.5, ColorResource, false)
			vector.StrokeLine(screen, float32(sx), float32(sy+size), float32(sx-size), float32(sy), 1.5, ColorResource, false)
			vector.StrokeLine(screen, float32(sx-size), float32(sy), float32(sx), float32(sy-size), 1.5, ColorResource, false)
		}
	}
}

func (r *Renderer) drawBotTrails(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	for _, b := range s.Bots {
		if !b.IsAlive() {
			continue
		}
		trail := b.GetTrail()
		col := BotColor(b.Type())
		for i := 0; i < bot.TrailLen-1; i++ {
			alpha := uint8(20 + i*15)
			c := color.RGBA{col.R, col.G, col.B, alpha}
			sx1, sy1 := r.Camera.WorldToScreen(trail[i].X, trail[i].Y, sw, sh)
			sx2, sy2 := r.Camera.WorldToScreen(trail[i+1].X, trail[i+1].Y, sw, sh)
			vector.StrokeLine(screen, float32(sx1), float32(sy1), float32(sx2), float32(sy2), 1, c, false)
		}
	}
}

func (r *Renderer) drawBots(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	for _, b := range s.Bots {
		if !b.IsAlive() {
			continue
		}
		pos := b.Position()
		vel := b.Velocity()
		sx, sy := r.Camera.WorldToScreen(pos.X, pos.Y, sw, sh)
		rad := b.GetRadius() * r.Camera.Zoom
		col := BotColor(b.Type())
		energy := b.GetEnergy()

		if energy > 0 && energy < 20 {
			if (s.Tick/15)%2 == 0 {
				col.A = 100
			}
		}
		if energy <= 0 {
			col = ColorBotDisabled
		}

		angle := math.Atan2(vel.Y, vel.X)
		if vel.Len() < 0.01 {
			angle = 0
		}
		r.drawBotSprite(screen, sx, sy, rad, angle, b.Type(), col)

		drawBar(screen, float32(sx), float32(sy-rad-6), float32(rad*2), 2,
			b.Health()/b.MaxHealth(), ColorHealthBar, ColorHealthBg)
		drawBar(screen, float32(sx), float32(sy-rad-3), float32(rad*2), 2,
			energy/100.0, ColorEnergyBar, ColorEnergyBg)

		if s.SelectedBotID == b.ID() {
			vector.StrokeCircle(screen, float32(sx), float32(sy), float32(rad+4), 1.5, color.RGBA{255, 255, 255, 200}, false)
		}
	}
}

func (r *Renderer) drawRadii(screen *ebiten.Image, s *simulation.Simulation, sw, sh int, commMode bool) {
	for _, b := range s.Bots {
		if !b.IsAlive() {
			continue
		}
		pos := b.Position()
		sx, sy := r.Camera.WorldToScreen(pos.X, pos.Y, sw, sh)
		var rad float64
		var c color.RGBA
		if commMode {
			rad = b.GetCommRange() * r.Camera.Zoom
			c = ColorCommRad
		} else {
			rad = b.GetSensorRange() * r.Camera.Zoom
			c = ColorSensorRad
		}
		vector.StrokeCircle(screen, float32(sx), float32(sy), float32(rad), 1, c, false)
	}
}

func (r *Renderer) drawCommLines(screen *ebiten.Image, s *simulation.Simulation, sw, sh int) {
	if s.Hash == nil || len(s.Bots) == 0 {
		return
	}

	// Build ID→index map and find the max comm range across all bots.
	idToIdx := make(map[int]int, len(s.Bots))
	var maxCommR float64
	for i, b := range s.Bots {
		if b.IsAlive() {
			idToIdx[b.ID()] = i
			if cr := b.GetCommRange(); cr > maxCommR {
				maxCommR = cr
			}
		}
	}

	// Each bot queries with the global max comm range to ensure we find all
	// pairs where either bot's range covers the other. We only process
	// neighbors with j > i to draw each line exactly once.
	for i, a := range s.Bots {
		if !a.IsAlive() {
			continue
		}
		apos := a.Position()
		commA := a.GetCommRange()

		nearIDs := s.Hash.Query(apos.X, apos.Y, maxCommR)
		for _, nid := range nearIDs {
			j, ok := idToIdx[nid]
			if !ok || j <= i {
				continue
			}
			b := s.Bots[j]
			dist := apos.Dist(b.Position())
			if dist < commA || dist < b.GetCommRange() {
				ax, ay := r.Camera.WorldToScreen(apos.X, apos.Y, sw, sh)
				bx, by := r.Camera.WorldToScreen(b.Position().X, b.Position().Y, sw, sh)
				vector.StrokeLine(screen, float32(ax), float32(ay), float32(bx), float32(by), 1.5, ColorCommLine, false)
			}
		}
	}
}

// drawBotSprite draws a pre-rendered bot triangle sprite with rotation and color tinting.
func (r *Renderer) drawBotSprite(screen *ebiten.Image, sx, sy, rad, angle float64, btype bot.BotType, col color.RGBA) {
	sprite := r.botSprites[int(btype)%len(r.botSprites)]
	half := float64(botSpriteSize) / 2
	scale := (rad * 2) / float64(botSpriteSize) // scale sprite to match bot radius

	op := &ebiten.DrawImageOptions{}
	// Center sprite at origin, rotate, scale, then translate to screen position
	op.GeoM.Translate(-half, -half)
	op.GeoM.Rotate(angle)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(sx, sy)
	op.ColorScale.Scale(float32(col.R)/255, float32(col.G)/255, float32(col.B)/255, float32(col.A)/255)
	screen.DrawImage(sprite, op)
}

func drawTriangle(screen *ebiten.Image, cx, cy, size, angle float32, col color.RGBA) {
	a64 := float64(angle)
	s64 := float64(size)
	x0 := cx + float32(math.Cos(a64)*s64*1.5)
	y0 := cy + float32(math.Sin(a64)*s64*1.5)
	x1 := cx + float32(math.Cos(a64+2.5)*s64)
	y1 := cy + float32(math.Sin(a64+2.5)*s64)
	x2 := cx + float32(math.Cos(a64-2.5)*s64)
	y2 := cy + float32(math.Sin(a64-2.5)*s64)
	vector.StrokeLine(screen, x0, y0, x1, y1, 1.5, col, false)
	vector.StrokeLine(screen, x1, y1, x2, y2, 1.5, col, false)
	vector.StrokeLine(screen, x2, y2, x0, y0, 1.5, col, false)
}

func drawBar(screen *ebiten.Image, cx, cy, w, h float32, ratio float64, fgCol, bgCol color.RGBA) {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	x := cx - w/2
	vector.DrawFilledRect(screen, x, cy, w, h, bgCol, false)
	vector.DrawFilledRect(screen, x, cy, w*float32(ratio), h, fgCol, false)
}

// BotColor returns the display color for a bot type.
func BotColor(t bot.BotType) color.RGBA {
	switch t {
	case bot.TypeScout:
		return ColorScout
	case bot.TypeWorker:
		return ColorWorker
	case bot.TypeLeader:
		return ColorLeader
	case bot.TypeTank:
		return ColorTank
	case bot.TypeHealer:
		return ColorHealer
	}
	return color.RGBA{255, 255, 255, 255}
}

// ObstacleAt returns the obstacle at world position (wx, wy), if any.
func ObstacleAt(arena *physics.Arena, wx, wy float64) *physics.Obstacle {
	for _, obs := range arena.Obstacles {
		if wx >= obs.X && wx <= obs.X+obs.W && wy >= obs.Y && wy <= obs.Y+obs.H {
			return obs
		}
	}
	return nil
}
