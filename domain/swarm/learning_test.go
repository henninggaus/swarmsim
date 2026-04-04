package swarm

import "testing"

func TestGetAllLessons(t *testing.T) {
	lessons := GetAllLessons()
	if len(lessons) != int(LessonCount) {
		t.Errorf("expected %d lessons, got %d", LessonCount, len(lessons))
	}
	for i, l := range lessons {
		if l.TitleKey == "" {
			t.Errorf("lesson %d: empty title key", i)
		}
		if len(l.Steps) == 0 {
			t.Errorf("lesson %d: no steps", i)
		}
		if l.Level < 0 || l.Level > 2 {
			t.Errorf("lesson %d: invalid level %d", i, l.Level)
		}
	}
}

func TestStartLesson(t *testing.T) {
	ls := &LearningState{}
	StartLesson(ls, LessonAggregation)
	if !ls.Active {
		t.Error("should be active")
	}
	if ls.CurrentLesson != LessonAggregation {
		t.Error("wrong lesson")
	}
	if ls.CurrentStep != 0 {
		t.Error("should start at step 0")
	}
}

func TestAdvanceStep(t *testing.T) {
	ls := &LearningState{}
	StartLesson(ls, LessonAggregation)
	lessons := GetAllLessons()
	maxSteps := len(lessons[LessonAggregation].Steps)

	// Advance through all steps
	for i := 0; i < maxSteps-1; i++ {
		AdvanceStep(ls)
		if ls.CurrentStep != i+1 {
			t.Errorf("step %d: expected %d", i, i+1)
		}
	}

	// One more advance should end the lesson (or start challenge)
	AdvanceStep(ls)
}

func TestCompleteChallenge(t *testing.T) {
	ls := &LearningState{}
	StartLesson(ls, LessonAggregation)
	// Advance past all steps to activate challenge
	lessons := GetAllLessons()
	maxSteps := len(lessons[LessonAggregation].Steps)
	for i := 0; i < maxSteps; i++ {
		AdvanceStep(ls)
	}
	if !ls.ChallengeActive {
		t.Fatal("challenge should be active after advancing past all steps")
	}

	CompleteChallenge(ls, 3) // gold
	if ls.Completed[LessonAggregation] != 3 {
		t.Error("should be gold (3)")
	}
	if ls.TotalStars != 3 {
		t.Error("should have 3 stars")
	}

	// Don't downgrade: restart and complete with bronze
	StartLesson(ls, LessonAggregation)
	for i := 0; i < maxSteps; i++ {
		AdvanceStep(ls)
	}
	CompleteChallenge(ls, 1) // bronze
	if ls.Completed[LessonAggregation] != 3 {
		t.Error("should stay gold")
	}
}
