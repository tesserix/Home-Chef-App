package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

// newSupportChatRig wires a gin engine whose routes are guarded by a stub
// middleware that mimics BFFAuth (sets the hydrated *models.User on the
// context), pointed at a fake otto server. Returns the engine + the fake's
// closer.
func newSupportChatRig(t *testing.T, base, secret, wsBase string, otto http.HandlerFunc) (*gin.Engine, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var srv *httptest.Server
	if otto != nil {
		srv = httptest.NewServer(otto)
		if base == "" {
			base = srv.URL
		}
	}

	h := newSupportChatHandler(base, secret, wsBase)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user", &models.User{
			ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Email:     "buyer@example.com",
			FirstName: "Sam",
			LastName:  "Rout",
		})
		c.Next()
	})
	h.Register(r.Group("/support/chat"))

	closer := func() {
		if srv != nil {
			srv.Close()
		}
	}
	return r, closer
}

func TestSupportChat_Create_ForwardsScopeIdentityAndMergesSessionToken(t *testing.T) {
	var sawTenant, sawStore, sawUser, sawEmail, sawName, sawBody string
	r, closeFn := newSupportChatRig(t, "", "secret", "wss://api.test", func(w http.ResponseWriter, req *http.Request) {
		sawTenant = req.Header.Get("X-Tenant-Id")
		sawStore = req.Header.Get("X-Store-Id")
		sawUser = req.Header.Get("X-User-Id")
		sawEmail = req.Header.Get("X-User-Email")
		sawName = req.Header.Get("X-User-Name")
		b, _ := io.ReadAll(req.Body)
		sawBody = string(b)
		if !strings.HasSuffix(req.URL.Path, "/api/v1/storefront/otto/conversations") {
			t.Errorf("otto path = %q", req.URL.Path)
		}
		http.SetCookie(w, &http.Cookie{Name: "otto_session", Value: "sess-xyz", Path: "/"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"conversation":{"id":"c1","case_id":"CS-1"},"first_message":{"id":"m1"}}`))
	})
	defer closeFn()

	rec := httptest.NewRecorder()
	body := `{"message":"hi","reason":"order_tracking","status_info":"late"}`
	req, _ := http.NewRequest(http.MethodPost, "/support/chat/conversations", strings.NewReader(body))
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if sawTenant != "homechef" || sawStore != "default" {
		t.Errorf("scope: tenant=%q store=%q", sawTenant, sawStore)
	}
	if sawUser != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("X-User-Id = %q (skips OTP)", sawUser)
	}
	if sawEmail != "buyer@example.com" || sawName != "Sam Rout" {
		t.Errorf("identity: email=%q name=%q", sawEmail, sawName)
	}
	if !strings.Contains(sawBody, `"reason":"order_tracking"`) {
		t.Errorf("body not forwarded verbatim: %q", sawBody)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["session_token"] != "sess-xyz" {
		t.Errorf("session_token = %v, want sess-xyz", out["session_token"])
	}
	if _, ok := out["conversation"]; !ok {
		t.Errorf("conversation missing from relayed body")
	}
	if rec.Header().Get("X-Otto-Session") != "sess-xyz" {
		t.Errorf("X-Otto-Session header not set")
	}
}

func TestSupportChat_PostMessage_ForwardsSessionHeaderAndId(t *testing.T) {
	var sawSession, sawPath string
	r, closeFn := newSupportChatRig(t, "", "secret", "", func(w http.ResponseWriter, req *http.Request) {
		sawSession = req.Header.Get("X-Otto-Session")
		sawPath = req.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":{"id":"m2"}}`))
	})
	defer closeFn()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/support/chat/conversations/c1/messages", strings.NewReader(`{"body":"hello"}`))
	req.Header.Set("X-Otto-Session", "sess-xyz")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if sawSession != "sess-xyz" {
		t.Errorf("X-Otto-Session forwarded = %q", sawSession)
	}
	if !strings.HasSuffix(sawPath, "/api/v1/storefront/otto/conversations/c1/messages") {
		t.Errorf("path = %q", sawPath)
	}
}

func TestSupportChat_WsTicket_AugmentsWithWsURL(t *testing.T) {
	r, closeFn := newSupportChatRig(t, "", "secret", "wss://fe3dr.com", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ticket":"tkt-123"}`))
	})
	defer closeFn()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/support/chat/conversations/c1/ws-ticket", nil)
	req.Header.Set("X-Otto-Session", "sess-xyz")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Ticket string `json:"ticket"`
		WsURL  string `json:"ws_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Ticket != "tkt-123" {
		t.Errorf("ticket = %q", out.Ticket)
	}
	want := "wss://fe3dr.com/api/v1/storefront/otto/conversations/c1/ws"
	if out.WsURL != want {
		t.Errorf("ws_url = %q, want %q", out.WsURL, want)
	}
}

func TestSupportChat_5xxSuppressed(t *testing.T) {
	r, closeFn := newSupportChatRig(t, "", "secret", "", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"otto_internal_secret_leak"}`))
	})
	defer closeFn()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/support/chat/resume", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "otto_internal_secret_leak") {
		t.Errorf("5xx body leaked: %s", rec.Body.String())
	}
}

func TestSupportChat_Dark_Returns503(t *testing.T) {
	// Empty base/secret ⇒ dark handler; no fake otto needed.
	r, closeFn := newSupportChatRig(t, "", "", "", nil)
	defer closeFn()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/support/chat/resume", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}
