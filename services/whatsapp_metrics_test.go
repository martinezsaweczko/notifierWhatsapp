package services

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

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

func TestWhatsAppClientSendInvalidRecipientRecordsMetric(t *testing.T) {
	t.Parallel()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	defer provider.Shutdown(t.Context())

	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	client, err := NewWhatsAppClient(ctx, dbPath, slog.New(slog.NewTextHandler(os.Stdout, nil)), provider)
	if err != nil {
		t.Fatalf("NewWhatsAppClient() error = %v", err)
	}
	defer client.Close()

	_, err = client.Send(ctx, "invalid", "hello")
	if err == nil {
		t.Fatal("Send() expected error for invalid recipient")
	}

	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	count := countMetric(data, "whatsapp_messages_sent_total")
	if count != 1 {
		t.Errorf("whatsapp_messages_sent_total = %d, want 1", count)
	}

	duration := countMetric(data, "whatsapp_send_duration_seconds")
	if duration != 1 {
		t.Errorf("whatsapp_send_duration_seconds observations = %d, want 1", duration)
	}
}
