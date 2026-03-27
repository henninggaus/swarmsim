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

	// Mode selection — 3 clickable buttons
	modeY := 250
	dimCol := color.RGBA{100, 100, 120, 255}
	btnBorderCol := color.RGBA{80, 100, 140, 150}

	// Button dimensions
	btnW := 340
	btnH := 48
	btnX := (sw - btnW) / 2

	// ---- Button 1: Swarm Lab (empfohlen, pulsing) ----
	pulse := 0.7 + 0.3*math.Sin(float64(tick)*0.05)
	glowAlpha := uint8(float64(50) * pulse)
	vector.DrawFilledRect(screen, float32(btnX), float32(modeY), float32(btnW), float32(btnH),
		color.RGBA{30, 50, 100, 200}, false)
	vector.DrawFilledRect(screen, float32(btnX), float32(modeY), float32(btnW), float32(btnH),
		color.RGBA{40, 80, 160, glowAlpha}, false)
	vector.StrokeRect(screen, float32(btnX), float32(modeY), float32(btnW), float32(btnH),
		2, color.RGBA{136, 204, 255, uint8(float64(120) * pulse)}, false)

	labTitle := "Swarm Lab — Editor (empfohlen)"
	labTitleW := len(labTitle) * charW
	printColoredAt(screen, labTitle, sw/2-labTitleW/2, modeY+6, color.RGBA{255, 255, 255, 255})
	labDesc := "IF...THEN Regeln, 20 Presets, Evolution, GP"
	labDescW := len(labDesc) * charW
	printColoredAt(screen, labDesc, sw/2-labDescW/2, modeY+22, dimCol)
	// Store hit zone ID
	r.WelcomeBtn1 = [4]int{btnX, modeY, btnW, btnH}

	// ---- Button 2: Tutorial (gruen) ----
	tutY := modeY + btnH + 10
	pulse3 := 0.7 + 0.3*math.Sin(float64(tick)*0.06+1.0)
	glowAlpha3 := uint8(float64(35) * pulse3)
	vector.DrawFilledRect(screen, float32(btnX), float32(tutY), float32(btnW), float32(btnH),
		color.RGBA{25, 50, 30, 200}, false)
	vector.DrawFilledRect(screen, float32(btnX), float32(tutY), float32(btnW), float32(btnH),
		color.RGBA{40, 100, 50, glowAlpha3}, false)
	vector.StrokeRect(screen, float32(btnX), float32(tutY), float32(btnW), float32(btnH),
		2, color.RGBA{120, 200, 80, uint8(float64(100) * pulse3)}, false)

	tutTitle := "Tutorial starten"
	tutTitleW := len(tutTitle) * charW
	printColoredAt(screen, tutTitle, sw/2-tutTitleW/2, tutY+6, color.RGBA{220, 255, 220, 255})
	tutDesc := "15 Schritte von der ersten Regel bis zur Evolution"
	tutDescW := len(tutDesc) * charW
	printColoredAt(screen, tutDesc, sw/2-tutDescW/2, tutY+22, dimCol)
	r.WelcomeBtn2 = [4]int{btnX, tutY, btnW, btnH}

	// ---- Button 3: Algo-Labor (orange accent) ----
	algoY := tutY + btnH + 10
	pulse4 := 0.7 + 0.3*math.Sin(float64(tick)*0.04+2.0)
	glowAlpha4 := uint8(float64(30) * pulse4)
	vector.DrawFilledRect(screen, float32(btnX), float32(algoY), float32(btnW), float32(btnH),
		color.RGBA{50, 35, 15, 200}, false)
	vector.DrawFilledRect(screen, float32(btnX), float32(algoY), float32(btnW), float32(btnH),
		color.RGBA{100, 70, 20, glowAlpha4}, false)
	vector.StrokeRect(screen, float32(btnX), float32(algoY), float32(btnW), float32(btnH),
		2, color.RGBA{200, 150, 50, uint8(float64(100) * pulse4)}, false)

	algoTitle := "Algo-Labor — Optimierungsalgorithmen (F4)"
	algoTitleW := len(algoTitle) * charW
	printColoredAt(screen, algoTitle, sw/2-algoTitleW/2, algoY+6, color.RGBA{255, 220, 150, 255})
	algoDesc := "20 Algorithmen vergleichen: Woelfe, Wale, Bienen & mehr"
	algoDescW := len(algoDesc) * charW
	printColoredAt(screen, algoDesc, sw/2-algoDescW/2, algoY+22, dimCol)
	r.WelcomeBtn4 = [4]int{btnX, algoY, btnW, btnH}

	// ---- Button 4: Classic Mode (dezent) ----
	classicY := algoY + btnH + 10
	vector.DrawFilledRect(screen, float32(btnX), float32(classicY), float32(btnW), 38,
		color.RGBA{30, 30, 45, 180}, false)
	vector.StrokeRect(screen, float32(btnX), float32(classicY), float32(btnW), 38,
		1, btnBorderCol, false)

	classTitle := "Classic Mode"
	classTitleW := len(classTitle) * charW
	printColoredAt(screen, classTitle, sw/2-classTitleW/2, classicY+4, color.RGBA{180, 200, 220, 255})
	classDesc := "5 Bot-Typen, Genom-Evolution, Pheromone"
	classDescW := len(classDesc) * charW
	printColoredAt(screen, classDesc, sw/2-classDescW/2, classicY+20, dimCol)
	r.WelcomeBtn3 = [4]int{btnX, classicY, btnW, 38}

	// Separator
	sepY2 := float32(classicY + 50)
	vector.StrokeLine(screen, float32(sw/2-250), sepY2, float32(sw/2+250), sepY2, 1, color.RGBA{60, 60, 80, 255}, false)

	// Feature highlights
	featureY := int(sepY2) + 12
	featureCol := color.RGBA{140, 160, 180, 255}
	highlightCol := color.RGBA{180, 220, 255, 200}

	features := []struct {
		icon string
		text string
	}{
		{"Schwarm", "20-500 autonome Bots, jeder sieht nur 120px — kein GPS!"},
		{"Script", "IF nearest_dist < 40 THEN TURN_FROM_NEAREST — so einfach"},
		{"Evolve", "GA optimiert Parameter, GP evolviert ganze Programme"},
		{"Neuro", "Neuronale Netze pro Bot — lernt durch Evolution!"},
		{"Logist", "Pakete abholen & liefern, LKW entladen, Maze loesen"},
		{"Battle", "Blau vs Rot: wessen Programm liefert mehr Pakete?"},
	}

	for i, f := range features {
		fy := featureY + i*(lineH+4)
		printColoredAt(screen, f.icon, centerX-280, fy, highlightCol)
		printColoredAt(screen, f.text, centerX-235, fy, featureCol)
	}

	// Hint text (blinking)
	hintY := featureY + len(features)*(lineH+4) + 14
	hintAlpha := uint8(120 + int(80*math.Sin(float64(tick)*0.08)))
	hint := "Klicke auf einen Button oben oder druecke F1 / F2 / F3 / F4"
	hintW := len(hint) * charW
	printColoredAt(screen, hint, sw/2-hintW/2, hintY,
		color.RGBA{180, 180, 200, hintAlpha})

	// Version
	version := "v2.0"
	printColoredAt(screen, version, sw-len(version)*charW-15, sh-20, color.RGBA{60, 60, 70, 255})
}
