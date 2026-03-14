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
