package database

// pool_config_test.go — backend-testable slice of issue #64 (network resilience
// + cold-start/load). The actual load/burst behaviour is verified under a load
// test, but the connection-pool *sizing* is a silent-regression risk: Postgres
// caps total connections across all pods, so the tuned 20/5 per-pod budget and
// its env override must not drift. These guard the parsing/fallback that decide
// the pool size before any DB is opened.

import (
	"testing"
)

func TestPoolDefaults_StayTuned(t *testing.T) {
	// The "20/5 per pod" budget is a deliberate Cloud SQL db-f1-micro tuning
	// (keeps ~5 pods under the ~100 connection cap). A change here should be
	// intentional, not accidental.
	if defaultMaxOpenConns != 20 {
		t.Errorf("defaultMaxOpenConns drifted: got %d, want 20", defaultMaxOpenConns)
	}
	if defaultMaxIdleConns != 5 {
		t.Errorf("defaultMaxIdleConns drifted: got %d, want 5", defaultMaxIdleConns)
	}
	if defaultMaxIdleConns > defaultMaxOpenConns {
		t.Errorf("idle (%d) must not exceed open (%d)", defaultMaxIdleConns, defaultMaxOpenConns)
	}
}

func TestEnvInt(t *testing.T) {
	const key = "HOMECHEF_TEST_POOL_ENVINT"

	cases := []struct {
		name string
		set  bool
		val  string
		def  int
		want int
	}{
		{"unset falls back to default", false, "", 20, 20},
		{"valid positive override", true, "50", 20, 50},
		{"empty string falls back", true, "", 20, 20},
		{"non-numeric falls back", true, "abc", 20, 20},
		{"zero falls back (must be > 0)", true, "0", 20, 20},
		{"negative falls back (must be > 0)", true, "-5", 20, 20},
		{"whitespace falls back", true, "  ", 5, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(key, "") // ensure clean baseline; Setenv auto-restores
			if tc.set {
				t.Setenv(key, tc.val)
			} else {
				// Not set: rely on the empty Setenv above being treated as unset
				// by envInt (it checks for empty string).
			}
			if got := envInt(key, tc.def); got != tc.want {
				t.Errorf("envInt(%q=%q, def=%d) = %d, want %d", key, tc.val, tc.def, got, tc.want)
			}
		})
	}
}
