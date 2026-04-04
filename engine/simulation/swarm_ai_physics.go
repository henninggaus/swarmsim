package simulation

import (
	"math"
	"swarmsim/domain/physics"
	"swarmsim/domain/swarm"
	"swarmsim/logger"
)

// applyFollowBehavior steers followers toward their leader.
func applyFollowBehavior(ss *swarm.SwarmState, i int) {
	bot := &ss.Bots[i]
	if bot.FollowTargetIdx < 0 || bot.FollowTargetIdx >= len(ss.Bots) {
		return
	}

	target := &ss.Bots[bot.FollowTargetIdx]

	// Validate link integrity
	if target.FollowerIdx != i {
		bot.FollowTargetIdx = -1
		return
	}

	// Measure direct distance to leader
	dx, dy := swarm.NeighborDelta(bot.X, bot.Y, target.X, target.Y, ss)
	leaderDist := math.Sqrt(dx*dx + dy*dy)

	// Break link if leader is too far away (lost contact)
	if leaderDist > swarm.SwarmSensorRange*1.5 {
		target.FollowerIdx = -1
		bot.FollowTargetIdx = -1
		return
	}

	// Steer directly toward leader and maintain ~20px distance
	desiredDist := 20.0
	if leaderDist < desiredDist-5 {
		// Too close — stop and wait
		bot.Speed = 0
	} else if leaderDist < desiredDist+5 {
		// In sweet spot — match leader speed and heading
		bot.Angle = math.Atan2(dy, dx)
		bot.Speed = target.Speed
		if bot.Speed < swarm.SwarmBotSpeed*0.3 {
			bot.Speed = swarm.SwarmBotSpeed * 0.3
		}
	} else {
		// Too far — chase leader directly at full speed
		bot.Angle = math.Atan2(dy, dx)
		bot.Speed = swarm.SwarmBotSpeed
	}
}

