package simulation

import (
	"fmt"
	"math"
	"swarmsim/domain/bot"
	"swarmsim/domain/comm"
	"swarmsim/domain/physics"
	"swarmsim/engine/pheromone"
)

// updateTruckMode is the main update loop for truck unloading mode.
// It replaces the standard bot update loop when TruckMode is active.
func (s *Simulation) updateTruckMode() {
	ts := s.TruckState
	if ts.Completed {
		return
	}
	ts.Timer++

	// Periodic debug info
	if ts.Timer%90 == 1 {
		accessible := ts.AccessiblePackages()
		inTruck := 0
		for _, p := range ts.Packages {
			if p.State == PkgInTruck {
				inTruck++
			}
		}
		fmt.Printf("[TRUCK] Timer=%d InTruck=%d Accessible=%d Delivered=%d/%d Score=%d\n",
			ts.Timer, inTruck, len(accessible), ts.DeliveredPkgs, ts.TotalPkgs, s.Score)
	}

	// Rebuild spatial hash
	s.Hash.Clear()
	for _, b := range s.Bots {
		if b.IsAlive() {
			pos := b.Position()
			s.Hash.Insert(b.ID(), pos.X, pos.Y)
		}
	}

	eCfg := bot.EnergyCfg{
		MoveCost:  s.Cfg.EnergyMoveCost,
		MsgCost:   s.Cfg.EnergyMsgCost,
		CarryCost: s.Cfg.EnergyCarryCost,
		PherCost:  s.Cfg.EnergyPherCost,
		TankPush:  s.Cfg.EnergyTankPush,
		DecayMult: s.Cfg.EnergyDecayMult,
	}

	// Build bot position map for ramp congestion check
	botPositions := make(map[int][2]float64)
	for _, b := range s.Bots {
		if b.IsAlive() {
			pos := b.Position()
			botPositions[b.ID()] = [2]float64{pos.X, pos.Y}
		}
	}

	for _, b := range s.Bots {
		if !b.IsAlive() {
			continue
		}
		base := b.GetBase()
		pos := b.Position()

		// Energy check
		if !base.HasEnergy() {
			base.State = bot.StateNoEnergy
			base.FitZeroEnergyTicks++
			base.Vel = bot.Vec2{}
			// Steer to nearest charger when possible
			task := ts.GetBotTask(b.ID())
			task.Type = TaskRecharging
			continue
		}

		// Build nearby bots list
		maxRange := b.GetSensorRange()
		if b.GetCommRange() > maxRange {
			maxRange = b.GetCommRange()
		}
		nearbyIDs := s.Hash.Query(pos.X, pos.Y, maxRange)
		var nearby []bot.Bot
		for _, nid := range nearbyIDs {
			if nid != b.ID() && nid < len(s.Bots) && s.Bots[nid].IsAlive() {
				if pos.Dist(s.Bots[nid].Position()) <= b.GetSensorRange() {
					nearby = append(nearby, s.Bots[nid])
				}
			}
		}

		inbox := s.Channel.Deliver(pos.X, pos.Y)

		task := ts.GetBotTask(b.ID())

		// Dispatch to bot-type-specific AI
		var outbox []comm.Message
		switch b.Type() {
		case bot.TypeScout:
			outbox = s.truckAI_Scout(b, task, nearby, inbox, eCfg)
		case bot.TypeWorker:
			outbox = s.truckAI_Worker(b, task, nearby, inbox, eCfg)
		case bot.TypeTank:
			outbox = s.truckAI_Tank(b, task, nearby, inbox, eCfg)
		case bot.TypeLeader:
			outbox = s.truckAI_Leader(b, task, nearby, inbox, eCfg, botPositions)
		case bot.TypeHealer:
			outbox = s.truckAI_Healer(b, task, nearby, inbox, eCfg)
		}

		// Send messages
		for _, msg := range outbox {
			s.Channel.Send(msg, pos.X, pos.Y, b.GetCommRange())
			s.TotalMsgsSent++
		}

		// Ramp extra energy cost
		if ts.IsInRamp(base.Pos.X, base.Pos.Y) {
			base.ConsumeEnergy(s.Cfg.TruckRampExtraCost, eCfg.DecayMult)
		}

		// Charging station recharge
		for _, c := range ts.Depot.Chargers {
			dx := c.X - base.Pos.X
			dy := c.Y - base.Pos.Y
			if math.Sqrt(dx*dx+dy*dy) < s.Cfg.TruckChargeRange {
				base.RechargeEnergy(s.Cfg.TruckChargeRate)
				break
			}
		}

		// Resolve physics (boundaries only, no arena obstacles in truck mode)
		s.resolveTruckPhysics(b)
	}

	// Resolve cooperative lifts
	s.resolveCoopLifts()

	// Resolve deliveries (packages entering zones)
	s.resolveTruckDeliveries()

	// Update pheromones and channel
	s.Pheromones.Update()
	s.ActiveMsgs = s.Channel.Tick()

	// Check completion
	if ts.DeliveredPkgs >= ts.TotalPkgs && ts.TotalPkgs > 0 {
		ts.Completed = true
	}

	// Auto-evolve check
	if s.Cfg.AutoEvolve && s.GenerationTick >= s.Cfg.GenerationLength {
		s.EndGeneration()
	}
}

