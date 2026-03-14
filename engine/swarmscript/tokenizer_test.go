package swarmscript

import (
	"testing"
)

// --- TokenizeLine tests ---

func TestTokenizeComment(t *testing.T) {
	tokens := TokenizeLine("# This is a comment")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token for comment, got %d", len(tokens))
	}
	if tokens[0].Type != TokComment {
		t.Errorf("expected TokComment, got %d", tokens[0].Type)
	}
}

func TestTokenizeKeywords(t *testing.T) {
	tokens := TokenizeLine("IF true THEN STOP")
	// Expect: IF(keyword) true(condition) THEN(keyword) STOP(action)
	keywordCount := 0
	for _, tok := range tokens {
		if tok.Type == TokKeyword {
			keywordCount++
		}
	}
	if keywordCount != 2 {
		t.Errorf("expected 2 keywords (IF, THEN), got %d", keywordCount)
	}
}

func TestTokenizeConditionAndAction(t *testing.T) {
	tokens := TokenizeLine("IF neighbors_count > 5 THEN MOVE_FORWARD")
	foundCond, foundAct, foundOp, foundNum := false, false, false, false
	for _, tok := range tokens {
		switch tok.Type {
		case TokCondition:
			foundCond = true
		case TokAction:
			foundAct = true
		case TokOperator:
			foundOp = true
		case TokNumber:
			foundNum = true
		}
	}
	if !foundCond {
		t.Error("expected a condition token")
	}
	if !foundAct {
		t.Error("expected an action token")
	}
	if !foundOp {
		t.Error("expected an operator token")
	}
	if !foundNum {
		t.Error("expected a number token")
	}
}

func TestTokenizeColumnPositions(t *testing.T) {
	tokens := TokenizeLine("IF true THEN STOP")
	// IF at col 0, true at col 3, THEN at col 8, STOP at col 13
	if tokens[0].Col != 0 {
		t.Errorf("IF should be at col 0, got %d", tokens[0].Col)
	}
	if tokens[1].Col != 3 {
		t.Errorf("true should be at col 3, got %d", tokens[1].Col)
	}
	if tokens[2].Col != 8 {
		t.Errorf("THEN should be at col 8, got %d", tokens[2].Col)
	}
	if tokens[3].Col != 13 {
		t.Errorf("STOP should be at col 13, got %d", tokens[3].Col)
	}
}

func TestTokenizeAliases(t *testing.T) {
	tokens := TokenizeLine("IF nbrs > 0 THEN FWD")
	foundCond, foundAct := false, false
	for _, tok := range tokens {
		if tok.Type == TokCondition && tok.Text == "nbrs" {
			foundCond = true
		}
		if tok.Type == TokAction && tok.Text == "FWD" {
			foundAct = true
		}
	}
	if !foundCond {
		t.Error("'nbrs' should be recognized as a condition token")
	}
	if !foundAct {
		t.Error("'FWD' should be recognized as an action token")
	}
}

func TestTokenizeEmptyLine(t *testing.T) {
	tokens := TokenizeLine("")
	if len(tokens) != 0 {
		t.Errorf("empty line should produce 0 tokens, got %d", len(tokens))
	}
}

func TestTokenizeOperators(t *testing.T) {
	tests := []string{">", "<", "==", "="}
	for _, op := range tests {
		tokens := TokenizeLine("IF state " + op + " 1 THEN STOP")
		foundOp := false
		for _, tok := range tokens {
			if tok.Type == TokOperator && tok.Text == op {
				foundOp = true
			}
		}
		if !foundOp {
			t.Errorf("operator '%s' should be recognized as TokOperator", op)
		}
	}
}

// --- isNumeric helper tests ---

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"123", true},
		{"-42", true},
		{"+7", true},
		{"0", true},
		{"abc", false},
		{"", false},
		{"-", false},
		{"+", false},
		{"12.5", false}, // no decimal support
	}
	for _, tc := range tests {
		got := isNumeric(tc.input)
		if got != tc.want {
			t.Errorf("isNumeric(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// --- splitKeepingPositions helper tests ---

func TestSplitKeepingPositions(t *testing.T) {
	result := splitKeepingPositions("IF  true   THEN")
	if len(result) != 3 {
		t.Fatalf("expected 3 words, got %d", len(result))
	}
	expected := []wordPos{
		{text: "IF", col: 0},
		{text: "true", col: 4},
		{text: "THEN", col: 11},
	}
	for i, wp := range result {
		if wp.text != expected[i].text || wp.col != expected[i].col {
			t.Errorf("word %d: got {%q, %d}, want {%q, %d}",
				i, wp.text, wp.col, expected[i].text, expected[i].col)
		}
	}
}
