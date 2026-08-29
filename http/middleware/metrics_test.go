package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMetricsMiddlewareRecordsRequest(t *testing.T) {
	t.Parallel()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	defer provider.Shutdown(t.Context())

	middleware, err := MetricsMiddleware(provider)
	if err != nil {
		t.Fatalf("MetricsMiddleware() error = %v", err)
	}

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}

	var data metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &data); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	count := countMetric(data, "http_requests_total")
	if count != 1 {
		t.Errorf("http_requests_total = %d, want 1", count)
	}

	duration := countMetric(data, "http_request_duration_seconds")
	if duration != 1 {
		t.Errorf("http_request_duration_seconds observations = %d, want 1", duration)
	}
}

func countMetric(data metricdata.ResourceMetrics, name string) int {
	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name == name {
				switch v := m.Data.(type) {
				case metricdata.Sum[int64]:
					total := 0
					for _, dp := range v.DataPoints {
						total += int(dp.Value)
					}
					return total
				case metricdata.Histogram[float64]:
					total := 0
					for _, dp := range v.DataPoints {
						total += int(dp.Count)
					}
					return total
				}
			}
		}
	}
	return 0
}
