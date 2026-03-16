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

	// Title: "SwarmSim" (large, cyan)
	title := "SwarmSim"
	titleImg := cachedTextImage(title)
	titleW := titleImg.Bounds().Dx()
	titleScale := 3.0
	titleTotalW := float64(titleW) * titleScale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(titleScale, titleScale)
	op.GeoM.Translate(float64(sw)/2-titleTotalW/2, 100)
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
	op2.GeoM.Translate(float64(sw)/2-subTotalW/2, 155)
	op2.ColorScale.Scale(0.5, 0.5, 0.7, 1.0)
	screen.DrawImage(subImg, op2)

	// Tagline
	tagline := "Emergentes Verhalten durch einfache Regeln"
	tagW := len(tagline) * charW
	printColoredAt(screen, tagline, sw/2-tagW/2, 190, color.RGBA{100, 100, 130, 255})

	// Description
	desc := "Programmiere autonome Bots mit lokalen Sensoren — kein Masterplan, nur Schwarm-Intelligenz."
	descW := len(desc) * charW
	printColoredAt(screen, desc, sw/2-descW/2, 210, color.RGBA{80, 80, 100, 255})

	// Separator line
	sepY := float32(235)
	vector.StrokeLine(screen, float32(sw/2-250), sepY, float32(sw/2+250), sepY, 1, color.RGBA{60, 60, 80, 255}, false)

	// Mode selection (2 modes)
	modeY := 255
	keyCol := color.RGBA{136, 204, 255, 255} // cyan for keys
	dimCol := color.RGBA{100, 100, 120, 255}
	scenarioCol := color.RGBA{180, 200, 220, 255}

	// [F1] Classic Mode
	f1X := centerX - 200
	printColoredAt(screen, "[F1]", f1X, modeY, keyCol)
	printColoredAt(screen, "Classic Mode", f1X+30, modeY, scenarioCol)
	printColoredAt(screen, "5 Bot-Typen mit Genom-Evolution und Pheromonen.", f1X+30, modeY+lineH+2, dimCol)
	printColoredAt(screen, "Ideal fuer Schwarm-Beobachtung ohne Programmieren.", f1X+30, modeY+2*lineH+2, dimCol)

	// [F2] Swarm Lab — highlighted as recommended
	f2Y := modeY + 60
	f2Text := "[F2] Swarm Lab — SwarmScript Editor (empfohlen)"
	f2W := len(f2Text) * charW
	f2X := sw/2 - f2W/2

	// Pulsing glow for F2 (recommended default)
	pulse := 0.7 + 0.3*math.Sin(float64(tick)*0.05)
	glowAlpha := uint8(float64(40) * pulse)
	vector.DrawFilledRect(screen, float32(f2X-8), float32(f2Y-4), float32(f2W+16), 42,
		color.RGBA{40, 80, 140, glowAlpha}, false)
	vector.StrokeRect(screen, float32(f2X-8), float32(f2Y-4), float32(f2W+16), 42,
		1, color.RGBA{136, 204, 255, uint8(float64(100) * pulse)}, false)

	printColoredAt(screen, "[F2]", f2X, f2Y, keyCol)
	printColoredAt(screen, "Swarm Lab", f2X+30, f2Y, color.RGBA{255, 255, 255, 255})
	printColoredAt(screen, "— SwarmScript Editor (empfohlen)", f2X+30+10*charW, f2Y, dimCol)
	printColoredAt(screen, "Schreibe eigene IF...THEN Regeln oder lade 20 Presets.", f2X+30, f2Y+lineH+2, dimCol)
	printColoredAt(screen, "GP, Evolution, Delivery-Logistik, Teams. Am meisten Spass!", f2X+30, f2Y+2*lineH+2, dimCol)

	// Separator
	sepY2 := float32(f2Y + 60)
	vector.StrokeLine(screen, float32(sw/2-250), sepY2, float32(sw/2+250), sepY2, 1, color.RGBA{60, 60, 80, 255}, false)

	// Feature highlights
	featureY := int(sepY2) + 15
	featureCol := color.RGBA{140, 160, 180, 255}
	highlightCol := color.RGBA{180, 220, 255, 200}

	features := []struct {
		icon string
		text string
	}{
		{"Schwarm", "50-500 autonome Bots, jeder sieht nur 120px — kein GPS!"},
		{"Script", "IF nearest_dist < 40 THEN TURN_FROM_NEAREST — so einfach"},
		{"Evolve", "GA optimiert Parameter, GP evolviert ganze Programme"},
		{"Neuro", "Neuronale Netze pro Bot — lernt durch Evolution!"},
		{"Logist", "Pakete abholen & liefern, LKW entladen, Maze loesen"},
		{"Battle", "Blau vs Rot: wessen Programm liefert mehr Pakete?"},
	}

	for i, f := range features {
		fy := featureY + i*(lineH+6)
		printColoredAt(screen, f.icon, centerX-280, fy, highlightCol)
		printColoredAt(screen, f.text, centerX-235, fy, featureCol)
	}

	// Hint text (blinking)
	hintY := featureY + len(features)*(lineH+6) + 20
	hintAlpha := uint8(120 + int(80*math.Sin(float64(tick)*0.08)))
	hint := "Druecke F2 fuer Swarm Lab oder F1 fuer Classic Mode"
	hintW := len(hint) * charW
	printColoredAt(screen, hint, sw/2-hintW/2, hintY,
		color.RGBA{180, 180, 200, hintAlpha})

	// Keyboard shortcuts hint
	shortcutHint := "H = Hilfe  |  F3 = Tutorial  |  ESC = Beenden"
	shW := len(shortcutHint) * charW
	printColoredAt(screen, shortcutHint, sw/2-shW/2, sh-50, dimCol)

	// Version
	version := "v1.1"
	printColoredAt(screen, version, sw-len(version)*charW-15, sh-20, color.RGBA{60, 60, 70, 255})
}
