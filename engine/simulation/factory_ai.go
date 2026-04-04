package simulation

import "swarmsim/domain/factory"

// updateFactoryMode advances the factory simulation by one or more ticks
// depending on the Speed setting.
func (s *Simulation) updateFactoryMode() {
	fs := s.FactoryState
	if fs == nil || fs.Paused {
		return
	}

	// Run multiple ticks per frame when speed > 1
	ticks := int(fs.Speed)
	if ticks < 1 {
		ticks = 1
	}
	if ticks > 50 {
		ticks = 50 // cap to prevent freezing
	}

	for t := 0; t < ticks; t++ {
		fs.Tick++

		// Phase 0: Rebuild spatial hash for bot-bot collision avoidance
		if fs.BotHash != nil {
			fs.BotHash.Clear()
			for i := range fs.Bots {
				fs.BotHash.Insert(i, fs.Bots[i].X, fs.Bots[i].Y)
			}
		}

		// Phase 0.5: Emergency system tick
		factory.TickEmergency(fs)

		// Phase 0.6: Shift system tick
		factory.TickShiftSystem(fs)

		// Phase 1: Truck arrivals/departures (paused during emergency)
		if !fs.Emergency {
			truckCountBefore := len(fs.Trucks)
			factory.TickTrucks(fs)
			if len(fs.Trucks) > truckCountBefore {
				fs.TruckArriveFlash = factory.TruckArriveFlash{Tick: 0}
			}
		}

		// Phase 2: Generate tasks from structures (rate-limited internally)
		if !fs.Emergency {
			factory.GenerateTasks(fs)
		}

		// Phase 2.5: Prune stale/invalid tasks (every 100 ticks)
		factory.PruneStaleTasks(fs)

		// Phase 3: Assign tasks to idle bots (50 per tick max)
		factory.AssignIdleBots(fs, 50)

		// Phase 3.5: Pre-compute parking slot assignments (O(n) once, replaces O(n²) in botWander)
		factory.PrecomputeParkingSlots(fs)

		// Phase 4: Execute bot behavior (navigate to target, pickup/deliver)
		factory.TickBotBehavior(fs)

		// Phase 5: Machine processing — detect completions for FX
		for mi := range fs.Machines {
			m := &fs.Machines[mi]
			wasActive := m.Active && m.ProcessTimer == 1 // about to finish
			if wasActive {
				fs.MachineFinishFX = append(fs.MachineFinishFX, factory.MachineFinishEffect{
					X: m.X + m.W/2, Y: m.Y + m.H/2,
				})
			}
		}
		factory.TickMachines(fs)

		// Phase 6: Energy & charging
		factory.TickCharging(fs)

		// Phase 6.5: Order system tick
		if !fs.Emergency {
			factory.TickOrders(fs)
		}

		// Phase 7: Repair/malfunctions (staggered internally)
		// Detect new malfunctions for spark effects
		oldMalf := make([]bool, len(fs.Malfunctioning))
		copy(oldMalf, fs.Malfunctioning)
		factory.TickRepair(fs)
		for i := range fs.Malfunctioning {
			if i < len(oldMalf) && fs.Malfunctioning[i] && !oldMalf[i] && i < len(fs.Bots) {
				bot := &fs.Bots[i]
				spark := factory.SparkEffect{X: bot.X, Y: bot.Y}
				for s := 0; s < 4; s++ {
					spark.VX[s] = (fs.Rng.Float64() - 0.5) * 4
					spark.VY[s] = (fs.Rng.Float64() - 0.5) * 4
				}
				fs.SparkEffects = append(fs.SparkEffects, spark)
			}
		}

		// Phase 7.5: Random events system
		factory.TickEvents(fs)

		// Phase 8: Weather system
		fs.WeatherTimer--
		if fs.WeatherTimer <= 0 {
			if fs.Weather == factory.WeatherClear {
				fs.Weather = factory.WeatherRain
				fs.WeatherTimer = 1500 + fs.Rng.Intn(2000)
			} else {
				fs.Weather = factory.WeatherClear
				fs.WeatherTimer = 3000 + fs.Rng.Intn(4000)
			}
		}

		// Phase 9: Heatmap accumulation (every 5 ticks to save CPU)
		if fs.Tick%5 == 0 && fs.HeatmapW > 0 && fs.HeatmapH > 0 {
			cellW := factory.WorldW / float64(fs.HeatmapW)
			cellH := factory.WorldH / float64(fs.HeatmapH)
			for i := range fs.Bots {
				gx := int(fs.Bots[i].X / cellW)
				gy := int(fs.Bots[i].Y / cellH)
				if gx < 0 {
					gx = 0
				}
				if gx >= fs.HeatmapW {
					gx = fs.HeatmapW - 1
				}
				if gy < 0 {
					gy = 0
				}
				if gy >= fs.HeatmapH {
					gy = fs.HeatmapH - 1
				}
				idx := gy*fs.HeatmapW + gx
				if idx < len(fs.HeatmapGrid) {
					fs.HeatmapGrid[idx]++
				}
			}
		}

		// Phase 10: Throughput sampling (every 100 ticks)
		if fs.Tick%100 == 0 && len(fs.ThroughputHistory) > 0 {
			delta := fs.Stats.PartsProcessed - fs.LastPartsProcessed
			fs.LastPartsProcessed = fs.Stats.PartsProcessed
			fs.ThroughputHistory[fs.ThroughputIdx%len(fs.ThroughputHistory)] = delta
			fs.ThroughputIdx++
		}

		// Phase 11: Selected bot path tracking
		if fs.SelectedBot >= 0 && fs.SelectedBot < len(fs.Bots) && len(fs.SelectedBotPath) > 0 {
			bot := &fs.Bots[fs.SelectedBot]
			idx := fs.SelectedBotPathIdx % len(fs.SelectedBotPath)
			fs.SelectedBotPath[idx] = [2]float64{bot.X, bot.Y}
			fs.SelectedBotPathIdx++
		}

		// Phase 12: Update stats (only on last tick of batch)
		if t == ticks-1 {
			factory.UpdateStats(fs)
		}

		// Phase 13: Efficiency sparkline sampling
		factory.TickEfficiencySample(fs)

		// Phase 14: Achievement checks
		factory.TickAchievements(fs)
	}

	// Advance truck arrive flash (per frame, not per tick)
	fs.TruckArriveFlash.Tick++
}
