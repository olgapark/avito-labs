//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package finish_test

package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

type usecase interface {
	Execute(ctx context.Context, id uuid.UUID) (model.Trip, error)
}