// resolveTruckPhysics handles boundary clamping and truck wall collisions.
func (s *Simulation) resolveTruckPhysics(b bot.Bot) {
	base := b.GetBase()
	r := base.Radius

	// Arena boundary clamping
	nx, ny, hit := physics.ClampToBounds(base.Pos.X, base.Pos.Y, r, s.Cfg.ArenaWidth, s.Cfg.ArenaHeight)
	if hit {
		base.Vel.X, base.Vel.Y = physics.ReflectVelocity(base.Pos.X, base.Pos.Y, base.Vel.X, base.Vel.Y, r, s.Cfg.ArenaWidth, s.Cfg.ArenaHeight)
		base.Pos.X = nx
		base.Pos.Y = ny
	}

	ts := s.TruckState
	truck := ts.Truck

	// Truck cargo walls - left wall
	collides, _, _ := physics.CircleRectCollision(base.Pos.X, base.Pos.Y, r, truck.CargoX-10, truck.CargoY, 10, truck.CargoH)
	if collides {
		base.Pos.X, base.Pos.Y = physics.ResolveCircleRectOverlap(base.Pos.X, base.Pos.Y, r, truck.CargoX-10, truck.CargoY, 10, truck.CargoH)
		base.Vel = base.Vel.Scale(0.3)
	}

	// Truck cargo walls - top wall (shortened to leave opening clear)
	// Wall stops 100px before the opening so bots can enter from the ramp
	cargoWallW := truck.CargoW - 100
	collides, _, _ = physics.CircleRectCollision(base.Pos.X, base.Pos.Y, r, truck.CargoX, truck.CargoY-10, cargoWallW, 10)
	if collides {
		base.Pos.X, base.Pos.Y = physics.ResolveCircleRectOverlap(base.Pos.X, base.Pos.Y, r, truck.CargoX, truck.CargoY-10, cargoWallW, 10)
		base.Vel = base.Vel.Scale(0.3)
	}

	// Truck cargo walls - bottom wall (shortened to leave opening clear)
	collides, _, _ = physics.CircleRectCollision(base.Pos.X, base.Pos.Y, r, truck.CargoX, truck.CargoY+truck.CargoH, cargoWallW, 10)
	if collides {
		base.Pos.X, base.Pos.Y = physics.ResolveCircleRectOverlap(base.Pos.X, base.Pos.Y, r, truck.CargoX, truck.CargoY+truck.CargoH, cargoWallW, 10)
		base.Vel = base.Vel.Scale(0.3)
	}

	// Cabin wall (blocks entrance from left)
	collides, _, _ = physics.CircleRectCollision(base.Pos.X, base.Pos.Y, r, truck.CabinX, truck.CabinY, truck.CabinW, truck.CabinH)
	if collides {
		base.Pos.X, base.Pos.Y = physics.ResolveCircleRectOverlap(base.Pos.X, base.Pos.Y, r, truck.CabinX, truck.CabinY, truck.CabinW, truck.CabinH)
		base.Vel = base.Vel.Scale(0.3)
	}

	// Ramp top edge (above ramp opening)
	rampTopWallY := truck.RampY - 10
	if base.Pos.X >= truck.RampX && base.Pos.X <= truck.RampX+truck.RampW {
		collides, _, _ = physics.CircleRectCollision(base.Pos.X, base.Pos.Y, r, truck.RampX, rampTopWallY, truck.RampW, 10)
		if collides {
			base.Pos.X, base.Pos.Y = physics.ResolveCircleRectOverlap(base.Pos.X, base.Pos.Y, r, truck.RampX, rampTopWallY, truck.RampW, 10)
			base.Vel = base.Vel.Scale(0.3)
		}
	}

	// Ramp bottom edge (below ramp opening)
	rampBotWallY := truck.RampY + truck.RampH
	if base.Pos.X >= truck.RampX && base.Pos.X <= truck.RampX+truck.RampW {
		collides, _, _ = physics.CircleRectCollision(base.Pos.X, base.Pos.Y, r, truck.RampX, rampBotWallY, truck.RampW, 10)
		if collides {
			base.Pos.X, base.Pos.Y = physics.ResolveCircleRectOverlap(base.Pos.X, base.Pos.Y, r, truck.RampX, rampBotWallY, truck.RampW, 10)
			base.Vel = base.Vel.Scale(0.3)
		}
	}
}

