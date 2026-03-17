package comm

import "testing"

func TestNewSignalChannel(t *testing.T) {
	sc := NewSignalChannel()
	if sc == nil {
		t.Fatal("expected non-nil SignalChannel")
	}
	if sc.SignalCount() != 0 {
		t.Error("new channel should have 0 signals")
	}
}

func TestSendAndDeliverSignal(t *testing.T) {
	sc := NewSignalChannel()
	sig := NewSignal("found_food", 1, 100, 100, [4]float64{42, 0, 0, 0})
	sc.SendSignal(sig, 100, 100, 50)

	// Within range
	received := sc.DeliverSignals(120, 100, "")
	if len(received) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(received))
	}
	if received[0].Name != "found_food" {
		t.Errorf("expected name 'found_food', got '%s'", received[0].Name)
	}
	if received[0].Payload[0] != 42 {
		t.Errorf("expected payload[0]=42, got %f", received[0].Payload[0])
	}

	// Out of range
	far := sc.DeliverSignals(500, 500, "")
	if len(far) != 0 {
		t.Errorf("expected 0 signals out of range, got %d", len(far))
	}
}

func TestDeliverSignalsWithFilter(t *testing.T) {
	sc := NewSignalChannel()
	sc.SendSignal(NewSignal("food", 1, 100, 100, [4]float64{}), 100, 100, 50)
	sc.SendSignal(NewSignal("danger", 2, 100, 100, [4]float64{}), 100, 100, 50)
	sc.SendSignal(NewSignal("food", 3, 100, 100, [4]float64{}), 100, 100, 50)

	food := sc.DeliverSignals(100, 100, "food")
	if len(food) != 2 {
		t.Errorf("expected 2 food signals, got %d", len(food))
	}

	danger := sc.DeliverSignals(100, 100, "danger")
	if len(danger) != 1 {
		t.Errorf("expected 1 danger signal, got %d", len(danger))
	}

	all := sc.DeliverSignals(100, 100, "")
	if len(all) != 3 {
		t.Errorf("expected 3 total signals, got %d", len(all))
	}
}

func TestTickSignals(t *testing.T) {
	sc := NewSignalChannel()
	sig := NewSignal("test", 1, 100, 100, [4]float64{})
	sig.TTL = 3
	sc.SendSignal(sig, 100, 100, 50)

	sc.TickSignals()
	if sc.SignalCount() != 1 {
		t.Error("signal should survive first tick (TTL=2)")
	}

	sc.TickSignals()
	if sc.SignalCount() != 1 {
		t.Error("signal should survive second tick (TTL=1)")
	}

	sc.TickSignals()
	if sc.SignalCount() != 0 {
		t.Error("signal should expire after third tick")
	}
}

func TestClearSignals(t *testing.T) {
	sc := NewSignalChannel()
	sc.SendSignal(NewSignal("a", 1, 0, 0, [4]float64{}), 0, 0, 100)
	sc.SendSignal(NewSignal("b", 2, 0, 0, [4]float64{}), 0, 0, 100)
	sc.ClearSignals()
	if sc.SignalCount() != 0 {
		t.Error("clear should remove all signals")
	}
}

func TestSignalCountByName(t *testing.T) {
	sc := NewSignalChannel()
	sc.SendSignal(NewSignal("food", 1, 0, 0, [4]float64{}), 0, 0, 100)
	sc.SendSignal(NewSignal("food", 2, 0, 0, [4]float64{}), 0, 0, 100)
	sc.SendSignal(NewSignal("danger", 3, 0, 0, [4]float64{}), 0, 0, 100)

	if sc.SignalCountByName("food") != 2 {
		t.Errorf("expected 2 food signals")
	}
	if sc.SignalCountByName("danger") != 1 {
		t.Errorf("expected 1 danger signal")
	}
	if sc.SignalCountByName("nope") != 0 {
		t.Errorf("expected 0 for unknown name")
	}
}

func TestSignalPayload(t *testing.T) {
	payload := [4]float64{1.5, 2.5, 3.5, 4.5}
	sig := NewSignal("data", 0, 50, 50, payload)
	for i := range payload {
		if sig.Payload[i] != payload[i] {
			t.Errorf("payload[%d] mismatch: %f != %f", i, sig.Payload[i], payload[i])
		}
	}
}
