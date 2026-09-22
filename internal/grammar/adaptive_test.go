package grammar

import "testing"

func TestAdvanceDifficultyRaisesAfterTwoCorrectAnswers(t *testing.T) {
	state := AdaptiveState{Difficulty: 1}
	state = AdvanceDifficulty(state, true)
	if state.Difficulty != 1 || state.CorrectStreak != 1 {
		t.Fatalf("unexpected state after first answer: %#v", state)
	}
	state = AdvanceDifficulty(state, true)
	if state.Difficulty != 2 || state.CorrectStreak != 0 {
		t.Fatalf("expected level two after streak: %#v", state)
	}
}

func TestAdvanceDifficultyReturnsToEasierQuestions(t *testing.T) {
	state := AdaptiveState{Difficulty: 3}
	state = AdvanceDifficulty(state, false)
	state = AdvanceDifficulty(state, false)
	if state.Difficulty != 2 || state.MistakeStreak != 0 {
		t.Fatalf("expected a step down after two mistakes: %#v", state)
	}
	state = AdvanceDifficulty(state, false)
	state = AdvanceDifficulty(state, false)
	if state.Difficulty != 1 {
		t.Fatalf("expected the easiest level after another two mistakes: %#v", state)
	}
}

func TestAdvanceDifficultyResetsOppositeStreakAndClampsLevel(t *testing.T) {
	state := AdaptiveState{Difficulty: 9, MistakeStreak: 1}
	state = AdvanceDifficulty(state, true)
	if state.Difficulty != 3 || state.MistakeStreak != 0 || state.CorrectStreak != 1 {
		t.Fatalf("unexpected clamped state: %#v", state)
	}
	state = AdvanceDifficulty(state, false)
	if state.CorrectStreak != 0 || state.MistakeStreak != 1 {
		t.Fatalf("opposite streak was not reset: %#v", state)
	}
}
