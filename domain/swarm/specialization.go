package swarm

import (
	"math"
	"swarmsim/logger"
)

// SpecializationState manages emergent division of labor.
// Each bot accumulates experience in different tasks (foraging, scouting,
// guarding, building). Over time, bots naturally specialize based on
// reinforcement — bots that succeed at a task get better at it.
type SpecializationState struct {
	Profiles []SpecProfile // per-bot specialization profiles

	// Population-level stats
	AvgSpecialization float64 // how specialized the population is (0=generalist, 1=specialist)
	RoleCounts        [4]int  // count per role
	Generation        int
}

// SpecRole represents a bot's primary role.
type SpecRole int

const (
	RoleForager SpecRole = iota // collect and deliver resources
	RoleScout                   // explore unknown areas
	RoleGuard                   // protect territory / other bots
	RoleBuilder                 // stay near base, organize
)

// RoleName returns the display name for a role.
func RoleName(r SpecRole) string {
	switch r {
	case RoleForager:
		return "Sammler"
	case RoleScout:
		return "Kundschafter"
	case RoleGuard:
		return "Waechter"
	case RoleBuilder:
		return "Bauer"
	default:
		return "?"
	}
}

// SpecProfile holds a bot's experience and specialization.
type SpecProfile struct {
	Experience   [4]float64 // accumulated experience per role
	CurrentRole  SpecRole   // dominant role
	Specialization float64  // degree of specialization (0-1)
	RoleTicks    int        // ticks in current role
}

// InitSpecialization sets up the specialization system.
func InitSpecialization(ss *SwarmState) {
	n := len(ss.Bots)
	sp := &SpecializationState{
		Profiles: make([]SpecProfile, n),
	}

	// Start with slight random biases
	for i := 0; i < n; i++ {
		for r := 0; r < 4; r++ {
			sp.Profiles[i].Experience[r] = ss.Rng.Float64() * 0.1
		}
		sp.Profiles[i].CurrentRole = SpecRole(ss.Rng.Intn(4))
	}

	ss.Specialization = sp
	logger.Info("SPEC", "Initialisiert: %d Bots mit Spezialisierungs-Profilen", n)
}

// ClearSpecialization disables the specialization system.
func ClearSpecialization(ss *SwarmState) {
	ss.Specialization = nil
	ss.SpecializationOn = false
}

// TickSpecialization runs one tick of the specialization system.
func TickSpecialization(ss *SwarmState) {
	sp := ss.Specialization
	if sp == nil {
		return
	}

	n := len(ss.Bots)
	if len(sp.Profiles) != n {
		return
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		prof := &sp.Profiles[i]

		// Accumulate experience based on current activity
		accumulateExperience(ss, bot, prof)

		// Determine dominant role
		bestRole := SpecRole(0)
		bestExp := prof.Experience[0]
		totalExp := prof.Experience[0]
		for r := 1; r < 4; r++ {
			totalExp += prof.Experience[r]
			if prof.Experience[r] > bestExp {
				bestExp = prof.Experience[r]
				bestRole = SpecRole(r)
			}
		}

		if bestRole != prof.CurrentRole {
			prof.CurrentRole = bestRole
			prof.RoleTicks = 0
		}
		prof.RoleTicks++

		// Compute specialization degree (how dominant is the best role)
		if totalExp > 0 {
			prof.Specialization = bestExp / totalExp
		}

		// Apply role-based behavior modifications
		applyRoleBehavior(ss, bot, prof)

		// Role-based LED colors
		applyRoleColor(bot, prof)
	}

	updateSpecStats(sp)
}

