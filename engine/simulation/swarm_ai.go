package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"swarmsim/domain/physics"
	"swarmsim/domain/swarm"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
)

// updateSwarmMode is the main update loop for programmable swarm mode.
func (s *Simulation) updateSwarmMode() {
	ss := s.SwarmState
	ss.Tick++

	// Reset per-tick counters
	ss.CollisionCount = 0

	// Swap message buffers: this tick reads PrevMessages, writes to NextMessages
	ss.PrevMessages = ss.NextMessages
	ss.NextMessages = nil

	// Rebuild spatial hash
	ss.Hash.Clear()
	for i := range ss.Bots {
		ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
	}

	// Decrement timers
	for i := range ss.Bots {
		if ss.Bots[i].Timer > 0 {
			ss.Bots[i].Timer--
		}
		if ss.Bots[i].BlinkTimer > 0 {
			ss.Bots[i].BlinkTimer--
		}
	}

	// Phase 1: Build environment (sensor values) for each bot
	for i := range ss.Bots {
		buildSwarmEnvironment(ss, i)
	}

	// Phase 2: Execute program on each bot (skip if anti-stuck breakout active)
	if ss.Program != nil {
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			if bot.AntiStuckTimer > 0 {
				// Breakout mode: forced random movement, ignore program
				bot.Angle = bot.AntiStuckAngle
				bot.Speed = swarm.SwarmBotSpeed
				bot.LEDColor = [3]uint8{255, 255, 255} // white LED
				bot.AntiStuckTimer--
				continue
			}
			executeSwarmProgram(ss, i)
		}
	}

	// Phase 2.3: Max neighbors cap — force scatter when too crowded
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.AntiStuckTimer > 0 {
			continue // already in breakout
		}
		if bot.CloseNeighbors > 4 && bot.NearestIdx >= 0 {
			// Force turn away from nearest + full speed
			other := &ss.Bots[bot.NearestIdx]
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
			bot.Angle = math.Atan2(-dy, -dx) + (ss.Rng.Float64()-0.5)*0.6
			bot.Speed = swarm.SwarmBotSpeed
		}
	}

	// Phase 2.5: Follow behavior override
	for i := range ss.Bots {
		if ss.Bots[i].AntiStuckTimer > 0 {
			continue // breakout overrides follow
		}
		applyFollowBehavior(ss, i)
	}

	// Phase 2.6: Record trails (every 3 ticks)
	if ss.ShowTrails && ss.Tick%3 == 0 {
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			bot.Trail[bot.TrailIdx] = [2]float64{bot.X, bot.Y}
			bot.TrailIdx = (bot.TrailIdx + 1) % len(bot.Trail)
		}
	}

	// Phase 3: Broadcast messages
	for i := range ss.Bots {
		if ss.Bots[i].PendingMsg > 0 {
			ss.NextMessages = append(ss.NextMessages, swarm.SwarmMessage{
				Value: ss.Bots[i].PendingMsg,
				X:     ss.Bots[i].X,
				Y:     ss.Bots[i].Y,
			})
			ss.Bots[i].Stats.MessagesSent++
		}
	}

	// Phase 4: Physics — move bots, clamp to bounds
	for i := range ss.Bots {
		applySwarmPhysics(ss, i)
	}

	// Phase 4.1: Hard separation — symmetric rigid-body push for all pairs
	applyHardSeparation(ss)

	// Phase 4.2: Repulsion force — active push when bots are closer than 30px
	applyRepulsionForce(ss)

	// Phase 4.5: Delivery system updates (pickup/drop, respawn, carried package position)
	if ss.DeliveryOn {
		swarm.UpdateDeliverySystem(ss)
		// Periodic delivery stats log
		if ss.Tick%600 == 0 {
			carrying := 0
			for ci := range ss.Bots {
				if ss.Bots[ci].CarryingPkg >= 0 {
					carrying++
				}
			}
			fmt.Printf("[DELIVERY] tick=%d delivered=%d (correct=%d wrong=%d) carrying=%d\n",
				ss.Tick, ss.DeliveryStats.TotalDelivered, ss.DeliveryStats.CorrectDelivered,
				ss.DeliveryStats.WrongDelivered, carrying)
		}
	}

	// Phase 4.6: Truck system updates
	if ss.TruckToggle && ss.TruckState != nil {
		swarm.UpdateSwarmTruck(ss)
	}

	// Phase 4.9: Accumulate lifetime stats (before StuckPrevX/Y is overwritten)
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		ddx := bot.X - bot.StuckPrevX
		ddy := bot.Y - bot.StuckPrevY
		bot.Stats.TotalDistance += math.Sqrt(ddx*ddx + ddy*ddy)
		bot.Stats.TicksAlive++
		if bot.Speed == 0 {
			bot.Stats.TicksIdle++
		}
		if bot.CarryingPkg >= 0 {
			bot.Stats.TicksCarrying++
		}
	}

	// Phase 5: Anti-stuck detection & cooldown
	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Decrement stuck cooldown
		if bot.StuckCooldown > 0 {
			bot.StuckCooldown--
		}

		// Skip stuck detection if already in breakout
		if bot.AntiStuckTimer > 0 {
			bot.StuckTicks = 0
			bot.StuckPrevX = bot.X
			bot.StuckPrevY = bot.Y
			continue
		}

		// Measure movement since last tick
		dx := bot.X - bot.StuckPrevX
		dy := bot.Y - bot.StuckPrevY
		moved := math.Sqrt(dx*dx + dy*dy)

		// Count as stuck if bot moved < 3px cumulative over many ticks
		if moved < 0.5 && bot.Speed > 0 {
			bot.StuckTicks++
		} else {
			if bot.StuckTicks > 0 {
				bot.StuckTicks -= 2 // slow decay when moving
				if bot.StuckTicks < 0 {
					bot.StuckTicks = 0
				}
			}
		}

		// Anti-stuck breakout after 90 ticks stuck
		if bot.StuckTicks >= 90 {
			// Break follow link
			if bot.FollowTargetIdx >= 0 && bot.FollowTargetIdx < len(ss.Bots) {
				ss.Bots[bot.FollowTargetIdx].FollowerIdx = -1
			}
			bot.FollowTargetIdx = -1
			if bot.FollowerIdx >= 0 && bot.FollowerIdx < len(ss.Bots) {
				ss.Bots[bot.FollowerIdx].FollowTargetIdx = -1
			}
			bot.FollowerIdx = -1

			// Activate breakout: random direction, full speed for 45 ticks
			bot.AntiStuckAngle = ss.Rng.Float64() * 2 * math.Pi
			bot.AntiStuckTimer = 45
			bot.StuckCooldown = 30
			bot.StuckTicks = 0
			bot.Stats.AntiStuckCount++
			logger.WarnBot(i, "STUCK", "Bot #%d anti-stuck at (%.0f, %.0f)", i, bot.X, bot.Y)
		} else if bot.StuckTicks >= 60 {
			// Legacy: auto-unfollow at 60 ticks (before full breakout at 90)
			if bot.FollowTargetIdx >= 0 && bot.FollowTargetIdx < len(ss.Bots) {
				ss.Bots[bot.FollowTargetIdx].FollowerIdx = -1
			}
			bot.FollowTargetIdx = -1
			if bot.FollowerIdx >= 0 && bot.FollowerIdx < len(ss.Bots) {
				ss.Bots[bot.FollowerIdx].FollowTargetIdx = -1
			}
			bot.FollowerIdx = -1
			bot.Angle = ss.Rng.Float64() * 2 * math.Pi
			bot.Speed = swarm.SwarmBotSpeed
			bot.StuckCooldown = 30
		}

		// Save position for next tick
		bot.StuckPrevX = bot.X
		bot.StuckPrevY = bot.Y
	}
}

