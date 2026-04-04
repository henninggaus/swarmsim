// Package swarm — Learning path system for guided swarm robotics education.
// Provides 12 lessons across 3 difficulty levels, each with step-by-step
// explanations and optional challenges with star ratings.
package swarm

// LessonID identifies a lesson in the learning path.
type LessonID int

const (
	// Beginner: Local Rules -> Global Patterns
	LessonAggregation LessonID = iota // "Why do bots form clusters?"
	LessonDispersion                   // "How do bots spread evenly?"
	LessonFlocking                     // "The 3 rules of bird flocking"
	LessonEmergence                    // "What is emergent behavior?"

	// Intermediate: Communication & Delivery
	LessonDelivery      // "Package logistics with simple rules"
	LessonCommunication // "How bots share information"
	LessonEvolution     // "Natural selection optimizes parameters"
	LessonObstacles     // "Navigating mazes with local sensors"

	// Advanced: Algorithms & Optimization
	LessonAlgoIntro     // "What is an optimization algorithm?"
	LessonGWO           // "Wolves hunting in packs"
	LessonSwarmVsClassic // "Swarm intelligence vs central control"
	LessonFactory       // "From theory to practice: Factory Mode"

	LessonCount
)

// Lesson defines a complete guided lesson with steps and an optional challenge.
type Lesson struct {
	ID         LessonID
	TitleKey   string // locale key for title
	DescKey    string // locale key for description
	Level      int    // 0=beginner, 1=intermediate, 2=advanced
	PresetName string // which preset to auto-load (e.g. "Aggregation")
	Steps      []LessonStep
	Challenge  *Challenge // optional challenge after lesson
}

// LessonStep defines a single step within a lesson.
type LessonStep struct {
	TextKey   string // locale key for explanation text (3 lines max)
	WaitTicks int    // auto-advance after N ticks (0 = wait for click)
	Highlight string // what to highlight: "bots", "clusters", "fitness", ""
	SetupFunc string // optional: "enable_delivery", "enable_evolution", etc.
}

// Challenge defines a skill test at the end of a lesson.
type Challenge struct {
	DescKey         string  // locale key for challenge description
	MetricFunc      string  // "delivery_accuracy", "cluster_count", "fitness_over_200"
	ThresholdGold   float64
	ThresholdSilver float64
	ThresholdBronze float64
}

// LearningState tracks the user's progress through the learning path.
type LearningState struct {
	CurrentLesson LessonID
	CurrentStep   int
	StepTimer     int
	Active        bool
	Completed     [LessonCount]int // 0=not done, 1=bronze, 2=silver, 3=gold
	TotalStars    int

	// Challenge tracking
	ChallengeActive bool
	ChallengeTicks  int
	ChallengeValue  float64

	// Menu state
	ShowMenu bool
}

