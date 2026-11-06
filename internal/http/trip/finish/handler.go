package finish

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/http/domainerror"
	"github.com/olgapark/avito-labs/internal/http/response"
)

type Handler struct{ usecase usecase }

func New(usecase usecase) *Handler { return &Handler{usecase: usecase} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		domainerror.Write(w, r, &trip.ValidationError{Field: "tripId", Message: "must be a UUID"})
		return
	}
	result, err := h.usecase.Execute(r.Context(), id)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, response.Trip(result))
}
