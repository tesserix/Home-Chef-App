package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var wsTestKey = []byte("a-test-hmac-key-that-is-long-enough")

func wsTestIdentity() BFFIdentity {
	return BFFIdentity{
		UserID: "6f1a4c2e-0000-4000-8000-000000000001",
		Email:  "customer@example.com",
		Role:   "customer",
		Pool:   "customer",
	}
}

func TestMintAndVerifyRoundTrip(t *testing.T) {
	now := time.Now()
	ticket, err := MintWSTicket(wsTestKey, wsTestIdentity(), now)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	got, err := VerifyWSTicket(wsTestKey, ticket, now)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if *got != wsTestIdentity() {
		t.Fatalf("identity round-trip mismatch: got %+v want %+v", *got, wsTestIdentity())
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	now := time.Now()
	ticket, _ := MintWSTicket(wsTestKey, wsTestIdentity(), now)
	if _, err := VerifyWSTicket([]byte("a-different-key-entirely-padded-ok"), ticket, now); err != ErrWSTicketSignature {
		t.Fatalf("a ticket signed with another key must not verify, got %v", err)
	}
}

// The whole point of the signature: the claims must not be trusted before it
// is checked. Re-encoding a different user with the ORIGINAL signature is the
// attack this defends against.
func TestVerifyRejectsTamperedClaims(t *testing.T) {
	now := time.Now()
	ticket, _ := MintWSTicket(wsTestKey, wsTestIdentity(), now)
	body, sig, _ := strings.Cut(ticket, ".")

	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var claims wsTicketClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	claims.UID = "6f1a4c2e-0000-4000-8000-0000000000ff" // escalate to another user
	claims.Role = "admin"
	tampered, _ := json.Marshal(claims)
	forged := base64.RawURLEncoding.EncodeToString(tampered) + "." + sig

	if _, err := VerifyWSTicket(wsTestKey, forged, now); err != ErrWSTicketSignature {
		t.Fatalf("tampered claims must be rejected, got %v", err)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	now := time.Now()
	ticket, _ := MintWSTicket(wsTestKey, wsTestIdentity(), now)
	// One second past the TTL — a ticket must not outlive its window.
	if _, err := VerifyWSTicket(wsTestKey, ticket, now.Add(WSTicketTTL+time.Second)); err != ErrWSTicketExpired {
		t.Fatalf("expired ticket must be rejected, got %v", err)
	}
	// Still good on the last second, so a slow handshake is not punished.
	if _, err := VerifyWSTicket(wsTestKey, ticket, now.Add(WSTicketTTL)); err != nil {
		t.Fatalf("ticket must remain valid to the end of its TTL, got %v", err)
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct{ name, ticket string }{
		{"empty", ""},
		{"no separator", "abcdef"},
		{"empty body", ".c2ln"},
		{"empty sig", "Ym9keQ=="},
		{"not base64", "!!!.???"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := VerifyWSTicket(wsTestKey, tc.ticket, now); err == nil {
				t.Fatalf("malformed ticket %q must be rejected", tc.ticket)
			}
		})
	}
}

// A ticket signed for another audience with the same key must not open a
// socket — that is what the audience prefix in the MAC is for.
func TestVerifyRejectsForeignAudience(t *testing.T) {
	now := time.Now()
	claims := wsTicketClaims{Aud: "some-other-surface", UID: wsTestIdentity().UserID, Exp: now.Add(time.Minute).Unix()}
	payload, _ := json.Marshal(claims)
	body := base64.RawURLEncoding.EncodeToString(payload)
	// Signed with the real key, but over the foreign audience.
	mac := signWSTicket(wsTestKey, body)
	foreign := body + "." + base64.RawURLEncoding.EncodeToString(mac)

	// The MAC matches (same helper), so this isolates the in-payload audience check.
	if _, err := VerifyWSTicket(wsTestKey, foreign, now); err != ErrWSTicketMalformed {
		t.Fatalf("foreign-audience ticket must be rejected, got %v", err)
	}
}

func TestBFFAuthOrTicketAcceptsValidTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ticket, _ := MintWSTicket(wsTestKey, wsTestIdentity(), time.Now())

	r := gin.New()
	r.GET("/ws/notifications", BFFAuthOrTicket(BFFAuthConfig{HMACKey: wsTestKey}), func(c *gin.Context) {
		uid, ok := GetUserID(c)
		if !ok {
			c.String(http.StatusInternalServerError, "no identity on context")
			return
		}
		c.String(http.StatusOK, uid.String())
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws/notifications?ticket="+ticket, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("valid ticket should authenticate, got %d (%s)", w.Code, w.Body.String())
	}
	if w.Body.String() != wsTestIdentity().UserID {
		t.Fatalf("wrong identity applied: got %q", w.Body.String())
	}
}

func TestBFFAuthOrTicketRejectsBadTicketWithout401Fallthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	reached := false
	r.GET("/ws/notifications", BFFAuthOrTicket(BFFAuthConfig{HMACKey: wsTestKey}), func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws/notifications?ticket=garbage.sig", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad ticket must 401, got %d", w.Code)
	}
	if reached {
		t.Fatal("handler must not run for a bad ticket")
	}
}