// applySwarmPhysics moves a bot and handles boundary collisions + separation.
func applySwarmPhysics(ss *swarm.SwarmState, i int) {
	bot := &ss.Bots[i]

	// Dash timer: double speed while active
	if bot.DashTimer > 0 {
		bot.DashTimer--
		bot.Speed *= 2.0
	}
	if bot.DashCooldown > 0 {
		bot.DashCooldown--
	}

	// Brake timer: reduce speed over 3 ticks
	if bot.BrakeTimer > 0 {
		bot.Speed *= float64(bot.BrakeTimer-1) / 3.0
		bot.BrakeTimer--
	}

	// Move
	if bot.Speed > 0 {
		bot.X += math.Cos(bot.Angle) * bot.Speed
		bot.Y += math.Sin(bot.Angle) * bot.Speed
	}

	// Obstacle collision — resolve overlap and wall-slide redirection
	allObs := ss.AllObstacles()
	for _, obs := range allObs {
		hit, _, _ := physics.CircleRectCollision(bot.X, bot.Y, swarm.SwarmBotRadius, obs.X, obs.Y, obs.W, obs.H)
		if hit {
			ss.CollisionCount++
			bot.CollisionTimer = 10 // trigger recent_collision sensor for 10 ticks
			newX, newY := physics.ResolveCircleRectOverlap(bot.X, bot.Y, swarm.SwarmBotRadius, obs.X, obs.Y, obs.W, obs.H)
			pushDx := newX - bot.X
			pushDy := newY - bot.Y
			pushLen := math.Sqrt(pushDx*pushDx + pushDy*pushDy)
			if pushLen > 0.1 {
				// Compute wall normal (direction bot was pushed out)
				nx := pushDx / pushLen
				ny := pushDy / pushLen
				// Check if heading into wall
				hx := math.Cos(bot.Angle)
				hy := math.Sin(bot.Angle)
				dot := hx*nx + hy*ny
				if dot < 0 {
					// Wall-slide: remove normal component from heading so bot slides along wall
					hx -= dot * nx
					hy -= dot * ny
					slideLen := math.Sqrt(hx*hx + hy*hy)
					if slideLen > 0.01 {
						bot.Angle = math.Atan2(hy, hx)
					} else {
						// Head-on collision: random tangent direction
						bot.Angle += math.Pi/2 + ss.Rng.Float64()*math.Pi
					}
				}
			}
			bot.X = newX
			bot.Y = newY
		}
	}

	// Ramp barrier — prevent bots from entering ramp zone (trucks only)
	if ss.TruckToggle && ss.TruckState != nil {
		rampRight := swarm.SwarmRampX + swarm.SwarmRampW  // right edge of ramp (200)
		rampTop := swarm.SwarmRampY                       // 200
		rampBottom := swarm.SwarmRampY + swarm.SwarmRampH // 550
		br := swarm.SwarmBotRadius
		// If bot circle overlaps ramp rectangle, push it out to the right
		if bot.X-br < rampRight && bot.Y+br > rampTop && bot.Y-br < rampBottom {
			bot.X = rampRight + br
		}
	}

	// Boundary handling
	r := swarm.SwarmBotRadius
	if ss.WrapMode {
		// Toroidal wrap
		if bot.X < 0 {
			bot.X += ss.ArenaW
		}
		if bot.X > ss.ArenaW {
			bot.X -= ss.ArenaW
		}
		if bot.Y < 0 {
			bot.Y += ss.ArenaH
		}
		if bot.Y > ss.ArenaH {
			bot.Y -= ss.ArenaH
		}
	} else {
		// Bounce — clamp position, then redirect angle (but NOT for followers)
		hitEdge := false

		// Check if bot is in the ramp entrance zone (exempt from left-edge bounce when trucks active)
		inRampZone := ss.TruckToggle && ss.TruckState != nil &&
			bot.Y >= swarm.SwarmRampY-r && bot.Y <= swarm.SwarmRampY+swarm.SwarmRampH+r

		if bot.X < r {
			if inRampZone {
				// Allow bots to reach X=0 (ramp area) — only soft clamp
				if bot.X < -r {
					bot.X = -r
					hitEdge = true
				}
			} else {
				bot.X = r
				hitEdge = true
			}
		}
		if bot.X > ss.ArenaW-r {
			bot.X = ss.ArenaW - r
			hitEdge = true
		}
		if bot.Y < r {
			bot.Y = r
			hitEdge = true
		}
		if bot.Y > ss.ArenaH-r {
			bot.Y = ss.ArenaH - r
			hitEdge = true
		}
		// Only redirect angle for FREE bots (not following anyone).
		// Followers get their angle from applyFollowBehavior — overwriting
		// it here would steer them away from their leader and break chains!
		if hitEdge && bot.FollowTargetIdx < 0 {
			// Wall-reflection: bounce like a billiard ball to spread bots evenly.
			// (Old center-aim code made ALL bots converge to the middle.)
			if bot.X <= r || bot.X >= ss.ArenaW-r {
				bot.Angle = math.Pi - bot.Angle // flip horizontal component
			}
			if bot.Y <= r || bot.Y >= ss.ArenaH-r {
				bot.Angle = -bot.Angle // flip vertical component
			}
			// Random perturbation to avoid deterministic ping-pong paths
			bot.Angle += (ss.Rng.Float64() - 0.5) * math.Pi / 3 // ±30°
		}
	}

	// Normalize angle to [0, 2π)
	for bot.Angle < 0 {
		bot.Angle += 2 * math.Pi
	}
	for bot.Angle >= 2*math.Pi {
		bot.Angle -= 2 * math.Pi
	}
}

