package render

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TutorialStep defines one step of the interactive tutorial.
type TutorialStep struct {
	Lines     [3]string // up to 3 lines of text
	ArrowX    int       // arrow target X (0 = no arrow)
	ArrowY    int       // arrow target Y
	WaitInput string    // "", "key:F2", "click:deploy", "click:bot", "key:F", "key:H", "click:delivery", "timer:300", "click:deploy_any", "click:blocks"
}

// TutorialState tracks current tutorial progress.
type TutorialState struct {
	Active      bool
	Step        int
	WaitTimer   int  // countdown for timed steps
	InputDone   bool // current step's input requirement met
	PulseTimer  int  // for arrow pulsing animation
	SkipHeld    int  // how long ESC has been held
	Dismissed   bool // user skipped or finished
}

// TutorialSteps defines all 15 tutorial steps.
var TutorialSteps = []TutorialStep{
	// Step 0: Welcome
	{
		Lines:     [3]string{"Willkommen bei SwarmSim! Ein Schwarm-Robotik-Simulator.", "Du programmierst autonome Bots mit einfachen Regeln —", "ohne zentrale Steuerung entstehen komplexe Muster."},
		WaitInput: "",
	},
	// Step 1: Modes
	{
		Lines:     [3]string{"F1 = Classic Mode (5 Bot-Typen, Pheromone, Genom-Evolution)", "F2 = Swarm Lab (SwarmScript Editor — hier wird's spannend!)", "Druecke F2 um das Swarm Lab zu oeffnen."},
		WaitInput: "key:F2",
	},
	// Step 2: Editor
	{
		Lines:     [3]string{"Links siehst du den SwarmScript Editor.", "Jede Zeile ist eine Regel: IF <Sensor> <Op> <Wert> THEN <Aktion>.", "Die erste passende Regel bestimmt was der Bot tut."},
		ArrowX:    175, ArrowY: 300,
		WaitInput: "",
	},
	// Step 3: First program
	{
		Lines:     [3]string{"Waehle 'Aggregation' im Dropdown und druecke DEPLOY.", "Aggregation: Bots drehen sich zum naechsten Nachbarn", "und finden sich zu Clustern zusammen (soziale Anziehung)."},
		ArrowX:    100, ArrowY: 36,
		WaitInput: "click:deploy_any",
	},
	// Step 4: Observe
	{
		Lines:     [3]string{"Beobachte: Jeder Bot sieht nur 120px weit — keine globale Karte!", "Trotzdem bilden sich Gruppen. Das ist emergentes Verhalten:", "komplexe Muster aus einfachen lokalen Regeln."},
		WaitInput: "timer:300",
	},
	// Step 5: Switch program
	{
		Lines:     [3]string{"Jetzt waehle 'Dispersion' und druecke DEPLOY.", "Das Gegenteil: Bots stossen sich ab und verteilen sich", "gleichmaessig — wie Gas-Molekuele in einem Raum."},
		ArrowX:    100, ArrowY: 36,
		WaitInput: "click:deploy_any",
	},
	// Step 6: Delivery on
	{
		Lines:     [3]string{"Klicke auf 'Delivery: ON'. Farbige Stationen erscheinen:", "Gefuellte Kreise = Pickup (Paket abholen),", "Ringe = Dropoff (Paket zur gleichen Farbe bringen)."},
		ArrowX:    88, ArrowY: 725,
		WaitInput: "click:delivery",
	},
	// Step 7: Delivery program
	{
		Lines:     [3]string{"Waehle 'Simple Delivery' und druecke DEPLOY.", "Bots sammeln Pakete ein (!) und liefern zur passenden", "Dropoff. Beobachte wie der LED-Gradient als Wegweiser wirkt!"},
		ArrowX:    100, ArrowY: 36,
		WaitInput: "click:deploy_any",
	},
	// Step 8: Stats
	{
		Lines:     [3]string{"Oben: Deliveries (geliefert), Correct (richtige Farbe), Wrong.", "AvgTime = durchschnittliche Lieferzeit in Ticks.", "Druecke D fuer das Dashboard mit Graphen und Heatmap."},
		ArrowX:    600, ArrowY: 15,
		WaitInput: "",
	},
	// Step 9: Bot select
	{
		Lines:     [3]string{"Klicke auf einen Bot in der Arena!", "Du siehst sein Info-Panel: aktuelle Sensoren, Zustand,", "ob er ein Paket traegt und seine Lieferstatistiken."},
		WaitInput: "click:bot",
	},
	// Step 10: Follow-cam
	{
		Lines:     [3]string{"Druecke F um dem Bot zu folgen. Die Kamera zentriert sich", "auf ihn. So siehst du die Welt aus Bot-Perspektive.", "Nochmal F oder Klick ins Leere zum Beenden."},
		WaitInput: "key:F",
	},
	// Step 11: Block editor
	{
		Lines:     [3]string{"Klicke auf BLOCKS oben fuer den visuellen Editor.", "Statt Text zu tippen, waehlst du Sensoren, Operatoren", "und Aktionen aus Dropdown-Menues — ideal zum Experimentieren!"},
		ArrowX:    288, ArrowY: 10,
		WaitInput: "click:blocks",
	},
	// Step 12: Toggles
	{
		Lines:     [3]string{"Unten links: Arena-Konfiguration. Obstacles = Hindernisse,", "Maze = Labyrinth, Evolution = Param-Optimierung,", "GP = Programme evolvieren sich selbst! Hover fuer Details."},
		ArrowX:    175, ArrowY: 710,
		WaitInput: "",
	},
	// Step 13: Help
	{
		Lines:     [3]string{"Druecke H fuer die komplette Hilfe mit allen Sensoren,", "Aktionen und Algorithmus-Erklaerungen.", "Log-Konsole: ` oder Oe-Taste."},
		WaitInput: "key:H",
	},
	// Step 14: Finish
	{
		Lines:     [3]string{"Fertig! Tipps: 'Evolving Delivery'+Evolution ON optimiert Parameter.", "GP: Random Start+GP ON = Programme evolvieren sich selbst!", "Neuro: Delivery = Neuronale Netze lernen ohne Code! Viel Spass!"},
		WaitInput: "",
	},
}

