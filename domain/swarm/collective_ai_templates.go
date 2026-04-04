package swarm

import "fmt"

// generateCodeForIssue produces SwarmScript rules based on problem type and sensor context.
func generateCodeForIssue(ss *SwarmState, issue *SwarmIssue) string {
	// Check proven solutions first (swarm learning!)
	if proven, ok := ss.IssueBoard.ProvenSolutions[issue.Problem]; ok {
		return proven
	}

	switch issue.Problem {
	case "stuck":
		return generateStuckSolution(ss, issue)
	case "no_package":
		return generateNoPackageSolution(ss, issue)
	case "obstacle":
		return generateObstacleSolution(ss, issue)
	case "isolated":
		return generateIsolatedSolution(ss, issue)
	case "slow_delivery":
		return generateSlowDeliverySolution(ss, issue)
	case "energy_crisis":
		return "IF energy < 15 THEN STOP"
	}
	return "IF true THEN TURN_RANDOM"
}

// generateStuckSolution creates rules for stuck bots based on sensor context.
func generateStuckSolution(ss *SwarmState, issue *SwarmIssue) string {
	bot := &ss.Bots[issue.BotIdx]
	variant := ss.Rng.Intn(5)

	switch {
	case bot.ObstacleAhead && variant < 2:
		// Obstacle in front: turn away
		angle := 90 + ss.Rng.Intn(90) // 90-179 degrees
		return fmt.Sprintf("IF obstacle_ahead == 1 THEN TURN_RIGHT %d\nIF true THEN MOVE_FORWARD", angle)

	case bot.ObstacleAhead:
		// Wall following variant
		if bot.WallRight {
			return "IF wall_right == 1 THEN TURN_LEFT 90\nIF obstacle_ahead == 1 THEN TURN_LEFT 45\nIF true THEN MOVE_FORWARD"
		}
		return "IF wall_left == 1 THEN TURN_RIGHT 90\nIF obstacle_ahead == 1 THEN TURN_RIGHT 45\nIF true THEN MOVE_FORWARD"

	case bot.NeighborCount > 3 && variant < 3:
		// Crowded: scatter
		return "IF neighbors_count > 3 THEN TURN_FROM_NEAREST\nIF true THEN MOVE_FORWARD"

	case bot.NeighborCount > 3:
		// Crowded variant: dash away
		return "IF neighbors_count > 2 THEN DASH\nIF true THEN MOVE_FORWARD"

	default:
		// Random exploration with turn
		angle := 30 + ss.Rng.Intn(150) // 30-179 degrees
		return fmt.Sprintf("IF speed < 1 THEN TURN_RIGHT %d\nIF true THEN MOVE_FORWARD", angle)
	}
}

// generateNoPackageSolution creates rules for bots that can't find packages.
func generateNoPackageSolution(ss *SwarmState, issue *SwarmIssue) string {
	bot := &ss.Bots[issue.BotIdx]
	variant := ss.Rng.Intn(4)

	switch {
	case bot.NearestPickupDist > 0 && bot.NearestPickupDist < 300 && variant < 2:
		// Pickup exists but bot can't reach it: approach + pickup
		return "IF carrying == 0 AND nearest_pickup_dist < 200 THEN GOTO_PICKUP\nIF carrying == 0 THEN TURN_RANDOM\nIF true THEN MOVE_FORWARD"

	case variant == 2:
		// Explore with random turns
		return "IF carrying == 0 AND timer == 0 THEN SET_TIMER 30\nIF timer > 0 THEN MOVE_FORWARD\nIF carrying == 0 THEN TURN_RANDOM\nIF true THEN MOVE_FORWARD"

	case variant == 3:
		// Follow pheromone trails to find resources
		return "IF carrying == 0 AND pheromone > 10 THEN FOLLOW_PHER\nIF carrying == 0 THEN TURN_RANDOM\nIF true THEN MOVE_FORWARD"

	default:
		// Simple: go to nearest pickup
		return "IF carrying == 0 THEN GOTO_PICKUP\nIF true THEN MOVE_FORWARD"
	}
}

