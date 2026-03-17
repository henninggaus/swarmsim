package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func makeRLState() *RLState {
	rl := &RLState{
		NumStates:    rlNumStates,
		NumActions:   rlNumActions,
		Alpha:        0.1,
		Gamma:        0.95,
		Epsilon:      0.2,
		EpsilonDecay: 0.995,
		EpsilonMin:   0.01,
		MaxHistory:   10,
	}
	rl.QTable = make([][]float64, rl.NumStates)
	for i := range rl.QTable {
		rl.QTable[i] = make([]float64, rl.NumActions)
	}
	return rl
}

func TestInitRL(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitRL(ss)
	if ss.RLState == nil {
		t.Fatal("RLState should not be nil")
	}
	if !ss.RLEnabled {
		t.Error("RL should be enabled")
	}
	if len(ss.RLState.QTable) != rlNumStates {
		t.Errorf("QTable should have %d states, got %d", rlNumStates, len(ss.RLState.QTable))
	}
	if len(ss.RLBotStates) != len(ss.Bots) {
		t.Error("RLBotStates should match bot count")
	}
}

func TestClearRL(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitRL(ss)
	ClearRL(ss)
	if ss.RLState != nil {
		t.Error("RLState should be nil")
	}
	if ss.RLEnabled {
		t.Error("RL should be disabled")
	}
}

func TestDiscretizeState(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	bot := &SwarmBot{X: 50, Y: 50, CarryingPkg: -1}
	state := DiscretizeState(bot, ss)
	if state < 0 || state >= rlNumStates {
		t.Errorf("state %d out of bounds [0,%d)", state, rlNumStates)
	}
}

func TestDiscretizeStateBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	// Edge case: bot at arena boundary
	bot := &SwarmBot{X: ss.ArenaW, Y: ss.ArenaH, CarryingPkg: -1}
	state := DiscretizeState(bot, ss)
	if state < 0 || state >= rlNumStates {
		t.Errorf("boundary state %d out of bounds", state)
	}
	// Edge case: negative coords
	bot2 := &SwarmBot{X: -10, Y: -10, CarryingPkg: -1}
	state2 := DiscretizeState(bot2, ss)
	if state2 < 0 || state2 >= rlNumStates {
		t.Errorf("negative state %d out of bounds", state2)
	}
}

func TestDiscretizeStateCarrying(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	notCarrying := &SwarmBot{X: 100, Y: 100, CarryingPkg: -1}
	carrying := &SwarmBot{X: 100, Y: 100, CarryingPkg: 0}
	s1 := DiscretizeState(notCarrying, ss)
	s2 := DiscretizeState(carrying, ss)
	if s1 == s2 {
		t.Error("carrying and not-carrying should map to different states")
	}
}

func TestRLChooseActionGreedy(t *testing.T) {
	rl := makeRLState()
	rng := rand.New(rand.NewSource(42))
	rl.Epsilon = 0 // pure greedy
	rl.QTable[5][3] = 10.0
	action := RLChooseAction(rl, 5, rng)
	if action != 3 {
		t.Errorf("greedy should pick action 3, got %d", action)
	}
}

func TestRLChooseActionExploration(t *testing.T) {
	rl := makeRLState()
	rng := rand.New(rand.NewSource(42))
	rl.Epsilon = 1.0 // always explore
	actions := make(map[int]bool)
	for i := 0; i < 100; i++ {
		a := RLChooseAction(rl, 0, rng)
		actions[a] = true
	}
	if len(actions) < 3 {
		t.Error("full exploration should produce diverse actions")
	}
}

func TestRLChooseActionNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	a := RLChooseAction(nil, 0, rng)
	if a != 0 {
		t.Errorf("nil RL should return 0, got %d", a)
	}
}

func TestRLUpdate(t *testing.T) {
	rl := makeRLState()
	RLUpdate(rl, 0, 0, 1, 10.0)
	if rl.QTable[0][0] == 0 {
		t.Error("Q-value should be updated")
	}
	// Q(0,0) = 0 + 0.1 * (10 + 0.95*0 - 0) = 1.0
	expected := 1.0
	if math.Abs(rl.QTable[0][0]-expected) > 0.001 {
		t.Errorf("expected Q=%.3f, got %.3f", expected, rl.QTable[0][0])
	}
}

