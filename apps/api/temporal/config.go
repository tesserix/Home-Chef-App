// Package temporal wires the HomeChef API into the shared tesserix Temporal
// platform (durable execution). It provides client/worker bootstrap, sane retry
// defaults, and task-queue conventions. See epic tesserix/Home-Chef-App#116.
//
// This package is intentionally self-contained; once the shared go-shared/temporal
// package (issue #120) is published it will be extracted there unchanged.
package temporal

import "os"

// Config holds the connection settings for the Temporal frontend. Defaults are
// the local dev cluster; production values come from env (External Secrets).
type Config struct {
	HostPort   string // TEMPORAL_HOSTPORT (e.g. temporal-frontend.temporal-system:7233)
	Namespace  string // TEMPORAL_NAMESPACE (one per product; HomeChef = "homechef")
	TLSEnabled bool   // TEMPORAL_TLS=true for mTLS / Temporal Cloud
}

// LoadConfig reads Temporal settings from the environment with dev-safe defaults.
func LoadConfig() Config {
	return Config{
		HostPort:   env("TEMPORAL_HOSTPORT", "localhost:7233"),
		Namespace:  env("TEMPORAL_NAMESPACE", "homechef"),
		TLSEnabled: os.Getenv("TEMPORAL_TLS") == "true",
	}
}

// Enabled reports whether Temporal is configured for this process. The API can
// boot without Temporal during the incremental migration (workers run separately).
func (c Config) Enabled() bool { return c.HostPort != "" }

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
