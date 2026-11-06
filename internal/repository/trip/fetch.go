package trip

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/model"
)

const selectTrip = `SELECT id, user_id, driver_id, start_latitude, start_longitude,
	end_latitude, end_longitude, price, status, started_at, finished_at FROM trips WHERE id = $1`

func (r *Repository) Fetch(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return scanTrip(r.executor(ctx).QueryRow(queryCtx, selectTrip, id))
}

func (r *Repository) FetchForUpdate(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return scanTrip(r.executor(ctx).QueryRow(queryCtx, selectTrip+" FOR UPDATE", id))
}

func (r *Repository) CountActive(ctx context.Context) (int64, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	var count int64
	if err := r.executor(ctx).QueryRow(queryCtx, `SELECT count(*) FROM trips WHERE status = 'active'`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active trips: %w", err)
	}
	return count, nil
}
