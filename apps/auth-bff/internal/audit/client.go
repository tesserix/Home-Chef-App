package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Event struct {
	Type       string         `json:"type"`
	UserID     string         `json:"user_id,omitempty"`
	Email      string         `json:"email,omitempty"`
	Pool       string         `json:"pool,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
	Attrs      map[string]any `json:"attrs,omitempty"`
}

type Client struct {
	url  string
	http *http.Client
	wg   sync.WaitGroup
}

func New(url string) *Client {
	return &Client{
		url:  url,
		http: &http.Client{Timeout: 2 * time.Second},
	}
}

// Emit fires an audit event asynchronously. It returns immediately.
// If the client has no URL configured (empty), it's a no-op (counts as success).
// Errors are intentionally swallowed — auditing is best-effort and never blocks user flows.
func (c *Client) Emit(e Event) {
	if c == nil || c.url == "" {
		return
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now()
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		body, err := json.Marshal(e)
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}()
}

// Wait blocks until all in-flight Emit goroutines complete. Useful for tests
// and graceful shutdown.
func (c *Client) Wait() {
	if c == nil {
		return
	}
	c.wg.Wait()
}