// generateObstacleSolution creates rules for bots stuck at obstacles.
func generateObstacleSolution(ss *SwarmState, issue *SwarmIssue) string {
	bot := &ss.Bots[issue.BotIdx]
	variant := ss.Rng.Intn(5)

	switch {
	case bot.WallRight && bot.WallLeft:
		// Dead end: U-turn
		return "IF wall_right == 1 AND wall_left == 1 THEN TURN_RIGHT 180\nIF obstacle_ahead == 1 THEN TURN_RIGHT 90\nIF true THEN MOVE_FORWARD"

	case bot.WallRight && variant < 2:
		// Right wall: follow left
		return "IF wall_right == 1 THEN TURN_LEFT 45\nIF obstacle_ahead == 1 THEN TURN_LEFT 90\nIF true THEN MOVE_FORWARD"

	case bot.WallLeft && variant < 2:
		// Left wall: follow right
		return "IF wall_left == 1 THEN TURN_RIGHT 45\nIF obstacle_ahead == 1 THEN TURN_RIGHT 90\nIF true THEN MOVE_FORWARD"

	case variant == 2:
		// Wall following: right-hand rule
		return "IF obstacle_ahead == 1 THEN TURN_LEFT 90\nIF wall_right == 0 THEN TURN_RIGHT 90\nIF true THEN MOVE_FORWARD"

	case variant == 3:
		// Wall following: left-hand rule
		return "IF obstacle_ahead == 1 THEN TURN_RIGHT 90\nIF wall_left == 0 THEN TURN_LEFT 90\nIF true THEN MOVE_FORWARD"

	default:
		// Random avoidance
		angle := 60 + ss.Rng.Intn(120)
		if ss.Rng.Intn(2) == 0 {
			return fmt.Sprintf("IF obstacle_ahead == 1 THEN TURN_LEFT %d\nIF true THEN MOVE_FORWARD", angle)
		}
		return fmt.Sprintf("IF obstacle_ahead == 1 THEN TURN_RIGHT %d\nIF true THEN MOVE_FORWARD", angle)
	}
}

// generateIsolatedSolution creates rules for isolated bots with no neighbors.
func generateIsolatedSolution(ss *SwarmState, issue *SwarmIssue) string {
	variant := ss.Rng.Intn(4)

	switch variant {
	case 0:
		// Move toward center of arena
		return "IF neighbors_count == 0 THEN TURN_TO_CENTER\nIF true THEN MOVE_FORWARD"
	case 1:
		// Spiral search pattern
		return "IF neighbors_count == 0 AND timer == 0 THEN SET_TIMER 20\nIF timer > 10 THEN MOVE_FORWARD\nIF timer > 0 THEN TURN_RIGHT 15\nIF true THEN MOVE_FORWARD"
	case 2:
		// Random walk with larger turns
		return "IF neighbors_count == 0 AND random < 30 THEN TURN_RANDOM\nIF true THEN MOVE_FORWARD"
	default:
		// Go toward nearest detected bot
		return "IF neighbors_count == 0 THEN TURN_TO_NEAREST\nIF true THEN MOVE_FORWARD"
	}
}

// generateSlowDeliverySolution creates rules for slow delivery bots.
func generateSlowDeliverySolution(ss *SwarmState, issue *SwarmIssue) string {
	bot := &ss.Bots[issue.BotIdx]
	variant := ss.Rng.Intn(4)

	switch {
	case bot.CarryingPkg >= 0 && variant < 2:
		// Carrying but slow: focus on dropoff
		return "IF carrying == 1 AND dropoff_match == 1 THEN GOTO_DROPOFF\nIF carrying == 1 THEN TURN_RANDOM\nIF true THEN MOVE_FORWARD"

	case bot.CarryingPkg >= 0:
		// Carrying: use beacon if available
		return "IF carrying == 1 AND heard_beacon == 1 THEN GOTO_BEACON\nIF carrying == 1 THEN GOTO_DROPOFF\nIF true THEN MOVE_FORWARD"

	case variant == 3:
		// Not carrying: pickup faster
		return "IF carrying == 0 AND nearest_pickup_dist < 200 AND nearest_pickup_has_package == 1 THEN GOTO_PICKUP\nIF carrying == 0 THEN TURN_RANDOM\nIF true THEN MOVE_FORWARD"

	default:
		// General delivery improvement
		return "IF carrying == 0 THEN GOTO_PICKUP\nIF carrying == 1 THEN GOTO_DROPOFF\nIF true THEN MOVE_FORWARD"
	}
}
