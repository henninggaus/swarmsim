package resource

// Resource represents a collectible item on the field.
type Resource struct {
	ID         int
	X, Y       float64
	Value      float64
	Taken      bool
	CarrierID  int  // bot ID carrying this, -1 if none, -2 if delivered
	Heavy      bool // requires 3 workers to pick up
	PointValue int  // scoring: 1 for normal, 10 for heavy
}

// NewResource creates a normal resource at the given position.
func NewResource(id int, x, y, value float64) *Resource {
	return &Resource{ID: id, X: x, Y: y, Value: value, CarrierID: -1, Heavy: false, PointValue: 1}
}

// NewHeavyResource creates a heavy resource that needs cooperative pickup.
func NewHeavyResource(id int, x, y, value float64) *Resource {
	return &Resource{ID: id, X: x, Y: y, Value: value, CarrierID: -1, Heavy: true, PointValue: 10}
}

// PickUp marks the resource as taken by the given bot.
func (r *Resource) PickUp(botID int) {
	r.Taken = true
	r.CarrierID = botID
}

// Drop releases the resource at the given position.
func (r *Resource) Drop(x, y float64) {
	r.X = x
	r.Y = y
	r.Taken = false
	r.CarrierID = -1
}

// Deliver marks the resource as delivered (removes from play).
func (r *Resource) Deliver() {
	r.Taken = true
	r.CarrierID = -2 // delivered
}

// IsAvailable returns true if the resource can be picked up.
func (r *Resource) IsAvailable() bool {
	return !r.Taken
}

// IsDelivered returns true if the resource has been delivered to base.
func (r *Resource) IsDelivered() bool {
	return r.CarrierID == -2
}
