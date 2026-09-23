package assistant

import (
	"strings"
	"testing"
)

func TestBuildSystemInstructionCarriesLearnerContext(t *testing.T) {
	prompt := BuildSystemInstruction(Context{
		FirstName:    "Bekzod",
		Level:        "A2",
		RecentTopics: []string{"Город", "Дружба"},
		DueCards:     7,
	})

	for _, want := range []string{"Bekzod", "A2", "Город", "Дружба", "7 flashcards"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt is missing %q", want)
		}
	}
	if !strings.Contains(prompt, "Uzbek") || !strings.Contains(prompt, "Russian") {
		t.Error("prompt does not state the language rules")
	}
}

// Maks kept slipping into Uzbek, which defeats the immersion the app is for.
// Russian is the default and Uzbek is strictly on request.
func TestPromptKeepsMaksInRussian(t *testing.T) {
	prompt := BuildSystemInstruction(Context{FirstName: "Bekzod", Level: "A2"})

	if !strings.Contains(prompt, "Do NOT speak Uzbek unless the learner explicitly asks") {
		t.Error("prompt does not forbid unprompted Uzbek")
	}
	if !strings.Contains(prompt, "stay in Russian and make the Russian easier") {
		t.Error("prompt does not say what to do instead of falling back to Uzbek")
	}
	// Being spoken to in Uzbek is not the same as being asked for Uzbek, and
	// that was exactly the gap the learner fell through.
	if !strings.Contains(prompt, "merely speaking Uzbek to you is not") {
		t.Error("prompt does not separate a request for Uzbek from being spoken to in Uzbek")
	}
}

func TestLessonVocabularyIsMarkedAsReferenceOnly(t *testing.T) {
	prompt := BuildSystemInstruction(Context{
		Topic: &TopicContext{
			ID:    "topic_gorod",
			Slug:  "gorod",
			Title: "Город",
			Words: []Word{{Russian: "Столица", Uzbek: "poytaxt"}},
		},
	})

	// He needs the Uzbek meanings to teach, but handing him a bilingual list is
	// an invitation to read it out.
	if !strings.Contains(prompt, "for your own reference only") {
		t.Error("the Uzbek column is not marked as reference only")
	}
}

// The Uzbek side of a card is display data. Without this rule Maks quizzes by
// reading it out, which both breaks immersion and makes the review trivial.
func TestPromptQuizzesInRussianNotUzbek(t *testing.T) {
	prompt := BuildSystemInstruction(Context{})
	if !strings.Contains(prompt, "Never read the Uzbek side aloud") {
		t.Error("prompt does not forbid quizzing by reading the Uzbek side")
	}
	if !strings.Contains(prompt, "describe the word in simple Russian") {
		t.Error("prompt does not say how to prompt a card instead")
	}
}

func TestKnownLevelGetsConcreteGuidance(t *testing.T) {
	a1 := BuildSystemInstruction(Context{Level: "A1"})
	if !strings.Contains(a1, "lots of repetition") {
		t.Error("A1 prompt is missing beginner guidance")
	}
	b2 := BuildSystemInstruction(Context{Level: "B2"})
	if !strings.Contains(b2, "idioms included") {
		t.Error("B2 prompt is missing advanced guidance")
	}
	if strings.Contains(a1, "idioms included") {
		t.Error("A1 prompt carries B2 guidance")
	}
}

func TestBuildSystemInstructionWithoutLevel(t *testing.T) {
	prompt := BuildSystemInstruction(Context{FirstName: "Bekzod"})
	if !strings.Contains(prompt, "Level unknown") {
		t.Error("an unknown level should tell Maks to start easy")
	}
	if strings.Contains(prompt, "CURRENT LESSON") {
		t.Error("no lesson was supplied, so none should be described")
	}
}

func TestBuildSystemInstructionWithLesson(t *testing.T) {
	prompt := BuildSystemInstruction(Context{
		FirstName: "Bekzod",
		Topic: &TopicContext{
			ID:        "topic_gorod",
			Slug:      "gorod",
			Title:     "Город",
			Summary:   "Город и транспорт",
			Words:     []Word{{Russian: "Столица", Uzbek: "poytaxt"}},
			Unlearned: []string{"Столица"},
		},
	})

	for _, want := range []string{"CURRENT LESSON", "topic_gorod", "gorod", "Столица", "poytaxt", "NOT added"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt is missing %q", want)
		}
	}
}

func TestBuildSystemInstructionSkipsEmptyFields(t *testing.T) {
	prompt := BuildSystemInstruction(Context{})
	if strings.Contains(prompt, "- Name:") {
		t.Error("an unknown name should be left out rather than rendered empty")
	}
	if strings.Contains(prompt, "flashcards due") {
		t.Error("a learner with nothing due should not be offered a review")
	}
}
