package trip

import (
	"context"
	"fmt"

	"github.com/olgapark/avito-labs/internal/model"
)

func (r *Repository) Save(ctx context.Context, trip model.Trip) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	_, err := r.executor(ctx).Exec(queryCtx, `INSERT INTO trips
		(id, user_id, driver_id, start_latitude, start_longitude, end_latitude,
		end_longitude, price, status, started_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		trip.ID, trip.UserID, trip.DriverID, trip.StartLatitude, trip.StartLongitude,
		trip.EndLatitude, trip.EndLongitude, trip.Price, trip.Status, trip.StartedAt)
	if err != nil {
		return saveError(err)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, trip model.Trip) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	_, err := r.executor(ctx).Exec(queryCtx, `UPDATE trips SET status = $2,
		finished_at = $3, updated_at = $3 WHERE id = $1`, trip.ID, trip.Status, trip.FinishedAt)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	return nil
}

func (r *Repository) AddHistory(ctx context.Context, trip model.Trip, fromStatus *string) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	_, err := r.executor(ctx).Exec(queryCtx, `INSERT INTO trip_status_history
		(trip_id, from_status, to_status) VALUES ($1,$2,$3)`, trip.ID, fromStatus, trip.Status)
	if err != nil {
		return fmt.Errorf("insert trip status history: %w", err)
	}
	return nil
}
