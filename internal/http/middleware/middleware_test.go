package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/http/domainerror"
	"github.com/olgapark/avito-labs/internal/http/middleware"
)

func TestMiddleware(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		panic  bool
		status int
		level  string
	}{
		{"success", nil, false, http.StatusOK, "INFO"},
		{"client error", trip.ErrTripNotFound, false, http.StatusNotFound, "INFO"},
		{"internal error", errors.New("database secret"), false, http.StatusInternalServerError, "ERROR"},
		{"panic", nil, true, http.StatusInternalServerError, "ERROR"},
	} {
		t.Run(test.name, func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			traces := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
			t.Cleanup(func() {
				if err := traces.Shutdown(context.Background()); err != nil {
					t.Error(err)
				}
			})
			reader := sdkmetric.NewManualReader()
			meters := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() {
				if err := meters.Shutdown(context.Background()); err != nil {
					t.Error(err)
				}
			})
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			router := chi.NewRouter()
			router.Get("/api/v1/trips/{tripId}", func(w http.ResponseWriter, r *http.Request) {
				if test.panic {
					panic("broken handler")
				}
				if test.err != nil {
					domainerror.Write(w, r, test.err)
					return
				}
				w.WriteHeader(http.StatusOK)
			})
			handler, err := middleware.Metrics(meters.Meter("test"), router)
			if err != nil {
				t.Fatal(err)
			}
			handler = middleware.Recover(logger, middleware.Tracing(router, traces, middleware.Logging(logger, handler)))
			request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/5cb72c04-7650-45c9-a79b-bcdba0631e0c", nil)
			request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || strings.Contains(response.Body.String(), "secret") {
				t.Fatalf("wrong response: %d %s", response.Code, response.Body.String())
			}
			assertRequestLog(t, logs.Bytes(), test.status, test.level)
			spans := exporter.GetSpans()
			if len(spans) != 1 {
				t.Fatalf("want one HTTP span, got %d", len(spans))
			}
			span := spans[0]
			if span.Name != "GET /api/v1/trips/{tripId}" || span.SpanContext.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
				t.Fatalf("wrong span: %+v", span)
			}
			if test.status >= 500 && span.Status.Code != codes.Error {
				t.Fatalf("error span not marked: %+v", span.Status)
			}
			var data metricdata.ResourceMetrics
			if err := reader.Collect(t.Context(), &data); err != nil {
				t.Fatal(err)
			}
			assertHTTPMetrics(t, data, test.status)
		})
	}
}

func assertRequestLog(t *testing.T, data []byte, status int, level string) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	count := 0
	for decoder.More() {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			t.Fatal(err)
		}
		if record["msg"] != "request handled" {
			continue
		}
		count++
		if record["level"] != level || record["status"] != float64(status) || record["route"] != "/api/v1/trips/{tripId}" {
			t.Fatalf("wrong request log: %+v", record)
		}
	}
	if count != 1 {
		t.Fatalf("want one request log, got %d", count)
	}
}

func assertHTTPMetrics(t *testing.T, data metricdata.ResourceMetrics, status int) {
	t.Helper()
	foundDuration, foundActive := false, false
	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			switch m.Name {
			case "http.server.request.duration":
				points := m.Data.(metricdata.Histogram[float64]).DataPoints
				if len(points) != 1 || points[0].Count != 1 {
					t.Fatalf("wrong duration: %+v", points)
				}
				value, _ := points[0].Attributes.Value(attribute.Key("http.response.status_code"))
				if value.AsInt64() != int64(status) {
					t.Fatalf("wrong metric status: %v", value)
				}
				route, _ := points[0].Attributes.Value(attribute.Key("http.route"))
				if route.AsString() != "/api/v1/trips/{tripId}" {
					t.Fatalf("wrong metric route: %v", route)
				}
				foundDuration = true
			case "http.server.active_requests":
				points := m.Data.(metricdata.Sum[int64]).DataPoints
				if len(points) != 1 || points[0].Value != 0 {
					t.Fatalf("active requests leaked: %+v", points)
				}
				foundActive = true
			}
		}
	}
	if !foundDuration || !foundActive {
		t.Fatalf("missing metrics: duration=%v, active=%v", foundDuration, foundActive)
	}
}