// --- Per-bot-type AI functions ---

func (s *Simulation) truckAI_Scout(b bot.Bot, task *BotTask, nearby []bot.Bot, inbox []comm.Message, eCfg bot.EnergyCfg) []comm.Message {
	base := b.GetBase()
	ts := s.TruckState
	var outbox []comm.Message

	// Should recharge?
	if base.ShouldReturnForEnergy() {
		task.Type = TaskRecharging
	}

	if task.Type == TaskRecharging {
		cx, cy := ts.NearestCharger(base.Pos.X, base.Pos.Y)
		steer := base.SteerToward(bot.Vec2{X: cx, Y: cy}, 0.5)
		base.Vel = base.Vel.Add(steer)
		base.ApplyVelocity(eCfg)
		base.State = bot.StateNoEnergy
		if base.Energy > base.MaxEnergy*0.8 {
			task.Type = TaskScanning
		}
		return outbox
	}

	task.Type = TaskScanning
	base.State = bot.StateScouting

	// Move into cargo area with a scanning pattern
	cargoCenter := bot.Vec2{
		X: ts.Truck.CargoX + ts.Truck.CargoW*0.5,
		Y: ts.Truck.CargoY + ts.Truck.CargoH*0.5,
	}

	// Scan across cargo, oscillating vertically
	scanX := ts.Truck.CargoX + ts.Truck.CargoW*0.3 + float64(task.SubState%200)*1.5
	if scanX > ts.Truck.CargoX+ts.Truck.CargoW-30 {
		task.SubState = 0
		scanX = ts.Truck.CargoX + ts.Truck.CargoW*0.3
	}
	scanY := cargoCenter.Y + math.Sin(float64(task.SubState)*0.05)*ts.Truck.CargoH*0.3
	task.SubState++

	target := bot.Vec2{X: scanX, Y: scanY}
	steer := base.SteerToward(target, 0.3)
	sep := truckSeparation(base, nearby, 25)
	base.Vel = base.Vel.Add(steer).Add(sep.Scale(1.0))
	base.ApplyVelocity(eCfg)

	// Broadcast accessible packages in sensor range
	if base.ShouldCommunicate(s.Tick) {
		accessible := ts.AccessiblePackages()
		for _, pkg := range accessible {
			pkgPos := bot.Vec2{X: pkg.X, Y: pkg.Y}
			if base.Pos.Dist(pkgPos) <= base.SensorRange {
				outbox = append(outbox, comm.NewPackageFound(base.BotID, pkg.X, pkg.Y, pkg.ID))
				base.ConsumeEnergy(eCfg.MsgCost, eCfg.DecayMult)
				base.FitMessagesRelayed++

				// Deposit pheromone near packages
				if s.Pheromones != nil {
					base.DepositPheromone(s.Pheromones, pheromone.PherFoundResource, 0.2, eCfg)
				}
			}
		}
	}

	return outbox
}

func (s *Simulation) truckAI_Worker(b bot.Bot, task *BotTask, nearby []bot.Bot, inbox []comm.Message, eCfg bot.EnergyCfg) []comm.Message {
	return s.truckAI_Carrier(b, task, nearby, inbox, eCfg)
}

