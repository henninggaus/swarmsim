package swarm

import "math"

// SwarmAlgorithmType identifies a classic swarm intelligence algorithm.
type SwarmAlgorithmType int

const (
	AlgoNone     SwarmAlgorithmType = iota
	AlgoBoids                       // Craig Reynolds' Boids (1986)
	AlgoPSO                         // Particle Swarm Optimization
	AlgoACO                         // Ant Colony Optimization
	AlgoFirefly                     // Firefly Algorithm
	AlgoCount
)

// SwarmAlgorithmState holds the state for classic swarm algorithms.
type SwarmAlgorithmState struct {
	ActiveAlgo SwarmAlgorithmType

	// Boids parameters
	BoidsSeparationDist float64 // minimum distance (default 15)
	BoidsAlignmentDist  float64 // alignment range (default 50)
	BoidsCohesionDist   float64 // cohesion range (default 80)
	BoidsSepWeight      float64 // separation weight (default 1.5)
	BoidsAlignWeight    float64 // alignment weight (default 1.0)
	BoidsCohWeight      float64 // cohesion weight (default 1.0)
	BoidsMaxSpeed       float64 // max speed (default 2.0)
	BoidsMaxTurn        float64 // max turn per tick in radians (default 0.2)

	// PSO parameters
	PSOGlobalBestX float64 // global best position
	PSOGlobalBestY float64
	PSOGlobalBestF float64 // global best fitness
	PSOInertia     float64 // inertia weight (default 0.7)
	PSOCognitive   float64 // cognitive coefficient (default 1.5)
	PSOSocial      float64 // social coefficient (default 1.5)
	PSOPersonalBest []PSOParticle // per-bot personal best

	// ACO parameters
	ACOPheromoneDeposit float64 // amount deposited per step (default 1.0)
	ACOEvaporation      float64 // evaporation rate per tick (default 0.01)
	ACOAlpha            float64 // pheromone influence (default 1.0)
	ACOBeta             float64 // distance influence (default 2.0)
	ACOGrid             []float64 // pheromone grid
	ACOGridCols         int
	ACOGridRows         int
	ACOCellSize         float64

	// Firefly parameters
	FireflyBeta0      float64 // base attractiveness (default 1.0)
	FireflyGamma      float64 // light absorption (default 0.01)
	FireflyAlpha      float64 // randomization parameter (default 0.5)
	FireflyBrightness []float64 // per-bot brightness (fitness-based)
}

// PSOParticle stores personal best for one PSO particle.
type PSOParticle struct {
	BestX, BestY float64
	BestFitness  float64
	VelX, VelY   float64
}

// SwarmAlgorithmName returns the display name of an algorithm.
func SwarmAlgorithmName(algo SwarmAlgorithmType) string {
	switch algo {
	case AlgoBoids:
		return "Boids (Reynolds)"
	case AlgoPSO:
		return "Particle Swarm (PSO)"
	case AlgoACO:
		return "Ant Colony (ACO)"
	case AlgoFirefly:
		return "Firefly"
	default:
		return "Keiner"
	}
}

