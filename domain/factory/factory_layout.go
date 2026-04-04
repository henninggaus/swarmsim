package factory

import "swarmsim/domain/physics"

// Zone constants define the functional areas of the factory.
// Layout flows left-to-right: WARENEINGANG → LAGER → PRODUKTION → VERSAND → WARENAUSGANG
//
//  +======================== YARD (3000 x 600) =========================+
//  |  [LKW Road ================================================]      |
//  |  [Dock IN-1] [Dock IN-2]              [Dock OUT-1] [Dock OUT-2]   |
//  |  [Gate 1]                                           [Gate 2]      |
//  +===+===========================================================+===+
//  |   |                    FACTORY HALL                            |   |
//  |   |                                                           |   |
//  |   | WARENEINGANG     LAGER        PRODUKTION       VERSAND    |   |
//  |   | (Receiving)    (Storage)     (Production)    (Shipping)   |   |
//  |   |                                                           |   |
//  |   | [Inbound    ]  [Regal 1]   [CNC-1] [CNC-2]  [Outbound ] |   |
//  |   | [Buffer     ]  [Regal 2]                     [Buffer   ] |   |
//  |   | [QC Station ]  [Regal 3]   [Assembly-1   ]  [Pack Stn ] |   |
//  |   |                [Regal 4]   [Assembly-2   ]              |   |
//  |   | [Charger][Charger]          [QC-Final    ]   [Charger]  |   |
//  |   |                                                          |   |
//  |   | [=== WORKSHOP / MAINTENANCE ===]                         |   |
//  +===+===========================================================+===+

const (
	// Zone X boundaries (left-to-right flow)
	ZoneReceivingX  = HallX + 30      // Wareneingang
	ZoneReceivingW  = 350
	ZoneStorageX    = HallX + 420      // Lager
	ZoneStorageW    = 400
	ZoneProductionX = HallX + 860      // Produktion
	ZoneProductionW = 900
	ZoneShippingX   = HallX + 1800     // Versand
	ZoneShippingW   = 350

	// Main aisle Y (horizontal corridor through the hall)
	AisleY = HallY + HallH/2 - 30
	AisleH = 60.0
)

// buildLayout creates the factory walls, machines, chargers, docks, doors, and storage.
func (fs *FactoryState) buildLayout() {
	fs.buildWalls()
	fs.buildDoors()
	fs.buildReceivingArea()
	fs.buildStorageArea()
	fs.buildProductionArea()
	fs.buildShippingArea()
	fs.buildChargers()
	fs.buildWorkshop()
	fs.buildDocks()
}

// buildWalls creates the hall perimeter walls with gate openings.
func (fs *FactoryState) buildWalls() {
	w := WallThick
	gateW := 80.0 // gate width

	// Gate positions in the top wall
	gate1X := HallX + 200.0              // left gate (near receiving)
	gate2X := HallX + HallW - 280.0      // right gate (near shipping)

	fs.Walls = []*physics.Obstacle{
		// Hall — left wall
		{X: HallX - w, Y: HallY, W: w, H: HallH + w},
		// Hall — right wall
		{X: HallX + HallW, Y: HallY, W: w, H: HallH + w},
		// Hall — bottom wall
		{X: HallX - w, Y: HallY + HallH, W: HallW + 2*w, H: w},

		// Hall — top wall with two gate gaps
		// Section 1: left edge to gate 1
		{X: HallX - w, Y: HallY - w, W: gate1X - HallX + w, H: w},
		// Section 2: after gate 1 to gate 2
		{X: gate1X + gateW, Y: HallY - w, W: gate2X - gate1X - gateW, H: w},
		// Section 3: after gate 2 to right edge
		{X: gate2X + gateW, Y: HallY - w, W: HallX + HallW - gate2X - gateW + w, H: w},

		// World boundaries
		{X: 0, Y: 0, W: WorldW, H: w},           // top
		{X: 0, Y: 0, W: w, H: WorldH},            // left
		{X: WorldW - w, Y: 0, W: w, H: WorldH},   // right
		{X: 0, Y: WorldH - w, W: WorldW, H: w},   // bottom

		// Zone dividers — short markers only (NOT full walls, so bots can pass freely)
		// These are visual reference posts at the top and bottom of zone boundaries.
		// Receiving | Storage post
		{X: ZoneStorageX - w/2, Y: HallY, W: w, H: 30},
		{X: ZoneStorageX - w/2, Y: HallY + HallH - 30, W: w, H: 30},
		// Storage | Production post
		{X: ZoneProductionX - w/2, Y: HallY, W: w, H: 30},
		{X: ZoneProductionX - w/2, Y: HallY + HallH - 30, W: w, H: 30},
		// Production | Shipping post
		{X: ZoneShippingX - w/2, Y: HallY, W: w, H: 30},
		{X: ZoneShippingX - w/2, Y: HallY + HallH - 30, W: w, H: 30},
	}
}

