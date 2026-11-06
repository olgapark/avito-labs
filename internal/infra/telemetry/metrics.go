package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type TripMetrics struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewTripMetrics(meter metric.Meter, trips activeTrips) (*TripMetrics, error) {
	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		return nil, fmt.Errorf("create trips counter: %w", err)
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		return nil, fmt.Errorf("complete trips counter: %w", err)
	}
	_, err = meter.Int64ObservableGauge("tripgo.trips.active", metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
		count, err := trips.CountActive(ctx)
		if err != nil {
			return err
		}
		observer.Observe(count)
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("active trips gauge: %w", err)
	}
	return &TripMetrics{created: created, completed: completed}, nil
}

func (m *TripMetrics) Created(ctx context.Context)   { m.created.Add(ctx, 1) }
func (m *TripMetrics) Completed(ctx context.Context) { m.completed.Add(ctx, 1) }