// buildSwarmEnvironment computes sensor values for bot at index i.
func buildSwarmEnvironment(ss *swarm.SwarmState, i int) {
	bot := &ss.Bots[i]

	// Reset sensor values
	bot.NeighborCount = 0
	bot.CloseNeighbors = 0
	bot.NearestDist = 1e9
	bot.NearestAngle = 0
	bot.AvgNeighborX = 0
	bot.AvgNeighborY = 0
	bot.OnEdge = false
	bot.ReceivedMsg = 0
	bot.LightValue = 0
	bot.NearestIdx = -1
	bot.NearestLEDR = 0
	bot.NearestLEDG = 0
	bot.NearestLEDB = 0
	bot.ObstacleAhead = false
	bot.ObstacleDist = 999

	// Query neighbors within sensor range
	candidateIDs := ss.Hash.Query(bot.X, bot.Y, swarm.SwarmSensorRange)
	var sumX, sumY float64
	count := 0

	for _, cid := range candidateIDs {
		if cid == i || cid < 0 || cid >= len(ss.Bots) {
			continue
		}
		other := &ss.Bots[cid]

		// Compute distance (with wrap-mode support)
		dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > swarm.SwarmSensorRange {
			continue
		}
		count++
		if dist < 30 {
			bot.CloseNeighbors++
		}
		sumX += dx
		sumY += dy
		if dist < bot.NearestDist {
			bot.NearestDist = dist
			bot.NearestAngle = math.Atan2(dy, dx)
			bot.NearestIdx = cid
			bot.NearestLEDR = other.LEDColor[0]
			bot.NearestLEDG = other.LEDColor[1]
			bot.NearestLEDB = other.LEDColor[2]
		}
	}

	bot.NeighborCount = count
	if count > 0 {
		bot.AvgNeighborX = sumX / float64(count)
		bot.AvgNeighborY = sumY / float64(count)
	}
	if bot.NearestDist > 1e8 {
		bot.NearestDist = 999
	}

	// On edge check
	if bot.X < swarm.SwarmEdgeMargin || bot.X > ss.ArenaW-swarm.SwarmEdgeMargin ||
		bot.Y < swarm.SwarmEdgeMargin || bot.Y > ss.ArenaH-swarm.SwarmEdgeMargin {
		bot.OnEdge = true
	}

	// Received message — find first message within comm range
	for _, msg := range ss.PrevMessages {
		dx, dy := swarm.NeighborDelta(bot.X, bot.Y, msg.X, msg.Y, ss)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= swarm.SwarmCommRange {
			bot.ReceivedMsg = msg.Value
			bot.Stats.MessagesReceived++
			break
		}
	}

	// Light value
	if ss.Light.Active {
		dx, dy := swarm.NeighborDelta(bot.X, bot.Y, ss.Light.X, ss.Light.Y, ss)
		dist := math.Sqrt(dx*dx + dy*dy)
		lv := 100 - int(dist/5.0)
		if lv < 0 {
			lv = 0
		}
		if lv > 100 {
			lv = 100
		}
		bot.LightValue = lv
	}

	// Delivery sensors: scan visible stations
	bot.NearestPickupDist = 999
	bot.NearestPickupColor = 0
	bot.NearestPickupHasPkg = false
	bot.NearestPickupIdx = -1
	bot.NearestDropoffDist = 999
	bot.NearestDropoffColor = 0
	bot.NearestDropoffIdx = -1
	bot.DropoffMatch = false
	bot.HeardPickupColor = 0
	bot.HeardDropoffColor = 0
	bot.NearestMatchLEDDist = 999
	bot.NearestMatchLEDAngle = 0

	if ss.DeliveryOn {
		// Use extended sensor range for delivery station scanning
		delivRange := swarm.SwarmDeliverySensorRange

		for si := range ss.Stations {
			st := &ss.Stations[si]
			sdx, sdy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			sdist := math.Sqrt(sdx*sdx + sdy*sdy)
			if sdist > delivRange {
				continue
			}
			if st.IsPickup {
				if sdist < bot.NearestPickupDist {
					bot.NearestPickupDist = sdist
					bot.NearestPickupColor = st.Color
					bot.NearestPickupHasPkg = st.HasPackage
					bot.NearestPickupIdx = si
				}
			} else {
				if sdist < bot.NearestDropoffDist {
					bot.NearestDropoffDist = sdist
					bot.NearestDropoffColor = st.Color
					bot.NearestDropoffIdx = si
				}
				// Check match against ANY visible dropoff (not just nearest)
				if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
					if ss.Packages[bot.CarryingPkg].Color == st.Color {
						bot.DropoffMatch = true
					}
				}
			}
		}
		// Also check ground packages as "pickups" (nearby packages on the ground)
		for pi := range ss.Packages {
			pkg := &ss.Packages[pi]
			if !pkg.Active || pkg.CarriedBy >= 0 || !pkg.OnGround {
				continue
			}
			pdx, pdy := swarm.NeighborDelta(bot.X, bot.Y, pkg.X, pkg.Y, ss)
			pdist := math.Sqrt(pdx*pdx + pdy*pdy)
			if pdist < bot.NearestPickupDist && pdist <= delivRange {
				bot.NearestPickupDist = pdist
				bot.NearestPickupColor = pkg.Color
				bot.NearestPickupHasPkg = true
			}
		}

		// Decode delivery messages from PrevMessages
		// Values 10-14 = PICKUP color 1-4, values 20-24 = DROPOFF color 1-4
		for _, msg := range ss.PrevMessages {
			mdx, mdy := swarm.NeighborDelta(bot.X, bot.Y, msg.X, msg.Y, ss)
			mdist := math.Sqrt(mdx*mdx + mdy*mdy)
			if mdist > swarm.SwarmCommRange {
				continue
			}
			if msg.Value >= 11 && msg.Value <= 14 && bot.HeardPickupColor == 0 {
				bot.HeardPickupColor = msg.Value - 10
				bot.HeardPickupAngle = math.Atan2(mdy, mdx)
			}
			if msg.Value >= 21 && msg.Value <= 24 && bot.HeardDropoffColor == 0 {
				bot.HeardDropoffColor = msg.Value - 20
				bot.HeardDropoffAngle = math.Atan2(mdy, mdx)
			}
		}

		// LED-based pheromone matching: find nearest bot whose LED matches carrying color
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			carryColor := ss.Packages[bot.CarryingPkg].Color
			// Scan neighbors with extended delivery range
			ledCandidates := ss.Hash.Query(bot.X, bot.Y, delivRange)
			for _, cid := range ledCandidates {
				if cid == i || cid < 0 || cid >= len(ss.Bots) {
					continue
				}
				other := &ss.Bots[cid]
				// Check if other bot's LED matches our carrying color
				otherLEDColor := ledToDeliveryColor(other.LEDColor)
				if otherLEDColor != carryColor {
					continue
				}
				ldx, ldy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
				ldist := math.Sqrt(ldx*ldx + ldy*ldy)
				if ldist > delivRange {
					continue
				}
				if ldist < bot.NearestMatchLEDDist {
					bot.NearestMatchLEDDist = ldist
					bot.NearestMatchLEDAngle = math.Atan2(ldy, ldx)
				}
			}
		}
	}

	// Obstacle raycast — check 10 steps at 5px in facing direction (50px lookahead)
	allObs := ss.AllObstacles()
	if len(allObs) > 0 {
		for step := 1; step <= 10; step++ {
			px := bot.X + math.Cos(bot.Angle)*float64(step)*5.0
			py := bot.Y + math.Sin(bot.Angle)*float64(step)*5.0
			for _, obs := range allObs {
				if pointInRect(px, py, obs) {
					bot.ObstacleAhead = true
					d := float64(step) * 5.0
					if d < bot.ObstacleDist {
						bot.ObstacleDist = d
					}
					break
				}
			}
			if bot.ObstacleAhead {
				break
			}
		}
	}
}

