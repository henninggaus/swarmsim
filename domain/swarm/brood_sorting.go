package swarm

import "swarmsim/domain/physics"

// Ant Brood Sorting (Deneubourg model): Bots sort colored items by
// picking up when surrounded by few same-color neighbors and dropping
// when surrounded by many. Creates emergent color clusters from chaos.
// Based on Deneubourg et al. (1991) differential stigmergy.

const (
	broodPickupProb  = 0.8  // base pickup probability
	broodDropProb    = 0.3  // base drop probability
	broodSenseRadius = 60.0 // radius to sense nearby item density
	broodItemRadius  = 8.0  // item interaction distance
)

// BroodItem represents a colored item to be sorted.
type BroodItem struct {
	X, Y  float64
	Color int     // 0=red, 1=green, 2=blue
	Held  bool    // carried by a bot?
	Holder int    // bot index carrying this item (-1 if not held)
}

// BroodState holds brood sorting state.
type BroodState struct {
	Items    []BroodItem
	Carrying []int // per-bot: item index being carried (-1 = none)
	ItemHash *physics.SpatialHash // spatial hash for unheld items
}

// InitBrood allocates brood sorting state with scattered items.
func InitBrood(ss *SwarmState) {
	n := len(ss.Bots)
	numItems := n * 3 // 3 items per bot

	st := &BroodState{
		Items:    make([]BroodItem, numItems),
		Carrying: make([]int, n),
	}

	for i := range st.Carrying {
		st.Carrying[i] = -1
	}

	// Scatter items randomly with 3 colors
	for i := range st.Items {
		st.Items[i] = BroodItem{
			X:      20 + ss.Rng.Float64()*(ss.ArenaW-40),
			Y:      20 + ss.Rng.Float64()*(ss.ArenaH-40),
			Color:  i % 3,
			Held:   false,
			Holder: -1,
		}
	}

	ss.Brood = st
	ss.BroodOn = true
}

// ClearBrood frees brood sorting state.
func ClearBrood(ss *SwarmState) {
	ss.Brood = nil
	ss.BroodOn = false
}

// rebuildItemHash rebuilds the spatial hash for unheld brood items.
func rebuildItemHash(st *BroodState, arenaW, arenaH float64) {
	if st.ItemHash == nil {
		st.ItemHash = physics.NewSpatialHash(arenaW, arenaH, broodSenseRadius)
	}
	st.ItemHash.Clear()
	for i := range st.Items {
		if !st.Items[i].Held {
			st.ItemHash.Insert(i, st.Items[i].X, st.Items[i].Y)
		}
	}
}

// TickBrood updates brood sorting: compute local density, move held items.
func TickBrood(ss *SwarmState) {
	if ss.Brood == nil {
		return
	}
	st := ss.Brood

	// Grow carrying slice
	for len(st.Carrying) < len(ss.Bots) {
		st.Carrying = append(st.Carrying, -1)
	}

	// Update held item positions (follow carrier)
	for i := range st.Items {
		if st.Items[i].Held && st.Items[i].Holder >= 0 && st.Items[i].Holder < len(ss.Bots) {
			carrier := &ss.Bots[st.Items[i].Holder]
			st.Items[i].X = carrier.X
			st.Items[i].Y = carrier.Y
		}
	}

	// Build spatial hash for unheld items
	rebuildItemHash(st, ss.ArenaW, ss.ArenaH)

	// Update sensor cache
	senseRadSq := broodSenseRadius * broodSenseRadius
	for i := range ss.Bots {
		bot := &ss.Bots[i]

		if st.Carrying[i] >= 0 {
			ss.Bots[i].BroodCarrying = 1
			ss.Bots[i].BroodItemColor = st.Items[st.Carrying[i]].Color + 1
		} else {
			ss.Bots[i].BroodCarrying = 0
			ss.Bots[i].BroodItemColor = 0
		}

		// Count nearby items and same-color density via spatial hash
		nearCount := 0
		sameCount := 0
		nearIDs := st.ItemHash.Query(bot.X, bot.Y, broodSenseRadius)
		for _, j := range nearIDs {
			if j < 0 || j >= len(st.Items) || st.Items[j].Held {
				continue
			}
			dx := st.Items[j].X - bot.X
			dy := st.Items[j].Y - bot.Y
			if dx*dx+dy*dy < senseRadSq {
				nearCount++
				if st.Carrying[i] >= 0 && st.Items[j].Color == st.Items[st.Carrying[i]].Color {
					sameCount++
				}
			}
		}

		ss.Bots[i].BroodDensity = nearCount
		ss.Bots[i].BroodSameColor = sameCount
	}
}

