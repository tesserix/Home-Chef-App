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
	"github.com/homechef/api/database"
	"github.com/homechef/api/handlers"
	"github.com/homechef/api/logger"
	"github.com/homechef/api/routes"
	"github.com/homechef/api/services"
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

	// Initialize GCP Secret Manager
	if err := services.InitSecretManager(); err != nil {
		log.Printf("Warning: Failed to initialize Secret Manager: %v", err)
		log.Println("Vendor payment secrets will be unavailable")
	} else {
		defer services.CloseSecretManager()
	}

	// Warm up the Razorpay client. Credentials are fetched lazily from GCP
	// Secret Manager on first use (and refreshed every razorpayCacheTTL), so
	// this is just a best-effort pre-fetch to log any config gap at startup.
	services.InitRazorpay()

	// Same for Stripe — second payment provider for markets where Razorpay
	// isn't available. Both clients coexist; per-chef PaymentProvider
	// decides which one handles a given order.
	services.InitStripe()

	// Initialize SendGrid email service
	services.InitEmailService()

	// Initialize FCM push notification service
	if err := services.InitPushService(); err != nil {
		log.Printf("Warning: Failed to initialize push service: %v", err)
	}

	// Connect to Redis
	redisClient := services.GetRedisClient()
	if err := redisClient.Connect(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Println("Redis caching will be unavailable")
	} else {
		defer redisClient.Close()
	}

	// Connect to NATS
	natsClient := services.GetNATSClient()
	if err := natsClient.Connect(); err != nil {
		log.Printf("Warning: Failed to connect to NATS: %v", err)
		log.Println("NATS messaging will be unavailable")
	} else {
		defer natsClient.Close()

		// Start notification service
		notificationService := services.GetNotificationService()
		if err := notificationService.Start(); err != nil {
			log.Printf("Warning: Failed to start notification service: %v", err)
		} else {
			defer notificationService.Stop()
		}

		// Register push consumers (NATS → FCM bridge for vendor/driver/customer pushes)
		handlers.RegisterPushConsumers()
	}

	// Background daily scan: ping chefs whose FSSAI license expires
	// in 30/15/7 days. Runs under the root context so SIGTERM stops
	// it cleanly with the rest of the service. Idempotent across pods
	// via Redis SETNX (or fails open on Redis outage).
	cronCtx, cronCancel := context.WithCancel(context.Background())
	defer cronCancel()
	services.StartFSSAIReminderCron(cronCtx)

	// Background weekly settlement statements: once a Mon–Sun week (IST)
	// closes, issue an immutable statement + "ready" push per chef. Shares
	// the cron's lifecycle/idempotency model with the FSSAI reminder.
	services.StartWeeklyStatementCron(cronCtx)

	// Background settlement reconciliation: daily, cross-check the previous
	// day's payment/refund records against Razorpay/Stripe and alert on drift.
	// Read-only — surfaces discrepancies for finance ops, never mutates.
	services.StartReconciliationCron(cronCtx)

	// Auto-resume timed kitchen pauses ("Back in {15,30,60} min") once the
	// window elapses. 1-minute granularity; reliable under min-scale:1.
	services.StartAvailabilityResumeCron(cronCtx)

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