// pointInRect checks if a point is inside an obstacle rect.
func pointInRect(px, py float64, obs *physics.Obstacle) bool {
	return px >= obs.X && px <= obs.X+obs.W && py >= obs.Y && py <= obs.Y+obs.H
}

// ledToDeliveryColor maps an LED color to a delivery color (1-4), or 0 if no match.
// Uses generous thresholds to allow slight color variations.
func ledToDeliveryColor(led [3]uint8) int {
	r, g, b := int(led[0]), int(led[1]), int(led[2])
	// White/default = no delivery color
	if r > 200 && g > 200 && b > 200 {
		return 0
	}
	// Black/dim = no color
	if r < 50 && g < 50 && b < 50 {
		return 0
	}
	// Red: R dominant
	if r > 180 && g < 120 && b < 120 {
		return 1
	}
	// Blue: B dominant
	if b > 180 && r < 120 {
		return 2
	}
	// Yellow: R+G high, B low
	if r > 180 && g > 150 && b < 100 {
		return 3
	}
	// Green: G dominant
	if g > 150 && r < 120 && b < 120 {
		return 4
	}
	return 0
}

// deliveryColorRGB maps a delivery color (1-4) to LED RGB values.
func deliveryColorRGB(c int) (uint8, uint8, uint8) {
	switch c {
	case 1:
		return 255, 60, 60 // red
	case 2:
		return 60, 100, 255 // blue
	case 3:
		return 255, 220, 40 // yellow
	case 4:
		return 40, 200, 60 // green
	}
	return 200, 200, 200 // default gray
}

