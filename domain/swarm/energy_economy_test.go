package swarm

import (
	"math/rand"
	"testing"
)

func TestInitEnergyEconomy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitEnergyEconomy(ss)

	ee := ss.EnergyEconomy
	if ee == nil {
		t.Fatal("economy should be initialized")
	}
	if len(ee.Wallets) != 20 {
		t.Fatalf("expected 20 wallets, got %d", len(ee.Wallets))
	}
	if ee.Wallets[0].Energy != 50 {
		t.Fatalf("expected start energy 50, got %.0f", ee.Wallets[0].Energy)
	}
	if ee.TotalWealth != 1000 {
		t.Fatalf("expected total wealth 1000, got %.0f", ee.TotalWealth)
	}
}

func TestClearEnergyEconomy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.EnergyEconomyOn = true
	InitEnergyEconomy(ss)
	ClearEnergyEconomy(ss)

	if ss.EnergyEconomy != nil {
		t.Fatal("should be nil after clear")
	}
	if ss.EnergyEconomyOn {
		t.Fatal("should be false after clear")
	}
}

func TestTickEnergyEconomy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitEnergyEconomy(ss)

	// Set some bots moving
	for i := range ss.Bots {
		ss.Bots[i].Speed = SwarmBotSpeed
	}

	for tick := 0; tick < 100; tick++ {
		ss.Tick = tick
		TickEnergyEconomy(ss)
	}

	ee := ss.EnergyEconomy
	// Bots should have spent energy on movement
	for _, w := range ee.Wallets {
		if w.Spent <= 0 {
			t.Fatal("bots should have spent energy")
		}
	}
}

func TestTickEnergyEconomyNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickEnergyEconomy(ss) // should not panic
}

func TestEnergyEconomyTrading(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitEnergyEconomy(ss)
	ee := ss.EnergyEconomy

	// Place bots close and make some altruists with energy
	for i := range ss.Bots {
		ss.Bots[i].X = 400
		ss.Bots[i].Y = 400
	}
	ee.Wallets[0].Role = EconAltruist
	ee.Wallets[0].Energy = 100
	ee.Wallets[1].Energy = 10 // poor

	ss.Tick = 3 // trigger trading
	TickEnergyEconomy(ss)

	if ee.TradeCount == 0 {
		t.Fatal("trading should have occurred between close bots")
	}
}

func TestAdaptRoles(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitEnergyEconomy(ss)
	ee := ss.EnergyEconomy

	// Make a bot very poor
	ee.Wallets[0].Energy = 5
	adaptRoles(ss, ee)
	if ee.Wallets[0].Role != EconWorker {
		t.Fatal("poor bot should become worker")
	}

	// Make a bot very rich
	ee.Wallets[1].Energy = 180
	adaptRoles(ss, ee)
	if ee.Wallets[1].Role != EconAltruist && ee.Wallets[1].Role != EconHoarder {
		t.Fatal("rich bot should become altruist or hoarder")
	}
}

func TestUpdateEconomyStats(t *testing.T) {
	ee := &EnergyEconomyState{
		Wallets: []BotWallet{
			{Energy: 100},
			{Energy: 0},
		},
		TotalWealth: 100,
	}
	updateEconomyStats(ee)

	if ee.TotalWealth != 100 {
		t.Fatalf("expected 100, got %.0f", ee.TotalWealth)
	}
	if ee.GiniCoeff < 0.4 {
		t.Fatalf("gini should be high for unequal wealth, got %.3f", ee.GiniCoeff)
	}
}

func TestEconomyGini(t *testing.T) {
	if EconomyGini(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestEconomyTradeCount(t *testing.T) {
	if EconomyTradeCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestBotEnergy(t *testing.T) {
	if BotEnergy(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}
	ee := &EnergyEconomyState{
		Wallets: []BotWallet{{Energy: 42}},
	}
	if BotEnergy(ee, 0) != 42 {
		t.Fatal("expected 42")
	}
	if BotEnergy(ee, 5) != 0 {
		t.Fatal("out of bounds should return 0")
	}
}

func TestEconRoleName(t *testing.T) {
	if EconRoleName(EconWorker) != "Arbeiter" {
		t.Fatal("expected Arbeiter")
	}
	if EconRoleName(EconAltruist) != "Altruist" {
		t.Fatal("expected Altruist")
	}
	if EconRoleName(99) != "?" {
		t.Fatal("expected ?")
	}
}
