package services

// account_restore_token.go — the short-lived credential that lets a returning
// user act on their own deleted account.
//
// After deletion the GIP identity is gone, so someone signing up again has no
// session with which to authenticate "restore my account". The alternative —
// teaching the auth middleware to admit soft-deleted users — would widen the
// authenticated surface for every request in order to serve two endpoints.
//
// Instead UpsertUser mints a token bound to (deleted user id, the NEW gip uid)
// with a 15-minute life. It authorises exactly two calls, /account/restore and
// /account/start-fresh, and nothing else. Binding the new uid matters: a token
// leaked to a third party is useless to them, because acting on it requires
// also holding the GIP identity it names.

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

// RestoreTokenTTL is deliberately short: the token is handed straight to the
// app, which acts on it within one screen transition.
const RestoreTokenTTL = 15 * time.Minute

// MintRestoreToken returns "<userID>.<gipUid>.<expiryUnix>.<signature>".
func MintRestoreToken(userID uuid.UUID, gipUid string) string {
	exp := time.Now().UTC().Add(RestoreTokenTTL).Unix()
	payload := restorePayload(userID.String(), gipUid, exp)
	return payload + "." + signRestorePayload(payload)
}

// ParseRestoreToken verifies signature and expiry and returns the account the
// token authorises, plus the GIP uid it is bound to.
func ParseRestoreToken(token string) (uuid.UUID, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 4 {
		return uuid.Nil, "", fmt.Errorf("restore token: malformed")
	}
	payload := strings.Join(parts[:3], ".")

	// Constant-time compare so a caller cannot narrow in on a valid signature
	// by timing repeated guesses.
	if !hmac.Equal([]byte(signRestorePayload(payload)), []byte(parts[3])) {
		return uuid.Nil, "", fmt.Errorf("restore token: bad signature")
	}

	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("restore token: bad expiry")
	}
	if time.Now().UTC().After(time.Unix(exp, 0).UTC()) {
		return uuid.Nil, "", fmt.Errorf("restore token: expired")
	}

	userID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("restore token: bad subject")
	}
	if parts[1] == "" {
		return uuid.Nil, "", fmt.Errorf("restore token: missing identity binding")
	}
	return userID, parts[1], nil
}

func restorePayload(userID, gipUid string, exp int64) string {
	return userID + "." + gipUid + "." + strconv.FormatInt(exp, 10)
}

func signRestorePayload(payload string) string {
	mac := hmac.New(sha256.New, config.AppConfig.BFFInternalHMACKey)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