// GetAllLessons returns all 12 lesson definitions.
func GetAllLessons() []Lesson {
	return []Lesson{
		// ===== BEGINNER (Level 0) =====
		{
			ID: LessonAggregation, TitleKey: "lesson.agg.title", DescKey: "lesson.agg.desc",
			Level: 0, PresetName: "Aggregation",
			Steps: []LessonStep{
				{TextKey: "lesson.agg.1", WaitTicks: 0},
				{TextKey: "lesson.agg.2", WaitTicks: 0},
				{TextKey: "lesson.agg.3", WaitTicks: 300, Highlight: "bots"},
				{TextKey: "lesson.agg.4", WaitTicks: 0, Highlight: "clusters"},
				{TextKey: "lesson.agg.5", WaitTicks: 0},
				{TextKey: "lesson.agg.6", WaitTicks: 0},
			},
			Challenge: &Challenge{
				DescKey: "lesson.agg.challenge", MetricFunc: "cluster_count",
				ThresholdGold: 5, ThresholdSilver: 3, ThresholdBronze: 1,
			},
		},
		{
			ID: LessonDispersion, TitleKey: "lesson.disp.title", DescKey: "lesson.disp.desc",
			Level: 0, PresetName: "Dispersion",
			Steps: []LessonStep{
				{TextKey: "lesson.disp.1", WaitTicks: 0},
				{TextKey: "lesson.disp.2", WaitTicks: 0},
				{TextKey: "lesson.disp.3", WaitTicks: 300, Highlight: "bots"},
				{TextKey: "lesson.disp.4", WaitTicks: 0},
				{TextKey: "lesson.disp.5", WaitTicks: 0},
			},
			Challenge: &Challenge{
				DescKey: "lesson.disp.challenge", MetricFunc: "coverage",
				ThresholdGold: 80, ThresholdSilver: 60, ThresholdBronze: 40,
			},
		},
		{
			ID: LessonFlocking, TitleKey: "lesson.flock.title", DescKey: "lesson.flock.desc",
			Level: 0, PresetName: "Flocking",
			Steps: []LessonStep{
				{TextKey: "lesson.flock.1", WaitTicks: 0},
				{TextKey: "lesson.flock.2", WaitTicks: 0},
				{TextKey: "lesson.flock.3", WaitTicks: 300, Highlight: "bots"},
				{TextKey: "lesson.flock.4", WaitTicks: 0},
				{TextKey: "lesson.flock.5", WaitTicks: 0},
				{TextKey: "lesson.flock.6", WaitTicks: 0},
			},
			Challenge: &Challenge{
				DescKey: "lesson.flock.challenge", MetricFunc: "alignment",
				ThresholdGold: 0.9, ThresholdSilver: 0.7, ThresholdBronze: 0.5,
			},
		},
		{
			ID: LessonEmergence, TitleKey: "lesson.emerge.title", DescKey: "lesson.emerge.desc",
			Level: 0, PresetName: "Aggregation",
			Steps: []LessonStep{
				{TextKey: "lesson.emerge.1", WaitTicks: 0},
				{TextKey: "lesson.emerge.2", WaitTicks: 0},
				{TextKey: "lesson.emerge.3", WaitTicks: 300, Highlight: "bots"},
				{TextKey: "lesson.emerge.4", WaitTicks: 0, Highlight: "clusters"},
				{TextKey: "lesson.emerge.5", WaitTicks: 0},
				{TextKey: "lesson.emerge.6", WaitTicks: 0},
				{TextKey: "lesson.emerge.7", WaitTicks: 0},
			},
			Challenge: nil, // observation-only lesson
		},

		// ===== INTERMEDIATE (Level 1) =====
		{
			ID: LessonDelivery, TitleKey: "lesson.deliv.title", DescKey: "lesson.deliv.desc",
			Level: 1, PresetName: "Color Sort",
			Steps: []LessonStep{
				{TextKey: "lesson.deliv.1", WaitTicks: 0},
				{TextKey: "lesson.deliv.2", WaitTicks: 0, SetupFunc: "enable_delivery"},
				{TextKey: "lesson.deliv.3", WaitTicks: 300, Highlight: "bots"},
				{TextKey: "lesson.deliv.4", WaitTicks: 0},
				{TextKey: "lesson.deliv.5", WaitTicks: 0},
				{TextKey: "lesson.deliv.6", WaitTicks: 0},
			},
			Challenge: &Challenge{
				DescKey: "lesson.deliv.challenge", MetricFunc: "delivery_accuracy",
				ThresholdGold: 80, ThresholdSilver: 50, ThresholdBronze: 20,
			},
		},
		{
			ID: LessonCommunication, TitleKey: "lesson.comm.title", DescKey: "lesson.comm.desc",
			Level: 1, PresetName: "LED Gradient",
			Steps: []LessonStep{
				{TextKey: "lesson.comm.1", WaitTicks: 0},
				{TextKey: "lesson.comm.2", WaitTicks: 0},
				{TextKey: "lesson.comm.3", WaitTicks: 300, Highlight: "bots"},
				{TextKey: "lesson.comm.4", WaitTicks: 0},
				{TextKey: "lesson.comm.5", WaitTicks: 0},
				{TextKey: "lesson.comm.6", WaitTicks: 0},
				{TextKey: "lesson.comm.7", WaitTicks: 0},
			},
			Challenge: nil,
		},
		{
			ID: LessonEvolution, TitleKey: "lesson.evo.title", DescKey: "lesson.evo.desc",
			Level: 1, PresetName: "Aggregation",
			Steps: []LessonStep{
				{TextKey: "lesson.evo.1", WaitTicks: 0},
				{TextKey: "lesson.evo.2", WaitTicks: 0, SetupFunc: "enable_evolution"},
				{TextKey: "lesson.evo.3", WaitTicks: 600, Highlight: "fitness"},
				{TextKey: "lesson.evo.4", WaitTicks: 0},
				{TextKey: "lesson.evo.5", WaitTicks: 0},
				{TextKey: "lesson.evo.6", WaitTicks: 0},
				{TextKey: "lesson.evo.7", WaitTicks: 0},
			},
			Challenge: &Challenge{
				DescKey: "lesson.evo.challenge", MetricFunc: "fitness_over_200",
				ThresholdGold: 300, ThresholdSilver: 200, ThresholdBronze: 100,
			},
		},
		{
			ID: LessonObstacles, TitleKey: "lesson.obs.title", DescKey: "lesson.obs.desc",
			Level: 1, PresetName: "Wall Follow",
			Steps: []LessonStep{
				{TextKey: "lesson.obs.1", WaitTicks: 0},
				{TextKey: "lesson.obs.2", WaitTicks: 0, SetupFunc: "enable_maze"},
				{TextKey: "lesson.obs.3", WaitTicks: 300, Highlight: "bots"},
				{TextKey: "lesson.obs.4", WaitTicks: 0},
				{TextKey: "lesson.obs.5", WaitTicks: 0},
			},
			Challenge: &Challenge{
				DescKey: "lesson.obs.challenge", MetricFunc: "coverage",
				ThresholdGold: 70, ThresholdSilver: 50, ThresholdBronze: 30,
			},
		},

		// ===== ADVANCED (Level 2) =====
		{
			ID: LessonAlgoIntro, TitleKey: "lesson.algo.title", DescKey: "lesson.algo.desc",
			Level: 2, PresetName: "Aggregation",
			Steps: []LessonStep{
				{TextKey: "lesson.algo.1", WaitTicks: 0},
				{TextKey: "lesson.algo.2", WaitTicks: 0},
				{TextKey: "lesson.algo.3", WaitTicks: 0},
				{TextKey: "lesson.algo.4", WaitTicks: 0},
				{TextKey: "lesson.algo.5", WaitTicks: 0},
				{TextKey: "lesson.algo.6", WaitTicks: 0},
			},
			Challenge: nil,
		},
		{
			ID: LessonGWO, TitleKey: "lesson.gwo.title", DescKey: "lesson.gwo.desc",
			Level: 2, PresetName: "Aggregation",
			Steps: []LessonStep{
				{TextKey: "lesson.gwo.1", WaitTicks: 0},
				{TextKey: "lesson.gwo.2", WaitTicks: 0, SetupFunc: "enable_gwo"},
				{TextKey: "lesson.gwo.3", WaitTicks: 600, Highlight: "bots"},
				{TextKey: "lesson.gwo.4", WaitTicks: 0},
				{TextKey: "lesson.gwo.5", WaitTicks: 0},
				{TextKey: "lesson.gwo.6", WaitTicks: 0},
				{TextKey: "lesson.gwo.7", WaitTicks: 0},
				{TextKey: "lesson.gwo.8", WaitTicks: 0},
			},
			Challenge: &Challenge{
				DescKey: "lesson.gwo.challenge", MetricFunc: "fitness_over_200",
				ThresholdGold: 250, ThresholdSilver: 180, ThresholdBronze: 100,
			},
		},
		{
			ID: LessonSwarmVsClassic, TitleKey: "lesson.svc.title", DescKey: "lesson.svc.desc",
			Level: 2, PresetName: "Aggregation",
			Steps: []LessonStep{
				{TextKey: "lesson.svc.1", WaitTicks: 0},
				{TextKey: "lesson.svc.2", WaitTicks: 0},
				{TextKey: "lesson.svc.3", WaitTicks: 0},
				{TextKey: "lesson.svc.4", WaitTicks: 0},
				{TextKey: "lesson.svc.5", WaitTicks: 0},
				{TextKey: "lesson.svc.6", WaitTicks: 0},
			},
			Challenge: nil,
		},
		{
			ID: LessonFactory, TitleKey: "lesson.fact.title", DescKey: "lesson.fact.desc",
			Level: 2, PresetName: "Aggregation",
			Steps: []LessonStep{
				{TextKey: "lesson.fact.1", WaitTicks: 0},
				{TextKey: "lesson.fact.2", WaitTicks: 0},
				{TextKey: "lesson.fact.3", WaitTicks: 0},
				{TextKey: "lesson.fact.4", WaitTicks: 0},
				{TextKey: "lesson.fact.5", WaitTicks: 0},
				{TextKey: "lesson.fact.6", WaitTicks: 0},
			},
			Challenge: nil,
		},
	}
}

