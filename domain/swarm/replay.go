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