// buildDoors creates gate openings between hall and yard.
func (fs *FactoryState) buildDoors() {
	gate1X := HallX + 200.0
	gate2X := HallX + HallW - 280.0

	fs.Doors = []Door{
		{X: gate1X, Y: HallY - WallThick, W: 80, Open: true},     // Gate 1: Receiving side
		{X: gate2X, Y: HallY - WallThick, W: 80, Open: true},     // Gate 2: Shipping side
	}
}

// buildReceivingArea creates the Wareneingang (goods receiving) zone.
func (fs *FactoryState) buildReceivingArea() {
	baseX := ZoneReceivingX
	baseY := HallY + 40

	// Inbound buffer tables (where unloaded goods are placed)
	fs.InboundStorageIdx = len(fs.Storage)
	fs.Storage = append(fs.Storage, StorageArea{
		X: baseX, Y: baseY, W: 200, H: 100, MaxParts: 30,
	})

	// Quality control inspection station (another storage/buffer)
	fs.QCStorageIdx = len(fs.Storage)
	fs.Storage = append(fs.Storage, StorageArea{
		X: baseX, Y: baseY + 200, W: 150, H: 80, MaxParts: 20,
	})
}

// buildStorageArea creates the Lager (warehouse/shelving) zone.
func (fs *FactoryState) buildStorageArea() {
	baseX := ZoneStorageX + 30
	baseY := HallY + 40
	shelfW := 300.0
	shelfH := 50.0
	gap := 70.0

	// 5 shelf rows
	for i := 0; i < 5; i++ {
		fy := baseY + float64(i)*( shelfH + gap)
		if fy+shelfH > AisleY-20 {
			break // don't block the aisle
		}
		fs.Storage = append(fs.Storage, StorageArea{
			X: baseX, Y: fy, W: shelfW, H: shelfH, MaxParts: 40,
		})
	}

	// Lower shelves (below aisle)
	for i := 0; i < 3; i++ {
		fy := AisleY + AisleH + 40 + float64(i)*(shelfH+gap)
		if fy+shelfH > HallY+HallH-80 {
			break
		}
		fs.Storage = append(fs.Storage, StorageArea{
			X: baseX, Y: fy, W: shelfW, H: shelfH, MaxParts: 40,
		})
	}
}

// buildProductionArea creates machines in the Produktion zone.
func (fs *FactoryState) buildProductionArea() {
	baseX := ZoneProductionX + 50
	mw, mh := 120.0, 100.0

	// Upper production line: CNC machines (power cost 0.02)
	fs.Machines = []Machine{
		{X: baseX, Y: HallY + 60, W: mw, H: mh,
			InputColor: 1, OutputColor: 3, MaxInput: 5, ProcessTime: 300, PowerCostPerTick: 0.02},
		{X: baseX + 250, Y: HallY + 60, W: mw, H: mh,
			InputColor: 2, OutputColor: 4, MaxInput: 5, ProcessTime: 250, PowerCostPerTick: 0.02},
	}

	// Middle: Assembly stations (larger, power cost 0.04)
	assemblyW, assemblyH := 200.0, 120.0
	fs.Machines = append(fs.Machines, Machine{
		X: baseX + 50, Y: HallY + 280, W: assemblyW, H: assemblyH,
		InputColor: 3, OutputColor: 1, MaxInput: 8, ProcessTime: 500, PowerCostPerTick: 0.04,
	})

	// Lower production: Drill machines (power cost 0.02)
	fs.Machines = append(fs.Machines, Machine{
		X: baseX, Y: AisleY + AisleH + 60, W: mw, H: mh,
		InputColor: 4, OutputColor: 2, MaxInput: 5, ProcessTime: 200, PowerCostPerTick: 0.02,
	})
	fs.Machines = append(fs.Machines, Machine{
		X: baseX + 250, Y: AisleY + AisleH + 60, W: mw, H: mh,
		InputColor: 1, OutputColor: 4, MaxInput: 5, ProcessTime: 350, PowerCostPerTick: 0.02,
	})

	// QC Final inspection (power cost 0.01)
	fs.Machines = append(fs.Machines, Machine{
		X: baseX + 100, Y: AisleY + AisleH + 250, W: 160, H: 80,
		InputColor: 0, OutputColor: 0, MaxInput: 10, ProcessTime: 100, PowerCostPerTick: 0.01,
	})
}

