package swarm

import (
	"math"
	"swarmsim/logger"
)

// EnergyEconomyState manages a resource economy among bots.
// Bots earn energy by completing tasks, spend it on actions,
// and can trade with nearby bots. Prices emerge from supply/demand.
type EnergyEconomyState struct {
	// Global market
	MarketPrice  float64 // current energy trading price (emerges from supply/demand)
	TotalWealth  float64 // total energy in the system
	GiniCoeff    float64 // wealth inequality (0=equal, 1=unequal)
	TradeCount   int     // total trades completed
	Inflation    float64 // price change rate

	// Per-bot economy
	Wallets []BotWallet

	// Parameters
	EarnRate     float64 // energy earned per delivery (default 10)
	MoveCost     float64 // energy cost per tick of movement (default 0.01)
	SensorCost   float64 // energy cost per sensor reading (default 0.005)
	CommCost     float64 // energy cost per communication (default 0.002)
	TradeRange   float64 // max distance for trading (default 50)
	StartEnergy  float64 // initial energy per bot (default 50)
	MaxEnergy    float64 // energy cap per bot (default 200)
	TaxRate      float64 // redistribution tax on trades (default 0.05)
	MinTradeAmt  float64 // minimum trade amount (default 1)
}

// BotWallet holds a bot's economic state.
type BotWallet struct {
	Energy     float64 // current energy
	Earned     float64 // total earned
	Spent      float64 // total spent
	Traded     float64 // total traded (given + received)
	TradeCount int     // number of trades
	Bankrupt   int     // ticks spent at zero energy
	Role       EconRole // current economic role
}

// EconRole describes a bot's economic behavior.
type EconRole int

const (
	EconWorker   EconRole = iota // earns by delivering
	EconTrader                    // buys low, sells high
	EconHoarder                   // accumulates energy
	EconAltruist                  // gives to low-energy neighbors
)

// EconRoleName returns the display name.
func EconRoleName(r EconRole) string {
	switch r {
	case EconWorker:
		return "Arbeiter"
	case EconTrader:
		return "Haendler"
	case EconHoarder:
		return "Sammler"
	case EconAltruist:
		return "Altruist"
	default:
		return "?"
	}
}

// InitEnergyEconomy sets up the energy economy system.
func InitEnergyEconomy(ss *SwarmState) {
	n := len(ss.Bots)
	ee := &EnergyEconomyState{
		MarketPrice: 1.0,
		Wallets:     make([]BotWallet, n),
		EarnRate:    10,
		MoveCost:    0.01,
		SensorCost:  0.005,
		CommCost:    0.002,
		TradeRange:  50,
		StartEnergy: 50,
		MaxEnergy:   200,
		TaxRate:     0.05,
		MinTradeAmt: 1,
	}

	for i := range ee.Wallets {
		ee.Wallets[i].Energy = ee.StartEnergy
		ee.Wallets[i].Role = EconRole(ss.Rng.Intn(4))
	}

	ee.TotalWealth = ee.StartEnergy * float64(n)
	ss.EnergyEconomy = ee
	logger.Info("ECONOMY", "Initialisiert: %d Bots, Start=%.0f, MaxEnergie=%.0f",
		n, ee.StartEnergy, ee.MaxEnergy)
}

// ClearEnergyEconomy disables the energy economy system.
func ClearEnergyEconomy(ss *SwarmState) {
	ss.EnergyEconomy = nil
	ss.EnergyEconomyOn = false
}

// TickEnergyEconomy runs one tick of the economy.
func TickEnergyEconomy(ss *SwarmState) {
	ee := ss.EnergyEconomy
	if ee == nil {
		return
	}

	n := len(ss.Bots)
	if len(ee.Wallets) != n {
		return
	}

	// Phase 1: Costs — moving bots spend energy
	for i := range ss.Bots {
		w := &ee.Wallets[i]
		if ss.Bots[i].Speed > 0 {
			cost := ee.MoveCost * (ss.Bots[i].Speed / SwarmBotSpeed)
			w.Energy -= cost
			w.Spent += cost
		}
		// Sensor cost (always active)
		w.Energy -= ee.SensorCost
		w.Spent += ee.SensorCost

		// Bankrupt bots slow down
		if w.Energy <= 0 {
			w.Energy = 0
			w.Bankrupt++
			ss.Bots[i].Speed *= 0.5
		}
	}

	// Phase 2: Earnings — reward deliveries
	for i := range ss.Bots {
		if ss.Bots[i].Stats.TotalDeliveries > 0 && ss.Tick > 0 {
			// Earn proportionally to recent activity
			recentDeliveries := ss.Bots[i].Stats.TotalDeliveries
			if recentDeliveries > 0 && ss.Tick%100 == 0 {
				earned := ee.EarnRate * float64(recentDeliveries) / float64(ss.Tick) * 100
				ee.Wallets[i].Energy += earned
				ee.Wallets[i].Earned += earned
				if ee.Wallets[i].Energy > ee.MaxEnergy {
					ee.Wallets[i].Energy = ee.MaxEnergy
				}
			}
		}
	}

	// Phase 3: Trading — nearby bots exchange energy
	if ss.Tick%3 == 0 {
		tickTrading(ss, ee)
	}

	// Phase 4: Role adaptation
	if ss.Tick%50 == 0 {
		adaptRoles(ss, ee)
	}

	// Phase 5: Update stats
	updateEconomyStats(ee)

	// Phase 6: Visual — LED color by wealth
	maxE := ee.MaxEnergy
	for i := range ss.Bots {
		ratio := ee.Wallets[i].Energy / maxE
		if ratio > 1 {
			ratio = 1
		}
		// Green = wealthy, Red = poor
		ss.Bots[i].LEDColor = [3]uint8{
			uint8((1 - ratio) * 255),
			uint8(ratio * 255),
			50,
		}
	}
}

