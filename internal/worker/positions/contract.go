//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package positions_test

package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

type repository interface {
	ListStaleTrips(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error)
}

type pushService interface {
	RequestPosition(ctx context.Context, tripID, driverID uuid.UUID) error
}

type metrics interface {
	StaleCheck(ctx context.Context, result string)
	RequestDone(ctx context.Context, result string)
	PushDuration(ctx context.Context, result string, elapsed time.Duration)
}

type Ticker interface {
	Ticks() <-chan time.Time
	Stop()
}
