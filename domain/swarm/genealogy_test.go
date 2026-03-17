package swarm

import (
	"math/rand"
	"testing"
)

func TestAssignBotIDMonotonic(t *testing.T) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, genealogyMaxHistory),
		MaxHistory: genealogyMaxHistory,
		NextBotID:  0,
	}
	ids := make([]int, 100)
	for i := range ids {
		ids[i] = AssignBotID(gt)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("BotID not monotonic: ids[%d]=%d <= ids[%d]=%d", i, ids[i], i-1, ids[i-1])
		}
	}
}

func TestAssignBotIDNilTracker(t *testing.T) {
	id := AssignBotID(nil)
	if id != -1 {
		t.Errorf("Expected -1 for nil tracker, got %d", id)
	}
}

func TestRecordGenerationSize(t *testing.T) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, genealogyMaxHistory),
		MaxHistory: genealogyMaxHistory,
		NextBotID:  10,
	}
	bots := make([]SwarmBot, 5)
	for i := range bots {
		bots[i].BotID = i
		bots[i].ParentA = -1
		bots[i].ParentB = -1
	}
	RecordGeneration(gt, bots, 0)

	if gt.Count != 1 {
		t.Errorf("Expected Count=1, got %d", gt.Count)
	}
	records := gt.Records[0]
	if len(records) != 5 {
		t.Errorf("Expected 5 records, got %d", len(records))
	}
}

func TestRecordGenerationRingBuffer(t *testing.T) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, 3), // small buffer for testing
		MaxHistory: 3,
		NextBotID:  0,
	}
	bots := []SwarmBot{{BotID: 0, ParentA: -1, ParentB: -1}}

	// Record 5 generations into a buffer of size 3
	for g := 0; g < 5; g++ {
		bots[0].BotID = g
		RecordGeneration(gt, bots, g)
	}

	if gt.Count != 5 {
		t.Errorf("Expected Count=5, got %d", gt.Count)
	}
	// Oldest should be overwritten
	// WriteIdx should be 5 % 3 = 2
	if gt.WriteIdx != 2 {
		t.Errorf("Expected WriteIdx=2, got %d", gt.WriteIdx)
	}
}

func TestLineageDepthFreshBot(t *testing.T) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, genealogyMaxHistory),
		MaxHistory: genealogyMaxHistory,
		NextBotID:  1,
		Count:      1,
		WriteIdx:   1,
	}
	gt.Records[0] = []GenealogyRecord{
		{BotID: 0, ParentA: -1, ParentB: -1, Generation: 0},
	}

	depth := ComputeLineageDepth(gt, 0)
	if depth != 0 {
		t.Errorf("Fresh bot should have depth 0, got %d", depth)
	}
}

func TestLineageDepthWithParents(t *testing.T) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, genealogyMaxHistory),
		MaxHistory: genealogyMaxHistory,
		NextBotID:  3,
		Count:      3,
		WriteIdx:   3,
	}
	// Gen 0: Bot 0 (fresh)
	gt.Records[0] = []GenealogyRecord{
		{BotID: 0, ParentA: -1, ParentB: -1, Generation: 0},
	}
	// Gen 1: Bot 1 (child of Bot 0)
	gt.Records[1] = []GenealogyRecord{
		{BotID: 1, ParentA: 0, ParentB: -1, Generation: 1},
	}
	// Gen 2: Bot 2 (child of Bot 1)
	gt.Records[2] = []GenealogyRecord{
		{BotID: 2, ParentA: 1, ParentB: -1, Generation: 2},
	}

	depth := ComputeLineageDepth(gt, 2)
	if depth != 2 {
		t.Errorf("Expected depth 2, got %d", depth)
	}
}

func TestLineageDepthNilTracker(t *testing.T) {
	depth := ComputeLineageDepth(nil, 0)
	if depth != 0 {
		t.Errorf("Expected 0 for nil tracker, got %d", depth)
	}
}

