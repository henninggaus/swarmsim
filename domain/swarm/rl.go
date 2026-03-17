package swarm

import (
	"math"
	"math/rand"
)

// RLState holds the Q-learning state for reinforcement learning bots.
type RLState struct {
	QTable       [][]float64 // [state][action] → Q-value
	NumStates    int
	NumActions   int
	Alpha        float64 // learning rate (default 0.1)
	Gamma        float64 // discount factor (default 0.95)
	Epsilon      float64 // exploration rate (default 0.2)
	EpsilonDecay float64 // decay per episode (default 0.995)
	EpsilonMin   float64 // minimum exploration (default 0.01)
	Episode      int
	TotalReward  float64
	AvgReward    float64
	RewardHistory []float64
	MaxHistory   int
}

// RLBotState tracks per-bot RL data.
type RLBotState struct {
	PrevState  int
	PrevAction int
	Reward     float64
}

const (
	rlStateGridCols = 4  // discretize X into 4 bins
	rlStateGridRows = 4  // discretize Y into 4 bins
	rlCarryStates   = 2  // carrying or not
	rlProxStates    = 3  // near nothing, near pickup, near dropoff
	rlNumStates     = rlStateGridCols * rlStateGridRows * rlCarryStates * rlProxStates // 96
	rlNumActions    = 8  // same as NeuroOutputs
)

// InitRL initializes the Q-learning system.
func InitRL(ss *SwarmState) {
	rl := &RLState{
		NumStates:    rlNumStates,
		NumActions:   rlNumActions,
		Alpha:        0.1,
		Gamma:        0.95,
		Epsilon:      0.2,
		EpsilonDecay: 0.995,
		EpsilonMin:   0.01,
		MaxHistory:   100,
	}
	rl.QTable = make([][]float64, rl.NumStates)
	for i := range rl.QTable {
		rl.QTable[i] = make([]float64, rl.NumActions)
	}
	ss.RLState = rl
	ss.RLEnabled = true

	// Init per-bot RL states
	ss.RLBotStates = make([]RLBotState, len(ss.Bots))
}

// ClearRL disables RL.
func ClearRL(ss *SwarmState) {
	ss.RLState = nil
	ss.RLEnabled = false
	ss.RLBotStates = nil
}

// DiscretizeState maps continuous bot state to a discrete state index.
func DiscretizeState(bot *SwarmBot, ss *SwarmState) int {
	// Grid position
	col := int(bot.X / (ss.ArenaW / float64(rlStateGridCols)))
	if col >= rlStateGridCols {
		col = rlStateGridCols - 1
	}
	if col < 0 {
		col = 0
	}
	row := int(bot.Y / (ss.ArenaH / float64(rlStateGridRows)))
	if row >= rlStateGridRows {
		row = rlStateGridRows - 1
	}
	if row < 0 {
		row = 0
	}

	// Carrying state
	carry := 0
	if bot.CarryingPkg >= 0 {
		carry = 1
	}

	// Proximity state
	prox := 0
	if bot.NearestPickupDist < 80 {
		prox = 1
	} else if bot.NearestDropoffDist < 80 {
		prox = 2
	}

	return (col*rlStateGridRows+row)*rlCarryStates*rlProxStates + carry*rlProxStates + prox
}

// RLChooseAction selects an action using epsilon-greedy policy.
func RLChooseAction(rl *RLState, state int, rng *rand.Rand) int {
	if rl == nil || state < 0 || state >= rl.NumStates {
		return 0
	}
	// Epsilon-greedy
	if rng.Float64() < rl.Epsilon {
		return rng.Intn(rl.NumActions)
	}
	// Greedy: pick action with highest Q-value
	bestAction := 0
	bestQ := rl.QTable[state][0]
	for a := 1; a < rl.NumActions; a++ {
		if rl.QTable[state][a] > bestQ {
			bestQ = rl.QTable[state][a]
			bestAction = a
		}
	}
	return bestAction
}

// RLComputeReward computes reward for current bot state.
func RLComputeReward(bot *SwarmBot, ss *SwarmState) float64 {
	reward := 0.0

	// Reward for deliveries
	reward += float64(bot.Stats.TotalDeliveries) * 10.0

	// Reward for pickups
	reward += float64(bot.Stats.TotalPickups) * 3.0

	// Small reward for movement (anti-idle)
	if bot.Stats.TicksIdle < bot.Stats.TicksAlive/2 {
		reward += 0.1
	}

	// Penalty for collisions
	if bot.RecentCollision > 0 {
		reward -= 1.0
	}

	return reward
}

// RLUpdate performs a Q-learning update for one bot.
func RLUpdate(rl *RLState, prevState, action, nextState int, reward float64) {
	if rl == nil || prevState < 0 || prevState >= rl.NumStates {
		return
	}
	if nextState < 0 || nextState >= rl.NumStates {
		return
	}
	if action < 0 || action >= rl.NumActions {
		return
	}

	// Find max Q for next state
	maxQ := rl.QTable[nextState][0]
	for a := 1; a < rl.NumActions; a++ {
		if rl.QTable[nextState][a] > maxQ {
			maxQ = rl.QTable[nextState][a]
		}
	}

	// Q-learning update
	oldQ := rl.QTable[prevState][action]
	rl.QTable[prevState][action] = oldQ + rl.Alpha*(reward+rl.Gamma*maxQ-oldQ)
}

// RLDecayEpsilon reduces exploration rate over time.
func RLDecayEpsilon(rl *RLState) {
	if rl == nil {
		return
	}
	rl.Epsilon *= rl.EpsilonDecay
	if rl.Epsilon < rl.EpsilonMin {
		rl.Epsilon = rl.EpsilonMin
	}
	rl.Episode++
}

// RLRecordReward stores episode reward in history.
func RLRecordReward(rl *RLState, totalReward float64) {
	if rl == nil {
		return
	}
	rl.TotalReward = totalReward
	rl.RewardHistory = append(rl.RewardHistory, totalReward)
	if len(rl.RewardHistory) > rl.MaxHistory {
		rl.RewardHistory = rl.RewardHistory[1:]
	}
	// Compute average
	sum := 0.0
	for _, r := range rl.RewardHistory {
		sum += r
	}
	rl.AvgReward = sum / float64(len(rl.RewardHistory))
}

// RLMaxQ returns the maximum Q-value across all state-action pairs.
func RLMaxQ(rl *RLState) float64 {
	if rl == nil {
		return 0
	}
	maxQ := -math.MaxFloat64
	for s := 0; s < rl.NumStates; s++ {
		for a := 0; a < rl.NumActions; a++ {
			if rl.QTable[s][a] > maxQ {
				maxQ = rl.QTable[s][a]
			}
		}
	}
	if maxQ == -math.MaxFloat64 {
		return 0
	}
	return maxQ
}

// RLNonZeroEntries counts how many Q-table entries have been updated.
func RLNonZeroEntries(rl *RLState) int {
	if rl == nil {
		return 0
	}
	count := 0
	for s := 0; s < rl.NumStates; s++ {
		for a := 0; a < rl.NumActions; a++ {
			if rl.QTable[s][a] != 0 {
				count++
			}
		}
	}
	return count
}
