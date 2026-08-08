package services

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Push fan-out (#1164). Before this, one account had one token, so a second
// device stole delivery from the first. These cover the delivery contract
// without touching FCM: the send is injected.

func TestFanOutPush_ReachesEveryDevice(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	a, b, c := repeatToken("a"), repeatToken("b"), repeatToken("c")
	for id, tok := range map[string]string{"d-a": a, "d-b": b, "d-c": c} {
		require.NoError(t, SetDeviceToken(db, uid, id, "customer", tok))
	}

	var sent []string
	require.NoError(t, fanOutPush(db, uid, func(token string) error {
		sent = append(sent, token)
		return nil
	}))

	require.ElementsMatch(t, []string{a, b, c}, sent)
}

func TestFanOutPush_NoDevicesIsNotAnError(t *testing.T) {
	db := setupDeviceDB(t)

	calls := 0
	require.NoError(t, fanOutPush(db, uuid.New(), func(string) error {
		calls++
		return nil
	}))
	require.Zero(t, calls)
}

// A dead token on one device must neither fail the send nor cost the user
// delivery on their other devices.
func TestFanOutPush_PrunesOnlyTheDeadToken(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	good, dead := repeatToken("g"), repeatToken("z")
	require.NoError(t, SetDeviceToken(db, uid, "good", "customer", good))
	require.NoError(t, SetDeviceToken(db, uid, "dead", "customer", dead))

	err := fanOutPush(db, uid, func(token string) error {
		if token == dead {
			return fmt.Errorf("%w: gone", ErrPushTokenInvalid)
		}
		return nil
	})
	require.NoError(t, err, "a dead sibling token must not fail the whole send")

	remaining, err := ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	require.Equal(t, []string{good}, remaining)
}

// A transient failure has to surface so the caller can retry, but only after
// every other device has been attempted.
func TestFanOutPush_TransientErrorSurfacesAfterAllAttempts(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	a, b := repeatToken("a"), repeatToken("b")
	require.NoError(t, SetDeviceToken(db, uid, "d-a", "customer", a))
	require.NoError(t, SetDeviceToken(db, uid, "d-b", "customer", b))

	boom := errors.New("fcm unavailable")
	var attempted []string
	err := fanOutPush(db, uid, func(token string) error {
		attempted = append(attempted, token)
		if token == a {
			return boom
		}
		return nil
	})

	require.ErrorIs(t, err, boom)
	sort.Strings(attempted)
	want := []string{a, b}
	sort.Strings(want)
	require.Equal(t, want, attempted, "a failure on one device must not abort the rest")
}

// A transient failure must never be mistaken for a dead token — pruning on a
// blip would cost the user push until the app happened to re-register.
func TestFanOutPush_TransientErrorKeepsTheToken(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	tok := repeatToken("a")
	require.NoError(t, SetDeviceToken(db, uid, "d-a", "customer", tok))

	err := fanOutPush(db, uid, func(string) error { return errors.New("timeout") })
	require.Error(t, err)

	remaining, err := ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	require.Equal(t, []string{tok}, remaining)
}
