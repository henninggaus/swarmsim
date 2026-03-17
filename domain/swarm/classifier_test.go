package swarm

import (
	"math/rand"
	"testing"
)

func TestInitClassifier(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitClassifier(ss)

	cs := ss.Classifier
	if cs == nil {
		t.Fatal("classifier should be initialized")
	}
	if len(cs.RuleSets) != 15 {
		t.Fatalf("expected 15 rule sets, got %d", len(cs.RuleSets))
	}
	for i, rs := range cs.RuleSets {
		if len(rs.Rules) != 16 {
			t.Fatalf("bot %d: expected 16 rules, got %d", i, len(rs.Rules))
		}
	}
}

func TestClearClassifier(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.ClassifierOn = true
	InitClassifier(ss)
	ClearClassifier(ss)

	if ss.Classifier != nil {
		t.Fatal("should be nil")
	}
	if ss.ClassifierOn {
		t.Fatal("should be false")
	}
}

func TestTickClassifier(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitClassifier(ss)

	for i := 0; i < 10; i++ {
		ss.Bots[i].NearestPickupDist = 30
	}

	for tick := 0; tick < 100; tick++ {
		TickClassifier(ss)
	}

	// Some rules should have been activated
	cs := ss.Classifier
	totalMatches := 0
	for _, rs := range cs.RuleSets {
		for _, r := range rs.Rules {
			totalMatches += r.MatchCount
		}
	}
	if totalMatches == 0 {
		t.Fatal("some rules should have been activated")
	}
}

func TestTickClassifierNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickClassifier(ss) // should not panic
}

func TestEvolveClassifier(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitClassifier(ss)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveClassifier(ss, sorted)
	if ss.Classifier.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.Classifier.Generation)
	}
}

func TestRuleMatches(t *testing.T) {
	rule := ClassifierRule{
		Conditions: [5][2]float64{
			{0.0, 0.5},
			{0.0, 1.0},
			{0.0, 1.0},
			{0.0, 1.0},
			{0.0, 1.0},
		},
	}

	// Should match
	inputs := [5]float64{0.3, 0.5, 0.5, 0.5, 0.5}
	if !ruleMatches(rule, inputs) {
		t.Fatal("should match")
	}

	// Should not match (dim 0 too high)
	inputs[0] = 0.8
	if ruleMatches(rule, inputs) {
		t.Fatal("should not match")
	}
}

func TestClassifierAvgStrength(t *testing.T) {
	if ClassifierAvgStrength(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestClassifierBestRule(t *testing.T) {
	if ClassifierBestRule(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBotRuleCount(t *testing.T) {
	if BotRuleCount(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitClassifier(ss)

	if BotRuleCount(ss.Classifier, 0) != 16 {
		t.Fatal("expected 16 rules")
	}
	if BotRuleCount(ss.Classifier, 10) != 0 {
		t.Fatal("out of bounds should return 0")
	}
}

func TestCloneRuleSet(t *testing.T) {
	src := BotRuleSet{
		Rules: []ClassifierRule{
			{Action: 1, Strength: 0.8},
			{Action: 3, Strength: 0.5},
		},
		LastAction: 2,
	}

	dst := cloneRuleSet(src)
	dst.Rules[0].Strength = 0.1

	if src.Rules[0].Strength == 0.1 {
		t.Fatal("clone should be independent")
	}
}
