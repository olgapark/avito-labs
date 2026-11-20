package response

import (
	"encoding/json"
	"net/http"

	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/model"
)

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func Trip(trip model.Trip) api.Trip {
	return api.Trip{
		Id: trip.ID, UserId: trip.UserID, DriverId: trip.DriverID,
		StartPoint: api.Coordinates{Latitude: trip.StartLatitude, Longitude: trip.StartLongitude},
		EndPoint:   api.Coordinates{Latitude: trip.EndLatitude, Longitude: trip.EndLongitude},
		Price:      trip.Price, Status: api.TripStatus(trip.Status), StartedAt: trip.StartedAt, FinishedAt: trip.FinishedAt,
	}
}
