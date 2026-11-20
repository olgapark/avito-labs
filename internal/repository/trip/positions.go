package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/olgapark/avito-labs/internal/model"
)

func (r *Repository) InsertPosition(ctx context.Context, position model.Position) (model.Position, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	err := r.executor(ctx).QueryRow(ctx, `INSERT INTO trip_positions (trip_id, latitude, longitude, recorded_at)
		VALUES ($1, $2, $3, $4) RETURNING id, recorded_at`,
		position.TripID, position.Latitude, position.Longitude, position.RecordedAt).
		Scan(&position.ID, &position.RecordedAt)
	if err != nil {
		return model.Position{}, fmt.Errorf("insert trip position: %w", err)
	}
	position.RecordedAt = position.RecordedAt.UTC()
	return position, nil
}

func (r *Repository) UpdateLastPosition(ctx context.Context, id uuid.UUID, recordedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	_, err := r.executor(ctx).Exec(ctx, `UPDATE trips SET last_position_at = GREATEST(last_position_at, $2)
		WHERE id = $1`, id, recordedAt)
	if err != nil {
		return fmt.Errorf("update last position time: %w", err)
	}
	return nil
}

func (r *Repository) ListPositions(ctx context.Context, id uuid.UUID) ([]model.Position, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	rows, err := r.executor(ctx).Query(ctx, `SELECT id, latitude, longitude, recorded_at FROM trip_positions
		WHERE trip_id = $1 ORDER BY recorded_at, id`, id)
	if err != nil {
		return nil, fmt.Errorf("query trip positions: %w", err)
	}
	positions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Position, error) {
		position := model.Position{TripID: id}
		err := row.Scan(&position.ID, &position.Latitude, &position.Longitude, &position.RecordedAt)
		position.RecordedAt = position.RecordedAt.UTC()
		return position, err
	})
	if err != nil {
		return nil, fmt.Errorf("collect trip positions: %w", err)
	}
	return positions, nil
}

func (r *Repository) ListStaleTrips(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	rows, err := r.executor(ctx).Query(ctx, `SELECT id, driver_id FROM trips
		WHERE status = 'active' AND COALESCE(last_position_at, started_at) < now() - $1::interval
		ORDER BY COALESCE(last_position_at, started_at), id`, pgtype.Interval{Microseconds: staleAfter.Microseconds(), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("query stale trips: %w", err)
	}
	trips, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.StaleTrip, error) {
		var trip model.StaleTrip
		err := row.Scan(&trip.ID, &trip.DriverID)
		return trip, err
	})
	if err != nil {
		return nil, fmt.Errorf("collect stale trips: %w", err)
	}
	return trips, nil
}