func (s *Simulation) truckAI_Tank(b bot.Bot, task *BotTask, nearby []bot.Bot, inbox []comm.Message, eCfg bot.EnergyCfg) []comm.Message {
	return s.truckAI_Carrier(b, task, nearby, inbox, eCfg)
}

// truckAI_Carrier handles the shared pickup/carry/deliver logic for Workers and Tanks.
func (s *Simulation) truckAI_Carrier(b bot.Bot, task *BotTask, nearby []bot.Bot, inbox []comm.Message, eCfg bot.EnergyCfg) []comm.Message {
	base := b.GetBase()
	ts := s.TruckState
	var outbox []comm.Message
	carryCap := ts.CarryCaps[b.ID()]

	// Pickup distance scales with bot radius so bots can reach packages
	pickupDist := base.Radius + 15.0

	// Process inbox - respond to coop help requests
	for _, msg := range inbox {
		if msg.Type == comm.MsgNeedCoopHelp && task.Type == TaskIdle {
			pkg := ts.GetPackageByID(msg.ExtraID)
			if pkg != nil && (pkg.State == PkgInTruck || pkg.State == PkgLifting) {
				task.Type = TaskGoToPackage
				task.TargetPkgID = pkg.ID
				task.TargetX = pkg.X
				task.TargetY = pkg.Y
			}
		}
		if msg.Type == comm.MsgTaskAssign && task.Type == TaskIdle {
			task.Type = TaskGoToPackage
			task.TargetPkgID = msg.ExtraID
			task.TargetX = msg.X
			task.TargetY = msg.Y
		}
		if msg.Type == comm.MsgPackageFound && task.Type == TaskIdle {
			pkg := ts.GetPackageByID(msg.ExtraID)
			if pkg != nil && pkg.State == PkgInTruck && carryCap >= pkg.Def.MinCarryCap {
				task.Type = TaskGoToPackage
				task.TargetPkgID = pkg.ID
				task.TargetX = pkg.X
				task.TargetY = pkg.Y
			}
		}
	}

	// Should recharge? Only interrupt if energy is critically low (not during active tasks)
	if base.Energy < 5 && task.Type != TaskCarrying && task.Type != TaskLifting {
		task.Type = TaskRecharging
	}

	switch task.Type {
	case TaskIdle:
		base.State = bot.StateIdle
		// Find an accessible package to pick up
		accessible := ts.AccessiblePackages()
		bestDist := 1e9
		var bestPkg *Package

		// First pass: prefer packages this bot can carry alone
		for _, pkg := range accessible {
			if carryCap >= pkg.Def.MinCarryCap {
				pkgPos := bot.Vec2{X: pkg.X, Y: pkg.Y}
				dist := base.Pos.Dist(pkgPos)
				if dist < bestDist {
					bestDist = dist
					bestPkg = pkg
				}
			}
		}

		// Second pass: if nothing soloable, consider heavy packages for coop
		if bestPkg == nil {
			bestDist = 1e9
			for _, pkg := range accessible {
				if carryCap > 0 && carryCap < pkg.Def.MinCarryCap {
					pkgPos := bot.Vec2{X: pkg.X, Y: pkg.Y}
					dist := base.Pos.Dist(pkgPos)
					if dist < bestDist {
						bestDist = dist
						bestPkg = pkg
					}
				}
			}
		}

		if bestPkg != nil {
			task.Type = TaskGoToPackage
			task.TargetPkgID = bestPkg.ID
			task.TargetX = bestPkg.X
			task.TargetY = bestPkg.Y
			fmt.Printf("[TRUCK] Bot %d (%s, cap=%.1f) → GoToPackage #%d (%s, minCap=%.1f) dist=%.0f\n",
				b.ID(), b.Type(), carryCap, bestPkg.ID, bestPkg.Def.Name, bestPkg.Def.MinCarryCap, bestDist)
		} else {
			// Wander near ramp entrance
			target := bot.Vec2{X: s.TruckState.Truck.RampX + 150, Y: s.TruckState.Truck.RampY + s.TruckState.Truck.RampH/2}
			steer := base.SteerToward(target, 0.2)
			sep := truckSeparation(base, nearby, 30)
			base.Vel = base.Vel.Add(steer).Add(sep.Scale(1.0))
			base.ApplyVelocity(eCfg)
			if s.Tick%60 == 0 {
				fmt.Printf("[TRUCK] Bot %d (%s) idle, accessible=%d total=%d\n",
					b.ID(), b.Type(), len(accessible), len(ts.Packages))
			}
		}

	case TaskGoToPackage:
		base.State = bot.StateForaging
		pkg := ts.GetPackageByID(task.TargetPkgID)
		if pkg == nil || pkg.State == PkgDelivered || pkg.State == PkgCarried {
			task.Type = TaskIdle
			task.TargetPkgID = -1
			return outbox
		}

		pkgPos := bot.Vec2{X: pkg.X, Y: pkg.Y}
		dist := base.Pos.Dist(pkgPos)

		if s.Tick%30 == 0 {
			fmt.Printf("[TRUCK] Bot %d → pkg #%d dist=%.1f (need <%.1f) pos=(%.0f,%.0f) pkg=(%.0f,%.0f) energy=%.0f\n",
				b.ID(), task.TargetPkgID, dist, pickupDist, base.Pos.X, base.Pos.Y, pkg.X, pkg.Y, base.Energy)
		}

		if dist < pickupDist {
			fmt.Printf("[TRUCK] Bot %d PICKUP ATTEMPT pkg #%d (%s) dist=%.1f carryCap=%.1f minCap=%.1f\n",
				b.ID(), pkg.ID, pkg.Def.Name, dist, carryCap, pkg.Def.MinCarryCap)
			// Attempt to lift
			if carryCap >= pkg.Def.MinCarryCap {
				// Can lift alone
				fmt.Printf("[TRUCK] Bot %d LIFTING pkg #%d solo!\n", b.ID(), pkg.ID)
				task.Type = TaskLifting
				task.SubState = s.Cfg.TruckLiftTicks
				pkg.State = PkgLifting
				pkg.CarrierBotIDs = []int{b.ID()}
				base.State = bot.StateLifting
			} else {
				// Need help
				fmt.Printf("[TRUCK] Bot %d needs COOP help for pkg #%d\n", b.ID(), pkg.ID)
				task.Type = TaskWaitingHelp
				pkg.State = PkgLifting
				if pkg.CarrierBotIDs == nil {
					pkg.CarrierBotIDs = []int{b.ID()}
				} else {
					// Add this bot if not already in list
					found := false
					for _, id := range pkg.CarrierBotIDs {
						if id == b.ID() {
							found = true
							break
						}
					}
					if !found {
						pkg.CarrierBotIDs = append(pkg.CarrierBotIDs, b.ID())
					}
				}
				base.State = bot.StateWaitingHelp
				if base.ShouldCommunicate(s.Tick) {
					outbox = append(outbox, comm.NewNeedCoopHelp(base.BotID, pkg.X, pkg.Y, pkg.ID))
					base.ConsumeEnergy(eCfg.MsgCost, eCfg.DecayMult)
				}
			}
		} else {
			// Navigate to package
			steer := base.SteerToward(pkgPos, 0.5)
			sep := truckSeparation(base, nearby, 25)
			base.Vel = base.Vel.Add(steer).Add(sep.Scale(1.0))
			base.ApplyVelocity(eCfg)
		}

	case TaskWaitingHelp:
		base.State = bot.StateWaitingHelp
		pkg := ts.GetPackageByID(task.TargetPkgID)
		if pkg == nil || pkg.State == PkgDelivered || pkg.State == PkgCarried {
			task.Type = TaskIdle
			task.TargetPkgID = -1
			return outbox
		}
		// Stay near package
		pkgPos := bot.Vec2{X: pkg.X, Y: pkg.Y}
		if base.Pos.Dist(pkgPos) > pickupDist {
			steer := base.SteerToward(pkgPos, 0.3)
			base.Vel = steer.Scale(0.3)
			base.ApplyVelocity(eCfg)
		} else {
			base.Vel = bot.Vec2{}
		}
		// Broadcast need for help
		if base.ShouldCommunicate(s.Tick) {
			outbox = append(outbox, comm.NewNeedCoopHelp(base.BotID, pkg.X, pkg.Y, pkg.ID))
			base.ConsumeEnergy(eCfg.MsgCost, eCfg.DecayMult)
		}

	case TaskLifting:
		base.State = bot.StateLifting
		pkg := ts.GetPackageByID(task.TargetPkgID)
		if pkg == nil {
			task.Type = TaskIdle
			return outbox
		}
		task.SubState--
		// Consume lifting energy spread over ticks
		base.ConsumeEnergy(s.Cfg.TruckLiftEnergy/float64(s.Cfg.TruckLiftTicks), eCfg.DecayMult)
		base.Vel = bot.Vec2{}

		if task.SubState <= 0 {
			// Lift complete - start carrying
			pkg.State = PkgCarried
			task.Type = TaskCarrying
			base.State = bot.StateCarryingPkg
			base.FitResourcesCollected++

			// Credit all cooperating bots
			for _, cid := range pkg.CarrierBotIDs {
				if cid != b.ID() {
					cBot := s.GetBotByID(cid)
					if cBot != nil {
						cBase := cBot.GetBase()
						cBase.FitResourcesCollected++
						cTask := ts.GetBotTask(cid)
						cTask.Type = TaskCarrying
						cTask.TargetPkgID = pkg.ID
					}
				}
			}

			// Determine target zone
			zx, zy := ts.ZoneCenter(pkg.Def.Zone)
			task.TargetX = zx
			task.TargetY = zy

			// Emit pickup event for particles
			s.CoopPickupEvents = append(s.CoopPickupEvents, CoopPickupEvent{
				X: pkg.X, Y: pkg.Y, Tick: s.Tick,
			})
		}

	case TaskCarrying:
		base.State = bot.StateCarryingPkg
		pkg := ts.GetPackageByID(task.TargetPkgID)
		if pkg == nil || pkg.State == PkgDelivered {
			task.Type = TaskIdle
			task.TargetPkgID = -1
			return outbox
		}

		// Carrying energy cost
		base.ConsumeEnergy(s.Cfg.TruckCarryPerWeight*pkg.Def.Weight, eCfg.DecayMult)

		// Steer toward target zone
		zx, zy := ts.ZoneCenter(pkg.Def.Zone)
		target := bot.Vec2{X: zx, Y: zy}
		steer := base.SteerToward(target, 0.5)
		sep := truckSeparation(base, nearby, 30)
		base.Vel = base.Vel.Add(steer).Add(sep.Scale(0.8))

		// Slow down when carrying
		maxCarrySpeed := base.MaxSpeed * 0.7
		if base.Vel.Len() > maxCarrySpeed {
			base.Vel = base.Vel.Normalized().Scale(maxCarrySpeed)
		}
		base.ApplyVelocity(eCfg)

		// Move package with bot (centroid of carriers if cooperative)
		if len(pkg.CarrierBotIDs) > 1 {
			var cx, cy float64
			count := 0
			for _, cid := range pkg.CarrierBotIDs {
				cBot := s.GetBotByID(cid)
				if cBot != nil && cBot.IsAlive() {
					cPos := cBot.Position()
					cx += cPos.X
					cy += cPos.Y
					count++
				}
			}
			if count > 0 {
				pkg.X = cx / float64(count)
				pkg.Y = cy / float64(count)
			}
		} else {
			pkg.X = base.Pos.X
			pkg.Y = base.Pos.Y
		}

	case TaskRecharging:
		base.State = bot.StateNoEnergy
		cx, cy := ts.NearestCharger(base.Pos.X, base.Pos.Y)
		target := bot.Vec2{X: cx, Y: cy}
		steer := base.SteerToward(target, 0.5)
		base.Vel = base.Vel.Add(steer)
		base.ApplyVelocity(eCfg)
		if base.Energy > base.MaxEnergy*0.8 {
			task.Type = TaskIdle
		}
	}

	return outbox
}

