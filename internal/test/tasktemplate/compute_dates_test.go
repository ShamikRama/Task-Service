package tasktemplate_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
	tplusecase "example.com/taskservice/internal/usecase/tasktemplate"
)

func TestComputeDates_clipsRequestToTemplateStartEnd(t *testing.T) {
	raw := json.RawMessage(`{"interval":1}`)
	params, err := tpldomain.ParsePeriodicityParams(tpldomain.PeriodicityDaily, raw)
	require.NoError(t, err)
	tpl := &tpldomain.Template{
		StartDate:         td(2026, 4, 10),
		EndDate:           ptrDate(td(2026, 4, 20)),
		PeriodicityParams: params,
	}
	got := tplusecase.ComputeDates(tpl, td(2026, 4, 1), td(2026, 4, 30))
	want := []time.Time{
		td(2026, 4, 10), td(2026, 4, 11), td(2026, 4, 12), td(2026, 4, 13), td(2026, 4, 14),
		td(2026, 4, 15), td(2026, 4, 16), td(2026, 4, 17), td(2026, 4, 18), td(2026, 4, 19), td(2026, 4, 20),
	}
	require.Len(t, got, len(want))
	for i := range got {
		require.True(t, got[i].Equal(want[i]), "[%d] got %s want %s", i, got[i].Format(time.DateOnly), want[i].Format(time.DateOnly))
	}
}

func TestComputeDates_emptyWhenRequestDisjointFromTemplate(t *testing.T) {
	raw := json.RawMessage(`{"interval":1}`)
	params, err := tpldomain.ParsePeriodicityParams(tpldomain.PeriodicityDaily, raw)
	require.NoError(t, err)
	tpl := &tpldomain.Template{
		StartDate:         td(2026, 5, 1),
		EndDate:           ptrDate(td(2026, 5, 31)),
		PeriodicityParams: params,
	}
	got := tplusecase.ComputeDates(tpl, td(2026, 4, 1), td(2026, 4, 30))
	require.Empty(t, got)
}

func TestComputeDates_endDateOpenEnded(t *testing.T) {
	raw := json.RawMessage(`{"interval":7}`)
	params, err := tpldomain.ParsePeriodicityParams(tpldomain.PeriodicityDaily, raw)
	require.NoError(t, err)
	tpl := &tpldomain.Template{
		StartDate:         td(2026, 4, 1),
		EndDate:           nil,
		PeriodicityParams: params,
	}
	got := tplusecase.ComputeDates(tpl, td(2026, 4, 1), td(2026, 4, 15))
	want := []time.Time{td(2026, 4, 1), td(2026, 4, 8), td(2026, 4, 15)}
	require.Len(t, got, len(want))
	for i := range got {
		require.True(t, got[i].Equal(want[i]), "[%d] got %s want %s", i, got[i].Format(time.DateOnly), want[i].Format(time.DateOnly))
	}
}
