package swarm

import "math"

// Mexican Wave (La Ola): Cascading wave patterns through the swarm.
// A wave front propagates based on bot position, creating stunning visual
// patterns where bots flash sequentially like a stadium wave.
// Supports multiple wave modes: linear, radial, spiral.

const (
	waveSpeed    = 4.0   // pixels per tick wave propagation speed
	wavePeriod   = 120   // ticks for one full wave cycle
	waveFlashLen = 20    // ticks a bot stays "flashed"
)

// WaveMode determines wave propagation pattern.
type WaveMode int

const (
	WaveLinear WaveMode = iota // left-to-right sweep
	WaveRadial                 // expanding from center
	WaveSpiral                 // rotating spiral
	WaveModeCount
)

// WaveState holds global wave state.
type WaveState struct {
	Mode      WaveMode
	Phase     float64 // current wave phase (0 to 2π)
	Speed     float64 // phase advance per tick
	CenterX   float64
	CenterY   float64
	FlashTick []int // tick when this bot last flashed (0 = never)
}

// InitWave allocates wave state.
func InitWave(ss *SwarmState) {
	n := len(ss.Bots)
	ss.Wave = &WaveState{
		Mode:    WaveLinear,
		Phase:   0,
		Speed:   2 * math.Pi / float64(wavePeriod),
		CenterX: ss.ArenaW / 2,
		CenterY: ss.ArenaH / 2,
		FlashTick: make([]int, n),
	}
	ss.WaveOn = true
}

// ClearWave frees wave state.
func ClearWave(ss *SwarmState) {
	ss.Wave = nil
	ss.WaveOn = false
}

// CycleWaveMode advances to the next wave mode.
func CycleWaveMode(ss *SwarmState) {
	if ss.Wave == nil {
		return
	}
	ss.Wave.Mode = WaveMode((int(ss.Wave.Mode) + 1) % int(WaveModeCount))
}

// TickWave advances the wave and triggers flashes.
func TickWave(ss *SwarmState) {
	if ss.Wave == nil {
		return
	}
	st := ss.Wave

	// Advance phase
	st.Phase += st.Speed
	if st.Phase > 2*math.Pi {
		st.Phase -= 2 * math.Pi
	}

	// Grow slices
	for len(st.FlashTick) < len(ss.Bots) {
		st.FlashTick = append(st.FlashTick, 0)
	}

	// Compute wave position for each bot
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		var botPhase float64

		switch st.Mode {
		case WaveLinear:
			// Phase based on X position
			botPhase = (bot.X / ss.ArenaW) * 2 * math.Pi

		case WaveRadial:
			// Phase based on distance from center
			dx := bot.X - st.CenterX
			dy := bot.Y - st.CenterY
			dist := math.Sqrt(dx*dx + dy*dy)
			maxDist := math.Sqrt(st.CenterX*st.CenterX + st.CenterY*st.CenterY)
			botPhase = (dist / maxDist) * 2 * math.Pi

		case WaveSpiral:
			// Phase based on angle + distance from center
			dx := bot.X - st.CenterX
			dy := bot.Y - st.CenterY
			angle := math.Atan2(dy, dx)
			if angle < 0 {
				angle += 2 * math.Pi
			}
			dist := math.Sqrt(dx*dx + dy*dy)
			maxDist := math.Sqrt(st.CenterX*st.CenterX + st.CenterY*st.CenterY)
			botPhase = angle + (dist/maxDist)*2*math.Pi
		}

		// Check if wave front is passing this bot
		phaseDiff := st.Phase - botPhase
		phaseDiff = WrapAngle(phaseDiff)

		// Flash when phase difference is small (wave front nearby)
		if math.Abs(phaseDiff) < 0.3 {
			st.FlashTick[i] = ss.Tick
		}

		// Update sensor cache
		ticksSinceFlash := ss.Tick - st.FlashTick[i]
		if ticksSinceFlash < waveFlashLen {
			bot.WaveFlash = 1
			bot.WavePhase = int(math.Abs(phaseDiff) * 100)
		} else {
			bot.WaveFlash = 0
			bot.WavePhase = int(math.Min(100, math.Abs(phaseDiff)*100))
		}
	}
}

// ApplyWaveFlash sets the bot's LED based on its wave state.
func ApplyWaveFlash(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Wave == nil {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Wave

	if idx >= len(st.FlashTick) {
		bot.Speed = SwarmBotSpeed
		return
	}

	ticksSinceFlash := ss.Tick - st.FlashTick[idx]
	if ticksSinceFlash < waveFlashLen {
		// Flashing — bright color that fades
		intensity := uint8(255 * (1.0 - float64(ticksSinceFlash)/float64(waveFlashLen)))
		switch st.Mode {
		case WaveLinear:
			bot.LEDColor = [3]uint8{intensity, intensity, 0} // yellow wave
		case WaveRadial:
			bot.LEDColor = [3]uint8{0, intensity, intensity} // cyan wave
		case WaveSpiral:
			bot.LEDColor = [3]uint8{intensity, 0, intensity} // magenta wave
		}
	} else {
		bot.LEDColor = [3]uint8{10, 10, 30} // dim background
	}
	bot.Speed = SwarmBotSpeed
}
