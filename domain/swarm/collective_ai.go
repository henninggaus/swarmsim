package swarm

import (
	"fmt"
	"swarmsim/engine/swarmscript"
)

// IssueStatus tracks the lifecycle of a swarm issue.
type IssueStatus int

const (
	IssueOpen    IssueStatus = iota // problem detected, waiting for AI
	IssueCodeGen                    // AI generated code
	IssueTesting                    // bot is testing the new code
	IssueResolved                   // code works -- adopted
	IssueFailed                     // code didn't help -- reverted
)

// SwarmIssue represents a problem detected by a bot.
type SwarmIssue struct {
	ID              int
	BotIdx          int
	Tick            int
	Problem         string // "stuck", "no_package", "slow_delivery", "obstacle", "isolated", "energy_crisis"
	SensorSnap      string // snapshot of key sensors
	Status          IssueStatus
	GeneratedCode   string // AI-generated SwarmScript rules
	TestStartTick   int    // when testing began
	TestDuration    int    // how long to test (500 ticks)
	PreTestFitness  float64
	PostTestFitness float64
	Resolved        bool
	// Backup of bot's original OwnProgram so we can revert on failure.
	backupProgram *swarmscript.SwarmProgram
}

// IssueBoardState manages all issues in the simulation.
type IssueBoardState struct {
	Issues    []SwarmIssue
	NextID    int
	MaxIssues int // cap at 50 to prevent flooding

	// Per-bot tracking for problem detection
	StuckTicks    []int // how many ticks each bot has been stuck (speed < 0.1)
	NoCarryTicks  []int // how many ticks carrying==-1
	LastDelivery  []int // tick of last delivery per bot
	ObstacleTicks []int // ticks with obstacle_ahead
	IsolatedTicks []int // ticks with neighbors==0

	// Swarm learning: proven solutions that can be shared
	ProvenSolutions map[string]string // problem type -> SwarmScript code that worked

	// Claude API backend (optional, for real LLM code generation)
	UseClaudeAPI   bool             // true = send issues to Claude API
	ClaudeBackend  *ClaudeAIBackend // nil when API is disabled
	ClaudeLastErr  string           // last API error message (for UI display)

	// Internal detection counter (only run every 200 ticks)
	detectCounter int
}

// BotChatEntry is one entry in a bot's AI conversation history.
type BotChatEntry struct {
	Tick    int
	Problem string
	Sensors string
	Code    string
	Result  string // "RESOLVED", "FAILED", "TESTING", "ADOPTED (from Bot#N)"
}

// hasActiveIssue returns true if the given bot already has an open or testing issue.
func hasActiveIssue(ib *IssueBoardState, botIdx int) bool {
	for i := range ib.Issues {
		if ib.Issues[i].BotIdx == botIdx &&
			(ib.Issues[i].Status == IssueOpen || ib.Issues[i].Status == IssueTesting) {
			return true
		}
	}
	return false
}

// createIssue files a new issue on the board.
func createIssue(ss *SwarmState, botIdx int, problem, sensorSnap string) {
	ib := ss.IssueBoard
	issue := SwarmIssue{
		ID:           ib.NextID,
		BotIdx:       botIdx,
		Tick:         ss.Tick,
		Problem:      problem,
		SensorSnap:   sensorSnap,
		Status:       IssueOpen,
		TestDuration: 500,
	}
	ib.NextID++
	ib.Issues = append(ib.Issues, issue)
}

// addChatEntry appends a chat entry for a bot.
func addChatEntry(ss *SwarmState, botIdx, tick int, problem, sensors, code, result string) {
	if botIdx < 0 || botIdx >= len(ss.BotChatLog) {
		return
	}
	if ss.BotChatLog[botIdx] == nil {
		ss.BotChatLog[botIdx] = make([]BotChatEntry, 0, 8)
	}
	ss.BotChatLog[botIdx] = append(ss.BotChatLog[botIdx], BotChatEntry{
		Tick:    tick,
		Problem: problem,
		Sensors: sensors,
		Code:    code,
		Result:  result,
	})
	// Cap at 20 entries per bot
	if len(ss.BotChatLog[botIdx]) > 20 {
		ss.BotChatLog[botIdx] = ss.BotChatLog[botIdx][len(ss.BotChatLog[botIdx])-20:]
	}
}

