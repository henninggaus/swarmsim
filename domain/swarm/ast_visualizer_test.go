package swarm

import (
	"swarmsim/engine/swarmscript"
	"testing"
)

func makeTestProgram() *swarmscript.SwarmProgram {
	return &swarmscript.SwarmProgram{
		Rules: []swarmscript.Rule{
			{
				Conditions: []swarmscript.Condition{
					{Type: swarmscript.CondNearestDistance, Op: swarmscript.OpLT, Value: 50},
				},
				Action: swarmscript.Action{Type: swarmscript.ActTurnLeft, Param1: 30},
			},
			{
				Conditions: []swarmscript.Condition{
					{Type: swarmscript.CondNeighborsCount, Op: swarmscript.OpGT, Value: 3},
					{Type: swarmscript.CondOnEdge, Op: swarmscript.OpEQ, Value: 1},
				},
				Action: swarmscript.Action{Type: swarmscript.ActMoveForward},
			},
		},
	}
}

func TestBuildASTLayoutNil(t *testing.T) {
	if BuildASTLayout(nil) != nil {
		t.Error("nil program should return nil layout")
	}
	if BuildASTLayout(&swarmscript.SwarmProgram{}) != nil {
		t.Error("empty program should return nil layout")
	}
}

func TestBuildASTLayout(t *testing.T) {
	prog := makeTestProgram()
	layout := BuildASTLayout(prog)
	if layout == nil {
		t.Fatal("layout should not be nil")
	}
	if layout.Root == nil {
		t.Fatal("root should not be nil")
	}
	if layout.Root.Label != "PROGRAMM" {
		t.Errorf("root label should be PROGRAMM, got %s", layout.Root.Label)
	}
	if len(layout.Root.Children) != 2 {
		t.Errorf("expected 2 rule nodes, got %d", len(layout.Root.Children))
	}
}

func TestASTNodeCount(t *testing.T) {
	prog := makeTestProgram()
	layout := BuildASTLayout(prog)
	// Root + 2 rules + (1 cond + 1 action) + (2 cond + 1 action) = 1 + 2 + 2 + 3 = 8
	if layout.NodeCount != 8 {
		t.Errorf("expected 8 nodes, got %d", layout.NodeCount)
	}
}

func TestASTMaxDepth(t *testing.T) {
	prog := makeTestProgram()
	layout := BuildASTLayout(prog)
	if layout.MaxDepth != 2 {
		t.Errorf("expected max depth 2, got %d", layout.MaxDepth)
	}
}

func TestASTPositions(t *testing.T) {
	prog := makeTestProgram()
	layout := BuildASTLayout(prog)
	// All nodes should have been positioned
	nodes := CollectASTNodes(layout)
	for _, node := range nodes {
		if node.Width == 0 || node.Height == 0 {
			t.Errorf("node %s has zero dimensions", node.Label)
		}
	}
	// Root should be at depth 0
	if layout.Root.Y != 0 {
		t.Error("root Y should be 0")
	}
	// Children should be below root
	for _, child := range layout.Root.Children {
		if child.Y <= layout.Root.Y {
			t.Error("children should be below root")
		}
	}
}

func TestASTTotalDimensions(t *testing.T) {
	prog := makeTestProgram()
	layout := BuildASTLayout(prog)
	if layout.TotalWidth <= 0 {
		t.Error("total width should be positive")
	}
	if layout.TotalHeight <= 0 {
		t.Error("total height should be positive")
	}
}

func TestMarkActiveRules(t *testing.T) {
	prog := makeTestProgram()
	layout := BuildASTLayout(prog)
	MarkActiveRules(layout, []int{0})
	if !layout.Root.Children[0].Active {
		t.Error("rule 0 should be active")
	}
	if layout.Root.Children[1].Active {
		t.Error("rule 1 should not be active")
	}
}

func TestMarkActiveRulesNil(t *testing.T) {
	MarkActiveRules(nil, []int{0}) // should not panic
	MarkActiveRules(&ASTLayout{}, []int{0})
}

func TestMarkActiveRulesClear(t *testing.T) {
	prog := makeTestProgram()
	layout := BuildASTLayout(prog)
	MarkActiveRules(layout, []int{0, 1})
	MarkActiveRules(layout, []int{}) // clear all
	if layout.Root.Children[0].Active {
		t.Error("rule 0 should be cleared")
	}
}

func TestCollectASTNodes(t *testing.T) {
	prog := makeTestProgram()
	layout := BuildASTLayout(prog)
	nodes := CollectASTNodes(layout)
	if len(nodes) != layout.NodeCount {
		t.Errorf("collect should return %d nodes, got %d", layout.NodeCount, len(nodes))
	}
}

func TestCollectASTNodesNil(t *testing.T) {
	nodes := CollectASTNodes(nil)
	if nodes != nil {
		t.Error("nil layout should return nil")
	}
}

func TestConditionLabel(t *testing.T) {
	cond := swarmscript.Condition{Type: swarmscript.CondNearestDistance, Op: swarmscript.OpLT, Value: 50}
	label := conditionLabel(cond)
	expected := swarmscript.ConditionTypeName(swarmscript.CondNearestDistance) + " < 50"
	if label != expected {
		t.Errorf("unexpected label: %s (expected %s)", label, expected)
	}
}

func TestActionLabel(t *testing.T) {
	action := swarmscript.Action{Type: swarmscript.ActMoveForward}
	name := swarmscript.ActionTypeName(swarmscript.ActMoveForward)
	if actionLabel(action) != name {
		t.Errorf("expected %s, got %s", name, actionLabel(action))
	}
	action2 := swarmscript.Action{Type: swarmscript.ActTurnLeft, Param1: 30}
	expected := swarmscript.ActionTypeName(swarmscript.ActTurnLeft) + "(30)"
	if actionLabel(action2) != expected {
		t.Errorf("expected %s, got %s", expected, actionLabel(action2))
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 1: "1", 42: "42", -5: "-5", 100: "100"}
	for n, expected := range cases {
		if itoa(n) != expected {
			t.Errorf("itoa(%d) = %s, want %s", n, itoa(n), expected)
		}
	}
}
