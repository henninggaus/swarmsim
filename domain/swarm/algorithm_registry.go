package swarm

import "math"

// algorithm_registry.go provides a dispatch table for swarm algorithm lifecycle
// and query functions. Each algorithm registers an algoHandler with lifecycle
// functions (Init, Clear, Tick, Apply) and optional query functions for fitness,
// position, and exploration ratio. This eliminates large switch statements and
// makes adding a new algorithm a single-location change.
//
// Adding a new algorithm now requires only:
//  1. Defining AlgoXxx in the SwarmAlgorithmType enum.
//  2. Creating the Init/Clear/Tick/Apply functions in a dedicated file.
//  3. Adding one entry to algoRegistry below (including query functions).
//
// No switch statement edits are needed.

// algoHandler bundles the lifecycle and query functions for a single swarm
// algorithm. Any nil field means the algorithm does not provide that capability.
type algoHandler struct {
	// Lifecycle
	init  func(*SwarmState)                 // allocate per-bot state
	clear func(*SwarmState)                 // free state
	tick  func(*SwarmState)                 // global per-tick update
	apply func(*SwarmBot, *SwarmState, int) // per-bot steering (nil if tick handles it)

	// Queries — nil means the algorithm does not support the query.
	bestFitness      func(*SwarmState) float64                   // global best fitness
	avgFitnessVals   func(*SwarmState) []float64                 // per-bot fitness slice for averaging
	bestPos          func(*SwarmState) (float64, float64, bool)  // (x, y, ok) of global best
	explorationRatio func(*SwarmState) float64                   // 0-100 (100=exploration), -1 if N/A
}

// sliceMax returns the maximum value in a float64 slice, or 0 if empty.
func sliceMax(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	best := vals[0]
	for _, v := range vals[1:] {
		if v > best {
			best = v
		}
	}
	return best
}

// sliceMaxIdx returns the index of the maximum value in a float64 slice, or -1.
func sliceMaxIdx(vals []float64) int {
	if len(vals) == 0 {
		return -1
	}
	best := 0
	for i := 1; i < len(vals); i++ {
		if vals[i] > vals[best] {
			best = i
		}
	}
	return best
}

// botPos returns the (x, y, true) of a bot at index idx, or (0, 0, false) if out of range.
func botPos(ss *SwarmState, idx int) (float64, float64, bool) {
	if idx >= 0 && idx < len(ss.Bots) {
		return ss.Bots[idx].X, ss.Bots[idx].Y, true
	}
	return 0, 0, false
}

// cycleExplRatio returns (1 - tick/maxTicks) * 100, clamped to [0, 100].
func cycleExplRatio(tick, maxTicks int) float64 {
	t := float64(tick) / float64(maxTicks)
	if t > 1 {
		t = 1
	}
	return (1.0 - t) * 100
}

