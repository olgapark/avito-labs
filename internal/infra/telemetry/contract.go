package telemetry

import "context"

type activeTrips interface {
	CountActive(ctx context.Context) (int64, error)
}

type positionPool interface {
	QueueSize() int64
	ActiveWorkers() int64
}