// executeSwarmProgram runs all matching rules on a bot.
// Conditions evaluate against a snapshot; actions mutate the bot live.
func executeSwarmProgram(ss *swarm.SwarmState, i int) {
	bot := &ss.Bots[i]
	prog := ss.Program

	// Snapshot mutable vars for condition evaluation
	snapState := bot.State
	snapCounter := bot.Counter
	snapTimer := bot.Timer

	// Reset per-tick outputs
	bot.Speed = 0
	bot.PendingMsg = 0

	for _, rule := range prog.Rules {
		// Evaluate all conditions
		allMatch := true
		for _, cond := range rule.Conditions {
			if !evaluateSwarmCondition(cond, bot, snapState, snapCounter, snapTimer, ss.Rng, ss, i) {
				allMatch = false
				break
			}
		}
		if allMatch {
			executeSwarmAction(rule.Action, bot, ss, i)
		}
	}
}

// evaluateSwarmCondition checks a single condition against bot sensors.
func evaluateSwarmCondition(cond swarmscript.Condition, bot *swarm.SwarmBot, snapState, snapCounter, snapTimer int, rng *rand.Rand, ss *swarm.SwarmState, botIdx int) bool {
	switch cond.Type {
	case swarmscript.CondTrue:
		return true

	case swarmscript.CondNeighborsCount:
		return compareInt(bot.NeighborCount, cond.Op, cond.Value)

	case swarmscript.CondNearestDistance:
		return compareInt(int(bot.NearestDist), cond.Op, cond.Value)

	case swarmscript.CondState, swarmscript.CondMyState:
		return compareInt(snapState, cond.Op, cond.Value)

	case swarmscript.CondCounter:
		return compareInt(snapCounter, cond.Op, cond.Value)

	case swarmscript.CondTimer:
		return compareInt(snapTimer, cond.Op, cond.Value)

	case swarmscript.CondOnEdge:
		onEdgeVal := 0
		if bot.OnEdge {
			onEdgeVal = 1
		}
		return compareInt(onEdgeVal, cond.Op, cond.Value)

	case swarmscript.CondReceivedMessage:
		return compareInt(bot.ReceivedMsg, cond.Op, cond.Value)

	case swarmscript.CondLightValue:
		return compareInt(bot.LightValue, cond.Op, cond.Value)

	case swarmscript.CondRandom:
		// random < N means N% chance
		return rng.Intn(100) < cond.Value

	case swarmscript.CondHasLeader:
		hasLeader := 0
		if bot.FollowTargetIdx >= 0 {
			hasLeader = 1
		}
		return compareInt(hasLeader, cond.Op, cond.Value)

	case swarmscript.CondHasFollower:
		hasFollower := 0
		if bot.FollowerIdx >= 0 {
			hasFollower = 1
		}
		return compareInt(hasFollower, cond.Op, cond.Value)

	case swarmscript.CondChainLength:
		cl := computeChainLength(ss, botIdx)
		return compareInt(cl, cond.Op, cond.Value)

	case swarmscript.CondNearestLEDR:
		return compareInt(int(bot.NearestLEDR), cond.Op, cond.Value)

	case swarmscript.CondNearestLEDG:
		return compareInt(int(bot.NearestLEDG), cond.Op, cond.Value)

	case swarmscript.CondNearestLEDB:
		return compareInt(int(bot.NearestLEDB), cond.Op, cond.Value)

	case swarmscript.CondTick:
		return compareInt(ss.Tick, cond.Op, cond.Value)

	case swarmscript.CondObstacleAhead:
		obsVal := 0
		if bot.ObstacleAhead {
			obsVal = 1
		}
		return compareInt(obsVal, cond.Op, cond.Value)

	case swarmscript.CondObstacleDist:
		return compareInt(int(bot.ObstacleDist), cond.Op, cond.Value)

	case swarmscript.CondValue1:
		return compareInt(bot.Value1, cond.Op, cond.Value)

	case swarmscript.CondValue2:
		return compareInt(bot.Value2, cond.Op, cond.Value)

	case swarmscript.CondCarrying:
		carryVal := 0
		if bot.CarryingPkg >= 0 {
			carryVal = 1
		}
		return compareInt(carryVal, cond.Op, cond.Value)

	case swarmscript.CondCarryingColor:
		cc := 0
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			cc = ss.Packages[bot.CarryingPkg].Color
		}
		return compareInt(cc, cond.Op, cond.Value)

	case swarmscript.CondNearestPickupDist:
		return compareInt(int(bot.NearestPickupDist), cond.Op, cond.Value)

	case swarmscript.CondNearestPickupColor:
		return compareInt(bot.NearestPickupColor, cond.Op, cond.Value)

	case swarmscript.CondNearestPickupHasPkg:
		v := 0
		if bot.NearestPickupHasPkg {
			v = 1
		}
		return compareInt(v, cond.Op, cond.Value)

	case swarmscript.CondNearestDropoffDist:
		return compareInt(int(bot.NearestDropoffDist), cond.Op, cond.Value)

	case swarmscript.CondNearestDropoffColor:
		return compareInt(bot.NearestDropoffColor, cond.Op, cond.Value)

	case swarmscript.CondDropoffMatch:
		v := 0
		if bot.DropoffMatch {
			v = 1
		}
		return compareInt(v, cond.Op, cond.Value)

	case swarmscript.CondHeardPickupColor:
		return compareInt(bot.HeardPickupColor, cond.Op, cond.Value)

	case swarmscript.CondHeardDropoffColor:
		return compareInt(bot.HeardDropoffColor, cond.Op, cond.Value)

	case swarmscript.CondNearestMatchLEDDist:
		return compareInt(int(bot.NearestMatchLEDDist), cond.Op, cond.Value)
	}

	return false
}

// computeChainLength walks the follow chain in both directions from bot i and returns total length.
func computeChainLength(ss *swarm.SwarmState, i int) int {
	count := 1

	// Walk up (follow leaders)
	cur := ss.Bots[i].FollowTargetIdx
	visited := make(map[int]bool)
	visited[i] = true
	for cur >= 0 && cur < len(ss.Bots) && !visited[cur] {
		visited[cur] = true
		count++
		cur = ss.Bots[cur].FollowTargetIdx
	}

	// Walk down (follow followers)
	cur = ss.Bots[i].FollowerIdx
	for cur >= 0 && cur < len(ss.Bots) && !visited[cur] {
		visited[cur] = true
		count++
		cur = ss.Bots[cur].FollowerIdx
	}

	return count
}

