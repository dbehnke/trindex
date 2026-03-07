// Package observability provides structured logging, metrics, and tracing
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Metrics
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	requestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// Memory operation metrics
	memoryOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memory_operations_total",
			Help: "Total memory operations",
		},
		[]string{"operation", "namespace"},
	)

	memoryOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "memory_operation_duration_seconds",
			Help:    "Memory operation duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
		},
		[]string{"operation"},
	)

	// Database metrics
	dbConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_connections",
			Help: "Database connection pool statistics",
		},
		[]string{"state"},
	)

	// MCP metrics
	mcpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_requests_total",
			Help: "Total MCP tool requests",
		},
		[]string{"tool"},
	)

	mcpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_request_duration_seconds",
			Help:    "MCP tool request duration",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"tool"},
	)
)

func init() {
	// Register all metrics
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(requestTotal)
	prometheus.MustRegister(activeConnections)
	prometheus.MustRegister(memoryOperations)
	prometheus.MustRegister(memoryOperationDuration)
	prometheus.MustRegister(dbConnections)
	prometheus.MustRegister(mcpRequests)
	prometheus.MustRegister(mcpRequestDuration)
}

// contextKey for storing request ID in context
type contextKey string

const requestIDKey contextKey = "request_id"

// InitLogger configures structured logging with the specified level
func InitLogger(level string) {
	var logLevel slog.Level

	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: logLevel == slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("logger initialized", "level", level)
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggingMiddleware logs HTTP requests with structured fields
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := GetRequestID(r.Context())

		// Wrap response writer to capture status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		activeConnections.Inc()
		defer activeConnections.Dec()

		next.ServeHTTP(ww, r.WithContext(r.Context()))

		duration := time.Since(start)
		status := ww.Status()

		// Record metrics
		requestDuration.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", status)).Observe(duration.Seconds())
		requestTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", status)).Inc()

		// Log with appropriate level based on status
		logAttrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"request_id", requestID,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		}

		if status >= 500 {
			slog.Error("http request error", logAttrs...)
		} else if status >= 400 {
			slog.Warn("http request warning", logAttrs...)
		} else {
			slog.Debug("http request completed", logAttrs...)
		}
	})
}

// MetricsHandler returns the Prometheus metrics endpoint handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RecordMemoryOperation records a memory operation metric
func RecordMemoryOperation(operation, namespace string) {
	memoryOperations.WithLabelValues(operation, namespace).Inc()
}

// RecordMemoryOperationDuration records the duration of a memory operation
func RecordMemoryOperationDuration(operation string, duration time.Duration) {
	memoryOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordDBConnections updates database connection metrics
func RecordDBConnections(open, idle int32) {
	dbConnections.WithLabelValues("open").Set(float64(open))
	dbConnections.WithLabelValues("idle").Set(float64(idle))
}

// RecordMCPRequest records an MCP tool request
func RecordMCPRequest(tool string) {
	mcpRequests.WithLabelValues(tool).Inc()
}

// RecordMCPRequestDuration records MCP request duration
func RecordMCPRequestDuration(tool string, duration time.Duration) {
	mcpRequestDuration.WithLabelValues(tool).Observe(duration.Seconds())
}

// LoggerWithRequestID returns a logger with request ID attached
func LoggerWithRequestID(ctx context.Context) *slog.Logger {
	requestID := GetRequestID(ctx)
	if requestID != "" {
		return slog.With("request_id", requestID)
	}
	return slog.Default()
}
