package tasktemplate

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo   Repository
	now    func() time.Time
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		now:    func() time.Time { return time.Now().UTC() },
		logger: logger,
	}
}

func (s *Service) CreateWithGeneration(ctx context.Context, input CreateInput, from, to time.Time) (*tpldomain.Template, GenerateResult, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, GenerateResult{}, err
	}

	params, err := tpldomain.ParsePeriodicityParams(input.PeriodicityType, input.PeriodicityParams)
	if err != nil {
		return nil, GenerateResult{}, err
	}
	if err := params.Validate(); err != nil {
		return nil, GenerateResult{}, err
	}

	now := s.now()
	model := &tpldomain.Template{
		Title:                input.Title,
		Description:          input.Description,
		PeriodicityType:      input.PeriodicityType,
		PeriodicityParams:    params,
		RawPeriodicityParams: input.PeriodicityParams,
		StartDate:            input.StartDate,
		EndDate:              input.EndDate,
		IsActive:             input.IsActive,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	var tasks []taskdomain.Task
	if model.IsActive {
		dates := ComputeDates(model, from, to)
		tasks = make([]taskdomain.Task, 0, len(dates))
		for _, d := range dates {
			sd := d
			tasks = append(tasks, taskdomain.Task{
				Title:         model.Title,
				Description:   model.Description,
				Status:        taskdomain.StatusNew,
				ScheduledDate: &sd,
				CreatedAt:     now,
				UpdatedAt:     now,
			})
		}
	}

	created, inserted, err := s.repo.CreateWithTasks(ctx, model, tasks)
	if err != nil {
		return nil, GenerateResult{}, err
	}

	return created, GenerateResult{
		TotalDates:     len(tasks),
		GeneratedCount: inserted,
	}, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*tpldomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*tpldomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	if err := validateUpdateInput(input); err != nil {
		return nil, err
	}

	params, err := tpldomain.ParsePeriodicityParams(input.PeriodicityType, input.PeriodicityParams)
	if err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}

	model := &tpldomain.Template{
		ID:                   id,
		Title:                input.Title,
		Description:          input.Description,
		PeriodicityType:      input.PeriodicityType,
		PeriodicityParams:    params,
		RawPeriodicityParams: input.PeriodicityParams,
		StartDate:            input.StartDate,
		EndDate:              input.EndDate,
		IsActive:             input.IsActive,
		UpdatedAt:            s.now(),
	}

	return s.repo.Update(ctx, model)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]tpldomain.Template, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) GenerateAllActive(ctx context.Context, from, to time.Time) (GenerateResult, error) {
	templates, err := s.repo.List(ctx, ListFilter{ActiveOnly: true})
	if err != nil {
		return GenerateResult{}, fmt.Errorf("list active templates: %w", err)
	}

	var total GenerateResult
	for i := range templates {
		res, err := s.generateForTemplate(ctx, &templates[i], from, to)
		if err != nil {
			s.logger.Error("generate tasks for template",
				"template_id", templates[i].ID,
				"error", err,
			)
			continue
		}
		total.TotalDates += res.TotalDates
		total.GeneratedCount += res.GeneratedCount
	}

	return total, nil
}

func (s *Service) generateForTemplate(ctx context.Context, tpl *tpldomain.Template, from, to time.Time) (GenerateResult, error) {
	dates := ComputeDates(tpl, from, to)
	if len(dates) == 0 {
		return GenerateResult{}, nil
	}

	now := s.now()
	tasks := make([]taskdomain.Task, 0, len(dates))
	for _, d := range dates {
		sd := d
		tasks = append(tasks, taskdomain.Task{
			Title:         tpl.Title,
			Description:   tpl.Description,
			Status:        taskdomain.StatusNew,
			TemplateID:    &tpl.ID,
			ScheduledDate: &sd,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}

	inserted, err := s.repo.BatchCreateTasks(ctx, tasks)
	if err != nil {
		return GenerateResult{}, err
	}

	return GenerateResult{
		TotalDates:     len(dates),
		GeneratedCount: inserted,
	}, nil
}

// ComputeDates возвращает календарные даты вхождений шаблона в [from, to] с учётом start_date/end_date шаблона.
func ComputeDates(tpl *tpldomain.Template, from, to time.Time) []time.Time {
	effectiveFrom := from
	if tpl.StartDate.After(effectiveFrom) {
		effectiveFrom = tpl.StartDate
	}

	effectiveTo := to
	if tpl.EndDate != nil && tpl.EndDate.Before(effectiveTo) {
		effectiveTo = *tpl.EndDate
	}

	if effectiveFrom.After(effectiveTo) {
		return nil
	}

	return tpl.PeriodicityParams.MatchDates(effectiveFrom, effectiveTo)
}

func validateCreateInput(input CreateInput) error {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.PeriodicityType.Valid() {
		return fmt.Errorf("%w: invalid periodicity_type", ErrInvalidInput)
	}

	if input.StartDate.IsZero() {
		return fmt.Errorf("%w: start_date is required", ErrInvalidInput)
	}

	if input.EndDate != nil && input.EndDate.Before(input.StartDate) {
		return fmt.Errorf("%w: end_date must not be before start_date", ErrInvalidInput)
	}

	return nil
}

func validateUpdateInput(input UpdateInput) error {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.PeriodicityType.Valid() {
		return fmt.Errorf("%w: invalid periodicity_type", ErrInvalidInput)
	}

	if input.StartDate.IsZero() {
		return fmt.Errorf("%w: start_date is required", ErrInvalidInput)
	}

	if input.EndDate != nil && input.EndDate.Before(input.StartDate) {
		return fmt.Errorf("%w: end_date must not be before start_date", ErrInvalidInput)
	}

	return nil
}
