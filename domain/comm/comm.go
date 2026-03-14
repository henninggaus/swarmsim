package comm

import "math"

// NewResourceFound creates a RESOURCE_FOUND message.
func NewResourceFound(senderID int, x, y float64) Message {
	return Message{Type: MsgResourceFound, SenderID: senderID, X: x, Y: y, TTL: 3}
}

// NewHelpNeeded creates a HELP_NEEDED message.
func NewHelpNeeded(senderID int, x, y float64) Message {
	return Message{Type: MsgHelpNeeded, SenderID: senderID, X: x, Y: y, TTL: 3}
}

// NewFormationJoin creates a FORMATION_JOIN message.
func NewFormationJoin(senderID, formationID, slot int, x, y float64) Message {
	return Message{Type: MsgFormationJoin, SenderID: senderID, X: x, Y: y, ExtraID: formationID, Slot: slot, TTL: 3}
}

// NewDanger creates a DANGER message.
func NewDanger(senderID int, x, y float64) Message {
	return Message{Type: MsgDanger, SenderID: senderID, X: x, Y: y, TTL: 3}
}

// NewHeartbeat creates a HEARTBEAT message.
func NewHeartbeat(senderID int, x, y float64) Message {
	return Message{Type: MsgHeartbeat, SenderID: senderID, X: x, Y: y, TTL: 1}
}

// NewHeavyResourceFound creates a HEAVY_RESOURCE_FOUND message with longer TTL.
func NewHeavyResourceFound(senderID int, x, y float64) Message {
	return Message{Type: MsgHeavyResourceFound, SenderID: senderID, X: x, Y: y, TTL: 5}
}

// NewPackageFound creates a PACKAGE_FOUND message (truck mode).
func NewPackageFound(senderID int, x, y float64, pkgID int) Message {
	return Message{Type: MsgPackageFound, SenderID: senderID, X: x, Y: y, ExtraID: pkgID, TTL: 5}
}

// NewNeedCoopHelp creates a NEED_COOP_HELP message (truck mode).
func NewNeedCoopHelp(senderID int, x, y float64, pkgID int) Message {
	return Message{Type: MsgNeedCoopHelp, SenderID: senderID, X: x, Y: y, ExtraID: pkgID, TTL: 8}
}

// NewRampCongested creates a RAMP_CONGESTED message (truck mode).
func NewRampCongested(senderID int, x, y float64) Message {
	return Message{Type: MsgRampCongested, SenderID: senderID, X: x, Y: y, TTL: 3}
}

// NewTaskAssign creates a TASK_ASSIGN message (truck mode).
func NewTaskAssign(senderID int, x, y float64, pkgID int) Message {
	return Message{Type: MsgTaskAssign, SenderID: senderID, X: x, Y: y, ExtraID: pkgID, TTL: 5}
}

// Channel handles radius-based message delivery between bots.
type Channel struct {
	pending []PendingMsg
}

// NewChannel creates a new communication channel.
func NewChannel() *Channel {
	return &Channel{}
}

// Send queues a message from a position with the sender's communication range.
func (c *Channel) Send(msg Message, originX, originY, commRange float64) {
	c.pending = append(c.pending, PendingMsg{
		Msg: msg, OriginX: originX, OriginY: originY, CommRange: commRange,
	})
}

// Deliver returns all messages receivable at position (rx, ry).
func (c *Channel) Deliver(rx, ry float64) []Message {
	var result []Message
	for _, p := range c.pending {
		dx := p.OriginX - rx
		dy := p.OriginY - ry
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= p.CommRange {
			result = append(result, p.Msg)
		}
	}
	return result
}

// Tick ages all messages and removes expired ones.
func (c *Channel) Tick() int {
	alive := c.pending[:0]
	for _, p := range c.pending {
		p.Msg.TTL--
		if p.Msg.TTL > 0 {
			alive = append(alive, p)
		}
	}
	c.pending = alive
	return len(c.pending)
}

// Clear removes all pending messages.
func (c *Channel) Clear() {
	c.pending = c.pending[:0]
}

// ActiveCount returns the number of active messages.
func (c *Channel) ActiveCount() int {
	return len(c.pending)
}

// PendingMessages returns a copy of all pending messages (for debug visualization).
func (c *Channel) PendingMessages() []PendingMsg {
	cp := make([]PendingMsg, len(c.pending))
	copy(cp, c.pending)
	return cp
}

// PendingMsgOrigin returns origin position and range info for debug rendering.
func PendingMsgOrigin(p PendingMsg) (float64, float64, float64, Message) {
	return p.OriginX, p.OriginY, p.CommRange, p.Msg
}
