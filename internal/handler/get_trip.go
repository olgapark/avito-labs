package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/olgapark/avito-labs/internal/domain"

	api "github.com/olgapark/avito-labs/internal/generated"
)

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	trip, err := h.service.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, domain.ErrTripNotFound) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusNotFound)

			_ = json.NewEncoder(w).Encode(api.Problem{
				Instance: &r.URL.Path,
				Type:     "https://tripgo.example/problems/trip-not-found",
				Title:    "Trip not found",
				Status:   http.StatusNotFound,
				Code:     "trip_not_found",
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

	response := api.Trip{
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
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(response)
}