// AdvanceStep advances the lesson to the next step, handling challenge transitions.
func AdvanceStep(ls *LearningState) {
	if ls == nil || !ls.Active {
		return
	}
	lessons := GetAllLessons()
	if int(ls.CurrentLesson) >= len(lessons) {
		return
	}
	lesson := lessons[ls.CurrentLesson]
	ls.CurrentStep++
	ls.StepTimer = 0
	if ls.CurrentStep >= len(lesson.Steps) {
		// Lesson steps complete -- start challenge if present
		if lesson.Challenge != nil && !ls.ChallengeActive {
			ls.ChallengeActive = true
			ls.ChallengeTicks = 0
			ls.ChallengeValue = 0
		} else {
			// No challenge or challenge already done; end lesson
			ls.Active = false
			ls.ChallengeActive = false
		}
	}
}

// TickLesson updates the lesson timer each simulation tick.
func TickLesson(ls *LearningState) {
	if ls == nil || !ls.Active {
		return
	}
	if ls.ChallengeActive {
		ls.ChallengeTicks++
		return
	}
	lessons := GetAllLessons()
	if int(ls.CurrentLesson) >= len(lessons) {
		return
	}
	lesson := lessons[ls.CurrentLesson]
	if ls.CurrentStep >= len(lesson.Steps) {
		return
	}
	step := lesson.Steps[ls.CurrentStep]
	if step.WaitTicks > 0 {
		ls.StepTimer++
		if ls.StepTimer >= step.WaitTicks {
			AdvanceStep(ls)
		}
	}
}

