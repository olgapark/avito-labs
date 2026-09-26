package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/olgapark/avito-labs/internal/domain"
	api "github.com/olgapark/avito-labs/internal/generated"
)

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	params api.CreateTripParams,
) {
	var req api.CreateTripJSONRequestBody

	if err := decodeTrip(r.Body, &req); err != nil {
		HandleAPIError(w, r, err)
		return
	}

	if req.Price < 0 {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)

		_ = json.NewEncoder(w).Encode(api.Problem{
			Instance: &r.URL.Path,
			Type:     "https://tripgo.example/problems/invalid-request",
			Title:    "Invalid request",
			Status:   http.StatusBadRequest,
			Code:     "invalid_request",
		})
		return
	}

	if req.StartPoint.Latitude < -90 || req.StartPoint.Latitude > 90 ||
		req.EndPoint.Latitude < -90 || req.EndPoint.Latitude > 90 ||
		req.StartPoint.Longitude < -180 || req.StartPoint.Longitude > 180 ||
		req.EndPoint.Longitude < -180 || req.EndPoint.Longitude > 180 {

		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)

		_ = json.NewEncoder(w).Encode(api.Problem{
			Instance: &r.URL.Path,
			Type:     "https://tripgo.example/problems/invalid-request",
			Title:    "Invalid request",
			Status:   http.StatusBadRequest,
			Code:     "invalid_request",
		})
		return
	}

	if req.UserId == (uuid.UUID{}) || req.DriverId == (uuid.UUID{}) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)

		_ = json.NewEncoder(w).Encode(api.Problem{
			Instance: &r.URL.Path,
			Type:     "https://tripgo.example/problems/invalid-request",
			Title:    "Invalid request",
			Status:   http.StatusBadRequest,
			Code:     "invalid_request",
		})
		return
	}

	trip := &domain.Trip{
		UserID:         req.UserId,
		DriverID:       req.DriverId,
		StartLatitude:  req.StartPoint.Latitude,
		StartLongitude: req.StartPoint.Longitude,
		EndLatitude:    req.EndPoint.Latitude,
		EndLongitude:   req.EndPoint.Longitude,
		Price:          req.Price,
	}

	err := h.service.CreateTrip(r.Context(), trip)
	if err != nil {
		if errors.Is(err, domain.ErrDriverBusy) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusConflict)

			_ = json.NewEncoder(w).Encode(api.Problem{
				Instance: &r.URL.Path,
				Type:     "https://tripgo.example/problems/driver-busy",
				Title:    "Driver busy",
				Status:   http.StatusConflict,
				Code:     "driver_busy",
			})
			return
		}

		w.Header().Set("Content-Type", "application/problem+json")
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)

		_ = json.NewEncoder(w).Encode(api.Problem{
			Instance: &r.URL.Path,
			Type:     "https://tripgo.example/problems/internal-error",
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Code:     "internal_error",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(api.Trip{
		Id:       trip.ID,
		UserId:   trip.UserID,
		DriverId: trip.DriverID,

		StartPoint: api.Coordinates{
			Latitude:  trip.StartLatitude,
			Longitude: trip.StartLongitude,
		},

		EndPoint: api.Coordinates{
			Latitude:  trip.EndLatitude,
			Longitude: trip.EndLongitude,
		},

		Price:      trip.Price,
		Status:     api.TripStatus(trip.Status),
		StartedAt:  trip.StartedAt,
		FinishedAt: trip.FinishedAt,
	})
}

func decodeTrip(body io.Reader, req *api.CreateTripJSONRequestBody) error {
	decoder := json.NewDecoder(body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("expected exactly one JSON object")
	}
	if err := requiredFields(raw, "user_id", "driver_id", "start_point", "end_point", "price"); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, key := range []string{"start_point", "end_point"} {
		if err := requiredFields(fields[key], "latitude", "longitude"); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(req)
}
func requiredFields(raw json.RawMessage, names ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, name := range names {
		value, exists := fields[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s is required and must not be null", name)
		}
	}
	return nil
}