// InitSwarmAlgorithm initializes a classic swarm algorithm.
func InitSwarmAlgorithm(ss *SwarmState, algo SwarmAlgorithmType) {
	sa := &SwarmAlgorithmState{
		ActiveAlgo: algo,
		// Boids defaults
		BoidsSeparationDist: 15,
		BoidsAlignmentDist:  50,
		BoidsCohesionDist:   80,
		BoidsSepWeight:      1.5,
		BoidsAlignWeight:    1.0,
		BoidsCohWeight:      1.0,
		BoidsMaxSpeed:       2.0,
		BoidsMaxTurn:        0.2,
		// PSO defaults
		PSOInertia:   0.7,
		PSOCognitive: 1.5,
		PSOSocial:    1.5,
		// ACO defaults
		ACOPheromoneDeposit: 1.0,
		ACOEvaporation:      0.01,
		ACOAlpha:            1.0,
		ACOBeta:             2.0,
		ACOCellSize:         20,
		// Firefly defaults
		FireflyBeta0: 1.0,
		FireflyGamma: 0.01,
		FireflyAlpha: 0.5,
	}

	n := len(ss.Bots)

	switch algo {
	case AlgoPSO:
		sa.PSOPersonalBest = make([]PSOParticle, n)
		sa.PSOGlobalBestF = -1e9
		for i := range ss.Bots {
			sa.PSOPersonalBest[i] = PSOParticle{
				BestX:       ss.Bots[i].X,
				BestY:       ss.Bots[i].Y,
				BestFitness: -1e9,
			}
		}
	case AlgoACO:
		cols := int(math.Ceil(ss.ArenaW / sa.ACOCellSize))
		rows := int(math.Ceil(ss.ArenaH / sa.ACOCellSize))
		sa.ACOGridCols = cols
		sa.ACOGridRows = rows
		sa.ACOGrid = make([]float64, cols*rows)
	case AlgoFirefly:
		sa.FireflyBrightness = make([]float64, n)
	}

	ss.SwarmAlgo = sa
}

// ClearSwarmAlgorithm disables the swarm algorithm system.
func ClearSwarmAlgorithm(ss *SwarmState) {
	ss.SwarmAlgo = nil
	ss.SwarmAlgoOn = false
}

// TickSwarmAlgorithm runs one tick of the active algorithm.
func TickSwarmAlgorithm(ss *SwarmState) {
	sa := ss.SwarmAlgo
	if sa == nil {
		return
	}

	switch sa.ActiveAlgo {
	case AlgoBoids:
		tickBoids(ss, sa)
	case AlgoPSO:
		tickPSO(ss, sa)
	case AlgoACO:
		tickACO(ss, sa)
	case AlgoFirefly:
		tickFirefly(ss, sa)
	}
}

// ─── BOIDS ─────────────────────────────────────────────

func tickBoids(ss *SwarmState, sa *SwarmAlgorithmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]

		sepX, sepY := 0.0, 0.0 // separation
		aliX, aliY := 0.0, 0.0 // alignment
		cohX, cohY := 0.0, 0.0 // cohesion
		aliCount, cohCount := 0, 0

		for j := range ss.Bots {
			if i == j {
				continue
			}
			dx := ss.Bots[j].X - bot.X
			dy := ss.Bots[j].Y - bot.Y
			d := math.Sqrt(dx*dx + dy*dy)

			// Separation
			if d < sa.BoidsSeparationDist && d > 0 {
				sepX -= dx / d
				sepY -= dy / d
			}
			// Alignment
			if d < sa.BoidsAlignmentDist {
				aliX += math.Cos(ss.Bots[j].Angle)
				aliY += math.Sin(ss.Bots[j].Angle)
				aliCount++
			}
			// Cohesion
			if d < sa.BoidsCohesionDist {
				cohX += dx
				cohY += dy
				cohCount++
			}
		}

		// Average and weight
		desiredAngle := bot.Angle
		fx, fy := 0.0, 0.0

		fx += sepX * sa.BoidsSepWeight
		fy += sepY * sa.BoidsSepWeight

		if aliCount > 0 {
			fx += (aliX / float64(aliCount)) * sa.BoidsAlignWeight
			fy += (aliY / float64(aliCount)) * sa.BoidsAlignWeight
		}
		if cohCount > 0 {
			fx += (cohX / float64(cohCount)) * sa.BoidsCohWeight
			fy += (cohY / float64(cohCount)) * sa.BoidsCohWeight
		}

		if fx != 0 || fy != 0 {
			desiredAngle = math.Atan2(fy, fx)
		}

		// Smooth turning
		angleDiff := desiredAngle - bot.Angle
		for angleDiff > math.Pi {
			angleDiff -= 2 * math.Pi
		}
		for angleDiff < -math.Pi {
			angleDiff += 2 * math.Pi
		}
		if angleDiff > sa.BoidsMaxTurn {
			angleDiff = sa.BoidsMaxTurn
		}
		if angleDiff < -sa.BoidsMaxTurn {
			angleDiff = -sa.BoidsMaxTurn
		}
		bot.Angle += angleDiff
		bot.Speed = sa.BoidsMaxSpeed

		// LED: color by heading for visual effect
		hue := (bot.Angle + math.Pi) / (2 * math.Pi)
		r, g, b := hsvToRGB(hue, 0.8, 1.0)
		bot.LEDColor = [3]uint8{r, g, b}
	}
}

