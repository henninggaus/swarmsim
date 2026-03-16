package swarmscript

import (
	"math/rand"
)

// gpSensorPool defines sensors available for random program generation.
// Each entry: condition type, min value, max value.
type gpSensorEntry struct {
	Cond ConditionType
	Min  int
	Max  int
}

var gpSensorPool = []gpSensorEntry{
	{CondNearestDistance, 5, 200},
	{CondNeighborsCount, 0, 10},
	{CondCarrying, 0, 1},
	{CondDropoffMatch, 0, 1},
	{CondNearestPickupDist, 5, 300},
	{CondNearestDropoffDist, 5, 300},
	{CondNearestPickupHasPkg, 0, 1},
	{CondObstacleAhead, 0, 1},
	{CondOnEdge, 0, 1},
	{CondRandom, 0, 100},
	{CondWallRight, 0, 1},
	{CondObstacleAhead, 0, 1}, // wall_front alias
	{CondOnRamp, 0, 1},
	{CondTruckHere, 0, 1},
	{CondBotAhead, 0, 5},
	{CondBotBehind, 0, 5},
	{CondBotLeft, 0, 5},
	{CondBotRight, 0, 5},
	{CondVisitedHere, 0, 5},
	{CondVisitedAhead, 0, 5},
	{CondExplored, 0, 100},
	{CondGroupCarry, 0, 100},
	{CondGroupSpeed, 0, 200},
	{CondGroupSize, 0, 20},
}

// gpActionEntry defines an action template for random generation.
type gpActionEntry struct {
	Type   ActionType
	Param1 int
	Param2 int
	Param3 int
}

var gpActionPool = []gpActionEntry{
	{ActMoveForward, 0, 0, 0},
	{ActMoveForwardSlow, 0, 0, 0},
	{ActStop, 0, 0, 0},
	{ActTurnLeft, 30, 0, 0},
	{ActTurnLeft, 90, 0, 0},
	{ActTurnRight, 30, 0, 0},
	{ActTurnRight, 90, 0, 0},
	{ActTurnRandom, 0, 0, 0},
	{ActTurnFromNearest, 0, 0, 0},
	{ActTurnToNearest, 0, 0, 0},
	{ActTurnToPickup, 0, 0, 0},
	{ActTurnToMatchingDropoff, 0, 0, 0},
	{ActTurnToRamp, 0, 0, 0},
	{ActPickup, 0, 0, 0},
	{ActDrop, 0, 0, 0},
	{ActTurnAwayObstacle, 0, 0, 0},
	{ActWallFollowRight, 0, 0, 0},
	{ActSpiralFwd, 0, 0, 0},
	{ActSetLED, 255, 0, 0},   // red
	{ActSetLED, 0, 255, 0},   // green
	{ActSetLED, 0, 0, 255},   // blue
	{ActSendMessage, 1, 0, 0},
	{ActSendPickup, 1, 0, 0},
	{ActSendDropoff, 1, 0, 0},
}

// GenerateRandomProgram creates a random SwarmScript program with numRules rules.
// The last rule is always IF true THEN FWD as a fallback.
func GenerateRandomProgram(rng *rand.Rand, numRules int) *SwarmProgram {
	if numRules < 3 {
		numRules = 3
	}
	rules := make([]Rule, 0, numRules)

	for i := 0; i < numRules-1; i++ {
		rule := generateRandomRule(rng)
		rule.Line = i + 1
		rules = append(rules, rule)
	}

	// Fallback rule: IF true THEN FWD
	rules = append(rules, Rule{
		Conditions: []Condition{{Type: CondTrue}},
		Action:     Action{Type: ActMoveForward},
		Line:       numRules,
	})

	return &SwarmProgram{Rules: rules}
}