// compareInt compares two ints with the given operator.
func compareInt(a int, op swarmscript.ConditionOp, b int) bool {
	switch op {
	case swarmscript.OpGT:
		return a > b
	case swarmscript.OpLT:
		return a < b
	case swarmscript.OpEQ:
		return a == b
	}
	return false
}

// executeSwarmAction performs an action on a bot.
func executeSwarmAction(act swarmscript.Action, bot *swarm.SwarmBot, ss *swarm.SwarmState, botIdx int) {
	switch act.Type {
	case swarmscript.ActMoveForward:
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActTurnLeft:
		bot.Angle -= float64(act.Param1) * math.Pi / 180.0

	case swarmscript.ActTurnRight:
		bot.Angle += float64(act.Param1) * math.Pi / 180.0

	case swarmscript.ActTurnToNearest:
		if bot.NeighborCount > 0 {
			bot.Angle = bot.NearestAngle
		}

	case swarmscript.ActTurnFromNearest:
		if bot.NeighborCount > 0 {
			bot.Angle = bot.NearestAngle + math.Pi
		}

	case swarmscript.ActTurnToCenter:
		if bot.NeighborCount > 0 {
			bot.Angle = math.Atan2(bot.AvgNeighborY, bot.AvgNeighborX)
		}

	case swarmscript.ActTurnToLight:
		if ss.Light.Active {
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, ss.Light.X, ss.Light.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
		}

	case swarmscript.ActTurnRandom:
		bot.Angle = ss.Rng.Float64() * 2 * math.Pi

	case swarmscript.ActStop:
		bot.Speed = 0

	case swarmscript.ActSetState:
		bot.State = act.Param1

	case swarmscript.ActSetCounter:
		bot.Counter = act.Param1

	case swarmscript.ActIncCounter:
		bot.Counter++

	case swarmscript.ActDecCounter:
		bot.Counter--

	case swarmscript.ActSetLED:
		r := act.Param1
		g := act.Param2
		b := act.Param3
		if r < 0 {
			r = 0
		}
		if r > 255 {
			r = 255
		}
		if g < 0 {
			g = 0
		}
		if g > 255 {
			g = 255
		}
		if b < 0 {
			b = 0
		}
		if b > 255 {
			b = 255
		}
		bot.LEDColor = [3]uint8{uint8(r), uint8(g), uint8(b)}

	case swarmscript.ActSendMessage:
		bot.PendingMsg = act.Param1

	case swarmscript.ActSetTimer:
		bot.Timer = act.Param1

	case swarmscript.ActFollowNearest:
		// Find nearest bot within sensor range that has no follower (= chain tail or lone bot)
		if bot.FollowTargetIdx >= 0 {
			return // already following someone
		}
		if bot.StuckCooldown > 0 {
			return // in anti-stuck cooldown, forced solo movement
		}
		// Scan ALL neighbors for best available target (not just single nearest)
		candidateIDs := ss.Hash.Query(bot.X, bot.Y, swarm.SwarmSensorRange)
		bestIdx := -1
		bestDist := 1e9
		for _, cid := range candidateIDs {
			if cid == botIdx || cid < 0 || cid >= len(ss.Bots) {
				continue
			}
			other := &ss.Bots[cid]
			// Must not already have a follower
			if other.FollowerIdx >= 0 {
				continue
			}
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > swarm.SwarmSensorRange {
				continue
			}
			if dist < bestDist {
				bestDist = dist
				bestIdx = cid
			}
		}
		if bestIdx >= 0 {
			// Cycle detection: walk from target through its leader chain
			wouldCycle := false
			cur := bestIdx
			for steps := 0; steps < len(ss.Bots); steps++ {
				if cur == botIdx {
					wouldCycle = true
					break
				}
				next := ss.Bots[cur].FollowTargetIdx
				if next < 0 || next >= len(ss.Bots) {
					break
				}
				cur = next
			}
			if !wouldCycle {
				bot.FollowTargetIdx = bestIdx
				ss.Bots[bestIdx].FollowerIdx = botIdx
				logger.InfoBot(botIdx, "FOLLOW", "Bot #%d following Bot #%d", botIdx, bestIdx)
			}
		}

	case swarmscript.ActUnfollow:
		if bot.FollowTargetIdx >= 0 && bot.FollowTargetIdx < len(ss.Bots) {
			logger.InfoBot(botIdx, "FOLLOW", "Bot #%d unfollowed Bot #%d", botIdx, bot.FollowTargetIdx)
			// Clear leader's follower reference
			ss.Bots[bot.FollowTargetIdx].FollowerIdx = -1
		}
		bot.FollowTargetIdx = -1

	case swarmscript.ActTurnAwayObstacle:
		if bot.ObstacleAhead {
			// Turn 90° + random 0-45° in a random direction to avoid corner loops
			turn := math.Pi/2 + ss.Rng.Float64()*math.Pi/4
			if ss.Rng.Intn(2) == 0 {
				turn = -turn
			}
			bot.Angle += turn
		}

	case swarmscript.ActMoveForwardSlow:
		bot.Speed = swarm.SwarmBotSpeed * 0.5

	case swarmscript.ActSetValue1:
		bot.Value1 = act.Param1

	case swarmscript.ActSetValue2:
		bot.Value2 = act.Param1

	case swarmscript.ActCopyNearestLED:
		if ss.DeliveryOn {
			// In delivery mode, use extended range and only copy delivery colors (not white)
			copyRange := swarm.SwarmDeliverySensorRange
			candidates := ss.Hash.Query(bot.X, bot.Y, copyRange)
			bestIdx := -1
			bestDist := 1e9
			for _, cid := range candidates {
				if cid == botIdx || cid < 0 || cid >= len(ss.Bots) {
					continue
				}
				other := &ss.Bots[cid]
				// Only copy from bots that have a delivery color (not white/default)
				if ledToDeliveryColor(other.LEDColor) == 0 {
					continue
				}
				dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist <= copyRange && dist < bestDist {
					bestDist = dist
					bestIdx = cid
				}
			}
			if bestIdx >= 0 {
				bot.LEDColor = ss.Bots[bestIdx].LEDColor
			}
		} else {
			// Standard behavior: copy from nearest neighbor (60px)
			if bot.NearestIdx >= 0 && bot.NearestIdx < len(ss.Bots) {
				other := &ss.Bots[bot.NearestIdx]
				bot.LEDColor = other.LEDColor
			}
		}

	case swarmscript.ActPickup:
		if !ss.DeliveryOn || bot.CarryingPkg >= 0 {
			return
		}
		// Check ground packages first (closer interaction)
		for pi := range ss.Packages {
			pkg := &ss.Packages[pi]
			if !pkg.Active || pkg.CarriedBy >= 0 {
				continue
			}
			pdx, pdy := swarm.NeighborDelta(bot.X, bot.Y, pkg.X, pkg.Y, ss)
			if math.Sqrt(pdx*pdx+pdy*pdy) < 20 {
				pkg.CarriedBy = botIdx
				pkg.OnGround = false
				pkg.PickupTick = ss.Tick
				bot.CarryingPkg = pi
				bot.Stats.TotalPickups++
				logger.InfoBot(botIdx, "DELIVERY", "Bot #%d picked up %s package", botIdx, swarm.DeliveryColorName(pkg.Color))
				// Emit pickup event for particles
				ss.DeliveryEvents = append(ss.DeliveryEvents, swarm.SwarmDeliveryEvent{
					X: pkg.X, Y: pkg.Y,
					Color: pkg.Color, IsPickup: true,
				})
				// Mark station as empty if this was a station package
				for si := range ss.Stations {
					st := &ss.Stations[si]
					if st.IsPickup && st.HasPackage && st.Color == pkg.Color {
						sdx, sdy := swarm.NeighborDelta(pkg.X, pkg.Y, st.X, st.Y, ss)
						if math.Sqrt(sdx*sdx+sdy*sdy) < 30 {
							st.HasPackage = false
							break
						}
					}
				}
				return
			}
		}

	case swarmscript.ActDrop:
		if !ss.DeliveryOn || bot.CarryingPkg < 0 || bot.CarryingPkg >= len(ss.Packages) {
			return
		}
		pkg := &ss.Packages[bot.CarryingPkg]
		// Check if at a dropoff station
		delivered := false
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup {
				continue
			}
			sdx, sdy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			if math.Sqrt(sdx*sdx+sdy*sdy) < 25 {
				// Delivery!
				ss.DeliveryStats.TotalDelivered++
				correct := pkg.Color == st.Color
				if correct {
					ss.DeliveryStats.CorrectDelivered++
					st.FlashOK = true
				} else {
					ss.DeliveryStats.WrongDelivered++
					st.FlashOK = false
				}
				st.FlashTimer = 30
				st.DeliverCount++
				ss.DeliveryStats.ColorDelivered[pkg.Color]++
				// Emit score popup
				scoreText := "+5"
				scoreColor := [3]uint8{255, 80, 80}
				if correct {
					scoreText = "+10"
					scoreColor = [3]uint8{80, 255, 80}
				}
				ss.ScorePopups = append(ss.ScorePopups, swarm.ScorePopup{
					X: st.X, Y: st.Y - 30,
					Text: scoreText, Timer: 60, Color: scoreColor,
				})
				// Emit delivery event for particles
				ss.DeliveryEvents = append(ss.DeliveryEvents, swarm.SwarmDeliveryEvent{
					X: st.X, Y: st.Y,
					Color: pkg.Color, Correct: correct,
				})
				deliveryTime := ss.Tick - pkg.PickupTick
				if deliveryTime > 0 {
					ss.DeliveryStats.DeliveryTimes = append(ss.DeliveryStats.DeliveryTimes, deliveryTime)
				}
				bot.Stats.TotalDeliveries++
				if correct {
					bot.Stats.CorrectDeliveries++
				} else {
					bot.Stats.WrongDeliveries++
				}
				if deliveryTime > 0 {
					bot.Stats.DeliveryTimes = append(bot.Stats.DeliveryTimes, deliveryTime)
				}
				if correct {
					logger.InfoBot(botIdx, "DELIVERY", "Bot #%d delivered %s CORRECT (%d ticks)", botIdx, swarm.DeliveryColorName(pkg.Color), deliveryTime)
				} else {
					logger.WarnBot(botIdx, "DELIVERY", "Bot #%d delivered %s WRONG (%d ticks)", botIdx, swarm.DeliveryColorName(pkg.Color), deliveryTime)
				}
				// Deactivate package, schedule respawn at its pickup station
				pkg.Active = false
				pkg.CarriedBy = -1
				bot.CarryingPkg = -1
				// Find matching pickup and schedule respawn
				for psi := range ss.Stations {
					pst := &ss.Stations[psi]
					if pst.IsPickup && pst.Color == pkg.Color {
						pst.RespawnIn = 100
						break
					}
				}
				delivered = true
				break
			}
		}
		if !delivered {
			// Drop on ground
			pkg.CarriedBy = -1
			pkg.X = bot.X
			pkg.Y = bot.Y
			pkg.OnGround = true
			bot.CarryingPkg = -1
		}

	case swarmscript.ActTurnToPickup:
		if !ss.DeliveryOn {
			return
		}
		if bot.NearestPickupIdx >= 0 && bot.NearestPickupIdx < len(ss.Stations) {
			st := &ss.Stations[bot.NearestPickupIdx]
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
		}

	case swarmscript.ActTurnToDropoff:
		if !ss.DeliveryOn {
			return
		}
		if bot.NearestDropoffIdx >= 0 && bot.NearestDropoffIdx < len(ss.Stations) {
			st := &ss.Stations[bot.NearestDropoffIdx]
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
		}

	case swarmscript.ActTurnToMatchingDropoff:
		if !ss.DeliveryOn || bot.CarryingPkg < 0 {
			return
		}
		pkg := &ss.Packages[bot.CarryingPkg]
		// Find nearest visible dropoff matching package color (extended range)
		bestDist := 1e9
		bestAngle := bot.Angle
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup || st.Color != pkg.Color {
				continue
			}
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= swarm.SwarmDeliverySensorRange && d < bestDist {
				bestDist = d
				bestAngle = math.Atan2(dy, dx)
			}
		}
		if bestDist < 1e9 {
			bot.Angle = bestAngle
		}

	case swarmscript.ActSendPickup:
		if !ss.DeliveryOn {
			return
		}
		// Encode: 10 + color (11-14)
		color := act.Param1
		if bot.NearestPickupColor > 0 {
			color = bot.NearestPickupColor
		}
		if color >= 1 && color <= 4 {
			bot.PendingMsg = 10 + color
		}

	case swarmscript.ActSendDropoff:
		if !ss.DeliveryOn {
			return
		}
		// Encode: 20 + color (21-24)
		color := act.Param1
		if bot.NearestDropoffColor > 0 {
			color = bot.NearestDropoffColor
		}
		if color >= 1 && color <= 4 {
			bot.PendingMsg = 20 + color
		}

	case swarmscript.ActTurnToHeardPickup:
		if !ss.DeliveryOn || bot.HeardPickupColor == 0 {
			return
		}
		bot.Angle = bot.HeardPickupAngle

	case swarmscript.ActTurnToHeardDropoff:
		if !ss.DeliveryOn || bot.HeardDropoffColor == 0 {
			return
		}
		bot.Angle = bot.HeardDropoffAngle

	case swarmscript.ActTurnToMatchingLED:
		// Turn toward the nearest bot whose LED matches carrying color
		if !ss.DeliveryOn || bot.NearestMatchLEDDist >= 999 {
			return
		}
		bot.Angle = bot.NearestMatchLEDAngle

	case swarmscript.ActSetLEDPickupColor:
		// Set LED to the delivery color of nearest visible pickup station
		if !ss.DeliveryOn || bot.NearestPickupIdx < 0 || bot.NearestPickupIdx >= len(ss.Stations) {
			return
		}
		r, g, b := deliveryColorRGB(ss.Stations[bot.NearestPickupIdx].Color)
		bot.LEDColor = [3]uint8{r, g, b}

	case swarmscript.ActSetLEDDropoffColor:
		// Set LED to the delivery color of nearest visible dropoff station
		if !ss.DeliveryOn || bot.NearestDropoffIdx < 0 || bot.NearestDropoffIdx >= len(ss.Stations) {
			return
		}
		r, g, b := deliveryColorRGB(ss.Stations[bot.NearestDropoffIdx].Color)
		bot.LEDColor = [3]uint8{r, g, b}
	}
}

