package swarm

import (
	"math"
	"swarmsim/locale"
	"swarmsim/logger"
)

// StockMarketState manages a strategy stock market in the swarm.
// Bots trade "shares" of behavioral strategies. Successful strategies
// rise in price, others fall. Bots invest their fitness. Bubbles and
// crashes can emerge — a complete financial market simulation.
type StockMarketState struct {
	Stocks     []StrategyStock    // available strategy stocks
	Portfolios []BotPortfolio     // per-bot portfolio
	OrderBook  []MarketOrder      // pending orders

	// Market parameters
	TickInterval int     // ticks between market cycles (default 10)
	Volatility   float64 // price volatility (default 0.05)
	MaxShares    int     // max shares per bot per stock (default 10)

	// Stats
	MarketCap     float64 // total market capitalization
	TotalTrades   int
	BiggestBubble float64 // highest ever price
	BiggestCrash  float64 // biggest single-tick drop
	MarketCycle   int
}

// StrategyStock is a tradeable strategy.
type StrategyStock struct {
	Name        string
	Price       float64 // current price
	PrevPrice   float64
	Supply      int     // total shares outstanding
	Demand      int     // buy orders this cycle
	Performance float64 // recent strategy effectiveness
	PriceHistory []float64 // last 50 prices
}

// BotPortfolio tracks one bot's investments.
type BotPortfolio struct {
	Holdings []int     // shares held per stock
	Cash     float64   // available investment points
	NetWorth float64   // total portfolio value
	Strategy int       // which strategy this bot follows (-1 = none)
}

// MarketOrder is a buy/sell order.
type MarketOrder struct {
	BotIdx   int
	StockIdx int
	IsBuy    bool
	Shares   int
}

var strategyStocks = []StrategyStock{
	{Name: "Aggressive", Price: 10.0},
	{Name: "Defensive", Price: 10.0},
	{Name: "Explorer", Price: 10.0},
	{Name: "Cooperative", Price: 10.0},
	{Name: "Specialist", Price: 10.0},
}

// stockLocaleKeys maps internal strategy stock names to locale keys.
var stockLocaleKeys = map[string]string{
	"Aggressive":  "stock.aggressive",
	"Defensive":   "stock.defensive",
	"Explorer":    "stock.explorer",
	"Cooperative": "stock.cooperative",
	"Specialist":  "stock.specialist",
}

// StockDisplayName returns the localized display name for a StrategyStock.
func StockDisplayName(s *StrategyStock) string {
	if key, ok := stockLocaleKeys[s.Name]; ok {
		return locale.T(key)
	}
	return s.Name
}

// InitStockMarket sets up the strategy stock market.
func InitStockMarket(ss *SwarmState) {
	n := len(ss.Bots)
	sm := &StockMarketState{
		Stocks:       make([]StrategyStock, len(strategyStocks)),
		Portfolios:   make([]BotPortfolio, n),
		OrderBook:    make([]MarketOrder, 0, 100),
		TickInterval: 10,
		Volatility:   0.05,
		MaxShares:    10,
	}

	for i := range strategyStocks {
		sm.Stocks[i] = strategyStocks[i]
		sm.Stocks[i].PriceHistory = make([]float64, 0, 50)
		sm.Stocks[i].Supply = n * 5
	}

	for i := 0; i < n; i++ {
		sm.Portfolios[i] = BotPortfolio{
			Holdings: make([]int, len(sm.Stocks)),
			Cash:     50.0,
			Strategy: -1,
		}
		// Give initial holdings
		favStock := ss.Rng.Intn(len(sm.Stocks))
		sm.Portfolios[i].Holdings[favStock] = 2
		sm.Portfolios[i].Strategy = favStock
	}

	ss.StockMarket = sm
	logger.Info("BOERSE", "Initialisiert: %d Aktien, %d Haendler", len(sm.Stocks), n)
}

// ClearStockMarket disables the stock market.
func ClearStockMarket(ss *SwarmState) {
	ss.StockMarket = nil
	ss.StockMarketOn = false
}

