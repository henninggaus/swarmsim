package swarm

import (
	"math"
	"swarmsim/logger"
)

const (
	DynObstacleSpeed  = 0.3  // pixels per tick
	DynPackageExpiry  = 3000 // ticks until uncollected package expires
	DynExpiredFlash   = 30   // flash timer when package expires
)

// UpdateDynamicEnvironment moves obstacles and expires packages.
// Called every tick when DynamicEnv is true.
func UpdateDynamicEnvironment(ss *SwarmState) {
	// 1. Move obstacles
	for _, obs := range ss.Obstacles {
		if obs.VX == 0 && obs.VY == 0 {
			// Assign random velocity if not set
			angle := ss.Rng.Float64() * 2 * math.Pi
			obs.VX = math.Cos(angle) * DynObstacleSpeed
			obs.VY = math.Sin(angle) * DynObstacleSpeed
		}

		obs.X += obs.VX
		obs.Y += obs.VY

		// Bounce off arena edges
		if obs.X < 0 {
			obs.X = 0
			obs.VX = -obs.VX
		}
		if obs.Y < 0 {
			obs.Y = 0
			obs.VY = -obs.VY
		}
		if obs.X+obs.W > ss.ArenaW {
			obs.X = ss.ArenaW - obs.W
			obs.VX = -obs.VX
		}
		if obs.Y+obs.H > ss.ArenaH {
			obs.Y = ss.ArenaH - obs.H
			obs.VY = -obs.VY
		}

		// Slight random direction change every ~200 ticks
		if ss.Rng.Intn(200) == 0 {
			angle := math.Atan2(obs.VY, obs.VX)
			angle += (ss.Rng.Float64() - 0.5) * math.Pi * 0.5 // ±45 degrees
			speed := math.Sqrt(obs.VX*obs.VX + obs.VY*obs.VY)
			obs.VX = math.Cos(angle) * speed
			obs.VY = math.Sin(angle) * speed
		}
	}

	// 2. Expire packages (only in delivery mode)
	if !ss.DeliveryOn {
		return
	}

	for pi := range ss.Packages {
		pkg := &ss.Packages[pi]
		if !pkg.Active || pkg.CarriedBy >= 0 {
			continue // skip inactive or carried packages
		}

		age := ss.Tick - pkg.SpawnTick
		if age > DynPackageExpiry {
			// Expire this package — respawn at station
			pkg.Active = false
			pkg.OnGround = false
			pkg.CarriedBy = -1

			// Find matching pickup station and trigger respawn
			for si := range ss.Stations {
				st := &ss.Stations[si]
				if st.IsPickup && st.Color == pkg.Color && !st.HasPackage {
					st.RespawnIn = 150 // respawn delay
					st.FlashTimer = DynExpiredFlash
					st.FlashOK = false // red flash = expired
					break
				}
			}

			logger.Info("DYNAMIC", "Package expired (color %d, age %d ticks)", pkg.Color, age)
		}
	}
}
