package domainerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{
		{&trip.ValidationError{Field: "price", Message: "must be non-negative"}, http.StatusBadRequest, "invalid_request"},
		{trip.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{trip.ErrDriverBusy, http.StatusConflict, "driver_busy"},
		{trip.ErrTripCompleted, http.StatusConflict, "trip_completed"},
		{errors.New("password=secret SQL INSERT"), http.StatusInternalServerError, "internal_error"},
	} {
		t.Run(test.code, func(t *testing.T) {
			status, body := domainerror.Problem(fmt.Errorf("wrapped: %w", test.err))
			if status != test.status || int(body.Status) != test.status || body.Code != test.code || body.Title == "" || body.Type == "" {
				t.Fatalf("unexpected problem: %d, %+v", status, body)
			}
			if status == http.StatusInternalServerError && body.Detail != nil {
				t.Fatalf("internal error leaked: %+v", body)
			}
			if status == http.StatusBadRequest && (body.Detail == nil || *body.Detail != test.err.Error()) {
				t.Fatalf("validation detail lost: %+v", body)
			}
		})
	}
}
