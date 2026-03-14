package swarmscript

import (
	"fmt"
	"strings"
)

// SerializeRule converts a Rule back to SwarmScript text.
func SerializeRule(r Rule) string {
	var parts []string
	for i, cond := range r.Conditions {
		if i > 0 {
			parts = append(parts, "AND")
		}
		if cond.Type == CondTrue {
			parts = append(parts, "true")
		} else {
			parts = append(parts, fmt.Sprintf("%s %s %d",
				ConditionTypeName(cond.Type),
				OpString(cond.Op),
				cond.Value))
		}
	}

	condStr := strings.Join(parts, " ")
	actStr := ActionTypeName(r.Action.Type)
	pc := ActionParamCount(r.Action.Type)
	if pc >= 1 {
		actStr += fmt.Sprintf(" %d", r.Action.Param1)
	}
	if pc >= 2 {
		actStr += fmt.Sprintf(" %d", r.Action.Param2)
	}
	if pc >= 3 {
		actStr += fmt.Sprintf(" %d", r.Action.Param3)
	}

	return fmt.Sprintf("IF %s THEN %s", condStr, actStr)
}

// SerializeProgram converts a SwarmProgram back to SwarmScript text.
func SerializeProgram(prog *SwarmProgram) string {
	var lines []string
	for _, rule := range prog.Rules {
		lines = append(lines, SerializeRule(rule))
	}
	return strings.Join(lines, "\n")
}
