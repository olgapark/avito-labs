package positions

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/http/domainerror"
	"github.com/olgapark/avito-labs/internal/http/response"
	"github.com/olgapark/avito-labs/internal/model"
)

type Handler struct{ usecase usecase }

func New(usecase usecase) *Handler { return &Handler{usecase: usecase} }

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	id, err := tripID(r)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	input, err := decode(r.Body)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	position, err := h.usecase.Save(r.Context(), id, input)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toResponse(position))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	id, err := tripID(r)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	positions, err := h.usecase.List(r.Context(), id)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	result := api.TripPositionsResponse{Positions: make([]api.TripPosition, 0, len(positions))}
	for _, position := range positions {
		result.Positions = append(result.Positions, toResponse(position))
	}
	response.JSON(w, http.StatusOK, result)
}

func tripID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		return uuid.Nil, &trip.ValidationError{Field: "tripId", Message: "must be a UUID"}
	}
	return id, nil
}

func decode(body io.Reader) (model.PositionData, error) {
	var request struct {
		Coordinates *struct {
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"coordinates"`
		RecordedAt *time.Time `json:"recorded_at"`
	}
	invalid := &trip.ValidationError{Field: "body", Message: "must be one JSON object with coordinates and a valid recorded_at"}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return model.PositionData{}, invalid
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return model.PositionData{}, invalid
	}
	if request.Coordinates == nil || request.Coordinates.Latitude == nil || request.Coordinates.Longitude == nil || request.RecordedAt == nil {
		return model.PositionData{}, invalid
	}
	return model.PositionData{
		Latitude: *request.Coordinates.Latitude, Longitude: *request.Coordinates.Longitude, RecordedAt: *request.RecordedAt,
	}, nil
}

func toResponse(position model.Position) api.TripPosition {
	return api.TripPosition{
		Id: position.ID, TripId: position.TripID, RecordedAt: position.RecordedAt,
		Coordinates: api.Coordinates{Latitude: position.Latitude, Longitude: position.Longitude},
	}
}