// applyFollowBehavior steers followers toward their leader.
func applyFollowBehavior(ss *swarm.SwarmState, i int) {
	bot := &ss.Bots[i]
	if bot.FollowTargetIdx < 0 || bot.FollowTargetIdx >= len(ss.Bots) {
		return
	}

	target := &ss.Bots[bot.FollowTargetIdx]

	// Validate link integrity
	if target.FollowerIdx != i {
		bot.FollowTargetIdx = -1
		return
	}

	// Measure direct distance to leader
	dx, dy := swarm.NeighborDelta(bot.X, bot.Y, target.X, target.Y, ss)
	leaderDist := math.Sqrt(dx*dx + dy*dy)

	// Break link if leader is too far away (lost contact)
	if leaderDist > swarm.SwarmSensorRange*1.5 {
		target.FollowerIdx = -1
		bot.FollowTargetIdx = -1
		return
	}

	// Steer directly toward leader and maintain ~20px distance
	desiredDist := 20.0
	if leaderDist < desiredDist-5 {
		// Too close — stop and wait
		bot.Speed = 0
	} else if leaderDist < desiredDist+5 {
		// In sweet spot — match leader speed and heading
		bot.Angle = math.Atan2(dy, dx)
		bot.Speed = target.Speed
		if bot.Speed < swarm.SwarmBotSpeed*0.3 {
			bot.Speed = swarm.SwarmBotSpeed * 0.3
		}
	} else {
		// Too far — chase leader directly at full speed
		bot.Angle = math.Atan2(dy, dx)
		bot.Speed = swarm.SwarmBotSpeed
	}
}

