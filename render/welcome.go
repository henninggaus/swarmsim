package render

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// WelcomeBot is a decorative bot that bounces around the welcome screen.
type WelcomeBot struct {
	X, Y   float64
	Angle  float64
	Speed  float64
	Color  color.RGBA
	Size   float64
	TurnCD int // ticks until next random turn
}

const welcomeBotCount = 10

// InitWelcomeBots creates decorative bots for the welcome screen background.
func (r *Renderer) InitWelcomeBots(sw, sh int) {
	rng := rand.New(rand.NewSource(42))
	colors := []color.RGBA{
		ColorScout, ColorWorker, ColorLeader, ColorTank, ColorHealer,
		{100, 200, 255, 120}, {255, 150, 80, 120}, {80, 255, 150, 120},
	}
	r.WelcomeBots = make([]WelcomeBot, welcomeBotCount)
	for i := range r.WelcomeBots {
		r.WelcomeBots[i] = WelcomeBot{
			X:     50 + rng.Float64()*float64(sw-100),
			Y:     50 + rng.Float64()*float64(sh-100),
			Angle: rng.Float64() * 2 * math.Pi,
			Speed: 0.5 + rng.Float64()*0.8,
			Color: colors[i%len(colors)],
			Size:  6 + rng.Float64()*4,
		}
		// Lower alpha for background effect
		r.WelcomeBots[i].Color.A = 80
	}
}

// UpdateWelcomeBots moves decorative bots each tick.
func (r *Renderer) UpdateWelcomeBots(sw, sh int) {
	for i := range r.WelcomeBots {
		b := &r.WelcomeBots[i]
		b.X += math.Cos(b.Angle) * b.Speed
		b.Y += math.Sin(b.Angle) * b.Speed

		// Bounce off screen edges
		margin := 20.0
		if b.X < margin {
			b.X = margin
			b.Angle = math.Pi - b.Angle
		}
		if b.X > float64(sw)-margin {
			b.X = float64(sw) - margin
			b.Angle = math.Pi - b.Angle
		}
		if b.Y < margin {
			b.Y = margin
			b.Angle = -b.Angle
		}
		if b.Y > float64(sh)-margin {
			b.Y = float64(sh) - margin
			b.Angle = -b.Angle
		}

		// Random turn
		b.TurnCD--
		if b.TurnCD <= 0 {
			b.Angle += (rand.Float64() - 0.5) * 0.6
			b.TurnCD = 30 + rand.Intn(60)
		}
	}
}

