//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package create_test

package create

import (
	"context"

	"github.com/olgapark/avito-labs/internal/model"
)

type usecase interface {
	Execute(ctx context.Context, input model.CreateTrip) (model.Trip, error)
}
