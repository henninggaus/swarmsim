package factory

import "swarmsim/locale"

const (
	TruckSpawnInterval = 800 // ticks between inbound truck arrivals (faster for 1000 bots)
	TruckSpeed         = 2.0
	TruckW             = 120.0
	TruckH             = 50.0
	TruckMinParts      = 5
	TruckMaxParts      = 10
	OutboundThreshold  = 5 // outbound storage items needed to spawn outbound truck
	OutboundTimeout    = 3000 // ticks before outbound truck departs even if not full
)

// TickTrucks manages truck arrivals, parking, and departures.
func TickTrucks(fs *FactoryState) {
	// Spawn inbound trucks on timer (Feature 11: skip during Supply Shortage)
	fs.TruckTimer--
	if fs.TruckTimer <= 0 {
		fs.TruckTimer = TruckSpawnInterval
		if !IsEventActive(fs, EventSupplyShortage) {
			spawnInboundTruck(fs)
		}
	}

	// Check if outbound truck is needed
	checkOutboundTruck(fs)

	// Update each truck
	for i := range fs.Trucks {
		truck := &fs.Trucks[i]
		updateTruck(fs, truck, i)
	}

	// Clean up exited trucks
	cleanupTrucks(fs)
}

func spawnInboundTruck(fs *FactoryState) {
	// Generate random parts
	numParts := TruckMinParts + fs.Rng.Intn(TruckMaxParts-TruckMinParts+1)
	parts := make([]int, numParts)
	for j := range parts {
		parts[j] = 1 + fs.Rng.Intn(4) // colors 1-4
	}

	truck := FactoryTruck{
		X:         -TruckW, // start off-screen left
		Y:         RoadY - TruckH/2,
		W:         TruckW,
		H:         TruckH,
		Speed:     TruckSpeed,
		Direction: 0, // moving right
		Phase:     TruckEntering,
		DockIdx:   -1,
		Parts:     parts,
		MaxParts:  TruckMaxParts,
	}
	fs.Trucks = append(fs.Trucks, truck)
}

func checkOutboundTruck(fs *FactoryState) {
	if fs.OutboundStorageIdx >= len(fs.Storage) {
		return
	}
	outStorage := &fs.Storage[fs.OutboundStorageIdx]
	// Use Slots as the primary count; fall back to Parts for legacy compat
	outCount := len(outStorage.Slots)
	if outCount == 0 {
		outCount = len(outStorage.Parts)
	}
	if outCount < OutboundThreshold {
		return
	}

	// Check if an outbound truck already exists and is not exiting
	for i := range fs.Trucks {
		t := &fs.Trucks[i]
		if t.Direction == 1 && t.Phase != TruckExiting {
			return // already have an outbound truck
		}
	}

	// Find an available outbound dock
	dockIdx := -1
	for di := range fs.Docks {
		if !fs.Docks[di].IsInbound && fs.Docks[di].TruckIdx < 0 {
			dockIdx = di
			break
		}
	}
	if dockIdx < 0 {
		return // no dock available
	}

	truck := FactoryTruck{
		X:         WorldW, // start off-screen right
		Y:         RoadY - TruckH/2,
		W:         TruckW,
		H:         TruckH,
		Speed:     TruckSpeed,
		Direction: 1, // moving left (inbound to dock)
		Phase:     TruckEntering,
		DockIdx:   -1,
		MaxParts:  TruckMaxParts,
	}
	fs.Trucks = append(fs.Trucks, truck)
}

