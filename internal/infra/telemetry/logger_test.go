package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/olgapark/avito-labs/internal/infra/telemetry"
)

func TestLogger(t *testing.T) {
	var output bytes.Buffer
	logger := telemetry.NewLogger(&output, slog.LevelInfo, "trip-service")
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID}))
	logger.DebugContext(ctx, "hidden")
	logger.With("trip_id", "test-trip").InfoContext(ctx, "trip created")
	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["trace_id"] != traceID.String() || record["span_id"] != spanID.String() || record["service"] != "trip-service" || record["trip_id"] != "test-trip" || record["level"] != "INFO" {
		t.Fatalf("wrong log: %+v", record)
	}
}
