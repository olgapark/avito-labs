package positions

import (
	"context"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

type usecase interface {
	Save(ctx context.Context, id uuid.UUID, input model.PositionData) (model.Position, error)
	List(ctx context.Context, id uuid.UUID) ([]model.Position, error)
}
