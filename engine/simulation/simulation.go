package simulation

import (
	"math/rand"
	"swarmsim/domain/bot"
	"swarmsim/domain/comm"
	"swarmsim/domain/genetics"
	"swarmsim/domain/physics"
	"swarmsim/domain/resource"
	"swarmsim/domain/swarm"
	"swarmsim/engine/pheromone"
)

// NewSimulation creates and initializes a new simulation.
func NewSimulation(cfg Config) *Simulation {
	rng := rand.New(rand.NewSource(42))
	arena := physics.NewArena(cfg.ArenaWidth, cfg.ArenaHeight, cfg.HomeBaseX, cfg.HomeBaseY, cfg.HomeBaseR)
	arena.GenerateObstacles(cfg.InitObstacles, rng)

	s := &Simulation{
		Cfg:             cfg,
		Arena:           arena,
		Channel:         comm.NewChannel(),
		Hash:            physics.NewSpatialHash(cfg.ArenaWidth, cfg.ArenaHeight, cfg.SpatialCellSize),
		Pheromones:      pheromone.NewPheromoneGrid(cfg.ArenaWidth, cfg.ArenaHeight, cfg.PherCellSize, cfg.PherDecay, cfg.PherDiffusion),
		Speed:           1.0,
		Rng:             rng,
		SelectedBotID:   -1,
		CurrentScenario: ScenarioSandbox,
		GenomePool:      make(map[bot.BotType][]bot.Genome),
	}

	s.spawnInitialBots(cfg.InitScouts, bot.TypeScout)
	s.spawnInitialBots(cfg.InitWorkers, bot.TypeWorker)
	s.spawnInitialBots(cfg.InitLeaders, bot.TypeLeader)
	s.spawnInitialBots(cfg.InitTanks, bot.TypeTank)
	s.spawnInitialBots(cfg.InitHealers, bot.TypeHealer)

	for i := 0; i < cfg.InitResources; i++ {
		s.SpawnResourceRandom()
	}

	if cfg.WaveEnabled {
		s.WaveTicksLeft = cfg.WaveInterval
	}

	return s
}

func (s *Simulation) spawnInitialBots(count int, typ bot.BotType) {
	for i := 0; i < count; i++ {
		x, y := s.randomEdgePosition()
		s.SpawnBot(typ, x, y)
	}
}

func (s *Simulation) randomEdgePosition() (float64, float64) {
	margin := 80.0
	side := s.Rng.Intn(4)
	switch side {
	case 0:
		return margin + s.Rng.Float64()*(s.Cfg.ArenaWidth-2*margin), margin
	case 1:
		return margin + s.Rng.Float64()*(s.Cfg.ArenaWidth-2*margin), s.Cfg.ArenaHeight - margin
	case 2:
		return margin, margin + s.Rng.Float64()*(s.Cfg.ArenaHeight-2*margin)
	default:
		return s.Cfg.ArenaWidth - margin, margin + s.Rng.Float64()*(s.Cfg.ArenaHeight-2*margin)
	}
}

// SpawnBot creates a new bot with a genome from pool or random.
func (s *Simulation) SpawnBot(typ bot.BotType, x, y float64) bot.Bot {
	id := s.NextBotID
	s.NextBotID++
	var b bot.Bot
	switch typ {
	case bot.TypeScout:
		b = bot.NewScout(id, x, y)
	case bot.TypeWorker:
		b = bot.NewWorker(id, x, y)
	case bot.TypeLeader:
		b = bot.NewLeader(id, x, y)
	case bot.TypeTank:
		b = bot.NewTank(id, x, y)
	case bot.TypeHealer:
		b = bot.NewHealer(id, x, y)
	}

	base := b.GetBase()
	pool := s.GenomePool[typ]
	if len(pool) > 0 {
		base.Genome = pool[0]
		s.GenomePool[typ] = pool[1:]
	} else {
		base.Genome = genetics.NewRandomGenome(s.Rng)
	}
	base.ApplyGenomeSpeed()

	s.Bots = append(s.Bots, b)
	return b
}

func (s *Simulation) SpawnResourceAt(x, y float64) {
	id := s.NextResID
	s.NextResID++
	s.Resources = append(s.Resources, resource.NewResource(id, x, y, s.Cfg.ResourceValue))
}

func (s *Simulation) SpawnResourceRandom() {
	margin := 50.0
	x := margin + s.Rng.Float64()*(s.Cfg.ArenaWidth-2*margin)
	y := margin + s.Rng.Float64()*(s.Cfg.ArenaHeight-2*margin)
	s.SpawnResourceAt(x, y)
}