// applyHardSeparation is a symmetric rigid-body pass that guarantees no two
// bots overlap. Both bots in a pair are pushed apart to exactly minDist.
// Runs twice per tick to resolve multi-bot clusters (triangles etc.).
func applyHardSeparation(ss *swarm.SwarmState) {
	const minDist = swarm.SwarmBotRadius * 2.4 // 24px hard shell

	for iter := 0; iter < 2; iter++ {
		// Rebuild spatial hash each iteration (positions shifted)
		ss.Hash.Clear()
		for i := range ss.Bots {
			ss.Hash.Insert(i, ss.Bots[i].X, ss.Bots[i].Y)
		}

		for i := range ss.Bots {
			a := &ss.Bots[i]
			nearIDs := ss.Hash.Query(a.X, a.Y, minDist+1)
			for _, j := range nearIDs {
				if j <= i || j >= len(ss.Bots) {
					continue // each pair once (j > i)
				}
				// Skip directly linked follower↔leader pairs
				if a.FollowTargetIdx == j || a.FollowerIdx == j {
					continue
				}
				b := &ss.Bots[j]
				dx := a.X - b.X
				dy := a.Y - b.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist >= minDist {
					continue
				}
				if dist < 0.001 {
					// Coincident — nudge with random direction
					angle := ss.Rng.Float64() * 2 * math.Pi
					dx = math.Cos(angle)
					dy = math.Sin(angle)
					dist = 0.001
				}
				// Push BOTH apart to full minDist (each gets half)
				nx := dx / dist
				ny := dy / dist
				half := (minDist - dist) * 0.5
				a.X += nx * half
				a.Y += ny * half
				b.X -= nx * half
				b.Y -= ny * half

				// Elastic heading deflection when very close
				if dist < 12 {
					a.Angle = math.Atan2(ny, nx) + (ss.Rng.Float64()-0.5)*0.3
					b.Angle = math.Atan2(-ny, -nx) + (ss.Rng.Float64()-0.5)*0.3
				}
			}
		}
	}
}

// applyRepulsionForce adds a continuous push between bots closer than 30px.
// Unlike hard separation (which resolves overlap), this creates an active
// force field that prevents clustering before contact.
func applyRepulsionForce(ss *swarm.SwarmState) {
	const baseRepulsionRange = 30.0
	const rampRepulsionRange = 40.0
	const repulsionStrength = 0.15

	for i := range ss.Bots {
		a := &ss.Bots[i]
		// Use wider repulsion range on ramp to prevent clustering
		repRange := baseRepulsionRange
		if a.OnRamp {
			repRange = rampRepulsionRange
		}
		nearIDs := ss.Hash.Query(a.X, a.Y, repRange+1)
		for _, j := range nearIDs {
			if j <= i || j >= len(ss.Bots) {
				continue
			}
			// Skip linked pairs
			if a.FollowTargetIdx == j || a.FollowerIdx == j {
				continue
			}
			b := &ss.Bots[j]
			dx := a.X - b.X
			dy := a.Y - b.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			// Use wider range if either bot is on ramp
			effectiveRange := baseRepulsionRange
			if a.OnRamp || b.OnRamp {
				effectiveRange = rampRepulsionRange
			}
			if dist >= effectiveRange || dist < 0.001 {
				continue
			}
			// Force = (range - dist) * strength, applied symmetrically
			force := (effectiveRange - dist) * repulsionStrength
			nx := dx / dist
			ny := dy / dist
			halfForce := force * 0.5
			a.X += nx * halfForce
			a.Y += ny * halfForce
			b.X -= nx * halfForce
			b.Y -= ny * halfForce
		}
	}
}