var (
	colorTutBg     = color.RGBA{0, 0, 0, 200}
	colorTutBox    = color.RGBA{20, 25, 40, 240}
	colorTutBorder = color.RGBA{80, 160, 255, 255}
	colorTutText   = color.RGBA{220, 225, 240, 255}
	colorTutDim    = color.RGBA{140, 145, 160, 255}
	colorTutBtn    = color.RGBA{50, 120, 220, 255}
	colorTutBtnH   = color.RGBA{70, 150, 255, 255}
	colorTutArrow  = color.RGBA{255, 180, 50, 255}
	colorTutStep   = color.RGBA{80, 160, 255, 200}
)

// DrawTutorial renders the tutorial overlay.
func DrawTutorial(screen *ebiten.Image, tut *TutorialState, tick int) {
	if !tut.Active || tut.Step < 0 || tut.Step >= len(TutorialSteps) {
		return
	}

	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()
	step := TutorialSteps[tut.Step]

	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), colorTutBg, false)

	// Text box at bottom center
	boxW := 700
	boxH := 110
	boxX := (sw - boxW) / 2
	boxY := sh - boxH - 40

	// Box background with border
	vector.DrawFilledRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), colorTutBox, false)
	vector.StrokeRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), 2, colorTutBorder, false)

	// Step indicator
	stepText := fmt.Sprintf("Schritt %d / %d", tut.Step+1, len(TutorialSteps))
	printColoredAt(screen, stepText, boxX+10, boxY+6, colorTutStep)

	// Text lines
	for i, line := range step.Lines {
		if line != "" {
			printColoredAt(screen, line, boxX+15, boxY+24+i*lineH, colorTutText)
		}
	}

	// Arrow pointing to target
	if step.ArrowX > 0 && step.ArrowY > 0 {
		drawTutorialArrow(screen, step.ArrowX, step.ArrowY, boxX+boxW/2, boxY, tut.PulseTimer)
	}

	// Buttons
	btnY := boxY + boxH - 28

	// "Weiter" button (only if no specific input is waited for, or input is done)
	needsInput := step.WaitInput != ""
	if !needsInput || tut.InputDone {
		btnW := 80
		btnX := boxX + boxW - btnW - 10
		hovered := false
		mx, my := ebiten.CursorPosition()
		if mx >= btnX && mx < btnX+btnW && my >= btnY && my < btnY+22 {
			hovered = true
		}
		btnColor := colorTutBtn
		if hovered {
			btnColor = colorTutBtnH
		}
		vector.DrawFilledRect(screen, float32(btnX), float32(btnY), float32(btnW), 22, btnColor, false)
		printColoredAt(screen, "Weiter >", btnX+10, btnY+4, colorTutText)
	} else {
		// Show hint about what input is expected
		hint := tutorialInputHint(step.WaitInput)
		if hint != "" {
			printColoredAt(screen, hint, boxX+boxW-len(hint)*charW-15, btnY+4, colorTutDim)
		}
	}

	// "Ueberspringen" button (always visible)
	skipW := 100
	skipX := boxX + 10
	vector.DrawFilledRect(screen, float32(skipX), float32(btnY), float32(skipW), 22, color.RGBA{60, 30, 30, 200}, false)
	printColoredAt(screen, "Ueberspringen", skipX+5, btnY+4, colorTutDim)
}

