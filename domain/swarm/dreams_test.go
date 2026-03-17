package swarm

import (
	"math/rand"
	"testing"
)

func TestInitDreams(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 15)
	InitDreams(ss)

	ds := ss.Dreams
	if ds == nil {
		t.Fatal("dreams should be initialized")
	}
	if len(ds.Experiences) != 15 {
		t.Fatalf("expected 15 experience buffers, got %d", len(ds.Experiences))
	}
	if ds.MaxBuffer != 50 {
		t.Fatal("default max buffer should be 50")
	}
}

func TestClearDreams(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.DreamsOn = true
	InitDreams(ss)
	ClearDreams(ss)

	if ss.Dreams != nil {
		t.Fatal("should be nil")
	}
	if ss.DreamsOn {
		t.Fatal("should be false")
	}
}

func TestTickDreams(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitDreams(ss)

	// Set some bots carrying packages near dropoffs
	for i := 0; i < 5; i++ {
		ss.Bots[i].CarryingPkg = 0
		ss.Bots[i].NearestDropoffDist = 50
	}
	// Some slow bots to trigger dreaming
	for i := 5; i < 10; i++ {
		ss.Bots[i].Speed = SwarmBotSpeed * 0.2
	}

	for tick := 0; tick < 200; tick++ {
		ss.Tick = tick
		TickDreams(ss)
	}

	ds := ss.Dreams
	if ds.TotalExperiences == 0 {
		t.Fatal("should have recorded some experiences")
	}
}

func TestTickDreamsNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	TickDreams(ss) // should not panic
}

func TestDreamPhase(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitDreams(ss)

	// Fill experience buffers
	for i := range ss.Dreams.Experiences {
		for j := 0; j < 10; j++ {
			ss.Dreams.Experiences[i] = append(ss.Dreams.Experiences[i], DreamExperience{
				Tick:   j * 10,
				X:      float64(j * 20),
				Y:      float64(j * 20),
				Reward: 0.5,
			})
		}
	}

	// Tick at global dream phase (tick 490-499)
	ss.Tick = 495
	TickDreams(ss)

	if !ss.Dreams.DreamPhase {
		t.Fatal("should be in dream phase at tick 495")
	}

	// Tick outside dream phase
	ss.Tick = 100
	TickDreams(ss)

	if ss.Dreams.DreamPhase {
		t.Fatal("should not be in dream phase at tick 100")
	}
}

func TestDreamCount(t *testing.T) {
	if DreamCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	ds := &DreamState{TotalDreams: 42}
	if DreamCount(ds) != 42 {
		t.Fatal("expected 42")
	}
}

func TestExperienceCount(t *testing.T) {
	if ExperienceCount(nil) != 0 {
		t.Fatal("nil should return 0")
	}
	ds := &DreamState{TotalExperiences: 100}
	if ExperienceCount(ds) != 100 {
		t.Fatal("expected 100")
	}
}

func TestBotExperienceCount(t *testing.T) {
	if BotExperienceCount(nil, 0) != 0 {
		t.Fatal("nil should return 0")
	}

	ds := &DreamState{
		Experiences: [][]DreamExperience{
			{{}, {}, {}},
			{{}},
		},
	}
	if BotExperienceCount(ds, 0) != 3 {
		t.Fatal("expected 3")
	}
	if BotExperienceCount(ds, 5) != 0 {
		t.Fatal("out of bounds should return 0")
	}
}

func TestIsDreamPhase(t *testing.T) {
	if IsDreamPhase(nil) {
		t.Fatal("nil should return false")
	}
	ds := &DreamState{DreamPhase: true}
	if !IsDreamPhase(ds) {
		t.Fatal("should be true")
	}
}

func TestReplayExperiences(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 1)
	InitDreams(ss)

	bot := &ss.Bots[0]
	bot.X = 100
	bot.Y = 100
	originalAngle := bot.Angle

	exps := []DreamExperience{
		{X: 200, Y: 200, Reward: 0.8},
		{X: 50, Y: 50, Reward: -0.5},
		{X: 300, Y: 100, Reward: 0.6},
		{X: 100, Y: 300, Reward: 0.4},
		{X: 150, Y: 150, Reward: 0.9},
	}

	ds := ss.Dreams
	val := replayExperiences(ss, bot, exps, ds)

	// Should have modified angle
	if bot.Angle == originalAngle {
		t.Fatal("replay should have changed bot angle")
	}
	// Average value should be nonzero
	if val == 0 {
		t.Fatal("replay value should be nonzero")
	}
}
