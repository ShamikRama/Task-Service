package tasktemplate_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	taskdomain "example.com/taskservice/internal/domain/task"
	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
	tasktemplatemocks "example.com/taskservice/internal/mocks/tasktemplate"
	tplusecase "example.com/taskservice/internal/usecase/tasktemplate"
)

func TestService_GetByID_negativeIDDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := svc.GetByID(context.Background(), 0)
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestService_GetByID_delegatesToRepository(t *testing.T) {
	want := &tpldomain.Template{ID: 5, Title: "x"}
	repo := tasktemplatemocks.NewRepository(t)
	repo.On("GetByID", mock.Anything, int64(5)).Return(want, nil).Once()

	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	got, err := svc.GetByID(context.Background(), 5)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestService_CreateWithGeneration_emptyTitleDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.CreateWithGeneration(context.Background(), tplusecase.CreateInput{
		Title:             "   ",
		PeriodicityType:   tpldomain.PeriodicityDaily,
		PeriodicityParams: []byte(`{"interval":1}`),
		StartDate:         start,
		IsActive:          true,
	}, start, start)
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "CreateWithTasks", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_CreateWithGeneration_delegatesToRepository(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	created := &tpldomain.Template{ID: 9, Title: "T", StartDate: start}

	repo := tasktemplatemocks.NewRepository(t)
	repo.On("CreateWithTasks", mock.Anything, mock.MatchedBy(func(m *tpldomain.Template) bool {
		return m.Title == "T" && m.StartDate.Equal(start)
	}), mock.MatchedBy(func(tasks []taskdomain.Task) bool {
		return len(tasks) == 0
	})).Return(created, int64(0), nil).Once()

	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	got, _, err := svc.CreateWithGeneration(context.Background(), tplusecase.CreateInput{
		Title:             "T",
		PeriodicityType:   tpldomain.PeriodicityDaily,
		PeriodicityParams: []byte(`{"interval":1}`),
		StartDate:         start,
		IsActive:          false,
	}, start, start)
	require.NoError(t, err)
	require.Equal(t, created, got)
}

func TestService_List_delegatesToRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	filter := tplusecase.ListFilter{ActiveOnly: true}
	repo.On("List", mock.Anything, filter).Return([]tpldomain.Template{{ID: 1}}, nil).Once()

	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	got, err := svc.List(context.Background(), filter)
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestService_Delete_negativeIDDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	err := svc.Delete(context.Background(), -1)
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestService_Update_negativeIDDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := svc.Update(context.Background(), 0, tplusecase.UpdateInput{
		Title:             "x",
		PeriodicityType:   tpldomain.PeriodicityDaily,
		PeriodicityParams: []byte(`{"interval":1}`),
		StartDate:         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		IsActive:          true,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func validCreateInput() tplusecase.CreateInput {
	return tplusecase.CreateInput{
		Title:             "Title",
		PeriodicityType:   tpldomain.PeriodicityDaily,
		PeriodicityParams: []byte(`{"interval":1}`),
		StartDate:         time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		IsActive:          true,
	}
}

func validUpdateInput() tplusecase.UpdateInput {
	return tplusecase.UpdateInput{
		Title:             "Title",
		PeriodicityType:   tpldomain.PeriodicityDaily,
		PeriodicityParams: []byte(`{"interval":1}`),
		StartDate:         time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		IsActive:          true,
	}
}

func TestService_CreateWithGeneration_invalidPeriodicityTypeDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	in := validCreateInput()
	in.PeriodicityType = tpldomain.PeriodicityType("weekly")
	_, _, err := svc.CreateWithGeneration(context.Background(), in, in.StartDate, in.StartDate)
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "CreateWithTasks", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_CreateWithGeneration_zeroStartDateDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	in := validCreateInput()
	in.StartDate = time.Time{}
	_, _, err := svc.CreateWithGeneration(context.Background(), in, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "CreateWithTasks", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_CreateWithGeneration_endDateBeforeStartDateDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	start := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	in := validCreateInput()
	in.StartDate = start
	in.EndDate = &end
	_, _, err := svc.CreateWithGeneration(context.Background(), in, start, start)
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "CreateWithTasks", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_CreateWithGeneration_endDateOnOrAfterStartDate_callsRepository(t *testing.T) {
	start := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	same := start
	later := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	for _, name := range []string{"end_equals_start", "end_after_start"} {
		t.Run(name, func(t *testing.T) {
			repo := tasktemplatemocks.NewRepository(t)
			repo.On("CreateWithTasks", mock.Anything, mock.Anything, mock.MatchedBy(func(tasks []taskdomain.Task) bool {
				return len(tasks) == 0
			})).Return(&tpldomain.Template{ID: 1}, int64(0), nil).Once()

			svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
			in := validCreateInput()
			in.StartDate = start
			in.IsActive = false
			if name == "end_equals_start" {
				in.EndDate = &same
			} else {
				in.EndDate = &later
			}
			_, _, err := svc.CreateWithGeneration(context.Background(), in, start, start)
			require.NoError(t, err)
		})
	}
}

func TestService_Update_invalidPeriodicityTypeDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	in := validUpdateInput()
	in.PeriodicityType = tpldomain.PeriodicityType("")
	_, err := svc.Update(context.Background(), 1, in)
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_Update_zeroStartDateDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	in := validUpdateInput()
	in.StartDate = time.Time{}
	_, err := svc.Update(context.Background(), 42, in)
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_Update_endDateBeforeStartDateDoesNotCallRepository(t *testing.T) {
	repo := tasktemplatemocks.NewRepository(t)
	svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	start := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	in := validUpdateInput()
	in.StartDate = start
	in.EndDate = &end
	_, err := svc.Update(context.Background(), 42, in)
	require.Error(t, err)
	require.ErrorIs(t, err, tplusecase.ErrInvalidInput)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_Update_endDateOnOrAfterStartDate_callsRepository(t *testing.T) {
	start := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	same := start
	later := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	for _, name := range []string{"end_equals_start", "end_after_start"} {
		t.Run(name, func(t *testing.T) {
			repo := tasktemplatemocks.NewRepository(t)
			repo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(&tpldomain.Template{ID: 7}, nil).Once()

			svc := tplusecase.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
			in := validUpdateInput()
			in.StartDate = start
			if name == "end_equals_start" {
				in.EndDate = &same
			} else {
				in.EndDate = &later
			}
			_, err := svc.Update(context.Background(), 7, in)
			require.NoError(t, err)
		})
	}
}