// applySwarmPhysics moves a bot and handles boundary collisions + separation.
func applySwarmPhysics(ss *swarm.SwarmState, i int) {
	bot := &ss.Bots[i]

	// Move
	if bot.Speed > 0 {
		bot.X += math.Cos(bot.Angle) * bot.Speed
		bot.Y += math.Sin(bot.Angle) * bot.Speed
	}

	// Obstacle collision — resolve overlap and wall-slide redirection
	allObs := ss.AllObstacles()
	for _, obs := range allObs {
		hit, _, _ := physics.CircleRectCollision(bot.X, bot.Y, swarm.SwarmBotRadius, obs.X, obs.Y, obs.W, obs.H)
		if hit {
			ss.CollisionCount++
			newX, newY := physics.ResolveCircleRectOverlap(bot.X, bot.Y, swarm.SwarmBotRadius, obs.X, obs.Y, obs.W, obs.H)
			pushDx := newX - bot.X
			pushDy := newY - bot.Y
			pushLen := math.Sqrt(pushDx*pushDx + pushDy*pushDy)
			if pushLen > 0.1 {
				// Compute wall normal (direction bot was pushed out)
				nx := pushDx / pushLen
				ny := pushDy / pushLen
				// Check if heading into wall
				hx := math.Cos(bot.Angle)
				hy := math.Sin(bot.Angle)
				dot := hx*nx + hy*ny
				if dot < 0 {
					// Wall-slide: remove normal component from heading so bot slides along wall
					hx -= dot * nx
					hy -= dot * ny
					slideLen := math.Sqrt(hx*hx + hy*hy)
					if slideLen > 0.01 {
						bot.Angle = math.Atan2(hy, hx)
					} else {
						// Head-on collision: random tangent direction
						bot.Angle += math.Pi/2 + ss.Rng.Float64()*math.Pi
					}
				}
			}
			bot.X = newX
			bot.Y = newY
		}
	}

	// Boundary handling
	r := swarm.SwarmBotRadius
	if ss.WrapMode {
		// Toroidal wrap
		if bot.X < 0 {
			bot.X += ss.ArenaW
		}
		if bot.X > ss.ArenaW {
			bot.X -= ss.ArenaW
		}
		if bot.Y < 0 {
			bot.Y += ss.ArenaH
		}
		if bot.Y > ss.ArenaH {
			bot.Y -= ss.ArenaH
		}
	} else {
		// Bounce — clamp position, then redirect angle (but NOT for followers)
		hitEdge := false
		if bot.X < r {
			bot.X = r
			hitEdge = true
		}
		if bot.X > ss.ArenaW-r {
			bot.X = ss.ArenaW - r
			hitEdge = true
		}
		if bot.Y < r {
			bot.Y = r
			hitEdge = true
		}
		if bot.Y > ss.ArenaH-r {
			bot.Y = ss.ArenaH - r
			hitEdge = true
		}
		// Only redirect angle for FREE bots (not following anyone).
		// Followers get their angle from applyFollowBehavior — overwriting
		// it here would steer them away from their leader and break chains!
		if hitEdge && bot.FollowTargetIdx < 0 {
			// Wall-reflection: bounce like a billiard ball to spread bots evenly.
			// (Old center-aim code made ALL bots converge to the middle.)
			if bot.X <= r || bot.X >= ss.ArenaW-r {
				bot.Angle = math.Pi - bot.Angle // flip horizontal component
			}
			if bot.Y <= r || bot.Y >= ss.ArenaH-r {
				bot.Angle = -bot.Angle // flip vertical component
			}
			// Random perturbation to avoid deterministic ping-pong paths
			bot.Angle += (ss.Rng.Float64() - 0.5) * math.Pi / 3 // ±30°
		}
	}

	// Normalize angle to [0, 2π)
	for bot.Angle < 0 {
		bot.Angle += 2 * math.Pi
	}
	for bot.Angle >= 2*math.Pi {
		bot.Angle -= 2 * math.Pi
	}
}

