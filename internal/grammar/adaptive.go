package grammar

const (
	MinDifficulty = 1
	MaxDifficulty = 3
)

type AdaptiveState struct {
	Difficulty    int
	CorrectStreak int
	MistakeStreak int
}

// AdvanceDifficulty adjusts a session after two consecutive answers of the
// same quality. A learner moves up after two correct answers and receives an
// easier question after two mistakes.
func AdvanceDifficulty(state AdaptiveState, correct bool) AdaptiveState {
	state.Difficulty = clampDifficulty(state.Difficulty)
	if correct {
		state.CorrectStreak++
		state.MistakeStreak = 0
		if state.CorrectStreak >= 2 && state.Difficulty < MaxDifficulty {
			state.Difficulty++
			state.CorrectStreak = 0
		}
		return state
	}

	state.MistakeStreak++
	state.CorrectStreak = 0
	if state.MistakeStreak >= 2 && state.Difficulty > MinDifficulty {
		state.Difficulty--
		state.MistakeStreak = 0
	}
	return state
}

func clampDifficulty(value int) int {
	if value < MinDifficulty {
		return MinDifficulty
	}
	if value > MaxDifficulty {
		return MaxDifficulty
	}
	return value
}
