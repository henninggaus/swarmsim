package swarm

import (
	"math/rand"
	"testing"
)

func TestDefaultCurriculumStages(t *testing.T) {
	stages := DefaultCurriculumStages()
	if len(stages) != 6 {
		t.Fatalf("expected 6 stages, got %d", len(stages))
	}
	for i, s := range stages {
		if s.Level != i+1 {
			t.Fatalf("stage %d: expected level %d, got %d", i, i+1, s.Level)
		}
	}
}

func TestInitCurriculum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitCurriculum(ss)
	if ss.Curriculum == nil {
		t.Fatal("curriculum should be initialized")
	}
	if ss.Curriculum.CurrentStage != 0 {
		t.Fatal("should start at stage 0")
	}
}

func TestAdvanceCurriculum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitCurriculum(ss)

	AdvanceCurriculum(ss)
	if ss.Curriculum.CurrentStage != 1 {
		t.Fatalf("expected stage 1, got %d", ss.Curriculum.CurrentStage)
	}
	if ss.Curriculum.TotalAdvances != 1 {
		t.Fatal("advances should be counted")
	}
}

func TestRetreatCurriculum(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitCurriculum(ss)

	AdvanceCurriculum(ss)
	RetreatCurriculum(ss)
	if ss.Curriculum.CurrentStage != 0 {
		t.Fatalf("expected stage 0 after retreat, got %d", ss.Curriculum.CurrentStage)
	}
}

func TestCurriculumProgress(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitCurriculum(ss)

	if CurriculumProgress(ss.Curriculum) != 0 {
		t.Fatal("progress should be 0 at start")
	}
	// Advance to last stage
	for i := 0; i < 5; i++ {
		AdvanceCurriculum(ss)
	}
	if CurriculumProgress(ss.Curriculum) != 1.0 {
		t.Fatal("progress should be 1.0 at last stage")
	}
}

func TestCurriculumNoAdvancePastMax(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitCurriculum(ss)

	for i := 0; i < 10; i++ {
		AdvanceCurriculum(ss)
	}
	if ss.Curriculum.CurrentStage != 5 { // 0-indexed, 6 stages
		t.Fatalf("should cap at last stage, got %d", ss.Curriculum.CurrentStage)
	}
}

func TestCheckImprovement(t *testing.T) {
	cs := &CurriculumState{
		WindowSize:       4,
		PlateauThreshold: 0.05,
	}

	// Stagnant: all same
	cs.PerformanceWindow = []float64{100, 100, 100, 100}
	if checkImprovement(cs) {
		t.Fatal("should detect stagnation")
	}

	// Improving
	cs.PerformanceWindow = []float64{100, 100, 200, 200}
	if !checkImprovement(cs) {
		t.Fatal("should detect improvement")
	}
}

func TestTickCurriculumAdvances(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitCurriculum(ss)
	ss.Curriculum.PlateauLimit = 2
	ss.Curriculum.WindowSize = 2

	// Feed stagnant fitness to trigger advancement
	for i := 0; i < 10; i++ {
		TickCurriculum(ss, 100.0)
	}
	if ss.Curriculum.CurrentStage == 0 {
		t.Fatal("should have advanced after plateau")
	}
}