// tickTrading handles peer-to-peer energy trading.
func tickTrading(ss *SwarmState, ee *EnergyEconomyState) {
	rangeSq := ee.TradeRange * ee.TradeRange
	n := len(ss.Bots)

	for i := 0; i < n; i++ {
		wi := &ee.Wallets[i]

		// Only altruists and traders initiate trades
		if wi.Role != EconAltruist && wi.Role != EconTrader {
			continue
		}
		if wi.Energy < ee.MinTradeAmt*2 {
			continue
		}

		// Find nearest eligible partner
		bestJ := -1
		bestDist := rangeSq

		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			dx := ss.Bots[i].X - ss.Bots[j].X
			dy := ss.Bots[i].Y - ss.Bots[j].Y
			dSq := dx*dx + dy*dy
			if dSq >= bestDist {
				continue
			}

			wj := &ee.Wallets[j]
			// Altruists give to poor, traders give to those with less
			switch wi.Role {
			case EconAltruist:
				if wj.Energy < wi.Energy*0.5 {
					bestJ = j
					bestDist = dSq
				}
			case EconTrader:
				if wj.Energy < wi.Energy*0.8 {
					bestJ = j
					bestDist = dSq
				}
			}
		}

		if bestJ < 0 {
			continue
		}

		// Execute trade
		wj := &ee.Wallets[bestJ]
		amount := math.Min(wi.Energy*0.1, ee.MaxEnergy-wj.Energy)
		if amount < ee.MinTradeAmt {
			continue
		}

		tax := amount * ee.TaxRate
		wi.Energy -= amount
		wj.Energy += amount - tax
		wi.Traded += amount
		wj.Traded += amount
		wi.TradeCount++
		wj.TradeCount++
		ee.TradeCount++
	}
}

// adaptRoles lets bots switch economic roles based on their situation.
func adaptRoles(ss *SwarmState, ee *EnergyEconomyState) {
	for i := range ee.Wallets {
		w := &ee.Wallets[i]
		// Poor bots become workers
		if w.Energy < ee.StartEnergy*0.3 {
			w.Role = EconWorker
			continue
		}
		// Rich bots become altruists or hoarders
		if w.Energy > ee.MaxEnergy*0.7 {
			if ss.Rng.Float64() < 0.5 {
				w.Role = EconAltruist
			} else {
				w.Role = EconHoarder
			}
			continue
		}
		// Random role switch with low probability
		if ss.Rng.Float64() < 0.05 {
			w.Role = EconRole(ss.Rng.Intn(4))
		}
	}
}

// updateEconomyStats computes aggregate economic metrics.
func updateEconomyStats(ee *EnergyEconomyState) {
	n := len(ee.Wallets)
	if n == 0 {
		return
	}

	// Total wealth
	total := 0.0
	for _, w := range ee.Wallets {
		total += w.Energy
	}
	prevWealth := ee.TotalWealth
	ee.TotalWealth = total

	// Inflation
	if prevWealth > 0 {
		ee.Inflation = (total - prevWealth) / prevWealth
	}

	// Gini coefficient
	mean := total / float64(n)
	if mean <= 0 {
		ee.GiniCoeff = 0
		return
	}

	sumDiff := 0.0
	for i := range ee.Wallets {
		for j := range ee.Wallets {
			sumDiff += math.Abs(ee.Wallets[i].Energy - ee.Wallets[j].Energy)
		}
	}
	ee.GiniCoeff = sumDiff / (2 * float64(n) * float64(n) * mean)
}

// EconomyGini returns the current Gini coefficient.
func EconomyGini(ee *EnergyEconomyState) float64 {
	if ee == nil {
		return 0
	}
	return ee.GiniCoeff
}

// EconomyTradeCount returns total trades.
func EconomyTradeCount(ee *EnergyEconomyState) int {
	if ee == nil {
		return 0
	}
	return ee.TradeCount
}

// BotEnergy returns a bot's current energy.
func BotEnergy(ee *EnergyEconomyState, idx int) float64 {
	if ee == nil || idx < 0 || idx >= len(ee.Wallets) {
		return 0
	}
	return ee.Wallets[idx].Energy
}