// SpawnHeavyResourceRandom creates a heavy resource at a random position.
func (s *Simulation) SpawnHeavyResourceRandom() {
	margin := 50.0
	x := margin + s.Rng.Float64()*(s.Cfg.ArenaWidth-2*margin)
	y := margin + s.Rng.Float64()*(s.Cfg.ArenaHeight-2*margin)
	id := s.NextResID
	s.NextResID++
	s.Resources = append(s.Resources, resource.NewHeavyResource(id, x, y, 10.0))
}

func (s *Simulation) AddObstacleAt(x, y float64) {
	w := 30 + s.Rng.Float64()*50
	h := 30 + s.Rng.Float64()*50
	s.Arena.AddObstacle(x-w/2, y-h/2, w, h)
}

// spawnWave spawns a new wave of resources (mix of normal and heavy).
func (s *Simulation) spawnWave() {
	s.WaveNumber++
	totalCount := s.Cfg.WaveMinResources + s.Rng.Intn(s.Cfg.WaveMaxResources-s.Cfg.WaveMinResources+1)
	heavyCount := s.Cfg.WaveMinHeavy + s.Rng.Intn(s.Cfg.WaveMaxHeavy-s.Cfg.WaveMinHeavy+1)
	if heavyCount > totalCount {
		heavyCount = totalCount
	}
	normalCount := totalCount - heavyCount

	for i := 0; i < normalCount; i++ {
		s.SpawnResourceRandom()
	}
	for i := 0; i < heavyCount; i++ {
		s.SpawnHeavyResourceRandom()
	}
	s.WaveTicksLeft = s.Cfg.WaveInterval
}

// resolveCooperativePickups checks heavy resources for 3+ workers nearby.
func (s *Simulation) resolveCooperativePickups() {
	for _, r := range s.Resources {
		if !r.IsAvailable() || !r.Heavy {
			continue
		}

		rPos := bot.Vec2{X: r.X, Y: r.Y}
		var nearbyWorkers []*bot.BaseBot
		for _, b := range s.Bots {
			if !b.IsAlive() || b.Type() != bot.TypeWorker {
				continue
			}
			if b.Position().Dist(rPos) <= 15.0 && b.GetBase().CanCarry() {
				nearbyWorkers = append(nearbyWorkers, b.GetBase())
			}
		}

		if len(nearbyWorkers) >= 3 {
			// Cooperative pickup succeeds
			carrier := nearbyWorkers[0]
			r.PickUp(carrier.BotID)
			carrier.Inventory = append(carrier.Inventory, r)
			carrier.FitResourcesCollected++
			carrier.State = bot.StateReturning

			// Credit other participants
			for i := 1; i < 3 && i < len(nearbyWorkers); i++ {
				nearbyWorkers[i].FitResourcesCollected++
				nearbyWorkers[i].State = bot.StateForaging
			}

			// Emit event for particle effect
			s.CoopPickupEvents = append(s.CoopPickupEvents, CoopPickupEvent{
				X: r.X, Y: r.Y, Tick: s.Tick,
			})
		}
	}
}

