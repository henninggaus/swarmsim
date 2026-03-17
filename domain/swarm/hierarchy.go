package swarm

import (
	"math"
	"swarmsim/logger"
)

// HierarchyState manages a multi-scale hierarchical organization.
// Bots self-organize into squads with elected leaders. Leaders form
// platoons, and platoons form companies — creating emergent command chains.
type HierarchyState struct {
	SquadSize    int     // bots per squad (default 5)
	MaxLevels    int     // maximum hierarchy depth (default 3)
	LeaderRange  float64 // leader influence range (default 100)
	ElectionFreq int     // ticks between leader elections (default 200)

	Squads   []Squad
	Levels   []HierarchyLevel // level 0 = squads, level 1 = platoons, etc.
	BotSquad []int            // per-bot: which squad (-1 = unassigned)

	// Stats
	TotalSquads  int
	AvgSquadSize float64
	LeaderCount  int
	LastElection int
}

// Squad is a group of bots with a leader.
type Squad struct {
	ID       int
	LeaderID int     // bot index of squad leader
	Members  []int   // bot indices
	CenterX  float64
	CenterY  float64
	Cohesion float64 // 0-1, how close members are
	Task     int     // current squad task
}

// HierarchyLevel represents one level of the hierarchy.
type HierarchyLevel struct {
	Name   string
	Groups [][]int // each group is a list of squad/group IDs from the level below
}

// Squad tasks
const (
	TaskExplore  = 0
	TaskGather   = 1
	TaskDeliver  = 2
	TaskDefend   = 3
)

// TaskName returns the display name.
func TaskName(t int) string {
	switch t {
	case TaskExplore:
		return "Erkunden"
	case TaskGather:
		return "Sammeln"
	case TaskDeliver:
		return "Liefern"
	case TaskDefend:
		return "Verteidigen"
	default:
		return "?"
	}
}

// InitHierarchy sets up the hierarchical swarm system.
func InitHierarchy(ss *SwarmState, squadSize int) {
	if squadSize < 2 {
		squadSize = 2
	}
	if squadSize > 15 {
		squadSize = 15
	}

	n := len(ss.Bots)
	hs := &HierarchyState{
		SquadSize:    squadSize,
		MaxLevels:    3,
		LeaderRange:  100,
		ElectionFreq: 200,
		BotSquad:     make([]int, n),
	}

	// Initial squad assignment: geographic clustering
	assignSquads(ss, hs)

	// Build hierarchy levels
	buildHierarchy(hs)

	ss.Hierarchy = hs
	logger.Info("HIERARCHY", "Initialisiert: %d Squads, %d Bots, SquadGroesse=%d",
		hs.TotalSquads, n, squadSize)
}

// ClearHierarchy disables the hierarchy system.
func ClearHierarchy(ss *SwarmState) {
	ss.Hierarchy = nil
	ss.HierarchyOn = false
}

// assignSquads creates squads using simple sequential assignment.
func assignSquads(ss *SwarmState, hs *HierarchyState) {
	n := len(ss.Bots)
	numSquads := n / hs.SquadSize
	if numSquads < 1 {
		numSquads = 1
	}

	hs.Squads = make([]Squad, numSquads)
	for i := range hs.BotSquad {
		hs.BotSquad[i] = -1
	}

	for i := range hs.Squads {
		hs.Squads[i] = Squad{
			ID:      i,
			Members: make([]int, 0, hs.SquadSize),
		}
	}

	// Assign bots to squads
	for i := 0; i < n; i++ {
		squadIdx := i / hs.SquadSize
		if squadIdx >= numSquads {
			squadIdx = numSquads - 1
		}
		hs.Squads[squadIdx].Members = append(hs.Squads[squadIdx].Members, i)
		hs.BotSquad[i] = squadIdx
	}

	// Elect initial leaders (first member)
	for i := range hs.Squads {
		if len(hs.Squads[i].Members) > 0 {
			hs.Squads[i].LeaderID = hs.Squads[i].Members[0]
		}
	}

	hs.TotalSquads = numSquads
}

// buildHierarchy creates higher-level groupings.
func buildHierarchy(hs *HierarchyState) {
	hs.Levels = nil

	// Level 0: squads
	squadIDs := make([]int, len(hs.Squads))
	for i := range squadIDs {
		squadIDs[i] = i
	}

	// Group squads into platoons (groups of 3-4 squads)
	platoonSize := 3
	if len(squadIDs) > 0 {
		groups := groupIDs(squadIDs, platoonSize)
		hs.Levels = append(hs.Levels, HierarchyLevel{
			Name:   "Zuege",
			Groups: groups,
		})

		// Group platoons into companies
		if len(groups) > 1 {
			platoonIDs := make([]int, len(groups))
			for i := range platoonIDs {
				platoonIDs[i] = i
			}
			companyGroups := groupIDs(platoonIDs, platoonSize)
			hs.Levels = append(hs.Levels, HierarchyLevel{
				Name:   "Kompanien",
				Groups: companyGroups,
			})
		}
	}
}

// groupIDs splits a list of IDs into groups of given size.
func groupIDs(ids []int, size int) [][]int {
	var groups [][]int
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		group := make([]int, end-i)
		copy(group, ids[i:end])
		groups = append(groups, group)
	}
	return groups
}

