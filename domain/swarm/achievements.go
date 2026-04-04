package swarm

import (
	"swarmsim/locale"
	"time"
)

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

	// --- Algo-Labor Achievements ---
	AchAlgoActivated                          // First optimization algorithm activated in Algo-Labor
	AchAlgoLaborist                           // 5+ different algorithms tried in Algo-Labor
	AchAlgoTournament                         // Auto-Tournament completed in Algo-Labor

	// --- Neuro / GP Achievements ---
	AchNeuroDrives                            // First delivery with Neuro enabled
	AchGPMaster                              // GP fitness exceeds 500

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
	ID         AchievementID
	NameKey    string // locale key for display name
	DescKey    string // locale key for description
	Icon       string // emoji-like short code
	Difficulty AchievementDifficulty
}

// DisplayName returns the localized achievement name.
func (d AchievementDef) DisplayName() string { return locale.T(d.NameKey) }

// DisplayDesc returns the localized achievement description.
func (d AchievementDef) DisplayDesc() string { return locale.T(d.DescKey) }

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
	OverlaysUsed    map[string]bool            // which overlays have been toggled
	TriedAlgos      map[SwarmAlgorithmType]bool // which algo-labor algorithms have been tried
	DayNightCycles  int                        // completed day/night cycles
	PrevDayNight    float64                    // previous phase for cycle detection
}

// AchievementPopup holds display state for the unlock notification.
type AchievementPopup struct {
	ID    AchievementID
	Timer int // frames remaining (counts down from 180 = ~3 seconds at 60fps)
}

// AllAchievements defines every achievement in the game.
var AllAchievements = [AchCount]AchievementDef{
	AchFirstDelivery:    {AchFirstDelivery, "ach.name.first_delivery", "ach.desc.first_delivery", "📦", DiffBronze},
	AchDelivery10:       {AchDelivery10, "ach.name.delivery10", "ach.desc.delivery10", "📬", DiffBronze},
	AchDelivery50:       {AchDelivery50, "ach.name.delivery50", "ach.desc.delivery50", "🚚", DiffSilver},
	AchDelivery100:      {AchDelivery100, "ach.name.delivery100", "ach.desc.delivery100", "🏭", DiffGold},
	AchSpeedDemon:       {AchSpeedDemon, "ach.name.speed_demon", "ach.desc.speed_demon", "⚡", DiffGold},
	AchPerfectRound:     {AchPerfectRound, "ach.name.perfect_round", "ach.desc.perfect_round", "✨", DiffSilver},

	AchFirstGen:         {AchFirstGen, "ach.name.first_gen", "ach.desc.first_gen", "🧬", DiffBronze},
	AchGen10:            {AchGen10, "ach.name.gen10", "ach.desc.gen10", "🔬", DiffBronze},
	AchGen50:            {AchGen50, "ach.name.gen50", "ach.desc.gen50", "🏆", DiffSilver},
	AchFitnessJump:      {AchFitnessJump, "ach.name.fitness_jump", "ach.desc.fitness_jump", "🚀", DiffGold},
	AchSpeciesExplosion: {AchSpeciesExplosion, "ach.name.species_explosion", "ach.desc.species_explosion", "🌈", DiffSilver},
	AchConvergence:      {AchConvergence, "ach.name.convergence", "ach.desc.convergence", "🎯", DiffGold},

	AchPerfectAlignment: {AchPerfectAlignment, "ach.name.perfect_alignment", "ach.desc.perfect_alignment", "➡️", DiffSilver},
	AchTightCluster:     {AchTightCluster, "ach.name.tight_cluster", "ach.desc.tight_cluster", "🫂", DiffSilver},
	AchVortex:           {AchVortex, "ach.name.vortex", "ach.desc.vortex", "🌀", DiffGold},
	AchCircle:           {AchCircle, "ach.name.circle", "ach.desc.circle", "⭕", DiffGold},
	AchStream:           {AchStream, "ach.name.stream", "ach.desc.stream", "🌊", DiffSilver},
	AchChaos:            {AchChaos, "ach.name.chaos", "ach.desc.chaos", "💥", DiffBronze},

	AchExplorer25:       {AchExplorer25, "ach.name.explorer25", "ach.desc.explorer25", "🗺️", DiffBronze},
	AchExplorer50:       {AchExplorer50, "ach.name.explorer50", "ach.desc.explorer50", "🧭", DiffSilver},
	AchExplorer90:       {AchExplorer90, "ach.name.explorer90", "ach.desc.explorer90", "🌍", DiffDiamond},

	AchBots100:          {AchBots100, "ach.name.bots100", "ach.desc.bots100", "🐝", DiffBronze},
	AchBots300:          {AchBots300, "ach.name.bots300", "ach.desc.bots300", "🐜", DiffSilver},
	AchBots500:          {AchBots500, "ach.name.bots500", "ach.desc.bots500", "👑", DiffGold},
	AchMarathon:         {AchMarathon, "ach.name.marathon", "ach.desc.marathon", "⏱️", DiffSilver},
	AchNightOwl:         {AchNightOwl, "ach.name.night_owl", "ach.desc.night_owl", "🦉", DiffSilver},

	AchAllOverlays:      {AchAllOverlays, "ach.name.all_overlays", "ach.desc.all_overlays", "🎛️", DiffDiamond},
	AchDSLExpert:        {AchDSLExpert, "ach.name.dsl_expert", "ach.desc.dsl_expert", "💻", DiffGold},

	AchAlgoActivated:    {AchAlgoActivated, "ach.name.algo_activated", "ach.desc.algo_activated", "⚗️", DiffBronze},
	AchAlgoLaborist:     {AchAlgoLaborist, "ach.name.algo_laborist", "ach.desc.algo_laborist", "🔭", DiffSilver},
	AchAlgoTournament:   {AchAlgoTournament, "ach.name.algo_tournament", "ach.desc.algo_tournament", "🏅", DiffGold},

	AchNeuroDrives:      {AchNeuroDrives, "ach.name.neuro_drives", "ach.desc.neuro_drives", "🧠", DiffSilver},
	AchGPMaster:         {AchGPMaster, "ach.name.gp_master", "ach.desc.gp_master", "🌱", DiffGold},
}

