package resource

import "testing"

func TestNewResource(t *testing.T) {
	r := NewResource(1, 50, 60, 10)
	if r.ID != 1 {
		t.Errorf("expected ID=1, got %d", r.ID)
	}
	if r.X != 50 || r.Y != 60 {
		t.Errorf("expected pos (50,60), got (%.0f,%.0f)", r.X, r.Y)
	}
	if r.Value != 10 {
		t.Errorf("expected Value=10, got %v", r.Value)
	}
	if r.CarrierID != -1 {
		t.Errorf("expected CarrierID=-1, got %d", r.CarrierID)
	}
	if r.Heavy {
		t.Error("normal resource should not be heavy")
	}
	if r.PointValue != 1 {
		t.Errorf("expected PointValue=1, got %d", r.PointValue)
	}
}

func TestNewHeavyResource(t *testing.T) {
	r := NewHeavyResource(2, 100, 200, 50)
	if !r.Heavy {
		t.Error("heavy resource should be heavy")
	}
	if r.PointValue != 10 {
		t.Errorf("expected PointValue=10, got %d", r.PointValue)
	}
}

func TestPickUpAndDrop(t *testing.T) {
	r := NewResource(1, 50, 50, 10)
	if !r.IsAvailable() {
		t.Error("new resource should be available")
	}

	r.PickUp(42)
	if r.IsAvailable() {
		t.Error("picked up resource should not be available")
	}
	if r.CarrierID != 42 {
		t.Errorf("expected CarrierID=42, got %d", r.CarrierID)
	}
	if !r.Taken {
		t.Error("expected Taken=true")
	}

	r.Drop(100, 200)
	if !r.IsAvailable() {
		t.Error("dropped resource should be available")
	}
	if r.CarrierID != -1 {
		t.Errorf("expected CarrierID=-1 after drop, got %d", r.CarrierID)
	}
	if r.X != 100 || r.Y != 200 {
		t.Errorf("expected pos (100,200), got (%.0f,%.0f)", r.X, r.Y)
	}
}

func TestDeliver(t *testing.T) {
	r := NewResource(1, 50, 50, 10)
	r.Deliver()
	if !r.IsDelivered() {
		t.Error("delivered resource should report IsDelivered")
	}
	if r.CarrierID != -2 {
		t.Errorf("expected CarrierID=-2, got %d", r.CarrierID)
	}
	if r.IsAvailable() {
		t.Error("delivered resource should not be available")
	}
}

func TestIsDeliveredFalse(t *testing.T) {
	r := NewResource(1, 50, 50, 10)
	if r.IsDelivered() {
		t.Error("new resource should not be delivered")
	}
	r.PickUp(5)
	if r.IsDelivered() {
		t.Error("picked up resource should not be delivered")
	}
}