func updateTruck(fs *FactoryState, truck *FactoryTruck, truckIdx int) {
	// Track movement ticks for exhaust acceleration
	if truck.Phase == TruckEntering || truck.Phase == TruckExiting {
		truck.MoveTick++
	} else {
		truck.MoveTick = 0
	}

	switch truck.Phase {
	case TruckEntering:
		// Drive toward an available dock
		if truck.Direction == 0 {
			// Inbound: drive right
			truck.X += truck.Speed

			// Find available inbound dock
			if truck.DockIdx < 0 {
				for di := range fs.Docks {
					if fs.Docks[di].IsInbound && fs.Docks[di].TruckIdx < 0 {
						truck.DockIdx = di
						fs.Docks[di].TruckIdx = truckIdx
						break
					}
				}
			}

			// Check if arrived at dock
			if truck.DockIdx >= 0 {
				dock := &fs.Docks[truck.DockIdx]
				targetX := dock.X + dock.W/2 - truck.W/2
				if truck.X >= targetX {
					truck.X = targetX
					truck.Phase = TruckUnloading
					truck.Speed = 0
					// Feature: Material Cost — deduct cost when inbound truck arrives
					cost := float64(len(truck.Parts)) * fs.MaterialCostPerPart
					fs.TotalMaterialCost += cost
					fs.Budget -= cost
					AddAlert(fs, locale.Tf("factory.alert.truck_arrived", truckIdx+1, truck.DockIdx+1), [3]uint8{80, 200, 220})
				}
			} else if truck.X > WorldW+TruckW {
				// No dock available, truck drives off
				truck.Phase = TruckExiting
			}
		} else {
			// Outbound: drive left toward dock
			truck.X -= truck.Speed

			if truck.DockIdx < 0 {
				for di := range fs.Docks {
					if !fs.Docks[di].IsInbound && fs.Docks[di].TruckIdx < 0 {
						truck.DockIdx = di
						fs.Docks[di].TruckIdx = truckIdx
						break
					}
				}
			}

			if truck.DockIdx >= 0 {
				dock := &fs.Docks[truck.DockIdx]
				targetX := dock.X + dock.W/2 - truck.W/2
				if truck.X <= targetX {
					truck.X = targetX
					truck.Phase = TruckLoading
					truck.Speed = 0
					truck.Counter = 0 // use as timeout counter
				}
			} else if truck.X < -TruckW*2 {
				truck.Phase = TruckExiting
			}
		}

	case TruckUnloading:
		// Parts are removed by bots completing TaskUnloadTruck
		// When empty, depart
		if len(truck.Parts) == 0 {
			truck.Phase = TruckExiting
			truck.Speed = TruckSpeed
			truck.Direction = 0 // drive off to the right
			if truck.DockIdx >= 0 && truck.DockIdx < len(fs.Docks) {
				fs.Docks[truck.DockIdx].TruckIdx = -1
			}
			fs.Stats.TrucksUnloaded++
		}

	case TruckLoading:
		truck.Counter++
		// Truck departs when full or timeout
		if len(truck.Parts) >= truck.MaxParts || truck.Counter > OutboundTimeout {
			truck.Phase = TruckExiting
			truck.Speed = TruckSpeed
			truck.Direction = 0 // drive off to the right
			if truck.DockIdx >= 0 && truck.DockIdx < len(fs.Docks) {
				fs.Docks[truck.DockIdx].TruckIdx = -1
			}
			if len(truck.Parts) > 0 {
				fs.Stats.TrucksLoaded++
			}
		}

	case TruckExiting:
		truck.X += truck.Speed * 2 // drive off faster
	}
}

// ForceSpawnInboundTruck spawns an inbound truck immediately (manual dispatch).
func ForceSpawnInboundTruck(fs *FactoryState) {
	spawnInboundTruck(fs)
}

// ForceSpawnOutboundTruck spawns an outbound truck immediately (manual dispatch).
func ForceSpawnOutboundTruck(fs *FactoryState) {
	// Find available outbound dock
	dockIdx := -1
	for di := range fs.Docks {
		if !fs.Docks[di].IsInbound && fs.Docks[di].TruckIdx < 0 {
			dockIdx = di
			break
		}
	}
	_ = dockIdx // we spawn regardless, it will find a dock during update

	truck := FactoryTruck{
		X:         WorldW,
		Y:         RoadY - TruckH/2,
		W:         TruckW,
		H:         TruckH,
		Speed:     TruckSpeed,
		Direction: 1,
		Phase:     TruckEntering,
		DockIdx:   -1,
		MaxParts:  TruckMaxParts,
	}
	fs.Trucks = append(fs.Trucks, truck)
}

func cleanupTrucks(fs *FactoryState) {
	n := 0
	for i := range fs.Trucks {
		if fs.Trucks[i].Phase == TruckExiting && fs.Trucks[i].X > WorldW+TruckW*2 {
			// Remove truck — but first fix dock references
			continue
		}
		if n != i {
			fs.Trucks[n] = fs.Trucks[i]
			// Fix dock references to the moved truck
			for di := range fs.Docks {
				if fs.Docks[di].TruckIdx == i {
					fs.Docks[di].TruckIdx = n
				}
			}
		}
		n++
	}
	fs.Trucks = fs.Trucks[:n]
}
