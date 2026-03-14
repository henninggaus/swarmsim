package bot

import "swarmsim/domain/resource"

// FindNearestResource returns the nearest available resource, or nil if none.
func FindNearestResource(b Bot, res []*resource.Resource) *resource.Resource {
	var nearest *resource.Resource
	minDist := 1e18
	for _, r := range res {
		if !r.IsAvailable() {
			continue
		}
		d := b.Position().Dist(Vec2{X: r.X, Y: r.Y})
		if d < minDist {
			minDist = d
			nearest = r
		}
	}
	return nearest
}

// IsNearHome returns true if the bot is within range of the home base.
func IsNearHome(b Bot, homeX, homeY, homeR float64) bool {
	return b.Position().Dist(Vec2{X: homeX, Y: homeY}) < homeR+20
}
