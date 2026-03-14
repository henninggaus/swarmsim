package comm

import (
	"testing"
)

// --- Message constructor tests ---

func TestMessageConstructors(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		msgType MsgType
		ttl     int
	}{
		{"ResourceFound", NewResourceFound(1, 10, 20), MsgResourceFound, 3},
		{"HelpNeeded", NewHelpNeeded(2, 30, 40), MsgHelpNeeded, 3},
		{"Danger", NewDanger(3, 50, 60), MsgDanger, 3},
		{"Heartbeat", NewHeartbeat(4, 70, 80), MsgHeartbeat, 1},
		{"HeavyResourceFound", NewHeavyResourceFound(5, 90, 100), MsgHeavyResourceFound, 5},
		{"PackageFound", NewPackageFound(6, 10, 20, 99), MsgPackageFound, 5},
		{"NeedCoopHelp", NewNeedCoopHelp(7, 30, 40, 88), MsgNeedCoopHelp, 8},
		{"RampCongested", NewRampCongested(8, 50, 60), MsgRampCongested, 3},
		{"TaskAssign", NewTaskAssign(9, 70, 80, 77), MsgTaskAssign, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.msg.Type != tc.msgType {
				t.Errorf("expected type %d, got %d", tc.msgType, tc.msg.Type)
			}
			if tc.msg.TTL != tc.ttl {
				t.Errorf("expected TTL %d, got %d", tc.ttl, tc.msg.TTL)
			}
		})
	}
}

func TestFormationJoinMessage(t *testing.T) {
	msg := NewFormationJoin(1, 5, 2, 100, 200)
	if msg.Type != MsgFormationJoin {
		t.Errorf("expected MsgFormationJoin, got %d", msg.Type)
	}
	if msg.ExtraID != 5 || msg.Slot != 2 {
		t.Errorf("expected ExtraID=5, Slot=2, got ExtraID=%d, Slot=%d", msg.ExtraID, msg.Slot)
	}
	if msg.X != 100 || msg.Y != 200 {
		t.Errorf("expected position (100,200), got (%.0f,%.0f)", msg.X, msg.Y)
	}
}

// --- Channel broadcast radius tests ---

func TestMessageBroadcastRadius(t *testing.T) {
	ch := NewChannel()

	// Send a message from (100, 100) with comm range 50
	msg := NewResourceFound(1, 100, 100)
	ch.Send(msg, 100, 100, 50)

	// Receiver at (120, 120) — distance ~28.3, within range
	received := ch.Deliver(120, 120)
	if len(received) != 1 {
		t.Fatalf("expected 1 message within range, got %d", len(received))
	}
	if received[0].Type != MsgResourceFound {
		t.Errorf("expected MsgResourceFound, got %d", received[0].Type)
	}

	// Receiver at (500, 500) — far outside range
	received = ch.Deliver(500, 500)
	if len(received) != 0 {
		t.Errorf("expected 0 messages outside range, got %d", len(received))
	}
}

func TestMessageBroadcastExactEdge(t *testing.T) {
	ch := NewChannel()
	msg := NewResourceFound(1, 0, 0)
	ch.Send(msg, 0, 0, 100)

	// Receiver exactly at range boundary (distance == CommRange)
	received := ch.Deliver(100, 0)
	if len(received) != 1 {
		t.Errorf("receiver at exact range boundary should receive message (<=), got %d", len(received))
	}
}

// --- TTL tests ---

func TestMessageTTL(t *testing.T) {
	ch := NewChannel()
	// Heartbeat has TTL=1
	msg := NewHeartbeat(1, 0, 0)
	ch.Send(msg, 0, 0, 100)

	if ch.ActiveCount() != 1 {
		t.Fatalf("expected 1 active message, got %d", ch.ActiveCount())
	}

	// One tick: TTL goes from 1 to 0, message should be removed
	remaining := ch.Tick()
	if remaining != 0 {
		t.Errorf("expected 0 remaining after TTL=1 message ticked, got %d", remaining)
	}
	if ch.ActiveCount() != 0 {
		t.Errorf("expected 0 active messages, got %d", ch.ActiveCount())
	}
}

func TestMessageTTLMultipleTicks(t *testing.T) {
	ch := NewChannel()
	// ResourceFound has TTL=3
	msg := NewResourceFound(1, 0, 0)
	ch.Send(msg, 0, 0, 100)

	// After 1 tick: TTL=2
	ch.Tick()
	if ch.ActiveCount() != 1 {
		t.Error("message should survive first tick (TTL 3->2)")
	}

	// After 2 ticks: TTL=1
	ch.Tick()
	if ch.ActiveCount() != 1 {
		t.Error("message should survive second tick (TTL 2->1)")
	}

	// After 3 ticks: TTL=0, removed
	ch.Tick()
	if ch.ActiveCount() != 0 {
		t.Error("message should expire after 3 ticks")
	}
}

// --- Relay / Multiple messages ---

func TestMessageRelay(t *testing.T) {
	ch := NewChannel()

	// Bot A sends from (0, 0) with range 100
	ch.Send(NewResourceFound(1, 50, 50), 0, 0, 100)
	// Bot B sends from (200, 0) with range 100
	ch.Send(NewDanger(2, 200, 0), 200, 0, 100)

	// Receiver at (100, 0): within range of both senders
	received := ch.Deliver(100, 0)
	if len(received) != 2 {
		t.Errorf("expected 2 messages at (100,0), got %d", len(received))
	}

	// Receiver at (0, 0): only within range of bot A
	received = ch.Deliver(0, 0)
	if len(received) != 1 {
		t.Errorf("expected 1 message at (0,0), got %d", len(received))
	}
	if received[0].SenderID != 1 {
		t.Errorf("expected sender 1, got %d", received[0].SenderID)
	}
}

// --- Channel Clear tests ---

func TestChannelClear(t *testing.T) {
	ch := NewChannel()
	ch.Send(NewResourceFound(1, 0, 0), 0, 0, 100)
	ch.Send(NewDanger(2, 10, 10), 10, 10, 100)

	if ch.ActiveCount() != 2 {
		t.Fatalf("expected 2 active, got %d", ch.ActiveCount())
	}

	ch.Clear()
	if ch.ActiveCount() != 0 {
		t.Errorf("expected 0 after clear, got %d", ch.ActiveCount())
	}
}

// --- PendingMessages tests ---

func TestPendingMessages(t *testing.T) {
	ch := NewChannel()
	ch.Send(NewResourceFound(1, 50, 60), 10, 20, 100)

	pending := ch.PendingMessages()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}

	ox, oy, cr, msg := PendingMsgOrigin(pending[0])
	if ox != 10 || oy != 20 {
		t.Errorf("expected origin (10,20), got (%.0f,%.0f)", ox, oy)
	}
	if cr != 100 {
		t.Errorf("expected commRange 100, got %.0f", cr)
	}
	if msg.X != 50 || msg.Y != 60 {
		t.Errorf("expected payload position (50,60), got (%.0f,%.0f)", msg.X, msg.Y)
	}
}