func TestLineageStatsNilTracker(t *testing.T) {
	longest, extinct, avg := LineageStats(nil)
	if longest != 0 || extinct != 0 || avg != 0 {
		t.Errorf("Expected all zeros for nil tracker, got %d, %d, %f", longest, extinct, avg)
	}
}

func TestInitGenealogy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:    make([]SwarmBot, 10),
		BotCount: 10,
		Rng:     rng,
	}
	InitGenealogy(ss)

	if ss.Genealogy == nil {
		t.Fatal("Genealogy should not be nil after init")
	}
	if ss.Genealogy.NextBotID != 10 {
		t.Errorf("Expected NextBotID=10, got %d", ss.Genealogy.NextBotID)
	}
	for i := range ss.Bots {
		if ss.Bots[i].BotID != i {
			t.Errorf("Bot %d should have ID %d, got %d", i, i, ss.Bots[i].BotID)
		}
		if ss.Bots[i].ParentA != -1 || ss.Bots[i].ParentB != -1 {
			t.Errorf("Bot %d should have no parents", i)
		}
	}
}

func TestRecordGenerationNilTracker(t *testing.T) {
	// Should not panic
	bots := []SwarmBot{{BotID: 0}}
	RecordGeneration(nil, bots, 0)
}

func TestRecordGenerationEmptyBots(t *testing.T) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, genealogyMaxHistory),
		MaxHistory: genealogyMaxHistory,
	}
	// Should not panic
	RecordGeneration(gt, nil, 0)
	RecordGeneration(gt, []SwarmBot{}, 0)
	if gt.Count != 0 {
		t.Errorf("Expected Count=0 for empty bots, got %d", gt.Count)
	}
}

func TestExtinctionCounting(t *testing.T) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, genealogyMaxHistory),
		MaxHistory: genealogyMaxHistory,
		NextBotID:  0,
	}

	// Gen 0: 3 fresh bots
	bots := make([]SwarmBot, 3)
	for i := range bots {
		bots[i].BotID = AssignBotID(gt)
		bots[i].ParentA = -1
		bots[i].ParentB = -1
	}
	RecordGeneration(gt, bots, 0)

	// Gen 1: Bot 3 descends from Bot 0, Bot 4 descends from Bot 1, Bot 2 goes extinct
	bots[0].BotID = AssignBotID(gt)
	bots[0].ParentA = 0
	bots[0].ParentB = -1
	bots[1].BotID = AssignBotID(gt)
	bots[1].ParentA = 1
	bots[1].ParentB = -1
	bots[2].BotID = AssignBotID(gt)
	bots[2].ParentA = -1 // fresh, not descending from Bot 2
	bots[2].ParentB = -1
	RecordGeneration(gt, bots, 1)

	// Bot 2 from gen 0 should be extinct (not referenced as parent)
	if gt.TotalExtinct < 1 {
		t.Errorf("Expected at least 1 extinction, got %d", gt.TotalExtinct)
	}
}

func TestGenealogyRecordIsEliteIsFresh(t *testing.T) {
	gt := &GenealogyTracker{
		Records:    make([][]GenealogyRecord, genealogyMaxHistory),
		MaxHistory: genealogyMaxHistory,
		NextBotID:  0,
	}

	bots := []SwarmBot{
		{BotID: 0, ParentA: 5, ParentB: -1},  // elite (one parent)
		{BotID: 1, ParentA: -1, ParentB: -1},  // fresh
		{BotID: 2, ParentA: 3, ParentB: 4},    // crossover
	}
	RecordGeneration(gt, bots, 0)

	records := gt.Records[0]
	if !records[0].IsElite {
		t.Error("Bot 0 should be marked as elite")
	}
	if records[0].IsFresh {
		t.Error("Bot 0 should not be marked as fresh")
	}
	if !records[1].IsFresh {
		t.Error("Bot 1 should be marked as fresh")
	}
	if records[1].IsElite {
		t.Error("Bot 1 should not be marked as elite")
	}
	if records[2].IsElite || records[2].IsFresh {
		t.Error("Bot 2 should be neither elite nor fresh")
	}
}
