package simulation

import (
	"math/rand"
)

// --- Truck layout constants (world coordinates) ---

const (
	// Cabin (visual only, leftmost)
	TruckCabinX = 10.0
	TruckCabinY = 330.0
	TruckCabinW = 150.0
	TruckCabinH = 240.0

	// Cargo area (packages go here)
	TruckCargoX = 160.0
	TruckCargoY = 330.0
	TruckCargoW = 600.0
	TruckCargoH = 240.0

	// Ramp (connects cargo opening to depot area)
	TruckRampX = 760.0
	TruckRampY = 340.0
	TruckRampW = 300.0
	TruckRampH = 220.0
)

// Truck represents the truck's physical layout in the arena.
type Truck struct {
	CabinX, CabinY, CabinW, CabinH float64
	CargoX, CargoY, CargoW, CargoH float64
	RampX, RampY, RampW, RampH     float64
}

// NewTruck creates a truck with the standard layout.
func NewTruck() Truck {
	return Truck{
		CabinX: TruckCabinX, CabinY: TruckCabinY, CabinW: TruckCabinW, CabinH: TruckCabinH,
		CargoX: TruckCargoX, CargoY: TruckCargoY, CargoW: TruckCargoW, CargoH: TruckCargoH,
		RampX: TruckRampX, RampY: TruckRampY, RampW: TruckRampW, RampH: TruckRampH,
	}
}

// CargoRight returns the X coordinate of the cargo opening (right edge).
func (t Truck) CargoRight() float64 { return t.CargoX + t.CargoW }

// --- Sort Zone ---

// SortZoneRect defines a sorting zone's area.
type SortZoneRect struct {
	Zone  SortZone
	X, Y  float64
	W, H  float64
	Label string
}

// ContainsPoint checks if a point is inside the zone rect.
func (z SortZoneRect) ContainsPoint(x, y float64) bool {
	return x >= z.X && x <= z.X+z.W && y >= z.Y && y <= z.Y+z.H
}

// Center returns the center of the zone.
func (z SortZoneRect) Center() (float64, float64) {
	return z.X + z.W/2, z.Y + z.H/2
}

// --- Charging Station ---

// ChargingStation recharges bot energy when nearby.
type ChargingStation struct {
	X, Y   float64
	Radius float64
}

// --- Depot ---

// Depot holds sorting zones and charging stations.
type Depot struct {
	Zones    [4]SortZoneRect
	Chargers [3]ChargingStation
}

// NewDepot creates the standard depot layout.
func NewDepot() Depot {
	return Depot{
		Zones: [4]SortZoneRect{
			{ZoneA, 1100, 50, 150, 150, "A"},
			{ZoneB, 1300, 50, 150, 150, "B"},
			{ZoneC, 1100, 250, 150, 150, "C"},
			{ZoneD, 1300, 250, 150, 150, "D"},
		},
		Chargers: [3]ChargingStation{
			{1100, 550, 50},
			{1300, 550, 50},
			{1200, 750, 50},
		},
	}
}

// --- Bot Task (truck-mode AI state per bot) ---

// TruckTaskType describes what a bot is doing in truck mode.
type TruckTaskType int

const (
	TaskIdle         TruckTaskType = iota
	TaskScanning                   // Scout: exploring truck
	TaskGoToPackage                // Moving toward a package
	TaskLifting                    // Lifting animation
	TaskCarrying                   // Transporting to zone
	TaskDelivering                 // Placing in zone
	TaskWaitingHelp                // Waiting for cooperative lift
	TaskCoordinating               // Leader: managing ramp traffic
	TaskHealing                    // Healer: recharging bots
	TaskRecharging                 // Going to charging station
)

// BotTask tracks what a specific bot is doing in truck mode.
type BotTask struct {
	Type        TruckTaskType
	TargetPkgID int // package ID, -1 if none
	TargetX     float64
	TargetY     float64
	CoopGroup   []int // other bot IDs cooperating
	SubState    int   // internal state machine step
}

// --- TruckState ---

// TruckState holds all state for an active truck scenario.
type TruckState struct {
	Truck         Truck
	Depot         Depot
	Packages      []*Package
	NextPkgID     int
	CarryCaps     map[int]float64  // botID -> carrying capacity
	BotTasks      map[int]*BotTask // botID -> current task
	Timer         int              // ticks elapsed
	TotalPkgs     int
	DeliveredPkgs int
	CorrectZone   int
	WrongZone     int
	Completed     bool
}

// NewTruckState creates and initializes a fresh truck scenario.
func NewTruckState(rng *rand.Rand, packageCount int) *TruckState {
	ts := &TruckState{
		Truck:     NewTruck(),
		Depot:     NewDepot(),
		CarryCaps: make(map[int]float64),
		BotTasks:  make(map[int]*BotTask),
	}
	ts.GeneratePackages(rng, packageCount)
	return ts
}