// accumulateExperience gives experience points based on bot activity.
func accumulateExperience(ss *SwarmState, bot *SwarmBot, prof *SpecProfile) {
	learnRate := 0.01

	// Foraging: near pickups or carrying
	if bot.CarryingPkg >= 0 {
		prof.Experience[RoleForager] += learnRate * 2.0
	} else if bot.NearestPickupDist < 80 {
		prof.Experience[RoleForager] += learnRate
	}

	// Scouting: far from other bots, moving fast
	if bot.NeighborCount < 3 && bot.Speed > SwarmBotSpeed*0.8 {
		prof.Experience[RoleScout] += learnRate * 1.5
	}

	// Guarding: near many bots, staying in area
	if bot.NeighborCount > 5 {
		prof.Experience[RoleGuard] += learnRate * 1.5
	}

	// Building: near base/center
	cx := float64(ss.ArenaW) / 2
	cy := float64(ss.ArenaH) / 2
	dx := bot.X - cx
	dy := bot.Y - cy
	distToCenter := math.Sqrt(dx*dx + dy*dy)
	if distToCenter < float64(ss.ArenaW)/4 {
		prof.Experience[RoleBuilder] += learnRate
	}

	// Decay all experience slowly to allow role switching
	for r := 0; r < 4; r++ {
		prof.Experience[r] *= 0.9999
	}
}

// applyRoleBehavior modifies bot behavior based on its role.
func applyRoleBehavior(ss *SwarmState, bot *SwarmBot, prof *SpecProfile) {
	strength := math.Min(prof.Specialization, 1.0) * 0.3

	switch prof.CurrentRole {
	case RoleForager:
		// Better at finding resources
		if bot.NearestPickupDist < 120 && bot.CarryingPkg < 0 {
			bot.Speed *= 1.0 + strength
		}
	case RoleScout:
		// Move faster, wider turns
		bot.Speed *= 1.0 + strength*0.5
		bot.Angle += (ss.Rng.Float64() - 0.5) * strength * 0.2
	case RoleGuard:
		// Slow down, stay in area
		bot.Speed *= 1.0 - strength*0.3
	case RoleBuilder:
		// Gravitate toward center
		cx := float64(ss.ArenaW) / 2
		cy := float64(ss.ArenaH) / 2
		dx := cx - bot.X
		dy := cy - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 10 {
			angle := math.Atan2(dy, dx)
			diff := angle - bot.Angle
			diff = WrapAngle(diff)
			bot.Angle += diff * strength * 0.1
		}
	}
}

// applyRoleColor sets LED color based on role.
func applyRoleColor(bot *SwarmBot, prof *SpecProfile) {
	intensity := uint8(80 + prof.Specialization*175)
	switch prof.CurrentRole {
	case RoleForager:
		bot.LEDColor = [3]uint8{intensity, intensity / 2, 0} // orange
	case RoleScout:
		bot.LEDColor = [3]uint8{0, intensity, intensity} // cyan
	case RoleGuard:
		bot.LEDColor = [3]uint8{intensity, 0, 0} // red
	case RoleBuilder:
		bot.LEDColor = [3]uint8{intensity / 2, intensity, 0} // green-yellow
	}
}

// updateSpecStats computes population-level specialization statistics.
func updateSpecStats(sp *SpecializationState) {
	n := len(sp.Profiles)
	if n == 0 {
		return
	}

	totalSpec := 0.0
	sp.RoleCounts = [4]int{}

	for _, prof := range sp.Profiles {
		totalSpec += prof.Specialization
		sp.RoleCounts[prof.CurrentRole]++
	}

	sp.AvgSpecialization = totalSpec / float64(n)
}

// GetBotRole returns the current role of a bot.
func GetBotRole(sp *SpecializationState, botIdx int) SpecRole {
	if sp == nil || botIdx < 0 || botIdx >= len(sp.Profiles) {
		return RoleForager
	}
	return sp.Profiles[botIdx].CurrentRole
}

// GetBotSpecialization returns how specialized a bot is (0-1).
func GetBotSpecialization(sp *SpecializationState, botIdx int) float64 {
	if sp == nil || botIdx < 0 || botIdx >= len(sp.Profiles) {
		return 0
	}
	return sp.Profiles[botIdx].Specialization
}

// PopulationSpecialization returns average specialization.
func PopulationSpecialization(sp *SpecializationState) float64 {
	if sp == nil {
		return 0
	}
	return sp.AvgSpecialization
}
