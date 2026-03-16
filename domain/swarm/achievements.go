package swarm

import "time"

// AchievementID uniquely identifies an achievement.
type AchievementID int

const (
	// --- Delivery Achievements ---
	AchFirstDelivery    AchievementID = iota // First package delivered
	AchDelivery10                             // 10 packages delivered
	AchDelivery50                             // 50 packages delivered
	AchDelivery100                            // 100 packages delivered
	AchSpeedDemon                             // 10 deliveries in under 500 ticks
	AchPerfectRound                           // Truck round with 0 collisions

	// --- Evolution Achievements ---
	AchFirstGen                               // Complete first generation
	AchGen10                                  // Reach generation 10
	AchGen50                                  // Reach generation 50
	AchFitnessJump                            // Fitness doubles in one generation
	AchSpeciesExplosion                       // 5+ species alive at once
	AchConvergence                            // All bots within 10% fitness

	// --- Swarm Behavior Achievements ---
	AchPerfectAlignment                       // Alignment > 0.95
	AchTightCluster                           // Cohesion > 0.95
	AchVortex                                 // Vortex pattern detected
	AchCircle                                 // Circle pattern detected
	AchStream                                 // Stream pattern with 50+ bots
	AchChaos                                  // Entropy > 0.95

	// --- Exploration Achievements ---
	AchExplorer25                             // 25% of heatmap visited
	AchExplorer50                             // 50% of heatmap visited
	AchExplorer90                             // 90% of heatmap visited

	// --- Scale Achievements ---
	AchBots100                                // Run with 100+ bots
	AchBots300                                // Run with 300+ bots
	AchBots500                                // Run with 500 bots (max)
	AchMarathon                               // Simulation runs 100,000 ticks
	AchNightOwl                               // Survive 5 full day/night cycles

	// --- Mastery Achievements ---
	AchAllOverlays                            // Toggle every overlay at least once
	AchDSLExpert                              // Use 10+ different actions in one program

	AchCount // sentinel
)

// AchievementDifficulty ranks how hard an achievement is to earn.
type AchievementDifficulty int

const (
	DiffBronze   AchievementDifficulty = iota // easy
	DiffSilver                                 // medium
	DiffGold                                   // hard
	DiffDiamond                                // legendary
)

// AchievementDef describes a single achievement.
type AchievementDef struct {
	ID          AchievementID
	Name        string // German display name
	Description string // German description
	Icon        string // emoji-like short code
	Difficulty  AchievementDifficulty
}

// Achievement is a runtime instance of an earned achievement.
type Achievement struct {
	ID       AchievementID
	Unlocked bool
	Tick     int       // simulation tick when unlocked
	Time     time.Time // wall-clock time when unlocked
}

// AchievementState tracks all achievements for the session.
type AchievementState struct {
	Achievements [AchCount]Achievement
	RecentPopup  *AchievementPopup // currently displaying popup (nil = none)
	PopupQueue   []AchievementID   // queued popups
	TotalUnlocked int

	// Tracking helpers
	OverlaysUsed    map[string]bool // which overlays have been toggled
	DayNightCycles  int             // completed day/night cycles
	PrevDayNight    float64         // previous phase for cycle detection
}

// AchievementPopup holds display state for the unlock notification.
type AchievementPopup struct {
	ID    AchievementID
	Timer int // frames remaining (counts down from 180 = ~3 seconds at 60fps)
}

