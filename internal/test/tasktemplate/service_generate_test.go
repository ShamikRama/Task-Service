package tasktemplate_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	taskdomain "example.com/taskservice/internal/domain/task"
	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
	tasktemplatemocks "example.com/taskservice/internal/mocks/tasktemplate"
	tplusecase "example.com/taskservice/internal/usecase/tasktemplate"
)

func TestService_GenerateAllActive_emptyListDoesNotCallBatch(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	repo.On("List", mock.Anything, tplusecase.ListFilter{ActiveOnly: true}).Return(nil, nil).Once()

	svc := tplusecase.NewService(repo, discardLogger())
	res, err := svc.GenerateAllActive(context.Background(), td(2026, 4, 1), td(2026, 4, 5))
	require.NoError(t, err)
	require.Zero(t, res.TotalDates)
	require.Zero(t, res.GeneratedCount)
	repo.AssertNotCalled(t, "BatchCreateTasks", mock.Anything, mock.Anything)
}

func TestService_GenerateAllActive_idempotentBatchReturnsZeroInserted(t *testing.T) {
	id := int64(1)
	raw := json.RawMessage(`{"interval":1}`)
	params, err := tpldomain.ParsePeriodicityParams(tpldomain.PeriodicityDaily, raw)
	require.NoError(t, err)
	tpl := tpldomain.Template{
		ID:                   id,
		IsActive:             true,
		StartDate:            td(2026, 4, 1),
		PeriodicityParams:    params,
		RawPeriodicityParams: raw,
	}

	repo := tasktemplatemocks.NewRepository(t)
	repo.On("List", mock.Anything, tplusecase.ListFilter{ActiveOnly: true}).Return([]tpldomain.Template{tpl}, nil).Once()
	repo.On("BatchCreateTasks", mock.Anything, mock.MatchedBy(func(tasks []taskdomain.Task) bool {
		return len(tasks) == 3
	})).Return(int64(0), nil).Once()

	svc := tplusecase.NewService(repo, discardLogger())
	res, err := svc.GenerateAllActive(context.Background(), td(2026, 4, 1), td(2026, 4, 3))
	require.NoError(t, err)
	require.Equal(t, 3, res.TotalDates)
	require.Zero(t, res.GeneratedCount)
}

func TestService_GenerateAllActive_continuesWhenOneTemplateBatchFails(t *testing.T) {
	raw := json.RawMessage(`{"interval":1}`)
	p1, err := tpldomain.ParsePeriodicityParams(tpldomain.PeriodicityDaily, raw)
	require.NoError(t, err)
	list := []tpldomain.Template{
		{ID: 1, IsActive: true, StartDate: td(2026, 4, 1), PeriodicityParams: p1, RawPeriodicityParams: raw},
		{ID: 2, IsActive: true, StartDate: td(2026, 4, 1), PeriodicityParams: p1, RawPeriodicityParams: raw},
	}

	repo := tasktemplatemocks.NewRepository(t)
	repo.On("List", mock.Anything, tplusecase.ListFilter{ActiveOnly: true}).Return(list, nil).Once()
	repo.On("BatchCreateTasks", mock.Anything, mock.MatchedBy(func(tasks []taskdomain.Task) bool {
		return len(tasks) == 1 && tasks[0].TemplateID != nil && *tasks[0].TemplateID == 1
	})).Return(int64(0), errors.New("simulated db error")).Once()
	repo.On("BatchCreateTasks", mock.Anything, mock.MatchedBy(func(tasks []taskdomain.Task) bool {
		return len(tasks) == 1 && tasks[0].TemplateID != nil && *tasks[0].TemplateID == 2
	})).Return(int64(1), nil).Once()

	svc := tplusecase.NewService(repo, discardLogger())
	res, err := svc.GenerateAllActive(context.Background(), td(2026, 4, 1), td(2026, 4, 1))
	require.NoError(t, err)
	require.Equal(t, 1, res.TotalDates)
	require.Equal(t, int64(1), res.GeneratedCount)
}
