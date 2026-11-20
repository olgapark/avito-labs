//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/infra/postgres/transaction"
	"github.com/olgapark/avito-labs/internal/model"
	triprepo "github.com/olgapark/avito-labs/internal/repository/trip"
)

const queryTimeout = 5 * time.Second

func TestRepository_Create(t *testing.T) {
	pool := testPool(t)
	repo := triprepo.New(pool, queryTimeout)
	tx := transaction.New(pool, queryTimeout)
	ctx := t.Context()
	active := newActiveTrip(t, pool)

	require.NoError(t, tx.Do(ctx, func(txCtx context.Context) error {
		return saveWithHistory(txCtx, repo, active)
	}))

	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT to_status FROM trip_status_history
		WHERE trip_id = $1 AND from_status IS NULL`, active.ID).Scan(&status))
	require.Equal(t, trip.StatusActive, status)
}

func TestRepository_Update(t *testing.T) {
	pool := testPool(t)
	repo := triprepo.New(pool, queryTimeout)
	ctx := t.Context()
	active := newActiveTrip(t, pool)
	require.NoError(t, repo.Save(ctx, active))

	finishedAt := time.Date(2026, time.November, 19, 9, 20, 0, 0, time.UTC)
	completed := active
	completed.Status = trip.StatusCompleted
	completed.FinishedAt = &finishedAt

	require.NoError(t, repo.Update(ctx, completed))

	var status string
	var storedFinishedAt time.Time
	require.NoError(t, pool.QueryRow(ctx, "SELECT status, finished_at FROM trips WHERE id = $1", active.ID).
		Scan(&status, &storedFinishedAt))
	require.Equal(t, trip.StatusCompleted, status)
	require.True(t, finishedAt.Equal(storedFinishedAt))
}

func TestRepository_Fetch(t *testing.T) {
	pool := testPool(t)
	repo := triprepo.New(pool, queryTimeout)
	ctx := t.Context()
	active := newActiveTrip(t, pool)
	require.NoError(t, repo.Save(ctx, active))

	got, err := repo.Fetch(ctx, active.ID)

	require.NoError(t, err)
	require.Equal(t, active, got)
}

func TestRepository_Save(t *testing.T) {
	pool := testPool(t)
	repo := triprepo.New(pool, queryTimeout)
	ctx := t.Context()
	active := newActiveTrip(t, pool)

	require.NoError(t, repo.Save(ctx, active))

	var driverID uuid.UUID
	var price int64
	require.NoError(t, pool.QueryRow(ctx, "SELECT driver_id, price FROM trips WHERE id = $1", active.ID).
		Scan(&driverID, &price))
	require.Equal(t, active.DriverID, driverID)
	require.Equal(t, active.Price, price)
}

func TestRepository_Rollback(t *testing.T) {
	pool := testPool(t)
	repo := triprepo.New(pool, queryTimeout)
	tx := transaction.New(pool, queryTimeout)
	ctx := t.Context()
	active := newActiveTrip(t, pool)
	abort := errors.New("abort after both inserts")

	err := tx.Do(ctx, func(txCtx context.Context) error {
		if err := saveWithHistory(txCtx, repo, active); err != nil {
			return err
		}
		return abort
	})

	assert.ErrorIs(t, err, abort)
	var counts [2]int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM trips WHERE id = $1),
		(SELECT count(*) FROM trip_status_history WHERE trip_id = $1)`, active.ID).
		Scan(&counts[0], &counts[1]))
	assert.Equal(t, [2]int64{0, 0}, counts)
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, dsn, "укажите DATABASE_URL тестовой PostgreSQL с миграциями проекта")
	pool, err := pgxpool.New(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func newActiveTrip(t *testing.T, pool *pgxpool.Pool) model.Trip {
	t.Helper()
	id := uuid.New()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		_, err := pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", id)
		assert.NoError(t, err)
	})
	return model.Trip{
		ID: id,
		Data: trip.Data{
			UserID:         uuid.New(),
			DriverID:       uuid.New(),
			StartLatitude:  59.93,
			StartLongitude: 30.31,
			EndLatitude:    60.01,
			EndLongitude:   29.76,
			Price:          1250,
		},
		Status:    trip.StatusActive,
		StartedAt: time.Date(2026, time.November, 19, 9, 0, 0, 123456000, time.UTC),
	}
}

func saveWithHistory(ctx context.Context, repo *triprepo.Repository, active model.Trip) error {
	if err := repo.Save(ctx, active); err != nil {
		return err
	}
	return repo.AddHistory(ctx, active, nil)
}
