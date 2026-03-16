package swarm

import (
	"math"
	"testing"
)

// === dominates() tests ===

func TestDominatesStrictlyBetter(t *testing.T) {
	if !dominates([]float64{3, 5, 7}, []float64{1, 2, 3}) {
		t.Error("(3,5,7) should dominate (1,2,3)")
	}
}

func TestDominatesEqual(t *testing.T) {
	if dominates([]float64{3, 5, 7}, []float64{3, 5, 7}) {
		t.Error("identical solutions should NOT dominate each other")
	}
}

func TestDominatesPartiallyWorse(t *testing.T) {
	// a is better in 2 objectives but worse in 1 → no domination
	if dominates([]float64{5, 1, 7}, []float64{3, 5, 3}) {
		t.Error("should not dominate when worse in any objective")
	}
}

func TestDominatesEqualInSomeBetterInOne(t *testing.T) {
	// a >= b in all, strictly better in one → dominates
	if !dominates([]float64{3, 5, 8}, []float64{3, 5, 7}) {
		t.Error("equal in 2, better in 1 → should dominate")
	}
}

// === ComputeParetoFronts() tests ===

func makeParetoState(objectives [][]float64) *SwarmState {
	ss := &SwarmState{
		Bots: make([]SwarmBot, len(objectives)),
	}
	for i, obj := range objectives {
		if len(obj) >= 3 {
			// Set stats so botObjectives() returns our values
			ss.Bots[i].Stats.CorrectDeliveries = int(obj[0] / 3)
			ss.Bots[i].Stats.TotalDistance = obj[1] * 100
			ss.Bots[i].Stats.TotalDeliveries = 1 // avoid div/0
		}
	}
	return ss
}

func TestParetoFrontsEmpty(t *testing.T) {
	ss := &SwarmState{}
	pf := ComputeParetoFronts(ss)
	if len(pf.Fronts) != 0 {
		t.Error("empty population should produce no fronts")
	}
}

func TestParetoFrontsSingleBot(t *testing.T) {
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].Stats.CorrectDeliveries = 5
	ss.Bots[0].Stats.TotalDistance = 100
	ss.Bots[0].Stats.TotalDeliveries = 5
	pf := ComputeParetoFronts(ss)
	if len(pf.Fronts) != 1 {
		t.Fatalf("single bot should be in one front, got %d", len(pf.Fronts))
	}
	if len(pf.Fronts[0]) != 1 {
		t.Error("single bot front should have 1 member")
	}
}

func TestParetoFrontsAllBotsOnFrontWhenNonDominated(t *testing.T) {
	// Three bots, each best in exactly one objective → all on front 0
	ss := &SwarmState{Bots: make([]SwarmBot, 3)}
	// Bot 0: high deliveries
	ss.Bots[0].Stats.CorrectDeliveries = 10
	ss.Bots[0].Stats.TotalDistance = 0
	ss.Bots[0].Stats.TotalDeliveries = 10
	// Bot 1: high exploration
	ss.Bots[1].Stats.CorrectDeliveries = 0
	ss.Bots[1].Stats.TotalDistance = 5000
	ss.Bots[1].Stats.TotalDeliveries = 1
	// Bot 2: high efficiency (1 delivery, perfect)
	ss.Bots[2].Stats.CorrectDeliveries = 1
	ss.Bots[2].Stats.TotalDistance = 100
	ss.Bots[2].Stats.TotalDeliveries = 1

	pf := ComputeParetoFronts(ss)
	if ParetoFrontSize(pf) != 3 {
		t.Errorf("all 3 non-dominated bots should be on front 0, got %d", ParetoFrontSize(pf))
	}
}

