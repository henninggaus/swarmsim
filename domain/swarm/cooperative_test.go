package swarm

import (
	"math/rand"
	"testing"
)

func makeCoopSS(rng *rand.Rand) *SwarmState {
	ss := NewSwarmState(rng, 10)
	ss.CoopState = NewCooperativeState()
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*10
		ss.Bots[i].Y = 400
		ss.Bots[i].Stats.TotalDeliveries = i * 5 // varying fitness
		ss.Bots[i].Stats.TicksAlive = 100
		// Give bots different param values so transfer can happen
		for p := 0; p < 26; p++ {
			ss.Bots[i].ParamValues[p] = float64(i) * 10.0
		}
	}
	return ss
}

func TestNewCooperativeState(t *testing.T) {
	cs := NewCooperativeState()
	if cs == nil {
		t.Fatal("should not be nil")
	}
	if cs.Config.TeachRange != 60 {
		t.Error("wrong default TeachRange")
	}
	if cs.MaxEvents != 100 {
		t.Error("wrong default MaxEvents")
	}
}

func TestRunCooperativeLearningNilState(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	RunCooperativeLearning(rng, ss) // should not panic
}

func TestRunCooperativeLearningTransfers(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := makeCoopSS(rng)
	ss.CoopState.Config.TeachRate = 1.0 // always attempt
	ss.CoopState.Config.MinFitGap = 1.0

	for i := 0; i < 100; i++ {
		RunCooperativeLearning(rng, ss)
		ss.Tick++
	}

	if ss.CoopState.TotalTeaches == 0 {
		t.Error("expected some teaches with 100% rate")
	}
}

func TestTransferNeuroWeights(t *testing.T) {
	teacher := &SwarmBot{}
	learner := &SwarmBot{}
	teacher.Brain = &NeuroBrain{}
	learner.Brain = &NeuroBrain{}
	for w := range teacher.Brain.Weights {
		teacher.Brain.Weights[w] = 10
		learner.Brain.Weights[w] = 0
	}

	ok := transferNeuroWeights(teacher, learner, 0.5)
	if !ok {
		t.Fatal("should succeed")
	}
	if learner.Brain.Weights[0] != 5 {
		t.Errorf("expected 5 (blend 50%%), got %f", learner.Brain.Weights[0])
	}
}

func TestTransferNeuroWeightsNilBrains(t *testing.T) {
	a := &SwarmBot{}
	b := &SwarmBot{}
	if transferNeuroWeights(a, b, 0.5) {
		t.Error("should fail with nil brains")
	}
}

func TestTransferParamValues(t *testing.T) {
	teacher := &SwarmBot{}
	learner := &SwarmBot{}
	teacher.ParamValues[0] = 100
	learner.ParamValues[0] = 0

	ok := transferParamValues(teacher, learner, 0.3)
	if !ok {
		t.Fatal("should succeed")
	}
	if learner.ParamValues[0] != 30 {
		t.Errorf("expected 30, got %f", learner.ParamValues[0])
	}
}

func TestRecentTeachCount(t *testing.T) {
	coop := NewCooperativeState()
	addTeachEvent(coop, TeachEvent{Tick: 100})
	addTeachEvent(coop, TeachEvent{Tick: 150})
	addTeachEvent(coop, TeachEvent{Tick: 200})

	if RecentTeachCount(coop, 200, 60) != 2 {
		t.Error("expected 2 events in last 60 ticks")
	}
	if RecentTeachCount(coop, 200, 200) != 3 {
		t.Error("expected 3 events in last 200 ticks")
	}
}

func TestRecentTeachCountNil(t *testing.T) {
	if RecentTeachCount(nil, 100, 50) != 0 {
		t.Error("nil should return 0")
	}
}

func TestTeachEventRingBuffer(t *testing.T) {
	coop := NewCooperativeState()
	coop.MaxEvents = 5
	for i := 0; i < 10; i++ {
		addTeachEvent(coop, TeachEvent{Tick: i})
	}
	if len(coop.TeachEvents) != 5 {
		t.Errorf("expected 5 events (ring buffer), got %d", len(coop.TeachEvents))
	}
	if coop.TeachEvents[0].Tick != 5 {
		t.Errorf("oldest should be tick 5, got %d", coop.TeachEvents[0].Tick)
	}
}