func (s *Simulation) truckAI_Leader(b bot.Bot, task *BotTask, nearby []bot.Bot, inbox []comm.Message, eCfg bot.EnergyCfg, botPositions map[int][2]float64) []comm.Message {
	base := b.GetBase()
	ts := s.TruckState
	var outbox []comm.Message

	// Should recharge?
	if base.ShouldReturnForEnergy() {
		cx, cy := ts.NearestCharger(base.Pos.X, base.Pos.Y)
		steer := base.SteerToward(bot.Vec2{X: cx, Y: cy}, 0.5)
		base.Vel = base.Vel.Add(steer)
		base.ApplyVelocity(eCfg)
		base.State = bot.StateNoEnergy
		return outbox
	}

	task.Type = TaskCoordinating
	base.State = bot.StateCoordinating

	// Position near ramp entrance
	rampEntrance := bot.Vec2{
		X: ts.Truck.RampX + ts.Truck.RampW + 30,
		Y: ts.Truck.RampY + ts.Truck.RampH/2,
	}
	if base.Pos.Dist(rampEntrance) > 50 {
		steer := base.SteerToward(rampEntrance, 0.3)
		base.Vel = base.Vel.Add(steer)
		base.ApplyVelocity(eCfg)
	} else {
		base.Vel = bot.Vec2{}
	}

	if !base.ShouldCommunicate(s.Tick) {
		return outbox
	}

	// Monitor ramp congestion
	rampCount := ts.CountBotsOnRamp(botPositions)
	if rampCount > 3 {
		// Deposit danger pheromone at ramp entrance
		if s.Pheromones != nil {
			base.DepositPheromone(s.Pheromones, pheromone.PherDanger, 0.5, eCfg)
		}
		outbox = append(outbox, comm.NewRampCongested(base.BotID, rampEntrance.X, rampEntrance.Y))
		base.ConsumeEnergy(eCfg.MsgCost, eCfg.DecayMult)
	}

	// Relay all messages with extended comm range
	for _, msg := range inbox {
		outbox = append(outbox, msg)
		base.FitMessagesRelayed++
	}

	// Assign tasks to idle workers
	accessible := ts.AccessiblePackages()
	if len(accessible) > 0 {
		for _, other := range nearby {
			if other.Type() == bot.TypeWorker || other.Type() == bot.TypeTank {
				otherTask := ts.GetBotTask(other.ID())
				if otherTask.Type == TaskIdle {
					// Find nearest unassigned package
					for _, pkg := range accessible {
						outbox = append(outbox, comm.NewTaskAssign(base.BotID, pkg.X, pkg.Y, pkg.ID))
						base.ConsumeEnergy(eCfg.MsgCost, eCfg.DecayMult)
						break
					}
				}
			}
		}
	}

	return outbox
}

