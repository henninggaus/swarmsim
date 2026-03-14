package swarmscript

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseSwarmScript parses a SwarmScript source into a compiled program.
// Returns an error with line number if parsing fails.
func ParseSwarmScript(source string) (*SwarmProgram, error) {
	lines := strings.Split(source, "\n")
	prog := &SwarmProgram{}

	for lineNum, rawLine := range lines {
		lineNo := lineNum + 1 // 1-based
		line := strings.TrimSpace(rawLine)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule, err := parseLine(line, lineNo)
		if err != nil {
			return nil, err
		}
		prog.Rules = append(prog.Rules, rule)
	}

	if len(prog.Rules) == 0 {
		return nil, fmt.Errorf("line 1: program is empty — no rules defined")
	}

	return prog, nil
}

// parseLine parses a single IF ... THEN ... line.
func parseLine(line string, lineNo int) (Rule, error) {
	upper := strings.ToUpper(line)

	// Must start with IF
	if !strings.HasPrefix(upper, "IF ") {
		return Rule{}, fmt.Errorf("line %d: expected IF at start of rule", lineNo)
	}

	// Find THEN
	thenIdx := strings.Index(upper, " THEN ")
	if thenIdx < 0 {
		return Rule{}, fmt.Errorf("line %d: missing THEN keyword", lineNo)
	}

	condPart := strings.TrimSpace(line[3:thenIdx]) // after "IF ", before " THEN "
	actPart := strings.TrimSpace(line[thenIdx+6:]) // after " THEN "

	if condPart == "" {
		return Rule{}, fmt.Errorf("line %d: missing condition after IF", lineNo)
	}
	if actPart == "" {
		return Rule{}, fmt.Errorf("line %d: missing action after THEN", lineNo)
	}

	// Parse conditions (split by AND)
	conditions, err := parseConditions(condPart, lineNo)
	if err != nil {
		return Rule{}, err
	}

	// Parse action
	action, err := parseAction(actPart, lineNo)
	if err != nil {
		return Rule{}, err
	}

	return Rule{
		Conditions: conditions,
		Action:     action,
		Line:       lineNo,
	}, nil
}

// parseConditions parses the conditions part (between IF and THEN).
func parseConditions(condStr string, lineNo int) ([]Condition, error) {
	// Split by AND (case-insensitive)
	parts := splitByAND(condStr)
	var conditions []Condition

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		cond, err := parseSingleCondition(part, lineNo)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, cond)
	}

	if len(conditions) == 0 {
		return nil, fmt.Errorf("line %d: no valid conditions found", lineNo)
	}

	return conditions, nil
}

// splitByAND splits a condition string by "AND" (case-insensitive), preserving original case.
func splitByAND(s string) []string {
	upper := strings.ToUpper(s)
	var parts []string
	for {
		idx := strings.Index(upper, " AND ")
		if idx < 0 {
			parts = append(parts, strings.TrimSpace(s))
			break
		}
		parts = append(parts, strings.TrimSpace(s[:idx]))
		s = s[idx+5:]
		upper = upper[idx+5:]
	}
	return parts
}

// parseSingleCondition parses one condition like "neighbors_count > 5" or "true".
func parseSingleCondition(s string, lineNo int) (Condition, error) {
	s = strings.TrimSpace(s)

	// Special: "true" (always matches)
	if strings.ToLower(s) == "true" {
		return Condition{Type: CondTrue}, nil
	}

	// Tokenize: expect "sensor op value"
	tokens := strings.Fields(s)
	if len(tokens) < 3 {
		return Condition{}, fmt.Errorf("line %d: invalid condition '%s' (expected: sensor op value)", lineNo, s)
	}

	sensorName := strings.ToLower(tokens[0])
	opStr := tokens[1]
	valueStr := tokens[2]

	// Look up sensor
	condType, ok := conditionNames[sensorName]
	if !ok {
		return Condition{}, fmt.Errorf("line %d: unknown sensor '%s'", lineNo, tokens[0])
	}

	// Parse operator
	var op ConditionOp
	switch opStr {
	case ">":
		op = OpGT
	case "<":
		op = OpLT
	case "==", "=":
		op = OpEQ
	default:
		return Condition{}, fmt.Errorf("line %d: unknown operator '%s' (use >, <, or ==)", lineNo, opStr)
	}

	// Parse value
	val, err := strconv.Atoi(valueStr)
	if err != nil {
		// Handle "true"/"false" for on_edge
		if strings.ToLower(valueStr) == "true" {
			val = 1
		} else if strings.ToLower(valueStr) == "false" {
			val = 0
		} else {
			return Condition{}, fmt.Errorf("line %d: expected number but got '%s'", lineNo, valueStr)
		}
	}

	return Condition{Type: condType, Op: op, Value: val}, nil
}

// parseAction parses an action string like "MOVE_FORWARD" or "SET_LED 255 0 0".
func parseAction(s string, lineNo int) (Action, error) {
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return Action{}, fmt.Errorf("line %d: empty action", lineNo)
	}

	actionName := strings.ToUpper(tokens[0])

	// Handle two-word action names
	fullName := actionName
	paramStart := 1
	if len(tokens) > 1 {
		twoWord := actionName + "_" + strings.ToUpper(tokens[1])
		if _, ok := actionNames[twoWord]; ok {
			fullName = twoWord
			paramStart = 2
		}
	}

	info, ok := actionNames[fullName]
	if !ok {
		return Action{}, fmt.Errorf("line %d: unknown action '%s'", lineNo, tokens[0])
	}

	params := tokens[paramStart:]
	if len(params) < info.ParamCount {
		return Action{}, fmt.Errorf("line %d: %s requires %d parameter(s), got %d", lineNo, fullName, info.ParamCount, len(params))
	}

	act := Action{Type: info.Type}

	if info.ParamCount >= 1 {
		val, err := strconv.Atoi(params[0])
		if err != nil {
			return Action{}, fmt.Errorf("line %d: %s parameter must be a number, got '%s'", lineNo, fullName, params[0])
		}
		act.Param1 = val
	}
	if info.ParamCount >= 2 {
		val, err := strconv.Atoi(params[1])
		if err != nil {
			return Action{}, fmt.Errorf("line %d: %s parameter 2 must be a number, got '%s'", lineNo, fullName, params[1])
		}
		act.Param2 = val
	}
	if info.ParamCount >= 3 {
		val, err := strconv.Atoi(params[2])
		if err != nil {
			return Action{}, fmt.Errorf("line %d: %s parameter 3 must be a number, got '%s'", lineNo, fullName, params[2])
		}
		act.Param3 = val
	}

	return act, nil
}