// Update advances the simulation by one tick.
func (s *Simulation) Update() {
	if s.Paused {
		return
	}
	s.Tick++
	s.GenerationTick++

	// Clear per-tick events
	s.CoopPickupEvents = s.CoopPickupEvents[:0]
	s.DeliveryEvents = s.DeliveryEvents[:0]

	// Swarm mode uses its own update loop
	if s.SwarmMode && s.SwarmState != nil {
		s.updateSwarmMode()
		return
	}

	// Rebuild spatial hash
	s.Hash.Clear()
	for _, b := range s.Bots {
		if b.IsAlive() {
			pos := b.Position()
			s.Hash.Insert(b.ID(), pos.X, pos.Y)
		}
	}

	eCfg := bot.EnergyCfg{
		MoveCost:  s.Cfg.EnergyMoveCost,
		MsgCost:   s.Cfg.EnergyMsgCost,
		CarryCost: s.Cfg.EnergyCarryCost,
		PherCost:  s.Cfg.EnergyPherCost,
		TankPush:  s.Cfg.EnergyTankPush,
		DecayMult: s.Cfg.EnergyDecayMult,
	}

	for _, b := range s.Bots {
		if !b.IsAlive() {
			continue
		}
		pos := b.Position()

		maxRange := b.GetSensorRange()
		if b.GetCommRange() > maxRange {
			maxRange = b.GetCommRange()
		}
		nearbyIDs := s.Hash.Query(pos.X, pos.Y, maxRange)
		var nearby []bot.Bot
		for _, nid := range nearbyIDs {
			if nid != b.ID() && nid < len(s.Bots) && s.Bots[nid].IsAlive() {
				if pos.Dist(s.Bots[nid].Position()) <= b.GetSensorRange() {
					nearby = append(nearby, s.Bots[nid])
				}
			}
		}

		var nearbyRes []*resource.Resource
		for _, r := range s.Resources {
			if r.IsAvailable() {
				if pos.Dist(bot.Vec2{X: r.X, Y: r.Y}) <= b.GetSensorRange() {
					nearbyRes = append(nearbyRes, r)
				}
			}
		}

		inbox := s.Channel.Deliver(pos.X, pos.Y)

		ctx := &bot.UpdateContext{
			Nearby:     nearby,
			Resources:  nearbyRes,
			Inbox:      inbox,
			HomeX:      s.Cfg.HomeBaseX,
			HomeY:      s.Cfg.HomeBaseY,
			Pheromones: s.Pheromones,
			ECfg:       eCfg,
			Tick:       s.Tick,
		}

		outbox := b.Update(ctx)

		for _, msg := range outbox {
			s.Channel.Send(msg, pos.X, pos.Y, b.GetCommRange())
			s.TotalMsgsSent++
		}

		// Energy recharge near home base
		base := b.GetBase()
		homeDist := pos.Dist(bot.Vec2{X: s.Cfg.HomeBaseX, Y: s.Cfg.HomeBaseY})
		if homeDist < s.Cfg.EnergyBaseRange {
			base.RechargeEnergy(s.Cfg.EnergyBaseRecharge)
		}

		s.resolvePhysics(b)
	}

	// Cooperative pickup resolution (after all bots updated)
	s.resolveCooperativePickups()

	s.Pheromones.Update()
	s.ActiveMsgs = s.Channel.Tick()

	// Scoring and delivery count
	delivered := 0
	score := 0
	for _, r := range s.Resources {
		if r.IsDelivered() {
			delivered++
			score += r.PointValue
		}
	}
	s.Delivered = delivered
	s.Score = score

	// Wave system
	if s.Cfg.WaveEnabled {
		s.WaveTicksLeft--
		if s.WaveTicksLeft <= 0 {
			s.spawnWave()
		}
	} else if s.Cfg.ResourceRespawn && s.Cfg.RespawnInterval > 0 && s.Tick%s.Cfg.RespawnInterval == 0 {
		// Legacy respawn for backward compatibility
		available := 0
		for _, r := range s.Resources {
			if r.IsAvailable() {
				available++
			}
		}
		if available < 10 {
			s.SpawnResourceRandom()
		}
	}

	// Auto-evolve check
	if s.Cfg.AutoEvolve && s.GenerationTick >= s.Cfg.GenerationLength {
		s.EndGeneration()
	}
}

// EndGeneration triggers evolution.
func (s *Simulation) EndGeneration() {
	best, avg := EvolveGeneration(s.Bots, s.Rng, s.Cfg.MutationRate, s.Cfg.MutationSigma, s.Cfg.EliteRatio)
	s.BestFitness = best
	s.AvgFitness = avg
	s.FitnessHistory = append(s.FitnessHistory, avg)
	s.Generation++
	s.GenerationTick = 0
}

// ForceEndGeneration ends the current generation immediately.
func (s *Simulation) ForceEndGeneration() {
	s.EndGeneration()
}

// LoadScenario switches to a new scenario, preserving genomes.
func (s *Simulation) LoadScenario(sc Scenario) {
	s.GenomePool = CollectGenomes(s.Bots)

	cfg := sc.Cfg
	s.Cfg = cfg
	s.Arena = physics.NewArena(cfg.ArenaWidth, cfg.ArenaHeight, cfg.HomeBaseX, cfg.HomeBaseY, cfg.HomeBaseR)
	s.Arena.GenerateObstacles(cfg.InitObstacles, s.Rng)
	s.Bots = nil
	s.Resources = nil
	s.Channel = comm.NewChannel()
	s.Hash = physics.NewSpatialHash(cfg.ArenaWidth, cfg.ArenaHeight, cfg.SpatialCellSize)
	s.Pheromones = pheromone.NewPheromoneGrid(cfg.ArenaWidth, cfg.ArenaHeight, cfg.PherCellSize, cfg.PherDecay, cfg.PherDiffusion)
	s.NextBotID = 0
	s.NextResID = 0
	s.Tick = 0
	s.GenerationTick = 0
	s.Delivered = 0
	s.ActiveMsgs = 0
	s.Score = 0
	s.WaveNumber = 0
	s.CoopPickupEvents = nil
	s.DeliveryEvents = nil
	s.SwarmMode = false
	s.SwarmState = nil
	s.SelectedBotID = -1
	s.CurrentScenario = sc.ID
	s.ScenarioTitle = sc.Name
	s.ScenarioTimer = 120

	if cfg.WaveEnabled {
		s.WaveTicksLeft = cfg.WaveInterval
	}

	s.spawnInitialBots(cfg.InitScouts, bot.TypeScout)
	s.spawnInitialBots(cfg.InitWorkers, bot.TypeWorker)
	s.spawnInitialBots(cfg.InitLeaders, bot.TypeLeader)
	s.spawnInitialBots(cfg.InitTanks, bot.TypeTank)
	s.spawnInitialBots(cfg.InitHealers, bot.TypeHealer)

	for i := 0; i < cfg.InitResources; i++ {
		s.SpawnResourceRandom()
	}

	if sc.CustomSetup != nil {
		sc.CustomSetup(s)
	}
}