// algoRegistry maps each algorithm type to its handler functions.
// Boids, ACO, and Firefly do not optimise a fitness function and have no
// fitness/position queries.
var algoRegistry = map[SwarmAlgorithmType]algoHandler{
	AlgoBoids: {
		tick: func(ss *SwarmState) { tickBoids(ss, ss.SwarmAlgo) },
	},
	AlgoPSO: {
		init:  InitPSO,
		clear: ClearPSO,
		tick:  TickPSO,
		apply: ApplyPSOMove,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.PSO != nil { return ss.PSO.GlobalFit }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.PSO != nil { return ss.PSO.BestFit }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.PSO != nil { return ss.PSO.GlobalX, ss.PSO.GlobalY, true }
			return 0, 0, false
		},
	},
	AlgoACO: {
		init:  InitACO,
		clear: ClearACO,
		tick:  TickACO,
		apply: ApplyACO,
	},
	AlgoFirefly: {
		init:  InitFireflyAlgo,
		clear: ClearFireflyAlgo,
		tick:  func(ss *SwarmState) { tickFirefly(ss, ss.SwarmAlgo) },
		bestFitness: func(ss *SwarmState) float64 {
			if ss.SwarmAlgo != nil {
				return ss.SwarmAlgo.FireflyBestF
			}
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.SwarmAlgo != nil {
				return ss.SwarmAlgo.FireflyBrightness
			}
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.SwarmAlgo != nil && ss.SwarmAlgo.FireflyBestIdx >= 0 {
				return ss.SwarmAlgo.FireflyBestX, ss.SwarmAlgo.FireflyBestY, true
			}
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.SwarmAlgo == nil {
				return -1
			}
			return cycleExplRatio(ss.SwarmAlgo.FireflyCycleTick, fireflyMaxTicks)
		},
	},
	AlgoGWO: {
		init:  InitGWO,
		clear: ClearGWO,
		tick:  TickGWO,
		apply: ApplyGWO,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.GWO != nil && ss.GWO.AlphaIdx >= 0 && ss.GWO.AlphaIdx < len(ss.GWO.Fitness) {
				return ss.GWO.Fitness[ss.GWO.AlphaIdx]
			}
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.GWO != nil { return ss.GWO.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.GWO != nil && ss.GWO.AlphaIdx >= 0 {
				return ss.GWO.AlphaX, ss.GWO.AlphaY, true
			}
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.GWO == nil { return -1 }
			a := 2.0 * (1.0 - float64(ss.GWO.HuntTick)/float64(gwoMaxTicks))
			if a < 0 { a = 0 }
			return (a / 2.0) * 100
		},
	},
	AlgoWOA: {
		init:  InitWOA,
		clear: ClearWOA,
		tick:  TickWOA,
		apply: ApplyWOA,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.WOA != nil && ss.WOA.BestIdx >= 0 && ss.WOA.BestIdx < len(ss.WOA.Fitness) {
				return ss.WOA.Fitness[ss.WOA.BestIdx]
			}
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.WOA != nil { return ss.WOA.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.WOA != nil && ss.WOA.BestIdx >= 0 {
				return botPos(ss, ss.WOA.BestIdx)
			}
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.WOA == nil { return -1 }
			a := 2.0 * (1.0 - float64(ss.WOA.HuntTick)/float64(woaMaxTicks))
			if a < 0 { a = 0 }
			return (a / 2.0) * 100
		},
	},
	AlgoBFO: {
		init:  InitBFO,
		clear: ClearBFO,
		tick:  TickBFO,
		apply: ApplyBFO,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.BFO != nil { return sliceMax(ss.BFO.Health) }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.BFO != nil { return ss.BFO.Health }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.BFO != nil && len(ss.BFO.Health) > 0 {
				return botPos(ss, sliceMaxIdx(ss.BFO.Health))
			}
			return 0, 0, false
		},
	},
	AlgoMFO: {
		init:  InitMFO,
		clear: ClearMFO,
		tick:  TickMFO,
		apply: ApplyMFO,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.MFO != nil { return sliceMax(ss.MFO.BotFitness) }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.MFO != nil { return ss.MFO.BotFitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.MFO != nil && len(ss.MFO.BotFitness) > 0 {
				return botPos(ss, sliceMaxIdx(ss.MFO.BotFitness))
			}
			return 0, 0, false
		},
	},
	AlgoCuckoo: {
		init:  InitCuckoo,
		clear: ClearCuckoo,
		tick:  TickCuckoo,
		apply: ApplyCuckoo,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.Cuckoo != nil { return ss.Cuckoo.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.Cuckoo != nil { return ss.Cuckoo.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.Cuckoo != nil { return ss.Cuckoo.BestX, ss.Cuckoo.BestY, true }
			return 0, 0, false
		},
	},
	AlgoDE: {
		init:  InitDE,
		clear: ClearDE,
		tick:  TickDE,
		apply: ApplyDE,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.DE != nil { return ss.DE.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.DE != nil { return ss.DE.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.DE != nil && ss.DE.BestIdx >= 0 {
				return botPos(ss, ss.DE.BestIdx)
			}
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.DE == nil { return -1 }
			return cycleExplRatio(ss.DE.GenTick, deMaxTicks)
		},
	},
	AlgoABC: {
		init:  InitABC,
		clear: ClearABC,
		tick:  TickABC,
		apply: ApplyABC,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.ABC != nil { return ss.ABC.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.ABC != nil { return ss.ABC.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.ABC != nil { return ss.ABC.BestX, ss.ABC.BestY, true }
			return 0, 0, false
		},
	},
	AlgoHSO: {
		init:  InitHSO,
		clear: ClearHSO,
		tick:  TickHSO,
		apply: ApplyHSO,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.HSO != nil { return ss.HSO.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.HSO != nil { return ss.HSO.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.HSO != nil { return ss.HSO.BestX, ss.HSO.BestY, true }
			return 0, 0, false
		},
	},
	AlgoBat: {
		init:  InitBat,
		clear: ClearBat,
		tick:  TickBat,
		apply: ApplyBat,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.Bat != nil { return ss.Bat.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.Bat != nil { return ss.Bat.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.Bat != nil { return ss.Bat.BestX, ss.Bat.BestY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.Bat == nil { return -1 }
			avgLoud := ss.Bat.AvgLoud
			if avgLoud > 1 { avgLoud = 1 }
			return avgLoud * 100
		},
	},
	AlgoSSA: {
		init:  InitSSA,
		clear: ClearSSA,
		tick:  TickSSA,
		apply: ApplySSA,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.SSA != nil { return ss.SSA.FoodFit }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.SSA != nil { return ss.SSA.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.SSA != nil { return ss.SSA.FoodX, ss.SSA.FoodY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.SSA == nil { return -1 }
			t := float64(ss.SSA.CycleTick)
			T := float64(ssaMaxTicks)
			c1 := 2.0 * math.Exp(-math.Pow(4*t/T, 2))
			return (c1 / 2.0) * 100
		},
	},
	AlgoGSA: {
		init:  InitGSA,
		clear: ClearGSA,
		tick:  TickGSA,
		apply: ApplyGSA,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.GSA != nil { return sliceMax(ss.GSA.Fitness) }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.GSA != nil { return ss.GSA.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.GSA != nil { return ss.GSA.BestX, ss.GSA.BestY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.GSA == nil { return -1 }
			ratio := ss.GSA.G / gsaG0
			if ratio > 1 { ratio = 1 }
			return ratio * 100
		},
	},
	AlgoFPA: {
		init:  InitFPA,
		clear: ClearFPA,
		tick:  TickFPA,
		apply: ApplyFPA,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.FPA != nil { return ss.FPA.GlobalBestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.FPA != nil { return ss.FPA.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.FPA != nil { return ss.FPA.GlobalBestX, ss.FPA.GlobalBestY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.FPA == nil { return -1 }
			return cycleExplRatio(ss.FPA.PollTick, fpaMaxTicks)
		},
	},
	AlgoHHO: {
		init:  InitHHO,
		clear: ClearHHO,
		tick:  TickHHO,
		apply: ApplyHHO,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.HHO != nil { return ss.HHO.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.HHO != nil { return ss.HHO.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.HHO != nil { return ss.HHO.BestX, ss.HHO.BestY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.HHO == nil { return -1 }
			tRatio := float64(ss.HHO.HuntTick) / float64(hhoMaxTicks)
			e := 2.0 * (1.0 - tRatio)
			if e < 0 { e = 0 }
			return (e / 2.0) * 100
		},
	},
	AlgoSA: {
		init:  InitSA,
		clear: ClearSA,
		tick:  TickSA,
		apply: ApplySA,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.SA != nil { return ss.SA.GlobalBestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.SA != nil { return ss.SA.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.SA != nil && ss.SA.GlobalBestIdx >= 0 {
				return botPos(ss, ss.SA.GlobalBestIdx)
			}
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.SA == nil || len(ss.SA.Temp) == 0 { return -1 }
			avgTemp := 0.0
			for _, t := range ss.SA.Temp {
				avgTemp += t
			}
			avgTemp /= float64(len(ss.SA.Temp))
			ratio := avgTemp / ss.SA.InitialTemp
			if ratio > 1 { ratio = 1 }
			return ratio * 100
		},
	},
	AlgoAO: {
		init:  InitAO,
		clear: ClearAO,
		tick:  TickAO,
		apply: ApplyAO,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.AO != nil { return ss.AO.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.AO != nil { return ss.AO.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.AO != nil { return ss.AO.BestX, ss.AO.BestY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.AO == nil { return -1 }
			return cycleExplRatio(ss.AO.HuntTick, aoMaxTicks)
		},
	},
	AlgoSCA: {
		init:  InitSCA,
		clear: ClearSCA,
		tick:  TickSCA,
		apply: ApplySCA,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.SCA != nil { return ss.SCA.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.SCA != nil { return ss.SCA.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.SCA != nil { return ss.SCA.BestX, ss.SCA.BestY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.SCA == nil { return -1 }
			r1 := scaAMax * (1.0 - float64(ss.SCA.Tick)/float64(scaMaxTicks))
			if r1 < 0 { r1 = 0 }
			return (r1 / scaAMax) * 100
		},
	},
	AlgoDA: {
		init:  InitDA,
		clear: ClearDA,
		tick:  TickDA,
		apply: ApplyDA,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.DA != nil { return ss.DA.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.DA != nil { return ss.DA.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.DA != nil { return ss.DA.BestX, ss.DA.BestY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.DA == nil { return -1 }
			return cycleExplRatio(ss.DA.Tick, daMaxTicks)
		},
	},
	AlgoTLBO: {
		init:  InitTLBO,
		clear: ClearTLBO,
		tick:  TickTLBO,
		apply: ApplyTLBO,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.TLBO != nil { return ss.TLBO.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.TLBO != nil { return ss.TLBO.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.TLBO != nil { return ss.TLBO.BestX, ss.TLBO.BestY, true }
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.TLBO == nil { return -1 }
			return cycleExplRatio(ss.TLBO.Tick, tlboMaxTicks)
		},
	},
	AlgoEO: {
		init:  InitEO,
		clear: ClearEO,
		tick:  TickEO,
		apply: ApplyEO,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.EO != nil { return ss.EO.BestFit }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.EO != nil { return ss.EO.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.EO != nil && ss.EO.BestIdx >= 0 {
				return botPos(ss, ss.EO.BestIdx)
			}
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.EO == nil { return -1 }
			return cycleExplRatio(ss.EO.CycleTick, eoMaxTicks)
		},
	},
	AlgoJaya: {
		init:  InitJaya,
		clear: ClearJaya,
		tick:  TickJaya,
		apply: ApplyJaya,
		bestFitness: func(ss *SwarmState) float64 {
			if ss.Jaya != nil { return ss.Jaya.BestF }
			return 0
		},
		avgFitnessVals: func(ss *SwarmState) []float64 {
			if ss.Jaya != nil { return ss.Jaya.Fitness }
			return nil
		},
		bestPos: func(ss *SwarmState) (float64, float64, bool) {
			if ss.Jaya != nil && ss.Jaya.BestIdx >= 0 {
				return ss.Jaya.BestX, ss.Jaya.BestY, true
			}
			return 0, 0, false
		},
		explorationRatio: func(ss *SwarmState) float64 {
			if ss.Jaya == nil { return -1 }
			return cycleExplRatio(ss.Jaya.Tick, jayaMaxTicks)
		},
	},
}
