//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package finish_test

package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

type tripRepository interface {
	FetchForUpdate(ctx context.Context, id uuid.UUID) (model.Trip, error)
	Update(ctx context.Context, trip model.Trip) error
	AddHistory(ctx context.Context, trip model.Trip, fromStatus *string) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type metrics interface {
	Completed(ctx context.Context)
}
