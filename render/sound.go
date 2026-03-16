package render

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	soundSampleRate = 44100
	soundBPS        = 4 // bytes per sample: 2 bytes × 2 channels (16-bit stereo)
)

// SoundSystem handles procedural audio for the simulation.
type SoundSystem struct {
	ctx          *audio.Context
	Enabled      bool
	pickupBuf    []byte // 800Hz sine, 50ms, fade-out
	dropOKBuf    []byte // 400Hz→600Hz two-tone
	dropFailBuf  []byte // 400Hz→200Hz descending sweep
	collisionBuf  []byte // white noise click, 5ms
	evolutionBuf  []byte // low gong for new generation
	broadcastBuf  []byte // short blip for message broadcast
	deployBuf     []byte // rising chime on deploy
	resetBuf      []byte // descending sweep on reset

	ambientPlayer *audio.Player
	ambientStream *ambientNoise

	collisionCooldown  int // throttle collision clicks
	broadcastCooldown  int // throttle broadcast blips
}

// NewSoundSystem creates a sound system with pre-generated audio buffers.
func NewSoundSystem() *SoundSystem {
	ctx := audio.NewContext(soundSampleRate)
	ss := &SoundSystem{
		ctx:     ctx,
		Enabled: false,
	}
	ss.pickupBuf = generateSine(800, 0.05, 0.3)
	ss.dropOKBuf = generateTwoTone(400, 600, 0.03, 0.03, 0.3)
	ss.dropFailBuf = generateSweep(400, 200, 0.08, 0.3)
	ss.collisionBuf = generateNoise(0.005, 0.15)
	ss.evolutionBuf = generateTwoTone(200, 300, 0.06, 0.08, 0.25) // low ascending gong
	ss.broadcastBuf = generateSine(1200, 0.02, 0.15)               // short high blip
	ss.deployBuf = generateTwoTone(500, 800, 0.04, 0.06, 0.25)    // rising chime
	ss.resetBuf = generateSweep(600, 200, 0.12, 0.2)              // descending sweep

	ss.ambientStream = &ambientNoise{
		sampleRate: soundSampleRate,
		rng:        rand.New(rand.NewSource(42)),
	}

	return ss
}

// PlayPickup plays the pickup sound (high sine tone).
func (ss *SoundSystem) PlayPickup() {
	ss.playBuf(ss.pickupBuf)
}

// PlayDropOK plays the correct delivery sound (ascending two-tone).
func (ss *SoundSystem) PlayDropOK() {
	ss.playBuf(ss.dropOKBuf)
}

// PlayDropFail plays the wrong delivery sound (descending sweep).
func (ss *SoundSystem) PlayDropFail() {
	ss.playBuf(ss.dropFailBuf)
}

// PlayCollision plays a short noise click (throttled to once per 5 frames).
func (ss *SoundSystem) PlayCollision() {
	if ss.collisionCooldown > 0 {
		ss.collisionCooldown--
		return
	}
	ss.collisionCooldown = 5
	ss.playBuf(ss.collisionBuf)
}

// PlayEvolution plays the generation-complete gong.
func (ss *SoundSystem) PlayEvolution() {
	ss.playBuf(ss.evolutionBuf)
}

// PlayBroadcast plays a short blip for message broadcast (throttled).
func (ss *SoundSystem) PlayBroadcast() {
	if ss.broadcastCooldown > 0 {
		ss.broadcastCooldown--
		return
	}
	ss.broadcastCooldown = 10
	ss.playBuf(ss.broadcastBuf)
}

// PlayDeploy plays a rising chime on program deploy.
func (ss *SoundSystem) PlayDeploy() {
	ss.playBuf(ss.deployBuf)
}

// PlayReset plays a descending sweep on simulation reset.
func (ss *SoundSystem) PlayReset() {
	ss.playBuf(ss.resetBuf)
}

func (ss *SoundSystem) playBuf(buf []byte) {
	if ss.ctx == nil || len(buf) == 0 {
		return
	}
	player, err := ss.ctx.NewPlayer(bytes.NewReader(buf))
	if err != nil {
		return
	}
	player.Play()
}

// SetBotCount updates the ambient noise volume based on visible bot count.
func (ss *SoundSystem) SetBotCount(n int) {
	if ss.ambientStream != nil {
		ss.ambientStream.botCount = n
	}
}

// StartAmbient begins playing continuous ambient noise.
func (ss *SoundSystem) StartAmbient() {
	if ss.ambientPlayer != nil {
		ss.ambientPlayer.Close()
		ss.ambientPlayer = nil
	}
	ss.ambientStream.pos = 0
	player, err := ss.ctx.NewPlayer(ss.ambientStream)
	if err != nil {
		fmt.Println("[SOUND] Error creating ambient player:", err)
		return
	}
	ss.ambientPlayer = player
	ss.ambientPlayer.Play()
}

// StopAmbient stops the ambient noise.
func (ss *SoundSystem) StopAmbient() {
	if ss.ambientPlayer != nil {
		ss.ambientPlayer.Close()
		ss.ambientPlayer = nil
	}
}

// --- Sound generation helpers ---

