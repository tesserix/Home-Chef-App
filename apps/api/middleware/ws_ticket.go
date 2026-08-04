package middleware

// ws_ticket.go — query-string authentication for WebSocket upgrades.
//
// Why this exists: neither client that needs a socket can authenticate one.
// A browser's WebSocket API cannot set request headers at all, and the BFF
// cannot proxy an upgrade, so the browser portals reach /ws/* only via the
// BFF's HMAC. A mobile client can set headers (React Native allows it) but
// holds a Bearer session, not the BFF key — and the header trick is a
// React-Native extension, not WHATWG, so it cannot be shared with web.
//
// A short-lived ticket in the query string is the one credential every client
// can carry. The caller mints it over the normal authenticated REST path
// (where it is already identified, by HMAC or Bearer) and spends it on the
// upgrade. Same shape support chat already uses for otto (#982).
//
// The ticket is a bearer credential in a URL, so it is deliberately narrow:
// ~60s TTL, single audience, and it carries the identity rather than pointing
// at a server-side session — no store to keep, nothing to revoke, and a
// leaked URL is stale before it is useful.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// WSTicketTTL is how long a minted ticket stays valid. Long enough to survive
// the round trip and a slow handshake, short enough that a ticket captured
// from a URL (proxy log, crash report) is already dead.
const WSTicketTTL = 60 * time.Second

// wsTicketAudience namespaces the signature so a ticket can never be replayed
// against any other HMAC surface that shares the key.
const wsTicketAudience = "ws"

var (
	ErrWSTicketMalformed = errors.New("malformed ws ticket")
	ErrWSTicketSignature = errors.New("bad ws ticket signature")
	ErrWSTicketExpired   = errors.New("expired ws ticket")
)

// wsTicketClaims is the identity carried by a ticket. Field names are short
// because the whole thing rides in a query string.
type wsTicketClaims struct {
	Aud   string `json:"aud"`
	UID   string `json:"u"`
	Email string `json:"e,omitempty"`
	Role  string `json:"r,omitempty"`
	Pool  string `json:"p,omitempty"`
	Exp   int64  `json:"x"` // unix seconds
}

// MintWSTicket signs an identity into a ticket valid for WSTicketTTL.
func MintWSTicket(key []byte, id BFFIdentity, now time.Time) (string, error) {
	claims := wsTicketClaims{
		Aud:   wsTicketAudience,
		UID:   id.UserID,
		Email: id.Email,
		Role:  id.Role,
		Pool:  id.Pool,
		Exp:   now.Add(WSTicketTTL).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return body + "." + base64.RawURLEncoding.EncodeToString(signWSTicket(key, body)), nil
}

func signWSTicket(key []byte, body string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(wsTicketAudience + "." + body))
	return mac.Sum(nil)
}

// VerifyWSTicket checks the signature and expiry and returns the identity.
// The signature is compared before the claims are trusted for anything.
func VerifyWSTicket(key []byte, ticket string, now time.Time) (*BFFIdentity, error) {
	body, sig, found := strings.Cut(ticket, ".")
	if !found || body == "" || sig == "" {
		return nil, ErrWSTicketMalformed
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return nil, ErrWSTicketMalformed
	}
	if !hmac.Equal(got, signWSTicket(key, body)) {
		return nil, ErrWSTicketSignature
	}
	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, ErrWSTicketMalformed
	}
	var claims wsTicketClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrWSTicketMalformed
	}
	// Audience is inside the signed payload as well as the MAC prefix so a
	// ticket minted for some future audience cannot be spent here.
	if claims.Aud != wsTicketAudience || claims.UID == "" {
		return nil, ErrWSTicketMalformed
	}
	if now.Unix() > claims.Exp {
		return nil, ErrWSTicketExpired
	}
	return &BFFIdentity{
		UserID: claims.UID,
		Email:  claims.Email,
		Role:   claims.Role,
		Pool:   claims.Pool,
	}, nil
}

// BFFAuthOrTicket authenticates a WebSocket upgrade by either credential: the
// BFF's HMAC signature (browser portals, which reach /ws/* through Istio with
// the BFF's headers) or a `?ticket=` minted by MintWSTicket (mobile, and any
// client that cannot set headers).
//
// Ticket first, and only when present: a request carrying a ticket is by
// definition not BFF-signed, and falling through to BFFAuth would answer a bad
// ticket with a confusing "missing signature" 401.
func BFFAuthOrTicket(cfg BFFAuthConfig) gin.HandlerFunc {
	hmacAuth := BFFAuth(cfg)
	return func(c *gin.Context) {
		ticket := c.Query("ticket")
		if ticket == "" {
			hmacAuth(c)
			return
		}
		id, err := VerifyWSTicket(cfg.HMACKey, ticket, time.Now())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		applyBFFIdentity(c, id)
		c.Set(ctxBFFResolved, true)
		c.Next()
	}
}

// BFFAuthOrTicketOptional is the optional-auth counterpart, for sockets whose
// payload is public and which a signed-out visitor must still be able to open
// — chef open/close, which the customer app subscribes to while browsing as a
// guest. A credential is honoured when supplied and the identity applied; its
// absence is not an error.
//
// An INVALID credential still 401s. Anonymous means none at all, never a
// rejected one, so a forged ticket cannot buy anonymous access to a socket
// that later grows a user-scoped payload.
func BFFAuthOrTicketOptional(cfg BFFAuthConfig) gin.HandlerFunc {
	strict := BFFAuthOrTicket(cfg)
	return func(c *gin.Context) {
		if c.Query("ticket") == "" && c.Request.Header.Get(HdrSignature) == "" {
			c.Next()
			return
		}
		strict(c)
	}
}
