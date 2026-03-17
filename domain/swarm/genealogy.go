package swarm

// ═══════════════════════════════════════════════════════════
// GENEALOGIE — Stammbaum-Tracking ueber Generationen
// ═══════════════════════════════════════════════════════════
//
// Jeder Bot bekommt eine eindeutige ID und Referenzen auf
// seine Eltern. Der GenealogyTracker speichert die Historie
// als Ring-Buffer (max 50 Generationen) und berechnet:
//   - Lineage-Tiefe: Wie viele Generationen hat eine Linie ueberlebt?
//   - Extinctions: Wie viele Linien sind ausgestorben?
//   - Laengste aktive Linie

const genealogyMaxHistory = 50

// GenealogyRecord stores one bot's lineage for a single generation.
type GenealogyRecord struct {
	BotID      int
	ParentA    int // -1 = fresh/random
	ParentB    int // -1 = elite/fresh (only one parent)
	Generation int
	Fitness    float64
	IsElite    bool
	IsFresh    bool
}

// GenealogyTracker stores lineage history across generations.
type GenealogyTracker struct {
	Records        [][]GenealogyRecord // [generation mod MaxHistory][bot]
	MaxHistory     int
	WriteIdx       int // next write position in ring buffer
	Count          int // total generations recorded
	NextBotID      int
	TotalExtinct   int // cumulative extinct lineage count
	LongestLineage int
}

// InitGenealogy creates a new GenealogyTracker and assigns initial BotIDs.
func InitGenealogy(ss *SwarmState) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, genealogyMaxHistory),
		MaxHistory: genealogyMaxHistory,
		NextBotID:  0,
	}
	// Assign initial IDs (generation 0, all fresh)
	for i := range ss.Bots {
		ss.Bots[i].BotID = gt.NextBotID
		ss.Bots[i].ParentA = -1
		ss.Bots[i].ParentB = -1
		gt.NextBotID++
	}
	ss.Genealogy = gt
}

// AssignBotID returns the next unique bot ID.
func AssignBotID(gt *GenealogyTracker) int {
	if gt == nil {
		return -1
	}
	id := gt.NextBotID
	gt.NextBotID++
	return id
}

// RecordGeneration records all bots' lineage for the current generation.
func RecordGeneration(gt *GenealogyTracker, bots []SwarmBot, generation int) {
	if gt == nil || len(bots) == 0 {
		return
	}
	records := make([]GenealogyRecord, len(bots))
	for i := range bots {
		records[i] = GenealogyRecord{
			BotID:      bots[i].BotID,
			ParentA:    bots[i].ParentA,
			ParentB:    bots[i].ParentB,
			Generation: generation,
			Fitness:    bots[i].Fitness,
			IsElite:    bots[i].ParentA >= 0 && bots[i].ParentB == -1,
			IsFresh:    bots[i].ParentA == -1 && bots[i].ParentB == -1,
		}
	}
	gt.Records[gt.WriteIdx] = records
	gt.WriteIdx = (gt.WriteIdx + 1) % gt.MaxHistory
	gt.Count++

	// Update lineage metrics
	gt.LongestLineage, gt.TotalExtinct, _ = computeLineageStats(gt)
}

// ComputeLineageDepth computes how many consecutive generations a lineage survived.
// Traces parentA backwards through recorded generations.
func ComputeLineageDepth(gt *GenealogyTracker, botID int) int {
	if gt == nil || gt.Count == 0 {
		return 0
	}

	depth := 0
	currentID := botID

	// Walk backward through recorded generations
	for step := 0; step < gt.Count && step < gt.MaxHistory; step++ {
		genIdx := (gt.WriteIdx - 1 - step + gt.MaxHistory) % gt.MaxHistory
		records := gt.Records[genIdx]
		if records == nil {
			break
		}

		found := false
		for _, rec := range records {
			if rec.BotID == currentID {
				if rec.ParentA < 0 {
					return depth // fresh bot, end of lineage
				}
				currentID = rec.ParentA
				depth++
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return depth
}

// LineageStats returns summary statistics about the genealogy.
func LineageStats(gt *GenealogyTracker) (longestLineage int, extinctCount int, avgDepth float64) {
	if gt == nil {
		return 0, 0, 0
	}
	return gt.LongestLineage, gt.TotalExtinct, 0
}

// computeLineageStats computes lineage metrics from the recorded history.
func computeLineageStats(gt *GenealogyTracker) (longest int, extinct int, avgDepth float64) {
	if gt == nil || gt.Count < 2 {
		return 0, 0, 0
	}

	// Get the two most recent generations
	currIdx := (gt.WriteIdx - 1 + gt.MaxHistory) % gt.MaxHistory
	prevIdx := (gt.WriteIdx - 2 + gt.MaxHistory) % gt.MaxHistory

	currRecs := gt.Records[currIdx]
	prevRecs := gt.Records[prevIdx]

	if currRecs == nil || prevRecs == nil {
		return gt.LongestLineage, gt.TotalExtinct, 0
	}

	// Count extinctions: parent IDs from previous gen that appear nowhere in current gen
	prevIDs := make(map[int]bool, len(prevRecs))
	for _, rec := range prevRecs {
		prevIDs[rec.BotID] = true
	}

	// Collect all parent references in current generation
	referencedParents := make(map[int]bool)
	for _, rec := range currRecs {
		if rec.ParentA >= 0 {
			referencedParents[rec.ParentA] = true
		}
		if rec.ParentB >= 0 {
			referencedParents[rec.ParentB] = true
		}
	}

	newExtinct := 0
	for id := range prevIDs {
		if !referencedParents[id] {
			newExtinct++
		}
	}
	extinct = gt.TotalExtinct + newExtinct

	// Compute longest lineage from current generation
	longest = gt.LongestLineage
	totalDepth := 0
	for _, rec := range currRecs {
		d := ComputeLineageDepth(gt, rec.BotID)
		if d > longest {
			longest = d
		}
		totalDepth += d
	}
	if len(currRecs) > 0 {
		avgDepth = float64(totalDepth) / float64(len(currRecs))
	}

	return longest, extinct, avgDepth
}
