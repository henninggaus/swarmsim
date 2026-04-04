package simulation

import (
	"math"
	"swarmsim/domain/physics"
	"swarmsim/domain/swarm"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
	"time"
)

// updateSwarmMode is the main update loop for programmable swarm mode.
func (s *Simulation) updateSwarmMode() {
	tickStart := time.Now()
	defer func() {
		s.SwarmState.LastTickDuration = time.Since(tickStart).Seconds()
	}()

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
				if bot.CarryingPkg < 0 {
					if ss.TruckToggle && bot.OnRamp {
						// Truck mode: pickup at ramp via crane
						executeSwarmAction(swarmscript.Action{Type: swarmscript.ActPickup}, bot, ss, i)
					} else if bot.NearestPickupDist < 20 && bot.NearestPickupHasPkg {
						// Delivery mode: pickup at station
						executeSwarmAction(swarmscript.Action{Type: swarmscript.ActPickup}, bot, ss, i)
					}
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

	// Phase 4.93e2: Collective AI -- problem detection + code generation + testing
	if ss.CollectiveAIOn && ss.IssueBoard != nil && ss.Tick%10 == 0 {
		swarm.DetectBotProblems(ss)
		swarm.ProcessOpenIssues(ss)
		swarm.EvaluateTestingIssues(ss)
	}

	// Phase 4.93f: Learning system tick (lesson auto-advance)
	if ss.Learning != nil && ss.Learning.Active {
		swarm.TickLesson(ss.Learning)
	}

	// Phase 4.93g: Emergence detection (every 60 ticks, needs PatternResult)
	if ss.PatternResult != nil && ss.Tick%60 == 0 {
		popup := swarm.DetectEmergence(ss)
		if popup != nil && ss.EmergencePopup == nil {
			ss.EmergencePopup = popup
		}
	}
	swarm.TickEmergencePopup(ss)

	// Phase 4.93h: Also run pattern detection for learning/emergence if needed
	if ss.PatternResult == nil && (ss.Learning != nil && ss.Learning.Active) && ss.Tick%30 == 0 {
		pr := swarm.DetectPatterns(ss)
		ss.PatternResult = &pr
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
