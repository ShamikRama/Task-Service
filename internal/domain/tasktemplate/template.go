package tasktemplate

import (
	"encoding/json"
	"fmt"
	"time"
)

type PeriodicityType string

const (
	PeriodicityDaily         PeriodicityType = "daily"
	PeriodicityMonthly       PeriodicityType = "monthly"
	PeriodicitySpecificDates PeriodicityType = "specific_dates"
	PeriodicityEvenDays      PeriodicityType = "even_days"
	PeriodicityOddDays       PeriodicityType = "odd_days"
)

func (p PeriodicityType) Valid() bool {
	switch p {
	case PeriodicityDaily, PeriodicityMonthly, PeriodicitySpecificDates,
		PeriodicityEvenDays, PeriodicityOddDays:
		return true
	default:
		return false
	}
}

type Template struct {
	ID                   int64
	Title                string
	Description          string
	PeriodicityType      PeriodicityType
	PeriodicityParams    PeriodicityParams
	RawPeriodicityParams json.RawMessage
	StartDate            time.Time
	EndDate              *time.Time
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type PeriodicityParams interface {
	MatchDates(from, to time.Time) []time.Time
	Validate() error
}

func ParsePeriodicityParams(ptype PeriodicityType, raw json.RawMessage) (PeriodicityParams, error) {
	switch ptype {
	case PeriodicityDaily:
		var p DailyParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("%w: invalid daily params: %v", ErrInvalidInput, err)
		}
		return &p, nil

	case PeriodicityMonthly:
		var p MonthlyParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("%w: invalid monthly params: %v", ErrInvalidInput, err)
		}
		return &p, nil

	case PeriodicitySpecificDates:
		var p SpecificDatesParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("%w: invalid specific_dates params: %v", ErrInvalidInput, err)
		}
		return &p, nil

	case PeriodicityEvenDays:
		return &EvenDaysParams{}, nil

	case PeriodicityOddDays:
		return &OddDaysParams{}, nil

	default:
		return nil, fmt.Errorf("%w: unknown periodicity type %q", ErrInvalidInput, ptype)
	}
}
