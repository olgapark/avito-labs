package domainerror_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		in         error
		wantStatus int
		wantBody   api.Problem
	}{
		{
			name:       "обёрнутая ошибка отсутствующей поездки",
			in:         fmt.Errorf("fetch trip: %w", trip.ErrTripNotFound),
			wantStatus: http.StatusNotFound,
			wantBody: api.Problem{
				Type:   "https://tripgo.example/problems/trip-not-found",
				Title:  "Trip not found",
				Status: 404,
				Code:   "trip_not_found",
			},
		},
		{
			name:       "занят водитель",
			in:         trip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantBody: api.Problem{
				Type:   "https://tripgo.example/problems/driver-busy",
				Title:  "Driver busy",
				Status: 409,
				Code:   "driver_busy",
			},
		},
		{
			name:       "повторное завершение",
			in:         trip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantBody: api.Problem{
				Type:   "https://tripgo.example/problems/trip-completed",
				Title:  "Trip completed",
				Status: 409,
				Code:   "trip_completed",
			},
		},
		{
			name:       "техническая ошибка",
			in:         fmt.Errorf("fetch trip: %w", assert.AnError),
			wantStatus: http.StatusInternalServerError,
			wantBody: api.Problem{
				Type:   "https://tripgo.example/problems/internal-error",
				Title:  "Internal Server Error",
				Status: 500,
				Code:   "internal_error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := domainerror.Problem(tt.in)

			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantBody.Type, body.Type)
			assert.Equal(t, tt.wantBody.Title, body.Title)
			assert.Equal(t, tt.wantBody.Status, body.Status)
			assert.Equal(t, tt.wantBody.Code, body.Code)
			if status == http.StatusInternalServerError {
				assert.Nil(t, body.Detail)
			}
		})
	}
}
