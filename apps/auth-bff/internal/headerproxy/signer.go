package headerproxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrMissingSignature  = errors.New("missing X-Internal-Auth")
	ErrStaleTimestamp    = errors.New("stale X-Auth-Ts")
	ErrSignatureMismatch = errors.New("HMAC mismatch")
)

const (
	HdrUserID    = "X-User-Id"
	HdrUserEmail = "X-User-Email"
	HdrUserRole  = "X-User-Role"
	HdrAuthPool  = "X-Auth-Pool"
	HdrAuthTs    = "X-Auth-Ts"
	HdrSignature = "X-Internal-Auth"
)

type SignerConfig struct {
	Key    []byte
	Window time.Duration
}

type Signer struct {
	cfg SignerConfig
}

type Identity struct {
	UserID string
	Email  string
	Role   string
	Pool   string
}

func NewSigner(cfg SignerConfig) *Signer {
	if cfg.Window == 0 {
		cfg.Window = 60 * time.Second
	}
	return &Signer{cfg: cfg}
}

func (s *Signer) Sign(r *http.Request, body []byte, id Identity) error {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r.Header.Set(HdrUserID, id.UserID)
	r.Header.Set(HdrUserEmail, id.Email)
	r.Header.Set(HdrUserRole, id.Role)
	r.Header.Set(HdrAuthPool, id.Pool)
	r.Header.Set(HdrAuthTs, ts)
	r.Header.Set(HdrSignature, s.compute(r.Method, r.URL.Path, body, ts))
	return nil
}

func (s *Signer) Verify(r *http.Request, body []byte) (*Identity, error) {
	sig := r.Header.Get(HdrSignature)
	if sig == "" {
		return nil, ErrMissingSignature
	}
	ts := r.Header.Get(HdrAuthTs)
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad X-Auth-Ts: %w", err)
	}
	// Verify signature FIRST (constant-time, no timing oracle on ts validity)
	want := s.compute(r.Method, r.URL.Path, body, ts)
	if !hmac.Equal([]byte(sig), []byte(want)) {
		return nil, ErrSignatureMismatch
	}
	// Then check the freshness window
	d := time.Since(time.Unix(tsInt, 0))
	if d > s.cfg.Window || d < -s.cfg.Window {
		return nil, ErrStaleTimestamp
	}
	return &Identity{
		UserID: r.Header.Get(HdrUserID),
		Email:  r.Header.Get(HdrUserEmail),
		Role:   r.Header.Get(HdrUserRole),
		Pool:   r.Header.Get(HdrAuthPool),
	}, nil
}

func (s *Signer) compute(method, path string, body []byte, ts string) string {
	bodyHash := sha256.Sum256(body)
	m := hmac.New(sha256.New, s.cfg.Key)
	fmt.Fprintf(m, "%s\n%s\n%s\n%s", method, path, hex.EncodeToString(bodyHash[:]), ts)
	return hex.EncodeToString(m.Sum(nil))
}
