package genetics

// Genome holds evolvable behavior parameters.
type Genome struct {
	FlockingWeight     float64 // 0-1
	PheromoneFollow    float64 // 0-1
	ExplorationDrive   float64 // 0-1
	CommFrequency      float64 // 0-1
	EnergyConservation float64 // 0-1
	SpeedPreference    float64 // 0.5-1.5
	CooperationBias    float64 // 0-1
}

// GenomeLabels returns the label names for HUD display.
func GenomeLabels() [7]string {
	return [7]string{"Flock", "Pher", "Explor", "Comm", "EnCons", "Speed", "Coop"}
}

// Values returns all genome values normalized to 0-1 for display.
func (g *Genome) Values() [7]float64 {
	return [7]float64{
		g.FlockingWeight,
		g.PheromoneFollow,
		g.ExplorationDrive,
		g.CommFrequency,
		g.EnergyConservation,
		(g.SpeedPreference - 0.5), // normalize 0.5-1.5 to 0-1
		g.CooperationBias,
	}
}