// applyHardSeparation is a symmetric rigid-body pass that guarantees no two
// bots overlap. Both bots in a pair are pushed apart to exactly minDist.
// Runs twice per tick to resolve multi-bot clusters (triangles etc.).
func applyHardSeparation(ss *swarm.SwarmState) {
	const minDist = swarm.SwarmBotRadius * 2.4 // 24px hard shell

	for iter := 0; iter < 2; iter++ {
		// Rebuild spatial hash each iteration (positions shifted)
		ss.Hash.Clear()
		for i := range ss.Bots {
			ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
		}

		for i := range ss.Bots {
			a := &ss.Bots[i]
			nearIDs := ss.Hash.Query(a.X, a.Y, minDist+1)
			for _, j := range nearIDs {
				if j <= i || j >= len(ss.Bots) {
					continue // each pair once (j > i)
				}
				// Skip directly linked follower↔leader pairs
				if a.FollowTargetIdx == j || a.FollowerIdx == j {
					continue
				}
				b := &ss.Bots[j]
				dx := a.X - b.X
				dy := a.Y - b.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist >= minDist {
					continue
				}
				if dist < 0.001 {
					// Coincident — nudge with random direction
					angle := ss.Rng.Float64() * 2 * math.Pi
					dx = math.Cos(angle)
					dy = math.Sin(angle)
					dist = 0.001
				}
				// Push BOTH apart to full minDist (each gets half)
				nx := dx / dist
				ny := dy / dist
				half := (minDist - dist) * 0.5
				a.X += nx * half
				a.Y += ny * half
				b.X -= nx * half
				b.Y -= ny * half

				// Elastic heading deflection when very close
				if dist < 12 {
					a.Angle = math.Atan2(ny, nx) + (ss.Rng.Float64()-0.5)*0.3
					b.Angle = math.Atan2(-ny, -nx) + (ss.Rng.Float64()-0.5)*0.3
				}
			}
		}
	}
}

// applyRepulsionForce adds a continuous push between bots closer than 30px.
// Unlike hard separation (which resolves overlap), this creates an active
// force field that prevents clustering before contact.
func applyRepulsionForce(ss *swarm.SwarmState) {
	const repulsionRange = 30.0
	const repulsionStrength = 0.15

	for i := range ss.Bots {
		a := &ss.Bots[i]
		nearIDs := ss.Hash.Query(a.X, a.Y, repulsionRange+1)
		for _, j := range nearIDs {
			if j <= i || j >= len(ss.Bots) {
				continue
			}
			// Skip linked pairs
			if a.FollowTargetIdx == j || a.FollowerIdx == j {
				continue
			}
			b := &ss.Bots[j]
			dx := a.X - b.X
			dy := a.Y - b.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist >= repulsionRange || dist < 0.001 {
				continue
			}
			// Force = (30 - dist) * 0.15, applied symmetrically
			force := (repulsionRange - dist) * repulsionStrength
			nx := dx / dist
			ny := dy / dist
			halfForce := force * 0.5
			a.X += nx * halfForce
			a.Y += ny * halfForce
			b.X -= nx * halfForce
			b.Y -= ny * halfForce
		}
	}
}
