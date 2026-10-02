package services

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRedisClient_KeysStayInsideACLPrefix(t *testing.T) {
	mr := withMiniredis(t)
	r := GetRedisClient()
	ctx := context.Background()

	if err := r.Set(ctx, "a", "1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := r.SetJSON(ctx, "b", map[string]int{"n": 1}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := r.SetNX(ctx, "c", "1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := r.IncrAndExpire(ctx, "d", time.Minute); err != nil {
		t.Fatal(err)
	}

	keys := mr.Keys()
	if len(keys) != 4 {
		t.Fatalf("want 4 keys, got %v", keys)
	}
	for _, k := range keys {
		if !strings.HasPrefix(k, "homechef:") {
			t.Fatalf("key %q escapes the homechef ACL prefix", k)
		}
	}
	if ttl := mr.TTL("homechef:d"); ttl != time.Minute {
		t.Fatalf("counter TTL = %v, want 1m", ttl)
	}

	if v, err := r.Get(ctx, "a"); err != nil || v != "1" {
		t.Fatalf("Get(a) = %q, %v", v, err)
	}
	var b map[string]int
	if err := r.GetJSON(ctx, "b", &b); err != nil || b["n"] != 1 {
		t.Fatalf("GetJSON(b) = %v, %v", b, err)
	}
	if err := r.Del(ctx, "a"); err != nil || mr.Exists("homechef:a") {
		t.Fatalf("Del(a) left key behind: %v", err)
	}
	if RedisKey("x") != "homechef:x" {
		t.Fatalf("RedisKey(x) = %q", RedisKey("x"))
	}
}
