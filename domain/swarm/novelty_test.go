package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestComputeBehaviorNormalized(t *testing.T) {
	ss := &SwarmState{ArenaW: 800, ArenaH: 800}
	bot := &SwarmBot{
		X: 400, Y: 200,
		Stats: BotLifetimeStats{
			TotalDistance:    3000,
			TotalDeliveries: 5,
			TotalPickups:    7,
			TicksAlive:      1000,
			TicksCarrying:   300,
			TicksIdle:       100,
			SumNeighborCount: 2000,
		},
	}

	b := ComputeBehavior(bot, ss)
	for i := 0; i < BehaviorDims; i++ {
		if b[i] < 0 || b[i] > 1.0 {
			t.Errorf("Dimension %d out of range [0,1]: %f", i, b[i])
		}
	}
}

func TestComputeBehaviorDifferentBots(t *testing.T) {
	ss := &SwarmState{ArenaW: 800, ArenaH: 800}
	bot1 := &SwarmBot{X: 100, Y: 100, Stats: BotLifetimeStats{TotalDistance: 500, TicksAlive: 100}}
	bot2 := &SwarmBot{X: 700, Y: 700, Stats: BotLifetimeStats{TotalDistance: 4000, TicksAlive: 100, TotalDeliveries: 8}}

	b1 := ComputeBehavior(bot1, ss)
	b2 := ComputeBehavior(bot2, ss)

	if b1 == b2 {
		t.Error("Different bots should have different behavior descriptors")
	}
}

func TestComputeBehaviorZeroAlive(t *testing.T) {
	ss := &SwarmState{ArenaW: 800, ArenaH: 800}
	bot := &SwarmBot{X: 100, Y: 100}

	b := ComputeBehavior(bot, ss)
	// Should not panic and fractions should be 0
	if b[5] != 0 || b[6] != 0 || b[7] != 0 {
		t.Errorf("Expected 0 for fraction dimensions with 0 TicksAlive, got %v", b)
	}
}

func TestBehaviorDistanceIdentical(t *testing.T) {
	a := BehaviorDescriptor{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	d := BehaviorDistance(a, a)
	if d != 0 {
		t.Errorf("Distance to self should be 0, got %f", d)
	}
}

func TestBehaviorDistanceSymmetric(t *testing.T) {
	a := BehaviorDescriptor{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	b := BehaviorDescriptor{0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.2, 0.1}
	dab := BehaviorDistance(a, b)
	dba := BehaviorDistance(b, a)
	if math.Abs(dab-dba) > 1e-10 {
		t.Errorf("Distance should be symmetric: %f != %f", dab, dba)
	}
}

func TestBehaviorDistanceMaxCorners(t *testing.T) {
	a := BehaviorDescriptor{0, 0, 0, 0, 0, 0, 0, 0}
	b := BehaviorDescriptor{1, 1, 1, 1, 1, 1, 1, 1}
	d := BehaviorDistance(a, b)
	expected := math.Sqrt(8)
	if math.Abs(d-expected) > 1e-10 {
		t.Errorf("Max distance should be sqrt(8)=%f, got %f", expected, d)
	}
}

func TestKNearestAvgDistSmallPool(t *testing.T) {
	target := BehaviorDescriptor{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	pool := []BehaviorDescriptor{
		{0.6, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5},
	}
	// k=15 but only 1 neighbor
	d := kNearestAvgDist(target, pool, 15)
	if d <= 0 {
		t.Error("Should return positive distance with 1 neighbor")
	}
}

func TestKNearestAvgDistEmptyPool(t *testing.T) {
	target := BehaviorDescriptor{}
	d := kNearestAvgDist(target, nil, 15)
	if d != 0 {
		t.Errorf("Empty pool should return 0, got %f", d)
	}
}

func TestComputeNoveltyScoresAllIdentical(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 10),
		ArenaW:   800,
		ArenaH:   800,
		Rng:      rng,
	}
	InitNovelty(ss)

	// All bots at same position with same stats → low novelty
	for i := range ss.Bots {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
		ss.Bots[i].Stats.TotalDistance = 1000
		ss.Bots[i].Stats.TicksAlive = 100
		ss.Bots[i].Behavior = ComputeBehavior(&ss.Bots[i], ss)
	}

	scores := ComputeNoveltyScores(ss)
	if scores == nil {
		t.Fatal("Scores should not be nil")
	}
	// All identical → all distances 0 → all novelty 0
	for i, s := range scores {
		if s > 0.01 {
			t.Errorf("Identical bots should have near-zero novelty, bot %d got %f", i, s)
		}
	}
}

func TestComputeNoveltyScoresDiverse(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 5),
		ArenaW:   800,
		ArenaH:   800,
		Rng:      rng,
	}
	InitNovelty(ss)

	// Diverse bots at different positions
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i) * 200
		ss.Bots[i].Y = float64(i) * 200
		ss.Bots[i].Stats.TotalDistance = float64(i) * 1000
		ss.Bots[i].Stats.TicksAlive = 100
		ss.Bots[i].Stats.TotalDeliveries = i * 3
		ss.Bots[i].Behavior = ComputeBehavior(&ss.Bots[i], ss)
	}

	scores := ComputeNoveltyScores(ss)
	if scores == nil {
		t.Fatal("Scores should not be nil")
	}
	// At least some novelty should be positive
	anyPositive := false
	for _, s := range scores {
		if s > 0 {
			anyPositive = true
		}
	}
	if !anyPositive {
		t.Error("Diverse bots should have some positive novelty scores")
	}
}

