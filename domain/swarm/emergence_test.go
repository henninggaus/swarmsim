package swarm

import "testing"

func TestDetectEmergence_NoPatternResult(t *testing.T) {
	ss := &SwarmState{
		EmergenceShown: make(map[EmergenceEvent]bool),
	}
	popup := DetectEmergence(ss)
	if popup != nil {
		t.Error("should return nil without PatternResult")
	}
}

func TestDetectEmergence_ClusterFormed(t *testing.T) {
	ss := &SwarmState{
		EmergenceShown: make(map[EmergenceEvent]bool),
		PatternResult: &PatternResult{
			Cohesion:     0.9,
			ClusterCount: 4,
		},
	}
	popup := DetectEmergence(ss)
	if popup == nil {
		t.Fatal("should detect cluster")
	}
	if popup.Event != EmergenceClusterFormed {
		t.Error("wrong event type")
	}
}

func TestDetectEmergence_OnlyOnce(t *testing.T) {
	ss := &SwarmState{
		EmergenceShown: make(map[EmergenceEvent]bool),
		PatternResult: &PatternResult{
			Cohesion:     0.9,
			ClusterCount: 4,
		},
	}
	popup1 := DetectEmergence(ss)
	if popup1 == nil {
		t.Fatal("first detection should work")
	}
	// DetectEmergence already marks as shown internally
	popup2 := DetectEmergence(ss)
	// Should not detect same event again (unless other events trigger)
	if popup2 != nil && popup2.Event == EmergenceClusterFormed {
		t.Error("should not re-trigger same event")
	}
}

func TestTickEmergencePopup(t *testing.T) {
	ss := &SwarmState{
		EmergencePopup: &EmergencePopup{Timer: 5},
	}
	TickEmergencePopup(ss)
	if ss.EmergencePopup.Timer != 4 {
		t.Error("timer should decrement")
	}
	// Tick to 0
	for i := 0; i < 10; i++ {
		TickEmergencePopup(ss)
	}
	if ss.EmergencePopup != nil {
		t.Error("should be nil when timer expires")
	}
}
