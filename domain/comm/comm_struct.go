package comm

// MsgType identifies the kind of message.
type MsgType int

const (
	MsgResourceFound MsgType = iota
	MsgHelpNeeded
	MsgFormationJoin
	MsgDanger
	MsgHeartbeat
	MsgHeavyResourceFound
	MsgPackageFound  // scout found a package (truck mode)
	MsgNeedCoopHelp  // bot needs cooperative lift (truck mode)
	MsgRampCongested // leader warns about ramp traffic (truck mode)
	MsgTaskAssign    // leader assigns package to worker (truck mode)
)

// Message is a communication between bots.
type Message struct {
	Type     MsgType
	SenderID int
	X, Y     float64 // position payload
	ExtraID  int     // formation ID or other context
	Slot     int     // formation slot
	TTL      int     // ticks remaining
}

// PendingMsg is a message queued for delivery with sender position info.
type PendingMsg struct {
	Msg       Message
	OriginX   float64
	OriginY   float64
	CommRange float64
}

// --- Named Signal Protocol ---

// Signal is a named broadcast with arbitrary payload data.
type Signal struct {
	Name     string     // signal name (e.g. "found_food", "need_help")
	SenderID int
	X, Y     float64    // sender position
	Payload  [4]float64 // up to 4 float values as payload
	TTL      int
}

// SignalChannel handles named signal delivery between bots.
type SignalChannel struct {
	pending []PendingSignal
}

// PendingSignal wraps a signal with delivery metadata.
type PendingSignal struct {
	Sig       Signal
	OriginX   float64
	OriginY   float64
	CommRange float64
}
