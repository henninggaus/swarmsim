package swarm

import (
	"math"
	"math/rand"
	"testing"
)

func TestDiploidExpress(t *testing.T) {
	g := &DiploidGenome{}
	g.AllelesA[0] = 10
	g.AllelesB[0] = 20
	if g.Express(0) != 15 {
		t.Errorf("expected average 15, got %f", g.Express(0))
	}
}

func TestDiploidCrossover(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	a := &DiploidGenome{}
	b := &DiploidGenome{}
	for p := 0; p < 26; p++ {
		a.AllelesA[p] = 100
		a.AllelesB[p] = 100
		b.AllelesA[p] = 0
		b.AllelesB[p] = 0
	}

	child := DiploidCrossover(rng, a, b)
	// Child should have mix of 0 and 100 alleles
	has100 := false
	has0 := false
	for p := 0; p < 26; p++ {
		if child.AllelesA[p] == 100 || child.AllelesB[p] == 100 {
			has100 = true
		}
		if child.AllelesA[p] == 0 || child.AllelesB[p] == 0 {
			has0 = true
		}
	}
	if !has100 || !has0 {
		t.Error("child should have alleles from both parents")
	}
}

func TestMutateDiploid(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := &DiploidGenome{}
	for p := 0; p < 26; p++ {
		g.AllelesA[p] = 50
		g.AllelesB[p] = 50
	}

	MutateDiploid(rng, g, 1.0, 5.0) // 100% mutation rate

	// At least some alleles should have changed
	changed := 0
	for p := 0; p < 26; p++ {
		if math.Abs(g.AllelesA[p]-50) > 0.01 {
			changed++
		}
	}
	if changed == 0 {
		t.Error("mutation should change at least some alleles")
	}
}

func TestMateSelectionNoMates(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 5),
		BotCount: 5,
		Rng:      rng,
		ArenaW:   800,
		ArenaH:   800,
	}
	// Scatter bots far apart
	for i := range ss.Bots {
		ss.Bots[i].X = float64(i) * 500
		ss.Bots[i].Y = 400
		ss.Bots[i].Stats.TicksAlive = 100
	}

	mate := MateSelection(ss, 0, 50) // very short range
	if mate != -1 {
		t.Errorf("expected no mate at short range, got %d", mate)
	}
}

func TestMateSelectionFindsNearby(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{
		Bots:     make([]SwarmBot, 5),
		BotCount: 5,
		Rng:      rng,
		ArenaW:   800,
		ArenaH:   800,
	}
	// Cluster bots together
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*10
		ss.Bots[i].Y = 400
		ss.Bots[i].Stats.TicksAlive = 100
		ss.Bots[i].Stats.TotalDeliveries = i + 1
	}

	mate := MateSelection(ss, 0, 200)
	if mate == -1 {
		t.Error("expected to find a mate")
	}
	if mate == 0 {
		t.Error("mate should not be self")
	}
}

func TestRunSexualEvolutionMinBots(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := &SwarmState{Bots: make([]SwarmBot, 2), BotCount: 2, Rng: rng}
	RunSexualEvolution(ss) // should not panic
}

func TestRunSexualEvolutionBasic(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	n := 20
	ss := &SwarmState{
		Bots:     make([]SwarmBot, n),
		BotCount: n,
		Rng:      rng,
		ArenaW:   800,
		ArenaH:   800,
	}

	// Setup bots with positions and stats
	for i := range ss.Bots {
		ss.Bots[i].X = 400 + float64(i)*10
		ss.Bots[i].Y = 400
		ss.Bots[i].Stats.TicksAlive = 100
		ss.Bots[i].Stats.TotalDeliveries = rng.Intn(10)
		for p := 0; p < 26; p++ {
			ss.Bots[i].ParamValues[p] = rng.Float64() * 100
		}
	}

	RunSexualEvolution(ss)

	// All bots should have diploid genomes
	for i := range ss.Bots {
		if ss.Bots[i].DiploidGenome == nil {
			t.Errorf("bot %d should have diploid genome", i)
		}
	}

	// Generation should increment
	if ss.Generation != 1 {
		t.Errorf("expected generation 1, got %d", ss.Generation)
	}
}

func TestDiploidExpressCoDominance(t *testing.T) {
	g := &DiploidGenome{}
	g.AllelesA[5] = 30
	g.AllelesB[5] = 70
	expressed := g.Express(5)
	if math.Abs(expressed-50) > 0.01 {
		t.Errorf("co-dominance: expected 50, got %f", expressed)
	}
}
