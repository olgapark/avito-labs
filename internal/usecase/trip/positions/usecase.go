package positions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/model"
)

type Usecase struct {
	repo     tripRepository
	tx       txManager
	requests positionRequests
	metrics  metrics
	now      func() time.Time
}

func New(repo tripRepository, tx txManager, requests positionRequests, metrics metrics, now func() time.Time) *Usecase {
	return &Usecase{repo: repo, tx: tx, requests: requests, metrics: metrics, now: now}
}

func (u *Usecase) Save(ctx context.Context, id uuid.UUID, input model.PositionData) (model.Position, error) {
	if err := input.Validate(u.now()); err != nil {
		return model.Position{}, err
	}
	var saved model.Position
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		current, err := u.repo.FetchForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if current.Status != trip.StatusActive {
			return trip.ErrTripCompleted
		}
		saved, err = u.repo.InsertPosition(ctx, model.Position{TripID: id, PositionData: input})
		if err != nil {
			return err
		}
		return u.repo.UpdateLastPosition(ctx, id, saved.RecordedAt)
	})
	if err != nil {
		return model.Position{}, fmt.Errorf("save position transaction: %w", err)
	}
	u.requests.Cancel(ctx, id)
	u.metrics.Saved(ctx)
	return saved, nil
}

func (u *Usecase) List(ctx context.Context, id uuid.UUID) ([]model.Position, error) {
	if _, err := u.repo.Fetch(ctx, id); err != nil {
		return nil, fmt.Errorf("fetch trip for route: %w", err)
	}
	return u.repo.ListPositions(ctx, id)
}
