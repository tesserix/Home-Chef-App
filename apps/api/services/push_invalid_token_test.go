package services

// push_invalid_token_test.go — FCM's permanent token verdicts must be told apart
// from transient failures.
//
// Every 4xx used to surface as one opaque error, so notify-dispatch burned all
// 5 retries on a token that could never work, dead-lettered, and LEFT the dead
// token in users.fcm_token — so every later notification for that user failed
// identically, forever. Seen in production as:
//
//   FCM returned status 400: "The registration token is not a valid FCM
//   registration token"  ->  dead-lettered after 5 deliveries
//
// The bodies below are the real shapes FCM returns (the INVALID_ARGUMENT one is
// copied verbatim from the production log). The narrow rule matters:
// INVALID_ARGUMENT also covers a malformed PAYLOAD, which is our bug and must
// stay retryable rather than silently costing a user their token.

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Verbatim from the homechef-api production log, 2026-07-17T05:29:09Z.
const fcmInvalidRegistrationTokenBody = `{
  "error": {
    "code": 400,
    "message": "The registration token is not a valid FCM registration token",
    "status": "INVALID_ARGUMENT",
    "details": [
      {
        "@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError",
        "errorCode": "INVALID_ARGUMENT"
      },
      {
        "@type": "type.googleapis.com/google.rpc.BadRequest",
        "fieldViolations": [
          {
            "field": "message.token",
            "description": "The registration token is not a valid FCM registration token"
          }
        ]
      }
    ]
  }
}`

const fcmUnregisteredBody = `{
  "error": {
    "code": 404,
    "message": "Requested entity was not found.",
    "status": "NOT_FOUND",
    "details": [
      {
        "@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError",
        "errorCode": "UNREGISTERED"
      }
    ]
  }
}`

// The 403 we hit before the IAM grant — our permission problem, NOT the token's.
const fcmPermissionDeniedBody = `{
  "error": {
    "code": 403,
    "message": "Permission 'cloudmessaging.messages.create' denied on resource",
    "status": "PERMISSION_DENIED",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.ErrorInfo",
        "reason": "IAM_PERMISSION_DENIED"
      }
    ]
  }
}`

// INVALID_ARGUMENT about the PAYLOAD, not the token — our bug; must stay visible.
const fcmBadPayloadBody = `{
  "error": {
    "code": 400,
    "message": "Invalid value at 'message.android.notification.color'",
    "status": "INVALID_ARGUMENT",
    "details": [
      {
        "@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError",
        "errorCode": "INVALID_ARGUMENT"
      }
    ]
  }
}`

func TestIsInvalidTokenResponse_ProductionInvalidRegistrationToken(t *testing.T) {
	assert.True(t,
		isInvalidTokenResponse(http.StatusBadRequest, []byte(fcmInvalidRegistrationTokenBody)),
		"the exact body that dead-lettered in prod must be recognised as a dead token")
}

func TestIsInvalidTokenResponse_Unregistered(t *testing.T) {
	assert.True(t,
		isInvalidTokenResponse(http.StatusNotFound, []byte(fcmUnregisteredBody)),
		"UNREGISTERED is FCM's documented signal to delete the token")
}

// The most important negative: a quota/permission failure is transient or ours.
// Treating it as a dead token would have wiped every user's token during this
// morning's 403 outage.
func TestIsInvalidTokenResponse_PermissionDenied_IsNotATokenProblem(t *testing.T) {
	assert.False(t,
		isInvalidTokenResponse(http.StatusForbidden, []byte(fcmPermissionDeniedBody)),
		"a 403 is OUR IAM problem — pruning tokens over it would destroy good data")
}

func TestIsInvalidTokenResponse_BadPayload_IsNotATokenProblem(t *testing.T) {
	assert.False(t,
		isInvalidTokenResponse(http.StatusBadRequest, []byte(fcmBadPayloadBody)),
		"INVALID_ARGUMENT about the payload is our bug; keep it retryable and loud")
}

func TestIsInvalidTokenResponse_UnparseableBody(t *testing.T) {
	assert.False(t, isInvalidTokenResponse(http.StatusBadRequest, []byte("<html>502</html>")),
		"never prune a token on a body we cannot understand")
	assert.False(t, isInvalidTokenResponse(http.StatusInternalServerError, []byte("")))
}

func TestIsInvalidTokenResponse_ServerErrorsAreRetryable(t *testing.T) {
	// FCM 500/503 — always transient.
	body := `{"error":{"code":503,"status":"UNAVAILABLE","message":"The service is currently unavailable."}}`
	assert.False(t, isInvalidTokenResponse(http.StatusServiceUnavailable, []byte(body)))
}

// A 404 whose body names the registration token but carries no details block.
func TestIsInvalidTokenResponse_NotFoundFallback(t *testing.T) {
	body := `{"error":{"code":404,"status":"NOT_FOUND","message":"The registration token is not registered"}}`
	assert.True(t, isInvalidTokenResponse(http.StatusNotFound, []byte(body)))
}
