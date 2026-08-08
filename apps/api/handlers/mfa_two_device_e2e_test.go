package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// #1164 finding 3, end to end: one account, two installs of the customer app.
//
// The reported symptom was that a second handset "looks dead rather than
// challenged". Two separate things had to hold for that to be false, and only
// one of them did: the gate must refuse the new device without disturbing the
// old one (it did), and the new device must be able to obtain and redeem its own
// code (it could not — the challenge was keyed by account, so the second device
// overwrote the first's code and sat behind its resend cooldown).
//
// This drives the real routes rather than the services, because the bug lived in
// how the handler keyed the call, not in the key builder.

const (
	installA = "install-a"
	installB = "install-b"
)

type twoDeviceEnv struct {
	router *gin.Engine
	redis  *miniredis.Miniredis
	userID uuid.UUID
	email  string
}

func newTwoDeviceEnv(t *testing.T) *twoDeviceEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE user_mfa_settings (
		user_id TEXT PRIMARY KEY, enabled INTEGER DEFAULT 0, email_enrolled INTEGER DEFAULT 0,
		phone_enrolled INTEGER DEFAULT 0, phone_e164_enc TEXT DEFAULT '', phone_e164_bidx TEXT DEFAULT '',
		enrolled_at DATETIME, disabled_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE trusted_devices (
		id TEXT PRIMARY KEY, user_id TEXT, app TEXT, token_hash TEXT, label TEXT, platform TEXT,
		created_at DATETIME, last_seen_at DATETIME, revoked_at DATETIME)`).Error)

	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prevRedis := services.SetRedisClientForTest(client)
	t.Cleanup(func() {
		_ = client.Close()
		services.SetRedisClientForTest(prevRedis)
	})

	prevCfg := config.AppConfig
	config.AppConfig = &config.Config{MFAEnabled: true}
	t.Cleanup(func() { config.AppConfig = prevCfg })

	uid, email := uuid.New(), "chef@fe3dr.com"
	now := time.Now()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, first_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		uid.String(), email, "Chef", now, now).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO user_mfa_settings (user_id, enabled, email_enrolled, enrolled_at, updated_at)
		 VALUES (?, 1, 1, ?, ?)`, uid.String(), now, now).Error)

	h := NewMFAHandler()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uid)
		c.Set(middleware.CtxUserID, uid.String())
		c.Set(middleware.CtxUserEmail, email)
		c.Next()
	})
	r.Use(middleware.MFAGate(db, true))
	r.POST("/api/v1/auth/mfa/challenge", h.Challenge)
	r.POST("/api/v1/auth/mfa/verify", h.Verify)
	r.GET("/api/v1/orders", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	return &twoDeviceEnv{router: r, redis: mr, userID: uid, email: email}
}

