package srs

import (
	"testing"
	"time"
)

func TestScheduleNewCard(t *testing.T) {
	today := time.Date(2026, 9, 16, 15, 0, 0, 0, time.UTC)
	tests := []struct {
		rating Rating
		want   int
	}{
		{Hard, 1},
		{Good, 3},
		{Easy, 7},
		{Again, 0},
	}
	for _, tt := range tests {
		t.Run(string(tt.rating), func(t *testing.T) {
			got, err := Schedule(Card{State: "new", EaseFactor: 2.5}, tt.rating, today)
			if err != nil {
				t.Fatal(err)
			}
			if got.IntervalDays != tt.want {
				t.Fatalf("interval = %d, want %d", got.IntervalDays, tt.want)
			}
		})
	}
}

func TestAgainMovesReviewToRelearning(t *testing.T) {
	result, err := Schedule(Card{State: "review", IntervalDays: 10, EaseFactor: 2.5}, Again, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "relearning" || result.Lapses != 1 || result.IntervalDays != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
