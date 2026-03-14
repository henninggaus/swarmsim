package bot

// FlockingParams holds the tuning constants for boids flocking.
type FlockingParams struct {
	SeparationDist   float64
	AlignmentDist    float64
	CohesionDist     float64
	SeparationWeight float64
	AlignmentWeight  float64
	CohesionWeight   float64
}

// DefaultFlockingParams returns the default flocking parameters.
func DefaultFlockingParams() FlockingParams {
	return FlockingParams{
		SeparationDist:   30,
		AlignmentDist:    60,
		CohesionDist:     80,
		SeparationWeight: 1.5,
		AlignmentWeight:  0.3,
		CohesionWeight:   0.3,
	}
}

// ComputeFlocking computes the boids steering vector for a bot given its neighbors.
func ComputeFlocking(self Bot, nearby []Bot, params FlockingParams) Vec2 {
	sep := computeSeparation(self, nearby, params.SeparationDist)
	ali := computeAlignment(self, nearby, params.AlignmentDist)
	coh := computeCohesion(self, nearby, params.CohesionDist)

	return sep.Scale(params.SeparationWeight).
		Add(ali.Scale(params.AlignmentWeight)).
		Add(coh.Scale(params.CohesionWeight))
}

func computeSeparation(self Bot, nearby []Bot, dist float64) Vec2 {
	var steer Vec2
	count := 0
	for _, other := range nearby {
		if other.ID() == self.ID() || !other.IsAlive() {
			continue
		}
		d := self.Position().Dist(other.Position())
		if d > 0 && d < dist {
			diff := self.Position().Sub(other.Position()).Normalized().Scale(1.0 / d)
			steer = steer.Add(diff)
			count++
		}
	}
	if count > 0 {
		steer = steer.Scale(1.0 / float64(count))
	}
	return steer
}

func computeAlignment(self Bot, nearby []Bot, dist float64) Vec2 {
	var avg Vec2
	count := 0
	for _, other := range nearby {
		if other.ID() == self.ID() || !other.IsAlive() {
			continue
		}
		if self.Position().Dist(other.Position()) < dist {
			avg = avg.Add(other.Velocity())
			count++
		}
	}
	if count == 0 {
		return Vec2{}
	}
	avg = avg.Scale(1.0 / float64(count))
	return avg.Normalized()
}

func computeCohesion(self Bot, nearby []Bot, dist float64) Vec2 {
	var center Vec2
	count := 0
	for _, other := range nearby {
		if other.ID() == self.ID() || !other.IsAlive() {
			continue
		}
		if self.Position().Dist(other.Position()) < dist {
			center = center.Add(other.Position())
			count++
		}
	}
	if count == 0 {
		return Vec2{}
	}
	center = center.Scale(1.0 / float64(count))
	return center.Sub(self.Position()).Normalized()
}
