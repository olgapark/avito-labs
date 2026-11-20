package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/model"
	"github.com/olgapark/avito-labs/internal/usecase/trip/finish"
)

func TestUsecase_Execute(t *testing.T) {
	tripID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	active := model.Trip{
		ID: tripID,
		Data: trip.Data{
			UserID:         uuid.MustParse("00000000-0000-4000-8000-000000000002"),
			DriverID:       uuid.MustParse("00000000-0000-4000-8000-000000000003"),
			StartLatitude:  59.93,
			StartLongitude: 30.31,
			EndLatitude:    60.01,
			EndLongitude:   29.76,
			Price:          1250,
		},
		Status:    trip.StatusActive,
		StartedAt: time.Date(2026, time.November, 19, 8, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, time.November, 19, 9, 0, 0, 0, time.UTC)
	completed := model.Trip{
		ID:         tripID,
		Data:       active.Data,
		Status:     trip.StatusCompleted,
		StartedAt:  active.StartedAt,
		FinishedAt: new(now),
	}
	updateErr := errors.New("update failed")
	historyErr := errors.New("insert history failed")

	expectWrites := func(repo *MocktripRepository, txCtx context.Context, id uuid.UUID, err error) {
		fromStatus := trip.StatusActive
		gomock.InOrder(
			repo.EXPECT().FetchForUpdate(txCtx, id).Return(active, nil),
			repo.EXPECT().Update(txCtx, completed).Return(nil),
			repo.EXPECT().AddHistory(txCtx, completed, &fromStatus).Return(err),
		)
	}

	tests := []struct {
		name      string
		in        uuid.UUID
		setup     func(repo *MocktripRepository, txCtx context.Context, id uuid.UUID)
		wantRes   model.Trip
		wantEvent bool
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "активная поездка завершается и публикацией события",
			in:   tripID,
			setup: func(repo *MocktripRepository, txCtx context.Context, id uuid.UUID) {
				expectWrites(repo, txCtx, id, nil)
			},
			wantRes:   completed,
			wantEvent: true,
			wantErr:   assert.NoError,
		},
		{
			name: "отсутствующая поездка не обновляется",
			in:   tripID,
			setup: func(repo *MocktripRepository, txCtx context.Context, id uuid.UUID) {
				repo.EXPECT().FetchForUpdate(txCtx, id).Return(model.Trip{}, trip.ErrTripNotFound)
			},
			wantErr: errorIs(trip.ErrTripNotFound),
		},
		{
			name: "завершённая поездка не обновляется повторно",
			in:   tripID,
			setup: func(repo *MocktripRepository, txCtx context.Context, id uuid.UUID) {
				previous := completed
				previous.FinishedAt = new(time.Date(2026, time.November, 19, 8, 30, 0, 0, time.UTC))
				repo.EXPECT().FetchForUpdate(txCtx, id).Return(previous, nil)
			},
			wantErr: errorIs(trip.ErrTripCompleted),
		},
		{
			name: "сбой обновления останавливает завершение до записи истории",
			in:   tripID,
			setup: func(repo *MocktripRepository, txCtx context.Context, id uuid.UUID) {
				gomock.InOrder(
					repo.EXPECT().FetchForUpdate(txCtx, id).Return(active, nil),
					repo.EXPECT().Update(txCtx, completed).Return(updateErr),
				)
			},
			wantErr: errorIs(updateErr),
		},
		{
			name: "сбой записи истории прерывает транзакцию",
			in:   tripID,
			setup: func(repo *MocktripRepository, txCtx context.Context, id uuid.UUID) {
				expectWrites(repo, txCtx, id, historyErr)
			},
			wantErr: errorIs(historyErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			txCtx, cancel := context.WithCancel(ctx)
			t.Cleanup(cancel)
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			logger := NewMocklogger(ctrl)
			metrics := NewMockmetrics(ctrl)
			tt.setup(repo, txCtx, tt.in)
			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
				return fn(txCtx)
			})
			if tt.wantEvent {
				metrics.EXPECT().Completed(ctx)
				logger.EXPECT().InfoContext(ctx, gomock.Any(), "trip_id", tripID)
			}
			usecase := finish.New(repo, tx, logger, metrics, func() time.Time { return now })

			got, err := usecase.Execute(ctx, tt.in)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantRes, got)
		})
	}
}

func errorIs(want error) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, args ...any) bool {
		return assert.ErrorIs(t, err, want, args...)
	}
}
