package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func makeChainSS() *SwarmState {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	return ss
}

func TestScenarioChainStopNil(t *testing.T) {
	ss := &SwarmState{}
	ScenarioChainStop(ss) // should not panic
}

func TestScenarioChainStopActive(t *testing.T) {
	ss := &SwarmState{}
	ss.ScenarioChain = &ScenarioChainState{Active: true}
	ScenarioChainStop(ss)
	if ss.ScenarioChain.Active {
		t.Error("chain should be stopped")
	}
}

func TestBuildChainFromTemplateNil(t *testing.T) {
	steps := BuildChainFromTemplate(nil)
	if steps != nil {
		t.Error("nil template should return nil")
	}
}

func TestBuildChainFromTemplate(t *testing.T) {
	tmpl := &ScenarioTemplate{
		Name: "test",
		Steps: []ScenarioStepDef{
			{Name: "step1", TickLimit: 1000, Delivery: true},
			{Name: "step2", TickLimit: 2000, Obstacles: true, Delivery: true},
		},
	}
	steps := BuildChainFromTemplate(tmpl)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Name != "step1" {
		t.Error("wrong step name")
	}
	if steps[1].TickLimit != 2000 {
		t.Error("wrong tick limit")
	}
}

func TestGetDefaultTemplate(t *testing.T) {
	tmpl := GetDefaultTemplate()
	if tmpl == nil {
		t.Fatal("should not be nil")
	}
	if len(tmpl.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(tmpl.Steps))
	}
}

func TestChainProgressEmpty(t *testing.T) {
	if ChainProgress(nil) != 0 {
		t.Error("nil chain should return 0")
	}
	if ChainProgress(&ScenarioChainState{}) != 0 {
		t.Error("empty steps should return 0")
	}
}

func TestChainProgress(t *testing.T) {
	chain := &ScenarioChainState{
		StepIdx: 1,
		Steps:   make([]ScenarioChainStep, 4),
	}
	p := ChainProgress(chain)
	if math.Abs(p-0.25) > 0.01 {
		t.Errorf("expected 0.25, got %f", p)
	}
}

func TestStepPathTracking(t *testing.T) {
	chain := &ScenarioChainState{
		StepPath: []int{0},
	}
	chain.StepPath = append(chain.StepPath, 1)
	chain.StepPath = append(chain.StepPath, 2)
	if len(chain.StepPath) != 3 {
		t.Errorf("expected 3 path entries, got %d", len(chain.StepPath))
	}
	if chain.StepPath[2] != 2 {
		t.Error("last path entry should be 2")
	}
}

func TestBranchFuncField(t *testing.T) {
	step := ScenarioChainStep{
		Name:      "test",
		TickLimit: 1000,
		BranchFunc: func(ss *SwarmState, score int) int {
			if score > 50 {
				return 2 // skip to step 2
			}
			return -1 // sequential
		},
	}
	// Score > 50 should branch
	result := step.BranchFunc(nil, 60)
	if result != 2 {
		t.Errorf("expected branch to 2, got %d", result)
	}
	// Score <= 50 should return -1 (sequential)
	result = step.BranchFunc(nil, 30)
	if result != -1 {
		t.Errorf("expected -1 (sequential), got %d", result)
	}
}

func TestMinScoreFailStep(t *testing.T) {
	step := ScenarioChainStep{
		Name:     "with min",
		MinScore: 100,
		FailStep: 0, // retry from step 0
	}
	if step.MinScore != 100 {
		t.Error("MinScore should be 100")
	}
	if step.FailStep != 0 {
		t.Error("FailStep should be 0")
	}
}

func TestScenarioStepDefFields(t *testing.T) {
	def := ScenarioStepDef{
		Name:      "maze challenge",
		TickLimit: 5000,
		Obstacles: false,
		Maze:      true,
		Delivery:  true,
		MinScore:  50,
		FailStep:  -1,
	}
	if def.Name != "maze challenge" {
		t.Error("wrong name")
	}
	if !def.Maze {
		t.Error("should have maze")
	}
	if def.Obstacles {
		t.Error("should not have obstacles")
	}
}