// ─── PSO ───────────────────────────────────────────────

func tickPSO(ss *SwarmState, sa *SwarmAlgorithmState) {
	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Evaluate fitness at current position (distance to light or center)
		fitness := psoFitness(bot, ss)

		// Update personal best
		if i < len(sa.PSOPersonalBest) {
			pb := &sa.PSOPersonalBest[i]
			if fitness > pb.BestFitness {
				pb.BestFitness = fitness
				pb.BestX = bot.X
				pb.BestY = bot.Y
			}

			// Update global best
			if fitness > sa.PSOGlobalBestF {
				sa.PSOGlobalBestF = fitness
				sa.PSOGlobalBestX = bot.X
				sa.PSOGlobalBestY = bot.Y
			}

			// PSO velocity update
			r1, r2 := ss.Rng.Float64(), ss.Rng.Float64()
			pb.VelX = sa.PSOInertia*pb.VelX +
				sa.PSOCognitive*r1*(pb.BestX-bot.X) +
				sa.PSOSocial*r2*(sa.PSOGlobalBestX-bot.X)
			pb.VelY = sa.PSOInertia*pb.VelY +
				sa.PSOCognitive*r1*(pb.BestY-bot.Y) +
				sa.PSOSocial*r2*(sa.PSOGlobalBestY-bot.Y)

			// Clamp velocity
			maxV := 3.0
			vMag := math.Sqrt(pb.VelX*pb.VelX + pb.VelY*pb.VelY)
			if vMag > maxV {
				pb.VelX = pb.VelX / vMag * maxV
				pb.VelY = pb.VelY / vMag * maxV
			}

			// Apply velocity as heading + speed
			if pb.VelX != 0 || pb.VelY != 0 {
				bot.Angle = math.Atan2(pb.VelY, pb.VelX)
				bot.Speed = math.Min(vMag, SwarmBotSpeed*1.5)
			}
		}

		// LED: brightness by fitness
		fit01 := math.Min(fitness/100.0, 1.0)
		if fit01 < 0 {
			fit01 = 0
		}
		c := uint8(50 + fit01*205)
		bot.LEDColor = [3]uint8{c, c / 2, 0}
	}
}

func psoFitness(bot *SwarmBot, ss *SwarmState) float64 {
	// Use light source if active, otherwise use center
	targetX, targetY := ss.ArenaW/2, ss.ArenaH/2
	if ss.Light.Active {
		targetX = ss.Light.X
		targetY = ss.Light.Y
	}
	dx := bot.X - targetX
	dy := bot.Y - targetY
	dist := math.Sqrt(dx*dx + dy*dy)
	return 100 - dist*0.2 // higher when closer
}

// ─── ACO ───────────────────────────────────────────────

