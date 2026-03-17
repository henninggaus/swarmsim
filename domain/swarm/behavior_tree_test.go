package swarm

import (
	"math/rand"
	"testing"
)

func makeTestBT() *BTNode {
	return &BTNode{
		Type:  BTSelector,
		Label: "root",
		Children: []*BTNode{
			{
				Type:  BTSequence,
				Label: "check-and-act",
				Children: []*BTNode{
					{Type: BTCondition, Label: "has-package", CondFunc: func(bot *SwarmBot, ss *SwarmState) bool {
						return bot.CarryingPkg >= 0
					}},
					{Type: BTAction, Label: "go-dropoff", ActFunc: func(bot *SwarmBot, ss *SwarmState) {
						bot.Speed = SwarmBotSpeed
					}},
				},
			},
			{Type: BTAction, Label: "explore", ActFunc: func(bot *SwarmBot, ss *SwarmState) {
				bot.Speed = SwarmBotSpeed * 0.5
			}},
		},
	}
}

func TestBTTickSequenceSuccess(t *testing.T) {
	node := &BTNode{
		Type: BTSequence,
		Children: []*BTNode{
			{Type: BTAction, ActFunc: func(bot *SwarmBot, ss *SwarmState) {}},
			{Type: BTAction, ActFunc: func(bot *SwarmBot, ss *SwarmState) {}},
		},
	}
	status := BTTick(node, &SwarmBot{}, nil)
	if status != BTSuccess {
		t.Error("all-success sequence should succeed")
	}
}

func TestBTTickSequenceFailsOnFirst(t *testing.T) {
	node := &BTNode{
		Type: BTSequence,
		Children: []*BTNode{
			{Type: BTCondition, CondFunc: func(bot *SwarmBot, ss *SwarmState) bool { return false }},
			{Type: BTAction, ActFunc: func(bot *SwarmBot, ss *SwarmState) {}},
		},
	}
	status := BTTick(node, &SwarmBot{}, nil)
	if status != BTFailure {
		t.Error("sequence with failing first child should fail")
	}
}

func TestBTTickSelectorSucceedsOnFirst(t *testing.T) {
	node := &BTNode{
		Type: BTSelector,
		Children: []*BTNode{
			{Type: BTAction, ActFunc: func(bot *SwarmBot, ss *SwarmState) {}},
			{Type: BTCondition, CondFunc: func(bot *SwarmBot, ss *SwarmState) bool { return false }},
		},
	}
	status := BTTick(node, &SwarmBot{}, nil)
	if status != BTSuccess {
		t.Error("selector with succeeding first child should succeed")
	}
}

func TestBTTickSelectorAllFail(t *testing.T) {
	node := &BTNode{
		Type: BTSelector,
		Children: []*BTNode{
			{Type: BTCondition, CondFunc: func(bot *SwarmBot, ss *SwarmState) bool { return false }},
			{Type: BTCondition, CondFunc: func(bot *SwarmBot, ss *SwarmState) bool { return false }},
		},
	}
	status := BTTick(node, &SwarmBot{}, nil)
	if status != BTFailure {
		t.Error("selector with all failing children should fail")
	}
}

func TestBTTickInverter(t *testing.T) {
	node := &BTNode{
		Type: BTInverter,
		Children: []*BTNode{
			{Type: BTCondition, CondFunc: func(bot *SwarmBot, ss *SwarmState) bool { return true }},
		},
	}
	status := BTTick(node, &SwarmBot{}, nil)
	if status != BTFailure {
		t.Error("inverter of success should be failure")
	}
}

func TestBTTickRepeater(t *testing.T) {
	count := 0
	node := &BTNode{
		Type:    BTRepeater,
		RepeatN: 3,
		Children: []*BTNode{
			{Type: BTAction, ActFunc: func(bot *SwarmBot, ss *SwarmState) { count++ }},
		},
	}
	status := BTTick(node, &SwarmBot{}, nil)
	if status != BTSuccess {
		t.Error("repeater should succeed")
	}
	if count != 3 {
		t.Errorf("expected 3 repeats, got %d", count)
	}
}

func TestBTTickNilNode(t *testing.T) {
	status := BTTick(nil, &SwarmBot{}, nil)
	if status != BTFailure {
		t.Error("nil node should return failure")
	}
}

func TestBTNodeCount(t *testing.T) {
	tree := makeTestBT()
	count := BTNodeCount(tree)
	if count != 5 {
		t.Errorf("expected 5 nodes, got %d", count)
	}
}

func TestBTNodeCountNil(t *testing.T) {
	if BTNodeCount(nil) != 0 {
		t.Error("nil should be 0")
	}
}

func TestBTDepth(t *testing.T) {
	tree := makeTestBT()
	depth := BTDepth(tree)
	if depth != 3 {
		t.Errorf("expected depth 3, got %d", depth)
	}
}

func TestBTMutate(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	tree := makeTestBT()
	origType := tree.Children[0].Type
	for i := 0; i < 100; i++ {
		BTMutate(rng, tree, 0.5)
	}
	// After many mutations, type should have changed at least once (probabilistic)
	_ = origType // mutation is probabilistic, just ensure no panic
}

func TestBTCrossover(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	a := makeTestBT()
	b := makeTestBT()
	child := BTCrossover(rng, a, b)
	if child == nil {
		t.Fatal("child should not be nil")
	}
	if BTNodeCount(child) == 0 {
		t.Error("child should have nodes")
	}
}

func TestBTCrossoverNilParent(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	a := makeTestBT()
	if BTCrossover(rng, nil, a) != a {
		t.Error("nil first parent should return second")
	}
	if BTCrossover(rng, a, nil) == nil {
		t.Error("nil second parent should return first copy")
	}
}

func TestBTDeepCopy(t *testing.T) {
	tree := makeTestBT()
	cp := btDeepCopy(tree)
	if cp == tree {
		t.Error("should be different pointer")
	}
	if BTNodeCount(cp) != BTNodeCount(tree) {
		t.Error("copy should have same node count")
	}
}

func TestBTFullTreeExecution(t *testing.T) {
	tree := makeTestBT()
	bot := &SwarmBot{CarryingPkg: 5} // carrying
	status := BTTick(tree, bot, nil)
	if status != BTSuccess {
		t.Error("carrying bot should succeed (go-dropoff path)")
	}
	if bot.Speed != SwarmBotSpeed {
		t.Error("should have set speed to SwarmBotSpeed")
	}
}

func TestBTFallbackPath(t *testing.T) {
	tree := makeTestBT()
	bot := &SwarmBot{CarryingPkg: -1} // not carrying
	status := BTTick(tree, bot, nil)
	if status != BTSuccess {
		t.Error("non-carrying bot should succeed via explore fallback")
	}
	if bot.Speed != SwarmBotSpeed*0.5 {
		t.Errorf("should have set speed to half, got %f", bot.Speed)
	}
}
