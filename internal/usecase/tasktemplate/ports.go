package tasktemplate

import (
	"context"
	"encoding/json"
	"time"

	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	// CreateWithTasks атомарно создаёт шаблон и вставляет задачи (одна транзакция). tasks могут быть пустыми.
	CreateWithTasks(ctx context.Context, t *tpldomain.Template, tasks []taskdomain.Task) (*tpldomain.Template, int64, error)
	GetByID(ctx context.Context, id int64) (*tpldomain.Template, error)
	Update(ctx context.Context, t *tpldomain.Template) (*tpldomain.Template, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter ListFilter) ([]tpldomain.Template, error)
	BatchCreateTasks(ctx context.Context, tasks []taskdomain.Task) (int64, error)
}

type Usecase interface {
	GetByID(ctx context.Context, id int64) (*tpldomain.Template, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*tpldomain.Template, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter ListFilter) ([]tpldomain.Template, error)
	GenerateAllActive(ctx context.Context, from, to time.Time) (GenerateResult, error)
	// CreateWithGeneration создаёт шаблон и материализует задачи на [from, to] в одной транзакции.
	CreateWithGeneration(ctx context.Context, input CreateInput, from, to time.Time) (*tpldomain.Template, GenerateResult, error)
}

type CreateInput struct {
	Title             string
	Description       string
	PeriodicityType   tpldomain.PeriodicityType
	PeriodicityParams json.RawMessage
	StartDate         time.Time
	EndDate           *time.Time
	IsActive          bool
}

type UpdateInput struct {
	Title             string
	Description       string
	PeriodicityType   tpldomain.PeriodicityType
	PeriodicityParams json.RawMessage
	StartDate         time.Time
	EndDate           *time.Time
	IsActive          bool
}

type ListFilter struct {
	ActiveOnly bool
}

type GenerateResult struct {
	TotalDates     int
	GeneratedCount int64
}
