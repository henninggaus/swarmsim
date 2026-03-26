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

	// Day/Night cycle: advance phase
	if ss.DayNightOn {
		ss.DayNightPhase += ss.DayNightSpeed
		if ss.DayNightPhase >= 1.0 {
			ss.DayNightPhase -= 1.0
		}
	}

	// Reset per-tick counters
	ss.CollisionCount = 0

	// Swap message buffers: this tick reads PrevMessages, writes to NextMessages
	ss.PrevMessages = ss.NextMessages
	ss.NextMessages = nil

	// Dropoff beacons: stations emit virtual SEND_DROPOFF messages periodically
	if ss.DeliveryOn && ss.Tick%DropoffBeaconInterval == 0 {
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

	// Phase 1.1: Apply sensor noise and failures
	swarm.ApplySensorNoise(ss)

	// Phase 1.5: Rebuild ramp semaphore count
	if ss.TruckToggle {
		ss.RampBotCount = 0
		for i := range ss.Bots {
			if ss.Bots[i].OnRamp && ss.Bots[i].CarryingPkg < 0 {
				ss.RampBotCount++
			}
		}
	}

	// Phase 1.6: A* Pathfinding (staggered batch computation)
	if ss.AStarOn {
		swarm.TickAStar(ss)
	}

	// Phase 1.7: Flocking (Boids) sensor computation
	swarm.TickFlocking(ss)

	// Phase 1.8: Dynamic Role + Rogue Detection sensors
	swarm.TickRoles(ss)
	swarm.TickRogue(ss)

	// Phase 1.9: Quorum Sensing (copy subsystem data to sensor cache)
	if ss.QuorumOn && ss.Quorum != nil {
		for i := range ss.Bots {
			if i < len(ss.Quorum.Votes) {
				ss.Bots[i].QuorumCount = ss.Quorum.Votes[i].LocalAgree
				if ss.Quorum.Votes[i].QuorumMet {
					ss.Bots[i].QuorumReached = 1
				} else {
					ss.Bots[i].QuorumReached = 0
				}
			}
		}
	}

	// Phase 1.10: Lévy-Flight Foraging
	if ss.LevyOn {
		swarm.TickLevy(ss)
	}

	// Phase 1.11: Firefly Synchronization
	if ss.FireflyOn {
		swarm.TickFirefly(ss)
	}

	// Phase 1.12: Collective Transport
	if ss.TransportOn {
		swarm.TickTransport(ss)
	}

	// Phase 1.13: Vortex Swarming sensor computation
	swarm.TickVortex(ss)

	// Phase 1.14: Waggle Dance
	if ss.WaggleOn {
		swarm.TickWaggle(ss)
	}

	// Phase 1.15: Morphogen Gradients
	if ss.MorphogenOn {
		swarm.TickMorphogen(ss)
	}

	// Phase 1.16: Predator Evasion Waves
	if ss.EvasionOn {
		swarm.TickEvasion(ss)
	}

	// Phase 1.17: Slime Mold Network
	if ss.SlimeOn {
		swarm.TickSlime(ss)
	}

	// Phase 1.18: Ant Bridge
	if ss.BridgeOn {
		swarm.TickBridge(ss)
	}

	// Phase 1.19: Shape Formation (SwarmScript sensor cache)
	if ss.ShapeFormationOn && ss.ShapeFormation != nil {
		// Update sensor cache for SwarmScript from existing shape formation
		for i := range ss.Bots {
			if i < len(ss.ShapeFormation.Assigned) && ss.ShapeFormation.Assigned[i] >= 0 {
				target := ss.ShapeFormation.TargetPositions[ss.ShapeFormation.Assigned[i]]
				dx := target[0] - ss.Bots[i].X
				dy := target[1] - ss.Bots[i].Y
				dist := math.Sqrt(dx*dx + dy*dy)
				ss.Bots[i].ShapeDist = int(math.Min(9999, dist))
				targetAngle := math.Atan2(dy, dx)
				diff := targetAngle - ss.Bots[i].Angle
				for diff > math.Pi {
					diff -= 2 * math.Pi
				}
				for diff < -math.Pi {
					diff += 2 * math.Pi
				}
				ss.Bots[i].ShapeAngle = int(diff * 180 / math.Pi)
			}
			ss.Bots[i].ShapeProgress = int(ss.ShapeFormation.Convergence * 100)
		}
	}

	// Phase 1.20: Mexican Wave
	if ss.WaveOn {
		swarm.TickWave(ss)
	}

	// Phase 1.21: Shepherd-Flock
	if ss.ShepherdOn {
		swarm.TickShepherd(ss)
	}

	// Phase 1.22: Particle Swarm Optimization
	if ss.PSOOn {
		swarm.TickPSO(ss)
	}

	// Phase 1.23: Predator-Prey sensor cache
	// Uses spatial hash with expanding search radii instead of O(n²) brute force.
	if ss.PredatorPreyOn && ss.PredatorPrey != nil {
		for i := range ss.Bots {
			if i >= len(ss.PredatorPrey.Roles) {
				continue
			}
			if ss.PredatorPrey.Roles[i] == swarm.RolePredator {
				ss.Bots[i].PredRole = 1
			} else {
				ss.Bots[i].PredRole = 0
			}
			// Find nearest opponent using spatial hash with expanding radii.
			// Most opponents are nearby, so start small and widen only if needed.
			bestDist := 9999.0
			myRole := ss.PredatorPrey.Roles[i]
			bx, by := ss.Bots[i].X, ss.Bots[i].Y
			for _, radius := range [3]float64{150, 400, ss.ArenaW + ss.ArenaH} {
				candidates := ss.Hash.Query(bx, by, radius)
				for _, j := range candidates {
					if j == i || j >= len(ss.PredatorPrey.Roles) {
						continue
					}
					if ss.PredatorPrey.Roles[j] == myRole {
						continue
					}
					dx := ss.Bots[j].X - bx
					dy := ss.Bots[j].Y - by
					d := math.Sqrt(dx*dx + dy*dy)
					if d < bestDist {
						bestDist = d
					}
				}
				if bestDist < radius {
					break // found opponent within this radius, no need to search wider
				}
			}
			ss.Bots[i].PreyDist = int(math.Min(9999, bestDist))
			if i < len(ss.PredatorPrey.CatchCount) {
				ss.Bots[i].PredCatches = ss.PredatorPrey.CatchCount[i]
			}
		}
	}

	// Phase 1.24: Magnetic Dipole Chains
	if ss.MagneticOn {
		swarm.TickMagnetic(ss)
	}

	// Phase 1.25: Cell Division
	if ss.DivisionOn {
		swarm.TickDivision(ss)
	}

	// Phase 1.26: V-Formation
	if ss.VFormationOn {
		swarm.TickVFormation(ss)
	}

	// Phase 1.27: Brood Sorting
	if ss.BroodOn {
		swarm.TickBrood(ss)
	}

	// Phase 1.28: Jellyfish Pulse
	if ss.JellyfishOn {
		swarm.TickJellyfish(ss)
	}

	// Phase 1.29: Immune Swarm (antibody/pathogen)
	if ss.ImmuneSwarmOn {
		swarm.TickImmuneSwarm(ss)
	}

	// Phase 1.30: Gravitational N-Body
	if ss.GravityOn {
		swarm.TickGravity(ss)
	}

	// Phase 1.31: Crystallization
	if ss.CrystalOn {
		swarm.TickCrystal(ss)
	}

	// Phase 1.32: Amoeba Locomotion
	if ss.AmoebaOn {
		swarm.TickAmoeba(ss)
	}

	// Phase 1.33: Ant Colony Optimization
	if ss.ACOOn {
		swarm.TickACO(ss)
	}

	// Phase 1.34: Bacterial Foraging Optimization
	if ss.BFOOn {
		swarm.TickBFO(ss)
	}

	// Phase 1.35: Grey Wolf Optimizer
	if ss.GWOOn {
		swarm.TickGWO(ss)
	}

	// Phase 1.36: Whale Optimization Algorithm
	if ss.WOAOn {
		swarm.TickWOA(ss)
	}

	// Phase 1.37: Moth-Flame Optimization
	if ss.MFOOn {
		swarm.TickMFO(ss)
	}

	// Phase 1.38: Cuckoo Search
	if ss.CuckooOn {
		swarm.TickCuckoo(ss)
	}

	// Phase 1.39: Differential Evolution
	if ss.DEOn {
		swarm.TickDE(ss)
	}

	// Phase 1.40: Artificial Bee Colony
	if ss.ABCOn {
		swarm.TickABC(ss)
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
			// Truck mode: use truck-specific sensors and actions
			if ss.TruckToggle && ss.TruckState != nil {
				inputs := swarm.BuildNeuroTruckInputs(bot, ss)
				actionIdx := swarm.NeuroForward(bot.Brain, inputs)
				swarm.ExecuteNeuroTruckAction(actionIdx, bot, ss, i)
			} else {
				inputs := swarm.BuildNeuroInputs(bot, ss)
				actionIdx := swarm.NeuroForward(bot.Brain, inputs)
				swarm.ExecuteNeuroAction(actionIdx, bot, ss, i)
			}
			// Auto-pickup/drop for neuro bots: the net navigates, the system handles interaction
			if ss.DeliveryOn {
				if bot.CarryingPkg < 0 && bot.NearestPickupDist < 20 && bot.NearestPickupHasPkg {
					executeSwarmAction(swarmscript.Action{Type: swarmscript.ActPickup}, bot, ss, i)
				}
				if bot.CarryingPkg >= 0 && bot.DropoffMatch && bot.NearestDropoffDist < 30 {
					executeSwarmAction(swarmscript.Action{Type: swarmscript.ActDrop}, bot, ss, i)
				}
			}
			// Auto-pickup from truck ramp for neuro bots
			if ss.TruckToggle && bot.OnRamp && bot.CarryingPkg < 0 && bot.NearestTruckPkgIdx >= 0 {
				executeSwarmAction(swarmscript.Action{Type: swarmscript.ActTurnToTruckPkg}, bot, ss, i)
			}
			continue
		}
		// LSTM: use LSTM neural network if LSTM is ON
		if ss.LSTMEnabled && bot.LSTMBrain != nil {
			inputs := swarm.BuildNeuroInputs(bot, ss)
			actionIdx := swarm.LSTMForward(bot.LSTMBrain, inputs)
			swarm.ExecuteNeuroAction(actionIdx, bot, ss, i)
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

		// (A) Scatter: >3 close neighbors → forced TURN_FROM_NEAREST + FWD
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
			bot.ScatterTimer = ScatterDuration
			bot.ScatterCooldown = ScatterCooldownTicks
			if bot.NearestIdx >= 0 {
				other := &ss.Bots[bot.NearestIdx]
				dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
				bot.Angle = math.Atan2(-dy, -dx) + (ss.Rng.Float64()-0.5)*0.4
			}
			bot.Speed = swarm.SwarmBotSpeed
			continue
		}

		// (B) Idle exploration: carry==0 bots that stay still → forced random FWD
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
			if bot.IdleMoveTicks >= IdleThreshold {
				bot.IdleMoveTimer = IdleExploreDuration
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
		if ss.Tick%DeliveryLogInterval == 0 && ss.DeliveryStats.TotalDelivered > 0 {
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
		if ss.Tick%TruckDebugLogInterval == 0 {
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
		if ss.EvolutionTimer >= EvolutionInterval {
			swarm.RunEvolution(ss)
			ss.EvolutionSoundPending = true
			dm := swarm.MeasureParamDiversity(ss)
			ss.Diversity = &dm
			// Speciation update after evolution
			if ss.SpeciationOn {
				if ss.Speciation == nil {
					swarm.InitSpeciation(ss)
				}
				swarm.UpdateSpeciation(ss)
			}
		}
	}

	// Phase 4.85: Genetic Programming evolution
	if ss.GPEnabled {
		ss.GPTimer++
		if ss.GPTimer >= GPEvolutionInterval {
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
		if ss.NeuroTimer >= NeuroEvolutionInterval {
			swarm.RunNeuroEvolution(ss)
			ss.EvolutionSoundPending = true
			dm := swarm.MeasureNeuroDiversity(ss)
			ss.Diversity = &dm
		}
	}

	// Phase 4.88: LSTM neuroevolution
	if ss.LSTMEnabled {
		ss.LSTMTimer++
		if ss.LSTMTimer >= NeuroEvolutionInterval {
			swarm.RunLSTMEvolution(ss)
			ss.EvolutionSoundPending = true
			dm := swarm.MeasureLSTMDiversity(ss)
			ss.Diversity = &dm
		}
	}

	// Count broadcasts for sound
	ss.BroadcastCount = len(ss.NextMessages)

	// Phase 4.88: Tournament timer
	if ss.TournamentOn && ss.TournamentPhase == 1 {
		swarm.TournamentTick(ss)
	}

	// Phase 4.88b: Algorithm auto-tournament — tick the active algorithm via
	// the centralized dispatch (which also records convergence) and advance
	// the tournament queue when the current algorithm's budget expires.
	if ss.AlgoTournamentOn {
		swarm.TickSwarmAlgorithm(ss)
		swarm.TickAlgoTournament(ss)
	}

	// Phase 4.89: Telemetry sampling (writes to telemetry.jsonl if enabled)
	if ss.Telemetry != nil {
		ss.Telemetry.Sample(ss)
	}

	// Phase 4.9: Challenge timer update
	if ss.TeamsEnabled {
		swarm.UpdateChallenge(ss)
	}

	// Phase 4.92: Statistics tracker update
	if ss.StatsTracker != nil {
		ss.StatsTracker.Update(ss)
		// Heatmap update every 100 ticks
		if ss.Tick%HeatmapUpdateInterval == 0 {
			ss.StatsTracker.UpdateHeatmap(ss)
		}
		// Rankings update periodically
		if ss.Tick%RankingsUpdateInterval == 0 {
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

	// Phase 4.93: Heatmap accumulation (periodic for performance)
	if ss.ShowHeatmap && ss.Tick%SwarmHeatmapInterval == 0 {
		swarm.UpdateHeatmap(ss)
	}

	// Phase 4.93b: Swarm center + congestion grid (every 10 ticks)
	if ss.Tick%10 == 0 {
		if ss.ShowSwarmCenter || ss.ShowPrediction {
			swarm.ComputeSwarmCenter(ss)
		}
		if ss.ShowZones {
			swarm.UpdateCongestionGrid(ss)
		}
	}

	// Phase 4.93c: Swarm awareness sensors (every tick when needed)
	swarm.ComputeSwarmAwarenessSensors(ss)

	// Phase 4.93d: Pattern detection (every 30 ticks)
	if ss.ShowPatterns && ss.Tick%30 == 0 {
		pr := swarm.DetectPatterns(ss)
		ss.PatternResult = &pr
	}

	// Phase 4.93e: Achievement checking (every 60 ticks)
	if ss.AchievementState != nil && ss.Tick%60 == 0 {
		swarm.CheckAchievements(ss)
	}

	// Phase 4.94: Bot spatial memory update (periodic)
	if ss.MemoryEnabled && ss.Tick%MemoryUpdateInterval == 0 {
		swarm.UpdateBotMemory(ss)
	}

	// Phase 4.94a: Memory decay (exponential forgetting, every 50 ticks)
	if ss.MemoryEnabled && ss.Tick%50 == 0 && ss.Tick > 0 {
		swarm.DecayBotMemory(ss, 0.92) // ~8% decay per 50 ticks
	}

	// Phase 4.94b: Update moving obstacles (patrol + rotation)
	if swarm.HasMovingObstacles(ss) {
		swarm.UpdateMovingObstacles(ss)
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
		// Novelty Search: accumulate neighbor count for behavior descriptor
		bot.Stats.SumNeighborCount += float64(bot.NeighborCount)
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
	bot.GroupCarry = 0
	bot.GroupSpeed = 0
	bot.GroupSize = 0

	// Query neighbors within sensor range (carrying bots get extended 200px range)
	sensorRange := swarm.SwarmSensorRange
	if bot.CarryingPkg >= 0 {
		sensorRange = swarm.SwarmDeliverySensorRange
	}
	// Day/Night: reduce sensor range at night
	if ss.DayNightOn {
		brightness := swarm.DayNightBrightness(ss)
		sensorRange *= (0.4 + 0.6*brightness)
	}
	candidateIDs := ss.Hash.Query(bot.X, bot.Y, sensorRange)
	var sumX, sumY float64
	count := 0
	var carryCount int
	var speedSum float64

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

		// Cooperative sensor accumulation
		if other.CarryingPkg >= 0 {
			carryCount++
		}
		speedSum += other.Speed
	}

	bot.NeighborCount = count
	if count > 0 {
		bot.AvgNeighborX = sumX / float64(count)
		bot.AvgNeighborY = sumY / float64(count)
		bot.GroupCarry = carryCount * 100 / count
		bot.GroupSpeed = int(speedSum / float64(count) * 100)
	}
	if bot.NearestDist > 1e8 {
		bot.NearestDist = 999
	}
	// Cluster size (simple: count + self; could be BFS but that's expensive per tick)
	bot.GroupSize = count + 1

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

	// Advanced sensors: neighbor_min_dist (integer form of NearestDist)
	if bot.NearestDist < 1e8 {
		bot.NeighborMinDist = int(bot.NearestDist)
	} else {
		bot.NeighborMinDist = 9999
	}

	// bot_carrying: count of neighbors that are carrying packages
	bot.BotCarryingCount = carryCount

	// time_since_delivery: ticks since last successful delivery (tracked in stats)
	bot.TimeSinceDelivery++

	// recent_collision: update collision timer
	if bot.CollisionTimer > 0 {
		bot.CollisionTimer--
		bot.RecentCollision = 1
	} else {
		bot.RecentCollision = 0
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
	case swarmscript.CondGroupCarry:
		return compareInt(bot.GroupCarry, cond.Op, cv)
	case swarmscript.CondGroupSpeed:
		return compareInt(bot.GroupSpeed, cond.Op, cv)
	case swarmscript.CondGroupSize:
		return compareInt(bot.GroupSize, cond.Op, cv)
	case swarmscript.CondSwarmCenterDist:
		return compareInt(bot.SwarmCenterDist, cond.Op, cv)
	case swarmscript.CondSwarmSpread:
		return compareInt(bot.SwarmSpreadSensor, cond.Op, cv)
	case swarmscript.CondIsolationLevel:
		return compareInt(bot.IsolationLevel, cond.Op, cv)
	case swarmscript.CondResourceGradientX:
		return compareInt(bot.ResourceGradientX, cond.Op, cv)
	case swarmscript.CondResourceGradientY:
		return compareInt(bot.ResourceGradientY, cond.Op, cv)

	case swarmscript.CondEnergy:
		return compareInt(int(bot.Energy), cond.Op, cv)
	case swarmscript.CondBotCarrying:
		return compareInt(bot.BotCarryingCount, cond.Op, cv)
	case swarmscript.CondTimeSinceDelivery:
		return compareInt(bot.TimeSinceDelivery, cond.Op, cv)
	case swarmscript.CondRecentCollision:
		return compareInt(bot.RecentCollision, cond.Op, cv)
	case swarmscript.CondNeighborMinDist:
		return compareInt(bot.NeighborMinDist, cond.Op, cv)
	case swarmscript.CondPathDist:
		return compareInt(bot.PathDist, cond.Op, cv)
	case swarmscript.CondPathAngle:
		return compareInt(bot.PathAngle, cond.Op, cv)

	// Flocking (Boids) sensors
	case swarmscript.CondFlockAlign:
		return compareInt(bot.FlockAlign, cond.Op, cv)
	case swarmscript.CondFlockCohesion:
		return compareInt(bot.FlockCohesion, cond.Op, cv)
	case swarmscript.CondFlockSeparation:
		return compareInt(bot.FlockSeparation, cond.Op, cv)

	// Dynamic Role sensors
	case swarmscript.CondRole:
		return compareInt(bot.Role, cond.Op, cv)
	case swarmscript.CondRoleDemand:
		return compareInt(bot.RoleDemand, cond.Op, cv)

	// Quorum Sensing sensors
	case swarmscript.CondVote:
		return compareInt(bot.Vote, cond.Op, cv)
	case swarmscript.CondQuorumCount:
		return compareInt(bot.QuorumCount, cond.Op, cv)
	case swarmscript.CondQuorumReached:
		return compareInt(bot.QuorumReached, cond.Op, cv)

	// Rogue Detection sensors
	case swarmscript.CondReputation:
		return compareInt(bot.Reputation, cond.Op, cv)
	case swarmscript.CondSuspectNearby:
		return compareInt(bot.SuspectNearby, cond.Op, cv)

	// Lévy-Flight sensors
	case swarmscript.CondLevyPhase:
		return compareInt(bot.LevyPhase, cond.Op, cv)
	case swarmscript.CondLevyStep:
		return compareInt(bot.LevyStep, cond.Op, cv)

	// Firefly Sync sensors
	case swarmscript.CondFlashPhase:
		return compareInt(bot.FlashPhase, cond.Op, cv)
	case swarmscript.CondFlashSync:
		return compareInt(bot.FlashSync, cond.Op, cv)

	// Collective Transport sensors
	case swarmscript.CondTransportNearby:
		return compareInt(bot.TransportNearby, cond.Op, cv)
	case swarmscript.CondTransportCount:
		return compareInt(bot.TransportCount, cond.Op, cv)

	// Vortex Swarming sensors
	case swarmscript.CondVortexStrength:
		return compareInt(bot.VortexStrength, cond.Op, cv)

	// Waggle Dance sensors
	case swarmscript.CondWaggleDancing:
		return compareInt(bot.WaggleDancing, cond.Op, cv)
	case swarmscript.CondWaggleTarget:
		return compareInt(bot.WaggleTarget, cond.Op, cv)

	// Morphogen sensors
	case swarmscript.CondMorphA:
		return compareInt(bot.MorphA, cond.Op, cv)
	case swarmscript.CondMorphH:
		return compareInt(bot.MorphH, cond.Op, cv)

	// Evasion Wave sensors
	case swarmscript.CondEvasionAlert:
		return compareInt(bot.EvasionAlert, cond.Op, cv)
	case swarmscript.CondEvasionWave:
		return compareInt(bot.EvasionWave, cond.Op, cv)

	// Slime Mold sensors
	case swarmscript.CondSlimeTrail:
		return compareInt(bot.SlimeTrail, cond.Op, cv)
	case swarmscript.CondSlimeGrad:
		return compareInt(bot.SlimeGrad, cond.Op, cv)

	// Ant Bridge sensors
	case swarmscript.CondBridgeActive:
		return compareInt(bot.BridgeActive, cond.Op, cv)
	case swarmscript.CondBridgeNearby:
		return compareInt(bot.BridgeNearby, cond.Op, cv)

	// Shape Formation sensors
	case swarmscript.CondShapeDist:
		return compareInt(bot.ShapeDist, cond.Op, cv)
	case swarmscript.CondShapeAngle:
		return compareInt(bot.ShapeAngle, cond.Op, cv)
	case swarmscript.CondShapeProgress:
		return compareInt(bot.ShapeProgress, cond.Op, cv)

	// Mexican Wave sensors
	case swarmscript.CondWaveFlash:
		return compareInt(bot.WaveFlash, cond.Op, cv)
	case swarmscript.CondWavePhase:
		return compareInt(bot.WavePhase, cond.Op, cv)

	// Shepherd-Flock sensors
	case swarmscript.CondShepherdRole:
		return compareInt(bot.ShepherdRole, cond.Op, cv)
	case swarmscript.CondShepherdDist:
		return compareInt(bot.ShepherdDist, cond.Op, cv)
	case swarmscript.CondFlockToTarget:
		return compareInt(bot.FlockToTarget, cond.Op, cv)

	// PSO sensors
	case swarmscript.CondPSOFitness:
		return compareInt(bot.PSOFitness, cond.Op, cv)
	case swarmscript.CondPSOBest:
		return compareInt(bot.PSOBest, cond.Op, cv)
	case swarmscript.CondPSOGlobalDist:
		return compareInt(bot.PSOGlobalDist, cond.Op, cv)

	// Predator-Prey sensors
	case swarmscript.CondPredRole:
		return compareInt(bot.PredRole, cond.Op, cv)
	case swarmscript.CondPreyDist:
		return compareInt(bot.PreyDist, cond.Op, cv)
	case swarmscript.CondPredCatches:
		return compareInt(bot.PredCatches, cond.Op, cv)

	// Magnetic Chain sensors
	case swarmscript.CondMagChainLen:
		return compareInt(bot.MagChainLen, cond.Op, cv)
	case swarmscript.CondMagLinked:
		return compareInt(bot.MagLinked, cond.Op, cv)
	case swarmscript.CondMagAlign:
		return compareInt(bot.MagAlign, cond.Op, cv)

	// Cell Division sensors
	case swarmscript.CondDivGroup:
		return compareInt(bot.DivGroup, cond.Op, cv)
	case swarmscript.CondDivPhase:
		return compareInt(bot.DivPhase, cond.Op, cv)
	case swarmscript.CondDivDist:
		return compareInt(bot.DivDist, cond.Op, cv)
	// V-Formation conditions
	case swarmscript.CondVFormPos:
		return compareInt(bot.VFormPos, cond.Op, cv)
	case swarmscript.CondVFormDraft:
		return compareInt(bot.VFormDraft, cond.Op, cv)
	case swarmscript.CondVFormLeader:
		return compareInt(bot.VFormLeader, cond.Op, cv)
	// Brood Sorting conditions
	case swarmscript.CondBroodCarrying:
		return compareInt(bot.BroodCarrying, cond.Op, cv)
	case swarmscript.CondBroodItemColor:
		return compareInt(bot.BroodItemColor, cond.Op, cv)
	case swarmscript.CondBroodDensity:
		return compareInt(bot.BroodDensity, cond.Op, cv)
	case swarmscript.CondBroodSameColor:
		return compareInt(bot.BroodSameColor, cond.Op, cv)
	// Jellyfish Pulse conditions
	case swarmscript.CondJellyPhase:
		return compareInt(bot.JellyPhase, cond.Op, cv)
	case swarmscript.CondJellyExpanding:
		return compareInt(bot.JellyExpanding, cond.Op, cv)
	case swarmscript.CondJellyRadius:
		return compareInt(bot.JellyRadius, cond.Op, cv)
	// Immune System conditions
	case swarmscript.CondImmuneRole:
		return compareInt(bot.ImmuneRole, cond.Op, cv)
	case swarmscript.CondImmuneAlert:
		return compareInt(bot.ImmuneAlert, cond.Op, cv)
	case swarmscript.CondImmunePathDist:
		return compareInt(bot.ImmunePathDist, cond.Op, cv)
	// Gravitational N-Body conditions
	case swarmscript.CondGravMass:
		return compareInt(bot.GravMass, cond.Op, cv)
	case swarmscript.CondGravForce:
		return compareInt(bot.GravForce, cond.Op, cv)
	case swarmscript.CondGravNearHeavy:
		return compareInt(bot.GravNearHeavy, cond.Op, cv)
	// Crystallization conditions
	case swarmscript.CondCrystalNeigh:
		return compareInt(bot.CrystalNeigh, cond.Op, cv)
	case swarmscript.CondCrystalDefect:
		return compareInt(bot.CrystalDefect, cond.Op, cv)
	case swarmscript.CondCrystalSettled:
		return compareInt(bot.CrystalSettled, cond.Op, cv)
	// Amoeba conditions
	case swarmscript.CondAmoebaDistCenter:
		return compareInt(bot.AmoebaDistCenter, cond.Op, cv)
	case swarmscript.CondAmoebaSkin:
		return compareInt(bot.AmoebaSkin, cond.Op, cv)
	case swarmscript.CondAmoebaPseudo:
		return compareInt(bot.AmoebaPseudo, cond.Op, cv)
	// ACO conditions
	case swarmscript.CondACOTrail:
		return compareInt(bot.ACOTrail, cond.Op, cv)
	case swarmscript.CondACOGrad:
		return compareInt(bot.ACOGrad, cond.Op, cv)
	// Bacterial Foraging conditions
	case swarmscript.CondBFOHealth:
		return compareInt(bot.BFOHealth, cond.Op, cv)
	case swarmscript.CondBFOSwimming:
		return compareInt(bot.BFOSwimming, cond.Op, cv)
	case swarmscript.CondBFONutrient:
		return compareInt(bot.BFONutrient, cond.Op, cv)
	// Grey Wolf Optimizer conditions
	case swarmscript.CondGWORank:
		return compareInt(bot.GWORank, cond.Op, cv)
	case swarmscript.CondGWOFitness:
		return compareInt(bot.GWOFitness, cond.Op, cv)
	case swarmscript.CondGWOAlphaDist:
		return compareInt(bot.GWOAlphaDist, cond.Op, cv)
	// Whale Optimization conditions
	case swarmscript.CondWOAPhase:
		return compareInt(bot.WOAPhase, cond.Op, cv)
	case swarmscript.CondWOAFitness:
		return compareInt(bot.WOAFitness, cond.Op, cv)
	case swarmscript.CondWOABestDist:
		return compareInt(bot.WOABestDist, cond.Op, cv)
	// Moth-Flame Optimization conditions
	case swarmscript.CondMFOFlame:
		return compareInt(bot.MFOFlame, cond.Op, cv)
	case swarmscript.CondMFOFitness:
		return compareInt(bot.MFOFitness, cond.Op, cv)
	case swarmscript.CondMFOFlameDist:
		return compareInt(bot.MFOFlameDist, cond.Op, cv)
	// Cuckoo Search conditions
	case swarmscript.CondCuckooFitness:
		return compareInt(bot.CuckooFitness, cond.Op, cv)
	case swarmscript.CondCuckooNestAge:
		return compareInt(bot.CuckooNestAge, cond.Op, cv)
	case swarmscript.CondCuckooBest:
		return compareInt(bot.CuckooBest, cond.Op, cv)
	// Differential Evolution conditions
	case swarmscript.CondDEFitness:
		return compareInt(bot.DEFitness, cond.Op, cv)
	case swarmscript.CondDEBestDist:
		return compareInt(bot.DEBestDist, cond.Op, cv)
	case swarmscript.CondDEPhase:
		return compareInt(bot.DEPhase, cond.Op, cv)
	// Artificial Bee Colony conditions
	case swarmscript.CondABCFitness:
		return compareInt(bot.ABCFitness, cond.Op, cv)
	case swarmscript.CondABCRole:
		return compareInt(bot.ABCRole, cond.Op, cv)
	case swarmscript.CondABCBestDist:
		return compareInt(bot.ABCBestDist, cond.Op, cv)
	// Harmony Search Optimization conditions
	case swarmscript.CondHSOFitness:
		return compareInt(bot.HSOFitness, cond.Op, cv)
	case swarmscript.CondHSOPhase:
		return compareInt(bot.HSOPhase, cond.Op, cv)
	case swarmscript.CondHSOBestDist:
		return compareInt(bot.HSOBestDist, cond.Op, cv)
	// Bat Algorithm conditions
	case swarmscript.CondBatLoud:
		return compareInt(bot.BatLoud, cond.Op, cv)
	case swarmscript.CondBatPulse:
		return compareInt(bot.BatPulse, cond.Op, cv)
	case swarmscript.CondBatFitness:
		return compareInt(bot.BatFitness, cond.Op, cv)
	case swarmscript.CondBatBestDist:
		return compareInt(bot.BatBestDist, cond.Op, cv)
	// Salp Swarm Algorithm conditions
	case swarmscript.CondSSARole:
		return compareInt(bot.SSARole, cond.Op, cv)
	case swarmscript.CondSSAFitness:
		return compareInt(bot.SSAFitness, cond.Op, cv)
	case swarmscript.CondSSAFoodDist:
		return compareInt(bot.SSAFoodDist, cond.Op, cv)
	// Gravitational Search Algorithm conditions
	case swarmscript.CondGSAMass:
		return compareInt(bot.GSAMass, cond.Op, cv)
	case swarmscript.CondGSAForce:
		return compareInt(bot.GSAForce, cond.Op, cv)
	case swarmscript.CondGSABestDist:
		return compareInt(bot.GSABestDist, cond.Op, cv)
	// Flower Pollination Algorithm conditions
	case swarmscript.CondFPAFitness:
		return compareInt(bot.FPAFitness, cond.Op, cv)
	case swarmscript.CondFPAType:
		return compareInt(bot.FPAType, cond.Op, cv)
	case swarmscript.CondFPABestDist:
		return compareInt(bot.FPABestDist, cond.Op, cv)
	// Harris Hawks Optimization conditions
	case swarmscript.CondHHOPhase:
		return compareInt(bot.HHOPhase, cond.Op, cv)
	case swarmscript.CondHHOFitness:
		return compareInt(bot.HHOFitness, cond.Op, cv)
	case swarmscript.CondHHOBestDist:
		return compareInt(bot.HHOBestDist, cond.Op, cv)
	// Simulated Annealing conditions
	case swarmscript.CondSAFitness:
		return compareInt(bot.SAFitness, cond.Op, cv)
	case swarmscript.CondSATemp:
		return compareInt(bot.SATemp, cond.Op, cv)
	case swarmscript.CondSABestDist:
		return compareInt(bot.SABestDist, cond.Op, cv)
	// Aquila Optimizer conditions
	case swarmscript.CondAOPhase:
		return compareInt(bot.AOPhase, cond.Op, cv)
	case swarmscript.CondAOFitness:
		return compareInt(bot.AOFitness, cond.Op, cv)
	case swarmscript.CondAOBestDist:
		return compareInt(bot.AOBestDist, cond.Op, cv)
	// Sine Cosine Algorithm conditions
	case swarmscript.CondSCAFitness:
		return compareInt(bot.SCAFitness, cond.Op, cv)
	case swarmscript.CondSCAPhase:
		return compareInt(bot.SCAPhase, cond.Op, cv)
	case swarmscript.CondSCABestDist:
		return compareInt(bot.SCABestDist, cond.Op, cv)
	// Dragonfly Algorithm conditions
	case swarmscript.CondDAFitness:
		return compareInt(bot.DAFitness, cond.Op, cv)
	case swarmscript.CondDARole:
		return compareInt(bot.DARole, cond.Op, cv)
	case swarmscript.CondDAFoodDist:
		return compareInt(bot.DAFoodDist, cond.Op, cv)
	// Teaching-Learning-Based Optimization conditions
	case swarmscript.CondTLBOFitness:
		return compareInt(bot.TLBOFitness, cond.Op, cv)
	case swarmscript.CondTLBOPhase:
		return compareInt(bot.TLBOPhase, cond.Op, cv)
	case swarmscript.CondTLBOTeacherDist:
		return compareInt(bot.TLBOTeacherDist, cond.Op, cv)
	// Equilibrium Optimizer conditions
	case swarmscript.CondEOFitness:
		return compareInt(bot.EOFitness, cond.Op, cv)
	case swarmscript.CondEOPhase:
		return compareInt(bot.EOPhase, cond.Op, cv)
	case swarmscript.CondEOEquilDist:
		return compareInt(bot.EOEquilDist, cond.Op, cv)

	// Jaya Algorithm conditions
	case swarmscript.CondJayaFitness:
		return compareInt(bot.JayaFitness, cond.Op, cv)
	case swarmscript.CondJayaBestDist:
		return compareInt(bot.JayaBestDist, cond.Op, cv)
	case swarmscript.CondJayaWorstDist:
		return compareInt(bot.JayaWorstDist, cond.Op, cv)
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
				bot.TimeSinceDelivery = 0 // reset time-since-delivery sensor
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

	case swarmscript.ActFollowPath:
		// Follow computed A* path toward next waypoint
		if ss.AStarOn && ss.AStar != nil {
			swarm.FollowPath(bot, ss, botIdx)
		} else {
			bot.Speed = swarm.SwarmBotSpeed
		}

	case swarmscript.ActFlock:
		// Apply all three Reynolds rules (separation + alignment + cohesion)
		swarm.ApplyFlock(bot, ss, botIdx)

	case swarmscript.ActAlign:
		// Align heading with neighbors
		swarm.ApplyAlign(bot, ss, botIdx)

	case swarmscript.ActCohere:
		// Steer toward neighbor center of mass
		swarm.ApplyCohere(bot, ss, botIdx)

	case swarmscript.ActBecomeScout:
		swarm.SetRole(bot, swarm.BotRoleScout)

	case swarmscript.ActBecomeWorker:
		swarm.SetRole(bot, swarm.BotRoleWorker)

	case swarmscript.ActBecomeGuard:
		swarm.SetRole(bot, swarm.BotRoleGuard)

	case swarmscript.ActVote:
		bot.Vote = act.Param1

	case swarmscript.ActFlagRogue:
		swarm.FlagRogue(bot, ss, botIdx)

	case swarmscript.ActLevyWalk:
		swarm.ApplyLevyWalk(bot, ss, botIdx)

	case swarmscript.ActFlash:
		swarm.ApplyFlash(bot, ss, botIdx)

	case swarmscript.ActAssistTransport:
		swarm.ApplyAssistTransport(bot, ss, botIdx)

	case swarmscript.ActVortex:
		swarm.ApplyVortex(bot, ss, botIdx)

	case swarmscript.ActWaggleDance:
		swarm.ApplyWaggleDance(bot, ss, botIdx)

	case swarmscript.ActFollowDance:
		swarm.ApplyFollowDance(bot, ss, botIdx)

	case swarmscript.ActMorphColor:
		swarm.ApplyMorphColor(bot, ss, botIdx)

	case swarmscript.ActEvade:
		swarm.ApplyEvade(bot, ss, botIdx)

	case swarmscript.ActFollowSlime:
		swarm.ApplyFollowSlime(bot, ss, botIdx)

	case swarmscript.ActFormBridge:
		swarm.ApplyFormBridge(bot, ss, botIdx)

	case swarmscript.ActCrossBridge:
		swarm.ApplyCrossBridge(bot, ss, botIdx)

	case swarmscript.ActFormShape:
		if ss.ShapeFormationOn && ss.ShapeFormation != nil {
			// Steer toward assigned shape target
			if botIdx < len(ss.ShapeFormation.Assigned) && ss.ShapeFormation.Assigned[botIdx] >= 0 {
				target := ss.ShapeFormation.TargetPositions[ss.ShapeFormation.Assigned[botIdx]]
				dx := target[0] - bot.X
				dy := target[1] - bot.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > 8 {
					targetAngle := math.Atan2(dy, dx)
					diff := targetAngle - bot.Angle
					for diff > math.Pi {
						diff -= 2 * math.Pi
					}
					for diff < -math.Pi {
						diff += 2 * math.Pi
					}
					if diff > 0.15 {
						diff = 0.15
					} else if diff < -0.15 {
						diff = -0.15
					}
					bot.Angle += diff
					if dist < 30 {
						bot.Speed = swarm.SwarmBotSpeed * 0.4
					} else {
						bot.Speed = swarm.SwarmBotSpeed
					}
				} else {
					bot.Speed = 0
					bot.LEDColor = [3]uint8{0, 255, 100}
				}
			}
		}

	case swarmscript.ActWaveFlash:
		swarm.ApplyWaveFlash(bot, ss, botIdx)

	case swarmscript.ActShepherd:
		swarm.ApplyShepherd(bot, ss, botIdx)

	case swarmscript.ActPSOMove:
		swarm.ApplyPSOMove(bot, ss, botIdx)

	case swarmscript.ActPredator:
		swarm.ApplyPredator(bot, ss, botIdx)

	case swarmscript.ActMagnetic:
		swarm.ApplyMagnetic(bot, ss, botIdx)

	case swarmscript.ActDivide:
		swarm.ApplyDivision(bot, ss, botIdx)

	case swarmscript.ActVFormation:
		swarm.ApplyVFormation(bot, ss, botIdx)

	case swarmscript.ActBroodSort:
		swarm.ApplyBroodSort(bot, ss, botIdx)

	case swarmscript.ActJellyfishPulse:
		swarm.ApplyJellyfishPulse(bot, ss, botIdx)

	case swarmscript.ActImmune:
		swarm.ApplyImmuneSwarm(bot, ss, botIdx)

	case swarmscript.ActGravity:
		swarm.ApplyGravity(bot, ss, botIdx)

	case swarmscript.ActCrystal:
		swarm.ApplyCrystal(bot, ss, botIdx)

	case swarmscript.ActAmoeba:
		swarm.ApplyAmoeba(bot, ss, botIdx)

	case swarmscript.ActACO:
		swarm.ApplyACO(bot, ss, botIdx)

	case swarmscript.ActBFO:
		swarm.ApplyBFO(bot, ss, botIdx)

	case swarmscript.ActGWO:
		swarm.ApplyGWO(bot, ss, botIdx)

	case swarmscript.ActWOA:
		swarm.ApplyWOA(bot, ss, botIdx)

	case swarmscript.ActMFO:
		swarm.ApplyMFO(bot, ss, botIdx)

	case swarmscript.ActCuckoo:
		swarm.ApplyCuckoo(bot, ss, botIdx)

	case swarmscript.ActDE:
		swarm.ApplyDE(bot, ss, botIdx)

	case swarmscript.ActABC:
		swarm.ApplyABC(bot, ss, botIdx)

	case swarmscript.ActDash:
		// Double-speed burst for 10 ticks (costs 15 energy, 60 tick cooldown)
		if bot.DashCooldown <= 0 && bot.Energy >= 15 {
			bot.DashTimer = 10
			bot.DashCooldown = 60
			bot.Energy -= 15
		}
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActEmergencyBroadcast:
		// Broadcast with 3x communication range (message value from param)
		msgVal := act.Param1
		if msgVal == 0 {
			msgVal = 99 // default emergency value
		}
		// Send 3 copies at wider offsets to simulate 3x range
		for _, offset := range [][2]float64{{0, 0}, {swarm.SwarmCommRange, 0}, {-swarm.SwarmCommRange, 0}, {0, swarm.SwarmCommRange}, {0, -swarm.SwarmCommRange}} {
			ss.NextMessages = append(ss.NextMessages, swarm.SwarmMessage{
				Value: msgVal,
				X:     bot.X + offset[0],
				Y:     bot.Y + offset[1],
			})
		}
		// Visual: brighter wave ring
		if ss.ShowMsgWaves {
			ss.MsgWaves = append(ss.MsgWaves, swarm.MsgWave{
				X: bot.X, Y: bot.Y, Radius: 5, Timer: 45, Value: msgVal,
			})
		}

	case swarmscript.ActReverse:
		// Turn 180° and move forward
		bot.Angle += math.Pi
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActBrake:
		// Initiate braking: speed ramps down over 3 ticks
		bot.BrakeTimer = 3

	case swarmscript.ActScatterRandom:
		// Scatter away from neighbors with random perturbation
		if bot.NeighborCount > 0 {
			// Turn away from center of neighbors + random offset
			awayAngle := math.Atan2(-bot.AvgNeighborY, -bot.AvgNeighborX)
			awayAngle += (ss.Rng.Float64() - 0.5) * math.Pi / 2 // ±45° random
			bot.Angle = awayAngle
		} else {
			bot.Angle = ss.Rng.Float64() * 2 * math.Pi
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

	// Dash timer: double speed while active
	if bot.DashTimer > 0 {
		bot.DashTimer--
		bot.Speed *= 2.0
	}
	if bot.DashCooldown > 0 {
		bot.DashCooldown--
	}

	// Brake timer: reduce speed over 3 ticks
	if bot.BrakeTimer > 0 {
		bot.Speed *= float64(bot.BrakeTimer-1) / 3.0
		bot.BrakeTimer--
	}

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
			bot.CollisionTimer = 10 // trigger recent_collision sensor for 10 ticks
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
