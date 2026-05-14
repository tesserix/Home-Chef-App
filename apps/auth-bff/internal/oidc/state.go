package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// StateEntry is the data stored against an OAuth state value during the
// authorize -> callback round-trip. It carries the resolved app name, the
// nonce we'll later verify against the id_token, and an optional post-login
// return URL captured at /auth/login time.
type StateEntry struct {
	AppName  string
	Nonce    string
	ReturnTo string
	Created  time.Time
}

// StateStore is a minimal one-shot store: Put writes an entry, Take returns
// and removes it. Implementations must be safe for concurrent use.
type StateStore interface {
	Put(key string, e StateEntry)
	Take(key string) (StateEntry, bool)
}

type memStore struct {
	mu  sync.Mutex
	m   map[string]StateEntry
	ttl time.Duration
}

// NewMemStateStore returns an in-memory state store with a 5-minute TTL.
// Single-process only; multi-replica deployments need a shared store.
func NewMemStateStore() StateStore {
	return &memStore{m: map[string]StateEntry{}, ttl: 5 * time.Minute}
}

func (s *memStore) Put(k string, e StateEntry) {
	e.Created = time.Now()
	s.mu.Lock()
	s.m[k] = e
	s.mu.Unlock()
}

func (s *memStore) Take(k string) (StateEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[k]
	delete(s.m, k)
	if ok && time.Since(e.Created) > s.ttl {
		return StateEntry{}, false
	}
	return e, ok
}

// NewStateID returns a cryptographically random URL-safe identifier suitable
// for use as either an OAuth state or nonce value.
func NewStateID() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
