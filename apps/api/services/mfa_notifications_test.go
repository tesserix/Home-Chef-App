package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Security alerts must ignore notification preferences. They land in the
// "account" category, which users can switch off in settings — so without the
// bypass, someone who muted account email would never learn that their second
// factor was removed. That is the failure mode the alerts exist to prevent.
func TestIsSecurityAlert_CoversEverySecurityType(t *testing.T) {
	for _, notifType := range []string{
		NotifTypeMFAEnabled,
		NotifTypeMFADisabled,
		NotifTypeMFADeviceTrusted,
		NotifTypeMFADeviceRevoked,
		NotifTypeMFABackupCodesLow,
		NotifTypeMFAResetPending,
		NotifTypeMFAResetCancelled,
		NotifTypeMFAResetApplied,
	} {
		require.True(t, IsSecurityAlert(notifType), "%s must bypass preferences", notifType)
	}
}

func TestIsSecurityAlert_DoesNotOverreach(t *testing.T) {
	// Ordinary notifications must stay opt-outable; a prefix match or a
	// too-broad default here would quietly make marketing unmutable.
	for _, notifType := range []string{
		"order_confirmation", "promo", "marketing", "chef_new_order",
		"welcome", "delivery_assigned", "", "security", "security_",
	} {
		require.False(t, IsSecurityAlert(notifType), "%q must respect preferences", notifType)
	}
}

// Every security type needs copy; a type without it silently sends nothing,
// which is indistinguishable from the alert never having been wired up.
func TestSecurityCopy_ExistsForEverySecurityType(t *testing.T) {
	for _, notifType := range []string{
		NotifTypeMFAEnabled,
		NotifTypeMFADisabled,
		NotifTypeMFADeviceTrusted,
		NotifTypeMFADeviceRevoked,
		NotifTypeMFABackupCodesLow,
		NotifTypeMFAResetPending,
		NotifTypeMFAResetCancelled,
		NotifTypeMFAResetApplied,
	} {
		copy, ok := securityCopy[notifType]
		require.True(t, ok, "%s has no copy", notifType)
		require.NotEmpty(t, copy.title, "%s has no title", notifType)
		require.NotEmpty(t, copy.message, "%s has no message", notifType)
	}
}

// An alert that only says "something changed" is not actionable. The ones that
// signal a possible compromise must both raise the "was this you?" question and
// name a concrete next step. Asserted as a property rather than exact wording,
// so the copy can be reworded without a false failure.
func TestSecurityCopy_CompromiseAlertsAreActionable(t *testing.T) {
	doubt := []string{"wasn't you", "didn't ask for this", "didn't request this"}
	remedy := []string{"change your password", "revoke", "cancel it now", "turn two-factor back on"}

	containsAny := func(s string, options []string) bool {
		for _, o := range options {
			if strings.Contains(s, o) {
				return true
			}
		}
		return false
	}

	for _, notifType := range []string{
		NotifTypeMFADisabled,
		NotifTypeMFADeviceTrusted,
		NotifTypeMFAResetPending,
		NotifTypeMFAResetApplied,
	} {
		msg := securityCopy[notifType].message
		require.True(t, containsAny(msg, doubt),
			"%s must prompt the reader to question whether it was them: %q", notifType, msg)
		require.True(t, containsAny(msg, remedy),
			"%s must name a concrete next step: %q", notifType, msg)
	}
}

// Each auth.* subject that has user-visible meaning must map to an alert,
// otherwise PublishMFAEvent records the event and tells nobody.
func TestSubjectToNotifTypeMapping(t *testing.T) {
	for subject, want := range map[string]string{
		SubjectMFAEnabled:        NotifTypeMFAEnabled,
		SubjectMFADisabled:       NotifTypeMFADisabled,
		SubjectMFADeviceTrusted:  NotifTypeMFADeviceTrusted,
		SubjectMFADeviceRevoked:  NotifTypeMFADeviceRevoked,
		SubjectMFABackupCodesLow: NotifTypeMFABackupCodesLow,
	} {
		got, ok := notifTypeForSubject[subject]
		require.True(t, ok, "subject %s has no alert mapping", subject)
		require.Equal(t, want, got)
		require.True(t, IsSecurityAlert(got))
	}
}
