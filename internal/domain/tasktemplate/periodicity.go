package tasktemplate

import (
	"fmt"
	"sort"
	"time"
)

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func dateRange(from, to time.Time) (time.Time, time.Time) {
	return truncateToDate(from), truncateToDate(to)
}

type DailyParams struct {
	Interval int `json:"interval"`
}

func (p *DailyParams) Validate() error {
	if p.Interval < 1 {
		return fmt.Errorf("%w: daily interval must be >= 1", ErrInvalidInput)
	}
	return nil
}

func (p *DailyParams) MatchDates(from, to time.Time) []time.Time {
	from, to = dateRange(from, to)
	if from.After(to) {
		return nil
	}

	var dates []time.Time
	for d := from; !d.After(to); d = d.AddDate(0, 0, p.Interval) {
		dates = append(dates, d)
	}
	return dates
}

type MonthlyParams struct {
	Days []int `json:"days"`
}

func (p *MonthlyParams) Validate() error {
	if len(p.Days) == 0 {
		return fmt.Errorf("%w: monthly days must not be empty", ErrInvalidInput)
	}
	for _, d := range p.Days {
		if d < 1 || d > 31 {
			return fmt.Errorf("%w: monthly day must be between 1 and 31, got %d", ErrInvalidInput, d)
		}
	}
	return nil
}

func (p *MonthlyParams) MatchDates(from, to time.Time) []time.Time {
	from, to = dateRange(from, to)
	if from.After(to) {
		return nil
	}

	var dates []time.Time
	current := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)

	for current.Before(end) {
		daysInMonth := daysIn(current.Year(), current.Month())
		for _, day := range p.Days {
			if day > daysInMonth {
				continue
			}
			candidate := time.Date(current.Year(), current.Month(), day, 0, 0, 0, 0, time.UTC)
			if !candidate.Before(from) && !candidate.After(to) {
				dates = append(dates, candidate)
			}
		}
		current = current.AddDate(0, 1, 0)
	}

	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}

type SpecificDatesParams struct {
	Dates []string `json:"dates"`
}

func (p *SpecificDatesParams) Validate() error {
	if len(p.Dates) == 0 {
		return fmt.Errorf("%w: specific_dates list must not be empty", ErrInvalidInput)
	}
	for _, ds := range p.Dates {
		if _, err := time.Parse(time.DateOnly, ds); err != nil {
			return fmt.Errorf("%w: invalid date format %q, expected YYYY-MM-DD", ErrInvalidInput, ds)
		}
	}
	return nil
}

func (p *SpecificDatesParams) MatchDates(from, to time.Time) []time.Time {
	from, to = dateRange(from, to)
	if from.After(to) {
		return nil
	}

	var dates []time.Time
	for _, ds := range p.Dates {
		d, err := time.Parse(time.DateOnly, ds)
		if err != nil {
			continue
		}
		if !d.Before(from) && !d.After(to) {
			dates = append(dates, d)
		}
	}

	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}

type EvenDaysParams struct{}

func (p *EvenDaysParams) Validate() error { return nil }

func (p *EvenDaysParams) MatchDates(from, to time.Time) []time.Time {
	return filterDaysByParity(from, to, func(day int) bool { return day%2 == 0 })
}

type OddDaysParams struct{}

func (p *OddDaysParams) Validate() error { return nil }

func (p *OddDaysParams) MatchDates(from, to time.Time) []time.Time {
	return filterDaysByParity(from, to, func(day int) bool { return day%2 != 0 })
}

func filterDaysByParity(from, to time.Time, match func(int) bool) []time.Time {
	from, to = dateRange(from, to)
	if from.After(to) {
		return nil
	}

	var dates []time.Time
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if match(d.Day()) {
			dates = append(dates, d)
		}
	}
	return dates
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
