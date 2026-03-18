package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "homechef_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "homechef_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestSize = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "homechef_http_request_size_bytes",
			Help: "HTTP request size in bytes",
		},
		[]string{"method", "path"},
	)

	httpResponseSize = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "homechef_http_response_size_bytes",
			Help: "HTTP response size in bytes",
		},
		[]string{"method", "path"},
	)

	// Active connections
	httpActiveRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "homechef_http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	// Business metrics
	ordersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "homechef_orders_total",
			Help: "Total number of orders",
		},
		[]string{"status"},
	)

	orderValue = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "homechef_order_value_dollars",
			Help:    "Order value in dollars",
			Buckets: []float64{10, 25, 50, 75, 100, 150, 200, 300, 500},
		},
	)

	usersRegistered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "homechef_users_registered_total",
			Help: "Total number of registered users",
		},
	)

	chefSignups = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "homechef_chef_signups_total",
			Help: "Total number of chef signups",
		},
	)

	authAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "homechef_auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"type", "success"},
	)

	// Database metrics
	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "homechef_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "homechef_db_connections_active",
			Help: "Number of active database connections",
		},
	)
)

// PrometheusMiddleware collects HTTP metrics
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics endpoint to avoid infinite loop
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		httpActiveRequests.Inc()

		// Request size
		reqSize := float64(c.Request.ContentLength)
		if reqSize < 0 {
			reqSize = 0
		}

		c.Next()

		httpActiveRequests.Dec()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Record metrics
		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path, status).Observe(duration)
		httpRequestSize.WithLabelValues(c.Request.Method, path).Observe(reqSize)
		httpResponseSize.WithLabelValues(c.Request.Method, path).Observe(float64(c.Writer.Size()))
	}
}

// Business metric helpers

// RecordOrder records an order metric
func RecordOrder(status string, value float64) {
	ordersTotal.WithLabelValues(status).Inc()
	if value > 0 {
		orderValue.Observe(value)
	}
}

// RecordUserRegistration records a user registration
func RecordUserRegistration() {
	usersRegistered.Inc()
}

// RecordChefSignup records a chef signup
func RecordChefSignup() {
	chefSignups.Inc()
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(authType string, success bool) {
	successStr := "false"
	if success {
		successStr = "true"
	}
	authAttempts.WithLabelValues(authType, successStr).Inc()
}

// RecordDBQuery records a database query duration
func RecordDBQuery(operation string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// UpdateDBConnections updates the active database connections gauge
func UpdateDBConnections(count int) {
	dbConnectionsActive.Set(float64(count))
}
