package swarm

// MathStepKind categorizes a calculation step for color coding.
type MathStepKind int

const (
	MathInput        MathStepKind = iota // blue: input parameters
	MathIntermediate                     // yellow: intermediate values
	MathOutput                           // green: final outputs
	MathBranch                           // orange: branch/phase decision
)

// MathStep represents one line of a live calculation display.
type MathStep struct {
	Label       string
	Symbolic    string
	Substituted string
	Value       float64
	Kind        MathStepKind
}

// MathTrace captures the full calculation for one bot in one tick.
type MathTrace struct {
	AlgoName  string
	PhaseName string
	Steps     []MathStep
}

// AddStep appends a calculation step to the trace. Used by algorithm trace
// functions to record each variable in the computation.
func (mt *MathTrace) AddStep(label, symbolic, substituted string, value float64, kind MathStepKind) {
	mt.Steps = append(mt.Steps, MathStep{label, symbolic, substituted, value, kind})
}
