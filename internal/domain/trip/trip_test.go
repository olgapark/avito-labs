package trip_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/olgapark/avito-labs/internal/domain/trip"
)

func TestTrip_Finish(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	data := trip.Data{
		UserID:         uuid.MustParse("00000000-0000-4000-8000-000000000002"),
		DriverID:       uuid.MustParse("00000000-0000-4000-8000-000000000003"),
		StartLatitude:  59.93,
		StartLongitude: 30.31,
		EndLatitude:    60.01,
		EndLongitude:   29.76,
	}
	startedAt := time.Date(2026, time.November, 19, 8, 0, 0, 0, time.UTC)
	at := time.Date(2026, time.November, 19, 12, 0, 0, 123456789, time.FixedZone("MSK", 3*60*60))
	finishedAt := time.Date(2026, time.November, 19, 9, 0, 0, 123456000, time.UTC)
	prevFinishedAt := time.Date(2026, time.November, 19, 8, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		in      trip.Trip
		wantRes trip.Trip
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "завершить активную поездку",
			in:   trip.Trip{Data: data, ID: id, Status: trip.StatusActive, StartedAt: startedAt},
			wantRes: trip.Trip{
				Data: data, ID: id, Status: trip.StatusCompleted,
				StartedAt: startedAt, FinishedAt: &finishedAt,
			},
			wantErr: assert.NoError,
		},
		{
			name: "повторное завершение не меняет время",
			in: trip.Trip{
				Data: data, ID: id, Status: trip.StatusCompleted,
				StartedAt: startedAt, FinishedAt: new(prevFinishedAt),
			},
			wantRes: trip.Trip{
				Data: data, ID: id, Status: trip.StatusCompleted,
				StartedAt: startedAt, FinishedAt: &prevFinishedAt,
			},
			wantErr: func(t assert.TestingT, err error, args ...any) bool {
				return assert.ErrorIs(t, err, trip.ErrTripCompleted, args...)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			if got.FinishedAt != nil {
				got.FinishedAt = new(*got.FinishedAt)
			}

			err := got.Finish(at)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantRes, got)
		})
	}
}