func drawTutorialArrow(screen *ebiten.Image, tx, ty, bx, by, pulse int) {
	// Pulsing factor
	scale := 1.0 + 0.15*math.Sin(float64(pulse)*0.1)
	arrowLen := 30.0 * scale

	// Direction from box center to target
	dx := float64(tx - bx)
	dy := float64(ty - by)
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 10 {
		return
	}

	// Arrow tip at target, tail toward box
	nx := dx / dist
	ny := dy / dist

	tipX := float64(tx)
	tipY := float64(ty)
	tailX := tipX - nx*arrowLen
	tailY := tipY - ny*arrowLen

	// Draw line
	vector.StrokeLine(screen, float32(tailX), float32(tailY), float32(tipX), float32(tipY), 3, colorTutArrow, false)

	// Arrowhead
	perpX := -ny * 8
	perpY := nx * 8
	headBackX := tipX - nx*12
	headBackY := tipY - ny*12

	// Triangle arrowhead
	vector.StrokeLine(screen, float32(tipX), float32(tipY), float32(headBackX+perpX), float32(headBackY+perpY), 3, colorTutArrow, false)
	vector.StrokeLine(screen, float32(tipX), float32(tipY), float32(headBackX-perpX), float32(headBackY-perpY), 3, colorTutArrow, false)

	// Pulsing circle at target
	r := float32(8 + 4*math.Sin(float64(pulse)*0.15))
	vector.StrokeCircle(screen, float32(tx), float32(ty), r, 2, colorTutArrow, false)
}

func tutorialInputHint(waitInput string) string {
	switch waitInput {
	case "key:F2":
		return "[Druecke F2]"
	case "key:F":
		return "[Druecke F]"
	case "key:H":
		return "[Druecke H]"
	case "click:deploy_any":
		return "[Waehle Preset + DEPLOY]"
	case "click:delivery":
		return "[Klicke Delivery]"
	case "click:bot":
		return "[Klicke einen Bot]"
	case "click:blocks":
		return "[Klicke BLOCKS]"
	}
	if len(waitInput) > 6 && waitInput[:6] == "timer:" {
		return "[Warte...]"
	}
	return ""
}

// TutorialWeiterHitTest checks if "Weiter" or "Ueberspringen" was clicked.
// Returns "weiter", "skip", or "".
func TutorialWeiterHitTest(mx, my, sw, sh int, tut *TutorialState) string {
	if !tut.Active || tut.Step < 0 || tut.Step >= len(TutorialSteps) {
		return ""
	}
	step := TutorialSteps[tut.Step]

	boxW := 700
	boxH := 110
	boxX := (sw - boxW) / 2
	boxY := sh - boxH - 40
	btnY := boxY + boxH - 28

	// Skip button
	if mx >= boxX+10 && mx < boxX+10+100 && my >= btnY && my < btnY+22 {
		return "skip"
	}

	// Weiter button (only if no input required or input done)
	needsInput := step.WaitInput != ""
	if !needsInput || tut.InputDone {
		btnW := 80
		btnX := boxX + boxW - btnW - 10
		if mx >= btnX && mx < btnX+btnW && my >= btnY && my < btnY+22 {
			return "weiter"
		}
	}

	return ""
}
