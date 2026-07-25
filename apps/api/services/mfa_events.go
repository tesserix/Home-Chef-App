package services

import (
	"log"

	"github.com/google/uuid"
)

// Security events for two-factor changes.
//
// Fan-out only. These drive the notifications that make 2FA protective — "a new
// device signed in", "two-factor was turned off" — and the audit trail. None of
// them is on the critical path, so publishing is best-effort: a NATS outage must
// never be the reason a user cannot finish signing in.
//
// The login code itself is deliberately NOT published this way. Returning
// success for a code that a downed consumer never delivered would leave the user
// waiting for a message that is not coming.

const (
	SubjectMFAEnabled        = "auth.mfa.enabled"
	SubjectMFADisabled       = "auth.mfa.disabled"
	SubjectMFADeviceTrusted  = "auth.device.trusted"
	SubjectMFADeviceRevoked  = "auth.device.revoked"
	SubjectMFABackupCodesLow = "auth.mfa.backup_codes_low"
	SubjectMFAFailedBurst    = "auth.mfa.failed_burst"
)

// PublishMFAEvent emits a security event, swallowing failures by design.
//
// The caller has already committed the state change the event describes; failing
// the request now would leave the user with two-factor on and an error on screen.
// The failure is logged so the gap is visible in operations rather than silent.
func PublishMFAEvent(subject string, userID uuid.UUID) {
	if err := PublishEvent(subject, subject, userID, map[string]any{
		"userId": userID.String(),
	}); err != nil {
		log.Printf("mfa-event: publish %s for user=%s failed: %v", subject, userID, err)
	}
}