// DrawWelcomeScreen renders the full welcome/start screen.
func (r *Renderer) DrawWelcomeScreen(screen *ebiten.Image, tick int) {
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	// Dark background
	screen.Fill(color.RGBA{12, 12, 22, 255})

	// Animated background bots
	for _, b := range r.WelcomeBots {
		drawTriangle(screen, float32(b.X), float32(b.Y), float32(b.Size), float32(b.Angle), b.Color)
		// Subtle trail dot
		trailAlpha := uint8(30)
		tx := b.X - math.Cos(b.Angle)*b.Size*2
		ty := b.Y - math.Sin(b.Angle)*b.Size*2
		vector.DrawFilledCircle(screen, float32(tx), float32(ty), 2, color.RGBA{b.Color.R, b.Color.G, b.Color.B, trailAlpha}, false)
	}

	// --- Centered content ---
	centerX := sw / 2
	_ = centerX

	// Title: "SwarmSim" (large, cyan)
	title := "SwarmSim"
	titleImg := cachedTextImage(title)
	titleW := titleImg.Bounds().Dx()
	titleScale := 3.0
	titleTotalW := float64(titleW) * titleScale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(titleScale, titleScale)
	op.GeoM.Translate(float64(sw)/2-titleTotalW/2, 140)
	op.ColorScale.Scale(136.0/255, 204.0/255, 1.0, 1.0) // #88ccff
	screen.DrawImage(titleImg, op)

	// Subtitle
	subtitle := "Schwarm-Robotik-Simulator"
	subImg := cachedTextImage(subtitle)
	subW := subImg.Bounds().Dx()
	subScale := 1.5
	subTotalW := float64(subW) * subScale
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Scale(subScale, subScale)
	op2.GeoM.Translate(float64(sw)/2-subTotalW/2, 195)
	op2.ColorScale.Scale(0.5, 0.5, 0.7, 1.0)
	screen.DrawImage(subImg, op2)

	// Tagline
	tagline := "Emergentes Verhalten durch einfache Regeln"
	tagW := len(tagline) * charW
	printColoredAt(screen, tagline, sw/2-tagW/2, 230, color.RGBA{100, 100, 130, 255})

	// Separator line
	sepY := float32(260)
	vector.StrokeLine(screen, float32(sw/2-200), sepY, float32(sw/2+200), sepY, 1, color.RGBA{60, 60, 80, 255}, false)

	// Scenario list
	scenarioY := 285
	scenarioCol := color.RGBA{180, 200, 220, 255}
	keyCol := color.RGBA{136, 204, 255, 255} // cyan for keys
	dimCol := color.RGBA{100, 100, 120, 255}

	scenarios := []struct {
		key  string
		name string
		desc string
	}{
		{"F1", "Basis-Schwarm", "Einfache Bots mit Sensoren und Energie"},
		{"F2", "Kooperatives Sammeln", "Bots sammeln Ressourcen kooperativ"},
		{"F3", "Nachrichtenbasiert", "Kommunikation zwischen Bots"},
		{"F4", "Wellen-Modus", "Wellen von Ressourcen einsammeln"},
		{"F5", "Genetik-Evolution", "Evolutionaere Optimierung der Bots"},
		{"F6", "LKW-Entladung", "Pakete sortieren und entladen"},
	}

	// Draw scenarios in two columns
	colW := 300
	col1X := sw/2 - colW - 20
	col2X := sw/2 + 20

	for i, sc := range scenarios {
		x := col1X
		y := scenarioY + (i/2)*28
		if i%2 == 1 {
			x = col2X
		}
		if i >= 2 && i%2 == 0 {
			y = scenarioY + (i/2)*28
		}

		printColoredAt(screen, "["+sc.key+"]", x, y, keyCol)
		printColoredAt(screen, sc.name, x+30, y, scenarioCol)
	}

	// F7 highlight (default option)
	f7Y := scenarioY + 3*28 + 10

	// Highlight box behind F7
	f7Text := "[F7] Programmable Swarm — SwarmScript Editor"
	f7W := len(f7Text) * charW
	f7X := sw/2 - f7W/2

	// Pulsing glow for F7
	pulse := 0.7 + 0.3*math.Sin(float64(tick)*0.05)
	glowAlpha := uint8(float64(40) * pulse)
	vector.DrawFilledRect(screen, float32(f7X-8), float32(f7Y-4), float32(f7W+16), 24,
		color.RGBA{40, 80, 140, glowAlpha}, false)
	vector.StrokeRect(screen, float32(f7X-8), float32(f7Y-4), float32(f7W+16), 24,
		1, color.RGBA{136, 204, 255, uint8(float64(100) * pulse)}, false)

	printColoredAt(screen, "[F7]", f7X, f7Y, keyCol)
	printColoredAt(screen, "Programmable Swarm", f7X+30, f7Y, color.RGBA{255, 255, 255, 255})
	printColoredAt(screen, "— SwarmScript Editor", f7X+30+19*charW, f7Y, dimCol)

	// Bottom separator
	sepY2 := float32(f7Y + 45)
	vector.StrokeLine(screen, float32(sw/2-200), sepY2, float32(sw/2+200), sepY2, 1, color.RGBA{60, 60, 80, 255}, false)

	// Hint text (blinking)
	hintAlpha := uint8(120 + int(80*math.Sin(float64(tick)*0.08)))
	hint := "Druecke eine Taste oder klicke um zu starten"
	hintW := len(hint) * charW
	printColoredAt(screen, hint, sw/2-hintW/2, int(sepY2)+25,
		color.RGBA{180, 180, 200, hintAlpha})

	// Keyboard shortcuts hint
	shortcutHint := "H = Hilfe  |  ESC = Beenden"
	shW := len(shortcutHint) * charW
	printColoredAt(screen, shortcutHint, sw/2-shW/2, sh-50, dimCol)

	// Version
	version := "v1.0"
	printColoredAt(screen, version, sw-len(version)*charW-15, sh-20, color.RGBA{60, 60, 70, 255})
}