func tickACO(ss *SwarmState, sa *SwarmAlgorithmState) {
	if sa.ACOGrid == nil {
		return
	}

	// Evaporate pheromones
	for i := range sa.ACOGrid {
		sa.ACOGrid[i] *= (1 - sa.ACOEvaporation)
		if sa.ACOGrid[i] < 0.001 {
			sa.ACOGrid[i] = 0
		}
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]

		// Deposit pheromone at current cell
		col := int(bot.X / sa.ACOCellSize)
		row := int(bot.Y / sa.ACOCellSize)
		if col >= 0 && col < sa.ACOGridCols && row >= 0 && row < sa.ACOGridRows {
			// Deposit more if carrying (found food)
			deposit := sa.ACOPheromoneDeposit
			if bot.CarryingPkg >= 0 {
				deposit *= 3.0
			}
			sa.ACOGrid[row*sa.ACOGridCols+col] += deposit
		}

		// Choose direction based on pheromone gradient
		bestPher := -1.0
		bestAngle := bot.Angle
		for angle := 0.0; angle < 2*math.Pi; angle += math.Pi / 4 {
			cx := bot.X + math.Cos(angle)*sa.ACOCellSize
			cy := bot.Y + math.Sin(angle)*sa.ACOCellSize
			tc := int(cx / sa.ACOCellSize)
			tr := int(cy / sa.ACOCellSize)
			if tc >= 0 && tc < sa.ACOGridCols && tr >= 0 && tr < sa.ACOGridRows {
				pher := sa.ACOGrid[tr*sa.ACOGridCols+tc]
				// Add randomness
				pher += ss.Rng.Float64() * sa.ACOAlpha * 0.5
				if pher > bestPher {
					bestPher = pher
					bestAngle = angle
				}
			}
		}

		// Sometimes explore randomly
		if ss.Rng.Float64() < 0.1 {
			bestAngle = ss.Rng.Float64() * 2 * math.Pi
		}

		bot.Angle = bestAngle
		bot.Speed = SwarmBotSpeed

		// LED: pheromone intensity at current pos
		pher := 0.0
		if col >= 0 && col < sa.ACOGridCols && row >= 0 && row < sa.ACOGridRows {
			pher = sa.ACOGrid[row*sa.ACOGridCols+col]
		}
		g := uint8(math.Min(pher*50, 255))
		bot.LEDColor = [3]uint8{0, g, 50}
	}
}

// ─── FIREFLY ───────────────────────────────────────────

func tickFirefly(ss *SwarmState, sa *SwarmAlgorithmState) {
	n := len(ss.Bots)
	if len(sa.FireflyBrightness) != n {
		sa.FireflyBrightness = make([]float64, n)
	}

	// Compute brightness (fitness) for each firefly
	for i := range ss.Bots {
		sa.FireflyBrightness[i] = psoFitness(&ss.Bots[i], ss) // reuse PSO fitness
	}

	for i := range ss.Bots {
		bot := &ss.Bots[i]
		moveX, moveY := 0.0, 0.0

		for j := range ss.Bots {
			if i == j {
				continue
			}
			// Only move toward brighter fireflies
			if sa.FireflyBrightness[j] <= sa.FireflyBrightness[i] {
				continue
			}

			dx := ss.Bots[j].X - bot.X
			dy := ss.Bots[j].Y - bot.Y
			r2 := dx*dx + dy*dy
			r := math.Sqrt(r2)

			// Attractiveness decreases with distance
			beta := sa.FireflyBeta0 * math.Exp(-sa.FireflyGamma*r2)

			if r > 0 {
				moveX += beta * dx / r
				moveY += beta * dy / r
			}
		}

		// Add random walk
		moveX += sa.FireflyAlpha * (ss.Rng.Float64() - 0.5) * 2
		moveY += sa.FireflyAlpha * (ss.Rng.Float64() - 0.5) * 2

		if moveX != 0 || moveY != 0 {
			bot.Angle = math.Atan2(moveY, moveX)
			bot.Speed = math.Min(math.Sqrt(moveX*moveX+moveY*moveY), SwarmBotSpeed*1.5)
		}

		// LED: brightness as yellow intensity
		b01 := (sa.FireflyBrightness[i] + 50) / 150
		if b01 < 0 {
			b01 = 0
		}
		if b01 > 1 {
			b01 = 1
		}
		c := uint8(b01 * 255)
		bot.LEDColor = [3]uint8{c, c, 0}
	}
}

// ─── HELPERS ───────────────────────────────────────────

// hsvToRGB converts HSV (h: 0-1, s: 0-1, v: 0-1) to RGB.
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	h = h - math.Floor(h) // wrap to 0-1
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)

	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	}
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}

// AlgorithmNames returns all available algorithm names.
func AlgorithmNames() []string {
	return []string{"Keiner", "Boids", "PSO", "ACO", "Firefly"}
}
