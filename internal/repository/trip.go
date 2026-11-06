package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/olgapark/avito-labs/internal/domain"
)

func (r *Repository) GetTrip(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Trip, error) {
	query, args, err := squirrel.
		Select(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
		).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build get trip query: %w", err)
	}

	var trip domain.Trip

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	err = r.db(ctx).QueryRow(queryCtx, query, args...).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartLatitude,
		&trip.StartLongitude,
		&trip.EndLatitude,
		&trip.EndLongitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTripNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	return &trip, nil
}

func (r *Repository) CreateTrip(
	ctx context.Context,
	trip *domain.Trip,
) error {
	query, args, err := squirrel.
		Insert("trips").
		Columns(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
		).
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartLatitude,
			trip.StartLongitude,
			trip.EndLatitude,
			trip.EndLongitude,
			trip.Price,
			trip.Status,
			trip.StartedAt,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("build create trip query: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	_, err = r.db(ctx).Exec(queryCtx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return domain.ErrDriverBusy
		}

		return fmt.Errorf("create trip: %w", err)
	}

	return nil
}

func (r *Repository) AddTripStatusHistory(
	ctx context.Context,
	tripID uuid.UUID,
	fromStatus *string,
	toStatus string,
	reason *string,
) error {
	query, args, err := squirrel.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"reason",
		).
		Values(
			tripID,
			fromStatus,
			toStatus,
			reason,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("build status history query: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	_, err = r.db(ctx).Exec(queryCtx, query, args...)
	if err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}

	return nil
}

func (r *Repository) FinishTrip(
	ctx context.Context,
	id uuid.UUID,
	finishedAt time.Time,
) error {
	query, args, err := squirrel.
		Update("trips").
		Set("status", "completed").
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(squirrel.Eq{
			"id":     id,
			"status": "active",
		}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("build finish trip query: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	tag, err := r.db(ctx).Exec(queryCtx, query, args...)
	if err != nil {
		return fmt.Errorf("finish trip: %w", err)
	}

	if tag.RowsAffected() == 0 {
		trip, err := r.GetTrip(ctx, id)
		if err != nil {
			return err
		}

		if trip.Status == "completed" {
			return domain.ErrTripCompleted
		}
	}

	return nil
}
