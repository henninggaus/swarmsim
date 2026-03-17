package swarm

import (
	"math/rand"
	"testing"
)

func TestInitMultiSwarm(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 2)
	if ss.MultiSwarm == nil {
		t.Fatal("MultiSwarm should not be nil")
	}
	if len(ss.MultiSwarm.Swarms) != 2 {
		t.Errorf("expected 2 teams, got %d", len(ss.MultiSwarm.Swarms))
	}
	// Check all bots have a team
	for i, bot := range ss.Bots {
		if bot.Team < 0 || bot.Team > 1 {
			t.Errorf("bot %d has invalid team %d", i, bot.Team)
		}
	}
}

func TestInitMultiSwarmClamp(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 10) // should clamp to 4
	if len(ss.MultiSwarm.Swarms) != 4 {
		t.Errorf("expected 4 teams (clamped), got %d", len(ss.MultiSwarm.Swarms))
	}
}

func TestInitMultiSwarmMinTeams(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 1) // should clamp to 2
	if len(ss.MultiSwarm.Swarms) != 2 {
		t.Errorf("expected 2 teams (min), got %d", len(ss.MultiSwarm.Swarms))
	}
}

func TestClearMultiSwarm(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 2)
	ClearMultiSwarm(ss)
	if ss.MultiSwarm != nil {
		t.Error("MultiSwarm should be nil")
	}
}

func TestTickMultiSwarmNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	TickMultiSwarm(ss, rng) // should not panic
}

func TestTickMultiSwarm(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 2)
	for i := 0; i < 100; i++ {
		TickMultiSwarm(ss, rng)
	}
	if ss.MultiSwarm.RoundTicks != 100 {
		t.Errorf("expected 100 round ticks, got %d", ss.MultiSwarm.RoundTicks)
	}
}

func TestTickMultiSwarmRoundReset(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 2)
	ss.MultiSwarm.RoundLimit = 10
	for i := 0; i < 15; i++ {
		TickMultiSwarm(ss, rng)
	}
	if ss.MultiSwarm.Round != 1 {
		t.Errorf("expected round 1, got %d", ss.MultiSwarm.Round)
	}
}

func TestTerritory(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 2)
	// Put all bots of team 0 in top-left
	for i := ss.MultiSwarm.Swarms[0].BotStart; i < ss.MultiSwarm.Swarms[0].BotEnd; i++ {
		ss.Bots[i].X = 10
		ss.Bots[i].Y = 10
	}
	// Put all bots of team 1 in bottom-right
	for i := ss.MultiSwarm.Swarms[1].BotStart; i < ss.MultiSwarm.Swarms[1].BotEnd; i++ {
		ss.Bots[i].X = ss.ArenaW - 10
		ss.Bots[i].Y = ss.ArenaH - 10
	}
	computeTerritory(ss, ss.MultiSwarm)
	totalTerritory := 0.0
	for _, team := range ss.MultiSwarm.Swarms {
		totalTerritory += team.Territory
		if team.Territory < 0 || team.Territory > 1 {
			t.Errorf("team %d territory %.2f out of range", team.ID, team.Territory)
		}
	}
	// Both should have some territory
	if ss.MultiSwarm.Swarms[0].Territory == 0 {
		t.Error("team 0 should have some territory")
	}
	if ss.MultiSwarm.Swarms[1].Territory == 0 {
		t.Error("team 1 should have some territory")
	}
}

func TestMultiSwarmLeader(t *testing.T) {
	if MultiSwarmLeader(nil) != -1 {
		t.Error("nil should return -1")
	}
	ms := &MultiSwarmState{
		Swarms: []SwarmTeam{
			{ID: 0, Score: 10},
			{ID: 1, Score: 50},
			{ID: 2, Score: 30},
		},
	}
	if MultiSwarmLeader(ms) != 1 {
		t.Errorf("expected leader 1, got %d", MultiSwarmLeader(ms))
	}
}

func TestMultiSwarmTotalBots(t *testing.T) {
	if MultiSwarmTotalBots(nil) != 0 {
		t.Error("nil should return 0")
	}
	ms := &MultiSwarmState{
		Swarms: []SwarmTeam{
			{BotStart: 0, BotEnd: 10},
			{BotStart: 10, BotEnd: 20},
		},
	}
	if MultiSwarmTotalBots(ms) != 20 {
		t.Errorf("expected 20, got %d", MultiSwarmTotalBots(ms))
	}
}

func TestMultiSwarmTeamDistance(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 2)
	// Separate teams
	for i := ss.MultiSwarm.Swarms[0].BotStart; i < ss.MultiSwarm.Swarms[0].BotEnd; i++ {
		ss.Bots[i].X = 0
		ss.Bots[i].Y = 0
	}
	for i := ss.MultiSwarm.Swarms[1].BotStart; i < ss.MultiSwarm.Swarms[1].BotEnd; i++ {
		ss.Bots[i].X = 100
		ss.Bots[i].Y = 0
	}
	dist := MultiSwarmTeamDistance(ss, ss.MultiSwarm)
	if dist < 99 || dist > 101 {
		t.Errorf("expected distance ~100, got %f", dist)
	}
}

func TestMultiSwarmTeamDistanceNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	if MultiSwarmTeamDistance(ss, nil) != 0 {
		t.Error("nil should return 0")
	}
}

func TestMigration(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitMultiSwarm(ss, 2)
	ss.MultiSwarm.MigrationRate = 1.0 // always migrate
	initialTeams := make([]int, len(ss.Bots))
	for i := range ss.Bots {
		initialTeams[i] = ss.Bots[i].Team
	}
	TickMultiSwarm(ss, rng)
	changed := false
	for i := range ss.Bots {
		if ss.Bots[i].Team != initialTeams[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Error("with migration rate 1.0, at least one bot should migrate")
	}
	if ss.MultiSwarm.MigrationCount == 0 {
		t.Error("migration count should be > 0")
	}
}
