package get_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/model"
	"github.com/olgapark/avito-labs/internal/usecase/trip/get"
)

func TestUsecase_Execute(t *testing.T) {
	tripID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	found := model.Trip{
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
		StartedAt: time.Date(2026, time.November, 19, 9, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name    string
		in      uuid.UUID
		setup   func(repo *MocktripRepository, ctx context.Context, id uuid.UUID)
		wantRes model.Trip
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "поездка найдена",
			in:   tripID,
			setup: func(repo *MocktripRepository, ctx context.Context, id uuid.UUID) {
				repo.EXPECT().Fetch(ctx, id).Return(found, nil)
			},
			wantRes: found,
			wantErr: assert.NoError,
		},
		{
			name: "поездка не найдена",
			in:   tripID,
			setup: func(repo *MocktripRepository, ctx context.Context, id uuid.UUID) {
				repo.EXPECT().Fetch(ctx, id).Return(model.Trip{}, trip.ErrTripNotFound)
			},
			wantRes: model.Trip{},
			wantErr: func(t assert.TestingT, err error, args ...any) bool {
				return assert.ErrorIs(t, err, trip.ErrTripNotFound, args...)
			},
		},
		{
			name: "репозиторий вернул техническую ошибку",
			in:   tripID,
			setup: func(repo *MocktripRepository, ctx context.Context, id uuid.UUID) {
				repo.EXPECT().Fetch(ctx, id).Return(model.Trip{}, assert.AnError)
			},
			wantRes: model.Trip{},
			wantErr: func(t assert.TestingT, err error, args ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError, args...)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tt.setup(repo, ctx, tt.in)

			got, err := get.New(repo).Execute(ctx, tt.in)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantRes, got)
		})
	}
}