// GeneratePackages places random packages in the truck cargo using column-based packing.
// Packages are placed from left (deepest) to right (near opening).
func (ts *TruckState) GeneratePackages(rng *rand.Rand, count int) {
	defs := PackageDefs()
	cargo := ts.Truck

	margin := 15.0
	gap := 5.0

	placed := 0
	cursorX := cargo.CargoX + margin

	for placed < count && cursorX < cargo.CargoX+cargo.CargoW-margin {
		cursorY := cargo.CargoY + margin
		colWidth := 0.0

		for cursorY < cargo.CargoY+cargo.CargoH-margin && placed < count {
			defIdx := rng.Intn(6)
			def := defs[defIdx]

			// Check if package fits vertically
			if cursorY+def.Height > cargo.CargoY+cargo.CargoH-margin {
				break
			}
			// Check if package fits horizontally
			if cursorX+def.Width > cargo.CargoX+cargo.CargoW-margin {
				break
			}

			pkg := &Package{
				ID:    ts.NextPkgID,
				Def:   def,
				X:     cursorX + def.Width/2,
				Y:     cursorY + def.Height/2,
				State: PkgInTruck,
			}
			ts.NextPkgID++
			ts.Packages = append(ts.Packages, pkg)
			placed++

			cursorY += def.Height + gap
			if def.Width > colWidth {
				colWidth = def.Width
			}
		}
		if colWidth == 0 {
			colWidth = 40 // safety fallback
		}
		cursorX += colWidth + gap
	}
	ts.TotalPkgs = len(ts.Packages)
}

// IsInRamp checks if world position (x,y) is on the ramp.
func (ts *TruckState) IsInRamp(x, y float64) bool {
	r := ts.Truck
	return x >= r.RampX && x <= r.RampX+r.RampW &&
		y >= r.RampY && y <= r.RampY+r.RampH
}

// IsInCargo checks if world position (x,y) is in the cargo area.
func (ts *TruckState) IsInCargo(x, y float64) bool {
	c := ts.Truck
	return x >= c.CargoX && x <= c.CargoX+c.CargoW &&
		y >= c.CargoY && y <= c.CargoY+c.CargoH
}

// FindZoneAt returns the sort zone at position (x,y), or false if none.
func (ts *TruckState) FindZoneAt(x, y float64) (SortZone, bool) {
	for _, z := range ts.Depot.Zones {
		if z.ContainsPoint(x, y) {
			return z.Zone, true
		}
	}
	return ZoneA, false
}

// AccessiblePackages returns packages that can be picked up right now.
func (ts *TruckState) AccessiblePackages() []*Package {
	cargoRight := ts.Truck.CargoRight()
	var result []*Package
	for _, p := range ts.Packages {
		if p.State == PkgInTruck && p.IsAccessible(ts.Packages, cargoRight) {
			result = append(result, p)
		}
	}
	return result
}

// GetPackageByID returns a package by ID or nil.
func (ts *TruckState) GetPackageByID(id int) *Package {
	for _, p := range ts.Packages {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// GetBotTask returns or creates a BotTask for a given bot ID.
func (ts *TruckState) GetBotTask(botID int) *BotTask {
	task, ok := ts.BotTasks[botID]
	if !ok {
		task = &BotTask{Type: TaskIdle, TargetPkgID: -1}
		ts.BotTasks[botID] = task
	}
	return task
}

// ZoneCenter returns the center position of a sort zone.
func (ts *TruckState) ZoneCenter(zone SortZone) (float64, float64) {
	for _, z := range ts.Depot.Zones {
		if z.Zone == zone {
			return z.Center()
		}
	}
	return 1200, 200 // fallback
}

// NearestCharger returns the position of the nearest charging station.
func (ts *TruckState) NearestCharger(x, y float64) (float64, float64) {
	bestDist := 1e9
	bx, by := ts.Depot.Chargers[0].X, ts.Depot.Chargers[0].Y
	for _, c := range ts.Depot.Chargers {
		dx := c.X - x
		dy := c.Y - y
		dist := dx*dx + dy*dy
		if dist < bestDist {
			bestDist = dist
			bx, by = c.X, c.Y
		}
	}
	return bx, by
}

// CountBotsOnRamp returns how many bots (by ID) are currently on the ramp.
func (ts *TruckState) CountBotsOnRamp(positions map[int][2]float64) int {
	count := 0
	for _, pos := range positions {
		if ts.IsInRamp(pos[0], pos[1]) {
			count++
		}
	}
	return count
}
