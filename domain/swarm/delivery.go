package swarm

// UpdateDeliverySystem handles package respawn, carried package position, speed reduction.
func UpdateDeliverySystem(ss *SwarmState) {
	// Respawn timers on pickup stations
	for si := range ss.Stations {
		st := &ss.Stations[si]
		if st.FlashTimer > 0 {
			st.FlashTimer--
		}
		if st.IsPickup && !st.HasPackage && st.RespawnIn > 0 {
			st.RespawnIn--
			if st.RespawnIn <= 0 {
				// Spawn new package
				st.HasPackage = true
				ss.Packages = append(ss.Packages, DeliveryPackage{
					Color:     st.Color,
					CarriedBy: -1,
					X:         st.X,
					Y:         st.Y,
					Active:    true,
				})
			}
		}
	}

	// Update score popups (rise and fade)
	alive := 0
	for i := range ss.ScorePopups {
		ss.ScorePopups[i].Y -= 0.5
		ss.ScorePopups[i].Timer--
		if ss.ScorePopups[i].Timer > 0 {
			ss.ScorePopups[alive] = ss.ScorePopups[i]
			alive++
		}
	}
	ss.ScorePopups = ss.ScorePopups[:alive]

	// Update carried package positions + speed reduction
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			pkg := &ss.Packages[bot.CarryingPkg]
			pkg.X = bot.X
			pkg.Y = bot.Y
			// Slow down bot carrying a package
			if bot.Speed > 0 {
				bot.Speed *= 0.7
			}
		}
	}
}
