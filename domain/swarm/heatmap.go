package swarm

const HeatmapCellSize = 20.0

// InitHeatmap initializes the heatmap grid for the arena.
func InitHeatmap(ss *SwarmState) {
	ss.HeatmapCols = int(ss.ArenaW/HeatmapCellSize) + 1
	ss.HeatmapRows = int(ss.ArenaH/HeatmapCellSize) + 1
	ss.HeatmapGrid = make([]float64, ss.HeatmapCols*ss.HeatmapRows)
}

// UpdateHeatmap records bot positions into the heatmap grid.
// Called every tick when ShowHeatmap is true.
func UpdateHeatmap(ss *SwarmState) {
	if ss.HeatmapGrid == nil {
		InitHeatmap(ss)
	}
	for i := range ss.Bots {
		col := int(ss.Bots[i].X / HeatmapCellSize)
		row := int(ss.Bots[i].Y / HeatmapCellSize)
		if col >= 0 && col < ss.HeatmapCols && row >= 0 && row < ss.HeatmapRows {
			ss.HeatmapGrid[row*ss.HeatmapCols+col] += 1.0
		}
	}
}

// ClearHeatmap resets the heatmap grid.
func ClearHeatmap(ss *SwarmState) {
	for i := range ss.HeatmapGrid {
		ss.HeatmapGrid[i] = 0
	}
}
