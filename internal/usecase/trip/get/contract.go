//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package get_test

package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

type tripRepository interface {
	Fetch(ctx context.Context, id uuid.UUID) (model.Trip, error)
}