// updateChatEntry updates the last chat entry result for a bot.
func updateChatEntry(ss *SwarmState, botIdx int, result string) {
	if botIdx < 0 || botIdx >= len(ss.BotChatLog) {
		return
	}
	log := ss.BotChatLog[botIdx]
	if len(log) == 0 {
		return
	}
	log[len(log)-1].Result = result
}

// growTrackingArrays ensures per-bot tracking arrays match bot count.
func growTrackingArrays(ib *IssueBoardState, tick, botCount int) {
	for len(ib.StuckTicks) < botCount {
		ib.StuckTicks = append(ib.StuckTicks, 0)
		ib.NoCarryTicks = append(ib.NoCarryTicks, 0)
		ib.LastDelivery = append(ib.LastDelivery, tick)
		ib.ObstacleTicks = append(ib.ObstacleTicks, 0)
		ib.IsolatedTicks = append(ib.IsolatedTicks, 0)
	}
}

// DetectBotProblems checks all bots for problems. Called every tick but only
// runs detection every 200 ticks internally.
func DetectBotProblems(ss *SwarmState) {
	if ss.IssueBoard == nil || !ss.CollectiveAIOn {
		return
	}
	ib := ss.IssueBoard

	ib.detectCounter++
	if ib.detectCounter < 20 {
		return // only run every 20 calls (called every 10 ticks = every 200 ticks)
	}
	ib.detectCounter = 0

	growTrackingArrays(ib, ss.Tick, len(ss.Bots))

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Update counters
		if bot.Speed < 0.1 {
			ib.StuckTicks[i]++
		} else {
			ib.StuckTicks[i] = 0
		}
		if bot.CarryingPkg < 0 {
			ib.NoCarryTicks[i]++
		} else {
			ib.NoCarryTicks[i] = 0
		}
		if bot.ObstacleAhead {
			ib.ObstacleTicks[i]++
		} else {
			ib.ObstacleTicks[i] = 0
		}
		if bot.NeighborCount == 0 {
			ib.IsolatedTicks[i]++
		} else {
			ib.IsolatedTicks[i] = 0
		}

		// Check for problems (only if bot doesn't already have an active issue)
		if hasActiveIssue(ib, i) {
			continue
		}
		if len(ib.Issues) >= ib.MaxIssues {
			continue
		}

		if ib.StuckTicks[i] > 100 {
			createIssue(ss, i, "stuck", fmt.Sprintf("speed=%.1f obs=%v neighbors=%d", bot.Speed, bot.ObstacleAhead, bot.NeighborCount))
		} else if ss.DeliveryOn && ib.NoCarryTicks[i] > 500 {
			createIssue(ss, i, "no_package", fmt.Sprintf("carry=%d p_dist=%.0f d_dist=%.0f", bot.CarryingPkg, bot.NearestPickupDist, bot.NearestDropoffDist))
		} else if ib.ObstacleTicks[i] > 50 {
			createIssue(ss, i, "obstacle", fmt.Sprintf("obs=%v wall_r=%v wall_l=%v", bot.ObstacleAhead, bot.WallRight, bot.WallLeft))
		} else if ib.IsolatedTicks[i] > 300 {
			createIssue(ss, i, "isolated", fmt.Sprintf("neighbors=%d nearest=%.0f", bot.NeighborCount, bot.NearestDist))
		} else if ss.DeliveryOn && ss.Tick-ib.LastDelivery[i] > 1000 {
			createIssue(ss, i, "slow_delivery", fmt.Sprintf("carry=%d ticks_since=%d", bot.CarryingPkg, ss.Tick-ib.LastDelivery[i]))
		} else if ss.EnergyEnabled && bot.Energy < 10 {
			createIssue(ss, i, "energy_crisis", fmt.Sprintf("energy=%.1f", bot.Energy))
		}
	}

	// Prune resolved/failed issues older than 2000 ticks
	pruneOldIssues(ib, ss.Tick)
}

// pruneOldIssues removes resolved/failed issues older than 2000 ticks.
func pruneOldIssues(ib *IssueBoardState, tick int) {
	alive := 0
	for i := range ib.Issues {
		issue := &ib.Issues[i]
		if (issue.Status == IssueResolved || issue.Status == IssueFailed) && tick-issue.Tick > 2000 {
			continue
		}
		ib.Issues[alive] = ib.Issues[i]
		alive++
	}
	ib.Issues = ib.Issues[:alive]
}