// TickHierarchy runs one tick of the hierarchical system.
func TickHierarchy(ss *SwarmState) {
	hs := ss.Hierarchy
	if hs == nil {
		return
	}

	// Update squad centers and cohesion
	for i := range hs.Squads {
		updateSquadStats(ss, &hs.Squads[i])
	}

	// Leader elections
	if ss.Tick-hs.LastElection >= hs.ElectionFreq {
		electLeaders(ss, hs)
		hs.LastElection = ss.Tick
	}

	// Squad behavior: members follow leaders
	for i := range hs.Squads {
		applySquadBehavior(ss, hs, &hs.Squads[i])
	}

	// Assign squad tasks based on situation
	if ss.Tick%100 == 0 {
		assignSquadTasks(ss, hs)
	}

	// Update stats
	hs.LeaderCount = 0
	totalSize := 0
	for _, sq := range hs.Squads {
		totalSize += len(sq.Members)
		hs.LeaderCount++
	}
	if len(hs.Squads) > 0 {
		hs.AvgSquadSize = float64(totalSize) / float64(len(hs.Squads))
	}
}

// updateSquadStats computes squad center and cohesion.
func updateSquadStats(ss *SwarmState, sq *Squad) {
	if len(sq.Members) == 0 {
		return
	}

	cx, cy := 0.0, 0.0
	for _, m := range sq.Members {
		if m < len(ss.Bots) {
			cx += ss.Bots[m].X
			cy += ss.Bots[m].Y
		}
	}
	n := float64(len(sq.Members))
	sq.CenterX = cx / n
	sq.CenterY = cy / n

	// Cohesion: inverse of average distance to center
	totalDist := 0.0
	for _, m := range sq.Members {
		if m < len(ss.Bots) {
			dx := ss.Bots[m].X - sq.CenterX
			dy := ss.Bots[m].Y - sq.CenterY
			totalDist += math.Sqrt(dx*dx + dy*dy)
		}
	}
	avgDist := totalDist / n
	sq.Cohesion = 1.0 / (1.0 + avgDist/50.0) // normalized 0-1
}

// electLeaders picks the best-performing bot in each squad as leader.
func electLeaders(ss *SwarmState, hs *HierarchyState) {
	for i := range hs.Squads {
		sq := &hs.Squads[i]
		if len(sq.Members) == 0 {
			continue
		}

		bestFit := -1.0
		bestBot := sq.Members[0]
		for _, m := range sq.Members {
			if m >= len(ss.Bots) {
				continue
			}
			fit := float64(ss.Bots[m].Stats.TotalDeliveries) +
				float64(ss.Bots[m].Stats.TotalPickups)*0.5
			if fit > bestFit {
				bestFit = fit
				bestBot = m
			}
		}
		sq.LeaderID = bestBot
	}
}

// applySquadBehavior makes squad members loosely follow their leader.
func applySquadBehavior(ss *SwarmState, hs *HierarchyState, sq *Squad) {
	if sq.LeaderID < 0 || sq.LeaderID >= len(ss.Bots) {
		return
	}
	leader := &ss.Bots[sq.LeaderID]

	for _, m := range sq.Members {
		if m == sq.LeaderID || m >= len(ss.Bots) {
			continue
		}
		bot := &ss.Bots[m]

		dx := leader.X - bot.X
		dy := leader.Y - bot.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		// Only follow if too far from leader
		if dist > hs.LeaderRange*0.6 {
			angle := math.Atan2(dy, dx)
			diff := angle - bot.Angle
			for diff > math.Pi {
				diff -= 2 * math.Pi
			}
			for diff < -math.Pi {
				diff += 2 * math.Pi
			}
			bot.Angle += diff * 0.05 // gentle steering toward leader
		}

		// Squad color: same hue per squad
		hue := float64(sq.ID%8) / 8.0 * 255
		isLeader := m == sq.LeaderID
		brightness := uint8(150)
		if isLeader {
			brightness = 255
		}
		bot.LEDColor = [3]uint8{
			uint8(hue) | brightness>>2,
			uint8(255-hue) | brightness>>2,
			brightness >> 1,
		}
	}
}

// assignSquadTasks determines what each squad should focus on.
func assignSquadTasks(ss *SwarmState, hs *HierarchyState) {
	for i := range hs.Squads {
		sq := &hs.Squads[i]

		// Count members carrying packages
		carrying := 0
		for _, m := range sq.Members {
			if m < len(ss.Bots) && ss.Bots[m].CarryingPkg >= 0 {
				carrying++
			}
		}

		ratio := 0.0
		if len(sq.Members) > 0 {
			ratio = float64(carrying) / float64(len(sq.Members))
		}

		if ratio > 0.5 {
			sq.Task = TaskDeliver
		} else if ratio > 0 {
			sq.Task = TaskGather
		} else {
			sq.Task = TaskExplore
		}
	}
}

// SquadCount returns the number of squads.
func SquadCount(hs *HierarchyState) int {
	if hs == nil {
		return 0
	}
	return len(hs.Squads)
}

// BotSquadID returns which squad a bot belongs to.
func BotSquadID(hs *HierarchyState, botIdx int) int {
	if hs == nil || botIdx < 0 || botIdx >= len(hs.BotSquad) {
		return -1
	}
	return hs.BotSquad[botIdx]
}

// HierarchyDepth returns the number of hierarchy levels.
func HierarchyDepth(hs *HierarchyState) int {
	if hs == nil {
		return 0
	}
	return len(hs.Levels) + 1 // +1 for squad level
}
