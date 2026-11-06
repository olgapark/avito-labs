package finish

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

type Usecase struct {
	repo    tripRepository
	tx      txManager
	logger  *slog.Logger
	metrics metrics
	now     func() time.Time
}

func New(repo tripRepository, tx txManager, logger *slog.Logger, metrics metrics, now func() time.Time) *Usecase {
	return &Usecase{repo: repo, tx: tx, logger: logger, metrics: metrics, now: now}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	var result model.Trip
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		current, err := u.repo.FetchForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("fetch trip for finish: %w", err)
		}
		fromStatus := current.Status
		if err := current.Finish(u.now()); err != nil {
			return err
		}
		if err := u.repo.Update(ctx, current); err != nil {
			return fmt.Errorf("update trip: %w", err)
		}
		if err := u.repo.AddHistory(ctx, current, &fromStatus); err != nil {
			return fmt.Errorf("save completed status: %w", err)
		}
		result = current
		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("finish trip transaction: %w", err)
	}
	u.metrics.Completed(ctx)
	u.logger.InfoContext(ctx, "trip completed", "trip_id", result.ID)
	return result, nil
}
