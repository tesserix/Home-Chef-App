package services

// gateway_idempotency_test.go — #574. Stable idempotency keys are what stop a
// timeout-AFTER-success retry from issuing a SECOND real refund or transfer.
//
// The per-endpoint header tests went with the Razorpay client in #1086; Cashfree's
// own header and refund_id wiring is covered in cashfree_test.go. What must stay
// pinned here is the key algebra those paths depend on: deterministic for the same
// logical operation, distinct across different ones, and normalizing into the
// charset/length window BOTH gateways accept (Cashfree refund_id is 3–40
// alphanumeric, the tightest of them).

import (
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// gatewayKeyCharset is what a normalized key must satisfy (hex is a strict subset).
var gatewayKeyCharset = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestNormalizeIdempotencyKey_ValidForBothEndpoints(t *testing.T) {
	// Even a long, colon-laden logical key normalizes to a charset+length-valid token.
	logical := "refund:" + uuid.New().String() + ":line:" + uuid.New().String()
	got := normalizeIdempotencyKey(logical)
	require.Regexp(t, gatewayKeyCharset, got)
	require.GreaterOrEqual(t, len(got), 10)
	require.LessOrEqual(t, len(got), 36)
	// Deterministic: same logical input → same normalized key (required for retry dedup).
	require.Equal(t, got, normalizeIdempotencyKey(logical))
	// Distinct logical inputs → distinct normalized keys.
	require.NotEqual(t, got, normalizeIdempotencyKey(logical+"x"))
}

func TestGatewayIdempotencyKeys_StableAndDistinct(t *testing.T) {
	o1, o2 := uuid.New(), uuid.New()
	line1, line2 := uuid.New(), uuid.New()

	// Deterministic: same inputs → same logical key.
	require.Equal(t, RefundFullIdempotencyKey(o1), RefundFullIdempotencyKey(o1))
	require.Equal(t, RefundLineIdempotencyKey(o1, line1), RefundLineIdempotencyKey(o1, line1))
	require.Equal(t, RefundPartialIdempotencyKey(o1, 100), RefundPartialIdempotencyKey(o1, 100))
	require.Equal(t, TopupIdempotencyKey(o1, 0, ")acc_x"), TopupIdempotencyKey(o1, 0, ")acc_x"))
	require.Equal(t, HoldPayoutIdempotencyKey("group", o1), HoldPayoutIdempotencyKey("group", o1))

	// Distinct across genuinely-different operations (the anti-#549 property: a
	// too-stable key would silently drop the second refund/transfer).
	require.NotEqual(t, RefundFullIdempotencyKey(o1), RefundFullIdempotencyKey(o2))
	require.NotEqual(t, RefundLineIdempotencyKey(o1, line1), RefundLineIdempotencyKey(o1, line2), "different lines of the same order must not collide")
	require.NotEqual(t, RefundPartialIdempotencyKey(o1, 100), RefundPartialIdempotencyKey(o1, 200), "sequential partials must not collide")
	require.NotEqual(t, TopupIdempotencyKey(o1, 0, ")acc_chef"), TopupIdempotencyKey(o1, 0, ")acc_driver"), "chef vs driver top-up must not collide")
	require.NotEqual(t, HoldPayoutIdempotencyKey("group", o1), HoldPayoutIdempotencyKey("mealplanday", o1))
	// A full-order refund and a line refund of the same order are distinct operations.
	require.NotEqual(t, RefundFullIdempotencyKey(o1), RefundLineIdempotencyKey(o1, line1))

	// And every normalized form is gateway-valid.
	for _, k := range []string{
		RefundFullIdempotencyKey(o1), RefundLineIdempotencyKey(o1, line1),
		RefundPartialIdempotencyKey(o1, 100), TopupIdempotencyKey(o1, 0, ")acc_x"),
		HoldPayoutIdempotencyKey("group", o1),
	} {
		n := normalizeIdempotencyKey(k)
		require.Regexp(t, gatewayKeyCharset, n)
		require.GreaterOrEqual(t, len(n), 10)
		require.LessOrEqual(t, len(n), 36)
	}
}
