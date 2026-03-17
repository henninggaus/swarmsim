package swarm

// ReplaySnapshot stores minimal bot state for one tick.
type ReplaySnapshot struct {
	Tick     int
	BotData  []ReplayBot
}

// ReplayBot stores position + LED for replay rendering.
type ReplayBot struct {
	X, Y     float64
	Angle    float64
	LEDR     uint8
	LEDG     uint8
	LEDB     uint8
	Carrying bool // carrying a package?
}

// ReplayBuffer is a ring buffer of simulation snapshots.
type ReplayBuffer struct {
	Snapshots []ReplaySnapshot
	MaxSize   int
	WriteIdx  int
	Count     int
}

// NewReplayBuffer creates a buffer for up to maxSize snapshots.
func NewReplayBuffer(maxSize int) *ReplayBuffer {
	return &ReplayBuffer{
		Snapshots: make([]ReplaySnapshot, maxSize),
		MaxSize:   maxSize,
	}
}

// Record captures current bot state as a snapshot.
func (rb *ReplayBuffer) Record(ss *SwarmState) {
	snap := ReplaySnapshot{
		Tick:    ss.Tick,
		BotData: make([]ReplayBot, len(ss.Bots)),
	}
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		snap.BotData[i] = ReplayBot{
			X: bot.X, Y: bot.Y,
			Angle:    bot.Angle,
			LEDR:     bot.LEDColor[0],
			LEDG:     bot.LEDColor[1],
			LEDB:     bot.LEDColor[2],
			Carrying: bot.CarryingPkg >= 0,
		}
	}
	rb.Snapshots[rb.WriteIdx] = snap
	rb.WriteIdx = (rb.WriteIdx + 1) % rb.MaxSize
	if rb.Count < rb.MaxSize {
		rb.Count++
	}
}

// Get returns the snapshot at logical index (0 = oldest, Count-1 = newest).
func (rb *ReplayBuffer) Get(idx int) *ReplaySnapshot {
	if idx < 0 || idx >= rb.Count {
		return nil
	}
	// Ring buffer: oldest entry is at WriteIdx - Count
	realIdx := (rb.WriteIdx - rb.Count + idx + rb.MaxSize) % rb.MaxSize
	return &rb.Snapshots[realIdx]
}

// Newest returns the most recent snapshot.
func (rb *ReplayBuffer) Newest() *ReplaySnapshot {
	if rb.Count == 0 {
		return nil
	}
	return rb.Get(rb.Count - 1)
}

// Oldest returns the oldest snapshot.
func (rb *ReplayBuffer) Oldest() *ReplaySnapshot {
	if rb.Count == 0 {
		return nil
	}
	return rb.Get(0)
}

// ReplayPlayer provides seek/rewind/step-through controls over a ReplayBuffer.
type ReplayPlayer struct {
	Buffer    *ReplayBuffer
	Cursor    int  // current logical index in buffer (0..Count-1)
	Playing   bool // auto-advancing?
	Speed     int  // playback speed: 1=normal, 2=2x, -1=reverse, etc.
	LoopMode  bool // loop back to start when reaching end
}

// NewReplayPlayer creates a player for the given buffer, starting at newest frame.
func NewReplayPlayer(buf *ReplayBuffer) *ReplayPlayer {
	cursor := 0
	if buf != nil && buf.Count > 0 {
		cursor = buf.Count - 1
	}
	return &ReplayPlayer{
		Buffer: buf,
		Cursor: cursor,
		Speed:  1,
	}
}

// Current returns the snapshot at the current cursor position.
func (rp *ReplayPlayer) Current() *ReplaySnapshot {
	if rp.Buffer == nil {
		return nil
	}
	return rp.Buffer.Get(rp.Cursor)
}

// SeekTo moves the cursor to a specific logical index, clamped to valid range.
func (rp *ReplayPlayer) SeekTo(idx int) {
	if rp.Buffer == nil || rp.Buffer.Count == 0 {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= rp.Buffer.Count {
		idx = rp.Buffer.Count - 1
	}
	rp.Cursor = idx
}

// SeekStart jumps to the oldest snapshot.
func (rp *ReplayPlayer) SeekStart() {
	rp.SeekTo(0)
}

// SeekEnd jumps to the newest snapshot.
func (rp *ReplayPlayer) SeekEnd() {
	if rp.Buffer != nil {
		rp.SeekTo(rp.Buffer.Count - 1)
	}
}

// StepForward advances one frame. Returns false if at end (and not looping).
func (rp *ReplayPlayer) StepForward() bool {
	if rp.Buffer == nil || rp.Buffer.Count == 0 {
		return false
	}
	if rp.Cursor >= rp.Buffer.Count-1 {
		if rp.LoopMode {
			rp.Cursor = 0
			return true
		}
		return false
	}
	rp.Cursor++
	return true
}

// StepBackward goes back one frame. Returns false if at start (and not looping).
func (rp *ReplayPlayer) StepBackward() bool {
	if rp.Buffer == nil || rp.Buffer.Count == 0 {
		return false
	}
	if rp.Cursor <= 0 {
		if rp.LoopMode {
			rp.Cursor = rp.Buffer.Count - 1
			return true
		}
		return false
	}
	rp.Cursor--
	return true
}

// TickPlayback advances the player by Speed frames (called once per render tick).
// Positive Speed = forward, negative = rewind.
func (rp *ReplayPlayer) TickPlayback() {
	if !rp.Playing || rp.Buffer == nil || rp.Buffer.Count == 0 {
		return
	}
	steps := rp.Speed
	if steps == 0 {
		steps = 1
	}
	if steps > 0 {
		for i := 0; i < steps; i++ {
			if !rp.StepForward() {
				rp.Playing = false
				break
			}
		}
	} else {
		for i := 0; i < -steps; i++ {
			if !rp.StepBackward() {
				rp.Playing = false
				break
			}
		}
	}
}

// SeekPercent seeks to a position given as 0.0 (oldest) to 1.0 (newest).
func (rp *ReplayPlayer) SeekPercent(pct float64) {
	if rp.Buffer == nil || rp.Buffer.Count == 0 {
		return
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	idx := int(pct * float64(rp.Buffer.Count-1))
	rp.SeekTo(idx)
}

// Progress returns current position as 0.0..1.0.
func (rp *ReplayPlayer) Progress() float64 {
	if rp.Buffer == nil || rp.Buffer.Count <= 1 {
		return 0
	}
	return float64(rp.Cursor) / float64(rp.Buffer.Count-1)
}

// FramesTotal returns the total number of recorded frames.
func (rp *ReplayPlayer) FramesTotal() int {
	if rp.Buffer == nil {
		return 0
	}
	return rp.Buffer.Count
}

// TogglePlay toggles play/pause.
func (rp *ReplayPlayer) TogglePlay() {
	rp.Playing = !rp.Playing
}

// SetSpeed sets playback speed. Positive=forward, negative=rewind.
func (rp *ReplayPlayer) SetSpeed(s int) {
	rp.Speed = s
}
