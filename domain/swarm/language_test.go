package swarm

import (
	"math/rand"
	"testing"
)

func TestInitLanguage(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitLanguage(ss)

	ls := ss.Language
	if ls == nil {
		t.Fatal("language should be initialized")
	}
	if len(ls.Vocabs) != 15 {
		t.Fatalf("expected 15 vocabs, got %d", len(ls.Vocabs))
	}
}

func TestClearLanguage(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.LanguageOn = true
	InitLanguage(ss)
	ClearLanguage(ss)

	if ss.Language != nil {
		t.Fatal("should be nil")
	}
	if ss.LanguageOn {
		t.Fatal("should be false")
	}
}

func TestTickLanguage(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitLanguage(ss)

	// Set varied contexts
	for i := 0; i < 5; i++ {
		ss.Bots[i].NearestPickupDist = 30
	}
	for i := 5; i < 10; i++ {
		ss.Bots[i].CarryingPkg = 0
	}

	for tick := 0; tick < 50; tick++ {
		TickLanguage(ss)
	}

	// Should have used some symbols
	totalFreq := 0
	for _, f := range ss.Language.SymbolFreq {
		totalFreq += f
	}
	if totalFreq == 0 {
		t.Fatal("should have broadcast some symbols")
	}
}

func TestTickLanguageNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickLanguage(ss) // should not panic
}

func TestEvolveSymbolLanguage(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitLanguage(ss)

	sorted := make([]int, 20)
	for i := range sorted {
		sorted[i] = i
	}

	EvolveSymbolLanguage(ss, sorted)
	if ss.Language.Generation != 1 {
		t.Fatalf("expected gen 1, got %d", ss.Language.Generation)
	}
}

func TestDetermineContext(t *testing.T) {
	bot := &SwarmBot{}

	bot.NearestPickupDist = 30
	bot.CarryingPkg = -1
	if determineContext(bot) != CtxFoundFood {
		t.Fatal("expected CtxFoundFood")
	}

	bot.NearestPickupDist = 200
	bot.CarryingPkg = 0
	bot.NearestDropoffDist = 50
	if determineContext(bot) != CtxNearDropoff {
		t.Fatal("expected CtxNearDropoff")
	}

	bot.NearestDropoffDist = 200
	if determineContext(bot) != CtxCarrying {
		t.Fatal("expected CtxCarrying")
	}
}

func TestSharedMeaning(t *testing.T) {
	if SharedMeaning(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBotCurrentSymbol(t *testing.T) {
	if BotCurrentSymbol(nil, 0) != -1 {
		t.Fatal("nil should return -1")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitLanguage(ss)
	TickLanguage(ss)

	sym := BotCurrentSymbol(ss.Language, 0)
	if sym < 0 || sym > 7 {
		t.Fatalf("symbol out of range: %d", sym)
	}
}

func TestChooseSymbol(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitLanguage(ss)

	vocab := &ss.Language.Vocabs[0]
	// Bias one symbol strongly
	vocab.Encode[0][3] = 10.0

	counts := [8]int{}
	for i := 0; i < 100; i++ {
		s := chooseSymbol(ss, vocab, 0)
		counts[s]++
	}

	// Symbol 3 should be most popular
	if counts[3] < 50 {
		t.Fatal("strongly biased symbol should be chosen most often")
	}
}
