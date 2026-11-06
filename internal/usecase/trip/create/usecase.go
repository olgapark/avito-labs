package create

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/model"
)

type Usecase struct {
	repo    tripRepository
	tx      txManager
	logger  *slog.Logger
	metrics metrics
	now     func() time.Time
	newID   func() uuid.UUID
}

func New(repo tripRepository, tx txManager, logger *slog.Logger, metrics metrics, now func() time.Time, newID func() uuid.UUID) *Usecase {
	return &Usecase{repo: repo, tx: tx, logger: logger, metrics: metrics, now: now, newID: newID}
}

func (u *Usecase) Execute(ctx context.Context, input model.CreateTrip) (model.Trip, error) {
	created, err := trip.New(input, u.newID(), u.now())
	if err != nil {
		return model.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.repo.Save(ctx, created); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}
		if err := u.repo.AddHistory(ctx, created, nil); err != nil {
			return fmt.Errorf("save initial status: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("create trip transaction: %w", err)
	}
	u.metrics.Created(ctx)
	u.logger.InfoContext(ctx, "trip created", "trip_id", created.ID)
	return created, nil
}
