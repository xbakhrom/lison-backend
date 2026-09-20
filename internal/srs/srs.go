package srs

import (
	"fmt"
	"math"
	"time"
)

type Rating string

const (
	Hard Rating = "hard"
	Good Rating = "good"
	Easy Rating = "easy"
)

type Card struct {
	State        string
	IntervalDays int
	EaseFactor   float64
	Repetitions  int
	Lapses       int
}

type Result struct {
	State        string
	IntervalDays int
	EaseFactor   float64
	Repetitions  int
	Lapses       int
	DueDate      time.Time
}

func Schedule(card Card, rating Rating, today time.Time) (Result, error) {
	if card.EaseFactor == 0 {
		card.EaseFactor = 2.5
	}
	quality, err := sm2Quality(rating)
	if err != nil {
		return Result{}, err
	}

	qualityGap := float64(5 - quality)
	nextEase := math.Max(1.3, card.EaseFactor+0.1-qualityGap*(0.08+qualityGap*0.02))
	successfulRepetitions := card.Repetitions
	if card.State == "new" || card.State == "relearning" || card.IntervalDays <= 0 {
		successfulRepetitions = 0
	}

	intervalDays := 1
	switch successfulRepetitions {
	case 0:
		intervalDays = 1
	case 1:
		intervalDays = 6
	default:
		intervalDays = max(1, int(math.Ceil(float64(card.IntervalDays)*card.EaseFactor)))
	}

	result := Result{
		State:        "review",
		IntervalDays: intervalDays,
		EaseFactor:   nextEase,
		Repetitions:  successfulRepetitions + 1,
		Lapses:       card.Lapses,
	}
	result.DueDate = day(today).AddDate(0, 0, result.IntervalDays)
	return result, nil
}

func sm2Quality(rating Rating) (int, error) {
	switch rating {
	case Hard:
		return 3, nil
	case Good:
		return 4, nil
	case Easy:
		return 5, nil
	default:
		return 0, fmt.Errorf("unknown rating %q", rating)
	}
}

func day(t time.Time) time.Time {
	year, month, date := t.Date()
	return time.Date(year, month, date, 0, 0, 0, 0, t.Location())
}