// AllAchievements defines every achievement in the game.
var AllAchievements = [AchCount]AchievementDef{
	AchFirstDelivery:    {AchFirstDelivery, "Erste Lieferung", "Erstes Paket erfolgreich zugestellt", "📦", DiffBronze},
	AchDelivery10:       {AchDelivery10, "Fleissige Boten", "10 Pakete zugestellt", "📬", DiffBronze},
	AchDelivery50:       {AchDelivery50, "Logistik-Profi", "50 Pakete zugestellt", "🚚", DiffSilver},
	AchDelivery100:      {AchDelivery100, "Paket-Imperium", "100 Pakete zugestellt", "🏭", DiffGold},
	AchSpeedDemon:       {AchSpeedDemon, "Blitzlieferung", "10 Lieferungen in unter 500 Ticks", "⚡", DiffGold},
	AchPerfectRound:     {AchPerfectRound, "Makellos", "Truck-Runde ohne Kollision", "✨", DiffSilver},

	AchFirstGen:         {AchFirstGen, "Evolution!", "Erste Generation abgeschlossen", "🧬", DiffBronze},
	AchGen10:            {AchGen10, "Selektion", "Generation 10 erreicht", "🔬", DiffBronze},
	AchGen50:            {AchGen50, "Darwin waere stolz", "Generation 50 erreicht", "🏆", DiffSilver},
	AchFitnessJump:      {AchFitnessJump, "Quantensprung", "Fitness verdoppelt in einer Generation", "🚀", DiffGold},
	AchSpeciesExplosion: {AchSpeciesExplosion, "Artenvielfalt", "5+ Spezies gleichzeitig", "🌈", DiffSilver},
	AchConvergence:      {AchConvergence, "Konvergenz", "Alle Bots innerhalb 10% Fitness", "🎯", DiffGold},

	AchPerfectAlignment: {AchPerfectAlignment, "Gleichschritt", "Alignment > 0.95", "➡️", DiffSilver},
	AchTightCluster:     {AchTightCluster, "Zusammenhalt", "Kohaesion > 0.95", "🫂", DiffSilver},
	AchVortex:           {AchVortex, "Wirbelwind", "Wirbel-Muster erkannt", "🌀", DiffGold},
	AchCircle:           {AchCircle, "Kreismeister", "Kreis-Formation erkannt", "⭕", DiffGold},
	AchStream:           {AchStream, "Schwarmstrom", "Strom-Muster mit 50+ Bots", "🌊", DiffSilver},
	AchChaos:            {AchChaos, "Totales Chaos", "Entropie > 0.95", "💥", DiffBronze},

	AchExplorer25:       {AchExplorer25, "Entdecker", "25% der Arena erkundet", "🗺️", DiffBronze},
	AchExplorer50:       {AchExplorer50, "Kartograph", "50% der Arena erkundet", "🧭", DiffSilver},
	AchExplorer90:       {AchExplorer90, "Welteroberer", "90% der Arena erkundet", "🌍", DiffDiamond},

	AchBots100:          {AchBots100, "Schwarm", "100+ Bots gleichzeitig", "🐝", DiffBronze},
	AchBots300:          {AchBots300, "Armee", "300+ Bots gleichzeitig", "🐜", DiffSilver},
	AchBots500:          {AchBots500, "Legion", "500 Bots (Maximum!)", "👑", DiffGold},
	AchMarathon:         {AchMarathon, "Marathon", "100.000 Ticks Simulation", "⏱️", DiffSilver},
	AchNightOwl:         {AchNightOwl, "Nachteule", "5 Tag/Nacht-Zyklen ueberlebt", "🦉", DiffSilver},

	AchAllOverlays:      {AchAllOverlays, "Kontrollfreak", "Jedes Overlay mindestens einmal benutzt", "🎛️", DiffDiamond},
	AchDSLExpert:        {AchDSLExpert, "Codemaster", "10+ verschiedene Aktionen im Programm", "💻", DiffGold},
}

// DifficultyName returns the German display name for a difficulty.
func DifficultyName(d AchievementDifficulty) string {
	switch d {
	case DiffBronze:
		return "Bronze"
	case DiffSilver:
		return "Silber"
	case DiffGold:
		return "Gold"
	case DiffDiamond:
		return "Diamant"
	}
	return "?"
}

// DifficultyColor returns RGBA color for a difficulty tier.
func DifficultyColor(d AchievementDifficulty) [3]uint8 {
	switch d {
	case DiffBronze:
		return [3]uint8{205, 127, 50}
	case DiffSilver:
		return [3]uint8{192, 192, 192}
	case DiffGold:
		return [3]uint8{255, 215, 0}
	case DiffDiamond:
		return [3]uint8{0, 255, 255}
	}
	return [3]uint8{255, 255, 255}
}

// NewAchievementState creates a fresh achievement tracking state.
func NewAchievementState() *AchievementState {
	return &AchievementState{
		OverlaysUsed: make(map[string]bool),
	}
}

// Unlock marks an achievement as earned if not already unlocked.
func (as *AchievementState) Unlock(id AchievementID, tick int) {
	if as.Achievements[id].Unlocked {
		return
	}
	as.Achievements[id] = Achievement{
		ID:       id,
		Unlocked: true,
		Tick:     tick,
		Time:     time.Now(),
	}
	as.TotalUnlocked++
	as.PopupQueue = append(as.PopupQueue, id)
}

// IsUnlocked checks if an achievement has been earned.
func (as *AchievementState) IsUnlocked(id AchievementID) bool {
	return as.Achievements[id].Unlocked
}

// UpdatePopup advances the popup animation. Call once per frame.
func (as *AchievementState) UpdatePopup() {
	if as.RecentPopup != nil {
		as.RecentPopup.Timer--
		if as.RecentPopup.Timer <= 0 {
			as.RecentPopup = nil
		}
	}
	// Dequeue next popup if none active
	if as.RecentPopup == nil && len(as.PopupQueue) > 0 {
		id := as.PopupQueue[0]
		as.PopupQueue = as.PopupQueue[1:]
		as.RecentPopup = &AchievementPopup{ID: id, Timer: 180}
	}
}

