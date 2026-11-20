package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type PositionMetrics struct {
	saved    metric.Int64Counter
	checks   metric.Int64Counter
	requests metric.Int64Counter
	duration metric.Float64Histogram
}

func NewPositionMetrics(meter metric.Meter) (*PositionMetrics, error) {
	saved, err := meter.Int64Counter("tripgo.positions.saved")
	if err != nil {
		return nil, fmt.Errorf("saved positions counter: %w", err)
	}
	checks, err := meter.Int64Counter("tripgo.stale_checks")
	if err != nil {
		return nil, fmt.Errorf("stale check counter: %w", err)
	}
	requests, err := meter.Int64Counter("tripgo.position_requests")
	if err != nil {
		return nil, fmt.Errorf("position request counter: %w", err)
	}
	duration, err := meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10))
	if err != nil {
		return nil, fmt.Errorf("position request duration: %w", err)
	}
	return &PositionMetrics{saved: saved, checks: checks, requests: requests, duration: duration}, nil
}

func ObservePositionPool(meter metric.Meter, pool positionPool) error {
	_, err := meter.Int64ObservableGauge("tripgo.position_requests.queue.size",
		metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
			observer.Observe(pool.QueueSize())
			return nil
		}))
	if err != nil {
		return fmt.Errorf("position queue size gauge: %w", err)
	}
	_, err = meter.Int64ObservableGauge("tripgo.position_requests.workers.active",
		metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
			observer.Observe(pool.ActiveWorkers())
			return nil
		}))
	if err != nil {
		return fmt.Errorf("active position workers gauge: %w", err)
	}
	return nil
}

func (m *PositionMetrics) Saved(ctx context.Context) { m.saved.Add(ctx, 1) }

func (m *PositionMetrics) StaleCheck(ctx context.Context, result string) {
	m.checks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *PositionMetrics) RequestDone(ctx context.Context, result string) {
	m.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *PositionMetrics) PushDuration(ctx context.Context, result string, elapsed time.Duration) {
	m.duration.Record(ctx, elapsed.Seconds(), metric.WithAttributes(attribute.String("result", result)))
}
