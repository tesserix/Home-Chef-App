package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/routes"
	"github.com/homechef/api/services"
)

func main() {
	// Load configuration
	config.Load()
	log.Printf("Starting HomeChef API in %s mode", config.AppConfig.Environment)

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
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

	// Initialize Razorpay
	services.InitRazorpay()

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
	}

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
