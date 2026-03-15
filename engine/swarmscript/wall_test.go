package swarmscript

import "testing"

// --- Wall sensor parsing ---

func TestParseWallRight(t *testing.T) {
	prog, err := ParseSwarmScript("IF wall_right == 1 THEN TURN_LEFT 90")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(prog.Rules))
	}
	if prog.Rules[0].Conditions[0].Type != CondWallRight {
		t.Errorf("expected CondWallRight, got %d", prog.Rules[0].Conditions[0].Type)
	}
}

func TestParseWallLeft(t *testing.T) {
	prog, err := ParseSwarmScript("IF wall_left == 1 THEN TURN_RIGHT 90")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Type != CondWallLeft {
		t.Errorf("expected CondWallLeft, got %d", prog.Rules[0].Conditions[0].Type)
	}
}

func TestParseWallFrontAlias(t *testing.T) {
	prog, err := ParseSwarmScript("IF wall_front == 1 THEN AVOID_OBSTACLE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Type != CondObstacleAhead {
		t.Errorf("expected CondObstacleAhead (wall_front alias), got %d", prog.Rules[0].Conditions[0].Type)
	}
}

// --- Wall follow action parsing ---

func TestParseWallFollowRight(t *testing.T) {
	prog, err := ParseSwarmScript("IF true THEN WALL_FOLLOW_RIGHT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Action.Type != ActWallFollowRight {
		t.Errorf("expected ActWallFollowRight, got %d", prog.Rules[0].Action.Type)
	}
}

func TestParseWallFollowLeft(t *testing.T) {
	prog, err := ParseSwarmScript("IF true THEN WALL_FOLLOW_LEFT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Action.Type != ActWallFollowLeft {
		t.Errorf("expected ActWallFollowLeft, got %d", prog.Rules[0].Action.Type)
	}
}

// --- Combined wall-following program ---

func TestParseMazeExplorerProgram(t *testing.T) {
	prog, err := ParseSwarmScript(`
# Maze Explorer
IF wall_front == 1 THEN TURN_LEFT 90
IF wall_right == 0 THEN TURN_RIGHT 90
IF true THEN FWD
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(prog.Rules))
	}
	// Rule 1: wall_front -> TURN_LEFT 90
	if prog.Rules[0].Conditions[0].Type != CondObstacleAhead {
		t.Errorf("rule 1: expected wall_front (CondObstacleAhead), got %d", prog.Rules[0].Conditions[0].Type)
	}
	if prog.Rules[0].Action.Type != ActTurnLeft {
		t.Errorf("rule 1: expected ActTurnLeft, got %d", prog.Rules[0].Action.Type)
	}
	if prog.Rules[0].Action.Param1 != 90 {
		t.Errorf("rule 1: expected param 90, got %d", prog.Rules[0].Action.Param1)
	}
	// Rule 2: wall_right == 0 -> TURN_RIGHT 90
	if prog.Rules[1].Conditions[0].Type != CondWallRight {
		t.Errorf("rule 2: expected CondWallRight, got %d", prog.Rules[1].Conditions[0].Type)
	}
	if prog.Rules[1].Conditions[0].Value != 0 {
		t.Errorf("rule 2: expected value 0, got %d", prog.Rules[1].Conditions[0].Value)
	}
}
