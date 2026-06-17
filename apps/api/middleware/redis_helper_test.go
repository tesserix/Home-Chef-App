package middleware

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/homechef/api/services"
	"github.com/redis/go-redis/v9"
)

// startMiniredis spins up an in-memory Redis, wires it into the
// services.GetRedisClient() singleton via the SetRedisClientForTest seam,
// and restores the prior client on test cleanup. Returns the miniredis
// handle so tests can FlushAll, FastForward TTLs, or Close it to simulate
// an outage. Tests run serially (no t.Parallel) because they share the
// process-global Redis singleton.
func startMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prev := services.SetRedisClientForTest(client)
	t.Cleanup(func() {
		_ = client.Close()
		services.SetRedisClientForTest(prev)
	})
	return mr
}

// clearRedisClient detaches any Redis client from the singleton so
// IsConnected() reports false — used to exercise the fail-open paths.
func clearRedisClient(t *testing.T) {
	t.Helper()
	prev := services.SetRedisClientForTest(nil)
	t.Cleanup(func() { services.SetRedisClientForTest(prev) })
}