// countSameColorNear counts unheld items of the given color within broodSenseRadius
// of (x, y) using the item spatial hash.
func countSameColorNear(st *BroodState, x, y float64, color, excludeIdx int) int {
	if st.ItemHash == nil {
		return 0
	}
	senseRadSq := broodSenseRadius * broodSenseRadius
	sameNear := 0
	nearIDs := st.ItemHash.Query(x, y, broodSenseRadius)
	for _, j := range nearIDs {
		if j == excludeIdx || j < 0 || j >= len(st.Items) || st.Items[j].Held {
			continue
		}
		dx := st.Items[j].X - x
		dy := st.Items[j].Y - y
		if dx*dx+dy*dy < senseRadSq && st.Items[j].Color == color {
			sameNear++
		}
	}
	return sameNear
}

// ApplyBroodSort picks up or drops items based on local color density.
func ApplyBroodSort(bot *SwarmBot, ss *SwarmState, idx int) {
	if ss.Brood == nil || ss.Rng == nil || idx >= len(ss.Brood.Carrying) {
		bot.Speed = SwarmBotSpeed
		return
	}
	st := ss.Brood

	if st.Carrying[idx] >= 0 {
		// Carrying an item: consider dropping
		itemIdx := st.Carrying[idx]
		item := &st.Items[itemIdx]

		// Count same-color items nearby via spatial hash
		sameNear := countSameColorNear(st, bot.X, bot.Y, item.Color, itemIdx)

		// Higher same-color density → higher drop probability
		dropP := float64(sameNear) / 10.0
		if dropP > broodDropProb {
			dropP = broodDropProb
		}

		if ss.Rng.Float64() < dropP && sameNear >= 2 {
			// Drop item
			item.Held = false
			item.Holder = -1
			st.Carrying[idx] = -1
			bot.LEDColor = [3]uint8{100, 100, 100}
		} else {
			// Keep carrying — wander
			bot.Speed = SwarmBotSpeed
			setItemLED(bot, item.Color)
		}
	} else {
		// Not carrying: consider picking up nearest item via spatial hash
		bestDistSq := broodItemRadius * broodItemRadius
		bestItem := -1
		if st.ItemHash != nil {
			nearIDs := st.ItemHash.Query(bot.X, bot.Y, broodItemRadius)
			for _, j := range nearIDs {
				if j < 0 || j >= len(st.Items) || st.Items[j].Held {
					continue
				}
				dx := st.Items[j].X - bot.X
				dy := st.Items[j].Y - bot.Y
				dSq := dx*dx + dy*dy
				if dSq < bestDistSq {
					bestDistSq = dSq
					bestItem = j
				}
			}
		}

		if bestItem >= 0 {
			item := &st.Items[bestItem]
			// Count same-color items nearby via spatial hash
			sameNear := countSameColorNear(st, bot.X, bot.Y, item.Color, bestItem)

			// Lower same-color density → higher pickup probability
			pickP := broodPickupProb / (1 + float64(sameNear))
			if ss.Rng.Float64() < pickP {
				item.Held = true
				item.Holder = idx
				st.Carrying[idx] = bestItem
				setItemLED(bot, item.Color)
			}
		}

		bot.Speed = SwarmBotSpeed
	}
}

func setItemLED(bot *SwarmBot, color int) {
	switch color {
	case 0:
		bot.LEDColor = [3]uint8{255, 80, 80}   // red
	case 1:
		bot.LEDColor = [3]uint8{80, 255, 80}   // green
	case 2:
		bot.LEDColor = [3]uint8{80, 80, 255}   // blue
	}
}
