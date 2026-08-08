package payouts

// #1156. The key a chef batch derives is 68 characters, so a varchar(64)
// column refused every insert with a 22001 — no Cashfree payout could be
// prepared at all.

import (
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var varcharWidth = regexp.MustCompile(`varchar\((\d+)\)`)

func declaredWidth(t *testing.T, field string) int {
	t.Helper()
	f, ok := reflect.TypeOf(Batch{}).FieldByName(field)
	require.True(t, ok, "Batch has no field %s", field)
	m := varcharWidth.FindStringSubmatch(f.Tag.Get("gorm"))
	require.Len(t, m, 2, "%s declares no varchar width", field)
	w, err := strconv.Atoi(m[1])
	require.NoError(t, err)
	return w
}

func TestIdempotencyKeyFitsItsColumn(t *testing.T) {
	key := IdempotencyKeyFor("homechef",
		PayeeRef{Type: PayeeChef, ID: uuid.MustParse("e150c72a-42e2-4beb-8cb1-389666dd813c")},
		"2026-07-19")

	require.LessOrEqual(t, len(key), declaredWidth(t, "IdempotencyKey"),
		"key %q (%d chars) does not fit the column", key, len(key))
}

func TestIdempotencyKeyFitsForEveryPayeeType(t *testing.T) {
	width := declaredWidth(t, "IdempotencyKey")
	for _, payee := range []PayeeType{PayeeChef, PayeeDeliveryPartner} {
		key := IdempotencyKeyFor("homechef", PayeeRef{Type: payee, ID: uuid.New()}, "2026-12-31")
		require.LessOrEqual(t, len(key), width, "%s key %q is %d chars", payee, key, len(key))
	}
}