// TickStockMarket runs one tick of the market.
func TickStockMarket(ss *SwarmState) {
	sm := ss.StockMarket
	if sm == nil {
		return
	}

	n := len(ss.Bots)
	if len(sm.Portfolios) != n {
		return
	}

	if ss.Tick%sm.TickInterval != 0 {
		return
	}

	sm.MarketCycle++

	// Phase 1: Evaluate strategy performance
	evaluateStrategies(ss, sm)

	// Phase 2: Generate orders
	sm.OrderBook = sm.OrderBook[:0]
	generateOrders(ss, sm)

	// Phase 3: Execute orders and update prices
	executeOrders(ss, sm)

	// Phase 4: Update portfolios and apply strategies
	updatePortfolios(sm)
	applyStockStrategies(ss, sm)

	// Phase 5: Record history
	for i := range sm.Stocks {
		s := &sm.Stocks[i]
		s.PriceHistory = append(s.PriceHistory, s.Price)
		if len(s.PriceHistory) > 50 {
			s.PriceHistory = s.PriceHistory[1:]
		}
		if s.Price > sm.BiggestBubble {
			sm.BiggestBubble = s.Price
		}
		drop := s.PrevPrice - s.Price
		if drop > sm.BiggestCrash {
			sm.BiggestCrash = drop
		}
	}
}

// evaluateStrategies scores how well each strategy is performing.
func evaluateStrategies(ss *SwarmState, sm *StockMarketState) {
	numStocks := len(sm.Stocks)
	perfCounts := make([]int, numStocks)
	perfSums := make([]float64, numStocks)

	for i := range ss.Bots {
		strat := sm.Portfolios[i].Strategy
		if strat < 0 || strat >= numStocks {
			continue
		}

		// Bot fitness proxy
		fitness := 0.0
		if ss.Bots[i].CarryingPkg >= 0 && ss.Bots[i].NearestDropoffDist < 100 {
			fitness = 0.8
		} else if ss.Bots[i].NearestPickupDist < 80 {
			fitness = 0.4
		}
		fitness += ss.Bots[i].Speed / SwarmBotSpeed * 0.2

		perfSums[strat] += fitness
		perfCounts[strat]++
	}

	for i := range sm.Stocks {
		if perfCounts[i] > 0 {
			sm.Stocks[i].Performance = perfSums[i] / float64(perfCounts[i])
		} else {
			sm.Stocks[i].Performance *= 0.9 // decay
		}
	}
}

// generateOrders creates buy/sell orders from bots.
func generateOrders(ss *SwarmState, sm *StockMarketState) {
	for i := range ss.Bots {
		port := &sm.Portfolios[i]

		// Buy strategy with best performance
		bestStock := 0
		for s := 1; s < len(sm.Stocks); s++ {
			if sm.Stocks[s].Performance > sm.Stocks[bestStock].Performance {
				bestStock = s
			}
		}

		// Buy order if affordable
		if port.Cash >= sm.Stocks[bestStock].Price && port.Holdings[bestStock] < sm.MaxShares {
			sm.OrderBook = append(sm.OrderBook, MarketOrder{
				BotIdx:   i,
				StockIdx: bestStock,
				IsBuy:    true,
				Shares:   1,
			})
			sm.Stocks[bestStock].Demand++
		}

		// Sell worst-performing stock
		worstStock := -1
		worstPerf := math.MaxFloat64
		for s := range sm.Stocks {
			if port.Holdings[s] > 0 && sm.Stocks[s].Performance < worstPerf {
				worstPerf = sm.Stocks[s].Performance
				worstStock = s
			}
		}

		if worstStock >= 0 && ss.Rng.Float64() < 0.3 {
			sm.OrderBook = append(sm.OrderBook, MarketOrder{
				BotIdx:   i,
				StockIdx: worstStock,
				IsBuy:    false,
				Shares:   1,
			})
		}
	}
}

