package swarm

import (
	"strings"
	"testing"
)

func TestSubmitScoreValid(t *testing.T) {
	lb := &LeaderboardState{}
	entry := LeaderboardEntry{
		Name:       "TestBot",
		Correct:    5,
		Wrong:      2,
		Deliveries: 7,
		BotCount:   50,
		Ticks:      1000,
	}
	ok := SubmitScore(lb, entry)
	if !ok {
		t.Fatal("expected score to be accepted")
	}
	if len(lb.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(lb.Entries))
	}
	// Score = Correct*10 + Wrong*2 = 54
	if lb.Entries[0].Score != 54 {
		t.Errorf("expected score 54, got %d", lb.Entries[0].Score)
	}
	// Efficiency = 5/7 * 100
	if lb.Entries[0].Efficiency < 71 || lb.Entries[0].Efficiency > 72 {
		t.Errorf("expected efficiency ~71.4, got %.1f", lb.Entries[0].Efficiency)
	}
}

func TestSubmitScoreRejectsEmptyName(t *testing.T) {
	lb := &LeaderboardState{}
	entry := LeaderboardEntry{
		Name:     "",
		Correct:  5,
		BotCount: 50,
		Ticks:    100,
	}
	if SubmitScore(lb, entry) {
		t.Error("should reject empty name")
	}
}

func TestSubmitScoreRejectsTooLongName(t *testing.T) {
	lb := &LeaderboardState{}
	entry := LeaderboardEntry{
		Name:     strings.Repeat("x", 200),
		Correct:  5,
		BotCount: 50,
		Ticks:    100,
	}
	if SubmitScore(lb, entry) {
		t.Error("should reject name > 100 chars")
	}
}

func TestSubmitScoreRejectsNegativeValues(t *testing.T) {
	lb := &LeaderboardState{}

	cases := []struct {
		name    string
		entry   LeaderboardEntry
	}{
		{"negative correct", LeaderboardEntry{Name: "a", Correct: -1, BotCount: 1, Ticks: 1}},
		{"negative wrong", LeaderboardEntry{Name: "a", Wrong: -1, Correct: 1, BotCount: 1, Ticks: 1}},
		{"zero bot count", LeaderboardEntry{Name: "a", Correct: 1, BotCount: 0, Ticks: 1}},
		{"negative ticks", LeaderboardEntry{Name: "a", Correct: 1, BotCount: 1, Ticks: -5}},
	}

	for _, tc := range cases {
		if SubmitScore(lb, tc.entry) {
			t.Errorf("should reject: %s", tc.name)
		}
	}
}

func TestSubmitScoreRejectsZeroScore(t *testing.T) {
	lb := &LeaderboardState{}
	entry := LeaderboardEntry{
		Name:     "ZeroBot",
		Correct:  0,
		Wrong:    0,
		BotCount: 10,
		Ticks:    100,
	}
	if SubmitScore(lb, entry) {
		t.Error("should reject zero score")
	}
}

func TestSubmitScoreSortsByScoreDescending(t *testing.T) {
	lb := &LeaderboardState{}

	entries := []LeaderboardEntry{
		{Name: "Low", Correct: 1, BotCount: 10, Ticks: 100},
		{Name: "High", Correct: 10, BotCount: 10, Ticks: 100},
		{Name: "Mid", Correct: 5, BotCount: 10, Ticks: 100},
	}

	for _, e := range entries {
		SubmitScore(lb, e)
	}

	if len(lb.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(lb.Entries))
	}
	if lb.Entries[0].Name != "High" {
		t.Errorf("first entry should be High, got %s", lb.Entries[0].Name)
	}
	if lb.Entries[2].Name != "Low" {
		t.Errorf("last entry should be Low, got %s", lb.Entries[2].Name)
	}
}

func TestSubmitScoreTrimsToMaxEntries(t *testing.T) {
	lb := &LeaderboardState{}

	// Fill with 20 entries
	for i := 1; i <= 20; i++ {
		SubmitScore(lb, LeaderboardEntry{
			Name: "Bot", Correct: i, BotCount: 10, Ticks: 100,
		})
	}

	if len(lb.Entries) != maxLeaderboardEntries {
		t.Fatalf("expected %d entries, got %d", maxLeaderboardEntries, len(lb.Entries))
	}

	// Adding a low score should fail
	ok := SubmitScore(lb, LeaderboardEntry{
		Name: "Weak", Correct: 0, Wrong: 1, BotCount: 10, Ticks: 100,
	})
	if ok {
		t.Error("low score should not make it into full leaderboard")
	}
}

func TestLeaderboardTopReturnsCorrectCount(t *testing.T) {
	lb := &LeaderboardState{}
	for i := 1; i <= 5; i++ {
		SubmitScore(lb, LeaderboardEntry{
			Name: "Bot", Correct: i, BotCount: 10, Ticks: 100,
		})
	}

	top3 := LeaderboardTop(lb, 3)
	if len(top3) != 3 {
		t.Fatalf("expected 3, got %d", len(top3))
	}

	// Requesting more than available
	top10 := LeaderboardTop(lb, 10)
	if len(top10) != 5 {
		t.Fatalf("expected 5 (all), got %d", len(top10))
	}
}
