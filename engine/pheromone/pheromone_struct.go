package pheromone

// PheromoneType identifies a pheromone channel.
type PheromoneType int

const (
	PherSearch        PheromoneType = 0 // blue — scouts exploring
	PherFoundResource PheromoneType = 1 // green — path to resource
	PherDanger        PheromoneType = 2 // red — low-health warning
	PherCount                       = 3
)

// PheromoneGrid stores pheromone intensities on a 2D grid.
type PheromoneGrid struct {
	Cols, Rows int
	CellSize   float64
	Data       [PherCount][]float64
	Temp       []float64 // scratch buffer for diffusion
	Decay      float64
	Diffusion  float64
}