// computeBotFitness computes a simple fitness score for a bot.
// Uses delivery stats if delivery is on, otherwise uses distance traveled.
func computeBotFitness(ss *SwarmState, botIdx int) float64 {
	bot := &ss.Bots[botIdx]
	if ss.DeliveryOn {
		// Delivery fitness: correct deliveries * 100 + total deliveries * 10 + distance / 100
		return float64(bot.Stats.CorrectDeliveries)*100.0 +
			float64(bot.Stats.TotalDeliveries)*10.0 +
			bot.Stats.TotalDistance/100.0
	}
	// General fitness: distance + neighbors (exploration + sociality)
	return bot.Stats.TotalDistance/10.0 + bot.Stats.SumNeighborCount/100.0
}

// ProcessOpenIssues takes open issues, generates code, parses it, applies to bot.
// When Claude API is enabled, open issues are sent asynchronously; results are
// collected each tick and applied once they arrive.
func ProcessOpenIssues(ss *SwarmState) {
	if ss.IssueBoard == nil {
		return
	}
	ib := ss.IssueBoard

	// --- Claude API path: collect async results and apply them ---
	if ib.UseClaudeAPI && ib.ClaudeBackend != nil {
		// Collect completed API results
		results := ib.ClaudeBackend.CollectResults()
		for id, code := range results {
			for i := range ib.Issues {
				issue := &ib.Issues[i]
				if issue.ID != id || issue.Status != IssueCodeGen {
					continue
				}
				applyGeneratedCode(ss, issue, code)
			}
		}

		// Collect API errors -> fall back to template engine
		errs := ib.ClaudeBackend.CollectErrors()
		for id, errMsg := range errs {
			for i := range ib.Issues {
				issue := &ib.Issues[i]
				if issue.ID != id || issue.Status != IssueCodeGen {
					continue
				}
				ib.ClaudeLastErr = errMsg
				// Fallback: use template engine instead
				code := generateCodeForIssue(ss, issue)
				applyGeneratedCode(ss, issue, code)
			}
		}

		// Send new open issues to the API
		for i := range ib.Issues {
			issue := &ib.Issues[i]
			if issue.Status != IssueOpen {
				continue
			}
			if issue.BotIdx < 0 || issue.BotIdx >= len(ss.Bots) {
				issue.Status = IssueFailed
				continue
			}
			issue.Status = IssueCodeGen // mark as waiting for API
			ib.ClaudeBackend.RequestCodeGeneration(issue)
		}
		return
	}

	// --- Template engine path (default) ---
	for i := range ib.Issues {
		issue := &ib.Issues[i]
		if issue.Status != IssueOpen {
			continue
		}
		if issue.BotIdx < 0 || issue.BotIdx >= len(ss.Bots) {
			issue.Status = IssueFailed
			continue
		}

		// Generate code via template engine
		code := generateCodeForIssue(ss, issue)
		applyGeneratedCode(ss, issue, code)
	}
}

// applyGeneratedCode parses SwarmScript code, applies it to the bot, and starts testing.
func applyGeneratedCode(ss *SwarmState, issue *SwarmIssue, code string) {
	issue.GeneratedCode = code
	issue.Status = IssueTesting
	issue.TestStartTick = ss.Tick
	issue.TestDuration = 500

	// Parse and prepend to bot's program
	newRules, err := swarmscript.ParseSwarmScript(code)
	if err != nil {
		issue.Status = IssueFailed
		addChatEntry(ss, issue.BotIdx, issue.Tick, issue.Problem, issue.SensorSnap, code, "FAILED (parse error)")
		return
	}

	// Record pre-test fitness
	issue.PreTestFitness = computeBotFitness(ss, issue.BotIdx)

	// Apply: backup current OwnProgram, then prepend new rules
	bot := &ss.Bots[issue.BotIdx]
	issue.backupProgram = bot.OwnProgram

	if bot.OwnProgram != nil {
		// Merge: prepend new rules to existing OwnProgram
		merged := &swarmscript.SwarmProgram{
			Rules: append(newRules.Rules, bot.OwnProgram.Rules...),
		}
		bot.OwnProgram = merged
	} else if ss.Program != nil {
		// Create OwnProgram from global program + new rules
		merged := &swarmscript.SwarmProgram{
			Rules: append(newRules.Rules, ss.Program.Rules...),
		}
		bot.OwnProgram = merged
	} else {
		bot.OwnProgram = newRules
	}

	addChatEntry(ss, issue.BotIdx, issue.Tick, issue.Problem, issue.SensorSnap, code, "TESTING")
}