// CheckAchievements evaluates all achievement conditions against current state.
// Called periodically (every ~60 ticks) to avoid overhead.
func CheckAchievements(ss *SwarmState) {
	as := ss.AchievementState
	if as == nil {
		return
	}
	tick := ss.Tick

	// --- Delivery ---
	d := ss.DeliveryStats.TotalDelivered
	if d >= 1 {
		as.Unlock(AchFirstDelivery, tick)
	}
	if d >= 10 {
		as.Unlock(AchDelivery10, tick)
	}
	if d >= 50 {
		as.Unlock(AchDelivery50, tick)
	}
	if d >= 100 {
		as.Unlock(AchDelivery100, tick)
	}

	// Speed demon: 10+ deliveries and tick < 500
	if d >= 10 && tick < 500 {
		as.Unlock(AchSpeedDemon, tick)
	}

	// --- Evolution ---
	gen := ss.Generation
	if ss.GPEnabled {
		gen = ss.GPGeneration
	}
	if ss.NeuroEnabled {
		gen = ss.NeuroGeneration
	}
	if gen >= 1 {
		as.Unlock(AchFirstGen, tick)
	}
	if gen >= 10 {
		as.Unlock(AchGen10, tick)
	}
	if gen >= 50 {
		as.Unlock(AchGen50, tick)
	}

	// Fitness jump: check recent history
	if len(ss.FitnessHistory) >= 2 {
		last := ss.FitnessHistory[len(ss.FitnessHistory)-1]
		prev := ss.FitnessHistory[len(ss.FitnessHistory)-2]
		if prev.Best > 0 && last.Best >= prev.Best*2 {
			as.Unlock(AchFitnessJump, tick)
		}
	}

	// Convergence: all bots within 10% fitness
	if ss.AvgFitness > 0 && ss.BestFitness > 0 {
		ratio := ss.AvgFitness / ss.BestFitness
		if ratio > 0.9 {
			as.Unlock(AchConvergence, tick)
		}
	}

	// Species explosion
	if ss.Speciation != nil && len(ss.Speciation.Species) >= 5 {
		as.Unlock(AchSpeciesExplosion, tick)
	}

	// --- Pattern Detection ---
	if ss.PatternResult != nil {
		pr := ss.PatternResult
		if pr.Alignment > 0.95 {
			as.Unlock(AchPerfectAlignment, tick)
		}
		if pr.Cohesion > 0.95 {
			as.Unlock(AchTightCluster, tick)
		}
		if pr.Primary == PatternVortex && pr.PrimaryScore > 0.3 {
			as.Unlock(AchVortex, tick)
		}
		if pr.Primary == PatternCircle && pr.PrimaryScore > 0.3 {
			as.Unlock(AchCircle, tick)
		}
		if pr.Primary == PatternStream && len(ss.Bots) >= 50 {
			as.Unlock(AchStream, tick)
		}
		if pr.Entropy > 0.95 {
			as.Unlock(AchChaos, tick)
		}
	}

	// --- Exploration (heatmap coverage) ---
	if ss.ShowHeatmap && len(ss.HeatmapGrid) > 0 {
		visited := 0
		for _, v := range ss.HeatmapGrid {
			if v > 0 {
				visited++
			}
		}
		total := len(ss.HeatmapGrid)
		pct := float64(visited) / float64(total)
		if pct >= 0.25 {
			as.Unlock(AchExplorer25, tick)
		}
		if pct >= 0.50 {
			as.Unlock(AchExplorer50, tick)
		}
		if pct >= 0.90 {
			as.Unlock(AchExplorer90, tick)
		}
	}

	// --- Scale ---
	n := len(ss.Bots)
	if n >= 100 {
		as.Unlock(AchBots100, tick)
	}
	if n >= 300 {
		as.Unlock(AchBots300, tick)
	}
	if n >= 500 {
		as.Unlock(AchBots500, tick)
	}
	if tick >= 100000 {
		as.Unlock(AchMarathon, tick)
	}

	// Day/Night cycles
	if ss.DayNightOn {
		// Detect full cycle: phase wraps past 0
		if as.PrevDayNight > 0.9 && ss.DayNightPhase < 0.1 {
			as.DayNightCycles++
		}
		as.PrevDayNight = ss.DayNightPhase
		if as.DayNightCycles >= 5 {
			as.Unlock(AchNightOwl, tick)
		}
	}

	// All overlays
	requiredOverlays := []string{"trails", "heatmap", "routes", "commgraph", "formation",
		"genome", "livechart", "dashboard", "leaderboard", "aurora", "prediction",
		"daynight", "center", "zones", "speciation", "patterns"}
	allUsed := true
	for _, ov := range requiredOverlays {
		if !as.OverlaysUsed[ov] {
			allUsed = false
			break
		}
	}
	if allUsed {
		as.Unlock(AchAllOverlays, tick)
	}

	// DSL expert: 10+ different actions in current program
	if ss.Program != nil {
		actionSet := make(map[int]bool)
		for _, r := range ss.Program.Rules {
			actionSet[int(r.Action.Type)] = true
		}
		if len(actionSet) >= 10 {
			as.Unlock(AchDSLExpert, tick)
		}
	}
}

// RecordOverlayUsed marks an overlay as having been toggled.
func RecordOverlayUsed(ss *SwarmState, name string) {
	if ss.AchievementState != nil {
		ss.AchievementState.OverlaysUsed[name] = true
	}
}