// No ticket at all must still fall through to the BFF HMAC path, or the
// browser portals that reach /ws/* through the BFF would break.
func TestBFFAuthOrTicketFallsBackToHMACWhenNoTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	reached := false
	r.GET("/ws/notifications", BFFAuthOrTicket(BFFAuthConfig{HMACKey: wsTestKey}), func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws/notifications", nil))

	// Unsigned and unticketed: rejected by the HMAC path, not silently allowed.
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned request must 401 via the HMAC path, got %d", w.Code)
	}
	if reached {
		t.Fatal("handler must not run without any credential")
	}
}

// Chef availability is public and the customer app subscribes while browsing
// signed out, so no credential at all must pass through.
func TestBFFAuthOrTicketOptionalAllowsAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	reached := false
	r.GET("/ws/chefs/:id/availability",
		BFFAuthOrTicketOptional(BFFAuthConfig{HMACKey: wsTestKey}),
		func(c *gin.Context) { reached = true; c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws/chefs/abc/availability", nil))

	if w.Code != http.StatusOK || !reached {
		t.Fatalf("anonymous must be allowed on the optional gate, got %d", w.Code)
	}
}

func TestBFFAuthOrTicketOptionalAppliesIdentityWhenTicketed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ticket, _ := MintWSTicket(wsTestKey, wsTestIdentity(), time.Now())
	r := gin.New()
	r.GET("/ws/chefs/:id/availability",
		BFFAuthOrTicketOptional(BFFAuthConfig{HMACKey: wsTestKey}),
		func(c *gin.Context) {
			uid, ok := GetUserID(c)
			if !ok {
				c.String(http.StatusOK, "anonymous")
				return
			}
			c.String(http.StatusOK, uid.String())
		})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws/chefs/abc/availability?ticket="+ticket, nil))

	if w.Body.String() != wsTestIdentity().UserID {
		t.Fatalf("a supplied ticket must still identify the caller, got %q", w.Body.String())
	}
}

// An INVALID credential is not the same as none: a forged ticket must not buy
// anonymous access to a socket that may later carry a user-scoped payload.
func TestBFFAuthOrTicketOptionalRejectsBadTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	reached := false
	r.GET("/ws/chefs/:id/availability",
		BFFAuthOrTicketOptional(BFFAuthConfig{HMACKey: wsTestKey}),
		func(c *gin.Context) { reached = true; c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws/chefs/abc/availability?ticket=forged.sig", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("a forged ticket must 401 rather than degrade to anonymous, got %d", w.Code)
	}
	if reached {
		t.Fatal("handler must not run for a forged ticket")
	}
}
