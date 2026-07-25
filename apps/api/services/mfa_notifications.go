package services

import (
	"github.com/google/uuid"
)

// Security alerts for two-factor changes.
//
// These are what make 2FA protective rather than decorative. An attacker who
// gets in and disables the second factor, or trusts their own device, is
// invisible unless the real owner is told — so each of those transitions sends
// mail the moment it happens.
//
// They bypass notification preferences. Every other notification type routes
// through ShouldSendForType, and security alerts would otherwise land in the
// "account" category, which users can switch off in settings. Someone who muted
// account email must still find out that their second factor was removed.

// Security-alert notification types. IsSecurityAlert must list every one.
const (
	NotifTypeMFAEnabled        = "security_mfa_enabled"
	NotifTypeMFADisabled       = "security_mfa_disabled"
	NotifTypeMFADeviceTrusted  = "security_device_trusted"
	NotifTypeMFADeviceRevoked  = "security_device_revoked"
	NotifTypeMFABackupCodesLow = "security_backup_codes_low"
	NotifTypeMFAResetPending   = "security_mfa_reset_pending"
	NotifTypeMFAResetCancelled = "security_mfa_reset_cancelled"
	NotifTypeMFAResetApplied   = "security_mfa_reset_applied"
)

// IsSecurityAlert reports whether a notification type must ignore the user's
// preferences. Deliberately an explicit allowlist rather than a prefix match, so
// adding a type is a decision someone makes rather than something a name causes.
func IsSecurityAlert(notifType string) bool {
	switch notifType {
	case NotifTypeMFAEnabled,
		NotifTypeMFADisabled,
		NotifTypeMFADeviceTrusted,
		NotifTypeMFADeviceRevoked,
		NotifTypeMFABackupCodesLow,
		NotifTypeMFAResetPending,
		NotifTypeMFAResetCancelled,
		NotifTypeMFAResetApplied:
		return true
	default:
		return false
	}
}

// securityCopy is the user-facing wording per type.
//
// Each message names the action and how to respond. "Two-factor was turned off"
// is useless on its own; the person reading it needs to know what to do if it
// wasn't them.
var securityCopy = map[string]struct{ title, message string }{
	NotifTypeMFAEnabled: {
		"Two-factor is on",
		"You turned on two-factor authentication. You'll be asked for a code when you sign in on a new device. Keep your backup codes somewhere safe.",
	},
	NotifTypeMFADisabled: {
		"Two-factor was turned off",
		"Two-factor authentication was switched off on your account, and any remembered devices were forgotten. If this wasn't you, change your password and turn two-factor back on now.",
	},
	NotifTypeMFADeviceTrusted: {
		"New device remembered",
		"You chose to skip the verification code on a new device. If this wasn't you, open Security settings and revoke it — that device can sign in without a code until you do.",
	},
	NotifTypeMFADeviceRevoked: {
		"A device was removed",
		"A remembered device was removed from your account. It will need a verification code to sign in again.",
	},
	NotifTypeMFABackupCodesLow: {
		"You're low on backup codes",
		"You have two or fewer backup codes left. Generate a new set in Security settings so you don't get locked out if you lose access to your email or phone.",
	},
	NotifTypeMFAResetPending: {
		"Two-factor reset requested",
		"Someone from our support team requested that two-factor be removed from your account. It takes effect in 24 hours. If you didn't ask for this, cancel it now from Security settings and change your password.",
	},
	NotifTypeMFAResetCancelled: {
		"Two-factor reset cancelled",
		"The pending two-factor reset on your account was cancelled. Nothing changed.",
	},
	NotifTypeMFAResetApplied: {
		"Two-factor was removed",
		"Two-factor authentication was removed from your account by our support team. If you didn't request this, change your password and turn two-factor back on immediately.",
	},
}

// NotifySecurityEvent sends one security alert as email and push.
//
// Best-effort by design: the state change it describes has already been
// committed, so failing the caller now would leave a user staring at an error
// after successfully changing a security setting. PublishNotification already
// logs its own failures.
func NotifySecurityEvent(notifType string, userID uuid.UUID) {
	copy, ok := securityCopy[notifType]
	if !ok {
		return
	}
	for _, channel := range []string{SubjectNotificationEmail, SubjectNotificationPush} {
		_ = PublishNotification(NotificationEvent{
			UserID:  userID,
			Type:    channel,
			Title:   copy.title,
			Message: copy.message,
			Data:    map[string]any{"type": notifType},
		})
	}
}

// notifTypeForSubject maps an auth.* event subject to its user-facing alert, so
// one PublishMFAEvent call both records the event and tells the user.
var notifTypeForSubject = map[string]string{
	SubjectMFAEnabled:        NotifTypeMFAEnabled,
	SubjectMFADisabled:       NotifTypeMFADisabled,
	SubjectMFADeviceTrusted:  NotifTypeMFADeviceTrusted,
	SubjectMFADeviceRevoked:  NotifTypeMFADeviceRevoked,
	SubjectMFABackupCodesLow: NotifTypeMFABackupCodesLow,
}
