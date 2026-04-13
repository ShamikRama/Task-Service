package scheduler

import (
	"context"
	"log/slog"
	"time"

	tplusecase "example.com/taskservice/internal/usecase/tasktemplate"
)

type Scheduler struct {
	usecase  tplusecase.Usecase
	horizon  int
	interval time.Duration
	logger   *slog.Logger
}

func New(usecase tplusecase.Usecase, horizonDays int, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		usecase:  usecase,
		horizon:  horizonDays,
		interval: 24 * time.Hour,
		logger:   logger,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.logger.Info("scheduler started", "horizon_days", s.horizon)

	s.generate(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.generate(ctx)
		}
	}
}

func (s *Scheduler) generate(ctx context.Context) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, s.horizon)

	result, err := s.usecase.GenerateAllActive(ctx, from, to)
	if err != nil {
		s.logger.Error("scheduler generate failed", "error", err)
		return
	}

	if result.GeneratedCount > 0 {
		s.logger.Info("scheduler generated tasks",
			"generated", result.GeneratedCount,
			"total_dates", result.TotalDates,
			"skipped", int64(result.TotalDates)-result.GeneratedCount,
			"from", from.Format(time.DateOnly),
			"to", to.Format(time.DateOnly),
		)
	}
}
