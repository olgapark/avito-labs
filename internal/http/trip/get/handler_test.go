package get_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/http/trip/get"
	"github.com/olgapark/avito-labs/internal/model"
)

func TestHandler(t *testing.T) {
	for _, test := range []struct {
		name   string
		id     string
		err    error
		status int
		code   string
	}{
		{"success", uuid.New().String(), nil, http.StatusOK, ""},
		{"missing", uuid.New().String(), trip.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{"internal", uuid.New().String(), errors.New("SQL password=secret"), http.StatusInternalServerError, "internal_error"},
		{"invalid uuid", "bad", nil, http.StatusBadRequest, "invalid_request"},
	} {
		t.Run(test.name, func(t *testing.T) {
			usecase := NewMockusecase(gomock.NewController(t))
			id, err := uuid.Parse(test.id)
			if err == nil {
				usecase.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{ID: id}, test.err)
			}
			router := chi.NewRouter()
			router.Get("/api/v1/trips/{tripId}", get.New(usecase).ServeHTTP)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+test.id+"", nil))
			if response.Code != test.status {
				t.Fatalf("got %d: %s", response.Code, response.Body.String())
			}
			if test.status == http.StatusOK {
				var body api.Trip
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Id != id {
					t.Fatalf("wrong trip: %+v", body)
				}
			} else {
				var body api.Problem
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != test.code || response.Header().Get("Content-Type") != "application/problem+json" || strings.Contains(response.Body.String(), "secret") {
					t.Fatalf("wrong problem: %+v", body)
				}
			}
		})
	}
}
