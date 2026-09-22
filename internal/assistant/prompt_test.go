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
