package swarmscript

import (
	"strings"
	"testing"
)

// --- Parse simple rule ---

func TestParseSimpleRule(t *testing.T) {
	prog, err := ParseSwarmScript("IF true THEN MOVE_FORWARD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(prog.Rules))
	}
	rule := prog.Rules[0]
	if len(rule.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(rule.Conditions))
	}
	if rule.Conditions[0].Type != CondTrue {
		t.Errorf("expected CondTrue, got %d", rule.Conditions[0].Type)
	}
	if rule.Action.Type != ActMoveForward {
		t.Errorf("expected ActMoveForward, got %d", rule.Action.Type)
	}
}

func TestParseRuleWithParams(t *testing.T) {
	prog, err := ParseSwarmScript("IF neighbors_count > 5 THEN TURN_LEFT 45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rule := prog.Rules[0]
	if rule.Conditions[0].Type != CondNeighborsCount {
		t.Errorf("expected CondNeighborsCount, got %d", rule.Conditions[0].Type)
	}
	if rule.Conditions[0].Op != OpGT {
		t.Errorf("expected OpGT, got %d", rule.Conditions[0].Op)
	}
	if rule.Conditions[0].Value != 5 {
		t.Errorf("expected value 5, got %d", rule.Conditions[0].Value)
	}
	if rule.Action.Type != ActTurnLeft {
		t.Errorf("expected ActTurnLeft, got %d", rule.Action.Type)
	}
	if rule.Action.Param1 != 45 {
		t.Errorf("expected param 45, got %d", rule.Action.Param1)
	}
}

func TestParseLessThanOperator(t *testing.T) {
	prog, err := ParseSwarmScript("IF nearest_distance < 20 THEN STOP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Op != OpLT {
		t.Errorf("expected OpLT, got %d", prog.Rules[0].Conditions[0].Op)
	}
}

func TestParseEqualOperator(t *testing.T) {
	prog, err := ParseSwarmScript("IF state == 1 THEN MOVE_FORWARD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Op != OpEQ {
		t.Errorf("expected OpEQ, got %d", prog.Rules[0].Conditions[0].Op)
	}
	if prog.Rules[0].Conditions[0].Value != 1 {
		t.Errorf("expected value 1, got %d", prog.Rules[0].Conditions[0].Value)
	}
}

func TestParseSingleEqualSign(t *testing.T) {
	prog, err := ParseSwarmScript("IF state = 1 THEN STOP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Op != OpEQ {
		t.Errorf("single = should be treated as ==, got op %d", prog.Rules[0].Conditions[0].Op)
	}
}

func TestParseBooleanValue(t *testing.T) {
	prog, err := ParseSwarmScript("IF on_edge == true THEN TURN_RIGHT 180")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Value != 1 {
		t.Errorf("'true' should parse as 1, got %d", prog.Rules[0].Conditions[0].Value)
	}
}

func TestParseBooleanFalseValue(t *testing.T) {
	prog, err := ParseSwarmScript("IF on_edge == false THEN MOVE_FORWARD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Value != 0 {
		t.Errorf("'false' should parse as 0, got %d", prog.Rules[0].Conditions[0].Value)
	}
}

// --- Condition AND ---

func TestParseConditionAND(t *testing.T) {
	prog, err := ParseSwarmScript("IF state == 0 AND neighbors_count > 3 THEN MOVE_FORWARD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rule := prog.Rules[0]
	if len(rule.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(rule.Conditions))
	}
	if rule.Conditions[0].Type != CondState {
		t.Errorf("first condition: expected CondState, got %d", rule.Conditions[0].Type)
	}
	if rule.Conditions[1].Type != CondNeighborsCount {
		t.Errorf("second condition: expected CondNeighborsCount, got %d", rule.Conditions[1].Type)
	}
}

func TestParseConditionANDCaseInsensitive(t *testing.T) {
	prog, err := ParseSwarmScript("IF state == 0 and timer == 0 THEN STOP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules[0].Conditions) != 2 {
		t.Error("lowercase 'and' should work as AND separator")
	}
}

// --- Aliases ---

func TestParseAllAliases(t *testing.T) {
	tests := []struct {
		alias    string
		expected ConditionType
	}{
		{"nbrs", CondNeighborsCount},
		{"rnd", CondRandom},
		{"carry", CondCarrying},
		{"near_dist", CondNearestDistance},
		{"p_dist", CondNearestPickupDist},
		{"d_dist", CondNearestDropoffDist},
		{"match", CondDropoffMatch},
		{"has_pkg", CondNearestPickupHasPkg},
		{"obs_ahead", CondObstacleAhead},
		{"msg", CondReceivedMessage},
		{"light", CondLightValue},
		{"edge", CondOnEdge},
		{"neighbors", CondNeighborsCount},
		{"leader", CondHasLeader},
		{"follower", CondHasFollower},
		{"chain_len", CondChainLength},
	}
	for _, tc := range tests {
		source := "IF " + tc.alias + " > 0 THEN STOP"
		prog, err := ParseSwarmScript(source)
		if err != nil {
			t.Errorf("alias '%s': unexpected error: %v", tc.alias, err)
			continue
		}
		if prog.Rules[0].Conditions[0].Type != tc.expected {
			t.Errorf("alias '%s': expected condition %d, got %d",
				tc.alias, tc.expected, prog.Rules[0].Conditions[0].Type)
		}
	}
}

func TestParseActionAliases(t *testing.T) {
	tests := []struct {
		alias    string
		expected ActionType
	}{
		{"FWD", ActMoveForward},
		{"FWD_SLOW", ActMoveForwardSlow},
		{"GOTO_MATCH", ActTurnToMatchingDropoff},
		{"GOTO_PICKUP", ActTurnToPickup},
		{"GOTO_DROPOFF", ActTurnToMatchingDropoff},
		{"AVOID_OBSTACLE", ActTurnAwayObstacle},
		{"COPY_LED", ActCopyNearestLED},
		{"LED_PICKUP", ActSetLEDPickupColor},
		{"LED_DROPOFF", ActSetLEDDropoffColor},
	}
	for _, tc := range tests {
		source := "IF true THEN " + tc.alias
		prog, err := ParseSwarmScript(source)
		if err != nil {
			t.Errorf("action alias '%s': unexpected error: %v", tc.alias, err)
			continue
		}
		if prog.Rules[0].Action.Type != tc.expected {
			t.Errorf("action alias '%s': expected action %d, got %d",
				tc.alias, tc.expected, prog.Rules[0].Action.Type)
		}
	}
}

// --- SET_LED with 3 params ---

func TestParseSetLED(t *testing.T) {
	prog, err := ParseSwarmScript("IF true THEN SET_LED 255 128 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	act := prog.Rules[0].Action
	if act.Type != ActSetLED {
		t.Errorf("expected ActSetLED, got %d", act.Type)
	}
	if act.Param1 != 255 || act.Param2 != 128 || act.Param3 != 0 {
		t.Errorf("expected params (255,128,0), got (%d,%d,%d)", act.Param1, act.Param2, act.Param3)
	}
}

// --- Two-word action names ---

func TestParseTwoWordAction(t *testing.T) {
	prog, err := ParseSwarmScript("IF true THEN MOVE FORWARD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Action.Type != ActMoveForward {
		t.Errorf("expected ActMoveForward for two-word 'MOVE FORWARD', got %d", prog.Rules[0].Action.Type)
	}
}

// --- Error handling ---

func TestParseError_Empty(t *testing.T) {
	_, err := ParseSwarmScript("")
	if err == nil {
		t.Fatal("expected error for empty program")
	}
	if !strings.Contains(err.Error(), "program is empty") {
		t.Errorf("expected 'program is empty' error, got: %v", err)
	}
}

func TestParseError_MissingTHEN(t *testing.T) {
	_, err := ParseSwarmScript("IF true MOVE_FORWARD")
	if err == nil {
		t.Fatal("expected error for missing THEN")
	}
	if !strings.Contains(err.Error(), "missing THEN") {
		t.Errorf("expected 'missing THEN' error, got: %v", err)
	}
}

func TestParseError_MissingAction(t *testing.T) {
	// "IF true THEN" after TrimSpace has no trailing space, so " THEN " won't match.
	// The parser reports "missing THEN keyword" because the pattern requires surrounding spaces.
	_, err := ParseSwarmScript("IF true THEN")
	if err == nil {
		t.Fatal("expected error for missing/dangling THEN")
	}
	if !strings.Contains(err.Error(), "missing THEN") {
		t.Errorf("expected 'missing THEN' error, got: %v", err)
	}
}

func TestParseEmptyAction(t *testing.T) {
	// Directly test parseLine to cover the "missing action after THEN" path.
	// ParseSwarmScript trims lines, making this unreachable via the public API,
	// but parseLine should still handle it gracefully.
	_, err := parseLine("IF true THEN  ", 1)
	if err == nil {
		t.Fatal("expected error for empty action")
	}
	if !strings.Contains(err.Error(), "missing action after THEN") {
		t.Errorf("expected 'missing action after THEN' error, got: %v", err)
	}
}

func TestParseError_UnknownSensor(t *testing.T) {
	_, err := ParseSwarmScript("IF foobar > 5 THEN STOP")
	if err == nil {
		t.Fatal("expected error for unknown sensor")
	}
	if !strings.Contains(err.Error(), "unknown sensor") {
		t.Errorf("expected 'unknown sensor' error, got: %v", err)
	}
}

func TestParseError_UnknownAction(t *testing.T) {
	_, err := ParseSwarmScript("IF true THEN EXPLODE")
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("expected 'unknown action' error, got: %v", err)
	}
}

func TestParseError_MissingParams(t *testing.T) {
	_, err := ParseSwarmScript("IF true THEN TURN_LEFT")
	if err == nil {
		t.Fatal("expected error for missing parameters")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Errorf("expected parameter requirement error, got: %v", err)
	}
}

func TestParseError_NonNumericParam(t *testing.T) {
	_, err := ParseSwarmScript("IF true THEN TURN_LEFT abc")
	if err == nil {
		t.Fatal("expected error for non-numeric parameter")
	}
	if !strings.Contains(err.Error(), "must be a number") {
		t.Errorf("expected 'must be a number' error, got: %v", err)
	}
}

func TestParseError_UnknownOperator(t *testing.T) {
	_, err := ParseSwarmScript("IF state != 1 THEN STOP")
	if err == nil {
		t.Fatal("expected error for unknown operator")
	}
	if !strings.Contains(err.Error(), "unknown operator") {
		t.Errorf("expected 'unknown operator' error, got: %v", err)
	}
}

func TestParseError_MissingIF(t *testing.T) {
	_, err := ParseSwarmScript("true THEN STOP")
	if err == nil {
		t.Fatal("expected error for missing IF")
	}
	if !strings.Contains(err.Error(), "expected IF") {
		t.Errorf("expected 'expected IF' error, got: %v", err)
	}
}

func TestParseError_MissingCondition(t *testing.T) {
	_, err := ParseSwarmScript("IF THEN STOP")
	if err == nil {
		t.Fatal("expected error for missing condition")
	}
	if !strings.Contains(err.Error(), "missing condition after IF") {
		t.Errorf("expected 'missing condition after IF' error, got: %v", err)
	}
}

func TestParseError_InvalidConditionFormat(t *testing.T) {
	_, err := ParseSwarmScript("IF state THEN STOP")
	if err == nil {
		t.Fatal("expected error for invalid condition format")
	}
	if !strings.Contains(err.Error(), "invalid condition") {
		t.Errorf("expected 'invalid condition' error, got: %v", err)
	}
}

func TestParseError_NonNumericCondValue(t *testing.T) {
	_, err := ParseSwarmScript("IF state > xyz THEN STOP")
	if err == nil {
		t.Fatal("expected error for non-numeric condition value")
	}
	if !strings.Contains(err.Error(), "expected number") {
		t.Errorf("expected 'expected number' error, got: %v", err)
	}
}

// --- Comment handling ---

func TestParseComment(t *testing.T) {
	source := `# This is a comment
IF true THEN STOP
# Another comment`

	prog, err := ParseSwarmScript(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules) != 1 {
		t.Errorf("expected 1 rule (comments skipped), got %d", len(prog.Rules))
	}
}

func TestParseEmptyLines(t *testing.T) {
	source := `
IF true THEN MOVE_FORWARD

IF state == 1 THEN STOP
`
	prog, err := ParseSwarmScript(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules) != 2 {
		t.Errorf("expected 2 rules (empty lines skipped), got %d", len(prog.Rules))
	}
}

func TestParseOnlyComments(t *testing.T) {
	_, err := ParseSwarmScript("# Only comments\n# No rules")
	if err == nil {
		t.Fatal("expected error for program with only comments")
	}
	if !strings.Contains(err.Error(), "program is empty") {
		t.Errorf("expected 'program is empty' error, got: %v", err)
	}
}

// --- Multi-rule programs ---

func TestParseMultipleRules(t *testing.T) {
	source := `IF near_dist < 15 THEN TURN_FROM_NEAREST
IF near_dist < 15 THEN FWD
IF true THEN FWD`
	prog, err := ParseSwarmScript(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(prog.Rules))
	}
}

// --- Line numbers ---

func TestParseLineNumbers(t *testing.T) {
	source := `# comment
IF true THEN STOP
IF state == 1 THEN FWD`
	prog, err := ParseSwarmScript(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Line != 2 {
		t.Errorf("first rule should be on line 2, got %d", prog.Rules[0].Line)
	}
	if prog.Rules[1].Line != 3 {
		t.Errorf("second rule should be on line 3, got %d", prog.Rules[1].Line)
	}
}

func TestParseErrorLineNumber(t *testing.T) {
	source := "IF true THEN STOP\nIF true THEN EXPLODE"
	_, err := ParseSwarmScript(source)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("expected error on line 2, got: %v", err)
	}
}