// EvaluateChallenge checks the current challenge metric and returns a rating (0-3).
func EvaluateChallenge(ls *LearningState, metricValue float64) int {
	if ls == nil || !ls.ChallengeActive {
		return 0
	}
	lessons := GetAllLessons()
	if int(ls.CurrentLesson) >= len(lessons) {
		return 0
	}
	ch := lessons[ls.CurrentLesson].Challenge
	if ch == nil {
		return 0
	}
	if metricValue >= ch.ThresholdGold {
		return 3
	}
	if metricValue >= ch.ThresholdSilver {
		return 2
	}
	if metricValue >= ch.ThresholdBronze {
		return 1
	}
	return 0
}

// CompleteChallenge ends the active challenge and records the star rating.
func CompleteChallenge(ls *LearningState, rating int) {
	if ls == nil || !ls.ChallengeActive {
		return
	}
	old := ls.Completed[ls.CurrentLesson]
	if rating > old {
		ls.TotalStars -= old
		ls.Completed[ls.CurrentLesson] = rating
		ls.TotalStars += rating
	}
	ls.ChallengeActive = false
	ls.Active = false
}

// StartLesson begins a specific lesson from step 0.
func StartLesson(ls *LearningState, id LessonID) {
	if ls == nil {
		return
	}
	ls.CurrentLesson = id
	ls.CurrentStep = 0
	ls.StepTimer = 0
	ls.Active = true
	ls.ChallengeActive = false
	ls.ChallengeTicks = 0
	ls.ChallengeValue = 0
	ls.ShowMenu = false
}
