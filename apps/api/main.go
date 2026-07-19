// Deploy note: forces a fresh apps/api build so the prod image tag resolves to
// the full v1 backend (#369 fulfillment + #372 delivery-area + #370 weekly-menu).
// The ensure-image-tags carry-forward had tagged the deploy commit with the
// pre-#372 API image because the intermediate deploy-advances were cancelled by
// GitHub Actions concurrency when several PRs merged in quick succession.

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/piicrypto"
	"github.com/homechef/api/database"
	"github.com/homechef/api/handlers"
	"github.com/homechef/api/internal/observability"
	"github.com/homechef/api/logger"
	"github.com/homechef/api/routes"
	"github.com/homechef/api/services"
	"github.com/homechef/api/temporal"
	"github.com/homechef/api/tracing"

	// Aligns GOMAXPROCS with the Knative container CPU limit (cgroup quota)
	// instead of the node's core count — prevents the Go scheduler from
	// spinning up far more OS threads than the pod is allowed to run,
	// which causes CPU throttling and tail-latency spikes.
	_ "go.uber.org/automaxprocs"
)

func main() {
	// Load configuration
	config.Load()

	// Structured JSON logging — initialise before anything else logs so
	// startup lines are machine-parsable in Cloud Logging too.
	logger.Init(config.AppConfig.Environment)
	log.Printf("Starting HomeChef API in %s mode", config.AppConfig.Environment)

	// Init Sentry as early as possible so any startup failure below
	// lands in Sentry rather than vanishing into the pod log. Gated
	// on SENTRY_DSN_API — no-ops cleanly when unset (dev / staging).
	services.InitSentry()
	defer services.FlushSentry(2 * time.Second)

	// OpenTelemetry tracing → Cloud Trace. Degrades to a no-op when there's
	// no GCP project / credentials (local dev), so it never blocks startup.
	otelSample := 0.1
	if v := os.Getenv("OTEL_SAMPLING_RATE"); v != "" {
		if f, perr := strconv.ParseFloat(v, 64); perr == nil {
			otelSample = f
		}
	}
	traceShutdown, terr := tracing.Init(
		context.Background(),
		config.AppConfig.GCSProjectID,
		config.AppConfig.Environment,
		os.Getenv("APP_VERSION"),
		otelSample,
	)
	if terr != nil {
		log.Printf("Warning: tracing init failed: %v", terr)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = traceShutdown(shutdownCtx)
	}()

	// OpenTelemetry traces + metrics over OTLP gRPC to the in-cluster collector.
	// Set up after the Cloud Trace provider so the OTLP provider becomes the
	// active global Tracer/Meter provider. No-ops when OTEL_EXPORTER_OTLP_ENDPOINT
	// is empty, so local dev boots without a collector.
	otelShutdown, oerr := observability.Init(context.Background(), "homechef-api")
	if oerr != nil {
		log.Printf("Warning: observability init failed: %v", oerr)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = otelShutdown(shutdownCtx)
	}()

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Seed baseline per-country tax rules so the invoicing pipeline has
	// something to resolve on day one. Idempotent — existing admin edits
	// via the /admin/tax-rates endpoint survive future restarts.
	services.SeedTaxRates()

	// Seed location reference data (India + 36 states + ~50 major cities +
	// representative PIN codes). Idempotent — relies on ON CONFLICT DO
	// NOTHING so re-runs are free. The schema is country-agnostic; the
	// seeded scope is intentionally India-only for the initial release.
	if err := services.SeedLocations(context.Background(), database.DB); err != nil {
		log.Printf("Warning: Failed to seed location reference data: %v", err)
	}

	// Initialize GCS storage
	if err := services.InitStorage(); err != nil {
		log.Printf("Warning: Failed to initialize GCS storage: %v", err)
		log.Println("File uploads will be unavailable")
	} else {
		defer services.CloseStorage()
	}

	// Cloud Vision OCR — non-fatal. Powers document pre-fill (FSSAI number +
	// expiry); the chef always confirms, so the app works without it.
	if err := services.InitVision(); err != nil {
		log.Printf("Warning: Cloud Vision OCR unavailable: %v", err)
	} else {
		defer services.CloseVision()
	}

	// Initialize GCP Secret Manager
	if err := services.InitSecretManager(); err != nil {
		log.Printf("Warning: Failed to initialize Secret Manager: %v", err)
		log.Println("Vendor payment secrets will be unavailable")
	} else {
		defer services.CloseSecretManager()
	}

	// PII column encryption (#710). Off by default; when enabled, unwrap the DEK
	// (GCP KMS) + load the blind-index key at boot. Fatal if it can't init while
	// enabled — an enabled-but-uninitialised state would silently store plaintext.
	if config.AppConfig.PIIEncryptionEnabled {
		if err := piicrypto.Init(context.Background(), config.AppConfig.GCSProjectID); err != nil {
			log.Fatalf("PII encryption enabled but failed to initialize: %v", err)
		}
		log.Println("PII column encryption initialized")
	}

	// Warm up the Razorpay client. Credentials are fetched lazily from GCP
	// Secret Manager on first use (and refreshed every razorpayCacheTTL), so
	// this is just a best-effort pre-fetch to log any config gap at startup.
	services.InitRazorpay()

	// Same for Stripe — second payment provider for markets where Razorpay
	// isn't available. Both clients coexist; per-chef PaymentProvider
	// decides which one handles a given order.
	services.InitStripe()

	// Initialize email service (SendGrid primary, Resend fallback)
	services.InitEmailService()

	// Initialize FCM push notification service
	if err := services.InitPushService(); err != nil {
		log.Printf("Warning: Failed to initialize push service: %v", err)
	}

	// Initialize Temporal (durable execution). Opt-in: only dials when
	// TEMPORAL_HOSTPORT is set, so the API boots normally with inline fallbacks
	// until the shared cluster (#119) is deployed. Workers run as a separate
	// process (cmd/worker); the API is the producer that starts workflows.
	if tcfg := temporal.LoadConfig(); tcfg.Enabled() {
		if rt, err := temporal.NewRuntime(); err != nil {
			log.Printf("Warning: Temporal enabled but connect failed: %v — using inline fallbacks", err)
		} else {
			services.SetTemporalRuntime(rt)
			defer rt.Close()
			log.Printf("Temporal connected (namespace=%s)", tcfg.Namespace)
		}
	}

	// Connect to Redis
	redisClient := services.GetRedisClient()
	if err := redisClient.Connect(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Println("Redis caching will be unavailable")
	} else {
		defer redisClient.Close()
	}

	// Install the delivery distance router (#701). Uses Google Routes when a Maps
	// API key is configured (cached in Redis + Postgres so it's billed at most once
	// per trip), else the haversine fallback. Must run after Redis + DB are up.
	services.InitDeliveryRouter()
	// Weather surge provider (#706) — Google Weather when keyed, else neutral.
	services.InitWeatherProvider()
	// Traffic surge provider (#705) — reuses the Routes key, else neutral.
	services.InitTrafficProvider()

	// Connect to MongoDB (in-app chat + document uploads, #53). Optional: a
	// failure leaves the Mongo-backed features disabled without affecting the
	// rest of the API.
	mongoClient := services.GetMongoClient()
	if err := mongoClient.Connect(); err != nil {
		log.Printf("Warning: Failed to connect to MongoDB: %v — chat/upload-on-Mongo unavailable", err)
	} else {
		defer mongoClient.Close(context.Background())
		// Best-effort: messaging indexes (idempotent) for the mediated chat (#53).
		if err := services.EnsureMessagingIndexes(context.Background()); err != nil {
			log.Printf("Warning: messaging index setup failed: %v", err)
		}
	}

	// Connect to NATS
	natsClient := services.GetNATSClient()
	if err := natsClient.Connect(); err != nil {
		log.Printf("Warning: Failed to connect to NATS: %v", err)
		log.Println("NATS messaging will be unavailable")
	} else {
		defer natsClient.Close()

		// Background context for the reliable event backbone (outbox relay +
		// durable consumers). Cancelled on shutdown to stop the loops cleanly.
		eventCtx, eventCancel := context.WithCancel(context.Background())
		defer eventCancel()

		// Transactional outbox relay: durably publishes staged events to JetStream
		// with PubAck, so events are never lost between commit and publish (#131).
		services.NewOutboxRelay(database.DB, natsClient).Start(eventCtx)

		// Shared durable-consumer manager: notification + push workers register on
		// it (ack/retry/idempotency/DLQ). One Stop() drains every consume loop.
		consumerManager := services.NewConsumerManager(natsClient, database.DB)
		defer consumerManager.Stop()

		// Start notification service (durable "notify-*" consumers)
		notificationService := services.GetNotificationService()
		if err := notificationService.Start(consumerManager); err != nil {
			log.Printf("Warning: Failed to start notification service: %v", err)
		} else {
			defer notificationService.Stop()
		}

		// Register push consumers (NATS → FCM bridge for vendor/driver/customer pushes)
		if err := handlers.RegisterPushConsumers(eventCtx, consumerManager); err != nil {
			log.Printf("Warning: Failed to register push consumers: %v", err)
		}
	}

	// Background daily scan: ping chefs whose FSSAI license expires
	// in 30/15/7 days. Runs under the root context so SIGTERM stops
	// it cleanly with the rest of the service. Idempotent across pods
	// via Redis SETNX (or fails open on Redis outage).
	cronCtx, cronCancel := context.WithCancel(context.Background())
	defer cronCancel()
	// Scheduled jobs: weekly settlement statements, daily reconciliation, FSSAI
	// expiry reminders, kitchen auto-resume, and audit-log retention. When
	// Temporal is enabled these run as durable Schedules (exactly-once,
	// leader-elected, catching up on windows missed during downtime); otherwise
	// the legacy in-process tickers run unchanged. See #116 / #125.
	services.StartCronJobs(cronCtx)

	// Keep the homechef_fssai_locked_chefs gauge current (#94) — recounts
	// locked chefs every 10 min so dashboards/alerts can watch the lockout.
	// (Prometheus gauge updater, not a Temporal-scheduled job.)
	services.StartFSSAILockedGaugeUpdater(cronCtx)

	// One-time-ish: geocode chefs missing coordinates. Background so boot isn't
	// blocked by Photon latency.
	go services.BackfillChefCoordinates(database.DB)

	// Setup router
	router := routes.SetupRouter()

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + config.AppConfig.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on port %s", config.AppConfig.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited gracefully")
}
