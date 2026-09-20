package srs

import (
	"fmt"
	"math"
	"time"
)

type Rating string

const (
	Again Rating = "again"
	Hard  Rating = "hard"
	Good  Rating = "good"
	Easy  Rating = "easy"
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
	result := Result{
		State:        card.State,
		IntervalDays: card.IntervalDays,
		EaseFactor:   card.EaseFactor,
		Repetitions:  card.Repetitions + 1,
		Lapses:       card.Lapses,
	}

	switch rating {
	case Again:
		if card.State == "review" {
			result.Lapses++
		}
		result.State = "relearning"
		result.IntervalDays = 0
		result.EaseFactor = math.Max(1.3, card.EaseFactor-0.2)
	case Hard:
		result.State = "review"
		if card.State == "new" || card.State == "relearning" || card.IntervalDays == 0 {
			result.IntervalDays = 1
		} else {
			result.IntervalDays = max(1, int(math.Ceil(float64(card.IntervalDays)*1.2)))
		}
		result.EaseFactor = math.Max(1.3, card.EaseFactor-0.15)
	case Good:
		result.State = "review"
		switch card.State {
		case "new":
			result.IntervalDays = 3
		case "relearning":
			result.IntervalDays = 1
		default:
			result.IntervalDays = max(1, int(math.Ceil(float64(card.IntervalDays)*card.EaseFactor)))
		}
	case Easy:
		result.State = "review"
		switch card.State {
		case "new":
			result.IntervalDays = 7
		case "relearning":
			result.IntervalDays = 3
		default:
			result.IntervalDays = max(1, int(math.Ceil(float64(card.IntervalDays)*(card.EaseFactor+0.3))))
		}
		result.EaseFactor = math.Min(3.0, card.EaseFactor+0.15)
	default:
		return Result{}, fmt.Errorf("unknown rating %q", rating)
	}

	result.DueDate = day(today).AddDate(0, 0, result.IntervalDays)
	return result, nil
}

func day(t time.Time) time.Time {
	year, month, date := t.Date()
	return time.Date(year, month, date, 0, 0, 0, 0, t.Location())
}
