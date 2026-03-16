package render

import "fmt"

// hudStringCache caches formatted strings for HUD rendering.
// Strings are only re-formatted every N frames to reduce GC pressure.
type hudStringCache struct {
	updateInterval int // re-format every N frames
	frameCounter   int

	// Cached strings (swarm HUD)
	evoInfo   string
	gpInfo    string
	chainInfo string
	optInfo   string
	chainDone string
	followCam string

	// Cached strings (classic HUD)
	classicInfo  string
	classicScene string
	classicGen   string
	classicRes   string
	fpsWarn      string

	// Cached dashboard strings
	dashDiversity string
	dashUnique    string
	dashMaxFit    string
	dashGenLabel  string
	dashSpeedMax  string
	dashSpeedMin  string

	// Last seen values for dirty-check (only update when changed)
	lastGen        int
	lastGPGen      int
	lastTick       int
	lastScore      int
	lastBestFit    float64
	lastAvgFit     float64
	lastFollowBot  int
	lastOptTrial   int
	lastChainStep  int
	lastChainTimer int
}

var hudCache = hudStringCache{
	updateInterval: 10, // update every 10 frames (~6 Hz at 60 FPS)
}

// hudCacheTick advances the frame counter and returns true if strings should be updated.
func hudCacheTick() bool {
	hudCache.frameCounter++
	if hudCache.frameCounter >= hudCache.updateInterval {
		hudCache.frameCounter = 0
		return true
	}
	return false
}

// cachedSprintf returns a cached string, only re-formatting when shouldUpdate is true.
// Usage: call hudCacheTick() once per frame, then use the result for all cache lookups.
func cachedEvoInfo(shouldUpdate bool, gen int, best, avg float64, timer, interval int) string {
	if shouldUpdate || hudCache.evoInfo == "" {
		hudCache.evoInfo = fmt.Sprintf("Gen: %d | Best: %.0f | Avg: %.1f | Timer: %d/%d",
			gen, best, avg, timer, interval)
	}
	return hudCache.evoInfo
}

func cachedGPInfo(shouldUpdate bool, gen int, best, avg float64, timer, interval int) string {
	if shouldUpdate || hudCache.gpInfo == "" {
		hudCache.gpInfo = fmt.Sprintf("GP Gen:%d | Best:%.0f | Avg:%.0f | %d/%d",
			gen, best, avg, timer, interval)
	}
	return hudCache.gpInfo
}

func cachedChainInfo(shouldUpdate bool, name string, elapsed, total, score int) string {
	if shouldUpdate || hudCache.chainInfo == "" {
		hudCache.chainInfo = fmt.Sprintf("SZENARIO-KETTE: %s | %d/%d Ticks | Score:%d | F5=Stop",
			name, elapsed, total, score)
	}
	return hudCache.chainInfo
}

func cachedChainDone(shouldUpdate bool, totalScore int, stepScores []int) string {
	if shouldUpdate || hudCache.chainDone == "" {
		result := fmt.Sprintf("KETTE FERTIG! Gesamt-Score: %d", totalScore)
		for i, s := range stepScores {
			result += fmt.Sprintf(" | S%d:%d", i+1, s)
		}
		hudCache.chainDone = result
	}
	return hudCache.chainDone
}

func cachedOptInfo(shouldUpdate bool, trial, maxTrials int, current, best float64) string {
	if shouldUpdate || hudCache.optInfo == "" {
		hudCache.optInfo = fmt.Sprintf("AUTO-OPTIMIZER: Trial %d/%d | Score:%.0f | Best:%.0f | F4=Stop",
			trial+1, maxTrials, current, best)
	}
	return hudCache.optInfo
}

func cachedFollowCam(shouldUpdate bool, botIdx int) string {
	if shouldUpdate || hudCache.followCam == "" || hudCache.lastFollowBot != botIdx {
		hudCache.followCam = fmt.Sprintf("Folge Bot #%d [F zum Stoppen]", botIdx)
		hudCache.lastFollowBot = botIdx
	}
	return hudCache.followCam
}

func cachedClassicInfo(shouldUpdate bool, fps float64, tick int, speed float64, paused bool) string {
	if shouldUpdate || hudCache.classicInfo == "" {
		info := fmt.Sprintf("FPS: %.0f  Tick: %d  Speed: %.1fx", fps, tick, speed)
		if paused {
			info += "  [PAUSED]"
		}
		hudCache.classicInfo = info
	}
	return hudCache.classicInfo
}

func cachedClassicGen(shouldUpdate bool, gen, tick, genLen int, best, avg float64) string {
	if shouldUpdate || hudCache.classicGen == "" {
		hudCache.classicGen = fmt.Sprintf("Gen: %d  Tick: %d/%d  Best: %.0f  Avg: %.0f",
			gen, tick, genLen, best, avg)
	}
	return hudCache.classicGen
}

func cachedClassicRes(shouldUpdate bool, avail, delivered, score, msgs, totalMsgs int) string {
	if shouldUpdate || hudCache.classicRes == "" {
		hudCache.classicRes = fmt.Sprintf("Ressourcen: %d  Geliefert: %d  Score: %d  Msgs: %d (gesamt: %d)",
			avail, delivered, score, msgs, totalMsgs)
	}
	return hudCache.classicRes
}
