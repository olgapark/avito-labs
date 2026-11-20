package middleware

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func Metrics(meter metric.Meter, next http.Handler) (http.Handler, error) {
	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10))
	if err != nil {
		return nil, fmt.Errorf("http duration histogram: %w", err)
	}
	active, err := meter.Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		return nil, fmt.Errorf("active requests counter: %w", err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		attrs := []attribute.KeyValue{attribute.String("http.request.method", r.Method), attribute.String("http.route", r.Pattern)}
		active.Add(r.Context(), 1, metric.WithAttributes(attrs...))
		writer := &responseWriter{ResponseWriter: w}
		completed := false
		defer func() {
			active.Add(r.Context(), -1, metric.WithAttributes(attrs...))
			duration.Record(r.Context(), time.Since(started).Seconds(), metric.WithAttributes(append(attrs, attribute.Int("http.response.status_code", writer.finalStatus(completed)))...))
		}()
		next.ServeHTTP(writer, r)
		completed = true
	}), nil
}
