package tasktemplate_test

import (
	"io"
	"log/slog"
	"time"
)

func td(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func ptrDate(t time.Time) *time.Time { return &t }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
