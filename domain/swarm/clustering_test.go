package swarm

import (
	"math/rand"
	"testing"
)

func TestComputeBehaviorClustersNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{Bots: make([]SwarmBot, 2), BotCount: 2, Rng: rng, ArenaW: 800, ArenaH: 800}
	// k > n → nil
	result := ComputeBehaviorClusters(ss, 5)
	if result != nil {
		t.Error("expected nil when k > n")
	}
}

func TestComputeBehaviorClustersKLessThan2(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{Bots: make([]SwarmBot, 10), BotCount: 10, Rng: rng, ArenaW: 800, ArenaH: 800}
	result := ComputeBehaviorClusters(ss, 1)
	if result != nil {
		t.Error("expected nil when k < 2")
	}
}

func TestComputeBehaviorClustersBasic(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	n := 20
	ss := &SwarmState{
		Bots:     make([]SwarmBot, n),
		BotCount: n,
		Rng:      rng,
		ArenaW:   800,
		ArenaH:   800,
	}

	// Give bots distinct positions for clustering
	for i := range ss.Bots {
		if i < 10 {
			ss.Bots[i].X = 100
			ss.Bots[i].Y = 100
		} else {
			ss.Bots[i].X = 700
			ss.Bots[i].Y = 700
		}
		ss.Bots[i].Stats.TicksAlive = 100
	}

	result := ComputeBehaviorClusters(ss, 3)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.K != 3 {
		t.Errorf("expected K=3, got %d", result.K)
	}

	// Check that all bots are assigned
	totalMembers := 0
	for _, cluster := range result.Clusters {
		totalMembers += len(cluster.Members)
	}
	if totalMembers != n {
		t.Errorf("expected %d total members, got %d", n, totalMembers)
	}
}

func TestComputeBehaviorClustersLabels(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	n := 30
	ss := &SwarmState{
		Bots:     make([]SwarmBot, n),
		BotCount: n,
		Rng:      rng,
		ArenaW:   800,
		ArenaH:   800,
	}
	for i := range ss.Bots {
		ss.Bots[i].Stats.TicksAlive = 200
		if i < 10 {
			// High deliveries → Lieferant
			ss.Bots[i].Stats.TotalDeliveries = 20
			ss.Bots[i].X = 400
			ss.Bots[i].Y = 400
		} else if i < 20 {
			// High idle → Wachposten
			ss.Bots[i].Stats.TicksIdle = 180
			ss.Bots[i].X = 100
			ss.Bots[i].Y = 100
		} else {
			// Explorers
			ss.Bots[i].Stats.TotalDistance = 5000
			ss.Bots[i].X = 750
			ss.Bots[i].Y = 50
		}
	}

	result := ComputeBehaviorClusters(ss, 3)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	labels := make(map[string]bool)
	for _, c := range result.Clusters {
		if c.Label != "" {
			labels[c.Label] = true
		}
	}
	// Should have at least 2 distinct labels
	if len(labels) < 2 {
		t.Errorf("expected at least 2 distinct labels, got %v", labels)
	}
}

func TestBehaviorDistSqZero(t *testing.T) {
	a := BehaviorDescriptor{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	d := behaviorDistSq(a, a)
	if d != 0 {
		t.Errorf("distance to self should be 0, got %f", d)
	}
}

func TestBehaviorDistSqSymmetric(t *testing.T) {
	a := BehaviorDescriptor{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	b := BehaviorDescriptor{0.9, 0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.2}
	if behaviorDistSq(a, b) != behaviorDistSq(b, a) {
		t.Error("distance should be symmetric")
	}
}

func TestClassifyCluster(t *testing.T) {
	// High deliveries
	c := BehaviorDescriptor{0.5, 0.5, 0.5, 0.8, 0.5, 0.3, 0.1, 0.5}
	if classifyCluster(c) != "Lieferant" {
		t.Errorf("expected Lieferant, got %s", classifyCluster(c))
	}

	// High idle
	c = BehaviorDescriptor{0.5, 0.5, 0.2, 0.0, 0.0, 0.0, 0.6, 0.5}
	if classifyCluster(c) != "Wachposten" {
		t.Errorf("expected Wachposten, got %s", classifyCluster(c))
	}
}