// generateRandomRule creates a single random rule with 1-2 conditions and 1 action.
func generateRandomRule(rng *rand.Rand) Rule {
	// 1-2 conditions
	numConds := 1
	if rng.Float64() < 0.4 {
		numConds = 2
	}

	conds := make([]Condition, numConds)
	for c := 0; c < numConds; c++ {
		sensor := gpSensorPool[rng.Intn(len(gpSensorPool))]
		op := ConditionOp(rng.Intn(3)) // OpGT, OpLT, OpEQ

		// Generate value in context-appropriate range
		val := sensor.Min + rng.Intn(sensor.Max-sensor.Min+1)

		conds[c] = Condition{
			Type:  sensor.Cond,
			Op:    op,
			Value: val,
		}
	}

	// Random action
	act := gpActionPool[rng.Intn(len(gpActionPool))]

	return Rule{
		Conditions: conds,
		Action:     Action{Type: act.Type, Param1: act.Param1, Param2: act.Param2, Param3: act.Param3},
	}
}

// CrossoverPrograms combines two parent programs into a child program.
// Takes first half of rules from parent A, second half from parent B.
func CrossoverPrograms(rng *rand.Rand, a, b *SwarmProgram) *SwarmProgram {
	if a == nil || b == nil || len(a.Rules) == 0 || len(b.Rules) == 0 {
		return GenerateRandomProgram(rng, 10)
	}

	splitA := len(a.Rules) / 2
	splitB := len(b.Rules) / 2

	rules := make([]Rule, 0, splitA+len(b.Rules)-splitB)

	// First half from parent A
	for i := 0; i < splitA && i < len(a.Rules); i++ {
		rules = append(rules, copyRule(a.Rules[i]))
	}

	// Second half from parent B
	for i := splitB; i < len(b.Rules); i++ {
		rules = append(rules, copyRule(b.Rules[i]))
	}

	// Clamp to 5-20 rules
	if len(rules) < 5 {
		for len(rules) < 5 {
			rule := generateRandomRule(rng)
			rules = append(rules, rule)
		}
	}
	if len(rules) > 20 {
		rules = rules[:20]
	}

	// Re-number lines
	for i := range rules {
		rules[i].Line = i + 1
	}

	return &SwarmProgram{Rules: rules}
}

// MutateProgram applies random mutations to a program in-place.
func MutateProgram(rng *rand.Rand, p *SwarmProgram) {
	if p == nil || len(p.Rules) == 0 {
		return
	}

	for i := range p.Rules {
		if i == len(p.Rules)-1 {
			// Don't mutate fallback rule
			continue
		}

		// 20% chance to mutate each rule
		if rng.Float64() >= 0.20 {
			continue
		}

		roll := rng.Float64()
		switch {
		case roll < 0.30:
			// Sensor mutation: replace a condition's sensor type
			if len(p.Rules[i].Conditions) > 0 {
				ci := rng.Intn(len(p.Rules[i].Conditions))
				sensor := gpSensorPool[rng.Intn(len(gpSensorPool))]
				p.Rules[i].Conditions[ci].Type = sensor.Cond
				p.Rules[i].Conditions[ci].Value = sensor.Min + rng.Intn(sensor.Max-sensor.Min+1)
			}
		case roll < 0.55:
			// Value mutation: adjust comparison value by ±10-30%
			if len(p.Rules[i].Conditions) > 0 {
				ci := rng.Intn(len(p.Rules[i].Conditions))
				v := p.Rules[i].Conditions[ci].Value
				delta := int(float64(v) * (0.1 + rng.Float64()*0.2))
				if delta < 1 {
					delta = 1
				}
				if rng.Float64() < 0.5 {
					v += delta
				} else {
					v -= delta
				}
				if v < 0 {
					v = 0
				}
				if v > 500 {
					v = 500
				}
				p.Rules[i].Conditions[ci].Value = v
			}
		case roll < 0.80:
			// Action mutation: replace action
			act := gpActionPool[rng.Intn(len(gpActionPool))]
			p.Rules[i].Action = Action{Type: act.Type, Param1: act.Param1, Param2: act.Param2, Param3: act.Param3}
		case roll < 0.90:
			// Rule swap: swap with another random rule
			j := rng.Intn(len(p.Rules) - 1) // exclude fallback
			p.Rules[i], p.Rules[j] = p.Rules[j], p.Rules[i]
		default:
			// Operator mutation: change comparison operator
			if len(p.Rules[i].Conditions) > 0 {
				ci := rng.Intn(len(p.Rules[i].Conditions))
				p.Rules[i].Conditions[ci].Op = ConditionOp(rng.Intn(3))
			}
		}
	}

	// 5% chance to insert a new random rule
	if rng.Float64() < 0.05 && len(p.Rules) < 20 {
		rule := generateRandomRule(rng)
		// Insert before fallback
		idx := len(p.Rules) - 1
		p.Rules = append(p.Rules, Rule{})
		copy(p.Rules[idx+1:], p.Rules[idx:])
		p.Rules[idx] = rule
	}

	// 5% chance to delete a rule (keep min 5)
	if rng.Float64() < 0.05 && len(p.Rules) > 5 {
		idx := rng.Intn(len(p.Rules) - 1) // exclude fallback
		p.Rules = append(p.Rules[:idx], p.Rules[idx+1:]...)
	}

	// Re-number lines
	for i := range p.Rules {
		p.Rules[i].Line = i + 1
	}
}