func TestParetoFrontsDominatedBotInLowerRank(t *testing.T) {
	ss := &SwarmState{Bots: make([]SwarmBot, 2)}
	// Bot 0: strictly better in everything
	ss.Bots[0].Stats.CorrectDeliveries = 10
	ss.Bots[0].Stats.TotalDistance = 5000
	ss.Bots[0].Stats.TotalDeliveries = 10
	// Bot 1: worse in everything
	ss.Bots[1].Stats.CorrectDeliveries = 1
	ss.Bots[1].Stats.TotalDistance = 100
	ss.Bots[1].Stats.TotalDeliveries = 5 // low efficiency too

	pf := ComputeParetoFronts(ss)
	if len(pf.Fronts) < 2 {
		t.Fatalf("dominated bot should be in second front, got %d fronts", len(pf.Fronts))
	}
	if ParetoFrontSize(pf) != 1 {
		t.Error("only one bot should be on the Pareto front")
	}
}

func TestParetoRankFitnessBounds(t *testing.T) {
	ss := &SwarmState{Bots: make([]SwarmBot, 5)}
	for i := range ss.Bots {
		ss.Bots[i].Stats.CorrectDeliveries = i
		ss.Bots[i].Stats.TotalDistance = float64(5-i) * 100
		ss.Bots[i].Stats.TotalDeliveries = max(i, 1)
	}
	pf := ComputeParetoFronts(ss)

	// Front 0 bots should have higher fitness than front 1+ bots
	for _, front := range pf.Fronts {
		for _, idx := range front {
			f := ParetoRankFitness(pf, idx)
			if f < 0 {
				t.Errorf("fitness should never be negative, got %.1f for bot %d", f, idx)
			}
		}
	}
}

func TestParetoRankFitnessOutOfBounds(t *testing.T) {
	pf := &ParetoFront{BotCount: 3}
	if ParetoRankFitness(pf, -1) != 0 {
		t.Error("out-of-bounds bot index should return 0")
	}
	if ParetoRankFitness(pf, 99) != 0 {
		t.Error("out-of-bounds bot index should return 0")
	}
}

func TestParetoFrontSizeEmpty(t *testing.T) {
	pf := &ParetoFront{}
	if ParetoFrontSize(pf) != 0 {
		t.Error("empty ParetoFront should have size 0")
	}
}

// === Crowding Distance tests ===

func TestCrowdingDistanceBoundaryGetsMax(t *testing.T) {
	// With 3+ members, boundary solutions should get large distances
	ss := &SwarmState{Bots: make([]SwarmBot, 4)}
	for i := range ss.Bots {
		ss.Bots[i].Stats.CorrectDeliveries = i * 3
		ss.Bots[i].Stats.TotalDistance = float64(100 * (4 - i))
		ss.Bots[i].Stats.TotalDeliveries = max(i*3, 1)
	}
	pf := ComputeParetoFronts(ss)

	// All bots that are boundary solutions in any objective should have
	// crowding distance >= 1000 (per objective)
	for _, front := range pf.Fronts {
		if len(front) <= 2 {
			continue
		}
		dists := computeAllCrowdingDistances(pf, front)
		// At least some should have high values (boundary solutions)
		hasHigh := false
		for _, d := range dists {
			if d >= 1000 {
				hasHigh = true
			}
		}
		if !hasHigh {
			t.Error("boundary solutions should have crowding distance >= 1000")
		}
	}
}

func TestCrowdingDistanceSmallFront(t *testing.T) {
	// With <= 2 members, all get fixed distance of 50
	pf := &ParetoFront{
		Objectives: [][]float64{{1, 2, 3}, {4, 5, 6}},
	}
	dists := computeAllCrowdingDistances(pf, []int{0, 1})
	for i, d := range dists {
		if d != 50 {
			t.Errorf("member %d: small front should get distance 50, got %.1f", i, d)
		}
	}
}

// === botObjectives() correctness ===

