package grammar

import (
	"testing"
	"time"
)

func TestScheduleLearningAndReviewIntervals(t *testing.T) {
	today := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

	failed, err := Schedule(Progress{}, 40, today)
	if err != nil || failed.Status != "learning" || failed.IntervalDays != 0 || !failed.DueDate.Equal(day(today)) {
		t.Fatalf("unexpected failed result: %#v, %v", failed, err)
	}

	first, _ := Schedule(Progress{}, 80, today)
	if first.Status != "review" || first.IntervalDays != 1 || first.Repetitions != 1 {
		t.Fatalf("unexpected first result: %#v", first)
	}

	second, _ := Schedule(Progress{IntervalDays: 1, EaseFactor: first.EaseFactor, Repetitions: 1}, 90, today)
	if second.IntervalDays != 3 || second.Repetitions != 2 {
		t.Fatalf("unexpected second result: %#v", second)
	}

	third, _ := Schedule(Progress{IntervalDays: 3, EaseFactor: second.EaseFactor, Repetitions: 2}, 90, today)
	if third.IntervalDays != 7 || third.Repetitions != 3 {
		t.Fatalf("unexpected third result: %#v", third)
	}
}

func TestScheduleRejectsInvalidScore(t *testing.T) {
	if _, err := Schedule(Progress{}, 101, time.Now()); err == nil {
		t.Fatal("expected invalid score error")
	}
}