func (s *Simulation) truckAI_Healer(b bot.Bot, task *BotTask, nearby []bot.Bot, inbox []comm.Message, eCfg bot.EnergyCfg) []comm.Message {
	base := b.GetBase()
	ts := s.TruckState
	var outbox []comm.Message

	// Should recharge self?
	if base.ShouldReturnForEnergy() {
		cx, cy := ts.NearestCharger(base.Pos.X, base.Pos.Y)
		steer := base.SteerToward(bot.Vec2{X: cx, Y: cy}, 0.5)
		base.Vel = base.Vel.Add(steer)
		base.ApplyVelocity(eCfg)
		base.State = bot.StateNoEnergy
		return outbox
	}

	task.Type = TaskHealing
	base.State = bot.StateRepairing

	// Find the nearest bot with low energy
	var healTarget bot.Bot
	bestNeed := 0.0
	for _, other := range nearby {
		if other.ID() == b.ID() {
			continue
		}
		oBase := other.GetBase()
		energyNeed := 1.0 - (oBase.Energy / oBase.MaxEnergy)
		if energyNeed > 0.3 && energyNeed > bestNeed {
			bestNeed = energyNeed
			healTarget = other
		}
	}

	if healTarget != nil {
		targetPos := healTarget.Position()
		dist := base.Pos.Dist(targetPos)

		if dist < 20 {
			// Heal the target
			targetBase := healTarget.GetBase()
			targetBase.RechargeEnergy(s.Cfg.EnergyHealGive)
			base.ConsumeEnergy(s.Cfg.EnergyHealCost, eCfg.DecayMult)
			base.FitBotsHealed++
			base.Vel = bot.Vec2{}
		} else {
			// Move toward target
			steer := base.SteerToward(targetPos, 0.5)
			sep := truckSeparation(base, nearby, 20)
			base.Vel = base.Vel.Add(steer).Add(sep.Scale(0.5))
			base.ApplyVelocity(eCfg)
		}
	} else {
		// Position near ramp/depot
		depotCenter := bot.Vec2{X: 1100, Y: 450}
		if base.Pos.Dist(depotCenter) > 100 {
			steer := base.SteerToward(depotCenter, 0.2)
			base.Vel = base.Vel.Add(steer)
			base.ApplyVelocity(eCfg)
		} else {
			base.Vel = bot.Vec2{}
		}
	}

	return outbox
}