// DifficultyName returns the localized display name for a difficulty.
func DifficultyName(d AchievementDifficulty) string {
	switch d {
	case DiffBronze:
		return locale.T("diff.bronze")
	case DiffSilver:
		return locale.T("diff.silver")
	case DiffGold:
		return locale.T("diff.gold")
	case DiffDiamond:
		return locale.T("diff.diamond")
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
		TriedAlgos:   make(map[SwarmAlgorithmType]bool),
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

	// --- Algo-Labor ---
	if ss.SwarmAlgo != nil && ss.SwarmAlgo.ActiveAlgo != AlgoNone {
		as.Unlock(AchAlgoActivated, tick)
		if as.TriedAlgos == nil {
			as.TriedAlgos = make(map[SwarmAlgorithmType]bool)
		}
		as.TriedAlgos[ss.SwarmAlgo.ActiveAlgo] = true
		if len(as.TriedAlgos) >= 5 {
			as.Unlock(AchAlgoLaborist, tick)
		}
	}
	if ss.AlgoTournamentTotal > 0 && ss.AlgoTournamentDone >= ss.AlgoTournamentTotal {
		as.Unlock(AchAlgoTournament, tick)
	}

	// --- Neuro Delivery ---
	if ss.NeuroEnabled && ss.DeliveryStats.TotalDelivered >= 1 {
		as.Unlock(AchNeuroDrives, tick)
	}

	// --- GP Master ---
	if ss.GPEnabled && ss.BestFitness >= 500 {
		as.Unlock(AchGPMaster, tick)
	}
}

// RecordOverlayUsed marks an overlay as having been toggled.
func RecordOverlayUsed(ss *SwarmState, name string) {
	if ss.AchievementState != nil {
		ss.AchievementState.OverlaysUsed[name] = true
	}
}
