package bot

import "math"

// FormationType identifies the formation shape.
type FormationType int

const (
	FormationCircle FormationType = iota
	FormationLine
	FormationV
)

// FormationSlotPos returns the world position for a given slot in a formation
// centered at (cx, cy) with the given heading angle.
func FormationSlotPos(ftype FormationType, slot int, cx, cy, heading float64, spacing float64) (float64, float64) {
	switch ftype {
	case FormationCircle:
		n := 8 // max slots
		angle := heading + float64(slot)*2*math.Pi/float64(n)
		return cx + math.Cos(angle)*spacing, cy + math.Sin(angle)*spacing
	case FormationLine:
		offset := float64(slot-3) * spacing * 0.5
		perpX := -math.Sin(heading)
		perpY := math.Cos(heading)
		return cx + perpX*offset, cy + perpY*offset
	case FormationV:
		side := 1.0
		if slot%2 == 1 {
			side = -1.0
		}
		rank := float64(slot/2 + 1)
		dx := -math.Cos(heading) * rank * spacing * 0.5
		dy := -math.Sin(heading) * rank * spacing * 0.5
		perpX := -math.Sin(heading) * side * rank * spacing * 0.3
		perpY := math.Cos(heading) * side * rank * spacing * 0.3
		return cx + dx + perpX, cy + dy + perpY
	}
	return cx, cy
}

// SteerToFormationSlot returns a steering vector for a bot to reach its formation slot.
func SteerToFormationSlot(b Bot, slotX, slotY float64) Vec2 {
	target := Vec2{X: slotX, Y: slotY}
	return b.GetBase().SteerToward(target, 0.4)
}
