package swarm

import "math"

// Dynamic Role Assignment: Bots specialize as Scout, Worker, or Guard
// based on local conditions. Roles change dynamically according to demand.
// Uses bot.Role (int) field separate from the SpecRole type in specialization.go.

const (
	BotRoleNone   = 0
	BotRoleScout  = 1 // explore unknown areas, low neighbor count
	BotRoleWorker = 2 // deliver packages, carrying or near stations
	BotRoleGuard  = 3 // protect area, high neighbor density
)

// TickRoles computes role demand sensors for all bots.
// Must be called after spatial hash and basic sensors are built.
func TickRoles(ss *SwarmState) {
	// Count global role distribution
	var scouts, workers, guards, total int
	for i := range ss.Bots {
		total++
		switch ss.Bots[i].Role {
		case BotRoleScout:
			scouts++
		case BotRoleWorker:
			workers++
		case BotRoleGuard:
			guards++
		}
	}
	if total == 0 {
		return
	}

	// Ideal ratios depend on delivery mode
	idealScout := 0.30
	idealWorker := 0.50
	idealGuard := 0.20
	if !ss.DeliveryOn {
		idealScout = 0.50
		idealWorker = 0.20
		idealGuard = 0.30
	}

	// Demand: how much each role is under-represented (0-100)
	scoutDemand := int(math.Max(0, (idealScout-float64(scouts)/float64(total))*200))
	workerDemand := int(math.Max(0, (idealWorker-float64(workers)/float64(total))*200))
	guardDemand := int(math.Max(0, (idealGuard-float64(guards)/float64(total))*200))

	// Cap at 100
	if scoutDemand > 100 {
		scoutDemand = 100
	}
	if workerDemand > 100 {
		workerDemand = 100
	}
	if guardDemand > 100 {
		guardDemand = 100
	}

	// Find which role has highest demand
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		// Local demand adjustment: factor in personal situation
		localScout := scoutDemand
		localWorker := workerDemand
		localGuard := guardDemand

		// If isolated → scout demand up
		if bot.IsolationLevel > 50 {
			localScout += 20
		}
		// If carrying → worker demand up
		if bot.CarryingPkg >= 0 {
			localWorker += 30
		}
		// If many neighbors → guard demand up
		if bot.NeighborCount > 5 {
			localGuard += 20
		}

		// Role demand = most needed role (1=scout, 2=worker, 3=guard)
		maxDemand := localScout
		bot.RoleDemand = BotRoleScout
		if localWorker > maxDemand {
			maxDemand = localWorker
			bot.RoleDemand = BotRoleWorker
		}
		if localGuard > maxDemand {
			bot.RoleDemand = BotRoleGuard
		}
	}
}

// SetRole assigns a role to a bot and sets the LED color accordingly.
func SetRole(bot *SwarmBot, role int) {
	bot.Role = role
	switch role {
	case BotRoleScout:
		bot.LEDColor = [3]uint8{0, 200, 255}   // cyan
	case BotRoleWorker:
		bot.LEDColor = [3]uint8{255, 200, 0}   // gold
	case BotRoleGuard:
		bot.LEDColor = [3]uint8{255, 50, 50}   // red
	}
}