// resolveCoopLifts checks packages in PkgLifting state and resolves cooperative lifts.
func (s *Simulation) resolveCoopLifts() {
	ts := s.TruckState

	for _, pkg := range ts.Packages {
		if pkg.State != PkgLifting || len(pkg.CarrierBotIDs) == 0 {
			continue
		}

		// Sum carry capacity of all bots at the package
		totalCap := 0.0
		var validBots []int
		for _, botID := range pkg.CarrierBotIDs {
			b := s.GetBotByID(botID)
			if b == nil || !b.IsAlive() {
				continue
			}
			botBase := b.GetBase()
			coopDist := botBase.Radius + 20.0
			dist := b.Position().Dist(bot.Vec2{X: pkg.X, Y: pkg.Y})
			if dist <= coopDist {
				totalCap += ts.CarryCaps[botID]
				validBots = append(validBots, botID)
			}
		}

		pkg.CarrierBotIDs = validBots

		if totalCap >= pkg.Def.MinCarryCap && len(validBots) >= 1 {
			// Check if any bot has already started the lift timer
			anyLifting := false
			for _, bid := range validBots {
				task := ts.GetBotTask(bid)
				if task.Type == TaskLifting {
					anyLifting = true
					break
				}
			}

			if !anyLifting {
				// Start cooperative lift for all bots
				for _, bid := range validBots {
					task := ts.GetBotTask(bid)
					task.Type = TaskLifting
					task.SubState = s.Cfg.TruckLiftTicks
					task.TargetPkgID = pkg.ID
					b := s.GetBotByID(bid)
					if b != nil {
						b.GetBase().State = bot.StateLifting
					}
				}
			}
		}
	}
}

