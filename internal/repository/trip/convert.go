package trip

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	domain "github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/model"
)

func scanTrip(row pgx.Row) (model.Trip, error) {
	var result model.Trip
	err := row.Scan(&result.ID, &result.UserID, &result.DriverID,
		&result.StartLatitude, &result.StartLongitude, &result.EndLatitude, &result.EndLongitude,
		&result.Price, &result.Status, &result.StartedAt, &result.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, domain.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("scan trip: %w", err)
	}
	result.StartedAt = result.StartedAt.UTC()
	if result.FinishedAt != nil {
		finishedAt := result.FinishedAt.UTC()
		result.FinishedAt = &finishedAt
	}
	return result, nil
}

func saveError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
		return fmt.Errorf("save trip: %w", domain.ErrDriverBusy)
	}
	return fmt.Errorf("save trip: %w", err)
}
