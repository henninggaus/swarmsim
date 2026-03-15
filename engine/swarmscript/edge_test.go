package swarmscript

import (
	"strings"
	"testing"
)

// --- Parser edge cases ---

func TestParseEvolutionParam(t *testing.T) {
	prog, err := ParseSwarmScript("IF d_dist < $A:25 THEN DROP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(prog.Rules))
	}
	cond := prog.Rules[0].Conditions[0]
	if !cond.IsParamRef {
		t.Error("condition should be marked as parameterized")
	}
	if cond.ParamIdx != 0 { // $A = index 0
		t.Errorf("expected param index 0 ($A), got %d", cond.ParamIdx)
	}
	if cond.Value != 25 {
		t.Errorf("expected default value 25, got %d", cond.Value)
	}
}

func TestParseEvolutionParamZ(t *testing.T) {
	prog, err := ParseSwarmScript("IF rnd < $Z:99 THEN FWD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cond := prog.Rules[0].Conditions[0]
	if !cond.IsParamRef {
		t.Error("condition should be parameterized")
	}
	if cond.ParamIdx != 25 { // $Z = index 25
		t.Errorf("expected param index 25 ($Z), got %d", cond.ParamIdx)
	}
}

func TestParsePheromoneCondition(t *testing.T) {
	prog, err := ParseSwarmScript("IF pheromone > 50 THEN FOLLOW_PHER")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Type != CondPherAhead {
		t.Errorf("expected CondPherAhead, got %d", prog.Rules[0].Conditions[0].Type)
	}
	if prog.Rules[0].Action.Type != ActFollowPheromone {
		t.Errorf("expected ActFollowPheromone, got %d", prog.Rules[0].Action.Type)
	}
}

func TestParsePherAlias(t *testing.T) {
	prog, err := ParseSwarmScript("IF pher > 10 THEN GOTO_PHER")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Type != CondPherAhead {
		t.Errorf("expected CondPherAhead, got %d", prog.Rules[0].Conditions[0].Type)
	}
	if prog.Rules[0].Action.Type != ActFollowPheromone {
		t.Errorf("expected ActFollowPheromone, got %d", prog.Rules[0].Action.Type)
	}
}

func TestParseAllDeliveryActions(t *testing.T) {
	actions := []struct {
		name string
		typ  ActionType
	}{
		{"PICKUP", ActPickup},
		{"DROP", ActDrop},
		{"GOTO_PICKUP", ActTurnToPickup},
		{"GOTO_DROPOFF", ActTurnToMatchingDropoff},
		{"LED_PICKUP", ActSetLEDPickupColor},
		{"LED_DROPOFF", ActSetLEDDropoffColor},
		{"GOTO_BEACON", ActTurnToBeaconDropoff},
		{"GOTO_LED", ActTurnToMatchingLED},
		{"GOTO_LED_MATCH", ActTurnToMatchingLED},
		{"GOTO_RAMP", ActTurnToRamp},
	}
	for _, tc := range actions {
		prog, err := ParseSwarmScript("IF true THEN " + tc.name)
		if err != nil {
			t.Errorf("failed to parse %s: %v", tc.name, err)
			continue
		}
		if prog.Rules[0].Action.Type != tc.typ {
			t.Errorf("%s: expected action type %d, got %d", tc.name, tc.typ, prog.Rules[0].Action.Type)
		}
	}
}

func TestParseAllDeliveryConditions(t *testing.T) {
	conditions := []struct {
		name string
		typ  ConditionType
	}{
		{"carry", CondCarrying},
		{"match", CondDropoffMatch},
		{"d_dist", CondNearestDropoffDist},
		{"p_dist", CondNearestPickupDist},
		{"has_pkg", CondNearestPickupHasPkg},
		{"heard_pickup", CondHeardPickupColor},
		{"heard_dropoff", CondHeardDropoffColor},
		{"heard_beacon", CondHeardBeaconDropoff},
		{"led_dist", CondNearestMatchLEDDist},
		{"led_match", CondNearestMatchLEDDist},
		{"on_ramp", CondOnRamp},
		{"truck_here", CondTruckHere},
		{"exploring", CondExploring},
	}
	for _, tc := range conditions {
		prog, err := ParseSwarmScript("IF " + tc.name + " == 1 THEN FWD")
		if err != nil {
			t.Errorf("failed to parse condition %s: %v", tc.name, err)
			continue
		}
		if prog.Rules[0].Conditions[0].Type != tc.typ {
			t.Errorf("%s: expected condition type %d, got %d", tc.name, tc.typ, prog.Rules[0].Conditions[0].Type)
		}
	}
}

func TestParseMultipleANDConditions(t *testing.T) {
	prog, err := ParseSwarmScript("IF carry == 1 AND match == 1 AND d_dist < 25 THEN DROP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prog.Rules[0].Conditions) != 3 {
		t.Fatalf("expected 3 conditions, got %d", len(prog.Rules[0].Conditions))
	}
	if prog.Rules[0].Conditions[0].Type != CondCarrying {
		t.Error("first condition should be CondCarrying")
	}
	if prog.Rules[0].Conditions[1].Type != CondDropoffMatch {
		t.Error("second condition should be CondDropoffMatch")
	}
	if prog.Rules[0].Conditions[2].Type != CondNearestDropoffDist {
		t.Error("third condition should be CondNearestDropoffDist")
	}
}

func TestParseWhitespaceVariants(t *testing.T) {
	prog, err := ParseSwarmScript("  IF   true   THEN   FWD  ")
	if err != nil {
		t.Fatalf("extra spaces should parse: %v", err)
	}
	if len(prog.Rules) != 1 {
		t.Fatal("should parse 1 rule")
	}
}

func TestParseAllPresets(t *testing.T) {
	presets := []string{
		"IF true THEN FWD",
		"IF near_dist < 12 THEN TURN_FROM_NEAREST\nIF true THEN FWD",
		"IF carry == 1 AND match == 1 AND d_dist < 25 THEN DROP\nIF carry == 0 AND p_dist < 20 THEN PICKUP",
		"IF wall_front == 1 THEN TURN_LEFT 90\nIF wall_right == 0 THEN TURN_RIGHT 90\nIF true THEN FWD",
		"IF carry == 0 AND on_ramp == 1 AND truck_here == 1 THEN PICKUP",
	}
	for i, p := range presets {
		_, err := ParseSwarmScript(p)
		if err != nil {
			t.Errorf("preset %d failed to parse: %v", i, err)
		}
	}
}

func TestParseErrorBadParam(t *testing.T) {
	_, err := ParseSwarmScript("IF d_dist < $1:25 THEN DROP")
	if err == nil {
		t.Error("expected error for invalid param $1")
	}
}

func TestParseSendPickupAction(t *testing.T) {
	prog, err := ParseSwarmScript("IF has_pkg == 1 THEN SEND_PICKUP 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Action.Type != ActSendPickup {
		t.Errorf("expected ActSendPickup, got %d", prog.Rules[0].Action.Type)
	}
}

func TestParseSendDropoffAction(t *testing.T) {
	prog, err := ParseSwarmScript("IF d_dist < 200 THEN SEND_DROPOFF 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Action.Type != ActSendDropoff {
		t.Errorf("expected ActSendDropoff, got %d", prog.Rules[0].Action.Type)
	}
}

func TestParseSpiralAction(t *testing.T) {
	prog, err := ParseSwarmScript("IF exploring == 1 THEN SPIRAL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Action.Type != ActSpiralFwd {
		t.Errorf("expected ActSpiralFwd, got %d", prog.Rules[0].Action.Type)
	}
}

func TestParseCaseInsensitiveIF(t *testing.T) {
	_, err := ParseSwarmScript("if true then FWD")
	if err != nil {
		t.Fatalf("case-insensitive IF/THEN should parse: %v", err)
	}
}

func TestParseOperatorGT(t *testing.T) {
	prog, err := ParseSwarmScript("IF d_dist > 100 THEN FWD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Op != OpGT {
		t.Errorf("expected OpGT, got %d", prog.Rules[0].Conditions[0].Op)
	}
}

func TestParseOperatorLT(t *testing.T) {
	prog, err := ParseSwarmScript("IF d_dist < 30 THEN DROP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Op != OpLT {
		t.Errorf("expected OpLT, got %d", prog.Rules[0].Conditions[0].Op)
	}
}

func TestParseOperatorEQ(t *testing.T) {
	prog, err := ParseSwarmScript("IF state == 0 THEN FWD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prog.Rules[0].Conditions[0].Op != OpEQ {
		t.Errorf("expected OpEQ, got %d", prog.Rules[0].Conditions[0].Op)
	}
}

func TestParseLargeProgram(t *testing.T) {
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "IF true THEN FWD")
	}
	prog, err := ParseSwarmScript(strings.Join(lines, "\n"))
	if err != nil {
		t.Fatalf("large program should parse: %v", err)
	}
	if len(prog.Rules) != 100 {
		t.Errorf("expected 100 rules, got %d", len(prog.Rules))
	}
}
