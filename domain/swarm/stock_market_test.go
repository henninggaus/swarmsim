package swarm

import (
	"math/rand"
	"testing"
)

func TestInitStockMarket(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitStockMarket(ss)

	sm := ss.StockMarket
	if sm == nil {
		t.Fatal("stock market should be initialized")
	}
	if len(sm.Stocks) != 5 {
		t.Fatalf("expected 5 stocks, got %d", len(sm.Stocks))
	}
	if len(sm.Portfolios) != 15 {
		t.Fatalf("expected 15 portfolios, got %d", len(sm.Portfolios))
	}
}

func TestClearStockMarket(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.StockMarketOn = true
	InitStockMarket(ss)
	ClearStockMarket(ss)

	if ss.StockMarket != nil {
		t.Fatal("should be nil")
	}
	if ss.StockMarketOn {
		t.Fatal("should be false")
	}
}

func TestTickStockMarket(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitStockMarket(ss)

	for i := range ss.Bots {
		ss.Bots[i].Speed = SwarmBotSpeed
		ss.Bots[i].NearestPickupDist = 50
	}

	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		TickStockMarket(ss)
	}

	sm := ss.StockMarket
	if sm.TotalTrades == 0 {
		t.Fatal("should have executed some trades")
	}
	if sm.MarketCap <= 0 {
		t.Fatal("market cap should be positive")
	}
}

func TestTickStockMarketNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickStockMarket(ss) // should not panic
}

func TestStockPriceFluctuations(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitStockMarket(ss)

	initialPrices := make([]float64, len(ss.StockMarket.Stocks))
	for i, s := range ss.StockMarket.Stocks {
		initialPrices[i] = s.Price
	}

	for tick := 0; tick < 500; tick++ {
		ss.Tick = tick
		TickStockMarket(ss)
	}

	// Prices should have changed
	changed := false
	for i, s := range ss.StockMarket.Stocks {
		if s.Price != initialPrices[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("prices should have fluctuated")
	}
}

func TestStockMarketCap(t *testing.T) {
	if StockMarketCap(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestStockTotalTrades(t *testing.T) {
	if StockTotalTrades(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestStockBiggestBubble(t *testing.T) {
	if StockBiggestBubble(nil) != 0 {
		t.Fatal("nil should return 0")
	}
}

func TestStockPrice(t *testing.T) {
	if StockPrice(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitStockMarket(ss)

	p := StockPrice(ss.StockMarket, 0)
	if p != 10.0 {
		t.Fatalf("expected initial price 10.0, got %.2f", p)
	}
	if StockPrice(ss.StockMarket, 99) != 0 {
		t.Fatal("out of bounds should return 0")
	}
}
