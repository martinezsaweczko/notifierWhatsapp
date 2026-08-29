package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsMiddleware records request count and duration for all routes.
func MetricsMiddleware(meterProvider metric.MeterProvider) (Middleware, error) {
	meter := meterProvider.Meter("http")

	requestCount, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &wrappedResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()
			attrs := metric.WithAttributes(
				attribute.String("method", r.Method),
				attribute.String("path", r.URL.Path),
				attribute.Int("status", wrapped.statusCode),
			)

			requestCount.Add(r.Context(), 1, attrs)
			requestDuration.Record(r.Context(), duration, attrs)
		})
	}, nil
}
