package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

func TestCronJobActivityAcknowledgesRetiredJob(t *testing.T) {
	t.Parallel()

	require.NoError(t, CronJobActivity(t.Context(), "meal-plan-hold-reconcile"))
}

func TestPauseRetiredCronSchedules(t *testing.T) {
	t.Parallel()

	schedules := mocks.NewScheduleClient(t)
	handle := mocks.NewScheduleHandle(t)
	schedules.On(
		"GetHandle",
		mock.Anything,
		"homechef-cron-meal-plan-hold-reconcile",
	).Return(handle).Once()
	handle.On(
		"Pause",
		mock.Anything,
		client.SchedulePauseOptions{
			Note: "retired: escrow day-transfer layer removed in #1106",
		},
	).Return(nil).Once()

	require.NoError(t, pauseRetiredCronSchedules(t.Context(), schedules))
}

func TestPauseRetiredCronSchedulesIgnoresMissingSchedule(t *testing.T) {
	t.Parallel()

	schedules := mocks.NewScheduleClient(t)
	handle := mocks.NewScheduleHandle(t)
	schedules.On("GetHandle", mock.Anything, mock.Anything).Return(handle).Once()
	handle.On("Pause", mock.Anything, mock.Anything).
		Return(serviceerror.NewNotFound("schedule not found")).Once()

	require.NoError(t, pauseRetiredCronSchedules(t.Context(), schedules))
}

func TestPauseRetiredCronSchedulesReturnsUnexpectedError(t *testing.T) {
	t.Parallel()

	schedules := mocks.NewScheduleClient(t)
	handle := mocks.NewScheduleHandle(t)
	schedules.On("GetHandle", mock.Anything, mock.Anything).Return(handle).Once()
	handle.On("Pause", mock.Anything, mock.Anything).
		Return(errors.New("temporal unavailable")).Once()

	err := pauseRetiredCronSchedules(context.Background(), schedules)
	require.ErrorContains(t, err, "pause retired cron schedule")
}
