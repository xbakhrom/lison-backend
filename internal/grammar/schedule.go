package grammar

import (
	"fmt"
	"math"
	"time"
)

type Progress struct {
	IntervalDays int
	EaseFactor   float64
	Repetitions  int
}

type Result struct {
	Status       string
	IntervalDays int
	EaseFactor   float64
	Repetitions  int
	DueDate      time.Time
}

// Schedule keeps grammar reviews deliberately lighter than vocabulary reviews.
// A successful game moves the whole topic forward as one review item.
func Schedule(progress Progress, score int, today time.Time) (Result, error) {
	if score < 0 || score > 100 {
		return Result{}, fmt.Errorf("score must be between 0 and 100")
	}
	if progress.EaseFactor == 0 {
		progress.EaseFactor = 2.3
	}

	if score < 60 {
		return Result{
			Status:       "learning",
			IntervalDays: 0,
			EaseFactor:   math.Max(1.3, progress.EaseFactor-0.2),
			Repetitions:  0,
			DueDate:      day(today),
		}, nil
	}

	interval := 1
	switch progress.Repetitions {
	case 0:
		interval = 1
	case 1:
		interval = 3
	case 2:
		interval = 7
	default:
		interval = max(1, int(math.Ceil(float64(progress.IntervalDays)*progress.EaseFactor)))
	}

	nextEase := progress.EaseFactor
	if score >= 90 {
		nextEase += 0.1
	} else if score < 75 {
		nextEase = math.Max(1.3, nextEase-0.15)
	}

	return Result{
		Status:       "review",
		IntervalDays: interval,
		EaseFactor:   nextEase,
		Repetitions:  progress.Repetitions + 1,
		DueDate:      day(today).AddDate(0, 0, interval),
	}, nil
}

func day(t time.Time) time.Time {
	year, month, date := t.Date()
	return time.Date(year, month, date, 0, 0, 0, 0, t.Location())
}