// copyRule creates a deep copy of a rule.
func copyRule(r Rule) Rule {
	conds := make([]Condition, len(r.Conditions))
	copy(conds, r.Conditions)
	return Rule{
		Conditions: conds,
		Action:     r.Action,
		Line:       r.Line,
	}
}

// CopyProgram creates a deep copy of a SwarmProgram.
func CopyProgram(p *SwarmProgram) *SwarmProgram {
	if p == nil {
		return nil
	}
	rules := make([]Rule, len(p.Rules))
	for i, r := range p.Rules {
		rules[i] = copyRule(r)
	}
	return &SwarmProgram{Rules: rules}
}

// ProgramToText converts a SwarmProgram back to human-readable SwarmScript text.
func ProgramToText(p *SwarmProgram) string {
	if p == nil || len(p.Rules) == 0 {
		return "# Empty program\nIF true THEN FWD"
	}

	text := "# GP-evolved program\n"
	for _, rule := range p.Rules {
		line := "IF "
		for ci, cond := range rule.Conditions {
			if ci > 0 {
				line += " AND "
			}
			line += conditionToText(cond)
		}
		line += " THEN " + actionToText(rule.Action)
		text += line + "\n"
	}
	return text
}

// conditionToText converts a condition to text.
func conditionToText(c Condition) string {
	name := condTypeName(c.Type)
	opStr := ">"
	switch c.Op {
	case OpLT:
		opStr = "<"
	case OpEQ:
		opStr = "=="
	}
	if c.Type == CondTrue {
		return "true"
	}
	return name + " " + opStr + " " + itoa(c.Value)
}

// actionToText converts an action to text.
func actionToText(a Action) string {
	name := actTypeName(a.Type)
	switch a.Type {
	case ActTurnLeft, ActTurnRight:
		return name + " " + itoa(a.Param1)
	case ActSetLED:
		return name + " " + itoa(a.Param1) + " " + itoa(a.Param2) + " " + itoa(a.Param3)
	case ActSendMessage, ActSendPickup, ActSendDropoff, ActSetState, ActSetCounter,
		ActSetValue1, ActSetValue2, ActSetTimer:
		return name + " " + itoa(a.Param1)
	}
	return name
}

// condTypeName returns a human-readable name for a condition type.
func condTypeName(t ConditionType) string {
	switch t {
	case CondNearestDistance:
		return "near_dist"
	case CondNeighborsCount:
		return "neighbors"
	case CondCarrying:
		return "carry"
	case CondDropoffMatch:
		return "match"
	case CondNearestPickupDist:
		return "p_dist"
	case CondNearestDropoffDist:
		return "d_dist"
	case CondNearestPickupHasPkg:
		return "has_pkg"
	case CondObstacleAhead:
		return "obs_ahead"
	case CondOnEdge:
		return "edge"
	case CondRandom:
		return "rnd"
	case CondWallRight:
		return "wall_right"
	case CondWallLeft:
		return "wall_left"
	case CondOnRamp:
		return "on_ramp"
	case CondTruckHere:
		return "truck_here"
	case CondTruckPkgCount:
		return "truck_pkg"
	case CondState:
		return "state"
	case CondCounter:
		return "counter"
	case CondTimer:
		return "timer"
	case CondTrue:
		return "true"
	case CondReceivedMessage:
		return "msg"
	case CondLightValue:
		return "light"
	case CondExploring:
		return "exploring"
	case CondHeardPickupColor:
		return "heard_pickup"
	case CondHeardDropoffColor:
		return "heard_dropoff"
	case CondPherAhead:
		return "pheromone"
	case CondBotAhead:
		return "bot_ahead"
	case CondBotBehind:
		return "bot_behind"
	case CondBotLeft:
		return "bot_left"
	case CondBotRight:
		return "bot_right"
	case CondHeading:
		return "heading"
	case CondSpeed:
		return "speed"
	case CondVisitedHere:
		return "visited_here"
	case CondVisitedAhead:
		return "visited_ahead"
	case CondExplored:
		return "explored"
	case CondGroupCarry:
		return "group_carry"
	case CondGroupSpeed:
		return "group_speed"
	case CondGroupSize:
		return "group_size"
	default:
		return "true"
	}
}

