package simulation

// PackageType identifies a package category.
type PackageType int

const (
	PkgSmallBox  PackageType = iota // 20x20, 2kg
	PkgMediumBox                    // 35x35, 8kg
	PkgLargeBox                     // 50x50, 15kg
	PkgFragile                      // 25x25, 5kg
	PkgPallet                       // 60x60, 30kg
	PkgLongItem                     // 80x25, 12kg
)

func (t PackageType) String() string {
	switch t {
	case PkgSmallBox:
		return "SmallBox"
	case PkgMediumBox:
		return "MediumBox"
	case PkgLargeBox:
		return "LargeBox"
	case PkgFragile:
		return "Fragile"
	case PkgPallet:
		return "Pallet"
	case PkgLongItem:
		return "LongItem"
	}
	return "Unknown"
}

// SortZone identifies a depot sorting zone.
type SortZone int

const (
	ZoneA SortZone = iota
	ZoneB
	ZoneC
	ZoneD
)

func (z SortZone) String() string {
	switch z {
	case ZoneA:
		return "A"
	case ZoneB:
		return "B"
	case ZoneC:
		return "C"
	case ZoneD:
		return "D"
	}
	return "?"
}

// PackageState tracks a package's lifecycle.
type PackageState int

const (
	PkgInTruck   PackageState = iota // sitting in cargo area
	PkgLifting                       // being lifted (15-tick animation)
	PkgCarried                       // being carried by bot(s)
	PkgDelivered                     // placed in a depot zone
)

// PackageDef holds the static definition for each package type.
type PackageDef struct {
	Type        PackageType
	Name        string
	Width       float64 // pixels (cm)
	Height      float64
	Weight      float64 // kg
	MinCarryCap float64 // minimum carrying capacity needed
	Zone        SortZone
	ColorR      uint8
	ColorG      uint8
	ColorB      uint8
}

// PackageDefs returns definitions for all 6 package types.
func PackageDefs() [6]PackageDef {
	return [6]PackageDef{
		{PkgSmallBox, "SmallBox", 20, 20, 2, 0.5, ZoneA, 160, 120, 80},
		{PkgMediumBox, "MediumBox", 35, 35, 8, 3.0, ZoneB, 120, 85, 50},
		{PkgLargeBox, "LargeBox", 50, 50, 15, 6.0, ZoneC, 200, 150, 80},
		{PkgFragile, "Fragile", 25, 25, 5, 2.0, ZoneA, 220, 50, 50},
		{PkgPallet, "Pallet", 60, 60, 30, 8.0, ZoneD, 150, 150, 150},
		{PkgLongItem, "LongItem", 80, 25, 12, 4.0, ZoneC, 80, 120, 200},
	}
}

// Package represents a single package instance in the truck/arena.
type Package struct {
	ID            int
	Def           PackageDef
	X, Y          float64 // world position (center)
	State         PackageState
	CarrierBotIDs []int // bot(s) currently carrying/lifting
	LiftTick      int   // ticks remaining in lift animation
	Delivered     bool
	DeliveredZone SortZone // which zone it was delivered to
	CorrectZone   bool     // was it delivered to the right zone?
}

// IsAccessible returns true if no package blocks this one from being extracted.
// A package is blocked if any other non-removed package sits between it and the
// cargo opening (cargoRight) with vertical overlap.
func (p *Package) IsAccessible(packages []*Package, cargoRight float64) bool {
	if p.State != PkgInTruck {
		return false
	}

	pRight := p.X + p.Def.Width/2
	pTop := p.Y - p.Def.Height/2
	pBot := p.Y + p.Def.Height/2

	for _, other := range packages {
		if other.ID == p.ID {
			continue
		}
		if other.State == PkgDelivered || other.State == PkgCarried || other.State == PkgLifting {
			continue
		}

		otherLeft := other.X - other.Def.Width/2
		if otherLeft < pRight {
			continue // other is behind or beside, not blocking
		}

		// Check vertical overlap
		oTop := other.Y - other.Def.Height/2
		oBot := other.Y + other.Def.Height/2

		if pBot > oTop && pTop < oBot {
			return false // blocked
		}
	}
	return true
}
