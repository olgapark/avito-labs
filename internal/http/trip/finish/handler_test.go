package finish_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/http/trip/finish"
)

func TestHandler_ServeHTTP(t *testing.T) {
	const route = "/api/v1/trips/{tripId}/finish"
	tripID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	sampleTrip := trip.Trip{
		Data: trip.Data{
			UserID:   uuid.MustParse("00000000-0000-4000-8000-000000000002"),
			DriverID: uuid.MustParse("00000000-0000-4000-8000-000000000003"),
		},
		ID:         tripID,
		Status:     trip.StatusCompleted,
		StartedAt:  time.Date(2026, time.November, 19, 9, 0, 0, 0, time.UTC),
		FinishedAt: new(time.Date(2026, time.November, 19, 9, 20, 0, 0, time.UTC)),
	}
	usecaseErr := errors.New("internal failure")

	tests := []struct {
		name            string
		tripID          string
		setup           func(*Mockusecase)
		wantStatus      int
		wantProblemCode string
	}{
		{
			name:   "возвращает завершённую поездку с кодом 200",
			tripID: tripID.String(),
			setup: func(usecase *Mockusecase) {
				usecase.EXPECT().Execute(gomock.Any(), tripID).Return(sampleTrip, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:            "некорректный идентификатор поездки, возвращает 400",
			tripID:          "invalid-id",
			wantStatus:      http.StatusBadRequest,
			wantProblemCode: "invalid_request",
		},
		{
			name:   "внутренняя ошибка, возвращает 500",
			tripID: tripID.String(),
			setup: func(usecase *Mockusecase) {
				usecase.EXPECT().Execute(gomock.Any(), tripID).Return(trip.Trip{}, usecaseErr)
			},
			wantStatus:      http.StatusInternalServerError,
			wantProblemCode: "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tt.tripID+"/finish", nil)
			rec := httptest.NewRecorder()
			usecase := NewMockusecase(gomock.NewController(t))
			if tt.setup != nil {
				tt.setup(usecase)
			}
			router := chi.NewRouter()
			router.Post(route, finish.New(usecase).ServeHTTP)

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantProblemCode != "" {
				var problem api.Problem
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
				assert.Equal(t, tt.wantProblemCode, problem.Code)
				if tt.wantStatus == http.StatusInternalServerError {
					assert.NotContains(t, rec.Body.String(), usecaseErr.Error())
					assert.Nil(t, problem.Detail)
				}
				return
			}

			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			var resp api.Trip
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Equal(t, tripID, resp.Id)
			assert.Equal(t, api.TripStatus(trip.StatusCompleted), resp.Status)
			require.NotNil(t, resp.FinishedAt)
			assert.True(t, sampleTrip.FinishedAt.Equal(*resp.FinishedAt))
		})
	}
}
