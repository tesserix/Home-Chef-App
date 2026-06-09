package services

import (
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// sentryInitialized is flipped after a successful sentry.Init so the
// middleware + flush helpers can no-op cleanly when the DSN was unset
// (dev / staging without telemetry) instead of throwing.
var sentryInitialized bool

// InitSentry sets up the global Sentry hub when SENTRY_DSN_API is set.
// Safe to call even when the DSN is missing — it just no-ops.
//
// Env:
//   - SENTRY_DSN_API         project DSN for homechef-api
//   - SENTRY_ENVIRONMENT     production / staging / dev (defaults to ENVIRONMENT)
//
// Sample rate is 10% for traces (stays under the 5k errors/mo free tier
// cap while keeping enough perf signal to spot regressions). All
// panics + captured errors land at 100% regardless.
func InitSentry() {
	dsn := os.Getenv("SENTRY_DSN_API")
	if dsn == "" {
		log.Println("SENTRY_DSN_API not set — Sentry disabled")
		return
	}
	env := os.Getenv("SENTRY_ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "development"
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          os.Getenv("APP_VERSION"),
		AttachStacktrace: true,
		EnableTracing:    true,
		TracesSampleRate: 0.1,
		// PII is off by default — we never auto-attach IPs, headers,
		// or request bodies. Specific handlers can sentry.WithScope
		// and attach user context explicitly after authentication.
		SendDefaultPII: false,
	})
	if err != nil {
		log.Printf("Sentry init failed: %v", err)
		return
	}
	sentryInitialized = true
	log.Printf("Sentry initialized (env=%s)", env)
}

// FlushSentry drains pending events. Call before shutdown so the last
// few errors don't get dropped when the pod terminates.
func FlushSentry(timeout time.Duration) {
	if !sentryInitialized {
		return
	}
	sentry.Flush(timeout)
}

// SentryGinMiddleware returns the Gin handler that recovers panics +
// captures them into the per-request Hub. No-ops when Sentry isn't
// initialized so dev builds don't pay the cost.
//
// Repanic: true so Gin's default recovery still produces the 500 + log.
// Without it the request would just hang on a swallowed panic.
func SentryGinMiddleware() gin.HandlerFunc {
	if !sentryInitialized {
		return func(c *gin.Context) { c.Next() }
	}
	return sentrygin.New(sentrygin.Options{
		Repanic:         true,
		WaitForDelivery: false,
		Timeout:         2 * time.Second,
	})
}

// CaptureSentryError is a thin helper for handlers that catch errors
// they want to surface in Sentry without converting to a panic. No-op
// when Sentry isn't initialized.
func CaptureSentryError(c *gin.Context, err error) {
	if !sentryInitialized || err == nil {
		return
	}
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.CaptureException(err)
		return
	}
	sentry.CaptureException(err)
}
