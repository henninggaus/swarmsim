package swarm

import (
	"math"
	"sort"
)

// ═══════════════════════════════════════════════════════════
// NOVELTY SEARCH — Verhaltens-basierte Fitness
// ═══════════════════════════════════════════════════════════
//
// Statt nur Task-Fitness (Deliveries, Distance, etc.) zu
// belohnen, wird auch Neuartigkeit des Verhaltens bewertet.
// Das verhindert fruehes Konvergieren zu lokalen Optima.
//
// Jeder Bot wird durch einen 8-dimensionalen Behavior-Vektor
// beschrieben. Novelty = durchschnittliche Distanz zu den
// k naechsten Nachbarn im Behavior-Space (Archiv + Population).
//
// Blended Fitness = alpha * TaskFitness + (1-alpha) * NoveltyScore

const BehaviorDims = 8

// BehaviorDescriptor is a fixed-length vector characterizing a bot's behavior.
type BehaviorDescriptor [BehaviorDims]float64

// NoveltyArchive stores interesting behaviors discovered over time.
type NoveltyArchive struct {
	Archive        []BehaviorDescriptor
	MaxSize        int     // max archive entries (default 500)
	KNeighbors     int     // k for k-NN novelty (default 15)
	AddThreshold   float64 // min novelty to be added to archive
	Alpha          float64 // blend: 1.0=pure task, 0.0=pure novelty (default 0.5)
	NoveltyScores  []float64
	AvgNovelty     float64
	MaxNovelty     float64
	NoveltyHistory []NoveltyRecord
}

// NoveltyRecord stores per-generation novelty statistics.
type NoveltyRecord struct {
	AvgNovelty  float64
	MaxNovelty  float64
	ArchiveSize int
}

// InitNovelty creates a new NoveltyArchive with default parameters.
func InitNovelty(ss *SwarmState) {
	ss.NoveltyArchive = &NoveltyArchive{
		Archive:      nil,
		MaxSize:      500,
		KNeighbors:   15,
		AddThreshold: 0.1,
		Alpha:        0.5,
	}
	ss.NoveltyEnabled = true
}

// ClearNovelty disables novelty search and removes the archive.
func ClearNovelty(ss *SwarmState) {
	ss.NoveltyArchive = nil
	ss.NoveltyEnabled = false
}

// ComputeBehavior builds a behavior descriptor from a bot's lifetime stats.
// All dimensions are normalized to roughly [0,1].
func ComputeBehavior(bot *SwarmBot, ss *SwarmState) BehaviorDescriptor {
	var b BehaviorDescriptor

	// 0: Final X position (normalized)
	if ss.ArenaW > 0 {
		b[0] = bot.X / ss.ArenaW
	}
	// 1: Final Y position (normalized)
	if ss.ArenaH > 0 {
		b[1] = bot.Y / ss.ArenaH
	}
	// 2: Total distance (clamped to [0,1])
	b[2] = clampF(bot.Stats.TotalDistance/5000.0, 0, 1)
	// 3: Delivery count
	b[3] = clampF(float64(bot.Stats.TotalDeliveries)/10.0, 0, 1)
	// 4: Pickup count
	b[4] = clampF(float64(bot.Stats.TotalPickups)/10.0, 0, 1)
	// 5: Fraction of time carrying
	alive := float64(bot.Stats.TicksAlive)
	if alive > 0 {
		b[5] = clampF(float64(bot.Stats.TicksCarrying)/alive, 0, 1)
	}
	// 6: Fraction of time idle
	if alive > 0 {
		b[6] = clampF(float64(bot.Stats.TicksIdle)/alive, 0, 1)
	}
	// 7: Average neighbor count (sociality)
	if alive > 0 {
		b[7] = clampF(bot.Stats.SumNeighborCount/alive/10.0, 0, 1)
	}

	return b
}

// BehaviorDistance computes Euclidean distance between two behavior vectors.
func BehaviorDistance(a, b BehaviorDescriptor) float64 {
	sum := 0.0
	for i := 0; i < BehaviorDims; i++ {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

// ComputeNoveltyScores calculates novelty for all bots.
// Novelty = average distance to k nearest neighbors in (archive + population).
func ComputeNoveltyScores(ss *SwarmState) []float64 {
	na := ss.NoveltyArchive
	if na == nil {
		return nil
	}
	n := len(ss.Bots)
	if n == 0 {
		return nil
	}

	// Build behavior pool: current population + archive
	behaviors := make([]BehaviorDescriptor, n)
	for i := range ss.Bots {
		behaviors[i] = ss.Bots[i].Behavior
	}
	pool := make([]BehaviorDescriptor, 0, n+len(na.Archive))
	pool = append(pool, behaviors...)
	pool = append(pool, na.Archive...)

	scores := make([]float64, n)
	maxScore := 0.0
	totalScore := 0.0

	for i := 0; i < n; i++ {
		scores[i] = kNearestAvgDist(behaviors[i], pool, na.KNeighbors)
		totalScore += scores[i]
		if scores[i] > maxScore {
			maxScore = scores[i]
		}
	}

	na.NoveltyScores = scores
	na.MaxNovelty = maxScore
	if n > 0 {
		na.AvgNovelty = totalScore / float64(n)
	}

	return scores
}

// UpdateNoveltyArchive adds novel behaviors to the archive.
func UpdateNoveltyArchive(ss *SwarmState, behaviors []BehaviorDescriptor, noveltyScores []float64) {
	na := ss.NoveltyArchive
	if na == nil {
		return
	}

	for i, score := range noveltyScores {
		if score >= na.AddThreshold && i < len(behaviors) {
			if len(na.Archive) < na.MaxSize {
				na.Archive = append(na.Archive, behaviors[i])
			}
		}
	}

	// Record history
	na.NoveltyHistory = append(na.NoveltyHistory, NoveltyRecord{
		AvgNovelty:  na.AvgNovelty,
		MaxNovelty:  na.MaxNovelty,
		ArchiveSize: len(na.Archive),
	})
}

// BlendFitness combines task fitness and novelty score.
// alpha=1.0 → pure task fitness, alpha=0.0 → pure novelty.
func BlendFitness(taskFitness, noveltyScore, alpha float64) float64 {
	if alpha > 1.0 {
		alpha = 1.0
	}
	if alpha < 0.0 {
		alpha = 0.0
	}
	return alpha*taskFitness + (1.0-alpha)*noveltyScore*100.0 // scale novelty to comparable range
}

// kNearestAvgDist returns average distance to k nearest neighbors in a pool.
// Excludes the point itself (distance 0).
func kNearestAvgDist(target BehaviorDescriptor, pool []BehaviorDescriptor, k int) float64 {
	if len(pool) == 0 || k <= 0 {
		return 0
	}

	distances := make([]float64, 0, len(pool))
	for _, p := range pool {
		d := BehaviorDistance(target, p)
		if d > 1e-10 { // skip self
			distances = append(distances, d)
		}
	}

	if len(distances) == 0 {
		return 0
	}

	sort.Float64s(distances)

	if k > len(distances) {
		k = len(distances)
	}

	sum := 0.0
	for i := 0; i < k; i++ {
		sum += distances[i]
	}
	return sum / float64(k)
}

// clampF clamps a float64 to [lo, hi].
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