// applyStationRepulsion pushes non-carrying bots away from dropoff stations.
// Bots without a matching package within 50px of a dropoff get a force pushing them away.
// Only bots carrying a matching package may approach.
func applyStationRepulsion(ss *swarm.SwarmState) {
	if !ss.DeliveryOn {
		return
	}
	const stationRepRange = 50.0
	const stationRepStrength = 0.2

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Bots carrying a matching package may approach dropoffs freely
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			pkg := &ss.Packages[bot.CarryingPkg]
			if bot.NearestDropoffIdx >= 0 && bot.NearestDropoffIdx < len(ss.Stations) {
				st := &ss.Stations[bot.NearestDropoffIdx]
				if pkg.Color == st.Color {
					continue // carrying matching package — don't repel
				}
			}
		}

		// Push non-carrying (or wrong-color-carrying) bots away from all nearby dropoffs
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup {
				continue
			}
			dx := bot.X - st.X
			dy := bot.Y - st.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist >= stationRepRange || dist < 0.1 {
				continue
			}
			// Force = (50 - dist) * 0.2, directed away from station
			force := (stationRepRange - dist) * stationRepStrength
			nx := dx / dist
			ny := dy / dist
			bot.X += nx * force
			bot.Y += ny * force
		}
	}
}

// applyClusterBreaker detects large clusters (connected components via 30px radius)
// and applies an outward explosion impulse to all bots in clusters > 5.
func applyClusterBreaker(ss *swarm.SwarmState) {
	n := len(ss.Bots)
	if n == 0 {
		return
	}

	// Union-Find
	parent := make([]int, n)
	rank := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra == rb {
			return
		}
		if rank[ra] < rank[rb] {
			ra, rb = rb, ra
		}
		parent[rb] = ra
		if rank[ra] == rank[rb] {
			rank[ra]++
		}
	}

	// Build clusters: bots within 30px are connected
	const clusterRadius = 30.0
	for i := range ss.Bots {
		a := &ss.Bots[i]
		nearIDs := ss.Hash.Query(a.X, a.Y, clusterRadius+1)
		for _, j := range nearIDs {
			if j <= i || j >= n {
				continue
			}
			b := &ss.Bots[j]
			dx := a.X - b.X
			dy := a.Y - b.Y
			if dx*dx+dy*dy < clusterRadius*clusterRadius {
				union(i, j)
			}
		}
	}

	// Count cluster sizes and find centroids
	type clusterInfo struct {
		count   int
		sumX    float64
		sumY    float64
		members []int
	}
	clusters := make(map[int]*clusterInfo)
	for i := range ss.Bots {
		root := find(i)
		ci, ok := clusters[root]
		if !ok {
			ci = &clusterInfo{}
			clusters[root] = ci
		}
		ci.count++
		ci.sumX += ss.Bots[i].X
		ci.sumY += ss.Bots[i].Y
		ci.members = append(ci.members, i)
	}

	// Explode clusters > 5 bots (only if avg speed is low)
	for _, ci := range clusters {
		if ci.count <= 5 {
			continue
		}
		// Check average speed — only break slow-moving clusters
		var avgSpeed float64
		for _, idx := range ci.members {
			avgSpeed += ss.Bots[idx].Speed
		}
		avgSpeed /= float64(ci.count)
		if avgSpeed > 0.3 {
			continue // cluster is moving, leave it alone
		}

		cx := ci.sumX / float64(ci.count)
		cy := ci.sumY / float64(ci.count)
		logger.Info("CLUSTER", "Broke cluster of %d bots at (%.0f, %.0f) avgSpeed=%.2f", ci.count, cx, cy, avgSpeed)
		for _, idx := range ci.members {
			bot := &ss.Bots[idx]
			if bot.AntiStuckTimer > 0 || bot.ScatterTimer > 0 {
				continue // already being handled
			}
			// Exempt bots actively delivering (carry + match)
			if bot.CarryingPkg >= 0 && bot.DropoffMatch {
				continue
			}
			// Random outward impulse from centroid
			dx := bot.X - cx
			dy := bot.Y - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1.0 {
				angle := ss.Rng.Float64() * 2 * math.Pi
				bot.Angle = angle
			} else {
				bot.Angle = math.Atan2(dy, dx) + (ss.Rng.Float64()-0.5)*0.8
			}
			bot.Speed = swarm.SwarmBotSpeed * 1.3
			bot.ScatterTimer = 20
			bot.ScatterCooldown = 40
		}
	}
}
