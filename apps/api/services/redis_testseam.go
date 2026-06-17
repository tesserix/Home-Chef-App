package services

import "github.com/redis/go-redis/v9"

// SetRedisClientForTest injects a *redis.Client into the package-level
// singleton so middleware that calls services.GetRedisClient() exercises
// real Redis semantics against an in-memory miniredis server in tests.
//
// This is the single justified test seam for the security-middleware
// test suite (Issue #10): the Redis-backed rate-limit and idempotency
// middleware resolve their client through the GetRedisClient() singleton,
// whose underlying *redis.Client is unexported and otherwise only set by
// Connect() against a real Redis URL. Without this seam there is no way
// to drive the connected/INCR/SetNX/Get/Del paths in a hermetic unit test.
//
// It returns the previous client so tests can restore the prior state in
// a defer, keeping the global singleton clean across test functions.
func SetRedisClientForTest(client *redis.Client) *redis.Client {
	inst := GetRedisClient()
	prev := inst.client
	inst.client = client
	return prev
}