// buildShippingArea creates the Versand (shipping/outbound) zone.
func (fs *FactoryState) buildShippingArea() {
	baseX := ZoneShippingX + 30
	baseY := HallY + 40

	// Outbound buffer / packing area
	fs.OutboundStorageIdx = len(fs.Storage)
	fs.Storage = append(fs.Storage, StorageArea{
		X: baseX, Y: baseY, W: 200, H: 120, MaxParts: 50,
	})

	// Packing station (another storage buffer)
	fs.Storage = append(fs.Storage, StorageArea{
		X: baseX, Y: baseY + 200, W: 180, H: 80, MaxParts: 30,
	})

	// Lower outbound buffer
	fs.Storage = append(fs.Storage, StorageArea{
		X: baseX, Y: AisleY + AisleH + 60, W: 200, H: 100, MaxParts: 40,
	})
}

// buildChargers places charging stations in corners and along the maintenance area.
func (fs *FactoryState) buildChargers() {
	fs.Chargers = []ChargingStation{
		// Receiving area chargers
		{X: ZoneReceivingX, Y: AisleY + AisleH + 80, W: 50, H: 50, MaxBots: 4, ChargeRate: 2.0},
		{X: ZoneReceivingX + 70, Y: AisleY + AisleH + 80, W: 50, H: 50, MaxBots: 4, ChargeRate: 2.0},

		// Production area chargers
		{X: ZoneProductionX + 600, Y: HallY + HallH - 120, W: 50, H: 50, MaxBots: 4, ChargeRate: 2.0},
		{X: ZoneProductionX + 670, Y: HallY + HallH - 120, W: 50, H: 50, MaxBots: 4, ChargeRate: 2.0},

		// Shipping area charger
		{X: ZoneShippingX + 200, Y: HallY + HallH - 120, W: 50, H: 50, MaxBots: 4, ChargeRate: 2.0},
	}
}

// buildWorkshop places the maintenance workshop at the bottom of the hall.
func (fs *FactoryState) buildWorkshop() {
	fs.Workshop = RepairWorkshop{
		X:          HallX + 40,
		Y:          HallY + HallH - 100,
		W:          300,
		H:          80,
		RepairTime: 200,
		CurrentBot: -1,
	}
}

// buildDocks creates loading docks on the yard side.
func (fs *FactoryState) buildDocks() {
	dockW, dockH := 120.0, 70.0
	dockY := YardH - dockH - 30

	fs.Docks = []LoadingDock{
		// Inbound docks (left side, near receiving gate)
		{X: HallX + 100, Y: dockY, W: dockW, H: dockH, IsInbound: true, TruckIdx: -1},
		{X: HallX + 280, Y: dockY, W: dockW, H: dockH, IsInbound: true, TruckIdx: -1},

		// Outbound docks (right side, near shipping gate)
		{X: HallX + HallW - 400, Y: dockY, W: dockW, H: dockH, IsInbound: false, TruckIdx: -1},
		{X: HallX + HallW - 220, Y: dockY, W: dockW, H: dockH, IsInbound: false, TruckIdx: -1},
	}
}
