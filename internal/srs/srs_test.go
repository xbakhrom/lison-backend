package srs

import (
	"math"
	"testing"
	"time"
)

func TestScheduleUsesSM2LearningIntervals(t *testing.T) {
	today := time.Date(2026, 9, 16, 15, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		card     Card
		rating   Rating
		wantDays int
		wantEase float64
		wantReps int
	}{
		{"new hard", Card{State: "new", EaseFactor: 2.5}, Hard, 1, 2.36, 1},
		{"new good", Card{State: "new", EaseFactor: 2.5}, Good, 1, 2.5, 1},
		{"new easy", Card{State: "new", EaseFactor: 2.5}, Easy, 1, 2.6, 1},
		{"second repetition", Card{State: "review", IntervalDays: 1, EaseFactor: 2.5, Repetitions: 1}, Good, 6, 2.5, 2},
		{"mature hard", Card{State: "review", IntervalDays: 6, EaseFactor: 2.5, Repetitions: 2}, Hard, 15, 2.36, 3},
		{"mature good", Card{State: "review", IntervalDays: 6, EaseFactor: 2.5, Repetitions: 2}, Good, 15, 2.5, 3},
		{"mature easy", Card{State: "review", IntervalDays: 6, EaseFactor: 2.5, Repetitions: 2}, Easy, 15, 2.6, 3},
		{"fractional interval rounds up", Card{State: "review", IntervalDays: 7, EaseFactor: 2.36, Repetitions: 3}, Good, 17, 2.36, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Schedule(tt.card, tt.rating, today)
			if err != nil {
				t.Fatal(err)
			}
			if got.IntervalDays != tt.wantDays || math.Abs(got.EaseFactor-tt.wantEase) > 1e-9 || got.Repetitions != tt.wantReps {
				t.Fatalf("result = %+v, want days=%d ease=%v repetitions=%d", got, tt.wantDays, tt.wantEase, tt.wantReps)
			}
			if got.State != "review" {
				t.Fatalf("state = %q, want review", got.State)
			}
		})
	}
}

func TestScheduleKeepsMinimumEaseFactor(t *testing.T) {
	result, err := Schedule(Card{State: "review", IntervalDays: 10, EaseFactor: 1.3, Repetitions: 3}, Hard, time.Now())
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if result.EaseFactor != 1.3 {
		t.Fatalf("ease factor = %v, want 1.3", result.EaseFactor)
	}
}

func TestScheduleRejectsUnknownRating(t *testing.T) {
	if _, err := Schedule(Card{State: "new", EaseFactor: 2.5}, Rating("again"), time.Now()); err == nil {
		t.Fatal("Schedule() accepted an unknown rating")
	}
}
