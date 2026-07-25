package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// The claim shape is the fiddly part — the verified number is nested two levels
// inside an untyped map, and reading the wrong key means either rejecting valid
// enrollments or, worse, trusting a number Google never verified.
func TestPhoneFromFirebaseClaims_PrefersVerifiedIdentity(t *testing.T) {
	claims := map[string]any{
		"phone_number": "+919999999999", // convenience claim
		"firebase": map[string]any{
			"identities": map[string]any{
				"phone": []any{"+919876543210"}, // what Google actually verified
			},
		},
	}
	require.Equal(t, "+919876543210", PhoneFromFirebaseClaims(claims))
}

func TestPhoneFromFirebaseClaims_FallsBackToPhoneNumber(t *testing.T) {
	require.Equal(t, "+919876543210", PhoneFromFirebaseClaims(map[string]any{
		"phone_number": "+91 98765-43210",
	}))
}

// A token from an email or Google sign-in carries no phone. Returning "" here is
// what makes the handler refuse rather than enrol an empty number.
func TestPhoneFromFirebaseClaims_NoPhone(t *testing.T) {
	for _, claims := range []map[string]any{
		{},
		{"firebase": map[string]any{"identities": map[string]any{"email": []any{"a@b.com"}}}},
		{"firebase": map[string]any{"identities": map[string]any{"phone": []any{}}}},
		{"firebase": "not-a-map"},
		{"phone_number": ""},
	} {
		require.Equal(t, "", PhoneFromFirebaseClaims(claims), "claims: %v", claims)
	}
}

func TestNormalizeE164(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"+91 98765 43210", "+919876543210"},
		{"+91-98765-43210", "+919876543210"},
		{" +919876543210 ", "+919876543210"},
		{"(+91) 98765 43210", "+919876543210"},
		{"", ""},
	} {
		require.Equal(t, tc.want, NormalizeE164(tc.in), "input %q", tc.in)
	}
}

// An empty token must never reach Firebase — and must never be treated as valid.
func TestVerifyFirebasePhoneToken_RejectsEmpty(t *testing.T) {
	_, err := VerifyFirebasePhoneToken(context.Background(), "   ", "uid-1")
	require.ErrorIs(t, err, ErrPhoneTokenInvalid)
}

// With no Firebase configured the phone channel is unavailable — it must fail
// closed rather than letting an unverified number through.
func TestVerifyFirebasePhoneToken_UnconfiguredFailsClosed(t *testing.T) {
	prev := appConfigForFirebase
	SetFirebaseConfigProvider(func() FirebaseConfig { return FirebaseConfig{} })
	t.Cleanup(func() { SetFirebaseConfigProvider(prev) })

	_, err := VerifyFirebasePhoneToken(context.Background(), "some.jwt.token", "uid-1")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrPhoneTokenNoPhone, "must not read as a verified-but-phoneless token")
}
