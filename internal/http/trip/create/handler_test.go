package create_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/http/trip/create"
	"github.com/olgapark/avito-labs/internal/model"
)

func TestHandler_ServeHTTP(t *testing.T) {
	const (
		path      = "/api/v1/trips"
		tripID    = "00000000-0000-4000-8000-000000000001"
		validBody = `{
			"user_id": "00000000-0000-4000-8000-000000000002",
			"driver_id": "00000000-0000-4000-8000-000000000003",
			"start_point": {"latitude": 59.93, "longitude": 30.31},
			"end_point": {"latitude": 60.01, "longitude": 29.76},
			"price": 1250
		}`
	)
	input := model.CreateTrip{
		UserID:         uuid.MustParse("00000000-0000-4000-8000-000000000002"),
		DriverID:       uuid.MustParse("00000000-0000-4000-8000-000000000003"),
		StartLatitude:  59.93,
		StartLongitude: 30.31,
		EndLatitude:    60.01,
		EndLongitude:   29.76,
		Price:          1250,
	}
	created := model.Trip{
		ID:        uuid.MustParse(tripID),
		Data:      input,
		Status:    trip.StatusActive,
		StartedAt: time.Date(2026, time.November, 19, 9, 0, 0, 123456000, time.UTC),
	}
	usecaseErr := errors.New("internal failure")

	tests := []struct {
		name            string
		body            string
		setup           func(*Mockusecase)
		wantStatus      int
		wantProblemCode string
	}{
		{
			name: "создаёт поездку, возвращает 201",
			body: validBody,
			setup: func(usecase *Mockusecase) {
				usecase.EXPECT().Execute(gomock.Any(), input).Return(created, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:            "запрос без долготы, возвращает 400",
			body:            strings.Replace(validBody, `, "longitude": 29.76`, "", 1),
			wantStatus:      http.StatusBadRequest,
			wantProblemCode: "invalid_request",
		},
		{
			name: "внутренняя ошибка, возвращает 500",
			body: validBody,
			setup: func(usecase *Mockusecase) {
				usecase.EXPECT().Execute(gomock.Any(), input).Return(model.Trip{}, usecaseErr)
			},
			wantStatus:      http.StatusInternalServerError,
			wantProblemCode: "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			usecase := NewMockusecase(gomock.NewController(t))
			if tt.setup != nil {
				tt.setup(usecase)
			}

			create.New(usecase).ServeHTTP(rec, req)

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
			assert.Equal(t, created.ID, resp.Id)
			assert.Equal(t, path+"/"+tripID, rec.Header().Get("Location"))
		})
	}
}
