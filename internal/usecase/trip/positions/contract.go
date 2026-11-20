package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

type tripRepository interface {
	Fetch(ctx context.Context, id uuid.UUID) (model.Trip, error)
	FetchForUpdate(ctx context.Context, id uuid.UUID) (model.Trip, error)
	InsertPosition(ctx context.Context, position model.Position) (model.Position, error)
	UpdateLastPosition(ctx context.Context, id uuid.UUID, recordedAt time.Time) error
	ListPositions(ctx context.Context, id uuid.UUID) ([]model.Position, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type positionRequests interface {
	Cancel(ctx context.Context, tripID uuid.UUID)
}

type metrics interface {
	Saved(ctx context.Context)
}