// resolveTruckDeliveries checks if carried packages have entered a zone.
func (s *Simulation) resolveTruckDeliveries() {
	ts := s.TruckState

	for _, pkg := range ts.Packages {
		if pkg.State != PkgCarried {
			continue
		}

		zone, inZone := ts.FindZoneAt(pkg.X, pkg.Y)
		if !inZone {
			continue
		}

		// Package delivered to zone
		pkg.State = PkgDelivered
		pkg.Delivered = true
		pkg.DeliveredZone = zone
		pkg.CorrectZone = (zone == pkg.Def.Zone)
		fmt.Printf("[TRUCK] Package #%d (%s) DELIVERED to zone %s (correct=%v) at (%.0f,%.0f)\n",
			pkg.ID, pkg.Def.Name, zone, pkg.CorrectZone, pkg.X, pkg.Y)

		// Scoring
		basePoints := int(pkg.Def.Weight)
		if pkg.CorrectZone {
			s.Score += basePoints
			ts.CorrectZone++
		} else {
			s.Score += basePoints / 2
			ts.WrongZone++
		}
		ts.DeliveredPkgs++

		// Emit delivery event
		s.TruckDeliveryEvents = append(s.TruckDeliveryEvents, TruckDeliveryEvent{
			X: pkg.X, Y: pkg.Y, Tick: s.Tick, Correct: pkg.CorrectZone,
		})

		// Credit fitness to all carrying bots and free them
		for _, botID := range pkg.CarrierBotIDs {
			b := s.GetBotByID(botID)
			if b != nil {
				b.GetBase().FitResourcesDelivered++
				b.GetBase().State = bot.StateIdle
			}
			task := ts.GetBotTask(botID)
			task.Type = TaskIdle
			task.TargetPkgID = -1
		}
		pkg.CarrierBotIDs = nil
	}
}

// truckSeparation computes a separation steering vector from nearby bots.
func truckSeparation(self *bot.BaseBot, nearby []bot.Bot, radius float64) bot.Vec2 {
	var steer bot.Vec2
	count := 0
	for _, other := range nearby {
		if other.ID() == self.BotID || !other.IsAlive() {
			continue
		}
		diff := self.Pos.Sub(other.Position())
		dist := diff.Len()
		if dist > 0 && dist < radius {
			steer = steer.Add(diff.Normalized().Scale(1.0 / dist))
			count++
		}
	}
	if count > 0 {
		steer = steer.Scale(1.0 / float64(count))
	}
	return steer
}
