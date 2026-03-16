package swarm

import (
	"swarmsim/logger"
)

const (
	TournamentRoundTicks = 3000 // each program runs for 3000 ticks
)

// TournamentAddEntry adds the current program to the tournament roster.
func TournamentAddEntry(ss *SwarmState, name, source string) {
	// Check for duplicate name — replace if exists
	for i := range ss.TournamentEntries {
		if ss.TournamentEntries[i].Name == name {
			ss.TournamentEntries[i].Source = source
			ss.TournamentEntries[i].Program = ss.Program
			logger.Info("TOURNAMENT", "Updated entry: %s", name)
			return
		}
	}
	entry := TournamentEntry{
		Name:    name,
		Source:  source,
		Program: ss.Program,
	}
	ss.TournamentEntries = append(ss.TournamentEntries, entry)
	logger.Info("TOURNAMENT", "Added entry: %s (total: %d)", name, len(ss.TournamentEntries))
}

// TournamentStart begins the tournament by running the first program.
func TournamentStart(ss *SwarmState) {
	if len(ss.TournamentEntries) < 2 {
		logger.Warn("TOURNAMENT", "Need at least 2 entries to start tournament")
		return
	}
	ss.TournamentOn = true
	ss.TournamentPhase = 1 // running
	ss.TournamentRound = 0
	ss.TournamentResults = make([]TournamentResult, len(ss.TournamentEntries))
	for i := range ss.TournamentResults {
		ss.TournamentResults[i].Name = ss.TournamentEntries[i].Name
	}
	TournamentLoadRound(ss)
	logger.Info("TOURNAMENT", "Tournament started with %d entries", len(ss.TournamentEntries))
}

// TournamentLoadRound loads the current round's program and resets bots.
func TournamentLoadRound(ss *SwarmState) {
	entry := &ss.TournamentEntries[ss.TournamentRound]
	ss.Program = entry.Program
	ss.ProgramText = entry.Source
	ss.ProgramName = entry.Name
	ss.TournamentTimer = TournamentRoundTicks

	// Reset bots and delivery state
	ss.ResetBots()
	ss.DeliveryStats = DeliveryStats{}
	if ss.DeliveryOn {
		ss.ResetDeliveryState()
		GenerateDeliveryStations(ss)
	}

	logger.Info("TOURNAMENT", "Round %d/%d: %s (%d ticks)",
		ss.TournamentRound+1, len(ss.TournamentEntries), entry.Name, TournamentRoundTicks)
}

// TournamentTick decrements timer and advances rounds.
func TournamentTick(ss *SwarmState) {
	if ss.TournamentPhase != 1 {
		return
	}
	ss.TournamentTimer--
	if ss.TournamentTimer > 0 {
		return
	}

	// Round complete — record results
	r := &ss.TournamentResults[ss.TournamentRound]
	r.Deliveries = ss.DeliveryStats.TotalDelivered
	r.Correct = ss.DeliveryStats.CorrectDelivered
	r.Wrong = ss.DeliveryStats.WrongDelivered
	r.Score = r.Correct*30 - r.Wrong*10

	logger.Info("TOURNAMENT", "Round %d complete: %s — Score:%d (Correct:%d Wrong:%d)",
		ss.TournamentRound+1, r.Name, r.Score, r.Correct, r.Wrong)

	// Next round or finish
	ss.TournamentRound++
	if ss.TournamentRound >= len(ss.TournamentEntries) {
		ss.TournamentPhase = 2 // results
		logger.Info("TOURNAMENT", "Tournament complete!")
	} else {
		TournamentLoadRound(ss)
	}
}

// TournamentStop ends the tournament.
func TournamentStop(ss *SwarmState) {
	ss.TournamentOn = false
	ss.TournamentPhase = 0
	logger.Info("TOURNAMENT", "Tournament stopped")
}
