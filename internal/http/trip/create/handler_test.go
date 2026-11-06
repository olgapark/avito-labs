package create_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/http/trip/create"
	"github.com/olgapark/avito-labs/internal/model"
)

const validBody = `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":0,"longitude":0},"end_point":{"latitude":90,"longitude":180},"price":0}`

func TestHandler(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"success", nil, http.StatusCreated, ""},
		{"driver busy", trip.ErrDriverBusy, http.StatusConflict, "driver_busy"},
		{"internal error", errors.New("SQL INSERT password=secret"), http.StatusInternalServerError, "internal_error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			usecase := NewMockusecase(gomock.NewController(t))
			want := model.Trip{ID: uuid.New(), Status: trip.StatusActive}
			usecase.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ interface{}, input model.CreateTrip) (model.Trip, error) {
				if input.UserID == uuid.Nil || input.DriverID == uuid.Nil || input.EndLatitude != 90 || input.EndLongitude != 180 || input.Price != 0 {
					t.Fatalf("wrong input: %+v", input)
				}
				return want, test.err
			})
			response := httptest.NewRecorder()
			create.New(usecase).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validBody)))
			if response.Code != test.status {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			if test.err == nil {
				var body api.Trip
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Id != want.ID || response.Header().Get("Location") != "/api/v1/trips/"+want.ID.String() {
					t.Fatalf("wrong response: %+v", body)
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

func TestInvalidBody(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"malformed", "{"},
		{"null", "null"},
		{"array", "[]"},
		{"missing fields", "{}"},
		{"trailing object", validBody + "{}"},
		{"unknown field", strings.Replace(validBody, `"price":0`, `"price":0,"extra":1`, 1)},
		{"null price", strings.Replace(validBody, `"price":0`, `"price":null`, 1)},
		{"negative price", strings.Replace(validBody, `"price":0`, `"price":-1`, 1)},
		{"missing coordinate", strings.Replace(validBody, `"latitude":0,`, "", 1)},
		{"null coordinate", strings.Replace(validBody, `"latitude":0`, `"latitude":null`, 1)},
		{"range", strings.Replace(validBody, `"latitude":0`, `"latitude":91`, 1)},
		{"bad uuid", strings.Replace(validBody, "5cb72c04-7650-45c9-a79b-bcdba0631e0c", "bad", 1)},
		{"zero uuid", strings.Replace(validBody, "5cb72c04-7650-45c9-a79b-bcdba0631e0c", uuid.Nil.String(), 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			usecase := NewMockusecase(gomock.NewController(t))
			response := httptest.NewRecorder()
			create.New(usecase).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(test.body)))
			var problem api.Problem
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusBadRequest || problem.Code != "invalid_request" {
				t.Fatalf("got %d: %+v", response.Code, problem)
			}
		})
	}
}