// call sends one request as a given install, optionally presenting a device token.
func (e *twoDeviceEnv) call(t *testing.T, method, path, install, deviceToken string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-App", "customer")
	if install != "" {
		req.Header.Set("X-Device-Id", install)
	}
	if deviceToken != "" {
		req.Header.Set("X-Device-Token", deviceToken)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// codeFor reads the code the server just stored for one install. Reaching into
// Redis is the only way to learn it — delivery is by email, deliberately.
func (e *twoDeviceEnv) codeFor(t *testing.T, install string) string {
	t.Helper()
	suffix := ":customer:" + install
	for _, k := range e.redis.Keys() {
		if strings.HasPrefix(k, "otp:mfa_login:code:") && strings.HasSuffix(k, suffix) {
			v, err := e.redis.Get(k)
			require.NoError(t, err)
			return v
		}
	}
	t.Fatalf("no code stored for %s; keys=%v", install, e.redis.Keys())
	return ""
}

func TestMFA_TwoInstallsEachCompleteTheirOwnChallenge(t *testing.T) {
	e := newTwoDeviceEnv(t)

	// Neither install is trusted yet, so both are challenged rather than served.
	require.Equal(t, http.StatusForbidden, e.call(t, "GET", "/api/v1/orders", installA, "", nil).Code)
	require.Equal(t, http.StatusForbidden, e.call(t, "GET", "/api/v1/orders", installB, "", nil).Code)

	// A asks for a code. B must be able to ask for its own straight away — this
	// is what used to fail, with B either sharing A's code or told to wait.
	require.Equal(t, http.StatusOK,
		e.call(t, "POST", "/api/v1/auth/mfa/challenge", installA, "", map[string]string{"channel": "email"}).Code)
	require.Equal(t, http.StatusOK,
		e.call(t, "POST", "/api/v1/auth/mfa/challenge", installB, "", map[string]string{"channel": "email"}).Code)

	codeA, codeB := e.codeFor(t, installA), e.codeFor(t, installB)
	require.NotEqual(t, codeA, codeB)

	tokenA := e.verify(t, installA, codeA)
	require.Equal(t, http.StatusOK, e.call(t, "GET", "/api/v1/orders", installA, tokenA, nil).Code)

	// A passing must not have let B in, and B's code must still be good.
	require.Equal(t, http.StatusForbidden, e.call(t, "GET", "/api/v1/orders", installB, "", nil).Code)

	tokenB := e.verify(t, installB, codeB)
	require.NotEqual(t, tokenA, tokenB)

	// Both installs now work, at the same time.
	require.Equal(t, http.StatusOK, e.call(t, "GET", "/api/v1/orders", installB, tokenB, nil).Code)
	require.Equal(t, http.StatusOK, e.call(t, "GET", "/api/v1/orders", installA, tokenA, nil).Code)
}

// One install's code must not sign the other in.
func TestMFA_CodeFromOneInstallDoesNotVerifyTheOther(t *testing.T) {
	e := newTwoDeviceEnv(t)

	for _, install := range []string{installA, installB} {
		require.Equal(t, http.StatusOK,
			e.call(t, "POST", "/api/v1/auth/mfa/challenge", install, "", map[string]string{"channel": "email"}).Code)
	}

	w := e.call(t, "POST", "/api/v1/auth/mfa/verify", installB, "", map[string]any{
		"code": e.codeFor(t, installA), "rememberDevice": true,
	})
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// The challenge endpoints have to stay reachable from an install that is being
// refused everywhere else, or it really would be dead.
func TestMFA_ChallengeIsReachableFromTheRefusedInstall(t *testing.T) {
	e := newTwoDeviceEnv(t)

	blocked := e.call(t, "GET", "/api/v1/orders", installB, "", nil)
	require.Equal(t, http.StatusForbidden, blocked.Code)

	var payload struct {
		Error    string            `json:"error"`
		Channels []string          `json:"channels"`
		Masked   map[string]string `json:"masked"`
	}
	require.NoError(t, json.Unmarshal(blocked.Body.Bytes(), &payload))
	require.Equal(t, "mfa_required", payload.Error)
	require.Contains(t, payload.Channels, "email")
	require.NotEmpty(t, payload.Masked["email"], "the client needs a hint to render the challenge screen")

	require.Equal(t, http.StatusOK,
		e.call(t, "POST", "/api/v1/auth/mfa/challenge", installB, "", map[string]string{"channel": "email"}).Code)
}

func (e *twoDeviceEnv) verify(t *testing.T, install, code string) string {
	t.Helper()
	w := e.call(t, "POST", "/api/v1/auth/mfa/verify", install, "", map[string]any{
		"code": code, "rememberDevice": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var out struct {
		DeviceToken string `json:"deviceToken"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.NotEmpty(t, out.DeviceToken)
	return out.DeviceToken
}

func TestMFA_TrustedInstallSurvivesTheOtherBeingRevoked(t *testing.T) {
	e := newTwoDeviceEnv(t)

	for _, install := range []string{installA, installB} {
		require.Equal(t, http.StatusOK,
			e.call(t, "POST", "/api/v1/auth/mfa/challenge", install, "", map[string]string{"channel": "email"}).Code)
	}
	tokenA := e.verify(t, installA, e.codeFor(t, installA))
	tokenB := e.verify(t, installB, e.codeFor(t, installB))

	devices, err := services.ListTrustedDevices(database.DB, e.userID)
	require.NoError(t, err)
	require.Len(t, devices, 2)

	require.NoError(t, services.RevokeTrustedDevice(database.DB, e.userID, devices[0].ID))

	// Losing one device must cost exactly one device its access.
	codes := map[string]int{
		tokenA: e.call(t, "GET", "/api/v1/orders", installA, tokenA, nil).Code,
		tokenB: e.call(t, "GET", "/api/v1/orders", installB, tokenB, nil).Code,
	}
	var ok, refused int
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusForbidden:
			refused++
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, refused)
}
