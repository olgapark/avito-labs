package domainerror

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	api "github.com/olgapark/avito-labs/internal/generated"
)

type stateKey struct{}

type State struct {
	Err     error
	Context context.Context
}

func WithState(ctx context.Context) (context.Context, *State) {
	state := &State{Context: ctx}
	return context.WithValue(ctx, stateKey{}, state), state
}

func RequestState(ctx context.Context) *State {
	state, _ := ctx.Value(stateKey{}).(*State)
	return state
}

func Problem(err error) (int, api.Problem) {
	status, code, title := http.StatusInternalServerError, "internal_error", "Internal Server Error"
	var validation *trip.ValidationError
	switch {
	case errors.As(err, &validation):
		status, code, title = http.StatusBadRequest, "invalid_request", "Invalid request"
	case errors.Is(err, trip.ErrTripNotFound):
		status, code, title = http.StatusNotFound, "trip_not_found", "Trip not found"
	case errors.Is(err, trip.ErrDriverBusy):
		status, code, title = http.StatusConflict, "driver_busy", "Driver busy"
	case errors.Is(err, trip.ErrTripCompleted):
		status, code, title = http.StatusConflict, "trip_completed", "Trip completed"
	}
	body := api.Problem{Type: "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"), Title: title, Status: int32(status), Code: code}
	if validation != nil {
		detail := validation.Error()
		body.Detail = &detail
	}
	return status, body
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	status, body := Problem(err)
	body.Instance = &r.URL.Path
	if state := RequestState(r.Context()); state != nil {
		state.Err = err
	}
	span := trace.SpanFromContext(r.Context())
	span.SetStatus(codes.Error, body.Code)
	span.SetAttributes(attribute.String("error.type", body.Code))
	span.RecordError(err)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