// generateSine creates a sine wave with linear fade-out envelope.
func generateSine(freq, duration, volume float64) []byte {
	numSamples := int(float64(soundSampleRate) * duration)
	buf := make([]byte, numSamples*soundBPS)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(soundSampleRate)
		envelope := 1.0 - t/duration // linear fade-out
		sample := math.Sin(2*math.Pi*freq*t) * volume * envelope
		writeStereoSample(buf, i, sample)
	}
	return buf
}

// generateTwoTone creates two consecutive sine tones with small fade edges.
func generateTwoTone(freq1, freq2, dur1, dur2, volume float64) []byte {
	n1 := int(float64(soundSampleRate) * dur1)
	n2 := int(float64(soundSampleRate) * dur2)
	total := n1 + n2
	buf := make([]byte, total*soundBPS)

	fadeEdge := soundSampleRate * 5 / 1000 // 5ms fade

	for i := 0; i < n1; i++ {
		t := float64(i) / float64(soundSampleRate)
		env := 1.0
		if i < fadeEdge {
			env = float64(i) / float64(fadeEdge)
		}
		if i > n1-fadeEdge {
			env = float64(n1-i) / float64(fadeEdge)
		}
		sample := math.Sin(2*math.Pi*freq1*t) * volume * env
		writeStereoSample(buf, i, sample)
	}

	for i := 0; i < n2; i++ {
		t := float64(i) / float64(soundSampleRate)
		env := 1.0
		if i < fadeEdge {
			env = float64(i) / float64(fadeEdge)
		}
		if i > n2-fadeEdge {
			env = float64(n2-i) / float64(fadeEdge)
		}
		sample := math.Sin(2*math.Pi*freq2*t) * volume * env
		writeStereoSample(buf, n1+i, sample)
	}
	return buf
}

// generateSweep creates a linear frequency sweep with fade edges.
func generateSweep(freqStart, freqEnd, duration, volume float64) []byte {
	numSamples := int(float64(soundSampleRate) * duration)
	buf := make([]byte, numSamples*soundBPS)

	fadeIn := soundSampleRate * 5 / 1000   // 5ms
	fadeOut := soundSampleRate * 10 / 1000 // 10ms

	phase := 0.0
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(numSamples)
		freq := freqStart + (freqEnd-freqStart)*t

		env := 1.0
		if i < fadeIn {
			env = float64(i) / float64(fadeIn)
		}
		if i > numSamples-fadeOut {
			env = float64(numSamples-i) / float64(fadeOut)
		}

		phase += 2 * math.Pi * freq / float64(soundSampleRate)
		sample := math.Sin(phase) * volume * env
		writeStereoSample(buf, i, sample)
	}
	return buf
}

// generateNoise creates a short burst of white noise with fade-out.
func generateNoise(duration, volume float64) []byte {
	numSamples := int(float64(soundSampleRate) * duration)
	if numSamples < 1 {
		numSamples = 1
	}
	buf := make([]byte, numSamples*soundBPS)

	rng := rand.New(rand.NewSource(99))
	fadeOut := soundSampleRate * 2 / 1000 // 2ms fade-out

	for i := 0; i < numSamples; i++ {
		env := 1.0
		if i > numSamples-fadeOut {
			env = float64(numSamples-i) / float64(fadeOut)
		}
		sample := (rng.Float64()*2 - 1) * volume * env
		writeStereoSample(buf, i, sample)
	}
	return buf
}

// writeStereoSample writes a mono sample value to both channels at position i.
func writeStereoSample(buf []byte, i int, sample float64) {
	if sample > 1.0 {
		sample = 1.0
	}
	if sample < -1.0 {
		sample = -1.0
	}
	s16 := int16(sample * 32767)
	off := i * soundBPS
	binary.LittleEndian.PutUint16(buf[off:], uint16(s16))
	binary.LittleEndian.PutUint16(buf[off+2:], uint16(s16))
}

// --- Ambient noise stream ---

// ambientNoise implements io.ReadSeeker for continuous filtered white noise.
type ambientNoise struct {
	sampleRate int
	botCount   int
	pos        int64
	rng        *rand.Rand
	prev       float64 // low-pass filter state
}

func (a *ambientNoise) Read(buf []byte) (int, error) {
	// Calculate volume proportional to bot count
	volume := float64(a.botCount) * 0.0004
	if volume > 0.08 {
		volume = 0.08
	}

	n := len(buf) / soundBPS * soundBPS // align to sample boundary
	for i := 0; i < n; i += soundBPS {
		// White noise
		raw := a.rng.Float64()*2 - 1
		// Simple low-pass: average with previous
		filtered := (raw + a.prev) * 0.5
		a.prev = filtered

		sample := filtered * volume
		s16 := int16(sample * 32767)
		binary.LittleEndian.PutUint16(buf[i:], uint16(s16))
		binary.LittleEndian.PutUint16(buf[i+2:], uint16(s16))
	}
	a.pos += int64(n)
	return n, nil
}

func (a *ambientNoise) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0: // io.SeekStart
		a.pos = offset
	case 1: // io.SeekCurrent
		a.pos += offset
	case 2: // io.SeekEnd
		// Infinite stream: treat as current
		a.pos += offset
	}
	if a.pos < 0 {
		a.pos = 0
	}
	return a.pos, nil
}
