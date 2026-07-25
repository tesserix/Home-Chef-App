package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// testNotificationPrefix marks a notification as belonging to a sandbox order.
const testNotificationPrefix = "[TEST] "

// TagSubjectForMode prefixes an email subject or push title for a test-mode
// notification.
//
// Test notifications are deliberately NOT suppressed. Their recipients are
// structurally limited to the test-mode viewer allowlist plus the test kitchen
// and its assigned driver, so there is no blast radius — and firing them is the
// only way to verify the FCM and Temporal dispatch path on production, where it
// has broken before.
//
// Idempotent: notification subjects are composed from several helpers, and a
// doubled "[TEST] [TEST]" reads like a bug.
func TagSubjectForMode(mode, subject string) string {
	if !models.IsTestMode(mode) || strings.HasPrefix(subject, testNotificationPrefix) {
		return subject
	}
	return testNotificationPrefix + subject
}
