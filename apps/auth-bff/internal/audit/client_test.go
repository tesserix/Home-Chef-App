package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Emit_PostsAsync(t *testing.T) {
	var calls atomic.Int32
	var received Event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.Emit(Event{Type: "sign_in", UserID: "u1", Email: "a@b.com", Pool: "customer"})
	c.Wait()

	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, "sign_in", received.Type)
	assert.Equal(t, "u1", received.UserID)
	assert.False(t, received.OccurredAt.IsZero())
}

func TestClient_EmptyURL_NoOp(t *testing.T) {
	c := New("")
	c.Emit(Event{Type: "x"})
	c.Wait() // returns immediately
	// no panic, no goroutine leak
}

func TestClient_BadEndpoint_SwallowsError(t *testing.T) {
	// Unreachable port; client must not crash and Emit must return immediately
	c := New("http://127.0.0.1:1")
	c.Emit(Event{Type: "x"})
	done := make(chan struct{})
	go func() { c.Wait(); close(done) }()
	select {
	case <-done:
		// good, wait returned (after ~2s timeout)
	case <-time.After(3 * time.Second):
		require.Fail(t, "Wait blocked beyond client timeout")
	}
}

func TestClient_NilReceiver_NoOp(t *testing.T) {
	var c *Client
	c.Emit(Event{Type: "x"}) // must not panic
	c.Wait()
}
