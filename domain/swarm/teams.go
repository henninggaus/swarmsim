package swarm

// InitTeams splits bots into two teams and positions them on opposite sides.
func InitTeams(ss *SwarmState) {
	half := len(ss.Bots) / 2
	for i := range ss.Bots {
		if i < half {
			ss.Bots[i].Team = 1 // Team A (Blue) — left side
			ss.Bots[i].X = ss.Rng.Float64() * ss.ArenaW * 0.4
			ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
		} else {
			ss.Bots[i].Team = 2 // Team B (Red) — right side
			ss.Bots[i].X = ss.ArenaW*0.6 + ss.Rng.Float64()*ss.ArenaW*0.4
			ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
		}
		ss.Bots[i].Angle = ss.Rng.Float64() * 6.283
		ss.Bots[i].Speed = 0
	}
	ss.TeamAScore = 0
	ss.TeamBScore = 0
	ss.ChallengeActive = false
	ss.ChallengeTicks = 0
	ss.ChallengeResult = ""
}

// ResetTeamScores clears team scores and repositions bots.
func ResetTeamScores(ss *SwarmState) {
	ss.TeamAScore = 0
	ss.TeamBScore = 0
	ss.ChallengeActive = false
	ss.ChallengeTicks = 0
	ss.ChallengeResult = ""
	// Reposition bots to team sides
	half := len(ss.Bots) / 2
	for i := range ss.Bots {
		if i < half {
			ss.Bots[i].X = ss.Rng.Float64() * ss.ArenaW * 0.4
			ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
		} else {
			ss.Bots[i].X = ss.ArenaW*0.6 + ss.Rng.Float64()*ss.ArenaW*0.4
			ss.Bots[i].Y = ss.Rng.Float64() * ss.ArenaH
		}
		ss.Bots[i].Angle = ss.Rng.Float64() * 6.283
		ss.Bots[i].CarryingPkg = -1
	}
}

// ClearTeams removes team assignments.
func ClearTeams(ss *SwarmState) {
	for i := range ss.Bots {
		ss.Bots[i].Team = 0
	}
	ss.TeamsEnabled = false
	ss.TeamAScore = 0
	ss.TeamBScore = 0
	ss.ChallengeActive = false
	ss.ChallengeResult = ""
}

// UpdateChallenge ticks down the challenge timer and determines winner.
func UpdateChallenge(ss *SwarmState) {
	if !ss.ChallengeActive || ss.ChallengeResult != "" {
		return
	}
	ss.ChallengeTicks--
	if ss.ChallengeTicks <= 0 {
		if ss.TeamAScore > ss.TeamBScore {
			ss.ChallengeResult = "TEAM A WINS!"
		} else if ss.TeamBScore > ss.TeamAScore {
			ss.ChallengeResult = "TEAM B WINS!"
		} else {
			ss.ChallengeResult = "DRAW!"
		}
	}
}
