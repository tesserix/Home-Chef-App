package services

// delivery_quote_pin.go — pins the surge a customer was quoted so the charge
// matches it.
//
// Surge is live: traffic re-reads every 5 minutes, weather every 20. Once surge
// moves the CHARGED fee, recomputing it at order creation would bill a number the
// customer never saw — the exact drift services/delivery_fee.go exists to prevent.
// So the quote signs the multiplier it used and CreateOrder replays it.
//
// Stateless (HMAC over the existing JWT secret) so there is no row to migrate,
// expire or clean up. The pin is not a secret — it carries no customer data and
// only ever reproduces a number the customer was already shown.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
)

// surgePinTTL is how long a quoted surge stays chargeable. Long enough to pay for
// a cart, short enough that a stale storm doesn't price tomorrow's order.
const surgePinTTL = 20 * time.Minute

// surgePinCoordPrecision — the pin is bound to the drop it was quoted for, at
// ~100 m. Tighter would void the pin on GPS jitter; looser would let a pin from
// one address price another.
const surgePinCoordPrecision = 3

// SignSurgePin returns a token binding a surge multiplier to a chef + drop.
// Returns "" when there is nothing to pin (neutral surge), so the field is simply
// absent from a quote with no live signal.
func SignSurgePin(chefID uuid.UUID, dropLat, dropLng, surge float64) string {
	if surge <= 1.0 {
		return ""
	}
	body := surgePinBody(chefID, dropLat, dropLng, surge, time.Now().Add(surgePinTTL).Unix())
	return body + "." + surgePinSignature(body)
}

// VerifySurgePin recovers the pinned multiplier for this chef + drop. Any failure
// — tampered, expired, or issued for a different chef/address — returns
// (1.0, false) and the caller resolves surge live instead.
func VerifySurgePin(token string, chefID uuid.UUID, dropLat, dropLng float64) (float64, bool) {
	if token == "" {
		return 1.0, false
	}
	idx := strings.LastIndex(token, ".")
	if idx <= 0 {
		return 1.0, false
	}
	body, sig := token[:idx], token[idx+1:]
	if !hmac.Equal([]byte(sig), []byte(surgePinSignature(body))) {
		return 1.0, false
	}

	parts := strings.Split(body, "|")
	if len(parts) != 5 {
		return 1.0, false
	}
	exp, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return 1.0, false
	}
	surge, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return 1.0, false
	}
	// Rebuild the identity from the CALLER's chef + coords: if either differs, the
	// body won't match and the pin can't be replayed onto a different order.
	if body != surgePinBody(chefID, dropLat, dropLng, surge, exp) {
		return 1.0, false
	}
	return clampSurge(surge), true
}

func surgePinBody(chefID uuid.UUID, dropLat, dropLng, surge float64, exp int64) string {
	return fmt.Sprintf("%s|%.*f|%.*f|%.4f|%d",
		chefID.String(),
		surgePinCoordPrecision, dropLat,
		surgePinCoordPrecision, dropLng,
		surge, exp)
}

func surgePinSignature(body string) string {
	mac := hmac.New(sha256.New, []byte(surgePinKey()))
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// surgePinKey is the HMAC key. Empty when unconfigured, which SurgeChargeEnabled
// treats as "surge cannot be charged" — so an unsigned pin is never chargeable.
func surgePinKey() string {
	if config.AppConfig == nil {
		return ""
	}
	return config.AppConfig.DeliverySurgePinKey
}
