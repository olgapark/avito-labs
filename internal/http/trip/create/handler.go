package create

import (
	"net/http"

	"github.com/olgapark/avito-labs/internal/http/domainerror"
	"github.com/olgapark/avito-labs/internal/http/response"
)

type Handler struct{ usecase usecase }

func New(usecase usecase) *Handler { return &Handler{usecase: usecase} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	input, err := decode(r.Body)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	trip, err := h.usecase.Execute(r.Context(), input)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	response.JSON(w, http.StatusCreated, response.Trip(trip))
}
