package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

type Usecase struct{ repo tripRepository }

func New(repo tripRepository) *Usecase { return &Usecase{repo: repo} }

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	result, err := u.repo.Fetch(ctx, id)
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return result, nil
}