func TestRLUpdateNil(t *testing.T) {
	RLUpdate(nil, 0, 0, 0, 1.0) // should not panic
}

func TestRLUpdateOutOfBounds(t *testing.T) {
	rl := makeRLState()
	RLUpdate(rl, -1, 0, 0, 1.0) // should not panic
	RLUpdate(rl, 0, -1, 0, 1.0)
	RLUpdate(rl, 0, 0, -1, 1.0)
	RLUpdate(rl, rlNumStates, 0, 0, 1.0)
}

func TestRLDecayEpsilon(t *testing.T) {
	rl := makeRLState()
	rl.Epsilon = 0.5
	RLDecayEpsilon(rl)
	if rl.Epsilon >= 0.5 {
		t.Error("epsilon should decrease")
	}
	if rl.Episode != 1 {
		t.Error("episode should increment")
	}
}

func TestRLDecayEpsilonMin(t *testing.T) {
	rl := makeRLState()
	rl.Epsilon = 0.005
	rl.EpsilonMin = 0.01
	RLDecayEpsilon(rl)
	if rl.Epsilon != 0.01 {
		t.Errorf("epsilon should clamp to min, got %f", rl.Epsilon)
	}
}

func TestRLDecayEpsilonNil(t *testing.T) {
	RLDecayEpsilon(nil) // should not panic
}

func TestRLRecordReward(t *testing.T) {
	rl := makeRLState()
	RLRecordReward(rl, 50.0)
	RLRecordReward(rl, 100.0)
	if len(rl.RewardHistory) != 2 {
		t.Errorf("expected 2 entries, got %d", len(rl.RewardHistory))
	}
	if rl.AvgReward != 75.0 {
		t.Errorf("expected avg 75, got %f", rl.AvgReward)
	}
}

func TestRLRecordRewardPruning(t *testing.T) {
	rl := makeRLState()
	rl.MaxHistory = 3
	for i := 0; i < 5; i++ {
		RLRecordReward(rl, float64(i))
	}
	if len(rl.RewardHistory) != 3 {
		t.Errorf("history should be capped at 3, got %d", len(rl.RewardHistory))
	}
}

func TestRLMaxQ(t *testing.T) {
	rl := makeRLState()
	if RLMaxQ(rl) != 0 {
		t.Error("empty table should have maxQ 0")
	}
	rl.QTable[10][5] = 42.0
	if RLMaxQ(rl) != 42.0 {
		t.Errorf("expected maxQ 42, got %f", RLMaxQ(rl))
	}
}

func TestRLMaxQNil(t *testing.T) {
	if RLMaxQ(nil) != 0 {
		t.Error("nil should return 0")
	}
}

func TestRLNonZeroEntries(t *testing.T) {
	rl := makeRLState()
	if RLNonZeroEntries(rl) != 0 {
		t.Error("empty table should have 0 non-zero entries")
	}
	rl.QTable[0][0] = 1.0
	rl.QTable[5][3] = 2.0
	if RLNonZeroEntries(rl) != 2 {
		t.Errorf("expected 2 non-zero, got %d", RLNonZeroEntries(rl))
	}
}

func TestRLNonZeroNil(t *testing.T) {
	if RLNonZeroEntries(nil) != 0 {
		t.Error("nil should return 0")
	}
}

func TestRLComputeReward(t *testing.T) {
	bot := &SwarmBot{}
	bot.Stats.TotalDeliveries = 2
	bot.Stats.TotalPickups = 1
	bot.Stats.TicksAlive = 100
	bot.Stats.TicksIdle = 10
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	r := RLComputeReward(bot, ss)
	// 2*10 + 1*3 + 0.1 = 23.1
	if math.Abs(r-23.1) > 0.01 {
		t.Errorf("expected reward ~23.1, got %f", r)
	}
}

func TestRLStateCount(t *testing.T) {
	if rlNumStates != 96 {
		t.Errorf("expected 96 states, got %d", rlNumStates)
	}
}
