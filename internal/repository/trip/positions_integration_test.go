//go:build integration

package trip_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/olgapark/avito-labs/internal/infra/postgres/transaction"
	"github.com/olgapark/avito-labs/internal/model"
	triprepo "github.com/olgapark/avito-labs/internal/repository/trip"
	"github.com/olgapark/avito-labs/internal/usecase/trip/positions"
)

func TestRepository_ListStaleTrips(t *testing.T) {
	const staleAfter = time.Minute
	pool := testPool(t)
	repo := triprepo.New(pool, queryTimeout)
	tests := []struct {
		name        string
		positionAge time.Duration
		completed   bool
		wantListed  bool
	}{
		{name: "поездка со свежей координатой пропускается", positionAge: time.Second},
		{name: "завершённая поездка пропускается", completed: true},
		{name: "активная поездка без координат попадает в выборку", wantListed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			candidate := newActiveTrip(t, pool)
			var now time.Time
			require.NoError(t, pool.QueryRow(ctx, "SELECT now()").Scan(&now))
			candidate.StartedAt = now.Add(-2 * staleAfter)
			require.NoError(t, repo.Save(ctx, candidate))
			if tt.positionAge > 0 {
				require.NoError(t, repo.UpdateLastPosition(ctx, candidate.ID, now.Add(-tt.positionAge)))
			}
			if tt.completed {
				require.NoError(t, candidate.Finish(now))
				require.NoError(t, repo.Update(ctx, candidate))
			}

			got, err := repo.ListStaleTrips(ctx, staleAfter)

			require.NoError(t, err)
			listed := slices.ContainsFunc(got, func(result model.StaleTrip) bool {
				return result.ID == candidate.ID
			})
			assert.Equal(t, tt.wantListed, listed)
		})
	}
}

func TestRepository_SavePositionAtomically(t *testing.T) {
	pool := testPool(t)
	repo := triprepo.New(pool, queryTimeout)
	tx := transaction.New(pool, queryTimeout)
	recordedAt := time.Date(2026, time.November, 19, 9, 10, 0, 0, time.UTC)
	input := model.PositionData{Latitude: 59.93, Longitude: 30.33, RecordedAt: recordedAt}
	now := func() time.Time { return recordedAt.Add(time.Minute) }
	updateErr := errors.New("abort after updating last position")

	tests := []struct {
		name               string
		updateErr          error
		wantLastPositionAt *time.Time
		wantPositions      int
	}{
		{
			name:               "координата сохраняется и обновляет время последней точки",
			wantLastPositionAt: &recordedAt,
			wantPositions:      1,
		},
		{
			name:      "при ошибке откатываются координата и время последней точки",
			updateErr: updateErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			active := newActiveTrip(t, pool)
			require.NoError(t, repo.Save(ctx, active))
			positionRepo := positionUpdateRepository{Repository: repo, err: tt.updateErr}
			usecase := positions.New(positionRepo, tx, positionObserver{}, positionObserver{}, now)

			_, err := usecase.Save(ctx, active.ID, input)

			require.ErrorIs(t, err, tt.updateErr)
			stored, err := repo.Fetch(ctx, active.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantLastPositionAt, stored.LastPositionAt)
			points, err := repo.ListPositions(ctx, active.ID)
			require.NoError(t, err)
			assert.Len(t, points, tt.wantPositions)
		})
	}
}

type positionUpdateRepository struct {
	*triprepo.Repository
	err error
}

func (r positionUpdateRepository) UpdateLastPosition(ctx context.Context, id uuid.UUID, recordedAt time.Time) error {
	if err := r.Repository.UpdateLastPosition(ctx, id, recordedAt); err != nil {
		return err
	}
	return r.err
}

type positionObserver struct{}

func (positionObserver) Saved(context.Context)             {}
func (positionObserver) Cancel(context.Context, uuid.UUID) {}
