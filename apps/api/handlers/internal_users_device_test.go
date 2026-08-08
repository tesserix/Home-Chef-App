package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// captureSightings replaces the off-request-path notifier with a synchronous
// recorder, so the test observes what the handler decided rather than racing
// the goroutine that normally carries it.
func captureSightings(t *testing.T) *[]services.LoginSighting {
	t.Helper()
	var got []services.LoginSighting
	prev := loginDeviceNotifier
	loginDeviceNotifier = func(_ *gorm.DB, _ models.User, in services.LoginSighting) {
		got = append(got, in)
	}
	t.Cleanup(func() { loginDeviceNotifier = prev })
	return &got
}

func upsertBody(deviceID string) map[string]any {
	return map[string]any{
		"gip_uid": "uid-device", "gip_tenant_id": "t", "gip_provider": "google",
		"auth_pool": "customer", "email": "asha@example.com", "role": "customer",
		"email_verified": true,
		"device_id":      deviceID, "platform": "ios", "device_label": "iPhone 17",
		"app_version": "1.4.0", "ip": "49.207.1.1",
	}
}

func TestUpsert_ForwardsTheDeviceSighting(t *testing.T) {
	h := NewInternalUsersHandler(setupDB(t))
	got := captureSightings(t)

	require.Equal(t, http.StatusOK, postUpsert(t, h, upsertBody("device-a")).Code)

	require.Len(t, *got, 1)
	in := (*got)[0]
	require.Equal(t, "device-a", in.DeviceID)
	require.Equal(t, "ios", in.Platform)
	require.Equal(t, "iPhone 17", in.Label)
	require.Equal(t, "1.4.0", in.AppVersion)
	require.Equal(t, "49.207.1.1", in.IP)
	require.Equal(t, "customer", in.App)
}

// The sign-in that creates the account must be flagged, so the user isn't
// emailed about the device they just registered on; the next one must not be.
func TestUpsert_FlagsOnlyTheAccountCreatingSignIn(t *testing.T) {
	h := NewInternalUsersHandler(setupDB(t))
	got := captureSightings(t)

	require.Equal(t, http.StatusOK, postUpsert(t, h, upsertBody("device-a")).Code)
	require.Equal(t, http.StatusOK, postUpsert(t, h, upsertBody("device-b")).Code)

	require.Len(t, *got, 2)
	require.True(t, (*got)[0].FirstLogin, "signup")
	require.False(t, (*got)[1].FirstLogin, "returning login on a new device")
}

// A restorable deleted account never reaches a live session, so recording a
// device against it would be noise at best.
func TestUpsert_DoesNotNoteADeviceForARestorableAccount(t *testing.T) {
	withUpsertRestoreKey(t)
	db := setupDB(t)
	h := NewInternalUsersHandler(db)
	require.Equal(t, http.StatusOK, postUpsert(t, h, upsertBody("device-a")).Code)
	require.NoError(t, db.Exec(
		`UPDATE users SET deleted_at = CURRENT_TIMESTAMP, purge_after = DATETIME('now', '+10 days')`).Error)

	got := captureSightings(t)
	require.Equal(t, http.StatusOK, postUpsert(t, h, upsertBody("device-a")).Code)

	require.Empty(t, *got)
}
