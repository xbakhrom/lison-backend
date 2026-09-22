package discussion

import (
	"strings"
	"testing"
)

func TestShippedContentIsValid(t *testing.T) {
	sets, err := LoadSets()
	if err != nil {
		t.Fatalf("LoadSets: %v", err)
	}
	if issues := Validate(sets); len(issues) > 0 {
		t.Fatalf("shipped discussion content is invalid: %s", strings.Join(issues, "; "))
	}
	if len(sets) == 0 {
		t.Fatal("no discussion sets are shipped")
	}
	// Every conversation needs somewhere to start when no lesson is open.
	general := false
	for _, set := range sets {
		if set.TopicID == "" {
			general = true
		}
	}
	if !general {
		t.Error("no general question set: a conversation outside a lesson has nothing to ask")
	}
}

func TestSetsAreOrderedByPosition(t *testing.T) {
	sets, err := LoadSets()
	if err != nil {
		t.Fatalf("LoadSets: %v", err)
	}
	for i := 1; i < len(sets); i++ {
		if sets[i-1].Position > sets[i].Position {
			t.Fatalf("sets are not ordered: %s (%d) came before %s (%d)",
				sets[i-1].ID, sets[i-1].Position, sets[i].ID, sets[i].Position)
		}
	}
}

func TestValidateCatchesBrokenSets(t *testing.T) {
	tests := map[string]struct {
		set  Set
		want string
	}{
		"bad id":        {Set{ID: "gorod", Title: "T", Position: 1}, "discussion_<lower_snake_case>"},
		"bad topic":     {Set{ID: "discussion_x", Title: "T", Position: 1, TopicID: "gorod"}, "topicId must match"},
		"no title":      {Set{ID: "discussion_x", Position: 1}, "title is required"},
		"bad position":  {Set{ID: "discussion_x", Title: "T", Position: 0}, "position must be 1 or greater"},
		"too few":       {Set{ID: "discussion_x", Title: "T", Position: 1}, "needs at least"},
		"no hint":       {Set{ID: "discussion_x", Title: "T", Position: 1, Questions: []Question{{ID: "q1", Question: "Как дела?", FollowUps: []string{"А?"}}}}, "uzbekHint is required"},
		"no follow-ups": {Set{ID: "discussion_x", Title: "T", Position: 1, Questions: []Question{{ID: "q1", Question: "Как дела?", UzbekHint: "Ahvol?"}}}, "needs at least one follow-up"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			issues := Validate([]Set{test.set})
			if !containsSubstring(issues, test.want) {
				t.Errorf("issues = %v, want one containing %q", issues, test.want)
			}
		})
	}
}

func TestValidateCatchesDuplicateQuestionIDs(t *testing.T) {
	question := Question{ID: "shared", Question: "Как дела?", UzbekHint: "Ahvol?", FollowUps: []string{"А?"}}
	issues := Validate([]Set{
		{ID: "discussion_a", Title: "A", Position: 1, Questions: []Question{question}},
		{ID: "discussion_b", Title: "B", Position: 2, Questions: []Question{question}},
	})
	if !containsSubstring(issues, "duplicate question id") {
		t.Errorf("issues = %v, want a duplicate id report", issues)
	}
}

func TestValidateRejectsEmptyContent(t *testing.T) {
	if issues := Validate(nil); len(issues) != 1 {
		t.Errorf("issues = %v, want exactly one 'no sets' report", issues)
	}
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