func (s *Simulation) resolvePhysics(b bot.Bot) {
	base := b.GetBase()
	pos := base.Pos
	r := base.Radius

	nx, ny, hit := physics.ClampToBounds(pos.X, pos.Y, r, s.Cfg.ArenaWidth, s.Cfg.ArenaHeight)
	if hit {
		base.Vel.X, base.Vel.Y = physics.ReflectVelocity(pos.X, pos.Y, base.Vel.X, base.Vel.Y, r, s.Cfg.ArenaWidth, s.Cfg.ArenaHeight)
		base.Pos.X = nx
		base.Pos.Y = ny
	}

	for _, obs := range s.Arena.Obstacles {
		collides, _, _ := physics.CircleRectCollision(base.Pos.X, base.Pos.Y, r, obs.X, obs.Y, obs.W, obs.H)
		if collides {
			if b.Type() == bot.TypeTank && obs.Pushable {
				dx, dy := physics.Normalize(base.Vel.X, base.Vel.Y)
				obs.X += dx * 0.5
				obs.Y += dy * 0.5
			}
			base.Pos.X, base.Pos.Y = physics.ResolveCircleRectOverlap(base.Pos.X, base.Pos.Y, r, obs.X, obs.Y, obs.W, obs.H)
			base.Vel = base.Vel.Scale(0.3)
		}
	}

	if len(base.Inventory) > 0 && s.Arena.InHomeBase(base.Pos.X, base.Pos.Y) {
		// Emit delivery events before delivering
		for _, res := range base.Inventory {
			s.DeliveryEvents = append(s.DeliveryEvents, DeliveryEvent{
				Tick:       s.Tick,
				PointValue: res.PointValue,
			})
		}
		base.DeliverResources()
	}
}

func (s *Simulation) BotCount() map[bot.BotType]int {
	counts := make(map[bot.BotType]int)
	for _, b := range s.Bots {
		if b.IsAlive() {
			counts[b.Type()]++
		}
	}
	return counts
}

func (s *Simulation) GetBotByID(id int) bot.Bot {
	if id >= 0 && id < len(s.Bots) {
		return s.Bots[id]
	}
	return nil
}

func (s *Simulation) FindBotAt(x, y, maxDist float64) bot.Bot {
	var nearest bot.Bot
	minD := maxDist
	p := bot.Vec2{X: x, Y: y}
	for _, b := range s.Bots {
		if !b.IsAlive() {
			continue
		}
		d := p.Dist(b.Position())
		if d < minD {
			minD = d
			nearest = b
		}
	}
	return nearest
}

// LoadSwarmScenario sets up the programmable swarm scenario.
func (s *Simulation) LoadSwarmScenario() {
	cfg := DefaultConfig()
	cfg.ArenaWidth = swarm.SwarmArenaSize
	cfg.ArenaHeight = swarm.SwarmArenaSize
	cfg.InitObstacles = 0
	cfg.InitResources = 0
	cfg.InitScouts = 0
	cfg.InitWorkers = 0
	cfg.InitLeaders = 0
	cfg.InitTanks = 0
	cfg.InitHealers = 0
	cfg.ResourceRespawn = false
	cfg.WaveEnabled = false
	cfg.HomeBaseX = -100
	cfg.HomeBaseY = -100
	cfg.HomeBaseR = 0

	sc := Scenario{
		ID:   ScenarioSwarm,
		Name: "PROGRAMMABLE SWARM",
		Cfg:  cfg,
	}
	s.LoadScenario(sc)

	s.SwarmMode = true
	s.SwarmState = swarm.NewSwarmState(s.Rng, swarm.SwarmDefaultBots)
}