// EvaluateTestingIssues checks testing issues and resolves or fails them.
func EvaluateTestingIssues(ss *SwarmState) {
	if ss.IssueBoard == nil {
		return
	}
	for i := range ss.IssueBoard.Issues {
		issue := &ss.IssueBoard.Issues[i]
		if issue.Status != IssueTesting {
			continue
		}
		if ss.Tick-issue.TestStartTick < issue.TestDuration {
			continue
		}
		if issue.BotIdx < 0 || issue.BotIdx >= len(ss.Bots) {
			issue.Status = IssueFailed
			continue
		}

		// Compare fitness
		issue.PostTestFitness = computeBotFitness(ss, issue.BotIdx)
		improved := issue.PostTestFitness > issue.PreTestFitness*1.1 // 10% improvement threshold

		if improved {
			issue.Status = IssueResolved
			issue.Resolved = true
			// Store as proven solution for swarm learning
			ss.IssueBoard.ProvenSolutions[issue.Problem] = issue.GeneratedCode
			updateChatEntry(ss, issue.BotIdx, "RESOLVED")
			// Phase 5: Spread proven solution to nearby bots with same problem
			spreadProvenSolution(ss, issue)
		} else {
			issue.Status = IssueFailed
			// Revert test rules from bot
			ss.Bots[issue.BotIdx].OwnProgram = issue.backupProgram
			updateChatEntry(ss, issue.BotIdx, "FAILED")
		}
	}
}

// spreadProvenSolution applies a resolved solution to other bots with the same problem.
func spreadProvenSolution(ss *SwarmState, issue *SwarmIssue) {
	ib := ss.IssueBoard
	growTrackingArrays(ib, ss.Tick, len(ss.Bots))

	for i := range ss.Bots {
		if i == issue.BotIdx {
			continue
		}
		// Check if this bot has the same problem type (via tracking counters)
		hasSameProblem := false
		switch issue.Problem {
		case "stuck":
			hasSameProblem = i < len(ib.StuckTicks) && ib.StuckTicks[i] > 50
		case "obstacle":
			hasSameProblem = i < len(ib.ObstacleTicks) && ib.ObstacleTicks[i] > 30
		case "isolated":
			hasSameProblem = i < len(ib.IsolatedTicks) && ib.IsolatedTicks[i] > 150
		case "no_package":
			hasSameProblem = i < len(ib.NoCarryTicks) && ib.NoCarryTicks[i] > 300
		case "slow_delivery":
			hasSameProblem = i < len(ib.LastDelivery) && ss.Tick-ib.LastDelivery[i] > 800
		case "energy_crisis":
			hasSameProblem = ss.EnergyEnabled && ss.Bots[i].Energy < 15
		}
		if !hasSameProblem {
			continue
		}

		// Apply proven solution to this bot too
		newRules, err := swarmscript.ParseSwarmScript(issue.GeneratedCode)
		if err != nil {
			continue
		}
		bot := &ss.Bots[i]
		if bot.OwnProgram != nil {
			merged := &swarmscript.SwarmProgram{
				Rules: append(newRules.Rules, bot.OwnProgram.Rules...),
			}
			bot.OwnProgram = merged
		} else if ss.Program != nil {
			merged := &swarmscript.SwarmProgram{
				Rules: append(newRules.Rules, ss.Program.Rules...),
			}
			bot.OwnProgram = merged
		} else {
			bot.OwnProgram = newRules
		}
		addChatEntry(ss, i, ss.Tick, issue.Problem, "", issue.GeneratedCode,
			fmt.Sprintf("ADOPTED (from Bot#%d)", issue.BotIdx))
	}
}

// RecordDelivery should be called when a bot delivers a package to update tracking.
func RecordDelivery(ss *SwarmState, botIdx int) {
	if ss.IssueBoard == nil {
		return
	}
	growTrackingArrays(ss.IssueBoard, ss.Tick, len(ss.Bots))
	if botIdx >= 0 && botIdx < len(ss.IssueBoard.LastDelivery) {
		ss.IssueBoard.LastDelivery[botIdx] = ss.Tick
	}
}
