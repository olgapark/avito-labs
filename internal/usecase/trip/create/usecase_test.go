package create_test

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
	"github.com/olgapark/avito-labs/internal/usecase/trip/create"
)

func TestUsecase_Execute(t *testing.T) {
	tripID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	input := model.CreateTrip{
		UserID:         uuid.MustParse("00000000-0000-4000-8000-000000000002"),
		DriverID:       uuid.MustParse("00000000-0000-4000-8000-000000000003"),
		StartLatitude:  59.93,
		StartLongitude: 30.31,
		EndLatitude:    60.01,
		EndLongitude:   29.76,
		Price:          1250,
	}
	now := time.Date(2026, time.November, 19, 9, 0, 0, 0, time.UTC)
	created := model.Trip{
		ID:        tripID,
		Data:      input,
		Status:    trip.StatusActive,
		StartedAt: now,
	}
	commitErr := errors.New("commit failed")

	expectWrites := func(repo *MocktripRepository, txCtx context.Context) {
		gomock.InOrder(
			repo.EXPECT().Save(txCtx, created).Return(nil),
			repo.EXPECT().AddHistory(txCtx, created, nil).Return(nil),
		)
	}

	tests := []struct {
		name      string
		setup     func(repo *MocktripRepository, txCtx context.Context)
		commitErr error
		wantRes   model.Trip
		wantEvent bool
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:      "поездка и история записываются, после транзакции публикуется событие",
			setup:     expectWrites,
			wantRes:   created,
			wantEvent: true,
			wantErr:   assert.NoError,
		},
		{
			name: "репозиторий отклоняет создание для занятого водителя",
			setup: func(repo *MocktripRepository, txCtx context.Context) {
				repo.EXPECT().Save(txCtx, created).Return(trip.ErrDriverBusy)
			},
			wantErr: errorIs(trip.ErrDriverBusy),
		},
		{
			name:      "при ошибке COMMIT возвращает ошибку и пустой результат",
			setup:     expectWrites,
			commitErr: commitErr,
			wantErr:   errorIs(commitErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			logger := NewMocklogger(ctrl)
			metrics := NewMockmetrics(ctrl)
			txCtx, cancel := context.WithCancel(ctx)
			t.Cleanup(cancel)
			tt.setup(repo, txCtx)
			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
				if err := fn(txCtx); err != nil {
					return err
				}
				return tt.commitErr
			})
			if tt.wantEvent {
				metrics.EXPECT().Created(ctx)
				logger.EXPECT().InfoContext(ctx, gomock.Any(), "trip_id", tripID)
			}
			usecase := create.New(repo, tx, logger, metrics, func() time.Time { return now }, func() uuid.UUID { return tripID })

			got, err := usecase.Execute(ctx, input)

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
