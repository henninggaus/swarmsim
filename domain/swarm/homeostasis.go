package swarm

import (
	"math"
	"swarmsim/logger"
)

// HomeostasisState manages internal physiological regulation for bots.
// Each bot has internal drives (energy, stress, curiosity, safety) that
// must be kept in balance. Imbalances drive behavior: low energy triggers
// foraging, high stress triggers fleeing, high curiosity triggers exploring.
// This creates naturally adaptive, lifelike behavior.
type HomeostasisState struct {
	Drives []BotDrives // per-bot internal states

	// Global parameters
	EnergyDecay    float64 // energy drain per tick (default 0.002)
	StressDecay    float64 // natural stress reduction (default 0.005)
	CuriosityGrow  float64 // curiosity increase per tick (default 0.003)
	SafetyDecay    float64 // safety feeling decay (default 0.004)

	// Stats
	AvgEnergy    float64
	AvgStress    float64
	AvgCuriosity float64
	AvgSafety    float64
	CriticalBots int // bots with critically low energy
}

// BotDrives holds the internal state of one bot.
type BotDrives struct {
	Energy    float64 // 0-1, depletes with movement, restored near resources
	Stress    float64 // 0-1, increases from danger/crowding, decreases with safety
	Curiosity float64 // 0-1, grows naturally, reduced by exploring new areas
	Safety    float64 // 0-1, increases near allies, decreases when alone

	// Behavior mode determined by dominant drive
	DominantDrive DriveType
	DriveStrength float64 // how urgently the dominant drive needs attention
}

// DriveType represents which internal need is most pressing.
type DriveType int

const (
	DriveEnergy    DriveType = iota // need to find resources/rest
	DriveStress                     // need to escape/calm down
	DriveCuriosity                  // need to explore
	DriveSafety                     // need to find allies
)

// DriveTypeName returns the display name.
func DriveTypeName(d DriveType) string {
	switch d {
	case DriveEnergy:
		return "Energie"
	case DriveStress:
		return "Stress"
	case DriveCuriosity:
		return "Neugier"
	case DriveSafety:
		return "Sicherheit"
	default:
		return "?"
	}
}

// InitHomeostasis sets up the homeostasis system.
func InitHomeostasis(ss *SwarmState) {
	n := len(ss.Bots)
	hs := &HomeostasisState{
		Drives:        make([]BotDrives, n),
		EnergyDecay:   0.002,
		StressDecay:   0.005,
		CuriosityGrow: 0.003,
		SafetyDecay:   0.004,
	}

	for i := 0; i < n; i++ {
		hs.Drives[i] = BotDrives{
			Energy:    0.7 + ss.Rng.Float64()*0.3,
			Stress:    ss.Rng.Float64() * 0.2,
			Curiosity: ss.Rng.Float64() * 0.3,
			Safety:    0.5 + ss.Rng.Float64()*0.3,
		}
	}

	ss.Homeostasis = hs
	logger.Info("HOMEO", "Initialisiert: %d Bots mit internen Antrieben", n)
}

// ClearHomeostasis disables the homeostasis system.
func ClearHomeostasis(ss *SwarmState) {
	ss.Homeostasis = nil
	ss.HomeostasisOn = false
}

// TickHomeostasis runs one tick of homeostatic regulation.
func TickHomeostasis(ss *SwarmState) {
	hs := ss.Homeostasis
	if hs == nil {
		return
	}

	n := len(ss.Bots)
	if len(hs.Drives) != n {
		return
	}

	sumE, sumS, sumC, sumSf := 0.0, 0.0, 0.0, 0.0
	critical := 0

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		d := &hs.Drives[i]

		// Update drives based on environment
		updateDrives(ss, hs, bot, d)

		// Determine dominant drive (most urgent need)
		determineDominant(d)

		// Apply behavior based on dominant drive
		applyDriveBehavior(ss, bot, d)

		// Apply drive-based LED color
		applyDriveColor(bot, d)

		// Track stats
		sumE += d.Energy
		sumS += d.Stress
		sumC += d.Curiosity
		sumSf += d.Safety
		if d.Energy < 0.15 {
			critical++
		}
	}

	fn := float64(n)
	hs.AvgEnergy = sumE / fn
	hs.AvgStress = sumS / fn
	hs.AvgCuriosity = sumC / fn
	hs.AvgSafety = sumSf / fn
	hs.CriticalBots = critical
}

