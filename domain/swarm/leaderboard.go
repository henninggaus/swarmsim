package swarm

import (
	"encoding/json"
	"os"
	"sort"
	"swarmsim/logger"
)

const leaderboardFile = "swarmsim_leaderboard.json"
const maxLeaderboardEntries = 20

// LeaderboardEntry represents one highscore record.
type LeaderboardEntry struct {
	Name       string  `json:"name"`
	Score      int     `json:"score"`       // total correct deliveries
	Deliveries int     `json:"deliveries"`  // total deliveries
	Correct    int     `json:"correct"`
	Wrong      int     `json:"wrong"`
	BotCount   int     `json:"bot_count"`
	Ticks      int     `json:"ticks"`       // how many ticks the program ran
	Generation int     `json:"generation"`  // evolution generation reached
	Efficiency float64 `json:"efficiency"`  // correct / total * 100
	Mode       string  `json:"mode"`        // "Script", "GP", "Neuro", "Evolution"
}

// LeaderboardState holds the runtime leaderboard.
type LeaderboardState struct {
	Entries []LeaderboardEntry `json:"entries"`
}

// LoadLeaderboard reads the leaderboard from disk.
func LoadLeaderboard() *LeaderboardState {
	lb := &LeaderboardState{}
	data, err := os.ReadFile(leaderboardFile)
	if err != nil {
		return lb
	}
	if err := json.Unmarshal(data, lb); err != nil {
		logger.Warn("LEADERBOARD", "Parse error: %v", err)
		return &LeaderboardState{}
	}
	return lb
}

// SaveLeaderboard writes the leaderboard to disk.
func SaveLeaderboard(lb *LeaderboardState) {
	data, err := json.MarshalIndent(lb, "", "  ")
	if err != nil {
		logger.Error("LEADERBOARD", "Marshal error: %v", err)
		return
	}
	if err := os.WriteFile(leaderboardFile, data, 0644); err != nil {
		logger.Error("LEADERBOARD", "Write error: %v", err)
	}
}

// SubmitScore adds a score if it qualifies for the leaderboard.
// Returns true if the score was added (new highscore).
func SubmitScore(lb *LeaderboardState, entry LeaderboardEntry) bool {
	// Calculate score: correct * 10 + wrong * 2
	entry.Score = entry.Correct*10 + entry.Wrong*2
	if entry.Deliveries > 0 {
		entry.Efficiency = float64(entry.Correct) / float64(entry.Deliveries) * 100
	}

	// Skip zero-score entries
	if entry.Score <= 0 {
		return false
	}

	// Check if it qualifies (find actual worst score defensively)
	if len(lb.Entries) >= maxLeaderboardEntries {
		worstScore := lb.Entries[0].Score
		for _, e := range lb.Entries {
			if e.Score < worstScore {
				worstScore = e.Score
			}
		}
		if entry.Score <= worstScore {
			return false
		}
	}

	lb.Entries = append(lb.Entries, entry)

	// Sort by score descending, then by ticks ascending (faster = better at same score)
	sort.SliceStable(lb.Entries, func(i, j int) bool {
		if lb.Entries[i].Score != lb.Entries[j].Score {
			return lb.Entries[i].Score > lb.Entries[j].Score
		}
		return lb.Entries[i].Ticks < lb.Entries[j].Ticks
	})

	// Trim to max entries
	if len(lb.Entries) > maxLeaderboardEntries {
		lb.Entries = lb.Entries[:maxLeaderboardEntries]
	}

	// Find actual rank of the newly inserted entry
	rank := len(lb.Entries)
	for i, e := range lb.Entries {
		if e.Name == entry.Name && e.Score == entry.Score && e.Ticks == entry.Ticks {
			rank = i + 1
			break
		}
	}

	SaveLeaderboard(lb)
	logger.Info("LEADERBOARD", "Neuer Eintrag: %s — Score: %d (Rang %d)",
		entry.Name, entry.Score, rank)
	return true
}

// LeaderboardTop returns up to n top entries.
func LeaderboardTop(lb *LeaderboardState, n int) []LeaderboardEntry {
	if n > len(lb.Entries) {
		n = len(lb.Entries)
	}
	return lb.Entries[:n]
}
