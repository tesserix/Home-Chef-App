package handlers

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	startTime time.Time
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
}

// CheckResult represents a single health check result
type CheckResult struct {
	Status   string        `json:"status"`
	Duration string        `json:"duration,omitempty"`
	Message  string        `json:"message,omitempty"`
	Details  interface{}   `json:"details,omitempty"`
}

// ReadinessStatus represents the readiness status
type ReadinessStatus struct {
	Ready     bool                   `json:"ready"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

// SystemInfo represents system information
type SystemInfo struct {
	Uptime       string `json:"uptime"`
	GoVersion    string `json:"goVersion"`
	NumGoroutine int    `json:"numGoroutine"`
	NumCPU       int    `json:"numCPU"`
	MemStats     MemInfo `json:"memStats"`
}

// MemInfo represents memory statistics
type MemInfo struct {
	Alloc      uint64 `json:"allocMB"`
	TotalAlloc uint64 `json:"totalAllocMB"`
	Sys        uint64 `json:"sysMB"`
	NumGC      uint32 `json:"numGC"`
}

var (
	healthHandler *HealthHandler
	healthOnce    sync.Once
)

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	healthOnce.Do(func() {
		healthHandler = &HealthHandler{
			startTime: time.Now(),
		}
	})
	return healthHandler
}

// Liveness handles the liveness probe (is the application running?)
// GET /health/live
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, HealthStatus{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Service:   "homechef-api",
		Version:   "1.0.0",
	})
}

// Readiness handles the readiness probe (is the application ready to serve traffic?)
// GET /health/ready
func (h *HealthHandler) Readiness(c *gin.Context) {
	checks := make(map[string]CheckResult)
	allHealthy := true

	// Check database connection
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		allHealthy = false
	}

	// Check NATS connection (optional - don't fail readiness if NATS is down)
	natsCheck := h.checkNATS()
	checks["nats"] = natsCheck

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, ReadinessStatus{
		Ready:     allHealthy,
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	})
}

// Health handles the comprehensive health check
// GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	checks := make(map[string]CheckResult)
	allHealthy := true

	// Check database
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		allHealthy = false
	}

	// Check NATS
	natsCheck := h.checkNATS()
	checks["nats"] = natsCheck

	overallStatus := "healthy"
	status := http.StatusOK
	if !allHealthy {
		overallStatus = "unhealthy"
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now().UTC(),
		Service:   "homechef-api",
		Version:   "1.0.0",
		Checks:    checks,
	})
}

// SystemStats returns system statistics
// GET /health/stats
func (h *HealthHandler) SystemStats(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := time.Since(h.startTime)

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"system": SystemInfo{
			Uptime:       uptime.String(),
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			NumCPU:       runtime.NumCPU(),
			MemStats: MemInfo{
				Alloc:      memStats.Alloc / 1024 / 1024,
				TotalAlloc: memStats.TotalAlloc / 1024 / 1024,
				Sys:        memStats.Sys / 1024 / 1024,
				NumGC:      memStats.NumGC,
			},
		},
	})
}

// checkDatabase checks database connectivity
func (h *HealthHandler) checkDatabase() CheckResult {
	start := time.Now()

	sqlDB, err := database.DB.DB()
	if err != nil {
		return CheckResult{
			Status:   "unhealthy",
			Duration: time.Since(start).String(),
			Message:  "Failed to get database instance",
			Details:  err.Error(),
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return CheckResult{
			Status:   "unhealthy",
			Duration: time.Since(start).String(),
			Message:  "Database ping failed",
			Details:  err.Error(),
		}
	}

	// Get connection pool stats
	stats := sqlDB.Stats()

	return CheckResult{
		Status:   "healthy",
		Duration: time.Since(start).String(),
		Message:  "Database connection successful",
		Details: map[string]interface{}{
			"openConnections": stats.OpenConnections,
			"inUse":           stats.InUse,
			"idle":            stats.Idle,
			"maxOpenConns":    stats.MaxOpenConnections,
		},
	}
}

// checkNATS checks NATS connectivity
func (h *HealthHandler) checkNATS() CheckResult {
	start := time.Now()

	natsClient := services.GetNATSClient()
	if !natsClient.IsConnected() {
		return CheckResult{
			Status:   "unhealthy",
			Duration: time.Since(start).String(),
			Message:  "NATS not connected",
		}
	}

	return CheckResult{
		Status:   "healthy",
		Duration: time.Since(start).String(),
		Message:  "NATS connection active",
	}
}
