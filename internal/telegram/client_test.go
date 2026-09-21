package telegram

import "testing"

func TestRussianTopicNoun(t *testing.T) {
	tests := map[int]string{
		1: "тема", 2: "темы", 5: "тем", 11: "тем", 21: "тема", 24: "темы", 25: "тем",
	}
	for count, want := range tests {
		if got := russianTopicNoun(count); got != want {
			t.Errorf("russianTopicNoun(%d) = %q, want %q", count, got, want)
		}
	}
}
