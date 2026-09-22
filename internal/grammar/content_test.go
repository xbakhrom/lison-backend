package grammar

import (
	"strings"
	"testing"
)

// The shipped bank is the product: a broken file would reorder the route or
// leave a learner without questions, so the test suite guards it directly.
func TestShippedContentIsValid(t *testing.T) {
	topics, err := LoadTopics()
	if err != nil {
		t.Fatalf("load topics: %v", err)
	}
	for _, issue := range Validate(topics) {
		t.Errorf("content issue: %s", issue)
	}
}

func TestRouteIsContinuousAndStagesAreOrdered(t *testing.T) {
	topics, err := LoadTopics()
	if err != nil {
		t.Fatalf("load topics: %v", err)
	}
	if len(topics) == 0 {
		t.Fatal("no topics loaded")
	}

	order := map[string]int{StageFoundation: 0, StageVerbs: 1, StageCases: 2, StageFluency: 3}
	previousStage := 0
	for index, topic := range topics {
		if topic.Position != index+1 {
			t.Errorf("topic %s sits at position %d, expected %d: the route must have no gaps", topic.ID, topic.Position, index+1)
		}
		if order[topic.Stage] < previousStage {
			t.Errorf("topic %s returns to stage %q after a later stage", topic.ID, topic.Stage)
		}
		previousStage = order[topic.Stage]
	}
}

func TestValidateRejectsBrokenQuestions(t *testing.T) {
	topic := Topic{
		ID: "grammar_demo", Slug: "demo", Title: "Demo", Summary: "Demo", Level: "A1",
		Icon: "✦", Stage: StageFoundation, Position: 1, Status: "published",
		Lesson: []LessonBlock{
			{Title: "a", Text: "a", Examples: []string{"a"}},
			{Title: "b", Text: "b", Examples: []string{"b"}},
			{Title: "c", Text: "c", Examples: []string{"c"}},
		},
		Practice: []Question{{
			ID: "dm-p01", Kind: "choice", Difficulty: 1, Prompt: "p", Phrase: "f",
			Options: []string{"один", "два", "три"}, Answer: "четыре", Explanation: "e",
		}},
		Game: []Question{{
			ID: "dm-g01", Kind: "order", Difficulty: 1, Prompt: "p", Phrase: "f",
			Tokens: []string{"Мы", "идём"}, Answer: "Мы идём", Explanation: "e",
		}},
	}

	issues := Validate([]Topic{topic})
	expectations := []string{
		"practice bank needs 60 questions",
		"is not one of the options",
		"the game bank only accepts choice and true_false questions",
		"order questions need at least 3 tokens",
	}
	for _, expectation := range expectations {
		if !containsSubstring(issues, expectation) {
			t.Errorf("expected an issue mentioning %q, got %v", expectation, issues)
		}
	}
}

func containsSubstring(issues []string, needle string) bool {
	for _, issue := range issues {
		if strings.Contains(issue, needle) {
			return true
		}
	}
	return false
}