// updateDrives adjusts internal states based on the environment.
func updateDrives(ss *SwarmState, hs *HomeostasisState, bot *SwarmBot, d *BotDrives) {
	// Energy: depletes with movement, restored near resources
	d.Energy -= hs.EnergyDecay * (0.5 + bot.Speed/SwarmBotSpeed*0.5)
	if bot.NearestPickupDist < 50 {
		d.Energy += 0.01 // resources nearby restore energy
	}
	if bot.CarryingPkg >= 0 {
		d.Energy -= 0.001 // carrying costs extra energy
	}

	// Stress: increases from crowding and speed, decreases naturally
	d.Stress -= hs.StressDecay
	if bot.NeighborCount > 8 {
		d.Stress += 0.01 * float64(bot.NeighborCount-8)
	}
	if bot.Speed > SwarmBotSpeed*1.2 {
		d.Stress += 0.003
	}

	// Curiosity: grows naturally, reduced when exploring (moving fast alone)
	d.Curiosity += hs.CuriosityGrow
	if bot.NeighborCount < 3 && bot.Speed > SwarmBotSpeed*0.7 {
		d.Curiosity -= 0.005 // exploring satisfies curiosity
	}

	// Safety: increases near allies, decreases when alone
	d.Safety -= hs.SafetyDecay
	if bot.NeighborCount > 3 {
		d.Safety += 0.008 * math.Min(float64(bot.NeighborCount), 10) / 10
	}

	// Clamp all drives
	d.Energy = clampF(d.Energy, 0, 1)
	d.Stress = clampF(d.Stress, 0, 1)
	d.Curiosity = clampF(d.Curiosity, 0, 1)
	d.Safety = clampF(d.Safety, 0, 1)
}

// determineDominant finds the most urgent unmet need.
func determineDominant(d *BotDrives) {
	// Urgency: how far from ideal each drive is
	// Energy: low = urgent, Stress: high = urgent,
	// Curiosity: high = urgent, Safety: low = urgent
	urgencies := [4]float64{
		1.0 - d.Energy,  // low energy = high urgency
		d.Stress,         // high stress = high urgency
		d.Curiosity,      // high curiosity = high urgency
		1.0 - d.Safety,  // low safety = high urgency
	}

	bestDrive := DriveType(0)
	bestUrgency := urgencies[0]
	for i := 1; i < 4; i++ {
		if urgencies[i] > bestUrgency {
			bestUrgency = urgencies[i]
			bestDrive = DriveType(i)
		}
	}

	d.DominantDrive = bestDrive
	d.DriveStrength = bestUrgency
}

// applyDriveBehavior modifies bot movement based on its dominant need.
func applyDriveBehavior(ss *SwarmState, bot *SwarmBot, d *BotDrives) {
	strength := d.DriveStrength * 0.3

	switch d.DominantDrive {
	case DriveEnergy:
		// Low energy: slow down to conserve, seek resources
		bot.Speed *= 1.0 - strength*0.4
		if d.Energy < 0.1 {
			bot.Speed *= 0.5 // critical: nearly stop
		}

	case DriveStress:
		// High stress: speed up, turn away from crowds
		bot.Speed *= 1.0 + strength*0.5
		bot.Angle += (ss.Rng.Float64() - 0.5) * strength * 0.3

	case DriveCuriosity:
		// High curiosity: explore widely
		bot.Speed *= 1.0 + strength*0.3
		bot.Angle += (ss.Rng.Float64() - 0.5) * strength * 0.4

	case DriveSafety:
		// Low safety: slow down slightly, stay near others
		bot.Speed *= 1.0 - strength*0.2
	}
}

// applyDriveColor sets LED based on dominant drive.
func applyDriveColor(bot *SwarmBot, d *BotDrives) {
	intensity := uint8(100 + d.DriveStrength*155)
	switch d.DominantDrive {
	case DriveEnergy:
		bot.LEDColor = [3]uint8{intensity, intensity / 2, 0} // orange: hungry
	case DriveStress:
		bot.LEDColor = [3]uint8{intensity, 0, 0} // red: stressed
	case DriveCuriosity:
		bot.LEDColor = [3]uint8{0, intensity, intensity} // cyan: curious
	case DriveSafety:
		bot.LEDColor = [3]uint8{0, intensity / 2, intensity} // blue: lonely
	}
}

// HomeoAvgEnergy returns the population average energy.
func HomeoAvgEnergy(hs *HomeostasisState) float64 {
	if hs == nil {
		return 0
	}
	return hs.AvgEnergy
}

// HomeoAvgStress returns the population average stress.
func HomeoAvgStress(hs *HomeostasisState) float64 {
	if hs == nil {
		return 0
	}
	return hs.AvgStress
}

// HomeoCriticalBots returns how many bots are critically low on energy.
func HomeoCriticalBots(hs *HomeostasisState) int {
	if hs == nil {
		return 0
	}
	return hs.CriticalBots
}

// BotDominantDrive returns a bot's most urgent drive.
func BotDominantDrive(hs *HomeostasisState, botIdx int) DriveType {
	if hs == nil || botIdx < 0 || botIdx >= len(hs.Drives) {
		return DriveEnergy
	}
	return hs.Drives[botIdx].DominantDrive
}

// BotDriveEnergy returns a bot's homeostatic energy level.
func BotDriveEnergy(hs *HomeostasisState, botIdx int) float64 {
	if hs == nil || botIdx < 0 || botIdx >= len(hs.Drives) {
		return 0
	}
	return hs.Drives[botIdx].Energy
}