func TestUpdateArchiveGrows(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{Rng: rng}
	InitNovelty(ss)
	ss.NoveltyArchive.AddThreshold = 0.0 // accept everything

	behaviors := []BehaviorDescriptor{
		{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8},
		{0.9, 0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.2},
	}
	scores := []float64{1.0, 1.0}
	UpdateNoveltyArchive(ss, behaviors, scores)

	if len(ss.NoveltyArchive.Archive) != 2 {
		t.Errorf("Archive should have 2 entries, got %d", len(ss.NoveltyArchive.Archive))
	}
}

func TestUpdateArchiveMaxSize(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{Rng: rng}
	InitNovelty(ss)
	ss.NoveltyArchive.MaxSize = 3
	ss.NoveltyArchive.AddThreshold = 0.0

	behaviors := make([]BehaviorDescriptor, 10)
	scores := make([]float64, 10)
	for i := range behaviors {
		behaviors[i][0] = float64(i) / 10
		scores[i] = 1.0
	}
	UpdateNoveltyArchive(ss, behaviors, scores)

	if len(ss.NoveltyArchive.Archive) > 3 {
		t.Errorf("Archive should not exceed MaxSize 3, got %d", len(ss.NoveltyArchive.Archive))
	}
}

func TestBlendFitnessAlphaOne(t *testing.T) {
	result := BlendFitness(100, 0.5, 1.0)
	if result != 100 {
		t.Errorf("Alpha=1 should return pure task fitness 100, got %f", result)
	}
}

func TestBlendFitnessAlphaZero(t *testing.T) {
	result := BlendFitness(100, 0.5, 0.0)
	expected := 0.5 * 100.0 // novelty * 100 scaling
	if math.Abs(result-expected) > 1e-10 {
		t.Errorf("Alpha=0 should return pure novelty %f, got %f", expected, result)
	}
}

func TestBlendFitnessAlphaHalf(t *testing.T) {
	result := BlendFitness(100, 0.5, 0.5)
	expected := 0.5*100 + 0.5*0.5*100
	if math.Abs(result-expected) > 1e-10 {
		t.Errorf("Alpha=0.5 expected %f, got %f", expected, result)
	}
}

func TestNoveltyWithNilArchive(t *testing.T) {
	ss := &SwarmState{Bots: make([]SwarmBot, 5)}
	// Should not panic
	scores := ComputeNoveltyScores(ss)
	if scores != nil {
		t.Error("Nil archive should return nil scores")
	}
	UpdateNoveltyArchive(ss, nil, nil)
}

func TestInitNovelty(t *testing.T) {
	ss := &SwarmState{}
	InitNovelty(ss)
	if ss.NoveltyArchive == nil {
		t.Fatal("Archive should not be nil after init")
	}
	if ss.NoveltyArchive.MaxSize != 500 {
		t.Errorf("MaxSize should be 500, got %d", ss.NoveltyArchive.MaxSize)
	}
	if ss.NoveltyArchive.KNeighbors != 15 {
		t.Errorf("KNeighbors should be 15, got %d", ss.NoveltyArchive.KNeighbors)
	}
	if ss.NoveltyArchive.Alpha != 0.5 {
		t.Errorf("Alpha should be 0.5, got %f", ss.NoveltyArchive.Alpha)
	}
	if !ss.NoveltyEnabled {
		t.Error("NoveltyEnabled should be true after init")
	}
}

func TestClampF(t *testing.T) {
	if clampF(-1, 0, 1) != 0 {
		t.Error("clampF(-1, 0, 1) should be 0")
	}
	if clampF(2, 0, 1) != 1 {
		t.Error("clampF(2, 0, 1) should be 1")
	}
	if clampF(0.5, 0, 1) != 0.5 {
		t.Error("clampF(0.5, 0, 1) should be 0.5")
	}
}