// executeOrders processes market orders and updates prices.
func executeOrders(ss *SwarmState, sm *StockMarketState) {
	for _, order := range sm.OrderBook {
		stock := &sm.Stocks[order.StockIdx]
		port := &sm.Portfolios[order.BotIdx]

		if order.IsBuy {
			cost := stock.Price * float64(order.Shares)
			if port.Cash >= cost {
				port.Cash -= cost
				port.Holdings[order.StockIdx] += order.Shares
				sm.TotalTrades++
			}
		} else {
			if port.Holdings[order.StockIdx] >= order.Shares {
				port.Holdings[order.StockIdx] -= order.Shares
				port.Cash += stock.Price * float64(order.Shares)
				sm.TotalTrades++
			}
		}
	}

	// Update prices based on supply/demand
	for i := range sm.Stocks {
		stock := &sm.Stocks[i]
		stock.PrevPrice = stock.Price

		// Price moves based on performance and demand
		demandPressure := float64(stock.Demand) / float64(max(len(ss.Bots)/2, 1))
		perfPressure := (stock.Performance - 0.5) * 0.5

		priceChange := (demandPressure + perfPressure) * sm.Volatility
		priceChange += (ss.Rng.Float64() - 0.5) * sm.Volatility * 0.5 // noise

		stock.Price *= 1.0 + priceChange
		if stock.Price < 0.1 {
			stock.Price = 0.1
		}
		stock.Demand = 0
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// updatePortfolios recalculates net worth.
func updatePortfolios(sm *StockMarketState) {
	sm.MarketCap = 0
	for i := range sm.Portfolios {
		port := &sm.Portfolios[i]
		port.NetWorth = port.Cash
		for s := range sm.Stocks {
			port.NetWorth += float64(port.Holdings[s]) * sm.Stocks[s].Price
		}
		sm.MarketCap += port.NetWorth

		// Follow the stock they hold most of
		maxHolding := 0
		bestStock := -1
		for s := range port.Holdings {
			if port.Holdings[s] > maxHolding {
				maxHolding = port.Holdings[s]
				bestStock = s
			}
		}
		port.Strategy = bestStock
	}
}

// applyStockStrategies applies invested strategy to bot behavior.
func applyStockStrategies(ss *SwarmState, sm *StockMarketState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]
		strat := sm.Portfolios[i].Strategy
		if strat < 0 {
			continue
		}

		switch strat {
		case 0: // Aggressiv: fast, direct
			bot.Speed *= 1.2
		case 1: // Defensiv: careful, efficient
			bot.Speed *= 0.85
		case 2: // Explorer: random exploration
			bot.Angle += (ss.Rng.Float64() - 0.5) * 0.2
		case 3: // Kooperativ: cluster
			bot.Speed *= 0.95
		case 4: // Spezialist: focused on task
			if bot.CarryingPkg >= 0 {
				bot.Speed *= 1.1
			}
		}

		// LED color based on held stock (wealth gradient)
		wealthRatio := sm.Portfolios[i].NetWorth / 100.0
		if wealthRatio > 1 {
			wealthRatio = 1
		}
		colors := [][3]uint8{
			{255, 50, 50},   // Aggressiv: red
			{50, 200, 50},   // Defensiv: green
			{50, 150, 255},  // Explorer: blue
			{255, 200, 50},  // Kooperativ: yellow
			{200, 50, 200},  // Spezialist: purple
		}
		if strat >= 0 && strat < len(colors) {
			c := colors[strat]
			bot.LEDColor = [3]uint8{
				uint8(float64(c[0]) * wealthRatio),
				uint8(float64(c[1]) * wealthRatio),
				uint8(float64(c[2]) * wealthRatio),
			}
		}
	}
}

// StockMarketCap returns total market capitalization.
func StockMarketCap(sm *StockMarketState) float64 {
	if sm == nil {
		return 0
	}
	return sm.MarketCap
}

// StockTotalTrades returns total executed trades.
func StockTotalTrades(sm *StockMarketState) int {
	if sm == nil {
		return 0
	}
	return sm.TotalTrades
}

// StockBiggestBubble returns the highest ever price.
func StockBiggestBubble(sm *StockMarketState) float64 {
	if sm == nil {
		return 0
	}
	return sm.BiggestBubble
}

// StockPrice returns the current price of a stock.
func StockPrice(sm *StockMarketState, idx int) float64 {
	if sm == nil || idx < 0 || idx >= len(sm.Stocks) {
		return 0
	}
	return sm.Stocks[idx].Price
}
