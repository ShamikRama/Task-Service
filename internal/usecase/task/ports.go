package task

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter ListFilter) ([]taskdomain.Task, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter ListFilter) ([]taskdomain.Task, error)
}

type ListFilter struct {
	TemplateID *int64
	DateFrom   *time.Time
	DateTo     *time.Time
	Status     *taskdomain.Status
}

type CreateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
}

type UpdateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	// UnlinkTemplate — обнулить template_id.
	UnlinkTemplate bool
	// TemplateID — привязать к шаблону (игнорируется при UnlinkTemplate).
	TemplateID *int64
	// ClearScheduledDate — обнулить scheduled_date.
	ClearScheduledDate bool
	// ScheduledDate — календарный день (дата по UTC).
	ScheduledDate *time.Time
}