// actTypeName returns a human-readable name for an action type.
func actTypeName(t ActionType) string {
	switch t {
	case ActMoveForward:
		return "FWD"
	case ActMoveForwardSlow:
		return "FWD_SLOW"
	case ActStop:
		return "STOP"
	case ActTurnLeft:
		return "TURN_LEFT"
	case ActTurnRight:
		return "TURN_RIGHT"
	case ActTurnRandom:
		return "TURN_RANDOM"
	case ActTurnFromNearest:
		return "TURN_FROM_NEAREST"
	case ActTurnToNearest:
		return "TURN_TO_NEAREST"
	case ActTurnToCenter:
		return "TURN_TO_CENTER"
	case ActTurnToLight:
		return "TURN_TO_LIGHT"
	case ActTurnToPickup:
		return "GOTO_PICKUP"
	case ActTurnToMatchingDropoff:
		return "GOTO_DROPOFF"
	case ActTurnToRamp:
		return "GOTO_RAMP"
	case ActPickup:
		return "PICKUP"
	case ActDrop:
		return "DROP"
	case ActTurnAwayObstacle:
		return "AVOID_OBSTACLE"
	case ActWallFollowRight:
		return "WALL_FOLLOW_RIGHT"
	case ActWallFollowLeft:
		return "WALL_FOLLOW_LEFT"
	case ActSpiralFwd:
		return "SPIRAL"
	case ActSetLED:
		return "SET_LED"
	case ActSendMessage:
		return "SEND_MESSAGE"
	case ActSendPickup:
		return "SEND_PICKUP"
	case ActSendDropoff:
		return "SEND_DROPOFF"
	case ActFollowPheromone:
		return "FOLLOW_PHER"
	case ActSetState:
		return "SET_STATE"
	case ActSetCounter:
		return "SET_COUNTER"
	case ActIncCounter:
		return "INC_COUNTER"
	case ActDecCounter:
		return "DEC_COUNTER"
	case ActSetValue1:
		return "SET_VALUE1"
	case ActSetValue2:
		return "SET_VALUE2"
	case ActSetTimer:
		return "SET_TIMER"
	case ActFollowNearest:
		return "FOLLOW_NEAREST"
	case ActUnfollow:
		return "UNFOLLOW"
	case ActCopyNearestLED:
		return "COPY_NEAREST_LED"
	case ActTurnToHeardPickup:
		return "GOTO_HEARD_PICKUP"
	case ActTurnToHeardDropoff:
		return "GOTO_HEARD_DROPOFF"
	case ActTurnToMatchingLED:
		return "GOTO_MATCH"
	case ActTurnToTruckPkg:
		return "GOTO_TRUCK_PKG"
	case ActTurnToBeaconDropoff:
		return "GOTO_BEACON"
	case ActSetLEDPickupColor:
		return "SET_LED_PICKUP_COLOR"
	case ActSetLEDDropoffColor:
		return "SET_LED_DROPOFF_COLOR"
	default:
		return "FWD"
	}
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	if neg {
		digits = append(digits, '-')
	}
	// reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

// RuleToShortText converts a rule to a compact text representation for display.
func RuleToShortText(r *Rule) string {
	line := ""
	for ci, cond := range r.Conditions {
		if ci > 0 {
			line += " & "
		}
		line += conditionToText(cond)
	}
	line += " > " + actionToText(r.Action)
	return line
}
