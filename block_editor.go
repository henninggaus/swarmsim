package main

import (
	"fmt"
	"strings"
	"swarmsim/domain/swarm"
	"swarmsim/engine/swarmscript"
)

func rulesToBlockRules(rules []swarmscript.Rule) []swarm.BlockRule {
	var result []swarm.BlockRule
	for _, r := range rules {
		br := swarm.BlockRule{
			ActionName:   swarmscript.ActionTypeName(r.Action.Type),
			ActionParams: [3]int{r.Action.Param1, r.Action.Param2, r.Action.Param3},
		}
		for _, c := range r.Conditions {
			br.Conditions = append(br.Conditions, swarm.BlockCondition{
				SensorName: swarmscript.ConditionTypeName(c.Type),
				OpStr:      swarmscript.OpString(c.Op),
				Value:      c.Value,
			})
		}
		if len(br.Conditions) == 0 {
			br.Conditions = append(br.Conditions, swarm.BlockCondition{
				SensorName: "true", OpStr: "==", Value: 1,
			})
		}
		result = append(result, br)
	}
	return result
}

func serializeBlockRules(blocks []swarm.BlockRule) string {
	var lines []string
	for _, br := range blocks {
		line := "IF "
		for i, cond := range br.Conditions {
			if i > 0 {
				line += " AND "
			}
			if cond.SensorName == "true" {
				line += "true"
			} else {
				line += fmt.Sprintf("%s %s %d", cond.SensorName, cond.OpStr, cond.Value)
			}
		}
		line += " THEN " + br.ActionName
		pc := swarmscript.ActionParamCountByName(br.ActionName)
		if pc >= 1 {
			line += fmt.Sprintf(" %d", br.ActionParams[0])
		}
		if pc >= 2 {
			line += fmt.Sprintf(" %d", br.ActionParams[1])
		}
		if pc >= 3 {
			line += fmt.Sprintf(" %d", br.ActionParams[2])
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func flattenGroups(groups [][]string) []string {
	var result []string
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}