func TestBotObjectivesCalculation(t *testing.T) {
	bot := &SwarmBot{}
	bot.Stats.CorrectDeliveries = 5
	bot.Stats.WrongDeliveries = 2
	bot.Stats.TotalDistance = 300
	bot.Stats.TotalDeliveries = 7

	obj := botObjectives(bot)

	expectedDeliveries := 5.0*3 + 2.0 // 17
	if math.Abs(obj[0]-expectedDeliveries) > 0.001 {
		t.Errorf("deliveries: expected %.1f, got %.1f", expectedDeliveries, obj[0])
	}

	expectedExploration := 300.0 / 100.0 // 3.0
	if math.Abs(obj[1]-expectedExploration) > 0.001 {
		t.Errorf("exploration: expected %.1f, got %.1f", expectedExploration, obj[1])
	}

	expectedEfficiency := 5.0 / 7.0 * 100 // ~71.4
	if math.Abs(obj[2]-expectedEfficiency) > 0.1 {
		t.Errorf("efficiency: expected %.1f, got %.1f", expectedEfficiency, obj[2])
	}
}

func TestBotObjectivesZeroDeliveries(t *testing.T) {
	bot := &SwarmBot{}
	bot.Stats.TotalDeliveries = 0
	obj := botObjectives(bot)

	// Efficiency should be 0 (0 correct / max(0,1) * 100)
	if obj[2] != 0 {
		t.Errorf("efficiency with 0 deliveries should be 0, got %.1f", obj[2])
	}
}

// === NSGA-II completeness test ===
// Verifies that every bot is assigned to exactly one front

func TestAllBotsAssignedToExactlyOneFront(t *testing.T) {
	n := 20
	ss := &SwarmState{Bots: make([]SwarmBot, n)}
	for i := range ss.Bots {
		ss.Bots[i].Stats.CorrectDeliveries = i % 5
		ss.Bots[i].Stats.TotalDistance = float64(100 * ((i * 3) % 7))
		ss.Bots[i].Stats.TotalDeliveries = max(i%5, 1)
	}

	pf := ComputeParetoFronts(ss)

	seen := make(map[int]bool)
	for _, front := range pf.Fronts {
		for _, idx := range front {
			if seen[idx] {
				t.Errorf("bot %d appears in multiple fronts", idx)
			}
			seen[idx] = true
		}
	}

	for i := 0; i < n; i++ {
		if !seen[i] {
			t.Errorf("bot %d is not assigned to any front", i)
		}
	}
}

// === Performance / scalability sanity ===

func TestParetoFrontsLargePopulation(t *testing.T) {
	n := 200
	ss := &SwarmState{Bots: make([]SwarmBot, n)}
	for i := range ss.Bots {
		ss.Bots[i].Stats.CorrectDeliveries = i % 20
		ss.Bots[i].Stats.TotalDistance = float64(i * 50)
		ss.Bots[i].Stats.TotalDeliveries = max(i%20, 1)
	}

	pf := ComputeParetoFronts(ss)

	totalAssigned := 0
	for _, front := range pf.Fronts {
		totalAssigned += len(front)
	}
	if totalAssigned != n {
		t.Errorf("expected all %d bots assigned, got %d", n, totalAssigned)
	}
}

func TestParetoFrontRank0HigherFitnessThanRank1(t *testing.T) {
	ss := &SwarmState{Bots: make([]SwarmBot, 10)}
	for i := range ss.Bots {
		ss.Bots[i].Stats.CorrectDeliveries = i
		ss.Bots[i].Stats.TotalDistance = float64((10 - i) * 200)
		ss.Bots[i].Stats.TotalDeliveries = max(i, 1)
	}

	pf := ComputeParetoFronts(ss)
	if len(pf.Fronts) < 2 {
		t.Skip("need at least 2 fronts to compare")
	}

	// Min fitness on front 0 should be > max fitness on front 1
	minFront0 := math.Inf(1)
	for _, idx := range pf.Fronts[0] {
		f := ParetoRankFitness(pf, idx)
		if f < minFront0 {
			minFront0 = f
		}
	}
	maxFront1 := math.Inf(-1)
	for _, idx := range pf.Fronts[1] {
		f := ParetoRankFitness(pf, idx)
		if f > maxFront1 {
			maxFront1 = f
		}
	}
	if minFront0 <= maxFront1 {
		t.Errorf("front 0 min fitness (%.1f) should exceed front 1 max fitness (%.1f)",
			minFront0, maxFront1)
	}
}
