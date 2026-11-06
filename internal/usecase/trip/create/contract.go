//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package create_test

package create

import (
	"context"

	"github.com/olgapark/avito-labs/internal/model"
)

type tripRepository interface {
	Save(ctx context.Context, trip model.Trip) error
	AddHistory(ctx context.Context, trip model.Trip, fromStatus *string) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type metrics interface {
	Created(ctx context.Context)
}
