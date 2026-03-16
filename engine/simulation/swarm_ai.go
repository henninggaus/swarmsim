package simulation

import (
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

	// Dropoff beacons: stations emit virtual SEND_DROPOFF messages every 30 ticks
	if ss.DeliveryOn && ss.Tick%30 == 0 {
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup {
				continue
			}
			ss.PrevMessages = append(ss.PrevMessages, swarm.SwarmMessage{
				Value: 20 + st.Color,
				X:     st.X,
				Y:     st.Y,
			})
		}
	}

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

	// Phase 1.5: Rebuild ramp semaphore count
	if ss.TruckToggle {
		ss.RampBotCount = 0
		for i := range ss.Bots {
			if ss.Bots[i].OnRamp && ss.Bots[i].CarryingPkg < 0 {
				ss.RampBotCount++
			}
		}
	}

	// Phase 2: Execute program on each bot (skip if anti-stuck breakout active)
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
		// Neuroevolution: use neural network if NEURO is ON
		if ss.NeuroEnabled && bot.Brain != nil {
			inputs := swarm.BuildNeuroInputs(bot, ss)
			actionIdx := swarm.NeuroForward(bot.Brain, inputs)
			swarm.ExecuteNeuroAction(actionIdx, bot, ss, i)
			// Auto-pickup/drop for neuro bots: the net navigates, the system handles interaction
			if ss.DeliveryOn {
				if bot.CarryingPkg < 0 && bot.NearestPickupDist < 20 && bot.NearestPickupHasPkg {
					executeSwarmAction(swarmscript.Action{Type: swarmscript.ActPickup}, bot, ss, i)
				}
				if bot.CarryingPkg >= 0 && bot.DropoffMatch && bot.NearestDropoffDist < 30 {
					executeSwarmAction(swarmscript.Action{Type: swarmscript.ActDrop}, bot, ss, i)
				}
			}
			continue
		}
		if ss.Program != nil {
			executeSwarmProgram(ss, i)
		}
	}

	// Phase 2.3: Anti-clustering — scatter, idle exploration, dropoff repulsion
	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Decrement scatter/idle timers
		if bot.ScatterCooldown > 0 {
			bot.ScatterCooldown--
		}

		if bot.AntiStuckTimer > 0 {
			continue // already in breakout
		}

		// (A) Scatter: >3 close neighbors → forced TURN_FROM_NEAREST + FWD for 15 ticks
		if bot.ScatterTimer > 0 {
			if bot.NearestIdx >= 0 {
				other := &ss.Bots[bot.NearestIdx]
				dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
				bot.Angle = math.Atan2(-dy, -dx) + (ss.Rng.Float64()-0.5)*0.4
			} else {
				bot.Angle += (ss.Rng.Float64() - 0.5) * 1.0
			}
			bot.Speed = swarm.SwarmBotSpeed
			bot.ScatterTimer--
			continue
		}
		if bot.CloseNeighbors > 3 && bot.ScatterCooldown == 0 {
			bot.ScatterTimer = 15
			bot.ScatterCooldown = 30
			if bot.NearestIdx >= 0 {
				other := &ss.Bots[bot.NearestIdx]
				dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
				bot.Angle = math.Atan2(-dy, -dx) + (ss.Rng.Float64()-0.5)*0.4
			}
			bot.Speed = swarm.SwarmBotSpeed
			continue
		}

		// (B) Idle exploration: carry==0 bots that stay still for 60 ticks → forced random FWD 30 ticks
		if bot.IdleMoveTimer > 0 {
			// Active exploration override
			if ss.Rng.Float64() < 0.1 {
				bot.Angle += (ss.Rng.Float64() - 0.5) * 1.5
			}
			bot.Speed = swarm.SwarmBotSpeed
			bot.IdleMoveTimer--
			if bot.IdleMoveTimer == 0 {
				bot.IdleMoveTicks = 0
			}
			continue
		}
		if bot.CarryingPkg < 0 && bot.Speed < 0.5 {
			bot.IdleMoveTicks++
			if bot.IdleMoveTicks >= 60 {
				bot.IdleMoveTimer = 30
				bot.Angle = ss.Rng.Float64() * 2 * math.Pi
				bot.Speed = swarm.SwarmBotSpeed
				bot.IdleMoveTicks = 0
			}
		} else {
			bot.IdleMoveTicks = 0
		}
	}

	// Phase 2.4: Cluster breaker — detect large clusters and explode them (every 10 ticks)
	if ss.Tick%10 == 0 {
		applyClusterBreaker(ss)
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
			// Add visual wave ring
			if ss.ShowMsgWaves {
				ss.MsgWaves = append(ss.MsgWaves, swarm.MsgWave{
					X: ss.Bots[i].X, Y: ss.Bots[i].Y,
					Radius: 5, Timer: 30,
					Value: ss.Bots[i].PendingMsg,
				})
			}
		}
	}

	// Update wave rings (expand and fade)
	if ss.ShowMsgWaves {
		alive := 0
		for i := range ss.MsgWaves {
			ss.MsgWaves[i].Radius += 3
			ss.MsgWaves[i].Timer--
			if ss.MsgWaves[i].Timer > 0 {
				ss.MsgWaves[alive] = ss.MsgWaves[i]
				alive++
			}
		}
		ss.MsgWaves = ss.MsgWaves[:alive]
	}

	// Phase 4: Physics — move bots, clamp to bounds
	for i := range ss.Bots {
		applySwarmPhysics(ss, i)
	}

	// Phase 4.1: Hard separation — symmetric rigid-body push for all pairs
	applyHardSeparation(ss)

	// Phase 4.2: Repulsion force — active push when bots are closer than 30px
	applyRepulsionForce(ss)

	// Phase 4.3: Station repulsion — push non-carrying bots away from dropoffs
	applyStationRepulsion(ss)

	// Phase 4.5: Delivery system updates (pickup/drop, respawn, carried package position)
	if ss.DeliveryOn {
		swarm.UpdateDeliverySystem(ss)
		// Periodic delivery stats log
		if ss.Tick%600 == 0 && ss.DeliveryStats.TotalDelivered > 0 {
			carrying := 0
			for ci := range ss.Bots {
				if ss.Bots[ci].CarryingPkg >= 0 {
					carrying++
				}
			}
			correctRate := 0
			if ss.DeliveryStats.TotalDelivered > 0 {
				correctRate = ss.DeliveryStats.CorrectDelivered * 100 / ss.DeliveryStats.TotalDelivered
			}
			logger.Info("DELIVERY", "Tick %d: %d geliefert (%d%% korrekt), %d Bots tragen Pakete",
				ss.Tick, ss.DeliveryStats.TotalDelivered, correctRate, carrying)
		}
	}

	// Phase 4.52: Dynamic environment — moving obstacles + expiring packages
	if ss.DynamicEnv {
		swarm.UpdateDynamicEnvironment(ss)
	}

	// Phase 4.55: Pheromone trails — carrying bots deposit, grid decays
	if ss.PherGrid != nil {
		for i := range ss.Bots {
			if ss.Bots[i].CarryingPkg >= 0 {
				ss.PherGrid.Deposit(ss.Bots[i].X, ss.Bots[i].Y, 0.3)
			}
		}
		ss.PherGrid.Update()
	}

	// Phase 4.6: Truck system updates
	if ss.TruckToggle && ss.TruckState != nil {
		swarm.UpdateSwarmTruck(ss)

		// Debug logging: truck sensor values every 200 ticks
		if ss.Tick%200 == 0 {
			onRampCount := 0
			truckHereCount := 0
			carryingCount := 0
			for bi := range ss.Bots {
				if ss.Bots[bi].OnRamp {
					onRampCount++
				}
				if ss.Bots[bi].TruckHere {
					truckHereCount++
				}
				if ss.Bots[bi].CarryingPkg >= 0 {
					carryingCount++
				}
			}
			phase := "nil"
			if ss.TruckState.CurrentTruck != nil {
				switch ss.TruckState.CurrentTruck.Phase {
				case swarm.TruckDrivingIn:
					phase = "DrivingIn"
				case swarm.TruckParked:
					phase = "Parked"
				case swarm.TruckComplete:
					phase = "Complete"
				case swarm.TruckDrivingOut:
					phase = "DrivingOut"
				case swarm.TruckWaiting:
					phase = "Waiting"
				case swarm.TruckRoundDone:
					phase = "RoundDone"
				}
			}
			logger.Info("TRUCK", "tick=%d truck=%s ramp=%dx%d on_ramp=%d truck_here=%d carrying=%d delivered=%d score=%d",
				ss.Tick, phase,
				int(ss.TruckState.RampW), int(ss.TruckState.RampH),
				onRampCount, truckHereCount, carryingCount,
				ss.TruckState.DeliveredPkgs, ss.TruckState.Score)

			// Log first 3 bots details
			for bi := 0; bi < 3 && bi < len(ss.Bots); bi++ {
				b := &ss.Bots[bi]
				logger.Info("TRUCK", "  bot#%d pos=(%.0f,%.0f) onRamp=%v truckHere=%v pkgCount=%d nearPkgDist=%.0f carry=%d",
					bi, b.X, b.Y, b.OnRamp, b.TruckHere, b.TruckPkgCount, b.NearestTruckPkgDist, b.CarryingPkg)
			}
		}
	}

	// Phase 4.8: Evolution system
	if ss.EvolutionOn {
		ss.EvolutionTimer++
		if ss.EvolutionTimer >= 1500 {
			swarm.RunEvolution(ss)
			ss.EvolutionSoundPending = true
			dm := swarm.MeasureParamDiversity(ss)
			ss.Diversity = &dm
		}
	}

	// Phase 4.85: Genetic Programming evolution
	if ss.GPEnabled {
		ss.GPTimer++
		if ss.GPTimer >= 2000 {
			swarm.RunGPEvolution(ss)
			ss.GPTimer = 0
			ss.EvolutionSoundPending = true
			dm := swarm.MeasureGPDiversity(ss)
			ss.Diversity = &dm
		}
	}

	// Phase 4.87: Neuroevolution
	if ss.NeuroEnabled {
		ss.NeuroTimer++
		if ss.NeuroTimer >= 2000 {
			swarm.RunNeuroEvolution(ss)
			ss.EvolutionSoundPending = true
			dm := swarm.MeasureNeuroDiversity(ss)
			ss.Diversity = &dm
		}
	}

	// Count broadcasts for sound
	ss.BroadcastCount = len(ss.NextMessages)

	// Phase 4.88: Tournament timer
	if ss.TournamentOn && ss.TournamentPhase == 1 {
		swarm.TournamentTick(ss)
	}

	// Phase 4.9: Challenge timer update
	if ss.TeamsEnabled {
		swarm.UpdateChallenge(ss)
	}

	// Phase 4.92: Statistics tracker update
	if ss.StatsTracker != nil {
		ss.StatsTracker.Update(ss)
		// Heatmap update every 100 ticks
		if ss.Tick%100 == 0 {
			ss.StatsTracker.UpdateHeatmap(ss)
		}
		// Rankings update every 500 ticks
		if ss.Tick%500 == 0 {
			ss.StatsTracker.UpdateRankings(ss)
		}
	}

	// Phase 4.92a: Scenario chain tick
	if ss.ScenarioChain != nil && ss.ScenarioChain.Active {
		swarm.ScenarioChainTick(ss)
	}

	// Phase 4.92b: Auto-Optimizer tick
	if ss.AutoOptimizer != nil && ss.AutoOptimizer.Active {
		swarm.AutoOptimizerTick(ss)
	}

	// Phase 4.93: Heatmap accumulation (every 5 ticks for performance)
	if ss.ShowHeatmap && ss.Tick%5 == 0 {
		swarm.UpdateHeatmap(ss)
	}

	// Phase 4.94: Bot spatial memory update (every 5 ticks)
	if ss.MemoryEnabled && ss.Tick%5 == 0 {
		swarm.UpdateBotMemory(ss)
	}

	// Phase 4.95: Accumulate lifetime stats (before StuckPrevX/Y is overwritten)
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

	// Phase 4.97: Energy system
	if ss.EnergyEnabled {
		for i := range ss.Bots {
			bot := &ss.Bots[i]
			// Drain energy when moving
			if bot.Speed > 0 {
				bot.Energy -= 0.05 // ~2000 ticks of movement on full charge
				if bot.Energy < 0 {
					bot.Energy = 0
				}
			}
			// Zero energy: force stop
			if bot.Energy <= 0 {
				bot.Speed = 0
			}
			// Recharge near stations (within 40px of any pickup or dropoff)
			if ss.DeliveryOn {
				for si := range ss.Stations {
					st := &ss.Stations[si]
					dx := bot.X - st.X
					dy := bot.Y - st.Y
					if dx*dx+dy*dy < 1600 { // 40px radius
						bot.Energy += 0.5 // recharges ~200x faster than drain
						if bot.Energy > 100 {
							bot.Energy = 100
						}
						break
					}
				}
			} else {
				// Without delivery: slow passive recharge
				bot.Energy += 0.02
				if bot.Energy > 100 {
					bot.Energy = 100
				}
			}
		}
	}

	// Phase 5: Anti-stuck detection & cooldown
	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Decrement stuck cooldown
		if bot.StuckCooldown > 0 {
			bot.StuckCooldown--
		}
		// Decrement ramp cooldown
		if bot.RampCooldown > 0 {
			bot.RampCooldown--
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

		// Anti-stuck at ramp: non-carrying bots stuck near ramp get pushed into arena
		if bot.StuckTicks >= 45 && ss.TruckToggle && bot.OnRamp && bot.CarryingPkg < 0 {
			bot.Angle = math.Pi / 4 // push rightward into arena
			bot.Speed = swarm.SwarmBotSpeed
			bot.StuckTicks = 0
			bot.StuckCooldown = 30
			logger.WarnBot(i, "RAMP", "Bot #%d anti-stuck at ramp — pushed into arena", i)
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
	bot.BotAhead = 0
	bot.BotBehind = 0
	bot.BotLeft = 0
	bot.BotRight = 0

	// Query neighbors within sensor range (carrying bots get extended 200px range)
	sensorRange := swarm.SwarmSensorRange
	if bot.CarryingPkg >= 0 {
		sensorRange = swarm.SwarmDeliverySensorRange
	}
	candidateIDs := ss.Hash.Query(bot.X, bot.Y, sensorRange)
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
		if dist > sensorRange {
			continue
		}
		count++
		closeRange := 30.0
		if bot.OnRamp {
			closeRange = 40.0 // wider detection on ramp to prevent clustering
		}
		if dist < closeRange {
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

		// Directional classification (90° cones relative to heading)
		angleToOther := math.Atan2(dy, dx)
		relAngle := angleToOther - bot.Angle
		// Normalize to [-π, π]
		for relAngle > math.Pi {
			relAngle -= 2 * math.Pi
		}
		for relAngle < -math.Pi {
			relAngle += 2 * math.Pi
		}
		if relAngle >= -math.Pi/4 && relAngle < math.Pi/4 {
			bot.BotAhead++
		} else if relAngle >= math.Pi/4 && relAngle < 3*math.Pi/4 {
			bot.BotRight++
		} else if relAngle >= -3*math.Pi/4 && relAngle < -math.Pi/4 {
			bot.BotLeft++
		} else {
			bot.BotBehind++
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

	// On edge check (exempt bots in the ramp zone from left-edge detection)
	onLeftEdge := bot.X < swarm.SwarmEdgeMargin
	if onLeftEdge && ss.TruckToggle && ss.TruckState != nil &&
		bot.Y >= swarm.SwarmRampY && bot.Y <= swarm.SwarmRampY+swarm.SwarmRampH {
		onLeftEdge = false // ramp zone: not considered "edge"
	}
	if onLeftEdge || bot.X > ss.ArenaW-swarm.SwarmEdgeMargin ||
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
		// Carrying bots get beacon range for dropoff detection
		delivRange := swarm.SwarmDeliverySensorRange
		dropoffRange := swarm.SwarmDeliverySensorRange
		if bot.CarryingPkg >= 0 {
			dropoffRange = swarm.SwarmBeaconRange
		}

		for si := range ss.Stations {
			st := &ss.Stations[si]
			sdx, sdy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			sdist := math.Sqrt(sdx*sdx + sdy*sdy)
			if st.IsPickup {
				if sdist > delivRange {
					continue
				}
				if sdist < bot.NearestPickupDist {
					bot.NearestPickupDist = sdist
					bot.NearestPickupColor = st.Color
					bot.NearestPickupHasPkg = st.HasPackage
					bot.NearestPickupIdx = si
				}
			} else {
				if sdist > dropoffRange {
					continue
				}
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
			if msg.Value >= 11 && msg.Value <= 14 && mdist <= swarm.SwarmCommRange && bot.HeardPickupColor == 0 {
				bot.HeardPickupColor = msg.Value - 10
				bot.HeardPickupAngle = math.Atan2(mdy, mdx)
			}
			// Dropoff messages use extended beacon range (150px)
			if msg.Value >= 21 && msg.Value <= 24 && mdist <= swarm.SwarmDropoffBeaconRange && bot.HeardDropoffColor == 0 {
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

	// Beacon sensors: dropoff stations broadcast to carrying bots within BeaconRange
	bot.HeardBeaconDropoffColor = 0
	bot.HeardBeaconDropoffDist = 9999
	bot.HeardBeaconDropoffAngle = 0
	if ss.DeliveryOn && bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
		pkg := &ss.Packages[bot.CarryingPkg]
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup {
				continue
			}
			if st.Color != pkg.Color {
				continue // only matching color beacons
			}
			bdx, bdy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			bdist := math.Sqrt(bdx*bdx + bdy*bdy)
			if bdist < swarm.SwarmBeaconRange && bdist < bot.HeardBeaconDropoffDist {
				bot.HeardBeaconDropoffColor = st.Color
				bot.HeardBeaconDropoffDist = bdist
				bot.HeardBeaconDropoffAngle = math.Atan2(bdy, bdx)
			}
		}
	}

	// Exploration timer: increment when carrying but no dropoff/beacon visible
	if bot.CarryingPkg >= 0 {
		if bot.DropoffMatch || bot.HeardBeaconDropoffColor > 0 {
			bot.ExplorationTimer = 0
			bot.ExplorationAngle = 0
		} else {
			bot.ExplorationTimer++
		}
	} else {
		bot.ExplorationTimer = 0
		bot.ExplorationAngle = 0
	}

	// Truck sensors: build when truck toggle active
	bot.TruckHere = false
	bot.TruckPkgCount = 0
	bot.OnRamp = false
	bot.NearestTruckPkgDist = 999
	bot.NearestTruckPkgIdx = -1

	if ss.TruckToggle && ss.TruckState != nil {
		ts := ss.TruckState
		// OnRamp = bot is near the right edge of the ramp (ready for crane pickup)
		// Detection zone: rampEdge ±50px in X, full ramp height in Y
		rampEdgeX := ts.RampX + ts.RampW // right edge (200)
		if bot.Y >= ts.RampY && bot.Y <= ts.RampY+ts.RampH &&
			bot.X >= rampEdgeX-50 && bot.X <= rampEdgeX+50 {
			if !bot.OnRamp && ss.Tick%60 == 0 {
				logger.InfoBot(i, "RAMP", "Bot #%d entered ramp zone (%.0f, %.0f) bounds X:[%.0f-%.0f] Y:[%.0f-%.0f]",
					i, bot.X, bot.Y, rampEdgeX-50, rampEdgeX+50, ts.RampY, ts.RampY+ts.RampH)
			}
			bot.OnRamp = true
		}
		if ts.CurrentTruck != nil {
			// Count remaining packages (available in all phases so bots head to ramp early)
			for pi, pkg := range ts.CurrentTruck.Packages {
				if pkg.PickedUp {
					continue
				}
				bot.TruckPkgCount++
				// Package world position (only meaningful when truck visible)
				if ts.CurrentTruck.Phase == swarm.TruckParked ||
					ts.CurrentTruck.Phase == swarm.TruckDrivingIn {
					wpx := ts.CurrentTruck.X + pkg.RelX + 18 + 4 // cabin offset + center
					wpy := ts.CurrentTruck.Y + pkg.RelY + 4      // center
					pdx := bot.X - wpx
					pdy := bot.Y - wpy
					pdist := math.Sqrt(pdx*pdx + pdy*pdy)
					if pdist < bot.NearestTruckPkgDist {
						bot.NearestTruckPkgDist = pdist
						bot.NearestTruckPkgIdx = pi
					}
				}
			}
			// TruckHere only when parked (packages can be picked up)
			if ts.CurrentTruck.Phase == swarm.TruckParked {
				bot.TruckHere = true
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

	// Wall sensors: raycast 25px to the right and left of heading
	bot.WallRight = false
	bot.WallLeft = false
	if len(allObs) > 0 {
		rightAngle := bot.Angle + math.Pi/2
		leftAngle := bot.Angle - math.Pi/2
		for step := 1; step <= 5; step++ {
			d := float64(step) * 5.0
			if !bot.WallRight {
				px := bot.X + math.Cos(rightAngle)*d
				py := bot.Y + math.Sin(rightAngle)*d
				for _, obs := range allObs {
					if pointInRect(px, py, obs) {
						bot.WallRight = true
						break
					}
				}
			}
			if !bot.WallLeft {
				px := bot.X + math.Cos(leftAngle)*d
				py := bot.Y + math.Sin(leftAngle)*d
				for _, obs := range allObs {
					if pointInRect(px, py, obs) {
						bot.WallLeft = true
						break
					}
				}
			}
			if bot.WallRight && bot.WallLeft {
				break
			}
		}
	}

	// Pheromone sensor: sample 20px ahead
	bot.PherAhead = 0
	if ss.PherGrid != nil {
		px := bot.X + math.Cos(bot.Angle)*20
		py := bot.Y + math.Sin(bot.Angle)*20
		bot.PherAhead = ss.PherGrid.Get(px, py)
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

	// Genetic Programming: use bot's own program if GP is ON
	if ss.GPEnabled && bot.OwnProgram != nil {
		prog = bot.OwnProgram
	}

	// Teams: use team-specific program if teams enabled
	if ss.TeamsEnabled {
		if bot.Team == 1 && ss.TeamAProgram != nil {
			prog = ss.TeamAProgram
		} else if bot.Team == 2 && ss.TeamBProgram != nil {
			prog = ss.TeamBProgram
		}
	}

	if prog == nil {
		return
	}

	// Snapshot mutable vars for condition evaluation
	snapState := bot.State
	snapCounter := bot.Counter
	snapTimer := bot.Timer

	// Reset per-tick outputs
	bot.Speed = 0
	bot.PendingMsg = 0

	// Track matched rules for GP visualization
	if ss.GPEnabled {
		bot.LastMatchedRules = bot.LastMatchedRules[:0]
	}

	for ri, rule := range prog.Rules {
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
			if ss.GPEnabled {
				bot.LastMatchedRules = append(bot.LastMatchedRules, ri)
			}
		}
	}
}

// evaluateSwarmCondition checks a single condition against bot sensors.
func evaluateSwarmCondition(cond swarmscript.Condition, bot *swarm.SwarmBot, snapState, snapCounter, snapTimer int, rng *rand.Rand, ss *swarm.SwarmState, botIdx int) bool {
	cv := resolveCondValue(cond, bot, ss)
	switch cond.Type {
	case swarmscript.CondTrue:
		return true

	case swarmscript.CondNeighborsCount:
		return compareInt(bot.NeighborCount, cond.Op, cv)

	case swarmscript.CondNearestDistance:
		return compareInt(int(bot.NearestDist), cond.Op, cv)

	case swarmscript.CondState, swarmscript.CondMyState:
		return compareInt(snapState, cond.Op, cv)

	case swarmscript.CondCounter:
		return compareInt(snapCounter, cond.Op, cv)

	case swarmscript.CondTimer:
		return compareInt(snapTimer, cond.Op, cv)

	case swarmscript.CondOnEdge:
		onEdgeVal := 0
		if bot.OnEdge {
			onEdgeVal = 1
		}
		return compareInt(onEdgeVal, cond.Op, cv)

	case swarmscript.CondReceivedMessage:
		return compareInt(bot.ReceivedMsg, cond.Op, cv)

	case swarmscript.CondLightValue:
		return compareInt(bot.LightValue, cond.Op, cv)

	case swarmscript.CondRandom:
		// random < N means N% chance
		return rng.Intn(100) < cv

	case swarmscript.CondHasLeader:
		hasLeader := 0
		if bot.FollowTargetIdx >= 0 {
			hasLeader = 1
		}
		return compareInt(hasLeader, cond.Op, cv)

	case swarmscript.CondHasFollower:
		hasFollower := 0
		if bot.FollowerIdx >= 0 {
			hasFollower = 1
		}
		return compareInt(hasFollower, cond.Op, cv)

	case swarmscript.CondChainLength:
		cl := computeChainLength(ss, botIdx)
		return compareInt(cl, cond.Op, cv)

	case swarmscript.CondNearestLEDR:
		return compareInt(int(bot.NearestLEDR), cond.Op, cv)

	case swarmscript.CondNearestLEDG:
		return compareInt(int(bot.NearestLEDG), cond.Op, cv)

	case swarmscript.CondNearestLEDB:
		return compareInt(int(bot.NearestLEDB), cond.Op, cv)

	case swarmscript.CondTick:
		return compareInt(ss.Tick, cond.Op, cv)

	case swarmscript.CondObstacleAhead:
		obsVal := 0
		if bot.ObstacleAhead {
			obsVal = 1
		}
		return compareInt(obsVal, cond.Op, cv)

	case swarmscript.CondObstacleDist:
		return compareInt(int(bot.ObstacleDist), cond.Op, cv)

	case swarmscript.CondValue1:
		return compareInt(bot.Value1, cond.Op, cv)

	case swarmscript.CondValue2:
		return compareInt(bot.Value2, cond.Op, cv)

	case swarmscript.CondCarrying:
		carryVal := 0
		if bot.CarryingPkg >= 0 {
			carryVal = 1
		}
		return compareInt(carryVal, cond.Op, cv)

	case swarmscript.CondCarryingColor:
		cc := 0
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			cc = ss.Packages[bot.CarryingPkg].Color
		}
		return compareInt(cc, cond.Op, cv)

	case swarmscript.CondNearestPickupDist:
		return compareInt(int(bot.NearestPickupDist), cond.Op, cv)

	case swarmscript.CondNearestPickupColor:
		return compareInt(bot.NearestPickupColor, cond.Op, cv)

	case swarmscript.CondNearestPickupHasPkg:
		v := 0
		if bot.NearestPickupHasPkg {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondNearestDropoffDist:
		return compareInt(int(bot.NearestDropoffDist), cond.Op, cv)

	case swarmscript.CondNearestDropoffColor:
		return compareInt(bot.NearestDropoffColor, cond.Op, cv)

	case swarmscript.CondDropoffMatch:
		v := 0
		if bot.DropoffMatch {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondHeardPickupColor:
		return compareInt(bot.HeardPickupColor, cond.Op, cv)

	case swarmscript.CondHeardDropoffColor:
		return compareInt(bot.HeardDropoffColor, cond.Op, cv)

	case swarmscript.CondNearestMatchLEDDist:
		return compareInt(int(bot.NearestMatchLEDDist), cond.Op, cv)

	case swarmscript.CondTruckHere:
		v := 0
		if bot.TruckHere {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondTruckPkgCount:
		return compareInt(bot.TruckPkgCount, cond.Op, cv)

	case swarmscript.CondOnRamp:
		v := 0
		if bot.OnRamp {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondNearestTruckPkgDist:
		return compareInt(int(bot.NearestTruckPkgDist), cond.Op, cv)

	case swarmscript.CondHeardBeaconDropoff:
		v := 0
		if bot.HeardBeaconDropoffColor > 0 {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondHeardBeaconDropoffDist:
		return compareInt(int(bot.HeardBeaconDropoffDist), cond.Op, cv)

	case swarmscript.CondExploring:
		v := 0
		if bot.ExplorationTimer > 60 {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondWallRight:
		v := 0
		if bot.WallRight {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondWallLeft:
		v := 0
		if bot.WallLeft {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondPherAhead:
		return compareInt(int(bot.PherAhead*100), cond.Op, cv)

	case swarmscript.CondTeam:
		return compareInt(bot.Team, cond.Op, cv)

	case swarmscript.CondTeamScore:
		score := 0
		if bot.Team == 1 {
			score = ss.TeamAScore
		} else if bot.Team == 2 {
			score = ss.TeamBScore
		}
		return compareInt(score, cond.Op, cv)

	case swarmscript.CondEnemyScore:
		score := 0
		if bot.Team == 1 {
			score = ss.TeamBScore
		} else if bot.Team == 2 {
			score = ss.TeamAScore
		}
		return compareInt(score, cond.Op, cv)

	case swarmscript.CondBotAhead:
		return compareInt(bot.BotAhead, cond.Op, cv)
	case swarmscript.CondBotBehind:
		return compareInt(bot.BotBehind, cond.Op, cv)
	case swarmscript.CondBotLeft:
		return compareInt(bot.BotLeft, cond.Op, cv)
	case swarmscript.CondBotRight:
		return compareInt(bot.BotRight, cond.Op, cv)
	case swarmscript.CondHeading:
		deg := int(bot.Angle * 180 / math.Pi)
		if deg < 0 {
			deg += 360
		}
		return compareInt(deg, cond.Op, cv)
	case swarmscript.CondSpeed:
		return compareInt(int(bot.Speed*100), cond.Op, cv)

	case swarmscript.CondVisitedHere:
		return compareInt(swarm.BotVisitedHere(bot), cond.Op, cv)
	case swarmscript.CondVisitedAhead:
		return compareInt(swarm.BotVisitedAhead(bot), cond.Op, cv)
	case swarmscript.CondExplored:
		return compareInt(swarm.BotExploredPercent(bot), cond.Op, cv)
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

// resolveCondValue returns the effective comparison value for a condition,
// using per-bot evolved parameters when IsParamRef and EvolutionOn.
func resolveCondValue(cond swarmscript.Condition, bot *swarm.SwarmBot, ss *swarm.SwarmState) int {
	if cond.IsParamRef && ss.EvolutionOn {
		return int(bot.ParamValues[cond.ParamIdx])
	}
	return cond.Value
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
		// Truck packages: crane-based pickup at ramp edge
		// Bot waits at the ramp edge (OnRamp=true), a crane transfers the package
		if ss.TruckToggle && ss.TruckState != nil && bot.OnRamp &&
			ss.TruckState.CurrentTruck != nil && ss.TruckState.CurrentTruck.Phase == swarm.TruckParked {
			t := ss.TruckState.CurrentTruck
			// Find first unpicked package (crane handles transfer — no distance check)
			for tpi := range t.Packages {
				tpkg := &t.Packages[tpi]
				if tpkg.PickedUp {
					continue
				}
				tpkg.PickedUp = true
				ss.TruckState.TotalPkgs++
				// Convert to DeliveryPackage
				dpkg := swarm.DeliveryPackage{
					Color:      tpkg.Color,
					CarriedBy:  botIdx,
					X:          bot.X,
					Y:          bot.Y,
					Active:     true,
					PickupTick: ss.Tick,
				}
				ss.Packages = append(ss.Packages, dpkg)
				bot.CarryingPkg = len(ss.Packages) - 1
				bot.Stats.TotalPickups++
				// Emit pickup event for particles (at ramp edge where bot is)
				ss.DeliveryEvents = append(ss.DeliveryEvents, swarm.SwarmDeliveryEvent{
					X: bot.X, Y: bot.Y,
					Color: tpkg.Color, IsPickup: true,
				})
				// Stats tracker pickup event
				if ss.StatsTracker != nil {
					ss.StatsTracker.AddPickupEvent(botIdx, swarm.DeliveryColorName(tpkg.Color))
					ss.StatsTracker.RecordActionAt(bot.X, bot.Y, ss.ArenaW, ss.ArenaH)
				}
				// Turn away from ramp (toward arena interior) so bot leaves immediately
				bot.Angle = (ss.Rng.Float64() - 0.5) * math.Pi / 2 // roughly rightward ±45°
				bot.Speed = swarm.SwarmBotSpeed
				logger.InfoBot(botIdx, "TRUCK", "Bot #%d crane-pickup %s from truck", botIdx, swarm.DeliveryColorName(tpkg.Color))
				return
			}
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
				// Stats tracker pickup event
				if ss.StatsTracker != nil {
					ss.StatsTracker.AddPickupEvent(botIdx, swarm.DeliveryColorName(pkg.Color))
					ss.StatsTracker.RecordActionAt(bot.X, bot.Y, ss.ArenaW, ss.ArenaH)
				}
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
		// DROP safety: only allow within 30px of a dropoff station
		nearStation := false
		delivered := false
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup {
				continue
			}
			sdx, sdy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			stDist := math.Sqrt(sdx*sdx + sdy*sdy)
			if stDist < 30 {
				nearStation = true
				correct := pkg.Color == st.Color
				if !correct {
					logger.WarnBot(botIdx, "DELIVERY", "Bot #%d dropped %s at wrong %s station",
						botIdx, swarm.DeliveryColorName(pkg.Color), swarm.DeliveryColorName(st.Color))
				}
				// Delivery!
				ss.DeliveryStats.TotalDelivered++
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
					X: st.X + 35, Y: st.Y,
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
				// Stats tracker
				if ss.StatsTracker != nil {
					ss.StatsTracker.RecordDelivery(correct, bot.Team)
					ss.StatsTracker.AddDeliveryEvent(botIdx, swarm.DeliveryColorName(pkg.Color), correct, deliveryTime)
					ss.StatsTracker.RecordActionAt(bot.X, bot.Y, ss.ArenaW, ss.ArenaH)
				}
				// Truck scoring
				if ss.TruckToggle && ss.TruckState != nil {
					ts := ss.TruckState
					ts.DeliveredPkgs++
					if correct {
						ts.Score += 10
						ts.CorrectPkgs++
					} else {
						ts.Score += 3
						ts.WrongPkgs++
					}
				}
				// Evolution fitness
				if ss.EvolutionOn {
					if correct {
						bot.Fitness += 10
					} else {
						bot.Fitness += 3
					}
				}
				// Team scoring
				if ss.TeamsEnabled {
					if bot.Team == 1 {
						ss.TeamAScore++
					} else if bot.Team == 2 {
						ss.TeamBScore++
					}
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
		if !delivered && !nearStation {
			// Not near any station — ignore DROP, log warning
			logger.WarnBot(botIdx, "DELIVERY", "Bot #%d tried DROP without station nearby", botIdx)
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
		// Find nearest visible dropoff matching package color (beacon range for carrying bots)
		scanRange := swarm.SwarmBeaconRange
		bestDist := 1e9
		bestAngle := bot.Angle
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup || st.Color != pkg.Color {
				continue
			}
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= scanRange && d < bestDist {
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

	case swarmscript.ActTurnToRamp:
		if !ss.TruckToggle || ss.TruckState == nil {
			return
		}
		// Ramp semaphore: limit concurrent non-carrying bots on ramp
		if !bot.OnRamp {
			if bot.RampCooldown > 0 {
				// Cooldown active — turn random instead
				bot.Angle += (ss.Rng.Float64() - 0.5) * math.Pi
				bot.Speed = swarm.SwarmBotSpeed
				return
			}
			maxBots := ss.RampMaxBots
			if maxBots <= 0 {
				maxBots = 3
			}
			if ss.RampBotCount >= maxBots {
				// Ramp full — turn random, cooldown 60 ticks
				bot.Angle += (ss.Rng.Float64() - 0.5) * math.Pi
				bot.Speed = swarm.SwarmBotSpeed
				bot.RampCooldown = 60
				return
			}
		}
		ts := ss.TruckState
		// Target the right edge of the ramp, spread bots along Y axis
		cx := ts.RampX + ts.RampW + 20 // just outside ramp right edge
		// Distribute bots evenly along ramp height (use bot index for offset)
		slots := 20
		slot := botIdx % slots
		yFrac := (float64(slot) + 0.5) / float64(slots) // 0.025 .. 0.975
		cy := ts.RampY + ts.RampH*0.1 + ts.RampH*0.8*yFrac
		dx := cx - bot.X
		dy := cy - bot.Y
		bot.Angle = math.Atan2(dy, dx)

	case swarmscript.ActTurnToTruckPkg:
		if !ss.TruckToggle || ss.TruckState == nil || ss.TruckState.CurrentTruck == nil {
			return
		}
		t := ss.TruckState.CurrentTruck
		if t.Phase != swarm.TruckParked || bot.NearestTruckPkgIdx < 0 {
			return
		}
		pkg := &t.Packages[bot.NearestTruckPkgIdx]
		wpx := t.X + pkg.RelX + 18 + 4
		wpy := t.Y + pkg.RelY + 4
		dx := wpx - bot.X
		dy := wpy - bot.Y
		bot.Angle = math.Atan2(dy, dx)

	case swarmscript.ActTurnToBeaconDropoff:
		// Turn toward nearest beacon-heard matching dropoff station
		if !ss.DeliveryOn || bot.HeardBeaconDropoffColor == 0 {
			return
		}
		bot.Angle = bot.HeardBeaconDropoffAngle

	case swarmscript.ActSpiralFwd:
		// Spiral outward to explore arena when lost
		bot.ExplorationAngle += 0.03
		bot.Angle += bot.ExplorationAngle * 0.1
		bot.Speed = swarm.SwarmBotSpeed
		// If near edge, reverse spiral direction to bounce back into arena
		margin := 40.0
		if bot.X < margin || bot.X > ss.ArenaW-margin ||
			bot.Y < margin || bot.Y > ss.ArenaH-margin {
			bot.ExplorationAngle = -bot.ExplorationAngle
			bot.Angle += math.Pi * 0.3 // partial turn inward
		}

	case swarmscript.ActWallFollowRight:
		// Right-hand rule: keep wall on right side
		if bot.ObstacleAhead {
			bot.Angle -= math.Pi / 2 // wall in front → turn left 90°
		} else if !bot.WallRight {
			bot.Angle += math.Pi / 2 // lost wall on right → turn right 90° to refind
		}
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActWallFollowLeft:
		// Left-hand rule: keep wall on left side
		if bot.ObstacleAhead {
			bot.Angle += math.Pi / 2 // wall in front → turn right 90°
		} else if !bot.WallLeft {
			bot.Angle -= math.Pi / 2 // lost wall on left → turn left 90° to refind
		}
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActFollowPheromone:
		// Follow pheromone gradient uphill
		if ss.PherGrid != nil {
			gx, gy := ss.PherGrid.Gradient(bot.X, bot.Y)
			if gx != 0 || gy != 0 {
				bot.Angle = math.Atan2(gy, gx)
			}
		}
		bot.Speed = swarm.SwarmBotSpeed
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

	// Ramp barrier — prevent bots from entering ramp zone (trucks only)
	if ss.TruckToggle && ss.TruckState != nil {
		rampRight := swarm.SwarmRampX + swarm.SwarmRampW  // right edge of ramp (200)
		rampTop := swarm.SwarmRampY                       // 200
		rampBottom := swarm.SwarmRampY + swarm.SwarmRampH // 550
		br := swarm.SwarmBotRadius
		// If bot circle overlaps ramp rectangle, push it out to the right
		if bot.X-br < rampRight && bot.Y+br > rampTop && bot.Y-br < rampBottom {
			bot.X = rampRight + br
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

		// Check if bot is in the ramp entrance zone (exempt from left-edge bounce when trucks active)
		inRampZone := ss.TruckToggle && ss.TruckState != nil &&
			bot.Y >= swarm.SwarmRampY-r && bot.Y <= swarm.SwarmRampY+swarm.SwarmRampH+r

		if bot.X < r {
			if inRampZone {
				// Allow bots to reach X=0 (ramp area) — only soft clamp
				if bot.X < -r {
					bot.X = -r
					hitEdge = true
				}
			} else {
				bot.X = r
				hitEdge = true
			}
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
	const baseRepulsionRange = 30.0
	const rampRepulsionRange = 40.0
	const repulsionStrength = 0.15

	for i := range ss.Bots {
		a := &ss.Bots[i]
		// Use wider repulsion range on ramp to prevent clustering
		repRange := baseRepulsionRange
		if a.OnRamp {
			repRange = rampRepulsionRange
		}
		nearIDs := ss.Hash.Query(a.X, a.Y, repRange+1)
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
			// Use wider range if either bot is on ramp
			effectiveRange := baseRepulsionRange
			if a.OnRamp || b.OnRamp {
				effectiveRange = rampRepulsionRange
			}
			if dist >= effectiveRange || dist < 0.001 {
				continue
			}
			// Force = (range - dist) * strength, applied symmetrically
			force := (effectiveRange - dist) * repulsionStrength
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

// applyStationRepulsion pushes non-carrying bots away from dropoff stations.
// Bots without a matching package within 50px of a dropoff get a force pushing them away.
// Only bots carrying a matching package may approach.
func applyStationRepulsion(ss *swarm.SwarmState) {
	if !ss.DeliveryOn {
		return
	}
	const stationRepRange = 50.0
	const stationRepStrength = 0.2

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Bots carrying a matching package may approach dropoffs freely
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			pkg := &ss.Packages[bot.CarryingPkg]
			if bot.NearestDropoffIdx >= 0 && bot.NearestDropoffIdx < len(ss.Stations) {
				st := &ss.Stations[bot.NearestDropoffIdx]
				if pkg.Color == st.Color {
					continue // carrying matching package — don't repel
				}
			}
		}

		// Push non-carrying (or wrong-color-carrying) bots away from all nearby dropoffs
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup {
				continue
			}
			dx := bot.X - st.X
			dy := bot.Y - st.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist >= stationRepRange || dist < 0.1 {
				continue
			}
			// Force = (50 - dist) * 0.2, directed away from station
			force := (stationRepRange - dist) * stationRepStrength
			nx := dx / dist
			ny := dy / dist
			bot.X += nx * force
			bot.Y += ny * force
		}
	}
}

// applyClusterBreaker detects large clusters (connected components via 30px radius)
// and applies an outward explosion impulse to all bots in clusters > 5.
func applyClusterBreaker(ss *swarm.SwarmState) {
	n := len(ss.Bots)
	if n == 0 {
		return
	}

	// Union-Find
	parent := make([]int, n)
	rank := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra == rb {
			return
		}
		if rank[ra] < rank[rb] {
			ra, rb = rb, ra
		}
		parent[rb] = ra
		if rank[ra] == rank[rb] {
			rank[ra]++
		}
	}

	// Build clusters: bots within 30px are connected
	const clusterRadius = 30.0
	for i := range ss.Bots {
		a := &ss.Bots[i]
		nearIDs := ss.Hash.Query(a.X, a.Y, clusterRadius+1)
		for _, j := range nearIDs {
			if j <= i || j >= n {
				continue
			}
			b := &ss.Bots[j]
			dx := a.X - b.X
			dy := a.Y - b.Y
			if dx*dx+dy*dy < clusterRadius*clusterRadius {
				union(i, j)
			}
		}
	}

	// Count cluster sizes and find centroids
	type clusterInfo struct {
		count    int
		sumX     float64
		sumY     float64
		members  []int
	}
	clusters := make(map[int]*clusterInfo)
	for i := range ss.Bots {
		root := find(i)
		ci, ok := clusters[root]
		if !ok {
			ci = &clusterInfo{}
			clusters[root] = ci
		}
		ci.count++
		ci.sumX += ss.Bots[i].X
		ci.sumY += ss.Bots[i].Y
		ci.members = append(ci.members, i)
	}

	// Explode clusters > 5 bots (only if avg speed is low)
	for _, ci := range clusters {
		if ci.count <= 5 {
			continue
		}
		// Check average speed — only break slow-moving clusters
		var avgSpeed float64
		for _, idx := range ci.members {
			avgSpeed += ss.Bots[idx].Speed
		}
		avgSpeed /= float64(ci.count)
		if avgSpeed > 0.3 {
			continue // cluster is moving, leave it alone
		}

		cx := ci.sumX / float64(ci.count)
		cy := ci.sumY / float64(ci.count)
		logger.Info("CLUSTER", "Broke cluster of %d bots at (%.0f, %.0f) avgSpeed=%.2f", ci.count, cx, cy, avgSpeed)
		for _, idx := range ci.members {
			bot := &ss.Bots[idx]
			if bot.AntiStuckTimer > 0 || bot.ScatterTimer > 0 {
				continue // already being handled
			}
			// Exempt bots actively delivering (carry + match)
			if bot.CarryingPkg >= 0 && bot.DropoffMatch {
				continue
			}
			// Random outward impulse from centroid
			dx := bot.X - cx
			dy := bot.Y - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1.0 {
				angle := ss.Rng.Float64() * 2 * math.Pi
				bot.Angle = angle
			} else {
				bot.Angle = math.Atan2(dy, dx) + (ss.Rng.Float64()-0.5)*0.8
			}
			bot.Speed = swarm.SwarmBotSpeed * 1.3
			bot.ScatterTimer = 20
			bot.ScatterCooldown = 40
		}
	}
}
