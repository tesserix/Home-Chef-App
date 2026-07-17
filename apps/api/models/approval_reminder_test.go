package models

// approval_reminder_test.go — #697. The reminder cadence.
//
// The rule, as specified: a chef can bump an unattended request 24h after
// raising it, then 24h after each of the next two bumps. From the 4th bump on —
// once escalated — the wait drops to 6h so they can keep expediting. Three bumps
// means escalated.
//
// This is a tool for the CHEF to expedite their own request. It is never a mark
// against them, and nothing may penalise a chef for using it.
//
// These pin the CADENCE rather than the plumbing, because the cadence is the
// part that is easy to get subtly wrong (base off creation instead of the last
// bump, off-by-one on the threshold, escalating at the wrong count).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func pendingReq(created time.Time) *ApprovalRequest {
	return &ApprovalRequest{Status: ApprovalPending, CreatedAt: created}
}

func TestReminderCooldownFor_FirstThreeAreDaily_ThenSixHourly(t *testing.T) {
	for _, n := range []int{1, 2, 3} {
		require.Equal(t, 24*time.Hour, ReminderCooldownFor(n),
			"bump #%d is one of the first three — a day apart", n)
	}
	for _, n := range []int{4, 5, 12} {
		require.Equal(t, 6*time.Hour, ReminderCooldownFor(n),
			"bump #%d is past escalation — a chef expediting can keep pushing without "+
				"waiting another full day", n)
	}
	// A caller asking about a never-reminded request is asking about bump #1.
	require.Equal(t, 24*time.Hour, ReminderCooldownFor(0))
}

// The first bump is timed off when the request was RAISED.
func TestFirstReminder_UnlocksTwentyFourHoursAfterTheRequestWasRaised(t *testing.T) {
	created := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	r := pendingReq(created)

	require.False(t, r.CanRemindAt(created), "cannot bump the instant it is raised")
	require.False(t, r.CanRemindAt(created.Add(23*time.Hour+59*time.Minute)),
		"still inside the 24h window")
	require.True(t, r.CanRemindAt(created.Add(24*time.Hour)),
		"exactly 24h is allowed — the boundary is inclusive, not one tick late")
	require.True(t, r.CanRemindAt(created.Add(48*time.Hour)))
}

// The 2nd and 3rd are timed off the LAST BUMP, not off creation. Basing them on
// creation would make every bump after the first instantly available — the whole
// cooldown would collapse.
func TestSecondAndThirdReminders_AreTimedOffTheLastBump_NotCreation(t *testing.T) {
	created := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	firstBump := created.Add(30 * time.Hour) // chef waited longer than the minimum

	r := pendingReq(created)
	r.ReminderCount = 1
	r.LastRemindedAt = &firstBump

	require.False(t, r.CanRemindAt(firstBump.Add(23*time.Hour)),
		"only 23h since the last bump — even though it is 53h since the request was raised")
	require.True(t, r.CanRemindAt(firstBump.Add(24*time.Hour)))

	secondBump := firstBump.Add(24 * time.Hour)
	r.ReminderCount = 2
	r.LastRemindedAt = &secondBump
	require.False(t, r.CanRemindAt(secondBump.Add(23*time.Hour)))
	require.True(t, r.CanRemindAt(secondBump.Add(24*time.Hour)), "the 3rd bump is still daily")
}

// The 4th is the first at the tightened cadence.
func TestFourthReminder_UnlocksAfterSixHours(t *testing.T) {
	created := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	thirdBump := created.Add(72 * time.Hour)

	r := pendingReq(created)
	r.ReminderCount = 3 // escalated
	r.LastRemindedAt = &thirdBump

	require.False(t, r.CanRemindAt(thirdBump.Add(5*time.Hour+59*time.Minute)))
	require.True(t, r.CanRemindAt(thirdBump.Add(6*time.Hour)),
		"once escalated, a chef trying to expedite should not have to wait another full day")
}

func TestEscalation_AtTheThirdReminder(t *testing.T) {
	r := pendingReq(time.Now())
	for _, n := range []int{0, 1, 2} {
		r.ReminderCount = n
		require.False(t, r.IsEscalated(), "%d bumps is a nudge, not an escalation", n)
	}
	for _, n := range []int{3, 4, 9} {
		r.ReminderCount = n
		require.True(t, r.IsEscalated(), "%d bumps means it has been ignored for days", n)
	}
}

// Bumping a decided request is meaningless — it blocks nobody. Refuse rather
// than accept-and-ignore, so the app can hide the button instead of offering
// one that does nothing.
func TestDecidedRequests_RefuseReminders(t *testing.T) {
	long := time.Now().Add(-30 * 24 * time.Hour)
	for _, s := range []ApprovalRequestStatus{ApprovalApproved, ApprovalRejected, ApprovalCancelled} {
		r := &ApprovalRequest{Status: s, CreatedAt: long}
		require.False(t, r.AcceptsReminders(), "%s is decided", s)
		require.False(t, r.CanRemindAt(time.Now()),
			"%s: no cooldown should ever make a decided request remindable", s)
	}
}

// info_requested is NOT decided: the chef answers, and the ball is back with the
// admin. That is exactly when it can go quiet again.
func TestInfoRequested_StaysRemindable(t *testing.T) {
	created := time.Now().Add(-48 * time.Hour)
	r := &ApprovalRequest{Status: ApprovalInfoRequested, CreatedAt: created}
	require.True(t, r.AcceptsReminders())
	require.True(t, r.CanRemindAt(time.Now()))
}

// The computed fields the clients render.
func TestPopulateReminderFields(t *testing.T) {
	created := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	ready := pendingReq(created)
	ready.PopulateReminderFields(created.Add(25 * time.Hour))
	require.True(t, ready.CanRemind)
	require.NotNil(t, ready.NextRemindAt)
	require.Equal(t, created.Add(24*time.Hour), *ready.NextRemindAt,
		"the app renders a countdown to this, so it must be the real unlock time")

	waiting := pendingReq(created)
	waiting.PopulateReminderFields(created.Add(time.Hour))
	require.False(t, waiting.CanRemind)
	require.NotNil(t, waiting.NextRemindAt, "still send the target so the app can count down to it")

	done := &ApprovalRequest{Status: ApprovalApproved, CreatedAt: created}
	done.PopulateReminderFields(created.Add(90 * 24 * time.Hour))
	require.False(t, done.CanRemind)
	require.Nil(t, done.NextRemindAt, "nothing to count down to on a decided request")
}
