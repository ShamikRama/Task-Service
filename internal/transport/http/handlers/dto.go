package handlers

import (
	"encoding/json"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
}

type taskUpdateDTO struct {
	Title              string            `json:"title"`
	Description        string            `json:"description"`
	Status             taskdomain.Status `json:"status"`
	UnlinkTemplate     bool              `json:"unlink_template"`
	TemplateID         *int64            `json:"template_id"`
	ClearScheduledDate bool              `json:"clear_scheduled_date"`
	ScheduledDate      *string           `json:"scheduled_date"`
}

type taskDTO struct {
	ID            int64             `json:"id"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Status        taskdomain.Status `json:"status"`
	TemplateID    *int64            `json:"template_id,omitempty"`
	ScheduledDate *string           `json:"scheduled_date,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type taskExpandedDTO struct {
	taskDTO
	Template *templateDTO `json:"template,omitempty"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		TemplateID:  task.TemplateID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
	if task.ScheduledDate != nil {
		formatted := task.ScheduledDate.Format(time.DateOnly)
		dto.ScheduledDate = &formatted
	}
	return dto
}

func newTaskExpandedDTO(task *taskdomain.Task, tpl *templateDTO) taskExpandedDTO {
	return taskExpandedDTO{
		taskDTO:  newTaskDTO(task),
		Template: tpl,
	}
}

type createTemplateDTO struct {
	Title             string                    `json:"title"`
	Description       string                    `json:"description"`
	PeriodicityType   tpldomain.PeriodicityType `json:"periodicity_type"`
	PeriodicityParams json.RawMessage           `json:"periodicity_params"`
	StartDate         string                    `json:"start_date"`
	EndDate           *string                   `json:"end_date,omitempty"`
	IsActive          *bool                     `json:"is_active,omitempty"`
}

type updateTemplateDTO struct {
	Title             string                    `json:"title"`
	Description       string                    `json:"description"`
	PeriodicityType   tpldomain.PeriodicityType `json:"periodicity_type"`
	PeriodicityParams json.RawMessage           `json:"periodicity_params"`
	StartDate         string                    `json:"start_date"`
	EndDate           *string                   `json:"end_date,omitempty"`
	IsActive          *bool                     `json:"is_active,omitempty"`
}

type templateDTO struct {
	ID                int64                     `json:"id"`
	Title             string                    `json:"title"`
	Description       string                    `json:"description"`
	PeriodicityType   tpldomain.PeriodicityType `json:"periodicity_type"`
	PeriodicityParams json.RawMessage           `json:"periodicity_params"`
	StartDate         string                    `json:"start_date"`
	EndDate           *string                   `json:"end_date,omitempty"`
	IsActive          bool                      `json:"is_active"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

func newTemplateDTO(t *tpldomain.Template, rawParams json.RawMessage) templateDTO {
	dto := templateDTO{
		ID:                t.ID,
		Title:             t.Title,
		Description:       t.Description,
		PeriodicityType:   t.PeriodicityType,
		PeriodicityParams: rawParams,
		StartDate:         t.StartDate.Format(time.DateOnly),
		IsActive:          t.IsActive,
		CreatedAt:         t.CreatedAt,
		UpdatedAt:         t.UpdatedAt,
	}
	if t.EndDate != nil {
		formatted := t.EndDate.Format(time.DateOnly)
		dto.EndDate = &formatted
	}
	return dto
}

type generateResponseDTO struct {
	TotalDates     int   `json:"total_dates"`
	GeneratedCount int64 `json:"generated_count"`
	SkippedCount   int64 `json:"skipped_count"`
}

type createTemplateWithGenerateDTO struct {
	createTemplateDTO
	GenerateFrom string `json:"generate_from"`
	GenerateTo   string `json:"generate_to"`
}

type createTemplateWithGenerateResponseDTO struct {
	Template   templateDTO         `json:"template"`
	Generation generateResponseDTO `json:"generation"`
}
