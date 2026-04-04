package swarm

import (
	"math/rand"
	"testing"
)

// newCAITestState creates a minimal SwarmState with IssueBoard initialized
// for collective AI tests.
func newCAITestState(botCount int) *SwarmState {
	ss := newTestSwarmState(botCount)
	ss.Rng = rand.New(rand.NewSource(42))
	ss.IssueBoard = &IssueBoardState{
		MaxIssues:       50,
		ProvenSolutions: make(map[string]string),
	}
	ss.BotChatLog = make([][]BotChatEntry, botCount)
	return ss
}

func TestIssueBoardInit(t *testing.T) {
	ss := newCAITestState(20)
	if ss.IssueBoard == nil {
		t.Fatal("issue board should be initialized")
	}
	if ss.IssueBoard.MaxIssues != 50 {
		t.Error("max issues should be 50")
	}
	if ss.IssueBoard.ProvenSolutions == nil {
		t.Error("proven solutions map should exist")
	}
}

func TestCreateIssue(t *testing.T) {
	ss := newCAITestState(20)
	ss.CollectiveAIOn = true
	// Manually create an issue
	createIssue(ss, 0, "stuck", "speed=0 obs=true")
	if len(ss.IssueBoard.Issues) != 1 {
		t.Fatal("should have 1 issue")
	}
	iss := ss.IssueBoard.Issues[0]
	if iss.BotIdx != 0 {
		t.Error("wrong bot")
	}
	if iss.Problem != "stuck" {
		t.Error("wrong problem")
	}
	if iss.Status != IssueOpen {
		t.Error("should be open")
	}
}

func TestHasActiveIssue(t *testing.T) {
	ss := newCAITestState(20)
	ss.CollectiveAIOn = true
	if hasActiveIssue(ss.IssueBoard, 0) {
		t.Error("should have no active issue")
	}
	createIssue(ss, 0, "stuck", "test")
	if !hasActiveIssue(ss.IssueBoard, 0) {
		t.Error("should have active issue")
	}
}

func TestGenerateCodeForIssue(t *testing.T) {
	ss := newCAITestState(10)
	issue := &SwarmIssue{Problem: "stuck", SensorSnap: "speed=0"}
	code := generateCodeForIssue(ss, issue)
	if code == "" {
		t.Error("should generate non-empty code")
	}
	// Code should be valid SwarmScript (contain IF...THEN)
	if len(code) < 10 {
		t.Errorf("code too short: %q", code)
	}
}

func TestProvenSolutions(t *testing.T) {
	ss := newCAITestState(10)
	ss.IssueBoard.ProvenSolutions["stuck"] = "IF obs_ahead == 1 THEN TURN_RANDOM"

	issue := &SwarmIssue{Problem: "stuck"}
	code := generateCodeForIssue(ss, issue)
	if code != "IF obs_ahead == 1 THEN TURN_RANDOM" {
		t.Error("should use proven solution")
	}
}

func TestRecordDelivery(t *testing.T) {
	ss := newCAITestState(10)
	ss.CollectiveAIOn = true
	RecordDelivery(ss, 0)
	if ss.IssueBoard.LastDelivery[0] != ss.Tick {
		t.Error("should record delivery tick")
	}
}
